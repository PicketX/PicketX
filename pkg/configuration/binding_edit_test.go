// SPDX-License-Identifier: Apache-2.0
package configuration

import (
	"encoding/json"
	"errors"
	"os"
	"reflect"
	"slices"
	"strings"
	"testing"
)

// A convenience edit must not convert multi-target or dynamic scope to named
// scope, even when an advanced caller can publish the original document.
func TestBindingEndpointEditShapeBoundary(t *testing.T) {
	raw, e := os.ReadFile("../../examples/static/gitea.yaml")
	if e != nil {
		t.Fatal(e)
	}
	b, e := Decode(raw)
	if e != nil {
		t.Fatal(e)
	}
	index := slices.IndexFunc(b.objects, func(o document) bool { return o.Kind == "EnforcementBinding" })
	valid := b.objects[index].Spec.(bindingSpec)
	valid.Resources[0].EndpointNames = []string{"https"}
	for name, change := range map[string]func(*bindingSpec){
		"all": func(s *bindingSpec) { s.Resources[0].EndpointNames = nil },
		"selector": func(s *bindingSpec) {
			s.Resources[0].EndpointSelector = &selector{MatchLabels: map[string]string{"x": "y"}}
		},
		"resources": func(s *bindingSpec) { s.Resources = append(s.Resources, s.Resources[0]) },
		"agents":    func(s *bindingSpec) { s.Agents = append(s.Agents, s.Agents[0]) },
		"agent selector": func(s *bindingSpec) {
			s.Agents[0].AgentRef = nil
			s.Agents[0].AgentSelector = &selector{MatchLabels: map[string]string{"x": "y"}}
		},
		"stage": func(s *bindingSpec) { s.Enforcement.Stage = "translated-destination" },
		"class": func(s *bindingSpec) { s.Enforcement.Class = "forwarded" },
	} {
		t.Run(name, func(t *testing.T) {
			data, _ := json.Marshal(valid)
			var s bindingSpec
			if e := json.Unmarshal(data, &s); e != nil {
				t.Fatal(e)
			}
			change(&s)
			b.objects[index].Spec = s
			if b.Inventory(Facts{}).Bindings[0].EndpointEdit != nil {
				t.Fatal("unsafe edit advertised")
			}
			if _, e := b.UpdateBindingEndpoints("binding-gitea-v1", []string{"ssh"}); !errors.Is(e, ErrBindingNotEditable) {
				t.Fatal(e)
			}
		})
	}
	b.objects[index].Spec = valid
	before, _ := json.Marshal(b.objects)
	out, e := b.UpdateBindingEndpoints("binding-gitea-v1", []string{"ssh"})
	if e != nil {
		t.Fatal(e)
	}
	after, _ := json.Marshal(b.objects)
	if !reflect.DeepEqual(before, after) {
		t.Fatal("mutated original Bundle")
	}
	if !strings.HasPrefix(string(out), strings.Join(strings.Split(string(raw), "---\n")[:3], "---\n")) {
		t.Fatal("unrelated source changed")
	}
}
