// SPDX-License-Identifier: Apache-2.0
package publication

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/PicketX/PicketX/pkg/configuration"
	"gorm.io/gorm"
)

var ErrPrecondition = errors.New("publication revision precondition failed")

// Document retains the accepted presence-sensitive input, not a lossy ORM/domain serialization.
type Document struct {
	Authority string `json:"authority"`
	Revision  uint64 `json:"revision"`
	Policy    string `json:"policy"`
	Facts     string `json:"facts"`
}
type AuditEvent struct {
	ID       int64     `json:"id"`
	Actor    string    `json:"actor"`
	Reason   string    `json:"reason"`
	Revision uint64    `json:"revision"`
	Digest   string    `json:"digest"`
	At       time.Time `json:"at"`
}

func migrateManaged(tx *gorm.DB, requested string) error {
	var count int64
	if e := tx.Raw("SELECT count(*) FROM controller_publication").Scan(&count).Error; e != nil {
		return e
	}
	// Every deployed version1 record came from the file provider. Never adopt it implicitly.
	mode := requested
	if count > 0 {
		mode = "file"
	}
	for _, sql := range []string{
		"CREATE TABLE controller_source (id INTEGER PRIMARY KEY CHECK(id=1), mode TEXT NOT NULL CHECK(mode IN ('file','database')))",
		"CREATE TABLE controller_audit (id INTEGER PRIMARY KEY AUTOINCREMENT, actor TEXT NOT NULL, reason TEXT NOT NULL, revision INTEGER NOT NULL UNIQUE CHECK(revision>0), digest TEXT NOT NULL, at DATETIME NOT NULL)",
	} {
		if e := tx.Exec(sql).Error; e != nil {
			return e
		}
	}
	if e := tx.Exec("INSERT INTO controller_source(id,mode) VALUES(1,?)", mode).Error; e != nil {
		return e
	}
	return tx.Exec("UPDATE controller_schema SET version=2 WHERE id=1").Error
}

func (s *Store) Snapshot() (Document, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return Document{}, fmt.Errorf("publication store closed")
	}
	d := s.document
	d.Authority = s.authority
	return d, nil
}

func (s *Store) AcceptManaged(expected uint64, actor, reason string, policy, facts []byte, preflight func(*configuration.Candidate, uint64) error) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed || s.mode != "database" {
		return false, fmt.Errorf("managed publication disabled")
	}
	if expected != s.revision {
		return false, ErrPrecondition
	}
	if strings.TrimSpace(actor) == "" || len(actor) > 128 || strings.TrimSpace(reason) == "" || len(reason) > 512 {
		return false, fmt.Errorf("invalid publication provenance")
	}
	return s.accept(policy, facts, actor, reason, preflight)
}

func insertAudit(tx *gorm.DB, actor, reason string, revision uint64, digest string) error {
	return tx.Exec("INSERT INTO controller_audit(actor,reason,revision,digest,at) VALUES(?,?,?,?,?)", actor, reason, int64(revision), digest, time.Now().UTC()).Error
}

func (s *Store) Audit(after int64, limit int) ([]AuditEvent, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed || after < 0 || limit < 1 || limit > 100 {
		return nil, fmt.Errorf("invalid audit query")
	}
	events := []AuditEvent{}
	e := s.db.Raw("SELECT id,actor,reason,revision,digest,at FROM controller_audit WHERE id>? ORDER BY id ASC LIMIT ?", after, limit).Scan(&events).Error
	if e != nil {
		return nil, &PersistenceError{Err: e}
	}
	return events, nil
}

// GORM is an assembly-only infrastructure boundary for the official Casbin adapter.
// The caller does not own/close the database; stop dependent workers before Store.Close.
func (s *Store) GORM() *gorm.DB { return s.db }
