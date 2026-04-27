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

	"charm.land/huh/v2"
	tea "charm.land/bubbletea/v2"
	lipgloss "charm.land/lipgloss/v2"
	"github.com/truestamp/truestamp-cli/internal/console/events"
	"github.com/truestamp/truestamp-cli/internal/ui"
	"github.com/truestamp/truestamp-cli/internal/wschannel"
)

// hashTypeOption describes one supported `hash_type` value the
// server accepts on items.create. Mirrors lib/truestamp/hash.ex's
// @hash_types map on the v2 backend — keep in sync if the server
// adds or drops algorithms.
//
// Ordering: secure-modern first (recommended defaults at the top),
// then secure-legacy SHA-3 + BLAKE families, then insecure legacy
// algorithms at the bottom. The user picks via huh.NewSelect; the
// list order is what they see in the dropdown.
type hashTypeOption struct {
	Value  string // wire value sent to the server
	Label  string // human-readable Select label
	HexLen int    // exact length the hash field must match (2 × byte size)
	Secure bool   // false flags md5 / sha1; UI may surface a warning
}

var hashTypeOptions = []hashTypeOption{
	{Value: "sha256", Label: "SHA-256 (64 hex chars) — recommended", HexLen: 64, Secure: true},
	{Value: "sha512", Label: "SHA-512 (128 hex chars)", HexLen: 128, Secure: true},
	{Value: "sha384", Label: "SHA-384 (96 hex chars)", HexLen: 96, Secure: true},
	{Value: "sha224", Label: "SHA-224 (56 hex chars)", HexLen: 56, Secure: true},
	{Value: "sha3_256", Label: "SHA3-256 (64 hex chars)", HexLen: 64, Secure: true},
	{Value: "sha3_512", Label: "SHA3-512 (128 hex chars)", HexLen: 128, Secure: true},
	{Value: "sha3_384", Label: "SHA3-384 (96 hex chars)", HexLen: 96, Secure: true},
	{Value: "sha3_224", Label: "SHA3-224 (56 hex chars)", HexLen: 56, Secure: true},
	{Value: "blake2b", Label: "BLAKE2b (128 hex chars)", HexLen: 128, Secure: true},
	{Value: "blake2s", Label: "BLAKE2s (64 hex chars)", HexLen: 64, Secure: true},
	{Value: "sha1", Label: "SHA-1 (40 hex chars) — legacy / insecure", HexLen: 40, Secure: false},
	{Value: "md5", Label: "MD5 (32 hex chars) — legacy / insecure", HexLen: 32, Secure: false},
}

// lookupHashType resolves a wire value to its canonical option.
// Returns nil for unknown types so the caller can decide whether
// to fail or fall through.
func lookupHashType(value string) *hashTypeOption {
	for i := range hashTypeOptions {
		if hashTypeOptions[i].Value == value {
			return &hashTypeOptions[i]
		}
	}
	return nil
}

// newItemModel implements the form pane. The form fields are owned by
// a huh.Form so tab/shift-tab navigation, validators, and field
// chrome are all standard charm widgets — no hand-rolled textinput
// management.
//
// After submission the pane switches into "watching" mode and surfaces
// lifecycle pushes (item.created → item.updated → item.committed) for
// the just-created item.
type newItemModel struct {
	client *wschannel.Client

	state newItemState
	form  *huh.Form

	// Form values bound to the huh.Form fields. Filled by the form's
	// own bindings (Value(&...)).
	formName        string
	formDescription string
	formHash        string
	formHashType    string

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
		client:       client,
		state:        formEntering,
		formHashType: "sha256",
	}
	m.form = m.makeForm()
	return m
}

// makeForm constructs a fresh huh.Form bound to the model's value
// fields. Validators are attached so huh renders error text inline
// at the offending field and keeps focus there until it's resolved
// — that's the bubbles/huh native UX.
//
// Field order matters. The cross-field dependency (Hash length
// depends on Hash type) is resolved by putting Hash type BEFORE
// Hash: the user picks an algorithm first, then provides a digest
// for it. If they change their mind about the algorithm they can
// shift+tab back to Hash type — Prev navigation is never blocked
// by huh, only Next/Submit is, so the form never deadlocks the way
// it would if Hash came before Hash type.
func (m *newItemModel) makeForm() *huh.Form {
	form := huh.NewForm(
		huh.NewGroup(
			huh.NewInput().
				Title("Name").
				Description("Required. Free-form label, ≤ 200 chars.").
				CharLimit(200).
				Validate(requiredString("name")).
				Value(&m.formName),

			huh.NewText().
				Title("Description").
				Description("Optional context for the item.").
				CharLimit(1000).
				Value(&m.formDescription),

			huh.NewSelect[string]().
				Title("Hash type").
				Description("Selects which algorithm produced the hash; sets the expected hex length.").
				Options(hashTypeSelectOptions()...).
				Value(&m.formHashType),

			huh.NewInput().
				Title("Hash").
				Description("Hex digest of the data being timestamped. Must be even-length hex of the size matching the chosen Hash type.").
				CharLimit(128).
				Validate(validateHash(&m.formHashType)).
				Value(&m.formHash),
		),
	).
		WithShowHelp(false). // we render help in the page footer
		WithShowErrors(true)
	return form
}

