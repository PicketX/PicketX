// SPDX-License-Identifier: Apache-2.0
package controller

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"
)

func TestBindingRemovalBrowserAndContract(t *testing.T) {
	h, _, _ := browserFixture(t)
	client := browserClient{h: h, cookies: map[string]*http.Cookie{}}
	client.bootstrap(t)
	if w := client.request("POST", "/console/api/v1/login", `{"token":"`+managementToken+`"}`, "http://127.0.0.1:8080", client.csrf); w.Code != 200 {
		t.Fatal(w.Code)
	}
	path := "/console/api/v1/bindings/binding-gitea-v1/remove"
	if w := client.request("POST", path, removalBody, "http://127.0.0.1:8080", ""); w.Code != 403 {
		t.Fatal("CSRF", w.Code)
	}
	if w := client.request("POST", path, removalBody, "http://127.0.0.1:8080", client.csrf); w.Code != 428 {
		t.Fatal("CAS", w.Code)
	}
	raw, e := os.ReadFile("../../docs/api/swagger.json")
	if e != nil {
		t.Fatal(e)
	}
	var spec struct {
		Paths map[string]map[string]struct {
			Security  []map[string][]string `json:"security"`
			Responses map[string]any        `json:"responses"`
		} `json:"paths"`
		Components struct {
			Schemas map[string]struct {
				AdditionalProperties *bool    `json:"additionalProperties"`
				Required             []string `json:"required"`
				Properties           map[string]struct {
					Const     *bool `json:"const"`
					MaxLength int   `json:"maxLength"`
				} `json:"properties"`
			} `json:"schemas"`
		} `json:"components"`
	}
	if e = json.Unmarshal(raw, &spec); e != nil {
		t.Fatal(e)
	}
	for prefix, security := range map[string]string{"/api/v1": "ManagementBearer", "/console/api/v1": "BrowserCookie"} {
		op := spec.Paths[prefix+"/bindings/{uid}/remove"]["post"]
		if len(op.Security) != 1 || op.Security[0][security] == nil {
			t.Fatal("missing security", prefix)
		}
		for _, code := range []string{"200", "400", "401", "403", "404", "412", "413", "415", "428", "429", "503"} {
			if _, ok := op.Responses[code]; !ok {
				t.Fatal(prefix, code)
			}
		}
	}
	schema := spec.Components.Schemas["controller.RemoveBindingRequest"]
	if schema.AdditionalProperties == nil || *schema.AdditionalProperties || len(schema.Required) != 2 || schema.Properties["reason"].MaxLength != 512 || schema.Properties["acknowledge_unprotection"].Const == nil || !*schema.Properties["acknowledge_unprotection"].Const {
		t.Fatal("unsafe acknowledgement schema", schema)
	}
}

const removalBody = `{"reason":"retire protection","acknowledge_unprotection":true}`

func TestBindingRemovalRetainsBoundaryCommentsHTTP(t *testing.T) {
	h, store, _ := managementFixture(t)
	original, err := os.ReadFile("../../examples/static/gitea.yaml")
	if err != nil {
		t.Fatal(err)
	}
	facts, err := os.ReadFile("../../examples/static/facts.json")
	if err != nil {
		t.Fatal(err)
	}
	parts := strings.Split(string(original), "---\n")
	suffix := "\n# preserve operator rationale\n---\n" + strings.Join(parts[:3], "---\n")
	policy := parts[3] + "...\n" + suffix
	raw, _ := json.Marshal(map[string]string{"policy": policy, "facts": string(facts), "reason": "annotated publication"})
	if w := request(h, "PUT", "/api/v1/publication", string(raw), managementToken, `"0"`); w.Code != 200 {
		t.Fatal(w.Code, w.Body.String())
	}
	w := request(h, "POST", "/api/v1/bindings/binding-gitea-v1/remove", removalBody, managementToken, `"1"`)
	if w.Code != 200 || w.Header().Get("ETag") != `"2"` {
		t.Fatal(w.Code, w.Body.String())
	}
	doc, err := store.Snapshot()
	if err != nil || doc.Revision != 2 || doc.Policy != suffix || doc.Facts != string(facts) {
		t.Fatal("persisted source/facts mismatch", err, doc.Policy)
	}
	audit, err := store.Audit(0, 100)
	if err != nil || len(audit) != 2 || audit[1].Reason != "retire protection" {
		t.Fatal("audit", audit, err)
	}
}

