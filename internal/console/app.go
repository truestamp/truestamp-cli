// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: MIT

package console

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"charm.land/bubbles/v2/help"
	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	lipgloss "charm.land/lipgloss/v2"
	"github.com/truestamp/truestamp-cli/internal/auth"
	"github.com/truestamp/truestamp-cli/internal/console/chrome"
	"github.com/truestamp/truestamp-cli/internal/console/keys"
	"github.com/truestamp/truestamp-cli/internal/logging"
	"github.com/truestamp/truestamp-cli/internal/wschannel"
)

// Run launches the interactive console TUI. Blocks until the user quits.
func Run(ctx context.Context, opts Options) error {
	logger := opts.Logger
	if logger == nil {
		logger = logging.Discard()
	}

	wsOpts := wschannel.Options{URL: opts.WSURL, Logger: logger}
	switch {
	case opts.Authorizer != nil && opts.Authorizer.Mode() == auth.ModeOAuth:
		// OAuth: a per-dial fresh token, passed as the `access_token`
		// query param on the upgrade because a Phoenix WS upgrade can't
		// expose the Authorization header to the socket. A dead session
		// is fatal (stop retrying → prompt re-login).
		wsOpts.BearerToken = opts.Authorizer.BearerToken
		wsOpts.ForceRefresh = opts.Authorizer.ForceRefresh
		wsOpts.AccessTokenExpiry = opts.Authorizer.AccessTokenExpiry
		wsOpts.FatalDialErr = func(err error) bool { return errors.Is(err, auth.ErrSessionExpired) }
	case opts.Authorizer != nil:
		// API-key mode: resolve the key once for query-param auth.
		key, kerr := opts.Authorizer.BearerToken(ctx)
		if kerr != nil {
			return fmt.Errorf("resolving credential: %w", kerr)
		}
		wsOpts.APIKey = key
	}

	client, err := wschannel.New(wsOpts)
	if err != nil {
		return fmt.Errorf("wschannel.New: %w", err)
	}

	mdl := newModel(client, opts, logger)

	// Mouse capture is intentionally OFF (model.View leaves
	// MouseMode at the zero value, MouseModeNone). With capture on,
	// the terminal sends mouse events to the program and disables
	// native click-drag text selection — defeating the Detail
	// Panel's purpose of letting operators copy full untruncated
	// hashes / IDs out of the TUI. Keyboard scroll (↑/↓/j/k/PgUp
	// /PgDn/g/G) handles the waterfall fine without a trackpad.
	p := tea.NewProgram(mdl, tea.WithContext(ctx))

	// Run Connect in a goroutine after the program has started, so the
	// initial render can show "connecting…" immediately.
	go func() {
		connectCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
		defer cancel()
		raw, err := client.Connect(connectCtx)
		if err != nil {
			logger.Warn("console connect failed", "err", err.Error())
			p.Send(connectFailedMsg{Err: err})
			return
		}
		var w Welcome
		if jerr := json.Unmarshal(raw, &w); jerr != nil {
			p.Send(connectFailedMsg{Err: fmt.Errorf("decode welcome: %w", jerr)})
			return
		}
		logger.Info("console connected", "ws_url", opts.WSURL)
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
	WSURL string

	// Authorizer supplies the WebSocket credential: OAuth (Bearer header,
	// auto-refreshed) or API key (query param). The Teams pane's HTTP
	// calls use the process-wide auth.Default() independently.
	Authorizer auth.Authorizer

	// APIURL is the base JSON:API URL (e.g. https://www.truestamp.com/api/json).
	// Threaded into the Teams pane so it can call ListMyMemberships /
	// GetTeam without re-importing the top-level config package.
	APIURL string

	// ActiveTeamID is the team id currently stored in config.toml. The
	// Teams pane uses this to mark the row in the membership list and
	// drive the access-loss check on startup. May be empty when no
	// team has been configured yet.
	ActiveTeamID string

	// Logger receives diagnostic / postmortem entries from the
	// wschannel transport and from the TUI itself. May be nil — the
	// TUI substitutes a discard logger so panes can call Log methods
	// unconditionally.
	Logger *slog.Logger

	// LogFilePath is shown on the Connection pane so the user can
	// `tail -f` it if they need to debug something.
	LogFilePath string

	// ConfigFilePath is the path to the user's TOML settings file
	// (e.g. ~/.config/truestamp/config.toml on macOS/Linux). Surfaced
	// on the Connection pane so a user who can't sign in can find
	// the file and edit it directly without knowing CLI internals.
	// May be empty when the platform's home directory can't be
	// resolved; the pane suppresses the line in that case.
	ConfigFilePath string

	// HealthTargets is the list of services the Connection pane
	// probes for liveness. Populated by cmd/console.go from the
	// active configuration so user-overridden api_url / keyring_url
	// values are honored. Empty disables the section entirely.
	HealthTargets []HealthTarget
}

// pane identifies the active pane.
type pane int

const (
	paneMonitor pane = iota
	paneNewItem
	paneTeam
	paneConnection
)

// numPanes is the count of registered panes — used for cyclic
// next/prev navigation in Update. Adding a new pane requires
// updating this constant alongside the enum and the title() switch.
const numPanes = 4

func (p pane) title() string {
	switch p {
	case paneMonitor:
		return "Monitor"
	case paneNewItem:
		return "New Item"
	case paneTeam:
		return "Teams"
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

	// Page chrome — owns header/footer rendering and theme. Built
	// once at startup; pane Views render into the body area.
	theme               *chrome.Theme
	footer              chrome.Footer
	appKeys             keys.AppKeys
	monitorKeys         keys.MonitorKeys
	newItemKeys         keys.NewItemKeys
	newItemWatchingKeys keys.NewItemWatchingKeys
	teamKeys            keys.TeamKeys
	teamCreateKeys      keys.TeamCreateKeys
	connKeys            keys.ConnectionKeys

	// confirmingQuit is true while the "Really quit? y/n" prompt is
	// up. Set by q (when the active pane isn't accepting text input)
	// or by ctrl+c. y/Y/enter confirms; any other key cancels;
	// double-ctrl+c hard-quits without further confirmation.
	confirmingQuit bool

	// scope is the canonical "what team am I on" state, shared across
	// the chrome header, the Connection pane's scope rows, and the
	// Teams pane's renderer. Mutated only inside Update via the
	// applyWelcome / applyTeamAccess / applyTeamSwitched helpers on
	// activeScope. Panes read from it in their View().
	scope activeScope

	// Pane-specific state.
	monitor    *monitorModel
	newItem    *newItemModel
	team       *teamModel
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
	theme := chrome.NewTheme()
	appKeys := keys.NewAppKeys()
	m := &model{
		client:              client,
		opts:                opts,
		log:                 log,
		state:               connConnecting,
		active:              paneMonitor,
		theme:               theme,
		footer:              chrome.NewFooter(theme),
		appKeys:             appKeys,
		monitorKeys:         keys.NewMonitorKeys(appKeys),
		newItemKeys:         keys.NewNewItemKeys(appKeys),
		newItemWatchingKeys: keys.NewNewItemWatchingKeys(appKeys),
		teamKeys:            keys.NewTeamKeys(appKeys),
		teamCreateKeys:      keys.NewTeamCreateKeys(appKeys),
		connKeys:            keys.NewConnectionKeys(appKeys),
	}
	// Seed the shared active-team state with the user's config-time
	// preference. Welcome / scope.switch_team replies fill in the
	// rest.
	m.scope = activeScope{PreferredID: opts.ActiveTeamID}
	m.monitor = newMonitorModel(client, log)
	m.newItem = newNewItemModel(client)
	m.team = newTeamModel(opts.APIURL, &m.scope, client)
	m.connection = newConnectionModel(opts.LogFilePath, opts.ConfigFilePath, opts.HealthTargets, &m.scope)
	return m
}

func (m *model) Init() tea.Cmd {
	// huh.Form needs its Init() called before it can produce output.
	// Batching the form's Init at the root keeps the New Item pane
	// in a renderable state from the first frame. The Teams pane
	// also kicks off its initial GET /memberships + access-check
	// fetches so the membership table populates while the WS dial
	// proceeds in parallel.
	return tea.Batch(m.newItem.form.Init(), m.team.Init())
}

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
		// Quit-confirmation prompt has the highest precedence — it
		// pre-empts every other binding so the user can answer
		// without being misinterpreted by pane handlers.
		if m.confirmingQuit {
			switch msg.String() {
			case "y", "Y", "enter":
				return m, tea.Quit
			case "ctrl+c":
				// Double-ctrl+c bypasses confirmation as a panic
				// quit.
				return m, tea.Quit
			default:
				m.confirmingQuit = false
				return m, nil
			}
		}

		// Global bindings handled at the root. Plain `tab` is
		// deliberately NOT handled here — it falls through to the
		// active pane (the New Item form needs it for field
		// navigation). Pane switching uses `]` / `[` / `1` / `2` / `3`.
		//
		// `q` and `ctrl+c` both raise the quit-confirmation prompt.
		// `q` is only consumed when the active pane isn't expecting
		// typed text — otherwise the user couldn't type "q" into a
		// form field. `ctrl+c` is always consumed.
		switch keyStr := msg.String(); keyStr {
		case "ctrl+c":
			m.confirmingQuit = true
			return m, nil
		case "q":
			if !m.activePaneAcceptsTextInput() {
				m.confirmingQuit = true
				return m, nil
			}
		}

		// Pane navigation + help toggle. While the create-team modal is
		// open it is fully modal — it captures every key (except ctrl+c)
		// so the plain-key pane bindings (1-4, [, ], ?) land in the name
		// field instead of switching panes (a team named "Q3" must be
		// typable). ctrl+tab / ctrl+shift+tab still switch panes.
		modalOpen := m.teamModalOpen()
		switch {
		case key.Matches(msg, m.appKeys.NextPane) && (!modalOpen || msg.String() == "ctrl+tab"):
			return m, m.activatePane((m.active + 1) % numPanes)
		case key.Matches(msg, m.appKeys.PrevPane) && (!modalOpen || msg.String() == "ctrl+shift+tab"):
			return m, m.activatePane((m.active + numPanes - 1) % numPanes)
		case !modalOpen && key.Matches(msg, m.appKeys.GoMonitor):
			return m, m.activatePane(paneMonitor)
		case !modalOpen && key.Matches(msg, m.appKeys.GoNewItem):
			return m, m.activatePane(paneNewItem)
		case !modalOpen && key.Matches(msg, m.appKeys.GoTeam):
			return m, m.activatePane(paneTeam)
		case !modalOpen && key.Matches(msg, m.appKeys.GoConnection):
			return m, m.activatePane(paneConnection)
		case !modalOpen && key.Matches(msg, m.appKeys.ToggleHelp):
			m.footer.SetShowAll(!m.footer.ShowAll())
			return m, nil
		}

		// Connection-pane-specific keys (only when that pane is
		// active). Refresh triggers an immediate health-check
		// re-poll, throttled by canRunManualHealthCheck so a held
		// key can't hammer external services.
		if m.active == paneConnection && key.Matches(msg, m.connKeys.Refresh) {
			if m.connection.canRunManualHealthCheck() {
				return m, m.connection.dispatchAllChecks(context.Background())
			}
			return m, nil
		}

	case welcomeMsg:
		m.state = connConnected
		m.welcome = msg.Welcome
		m.monitor.applyWelcome(msg.Welcome)
		m.connection.applyWelcome(msg.Welcome)
		// Reconcile the shared scope state with the welcome envelope.
		// Two outcomes: if the user's config preference matches the
		// server scope, fetch team details over REST so the header /
		// Connection / Teams panes have name + role to render. If
		// they differ, fire scope.switch_team to align the server to
		// the user's config preference. Either way, all panes read
		// the resulting state from m.scope.
		teamCmd := m.applyWelcomeToScope(msg.Welcome)
		return m, tea.Batch(
			m.monitor.subscribeAllStreams(),
			teamCmd,
			waitForPush(m.client),
		)

	case connectFailedMsg:
		m.state = connFailed
		m.connErr = msg.Err
		m.connection.setConnError(msg.Err, m.opts.WSURL)
		// Surface the diagnostic pane so the user sees the friendly
		// error explanation immediately, not a cryptic header banner.
		// They can still tab back to Monitor / New Item if they want
		// to read the empty-state placeholders. activatePane also
		// arms the health-check poll loop so the user sees right
		// away whether their internet or just the Truestamp service
		// is the problem.
		return m, m.activatePane(paneConnection)

	case healthCheckTickMsg:
		// Tick fired while the user is on the Connection pane: kick
		// off another wave and re-arm. tickHealthChecks returns nil
		// if the user has tabbed away in the meantime, breaking
		// the chain.
		return m, m.connection.tickHealthChecks(context.Background())

	case healthCheckResultMsg:
		m.connection.applyHealthResult(msg)
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

	case tokenRefreshingMsg:
		// The OAuth access token expired; the client is transparently
		// re-dialling with a refreshed token. The reconnect events that
		// follow drive the visible header status — just keep pumping.
		m.log.Info("oauth access token expired; refreshing session in-band")
		return m, waitForPush(m.client)

	case authFailedMsg:
		// The OAuth session is permanently dead; the client stopped
		// reconnecting. Surface a clear re-login prompt on the
		// Connection pane.
		m.state = connFailed
		m.connErr = fmt.Errorf("OAuth session expired — run `truestamp auth login`, then reopen the console")
		m.connection.setConnError(m.connErr, m.opts.WSURL)
		m.log.Warn("oauth session dead; reconnect stopped", "reason", msg.Reason)
		return m, m.activatePane(paneConnection)

	case pushMsg:
		m.monitor.handlePush(msg)
		m.newItem.handlePush(msg)
		m.connection.recordPush(msg.Event)
		return m, waitForPush(m.client)

	case teamMembershipsMsg:
		// Teams pane owns the membership list lifecycle.
		var cmd tea.Cmd
		m.team, cmd = m.team.Update(msg)
		return m, cmd

	case teamAccessMsg:
		// Source of truth: update m.scope. Auto-focus the Teams pane
		// on access loss so the user lands on the corrective UI.
		m.scope.applyTeamAccess(msg)
		if msg.TeamID != "" && !msg.Found {
			return m, m.activatePane(paneTeam)
		}
		return m, nil

	case teamSwitchedMsg:
		// Source of truth: m.scope. Cached welcome envelope follows
		// (so the Connection pane's user_id / plan rows pick up the
		// post-switch values). The Teams pane needs to refresh its
		// membership list, so route the message through it too.
		m.scope.applyTeamSwitched(msg.Reply)
		if msg.Reply != nil {
			m.welcome.Scope.UserID = msg.Reply.UserID
			m.welcome.Scope.TeamID = msg.Reply.TeamID
			m.welcome.Scope.Plan = msg.Reply.Plan
		}
		var teamCmd tea.Cmd
		m.team, teamCmd = m.team.Update(msg)
		return m, teamCmd

	case teamSwitchFailedMsg:
		var cmd tea.Cmd
		m.team, cmd = m.team.Update(msg)
		return m, cmd

	case teamCreatedMsg, teamCreateFailedMsg:
		// Create-team lifecycle. Routed unconditionally to the Teams pane
		// (not just when active) so an in-flight create still completes if
		// the user tabs away.
		var cmd tea.Cmd
		m.team, cmd = m.team.Update(msg)
		return m, cmd
	}

	// Forward other messages (e.g. form keystrokes) to the active pane.
	var cmd tea.Cmd
	switch m.active {
	case paneMonitor:
		m.monitor, cmd = m.monitor.Update(msg)
	case paneNewItem:
		m.newItem, cmd = m.newItem.Update(msg)
	case paneTeam:
		m.team, cmd = m.team.Update(msg)
	case paneConnection:
		m.connection, cmd = m.connection.Update(msg)
	}
	return m, cmd
}

