// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: MIT

package chrome

import (
	"strings"
	"testing"

	lipgloss "charm.land/lipgloss/v2"
)

// TestHeaderRendersTabsAndStatus confirms a typical input produces a
// header containing every tab label and the status text.
// We strip ANSI escapes via lipgloss.Width-style measurement isn't
// what we want here — assert on plain content via Contains since
// styles don't change the rendered glyphs.
func TestHeaderRendersTabsAndStatus(t *testing.T) {
	t.Parallel()

	theme := NewTheme()
	out := Render(HeaderInput{
		Width: 100,
		Tabs: []TabItem{
			{Number: 1, Title: "Monitor", Active: true},
			{Number: 2, Title: "New Item"},
			{Number: 3, Title: "Connection"},
		},
		Status:     "connected • pro • 7 streams",
		StatusKind: StatusKindOK,
		Clock:      "2026-04-27T20:39:02Z",
		Theme:      theme,
	})

	for _, want := range []string{
		"Monitor", "New Item", "Connection",
		"[1]", "[2]", "[3]",
		"connected", "7 streams",
		"2026-04-27T20:39:02Z",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("header missing %q\noutput:\n%s", want, out)
		}
	}
}

// TestHeaderUnderlineSpansFullWidth confirms the rule row beneath the
// tabs reaches the configured page width — visual containment relies
// on the rule extending edge-to-edge.
func TestHeaderUnderlineSpansFullWidth(t *testing.T) {
	t.Parallel()

	theme := NewTheme()
	out := Render(HeaderInput{
		Width: 80,
		Tabs: []TabItem{
			{Number: 1, Title: "Monitor", Active: true},
		},
		Status:     "x",
		StatusKind: StatusKindOK,
		Theme:      theme,
	})

	lines := strings.Split(out, "\n")
	if len(lines) < 2 {
		t.Fatalf("expected ≥2 lines, got %d:\n%s", len(lines), out)
	}
	rule := lines[1]
	if w := lipgloss.Width(rule); w != 80 {
		t.Errorf("rule width = %d, want 80\nrule: %q", w, rule)
	}
}

// TestHeaderRespectsZeroWidth verifies a degenerate input doesn't
// panic — the chrome package should always degrade gracefully.
func TestHeaderRespectsZeroWidth(t *testing.T) {
	t.Parallel()

	out := Render(HeaderInput{
		Width: 0,
		Tabs:  []TabItem{{Number: 1, Title: "x", Active: true}},
		Theme: NewTheme(),
	})
	if out != "" {
		t.Errorf("expected empty output for width=0, got %q", out)
	}
}
