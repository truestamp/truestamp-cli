// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: MIT

package chrome

import (
	"strings"

	lipgloss "charm.land/lipgloss/v2"
)

// TabItem is a single entry in the header's tab bar. Number is the
// 1-indexed quick-jump key surfaced as `[N]`; Title is the human
// label.
type TabItem struct {
	Number int
	Title  string
	Active bool
}

// StatusKind classifies the right-side status pill so the renderer
// can pick the matching color (green for connected, yellow for
// connecting, red for disconnected/error states).
type StatusKind int

const (
	StatusKindNeutral StatusKind = iota
	StatusKindOK
	StatusKindWarn
	StatusKindErr
)

// HeaderInput bundles everything the renderer needs. Built fresh on
// each render call so the Header itself stays stateless — easier to
// reason about, easier to test.
type HeaderInput struct {
	Width      int
	Tabs       []TabItem
	Team       string     // optional active team label, rendered to the left of the status pill so the user can see scope on every pane
	Status     string     // primary status text (e.g. "connected • pro • 7 streams")
	StatusKind StatusKind // colors the status pill
	Clock      string     // optional trailing server-time, faint-styled
	Theme      *Theme
}

// Render returns a 2-row block: the tab/status line plus a thin
// horizontal rule beneath it. The rule is the only structural element
// — no colored background, no hard border — so the header reads as a
// "zone" without dominating the screen visually.
//
// Layout:
//
//	[1] Monitor   [2] New Item   [3] Connection                 connected • pro • 7 streams  •  …
//	──────────────────────────────────────────────────────────────────────────────────────────────
func Render(in HeaderInput) string {
	if in.Theme == nil || in.Width <= 0 {
		return ""
	}

	tabBar := renderTabs(in.Tabs, in.Theme)
	right := renderStatus(in.Team, in.Status, in.StatusKind, in.Clock, in.Theme)

	leftWidth := lipgloss.Width(tabBar)
	rightWidth := lipgloss.Width(right)
	gap := max(in.Width-leftWidth-rightWidth, 1)

	row := tabBar + strings.Repeat(" ", gap) + right
	rule := in.Theme.HeaderUnderln.Render(strings.Repeat("─", in.Width))

	return lipgloss.JoinVertical(lipgloss.Left, row, rule)
}

func renderTabs(tabs []TabItem, theme *Theme) string {
	rendered := make([]string, 0, len(tabs))
	for _, t := range tabs {
		label := formatTabLabel(t)
		if t.Active {
			rendered = append(rendered, theme.TabActive.Render(label))
		} else {
			rendered = append(rendered, theme.TabInactive.Render(label))
		}
	}
	return strings.Join(rendered, "")
}

func formatTabLabel(t TabItem) string {
	// Number prefix is always shown (lets users hit 1/2/3 to jump
	// without remembering the title). Cheap visual debug aid too.
	if t.Number > 0 {
		return numberGlyph(t.Number) + " " + t.Title
	}
	return t.Title
}

// numberGlyph returns a low-noise prefix for a tab. `①②③` would be
// fancier but slim; we use bracketed ASCII for terminal-font safety.
func numberGlyph(n int) string {
	switch n {
	case 1:
		return "[1]"
	case 2:
		return "[2]"
	case 3:
		return "[3]"
	case 4:
		return "[4]"
	case 5:
		return "[5]"
	case 6:
		return "[6]"
	case 7:
		return "[7]"
	case 8:
		return "[8]"
	case 9:
		return "[9]"
	}
	return ""
}

func renderStatus(team, status string, kind StatusKind, clock string, theme *Theme) string {
	var styled string
	switch kind {
	case StatusKindOK:
		styled = theme.StatusOK.Render(status)
	case StatusKindWarn:
		styled = theme.StatusWarn.Render(status)
	case StatusKindErr:
		styled = theme.StatusErr.Render(status)
	default:
		styled = theme.StatusPill.Render(status)
	}

	sep := theme.HeaderUnderln.Render(" • ")
	parts := []string{}
	if team != "" {
		// Team label rendered in the same faint underline color as
		// the clock so the eye still parks on the colored status
		// pill, not the surrounding context labels.
		parts = append(parts, theme.HeaderUnderln.Render(team))
	}
	parts = append(parts, styled)
	if clock != "" {
		parts = append(parts, theme.HeaderUnderln.Render(clock))
	}
	return strings.Join(parts, sep)
}