func (m *model) View() tea.View {
	// Render header + footer first so we know their actual heights.
	// Footer height varies: the help component is one row when
	// short, multiple rows when expanded via `?`. Computing the body
	// area from real heights (rather than fixed Theme constants)
	// means the help-toggle grows downward into the body instead of
	// pushing rows off the bottom of the screen.
	header := m.renderHeader()
	footer := m.renderFooter()

	page := chrome.Page{Width: m.width, Height: m.height, Theme: m.theme}
	bodyW, bodyH := page.BodyArea(header, footer)

	// Connection pane mirrors the Monitor's subscription count so the
	// number appears alongside the rest of the scope info. The Monitor
	// is the source of truth for active subscriptions; the Connection
	// pane is read-only.
	m.connection.setActiveStreams(len(m.monitor.activeStreams()))

	var body string
	switch m.active {
	case paneMonitor:
		body = m.monitor.View(bodyW, bodyH)
	case paneNewItem:
		body = m.newItem.View(bodyW, bodyH)
	case paneTeam:
		body = m.team.View(bodyW, bodyH)
	case paneConnection:
		body = m.connection.View(bodyW, bodyH)
	}

	content := page.Render(header, body, footer)

	v := tea.NewView(content)
	v.AltScreen = true
	// MouseMode left at MouseModeNone so the terminal handles click-drag
	// text selection natively. See the comment on tea.NewProgram above for
	// the rationale.
	return v
}

