// SPDX-License-Identifier: Apache-2.0
package controller

import (
	"encoding/json"
	"net/http"
	"os"
	"testing"
)

func TestStructuredGrantOpenAPIAndBrowserCSRF(t *testing.T) {
	raw, err := os.ReadFile("../../docs/api/swagger.json")
	if err != nil {
		t.Fatal(err)
	}
	var spec struct {
		Paths map[string]map[string]struct {
			Security  []map[string][]string `json:"security"`
			Responses map[string]any        `json:"responses"`
		} `json:"paths"`
		Components struct {
			SecuritySchemes map[string]any `json:"securitySchemes"`
			Schemas         map[string]struct {
				AdditionalProperties bool     `json:"additionalProperties"`
				Required             []string `json:"required"`
			} `json:"schemas"`
		} `json:"components"`
	}
	if err = json.Unmarshal(raw, &spec); err != nil {
		t.Fatal(err)
	}
	for _, prefix := range []string{"/api/v1", "/console/api/v1"} {
		for path, methods := range map[string][]string{"/grants": {"get", "post"}, "/grants/{uid}/revoke": {"post"}} {
			for _, method := range methods {
				op := spec.Paths[prefix+path][method]
				if len(op.Security) != 1 || len(op.Responses) < 7 {
					t.Fatal(prefix, path, method, op)
				}
				for name := range op.Security[0] {
					if _, ok := spec.Components.SecuritySchemes[name]; !ok {
						t.Fatal("undefined security scheme", name)
					}
				}
			}
		}
	}
	req := spec.Components.Schemas["controller.IssueGrantRequest"]
	if req.AdditionalProperties || len(req.Required) != 7 {
		t.Fatal("inaccurate strict DTO", req)
	}
	h, _, _ := browserFixture(t)
	client := browserClient{h: h, cookies: map[string]*http.Cookie{}}
	client.bootstrap(t)
	if w := client.request("POST", "/console/api/v1/login", `{"token":"`+managementToken+`"}`, "http://127.0.0.1:8080", client.csrf); w.Code != 200 {
		t.Fatal(w.Code, w.Body.String())
	}
	for _, path := range []string{"/console/api/v1/grants", "/console/api/v1/grants/one/revoke"} {
		if w := client.request("POST", path, `{}`, "http://127.0.0.1:8080", ""); w.Code != 403 {
			t.Fatal("missing CSRF accepted", path, w.Code)
		}
		if w := client.request("POST", path, `{}`, "http://127.0.0.1:8080", client.csrf); w.Code != 428 {
			t.Fatal("authenticated route not reached", path, w.Code, w.Body.String())
		}
	}
}