// hashTypeSelectOptions converts the canonical hashTypeOptions slice
// into huh.Option values for the Select field. Built once per form
// constructor — cheap, but kept as a helper to keep makeForm readable.
func hashTypeSelectOptions() []huh.Option[string] {
	opts := make([]huh.Option[string], len(hashTypeOptions))
	for i, h := range hashTypeOptions {
		opts[i] = huh.NewOption(h.Label, h.Value)
	}
	return opts
}

func requiredString(name string) func(string) error {
	return func(s string) error {
		if strings.TrimSpace(s) == "" {
			return fmt.Errorf("%s is required", name)
		}
		return nil
	}
}

// hexHash matches a non-empty hex string of EVEN length. The pair
// match (`{2}+`) implicitly rejects single-char inputs and odd-
// length strings (which can't represent a whole-byte digest) even
// when the hashType is unknown — defense in depth, since downstream
// table-driven length checking is skipped for unknown algorithms.
var hexHash = regexp.MustCompile(`^([0-9a-fA-F]{2})+$`)

// validateHash enforces three invariants. All three are mandatory;
// none is skipped on any input path.
//
//  1. non-empty
//  2. hex-only AND even length (one or more `[0-9a-fA-F]{2}` pairs)
//  3. exact byte length matching the chosen hash type's HexLen
//
// hashType is always one of hashTypeOptions when the form is driven
// through the UI Select. An empty or unknown hashType is rejected
// loudly rather than skipped — defense in depth against any future
// caller that bypasses the Select. There is no "soft-fail" path:
// if any of the three checks fails, the validator returns an error.
func validateHash(hashType *string) func(string) error {
	return func(s string) error {
		s = strings.ToLower(strings.TrimSpace(s))
		if s == "" {
			return fmt.Errorf("hash is required")
		}
		if !hexHash.MatchString(s) {
			return fmt.Errorf("hash must be an even-length hex string")
		}
		if hashType == nil || *hashType == "" {
			return fmt.Errorf("hash type is required")
		}
		entry := lookupHashType(*hashType)
		if entry == nil {
			return fmt.Errorf("unknown hash type %q", *hashType)
		}
		if len(s) != entry.HexLen {
			return fmt.Errorf("%s hash must be %d hex characters (got %d)",
				entry.Value, entry.HexLen, len(s))
		}
		return nil
	}
}

