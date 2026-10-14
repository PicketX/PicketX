// SPDX-License-Identifier: Apache-2.0
package core

import (
	"fmt"
	"maps"
	"slices"
)

type SelectorOperator string

const (
	SelectorIn           SelectorOperator = "In"
	SelectorNotIn        SelectorOperator = "NotIn"
	SelectorExists       SelectorOperator = "Exists"
	SelectorDoesNotExist SelectorOperator = "DoesNotExist"
)

type Requirement struct {
	Key      string
	Operator SelectorOperator
	Values   []string
}
type LabelSelector struct {
	MatchLabels      map[string]string
	MatchExpressions []Requirement
}

// NormalizeSelector validates presence and returns an owned copy.
func NormalizeSelector(s LabelSelector) (LabelSelector, error) {
	if err := validateSelector(s); err != nil {
		return LabelSelector{}, err
	}
	out := LabelSelector{MatchLabels: maps.Clone(s.MatchLabels), MatchExpressions: slices.Clone(s.MatchExpressions)}
	for i := range out.MatchExpressions {
		out.MatchExpressions[i].Values = slices.Clone(s.MatchExpressions[i].Values)
		slices.Sort(out.MatchExpressions[i].Values)
		out.MatchExpressions[i].Values = slices.Compact(out.MatchExpressions[i].Values)
	}
	return out, nil
}
func validateSelector(s LabelSelector) error {
	if len(s.MatchLabels)+len(s.MatchExpressions) == 0 {
		return fmt.Errorf("label selector must not be empty")
	}
	for k := range s.MatchLabels {
		if k == "" {
			return fmt.Errorf("label key must not be empty")
		}
	}
	for _, r := range s.MatchExpressions {
		if r.Key == "" {
			return fmt.Errorf("requirement key must not be empty")
		}
		switch r.Operator {
		case SelectorIn, SelectorNotIn:
			if len(r.Values) == 0 {
				return fmt.Errorf("%s requires values", r.Operator)
			}
		case SelectorExists, SelectorDoesNotExist:
			if r.Values != nil {
				return fmt.Errorf("%s forbids supplied values", r.Operator)
			}
		default:
			return fmt.Errorf("unsupported selector operator %q", r.Operator)
		}
	}
	return nil
}

// Matches is an AND of all requirements. Invalid selectors fail closed.
func (s LabelSelector) Matches(labels map[string]string) bool {
	if validateSelector(s) != nil {
		return false
	}
	for k, want := range s.MatchLabels {
		got, ok := labels[k]
		if !ok || got != want {
			return false
		}
	}
	for _, r := range s.MatchExpressions {
		value, exists := labels[r.Key]
		switch r.Operator {
		case SelectorIn:
			if !exists || !slices.Contains(r.Values, value) {
				return false
			}
		case SelectorNotIn:
			if exists && slices.Contains(r.Values, value) {
				return false
			}
		case SelectorExists:
			if !exists {
				return false
			}
		case SelectorDoesNotExist:
			if exists {
				return false
			}
		}
	}
	return true
}
