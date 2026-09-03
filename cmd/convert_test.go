// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: MIT

package cmd

import (
	"bytes"
	"encoding/json"
	"github.com/truestamp/truestamp-cli/internal/testfixtures"
	"os"
	"os/exec"
	"path/filepath"
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

// TestCLI_Convert_Proof_RoundTrip confirms that json → cbor → json
// survives verify end-to-end (the ultimate round-trip gate: the cbor
// form must be a valid proof that verify can accept).
func TestCLI_Convert_Proof_RoundTrip(t *testing.T) {
	src := testfixtures.Path(testfixtures.ProdDir, testfixtures.ProdComplete)
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

// bigIntegerBundlePath writes the complete production bundle with two
// integers beyond 2^53 injected into its claims. The claims no longer match
// the signed Merkle leaf, so the bundle does not verify; what these tests
// pin is that no conversion path ever rounds the literals, and that the
// verifier canonicalizes them exactly and reports the portability hazard.
func bigIntegerBundlePath(t *testing.T) string {
	t.Helper()
	return rewriteBundle(t, testfixtures.Path(testfixtures.ProdDir, testfixtures.ProdComplete), func(m map[string]any) {
		claims := m["subject"].(map[string]any)["claims"].(map[string]any)
		claims["big"] = json.Number("9007199254740993")
		claims["huge"] = json.Number("18446744073709551615")
	})
}

// TestCLI_Convert_Proof_PrettyPreservesNumbers pins the default --to json
// path against silent rounding: a round trip through `any` decodes every
// JSON number into a float64, and an integer above 2^53 comes back changed,
// altering the very bytes a claims hash is computed over.
func TestCLI_Convert_Proof_PrettyPreservesNumbers(t *testing.T) {
	src := bigIntegerBundlePath(t)
	out, err := exec.Command(binaryPath, "convert", "proof", "--to", "json", src).Output()
	if err != nil {
		t.Fatalf("convert proof --to json: %v", err)
	}
	for _, literal := range []string{"9007199254740993", "18446744073709551615"} {
		if !strings.Contains(string(out), literal) {
			t.Errorf("pretty output lost the literal %s:\n%s", literal, out)
		}
	}
	for _, rounded := range []string{"9007199254740992", "18446744073709552000"} {
		if strings.Contains(string(out), rounded) {
			t.Errorf("pretty output rounded a number to %s:\n%s", rounded, out)
		}
	}
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
// invariant on the value space that breaks it: integers above 2^53 must
// survive JSON -> CBOR -> JSON exactly, and the verifier must canonicalize
// them as carried while reporting that a strict RFC 8785 implementation
// would not.
func TestCLI_Convert_Proof_CBORRoundTripPreservesNumbers(t *testing.T) {
	src := bigIntegerBundlePath(t)
	cborBytes, err := exec.Command(binaryPath, "convert", "proof", "--to", "cbor", src).Output()
	if err != nil {
		t.Fatalf("json to cbor: %v", err)
	}
	cborPath := filepath.Join(t.TempDir(), "bigint.cbor")
	if err := os.WriteFile(cborPath, cborBytes, 0644); err != nil {
		t.Fatal(err)
	}
	back, err := exec.Command(binaryPath, "convert", "proof", "--to", "json", cborPath).Output()
	if err != nil {
		t.Fatalf("cbor to json: %v", err)
	}
	for _, literal := range []string{"9007199254740993", "18446744073709551615"} {
		if !strings.Contains(string(back), literal) {
			t.Errorf("the CBOR round-trip lost the literal %s:\n%s", literal, back)
		}
	}
	for _, path := range []string{src, cborPath} {
		out, _ := runVerifyJSON(t, binaryPath, path, "--offline")
		if st := stepStatuses(out, "Subject Data"); !st["pass"] || !st["warn"] {
			t.Errorf("%s: Subject Data = %v, want the derivations to pass and the portability warn\n%s", path, st, formatCLISteps(out.Steps))
		}
		if !strings.Contains(rawIssueText(out), "not portably verifiable") {
			t.Errorf("%s: the 2^53 warn is missing", path)
		}
	}
}

// TestCLI_Convert_Proof_AutoDetectsBareCBORMap: the self-describing tag
// 55799 is a convenience, not a requirement. --from auto must recognize a
// bare CBOR map by content, and verify must accept it directly.
func TestCLI_Convert_Proof_AutoDetectsBareCBORMap(t *testing.T) {
	src := testfixtures.Path(testfixtures.ProdDir, testfixtures.ProdComplete)
	tagged, err := exec.Command(binaryPath, "convert", "proof", "--to", "cbor", src).Output()
	if err != nil {
		t.Fatalf("json to cbor: %v", err)
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
		t.Fatalf("bare cbor to json with --from auto: %v", err)
	}
	if !bytes.Contains(back, []byte(`"type": "item"`)) {
		t.Errorf("round-tripped JSON is missing the subject type:\n%s", back)
	}
	if err := exec.Command(binaryPath, "verify", barePath, "--offline", "--silent").Run(); err != nil {
		t.Errorf("verify rejected a bare (untagged) CBOR bundle: %v", err)
	}
}
