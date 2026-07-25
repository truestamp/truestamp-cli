// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: MIT

package cmd

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// TestCLI_Convert_Time_UnixToRFC3339 anchors the unix-seconds → RFC 3339
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

// TestCLI_Convert_KeyID_FixedVector anchors the kid derivation against
// the public key embedded in the fake proof fixture. The resulting kid
// must match the fixture's `kid` field — cross-validates our domain
// prefix (0x51) and 4-byte truncation.
func TestCLI_Convert_KeyID_FixedVector(t *testing.T) {
	out, err := exec.Command(binaryPath, "convert", "keyid",
		"CTwMqDZnPd/QTLSq8aTeSD3a+j2DQxKcGfhhIYJQ65Y=").Output()
	if err != nil {
		t.Fatal(err)
	}
	got := strings.TrimSpace(string(out))
	// The fake proof's kid is 4ceefa4a (see fakeProofJSON in verify_test).
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

// TestCLI_Convert_Proof_RoundTrip confirms that json → cbor → json
// survives verify end-to-end (the ultimate round-trip gate: the cbor
// form must be a valid proof that verify can accept).
func TestCLI_Convert_Proof_RoundTrip(t *testing.T) {
	src := filepath.Join(findTestdata(t), "proof_item.json")
	if _, err := os.Stat(src); err != nil {
		t.Skipf("no fixture: %v", err)
	}

	// Convert JSON → CBOR.
	cborPath := filepath.Join(t.TempDir(), "proof.cbor")
	cbor, err := exec.Command(binaryPath, "convert", "proof", "--to", "cbor", src).Output()
	if err != nil {
		t.Fatalf("json→cbor: %v", err)
	}
	if err := os.WriteFile(cborPath, cbor, 0644); err != nil {
		t.Fatal(err)
	}

	// Convert CBOR → JSON.
	cmd := exec.Command(binaryPath, "convert", "proof", "--to", "json", cborPath)
	if _, err := cmd.Output(); err != nil {
		t.Fatalf("cbor→json: %v", err)
	}

	// The ultimate test: verify accepts the CBOR form end-to-end (skip
	// external for speed and offline friendliness).
	vrf := exec.Command(binaryPath, "verify", cborPath, "--skip-external")
	if err := vrf.Run(); err != nil {
		t.Errorf("verify on round-tripped CBOR failed: %v", err)
	}
}

// TestCLI_Convert_Proof_PrettyPreservesNumbers pins the default --to json
// path against silent rounding. Pretty output is the default, and it used to
// round-trip through `any`, which decodes every JSON number into a float64:
// an integer above 2^53 came back changed, so `convert proof` could quietly
// alter the very bytes a claims_hash is computed over.
func TestCLI_Convert_Proof_PrettyPreservesNumbers(t *testing.T) {
	src := filepath.Join(findTestdata(t), "fixtures", "item-bigint.json")
	if _, err := os.Stat(src); err != nil {
		t.Skipf("no fixture: %v", err)
	}
	out, err := exec.Command(binaryPath, "convert", "proof", "--to", "json", src).Output()
	if err != nil {
		t.Fatalf("convert proof --to json: %v", err)
	}
	for _, literal := range []string{"9007199254740993", "18446744073709551615"} {
		if !strings.Contains(string(out), literal) {
			t.Errorf("pretty output lost the literal %s:\n%s", literal, out)
		}
	}
	// The rounded forms the float64 round-trip produced.
	for _, rounded := range []string{"9007199254740992", "18446744073709552000"} {
		if strings.Contains(string(out), rounded) {
			t.Errorf("pretty output rounded a number to %s:\n%s", rounded, out)
		}
	}
	// --compact never re-encoded, so the two forms must agree once
	// whitespace is removed.
	compact, err := exec.Command(binaryPath, "convert", "proof", "--to", "json", "--compact", src).Output()
	if err != nil {
		t.Fatalf("convert proof --compact: %v", err)
	}
	var pretty, flat bytes.Buffer
	if err := json.Compact(&pretty, out); err != nil {
		t.Fatalf("compacting pretty output: %v", err)
	}
	if err := json.Compact(&flat, compact); err != nil {
		t.Fatalf("compacting compact output: %v", err)
	}
	if pretty.String() != flat.String() {
		t.Errorf("pretty and --compact disagree:\n  pretty:  %s\n  compact: %s", pretty.String(), flat.String())
	}
}

// TestCLI_Convert_Proof_CBORRoundTripPreservesNumbers pins the cross-format
// invariant on the value space that breaks it. The CBOR marshaller decodes
// `s.d` before re-encoding it, so without UseNumber every integer above 2^53
// came back as a float64 — and because both wire formats rounded identically,
// a naive round-trip comparison could not see it. Asserting the literals
// survive is what catches a regression, and verifying the CBOR form proves the
// claims hash still reproduces.
func TestCLI_Convert_Proof_CBORRoundTripPreservesNumbers(t *testing.T) {
	src := filepath.Join(findTestdata(t), "fixtures", "item-bigint.json")
	if _, err := os.Stat(src); err != nil {
		t.Skipf("no fixture: %v", err)
	}
	cborBytes, err := exec.Command(binaryPath, "convert", "proof", "--to", "cbor", src).Output()
	if err != nil {
		t.Fatalf("json→cbor: %v", err)
	}
	cborPath := filepath.Join(t.TempDir(), "bigint.cbor")
	if err := os.WriteFile(cborPath, cborBytes, 0644); err != nil {
		t.Fatal(err)
	}
	back, err := exec.Command(binaryPath, "convert", "proof", "--to", "json", cborPath).Output()
	if err != nil {
		t.Fatalf("cbor→json: %v", err)
	}
	for _, literal := range []string{"9007199254740993", "18446744073709551615"} {
		if !strings.Contains(string(back), literal) {
			t.Errorf("the CBOR round-trip lost the literal %s:\n%s", literal, back)
		}
	}
	// The subject hash is computed over the canonicalized `s.d`, so a
	// rounded integer would show up here as a Subject Data failure.
	vrf := exec.Command(binaryPath, "verify", cborPath, "--skip-external", "--skip-signatures", "--silent")
	vrf.Env = cleanEnv()
	if err := vrf.Run(); err != nil {
		t.Errorf("verify rejected the CBOR round-trip of a big-integer bundle: %v", err)
	}
}

// TestCLI_Convert_Proof_AutoDetectsBareCBORMap: the self-describing tag 55799
// is a convenience, not a requirement — a producer may emit the bundle as a
// bare CBOR map. --from auto must recognize it by content rather than by the
// tag, and the result must still verify.
func TestCLI_Convert_Proof_AutoDetectsBareCBORMap(t *testing.T) {
	src := filepath.Join(findTestdata(t), "fixtures", "item.json")
	if _, err := os.Stat(src); err != nil {
		t.Skipf("no fixture: %v", err)
	}
	tagged, err := exec.Command(binaryPath, "convert", "proof", "--to", "cbor", src).Output()
	if err != nil {
		t.Fatalf("json→cbor: %v", err)
	}
	if len(tagged) < 3 || tagged[0] != 0xd9 || tagged[1] != 0xd9 || tagged[2] != 0xf7 {
		t.Fatalf("expected a 55799-tagged CBOR bundle, got prefix %x", tagged[:min(3, len(tagged))])
	}
	barePath := filepath.Join(t.TempDir(), "bare.cbor")
	if err := os.WriteFile(barePath, tagged[3:], 0644); err != nil {
		t.Fatal(err)
	}

	back, err := exec.Command(binaryPath, "convert", "proof", "--from", "auto", "--to", "json", barePath).Output()
	if err != nil {
		t.Fatalf("bare cbor→json with --from auto: %v", err)
	}
	if !bytes.Contains(back, []byte(`"t"`)) {
		t.Errorf("round-tripped JSON is missing the subject type code:\n%s", back)
	}
	// verify must accept the bare form directly too.
	if err := exec.Command(binaryPath, "verify", barePath, "--skip-external", "--silent").Run(); err != nil {
		t.Errorf("verify rejected a bare (untagged) CBOR bundle: %v", err)
	}
}

// findTestdata walks up from the test working directory to find
// internal/verify/testdata. Tests in other packages use a similar
// helper; inlined here to keep cmd_test.go independent of internal.
func findTestdata(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 6; i++ {
		candidate := filepath.Join(dir, "internal", "verify", "testdata")
		if st, err := os.Stat(candidate); err == nil && st.IsDir() {
			return candidate
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	t.Skip("testdata not found")
	return ""
}
