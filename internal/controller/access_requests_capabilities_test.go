// SPDX-License-Identifier: Apache-2.0
package controller

import (
	"github.com/PicketX/PicketX/internal/accessrequest"
	"github.com/PicketX/PicketX/internal/management"
	"github.com/PicketX/PicketX/internal/managementaccess"
	"io"
	"log/slog"
	"os"
	"strings"
	"testing"
	"time"
)

func TestAccessRequestReviewCatalogDoesNotRequireCreation(t *testing.T) {
	_, store, path := managementFixture(t)
	raw, _ := os.ReadFile(path)
	if err := os.WriteFile(path, []byte(strings.Replace(string(raw), `["role:admin"]`, `["role:admin","role:no-create"]`, 1)), 0600); err != nil {
		t.Fatal(err)
	}
	if err := store.GORM().Exec("INSERT INTO casbin_rule(ptype,v0,v1,v2,v3,v4,v5) VALUES('p','role:no-create','request:create','deny','','','')").Error; err != nil {
		t.Fatal(err)
	}
	access, err := managementaccess.New(store.GORM(), managementaccess.Settings{Credentials: path, Reload: time.Second, WarnRules: 10000, WarnLoadPercent: 50}, slog.New(slog.NewTextHandler(io.Discard, nil)))
	if err != nil {
		t.Fatal(err)
	}
	h := NewHandlerWithManagement(nil, management.New(access, directRepository{store}), ManagementLimits{RequestsPerSecond: 1000, Burst: 1000})
	policy, _ := os.ReadFile("../../examples/static/gitea.yaml")
	facts, _ := os.ReadFile("../../examples/static/facts.json")
	if _, err := store.AcceptManaged(0, "seed", "seed", policy, facts, nil); err != nil {
		t.Fatal(err)
	}
	if w := request(h, "POST", "/api/v1/access-requests", `{}`, managementToken, `"1"`); w.Code != 403 {
		t.Fatal("review role can create", w.Code, w.Body.String())
	}
	if w := request(h, "GET", "/api/v1/access-requests", "", managementToken, ""); w.Code != 200 {
		t.Fatal(w.Code, w.Body.String())
	}
	if w := request(h, "GET", "/api/v1/access-request-catalog", "", managementToken, ""); w.Code != 200 || w.Header().Get("ETag") != `"1"` || !strings.Contains(w.Body.String(), "resource-gitea-v1") {
		t.Fatal("authorized reviewer catalog", w.Code, w.Body.String())
	}
}

func TestAccessRequestIndependentCapabilities(t *testing.T) {
	for _, tc := range []struct {
		name                          string
		operations                    []string
		catalog, list, create, reject int
	}{
		{"create-only", []string{"request:create"}, 200, 403, 201, 403},
		{"review-only", []string{"request:review"}, 403, 200, 403, 200},
		{"approve-only", []string{"request:review", "publication:write"}, 200, 200, 403, 200},
		{"write-only", []string{"publication:write"}, 403, 403, 403, 403},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, store, path := managementFixture(t)
			raw, _ := os.ReadFile(path)
			if err := os.WriteFile(path, []byte(strings.Replace(string(raw), `["role:admin"]`, `["role:custom"]`, 1)), 0600); err != nil {
				t.Fatal(err)
			}
			for _, op := range tc.operations {
				if err := store.GORM().Exec("INSERT INTO casbin_rule(ptype,v0,v1,v2,v3,v4,v5) VALUES('p','role:custom',?,'allow','','','')", op).Error; err != nil {
					t.Fatal(err)
				}
			}
			a, err := managementaccess.New(store.GORM(), managementaccess.Settings{Credentials: path, Reload: time.Second, WarnRules: 10000, WarnLoadPercent: 50}, slog.New(slog.NewTextHandler(io.Discard, nil)))
			if err != nil {
				t.Fatal(err)
			}
			h := NewHandlerWithManagement(nil, management.New(a, directRepository{store}), ManagementLimits{RequestsPerSecond: 1000, Burst: 1000})
			policy, _ := os.ReadFile("../../examples/static/gitea.yaml")
			facts, _ := os.ReadFile("../../examples/static/facts.json")
			if _, err = store.AcceptManaged(0, "seed", "seed", policy, facts, nil); err != nil {
				t.Fatal(err)
			}
			if _, err = store.CreateRequest(1, accessrequest.Record{UID: "request-custom", Requester: "principal:other", State: "pending", Version: 1}); err != nil {
				t.Fatal(err)
			}
			for path, want := range map[string]int{"/access-requests": tc.list, "/access-request-catalog": tc.catalog} {
				if w := request(h, "GET", "/api/v1"+path, "", managementToken, ""); w.Code != want {
					t.Fatal(path, w.Code, w.Body.String())
				}
			}
			body := `{"source":"192.0.2.42","resource_name":"gitea","resource_uid":"resource-gitea-v1","endpoint_names":["https"],"duration_seconds":3600,"reason":"custom capability"}`
			if w := request(h, "POST", "/api/v1/access-requests", body, managementToken, `"1"`); w.Code != tc.create {
				t.Fatal(w.Code, w.Body.String())
			}
			if w := request(h, "POST", "/api/v1/access-requests/request-custom/decision", `{"version":1,"action":"reject","reason":"custom rejection"}`, managementToken, ""); w.Code != tc.reject {
				t.Fatal(w.Code, w.Body.String())
			}
		})
	}
}
