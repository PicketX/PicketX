// SPDX-License-Identifier: Apache-2.0
package core

import (
	"net/netip"
	"reflect"
	"slices"
	"sync"
	"testing"
	"time"
)

func baseSnapshot() Snapshot {
	return Snapshot{Resources: []Resource{{Name: "dev-gitea", Labels: map[string]string{"kind": "resource"}, Endpoints: entries()}}, Grants: []Grant{
		{ID: "a-host", Source: netip.MustParsePrefix("192.0.2.10/32"), Scopes: []Scope{{ResourceName: "dev-gitea"}}, Validity: Validity{Mode: LifetimeBounded, ExpiresAt: atHour("10:00")}},
		{ID: "b-prefix", Source: netip.MustParsePrefix("192.0.2.0/24"), Scopes: []Scope{{ResourceName: "dev-gitea"}}, Validity: Validity{Mode: LifetimeBounded, ExpiresAt: atHour("11:00")}},
	}}
}
func query() Query {
	return Query{Source: netip.MustParseAddr("192.0.2.10"), ResourceName: "dev-gitea", EndpointName: "https", At: atHour("09:00")}
}
func compileOK(t *testing.T, s Snapshot) *Engine {
	t.Helper()
	e, err := Compile(s)
	if err != nil || e == nil {
		t.Fatalf("compile: %v %v", e, err)
	}
	return e
}
func decideOK(t *testing.T, e *Engine, q Query) Decision {
	t.Helper()
	d, err := e.Decide(q)
	if err != nil {
		t.Fatal(err)
	}
	return d
}
func assertDecision(t *testing.T, d Decision, allow bool, reason DecisionReason, grants, denies []string) {
	t.Helper()
	if d.Allowed != allow || d.Reason != reason || !reflect.DeepEqual(d.GrantIDs, grants) || !reflect.DeepEqual(d.DenyIDs, denies) {
		t.Fatalf("got %+v; want allow=%v reason=%s grants=%v denies=%v", d, allow, reason, grants, denies)
	}
}
func TestCompileRejectsInvalidSnapshot(t *testing.T) {
	mutations := map[string]func(*Snapshot){
		"duplicate resource": func(s *Snapshot) { s.Resources = append(s.Resources, s.Resources[0]) }, "empty resource": func(s *Snapshot) { s.Resources[0].Name = "" },
		"duplicate endpoint": func(s *Snapshot) {
			s.Resources[0].Endpoints = append(s.Resources[0].Endpoints, s.Resources[0].Endpoints[0])
		}, "empty endpoint": func(s *Snapshot) { s.Resources[0].Endpoints[0].Name = "" },
		"duplicate grant": func(s *Snapshot) { s.Grants = append(s.Grants, s.Grants[0]) }, "duplicate deny": func(s *Snapshot) { s.Denies = []Deny{Deny(s.Grants[0]), Deny(s.Grants[0])} }, "empty id": func(s *Snapshot) { s.Grants[0].ID = "" },
		"missing resource": func(s *Snapshot) { s.Grants[0].Scopes[0].ResourceName = "missing" }, "missing endpoint": func(s *Snapshot) {
			s.Grants[0].Scopes[0].Endpoints = EndpointSelection{Mode: SelectionNames, Names: []string{"missing"}}
		},
		"empty scopes": func(s *Snapshot) { s.Grants[0].Scopes = nil }, "empty actions": func(s *Snapshot) { s.Grants[0].Scopes[0].Actions = []string{} }, "empty action": func(s *Snapshot) { s.Grants[0].Scopes[0].Actions = []string{""} },
		"unmasked": func(s *Snapshot) { s.Grants[0].Source = netip.MustParsePrefix("192.0.2.10/24") }, "missing source": func(s *Snapshot) { s.Grants[0].Source = netip.Prefix{} }, "invalid validity": func(s *Snapshot) { s.Grants[0].Validity = Validity{} },
	}
	for name, mutate := range mutations {
		t.Run(name, func(t *testing.T) {
			s := baseSnapshot()
			mutate(&s)
			e, err := Compile(s)
			if err == nil || e != nil {
				t.Fatalf("invalid candidate yielded %v %v", e, err)
			}
		})
	}
	s := baseSnapshot()
	s.Denies = []Deny{Deny(s.Grants[0])}
	d := decideOK(t, compileOK(t, s), query())
	assertDecision(t, d, false, ReasonExplicitDeny, []string{"a-host", "b-prefix"}, []string{"a-host"})
}
func TestCompileValidatesInactiveContributions(t *testing.T) {
	for _, state := range []string{"revoked", "expired", "unmatched"} {
		for _, deny := range []bool{false, true} {
			t.Run(state+map[bool]string{false: " grant", true: " deny"}[deny], func(t *testing.T) {
				s := baseSnapshot()
				g := s.Grants[0]
				switch state {
				case "revoked":
					g.Revoked = true
				case "expired":
					g.Validity.ExpiresAt = atHour("07:00")
				case "unmatched":
					g.Scopes[0].Endpoints = EndpointSelection{Mode: SelectionLabels, Selector: &LabelSelector{MatchLabels: map[string]string{"missing": "x"}}}
				}
				g.Scopes[0].Actions = []string{"read"}
				if deny {
					s.Grants = nil
					s.Denies = []Deny{Deny(g)}
				} else {
					s.Grants = []Grant{g}
				}
				if e, err := Compile(s); err == nil || e != nil {
					t.Fatal("inactive malformed requirement ignored")
				}
				g.Scopes[0].Actions = nil
				g.Scopes[0].ResourceName = "missing"
				if deny {
					s.Denies = []Deny{Deny(g)}
				} else {
					s.Grants = []Grant{g}
				}
				if e, err := Compile(s); err == nil || e != nil {
					t.Fatal("inactive malformed reference ignored")
				}
			})
		}
	}
}
func TestCompileRejectsUnsupportedRequirements(t *testing.T) {
	for name, mutate := range map[string]func(*Snapshot){
		"user": func(s *Snapshot) { s.Grants[0].SubjectType = "user" }, "service": func(s *Snapshot) { s.Grants[0].SubjectType = "service" },
		"http": func(s *Snapshot) { s.Resources[0].Endpoints[0].Type = "http" }, "read": func(s *Snapshot) { s.Grants[0].Scopes[0].Actions = []string{"read"} },
		"condition": func(s *Snapshot) { s.Grants[0].Conditions = []string{"verified-user"} }, "renewal": func(s *Snapshot) { s.Grants[0].Renewal = "unique-only" },
	} {
		for _, deny := range []bool{false, true} {
			t.Run(name+map[bool]string{false: " grant", true: " deny"}[deny], func(t *testing.T) {
				s := baseSnapshot()
				mutate(&s)
				if deny {
					s.Denies = []Deny{Deny(s.Grants[0])}
					s.Grants = nil
				}
				if e, err := Compile(s); err == nil || e != nil {
					t.Fatalf("unsupported accepted %v %v", e, err)
				}
			})
		}
	}
}
func TestCompileDefaults(t *testing.T) {
	s := baseSnapshot()
	first := decideOK(t, compileOK(t, s), query())
	assertDecision(t, first, true, ReasonAllowed, []string{"a-host", "b-prefix"}, nil)
	for i := range s.Resources[0].Endpoints {
		s.Resources[0].Endpoints[i].Type = "network"
	}
	for i := range s.Grants {
		s.Grants[i].SubjectType = "source"
		s.Grants[i].Renewal = "disabled"
		s.Grants[i].Scopes[0].Actions = []string{"connect", "connect"}
		s.Grants[i].Scopes[0].Endpoints = EndpointSelection{Mode: SelectionAll}
	}
	q := query()
	q.Action = "connect"
	if d := decideOK(t, compileOK(t, s), q); !reflect.DeepEqual(first, d) {
		t.Fatalf("defaults differ: %+v %+v", first, d)
	}
	empty := compileOK(t, Snapshot{})
	if _, err := empty.Decide(q); err == nil {
		t.Fatal("empty catalog allowed query")
	}
	compileOK(t, Snapshot{Resources: []Resource{{Name: "empty"}}})
	for i := range s.Grants {
		s.Grants[i].Scopes[0].Endpoints = EndpointSelection{Mode: SelectionLabels, Selector: &LabelSelector{MatchLabels: map[string]string{"kind": "resource"}}}
	}
	d := decideOK(t, compileOK(t, s), q)
	assertDecision(t, d, false, ReasonNoActiveGrant, nil, nil)
	if d.Contributions != nil || !d.NextChange.IsZero() {
		t.Fatal("no-match invents contribution/boundary or inherits Resource labels")
	}
}
func TestDecisionPermissionUnion(t *testing.T) {
	for _, ipv6 := range []bool{false, true} {
		t.Run(map[bool]string{false: "ipv4", true: "ipv6"}[ipv6], func(t *testing.T) {
			s := baseSnapshot()
			q := query()
			if ipv6 {
				s.Grants[0].Source = netip.MustParsePrefix("2001:db8::10/128")
				s.Grants[1].Source = netip.MustParsePrefix("2001:db8::/64")
				q.Source = netip.MustParseAddr("2001:db8::10")
			}
			e := compileOK(t, s)
			d := decideOK(t, e, q)
			assertDecision(t, d, true, ReasonAllowed, []string{"a-host", "b-prefix"}, nil)
			expected := []Contribution{{GrantID: "a-host", Source: netip.MustParsePrefix("192.0.2.10/32"), Validity: Validity{Mode: LifetimeBounded, ExpiresAt: atHour("10:00")}}, {GrantID: "b-prefix", Source: netip.MustParsePrefix("192.0.2.0/24"), Validity: Validity{Mode: LifetimeBounded, ExpiresAt: atHour("11:00")}}}
			if ipv6 {
				expected[0].Source = netip.MustParsePrefix("2001:db8::10/128")
				expected[1].Source = netip.MustParsePrefix("2001:db8::/64")
			}
			if !reflect.DeepEqual(d.Contributions, expected) || !d.NextChange.Equal(atHour("10:00")) {
				t.Fatalf("lost independent provenance %+v", d)
			}
			if ipv6 {
				q.Source = netip.MustParseAddr("2001:db8::11")
			} else {
				q.Source = netip.MustParseAddr("192.0.2.11")
			}
			assertDecision(t, decideOK(t, e, q), true, ReasonAllowed, []string{"b-prefix"}, nil)
			if ipv6 {
				q.Source = netip.MustParseAddr("192.0.2.10")
			} else {
				q.Source = netip.MustParseAddr("2001:db8::10")
			}
			assertDecision(t, decideOK(t, e, q), false, ReasonNoActiveGrant, nil, nil)
		})
	}
}
func TestDecisionDenyPrecedence(t *testing.T) {
	for _, state := range []string{"active", "future", "expired", "revoked"} {
		t.Run(state, func(t *testing.T) {
			s := baseSnapshot()
			deny := Deny(s.Grants[0])
			deny.ID = "restriction"
			switch state {
			case "future":
				deny.Validity.NotBefore = atHour("09:30")
			case "expired":
				deny.Validity.ExpiresAt = atHour("09:00")
			case "revoked":
				deny.Revoked = true
			}
			s.Denies = []Deny{deny}
			d := decideOK(t, compileOK(t, s), query())
			if state == "active" {
				assertDecision(t, d, false, ReasonExplicitDeny, []string{"a-host", "b-prefix"}, []string{"restriction"})
			} else {
				assertDecision(t, d, true, ReasonAllowed, []string{"a-host", "b-prefix"}, nil)
			}
		})
	}
	s := baseSnapshot()
	s.Grants[0].Scopes[0].Endpoints = EndpointSelection{Mode: SelectionLabels, Selector: &LabelSelector{MatchExpressions: []Requirement{{Key: "kind", Operator: SelectorNotIn, Values: []string{"web"}}}}}
	assertDecision(t, decideOK(t, compileOK(t, s), query()), true, ReasonAllowed, []string{"b-prefix"}, nil)
}
func TestDecisionRevocation(t *testing.T) {
	s := baseSnapshot()
	s.Grants[0].Revoked = true
	s.Grants[0].Validity = Validity{Mode: LifetimePermanent}
	assertDecision(t, decideOK(t, compileOK(t, s), query()), true, ReasonAllowed, []string{"b-prefix"}, nil)
	s.Grants[1].Revoked = true
	d := decideOK(t, compileOK(t, s), query())
	assertDecision(t, d, false, ReasonNoActiveGrant, nil, nil)
	if !d.NextChange.IsZero() {
		t.Fatal("revoked future boundary")
	}
	s.Grants = nil
	assertDecision(t, decideOK(t, compileOK(t, s), query()), false, ReasonNoActiveGrant, nil, nil)
}
func TestDecisionNextChange(t *testing.T) {
	s := baseSnapshot()
	s.Grants = s.Grants[:1]
	s.Grants[0].Validity.NotBefore = atHour("08:30")
	deny := Deny(s.Grants[0])
	deny.ID = "deny"
	deny.Validity = Validity{Mode: LifetimeBounded, NotBefore: atHour("09:00"), ExpiresAt: atHour("09:30")}
	s.Denies = []Deny{deny}
	// Unrelated earlier boundaries must not alter this entry/source's result.
	other := s.Grants[0]
	other.ID = "unrelated"
	other.Source = netip.MustParsePrefix("198.51.100.0/24")
	other.Validity.NotBefore = atHour("08:01")
	s.Grants = append(s.Grants, other)
	e := compileOK(t, s)
	for _, tc := range []struct {
		at, next string
		allowed  bool
		reason   DecisionReason
	}{{"08:00", "08:30", false, ReasonNoActiveGrant}, {"08:30", "09:00", true, ReasonAllowed}, {"09:00", "09:30", false, ReasonExplicitDeny}, {"09:30", "10:00", true, ReasonAllowed}, {"10:00", "", false, ReasonNoActiveGrant}} {
		q := query()
		q.At = atHour(tc.at)
		d := decideOK(t, e, q)
		want := time.Time{}
		if tc.next != "" {
			want = atHour(tc.next)
		}
		if d.Allowed != tc.allowed || d.Reason != tc.reason || !d.NextChange.Equal(want) {
			t.Fatalf("at %s: %+v want next %v", tc.at, d, want)
		}
	}
}
func TestDecisionRejectsInvalidQuery(t *testing.T) {
	e := compileOK(t, baseSnapshot())
	for name, mutate := range map[string]func(*Query){"zero source": func(q *Query) { q.Source = netip.Addr{} }, "mapped": func(q *Query) { q.Source = netip.MustParseAddr("::ffff:192.0.2.10") }, "zoned": func(q *Query) { q.Source = netip.MustParseAddr("fe80::1%eth0") }, "resource": func(q *Query) { q.ResourceName = "missing" }, "endpoint": func(q *Query) { q.EndpointName = "missing" }, "action": func(q *Query) { q.Action = "read" }, "clock": func(q *Query) { q.At = time.Time{} }} {
		t.Run(name, func(t *testing.T) {
			q := query()
			mutate(&q)
			d, err := e.Decide(q)
			if err == nil || d.Allowed {
				t.Fatalf("invalid query %+v %v", d, err)
			}
		})
	}
	for _, bad := range []*Engine{nil, {}} {
		if d, err := bad.Decide(query()); err == nil || d.Allowed {
			t.Fatal("nil/zero engine")
		}
	}
}
func TestCompileOwnsInputs(t *testing.T) {
	s := baseSnapshot()
	s.Grants[0].Scopes[0].Endpoints = EndpointSelection{Mode: SelectionNames, Names: []string{"https"}}
	s.Grants[1].Scopes[0].Endpoints = EndpointSelection{Mode: SelectionLabels, Selector: &LabelSelector{MatchLabels: map[string]string{"kind": "web"}, MatchExpressions: []Requirement{{Key: "kind", Operator: SelectorIn, Values: []string{"web"}}}}}
	s.Denies = []Deny{Deny(s.Grants[0])}
	s.Denies[0].ID = "deny"
	s.Denies[0].Scopes = []Scope{{ResourceName: "dev-gitea", Endpoints: EndpointSelection{Mode: SelectionNames, Names: []string{"https"}}}}
	e := compileOK(t, s)
	before := decideOK(t, e, query())
	assertDecision(t, before, false, ReasonExplicitDeny, []string{"a-host", "b-prefix"}, []string{"deny"})
	s.Resources[0].Name = "changed"
	s.Resources[0].Labels["kind"] = "changed"
	s.Resources[0].Endpoints[1].Name = "changed"
	s.Resources[0].Endpoints[1].Labels["kind"] = "changed"
	s.Grants[0].Scopes[0].Endpoints.Names[0] = "ssh"
	s.Grants[0].Scopes[0].Actions = []string{"read"}
	s.Grants[0].Scopes[0].ResourceName = "changed"
	s.Grants[0].ID = "changed"
	s.Grants[0].Source = netip.MustParsePrefix("198.51.100.0/24")
	s.Grants[0].Validity.ExpiresAt = atHour("08:00")
	s.Grants[0].Revoked = true
	s.Grants[1].Scopes[0].Endpoints.Selector.MatchLabels["kind"] = "changed"
	s.Grants[1].Scopes[0].Endpoints.Selector.MatchExpressions[0].Values[0] = "changed"
	s.Denies[0].Scopes[0].Endpoints.Names[0] = "ssh"
	s.Denies[0].Revoked = true
	s.Denies[0].ID = "changed"
	s.Denies[0].Validity.ExpiresAt = atHour("08:00")
	if got := decideOK(t, e, query()); !reflect.DeepEqual(got, before) {
		t.Fatalf("input alias %+v", got)
	}
}
func TestDecisionOwnsResults(t *testing.T) {
	s := baseSnapshot()
	s.Denies = []Deny{Deny(s.Grants[0])}
	e := compileOK(t, s)
	d := decideOK(t, e, query())
	assertDecision(t, d, false, ReasonExplicitDeny, []string{"a-host", "b-prefix"}, []string{"a-host"})
	d.GrantIDs[0] = "changed"
	d.DenyIDs[0] = "changed"
	d.Contributions[0].GrantID = "changed"
	d.Contributions[0].Source = netip.Prefix{}
	d.Contributions[0].Validity.ExpiresAt = atHour("12:00")
	got := decideOK(t, e, query())
	assertDecision(t, got, false, ReasonExplicitDeny, []string{"a-host", "b-prefix"}, []string{"a-host"})
	if len(got.Contributions) != 2 || got.Contributions[0].GrantID != "a-host" || got.Contributions[0].Source.String() != "192.0.2.10/32" || !got.Contributions[0].Validity.ExpiresAt.Equal(atHour("10:00")) {
		t.Fatalf("output alias %+v", got)
	}
}
func TestDecisionDeterministic(t *testing.T) {
	s := baseSnapshot()
	s.Grants[0].Scopes = append(s.Grants[0].Scopes, s.Grants[0].Scopes[0])
	s.Grants[1].Scopes[0].Endpoints = EndpointSelection{Mode: SelectionLabels, Selector: &LabelSelector{MatchExpressions: []Requirement{{Key: "kind", Operator: SelectorIn, Values: []string{"web", "git", "web"}}}}}
	first := decideOK(t, compileOK(t, s), query())
	assertDecision(t, first, true, ReasonAllowed, []string{"a-host", "b-prefix"}, nil)
	slices.Reverse(s.Resources[0].Endpoints)
	slices.Reverse(s.Grants)
	slices.Reverse(s.Grants[0].Scopes[0].Endpoints.Selector.MatchExpressions[0].Values)
	slices.Reverse(s.Grants[1].Scopes)
	if got := decideOK(t, compileOK(t, s), query()); !reflect.DeepEqual(got, first) {
		t.Fatalf("order affects result %+v", got)
	}
}
func TestDecisionConcurrentRead(t *testing.T) {
	e := compileOK(t, baseSnapshot())
	var wg sync.WaitGroup
	for range 16 {
		wg.Go(func() {
			for range 50 {
				d, err := e.Decide(query())
				if err != nil || !d.Allowed || !reflect.DeepEqual(d.GrantIDs, []string{"a-host", "b-prefix"}) {
					t.Errorf("concurrent result %+v %v", d, err)
					return
				}
				d.GrantIDs[0] = "changed"
				d.Contributions[0].GrantID = "changed"
			}
		})
	}
	wg.Wait()
}
func TestInvalidReplacementPreservesEngine(t *testing.T) {
	s := baseSnapshot()
	e := compileOK(t, s)
	s.Grants[0].Scopes[0].ResourceName = "missing"
	if candidate, err := Compile(s); err == nil || candidate != nil {
		t.Fatal("invalid replacement accepted")
	}
	assertDecision(t, decideOK(t, e, query()), true, ReasonAllowed, []string{"a-host", "b-prefix"}, nil)
	q := query()
	q.At = atHour("10:00")
	assertDecision(t, decideOK(t, e, q), true, ReasonAllowed, []string{"b-prefix"}, nil)
	q.At = atHour("11:00")
	assertDecision(t, decideOK(t, e, q), false, ReasonNoActiveGrant, nil, nil)
}
