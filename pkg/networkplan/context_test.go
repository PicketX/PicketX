// SPDX-License-Identifier: Apache-2.0
package networkplan

import (
	"net/netip"
	"reflect"
	"testing"
)

func contextFixture() (ContextRequest, ExecutionContext, *CapabilityProfile) {
	return ContextRequest{ProfileRef: "host", Class: HostLocal, Stage: OriginalDestination, AllowedDestinations: prefixes("192.0.2.0/24"), Handling: Graceful},
		ExecutionContext{AgentRef: "agent-a", ProfileRef: "host", ID: "ctx-a", NetworkSpaceID: "host-net", Class: HostLocal, Stage: OriginalDestination, Destinations: prefixes("192.0.2.128/25"), Handling: Graceful},
		&CapabilityProfile{Classes: []TrafficClass{HostLocal}, Protocols: []Protocol{TCP, UDP}, Families: []Family{IPv4, IPv6}, Handling: []HandlingMode{Graceful, Strict}, OriginalDestination: true}
}
func TestResolveContextAuthorityBoundary(t *testing.T) {
	r, c, caps := contextFixture()
	got, err := ResolveContext(r, c, caps)
	if err != nil || !reflect.DeepEqual(got.Destinations, prefixes("192.0.2.128/25")) {
		t.Fatal(got, err)
	}
	got.Destinations[0] = netip.Prefix{}
	if !c.Destinations[0].IsValid() {
		t.Fatal("alias")
	}
	for _, mutate := range []func(*ExecutionContext){
		func(c *ExecutionContext) { c.Destinations = prefixes("192.0.2.0/23") }, func(c *ExecutionContext) { c.Class = Transit }, func(c *ExecutionContext) { c.Stage = "post-dnat" }, func(c *ExecutionContext) { c.Handling = Strict }, func(c *ExecutionContext) { c.ProfileRef = "other" }, func(c *ExecutionContext) { c.Destinations = nil },
	} {
		bad := c
		mutate(&bad)
		if _, err := ResolveContext(r, bad, caps); err == nil {
			t.Errorf("accepted %v", bad)
		}
	}
	for _, mutate := range []func(*ContextRequest){func(r *ContextRequest) { r.ProfileRef = "" }, func(r *ContextRequest) { r.Class = "bad" }, func(r *ContextRequest) { r.Stage = "" }, func(r *ContextRequest) { r.Handling = "" }, func(r *ContextRequest) { r.AllowedDestinations = nil }, func(r *ContextRequest) { r.AllowedDestinations = []netip.Prefix{} }} {
		bad := r
		mutate(&bad)
		if _, err := NormalizeContextRequest(bad); err == nil {
			t.Errorf("accepted %v", bad)
		}
	}
}
func TestNormalizeExecutionContext(t *testing.T) {
	_, c, _ := contextFixture()
	for _, mutate := range []func(*ExecutionContext){
		func(c *ExecutionContext) { c.AgentRef = "" }, func(c *ExecutionContext) { c.ProfileRef = "" }, func(c *ExecutionContext) { c.ID = "" }, func(c *ExecutionContext) { c.NetworkSpaceID = "" }, func(c *ExecutionContext) { c.Class = "bad" }, func(c *ExecutionContext) { c.Stage = "bad" }, func(c *ExecutionContext) { c.Handling = "bad" }, func(c *ExecutionContext) { c.Destinations = []netip.Prefix{} },
	} {
		bad := c
		mutate(&bad)
		if _, err := NormalizeExecutionContext(bad); err == nil {
			t.Errorf("accepted %v", bad)
		}
	}
	for _, class := range []TrafficClass{HostLocal, PublishedIngress, Transit} {
		c.Class = class
		c.Destinations = prefixes("0.0.0.0/0", "::/0")
		got, err := NormalizeExecutionContext(c)
		if err != nil || got.Class != class || len(got.Destinations) != 2 {
			t.Fatal(got, err)
		}
	}
}
func TestCapabilityValidation(t *testing.T) {
	if got, err := NormalizeCapabilities(nil); err != nil || got != nil {
		t.Fatal(got, err)
	}
	r, c, caps := contextFixture()
	if _, err := ResolveContext(r, c, nil); err == nil {
		t.Fatal("unknown caps accepted")
	}
	for _, mutate := range []func(*CapabilityProfile){
		func(c *CapabilityProfile) { c.Classes = nil }, func(c *CapabilityProfile) { c.Handling = nil }, func(c *CapabilityProfile) { c.OriginalDestination = false },
	} {
		bad := *caps
		mutate(&bad)
		if _, err := ResolveContext(r, c, &bad); err == nil {
			t.Fatal("unsupported context")
		}
	}
	for _, mutate := range []func(*CapabilityProfile){func(c *CapabilityProfile) { c.Classes = []TrafficClass{"bad"} }, func(c *CapabilityProfile) { c.Protocols = []Protocol{"bad"} }, func(c *CapabilityProfile) { c.Families = []Family{99} }, func(c *CapabilityProfile) { c.Handling = []HandlingMode{"bad"} }} {
		bad := *caps
		mutate(&bad)
		if _, err := NormalizeCapabilities(&bad); err == nil {
			t.Fatal("unknown enum")
		}
	}
	caps.Protocols = []Protocol{UDP, TCP, TCP}
	caps.Families = []Family{IPv6, IPv4, IPv4}
	got, err := NormalizeCapabilities(caps)
	if err != nil || !reflect.DeepEqual(got.Protocols, []Protocol{TCP, UDP}) || !reflect.DeepEqual(got.Families, []Family{IPv4, IPv6}) {
		t.Fatal(got, err)
	}
	got.Protocols[0] = "bad"
	got.Families[0] = 99
	got.Classes[0] = "bad"
	got.Handling[0] = "bad"
	if caps.Protocols[0] != UDP || caps.Families[0] != IPv6 || caps.Classes[0] != HostLocal || caps.Handling[0] != Graceful {
		t.Fatal("caps alias")
	}
}
