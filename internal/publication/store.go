// SPDX-License-Identifier: Apache-2.0
package publication

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math"
	"net/url"
	"os"
	"path/filepath"
	"sync"

	"github.com/PicketX/PicketX/pkg/configuration"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// Store owns one process's accepted publication. SQLite revision CAS rejects a
// concurrent stale writer; it does not provide multi-Controller cache coherence.
type Store struct {
	mu        sync.Mutex
	db        *gorm.DB
	authority string
	candidate *configuration.Candidate
	revision  uint64
	semantic  string
	closed    bool
	mode      string
	document  Document
}

// PersistenceError means durability failed; the runtime must stop and reopen.
type PersistenceError struct{ Err error }

func (e *PersistenceError) Error() string { return "publication persistence failed" }
func (e *PersistenceError) Unwrap() error { return e.Err }

type record struct {
	ID            int
	Authority     string
	Revision      int64
	Policy, Facts []byte
	Digest        string
}

func Open(path, authority string) (out *Store, err error) {
	return OpenMode(path, authority, "file")
}

func OpenMode(path, authority, mode string) (out *Store, err error) {
	if path == "" || authority == "" {
		return nil, fmt.Errorf("invalid publication database identity")
	}
	if mode != "file" && mode != "database" {
		return nil, fmt.Errorf("invalid publication source mode")
	}
	abs, e := filepath.Abs(path)
	if e != nil {
		return nil, e
	}
	dir := filepath.Dir(abs)
	if e = os.MkdirAll(dir, 0700); e != nil {
		return nil, e
	}
	info, e := os.Lstat(dir)
	if e != nil || !info.IsDir() || info.Mode().Perm()&0077 != 0 {
		return nil, fmt.Errorf("publication directory must be private")
	}
	f, e := os.OpenFile(abs, os.O_CREATE|os.O_EXCL|os.O_RDWR, 0600)
	if e == nil {
		_ = f.Close()
	} else if !os.IsExist(e) {
		return nil, e
	}
	info, e = os.Lstat(abs)
	if e != nil || !info.Mode().IsRegular() || info.Mode().Perm()&0077 != 0 {
		return nil, fmt.Errorf("publication database must be a private regular file")
	}
	uri := (&url.URL{Scheme: "file", Path: abs}).String() + "?_busy_timeout=5000&_journal_mode=DELETE&_synchronous=FULL&_foreign_keys=on"
	db, e := gorm.Open(sqlite.Open(uri), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if e != nil {
		return nil, fmt.Errorf("open publication database: %w", e)
	}
	sqlDB, e := db.DB()
	if e != nil {
		return nil, e
	}
	sqlDB.SetMaxOpenConns(1)
	sqlDB.SetMaxIdleConns(1)
	defer func() {
		if err != nil {
			_ = sqlDB.Close()
		}
	}()
	s := &Store{db: db, authority: authority, mode: mode}
	e = db.Transaction(func(tx *gorm.DB) error {
		if e := tx.Exec("CREATE TABLE IF NOT EXISTS controller_schema (id INTEGER PRIMARY KEY CHECK(id=1), version INTEGER NOT NULL)").Error; e != nil {
			return e
		}
		var version struct{ Version int }
		r := tx.Raw("SELECT version FROM controller_schema WHERE id=1").Scan(&version)
		if r.Error != nil {
			return r.Error
		}
		if r.RowsAffected == 0 {
			if e := tx.Exec("CREATE TABLE controller_publication (id INTEGER PRIMARY KEY CHECK(id=1), authority TEXT NOT NULL, revision INTEGER NOT NULL CHECK(revision>0), policy BLOB NOT NULL, facts BLOB NOT NULL, digest TEXT NOT NULL)").Error; e != nil {
				return e
			}
			if e := tx.Exec("INSERT INTO controller_schema(id,version) VALUES(1,1)").Error; e != nil {
				return e
			}
			version.Version = 1
		}
		if version.Version == 1 {
			if e := migrateManaged(tx, mode); e != nil {
				return e
			}
			version.Version = 2
		}
		if version.Version == 2 {
			if e := migrateHTTP(tx); e != nil {
				return e
			}
			version.Version = 3
		}
		if version.Version != 3 {
			return fmt.Errorf("unsupported publication schema version")
		}
		var source struct{ Mode string }
		result := tx.Raw("SELECT mode FROM controller_source WHERE id=1").Scan(&source)
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected != 1 || source.Mode != mode {
			return fmt.Errorf("publication source mode mismatch")
		}
		return nil
	})
	if e != nil {
		return nil, e
	}
	var r record
	result := db.Raw("SELECT id,authority,revision,policy,facts,digest FROM controller_publication WHERE id=1").Scan(&r)
	if result.Error != nil {
		return nil, result.Error
	}
	if result.RowsAffected != 0 {
		if r.Authority != authority || r.Revision < 1 || len(r.Policy) > configuration.MaxBytes || len(r.Facts) > configuration.MaxBytes || r.Digest != recordDigest(r) {
			return nil, fmt.Errorf("invalid publication database record")
		}
		c, semantic, e := compile(nil, r.Policy, r.Facts, authority)
		if e != nil {
			return nil, fmt.Errorf("invalid recovered publication: %w", e)
		}
		s.candidate = c
		s.semantic = semantic
		s.revision = uint64(r.Revision)
		s.document = Document{Authority: authority, Revision: s.revision, Policy: string(r.Policy), Facts: string(r.Facts)}
	}
	return s, nil
}
func (s *Store) Current() (*configuration.Candidate, uint64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.candidate, s.revision
}
func (s *Store) Accept(policy, facts []byte, preflight func(*configuration.Candidate, uint64) error) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.mode != "file" {
		return false, fmt.Errorf("file publication disabled")
	}
	return s.accept(policy, facts, "", "", preflight)
}

