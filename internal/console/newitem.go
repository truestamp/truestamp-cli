// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: MIT

package console

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"time"

	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	lipgloss "charm.land/lipgloss/v2"
	"github.com/truestamp/truestamp-cli/internal/console/events"
	"github.com/truestamp/truestamp-cli/internal/ui"
	"github.com/truestamp/truestamp-cli/internal/wschannel"
)

// newItemModel implements the form pane. It walks through four fields
// (name, description, hash, hash_type) and submits an items.create
// command. After submission it switches into "watching" mode and
// surfaces lifecycle pushes (item.created → item.updated → item.committed)
// for that specific item.
type newItemModel struct {
	client *wschannel.Client

	state newItemState

	// Form state.
	fields []textinput.Model
	cursor int

	// Created item state.
	created     *createdItem
	transitions []itemTransition

	formError string
}

type newItemState int

const (
	formEntering newItemState = iota
	formSubmitting
	formWatching
)

type createdItem struct {
	ID     string `json:"id"`
	State  string `json:"state"`
	Claims struct {
		Name        string `json:"name"`
		Description string `json:"description"`
		Hash        string `json:"hash"`
		HashType    string `json:"hash_type"`
	} `json:"claims"`
	ClaimsHash string `json:"claims_hash"`
	ItemHash   string `json:"item_hash"`
}

type itemTransition struct {
	when  time.Time
	kind  string
	state string
}

// itemCreateReplyMsg is dispatched after the items.create round-trip.
type itemCreateReplyMsg struct {
	item *createdItem
	err  error
}

// resetFormMsg requests resetting the pane back to a blank form.
type resetFormMsg struct{}

func newNewItemModel(client *wschannel.Client) *newItemModel {
	m := &newItemModel{
		client: client,
		state:  formEntering,
	}
	m.makeForm()
	return m
}

func (m *newItemModel) makeForm() {
	labels := []struct {
		placeholder string
		width       int
		charLimit   int
	}{
		{"name (required)", 60, 200},
		{"description (optional)", 60, 1000},
		{"hash (64 hex chars for sha256)", 70, 128},
		{"hash type (sha256)", 30, 32},
	}
	m.fields = make([]textinput.Model, len(labels))
	for i, l := range labels {
		ti := textinput.New()
		ti.Placeholder = l.placeholder
		ti.CharLimit = l.charLimit
		ti.SetWidth(l.width)
		m.fields[i] = ti
	}
	m.fields[3].SetValue("sha256")
	m.fields[0].Focus()
	m.cursor = 0
	m.formError = ""
}

func (m *newItemModel) Update(msg tea.Msg) (*newItemModel, tea.Cmd) {
	switch tmsg := msg.(type) {
	case tea.KeyPressMsg:
		key := tmsg.String()

		switch m.state {
		case formEntering:
			switch key {
			case "tab", "down":
				m.advanceCursor(1)
				return m, nil
			case "shift+tab", "up":
				m.advanceCursor(-1)
				return m, nil
			case "enter":
				if m.cursor < len(m.fields)-1 {
					m.advanceCursor(1)
					return m, nil
				}
				return m, m.submit()
			case "esc":
				// Clear form.
				m.makeForm()
				return m, nil
			}

		case formWatching:
			if key == "n" {
				m.created = nil
				m.transitions = nil
				m.makeForm()
				m.state = formEntering
				return m, nil
			}
		}

	case itemCreateReplyMsg:
		if tmsg.err != nil {
			m.state = formEntering
			m.formError = tmsg.err.Error()
			return m, nil
		}
		m.state = formWatching
		m.created = tmsg.item
		m.transitions = []itemTransition{
			{when: time.Now(), kind: "item.created", state: tmsg.item.State},
		}
		return m, nil

	case resetFormMsg:
		m.created = nil
		m.transitions = nil
		m.makeForm()
		m.state = formEntering
		return m, nil
	}

	if m.state == formEntering {
		var cmd tea.Cmd
		m.fields[m.cursor], cmd = m.fields[m.cursor].Update(msg)
		return m, cmd
	}
	return m, nil
}

func (m *newItemModel) advanceCursor(delta int) {
	m.fields[m.cursor].Blur()
	m.cursor = (m.cursor + delta + len(m.fields)) % len(m.fields)
	m.fields[m.cursor].Focus()
}

var hexHash = regexp.MustCompile(`^[0-9a-fA-F]+$`)

func (m *newItemModel) submit() tea.Cmd {
	name := strings.TrimSpace(m.fields[0].Value())
	description := strings.TrimSpace(m.fields[1].Value())
	hash := strings.ToLower(strings.TrimSpace(m.fields[2].Value()))
	hashType := strings.TrimSpace(m.fields[3].Value())

	if name == "" {
		m.formError = "name is required"
		return nil
	}
	if hashType == "" {
		hashType = "sha256"
	}
	if hash == "" || !hexHash.MatchString(hash) {
		m.formError = "hash must be a hex string"
		return nil
	}
	if hashType == "sha256" && len(hash) != 64 {
		m.formError = "sha256 hash must be 64 hex characters"
		return nil
	}

	m.formError = ""
	m.state = formSubmitting

	client := m.client
	payload := map[string]any{
		"name":        name,
		"description": description,
		"hash":        hash,
		"hash_type":   hashType,
		"watch":       true,
	}

	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		reply, err := client.Push(ctx, lobbyTopic, "items.create", payload)
		if err != nil {
			return itemCreateReplyMsg{err: err}
		}
		if reply.Status != "ok" {
			var e struct {
				Code    string `json:"code"`
				Message string `json:"message"`
			}
			_ = json.Unmarshal(reply.Response, &e)
			msg := e.Message
			if msg == "" {
				msg = string(reply.Response)
			}
			return itemCreateReplyMsg{err: fmt.Errorf("%s: %s", e.Code, msg)}
		}
		var resp struct {
			Item createdItem `json:"item"`
		}
		if err := json.Unmarshal(reply.Response, &resp); err != nil {
			return itemCreateReplyMsg{err: err}
		}
		return itemCreateReplyMsg{item: &resp.Item}
	}
}

