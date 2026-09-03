// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: MIT

package console

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"sort"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	lipgloss "charm.land/lipgloss/v2"
	ltable "charm.land/lipgloss/v2/table"
	"github.com/truestamp/truestamp-cli/internal/console/events"
	"github.com/truestamp/truestamp-cli/internal/ui"
	"github.com/truestamp/truestamp-cli/internal/wschannel"
)

// monitorModel is the live event waterfall pane: a left column of
// toggleable stream subscriptions and a right column of the most recent
// stream events received over the channel.
type monitorModel struct {
	client *wschannel.Client
	log    *slog.Logger

	streams []StreamMeta
	active  map[string]bool // local mirror of server-side subscription set
	cursor  int             // index into streams (left list)
	events  []events.Row    // bounded ring of recent events
	pending map[string]bool // streams with an inflight subscribe/unsubscribe

	// Waterfall selection + viewport state.
	//
	// selected is the absolute index in `events` of the highlighted row.
	// -1 means "no events yet". When the user is at the most recent
	// event (selected == len(events)-1), new events arriving auto-advance
	// selected so the cursor follows live data. Otherwise the cursor
	// stays pinned on a specific event while the buffer grows behind it.
	//
	// viewStart is the absolute index of the first event in the visible
	// window (in chronological order; reverseOrder only flips rendering).
	// It is maintained only on user navigation: when selected leaves the
	// visible range, viewStart slides just enough to bring it back in.
	selected  int
	viewStart int

	// reverseOrder swaps the visual order of the visible window:
	//   false → oldest at top, newest at bottom (chronological)
	//   true  → newest at top, oldest at bottom (reverse)
	// In live mode, the "edge" is whichever side shows the newest event.
	reverseOrder bool

	// focus determines which side receives ↑/↓ keys. ←/→ swap it. The
	// app-level `]` / `[` (and ctrl+tab / ctrl+shift+tab) still cycle the
	// outer panes (Monitor → New Item → Teams → Connection); focus is
	// internal to Monitor.
	focus monitorFocus

	// detailPanelHidden collapses the Detail Panel below the
	// waterfall. Default false (panel visible). `d` toggles it.
	detailPanelHidden bool
}

type monitorFocus int

const (
	focusList monitorFocus = iota
	focusWaterfall
)

// subscribeReplyMsg is dispatched after a subscribe/unsubscribe round-trip.
type subscribeReplyMsg struct {
	subscribe    bool
	streamID     string
	subscribed   []string
	unsubscribed []string
	rejected     []string
	err          error
}

func newMonitorModel(client *wschannel.Client, log *slog.Logger) *monitorModel {
	return &monitorModel{
		client:       client,
		log:          log,
		active:       make(map[string]bool),
		pending:      make(map[string]bool),
		reverseOrder: true,
		selected:     -1,
		// Default focus is the waterfall (Events). The stream list
		// on the left is rarely the user's first action; the events
		// stream is what they came here to watch. ←/h shifts focus
		// to the list when they want to toggle subscriptions.
		focus: focusWaterfall,
	}
}

// Event retention is time-windowed: we keep everything from the last
// `eventRetention` interval. A hard count ceiling guards against an
// adversarial / runaway emission rate (e.g. a stream regression that
// fires 10k events/sec); under normal operation the time window evicts
// before this limit is reached. At ~2k events/hr, 24h ≈ 48k events ≈
// ~10 MB resident.
const (
	eventRetention = 24 * time.Hour
	eventHardCap   = 100_000
)

