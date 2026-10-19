// SPDX-License-Identifier: Apache-2.0
package configuration

import (
	"bytes"
	"encoding/json"
	"fmt"
	"slices"
	"strings"

	"github.com/PicketX/PicketX/pkg/networkplan"
)

type InventoryEndpoint struct {
	Name      string   `json:"name"`
	Protocol  string   `json:"protocol"`
	Port      uint16   `json:"port"`
	Addresses []string `json:"addresses,omitempty"`
}
type InventoryResource struct {
	Name      string              `json:"name"`
	UID       string              `json:"uid"`
	Endpoints []InventoryEndpoint `json:"endpoints"`
}
type InventoryBinding struct {
	Name string `json:"name"`
	UID  string `json:"uid"`
	// JSON preserves imported named/all/selector selections without claiming matches.
	Resources    string               `json:"resources"`
	Agents       string               `json:"agents"`
	Enforcement  string               `json:"enforcement"`
	EndpointEdit *BindingEndpointEdit `json:"endpoint_edit,omitempty"`
}
type InventoryContext struct {
	AgentName    string   `json:"agent_name"`
	ID           string   `json:"id"`
	Profile      string   `json:"profile"`
	Class        string   `json:"class"`
	Stage        string   `json:"stage"`
	Destinations []string `json:"destinations"`
	Handling     string   `json:"handling"`
	Selectable   bool     `json:"selectable"`
}
type Inventory struct {
	HTTPRoutes []InventoryHTTPRoute `json:"http_routes"`
	Resources  []InventoryResource  `json:"resources"`
	Bindings   []InventoryBinding   `json:"bindings"`
	Contexts   []InventoryContext   `json:"contexts"`
}

func (b *Bundle) Inventory(f Facts) Inventory {
	out := Inventory{Resources: []InventoryResource{}, Bindings: []InventoryBinding{}, Contexts: []InventoryContext{}, HTTPRoutes: []InventoryHTTPRoute{}}
	for _, o := range b.objects {
		switch s := o.Spec.(type) {
		case httpRouteSpec:
			out.HTTPRoutes = append(out.HTTPRoutes, InventoryHTTPRoute{o.Metadata.Name, o.Metadata.UID, s.ResourceRef.Name, s.ResourceRef.UID, s.EndpointName, s.Host, slices.Clone(s.Methods), slices.Clone(s.Paths)})
		case resourceSpec:
			r := InventoryResource{Name: o.Metadata.Name, UID: o.Metadata.UID, Endpoints: []InventoryEndpoint{}}
			for _, e := range s.Endpoints {
				r.Endpoints = append(r.Endpoints, InventoryEndpoint{e.Name, e.Network.Protocol, e.Network.Port, slices.Clone(e.Network.Addresses)})
			}
			out.Resources = append(out.Resources, r)
		case bindingSpec:
			resources, _ := json.Marshal(s.Resources)
			agents, _ := json.Marshal(s.Agents)
			enforcement, _ := json.Marshal(s.Enforcement)
			out.Bindings = append(out.Bindings, InventoryBinding{o.Metadata.Name, o.Metadata.UID, string(resources), string(agents), string(enforcement), bindingEndpointEdit(s)})
		}
	}
	for _, c := range f.Contexts {
		v := InventoryContext{AgentName: c.AgentRef, ID: c.ID, Profile: c.ProfileRef, Class: string(c.Class), Stage: string(c.Stage), Destinations: []string{}, Handling: string(c.Handling), Selectable: c.Class == networkplan.HostLocal && c.Stage == networkplan.OriginalDestination}
		for _, p := range c.Destinations {
			v.Destinations = append(v.Destinations, p.String())
		}
		out.Contexts = append(out.Contexts, v)
	}
	return out
}

type InventoryHTTPRoute struct {
	Name         string   `json:"name"`
	UID          string   `json:"uid"`
	ResourceName string   `json:"resource_name"`
	ResourceUID  string   `json:"resource_uid"`
	EndpointName string   `json:"endpoint_name"`
	Host         string   `json:"host"`
	Methods      []string `json:"methods"`
	Paths        []string `json:"paths"`
}

