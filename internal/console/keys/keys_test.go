// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: MIT

package keys

import (
	"testing"

	"charm.land/bubbles/v2/help"
	"charm.land/bubbles/v2/key"
)

// TestKeyMapsImplementHelpKeyMap is a compile-time + runtime check
// that every pane's KeyMap satisfies bubbles/help.KeyMap. The footer
// component requires this interface — if we ever drop ShortHelp /
// FullHelp on a struct, this test catches it before runtime.
func TestKeyMapsImplementHelpKeyMap(t *testing.T) {
	t.Parallel()

	app := NewAppKeys()

	tests := []struct {
		name string
		km   help.KeyMap
	}{
		{"MonitorKeys", NewMonitorKeys(app)},
		{"NewItemKeys", NewNewItemKeys(app)},
		{"ConnectionKeys", NewConnectionKeys(app)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.km.ShortHelp(); len(got) == 0 {
				t.Errorf("%s.ShortHelp() returned empty slice", tt.name)
			}
			if got := tt.km.FullHelp(); len(got) == 0 {
				t.Errorf("%s.FullHelp() returned empty slice", tt.name)
			}
		})
	}
}

// TestAppKeysAreSurfacedInEveryPane confirms global bindings (Quit,
// pane-switch, help-toggle) appear in every pane's FullHelp. The help
// component renders FullHelp grouped by column; the test doesn't care
// about columns, only that the global keys are present somewhere so
// the user always sees how to leave the current pane.
func TestAppKeysAreSurfacedInEveryPane(t *testing.T) {
	t.Parallel()

	app := NewAppKeys()
	maps := []struct {
		name string
		km   help.KeyMap
	}{
		{"MonitorKeys", NewMonitorKeys(app)},
		{"NewItemKeys", NewNewItemKeys(app)},
		{"ConnectionKeys", NewConnectionKeys(app)},
	}

	for _, m := range maps {
		t.Run(m.name, func(t *testing.T) {
			groups := m.km.FullHelp()

			for _, want := range []key.Binding{app.Quit, app.NextPane, app.ToggleHelp} {
				if !containsBinding(groups, want) {
					t.Errorf("%s.FullHelp() does not surface %q", m.name, want.Help().Key)
				}
			}
		})
	}
}

// TestTabKeyIsClaimedByNewItemPaneNotApp verifies the bug fix from
// Phase 1: pane-switch bindings deliberately do NOT include "tab", so
// the New Item form's tab-between-fields binding can win when the
// form pane has focus.
func TestTabKeyIsClaimedByNewItemPaneNotApp(t *testing.T) {
	t.Parallel()

	app := NewAppKeys()
	for _, k := range app.NextPane.Keys() {
		if k == "tab" {
			t.Fatalf("AppKeys.NextPane includes %q — tab must be left for the New Item form", k)
		}
	}
	for _, k := range app.PrevPane.Keys() {
		if k == "shift+tab" {
			t.Fatalf("AppKeys.PrevPane includes %q — shift+tab must be left for the form", k)
		}
	}

	form := NewNewItemKeys(app)
	if !hasKey(form.NextField, "tab") {
		t.Error("NewItemKeys.NextField does not bind tab")
	}
	if !hasKey(form.PrevField, "shift+tab") {
		t.Error("NewItemKeys.PrevField does not bind shift+tab")
	}
}

// containsBinding looks for `want` in any group of `groups`, matching
// on the binding's help key (the rendered glyph). Two distinct
// key.Binding values built with the same WithHelp call compare equal
// via this; identity comparison would not.
func containsBinding(groups [][]key.Binding, want key.Binding) bool {
	wantKey := want.Help().Key
	for _, g := range groups {
		for _, b := range g {
			if b.Help().Key == wantKey {
				return true
			}
		}
	}
	return false
}

// hasKey checks whether a binding's Keys() slice contains s.
func hasKey(b key.Binding, s string) bool {
	for _, k := range b.Keys() {
		if k == s {
			return true
		}
	}
	return false
}
