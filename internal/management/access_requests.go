// SPDX-License-Identifier: Apache-2.0
package management

import (
	"errors"
	"github.com/PicketX/PicketX/internal/accessrequest"
	"github.com/PicketX/PicketX/internal/publication"
	"github.com/PicketX/PicketX/pkg/configuration"
	"github.com/PicketX/PicketX/pkg/core"
	"reflect"
	"slices"
	"strings"
	"time"
)

type requestRepository interface {
	CreateRequest(uint64, accessrequest.Record) (accessrequest.Record, error)
	GetRequest(string) (accessrequest.Record, error)
	ListRequests(string, int64, int) (accessrequest.Page, error)
	DecideAccessRequest(uint64, accessrequest.Record, string, string) (accessrequest.Record, error)
}
type RequestCatalog struct {
	Revision   uint64                            `json:"revision"`
	Resources  []configuration.InventoryResource `json:"resources"`
	ObservedAt time.Time                         `json:"observed_at"`
}

func (s *Service) RequestCatalog(p Principal) (RequestCatalog, error) {
	if err := s.Check(p, "request:create"); err != nil {
		return RequestCatalog{}, err
	}
	doc, err := s.repository.Snapshot()
	if err != nil {
		return RequestCatalog{}, ErrUnavailable
	}
	b, err := configuration.Decode([]byte(doc.Policy))
	if err != nil {
		return RequestCatalog{}, ErrUnavailable
	}
	return RequestCatalog{doc.Revision, b.Inventory(configuration.Facts{}).Resources, time.Now().UTC()}, nil
}

