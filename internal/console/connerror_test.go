// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: MIT

package console

import (
	"errors"
	"testing"
)

// TestClassifyConnError covers the substring patterns we lift from
// the public contracts of net/http and coder/websocket. Each case
// pairs a representative error string with the kind we expect, so a
// silent change in either upstream surface is caught here rather than
// degrading the user-facing diagnostics.
func TestClassifyConnError(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		msg  string
		want connErrorKind
	}{
		{
			name: "DNS failure",
			msg:  `dial: failed to WebSocket dial: failed to send handshake request: Get "wss://nope.example.com/console/websocket": dial tcp: lookup nope.example.com: no such host`,
			want: connErrorDNS,
		},
		{
			name: "connection refused (server not running)",
			msg:  `dial: failed to WebSocket dial: failed to send handshake request: Get "ws://localhost:4010/console/websocket": dial tcp [::1]:4010: connect: connection refused`,
			want: connErrorRefused,
		},
		{
			name: "i/o timeout",
			msg:  `dial: failed to WebSocket dial: failed to send handshake request: Get "wss://slow.example.com/": dial tcp 1.2.3.4:443: i/o timeout`,
			want: connErrorTimeout,
		},
		{
			name: "context deadline exceeded",
			msg:  `dial: failed to WebSocket dial: context deadline exceeded`,
			want: connErrorTimeout,
		},
		{
			name: "TLS verify",
			msg:  `dial: failed to WebSocket dial: tls: failed to verify certificate: x509: certificate signed by unknown authority`,
			want: connErrorTLS,
		},
		{
			name: "auth 401",
			msg:  `dial: failed to WebSocket dial: expected handshake response status code 101 but got 401`,
			want: connErrorAuth,
		},
		{
			name: "auth 403",
			msg:  `dial: failed to WebSocket dial: expected handshake response status code 101 but got 403`,
			want: connErrorAuth,
		},
		{
			name: "not found 404",
			msg:  `dial: failed to WebSocket dial: expected handshake response status code 101 but got 404`,
			want: connErrorNotFound,
		},
		{
			name: "server 502",
			msg:  `dial: failed to WebSocket dial: expected handshake response status code 101 but got 502`,
			want: connErrorServerError,
		},
		{
			name: "other non-101",
			msg:  `dial: failed to WebSocket dial: expected handshake response status code 101 but got 418`,
			want: connErrorProtocol,
		},
		{
			name: "bad URL",
			msg:  `parse url: parse "://broken": missing protocol scheme`,
			want: connErrorBadURL,
		},
		{
			name: "decode welcome",
			msg:  `decode welcome: invalid character '<' looking for beginning of value`,
			want: connErrorDecodeWelcome,
		},
		{
			name: "unknown shape falls through",
			msg:  `something completely unexpected`,
			want: connErrorUnknown,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := classifyConnError(errors.New(tc.msg))
			if got != tc.want {
				t.Errorf("classifyConnError(%q) = %d, want %d", tc.msg, got, tc.want)
			}
		})
	}
}

// TestConnErrorKindHasUserFacingText guards the contract that every
// kind has non-empty short status / title / hints, empty strings
// would leave a blank header pill or an empty pane section in
// production.
func TestConnErrorKindHasUserFacingText(t *testing.T) {
	t.Parallel()

	for k := connErrorUnknown; k <= connErrorDecodeWelcome; k++ {
		if got := k.shortStatus(); got == "" {
			t.Errorf("kind %d: shortStatus() is empty", k)
		}
		if got := k.title(); got == "" {
			t.Errorf("kind %d: title() is empty", k)
		}
		if hints := k.hints(); len(hints) == 0 {
			t.Errorf("kind %d: hints() is empty", k)
		}
	}
}

// TestClassifyConnErrorNil ensures the classifier doesn't panic on
// a nil error, it shouldn't be called that way but defensive cheap.
func TestClassifyConnErrorNil(t *testing.T) {
	t.Parallel()
	if got := classifyConnError(nil); got != connErrorUnknown {
		t.Errorf("classifyConnError(nil) = %d, want connErrorUnknown", got)
	}
}
