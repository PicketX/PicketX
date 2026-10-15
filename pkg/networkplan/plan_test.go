// SPDX-License-Identifier: Apache-2.0
package networkplan

import (
	"github.com/PicketX/PicketX/pkg/core"
	"net/netip"
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestProjectionBoundaryAndFamilies(t *testing.T) {
	in := baseInput()
	in.Networks[0].Addresses = prefixes("192.0.2.128/25")
	v := projectionOK(t, planOK(t, in), "09:00")
	var https Protection
	for _, x := range v.Protected {
		if x.Key.Entry.EndpointName == "https" {
			https = x
		}
	}
	if !reflect.DeepEqual(https.Matches, []Match{{Destination: netip.MustParsePrefix("192.0.2.128/25"), Protocol: TCP, Port: 443}}) {
		t.Fatal(https)
	}
	for _, m := range v.Allow {
		if m.Match.Destination.Addr().Is6() {
			t.Fatal("cross-family allow", m)
		}
	}
	in.Networks[0].Addresses = prefixes("203.0.113.0/24")
	p, e := Compile(in)
	if e == nil || p != nil {
		t.Fatal(p, e)
	}
	for _, s := range []string{"agent-a", "b1", "gitea", "https"} {
		if !strings.Contains(e.Error(), s) {
			t.Errorf("%s missing in %v", s, e)
		}
	}
	in = baseInput()
	v = projectionOK(t, planOK(t, in), "09:00")
	if len(v.Protected) != 2 || len(v.Protected[0].Matches) != 2 {
		t.Fatal(v)
	}
}
func TestProjectionMembershipAndSchedule(t *testing.T) {
	in := baseInput()
	g := in.Logical.Grants[0]
	g.ID = "shared"
	g.Validity = core.Validity{Mode: core.LifetimeBounded, NotBefore: clock("09:30"), ExpiresAt: clock("10:30")}
	in.Logical.Denies = []core.Deny{core.Deny(g)}
	in.Logical.Grants[0].ID = "shared"
	revoked := g
	revoked.ID = "revoked"
	revoked.Revoked = true
	revoked.Validity = core.Validity{Mode: core.LifetimePermanent}
	in.Logical.Grants = append(in.Logical.Grants, revoked)
	p := planOK(t, in)
	for _, tc := range []struct {
		at          string
		allow, deny int
		next        string
	}{
		{"09:00", 4, 0, "09:30"}, {"09:30", 4, 2, "10:00"}, {"10:00", 2, 2, "10:30"}, {"10:30", 2, 0, "11:00"}, {"11:00", 0, 0, ""},
	} {
		v := projectionOK(t, p, tc.at)
		if len(v.Allow) != tc.allow || len(v.Deny) != tc.deny || len(v.Protected) != 2 {
			t.Fatalf("at %s: %+v", tc.at, v)
		}
		want := time.Time{}
		if tc.next != "" {
			want = clock(tc.next)
		}
		if !v.NextChange.Equal(want) {
			t.Fatalf("at %s next %v want %v", tc.at, v.NextChange, want)
		}
		for _, m := range v.Allow {
			if m.Contribution.ID == "revoked" || m.Match.Destination.Addr().Is6() {
				t.Fatal(m)
			}
		}
	}
	in = baseInput()
	in.Logical.Grants = in.Logical.Grants[:1]
	in.Logical.Grants[0].Validity = core.Validity{Mode: core.LifetimePermanent, NotBefore: clock("08:30")}
	p = planOK(t, in)
	v := projectionOK(t, p, "08:00")
	if len(v.Allow) != 0 || !v.NextChange.Equal(clock("08:30")) {
		t.Fatal(v)
	}
	v = projectionOK(t, p, "08:30")
	if len(v.Allow) != 2 || !v.NextChange.IsZero() {
		t.Fatal(v)
	}
}
func TestProtectedWithoutPermissions(t *testing.T) {
	in := baseInput()
	p := planOK(t, in)
	v := projectionOK(t, p, "12:00")
	if len(v.Protected) != 2 || len(v.Allow) != 0 {
		t.Fatal(v)
	}
	in.Bindings = nil
	p = planOK(t, in)
	v = projectionOK(t, p, "09:00")
	if len(v.Protected) != 0 || len(v.Allow) != 0 {
		t.Fatal(v)
	}
	refs, e := p.AgentRefs()
	if e != nil || len(refs) != 0 {
		t.Fatal(refs, e)
	}
	for _, bad := range []*Plan{nil, {}} {
		if _, e := bad.AgentRefs(); e == nil {
			t.Fatal("invalid AgentRefs")
		}
		if _, e := bad.Diagnostics(); e == nil {
			t.Fatal("invalid Diagnostics")
		}
		if _, e := bad.Inventory("agent-a"); e == nil {
			t.Fatal("invalid Inventory")
		}
		if _, e := bad.Projection("agent-a", clock("09:00")); e == nil {
			t.Fatal("invalid Projection")
		}
	}
	if _, e := p.Projection("unknown", clock("09:00")); e == nil {
		t.Fatal("unknown Agent")
	}
	if _, e := p.Inventory("unknown"); e == nil {
		t.Fatal("unknown inventory")
	}
	if _, e := p.Projection("agent-a", time.Time{}); e == nil {
		t.Fatal("zero time")
	}
	in = baseInput()
	p = planOK(t, in)
	other, e := p.Projection("agent-b", clock("09:00"))
	if e != nil || len(other.Protected) != 0 {
		t.Fatal(other, e)
	}
}
