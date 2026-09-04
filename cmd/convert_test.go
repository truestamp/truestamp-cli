// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: MIT

package cmd

import (
	"encoding/json"
	"os/exec"
	"strings"
	"testing"
)

// TestCLI_Convert_Time_UnixToRFC3339 pins the unix-seconds → RFC 3339
// conversion against a known point in time.
func TestCLI_Convert_Time_UnixToRFC3339(t *testing.T) {
	out, err := exec.Command(binaryPath, "convert", "time", "1700000000", "--to-zone", "UTC").Output()
	if err != nil {
		t.Fatal(err)
	}
	want := "2023-11-14T22:13:20Z\n"
	if string(out) != want {
		t.Errorf("got %q, want %q", out, want)
	}
}

// TestCLI_Convert_Time_HalfHourZone guards against any timezone arithmetic
// bug that only surfaces in non-hour offsets (Asia/Kolkata is +05:30).
func TestCLI_Convert_Time_HalfHourZone(t *testing.T) {
	out, err := exec.Command(binaryPath, "convert", "time",
		"2024-06-15T12:00:00Z", "--to-zone", "Asia/Kolkata").Output()
	if err != nil {
		t.Fatal(err)
	}
	got := strings.TrimSpace(string(out))
	if !strings.Contains(got, "+05:30") {
		t.Errorf("expected +05:30 offset in output, got %q", got)
	}
	if !strings.HasPrefix(got, "2024-06-15T17:30:00") {
		t.Errorf("expected 17:30 local time, got %q", got)
	}
}

// TestCLI_Convert_Time_UnixMS covers the --format unix-ms output.
func TestCLI_Convert_Time_UnixMS(t *testing.T) {
	out, err := exec.Command(binaryPath, "convert", "time",
		"1700000000", "--format", "unix-ms").Output()
	if err != nil {
		t.Fatal(err)
	}
	if strings.TrimSpace(string(out)) != "1700000000000" {
		t.Errorf("got %q, want 1700000000000", out)
	}
}

// TestCLI_Convert_ID_ULIDTimestamp extracts the ULID millisecond time.
// The fixture ULID 01HJHB01T8FYZ7YTR9P5N62K5B encodes 2023-12-25T20:34:54.408Z.
func TestCLI_Convert_ID_ULIDTimestamp(t *testing.T) {
	out, err := exec.Command(binaryPath, "convert", "id",
		"01HJHB01T8FYZ7YTR9P5N62K5B").Output()
	if err != nil {
		t.Fatalf("%v", err)
	}
	got := strings.TrimSpace(string(out))
	if !strings.HasPrefix(got, "2023-12-25T20:34:54") {
		t.Errorf("unexpected ULID timestamp: %q", got)
	}
}

// TestCLI_Convert_ID_UUIDv7_RejectsV4 ensures a wrong-version UUID is
// flagged rather than silently producing garbage.
func TestCLI_Convert_ID_UUIDv7_RejectsV4(t *testing.T) {
	// v4 UUIDs always have '4' as the first hex digit of the time_hi
	// group; extraction should fail.
	out, err := exec.Command(binaryPath, "convert", "id",
		"f47ac10b-58cc-4372-a567-0e02b2c3d479").CombinedOutput()
	if err == nil {
		t.Errorf("expected error for UUIDv4, got %s", out)
	}
}

// TestCLI_Convert_KeyID_FixedVector pins the kid derivation against
// the public key embedded in the fake proof fixture. The resulting kid
// must match the fixture's `kid` field, cross-validates our domain
// prefix (0x51) and 4-byte truncation.
func TestCLI_Convert_KeyID_FixedVector(t *testing.T) {
	out, err := exec.Command(binaryPath, "convert", "keyid",
		"CTwMqDZnPd/QTLSq8aTeSD3a+j2DQxKcGfhhIYJQ65Y=").Output()
	if err != nil {
		t.Fatal(err)
	}
	got := strings.TrimSpace(string(out))
	// The fixed vector: this public key derives key id 4ceefa4a.
	if got != "4ceefa4a" {
		t.Errorf("kid: got %q, want 4ceefa4a", got)
	}
}

// TestCLI_Convert_KeyID_JSON verifies the envelope keys.
func TestCLI_Convert_KeyID_JSON(t *testing.T) {
	out, err := exec.Command(binaryPath, "convert", "keyid",
		"CTwMqDZnPd/QTLSq8aTeSD3a+j2DQxKcGfhhIYJQ65Y=", "--json").Output()
	if err != nil {
		t.Fatal(err)
	}
	var m map[string]any
	if err := json.Unmarshal(out, &m); err != nil {
		t.Fatal(err)
	}
	if m["kid_hex"] != "4ceefa4a" {
		t.Errorf("kid_hex: %v", m["kid_hex"])
	}
	if _, ok := m["public_key_hex"].(string); !ok {
		t.Errorf("missing public_key_hex")
	}
}
