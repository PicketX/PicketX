// SPDX-License-Identifier: Apache-2.0
package controller

import (
	"encoding/json"
	"net/http"
	"os"
	"strings"
	"testing"
)

const routeRemovalBody = `{"reason":"retire exact HTTP route","acknowledge_removal":true}`

func TestHTTPRouteRemovalOpenAPI(t *testing.T) {
	raw, err := os.ReadFile("../../docs/api/swagger.json")
	if err != nil {
		t.Fatal(err)
	}
	var spec struct {
		Paths map[string]map[string]struct {
			Security  []map[string][]string
			Responses map[string]any
		}
		Components struct {
			Schemas map[string]struct {
				AdditionalProperties *bool
				Required             []string
				Properties           map[string]struct {
					Const     *bool
					MaxLength int
				}
			}
		}
	}
	if err = json.Unmarshal(raw, &spec); err != nil {
		t.Fatal(err)
	}
	for prefix, security := range map[string]string{"/api/v1": "ManagementBearer", "/console/api/v1": "BrowserCookie"} {
		op := spec.Paths[prefix+"/http-routes/{uid}/remove"]["post"]
		if len(op.Security) != 1 || op.Security[0][security] == nil {
			t.Fatal("security", prefix)
		}
		for _, code := range []string{"200", "400", "401", "403", "404", "412", "413", "415", "428", "429", "503"} {
			if _, ok := op.Responses[code]; !ok {
				t.Fatal(prefix, code)
			}
		}
	}
	s := spec.Components.Schemas["controller.RemoveHTTPRouteRequest"]
	if s.AdditionalProperties == nil || *s.AdditionalProperties || len(s.Required) != 2 || s.Properties["reason"].MaxLength != 512 || s.Properties["acknowledge_removal"].Const == nil || !*s.Properties["acknowledge_removal"].Const {
		t.Fatal("unsafe schema", s)
	}
}

// Catches wrong-target deletion, cascade, lost durable source/facts and missing audit.
func TestHTTPRouteRemovalHTTP(t *testing.T) {
	h, store, _ := managementFixture(t)
	original, facts := inventorySeed(t, h, false)
	w := request(h, "POST", "/api/v1/http-routes", routeCreate, managementToken, `"1"`)
	if w.Code != 201 {
		t.Fatal(w.Code, w.Body.String())
	}
	var target struct{ UID string }
	if err := json.Unmarshal(w.Body.Bytes(), &target); err != nil {
		t.Fatal(err)
	}
	for _, uid := range []string{"resource-gitea-v1", "gitea-web", "missing"} {
		w = request(h, "POST", "/api/v1/http-routes/"+uid+"/remove", routeRemovalBody, managementToken, `"2"`)
		if w.Code != 404 || !strings.Contains(w.Body.String(), "http_route_not_found") {
			t.Fatal(uid, w.Code, w.Body.String())
		}
	}
	path := "/api/v1/http-routes/" + target.UID + "/remove"
	w = request(h, "POST", path, routeRemovalBody, managementToken, `"2"`)
	if w.Code != 200 || w.Header().Get("ETag") != `"3"` {
		t.Fatal(w.Code, w.Body.String())
	}
	var receipt struct {
		UID      string
		Revision uint64
	}
	if err := json.Unmarshal(w.Body.Bytes(), &receipt); err != nil || receipt.UID != target.UID || receipt.Revision != 3 {
		t.Fatal(receipt, err)
	}
	var persisted struct {
		Policy, Facts []byte
		Revision      int64
	}
	if err := store.GORM().Raw("SELECT policy,facts,revision FROM controller_publication WHERE id=1").Scan(&persisted).Error; err != nil {
		t.Fatal(err)
	}
	if string(persisted.Policy) != original+"\n" || string(persisted.Facts) != facts || persisted.Revision != 3 {
		t.Fatal("durable source/facts changed", persisted.Revision, string(persisted.Policy))
	}
	events, err := store.Audit(0, 100)
	if err != nil || len(events) != 3 || events[2].Reason != "retire exact HTTP route" || events[2].Actor != "principal:operator" {
		t.Fatal(events, err)
	}
	if w = request(h, "POST", path, routeRemovalBody, managementToken, `"2"`); w.Code != 412 {
		t.Fatal("stale before absent", w.Code)
	}
	if w = request(h, "POST", path, routeRemovalBody, managementToken, `"3"`); w.Code != 404 {
		t.Fatal("absent", w.Code)
	}
}

