// SPDX-License-Identifier: Apache-2.0
package controller

import (
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
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
