// SPDX-License-Identifier: Apache-2.0
package controller

import (
	"encoding/json"
	"net/http"
	"os"
	"strings"
	"testing"
)

const resourceRemovalBody = `{"reason":"retire unused definition","acknowledge_removal":true}`

// A valid imported UID must not collide with the synthetic empty marker.
func TestResourceRemovalEmptyMarkerIdentity(t *testing.T) {
	h, store, _ := managementFixture(t)
	raw, err := os.ReadFile("../../examples/static/gitea.yaml")
	if err != nil {
		t.Fatal(err)
	}
	facts, err := os.ReadFile("../../examples/static/facts.json")
	if err != nil {
		t.Fatal(err)
	}
	policy := strings.ReplaceAll(strings.Split(string(raw), "---\n")[0], "resource-gitea-v1", "empty-binding-removal")
	body, _ := json.Marshal(map[string]string{"policy": policy, "facts": string(facts), "reason": "import valid marker-like UID"})
	if w := request(h, "PUT", "/api/v1/publication", string(body), managementToken, `"0"`); w.Code != 200 {
		t.Fatal(w.Code, w.Body.String())
	}
	w := request(h, "POST", "/api/v1/resources/empty-binding-removal/remove", resourceRemovalBody, managementToken, `"1"`)
	if w.Code != 200 || w.Header().Get("ETag") != `"2"` {
		t.Fatal(w.Code, w.Body.String())
	}
	doc, err := store.Snapshot()
	if err != nil || doc.Revision != 2 || !strings.Contains(doc.Policy, "kind: Configuration") || strings.Contains(doc.Policy, "uid: empty-binding-removal}") {
		t.Fatal("empty marker identity", doc, err)
	}
	audit, err := store.Audit(0, 100)
	if err != nil || len(audit) != 2 || audit[1].Reason != "retire unused definition" {
		t.Fatal(audit, err)
	}
}

// Catches silent cascade, missing audit and cached-only durability assertions.
func TestResourceRemovalHTTP(t *testing.T) {
	h, store, _ := managementFixture(t)
	original, facts := inventorySeed(t, h, false)
	w := request(h, "POST", "/api/v1/resources", inventoryResource, managementToken, `"1"`)
	if w.Code != 201 {
		t.Fatal(w.Code, w.Body.String())
	}
	var added struct {
		UID string `json:"uid"`
	}
	if e := json.Unmarshal(w.Body.Bytes(), &added); e != nil {
		t.Fatal(e)
	}
	path := "/api/v1/resources/" + added.UID + "/remove"
	w = request(h, "POST", path, resourceRemovalBody, managementToken, `"2"`)
	if w.Code != 200 || w.Header().Get("ETag") != `"3"` {
		t.Fatal(w.Code, w.Body.String())
	}
	var receipt struct {
		UID      string `json:"uid"`
		Revision uint64 `json:"revision"`
	}
	if e := json.Unmarshal(w.Body.Bytes(), &receipt); e != nil || receipt.UID != added.UID || receipt.Revision != 3 {
		t.Fatal(receipt, e)
	}
	var persisted struct {
		Policy, Facts []byte
		Revision      int64
	}
	if e := store.GORM().Raw("SELECT policy,facts,revision FROM controller_publication WHERE id=1").Scan(&persisted).Error; e != nil {
		t.Fatal(e)
	}
	if string(persisted.Policy) != original+"\n" || string(persisted.Facts) != facts || persisted.Revision != 3 {
		t.Fatal("durable source changed", persisted.Revision, string(persisted.Policy))
	}
	audit, e := store.Audit(0, 100)
	if e != nil || len(audit) != 3 || audit[2].Reason != "retire unused definition" || audit[2].Actor != "principal:operator" {
		t.Fatal(audit, e)
	}
	if w = request(h, "POST", path, resourceRemovalBody, managementToken, `"2"`); w.Code != 412 {
		t.Fatal("stale before absent", w.Code)
	}
	if w = request(h, "POST", path, resourceRemovalBody, managementToken, `"3"`); w.Code != 404 {
		t.Fatal("absent", w.Code)
	}
}

