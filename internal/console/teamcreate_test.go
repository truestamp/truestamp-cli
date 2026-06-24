// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: MIT

package console

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/truestamp/truestamp-cli/internal/teams"
)

func runeKey(r rune) tea.KeyPressMsg    { return tea.KeyPressMsg{Code: r, Text: string(r)} }
func specialKey(c rune) tea.KeyPressMsg { return tea.KeyPressMsg{Code: c} }

func newTestTeamModel() *teamModel {
	return newTeamModel("http://example.test/api/json", &activeScope{}, nil)
}

// --- Modal logic (direct method calls — version-robust) -----------------

func TestTeamCreateModal_DefaultsAndOwnershipOptions(t *testing.T) {
	m := newTeamCreateModel()
	if got := m.selectedOwnership(); got != teams.OwnershipCreatorRetains {
		t.Errorf("default ownership = %q, want creator_retains", got)
	}
	// Both models are offered; the server is the authority on entitlement.
	if len(m.ownership) != 2 {
		t.Errorf("ownership options = %v, want both", m.ownership)
	}
	if m.focus != focusName {
		t.Errorf("initial focus = %d, want focusName", m.focus)
	}
}

func TestTeamCreateModal_FocusCycle(t *testing.T) {
	m := newTeamCreateModel()
	m.moveFocus("tab")
	if m.focus != focusOwnership {
		t.Errorf("after tab focus = %d, want focusOwnership", m.focus)
	}
	m.moveFocus("shift+tab")
	if m.focus != focusName {
		t.Errorf("after shift+tab focus = %d, want focusName", m.focus)
	}
	// Wrap-around backwards from name -> cancel.
	m.moveFocus("shift+tab")
	if m.focus != focusCancel {
		t.Errorf("wrap focus = %d, want focusCancel", m.focus)
	}
}

func TestTeamCreateModal_OwnershipCycle(t *testing.T) {
	m := newTeamCreateModel()
	m.cycleOwnership("right")
	if m.ownIdx != 1 {
		t.Errorf("after right ownIdx = %d, want 1", m.ownIdx)
	}
	m.cycleOwnership("right")
	if m.ownIdx != 0 {
		t.Errorf("wrap ownIdx = %d, want 0", m.ownIdx)
	}
	m.cycleOwnership("left")
	if m.ownIdx != 1 {
		t.Errorf("after left ownIdx = %d, want 1", m.ownIdx)
	}
}

func TestTeamCreateModal_Validate(t *testing.T) {
	m := newTeamCreateModel()
	if m.validate() == "" {
		t.Error("blank name should be invalid")
	}
	m.name.SetValue("   ")
	if m.validate() == "" {
		t.Error("whitespace-only name should be invalid")
	}
	m.name.SetValue("Acme")
	if msg := m.validate(); msg != "" {
		t.Errorf("valid name reported error: %q", msg)
	}
}

func TestTeamCreateModal_UpdateDispatch(t *testing.T) {
	m := newTeamCreateModel()
	if res, _ := m.update(specialKey(tea.KeyEscape)); res != createCancel {
		t.Errorf("esc = %v, want cancel", res)
	}
	if res, _ := m.update(specialKey(tea.KeyEnter)); res != createSubmit {
		t.Errorf("enter = %v, want submit", res)
	}
	m.focus = focusCancel
	if res, _ := m.update(specialKey(tea.KeyEnter)); res != createCancel {
		t.Errorf("enter on Cancel = %v, want cancel", res)
	}
}

func TestTeamCreateModal_TypingIntoName(t *testing.T) {
	m := newTeamCreateModel()
	m.focusInit()
	for _, r := range "Acme" {
		m.update(runeKey(r))
	}
	if got := m.name.Value(); got != "Acme" {
		t.Errorf("name value = %q, want \"Acme\"", got)
	}
}

func TestTeamCreateModal_RenderShowsFields(t *testing.T) {
	m := newTeamCreateModel()
	body := m.render(60)
	for _, want := range []string{"Create a new team", "Name", "(required)", "Ownership model", "Create", "Cancel"} {
		if !strings.Contains(body, want) {
			t.Errorf("render missing %q:\n%s", want, body)
		}
	}
}

