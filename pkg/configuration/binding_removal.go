// SPDX-License-Identifier: Apache-2.0
package configuration

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"slices"
	"unicode/utf8"

	"go.yaml.in/yaml/v3"
)

var ErrBindingNotFound = errors.New("Binding not found")

// RemoveBinding withdraws protection, not access permission. The caller must
// compile/Replace the complete result before accepting it. Other objects survive.
func (b *Bundle) RemoveBinding(uid string) ([]byte, error) {
	index := slices.IndexFunc(b.objects, func(o document) bool { return o.Kind == "EnforcementBinding" && o.Metadata.UID == uid })
	if index < 0 {
		return nil, ErrBindingNotFound
	}
	return b.removeDocument(index)
}

// removeDocument preserves all surviving semantic objects and source bytes.
// Callers enforce kind-specific lifecycle/reference restrictions first.
func (b *Bundle) removeDocument(index int) ([]byte, error) {
	uid := b.objects[index].Metadata.UID
	// Decode accepts YAML's UTF-16 encodings, but the source editor uses UTF-8
	// byte coordinates. Reject unsupported presentation before indexing lines.
	if !utf8.Valid(b.source) {
		return nil, fmt.Errorf("structured policy edits require UTF-8")
	}
	expected := slices.Delete(slices.Clone(b.objects), index, index+1)
	lines := []int{0}
	for p := 0; p < len(b.source); {
		if n := yamlBreak(b.source, p); n > 0 {
			p += n
			lines = append(lines, p)
		} else {
			p++
		}
	}
	dec := yaml.NewDecoder(bytes.NewReader(b.source))
	start, end := -1, len(b.source)
	for {
		var doc yaml.Node
		if err := dec.Decode(&doc); err == io.EOF {
			break
		} else if err != nil {
			return nil, err
		}
		if start >= 0 {
			end = lines[doc.Line-1]
			break
		}
		_, meta := yamlField(doc.Content[0], "metadata")
		_, id := yamlField(meta, "uid")
		if id != nil && id.Value == uid {
			start = lines[doc.Line-1]
		}
	}
	if start < 0 {
		return nil, fmt.Errorf("target document not found in source")
	}
	// Document.Line identifies the next document, not the end of this one.
	// An explicit column-zero end marker closes the removed document sooner;
	// comments/blank lines/directives after its line belong to the surviving stream.
	for i, p := range lines {
		if p < start || p+3 > end || !bytes.Equal(b.source[p:p+3], []byte("...")) {
			continue
		}
		q := p + 3
		if q < len(b.source) && b.source[q] != ' ' && b.source[q] != '\t' && yamlBreak(b.source, q) == 0 {
			continue
		}
		if i+1 < len(lines) {
			end = lines[i+1]
		}
		break
	}
	if len(expected) == 0 {
		empty := []byte("apiVersion: picketx.io/v1alpha1\nkind: Configuration\nmetadata: {name: empty, uid: empty-binding-removal}\nspec: {empty: true}\n")
		return append(empty, b.source[end:]...), nil
	}
	out := append(slices.Clone(b.source[:start]), b.source[end:]...)
	next, err := Decode(out)
	if err != nil {
		return nil, err
	}
	want, err := json.Marshal(expected)
	if err != nil {
		return nil, err
	}
	got, err := json.Marshal(next.objects)
	if err != nil || !bytes.Equal(want, got) {
		return nil, fmt.Errorf("document removal changed unrelated policy")
	}
	return out, nil
}