func (m *monitorModel) Update(msg tea.Msg) (*monitorModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		key := msg.String()

		// Cross-side focus. h/l mirror left/right for vi-style nav.
		switch key {
		case "left", "h":
			m.focus = focusList
			return m, nil
		case "right", "l":
			m.focus = focusWaterfall
			return m, nil
		}

		// Reverse order works regardless of focus, it's the only
		// operation specific to the waterfall that we surface globally
		// so users don't have to remember to focus first.
		if key == "r" {
			m.reverseOrder = !m.reverseOrder
			return m, nil
		}

		// Detail Panel toggle: works regardless of focus so users
		// can collapse/expand from any state. Hidden state survives
		// pane switches, once a user opts out, we don't re-show
		// until they ask.
		if key == "d" {
			m.detailPanelHidden = !m.detailPanelHidden
			return m, nil
		}

		switch m.focus {
		case focusList:
			switch key {
			case "up", "k":
				if m.cursor > 0 {
					m.cursor--
				}
			case "down", "j":
				if m.cursor < len(m.streams)-1 {
					m.cursor++
				}
			case " ", "space":
				return m, m.toggleCursor()
			}

		case focusWaterfall:
			// Visual ↑ on screen always means "newer" in reverse mode and
			// "older" in chronological mode. We use a sign that flips with
			// reverseOrder so callers below can read in screen-up vs
			// screen-down terms without knowing which order is active.
			screenUpDelta := -1
			screenDnDelta := 1
			if m.reverseOrder {
				screenUpDelta, screenDnDelta = 1, -1
			}

			switch key {
			case "up", "k":
				m.moveSelected(screenUpDelta)
			case "down", "j":
				m.moveSelected(screenDnDelta)
			case "pgup", "K":
				m.moveSelected(screenUpDelta * pageStep)
			case "pgdn", "J":
				m.moveSelected(screenDnDelta * pageStep)
			case "home", "g":
				// "Top of screen", newest in reverse, oldest in chrono.
				if m.reverseOrder {
					m.setSelected(len(m.events) - 1)
				} else {
					m.setSelected(0)
				}
			case "end", "G":
				// "Bottom of screen", oldest in reverse, newest in chrono.
				if m.reverseOrder {
					m.setSelected(0)
				} else {
					m.setSelected(len(m.events) - 1)
				}
			}
		}

	case subscribeReplyMsg:
		delete(m.pending, msg.streamID)
		if msg.err != nil {
			// Roll back: server rejected the change.
			if msg.subscribe {
				delete(m.active, msg.streamID)
			} else {
				m.active[msg.streamID] = true
			}
			return m, nil
		}
		// Server is the source of truth, mirror the canonical set.
		if msg.subscribe {
			for _, id := range msg.subscribed {
				m.active[id] = true
			}
			for _, id := range msg.rejected {
				delete(m.active, id)
			}
		} else {
			for _, id := range msg.unsubscribed {
				delete(m.active, id)
			}
		}
	}

	return m, nil
}

// pageStep is the events-per-keypress for paged scroll keys. Adequate
// for typical terminal heights; selection is clamped so we never page
// past either edge of the buffer.
const pageStep = 10

func (m *monitorModel) moveSelected(delta int) {
	if len(m.events) == 0 {
		return
	}
	m.setSelected(m.selected + delta)
}

func (m *monitorModel) setSelected(idx int) {
	if len(m.events) == 0 {
		m.selected = -1
		return
	}
	if idx < 0 {
		idx = 0
	}
	if idx > len(m.events)-1 {
		idx = len(m.events) - 1
	}
	m.selected = idx
}

// atLive reports whether the cursor is on the most recent event. The
// status indicator and the auto-follow behaviour both key off this.
func (m *monitorModel) atLive() bool {
	return len(m.events) == 0 || m.selected == len(m.events)-1
}

func (m *monitorModel) toggleCursor() tea.Cmd {
	if m.cursor < 0 || m.cursor >= len(m.streams) {
		return nil
	}
	id := m.streams[m.cursor].ID
	if m.pending[id] {
		return nil
	}
	wasActive := m.active[id]
	subscribe := !wasActive

	// Optimistic local update.
	if subscribe {
		m.active[id] = true
	} else {
		delete(m.active, id)
	}
	m.pending[id] = true

	client := m.client

	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		var event string
		if subscribe {
			event = "subscribe"
		} else {
			event = "unsubscribe"
		}
		reply, err := client.Push(ctx, lobbyTopic, event, map[string]any{
			"streams": []string{id},
		})
		if err != nil {
			return subscribeReplyMsg{subscribe: subscribe, streamID: id, err: err}
		}
		var resp struct {
			Subscribed   []string `json:"subscribed"`
			Unsubscribed []string `json:"unsubscribed"`
			Rejected     []string `json:"rejected"`
		}
		if jerr := json.Unmarshal(reply.Response, &resp); jerr != nil {
			return subscribeReplyMsg{subscribe: subscribe, streamID: id, err: jerr}
		}
		if reply.Status != "ok" {
			return subscribeReplyMsg{
				subscribe: subscribe, streamID: id,
				err: fmt.Errorf("server rejected: %s", reply.Response),
			}
		}
		return subscribeReplyMsg{
			subscribe:    subscribe,
			streamID:     id,
			subscribed:   resp.Subscribed,
			unsubscribed: resp.Unsubscribed,
			rejected:     resp.Rejected,
		}
	}
}

