// SPDX-License-Identifier: Apache-2.0
package agentsync

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"sort"
	"strconv"
	"time"
)

// Observations is a process-local point-in-time view, not kernel verification.
// Only currently credentialed scopes appear. Revisions are decimal strings.
type Observations struct {
	Authority  string            `json:"authority"`
	ObservedAt time.Time         `json:"observed_at"`
	Nodes      []NodeObservation `json:"nodes"`
}
type NodeObservation struct {
	AgentRef        string         `json:"agent_ref"`
	DesiredRevision string         `json:"desired_revision"`
	DesiredSHA256   string         `json:"desired_sha256"`
	Connection      string         `json:"connection" enums:"connected,disconnected"`
	Freshness       string         `json:"freshness" enums:"unknown,stale,fresh"`
	FreshForMillis  int64          `json:"fresh_for_millis"`
	MatchesDesired  bool           `json:"matches_desired"`
	Report          *ReportedState `json:"report"`
}
type ReportedState struct {
	Revision   string    `json:"revision"`
	SHA256     string    `json:"sha256"`
	Phase      string    `json:"phase" enums:"RECEIVED,APPLIED,FAILED"`
	ObservedAt string    `json:"observed_at"`
	ReceivedAt time.Time `json:"received_at"`
}

func (s *Service) Observations() (Observations, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	// Sample after acquiring the lock; queued reads must not extend freshness.
	return s.observationsAt(time.Now().UTC())
}

// observationsAt requires Service.mu; the projection owns no mutable references.
func (s *Service) observationsAt(now time.Time) (Observations, error) {
	if s.closed {
		return Observations{}, fmt.Errorf("Agent service closed")
	}
	out := Observations{Authority: s.authority, ObservedAt: now, Nodes: []NodeObservation{}}
	scopes := map[string]bool{}
	for _, c := range s.credentials {
		scopes[c.AgentRef] = true
	}
	names := make([]string, 0, len(scopes))
	for name := range scopes {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		n := NodeObservation{AgentRef: name, Connection: "disconnected", Freshness: "unknown"}
		active := s.sessions[name]
		if active != nil {
			n.Connection = "connected"
		}
		desired := s.snapshots[name]
		if desired != nil {
			n.DesiredRevision = strconv.FormatUint(desired.Revision, 10)
			n.DesiredSHA256 = hex.EncodeToString(desired.Sha256)
		}
		if o, ok := s.reports[name]; ok {
			r := o.report
			n.Report = &ReportedState{Revision: strconv.FormatUint(r.Revision, 10), SHA256: hex.EncodeToString(r.Sha256), Phase: r.Phase.String(), ObservedAt: r.ObservedAt, ReceivedAt: o.received}
			n.MatchesDesired = desired != nil && r.Revision == desired.Revision && bytes.Equal(r.Sha256, desired.Sha256)
			n.Freshness = "stale"
			age := now.Sub(o.received)
			if active != nil && active == o.session && age >= 0 && age < 60*time.Second {
				n.FreshForMillis = (60*time.Second - age).Milliseconds()
				if n.FreshForMillis > 0 {
					n.Freshness = "fresh"
				}
			}
		}
		out.Nodes = append(out.Nodes, n)
	}
	return out, nil
}
