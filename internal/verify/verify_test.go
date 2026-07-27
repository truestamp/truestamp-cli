// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: MIT

package verify

import (
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path"
	"path/filepath"
	"strings"
	"testing"

	lipgloss "charm.land/lipgloss/v2"
	"github.com/charmbracelet/colorprofile"
	"github.com/truestamp/truestamp-cli/internal/external"
	"github.com/truestamp/truestamp-cli/internal/proof"
	"github.com/truestamp/truestamp-cli/internal/proof/ptype"
	"github.com/truestamp/truestamp-cli/internal/tscrypto"
)

// Minimal valid proof JSON in the compact format.
// Crypto values are fake -- this tests the orchestrator flow, not crypto correctness.
const minimalProofJSON = `{
  "v": 1,
  "t": 20,
  "pk": "CTwMqDZnPd/QTLSq8aTeSD3a+j2DQxKcGfhhIYJQ65Y=",
  "sig": "AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA==",
  "ts": "2026-04-06T23:25:06Z",
  "s": {
    "id": "01TEST",
    "d": {"name": "test"},
    "mh": "cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc",
    "kid": "4ceefa4a"
  },
  "b": {
    "id": "019cf813-99b8-730a-84f1-5a711a9c355e",
    "ph": "1111111111111111111111111111111111111111111111111111111111111111",
    "mr": "2222222222222222222222222222222222222222222222222222222222222222",
    "mh": "4444444444444444444444444444444444444444444444444444444444444444",
    "kid": "4ceefa4a"
  },
  "ip": "AA",
  "cx": [{"t": 40, "net": "testnet", "tx": "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", "memo": "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb", "l": 1, "ep": "AA"}]
}`

func writeTemp(t *testing.T, content string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "proof.json")
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestRun_FileNotFound(t *testing.T) {
	_, err := Run("/nonexistent/proof.json", Options{SkipExternal: true})
	if err == nil {
		t.Error("expected error for missing file")
	}
}

func TestRun_InvalidJSON(t *testing.T) {
	path := writeTemp(t, "not json")
	_, err := Run(path, Options{SkipExternal: true})
	if err == nil {
		t.Error("expected error for invalid JSON")
	}
}

func TestRun_MinimalProof_FailsCryptoButReturnsReport(t *testing.T) {
	path := writeTemp(t, minimalProofJSON)
	report, err := Run(path, Options{SkipExternal: true})
	if err != nil {
		t.Fatalf("Run should not return error for structurally valid proof: %s", err)
	}
	// Crypto will fail (fake hashes/signatures), but we should get a report
	if report == nil {
		t.Fatal("expected non-nil report")
	}
	if report.Passed() {
		t.Error("should fail with fake crypto values")
	}
	if report.FailedCount() == 0 {
		t.Error("should have at least one failure")
	}
}

func TestRun_MinimalProof_VerboseMode(t *testing.T) {
	path := writeTemp(t, minimalProofJSON)
	report, err := Run(path, Options{SkipExternal: true})
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	if len(report.Steps) == 0 {
		t.Error("expected some steps in report")
	}
}

func TestRun_SkipSignatures_StillVerifiesHashes(t *testing.T) {
	path := writeTemp(t, minimalProofJSON)
	report, err := Run(path, Options{SkipExternal: true, SkipSignatures: true})
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}

	// Should have skip steps for signatures
	hasSkip := false
	for _, s := range report.Steps {
		if s.Status == StatusSkip && containsStr(s.Message, "skip-signatures") {
			hasSkip = true
			break
		}
	}
	if !hasSkip {
		t.Error("expected at least one skip step for --skip-signatures")
	}

	// Should still have hash verification steps (pass or fail, not skipped)
	hasHashCheck := false
	for _, s := range report.Steps {
		if (s.Status == StatusPass || s.Status == StatusFail) && containsStr(s.Message, "hash") {
			hasHashCheck = true
			break
		}
	}
	if !hasHashCheck {
		t.Error("hash checks should still run when signatures are skipped")
	}
}

func containsStr(s, substr string) bool {
	return len(s) >= len(substr) && indexOf(s, substr) >= 0
}

func indexOf(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}

// Proof with stellar and bitcoin commitments (fake crypto, but exercises commitment code paths)
const proofWithCommitmentsJSON = `{
  "v": 1,
  "t": 20,
  "pk": "CTwMqDZnPd/QTLSq8aTeSD3a+j2DQxKcGfhhIYJQ65Y=",
  "sig": "AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA==",
  "ts": "2026-04-06T23:25:06Z",
  "s": {
    "id": "01TEST",
    "d": {"name": "test"},
    "mh": "cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc",
    "kid": "4ceefa4a"
  },
  "b": {
    "id": "019cf813-99b8-730a-84f1-5a711a9c355e",
    "ph": "1111111111111111111111111111111111111111111111111111111111111111",
    "mr": "2222222222222222222222222222222222222222222222222222222222222222",
    "mh": "4444444444444444444444444444444444444444444444444444444444444444",
    "kid": "4ceefa4a"
  },
  "ip": "AA",
  "cx": [
    {
      "t": 40,
      "net": "testnet",
      "tx": "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
      "memo": "3333333333333333333333333333333333333333333333333333333333333333",
      "l": 100,
      "ts": "2026-04-06T23:25:06Z",
      "ep": "AA"
    },
    {
      "t": 41,
      "net": "regtest",
      "tx": "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
      "op": "6666666666666666666666666666666666666666666666666666666666666666",
      "rtx": "0200000001abcdef",
      "txp": "aabbccdd",
      "bmr": "cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc",
      "h": 500,
      "ts": "2026-04-06T23:25:06Z",
      "ep": "AA"
    }
  ]
}`

func TestRun_WithCommitments_ExercisesAllPaths(t *testing.T) {
	path := writeTemp(t, proofWithCommitmentsJSON)
	report, err := Run(path, Options{SkipExternal: true, SkipSignatures: true})
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	// Will have failures (fake hashes) but should exercise stellar/bitcoin code paths
	if len(report.Steps) == 0 {
		t.Error("expected steps in report")
	}

	// Should have bitcoin groups (stellar has no local checks in new format beyond epoch proof)
	groups := make(map[string]bool)
	for _, s := range report.Steps {
		groups[s.Group] = true
	}
	if !groups["Bitcoin Commitment"] {
		t.Error("expected Bitcoin Commitment group in report")
	}
}

func TestReport_Passed_AllPass(t *testing.T) {
	r := &Report{Steps: []Step{
		{Status: StatusPass},
		{Status: StatusPass},
		{Status: StatusSkip},
		{Status: StatusInfo},
	}}
	if !r.Passed() {
		t.Error("all pass/skip/info should be passed")
	}
}

func TestReport_Passed_OneFail(t *testing.T) {
	r := &Report{Steps: []Step{
		{Status: StatusPass},
		{Status: StatusFail},
	}}
	if r.Passed() {
		t.Error("one fail should make it not passed")
	}
}

func TestReport_Passed_Empty(t *testing.T) {
	r := &Report{}
	if !r.Passed() {
		t.Error("empty report should be passed")
	}
}

func TestReport_FailedCount(t *testing.T) {
	r := &Report{Steps: []Step{
		{Status: StatusFail},
		{Status: StatusPass},
		{Status: StatusFail},
		{Status: StatusFail},
	}}
	if r.FailedCount() != 3 {
		t.Errorf("failed count: got %d, want 3", r.FailedCount())
	}
}

func TestPresent_DoesNotPanic(t *testing.T) {
	r := &Report{
		Filename: "test.json",
		FileSize: 100,
		Steps: []Step{
			{Group: "Test", Status: StatusPass, Message: "ok"},
			{Group: "Test", Status: StatusFail, Message: "bad"},
			{Group: "Test", Status: StatusSkip, Message: "skipped"},
			{Group: "Other", Status: StatusInfo, Message: "info"},
		},
	}
	// Just verify it doesn't panic -- covers all Status branches in presenter
	Present(r)
}

// =====================================================================
// Direct unit tests for internal verify functions
// =====================================================================

// testPubKeyB64 is a well-formed 32-byte Ed25519 public key whose
// derived key id is testKeyID.
const (
	testPubKeyB64 = "CTwMqDZnPd/QTLSq8aTeSD3a+j2DQxKcGfhhIYJQ65Y="
	testKeyID     = "4ceefa4a"
)

// stepsIn returns every step filed under group.
func stepsIn(r *Report, group string) []Step {
	var out []Step
	for _, s := range r.Steps {
		if s.Group == group {
			out = append(out, s)
		}
	}
	return out
}

// onlyStep returns the single step filed under group, failing the test
// when the count is anything other than one. Appendix E.22 requires
// exactly one graded Key Binding result per run, so "how many" is part
// of what these tests assert.
func onlyStep(t *testing.T, r *Report, group string) Step {
	t.Helper()
	got := stepsIn(r, group)
	if len(got) != 1 {
		t.Fatalf("%s: got %d steps, want exactly 1: %+v", group, len(got), got)
	}
	return got[0]
}

func TestVerifySigningKey_InvalidPublicKey(t *testing.T) {
	r := &Report{}
	bundle := &proof.ProofBundle{
		PublicKey: "not-valid-base64!!!",
	}
	pubkey, keyID := verifySigningKey(r, bundle)
	if pubkey != nil {
		t.Error("invalid base64 key should return nil pubkey")
	}
	if keyID != "" {
		t.Error("invalid base64 key should return empty keyID")
	}
	if r.Passed() {
		t.Error("invalid base64 key should fail")
	}
}

func TestVerifySigningKey_ValidKey(t *testing.T) {
	r := &Report{}
	bundle := &proof.ProofBundle{PublicKey: testPubKeyB64}
	pubkey, keyID := verifySigningKey(r, bundle)
	if pubkey == nil {
		t.Error("valid key should return non-nil pubkey")
	}
	if keyID != testKeyID {
		t.Errorf("expected keyID %s, got %q", testKeyID, keyID)
	}
	step := onlyStep(t, r, groupSigningKey)
	if step.Status != StatusPass || !containsStr(step.Message, "Public key valid") {
		t.Errorf("valid key should produce one pass step, got [%v] %s", step.Status, step.Message)
	}
}

// --- F8: E.17 Key Binding ---

// TestVerifyKeyBinding_UnreachableKeyringSkips pins the F8a fix: an
// unreachable keyring establishes nothing either way, and E.22 forbids a
// skipped external check from failing a proof. Before this, a network
// outage reported a sound proof as defective.
func TestVerifyKeyBinding_UnreachableKeyringSkips(t *testing.T) {
	r := &Report{}
	pubkey := decodeTestPubKey(t)
	verifyKeyBinding(r, testPubKeyB64, pubkey, testKeyID, Options{
		KeyringURL: "http://127.0.0.1:1/nonexistent",
	})
	for _, s := range r.Steps {
		if s.Status == StatusFail {
			t.Errorf("unreachable keyring must not fail any step, got: %s", s.Message)
		}
	}
	steps := stepsIn(r, groupKeyBinding)
	if len(steps) == 0 || steps[0].Status != StatusSkip {
		t.Fatalf("expected a leading Key Binding skip, got %+v", steps)
	}
	if steps[0].Category != CatCryptographic {
		t.Errorf("category: got %q, want %q", steps[0].Category, CatCryptographic)
	}
	if !r.Passed() {
		t.Error("report must still pass when only the keyring was unreachable")
	}
}

// TestVerifyKeyBinding_KeyNotPublishedFails is the other half of F8a:
// only a keyring that answered and does not vouch for the key may fail.
func TestVerifyKeyBinding_KeyNotPublishedFails(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		json.NewEncoder(w).Encode(map[string]any{
			"version": "1.0",
			"keys": []map[string]any{
				{"key_id": "deadbeef", "public_key": "AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA=", "sequence": 0, "active": true},
			},
		})
	}))
	defer server.Close()

	r := &Report{}
	verifyKeyBinding(r, testPubKeyB64, decodeTestPubKey(t), testKeyID, Options{KeyringURL: server.URL})
	step := onlyStep(t, r, groupKeyBinding)
	if step.Status != StatusFail {
		t.Errorf("a keyring that does not carry the key must fail: got [%v] %s", step.Status, step.Message)
	}
}

func TestVerifyKeyBinding_ExternalSuccess(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		json.NewEncoder(w).Encode(map[string]any{
			"version": "1.0",
			"keys": []map[string]any{
				{"key_id": testKeyID, "public_key": testPubKeyB64, "sequence": 0, "active": true},
			},
		})
	}))
	defer server.Close()

	r := &Report{}
	verifyKeyBinding(r, testPubKeyB64, decodeTestPubKey(t), testKeyID, Options{KeyringURL: server.URL})
	step := onlyStep(t, r, groupKeyBinding)
	if step.Status != StatusPass || !containsStr(step.Message, "confirmed via keyring") {
		t.Errorf("valid keyring response should produce a Key Binding pass, got [%v] %s", step.Status, step.Message)
	}
}

// TestVerifyKeyBinding_AlwaysReportsARow covers E.22's "reported, not
// omitted" requirement across every branch, including the one F16 made
// reachable: a bundle whose pk does not decode used to produce no Key
// Binding row at all, so a report could be read as having established a
// binding it never attempted.
func TestVerifyKeyBinding_AlwaysReportsARow(t *testing.T) {
	pubkey := decodeTestPubKey(t)
	cases := []struct {
		name   string
		pubkey []byte
		opts   Options
	}{
		{"no usable public key", nil, Options{SkipExternal: true}},
		{"--skip-signatures", pubkey, Options{SkipSignatures: true}},
		{"--skip-external", pubkey, Options{SkipExternal: true}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			r := &Report{}
			verifyKeyBinding(r, testPubKeyB64, tc.pubkey, testKeyID, tc.opts)
			step := onlyStep(t, r, groupKeyBinding)
			if step.Status != StatusSkip {
				t.Errorf("status: got %v, want skip (%s)", step.Status, step.Message)
			}
			if step.Category != CatCryptographic {
				t.Errorf("category: got %q, want %q", step.Category, CatCryptographic)
			}
		})
	}
}

// TestRunFromBytes_KeyBindingRowAlwaysPresent asserts the whole-pipeline
// invariant rather than the helper's: whatever the bundle looks like,
// the report carries exactly one Key Binding row, and no Signing Key row
// claims a keyring result.
func TestRunFromBytes_KeyBindingRowAlwaysPresent(t *testing.T) {
	report, err := RunFromBytes([]byte(minimalProofJSON), "(test)", Options{SkipExternal: true})
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	step := onlyStep(t, report, groupKeyBinding)
	if step.Status != StatusSkip {
		t.Errorf("offline run: got %v, want skip", step.Status)
	}
	for _, s := range stepsIn(report, groupSigningKey) {
		if containsStr(s.Message, "eyring") {
			t.Errorf("keyring result leaked into the Signing Key group: %s", s.Message)
		}
	}
}

func decodeTestPubKey(t *testing.T) []byte {
	t.Helper()
	b, err := base64.StdEncoding.DecodeString(testPubKeyB64)
	if err != nil {
		t.Fatalf("decoding test public key: %s", err)
	}
	return b
}

func TestVerifyProofSignature_MissingPubkey(t *testing.T) {
	r := &Report{}
	bundle := &proof.ProofBundle{
		Version:   1,
		Signature: "AAAA",
		Timestamp: "2026-04-06T23:25:06Z",
	}
	verifyProofSignature(r, bundle, nil, "", "", "", nil, Options{})
	hasFail := false
	for _, s := range r.Steps {
		if s.Status == StatusFail && containsStr(s.Message, "missing derived data") {
			hasFail = true
		}
	}
	if !hasFail {
		t.Error("missing pubkey should produce a failure")
	}
}

func TestVerifyProofSignature_MissingBlockHash(t *testing.T) {
	r := &Report{}
	bundle := &proof.ProofBundle{
		Version:   1,
		Signature: "AAAA",
		Timestamp: "2026-04-06T23:25:06Z",
	}
	verifyProofSignature(r, bundle, []byte("fakepubkey32byteslong!!!!!!!!!!??"), "4ceefa4a", "aabb", "", nil, Options{})
	hasFail := false
	for _, s := range r.Steps {
		if s.Status == StatusFail && containsStr(s.Message, "no block hash") {
			hasFail = true
		}
	}
	if !hasFail {
		t.Error("empty block hash should produce a failure")
	}
}

// TestVerifyProofSignature_CannotVerifyPhrasing pins Appendix E.16's
// required wording on every upstream-derivation-failure arm. E.16 quotes
// "cannot verify proof signature (missing derived data)" and E.9 makes
// an undecodable pk abort here rather than pass silently, so a report
// that names its own reason must still carry the shared phrase.
func TestVerifyProofSignature_CannotVerifyPhrasing(t *testing.T) {
	const canonical = "Cannot verify proof signature (missing derived data)"
	bundle := &proof.ProofBundle{Version: 1, Signature: "AAAA", Timestamp: "2026-04-06T23:25:06Z"}
	pubkey := []byte("fakepubkey32byteslong!!!!!!!!!!??")

	cases := []struct {
		name        string
		pubkey      []byte
		subjectHash string
		blockHash   string
		epochRoots  []string
	}{
		{"no public key", nil, "aabb", "ccdd", []string{"eeff"}},
		{"no subject hash", pubkey, "", "ccdd", []string{"eeff"}},
		{"no block hash", pubkey, "aabb", "", []string{"eeff"}},
		{"no epoch root", pubkey, "aabb", "ccdd", []string{""}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			r := &Report{}
			verifyProofSignature(r, bundle, tc.pubkey, "4ceefa4a", tc.subjectHash, tc.blockHash, tc.epochRoots, Options{})
			step := onlyStep(t, r, groupProof)
			if step.Status != StatusFail {
				t.Fatalf("status: got %v, want fail", step.Status)
			}
			if !strings.HasPrefix(step.Message, canonical) {
				t.Errorf("message %q does not open with E.16's required phrase %q", step.Message, canonical)
			}
		})
	}
}

func TestVerifyProofSignature_SkipSignatures(t *testing.T) {
	r := &Report{}
	bundle := &proof.ProofBundle{
		Version:   1,
		Signature: "AAAA",
		Timestamp: "2026-04-06T23:25:06Z",
	}
	verifyProofSignature(r, bundle, []byte("fakepubkey32byteslong!!!!!!!!!!??"), "4ceefa4a", "aabb", "ccdd", nil, Options{SkipSignatures: true})
	hasSkip := false
	for _, s := range r.Steps {
		if s.Status == StatusSkip && containsStr(s.Message, "skip-signatures") {
			hasSkip = true
		}
	}
	if !hasSkip {
		t.Error("should skip proof signature when --skip-signatures")
	}
}

func TestVerifyProofSignature_BadSignature(t *testing.T) {
	r := &Report{}
	bundle := &proof.ProofBundle{
		Version:   1,
		Signature: "not-base64!!!",
		Timestamp: "2026-04-06T23:25:06Z",
	}
	// Use a fake but valid-length pubkey
	pubkey := make([]byte, 32)
	verifyProofSignature(r, bundle, pubkey, "4ceefa4a", "aabbccddaabbccddaabbccddaabbccddaabbccddaabbccddaabbccddaabbccdd", "1111111111111111111111111111111111111111111111111111111111111111", nil, Options{})
	hasFail := false
	for _, s := range r.Steps {
		if s.Status == StatusFail && containsStr(s.Message, "signature") {
			hasFail = true
		}
	}
	if !hasFail {
		t.Error("bad signature should produce a failure")
	}
}

// TestDeriveClaimsHash_AbsentSubjectDataFails inverts the previous
// "redacted" skip. Appendix E.10 says any missing input fails this step,
// and a skip would report the derivation as not applicable when it
// applied and could not be performed. The category stays Cryptographic:
// E.22's Data Integrity exception is a closed list of two, and a result
// that names no hash is not one of them.
func TestDeriveClaimsHash_AbsentSubjectDataFails(t *testing.T) {
	for _, raw := range []string{"null", ""} {
		r := &Report{}
		hash := deriveClaimsHash(r, []byte(raw))
		if hash != "" {
			t.Errorf("%q claims should return empty hash, got %q", raw, hash)
		}
		step := onlyStep(t, r, groupSubjectData)
		if step.Status != StatusFail {
			t.Errorf("%q claims: got %v, want fail (%s)", raw, step.Status, step.Message)
		}
		if step.Category != CatCryptographic {
			t.Errorf("%q claims category: got %q, want %q", raw, step.Category, CatCryptographic)
		}
	}
}