// Every reference class blocks, even when its contribution is no longer active.
func TestResourceRemovalReferences(t *testing.T) {
	raw, e := os.ReadFile("../../examples/static/gitea.yaml")
	if e != nil {
		t.Fatal(e)
	}
	facts, e := os.ReadFile("../../examples/static/facts.json")
	if e != nil {
		t.Fatal(e)
	}
	parts := strings.Split(string(raw), "---\n")
	route := "apiVersion: picketx.io/v1alpha1\nkind: HTTPRoute\nmetadata: {name: web, uid: route-web}\nspec:\n  resourceRef: {name: gitea, uid: resource-gitea-v1}\n  endpointName: https\n  host: git.example\n  methods: [GET]\n  paths: [/]\n"
	for name, ref := range map[string]string{
		"expired Grant":              parts[1],
		"revoked Grant":              strings.Replace(parts[1], "spec:\n", "spec:\n  revoked: true\n", 1),
		"Deny":                       strings.Replace(parts[1], "kind: Grant", "kind: NetworkDeny", 1),
		"dynamic Endpoint selection": strings.Replace(parts[1], "endpointNames: [https]", "endpointSelector: {matchLabels: {service: web}}", 1),
		"Binding":                    parts[3], "HTTPRoute": route,
	} {
		t.Run(name, func(t *testing.T) {
			h, s, _ := managementFixture(t)
			policy := parts[0] + "---\n" + ref
			body, _ := json.Marshal(map[string]string{"policy": policy, "facts": string(facts), "reason": "reference seed"})
			if w := request(h, "PUT", "/api/v1/publication", string(body), managementToken, `"0"`); w.Code != 200 {
				t.Fatal(w.Code, w.Body.String())
			}
			w := request(h, "POST", "/api/v1/resources/resource-gitea-v1/remove", resourceRemovalBody, managementToken, `"1"`)
			if w.Code != 409 || !strings.Contains(w.Body.String(), "resource_in_use") {
				t.Fatal(w.Code, w.Body.String())
			}
			d, _ := s.Snapshot()
			a, _ := s.Audit(0, 100)
			if d.Revision != 1 || d.Policy != policy || len(a) != 1 {
				t.Fatal("reference refusal mutated publication")
			}
		})
	}
}