// accept requires s.mu. A managed audit and publication are committed together.
func (s *Store) accept(policy, facts []byte, actor, reason string, preflight func(*configuration.Candidate, uint64) error) (bool, error) {
	if s.closed {
		return false, fmt.Errorf("publication store closed")
	}
	c, semantic, e := compile(s.candidate, policy, facts, s.authority)
	if e != nil {
		return false, e
	}
	if semantic == s.semantic {
		return false, nil
	}
	if s.revision >= math.MaxInt64 {
		return false, fmt.Errorf("publication revision exhausted")
	}
	next := s.revision + 1
	if preflight != nil {
		if e = preflight(c, next); e != nil {
			return false, e
		}
	}
	r := record{ID: 1, Authority: s.authority, Revision: int64(next), Policy: append([]byte(nil), policy...), Facts: append([]byte(nil), facts...)}
	r.Digest = recordDigest(r)
	e = s.db.Transaction(func(tx *gorm.DB) error {
		if s.revision == 0 {
			if e := tx.Exec("INSERT INTO controller_publication(id,authority,revision,policy,facts,digest) VALUES(1,?,?,?,?,?)", r.Authority, r.Revision, r.Policy, r.Facts, r.Digest).Error; e != nil {
				return e
			}
		} else {
			result := tx.Exec("UPDATE controller_publication SET revision=?,policy=?,facts=?,digest=? WHERE id=1 AND authority=? AND revision=?", r.Revision, r.Policy, r.Facts, r.Digest, s.authority, int64(s.revision))
			if result.Error != nil {
				return result.Error
			}
			if result.RowsAffected != 1 {
				return fmt.Errorf("publication revision changed externally")
			}
		}
		if s.mode == "database" {
			return insertAudit(tx, actor, reason, next, semantic)
		}
		return nil
	})
	if e != nil {
		return false, &PersistenceError{Err: e}
	}
	s.candidate = c
	s.revision = next
	s.semantic = semantic
	s.document = Document{Authority: s.authority, Revision: next, Policy: string(policy), Facts: string(facts)}
	return true, nil
}
func (s *Store) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return nil
	}
	s.closed = true
	db, e := s.db.DB()
	if e != nil {
		return e
	}
	return db.Close()
}
func compile(previous *configuration.Candidate, policy, facts []byte, authority string) (*configuration.Candidate, string, error) {
	if len(policy) > configuration.MaxBytes || len(facts) > configuration.MaxBytes {
		return nil, "", fmt.Errorf("publication input exceeds byte limit")
	}
	f, e := configuration.DecodeFacts(facts)
	if e != nil {
		return nil, "", e
	}
	if f.AuthorityID != authority {
		return nil, "", fmt.Errorf("publication authority mismatch")
	}
	b, e := configuration.Decode(policy)
	if e != nil {
		return nil, "", e
	}
	var c *configuration.Candidate
	if previous == nil {
		c, e = configuration.Compile(b, f)
	} else {
		c, _, e = configuration.Replace(previous, b, f)
	}
	if e != nil {
		return nil, "", e
	}
	data, e := json.Marshal(struct {
		Hash    string
		Input   any
		Objects any
	}{c.Hash(), c.Input(), c.Objects()})
	if e != nil {
		return nil, "", e
	}
	sum := sha256.Sum256(data)
	return c, hex.EncodeToString(sum[:]), nil
}
func recordDigest(r record) string {
	r.Digest = ""
	data, _ := json.Marshal(r)
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}