// applyWelcomeToScope reconciles the welcome envelope with the
// shared activeScope. Two outcomes:
//
//   - When the user's persisted PreferredID matches (or isn't set),
//     fire fetchActiveDetailsCmd to populate name + role over REST.
//
//   - When PreferredID differs from the welcome's scope, fire
//     scope.switch_team to align the server scope to the user's
//     preference. The reply mutates m.scope via teamSwitchedMsg.
//
// AccessLost is reset on each welcome — a successful reconnect
// re-establishes the scope and any prior access-loss banner is no
// longer relevant until proven again.
func (m *model) applyWelcomeToScope(w Welcome) tea.Cmd {
	if m.scope.TeamID != w.Scope.TeamID {
		// Active id changed — clear stale name/role so the renderers
		// don't flash the wrong label until the follow-up arrives.
		m.scope.Name = ""
		m.scope.Personal = false
		m.scope.Role = ""
	}
	m.scope.UserID = w.Scope.UserID
	m.scope.Plan = w.Scope.Plan
	m.scope.TeamID = w.Scope.TeamID
	m.scope.AccessLost = false
	if m.scope.PreferredID != "" && m.scope.PreferredID != w.Scope.TeamID {
		// Silent variant: this alignment is internal bookkeeping
		// (the WS opens on Personal by server policy, we move it to
		// the user's configured team). The user didn't ask for it
		// and shouldn't see a flash-then-disappear "Switched to"
		// notice that would later shift the layout when they press
		// up/down.
		return m.team.silentSwitchCmd(m.scope.PreferredID)
	}
	return m.team.fetchActiveDetailsCmd(w.Scope.TeamID)
}

