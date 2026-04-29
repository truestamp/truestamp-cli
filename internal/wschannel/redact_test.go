// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: MIT

package wschannel

import (
	"bytes"
	"context"
	"log/slog"
	"strings"
	"testing"
	"time"

	"github.com/truestamp/truestamp-cli/internal/redact"
)

// TestRedactAPIKey confirms the api_key never appears verbatim in any
// log line, even when the underlying error chains through the
// websocket library (which echoes the URL).
func TestRedactAPIKey(t *testing.T) {
	const secret = "truestamp_VERY_SECRET_TOKEN_SHOULD_NOT_LEAK"

	var buf bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&buf, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}))

	c, err := New(Options{
		URL:               "ws://127.0.0.1:1/console/websocket", // port 1 → connection refused
		APIKey:            secret,
		Logger:            logger,
		HeartbeatInterval: time.Second,
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	t.Cleanup(func() { _ = c.Close() })

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	_, _ = c.Connect(ctx) // expected to fail, that's the point

	// Give any subsequent reconnect attempt a moment to also log.
	time.Sleep(500 * time.Millisecond)

	if strings.Contains(buf.String(), secret) {
		t.Fatalf("api_key leaked verbatim in log output: %s", buf.String())
	}
}

// TestRedactConnectError ensures the error returned from Connect (and
// therefore visible in the TUI's "connect failed" header) is also
// redacted, not just the log lines.
func TestRedactConnectError(t *testing.T) {
	const secret = "truestamp_VERY_SECRET_TOKEN_SHOULD_NOT_LEAK"

	c, err := New(Options{
		URL:    "ws://127.0.0.1:1/console/websocket",
		APIKey: secret,
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	t.Cleanup(func() { _ = c.Close() })

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	_, err = c.Connect(ctx)
	if err == nil {
		t.Fatal("expected Connect to fail (port 1 should refuse)")
	}
	if strings.Contains(err.Error(), secret) {
		t.Fatalf("api_key leaked in Connect error: %s", err.Error())
	}
}

// TestRedactSecretsIntegration exercises the shared redact package
// through the wschannel import path so that "the wschannel client uses
// the canonical redactor" is regression-tested at the right boundary.
// The regex's own correctness lives in internal/redact/redact_test.go;
// this file's role is to confirm wschannel hasn't quietly forked off a
// local redactor again.
func TestRedactSecretsIntegration(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want string
	}{
		{
			name: "simple query string",
			in:   `dial ws://x/y?api_key=truestamp_abc&vsn=2.0.0`,
			want: `dial ws://x/y?api_key=REDACTED&vsn=2.0.0`,
		},
		{
			name: "quoted url in error",
			in:   `Get "http://x/y?api_key=truestamp_secret123&vsn=2.0.0"`,
			want: `Get "http://x/y?api_key=REDACTED&vsn=2.0.0"`,
		},
		{
			name: "no api_key",
			in:   "connection refused",
			want: "connection refused",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := redact.String(tc.in); got != tc.want {
				t.Errorf("got %q, want %q", got, tc.want)
			}
		})
	}
}