func TestDeriveClaimsHash_ValidClaims(t *testing.T) {
	r := &Report{}
	hash := deriveClaimsHash(r, []byte(`{"name":"test"}`))
	if hash == "" {
		t.Error("valid claims should produce a non-empty hash")
	}
	hasPass := false
	for _, s := range r.Steps {
		if s.Status == StatusPass && containsStr(s.Message, "Claims hash derived") {
			hasPass = true
		}
	}
	if !hasPass {
		t.Error("valid claims should produce a pass step")
	}
}

// TestDeriveItemHash_MissingCompositeFields covers F26's category split
// and the s.id term the old predicate omitted.
//
// Appendix E.22 files a Subject Data result that names a hash without
// completing a derivation under Data Integrity; the missing-composite
// case is one of exactly two such results. The absent-data case names no
// hash and stays Cryptographic. s.id belongs in the composite predicate
// because it is an input to the 0x13 preimage — without the term,
// ComputeItemHash frames len32(0) and returns a digest over a short
// preimage with no error, i.e. a fabricated pass.
func TestDeriveItemHash_MissingCompositeFields(t *testing.T) {
	const claimsHash = "cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc"
	full := proof.Subject{
		ID:           "01KY9ZWEX0248J48HK6D248NAN",
		MetadataHash: "dddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddd",
		SigningKeyID: "4ceefa4a",
	}
	cases := []struct {
		name     string
		mutate   func(*proof.Subject)
		claims   string
		wantCat  string
		wantWord string
	}{
		{"no claims hash", func(*proof.Subject) {}, "", CatCryptographic, "no claims hash"},
		{"no s.id", func(s *proof.Subject) { s.ID = "" }, claimsHash, CatDataIntegrity, "composite subject hash"},
		{"no s.mh", func(s *proof.Subject) { s.MetadataHash = "" }, claimsHash, CatDataIntegrity, "composite subject hash"},
		{"no s.kid", func(s *proof.Subject) { s.SigningKeyID = "" }, claimsHash, CatDataIntegrity, "composite subject hash"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			subject := full
			tc.mutate(&subject)
			r := &Report{}
			if hash := deriveItemHash(r, &subject, tc.claims); hash != "" {
				t.Errorf("missing input should return empty hash, got %q", hash)
			}
			step := onlyStep(t, r, groupSubjectData)
			if step.Status != StatusFail {
				t.Errorf("status: got %v, want fail", step.Status)
			}
			if step.Category != tc.wantCat {
				t.Errorf("category: got %q, want %q (%s)", step.Category, tc.wantCat, step.Message)
			}
			if !containsStr(step.Message, tc.wantWord) {
				t.Errorf("message %q should name %q", step.Message, tc.wantWord)
			}
		})
	}
}

// TestDeriveObservationHash_MissingCompositeFields mirrors the item
// split on the entropy path — the two composites share E.22's rule.
func TestDeriveObservationHash_MissingCompositeFields(t *testing.T) {
	const entropyHash = "cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc"
	full := proof.Subject{
		ID:           "019cf813-99b8-730a-84f1-5a711a9c355e",
		MetadataHash: "dddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddd",
		SigningKeyID: "4ceefa4a",
	}
	cases := []struct {
		name    string
		mutate  func(*proof.Subject)
		entropy string
		wantCat string
	}{
		{"no entropy hash", func(*proof.Subject) {}, "", CatCryptographic},
		{"no s.id", func(s *proof.Subject) { s.ID = "" }, entropyHash, CatDataIntegrity},
		{"no s.mh", func(s *proof.Subject) { s.MetadataHash = "" }, entropyHash, CatDataIntegrity},
		{"no s.kid", func(s *proof.Subject) { s.SigningKeyID = "" }, entropyHash, CatDataIntegrity},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			subject := full
			tc.mutate(&subject)
			r := &Report{}
			if hash := deriveObservationHash(r, &subject, tc.entropy); hash != "" {
				t.Errorf("missing input should return empty hash, got %q", hash)
			}
			step := onlyStep(t, r, groupSubjectData)
			if step.Status != StatusFail {
				t.Errorf("status: got %v, want fail", step.Status)
			}
			if step.Category != tc.wantCat {
				t.Errorf("category: got %q, want %q (%s)", step.Category, tc.wantCat, step.Message)
			}
		})
	}
}

// TestVerifyVersion_IsTheWholeStructureGroup pins E.25 containment against
// the Structure group: D.4 reports exactly one Structure row and it
// establishes `v == 1`, so this group must never emit a second row.
func TestVerifyVersion_IsTheWholeStructureGroup(t *testing.T) {
	for _, tc := range []struct {
		name    string
		version int
		want    Status
	}{
		{"v1", 1, StatusPass},
		{"v0", 0, StatusFail},
	} {
		t.Run(tc.name, func(t *testing.T) {
			r := &Report{}
			bundle := &proof.ProofBundle{T: ptype.Item, Version: tc.version}
			verifyVersion(r, bundle)
			step := onlyStep(t, r, groupStructure)
			if step.Status != tc.want {
				t.Errorf("status: got %v, want %v (%s)", step.Status, tc.want, step.Message)
			}
			if step.Category != CatStructural {
				t.Errorf("category: got %q, want %q", step.Category, CatStructural)
			}
		})
	}
}

// TestStructuralDefectsStillDetected proves the four checks removed from the
// Structure group cost the verifier no detection. Two of them (unregistered
// `t`, empty `cx`) are E.6 hard rejections applied before any step runs; the
// other two (`b.id`, `b.mr`) are E.14's five-field precondition, reported by
// the Block Hash step.
func TestStructuralDefectsStillDetected(t *testing.T) {
	base := fixtureBytes(t, "appendix-d-item.json")

	for _, tc := range []struct {
		name     string
		mutate   func(map[string]any)
		rejects  bool // true = E.6 hard rejection, no report at all
		failedIn string
	}{
		{
			name:    "unregistered subject type",
			mutate:  func(m map[string]any) { m["t"] = 99 },
			rejects: true,
		},
		{
			name:    "empty cx",
			mutate:  func(m map[string]any) { m["cx"] = []any{} },
			rejects: true,
		},
		{
			name:     "block missing id",
			mutate:   func(m map[string]any) { delete(m["b"].(map[string]any), "id") },
			failedIn: groupBlockHash,
		},
		{
			name:     "block missing merkle_root",
			mutate:   func(m map[string]any) { delete(m["b"].(map[string]any), "mr") },
			failedIn: groupBlockHash,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			raw := mutateBundle(t, base, tc.mutate)
			rpt, err := RunFromBytes(raw, "tamper.json", Options{SkipExternal: true, SkipSignatures: true})
			if tc.rejects {
				if err == nil {
					t.Fatalf("want E.6 hard rejection, got a report:\n%s", formatSteps(rpt.Steps))
				}
				return
			}
			if err != nil {
				t.Fatalf("RunFromBytes: %v", err)
			}
			if rpt.Passed() {
				t.Fatal("tampered bundle passed verification")
			}
			var found bool
			for _, s := range rpt.Steps {
				if s.Group == tc.failedIn && s.Status == StatusFail {
					found = true
				}
			}
			if !found {
				t.Errorf("no fail row in group %q:\n%s", tc.failedIn, formatSteps(rpt.Steps))
			}
		})
	}
}

