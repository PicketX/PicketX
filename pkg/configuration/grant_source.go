// SPDX-License-Identifier: Apache-2.0
package configuration

import (
	"bytes"
	"fmt"
	"io"
	"strings"
	"unicode/utf8"

	"go.yaml.in/yaml/v3"
)

// Do not reserialize accepted policy for a one-bit withdrawal. YAML may be much
// smaller than JSON (HTML, quotes, multiline scalars, dense input). Parser-owned
// coordinates locate the edit; RevokeGrant re-decodes and compares the WHOLE
// semantic result before returning it. The existing byte/object bounds remain.
func revokeSource(data []byte, uid string) ([]byte, error) {
	if !utf8.Valid(data) {
		return nil, fmt.Errorf("structured policy edits require UTF-8")
	}
	lines := []int{0}
	for i := 0; i < len(data); {
		if size := yamlBreak(data, i); size > 0 {
			i += size
			lines = append(lines, i)
		} else {
			i++
		}
	}
	point := func(n *yaml.Node) int {
		p := lines[n.Line-1]
		// The YAML scanner does not count a byte-order mark as a content column.
		if bytes.HasPrefix(data[p:], []byte{0xef, 0xbb, 0xbf}) {
			p += 3
		}
		for col := 1; col < n.Column && p < len(data); col++ {
			_, size := utf8.DecodeRune(data[p:])
			p += size
		}
		return p
	}
	decoder := yaml.NewDecoder(bytes.NewReader(data))
	for {
		var doc yaml.Node
		if err := decoder.Decode(&doc); err == io.EOF {
			break
		} else if err != nil {
			return nil, err
		}
		root := doc.Content[0]
		_, meta := yamlField(root, "metadata")
		_, id := yamlField(meta, "uid")
		if id == nil || id.Value != uid {
			continue
		}
		specKey, spec := yamlField(root, "spec")
		if spec == nil || spec.Kind != yaml.MappingNode {
			return nil, fmt.Errorf("Grant spec is not a mapping")
		}
		key, value := yamlField(spec, "revoked")
		start, end, text := 0, 0, ""
		if value != nil {
			start = point(value)
			var err error
			end, text, err = booleanEnd(data, start, lines[key.Line-1], value.Style)
			if err != nil {
				return nil, err
			}
		} else if spec.Style&yaml.FlowStyle != 0 {
			start = taggedStart(data, point(spec))
			if start >= len(data) || data[start] != '{' {
				return nil, fmt.Errorf("missing flow mapping delimiter")
			}
			start++
			end = start
			text = `"revoked":true, `
		} else {
			if len(spec.Content) == 0 {
				return nil, fmt.Errorf("empty block mapping")
			}
			first := spec.Content[0]
			if first.Line == specKey.Line {
				return nil, fmt.Errorf("unsupported inline block mapping")
			}
			start = lines[first.Line-1]
			end = start
			p := start
			for p < len(data) && data[p] == ' ' {
				p++
			}
			text = strings.Repeat(" ", p-start) + "revoked: true\n"
		}
		out := make([]byte, 0, len(data)+len(text)-(end-start))
		out = append(out, data[:start]...)
		out = append(out, text...)
		out = append(out, data[end:]...)
		return out, nil
	}
	return nil, ErrGrantNotFound
}
func yamlField(n *yaml.Node, name string) (*yaml.Node, *yaml.Node) {
	if n == nil || n.Kind != yaml.MappingNode {
		return nil, nil
	}
	for i := 0; i < len(n.Content); i += 2 {
		if n.Content[i].Value == name {
			return n.Content[i], n.Content[i+1]
		}
	}
	return nil, nil
}

// The strict decoder accepts only literal false/true boolean VALUES, but their
// YAML presentation can include a bool tag, quotes/escapes or a block scalar.
func booleanEnd(data []byte, start, keyLine int, style yaml.Style) (int, string, error) {
	p := taggedStart(data, start)
	if p >= len(data) {
		return 0, "", fmt.Errorf("missing boolean scalar")
	}
	if style&(yaml.LiteralStyle|yaml.FoldedStyle) != 0 {
		indent := 0
		for keyLine+indent < len(data) && data[keyLine+indent] == ' ' {
			indent++
		}
		p = nextYAMLLine(data, p)
		for p < len(data) {
			line := p
			spaces := 0
			for p < len(data) && data[p] == ' ' {
				p++
				spaces++
			}
			if p < len(data) && yamlBreak(data, p) == 0 && spaces <= indent {
				return line, "true\n", nil
			}
			p = nextYAMLLine(data, p)
		}
		return p, "true\n", nil
	}
	if data[p] == '\'' || data[p] == '"' {
		quote := data[p]
		p++
		for p < len(data) {
			if data[p] == quote {
				if quote == '\'' && p+1 < len(data) && data[p+1] == '\'' {
					p += 2
					continue
				}
				return p + 1, "true", nil
			}
			if quote == '"' && data[p] == '\\' {
				p++
				if p >= len(data) {
					break
				}
			}
			p++
		}
		return 0, "", fmt.Errorf("unterminated boolean scalar")
	}
	if bytes.HasPrefix(data[p:], []byte("false")) {
		return p + 5, "true", nil
	}
	return 0, "", fmt.Errorf("unsupported boolean presentation")
}
func taggedStart(data []byte, p int) int {
	if p >= len(data) || data[p] != '!' {
		return p
	}
	if p+1 < len(data) && data[p+1] == '<' {
		for p < len(data) && data[p] != '>' {
			p++
		}
		if p < len(data) {
			p++
		}
	} else {
		for p < len(data) && data[p] != ' ' && data[p] != '\t' && yamlBreak(data, p) == 0 {
			p++
		}
	}
	for p < len(data) {
		if data[p] == ' ' || data[p] == '\t' {
			p++
			continue
		}
		if size := yamlBreak(data, p); size > 0 {
			p += size
			continue
		}
		if data[p] == '#' {
			p = nextYAMLLine(data, p)
			continue
		}
		break
	}
	return p
}
func nextYAMLLine(data []byte, p int) int {
	for p < len(data) {
		if size := yamlBreak(data, p); size > 0 {
			return p + size
		}
		p++
	}
	return p
}

// Match the selected yaml.v3 scanner: CRLF, CR, LF, NEL, LS and PS.
func yamlBreak(data []byte, p int) int {
	if p >= len(data) {
		return 0
	}
	if data[p] == '\r' {
		if p+1 < len(data) && data[p+1] == '\n' {
			return 2
		}
		return 1
	}
	if data[p] == '\n' {
		return 1
	}
	if p+1 < len(data) && data[p] == 0xc2 && data[p+1] == 0x85 {
		return 2
	}
	if p+2 < len(data) && data[p] == 0xe2 && data[p+1] == 0x80 && (data[p+2] == 0xa8 || data[p+2] == 0xa9) {
		return 3
	}
	return 0
}
