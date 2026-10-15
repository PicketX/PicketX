// SPDX-License-Identifier: Apache-2.0
package networkplan

import (
	"cmp"
	"fmt"
	"maps"
	"slices"
	"time"

	"github.com/PicketX/PicketX/pkg/core"
)

type contextKey struct{ agent, id string }

// Plan is immutable after Compile/Replace. Reads and independently mutable
// returned collections may be used concurrently. Desired intent is not proof of apply.
type Plan struct {
	authority   string
	agents      map[string]Agent
	view        core.NetworkView
	entries     map[EntryRef]core.ResolvedEntry
	networks    map[EntryRef]NetworkDefinition
	contexts    map[contextKey]ExecutionContext
	targets     map[TargetKey]TargetInventory
	diagnostics []Diagnostic
}

// Compile validates a complete authoritative candidate for trusted initial
// bootstrap. Use Replace with the previous accepted plan for later candidates.
// Neither path adopts unknown installed firewall state.
func Compile(input Input) (*Plan, error) {
	if input.AuthorityID == "" {
		return nil, fmt.Errorf("authority identity must not be empty")
	}
	if len(input.Requirements) > 0 {
		return nil, fmt.Errorf("unsupported requirements %q", input.Requirements)
	}
	engine, err := core.Compile(input.Logical)
	if err != nil {
		return nil, err
	}
	view, err := engine.NetworkView()
	if err != nil {
		return nil, err
	}
	p := &Plan{authority: input.AuthorityID, agents: make(map[string]Agent), view: view, entries: make(map[EntryRef]core.ResolvedEntry), networks: make(map[EntryRef]NetworkDefinition), contexts: make(map[contextKey]ExecutionContext), targets: make(map[TargetKey]TargetInventory)}
	resources := make(map[string]core.ResolvedResource)
	for _, r := range view.Resources {
		resources[r.Name] = r
		for _, e := range r.Entries {
			p.entries[EntryRef{r.Name, e.Endpoint.Name}] = e
		}
	}
	for _, raw := range input.Networks {
		n, err := NormalizeNetwork(raw)
		if err != nil {
			return nil, fmt.Errorf("network %s/%s: %w", raw.Entry.ResourceName, raw.Entry.EndpointName, err)
		}
		if _, ok := p.entries[n.Entry]; !ok {
			return nil, fmt.Errorf("orphan network %s/%s", n.Entry.ResourceName, n.Entry.EndpointName)
		}
		if _, ok := p.networks[n.Entry]; ok {
			return nil, fmt.Errorf("duplicate network %s/%s", n.Entry.ResourceName, n.Entry.EndpointName)
		}
		p.networks[n.Entry] = n
	}
	for _, r := range view.Resources {
		for _, e := range r.Entries {
			if _, ok := p.networks[EntryRef{r.Name, e.Endpoint.Name}]; !ok {
				return nil, fmt.Errorf("missing network %s/%s", r.Name, e.Endpoint.Name)
			}
		}
	}
	for _, a := range input.Agents {
		if a.Name == "" {
			return nil, fmt.Errorf("Agent name must not be empty")
		}
		if _, ok := p.agents[a.Name]; ok {
			return nil, fmt.Errorf("duplicate Agent %q", a.Name)
		}
		caps, err := NormalizeCapabilities(a.Capabilities)
		if err != nil {
			return nil, fmt.Errorf("Agent %q: %w", a.Name, err)
		}
		a.Labels = maps.Clone(a.Labels)
		a.Capabilities = caps
		p.agents[a.Name] = a
	}
	profiles := make(map[contextKey]ExecutionContext)
	for _, raw := range input.Contexts {
		c, err := NormalizeExecutionContext(raw)
		if err != nil {
			return nil, fmt.Errorf("context Agent %q ID %q: %w", raw.AgentRef, raw.ID, err)
		}
		if _, ok := p.agents[c.AgentRef]; !ok {
			return nil, fmt.Errorf("context %q has unknown Agent %q", c.ID, c.AgentRef)
		}
		id, profile := contextKey{c.AgentRef, c.ID}, contextKey{c.AgentRef, c.ProfileRef}
		if _, ok := p.contexts[id]; ok {
			return nil, fmt.Errorf("duplicate context ID %q on Agent %q", c.ID, c.AgentRef)
		}
		if _, ok := profiles[profile]; ok {
			return nil, fmt.Errorf("duplicate context profile %q on Agent %q", c.ProfileRef, c.AgentRef)
		}
		p.contexts[id] = c
		profiles[profile] = c
	}
	seen := make(map[string]struct{})
	for _, b := range input.Bindings {
		if b.ID == "" {
			return nil, fmt.Errorf("Binding ID must not be empty")
		}
		if _, ok := seen[b.ID]; ok {
			return nil, fmt.Errorf("duplicate Binding %q", b.ID)
		}
		seen[b.ID] = struct{}{}
		request, err := NormalizeContextRequest(b.Context)
		if err != nil {
			return nil, fmt.Errorf("Binding %q context: %w", b.ID, err)
		}
		entries, diagnostics, err := selectResources(b, resources)
		if err != nil {
			return nil, fmt.Errorf("Binding %q: %w", b.ID, err)
		}
		agents, err := selectAgents(b, p.agents)
		if err != nil {
			return nil, fmt.Errorf("Binding %q: %w", b.ID, err)
		}
		p.diagnostics = append(p.diagnostics, diagnostics...)
		if len(agents) == 0 {
			p.diagnostics = append(p.diagnostics, Diagnostic{Code: "no-agents-matched", BindingID: b.ID})
		}
		for _, agent := range agents {
			facts, ok := profiles[contextKey{agent, request.ProfileRef}]
			if !ok {
				return nil, fmt.Errorf("Binding %q Agent %q missing context profile %q", b.ID, agent, request.ProfileRef)
			}
			context, err := ResolveContext(request, facts, p.agents[agent].Capabilities)
			if err != nil {
				return nil, fmt.Errorf("Binding %q Agent %q: %w", b.ID, agent, err)
			}
			for _, entry := range entries {
				if err := p.place(b.ID, agent, entry, context); err != nil {
					return nil, fmt.Errorf("Binding %q Agent %q entry %s/%s: %w", b.ID, agent, entry.ResourceName, entry.EndpointName, err)
				}
			}
		}
	}
	slices.SortFunc(p.diagnostics, compareDiagnostic)
	p.diagnostics = slices.Compact(p.diagnostics)
	if err := p.validateConflicts(); err != nil {
		return nil, err
	}
	return p, nil
}
func (p *Plan) place(binding, agent string, entry EntryRef, context ExecutionContext) error {
	n := p.networks[entry]
	destinations := slices.Clone(context.Destinations)
	if n.Addresses != nil {
		var err error
		destinations, err = IntersectDestinations(n.Addresses, context.Destinations)
		if err != nil {
			return err
		}
	}
	if len(destinations) == 0 {
		return fmt.Errorf("empty selected destination intersection")
	}
	required := CapabilityProfile{Classes: []TrafficClass{context.Class}, Protocols: []Protocol{n.Protocol}, Handling: []HandlingMode{context.Handling}, OriginalDestination: true}
	var matches []Match
	for _, d := range destinations {
		family := IPv6
		if d.Addr().Is4() {
			family = IPv4
		}
		if !slices.Contains(p.agents[agent].Capabilities.Families, family) || !slices.Contains(p.agents[agent].Capabilities.Protocols, n.Protocol) {
			return fmt.Errorf("capabilities do not support protocol %q family %d", n.Protocol, family)
		}
		required.Families = append(required.Families, family)
		matches = append(matches, Match{Destination: d, Protocol: n.Protocol, Port: n.Port})
	}
	required.Families = sortedUnique(required.Families)
	slices.SortFunc(matches, compareMatch)
	key := TargetKey{AgentRef: agent, ContextID: context.ID, Entry: entry}
	if target, ok := p.targets[key]; ok {
		target.Protection.BindingIDs = sortedUnique(append(target.Protection.BindingIDs, binding))
		p.targets[key] = target
		return nil
	}
	protection := Protection{Key: key, Context: context, BindingIDs: []string{binding}, Matches: matches, Required: required}
	relevant := func(cs []core.ResolvedContribution) []core.ResolvedContribution {
		var out []core.ResolvedContribution
		for _, c := range cs {
			for _, d := range destinations {
				if c.Source.Addr().BitLen() == d.Addr().BitLen() {
					out = append(out, c)
					break
				}
			}
		}
		return out
	}
	resolved := p.entries[entry]
	p.targets[key] = TargetInventory{Protection: protection, Grants: relevant(resolved.Grants), Denies: relevant(resolved.Denies)}
	return nil
}
func (p *Plan) valid() bool { return p != nil && p.agents != nil }
func (p *Plan) targetKeys() []TargetKey {
	var keys []TargetKey
	for k := range p.targets {
		keys = append(keys, k)
	}
	slices.SortFunc(keys, compareTarget)
	return keys
}
func (p *Plan) AgentRefs() ([]string, error) {
	if !p.valid() {
		return nil, fmt.Errorf("plan is not compiled")
	}
	var out []string
	for k := range p.targets {
		out = append(out, k.AgentRef)
	}
	return sortedUnique(out), nil
}
func (p *Plan) Diagnostics() ([]Diagnostic, error) {
	if !p.valid() {
		return nil, fmt.Errorf("plan is not compiled")
	}
	return slices.Clone(p.diagnostics), nil
}
func cloneInventory(t TargetInventory) TargetInventory {
	t.Protection.BindingIDs = slices.Clone(t.Protection.BindingIDs)
	t.Protection.Matches = slices.Clone(t.Protection.Matches)
	t.Protection.Context.Destinations = slices.Clone(t.Protection.Context.Destinations)
	c := t.Protection.Required
	t.Protection.Required = CapabilityProfile{Classes: slices.Clone(c.Classes), Protocols: slices.Clone(c.Protocols), Families: slices.Clone(c.Families), Handling: slices.Clone(c.Handling), OriginalDestination: c.OriginalDestination}
	t.Grants = slices.Clone(t.Grants)
	t.Denies = slices.Clone(t.Denies)
	return t
}