func TestBindingRemovalHTTP(t *testing.T) {
	h, store, _ := managementFixture(t)
	_, facts := inventorySeed(t, h, false)
	create := `{"name":"overlap","resource_name":"gitea","resource_uid":"resource-gitea-v1","endpoint_names":["https","ssh"],"agent_name":"agent-a","context_id":"host-v1","reason":"overlapping protection"}`
	w := request(h, "POST", "/api/v1/bindings", create, managementToken, `"1"`)
	if w.Code != 201 {
		t.Fatal(w.Code, w.Body.String())
	}
	var added struct {
		UID string `json:"uid"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &added); err != nil {
		t.Fatal(err)
	}
	before, _ := store.Snapshot()
	for i, uid := range []string{"binding-gitea-v1", added.UID} {
		w = request(h, "POST", "/api/v1/bindings/"+uid+"/remove", removalBody, managementToken, fmt.Sprintf(`"%d"`, i+2))
		if w.Code != 200 || w.Header().Get("ETag") != fmt.Sprintf(`"%d"`, i+3) {
			t.Fatal("remove", w.Code, w.Body.String())
		}
		c, _ := store.Current()
		projection, err := c.Plan().Projection("agent-a", time.Now())
		if err != nil {
			t.Fatal(err)
		}
		if i == 0 && len(projection.Protected) != 2 {
			t.Fatal("lost other Binding protection", projection)
		}
		if i == 1 && len(projection.Protected) != 0 {
			t.Fatal("last Binding not withdrawn", projection)
		}
	}
	doc, _ := store.Snapshot()
	if doc.Facts != facts || !strings.Contains(doc.Policy, "grant-browser-v1") || !strings.Contains(doc.Policy, "grant-ssh6-v1") || strings.Contains(doc.Policy, "revoked: true") {
		t.Fatal("unrelated policy changed")
	}
	if !strings.HasPrefix(before.Policy, strings.TrimSuffix(doc.Policy, "\n")) {
		t.Fatal("surviving source changed")
	}
	audit, err := store.Audit(0, 100)
	if err != nil || len(audit) != 4 || audit[3].Reason != "retire protection" || audit[3].Actor != "principal:operator" {
		t.Fatal(audit, err)
	}
	if w = request(h, "POST", "/api/v1/bindings/"+added.UID+"/remove", removalBody, managementToken, `"3"`); w.Code != 412 {
		t.Fatal("stale before absent", w.Code)
	}
	if w = request(h, "POST", "/api/v1/bindings/"+added.UID+"/remove", removalBody, managementToken, `"4"`); w.Code != 404 {
		t.Fatal("absent", w.Code)
	}
}

func TestBindingRemovalRejectsUnsafeWrites(t *testing.T) {
	h, store, creds := managementFixture(t)
	inventorySeed(t, h, false)
	path := "/api/v1/bindings/binding-gitea-v1/remove"
	for _, tc := range []struct {
		body, uid, token, revision string
		status                     int
	}{
		{removalBody, "binding-gitea-v1", managementToken, "", 428},
		{removalBody, "binding-gitea-v1", managementToken, `"0"`, 412},
		{removalBody, "binding-gitea-v1", "", `"1"`, 401},
		{removalBody, "grant-browser-v1", managementToken, `"1"`, 404},
		{removalBody, "missing", managementToken, `"1"`, 404},
		{`{"reason":"x"}`, "binding-gitea-v1", managementToken, `"1"`, 400},
		{`{"reason":"x","acknowledge_unprotection":false}`, "binding-gitea-v1", managementToken, `"1"`, 400},
		{`{"reason":"x","acknowledge_unprotection":null}`, "binding-gitea-v1", managementToken, `"1"`, 400},
		{`{"reason":"x","acknowledge_unprotection":true,"acknowledge_unprotection":true}`, "binding-gitea-v1", managementToken, `"1"`, 400},
		{`{"reason":"x","acknowledge_unprotection":true,"other":true}`, "binding-gitea-v1", managementToken, `"1"`, 400},
		{strings.Replace(removalBody, "retire protection", "   ", 1), "binding-gitea-v1", managementToken, `"1"`, 400},
	} {
		w := request(h, "POST", "/api/v1/bindings/"+tc.uid+"/remove", tc.body, tc.token, tc.revision)
		if w.Code != tc.status {
			t.Fatalf("%+v: %d %s", tc, w.Code, w.Body.String())
		}
	}
	before, _ := store.Snapshot()
	if err := store.GORM().Exec(`CREATE TRIGGER reject_removal_audit BEFORE INSERT ON controller_audit BEGIN SELECT RAISE(FAIL, 'audit unavailable'); END`).Error; err != nil {
		t.Fatal(err)
	}
	w := request(h, "POST", path, removalBody, managementToken, `"1"`)
	if w.Code != 503 {
		t.Fatal("audit failure", w.Code, w.Body.String())
	}
	after, _ := store.Snapshot()
	if before != after || after.Revision != 1 {
		t.Fatal("failed mutation committed")
	}
	data, err := os.ReadFile(creds)
	if err != nil {
		t.Fatal(err)
	}
	if err = os.WriteFile(creds, []byte(strings.ReplaceAll(string(data), "role:admin", "role:viewer")), 0600); err != nil {
		t.Fatal(err)
	}
	if w = request(h, "POST", path, removalBody, managementToken, `"1"`); w.Code != 403 {
		t.Fatal("viewer", w.Code)
	}
}
