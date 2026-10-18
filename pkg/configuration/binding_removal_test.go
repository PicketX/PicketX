// SPDX-License-Identifier: Apache-2.0
package configuration

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
	"testing"
	"unicode/utf16"
)

func TestRemoveBindingRejectsUTF16WithoutPanic(t *testing.T) {
	raw, err := os.ReadFile("../../examples/static/gitea.yaml")
	if err != nil {
		t.Fatal(err)
	}
	units := utf16.Encode([]rune(strings.ReplaceAll(string(raw), "\n", "\u2028")))
	data := []byte{0xff, 0xfe}
	for _, unit := range units {
		data = binary.LittleEndian.AppendUint16(data, unit)
	}
	b, err := Decode(data)
	if err != nil {
		t.Fatal("fixture must be accepted", err)
	}
	if _, err = b.RemoveBinding("binding-gitea-v1"); err == nil {
		t.Fatal("unsupported encoding accepted")
	}
	if !bytes.Equal(b.source, data) {
		t.Fatal("source mutated")
	}
}

func TestRemoveBindingPreservesExplicitBoundaryComments(t *testing.T) {
	raw, err := os.ReadFile("../../examples/static/gitea.yaml")
	if err != nil {
		t.Fatal(err)
	}
	parts := strings.Split(string(raw), "---\n")
	for _, br := range []string{"\n", "\r\n", "\r", "\u0085", "\u2028", "\u2029"} {
		for _, position := range []int{0, 1, 3} {
			t.Run(fmtPosition(position, br), func(t *testing.T) {
				prefix := strings.Join(parts[:position], "...\n---\n")
				if prefix != "" {
					prefix += "...\n"
				}
				suffix := "\n# keep boundary documentation\n\n"
				if position < 3 {
					suffix += "%YAML 1.1\n---\n" + strings.Join(parts[position:3], "---\n")
				}
				source := prefix + "---\n" + parts[3] + "... # removed document end\n" + suffix
				b, e := Decode([]byte(strings.ReplaceAll(source, "\n", br)))
				if e != nil {
					t.Fatal(e)
				}
				out, e := b.RemoveBinding("binding-gitea-v1")
				if e != nil {
					t.Fatal(e)
				}
				want := strings.ReplaceAll(prefix+suffix, "\n", br)
				if string(out) != want {
					t.Fatalf("boundary bytes lost: got %q want %q", out, want)
				}
			})
		}
	}
}

func TestRemoveBindingPreservesCommentsAfterOnlyDocument(t *testing.T) {
	raw, err := os.ReadFile("../../examples/static/gitea.yaml")
	if err != nil {
		t.Fatal(err)
	}
	parts := strings.Split(string(raw), "---\n")
	b, err := Decode([]byte(parts[3] + "...\n# retain this operational note\n"))
	if err != nil {
		t.Fatal(err)
	}
	out, err := b.RemoveBinding("binding-gitea-v1")
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.HasSuffix(out, []byte("# retain this operational note\n")) {
		t.Fatal("last-object comment lost")
	}
	next, err := Decode(out)
	if err != nil || len(next.objects) != 1 || next.objects[0].Kind != "Configuration" {
		t.Fatal("missing explicit empty", err)
	}
}

func TestRemoveBindingDoesNotTreatScalarTextAsTerminator(t *testing.T) {
	raw, err := os.ReadFile("../../examples/static/gitea.yaml")
	if err != nil {
		t.Fatal(err)
	}
	parts := strings.Split(string(raw), "---\n")
	for _, note := range []string{"|\n      ...\n      keep reading", "\"first\n...not-a-marker\n      last\"", "'first\n      ...\n      last'"} {
		t.Run(note, func(t *testing.T) {
			binding := strings.Replace(parts[3], "metadata: {name: gitea-host, uid: binding-gitea-v1}", "metadata:\n  name: gitea-host\n  uid: binding-gitea-v1\n  annotations:\n    note: "+note, 1)
			if binding == parts[3] {
				t.Fatal("fixture metadata not replaced")
			}
			suffix := "# surviving comment\n---\n" + strings.Join(parts[:3], "---\n")
			b, err := Decode([]byte(binding + "...\t# terminator\n" + suffix))
			if err != nil {
				t.Fatal(err)
			}
			out, err := b.RemoveBinding("binding-gitea-v1")
			if err != nil || string(out) != suffix {
				t.Fatal("scalar/terminator boundary", err, string(out))
			}
		})
	}
}

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

func fmtPosition(n int, br string) string { return fmt.Sprintf("position%d-break%q", n, br) }

func TestRemoveBindingJSONCommentsAndEmpty(t *testing.T) {
	raw, e := os.ReadFile("../../examples/static/gitea.yaml")
	if e != nil {
		t.Fatal(e)
	}
	b, e := Decode(raw)
	if e != nil {
		t.Fatal(e)
	}
	var documents []string
	for _, o := range b.objects {
		data, err := json.Marshal(o)
		if err != nil {
			t.Fatal(err)
		}
		documents = append(documents, string(data))
	}
	for _, source := range []string{strings.Join(documents, "\n---\n"), "%YAML 1.1\n---\n" + string(raw), "\ufeff# header\n" + strings.ReplaceAll(string(raw), "---\n", "...\n# document boundary\n---\n")} {
		b, err := Decode([]byte(source))
		if err != nil {
			t.Fatal(err)
		}
		out, err := b.RemoveBinding("binding-gitea-v1")
		if err != nil {
			t.Fatal(err)
		}
		next, err := Decode(out)
		if err != nil || len(next.objects) != 3 {
			t.Fatal("presentation removal", err)
		}
	}
	parts := strings.Split(string(raw), "---\n")
	one, e := Decode([]byte(parts[3]))
	if e != nil {
		t.Fatal(e)
	}
	empty, e := one.RemoveBinding("binding-gitea-v1")
	if e != nil {
		t.Fatal(e)
	}
	decoded, e := Decode(empty)
	if e != nil || len(decoded.objects) != 1 || decoded.objects[0].Kind != "Configuration" {
		t.Fatal("not explicit empty", e)
	}
}
