// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: MIT

// Package redact is the single source of truth for stripping
// Truestamp-specific secrets out of strings before they reach disk,
// stderr, error chains, or anywhere else an attacker could read them.
//
// The CLI's slog handler ([internal/logging.RedactingHandler]) and
// [internal/wschannel].Client both flow every observable string through
// [String] so a future contributor can't accidentally leak a key by
// adding a `slog.String("url", urlWithKey)` call. Treat that handler as
// a safety net, not as license — call sites should still avoid putting
// secrets in attributes.
package redact

import "regexp"

// REDACTED is the sentinel that replaces matched secret values. Tests in
// this package and downstream packages assert "the secret is gone" by
// asserting "REDACTED is present" — keep the constant exported so those
// assertions don't drift.
const REDACTED = "REDACTED"

var (
	// apiKeyRe matches the `api_key=…` query-string fragment that
	// websocket.Dial errors echo verbatim from the upgrade URL, and
	// that any future logger of HTTP request URLs would also surface.
	// Stops at the next URL delimiter, quote, or whitespace so adjacent
	// query params (`&vsn=2.0.0`) survive.
	apiKeyRe = regexp.MustCompile(`api_key=[^&"\s]*`)

	// bearerRe matches `Bearer <token>` in error chains and any
	// hypothetical request-header dump. Case-insensitive (`(?i)`)
	// because some HTTP clients normalize header values to title-case
	// and others don't. Stops at the next whitespace or quote so a
	// surrounding error message stays readable.
	bearerRe = regexp.MustCompile(`(?i)Bearer\s+[^"\s]+`)
)

// String applies every redaction pattern to s and returns the cleaned
// copy. Safe for arbitrary input — no panics, no allocations beyond
// what regexp.ReplaceAllString needs. Returns s unchanged when no
// pattern matches, so the common "clean string" path is cheap.
func String(s string) string {
	if s == "" {
		return s
	}
	s = apiKeyRe.ReplaceAllString(s, "api_key="+REDACTED)
	s = bearerRe.ReplaceAllString(s, "Bearer "+REDACTED)
	return s
}

// Error returns err.Error() with redaction applied. Convenience wrapper
// for the common `slog.String("err", redact.Error(err))` pattern. nil
// returns the empty string, matching err.Error()'s panic-on-nil
// behavior in spirit but without the panic.
func Error(err error) string {
	if err == nil {
		return ""
	}
	return String(err.Error())
}
