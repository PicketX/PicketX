// SPDX-License-Identifier: Apache-2.0
package controllerapp

import (
	"errors"
	"github.com/PicketX/PicketX/internal/accessrequest"
	"github.com/PicketX/PicketX/internal/management"
	"github.com/PicketX/PicketX/internal/publication"
)

func (p *publisher) CreateRequest(expected uint64, r accessrequest.Record) (accessrequest.Record, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.failed || !p.ready.Load() {
		return r, management.ErrUnavailable
	}
	out, err := p.store.CreateRequest(expected, r)
	var persistence *publication.PersistenceError
	if errors.As(err, &persistence) {
		p.fail(err)
		return out, management.ErrUnavailable
	}
	return out, err
}
func (p *publisher) GetRequest(uid string) (accessrequest.Record, error) {
	return p.store.GetRequest(uid)
}
func (p *publisher) ListRequests(owner string, after int64, limit int) (accessrequest.Page, error) {
	return p.store.ListRequests(owner, after, limit)
}
func (p *publisher) DecideAccessRequest(expected uint64, r accessrequest.Record, policy, facts string) (accessrequest.Record, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.failed || !p.ready.Load() {
		return r, management.ErrUnavailable
	}
	out, err := p.store.DecideRequest(expected, r, policy, facts, p.service.CheckPublication)
	if err != nil {
		var persistence *publication.PersistenceError
		if errors.As(err, &persistence) {
			p.fail(err)
			return out, management.ErrUnavailable
		}
		return out, err
	}
	if r.State == "approved" {
		candidate, revision := p.store.Current()
		if err = p.service.Publish(candidate, revision); err != nil {
			p.fail(err)
			return out, management.ErrUnavailable
		}
	}
	return out, nil
}
