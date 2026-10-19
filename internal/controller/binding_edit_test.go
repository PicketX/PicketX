// SPDX-License-Identifier: Apache-2.0
package controller

import (
	"encoding/json"
	"net/http"
	"os"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/PicketX/PicketX/pkg/configuration"
)

const bindingEditPath = "/api/v1/bindings/binding-gitea-v1/endpoints"
const bindingEditBody = `{"endpoint_names":["ssh","https"],"reason":"adjust protection","acknowledge_scope_change":true}`

func bindingEditSeed(t *testing.T, h http.Handler, names string) (string, string) {
	t.Helper()
	raw, e := os.ReadFile("../../examples/static/gitea.yaml")
	if e != nil {
		t.Fatal(e)
	}
	parts := strings.Split(string(raw), "---\n")
	binding := strings.Replace(parts[3], "uid: binding-gitea-v1}", "uid: binding-gitea-v1, annotations: {owner: operations}}", 1)
	if names != "" {
		binding = strings.Replace(binding, "  agents:", "      "+names+"\n  agents:", 1)
	}
	deny := strings.ReplaceAll(strings.Replace(parts[1], "kind: Grant", "kind: NetworkDeny", 1), "grant-browser-v1", "deny-browser-v1")
	deny = strings.Replace(deny, "name: browser", "name: deny-browser", 1)
	return editSeed(t, h, "---\n"+parts[1]+"---\n"+parts[2]+"---\n"+deny+"---\n"+binding)
}

// Catches replacing the Binding/scope, losing other authorization documents,
// non-durable success, noncanonical input ordering and false protected projection.
func TestBindingEndpointEditHTTP(t *testing.T) {
	h, s, _ := managementFixture(t)
	policy, facts := bindingEditSeed(t, h, "endpointNames: [https]")
	before := request(h, "GET", "/api/v1/inventory", "", managementToken, "")
	if before.Code != 200 || !strings.Contains(before.Body.String(), `"endpoint_edit":{"resource_name":"gitea","resource_uid":"resource-gitea-v1","endpoint_names":["https"]}`) {
		t.Fatal("missing safe edit projection", before.Body.String())
	}
	w := request(h, "PUT", bindingEditPath, bindingEditBody, managementToken, `"1"`)
	if w.Code != 200 || w.Header().Get("ETag") != `"2"` {
		t.Fatal(w.Code, w.Body.String())
	}
	var receipt struct {
		UID      string
		Revision uint64
	}
	if e := json.Unmarshal(w.Body.Bytes(), &receipt); e != nil || receipt.UID != "binding-gitea-v1" || receipt.Revision != 2 {
		t.Fatal(receipt, e)
	}
	var d struct {
		Policy, Facts []byte
		Revision      int64
	}
	if e := s.GORM().Raw("SELECT policy,facts,revision FROM controller_publication WHERE id=1").Scan(&d).Error; e != nil {
		t.Fatal(e)
	}
	untouched := strings.Join(strings.Split(policy, "---\n")[:4], "---\n")
	if d.Revision != 2 || string(d.Facts) != facts || !strings.HasPrefix(string(d.Policy), untouched) || !strings.Contains(string(d.Policy), `"annotations":{"owner":"operations"}`) {
		t.Fatal("lost source/facts/metadata", string(d.Policy))
	}
	b, e := configuration.Decode(d.Policy)
	if e != nil {
		t.Fatal(e)
	}
	original, e := configuration.Decode([]byte(policy))
	if e != nil {
		t.Fatal(e)
	}
	got := b.Inventory(configuration.Facts{}).Bindings[0]
	old := original.Inventory(configuration.Facts{}).Bindings[0]
	if got.UID != old.UID || got.Name != old.Name || got.Agents != old.Agents || got.Enforcement != old.Enforcement || got.Resources != `[{"resourceRef":{"name":"gitea","uid":"resource-gitea-v1"},"endpointNames":["https","ssh"]}]` {
		t.Fatal("changed scope/identity or ordering", got)
	}
	c, _ := s.Current()
	projection, e := c.Plan().Projection("agent-a", time.Now())
	if e != nil || len(projection.Protected) != 2 {
		t.Fatal(projection, e)
	}
	w = request(h, "PUT", bindingEditPath, strings.Replace(bindingEditBody, `["ssh","https"]`, `["ssh"]`, 1), managementToken, `"2"`)
	if w.Code != 200 {
		t.Fatal(w.Code, w.Body.String())
	}
	c, _ = s.Current()
	projection, e = c.Plan().Projection("agent-a", time.Now())
	if e != nil || len(projection.Protected) != 1 {
		t.Fatal("removal not projected", projection, e)
	}
	a, e := s.Audit(0, 100)
	if e != nil || len(a) != 3 || a[2].Reason != "adjust protection" || a[2].Actor != "principal:operator" {
		t.Fatal(a, e)
	}
}

