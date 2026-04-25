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
	"github.com/truestamp/truestamp-cli/internal/logging"
	"github.com/truestamp/truestamp-cli/internal/ui"
	"github.com/truestamp/truestamp-cli/internal/wschannel"
)

// Run launches the interactive console TUI. Blocks until the user quits.
func Run(ctx context.Context, opts Options) error {
	logger := opts.Logger
	if logger == nil {
		logger = logging.Discard()
	}

	client, err := wschannel.New(wschannel.Options{
		URL:    opts.WSURL,
		APIKey: opts.APIKey,
		Logger: logger,
	})
	if err != nil {
		return fmt.Errorf("wschannel.New: %w", err)
	}

	mdl := newModel(client, opts, logger)

	p := tea.NewProgram(mdl, tea.WithContext(ctx))

	// Run Connect in a goroutine after the program has started, so the
	// initial render can show "connecting…" immediately.
	go func() {
		connectCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
		defer cancel()
		raw, err := client.Connect(connectCtx)
		if err != nil {
			p.Send(connectFailedMsg{Err: err})
			return
		}
		var w Welcome
		if jerr := json.Unmarshal(raw, &w); jerr != nil {
			p.Send(connectFailedMsg{Err: fmt.Errorf("decode welcome: %w", jerr)})
			return
		}
		// Join the auxiliary clock topic on the same socket. Any
		// failure here is non-fatal: the lobby still works without the
		// header clock; log it and move on.
		if _, jerr := client.JoinTopic(connectCtx, clockTopic); jerr != nil {
			logger.Warn("clock topic join failed", "err", jerr.Error())
		}
		p.Send(welcomeMsg{Welcome: w})
	}()

	_, runErr := p.Run()
	_ = client.Close()
	return runErr
}

// Options carries the runtime configuration into the TUI.
type Options struct {
	WSURL  string
	APIKey string

	// Logger receives diagnostic / postmortem entries from the
	// wschannel transport and from the TUI itself. May be nil — the
	// TUI substitutes a discard logger so panes can call Log methods
	// unconditionally.
	Logger *slog.Logger

	// LogFilePath is shown on the Connection pane so the user can
	// `tail -f` it if they need to debug something.
	LogFilePath string
}

// pane identifies the active pane.
type pane int

const (
	paneMonitor pane = iota
	paneNewItem
	paneConnection
)

func (p pane) title() string {
	switch p {
	case paneMonitor:
		return "Monitor"
	case paneNewItem:
		return "New Item"
	case paneConnection:
		return "Connection"
	default:
		return "?"
	}
}

// model is the root Bubble Tea model.
type model struct {
	client *wschannel.Client
	opts   Options
	log    *slog.Logger

	// Connection state.
	state   connState
	welcome Welcome
	connErr error

	// Latest server-time tick from the `console:clock` topic. Zero
	// value means "no tick received yet" — we render a placeholder
	// rather than the local clock so users always know it's a server
	// time, never a client guess.
	serverTime time.Time

	// Reconnect bookkeeping. Populated by reconnectingMsg events from
	// the wschannel session loop and consumed by the header countdown
	// and the Monitor pane's outage markers.
	reconnectAttempt int
	nextAttemptAt    time.Time
	disconnectedAt   time.Time
	lastOutageMark   time.Time

	// Active pane.
	active pane

	// Window size.
	width, height int

	// Pane-specific state.
	monitor    *monitorModel
	newItem    *newItemModel
	connection *connectionModel
}

type connState int

const (
	connConnecting connState = iota
	connConnected
	connReconnecting
	connClosed
	connFailed
)

func newModel(client *wschannel.Client, opts Options, log *slog.Logger) *model {
	m := &model{
		client: client,
		opts:   opts,
		log:    log,
		state:  connConnecting,
		active: paneMonitor,
	}
	m.monitor = newMonitorModel(client, log)
	m.newItem = newNewItemModel(client)
	m.connection = newConnectionModel(opts.LogFilePath)
	return m
}

func (m *model) Init() tea.Cmd { return nil }

// outageMarkerInterval gates how often we drop a synthetic "server
// down" line into the Monitor's waterfall while the session is in
// reconnect. The first marker fires immediately when we lose the
// connection; subsequent ones fire every interval. A short interval
// keeps the gap visually obvious in the scrollback without flooding
// the buffer.
const outageMarkerInterval = 10 * time.Second

