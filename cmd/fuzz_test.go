// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: MIT

package cmd

import (
	"testing"
	"time"
)

// referenceTime is a fixed instant used by the FuzzFormatTime seeds so
// the formatter output stays deterministic regardless of wall clock.
func referenceTime() time.Time {
	return time.Date(2026, 4, 21, 12, 0, 0, 0, time.UTC)
}

// These fuzz the CLI's small in-package parsers that consume
// user-controlled strings from argv or stdin. None is exported, so
// the fuzz targets live alongside the other cmd tests.

// FuzzParsePrefixByte: `--prefix` hex byte parser.
func FuzzParsePrefixByte(f *testing.F) {
	for _, s := range []string{"", "0x11", "0X1A", "ff", "11", "zz", "0xZZ", "abc"} {
		f.Add(s)
	}
	f.Fuzz(func(t *testing.T, s string) {
		_, _ = parsePrefixByte(s)
	})
}

// FuzzParseTime: `convert time` parser — RFC 3339 or Unix-{s,ms,us,ns}.
func FuzzParseTime(f *testing.F) {
	for _, raw := range []string{"", "now", "1700000000", "2026-04-21T12:00:00Z", "garbage"} {
		for _, from := range []string{"", "auto", "rfc3339", "unix-s", "unix-ms", "unix-us", "unix-ns"} {
			f.Add(raw, from)
		}
	}
	f.Fuzz(func(t *testing.T, raw, from string) {
		_, _ = parseTime(raw, from)
	})
}

// FuzzNormalizeTimestamp: `create --timestamp` normalizer.
func FuzzNormalizeTimestamp(f *testing.F) {
	for _, s := range []string{"", "2026-04-21", "2026-04-21T12:00:00Z", "bad"} {
		f.Add(s)
	}
	f.Fuzz(func(t *testing.T, s string) {
		_, _ = normalizeTimestamp(s)
	})
}

// FuzzDecodePublicKey: `convert keyid` auto-detecting decoder.
func FuzzDecodePublicKey(f *testing.F) {
	f.Add("CTwMqDZnPd/QTLSq8aTeSD3a+j2DQxKcGfhhIYJQ65Y=", "auto")
	f.Add("", "auto")
	f.Add("short", "auto")
	f.Add("notabase64", "base64")
	f.Fuzz(func(t *testing.T, raw, from string) {
		_, _ = decodePublicKey(raw, from)
	})
}

// FuzzDetectIDKind: `convert id` auto-detection.
func FuzzDetectIDKind(f *testing.F) {
	for _, s := range []string{
		"", "01HJHB01T8FYZ7YTR9P5N62K5B",
		"019cf813-99b8-730a-84f1-5a711a9c355e",
		"not-an-id",
	} {
		f.Add(s)
	}
	f.Fuzz(func(t *testing.T, s string) {
		_ = detectIDKind(s)
	})
}

// FuzzFormatTime: `convert time --format` dispatcher, which accepts
// an arbitrary Go time layout as a catch-all.
func FuzzFormatTime(f *testing.F) {
	f.Add("rfc3339")
	f.Add("unix-s")
	f.Add("")
	f.Add("2006-01-02")

	f.Fuzz(func(t *testing.T, layout string) {
		// Use a fixed time so format output is deterministic.
		_, _ = formatTime(referenceTime(), layout)
	})
}

// FuzzResolveZone: `--to-zone` resolver (IANA + "local"/"utc").
func FuzzResolveZone(f *testing.F) {
	for _, s := range []string{"", "UTC", "utc", "Local", "America/New_York", "Not/A/Zone"} {
		f.Add(s)
	}
	f.Fuzz(func(t *testing.T, s string) {
		_, _ = resolveZone(s)
	})
}

// FuzzInferTypeFromFilename: the verify command now consults arbitrary
// user-controlled strings (file paths, URL basenames, the "(stdin)"
// sentinel) to derive a default --type. Must never panic, must return a
// valid wire-type value or the empty string. Seeds cover every known
// good stem, malformed prefixes, path separators both flavours, URL
// shapes, and the empty / sentinel cases.
func FuzzInferTypeFromFilename(f *testing.F) {
	for _, s := range []string{
		"",
		"(stdin)",
		"truestamp-item-01HJHB01T8FYZ7YTR9P5N62K5B.json",
		"truestamp-entropy-nist-019db702-b08c-73dc-a7cd-2c5e011f1dad.cbor",
		"truestamp-entropy-stellar-019db702.json",
		"truestamp-entropy-bitcoin-019db702.json",
		"truestamp-block-019db702.json",
		"truestamp-beacon-019db702.json",
		"/tmp/truestamp-beacon-019db702.json",
		"C:\\downloads\\truestamp-block-019db702.json",
		"https://example.com/proofs/truestamp-block-019db702.json",
		"truestamp-unknown-stem-abc.json",
		"not-a-truestamp-file.json",
		"truestamp-",
		"truestamp-.json",
		"truestamp-beacon",       // no extension
		"truestamp-beacon-",      // trailing dash, no id
		"truestamp-entropy--xyz", // double dash
		".hidden",
		"...",
	} {
		f.Add(s)
	}
	f.Fuzz(func(t *testing.T, s string) {
		got := inferTypeFromFilename(s)
		// Post-condition: either "" or one of the six canonical values.
		switch got {
		case "", "item", "entropy_nist", "entropy_stellar", "entropy_bitcoin", "block", "beacon":
			// ok
		default:
			t.Fatalf("inferTypeFromFilename(%q) returned non-canonical value %q", s, got)
		}
	})
}
