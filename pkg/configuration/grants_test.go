// SPDX-License-Identifier: Apache-2.0
package configuration

import (
	"bytes"
	"encoding/json"
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

func TestStructuredGrantCatalogDetachedSelectorAndHTTPRoute(t *testing.T) {
	raw, err := os.ReadFile("../../examples/http/policy.yaml")
	if err != nil {
		t.Fatal(err)
	}
	text := strings.ReplaceAll(string(raw), "__HOST__", "app.example")
	text = strings.Replace(text, "endpoints: [{name: web, network:", "endpoints: [{name: web, labels: {public: 'yes'}, network:", 1)
	text = strings.Replace(text, "endpointNames: [web]", "endpointSelector: {matchLabels: {public: 'yes'}}", 1)
	b, err := Decode([]byte(text))
	if err != nil {
		t.Fatal(err)
	}
	resources, grants := b.GrantCatalog()
	if len(grants) != 2 || grants[0].Scopes[0].Selection != "selector" || !strings.Contains(grants[0].Scopes[0].Selector, `"public":"yes"`) {
		t.Fatal(grants)
	}
	resources[0].Endpoints[0].Name = "mutated"
	grants[0].Scopes[0].Selector = "mutated"
	resources, grants = b.GrantCatalog()
	if resources[0].Endpoints[0].Name != "web" || grants[0].Scopes[0].Selector == "mutated" {
		t.Fatal("catalog aliases private policy")
	}
	facts := Facts{AuthorityID: "test"}
	before, err := Compile(b, facts)
	if err != nil {
		t.Fatal(err)
	}
	changed, err := b.AddGrant(GrantInput{Name: "other", UID: "grant-other-v1", Source: "2001:db8::42/64", ResourceName: "private-app", ResourceUID: "private-app-v1", EndpointNames: []string{"web"}, ExpiresAt: time.Now().Add(time.Hour)})
	if err != nil {
		t.Fatal(err)
	}
	next, err := Decode(changed)
	if err != nil {
		t.Fatal(err)
	}
	candidate, _, err := Replace(before, next, facts)
	if err != nil {
		t.Fatal(err)
	}
	if !candidate.HTTP().HasRoute("route-v1") || !candidate.HTTP().HasRoute("route-private-v1") {
		t.Fatal("unrelated HTTP route lost")
	}
	_, out := next.GrantCatalog()
	for _, g := range out {
		if g.UID == "grant-other-v1" && g.Source != "2001:db8::/64" {
			t.Fatal("explicit CIDR not masked", g)
		}
	}
	for _, old := range b.objects {
		found := false
		for _, o := range next.objects {
			if old.Metadata.UID == o.Metadata.UID {
				found = true
				if !reflect.DeepEqual(old, o) {
					t.Fatal("changed selector/HTTP object", old.Metadata.UID)
				}
			}
		}
		if !found {
			t.Fatal("lost object", old.Metadata.UID)
		}
	}
}

func TestStructuredGrantAcceptedRepresentationDoesNotExpand(t *testing.T) {
	base, err := os.ReadFile("../../examples/static/gitea.yaml")
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		name, annotation string
		nearLimit        bool
	}{
		{name: "HTML escaping", annotation: strings.Repeat("<", 800000)},
		{name: "JSON quote escaping", annotation: strings.Repeat(`"`, 2200000)},
		{name: "dense input", nearLimit: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			// Put the large annotation on the Grant itself, not only an unrelated Resource.
			template := strings.Replace(string(base), "metadata: {name: browser, uid: grant-browser-v1}", "metadata: {name: browser, uid: grant-browser-v1, annotations: {note: '__NOTE__'}}", 1)
			text := strings.Replace(template, "__NOTE__", tc.annotation, 1)
			if tc.nearLimit {
				text = strings.Replace(template, "__NOTE__", strings.Repeat("x", MaxBytes-len(template)+len("__NOTE__")-64), 1)
			}
			b, err := Decode([]byte(text))
			if err != nil {
				t.Fatal(err)
			}
			factsData, err := os.ReadFile("../../examples/static/facts.json")
			if err != nil {
				t.Fatal(err)
			}
			var facts Facts
			if err = json.Unmarshal(factsData, &facts); err != nil {
				t.Fatal(err)
			}
			previous, err := Compile(b, facts)
			if err != nil {
				t.Fatal("original input not valid", err)
			}
			out, err := b.RevokeGrant("grant-browser-v1")
			if err != nil {
				t.Fatal("valid accepted Grant cannot be revoked", err)
			}
			if len(out) > MaxBytes {
				t.Fatal("shared bound bypassed")
			}
			next, err := Decode(out)
			if err != nil {
				t.Fatal(err)
			}
			if _, _, err = Replace(previous, next, facts); err != nil {
				t.Fatal(err)
			}
			if len(out) > len(text)+32 {
				t.Fatal("rewrote unrelated representation", len(text), len(out))
			}
			if !tc.nearLimit {
				added, err := b.AddGrant(GrantInput{Name: "another", UID: "new-source-v1", Source: "::1", ResourceName: "gitea", ResourceUID: "resource-gitea-v1", EndpointNames: []string{"https"}, ExpiresAt: time.Now().Add(time.Hour)})
				if err != nil {
					t.Fatal("unrelated representation blocked issue", err)
				}
				if !bytes.HasPrefix(added, []byte(text)) {
					t.Fatal("issue rewrites accepted input")
				}
			}
		})
	}
}