func (m *newItemModel) handlePush(msg pushMsg) {
	if m.state != formWatching || m.created == nil {
		return
	}
	if msg.Event != "stream" {
		return
	}
	var p events.Push
	if err := json.Unmarshal(msg.Payload, &p); err != nil {
		return
	}
	if !strings.HasPrefix(p.Kind, "item.") {
		return
	}
	var d struct {
		ID    string `json:"id"`
		State string `json:"state"`
	}
	_ = json.Unmarshal(p.Data, &d)
	if d.ID != m.created.ID {
		return
	}
	m.transitions = append(m.transitions, itemTransition{
		when:  time.Now(),
		kind:  p.Kind,
		state: d.State,
	})
	if len(m.transitions) > maxTransitions {
		m.transitions = m.transitions[len(m.transitions)-maxTransitions:]
	}
}

// maxTransitions caps the lifecycle log for a watched item so a long
// session followed by repeated `item.updated` events (e.g. tag edits)
// can't grow without bound. Real lifecycles are 3-5 entries.
const maxTransitions = 100

// =====================================================================
// Rendering
// =====================================================================

var (
	formLabelStyle  = lipgloss.NewStyle().Bold(true)
	formActiveStyle = lipgloss.NewStyle().Foreground(ui.Blue)
	formErrorStyle  = lipgloss.NewStyle().Foreground(ui.Red)
	cardBoxStyle    = lipgloss.NewStyle().BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(ui.Green).Padding(1, 2)
	transitionStyle = lipgloss.NewStyle().Foreground(ui.Yellow)
	stateStyle      = lipgloss.NewStyle().Bold(true).Foreground(ui.Blue)
)

func (m *newItemModel) View(width, height int) string {
	switch m.state {
	case formEntering, formSubmitting:
		return m.renderForm(width, height)
	case formWatching:
		return m.renderWatching(width, height)
	}
	return ""
}

func (m *newItemModel) renderForm(width, height int) string {
	var sb strings.Builder
	sb.WriteString(formLabelStyle.Render("Create a new timestamped item") + "\n\n")

	for i, ti := range m.fields {
		label := []string{"Name:", "Description:", "Hash:", "Hash type:"}[i]
		if i == m.cursor {
			label = formActiveStyle.Render(label)
		}
		sb.WriteString(label + "\n")
		sb.WriteString("  " + ti.View() + "\n\n")
	}

	if m.formError != "" {
		sb.WriteString(formErrorStyle.Render("✗ "+m.formError) + "\n\n")
	}

	if m.state == formSubmitting {
		sb.WriteString(formLabelStyle.Render("submitting…") + "\n")
	} else {
		sb.WriteString(lipgloss.NewStyle().Faint(true).Render(
			"tab: next field   shift+tab: previous   enter: submit   esc: clear"))
	}

	return paneStyle(width, height).Render(sb.String())
}

func (m *newItemModel) renderWatching(width, height int) string {
	if m.created == nil {
		return paneStyle(width, height).Render("(no item yet)")
	}

	var sb strings.Builder
	sb.WriteString(formLabelStyle.Render("Submitted") + "\n\n")

	currentState := m.created.State
	if len(m.transitions) > 0 {
		currentState = m.transitions[len(m.transitions)-1].state
	}

	card := fmt.Sprintf(
		"  ID            %s\n"+
			"  Name          %s\n"+
			"  State         %s\n"+
			"  Hash          %s\n"+
			"  Claims hash   %s\n"+
			"  Item hash     %s",
		m.created.ID,
		m.created.Claims.Name,
		stateStyle.Render(currentState),
		m.created.Claims.Hash,
		shortHash(m.created.ClaimsHash),
		shortHash(m.created.ItemHash),
	)
	sb.WriteString(cardBoxStyle.Render(card))
	sb.WriteString("\n\n")

	sb.WriteString(formLabelStyle.Render("Lifecycle (live)") + "\n\n")
	for _, t := range m.transitions {
		sb.WriteString(fmt.Sprintf("  %s  %s  %s\n",
			t.when.Format("15:04:05.000"),
			transitionStyle.Render(t.kind),
			t.state))
	}
	if len(m.transitions) == 1 {
		sb.WriteString("\n  ")
		sb.WriteString(lipgloss.NewStyle().Faint(true).Render(
			"waiting for state transitions… (will arrive when the item is committed to a block)"))
	}

	sb.WriteString("\n\n")
	sb.WriteString(lipgloss.NewStyle().Faint(true).Render("press n: new item   tab: switch pane"))

	return paneStyle(width, height).Render(sb.String())
}
