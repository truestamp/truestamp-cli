// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: MIT

package console

import (
	"errors"
	"strings"
	"testing"
)

// TestRenderErrorSectionShowsFriendlyContent verifies that, given a
// classifiable connect failure, the Connection pane's error section
// surfaces (1) a kind-specific headline, (2) the URL we tried, (3) at
// least one "what to try" hint, and (4) the raw error for support
// purposes. ANSI color codes are stripped before assertions so the
// test isn't sensitive to the active theme.
func TestRenderErrorSectionShowsFriendlyContent(t *testing.T) {
	t.Parallel()

	m := newConnectionModel("/tmp/log", "/home/alice/.config/truestamp/config.toml", nil)
	m.setConnError(
		errors.New(`dial: failed to WebSocket dial: failed to send handshake request: Get "ws://localhost:9999/console/websocket": dial tcp [::1]:9999: connect: connection refused`),
		"ws://localhost:9999/console/websocket",
	)

	out := stripANSI(m.renderErrorSection(120))

	wantParts := []string{
		"Connection error",
		"The server refused the connection.",
		"ws://localhost:9999/console/websocket",
		"/home/alice/.config/truestamp/config.toml",
		"open this file in any text editor",
		"What to try:",
		"connected to the internet",
		"Raw error:",
		"connection refused",
	}
	for _, want := range wantParts {
		if !strings.Contains(out, want) {
			t.Errorf("error section missing %q\n--- rendered ---\n%s", want, out)
		}
	}

	// End-user friendliness: the hints should NOT carry developer
	// jargon that a non-technical user wouldn't recognize.
	jargonNoNos := []string{
		"task serve",
		"task serve-preview",
		"--base-url",
		"DNS",
		"TLS handshake",
		"WebSocket upgrade",
		"reverse proxy",
		"staging",
		"mainnet",
	}
	for _, no := range jargonNoNos {
		if strings.Contains(out, no) {
			t.Errorf("error hints leak developer jargon %q\n--- rendered ---\n%s", no, out)
		}
	}
}

// TestRenderErrorSectionAuthFailure exercises a different category to
// make sure the hints really are kind-specific (not a single hardcoded
// list).
func TestRenderErrorSectionAuthFailure(t *testing.T) {
	t.Parallel()

	m := newConnectionModel("", "", nil)
	m.setConnError(
		errors.New(`dial: failed to WebSocket dial: expected handshake response status code 101 but got 401`),
		"wss://www.truestamp.com/console/websocket",
	)

	out := stripANSI(m.renderErrorSection(120))

	wantParts := []string{
		"The server rejected your API key.",
		"truestamp auth login",
	}
	for _, want := range wantParts {
		if !strings.Contains(out, want) {
			t.Errorf("auth error section missing %q\n--- rendered ---\n%s", want, out)
		}
	}
	// Empty configFilePath should not render the Settings line at all.
	if strings.Contains(out, "Settings file:") {
		t.Errorf("Settings file line should be suppressed when configFilePath is empty: %s", out)
	}
}

// TestViewSwitchesToErrorOnlyWhenSet confirms the View shows the
// normal Scope / Push / Reconnect sections during a healthy session
// and replaces them with the error section after setConnError fires.
func TestViewSwitchesToErrorOnlyWhenSet(t *testing.T) {
	t.Parallel()

	m := newConnectionModel("", "", nil)
	healthy := stripANSI(m.View(120, 30))
	if !strings.Contains(healthy, "Scope") {
		t.Errorf("healthy pane missing Scope section: %s", healthy)
	}
	if strings.Contains(healthy, "Connection error") {
		t.Errorf("healthy pane should not show Connection error: %s", healthy)
	}

	m.setConnError(errors.New("dial: failed: no such host"), "ws://nope/")
	failed := stripANSI(m.View(120, 30))
	if !strings.Contains(failed, "Connection error") {
		t.Errorf("failed pane missing Connection error section: %s", failed)
	}
	if strings.Contains(failed, "Scope") {
		t.Errorf("failed pane should suppress empty Scope section: %s", failed)
	}
}

// stripANSI removes ANSI CSI escape sequences so test assertions can
// match plain text. Mirrors the standard pattern used by other Charm
// projects' renderer tests.
func stripANSI(s string) string {
	var b strings.Builder
	in := false
	for _, r := range s {
		if !in {
			if r == 0x1b {
				in = true
				continue
			}
			b.WriteRune(r)
			continue
		}
		// We're inside an escape sequence; consume until a final byte
		// (ASCII 0x40–0x7E) or a newline.
		if (r >= 0x40 && r <= 0x7e) || r == '\n' {
			in = false
		}
	}
	return b.String()
}
