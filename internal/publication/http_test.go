// SPDX-License-Identifier: Apache-2.0
package publication

import (
	"context"
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
	for _, table := range []string{"controller_request_event", "controller_access_request"} {
		if e = s.db.Exec("DROP TABLE " + table).Error; e != nil {
			t.Fatal(e)
		}
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

// A request waiting for publication or SQLite has not admitted an application
// operation. A previously captured time cannot authorize after a hard boundary.
func TestHTTPDecisionRefreshesTimeAcrossQueueAndAudit(t *testing.T) {
	for _, boundary := range []string{"expiry", "deny-activation"} {
		for _, wait := range []string{"publication", "database"} {
			t.Run(boundary+"/"+wait, func(t *testing.T) {
				s, e := OpenMode(filepath.Join(t.TempDir(), "private", "state.sqlite"), "test", "database")
				if e != nil {
					t.Fatal(e)
				}
				defer s.Close()
				// Leave real compilation/fsync headroom under full race/build load;
				// the assertion still requires a pre-boundary query and an actual
				// blocked mutex/SQLite connection spanning the absolute boundary.
				deadline := time.Now().UTC().Add(2 * time.Second)
				policy := httpPolicy
				if boundary == "expiry" {
					policy = strings.Replace(policy, "validity: {mode: permanent}", "validity: {mode: bounded, expiresAt: '"+deadline.Format(time.RFC3339Nano)+"'}", 1)
				} else {
					policy += `---
apiVersion: picketx.io/v1alpha1
kind: NetworkDeny
metadata: {name: timed-deny, uid: timed-deny-v1}
spec:
  subject: {type: source, cidr: 127.0.0.1}
  scopes: [{resourceRef: {name: app, uid: app-v1}}]
  validity: {mode: bounded, notBefore: '` + deadline.Format(time.RFC3339Nano) + `', expiresAt: '` + deadline.Add(time.Hour).Format(time.RFC3339Nano) + `'}
`
				}
				if _, e = s.AcceptManaged(0, "admin", "deadline", []byte(policy), []byte(`{"authorityID":"test","agents":[],"contexts":[]}`), nil); e != nil {
					t.Fatal(e)
				}
				q := httpauth.Query{Route: "route-v1", Source: netip.MustParseAddr("127.0.0.1"), Host: "app.example", Method: "GET", URI: "/allowed", At: time.Now().UTC()}
				if !q.At.Before(deadline) {
					t.Fatal("fixture exhausted deadline before queue")
				}
				release := func() {}
				db, _ := s.db.DB()
				before := db.Stats().WaitCount
				if wait == "publication" {
					s.mu.Lock()
					release = s.mu.Unlock
				} else {
					conn, e := db.Conn(context.Background())
					if e != nil {
						t.Fatal(e)
					}
					release = func() { conn.Close() }
				}
				var got HTTPDecision
				var resultErr error
				done := make(chan struct{})
				go func() { defer close(done); got, resultErr = s.DecideHTTP("nginx", q) }()
				if wait == "database" {
					until := time.Now().Add(time.Second)
					for db.Stats().WaitCount == before && time.Now().Before(until) {
						time.Sleep(time.Millisecond)
					}
					if db.Stats().WaitCount == before {
						release()
						<-done
						t.Fatal("did not block on actual SQLite connection")
					}
				}
				if d := time.Until(deadline); d > 0 {
					<-time.After(d)
				}
				release()
				select {
				case <-done:
				case <-time.After(time.Second):
					t.Fatal("decision stalled")
				}
				if resultErr != nil || got.Allowed {
					t.Fatalf("returned stale boundary decision: %+v %v", got, resultErr)
				}
				if boundary == "deny-activation" && got.Reason != "explicit-deny" {
					t.Fatalf("deny activation ignored: %+v", got)
				}
			})
		}
	}
}
