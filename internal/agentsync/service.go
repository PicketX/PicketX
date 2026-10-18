// SPDX-License-Identifier: Apache-2.0
// Package agentsync implements the Controller-side alpha stream. It opens no
// listener and does not prove that a remote Agent applied its reported state.
package agentsync

import (
	"fmt"
	"regexp"
	"sync"
	"time"

	"github.com/PicketX/PicketX/pkg/agentsnapshot"
	"github.com/PicketX/PicketX/pkg/configuration"
	"github.com/PicketX/PicketX/pkg/networkplan"
	agentv1 "github.com/PicketX/PicketX/pkg/protocol/agent/v1alpha1"
	"google.golang.org/grpc/codes"
	"google.golang.org/protobuf/proto"
)

const MaxWireBytes = 4 << 20

var identity = regexp.MustCompile(`^[A-Za-z0-9_.:/-]{1,128}$`)

type Limits struct {
	MaxCredentials, MaxStreams, MaxSnapshotBytes int
	EstablishPerSecond, MessagesPerSecond        int
	HelloTimeout, SendTimeout                    time.Duration
}
type Credential struct {
	ID, AgentRef string
	SHA256       [32]byte
}

// Observation holds an untrusted Agent assertion and trusted local receive time.
// Connected is true only if the reporting stream is still the active stream.
type Observation struct {
	Report     *agentv1.Report
	ReceivedAt time.Time
	Connected  bool
}
type observed struct {
	report   *agentv1.Report
	received time.Time
	session  *session
}
type Service struct {
	agentv1.UnimplementedAgentSyncServer
	mu            sync.Mutex
	authority     string
	limits        Limits
	closed        bool
	credentials   map[[32]byte]Credential
	sessions      map[string]*session
	snapshots     map[string]*agentv1.Snapshot
	reports       map[string]observed
	candidate     *configuration.Candidate
	revision      uint64
	establishment window
}

func New(authority string, l Limits) (*Service, error) {
	if !identity.MatchString(authority) || l.MaxCredentials < 1 || l.MaxCredentials > 4096 || l.MaxStreams < 1 || l.MaxStreams > 4096 || l.MaxSnapshotBytes < 1 || l.MaxSnapshotBytes > MaxWireBytes || l.EstablishPerSecond < 1 || l.MessagesPerSecond < 1 || l.HelloTimeout <= 0 || l.SendTimeout <= 0 {
		return nil, fmt.Errorf("invalid Agent stream configuration")
	}
	return &Service{authority: authority, limits: l, credentials: make(map[[32]byte]Credential), sessions: make(map[string]*session), snapshots: make(map[string]*agentv1.Snapshot), reports: make(map[string]observed)}, nil
}
func (s *Service) Publish(c *configuration.Candidate, revision uint64) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed || c == nil || c.Plan() == nil || c.Hash() == "" || c.Input().AuthorityID != s.authority || revision == 0 || revision <= s.revision {
		return fmt.Errorf("invalid Agent publication")
	}
	if s.candidate != nil {
		kinds := make(map[string]string)
		for _, o := range s.candidate.Objects() {
			kinds[o.UID] = o.Kind
		}
		for _, o := range c.Objects() {
			if k, ok := kinds[o.UID]; ok && k != o.Kind {
				return fmt.Errorf("publication changes object kind")
			}
		}
		if _, e := networkplan.Replace(s.candidate.Plan(), c.Input()); e != nil {
			return fmt.Errorf("publication violates replacement constraints")
		}
	}
	snapshots, e := s.buildScopes(c, revision, s.credentials)
	if e != nil {
		return e
	}
	s.candidate = c
	s.revision = revision
	s.snapshots = snapshots
	for _, ss := range s.sessions {
		ss.wake()
	}
	return nil
}
func (s *Service) buildScopes(c *configuration.Candidate, r uint64, cs map[[32]byte]Credential) (map[string]*agentv1.Snapshot, error) {
	out := make(map[string]*agentv1.Snapshot)
	if c == nil {
		return out, nil
	}
	for _, cred := range cs {
		if out[cred.AgentRef] != nil {
			continue
		}
		snapshot, e := agentsnapshot.Build(c, cred.AgentRef, r)
		if e != nil {
			return nil, fmt.Errorf("invalid scoped snapshot")
		}
		if proto.Size(&agentv1.ControllerFrame{Snapshot: snapshot}) > s.limits.MaxSnapshotBytes {
			return nil, fmt.Errorf("Agent snapshot exceeds byte limit")
		}
		out[cred.AgentRef] = snapshot
	}
	return out, nil
}
func (s *Service) Report(agent string) (Observation, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	v, ok := s.reports[agent]
	if !ok {
		return Observation{}, false
	}
	return Observation{Report: proto.Clone(v.report).(*agentv1.Report), ReceivedAt: v.received, Connected: !s.closed && s.sessions[agent] == v.session}, true
}
func (s *Service) Close() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return
	}
	s.closed = true
	for _, ss := range s.sessions {
		ss.end(codes.Unavailable)
	}
	s.sessions = make(map[string]*session)
}

type window struct {
	start time.Time
	count int
}

func (w *window) take(now time.Time, limit int) bool {
	if w.start.IsZero() || !now.Before(w.start.Add(time.Second)) {
		w.start = now
		w.count = 0
	}
	if w.count >= limit {
		return false
	}
	w.count++
	return true
}