func TestBindingEndpointEditRefusalsAndRollback(t *testing.T) {
	h, s, creds := managementFixture(t)
	policy, facts := bindingEditSeed(t, h, "endpointNames: [https]")
	for _, tc := range []struct {
		body, token, rev string
		want             int
	}{
		{bindingEditBody, "", `"1"`, 401}, {bindingEditBody, managementToken, "", 428}, {bindingEditBody, managementToken, `"0"`, 412},
		{strings.Replace(bindingEditBody, `["ssh","https"]`, `[]`, 1), managementToken, `"1"`, 400},
		{strings.Replace(bindingEditBody, `["ssh","https"]`, `null`, 1), managementToken, `"1"`, 400},
		{strings.Replace(bindingEditBody, `["ssh","https"]`, `["missing"]`, 1), managementToken, `"1"`, 400},
		{strings.Replace(bindingEditBody, `["ssh","https"]`, `["ssh","ssh"]`, 1), managementToken, `"1"`, 400},
		{strings.Replace(bindingEditBody, `true`, `false`, 1), managementToken, `"1"`, 400},
		{strings.Replace(bindingEditBody, `true`, `null`, 1), managementToken, `"1"`, 400},
		{strings.Replace(bindingEditBody, `"reason":`, `"reason":"duplicate","reason":`, 1), managementToken, `"1"`, 400},
		{strings.Replace(bindingEditBody, `"reason":`, `"extra":true,"reason":`, 1), managementToken, `"1"`, 400},
		{strings.Replace(bindingEditBody, `"adjust protection"`, `"  "`, 1), managementToken, `"1"`, 400},
	} {
		w := request(h, "PUT", bindingEditPath, tc.body, tc.token, tc.rev)
		if w.Code != tc.want {
			t.Fatal(tc, w.Code, w.Body.String())
		}
	}
	for _, uid := range []string{"missing", "resource-gitea-v1", "grant-browser-v1"} {
		for rev, want := range map[string]int{`"0"`: 412, `"1"`: 404} {
			w := request(h, "PUT", "/api/v1/bindings/"+uid+"/endpoints", bindingEditBody, managementToken, rev)
			if w.Code != want {
				t.Fatal(uid, w.Code, w.Body.String())
			}
		}
	}
	if e := s.GORM().Exec(`CREATE TRIGGER reject_binding_edit BEFORE INSERT ON controller_audit BEGIN SELECT RAISE(FAIL, 'audit unavailable'); END`).Error; e != nil {
		t.Fatal(e)
	}
	if w := request(h, "PUT", bindingEditPath, bindingEditBody, managementToken, `"1"`); w.Code != 503 {
		t.Fatal(w.Code, w.Body.String())
	}
	var d struct {
		Policy, Facts []byte
		Revision      int64
	}
	if e := s.GORM().Raw("SELECT policy,facts,revision FROM controller_publication WHERE id=1").Scan(&d).Error; e != nil {
		t.Fatal(e)
	}
	if d.Revision != 1 || string(d.Policy) != policy || string(d.Facts) != facts {
		t.Fatal("failed write persisted")
	}
	a, e := s.Audit(0, 100)
	if e != nil || len(a) != 1 {
		t.Fatal(a, e)
	}
	data, e := os.ReadFile(creds)
	if e != nil {
		t.Fatal(e)
	}
	if e = os.WriteFile(creds, []byte(strings.ReplaceAll(string(data), "role:admin", "role:viewer")), 0600); e != nil {
		t.Fatal(e)
	}
	if w := request(h, "PUT", bindingEditPath, bindingEditBody, managementToken, `"1"`); w.Code != 403 {
		t.Fatal("viewer", w.Code)
	}
}

