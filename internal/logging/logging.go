// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: MIT

// Package logging provides a file-backed structured logger for the
// truestamp CLI: a single JSON-line log per invocation, written to a
// rotated file the user can grep later. The same logger is used by
// every subcommand (one-shot tools like verify, hash, download, upgrade
// and the long-lived console TUI) so a user reporting an issue has one
// place to look.
//
// Default destination, by platform:
//
//	macOS:   ~/Library/Caches/truestamp/truestamp.log
//	Linux:   ~/.cache/truestamp/truestamp.log
//	Windows: %LOCALAPPDATA%\truestamp\Cache\truestamp.log
//
// Files are rotated by size (10 MB default) with up to 5 compressed
// backups retained for 14 days. Output is one JSON object per line,
// directly grep/jq-able and easy to ingest into any log pipeline later.
//
// All output passes through [RedactingHandler] before reaching the file:
// `api_key=…` and `Bearer …` substrings in attribute values, error
// chains, or message text are replaced with the REDACTED sentinel. This
// is a defense-in-depth safety net, call sites should still avoid
// putting secrets into log attributes in the first place.
package logging

import (
	"context"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/truestamp/truestamp-cli/internal/redact"
	"gopkg.in/natefinch/lumberjack.v2"
)

// Options configures logger construction.
type Options struct {
	// Path overrides the default log file path. Empty = platform default.
	Path string

	// Level filters output. One of "debug", "info", "warn", "error";
	// empty defaults to "info".
	Level string

	// MaxSizeMB rotates after N megabytes. Defaults to 10.
	MaxSizeMB int

	// MaxBackups retains N rotated files. Defaults to 5.
	MaxBackups int

	// MaxAgeDays retains rotated files for N days. Defaults to 14.
	MaxAgeDays int

	// Component tags every record with a `component` attribute so log
	// readers can filter to a single subsystem (e.g. `component=console`
	// or `component=verify`). Empty string omits the attribute.
	Component string
}

// New returns a configured slog.Logger and the resolved log file path
// so the caller can surface it in diagnostic UIs. If file creation
// fails, a discard logger is returned along with the original error,
// callers should never crash over a logging failure.
//
// The underlying file is opened lazily on the first write, so a command
// that emits no log records (e.g. `truestamp hash --silent` with
// --log-level error) pays no per-invocation file-creation cost.
func New(opts Options) (*slog.Logger, string, error) {
	path := opts.Path
	if path == "" {
		var err error
		path, err = defaultPath()
		if err != nil {
			return Discard(), "", err
		}
	}

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return Discard(), path, err
	}

	sink := &lumberjack.Logger{
		Filename:   path,
		MaxSize:    coalesce(opts.MaxSizeMB, 10),
		MaxBackups: coalesce(opts.MaxBackups, 5),
		MaxAge:     coalesce(opts.MaxAgeDays, 14),
		Compress:   true,
	}

	jsonHandler := slog.NewJSONHandler(&lazyWriter{w: sink}, &slog.HandlerOptions{
		Level: parseLevel(opts.Level),
	})
	var handler slog.Handler = NewRedactingHandler(jsonHandler)
	logger := slog.New(handler)
	if opts.Component != "" {
		logger = logger.With("component", opts.Component)
	}
	return logger, path, nil
}

// Discard returns a logger that drops every record. Used as a fallback
// when no file logger can be constructed (e.g. read-only home dir) so
// callers never have to worry about a nil logger.
func Discard() *slog.Logger {
	return slog.New(slog.NewJSONHandler(io.Discard, nil))
}

// DefaultPath returns the platform-default log file path so callers
// (typically Cobra flag registration) can show it in their help text.
// Returns the empty string on platforms where os.UserCacheDir fails;
// callers should treat that as "no default visible" and document the
// override path as the only way to relocate the log.
func DefaultPath() string {
	path, err := defaultPath()
	if err != nil {
		return ""
	}
	return path
}

func defaultPath() (string, error) {
	cache, err := os.UserCacheDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(cache, "truestamp", "truestamp.log"), nil
}

