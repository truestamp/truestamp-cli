// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"testing"
	"time"

	"github.com/truestamp/truestamp-cli/internal/ui"
)

// TestHumanizeAge_BothSignsAtEachUnit pins every unit on both sides of
// now, with the edges between units. It kills the mutants that drop the
// sign (a future time reads "Nd ago"), that invert it, that truncate the
// age instead of rounding it to the second, and that move a unit's edge
// (secs <= 3600 for secs < 3600).
func TestHumanizeAge_BothSignsAtEachUnit(t *testing.T) {
	cases := []struct {
		d    time.Duration
		want string
	}{
		{0, "0s ago"},
		{400 * time.Millisecond, "0s ago"},
		{-400 * time.Millisecond, "0s ago"},
		{-600 * time.Millisecond, "in 1s"},
		{59 * time.Second, "59s ago"},
		{-59 * time.Second, "in 59s"},
		{59*time.Second + 600*time.Millisecond, "1m ago"},
		{-(59*time.Second + 600*time.Millisecond), "in 1m"},
		{60 * time.Second, "1m ago"},
		{-60 * time.Second, "in 1m"},
		{3599 * time.Second, "59m ago"},
		{-3599 * time.Second, "in 59m"},
		{3600 * time.Second, "1h ago"},
		{-3600 * time.Second, "in 1h"},
		{86399 * time.Second, "23h ago"},
		{-86399 * time.Second, "in 23h"},
		{86400 * time.Second, "1d ago"},
		{-86400 * time.Second, "in 1d"},
		{334 * 24 * time.Hour, "334d ago"},
		{-334 * 24 * time.Hour, "in 334d"},
	}
	for _, c := range cases {
		if got := humanizeAge(c.d); got != c.want {
			t.Errorf("humanizeAge(%v) = %q, want %q", c.d, got, c.want)
		}
	}
}

// TestTimestampWithRelative_FutureAndPast checks that the hint follows
// the instant's side of now: an expiry two days ahead reads "in 2d" and
// a time three hours back "3h ago". It kills the mutant that drops the
// sign in humanizeAge, and the one that passes time.Until for
// time.Since.
func TestTimestampWithRelative_FutureAndPast(t *testing.T) {
	now := time.Now().UTC()
	cases := []struct {
		at   time.Time
		hint string
	}{
		{now.Add(48*time.Hour + 30*time.Minute), "in 2d"},
		{now.Add(-(3*time.Hour + 30*time.Minute)), "3h ago"},
	}
	for _, c := range cases {
		ts := c.at.Format(time.RFC3339Nano)
		want := ui.TruncateToSecond(ts) + "  (" + c.hint + ")"
		if got := timestampWithRelative(ts); got != want {
			t.Errorf("timestampWithRelative(%q) = %q, want %q", ts, got, want)
		}
	}
}
