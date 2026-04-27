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
// @hash_types map on the v2 backend — Display matches the server's
// `name` field exactly so the same canonical string ("SHA-256",
// "BLAKE2b", "MD5", …) appears everywhere this hash type is shown:
// the form's algorithm picker, the watching-screen summary, and any
// future surfaces. Keep in sync with the server if names change.
//
// Ordering: secure-modern first (recommended defaults at the top),
// then secure-legacy SHA-3 + BLAKE families, then insecure legacy
// algorithms at the bottom. The user picks via huh.NewSelect; the
// list order is what they see in the dropdown.
type hashTypeOption struct {
	Value   string // wire value sent to the server (e.g. "sha256")
	Display string // canonical display name (e.g. "SHA-256")
	HexLen  int    // exact length the hash field must match (2 × byte size)
	Secure  bool   // false flags md5 / sha1
	Note    string // optional dropdown suffix ("recommended", "legacy / insecure")
}

var hashTypeOptions = []hashTypeOption{
	{Value: "sha256", Display: "SHA-256", HexLen: 64, Secure: true, Note: "recommended"},
	{Value: "sha512", Display: "SHA-512", HexLen: 128, Secure: true},
	{Value: "sha384", Display: "SHA-384", HexLen: 96, Secure: true},
	{Value: "sha224", Display: "SHA-224", HexLen: 56, Secure: true},
	{Value: "sha3_256", Display: "SHA3-256", HexLen: 64, Secure: true},
	{Value: "sha3_512", Display: "SHA3-512", HexLen: 128, Secure: true},
	{Value: "sha3_384", Display: "SHA3-384", HexLen: 96, Secure: true},
	{Value: "sha3_224", Display: "SHA3-224", HexLen: 56, Secure: true},
	{Value: "blake2b", Display: "BLAKE2b", HexLen: 128, Secure: true},
	{Value: "blake2s", Display: "BLAKE2s", HexLen: 64, Secure: true},
	{Value: "sha1", Display: "SHA-1", HexLen: 40, Secure: false, Note: "legacy / insecure"},
	{Value: "md5", Display: "MD5", HexLen: 32, Secure: false, Note: "legacy / insecure"},
}

// selectLabel renders the dropdown label for the form's Select
// widget: canonical Display name + the expected hex length + an
// optional Note ("recommended" / "legacy / insecure"). Derives
// from the canonical Display so the form picker and the watching
// screen agree on the same name with no possibility of drift.
func (h hashTypeOption) selectLabel() string {
	label := fmt.Sprintf("%s (%d hex chars)", h.Display, h.HexLen)
	if h.Note != "" {
		label += " — " + h.Note
	}
	return label
}

// displayHashType returns the canonical Display name for a wire
// value. Falls back to the wire value verbatim for unknown types
// — the lookup is best-effort so older items whose hash_type predates
// our table still render something rather than blanking the field.
func displayHashType(value string) string {
	if entry := lookupHashType(value); entry != nil {
		return entry.Display
	}
	return value
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
// into huh.Option values for the Select field. Labels are derived
// from the canonical Display name via selectLabel so the picker and
// the watching-screen summary stay in lockstep.
func hashTypeSelectOptions() []huh.Option[string] {
	opts := make([]huh.Option[string], len(hashTypeOptions))
	for i, h := range hashTypeOptions {
		opts[i] = huh.NewOption(h.selectLabel(), h.Value)
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
		// Watching mode: `n` or `esc` both return to a fresh form.
		// `n` is the discoverable mnemonic surfaced in the help
		// keymap; `esc` mirrors the form-mode "esc clears" semantics
		// so the user has one muscle-memory key for "go back / start
		// over" no matter which sub-state of the pane they're in.
		if m.state == formWatching {
			switch tmsg.String() {
			case "n", "esc":
				return m, func() tea.Msg { return resetFormMsg{} }
			}
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

	// Title carries the item name inline so the user doesn't have
	// to scan into the field list to know what they just submitted.
	// Green checkmark prefix communicates success at a glance.
	title := lipgloss.NewStyle().
		Bold(true).
		Foreground(ui.Green).
		Render("✓ Submitted")
	if name := strings.TrimSpace(m.created.Claims.Name); name != "" {
		title += "  " + lipgloss.NewStyle().Bold(true).Render(name)
	}

	// Wrap width for prose fields (Description). Hashes and IDs are
	// rendered single-line — wrapping a hex digest at character
	// boundaries breaks copy-paste and produces ugly mid-line
	// continuations, and the terminal's own soft-wrap handles
	// edge cases at narrow widths.
	const labelCol = 14
	descWrapWidth := width - labelCol - 4
	if descWrapWidth < 20 {
		descWrapWidth = 20
	}
	if descWrapWidth > 100 {
		descWrapWidth = 100
	}

	fields := []cardField{
		{label: "ID", value: m.created.ID},
		{label: "Name", value: m.created.Claims.Name},
	}
	if desc := strings.TrimSpace(m.created.Claims.Description); desc != "" {
		fields = append(fields, cardField{
			label: "Description",
			value: desc,
			wrap:  descWrapWidth,
		})
	}
	fields = append(fields,
		cardField{label: "State", value: stateStyle.Render(currentState)},
		cardField{label: "Hash type", value: displayHashType(m.created.Claims.HashType)},
		cardField{label: "Hash", value: m.created.Claims.Hash},
		cardField{label: "Item hash", value: m.created.ItemHash},
	)

	var sb strings.Builder
	sb.WriteString(title + "\n\n")
	sb.WriteString(renderCardFields(fields, labelCol))
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

	return paneStyle(width, height).Render(sb.String())
}

// cardField is a single labelled row in the watching-mode field list.
//
// `wrap` is the value-column wrap width in characters; zero means
// "render single-line, let the terminal handle any soft wrap". Used
// only for prose fields (Description) where breaking at word
// boundaries is helpful. Hash / ID values stay single-line so
// copy-paste yields the literal string the server sent.
type cardField struct {
	label string
	value string
	wrap  int
}

// renderCardFields lays out a labelled-value list with consistent
// label column alignment. No card border, no per-field indent —
// the label column itself provides the visual rhythm.
//
// For wrapped fields, continuation lines are aligned under the
// value column so the layout reads as columns rather than blocks.
func renderCardFields(fields []cardField, labelCol int) string {
	labelFmt := fmt.Sprintf("%%-%ds  ", labelCol)
	indentFmt := fmt.Sprintf("%%-%ds  ", labelCol) // same width, blank label
	var sb strings.Builder
	for i, f := range fields {
		value := f.value
		var lines []string
		if f.wrap > 0 {
			value = lipgloss.NewStyle().Width(f.wrap).Render(value)
			lines = strings.Split(value, "\n")
		} else {
			lines = []string{value}
		}
		fmt.Fprintf(&sb, labelFmt, f.label)
		sb.WriteString(lines[0])
		for _, cont := range lines[1:] {
			sb.WriteByte('\n')
			fmt.Fprintf(&sb, indentFmt, "")
			sb.WriteString(cont)
		}
		if i < len(fields)-1 {
			sb.WriteByte('\n')
		}
	}
	return sb.String()
}
