// SPDX-License-Identifier: Apache-2.0
package publication

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/PicketX/PicketX/internal/accessrequest"
	"github.com/PicketX/PicketX/pkg/configuration"
	"gorm.io/gorm"
	"time"
)

func migrateRequests(tx *gorm.DB) error {
	for _, q := range []string{
		"CREATE TABLE controller_access_request (id INTEGER PRIMARY KEY AUTOINCREMENT, uid TEXT NOT NULL UNIQUE, requester TEXT NOT NULL, state TEXT NOT NULL CHECK(state IN ('pending','approved','rejected')), version INTEGER NOT NULL CHECK(version>0), data BLOB NOT NULL)",
		"CREATE INDEX controller_access_request_owner ON controller_access_request(requester,id)",
		"CREATE TABLE controller_request_event (id INTEGER PRIMARY KEY AUTOINCREMENT, request_uid TEXT NOT NULL REFERENCES controller_access_request(uid), actor TEXT NOT NULL, action TEXT NOT NULL, at DATETIME NOT NULL, data BLOB NOT NULL)",
		"UPDATE controller_schema SET version=4 WHERE id=1",
	} {
		if err := tx.Exec(q).Error; err != nil {
			return err
		}
	}
	return nil
}

func requestRow(tx *gorm.DB, uid string) (accessrequest.Record, error) {
	var row struct {
		ID   int64
		Data []byte
	}
	result := tx.Raw("SELECT id,data FROM controller_access_request WHERE uid=?", uid).Scan(&row)
	if result.Error != nil {
		return accessrequest.Record{}, result.Error
	}
	if result.RowsAffected != 1 {
		return accessrequest.Record{}, accessrequest.ErrNotFound
	}
	var out accessrequest.Record
	if err := json.Unmarshal(row.Data, &out); err != nil {
		return out, err
	}
	out.ID = row.ID
	return out, nil
}
func (s *Store) GetRequest(uid string) (accessrequest.Record, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed || s.mode != "database" {
		return accessrequest.Record{}, fmt.Errorf("request store unavailable")
	}
	return requestRow(s.db, uid)
}
func (s *Store) ListRequests(owner string, after int64, limit int) (accessrequest.Page, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := accessrequest.Page{Requests: []accessrequest.Record{}, ObservedAt: time.Now().UTC()}
	if s.closed || s.mode != "database" || after < 0 || limit < 1 || limit > 100 {
		return out, fmt.Errorf("invalid request query")
	}
	var rows []struct {
		ID   int64
		Data []byte
	}
	q := "SELECT id,data FROM controller_access_request WHERE id>?"
	args := []any{after}
	if owner != "" {
		q += " AND requester=?"
		args = append(args, owner)
	}
	q += " ORDER BY id ASC LIMIT ?"
	args = append(args, limit+1)
	if err := s.db.Raw(q, args...).Scan(&rows).Error; err != nil {
		return out, &PersistenceError{Err: err}
	}
	out.HasMore = len(rows) > limit
	for _, row := range rows[:min(len(rows), limit)] {
		var r accessrequest.Record
		if err := json.Unmarshal(row.Data, &r); err != nil {
			return out, &PersistenceError{Err: err}
		}
		r.ID = row.ID
		out.Requests = append(out.Requests, r)
		out.NextAfter = r.ID
	}
	return out, nil
}
func (s *Store) CreateRequest(expected uint64, r accessrequest.Record) (accessrequest.Record, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed || s.mode != "database" {
		return r, fmt.Errorf("requests disabled")
	}
	if s.revision != expected {
		return r, ErrPrecondition
	}
	data, err := json.Marshal(r)
	if err != nil {
		return r, err
	}
	err = s.db.Transaction(func(tx *gorm.DB) error {
		// Submission is bound to the actual durable authority, not a stale process cache.
		var rev uint64
		if e := tx.Raw("SELECT revision FROM controller_publication WHERE id=1 AND authority=?", s.authority).Scan(&rev).Error; e != nil {
			return e
		}
		if rev != expected {
			return ErrPrecondition
		}
		if e := tx.Exec("INSERT INTO controller_access_request(uid,requester,state,version,data) VALUES(?,?,?,?,?)", r.UID, r.Requester, r.State, r.Version, data).Error; e != nil {
			return e
		}
		return tx.Exec("INSERT INTO controller_request_event(request_uid,actor,action,at,data) VALUES(?,?,?,?,?)", r.UID, r.Requester, "created", r.CreatedAt, data).Error
	})
	if errors.Is(err, ErrPrecondition) {
		return r, err
	}
	if err != nil {
		return r, &PersistenceError{Err: err}
	}
	return requestRow(s.db, r.UID)
}

// DecideRequest holds the same store mutex as publication. The callback runs
// inside publication's transaction, before memory and stream state advance.
func (s *Store) DecideRequest(expected uint64, r accessrequest.Record, policy, facts string, preflight func(*configuration.Candidate, uint64) error) (accessrequest.Record, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed || s.mode != "database" {
		return r, fmt.Errorf("requests disabled")
	}
	current, err := requestRow(s.db, r.UID)
	if err != nil {
		return r, err
	}
	if current.State != "pending" || current.Version+1 != r.Version {
		return r, accessrequest.ErrConflict
	}
	hook := func(tx *gorm.DB) error {
		if r.State == "approved" && (r.ApprovedExpiresAt == nil || !r.ApprovedExpiresAt.After(time.Now()) || r.ApprovedExpiresAt.After(current.ExpiresAt)) {
			return accessrequest.ErrConflict
		}
		data, e := json.Marshal(r)
		if e != nil {
			return e
		}
		result := tx.Exec("UPDATE controller_access_request SET state=?,version=?,data=? WHERE uid=? AND state='pending' AND version=?", r.State, r.Version, data, r.UID, current.Version)
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected != 1 {
			return accessrequest.ErrConflict
		}
		return tx.Exec("INSERT INTO controller_request_event(request_uid,actor,action,at,data) VALUES(?,?,?,?,?)", r.UID, r.Approver, r.State, r.DecidedAt, data).Error
	}
	if r.State == "approved" {
		if expected != s.revision {
			return r, ErrPrecondition
		}
		changed, e := s.acceptWithTransaction([]byte(policy), []byte(facts), r.Approver, "approve "+r.UID+": "+r.DecisionReason, preflight, hook)
		if e != nil {
			if errors.Is(e, accessrequest.ErrConflict) {
				return r, accessrequest.ErrConflict
			}
			return r, e
		}
		if !changed {
			return r, accessrequest.ErrConflict
		}
	} else if r.State == "rejected" {
		if err = s.db.Transaction(hook); err != nil {
			if errors.Is(err, accessrequest.ErrConflict) {
				return r, err
			}
			return r, &PersistenceError{Err: err}
		}
	} else {
		return r, fmt.Errorf("invalid request decision")
	}
	return requestRow(s.db, r.UID)
}
