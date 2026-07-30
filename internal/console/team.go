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
	"github.com/truestamp/truestamp-cli/internal/config"
	"github.com/truestamp/truestamp-cli/internal/teams"
	"github.com/truestamp/truestamp-cli/internal/ui"
	"github.com/truestamp/truestamp-cli/internal/wschannel"
)

// init wires the on-disk persistence helper. Test packages can
// reassign persistTeamSelection before constructing a teamModel to
// avoid touching the user's real config file.
func init() {
	persistTeamSelection = config.SetTeam
}

// teamPaneState enumerates the pane's render branches.
type teamPaneState int

const (
	teamPaneLoading teamPaneState = iota
	teamPaneReady
	teamPaneError
)

// teamSwitcher is the small slice of wschannel.Client the pane needs.
// Defined as an interface so unit tests can inject a stub without
// spinning up a real WebSocket.
type teamSwitcher interface {
	SwitchTeam(ctx context.Context, teamID string) (*wschannel.SwitchTeamReply, error)
}

// teamModel renders the membership list and the currently-active
// team card, and dispatches scope.switch_team over the live
// WebSocket on `enter`. Persists the new active team to config.toml on
// successful switches.
//
// Active vs preferred team (both live on the shared activeScope the
// pane reads through m.scope; see scope.go):
//
//   - scope.TeamID is the team the WebSocket scope is currently
//     bound to. Sourced from the welcome envelope at session start
//     and from scope.switch_team replies thereafter. This is the
//     value rendered in the "Active team" card and marked with ★ in
//     the membership table — it tracks server reality.
//
//   - scope.PreferredID is the team id stored in the user's config.toml
//     at console launch. On welcome, if it differs from
//     welcome.scope.team_id, the root model automatically fires
//     scope.switch_team to align the server scope to the user's
//     preference. So a user whose config says "Engineering" doesn't
//     have to manually switch every time they open the console — the
//     console catches up to where they left off.
type teamModel struct {
	apiURL string
	client teamSwitcher

	// scope points at the root model's activeScope so the pane
	// always renders against the canonical state — no local copies
	// of the active team's id / name / role.
	scope *activeScope

	state         teamPaneState
	memberships   []teamRow
	listErr       error
	listFetchedAt time.Time

	// Cursor-driven row selection. Index into memberships.
	cursor int

	// notice: a transient one-line status (success/failure of the
	// last switch, etc). Cleared on the next user input.
	notice      string
	noticeError bool

	// create is the open create-team modal (nil when closed). While
	// non-nil the pane renders the modal instead of the membership list
	// and routes input to it.
	create *teamCreateModel
}

func newTeamModel(apiURL string, scope *activeScope, client teamSwitcher) *teamModel {
	return &teamModel{
		apiURL: apiURL,
		scope:  scope,
		client: client,
		state:  teamPaneLoading,
	}
}

// fetchActiveDetailsCmd loads the active team's name + role for the
// shared scope state. Used by the root model after a welcome arrives
// (when no scope.switch_team was needed) and by the Teams pane on
// `r` refresh. The resulting teamAccessMsg flows through the root
// Update, which mutates m.scope; this pane just reads from it.
func (m *teamModel) fetchActiveDetailsCmd(teamID string) tea.Cmd {
	if teamID == "" {
		return func() tea.Msg { return teamAccessMsg{TeamID: "", Found: true} }
	}
	apiURL := m.apiURL
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		cfg := teams.Config{APIURL: apiURL, Team: teamID}
		t, err := teams.GetTeam(ctx, cfg, teamID)
		if err != nil {
			return teamAccessMsg{TeamID: teamID, Found: false, Err: err}
		}
		role, _ := teams.GetMyRoleOnTeam(ctx, cfg, teamID)
		return teamAccessMsg{
			TeamID:   teamID,
			Found:    true,
			Name:     t.Name,
			Role:     role,
			Personal: t.Personal,
		}
	}
}