// activatePane switches the active pane and starts/stops side-effect
// loops that should only run while a particular pane is visible.
//
// Today the only side-effect is the Connection pane's health-check poll
// (every healthCheckPollInterval): we start it on entry and stop it on exit so we
// don't hammer third-party services while the user is on the
// Monitor or New Item pane. Returns the tea.Cmd to fire the first
// poll wave (nil for non-Connection panes).
func (m *model) activatePane(p pane) tea.Cmd {
	if m.active == paneConnection && p != paneConnection {
		m.connection.stopHealthPolling()
	}
	m.active = p
	if p == paneConnection {
		return m.connection.startHealthPolling(context.Background())
	}
	return nil
}

// activeKeyMap returns the help.KeyMap for the currently-focused pane.
// Drives the footer's auto-rendered help row. Some panes have
// state-dependent keymaps (e.g. the New Item pane swaps between
// form-entry bindings and a watch-mode "n: new item" binding once
// the item has been submitted) so this function inspects pane state,
// not just the active pane id.
func (m *model) activeKeyMap() help.KeyMap {
	switch m.active {
	case paneMonitor:
		return m.monitorKeys
	case paneNewItem:
		if m.newItem.state == formWatching {
			return m.newItemWatchingKeys
		}
		return m.newItemKeys
	case paneTeam:
		if m.team.create != nil {
			return m.teamCreateKeys
		}
		return m.teamKeys
	case paneConnection:
		return m.connKeys
	}
	return chrome.EmptyKeyMap{}
}

