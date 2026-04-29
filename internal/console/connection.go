// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: MIT

package console

import (
	"context"
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
//
// On a hard connect failure the pane also surfaces a classified error
// section (set via setConnError) explaining what went wrong and what
// the user can try next.
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

	// configFilePath is the on-disk path to the user's TOML settings
	// file. Surfaced on the error pane so a user who can't connect
	// can find their config and edit it directly without knowing
	// where the CLI stores it.
	configFilePath string

	// activeStreams is the count of currently-subscribed streams.
	// Mirrored from the Monitor pane via setActiveStreams; lives
	// here (rather than the header) so ambient chrome stays focused
	// on liveness state.
	activeStreams int

	// connErr / wsURL populate the "connection error" section that
	// the pane renders when the initial dial / handshake fails. Both
	// stay zero-value during normal operation.
	connErr error
	wsURL   string

	// healthTargets are the third-party + Truestamp services the
	// pane probes for liveness. Set once at construction; the result
	// rows below mirror this slice's length and order.
	healthTargets []HealthTarget

	// healthResults parallels healthTargets — index i in this slice
	// is the latest outcome for healthTargets[i]. Updated in place
	// as healthCheckResultMsg events arrive so the table doesn't
	// flicker between cycles.
	healthResults []healthResult

	// lastHealthRunAt tracks when we last fired off a wave of
	// checks. Used for both the "freshness" footer and the manual-
	// refresh rate limiter.
	lastHealthRunAt time.Time

	// pollActive flags whether a tick chain is currently running.
	// Only the chain owner re-arms the next tick; this prevents two
	// chains from running concurrently if the user tabs away and
	// back during a poll cycle.
	pollActive bool
}

