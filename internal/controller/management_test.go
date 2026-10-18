// SPDX-License-Identifier: Apache-2.0
package controller

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/PicketX/PicketX/internal/management"
	"github.com/PicketX/PicketX/internal/managementaccess"
	"github.com/PicketX/PicketX/internal/publication"
	"time"
)

const managementToken = "management-api-independent-token-1234567890"

type directRepository struct{ *publication.Store }

func (r directRepository) Replace(expected uint64, actor, reason, policy, facts string) (publication.Document, error) {
	_, e := r.AcceptManaged(expected, actor, reason, []byte(policy), []byte(facts), nil)
	if e != nil {
		return publication.Document{}, e
	}
	return r.Snapshot()
}
func managementFixture(t *testing.T) (http.Handler, *publication.Store, string) {
	t.Helper()
	dir := t.TempDir()
	creds := filepath.Join(dir, "principals.json")
	sum := sha256.Sum256([]byte(managementToken))
	raw := fmt.Sprintf(`{"principals":[{"id":"principal:operator","sha256":%q,"roles":["role:admin"]}]}`, hex.EncodeToString(sum[:]))
	if e := os.WriteFile(creds, []byte(raw), 0600); e != nil {
		t.Fatal(e)
	}
	s, e := publication.OpenMode(filepath.Join(dir, "state", "db.sqlite"), "static:example-v1", "database")
	if e != nil {
		t.Fatal(e)
	}
	t.Cleanup(func() { _ = s.Close() })
	a, e := managementaccess.New(s.GORM(), managementaccess.Settings{Credentials: creds, Reload: time.Second, WarnRules: 10000, WarnLoadPercent: 50}, slog.New(slog.NewTextHandler(io.Discard, nil)))
	if e != nil {
		t.Fatal(e)
	}
	return NewHandlerWithManagement(nil, management.New(a, directRepository{s}), ManagementLimits{RequestsPerSecond: 1000, Burst: 1000}), s, creds
}
func request(h http.Handler, method, path, body, token, match string) *httptest.ResponseRecorder {
	r := httptest.NewRequest(method, path, strings.NewReader(body))
	r.Header.Set("Content-Type", "application/json")
	if token != "" {
		r.Header.Set("Authorization", "Bearer "+token)
	}
	if match != "" {
		r.Header.Set("If-Match", match)
	}
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	return w
}
func TestManagementHTTPPreconditionsStrictDTOAndAudit(t *testing.T) {
	h, _, _ := managementFixture(t)
	if w := request(h, "GET", "/api/v1/publication", "", "", ""); w.Code != 401 {
		t.Fatal(w.Code, w.Body.String())
	}
	p, _ := os.ReadFile("../../examples/static/gitea.yaml")
	f, _ := os.ReadFile("../../examples/static/facts.json")
	raw, _ := json.Marshal(map[string]string{"policy": string(p), "facts": string(f), "reason": "initial desired state"})
	if w := request(h, "PUT", "/api/v1/publication", string(raw), managementToken, ""); w.Code != 428 {
		t.Fatal(w.Code, w.Body.String())
	}
	for _, bad := range []string{`W/"0"`, `"00"`, `"0", "1"`, `*`, `0`} {
		if w := request(h, "PUT", "/api/v1/publication", string(raw), managementToken, bad); w.Code != 400 {
			t.Fatal(bad, w.Code, w.Body.String())
		}
	}
	for _, bad := range []string{`null`, `{"policy":null}`, `{"Policy":"x"}`, `{"policy":"x","policy":"y"}`, `{"reason":"","policy":"x","facts":"{}"}`, `{"policy":"x","facts":"{}","reason":"x","roles":["role:admin"]}`} {
		if w := request(h, "PUT", "/api/v1/publication", bad, managementToken, `"0"`); w.Code != 400 {
			t.Fatal(bad, w.Code, w.Body.String())
		}
	}
	w := request(h, "PUT", "/api/v1/publication", string(raw), managementToken, `"0"`)
	if w.Code != 200 || w.Header().Get("ETag") != `"1"` {
		t.Fatal(w.Code, w.Body.String())
	}
	if strings.Contains(w.Body.String(), "applied") {
		t.Fatal("write claims execution", w.Body.String())
	}
	if w = request(h, "PUT", "/api/v1/publication", string(raw), managementToken, `"0"`); w.Code != 412 {
		t.Fatal(w.Code, w.Body.String())
	}
	w = request(h, "GET", "/api/v1/publication", "", managementToken, "")
	if w.Code != 200 || w.Header().Get("Cache-Control") != "no-store" {
		t.Fatal(w.Code, w.Body.String())
	}
	var doc publication.Document
	if e := json.Unmarshal(w.Body.Bytes(), &doc); e != nil || doc.Policy != string(p) || doc.Facts != string(f) {
		t.Fatal(doc, e)
	}
	w = request(h, "GET", "/api/v1/audit?limit=1", "", managementToken, "")
	if w.Code != 200 || !strings.Contains(w.Body.String(), "principal:operator") {
		t.Fatal(w.Code, w.Body.String())
	}
	for _, q := range []string{"limit=101", "after=-1", "limit=1&limit=2", "unexpected=1", "after=bad"} {
		if w = request(h, "GET", "/api/v1/audit?"+q, "", managementToken, ""); w.Code != 400 {
			t.Fatal(q, w.Code, w.Body.String())
		}
	}
}
func TestManagementHTTPUsesTrustedIdentityAndRevokesNextRequest(t *testing.T) {
	h, _, creds := managementFixture(t)
	w := request(h, "GET", "/api/v1/me", "", managementToken, "")
	if w.Code != 200 || !strings.Contains(w.Body.String(), "principal:operator") {
		t.Fatal(w.Code, w.Body.String())
	}
	data, _ := os.ReadFile(creds)
	data = []byte(strings.ReplaceAll(string(data), "role:admin", "role:viewer"))
	if e := os.WriteFile(creds, data, 0600); e != nil {
		t.Fatal(e)
	}
	if w = request(h, "PUT", "/api/v1/publication", `{}`, managementToken, `"0"`); w.Code != 403 {
		t.Fatal(w.Code, w.Body.String())
	}
	if e := os.WriteFile(creds, []byte(`{"principals":[]}`), 0600); e != nil {
		t.Fatal(e)
	}
	if w = request(h, "GET", "/api/v1/publication", "", managementToken, ""); w.Code != 401 {
		t.Fatal(w.Code, w.Body.String())
	}
	if w = request(NewHandler(), "GET", "/api/v1/publication", "", managementToken, ""); w.Code != 404 {
		t.Fatal("file mode exposed management", w.Code)
	}
}
