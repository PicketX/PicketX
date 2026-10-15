// SPDX-License-Identifier: Apache-2.0
package networkplan

import (
	"github.com/PicketX/PicketX/pkg/core"
	"net/netip"
	"reflect"
	"testing"
	"time"
)

func clock(h string) time.Time {
	v, e := time.Parse(time.RFC3339, "2026-10-15T"+h+":00Z")
	if e != nil {
		panic(e)
	}
	return v
}
func baseInput() Input {
	r, c, caps := contextFixture()
	r.AllowedDestinations = prefixes("192.0.2.0/24", "2001:db8::/64")
	c.Destinations = prefixes("192.0.2.0/24", "2001:db8::/64")
	second := c
	second.AgentRef = "agent-b"
	return Input{AuthorityID: "controller-1", Logical: core.Snapshot{
		Resources: []core.Resource{{Name: "gitea", Labels: map[string]string{"kind": "service"}, Endpoints: []core.Endpoint{{Name: "https", Labels: map[string]string{"access": "web"}}, {Name: "ssh", Labels: map[string]string{"access": "shell"}}}}},
		Grants: []core.Grant{
			{ID: "host", Source: netip.MustParsePrefix("198.51.100.9/32"), Scopes: []core.Scope{{ResourceName: "gitea"}, {ResourceName: "gitea"}}, Validity: core.Validity{Mode: core.LifetimeBounded, ExpiresAt: clock("10:00")}},
			{ID: "prefix", Source: netip.MustParsePrefix("198.51.100.0/24"), Scopes: []core.Scope{{ResourceName: "gitea"}}, Validity: core.Validity{Mode: core.LifetimeBounded, ExpiresAt: clock("11:00")}},
		},
	}, Networks: []NetworkDefinition{{Entry: EntryRef{"gitea", "https"}, Protocol: TCP, Port: 443}, {Entry: EntryRef{"gitea", "ssh"}, Protocol: TCP, Port: 22}},
		Agents:   []Agent{{Name: "agent-a", Labels: map[string]string{"site": "dev"}, Capabilities: caps}, {Name: "agent-b", Labels: map[string]string{"site": "dev"}, Capabilities: caps}},
		Contexts: []ExecutionContext{c, second}, Bindings: []Binding{{ID: "b1", Resources: []ResourceSelection{{ResourceName: "gitea"}}, Agents: []AgentSelection{{AgentRef: "agent-a"}}, Context: r}}}
}
func planOK(t *testing.T, in Input) *Plan {
	t.Helper()
	p, e := Compile(in)
	if e != nil || p == nil {
		t.Fatalf("compile: %v %v", p, e)
	}
	return p
}
func inventoryOK(t *testing.T, p *Plan, a string) []TargetInventory {
	t.Helper()
	v, e := p.Inventory(a)
	if e != nil {
		t.Fatal(e)
	}
	return v
}
func projectionOK(t *testing.T, p *Plan, at string) Projection {
	t.Helper()
	v, e := p.Projection("agent-a", clock(at))
	if e != nil {
		t.Fatal(e)
	}
	return v
}
func selector(k, v string) *core.LabelSelector {
	return &core.LabelSelector{MatchLabels: map[string]string{k: v}}
}
func TestBindingCrossProduct(t *testing.T) {
	in := baseInput()
	in.Bindings[0].Agents = append(in.Bindings[0].Agents, AgentSelection{Selector: selector("site", "dev")})
	p := planOK(t, in)
	a := inventoryOK(t, p, "agent-a")
	b := inventoryOK(t, p, "agent-b")
	if len(a) != 2 || len(b) != 2 {
		t.Fatal(a, b)
	}
	if refs, e := p.AgentRefs(); e != nil || !reflect.DeepEqual(refs, []string{"agent-a", "agent-b"}) {
		t.Fatal(refs, e)
	}
	for _, target := range append(a, b...) {
		if len(target.Grants) != 2 || !reflect.DeepEqual(target.Protection.BindingIDs, []string{"b1"}) {
			t.Fatal(target)
		}
	}
}
func TestBindingValidation(t *testing.T) {
	mutations := map[string]func(*Input){
		"authority": func(i *Input) { i.AuthorityID = "" }, "binding ID": func(i *Input) { i.Bindings[0].ID = "" }, "duplicate Binding": func(i *Input) { i.Bindings = append(i.Bindings, i.Bindings[0]) },
		"resource items": func(i *Input) { i.Bindings[0].Resources = nil }, "agent items": func(i *Input) { i.Bindings[0].Agents = nil },
		"resource selector": func(i *Input) { i.Bindings[0].Resources[0].ResourceSelector = selector("kind", "service") },
		"missing resource":  func(i *Input) { i.Bindings[0].Resources[0].ResourceName = "missing" },
		"missing endpoint": func(i *Input) {
			i.Bindings[0].Resources[0].Endpoints = core.EndpointSelection{Mode: core.SelectionNames, Names: []string{"missing"}}
		},
		"empty names": func(i *Input) {
			i.Bindings[0].Resources[0].Endpoints = core.EndpointSelection{Mode: core.SelectionNames}
		},
		"both Agent": func(i *Input) { i.Bindings[0].Agents[0].Selector = selector("site", "dev") }, "neither Agent": func(i *Input) { i.Bindings[0].Agents[0] = AgentSelection{} },
		"empty selector": func(i *Input) { i.Bindings[0].Agents[0] = AgentSelection{Selector: &core.LabelSelector{}} },
		"missing Agent":  func(i *Input) { i.Bindings[0].Agents[0].AgentRef = "missing" }, "empty Agent": func(i *Input) { i.Agents[0].Name = "" }, "duplicate Agent": func(i *Input) { i.Agents = append(i.Agents, i.Agents[0]) },
		"missing network": func(i *Input) { i.Networks = i.Networks[:1] }, "duplicate network": func(i *Input) { i.Networks = append(i.Networks, i.Networks[0]) }, "orphan network": func(i *Input) { i.Networks[1].Entry.EndpointName = "orphan" },
		"unused network invalid": func(i *Input) { i.Bindings = nil; i.Networks[0].Port = 0 },
		"unused context invalid": func(i *Input) { i.Contexts[1].Class = "bad" }, "unknown context Agent": func(i *Input) { i.Contexts[1].AgentRef = "missing" },
		"unused caps invalid": func(i *Input) { i.Agents[1].Capabilities = &CapabilityProfile{Protocols: []Protocol{"bad"}} },
		"inactive malformed grant": func(i *Input) {
			i.Logical.Grants[0].Revoked = true
			i.Logical.Grants[0].Scopes[0].Actions = []string{"read"}
		},
		"ref with no Agents": func(i *Input) {
			i.Bindings[0].Agents = []AgentSelection{{Selector: selector("no", "match")}}
			i.Bindings[0].Resources[0].ResourceName = "missing"
		},
	}
	for name, m := range mutations {
		t.Run(name, func(t *testing.T) {
			i := baseInput()
			m(&i)
			p, e := Compile(i)
			if e == nil || p != nil {
				t.Fatalf("accepted %v %v", p, e)
			}
		})
	}
	for _, r := range []string{"port-range", "nat64", "dns", "post-dnat", "unknown"} {
		i := baseInput()
		i.Requirements = []string{r}
		if p, e := Compile(i); e == nil || p != nil {
			t.Errorf("accepted %s", r)
		}
	}
}
func TestBindingNoMatchesAndOffline(t *testing.T) {
	in := baseInput()
	in.Bindings[0].Agents = []AgentSelection{{Selector: selector("kind", "service")}}
	p := planOK(t, in)
	d, e := p.Diagnostics()
	if e != nil || len(d) != 1 || d[0].Code != "no-agents-matched" || d[0].BindingID != "b1" {
		t.Fatal(d, e)
	}
	if v := inventoryOK(t, p, "agent-a"); len(v) != 0 {
		t.Fatal(v)
	}
	in = baseInput()
	in.Logical.Resources = append(in.Logical.Resources, core.Resource{Name: "empty"})
	in.Bindings[0].Resources = []ResourceSelection{{ResourceName: "empty"}}
	p = planOK(t, in)
	d, e = p.Diagnostics()
	if e != nil || len(d) != 1 || d[0].Code != "no-entries-matched" || d[0].ResourceName != "empty" {
		t.Fatal(d, e)
	}
	in = baseInput()
	in.Agents[0].Offline = true
	p = planOK(t, in)
	if v := inventoryOK(t, p, "agent-a"); len(v) != 2 {
		t.Fatal(v)
	}
}
func TestContextSelectionAndCapabilities(t *testing.T) {
	for name, m := range map[string]func(*Input){
		"missing context": func(i *Input) { i.Contexts = nil }, "nil caps": func(i *Input) { i.Agents[0].Capabilities = nil },
		"protocol": func(i *Input) {
			c := *i.Agents[0].Capabilities
			c.Protocols = []Protocol{UDP}
			i.Agents[0].Capabilities = &c
		},
		"family": func(i *Input) {
			c := *i.Agents[0].Capabilities
			c.Families = []Family{IPv4}
			i.Agents[0].Capabilities = &c
		},
		"duplicate profile": func(i *Input) { c := i.Contexts[0]; c.ID = "different"; i.Contexts = append(i.Contexts, c) },
		"duplicate ID":      func(i *Input) { c := i.Contexts[0]; c.ProfileRef = "different"; i.Contexts = append(i.Contexts, c) },
		"exact duplicate":   func(i *Input) { i.Contexts = append(i.Contexts, i.Contexts[0]) },
	} {
		t.Run(name, func(t *testing.T) {
			i := baseInput()
			m(&i)
			if p, e := Compile(i); e == nil || p != nil {
				t.Fatal(p, e)
			}
		})
	}
	in := baseInput()
	in.Bindings[0].Agents = []AgentSelection{{AgentRef: "agent-a"}, {AgentRef: "agent-b"}}
	p := planOK(t, in)
	if len(inventoryOK(t, p, "agent-a")) != 2 || len(inventoryOK(t, p, "agent-b")) != 2 {
		t.Fatal("context not Agent scoped")
	}
	in = baseInput()
	in.Agents[1].Capabilities = nil
	planOK(t, in)
	// Physical v4-only matches need no v6 backend even inside a dual-stack context.
	for j := range in.Networks {
		in.Networks[j].Addresses = prefixes("192.0.2.0/24")
	}
	c := *in.Agents[0].Capabilities
	c.Families = []Family{IPv4}
	in.Agents[0].Capabilities = &c
	p = planOK(t, in)
	if got := inventoryOK(t, p, "agent-a")[0].Protection.Required; !reflect.DeepEqual(got.Families, []Family{IPv4}) || !reflect.DeepEqual(got.Protocols, []Protocol{TCP}) || !got.OriginalDestination {
		t.Fatal(got)
	}
}
