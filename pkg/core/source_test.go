// SPDX-License-Identifier: Apache-2.0
package core

import (
	"net/netip"
	"testing"
)

// Catches host broadening, missing masking and mapped-prefix masking before validation.
func TestParseSource(t *testing.T) {
	for _, tc := range []struct{ in, want string }{
		{"192.0.2.10", "192.0.2.10/32"}, {"2001:db8::10", "2001:db8::10/128"},
		{"192.0.2.10/24", "192.0.2.0/24"}, {"2001:db8::1234/64", "2001:db8::/64"},
	} {
		t.Run(tc.in, func(t *testing.T) {
			p, err := ParseSource(tc.in)
			if err != nil || p.String() != tc.want {
				t.Fatalf("got %v, %v; want %s", p, err, tc.want)
			}
		})
	}
	for _, in := range []string{"", "example.com", " 192.0.2.10", "192.0.2.10 ", "192.0.2.10/33", "fe80::1%eth0", "fe80::1%eth0/64", "::ffff:192.0.2.10", "::ffff:192.0.2.10/64", "::ffff:192.0.2.10/120"} {
		t.Run(in, func(t *testing.T) {
			if _, err := ParseSource(in); err == nil {
				t.Fatal("invalid source accepted")
			}
		})
	}
}
func TestValidateSource(t *testing.T) {
	for _, p := range []netip.Prefix{{}, netip.MustParsePrefix("192.0.2.10/24"), netip.MustParsePrefix("::ffff:192.0.2.10/120")} {
		if ValidateSource(p) == nil {
			t.Errorf("accepted %v", p)
		}
	}
	for _, p := range []netip.Prefix{netip.MustParsePrefix("0.0.0.0/0"), netip.MustParsePrefix("::/0"), netip.MustParsePrefix("2001:db8::/64")} {
		if err := ValidateSource(p); err != nil {
			t.Errorf("%v: %v", p, err)
		}
	}
}
