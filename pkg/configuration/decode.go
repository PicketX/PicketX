// SPDX-License-Identifier: Apache-2.0
package configuration

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"reflect"
	"regexp"
	"slices"
	"strconv"
	"strings"

	"go.yaml.in/yaml/v3"
)

var uidPattern = regexp.MustCompile(`^[A-Za-z0-9._-]{1,128}$`)

// Decode validates one complete bounded multi-document YAML publication.
// Empty/ambiguous input produces no Bundle; withdrawal must be explicit.
func Decode(data []byte) (*Bundle, error) {
	if len(data) > MaxBytes {
		return nil, fmt.Errorf("configuration exceeds %d bytes", MaxBytes)
	}
	dec := yaml.NewDecoder(bytes.NewReader(data))
	b := &Bundle{}
	names, uids := map[string]bool{}, map[string]bool{}
	for {
		var node yaml.Node
		if err := dec.Decode(&node); err == io.EOF {
			break
		} else if err != nil {
			return nil, fmt.Errorf("document %d: %w", len(b.objects)+1, err)
		}
		if len(b.objects) >= MaxObjects {
			return nil, fmt.Errorf("too many configuration objects")
		}
		if len(node.Content) != 1 {
			return nil, fmt.Errorf("expected one object per document")
		}
		value, err := nodeValue(node.Content[0], 0)
		if err != nil {
			return nil, fmt.Errorf("document %d: %w", len(b.objects)+1, err)
		}
		raw, err := json.Marshal(value)
		if err != nil {
			return nil, err
		}
		var h header
		if err = strictJSON(raw, &h); err != nil {
			return nil, fmt.Errorf("document %d: %w", len(b.objects)+1, err)
		}
		if h.APIVersion != APIVersion {
			return nil, fmt.Errorf("unsupported apiVersion %q", h.APIVersion)
		}
		if h.Metadata.Name == "" || !uidPattern.MatchString(h.Metadata.UID) {
			return nil, fmt.Errorf("%s requires nonempty metadata.name and valid explicit uid", h.Kind)
		}
		key := h.Kind + "/" + h.Metadata.Name
		if names[key] || uids[h.Metadata.UID] {
			return nil, fmt.Errorf("duplicate object name or uid: %s", key)
		}
		names[key], uids[h.Metadata.UID] = true, true
		var spec any
		switch h.Kind {
		case "Resource":
			var s resourceSpec
			if err = strictJSON(h.Spec, &s); err != nil {
				return nil, fmt.Errorf("%s: %w", key, err)
			}
			for i := range s.Endpoints {
				if s.Endpoints[i].Type == nil {
					v := "network"
					s.Endpoints[i].Type = &v
				}
			}
			spec = s
		case "Grant", "NetworkDeny":
			var s contributionSpec
			if err = strictJSON(h.Spec, &s); err != nil {
				return nil, fmt.Errorf("%s: %w", key, err)
			}
			for i := range s.Scopes {
				if s.Scopes[i].Actions == nil {
					s.Scopes[i].Actions = []string{"connect"}
				}
			}
			spec = s
		case "EnforcementBinding":
			var s bindingSpec
			if err = strictJSON(h.Spec, &s); err != nil {
				return nil, fmt.Errorf("%s: %w", key, err)
			}
			spec = s
		case "HTTPRoute":
			var s httpRouteSpec
			if err = strictJSON(h.Spec, &s); err != nil {
				return nil, fmt.Errorf("%s: %w", key, err)
			}
			spec = s
		case "Configuration":
			var s emptySpec
			if err = strictJSON(h.Spec, &s); err != nil {
				return nil, fmt.Errorf("%s: %w", key, err)
			}
			if !s.Empty {
				return nil, fmt.Errorf("Configuration requires explicit empty: true")
			}
			spec = s
		default:
			return nil, fmt.Errorf("unsupported kind %q", h.Kind)
		}
		b.objects = append(b.objects, document{h.APIVersion, h.Kind, h.Metadata, spec})
	}
	if len(b.objects) == 0 {
		return nil, fmt.Errorf("configuration is empty; use explicit Configuration empty: true")
	}
	for _, o := range b.objects {
		if o.Kind == "Configuration" && len(b.objects) != 1 {
			return nil, fmt.Errorf("empty Configuration cannot coexist with policy objects")
		}
	}
	slices.SortFunc(b.objects, func(a, b document) int {
		if a.Kind < b.Kind {
			return -1
		}
		if a.Kind > b.Kind {
			return 1
		}
		if a.Metadata.Name < b.Metadata.Name {
			return -1
		}
		if a.Metadata.Name > b.Metadata.Name {
			return 1
		}
		return 0
	})
	canonical, err := json.Marshal(b.objects)
	if err != nil {
		return nil, err
	}
	b.hash = fmt.Sprintf("%x", sha256.Sum256(canonical))
	return b, nil
}
func strictJSON(data []byte, out any) error {
	if err := exactFields(data, reflect.TypeOf(out)); err != nil {
		return err
	}
	d := json.NewDecoder(bytes.NewReader(data))
	d.DisallowUnknownFields()
	if err := d.Decode(out); err != nil {
		return err
	}
	if err := d.Decode(new(any)); err != io.EOF {
		return fmt.Errorf("expected exactly one JSON value")
	}
	return nil
}

