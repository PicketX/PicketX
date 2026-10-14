// SPDX-License-Identifier: Apache-2.0
package core

import (
	"fmt"
	"net/netip"
	"strings"
)

// ParseSource accepts an explicit host or CIDR. Only an explicitly supplied CIDR is masked.
func ParseSource(text string) (netip.Prefix, error) {
	if strings.Contains(text, "/") {
		p, err := netip.ParsePrefix(text)
		if err != nil {
			return netip.Prefix{}, fmt.Errorf("source: %w", err)
		}
		if err := validateAddress(p.Addr()); err != nil {
			return netip.Prefix{}, err
		}
		return p.Masked(), nil
	}
	a, err := netip.ParseAddr(text)
	if err != nil {
		return netip.Prefix{}, fmt.Errorf("source: %w", err)
	}
	if err := validateAddress(a); err != nil {
		return netip.Prefix{}, err
	}
	return netip.PrefixFrom(a, a.BitLen()), nil
}

// ValidateSource rejects invalid, mapped and unmasked typed prefixes.
func ValidateSource(prefix netip.Prefix) error {
	if !prefix.IsValid() {
		return fmt.Errorf("source prefix is invalid")
	}
	if err := validateAddress(prefix.Addr()); err != nil {
		return err
	}
	if prefix != prefix.Masked() {
		return fmt.Errorf("source prefix must be masked")
	}
	return nil
}
func validateAddress(a netip.Addr) error {
	if !a.IsValid() || a.Is4In6() || a.Zone() != "" {
		return fmt.Errorf("source must be a valid unmapped, unzoned address")
	}
	return nil
}
