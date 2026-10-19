// SPDX-License-Identifier: Apache-2.0
package controller

import (
	"encoding/json"
	"net/http"
	"os"
	"strings"
	"testing"

	"github.com/PicketX/PicketX/internal/publication"
)

const routeCreate = `{"name":"gitea-web","resource_name":"gitea","resource_uid":"resource-gitea-v1","endpoint_name":"https","host":"git.example","methods":["GET","HEAD"],"paths":["/","/repositories"],"reason":"publish exact route"}`

// Catches missing route publication, source rewriting and lost durable audit.
func TestHTTPRouteCreateDurableInventory(t *testing.T) {
	t.Run("exact", func(t *testing.T) { testHTTPRouteCreateDurableInventory(t, routeCreate, false) })
	t.Run("prefix-only", func(t *testing.T) {
		testHTTPRouteCreateDurableInventory(t, strings.Replace(routeCreate, `"paths":["/","/repositories"]`, `"path_prefixes":["/repositories"]`, 1), true)
	})
}

func testHTTPRouteCreateDurableInventory(t *testing.T, payload string, prefix bool) {
	h, store, _ := managementFixture(t)
	original, facts := inventorySeed(t, h, false)
	w := request(h, "POST", "/api/v1/http-routes", payload, managementToken, `"1"`)
	if w.Code != 201 || w.Header().Get("ETag") != `"2"` {
		t.Fatal(w.Code, w.Body.String())
	}
	var change struct {
		UID      string `json:"uid"`
		Revision uint64 `json:"revision"`
	}
	if e := json.Unmarshal(w.Body.Bytes(), &change); e != nil {
		t.Fatal(e)
	}
	if !strings.HasPrefix(change.UID, "route-") || change.Revision != 2 {
		t.Fatal(change)
	}
	doc, e := store.Snapshot()
	if e != nil || !strings.HasPrefix(doc.Policy, original) || doc.Facts != facts {
		t.Fatal("source/facts changed", e)
	}
	w = request(h, "GET", "/api/v1/inventory", "", managementToken, "")
	var page struct {
		Routes []struct {
			UID, Name, Host string
			ResourceUID     string `json:"resource_uid"`
			EndpointName    string `json:"endpoint_name"`
			Methods, Paths  []string
			PathPrefixes    []string `json:"path_prefixes"`
		} `json:"http_routes"`
	}
	if e = json.Unmarshal(w.Body.Bytes(), &page); e != nil || w.Code != 200 || len(page.Routes) != 1 {
		t.Fatal(w.Code, w.Body.String(), e)
	}
	r := page.Routes[0]
	wantPaths, wantPrefixes := "/,/repositories", ""
	if prefix {
		wantPaths, wantPrefixes = "", "/repositories"
	}
	if r.UID != change.UID || r.Name != "gitea-web" || r.ResourceUID != "resource-gitea-v1" || r.EndpointName != "https" || r.Host != "git.example" || strings.Join(r.Methods, ",") != "GET,HEAD" || strings.Join(r.Paths, ",") != wantPaths || strings.Join(r.PathPrefixes, ",") != wantPrefixes {
		t.Fatal(r)
	}
	events, e := store.Audit(0, 100)
	if e != nil || len(events) != 2 || events[1].Reason != "publish exact route" || events[1].Actor != "principal:operator" {
		t.Fatal(events, e)
	}
	var files []struct{ Name, File string }
	if e = store.GORM().Raw("PRAGMA database_list").Scan(&files).Error; e != nil {
		t.Fatal(e)
	}
	var path string
	for _, f := range files {
		if f.Name == "main" {
			path = f.File
		}
	}
	if path == "" {
		t.Fatal("missing SQLite file")
	}
	if e = store.Close(); e != nil {
		t.Fatal(e)
	}
	reopened, e := publication.OpenMode(path, "static:example-v1", "database")
	if e != nil {
		t.Fatal(e)
	}
	defer reopened.Close()
	restored, e := reopened.Snapshot()
	if e != nil || restored.Policy != doc.Policy || restored.Revision != 2 || restored.Facts != facts {
		t.Fatal("route not durable", e)
	}
	events, e = reopened.Audit(0, 100)
	if e != nil || len(events) != 2 || events[1].Reason != "publish exact route" {
		t.Fatal("audit not durable", e)
	}
}

