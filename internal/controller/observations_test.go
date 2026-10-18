// SPDX-License-Identifier: Apache-2.0
package controller

import (
	"encoding/json"
	"net/http"
	"os"
	"strings"
	"testing"
)

func TestAgentObservationsAuthenticationAndUnavailable(t *testing.T) {
	h, _, creds := managementFixture(t)
	for _, tc := range []struct {
		token, path string
		code        int
	}{{"", "/api/v1/agent-observations", 401}, {managementToken, "/api/v1/agent-observations", 503}, {managementToken, "/api/v1/agent-observations?limit=1", 400}} {
		w := request(h, "GET", tc.path, "", tc.token, "")
		if w.Code != tc.code {
			t.Fatalf("%s: %d %s", tc.path, w.Code, w.Body.String())
		}
	}
	data, _ := os.ReadFile(creds)
	if e := os.WriteFile(creds, []byte(strings.ReplaceAll(string(data), "role:admin", "role:none")), 0600); e != nil {
		t.Fatal(e)
	}
	if w := request(h, "GET", "/api/v1/agent-observations", "", managementToken, ""); w.Code != 403 {
		t.Fatal("denied principal", w.Code)
	}
	bh, _, _ := browserFixture(t)
	client := browserClient{h: bh, cookies: map[string]*http.Cookie{}}
	if w := client.request("GET", "/console/api/v1/agent-observations", "", "", ""); w.Code != 401 {
		t.Fatal(w.Code, w.Body.String())
	}
	client.bootstrap(t)
	if w := client.request("POST", "/console/api/v1/login", `{"token":"`+managementToken+`"}`, "http://127.0.0.1:8080", client.csrf); w.Code != 200 {
		t.Fatal(w.Code, w.Body.String())
	}
	if w := client.request("GET", "/console/api/v1/agent-observations", "", "", ""); w.Code != 503 {
		t.Fatal(w.Code, w.Body.String())
	}
}

func TestAgentObservationsOpenAPI(t *testing.T) {
	raw, e := os.ReadFile("../../docs/api/swagger.json")
	if e != nil {
		t.Fatal(e)
	}
	var spec struct {
		Paths      map[string]map[string]json.RawMessage
		Components struct {
			Schemas map[string]struct {
				Properties map[string]struct {
					Type  string
					OneOf []map[string]string
				}
			}
		}
	}
	if e = json.Unmarshal(raw, &spec); e != nil {
		t.Fatal(e)
	}
	for _, prefix := range []string{"/api/v1", "/console/api/v1"} {
		if len(spec.Paths[prefix+"/agent-observations"]["get"]) == 0 {
			t.Fatal("undocumented observations", prefix)
		}
	}
	if spec.Components.Schemas["agentsync.ReportedState"].Properties["revision"].Type != "string" {
		t.Fatal("unsafe revision schema")
	}
	variants := spec.Components.Schemas["agentsync.NodeObservation"].Properties["report"].OneOf
	if len(variants) != 2 || variants[1]["type"] != "null" {
		t.Fatal("unreported null missing from schema")
	}
}