func (m *monitorModel) applyWelcome(w Welcome) {
	m.streams = w.Streams
}

// subscribeAllStreams batch-subscribes to every stream in the catalog.
// Called once after the welcome envelope arrives so the user sees
// live data without first having to toggle each stream on. Active set
// is populated optimistically; the server's reply reconciles any
// rejections (currently none, but the path is generic).
func (m *monitorModel) subscribeAllStreams() tea.Cmd {
	if len(m.streams) == 0 {
		return nil
	}

	streams := make([]string, 0, len(m.streams))
	for _, s := range m.streams {
		streams = append(streams, s.ID)
		m.active[s.ID] = true
	}

	client := m.client
	log := m.log
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		reply, err := client.Push(ctx, lobbyTopic, "subscribe", map[string]any{
			"streams": streams,
		})
		if err != nil {
			log.Warn("auto-subscribe-all failed", "err", err.Error())
			return nil
		}
		var resp struct {
			Subscribed []string `json:"subscribed"`
			Rejected   []string `json:"rejected"`
		}
		_ = json.Unmarshal(reply.Response, &resp)
		return subscribeReplyMsg{
			subscribe:  true,
			subscribed: resp.Subscribed,
			rejected:   resp.Rejected,
		}
	}
}

// appendOutageMarker injects a synthetic event into the waterfall
// indicating the server has been unreachable since `since`. `closing`
// flips the marker into a "resumed" line, dropped once when the
// connection comes back, capping the gap visually.
func (m *monitorModel) appendOutageMarker(since, at time.Time, closing bool) {
	row := events.Outage(at, since, closing)
	wasLive := m.atLive()
	m.events = append(m.events, row)
	m.evictAged()
	if wasLive {
		m.selected = len(m.events) - 1
	}
}

// evictAged drops events older than eventRetention from the head of
// the buffer, then enforces the hard count ceiling. Selection and
// viewport indices shift so they keep pointing at the same logical
// events while the user is scrolled into history.
func (m *monitorModel) evictAged() {
	cutoff := time.Now().Add(-eventRetention)
	drop := 0
	for drop < len(m.events) && m.events[drop].When.Before(cutoff) {
		drop++
	}
	if remaining := len(m.events) - drop; remaining > eventHardCap {
		drop += remaining - eventHardCap
	}
	if drop == 0 {
		return
	}
	m.events = m.events[drop:]
	m.selected -= drop
	m.viewStart -= drop
	if m.selected < 0 {
		m.selected = 0
	}
	if m.viewStart < 0 {
		m.viewStart = 0
	}
}

// replayAfterReconnect re-issues a `subscribe` for every stream the
// user had toggled on before the disconnect. Phoenix Channels do not
// remember subscriptions across a process restart, so the server's view
// is empty after a rejoin. Returns nil when the active set is empty.
func (m *monitorModel) replayAfterReconnect() tea.Cmd {
	if len(m.active) == 0 {
		return nil
	}
	streams := make([]string, 0, len(m.active))
	for id := range m.active {
		streams = append(streams, id)
	}
	client := m.client
	log := m.log
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_, err := client.Push(ctx, lobbyTopic, "subscribe", map[string]any{
			"streams": streams,
		})
		if err != nil {
			log.Warn("subscribe replay after reconnect failed",
				"err", err.Error(), "streams", streams)
		}
		return nil
	}
}

func (m *monitorModel) handlePush(msg pushMsg) {
	if msg.Event != "stream" {
		return
	}
	var p events.Push
	if err := json.Unmarshal(msg.Payload, &p); err != nil {
		return
	}

	row := events.Project(time.Now(), p)

	wasLive := m.atLive()
	m.events = append(m.events, row)
	m.evictAged()
	if wasLive {
		m.selected = len(m.events) - 1
	}
}

