// SPDX-License-Identifier: Apache-2.0
package managementaccess

import (
	"errors"
	"testing"
)

type oidcHumanAccess interface {
	AuthenticateHuman(string) (Principal, string, error)
	ResolveHumanSession(string, string) (Principal, error)
}

func TestVerifiedOIDCSelectsOnlyCurrentHumanAccounts(t *testing.T) {
	a, _, path := fixture(t)
	h, ok := any(a).(oidcHumanAccess)
	if !ok {
		t.Fatal("verified human resolution missing")
	}
	if _, _, e := h.AuthenticateHuman("principal:operator"); !errors.Is(e, ErrUnauthorized) {
		t.Fatal("key principal linked", e)
	}
	hash := testHumanHash(t)
	userDocument(t, path, []string{"role:admin"}, false, hash)
	p, fp, e := h.AuthenticateHuman("principal:alice")
	if e != nil || p.ID != "principal:alice" {
		t.Fatal(p, e)
	}
	userDocument(t, path, []string{"role:viewer", "role:blocked"}, false, hash)
	p, e = h.ResolveHumanSession(p.ID, fp)
	if e != nil || len(p.Roles) != 2 {
		t.Fatal("roles stale", p, e)
	}
	if allow, e := a.Allowed(p, "publication:read"); e != nil || allow {
		t.Fatal("deny lost", e)
	}
	userDocument(t, path, []string{"role:admin"}, true, hash)
	if _, e = h.ResolveHumanSession(p.ID, fp); !errors.Is(e, ErrUnauthorized) {
		t.Fatal("disabled user accepted", e)
	}
	userDocument(t, path, []string{"role:admin"}, false, dummyPasswordHash)
	if _, e = h.ResolveHumanSession(p.ID, fp); !errors.Is(e, ErrUnauthorized) {
		t.Fatal("rotated human revived", e)
	}
}