// fetchMembershipsCmd is the tea.Cmd run on init / refresh. The
// underlying call is HTTP-only, so callers don't need to thread the
// WS client into it. Errors flow back through teamMembershipsMsg.
func (m *teamModel) fetchMembershipsCmd() tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		ms, err := teams.ListMyMemberships(ctx, teams.Config{
			APIURL: m.apiURL, Team: m.scope.TeamID,
		})
		if err != nil {
			return teamMembershipsMsg{Err: err}
		}
		rows := make([]teamRow, 0, len(ms))
		for _, mem := range ms {
			r := teamRow{TeamID: mem.TeamID, Role: mem.Role}
			if mem.Team != nil {
				r.Name = mem.Team.Name
				r.Personal = mem.Team.Personal
			}
			rows = append(rows, r)
		}
		// Personal team first (exactly one per user), then privilege
		// rank Owner → Admin → Member → Viewer, then alphabetical
		// within each rank. Matches the order in `truestamp team
		// list` so the user sees a uniform layout across the CLI
		// and console.
		sort.SliceStable(rows, func(i, j int) bool {
			if rows[i].Personal != rows[j].Personal {
				return rows[i].Personal
			}
			ri, rj := teams.PrivilegeRank(rows[i].Role), teams.PrivilegeRank(rows[j].Role)
			if ri != rj {
				return ri < rj
			}
			return rows[i].Name < rows[j].Name
		})
		return teamMembershipsMsg{Memberships: rows}
	}
}

// Init kicks off the membership-list fetch. The active-team
// details fetch is fired by the root model after the welcome
// envelope arrives — see model.applyWelcomeToScope in app.go.
func (m *teamModel) Init() tea.Cmd {
	return m.fetchMembershipsCmd()
}

// Update handles tea messages routed to the team pane. The shared
// scope state (m.scope) is mutated by the root model in app.go; this
// pane only mutates pane-local state (cursor, notice, memberships,
// state machine).
func (m *teamModel) Update(msg tea.Msg) (*teamModel, tea.Cmd) {
	// Create-modal sub-loop: while the modal is open it owns keyboard +
	// cursor-blink input. Create-lifecycle and membership messages are
	// still handled there so the list underneath is fresh when it closes.
	if m.create != nil {
		return m.updateCreating(msg)
	}

	switch msg := msg.(type) {
	case teamMembershipsMsg:
		if msg.Err != nil {
			m.state = teamPaneError
			m.listErr = msg.Err
			return m, nil
		}
		m.memberships = msg.Memberships
		m.state = teamPaneReady
		m.listErr = nil
		m.listFetchedAt = time.Now()
		// Position the cursor on the active row when present.
		m.cursor = 0
		for i, r := range m.memberships {
			if r.TeamID == m.scope.TeamID {
				m.cursor = i
				break
			}
		}
		return m, nil

	case teamSwitchedMsg:
		// Silent switches (the auto-alignment fired by
		// applyWelcomeToScope) suppress the in-pane notice — we
		// don't want a "Switched to X" line to appear and then
		// disappear on the user's first up/down keypress, which
		// would visibly shift the table. User-initiated switches
		// (Silent=false) still get the confirmation banner.
		if !msg.Silent && msg.Reply != nil {
			m.notice = "Switched to " + msg.Reply.TeamName
			m.noticeError = false
		}
		// Refresh the membership list so any name updates pulled
		// in from the new scope are reflected.
		return m, m.fetchMembershipsCmd()

	case teamSwitchFailedMsg:
		m.notice = fmt.Sprintf("%s: %s", msg.Code, msg.Message)
		m.noticeError = true
		return m, nil

	case teamCreatedMsg:
		// Reaches the top level only when the modal is already closed (an
		// open modal handles this in updateCreating). Honors the
		// "create completes even if you tab away" contract: confirm +
		// refresh the membership list.
		m.notice = "Created " + teamCreatedName(msg.Team) + "."
		m.noticeError = false
		return m, m.fetchMembershipsCmd()

	case teamCreateFailedMsg:
		m.notice = "Create failed: " + friendlyCreateError(msg.Err)
		m.noticeError = true
		return m, nil

	case tea.KeyPressMsg:
		return m.handleKey(msg)
	}
	return m, nil
}

