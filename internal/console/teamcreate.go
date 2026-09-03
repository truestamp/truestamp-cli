// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: MIT

package console

import (
	"context"
	"errors"
	"strings"
	"time"

	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	lipgloss "charm.land/lipgloss/v2"
	"github.com/truestamp/truestamp-cli/internal/redact"
	"github.com/truestamp/truestamp-cli/internal/teams"
	"github.com/truestamp/truestamp-cli/internal/ui"
)

// teamCreateFocus enumerates the focusable controls in the create modal, in
// tab order. One ordered enum lets tab/shift-tab share a single source of
// truth with the renderer's focus highlight.
type teamCreateFocus int

const (
	focusName teamCreateFocus = iota
	focusOwnership
	focusCreate
	focusCancel
	focusCount // sentinel: number of focusable controls
)

// teamCreateModel is the in-pane "create a new team" modal: a bubbles
// textinput for the name, a vertical ownership-model radio, and Create /
// Cancel buttons, driven entirely by the keyboard (tab / shift-tab / arrows
// / enter / esc). It owns no network, the parent teamModel fires the create
// call when the modal reports a submit, and the server is the authority on
// plan entitlement (there is no pre-flight check).
type teamCreateModel struct {
	name      textinput.Model
	ownership []string // ownership models offered, in display order
	ownIdx    int      // selected index into ownership

	focus      teamCreateFocus
	submitting bool
	errMsg     string
}

// newTeamCreateModel builds a fresh modal offering both ownership models;
// the server rejects an unentitled choice at create time with a clear error.
func newTeamCreateModel() *teamCreateModel {
	ti := textinput.New()
	ti.Placeholder = "e.g. Acme Engineering"
	ti.CharLimit = 200
	ti.Prompt = "> "
	ti.SetWidth(40)

	return &teamCreateModel{
		name:      ti,
		ownership: teams.OwnershipModels(),
		focus:     focusName,
	}
}

// focusInit focuses the name field and returns its cursor-blink command.
func (m *teamCreateModel) focusInit() tea.Cmd {
	m.focus = focusName
	return m.name.Focus()
}

// selectedOwnership returns the chosen ownership model wire value.
func (m *teamCreateModel) selectedOwnership() string {
	if m.ownIdx < 0 || m.ownIdx >= len(m.ownership) {
		return teams.OwnershipCreatorRetains
	}
	return m.ownership[m.ownIdx]
}

// teamCreateResult communicates the outcome of a key to the parent.
type teamCreateResult int

const (
	createPending teamCreateResult = iota // stay open
	createSubmit                          // user asked to create
	createCancel                          // user dismissed the modal
)

// update handles one message. It returns the parent-visible result plus a
// cmd (e.g. the textinput cursor blink). The parent owns the network call
// and the modal lifecycle; this only manages local focus + field state.
func (m *teamCreateModel) update(msg tea.Msg) (teamCreateResult, tea.Cmd) {
	if m.submitting {
		return createPending, nil
	}
	key, ok := msg.(tea.KeyPressMsg)
	if !ok {
		// Forward non-key messages (e.g. cursor blink ticks) to the input.
		var cmd tea.Cmd
		m.name, cmd = m.name.Update(msg)
		return createPending, cmd
	}

	switch key.String() {
	case "esc":
		return createCancel, nil
	case "tab", "shift+tab", "up", "down":
		return createPending, m.moveFocus(key.String())
	case "enter":
		if m.focus == focusCancel {
			return createCancel, nil
		}
		return createSubmit, nil
	case "left", "right", "h", "l":
		if m.focus == focusOwnership {
			m.cycleOwnership(key.String())
			return createPending, nil
		}
	}

	// Anything else: only the name field consumes it (text entry).
	if m.focus == focusName {
		var cmd tea.Cmd
		m.name, cmd = m.name.Update(msg)
		m.errMsg = ""
		return createPending, cmd
	}
	return createPending, nil
}