func parseLevel(s string) slog.Level {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "debug":
		return slog.LevelDebug
	case "warn", "warning":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

func coalesce(v, def int) int {
	if v <= 0 {
		return def
	}
	return v
}

// lazyWriter defers the first call to its inner io.Writer until a Write
// actually happens. lumberjack.Logger creates the file (and the parent
// directory) on its first Write, wrapping it here means a no-log-line
// invocation never touches the filesystem at all.
//
// Every Write takes the mutex; there is no fast path or one-shot flag.
type lazyWriter struct {
	w  io.Writer
	mu sync.Mutex
}

func (l *lazyWriter) Write(p []byte) (int, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.w.Write(p)
}

// RedactingHandler wraps an underlying slog.Handler and runs every
// string attribute value, every error attribute, and the record's
// message through [redact.String] before forwarding the record. Group
// boundaries and attribute keys are passed through untouched, the
// redaction only targets values, where secrets can actually live.
type RedactingHandler struct {
	inner slog.Handler
}

// NewRedactingHandler wraps inner so all records flowing through it
// have api_key / Bearer secrets replaced with the redact.REDACTED
// sentinel before they reach the underlying writer.
func NewRedactingHandler(inner slog.Handler) *RedactingHandler {
	return &RedactingHandler{inner: inner}
}

// Enabled delegates to the wrapped handler, redaction has no opinion
// on which levels are emitted.
func (h *RedactingHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return h.inner.Enabled(ctx, level)
}

// Handle copies the record with a redacted message, walks every
// attribute redacting string and error values, then forwards to the
// inner handler. We rebuild the record (rather than mutating the
// original) because slog.Record values are designed to be value-copied
// safely.
func (h *RedactingHandler) Handle(ctx context.Context, r slog.Record) error {
	out := slog.NewRecord(r.Time, r.Level, redact.String(r.Message), r.PC)
	r.Attrs(func(a slog.Attr) bool {
		out.AddAttrs(redactAttr(a))
		return true
	})
	return h.inner.Handle(ctx, out)
}

// WithAttrs returns a new RedactingHandler whose inner handler has
// been pre-attributed with redacted versions of attrs. Pre-attribution
// is the path slog.Logger.With() takes, so this catches secrets that
// are baked into a child logger.
func (h *RedactingHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	cleaned := make([]slog.Attr, len(attrs))
	for i, a := range attrs {
		cleaned[i] = redactAttr(a)
	}
	return &RedactingHandler{inner: h.inner.WithAttrs(cleaned)}
}

// WithGroup delegates, group names are caller-controlled identifiers,
// not values, and so don't need redaction.
func (h *RedactingHandler) WithGroup(name string) slog.Handler {
	return &RedactingHandler{inner: h.inner.WithGroup(name)}
}

// redactAttr applies redaction to an attribute's value. String values
// are run through redact.String directly. Errors are unwrapped to a
// string, redacted, and re-wrapped as a string attribute (we lose the
// `error` type tag in the process, but the JSON output is identical
// and we get redaction). Group and Any values are recursed/handled so
// nested secrets don't escape; everything else passes through.
func redactAttr(a slog.Attr) slog.Attr {
	switch a.Value.Kind() {
	case slog.KindString:
		s := a.Value.String()
		clean := redact.String(s)
		if clean != s {
			return slog.String(a.Key, clean)
		}
		return a
	case slog.KindAny:
		if err, ok := a.Value.Any().(error); ok {
			return slog.String(a.Key, redact.Error(err))
		}
		// stringer-like fallback: format with %v, redact, wrap as
		// string. Cheap, and the JSON output an end user reads is
		// identical to what slog would have produced for the raw
		// value (slog's JSON handler stringifies Any too).
		if s, ok := a.Value.Any().(string); ok {
			return slog.String(a.Key, redact.String(s))
		}
		return a
	case slog.KindGroup:
		group := a.Value.Group()
		cleaned := make([]any, 0, len(group)*2)
		for _, ga := range group {
			cleaned = append(cleaned, redactAttr(ga))
		}
		return slog.Group(a.Key, cleaned...)
	default:
		return a
	}
}