// selectedRequestResource freezes only named entries; no dynamic selection is
// created, and address/protocol/port changes must be reviewed as a new request.
func selectedRequestResource(b *configuration.Bundle, name, uid string, names []string) (configuration.InventoryResource, error) {
	if len(names) < 1 || len(names) > 64 {
		return configuration.InventoryResource{}, ErrInvalid
	}
	wanted := slices.Clone(names)
	slices.Sort(wanted)
	if len(slices.Compact(slices.Clone(wanted))) != len(wanted) {
		return configuration.InventoryResource{}, ErrInvalid
	}
	for _, r := range b.Inventory(configuration.Facts{}).Resources {
		if r.Name != name || r.UID != uid {
			continue
		}
		result := configuration.InventoryResource{Name: r.Name, UID: r.UID, Endpoints: []configuration.InventoryEndpoint{}}
		for _, name := range wanted {
			found := false
			for _, e := range r.Endpoints {
				if e.Name == name {
					result.Endpoints = append(result.Endpoints, e)
					found = true
					break
				}
			}
			if !found {
				return result, ErrInvalid
			}
		}
		return result, nil
	}
	return configuration.InventoryResource{}, ErrInvalid
}
func requestError(err error) error {
	if err == nil || errors.Is(err, publication.ErrPrecondition) || errors.Is(err, accessrequest.ErrConflict) || errors.Is(err, accessrequest.ErrNotFound) || errors.Is(err, ErrUnavailable) {
		return err
	}
	return ErrUnavailable
}
func (s *Service) CreateAccessRequest(p Principal, expected uint64, in accessrequest.Input) (accessrequest.Record, error) {
	if err := s.Check(p, "request:create"); err != nil {
		return accessrequest.Record{}, err
	}
	repo, ok := s.repository.(requestRepository)
	if !ok {
		return accessrequest.Record{}, ErrUnavailable
	}
	if in.DurationSeconds < 60 || in.DurationSeconds > 86400 || strings.TrimSpace(in.Reason) == "" || len(in.Reason) > 512 {
		return accessrequest.Record{}, ErrInvalid
	}
	source, err := core.ParseSource(in.Source)
	if err != nil {
		return accessrequest.Record{}, ErrInvalid
	}
	doc, err := s.repository.Snapshot()
	if err != nil {
		return accessrequest.Record{}, ErrUnavailable
	}
	if doc.Revision != expected {
		return accessrequest.Record{}, publication.ErrPrecondition
	}
	b, err := configuration.Decode([]byte(doc.Policy))
	if err != nil {
		return accessrequest.Record{}, ErrUnavailable
	}
	resource, err := selectedRequestResource(b, in.ResourceName, in.ResourceUID, in.EndpointNames)
	if err != nil {
		return accessrequest.Record{}, err
	}
	uid, err := inventoryUID("request-")
	if err != nil {
		return accessrequest.Record{}, err
	}
	now := time.Now().UTC()
	names := slices.Clone(in.EndpointNames)
	slices.Sort(names)
	r := accessrequest.Record{UID: uid, Requester: p.ID, Source: source.String(), Resource: resource, EndpointNames: names, ExpiresAt: now.Add(time.Duration(in.DurationSeconds) * time.Second), Reason: in.Reason, CreatedAt: now, State: "pending", Version: 1}
	out, err := repo.CreateRequest(expected, r)
	return out, requestError(err)
}
func (s *Service) AccessRequests(p Principal, after int64, limit int) (accessrequest.Page, error) {
	owner := ""
	if err := s.Check(p, "request:review"); err != nil {
		if !errors.Is(err, ErrForbidden) {
			return accessrequest.Page{}, err
		}
		if err = s.Check(p, "request:read-own"); err != nil {
			return accessrequest.Page{}, err
		}
		owner = p.ID
	}
	if after < 0 || limit < 1 || limit > 100 {
		return accessrequest.Page{}, ErrInvalid
	}
	repo, ok := s.repository.(requestRepository)
	if !ok {
		return accessrequest.Page{}, ErrUnavailable
	}
	page, err := repo.ListRequests(owner, after, limit)
	return page, requestError(err)
}
func (s *Service) DecideAccessRequest(p Principal, uid string, expected uint64, in accessrequest.Decision) (accessrequest.Record, error) {
	if err := s.Check(p, "request:review"); err != nil {
		return accessrequest.Record{}, err
	}
	if strings.TrimSpace(in.Reason) == "" || len(in.Reason) > 400 || in.Version == 0 {
		return accessrequest.Record{}, ErrInvalid
	}
	repo, ok := s.repository.(requestRepository)
	if !ok {
		return accessrequest.Record{}, ErrUnavailable
	}
	r, err := repo.GetRequest(uid)
	if err != nil {
		return r, requestError(err)
	}
	if r.State != "pending" || in.Version != r.Version {
		return r, accessrequest.ErrConflict
	}
	now := time.Now().UTC()
	r.Version++
	r.Approver = p.ID
	r.DecisionReason = in.Reason
	r.DecidedAt = &now
	policy, facts := "", ""
	switch in.Action {
	case "reject":
		if in.EndpointNames != nil || in.ExpiresAt != "" {
			return r, ErrInvalid
		}
		r.State = "rejected"
	case "approve":
		if err = s.Check(p, "publication:write"); err != nil {
			return r, err
		}
		if p.ID == r.Requester {
			return r, ErrForbidden
		}
		expiry, e := time.Parse(time.RFC3339Nano, in.ExpiresAt)
		if e != nil || !expiry.After(now) || expiry.After(r.ExpiresAt) || len(in.EndpointNames) < 1 {
			return r, ErrInvalid
		}
		for _, name := range in.EndpointNames {
			if !slices.Contains(r.EndpointNames, name) {
				return r, ErrInvalid
			}
		}
		doc, e := s.repository.Snapshot()
		if e != nil {
			return r, ErrUnavailable
		}
		if doc.Revision != expected {
			return r, publication.ErrPrecondition
		}
		b, e := configuration.Decode([]byte(doc.Policy))
		if e != nil {
			return r, ErrUnavailable
		}
		frozen, e := selectedRequestResource(b, r.Resource.Name, r.Resource.UID, r.EndpointNames)
		if e != nil || !reflect.DeepEqual(frozen, r.Resource) {
			return r, accessrequest.ErrConflict
		}
		grant, e := inventoryUID("grant-")
		if e != nil {
			return r, e
		}
		added, e := b.AddGrant(configuration.GrantInput{Name: r.UID, UID: grant, Source: r.Source, ResourceName: r.Resource.Name, ResourceUID: r.Resource.UID, EndpointNames: in.EndpointNames, ExpiresAt: expiry})
		if e != nil {
			return r, ErrInvalid
		}
		r.State = "approved"
		r.GrantUID = grant
		r.ApprovedEndpoints = slices.Clone(in.EndpointNames)
		slices.Sort(r.ApprovedEndpoints)
		expiry = expiry.UTC()
		r.ApprovedExpiresAt = &expiry
		r.Revision = expected + 1
		policy, facts = string(added), doc.Facts
	default:
		return r, ErrInvalid
	}
	out, err := repo.DecideAccessRequest(expected, r, policy, facts)
	return out, requestError(err)
}
