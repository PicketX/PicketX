// SPDX-License-Identifier: Apache-2.0
// Package authz protects trusted request-context adapters, independently of
// management/Agent identities. Caller scope is an upper bound, never a Grant.
package authz

import (
	"errors"
	"github.com/PicketX/PicketX/internal/publication"
	"github.com/PicketX/PicketX/pkg/httpauth"
	"slices"
	"time"
)

var (
	ErrUnauthorized = errors.New("authorization caller unauthenticated")
	ErrForbidden    = errors.New("authorization route forbidden")
	ErrUnavailable  = errors.New("authorization unavailable")
	ErrInvalid      = errors.New("invalid request context")
)

type Repository interface {
	DecideHTTP(string, httpauth.Query) (publication.HTTPDecision, error)
}
type Service struct {
	credentials string
	repository  Repository
}

func New(path string, repo Repository) (*Service, error) {
	if repo == nil {
		return nil, ErrUnavailable
	}
	if _, e := load(path); e != nil {
		return nil, e
	}
	return &Service{credentials: path, repository: repo}, nil
}
func (s *Service) Ready() bool { _, e := load(s.credentials); return e == nil }
func (s *Service) Decide(token string, q httpauth.Query) (publication.HTTPDecision, error) {
	caller, e := authenticate(s.credentials, token)
	if e != nil {
		return publication.HTTPDecision{}, e
	}
	if !slices.Contains(caller.Routes, q.Route) {
		return publication.HTTPDecision{}, ErrForbidden
	}
	q.At = time.Now().UTC()
	if e = httpauth.Validate(q); e != nil {
		return publication.HTTPDecision{}, ErrInvalid
	}
	d, e := s.repository.DecideHTTP(caller.ID, q)
	if errors.Is(e, publication.ErrHTTPRoute) {
		return publication.HTTPDecision{}, ErrForbidden
	}
	if e != nil {
		return publication.HTTPDecision{}, ErrUnavailable
	}
	return d, nil
}
