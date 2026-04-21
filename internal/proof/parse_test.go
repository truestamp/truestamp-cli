// Copyright (c) 2021-2026 Truestamp, Inc.
// SPDX-License-Identifier: MIT

package proof

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/truestamp/truestamp-cli/internal/proof/ptype"
)

// Minimal valid proof JSON for testing (t=20, item subject).
const validProofJSON = `{
  "v": 1,
  "t": 20,
  "pk": "CTwMqDZnPd/QTLSq8aTeSD3a+j2DQxKcGfhhIYJQ65Y=",
  "sig": "AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA==",
  "ts": "2026-04-06T23:25:06Z",
  "s": {
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
      "t": 40,
      "net": "testnet",
      "tx": "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
      "memo": "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
      "l": 100,
      "ts": "2026-04-06T23:25:06Z",
      "ep": "AA"
    }
  ]
}`

// Minimal valid entropy proof JSON (t=30, nist entropy).
const validEntropyProofJSON = `{
  "v": 1,
  "t": 30,
  "pk": "CTwMqDZnPd/QTLSq8aTeSD3a+j2DQxKcGfhhIYJQ65Y=",
  "sig": "AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA==",
  "ts": "2026-04-06T23:25:06Z",
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

// Minimal valid block proof JSON (t=10, no s, no ip).
const validBlockProofJSON = `{
  "v": 1,
  "t": 10,
  "pk": "CTwMqDZnPd/QTLSq8aTeSD3a+j2DQxKcGfhhIYJQ65Y=",
  "sig": "AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA==",
  "ts": "2026-04-06T23:25:06Z",
  "b": {
    "id": "019cf813-99b8-730a-84f1-5a711a9c355e",
    "ph": "1111111111111111111111111111111111111111111111111111111111111111",
    "mr": "2222222222222222222222222222222222222222222222222222222222222222",
    "mh": "4444444444444444444444444444444444444444444444444444444444444444",
    "kid": "4ceefa4a"
  },
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

func TestParse_ValidItemProof(t *testing.T) {
	path := writeTemp(t, validProofJSON)
	bundle, err := Parse(path)
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}

	if bundle.Version != 1 {
		t.Errorf("version: got %d, want 1", bundle.Version)
	}
	if bundle.T != ptype.Item {
		t.Errorf("T: got %d, want %d", bundle.T, ptype.Item)
	}
	if !bundle.IsItem() || bundle.IsEntropy() || bundle.IsBlock() {
		t.Errorf("IsItem/IsEntropy/IsBlock: %v/%v/%v, want t/f/f", bundle.IsItem(), bundle.IsEntropy(), bundle.IsBlock())
	}
	if bundle.Subject == nil || bundle.Subject.ID != "01HJHB01T8FYZ7YTR9P5N62K5B" {
		t.Errorf("subject.id mismatch")
	}
	if bundle.Block.ID != "019cf813-99b8-730a-84f1-5a711a9c355e" {
		t.Errorf("block.id: got %s", bundle.Block.ID)
	}
	if bundle.InclusionProof != "AA" {
		t.Errorf("ip: got %s", bundle.InclusionProof)
	}
	if len(bundle.Commitments) != 1 {
		t.Fatalf("commitments length: got %d, want 1", len(bundle.Commitments))
	}
	if bundle.Commitments[0].Type != ptype.CommitmentStellar {
		t.Errorf("commitments[0].t: got %d, want %d", bundle.Commitments[0].Type, ptype.CommitmentStellar)
	}
}

