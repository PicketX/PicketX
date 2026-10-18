// SPDX-License-Identifier: Apache-2.0
package publication

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/PicketX/PicketX/pkg/httpauth"
	"gorm.io/gorm"
	"time"
)

var ErrHTTPRoute = errors.New("HTTP route unavailable")

type HTTPDecision struct {
	Allowed    bool      `json:"allowed"`
	Reason     string    `json:"reason"`
	Revision   uint64    `json:"revision"`
	GrantIDs   []string  `json:"grant_ids"`
	DenyIDs    []string  `json:"deny_ids"`
	NextChange time.Time `json:"next_change"`
	CacheTTL   int       `json:"cache_ttl"`
}
type HTTPAuditEvent struct {
	ID     int64  `json:"id"`
	Caller string `json:"caller"`
	Route  string `json:"route"`
	Source string `json:"source"`
	HTTPDecision
	At time.Time `json:"at"`
}

func migrateHTTP(tx *gorm.DB) error {
	if e := tx.Exec("CREATE TABLE controller_http_audit (id INTEGER PRIMARY KEY AUTOINCREMENT, caller TEXT NOT NULL, route TEXT NOT NULL, source TEXT NOT NULL, revision INTEGER NOT NULL CHECK(revision>0), allowed BOOLEAN NOT NULL, reason TEXT NOT NULL, grants TEXT NOT NULL, denies TEXT NOT NULL, at DATETIME NOT NULL)").Error; e != nil {
		return e
	}
	return tx.Exec("UPDATE controller_schema SET version=3 WHERE id=1").Error
}

// DecideHTTP serializes evaluation and durable evidence with publication. An old
// candidate cannot issue an allow after a newer restrictive commit is accepted.
func (s *Store) DecideHTTP(caller string, q httpauth.Query) (HTTPDecision, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed || s.candidate == nil || !httpauth.ValidID(caller) {
		return HTTPDecision{}, fmt.Errorf("HTTP authority unavailable")
	}
	if !s.candidate.HTTP().HasRoute(q.Route) {
		return HTTPDecision{}, ErrHTTPRoute
	}
	d, e := s.candidate.HTTP().Decide(q)
	if e != nil {
		return HTTPDecision{}, e
	}
	out := HTTPDecision{Allowed: d.Allowed, Reason: string(d.Reason), Revision: s.revision, GrantIDs: d.GrantIDs, DenyIDs: d.DenyIDs, NextChange: d.NextChange}
	if out.GrantIDs == nil {
		out.GrantIDs = []string{}
	}
	if out.DenyIDs == nil {
		out.DenyIDs = []string{}
	}
	grants, _ := json.Marshal(out.GrantIDs)
	denies, _ := json.Marshal(out.DenyIDs)
	if e = s.db.Exec("INSERT INTO controller_http_audit(caller,route,source,revision,allowed,reason,grants,denies,at) VALUES(?,?,?,?,?,?,?,?,?)", caller, q.Route, q.Source.String(), int64(s.revision), out.Allowed, out.Reason, string(grants), string(denies), q.At.UTC()).Error; e != nil {
		return HTTPDecision{}, &PersistenceError{Err: e}
	}
	return out, nil
}
func (s *Store) HTTPAudit(after int64, limit int) ([]HTTPAuditEvent, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed || after < 0 || limit < 1 || limit > 100 {
		return nil, fmt.Errorf("invalid HTTP audit query")
	}
	var rows []struct {
		ID                     int64
		Caller, Route, Source  string
		Revision               uint64
		Allowed                bool
		Reason, Grants, Denies string
		At                     time.Time
	}
	if e := s.db.Raw("SELECT id,caller,route,source,revision,allowed,reason,grants,denies,at FROM controller_http_audit WHERE id>? ORDER BY id ASC LIMIT ?", after, limit).Scan(&rows).Error; e != nil {
		return nil, &PersistenceError{Err: e}
	}
	out := []HTTPAuditEvent{}
	for _, r := range rows {
		ev := HTTPAuditEvent{ID: r.ID, Caller: r.Caller, Route: r.Route, Source: r.Source, HTTPDecision: HTTPDecision{Allowed: r.Allowed, Reason: r.Reason, Revision: r.Revision}, At: r.At}
		if e := json.Unmarshal([]byte(r.Grants), &ev.GrantIDs); e != nil {
			return nil, e
		}
		if e := json.Unmarshal([]byte(r.Denies), &ev.DenyIDs); e != nil {
			return nil, e
		}
		out = append(out, ev)
	}
	return out, nil
}
