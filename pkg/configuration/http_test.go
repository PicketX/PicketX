// SPDX-License-Identifier: Apache-2.0
package configuration

import (
	"net/netip"
	"strings"
	"testing"
	"time"

	"github.com/PicketX/PicketX/pkg/httpauth"
)

const httpPolicy = `apiVersion: picketx.io/v1alpha1
kind: Resource
metadata: {name: app, uid: app-v1}
spec:
  endpoints: [{name: web, network: {protocol: TCP, port: 443}}]
---
apiVersion: picketx.io/v1alpha1
kind: Grant
metadata: {name: first, uid: first-v1}
spec:
  subject: {type: source, cidr: 192.0.2.0/24}
  scopes: [{resourceRef: {name: app, uid: app-v1}, endpointNames: [web]}]
  validity: {mode: bounded, expiresAt: '2030-01-01T01:00:00Z'}
---
apiVersion: picketx.io/v1alpha1
kind: Grant
metadata: {name: second, uid: second-v1}
spec:
  subject: {type: source, cidr: 192.0.2.1/32}
  scopes: [{resourceRef: {name: app, uid: app-v1}, endpointNames: [web]}]
  validity: {mode: bounded, notBefore: '2030-01-01T00:30:00Z', expiresAt: '2030-01-01T02:00:00Z'}
---
apiVersion: picketx.io/v1alpha1
kind: Grant
metadata: {name: ipv6, uid: ipv6-v1}
spec:
  subject: {type: source, cidr: '2001:db8::/64'}
  scopes: [{resourceRef: {name: app, uid: app-v1}, endpointNames: [web]}]
  validity: {mode: permanent}
---
apiVersion: picketx.io/v1alpha1
kind: HTTPRoute
metadata: {name: app-web, uid: route-v1}
spec:
  resourceRef: {name: app, uid: app-v1}
  endpointName: web
  host: app.example
  methods: [GET]
  paths: [/allowed]
`

// A missing route constraint or copied connection-level allow would fail these real decisions.
func TestHTTPRouteSourceAndRequestConstraints(t *testing.T) {
	b, e := Decode([]byte(httpPolicy))
	if e != nil {
		t.Fatal(e)
	}
	c, e := Compile(b, Facts{AuthorityID: "test"})
	if e != nil {
		t.Fatal(e)
	}
	at := time.Date(2030, 1, 1, 0, 0, 0, 0, time.UTC)
	for _, tt := range []struct {
		name, source, host, method, uri, route string
		offset                                 time.Duration
		allowed                                bool
		reason                                 string
	}{
		{"allow", "192.0.2.1", "app.example", "GET", "/allowed?secret=never-record", "route-v1", 0, true, "allowed"},
		{"ipv6", "2001:db8::1", "app.example", "GET", "/allowed", "route-v1", 0, true, "allowed"},
		{"other-source", "192.0.3.1", "app.example", "GET", "/allowed", "route-v1", 0, false, "no-active-grant"},
		{"other-host", "192.0.2.1", "other.example", "GET", "/allowed", "route-v1", 0, false, "route-mismatch"},
		{"other-method", "192.0.2.1", "app.example", "POST", "/allowed", "route-v1", 0, false, "route-mismatch"},
		{"no-head-default", "192.0.2.1", "app.example", "HEAD", "/allowed", "route-v1", 0, false, "route-mismatch"},
		{"other-path", "192.0.2.1", "app.example", "GET", "/allowed/child", "route-v1", 0, false, "route-mismatch"},
		{"unknown", "192.0.2.1", "app.example", "GET", "/allowed", "other", 0, false, "unknown-route"},
		{"independent", "192.0.2.1", "app.example", "GET", "/allowed", "route-v1", time.Hour, true, "allowed"},
		{"expired-first", "192.0.2.2", "app.example", "GET", "/allowed", "route-v1", time.Hour, false, "no-active-grant"},
		{"all-expired", "192.0.2.1", "app.example", "GET", "/allowed", "route-v1", 2 * time.Hour, false, "no-active-grant"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			d, e := c.HTTP().Decide(httpauth.Query{Route: tt.route, Source: netip.MustParseAddr(tt.source), Host: tt.host, Method: tt.method, URI: tt.uri, At: at.Add(tt.offset)})
			if e != nil || d.Allowed != tt.allowed || string(d.Reason) != tt.reason {
				t.Fatalf("%+v %v", d, e)
			}
		})
	}
	for _, change := range []string{strings.Replace(httpPolicy, "kind: Grant", "kind: NetworkDeny", 1)} {
		b, e = Decode([]byte(change))
		if e != nil {
			t.Fatal(e)
		}
		c, e = Compile(b, Facts{AuthorityID: "test"})
		if e != nil {
			t.Fatal(e)
		}
		d, e := c.HTTP().Decide(httpauth.Query{Route: "route-v1", Source: netip.MustParseAddr("192.0.2.1"), Host: "app.example", Method: "GET", URI: "/allowed", At: at.Add(30 * time.Minute)})
		if e != nil || d.Allowed || d.Reason != "explicit-deny" {
			t.Fatalf("deny not dominant: %+v %v", d, e)
		}
	}
	b, _ = Decode([]byte(strings.Replace(httpPolicy, "subject: {type: source, cidr: 192.0.2.0/24}", "revoked: true\n  subject: {type: source, cidr: 192.0.2.0/24}", 1)))
	c, e = Compile(b, Facts{AuthorityID: "test"})
	if e != nil {
		t.Fatal(e)
	}
	q := httpauth.Query{Route: "route-v1", Source: netip.MustParseAddr("192.0.2.1"), Host: "app.example", Method: "GET", URI: "/allowed", At: at}
	d, e := c.HTTP().Decide(q)
	if e != nil || d.Allowed || !d.NextChange.Equal(at.Add(30*time.Minute)) {
		t.Fatalf("revoked/future contribution: %+v %v", d, e)
	}
	q.At = at.Add(30 * time.Minute)
	d, e = c.HTTP().Decide(q)
	if e != nil || !d.Allowed || len(d.GrantIDs) != 1 || d.GrantIDs[0] != "second-v1" {
		t.Fatalf("future activation: %+v %v", d, e)
	}
}