func TestParse_PreservesRawData(t *testing.T) {
	path := writeTemp(t, validProofJSON)
	bundle, err := Parse(path)
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	if len(bundle.RawData) == 0 {
		t.Error("RawData should not be empty")
	}
	if bundle.Subject == nil || len(bundle.Subject.Data) == 0 {
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
	_, err := ParseBytes([]byte(`{
		"t": 20, "pk": "aa", "sig": "bb", "ts": "2026-01-01T00:00:00Z",
		"s": {"id":"x","d":{},"mh":"cc","kid":"dd"},
		"b": {"id":"e","mr":"f","mh":"g","kid":"h"},
		"ip": "AA",
		"cx": [{"t":40,"net":"testnet","ep":"AA"}]
	}`))
	if err == nil {
		t.Error("expected error for missing version")
	}
}

func TestParse_MissingTypeCode(t *testing.T) {
	_, err := ParseBytes([]byte(`{
		"v": 1, "pk": "aa", "sig": "bb", "ts": "2026-01-01T00:00:00Z",
		"s": {"id":"x","d":{},"mh":"cc","kid":"dd"},
		"b": {"id":"e","mr":"f","mh":"g","kid":"h"},
		"ip": "AA",
		"cx": [{"t":40,"net":"testnet","ep":"AA"}]
	}`))
	if err == nil {
		t.Error("expected error for missing t field")
	}
}

func TestParse_InvalidTypeCode(t *testing.T) {
	_, err := ParseBytes([]byte(`{
		"v": 1, "t": 99,
		"pk": "aa", "sig": "bb", "ts": "2026-01-01T00:00:00Z",
		"b": {"id":"e","mr":"f","mh":"g","kid":"h"},
		"cx": [{"t":40,"net":"testnet","ep":"AA"}]
	}`))
	if err == nil {
		t.Error("expected error for unknown t code")
	}
}

func TestParse_MissingPublicKey(t *testing.T) {
	_, err := ParseBytes([]byte(`{
		"v": 1, "t": 20,
		"sig": "bb", "ts": "2026-01-01T00:00:00Z",
		"s": {"id":"x","d":{},"mh":"cc","kid":"dd"},
		"b": {"id":"e","mr":"f","mh":"g","kid":"h"},
		"ip": "AA",
		"cx": [{"t":40,"net":"testnet","ep":"AA"}]
	}`))
	if err == nil {
		t.Error("expected error for missing pk")
	}
}

func TestParse_MissingSignature(t *testing.T) {
	_, err := ParseBytes([]byte(`{
		"v": 1, "t": 20,
		"pk": "aa", "ts": "2026-01-01T00:00:00Z",
		"s": {"id":"x","d":{},"mh":"cc","kid":"dd"},
		"b": {"id":"e","mr":"f","mh":"g","kid":"h"},
		"ip": "AA",
		"cx": [{"t":40,"net":"testnet","ep":"AA"}]
	}`))
	if err == nil {
		t.Error("expected error for missing sig")
	}
}

func TestParse_MissingSubjectForItem(t *testing.T) {
	_, err := ParseBytes([]byte(`{
		"v": 1, "t": 20,
		"pk": "aa", "sig": "bb", "ts": "2026-01-01T00:00:00Z",
		"b": {"id":"e","mr":"f","mh":"g","kid":"h"},
		"ip": "AA",
		"cx": [{"t":40,"net":"testnet","ep":"AA"}]
	}`))
	if err == nil {
		t.Error("expected error for missing s on item proof")
	}
}

func TestParse_MissingInclusionProofForItem(t *testing.T) {
	_, err := ParseBytes([]byte(`{
		"v": 1, "t": 20,
		"pk": "aa", "sig": "bb", "ts": "2026-01-01T00:00:00Z",
		"s": {"id":"x","d":{},"mh":"cc","kid":"dd"},
		"b": {"id":"e","mr":"f","mh":"g","kid":"h"},
		"cx": [{"t":40,"net":"testnet","ep":"AA"}]
	}`))
	if err == nil {
		t.Error("expected error for missing ip on item proof")
	}
}

func TestParse_MissingBlock(t *testing.T) {
	_, err := ParseBytes([]byte(`{
		"v": 1, "t": 20,
		"pk": "aa", "sig": "bb", "ts": "2026-01-01T00:00:00Z",
		"s": {"id":"x","d":{},"mh":"cc","kid":"dd"},
		"ip": "AA",
		"cx": [{"t":40,"net":"testnet","ep":"AA"}]
	}`))
	if err == nil {
		t.Error("expected error for missing block")
	}
}

func TestParse_EmptyCxRejected(t *testing.T) {
	_, err := ParseBytes([]byte(`{
		"v": 1, "t": 20,
		"pk": "aa", "sig": "bb", "ts": "2026-01-01T00:00:00Z",
		"s": {"id":"x","d":{},"mh":"cc","kid":"dd"},
		"b": {"id":"e","mr":"f","mh":"g","kid":"h"},
		"ip": "AA",
		"cx": []
	}`))
	if err == nil {
		t.Error("empty cx must be rejected")
	}
}

func TestParse_MissingCxRejected(t *testing.T) {
	_, err := ParseBytes([]byte(`{
		"v": 1, "t": 20,
		"pk": "aa", "sig": "bb", "ts": "2026-01-01T00:00:00Z",
		"s": {"id":"x","d":{},"mh":"cc","kid":"dd"},
		"b": {"id":"e","mr":"f","mh":"g","kid":"h"},
		"ip": "AA"
	}`))
	if err == nil {
		t.Error("missing cx must be rejected")
	}
}

func TestParse_UnknownCxCodeRejected(t *testing.T) {
	_, err := ParseBytes([]byte(`{
		"v": 1, "t": 20,
		"pk": "aa", "sig": "bb", "ts": "2026-01-01T00:00:00Z",
		"s": {"id":"x","d":{},"mh":"cc","kid":"dd"},
		"b": {"id":"e","mr":"f","mh":"g","kid":"h"},
		"ip": "AA",
		"cx": [{"t": 99, "net": "testnet", "ep": "AA"}]
	}`))
	if err == nil {
		t.Error("unknown cx type code must be rejected")
	}
}

func TestParse_InvalidSubjectStructure(t *testing.T) {
	_, err := ParseBytes([]byte(`{
		"v": 1, "t": 20,
		"pk": "aa", "sig": "bb", "ts": "2026-01-01T00:00:00Z",
		"s": "not an object",
		"b": {"id":"e","mr":"f","mh":"g","kid":"h"},
		"ip": "AA",
		"cx": [{"t":40,"net":"testnet","ep":"AA"}]
	}`))
	if err == nil {
		t.Error("expected error for invalid subject structure")
	}
}

func TestParse_EntropyProof(t *testing.T) {
	bundle, err := ParseBytes([]byte(validEntropyProofJSON))
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	if bundle.T != ptype.EntropyNIST {
		t.Errorf("T: got %d, want %d", bundle.T, ptype.EntropyNIST)
	}
	if !bundle.IsEntropy() {
		t.Error("IsEntropy() should return true")
	}
	if bundle.Subject == nil {
		t.Fatal("subject must be present for entropy proofs")
	}
	if len(bundle.RawData) == 0 {
		t.Error("RawData should be populated")
	}
}

func TestParse_BlockProofNoSubjectNoIP(t *testing.T) {
	bundle, err := ParseBytes([]byte(validBlockProofJSON))
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	if !bundle.IsBlock() {
		t.Error("IsBlock() should be true")
	}
	if bundle.Subject != nil {
		t.Error("Subject must be nil for block proofs")
	}
	if bundle.InclusionProof != "" {
		t.Errorf("InclusionProof must be empty for block proofs, got %q", bundle.InclusionProof)
	}
}

func TestParse_BlockProofRejectsSubject(t *testing.T) {
	_, err := ParseBytes([]byte(`{
		"v": 1, "t": 10,
		"pk": "aa", "sig": "bb", "ts": "2026-01-01T00:00:00Z",
		"s": {"id":"x","d":{},"mh":"cc","kid":"dd"},
		"b": {"id":"e","mr":"f","mh":"g","kid":"h"},
		"cx": [{"t":40,"net":"testnet","ep":"AA"}]
	}`))
	if err == nil {
		t.Error("block proof with s must be rejected")
	}
}

func TestParse_BlockProofRejectsIP(t *testing.T) {
	_, err := ParseBytes([]byte(`{
		"v": 1, "t": 10,
		"pk": "aa", "sig": "bb", "ts": "2026-01-01T00:00:00Z",
		"b": {"id":"e","mr":"f","mh":"g","kid":"h"},
		"ip": "AA",
		"cx": [{"t":40,"net":"testnet","ep":"AA"}]
	}`))
	if err == nil {
		t.Error("block proof with ip must be rejected")
	}
}

func TestFileSizeFromData(t *testing.T) {
	if got := FileSizeFromData([]byte("hello world")); got != 11 {
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
