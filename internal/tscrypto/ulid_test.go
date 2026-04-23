// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: MIT

package tscrypto

import (
	"testing"
	"time"
)

func TestExtractULIDTimestamp(t *testing.T) {
	t.Parallel()
	// Known ULID from proof: 01KKW164FPQ6V353WM3NXFRTCE
	ulid := "01KKW164FPQ6V353WM3NXFRTCE"
	ts, err := ExtractULIDTimestamp(ulid)
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	if ts.Year() != 2026 {
		t.Errorf("expected year 2026, got %d", ts.Year())
	}
	if ts.Location() != time.UTC {
		t.Error("timestamp should be UTC")
	}
}

func TestExtractULIDTimestamp_TooShort(t *testing.T) {
	t.Parallel()
	_, err := ExtractULIDTimestamp("abc")
	if err == nil {
		t.Error("expected error for short ULID")
	}
}

func TestExtractULIDTimestamp_InvalidChar(t *testing.T) {
	t.Parallel()
	// 'U' is not in Crockford Base32 — 26-char string with invalid chars
	_, err := ExtractULIDTimestamp("0UUUUUUUUUUUUUUUUUUUUUUUUU")
	if err == nil {
		t.Error("expected error for invalid Crockford character")
	}
}

func TestFormatItemTime(t *testing.T) {
	t.Parallel()
	result := FormatItemTime("01KKW164FPQ6V353WM3NXFRTCE")
	if result == "unknown" {
		t.Error("should not return unknown for valid ULID")
	}
	if len(result) < 20 {
		t.Errorf("expected RFC3339 format, got %q", result)
	}
}

func TestFormatItemTime_Invalid(t *testing.T) {
	t.Parallel()
	result := FormatItemTime("!!!")
	if result != "unknown" {
		t.Errorf("expected 'unknown' for invalid ULID, got %q", result)
	}
}