func TestDeriveBlockHash_Valid(t *testing.T) {
	r := &Report{}
	block := proof.Block{ID: "019cf813-99b8-730a-84f1-5a711a9c355e", PreviousBlockHash: "0000000000000000000000000000000000000000000000000000000000000000", MerkleRoot: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", MetadataHash: "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb", SigningKeyID: "4ceefa4a"}
	hash := deriveBlockHash(r, block)
	if hash == "" {
		t.Error("valid block should produce a non-empty hash")
	}
	hasPass := false
	for _, s := range r.Steps {
		if s.Status == StatusPass && containsStr(s.Message, "Block hash derived") {
			hasPass = true
		}
	}
	if !hasPass {
		t.Error("valid block should produce a pass step")
	}
}

func TestVerifyBitcoinCommitment_FullPath(t *testing.T) {
	// Real bitcoin commitment data embedded in the new compact format
	r := &Report{}
	commits := []proof.ExternalCommit{
		{
			Type:            ptype.CommitmentBitcoin,
			Network:         "regtest",
			TransactionHash: "29c38e578c10ff89e1a0392d97b3f4fd4c83c439a5ad2977f294182724718752",
			OpReturn:        "16c095556481e3f5f5c410cd2d5cad50d17f09a5ae16ac2deda88d64db5c424b",
			RawTxHex:        "02000000000101cf12c8f31bb02b316407637338df0587e54b193b48a8e823c95f7d6cb80049780000000000fdffffff010000000000000000226a2016c095556481e3f5f5c410cd2d5cad50d17f09a5ae16ac2deda88d64db5c424b01403c1e5d4c2edf2f24e609bfcdbc040648df7cd8d1d6af3f8e9b16e01b87ebe173b9a047848b7370f8622f6d43a3947bf99f5e9a6f706e48595bdce662195295eb00000000",
			TxoutproofHex:   "0000002026cf000a60782b1b57c018b6482e6837c1165d5e2aa06f92d199163481b6c07b150edc6623b16f07e3239de42b6f67d08f5d71b4d06e35685dcaab533f951d5e395ab869ffff7f200000000002000000028205b7caabd985090d0ba024a7d2ce998e53d49122989f8eac68b6f25558bbee52877124271894f27729ada539c4834cfdf4b3972d39a0e189ff108c578ec3290105",
			BlockMerkleRoot: "5e1d953f53abca5d68356ed0b4715d8fd0676f2be49d23e3076fb12366dc0e15",
			BlockHeight:     9312,
			EpochProof:      "AA",
		},
	}

	verifyBitcoinCommitments(r, commits, Options{SkipExternal: true})

	// Should have pass steps for OP_RETURN, txid, merkle proof
	passCount := 0
	for _, step := range r.Steps {
		if step.Status == StatusPass {
			passCount++
		}
	}
	if passCount < 3 {
		t.Errorf("expected at least 3 pass steps for bitcoin verification, got %d", passCount)
		for _, step := range r.Steps {
			t.Logf("  %d: %s", step.Status, step.Message)
		}
	}
}

func TestVerifyBitcoinCommitment_MalformedTx(t *testing.T) {
	r := &Report{}
	commits := []proof.ExternalCommit{
		{
			Type:            ptype.CommitmentBitcoin,
			Network:         "regtest",
			TransactionHash: "aaaa",
			OpReturn:        "bbbb",
			RawTxHex:        "0100",
			TxoutproofHex:   "cccc",
			BlockMerkleRoot: "dddd",
			BlockHeight:     1,
			EpochProof:      "AA",
		},
	}

	verifyBitcoinCommitments(r, commits, Options{SkipExternal: true})
	hasFail := false
	for _, step := range r.Steps {
		if step.Status == StatusFail {
			hasFail = true
		}
	}
	if !hasFail {
		t.Error("malformed bitcoin tx should produce failures")
	}
}

func TestVerifyBitcoinCommitment_ExternalSkipped(t *testing.T) {
	r := &Report{}
	commits := []proof.ExternalCommit{
		{
			Type:            ptype.CommitmentBitcoin,
			Network:         "regtest",
			TransactionHash: "29c38e578c10ff89e1a0392d97b3f4fd4c83c439a5ad2977f294182724718752",
			OpReturn:        "16c095556481e3f5f5c410cd2d5cad50d17f09a5ae16ac2deda88d64db5c424b",
			RawTxHex:        "02000000000101cf12c8f31bb02b316407637338df0587e54b193b48a8e823c95f7d6cb80049780000000000fdffffff010000000000000000226a2016c095556481e3f5f5c410cd2d5cad50d17f09a5ae16ac2deda88d64db5c424b01403c1e5d4c2edf2f24e609bfcdbc040648df7cd8d1d6af3f8e9b16e01b87ebe173b9a047848b7370f8622f6d43a3947bf99f5e9a6f706e48595bdce662195295eb00000000",
			TxoutproofHex:   "0000002026cf000a60782b1b57c018b6482e6837c1165d5e2aa06f92d199163481b6c07b150edc6623b16f07e3239de42b6f67d08f5d71b4d06e35685dcaab533f951d5e395ab869ffff7f200000000002000000028205b7caabd985090d0ba024a7d2ce998e53d49122989f8eac68b6f25558bbee52877124271894f27729ada539c4834cfdf4b3972d39a0e189ff108c578ec3290105",
			BlockMerkleRoot: "5e1d953f53abca5d68356ed0b4715d8fd0676f2be49d23e3076fb12366dc0e15",
			BlockHeight:     9312,
			EpochProof:      "AA",
		},
	}

	verifyBitcoinCommitments(r, commits, Options{SkipExternal: false})
	hasSkip := false
	for _, step := range r.Steps {
		if step.Status == StatusSkip && containsStr(step.Message, "no public API") {
			hasSkip = true
		}
	}
	if !hasSkip {
		t.Error("regtest with external enabled should produce skip for no public API")
	}
}

// TestRun_RealProof uses the real proof fixture to exercise all success paths
// including valid Ed25519 signature verification.
func TestRun_RealProof(t *testing.T) {
	report, err := Run("testdata/proof_item.json", Options{SkipExternal: true})
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	if !report.Passed() {
		for _, s := range report.Steps {
			if s.Status == StatusFail {
				t.Errorf("FAIL: [%s] %s", s.Group, s.Message)
			}
		}
		t.Fatal("real proof should pass all checks with --skip-external")
	}

	// Should have pass step for proof signature
	hasSigPass := false
	for _, s := range report.Steps {
		if s.Status == StatusPass && containsStr(s.Message, "signature") {
			hasSigPass = true
			break
		}
	}
	if !hasSigPass {
		t.Error("expected at least one signature pass step")
	}
}

func TestRun_RealProof_NonVerbose(t *testing.T) {
	report, err := Run("testdata/proof_item.json", Options{SkipExternal: true})
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	if !report.Passed() {
		t.Fatal("real proof should pass all checks")
	}
}

func TestPresent_TemporalSummary(t *testing.T) {
	r := &Report{
		Filename: "test.json",
		FileSize: 100,
		Steps: []Step{
			{Group: "Test", Status: StatusPass, Message: "ok"},
		},
		Temporal: TemporalSummary{
			ClaimedAt:   "2025-01-01T00:00:00Z",
			SubmittedAt: "2025-01-01T00:01:30Z",
			CommittedAt: "2025-01-01T00:02:00Z",
		},
	}
	// Should not panic and should display all temporal fields
	Present(r)
}

func TestPresent_TemporalSummary_Partial(t *testing.T) {
	r := &Report{
		Filename: "test.json",
		FileSize: 100,
		Steps: []Step{
			{Group: "Test", Status: StatusPass, Message: "ok"},
		},
		Temporal: TemporalSummary{
			CommittedAt: "2025-01-01T00:02:00Z",
		},
	}
	Present(r)
}

func TestPresent_NormalMode_FiltersToFailures(t *testing.T) {
	r := &Report{
		Filename: "test.json",
		FileSize: 100,
		Steps: []Step{
			{Group: "Test", Status: StatusPass, Message: "pass"},
			{Group: "Test", Status: StatusFail, Message: "fail"},
			{Group: "Test", Status: StatusSkip, Message: "skip"},
			{Group: "Other", Status: StatusInfo, Message: "info"},
		},
	}
	// Should not panic -- normal mode only shows failures
	Present(r)
}

func TestPresent_PassedReport(t *testing.T) {
	r := &Report{
		Filename: "test.json",
		FileSize: 100,
		Steps: []Step{
			{Group: "Test", Status: StatusPass, Message: "ok"},
		},
	}
	Present(r) // should print PASSED
}

func TestExtractClaimsTimestamp_Valid(t *testing.T) {
	ts := extractClaimsTimestamp([]byte(`{"timestamp": "2025-01-01T00:00:00Z", "name": "test"}`))
	if ts != "2025-01-01T00:00:00Z" {
		t.Errorf("expected timestamp, got %q", ts)
	}
}

func TestExtractClaimsTimestamp_Missing(t *testing.T) {
	ts := extractClaimsTimestamp([]byte(`{"name": "test"}`))
	if ts != "" {
		t.Errorf("expected empty, got %q", ts)
	}
}

func TestExtractClaimsTimestamp_InvalidJSON(t *testing.T) {
	ts := extractClaimsTimestamp([]byte(`not json`))
	if ts != "" {
		t.Errorf("expected empty for invalid JSON, got %q", ts)
	}
}

func TestExtractClaimsTimestamp_NonStringTimestamp(t *testing.T) {
	ts := extractClaimsTimestamp([]byte(`{"timestamp": 12345}`))
	if ts != "" {
		t.Errorf("expected empty for non-string timestamp, got %q", ts)
	}
}

func TestRunFromBytes_Valid(t *testing.T) {
	data, err := os.ReadFile("testdata/proof_item.json")
	if err != nil {
		t.Fatalf("reading test proof: %s", err)
	}
	report, err := RunFromBytes(data, "(test)", Options{SkipExternal: true})
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	if len(report.Steps) == 0 {
		t.Error("expected steps in report")
	}
	if report.Filename != "(test)" {
		t.Errorf("filename: got %q, want (test)", report.Filename)
	}
	if report.FileSize != int64(len(data)) {
		t.Errorf("file size: got %d, want %d", report.FileSize, len(data))
	}
}

func TestRunFromBytes_InvalidJSON(t *testing.T) {
	_, err := RunFromBytes([]byte("not json"), "(bad)", Options{SkipExternal: true})
	if err == nil {
		t.Error("expected error for invalid JSON")
	}
}

func TestHashComparison_Match(t *testing.T) {
	data, err := os.ReadFile("testdata/proof_item.json")
	if err != nil {
		t.Fatalf("reading test proof: %s", err)
	}
	bundle, err := proof.ParseBytes(data)
	if err != nil {
		t.Fatalf("parsing proof: %s", err)
	}
	claims := parseClaims(bundle.RawData)

	report, err := RunFromBytes(data, "(test)", Options{
		SkipExternal: true,
		ExpectedHash: claims.Hash,
	})
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	if !report.HashMatched() {
		t.Error("expected hash to match")
	}
	if report.HashProvided != claims.Hash {
		t.Errorf("HashProvided: got %q, want %q", report.HashProvided, claims.Hash)
	}
}

func TestHashComparison_Mismatch(t *testing.T) {
	data, err := os.ReadFile("testdata/proof_item.json")
	if err != nil {
		t.Fatalf("reading test proof: %s", err)
	}
	report, err := RunFromBytes(data, "(test)", Options{
		SkipExternal: true,
		ExpectedHash: "deadbeef",
	})
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	if report.HashMatched() {
		t.Error("expected hash to not match")
	}
}

func TestHashComparison_NotProvided(t *testing.T) {
	data, err := os.ReadFile("testdata/proof_item.json")
	if err != nil {
		t.Fatalf("reading test proof: %s", err)
	}
	report, err := RunFromBytes(data, "(test)", Options{SkipExternal: true})
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	if report.HashProvided != "" {
		t.Error("HashProvided should be empty when --hash not used")
	}
	if report.HashMatched() {
		t.Error("HashMatched should be false when no hash provided")
	}
}

func TestReport_ProofPassed_ExcludesHashFailure(t *testing.T) {
	r := &Report{
		Steps: []Step{
			{Group: "Hash Comparison", Status: StatusFail, Message: "mismatch"},
			{Group: "Signing Key", Status: StatusPass, Message: "ok"},
		},
		HashProvided: "deadbeef",
	}
	if !r.ProofPassed() {
		t.Error("ProofPassed should be true when only hash comparison failed")
	}
	if r.ProofFailedCount() != 0 {
		t.Errorf("ProofFailedCount: got %d, want 0", r.ProofFailedCount())
	}
	if r.Passed() {
		t.Error("Passed should be false (hash failure is still a failure)")
	}
}

// TestRunFromBytes_HashWithEntropyProof_SkipsNotFails inverts the
// previous expectation. Appendix E.7 scopes the comparison to t=20 and
// forbids failing it for any other subject: an entropy payload carries
// no s.d.hash, so failing would report a sound proof as forged purely
// because the caller passed an argument that does not apply. The skip is
// required rather than silence so an ignored argument stays visible.
func TestRunFromBytes_HashWithEntropyProof_SkipsNotFails(t *testing.T) {
	minimalEntropy := `{
		"v": 1,
		"t": 30,
		"pk": "CTwMqDZnPd/QTLSq8aTeSD3a+j2DQxKcGfhhIYJQ65Y=",
		"sig": "AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA==",
		"ts": "2026-04-06T23:25:06Z",
		"s": {"id": "019cf813-99b8-730a-84f1-5a711a9c355e", "d": {}, "mh": "ccddccddccddccddccddccddccddccddccddccddccddccddccddccddccddccdd", "kid": "4ceefa4a"},
		"b": {"id": "019cf813-99b8-730a-84f1-5a711a9c355e", "ph": "1111111111111111111111111111111111111111111111111111111111111111", "mr": "2222222222222222222222222222222222222222222222222222222222222222", "mh": "4444444444444444444444444444444444444444444444444444444444444444", "kid": "4ceefa4a"},
		"ip": "AA",
		"cx": [{"t": 40, "net": "testnet", "tx": "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", "memo": "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb", "l": 1, "ep": "AA"}]
	}`
	report, err := RunFromBytes([]byte(minimalEntropy), "(test)", Options{
		SkipExternal:   true,
		SkipSignatures: true,
		ExpectedHash:   "deadbeef",
	})
	if err != nil {
		t.Fatalf("expected no error (failures should be in report), got: %s", err)
	}
	if report == nil {
		t.Fatal("expected a report, got nil")
	}
	step := onlyStep(t, report, groupHashComparison)
	if step.Status != StatusSkip {
		t.Errorf("status: got %v, want skip (%s)", step.Status, step.Message)
	}
	if step.Category != CatDataIntegrity {
		t.Errorf("category: got %q, want %q", step.Category, CatDataIntegrity)
	}
	if !strings.Contains(step.Message, "not applicable") {
		t.Errorf("message should say the flag does not apply, got: %s", step.Message)
	}
	// E.22 requires "an expected hash was supplied" to be readable
	// separately from "it matched". The caller DID supply one, so the
	// report records it — dropping it made the published surface
	// byte-identical to a run with no --hash at all, which is the one
	// case this skip exists to distinguish.
	if report.HashProvided != "deadbeef" {
		t.Errorf("HashProvided = %q, want the caller's argument recorded on the skipped branch", report.HashProvided)
	}
	if report.HashMatched() {
		t.Error("HashMatched must be false when the comparison did not run")
	}
	// Recording it must not upgrade the verdict: nothing was compared.
	if v := report.Verdict(); v == VerdictFullyVerified {
		t.Error("a skipped comparison must not read as fully verified")
	}
}

// TestRunFromBytes_HashWithBlockProof_DoesNotFailASoundProof is the
// end-to-end consequence of the same E.7 rule, on a fixture that
// verifies cleanly offline: passing an inapplicable --hash must leave
// the verdict untouched. Before this it flipped a sound proof to exit 1
// while the terminal still printed "VERIFIED".
func TestRunFromBytes_HashWithBlockProof_DoesNotFailASoundProof(t *testing.T) {
	data, err := os.ReadFile("testdata/fixtures/block.json")
	if err != nil {
		t.Fatalf("reading block fixture: %s", err)
	}
	opts := Options{SkipExternal: true, SkipSignatures: true}
	clean, err := RunFromBytes(data, "(test)", opts)
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	if !clean.Passed() {
		t.Skip("block fixture does not verify offline; the comparison below would prove nothing")
	}

	opts.ExpectedHash = "b47cc0f104b62d4c7c30bcd68fd8e67613e287dc4ad8c310ef10cbadea9c4380"
	withHash, err := RunFromBytes(data, "(test)", opts)
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	if !withHash.Passed() {
		for _, s := range withHash.Steps {
			if s.Status == StatusFail {
				t.Errorf("FAIL: [%s] %s", s.Group, s.Message)
			}
		}
		t.Fatal("an inapplicable --hash must not fail a sound proof")
	}
	if withHash.Verdict() != VerdictVerified {
		t.Errorf("verdict: got %v, want VerdictVerified", withHash.Verdict())
	}
}

// TestRunFromBytes_NoHashNonItem_EmitsNoHashComparisonRow is the other
// half of E.7's scoping: without an expected hash there is no argument
// to acknowledge, so no row is emitted at all.
func TestRunFromBytes_NoHashNonItem_EmitsNoHashComparisonRow(t *testing.T) {
	data, err := os.ReadFile("testdata/fixtures/block.json")
	if err != nil {
		t.Fatalf("reading block fixture: %s", err)
	}
	report, err := RunFromBytes(data, "(test)", Options{SkipExternal: true, SkipSignatures: true})
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	if got := stepsIn(report, groupHashComparison); len(got) != 0 {
		t.Errorf("expected no Hash Comparison rows without --hash, got %+v", got)
	}
}

// TestRunFromBytes_BlockLike_EmitsInclusionProofSkip covers F19: E.13
// requires block-like subjects to report this step as skipped ("not
// applicable"), never omitted, so a reader can tell a check that did not
// apply from one that was never reached.
func TestRunFromBytes_BlockLike_EmitsInclusionProofSkip(t *testing.T) {
	data, err := os.ReadFile("testdata/fixtures/block.json")
	if err != nil {
		t.Fatalf("reading block fixture: %s", err)
	}
	report, err := RunFromBytes(data, "(test)", Options{SkipExternal: true, SkipSignatures: true})
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	step := onlyStep(t, report, groupInclusion)
	if step.Status != StatusSkip {
		t.Errorf("status: got %v, want skip (%s)", step.Status, step.Message)
	}
	if step.Category != CatCryptographic {
		t.Errorf("category: got %q, want %q", step.Category, CatCryptographic)
	}
	if !strings.Contains(step.Message, "not applicable") {
		t.Errorf("message should say the step does not apply, got: %s", step.Message)
	}
}

// TestRunFromBytes_Item_InclusionProofIsGraded guards the other arm: an
// item subject must still have its inclusion proof actually walked.
func TestRunFromBytes_Item_InclusionProofIsGraded(t *testing.T) {
	report, err := Run("testdata/proof_item.json", Options{SkipExternal: true})
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	step := onlyStep(t, report, groupInclusion)
	if step.Status != StatusPass {
		t.Errorf("status: got %v, want pass (%s)", step.Status, step.Message)
	}
}

func TestPresent_HashMatch(t *testing.T) {
	r := &Report{
		Filename:     "test.json",
		FileSize:     100,
		HashProvided: "aabb",
		Claims: Claims{
			Name: "test", Hash: "aabb", HashType: "sha256",
			Timestamp: "2025-01-01T00:00:00Z", TimestampStatus: TimestampOK,
		},
		Steps: []Step{
			{Group: "Hash Comparison", Status: StatusPass, Message: "ok"},
			{Group: "Test", Status: StatusPass, Message: "ok"},
		},
	}
	Present(r)
}

func TestPresent_HashMismatch(t *testing.T) {
	r := &Report{
		Filename:     "test.json",
		FileSize:     100,
		HashProvided: "deadbeef",
		Claims: Claims{
			Name: "test", Hash: "aabb", HashType: "sha256",
		},
		Steps: []Step{
			{Group: "Hash Comparison", Status: StatusFail, Message: "mismatch"},
			{Group: "Test", Status: StatusPass, Message: "ok"},
		},
	}
	Present(r)
}

func TestPresent_NoHash_ShowsGuidance(t *testing.T) {
	r := &Report{
		Filename: "test.json",
		FileSize: 100,
		Claims: Claims{
			Name: "test", Hash: "aabb", HashType: "sha256",
		},
		Steps: []Step{
			{Group: "Test", Status: StatusPass, Message: "ok"},
		},
	}
	Present(r)
}

func TestPresent_FullyVerified(t *testing.T) {
	r := &Report{
		Filename:     "test.json",
		FileSize:     100,
		HashProvided: "aabb",
		Claims: Claims{
			Name: "test", Hash: "aabb", HashType: "sha256",
		},
		Steps: []Step{
			{Group: "Hash Comparison", Status: StatusPass, Message: "ok"},
			{Group: "Test", Status: StatusPass, Message: "ok"},
		},
	}
	Present(r)
}

func TestPresent_VerifiedSkippedExternal(t *testing.T) {
	r := &Report{
		Filename:        "test.json",
		FileSize:        100,
		HashProvided:    "aabb",
		SkippedExternal: true,
		Claims: Claims{
			Name: "test", Hash: "aabb", HashType: "sha256",
		},
		Steps: []Step{
			{Group: "Hash Comparison", Status: StatusPass, Message: "ok"},
			{Group: "Test", Status: StatusPass, Message: "ok"},
		},
	}
	Present(r)
}

func TestPresent_CryptoFailWithHash(t *testing.T) {
	r := &Report{
		Filename:     "test.json",
		FileSize:     100,
		HashProvided: "aabb",
		Claims: Claims{
			Name: "test", Hash: "aabb", HashType: "sha256",
		},
		Steps: []Step{
			{Group: "Hash Comparison", Status: StatusPass, Message: "ok"},
			{Group: "Test", Status: StatusFail, Message: "bad"},
		},
	}
	Present(r)
}

func TestPresent_CryptoFailWithHashMismatch(t *testing.T) {
	r := &Report{
		Filename:     "test.json",
		FileSize:     100,
		HashProvided: "deadbeef",
		Claims: Claims{
			Name: "test", Hash: "aabb", HashType: "sha256",
		},
		Steps: []Step{
			{Group: "Hash Comparison", Status: StatusFail, Message: "mismatch"},
			{Group: "Test", Status: StatusFail, Message: "bad"},
		},
	}
	Present(r)
}

func TestPresent_ClaimsWithAllFields(t *testing.T) {
	r := &Report{
		Filename: "test.json",
		FileSize: 100,
		Claims: Claims{
			Name:            "Test Item",
			Description:     "A description",
			Hash:            "aabbccdd",
			HashType:        "sha256",
			Timestamp:       "2025-01-01T00:00:00Z",
			URL:             "https://example.com",
			Location:        &LatLong{Latitude: 39.04, Longitude: -77.48},
			HasMetadata:     true,
			TimestampStatus: TimestampOK,
		},
		Steps: []Step{
			{Group: "Test", Status: StatusPass, Message: "ok"},
		},
	}
	Present(r)
}

func TestPresent_ClaimsTimestampFuture(t *testing.T) {
	r := &Report{
		Filename: "test.json",
		FileSize: 100,
		Claims: Claims{
			Name:            "test",
			Hash:            "aabb",
			Timestamp:       "2099-01-01T00:00:00Z",
			TimestampStatus: TimestampFuture,
			TimestampNote:   "not before submission time (future-dated claim)",
		},
		Steps: []Step{
			{Group: "Test", Status: StatusPass, Message: "ok"},
		},
	}
	Present(r)
}

func TestPresent_ClaimsTimestampStale(t *testing.T) {
	r := &Report{
		Filename: "test.json",
		FileSize: 100,
		Claims: Claims{
			Name:            "test",
			Hash:            "aabb",
			Timestamp:       "2020-01-01T00:00:00Z",
			TimestampStatus: TimestampStale,
			TimestampNote:   "100 days before submission (stale claim)",
		},
		Steps: []Step{
			{Group: "Test", Status: StatusPass, Message: "ok"},
		},
	}
	Present(r)
}

func TestParseClaims_WithMetadata(t *testing.T) {
	raw := []byte(`{"name":"test","hash":"aabb","hash_type":"sha256","metadata":{"key":"val"}}`)
	c := parseClaims(raw)
	if c.Name != "test" {
		t.Errorf("Name: got %q", c.Name)
	}
	if !c.HasMetadata {
		t.Error("expected HasMetadata to be true")
	}
}

func TestParseClaims_WithoutMetadata(t *testing.T) {
	raw := []byte(`{"name":"test","hash":"aabb","hash_type":"sha256"}`)
	c := parseClaims(raw)
	if c.HasMetadata {
		t.Error("expected HasMetadata to be false")
	}
}

func TestParseClaims_WithLocation(t *testing.T) {
	raw := []byte(`{"name":"test","hash":"aa","hash_type":"sha256","location":{"latitude":39.04,"longitude":-77.48}}`)
	c := parseClaims(raw)
	if c.Location == nil {
		t.Fatal("expected location to be parsed")
	}
	if c.Location.Latitude != 39.04 {
		t.Errorf("Latitude: got %f", c.Location.Latitude)
	}
}

func TestParseClaims_NullClaims(t *testing.T) {
	c := parseClaims([]byte("null"))
	if c.Name != "" {
		t.Error("expected empty claims for null")
	}
}

func TestParseClaims_EmptyClaims(t *testing.T) {
	c := parseClaims(nil)
	if c.Name != "" {
		t.Error("expected empty claims for nil")
	}
}

func TestTruncateToSecond(t *testing.T) {
	got := truncateToSecond("2025-01-01T12:30:45.123456Z")
	want := "2025-01-01T12:30:45Z"
	if got != want {
		t.Errorf("truncateToSecond: got %q, want %q", got, want)
	}
}

func TestTruncateToSecond_AlreadyTruncated(t *testing.T) {
	got := truncateToSecond("2025-01-01T12:30:45Z")
	if got != "2025-01-01T12:30:45Z" {
		t.Errorf("truncateToSecond: got %q", got)
	}
}

func TestTruncateToSecond_Invalid(t *testing.T) {
	got := truncateToSecond("not a timestamp")
	if got != "not a timestamp" {
		t.Errorf("truncateToSecond should return input for invalid timestamps, got %q", got)
	}
}

// =====================================================================
// Entropy proof tests
// =====================================================================

// Minimal valid entropy proof JSON for testing.
const minimalEntropyProofJSON = `{
  "v": 1,
  "t": 30,
  "pk": "CTwMqDZnPd/QTLSq8aTeSD3a+j2DQxKcGfhhIYJQ65Y=",
  "sig": "AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA==",
  "ts": "2026-03-27T13:48:35Z",
  "s": {
    "id": "019d2ae3-865c-7651-9923-b14c55bc8e33",
    "d": {"pulse": {"outputValue": "ABC123"}},
    "mh": "5555555555555555555555555555555555555555555555555555555555555555",
    "kid": "4ceefa4a"
  },
  "b": {
    "id": "019cf813-99b8-730a-84f1-5a711a9c355e",
    "ph": "1111111111111111111111111111111111111111111111111111111111111111",
    "mr": "2222222222222222222222222222222222222222222222222222222222222222",
    "mh": "4444444444444444444444444444444444444444444444444444444444444444",
    "kid": "4ceefa4a"
  },
  "ip": "AA",
  "cx": [{"t": 40, "net": "testnet", "tx": "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", "memo": "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb", "l": 1, "ep": "AA"}]
}`

func TestRun_MinimalEntropyProof_ParsesCorrectly(t *testing.T) {
	path := writeTemp(t, minimalEntropyProofJSON)
	report, err := Run(path, Options{SkipExternal: true})
	if err != nil {
		t.Fatalf("Run should not return error for structurally valid entropy proof: %s", err)
	}
	if report == nil {
		t.Fatal("expected non-nil report")
	}
	if report.SubjectType != "entropy_nist" {
		t.Errorf("SubjectType: got %q, want entropy_nist", report.SubjectType)
	}
	if report.SubjectID != "019d2ae3-865c-7651-9923-b14c55bc8e33" {
		t.Errorf("SubjectID: got %q", report.SubjectID)
	}
	if report.Source != "entropy_nist" {
		t.Errorf("Source: got %q, want entropy_nist", report.Source)
	}
	if report.GeneratedAt != "2026-03-27T13:48:35Z" {
		t.Errorf("GeneratedAt: got %q", report.GeneratedAt)
	}
}

func TestRun_MinimalEntropyProof_RunsWithoutPanic(t *testing.T) {
	path := writeTemp(t, minimalEntropyProofJSON)
	report, err := Run(path, Options{SkipExternal: true, SkipSignatures: true})
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	// Crypto will fail (fake hashes), but should have steps
	if len(report.Steps) == 0 {
		t.Error("expected steps in report")
	}
	// Should have Subject Data steps (0x21 entropy hash + 0x23 composite)
	hasSubjectDataStep := false
	for _, s := range report.Steps {
		if s.Group == "Subject Data" {
			hasSubjectDataStep = true
			break
		}
	}
	if !hasSubjectDataStep {
		t.Error("expected Subject Data group in report steps")
	}
}

func TestRun_RealItemProof(t *testing.T) {
	report, err := Run("testdata/proof_item.json", Options{SkipExternal: true})
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	if !report.Passed() {
		for _, s := range report.Steps {
			if s.Status == StatusFail {
				t.Errorf("FAIL: [%s] %s", s.Group, s.Message)
			}
		}
		t.Fatal("real item proof should pass all checks with --skip-external")
	}
	if report.SubjectType != "item" {
		t.Errorf("SubjectType: got %q, want item", report.SubjectType)
	}
}

// TestRun_RealItemProof_ClaimsOnly exercises a real item proof generated
// in claims-as-source-of-truth mode (no s.d.hash / s.d.hash_type). The
// fixture was generated against a live test server and committed to a
// block; this is the empirical regression that confirms the verify
// pipeline is content-agnostic.
//
// Expectations:
//   - Report passes cleanly (no fails, no warnings about claims hash
//     validation, no warning about "Claims hash not verified" since
//     there is no hash to compare against).
//   - The Claims struct on the report has empty Hash and HashType
//     fields, but a non-empty Name and Description.
func TestRun_RealItemProof_ClaimsOnly(t *testing.T) {
	report, err := Run("testdata/proof_item_claims_only.json", Options{SkipExternal: true})
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	if !report.Passed() {
		for _, s := range report.Steps {
			if s.Status == StatusFail {
				t.Errorf("FAIL: [%s] %s", s.Group, s.Message)
			}
		}
		t.Fatal("claims-only item proof should pass all checks")
	}
	if report.SubjectType != "item" {
		t.Errorf("SubjectType: got %q, want item", report.SubjectType)
	}
	if report.Claims.Hash != "" {
		t.Errorf("Claims.Hash should be empty for claims-only proof, got %q", report.Claims.Hash)
	}
	if report.Claims.HashType != "" {
		t.Errorf("Claims.HashType should be empty for claims-only proof, got %q", report.Claims.HashType)
	}
	if report.Claims.Name == "" {
		t.Error("Claims.Name should be populated for claims-only proof")
	}
	if report.Claims.Description == "" {
		t.Error("Claims.Description should be populated for claims-only proof (it IS the data)")
	}

	// The old "Claims hash not verified" warning never fires for
	// claims-only proofs — it was external-hash-specific and now
	// lives in groupVerificationNotes anyway.
	for _, s := range report.Steps {
		if containsStr(s.Message, "Claims hash not verified") {
			t.Errorf("claims-only proof should not emit legacy 'Claims hash not verified' warning, got: %s", s.Message)
		}
	}

	// Claims-only proofs SHOULD emit an info-level note explaining
	// the mode (so the absence of a Hash row in the Item Claims
	// section is acknowledged rather than left silent).
	hasModeNote := false
	for _, s := range report.Steps {
		if s.Group == "Verification Notes" && s.Status == StatusInfo &&
			containsStr(s.Message, "Claims-only item") {
			hasModeNote = true
			break
		}
	}
	if !hasModeNote {
		t.Error("claims-only proof should emit an info note under Verification Notes naming the mode")
	}
}

// TestRun_ExternalHash_NoHashFlag_WarnsUnderHashComparison pins the E.7
// re-grouping. The warn used to be filed under the CLI's own
// "Verification Notes" group, which Appendix E.22's group table does not
// define — so a consumer keying on "Hash Comparison", the group E.7
// names, found nothing on the no-hash path. E.7 requires the warn and
// requires the distinction stay visible; it must not fail the proof.
func TestRun_ExternalHash_NoHashFlag_WarnsUnderHashComparison(t *testing.T) {
	report, err := Run("testdata/proof_item.json", Options{SkipExternal: true})
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	if !report.Passed() {
		t.Fatal("external-hash proof should pass with --skip-external")
	}

	note := onlyStep(t, report, groupHashComparison)
	if note.Status != StatusWarn {
		t.Fatalf("status: got %v, want warn (%s)", note.Status, note.Message)
	}
	if note.Category != CatDataIntegrity {
		t.Errorf("category: got %q, want %q", note.Category, CatDataIntegrity)
	}
	if !containsStr(note.Message, "File hash not verified") || !containsStr(note.Message, "--hash") {
		t.Errorf("note should carry E.7's phrase and name the flag, got: %s", note.Message)
	}
	// Re-grouped, never duplicated: the old home must be empty of it.
	for _, s := range stepsIn(report, groupVerificationNotes) {
		if containsStr(s.Message, "hash") {
			t.Errorf("hash note left behind under Verification Notes: %s", s.Message)
		}
	}
	// E.7 keeps "provided" and "matched" distinct facts; neither is true
	// here and neither may be inferred from the warn.
	if report.HashProvided != "" || report.HashMatched() {
		t.Errorf("no --hash was supplied: HashProvided=%q HashMatched=%v", report.HashProvided, report.HashMatched())
	}
}

// TestRun_ExternalHash_WithHashFlag_NoWorkflowNote confirms supplying
// --hash drops the workflow note entirely — the pass/fail under
// Hash Comparison group covers it.
func TestRun_ExternalHash_WithHashFlag_NoWorkflowNote(t *testing.T) {
	data, err := os.ReadFile("testdata/proof_item.json")
	if err != nil {
		t.Fatalf("reading test proof: %s", err)
	}
	bundle, err := proof.ParseBytes(data)
	if err != nil {
		t.Fatalf("parsing proof: %s", err)
	}
	claims := parseClaims(bundle.RawData)
	report, err := Run("testdata/proof_item.json", Options{
		SkipExternal: true,
		ExpectedHash: claims.Hash,
	})
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	for _, s := range report.Steps {
		if s.Group == "Verification Notes" {
			t.Errorf("--hash supplied: no Verification Notes step expected, got: [%v] %s",
				s.Status, s.Message)
		}
	}
}

// TestPresent_ExternalHashWarn_IsVisible asserts the user-visible half
// of E.7: the warn reaches the rendered report and the run still reads
// as verified. It is rendered under "Issues" now rather than the CLI's
// Verification Notes section, which is the consequence of filing it
// under E.22's own Hash Comparison group — a warn cannot fail a proof
// (E.22), so the verdict line is unaffected.
func TestPresent_ExternalHashWarn_IsVisible(t *testing.T) {
	report, err := Run("testdata/proof_item.json", Options{SkipExternal: true})
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	captured := captureLipglossOutput(t, func() { Present(report) })

	if !strings.Contains(captured, "File hash not verified") {
		t.Errorf("expected E.7's warn wording in rendered output:\n%s", captured)
	}
	if !strings.Contains(captured, "VERIFIED") {
		t.Errorf("a warn must not change the verdict line:\n%s", captured)
	}
}

// TestPresent_ClaimsOnly_VerificationNotes asserts a claims-only proof
// renders the info note in "Verification Notes".
func TestPresent_ClaimsOnly_VerificationNotes(t *testing.T) {
	report, err := Run("testdata/proof_item_claims_only.json", Options{SkipExternal: true})
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	captured := captureLipglossOutput(t, func() { Present(report) })
	if !strings.Contains(captured, "Verification Notes") {
		t.Errorf("expected 'Verification Notes' section header for claims-only proof:\n%s", captured)
	}
	if !strings.Contains(captured, "Claims-only item") {
		t.Errorf("expected claims-only mode acknowledgment in rendered output:\n%s", captured)
	}
}

// TestBuildJSONOutput_ExternalHashWarn_IsMachineVisible is the --json
// half of the E.7 re-grouping: a consumer filtering steps[] on the group
// the appendix names must find the warn, and it must also reach the
// issues array (it no longer belongs to the group that array excludes).
func TestBuildJSONOutput_ExternalHashWarn_IsMachineVisible(t *testing.T) {
	report, err := Run("testdata/proof_item.json", Options{SkipExternal: true})
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	out := BuildJSONOutput(report)

	found := false
	for _, s := range out.Steps {
		if s.Group == groupHashComparison && s.Status == StatusWarn {
			found = true
		}
	}
	if !found {
		t.Errorf("no Hash Comparison warn in steps[]: %+v", out.Steps)
	}
	if out.HashComparison.Supplied || out.HashComparison.Matched {
		t.Errorf("E.7 keeps supplied/matched distinct and both false here: %+v", out.HashComparison)
	}
	inIssues := false
	for _, iss := range out.Issues {
		if strings.Contains(iss.Message, "File hash not verified") {
			inIssues = true
		}
	}
	if !inIssues {
		t.Errorf("the warn should surface in issues[] now that it is a Hash Comparison row: %+v", out.Issues)
	}
}

// TestBuildJSONOutput_VerificationNotes_ClaimsOnly confirms the JSON
// output carries an info-level note for claims-only proofs.
func TestBuildJSONOutput_VerificationNotes_ClaimsOnly(t *testing.T) {
	report, err := Run("testdata/proof_item_claims_only.json", Options{SkipExternal: true})
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	out := BuildJSONOutput(report)
	if len(out.VerificationNotes) == 0 {
		t.Fatal("expected at least one verification_notes entry for claims-only")
	}
	if out.VerificationNotes[0].Severity != "info" {
		t.Errorf("severity: got %q, want info", out.VerificationNotes[0].Severity)
	}
	if !strings.Contains(out.VerificationNotes[0].Message, "Claims-only item") {
		t.Errorf("message: got %q", out.VerificationNotes[0].Message)
	}
}

// TestBuildJSONOutput_EntropyTemporalKeys pins the wire-key cutover in the
// --json surface: the timeline's Truestamp record-time is emitted under
// inserted_at (the server's renamed key), while the entropy subject's
// upstream source-capture time stays captured_at — a distinct value the
// server never renamed. The two timestamps differ so the assertions can tell
// the keys apart.
func TestBuildJSONOutput_EntropyTemporalKeys(t *testing.T) {
	report := &Report{
		SubjectType: "entropy_nist",
		SubjectID:   "019d2ae3-865c-7651-9923-b14c55bc8e33",
		Temporal: TemporalSummary{
			CapturedAt:  "2026-03-26T16:02:03Z", // Truestamp record time -> inserted_at
			CommittedAt: "2026-03-26T16:05:00Z",
		},
		EntropySubject: EntropySubject{
			RawSource:  "entropy_nist",
			Source:     "NIST Beacon",
			CapturedAt: "2026-03-26T16:02:00Z", // upstream source time -> captured_at
			PulseIndex: 100,
		},
		ChainLength: 1,
	}

	data, err := json.Marshal(BuildJSONOutput(report))
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	got := string(data)

	// Timeline record-time cuts over to inserted_at, never the old captured_at.
	if !strings.Contains(got, `"inserted_at":"2026-03-26T16:02:03Z"`) {
		t.Errorf("timeline record-time should emit inserted_at, got:\n%s", got)
	}
	if strings.Contains(got, `"captured_at":"2026-03-26T16:02:03Z"`) {
		t.Errorf("timeline record-time must not use the old captured_at key, got:\n%s", got)
	}
	// The entropy subject's upstream source-capture time stays captured_at.
	if !strings.Contains(got, `"captured_at":"2026-03-26T16:02:00Z"`) {
		t.Errorf("entropy subject source time should keep captured_at, got:\n%s", got)
	}
}

// TestRunFromBytes_ClaimsOnly_CBOR exercises the CBOR variant of the
// claims-only fixture, confirming the parser handles the absence of
// hash / hash_type byte-string fields cleanly.
func TestRunFromBytes_ClaimsOnly_CBOR(t *testing.T) {
	data, err := os.ReadFile("testdata/proof_item_claims_only.cbor")
	if err != nil {
		t.Fatalf("reading CBOR fixture: %s", err)
	}
	report, err := RunFromBytes(data, "claims-only.cbor", Options{SkipExternal: true})
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	if !report.Passed() {
		for _, s := range report.Steps {
			if s.Status == StatusFail {
				t.Errorf("FAIL: [%s] %s", s.Group, s.Message)
			}
		}
		t.Fatal("CBOR claims-only proof should pass all checks")
	}
	if report.Claims.Hash != "" {
		t.Errorf("Claims.Hash should be empty in CBOR claims-only proof, got %q", report.Claims.Hash)
	}
}

// TestPresent_ClaimsOnly_OmitsHashRow walks the presenter against the
// claims-only fixture and confirms the rendered report does NOT include
// a "Hash" row — the existing `if r.Claims.Hash != ""` guard at
// presenter.go remains correct after the wire-shape change.
func TestPresent_ClaimsOnly_OmitsHashRow(t *testing.T) {
	report, err := Run("testdata/proof_item_claims_only.json", Options{SkipExternal: true})
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	// Render to string by intercepting lipgloss output. This catches
	// the case where a future presenter change might start emitting a
	// "Hash:" row unconditionally.
	captured := captureLipglossOutput(t, func() { Present(report) })
	if strings.Contains(captured, "\nHash  ") || strings.Contains(captured, "Hash:") {
		t.Errorf("claims-only render should not contain a Hash row, got:\n%s", captured)
	}
	// Sanity: it should still render the Name and Description rows.
	if !strings.Contains(captured, "Name") {
		t.Error("expected Name in rendered output")
	}
	if !strings.Contains(captured, "Description") {
		t.Error("expected Description in rendered output")
	}
}

// captureLipglossOutput temporarily redirects lipgloss.Writer to a
// buffer, runs fn, and returns what was written. The verify presenter
// emits everything via lipgloss.Println which writes to the package's
// internal Writer (a colorprofile.Writer wrapping os.Stdout captured
// at init time), so redirecting os.Stdout has no effect — we have to
// swap the lipgloss.Writer target itself.
func captureLipglossOutput(t *testing.T, fn func()) string {
	t.Helper()
	var buf strings.Builder
	orig := lipgloss.Writer
	lipgloss.Writer = colorprofile.NewWriter(&buf, os.Environ())
	defer func() { lipgloss.Writer = orig }()
	fn()
	return buf.String()
}

func TestPresent_EntropyReport(t *testing.T) {
	r := &Report{
		Filename:    "entropy.json",
		FileSize:    500,
		SubjectID:   "019d2ae3-865c-7651-9923-b14c55bc8e33",
		SubjectType: "entropy_nist",
		Source:      "entropy_nist",
		GeneratedAt: "2026-03-27T13:48:35Z",
		Steps: []Step{
			{Group: "Subject Data", Status: StatusPass, Message: "Entropy hash derived (0x21)"},
			{Group: "Test", Status: StatusPass, Message: "ok"},
		},
		Temporal: TemporalSummary{
			CapturedAt:  "2026-03-27T13:48:00Z",
			CommittedAt: "2026-03-27T13:49:00Z",
		},
	}
	// Should not panic
	Present(r)
}

// --- P2b: external confirmation grading (E.18, E.19, E.21) ---

// btcSample returns the regtest commitment used across the Bitcoin tests.
// Every offline field is present and internally consistent, so a test can
// delete exactly one and observe the absence branch in isolation.
func btcSample() proof.ExternalCommit {
	return proof.ExternalCommit{
		Type:            ptype.CommitmentBitcoin,
		Network:         "regtest",
		TransactionHash: "29c38e578c10ff89e1a0392d97b3f4fd4c83c439a5ad2977f294182724718752",
		OpReturn:        "16c095556481e3f5f5c410cd2d5cad50d17f09a5ae16ac2deda88d64db5c424b",
		RawTxHex:        "02000000000101cf12c8f31bb02b316407637338df0587e54b193b48a8e823c95f7d6cb80049780000000000fdffffff010000000000000000226a2016c095556481e3f5f5c410cd2d5cad50d17f09a5ae16ac2deda88d64db5c424b01403c1e5d4c2edf2f24e609bfcdbc040648df7cd8d1d6af3f8e9b16e01b87ebe173b9a047848b7370f8622f6d43a3947bf99f5e9a6f706e48595bdce662195295eb00000000",
		TxoutproofHex:   "0000002026cf000a60782b1b57c018b6482e6837c1165d5e2aa06f92d199163481b6c07b150edc6623b16f07e3239de42b6f67d08f5d71b4d06e35685dcaab533f951d5e395ab869ffff7f200000000002000000028205b7caabd985090d0ba024a7d2ce998e53d49122989f8eac68b6f25558bbee52877124271894f27729ada539c4834cfdf4b3972d39a0e189ff108c578ec3290105",
		BlockMerkleRoot: "5e1d953f53abca5d68356ed0b4715d8fd0676f2be49d23e3076fb12366dc0e15",
		BlockHeight:     9312,
		EpochProof:      "AA",
	}
}

// stepMessages joins every step message so a test can assert on the whole
// report without caring about row order.
func stepMessages(r *Report) string {
	var b strings.Builder
	for _, s := range r.Steps {
		b.WriteString(s.Message)
		b.WriteString("\n")
	}
	return b.String()
}

func countStatus(r *Report, want Status) int {
	n := 0
	for _, s := range r.Steps {
		if s.Status == want {
			n++
		}
	}
	return n
}

func onlyCommitment(t *testing.T, r *Report) CommitmentInfo {
	t.Helper()
	if len(r.CommitmentInfos) != 1 {
		t.Fatalf("expected exactly 1 CommitmentInfo, got %d", len(r.CommitmentInfos))
	}
	return r.CommitmentInfos[0]
}

// --- F2: E.19(c) absence branches ---

// The Appendix D bundle's t=41 entry carries no rtx, txp or bmr. E.19(c)
// says such an entry has no offline evidence and its commitment is
// reported skip; D.4 pins that as a single Bitcoin Commitment row.
func TestVerifySingleBitcoin_NoOfflineEvidence_SingleSkipRow(t *testing.T) {
	cx := btcSample()
	cx.RawTxHex = ""
	cx.TxoutproofHex = ""
	cx.BlockMerkleRoot = ""

	r := &Report{}
	verifySingleBitcoin(r, &cx, Options{})

	if len(r.Steps) != 1 {
		t.Fatalf("expected exactly 1 step, got %d:\n%s", len(r.Steps), stepMessages(r))
	}
	if r.Steps[0].Status != StatusSkip {
		t.Errorf("status = %v, want skip: %s", r.Steps[0].Status, r.Steps[0].Message)
	}
	if r.Steps[0].Group != groupBitcoin || r.Steps[0].Category != CatBlockchain {
		t.Errorf("group/category = %s/%s, want %s/%s",
			r.Steps[0].Group, r.Steps[0].Category, groupBitcoin, CatBlockchain)
	}
	if ci := onlyCommitment(t, r); ci.ExternalCheck != ExternalSkipped {
		t.Errorf("ExternalCheck = %v, want ExternalSkipped", ci.ExternalCheck)
	}
}

// An absent txp used to hit a bare `return` that erased the binding row
// AND the whole CommitmentInfo from the report.
func TestVerifySingleBitcoin_NoTxOutProof_SkipsAndKeepsCommitment(t *testing.T) {
	cx := btcSample()
	cx.TxoutproofHex = ""

	r := &Report{}
	verifySingleBitcoin(r, &cx, Options{SkipExternal: true})

	if n := countStatus(r, StatusFail); n != 0 {
		t.Errorf("absent txp produced %d failures:\n%s", n, stepMessages(r))
	}
	if !strings.Contains(stepMessages(r), "no txoutproof") {
		t.Errorf("expected a txoutproof skip row:\n%s", stepMessages(r))
	}
	// The commitment must survive so --json still reports the chain.
	ci := onlyCommitment(t, r)
	if ci.BlockHash != "" {
		t.Errorf("BlockHash = %q, want empty with no header to recompute from", ci.BlockHash)
	}
}

func TestVerifySingleBitcoin_NoRawTx_SkipsOpReturnAndTxid(t *testing.T) {
	cx := btcSample()
	cx.RawTxHex = ""

	r := &Report{}
	verifySingleBitcoin(r, &cx, Options{SkipExternal: true})

	if n := countStatus(r, StatusFail); n != 0 {
		t.Errorf("absent rtx produced %d failures:\n%s", n, stepMessages(r))
	}
	msgs := stepMessages(r)
	if !strings.Contains(msgs, "no raw transaction") {
		t.Errorf("expected one skip row covering OP_RETURN and txid:\n%s", msgs)
	}
	// The txoutproof steps are independent of rtx and must still run.
	if !strings.Contains(msgs, "matched set") {
		t.Errorf("placement check should still run without rtx:\n%s", msgs)
	}
}

func TestVerifySingleBitcoin_NoBlockMerkleRoot_SkipsCrossCheck(t *testing.T) {
	cx := btcSample()
	cx.BlockMerkleRoot = ""

	r := &Report{}
	verifySingleBitcoin(r, &cx, Options{SkipExternal: true})

	if n := countStatus(r, StatusFail); n != 0 {
		t.Errorf("absent bmr produced %d failures:\n%s", n, stepMessages(r))
	}
	if !strings.Contains(stepMessages(r), "no block merkle root") {
		t.Errorf("expected a bmr skip row:\n%s", stepMessages(r))
	}
}

// A txoutproof that is present but will not parse is a bundle defect, not
// an absence, so it stays a failure — but it must not erase the rest of
// the report the way the old bare `return` did.
func TestVerifySingleBitcoin_UnparseableTxOutProof_FailsAndFallsThrough(t *testing.T) {
	cx := btcSample()
	cx.TxoutproofHex = "cccc"

	r := &Report{}
	verifySingleBitcoin(r, &cx, Options{SkipExternal: true})

	if !strings.Contains(stepMessages(r), "Txoutproof parse failed") {
		t.Errorf("expected a parse failure:\n%s", stepMessages(r))
	}
	if !strings.Contains(stepMessages(r), "External Bitcoin verification skipped") {
		t.Errorf("binding row must still be reported:\n%s", stepMessages(r))
	}
	onlyCommitment(t, r)
}

// N1: chainhash.NewHashFromStr("") returns the all-zero hash with a nil
// error, so an absent tx used to fail the matched-set test. E.19(c)
// requires skip.
func TestVerifySingleBitcoin_AbsentTx_SkipsNeverFails(t *testing.T) {
	cx := btcSample()
	cx.TransactionHash = ""

	r := &Report{}
	verifySingleBitcoin(r, &cx, Options{SkipExternal: true})

	if n := countStatus(r, StatusFail); n != 0 {
		t.Errorf("absent tx produced %d failures:\n%s", n, stepMessages(r))
	}
	msgs := stepMessages(r)
	for _, want := range []string{"Txid comparison skipped", "placement skipped"} {
		if !strings.Contains(msgs, want) {
			t.Errorf("missing %q:\n%s", want, msgs)
		}
	}
}

// --- F10: E.19(b) step 4, the bmr cross-check ---

func TestVerifySingleBitcoin_BlockMerkleRootIsCompared(t *testing.T) {
	good := btcSample()
	r := &Report{}
	verifySingleBitcoin(r, &good, Options{SkipExternal: true})
	if !strings.Contains(stepMessages(r), "Bitcoin block merkle root matches") {
		t.Errorf("a correct bmr should pass its own step:\n%s", stepMessages(r))
	}

	bad := btcSample()
	bad.BlockMerkleRoot = strings.Repeat("de", 32)
	r2 := &Report{}
	verifySingleBitcoin(r2, &bad, Options{SkipExternal: true})
	if countStatus(r2, StatusFail) == 0 {
		t.Errorf("a bmr that disagrees with the txoutproof header must fail:\n%s", stepMessages(r2))
	}
	if !strings.Contains(stepMessages(r2), "does NOT match cx.bmr") {
		t.Errorf("expected a bmr mismatch failure:\n%s", stepMessages(r2))
	}
}

// --- F4 / N2: the binding step decides ExternalCheck ---

// A regtest commitment has no public API, so no lookup runs. Reporting
// that as externally verified is the F4 lie.
func TestVerifySingleBitcoin_NoPublicAPI_IsNotConfirmed(t *testing.T) {
	cx := btcSample()

	r := &Report{}
	verifySingleBitcoin(r, &cx, Options{SkipExternal: false})

	if !strings.Contains(stepMessages(r), "no public API for regtest") {
		t.Errorf("expected the no-public-API skip:\n%s", stepMessages(r))
	}
	if ci := onlyCommitment(t, r); ci.ExternalCheck != ExternalSkipped {
		t.Errorf("ExternalCheck = %v, want ExternalSkipped for a lookup that never ran", ci.ExternalCheck)
	}
}

func TestVerifySingleBitcoin_SkipExternal_IsNotConfirmed(t *testing.T) {
	cx := btcSample()

	r := &Report{}
	verifySingleBitcoin(r, &cx, Options{SkipExternal: true})

	if ci := onlyCommitment(t, r); ci.ExternalCheck != ExternalSkipped {
		t.Errorf("ExternalCheck = %v, want ExternalSkipped under --skip-external", ci.ExternalCheck)
	}
}

// N2: E.19(b) binds the header "at height h". With no h there is nothing
// to look up, which E.19(c) reports as skip.
func TestVerifySingleBitcoin_AbsentHeight_SkipsBinding(t *testing.T) {
	cx := btcSample()
	cx.Network = "mainnet"
	cx.BlockHeight = 0

	r := &Report{}
	verifySingleBitcoin(r, &cx, Options{SkipExternal: false})

	if n := countStatus(r, StatusFail); n != 0 {
		t.Errorf("absent h produced %d failures:\n%s", n, stepMessages(r))
	}
	if !strings.Contains(stepMessages(r), "no block height") {
		t.Errorf("expected an absent-height skip:\n%s", stepMessages(r))
	}
}

// blockstreamBlockBody writes the block JSON Blockstream publishes for
// the block the request asked for. The `id` echo is not decoration: the
// E.19(b) binding lookup grades an answer that names a different block —
// or names none at all — as malformed, so a stub that omits it exercises
// the malformed path instead of the one the test is about, and a stub
// that hardcodes an id cannot drift when the fixture changes.
func blockstreamBlockBody(t *testing.T, w http.ResponseWriter, req *http.Request, height int, timestamp int64) {
	t.Helper()
	if err := json.NewEncoder(w).Encode(map[string]any{
		"id":        path.Base(req.URL.Path),
		"height":    height,
		"timestamp": timestamp,
	}); err != nil {
		t.Errorf("encoding blockstream body: %s", err)
	}
}

// blockstreamStub redirects the mainnet Blockstream base URL at a stub for
// the duration of the test.
func blockstreamStub(t *testing.T, handler http.HandlerFunc) {
	t.Helper()
	srv := httptest.NewServer(handler)
	orig := external.BlockstreamMainnetURL
	external.BlockstreamMainnetURL = srv.URL
	t.Cleanup(func() {
		external.BlockstreamMainnetURL = orig
		srv.Close()
	})
}

func TestVerifySingleBitcoin_ChainConfirmsHeader_Confirmed(t *testing.T) {
	blockstreamStub(t, func(w http.ResponseWriter, req *http.Request) {
		blockstreamBlockBody(t, w, req, 9312, 1700000000)
	})

	cx := btcSample()
	cx.Network = "mainnet"

	r := &Report{}
	verifySingleBitcoin(r, &cx, Options{})

	if n := countStatus(r, StatusFail); n != 0 {
		t.Fatalf("unexpected failures:\n%s", stepMessages(r))
	}
	ci := onlyCommitment(t, r)
	if ci.ExternalCheck != ExternalConfirmed {
		t.Errorf("ExternalCheck = %v, want ExternalConfirmed", ci.ExternalCheck)
	}
	if ci.Timestamp == "" {
		t.Error("a confirmed lookup should archive the chain's timestamp")
	}
}

// N2: the height the bundle publishes must be the height the chain
// reports, or the report archives a lie next to a genuine header.
func TestVerifySingleBitcoin_HeightDisagrees_Fails(t *testing.T) {
	blockstreamStub(t, func(w http.ResponseWriter, req *http.Request) {
		blockstreamBlockBody(t, w, req, 999999, 1700000000)
	})

	cx := btcSample()
	cx.Network = "mainnet"

	r := &Report{}
	verifySingleBitcoin(r, &cx, Options{})

	if !strings.Contains(stepMessages(r), "height mismatch") {
		t.Errorf("expected a height mismatch failure:\n%s", stepMessages(r))
	}
	if ci := onlyCommitment(t, r); ci.ExternalCheck != ExternalFailed {
		t.Errorf("ExternalCheck = %v, want ExternalFailed", ci.ExternalCheck)
	}
}

// E.22: a skipped external check MUST NOT fail a proof. An unreachable
// Blockstream leaves the commitment unconfirmed, not failed.
func TestVerifySingleBitcoin_ChainUnavailable_Skips(t *testing.T) {
	blockstreamStub(t, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	})

	cx := btcSample()
	cx.Network = "mainnet"

	r := &Report{}
	verifySingleBitcoin(r, &cx, Options{})

	if n := countStatus(r, StatusFail); n != 0 {
		t.Errorf("an unavailable Blockstream failed the proof:\n%s", stepMessages(r))
	}
	if ci := onlyCommitment(t, r); ci.ExternalCheck != ExternalSkipped {
		t.Errorf("ExternalCheck = %v, want ExternalSkipped", ci.ExternalCheck)
	}
}

