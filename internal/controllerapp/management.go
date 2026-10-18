// SPDX-License-Identifier: Apache-2.0
package controllerapp

import (
	"errors"
	"fmt"

	"github.com/PicketX/PicketX/internal/management"
	"github.com/PicketX/PicketX/internal/publication"
)

const initialManagedPolicy = "apiVersion: picketx.io/v1alpha1\nkind: Configuration\nmetadata: {name: empty, uid: empty-v1}\nspec: {empty: true}\n"

func (p *publisher) initializeManaged() error {
	candidate, _ := p.store.Current()
	if candidate != nil {
		return nil
	}
	facts := fmt.Sprintf(`{"authorityID":%q,"agents":[],"contexts":[]}`, p.cfg.Agent.Authority)
	if _, e := p.store.AcceptManaged(0, "system:bootstrap", "initialize empty managed authority", []byte(initialManagedPolicy), []byte(facts), p.service.CheckPublication); e != nil {
		return e
	}
	return nil // Assembly publishes the recovered or initialized revision exactly once.
}

func (p *publisher) Snapshot() (publication.Document, error) { return p.store.Snapshot() }
func (p *publisher) Audit(after int64, limit int) ([]publication.AuditEvent, error) {
	return p.store.Audit(after, limit)
}

// Replace serializes durable commit and delivery with the credentials polling owner.
func (p *publisher) Replace(expected uint64, actor, reason, policy, facts string) (publication.Document, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.failed || !p.ready.Load() {
		return publication.Document{}, management.ErrUnavailable
	}
	changed, e := p.store.AcceptManaged(expected, actor, reason, []byte(policy), []byte(facts), p.service.CheckPublication)
	if e != nil {
		var persistence *publication.PersistenceError
		if errors.As(e, &persistence) {
			p.fail(e)
			return publication.Document{}, management.ErrUnavailable
		}
		return publication.Document{}, e
	}
	if changed {
		candidate, revision := p.store.Current()
		if e = p.service.Publish(candidate, revision); e != nil {
			p.fail(e)
			return publication.Document{}, management.ErrUnavailable
		}
		p.log.Info("accepted managed desired publication", "revision", revision, "actor", actor)
	}
	return p.store.Snapshot()
}
func (p *publisher) fail(e error) {
	p.failed = true
	p.ready.Store(false)
	if p.fatal != nil {
		p.fatal(e)
	}
}
