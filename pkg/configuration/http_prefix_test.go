// SPDX-License-Identifier: Apache-2.0
package configuration

import (
	"encoding/json"
	"fmt"
	"net/netip"
	"strings"
	"testing"
	"time"

	"github.com/PicketX/PicketX/pkg/httpauth"
)

// Catches byte-prefix widening and prefix selectors bypassing source/time/deny.
func TestHTTPPathPrefixes(t *testing.T) {
	at := time.Date(2030, 1, 1, 0, 0, 0, 0, time.UTC)
	for _, selectors := range []string{"pathPrefixes: [/repo]", "paths: [/exact]\n  pathPrefixes: [/repo]"} {
		b, err := Decode([]byte(strings.Replace(httpPolicy, "paths: [/allowed]", selectors, 1)))
		if err != nil {
			t.Fatal(err)
		}
		c, err := Compile(b, Facts{AuthorityID: "test"})
		if err != nil {
			t.Fatal(err)
		}
		for _, tt := range []struct {
			uri, source, method, host string
			offset                    time.Duration
			allow                     bool
		}{
			{"/repo", "192.0.2.1", "GET", "app.example", 0, true},
			{"/repo/", "192.0.2.1", "GET", "app.example", 0, true},
			{"/repo/child?token=secret", "2001:db8::1", "GET", "app.example", 0, true},
			{"/repository", "192.0.2.1", "GET", "app.example", 0, false},
			{"/Repo/child", "192.0.2.1", "GET", "app.example", 0, false},
			{"/repo/child", "192.0.3.1", "GET", "app.example", 0, false},
			{"/repo/child", "192.0.2.1", "POST", "app.example", 0, false},
			{"/repo/child", "192.0.2.1", "GET", "other.example", 0, false},
			{"/repo/child", "192.0.2.1", "GET", "app.example", time.Hour, true},
			{"/repo/child", "192.0.2.2", "GET", "app.example", time.Hour, false},
			{"/repo/child", "192.0.2.1", "GET", "app.example", 2 * time.Hour, false},
			{"/exact", "192.0.2.1", "GET", "app.example", 0, strings.Contains(selectors, "paths:")},
			{"/exact/child", "192.0.2.1", "GET", "app.example", 0, false},
		} {
			q := httpauth.Query{Route: "route-v1", Source: netip.MustParseAddr(tt.source), Host: tt.host, Method: tt.method, URI: tt.uri, At: at.Add(tt.offset)}
			d, err := c.HTTP().Decide(q)
			if err != nil || d.Allowed != tt.allow {
				t.Fatalf("%+v: %+v %v", tt, d, err)
			}
		}
		for _, uri := range []string{"/repo/%2e%2e/private", "/repo/../private", "/repo//child", "/repo\\child"} {
			if _, err := c.HTTP().Decide(httpauth.Query{Route: "route-v1", Source: netip.MustParseAddr("192.0.2.1"), Host: "app.example", Method: "GET", URI: uri, At: at}); err == nil {
				t.Fatalf("accepted %q", uri)
			}
		}
		// Inventory mutation and unrelated source append must not lose or widen selectors.
		inventory := b.Inventory(Facts{})
		encoded, _ := json.Marshal(inventory.HTTPRoutes[0])
		if !strings.Contains(string(encoded), `"path_prefixes":["/repo"]`) {
			t.Fatal(string(encoded))
		}
		data, err := b.AddResource("extra", "extra-v1", []InventoryEndpoint{{Name: "ssh", Protocol: "TCP", Port: 22}})
		if err != nil {
			t.Fatal(err)
		}
		restored, err := Decode(data)
		if err != nil {
			t.Fatal(err)
		}
		if _, err = Compile(restored, Facts{AuthorityID: "test"}); err != nil {
			t.Fatal(err)
		}
	}
	for _, prefix := range []string{"/", "/repo"} {
		policy := strings.Replace(httpPolicy, "paths: [/allowed]", "pathPrefixes: ["+prefix+"]", 1)
		policy = strings.Replace(policy, "kind: Grant", "kind: NetworkDeny", 1)
		b, err := Decode([]byte(policy))
		if err != nil {
			t.Fatal(err)
		}
		c, err := Compile(b, Facts{AuthorityID: "test"})
		if err != nil {
			t.Fatal(err)
		}
		d, err := c.HTTP().Decide(httpauth.Query{Route: "route-v1", Source: netip.MustParseAddr("192.0.2.1"), Host: "app.example", Method: "GET", URI: "/repo/child", At: at.Add(time.Hour / 2)})
		if err != nil || d.Allowed || d.Reason != "explicit-deny" {
			t.Fatalf("deny: %+v %v", d, err)
		}
	}
	b, err := Decode([]byte(strings.Replace(httpPolicy, "paths: [/allowed]", "pathPrefixes: [/]", 1)))
	if err != nil {
		t.Fatal(err)
	}
	c, err := Compile(b, Facts{AuthorityID: "test"})
	if err != nil {
		t.Fatal(err)
	}
	for _, uri := range []string{"/", "/repo", "/unrelated/child"} {
		d, err := c.HTTP().Decide(httpauth.Query{Route: "route-v1", Source: netip.MustParseAddr("192.0.2.1"), Host: "app.example", Method: "GET", URI: uri, At: at})
		if err != nil || !d.Allowed {
			t.Fatalf("root %s: %+v %v", uri, d, err)
		}
	}
}

