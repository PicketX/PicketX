// SPDX-License-Identifier: Apache-2.0
package networkplan

import (
	"cmp"
	"fmt"
	"net/netip"
	"slices"

	"github.com/PicketX/PicketX/pkg/core"
)

// ParseDestinations accepts literal hosts or CIDRs. Nil means omitted, not ALL.
func ParseDestinations(values []string) ([]netip.Prefix, error) {
	if values == nil {
		return nil, nil
	}
	if len(values) == 0 {
		return nil, fmt.Errorf("destinations must not be empty")
	}
	out := make([]netip.Prefix, 0, len(values))
	for _, v := range values {
		p, err := core.ParseSource(v)
		if err != nil {
			return nil, fmt.Errorf("destination %q: %w", v, err)
		}
		out = append(out, p)
	}
	return CanonicalDestinations(out)
}

// CanonicalDestinations returns an owned minimal prefix union. Work is bounded
// by prefix count, never by the number of represented addresses.
func CanonicalDestinations(values []netip.Prefix) ([]netip.Prefix, error) {
	if values == nil {
		return nil, nil
	}
	if len(values) == 0 {
		return nil, fmt.Errorf("destinations must not be empty")
	}
	for _, v := range values {
		if err := core.ValidateSource(v); err != nil {
			return nil, fmt.Errorf("destination %v: %w", v, err)
		}
	}
	sorted := slices.Clone(values)
	slices.SortFunc(sorted, comparePrefix)
	var out []netip.Prefix
	for _, p := range sorted {
		for {
			if len(out) == 0 {
				out = append(out, p)
				break
			}
			last := out[len(out)-1]
			if prefixContains(last, p) {
				break
			}
			if last.Addr().BitLen() == p.Addr().BitLen() && last.Bits() == p.Bits() && p.Bits() > 0 {
				parent := netip.PrefixFrom(p.Addr(), p.Bits()-1).Masked()
				if netip.PrefixFrom(last.Addr(), last.Bits()-1).Masked() == parent {
					out = out[:len(out)-1]
					p = parent
					continue
				}
			}
			out = append(out, p)
			break
		}
	}
	return out, nil
}
func comparePrefix(a, b netip.Prefix) int {
	if c := cmp.Compare(a.Addr().BitLen(), b.Addr().BitLen()); c != 0 {
		return c
	}
	if c := a.Addr().Compare(b.Addr()); c != 0 {
		return c
	}
	return cmp.Compare(a.Bits(), b.Bits())
}
func prefixContains(a, b netip.Prefix) bool {
	return a.Addr().BitLen() == b.Addr().BitLen() && a.Bits() <= b.Bits() && a.Contains(b.Addr())
}

// IntersectDestinations requires explicit nonempty operands. An empty result
// denotes no intersection and must not be reinterpreted as omitted scope.
func IntersectDestinations(a, b []netip.Prefix) ([]netip.Prefix, error) {
	a, err := CanonicalDestinations(a)
	if err != nil {
		return nil, err
	}
	b, err = CanonicalDestinations(b)
	if err != nil {
		return nil, err
	}
	if len(a) == 0 || len(b) == 0 {
		return nil, fmt.Errorf("intersection requires two explicit destination sets")
	}
	var out []netip.Prefix
	for _, x := range a {
		for _, y := range b {
			if prefixContains(x, y) {
				out = append(out, y)
			} else if prefixContains(y, x) {
				out = append(out, x)
			}
		}
	}
	if len(out) == 0 {
		return nil, nil
	}
	return CanonicalDestinations(out)
}
func NormalizeNetwork(n NetworkDefinition) (NetworkDefinition, error) {
	if n.Entry.ResourceName == "" || n.Entry.EndpointName == "" {
		return NetworkDefinition{}, fmt.Errorf("network requires Resource and Endpoint references")
	}
	if n.Protocol != TCP && n.Protocol != UDP {
		return NetworkDefinition{}, fmt.Errorf("unsupported protocol %q", n.Protocol)
	}
	if n.Port == 0 {
		return NetworkDefinition{}, fmt.Errorf("destination port must be in 1–65535")
	}
	addresses, err := CanonicalDestinations(n.Addresses)
	if err != nil {
		return NetworkDefinition{}, err
	}
	n.Addresses = addresses
	return n, nil
}
