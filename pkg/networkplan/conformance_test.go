// SPDX-License-Identifier: Apache-2.0
package networkplan

import (
	"github.com/PicketX/PicketX/pkg/core"
	"net/netip"
	"reflect"
	"slices"
	"sync"
	"testing"
	"time"
)

func giteaInput() Input {
	in := baseInput()
	in.Contexts[0].Destinations = prefixes("192.0.2.100/32", "2001:db8::100/128")
	in.Logical.Grants = []core.Grant{
		{ID: "https", Source: netip.MustParsePrefix("198.51.100.9/32"), Scopes: []core.Scope{{ResourceName: "gitea", Endpoints: core.EndpointSelection{Mode: core.SelectionNames, Names: []string{"https"}}}}, Validity: core.Validity{Mode: core.LifetimeBounded, ExpiresAt: clock("10:00")}},
		{ID: "ssh", Source: netip.MustParsePrefix("198.51.100.9/32"), Scopes: []core.Scope{{ResourceName: "gitea", Endpoints: core.EndpointSelection{Mode: core.SelectionNames, Names: []string{"ssh"}}}}, Validity: core.Validity{Mode: core.LifetimeBounded, ExpiresAt: clock("12:00")}},
		{ID: "prefix", Source: netip.MustParsePrefix("198.51.100.0/24"), Scopes: []core.Scope{{ResourceName: "gitea"}}, Validity: core.Validity{Mode: core.LifetimeBounded, ExpiresAt: clock("11:00")}},
		{ID: "v6", Source: netip.MustParsePrefix("2001:db8:feed::9/128"), Scopes: []core.Scope{{ResourceName: "gitea"}}, Validity: core.Validity{Mode: core.LifetimeBounded, ExpiresAt: clock("11:00")}},
	}
	deny := in.Logical.Grants[0]
	deny.ID = "block-host"
	deny.Validity = core.Validity{Mode: core.LifetimeBounded, NotBefore: clock("09:30"), ExpiresAt: clock("10:30")}
	in.Logical.Denies = []core.Deny{core.Deny(deny)}
	// Core may allow this catalog entry, but the Binding deliberately leaves it unmanaged.
	in.Logical.Resources[0].Endpoints = append(in.Logical.Resources[0].Endpoints, core.Endpoint{Name: "unbound"})
	in.Networks = append(in.Networks, NetworkDefinition{Entry: EntryRef{"gitea", "unbound"}, Protocol: TCP, Port: 8443})
	in.Bindings[0].Resources[0].Endpoints = core.EndpointSelection{Mode: core.SelectionNames, Names: []string{"https", "ssh"}}
	return in
}

