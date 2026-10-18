// SPDX-License-Identifier: Apache-2.0
package agentsync

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	agentv1 "github.com/PicketX/PicketX/pkg/protocol/agent/v1alpha1"
)

// Removing current-session, age or revision correlation would falsely turn
// historical APPLIED into a current assertion; each axis is tested separately.
func TestObservationsCorrelationAndAge(t *testing.T) {
	s, _ := setup(t, limits())
	if e := s.Publish(candidate(t, false), 9007199254740993); e != nil {
		t.Fatal(e)
	}
	now := time.Date(2026, 10, 19, 0, 0, 0, 0, time.UTC)
	ss := &session{}
	s.sessions["agent-a"] = ss
	v, e := s.observationsAt(now)
	if e != nil {
		t.Fatal(e)
	}
	n := v.Nodes[0]
	if n.Freshness != "unknown" || n.Report != nil || n.Connection != "connected" || n.DesiredRevision != "9007199254740993" || n.MatchesDesired {
		t.Fatalf("unreported %+v", n)
	}
	snap := s.snapshots["agent-a"]
	s.reports["agent-a"] = observed{report: &agentv1.Report{Revision: snap.Revision, Sha256: append([]byte(nil), snap.Sha256...), Phase: agentv1.Report_APPLIED, ObservedAt: "2099-01-01T00:00:00Z"}, received: now, session: ss}
	for _, tc := range []struct {
		age       time.Duration
		fresh     string
		remaining int64
	}{{0, "fresh", 60000}, {59 * time.Second, "fresh", 1000}, {60 * time.Second, "stale", 0}, {-time.Second, "stale", 0}} {
		v, e = s.observationsAt(now.Add(tc.age))
		if e != nil {
			t.Fatal(e)
		}
		n = v.Nodes[0]
		if n.Freshness != tc.fresh || n.FreshForMillis != tc.remaining || !n.MatchesDesired || n.Report.Phase != "APPLIED" {
			t.Fatalf("age %v: %+v", tc.age, n)
		}
	}
	raw, _ := json.Marshal(v)
	if !strings.Contains(string(raw), `"revision":"9007199254740993"`) {
		t.Fatalf("unsafe JS revision: %s", raw)
	}
	v.Nodes[0].Report.Phase = "FAILED"
	v, _ = s.observationsAt(now)
	if v.Nodes[0].Report.Phase != "APPLIED" {
		t.Fatal("returned view aliases internal report")
	}
	if e = s.Publish(candidate(t, true), 9007199254740994); e != nil {
		t.Fatal(e)
	}
	v, _ = s.observationsAt(now)
	if v.Nodes[0].MatchesDesired {
		t.Fatal("old revision reported current")
	}
	s.reports["agent-a"].report.Revision = 9007199254740994
	v, _ = s.observationsAt(now)
	if v.Nodes[0].MatchesDesired {
		t.Fatal("hash mismatch reported current")
	}
	delete(s.sessions, "agent-a")
	v, _ = s.observationsAt(now)
	if v.Nodes[0].Connection != "disconnected" || v.Nodes[0].Freshness != "stale" {
		t.Fatal(v)
	}
	s.sessions["agent-a"] = &session{}
	v, _ = s.observationsAt(now)
	if v.Nodes[0].Connection != "connected" || v.Nodes[0].Freshness != "stale" || v.Nodes[0].FreshForMillis != 0 {
		t.Fatal("replacement freshened old report", v)
	}
	// Test-only sessions have no terminal channel; remove before lifecycle cleanup.
	delete(s.sessions, "agent-a")
}

func TestObservationsCredentialScopeAndClosed(t *testing.T) {
	s, _ := setup(t, limits())
	if e := s.ReplaceCredentials([]Credential{credential("b", "z-node", tokenB), credential("a", "agent-a", tokenA), credential("a2", "agent-a", strings.Repeat("c", 32))}); e != nil {
		t.Fatal(e)
	}
	v, e := s.Observations()
	if e != nil || len(v.Nodes) != 2 || v.Nodes[0].AgentRef != "agent-a" || v.Nodes[1].AgentRef != "z-node" {
		t.Fatalf("dedup/sort %+v %v", v, e)
	}
	if v.Nodes[0].DesiredRevision != "" || v.Nodes[0].Report != nil {
		t.Fatal("invented desired/report", v)
	}
	if e = s.ReplaceCredentials(nil); e != nil {
		t.Fatal(e)
	}
	v, e = s.Observations()
	if e != nil || v.Nodes == nil || len(v.Nodes) != 0 {
		t.Fatal("revoked/empty", v, e)
	}
	s.Close()
	if _, e = s.Observations(); e == nil {
		t.Fatal("closed service served observations")
	}
}
