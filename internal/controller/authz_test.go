// SPDX-License-Identifier: Apache-2.0
package controller

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"github.com/PicketX/PicketX/internal/authz"
	"github.com/PicketX/PicketX/internal/publication"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const callerToken = "separate-ingress-authentication-token-12345"

func TestScopedHTTPAdaptersAndCredentialRevocation(t *testing.T) {
	s, e := publication.OpenMode(filepath.Join(t.TempDir(), "state", "state.sqlite"), "test", "database")
	if e != nil {
		t.Fatal(e)
	}
	defer s.Close()
	policy := `apiVersion: picketx.io/v1alpha1
kind: Resource
metadata: {name: app, uid: app-v1}
spec: {endpoints: [{name: web, network: {protocol: TCP, port: 443}}]}
---
apiVersion: picketx.io/v1alpha1
kind: Grant
metadata: {name: grant, uid: grant-v1}
spec: {subject: {type: source, cidr: 127.0.0.1}, scopes: [{resourceRef: {name: app, uid: app-v1}}], validity: {mode: permanent}}
---
apiVersion: picketx.io/v1alpha1
kind: HTTPRoute
metadata: {name: app, uid: route-v1}
spec: {resourceRef: {name: app, uid: app-v1}, endpointName: web, host: app.example, methods: [GET], paths: [/allowed]}
`
	if _, e = s.AcceptManaged(0, "admin", "route", []byte(policy), []byte(`{"authorityID":"test","agents":[],"contexts":[]}`), nil); e != nil {
		t.Fatal(e)
	}
	path := filepath.Join(t.TempDir(), "callers.json")
	sum := sha256.Sum256([]byte(callerToken))
	data, _ := json.Marshal(map[string]any{"callers": []map[string]any{{"id": "nginx", "sha256": hex.EncodeToString(sum[:]), "routes": []string{"route-v1"}}}})
	if e = os.WriteFile(path, data, 0600); e != nil {
		t.Fatal(e)
	}
	service, e := authz.New(path, s)
	if e != nil {
		t.Fatal(e)
	}
	h := NewAuthorizationHandler(service)
	body := `{"route":"route-v1","source":"127.0.0.1","host":"app.example","method":"GET","uri":"/allowed?secret=absent"}`
	call := func(method, path, body, token string, extra http.Header) *httptest.ResponseRecorder {
		r := httptest.NewRequest(method, path, strings.NewReader(body))
		r.Header.Set("Content-Type", "application/json")
		r.Header.Set("Authorization", "Bearer "+token)
		for k, v := range extra {
			r.Header[k] = v
		}
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)
		if w.Header().Get("Cache-Control") != "no-store" {
			t.Fatal("cacheable decision")
		}
		return w
	}
	for _, tt := range []struct {
		body, token string
		code        int
	}{
		{body, callerToken, 200}, {body, "management-token-not-ingress-123456", 401}, {strings.Replace(body, "route-v1", "other-route", 1), callerToken, 403},
		{strings.Replace(body, "app.example", "evil.example", 1), callerToken, 200}, {strings.Replace(body, "127.0.0.1", "::ffff:127.0.0.1", 1), callerToken, 400},
		{strings.Replace(body, "\"uri\":", "\"time\":\"2030-01-01\",\"uri\":", 1), callerToken, 400}, {strings.Replace(body, "\"method\":\"GET\"", "\"method\":\"GET\",\"method\":\"GET\"", 1), callerToken, 400},
	} {
		w := call("POST", "/authz/v1/decisions", tt.body, tt.token, nil)
		if w.Code != tt.code {
			t.Fatalf("%d %s", w.Code, w.Body)
		}
	}
	headers := http.Header{"X-Picketx-Route": {"route-v1"}, "X-Picketx-Source": {"127.0.0.1"}, "X-Picketx-Host": {"app.example"}, "X-Picketx-Method": {"GET"}, "X-Picketx-Uri": {"/allowed"}}
	if w := call("GET", "/authz/v1/nginx", "", callerToken, headers); w.Code != 204 {
		t.Fatal(w.Code, w.Body)
	}
	headers["X-Picketx-Route"] = []string{"route-v1", "route-v1"}
	if w := call("GET", "/authz/v1/nginx", "", callerToken, headers); w.Code != 400 {
		t.Fatal(w.Code, w.Body)
	}
	events, e := s.HTTPAudit(0, 100)
	if e != nil || len(events) != 3 {
		t.Fatal(events, e)
	}
	raw, _ := json.Marshal(events)
	if bytes.Contains(raw, []byte("secret")) || bytes.Contains(raw, []byte("app.example")) {
		t.Fatal("raw request recorded")
	}
	if e = s.GORM().Exec("CREATE TRIGGER fail_http BEFORE INSERT ON controller_http_audit BEGIN SELECT RAISE(ABORT,'fault'); END").Error; e != nil {
		t.Fatal(e)
	}
	if w := call("POST", "/authz/v1/decisions", body, callerToken, nil); w.Code != 503 || bytes.Contains(w.Body.Bytes(), []byte(`"allowed":true`)) {
		t.Fatal(w.Code, w.Body)
	}
	if e = os.WriteFile(path, []byte(`{"callers":[]}`), 0600); e != nil {
		t.Fatal(e)
	}
	if w := call("POST", "/authz/v1/decisions", body, callerToken, nil); w.Code != 401 {
		t.Fatal(w.Code, w.Body)
	}
	if e = os.WriteFile(path, []byte(`{"callers":null}`), 0600); e != nil {
		t.Fatal(e)
	}
	if w := call("POST", "/authz/v1/decisions", body, callerToken, nil); w.Code != 503 {
		t.Fatal(w.Code, w.Body)
	}
}
