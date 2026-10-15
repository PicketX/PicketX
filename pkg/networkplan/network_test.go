// SPDX-License-Identifier: Apache-2.0
package networkplan

import (
	"net/netip"
	"reflect"
	"testing"
)

func prefixes(values ...string) []netip.Prefix {
	var out []netip.Prefix
	for _, v := range values {
		out = append(out, netip.MustParsePrefix(v))
	}
	return out
}
func TestParseDestinationsPresenceAndFamilies(t *testing.T) {
	for _, tc := range []struct {
		in   []string
		want []netip.Prefix
		bad  bool
	}{
		{nil, nil, false}, {[]string{}, nil, true},
		{[]string{"192.0.2.9"}, prefixes("192.0.2.9/32"), false},
		{[]string{"2001:db8::9"}, prefixes("2001:db8::9/128"), false},
		{[]string{"192.0.2.129/24"}, prefixes("192.0.2.0/24"), false},
		{[]string{"::ffff:192.0.2.1"}, nil, true}, {[]string{"fe80::1%eth0"}, nil, true},
		{[]string{"::ffff:192.0.2.1/120"}, nil, true}, {[]string{"gitea.example"}, nil, true}, {[]string{"bad"}, nil, true},
	} {
		got, err := ParseDestinations(tc.in)
		if (err != nil) != tc.bad || !reflect.DeepEqual(got, tc.want) {
			t.Errorf("%v -> %v,%v want %v bad=%v", tc.in, got, err, tc.want, tc.bad)
		}
	}
}
func TestCanonicalDestinations(t *testing.T) {
	for _, tc := range []struct {
		in, want []netip.Prefix
		bad      bool
	}{
		{nil, nil, false}, {[]netip.Prefix{}, nil, true}, {[]netip.Prefix{{}}, nil, true},
		{prefixes("192.0.2.1/24"), nil, true}, {prefixes("::ffff:192.0.2.0/120"), nil, true},
		{prefixes("192.0.2.128/25", "192.0.2.9/32", "192.0.2.0/25", "192.0.2.0/25"), prefixes("192.0.2.0/24"), false},
		{prefixes("2001:db8:0:0:8000::/65", "2001:db8::/65"), prefixes("2001:db8::/64"), false},
		{prefixes("192.0.2.0/26", "192.0.2.64/26", "192.0.2.128/26", "192.0.2.192/26"), prefixes("192.0.2.0/24"), false},
		{prefixes("::/1", "8000::/1", "0.0.0.0/1", "128.0.0.0/1"), prefixes("0.0.0.0/0", "::/0"), false},
		{prefixes("192.0.2.0/26", "192.0.2.128/26"), prefixes("192.0.2.0/26", "192.0.2.128/26"), false},
	} {
		got, err := CanonicalDestinations(tc.in)
		if (err != nil) != tc.bad || !reflect.DeepEqual(got, tc.want) {
			t.Errorf("%v -> %v,%v want %v", tc.in, got, err, tc.want)
		}
	}
	in := prefixes("192.0.2.0/24")
	got, err := CanonicalDestinations(in)
	if err != nil {
		t.Fatal(err)
	}
	got[0] = netip.Prefix{}
	if !in[0].IsValid() {
		t.Fatal("input alias")
	}
}
func TestDestinationIntersection(t *testing.T) {
	for _, tc := range []struct {
		a, b, want []netip.Prefix
		bad        bool
	}{
		{prefixes("192.0.2.0/24"), prefixes("192.0.2.128/25"), prefixes("192.0.2.128/25"), false},
		{prefixes("192.0.2.0/24"), prefixes("198.51.100.0/24"), nil, false},
		{prefixes("192.0.2.0/24"), prefixes("::/0"), nil, false},
		{prefixes("::/0"), prefixes("2001:db8::/64"), prefixes("2001:db8::/64"), false},
		{nil, prefixes("0.0.0.0/0"), nil, true}, {prefixes("::/0"), []netip.Prefix{}, nil, true},
	} {
		got, err := IntersectDestinations(tc.a, tc.b)
		if (err != nil) != tc.bad || !reflect.DeepEqual(got, tc.want) {
			t.Errorf("%v/%v -> %v %v want %v", tc.a, tc.b, got, err, tc.want)
		}
	}
}
func TestNormalizeNetworkRejectsInvalid(t *testing.T) {
	base := NetworkDefinition{Entry: EntryRef{"gitea", "https"}, Protocol: TCP, Port: 443, Addresses: prefixes("192.0.2.0/24")}
	for _, mutate := range []func(*NetworkDefinition){
		func(n *NetworkDefinition) { n.Protocol = "" }, func(n *NetworkDefinition) { n.Protocol = "sctp" }, func(n *NetworkDefinition) { n.Port = 0 }, func(n *NetworkDefinition) { n.Entry.ResourceName = "" }, func(n *NetworkDefinition) { n.Entry.EndpointName = "" }, func(n *NetworkDefinition) { n.Addresses = []netip.Prefix{} },
	} {
		n := base
		mutate(&n)
		if _, err := NormalizeNetwork(n); err == nil {
			t.Errorf("accepted %v", n)
		}
	}
	for _, p := range []Protocol{TCP, UDP} {
		for _, port := range []uint16{1, 65535} {
			n := base
			n.Protocol = p
			n.Port = port
			got, err := NormalizeNetwork(n)
			if err != nil {
				t.Fatal(err)
			}
			got.Addresses[0] = netip.Prefix{}
			if !base.Addresses[0].IsValid() {
				t.Fatal("alias")
			}
		}
	}
	n := base
	n.Addresses = nil
	if got, err := NormalizeNetwork(n); err != nil || got.Addresses != nil {
		t.Fatal(got, err)
	}
}