// activePaneAcceptsTextInput reports whether the active pane is in a
// state where typed characters become field values. Used to gate the
// `q` quit-confirmation so users can still type "q" into form fields
// without raising a confirm prompt.
// teamModalOpen reports whether the create-team modal is open on the Teams
// pane — the one fully-modal state, which captures every key (except ctrl+c)
// so pane-nav digits / brackets reach the name field instead of switching panes.
func (m *model) teamModalOpen() bool {
	return m.active == paneTeam && m.team.create != nil
}

func (m *model) activePaneAcceptsTextInput() bool {
	if m.active == paneNewItem && m.newItem.state == formEntering {
		return true
	}
	return m.teamModalOpen()
}

// renderFooter returns the footer row(s). Default path delegates to
// the bubbles/help-driven Footer component; the quit-confirmation
// prompt overrides it with a single bold row so the user's eye lands
// on the question rather than the (now irrelevant) keybinding list.
func (m *model) renderFooter() string {
	if m.confirmingQuit {
		return m.renderQuitConfirmFooter()
	}
	return m.footer.Render(m.width, m.activeKeyMap())
}

// renderQuitConfirmFooter draws the quit-confirmation prompt over
// the footer area. Single row, theme-themed warning color, padded to
// page width so the chrome alignment doesn't shift when the prompt
// appears.
func (m *model) renderQuitConfirmFooter() string {
	prompt := lipgloss.NewStyle().
		Bold(true).
		Foreground(m.theme.WarnFg).
		Render("Really quit?")

	options := lipgloss.NewStyle().
		Foreground(m.theme.MutedFg).
		Render("y / enter: confirm   any other key: cancel")

	row := prompt + "  " + options
	return lipgloss.NewStyle().
		Padding(0, 1).
		Width(m.width).
		Render(row)
}

