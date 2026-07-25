// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: MIT

package cmd

import (
	"bytes"
	"encoding/json"
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

// FuzzPrettyJSON: `convert proof --to json` re-indents the marshaled
// bundle without re-encoding it, so json.Indent is handed bytes derived
// from an attacker-supplied proof. It must never panic, and — the whole
// reason it replaced a round-trip through `any` — must never alter a
// number literal, which is what a claims_hash is computed over. Seeds
// cover the boundary Appendix C.2a pins (2^53 and 2^53 + 1), a uint64
// that overflows float64, floats, and the shapes json.Indent rejects.
func FuzzPrettyJSON(f *testing.F) {
	for _, s := range []string{
		`{}`,
		`{"n":9007199254740992}`,
		`{"n":9007199254740993}`,
		`{"n":18446744073709551615}`,
		`{"n":-9007199254740993}`,
		`{"n":1.5e300,"m":-0.0}`,
		`{"a":[1,2,{"b":"c"}]}`,
		`[1,2,3]`,
		`  {"a" : 1}  `,
		`{`,
		``,
		"\x00",
	} {
		f.Add(s)
	}
	f.Fuzz(func(t *testing.T, s string) {
		out, err := prettyJSON([]byte(s))
		if err != nil {
			return
		}
		// Indentation is whitespace-only, so stripping whitespace outside
		// strings must return the input unchanged. Comparing the compacted
		// forms is the cheap way to say that without a JSON walker.
		var gotCompact, wantCompact bytes.Buffer
		if err := json.Compact(&gotCompact, out); err != nil {
			t.Fatalf("prettyJSON emitted bytes json.Compact rejects: %v", err)
		}
		if err := json.Compact(&wantCompact, []byte(s)); err != nil {
			// json.Indent accepted it, so Compact must too.
			t.Fatalf("json.Compact rejected input json.Indent accepted: %v", err)
		}
		if !bytes.Equal(gotCompact.Bytes(), wantCompact.Bytes()) {
			t.Fatalf("prettyJSON altered the document: got %q, want %q",
				gotCompact.String(), wantCompact.String())
		}
	})
}
