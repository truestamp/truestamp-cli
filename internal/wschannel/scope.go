// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: MIT

package wschannel

import (
	"context"
	"encoding/json"
	"fmt"
)

// ChannelErrorCode is one of the typed error codes the server returns
// in a `phx_reply` with status="error". Defined as constants here so
// CLI callers can pattern-match on them without sprinkling string
// literals.
type ChannelErrorCode string

const (
	// ChannelErrCodeForbidden — the actor is not a member of the target
	// (used by scope.switch_team for non-member targets).
	ChannelErrCodeForbidden ChannelErrorCode = "forbidden"

	// ChannelErrCodeNotFound — the target resource (team, item, …) does
	// not exist.
	ChannelErrCodeNotFound ChannelErrorCode = "not_found"

	// ChannelErrCodeNoop — the requested change has no effect (e.g.
	// switching to the team already in scope). Treated as "success-
	// equivalent" by callers that just want to confirm a desired
	// state, but surfaced as an error for those that want to know.
	ChannelErrCodeNoop ChannelErrorCode = "noop"

	// ChannelErrCodeInvalid — the inbound payload was malformed.
	ChannelErrCodeInvalid ChannelErrorCode = "invalid"

	// ChannelErrCodeRateLimited — the channel's @cmd_limit gate fired.
	ChannelErrCodeRateLimited ChannelErrorCode = "rate_limited"
)

// ChannelError carries the typed error envelope from a Phoenix Channel
// `phx_reply` whose status was "error". The Code field is one of the
// ChannelErrCode* constants above; Message is the server-provided
// human-readable detail.
type ChannelError struct {
	Code    ChannelErrorCode
	Message string
}

func (e *ChannelError) Error() string {
	if e.Message != "" {
		return fmt.Sprintf("%s: %s", e.Code, e.Message)
	}
	return string(e.Code)
}

// SwitchTeamReply is the success envelope of a `scope.switch_team`
// channel push. Mirrors the join-time welcome envelope so callers can
// overwrite their cached welcome state from this struct directly.
type SwitchTeamReply struct {
	Scope   ScopeBlock        `json:"scope"`
	Team    SwitchedTeam      `json:"team"`
	Role    string            `json:"role"`
	Streams SwitchedStreamSet `json:"streams"`
}

// ScopeBlock is the subset of the welcome envelope's `scope` block the
// client reads. Mirrors the server's
// TruestampWeb.ConsoleChannel.welcome_envelope/1 plus the new team_id.
type ScopeBlock struct {
	UserID string `json:"user_id"`
	TeamID string `json:"team_id"`
	Plan   string `json:"plan"`
}

// SwitchedTeam carries the new team's basic attributes so the client
// doesn't need a follow-up REST call to render the post-switch UI.
type SwitchedTeam struct {
	ID             string `json:"id"`
	Name           string `json:"name"`
	Personal       bool   `json:"personal"`
	OwnershipModel string `json:"ownership_model"`
}

// SwitchedStreamSet is the post-switch active stream split. `Catalog`
// stays subscribed (rebound against the new tenant); `Items` are item
// watches preserved as-is across the switch.
type SwitchedStreamSet struct {
	Catalog []string `json:"catalog"`
	Items   []string `json:"items"`
}

// SwitchTeam pushes `scope.switch_team` on `console:lobby` and waits
// for the reply. Returns the parsed welcome-shaped reply on success;
// on failure the error is a *ChannelError when the server replied
// with a typed code, or a transport error otherwise.
//
// Callers should typically Update their local welcome envelope from
// the SwitchTeamReply on success, then notify any pane state that
// depends on the active team (Connection pane scope rows, Monitor
// pane catalog stream toggles, etc.).
//
// On *ChannelError{Code: ChannelErrCodeNoop} the server confirmed the
// scope was already on the target team — callers may treat this as a
// success (state already matches the desired state) or surface a
// "no change" hint.
const switchTeamLobbyTopic = "console:lobby"
const switchTeamEvent = "scope.switch_team"

func (c *Client) SwitchTeam(ctx context.Context, teamID string) (*SwitchTeamReply, error) {
	if teamID == "" {
		return nil, fmt.Errorf("team_id is required")
	}
	reply, err := c.Push(ctx, switchTeamLobbyTopic, switchTeamEvent, map[string]string{
		"team_id": teamID,
	})
	if err != nil {
		return nil, err
	}
	if reply.Status != "ok" {
		return nil, decodeChannelError(reply.Response)
	}
	var out SwitchTeamReply
	if err := json.Unmarshal(reply.Response, &out); err != nil {
		return nil, fmt.Errorf("decode scope.switch_team reply: %w", err)
	}
	return &out, nil
}

// decodeChannelError pulls `code` and `message` out of an error reply
// envelope. Falls back to a code of "unknown" with the raw payload as
// the message when the shape isn't recognized so the user still sees
// useful diagnostic text.
func decodeChannelError(raw json.RawMessage) error {
	var env struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	}
	if err := json.Unmarshal(raw, &env); err == nil && env.Code != "" {
		return &ChannelError{Code: ChannelErrorCode(env.Code), Message: env.Message}
	}
	// Unrecognized shape — surface the raw bytes truncated for safety.
	body := string(raw)
	if len(body) > 200 {
		body = body[:200]
	}
	return &ChannelError{Code: "unknown", Message: body}
}