func (m *teamModel) handleKey(msg tea.KeyPressMsg) (*teamModel, tea.Cmd) {
	switch s := msg.String(); s {
	case "up", "k":
		if m.cursor > 0 {
			m.cursor--
		}
		m.notice = ""
		return m, nil
	case "down", "j":
		if m.cursor < len(m.memberships)-1 {
			m.cursor++
		}
		m.notice = ""
		return m, nil
	case "r":
		m.notice = ""
		m.state = teamPaneLoading
		// Refresh: re-pull memberships and re-fetch active team
		// details. The WS scope itself doesn't change here.
		return m, tea.Batch(m.fetchMembershipsCmd(), m.fetchActiveDetailsCmd(m.scope.TeamID))
	case "enter":
		return m.handleSetActive()
	case "c":
		m.notice = ""
		return m, m.openCreate()
	}
	return m, nil
}

func (m *teamModel) handleSetActive() (*teamModel, tea.Cmd) {
	if m.cursor < 0 || m.cursor >= len(m.memberships) {
		return m, nil
	}
	row := m.memberships[m.cursor]
	if row.TeamID == m.scope.TeamID {
		m.notice = "Already the active team."
		m.noticeError = false
		return m, nil
	}
	// Fire immediately on `enter`. The cursor lives on the row the user
	// is intentionally looking at, so a single press is unambiguous —
	// no confirmation step.
	m.notice = "Switching to " + teamRowDisplayName(row) + "…"
	m.noticeError = false
	return m, m.switchCmd(row.TeamID)
}

// switchCmd packages the live-channel scope.switch_team push +
// `config.SetTeam` persistence into one tea.Cmd. Used for user-
// initiated switches (the resulting teamSwitchedMsg has Silent=false
// so the pane shows a confirmation notice).
func (m *teamModel) switchCmd(targetID string) tea.Cmd {
	return m.doSwitchCmd(targetID, false)
}

// silentSwitchCmd is the same as switchCmd but emits a Silent
// teamSwitchedMsg — used by applyWelcomeToScope to align the
// server-side default Personal scope to the user's config
// preference without flashing a "Switched to X" notice on every
// console launch.
func (m *teamModel) silentSwitchCmd(targetID string) tea.Cmd {
	return m.doSwitchCmd(targetID, true)
}

func (m *teamModel) doSwitchCmd(targetID string, silent bool) tea.Cmd {
	if m.client == nil {
		return func() tea.Msg {
			return teamSwitchFailedMsg{
				TeamID:  targetID,
				Code:    "internal",
				Message: "no live WebSocket client",
			}
		}
	}
	apiURL := m.apiURL
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		reply, err := m.client.SwitchTeam(ctx, targetID)
		if err != nil {
			if cerr, ok := err.(*wschannel.ChannelError); ok {
				return teamSwitchFailedMsg{
					TeamID:  targetID,
					Code:    string(cerr.Code),
					Message: cerr.Message,
				}
			}
			return teamSwitchFailedMsg{
				TeamID:  targetID,
				Code:    "transport",
				Message: err.Error(),
			}
		}

		// Persist to config on success only — see the plan for the
		// rationale (don't write a team id we're not actually scoped
		// to). The config package is imported inside the package
		// boundary because the team pane lives in internal/console.
		if perr := persistTeamSelection(targetID); perr != nil {
			// The switch worked server-side; we just couldn't write
			// the file. The failure isn't surfaced today — the user
			// sees the normal success notice (degraded success) —
			// and the next launch reverts to the config-file team,
			// but the current session's state is already on the new
			// team.
			return teamSwitchedMsg{Reply: switchReplyForPersistFail(reply, apiURL, perr), Silent: silent}
		}
		return teamSwitchedMsg{Reply: switchReplyFromWire(reply), Silent: silent}
	}
}

