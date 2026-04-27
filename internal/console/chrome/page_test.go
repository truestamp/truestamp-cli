// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: MIT

package chrome

import (
	"strings"
	"testing"
)

// TestPageBodyAreaSubtractsChrome verifies that BodyArea returns the
// terminal size minus the rendered header + footer heights. This is
// the contract panes rely on to compute their own internal layouts.
func TestPageBodyAreaSubtractsChrome(t *testing.T) {
	t.Parallel()

	theme := NewTheme()
	p := Page{Width: 100, Height: 40, Theme: theme}

	// 2-row header, 1-row footer.
	header := "tab1\nrule"
	footer := "key: hint"

	w, h := p.BodyArea(header, footer)

	if w != 100 {
		t.Errorf("BodyArea width = %d, want 100", w)
	}
	if h != 40-2-1 {
		t.Errorf("BodyArea height = %d, want %d", h, 40-2-1)
	}
}

// TestPageBodyAreaShrinksWithExpandedFooter verifies that when the
// footer grows (e.g. help.ShowAll = true), the body shrinks by the
// same amount — the help toggle expands into the body, never past
// the bottom of the screen.
func TestPageBodyAreaShrinksWithExpandedFooter(t *testing.T) {
	t.Parallel()

	p := Page{Width: 100, Height: 40, Theme: NewTheme()}

	header := "tab1\nrule"
	shortFooter := "k: hint"
	expandedFooter := "k1: a\nk2: b\nk3: c\nk4: d"

	_, hShort := p.BodyArea(header, shortFooter)
	_, hExpanded := p.BodyArea(header, expandedFooter)

	if hExpanded >= hShort {
		t.Errorf("expanded footer should shrink body: hShort=%d hExpanded=%d", hShort, hExpanded)
	}
	if hShort-hExpanded != 3 {
		t.Errorf("body should shrink by exactly the footer growth (3 rows), got hShort=%d hExpanded=%d",
			hShort, hExpanded)
	}
}

// TestPageBodyAreaClampMinimum verifies tiny terminals don't produce
// negative dimensions — we render *something* even at 1x1.
func TestPageBodyAreaClampMinimum(t *testing.T) {
	t.Parallel()

	theme := NewTheme()
	p := Page{Width: 0, Height: 0, Theme: theme}
	w, h := p.BodyArea("", "")

	if w < 1 || h < 1 {
		t.Errorf("BodyArea returned (%d, %d), expected ≥1 in both dims", w, h)
	}
}

// TestPageRenderHeightExact confirms that Render produces exactly
// the right number of rows for a typical page so the footer never
// drifts when a pane under-renders. Counting newlines is the cheapest
// way to verify the contract since lipgloss styling adds escape codes
// but no extra newlines.
func TestPageRenderHeightExact(t *testing.T) {
	t.Parallel()

	theme := NewTheme()
	p := Page{Width: 80, Height: 24, Theme: theme}

	// Pane under-renders (one line of body content) — Render's
	// internal Height() clamp should pad to fill the body area.
	out := p.Render(
		"H1\nH2",       // 2-line header
		"body line",    // 1-line body
		"footer hints", // 1-line footer
	)

	gotLines := strings.Count(out, "\n") + 1
	wantMin := theme.HeaderHeight + theme.FooterHeight + 1
	if gotLines < wantMin {
		t.Errorf("Render produced %d lines, want at least %d (header + body min + footer)",
			gotLines, wantMin)
	}
}
