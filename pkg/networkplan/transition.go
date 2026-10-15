// SPDX-License-Identifier: Apache-2.0
package networkplan

import (
	"fmt"
	"github.com/PicketX/PicketX/pkg/core"
	"slices"
)

// Replace validates a complete candidate against the same authority's previous
// accepted desired plan. Failure returns no replacement or withdrawal actions.
// New contribution IDs are authoritative publisher input: this library cannot
// infer approval lineage or authorize fresh deadlines on the publisher's behalf.
func Replace(previous *Plan, input Input) (Replacement, error) {
	if !previous.valid() {
		return Replacement{}, fmt.Errorf("previous plan is not compiled")
	}
	if previous.authority != input.AuthorityID {
		return Replacement{}, fmt.Errorf("replacement authority %q differs from previous %q", input.AuthorityID, previous.authority)
	}
	next, err := Compile(input)
	if err != nil {
		return Replacement{}, err
	}
	var entries []EntryRef
	for ref := range previous.networks {
		entries = append(entries, ref)
	}
	slices.SortFunc(entries, compareEntry)
	for _, ref := range entries {
		old := previous.networks[ref]
		candidate, ok := next.networks[ref]
		if !ok || equalNetwork(old, candidate) {
			continue
		}
		before, after := previous.entries[ref], next.entries[ref]
		var retained []string
		for _, domain := range []struct {
			kind     string
			old, new []core.ResolvedContribution
		}{{"grant", before.Grants, after.Grants}, {"deny", before.Denies, after.Denies}} {
			ids := make(map[string]struct{})
			for _, c := range domain.new {
				ids[c.ID] = struct{}{}
			}
			for _, c := range domain.old {
				if _, ok := ids[c.ID]; ok {
					retained = append(retained, domain.kind+":"+c.ID)
				}
			}
		}
		if len(retained) > 0 {
			return Replacement{}, fmt.Errorf("retargeting entry %s/%s retains contributions %v; explicit fresh-ID reissuance is required", ref.ResourceName, ref.EndpointName, retained)
		}
	}
	// Check all trusted rows, not only current protected targets.
	var contexts []contextKey
	for key := range previous.contexts {
		contexts = append(contexts, key)
	}
	slices.SortFunc(contexts, func(a, b contextKey) int {
		if a.agent < b.agent {
			return -1
		}
		if a.agent > b.agent {
			return 1
		}
		if a.id < b.id {
			return -1
		}
		if a.id > b.id {
			return 1
		}
		return 0
	})
	for _, key := range contexts {
		if c, ok := next.contexts[key]; ok && !equalContext(previous.contexts[key], c) {
			return Replacement{}, fmt.Errorf("context ID %q on Agent %q changed meaning; provide a new explicit context identity", key.id, key.agent)
		}
	}
	var withdrawals []Withdrawal
	for _, key := range previous.targetKeys() {
		old := previous.targets[key]
		candidate, ok := next.targets[key]
		reason := ""
		if !ok {
			reason = "removed"
		} else if !slices.Equal(old.Protection.Matches, candidate.Protection.Matches) {
			reason = "match-changed"
		}
		if reason != "" {
			withdrawals = append(withdrawals, Withdrawal{Former: cloneInventory(old), Reason: reason})
		}
	}
	return Replacement{Plan: next, Withdrawals: withdrawals}, nil
}
func equalNetwork(a, b NetworkDefinition) bool {
	return a.Entry == b.Entry && a.Protocol == b.Protocol && a.Port == b.Port && (a.Addresses == nil) == (b.Addresses == nil) && slices.Equal(a.Addresses, b.Addresses)
}
func equalContext(a, b ExecutionContext) bool {
	return a.AgentRef == b.AgentRef && a.ProfileRef == b.ProfileRef && a.ID == b.ID && a.NetworkSpaceID == b.NetworkSpaceID && a.Class == b.Class && a.Stage == b.Stage && a.Handling == b.Handling && slices.Equal(a.Destinations, b.Destinations)
}