func (m *monitorModel) activeStreams() []string {
	out := make([]string, 0, len(m.active))
	for id := range m.active {
		out = append(out, id)
	}
	return out
}

// =====================================================================
// Rendering
// =====================================================================

var (
	streamSelectedStyle = lipgloss.NewStyle().Bold(true).Foreground(ui.Blue)
	streamPendingStyle  = lipgloss.NewStyle().Foreground(ui.Yellow)
	streamCursorStyle   = lipgloss.NewStyle().Reverse(true)
	focusedTitleStyle   = lipgloss.NewStyle().Bold(true).Foreground(ui.Yellow)
	unfocusedTitleStyle = lipgloss.NewStyle().Bold(true).Faint(true)
	outageRowStyle      = lipgloss.NewStyle().Foreground(ui.Red).Italic(true)

	kindBlockStyle      = lipgloss.NewStyle().Foreground(ui.Blue)
	kindBeaconStyle     = lipgloss.NewStyle().Foreground(ui.Cyan)
	kindCommitStyle     = lipgloss.NewStyle().Foreground(ui.Green)
	kindEntropyStyle    = lipgloss.NewStyle().Foreground(ui.Accent)
	kindItemStyle       = lipgloss.NewStyle().Foreground(ui.Yellow)
	kindHealingStyle    = lipgloss.NewStyle().Foreground(ui.Accent).Italic(true)
	tableHeaderStyle    = lipgloss.NewStyle().Foreground(ui.Dim).Faint(true)
	timestampStyle      = lipgloss.NewStyle().Foreground(ui.Dim)
	idStyle             = lipgloss.NewStyle().Foreground(ui.Label)
	burstCountStyle     = lipgloss.NewStyle().Foreground(ui.Yellow).Italic(true)
	panelLabelStyle     = lipgloss.NewStyle().Foreground(ui.Label)
	panelValueStyle     = lipgloss.NewStyle().Foreground(ui.Value)
	panelRuleStyle      = lipgloss.NewStyle().Foreground(ui.Dim).Faint(true)
	panelHintStyle      = lipgloss.NewStyle().Faint(true).Foreground(ui.Dim)
	leftPaneStyle       = lipgloss.NewStyle().BorderStyle(lipgloss.NormalBorder()).BorderRight(true).PaddingRight(1)
	emptyWaterfallStyle = lipgloss.NewStyle().Faint(true).Padding(1, 2)
)

// Layout constants for the waterfall + detail panel split. The
// waterfall table sizes its own columns, so no column width is pinned
// here.
const (
	detailPanelHeight   = 8 // body lines, exclusive of the rule + blank
	detailPanelOverhead = 2 // rule row + leading blank line
)

func (m *monitorModel) View(width, height int) string {
	if width < 40 {
		// Skinny terminal, render a single-column fallback.
		return paneStyle(width, height).Render(m.renderStreamList(width-4) + "\n\n" + m.renderWaterfall(width-4, height-len(m.streams)-4))
	}
	leftWidth := 32
	rightWidth := width - leftWidth - 1
	if rightWidth < 20 {
		rightWidth = 20
	}

	left := leftPaneStyle.Width(leftWidth).Height(height).Render(m.renderStreamList(leftWidth - 2))
	right := lipgloss.NewStyle().Width(rightWidth).Height(height).Render(m.renderWaterfall(rightWidth, height))
	return lipgloss.JoinHorizontal(lipgloss.Top, left, right)
}

func (m *monitorModel) renderStreamList(width int) string {
	_ = width

	var sb strings.Builder
	title := "Streams"
	if m.focus == focusList {
		title = focusedTitleStyle.Render("▸ Streams")
	} else {
		title = unfocusedTitleStyle.Render("  Streams")
	}
	sb.WriteString(title + "\n\n")

	if len(m.streams) == 0 {
		sb.WriteString(streamPendingStyle.Render("(connecting…)"))
		return sb.String()
	}

	for i, s := range m.streams {
		check := "[ ]"
		if m.active[s.ID] {
			check = "[x]"
		}
		if m.pending[s.ID] {
			check = "[~]"
		}

		line := fmt.Sprintf("%s %s", check, s.ID)
		styled := line
		if m.active[s.ID] {
			styled = streamSelectedStyle.Render(line)
		}
		if m.pending[s.ID] {
			styled = streamPendingStyle.Render(line)
		}
		if i == m.cursor {
			styled = streamCursorStyle.Render(styled)
		}
		sb.WriteString(styled)
		sb.WriteString("\n")
	}

	sb.WriteString("\n")
	sb.WriteString(lipgloss.NewStyle().Faint(true).Render("space toggles • ↑/↓ moves"))
	return sb.String()
}

