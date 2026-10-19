// SPDX-License-Identifier: Apache-2.0
package controller

import (
	"encoding/json"
	"net/http"
	"os"
	"reflect"
	"strings"
	"testing"

	"github.com/PicketX/PicketX/pkg/configuration"
)

const resourceNetworkBody = `{"reason":"correct network parameters","endpoints":[{"name":"ssh","protocol":"UDP","port":2222},{"name":"https","protocol":"TCP","port":8443,"addresses":["192.0.2.10","2001:db8::10"]}]}`
const resourceNetworkPath = "/api/v1/resources/resource-gitea-v1/network"

func editSeed(t *testing.T, h http.Handler, suffix string) (string, string) {
	t.Helper()
	raw, e := os.ReadFile("../../examples/static/gitea.yaml")
	if e != nil {
		t.Fatal(e)
	}
	facts, e := os.ReadFile("../../examples/static/facts.json")
	if e != nil {
		t.Fatal(e)
	}
	policy := strings.Split(string(raw), "---\n")[0]
	policy = strings.Replace(policy, "uid: resource-gitea-v1}", "uid: resource-gitea-v1, labels: {team: platform}}", 1)
	policy = strings.Replace(policy, "- name: https", "- name: https\n      labels: {entry: web}", 1) + suffix
	body, _ := json.Marshal(map[string]string{"policy": policy, "facts": string(facts), "reason": "edit seed"})
	if w := request(h, "PUT", "/api/v1/publication", string(body), managementToken, `"0"`); w.Code != 200 {
		t.Fatal(w.Code, w.Body.String())
	}
	return policy, string(facts)
}

// Catches missing mutation, metadata loss, input-driven reorder and cached-only persistence.
func TestResourceNetworkEditHTTP(t *testing.T) {
	h, s, _ := managementFixture(t)
	sibling := "---\n# untouched sibling\napiVersion: picketx.io/v1alpha1\nkind: Resource\nmetadata: {name: sibling, uid: sibling-v1}\nspec: {endpoints: [{name: dns, network: {protocol: UDP, port: 53}}]}\n"
	_, facts := editSeed(t, h, sibling)
	w := request(h, "PUT", resourceNetworkPath, resourceNetworkBody, managementToken, `"1"`)
	if w.Code != 200 || w.Header().Get("ETag") != `"2"` {
		t.Fatal(w.Code, w.Body.String())
	}
	var receipt struct {
		UID      string
		Revision uint64
	}
	if e := json.Unmarshal(w.Body.Bytes(), &receipt); e != nil || receipt.UID != "resource-gitea-v1" || receipt.Revision != 2 {
		t.Fatal(receipt, e)
	}
	var d struct {
		Policy, Facts []byte
		Revision      int64
	}
	if e := s.GORM().Raw("SELECT policy,facts,revision FROM controller_publication WHERE id=1").Scan(&d).Error; e != nil {
		t.Fatal(e)
	}
	if d.Revision != 2 || string(d.Facts) != facts || !strings.Contains(string(d.Policy), sibling) || !strings.Contains(string(d.Policy), `"team":"platform"`) || !strings.Contains(string(d.Policy), `"entry":"web"`) {
		t.Fatal("source/metadata/facts lost", string(d.Policy))
	}
	b, e := configuration.Decode(d.Policy)
	if e != nil {
		t.Fatal(e)
	}
	rs := b.Inventory(configuration.Facts{}).Resources
	var target configuration.InventoryResource
	for _, r := range rs {
		if r.UID == receipt.UID {
			target = r
		}
	}
	want := []configuration.InventoryEndpoint{{Name: "https", Protocol: "TCP", Port: 8443, Addresses: []string{"192.0.2.10", "2001:db8::10"}}, {Name: "ssh", Protocol: "UDP", Port: 2222}}
	if !reflect.DeepEqual(target.Endpoints, want) || target.Name != "gitea" {
		t.Fatal(target)
	}
	a, e := s.Audit(0, 100)
	if e != nil || len(a) != 2 || a[1].Reason != "correct network parameters" || a[1].Actor != "principal:operator" {
		t.Fatal(a, e)
	}
	if w = request(h, "PUT", resourceNetworkPath, resourceNetworkBody, managementToken, `"1"`); w.Code != 412 {
		t.Fatal(w.Code)
	}
}

