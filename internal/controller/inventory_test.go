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

	"github.com/PicketX/PicketX/pkg/configuration"
)

const inventoryResource = `{"name":"metrics","endpoints":[{"name":"web","protocol":"TCP","port":8443,"addresses":["192.0.2.100","2001:db8::100"]},{"name":"dns","protocol":"UDP","port":5353}],"reason":"create metrics"}`

func inventorySeed(t *testing.T, h http.Handler, empty bool) (string, string) {
	t.Helper()
	p, e := os.ReadFile("../../examples/static/gitea.yaml")
	if e != nil {
		t.Fatal(e)
	}
	f, e := os.ReadFile("../../examples/static/facts.json")
	if e != nil {
		t.Fatal(e)
	}
	if empty {
		p = []byte("apiVersion: picketx.io/v1alpha1\nkind: Configuration\nmetadata: {name: empty, uid: empty-v1}\nspec: {empty: true}\n")
	}
	raw, _ := json.Marshal(map[string]string{"policy": string(p), "facts": string(f), "reason": "inventory seed"})
	if w := request(h, "PUT", "/api/v1/publication", string(raw), managementToken, `"0"`); w.Code != 200 {
		t.Fatal(w.Code, w.Body.String())
	}
	return string(p), string(f)
}

func TestInventoryResourceAndBindingHTTP(t *testing.T) {
	h, store, _ := managementFixture(t)
	original, facts := inventorySeed(t, h, false)
	w := request(h, "POST", "/api/v1/resources", inventoryResource, managementToken, `"1"`)
	if w.Code != 201 {
		t.Fatal("resource creation", w.Code, w.Body.String())
	}
	var created struct {
		UID      string `json:"uid"`
		Revision uint64 `json:"revision"`
	}
	if e := json.Unmarshal(w.Body.Bytes(), &created); e != nil {
		t.Fatal(e)
	}
	if !strings.HasPrefix(created.UID, "resource-") || created.Revision != 2 {
		t.Fatal(created)
	}
	binding := fmt.Sprintf(`{"name":"metrics-host","resource_name":"metrics","resource_uid":%q,"endpoint_names":["web","dns"],"agent_name":"agent-a","context_id":"host-v1","reason":"place metrics"}`, created.UID)
	w = request(h, "POST", "/api/v1/bindings", binding, managementToken, `"2"`)
	if w.Code != 201 || w.Header().Get("ETag") != `"3"` {
		t.Fatal(w.Code, w.Body.String())
	}
	doc, e := store.Snapshot()
	if e != nil || doc.Facts != facts || !strings.HasPrefix(doc.Policy, original) {
		t.Fatal("unrelated source changed", e)
	}
	b, e := configuration.Decode([]byte(doc.Policy))
	if e != nil {
		t.Fatal(e)
	}
	f, e := configuration.DecodeFacts([]byte(facts))
	if e != nil {
		t.Fatal(e)
	}
	c, e := configuration.Compile(b, f)
	if e != nil {
		t.Fatal(e)
	}
	projection, e := c.Plan().Projection("agent-a", time.Now())
	if e != nil {
		t.Fatal(e)
	}
	count := 0
	for _, p := range projection.Protected {
		if p.Key.Entry.ResourceName == created.UID {
			count++
			if p.Context.ID != "host-v1" || p.Context.ProfileRef != "host" || p.Context.Handling != "graceful" || len(p.Matches) != 2 {
				t.Fatalf("wrong placement: %+v", p)
			}
		}
	}
	if count != 2 {
		t.Fatal("not projected", count)
	}
	for _, a := range projection.Allow {
		if a.Target.Entry.ResourceName == created.UID {
			t.Fatal("creation granted access")
		}
	}
	w = request(h, "GET", "/api/v1/inventory", "", managementToken, "")
	if w.Code != 200 || w.Header().Get("ETag") != `"3"` || w.Header().Get("Cache-Control") != "no-store" {
		t.Fatal(w.Code, w.Body.String())
	}
	var page struct {
		Resources []struct {
			UID       string `json:"uid"`
			Endpoints []struct {
				Addresses []string `json:"addresses"`
			} `json:"endpoints"`
		} `json:"resources"`
		Bindings []any `json:"bindings"`
		Contexts []struct {
			AgentName string `json:"agent_name"`
			ID        string `json:"id"`
		} `json:"contexts"`
	}
	if e = json.Unmarshal(w.Body.Bytes(), &page); e != nil || len(page.Resources) != 2 || len(page.Bindings) != 2 || len(page.Contexts) != 1 || page.Contexts[0].AgentName != "agent-a" {
		t.Fatal(page, e)
	}
	found := false
	for _, r := range page.Resources {
		if r.UID == created.UID {
			found = len(r.Endpoints[0].Addresses) == 2 || len(r.Endpoints[1].Addresses) == 2
		}
	}
	if !found {
		t.Fatal("destination restrictions lost")
	}
	events, e := store.Audit(0, 100)
	if e != nil || len(events) != 3 || events[2].Reason != "place metrics" || events[2].Actor != "principal:operator" {
		t.Fatal(events, e)
	}
}

