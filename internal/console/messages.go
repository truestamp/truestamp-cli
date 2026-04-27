// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: MIT

// Package console implements the interactive `truestamp console` TUI: a
// multi-pane Bubble Tea application that holds an authenticated WebSocket
// connection to the Truestamp backend and lets the user observe live
// events, create timestamped items, and inspect connection diagnostics.
package console

import (
	"encoding/json"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/truestamp/truestamp-cli/internal/wschannel"
)

// Channel topic identifiers shared between the wschannel client and the
// pane code. The console multiplexes these on a single WebSocket.
const (
	lobbyTopic = "console:lobby"
	clockTopic = "console:clock"
)

// pushMsg wraps a server-initiated WebSocket event as a tea.Msg.
type pushMsg struct {
	Topic   string
	Event   string
	Payload json.RawMessage
}

// clockTickMsg is dispatched whenever the server's TimeBroadcaster emits
// a one-second tick on the `console:clock` topic. Carries the parsed
// server time for header rendering.
type clockTickMsg struct {
	At time.Time
}

// closedMsg is dispatched when the WebSocket reader loop terminates,
// either via an explicit Close() or a transport error.
type closedMsg struct {
	Err error
}

// rejoinedMsg fires whenever the client successfully re-joins a topic
// after an automatic reconnect. Carries the welcome envelope so the
// pane code can resync any in-channel state the server doesn't
// remember (subscriptions, item watches).
type rejoinedMsg struct {
	Topic   string
	Welcome json.RawMessage
}

// reconnectingMsg fires before every dial attempt while the session is
// down. Drives the header countdown ("reconnecting in 7s…") and is the
// trigger for the periodic outage markers in the Monitor pane.
type reconnectingMsg struct {
	Attempt       int
	NextAttemptAt time.Time
}

// reconnectTickMsg fires once per second while the session is down. It
// drives the countdown re-render and decides whether enough time has
// passed since the last outage marker to inject a fresh one.
type reconnectTickMsg struct{}

// welcomeMsg is dispatched after a successful Connect — it carries the
// initial welcome envelope from the server (scope summary, stream catalog,
// server build info).
type welcomeMsg struct {
	Welcome Welcome
}

// connectFailedMsg is dispatched if Connect itself fails.
type connectFailedMsg struct {
	Err error
}

// healthCheckResultMsg carries one completed health-check result up
// to the model. The connection pane batches these into its result
// table; ordering is preserved by the embedded Index field so the
// renderer can update rows in place rather than re-sorting on every
// arrival (rendering sort happens at View time).
type healthCheckResultMsg struct {
	Index  int
	Result healthResult
}

// healthCheckTickMsg drives the 30-second poll loop while the
// Connection pane is the active pane. The pane re-arms the next
// tick on each fire; tabbing away cancels the loop because the
// pane stops returning the tick command.
type healthCheckTickMsg struct{}

// Welcome is the parsed shape of the join reply for `console:lobby`.
type Welcome struct {
	Scope   ScopeSummary `json:"scope"`
	Streams []StreamMeta `json:"streams"`
	Server  ServerInfo   `json:"server"`
}

// ScopeSummary mirrors the `scope` map in the welcome envelope.
type ScopeSummary struct {
	UserID string `json:"user_id"`
	TeamID string `json:"team_id"`
	Plan   string `json:"plan"`
}

// StreamMeta is one entry in the stream catalog.
type StreamMeta struct {
	ID    string `json:"id"`
	Scope string `json:"scope"`
	Title string `json:"title"`
}

// ServerInfo holds the server build metadata included in the welcome.
type ServerInfo struct {
	Version string `json:"version"`
}

// waitForPush is the bridge from the WS reader goroutine to the Tea
// Update loop: each call yields one tea.Msg per server push, and
// returning the next call from Update forms an unbounded queue.
//
// When the channel is closed (clean shutdown or transport error), this
// emits closedMsg so the UI can mark the connection lost.
func waitForPush(client *wschannel.Client) tea.Cmd {
	return func() tea.Msg {
		p, ok := <-client.Pushes()
		if !ok {
			return closedMsg{}
		}

		// Server-time ticks parse into a typed message so the root
		// model doesn't have to redo JSON decode for every header
		// render.
		if p.Topic == clockTopic && p.Event == "tick" {
			var t struct {
				At string `json:"at"`
			}
			if err := json.Unmarshal(p.Payload, &t); err == nil {
				if at, terr := time.Parse(time.RFC3339Nano, t.At); terr == nil {
					return clockTickMsg{At: at}
				}
			}
		}

		// Synthetic event that fires after each successful rejoin
		// on reconnect — the pane code uses it to replay any
		// in-channel subscriptions the server has forgotten.
		if p.Event == wschannel.ReconnectedEvent {
			return rejoinedMsg{Topic: p.Topic, Welcome: p.Payload}
		}

		if p.Event == wschannel.ReconnectingEvent {
			var s struct {
				Attempt       int    `json:"attempt"`
				NextAttemptAt string `json:"next_attempt_at"`
			}
			if err := json.Unmarshal(p.Payload, &s); err == nil {
				at, _ := time.Parse(time.RFC3339Nano, s.NextAttemptAt)
				return reconnectingMsg{Attempt: s.Attempt, NextAttemptAt: at}
			}
		}

		return pushMsg{Topic: p.Topic, Event: p.Event, Payload: p.Payload}
	}
}
