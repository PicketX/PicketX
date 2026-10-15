// SPDX-License-Identifier: Apache-2.0
package core

import (
	"net/netip"
	"reflect"
	"testing"
)

func giteaSnapshot(ipv6 bool) Snapshot {
	s := baseSnapshot()
	s.Grants[0].ID = "https-host"
	s.Grants[0].Scopes[0].Endpoints = EndpointSelection{Mode: SelectionNames, Names: []string{"https"}}
	s.Grants[1].ID = "ssh-host"
	s.Grants[1].Source = netip.MustParsePrefix("192.0.2.10/32")
	s.Grants[1].Scopes[0].Endpoints = EndpointSelection{Mode: SelectionNames, Names: []string{"ssh"}}
	s.Grants[1].Validity.ExpiresAt = atHour("12:00")
	if ipv6 {
		for i := range s.Grants {
			s.Grants[i].Source = netip.MustParsePrefix("2001:db8::10/128")
		}
	}
	return s
}
func addHTTPSPrefix(s *Snapshot, ipv6 bool) {
	g := s.Grants[0]
	g.ID = "https-prefix"
	g.Source = netip.MustParsePrefix("192.0.2.0/24")
	if ipv6 {
		g.Source = netip.MustParsePrefix("2001:db8::/64")
	}
	g.Validity.ExpiresAt = atHour("11:00")
	s.Grants = append(s.Grants, g)
}