func (m *monitorModel) renderWaterfall(width, height int) string {
	var titleText string
	if m.focus == focusWaterfall {
		titleText = focusedTitleStyle.Render("▸ Events")
	} else {
		titleText = unfocusedTitleStyle.Render("  Events")
	}

	// Reserve fixed vertical space for the Detail Panel (off when
	// hidden). Layout from top: 1 title row + 1 blank + waterfall
	// rows + (panelOverhead + panelHeight when visible).
	panelTotal := 0
	if !m.detailPanelHidden {
		panelTotal = detailPanelHeight + detailPanelOverhead
	}

	if len(m.events) == 0 {
		// No events yet: just the title and an empty-state hint.
		// The Detail Panel has no row to render to, so we collapse
		// it for the empty case too.
		return titleText + "\n\n" + emptyWaterfallStyle.Render(
			"No events yet.\n\n"+
				"Toggle a stream on the left with space to start receiving events.\n"+
				"Use →/← (or h/l) to focus this pane and ↑/↓ to scroll once events arrive.")
	}

	// Waterfall body rows = total height - title - blank - header -
	// (panel rule + blank + panel body).
	// The table renders its own header line via Headers() so we
	// subtract one row for that.
	rows := height - 2 /* title + blank */ - 1 /* table header */ - panelTotal
	if rows < 1 {
		rows = 1
	}

	total := len(m.events)

	// Slide viewStart just enough to keep selected visible.
	if m.selected < m.viewStart {
		m.viewStart = m.selected
	}
	if m.selected >= m.viewStart+rows {
		m.viewStart = m.selected - rows + 1
	}
	maxStart := total - rows
	if maxStart < 0 {
		maxStart = 0
	}
	if m.viewStart > maxStart {
		m.viewStart = maxStart
	}
	if m.viewStart < 0 {
		m.viewStart = 0
	}

	end := m.viewStart + rows
	if end > total {
		end = total
	}
	visibleAbs := make([]int, end-m.viewStart)
	for i := range visibleAbs {
		visibleAbs[i] = m.viewStart + i
	}

	if m.reverseOrder {
		for i, j := 0, len(visibleAbs)-1; i < j; i, j = i+1, j-1 {
			visibleAbs[i], visibleAbs[j] = visibleAbs[j], visibleAbs[i]
		}
	}

	// Build the rendered block via lipgloss/v2/table. Each visible
	// event maps to one [time, kind, id] row, Detail moved to the
	// per-selection panel below. A faint header row labels the
	// columns.
	tableData := make([][]string, len(visibleAbs))
	for i, abs := range visibleAbs {
		e := m.events[abs]
		tableData[i] = []string{
			e.When.Format("15:04:05.000"),
			e.Kind,
			e.ID,
		}
	}

	tbl := ltable.New().
		Border(lipgloss.HiddenBorder()).
		BorderTop(false).
		BorderBottom(false).
		BorderLeft(false).
		BorderRight(false).
		BorderColumn(false).
		BorderRow(false).
		BorderHeader(false).
		Headers("Time", "Kind", "ID").
		Rows(tableData...).
		StyleFunc(func(row, col int) lipgloss.Style {
			// lipgloss/v2/table uses row -1 for the header row when
			// Headers() is set. Row 0+ is the first data row.
			if row == ltable.HeaderRow {
				return tableHeaderStyle.PaddingRight(2)
			}
			if row < 0 || row >= len(visibleAbs) {
				return lipgloss.NewStyle()
			}
			abs := visibleAbs[row]
			e := m.events[abs]
			selected := abs == m.selected && m.focus == focusWaterfall

			base := columnStyle(col, e)
			if selected {
				base = base.Reverse(true)
			}
			return base
		})

	rendered := tbl.Render()

	// Indent the entire table by 2 columns so the first cell aligns
	// with the "E" in "Events" above it.
	rendered = lipgloss.NewStyle().PaddingLeft(2).Render(rendered)

	// Pad to exactly `rows + 1` lines (1 header + rows data) so the
	// Detail Panel below lands at a stable screen row regardless of
	// how many events fit.
	renderedLines := strings.Count(rendered, "\n")
	if want := rows; renderedLines < want {
		rendered += strings.Repeat("\n", want-renderedLines)
	}

	// Title row carries the indicator on the right side so its
	// position is pinned to the top of the pane and never shifts
	// as the table grows / shrinks below.
	indicator := m.scrollIndicator(m.viewStart, end, total)
	titleRow := titleText + "   " + indicator

	var sb strings.Builder
	sb.WriteString(titleRow + "\n\n")
	sb.WriteString(rendered)

	if !m.detailPanelHidden {
		sb.WriteString("\n")
		sb.WriteString(m.renderDetailPanel(width, detailPanelHeight))
	}
	return sb.String()
}

