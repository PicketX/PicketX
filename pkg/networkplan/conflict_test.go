// SPDX-License-Identifier: Apache-2.0
package networkplan

import (
	"github.com/PicketX/PicketX/pkg/core"
	"reflect"
	"testing"
)

func TestPhysicalConflict(t *testing.T) {
	in := baseInput()
	in.Logical.Grants = nil
	in.Networks[1].Port = 443
	if p, e := Compile(in); e == nil || p != nil {
		t.Fatal("same-resource alias", p, e)
	}
	// Distinct logical Resources cannot hide overlap either.
	in = baseInput()
	in.Logical.Grants = nil
	in.Logical.Resources = append(in.Logical.Resources, core.Resource{Name: "other", Endpoints: []core.Endpoint{{Name: "https"}}})
	in.Networks = append(in.Networks, NetworkDefinition{Entry: EntryRef{"other", "https"}, Protocol: TCP, Port: 443, Addresses: prefixes("192.0.2.128/25")})
	in.Bindings[0].Resources = append(in.Bindings[0].Resources, ResourceSelection{ResourceName: "other"})
	if p, e := Compile(in); e == nil || p != nil {
		t.Fatal("other resource alias", p, e)
	}
	for _, mode := range []string{"disjoint", "protocol", "port", "agent"} {
		t.Run(mode, func(t *testing.T) {
			i := baseInput()
			i.Networks[1].Port = 443
			switch mode {
			case "disjoint":
				i.Networks[0].Addresses = prefixes("192.0.2.0/25")
				i.Networks[1].Addresses = prefixes("192.0.2.128/25")
			case "protocol":
				i.Networks[1].Protocol = UDP
			case "port":
				i.Networks[1].Port = 444
			case "agent":
				i.Bindings[0].Resources[0].Endpoints = core.EndpointSelection{Mode: core.SelectionNames, Names: []string{"https"}}
				b := i.Bindings[0]
				b.ID = "b2"
				b.Resources = []ResourceSelection{{ResourceName: "gitea", Endpoints: core.EndpointSelection{Mode: core.SelectionNames, Names: []string{"ssh"}}}}
				b.Agents = []AgentSelection{{AgentRef: "agent-b"}}
				i.Bindings = append(i.Bindings, b)
			}
			planOK(t, i)
		})
	}
	for _, strict := range []bool{false, true} {
		i := baseInput()
		c := i.Contexts[0]
		c.ID = "ctx-new"
		c.ProfileRef = "alternate"
		if strict {
			c.Handling = Strict
		}
		i.Contexts = append(i.Contexts, c)
		b := i.Bindings[0]
		b.ID = "b2"
		b.Context.ProfileRef = "alternate"
		if strict {
			b.Context.Handling = Strict
		}
		i.Bindings = append(i.Bindings, b)
		if p, e := Compile(i); e == nil || p != nil {
			t.Fatal("IDs/handling hid overlap", p, e)
		}
	}
}
func TestBindingProvenanceMerge(t *testing.T) {
	in := baseInput()
	b := in.Bindings[0]
	b.ID = "b2"
	in.Bindings = []Binding{b, in.Bindings[0]}
	p := planOK(t, in)
	v := inventoryOK(t, p, "agent-a")
	if len(v) != 2 {
		t.Fatal(v)
	}
	for _, target := range v {
		if !reflect.DeepEqual(target.Protection.BindingIDs, []string{"b1", "b2"}) || len(target.Grants) != 2 {
			t.Fatal(target)
		}
	}
}
