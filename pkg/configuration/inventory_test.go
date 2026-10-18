// SPDX-License-Identifier: Apache-2.0
package configuration

import (
	"bytes"
	"net/netip"
	"os"
	"strings"
	"testing"

	"github.com/PicketX/PicketX/pkg/networkplan"
)

func TestInventoryAdditionsPreserveSourceAndContextIdentity(t *testing.T) {
	raw, e := os.ReadFile("../../examples/static/gitea.yaml")
	if e != nil {
		t.Fatal(e)
	}
	raw = []byte(strings.Replace(string(raw), "metadata: {name: gitea, uid: resource-gitea-v1}", "metadata: {name: gitea, uid: resource-gitea-v1, annotations: {note: '"+strings.Repeat("<", 800000)+"'}}", 1))
	b, e := Decode(raw)
	if e != nil {
		t.Fatal(e)
	}
	added, e := b.AddResource("metrics", "resource-metrics", []InventoryEndpoint{{Name: "web", Protocol: "TCP", Port: 8443}})
	if e != nil || !bytes.HasPrefix(added, raw) || len(added) > len(raw)+1000 {
		t.Fatal("source expanded", e)
	}
	b, e = Decode(added)
	if e != nil {
		t.Fatal(e)
	}
	f := Facts{Contexts: []networkplan.ExecutionContext{
		{AgentRef: "other", ID: "same", ProfileRef: "wrong", Class: networkplan.HostLocal, Stage: networkplan.OriginalDestination, Handling: networkplan.Strict, Destinations: []netip.Prefix{netip.MustParsePrefix("198.51.100.1/32")}},
		{AgentRef: "wanted", ID: "same", ProfileRef: "right", Class: networkplan.HostLocal, Stage: networkplan.OriginalDestination, Handling: networkplan.Graceful, Destinations: []netip.Prefix{netip.MustParsePrefix("2001:db8::1/128")}},
	}}
	policy, e := b.AddBinding(BindingInput{Name: "metrics-host", UID: "binding-metrics", ResourceName: "metrics", ResourceUID: "resource-metrics", EndpointNames: []string{"web"}, AgentName: "wanted", ContextID: "same"}, f)
	if e != nil {
		t.Fatal(e)
	}
	out, e := Decode(policy)
	if e != nil {
		t.Fatal(e)
	}
	inventory := out.Inventory(f)
	var binding InventoryBinding
	for _, v := range inventory.Bindings {
		if v.UID == "binding-metrics" {
			binding = v
		}
	}
	if !strings.Contains(binding.Enforcement, `"profileRef":"right"`) || strings.Contains(binding.Enforcement, "wrong") || !strings.Contains(binding.Enforcement, "2001:db8::1/128") || !strings.Contains(binding.Agents, `"wanted"`) {
		t.Fatal(binding)
	}
	inventory.Resources[0].Endpoints[0].Name = "changed"
	again := out.Inventory(f)
	if again.Resources[0].Endpoints[0].Name == "changed" {
		t.Fatal("mutable inventory")
	}
}

func TestInventoryBindingCannotChooseTransitOrAmbiguousContext(t *testing.T) {
	raw, e := os.ReadFile("../../examples/static/gitea.yaml")
	if e != nil {
		t.Fatal(e)
	}
	b, e := Decode(raw)
	if e != nil {
		t.Fatal(e)
	}
	input := BindingInput{Name: "new", UID: "binding-new", ResourceName: "gitea", ResourceUID: "resource-gitea-v1", EndpointNames: []string{"https"}, AgentName: "agent-a", ContextID: "host-v1"}
	c := networkplan.ExecutionContext{AgentRef: "agent-a", ID: "host-v1", ProfileRef: "host", Class: networkplan.Transit, Stage: networkplan.OriginalDestination}
	if _, e = b.AddBinding(input, Facts{Contexts: []networkplan.ExecutionContext{c}}); e == nil {
		t.Fatal("transit accepted")
	}
	c.Class = networkplan.HostLocal
	if _, e = b.AddBinding(input, Facts{Contexts: []networkplan.ExecutionContext{c, c}}); e == nil {
		t.Fatal("ambiguous context accepted")
	}
}