// encoding/json accepts case-insensitive field aliases even with
// DisallowUnknownFields. The policy contract uses exact field spellings.
func exactFields(data []byte, typ reflect.Type) error {
	for typ.Kind() == reflect.Pointer {
		typ = typ.Elem()
	}
	if typ == reflect.TypeOf(json.RawMessage{}) {
		return nil
	}
	switch typ.Kind() {
	case reflect.Struct:
		var fields map[string]json.RawMessage
		if err := json.Unmarshal(data, &fields); err != nil {
			return err
		}
		allowed := map[string]reflect.Type{}
		for i := 0; i < typ.NumField(); i++ {
			field := typ.Field(i)
			name, _, _ := strings.Cut(field.Tag.Get("json"), ",")
			if name == "" {
				name = field.Name
			}
			allowed[name] = field.Type
		}
		for name, value := range fields {
			fieldType, ok := allowed[name]
			if !ok {
				return fmt.Errorf("unknown field %q", name)
			}
			if err := exactFields(value, fieldType); err != nil {
				return err
			}
		}
	case reflect.Slice:
		var items []json.RawMessage
		if err := json.Unmarshal(data, &items); err != nil {
			return err
		}
		for _, value := range items {
			if err := exactFields(value, typ.Elem()); err != nil {
				return err
			}
		}
	}
	return nil
}

// nodeValue avoids YAML's implicit numeric/string conversions and preserves
// omitted versus supplied collections before strict typed decoding.
func nodeValue(n *yaml.Node, depth int) (any, error) {
	if depth > maxDepth {
		return nil, fmt.Errorf("configuration nesting exceeds %d", maxDepth)
	}
	if n.Anchor != "" || n.Kind == yaml.AliasNode {
		return nil, fmt.Errorf("YAML anchors and aliases are unsupported")
	}
	switch n.Kind {
	case yaml.MappingNode:
		if n.Tag != "!!map" {
			return nil, fmt.Errorf("unsupported mapping tag %q", n.Tag)
		}
		m := map[string]any{}
		for i := 0; i < len(n.Content); i += 2 {
			k := n.Content[i]
			if k.Kind != yaml.ScalarNode || k.Tag != "!!str" || k.Value == "<<" || k.Anchor != "" {
				return nil, fmt.Errorf("mapping keys must be plain strings; merges are unsupported")
			}
			if _, ok := m[k.Value]; ok {
				return nil, fmt.Errorf("duplicate key %q at line %d", k.Value, k.Line)
			}
			v, e := nodeValue(n.Content[i+1], depth+1)
			if e != nil {
				return nil, e
			}
			m[k.Value] = v
		}
		return m, nil
	case yaml.SequenceNode:
		if n.Tag != "!!seq" {
			return nil, fmt.Errorf("unsupported sequence tag %q", n.Tag)
		}
		a := make([]any, len(n.Content))
		for i, c := range n.Content {
			v, e := nodeValue(c, depth+1)
			if e != nil {
				return nil, e
			}
			a[i] = v
		}
		return a, nil
	case yaml.ScalarNode:
		switch n.Tag {
		case "!!str", "!!timestamp":
			return n.Value, nil
		case "!!bool":
			if n.Value == "true" {
				return true, nil
			}
			if n.Value == "false" {
				return false, nil
			}
			return nil, fmt.Errorf("boolean must be true or false")
		case "!!int":
			// Only unsigned decimal syntax is needed by the current policy schema.
			if n.Value == "" || (len(n.Value) > 1 && n.Value[0] == '0') {
				return nil, fmt.Errorf("integer must use decimal syntax")
			}
			for _, c := range n.Value {
				if c < '0' || c > '9' {
					return nil, fmt.Errorf("integer must use unsigned decimal syntax")
				}
			}
			v, e := strconv.ParseUint(n.Value, 10, 64)
			return v, e
		case "!!null":
			return nil, fmt.Errorf("explicit null is forbidden (line %d)", n.Line)
		default:
			return nil, fmt.Errorf("unsupported scalar tag %q", n.Tag)
		}
	default:
		return nil, fmt.Errorf("unsupported YAML node")
	}
}
