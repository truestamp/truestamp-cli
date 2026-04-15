// Copyright (c) 2021-2026 Truestamp, Inc.
// SPDX-License-Identifier: MIT

package tscrypto

import (
	"testing"
	"time"
)

func TestExtractUUIDv7Timestamp(t *testing.T) {
	// Known UUIDv7: 019cf812-af77-7c5b-a89d-9fc8459c7247
	// First 12 hex chars (no hyphens): 019cf812af77
	uuid := "019cf812-af77-7c5b-a89d-9fc8459c7247"
	ts, err := ExtractUUIDv7Timestamp(uuid)
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}

	// Should be in 2026
	if ts.Year() != 2026 {
		t.Errorf("expected year 2026, got %d", ts.Year())
	}
	if ts.Location() != time.UTC {
		t.Error("timestamp should be UTC")
	}
}

func TestExtractUUIDv7Timestamp_TooShort(t *testing.T) {
	_, err := ExtractUUIDv7Timestamp("abc")
	if err == nil {
		t.Error("expected error for short UUID")
	}
}

func TestFormatBlockTime(t *testing.T) {
	result := FormatBlockTime("019cf812-af77-7c5b-a89d-9fc8459c7247")
	if result == "unknown" {
		t.Error("should not return unknown for valid UUID")
	}
	// Should be RFC3339 format
	if len(result) < 20 {
		t.Errorf("expected RFC3339 format, got %q", result)
	}
}

func TestFormatBlockTime_Invalid(t *testing.T) {
	result := FormatBlockTime("not-a-uuid")
	if result != "unknown" {
		t.Errorf("expected 'unknown' for invalid UUID, got %q", result)
	}
}