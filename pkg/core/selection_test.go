// SPDX-License-Identifier: Apache-2.0
package core

import (
	"reflect"
	"testing"
)

func entries() []Endpoint {
	return []Endpoint{{Name: "ssh", Labels: map[string]string{"kind": "git"}}, {Name: "https", Labels: map[string]string{"kind": "web"}}}
}
func TestEndpointSelection(t *testing.T) {
	es := entries()
	for _, tc := range []struct {
		s    EndpointSelection
		want []string
	}{
		{EndpointSelection{}, []string{"https", "ssh"}}, {EndpointSelection{Mode: SelectionAll}, []string{"https", "ssh"}},
		{EndpointSelection{Mode: SelectionNames, Names: []string{"ssh", "ssh"}}, []string{"ssh"}},
		{EndpointSelection{Mode: SelectionLabels, Selector: &LabelSelector{MatchLabels: map[string]string{"kind": "web"}}}, []string{"https"}},
		{EndpointSelection{Mode: SelectionLabels, Selector: &LabelSelector{MatchLabels: map[string]string{"kind": "missing"}}}, nil},
	} {
		got, err := SelectEndpointNames(es, tc.s)
		if err != nil || !reflect.DeepEqual(got, tc.want) {
			t.Errorf("got %v %v want %v", got, err, tc.want)
		}
	}
	es = append(es, Endpoint{Name: "admin", Labels: map[string]string{"kind": "web"}})
	for _, tc := range []struct {
		s    EndpointSelection
		want []string
	}{
		{EndpointSelection{}, []string{"admin", "https", "ssh"}},
		{EndpointSelection{Mode: SelectionNames, Names: []string{"https"}}, []string{"https"}},
		{EndpointSelection{Mode: SelectionLabels, Selector: &LabelSelector{MatchLabels: map[string]string{"kind": "web"}}}, []string{"admin", "https"}},
	} {
		got, err := SelectEndpointNames(es, tc.s)
		if err != nil || !reflect.DeepEqual(got, tc.want) {
			t.Errorf("got %v %v want %v", got, err, tc.want)
		}
	}
	if got, err := SelectEndpointNames(nil, EndpointSelection{}); err != nil || got != nil {
		t.Fatalf("empty catalog: %v %v", got, err)
	}
	// Normalization must copy caller data even before Engine exists.
	in := EndpointSelection{Mode: SelectionLabels, Selector: &LabelSelector{MatchLabels: map[string]string{"kind": "web"}, MatchExpressions: []Requirement{{Key: "kind", Operator: SelectorIn, Values: []string{"web"}}}}}
	out, err := NormalizeSelection(in)
	if err != nil {
		t.Fatal(err)
	}
	in.Selector.MatchLabels["kind"] = "git"
	in.Selector.MatchExpressions[0].Values[0] = "git"
	if !out.Selector.Matches(map[string]string{"kind": "web"}) {
		t.Fatal("selector aliases input")
	}
	ep := es[0]
	norm, err := NormalizeEndpoint(ep)
	if err != nil || norm.Type != "network" {
		t.Fatalf("default: %v %v", norm, err)
	}
	ep.Labels["kind"] = "changed"
	if norm.Labels["kind"] != "git" {
		t.Fatal("endpoint aliases labels")
	}
}
func TestSelectionRejectsInvalidInputs(t *testing.T) {
	for _, s := range []EndpointSelection{
		{Names: []string{}}, {Selector: &LabelSelector{}}, {Mode: SelectionAll, Names: []string{"ssh"}},
		{Mode: SelectionNames}, {Mode: SelectionNames, Names: []string{}}, {Mode: SelectionNames, Names: []string{""}},
		{Mode: SelectionNames, Names: []string{"missing"}}, {Mode: SelectionNames, Names: []string{"ssh"}, Selector: &LabelSelector{MatchLabels: map[string]string{"x": "y"}}},
		{Mode: SelectionLabels}, {Mode: SelectionLabels, Selector: &LabelSelector{}}, {Mode: SelectionLabels, Selector: &LabelSelector{MatchLabels: map[string]string{"x": "y"}}, Names: []string{}}, {Mode: 99},
	} {
		if _, err := SelectEndpointNames(entries(), s); err == nil {
			t.Errorf("accepted %+v", s)
		}
	}
	for _, es := range [][]Endpoint{{{Name: "x"}, {Name: "x"}}, {{}}, {{Name: "x", Type: "http"}}} {
		if _, err := SelectEndpointNames(es, EndpointSelection{}); err == nil {
			t.Errorf("accepted %+v", es)
		}
	}
}