// Removing the reference guard must fail even for inactive or zero-match declarations.
func TestResourceNetworkEditReferences(t *testing.T) {
	raw, e := os.ReadFile("../../examples/static/gitea.yaml")
	if e != nil {
		t.Fatal(e)
	}
	parts := strings.Split(string(raw), "---\n")
	for name, ref := range map[string]string{
		"expired": parts[1], "revoked": strings.Replace(parts[1], "spec:\n", "spec:\n  revoked: true\n", 1),
		"deny":       strings.Replace(parts[1], "kind: Grant", "kind: NetworkDeny", 1),
		"zero match": strings.Replace(parts[1], "endpointNames: [https]", "endpointSelector: {matchLabels: {missing: label}}", 1),
		"binding":    parts[3],
		"route":      "apiVersion: picketx.io/v1alpha1\nkind: HTTPRoute\nmetadata: {name: route, uid: route-v1}\nspec: {resourceRef: {name: gitea, uid: resource-gitea-v1}, endpointName: https, host: git.example, methods: [GET], paths: [/]}\n",
	} {
		t.Run(name, func(t *testing.T) {
			h, s, _ := managementFixture(t)
			policy, _ := editSeed(t, h, "---\n"+ref)
			if name == "binding" {
				wrong := request(h, "PUT", "/api/v1/resources/binding-gitea-v1/network", resourceNetworkBody, managementToken, `"1"`)
				if wrong.Code != 404 {
					t.Fatal("wrong kind", wrong.Code, wrong.Body.String())
				}
			}
			w := request(h, "PUT", resourceNetworkPath, resourceNetworkBody, managementToken, `"1"`)
			if w.Code != 409 || !strings.Contains(w.Body.String(), "resource_in_use") {
				t.Fatal(w.Code, w.Body.String())
			}
			d, _ := s.Snapshot()
			a, _ := s.Audit(0, 100)
			if d.Policy != policy || d.Revision != 1 || len(a) != 1 {
				t.Fatal("refusal changed state")
			}
		})
	}
}

