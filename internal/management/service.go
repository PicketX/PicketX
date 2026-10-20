// SPDX-License-Identifier: Apache-2.0
// Package management defines authenticated configuration-management operations.
package management

import (
	"errors"
	"github.com/PicketX/PicketX/internal/agentsync"
	"github.com/PicketX/PicketX/internal/publication"
)

var (
	ErrUnauthorized = errors.New("management authentication required")
	ErrUnavailable  = errors.New("management service unavailable")
	ErrForbidden    = errors.New("management operation forbidden")
	ErrInvalid      = errors.New("invalid publication")
)

type Principal struct {
	ID    string   `json:"id"`
	Roles []string `json:"roles"`
}
type Access interface {
	Authenticate(string) (Principal, error)
	Allowed(Principal, string) (bool, error)
}
type Repository interface {
	Snapshot() (publication.Document, error)
	Audit(int64, int) ([]publication.AuditEvent, error)
	Replace(expected uint64, actor, reason, policy, facts string) (publication.Document, error)
}
type Service struct {
	access     Access
	repository Repository
}

func New(access Access, repository Repository) *Service {
	return &Service{access: access, repository: repository}
}
func (s *Service) Authenticate(token string) (Principal, error) { return s.access.Authenticate(token) }

func (s *Service) AuthenticatePassword(username, password string) (Principal, string, error) {
	a, ok := s.access.(interface {
		AuthenticatePassword(string, string) (Principal, string, error)
	})
	if !ok {
		return Principal{}, "", ErrUnauthorized
	}
	return a.AuthenticatePassword(username, password)
}
func (s *Service) PasswordLoginEnabled() (bool, error) {
	a, ok := s.access.(interface{ PasswordLoginEnabled() (bool, error) })
	if !ok {
		return false, nil
	}
	return a.PasswordLoginEnabled()
}

// External proof stays in its adapter; this boundary resolves only trusted humans.
func (s *Service) AuthenticateHuman(id string) (Principal, string, error) {
	a, ok := s.access.(interface {
		AuthenticateHuman(string) (Principal, string, error)
	})
	if !ok {
		return Principal{}, "", ErrUnauthorized
	}
	return a.AuthenticateHuman(id)
}
func (s *Service) ResolveHumanSession(id, fingerprint string) (Principal, error) {
	a, ok := s.access.(interface {
		ResolveHumanSession(string, string) (Principal, error)
	})
	if !ok {
		return Principal{}, ErrUnauthorized
	}
	return a.ResolveHumanSession(id, fingerprint)
}

func (s *Service) AgentObservations(p Principal) (agentsync.Observations, error) {
	if e := s.Check(p, "publication:read"); e != nil {
		return agentsync.Observations{}, e
	}
	repo, ok := s.repository.(interface {
		AgentObservations() (agentsync.Observations, error)
	})
	if !ok {
		return agentsync.Observations{}, ErrUnavailable
	}
	return repo.AgentObservations()
}
func (s *Service) ResolveSession(id, fingerprint string) (Principal, error) {
	resolver, ok := s.access.(interface {
		ResolveSession(string, string) (Principal, error)
	})
	if !ok {
		return Principal{}, ErrUnavailable
	}
	return resolver.ResolveSession(id, fingerprint)
}
func (s *Service) Check(p Principal, operation string) error {
	ok, e := s.access.Allowed(p, operation)
	if e != nil {
		return ErrUnavailable
	}
	if !ok {
		return ErrForbidden
	}
	return nil
}
func (s *Service) Snapshot(p Principal) (publication.Document, error) {
	if e := s.Check(p, "publication:read"); e != nil {
		return publication.Document{}, e
	}
	return s.repository.Snapshot()
}
func (s *Service) Replace(p Principal, expected uint64, reason, policy, facts string) (publication.Document, error) {
	if e := s.Check(p, "publication:write"); e != nil {
		return publication.Document{}, e
	}
	doc, e := s.repository.Replace(expected, p.ID, reason, policy, facts)
	if e == nil || errors.Is(e, publication.ErrPrecondition) || errors.Is(e, ErrUnavailable) {
		return doc, e
	}
	var persistence *publication.PersistenceError
	if errors.As(e, &persistence) {
		return publication.Document{}, ErrUnavailable
	}
	return publication.Document{}, ErrInvalid
}
func (s *Service) Audit(p Principal, after int64, limit int) ([]publication.AuditEvent, error) {
	if e := s.Check(p, "audit:read"); e != nil {
		return nil, e
	}
	return s.repository.Audit(after, limit)
}
func (s *Service) HTTPAudit(p Principal, after int64, limit int) ([]publication.HTTPAuditEvent, error) {
	if e := s.Check(p, "audit:read"); e != nil {
		return nil, e
	}
	repo, ok := s.repository.(interface {
		HTTPAudit(int64, int) ([]publication.HTTPAuditEvent, error)
	})
	if !ok {
		return nil, ErrUnavailable
	}
	return repo.HTTPAudit(after, limit)
}
