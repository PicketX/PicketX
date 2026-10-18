// SPDX-License-Identifier: Apache-2.0
package controllerapp

import (
	"encoding/json"
	"os"
	"testing"
	"time"

	agentv1 "github.com/PicketX/PicketX/pkg/protocol/agent/v1alpha1"
)

func TestStructuredGrantRuntimeGRPCAndDurableIndependentRevocation(t *testing.T) {
	c := managedConfig(t)
	p := start(t, c)
	ready(t, p, 200)
	policy, err := os.ReadFile("../../examples/static/gitea.yaml")
	if err != nil {
		t.Fatal(err)
	}
	facts, err := os.ReadFile("../../examples/static/facts.json")
	if err != nil {
		t.Fatal(err)
	}
	raw, _ := json.Marshal(map[string]string{"policy": string(policy), "facts": string(facts), "reason": "inventory"})
	if code, _, body := apiCall(t, p, "PUT", "/api/v1/publication", raw, `"1"`, adminToken); code != 200 {
		t.Fatal(code, string(body))
	}
	st := stream(t, p)
	_ = snapshot(t, st)
	var first, second struct {
		UID       string `json:"uid"`
		ExpiresAt string `json:"expires_at"`
	}
	var before *agentv1.Snapshot
	for i, dst := range []any{&first, &second} {
		name := "one"
		match := `"2"`
		if i == 1 {
			name = "two"
			match = `"3"`
		}
		body := []byte(`{"name":"` + name + `","source":"2001:db8::42","resource_name":"gitea","resource_uid":"resource-gitea-v1","endpoint_names":["https"],"duration_seconds":3600,"reason":"runtime independent source"}`)
		code, _, data := apiCall(t, p, "POST", "/api/v1/grants", body, match, adminToken)
		if code != 201 || json.Unmarshal(data, dst) != nil {
			t.Fatal(code, string(data))
		}
		published := snapshot(t, st)
		before = published
		if published.Revision != uint64(3+i) {
			t.Fatal(published)
		}
	}
	if first.UID == second.UID || first.UID == "" {
		t.Fatal(first, second)
	}
	find := func(s *agentv1.Snapshot, id string) *agentv1.Contribution {
		for _, target := range s.Targets {
			for _, g := range target.Grants {
				if g.Id == id {
					return g
				}
			}
		}
		return nil
	}
	one, two := find(before, first.UID), find(before, second.UID)
	if one == nil || two == nil || one.Source != "2001:db8::42/128" || one.ExpiresAt != first.ExpiresAt || two.ExpiresAt != second.ExpiresAt {
		t.Fatal(before)
	}
	if _, err = time.Parse(time.RFC3339Nano, one.ExpiresAt); err != nil {
		t.Fatal(err)
	}
	code, _, data := apiCall(t, p, "POST", "/api/v1/grants/"+first.UID+"/revoke", []byte(`{"reason":"revoke exactly one"}`), `"4"`, adminToken)
	if code != 200 {
		t.Fatal(code, string(data))
	}
	changed := snapshot(t, st)
	one, two = find(changed, first.UID), find(changed, second.UID)
	if changed.Revision != 5 || one == nil || !one.Revoked || one.ExpiresAt != first.ExpiresAt || two == nil || two.Revoked || two.ExpiresAt != second.ExpiresAt {
		t.Fatal(changed)
	}
	p.stop(t)
	p = start(t, c)
	ready(t, p, 200)
	recovered := snapshot(t, stream(t, p))
	one, two = find(recovered, first.UID), find(recovered, second.UID)
	if recovered.Revision != 5 || one == nil || !one.Revoked || two == nil || two.Revoked || one.ExpiresAt != first.ExpiresAt || two.ExpiresAt != second.ExpiresAt {
		t.Fatal("restart altered identity/deadline", recovered)
	}
}