func TestHTTPPathPrefixesRejectInvalidSelectors(t *testing.T) {
	for _, selectors := range []string{"", "pathPrefixes: []", "pathPrefixes: null", "pathPrefixes: [/repo, /repo]", "pathPrefixes: [/repo/]", "pathPrefixes: [/repo?x=1]", "pathPrefixes: [/%72epo]", "pathPrefixes: [/repo/../x]", "pathPrefixes: [/repo]\n  paths: []", "paths: [/exact]\n  pathPrefixes: []", "paths: [/exact]\n  pathPrefixes: null"} {
		b, err := Decode([]byte(strings.Replace(httpPolicy, "paths: [/allowed]", selectors, 1)))
		if err == nil {
			_, err = Compile(b, Facts{AuthorityID: "test"})
		}
		if err == nil {
			t.Fatalf("accepted %q", selectors)
		}
	}
}

// Catches serialization turning explicit emptiness into valid omission.
func TestHTTPPathPrefixesAddAndClone(t *testing.T) {
	b, err := Decode([]byte(httpPolicy))
	if err != nil {
		t.Fatal(err)
	}
	in := InventoryHTTPRoute{Name: "prefix", UID: "prefix-v1", ResourceName: "app", ResourceUID: "app-v1", EndpointName: "web", Host: "app.example", Methods: []string{"GET"}, Paths: []string{}, PathPrefixes: []string{"/repo"}}
	if _, err := b.AddHTTPRoute(in); err == nil {
		t.Fatal("empty exact list erased during append")
	}
	in.Paths = nil
	data, err := b.AddHTTPRoute(in)
	if err != nil {
		t.Fatal(err)
	}
	b, err = Decode(data)
	if err != nil {
		t.Fatal(err)
	}
	view := b.Inventory(Facts{})
	for i := range view.HTTPRoutes {
		if view.HTTPRoutes[i].UID == "prefix-v1" {
			view.HTTPRoutes[i].PathPrefixes[0] = "/"
		}
	}
	for _, r := range b.Inventory(Facts{}).HTTPRoutes {
		if r.UID == "prefix-v1" && r.PathPrefixes[0] != "/repo" {
			t.Fatal("inventory aliases policy")
		}
	}
	c, err := Compile(b, Facts{AuthorityID: "test"})
	if err != nil {
		t.Fatal(err)
	}
	routes := []httpauth.Route{{ID: "cloned", Resource: "app-v1", Endpoint: "web", Host: "app.example", Methods: []string{"GET"}, PathPrefixes: []string{"/repo"}}}
	e, err := httpauth.Compile(c.Input().Logical, routes)
	if err != nil {
		t.Fatal(err)
	}
	routes[0].PathPrefixes[0] = "/"
	d, err := e.Decide(httpauth.Query{Route: "cloned", Source: netip.MustParseAddr("192.0.2.1"), Host: "app.example", Method: "GET", URI: "/other", At: time.Date(2030, 1, 1, 0, 0, 0, 0, time.UTC)})
	if err != nil || d.Allowed {
		t.Fatalf("compiled selector aliases input: %+v %v", d, err)
	}
	paths := make([]string, 128)
	for i := range paths {
		paths[i] = fmt.Sprintf("/p%d", i)
	}
	routes[0].Paths = paths
	routes[0].PathPrefixes = []string{"/repo"}
	if _, err := httpauth.Compile(c.Input().Logical, routes); err == nil {
		t.Fatal("combined selector limit bypass")
	}
}
