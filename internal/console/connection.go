// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: MIT

package console

import (
	"fmt"
	"sort"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	lipgloss "charm.land/lipgloss/v2"
	ltable "charm.land/lipgloss/v2/table"
	"github.com/truestamp/truestamp-cli/internal/ui"
)

// connectionModel renders connection diagnostics: scope summary, push
// counts by event, reconnect summary, and the log file path. It does
// not display individual transport errors — those are routed to the
// file logger so the UI stays focused on operational state, not
// per-event diagnostic noise.
type connectionModel struct {
	welcome     Welcome
	pushCounts  map[string]int
	connectedAt time.Time

	// Reconnect bookkeeping. Updated by the root model on each
	// observed reconnect cycle so the user can see at-a-glance how
	// stable the session has been since they connected.
	reconnectCount int
	totalDowntime  time.Duration

	// logFilePath is shown so the user can `tail -f` it for live
	// transport diagnostics or grep it post-incident.
	logFilePath string
}

func newConnectionModel(logFilePath string) *connectionModel {
	return &connectionModel{
		pushCounts:  make(map[string]int),
		logFilePath: logFilePath,
	}
}

func (m *connectionModel) Update(msg tea.Msg) (*connectionModel, tea.Cmd) {
	_ = msg
	return m, nil
}

func (m *connectionModel) applyWelcome(w Welcome) {
	m.welcome = w
	m.connectedAt = time.Now()
}

func (m *connectionModel) recordPush(event string) {
	m.pushCounts[event]++
}

// recordReconnect increments the cycle counter and tallies downtime.
// Called by the root model exactly once per reconnect (when the
// session transitions from connReconnecting back to connConnected).
func (m *connectionModel) recordReconnect(downtime time.Duration) {
	m.reconnectCount++
	m.totalDowntime += downtime
}

// =====================================================================
// Rendering
// =====================================================================

var (
	connSectionTitle = lipgloss.NewStyle().Bold(true).Foreground(ui.Accent)
	connKeyStyle     = lipgloss.NewStyle().Foreground(ui.Label)
	connValueStyle   = lipgloss.NewStyle().Foreground(ui.Value)
	connFaintStyle   = lipgloss.NewStyle().Foreground(ui.Dim)
	connReconnectOK  = lipgloss.NewStyle().Foreground(ui.Green)
	connReconnectBad = lipgloss.NewStyle().Foreground(ui.Yellow)
)

func (m *connectionModel) View(width, height int) string {
	var sections []string
	sections = append(sections, m.renderScopeSection())
	sections = append(sections, m.renderPushCountsSection())
	sections = append(sections, m.renderReconnectsSection())
	if m.logFilePath != "" {
		sections = append(sections, m.renderLogsSection())
	}

	body := strings.Join(sections, "\n\n")
	return paneStyle(width, height).Render(body)
}

// renderScopeSection prints the session's scope as a 2-column k/v
// table. lipgloss/v2/table aligns the value column automatically so
// the labels line up regardless of value width.
func (m *connectionModel) renderScopeSection() string {
	rows := [][]string{
		{"user", m.welcome.Scope.UserID},
		{"team", m.welcome.Scope.TeamID},
		{"plan", m.welcome.Scope.Plan},
		{"server", m.welcome.Server.Version},
	}
	if !m.connectedAt.IsZero() {
		rows = append(rows, []string{"connected for", time.Since(m.connectedAt).Truncate(time.Second).String()})
	}

	tbl := keyValueTable(rows)

	return connSectionTitle.Render("Scope") + "\n" + tbl.Render()
}

// renderPushCountsSection prints the push event counts as a 2-column
// table sorted alphabetically (deterministic across renders).
func (m *connectionModel) renderPushCountsSection() string {
	if len(m.pushCounts) == 0 {
		return connSectionTitle.Render("Push counts") + "\n" +
			connFaintStyle.Render("  (none yet)")
	}

	names := make([]string, 0, len(m.pushCounts))
	for event := range m.pushCounts {
		names = append(names, event)
	}
	sort.Strings(names)

	rows := make([][]string, 0, len(names))
	for _, event := range names {
		rows = append(rows, []string{event, fmt.Sprintf("%d", m.pushCounts[event])})
	}
	tbl := keyValueTable(rows)

	return connSectionTitle.Render("Push counts by event") + "\n" + tbl.Render()
}

// renderReconnectsSection prints reconnect statistics. The "stable"
// case (no reconnects since session start) gets a green ✓ marker; any
// reconnects show in yellow with the count + total downtime.
func (m *connectionModel) renderReconnectsSection() string {
	if m.reconnectCount == 0 {
		return connSectionTitle.Render("Reconnects") + "\n" +
			connReconnectOK.Render("  ✓ stable since session start")
	}
	rows := [][]string{
		{"count", fmt.Sprintf("%d", m.reconnectCount)},
		{"total downtime", m.totalDowntime.Truncate(time.Second).String()},
	}
	tbl := keyValueTable(rows)
	return connSectionTitle.Render("Reconnects") + "\n" +
		connReconnectBad.Render("  ⚠ session has reconnected") + "\n" +
		tbl.Render()
}

// renderLogsSection points the user at the file logger for live
// transport diagnostics. The path is non-redacted (it's a local
// filesystem path) and the tail-f hint is faint so it doesn't compete
// with the status data above.
func (m *connectionModel) renderLogsSection() string {
	return connSectionTitle.Render("Logs") + "\n" +
		"  " + connValueStyle.Render(m.logFilePath) + "\n" +
		"  " + connFaintStyle.Render("(tail -f for live transport diagnostics)")
}

// keyValueTable builds a small 2-column table with the key column
// styled as a label and the value column styled as a value. No
// borders — the section title above and the leading indent provide
// enough visual containment for a stat block.
func keyValueTable(rows [][]string) *ltable.Table {
	return ltable.New().
		Border(lipgloss.HiddenBorder()).
		BorderTop(false).
		BorderBottom(false).
		BorderLeft(false).
		BorderRight(false).
		BorderColumn(false).
		BorderRow(false).
		BorderHeader(false).
		Rows(rows...).
		StyleFunc(func(_, col int) lipgloss.Style {
			switch col {
			case 0:
				return connKeyStyle.PaddingLeft(2).PaddingRight(2)
			case 1:
				return connValueStyle
			}
			return lipgloss.NewStyle()
		})
}
