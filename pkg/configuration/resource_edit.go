// SPDX-License-Identifier: Apache-2.0
package configuration

import (
	"fmt"
	"slices"

	"github.com/PicketX/PicketX/pkg/networkplan"
)

// UpdateResourceNetwork only edits existing entries of an unreferenced Resource.
// Whole-candidate compilation/transactional Replace remains the caller's job.
// The selected document is normalized; unrelated documents retain their bytes.
func (b *Bundle) UpdateResourceNetwork(uid string, entries []InventoryEndpoint) ([]byte, error) {
	remaining, err := b.RemoveResource(uid) // includes all inactive/dynamic references
	if err != nil {
		return nil, err
	}
	index := slices.IndexFunc(b.objects, func(o document) bool { return o.Kind == "Resource" && o.Metadata.UID == uid })
	original := b.objects[index]
	s := original.Spec.(resourceSpec)
	if len(entries) != len(s.Endpoints) || len(entries) == 0 || len(entries) > 64 {
		return nil, fmt.Errorf("endpoint membership is immutable")
	}
	byName := map[string]network{}
	for _, e := range entries {
		if _, ok := byName[e.Name]; ok || (e.Protocol != "TCP" && e.Protocol != "UDP") || e.Port == 0 {
			return nil, fmt.Errorf("invalid endpoint")
		}
		if e.Addresses != nil && (len(e.Addresses) == 0 || len(e.Addresses) > 64) {
			return nil, fmt.Errorf("invalid addresses")
		}
		if _, err := networkplan.ParseDestinations(e.Addresses); err != nil {
			return nil, err
		}
		byName[e.Name] = network{Protocol: e.Protocol, Port: e.Port, Addresses: slices.Clone(e.Addresses)}
	}
	s.Endpoints = slices.Clone(s.Endpoints)
	for i, e := range s.Endpoints {
		n, ok := byName[e.Name]
		if !ok {
			return nil, fmt.Errorf("endpoint membership is immutable")
		}
		s.Endpoints[i].Network = n
	}
	original.Spec = s
	rest, err := Decode(remaining)
	if err != nil {
		return nil, err
	}
	// A sole Resource leaves an explicit-empty marker. Preserve the trailing
	// stream comments that removeDocument kept, rather than appendObject's usual
	// empty-publication replacement dropping them.
	if len(b.objects) == 1 {
		start, end, err := b.documentSpan(index)
		if err != nil {
			return nil, err
		}
		encoded, err := (&Bundle{}).appendObject(original)
		if err != nil {
			return nil, err
		}
		out := append(slices.Clone(b.source[:start]), encoded...)
		out = append(out, []byte("...\n")...)
		out = append(out, b.source[end:]...)
		if _, err = Decode(out); err != nil {
			return nil, err
		}
		return out, nil
	}
	return rest.appendObject(original)
}
