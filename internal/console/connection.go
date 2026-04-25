// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: MIT

package console

import (
	"fmt"
	"sort"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
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

func (m *connectionModel) View(width, height int) string {
	var sb strings.Builder

	sb.WriteString("Scope\n")
	fmt.Fprintf(&sb, "  user_id : %s\n", m.welcome.Scope.UserID)
	fmt.Fprintf(&sb, "  team_id : %s\n", m.welcome.Scope.TeamID)
	fmt.Fprintf(&sb, "  plan    : %s\n", m.welcome.Scope.Plan)
	fmt.Fprintf(&sb, "  server  : %s\n\n", m.welcome.Server.Version)

	if !m.connectedAt.IsZero() {
		fmt.Fprintf(&sb, "Connected for %s\n\n",
			time.Since(m.connectedAt).Truncate(time.Second))
	}

	sb.WriteString("Push counts by event\n")
	if len(m.pushCounts) == 0 {
		sb.WriteString("  (none yet)\n")
	} else {
		eventNames := make([]string, 0, len(m.pushCounts))
		for event := range m.pushCounts {
			eventNames = append(eventNames, event)
		}
		sort.Strings(eventNames)
		for _, event := range eventNames {
			fmt.Fprintf(&sb, "  %-20s %d\n", event, m.pushCounts[event])
		}
	}

	sb.WriteString("\nReconnects\n")
	if m.reconnectCount == 0 {
		sb.WriteString("  none since session start\n")
	} else {
		fmt.Fprintf(&sb, "  count    %d\n", m.reconnectCount)
		fmt.Fprintf(&sb, "  downtime %s\n",
			m.totalDowntime.Truncate(time.Second))
	}

	if m.logFilePath != "" {
		fmt.Fprintf(&sb, "\nLogs\n  %s\n", m.logFilePath)
		sb.WriteString("  (tail -f for live transport diagnostics)\n")
	}

	return paneStyle(width, height).Render(sb.String())
}
