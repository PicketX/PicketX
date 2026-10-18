// SPDX-License-Identifier: Apache-2.0
package controller

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/PicketX/PicketX/internal/management"
	"github.com/PicketX/PicketX/internal/managementaccess"
)

func browserFixture(t *testing.T) (http.Handler, string, *Browser) {
	t.Helper()
	_, s, creds := managementFixture(t)
	a, e := managementaccess.New(s.GORM(), managementaccess.Settings{Credentials: creds, Reload: time.Second, WarnRules: 10000, WarnLoadPercent: 50}, slog.New(slog.NewTextHandler(io.Discard, nil)))
	if e != nil {
		t.Fatal(e)
	}
	service := management.New(a, directRepository{s})
	b, e := NewBrowser(service, "http://127.0.0.1:8080", "")
	if e != nil {
		t.Fatal(e)
	}
	return NewHandlerWithBrowser(nil, service, ManagementLimits{1000, 1000}, b), creds, b
}

type browserClient struct {
	h       http.Handler
	cookies map[string]*http.Cookie
	csrf    string
}

func (b *browserClient) request(method, path, body, origin, csrf string) *httptest.ResponseRecorder {
	r := httptest.NewRequest(method, "http://127.0.0.1:8080"+path, strings.NewReader(body))
	r.Header.Set("Content-Type", "application/json")
	if origin != "" {
		r.Header.Set("Origin", origin)
	}
	if csrf != "" {
		r.Header.Set("X-CSRF-Token", csrf)
	}
	for _, c := range b.cookies {
		r.AddCookie(c)
	}
	w := httptest.NewRecorder()
	b.h.ServeHTTP(w, r)
	for _, c := range w.Result().Cookies() {
		b.cookies[c.Name] = c
	}
	return w
}
func (b *browserClient) bootstrap(t *testing.T) {
	t.Helper()
	w := b.request("GET", "/console/api/v1/session", "", "", "")
	var body struct {
		CSRF string `json:"csrf"`
	}
	if w.Code != 200 || json.Unmarshal(w.Body.Bytes(), &body) != nil || body.CSRF == "" {
		t.Fatal(w.Code, w.Body.String())
	}
	b.csrf = body.CSRF
}
func TestBrowserSessionCSRFIsolationRevocationAndLogout(t *testing.T) {
	h, creds, _ := browserFixture(t)
	b := browserClient{h: h, cookies: map[string]*http.Cookie{}}
	b.bootstrap(t)
	login := `{"token":"` + managementToken + `"}`
	for _, v := range [][2]string{{"", b.csrf}, {"http://evil.test", b.csrf}, {"http://127.0.0.1:8080", ""}} {
		if w := b.request("POST", "/console/api/v1/login", login, v[0], v[1]); w.Code != 403 {
			t.Fatal("login CSRF", w.Code, w.Body.String())
		}
	}
	w := b.request("POST", "/console/api/v1/login", login, "http://127.0.0.1:8080", b.csrf)
	if w.Code != 200 {
		t.Fatal(w.Code, w.Body.String())
	}
	c := b.cookies["picketx_session"]
	if c == nil || !c.HttpOnly || c.SameSite != http.SameSiteStrictMode || c.Path != "/console/" || c.Secure {
		t.Fatal(c)
	}
	old := *c
	if w = b.request("GET", "/console/api/v1/me", "", "", ""); w.Code != 200 {
		t.Fatal(w.Code, w.Body.String())
	}
	if w = b.request("GET", "/api/v1/me", "", "", ""); w.Code != 401 {
		t.Fatal("cookie authenticated bearer API", w.Code)
	}
	r := httptest.NewRequest("GET", "http://evil.test/console/api/v1/session", nil)
	r.Header.Set("X-Forwarded-Host", "127.0.0.1:8080")
	out := httptest.NewRecorder()
	h.ServeHTTP(out, r)
	if out.Code != 403 {
		t.Fatal("untrusted Host", out.Code)
	}
	r = httptest.NewRequest("GET", "http://127.0.0.1:8080/console/api/v1/me", nil)
	r.Header.Set("Authorization", "Bearer "+managementToken)
	out = httptest.NewRecorder()
	h.ServeHTTP(out, r)
	if out.Code != 400 {
		t.Fatal("mixed authentication", out.Code)
	}
	if w = b.request("POST", "/console/api/v1/logout", `{}`, "http://127.0.0.1:8080", b.csrf); w.Code != 204 {
		t.Fatal(w.Code, w.Body.String())
	}
	b.cookies[old.Name] = &old
	if w = b.request("GET", "/console/api/v1/me", "", "", ""); w.Code != 401 {
		t.Fatal("logout session replay", w.Code)
	}
	b.cookies = map[string]*http.Cookie{}
	b.bootstrap(t)
	if w = b.request("POST", "/console/api/v1/login", login, "http://127.0.0.1:8080", b.csrf); w.Code != 200 {
		t.Fatal(w.Code)
	}
	data, _ := os.ReadFile(creds)
	if e := os.WriteFile(creds, []byte(strings.ReplaceAll(string(data), "role:admin", "role:viewer")), 0600); e != nil {
		t.Fatal(e)
	}
	if w = b.request("PUT", "/console/api/v1/publication", `{}`, "http://127.0.0.1:8080", b.csrf); w.Code != 403 {
		t.Fatal("stale roles", w.Code)
	}
	if e := os.WriteFile(creds, []byte(`{"principals":[]}`), 0600); e != nil {
		t.Fatal(e)
	}
	if w = b.request("GET", "/console/api/v1/me", "", "", ""); w.Code != 401 {
		t.Fatal("revoked session", w.Code, w.Body.String())
	}
}

