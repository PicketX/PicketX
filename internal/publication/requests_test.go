// SPDX-License-Identifier: Apache-2.0
package publication

import (
	"bytes"
	"errors"
	"github.com/PicketX/PicketX/internal/accessrequest"
	"github.com/PicketX/PicketX/pkg/configuration"
	"path/filepath"
	"testing"
	"time"
)

// A workflow-event or publication-audit failure must roll back every durable
// write, not merely leave the process cache unchanged.
func TestAccessRequestApprovalAtomicRollbackAndRestart(t *testing.T) {
	path := filepath.Join(t.TempDir(), "state", "db.sqlite")
	s := managed(t, path)
	p, f := inputs(t)
	if _, err := s.AcceptManaged(0, "seed", "seed", p, f, nil); err != nil {
		t.Fatal(err)
	}
	r, err := s.CreateRequest(1, accessrequest.Record{UID: "request-one", Requester: "person:a", State: "pending", Version: 1, CreatedAt: time.Now().UTC(), ExpiresAt: time.Now().Add(time.Hour).UTC()})
	if err != nil {
		t.Fatal(err)
	}
	b, _ := configuration.Decode(p)
	expiry := r.ExpiresAt
	policy, err := b.AddGrant(configuration.GrantInput{Name: "request-one", UID: "grant-request-one", Source: "2001:db8::42", ResourceName: "gitea", ResourceUID: "resource-gitea-v1", EndpointNames: []string{"https"}, ExpiresAt: expiry})
	if err != nil {
		t.Fatal(err)
	}
	decided := r
	decided.State = "approved"
	decided.Version = 2
	decided.Approver = "person:b"
	decided.DecisionReason = "independent"
	decided.GrantUID = "grant-request-one"
	decided.Revision = 2
	decided.ApprovedExpiresAt = &expiry
	now := time.Now().UTC()
	decided.DecidedAt = &now
	for _, table := range []string{"controller_request_event", "controller_audit"} {
		if err := s.db.Exec("CREATE TRIGGER fail_request BEFORE INSERT ON " + table + " BEGIN SELECT RAISE(ABORT,'fixture failure'); END").Error; err != nil {
			t.Fatal(err)
		}
		if _, err = s.DecideRequest(1, decided, string(policy), string(f), nil); err == nil {
			t.Fatal("failed transaction succeeded")
		}
		current, _ := s.GetRequest(r.UID)
		doc, _ := s.Snapshot()
		if current.State != "pending" || current.Version != 1 || doc.Revision != 1 {
			t.Fatal(current, doc.Revision)
		}
		var durable uint64
		_ = s.db.Raw("SELECT revision FROM controller_publication WHERE id=1").Scan(&durable).Error
		if durable != 1 {
			t.Fatal(durable)
		}
		var count int64
		_ = s.db.Raw("SELECT count(*) FROM controller_request_event").Scan(&count).Error
		if count != 1 {
			t.Fatal("event leaked", count)
		}
		if err := s.db.Exec("DROP TRIGGER fail_request").Error; err != nil {
			t.Fatal(err)
		}
	}
	if _, err = s.DecideRequest(1, decided, string(policy), string(f), nil); err != nil {
		t.Fatal(err)
	}
	if _, err = s.DecideRequest(2, decided, string(policy), string(f), nil); !errors.Is(err, accessrequest.ErrConflict) {
		t.Fatal(err)
	}
	_ = s.Close()
	s = managed(t, path)
	got, err := s.GetRequest(r.UID)
	doc, _ := s.Snapshot()
	if err != nil || got.Requester != "person:a" || got.Approver != "person:b" || got.GrantUID != "grant-request-one" || got.Revision != 2 || !got.ExpiresAt.Equal(expiry) || doc.Revision != 2 || !bytes.Contains([]byte(doc.Policy), []byte("grant-request-one")) {
		t.Fatal(got, err, doc)
	}
}

// PR12's deferred UPDATE-CAS case: both processes recovered revision1, unlike
// first-INSERT competition. A stale update may not publish or approve anything.
func TestAccessRequestStaleRevisionOneUpdateCAS(t *testing.T) {
	path := filepath.Join(t.TempDir(), "state", "db.sqlite")
	a := managed(t, path)
	p, f := inputs(t)
	if _, err := a.AcceptManaged(0, "seed", "seed", p, f, nil); err != nil {
		t.Fatal(err)
	}
	b := managed(t, path)
	if _, err := a.AcceptManaged(1, "a", "new", bytes.Replace(p, []byte("10:00:00Z"), []byte("11:00:00Z"), 1), f, nil); err != nil {
		t.Fatal(err)
	}
	if _, err := b.AcceptManaged(1, "b", "stale", bytes.Replace(p, []byte("10:00:00Z"), []byte("12:00:00Z"), 1), f, nil); err == nil {
		t.Fatal("stale UPDATE succeeded")
	}
	if _, err := b.CreateRequest(1, accessrequest.Record{UID: "stale", Requester: "a", State: "pending", Version: 1}); !errors.Is(err, ErrPrecondition) {
		t.Fatal("stale submission", err)
	}
	events, _ := a.Audit(0, 100)
	if len(events) != 2 {
		t.Fatal(events)
	}
}
