// SPDX-License-Identifier: Apache-2.0
package managementaccess

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/PicketX/PicketX/internal/publication"
)

const token = "management-independent-token-1234567890"

func fixture(t *testing.T) (*Access, *publication.Store, string) {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "principals.json")
	writeCredentials(t, path, `["role:admin"]`)
	s, e := publication.OpenMode(filepath.Join(dir, "state", "db.sqlite"), "authority:test", "database")
	if e != nil {
		t.Fatal(e)
	}
	t.Cleanup(func() { _ = s.Close() })
	a, e := New(s.GORM(), Settings{Credentials: path, Reload: 10 * time.Millisecond, WarnRules: 10000, WarnLoadPercent: 50}, slog.New(slog.NewTextHandler(io.Discard, nil)))
	if e != nil {
		t.Fatal(e)
	}
	return a, s, path
}
func writeCredentials(t *testing.T, path, roles string) {
	t.Helper()
	sum := sha256.Sum256([]byte(token))
	raw := fmt.Sprintf(`{"principals":[{"id":"principal:operator","sha256":%q,"roles":%s}]}`, hex.EncodeToString(sum[:]), roles)
	if e := os.WriteFile(path, []byte(raw), 0600); e != nil {
		t.Fatal(e)
	}
}
func TestTrustedRolesDenyPrecedenceAndCredentialRevocation(t *testing.T) {
	a, _, path := fixture(t)
	p, e := a.Authenticate(token)
	if e != nil || p.ID != "principal:operator" {
		t.Fatal(p, e)
	}
	for _, op := range []string{"publication:read", "publication:write", "audit:read"} {
		ok, e := a.Allowed(p, op)
		if e != nil || !ok {
			t.Fatal(op, ok, e)
		}
	}
	writeCredentials(t, path, `["role:admin","role:blocked"]`)
	p, e = a.Authenticate(token)
	if e != nil {
		t.Fatal(e)
	}
	if ok, e := a.Allowed(p, "publication:write"); e != nil || ok {
		t.Fatal("allow bypassed another role deny", ok, e)
	}
	writeCredentials(t, path, `["role:viewer"]`)
	p, _ = a.Authenticate(token)
	if ok, e := a.Allowed(p, "publication:write"); e != nil || ok {
		t.Fatal(ok, e)
	}
	if ok, e := a.Allowed(p, "publication:read"); e != nil || !ok {
		t.Fatal(ok, e)
	}
	if e := os.WriteFile(path, []byte(`{"principals":[]}`), 0600); e != nil {
		t.Fatal(e)
	}
	if _, e := a.Authenticate(token); e == nil {
		t.Fatal("removed principal accepted")
	}
}
func TestCredentialDocumentAndTokenFailClosed(t *testing.T) {
	a, _, path := fixture(t)
	if !a.Ready() {
		t.Fatal("valid access not ready")
	}
	for _, bad := range []string{"", strings.Repeat("a", 31), strings.Repeat("a", 257), token + "\n", token + " ", "\x00" + token} {
		if _, e := a.Authenticate(bad); e == nil {
			t.Fatal("bad token accepted")
		}
	}
	for _, raw := range []string{`null`, `{"Principals":[]}`, `{"principals":null}`, `{"principals":[],"principals":[]}`, `{"principals":[{"id":"x","sha256":"bad","roles":[]}]}`, `{"principals":[]} {}`} {
		if e := os.WriteFile(path, []byte(raw), 0600); e != nil {
			t.Fatal(e)
		}
		if _, e := a.Authenticate(token); e == nil {
			t.Fatal("malformed credential document accepted", raw)
		}
		if a.Ready() {
			t.Fatal("invalid credentials reported ready")
		}
	}
	writeCredentials(t, path, `["role:admin"]`)
	if e := os.Chmod(path, 0644); e != nil {
		t.Fatal(e)
	}
	if _, e := a.Authenticate(token); e == nil {
		t.Fatal("public credentials accepted")
	}
}
func TestPeriodicPolicyReloadAndFailureRecovery(t *testing.T) {
	a, s, _ := fixture(t)
	p, _ := a.Authenticate(token)
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() { a.Run(ctx); close(done) }()
	t.Cleanup(func() { cancel(); <-done })
	if e := s.GORM().Exec("INSERT INTO casbin_rule(ptype,v0,v1,v2,v3,v4,v5) VALUES('p','role:admin','publication:write','deny','','','')").Error; e != nil {
		t.Fatal(e)
	}
	until := time.Now().Add(time.Second)
	for {
		ok, e := a.Allowed(p, "publication:write")
		if e == nil && !ok {
			break
		}
		if time.Now().After(until) {
			t.Fatal("policy not periodically reloaded", ok, e)
		}
		time.Sleep(time.Millisecond)
	}
	if e := s.GORM().Exec("ALTER TABLE casbin_rule RENAME TO hidden_policy").Error; e != nil {
		t.Fatal(e)
	}
	until = time.Now().Add(time.Second)
	for {
		_, e := a.Allowed(p, "publication:read")
		if e != nil {
			break
		}
		if time.Now().After(until) {
			t.Fatal("failed reload retained available authorization")
		}
		time.Sleep(time.Millisecond)
	}
	if e := s.GORM().Exec("ALTER TABLE hidden_policy RENAME TO casbin_rule").Error; e != nil {
		t.Fatal(e)
	}
	until = time.Now().Add(time.Second)
	for {
		ok, e := a.Allowed(p, "publication:read")
		if e == nil && ok {
			break
		}
		if time.Now().After(until) {
			t.Fatal("policy reload never recovered", ok, e)
		}
		time.Sleep(time.Millisecond)
	}
}
