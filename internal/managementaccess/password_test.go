// SPDX-License-Identifier: Apache-2.0
package managementaccess

import (
	"encoding/json"
	"errors"
	"os"
	"strings"
	"sync"
	"testing"

	"golang.org/x/crypto/bcrypt"
)

const humanPassword = "a long test password!"

var humanHashOnce sync.Once
var humanHash string

func testHumanHash(t *testing.T) string {
	t.Helper()
	humanHashOnce.Do(func() {
		h, e := bcrypt.GenerateFromPassword([]byte(humanPassword), 12)
		if e != nil {
			panic(e)
		}
		humanHash = string(h)
	})
	return humanHash
}
func userDocument(t *testing.T, path string, roles []string, disabled bool, hash string) {
	t.Helper()
	raw, _ := os.ReadFile(path)
	var doc map[string]any
	if json.Unmarshal(raw, &doc) != nil {
		t.Fatal("bad existing credentials")
	}
	doc["users"] = []any{map[string]any{"id": "principal:alice", "username": "alice", "password_hash": hash, "roles": roles, "disabled": disabled}}
	data, _ := json.Marshal(doc)
	if e := os.WriteFile(path, data, 0600); e != nil {
		t.Fatal(e)
	}
}

type humanAccess interface {
	AuthenticatePassword(string, string) (Principal, string, error)
	PasswordLoginEnabled() (bool, error)
}

func humanAPI(t *testing.T, a *Access) humanAccess {
	t.Helper()
	h, ok := any(a).(humanAccess)
	if !ok {
		t.Fatal("local human login capability missing")
	}
	return h
}
func TestHumanLoginFreshRolesRotationDisableAndIsolation(t *testing.T) {
	a, _, path := fixture(t)
	h := humanAPI(t, a)
	if enabled, e := h.PasswordLoginEnabled(); e != nil || enabled {
		t.Fatal(enabled, e)
	}
	hash := testHumanHash(t)
	userDocument(t, path, []string{"role:admin"}, false, hash)
	if enabled, e := h.PasswordLoginEnabled(); e != nil || !enabled {
		t.Fatal(enabled, e)
	}
	p, fp, e := h.AuthenticatePassword("alice", humanPassword)
	if e != nil || p.ID != "principal:alice" || len(fp) != 64 {
		t.Fatal(p, fp, e)
	}
	if _, e = a.Authenticate(humanPassword); !errors.Is(e, ErrUnauthorized) {
		t.Fatal("password became API token", e)
	}
	if _, e = a.ResolveSession("principal:operator", fp); !errors.Is(e, ErrUnauthorized) {
		t.Fatal("cross-identity session", e)
	}
	userDocument(t, path, []string{"role:viewer"}, false, hash)
	p, e = a.ResolveSession(p.ID, fp)
	if e != nil || p.Roles[0] != "role:viewer" {
		t.Fatal("stale roles", p, e)
	}
	if ok, e := a.Allowed(p, "publication:write"); e != nil || ok {
		t.Fatal("viewer write", ok, e)
	}
	userDocument(t, path, []string{"role:admin", "role:blocked"}, false, hash)
	p, e = a.ResolveSession(p.ID, fp)
	if e != nil {
		t.Fatal(e)
	}
	if ok, e := a.Allowed(p, "publication:write"); e != nil || ok {
		t.Fatal("deny bypass", ok, e)
	}
	userDocument(t, path, []string{"role:admin"}, true, hash)
	if _, e = a.ResolveSession(p.ID, fp); !errors.Is(e, ErrUnauthorized) {
		t.Fatal("disabled session", e)
	}
	if _, _, e = h.AuthenticatePassword("alice", humanPassword); !errors.Is(e, ErrUnauthorized) {
		t.Fatal("disabled login", e)
	}
	userDocument(t, path, []string{"role:admin"}, false, hash)
	data, _ := os.ReadFile(path)
	rotated, _ := bcrypt.GenerateFromPassword([]byte("a different password!"), 12)
	userDocument(t, path, []string{"role:admin"}, false, string(rotated))
	if _, e = a.ResolveSession(p.ID, fp); !errors.Is(e, ErrUnauthorized) {
		t.Fatal("rotated session", e)
	}
	if _, _, e = h.AuthenticatePassword("alice", humanPassword); !errors.Is(e, ErrUnauthorized) {
		t.Fatal("old password", e)
	}
	if e = os.WriteFile(path, []byte(strings.Replace(string(data), `"username":"alice"`, `"username":"bob"`, 1)), 0600); e != nil {
		t.Fatal(e)
	}
	if _, e = a.ResolveSession(p.ID, fp); !errors.Is(e, ErrUnauthorized) {
		t.Fatal("renamed session", e)
	}
	writeCredentials(t, path, `["role:admin"]`)
	if _, e = a.ResolveSession(p.ID, fp); !errors.Is(e, ErrUnauthorized) {
		t.Fatal("removed session", e)
	}
	if _, e = a.Authenticate(token); e != nil {
		t.Fatal("key regression", e)
	}
}
func TestHumanCredentialDocumentFailsClosed(t *testing.T) {
	a, _, path := fixture(t)
	h := humanAPI(t, a)
	hash := testHumanHash(t)
	userDocument(t, path, []string{"role:admin"}, false, hash)
	valid, _ := os.ReadFile(path)
	for name, doc := range map[string]string{
		"null":               strings.Replace(string(valid), `"users":[`, `"users":null,"ignored":[`, 1),
		"ID collision":       strings.Replace(string(valid), "principal:alice", "principal:operator", 1),
		"username uppercase": strings.Replace(string(valid), `"username":"alice"`, `"username":"Alice"`, 1),
		"hash cost":          strings.Replace(string(valid), "$12$", "$31$", 1),
		"hash alphabet":      strings.Replace(string(valid), hash, strings.Repeat("!", 60), 1),
		"role":               strings.Replace(string(valid), `"roles":["role:admin"]`, `"roles":["admin"]`, 1),
		"unknown":            strings.Replace(string(valid), `"disabled":false`, `"disabled":false,"password":"secret"`, 1),
	} {
		t.Run(name, func(t *testing.T) {
			if e := os.WriteFile(path, []byte(doc), 0600); e != nil {
				t.Fatal(e)
			}
			if _, _, e := h.AuthenticatePassword("alice", humanPassword); !errors.Is(e, ErrUnavailable) {
				t.Fatal(e)
			}
			if _, e := a.Authenticate(token); !errors.Is(e, ErrUnavailable) {
				t.Fatal("partial authority accepted", e)
			}
		})
	}
	if e := os.WriteFile(path, valid, 0600); e != nil {
		t.Fatal(e)
	}
	for _, pair := range [][2]string{{"nobody", humanPassword}, {"alice", "not the right password"}, {"Alice", humanPassword}, {"alice", strings.Repeat("界", 25)}} {
		if _, _, e := h.AuthenticatePassword(pair[0], pair[1]); !errors.Is(e, ErrUnauthorized) {
			t.Fatal(pair[0], e)
		}
	}
}
