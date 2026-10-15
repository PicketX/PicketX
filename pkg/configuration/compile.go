// SPDX-License-Identifier: Apache-2.0
package configuration

import (
	"fmt"
	"strings"
	"time"

	"github.com/PicketX/PicketX/pkg/core"
	"github.com/PicketX/PicketX/pkg/networkplan"
)

// Facts are separate operator/runtime inputs. They are not policy-authored
// observations or evidence of a live network namespace or installed rules.
type Facts struct {
	AuthorityID string                         `json:"authorityID"`
	Agents      []networkplan.Agent            `json:"agents"`
	Contexts    []networkplan.ExecutionContext `json:"contexts"`
}

func translate(b *Bundle, f Facts) (networkplan.Input, error) {
	in := networkplan.Input{AuthorityID: f.AuthorityID, Agents: f.Agents, Contexts: f.Contexts}
	resources := map[string]string{}
	for _, o := range b.objects {
		s, ok := o.Spec.(resourceSpec)
		if !ok {
			continue
		}
		if len(s.Endpoints) == 0 {
			return in, fmt.Errorf("Resource %q endpoints must not be empty", o.Metadata.Name)
		}
		resources[o.Metadata.Name] = o.Metadata.UID
		r := core.Resource{Name: o.Metadata.UID, Labels: o.Metadata.Labels}
		for _, ep := range s.Endpoints {
			if ep.Type == nil || *ep.Type != "network" {
				return in, fmt.Errorf("unsupported endpoint type")
			}
			if ep.Network.Protocol != "TCP" && ep.Network.Protocol != "UDP" {
				return in, fmt.Errorf("endpoint %q requires TCP or UDP", ep.Name)
			}
			addresses, e := networkplan.ParseDestinations(ep.Network.Addresses)
			if e != nil {
				return in, e
			}
			r.Endpoints = append(r.Endpoints, core.Endpoint{Name: ep.Name, Type: *ep.Type, Labels: ep.Labels})
			in.Networks = append(in.Networks, networkplan.NetworkDefinition{Entry: networkplan.EntryRef{ResourceName: r.Name, EndpointName: ep.Name}, Protocol: networkplan.Protocol(strings.ToLower(ep.Network.Protocol)), Port: ep.Network.Port, Addresses: addresses})
		}
		in.Logical.Resources = append(in.Logical.Resources, r)
	}
	resolve := func(ref resourceRef) (string, error) {
		uid, ok := resources[ref.Name]
		if !ok || ref.UID == "" || uid != ref.UID {
			return "", fmt.Errorf("resource reference %q/%q is missing or UID differs", ref.Name, ref.UID)
		}
		return uid, nil
	}
	for _, o := range b.objects {
		switch s := o.Spec.(type) {
		case contributionSpec:
			if s.Subject.Type != "source" {
				return in, fmt.Errorf("unsupported subject type %q", s.Subject.Type)
			}
			source, e := core.ParseSource(s.Subject.CIDR)
			if e != nil {
				return in, e
			}
			v, e := parseValidity(s.Validity)
			if e != nil {
				return in, e
			}
			g := core.Grant{ID: o.Metadata.UID, Source: source, Validity: v, Revoked: s.Revoked, SubjectType: "source"}
			for _, sc := range s.Scopes {
				uid, e := resolve(sc.ResourceRef)
				if e != nil {
					return in, e
				}
				sel, e := endpointSelection(sc.EndpointNames, sc.EndpointSelector)
				if e != nil {
					return in, e
				}
				g.Scopes = append(g.Scopes, core.Scope{ResourceName: uid, Endpoints: sel, Actions: sc.Actions})
			}
			if o.Kind == "Grant" {
				in.Logical.Grants = append(in.Logical.Grants, g)
			} else {
				in.Logical.Denies = append(in.Logical.Denies, core.Deny(g))
			}
		case bindingSpec:
			dest, e := networkplan.ParseDestinations(s.Enforcement.AllowedDestinations)
			if e != nil {
				return in, e
			}
			bind := networkplan.Binding{ID: o.Metadata.UID, Context: networkplan.ContextRequest{ProfileRef: s.Enforcement.ProfileRef, Class: networkplan.TrafficClass(s.Enforcement.Class), Stage: networkplan.ObservationStage(s.Enforcement.Stage), AllowedDestinations: dest, Handling: networkplan.HandlingMode(s.Enforcement.Handling)}}
			for _, r := range s.Resources {
				uid, e := resolve(r.ResourceRef)
				if e != nil {
					return in, e
				}
				sel, e := endpointSelection(r.EndpointNames, r.EndpointSelector)
				if e != nil {
					return in, e
				}
				bind.Resources = append(bind.Resources, networkplan.ResourceSelection{ResourceName: uid, Endpoints: sel})
			}
			for _, a := range s.Agents {
				if (a.AgentRef != nil) == (a.AgentSelector != nil) {
					return in, fmt.Errorf("Agent selection requires exactly one reference or selector")
				}
				out := networkplan.AgentSelection{}
				if a.AgentRef != nil {
					if a.AgentRef.Name == "" {
						return in, fmt.Errorf("Agent reference name is empty")
					}
					out.AgentRef = a.AgentRef.Name
				} else {
					sel, e := labelSelector(a.AgentSelector)
					if e != nil {
						return in, e
					}
					out.Selector = sel
				}
				bind.Agents = append(bind.Agents, out)
			}
			in.Bindings = append(in.Bindings, bind)
		}
	}
	return in, nil
}
func endpointSelection(names []string, s *selector) (core.EndpointSelection, error) {
	if names != nil && s != nil {
		return core.EndpointSelection{}, fmt.Errorf("Endpoint restrictions are mutually exclusive")
	}
	out := core.EndpointSelection{}
	if names != nil {
		out.Mode = core.SelectionNames
		out.Names = names
	}
	if s != nil {
		v, e := labelSelector(s)
		if e != nil {
			return out, e
		}
		out.Mode = core.SelectionLabels
		out.Selector = v
	}
	return core.NormalizeSelection(out)
}
func labelSelector(s *selector) (*core.LabelSelector, error) {
	out := core.LabelSelector{MatchLabels: s.MatchLabels}
	for _, r := range s.MatchExpressions {
		out.MatchExpressions = append(out.MatchExpressions, core.Requirement{Key: r.Key, Operator: r.Operator, Values: r.Values})
	}
	n, e := core.NormalizeSelector(out)
	if e != nil {
		return nil, e
	}
	return &n, nil
}
func parseValidity(v validity) (core.Validity, error) {
	out := core.Validity{}
	switch v.Mode {
	case "bounded":
		out.Mode = core.LifetimeBounded
	case "permanent":
		out.Mode = core.LifetimePermanent
	default:
		return out, fmt.Errorf("unsupported validity mode %q", v.Mode)
	}
	for _, field := range []struct {
		text   *string
		target *time.Time
	}{{v.NotBefore, &out.NotBefore}, {v.ExpiresAt, &out.ExpiresAt}} {
		if field.text == nil {
			continue
		}
		at, e := time.Parse(time.RFC3339Nano, *field.text)
		if e != nil || at.IsZero() {
			return out, fmt.Errorf("validity requires nonzero absolute RFC3339 times")
		}
		*field.target = at
	}
	if out.Mode == core.LifetimePermanent && v.ExpiresAt != nil {
		return out, fmt.Errorf("permanent validity forbids expiresAt")
	}
	return core.NormalizeValidity(out)
}
