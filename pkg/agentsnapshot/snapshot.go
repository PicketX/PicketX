// SPDX-License-Identifier: Apache-2.0
// Package agentsnapshot maps compiled inventory to the alpha public wire contract.
// Snapshots express intent, never proof of installed policy.
package agentsnapshot

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"regexp"
	"time"

	"github.com/PicketX/PicketX/pkg/configuration"
	"github.com/PicketX/PicketX/pkg/core"
	"github.com/PicketX/PicketX/pkg/networkplan"
	agentv1 "github.com/PicketX/PicketX/pkg/protocol/agent/v1alpha1"
	"google.golang.org/protobuf/proto"
)

var identifier = regexp.MustCompile(`^[A-Za-z0-9_.:/-]{1,128}$`)

// Build creates an independently owned, complete scoped snapshot. An unknown
// Agent produces explicit empty intent, supporting removal while still enrolled.
func Build(c *configuration.Candidate, agent string, revision uint64) (*agentv1.Snapshot, error) {
	if c == nil || c.Plan() == nil || c.Hash() == "" || !identifier.MatchString(agent) || revision == 0 {
		return nil, fmt.Errorf("invalid snapshot input")
	}
	in := c.Input()
	if !identifier.MatchString(in.AuthorityID) {
		return nil, fmt.Errorf("invalid authority identity")
	}
	s := &agentv1.Snapshot{AuthorityId: in.AuthorityID, AgentRef: agent, Revision: revision}
	for _, a := range in.Agents {
		if a.Name != agent {
			continue
		}
		inv, e := c.Plan().Inventory(agent)
		if e != nil {
			return nil, e
		}
		for _, t := range inv {
			s.Targets = append(s.Targets, target(t))
		}
		break
	}
	d, e := digest(s)
	if e != nil {
		return nil, e
	}
	s.Sha256 = d
	return s, nil
}

// Verify checks envelope integrity only. The digest is not authentication;
// consumers must separately validate supported domain semantics/local scope.
func Verify(s *agentv1.Snapshot) error {
	if s == nil || !identifier.MatchString(s.AuthorityId) || !identifier.MatchString(s.AgentRef) || s.Revision == 0 || len(s.Sha256) != sha256.Size {
		return fmt.Errorf("invalid snapshot envelope")
	}
	d, e := digest(s)
	if e != nil {
		return e
	}
	if !bytes.Equal(d, s.Sha256) {
		return fmt.Errorf("snapshot digest mismatch")
	}
	return nil
}
func digest(s *agentv1.Snapshot) ([]byte, error) {
	owned := proto.Clone(s).(*agentv1.Snapshot)
	owned.Sha256 = nil
	data, e := (proto.MarshalOptions{Deterministic: true}).Marshal(owned)
	if e != nil {
		return nil, e
	}
	sum := sha256.Sum256(data)
	return sum[:], nil
}
func target(t networkplan.TargetInventory) *agentv1.Target {
	p := t.Protection
	c := p.Context
	r := p.Required
	out := &agentv1.Target{Key: &agentv1.TargetKey{AgentRef: p.Key.AgentRef, ContextId: p.Key.ContextID, ResourceId: p.Key.Entry.ResourceName, EndpointName: p.Key.Entry.EndpointName}, BindingIds: append([]string(nil), p.BindingIDs...), Context: &agentv1.Context{AgentRef: c.AgentRef, ProfileRef: c.ProfileRef, Id: c.ID, NetworkSpaceId: c.NetworkSpaceID, TrafficClass: string(c.Class), ObservationStage: string(c.Stage), Handling: string(c.Handling)}, Required: &agentv1.Capability{OriginalDestination: r.OriginalDestination}}
	for _, d := range c.Destinations {
		out.Context.Destinations = append(out.Context.Destinations, d.String())
	}
	for _, m := range p.Matches {
		out.Matches = append(out.Matches, &agentv1.Match{Destination: m.Destination.String(), Protocol: string(m.Protocol), Port: uint32(m.Port)})
	}
	for _, v := range r.Classes {
		out.Required.Classes = append(out.Required.Classes, string(v))
	}
	for _, v := range r.Protocols {
		out.Required.Protocols = append(out.Required.Protocols, string(v))
	}
	for _, v := range r.Families {
		out.Required.Families = append(out.Required.Families, uint32(v))
	}
	for _, v := range r.Handling {
		out.Required.Handling = append(out.Required.Handling, string(v))
	}
	out.Grants = contributions(t.Grants)
	out.Denies = contributions(t.Denies)
	return out
}
func contributions(in []core.ResolvedContribution) []*agentv1.Contribution {
	var out []*agentv1.Contribution
	for _, g := range in {
		out = append(out, &agentv1.Contribution{Id: g.ID, Source: g.Source.String(), LifetimeMode: uint32(g.Validity.Mode), NotBefore: formatTime(g.Validity.NotBefore), ExpiresAt: formatTime(g.Validity.ExpiresAt), Revoked: g.Revoked})
	}
	return out
}
func formatTime(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.UTC().Format(time.RFC3339Nano)
}
