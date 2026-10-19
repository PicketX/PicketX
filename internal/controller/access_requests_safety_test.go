// SPDX-License-Identifier: Apache-2.0
package controller

import (
	"encoding/json"
	"github.com/PicketX/PicketX/internal/accessrequest"
	"os"
	"strings"
	"testing"
	"time"
)

func TestAccessRequestRejectsStaleExpiredOrWidenedIntent(t *testing.T) {
	for _, scenario := range []struct {
		name string
		want int
	}{
		{"retarget", 409}, {"remove", 409}, {"recreate", 409}, {"expired", 400}, {"broader entries", 400}, {"longer deadline", 400}, {"stale version", 409}, {"stale publication", 412},
	} {
		t.Run(scenario.name, func(t *testing.T) {
			h, s, creds := managementFixture(t)
			policy, _ := os.ReadFile("../../examples/static/gitea.yaml")
			facts, _ := os.ReadFile("../../examples/static/facts.json")
			extra := "\n---\napiVersion: picketx.io/v1alpha1\nkind: Resource\nmetadata: {name: request-target, uid: request-target-v1}\nspec:\n  endpoints:\n  - name: web\n    network: {protocol: TCP, port: 9443}\n  - name: ssh\n    network: {protocol: TCP, port: 9222}\n"
			policy = append(policy, []byte(extra)...)
			publish := func(p []byte, tag string) {
				body, _ := json.Marshal(map[string]string{"policy": string(p), "facts": string(facts), "reason": "safety inventory"})
				w := request(h, "PUT", "/api/v1/publication", string(body), managementToken, tag)
				if w.Code != 200 {
					t.Fatal(w.Code, w.Body.String())
				}
			}
			publish(policy, `"0"`)
			w := request(h, "POST", "/api/v1/access-requests", `{"source":"192.0.2.42","resource_name":"request-target","resource_uid":"request-target-v1","endpoint_names":["web"],"duration_seconds":3600,"reason":"immutable intent"}`, managementToken, `"1"`)
			var r accessrequest.Record
			if w.Code != 201 || json.Unmarshal(w.Body.Bytes(), &r) != nil {
				t.Fatal(w.Code, w.Body.String())
			}
			current, _ := os.ReadFile(creds)
			if err := os.WriteFile(creds, []byte(strings.Replace(string(current), "principal:operator", "principal:approver", 1)), 0600); err != nil {
				t.Fatal(err)
			}
			decision := accessrequest.Decision{Version: 1, Action: "approve", EndpointNames: []string{"web"}, ExpiresAt: r.ExpiresAt.Format(time.RFC3339Nano), Reason: "review"}
			tag := `"1"`
			switch scenario.name {
			case "retarget":
				publish([]byte(strings.Replace(string(policy), "port: 9443", "port: 9444", 1)), tag)
				tag = `"2"`
			case "remove":
				publish(policy[:len(policy)-len(extra)], tag)
				tag = `"2"`
			case "recreate":
				publish([]byte(strings.Replace(string(policy), "request-target-v1", "request-target-v2", 1)), tag)
				tag = `"2"`
			case "expired":
				r.ExpiresAt = time.Now().Add(-time.Minute).UTC()
				raw, _ := json.Marshal(r)
				if err := s.GORM().Exec("UPDATE controller_access_request SET data=? WHERE uid=?", raw, r.UID).Error; err != nil {
					t.Fatal(err)
				}
				decision.ExpiresAt = r.ExpiresAt.Format(time.RFC3339Nano)
			case "broader entries":
				decision.EndpointNames = []string{"web", "ssh"}
			case "longer deadline":
				decision.ExpiresAt = r.ExpiresAt.Add(time.Second).Format(time.RFC3339Nano)
			case "stale version":
				decision.Version = 2
			case "stale publication":
				publish([]byte(strings.Replace(string(policy), "port: 9222", "port: 9223", 1)), tag)
			}
			raw, _ := json.Marshal(decision)
			w = request(h, "POST", "/api/v1/access-requests/"+r.UID+"/decision", string(raw), managementToken, tag)
			if w.Code != scenario.want {
				t.Fatal(w.Code, w.Body.String())
			}
			got, _ := s.GetRequest(r.UID)
			doc, _ := s.Snapshot()
			if got.State != "pending" || strings.Contains(doc.Policy, r.UID) {
				t.Fatal("failed approval persisted", got, doc.Policy)
			}
			// Rejection is still possible without a live target or unexpired deadline.
			w = request(h, "POST", "/api/v1/access-requests/"+r.UID+"/decision", `{"version":1,"action":"reject","reason":"not accepted"}`, managementToken, "")
			if w.Code != 200 {
				t.Fatal(w.Code, w.Body.String())
			}
		})
	}
}
