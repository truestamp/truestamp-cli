// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: Apache-2.0

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

// TestCLI_ProofsConvert_RoundTrip confirms that json → cbor → json
// survives verify end-to-end (the ultimate round-trip gate: the cbor
// form must be a valid proof that verify can accept).
func TestCLI_ProofsConvert_RoundTrip(t *testing.T) {
	src := testfixtures.Path(testfixtures.ProdDir, testfixtures.ProdComplete)
	if _, err := os.Stat(src); err != nil {
		t.Skipf("no fixture: %v", err)
	}

	// Convert JSON → CBOR.
	cborPath := filepath.Join(t.TempDir(), "proof.cbor")
	cbor, err := exec.Command(binaryPath, "proofs", "convert", "--to", "cbor", src).Output()
	if err != nil {
		t.Fatalf("json→cbor: %v", err)
	}
	if err := os.WriteFile(cborPath, cbor, 0644); err != nil {
		t.Fatal(err)
	}

	// Convert CBOR → JSON.
	cmd := exec.Command(binaryPath, "proofs", "convert", "--to", "json", cborPath)
	if _, err := cmd.Output(); err != nil {
		t.Fatalf("cbor→json: %v", err)
	}

	// The ultimate test: verify accepts the CBOR form end-to-end (skip
	// external for speed and offline friendliness).
	vrf := exec.Command(binaryPath, "verify", cborPath, "--offline")
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

// TestCLI_ProofsConvert_PrettyPreservesNumbers pins the default --to json
// path against silent rounding: a round trip through `any` decodes every
// JSON number into a float64, and an integer above 2^53 comes back changed,
// altering the very bytes a claims hash is computed over.
func TestCLI_ProofsConvert_PrettyPreservesNumbers(t *testing.T) {
	src := bigIntegerBundlePath(t)
	out, err := exec.Command(binaryPath, "proofs", "convert", "--to", "json", src).Output()
	if err != nil {
		t.Fatalf("proofs convert --to json: %v", err)
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
	compact, err := exec.Command(binaryPath, "proofs", "convert", "--to", "json", "--compact", src).Output()
	if err != nil {
		t.Fatalf("proofs convert --compact: %v", err)
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

// TestCLI_ProofsConvert_CBORRoundTripPreservesNumbers pins the cross-format
// invariant on the value space that breaks it: integers above 2^53 must
// survive JSON -> CBOR -> JSON exactly, and the verifier must canonicalize
// them as carried while reporting that a strict RFC 8785 implementation
// would not.
func TestCLI_ProofsConvert_CBORRoundTripPreservesNumbers(t *testing.T) {
	src := bigIntegerBundlePath(t)
	cborBytes, err := exec.Command(binaryPath, "proofs", "convert", "--to", "cbor", src).Output()
	if err != nil {
		t.Fatalf("json to cbor: %v", err)
	}
	cborPath := filepath.Join(t.TempDir(), "bigint.cbor")
	if err := os.WriteFile(cborPath, cborBytes, 0644); err != nil {
		t.Fatal(err)
	}
	back, err := exec.Command(binaryPath, "proofs", "convert", "--to", "json", cborPath).Output()
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

// TestCLI_ProofsConvert_AutoDetectsBareCBORMap: the self-describing tag
// 55799 is a convenience, not a requirement. --from auto must recognize a
// bare CBOR map by content, and verify must accept it directly.
func TestCLI_ProofsConvert_AutoDetectsBareCBORMap(t *testing.T) {
	src := testfixtures.Path(testfixtures.ProdDir, testfixtures.ProdComplete)
	tagged, err := exec.Command(binaryPath, "proofs", "convert", "--to", "cbor", src).Output()
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
	back, err := exec.Command(binaryPath, "proofs", "convert", "--from", "auto", "--to", "json", barePath).Output()
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
