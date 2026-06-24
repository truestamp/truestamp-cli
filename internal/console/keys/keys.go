// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: MIT

// Package keys centralises all keyboard bindings for the console TUI.
// Every binding lives here so:
//
//   - The footer's bubbles/help component can auto-render hints from
//     the active pane's KeyMap (no hand-maintained help strings).
//   - Pane Update handlers use key.Matches(msg, binding) instead of
//     stringly-typed key comparisons.
//   - Re-keying is one-line edit.
//
// Each pane exports a `KeyMap` struct that implements `help.KeyMap`
// (ShortHelp + FullHelp). `App` holds the cross-pane bindings and is
// composed into every pane's FullHelp so global shortcuts stay
// surfaced no matter where focus is.
package keys

import (
	"charm.land/bubbles/v2/key"
)

// AppKeys are global bindings: pane-switching, help toggle, quit.
// Used by the root model (`app.go`) and surfaced in every pane's full
// help so the user always sees how to leave the current pane.
type AppKeys struct {
	NextPane     key.Binding
	PrevPane     key.Binding
	GoMonitor    key.Binding
	GoNewItem    key.Binding
	GoTeam       key.Binding
	GoConnection key.Binding
	ToggleHelp   key.Binding
	Quit         key.Binding
}

// NewAppKeys returns the canonical app-level bindings. Plain `tab` is
// reserved for pane-internal navigation; pane switching uses `]` /
// `[` (vi-adjacent), `ctrl+tab` / `ctrl+shift+tab` (browser-adjacent),
// and the numeric `1` / `2` / `3` / `4` quick-jumps.
func NewAppKeys() AppKeys {
	return AppKeys{
		NextPane: key.NewBinding(
			key.WithKeys("]", "ctrl+tab"),
			key.WithHelp("]", "next pane"),
		),
		PrevPane: key.NewBinding(
			key.WithKeys("[", "ctrl+shift+tab"),
			key.WithHelp("[", "prev pane"),
		),
		GoMonitor: key.NewBinding(
			key.WithKeys("1"),
			key.WithHelp("1", "monitor"),
		),
		GoNewItem: key.NewBinding(
			key.WithKeys("2"),
			key.WithHelp("2", "new item"),
		),
		GoTeam: key.NewBinding(
			key.WithKeys("3"),
			key.WithHelp("3", "teams"),
		),
		GoConnection: key.NewBinding(
			key.WithKeys("4"),
			key.WithHelp("4", "connection"),
		),
		ToggleHelp: key.NewBinding(
			key.WithKeys("?"),
			key.WithHelp("?", "help"),
		),
		Quit: key.NewBinding(
			key.WithKeys("q", "ctrl+c"),
			key.WithHelp("q", "quit"),
		),
	}
}

// MonitorKeys covers the Monitor pane: focus left/right, scroll the
// waterfall, toggle subscriptions on the stream list.
type MonitorKeys struct {
	FocusLeft    key.Binding
	FocusRight   key.Binding
	Up           key.Binding
	Down         key.Binding
	PageUp       key.Binding
	PageDown     key.Binding
	Top          key.Binding
	Bottom       key.Binding
	ToggleStream key.Binding
	ReverseOrder key.Binding
	ToggleDetail key.Binding
	App          AppKeys
}

// NewMonitorKeys returns the Monitor pane's bindings.
func NewMonitorKeys(app AppKeys) MonitorKeys {
	return MonitorKeys{
		App: app,
		FocusLeft: key.NewBinding(
			key.WithKeys("left", "h"),
			key.WithHelp("←/h", "focus streams"),
		),
		FocusRight: key.NewBinding(
			key.WithKeys("right", "l"),
			key.WithHelp("→/l", "focus events"),
		),
		Up: key.NewBinding(
			key.WithKeys("up", "k"),
			key.WithHelp("↑/k", "up"),
		),
		Down: key.NewBinding(
			key.WithKeys("down", "j"),
			key.WithHelp("↓/j", "down"),
		),
		PageUp: key.NewBinding(
			key.WithKeys("pgup", "K"),
			key.WithHelp("pgup", "page up"),
		),
		PageDown: key.NewBinding(
			key.WithKeys("pgdn", "J"),
			key.WithHelp("pgdn", "page down"),
		),
		Top: key.NewBinding(
			key.WithKeys("home", "g"),
			key.WithHelp("g", "top"),
		),
		Bottom: key.NewBinding(
			key.WithKeys("end", "G"),
			key.WithHelp("G", "bottom"),
		),
		ToggleStream: key.NewBinding(
			key.WithKeys(" ", "space"),
			key.WithHelp("space", "toggle stream"),
		),
		ReverseOrder: key.NewBinding(
			key.WithKeys("r"),
			key.WithHelp("r", "reverse order"),
		),
		ToggleDetail: key.NewBinding(
			key.WithKeys("d"),
			key.WithHelp("d", "toggle detail panel"),
		),
	}
}

