// SPDX-License-Identifier: Apache-2.0
package publication

import (
	"bytes"
	"errors"
	"github.com/PicketX/PicketX/pkg/configuration"
	"os"
	"path/filepath"
	"testing"
)

const authority = "static:example-v1"

func inputs(t *testing.T) ([]byte, []byte) {
	t.Helper()
	p, e := os.ReadFile("../../examples/static/gitea.yaml")
	if e != nil {
		t.Fatal(e)
	}
	f, e := os.ReadFile("../../examples/static/facts.json")
	if e != nil {
		t.Fatal(e)
	}
	return p, f
}
func open(t *testing.T, p string) *Store {
	t.Helper()
	s, e := Open(p, authority)
	if e != nil {
		t.Fatal(e)
	}
	t.Cleanup(func() { _ = s.Close() })
	return s
}
func TestPersistReopenNoopAndPreflight(t *testing.T) {
	p := filepath.Join(t.TempDir(), "state", "controller.sqlite")
	s := open(t, p)
	policy, facts := inputs(t)
	if c, r := s.Current(); c != nil || r != 0 {
		t.Fatal(c, r)
	}
	denied := errors.New("denied")
	if _, e := s.Accept(policy, facts, func(*configuration.Candidate, uint64) error { return denied }); !errors.Is(e, denied) {
		t.Fatal(e)
	}
	if c, r := s.Current(); c != nil || r != 0 {
		t.Fatal("preflight mutated", c, r)
	}
	if changed, e := s.Accept(policy, facts, nil); e != nil || !changed {
		t.Fatal(changed, e)
	}
	if changed, e := s.Accept(policy, facts, nil); e != nil || changed {
		t.Fatal("noop changed", changed, e)
	}
	if _, e := s.Accept([]byte("invalid"), facts, nil); e == nil {
		t.Fatal("invalid accepted")
	}
	if _, e := s.Accept(bytes.Replace(policy, []byte("port: 443"), []byte("port: 8443"), 1), facts, nil); e == nil {
		t.Fatal("retarget accepted")
	}
	_, rev := s.Current()
	if rev != 1 {
		t.Fatal(rev)
	}
	_ = s.Close()
	reopened := open(t, p)
	c, r := reopened.Current()
	if c == nil || r != 1 {
		t.Fatal(c, r)
	}
	info, _ := os.Stat(p)
	if info.Mode().Perm() != 0600 {
		t.Fatal(info.Mode())
	}
}
func TestCASAndCorruption(t *testing.T) {
	p := filepath.Join(t.TempDir(), "private", "controller.sqlite")
	a := open(t, p)
	b := open(t, p)
	policy, facts := inputs(t)
	if _, e := a.Accept(policy, facts, nil); e != nil {
		t.Fatal(e)
	}
	if _, e := b.Accept(policy, facts, nil); e == nil {
		t.Fatal("stale writer overwrote accepted record")
	}
	_ = a.Close()
	_ = b.Close()
	if s, e := Open(p, "other"); e == nil {
		_ = s.Close()
		t.Fatal("authority mismatch accepted")
	}
	c := open(t, p)
	if e := c.db.Exec("UPDATE controller_publication SET digest = 'corrupt'").Error; e != nil {
		t.Fatal(e)
	}
	_ = c.Close()
	if s, e := Open(p, authority); e == nil {
		_ = s.Close()
		t.Fatal("corruption accepted")
	}
}
func TestMigrationAndOverflow(t *testing.T) {
	p := filepath.Join(t.TempDir(), "private", "controller.sqlite")
	s := open(t, p)
	policy, facts := inputs(t)
	_, e := s.Accept(policy, facts, nil)
	if e != nil {
		t.Fatal(e)
	}
	s.revision = uint64(1<<63 - 1)
	if _, e = s.Accept(append(policy, []byte("\n# revision\n")...), facts, nil); e != nil {
		t.Fatal("semantic noop should not consume revision", e)
	}
	data := bytes.Replace(policy, []byte("10:00:00Z"), []byte("11:00:00Z"), 1)
	if _, e = s.Accept(data, facts, nil); e == nil {
		t.Fatal("revision overflow accepted")
	}
	if e = s.db.Exec("UPDATE controller_schema SET version=2").Error; e != nil {
		t.Fatal(e)
	}
	_ = s.Close()
	if x, e := Open(p, authority); e == nil {
		_ = x.Close()
		t.Fatal("future migration accepted")
	}
}
func TestRejectUnsafeFilesAndFacts(t *testing.T) {
	p := filepath.Join(t.TempDir(), "private", "controller.sqlite")
	if e := os.MkdirAll(filepath.Dir(p), 0700); e != nil {
		t.Fatal(e)
	}
	if e := os.WriteFile(p, []byte(""), 0644); e != nil {
		t.Fatal(e)
	}
	if s, e := Open(p, authority); e == nil {
		_ = s.Close()
		t.Fatal("public DB accepted")
	}
	s := open(t, filepath.Join(t.TempDir(), "private", "safe.sqlite"))
	policy, facts := inputs(t)
	for _, f := range [][]byte{bytes.Replace(facts, []byte(`"authorityID"`), []byte(`"AuthorityID"`), 1), []byte("null"), append(facts, []byte("{}")...)} {
		if _, e := s.Accept(policy, f, nil); e == nil {
			t.Fatal("invalid facts accepted")
		}
	}
}

// A stale writer that opened an existing row must not overwrite a newer UPDATE.
func TestStaleUpdateCannotReplacePersistedRevision(t *testing.T) {
	path := filepath.Join(t.TempDir(), "private", "controller.sqlite")
	policy, facts := inputs(t)
	seed := open(t, path)
	if _, e := seed.Accept(policy, facts, nil); e != nil {
		t.Fatal(e)
	}
	_ = seed.Close()
	a := open(t, path)
	b := open(t, path)
	winner := bytes.Replace(policy, []byte("10:00:00Z"), []byte("11:00:00Z"), 1)
	stale := bytes.Replace(policy, []byte("grant-browser-v1"), []byte("grant-browser-stale"), 1)
	if changed, e := a.Accept(winner, facts, nil); e != nil || !changed {
		t.Fatal(changed, e)
	}
	accepted, revision := a.Current()
	if revision != 2 {
		t.Fatal(revision)
	}
	if changed, e := b.Accept(stale, facts, nil); e == nil || changed {
		t.Fatal("stale UPDATE replaced revision2", changed, e)
	}
	_, staleRevision := b.Current()
	if staleRevision != 1 {
		t.Fatal("failed CAS advanced stale in-memory state", staleRevision)
	}
	_ = a.Close()
	_ = b.Close()
	reopened := open(t, path)
	actual, revision := reopened.Current()
	if revision != 2 || actual.Hash() != accepted.Hash() {
		t.Fatal("persisted winner replaced", revision, actual.Hash(), accepted.Hash())
	}
}
