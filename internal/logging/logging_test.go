// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: MIT

package logging

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestRedactingHandlerScrubsAttrs covers the three places a secret can
// reach the JSON output: a string-valued attribute, an error-valued
// attribute, and the message text itself.
func TestRedactingHandlerScrubsAttrs(t *testing.T) {
	const secret = "truestamp_VERY_SECRET_TOKEN"

	var buf bytes.Buffer
	handler := NewRedactingHandler(slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug}))
	logger := slog.New(handler)

	logger.Info("connect failed: dial wss://x?api_key="+secret,
		slog.String("url", "wss://x?api_key="+secret),
		slog.Any("err", errors.New("upstream: api_key="+secret)),
	)

	out := buf.String()
	if strings.Contains(out, secret) {
		t.Fatalf("secret leaked: %s", out)
	}
	if !strings.Contains(out, "api_key=REDACTED") {
		t.Errorf("expected api_key=REDACTED in output: %s", out)
	}
}

// TestRedactingHandlerWithBaseAttrs verifies that secrets baked into a
// child logger via .With(...) — the path most subcommands will take —
// are redacted at attach time, not just per record.
func TestRedactingHandlerWithBaseAttrs(t *testing.T) {
	const secret = "truestamp_BASE_ATTR_TOKEN"

	var buf bytes.Buffer
	handler := NewRedactingHandler(slog.NewJSONHandler(&buf, nil))
	logger := slog.New(handler).With(
		slog.String("session_url", "wss://x?api_key="+secret),
	)
	logger.Info("hello")

	out := buf.String()
	if strings.Contains(out, secret) {
		t.Fatalf("secret leaked from base attrs: %s", out)
	}
	if !strings.Contains(out, "api_key=REDACTED") {
		t.Errorf("expected api_key=REDACTED: %s", out)
	}
}

// TestRedactingHandlerGroups confirms group nesting doesn't bypass
// redaction — a secret inside slog.Group(...) must still be scrubbed.
func TestRedactingHandlerGroups(t *testing.T) {
	const secret = "truestamp_GROUP_TOKEN"

	var buf bytes.Buffer
	handler := NewRedactingHandler(slog.NewJSONHandler(&buf, nil))
	logger := slog.New(handler)

	logger.Info("nested",
		slog.Group("transport",
			slog.String("url", "wss://x?api_key="+secret),
		),
	)
	if strings.Contains(buf.String(), secret) {
		t.Fatalf("secret leaked through group: %s", buf.String())
	}
}

// TestRedactingHandlerTransparent confirms no-secret records pass
// through byte-identical to the unwrapped JSON handler. We compare
// only the parsed attr maps because slog encodes timestamps differently
// per record; the assertion is on payload, not the time field.
func TestRedactingHandlerTransparent(t *testing.T) {
	var clean bytes.Buffer
	clean1 := slog.New(slog.NewJSONHandler(&clean, nil))
	clean1.Info("hello", slog.String("k", "value"), slog.Int("n", 42))

	var redacted bytes.Buffer
	redacted1 := slog.New(NewRedactingHandler(slog.NewJSONHandler(&redacted, nil)))
	redacted1.Info("hello", slog.String("k", "value"), slog.Int("n", 42))

	cleanMap := decodeJSONLine(t, clean.Bytes())
	redactedMap := decodeJSONLine(t, redacted.Bytes())
	delete(cleanMap, "time")
	delete(redactedMap, "time")

	if cleanMap["msg"] != redactedMap["msg"] || cleanMap["k"] != redactedMap["k"] {
		t.Fatalf("redacting handler altered clean record:\nclean:    %v\nredacted: %v", cleanMap, redactedMap)
	}
}

// TestRedactingHandlerEnabled checks the level filter is delegated, not
// shadowed — debug records are dropped when the inner handler's level
// is info, regardless of redaction.
func TestRedactingHandlerEnabled(t *testing.T) {
	var buf bytes.Buffer
	handler := NewRedactingHandler(slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: slog.LevelInfo}))
	logger := slog.New(handler)

	logger.Debug("should not appear")
	if buf.Len() != 0 {
		t.Fatalf("debug record leaked through level filter: %s", buf.String())
	}
	logger.Info("should appear")
	if buf.Len() == 0 {
		t.Fatalf("info record dropped — Enabled() not delegating")
	}

	// Direct Enabled() call should also delegate.
	if handler.Enabled(context.Background(), slog.LevelDebug) {
		t.Fatalf("Enabled(Debug) returned true for an info-only handler")
	}
}

// TestNewWritesRedactedJSON exercises the integrated pipeline (lazy
// writer + RedactingHandler + lumberjack) end-to-end against a tmpdir
// log path so we know production wiring redacts before disk.
func TestNewWritesRedactedJSON(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "truestamp.log")
	logger, gotPath, err := New(Options{Path: path, Level: "debug"})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	if gotPath != path {
		t.Fatalf("path: got %q want %q", gotPath, path)
	}

	const secret = "truestamp_END_TO_END_TOKEN"
	logger.Info("dial failed",
		slog.String("url", "wss://x?api_key="+secret),
	)

	// Lumberjack writes synchronously on each call; no flush needed.
	body, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read log: %v", err)
	}
	if strings.Contains(string(body), secret) {
		t.Fatalf("secret leaked to disk: %s", body)
	}
	if !strings.Contains(string(body), "api_key=REDACTED") {
		t.Fatalf("expected redacted form: %s", body)
	}
}

