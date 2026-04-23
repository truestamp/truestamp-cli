// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: MIT

package verify

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/truestamp/truestamp-cli/internal/proof"
	"github.com/truestamp/truestamp-cli/internal/proof/ptype"
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

func TestVerifySigningKey_InvalidPublicKey(t *testing.T) {
	r := &Report{}
	bundle := &proof.ProofBundle{
		PublicKey: "not-valid-base64!!!",
	}
	pubkey, keyID := verifySigningKey(r, bundle, Options{SkipExternal: true})
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
	bundle := &proof.ProofBundle{
		PublicKey: "CTwMqDZnPd/QTLSq8aTeSD3a+j2DQxKcGfhhIYJQ65Y=",
	}
	pubkey, keyID := verifySigningKey(r, bundle, Options{SkipExternal: true})
	if pubkey == nil {
		t.Error("valid key should return non-nil pubkey")
	}
	if keyID != "4ceefa4a" {
		t.Errorf("expected keyID 4ceefa4a, got %q", keyID)
	}
	hasPass := false
	for _, s := range r.Steps {
		if s.Status == StatusPass && containsStr(s.Message, "Public key valid") {
			hasPass = true
		}
	}
	if !hasPass {
		t.Error("valid key should produce a pass step")
	}
}

func TestVerifySigningKey_ExternalFails(t *testing.T) {
	r := &Report{}
	bundle := &proof.ProofBundle{
		PublicKey: "CTwMqDZnPd/QTLSq8aTeSD3a+j2DQxKcGfhhIYJQ65Y=",
	}
	verifySigningKey(r, bundle, Options{
		SkipExternal: false,
		KeyringURL:   "http://127.0.0.1:1/nonexistent",
	})
	hasFail := false
	for _, s := range r.Steps {
		if s.Status == StatusFail && containsStr(s.Message, "eyring") {
			hasFail = true
		}
	}
	if !hasFail {
		t.Error("unreachable keyring should produce a failure")
	}
}

func TestVerifySigningKey_ExternalSuccess(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]interface{}{
			"version": "1.0",
			"keys": []map[string]interface{}{
				{"key_id": "4ceefa4a", "public_key": "CTwMqDZnPd/QTLSq8aTeSD3a+j2DQxKcGfhhIYJQ65Y=", "sequence": 0, "active": true},
			},
		})
	}))
	defer server.Close()

	r := &Report{}
	bundle := &proof.ProofBundle{
		PublicKey: "CTwMqDZnPd/QTLSq8aTeSD3a+j2DQxKcGfhhIYJQ65Y=",
	}
	verifySigningKey(r, bundle, Options{
		SkipExternal: false,
		KeyringURL:   server.URL,
	})
	hasKeyringPass := false
	for _, s := range r.Steps {
		if s.Status == StatusPass && containsStr(s.Message, "confirmed via keyring") {
			hasKeyringPass = true
		}
	}
	if !hasKeyringPass {
		t.Error("valid keyring response should produce a keyring pass step")
	}
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
		if s.Status == StatusFail && containsStr(s.Message, "missing block hash") {
			hasFail = true
		}
	}
	if !hasFail {
		t.Error("empty block hash should produce a failure")
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

func TestDeriveClaimsHash_RedactedClaims(t *testing.T) {
	r := &Report{}
	hash := deriveClaimsHash(r, []byte("null"))
	if hash != "" {
		t.Error("null claims should return empty hash")
	}
	hasSkip := false
	for _, s := range r.Steps {
		if s.Status == StatusSkip && containsStr(s.Message, "redacted") {
			hasSkip = true
		}
	}
	if !hasSkip {
		t.Error("null claims should produce a skip step")
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

func TestDeriveItemHash_MissingInputs(t *testing.T) {
	r := &Report{}
	subject := &proof.Subject{
		ID:           "test",
		MetadataHash: "",
		SigningKeyID: "4ceefa4a",
	}
	hash := deriveItemHash(r, subject, "")
	if hash != "" {
		t.Error("missing inputs should return empty hash")
	}
	hasFail := false
	for _, s := range r.Steps {
		if s.Status == StatusFail && containsStr(s.Message, "missing inputs") {
			hasFail = true
		}
	}
	if !hasFail {
		t.Error("missing inputs should produce a failure")
	}
}

func TestVerifyStructure_EmptyBlock(t *testing.T) {
	r := &Report{}
	bundle := &proof.ProofBundle{T: ptype.Item, Commitments: []proof.ExternalCommit{{Type: ptype.CommitmentStellar}}}
	verifyStructure(r, bundle, proof.Block{})
	hasFail := false
	for _, s := range r.Steps {
		if s.Status == StatusFail {
			hasFail = true
		}
	}
	if !hasFail {
		t.Error("empty block should fail structure check")
	}
}

func TestVerifyStructure_ValidBlock(t *testing.T) {
	r := &Report{}
	bundle := &proof.ProofBundle{T: ptype.Item, Commitments: []proof.ExternalCommit{{Type: ptype.CommitmentStellar}}}
	verifyStructure(r, bundle, proof.Block{ID: "b1", MerkleRoot: "aaaa"})
	hasFail := false
	for _, s := range r.Steps {
		if s.Status == StatusFail {
			hasFail = true
		}
	}
	if hasFail {
		t.Error("valid block should not fail structure check")
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

	verifyBitcoinCommitments(r, commits, "fakeblockHash", Options{SkipExternal: true})

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

	verifyBitcoinCommitments(r, commits, "fakeblockHash", Options{SkipExternal: true})
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

	verifyBitcoinCommitments(r, commits, "fakeblockHash", Options{SkipExternal: false})
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

func TestRunFromBytes_HashWithEntropyProof_FailsInReport(t *testing.T) {
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
		SkipExternal: true,
		ExpectedHash: "deadbeef",
	})
	if err != nil {
		t.Fatalf("expected no error (failures should be in report), got: %s", err)
	}
	if report == nil {
		t.Fatal("expected a report, got nil")
	}
	if report.Passed() {
		t.Error("report should not pass when --hash used with entropy proof")
	}
	found := false
	for _, s := range report.Steps {
		if s.Status == StatusFail && strings.Contains(s.Message, "not applicable to entropy") {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected a fail step about --hash not applicable to entropy proofs")
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
