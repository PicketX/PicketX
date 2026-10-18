// SPDX-License-Identifier: Apache-2.0
package agentsnapshot

import (
	"bytes"
	"encoding/json"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/PicketX/PicketX/pkg/configuration"
	"github.com/PicketX/PicketX/pkg/core"
	"github.com/PicketX/PicketX/pkg/networkplan"
	agentv1 "github.com/PicketX/PicketX/pkg/protocol/agent/v1alpha1"
	"google.golang.org/protobuf/proto"
	"net/netip"
)

func fixture(t *testing.T, empty bool) *configuration.Candidate {
	t.Helper()
	data, e := os.ReadFile("../../examples/static/gitea.yaml")
	if e != nil {
		t.Fatal(e)
	}
	if empty {
		data, e = os.ReadFile("../../examples/static/empty.yaml")
		if e != nil {
			t.Fatal(e)
		}
	}
	// Preserve a deadline finer than a millisecond in the actual configuration path.
	data = bytes.ReplaceAll(data, []byte("10:00:00Z"), []byte("10:00:00.123456789Z"))
	raw, e := os.ReadFile("../../examples/static/facts.json")
	if e != nil {
		t.Fatal(e)
	}
	var f configuration.Facts
	if e = json.Unmarshal(raw, &f); e != nil {
		t.Fatal(e)
	}
	a := f.Agents[0]
	a.Name = "agent-b"
	f.Agents = append(f.Agents, a)
	c := f.Contexts[0]
	c.AgentRef = "agent-b"
	f.Contexts = append(f.Contexts, c)
	b, e := configuration.Decode(data)
	if e != nil {
		t.Fatal(e)
	}
	out, e := configuration.Compile(b, f)
	if e != nil {
		t.Fatal(e)
	}
	return out
}
func TestScopedCompleteInventory(t *testing.T) {
	c := fixture(t, false)
	s, e := Build(c, "agent-a", 7)
	if e != nil {
		t.Fatal(e)
	}
	if e = Verify(s); e != nil {
		t.Fatal(e)
	}
	inv, e := c.Plan().Inventory("agent-a")
	if e != nil {
		t.Fatal(e)
	}
	if s.AgentRef != "agent-a" || s.AuthorityId != c.Input().AuthorityID || s.Revision != 7 || len(s.Targets) != len(inv) || len(inv) == 0 {
		t.Fatal(s)
	}
	for i, v := range inv {
		got := s.Targets[i]
		p := v.Protection
		if got.Key.AgentRef != p.Key.AgentRef || got.Key.ContextId != p.Key.ContextID || got.Key.ResourceId != p.Key.Entry.ResourceName || got.Key.EndpointName != p.Key.Entry.EndpointName {
			t.Fatal(got)
		}
		if got.Context.NetworkSpaceId != p.Context.NetworkSpaceID || got.Context.ProfileRef != p.Context.ProfileRef || got.Context.Handling != string(p.Context.Handling) || got.Context.ObservationStage != string(p.Context.Stage) || got.Context.TrafficClass != string(p.Context.Class) {
			t.Fatal(got.Context)
		}
		if len(got.BindingIds) != len(p.BindingIDs) || got.BindingIds[0] != p.BindingIDs[0] || len(got.Matches) != len(p.Matches) || !got.Required.OriginalDestination {
			t.Fatal(got)
		}
		for j, m := range p.Matches {
			if got.Matches[j].Destination != m.Destination.String() || got.Matches[j].Protocol != string(m.Protocol) || got.Matches[j].Port != uint32(m.Port) {
				t.Fatal(got.Matches[j])
			}
		}
		for j, g := range v.Grants {
			x := got.Grants[j]
			if x.Id != g.ID || x.Source != g.Source.String() || x.LifetimeMode != uint32(g.Validity.Mode) || x.Revoked != g.Revoked || x.ExpiresAt != g.Validity.ExpiresAt.UTC().Format(time.RFC3339Nano) {
				t.Fatal(x, g)
			}
		}
	}
	for _, a := range []string{"agent-b", "removed-agent"} {
		x, e := Build(c, a, 7)
		if e != nil || len(x.Targets) != 0 {
			t.Fatal(x, e)
		}
	}
	another, e := Build(c, "agent-a", 7)
	if e != nil || !proto.Equal(s, another) {
		t.Fatal(e)
	}
	s.Targets[0].BindingIds[0] = "mutated"
	if Verify(s) == nil {
		t.Fatal("tamper accepted")
	}
	fresh, _ := Build(c, "agent-a", 7)
	if !proto.Equal(fresh, another) {
		t.Fatal("snapshot aliases Candidate")
	}
	empty, _ := Build(fixture(t, true), "agent-a", 8)
	if len(empty.Targets) != 0 {
		t.Fatal(empty)
	}
}
func TestContributionLifetimes(t *testing.T) {
	at := time.Date(2026, 10, 18, 1, 2, 3, 123456789, time.UTC)
	var items []core.ResolvedContribution
	for i, deadline := range []time.Time{at.Add(-time.Hour), at.Add(time.Hour)} {
		items = append(items, core.ResolvedContribution{ID: []string{"expired", "future"}[i], Source: netip.MustParsePrefix("198.51.100.1/32"), Validity: core.Validity{Mode: core.LifetimeBounded, NotBefore: deadline.Add(-time.Minute), ExpiresAt: deadline}, Revoked: i == 1})
	}
	items = append(items, core.ResolvedContribution{ID: "permanent", Source: netip.MustParsePrefix("2001:db8::/64"), Validity: core.Validity{Mode: core.LifetimePermanent}})
	got := target(networkplan.TargetInventory{Grants: items, Denies: items})
	for _, domain := range [][]*agentv1.Contribution{got.Grants, got.Denies} {
		if len(domain) != 3 {
			t.Fatal(domain)
		}
		for i, v := range items {
			x := domain[i]
			if x.Id != v.ID || x.Source != v.Source.String() || x.LifetimeMode != uint32(v.Validity.Mode) || x.Revoked != v.Revoked {
				t.Fatal(x)
			}
		}
		if domain[0].ExpiresAt != at.Add(-time.Hour).Format(time.RFC3339Nano) || domain[1].NotBefore != at.Add(time.Hour-time.Minute).Format(time.RFC3339Nano) || domain[2].ExpiresAt != "" || domain[2].NotBefore != "" {
			t.Fatal(domain)
		}
	}
}
func TestRejectInvalidSnapshotInputs(t *testing.T) {
	c := fixture(t, false)
	for _, tc := range []struct {
		c *configuration.Candidate
		a string
		r uint64
	}{{nil, "a", 1}, {&configuration.Candidate{}, "a", 1}, {c, "", 1}, {c, "a", 0}, {c, strings.Repeat("a", 129), 1}} {
		if s, e := Build(tc.c, tc.a, tc.r); e == nil || s != nil {
			t.Fatal("accepted", tc)
		}
	}
	if Verify(nil) == nil {
		t.Fatal("nil accepted")
	}
	s, _ := Build(c, "agent-a", 1)
	s.Sha256 = nil
	if Verify(s) == nil {
		t.Fatal("missing hash accepted")
	}
}
