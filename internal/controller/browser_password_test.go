// SPDX-License-Identifier: Apache-2.0
package controller

import (
	"encoding/json"
	"golang.org/x/crypto/bcrypt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync"
	"testing"
)

const browserPassword = "a human browser password!"

var browserHashOnce sync.Once
var browserPasswordHash string

func writeBrowserUser(t *testing.T, path string, roles []string, disabled bool) {
	t.Helper()
	browserHashOnce.Do(func() {
		h, e := bcrypt.GenerateFromPassword([]byte(browserPassword), 12)
		if e != nil {
			panic(e)
		}
		browserPasswordHash = string(h)
	})
	data, _ := os.ReadFile(path)
	var doc map[string]any
	if json.Unmarshal(data, &doc) != nil {
		t.Fatal("credential fixture")
	}
	doc["users"] = []any{map[string]any{"id": "principal:alice", "username": "alice", "password_hash": browserPasswordHash, "roles": roles, "disabled": disabled}}
	data, _ = json.Marshal(doc)
	if e := os.WriteFile(path, data, 0600); e != nil {
		t.Fatal(e)
	}
}
func TestBrowserPasswordLoginSessionAndRevocation(t *testing.T) {
	h, path, browser := browserFixture(t)
	writeBrowserUser(t, path, []string{"role:admin"}, false)
	b := browserClient{h: h, cookies: map[string]*http.Cookie{}}
	b.bootstrap(t)
	w := b.request("GET", "/console/api/v1/session", "", "", "")
	var session struct {
		PasswordLogin bool `json:"password_login"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &session)
	if w.Code != 200 || !session.PasswordLogin {
		t.Fatal("password login not advertised", w.Code, w.Body.String())
	}
	body := `{"username":"alice","password":"` + browserPassword + `"}`
	for _, v := range [][2]string{{"", b.csrf}, {"http://evil.test", b.csrf}, {"http://127.0.0.1:8080", ""}} {
		if w = b.request("POST", "/console/api/v1/login/password", body, v[0], v[1]); w.Code != 403 {
			t.Fatal("password CSRF", w.Code)
		}
	}
	if w = b.request("POST", "/console/api/v1/login/password", body, "http://127.0.0.1:8080", b.csrf); w.Code != 200 {
		t.Fatal("password route missing", w.Code, w.Body.String())
	}
	old := *b.cookies["picketx_session"]
	if strings.Contains(w.Body.String(), browserPassword) || strings.Contains(w.Body.String(), browserPasswordHash) {
		t.Fatal("secret in response")
	}
	r := httptest.NewRequest("GET", "http://127.0.0.1:8080/console/api/v1/me", nil)
	r.AddCookie(&old)
	check := browser.sessions.LoadAndSave(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		if browser.sessions.GetString(r.Context(), "id") != "principal:alice" || len(browser.sessions.GetString(r.Context(), "credential")) != 64 {
			t.Error("session identity/fingerprint")
		}
		if browser.sessions.GetString(r.Context(), "password") != "" {
			t.Error("raw password stored")
		}
	}))
	check.ServeHTTP(httptest.NewRecorder(), r)
	if w = b.request("GET", "/console/api/v1/me", "", "", ""); w.Code != 200 || !strings.Contains(w.Body.String(), "principal:alice") {
		t.Fatal(w.Code, w.Body.String())
	}
	if w = b.request("GET", "/api/v1/me", "", "", ""); w.Code != 401 {
		t.Fatal("password cookie authenticated API", w.Code)
	}
	if w = b.request("POST", "/console/api/v1/login/password", body, "http://127.0.0.1:8080", b.csrf); w.Code != 200 {
		t.Fatal(w.Code)
	}
	if old.Value == b.cookies["picketx_session"].Value {
		t.Fatal("login did not renew session")
	}
	writeBrowserUser(t, path, []string{"role:viewer"}, false)
	if w = b.request("PUT", "/console/api/v1/publication", `{}`, "http://127.0.0.1:8080", b.csrf); w.Code != 403 {
		t.Fatal("stale human role", w.Code)
	}
	writeBrowserUser(t, path, []string{"role:admin"}, true)
	if w = b.request("GET", "/console/api/v1/me", "", "", ""); w.Code != 401 {
		t.Fatal("disabled human session", w.Code)
	}
	if w = b.request("POST", "/console/api/v1/login/password", body, "http://127.0.0.1:8080", b.csrf); w.Code != 401 {
		t.Fatal("disabled human login", w.Code)
	}
}
func TestBrowserPasswordStrictInputAndPeerLimit(t *testing.T) {
	h, path, _ := browserFixture(t)
	writeBrowserUser(t, path, []string{"role:admin"}, false)
	b := browserClient{h: h, cookies: map[string]*http.Cookie{}}
	b.bootstrap(t)
	for _, body := range []string{`{"username":"alice","password":"a human browser password!","roles":["role:admin"]}`, `{"username":"alice","password":null}`, `{"username":"alice","password":"short"}`, `{"username":"alice","password":"` + strings.Repeat("界", 25) + `"}`} {
		if w := b.request("POST", "/console/api/v1/login/password", body, "http://127.0.0.1:8080", b.csrf); w.Code != 400 {
			t.Fatal("strict password input", w.Code, w.Body.String())
		}
	}
	if w := b.request("POST", "/console/api/v1/login/password", strings.Repeat("x", 4097), "http://127.0.0.1:8080", b.csrf); w.Code != 413 {
		t.Fatal(w.Code)
	}
	// Route-local budget is shared despite arbitrary forwarding headers; the
	// general session/read route remains available after password exhaustion.
	r := httptest.NewRequest("POST", "http://127.0.0.1:8080/console/api/v1/login/password", strings.NewReader(`{}`))
	r.Header.Set("Content-Type", "application/json")
	r.Header.Set("Origin", "http://127.0.0.1:8080")
	r.Header.Set("X-CSRF-Token", b.csrf)
	r.Header.Set("X-Forwarded-For", "203.0.113.88")
	for _, c := range b.cookies {
		r.AddCookie(c)
	}
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != 429 {
		t.Fatal("forwarded peer bypassed password limit", w.Code, w.Body.String())
	}
	if w = b.request("GET", "/console/api/v1/session", "", "", ""); w.Code != 200 {
		t.Fatal("password limit blocked key recovery", w.Code)
	}
}