func TestVerifySingleBitcoin_HeaderNotOnChain_Fails(t *testing.T) {
	blockstreamStub(t, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	})

	cx := btcSample()
	cx.Network = "mainnet"

	r := &Report{}
	verifySingleBitcoin(r, &cx, Options{})

	if !strings.Contains(stepMessages(r), "not on chain") {
		t.Errorf("expected a definitive not-on-chain failure:\n%s", stepMessages(r))
	}
	if ci := onlyCommitment(t, r); ci.ExternalCheck != ExternalFailed {
		t.Errorf("ExternalCheck = %v, want ExternalFailed", ci.ExternalCheck)
	}
}

// --- F6: E.18 Stellar grading ---

// horizonStub redirects both Horizon base URLs at a stub for the duration
// of the test, so a test controls the answer regardless of which instance
// E.18's endpoint rule selects.
func horizonStub(t *testing.T, handler http.HandlerFunc) {
	t.Helper()
	srv := httptest.NewServer(handler)
	origPublic, origTestnet := external.HorizonPublicURL, external.HorizonTestnetURL
	external.HorizonPublicURL = srv.URL
	external.HorizonTestnetURL = srv.URL
	t.Cleanup(func() {
		external.HorizonPublicURL = origPublic
		external.HorizonTestnetURL = origTestnet
		srv.Close()
	})
}