func TestBrowserInFlightIdleRefreshCannotReviveLogout(t *testing.T) {
	h, _, browser := browserFixture(t)
	b := browserClient{h: h, cookies: map[string]*http.Cookie{}}
	b.bootstrap(t)
	if w := b.request("POST", "/console/api/v1/login", `{"token":"`+managementToken+`"}`, "http://127.0.0.1:8080", b.csrf); w.Code != 200 {
		t.Fatal(w.Code)
	}
	old := *b.cookies["picketx_session"]
	entered, resume, done := make(chan struct{}), make(chan struct{}), make(chan struct{})
	pending := browser.wrap(browser.sessions.LoadAndSave(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = browser.sessions.GetString(r.Context(), "id")
		close(entered)
		<-resume
		w.WriteHeader(200)
	})))
	go func() {
		defer close(done)
		r := httptest.NewRequest("GET", "http://127.0.0.1:8080/console/api/v1/me", nil)
		r.AddCookie(&old)
		pending.ServeHTTP(httptest.NewRecorder(), r)
	}()
	<-entered
	logout := make(chan *httptest.ResponseRecorder, 1)
	go func() { logout <- b.request("POST", "/console/api/v1/logout", `{}`, "http://127.0.0.1:8080", b.csrf) }()
	var result *httptest.ResponseRecorder
	select {
	case result = <-logout:
	case <-time.After(100 * time.Millisecond):
	}
	close(resume)
	<-done
	if result == nil {
		result = <-logout
	}
	if result.Code != 204 {
		t.Fatal(result.Code)
	}
	b.cookies[old.Name] = &old
	if w := b.request("GET", "/console/api/v1/me", "", "", ""); w.Code != 401 {
		t.Fatal("in-flight idle refresh resurrected logout", w.Code, w.Body.String())
	}
}

func TestBrowserStaticAssetsConfinementAndSessionDefaults(t *testing.T) {
	_, _, b := browserFixture(t)
	if b.sessions.Lifetime != 8*time.Hour || b.sessions.IdleTimeout != 30*time.Minute || !b.sessions.Cookie.HttpOnly || !b.sessions.HashTokenInStore {
		t.Fatal("unsafe session defaults")
	}
	dir := t.TempDir()
	b.assets = dir
	if e := os.WriteFile(filepath.Join(dir, "index.html"), []byte("real index"), 0600); e != nil {
		t.Fatal(e)
	}
	if e := os.WriteFile(filepath.Join(dir, "app.js"), []byte("real asset"), 0600); e != nil {
		t.Fatal(e)
	}
	outside := filepath.Join(t.TempDir(), "private.js")
	if e := os.WriteFile(outside, []byte("private outside"), 0600); e != nil {
		t.Fatal(e)
	}
	if e := os.Symlink(outside, filepath.Join(dir, "escape.js")); e != nil {
		t.Fatal(e)
	}
	h := NewHandlerWithBrowser(nil, b.service, ManagementLimits{1000, 1000}, b)
	for _, tc := range []struct {
		path string
		want int
	}{{"/console/", 200}, {"/console/app.js", 200}, {"/console/escape.js", 404}, {"/console/../private.js", 404}, {"/console/api/missing", 404}, {"/console/missing", 404}, {"/console/unknown.json", 404}} {
		r := httptest.NewRequest("GET", "http://127.0.0.1:8080"+tc.path, nil)
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)
		if w.Code != tc.want || strings.Contains(w.Body.String(), "private outside") {
			t.Fatal(tc, w.Code, w.Body.String())
		}
		if w.Header().Get("Cache-Control") != "no-store" || !strings.Contains(w.Header().Get("Content-Security-Policy"), "frame-ancestors 'none'") {
			t.Fatal(w.Header())
		}
	}
	secure, e := NewBrowser(b.service, "https://control.example", dir)
	if e != nil || !secure.sessions.Cookie.Secure {
		t.Fatal("HTTPS cookie", e)
	}
	if _, e = NewBrowser(b.service, "https://control.example", t.TempDir()); e == nil {
		t.Fatal("missing index accepted")
	}
}