// ShortHelp implements help.KeyMap.
func (k MonitorKeys) ShortHelp() []key.Binding {
	return []key.Binding{
		k.FocusLeft, k.FocusRight, k.Up, k.Down,
		k.ToggleStream, k.ReverseOrder, k.ToggleDetail,
		k.App.ToggleHelp, k.App.Quit,
	}
}

// FullHelp implements help.KeyMap.
func (k MonitorKeys) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.FocusLeft, k.FocusRight, k.Up, k.Down, k.PageUp, k.PageDown, k.Top, k.Bottom},
		{k.ToggleStream, k.ReverseOrder, k.ToggleDetail},
		{k.App.NextPane, k.App.PrevPane, k.App.GoMonitor, k.App.GoNewItem, k.App.GoTeam, k.App.GoConnection},
		{k.App.ToggleHelp, k.App.Quit},
	}
}

// NewItemKeys covers the New Item form: tab between fields, submit,
// reset.
type NewItemKeys struct {
	NextField key.Binding
	PrevField key.Binding
	Submit    key.Binding
	Clear     key.Binding
	NewForm   key.Binding
	App       AppKeys
}

// NewNewItemKeys returns the New Item form bindings. Plain `tab` is
// kept inside the form here — the root model only consumes pane-
// switch keys (`]`, `[`, `1`/`2`/`3`) so this binding wins for
// in-form navigation.
func NewNewItemKeys(app AppKeys) NewItemKeys {
	return NewItemKeys{
		App: app,
		NextField: key.NewBinding(
			key.WithKeys("tab", "down"),
			key.WithHelp("tab", "next field"),
		),
		PrevField: key.NewBinding(
			key.WithKeys("shift+tab", "up"),
			key.WithHelp("shift+tab", "prev field"),
		),
		Submit: key.NewBinding(
			key.WithKeys("enter"),
			key.WithHelp("enter", "submit"),
		),
		Clear: key.NewBinding(
			key.WithKeys("esc"),
			key.WithHelp("esc", "clear"),
		),
		NewForm: key.NewBinding(
			key.WithKeys("n"),
			key.WithHelp("n", "new item"),
		),
	}
}

// ShortHelp implements help.KeyMap.
func (k NewItemKeys) ShortHelp() []key.Binding {
	return []key.Binding{
		k.NextField, k.PrevField, k.Submit, k.Clear,
		k.App.ToggleHelp, k.App.Quit,
	}
}

// FullHelp implements help.KeyMap.
func (k NewItemKeys) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.NextField, k.PrevField, k.Submit, k.Clear, k.NewForm},
		{k.App.NextPane, k.App.PrevPane, k.App.GoMonitor, k.App.GoNewItem, k.App.GoTeam, k.App.GoConnection},
		{k.App.ToggleHelp, k.App.Quit},
	}
}

// NewItemWatchingKeys covers the New Item pane after a successful
// items.create — the form is gone, the lifecycle card is up, and
// the only pane-local action is starting over with `n`. We surface
// a separate keymap for this state so the footer doesn't continue
// to advertise tab/shift+tab/enter/esc, none of which apply.
type NewItemWatchingKeys struct {
	NewForm key.Binding
	App     AppKeys
}

// NewNewItemWatchingKeys returns the watching-state bindings.
// `n` is the discoverable mnemonic; `esc` is the muscle-memory
// "go back" pair with the form-mode "esc clears" binding.
func NewNewItemWatchingKeys(app AppKeys) NewItemWatchingKeys {
	return NewItemWatchingKeys{
		App: app,
		NewForm: key.NewBinding(
			key.WithKeys("n", "esc"),
			key.WithHelp("n / esc", "new item"),
		),
	}
}

// ShortHelp implements help.KeyMap.
func (k NewItemWatchingKeys) ShortHelp() []key.Binding {
	return []key.Binding{
		k.NewForm,
		k.App.ToggleHelp,
		k.App.Quit,
	}
}

// FullHelp implements help.KeyMap.
func (k NewItemWatchingKeys) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.NewForm},
		{k.App.NextPane, k.App.PrevPane, k.App.GoMonitor, k.App.GoNewItem, k.App.GoTeam, k.App.GoConnection},
		{k.App.ToggleHelp, k.App.Quit},
	}
}