func TestStructuredGrantSourceLocalRevokeYAMLLayouts(t *testing.T) {
	base, err := os.ReadFile("../../examples/static/gitea.yaml")
	if err != nil {
		t.Fatal(err)
	}
	for _, value := range []string{"false", `!!bool 'false'`, `!!bool "f\x61lse"`, "!!bool |-\n    false"} {
		t.Run(value, func(t *testing.T) {
			text := strings.Replace(string(base), "  subject: {type: source, cidr: 198.51.100.9}", "  revoked: "+value+" # retained\n  subject: {type: source, cidr: 198.51.100.9}", 1)
			// A block scalar's comment belongs on its header, not its content.
			text = strings.Replace(text, "!!bool |-\n    false # retained", "!!bool |- # retained\n    false", 1)
			b, err := Decode([]byte(text))
			if err != nil {
				t.Fatal(err)
			}
			out, err := b.RevokeGrant("grant-browser-v1")
			if err != nil {
				t.Fatal(err)
			}
			n, err := Decode(out)
			if err != nil {
				t.Fatal(err)
			}
			_, gs := n.GrantCatalog()
			if !gs[0].Revoked || gs[1].Revoked {
				t.Fatal(gs)
			}
		})
	}
	for _, prefix := range []string{"", "\ufeff", "# 中文 policy\n"} {
		// JSON flow mapping with an explicit boolean, Unicode before its column,
		// CRLF, and directives exercise real parser positions, not a YAML regex.
		text := prefix + strings.Replace(string(base), "metadata: {name: browser, uid: grant-browser-v1}", "metadata: {name: 浏览器, uid: grant-browser-v1}", 1)
		b, err := Decode([]byte(strings.ReplaceAll(text, "\n", "\r\n")))
		if err != nil {
			t.Fatal(err)
		}
		if _, err = b.RevokeGrant("grant-browser-v1"); err != nil {
			t.Fatal(err)
		}
	}
	resource := strings.Split(string(base), "\n---\n")[0]
	for _, flag := range []string{"", ", revoked: false", `, revoked: !!bool "false"`} {
		for _, separator := range []string{"\n", "\r\n", "\r", "\u0085", "\u2028", "\u2029"} {
			raw := resource + `
---
{apiVersion: picketx.io/v1alpha1, kind: Grant, metadata: {name: 浏览器, uid: grant-browser-v1}, spec: !!map { ? # first explicit key
subject : {type: source, cidr: '::1'}, scopes: [{resourceRef: {name: gitea, uid: resource-gitea-v1}}], validity: {mode: permanent}` + flag + `}}
...
`
			input := []byte(strings.ReplaceAll(raw, "\n", separator))
			b, err := Decode(input)
			if err != nil {
				t.Fatal("layout not accepted", separator, flag, err)
			}
			out, err := b.RevokeGrant("grant-browser-v1")
			if err != nil {
				t.Fatal("flow/Unicode line position", separator, flag, err)
			}
			next, err := Decode(out)
			if err != nil {
				t.Fatal(err)
			}
			_, grants := next.GrantCatalog()
			if len(grants) != 1 || !grants[0].Revoked {
				t.Fatal(grants)
			}
		}
	}
}
