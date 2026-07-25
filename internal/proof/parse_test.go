// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: MIT

package proof

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
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

// Beacon bundle (t=11) has the same wire shape as a block bundle — no
// `s`, no `ip` — but a distinct type code. The signature would also
// differ from the block proof of the same block (the `t` byte is in
// the signing payload), but this fixture uses a placeholder `sig`
// like all the other test bundles.
const validBeaconProofJSON = `{
  "v": 1,
  "t": 11,
  "pk": "CTwMqDZnPd/QTLSq8aTeSD3a+j2DQxKcGfhhIYJQ65Y=",
  "sig": "BBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBB==",
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
	t.Parallel()
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
	if !bundle.IsItem() || bundle.IsEntropy() || bundle.IsBlockLike() {
		t.Errorf("item flags: IsItem/IsEntropy/IsBlockLike: %v/%v/%v, want t/f/f",
			bundle.IsItem(), bundle.IsEntropy(), bundle.IsBlockLike())
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
	t.Parallel()
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
	t.Parallel()
	_, err := Parse("/nonexistent/path/proof.json")
	if err == nil {
		t.Error("expected error for missing file")
	}
}

func TestParse_InvalidJSON(t *testing.T) {
	t.Parallel()
	path := writeTemp(t, "not json at all")
	_, err := Parse(path)
	if err == nil {
		t.Error("expected error for invalid JSON")
	}
}

// TestParse_MissingVersionIsNotARejection pins E.6's explicit exception:
// `v` is not a structural gate. An absent (or zero) version must parse and
// carry through so the version step can report it as a failure.
func TestParse_MissingVersionIsNotARejection(t *testing.T) {
	t.Parallel()
	for _, c := range []struct{ name, body string }{
		{"absent", `"t": 20,`},
		{"zero", `"v": 0, "t": 20,`},
		{"wrong", `"v": 2, "t": 20,`},
	} {
		t.Run(c.name, func(t *testing.T) {
			bundle, err := ParseBytes([]byte(`{` + c.body + `
				"pk": "aa", "sig": "bb", "ts": "2026-01-01T00:00:00Z",
				"s": {"id":"x","d":{},"mh":"cc","kid":"dd"},
				"b": {"id":"e","mr":"f","mh":"g","kid":"h"},
				"ip": "AA",
				"cx": [{"t":40,"net":"testnet","memo":"bb","ep":"AA"}]
			}`))
			if err != nil {
				t.Fatalf("v must not be a hard rejection: %v", err)
			}
			if c.name == "wrong" && bundle.Version != 2 {
				t.Errorf("version: got %d, want 2", bundle.Version)
			}
			if c.name != "wrong" && bundle.Version != 0 {
				t.Errorf("version: got %d, want 0", bundle.Version)
			}
		})
	}
}

func TestParse_MissingTypeCode(t *testing.T) {
	t.Parallel()
	_, err := ParseBytes([]byte(`{
		"v": 1, "pk": "aa", "sig": "bb", "ts": "2026-01-01T00:00:00Z",
		"s": {"id":"x","d":{},"mh":"cc","kid":"dd"},
		"b": {"id":"e","mr":"f","mh":"g","kid":"h"},
		"ip": "AA",
		"cx": [{"t":40,"net":"testnet","memo":"bb","ep":"AA"}]
	}`))
	if err == nil {
		t.Error("expected error for missing t field")
	}
}

func TestParse_InvalidTypeCode(t *testing.T) {
	t.Parallel()
	_, err := ParseBytes([]byte(`{
		"v": 1, "t": 99,
		"pk": "aa", "sig": "bb", "ts": "2026-01-01T00:00:00Z",
		"b": {"id":"e","mr":"f","mh":"g","kid":"h"},
		"cx": [{"t":40,"net":"testnet","memo":"bb","ep":"AA"}]
	}`))
	if err == nil {
		t.Error("expected error for unknown t code")
	}
}

// TestParse_UnauthorisedRejectionsAreGone covers every field E.6 does NOT
// list. Each of these used to abort the run, erasing the report the caller
// needed; they must now parse so the signing-key, subject-hash, block-hash
// and signature steps can each report their own failure.
func TestParse_UnauthorisedRejectionsAreGone(t *testing.T) {
	t.Parallel()
	cases := []struct{ name, body string }{
		{"pk absent", `"v":1,"t":20,"sig":"bb"`},
		{"sig absent", `"v":1,"t":20,"pk":"aa"`},
		{"pk short", `"v":1,"t":20,"pk":"YWI=","sig":"bb"`},
		{"pk not a string", `"v":1,"t":20,"pk":7,"sig":"bb"`},
		{"ts not a string", `"v":1,"t":20,"pk":"aa","sig":"bb","ts":7`},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			_, err := ParseBytes([]byte(`{` + c.body + `,
				"s": {"id":"x","d":{},"mh":"cc","kid":"dd"},
				"b": {"id":"e","mr":"f","mh":"g","kid":"h"},
				"ip": "AA",
				"cx": [{"t":40,"net":"testnet","memo":"bb","ep":"AA"}]
			}`))
			if err != nil {
				t.Errorf("E.6 authorises no rejection here, got: %v", err)
			}
		})
	}
}

// TestParse_SubjectAndBlockFieldsAreNotGates pins the other half of the same
// rule: absent or wrong-sized `s.*` / `b.*` members reach the report rather
// than aborting it.
func TestParse_SubjectAndBlockFieldsAreNotGates(t *testing.T) {
	t.Parallel()
	cases := []struct{ name, subject, block string }{
		{"s.id absent", `{"d":{},"mh":"cc","kid":"dd"}`, `{"id":"e","ph":"a","mr":"f","mh":"g","kid":"h"}`},
		{"s.mh absent", `{"id":"x","d":{},"kid":"dd"}`, `{"id":"e","ph":"a","mr":"f","mh":"g","kid":"h"}`},
		{"s.kid absent", `{"id":"x","d":{},"mh":"cc"}`, `{"id":"e","ph":"a","mr":"f","mh":"g","kid":"h"}`},
		{"b.ph absent", `{"id":"x","d":{},"mh":"cc","kid":"dd"}`, `{"id":"e","mr":"f","mh":"g","kid":"h"}`},
		{"b.mr absent", `{"id":"x","d":{},"mh":"cc","kid":"dd"}`, `{"id":"e","ph":"a","mh":"g","kid":"h"}`},
		{"b.mh absent", `{"id":"x","d":{},"mh":"cc","kid":"dd"}`, `{"id":"e","ph":"a","mr":"f","kid":"h"}`},
		{"b.kid absent", `{"id":"x","d":{},"mh":"cc","kid":"dd"}`, `{"id":"e","ph":"a","mr":"f","mh":"g"}`},
		{"b.id absent", `{"id":"x","d":{},"mh":"cc","kid":"dd"}`, `{"ph":"a","mr":"f","mh":"g","kid":"h"}`},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			_, err := ParseBytes([]byte(`{"v":1,"t":20,"pk":"aa","sig":"bb","ts":"2026-01-01T00:00:00Z",
				"s": ` + c.subject + `,
				"b": ` + c.block + `,
				"ip": "AA",
				"cx": [{"t":40,"net":"testnet","memo":"bb","ep":"AA"}]
			}`))
			if err != nil {
				t.Errorf("E.6 authorises no rejection here, got: %v", err)
			}
		})
	}
}

func TestParse_MissingSubjectForItem(t *testing.T) {
	t.Parallel()
	_, err := ParseBytes([]byte(`{
		"v": 1, "t": 20,
		"pk": "aa", "sig": "bb", "ts": "2026-01-01T00:00:00Z",
		"b": {"id":"e","mr":"f","mh":"g","kid":"h"},
		"ip": "AA",
		"cx": [{"t":40,"net":"testnet","memo":"bb","ep":"AA"}]
	}`))
	if err == nil {
		t.Error("expected error for missing s on item proof")
	}
}

func TestParse_MissingInclusionProofForItem(t *testing.T) {
	t.Parallel()
	_, err := ParseBytes([]byte(`{
		"v": 1, "t": 20,
		"pk": "aa", "sig": "bb", "ts": "2026-01-01T00:00:00Z",
		"s": {"id":"x","d":{},"mh":"cc","kid":"dd"},
		"b": {"id":"e","mr":"f","mh":"g","kid":"h"},
		"cx": [{"t":40,"net":"testnet","memo":"bb","ep":"AA"}]
	}`))
	if err == nil {
		t.Error("expected error for missing ip on item proof")
	}
}

func TestParse_MissingBlock(t *testing.T) {
	t.Parallel()
	_, err := ParseBytes([]byte(`{
		"v": 1, "t": 20,
		"pk": "aa", "sig": "bb", "ts": "2026-01-01T00:00:00Z",
		"s": {"id":"x","d":{},"mh":"cc","kid":"dd"},
		"ip": "AA",
		"cx": [{"t":40,"net":"testnet","memo":"bb","ep":"AA"}]
	}`))
	if err == nil {
		t.Error("expected error for missing block")
	}
}

func TestParse_EmptyCxRejected(t *testing.T) {
	t.Parallel()
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
	t.Parallel()
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
	t.Parallel()
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
	t.Parallel()
	_, err := ParseBytes([]byte(`{
		"v": 1, "t": 20,
		"pk": "aa", "sig": "bb", "ts": "2026-01-01T00:00:00Z",
		"s": "not an object",
		"b": {"id":"e","mr":"f","mh":"g","kid":"h"},
		"ip": "AA",
		"cx": [{"t":40,"net":"testnet","memo":"bb","ep":"AA"}]
	}`))
	if err == nil {
		t.Error("expected error for invalid subject structure")
	}
}

func TestParse_EntropyProof(t *testing.T) {
	t.Parallel()
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
	t.Parallel()
	bundle, err := ParseBytes([]byte(validBlockProofJSON))
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	// The raw code is what E.16 signs, so that is what gets pinned.
	if bundle.T != ptype.Block {
		t.Errorf("T: got %d, want %d (block)", bundle.T, ptype.Block)
	}
	if !bundle.IsBlockLike() {
		t.Error("IsBlockLike() should be true for a t=10 bundle")
	}
	if bundle.Subject != nil {
		t.Error("Subject must be nil for block proofs")
	}
	if bundle.InclusionProof != "" {
		t.Errorf("InclusionProof must be empty for block proofs, got %q", bundle.InclusionProof)
	}
}

// TestParse_BeaconProof covers the t=11 beacon wire shape: structurally
// identical to a block (no s, no ip, non-empty cx) but carrying a distinct
// type code. Both halves are pinned because both matter — the shared shape
// is why the pipeline guards on IsBlockLike, and the distinct code is the
// domain separation that makes a beacon signature differ from the block
// signature over the same block (`t` is inside E.16's signed payload).
func TestParse_BeaconProof(t *testing.T) {
	t.Parallel()
	bundle, err := ParseBytes([]byte(validBeaconProofJSON))
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	if bundle.T != ptype.Beacon {
		t.Errorf("T: got %d, want %d (beacon, NOT block=%d)", bundle.T, ptype.Beacon, ptype.Block)
	}
	if !bundle.IsBlockLike() {
		t.Error("IsBlockLike() should be true for a t=11 bundle")
	}
	if bundle.Subject != nil {
		t.Error("Subject must be nil for beacon proofs")
	}
	if bundle.InclusionProof != "" {
		t.Errorf("InclusionProof must be empty for beacon proofs, got %q", bundle.InclusionProof)
	}
}

// TestParse_BeaconProofRejectsSubject mirrors the block rejection test —
// a t=11 bundle carrying an `s` field must be rejected.
func TestParse_BeaconProofRejectsSubject(t *testing.T) {
	t.Parallel()
	_, err := ParseBytes([]byte(`{
		"v": 1, "t": 11,
		"pk": "aa", "sig": "bb", "ts": "2026-01-01T00:00:00Z",
		"s": {"id":"x","d":{},"mh":"cc","kid":"dd"},
		"b": {"id":"e","mr":"f","mh":"g","kid":"h"},
		"cx": [{"t":40,"net":"testnet","memo":"bb","ep":"AA"}]
	}`))
	if err == nil {
		t.Error("beacon proof with s must be rejected")
	}
}

func TestParse_BlockProofRejectsSubject(t *testing.T) {
	t.Parallel()
	_, err := ParseBytes([]byte(`{
		"v": 1, "t": 10,
		"pk": "aa", "sig": "bb", "ts": "2026-01-01T00:00:00Z",
		"s": {"id":"x","d":{},"mh":"cc","kid":"dd"},
		"b": {"id":"e","mr":"f","mh":"g","kid":"h"},
		"cx": [{"t":40,"net":"testnet","memo":"bb","ep":"AA"}]
	}`))
	if err == nil {
		t.Error("block proof with s must be rejected")
	}
}

func TestParse_BlockProofRejectsIP(t *testing.T) {
	t.Parallel()
	_, err := ParseBytes([]byte(`{
		"v": 1, "t": 10,
		"pk": "aa", "sig": "bb", "ts": "2026-01-01T00:00:00Z",
		"b": {"id":"e","mr":"f","mh":"g","kid":"h"},
		"ip": "AA",
		"cx": [{"t":40,"net":"testnet","memo":"bb","ep":"AA"}]
	}`))
	if err == nil {
		t.Error("block proof with ip must be rejected")
	}
}

// itemJSON builds an item bundle with the given `cx` array and `ip` value,
// so the E.6 gates can be exercised one field at a time.
func itemJSON(cx, ip string) string {
	return `{"v":1,"t":20,"pk":"aa","sig":"bb","ts":"2026-01-01T00:00:00Z",
		"s": {"id":"x","d":{},"mh":"cc","kid":"dd"},
		"b": {"id":"e","ph":"a","mr":"f","mh":"g","kid":"h"},
		"ip": ` + ip + `,
		"cx": ` + cx + `}`
}

// TestParse_CommitmentEpGate covers the one per-entry rejection E.6 does
// authorise for `ep`: presence plus string-ness. An empty `ep` is accepted
// and graded later by the epoch-proof step.
func TestParse_CommitmentEpGate(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name string
		cx   string
		code string
	}{
		{"absent", `[{"t":40,"net":"testnet","memo":"bb"}]`, CodeInvalidExternalCommitmentEntry},
		{"null", `[{"t":40,"net":"testnet","memo":"bb","ep":null}]`, CodeInvalidExternalCommitmentEntry},
		{"not a string", `[{"t":40,"net":"testnet","memo":"bb","ep":7}]`, CodeInvalidExternalCommitmentEntry},
		{"empty string", `[{"t":40,"net":"testnet","memo":"bb","ep":""}]`, ""},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			_, err := ParseBytes([]byte(itemJSON(c.cx, `"AA"`)))
			if got := RejectionCode(err); got != c.code {
				t.Errorf("RejectionCode = %q, want %q (err: %v)", got, c.code, err)
			}
		})
	}
}

// TestParse_CommitmentRootKeyGateIsPresenceOnly pins the deliberate
// asymmetry with `ep`: a malformed root key must reach the report as an
// epoch-proof failure, so only its absence aborts.
func TestParse_CommitmentRootKeyGateIsPresenceOnly(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name string
		cx   string
		code string
	}{
		{"stellar memo absent", `[{"t":40,"net":"testnet","ep":"AA"}]`, CodeInvalidExternalCommitmentEntry},
		{"stellar memo null", `[{"t":40,"net":"testnet","memo":null,"ep":"AA"}]`, CodeInvalidExternalCommitmentEntry},
		{"stellar memo empty", `[{"t":40,"net":"testnet","memo":"","ep":"AA"}]`, ""},
		{"stellar memo short", `[{"t":40,"net":"testnet","memo":"aabb","ep":"AA"}]`, ""},
		{"bitcoin op absent", `[{"t":41,"net":"regtest","ep":"AA"}]`, CodeInvalidExternalCommitmentEntry},
		{"bitcoin op empty", `[{"t":41,"net":"regtest","op":"","ep":"AA"}]`, ""},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			_, err := ParseBytes([]byte(itemJSON(c.cx, `"AA"`)))
			if got := RejectionCode(err); got != c.code {
				t.Errorf("RejectionCode = %q, want %q (err: %v)", got, c.code, err)
			}
		})
	}
}

// TestParse_OptionalCommitmentFields covers the fields E.5 grades as
// "required for external confirmation" or "optional": absence narrows what a
// confirmation step can establish and must never abort the parse. The
// Appendix D bundle itself carries no `bmr`.
func TestParse_OptionalCommitmentFields(t *testing.T) {
	t.Parallel()
	cases := []struct{ name, cx string }{
		{"stellar tx absent", `[{"t":40,"net":"testnet","memo":"bb","ep":"AA"}]`},
		{"stellar tx malformed", `[{"t":40,"net":"testnet","tx":"aabb","memo":"bb","ep":"AA"}]`},
		{"bitcoin bmr absent", `[{"t":41,"net":"regtest","op":"aa","ep":"AA"}]`},
		{"bitcoin bmr malformed", `[{"t":41,"net":"regtest","op":"aa","bmr":"aabb","ep":"AA"}]`},
		{"net absent", `[{"t":40,"memo":"bb","ep":"AA"}]`},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if _, err := ParseBytes([]byte(itemJSON(c.cx, `"AA"`))); err != nil {
				t.Errorf("must parse cleanly, got: %v", err)
			}
		})
	}
}

// TestParse_EmptyInclusionProofIsNotARejection pins that `ip` is gated on
// presence and string-ness, not on being non-empty — an empty proof fails
// the inclusion-proof step where the report can show it.
func TestParse_EmptyInclusionProofIsNotARejection(t *testing.T) {
	t.Parallel()
	bundle, err := ParseBytes([]byte(itemJSON(`[{"t":40,"net":"testnet","memo":"bb","ep":"AA"}]`, `""`)))
	if err != nil {
		t.Fatalf("empty ip must parse: %v", err)
	}
	if bundle.InclusionProof != "" {
		t.Errorf("ip: got %q, want empty", bundle.InclusionProof)
	}
	for _, ip := range []string{`null`, `7`} {
		_, err := ParseBytes([]byte(itemJSON(`[{"t":40,"net":"testnet","memo":"bb","ep":"AA"}]`, ip)))
		if got := RejectionCode(err); got != CodeMissingInclusionProof {
			t.Errorf("ip %s: RejectionCode = %q, want %q", ip, got, CodeMissingInclusionProof)
		}
	}
}

// blockLikeShapeCases is the E.6 truth table for `s` / `ip` on a block-like
// bundle, shared by the JSON and CBOR shape tests so the two serializations
// are pinned to one rule.
var blockLikeShapeCases = []struct {
	name     string
	subject  any // nil means "key absent"
	ip       any
	rejected bool
}{
	{"neither", nil, nil, false},
	{"s null", "null", nil, false},
	{"ip null", nil, "null", false},
	{"ip empty string", nil, "", true},
	{"s object", map[string]any{"id": "x"}, nil, true},
	{"s empty string", "", nil, true},
}

func TestParse_BlockLikeShapeRule(t *testing.T) {
	t.Parallel()
	for _, c := range blockLikeShapeCases {
		t.Run(c.name, func(t *testing.T) {
			body := map[string]any{
				"v": 1, "t": 10, "pk": "aa", "sig": "bb", "ts": "2026-01-01T00:00:00Z",
				"b":  map[string]any{"id": "e", "ph": "a", "mr": "f", "mh": "g", "kid": "h"},
				"cx": []any{map[string]any{"t": 40, "net": "testnet", "memo": "bb", "ep": "AA"}},
			}
			if c.subject != nil {
				body["s"] = c.subject
			}
			if c.ip != nil {
				body["ip"] = c.ip
			}
			raw, err := json.Marshal(body)
			if err != nil {
				t.Fatal(err)
			}
			// The "null" strings above must land as JSON null, not as the
			// four-character string.
			raw = bytes.ReplaceAll(raw, []byte(`"null"`), []byte(`null`))

			_, err = ParseBytes(raw)
			got := RejectionCode(err) == CodeUnexpectedSubjectFieldsForBlockLike
			if got != c.rejected {
				t.Errorf("rejected = %v, want %v (err: %v)", got, c.rejected, err)
			}
		})
	}
}

// TestParse_NotAJSONObject checks that non-object input is refused with the
// E.23 identifier instead of leaking the decoder's anonymous target struct.
func TestParse_NotAJSONObject(t *testing.T) {
	t.Parallel()
	for _, in := range []string{`[1,2]`, `null`, `"a string"`, `42`, `true`, ``, "   "} {
		_, err := ParseBytes([]byte(in))
		if got := RejectionCode(err); got != CodeNotAJSONObject {
			t.Errorf("ParseBytes(%q): RejectionCode = %q, want %q (err: %v)", in, got, CodeNotAJSONObject, err)
		}
		if strings.Contains(fmt.Sprint(err), "json.RawMessage") {
			t.Errorf("ParseBytes(%q) leaked the decode struct: %v", in, err)
		}
	}
	// Leading whitespace before the object must still be accepted.
	if _, err := ParseBytes([]byte("\n\t " + validProofJSON)); err != nil {
		t.Errorf("leading whitespace must be tolerated: %v", err)
	}
}

// TestRejectionCode_NonRejection confirms the accessor stays silent for the
// ordinary errors a caller also sees from this package.
func TestRejectionCode_NonRejection(t *testing.T) {
	t.Parallel()
	if got := RejectionCode(nil); got != "" {
		t.Errorf("RejectionCode(nil) = %q", got)
	}
	_, err := Parse("/nonexistent/path/proof.json")
	if got := RejectionCode(err); got != "" {
		t.Errorf("RejectionCode(file error) = %q", got)
	}
	if got := RejectionCode(fmt.Errorf("wrapped: %w", rejectf(CodeMissingBlock, "x"))); got != CodeMissingBlock {
		t.Errorf("RejectionCode through a wrap = %q, want %q", got, CodeMissingBlock)
	}
}

// TestParse_RejectionCodes maps each E.6 condition to its E.23 identifier.
func TestParse_RejectionCodes(t *testing.T) {
	t.Parallel()
	cases := []struct{ name, body, code string }{
		{"t absent", `{"v":1,"b":{},"cx":[]}`, CodeMissingTypeCode},
		{"t null", `{"v":1,"t":null,"b":{},"cx":[]}`, CodeMissingTypeCode},
		{"t not an integer", `{"v":1,"t":"20","b":{},"cx":[]}`, CodeMissingTypeCode},
		{"t unknown", `{"v":1,"t":99,"b":{},"cx":[]}`, CodeInvalidSubjectTypeCode},
		{"b absent", `{"v":1,"t":10,"cx":[]}`, CodeMissingBlock},
		{"b not a map", `{"v":1,"t":10,"b":"x","cx":[]}`, CodeMissingBlock},
		{"cx absent", `{"v":1,"t":10,"b":{}}`, CodeNoExternalCommitments},
		{"cx not a list", `{"v":1,"t":10,"b":{},"cx":{}}`, CodeNoExternalCommitments},
		{"cx empty", `{"v":1,"t":10,"b":{},"cx":[]}`, CodeNoExternalCommitments},
		{"cx entry not a map", `{"v":1,"t":10,"b":{},"cx":["x"]}`, CodeInvalidExternalCommitmentEntry},
		{"cx entry bad code", `{"v":1,"t":10,"b":{},"cx":[{"t":99,"ep":"AA"}]}`, CodeInvalidExternalCommitmentEntry},
		{"s absent", `{"v":1,"t":20,"b":{},"cx":[{"t":40,"memo":"bb","ep":"AA"}],"ip":"AA"}`, CodeMissingSubject},
		{"s not a map", `{"v":1,"t":20,"s":"x","b":{},"cx":[{"t":40,"memo":"bb","ep":"AA"}],"ip":"AA"}`, CodeMissingSubject},
		{"ip absent", `{"v":1,"t":20,"s":{},"b":{},"cx":[{"t":40,"memo":"bb","ep":"AA"}]}`, CodeMissingInclusionProof},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			_, err := ParseBytes([]byte(c.body))
			if got := RejectionCode(err); got != c.code {
				t.Errorf("RejectionCode = %q, want %q (err: %v)", got, c.code, err)
			}
		})
	}
}

func TestFileSizeFromData(t *testing.T) {
	t.Parallel()
	if got := FileSizeFromData([]byte("hello world")); got != 11 {
		t.Errorf("FileSizeFromData: got %d, want 11", got)
	}
}

func TestFileSizeFromData_Empty(t *testing.T) {
	t.Parallel()
	if got := FileSizeFromData(nil); got != 0 {
		t.Errorf("FileSizeFromData(nil): got %d, want 0", got)
	}
}

func TestFileSize_Exists(t *testing.T) {
	t.Parallel()
	path := writeTemp(t, "hello")
	size := FileSize(path)
	if size != 5 {
		t.Errorf("file size: got %d, want 5", size)
	}
}

func TestFileSize_NotExists(t *testing.T) {
	t.Parallel()
	size := FileSize("/nonexistent/file")
	if size != 0 {
		t.Errorf("file size for missing file: got %d, want 0", size)
	}
}
