// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: MIT

package console

// activeScope is the canonical source of truth for "what team is the
// console currently scoped to" inside the TUI. Every pane that needs
// to render the active team, the chrome header label, the
// Connection pane's scope rows, the Teams pane's ★ marker and active-
// team card, reads from this single struct on the root model
// instead of carrying its own copy.
//
// Update points are explicit and centralized in app.go's Update:
//
//   - welcomeMsg → model.applyWelcomeToScope sets UserID / Plan /
//     TeamID (and clears AccessLost), then fires either
//     fetchActiveDetailsCmd (REST) or a silent scope.switch_team
//     (channel) to reconcile.
//   - teamAccessMsg → applies Name / Role / Personal for the team
//     id we're scoped to; flips AccessLost on a 403/404.
//   - teamSwitchedMsg → applies the post-switch team data verbatim.
//
// Panes never mutate this struct directly, they read it during their
// View() to render the right thing.
type activeScope struct {
	// UserID + Plan come from the welcome envelope at session start.
	UserID string
	Plan   string

	// TeamID is the team the WebSocket scope is currently bound to.
	// Sourced from welcome.Scope.TeamID and from scope.switch_team
	// reply payloads.
	TeamID string

	// PreferredID is the team id stored in cfg.Team at console
	// launch, seeded once in newModel. On welcome,
	// applyWelcomeToScope auto-fires scope.switch_team to align
	// TeamID to PreferredID when they differ.
	PreferredID string

	// Name / Personal / Role are populated alongside TeamID from
	// scope.switch_team replies (always) and from the post-welcome
	// REST follow-up (when no switch was needed). Empty until the
	// reconciliation completes; renderers fall back gracefully.
	Name     string
	Personal bool
	Role     string

	// AccessLost is true when the configured TeamID is no longer
	// readable by this API key (membership revoked mid-session). The
	// Teams pane auto-activates and shows a corrective banner; the
	// Connection pane colors the team row red.
	AccessLost bool
}

// HeaderLabel returns a short label suitable for the chrome header's
// faint "active team" hint. Personal team gets a stable label; named
// teams use their server-side name. Empty when we don't yet have a
// resolved name (welcome hasn't arrived, or the scope follow-up
// hasn't completed).
func (s *activeScope) HeaderLabel() string {
	if s == nil {
		return ""
	}
	if s.Personal {
		return "Personal"
	}
	return s.Name
}

// applyTeamAccess copies the relevant fields out of a teamAccessMsg
// when the message refers to the team we're currently scoped to.
// Older messages for stale team ids (e.g. the active id was switched
// while a fetch was in flight) are ignored.
func (s *activeScope) applyTeamAccess(msg teamAccessMsg) {
	if msg.TeamID == "" || msg.TeamID != s.TeamID {
		return
	}
	if !msg.Found {
		s.AccessLost = true
		return
	}
	s.AccessLost = false
	s.Name = msg.Name
	s.Personal = msg.Personal
	s.Role = msg.Role
}

// applyTeamSwitched records the result of a successful
// scope.switch_team push. The reply is authoritative, TeamID +
// metadata replace whatever was there.
func (s *activeScope) applyTeamSwitched(reply *teamSwitchReply) {
	if reply == nil {
		return
	}
	s.TeamID = reply.TeamID
	s.Name = reply.TeamName
	s.Personal = reply.TeamPersonal
	s.Role = reply.Role
	s.Plan = reply.Plan
	s.UserID = reply.UserID
	s.AccessLost = false
}
