// SPDX-License-Identifier: Apache-2.0
package controllerapp

import (
	"encoding/json"
	"os"
	"testing"
	"time"

	"github.com/PicketX/PicketX/internal/agentsync"
	agentv1 "github.com/PicketX/PicketX/pkg/protocol/agent/v1alpha1"
)

// Actual gRPC reports are assertions, not a substitute for Linux kernel tests.
func TestAgentObservationsRuntime(t *testing.T) {
	c := managedConfig(t)
	p := start(t, c)
	ready(t, p, 200)
	read := func() agentsync.Observations {
		t.Helper()
		code, h, data := apiCall(t, p, "GET", "/api/v1/agent-observations", nil, "", adminToken)
		if code != 200 || h.Get("Cache-Control") != "no-store" {
			t.Fatalf("observations %d %s", code, data)
		}
		var v agentsync.Observations
		if e := json.Unmarshal(data, &v); e != nil {
			t.Fatal(e)
		}
		return v
	}
	v := read()
	if len(v.Nodes) != 1 || v.Nodes[0].Freshness != "unknown" || v.Nodes[0].Connection != "disconnected" {
		t.Fatal(v)
	}
	for _, tc := range []struct {
		token, path string
		code        int
	}{{"", "/api/v1/agent-observations", 401}, {adminToken, "/api/v1/agent-observations?x=1", 400}} {
		code, _, _ := apiCall(t, p, "GET", tc.path, nil, "", tc.token)
		if code != tc.code {
			t.Fatal(code, tc)
		}
	}
	st := stream(t, p)
	snap := snapshot(t, st)
	if e := st.Send(&agentv1.AgentFrame{Body: &agentv1.AgentFrame_Report{Report: &agentv1.Report{Revision: snap.Revision, Sha256: snap.Sha256, Phase: agentv1.Report_APPLIED, ObservedAt: "2099-01-01T00:00:00Z"}}}); e != nil {
		t.Fatal(e)
	}
	wait := func(check func(agentsync.NodeObservation) bool) agentsync.NodeObservation {
		t.Helper()
		deadline := time.Now().Add(3 * time.Second)
		for time.Now().Before(deadline) {
			v := read()
			if len(v.Nodes) == 1 && check(v.Nodes[0]) {
				return v.Nodes[0]
			}
			time.Sleep(time.Millisecond * 5)
		}
		t.Fatal("observation condition timed out")
		return agentsync.NodeObservation{}
	}
	n := wait(func(n agentsync.NodeObservation) bool {
		return n.Report != nil && n.Report.Phase == "APPLIED" && n.Freshness == "fresh" && n.MatchesDesired
	})
	if n.Report.ObservedAt != "2099-01-01T00:00:00Z" || n.Report.ReceivedAt.IsZero() {
		t.Fatal(n)
	}
	policy, _ := os.ReadFile("../../examples/static/gitea.yaml")
	facts, _ := os.ReadFile("../../examples/static/facts.json")
	body, _ := json.Marshal(map[string]string{"policy": string(policy), "facts": string(facts), "reason": "observe pending delivery"})
	if code, _, data := apiCall(t, p, "PUT", "/api/v1/publication", body, `"1"`, adminToken); code != 200 {
		t.Fatal(code, string(data))
	}
	snapshot(t, st)
	n = wait(func(n agentsync.NodeObservation) bool { return n.DesiredRevision == "2" && !n.MatchesDesired })
	if n.Report.Revision != "1" {
		t.Fatal(n)
	}
	if e := st.CloseSend(); e != nil {
		t.Fatal(e)
	}
	wait(func(n agentsync.NodeObservation) bool {
		return n.Connection == "disconnected" && n.Freshness == "stale"
	})
	next := stream(t, p)
	snapshot(t, next)
	wait(func(n agentsync.NodeObservation) bool { return n.Connection == "connected" && n.Freshness == "stale" })
}

func TestAgentObservationsRejectUnreadyPublisher(t *testing.T) {
	p := &publisher{}
	if _, e := p.AgentObservations(); e == nil {
		t.Fatal("unready served status")
	}
	p.ready.Store(true)
	p.failed = true
	if _, e := p.AgentObservations(); e == nil {
		t.Fatal("failed delivery served status")
	}
}
