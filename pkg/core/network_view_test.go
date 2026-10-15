// SPDX-License-Identifier: Apache-2.0
package core

import (
	"net/netip"
	"reflect"
	"slices"
	"sync"
	"testing"
)

func viewSnapshot() Snapshot {
	g := Grant{ID: "shared", Source: netip.MustParsePrefix("198.51.100.9/32"), Validity: Validity{Mode: LifetimeBounded, ExpiresAt: atHour("10:00")}, Scopes: []Scope{{ResourceName: "gitea"}, {ResourceName: "gitea"}}}
	return Snapshot{Resources: []Resource{
		{Name: "gitea", Labels: map[string]string{"kind": "service"}, Endpoints: []Endpoint{{Name: "ssh", Labels: map[string]string{"port": "ssh"}}, {Name: "https", Labels: map[string]string{"port": "web"}}}},
		{Name: "empty", Labels: map[string]string{"empty": "yes"}},
	}, Grants: []Grant{g}, Denies: []Deny{Deny(g)}}
}
func viewOK(t *testing.T, e *Engine) NetworkView {
	t.Helper()
	v, err := e.NetworkView()
	if err != nil {
		t.Fatal(err)
	}
	return v
}
func TestNetworkViewResolvedMembership(t *testing.T) {
	v := viewOK(t, compileOK(t, viewSnapshot()))
	if len(v.Resources) != 2 || v.Resources[0].Name != "empty" || len(v.Resources[0].Entries) != 0 {
		t.Fatal(v)
	}
	r := v.Resources[1]
	if r.Name != "gitea" || r.Labels["kind"] != "service" || len(r.Entries) != 2 {
		t.Fatal(r)
	}
	if r.Entries[0].Endpoint.Name != "https" || r.Entries[1].Endpoint.Name != "ssh" {
		t.Fatal(r)
	}
	for _, e := range r.Entries {
		if e.Endpoint.Type != "network" || len(e.Grants) != 1 || len(e.Denies) != 1 {
			t.Fatal(e)
		}
		want := ResolvedContribution{ID: "shared", Source: netip.MustParsePrefix("198.51.100.9/32"), Validity: Validity{Mode: LifetimeBounded, ExpiresAt: atHour("10:00")}}
		if e.Grants[0] != want || e.Denies[0] != want {
			t.Fatal(e)
		}
	}
}
func TestNetworkViewIncludesInactive(t *testing.T) {
	s := viewSnapshot()
	s.Denies = nil
	g := s.Grants[0]
	s.Grants = nil
	for _, tc := range []struct {
		id      string
		v       Validity
		revoked bool
	}{
		{"future", Validity{Mode: LifetimeBounded, NotBefore: atHour("11:00"), ExpiresAt: atHour("12:00")}, false},
		{"expired", Validity{Mode: LifetimeBounded, ExpiresAt: atHour("07:00")}, false},
		{"revoked", Validity{Mode: LifetimePermanent}, true},
	} {
		c := g
		c.ID = tc.id
		c.Validity = tc.v
		c.Revoked = tc.revoked
		s.Grants = append(s.Grants, c)
	}
	v := viewOK(t, compileOK(t, s))
	gots := v.Resources[1].Entries[0].Grants
	if len(gots) != 3 || gots[0].ID != "expired" || gots[1].ID != "future" || gots[2].ID != "revoked" || !gots[2].Revoked {
		t.Fatal(gots)
	}
}
func TestNetworkViewOwnsData(t *testing.T) {
	s := viewSnapshot()
	e := compileOK(t, s)
	before := viewOK(t, e)
	q := Query{Source: netip.MustParseAddr("198.51.100.9"), ResourceName: "gitea", EndpointName: "https", At: atHour("09:00")}
	d := decideOK(t, e, q)
	s.Resources[0].Labels["kind"] = "changed"
	s.Resources[0].Endpoints[1].Labels["port"] = "bad"
	s.Grants[0].ID = "changed"
	s.Denies[0].Revoked = true
	v := viewOK(t, e)
	v.Resources[1].Labels["kind"] = "bad"
	v.Resources[1].Entries[0].Endpoint.Labels["port"] = "bad"
	v.Resources[1].Entries[0].Grants[0].Revoked = true
	v.Resources[1].Entries[0].Denies[0].ID = "bad"
	v.Resources[1].Entries[0].Endpoint.Name = "bad"
	if after := viewOK(t, e); !reflect.DeepEqual(before, after) {
		t.Fatal(after)
	}
	if after := decideOK(t, e, q); !reflect.DeepEqual(d, after) {
		t.Fatal(after)
	}
}
func TestNetworkViewDeterministicAndConcurrent(t *testing.T) {
	s := viewSnapshot()
	e := compileOK(t, s)
	want := viewOK(t, e)
	slices.Reverse(s.Resources)
	slices.Reverse(s.Resources[1].Endpoints)
	slices.Reverse(s.Grants[0].Scopes)
	if got := viewOK(t, compileOK(t, s)); !reflect.DeepEqual(got, want) {
		t.Fatal(got)
	}
	var wg sync.WaitGroup
	for range 16 {
		wg.Go(func() {
			v, err := e.NetworkView()
			if err != nil {
				t.Error(err)
				return
			}
			if !reflect.DeepEqual(v, want) {
				t.Error(v)
			}
			v.Resources[1].Labels["kind"] = "private"
			_, err = e.Decide(Query{Source: netip.MustParseAddr("198.51.100.9"), ResourceName: "gitea", EndpointName: "https", At: atHour("09:00")})
			if err != nil {
				t.Error(err)
			}
		})
	}
	wg.Wait()
}
func TestNetworkViewEmptyAndInvalidEngine(t *testing.T) {
	v := viewOK(t, compileOK(t, Snapshot{}))
	if len(v.Resources) != 0 {
		t.Fatal(v)
	}
	for _, e := range []*Engine{nil, {}} {
		if _, err := e.NetworkView(); err == nil {
			t.Fatal("uncompiled engine accepted")
		}
	}
}