func (m *model) renderHeader() string {
	tabs := make([]chrome.TabItem, 0, numPanes)
	for _, p := range []pane{paneMonitor, paneNewItem, paneTeam, paneConnection} {
		tabs = append(tabs, chrome.TabItem{
			Number: int(p) + 1,
			Title:  p.title(),
			Active: p == m.active,
		})
	}

	status, kind := m.statusText()

	return chrome.Render(chrome.HeaderInput{
		Width: m.width,
		Tabs:  tabs,
		// Active team label comes from the shared scope. Empty
		// until welcome arrives — the header just shows the status
		// pill in that case.
		Team:       m.scope.HeaderLabel(),
		Status:     status,
		StatusKind: kind,
		Clock:      m.clockText(),
		Theme:      m.theme,
	})
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
	return m.serverTime.UTC().Format("2006-01-02T15:04:05Z")
}

// statusText returns the header's status string and a StatusKind
// so the chrome can color the right-side pill appropriately. The
// header carries only liveness state — plan tier and the active
// stream count belong on the Connection pane where the user can
// look them up deliberately, not in every-frame ambient chrome.
func (m *model) statusText() (string, chrome.StatusKind) {
	switch m.state {
	case connConnecting:
		return "connecting…", chrome.StatusKindWarn
	case connConnected:
		return "connected", chrome.StatusKindOK
	case connReconnecting:
		return m.reconnectingText(), chrome.StatusKindErr
	case connClosed:
		return "disconnected", chrome.StatusKindErr
	case connFailed:
		return classifyConnError(m.connErr).shortStatus(), chrome.StatusKindErr
	}
	return "", chrome.StatusKindNeutral
}
