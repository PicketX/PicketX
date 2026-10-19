// SPDX-License-Identifier: Apache-2.0
package configuration

import (
	"errors"
	"fmt"
	"slices"
	"strings"
)

var ErrBindingNotEditable = errors.New("Binding requires advanced editing")

// BindingEndpointEdit advertises only explicit, single-target host Bindings.
// It is a convenience projection, not an assertion that the Agent applied it.
type BindingEndpointEdit struct {
	ResourceName  string   `json:"resource_name"`
	ResourceUID   string   `json:"resource_uid"`
	EndpointNames []string `json:"endpoint_names"`
}

func bindingEndpointEdit(s bindingSpec) *BindingEndpointEdit {
	if len(s.Resources) != 1 || len(s.Agents) != 1 || s.Agents[0].AgentRef == nil || s.Agents[0].AgentSelector != nil || s.Enforcement.Class != "host-local" || s.Enforcement.Stage != "original-destination" {
		return nil
	}
	r := s.Resources[0]
	if r.ResourceRef.Name == "" || r.ResourceRef.UID == "" || len(r.EndpointNames) == 0 || r.EndpointSelector != nil {
		return nil
	}
	return &BindingEndpointEdit{r.ResourceRef.Name, r.ResourceRef.UID, slices.Clone(r.EndpointNames)}
}

// UpdateBindingEndpoints preserves identity, metadata and execution scope.
// The caller must compile and transactionally publish the whole candidate.
func (b *Bundle) UpdateBindingEndpoints(uid string, names []string) ([]byte, error) {
	index := slices.IndexFunc(b.objects, func(o document) bool { return o.Kind == "EnforcementBinding" && o.Metadata.UID == uid })
	if index < 0 {
		return nil, ErrBindingNotFound
	}
	original := b.objects[index]
	s := original.Spec.(bindingSpec)
	edit := bindingEndpointEdit(s)
	if edit == nil {
		return nil, ErrBindingNotEditable
	}
	if len(names) == 0 || len(names) > 64 {
		return nil, fmt.Errorf("invalid Endpoint names")
	}
	requested := map[string]bool{}
	for _, name := range names {
		if strings.TrimSpace(name) == "" || len(name) > 1024 || requested[name] {
			return nil, fmt.Errorf("invalid Endpoint names")
		}
		requested[name] = true
	}
	canonical := []string{}
	for _, o := range b.objects {
		if o.Kind == "Resource" && o.Metadata.Name == edit.ResourceName && o.Metadata.UID == edit.ResourceUID {
			for _, e := range o.Spec.(resourceSpec).Endpoints {
				if requested[e.Name] {
					canonical = append(canonical, e.Name)
				}
			}
		}
	}
	if len(canonical) != len(names) {
		return nil, fmt.Errorf("unknown Endpoint names")
	}
	s.Resources = slices.Clone(s.Resources)
	s.Resources[0].EndpointNames = canonical
	original.Spec = s
	remaining, err := b.removeDocument(index)
	if err != nil {
		return nil, err
	}
	rest, err := Decode(remaining)
	if err != nil {
		return nil, err
	}
	return rest.appendObject(original)
}
