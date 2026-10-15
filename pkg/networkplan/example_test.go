// SPDX-License-Identifier: Apache-2.0
package networkplan_test

import (
	"fmt"
	"github.com/PicketX/PicketX/pkg/core"
	"github.com/PicketX/PicketX/pkg/networkplan"
	"net/netip"
	"time"
)

func ExampleCompile() {
	destinations := []netip.Prefix{netip.MustParsePrefix("192.0.2.100/32")}
	input := networkplan.Input{AuthorityID: "static-example",
		Logical:  core.Snapshot{Resources: []core.Resource{{Name: "gitea", Endpoints: []core.Endpoint{{Name: "https"}}}}, Grants: []core.Grant{{ID: "approved-1", Source: netip.MustParsePrefix("198.51.100.9/32"), Scopes: []core.Scope{{ResourceName: "gitea"}}, Validity: core.Validity{Mode: core.LifetimeBounded, ExpiresAt: time.Date(2026, 10, 15, 10, 0, 0, 0, time.UTC)}}}},
		Networks: []networkplan.NetworkDefinition{{Entry: networkplan.EntryRef{ResourceName: "gitea", EndpointName: "https"}, Protocol: networkplan.TCP, Port: 443}},
		Agents:   []networkplan.Agent{{Name: "agent-a", Capabilities: &networkplan.CapabilityProfile{Classes: []networkplan.TrafficClass{networkplan.HostLocal}, Protocols: []networkplan.Protocol{networkplan.TCP}, Families: []networkplan.Family{networkplan.IPv4}, Handling: []networkplan.HandlingMode{networkplan.Graceful}, OriginalDestination: true}}},
		Contexts: []networkplan.ExecutionContext{{AgentRef: "agent-a", ProfileRef: "host", ID: "host-v1", NetworkSpaceID: "trusted-host-space", Class: networkplan.HostLocal, Stage: networkplan.OriginalDestination, Destinations: destinations, Handling: networkplan.Graceful}},
		Bindings: []networkplan.Binding{{ID: "deploy-gitea", Resources: []networkplan.ResourceSelection{{ResourceName: "gitea"}}, Agents: []networkplan.AgentSelection{{AgentRef: "agent-a"}}, Context: networkplan.ContextRequest{ProfileRef: "host", Class: networkplan.HostLocal, Stage: networkplan.OriginalDestination, AllowedDestinations: destinations, Handling: networkplan.Graceful}}},
	}
	plan, err := networkplan.Compile(input)
	if err != nil {
		panic(err)
	}
	projection, err := plan.Projection("agent-a", time.Date(2026, 10, 15, 9, 0, 0, 0, time.UTC))
	if err != nil {
		panic(err)
	}
	for _, p := range projection.Protected {
		for _, m := range p.Matches {
			fmt.Printf("%s %s/%s %s/%d %s\n", p.Key.AgentRef, p.Key.Entry.ResourceName, p.Key.Entry.EndpointName, m.Protocol, m.Port, m.Destination)
		}
	}
	// Output:
	// agent-a gitea/https tcp/443 192.0.2.100/32
}