// moveFocus advances or rewinds focus and toggles the textinput focus state
// so the cursor only blinks while the name field is focused.
func (m *teamCreateModel) moveFocus(key string) tea.Cmd {
	prev := m.focus
	switch key {
	case "tab", "down":
		m.focus = (m.focus + 1) % focusCount
	case "shift+tab", "up":
		m.focus = (m.focus + focusCount - 1) % focusCount
	}
	return m.syncInputFocus(prev)
}

func (m *teamCreateModel) cycleOwnership(key string) {
	if len(m.ownership) < 2 {
		return
	}
	switch key {
	case "left", "h":
		m.ownIdx = (m.ownIdx + len(m.ownership) - 1) % len(m.ownership)
	default: // right, l
		m.ownIdx = (m.ownIdx + 1) % len(m.ownership)
	}
}

// focusControl points focus at a control. Returns the cursor-blink cmd when
// the name field gains focus.
func (m *teamCreateModel) focusControl(f teamCreateFocus) tea.Cmd {
	prev := m.focus
	m.focus = f
	return m.syncInputFocus(prev)
}

// syncInputFocus blurs/focuses the textinput so its cursor blinks only while
// the name field is the focused control. prev is the focus before the move.
func (m *teamCreateModel) syncInputFocus(prev teamCreateFocus) tea.Cmd {
	if prev == focusName && m.focus != focusName {
		m.name.Blur()
	}
	if m.focus == focusName && prev != focusName {
		return m.name.Focus()
	}
	return nil
}

// validate returns a non-empty error string when the form can't be
// submitted yet. Mirrors the server-side requirement of a non-blank name.
func (m *teamCreateModel) validate() string {
	if strings.TrimSpace(m.name.Value()) == "" {
		return "Team name is required."
	}
	return ""
}

// =====================================================================
// Rendering
// =====================================================================

func (m *teamCreateModel) render(width int) string {
	var b strings.Builder
	line := func(s string) { b.WriteString(s); b.WriteByte('\n') }

	line(teamCreateTitle.Render("Create a new team"))
	line("")

	// Name field, with a live "(required)" cue while it's blank.
	nameLabel := teamCreateLabel.Render("Name") + m.focusMark(focusName)
	if strings.TrimSpace(m.name.Value()) == "" {
		nameLabel += "  " + teamFaintStyle.Render("(required)")
	}
	line("  " + nameLabel)
	m.name.SetWidth(min(width-8, 48))
	line("  " + m.name.View())
	if m.errMsg != "" {
		line("  " + teamCreateErr.Render("✗ "+m.errMsg))
	}
	line("")

	// Ownership radio, one option per line, color-independent ●/○ glyphs.
	line("  " + teamCreateLabel.Render("Ownership model") + m.focusMark(focusOwnership))
	for i, model := range m.ownership {
		marker := "○"
		style := teamFaintStyle
		if i == m.ownIdx {
			marker = "●"
			style = teamCreateValue
		}
		line("  " + style.Render(marker+" "+teams.OwnershipLabel(model)))
	}
	if desc := teams.OwnershipDescription(m.selectedOwnership()); desc != "" {
		line("    " + teamFaintStyle.Render(desc))
	}
	line("")

	// Buttons. The focused one carries a ▸ marker so it stays distinct even
	// under --no-color (where reverse-video is stripped).
	line("  " + m.button("Create", m.focus == focusCreate) + "  " + m.button("Cancel", m.focus == focusCancel))
	if m.submitting {
		line("")
		line("  " + teamFaintStyle.Render("Creating…"))
	}
	return strings.TrimRight(b.String(), "\n")
}

// focusMark returns a small caret appended to a label when that control is
// focused, so keyboard users always see where they are (color-independent).
func (m *teamCreateModel) focusMark(f teamCreateFocus) string {
	if m.focus == f {
		return " " + teamCreateValue.Render("▸")
	}
	return ""
}

func (m *teamCreateModel) button(label string, focused bool) string {
	if focused {
		return teamBtnFocused.Render("▸[ " + label + " ]")
	}
	return teamBtnBlurred.Render(" [ " + label + " ]")
}

