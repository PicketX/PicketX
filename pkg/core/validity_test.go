// SPDX-License-Identifier: Apache-2.0
package core

import (
	"testing"
	"time"
)

func instant(s string) time.Time {
	v, err := time.Parse(time.RFC3339Nano, s)
	if err != nil {
		panic(err)
	}
	return v
}
func atHour(h string) time.Time { return instant("2026-10-15T" + h + ":00Z") }
func TestValidityBoundaries(t *testing.T) {
	v := Validity{Mode: LifetimeBounded, NotBefore: atHour("08:00"), ExpiresAt: atHour("10:00")}
	for _, tc := range []struct {
		at   time.Time
		want bool
	}{{atHour("07:59"), false}, {v.NotBefore, true}, {v.ExpiresAt.Add(-time.Nanosecond), true}, {v.ExpiresAt, false}, {atHour("12:00"), false}} {
		if got := v.ActiveAt(tc.at); got != tc.want {
			t.Errorf("at %v: %v want %v", tc.at, got, tc.want)
		}
	}
	p := Validity{Mode: LifetimePermanent, NotBefore: v.NotBefore}
	if p.ActiveAt(atHour("07:59")) || !p.ActiveAt(atHour("12:01")) {
		t.Fatal("permanent start/end")
	}
	if !(Validity{Mode: LifetimePermanent}).ActiveAt(atHour("08:00")) {
		t.Fatal("permanent without start")
	}
}
func TestValidityRejectsInvalidModes(t *testing.T) {
	for _, v := range []Validity{{}, {Mode: 99}, {Mode: LifetimeBounded}, {Mode: LifetimePermanent, ExpiresAt: atHour("10:00")}, {Mode: LifetimeBounded, NotBefore: atHour("10:00"), ExpiresAt: atHour("10:00")}, {Mode: LifetimeBounded, NotBefore: atHour("12:00"), ExpiresAt: atHour("10:00")}} {
		if _, err := NormalizeValidity(v); err == nil {
			t.Errorf("accepted %+v", v)
		}
		if v.ActiveAt(atHour("08:00")) || !v.NextBoundary(atHour("08:00")).IsZero() {
			t.Errorf("invalid validity active/boundary %+v", v)
		}
	}
	v := Validity{Mode: LifetimePermanent}
	if v.ActiveAt(time.Time{}) || !v.NextBoundary(time.Time{}).IsZero() {
		t.Fatal("zero clock")
	}
}
func TestNextBoundary(t *testing.T) {
	v := Validity{Mode: LifetimeBounded, NotBefore: atHour("08:00"), ExpiresAt: atHour("10:00")}
	for _, tc := range []struct{ at, want time.Time }{{atHour("07:59"), v.NotBefore}, {v.NotBefore, v.ExpiresAt}, {v.ExpiresAt, time.Time{}}} {
		if got := v.NextBoundary(tc.at); !got.Equal(tc.want) {
			t.Errorf("at %v: %v want %v", tc.at, got, tc.want)
		}
	}
	v.NotBefore = instant("2026-10-15T16:00:00+08:00")
	v.ExpiresAt = instant("2026-10-15T18:00:00+08:00")
	n, err := NormalizeValidity(v)
	if err != nil || !n.NotBefore.Equal(atHour("08:00")) || n.NotBefore.Location() != time.UTC || n.ExpiresAt.Location() != time.UTC {
		t.Fatalf("UTC: %+v %v", n, err)
	}
}

func TestNextBoundaryOmittedStartHistoricalClock(t *testing.T) {
	at := time.Date(0, 1, 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(2026, 10, 15, 10, 0, 0, 0, time.UTC)
	v := Validity{Mode: LifetimeBounded, ExpiresAt: end}
	if got := v.NextBoundary(at); !got.Equal(end) {
		t.Fatalf("got %v want %v", got, end)
	}
	v.NotBefore = atHour("08:00")
	if got := v.NextBoundary(at); !got.Equal(v.NotBefore) {
		t.Fatal(got)
	}
	if got := (Validity{Mode: LifetimePermanent}).NextBoundary(at); !got.IsZero() {
		t.Fatal(got)
	}
}