// --- teamModel integration ----------------------------------------------

func TestTeamModel_OpenCreateViaKey(t *testing.T) {
	m := newTestTeamModel()
	if m.create != nil {
		t.Fatal("modal should start closed")
	}
	m, _ = m.Update(runeKey('c'))
	if m.create == nil {
		t.Fatal("`c` should open the create modal")
	}
}

func TestTeamModel_ApplyModalResult_SubmitValidates(t *testing.T) {
	m := newTestTeamModel()
	m, _ = m.Update(runeKey('c'))

	m, _ = m.applyModalResult(createSubmit, nil)
	if m.create == nil {
		t.Fatal("invalid submit should keep the modal open")
	}
	if m.create.errMsg == "" {
		t.Error("invalid submit should set an error message")
	}
	if m.create.submitting {
		t.Error("invalid submit must not enter submitting state")
	}

	m.create.name.SetValue("Acme")
	var cmd tea.Cmd
	m, cmd = m.applyModalResult(createSubmit, nil)
	if !m.create.submitting {
		t.Error("valid submit should set submitting")
	}
	if cmd == nil {
		t.Error("valid submit should return the create command")
	}
}

func TestTeamModel_ApplyModalResult_Cancel(t *testing.T) {
	m := newTestTeamModel()
	m, _ = m.Update(runeKey('c'))
	m, _ = m.applyModalResult(createCancel, nil)
	if m.create != nil {
		t.Error("cancel should close the modal")
	}
}

func TestTeamModel_CreatedClosesAndNotifies(t *testing.T) {
	m := newTestTeamModel()
	m, _ = m.Update(runeKey('c'))
	m, cmd := m.updateCreating(teamCreatedMsg{Team: &teams.Team{ID: "x", Name: "Acme"}})
	if m.create != nil {
		t.Error("created should close the modal")
	}
	if !strings.Contains(m.notice, "Acme") {
		t.Errorf("notice = %q, want it to mention the new team", m.notice)
	}
	if cmd == nil {
		t.Error("created should trigger a membership refresh")
	}
}

// teamCreatedMsg arriving while the modal is already closed must still be
// honored (the "completes if you tab away" contract).
func TestTeamModel_CreatedWhileModalClosed(t *testing.T) {
	m := newTestTeamModel()
	m, cmd := m.Update(teamCreatedMsg{Team: &teams.Team{ID: "x", Name: "Beta"}})
	if !strings.Contains(m.notice, "Beta") {
		t.Errorf("notice = %q, want it to confirm the team even with the modal closed", m.notice)
	}
	if cmd == nil {
		t.Error("a closed-modal created message should still refresh memberships")
	}
}

func TestTeamModel_CreateFailedShowsFriendlyError(t *testing.T) {
	m := newTestTeamModel()
	m, _ = m.Update(runeKey('c'))
	m.create.submitting = true
	m, _ = m.updateCreating(teamCreateFailedMsg{Err: teams.ErrTeamLimitReached})
	if m.create == nil {
		t.Fatal("failure should keep the modal open to retry")
	}
	if m.create.submitting {
		t.Error("failure should clear submitting")
	}
	if !strings.Contains(strings.ToLower(m.create.errMsg), "limit") {
		t.Errorf("errMsg = %q, want a plan-limit message", m.create.errMsg)
	}
}

func TestFriendlyCreateError(t *testing.T) {
	cases := []struct {
		err  error
		want string
	}{
		{teams.ErrTeamLimitReached, "limit"},
		{teams.ErrOwnershipNotEntitled, "ownership"},
		{teams.ErrUnauthorized, "login"},
	}
	for _, tc := range cases {
		if got := friendlyCreateError(tc.err); !strings.Contains(strings.ToLower(got), tc.want) {
			t.Errorf("friendlyCreateError(%v) = %q, want substring %q", tc.err, got, tc.want)
		}
	}
	// Redaction: a reflected credential in an API detail must be scrubbed.
	leaky := &teams.APIError{Detail: "boom api_key=SECRETKEY12345 boom"}
	if strings.Contains(friendlyCreateError(leaky), "SECRETKEY12345") {
		t.Error("friendlyCreateError leaked a secret from the API detail")
	}
}