// Catches selector widening, client UID injection, stale writes and RBAC bypass.
func TestHTTPRouteRejectsUnsafeCreation(t *testing.T) {
	h, store, creds := managementFixture(t)
	original, facts := inventorySeed(t, h, false)
	cases := []struct{ old, new string }{
		{`"paths":["/","/repositories"],`, ``},
		{`"paths":["/","/repositories"]`, `"path_prefixes":[]`},
		{`"paths":["/","/repositories"]`, `"path_prefixes":null`},
		{`"paths":["/","/repositories"]`, `"path_prefixes":["/repo","/repo"]`},
		{`"paths":["/","/repositories"]`, `"path_prefixes":["/repo/"]`},
		{`"paths":["/","/repositories"]`, `"paths":[],"path_prefixes":["/repo"]`},
		{`"paths":["/","/repositories"]`, `"paths":["/"],"path_prefixes":[]`},
		{`"paths":["/","/repositories"]`, `"paths":["/"],"path_prefixes":["/%72epo"]`},
		{`"gitea-web"`, `"gitea-web","uid":"caller-controlled"`}, {`"gitea-web"`, `"gitea-web","name":"other"`},
		{`resource-gitea-v1`, `wrong`}, {`"https"`, `"missing"`}, {`git.example`, `https://git.example`}, {`git.example`, `*.example`},
		{`["GET","HEAD"]`, `[]`}, {`["GET","HEAD"]`, `null`}, {`["GET","HEAD"]`, `["GET","GET"]`}, {`["GET","HEAD"]`, `["TRACE"]`},
		{`["/","/repositories"]`, `[]`}, {`["/","/repositories"]`, `null`}, {`["/","/repositories"]`, `["/","/"]`},
		{`["/","/repositories"]`, `["/path?query=1"]`}, {`["/","/repositories"]`, `["/%61"]`}, {`["/","/repositories"]`, `["/../"]`},
		{`publish exact route`, `   `}, {`gitea-web`, strings.Repeat("界", 43)},
	}
	for _, c := range cases {
		body := strings.Replace(routeCreate, c.old, c.new, 1)
		if w := request(h, "POST", "/api/v1/http-routes", body, managementToken, `"1"`); w.Code != 400 {
			t.Fatalf("%s => %d %s", body, w.Code, w.Body.String())
		}
	}
	for _, c := range []struct {
		etag string
		code int
	}{{"", 428}, {`"0"`, 412}} {
		if w := request(h, "POST", "/api/v1/http-routes", routeCreate, managementToken, c.etag); w.Code != c.code {
			t.Fatal(w.Code, w.Body.String())
		}
	}
	doc, _ := store.Snapshot()
	events, _ := store.Audit(0, 100)
	if doc.Revision != 1 || doc.Policy != original || doc.Facts != facts || len(events) != 1 {
		t.Fatal("rejection mutated publication")
	}
	// A real UDP endpoint must not be accepted merely because the name exists.
	if w := request(h, "POST", "/api/v1/resources", inventoryResource, managementToken, `"1"`); w.Code != 201 {
		t.Fatal(w.Code)
	}
	doc, _ = store.Snapshot()
	var uid string
	w := request(h, "GET", "/api/v1/inventory", "", managementToken, "")
	var inv struct{ Resources []struct{ Name, UID string } }
	if e := json.Unmarshal(w.Body.Bytes(), &inv); e != nil {
		t.Fatal(e)
	}
	for _, r := range inv.Resources {
		if r.Name == "metrics" {
			uid = r.UID
		}
	}
	body := strings.ReplaceAll(strings.ReplaceAll(strings.ReplaceAll(routeCreate, `"gitea"`, `"metrics"`), "resource-gitea-v1", uid), `"https"`, `"dns"`)
	if w = request(h, "POST", "/api/v1/http-routes", body, managementToken, `"2"`); w.Code != 400 {
		t.Fatal("UDP accepted", w.Code, w.Body.String())
	}
	if w = request(h, "POST", "/api/v1/http-routes", routeCreate, managementToken, `"2"`); w.Code != 201 {
		t.Fatal(w.Code, w.Body.String())
	}
	if w = request(h, "POST", "/api/v1/http-routes", routeCreate, managementToken, `"3"`); w.Code != 400 {
		t.Fatal("duplicate name", w.Code)
	}
	data, e := os.ReadFile(creds)
	if e != nil {
		t.Fatal(e)
	}
	if e = os.WriteFile(creds, []byte(strings.ReplaceAll(string(data), "role:admin", "role:viewer")), 0600); e != nil {
		t.Fatal(e)
	}
	if w = request(h, "POST", "/api/v1/http-routes", routeCreate, managementToken, `"3"`); w.Code != 403 {
		t.Fatal("viewer write", w.Code)
	}
	if w = request(h, "GET", "/api/v1/inventory", "", managementToken, ""); w.Code != 200 {
		t.Fatal("viewer read", w.Code)
	}
}