// TeamKeys covers the Teams pane: cursor up/down through the
// memberships table, set-as-active on `enter` (single press; the
// cursor's deliberate placement makes a separate confirm step
// redundant), and refresh-list (`r`). Pane switching is handled at
// the root.
type TeamKeys struct {
	App     AppKeys
	Up      key.Binding
	Down    key.Binding
	Set     key.Binding
	Create  key.Binding
	Refresh key.Binding
}

// NewTeamKeys returns the Teams pane bindings.
func NewTeamKeys(app AppKeys) TeamKeys {
	return TeamKeys{
		App: app,
		Up: key.NewBinding(
			key.WithKeys("up", "k"),
			key.WithHelp("↑/k", "up"),
		),
		Down: key.NewBinding(
			key.WithKeys("down", "j"),
			key.WithHelp("↓/j", "down"),
		),
		Set: key.NewBinding(
			key.WithKeys("enter"),
			key.WithHelp("enter", "set as active"),
		),
		Create: key.NewBinding(
			key.WithKeys("c"),
			key.WithHelp("c", "new team"),
		),
		Refresh: key.NewBinding(
			key.WithKeys("r"),
			key.WithHelp("r", "refresh"),
		),
	}
}

// ShortHelp implements help.KeyMap.
func (k TeamKeys) ShortHelp() []key.Binding {
	return []key.Binding{
		k.Up, k.Down, k.Set, k.Create, k.Refresh,
		k.App.ToggleHelp, k.App.Quit,
	}
}

// FullHelp implements help.KeyMap.
func (k TeamKeys) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.Up, k.Down, k.Set, k.Create, k.Refresh},
		{k.App.NextPane, k.App.PrevPane, k.App.GoMonitor, k.App.GoNewItem, k.App.GoTeam, k.App.GoConnection},
		{k.App.ToggleHelp, k.App.Quit},
	}
}

// TeamCreateKeys covers the create-team modal: move focus between the
// name field, ownership radio, and buttons; change the ownership choice;
// submit; cancel. Surfaced in the footer while the modal is open.
type TeamCreateKeys struct {
	App       AppKeys
	Field     key.Binding
	Ownership key.Binding
	Submit    key.Binding
	Cancel    key.Binding
	Quit      key.Binding
}

// NewTeamCreateKeys returns the create-modal bindings. The modal captures
// every key (including q, 1-4, [, ], ?) into the form, so quit is advertised
// as ctrl+c — the only key that still quits while the modal is open.
func NewTeamCreateKeys(app AppKeys) TeamCreateKeys {
	return TeamCreateKeys{
		App: app,
		Field: key.NewBinding(
			key.WithKeys("tab", "shift+tab", "up", "down"),
			key.WithHelp("tab/⇧tab", "move field"),
		),
		Ownership: key.NewBinding(
			key.WithKeys("left", "right"),
			key.WithHelp("←/→", "ownership"),
		),
		Submit: key.NewBinding(
			key.WithKeys("enter"),
			key.WithHelp("enter", "create"),
		),
		Cancel: key.NewBinding(
			key.WithKeys("esc"),
			key.WithHelp("esc", "cancel"),
		),
		Quit: key.NewBinding(
			key.WithKeys("ctrl+c"),
			key.WithHelp("ctrl+c", "quit"),
		),
	}
}

// ShortHelp implements help.KeyMap.
func (k TeamCreateKeys) ShortHelp() []key.Binding {
	return []key.Binding{
		k.Field, k.Ownership, k.Submit, k.Cancel, k.Quit,
	}
}

// FullHelp implements help.KeyMap.
func (k TeamCreateKeys) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.Field, k.Ownership, k.Submit, k.Cancel},
		{k.Quit},
	}
}

// ConnectionKeys covers the Connection pane: pane-local refresh of
// the external-services health checks plus the cross-pane app
// bindings.
type ConnectionKeys struct {
	App     AppKeys
	Refresh key.Binding
}

// NewConnectionKeys returns the Connection pane bindings.
func NewConnectionKeys(app AppKeys) ConnectionKeys {
	return ConnectionKeys{
		App: app,
		Refresh: key.NewBinding(
			key.WithKeys("r"),
			key.WithHelp("r", "refresh health checks"),
		),
	}
}

// ShortHelp implements help.KeyMap.
func (k ConnectionKeys) ShortHelp() []key.Binding {
	return []key.Binding{
		k.Refresh,
		k.App.NextPane, k.App.PrevPane,
		k.App.ToggleHelp, k.App.Quit,
	}
}

// FullHelp implements help.KeyMap.
func (k ConnectionKeys) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.Refresh},
		{k.App.NextPane, k.App.PrevPane, k.App.GoMonitor, k.App.GoNewItem, k.App.GoTeam, k.App.GoConnection},
		{k.App.ToggleHelp, k.App.Quit},
	}
}
