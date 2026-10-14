// SPDX-License-Identifier: Apache-2.0
package core

import (
	"fmt"
	"maps"
	"slices"
)

type Endpoint struct {
	Name, Type string
	Labels     map[string]string
}
type SelectionMode uint8

const (
	SelectionOmitted SelectionMode = iota
	SelectionAll
	SelectionNames
	SelectionLabels
)

type EndpointSelection struct {
	Mode     SelectionMode
	Names    []string
	Selector *LabelSelector
}

func NormalizeEndpoint(e Endpoint) (Endpoint, error) {
	if e.Name == "" {
		return Endpoint{}, fmt.Errorf("endpoint name must not be empty")
	}
	if e.Type == "" {
		e.Type = "network"
	}
	if e.Type != "network" {
		return Endpoint{}, fmt.Errorf("unsupported endpoint type %q", e.Type)
	}
	e.Labels = maps.Clone(e.Labels)
	return e, nil
}

// NormalizeSelection defaults only a completely omitted selection to whole-Resource.
func NormalizeSelection(s EndpointSelection) (EndpointSelection, error) {
	switch s.Mode {
	case SelectionOmitted, SelectionAll:
		if s.Names != nil || s.Selector != nil {
			return EndpointSelection{}, fmt.Errorf("whole selection forbids restrictions")
		}
		return EndpointSelection{Mode: SelectionAll}, nil
	case SelectionNames:
		if len(s.Names) == 0 || s.Selector != nil {
			return EndpointSelection{}, fmt.Errorf("named selection requires names only")
		}
		for _, n := range s.Names {
			if n == "" {
				return EndpointSelection{}, fmt.Errorf("endpoint name must not be empty")
			}
		}
		s.Names = slices.Clone(s.Names)
		slices.Sort(s.Names)
		s.Names = slices.Compact(s.Names)
		return s, nil
	case SelectionLabels:
		if s.Selector == nil || s.Names != nil {
			return EndpointSelection{}, fmt.Errorf("label selection requires selector only")
		}
		selector, err := NormalizeSelector(*s.Selector)
		if err != nil {
			return EndpointSelection{}, err
		}
		return EndpointSelection{Mode: SelectionLabels, Selector: &selector}, nil
	default:
		return EndpointSelection{}, fmt.Errorf("unsupported selection mode %d", s.Mode)
	}
}

// SelectEndpointNames resolves against this catalog; an empty result is not a wildcard.
func SelectEndpointNames(es []Endpoint, selection EndpointSelection) ([]string, error) {
	s, err := NormalizeSelection(selection)
	if err != nil {
		return nil, err
	}
	catalog := make(map[string]Endpoint, len(es))
	for _, e := range es {
		n, err := NormalizeEndpoint(e)
		if err != nil {
			return nil, err
		}
		if _, ok := catalog[n.Name]; ok {
			return nil, fmt.Errorf("duplicate endpoint %q", n.Name)
		}
		catalog[n.Name] = n
	}
	if s.Mode == SelectionNames {
		for _, name := range s.Names {
			if _, ok := catalog[name]; !ok {
				return nil, fmt.Errorf("unknown endpoint %q", name)
			}
		}
		return s.Names, nil
	}
	var names []string
	for name, e := range catalog {
		if s.Mode == SelectionAll || s.Selector.Matches(e.Labels) {
			names = append(names, name)
		}
	}
	slices.Sort(names)
	return names, nil
}