func newConnectionModel(logFilePath, configFilePath string, healthTargets []HealthTarget) *connectionModel {
	results := make([]healthResult, len(healthTargets))
	for i, t := range healthTargets {
		results[i] = healthResult{Target: t, State: healthUnknown}
	}
	return &connectionModel{
		pushCounts:     make(map[string]int),
		logFilePath:    logFilePath,
		configFilePath: configFilePath,
		healthTargets:  healthTargets,
		healthResults:  results,
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

// setActiveStreams mirrors the active-stream count from the Monitor
// pane. Called by the root model so the Connection pane can surface
// the number that used to live in the header.
func (m *connectionModel) setActiveStreams(n int) {
	m.activeStreams = n
}

// setConnError records a hard connect-time failure plus the URL we
// tried, so the pane can render a friendly diagnostic section. The
// root model calls this on connectFailedMsg and never clears it —
// once we hit a fatal connect error, the user is in an inspect-and-
// retry mode for the rest of this session.
func (m *connectionModel) setConnError(err error, wsURL string) {
	m.connErr = err
	m.wsURL = wsURL
}

// startHealthPolling fires the first wave of probes and arms the
// tick chain. Called once when the user activates the Connection
// pane. No-op if a chain is already active or there are no
// configured targets.
//
// Returns a tea.Batch of the per-target probe Cmds plus the next
// tick. Subsequent ticks come back through tickHealthChecks.
func (m *connectionModel) startHealthPolling(ctx context.Context) tea.Cmd {
	if m.pollActive || len(m.healthTargets) == 0 {
		return nil
	}
	m.pollActive = true
	return tea.Batch(m.dispatchAllChecks(ctx), healthCheckTickCmd())
}

// stopHealthPolling marks the chain inactive so a queued tick is
// ignored on arrival. Called when the user navigates away from the
// Connection pane.
func (m *connectionModel) stopHealthPolling() {
	m.pollActive = false
}

// tickHealthChecks handles a healthCheckTickMsg. Returns the next
// wave of probe Cmds plus a re-armed tick — but only if the chain
// is still active (the pane is still the focus). When the user has
// tabbed away the chain dies silently here.
func (m *connectionModel) tickHealthChecks(ctx context.Context) tea.Cmd {
	if !m.pollActive {
		return nil
	}
	return tea.Batch(m.dispatchAllChecks(ctx), healthCheckTickCmd())
}

// dispatchAllChecks marks every row as "checking" and returns a
// tea.Cmd that fans out per-target probes concurrently. The
// table's "checking" state renders as a spinner-like dim hint until
// the result arrives.
func (m *connectionModel) dispatchAllChecks(ctx context.Context) tea.Cmd {
	if len(m.healthTargets) == 0 {
		return nil
	}
	m.lastHealthRunAt = time.Now()
	cmds := make([]tea.Cmd, 0, len(m.healthTargets))
	for i, t := range m.healthTargets {
		m.healthResults[i].State = healthChecking
		m.healthResults[i].Err = nil
		cmds = append(cmds, runHealthCheckCmd(ctx, i, t))
	}
	return tea.Batch(cmds...)
}

// applyHealthResult updates the row at the indicated index. Out-of-
// range indices are silently ignored — they would only happen if
// the targets list mutated mid-poll, which it currently can't.
func (m *connectionModel) applyHealthResult(msg healthCheckResultMsg) {
	if msg.Index < 0 || msg.Index >= len(m.healthResults) {
		return
	}
	m.healthResults[msg.Index] = msg.Result
}

// canRunManualHealthCheck reports whether enough time has elapsed
// since the last probe wave to honour an explicit refresh request.
// Used by the `r` keybind to throttle accidental floods.
func (m *connectionModel) canRunManualHealthCheck() bool {
	return time.Since(m.lastHealthRunAt) >= healthCheckMinInterval
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
	connErrTitle     = lipgloss.NewStyle().Bold(true).Foreground(ui.Red)
	connErrHeadline  = lipgloss.NewStyle().Foreground(ui.Red)
	connErrLabel     = lipgloss.NewStyle().Foreground(ui.Label)
	connErrRaw       = lipgloss.NewStyle().Foreground(ui.Dim).Italic(true)
	connHealthOK     = lipgloss.NewStyle().Foreground(ui.Green)
	connHealthFail   = lipgloss.NewStyle().Foreground(ui.Red)
	connHealthWarn   = lipgloss.NewStyle().Foreground(ui.Yellow)
)

func (m *connectionModel) View(width, height int) string {
	var sections []string
	if m.connErr != nil {
		// Lead with the error block — it's why the user is on this
		// pane. Scope / Push / Reconnect would all render empty
		// because we never received a welcome envelope, so we
		// suppress them. Health checks DO render here because they
		// are exactly what helps the user diagnose: "is my
		// internet down, or is it just Truestamp?".
		sections = append(sections, m.renderErrorSection(width))
		if len(m.healthTargets) > 0 {
			sections = append(sections, m.renderHealthSection())
		}
		if m.logFilePath != "" {
			sections = append(sections, m.renderLogsSection())
		}
	} else {
		sections = append(sections, m.renderScopeSection())
		if len(m.healthTargets) > 0 {
			sections = append(sections, m.renderHealthSection())
		}
		sections = append(sections, m.renderPushCountsSection())
		sections = append(sections, m.renderReconnectsSection())
		if m.logFilePath != "" {
			sections = append(sections, m.renderLogsSection())
		}
	}

	body := strings.Join(sections, "\n\n")
	return paneStyle(width, height).Render(body)
}

// renderScopeSection prints the session's scope as a 2-column k/v
// table. lipgloss/v2/table aligns the value column automatically so
// the labels line up regardless of value width.
//
// "Active streams" lives here rather than the header because the
// number is reference data — it changes when the user toggles
// subscriptions and is otherwise stable. Ambient chrome should carry
// liveness state only.
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
	rows = append(rows, []string{"active streams", fmt.Sprintf("%d", m.activeStreams)})

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

// renderHealthSection renders the third-party health-check table:
// one row per target, leading with a ✓ / ✗ / · icon, then the
// human-readable name and the URL. Statuses sort with problems
// first so the user's eye lands on the broken row even on a long
// list.
//
// The "checked at HH:MM:SS" footer keeps users from misreading a
// stale snapshot. The pane re-probes every healthCheckPollInterval
// while it is the active pane; switching to another pane pauses
// the loop, so a paused tab would still show old data otherwise.
func (m *connectionModel) renderHealthSection() string {
	var b strings.Builder
	b.WriteString(connSectionTitle.Render("External services"))
	b.WriteString("\n")

	results := make([]healthResult, len(m.healthResults))
	copy(results, m.healthResults)
	sortHealthResults(results)

	for _, r := range results {
		b.WriteString("  ")
		b.WriteString(healthIcon(r.State))
		b.WriteString("  ")
		b.WriteString(connValueStyle.Render(padRight(r.Target.Name, 32)))
		b.WriteString("  ")
		b.WriteString(connFaintStyle.Render(r.Target.URL))
		if detail := healthDetail(r); detail != "" {
			b.WriteString("  ")
			b.WriteString(connFaintStyle.Render(detail))
		}
		b.WriteString("\n")
	}

	if !m.lastHealthRunAt.IsZero() {
		b.WriteString("  ")
		b.WriteString(connFaintStyle.Render(fmt.Sprintf("checked at %s · refreshes every %s (press r to refresh now)",
			m.lastHealthRunAt.Format("15:04:05"),
			formatPollInterval(healthCheckPollInterval))))
	}
	return b.String()
}

// formatPollInterval renders the health-check cadence as the
// shortest human-readable string: "1m" / "5m" for whole minutes,
// "30s" otherwise. Avoids time.Duration's default "1m0s".
func formatPollInterval(d time.Duration) string {
	if d >= time.Minute && d%time.Minute == 0 {
		return fmt.Sprintf("%dm", int(d/time.Minute))
	}
	return fmt.Sprintf("%ds", int(d/time.Second))
}

// healthIcon returns a colored ✓ / ✗ / spinner for the row's state.
// The icon is the eye-grabbing element so we render it with full
// fidelity; everything else (name, URL) renders in the same style
// as other Connection sections.
func healthIcon(s healthState) string {
	switch s {
	case healthOK:
		return connHealthOK.Render("✓")
	case healthFailed:
		return connHealthFail.Render("✗")
	case healthDegraded:
		return connHealthWarn.Render("!")
	case healthChecking:
		return connFaintStyle.Render("·")
	}
	return connFaintStyle.Render(" ")
}

// healthDetail returns a short trailing fragment for a row: latency
// for healthy targets, error reason for failures. Kept short so the
// table stays scannable; the full error sits in the log file.
func healthDetail(r healthResult) string {
	switch r.State {
	case healthOK:
		return fmt.Sprintf("(%s)", r.Latency.Round(time.Millisecond))
	case healthDegraded:
		return fmt.Sprintf("(HTTP %d)", r.StatusCode)
	case healthFailed:
		return "(unreachable)"
	case healthChecking:
		return "(checking…)"
	}
	return ""
}

// padRight pads s with spaces to width n. Used to keep the URL
// column left-aligned even when names vary in length. lipgloss's
// table primitive overcomplicates this for a 3-column ad-hoc list,
// so we fall back to a plain pad helper.
func padRight(s string, n int) string {
	if len(s) >= n {
		return s
	}
	return s + strings.Repeat(" ", n-len(s))
}

// renderErrorSection draws the friendly diagnostic for a hard
// connect-time failure: a one-line headline, the URL we tried, the
// classified hints ("what to try"), and the raw error string at the
// bottom in dim italic for the rare cases where the user wants to
// share it with support. Nothing here scrolls — if hints overflow
// the pane height the user can resize the terminal.
func (m *connectionModel) renderErrorSection(_ int) string {
	kind := classifyConnError(m.connErr)

	var b strings.Builder
	b.WriteString(connErrTitle.Render("Connection error"))
	b.WriteString("\n\n")

	b.WriteString("  ")
	b.WriteString(connErrHeadline.Render(kind.title()))
	b.WriteString("\n")

	if m.wsURL != "" {
		b.WriteString("\n  ")
		b.WriteString(connErrLabel.Render("Server: "))
		b.WriteString(connValueStyle.Render(m.wsURL))
		b.WriteString("\n")
	}

	if m.configFilePath != "" {
		b.WriteString("  ")
		b.WriteString(connErrLabel.Render("Settings file: "))
		b.WriteString(connValueStyle.Render(m.configFilePath))
		b.WriteString("\n")
		b.WriteString("    ")
		b.WriteString(connFaintStyle.Render("(open this file in any text editor to change the server address or other settings)"))
		b.WriteString("\n")
	}

	hints := kind.hints()
	if len(hints) > 0 {
		b.WriteString("\n  ")
		b.WriteString(connErrLabel.Render("What to try:"))
		b.WriteString("\n")
		for _, h := range hints {
			b.WriteString("    • ")
			b.WriteString(connValueStyle.Render(h))
			b.WriteString("\n")
		}
	}

	if msg := m.connErr.Error(); msg != "" {
		b.WriteString("\n  ")
		b.WriteString(connErrLabel.Render("Raw error:"))
		b.WriteString("\n    ")
		b.WriteString(connErrRaw.Render(msg))
	}

	return b.String()
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