// switchReplyFromWire converts a wschannel.SwitchTeamReply to the
// pane-internal teamSwitchReply. Keeps wschannel out of messages.go.
func switchReplyFromWire(r *wschannel.SwitchTeamReply) *teamSwitchReply {
	if r == nil {
		return nil
	}
	return &teamSwitchReply{
		UserID:         r.Scope.UserID,
		TeamID:         r.Scope.TeamID,
		Plan:           r.Scope.Plan,
		TeamName:       r.Team.Name,
		TeamPersonal:   r.Team.Personal,
		OwnershipModel: r.Team.OwnershipModel,
		Role:           r.Role,
		CatalogStreams: r.Streams.Catalog,
		ItemStreams:    r.Streams.Items,
	}
}

// switchReplyForPersistFail returns the same envelope with a notice
// embedded (we don't have a separate field, so today the persistence
// failure is silently ignored at the message level — the user sees
// the success banner unchanged. This is a degraded-success case).
func switchReplyForPersistFail(r *wschannel.SwitchTeamReply, _ string, _ error) *teamSwitchReply {
	return switchReplyFromWire(r)
}

func teamRowDisplayName(r teamRow) string {
	name := r.Name
	if name == "" {
		name = r.TeamID
	}
	if r.Personal {
		name += " (personal)"
	}
	return name
}

// persistTeamSelection is a function variable so the test suite can
// stub out the on-disk write. Defaults to the real `config.SetTeam`
// at runtime; assigned in the package init below.
var persistTeamSelection func(string) error

// View renders the active pane to the body area: the create modal when
// open, otherwise the active-team card, an optional access-loss banner and
// notice, the "+ New team" hint, and the memberships table.
func (m *teamModel) View(width, height int) string {
	if m.create != nil {
		return paneStyle(width, height).Render(m.create.render(width))
	}

	var sections []string
	sections = append(sections, m.renderActiveSection())
	if m.scope.AccessLost && m.scope.TeamID != "" {
		sections = append(sections, m.renderAccessLoss())
	}
	if m.notice != "" {
		sections = append(sections, m.renderNotice())
	}
	sections = append(sections, m.renderCreateHint())
	switch m.state {
	case teamPaneLoading:
		sections = append(sections, m.renderLoading())
	case teamPaneError:
		sections = append(sections, m.renderListError())
	case teamPaneReady:
		sections = append(sections, m.renderMemberships())
	}
	return paneStyle(width, height).Render(strings.Join(sections, "\n\n"))
}

// renderCreateHint shows the "+ New team" affordance and its `c` mnemonic.
func (m *teamModel) renderCreateHint() string {
	return teamCreateBtn.Render("+ New team") +
		teamFaintStyle.Render("  press c to create a new team")
}

// activeMembership returns the current row for the active team id, if
// any. Used to drive the active-team card.
func (m *teamModel) activeMembership() *teamRow {
	if m.scope.TeamID == "" {
		return nil
	}
	for i := range m.memberships {
		if m.memberships[i].TeamID == m.scope.TeamID {
			return &m.memberships[i]
		}
	}
	return nil
}

func (m *teamModel) renderActiveSection() string {
	title := teamSectionTitle.Render("Active team")
	if m.scope.TeamID == "" {
		body := teamFaintStyle.Render(
			"  No team is configured. Pick one below with `enter`, or run\n  `truestamp team set` outside the console.")
		return title + "\n" + body
	}

	rows := [][]string{
		{"id", m.scope.TeamID},
	}
	// Prefer the shared scope's name/role since they update from
	// switch replies and access checks. Fall back to the membership
	// list's row when the scope hasn't been populated yet.
	if m.scope.Name != "" {
		display := m.scope.Name
		if m.scope.Personal {
			display += " (personal)"
		}
		rows = append(rows, []string{"name", display})
	} else if row := m.activeMembership(); row != nil {
		rows = append(rows, []string{"name", teamRowDisplayName(*row)})
	}
	if m.scope.Role != "" {
		rows = append(rows, []string{"role", teams.FormatRole(m.scope.Role)})
	} else if row := m.activeMembership(); row != nil && row.Role != "" {
		rows = append(rows, []string{"role", teams.FormatRole(row.Role)})
	}
	tbl := teamKVTable(rows)
	return title + "\n" + tbl.Render()
}

