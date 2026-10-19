// SPDX-License-Identifier: Apache-2.0
package controllerapp

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/PicketX/PicketX/internal/accessrequest"
	agentv1 "github.com/PicketX/PicketX/pkg/protocol/agent/v1alpha1"
	"os"
	"testing"
	"time"
)

func TestAccessRequestRuntimeApprovalGRPCAndRestart(t *testing.T) {
	c := managedConfig(t)
	requester := "runtime-requester-token-123456789012345678901"
	hash := func(s string) string { v := sha256.Sum256([]byte(s)); return hex.EncodeToString(v[:]) }
	write(t, c.Management.Credentials, []byte(fmt.Sprintf(`{"principals":[{"id":"principal:approver","sha256":%q,"roles":["role:admin"]},{"id":"principal:requester","sha256":%q,"roles":["role:requester"]}]}`, hash(adminToken), hash(requester))))
	p := start(t, c)
	ready(t, p, 200)
	policy, _ := os.ReadFile("../../examples/static/gitea.yaml")
	facts, _ := os.ReadFile("../../examples/static/facts.json")
	body, _ := json.Marshal(map[string]string{"policy": string(policy), "facts": string(facts), "reason": "workflow inventory"})
	if code, _, data := apiCall(t, p, "PUT", "/api/v1/publication", body, `"1"`, adminToken); code != 200 {
		t.Fatal(code, string(data))
	}
	st := stream(t, p)
	_ = snapshot(t, st)
	var r accessrequest.Record
	body = []byte(`{"source":"2001:db8::42","resource_name":"gitea","resource_uid":"resource-gitea-v1","endpoint_names":["https","ssh"],"duration_seconds":3600,"reason":"runtime request"}`)
	code, _, data := apiCall(t, p, "POST", "/api/v1/access-requests", body, `"2"`, requester)
	if code != 201 || json.Unmarshal(data, &r) != nil {
		t.Fatal(code, string(data))
	}
	requested := r.ExpiresAt
	deadline := requested.Add(-time.Minute).UTC()
	body, _ = json.Marshal(accessrequest.Decision{Version: r.Version, Action: "approve", EndpointNames: []string{"https"}, ExpiresAt: deadline.Format(time.RFC3339Nano), Reason: "runtime independent narrowing"})
	code, _, data = apiCall(t, p, "POST", "/api/v1/access-requests/"+r.UID+"/decision", body, `"2"`, adminToken)
	if code != 200 || json.Unmarshal(data, &r) != nil || r.State != "approved" {
		t.Fatal(code, string(data))
	}
	check := func(s *agentv1.Snapshot) {
		count := 0
		for _, target := range s.Targets {
			for _, g := range target.Grants {
				if g.Id == r.GrantUID {
					count++
					if target.Key.EndpointName != "https" || g.Source != "2001:db8::42/128" || g.ExpiresAt != deadline.Format(time.RFC3339Nano) {
						t.Fatal(target, g)
					}
				}
			}
		}
		if s.Revision != 3 || count != 1 {
			t.Fatal("approval delivery not exact", s, count)
		}
	}
	check(snapshot(t, st))
	p.stop(t)
	p = start(t, c)
	ready(t, p, 200)
	check(snapshot(t, stream(t, p)))
	code, _, data = apiCall(t, p, "GET", "/api/v1/access-requests", nil, "", requester)
	var page accessrequest.Page
	if code != 200 || json.Unmarshal(data, &page) != nil || len(page.Requests) != 1 || page.Requests[0].Requester != "principal:requester" || page.Requests[0].Approver != "principal:approver" || !page.Requests[0].ExpiresAt.Equal(requested) || page.Requests[0].GrantUID != r.GrantUID {
		t.Fatal(code, string(data))
	}
}
