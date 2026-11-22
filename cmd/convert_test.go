// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"encoding/json/v2"
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

// TestCLI_Convert_KeyID_FixedVector pins the key id of SPEC.md section
// 3.3, the hex of the first 4 bytes of SHA-256(0x60 || the key's 32
// bytes), on a key given in standard base64. Mutant killed: the key id
// derived under the prefix 0x51 (4ceefa4a).
func TestCLI_Convert_KeyID_FixedVector(t *testing.T) {
	out, err := exec.Command(binaryPath, "convert", "keyid",
		"CTwMqDZnPd/QTLSq8aTeSD3a+j2DQxKcGfhhIYJQ65Y=").Output()
	if err != nil {
		t.Fatal(err)
	}
	if got := string(out); got != "97223329\n" {
		t.Errorf("kid: got %q, want %q", got, "97223329\n")
	}
}

// TestCLI_Convert_KeyID_OfTheVectorSigner: the key id convert keyid gives
// the signer of the spec's block vector is the signing_key_id the proof
// gives that key, fbd680e8, and the key_id of the key event that
// introduces it. Mutant killed: the key id derived under the prefix 0x51
// (f2c39df9, which the proof does not carry).
func TestCLI_Convert_KeyID_OfTheVectorSigner(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join(repoRoot(t), "testdata", "verification", "vectors", "positive", "block.proof.json"))
	if err != nil {
		t.Fatal(err)
	}
	var doc struct {
		Proof struct {
			SigningKeyEvent struct {
				Block struct {
					Record struct {
						SigningKeyID string `json:"signing_key_id"`
						Metadata     struct {
							KeyEvent struct {
								KeyID     string `json:"key_id"`
								PublicKey string `json:"public_key"`
							} `json:"key_event"`
						} `json:"metadata"`
					} `json:"record"`
				} `json:"block"`
			} `json:"signing_key_event"`
		} `json:"proof"`
		Signature struct {
			PublicKey string `json:"public_key"`
		} `json:"signature"`
	}
	if err := json.Unmarshal(raw, &doc); err != nil {
		t.Fatal(err)
	}
	record := doc.Proof.SigningKeyEvent.Block.Record
	if record.SigningKeyID != "fbd680e8" || record.Metadata.KeyEvent.KeyID != "fbd680e8" ||
		record.Metadata.KeyEvent.PublicKey != doc.Signature.PublicKey {
		t.Fatalf("vector: signing_key_id %q, key_id %q, event key %q, signer %q; want fbd680e8 twice and the signer's key",
			record.SigningKeyID, record.Metadata.KeyEvent.KeyID, record.Metadata.KeyEvent.PublicKey, doc.Signature.PublicKey)
	}
	out, err := exec.Command(binaryPath, "convert", "keyid", doc.Signature.PublicKey).Output()
	if err != nil {
		t.Fatal(err)
	}
	if got, want := string(out), record.SigningKeyID+"\n"; got != want {
		t.Errorf("convert keyid %s = %q, want %q", doc.Signature.PublicKey, got, want)
	}
}

// TestCLI_Convert_KeyID_JSON verifies the envelope keys and that kid_hex is
// the key id of SPEC.md section 3.3. Mutant killed: kid_hex derived under
// the prefix 0x51 (4ceefa4a).
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
	if m["kid_hex"] != "97223329" {
		t.Errorf("kid_hex: %v", m["kid_hex"])
	}
	if _, ok := m["public_key_hex"].(string); !ok {
		t.Errorf("missing public_key_hex")
	}
}

// convert id --to-zone names its default once, cobra's (default "UTC"),
// and not a second time in its usage text.
func TestCLI_ConvertID_Help_NoDuplicateDefault(t *testing.T) {
	out, err := exec.Command(binaryPath, "convert", "id", "--help").Output()
	if err != nil {
		t.Fatal(err)
	}
	if n := strings.Count(string(out), `(default "UTC")`); n != 1 {
		t.Errorf(`(default "UTC") appears %d times:\n%s`, n, out)
	}
	if strings.Contains(string(out), "(default UTC)") {
		t.Errorf("the usage repeats the default:\n%s", out)
	}
}
