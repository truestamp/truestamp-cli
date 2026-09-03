// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: MIT

// Package chrome provides the structural primitives that wrap every
// console pane: a fixed Page layout (header + body + footer), a Theme
// that centralizes colors / paddings / separators, a Header that draws
// the tab bar plus status pill, and a Footer that auto-renders
// keybinding hints from the active pane's help.KeyMap.
//
// The package owns the visual chrome only. Pane content is provided by
// each pane's own model.View call, sized to fit Page.BodyArea.
package chrome

import (
	"image/color"

	lipgloss "charm.land/lipgloss/v2"
	"github.com/truestamp/truestamp-cli/internal/ui"
)

// Theme is the single source of truth for the TUI's visual styling.
// Built once at startup from the shared `ui` palette so the console
// matches the rest of the CLI's output.
//
// Every style constant lives here; panes pull what they need rather
// than re-deriving from raw colors. Rationale: changing a color (e.g.
// switching from Catppuccin Mauve to Lavender) becomes a single-line
// edit instead of a project-wide grep.
type Theme struct {
	// Reserved row counts for the page chrome. Fixed so panes can
	// compute their body height without measuring rendered output.
	HeaderHeight int
	FooterHeight int

	// Foreground colors, re-exported from ui.* for ergonomic access
	// inside this package and in the chrome consumers.
	AccentFg color.Color // active tab text, focused pane title
	OkFg     color.Color // connected status, "live" indicator
	WarnFg   color.Color // reconnecting status, partial-state warnings
	ErrFg    color.Color // disconnected, error states, outage markers
	MutedFg  color.Color // inactive tabs, faint footer hints, separator

	// Composed styles. Built once and shared so we don't re-allocate
	// a lipgloss.Style on every render call.
	HeaderBar     lipgloss.Style // base style for the whole header row
	TabActive     lipgloss.Style // active tab pill
	TabInactive   lipgloss.Style // inactive tab pill
	StatusPill    lipgloss.Style // base for the right-side status
	StatusOK      lipgloss.Style // green-fg variant
	StatusWarn    lipgloss.Style // yellow-fg variant
	StatusErr     lipgloss.Style // red-fg variant
	HeaderUnderln lipgloss.Style // separator row beneath the header

	FooterBar     lipgloss.Style // base style for the whole footer row
	FooterHelpKey lipgloss.Style // key glyph in footer hint
	FooterHelpDsc lipgloss.Style // description after the glyph
	FooterHelpSep lipgloss.Style // separator between hints

	BodyPadding lipgloss.Style // applied to pane content
}

// NewTheme returns a Theme wired to the global `ui` palette. The
// global palette is set up once via ui.Init at CLI startup; calling
// NewTheme before that is safe because the palette has dark defaults.
func NewTheme() *Theme {
	t := &Theme{
		HeaderHeight: 2, // tab/status row + a thin separator row
		FooterHeight: 1,

		AccentFg: ui.Accent,
		OkFg:     ui.Green,
		WarnFg:   ui.Yellow,
		ErrFg:    ui.Red,
		MutedFg:  ui.Dim,
	}

	t.HeaderBar = lipgloss.NewStyle().Padding(0, 1)

	// Active tab: bold accent text, no background. We rely on the
	// faint-vs-bold contrast and the underline row beneath the header
	// for visual containment, rather than colored backgrounds (which
	// don't render consistently across terminal themes).
	t.TabActive = lipgloss.NewStyle().
		Bold(true).
		Foreground(ui.Accent).
		Padding(0, 1)

	t.TabInactive = lipgloss.NewStyle().
		Foreground(ui.Dim).
		Padding(0, 1)

	t.StatusPill = lipgloss.NewStyle().Padding(0, 1)
	t.StatusOK = t.StatusPill.Foreground(ui.Green)
	t.StatusWarn = t.StatusPill.Foreground(ui.Yellow)
	t.StatusErr = t.StatusPill.Foreground(ui.Red)

	// HeaderUnderln draws a thin horizontal rule beneath the tab row.
	// Rendered by repeating "─" across the page width, gives the
	// header a clear bottom edge so it reads as a banded zone instead
	// of floating text.
	t.HeaderUnderln = lipgloss.NewStyle().Foreground(ui.Dim)

	t.FooterBar = lipgloss.NewStyle().Padding(0, 1)
	t.FooterHelpKey = lipgloss.NewStyle().Foreground(ui.Label)
	t.FooterHelpDsc = lipgloss.NewStyle().Foreground(ui.Dim)
	t.FooterHelpSep = lipgloss.NewStyle().Foreground(ui.Dim)

	t.BodyPadding = lipgloss.NewStyle().Padding(1, 2)

	return t
}