// renderDetailPanel draws the fixed-height panel below the waterfall
// that shows the FULL payload of the currently-selected row with no
// truncation. Long values (64-char hashes, ULID/UUID lists, etc.) wrap
// onto continuation lines indented under their field name. The panel
// is bounded by `height` body rows; if the selected row has more
// content than fits, a faint "(N more lines, d hides panel)" footer
// replaces the last line.
func (m *monitorModel) renderDetailPanel(width, height int) string {
	rule := panelRuleStyle.Render("── Selected " + strings.Repeat("─", maxInt(0, width-len("── Selected ")-2)))

	if m.selected < 0 || m.selected >= len(m.events) {
		body := panelHintStyle.Render("  Select a row to see full payload")
		return rule + "\n" + body + strings.Repeat("\n", maxInt(0, height-1))
	}

	e := m.events[m.selected]
	lines := buildPanelLines(e, width)

	var sb strings.Builder
	sb.WriteString(rule)
	sb.WriteString("\n")

	rendered := 0
	for _, line := range lines {
		if rendered >= height {
			break
		}
		// If we're about to clip and there's more content, replace
		// this last visible line with a "(N more lines)" hint so
		// the operator knows there's something they're missing.
		if rendered == height-1 && len(lines) > height {
			more := len(lines) - rendered
			hint := fmt.Sprintf("  %s", panelHintStyle.Render(
				fmt.Sprintf("(%d more lines, d hides panel)", more)))
			sb.WriteString(hint)
			sb.WriteString("\n")
			rendered++
			break
		}
		sb.WriteString(line)
		sb.WriteString("\n")
		rendered++
	}

	if rendered < height {
		sb.WriteString(strings.Repeat("\n", height-rendered))
	}
	return sb.String()
}

// buildPanelLines turns a Row into the rendered key/value lines the
// Detail Panel displays. Lines are pre-styled (label + value pieces)
// and respect `width` for value wrapping; long values produce one
// label-bearing line plus N indented continuation lines.
func buildPanelLines(e events.Row, width int) []string {
	const labelW = 16
	const indentLeft = 2

	availForValue := width - indentLeft - labelW - 1
	if availForValue < 20 {
		availForValue = 20
	}
	contIndent := strings.Repeat(" ", indentLeft+labelW+1)
	leftIndent := strings.Repeat(" ", indentLeft)

	var lines []string
	addField := func(label, value string) {
		labelStr := panelLabelStyle.Render(fmt.Sprintf("%-*s", labelW, label))
		first, rest := wrapValue(value, availForValue)
		if first == "" {
			first = panelHintStyle.Render("(none)")
		} else {
			first = panelValueStyle.Render(first)
		}
		lines = append(lines, leftIndent+labelStr+" "+first)
		for _, c := range rest {
			lines = append(lines, contIndent+panelValueStyle.Render(c))
		}
	}

	// Top synthetic fields, always on, in canonical order.
	addField("kind", e.RawKind)
	addField("id", e.ID)
	if e.Stream != "" {
		addField("stream", e.Stream)
	}
	addField("at", formatPanelAt(e))

	// Payload fields. Skip "id" since we surfaced it above; skip
	// "kind"/"stream"/"at" if the server snuck them into Data
	// (defensive, they should only be at the Push level).
	skip := map[string]bool{"id": true, "kind": true, "stream": true, "at": true}

	for _, key := range orderedPayloadKeys(e.Payload, skip) {
		v := e.Payload[key]
		// Handle the nested `latest` projection inside burst payloads
		// by flattening it into the top level under "latest.<key>".
		if key == "latest" {
			if nested, ok := v.(map[string]any); ok {
				for _, lk := range orderedPayloadKeys(nested, nil) {
					addField("latest."+lk, formatPanelValue(nested[lk]))
				}
				continue
			}
		}
		// Maps (by_kind / by_state / claims) flatten one level deep.
		if nested, ok := v.(map[string]any); ok {
			lk := orderedPayloadKeys(nested, nil)
			if len(lk) == 0 {
				addField(key, "")
				continue
			}
			for _, sub := range lk {
				addField(key+"."+sub, formatPanelValue(nested[sub]))
			}
			continue
		}
		addField(key, formatPanelValue(v))
	}

	return lines
}