func TestInventoryRejectsUnsafeCreates(t *testing.T) {
	h, store, creds := managementFixture(t)
	inventorySeed(t, h, false)
	if w := request(h, "POST", "/api/v1/resources", inventoryResource, managementToken, ""); w.Code != 428 {
		t.Fatal(w.Code)
	}
	for _, body := range []string{
		strings.Replace(inventoryResource, `8443`, `0`, 1), strings.Replace(inventoryResource, `8443`, `65536`, 1), strings.Replace(inventoryResource, `"TCP"`, `"tcp"`, 1),
		strings.Replace(inventoryResource, `["192.0.2.100","2001:db8::100"]`, `[]`, 1), strings.Replace(inventoryResource, `["192.0.2.100","2001:db8::100"]`, `null`, 1), strings.Replace(inventoryResource, `192.0.2.100`, `hostname.test`, 1),
		strings.Replace(inventoryResource, `"dns"`, `"web"`, 1), strings.Replace(inventoryResource, `"metrics"`, `"gitea"`, 1), strings.Replace(inventoryResource, `"name":"metrics"`, `"name":"metrics","unexpected":true`, 1), strings.Replace(inventoryResource, `"name":"metrics"`, `"name":"metrics","name":"other"`, 1), strings.Replace(inventoryResource, `create metrics`, `   `, 1),
	} {
		if w := request(h, "POST", "/api/v1/resources", body, managementToken, `"1"`); w.Code != 400 {
			t.Fatal(body, w.Code, w.Body.String())
		}
	}
	binding := `{"name":"second","resource_name":"gitea","resource_uid":"resource-gitea-v1","endpoint_names":["https"],"agent_name":"agent-a","context_id":"host-v1","reason":"place"}`
	for _, body := range []string{strings.Replace(binding, `host-v1`, `missing`, 1), strings.Replace(binding, `agent-a`, `missing`, 1), strings.Replace(binding, `resource-gitea-v1`, `wrong`, 1), strings.Replace(binding, `["https"]`, `["https","https"]`, 1), strings.Replace(binding, `["https"]`, `[]`, 1), strings.Replace(binding, `"context_id"`, `"profile_ref"`, 1)} {
		if w := request(h, "POST", "/api/v1/bindings", body, managementToken, `"1"`); w.Code != 400 {
			t.Fatal(body, w.Code, w.Body.String())
		}
	}
	if w := request(h, "POST", "/api/v1/bindings", binding, managementToken, `"0"`); w.Code != 412 {
		t.Fatal(w.Code)
	}
	doc, _ := store.Snapshot()
	if doc.Revision != 1 {
		t.Fatal("rejected mutation changed state")
	}
	data, e := os.ReadFile(creds)
	if e != nil {
		t.Fatal(e)
	}
	if e = os.WriteFile(creds, []byte(strings.ReplaceAll(string(data), "role:admin", "role:viewer")), 0600); e != nil {
		t.Fatal(e)
	}
	for _, path := range []string{"/api/v1/resources", "/api/v1/bindings"} {
		if w := request(h, "POST", path, `{}`, managementToken, `"1"`); w.Code != 403 {
			t.Fatal(path, w.Code)
		}
	}
	if w := request(h, "GET", "/api/v1/inventory", "", "", ""); w.Code != 401 {
		t.Fatal(w.Code)
	}
}

func TestInventoryEmptyAndAuditRollback(t *testing.T) {
	h, store, _ := managementFixture(t)
	_, facts := inventorySeed(t, h, true)
	if w := request(h, "POST", "/api/v1/resources", inventoryResource, managementToken, `"1"`); w.Code != 201 {
		t.Fatal(w.Code, w.Body.String())
	}
	before, _ := store.Snapshot()
	if strings.Contains(before.Policy, `"kind":"Configuration"`) || before.Facts != facts {
		t.Fatal("empty marker/facts")
	}
	if e := store.GORM().Exec("DROP TABLE controller_audit").Error; e != nil {
		t.Fatal(e)
	}
	if w := request(h, "POST", "/api/v1/resources", strings.Replace(inventoryResource, "metrics", "other", -1), managementToken, `"2"`); w.Code != 503 {
		t.Fatal(w.Code, w.Body.String())
	}
	after, _ := store.Snapshot()
	if before.Policy != after.Policy || before.Revision != after.Revision {
		t.Fatal("unaudited resource persisted")
	}
}

func TestInventoryBrowserCSRFAndGrantSchema(t *testing.T) {
	h, _, _ := browserFixture(t)
	client := browserClient{h: h, cookies: map[string]*http.Cookie{}}
	client.bootstrap(t)
	if w := client.request("POST", "/console/api/v1/login", `{"token":"`+managementToken+`"}`, "http://127.0.0.1:8080", client.csrf); w.Code != 200 {
		t.Fatal(w.Code)
	}
	for _, path := range []string{"/console/api/v1/resources", "/console/api/v1/bindings"} {
		if w := client.request("POST", path, `{}`, "http://127.0.0.1:8080", ""); w.Code != 403 {
			t.Fatal(path, w.Code)
		}
		if w := client.request("POST", path, `{}`, "http://127.0.0.1:8080", client.csrf); w.Code != 428 {
			t.Fatal(path, w.Code)
		}
	}
	raw, e := os.ReadFile("../../docs/api/swagger.json")
	if e != nil {
		t.Fatal(e)
	}
	var spec struct {
		Components struct {
			Schemas map[string]struct {
				Properties map[string]struct {
					UniqueItems bool                               `json:"uniqueItems"`
					Items       struct{ MinLength, MaxLength int } `json:"items"`
				} `json:"properties"`
			} `json:"schemas"`
		} `json:"components"`
	}
	if e = json.Unmarshal(raw, &spec); e != nil {
		t.Fatal(e)
	}
	names := spec.Components.Schemas["controller.IssueGrantRequest"].Properties["endpoint_names"]
	if !names.UniqueItems || names.Items.MinLength != 1 || names.Items.MaxLength != 1024 {
		t.Fatal("Grant schema weaker than runtime", names)
	}
}
