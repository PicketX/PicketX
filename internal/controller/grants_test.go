// SPDX-License-Identifier: Apache-2.0
package controller

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/PicketX/PicketX/internal/publication"
)

func TestStructuredGrantHTTPAtomicCASAndAudit(t *testing.T) {
	h, store, _ := managementFixture(t)
	p, err := os.ReadFile("../../examples/static/gitea.yaml")
	if err != nil {
		t.Fatal(err)
	}
	f, err := os.ReadFile("../../examples/static/facts.json")
	if err != nil {
		t.Fatal(err)
	}
	raw, _ := json.Marshal(map[string]string{"policy": string(p), "facts": string(f), "reason": "initial"})
	if w := request(h, "PUT", "/api/v1/publication", string(raw), managementToken, `"0"`); w.Code != 200 {
		t.Fatal(w.Code, w.Body.String())
	}
	issue := `{"name":"temporary-ipv6","source":"2001:db8::42","resource_name":"gitea","resource_uid":"resource-gitea-v1","endpoint_names":["https"],"duration_seconds":3600,"reason":"temporary admin access"}`
	if w := request(h, "POST", "/api/v1/grants", issue, managementToken, ""); w.Code != 428 {
		t.Fatal(w.Code, w.Body.String())
	}
	for _, bad := range []string{strings.Replace(issue, `3600`, `0`, 1), strings.Replace(issue, `3600`, `86401`, 1), strings.Replace(issue, `["https"]`, `[]`, 1), strings.Replace(issue, `"temporary admin access"`, `"   "`, 1), strings.Replace(issue, `"source":`, `"unknown":`, 1), strings.Replace(issue, `"name":`, `"name":null,"name":`, 1)} {
		if w := request(h, "POST", "/api/v1/grants", bad, managementToken, `"1"`); w.Code != 400 {
			t.Fatal(bad, w.Code, w.Body.String())
		}
	}
	start := time.Now()
	w := request(h, "POST", "/api/v1/grants", issue, managementToken, `"1"`)
	if w.Code != 201 || w.Header().Get("ETag") != `"2"` {
		t.Fatal(w.Code, w.Body.String())
	}
	var created struct {
		UID      string `json:"uid"`
		Expires  string `json:"expires_at"`
		Revision uint64 `json:"revision"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &created); err != nil {
		t.Fatal(err)
	}
	expiry, err := time.Parse(time.RFC3339Nano, created.Expires)
	if err != nil || expiry.Before(start.Add(time.Hour)) || expiry.After(time.Now().Add(time.Hour)) || !strings.HasPrefix(created.UID, "grant-") || created.Revision != 2 {
		t.Fatal(created, expiry, err)
	}
	doc, err := store.Snapshot()
	if err != nil || doc.Facts != string(f) {
		t.Fatal(doc, err)
	}
	if w = request(h, "POST", "/api/v1/grants", issue, managementToken, `"1"`); w.Code != 412 {
		t.Fatal("stale issue", w.Code, w.Body.String())
	}
	if w = request(h, "POST", "/api/v1/grants", issue, managementToken, `"2"`); w.Code != 400 {
		t.Fatal("duplicate name", w.Code, w.Body.String())
	}
	w = request(h, "GET", "/api/v1/grants?limit=1", "", managementToken, "")
	if w.Code != 200 || w.Header().Get("ETag") != `"2"` || w.Header().Get("Cache-Control") != "no-store" {
		t.Fatal(w.Code, w.Body.String())
	}
	var page struct {
		Grants []struct {
			UID string `json:"uid"`
		}
		Next string `json:"next_after"`
		More bool   `json:"has_more"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &page); err != nil || len(page.Grants) != 1 || !page.More || page.Next != page.Grants[0].UID {
		t.Fatal(page, err)
	}
	next := "/api/v1/grants?limit=1&after=" + page.Next
	if w = request(h, "GET", next, "", managementToken, ""); w.Code != 428 {
		t.Fatal("unbound continuation", w.Code)
	}
	if w = request(h, "GET", next, "", managementToken, `"1"`); w.Code != 412 {
		t.Fatal("mixed revisions", w.Code)
	}
	if w = request(h, "GET", next, "", managementToken, `"2"`); w.Code != 200 {
		t.Fatal(w.Code, w.Body.String())
	}
	for _, q := range []string{"limit=101", "limit=0", "limit=1&limit=2", "after=a/b", "unexpected=1"} {
		if w = request(h, "GET", "/api/v1/grants?"+q, "", managementToken, ""); w.Code != 400 {
			t.Fatal(q, w.Code)
		}
	}
	revoke := "/api/v1/grants/" + created.UID + "/revoke"
	if w = request(h, "POST", revoke, `{"reason":"remove temporary access"}`, managementToken, `"2"`); w.Code != 200 || w.Header().Get("ETag") != `"3"` {
		t.Fatal(w.Code, w.Body.String())
	}
	if w = request(h, "POST", revoke, `{"reason":"repeat"}`, managementToken, `"2"`); w.Code != 412 {
		t.Fatal("stale idempotent revoke", w.Code)
	}
	if w = request(h, "POST", revoke, `{"reason":"repeat"}`, managementToken, `"3"`); w.Code != 200 || w.Header().Get("ETag") != `"3"` {
		t.Fatal("idempotent revoke", w.Code, w.Body.String())
	}
	if w = request(h, "POST", "/api/v1/grants/missing/revoke", `{"reason":"remove"}`, managementToken, `"3"`); w.Code != 404 {
		t.Fatal(w.Code, w.Body.String())
	}
	events, err := store.Audit(0, 100)
	if err != nil || len(events) != 3 || events[1].Actor != "principal:operator" || events[1].Reason != "temporary admin access" || events[2].Reason != "remove temporary access" {
		t.Fatal(events, err)
	}
	var final publication.Document
	w = request(h, "GET", "/api/v1/publication", "", managementToken, "")
	_ = json.Unmarshal(w.Body.Bytes(), &final)
	if final.Facts != string(f) || !strings.Contains(final.Policy, created.Expires) {
		t.Fatal("facts/deadline changed")
	}
}

func TestStructuredGrantHTTPCurrentRBAC(t *testing.T) {
	h, _, creds := managementFixture(t)
	data, err := os.ReadFile(creds)
	if err != nil {
		t.Fatal(err)
	}
	if err = os.WriteFile(creds, []byte(strings.ReplaceAll(string(data), "role:admin", "role:viewer")), 0600); err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{"/api/v1/grants", "/api/v1/grants/any/revoke"} {
		if w := request(h, "POST", path, `{}`, managementToken, `"0"`); w.Code != 403 {
			t.Fatal(path, w.Code, w.Body.String())
		}
	}
	if w := request(h, "GET", "/api/v1/grants", "", "", ""); w.Code != 401 {
		t.Fatal(fmt.Sprint(w.Code), w.Body.String())
	}
}