// orderedPayloadKeys returns map keys in a stable, scannable order:
// known important fields first (state, *_id, hash-like, timestamps),
// then everything else alphabetically.
func orderedPayloadKeys(m map[string]any, skip map[string]bool) []string {
	if len(m) == 0 {
		return nil
	}
	priority := []string{
		"state", "visibility", "method", "direction", "reason", "from_state", "to_state",
		"source", "capture_method",
		"count", "window_ms", "first_at", "last_at", "by_kind", "by_state",
		"team_id", "owner_kind", "owner_id", "creator_id",
		"block_id", "previous_block_id", "item_id", "entropy_observation_id", "epoch_id",
		"merkle_root", "block_hash", "previous_hash", "hash",
		"observation_hash", "entropy_hash", "claims_hash", "item_hash", "commitment_hash",
		"claims", "tags",
		"timestamp", "inserted_at",
		"node", "evidence",
		"latest",
		"since", "duration", "message",
	}
	seen := map[string]bool{}
	var out []string
	for _, k := range priority {
		if skip != nil && skip[k] {
			continue
		}
		if _, ok := m[k]; ok {
			out = append(out, k)
			seen[k] = true
		}
	}
	// Remaining keys alphabetically.
	rest := make([]string, 0, len(m))
	for k := range m {
		if seen[k] {
			continue
		}
		if skip != nil && skip[k] {
			continue
		}
		rest = append(rest, k)
	}
	sort.Strings(rest)
	out = append(out, rest...)
	return out
}

// formatPanelValue renders any payload value as a panel-friendly
// string. Strings pass through; numbers, bools, nils are stringified;
// arrays render as `[a, b, c]`; nested maps are handled by the caller
// (flattened one level).
func formatPanelValue(v any) string {
	if v == nil {
		return ""
	}
	switch x := v.(type) {
	case string:
		return x
	case float64:
		// JSON numbers decode to float64, render integer-shaped
		// numbers without a decimal tail.
		if x == float64(int64(x)) {
			return fmt.Sprintf("%d", int64(x))
		}
		return fmt.Sprintf("%g", x)
	case bool:
		return fmt.Sprintf("%t", x)
	case []any:
		if len(x) == 0 {
			return "[]"
		}
		parts := make([]string, len(x))
		for i, item := range x {
			parts[i] = formatPanelValue(item)
		}
		return "[" + strings.Join(parts, ", ") + "]"
	case map[string]any:
		// Caller flattens these one level; this branch only fires for
		// maps nested two-deep, which we render compactly.
		raw, err := json.Marshal(x)
		if err != nil {
			return fmt.Sprintf("%v", x)
		}
		return string(raw)
	}
	return fmt.Sprintf("%v", v)
}

// formatPanelAt renders the row's server timestamp + a relative
// receive-skew suffix when the gap is meaningful (> 0).
func formatPanelAt(e events.Row) string {
	at := e.When.UTC().Format(time.RFC3339Nano)
	skew := e.ReceivedAt.Sub(e.When)
	switch {
	case skew >= 5*time.Millisecond:
		return at + "   " + panelHintStyle.Render(fmt.Sprintf("(received +%s)", skew.Round(time.Millisecond)))
	case skew <= -5*time.Millisecond:
		return at + "   " + panelHintStyle.Render(fmt.Sprintf("(received %s)", skew.Round(time.Millisecond)))
	}
	return at
}