func TestHTTPRouteAuditFailureRollsBack(t *testing.T) {
	h, store, _ := managementFixture(t)
	inventorySeed(t, h, false)
	before, _ := store.Snapshot()
	if e := store.GORM().Exec("DROP TABLE controller_audit").Error; e != nil {
		t.Fatal(e)
	}
	if w := request(h, "POST", "/api/v1/http-routes", routeCreate, managementToken, `"1"`); w.Code != 503 {
		t.Fatal(w.Code, w.Body.String())
	}
	after, _ := store.Snapshot()
	if before.Policy != after.Policy || before.Revision != after.Revision || before.Facts != after.Facts {
		t.Fatal("partial route published")
	}
}

func TestHTTPRouteBrowserCSRF(t *testing.T) {
	h, _, _ := browserFixture(t)
	client := browserClient{h: h, cookies: map[string]*http.Cookie{}}
	client.bootstrap(t)
	if w := client.request("POST", "/console/api/v1/login", `{"token":"`+managementToken+`"}`, "http://127.0.0.1:8080", client.csrf); w.Code != 200 {
		t.Fatal(w.Code)
	}
	for _, c := range []struct {
		csrf string
		code int
	}{{"", 403}, {client.csrf, 428}} {
		if w := client.request("POST", "/console/api/v1/http-routes", routeCreate, "http://127.0.0.1:8080", c.csrf); w.Code != c.code {
			t.Fatal(w.Code, w.Body.String())
		}
	}
}

func TestHTTPRouteOpenAPIContract(t *testing.T) {
	raw, e := os.ReadFile("../../docs/api/swagger.json")
	if e != nil {
		t.Fatal(e)
	}
	var spec struct {
		Paths      map[string]map[string]json.RawMessage
		Components struct {
			Schemas map[string]struct {
				AdditionalProperties bool
				Required             []string
				AnyOf                []struct{ Required []string }
				Properties           map[string]struct {
					MinItems, MaxItems int
					UniqueItems        bool
					Items              struct {
						MinLength, MaxLength int
						Enum                 []string
					}
				}
			}
		}
	}
	if e = json.Unmarshal(raw, &spec); e != nil {
		t.Fatal(e)
	}
	for _, prefix := range []string{"/api/v1", "/console/api/v1"} {
		if len(spec.Paths[prefix+"/http-routes"]["post"]) == 0 {
			t.Fatal("undocumented route", prefix)
		}
	}
	schema := spec.Components.Schemas["controller.CreateHTTPRouteRequest"]
	if schema.AdditionalProperties || strings.Join(schema.Required, ",") != "endpoint_name,host,methods,name,reason,resource_name,resource_uid" || len(schema.AnyOf) != 2 || strings.Join(schema.AnyOf[0].Required, ",") != "paths" || strings.Join(schema.AnyOf[1].Required, ",") != "path_prefixes" {
		t.Fatal("non-strict required fields", schema)
	}
	methods, paths := schema.Properties["methods"], schema.Properties["paths"]
	if methods.MinItems != 1 || methods.MaxItems != 7 || !methods.UniqueItems || strings.Join(methods.Items.Enum, ",") != "GET,HEAD,POST,PUT,PATCH,DELETE,OPTIONS" {
		t.Fatal("methods contract", methods)
	}
	if paths.MinItems != 1 || paths.MaxItems != 128 || !paths.UniqueItems || paths.Items.MinLength != 1 || paths.Items.MaxLength != 8192 {
		t.Fatal("paths contract", paths)
	}
	prefixes := schema.Properties["path_prefixes"]
	if prefixes.MinItems != 1 || prefixes.MaxItems != 128 || !prefixes.UniqueItems || prefixes.Items.MinLength != 1 || prefixes.Items.MaxLength != 8192 {
		t.Fatal("prefixes contract", prefixes)
	}
}