// Catches a common latest TTL for HTTPS+SSH, and deletion of a still-authorized overlap.
func TestGiteaIndependentDeadlines(t *testing.T) {
	for _, ipv6 := range []bool{false, true} {
		for _, overlap := range []bool{false, true} {
			s := giteaSnapshot(ipv6)
			if overlap {
				addHTTPSPrefix(&s, ipv6)
			}
			e := compileOK(t, s)
			q := query()
			if ipv6 {
				q.Source = netip.MustParseAddr("2001:db8::10")
			}
			for _, hour := range []string{"09:00", "10:00", "11:00", "12:00"} {
				q.At = atHour(hour)
				q.EndpointName = "https"
				var ids []string
				if hour == "09:00" {
					ids = []string{"https-host"}
					if overlap {
						ids = append(ids, "https-prefix")
					}
				} else if hour == "10:00" && overlap {
					ids = []string{"https-prefix"}
				}
				reason := ReasonNoActiveGrant
				if len(ids) > 0 {
					reason = ReasonAllowed
				}
				assertDecision(t, decideOK(t, e, q), len(ids) > 0, reason, ids, nil)
				q.EndpointName = "ssh"
				d := decideOK(t, e, q)
				if hour == "12:00" {
					assertDecision(t, d, false, ReasonNoActiveGrant, nil, nil)
				} else {
					assertDecision(t, d, true, ReasonAllowed, []string{"ssh-host"}, nil)
					if len(d.Contributions) != 1 || !d.Contributions[0].Validity.ExpiresAt.Equal(atHour("12:00")) {
						t.Fatal("SSH deadline changed")
					}
				}
			}
		}
	}
}
func TestGiteaOverlapAndRevocation(t *testing.T) {
	for _, remove := range []bool{false, true} {
		s := giteaSnapshot(false)
		addHTTPSPrefix(&s, false)
		if remove {
			s.Grants = s.Grants[1:]
		} else {
			s.Grants[0].Revoked = true
		}
		e := compileOK(t, s)
		assertDecision(t, decideOK(t, e, query()), true, ReasonAllowed, []string{"https-prefix"}, nil)
		deny := Deny(s.Grants[len(s.Grants)-1])
		deny.ID = "https-deny"
		deny.Revoked = false
		s.Denies = []Deny{deny}
		e = compileOK(t, s)
		assertDecision(t, decideOK(t, e, query()), false, ReasonExplicitDeny, []string{"https-prefix"}, []string{"https-deny"})
		q := query()
		q.EndpointName = "ssh"
		assertDecision(t, decideOK(t, e, q), true, ReasonAllowed, []string{"ssh-host"}, nil)
	}
}
func TestCatalogSelectionEvolution(t *testing.T) {
	for _, mode := range []SelectionMode{SelectionAll, SelectionNames, SelectionLabels} {
		s := baseSnapshot()
		s.Grants = s.Grants[:1]
		switch mode {
		case SelectionNames:
			s.Grants[0].Scopes[0].Endpoints = EndpointSelection{Mode: mode, Names: []string{"https"}}
		case SelectionLabels:
			s.Grants[0].Scopes[0].Endpoints = EndpointSelection{Mode: mode, Selector: &LabelSelector{MatchLabels: map[string]string{"kind": "web"}}}
		}
		old := compileOK(t, s)
		assertDecision(t, decideOK(t, old, query()), true, ReasonAllowed, []string{"a-host"}, nil)
		s.Resources[0].Endpoints = append(s.Resources[0].Endpoints, Endpoint{Name: "git-admin", Labels: map[string]string{"kind": "web"}})
		s.Resources[0].Endpoints[1].Labels["kind"] = "git"
		replacement := compileOK(t, s)
		q := query()
		d := decideOK(t, replacement, q)
		want := mode != SelectionLabels
		if d.Allowed != want {
			t.Fatalf("https mode %v: %+v", mode, d)
		}
		q.EndpointName = "git-admin"
		d = decideOK(t, replacement, q)
		if d.Allowed != (mode != SelectionNames) {
			t.Fatalf("new entry mode %v: %+v", mode, d)
		}
		if _, err := old.Decide(q); err == nil {
			t.Fatal("old engine gained entry")
		}
		assertDecision(t, decideOK(t, old, query()), true, ReasonAllowed, []string{"a-host"}, nil)
	}
}
func TestEquivalentProviderInputs(t *testing.T) {
	// Independently authored normalized values; no provider or decoder is exercised.
	controller := Snapshot{Resources: []Resource{{Name: "gitea", Endpoints: []Endpoint{{Name: "https"}}}}, Grants: []Grant{{ID: "grant", Source: netip.MustParsePrefix("192.0.2.10/32"), Scopes: []Scope{{ResourceName: "gitea"}}, Validity: Validity{Mode: LifetimeBounded, ExpiresAt: atHour("10:00")}}}}
	static := Snapshot{Resources: []Resource{{Name: "gitea", Endpoints: []Endpoint{{Name: "https", Type: "network"}}}}, Grants: []Grant{{ID: "grant", SubjectType: "source", Renewal: "disabled", Source: netip.MustParsePrefix("192.0.2.10/32"), Scopes: []Scope{{ResourceName: "gitea", Endpoints: EndpointSelection{Mode: SelectionAll}, Actions: []string{"connect"}}}, Validity: Validity{Mode: LifetimeBounded, ExpiresAt: instant("2026-10-15T18:00:00+08:00")}}}}
	q := query()
	q.ResourceName = "gitea"
	a := decideOK(t, compileOK(t, controller), q)
	b := decideOK(t, compileOK(t, static), q)
	assertDecision(t, a, true, ReasonAllowed, []string{"grant"}, nil)
	if !reflect.DeepEqual(a, b) || !a.NextChange.Equal(atHour("10:00")) {
		t.Fatalf("provider intent differs %+v %+v", a, b)
	}
}
func TestRecompilePreservesDeadlines(t *testing.T) {
	s := giteaSnapshot(false)
	first := compileOK(t, s)
	replayed := compileOK(t, s)
	for _, hour := range []string{"09:00", "10:00", "12:00"} {
		for _, entry := range []string{"https", "ssh"} {
			q := query()
			q.At = atHour(hour)
			q.EndpointName = entry
			a := decideOK(t, first, q)
			b := decideOK(t, replayed, q)
			allowed := (entry == "https" && hour == "09:00") || (entry == "ssh" && hour != "12:00")
			if a.Allowed != allowed || !reflect.DeepEqual(a, b) {
				t.Fatalf("replay at %s %s: %+v %+v", entry, hour, a, b)
			}
			if allowed {
				end := atHour("10:00")
				if entry == "ssh" {
					end = atHour("12:00")
				}
				if len(b.Contributions) != 1 || !b.Contributions[0].Validity.ExpiresAt.Equal(end) {
					t.Fatal("recompile extended deadline")
				}
			}
		}
	}
}