func stellarSample() proof.ExternalCommit {
	return proof.ExternalCommit{
		Type:            ptype.CommitmentStellar,
		Network:         "testnet",
		TransactionHash: strings.Repeat("ab", 32),
		MemoHash:        strings.Repeat("cd", 32),
		Ledger:          51234567,
		EpochProof:      "AA",
	}
}

// horizonTxBody renders the Horizon transaction JSON for a memo given as
// hex, since Horizon publishes the memo base64-encoded. `hash` is part of
// every real Horizon transaction resource and is not optional here: E.18's
// identity guard grades a response that names no transaction as malformed,
// so a body without it exercises the malformed path rather than the memo
// comparison these tests are about.
func horizonTxBody(t *testing.T, txHash, memoHex string, ledger int) string {
	t.Helper()
	raw, err := hex.DecodeString(memoHex)
	if err != nil {
		t.Fatalf("decoding memo hex: %s", err)
	}
	body, err := json.Marshal(map[string]any{
		"hash":       txHash,
		"memo_type":  "hash",
		"memo":       base64.StdEncoding.EncodeToString(raw),
		"ledger":     ledger,
		"created_at": "2026-04-22T20:00:02Z",
	})
	if err != nil {
		t.Fatalf("encoding horizon body: %s", err)
	}
	return string(body)
}

// F6b: an entry with no `net` used to fail before any HTTP call. E.5
// forbids the absence of an optional field from failing a sound proof;
// E.18 routes an unnamed network to the testnet instance.
func TestVerifySingleStellar_AbsentNetwork_IsLookedUpNotFailed(t *testing.T) {
	cx := stellarSample()
	cx.Network = ""
	horizonStub(t, func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(horizonTxBody(t, cx.TransactionHash, cx.MemoHash, cx.Ledger)))
	})

	r := &Report{}
	verifySingleStellar(r, &cx, Options{})

	if n := countStatus(r, StatusFail); n != 0 {
		t.Fatalf("an unnamed network failed the step:\n%s", stepMessages(r))
	}
	if !strings.Contains(stepMessages(r), "confirmed on testnet") {
		t.Errorf("expected the queried instance to be named:\n%s", stepMessages(r))
	}
	if ci := onlyCommitment(t, r); ci.ExternalCheck != ExternalConfirmed {
		t.Errorf("ExternalCheck = %v, want ExternalConfirmed", ci.ExternalCheck)
	}
}

// F6a: same bytes, unreachable Horizon. E.18 requires skip, never fail.
func TestVerifySingleStellar_Unavailable_Skips(t *testing.T) {
	horizonStub(t, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadGateway)
	})

	cx := stellarSample()
	r := &Report{}
	verifySingleStellar(r, &cx, Options{})

	if n := countStatus(r, StatusFail); n != 0 {
		t.Errorf("an unreachable Horizon failed the proof:\n%s", stepMessages(r))
	}
	if ci := onlyCommitment(t, r); ci.ExternalCheck != ExternalSkipped {
		t.Errorf("ExternalCheck = %v, want ExternalSkipped", ci.ExternalCheck)
	}
}

// A 404 on a named network is a definitive absence from that chain.
func TestVerifySingleStellar_NotFoundOnNamedNetwork_Fails(t *testing.T) {
	horizonStub(t, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	})

	cx := stellarSample()
	cx.Network = "public"
	r := &Report{}
	verifySingleStellar(r, &cx, Options{})

	if !strings.Contains(stepMessages(r), "not on chain") {
		t.Errorf("expected a not-on-chain failure:\n%s", stepMessages(r))
	}
	if ci := onlyCommitment(t, r); ci.ExternalCheck != ExternalFailed {
		t.Errorf("ExternalCheck = %v, want ExternalFailed", ci.ExternalCheck)
	}
}

// The same 404 against a network the entry never named only means the
// default instance does not hold it — the transaction may live on the
// other one, so failing would punish a missing optional field.
func TestVerifySingleStellar_NotFoundOnDefaultedNetwork_Skips(t *testing.T) {
	horizonStub(t, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	})

	cx := stellarSample()
	cx.Network = ""
	r := &Report{}
	verifySingleStellar(r, &cx, Options{})

	if n := countStatus(r, StatusFail); n != 0 {
		t.Errorf("a defaulted-network 404 failed the proof:\n%s", stepMessages(r))
	}
	if ci := onlyCommitment(t, r); ci.ExternalCheck != ExternalSkipped {
		t.Errorf("ExternalCheck = %v, want ExternalSkipped", ci.ExternalCheck)
	}
}

// A memo that disagrees is the one networked outcome E.18 fails on, and
// the F6 relaxation must not soften it.
func TestVerifySingleStellar_MemoMismatch_Fails(t *testing.T) {
	cx := stellarSample()
	horizonStub(t, func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(horizonTxBody(t, cx.TransactionHash, strings.Repeat("ef", 32), cx.Ledger)))
	})

	r := &Report{}
	verifySingleStellar(r, &cx, Options{})

	if countStatus(r, StatusFail) == 0 {
		t.Errorf("a memo mismatch must fail:\n%s", stepMessages(r))
	}
	if ci := onlyCommitment(t, r); ci.ExternalCheck != ExternalFailed {
		t.Errorf("ExternalCheck = %v, want ExternalFailed", ci.ExternalCheck)
	}
}

// E.18: an entry that carries no `tx` to look up MUST be reported skipped.
func TestVerifySingleStellar_AbsentTx_Skips(t *testing.T) {
	cx := stellarSample()
	cx.TransactionHash = ""

	r := &Report{}
	verifySingleStellar(r, &cx, Options{})

	if n := countStatus(r, StatusFail); n != 0 {
		t.Errorf("an entry with no tx failed the proof:\n%s", stepMessages(r))
	}
	if !strings.Contains(stepMessages(r), "no transaction id") {
		t.Errorf("expected a nothing-to-look-up skip:\n%s", stepMessages(r))
	}
}

func TestVerifySingleStellar_SkipExternal_IsNotConfirmed(t *testing.T) {
	cx := stellarSample()
	r := &Report{}
	verifySingleStellar(r, &cx, Options{SkipExternal: true})

	if ci := onlyCommitment(t, r); ci.ExternalCheck != ExternalSkipped {
		t.Errorf("ExternalCheck = %v, want ExternalSkipped under --skip-external", ci.ExternalCheck)
	}
}

// --- F7: E.21 entropy source availability ---

func nistStub(t *testing.T, handler http.HandlerFunc) {
	t.Helper()
	srv := httptest.NewServer(handler)
	orig := external.NISTBeaconURL
	external.NISTBeaconURL = srv.URL
	t.Cleanup(func() {
		external.NISTBeaconURL = orig
		srv.Close()
	})
}

const nistSubject = `{"chainIndex":1,"pulseIndex":2847561,"outputValue":"AABB","timeStamp":"2026-04-22T19:45:00Z"}`

// E.21 fails on a value mismatch only; unavailability is reported skipped.
func TestVerifyEntropyNIST_Availability(t *testing.T) {
	cases := []struct {
		name    string
		status  int
		want    Status
		wantSub string
	}{
		{"server error", http.StatusInternalServerError, StatusSkip, "NIST Beacon unavailable"},
		{"rate limited", http.StatusTooManyRequests, StatusSkip, "NIST Beacon unavailable"},
		{"pulse retired", http.StatusNotFound, StatusSkip, "has no pulse"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			nistStub(t, func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(tc.status)
			})
			r := &Report{}
			verifyEntropyNIST(r, json.RawMessage(nistSubject))
			if len(r.Steps) != 1 {
				t.Fatalf("expected 1 step, got %d:\n%s", len(r.Steps), stepMessages(r))
			}
			if r.Steps[0].Status != tc.want {
				t.Errorf("status = %v, want %v: %s", r.Steps[0].Status, tc.want, r.Steps[0].Message)
			}
			if !strings.Contains(r.Steps[0].Message, tc.wantSub) {
				t.Errorf("message = %q, want it to contain %q", r.Steps[0].Message, tc.wantSub)
			}
		})
	}
}

// The mismatch path is the one E.21 names as a failure, and the
// availability relaxation must not reach it.
func TestVerifyEntropyNIST_ValueMismatch_StillFails(t *testing.T) {
	nistStub(t, func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"pulse": map[string]any{
				"chainIndex": 1, "pulseIndex": 2847561,
				"outputValue": "DEADBEEF", "timeStamp": "2026-04-22T19:45:00Z",
			},
		})
	})
	r := &Report{}
	verifyEntropyNIST(r, json.RawMessage(nistSubject))
	if countStatus(r, StatusFail) == 0 {
		t.Errorf("an entropy value mismatch must fail:\n%s", stepMessages(r))
	}
}

// A bundle field the fetcher cannot use is a defect in the bundle, not an
// availability problem, and stays a failure.
func TestVerifyEntropyNIST_UnusableIndex_Fails(t *testing.T) {
	r := &Report{}
	verifyEntropyNIST(r, json.RawMessage(`{"chainIndex":-1,"pulseIndex":5,"outputValue":"AA","timeStamp":"x"}`))
	if countStatus(r, StatusFail) == 0 {
		t.Errorf("an unusable chain index must fail:\n%s", stepMessages(r))
	}
	if !strings.Contains(stepMessages(r), "unusable") {
		t.Errorf("expected an unusable-input failure:\n%s", stepMessages(r))
	}
}

func TestVerifyEntropyStellar_Availability(t *testing.T) {
	cases := []struct {
		name    string
		status  int
		wantSub string
	}{
		{"horizon down", http.StatusBadGateway, "Stellar Horizon unavailable"},
		{"testnet reset", http.StatusNotFound, "is not in the"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			horizonStub(t, func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(tc.status)
			})
			r := &Report{}
			verifyEntropyStellar(r, json.RawMessage(
				`{"hash":"aa","sequence":56789012,"closed_at":"2026-04-22T19:45:00Z"}`), "testnet")
			if len(r.Steps) != 1 || r.Steps[0].Status != StatusSkip {
				t.Fatalf("expected a single skip step, got:\n%s", stepMessages(r))
			}
			if !strings.Contains(r.Steps[0].Message, tc.wantSub) {
				t.Errorf("message = %q, want it to contain %q", r.Steps[0].Message, tc.wantSub)
			}
		})
	}
}

func TestVerifyEntropyBitcoin_Availability(t *testing.T) {
	cases := []struct {
		name    string
		status  int
		wantSub string
	}{
		{"blockstream down", http.StatusServiceUnavailable, "Blockstream unavailable"},
		{"block absent", http.StatusNotFound, "has no block"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			blockstreamStub(t, func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(tc.status)
			})
			r := &Report{}
			verifyEntropyBitcoin(r, json.RawMessage(
				`{"hash":"`+strings.Repeat("cc", 32)+`","height":870000,"time":1700000000}`), "mainnet")
			if len(r.Steps) != 1 || r.Steps[0].Status != StatusSkip {
				t.Fatalf("expected a single skip step, got:\n%s", stepMessages(r))
			}
			if !strings.Contains(r.Steps[0].Message, tc.wantSub) {
				t.Errorf("message = %q, want it to contain %q", r.Steps[0].Message, tc.wantSub)
			}
		})
	}
}

func TestVerifyEntropyBitcoin_UnusableHash_Fails(t *testing.T) {
	r := &Report{}
	verifyEntropyBitcoin(r, json.RawMessage(`{"hash":"zz","height":1,"time":2}`), "mainnet")
	if countStatus(r, StatusFail) == 0 {
		t.Errorf("an unusable block hash must fail:\n%s", stepMessages(r))
	}
}

// --- F4 remote half ---

func TestRemoteExternalCheck_OnlyConfirmsAnUnqualifiedPass(t *testing.T) {
	cases := []struct {
		name    string
		skipped bool
		steps   []Step
		want    ExternalStatus
	}{
		{"no steps at all", false, nil, ExternalSkipped},
		{"server skipped external", true,
			[]Step{{Group: groupStellar, Status: StatusPass}}, ExternalSkipped},
		{"pass only", false,
			[]Step{{Group: groupStellar, Status: StatusPass}}, ExternalConfirmed},
		{"pass then skip", false,
			[]Step{{Group: groupStellar, Status: StatusPass}, {Group: groupStellar, Status: StatusSkip}},
			ExternalSkipped},
		{"skip then pass", false,
			[]Step{{Group: groupStellar, Status: StatusSkip}, {Group: groupStellar, Status: StatusPass}},
			ExternalSkipped},
		{"any fail", false,
			[]Step{{Group: groupStellar, Status: StatusPass}, {Group: groupStellar, Status: StatusFail}},
			ExternalFailed},
		{"other group ignored", false,
			[]Step{{Group: groupBitcoin, Status: StatusFail}}, ExternalSkipped},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			r := &Report{Steps: tc.steps, SkippedExternal: tc.skipped}
			if got := remoteExternalCheck(r, groupStellar); got != tc.want {
				t.Errorf("remoteExternalCheck = %v, want %v", got, tc.want)
			}
		})
	}
}

// --- F25: E.14's five required block fields ---

