// SPDX-License-Identifier: Apache-2.0
package controller

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/PicketX/PicketX/internal/management"
	"github.com/PicketX/PicketX/internal/managementaccess"
)

func browserFixture(t *testing.T) (http.Handler, string, *Browser) {
	return browserFixtureOrigin(t, "http://127.0.0.1:8080")
}

func browserFixtureOrigin(t *testing.T, origin string) (http.Handler, string, *Browser) {
	t.Helper()
	_, s, creds := managementFixture(t)
	a, e := managementaccess.New(s.GORM(), managementaccess.Settings{Credentials: creds, Reload: time.Second, WarnRules: 10000, WarnLoadPercent: 50}, slog.New(slog.NewTextHandler(io.Discard, nil)))
	if e != nil {
		t.Fatal(e)
	}
	service := management.New(a, directRepository{s})
	b, e := NewBrowser(service, origin, "")
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

type stalledLoginBody struct {
	entered, release chan struct{}
	once             sync.Once
}

func (b *stalledLoginBody) Read(_ []byte) (int, error) {
	b.once.Do(func() { close(b.entered) })
	<-b.release
	return 0, io.EOF
}
func (*stalledLoginBody) Close() error { return nil }

func TestBrowserUnknownCookieCannotBlockAuthenticatedRequest(t *testing.T) {
	browserSessionMustNotBlockAnother(t, false)
}
func TestBrowserDifferentStoredSessionCannotBlockAuthenticatedRequest(t *testing.T) {
	browserSessionMustNotBlockAnother(t, true)
}
func browserSessionMustNotBlockAnother(t *testing.T, known bool) {
	t.Helper()
	h, _, browser := browserFixture(t)
	client := browserClient{h: h, cookies: map[string]*http.Cookie{}}
	client.bootstrap(t)
	if w := client.request("POST", "/console/api/v1/login", `{"token":"`+managementToken+`"}`, "http://127.0.0.1:8080", client.csrf); w.Code != 200 {
		t.Fatal(w.Code)
	}
	sum := sha256.Sum256([]byte(client.cookies["picketx_session"].Value))
	var dummy string
	for n := 0; n < 100000; n++ {
		candidate := fmt.Sprintf("unknown-cookie-%d", n)
		hash := sha256.Sum256([]byte(candidate))
		if hash[0]%64 == sum[0]%64 {
			dummy = candidate
			break
		}
	}
	if dummy == "" {
		t.Fatal("no test collision")
	}
	if known {
		// Trusted test-store setup gives a second valid session an intentionally
		// colliding old stripe. Authentication and request handling remain real;
		// no external caller can populate this store or choose SCS login tokens.
		ctx, e := browser.sessions.Load(context.Background(), client.cookies["picketx_session"].Value)
		if e != nil {
			t.Fatal(e)
		}
		data, e := browser.sessions.Codec.Encode(browser.sessions.Deadline(ctx), map[string]interface{}{"id": browser.sessions.GetString(ctx, "id"), "credential": browser.sessions.GetString(ctx, "credential")})
		if e != nil {
			t.Fatal(e)
		}
		hash := sha256.Sum256([]byte(dummy))
		if e = browser.sessions.Store.Commit(base64.RawURLEncoding.EncodeToString(hash[:]), data, time.Now().Add(time.Hour)); e != nil {
			t.Fatal(e)
		}
	}
	body := &stalledLoginBody{entered: make(chan struct{}), release: make(chan struct{})}
	r := httptest.NewRequest("POST", "http://127.0.0.1:8080/console/api/v1/login", body)
	r.Header.Set("Content-Type", "application/json")
	r.Header.Set("Origin", "http://127.0.0.1:8080")
	r.Header.Set("Sec-Fetch-Site", "same-origin")
	r.AddCookie(&http.Cookie{Name: "picketx_session", Value: dummy})
	attackerDone := make(chan struct{})
	go func() { defer close(attackerDone); h.ServeHTTP(httptest.NewRecorder(), r) }()
	select {
	case <-body.entered:
	case <-time.After(time.Second):
		close(body.release)
		<-attackerDone
		t.Fatal("test body not reached")
	}
	legitimate := make(chan *httptest.ResponseRecorder, 1)
	go func() { legitimate <- client.request("GET", "/console/api/v1/me", "", "", "") }()
	var result *httptest.ResponseRecorder
	select {
	case result = <-legitimate:
	case <-time.After(200 * time.Millisecond):
		t.Error("another cookie's stalled body blocked authenticated session; stored=", known)
	}
	close(body.release)
	<-attackerDone
	if result == nil {
		result = <-legitimate
	}
	if result.Code != 200 {
		t.Fatal(result.Code, result.Body.String())
	}
}

func TestBrowserCanceledSessionWaitReleasesResources(t *testing.T) {
	h, _, browser := browserFixture(t)
	client := browserClient{h: h, cookies: map[string]*http.Cookie{}}
	client.bootstrap(t)
	if w := client.request("POST", "/console/api/v1/login", `{"token":"`+managementToken+`"}`, "http://127.0.0.1:8080", client.csrf); w.Code != 200 {
		t.Fatal(w.Code)
	}
	body := &stalledLoginBody{entered: make(chan struct{}), release: make(chan struct{})}
	r := httptest.NewRequest("POST", "http://127.0.0.1:8080/console/api/v1/login", body)
	r.Header.Set("Content-Type", "application/json")
	r.Header.Set("Origin", "http://127.0.0.1:8080")
	r.Header.Set("Sec-Fetch-Site", "same-origin")
	r.AddCookie(client.cookies["picketx_session"])
	done := make(chan struct{})
	go func() { defer close(done); h.ServeHTTP(httptest.NewRecorder(), r) }()
	<-body.entered
	ctx, cancel := context.WithCancel(context.Background())
	r = httptest.NewRequest("GET", "http://127.0.0.1:8080/console/api/v1/me", nil).WithContext(ctx)
	r.AddCookie(client.cookies["picketx_session"])
	cancel()
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	close(body.release)
	<-done
	if w.Code != 408 {
		t.Fatal("canceled queued request not released", w.Code)
	}
	if w = client.request("GET", "/console/api/v1/me", "", "", ""); w.Code != 200 {
		t.Fatal(w.Code)
	}
	browser.lockMu.Lock()
	remaining := len(browser.locks)
	browser.lockMu.Unlock()
	if remaining != 0 {
		t.Fatal("completed/canceled session locks retained", remaining)
	}
}

func TestOptionalBrowserOriginUsesRequestOrigin(t *testing.T) {
	h, _, _ := browserFixtureOrigin(t, "")
	for _, origin := range []string{"http://127.0.0.1:5173", "http://localhost:8080", "http://control.example:8081", "https://control.example"} {
		t.Run(origin, func(t *testing.T) {
			t.Parallel()
			r := httptest.NewRequest("GET", origin+"/console/api/v1/session", nil)
			out := httptest.NewRecorder()
			h.ServeHTTP(out, r)
			if out.Code != 200 {
				t.Fatal("optional origin rejected", out.Code)
			}
			var session struct {
				CSRF string `json:"csrf"`
			}
			if err := json.Unmarshal(out.Body.Bytes(), &session); err != nil {
				t.Fatal(err)
			}
			secure := strings.HasPrefix(origin, "https:")
			for _, c := range out.Result().Cookies() {
				if c.Name == "picketx_csrf" && c.Secure != secure {
					t.Fatal("wrong CSRF cookie security")
				}
			}
			post := func(requestOrigin string, forwarded bool) *httptest.ResponseRecorder {
				req := httptest.NewRequest("POST", origin+"/console/api/v1/login", strings.NewReader(`{"token":"`+managementToken+`"}`))
				req.Header.Set("Content-Type", "application/json")
				req.Header.Set("Origin", requestOrigin)
				req.Header.Set("X-CSRF-Token", session.CSRF)
				if forwarded {
					req.Header.Set("X-Forwarded-Proto", "https")
					req.Header.Set("X-Forwarded-Host", "evil.test")
					req.Header.Set("Forwarded", "proto=https;host=evil.test")
				}
				for _, c := range out.Result().Cookies() {
					req.AddCookie(c)
				}
				result := httptest.NewRecorder()
				h.ServeHTTP(result, req)
				return result
			}
			good := post(origin, true)
			if good.Code != 200 {
				t.Fatal("same-origin login rejected", good.Code)
			}
			found := false
			for _, c := range good.Result().Cookies() {
				if c.Name == "picketx_session" {
					found = true
					if c.Secure != secure {
						t.Fatal("wrong session cookie security")
					}
				}
			}
			if !found {
				t.Fatal("no session cookie")
			}
			for _, bad := range []string{"", "null", "http://evil.test", origin + ":1234", strings.Replace(origin, "http:", "https:", 1)} {
				if bad == origin {
					continue
				}
				if got := post(bad, false); got.Code != 403 {
					t.Fatal("cross-origin write accepted", bad, got.Code)
				}
			}
		})
	}
}
