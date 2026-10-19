// SPDX-License-Identifier: Apache-2.0
package managementaccess

import "testing"

func TestRequestPolicyMigrationPreservesCustomDenyAndOwnIsolation(t *testing.T) {
	a, s, path := fixture(t)
	// Reconstruct version1 while retaining an operator's custom deny.
	if err := s.GORM().Exec("DELETE FROM casbin_rule WHERE v1 LIKE 'request:%'").Error; err != nil {
		t.Fatal(err)
	}
	if err := s.GORM().Exec("INSERT INTO casbin_rule(ptype,v0,v1,v2,v3,v4,v5) VALUES('p','role:admin','request:review','deny','','','')").Error; err != nil {
		t.Fatal(err)
	}
	if err := s.GORM().Exec("UPDATE management_schema SET version=1 WHERE id=1").Error; err != nil {
		t.Fatal(err)
	}
	if err := bootstrap(s.GORM()); err != nil {
		t.Fatal(err)
	}
	if err := bootstrap(s.GORM()); err != nil {
		t.Fatal(err)
	}
	if err := a.reload(false); err != nil {
		t.Fatal(err)
	}
	p, _ := a.Authenticate(token)
	if ok, err := a.Allowed(p, "request:review"); err != nil || ok {
		t.Fatal("custom deny overwritten", ok, err)
	}
	writeCredentials(t, path, `["role:requester"]`)
	p, _ = a.Authenticate(token)
	for _, op := range []string{"request:create", "request:read-own"} {
		if ok, err := a.Allowed(p, op); err != nil || !ok {
			t.Fatal(op, ok, err)
		}
	}
	for _, op := range []string{"request:review", "publication:read", "publication:write", "audit:read"} {
		if ok, err := a.Allowed(p, op); err != nil || ok {
			t.Fatal("request role escaped", op, ok, err)
		}
	}
	writeCredentials(t, path, `["role:requester","role:admin","role:blocked"]`)
	p, _ = a.Authenticate(token)
	for _, op := range []string{"request:create", "request:read-own", "request:review"} {
		if ok, err := a.Allowed(p, op); err != nil || ok {
			t.Fatal("request deny bypass", op, ok, err)
		}
	}
}
