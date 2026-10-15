// SPDX-License-Identifier: Apache-2.0
package networkplan

import (
	"cmp"
	"fmt"
	"github.com/PicketX/PicketX/pkg/core"
	"slices"
)

func selectResources(b Binding, resources map[string]core.ResolvedResource) ([]EntryRef, []Diagnostic, error) {
	if len(b.Resources) == 0 {
		return nil, nil, fmt.Errorf("Resource selection list must not be empty")
	}
	selected := make(map[EntryRef]struct{})
	var diagnostics []Diagnostic
	for _, item := range b.Resources {
		if item.ResourceSelector != nil {
			return nil, nil, fmt.Errorf("broad Resource selectors are unsupported")
		}
		r, ok := resources[item.ResourceName]
		if !ok {
			return nil, nil, fmt.Errorf("unknown Resource %q", item.ResourceName)
		}
		endpoints := make([]core.Endpoint, 0, len(r.Entries))
		for _, e := range r.Entries {
			endpoints = append(endpoints, e.Endpoint)
		}
		names, err := core.SelectEndpointNames(endpoints, item.Endpoints)
		if err != nil {
			return nil, nil, fmt.Errorf("Resource %q: %w", r.Name, err)
		}
		if len(names) == 0 {
			diagnostics = append(diagnostics, Diagnostic{Code: "no-entries-matched", BindingID: b.ID, ResourceName: r.Name})
		}
		for _, n := range names {
			selected[EntryRef{r.Name, n}] = struct{}{}
		}
	}
	var out []EntryRef
	for e := range selected {
		out = append(out, e)
	}
	slices.SortFunc(out, compareEntry)
	return out, diagnostics, nil
}
func selectAgents(b Binding, agents map[string]Agent) ([]string, error) {
	if len(b.Agents) == 0 {
		return nil, fmt.Errorf("Agent selection list must not be empty")
	}
	selected := make(map[string]struct{})
	for _, item := range b.Agents {
		if (item.AgentRef != "") == (item.Selector != nil) {
			return nil, fmt.Errorf("Agent selection requires exactly one reference or nonempty selector")
		}
		if item.AgentRef != "" {
			if _, ok := agents[item.AgentRef]; !ok {
				return nil, fmt.Errorf("unknown Agent %q", item.AgentRef)
			}
			selected[item.AgentRef] = struct{}{}
			continue
		}
		selector, err := core.NormalizeSelector(*item.Selector)
		if err != nil {
			return nil, err
		}
		for name, a := range agents {
			if selector.Matches(a.Labels) {
				selected[name] = struct{}{}
			}
		}
	}
	var out []string
	for name := range selected {
		out = append(out, name)
	}
	slices.Sort(out)
	return out, nil
}
func compareEntry(a, b EntryRef) int {
	if c := cmp.Compare(a.ResourceName, b.ResourceName); c != 0 {
		return c
	}
	return cmp.Compare(a.EndpointName, b.EndpointName)
}
func compareTarget(a, b TargetKey) int {
	if c := cmp.Compare(a.AgentRef, b.AgentRef); c != 0 {
		return c
	}
	if c := cmp.Compare(a.ContextID, b.ContextID); c != 0 {
		return c
	}
	return compareEntry(a.Entry, b.Entry)
}
func compareMatch(a, b Match) int {
	if c := cmp.Compare(a.Protocol, b.Protocol); c != 0 {
		return c
	}
	if c := cmp.Compare(a.Port, b.Port); c != 0 {
		return c
	}
	return comparePrefix(a.Destination, b.Destination)
}
func compareDiagnostic(a, b Diagnostic) int {
	for _, pair := range [][2]string{{a.Code, b.Code}, {a.BindingID, b.BindingID}, {a.ResourceName, b.ResourceName}, {a.AgentRef, b.AgentRef}, {a.EndpointName, b.EndpointName}} {
		if c := cmp.Compare(pair[0], pair[1]); c != 0 {
			return c
		}
	}
	return 0
}
