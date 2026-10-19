// SPDX-License-Identifier: Apache-2.0
package controller

import (
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/PicketX/PicketX/internal/account"
	"github.com/PicketX/PicketX/internal/management"
	"github.com/PicketX/PicketX/internal/managementaccess"
)

func managedAccountsFixture(t *testing.T) http.Handler {
	t.Helper()
	_, store, path := managementFixture(t)
	a, err := managementaccess.New(store.GORM(), managementaccess.Settings{Credentials: path, UsersMode: "database", Reload: time.Second, WarnRules: 10000, WarnLoadPercent: 50}, slog.New(slog.NewTextHandler(io.Discard, nil)))
	if err != nil {
		t.Fatal(err)
	}
	return NewHandlerWithManagement(nil, management.New(a, directRepository{store}), ManagementLimits{RequestsPerSecond: 1000, Burst: 1000})
}

func TestManagedAccountsHTTPStrictCASAndSecretFreeDirectory(t *testing.T) {
	h := managedAccountsFixture(t)
	if w := request(h, "GET", "/api/v1/accounts", "", "", ""); w.Code != 401 {
		t.Fatal(w.Code, w.Body.String())
	}
	w := request(h, "GET", "/api/v1/accounts", "", managementToken, "")
	if w.Code != 200 || w.Header().Get("ETag") != `"0"` || w.Header().Get("Cache-Control") != "no-store" {
		t.Fatal(w.Code, w.Body.String())
	}
	create := `{"username":"alice","password":"correct human password!","roles":["role:requester"],"reason":"onboard"}`
	if w = request(h, "POST", "/api/v1/accounts", create, managementToken, ""); w.Code != 428 {
		t.Fatal(w.Code)
	}
	for _, bad := range []string{`W/"0"`, `"00"`, `*`, `"0", "1"`} {
		if w = request(h, "POST", "/api/v1/accounts", create, managementToken, bad); w.Code != 400 {
			t.Fatal(bad, w.Code)
		}
	}
	for _, bad := range []string{`null`, strings.Replace(create, `"username":`, `"username":null,"username":`, 1), strings.Replace(create, `"username":`, `"extra":`, 1), strings.Replace(create, `"role:requester"`, `"role:custom"`, 1), strings.Replace(create, `correct human password!`, `short`, 1)} {
		if w = request(h, "POST", "/api/v1/accounts", bad, managementToken, `"0"`); w.Code != 400 {
			t.Fatal(bad, w.Code, w.Body.String())
		}
	}
	w = request(h, "POST", "/api/v1/accounts", create, managementToken, `"0"`)
	if w.Code != 201 || w.Header().Get("ETag") != `"1"` {
		t.Fatal(w.Code, w.Body.String())
	}
	var created account.Account
	if err := json.Unmarshal(w.Body.Bytes(), &created); err != nil || created.Version != 1 {
		t.Fatal(created, err)
	}
	path := "/api/v1/accounts/" + created.ID
	for _, bad := range []string{`{"roles":["role:viewer"],"reason":"missing disabled"}`, `{"roles":["role:viewer"],"disabled":null,"reason":"null"}`, `{"username":"rename","roles":["role:viewer"],"disabled":false,"reason":"rename"}`} {
		if w = request(h, "PUT", path, bad, managementToken, `"1"`); w.Code != 400 {
			t.Fatal(bad, w.Code, w.Body.String())
		}
	}
	update := `{"roles":["role:viewer"],"disabled":false,"reason":"roles changed"}`
	if w = request(h, "PUT", path, update, managementToken, `"0"`); w.Code != 412 {
		t.Fatal(w.Code, w.Body.String())
	}
	if w = request(h, "PUT", path, update, managementToken, `"1"`); w.Code != 200 || w.Header().Get("ETag") != `"2"` {
		t.Fatal(w.Code, w.Body.String())
	}
	w = request(h, "GET", "/api/v1/accounts", "", managementToken, "")
	if w.Code != 200 || strings.Contains(w.Body.String(), "password") || strings.Contains(w.Body.String(), "$2") || !strings.Contains(w.Body.String(), `"revision":2`) {
		t.Fatal(w.Code, w.Body.String())
	}
	if w = request(h, "GET", "/api/v1/accounts?unknown=x", "", managementToken, ""); w.Code != 400 {
		t.Fatal(w.Code)
	}
	if w = request(h, "PUT", strings.ReplaceAll(path, ":", "%3A"), update, managementToken, `"2"`); w.Code != 200 {
		t.Fatal("browser encoded principal path", w.Code, w.Body.String())
	}
	file, _, _ := managementFixture(t)
	if w = request(file, "GET", "/api/v1/accounts", "", managementToken, ""); w.Code != 403 {
		t.Fatal("file account API exposed", w.Code, w.Body.String())
	}
}