// wrapValue splits a value string into a first-line chunk plus N
// continuation chunks, each ≤ width characters.
func wrapValue(s string, width int) (first string, rest []string) {
	if width < 1 {
		return s, nil
	}
	if len(s) <= width {
		return s, nil
	}
	first = s[:width]
	remaining := s[width:]
	for len(remaining) > width {
		rest = append(rest, remaining[:width])
		remaining = remaining[width:]
	}
	if remaining != "" {
		rest = append(rest, remaining)
	}
	return first, rest
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// columnStyle picks the per-cell style based on the column index
// (Time / Kind / ID) and the row's severity (Normal / Burst /
// Outage). Selection highlighting wraps this style in a Reverse(true)
// at the call site.
func columnStyle(col int, e events.Row) lipgloss.Style {
	if e.Severity == events.SeverityOutage {
		// All cells in an outage row share one italic-red treatment so
		// the line reads as a connection event, not a data event.
		return outageRowStyle
	}

	switch col {
	case 0: // Time
		return timestampStyle.PaddingRight(2)
	case 1: // Kind
		return kindStyleFor(e.Kind).PaddingRight(2)
	case 2: // ID
		if e.Severity == events.SeverityBurst {
			return burstCountStyle.PaddingRight(2)
		}
		return idStyle.PaddingRight(2)
	}
	return lipgloss.NewStyle()
}

// kindStyleFor returns the resource-colored style for a compact kind
// token. Bursts inherit the resource family color (commitment.burst
// stays green, item.burst stays yellow, etc.) so the rhythm of the
// table is consistent across bursts and individuals. Compact kinds
// for resources without a verb suffix (e.g. plain "block", "beacon",
// "item", "commitment") still match the prefix-with-trailing-dot
// branches because we also match the bare root.
func kindStyleFor(kind string) lipgloss.Style {
	switch {
	case kind == "block" || strings.HasPrefix(kind, "block."):
		return kindBlockStyle
	case kind == "beacon" || strings.HasPrefix(kind, "beacon."):
		return kindBeaconStyle
	case kind == "block_healing" || strings.HasPrefix(kind, "block_healing."):
		return kindHealingStyle
	case kind == "commitment" || strings.HasPrefix(kind, "commitment."):
		return kindCommitStyle
	case kind == "external_commitment" || strings.HasPrefix(kind, "external_commitment."):
		return kindCommitStyle
	case kind == "entropy" || strings.HasPrefix(kind, "entropy."):
		return kindEntropyStyle
	case kind == "item" || strings.HasPrefix(kind, "item."):
		return kindItemStyle
	case strings.HasPrefix(kind, "server."):
		return outageRowStyle
	}
	return lipgloss.NewStyle()
}

// scrollIndicator renders the live/scrolled status, range, and
// chronological order in a faint, subtle style, closer to the
// header clock's treatment than to a status pill. The live/scrolled
// glyph keeps its color cue (green dot / yellow pause) so it's
// scannable at a glance, but the surrounding text is dim so the
// title and the table content stay primary.
func (m *monitorModel) scrollIndicator(start, end, total int) string {
	live := m.atLive()
	order := "oldest→newest"
	if m.reverseOrder {
		order = "newest→oldest"
	}

	var glyph string
	if live {
		glyph = lipgloss.NewStyle().Foreground(ui.Green).Render("●")
	} else {
		glyph = lipgloss.NewStyle().Foreground(ui.Yellow).Render("⏸")
	}

	faint := lipgloss.NewStyle().Foreground(ui.Dim)
	statusWord := "live"
	if !live {
		statusWord = "scrolled"
	}

	parts := []string{
		glyph + " " + faint.Render(statusWord),
		faint.Render(fmt.Sprintf("events %d-%d / %d", start+1, end, total)),
		faint.Render(order),
	}
	return strings.Join(parts, faint.Render("  •  "))
}

func paneStyle(w, h int) lipgloss.Style {
	if w < 1 {
		w = 1
	}
	if h < 1 {
		h = 1
	}
	return lipgloss.NewStyle().
		Width(w).
		Height(h).
		Padding(1, 2)
}