func TestResourceNetworkEditValidationAndRollback(t *testing.T) {
	h, s, creds := managementFixture(t)
	policy, facts := editSeed(t, h, "")
	for _, tc := range []struct {
		body, token, rev string
		want             int
	}{
		{resourceNetworkBody, "", `"1"`, 401}, {resourceNetworkBody, managementToken, "", 428},
		{resourceNetworkBody, managementToken, `"0"`, 412},
		{strings.Replace(resourceNetworkBody, `"reason":`, `"other":true,"reason":`, 1), managementToken, `"1"`, 400},
		{strings.Replace(resourceNetworkBody, `"reason":`, `"reason":"duplicate","reason":`, 1), managementToken, `"1"`, 400},
		{strings.Replace(resourceNetworkBody, `"correct network parameters"`, `"   "`, 1), managementToken, `"1"`, 400},
		{`{"reason":"x","endpoints":[]}`, managementToken, `"1"`, 400},
		{`{"reason":"x","endpoints":null}`, managementToken, `"1"`, 400},
		{strings.Replace(resourceNetworkBody, `["192.0.2.10","2001:db8::10"]`, `null`, 1), managementToken, `"1"`, 400},
		{strings.Replace(resourceNetworkBody, `["192.0.2.10","2001:db8::10"]`, `[]`, 1), managementToken, `"1"`, 400},
		{strings.Replace(resourceNetworkBody, `"ssh"`, `"renamed"`, 1), managementToken, `"1"`, 400},
		{strings.Replace(resourceNetworkBody, `"ssh"`, `"https"`, 1), managementToken, `"1"`, 400},
		{strings.Replace(resourceNetworkBody, `2222`, `0`, 1), managementToken, `"1"`, 400},
		{`{"reason":"x","endpoints":[{"name":"https","protocol":"TCP","port":443}]}`, managementToken, `"1"`, 400},
	} {
		if w := request(h, "PUT", resourceNetworkPath, tc.body, tc.token, tc.rev); w.Code != tc.want {
			t.Fatal(tc, w.Code, w.Body.String())
		}
	}
	for _, uid := range []string{"missing", "same-name-recreated"} {
		if w := request(h, "PUT", "/api/v1/resources/"+uid+"/network", resourceNetworkBody, managementToken, `"0"`); w.Code != 412 {
			t.Fatal(w.Code)
		}
		if w := request(h, "PUT", "/api/v1/resources/"+uid+"/network", resourceNetworkBody, managementToken, `"1"`); w.Code != 404 {
			t.Fatal(w.Code)
		}
	}
	if e := s.GORM().Exec(`CREATE TRIGGER reject_network_audit BEFORE INSERT ON controller_audit BEGIN SELECT RAISE(FAIL, 'audit unavailable'); END`).Error; e != nil {
		t.Fatal(e)
	}
	if w := request(h, "PUT", resourceNetworkPath, resourceNetworkBody, managementToken, `"1"`); w.Code != 503 {
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
		t.Fatal("audit rollback not durable")
	}
	if e := s.GORM().Exec("DROP TRIGGER reject_network_audit").Error; e != nil {
		t.Fatal(e)
	}
	data, e := os.ReadFile(creds)
	if e != nil {
		t.Fatal(e)
	}
	if e = os.WriteFile(creds, []byte(strings.ReplaceAll(string(data), "role:admin", "role:viewer")), 0600); e != nil {
		t.Fatal(e)
	}
	if w := request(h, "PUT", resourceNetworkPath, resourceNetworkBody, managementToken, `"1"`); w.Code != 403 {
		t.Fatal("viewer", w.Code)
	}
}

func TestResourceNetworkEditCookieCSRF(t *testing.T) {
	h, _, _ := browserFixture(t)
	client := browserClient{h: h, cookies: map[string]*http.Cookie{}}
	client.bootstrap(t)
	if w := client.request("POST", "/console/api/v1/login", `{"token":"`+managementToken+`"}`, "http://127.0.0.1:8080", client.csrf); w.Code != 200 {
		t.Fatal(w.Code)
	}
	path := "/console" + resourceNetworkPath
	if w := client.request("PUT", path, resourceNetworkBody, "http://127.0.0.1:8080", ""); w.Code != 403 {
		t.Fatal(w.Code)
	}
	if w := client.request("PUT", path, resourceNetworkBody, "http://127.0.0.1:8080", client.csrf); w.Code != 428 {
		t.Fatal(w.Code)
	}
}

func TestResourceNetworkEditOpenAPI(t *testing.T) {
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
			}
		}
	}
	if err = json.Unmarshal(raw, &spec); err != nil {
		t.Fatal(err)
	}
	for prefix, security := range map[string]string{"/api/v1": "ManagementBearer", "/console/api/v1": "BrowserCookie"} {
		op := spec.Paths[prefix+"/resources/{uid}/network"]["put"]
		if len(op.Security) != 1 || op.Security[0][security] == nil {
			t.Fatal("missing security", prefix)
		}
		for _, code := range []string{"200", "400", "401", "403", "404", "409", "412", "413", "415", "428", "429", "503"} {
			if _, ok := op.Responses[code]; !ok {
				t.Fatal(prefix, code)
			}
		}
	}
	s := spec.Components.Schemas["controller.UpdateResourceNetworkRequest"]
	if s.AdditionalProperties == nil || *s.AdditionalProperties || len(s.Required) != 2 {
		t.Fatal("strict schema missing", s)
	}
}