func TestBrowserExpiryFixationAndModernCSRF(t *testing.T) {
	h, _, b := browserFixture(t)
	client := browserClient{h: h, cookies: map[string]*http.Cookie{}}
	client.bootstrap(t)
	login := `{"token":"` + managementToken + `"}`
	first := client.request("POST", "/console/api/v1/login", login, "http://127.0.0.1:8080", client.csrf)
	if first.Code != 200 {
		t.Fatal(first.Code)
	}
	old := *client.cookies["picketx_session"]
	if w := client.request("POST", "/console/api/v1/login", login, "http://127.0.0.1:8080", client.csrf); w.Code != 200 {
		t.Fatal(w.Code)
	}
	if old.Value == client.cookies[old.Name].Value {
		t.Fatal("login did not renew session")
	}
	replay := browserClient{h: h, cookies: map[string]*http.Cookie{old.Name: &old}}
	if w := replay.request("GET", "/console/api/v1/me", "", "", ""); w.Code != 401 {
		t.Fatal("old pre-login session accepted", w.Code)
	}
	ctx, e := b.sessions.Load(context.Background(), client.cookies[old.Name].Value)
	if e != nil {
		t.Fatal(e)
	}
	deadline := b.sessions.Deadline(ctx)
	if w := client.request("GET", "/console/api/v1/me", "", "", ""); w.Code != 200 {
		t.Fatal(w.Code)
	}
	ctx, e = b.sessions.Load(context.Background(), client.cookies[old.Name].Value)
	if e != nil || !b.sessions.Deadline(ctx).Equal(deadline) {
		t.Fatal("idle refresh extended absolute expiry", e)
	}
	// Native modern metadata is accepted only with our independent exact Origin.
	for _, site := range []string{"same-origin", "cross-site"} {
		r := httptest.NewRequest("POST", "http://127.0.0.1:8080/console/api/v1/login", strings.NewReader(login))
		r.Header.Set("Content-Type", "application/json")
		r.Header.Set("Origin", "http://127.0.0.1:8080")
		r.Header.Set("Sec-Fetch-Site", site)
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)
		want := 200
		if site == "cross-site" {
			want = 403
		}
		if w.Code != want {
			t.Fatal(site, w.Code, w.Body.String())
		}
	}
	for _, expiry := range []struct {
		name           string
		lifetime, idle time.Duration
	}{{"absolute", 10 * time.Millisecond, time.Hour}, {"idle", time.Hour, 10 * time.Millisecond}} {
		b.sessions.Lifetime = expiry.lifetime
		b.sessions.IdleTimeout = expiry.idle
		if w := client.request("POST", "/console/api/v1/login", login, "http://127.0.0.1:8080", client.csrf); w.Code != 200 {
			t.Fatal(w.Code)
		}
		time.Sleep(30 * time.Millisecond)
		if w := client.request("GET", "/console/api/v1/me", "", "", ""); w.Code != 401 {
			t.Fatal(expiry.name, "expired session accepted", w.Code)
		}
	}
}

func TestBrowserRejectsAmbiguousOriginAndDoesNotAdvertiseBearer(t *testing.T) {
	h, _, _ := browserFixture(t)
	r := httptest.NewRequest("POST", "http://127.0.0.1:8080/console/api/v1/login", strings.NewReader(`{"token":"`+managementToken+`"}`))
	r.Header.Set("Content-Type", "application/json")
	r.Header.Add("Origin", "http://127.0.0.1:8080")
	r.Header.Add("Origin", "http://evil.example")
	r.Header.Set("Sec-Fetch-Site", "same-origin")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != 403 {
		t.Error("ambiguous Origin accepted", w.Code)
	}
	r = httptest.NewRequest("GET", "http://127.0.0.1:8080/console/api/v1/me", nil)
	w = httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != 401 || w.Header().Get("WWW-Authenticate") != "" {
		t.Error("cookie API advertised unsupported Bearer", w.Code, w.Header())
	}
}