// reconnectTickCmd schedules a reconnectTickMsg one second from now.
// Update returns it back into the chain while the session is in the
// reconnecting state, then lets the chain die when state flips to
// connected.
func reconnectTickCmd() tea.Cmd {
	return tea.Tick(time.Second, func(time.Time) tea.Msg {
		return reconnectTickMsg{}
	})
}

func (m *model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		return m, nil

	case tea.KeyPressMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit
		case "tab":
			m.active = (m.active + 1) % 3
			return m, nil
		case "1":
			m.active = paneMonitor
			return m, nil
		case "2":
			m.active = paneNewItem
			return m, nil
		case "3":
			m.active = paneConnection
			return m, nil
		}

	case welcomeMsg:
		m.state = connConnected
		m.welcome = msg.Welcome
		m.monitor.applyWelcome(msg.Welcome)
		m.connection.applyWelcome(msg.Welcome)
		return m, tea.Batch(
			m.monitor.subscribeAllStreams(),
			waitForPush(m.client),
		)

	case connectFailedMsg:
		m.state = connFailed
		m.connErr = msg.Err
		return m, nil

	case closedMsg:
		m.state = connClosed
		return m, nil

	case clockTickMsg:
		m.serverTime = msg.At
		return m, waitForPush(m.client)

	case rejoinedMsg:
		// Reconnect just succeeded for this topic. The lobby needs its
		// subscriptions replayed; other topics (like clock) only need
		// the rejoin itself, which the client already did.
		var cmd tea.Cmd
		if msg.Topic == lobbyTopic {
			cmd = m.monitor.replayAfterReconnect()
		}
		// Cap the outage with a closing marker and tally the cycle
		// the first time we observe a successful rejoin per outage —
		// rejoinedMsg fires once per topic but this only counts once.
		if !m.disconnectedAt.IsZero() {
			downtime := time.Since(m.disconnectedAt)
			m.monitor.appendOutageMarker(m.disconnectedAt, time.Now(), true)
			m.connection.recordReconnect(downtime)
			m.log.Info("reconnect succeeded",
				"downtime", downtime.String(),
				"attempts", m.reconnectAttempt)
		}
		m.state = connConnected
		m.reconnectAttempt = 0
		m.nextAttemptAt = time.Time{}
		m.disconnectedAt = time.Time{}
		m.lastOutageMark = time.Time{}
		return m, tea.Batch(cmd, waitForPush(m.client))

	case reconnectingMsg:
		if m.disconnectedAt.IsZero() {
			m.disconnectedAt = time.Now()
		}
		m.state = connReconnecting
		m.reconnectAttempt = msg.Attempt
		m.nextAttemptAt = msg.NextAttemptAt
		// Drop a marker immediately so the gap is visible the moment
		// we lose the connection, not 10s later.
		if m.lastOutageMark.IsZero() {
			m.monitor.appendOutageMarker(m.disconnectedAt, time.Now(), false)
			m.lastOutageMark = time.Now()
		}
		return m, tea.Batch(reconnectTickCmd(), waitForPush(m.client))

	case reconnectTickMsg:
		if m.state != connReconnecting {
			// Reconnect succeeded between ticks — stop the ticker chain.
			return m, nil
		}
		// Once we've been down for the full marker interval since the
		// last marker, drop another one into the waterfall.
		now := time.Now()
		if now.Sub(m.lastOutageMark) >= outageMarkerInterval {
			m.monitor.appendOutageMarker(m.disconnectedAt, now, false)
			m.lastOutageMark = now
		}
		return m, reconnectTickCmd()

	case pushMsg:
		m.monitor.handlePush(msg)
		m.newItem.handlePush(msg)
		m.connection.recordPush(msg.Event)
		return m, waitForPush(m.client)
	}

	// Forward other messages (e.g. form keystrokes) to the active pane.
	var cmd tea.Cmd
	switch m.active {
	case paneMonitor:
		m.monitor, cmd = m.monitor.Update(msg)
	case paneNewItem:
		m.newItem, cmd = m.newItem.Update(msg)
	case paneConnection:
		m.connection, cmd = m.connection.Update(msg)
	}
	return m, cmd
}

