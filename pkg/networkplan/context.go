// SPDX-License-Identifier: Apache-2.0
package networkplan

import (
	"cmp"
	"fmt"
	"slices"
)

func validClass(c TrafficClass) bool    { return c == HostLocal || c == PublishedIngress || c == Transit }
func validHandling(h HandlingMode) bool { return h == Graceful || h == Strict }
func validateContextFields(class TrafficClass, stage ObservationStage, handling HandlingMode) error {
	if !validClass(class) {
		return fmt.Errorf("unsupported traffic class %q", class)
	}
	if stage != OriginalDestination {
		return fmt.Errorf("unsupported observation stage %q", stage)
	}
	if !validHandling(handling) {
		return fmt.Errorf("unsupported connection handling %q", handling)
	}
	return nil
}
func NormalizeContextRequest(r ContextRequest) (ContextRequest, error) {
	if r.ProfileRef == "" {
		return ContextRequest{}, fmt.Errorf("context profile reference is required")
	}
	if err := validateContextFields(r.Class, r.Stage, r.Handling); err != nil {
		return ContextRequest{}, err
	}
	addresses, err := CanonicalDestinations(r.AllowedDestinations)
	if err != nil {
		return ContextRequest{}, err
	}
	if len(addresses) == 0 {
		return ContextRequest{}, fmt.Errorf("context request requires an explicit destination boundary")
	}
	r.AllowedDestinations = addresses
	return r, nil
}
func NormalizeExecutionContext(c ExecutionContext) (ExecutionContext, error) {
	if c.AgentRef == "" || c.ProfileRef == "" || c.ID == "" || c.NetworkSpaceID == "" {
		return ExecutionContext{}, fmt.Errorf("context requires Agent, profile, identity and trusted network space")
	}
	if err := validateContextFields(c.Class, c.Stage, c.Handling); err != nil {
		return ExecutionContext{}, err
	}
	addresses, err := CanonicalDestinations(c.Destinations)
	if err != nil {
		return ExecutionContext{}, err
	}
	if len(addresses) == 0 {
		return ExecutionContext{}, fmt.Errorf("execution context requires an explicit destination boundary")
	}
	c.Destinations = addresses
	return c, nil
}
func sortedUnique[T cmp.Ordered](v []T) []T {
	out := slices.Clone(v)
	slices.Sort(out)
	return slices.Compact(out)
}
func NormalizeCapabilities(c *CapabilityProfile) (*CapabilityProfile, error) {
	if c == nil {
		return nil, nil
	}
	for _, v := range c.Classes {
		if !validClass(v) {
			return nil, fmt.Errorf("unsupported capability class %q", v)
		}
	}
	for _, v := range c.Protocols {
		if v != TCP && v != UDP {
			return nil, fmt.Errorf("unsupported capability protocol %q", v)
		}
	}
	for _, v := range c.Families {
		if v != IPv4 && v != IPv6 {
			return nil, fmt.Errorf("unsupported capability family %d", v)
		}
	}
	for _, v := range c.Handling {
		if !validHandling(v) {
			return nil, fmt.Errorf("unsupported capability handling %q", v)
		}
	}
	return &CapabilityProfile{Classes: sortedUnique(c.Classes), Protocols: sortedUnique(c.Protocols), Families: sortedUnique(c.Families), Handling: sortedUnique(c.Handling), OriginalDestination: c.OriginalDestination}, nil
}

// ResolveContext validates trusted resolved facts against the authoritative
// request. Broader facts are rejected rather than clipped under a reused ID.
// Traffic-class declarations still need concrete enforcement by the adapter.
func ResolveContext(r ContextRequest, c ExecutionContext, caps *CapabilityProfile) (ExecutionContext, error) {
	r, err := NormalizeContextRequest(r)
	if err != nil {
		return ExecutionContext{}, err
	}
	c, err = NormalizeExecutionContext(c)
	if err != nil {
		return ExecutionContext{}, err
	}
	caps, err = NormalizeCapabilities(caps)
	if err != nil {
		return ExecutionContext{}, err
	}
	if caps == nil {
		return ExecutionContext{}, fmt.Errorf("selected target has no capability profile")
	}
	if r.ProfileRef != c.ProfileRef || r.Class != c.Class || r.Stage != c.Stage || r.Handling != c.Handling {
		return ExecutionContext{}, fmt.Errorf("resolved context %q does not satisfy requested profile/class/stage/handling", c.ID)
	}
	if !slices.Contains(caps.Classes, r.Class) || !caps.OriginalDestination || !slices.Contains(caps.Handling, r.Handling) {
		return ExecutionContext{}, fmt.Errorf("selected target cannot support requested context")
	}
	overlap, err := IntersectDestinations(r.AllowedDestinations, c.Destinations)
	if err != nil {
		return ExecutionContext{}, err
	}
	if !slices.Equal(overlap, c.Destinations) {
		return ExecutionContext{}, fmt.Errorf("resolved context %q exceeds authoritative destination boundary", c.ID)
	}
	return c, nil
}
