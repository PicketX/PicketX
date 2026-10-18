// SPDX-License-Identifier: Apache-2.0
package agentsync

import (
	"crypto/sha256"
	"fmt"
	"strings"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

// ReplaceCredentials atomically replaces the full credential set. Revocation
// takes effect for new admission and existing RPCs in this process. Persistence
// and cross-process propagation are the calling authoritative store's job.
func (s *Service) ReplaceCredentials(in []Credential) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed || len(in) > s.limits.MaxCredentials {
		return fmt.Errorf("invalid credential replacement")
	}
	next := make(map[[32]byte]Credential, len(in))
	ids := make(map[string]bool)
	var zero [32]byte
	for _, c := range in {
		if !identity.MatchString(c.ID) || !identity.MatchString(c.AgentRef) || c.SHA256 == zero || ids[c.ID] {
			return fmt.Errorf("invalid Agent credential")
		}
		if _, exists := next[c.SHA256]; exists {
			return fmt.Errorf("duplicate Agent credential")
		}
		ids[c.ID] = true
		next[c.SHA256] = c
	}
	snapshots, e := s.buildScopes(s.candidate, s.revision, next)
	if e != nil {
		return e
	}
	s.credentials = next
	s.snapshots = snapshots
	for agent, ss := range s.sessions {
		if c, ok := next[ss.key]; !ok || c != ss.credential {
			ss.end(codes.Unauthenticated)
			delete(s.sessions, agent)
		}
	}
	// Bound retained status to currently authorized scopes.
	scopes := make(map[string]bool)
	for _, c := range next {
		scopes[c.AgentRef] = true
	}
	for agent := range s.reports {
		if !scopes[agent] {
			delete(s.reports, agent)
		}
	}
	return nil
}
func (s *Service) admit(md metadata.MD) (*session, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return nil, status.Error(codes.Unavailable, "Agent service unavailable")
	}
	if !s.establishment.take(time.Now(), s.limits.EstablishPerSecond) {
		return nil, status.Error(codes.ResourceExhausted, "Agent establishment limit")
	}
	values := md.Get("authorization")
	if len(values) != 1 || !strings.HasPrefix(values[0], "Bearer ") {
		return nil, status.Error(codes.Unauthenticated, "invalid Agent credential")
	}
	token := strings.TrimPrefix(values[0], "Bearer ")
	if len(token) < 32 || len(token) > 256 {
		return nil, status.Error(codes.Unauthenticated, "invalid Agent credential")
	}
	for _, b := range []byte(token) {
		if b < 33 || b > 126 {
			return nil, status.Error(codes.Unauthenticated, "invalid Agent credential")
		}
	}
	key := sha256.Sum256([]byte(token))
	cred, ok := s.credentials[key]
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "invalid Agent credential")
	}
	old := s.sessions[cred.AgentRef]
	if old == nil && len(s.sessions) >= s.limits.MaxStreams {
		return nil, status.Error(codes.ResourceExhausted, "Agent stream limit")
	}
	ss := &session{key: key, credential: cred, updates: make(chan struct{}, 1), terminal: make(chan struct{})}
	if old != nil {
		old.end(codes.Aborted)
	}
	s.sessions[cred.AgentRef] = ss
	return ss, nil
}
