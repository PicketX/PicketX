// SPDX-License-Identifier: Apache-2.0
package controller

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/PicketX/PicketX/internal/oidclogin"
)

func TestBrowserOIDCProtectedStartCorrelationAndSafeCallback(t *testing.T) {
	h, _, b := browserFixture(t)
	setter, ok := any(b).(interface{ SetOIDC(*oidclogin.Client) })
	if !ok {
		t.Fatal("OIDC browser adapter missing")
	}
	var issuer *httptest.Server
	issuer = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, `{"issuer":%q,"authorization_endpoint":%q,"token_endpoint":%q,"jwks_uri":%q,"id_token_signing_alg_values_supported":["RS256"]}`, issuer.URL, issuer.URL+"/authorize", issuer.URL+"/token", issuer.URL+"/keys")
	}))
	defer issuer.Close()
	dir := t.TempDir()
	bindings := filepath.Join(dir, "bindings.json")
	os.WriteFile(bindings, []byte(`{"bindings":[]}`), 0600)
	cfg := filepath.Join(dir, "provider.json")
	raw, _ := json.Marshal(map[string]any{"issuer": issuer.URL, "client_id": "client", "client_secret": "secret", "bindings": bindings})
	os.WriteFile(cfg, raw, 0600)
	client, e := oidclogin.New(context.Background(), cfg, "http://127.0.0.1:8080")
	if e != nil {
		t.Fatal(e)
	}
	setter.SetOIDC(client)
	h = NewHandlerWithBrowser(nil, b.service, ManagementLimits{1000, 1000}, b)
	c := &browserClient{h: h, cookies: map[string]*http.Cookie{}}
	c.bootstrap(t)
	if w := c.request("POST", "/console/api/v1/oidc/start", "", "http://evil.example", c.csrf); w.Code != 403 {
		t.Fatal("cross origin start", w.Code)
	}
	if w := c.request("POST", "/console/api/v1/oidc/start", "", "http://127.0.0.1:8080", ""); w.Code != 403 {
		t.Fatal("missing CSRF", w.Code)
	}
	if w := c.request("POST", "/console/api/v1/login", `{"token":"`+managementToken+`"}`, "http://127.0.0.1:8080", c.csrf); w.Code != 200 {
		t.Fatal("seed login", w.Code)
	}
	oldCookie := *c.cookies["picketx_session"]
	w := c.request("POST", "/console/api/v1/oidc/start", "", "http://127.0.0.1:8080", c.csrf)
	if w.Code != 200 {
		t.Fatal("start", w.Code, w.Body.String())
	}
	old := &browserClient{h: h, cookies: map[string]*http.Cookie{"picketx_session": &oldCookie}}
	if replay := old.request("GET", "/console/api/v1/me", "", "", ""); replay.Code != 401 {
		t.Fatal("OIDC replacement left originating session valid", replay.Code)
	}
	var body struct {
		URL string `json:"url"`
	}
	json.Unmarshal(w.Body.Bytes(), &body)
	auth, _ := url.Parse(body.URL)
	state := auth.Query().Get("state")
	cookie := c.cookies["picketx_oidc"]
	if cookie == nil || !cookie.HttpOnly || cookie.SameSite != http.SameSiteLaxMode || cookie.Path != "/console/api/v1/oidc/callback" || cookie.MaxAge != 300 {
		t.Fatal("unsafe correlation cookie", cookie)
	}
	if b.sessions.Cookie.SameSite != http.SameSiteStrictMode {
		t.Fatal("SCS weakened")
	}
	// All malformed callbacks clear correlation and disclose no provider diagnostic.
	for _, query := range []string{"state=" + state + "&state=duplicate&code=private-code", "state=" + state + "&code=one&code=two", "state=" + state + "&error=private-provider-error", "code=private-code", "state=unknown&code=private-code"} {
		w = c.request("GET", "/console/api/v1/oidc/callback?"+query, "", "", "")
		if w.Code != 303 || w.Header().Get("Location") != "/console/?login_error=oidc" || strings.Contains(w.Body.String(), "private") {
			t.Fatal("unsafe callback failure", w.Code, w.Header(), w.Body.String())
		}
	}
	if w = c.request("GET", "/console/api/v1/me", "", "", ""); w.Code != 401 {
		t.Fatal("callback created identity", w.Code)
	}
}