// AddHTTPRoute preserves existing source. The complete candidate compiler must
// validate the pinned TCP target and selectors before the result is published.
func (b *Bundle) AddHTTPRoute(in InventoryHTTPRoute) ([]byte, error) {
	if strings.TrimSpace(in.Name) == "" || len(in.Name) > 128 || !uidPattern.MatchString(in.UID) {
		return nil, fmt.Errorf("invalid HTTPRoute")
	}
	return b.appendObject(document{APIVersion, "HTTPRoute", metadata{Name: in.Name, UID: in.UID}, httpRouteSpec{
		ResourceRef: resourceRef{Name: in.ResourceName, UID: in.ResourceUID}, EndpointName: in.EndpointName,
		Host: in.Host, Methods: slices.Clone(in.Methods), Paths: slices.Clone(in.Paths),
	}})
}

func (b *Bundle) appendObject(o document) ([]byte, error) {
	var out bytes.Buffer
	if !(len(b.objects) == 1 && b.objects[0].Kind == "Configuration") {
		out.Write(b.source)
		out.WriteString("\n---\n")
	}
	enc := json.NewEncoder(&out)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(o); err != nil {
		return nil, err
	}
	if _, err := Decode(out.Bytes()); err != nil {
		return nil, err
	}
	return out.Bytes(), nil
}

func (b *Bundle) AddResource(name, uid string, entries []InventoryEndpoint) ([]byte, error) {
	if strings.TrimSpace(name) == "" || len(name) > 128 || !uidPattern.MatchString(uid) || len(entries) < 1 || len(entries) > 64 {
		return nil, fmt.Errorf("invalid resource")
	}
	s := resourceSpec{Endpoints: []endpoint{}}
	seen := map[string]bool{}
	for _, e := range entries {
		if strings.TrimSpace(e.Name) == "" || len(e.Name) > 128 || seen[e.Name] || (e.Protocol != "TCP" && e.Protocol != "UDP") || e.Port == 0 {
			return nil, fmt.Errorf("invalid endpoint")
		}
		seen[e.Name] = true
		if e.Addresses != nil && (len(e.Addresses) == 0 || len(e.Addresses) > 64) {
			return nil, fmt.Errorf("invalid addresses")
		}
		if _, err := networkplan.ParseDestinations(e.Addresses); err != nil {
			return nil, err
		}
		s.Endpoints = append(s.Endpoints, endpoint{Name: e.Name, Network: network{Protocol: e.Protocol, Port: e.Port, Addresses: slices.Clone(e.Addresses)}})
	}
	return b.appendObject(document{APIVersion, "Resource", metadata{Name: name, UID: uid}, s})
}

type BindingInput struct {
	Name, UID, ResourceName, ResourceUID, AgentName, ContextID string
	EndpointNames                                              []string
}

func (b *Bundle) AddBinding(in BindingInput, f Facts) ([]byte, error) {
	if strings.TrimSpace(in.Name) == "" || len(in.Name) > 128 || !uidPattern.MatchString(in.UID) || len(in.EndpointNames) < 1 || len(in.EndpointNames) > 64 {
		return nil, fmt.Errorf("invalid binding")
	}
	names := map[string]bool{}
	found := false
	for _, o := range b.objects {
		if o.Kind == "Resource" && o.Metadata.Name == in.ResourceName && o.Metadata.UID == in.ResourceUID {
			found = true
			for _, e := range o.Spec.(resourceSpec).Endpoints {
				names[e.Name] = true
			}
		}
	}
	if !found {
		return nil, fmt.Errorf("pinned resource missing")
	}
	for _, n := range in.EndpointNames {
		if !names[n] {
			return nil, fmt.Errorf("missing or repeated endpoint")
		}
		delete(names, n)
	}
	var context *networkplan.ExecutionContext
	for _, c := range f.Contexts {
		if c.AgentRef == in.AgentName && c.ID == in.ContextID {
			if context != nil {
				return nil, fmt.Errorf("ambiguous context")
			}
			copy := c
			context = &copy
		}
	}
	if context == nil || context.Class != networkplan.HostLocal || context.Stage != networkplan.OriginalDestination {
		return nil, fmt.Errorf("unsupported or missing context")
	}
	destinations := []string{}
	for _, p := range context.Destinations {
		destinations = append(destinations, p.String())
	}
	s := bindingSpec{Resources: []selection{{ResourceRef: resourceRef{in.ResourceName, in.ResourceUID}, EndpointNames: slices.Clone(in.EndpointNames)}}, Agents: []agentSelection{{AgentRef: &agentRef{Name: in.AgentName}}}, Enforcement: enforcement{ProfileRef: context.ProfileRef, Class: string(context.Class), Stage: string(context.Stage), AllowedDestinations: destinations, Handling: string(context.Handling)}}
	return b.appendObject(document{APIVersion, "EnforcementBinding", metadata{Name: in.Name, UID: in.UID}, s})
}
