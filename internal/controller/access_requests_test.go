// SPDX-License-Identifier: Apache-2.0
package controller

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"
)

const requesterToken = "requester-independent-key-12345678901234567890"

// Catches missing owner filtering, self-approval checks, narrowing, strict DTOs
// and approval/publication drift through real HTTP/Casbin/SQLite boundaries.
func TestAccessRequestIndependentApprovalAndOwnership(t *testing.T) {
	h, store, credentials := managementFixture(t)
	hash := func(s string) string { v := sha256.Sum256([]byte(s)); return hex.EncodeToString(v[:]) }
	raw := fmt.Sprintf(`{"principals":[{"id":"principal:operator","sha256":%q,"roles":["role:admin"]},{"id":"principal:requester","sha256":%q,"roles":["role:requester"]}]}`, hash(managementToken), hash(requesterToken))
	if err := os.WriteFile(credentials, []byte(raw), 0600); err != nil {
		t.Fatal(err)
	}
	policy, _ := os.ReadFile("../../examples/static/gitea.yaml")
	facts, _ := os.ReadFile("../../examples/static/facts.json")
	pub, _ := json.Marshal(map[string]string{"policy": string(policy), "facts": string(facts), "reason": "request inventory"})
	if w := request(h, "PUT", "/api/v1/publication", string(pub), managementToken, `"0"`); w.Code != 200 {
		t.Fatal(w.Code, w.Body.String())
	}
	w := request(h, "GET", "/api/v1/access-request-catalog", "", requesterToken, "")
	if w.Code != 200 || w.Header().Get("ETag") != `"1"` || !strings.Contains(w.Body.String(), "resource-gitea-v1") || strings.Contains(w.Body.String(), "facts") {
		t.Fatal(w.Code, w.Body.String())
	}
	for _, path := range []string{"/publication", "/inventory", "/grants", "/audit", "/agent-observations"} {
		if w := request(h, "GET", "/api/v1"+path, "", requesterToken, ""); w.Code != 403 {
			t.Fatal(path, w.Code, w.Body.String())
		}
	}
	me := request(h, "GET", "/api/v1/me", "", requesterToken, "")
	if !strings.Contains(me.Body.String(), "request:create") || strings.Contains(me.Body.String(), "publication:read") {
		t.Fatal(me.Body.String())
	}
	body := `{"source":"2001:db8::42","resource_name":"gitea","resource_uid":"resource-gitea-v1","endpoint_names":["https","ssh"],"duration_seconds":3600,"reason":"independent request"}`
	for _, bad := range []string{`null`, strings.TrimSuffix(body, "}") + `,"requester":"principal:operator"}`, strings.Replace(body, `["https","ssh"]`, `[]`, 1), strings.Replace(body, `["https","ssh"]`, `["https","https"]`, 1), strings.Replace(body, `3600`, `86401`, 1)} {
		if w := request(h, "POST", "/api/v1/access-requests", bad, requesterToken, `"1"`); w.Code != 400 {
			t.Fatal(bad, w.Code, w.Body.String())
		}
	}
	if w := request(h, "POST", "/api/v1/access-requests", body, requesterToken, ""); w.Code != 428 {
		t.Fatal(w.Code, w.Body.String())
	}
	w = request(h, "POST", "/api/v1/access-requests", body, requesterToken, `"1"`)
	if w.Code != 201 {
		t.Fatal(w.Code, w.Body.String())
	}
	var row struct {
		UID, Requester, Source, State string
		ExpiresAt                     time.Time `json:"expires_at"`
	}
	if json.Unmarshal(w.Body.Bytes(), &row) != nil || row.UID == "" || row.Requester != "principal:requester" || row.Source != "2001:db8::42/128" || row.State != "pending" {
		t.Fatal(w.Body.String())
	}
	if doc, _ := store.Snapshot(); doc.Revision != 1 {
		t.Fatal("submission granted access", doc.Revision)
	}
	decision := fmt.Sprintf(`{"version":1,"action":"approve","endpoint_names":["https"],"expires_at":%q,"reason":"approved narrowed https"}`, row.ExpiresAt.Add(-time.Minute).Format(time.RFC3339Nano))
	if w := request(h, "POST", "/api/v1/access-requests/"+row.UID+"/decision", decision, requesterToken, `"1"`); w.Code != 403 {
		t.Fatal(w.Code, w.Body.String())
	}
	tooLong := strings.Replace(decision, row.ExpiresAt.Add(-time.Minute).Format(time.RFC3339Nano), row.ExpiresAt.Add(time.Minute).Format(time.RFC3339Nano), 1)
	if w := request(h, "POST", "/api/v1/access-requests/"+row.UID+"/decision", tooLong, managementToken, `"1"`); w.Code != 400 {
		t.Fatal(w.Code, w.Body.String())
	}
	w = request(h, "POST", "/api/v1/access-requests/"+row.UID+"/decision", decision, managementToken, `"1"`)
	if w.Code != 200 || !strings.Contains(w.Body.String(), `"state":"approved"`) || w.Header().Get("ETag") != `"2"` {
		t.Fatal(w.Code, w.Body.String())
	}
	if w := request(h, "POST", "/api/v1/access-requests/"+row.UID+"/decision", decision, managementToken, `"2"`); w.Code != 409 {
		t.Fatal("duplicate decision", w.Code, w.Body.String())
	}
	grants := request(h, "GET", "/api/v1/grants", "", managementToken, "")
	if !strings.Contains(grants.Body.String(), "2001:db8::42/128") || !strings.Contains(grants.Body.String(), row.UID) {
		t.Fatal(grants.Body.String())
	}
	// An admin may request, but cannot approve their own request even with all permissions.
	w = request(h, "POST", "/api/v1/access-requests", strings.Replace(body, "2001:db8::42", "192.0.2.42", 1), managementToken, `"2"`)
	var own struct{ UID string }
	_ = json.Unmarshal(w.Body.Bytes(), &own)
	if w.Code != 201 || own.UID == "" {
		t.Fatal(w.Code, w.Body.String())
	}
	if w := request(h, "POST", "/api/v1/access-requests/"+own.UID+"/decision", decision, managementToken, `"2"`); w.Code != 403 {
		t.Fatal("self approval", w.Code, w.Body.String())
	}
	mine := request(h, "GET", "/api/v1/access-requests?limit=100", "", requesterToken, "")
	if mine.Code != 200 || strings.Contains(mine.Body.String(), own.UID) || !strings.Contains(mine.Body.String(), row.UID) {
		t.Fatal(mine.Code, mine.Body.String())
	}
	reject := `{"version":1,"action":"reject","reason":"not required"}`
	w = request(h, "POST", "/api/v1/access-requests/"+own.UID+"/decision", reject, managementToken, "")
	if w.Code != 200 || !strings.Contains(w.Body.String(), `"state":"rejected"`) {
		t.Fatal(w.Code, w.Body.String())
	}
	if doc, _ := store.Snapshot(); doc.Revision != 2 {
		t.Fatal("rejection changed publication", doc.Revision)
	}
	// An authorized advanced writer can edit Grant semantics; do not falsely
	// associate the new contribution with the immutable historical approval.
	for _, change := range []struct{ from, to, want string }{
		{`"cidr":"2001:db8::42/128"`, `"cidr":"2001:db8::99/128"`, "modified"},
		{`"endpointNames":["https"]`, `"endpointNames":["ssh"]`, "modified"},
		{row.ExpiresAt.Add(-time.Minute).Format(time.RFC3339Nano), row.ExpiresAt.Add(-2 * time.Minute).Format(time.RFC3339Nano), "modified"},
		{`"expiresAt":"` + row.ExpiresAt.Add(-time.Minute).Format(time.RFC3339Nano) + `"`, `"notBefore":"` + time.Now().Add(time.Minute).UTC().Format(time.RFC3339Nano) + `","expiresAt":"` + row.ExpiresAt.Add(-time.Minute).Format(time.RFC3339Nano) + `"`, "scheduled"},
	} {
		before, _ := store.Snapshot()
		changed := strings.Replace(before.Policy, change.from, change.to, 1)
		if changed == before.Policy {
			t.Fatal("mutation fixture missing", change.from)
		}
		_, err := store.AcceptManaged(before.Revision, "principal:operator", "authorized advanced edit", []byte(changed), []byte(before.Facts), nil)
		if err != nil {
			t.Fatal(err)
		}
		view := request(h, "GET", "/api/v1/access-requests", "", requesterToken, "")
		if !strings.Contains(view.Body.String(), `"grant_state":"`+change.want+`"`) || strings.Contains(view.Body.String(), "2001:db8::99") {
			t.Fatal("misleading desired approval", view.Body.String())
		}
		current, _ := store.Snapshot()
		if _, err = store.AcceptManaged(current.Revision, "principal:operator", "restore approved desired intent", []byte(before.Policy), []byte(before.Facts), nil); err != nil {
			t.Fatal(err)
		}
	}
	var grantList struct {
		Grants []struct{ UID, Source string }
	}
	_ = json.Unmarshal(grants.Body.Bytes(), &grantList)
	for _, g := range grantList.Grants {
		if g.Source != "2001:db8::42/128" {
			continue
		}
		current, _ := store.Snapshot()
		w = request(h, "POST", "/api/v1/grants/"+g.UID+"/revoke", `{"reason":"approved contribution revoked"}`, managementToken, fmt.Sprintf(`"%d"`, current.Revision))
		if w.Code != 200 {
			t.Fatal(w.Code, w.Body.String())
		}
	}
	mine = request(h, "GET", "/api/v1/access-requests", "", requesterToken, "")
	if !strings.Contains(mine.Body.String(), `"grant_state":"revoked"`) || !strings.Contains(mine.Body.String(), `"state":"approved"`) {
		t.Fatal("approval is not current permission", mine.Body.String())
	}
}