func TestResourceRemovalValidationAndRollback(t *testing.T) {
	h, s, creds := managementFixture(t)
	raw, e := os.ReadFile("../../examples/static/gitea.yaml")
	if e != nil {
		t.Fatal(e)
	}
	facts, e := os.ReadFile("../../examples/static/facts.json")
	if e != nil {
		t.Fatal(e)
	}
	body, _ := json.Marshal(map[string]string{"policy": strings.Split(string(raw), "---\n")[0], "facts": string(facts), "reason": "unused seed"})
	if w := request(h, "PUT", "/api/v1/publication", string(body), managementToken, `"0"`); w.Code != 200 {
		t.Fatal(w.Code, w.Body.String())
	}
	path := "/api/v1/resources/resource-gitea-v1/remove"
	for _, tc := range []struct {
		body, token, rev string
		want             int
	}{
		{resourceRemovalBody, managementToken, "", 428}, {resourceRemovalBody, managementToken, `"0"`, 412}, {resourceRemovalBody, "", `"1"`, 401},
		{`{"reason":"x"}`, managementToken, `"1"`, 400},
		{`{"reason":"x","acknowledge_removal":false}`, managementToken, `"1"`, 400},
		{`{"reason":"x","acknowledge_removal":null}`, managementToken, `"1"`, 400},
		{`{"reason":"x","acknowledge_removal":true,"acknowledge_removal":true}`, managementToken, `"1"`, 400},
		{`{"reason":"x","acknowledge_removal":true,"other":true}`, managementToken, `"1"`, 400},
		{`{"reason":"   ","acknowledge_removal":true}`, managementToken, `"1"`, 400},
	} {
		if w := request(h, "POST", path, tc.body, tc.token, tc.rev); w.Code != tc.want {
			t.Fatal(tc, w.Code, w.Body.String())
		}
	}
	if e = s.GORM().Exec(`CREATE TRIGGER reject_resource_audit BEFORE INSERT ON controller_audit BEGIN SELECT RAISE(FAIL, 'audit unavailable'); END`).Error; e != nil {
		t.Fatal(e)
	}
	if w := request(h, "POST", path, resourceRemovalBody, managementToken, `"1"`); w.Code != 503 {
		t.Fatal(w.Code, w.Body.String())
	}
	d, _ := s.Snapshot()
	if d.Revision != 1 {
		t.Fatal("audit failure committed")
	}
	if e = s.GORM().Exec("DROP TRIGGER reject_resource_audit").Error; e != nil {
		t.Fatal(e)
	}
	if w := request(h, "POST", path, resourceRemovalBody, managementToken, `"1"`); w.Code != 200 {
		t.Fatal(w.Code, w.Body.String())
	}
	d, _ = s.Snapshot()
	if !strings.Contains(d.Policy, "kind: Configuration") {
		t.Fatal("missing explicit empty", d.Policy)
	}
	if w := request(h, "POST", "/api/v1/resources/empty-binding-removal/remove", resourceRemovalBody, managementToken, `"2"`); w.Code != 404 {
		t.Fatal("wrong kind", w.Code)
	}
	data, e := os.ReadFile(creds)
	if e != nil {
		t.Fatal(e)
	}
	if e = os.WriteFile(creds, []byte(strings.ReplaceAll(string(data), "role:admin", "role:viewer")), 0600); e != nil {
		t.Fatal(e)
	}
	if w := request(h, "POST", path, resourceRemovalBody, managementToken, `"2"`); w.Code != 403 {
		t.Fatal("viewer", w.Code)
	}
}

func TestResourceRemovalCookieCSRF(t *testing.T) {
	h, _, _ := browserFixture(t)
	client := browserClient{h: h, cookies: map[string]*http.Cookie{}}
	client.bootstrap(t)
	if w := client.request("POST", "/console/api/v1/login", `{"token":"`+managementToken+`"}`, "http://127.0.0.1:8080", client.csrf); w.Code != 200 {
		t.Fatal(w.Code)
	}
	path := "/console/api/v1/resources/resource-gitea-v1/remove"
	if w := client.request("POST", path, resourceRemovalBody, "http://127.0.0.1:8080", ""); w.Code != 403 {
		t.Fatal("CSRF", w.Code)
	}
	if w := client.request("POST", path, resourceRemovalBody, "http://127.0.0.1:8080", client.csrf); w.Code != 428 {
		t.Fatal("CAS", w.Code)
	}
}

func TestResourceRemovalOpenAPI(t *testing.T) {
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
		op := spec.Paths[prefix+"/resources/{uid}/remove"]["post"]
		if len(op.Security) != 1 || op.Security[0][security] == nil {
			t.Fatal("security", prefix)
		}
		for _, code := range []string{"200", "400", "401", "403", "404", "409", "412", "413", "415", "428", "429", "503"} {
			if _, ok := op.Responses[code]; !ok {
				t.Fatal(prefix, code)
			}
		}
	}
	s := spec.Components.Schemas["controller.RemoveResourceRequest"]
	if s.AdditionalProperties == nil || *s.AdditionalProperties || len(s.Required) != 2 || s.Properties["reason"].MaxLength != 512 || s.Properties["acknowledge_removal"].Const == nil || !*s.Properties["acknowledge_removal"].Const {
		t.Fatal("unsafe removal schema", s)
	}
}