// TestDeriveBlockHash_MissingField_FailsAndReturnsEmpty covers the hole
// that made this finding real: tscrypto.HexToBytes("") returns an empty
// slice and no error, so ComputeBlockHash happily digests a 121-byte
// preimage for a block with no id and the step reported "Block hash
// derived (0x32)" — a positive assertion about a derivation that never
// happened. E.14 says the preimage is 157 bytes unconditionally and all
// five fields MUST be present.
func TestDeriveBlockHash_MissingField_FailsAndReturnsEmpty(t *testing.T) {
	full := proof.Block{
		ID:                "019cf813-99b8-730a-84f1-5a711a9c355e",
		PreviousBlockHash: "0000000000000000000000000000000000000000000000000000000000000000",
		MerkleRoot:        "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		MetadataHash:      "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
		SigningKeyID:      "4ceefa4a",
	}
	cases := []struct {
		name   string
		mutate func(*proof.Block)
	}{
		{"b.id", func(b *proof.Block) { b.ID = "" }},
		{"b.ph", func(b *proof.Block) { b.PreviousBlockHash = "" }},
		{"b.mr", func(b *proof.Block) { b.MerkleRoot = "" }},
		{"b.mh", func(b *proof.Block) { b.MetadataHash = "" }},
		{"b.kid", func(b *proof.Block) { b.SigningKeyID = "" }},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			block := full
			tc.mutate(&block)
			r := &Report{}
			if hash := deriveBlockHash(r, block); hash != "" {
				t.Errorf("expected no block hash, got %q", hash)
			}
			step := onlyStep(t, r, groupBlockHash)
			if step.Status != StatusFail {
				t.Errorf("status: got %v, want fail (%s)", step.Status, step.Message)
			}
			if containsStr(step.Message, "derived") {
				t.Errorf("the fail message must not read as a derivation: %s", step.Message)
			}
		})
	}
}

// TestRunFromBytes_BlockMissingID_CascadesToCannotVerify is the same
// guard seen end to end. With no block hash, E.15 must report one
// unverifiable epoch row per cx entry and E.16 must say "cannot verify",
// never assert a signature verdict.
func TestRunFromBytes_BlockMissingID_CascadesToCannotVerify(t *testing.T) {
	var bundle map[string]any
	if err := json.Unmarshal([]byte(minimalProofJSON), &bundle); err != nil {
		t.Fatalf("decoding fixture: %s", err)
	}
	delete(bundle["b"].(map[string]any), "id")
	data, err := json.Marshal(bundle)
	if err != nil {
		t.Fatalf("re-encoding fixture: %s", err)
	}

	report, err := RunFromBytes(data, "(test)", Options{SkipExternal: true})
	if err != nil {
		t.Fatalf("a missing b.id must reach a report, not a parse error: %s", err)
	}
	blockStep := onlyStep(t, report, groupBlockHash)
	if blockStep.Status != StatusFail {
		t.Errorf("Block Hash: got %v, want fail (%s)", blockStep.Status, blockStep.Message)
	}
	sigStep := onlyStep(t, report, groupProof)
	if sigStep.Status != StatusFail || !strings.Contains(sigStep.Message, "missing derived data") {
		t.Errorf("Proof Signature should report E.16's cannot-verify, got [%v] %s", sigStep.Status, sigStep.Message)
	}
}

// TestVerifyEpochProofs_NoBlockHash_OneRowPerEntry pins E.15's "each cx
// entry produces its own step result". A single row for the whole array
// drops the per-entry accounting the appendix requires.
func TestVerifyEpochProofs_NoBlockHash_OneRowPerEntry(t *testing.T) {
	commits := []proof.ExternalCommit{
		{Type: ptype.CommitmentStellar, MemoHash: "aa", EpochProof: "AA"},
		{Type: ptype.CommitmentBitcoin, OpReturn: "bb", EpochProof: "AA"},
	}
	r := &Report{}
	if roots := verifyEpochProofs(r, commits, ""); roots != nil {
		t.Errorf("expected no epoch roots, got %v", roots)
	}
	steps := stepsIn(r, groupEpoch)
	if len(steps) != len(commits) {
		t.Fatalf("got %d epoch rows, want one per cx entry (%d): %+v", len(steps), len(commits), steps)
	}
	for i, s := range steps {
		if s.Status != StatusFail {
			t.Errorf("row %d: got %v, want fail", i, s.Status)
		}
		if !containsStr(s.Message, fmt.Sprintf("Epoch proof %d", i)) {
			t.Errorf("row %d should name its own index, got: %s", i, s.Message)
		}
	}
}

// --- F3: E.4 number portability ---

// TestRun_BigIntegerBundle_WarnsAndStillVerifies is the F3 regression.
// The fixture's s.d carries 2^53+1 and an unsigned 64-bit ledger
// sequence; every derived value in it was computed against the exact
// literals, which is what the producer signs. Rounding them through a
// double — what the unwrapped RFC 8785 library does — yields a different
// claims hash and reports this sound bundle as forged.
//
// E.4 requires the non-portability be reported, and the appendix
// authorizes no skip for it: the digest this run derived is the one the
// producer signed, so the inclusion and epoch walks genuinely verify and
// must keep saying so.
func TestRun_BigIntegerBundle_WarnsAndStillVerifies(t *testing.T) {
	report, err := Run("testdata/fixtures/item-bigint.json", Options{
		SkipExternal:   true,
		SkipSignatures: true,
	})
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	if !report.Passed() {
		for _, s := range report.Steps {
			if s.Status == StatusFail {
				t.Errorf("FAIL: [%s] %s", s.Group, s.Message)
			}
		}
		t.Fatal("a bundle with oversized integers in s.d is sound and must verify")
	}

	var warns []Step
	for _, s := range stepsIn(report, groupSubjectData) {
		if s.Status == StatusWarn {
			warns = append(warns, s)
		}
	}
	if len(warns) != 1 {
		t.Fatalf("expected exactly one portability warn, got %+v", warns)
	}
	w := warns[0]
	if w.Category != CatCryptographic {
		t.Errorf("category: got %q, want %q", w.Category, CatCryptographic)
	}
	for _, want := range []string{"not portably verifiable", "2 integer(s)", "9007199254740993"} {
		if !strings.Contains(w.Message, want) {
			t.Errorf("warn message %q is missing %q", w.Message, want)
		}
	}

	// The dependent steps must not be downgraded. Reporting them as
	// skipped would suppress two cryptographic facts this run
	// established.
	for _, group := range []string{groupInclusion, groupBlockHash, groupEpoch} {
		step := onlyStep(t, report, group)
		if step.Status != StatusPass {
			t.Errorf("%s: got %v, want pass (%s)", group, step.Status, step.Message)
		}
	}
}

// TestDeriveClaimsHash_PortabilityThreshold pins the boundary: 2^53 is
// exactly representable and must not warn, 2^53+1 must. Floats are never
// flagged however lossy, matching the reference verifier.
func TestDeriveClaimsHash_PortabilityThreshold(t *testing.T) {
	cases := []struct {
		name     string
		claims   string
		wantWarn bool
	}{
		{"at the limit", `{"n":9007199254740992}`, false},
		{"one past the limit", `{"n":9007199254740993}`, true},
		{"negative past the limit", `{"n":-9007199254740993}`, true},
		{"lossy float", `{"n":1.7976931348623157e308}`, false},
		{"small integers", `{"a":1,"b":-42}`, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			r := &Report{}
			if hash := deriveClaimsHash(r, []byte(tc.claims)); hash == "" {
				t.Fatal("expected a derived claims hash")
			}
			warned := false
			for _, s := range r.Steps {
				if s.Status == StatusWarn {
					warned = true
				}
			}
			if warned != tc.wantWarn {
				t.Errorf("warned = %v, want %v (steps: %+v)", warned, tc.wantWarn, r.Steps)
			}
		})
	}
}

// TestDeriveEntropyHash_OversizedIntegerWarns covers the 0x21 path.
// Entropy payloads are where E.4's hazard actually bites — ledger
// sequences, nonces and difficulty values routinely exceed 2^53.
func TestDeriveEntropyHash_OversizedIntegerWarns(t *testing.T) {
	r := &Report{}
	if hash := deriveEntropyHash(r, []byte(`{"sequence":18446744073709551615}`)); hash == "" {
		t.Fatal("expected a derived entropy hash")
	}
	steps := stepsIn(r, groupSubjectData)
	if len(steps) != 2 || steps[0].Status != StatusPass || steps[1].Status != StatusWarn {
		t.Fatalf("expected a pass then a warn, got %+v", steps)
	}
	if !strings.Contains(steps[1].Message, "entropy hash") {
		t.Errorf("the warn should name the entropy hash, got: %s", steps[1].Message)
	}
}

// --- E.22 group names and the Temporal Info row ---

// TestReport_GroupNamesMatchAppendixE22 guards the names a consumer
// keys on. E.22's group table is normative, and the CLI previously
// filed E.20 under "Temporal Window", emitted no "Temporal Info" row at
// all, and had no "Key Binding" group.
func TestReport_GroupNamesMatchAppendixE22(t *testing.T) {
	report, err := Run("testdata/proof_item.json", Options{SkipExternal: true})
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	seen := map[string]bool{}
	for _, s := range report.Steps {
		seen[s.Group] = true
	}
	for _, want := range []string{
		"Hash Comparison", "Signing Key", "Subject Data", "Inclusion Proof",
		"Block Hash", "Epoch Proof", "Proof Signature", "Key Binding",
		"Structure", "Submission Window", "Temporal Info", "Stellar Commitment",
	} {
		if !seen[want] {
			t.Errorf("Appendix E.22 group %q is absent from the report", want)
		}
	}
	if seen["Temporal Window"] {
		t.Error(`"Temporal Window" is not an Appendix E.22 group name; E.20's group is "Submission Window"`)
	}
}

// TestAddTemporalInfo_EmitsATimingInfoRow pins the D.4 row directly.
// It records the bracket and carries no pass/fail — the one temporal
// assertion a bundle supports is E.20's ordering check, which has its
// own step.
func TestAddTemporalInfo_EmitsATimingInfoRow(t *testing.T) {
	block := proof.Block{ID: "019f93ff-2600-7c30-8000-000000000c30"}
	subject := &proof.Subject{ID: "01KY9ZWEX0248J48HK6D248NAN"}

	r := &Report{}
	addTemporalInfo(r, ptype.Item, subject, block)
	step := onlyStep(t, r, groupTemporalInfo)
	if step.Status != StatusInfo {
		t.Errorf("status: got %v, want info", step.Status)
	}
	if step.Category != CatTiming {
		t.Errorf("category: got %q, want %q", step.Category, CatTiming)
	}
	if !strings.Contains(step.Message, r.Temporal.SubmittedAt) ||
		!strings.Contains(step.Message, r.Temporal.CommittedAt) {
		t.Errorf("message %q should carry both bracket edges", step.Message)
	}

	// An id whose embedded milliseconds cannot be read records nothing,
	// so no row is emitted rather than one reading "unknown".
	empty := &Report{}
	addTemporalInfo(empty, ptype.Item, &proof.Subject{ID: "not-a-ulid"}, proof.Block{ID: "not-a-uuid"})
	if got := stepsIn(empty, groupTemporalInfo); len(got) != 0 {
		t.Errorf("expected no Temporal Info row when nothing is extractable, got %+v", got)
	}
}

// --- R2: E.22 message honesty and step correctness ---
//
// Every test below fails on the pre-fix code: the message it asserts
// against, or the status it pins, is the one the adversarial judging
// pass found the pipeline emitting.

// E.22 forbids a message from asserting what its branch did not
// establish. The E.7 warn is emitted at pipeline position 2, before the
// signature, inclusion, epoch and block-hash steps have run, so it
// cannot say the proof is verified — and it used to say exactly that
// inside reports whose verdict was `failed`.
func TestHashComparisonWarn_AssertsNothingAboutTheProof(t *testing.T) {
	data, err := os.ReadFile("testdata/fixtures/appendix-d-item.json")
	if err != nil {
		t.Fatalf("reading fixture: %s", err)
	}
	// Break the signature so the run ends in `failed`, then read the
	// warn that a passing run and a failing run share verbatim.
	var bundle map[string]any
	if err := json.Unmarshal(data, &bundle); err != nil {
		t.Fatalf("decoding fixture: %s", err)
	}
	sig, err := base64.StdEncoding.DecodeString(bundle["sig"].(string))
	if err != nil {
		t.Fatalf("decoding sig: %s", err)
	}
	sig[0] ^= 0xff
	bundle["sig"] = base64.StdEncoding.EncodeToString(sig)
	tampered, err := json.Marshal(bundle)
	if err != nil {
		t.Fatalf("re-encoding fixture: %s", err)
	}

	report, err := RunFromBytes(tampered, "(test)", Options{SkipExternal: true})
	if err != nil {
		t.Fatalf("RunFromBytes: %s", err)
	}
	if report.Passed() {
		t.Fatal("fixture setup: the tampered signature should fail the run")
	}
	step := onlyStep(t, report, groupHashComparison)
	if step.Status != StatusWarn {
		t.Fatalf("status: got %v, want warn (%s)", step.Status, step.Message)
	}
	for _, banned := range []string{"proof itself is verified", "is verified"} {
		if strings.Contains(step.Message, banned) {
			t.Errorf("the warn asserts %q inside a failed report: %s", banned, step.Message)
		}
	}
	// It must still name the hash the proof commits to, which is the
	// one thing this branch did establish.
	if !strings.Contains(step.Message, appendixDClaimsHash) {
		t.Errorf("the warn should name the hash the proof commits to: %s", step.Message)
	}
}

// E.7's "no s.d.hash" rule is about the subject, not about `t`. A t=20
// claims-as-source-of-truth item carries no file hash either, and
// comparing the caller's argument against "" reported a sound proof as
// covering different data than the caller's.
func TestHashComparison_ClaimsOnlyItem_SkipsAndKeepsTheNote(t *testing.T) {
	data, err := os.ReadFile("testdata/proof_item_claims_only.json")
	if err != nil {
		t.Fatalf("reading fixture: %s", err)
	}

	report, err := RunFromBytes(data, "(test)", Options{
		SkipExternal: true, SkipSignatures: true,
		ExpectedHash: "b47cc0f104b62d4c7c30bcd68fd8e67613e287dc4ad8c310ef10cbadea9c4380",
	})
	if err != nil {
		t.Fatalf("RunFromBytes: %s", err)
	}

	step := onlyStep(t, report, groupHashComparison)
	if step.Status != StatusSkip {
		t.Errorf("status: got %v, want skip — there is no s.d.hash to compare against (%s)", step.Status, step.Message)
	}
	if strings.Contains(step.Message, "does not match") {
		t.Errorf("a mismatch against an absent operand is not a mismatch: %s", step.Message)
	}
	// Supplying --hash must not delete the note that explains why there
	// is nothing to compare.
	if !strings.Contains(stepMessages(report), "Claims-only item") {
		t.Errorf("the claims-only note must survive a supplied --hash:\n%s", stepMessages(report))
	}
	// E.22 separability: the caller did supply one.
	if report.HashProvided == "" {
		t.Error("HashProvided must record the caller's argument even when it cannot be compared")
	}
	if report.HashMatched() {
		t.Error("nothing was compared, so nothing matched")
	}
	if v := report.Verdict(); v == VerdictHashMismatch {
		t.Error("a claims-only item with --hash must not report a hash mismatch")
	}
}

// E.22: a message must not name something the bundle does not carry.
// R1a's parse relaxation routes a present-but-wrong-typed b.id to this
// step as "", which used to render "Inclusion proof to block  (N steps)".
func TestVerifyInclusionProof_UnnamedBlock_IsNotNamed(t *testing.T) {
	const subjectHash = "5c8f1b3d4a2e6f7089abcdef0123456789abcdef0123456789abcdef01234567"
	leaf := tscrypto.DomainHash(tscrypto.PrefixMerkleLeaf, mustHex(t, subjectHash))

	r := &Report{}
	verifyInclusionProof(r, subjectHash, "AA", proof.Block{MerkleRoot: tscrypto.BytesToHex(leaf)})

	step := onlyStep(t, r, groupInclusion)
	if step.Status != StatusPass {
		t.Fatalf("fixture setup: expected a passing walk, got %v (%s)", step.Status, step.Message)
	}
	if strings.Contains(step.Message, "to block  ") || strings.HasSuffix(step.Message, "to block ") {
		t.Errorf("the message names an absent block: %q", step.Message)
	}
	if !strings.Contains(step.Message, "no usable id") {
		t.Errorf("the message should say the block cannot be named: %q", step.Message)
	}
}

func mustHex(t *testing.T, s string) []byte {
	t.Helper()
	b, err := hex.DecodeString(s)
	if err != nil {
		t.Fatalf("decoding %q: %s", s, err)
	}
	return b
}

// E.14 fixes the block preimage at 157 bytes unconditionally. A field
// that is present but short is framed by len32 at whatever width it has,
// so the step used to assert E.14's derivation over a 141-byte preimage.
func TestDeriveBlockHash_WrongWidthField_IsNotDerived(t *testing.T) {
	full := proof.Block{
		ID:                "019cf813-99b8-730a-84f1-5a711a9c355e",
		PreviousBlockHash: strings.Repeat("11", 32),
		MerkleRoot:        strings.Repeat("22", 32),
		MetadataHash:      strings.Repeat("44", 32),
		SigningKeyID:      "4ceefa4a",
	}
	// Baseline: the untouched block derives.
	if hash := deriveBlockHash(&Report{}, full); hash == "" {
		t.Fatal("fixture setup: the full block should derive a hash")
	}

	for _, tc := range []struct {
		name   string
		mutate func(*proof.Block)
	}{
		{"short b.mh", func(b *proof.Block) { b.MetadataHash = strings.Repeat("14", 16) }},
		{"short b.kid", func(b *proof.Block) { b.SigningKeyID = "f2c3" }},
		{"long b.ph", func(b *proof.Block) { b.PreviousBlockHash = strings.Repeat("11", 40) }},
		{"short b.id", func(b *proof.Block) { b.ID = "019cf813-99b8-730a" }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			block := full
			tc.mutate(&block)
			r := &Report{}
			if hash := deriveBlockHash(r, block); hash != "" {
				t.Errorf("a preimage of the wrong size must not derive a block hash, got %q", hash)
			}
			step := onlyStep(t, r, groupBlockHash)
			if step.Status != StatusFail {
				t.Errorf("status: got %v, want fail (%s)", step.Status, step.Message)
			}
			if !strings.Contains(step.Message, "157") {
				t.Errorf("the message should name E.14's fixed preimage size: %s", step.Message)
			}
		})
	}
}

// E.10 fixes the item composite at 111 bytes and the entropy composite
// at 121. Same len32 framing hazard as E.14.
func TestDeriveCompositeHashes_WrongWidthField_IsNotDerived(t *testing.T) {
	claimsHash := strings.Repeat("ab", 32)
	item := proof.Subject{
		ID:           "01KY9ZWEX0248J48HK6D248NAN",
		MetadataHash: strings.Repeat("cd", 32),
		SigningKeyID: "4ceefa4a",
	}
	if h := deriveItemHash(&Report{}, &item, claimsHash); h == "" {
		t.Fatal("fixture setup: the full item subject should derive")
	}
	short := item
	short.MetadataHash = strings.Repeat("cd", 16)
	r := &Report{}
	if h := deriveItemHash(r, &short, claimsHash); h != "" {
		t.Errorf("a short s.mh must not derive an item hash, got %q", h)
	}
	if step := onlyStep(t, r, groupSubjectData); step.Status != StatusFail || !strings.Contains(step.Message, "111") {
		t.Errorf("expected an E.10 width failure naming 111: %v %s", step.Status, step.Message)
	}

	entropy := proof.Subject{
		ID:           "019cf813-99b8-730a-84f1-5a711a9c355e",
		MetadataHash: strings.Repeat("cd", 32),
		SigningKeyID: "4ceefa4a",
	}
	if h := deriveObservationHash(&Report{}, &entropy, claimsHash); h == "" {
		t.Fatal("fixture setup: the full entropy subject should derive")
	}
	shortEnt := entropy
	shortEnt.SigningKeyID = "4cee"
	r2 := &Report{}
	if h := deriveObservationHash(r2, &shortEnt, claimsHash); h != "" {
		t.Errorf("a short s.kid must not derive an observation hash, got %q", h)
	}
	if step := onlyStep(t, r2, groupSubjectData); step.Status != StatusFail || !strings.Contains(step.Message, "121") {
		t.Errorf("expected an E.10 width failure naming 121: %v %s", step.Status, step.Message)
	}
}

// R1a's parse relaxation routes a present-but-wrong-typed field to these
// steps as "". A message reading "missing" would be false about a field
// the file demonstrably carries, and it must name WHICH field.
func TestDerivationGuards_NameTheUnusableField(t *testing.T) {
	r := &Report{}
	deriveBlockHash(r, proof.Block{
		ID:                "019cf813-99b8-730a-84f1-5a711a9c355e",
		PreviousBlockHash: strings.Repeat("11", 32),
		MetadataHash:      strings.Repeat("44", 32),
		SigningKeyID:      "4ceefa4a",
	})
	step := onlyStep(t, r, groupBlockHash)
	if strings.Contains(step.Message, "missing") {
		t.Errorf("a wrong-typed field is present, not missing: %s", step.Message)
	}
	if !strings.Contains(step.Message, "b.mr") {
		t.Errorf("the message should name the unusable field: %s", step.Message)
	}
}