// TestNewLazyOpen confirms that constructing a logger and never
// emitting a record leaves no file behind. Important because every CLI
// invocation (including hash-loop scripts) builds a logger up front.
func TestNewLazyOpen(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "truestamp.log")
	if _, _, err := New(Options{Path: path}); err != nil {
		t.Fatalf("New: %v", err)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("file should not exist before first write; stat err=%v", err)
	}
}

// TestNewComponentTag verifies the component option attaches to every
// record via With() without bypassing redaction.
func TestNewComponentTag(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "truestamp.log")
	logger, _, err := New(Options{Path: path, Component: "verify"})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	logger.Info("hello")
	body, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	rec := decodeJSONLine(t, body)
	if rec["component"] != "verify" {
		t.Fatalf("expected component=verify, got %v", rec["component"])
	}
}

// FuzzRedactingHandlerNoLeak asserts that the api_key value inside an
// `api_key=…` URL is scrubbed no matter which attribute channel it
// arrives through — string attribute, error attribute, nested group,
// base attribute attached via .With(), or the record's message text.
//
// This complements FuzzRedact (which fuzzes the pure regex) by
// exercising the slog.Handler walk: the redactAttr branches for
// KindString / KindAny / KindGroup, the message-text path through
// slog.NewRecord, and the WithAttrs base-attr path that .With() takes.
//
// Scope deliberately excludes "bare token detection" — the redactor
// only knows the api_key= and Bearer patterns by design (high-entropy
// token heuristics would false-positive on hashes / IDs / merkle
// roots), so fuzz inputs that contain the secret bytes outside of
// those wrappers are skipped: they test something the redactor never
// claimed to do.
//
// Run:
//
//	go test -run=^$ -fuzz=FuzzRedactingHandlerNoLeak -fuzztime=30s ./internal/logging/...
func FuzzRedactingHandlerNoLeak(f *testing.F) {
	const secret = "TRUESTAMP_FUZZ_SECRET_TOKEN_xyz"
	seeds := []struct {
		msg, key, val string
	}{
		{"hello", "k", "v"},
		{"", "", ""},
		{"dial failed", "url", "extra"},
		{"multi\nline\nmessage", "err", "boom"},
		{"unicode: 你好 🔑", "ключ", "значение"},
		{"long " + strings.Repeat("a", 4096), "k", strings.Repeat("b", 4096)},
		{"key with quotes", `"k"`, "v"},
		{"key with newline", "k\nl", "v"},
	}
	for _, s := range seeds {
		f.Add(s.msg, s.key, s.val)
	}

	f.Fuzz(func(t *testing.T, msg, key, val string) {
		// Skip inputs that already contain the secret bytes outside our
		// known wrapping patterns — the redactor only targets api_key=
		// and Bearer by design, not bare tokens.
		if strings.Contains(msg, secret) || strings.Contains(key, secret) || strings.Contains(val, secret) {
			return
		}

		// Inject the secret via every documented leak channel:
		//   1. a string attribute value (key is fuzz-controlled)
		//   2. an error attribute value
		//   3. a nested group attribute
		//   4. a base attribute attached via .With()
		//   5. the message text (appended after the fuzz-supplied msg)
		secretURL := "wss://x?api_key=" + secret + "&vsn=2.0.0"
		bearerHeader := "Authorization: Bearer " + secret
		secretErr := errors.New("upstream: " + secretURL + " (extra: " + val + ")")

		var buf bytes.Buffer
		base := slog.New(NewRedactingHandler(slog.NewJSONHandler(&buf, &slog.HandlerOptions{
			Level: slog.LevelDebug,
		}))).With(
			slog.String("session_url", secretURL),
			slog.String("auth_hdr", bearerHeader),
		)

		// Even malformed keys (empty, with quotes/newlines, etc.) must
		// not crash slog or the handler — the fuzz catches both panics
		// and post-redaction leaks.
		base.Info(msg+" "+secretURL+" "+bearerHeader,
			slog.String(key, secretURL),
			slog.String("body", val),
			slog.Any("err", secretErr),
			slog.Group("transport",
				slog.String("dial_url", secretURL),
				slog.String("note", val),
			),
		)

		out := buf.String()
		if strings.Contains(out, secret) {
			t.Fatalf("secret leaked through handler. msg=%q key=%q val=%q\noutput: %s",
				msg, key, val, out)
		}
	})
}

func decodeJSONLine(t *testing.T, b []byte) map[string]any {
	t.Helper()
	// Take just the first line in case multiple records were emitted.
	if i := bytes.IndexByte(b, '\n'); i >= 0 {
		b = b[:i]
	}
	var m map[string]any
	if err := json.Unmarshal(b, &m); err != nil {
		t.Fatalf("decode JSON line %q: %v", b, err)
	}
	return m
}
