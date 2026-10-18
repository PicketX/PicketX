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
	// Decode accepts YAML's UTF-16 encodings, but the source editor uses UTF-8
	// byte coordinates. Reject unsupported presentation before indexing lines.
	if !utf8.Valid(b.source) {
		return nil, fmt.Errorf("structured policy edits require UTF-8")
	}
	expected := slices.Delete(slices.Clone(b.objects), index, index+1)
	if len(expected) == 0 {
		return []byte("apiVersion: picketx.io/v1alpha1\nkind: Configuration\nmetadata: {name: empty, uid: empty-binding-removal}\nspec: {empty: true}\n"), nil
	}
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
		return nil, ErrBindingNotFound
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
		return nil, fmt.Errorf("Binding removal changed unrelated policy")
	}
	return out, nil
}
