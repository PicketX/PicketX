// SPDX-License-Identifier: Apache-2.0
package configuration

import (
	"bytes"
	"errors"
	"os"
	"strings"
	"testing"
)

func TestRemoveBindingPreservesSurvivingDocuments(t *testing.T) {
	raw, err := os.ReadFile("../../examples/static/gitea.yaml")
	if err != nil {
		t.Fatal(err)
	}
	parts := strings.Split(string(raw), "---\n")
	binding := parts[3]
	rest := strings.Join(parts[:3], "---\n")
	for _, linebreak := range []string{"\n", "\r\n", "\r", "\u0085", "\u2028", "\u2029"} {
		for _, position := range []int{0, 1, 3} {
			t.Run(fmtPosition(position, linebreak), func(t *testing.T) {
				documents := append([]string{}, parts[:3]...)
				documents = append(documents[:position], append([]string{binding}, documents[position:]...)...)
				source := strings.ReplaceAll(strings.Join(documents, "---\n"), "\n", linebreak)
				b, e := Decode([]byte(source))
				if e != nil {
					t.Fatal(e)
				}
				out, e := b.RemoveBinding("binding-gitea-v1")
				if e != nil {
					t.Fatal(e)
				}
				next, e := Decode(out)
				if e != nil {
					t.Fatal(e)
				}
				r, g := next.GrantCatalog()
				if len(r) != 1 || len(g) != 2 || len(next.Inventory(Facts{}).Bindings) != 0 {
					t.Fatal("unrelated objects changed")
				}
				for _, doc := range parts[:3] {
					if !bytes.Contains(out, []byte(strings.ReplaceAll(doc, "\n", linebreak))) {
						t.Fatal("surviving bytes changed")
					}
				}
				if _, e = b.RemoveBinding("grant-browser-v1"); !errors.Is(e, ErrBindingNotFound) {
					t.Fatal("wrong kind", e)
				}
			})
		}
	}
	// Large annotations must not expand through a full JSON reserialization.
	source := strings.Replace(rest, "metadata: {name: gitea, uid: resource-gitea-v1}", "metadata: {name: gitea, uid: resource-gitea-v1, annotations: {note: '"+strings.Repeat("<", 800000)+"'}}", 1) + "---\n" + binding
	b, e := Decode([]byte(source))
	if e != nil {
		t.Fatal(e)
	}
	out, e := b.RemoveBinding("binding-gitea-v1")
	if e != nil || len(out) > len(source) {
		t.Fatal("expanded", e)
	}
	if !bytes.Contains(out, []byte(strings.Repeat("<", 800000))) {
		t.Fatal("annotation changed")
	}
}

func fmtPosition(n int, br string) string {
	return strings.Repeat("x", n+1) + strings.ReplaceAll(br, "\n", "LF")
}