var (
	teamCreateTitle = lipgloss.NewStyle().Bold(true).Foreground(ui.Accent)
	teamCreateLabel = lipgloss.NewStyle().Bold(true).Foreground(ui.Label)
	teamCreateValue = lipgloss.NewStyle().Foreground(ui.Value)
	teamCreateErr   = lipgloss.NewStyle().Foreground(ui.Red)
	teamBtnFocused  = lipgloss.NewStyle().Bold(true).Reverse(true)
	teamBtnBlurred  = lipgloss.NewStyle().Foreground(ui.Value)
	teamCreateBtn   = lipgloss.NewStyle().Bold(true).Foreground(ui.Accent)
)

// =====================================================================
// teamModel integration: lifecycle + network command
// =====================================================================

// openCreate builds and focuses the create modal.
func (m *teamModel) openCreate() tea.Cmd {
	m.notice = ""
	m.create = newTeamCreateModel()
	return m.create.focusInit()
}

// updateCreating is the modal sub-loop. It owns input while the modal is
// open; create-lifecycle + membership messages are handled so state
// underneath stays consistent for when the modal closes.
func (m *teamModel) updateCreating(msg tea.Msg) (*teamModel, tea.Cmd) {
	switch msg := msg.(type) {
	case teamCreatedMsg:
		m.create = nil
		m.notice = "Created " + teamCreatedName(msg.Team) + "."
		m.noticeError = false
		return m, m.fetchMembershipsCmd()
	case teamCreateFailedMsg:
		if m.create != nil {
			m.create.submitting = false
			m.create.errMsg = friendlyCreateError(msg.Err)
		}
		return m, nil
	case teamMembershipsMsg:
		if msg.Err == nil {
			m.memberships = msg.Memberships
			m.state = teamPaneReady
		}
		return m, nil
	default:
		res, cmd := m.create.update(msg)
		return m.applyModalResult(res, cmd)
	}
}

// applyModalResult turns a modal key outcome into the next state + command:
// cancel closes, submit validates then fires the create call.
func (m *teamModel) applyModalResult(res teamCreateResult, cmd tea.Cmd) (*teamModel, tea.Cmd) {
	switch res {
	case createCancel:
		m.create = nil
		return m, nil
	case createSubmit:
		if errMsg := m.create.validate(); errMsg != "" {
			m.create.errMsg = errMsg
			return m, m.create.focusControl(focusName)
		}
		m.create.submitting = true
		m.create.errMsg = ""
		return m, m.createTeamCmd(m.create.name.Value(), m.create.selectedOwnership())
	}
	return m, cmd
}

// createTeamCmd performs the POST /teams call off the UI goroutine.
func (m *teamModel) createTeamCmd(name, ownership string) tea.Cmd {
	apiURL := m.apiURL
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		t, err := teams.CreateTeam(ctx, teams.Config{APIURL: apiURL}, name, ownership)
		if err != nil {
			return teamCreateFailedMsg{Err: err}
		}
		return teamCreatedMsg{Team: t}
	}
}

func teamCreatedName(t *teams.Team) string {
	if t == nil || t.Name == "" {
		return "team"
	}
	return t.Name
}

// friendlyCreateError maps a create error to a short modal message,
// preferring the two policy sentinels, then the (redacted) server detail,
// then the raw error. Redaction at this presentation boundary is defense in
// depth against a reflected credential reaching the screen.
func friendlyCreateError(err error) string {
	switch {
	case errors.Is(err, teams.ErrTeamLimitReached):
		return "Your plan's team limit is reached. Upgrade to add more."
	case errors.Is(err, teams.ErrOwnershipNotEntitled):
		return "That ownership model isn't available on your plan."
	case errors.Is(err, teams.ErrUnauthorized):
		return "Authentication failed, re-run `truestamp auth login`."
	}
	var ae *teams.APIError
	if errors.As(err, &ae) && ae.Detail != "" {
		return redact.String(ae.Detail)
	}
	return redact.String(err.Error())
}