// E.20's third bullet is a MUST with a negative object: a verifier that
// does not retrieve the previous block and its entropy leaves MUST
// report the submitted-after edge as not established. Silence is not a
// report.
func TestSubmissionWindowEdges_SubmittedAfterIsReportedUnestablished(t *testing.T) {
	report, err := Run(appendixDFixture(t), Options{SkipExternal: true, ExpectedHash: appendixDClaimsHash})
	if err != nil {
		t.Fatalf("Run: %s", err)
	}

	after := onlyStep(t, report, groupSubmittedAfter)
	if after.Status != StatusInfo {
		t.Errorf("status: got %v, want info — this row grades nothing", after.Status)
	}
	if after.Category != CatTiming {
		t.Errorf("category: got %q, want %q", after.Category, CatTiming)
	}
	if !strings.Contains(after.Message, "Not established from this bundle") {
		t.Errorf("E.20 requires the edge be reported as not established: %s", after.Message)
	}

	before := onlyStep(t, report, groupSubmittedBefore)
	if before.Status != StatusInfo {
		t.Errorf("status: got %v, want info", before.Status)
	}
	// E.20: the submitted-before edge is grounded by the EARLIEST cx
	// transaction, so that is the timestamp the row names.
	if !strings.Contains(before.Message, "2026-07-24T12:00:12Z") {
		t.Errorf("the row should name the earliest commitment timestamp: %s", before.Message)
	}

	// E.20 bullet 2: the ordering constraint is Truestamp's assertion
	// and MUST NOT be presented as externally verified.
	window := onlyStep(t, report, groupSubmissionWindow)
	if !strings.Contains(window.Message, "not externally verified") {
		t.Errorf("the ordering pass must say whose assertion it is: %s", window.Message)
	}

	// Block-like subjects have no submission window at all, so neither
	// edge row is emitted (E.20's closing paragraph).
	blockReport, err := Run("testdata/fixtures/block.json", Options{SkipExternal: true})
	if err != nil {
		t.Fatalf("Run block fixture: %s", err)
	}
	if got := stepsIn(blockReport, groupSubmittedAfter); len(got) != 0 {
		t.Errorf("a block-like subject has no submission window: %+v", got)
	}
}

// E.20's ordering check compares at millisecond resolution, so its
// failure must render at that resolution. Printed through RFC3339 a
// sub-second violation showed the same instant twice: "X is AFTER X".
func TestSubmissionWindow_SubSecondViolation_IsNotSelfRefuting(t *testing.T) {
	// b.id embeds 2026-07-24T12:00:00.000Z; s.id embeds +500ms.
	block := proof.Block{ID: "019f93ff-2600-7c30-8000-000000000c30"}
	subject := &proof.Subject{ID: "019f93ff-27f4-70a1-8000-0000000000a1"}

	r := &Report{}
	verifySubjectTemporalWindow(r, ptype.EntropyNIST, subject, block)
	step := onlyStep(t, r, groupSubmissionWindow)
	if step.Status != StatusFail {
		t.Fatalf("fixture setup: expected an ordering violation, got %v (%s)", step.Status, step.Message)
	}

	// The two instants named must differ, or the sentence refutes itself.
	parts := strings.Split(step.Message, " is AFTER committed block time ")
	if len(parts) != 2 {
		t.Fatalf("unexpected message shape: %s", step.Message)
	}
	subjectAt := parts[0][strings.LastIndex(parts[0], " at ")+len(" at "):]
	if subjectAt == parts[1] {
		t.Errorf("the message asserts %s is after %s — the same instant twice: %s", subjectAt, parts[1], step.Message)
	}
}

// E.4 makes every digest comparison constant-time, and E.21 fails only
// on a genuine value mismatch. A bare `!=` also made this comparison
// case-sensitive, so a subject carrying upstream's own bytes in the
// other hex case was graded a mismatch and failed the proof.
func TestVerifyEntropyNIST_HexCaseIsNotAMismatch(t *testing.T) {
	const upstream = "7665F054F21B50DF62CD3E50AF8EB783E30D271B091DE051212D301E0E3D17F17665F054F21B50DF62CD3E50AF8EB783E30D271B091DE051212D301E0E3D17F1"
	nistStub(t, func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"pulse": map[string]any{
				"chainIndex": 1, "pulseIndex": 1,
				"outputValue": upstream,
				"timeStamp":   "2026-07-24T12:00:00.000Z",
				"version":     "2.0",
			},
		})
	})

	subject := fmt.Sprintf(`{"chainIndex":1,"pulseIndex":1,"outputValue":%q,"timeStamp":"2026-07-24T12:00:00.000Z","version":"2.0"}`,
		strings.ToLower(upstream))
	r := &Report{}
	verifyEntropyNIST(r, json.RawMessage(subject))

	step := onlyStep(t, r, groupEntropySource)
	if step.Status != StatusPass {
		t.Errorf("a hex-case difference is not a value mismatch: %v (%s)", step.Status, step.Message)
	}

	// Control: a genuine value difference must still fail.
	other := strings.Repeat("ab", 64)
	bad := fmt.Sprintf(`{"chainIndex":1,"pulseIndex":1,"outputValue":%q,"timeStamp":"2026-07-24T12:00:00.000Z","version":"2.0"}`, other)
	r2 := &Report{}
	verifyEntropyNIST(r2, json.RawMessage(bad))
	if step := onlyStep(t, r2, groupEntropySource); step.Status != StatusFail {
		t.Errorf("a genuine outputValue mismatch must still fail: %v (%s)", step.Status, step.Message)
	}
}

// E.21's Bitcoin entropy comparison is made against `height` and `time`;
// the hash is the value the block was looked UP by. A subject carrying
// neither used to produce "Bitcoin block <upstream height> confirmed on
// mainnet" — a confirmation of a height the bundle never asserted.
func TestVerifyEntropyBitcoin_AbsentHeightAndTime_DoesNotAssertThem(t *testing.T) {
	const blockHash = "0000000000000000000087b8d732041d6d1ebb7cab4fe5c5ca74df2f362ff321"
	blockstreamStub(t, func(w http.ResponseWriter, req *http.Request) {
		blockstreamBlockBody(t, w, req, 959514, 1784958887)
	})

	r := &Report{}
	verifyEntropyBitcoin(r, json.RawMessage(fmt.Sprintf(`{"hash":%q}`, blockHash)), "mainnet")
	step := onlyStep(t, r, groupEntropySource)
	if strings.Contains(step.Message, "959514") {
		t.Errorf("the row asserts an upstream height the subject never carried: %s", step.Message)
	}
	if !strings.Contains(step.Message, "no height or time") {
		t.Errorf("the row should say what it could not compare: %s", step.Message)
	}

	// With the fields present, the comparison is real and the message
	// names what it compared.
	r2 := &Report{}
	verifyEntropyBitcoin(r2, json.RawMessage(fmt.Sprintf(`{"hash":%q,"height":959514,"time":1784958887}`, blockHash)), "mainnet")
	step2 := onlyStep(t, r2, groupEntropySource)
	if step2.Status != StatusPass {
		t.Fatalf("a matching subject should pass: %v (%s)", step2.Status, step2.Message)
	}
	if !strings.Contains(step2.Message, "height, time") {
		t.Errorf("the row should name the fields it compared: %s", step2.Message)
	}

	// Control: a height that disagrees still fails.
	r3 := &Report{}
	verifyEntropyBitcoin(r3, json.RawMessage(fmt.Sprintf(`{"hash":%q,"height":1}`, blockHash)), "mainnet")
	if step := onlyStep(t, r3, groupEntropySource); step.Status != StatusFail {
		t.Errorf("a height mismatch must still fail: %v (%s)", step.Status, step.Message)
	}
}

// An answer about a different block is a defect in the upstream, not in
// the bundle: the lookup was BY the subject's hash, and nothing the
// subject can assert makes a well-behaved endpoint answer about another
// block. E.22 forbids an uninterpretable upstream answer from failing a
// proof.
func TestVerifyEntropyBitcoin_MisaddressedAnswer_Skips(t *testing.T) {
	blockstreamStub(t, func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"id": strings.Repeat("de", 32), "height": 959514, "timestamp": 1784958887,
		})
	})

	r := &Report{}
	verifyEntropyBitcoin(r, json.RawMessage(fmt.Sprintf(`{"hash":%q,"height":959514}`, strings.Repeat("ab", 32))), "mainnet")
	step := onlyStep(t, r, groupEntropySource)
	if step.Status != StatusSkip {
		t.Errorf("status: got %v, want skip — the endpoint answered about another block (%s)", step.Status, step.Message)
	}
	if !strings.Contains(step.Message, "not the block") {
		t.Errorf("the message should say the answer was about another block: %s", step.Message)
	}
}

// E.18's defaulted-network branch covers an absent `net` AND a
// present-but-unrecognised one. Reporting both as "the entry names no
// network" contradicts CommitmentInfo.Network in the same report.
func TestVerifySingleStellar_UnrecognisedNetwork_MessageNamesIt(t *testing.T) {
	horizonStub(t, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	})

	cx := stellarSample()
	cx.Network = "bogusnet"
	r := &Report{}
	verifySingleStellar(r, &cx, Options{})

	step := onlyStep(t, r, groupStellar)
	if step.Status != StatusSkip {
		t.Fatalf("status: got %v, want skip (%s)", step.Status, step.Message)
	}
	if strings.Contains(step.Message, "names no network") {
		t.Errorf("the entry names %q; the message says it names none: %s", cx.Network, step.Message)
	}
	if !strings.Contains(step.Message, "bogusnet") {
		t.Errorf("the message should quote the name the entry carries: %s", step.Message)
	}

	// The absent case keeps its own wording.
	cx2 := stellarSample()
	cx2.Network = ""
	r2 := &Report{}
	verifySingleStellar(r2, &cx2, Options{})
	if step := onlyStep(t, r2, groupStellar); !strings.Contains(step.Message, "names no network") {
		t.Errorf("an absent net should still say so: %s", step.Message)
	}
}

// The Bitcoin binding skip must state the fact it has — which network
// name the entry carries and why no lookup ran — rather than asserting
// that the value is a Bitcoin network without a public API. It must also
// disclose that `net` is outside the signature, because a skip that
// reads as "not on chain was not established" is exactly what a rewritten
// `net` buys an attacker.
func TestVerifyBitcoinBinding_UnrecognisedNetwork_MessageAndDisclosure(t *testing.T) {
	for _, tc := range []struct {
		name    string
		network string
		want    string
	}{
		{"absent", "", "names no Bitcoin network"},
		{"unrecognised", "bogusnet", `"bogusnet"`},
		{"regtest", "regtest", "no public API for regtest"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cx := btcSample()
			cx.Network = tc.network
			r := &Report{}
			verifySingleBitcoin(r, &cx, Options{})

			msgs := stepMessages(r)
			if !strings.Contains(msgs, tc.want) {
				t.Errorf("expected %q in the skip message:\n%s", tc.want, msgs)
			}
			if !strings.Contains(msgs, "not covered by the proof signature") {
				t.Errorf("the report must disclose the downgrade lever:\n%s", msgs)
			}
			if n := countStatus(r, StatusFail); n != 0 {
				t.Errorf("an unlookupable network must not fail the proof:\n%s", msgs)
			}
		})
	}
}

// E.25 lists the steps a verifier MAY skip and still call the run
// verified; E.16 is not among them. The skip row alone sits next to a
// "VERIFIED" verdict, so the run also carries an explicit disclosure.
func TestSkipSignatures_DisclosesThatNothingWasChecked(t *testing.T) {
	report, err := Run(appendixDFixture(t), Options{SkipExternal: true, SkipSignatures: true})
	if err != nil {
		t.Fatalf("Run: %s", err)
	}

	got := stepsIn(report, groupProof)
	if len(got) != 2 {
		t.Fatalf("expected the skip plus its disclosure, got %+v", got)
	}
	var sawSkip, sawWarn bool
	for _, s := range got {
		switch s.Status {
		case StatusSkip:
			sawSkip = true
		case StatusWarn:
			sawWarn = true
			if !strings.Contains(s.Message, "not checked") {
				t.Errorf("the disclosure should say the signature was not checked: %s", s.Message)
			}
		}
	}
	if !sawSkip || !sawWarn {
		t.Errorf("want one skip and one warn under %s, got %+v", groupProof, got)
	}

	// The disclosure must not move the verdict: --skip-signatures is a
	// documented mode, not a failure.
	if !report.Passed() {
		t.Error("a warn must not fail the run")
	}
}

// The report names the key that actually signed, which E.9/E.16 make the
// pk-derived key id fed to the 0x61 payload, not the stored b.kid. Under
// E.9-blessed rotation the two differ and the report used to name b.kid.
func TestReportSigningKeyID_IsTheKeyThatSigned(t *testing.T) {
	data, err := os.ReadFile("testdata/fixtures/appendix-d-item.json")
	if err != nil {
		t.Fatalf("reading fixture: %s", err)
	}
	var bundle map[string]any
	if err := json.Unmarshal(data, &bundle); err != nil {
		t.Fatalf("decoding fixture: %s", err)
	}
	block := bundle["b"].(map[string]any)
	block["kid"] = "11223344" // rotation: b.kid is no longer the signer
	rotated, err := json.Marshal(bundle)
	if err != nil {
		t.Fatalf("re-encoding fixture: %s", err)
	}

	report, err := RunFromBytes(rotated, "(test)", Options{SkipExternal: true})
	if err != nil {
		t.Fatalf("RunFromBytes: %s", err)
	}
	if report.SigningKeyID == "11223344" {
		t.Error("the report names b.kid as the signer; E.16 signs with the pk-derived key id")
	}
	// The Signing Key row and the report field must agree — a report
	// that prints two different key ids for one key contradicts itself.
	step := onlyStep(t, report, groupSigningKey)
	if !strings.Contains(step.Message, report.SigningKeyID) {
		t.Errorf("Report.SigningKeyID %q disagrees with the Signing Key row: %s", report.SigningKeyID, step.Message)
	}
}

// The companion field: `b.kid` is a block field, an input to E.14's 0x32
// preimage, and every surface that describes the BLOCK must publish it
// verbatim rather than substituting the derived signer.
//
// Local mode never populated Report.BlockSigningKeyID, so BlockKeyID()
// fell through to the derived id at both display sites (the presenter's
// Block/Beacon Subject row and --json subject.signing_key). Remote mode
// did populate it. On a rotated bundle the two modes therefore published
// different key ids for the same block, which is the one disagreement
// the two fields exist to prevent.
func TestReportBlockSigningKeyID_IsTheBlocksOwnKid(t *testing.T) {
	data, err := os.ReadFile("testdata/fixtures/appendix-d-item.json")
	if err != nil {
		t.Fatalf("reading fixture: %s", err)
	}
	var bundle map[string]any
	if err := json.Unmarshal(data, &bundle); err != nil {
		t.Fatalf("decoding fixture: %s", err)
	}
	bundle["b"].(map[string]any)["kid"] = "11223344" // rotation
	rotated, err := json.Marshal(bundle)
	if err != nil {
		t.Fatalf("re-encoding fixture: %s", err)
	}

	report, err := RunFromBytes(rotated, "(test)", Options{SkipExternal: true})
	if err != nil {
		t.Fatalf("RunFromBytes: %s", err)
	}
	if report.BlockSigningKeyID != "11223344" {
		t.Errorf("BlockSigningKeyID: got %q, want the bundle's b.kid 11223344", report.BlockSigningKeyID)
	}
	if got := report.BlockKeyID(); got != "11223344" {
		t.Errorf("BlockKeyID(): got %q, want b.kid 11223344 — a block field must not display the derived signer", got)
	}
	// The two must stay distinguishable, or the assertion above would be
	// satisfied by a fixture that never rotated.
	if report.SigningKeyID == report.BlockSigningKeyID {
		t.Fatalf("fixture is not rotated: signer and b.kid are both %q", report.SigningKeyID)
	}
}

// --- R2: soundness proofs for the two gradings this pass relaxed ---

// The --hash mismatch was pinned by no test at all: the whole suite
// stayed green with the fail downgraded to a skip, so a wrong --hash
// reported "verified" and exited 0. This pins the status, not just the
// wording.
func TestHashComparison_Mismatch_FailsTheRun(t *testing.T) {
	report, err := Run(appendixDFixture(t), Options{
		SkipExternal: true,
		ExpectedHash: strings.Repeat("ab", 32),
	})
	if err != nil {
		t.Fatalf("Run: %s", err)
	}

	step := onlyStep(t, report, groupHashComparison)
	if step.Status != StatusFail {
		t.Fatalf("status: got %v, want fail (%s)", step.Status, step.Message)
	}
	if report.Passed() {
		t.Error("a wrong --hash must not exit 0")
	}
	if got := report.Verdict(); got != VerdictHashMismatch {
		t.Errorf("verdict: got %v, want VerdictHashMismatch", got)
	}
	// E.7: the proof itself is sound, so this is a statement about the
	// caller's data, not a broken proof.
	if !report.ProofPassed() {
		t.Error("a hash mismatch must not be reported as a defective proof")
	}
}

// A claims-only item reports --hash as a skip rather than a mismatch.
// That relaxation cannot be used to launder a mismatch, because s.d.hash
// is inside the 0x11 preimage: deleting it to reach the skip branch
// breaks the derivation chain and the run still fails.
func TestClaimsOnlySkip_CannotBeReachedByDeletingTheHash(t *testing.T) {
	data, err := os.ReadFile("testdata/fixtures/item.json")
	if err != nil {
		t.Fatalf("reading fixture: %s", err)
	}
	baseline, err := RunFromBytes(data, "(test)", Options{SkipExternal: true})
	if err != nil {
		t.Fatalf("RunFromBytes: %s", err)
	}
	if !baseline.Passed() {
		t.Fatalf("fixture setup: item.json should verify offline:\n%s", formatSteps(baseline.Steps))
	}

	var bundle map[string]any
	if err := json.Unmarshal(data, &bundle); err != nil {
		t.Fatalf("decoding fixture: %s", err)
	}
	sd := bundle["s"].(map[string]any)["d"].(map[string]any)
	if _, ok := sd["hash"]; !ok {
		t.Fatal("fixture setup: item.json should carry an s.d.hash")
	}
	delete(sd, "hash")
	stripped, err := json.Marshal(bundle)
	if err != nil {
		t.Fatalf("re-encoding fixture: %s", err)
	}

	report, err := RunFromBytes(stripped, "(test)", Options{
		SkipExternal: true,
		ExpectedHash: strings.Repeat("ab", 32),
	})
	if err != nil {
		t.Fatalf("RunFromBytes: %s", err)
	}
	if report.Passed() {
		t.Errorf("deleting s.d.hash to reach the skip branch must break the proof:\n%s", formatSteps(report.Steps))
	}
}

// The Bitcoin entropy hash comparison grades a mis-addressed upstream
// answer as unconfirmed rather than as a proof failure. That cannot hide
// a tampered s.d: the lookup is BY s.d.hash, so an endpoint that answers
// about the block it was asked for can never reach the skip branch,
// whatever the subject claims.
func TestVerifyEntropyBitcoin_WellBehavedEndpoint_CannotReachTheSkip(t *testing.T) {
	blockstreamStub(t, func(w http.ResponseWriter, req *http.Request) {
		blockstreamBlockBody(t, w, req, 959514, 1784958887)
	})

	for _, tc := range []struct {
		name    string
		subject string
		want    Status
	}{
		{"genuine", `{"hash":"%s","height":959514,"time":1784958887}`, StatusPass},
		{"tampered height", `{"hash":"%s","height":42,"time":1784958887}`, StatusFail},
		{"tampered time", `{"hash":"%s","height":959514,"time":1}`, StatusFail},
	} {
		t.Run(tc.name, func(t *testing.T) {
			r := &Report{}
			verifyEntropyBitcoin(r, json.RawMessage(fmt.Sprintf(tc.subject, strings.Repeat("0a", 32))), "mainnet")
			step := onlyStep(t, r, groupEntropySource)
			if step.Status != tc.want {
				t.Errorf("status: got %v, want %v (%s)", step.Status, tc.want, step.Message)
			}
		})
	}
}

