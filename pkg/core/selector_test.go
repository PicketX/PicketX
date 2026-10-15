// SPDX-License-Identifier: Apache-2.0
package core

import "testing"

// Catches treating missing keys as empty equality and treating NotIn as deny.
func TestSelectorOperators(t *testing.T) {
	for _, tc := range []struct {
		name   string
		s      LabelSelector
		labels map[string]string
		want   bool
	}{
		{"notin missing", LabelSelector{MatchExpressions: []Requirement{{Key: "access", Operator: SelectorNotIn, Values: []string{"blocked"}}}}, nil, true},
		{"notin exists", LabelSelector{MatchExpressions: []Requirement{{Key: "access", Operator: SelectorNotIn, Values: []string{"blocked"}}, {Key: "access", Operator: SelectorExists}}}, nil, false},
		{"in alternatives", LabelSelector{MatchLabels: map[string]string{"env": "dev"}, MatchExpressions: []Requirement{{Key: "entry", Operator: SelectorIn, Values: []string{"https", "ssh"}}}}, map[string]string{"env": "dev", "entry": "ssh"}, true},
		{"and", LabelSelector{MatchLabels: map[string]string{"env": "dev"}, MatchExpressions: []Requirement{{Key: "entry", Operator: SelectorIn, Values: []string{"https", "ssh"}}}}, map[string]string{"env": "prod", "entry": "ssh"}, false},
		{"doesnotexist", LabelSelector{MatchExpressions: []Requirement{{Key: "access", Operator: SelectorDoesNotExist}}}, nil, true},
		{"doesnotexist present", LabelSelector{MatchExpressions: []Requirement{{Key: "access", Operator: SelectorDoesNotExist}}}, map[string]string{"access": ""}, false},
		{"empty existing", LabelSelector{MatchLabels: map[string]string{"access": ""}}, map[string]string{"access": ""}, true},
		{"empty missing", LabelSelector{MatchLabels: map[string]string{"access": ""}}, nil, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s, err := NormalizeSelector(tc.s)
			if err != nil {
				t.Fatal(err)
			}
			if got := s.Matches(tc.labels); got != tc.want {
				t.Fatalf("got %v want %v", got, tc.want)
			}
		})
	}
}
func TestSelectorRejectsInvalidValues(t *testing.T) {
	for _, s := range []LabelSelector{
		{}, {MatchLabels: map[string]string{}}, {MatchExpressions: []Requirement{}}, {MatchLabels: map[string]string{"": "x"}},
		{MatchExpressions: []Requirement{{Key: "x", Operator: "unknown"}}}, {MatchExpressions: []Requirement{{Operator: SelectorExists}}},
		{MatchExpressions: []Requirement{{Key: "x", Operator: SelectorIn}}}, {MatchExpressions: []Requirement{{Key: "x", Operator: SelectorNotIn, Values: []string{}}}},
		{MatchExpressions: []Requirement{{Key: "x", Operator: SelectorExists, Values: []string{}}}}, {MatchExpressions: []Requirement{{Key: "x", Operator: SelectorDoesNotExist, Values: []string{"a"}}}},
	} {
		if _, err := NormalizeSelector(s); err == nil {
			t.Errorf("accepted %+v", s)
		}
		if s.Matches(nil) {
			t.Errorf("invalid selector matched %+v", s)
		}
	}
}