func TestBindingEndpointEditUnsupported(t *testing.T) {
	for _, selection := range []string{"", "endpointSelector: {matchLabels: {entry: web}}"} {
		t.Run(selection, func(t *testing.T) {
			h, s, _ := managementFixture(t)
			policy, _ := bindingEditSeed(t, h, selection)
			w := request(h, "PUT", bindingEditPath, bindingEditBody, managementToken, `"1"`)
			if w.Code != 409 || !strings.Contains(w.Body.String(), "binding_not_editable") {
				t.Fatal(w.Code, w.Body.String())
			}
			d, _ := s.Snapshot()
			if d.Policy != policy || d.Revision != 1 {
				t.Fatal("refusal mutated")
			}
			inv := request(h, "GET", "/api/v1/inventory", "", managementToken, "")
			if strings.Contains(inv.Body.String(), "endpoint_edit") {
				t.Fatal("unsupported advertised", inv.Body.String())
			}
		})
	}
}

func TestBindingEndpointEditCookieAndOpenAPI(t *testing.T) {
	h, _, _ := browserFixture(t)
	client := browserClient{h: h, cookies: map[string]*http.Cookie{}}
	client.bootstrap(t)
	if w := client.request("POST", "/console/api/v1/login", `{"token":"`+managementToken+`"}`, "http://127.0.0.1:8080", client.csrf); w.Code != 200 {
		t.Fatal(w.Code)
	}
	if w := client.request("PUT", "/console"+bindingEditPath, bindingEditBody, "http://127.0.0.1:8080", ""); w.Code != 403 {
		t.Fatal("CSRF", w.Code)
	}
	if w := client.request("PUT", "/console"+bindingEditPath, bindingEditBody, "http://127.0.0.1:8080", client.csrf); w.Code != 428 {
		t.Fatal("CAS", w.Code)
	}
	raw, e := os.ReadFile("../../docs/api/swagger.json")
	if e != nil {
		t.Fatal(e)
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
				Properties           map[string]struct{ Const *bool }
			}
		}
	}
	if e = json.Unmarshal(raw, &spec); e != nil {
		t.Fatal(e)
	}
	for prefix, security := range map[string]string{"/api/v1": "ManagementBearer", "/console/api/v1": "BrowserCookie"} {
		op := spec.Paths[prefix+"/bindings/{uid}/endpoints"]["put"]
		if len(op.Security) != 1 || op.Security[0][security] == nil {
			t.Fatal("security", prefix)
		}
		for _, code := range []string{"200", "400", "401", "403", "404", "409", "412", "413", "415", "428", "429", "503"} {
			if _, ok := op.Responses[code]; !ok {
				t.Fatal(prefix, code)
			}
		}
	}
	schema := spec.Components.Schemas["controller.UpdateBindingEndpointsRequest"]
	if schema.AdditionalProperties == nil || *schema.AdditionalProperties || !reflect.DeepEqual(schema.Required, []string{"acknowledge_scope_change", "endpoint_names", "reason"}) || schema.Properties["acknowledge_scope_change"].Const == nil || !*schema.Properties["acknowledge_scope_change"].Const {
		t.Fatal("strict schema", schema)
	}
}
