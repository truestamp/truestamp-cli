// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: MIT

package console

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
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
	// stays anchored on a specific event while the buffer grows behind it.
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
	// cross-pane Tab at the app level still cycles the outer panes
	// (Monitor → New Item → Connection); focus is internal to Monitor.
	focus monitorFocus
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
	case tea.MouseWheelMsg:
		// Trackpad / mouse wheel always means "move the cursor in
		// screen-up / screen-down direction", regardless of the
		// reverseOrder flag. Mirrors the ↑/↓ key handler which
		// already does the chrono-vs-screen translation.
		mw := msg.Mouse()
		screenUpDelta := -1
		screenDnDelta := 1
		if m.reverseOrder {
			screenUpDelta, screenDnDelta = 1, -1
		}
		switch mw.Button {
		case tea.MouseWheelUp:
			m.moveSelected(screenUpDelta)
		case tea.MouseWheelDown:
			m.moveSelected(screenDnDelta)
		case tea.MouseWheelLeft, tea.MouseWheelRight:
			// Horizontal wheel events are ignored — the waterfall
			// has no horizontal scrolling. Falls through silently.
		}
		return m, nil

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

		// Reverse order works regardless of focus — it's the only
		// operation specific to the waterfall that we surface globally
		// so users don't have to remember to focus first.
		if key == "r" {
			m.reverseOrder = !m.reverseOrder
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
				// "Top of screen" — newest in reverse, oldest in chrono.
				if m.reverseOrder {
					m.setSelected(len(m.events) - 1)
				} else {
					m.setSelected(0)
				}
			case "end", "G":
				// "Bottom of screen" — oldest in reverse, newest in chrono.
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
		// Server is the source of truth — mirror the canonical set.
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
	// Local color styles used by the scroll indicator. App-level
	// status colors live in chrome.Theme; this is the same palette,
	// just named for the contexts in which monitor uses them.
	statusOK  = lipgloss.NewStyle().Foreground(ui.Green)
	statusErr = lipgloss.NewStyle().Foreground(ui.Red)

	streamSelectedStyle = lipgloss.NewStyle().Bold(true).Foreground(ui.Blue)
	streamPendingStyle  = lipgloss.NewStyle().Foreground(ui.Yellow)
	streamCursorStyle   = lipgloss.NewStyle().Reverse(true)
	focusedTitleStyle   = lipgloss.NewStyle().Bold(true).Foreground(ui.Yellow)
	unfocusedTitleStyle = lipgloss.NewStyle().Bold(true).Faint(true)
	outageRowStyle      = lipgloss.NewStyle().Foreground(ui.Red).Italic(true)

	kindBlockStyle      = lipgloss.NewStyle().Foreground(ui.Blue)
	kindCommitStyle     = lipgloss.NewStyle().Foreground(ui.Green)
	kindEntropyStyle    = lipgloss.NewStyle().Foreground(ui.Accent)
	kindItemStyle       = lipgloss.NewStyle().Foreground(ui.Yellow)
	timestampStyle      = lipgloss.NewStyle().Foreground(ui.Dim)
	idStyle             = lipgloss.NewStyle().Foreground(ui.Label)
	burstCountStyle     = lipgloss.NewStyle().Foreground(ui.Yellow).Italic(true)
	detailStyle         = lipgloss.NewStyle().Foreground(ui.Value)
	leftPaneStyle       = lipgloss.NewStyle().BorderStyle(lipgloss.NormalBorder()).BorderRight(true).PaddingRight(1)
	emptyWaterfallStyle = lipgloss.NewStyle().Faint(true).Padding(1, 2)
)

func (m *monitorModel) View(width, height int) string {
	if width < 40 {
		// Skinny terminal — render a single-column fallback.
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
	_ = width

	var title string
	if m.focus == focusWaterfall {
		title = focusedTitleStyle.Render("▸ Events")
	} else {
		title = unfocusedTitleStyle.Render("  Events")
	}

	if len(m.events) == 0 {
		return title + "\n\n" + emptyWaterfallStyle.Render(
			"No events yet.\n\n"+
				"Toggle a stream on the left with space to start receiving events.\n"+
				"Use →/← (or h/l) to focus this pane and ↑/↓ to scroll once events arrive.")
	}

	// Reserve a line for the title and a line for the position indicator
	// below the event rows.
	rows := height - 4
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
	// event maps to one [time, kind, id, detail] row; the StyleFunc
	// applies per-row severity styling and selection highlight. The
	// table renders to a single string; we pad with blank lines so
	// the scroll indicator below lands at the same screen row even
	// when the buffer holds fewer events than fit.
	tableData := make([][]string, len(visibleAbs))
	for i, abs := range visibleAbs {
		e := m.events[abs]
		tableData[i] = []string{
			e.When.Format("15:04:05.000"),
			e.Kind,
			e.ID,
			e.Detail,
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
		Rows(tableData...).
		StyleFunc(func(row, col int) lipgloss.Style {
			// table row indices are 0-based for data when no header
			// is set; with our `BorderHeader(false)` and no Headers,
			// row 0 is the first data row. Adjust if we add headers.
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

	// Pad to fixed `rows` height so the indicator stays anchored.
	renderedLines := strings.Count(rendered, "\n")
	if renderedLines < rows {
		rendered += strings.Repeat("\n", rows-renderedLines)
	}

	var sb strings.Builder
	sb.WriteString(title + "\n\n")
	sb.WriteString(rendered)
	if !strings.HasSuffix(rendered, "\n") {
		sb.WriteString("\n")
	}
	sb.WriteString(m.scrollIndicator(m.viewStart, end, total))
	return sb.String()
}

// columnStyle picks the per-cell style based on the column index
// (Time / Kind / ID / Detail) and the row's severity (Normal / Burst /
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
	case 3: // Detail
		return detailStyle
	}
	return lipgloss.NewStyle()
}

// kindStyleFor returns the resource-colored style for a `kind`.
// Bursts inherit the resource family color (commitment.burst stays
// green, item.burst stays yellow, etc.) so the rhythm of the table
// is consistent across bursts and individuals.
func kindStyleFor(kind string) lipgloss.Style {
	switch {
	case strings.HasPrefix(kind, "block."):
		return kindBlockStyle
	case strings.HasPrefix(kind, "commitment.") || strings.HasPrefix(kind, "external_commitment."):
		return kindCommitStyle
	case strings.HasPrefix(kind, "entropy."):
		return kindEntropyStyle
	case strings.HasPrefix(kind, "item."):
		return kindItemStyle
	case strings.HasPrefix(kind, "server."):
		return outageRowStyle
	}
	return lipgloss.NewStyle()
}

func (m *monitorModel) scrollIndicator(start, end, total int) string {
	live := m.atLive()
	order := "oldest→newest"
	if m.reverseOrder {
		order = "newest→oldest"
	}

	var status string
	if live {
		status = statusOK.Render("● live")
	} else {
		status = statusErr.Render("⏸ scrolled")
	}

	rng := fmt.Sprintf("events %d-%d / %d (last 24h)", start+1, end, total)
	return fmt.Sprintf("%s  %s  (%s)", status, rng, order)
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
