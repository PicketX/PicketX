// SPDX-License-Identifier: Apache-2.0
package core_test

import (
	"fmt"
	"github.com/PicketX/PicketX/pkg/core"
	"net/netip"
	"time"
)

func ExampleCompile() {
	source, err := core.ParseSource("192.0.2.10")
	if err != nil {
		panic(err)
	}
	engine, err := core.Compile(core.Snapshot{
		Resources: []core.Resource{{Name: "gitea", Endpoints: []core.Endpoint{{Name: "https"}, {Name: "ssh"}}}},
		Grants:    []core.Grant{{ID: "browser-access", Source: source, Scopes: []core.Scope{{ResourceName: "gitea", Endpoints: core.EndpointSelection{Mode: core.SelectionNames, Names: []string{"https"}}}}, Validity: core.Validity{Mode: core.LifetimeBounded, ExpiresAt: time.Date(2026, 10, 15, 10, 0, 0, 0, time.UTC)}}},
	})
	if err != nil {
		panic(err)
	}
	d, err := engine.Decide(core.Query{Source: netip.MustParseAddr("192.0.2.10"), ResourceName: "gitea", EndpointName: "https", At: time.Date(2026, 10, 15, 9, 0, 0, 0, time.UTC)})
	if err != nil {
		panic(err)
	}
	fmt.Printf("allowed: %t\ngrants: %v\nnext change: %s\n", d.Allowed, d.GrantIDs, d.NextChange.Format(time.RFC3339))
	// Output:
	// allowed: true
	// grants: [browser-access]
	// next change: 2026-10-15T10:00:00Z
}