// Catches bypassed acknowledgement/auth/CAS and nontransactional audit failure.
func TestHTTPRouteRemovalValidationAndRollback(t *testing.T) {
	h, store, creds := managementFixture(t)
	inventorySeed(t, h, false)
	w := request(h, "POST", "/api/v1/http-routes", routeCreate, managementToken, `"1"`)
	if w.Code != 201 {
		t.Fatal(w.Code, w.Body.String())
	}
	var target struct{ UID string }
	if err := json.Unmarshal(w.Body.Bytes(), &target); err != nil {
		t.Fatal(err)
	}
	path := "/api/v1/http-routes/" + target.UID + "/remove"
	before, err := store.Snapshot()
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		body, token, rev string
		want             int
	}{
		{routeRemovalBody, managementToken, "", 428}, {routeRemovalBody, managementToken, `"1"`, 412}, {routeRemovalBody, "", `"2"`, 401},
		{`{"reason":"x"}`, managementToken, `"2"`, 400},
		{`{"reason":"x","acknowledge_removal":false}`, managementToken, `"2"`, 400},
		{`{"reason":"x","acknowledge_removal":null}`, managementToken, `"2"`, 400},
		{`{"reason":"x","acknowledge_removal":true,"acknowledge_removal":true}`, managementToken, `"2"`, 400},
		{`{"reason":"x","acknowledge_removal":true,"uid":"another"}`, managementToken, `"2"`, 400},
		{`{"reason":"   ","acknowledge_removal":true}`, managementToken, `"2"`, 400},
		{`{"reason":"` + strings.Repeat("x", 17000) + `","acknowledge_removal":true}`, managementToken, `"2"`, 413},
	} {
		if w = request(h, "POST", path, tc.body, tc.token, tc.rev); w.Code != tc.want {
			t.Fatal(tc.want, w.Code, w.Body.String())
		}
	}
	if err = store.GORM().Exec(`CREATE TRIGGER reject_route_remove_audit BEFORE INSERT ON controller_audit BEGIN SELECT RAISE(FAIL, 'audit unavailable'); END`).Error; err != nil {
		t.Fatal(err)
	}
	if w = request(h, "POST", path, routeRemovalBody, managementToken, `"2"`); w.Code != 503 {
		t.Fatal(w.Code, w.Body.String())
	}
	var persisted struct {
		Policy, Facts []byte
		Revision      int64
	}
	if err = store.GORM().Raw("SELECT policy,facts,revision FROM controller_publication WHERE id=1").Scan(&persisted).Error; err != nil {
		t.Fatal(err)
	}
	events, err := store.Audit(0, 100)
	if err != nil || len(events) != 2 || string(persisted.Policy) != before.Policy || string(persisted.Facts) != before.Facts || persisted.Revision != 2 {
		t.Fatal("rejection mutated durable state", events, err)
	}
	if err = store.GORM().Exec("DROP TRIGGER reject_route_remove_audit").Error; err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(creds)
	if err != nil {
		t.Fatal(err)
	}
	if err = os.WriteFile(creds, []byte(strings.ReplaceAll(string(data), "role:admin", "role:viewer")), 0600); err != nil {
		t.Fatal(err)
	}
	if w = request(h, "POST", path, routeRemovalBody, managementToken, `"2"`); w.Code != 403 {
		t.Fatal("viewer", w.Code)
	}
}

func TestHTTPRouteRemovalCookieCSRF(t *testing.T) {
	h, _, _ := browserFixture(t)
	client := browserClient{h: h, cookies: map[string]*http.Cookie{}}
	client.bootstrap(t)
	if w := client.request("POST", "/console/api/v1/login", `{"token":"`+managementToken+`"}`, "http://127.0.0.1:8080", client.csrf); w.Code != 200 {
		t.Fatal(w.Code)
	}
	path := "/console/api/v1/http-routes/route-web/remove"
	if w := client.request("POST", path, routeRemovalBody, "http://127.0.0.1:8080", ""); w.Code != 403 {
		t.Fatal("CSRF", w.Code)
	}
	if w := client.request("POST", path, routeRemovalBody, "http://127.0.0.1:8080", client.csrf); w.Code != 428 {
		t.Fatal("CAS", w.Code)
	}
}
