// SPDX-License-Identifier: Apache-2.0
// Package strictjson rejects ambiguous JSON before decoding trusted input DTOs.
package strictjson

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"reflect"
	"strings"
)

func Decode(data []byte, target any) error {
	// Token walk rejects duplicate/null keys before Go's ordinary struct decoder
	// could overwrite fields or interpret null as an omitted safety constraint.
	check := json.NewDecoder(bytes.NewReader(data))
	if e := checkJSONValue(check, 0); e != nil {
		return e
	}
	if _, e := check.Token(); e != io.EOF {
		return fmt.Errorf("expected exactly one facts JSON document")
	}
	if e := exactFactFields(data, reflect.TypeOf(target)); e != nil {
		return e
	}
	d := json.NewDecoder(bytes.NewReader(data))
	d.DisallowUnknownFields()
	if e := d.Decode(target); e != nil {
		return e
	}
	return nil
}
func checkJSONValue(d *json.Decoder, depth int) error {
	if depth > 64 {
		return fmt.Errorf("facts nesting exceeds 64")
	}
	t, e := d.Token()
	if e != nil {
		return e
	}
	if t == nil {
		return fmt.Errorf("facts must not contain null")
	}
	delim, ok := t.(json.Delim)
	if !ok {
		return nil
	}
	switch delim {
	case '{':
		seen := map[string]bool{}
		for d.More() {
			key, e := d.Token()
			if e != nil {
				return e
			}
			s, ok := key.(string)
			if !ok || seen[s] {
				return fmt.Errorf("duplicate or invalid facts key %q", key)
			}
			seen[s] = true
			if e = checkJSONValue(d, depth+1); e != nil {
				return e
			}
		}
		end, e := d.Token()
		if e != nil || end != json.Delim('}') {
			return fmt.Errorf("invalid facts object")
		}
	case '[':
		for d.More() {
			if e := checkJSONValue(d, depth+1); e != nil {
				return e
			}
		}
		end, e := d.Token()
		if e != nil || end != json.Delim(']') {
			return fmt.Errorf("invalid facts array")
		}
	default:
		return fmt.Errorf("invalid facts token")
	}
	return nil
}

// Require exact struct field spellings, including Unicode. Go JSON otherwise
// accepts Unicode-fold aliases such as long-s. Label maps keep their own keys.
func exactFactFields(data []byte, typ reflect.Type) error {
	for typ.Kind() == reflect.Pointer {
		typ = typ.Elem()
	}
	switch typ.Kind() {
	case reflect.Struct:
		if len(bytes.TrimSpace(data)) == 0 || bytes.TrimSpace(data)[0] != '{' {
			return nil
		}
		var fields map[string]json.RawMessage
		if e := json.Unmarshal(data, &fields); e != nil {
			return e
		}
		types := map[string]reflect.Type{}
		for i := 0; i < typ.NumField(); i++ {
			f := typ.Field(i)
			if !f.IsExported() {
				continue
			}
			name, _, _ := strings.Cut(f.Tag.Get("json"), ",")
			if name == "" {
				name = f.Name
			}
			types[name] = f.Type
		}
		seen := map[string]bool{}
		for key, value := range fields {
			name := key
			ft, ok := types[name]
			if !ok {
				return fmt.Errorf("unknown facts field %q", key)
			}
			if seen[name] {
				return fmt.Errorf("duplicate case-aliased facts field %q", key)
			}
			seen[name] = true
			if e := exactFactFields(value, ft); e != nil {
				return e
			}
		}
	case reflect.Slice:
		var items []json.RawMessage
		if e := json.Unmarshal(data, &items); e != nil {
			return e
		}
		for _, item := range items {
			if e := exactFactFields(item, typ.Elem()); e != nil {
				return e
			}
		}
	}
	return nil
}
