// SPDX-License-Identifier: Apache-2.0
package core

import (
	"fmt"
	"time"
)

type LifetimeMode uint8

const (
	LifetimeBounded   LifetimeMode = 1
	LifetimePermanent LifetimeMode = 2
)

type Validity struct {
	Mode                 LifetimeMode
	NotBefore, ExpiresAt time.Time
}

// NormalizeValidity preserves absolute instants and strips local timezone/monotonic state.
func NormalizeValidity(v Validity) (Validity, error) {
	switch v.Mode {
	case LifetimeBounded:
		if v.ExpiresAt.IsZero() {
			return Validity{}, fmt.Errorf("bounded validity requires expiry")
		}
		if !v.NotBefore.IsZero() && !v.ExpiresAt.After(v.NotBefore) {
			return Validity{}, fmt.Errorf("expiry must follow activation")
		}
	case LifetimePermanent:
		if !v.ExpiresAt.IsZero() {
			return Validity{}, fmt.Errorf("permanent validity forbids expiry")
		}
	default:
		return Validity{}, fmt.Errorf("unsupported lifetime mode %d", v.Mode)
	}
	v.NotBefore = v.NotBefore.UTC()
	v.ExpiresAt = v.ExpiresAt.UTC()
	return v, nil
}
func (v Validity) ActiveAt(at time.Time) bool {
	n, err := NormalizeValidity(v)
	if err != nil || at.IsZero() {
		return false
	}
	return (n.NotBefore.IsZero() || !at.Before(n.NotBefore)) && (n.Mode == LifetimePermanent || at.Before(n.ExpiresAt))
}
func (v Validity) NextBoundary(at time.Time) time.Time {
	n, err := NormalizeValidity(v)
	if err != nil || at.IsZero() {
		return time.Time{}
	}
	if n.NotBefore.After(at) {
		return n.NotBefore
	}
	if n.ExpiresAt.After(at) {
		return n.ExpiresAt
	}
	return time.Time{}
}