// E.21's other two entropy comparisons fail on a genuine value
// disagreement. Pinned by status so the sites cannot be downgraded to a
// skip unnoticed.
func TestEntropySource_ValueDisagreements_Fail(t *testing.T) {
	t.Run("nist timeStamp", func(t *testing.T) {
		const digest = "7665f054f21b50df62cd3e50af8eb783e30d271b091de051212d301e0e3d17f17665f054f21b50df62cd3e50af8eb783e30d271b091de051212d301e0e3d17f1"
		nistStub(t, func(w http.ResponseWriter, _ *http.Request) {
			_ = json.NewEncoder(w).Encode(map[string]any{"pulse": map[string]any{
				"chainIndex": 1, "pulseIndex": 1, "outputValue": digest,
				"timeStamp": "2026-07-24T12:00:00.000Z", "version": "2.0",
			}})
		})
		r := &Report{}
		verifyEntropyNIST(r, json.RawMessage(fmt.Sprintf(
			`{"chainIndex":1,"pulseIndex":1,"outputValue":%q,"timeStamp":"2020-01-01T00:00:00.000Z"}`, digest)))
		if step := onlyStep(t, r, groupEntropySource); step.Status != StatusFail {
			t.Errorf("status: got %v, want fail (%s)", step.Status, step.Message)
		}
	})

	t.Run("stellar ledger hash", func(t *testing.T) {
		horizonStub(t, func(w http.ResponseWriter, _ *http.Request) {
			_ = json.NewEncoder(w).Encode(map[string]any{
				"hash": strings.Repeat("aa", 32), "sequence": 51234567, "closed_at": "2026-07-24T12:00:00Z",
			})
		})
		r := &Report{}
		verifyEntropyStellar(r, json.RawMessage(fmt.Sprintf(
			`{"hash":%q,"sequence":51234567,"closed_at":"2026-07-24T12:00:00Z"}`, strings.Repeat("bb", 32))), "testnet")
		if step := onlyStep(t, r, groupEntropySource); step.Status != StatusFail {
			t.Errorf("status: got %v, want fail (%s)", step.Status, step.Message)
		}
	})
}

// E.18's settled grading: a 404 on a network the entry NAMED is a
// definitive absence and fails. Pinned by status — the message alone
// survives a fail -> skip downgrade.
func TestVerifySingleStellar_NotFoundOnNamedNetwork_StatusIsFail(t *testing.T) {
	horizonStub(t, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	})

	cx := stellarSample()
	cx.Network = "public"
	r := &Report{}
	verifySingleStellar(r, &cx, Options{})

	if step := onlyStep(t, r, groupStellar); step.Status != StatusFail {
		t.Errorf("status: got %v, want fail (%s)", step.Status, step.Message)
	}
	if r.Passed() {
		t.Error("a transaction absent from the network the entry named must fail the proof")
	}
}

// E.22: a non-event must not read as a positive assertion. The
// --skip-external commitment rows say they were skipped, and their
// status has to agree — as a `pass` each becomes a blockchain
// confirmation the run never made, and an additive pass row D.4 forbids.
func TestSkipExternal_CommitmentRowsAreSkips(t *testing.T) {
	report, err := Run(appendixDFixture(t), Options{SkipExternal: true, ExpectedHash: appendixDClaimsHash})
	if err != nil {
		t.Fatalf("Run: %s", err)
	}
	for _, group := range []string{groupStellar, groupBitcoin, groupKeyBinding} {
		for _, s := range stepsIn(report, group) {
			if s.Status != StatusSkip && s.Status != StatusInfo {
				t.Errorf("%s: status %v on a run that made no network call (%s)", group, s.Status, s.Message)
			}
		}
	}

	// Same rule on the Bitcoin path that carries offline evidence: the
	// binding row is the only one that may confirm, and --skip-external
	// takes it out of play.
	cx := btcSample()
	cx.Network = "mainnet"
	r := &Report{}
	verifySingleBitcoin(r, &cx, Options{SkipExternal: true})
	for _, s := range stepsIn(r, groupBitcoin) {
		if s.Status == StatusPass && strings.Contains(s.Message, "skipped") {
			t.Errorf("a row that says it was skipped must not be a pass: %s", s.Message)
		}
	}
	if ci := onlyCommitment(t, r); ci.ExternalCheck == ExternalConfirmed {
		t.Error("--skip-external cannot confirm a commitment")
	}
}

// The entropy verifiers and the Bitcoin binding step carry no
// "disagrees" arm, because only VerifyStellar and VerifyKeyring build an
// error Classify grades as a disagreement. This pins that invariant: if
// a fetcher ever starts grading one itself, the removed arms have to
// come back rather than the outcome falling silently into `default`.
func TestFetchers_NeverReportADisagreement(t *testing.T) {
	bodies := []string{`{}`, `null`, `{"foo":1}`, `[]`, `not json`, `{"id":"deadbeef"}`}
	for _, body := range bodies {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			_, _ = w.Write([]byte(body))
		}))

		origNIST, origPub, origTest, origBTC := external.NISTBeaconURL, external.HorizonPublicURL, external.HorizonTestnetURL, external.BlockstreamMainnetURL
		external.NISTBeaconURL, external.HorizonPublicURL, external.HorizonTestnetURL, external.BlockstreamMainnetURL = srv.URL, srv.URL, srv.URL, srv.URL

		_, nistErr := external.GetNISTPulse(1, 1)
		_, ledgerErr := external.GetStellarLedger(1, "public")
		_, _, headerErr := external.GetBitcoinBlockHeader(strings.Repeat("ab", 32), "mainnet")
		_, _, blockErr := external.VerifyBitcoinBlock(strings.Repeat("ab", 32), "mainnet")

		external.NISTBeaconURL, external.HorizonPublicURL, external.HorizonTestnetURL, external.BlockstreamMainnetURL = origNIST, origPub, origTest, origBTC
		srv.Close()

		for name, err := range map[string]error{
			"GetNISTPulse": nistErr, "GetStellarLedger": ledgerErr,
			"GetBitcoinBlockHeader": headerErr, "VerifyBitcoinBlock": blockErr,
		} {
			if external.Classify(err) == external.OutcomeMismatch {
				t.Errorf("%s graded body %q as a disagreement; the removed fail arm is reachable again", name, body)
			}
		}
	}
}

// --- E.4 lowercase-hex enforcement -----------------------------------------

// upperFirstHexByte returns s with its first lowercase a-f character
// uppercased, leaving a value that decodes to identical bytes under Go's
// case-insensitive hex.DecodeString. That equivalence is the whole point:
// every mutation below leaves the signature, the Merkle roots and every
// derived digest untouched, so nothing except an explicit encoding rule can
// tell the mutant from the original.
func upperFirstHexByte(t *testing.T, s string) string {
	t.Helper()
	for i := range len(s) {
		if s[i] >= 'a' && s[i] <= 'f' {
			return s[:i] + strings.ToUpper(string(s[i])) + s[i+1:]
		}
	}
	t.Fatalf("no lowercase hex letter to uppercase in %q", s)
	return ""
}

// TestAppendixD_UppercaseHexFieldsAreRejected covers all ten hex fields
// Appendix E.4 enumerates, one bundle per field, and asserts the three
// things the amended appendix makes MUSTs: the run is graded rather than
// aborted, the failure carries E.23's `invalid_hex_encoding` identifier,
// and it names the offending wire field.
//
// Before this enforcement all ten verified at exit 0, which meant two
// things. Byte-distinct spellings of the same bundle — at least 2^10 of
// them for the Appendix D vector — all verified under a single signature.
// And the CLI disagreed with the normative reference verifier, whose
// Base.decode16lower!/1 aborted on eight of these ten (it never reads tx),
// so "verified here, rejected there" was a live interoperability break on
// the one bundle E.25 asks a new verifier to self-certify against. That
// asymmetry is now closed from both sides: truestamp-v2 commit 4b1beaff0d
// made all three of its verifiers grade the same ten fields.
func TestAppendixD_UppercaseHexFieldsAreRejected(t *testing.T) {
	raw := readFileT(t, appendixDFixture(t))

	for _, tc := range []struct {
		name string
		// mutate uppercases exactly one hex field in the decoded bundle.
		mutate func(t *testing.T, m map[string]any)
		// wantField is the wire key the report must name.
		wantField string
		// pointOfUse is the group whose step consumes the field and must
		// fail on its own, in addition to E.4's Structure sweep. Empty for
		// cx[].tx and cx[].bmr, which no verifier decodes — they are
		// string-compared against values derived from rtx and txp, which is
		// exactly why E.4 requires the sweep to reach them.
		pointOfUse string
	}{
		{"s.mh", func(t *testing.T, m map[string]any) {
			s := m["s"].(map[string]any)
			s["mh"] = upperFirstHexByte(t, s["mh"].(string))
		}, "s.mh", groupSubjectData},
		{"s.kid", func(t *testing.T, m map[string]any) {
			s := m["s"].(map[string]any)
			s["kid"] = upperFirstHexByte(t, s["kid"].(string))
		}, "s.kid", groupSubjectData},
		{"b.ph", func(t *testing.T, m map[string]any) {
			b := m["b"].(map[string]any)
			b["ph"] = upperFirstHexByte(t, b["ph"].(string))
		}, "b.ph", groupBlockHash},
		{"b.mr", func(t *testing.T, m map[string]any) {
			b := m["b"].(map[string]any)
			b["mr"] = upperFirstHexByte(t, b["mr"].(string))
		}, "b.mr", groupBlockHash},
		{"b.mh", func(t *testing.T, m map[string]any) {
			b := m["b"].(map[string]any)
			b["mh"] = upperFirstHexByte(t, b["mh"].(string))
		}, "b.mh", groupBlockHash},
		{"b.kid", func(t *testing.T, m map[string]any) {
			b := m["b"].(map[string]any)
			b["kid"] = upperFirstHexByte(t, b["kid"].(string))
		}, "b.kid", groupBlockHash},
		{"cx[0].memo", func(t *testing.T, m map[string]any) {
			cx := m["cx"].([]any)[0].(map[string]any)
			cx["memo"] = upperFirstHexByte(t, cx["memo"].(string))
		}, "cx[0].memo", groupEpoch},
		{"cx[0].tx", func(t *testing.T, m map[string]any) {
			cx := m["cx"].([]any)[0].(map[string]any)
			cx["tx"] = upperFirstHexByte(t, cx["tx"].(string))
		}, "cx[0].tx", ""},
		{"cx[1].op", func(t *testing.T, m map[string]any) {
			cx := m["cx"].([]any)[1].(map[string]any)
			cx["op"] = upperFirstHexByte(t, cx["op"].(string))
		}, "cx[1].op", groupEpoch},
		{"cx[1].tx", func(t *testing.T, m map[string]any) {
			cx := m["cx"].([]any)[1].(map[string]any)
			cx["tx"] = upperFirstHexByte(t, cx["tx"].(string))
		}, "cx[1].tx", ""},
		// The Appendix D Bitcoin entry carries no bmr — the offline payload
		// is optional (E.5) — so the field is injected before being
		// uppercased. Without this case the tenth field of E.4's closed set
		// would be covered by no test at all.
		{"cx[1].bmr", func(t *testing.T, m map[string]any) {
			cx := m["cx"].([]any)[1].(map[string]any)
			if _, ok := cx["bmr"]; ok {
				t.Fatal("fixture now carries bmr; drop the injection and uppercase in place")
			}
			cx["bmr"] = strings.Repeat("aB", 32)
		}, "cx[1].bmr", ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var m map[string]any
			if err := json.Unmarshal(raw, &m); err != nil {
				t.Fatalf("unmarshal fixture: %v", err)
			}
			tc.mutate(t, m)
			mutated, err := json.Marshal(m)
			if err != nil {
				t.Fatalf("marshal mutant: %v", err)
			}

			report, err := RunFromBytes(mutated, "appendix-d-upper.json", Options{
				SkipExternal: true,
				ExpectedHash: appendixDClaimsHash,
			})
			if err != nil {
				// E.6 gates on the presence and type of structure, never on
				// the lexical form of a value, so this must stay a graded
				// report rather than becoming an abort.
				t.Fatalf("uppercase %s aborted the run; E.6 authorizes no rejection for hex case: %v", tc.wantField, err)
			}
			if report.Passed() {
				t.Fatalf("uppercase %s still verifies:\n%s", tc.wantField, formatSteps(report.Steps))
			}

			// E.4 requires the Structure sweep to name every offender.
			sweep := encodingFailIn(report, groupStructure, tc.wantField)
			if sweep == nil {
				t.Errorf("E.4's Structure sweep does not name %s:\n%s",
					tc.wantField, formatSteps(report.Steps))
			}

			// E.4 requires the consuming step to fail as well, so an
			// encoding defect is never reported as a generic mismatch.
			if tc.pointOfUse != "" && encodingFailIn(report, tc.pointOfUse, tc.wantField) == nil {
				t.Errorf("%s does not fail at its point of use naming %s; an encoding defect would surface as a generic mismatch:\n%s",
					tc.pointOfUse, tc.wantField, formatSteps(report.Steps))
			}
		})
	}
}

// encodingFailIn returns the failing step in group that carries E.23's
// invalid_hex_encoding identifier and names field, or nil.
//
// The identifier is checked rather than the prose because E.23 exists so
// two independent verifiers can be compared on an identifier instead of on
// wording that differs between them; asserting only the English would let
// the MUST regress silently.
func encodingFailIn(report *Report, group, field string) *Step {
	for i, s := range report.Steps {
		if s.Status != StatusFail || s.Group != group {
			continue
		}
		if strings.HasPrefix(s.Message, codeInvalidHexEncoding+": ") && strings.Contains(s.Message, field) {
			return &report.Steps[i]
		}
	}
	return nil
}

// TestHexEncodingExclusionsAreNotGraded pins the other half of E.4's closed
// set: the fields a verifier MUST NOT case-grade. Each mutation below is
// either inert or caught by the cryptography on its own, and none of them
// may produce an encoding row.
//
// Getting this wrong in the strict direction is the expensive mistake. A
// lowercase rule on rtx/txp is not merely over-strict but undefined — E.3
// files both as text carrying either base64url or hex, and the hex alphabet
// is a subset of the base64url one — and a rule on s.d values would derive a
// different 0x11 digest and report a valid proof as forged.
func TestHexEncodingExclusionsAreNotGraded(t *testing.T) {
	raw := readFileT(t, appendixDFixture(t))

	for _, tc := range []struct {
		name   string
		mutate func(t *testing.T, m map[string]any)
	}{
		{"s.id", func(t *testing.T, m map[string]any) {
			s := m["s"].(map[string]any)
			s["id"] = strings.ToUpper(s["id"].(string))
		}},
		{"b.id", func(t *testing.T, m map[string]any) {
			b := m["b"].(map[string]any)
			b["id"] = strings.ToUpper(b["id"].(string))
		}},
		{"ts", func(t *testing.T, m map[string]any) {
			m["ts"] = strings.ToUpper(m["ts"].(string))
		}},
		{"cx[].net", func(t *testing.T, m map[string]any) {
			for _, e := range m["cx"].([]any) {
				c := e.(map[string]any)
				if v, ok := c["net"].(string); ok {
					c["net"] = strings.ToUpper(v)
				}
			}
		}},
		{"s.d.hash", func(t *testing.T, m map[string]any) {
			d := m["s"].(map[string]any)["d"].(map[string]any)
			d["hash"] = strings.ToUpper(d["hash"].(string))
		}},
		{"cx[].rtx", func(t *testing.T, m map[string]any) {
			for _, e := range m["cx"].([]any) {
				c := e.(map[string]any)
				if _, ok := c["rtx"]; ok {
					t.Fatal("fixture now carries rtx; uppercase it in place instead of injecting")
				}
				c["rtx"] = strings.Repeat("aB", 16)
			}
		}},
		{"cx[].txp", func(t *testing.T, m map[string]any) {
			for _, e := range m["cx"].([]any) {
				c := e.(map[string]any)
				if _, ok := c["txp"]; ok {
					t.Fatal("fixture now carries txp; uppercase it in place instead of injecting")
				}
				c["txp"] = strings.Repeat("aB", 16)
			}
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var m map[string]any
			if err := json.Unmarshal(raw, &m); err != nil {
				t.Fatalf("unmarshal fixture: %v", err)
			}
			tc.mutate(t, m)
			mutated, err := json.Marshal(m)
			if err != nil {
				t.Fatalf("marshal mutant: %v", err)
			}

			report, err := RunFromBytes(mutated, "appendix-d-excluded.json", Options{
				SkipExternal: true,
				ExpectedHash: appendixDClaimsHash,
			})
			if err != nil {
				t.Fatalf("uppercase %s aborted the run: %v", tc.name, err)
			}

			for _, s := range report.Steps {
				if strings.HasPrefix(s.Message, codeInvalidHexEncoding+": ") {
					t.Errorf("uppercase %s produced an encoding row, but E.4 rules it out of scope: [%s] %s",
						tc.name, s.Group, s.Message)
				}
			}
		})
	}
}

// TestUppercaseRootDoesNotPassTheWalk is the honesty half of the fix. The
// E.13 and E.15 comparisons fold ASCII case, so a b.mr or cx[].memo spelled
// in uppercase derives the same root and those steps used to report `pass`
// — asserting a match against a value the bundle does not carry in E.4's
// encoding, and disagreeing with the reference verifier, whose
// secure_equal?/2 is a raw binary compare and grades the same bundle a
// failure at exactly those two rows.
func TestUppercaseRootDoesNotPassTheWalk(t *testing.T) {
	raw := readFileT(t, appendixDFixture(t))

	for _, tc := range []struct {
		name   string
		mutate func(t *testing.T, m map[string]any)
		group  string
		// row selects the one step in the group that reads the mutated
		// field. The Epoch Proof group carries a row per cx entry, and the
		// unmutated entry still passes on its own merits.
		row string
	}{
		{"b.mr", func(t *testing.T, m map[string]any) {
			b := m["b"].(map[string]any)
			b["mr"] = upperFirstHexByte(t, b["mr"].(string))
		}, groupInclusion, "Inclusion proof"},
		{"cx[0].memo", func(t *testing.T, m map[string]any) {
			cx := m["cx"].([]any)[0].(map[string]any)
			cx["memo"] = upperFirstHexByte(t, cx["memo"].(string))
		}, groupEpoch, "Epoch proof 0"},
		{"cx[1].op", func(t *testing.T, m map[string]any) {
			cx := m["cx"].([]any)[1].(map[string]any)
			cx["op"] = upperFirstHexByte(t, cx["op"].(string))
		}, groupEpoch, "Epoch proof 1"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var m map[string]any
			if err := json.Unmarshal(raw, &m); err != nil {
				t.Fatalf("unmarshal fixture: %v", err)
			}
			tc.mutate(t, m)
			mutated, err := json.Marshal(m)
			if err != nil {
				t.Fatalf("marshal mutant: %v", err)
			}
			report, err := RunFromBytes(mutated, "appendix-d-upper.json", Options{SkipExternal: true})
			if err != nil {
				t.Fatalf("RunFromBytes: %v", err)
			}
			for _, s := range stepsIn(report, tc.group) {
				if strings.HasPrefix(s.Message, tc.row) && s.Status == StatusPass {
					t.Errorf("%s reports a pass over a non-lowercase root: %s", tc.group, s.Message)
				}
			}
		})
	}
}

// TestConformingBundleEmitsNoEncodingRow keeps the E.4 check invisible on a
// conforming bundle. It is a Structural `fail` when it fires, and an extra
// verdict-moving row is exactly what E.25's containment forbids, so a
// regression that made it emit unconditionally would break D.4 conformance
// rather than merely add noise.
func TestConformingBundleEmitsNoEncodingRow(t *testing.T) {
	report, err := Run(appendixDFixture(t), Options{
		SkipExternal: true,
		ExpectedHash: appendixDClaimsHash,
	})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	for _, s := range stepsIn(report, groupStructure) {
		if strings.Contains(s.Message, "lowercase-hex") {
			t.Errorf("conforming bundle emitted an encoding row: %s", s.Message)
		}
	}
}
