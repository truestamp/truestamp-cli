// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: MIT

// Package logging provides a file-backed structured logger for the
// truestamp CLI, designed for the TUI's diagnostic-and-postmortem use
// case: never write to stdout/stderr (the TUI owns those), always
// write to a rotated file the user can grep later.
//
// Default destination, by platform:
//
//	macOS:   ~/Library/Caches/truestamp/console.log
//	Linux:   ~/.cache/truestamp/console.log
//	Windows: %LOCALAPPDATA%\truestamp\Cache\console.log
//
// Files are rotated by size (10 MB default) with up to 5 compressed
// backups retained for 14 days. Output is one JSON object per line —
// directly grep/jq-able and easy to ingest into any log pipeline later.
package logging

import (
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

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
}

// New returns a configured slog.Logger and the resolved log file path
// so the caller can surface it in diagnostic UIs. If file creation
// fails, a discard logger is returned along with the original error —
// callers should never crash the TUI over a logging failure.
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

	handler := slog.NewJSONHandler(sink, &slog.HandlerOptions{
		Level: parseLevel(opts.Level),
	})
	return slog.New(handler), path, nil
}

// Discard returns a logger that drops every record. Used as a fallback
// when no file logger can be constructed (e.g. read-only home dir) so
// the TUI never has to worry about a nil logger.
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
	return filepath.Join(cache, "truestamp", "console.log"), nil
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
