// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: MIT

package chrome

import (
	"charm.land/bubbles/v2/help"
	"charm.land/bubbles/v2/key"
	lipgloss "charm.land/lipgloss/v2"
)

// Footer wraps `bubbles/v2/help.Model` so each pane can hand it a
// help.KeyMap and get a consistent footer-height keybinding hint row
// for free. Toggle ShowAll to expand to the full multi-row layout.
type Footer struct {
	help  help.Model
	theme *Theme
}

// NewFooter constructs a Footer with theme-derived help styles.
// Width is set per-render via Render(width, keymap).
func NewFooter(theme *Theme) Footer {
	h := help.New()
	h.Styles = help.Styles{
		ShortKey:       theme.FooterHelpKey,
		ShortDesc:      theme.FooterHelpDsc,
		ShortSeparator: theme.FooterHelpSep,
		FullKey:        theme.FooterHelpKey,
		FullDesc:       theme.FooterHelpDsc,
		FullSeparator:  theme.FooterHelpSep,
		Ellipsis:       theme.FooterHelpSep,
	}
	return Footer{help: h, theme: theme}
}

// SetShowAll toggles between the short single-line help and the full
// expanded help. The TUI's `?` binding flips this.
func (f *Footer) SetShowAll(show bool) {
	f.help.ShowAll = show
}

// ShowAll returns the current expanded-help state.
func (f *Footer) ShowAll() bool {
	return f.help.ShowAll
}

// Render produces the footer string, padded to width. Hits the
// help.Model with the active pane's keymap so the displayed hints
// always reflect what the user can actually do right now.
func (f *Footer) Render(width int, keymap help.KeyMap) string {
	f.help.SetWidth(width)
	rendered := f.help.View(keymap)
	// The help model returns plain text without horizontal padding;
	// our FooterBar adds the same `Padding(0, 1)` as the header so the
	// chrome rows visually align.
	return f.theme.FooterBar.Width(width).Render(rendered)
}

// EmptyKeyMap is a no-op help.KeyMap that returns no bindings. Useful
// as a placeholder before the first pane has loaded.
type EmptyKeyMap struct{}

// ShortHelp implements help.KeyMap.
func (EmptyKeyMap) ShortHelp() []key.Binding { return nil }

// FullHelp implements help.KeyMap.
func (EmptyKeyMap) FullHelp() [][]key.Binding { return nil }

// guard against unused imports if all chrome consumers also stop
// referencing lipgloss directly (currently they don't, but keeping
// the var here documents the intent).
var _ = lipgloss.NoColor{}
