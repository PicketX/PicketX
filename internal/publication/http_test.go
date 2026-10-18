// SPDX-License-Identifier: Apache-2.0
package publication

import (
	"github.com/PicketX/PicketX/pkg/httpauth"
	"net/netip"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

const httpPolicy = `apiVersion: picketx.io/v1alpha1
kind: Resource
metadata: {name: app, uid: app-v1}
spec: {endpoints: [{name: web, network: {protocol: TCP, port: 443}}]}
---
apiVersion: picketx.io/v1alpha1
kind: Grant
metadata: {name: test, uid: test-v1}
spec:
  subject: {type: source, cidr: 127.0.0.1}
  scopes: [{resourceRef: {name: app, uid: app-v1}}]
  validity: {mode: permanent}
---
apiVersion: picketx.io/v1alpha1
kind: HTTPRoute
metadata: {name: app, uid: route-v1}
spec: {resourceRef: {name: app, uid: app-v1}, endpointName: web, host: app.example, methods: [GET], paths: [/allowed]}
`

func TestHTTPDecisionDurableAuditFailureAndUpgrade(t *testing.T) {
	path := filepath.Join(t.TempDir(), "state", "state.sqlite")
	s, e := OpenMode(path, "test", "database")
	if e != nil {
		t.Fatal(e)
	}
	if _, e = s.AcceptManaged(0, "admin", "HTTP", []byte(httpPolicy), []byte(`{"authorityID":"test","agents":[],"contexts":[]}`), nil); e != nil {
		t.Fatal(e)
	}
	q := httpauth.Query{Route: "route-v1", Source: netip.MustParseAddr("127.0.0.1"), Host: "app.example", Method: "GET", URI: "/allowed?secret=excluded", At: time.Now().UTC()}
	d, e := s.DecideHTTP("nginx", q)
	if e != nil || !d.Allowed || d.Revision != 1 {
		t.Fatalf("%+v %v", d, e)
	}
	events, e := s.HTTPAudit(0, 1)
	if e != nil || len(events) != 1 || events[0].Caller != "nginx" || events[0].Route != "route-v1" || events[0].GrantIDs[0] != "test-v1" {
		t.Fatalf("%+v %v", events, e)
	}
	if e = s.Close(); e != nil {
		t.Fatal(e)
	}
	s, e = OpenMode(path, "test", "database")
	if e != nil {
		t.Fatal(e)
	}
	defer s.Close()
	events, e = s.HTTPAudit(0, 100)
	if e != nil || len(events) != 1 {
		t.Fatal(events, e)
	}
	if _, e = s.AcceptManaged(1, "admin", "revoke", []byte(strings.Replace(httpPolicy, "subject:", "revoked: true\n  subject:", 1)), []byte(`{"authorityID":"test","agents":[],"contexts":[]}`), nil); e != nil {
		t.Fatal(e)
	}
	d, e = s.DecideHTTP("nginx", q)
	if e != nil || d.Allowed || d.Revision != 2 {
		t.Fatal(d, e)
	}
	if e = s.db.Exec("CREATE TRIGGER fail_http BEFORE INSERT ON controller_http_audit BEGIN SELECT RAISE(ABORT,'fault'); END").Error; e != nil {
		t.Fatal(e)
	}
	if d, e = s.DecideHTTP("nginx", q); e == nil || d.Allowed {
		t.Fatal("audit failure returned decision", d, e)
	}
	if e = s.db.Exec("DROP TABLE controller_http_audit").Error; e != nil {
		t.Fatal(e)
	}
	if e = s.db.Exec("UPDATE controller_schema SET version=2").Error; e != nil {
		t.Fatal(e)
	}
	s.Close()
	s, e = OpenMode(path, "test", "database")
	if e != nil {
		t.Fatal(e)
	}
	defer s.Close()
	if _, e = s.DecideHTTP("nginx", q); e != nil {
		t.Fatal("schema2 upgrade", e)
	}
}
