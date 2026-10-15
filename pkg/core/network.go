// SPDX-License-Identifier: Apache-2.0
package core

import (
	"cmp"
	"fmt"
	"maps"
	"slices"
)

type entryKey struct{ resource, endpoint string }
type compiledContribution struct {
	grant         Contribution
	revoked, deny bool
}

// Engine is an immutable logical network snapshot. Construct it with Compile.
// Decisions are safe for concurrent reads, including mutation of independent results.
type Engine struct {
	entries   map[entryKey][]compiledContribution
	resources []Resource
}

// NetworkView returns owned, deterministic resolved membership without filtering
// by time or revocation. It includes known Resources that have no entries.
func (e *Engine) NetworkView() (NetworkView, error) {
	if e == nil || e.entries == nil {
		return NetworkView{}, fmt.Errorf("engine is not compiled")
	}
	var view NetworkView
	for _, r := range e.resources {
		out := ResolvedResource{Name: r.Name, Labels: maps.Clone(r.Labels)}
		for _, ep := range r.Endpoints {
			entry := ResolvedEntry{Endpoint: ep}
			entry.Endpoint.Labels = maps.Clone(ep.Labels)
			for _, c := range e.entries[entryKey{r.Name, ep.Name}] {
				v := ResolvedContribution{ID: c.grant.GrantID, Source: c.grant.Source, Validity: c.grant.Validity, Revoked: c.revoked}
				if c.deny {
					entry.Denies = append(entry.Denies, v)
				} else {
					entry.Grants = append(entry.Grants, v)
				}
			}
			byID := func(a, b ResolvedContribution) int { return cmp.Compare(a.ID, b.ID) }
			slices.SortFunc(entry.Grants, byID)
			slices.SortFunc(entry.Denies, byID)
			out.Entries = append(out.Entries, entry)
		}
		slices.SortFunc(out.Entries, func(a, b ResolvedEntry) int { return cmp.Compare(a.Endpoint.Name, b.Endpoint.Name) })
		view.Resources = append(view.Resources, out)
	}
	slices.SortFunc(view.Resources, func(a, b ResolvedResource) int { return cmp.Compare(a.Name, b.Name) })
	return view, nil
}

// Compile validates the complete candidate before returning any replacement state.
// It retains only owned, resolved entry membership and immutable contribution values.
func Compile(s Snapshot) (*Engine, error) {
	e := &Engine{entries: make(map[entryKey][]compiledContribution)}
	catalog := make(map[string][]Endpoint, len(s.Resources))
	for _, r := range s.Resources {
		if r.Name == "" {
			return nil, fmt.Errorf("resource name must not be empty")
		}
		if _, ok := catalog[r.Name]; ok {
			return nil, fmt.Errorf("duplicate resource %q", r.Name)
		}
		endpoints := make([]Endpoint, 0, len(r.Endpoints))
		for _, ep := range r.Endpoints {
			n, err := NormalizeEndpoint(ep)
			if err != nil {
				return nil, fmt.Errorf("resource %q: %w", r.Name, err)
			}
			key := entryKey{r.Name, n.Name}
			if _, ok := e.entries[key]; ok {
				return nil, fmt.Errorf("duplicate endpoint %q in %q", n.Name, r.Name)
			}
			e.entries[key] = nil
			endpoints = append(endpoints, n)
		}
		catalog[r.Name] = endpoints
		e.resources = append(e.resources, Resource{Name: r.Name, Labels: maps.Clone(r.Labels), Endpoints: endpoints})
	}
	for _, deny := range []bool{false, true} {
		seen := make(map[string]struct{})
		count := len(s.Grants)
		kind := "grant"
		if deny {
			count = len(s.Denies)
			kind = "deny"
		}
		for i := 0; i < count; i++ {
			var g Grant
			if deny {
				g = Grant(s.Denies[i])
			} else {
				g = s.Grants[i]
			}
			if _, ok := seen[g.ID]; ok {
				return nil, fmt.Errorf("duplicate %s ID %q", kind, g.ID)
			}
			seen[g.ID] = struct{}{}
			c, keys, err := compileContribution(g, catalog)
			if err != nil {
				return nil, fmt.Errorf("%s %q: %w", kind, g.ID, err)
			}
			c.deny = deny
			for key := range keys {
				e.entries[key] = append(e.entries[key], c)
			}
		}
	}
	return e, nil
}

