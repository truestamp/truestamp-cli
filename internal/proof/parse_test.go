// Copyright (c) 2021-2026 Truestamp, Inc.
// SPDX-License-Identifier: MIT

package proof

import (
	"os"
	"path/filepath"
	"testing"
)

// Minimal valid proof JSON for testing (compact format)
const validProofJSON = `{
  "v": 1,
  "pk": "CTwMqDZnPd/QTLSq8aTeSD3a+j2DQxKcGfhhIYJQ65Y=",
  "sig": "c2lnbmF0dXJlX2J5dGVzX2hlcmU=",
  "ts": "2026-04-06T23:25:06Z",
  "s": {
    "src": "item",
    "id": "01HJHB01T8FYZ7YTR9P5N62K5B",
    "d": {"name": "test"},
    "mh": "ccddccddccddccddccddccddccddccddccddccddccddccddccddccddccddccdd",
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
      "t": "stellar",
      "net": "testnet",
      "tx": "aaaa",
      "memo": "bbbb",
      "l": 100,
      "ts": "2026-04-06T23:25:06Z",
      "ep": "AA"
    }
  ]
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

func TestParse_ValidProof(t *testing.T) {
	path := writeTemp(t, validProofJSON)
	bundle, err := Parse(path)
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}

	if bundle.Version != 1 {
		t.Errorf("version: got %d, want 1", bundle.Version)
	}
	if bundle.PublicKey != "CTwMqDZnPd/QTLSq8aTeSD3a+j2DQxKcGfhhIYJQ65Y=" {
		t.Errorf("public_key: got %s", bundle.PublicKey)
	}
	if bundle.Signature != "c2lnbmF0dXJlX2J5dGVzX2hlcmU=" {
		t.Errorf("signature: got %s", bundle.Signature)
	}
	if bundle.Timestamp != "2026-04-06T23:25:06Z" {
		t.Errorf("timestamp: got %s", bundle.Timestamp)
	}
	if bundle.Subject.ID != "01HJHB01T8FYZ7YTR9P5N62K5B" {
		t.Errorf("subject.id: got %s, want 01HJHB01T8FYZ7YTR9P5N62K5B", bundle.Subject.ID)
	}
	if bundle.Subject.Source != "item" {
		t.Errorf("subject.source: got %s, want item", bundle.Subject.Source)
	}
	if bundle.Subject.MetadataHash != "ccddccddccddccddccddccddccddccddccddccddccddccddccddccddccddccdd" {
		t.Errorf("subject.metadata_hash: got %s", bundle.Subject.MetadataHash)
	}
	if bundle.Subject.SigningKeyID != "4ceefa4a" {
		t.Errorf("subject.signing_key_id: got %s", bundle.Subject.SigningKeyID)
	}
	if bundle.Block.ID != "019cf813-99b8-730a-84f1-5a711a9c355e" {
		t.Errorf("block.id: got %s", bundle.Block.ID)
	}
	if bundle.Block.PreviousBlockHash != "1111111111111111111111111111111111111111111111111111111111111111" {
		t.Errorf("block.ph: got %s", bundle.Block.PreviousBlockHash)
	}
	if bundle.Block.MerkleRoot != "2222222222222222222222222222222222222222222222222222222222222222" {
		t.Errorf("block.mr: got %s", bundle.Block.MerkleRoot)
	}
	if bundle.InclusionProof != "AA" {
		t.Errorf("ip: got %s", bundle.InclusionProof)
	}
	if len(bundle.Commitments) != 1 {
		t.Errorf("commitments length: got %d, want 1", len(bundle.Commitments))
	}
	if bundle.Commitments[0].Type != "stellar" {
		t.Errorf("commitments[0].type: got %s", bundle.Commitments[0].Type)
	}
	if bundle.Commitments[0].TransactionHash != "aaaa" {
		t.Errorf("commitments[0].tx: got %s", bundle.Commitments[0].TransactionHash)
	}
}

func TestParse_PreservesRawData(t *testing.T) {
	path := writeTemp(t, validProofJSON)
	bundle, err := Parse(path)
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}

	// RawData should contain the original data JSON bytes
	if len(bundle.RawData) == 0 {
		t.Error("RawData should not be empty")
	}

	// Subject data should be preserved as raw JSON
	if len(bundle.Subject.Data) == 0 {
		t.Error("subject.Data should be preserved as raw JSON")
	}
}

func TestParse_FileNotFound(t *testing.T) {
	_, err := Parse("/nonexistent/path/proof.json")
	if err == nil {
		t.Error("expected error for missing file")
	}
}

func TestParse_InvalidJSON(t *testing.T) {
	path := writeTemp(t, "not json at all")
	_, err := Parse(path)
	if err == nil {
		t.Error("expected error for invalid JSON")
	}
}

func TestParse_MissingVersion(t *testing.T) {
	path := writeTemp(t, `{
		"pk": "aa", "sig": "bb", "ts": "2026-01-01T00:00:00Z",
		"s": {"src":"item","id":"x","d":{},"mh":"cc","kid":"dd"},
		"b": {"id":"e","mr":"f","mh":"g","kid":"h"},
		"ip": "AA"
	}`)
	_, err := Parse(path)
	if err == nil {
		t.Error("expected error for missing version")
	}
}

func TestParse_MissingPublicKey(t *testing.T) {
	path := writeTemp(t, `{
		"v": 1, "sig": "bb", "ts": "2026-01-01T00:00:00Z",
		"s": {"src":"item","id":"x","d":{},"mh":"cc","kid":"dd"},
		"b": {"id":"e","mr":"f","mh":"g","kid":"h"},
		"ip": "AA"
	}`)
	_, err := Parse(path)
	if err == nil {
		t.Error("expected error for missing public_key")
	}
}

func TestParse_MissingSignature(t *testing.T) {
	path := writeTemp(t, `{
		"v": 1, "pk": "aa", "ts": "2026-01-01T00:00:00Z",
		"s": {"src":"item","id":"x","d":{},"mh":"cc","kid":"dd"},
		"b": {"id":"e","mr":"f","mh":"g","kid":"h"},
		"ip": "AA"
	}`)
	_, err := Parse(path)
	if err == nil {
		t.Error("expected error for missing signature")
	}
}

func TestParse_MissingSubject(t *testing.T) {
	path := writeTemp(t, `{
		"v": 1, "pk": "aa", "sig": "bb", "ts": "2026-01-01T00:00:00Z",
		"b": {"id":"e","mr":"f","mh":"g","kid":"h"},
		"ip": "AA"
	}`)
	_, err := Parse(path)
	if err == nil {
		t.Error("expected error for missing subject")
	}
}

func TestParse_MissingBlock(t *testing.T) {
	path := writeTemp(t, `{
		"v": 1, "pk": "aa", "sig": "bb", "ts": "2026-01-01T00:00:00Z",
		"s": {"src":"item","id":"x","d":{},"mh":"cc","kid":"dd"},
		"ip": "AA"
	}`)
	_, err := Parse(path)
	if err == nil {
		t.Error("expected error for missing block")
	}
}

func TestParse_InvalidSubjectStructure(t *testing.T) {
	path := writeTemp(t, `{
		"v": 1, "pk": "aa", "sig": "bb", "ts": "2026-01-01T00:00:00Z",
		"s": "not an object",
		"b": {"id":"e","mr":"f","mh":"g","kid":"h"},
		"ip": "AA"
	}`)
	_, err := Parse(path)
	if err == nil {
		t.Error("expected error for invalid subject structure")
	}
}

func TestParse_NullCommitments(t *testing.T) {
	path := writeTemp(t, `{
		"v": 1, "pk": "aa", "sig": "bb", "ts": "2026-01-01T00:00:00Z",
		"s": {"src":"item","id":"x","d":{},"mh":"cc","kid":"dd"},
		"b": {"id":"e","mr":"f","mh":"g","kid":"h"},
		"ip": "AA",
		"cx": null
	}`)
	bundle, err := Parse(path)
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	if len(bundle.Commitments) != 0 {
		t.Errorf("commitments should be empty for null, got %d", len(bundle.Commitments))
	}
}

func TestParse_OmittedCommitments(t *testing.T) {
	path := writeTemp(t, `{
		"v": 1, "pk": "aa", "sig": "bb", "ts": "2026-01-01T00:00:00Z",
		"s": {"src":"item","id":"x","d":{},"mh":"cc","kid":"dd"},
		"b": {"id":"e","mr":"f","mh":"g","kid":"h"},
		"ip": "AA"
	}`)
	bundle, err := Parse(path)
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	if len(bundle.Commitments) != 0 {
		t.Errorf("commitments should be empty when omitted, got %d", len(bundle.Commitments))
	}
}

func TestParseBytes_Valid(t *testing.T) {
	bundle, err := ParseBytes([]byte(validProofJSON))
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	if bundle.Version != 1 {
		t.Errorf("version: got %d, want 1", bundle.Version)
	}
	if bundle.Subject.ID != "01HJHB01T8FYZ7YTR9P5N62K5B" {
		t.Errorf("subject.id: got %s, want 01HJHB01T8FYZ7YTR9P5N62K5B", bundle.Subject.ID)
	}
	if bundle.PublicKey == "" {
		t.Error("public_key should not be empty")
	}
	if bundle.Signature == "" {
		t.Error("signature should not be empty")
	}
}

func TestParseBytes_InvalidJSON(t *testing.T) {
	_, err := ParseBytes([]byte("not json"))
	if err == nil {
		t.Error("expected error for invalid JSON")
	}
}

func TestParseBytes_MissingVersion(t *testing.T) {
	_, err := ParseBytes([]byte(`{
		"pk": "aa", "sig": "bb", "ts": "2026-01-01T00:00:00Z",
		"s": {"src":"item","id":"x","d":{},"mh":"cc","kid":"dd"},
		"b": {"id":"e","mr":"f","mh":"g","kid":"h"},
		"ip": "AA"
	}`))
	if err == nil {
		t.Error("expected error for missing version")
	}
}

func TestParseBytes_MissingPublicKey(t *testing.T) {
	_, err := ParseBytes([]byte(`{
		"v": 1, "sig": "bb", "ts": "2026-01-01T00:00:00Z",
		"s": {"src":"item","id":"x","d":{},"mh":"cc","kid":"dd"},
		"b": {"id":"e","mr":"f","mh":"g","kid":"h"},
		"ip": "AA"
	}`))
	if err == nil {
		t.Error("expected error for missing public_key")
	}
}

func TestFileSizeFromData(t *testing.T) {
	data := []byte("hello world")
	if got := FileSizeFromData(data); got != 11 {
		t.Errorf("FileSizeFromData: got %d, want 11", got)
	}
}

func TestFileSizeFromData_Empty(t *testing.T) {
	if got := FileSizeFromData(nil); got != 0 {
		t.Errorf("FileSizeFromData(nil): got %d, want 0", got)
	}
}

func TestFileSize_Exists(t *testing.T) {
	path := writeTemp(t, "hello")
	size := FileSize(path)
	if size != 5 {
		t.Errorf("file size: got %d, want 5", size)
	}
}

func TestFileSize_NotExists(t *testing.T) {
	size := FileSize("/nonexistent/file")
	if size != 0 {
		t.Errorf("file size for missing file: got %d, want 0", size)
	}
}

// Minimal valid entropy proof JSON for testing (compact format)
const validEntropyProofJSON = `{
  "v": 1,
  "pk": "CTwMqDZnPd/QTLSq8aTeSD3a+j2DQxKcGfhhIYJQ65Y=",
  "sig": "c2lnbmF0dXJlX2J5dGVzX2hlcmU=",
  "ts": "2026-04-06T23:25:06Z",
  "s": {
    "src": "nist_beacon",
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
  "cx": []
}`

func TestParse_EntropyProof(t *testing.T) {
	path := writeTemp(t, validEntropyProofJSON)
	bundle, err := Parse(path)
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}

	if bundle.Type != "entropy" {
		t.Errorf("type: got %s, want entropy", bundle.Type)
	}
	if bundle.Timestamp != "2026-04-06T23:25:06Z" {
		t.Errorf("timestamp: got %s", bundle.Timestamp)
	}
	if !bundle.IsEntropy() {
		t.Error("IsEntropy() should return true")
	}
	if bundle.Subject.ID != "019d2ae3-865c-7651-9923-b14c55bc8e33" {
		t.Errorf("subject.id: got %s", bundle.Subject.ID)
	}
	if bundle.Subject.Source != "nist_beacon" {
		t.Errorf("subject.source: got %s", bundle.Subject.Source)
	}
	if bundle.Subject.SigningKeyID != "4ceefa4a" {
		t.Errorf("subject.signing_key_id: got %s", bundle.Subject.SigningKeyID)
	}
	if len(bundle.RawData) == 0 {
		t.Error("RawData should be populated")
	}
}

func TestParse_ItemProofHasType(t *testing.T) {
	bundle, err := ParseBytes([]byte(validProofJSON))
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	if bundle.Type != "item" {
		t.Errorf("type: got %s, want item", bundle.Type)
	}
	if bundle.IsEntropy() {
		t.Error("IsEntropy() should return false for item proofs")
	}
}
