// SPDX-License-Identifier: Apache-2.0
package publication

import (
	"bytes"
	"errors"
	"path/filepath"
	"sync"
	"testing"
)

func managed(t *testing.T, path string) *Store {
	t.Helper()
	s, e := OpenMode(path, authority, "database")
	if e != nil {
		t.Fatal(e)
	}
	t.Cleanup(func() { _ = s.Close() })
	return s
}

func TestManagedCASRawSnapshotAuditAndReopen(t *testing.T) {
	path := filepath.Join(t.TempDir(), "state", "db.sqlite")
	s := managed(t, path)
	policy, facts := inputs(t)
	if _, e := s.Accept(policy, facts, nil); e == nil {
		t.Fatal("file accept bypassed database authority")
	}
	if changed, e := s.AcceptManaged(0, "principal:one", "initial", policy, facts, nil); e != nil || !changed {
		t.Fatal(changed, e)
	}
	if _, e := s.AcceptManaged(0, "principal:one", "stale noop", policy, facts, nil); !errors.Is(e, ErrPrecondition) {
		t.Fatal("stale noop accepted", e)
	}
	if changed, e := s.AcceptManaged(1, "principal:one", "noop", append(policy, []byte("\n# comment\n")...), facts, nil); e != nil || changed {
		t.Fatal(changed, e)
	}
	doc, e := s.Snapshot()
	if e != nil || doc.Revision != 1 || doc.Policy != string(policy) || doc.Facts != string(facts) {
		t.Fatal(doc, e)
	}
	events, e := s.Audit(0, 1)
	if e != nil || len(events) != 1 || events[0].Actor != "principal:one" || events[0].Reason != "initial" || events[0].Revision != 1 || events[0].Digest == "" || events[0].At.IsZero() {
		t.Fatal(events, e)
	}
	if next, e := s.Audit(events[0].ID, 1); e != nil || len(next) != 0 {
		t.Fatal(next, e)
	}
	_ = s.Close()
	s = managed(t, path)
	if doc, e = s.Snapshot(); e != nil || doc.Policy != string(policy) || doc.Revision != 1 {
		t.Fatal(doc, e)
	}
	if x, e := Open(path, authority); e == nil {
		_ = x.Close()
		t.Fatal("file mode reopened managed DB")
	}
}

func TestManagedConcurrentCASAndAuditRollback(t *testing.T) {
	s := managed(t, filepath.Join(t.TempDir(), "state", "db.sqlite"))
	p, f := inputs(t)
	if _, e := s.AcceptManaged(0, "principal:seed", "seed", p, f, nil); e != nil {
		t.Fatal(e)
	}
	var wg sync.WaitGroup
	results := make(chan error, 2)
	for _, hour := range []string{"11:00:00Z", "12:00:00Z"} {
		wg.Add(1)
		go func(hour string) {
			defer wg.Done()
			_, e := s.AcceptManaged(1, "principal:writer", "update", bytes.Replace(p, []byte("10:00:00Z"), []byte(hour), 1), f, nil)
			results <- e
		}(hour)
	}
	wg.Wait()
	close(results)
	success, stale := 0, 0
	for e := range results {
		if e == nil {
			success++
		} else if errors.Is(e, ErrPrecondition) {
			stale++
		} else {
			t.Fatal(e)
		}
	}
	if success != 1 || stale != 1 {
		t.Fatal(success, stale)
	}
	events, e := s.Audit(0, 100)
	if e != nil || len(events) != 2 {
		t.Fatal(events, e)
	}
	before, _ := s.Snapshot()
	if e = s.db.Exec("CREATE TRIGGER fail_audit BEFORE INSERT ON controller_audit BEGIN SELECT RAISE(ABORT,'audit failed'); END").Error; e != nil {
		t.Fatal(e)
	}
	changed, e := s.AcceptManaged(2, "principal:writer", "rollback", bytes.Replace(p, []byte("10:00:00Z"), []byte("13:00:00Z"), 1), f, nil)
	var persistence *PersistenceError
	if changed || !errors.As(e, &persistence) {
		t.Fatal(changed, e)
	}
	after, _ := s.Snapshot()
	if after != before {
		t.Fatal("audit failure changed memory", after)
	}
	var revision int64
	if e = s.db.Raw("SELECT revision FROM controller_publication WHERE id=1").Scan(&revision).Error; e != nil || revision != 2 {
		t.Fatal("audit failure changed DB", revision, e)
	}
}

func TestModeBindingVersionOneMigrationAndValidation(t *testing.T) {
	path := filepath.Join(t.TempDir(), "state", "db.sqlite")
	s := open(t, path)
	p, f := inputs(t)
	if _, e := s.Accept(p, f, nil); e != nil {
		t.Fatal(e)
	}
	// Simulate the deployed version1 schema, before mode/audit were introduced.
	for _, table := range []string{"controller_request_event", "controller_access_request"} {
		if e := s.db.Exec("DROP TABLE " + table).Error; e != nil {
			t.Fatal(e)
		}
	}
	if e := s.db.Exec("DROP TABLE controller_http_audit").Error; e != nil {
		t.Fatal(e)
	}
	if e := s.db.Exec("DROP TABLE controller_source").Error; e != nil {
		t.Fatal(e)
	}
	if e := s.db.Exec("DROP TABLE controller_audit").Error; e != nil {
		t.Fatal(e)
	}
	if e := s.db.Exec("UPDATE controller_schema SET version=1").Error; e != nil {
		t.Fatal(e)
	}
	_ = s.Close()
	if x, e := OpenMode(path, authority, "database"); e == nil {
		_ = x.Close()
		t.Fatal("managed mode silently adopted file publication")
	}
	s = open(t, path)
	if _, e := s.AcceptManaged(1, "principal:one", "bad", p, f, nil); e == nil {
		t.Fatal("managed write in file mode")
	}
	if _, e := s.Audit(-1, 1); e == nil {
		t.Fatal("negative cursor")
	}
	if _, e := s.Audit(0, 101); e == nil {
		t.Fatal("unbounded audit")
	}
	if x, e := OpenMode(filepath.Join(t.TempDir(), "state", "db.sqlite"), authority, "unknown"); e == nil {
		_ = x.Close()
		t.Fatal("unknown mode")
	}
}