func (m *model) View() tea.View {
	bodyHeight := m.height - 3
	if bodyHeight < 1 {
		bodyHeight = 1
	}

	var body string
	switch m.active {
	case paneMonitor:
		body = m.monitor.View(m.width, bodyHeight)
	case paneNewItem:
		body = m.newItem.View(m.width, bodyHeight)
	case paneConnection:
		body = m.connection.View(m.width, bodyHeight)
	}

	header := m.renderHeader()
	footer := m.renderFooter()
	content := lipgloss.JoinVertical(lipgloss.Left, header, body, footer)

	v := tea.NewView(content)
	v.AltScreen = true
	return v
}

var (
	tabActiveStyle   = lipgloss.NewStyle().Bold(true).Padding(0, 1).Underline(true)
	tabInactiveStyle = lipgloss.NewStyle().Faint(true).Padding(0, 1)
	headerBarStyle   = lipgloss.NewStyle().Padding(0, 1)
	footerStyle      = lipgloss.NewStyle().Faint(true).Padding(0, 1)
	statusOK         = lipgloss.NewStyle().Foreground(ui.Green)
	statusErr        = lipgloss.NewStyle().Foreground(ui.Red)
	clockStyle       = lipgloss.NewStyle().Faint(true)
)

func (m *model) renderHeader() string {
	tabs := []string{}
	for _, p := range []pane{paneMonitor, paneNewItem, paneConnection} {
		label := fmt.Sprintf("[%d] %s", int(p)+1, p.title())
		if p == m.active {
			tabs = append(tabs, tabActiveStyle.Render(label))
		} else {
			tabs = append(tabs, tabInactiveStyle.Render(label))
		}
	}
	tabBar := strings.Join(tabs, "  ")

	right := m.statusText()
	if clock := m.clockText(); clock != "" {
		right = right + "  •  " + clock
	}

	rightSide := headerBarStyle.Render(right)
	leftSide := headerBarStyle.Render(tabBar)
	gap := strings.Repeat(" ", maxInt(0, m.width-lipgloss.Width(leftSide)-lipgloss.Width(rightSide)))
	return leftSide + gap + rightSide
}

// reconnectingText renders the header status while a reconnect is in
// flight: "reconnecting in 3s (attempt 4)" — countdown plus attempt
// number, both kept fresh by the per-second reconnectTickMsg loop.
func (m *model) reconnectingText() string {
	if m.nextAttemptAt.IsZero() {
		return "reconnecting"
	}
	remaining := time.Until(m.nextAttemptAt).Round(time.Second)
	if remaining <= 0 {
		return fmt.Sprintf("reconnecting now (attempt %d)", m.reconnectAttempt)
	}
	return fmt.Sprintf("reconnecting in %s (attempt %d)", remaining, m.reconnectAttempt)
}

// clockText renders the most recent server-time tick as an RFC 3339 /
// ISO 8601 string with millisecond precision. Empty until the first
// `tick` arrives so we never show a stale local guess.
func (m *model) clockText() string {
	if m.serverTime.IsZero() {
		return ""
	}
	return clockStyle.Render(m.serverTime.UTC().Format("2006-01-02T15:04:05Z"))
}

func (m *model) statusText() string {
	switch m.state {
	case connConnecting:
		return "connecting…"
	case connConnected:
		s := fmt.Sprintf("connected • %s • %d streams", m.welcome.Scope.Plan, len(m.monitor.activeStreams()))
		return statusOK.Render(s)
	case connReconnecting:
		return statusErr.Render(m.reconnectingText())
	case connClosed:
		return statusErr.Render("disconnected")
	case connFailed:
		return statusErr.Render("error: " + truncate(m.connErr.Error(), 40))
	}
	return ""
}

func (m *model) renderFooter() string {
	hints := "tab: switch pane   1/2/3: go to pane   q: quit"
	switch m.active {
	case paneMonitor:
		hints = "tab: pane   ←/→: focus   ↑/↓: scroll   pgup/pgdn: page   g/G: ends   space: toggle   r: reverse   q: quit"
	case paneNewItem:
		hints = "tab: switch pane   enter: submit   esc: cancel   q: quit"
	case paneConnection:
		hints = "tab: switch pane   r: send ping   q: quit"
	}
	return footerStyle.Render(hints)
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n-1] + "…"
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}