func TestHTTPRouteRejectsInvalidCompleteCandidateAndRetarget(t *testing.T) {
	for _, tt := range []struct{ old, new string }{
		{"methods: [GET]", "methods: []"}, {"paths: [/allowed]", "paths: []"}, {"paths: [/allowed]", "paths: [/%61llowed]"}, {"paths: [/allowed]", "paths: [/a/../allowed]"},
		{"host: app.example", "host: APP.example"}, {"endpointName: web", "endpointName: missing"}, {"protocol: TCP", "protocol: UDP"}, {"endpointName: web", "endpointName: web\n  verifiedUser: admin"},
	} {
		t.Run(tt.new, func(t *testing.T) {
			b, e := Decode([]byte(strings.Replace(httpPolicy, tt.old, tt.new, 1)))
			if e == nil {
				_, e = Compile(b, Facts{AuthorityID: "test"})
			}
			if e == nil {
				t.Fatal("accepted invalid route")
			}
		})
	}
	b, _ := Decode([]byte(httpPolicy))
	c, e := Compile(b, Facts{AuthorityID: "test"})
	if e != nil {
		t.Fatal(e)
	}
	b, _ = Decode([]byte(strings.Replace(httpPolicy, "host: app.example", "host: other.example", 1)))
	if _, _, e = Replace(c, b, Facts{AuthorityID: "test"}); e == nil {
		t.Fatal("route UID silently retargeted")
	}
}

func TestHTTPContextRejectsAmbiguousRequest(t *testing.T) {
	b, _ := Decode([]byte(httpPolicy))
	c, e := Compile(b, Facts{AuthorityID: "test"})
	if e != nil {
		t.Fatal(e)
	}
	for _, uri := range []string{"", "//allowed", "/%61llowed", "/a/../allowed", "/a/./allowed", "/allowed#fragment", "/a\\b", "http://app.example/allowed", "/allowed\n", "/中文"} {
		q := httpauth.Query{Route: "route-v1", Source: netip.MustParseAddr("192.0.2.1"), Host: "app.example", Method: "GET", URI: uri, At: time.Now()}
		if _, e := c.HTTP().Decide(q); e == nil {
			t.Fatalf("accepted %q", uri)
		}
	}
}
