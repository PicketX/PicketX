// SPDX-License-Identifier: Apache-2.0
package management

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"slices"
	"strings"
	"time"

	"github.com/PicketX/PicketX/internal/publication"
	"github.com/PicketX/PicketX/pkg/configuration"
)

var ErrGrantNotFound = errors.New("Grant not found")

type GrantPage struct {
	Authority  string                       `json:"authority"`
	Revision   uint64                       `json:"revision"`
	ObservedAt time.Time                    `json:"observed_at"`
	Resources  []configuration.ResourceView `json:"resources"`
	Grants     []configuration.GrantView    `json:"grants"`
	NextAfter  string                       `json:"next_after"`
	HasMore    bool                         `json:"has_more"`
}
type GrantChange struct {
	UID       string `json:"uid"`
	Revision  uint64 `json:"revision"`
	ExpiresAt string `json:"expires_at,omitempty"`
}
type IssueGrant struct {
	Name, Source, ResourceName, ResourceUID string
	EndpointNames                           []string
	DurationSeconds                         int
}

func (s *Service) Grants(p Principal, after string, limit int, expected *uint64) (GrantPage, error) {
	if err := s.Check(p, "publication:read"); err != nil {
		return GrantPage{}, err
	}
	doc, err := s.repository.Snapshot()
	if err != nil {
		return GrantPage{}, ErrUnavailable
	}
	if expected != nil && *expected != doc.Revision {
		return GrantPage{}, publication.ErrPrecondition
	}
	if limit < 1 || limit > 100 {
		return GrantPage{}, ErrInvalid
	}
	page := GrantPage{Authority: doc.Authority, Revision: doc.Revision, ObservedAt: time.Now().UTC(), Resources: []configuration.ResourceView{}, Grants: []configuration.GrantView{}}
	if doc.Policy == "" {
		return page, nil
	}
	b, err := configuration.Decode([]byte(doc.Policy))
	if err != nil {
		return GrantPage{}, ErrUnavailable
	}
	var grants []configuration.GrantView
	page.Resources, grants = b.GrantCatalog()
	index := 0
	for index < len(grants) && grants[index].UID <= after {
		index++
	}
	end := min(index+limit, len(grants))
	page.Grants = slices.Clone(grants[index:end])
	page.HasMore = end < len(grants)
	if len(page.Grants) > 0 {
		page.NextAfter = page.Grants[len(page.Grants)-1].UID
	}
	return page, nil
}
func (s *Service) grantBase(p Principal, expected uint64, reason string) (publication.Document, *configuration.Bundle, error) {
	if err := s.Check(p, "publication:write"); err != nil {
		return publication.Document{}, nil, err
	}
	doc, err := s.repository.Snapshot()
	if err != nil {
		return doc, nil, ErrUnavailable
	}
	if doc.Revision != expected {
		return doc, nil, publication.ErrPrecondition
	}
	if strings.TrimSpace(reason) == "" || len(reason) > 512 {
		return doc, nil, ErrInvalid
	}
	b, err := configuration.Decode([]byte(doc.Policy))
	if err != nil {
		return doc, nil, ErrInvalid
	}
	return doc, b, nil
}
func (s *Service) IssueGrant(p Principal, expected uint64, reason string, in IssueGrant) (GrantChange, error) {
	doc, b, err := s.grantBase(p, expected, reason)
	if err != nil {
		return GrantChange{}, err
	}
	if in.DurationSeconds < 60 || in.DurationSeconds > 86400 {
		return GrantChange{}, ErrInvalid
	}
	var random [16]byte
	if _, err = rand.Read(random[:]); err != nil {
		return GrantChange{}, ErrUnavailable
	}
	uid := "grant-" + hex.EncodeToString(random[:])
	expiry := time.Now().UTC().Add(time.Duration(in.DurationSeconds) * time.Second)
	policy, err := b.AddGrant(configuration.GrantInput{Name: in.Name, UID: uid, Source: in.Source, ResourceName: in.ResourceName, ResourceUID: in.ResourceUID, EndpointNames: in.EndpointNames, ExpiresAt: expiry})
	if err != nil {
		return GrantChange{}, ErrInvalid
	}
	changed, err := s.Replace(p, expected, reason, string(policy), doc.Facts)
	if err != nil {
		return GrantChange{}, err
	}
	return GrantChange{UID: uid, Revision: changed.Revision, ExpiresAt: expiry.Format(time.RFC3339Nano)}, nil
}
func (s *Service) RevokeGrant(p Principal, expected uint64, reason, uid string) (GrantChange, error) {
	doc, b, err := s.grantBase(p, expected, reason)
	if err != nil {
		return GrantChange{}, err
	}
	policy, err := b.RevokeGrant(uid)
	if errors.Is(err, configuration.ErrGrantNotFound) {
		return GrantChange{}, ErrGrantNotFound
	}
	if err != nil {
		return GrantChange{}, ErrInvalid
	}
	changed, err := s.Replace(p, expected, reason, string(policy), doc.Facts)
	if err != nil {
		return GrantChange{}, err
	}
	return GrantChange{UID: uid, Revision: changed.Revision}, nil
}
