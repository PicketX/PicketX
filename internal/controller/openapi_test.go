// SPDX-License-Identifier: Apache-2.0
package controller

import (
	"encoding/json"
	"go.yaml.in/yaml/v3"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGeneratedManagementContract(t *testing.T) {
	raw, e := os.ReadFile("../../docs/api/swagger.json")
	if e != nil {
		t.Fatal(e)
	}
	var spec struct {
		OpenAPI string `json:"openapi"`
		Paths   map[string]map[string]struct {
			Responses   map[string]any   `json:"responses"`
			Security    []map[string]any `json:"security"`
			OperationID string           `json:"operationId"`
		} `json:"paths"`
		Components struct {
			Schemas map[string]struct {
				AdditionalProperties *bool    `json:"additionalProperties"`
				Required             []string `json:"required"`
				Properties           map[string]struct {
					Type      string `json:"type"`
					MaxLength int    `json:"maxLength"`
					MinLength int    `json:"minLength"`
				} `json:"properties"`
			} `json:"schemas"`
		} `json:"components"`
	}
	if e = json.Unmarshal(raw, &spec); e != nil {
		t.Fatal(e)
	}
	if spec.OpenAPI != "3.1.0" {
		t.Fatal(spec.OpenAPI)
	}
	for path, methods := range map[string]map[string][]string{
		"/api/v1/publication": {"get": {"200", "401", "403", "429", "503"}, "put": {"200", "400", "401", "403", "412", "413", "415", "428", "429", "503"}},
		"/api/v1/audit":       {"get": {"200", "400", "401", "403", "429", "503"}},
		"/api/v1/me":          {"get": {"200", "401", "429", "503"}},
	} {
		for method, codes := range methods {
			op, ok := spec.Paths[path][method]
			if !ok || op.OperationID == "" || len(op.Security) != 1 {
				t.Fatal("missing route/auth contract", path, method)
			}
			if _, ok = op.Security[0]["ManagementBearer"]; !ok {
				t.Fatal("missing bearer", path)
			}
			if len(op.Responses) != len(codes) {
				t.Fatal("response mismatch", path, method)
			}
			for _, code := range codes {
				if _, ok := op.Responses[code]; !ok {
					t.Fatal("missing status", path, method, code)
				}
			}
		}
	}
	request := spec.Components.Schemas["controller.PublicationRequest"]
	if request.AdditionalProperties == nil || *request.AdditionalProperties || len(request.Required) != 3 {
		t.Fatal("DTO strictness not documented")
	}
	for key, max := range map[string]int{"policy": 4194304, "facts": 4194304, "reason": 512} {
		field := request.Properties[key]
		if field.Type != "string" || field.MaxLength != max || field.MinLength != 1 {
			t.Fatal(key, field)
		}
	}
}

func TestDocumentationCINodeVersionFilesExist(t *testing.T) {
	raw, e := os.ReadFile("../../.github/workflows/ci.yml")
	if e != nil {
		t.Fatal(e)
	}
	var workflow struct {
		Jobs map[string]struct {
			Steps []struct {
				Uses string            `yaml:"uses"`
				With map[string]string `yaml:"with"`
			} `yaml:"steps"`
		} `yaml:"jobs"`
	}
	if e = yaml.Unmarshal(raw, &workflow); e != nil {
		t.Fatal(e)
	}
	for job, definition := range workflow.Jobs {
		for _, step := range definition.Steps {
			if strings.HasPrefix(step.Uses, "actions/setup-node@") {
				path := step.With["node-version-file"]
				if path == "" || filepath.IsAbs(path) {
					t.Fatal("missing repository Node version file", job, path)
				}
				info, e := os.Stat(filepath.Join("../..", path))
				if e != nil || !info.Mode().IsRegular() {
					t.Fatal("Node version path does not exist", job, path, e)
				}
			}
		}
	}
}