func compileContribution(g Grant, catalog map[string][]Endpoint) (compiledContribution, map[entryKey]struct{}, error) {
	if g.ID == "" {
		return compiledContribution{}, nil, fmt.Errorf("contribution ID must not be empty")
	}
	if err := ValidateSource(g.Source); err != nil {
		return compiledContribution{}, nil, err
	}
	v, err := NormalizeValidity(g.Validity)
	if err != nil {
		return compiledContribution{}, nil, err
	}
	if g.SubjectType != "" && g.SubjectType != "source" {
		return compiledContribution{}, nil, fmt.Errorf("unsupported subject %q", g.SubjectType)
	}
	if len(g.Conditions) > 0 {
		return compiledContribution{}, nil, fmt.Errorf("conditions are unsupported")
	}
	if g.Renewal != "" && g.Renewal != "disabled" {
		return compiledContribution{}, nil, fmt.Errorf("unsupported renewal %q", g.Renewal)
	}
	if len(g.Scopes) == 0 {
		return compiledContribution{}, nil, fmt.Errorf("scopes must not be empty")
	}
	keys := make(map[entryKey]struct{})
	for _, scope := range g.Scopes {
		es, ok := catalog[scope.ResourceName]
		if !ok {
			return compiledContribution{}, nil, fmt.Errorf("unknown resource %q", scope.ResourceName)
		}
		if scope.Actions != nil {
			if len(scope.Actions) == 0 {
				return compiledContribution{}, nil, fmt.Errorf("actions must not be empty")
			}
			for _, action := range scope.Actions {
				if action != "connect" {
					return compiledContribution{}, nil, fmt.Errorf("unsupported action %q", action)
				}
			}
		}
		names, err := SelectEndpointNames(es, scope.Endpoints)
		if err != nil {
			return compiledContribution{}, nil, fmt.Errorf("resource %q selection: %w", scope.ResourceName, err)
		}
		for _, name := range names {
			keys[entryKey{scope.ResourceName, name}] = struct{}{}
		}
	}
	return compiledContribution{grant: Contribution{GrantID: g.ID, Source: g.Source, Validity: v}, revoked: g.Revoked}, keys, nil
}

// Decide evaluates a trusted logical entry at the supplied instant. A logical allow
// does not establish the physical packet destination or confirm Agent enforcement.
func (e *Engine) Decide(q Query) (Decision, error) {
	if e == nil || e.entries == nil {
		return Decision{}, fmt.Errorf("engine is not compiled")
	}
	if err := validateAddress(q.Source); err != nil {
		return Decision{}, err
	}
	if q.At.IsZero() {
		return Decision{}, fmt.Errorf("evaluation time must not be zero")
	}
	if q.Action != "" && q.Action != "connect" {
		return Decision{}, fmt.Errorf("unsupported action %q", q.Action)
	}
	contributions, ok := e.entries[entryKey{q.ResourceName, q.EndpointName}]
	if !ok {
		return Decision{}, fmt.Errorf("unknown logical entry %q/%q", q.ResourceName, q.EndpointName)
	}
	d := Decision{Reason: ReasonNoActiveGrant}
	for _, c := range contributions {
		if c.revoked || !c.grant.Source.Contains(q.Source) {
			continue
		}
		boundary := c.grant.Validity.NextBoundary(q.At)
		if !boundary.IsZero() && (d.NextChange.IsZero() || boundary.Before(d.NextChange)) {
			d.NextChange = boundary
		}
		if !c.grant.Validity.ActiveAt(q.At) {
			continue
		}
		if c.deny {
			d.DenyIDs = append(d.DenyIDs, c.grant.GrantID)
		} else {
			d.GrantIDs = append(d.GrantIDs, c.grant.GrantID)
			d.Contributions = append(d.Contributions, c.grant)
		}
	}
	slices.Sort(d.GrantIDs)
	slices.Sort(d.DenyIDs)
	slices.SortFunc(d.Contributions, func(a, b Contribution) int { return cmp.Compare(a.GrantID, b.GrantID) })
	if len(d.DenyIDs) > 0 {
		d.Reason = ReasonExplicitDeny
	} else if len(d.GrantIDs) > 0 {
		d.Allowed = true
		d.Reason = ReasonAllowed
	}
	return d, nil
}