// Inventory returns complete desired membership, including future/expired/revoked
// contributions. It does not establish what was installed or observed.
func (p *Plan) Inventory(agentRef string) ([]TargetInventory, error) {
	if !p.valid() {
		return nil, fmt.Errorf("plan is not compiled")
	}
	if _, ok := p.agents[agentRef]; !ok {
		return nil, fmt.Errorf("unknown Agent %q", agentRef)
	}
	var out []TargetInventory
	for _, key := range p.targetKeys() {
		if key.AgentRef == agentRef {
			out = append(out, cloneInventory(p.targets[key]))
		}
	}
	return out, nil
}

// Projection materializes active source domains at an explicit instant. Allow
// and Deny remain separate; an adapter evaluating a flow must apply deny first.
// NextChange requests reevaluation; this library does not run a scheduler.
func (p *Plan) Projection(agentRef string, at time.Time) (Projection, error) {
	if at.IsZero() {
		return Projection{}, fmt.Errorf("evaluation time must not be zero")
	}
	targets, err := p.Inventory(agentRef)
	if err != nil {
		return Projection{}, err
	}
	projection := Projection{AuthorityID: p.authority, AgentRef: agentRef, At: at.Round(0).UTC()}
	for _, t := range targets {
		projection.Protected = append(projection.Protected, t.Protection)
		for _, domain := range []struct {
			members []core.ResolvedContribution
			deny    bool
		}{{t.Grants, false}, {t.Denies, true}} {
			for _, c := range domain.members {
				if c.Revoked {
					continue
				}
				next := c.Validity.NextBoundary(at)
				if !next.IsZero() && (projection.NextChange.IsZero() || next.Before(projection.NextChange)) {
					projection.NextChange = next
				}
				if !c.Validity.ActiveAt(at) {
					continue
				}
				for _, m := range t.Protection.Matches {
					if c.Source.Addr().BitLen() == m.Destination.Addr().BitLen() {
						membership := Membership{Target: t.Protection.Key, Match: m, Contribution: c}
						if domain.deny {
							projection.Deny = append(projection.Deny, membership)
						} else {
							projection.Allow = append(projection.Allow, membership)
						}
					}
				}
			}
		}
	}
	slices.SortFunc(projection.Allow, compareMembership)
	slices.SortFunc(projection.Deny, compareMembership)
	return projection, nil
}
func compareMembership(a, b Membership) int {
	if c := compareTarget(a.Target, b.Target); c != 0 {
		return c
	}
	if c := compareMatch(a.Match, b.Match); c != 0 {
		return c
	}
	if c := comparePrefix(a.Contribution.Source, b.Contribution.Source); c != 0 {
		return c
	}
	return cmp.Compare(a.Contribution.ID, b.Contribution.ID)
}
