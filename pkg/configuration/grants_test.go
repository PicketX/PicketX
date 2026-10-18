// SPDX-License-Identifier: Apache-2.0
package configuration

import (
	"os"
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestStructuredGrantPreservesPublicationAndDeadline(t *testing.T) {
	raw, err := os.ReadFile("../../examples/static/gitea.yaml")
	if err != nil {
		t.Fatal(err)
	}
	b, err := Decode(raw)
	if err != nil {
		t.Fatal(err)
	}
	before, grants := b.GrantCatalog()
	if len(before) != 1 || len(grants) != 2 || grants[0].UID != "grant-browser-v1" {
		t.Fatal(before, grants)
	}
	expiry := time.Date(2026, 10, 18, 15, 0, 0, 0, time.UTC)
	input := GrantInput{Name: "temporary6", UID: "grant-new-v1", Source: "2001:db8::42", ResourceName: "gitea", ResourceUID: "resource-gitea-v1", EndpointNames: []string{"https"}, ExpiresAt: expiry}
	out, err := b.AddGrant(input)
	if err != nil {
		t.Fatal(err)
	}
	next, err := Decode(out)
	if err != nil {
		t.Fatal(err)
	}
	if len(next.objects) != len(b.objects)+1 {
		t.Fatal("lost objects")
	}
	for _, old := range b.objects {
		found := false
		for _, n := range next.objects {
			if old.Metadata.UID == n.Metadata.UID {
				found = true
				if !reflect.DeepEqual(old, n) {
					t.Fatalf("changed unrelated %s", old.Metadata.UID)
				}
			}
		}
		if !found {
			t.Fatal("lost", old.Metadata.UID)
		}
	}
	resources, gs := next.GrantCatalog()
	if !reflect.DeepEqual(before, resources) || len(gs) != 3 {
		t.Fatal(resources, gs)
	}
	var got GrantView
	for _, g := range gs {
		if g.UID == input.UID {
			got = g
		}
	}
	if got.Source != "2001:db8::42/128" || got.ExpiresAt != expiry.Format(time.RFC3339Nano) || got.Revoked {
		t.Fatal(got)
	}
	revoked, err := next.RevokeGrant(input.UID)
	if err != nil {
		t.Fatal(err)
	}
	rb, err := Decode(revoked)
	if err != nil {
		t.Fatal(err)
	}
	_, rgs := rb.GrantCatalog()
	for _, g := range rgs {
		if g.UID == input.UID && (!g.Revoked || g.ExpiresAt != got.ExpiresAt) {
			t.Fatal(g)
		}
	}
	again, err := rb.RevokeGrant(input.UID)
	if err != nil {
		t.Fatal(err)
	}
	ab, err := Decode(again)
	if err != nil || ab.hash != rb.hash {
		t.Fatal("revoke not idempotent", err)
	}
	if _, err = next.AddGrant(input); err == nil {
		t.Fatal("duplicate identity accepted")
	}
	for _, mutate := range []func(*GrantInput){func(i *GrantInput) { i.ResourceUID = "recreated" }, func(i *GrantInput) { i.EndpointNames = nil }, func(i *GrantInput) { i.EndpointNames = []string{"missing"} }, func(i *GrantInput) { i.EndpointNames = []string{"https", "https"} }, func(i *GrantInput) { i.Source = "not-an-ip" }, func(i *GrantInput) { i.ExpiresAt = time.Time{} }} {
		bad := input
		bad.EndpointNames = append([]string{}, input.EndpointNames...)
		mutate(&bad)
		if _, err = b.AddGrant(bad); err == nil {
			t.Fatal("invalid input accepted", bad)
		}
	}
}

func TestStructuredGrantRetainsDynamicScopeAndNeverRevokesDeny(t *testing.T) {
	raw, err := os.ReadFile("../../examples/static/gitea.yaml")
	if err != nil {
		t.Fatal(err)
	}
	text := strings.Replace(string(raw), "      endpointNames: [https]\n", "", 1)
	text += "\n---\napiVersion: picketx.io/v1alpha1\nkind: NetworkDeny\nmetadata: {name: blocked, uid: deny-v1}\nspec:\n  subject: {type: source, cidr: 192.0.2.1}\n  scopes: [{resourceRef: {name: gitea, uid: resource-gitea-v1}}]\n  validity: {mode: permanent}\n"
	b, err := Decode([]byte(text))
	if err != nil {
		t.Fatal(err)
	}
	_, gs := b.GrantCatalog()
	if gs[0].Scopes[0].Selection != "all" {
		t.Fatal(gs)
	}
	if _, err = b.RevokeGrant("deny-v1"); err != ErrGrantNotFound {
		t.Fatal("deny was treated as grant", err)
	}
	out, err := b.RevokeGrant("grant-browser-v1")
	if err != nil {
		t.Fatal(err)
	}
	next, err := Decode(out)
	if err != nil {
		t.Fatal(err)
	}
	for _, o := range next.objects {
		if o.Kind == "NetworkDeny" && o.Spec.(contributionSpec).Revoked {
			t.Fatal("deny revoked")
		}
	}
}
