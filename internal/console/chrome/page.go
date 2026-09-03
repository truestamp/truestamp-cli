// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: MIT

package chrome

import (
	lipgloss "charm.land/lipgloss/v2"
)

// Page is a value type carrying the terminal dimensions and a Theme
// reference. Constructed fresh on every render with the latest size.
//
// Page itself owns no state, it's a thin renderer that knows how the
// header / body / footer compose. Centralizing this composition keeps
// every pane a single Render(width, height) -> string call away from
// the rest of the layout, with no per-pane chrome reimplementation.
type Page struct {
	Width  int
	Height int
	Theme  *Theme
}

// BodyArea returns the inner-content rectangle for a given header
// and footer. Caller passes the already-rendered chrome strings;
// BodyArea measures their actual height (which can vary, the
// footer's bubbles/help expands when ShowAll toggles on) and
// subtracts. Clamped to a 1x1 minimum so a tiny terminal still
// produces something renderable rather than panicking on negative
// dimensions.
//
// The Theme.HeaderHeight / Theme.FooterHeight constants are kept as
// defaults for tests and for callers that don't want to render
// chrome up-front, but the runtime View path uses this dynamic
// measurement so the help-toggle can grow the footer without
// truncating the body.
func (p Page) BodyArea(header, footer string) (width, height int) {
	width = p.Width
	height = p.Height - lipgloss.Height(header) - lipgloss.Height(footer)
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
// have already been width-padded to p.Width by the caller. The body
// is height-clamped here so a pane that under-renders doesn't push
// the footer up the screen, and an over-rendering pane gets its
// trailing rows truncated rather than pushing the footer offscreen.
func (p Page) Render(header, body, footer string) string {
	_, bodyHeight := p.BodyArea(header, footer)

	body = lipgloss.NewStyle().
		Width(p.Width).
		Height(bodyHeight).
		Render(body)

	return lipgloss.JoinVertical(lipgloss.Left, header, body, footer)
}
