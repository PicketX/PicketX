// SPDX-License-Identifier: Apache-2.0
package managementaccess

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"os"
	"strings"
	"testing"
)

func TestSessionResolvesFreshRolesAndRejectsRotatedCredential(t *testing.T) {
	a, _, path := fixture(t)
	digest := sha256.Sum256([]byte(token))
	fingerprint := hex.EncodeToString(digest[:])
	p, e := a.ResolveSession("principal:operator", fingerprint)
	if e != nil || len(p.Roles) != 1 || p.Roles[0] != "role:admin" {
		t.Fatal(p, e)
	}
	writeCredentials(t, path, `["role:viewer"]`)
	p, e = a.ResolveSession("principal:operator", fingerprint)
	if e != nil || p.Roles[0] != "role:viewer" {
		t.Fatal("stale role", p, e)
	}
	data, _ := os.ReadFile(path)
	rotated := sha256.Sum256([]byte(token + "-rotated"))
	if e = os.WriteFile(path, []byte(strings.ReplaceAll(string(data), fingerprint, hex.EncodeToString(rotated[:]))), 0600); e != nil {
		t.Fatal(e)
	}
	if _, e = a.ResolveSession("principal:operator", fingerprint); !errors.Is(e, ErrUnauthorized) {
		t.Fatal("ID reuse revived session", e)
	}
	if _, e = a.ResolveSession("principal:other", hex.EncodeToString(rotated[:])); !errors.Is(e, ErrUnauthorized) {
		t.Fatal(e)
	}
	if e = os.WriteFile(path, []byte(`{"principals":[]}`), 0600); e != nil {
		t.Fatal(e)
	}
	if _, e = a.ResolveSession("principal:operator", fingerprint); !errors.Is(e, ErrUnauthorized) {
		t.Fatal(e)
	}
	if e = os.WriteFile(path, []byte(`invalid`), 0600); e != nil {
		t.Fatal(e)
	}
	if _, e = a.ResolveSession("principal:operator", fingerprint); !errors.Is(e, ErrUnavailable) {
		t.Fatal(e)
	}
}