func (m *teamModel) renderAccessLoss() string {
	body := strings.Builder{}
	body.WriteString(teamErrTitle.Render("Membership lost"))
	body.WriteString("\n\n  ")
	body.WriteString(teamErrHeadline.Render(
		"You no longer have access to team " + m.scope.TeamID + "."))
	body.WriteString("\n  ")
	body.WriteString(teamFaintStyle.Render(
		"Press `enter` on a team below to switch, or quit and run `truestamp team unset`."))
	return body.String()
}

func (m *teamModel) renderNotice() string {
	style := teamNoticeOK
	if m.noticeError {
		style = teamNoticeBad
	}
	return "  " + style.Render(m.notice)
}

func (m *teamModel) renderLoading() string {
	return teamSectionTitle.Render("Memberships") + "\n  " +
		teamFaintStyle.Render("· loading your teams…")
}

func (m *teamModel) renderListError() string {
	body := teamSectionTitle.Render("Memberships") + "\n  " +
		teamFailStyle.Render("Could not load memberships.")
	if m.listErr != nil {
		body += "\n  " + teamFaintStyle.Render(m.listErr.Error())
	}
	body += "\n  " + teamFaintStyle.Render("Press `r` to retry.")
	return body
}

func (m *teamModel) renderMemberships() string {
	if len(m.memberships) == 0 {
		return teamSectionTitle.Render("Memberships") + "\n  " +
			teamFaintStyle.Render("No teams yet — press c to create one.")
	}

	// Two leading one-cell marker columns keep rows from shifting:
	// column 0 = cursor (▸), column 1 = active-team star (★).
	rows := make([][]string, 0, len(m.memberships)+1)
	rows = append(rows, []string{"", "", "NAME", "ROLE", "TEAM ID"})
	for i, r := range m.memberships {
		cursor := " "
		if i == m.cursor {
			cursor = "▸"
		}
		star := " "
		if r.TeamID == m.scope.TeamID {
			star = "★"
		}
		name := r.Name
		if name == "" {
			name = r.TeamID
		}
		if r.Personal {
			name += " (personal)"
		}
		rows = append(rows, []string{cursor, star, name, teams.FormatRole(r.Role), r.TeamID})
	}

	tbl := ui.CompactTable().
		StyleFunc(func(row, col int) lipgloss.Style {
			base := lipgloss.NewStyle().PaddingLeft(2).PaddingRight(1)
			if row == 0 {
				return base.Foreground(ui.Label).Bold(true)
			}
			if row-1 == m.cursor {
				return base.Foreground(ui.Accent).Bold(col == 0)
			}
			return base.Foreground(ui.Value)
		}).
		Rows(rows...)

	return teamSectionTitle.Render("Memberships") + "\n" + tbl.Render()
}

// teamKVTable mirrors connection.go's keyValueTable but stays
// pane-local so the team pane can evolve its styling without
// touching the connection pane.
func teamKVTable(rows [][]string) *ltable.Table {
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
				return teamKeyStyle.PaddingLeft(2).PaddingRight(2)
			case 1:
				return teamValueStyle
			}
			return lipgloss.NewStyle()
		})
}

var (
	teamSectionTitle = lipgloss.NewStyle().Bold(true).Foreground(ui.Accent)
	teamKeyStyle     = lipgloss.NewStyle().Foreground(ui.Label)
	teamValueStyle   = lipgloss.NewStyle().Foreground(ui.Value)
	teamFaintStyle   = lipgloss.NewStyle().Foreground(ui.Dim)
	teamFailStyle    = lipgloss.NewStyle().Foreground(ui.Red)
	teamErrTitle     = lipgloss.NewStyle().Bold(true).Foreground(ui.Red)
	teamErrHeadline  = lipgloss.NewStyle().Foreground(ui.Red)
	teamNoticeOK     = lipgloss.NewStyle().Foreground(ui.Green)
	teamNoticeBad    = lipgloss.NewStyle().Foreground(ui.Red)
)