// observedIntent is only a test evaluator over explicit owned projection tuples,
// never an implementation of a runtime hook or a public policy decision service.
func observedIntent(p Projection, endpoint string, source netip.Addr) (bool, []string, []string) {
	var allows, denies []string
	for _, domain := range []struct {
		members []Membership
		deny    bool
	}{{p.Allow, false}, {p.Deny, true}} {
		for _, m := range domain.members {
			if m.Target.Entry.EndpointName == endpoint && m.Contribution.Source.Contains(source) {
				if domain.deny {
					denies = append(denies, m.Contribution.ID)
				} else {
					allows = append(allows, m.Contribution.ID)
				}
			}
		}
	}
	slices.Sort(allows)
	slices.Sort(denies)
	return len(allows) > 0 && len(denies) == 0, allows, denies
}
func TestGiteaConformance(t *testing.T) {
	in := giteaInput()
	p := planOK(t, in)
	engine, e := core.Compile(in.Logical)
	if e != nil {
		t.Fatal(e)
	}
	inv := inventoryOK(t, p, "agent-a")
	if len(inv) != 2 {
		t.Fatal(inv)
	}
	for _, target := range inv {
		port := uint16(443)
		if target.Protection.Key.Entry.EndpointName == "ssh" {
			port = 22
		}
		want := []Match{{Destination: netip.MustParsePrefix("192.0.2.100/32"), Protocol: TCP, Port: port}, {Destination: netip.MustParsePrefix("2001:db8::100/128"), Protocol: TCP, Port: port}}
		if !reflect.DeepEqual(target.Protection.Matches, want) || target.Protection.Context.Class != HostLocal || target.Protection.Context.Stage != OriginalDestination {
			t.Fatal(target)
		}
	}
	for _, tc := range []struct {
		endpoint, source, at string
		allowed              bool
		grants, denies       []string
		next                 string
	}{
		{"https", "198.51.100.9", "09:00", true, []string{"https", "prefix"}, nil, "09:30"},
		{"https", "198.51.100.9", "10:00", false, []string{"prefix"}, []string{"block-host"}, "10:30"},
		{"https", "198.51.100.9", "10:30", true, []string{"prefix"}, nil, "11:00"},
		{"https", "198.51.100.9", "11:00", false, nil, nil, "12:00"},
		{"ssh", "198.51.100.9", "11:00", true, []string{"ssh"}, nil, "12:00"},
		{"ssh", "198.51.100.9", "12:00", false, nil, nil, ""},
		{"https", "198.51.100.10", "09:45", true, []string{"prefix"}, nil, "10:00"},
		{"ssh", "2001:db8:feed::9", "10:00", true, []string{"v6"}, nil, "10:30"},
	} {
		t.Run(tc.endpoint+"/"+tc.source+"/"+tc.at, func(t *testing.T) {
			projection := projectionOK(t, p, tc.at)
			source := netip.MustParseAddr(tc.source)
			allowed, grants, denies := observedIntent(projection, tc.endpoint, source)
			if allowed != tc.allowed || !reflect.DeepEqual(grants, tc.grants) || !reflect.DeepEqual(denies, tc.denies) {
				t.Fatalf("physical %v %v %v want %v %v %v", allowed, grants, denies, tc.allowed, tc.grants, tc.denies)
			}
			decision, e := engine.Decide(core.Query{Source: source, ResourceName: "gitea", EndpointName: tc.endpoint, At: clock(tc.at)})
			if e != nil || decision.Allowed != tc.allowed || !reflect.DeepEqual(decision.GrantIDs, tc.grants) || !reflect.DeepEqual(decision.DenyIDs, tc.denies) {
				t.Fatal(decision, e)
			}
			next := time.Time{}
			if tc.next != "" {
				next = clock(tc.next)
			}
			if !projection.NextChange.Equal(next) {
				t.Fatalf("next %v want %v", projection.NextChange, next)
			}
		})
	}
	v := projectionOK(t, p, "09:00")
	for _, tgt := range v.Protected {
		if tgt.Key.Entry.EndpointName == "unbound" {
			t.Fatal("unmanaged entry claimed")
		}
	}
	d, e := engine.Decide(core.Query{Source: netip.MustParseAddr("198.51.100.9"), ResourceName: "gitea", EndpointName: "unbound", At: clock("09:00")})
	if e != nil || !d.Allowed {
		t.Fatal(d, e)
	}
	// Revoking one host contribution preserves an overlapping wider prefix permission.
	in.Logical.Grants[0].Revoked = true
	r := replaceOK(t, p, in)
	got := projectionOK(t, r.Plan, "09:00")
	allowed, grants, _ := observedIntent(got, "https", netip.MustParseAddr("198.51.100.9"))
	if !allowed || !reflect.DeepEqual(grants, []string{"prefix"}) {
		t.Fatal(allowed, grants)
	}
	// Fresh compilation and replacement preserve absolute ends instead of renewing.
	reload := projectionOK(t, planOK(t, in), "11:00")
	if allowed, _, _ := observedIntent(reload, "https", netip.MustParseAddr("198.51.100.9")); allowed {
		t.Fatal("reload renewed expiry")
	}
}
func TestProviderNormalizedEquivalence(t *testing.T) {
	a := giteaInput()
	b := giteaInput()
	b.AuthorityID = "static-1"
	slices.Reverse(b.Logical.Resources[0].Endpoints)
	slices.Reverse(b.Logical.Grants)
	slices.Reverse(b.Networks)
	slices.Reverse(b.Agents)
	slices.Reverse(b.Contexts)
	b.Contexts[1].Destinations = prefixes("2001:db8::100/128", "192.0.2.100/32", "192.0.2.100/32")
	// agent-a moved to row 1 after reversing; use explicit normalized endpoint defaults.
	for j := range b.Logical.Resources[0].Endpoints {
		b.Logical.Resources[0].Endpoints[j].Type = "network"
	}
	pa, pb := planOK(t, a), planOK(t, b)
	for _, at := range []string{"09:00", "10:00", "11:00", "12:00"} {
		x, y := projectionOK(t, pa, at), projectionOK(t, pb, at)
		x.AuthorityID = ""
		y.AuthorityID = ""
		if !reflect.DeepEqual(x, y) {
			t.Fatalf("%s: %v != %v", at, x, y)
		}
	}
	rejectReplacement(t, pa, b)
}
func TestPlanOwnershipAndConcurrentReads(t *testing.T) {
	in := baseInput()
	for j := range in.Networks {
		in.Networks[j].Addresses = prefixes("192.0.2.100/32")
	}
	deny := in.Logical.Grants[0]
	deny.ID = "deny"
	in.Logical.Denies = []core.Deny{core.Deny(deny)}
	in.Bindings[0].Agents = append(in.Bindings[0].Agents, AgentSelection{Selector: &core.LabelSelector{MatchExpressions: []core.Requirement{{Key: "site", Operator: core.SelectorIn, Values: []string{"dev"}}}}})
	in.Bindings[0].Resources[0].Endpoints = core.EndpointSelection{Mode: core.SelectionLabels, Selector: selector("access", "web")}
	p := planOK(t, in)
	before := projectionOK(t, p, "09:00")
	beforeInventory := inventoryOK(t, p, "agent-a")
	beforeRefs, _ := p.AgentRefs()
	in.Logical.Resources[0].Labels["kind"] = "mutated"
	in.Logical.Resources[0].Endpoints[0].Labels["access"] = "mutated"
	in.Logical.Resources[0].Endpoints[0].Name = "mutated"
	in.Logical.Grants[0].Source = netip.Prefix{}
	in.Logical.Grants[0].Scopes[0].ResourceName = "mutated"
	in.Logical.Grants[0].Validity.ExpiresAt = clock("12:00")
	in.Logical.Denies[0].ID = "mutated"
	in.Networks[0].Addresses[0] = netip.Prefix{}
	in.Contexts[0].Destinations[0] = netip.Prefix{}
	in.Bindings[0].Context.AllowedDestinations[0] = netip.Prefix{}
	in.Bindings[0].Resources[0].Endpoints.Selector.MatchLabels["access"] = "mutated"
	in.Bindings[0].Agents[1].Selector.MatchExpressions[0].Values[0] = "mutated"
	in.Bindings[0].ID = "mutated"
	in.Agents[0].Labels["site"] = "mutated"
	in.Agents[0].Capabilities.Classes[0] = "mutated"
	in.Agents[0].Capabilities.Protocols[0] = "mutated"
	in.Agents[0].Capabilities.Families[0] = 99
	in.Agents[0].Capabilities.Handling[0] = "mutated"
	mutateProjection := func(v Projection) {
		v.Protected[0].BindingIDs[0] = "mutated"
		v.Protected[0].Matches[0].Destination = netip.Prefix{}
		v.Protected[0].Context.Destinations[0] = netip.Prefix{}
		v.Protected[0].Required.Classes[0] = "mutated"
		v.Protected[0].Required.Protocols[0] = "mutated"
		v.Protected[0].Required.Families[0] = 99
		v.Protected[0].Required.Handling[0] = "mutated"
		v.Allow[0].Contribution.ID = "mutated"
		v.Deny[0].Contribution.Revoked = true
	}
	mutateInventory := func(v []TargetInventory) {
		v[0].Protection.BindingIDs[0] = "mutated"
		v[0].Protection.Matches[0].Destination = netip.Prefix{}
		v[0].Protection.Context.Destinations[0] = netip.Prefix{}
		v[0].Protection.Required.Classes[0] = "mutated"
		v[0].Protection.Required.Protocols[0] = "mutated"
		v[0].Protection.Required.Families[0] = 99
		v[0].Protection.Required.Handling[0] = "mutated"
		v[0].Grants[0].ID = "mutated"
		v[0].Denies[0].ID = "mutated"
	}
	mutateProjection(projectionOK(t, p, "09:00"))
	mutateInventory(inventoryOK(t, p, "agent-a"))
	refs, _ := p.AgentRefs()
	refs[0] = "mutated"
	if got := projectionOK(t, p, "09:00"); !reflect.DeepEqual(got, before) {
		t.Fatal(got)
	}
	if got := inventoryOK(t, p, "agent-a"); !reflect.DeepEqual(got, beforeInventory) {
		t.Fatal(got)
	}
	if got, _ := p.AgentRefs(); !reflect.DeepEqual(got, beforeRefs) {
		t.Fatal(got)
	}
	// Diagnostics and empty result ownership are separate from populated target results.
	empty := baseInput()
	empty.Bindings[0].Agents = []AgentSelection{{Selector: selector("missing", "x")}}
	ep := planOK(t, empty)
	diags, _ := ep.Diagnostics()
	diags[0].Code = "mutated"
	if got, _ := ep.Diagnostics(); len(got) != 1 || got[0].Code != "no-agents-matched" {
		t.Fatal(got)
	}
	var wg sync.WaitGroup
	for range 24 {
		wg.Go(func() {
			v, e := p.Projection("agent-a", clock("09:00"))
			if e != nil || !reflect.DeepEqual(v, before) {
				t.Error(v, e)
				return
			}
			mutateProjection(v)
			inv, e := p.Inventory("agent-a")
			if e != nil || !reflect.DeepEqual(inv, beforeInventory) {
				t.Error(inv, e)
				return
			}
			mutateInventory(inv)
			r, e := p.AgentRefs()
			if e != nil || !reflect.DeepEqual(r, beforeRefs) {
				t.Error(r, e)
				return
			}
			r[0] = "private"
			d, e := ep.Diagnostics()
			if e != nil || len(d) != 1 || d[0].Code != "no-agents-matched" {
				t.Error(d, e)
				return
			}
			d[0].Code = "private"
		})
	}
	wg.Wait()
}
