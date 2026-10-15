// SPDX-License-Identifier: Apache-2.0
package networkplan

import (
	"github.com/PicketX/PicketX/pkg/core"
	"net/netip"
	"reflect"
	"strings"
	"testing"
)

func oneTargetInput() Input {
	i := baseInput()
	i.Bindings[0].Resources[0].Endpoints = core.EndpointSelection{Mode: core.SelectionNames, Names: []string{"https"}}
	return i
}
func replaceOK(t *testing.T, old *Plan, in Input) Replacement {
	t.Helper()
	r, e := Replace(old, in)
	if e != nil || r.Plan == nil {
		t.Fatalf("replacement: %v %v", r, e)
	}
	return r
}
func rejectReplacement(t *testing.T, old *Plan, in Input) error {
	t.Helper()
	r, e := Replace(old, in)
	if e == nil || r.Plan != nil || len(r.Withdrawals) != 0 {
		t.Fatalf("accepted invalid replacement: %v %v", r, e)
	}
	return e
}
func TestReplaceAuthorityAndAtomicity(t *testing.T) {
	in := oneTargetInput()
	old := planOK(t, in)
	before := inventoryOK(t, old, "agent-a")
	rejectReplacement(t, nil, in)
	rejectReplacement(t, &Plan{}, in)
	changed := oneTargetInput()
	changed.AuthorityID = "static-1"
	rejectReplacement(t, old, changed)
	changed = oneTargetInput()
	changed.Networks[0].Port = 0
	rejectReplacement(t, old, changed)
	if after := inventoryOK(t, old, "agent-a"); !reflect.DeepEqual(before, after) {
		t.Fatal(after)
	}
	if got := projectionOK(t, old, "10:00"); len(got.Allow) != 1 || got.Allow[0].Contribution.ID != "prefix" || !got.NextChange.Equal(clock("11:00")) {
		t.Fatal(got)
	}
	if got := projectionOK(t, old, "11:00"); len(got.Allow) != 0 {
		t.Fatal(got)
	}
	// A valid unchanged complete replacement is still usable after failed candidates.
	replaceOK(t, old, oneTargetInput())
}
func TestRetargetRetainedContributions(t *testing.T) {
	for _, domain := range []string{"grant", "deny"} {
		for _, state := range []string{"active", "unbound", "expired", "revoked"} {
			for _, kind := range []string{"port", "protocol", "hidden address"} {
				t.Run(domain+"/"+state+"/"+kind, func(t *testing.T) {
					oldInput := oneTargetInput()
					oldInput.Logical.Grants = oldInput.Logical.Grants[:1]
					oldInput.Networks[0].Addresses = prefixes("192.0.2.0/24")
					oldInput.Contexts[0].Destinations = prefixes("192.0.2.128/25")
					g := oldInput.Logical.Grants[0]
					if state == "expired" {
						g.Validity.ExpiresAt = clock("07:00")
					}
					if state == "revoked" {
						g.Revoked = true
					}
					if state == "unbound" {
						oldInput.Bindings = nil
					}
					oldInput.Logical.Grants = []core.Grant{g}
					if domain == "deny" {
						oldInput.Logical.Grants = nil
						oldInput.Logical.Denies = []core.Deny{core.Deny(g)}
					}
					old := planOK(t, oldInput)
					candidate := oldInput
					candidate.Networks = append([]NetworkDefinition(nil), oldInput.Networks...)
					switch kind {
					case "port":
						candidate.Networks[0].Port = 8443
					case "protocol":
						candidate.Networks[0].Protocol = UDP
					case "hidden address":
						candidate.Networks[0].Addresses = prefixes("192.0.2.128/25")
					}
					e := rejectReplacement(t, old, candidate)
					for _, part := range []string{"gitea", "https", domain, "host"} {
						if !strings.Contains(e.Error(), part) {
							t.Fatalf("missing %s: %v", part, e)
						}
					}
				})
			}
		}
	}
	// A same string ID in the other domain is a new typed identity, not retention.
	in := oneTargetInput()
	in.Logical.Grants = in.Logical.Grants[:1]
	old := planOK(t, in)
	candidate := oneTargetInput()
	candidate.Logical.Grants = nil
	candidate.Logical.Denies = []core.Deny{core.Deny(in.Logical.Grants[0])}
	candidate.Networks[0].Port = 8443
	replaceOK(t, old, candidate)
}
func TestRetargetCanonicalAndReissue(t *testing.T) {
	in := oneTargetInput()
	in.Networks[0].Addresses = prefixes("192.0.2.0/24")
	old := planOK(t, in)
	candidate := oneTargetInput()
	candidate.Networks[0].Addresses = prefixes("192.0.2.128/25", "192.0.2.0/25")
	if r := replaceOK(t, old, candidate); len(r.Withdrawals) != 0 {
		t.Fatal(r)
	}
	candidate.Networks[0].Addresses = nil
	rejectReplacement(t, old, candidate)
	candidate = oneTargetInput()
	candidate.Networks[0].Port = 8443
	for j := range candidate.Logical.Grants {
		candidate.Logical.Grants[j].ID = "reissued-" + candidate.Logical.Grants[j].ID
	}
	r := replaceOK(t, old, candidate)
	if len(r.Withdrawals) != 1 || r.Withdrawals[0].Reason != "match-changed" {
		t.Fatal(r)
	}
	v := inventoryOK(t, r.Plan, "agent-a")
	if v[0].Protection.Matches[0].Port != 8443 {
		t.Fatal(v)
	}
	for _, c := range v[0].Grants {
		want := clock("10:00")
		if c.ID == "reissued-prefix" {
			want = clock("11:00")
		}
		if !c.Validity.ExpiresAt.Equal(want) {
			t.Fatal(c)
		}
	}
}
func TestContextIdentityMutation(t *testing.T) {
	for name, m := range map[string]func(*ExecutionContext){
		"space": func(c *ExecutionContext) { c.NetworkSpaceID = "new-net" }, "class": func(c *ExecutionContext) { c.Class = Transit }, "stage": func(c *ExecutionContext) { c.Stage = "post-dnat" }, "boundary": func(c *ExecutionContext) { c.Destinations = prefixes("192.0.2.128/25") }, "handling": func(c *ExecutionContext) { c.Handling = Strict }, "profile": func(c *ExecutionContext) { c.ProfileRef = "new-profile" },
	} {
		t.Run(name, func(t *testing.T) {
			in := oneTargetInput()
			in.Bindings = nil
			old := planOK(t, in)
			candidate := oneTargetInput()
			candidate.Bindings = nil
			m(&candidate.Contexts[0])
			rejectReplacement(t, old, candidate)
		})
	}
	in := oneTargetInput()
	old := planOK(t, in)
	candidate := oneTargetInput()
	candidate.Contexts[0].ID = "ctx-new"
	r := replaceOK(t, old, candidate)
	if len(r.Withdrawals) != 1 || r.Withdrawals[0].Former.Protection.Context.ID != "ctx-a" || inventoryOK(t, r.Plan, "agent-a")[0].Protection.Context.ID != "ctx-new" {
		t.Fatal(r)
	}
}
func TestWithdrawalInventory(t *testing.T) {
	in := oneTargetInput()
	b := in.Bindings[0]
	b.ID = "b2"
	in.Bindings = append(in.Bindings, b)
	old := planOK(t, in)
	in.Bindings = in.Bindings[1:]
	r := replaceOK(t, old, in)
	if len(r.Withdrawals) != 0 || !reflect.DeepEqual(inventoryOK(t, r.Plan, "agent-a")[0].Protection.BindingIDs, []string{"b2"}) {
		t.Fatal(r)
	}
	old = r.Plan
	in.Logical.Grants = nil
	r = replaceOK(t, old, in)
	if len(r.Withdrawals) != 0 || len(inventoryOK(t, r.Plan, "agent-a")) != 1 || len(inventoryOK(t, r.Plan, "agent-a")[0].Grants) != 0 {
		t.Fatal(r)
	}
	old = r.Plan
	in.Bindings = nil
	r = replaceOK(t, old, in)
	if len(r.Withdrawals) != 1 || r.Withdrawals[0].Reason != "removed" || len(inventoryOK(t, r.Plan, "agent-a")) != 0 {
		t.Fatal(r)
	}
	// Removed Agent still carries former complete ownership without a current lookup.
	in = oneTargetInput()
	old = planOK(t, in)
	before := inventoryOK(t, old, "agent-a")[0]
	in.Bindings = nil
	in.Agents = in.Agents[1:]
	in.Contexts = in.Contexts[1:]
	r = replaceOK(t, old, in)
	if len(r.Withdrawals) != 1 || !reflect.DeepEqual(r.Withdrawals[0].Former, before) {
		t.Fatal(r)
	}
	r.Withdrawals[0].Former.Protection.Matches[0].Destination = netip.Prefix{}
	r.Withdrawals[0].Former.Protection.BindingIDs[0] = "mutated"
	r.Withdrawals[0].Former.Grants[0].ID = "mutated"
	if got := inventoryOK(t, old, "agent-a")[0]; !reflect.DeepEqual(got, before) {
		t.Fatal(got)
	}
}
func TestHistoricalFormerInventory(t *testing.T) {
	in := oneTargetInput()
	old := planOK(t, in)
	if v := projectionOK(t, old, "12:00"); len(v.Allow) != 0 {
		t.Fatal(v)
	}
	in.Bindings = nil
	r := replaceOK(t, old, in)
	if len(r.Withdrawals) != 1 || len(r.Withdrawals[0].Former.Grants) != 2 {
		t.Fatal(r)
	}
	if !r.Withdrawals[0].Former.Grants[0].Validity.ExpiresAt.Equal(clock("10:00")) {
		t.Fatal(r)
	}
}
func TestEntryLifecycleSelections(t *testing.T) {
	for _, mode := range []string{"whole", "names", "labels"} {
		t.Run(mode, func(t *testing.T) {
			in := baseInput()
			if mode == "names" {
				in.Bindings[0].Resources[0].Endpoints = core.EndpointSelection{Mode: core.SelectionNames, Names: []string{"https"}}
			}
			if mode == "labels" {
				in.Bindings[0].Resources[0].Endpoints = core.EndpointSelection{Mode: core.SelectionLabels, Selector: selector("access", "web")}
			}
			old := planOK(t, in)
			candidate := baseInput()
			candidate.Bindings = in.Bindings
			candidate.Logical.Resources[0].Endpoints = append(candidate.Logical.Resources[0].Endpoints, core.Endpoint{Name: "extra", Labels: map[string]string{"access": "web"}})
			candidate.Networks = append(candidate.Networks, NetworkDefinition{Entry: EntryRef{"gitea", "extra"}, Protocol: TCP, Port: 8443})
			r := replaceOK(t, old, candidate)
			want := 3
			if mode == "names" {
				want = 1
			}
			if mode == "labels" {
				want = 2
			}
			if got := len(inventoryOK(t, r.Plan, "agent-a")); got != want {
				t.Fatalf("got %d want %d", got, want)
			}
			// Labels are intentionally dynamic; adding a matching label selects ssh too.
			if mode == "labels" {
				candidate.Logical.Resources[0].Endpoints[1].Labels["access"] = "web"
				r = replaceOK(t, r.Plan, candidate)
				if len(inventoryOK(t, r.Plan, "agent-a")) != 3 {
					t.Fatal(r)
				}
			}
		})
	}
	in := oneTargetInput()
	in.Logical.Grants[0].Scopes = []core.Scope{{ResourceName: "gitea", Endpoints: core.EndpointSelection{Mode: core.SelectionNames, Names: []string{"https"}}}}
	old := planOK(t, in)
	candidate := oneTargetInput()
	candidate.Logical.Grants[0].Scopes = in.Logical.Grants[0].Scopes
	candidate.Logical.Resources[0].Endpoints = candidate.Logical.Resources[0].Endpoints[1:]
	candidate.Networks = candidate.Networks[1:]
	candidate.Bindings[0].Resources[0].Endpoints = core.EndpointSelection{}
	rejectReplacement(t, old, candidate)
	// Remove the missing named contribution; whole scope follows the surviving entry.
	candidate.Logical.Grants = candidate.Logical.Grants[1:]
	r := replaceOK(t, old, candidate)
	if got := inventoryOK(t, r.Plan, "agent-a"); len(got) != 1 || got[0].Protection.Key.Entry.EndpointName != "ssh" {
		t.Fatal(got)
	}
}