func TestManagedAccountsBrowserCSRFAndPermanentRevocation(t *testing.T) {
	_, store, path := managementFixture(t)
	a, err := managementaccess.New(store.GORM(), managementaccess.Settings{Credentials: path, UsersMode: "database", Reload: time.Second, WarnRules: 10000, WarnLoadPercent: 50}, slog.New(slog.NewTextHandler(io.Discard, nil)))
	if err != nil {
		t.Fatal(err)
	}
	service := management.New(a, directRepository{store})
	browser, err := NewBrowser(service, "http://127.0.0.1:8080", "")
	if err != nil {
		t.Fatal(err)
	}
	h := NewHandlerWithBrowser(nil, service, ManagementLimits{1000, 1000}, browser)
	admin := browserClient{h: h, cookies: map[string]*http.Cookie{}}
	admin.bootstrap(t)
	if w := admin.request("POST", "/console/api/v1/login", `{"token":"`+managementToken+`"}`, "http://127.0.0.1:8080", admin.csrf); w.Code != 200 {
		t.Fatal(w.Code, w.Body.String())
	}
	for _, v := range [][2]string{{"", admin.csrf}, {"http://evil.test", admin.csrf}, {"http://127.0.0.1:8080", ""}} {
		if w := admin.request("POST", "/console/api/v1/accounts", `{}`, v[0], v[1]); w.Code != 403 {
			t.Fatal("account CSRF", w.Code, w.Body.String())
		}
	}
	create := `{"username":"alice","password":"correct human password!","roles":["role:requester"],"reason":"browser human"}`
	w := request(h, "POST", "/api/v1/accounts", create, managementToken, `"0"`)
	if w.Code != 201 {
		t.Fatal(w.Code, w.Body.String())
	}
	var created account.Account
	if err = json.Unmarshal(w.Body.Bytes(), &created); err != nil {
		t.Fatal(err)
	}
	human := browserClient{h: h, cookies: map[string]*http.Cookie{}}
	human.bootstrap(t)
	if w = human.request("POST", "/console/api/v1/login/password", `{"username":"alice","password":"correct human password!"}`, "http://127.0.0.1:8080", human.csrf); w.Code != 200 {
		t.Fatal(w.Code, w.Body.String())
	}
	old := *human.cookies["picketx_session"]
	if w = human.request("GET", "/console/api/v1/accounts", "", "", ""); w.Code != 403 {
		t.Fatal("requester directory access", w.Code)
	}
	for i, disabled := range []string{"true", "false"} {
		body := `{"roles":["role:requester"],"disabled":` + disabled + `,"reason":"lifecycle"}`
		match := `"1"`
		if i == 1 {
			match = `"2"`
		}
		if w = request(h, "PUT", "/api/v1/accounts/"+created.ID, body, managementToken, match); w.Code != 200 {
			t.Fatal(w.Code, w.Body.String())
		}
	}
	human.cookies[old.Name] = &old
	if w = human.request("GET", "/console/api/v1/me", "", "", ""); w.Code != 401 {
		t.Fatal("old cookie revived", w.Code, w.Body.String())
	}
}
