// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: MIT

package chrome

import (
	lipgloss "charm.land/lipgloss/v2"
)

// Page is a value type carrying the terminal dimensions and a Theme
// reference. Constructed fresh on every render with the latest size.
//
// Page itself owns no state — it's a thin renderer that knows how the
// header / body / footer compose. Centralizing this composition keeps
// every pane a single Render(width, height) -> string call away from
// the rest of the layout, with no per-pane chrome reimplementation.
type Page struct {
	Width  int
	Height int
	Theme  *Theme
}

// BodyArea returns the inner-content rectangle the active pane should
// render into. Computed by subtracting the chrome rows from the
// terminal height; clamped to a 1x1 minimum so a tiny terminal still
// produces something renderable rather than panicking on negative
// dimensions.
func (p Page) BodyArea() (width, height int) {
	width = p.Width
	height = p.Height - p.Theme.HeaderHeight - p.Theme.FooterHeight
	if width < 1 {
		width = 1
	}
	if height < 1 {
		height = 1
	}
	return width, height
}

// Render assembles the three chrome sections into a single string the
// program can hand to tea.NewView. Header and footer are expected to
// have already been width-padded to p.Width by the caller; the body
// is height-clamped here so a pane that under-renders doesn't push the
// footer up the screen.
func (p Page) Render(header, body, footer string) string {
	_, bodyHeight := p.BodyArea()

	// The Height(...) clamp truncates over-rendering panes (rather
	// than letting them push the footer offscreen) and right-pads
	// under-rendering panes (so the footer stays at the bottom of
	// the terminal even on a sparse view).
	body = lipgloss.NewStyle().
		Width(p.Width).
		Height(bodyHeight).
		Render(body)

	return lipgloss.JoinVertical(lipgloss.Left, header, body, footer)
}