func (m *newItemModel) Update(msg tea.Msg) (*newItemModel, tea.Cmd) {
	switch tmsg := msg.(type) {
	case itemCreateReplyMsg:
		if tmsg.err != nil {
			m.state = formEntering
			m.formError = tmsg.err.Error()
			m.form = m.makeForm()
			return m, m.form.Init()
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
		m.formName = ""
		m.formDescription = ""
		m.formHash = ""
		m.formHashType = "sha256"
		m.formError = ""
		m.state = formEntering
		m.form = m.makeForm()
		return m, m.form.Init()

	case tea.KeyPressMsg:
		// Watching mode: `n` resets back to a fresh form.
		if m.state == formWatching && tmsg.String() == "n" {
			return m, func() tea.Msg { return resetFormMsg{} }
		}
		// Entering mode: esc clears the form. We intercept before
		// huh sees the key — huh's Quit binding (ctrl+c) sets
		// StateAborted internally, but esc is otherwise a no-op
		// inside huh which made the previously-advertised "esc:
		// clear" footer hint a lie. Convert it into a real reset.
		if m.state == formEntering && tmsg.String() == "esc" {
			return m, func() tea.Msg { return resetFormMsg{} }
		}
	}

	if m.state != formEntering {
		return m, nil
	}

	// Delegate everything else to the huh.Form. It handles
	// tab/shift-tab between fields, enter to submit, esc to abort,
	// validators, and inline error rendering.
	form, cmd := m.form.Update(msg)
	if f, ok := form.(*huh.Form); ok {
		m.form = f
	}

	if m.form.State == huh.StateCompleted {
		// huh's per-field validators ran on every Next/Submit. If we
		// reach StateCompleted, every field passed its own validator
		// — no model-level re-run needed.
		m.formError = ""
		m.state = formSubmitting
		return m, m.submit()
	}

	return m, cmd
}

func (m *newItemModel) submit() tea.Cmd {
	client := m.client
	payload := map[string]any{
		"name":        strings.TrimSpace(m.formName),
		"description": strings.TrimSpace(m.formDescription),
		"hash":        strings.ToLower(strings.TrimSpace(m.formHash)),
		"hash_type":   m.formHashType,
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
	formTitleStyle  = lipgloss.NewStyle().Bold(true).Foreground(ui.Accent)
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
	sb.WriteString(formTitleStyle.Render("Create a new timestamped item") + "\n\n")

	// huh's own rendering owns the field chrome — labels, focus,
	// inline errors, prev/next field affordances. We just give it a
	// width hint and place its output beneath the page-level title.
	m.form.WithWidth(min(width-4, 80))
	sb.WriteString(m.form.View())

	if m.formError != "" {
		sb.WriteString("\n")
		sb.WriteString(formErrorStyle.Render("✗ " + m.formError))
		sb.WriteString("\n")
	}

	if m.state == formSubmitting {
		sb.WriteString("\n" + lipgloss.NewStyle().Bold(true).Render("submitting…"))
	}

	return paneStyle(width, height).Render(sb.String())
}

func (m *newItemModel) renderWatching(width, height int) string {
	if m.created == nil {
		return paneStyle(width, height).Render("(no item yet)")
	}

	currentState := m.created.State
	if len(m.transitions) > 0 {
		currentState = m.transitions[len(m.transitions)-1].state
	}

	// Title carries the item name + state inline so the user
	// doesn't have to scan into the card to know what they just did.
	// Green checkmark prefix communicates "submitted successfully"
	// at a glance.
	title := lipgloss.NewStyle().
		Bold(true).
		Foreground(ui.Green).
		Render("✓ Submitted")
	if name := strings.TrimSpace(m.created.Claims.Name); name != "" {
		title += "  " + lipgloss.NewStyle().Bold(true).Render(name)
	}

	// Card width sized to fit comfortably inside the body area.
	// The lipgloss Border style adds 2 chars (one each side); pad
	// + label column take 2 + 14; the rest is value column where
	// long fields wrap.
	cardOuterWidth := min(width-4, 80)
	if cardOuterWidth < 30 {
		cardOuterWidth = 30
	}
	valueWidth := cardOuterWidth - 2 - 2 - 14 // border + indent + label
	if valueWidth < 10 {
		valueWidth = 10
	}

	fields := []cardField{
		{"ID", m.created.ID},
		{"Name", m.created.Claims.Name},
	}
	if desc := strings.TrimSpace(m.created.Claims.Description); desc != "" {
		fields = append(fields, cardField{"Description", desc})
	}
	fields = append(fields, cardField{"State", stateStyle.Render(currentState)})
	fields = append(fields, cardField{"Hash", m.created.Claims.Hash})
	fields = append(fields, cardField{"Item hash", m.created.ItemHash})

	cardBody := renderCardFields(fields, valueWidth)

	var sb strings.Builder
	sb.WriteString(title + "\n\n")
	sb.WriteString(cardBoxStyle.Width(cardOuterWidth).Render(cardBody))
	sb.WriteString("\n\n")

	sb.WriteString(formTitleStyle.Render("Lifecycle (live)") + "\n\n")
	for _, t := range m.transitions {
		fmt.Fprintf(&sb, "  %s  %s  %s\n",
			t.when.Format("15:04:05.000"),
			transitionStyle.Render(t.kind),
			t.state)
	}
	if len(m.transitions) == 1 {
		sb.WriteString("\n  ")
		sb.WriteString(lipgloss.NewStyle().Faint(true).Render(
			"waiting for state transitions… (will arrive when the item is committed to a block)"))
	}

	// Note: the bottom hint ("press n: new item   tab: switch pane")
	// is intentionally gone. The footer's keymap is state-aware and
	// the help component renders the right bindings for this state
	// — duplicating them in the body is noise.
	return paneStyle(width, height).Render(sb.String())
}

// cardField is a single labelled row in the watching-mode card.
type cardField struct {
	label string
	value string
}

// renderCardFields lays out a labelled-value list with wrapping in
// the value column. Long values (e.g. a multi-line description, a
// 64-hex hash that doesn't fit at narrow widths) break across lines
// and align under the value column on continuation rows.
func renderCardFields(fields []cardField, valueWidth int) string {
	var sb strings.Builder
	for i, f := range fields {
		wrappedValue := lipgloss.NewStyle().Width(valueWidth).Render(f.value)
		lines := strings.Split(wrappedValue, "\n")
		fmt.Fprintf(&sb, "%-13s %s", f.label, lines[0])
		for _, cont := range lines[1:] {
			fmt.Fprintf(&sb, "\n%-13s %s", "", cont)
		}
		if i < len(fields)-1 {
			sb.WriteByte('\n')
		}
	}
	return sb.String()
}
