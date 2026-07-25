// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: MIT

package tscrypto

import (
	"encoding/base64"
	"sort"
	"strings"
	"testing"

	"github.com/truestamp/truestamp-cli/internal/jcs"
)

func TestDomainHash_ClaimsVector(t *testing.T) {
	t.Parallel()
	// Test vector from docs/CRYPTOGRAPHY.md: SHA256(0x11 || "truestamp")
	result := BytesToHex(DomainHash(0x11, []byte("truestamp")))
	expected := "9d9443e5133052d9fc837e150d32e3094fc979f097922034a984ddfbe5247aca"
	if result != expected {
		t.Errorf("claims hash: got %s, want %s", result, expected)
	}
}

func TestDomainHash_EntropyVector(t *testing.T) {
	t.Parallel()
	// Test vector: SHA256(0x21 || "truestamp")
	result := BytesToHex(DomainHash(0x21, []byte("truestamp")))
	expected := "22da1f4e92cfd2318f0cec8c3b8f7bf3c7c2531db0e5ac781f25e1e25ae65831"
	if result != expected {
		t.Errorf("entropy hash: got %s, want %s", result, expected)
	}
}

func TestDomainHash_GenesisVector(t *testing.T) {
	t.Parallel()
	// Test vector: SHA256(0x31 || "truestamp")
	result := BytesToHex(DomainHash(0x31, []byte("truestamp")))
	expected := "96965205fa528419d226675f8ad4a11f978d5523e31c462f4af0f1405d2ac8da"
	if result != expected {
		t.Errorf("genesis hash: got %s, want %s", result, expected)
	}
}

func TestDomainHash_DifferentPrefixesDiffer(t *testing.T) {
	t.Parallel()
	data := []byte("same input")
	h11 := BytesToHex(DomainHash(0x11, data))
	h12 := BytesToHex(DomainHash(0x12, data))
	h21 := BytesToHex(DomainHash(0x21, data))
	if h11 == h12 || h11 == h21 || h12 == h21 {
		t.Error("different domain prefixes should produce different hashes")
	}
}

func TestComputeKeyID(t *testing.T) {
	t.Parallel()
	// Test vector: known public key -> key_id
	pubkeyB64 := "CTwMqDZnPd/QTLSq8aTeSD3a+j2DQxKcGfhhIYJQ65Y="
	pubkey, _ := base64.StdEncoding.DecodeString(pubkeyB64)
	result := ComputeKeyID(pubkey)
	expected := "4ceefa4a"
	if result != expected {
		t.Errorf("key_id: got %s, want %s", result, expected)
	}
}

// Appendix D.1's bundle and the intermediate values D.3 publishes for it. They
// are the normative worked example, so they are the right anchor for the four
// preimages this package builds; a length-only assertion (the shape most of the
// tests below use) cannot tell a correct preimage from a permuted one.
const (
	dSubjectID       = "01KY9ZWEX0248J48HK6D248NAN"
	dSubjectMH       = "29c3caff391f2bbaeab20a8f109100e74d66ca1645cbe285c4c28149acb631d0"
	dKID             = "f2c39df9"
	dCanonicalData   = `{"description":"Illustrative item for the Truestamp whitepaper. Not a production record.","hash":"b47cc0f104b62d4c7c30bcd68fd8e67613e287dc4ad8c310ef10cbadea9c4380","hash_type":"sha256","name":"Appendix D worked example"}`
	dDataHash        = "ae5cdc73a52359a4fb0e335f004a6cd0cb1247024e812263e362368915a4b924"
	dSubjectHash     = "8b7bd9f508fd6327535a8be8ba3cd9297f3540ac639d149fa3ece2089d0576df"
	dBlockID         = "019f93ff-2600-7c30-8000-000000000c30"
	dBlockPH         = "a3b8c9d0e1f2a3b4c5d6e7f80910111213141516171819202122232425262728"
	dBlockMR         = "6d8f9587a0822e2cde700560afa919f7599615f6cdf15d0198997c03fce862b6"
	dBlockMH         = "14fe55ee4ce3cdbc8118a0a28e5a80a44f1f8a24d73d9949c0ecd91ee582ebe1"
	dBlockHash       = "f2ece2563a02208efe481f3aee38729aa81775e77756b312299e8fe6b6107c0e"
	dEpochRoot0      = "02f978c0e8c7e9a6b9bafe298c11a4f374114e947bb75dc10325c720920887e4"
	dEpochRoot1      = "07b98eb45c3b1651e574ec2c5186f7bf105322cef5ccf2b43d9a6037b615647b"
	dTimestampMs     = 1784894400000 // 2026-07-24T12:00:00Z
	dProofHash       = "e82a35028a29ead5d5ad8acdc460b15b952d4954a2008f4c6a58b4aacbfc269c"
	dPublicKeyBase64 = "IVL40Zt5HSRFMkLhXy6rbLfP+ntqXtMAl5YOBpiB2xI="
	dSignatureBase64 = "vW5q+DlpSfp9S+MZMnm2SjYghlTRMgC+JnvpAtpj+lcDOg8TRWFdt0LPnwGreBoNB4uTPf/4eQ6BBMm5NHchCw=="
)

// TestAppendixDKeyID pins E.9's derivation against the published key id, which
// is also the value that fills the E.16 payload's kid slot.
func TestAppendixDKeyID(t *testing.T) {
	t.Parallel()
	pubkey, err := base64.StdEncoding.DecodeString(dPublicKeyBase64)
	if err != nil {
		t.Fatalf("decoding pk: %s", err)
	}
	if got := ComputeKeyID(pubkey); got != dKID {
		t.Errorf("key_id = %s, want %s", got, dKID)
	}
}

// TestAppendixDDataHash pins E.7's 0x11 digest, canonicalized through the
// production canonicalizer rather than a library called directly, so the two
// cannot drift apart unnoticed.
func TestAppendixDDataHash(t *testing.T) {
	t.Parallel()
	canonical, err := jcs.Transform([]byte(dCanonicalData))
	if err != nil {
		t.Fatalf("JCS transform: %s", err)
	}
	if string(canonical) != dCanonicalData {
		t.Errorf("canonical form moved:\n got %s\nwant %s", canonical, dCanonicalData)
	}
	if got := BytesToHex(DomainHash(PrefixItemClaims, canonical)); got != dDataHash {
		t.Errorf("data_hash = %s, want %s", got, dDataHash)
	}
}

// TestAppendixDSubjectHash pins E.10's 0x13 composite: the field ORDER, not
// just the digest width.
func TestAppendixDSubjectHash(t *testing.T) {
	t.Parallel()
	got, err := ComputeItemHash(dSubjectID, dDataHash, dSubjectMH, dKID)
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	if got != dSubjectHash {
		t.Errorf("subject_hash = %s, want %s", got, dSubjectHash)
	}
}

// TestAppendixDBlockHash pins E.14's 0x32 composite over all five block fields.
// Without a known answer here, swapping b.mr and b.mh in the preimage passes
// every other test in this package.
func TestAppendixDBlockHash(t *testing.T) {
	t.Parallel()
	got, err := ComputeBlockHash(dBlockID, dBlockPH, dBlockMR, dBlockMH, dKID)
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	if got != dBlockHash {
		t.Errorf("block_hash = %s, want %s", got, dBlockHash)
	}
}

// TestAppendixDProofHashAndSignature pins E.16 end to end: the 81 + 32*N byte
// layout, the 0x61 proof hash it produces, and an Ed25519 verification of the
// published signature over it. BuildCompactProofPayload had no known-answer
// test at all before this — only a crash fuzzer — so every offset and width in
// the payload was unpinned inside the package that defines them.
func TestAppendixDProofHashAndSignature(t *testing.T) {
	t.Parallel()
	got, err := BuildCompactProofPayload(
		1, 20, dKID, dTimestampMs, dSubjectHash, dBlockHash,
		[]string{dEpochRoot0, dEpochRoot1},
	)
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	if BytesToHex(got) != dProofHash {
		t.Errorf("proof_hash = %s, want %s", BytesToHex(got), dProofHash)
	}

	// The same 145 bytes assembled independently from D.3 Step 9's field-by-
	// field trace, so a change to any offset or width in the production builder
	// shows up here rather than only as a signature failure downstream.
	payload := mustHex(t, "01"+"0014"+dKID+"0000019f93ff2600"+dSubjectHash+dBlockHash+"0002"+dEpochRoot0+dEpochRoot1)
	if len(payload) != 81+32*2 {
		t.Fatalf("hand-assembled payload is %d bytes, want %d", len(payload), 81+32*2)
	}
	if want := BytesToHex(DomainHash(PrefixProofHash, payload)); BytesToHex(got) != want {
		t.Errorf("builder produced %s, hand-assembled payload gives %s", BytesToHex(got), want)
	}

	pubkey, err := base64.StdEncoding.DecodeString(dPublicKeyBase64)
	if err != nil {
		t.Fatalf("decoding pk: %s", err)
	}
	ok, err := VerifyEd25519(got, dSignatureBase64, pubkey)
	if err != nil {
		t.Fatalf("VerifyEd25519: %s", err)
	}
	if !ok {
		t.Error("the published Appendix D signature does not verify over the derived proof hash")
	}
}

// TestPreimageFieldOrderIsLoadBearing is the generic companion to the Appendix
// D known answers: for every composite builder, exchanging two same-width
// inputs must move the digest. A builder that concatenated its inputs without
// the length prefixes, or that read them in the wrong order, would collapse
// these pairs onto one hash.
func TestPreimageFieldOrderIsLoadBearing(t *testing.T) {
	t.Parallel()
	x := "1111111111111111111111111111111111111111111111111111111111111111"
	y := "2222222222222222222222222222222222222222222222222222222222222222"

	item1, err := ComputeItemHash(dSubjectID, x, y, dKID)
	if err != nil {
		t.Fatalf("ComputeItemHash: %s", err)
	}
	item2, err := ComputeItemHash(dSubjectID, y, x, dKID)
	if err != nil {
		t.Fatalf("ComputeItemHash: %s", err)
	}
	if item1 == item2 {
		t.Error("ComputeItemHash: swapping claims_hash and metadata_hash left the digest unchanged")
	}

	obs1, err := ComputeObservationHash(dSubjectID, x, y, dKID)
	if err != nil {
		t.Fatalf("ComputeObservationHash: %s", err)
	}
	obs2, err := ComputeObservationHash(dSubjectID, y, x, dKID)
	if err != nil {
		t.Fatalf("ComputeObservationHash: %s", err)
	}
	if obs1 == obs2 {
		t.Error("ComputeObservationHash: swapping entropy_hash and metadata_hash left the digest unchanged")
	}
	if obs1 == item1 {
		t.Error("0x23 and 0x13 produced the same digest for the same inputs; the domain prefixes are not separating")
	}

	// Three independent swaps across the block preimage's five fields.
	base, err := ComputeBlockHash(dBlockID, dBlockPH, dBlockMR, dBlockMH, dKID)
	if err != nil {
		t.Fatalf("ComputeBlockHash: %s", err)
	}
	swaps := []struct {
		name                        string
		ph, mr, mh                  string
		wantDifferentFromTheBaseOne bool
	}{
		{"merkle_root and metadata_hash", dBlockPH, dBlockMH, dBlockMR, true},
		{"previous_hash and merkle_root", dBlockMR, dBlockPH, dBlockMH, true},
		{"previous_hash and metadata_hash", dBlockMH, dBlockMR, dBlockPH, true},
	}
	for _, sw := range swaps {
		got, err := ComputeBlockHash(dBlockID, sw.ph, sw.mr, sw.mh, dKID)
		if err != nil {
			t.Fatalf("ComputeBlockHash: %s", err)
		}
		if (got != base) != sw.wantDifferentFromTheBaseOne {
			t.Errorf("ComputeBlockHash: swapping %s left the digest unchanged", sw.name)
		}
	}

	// The epoch-root slots of the signature payload are ordered too: cx order
	// is part of what is signed.
	p1, err := BuildCompactProofPayload(1, 20, dKID, dTimestampMs, dSubjectHash, dBlockHash, []string{dEpochRoot0, dEpochRoot1})
	if err != nil {
		t.Fatalf("BuildCompactProofPayload: %s", err)
	}
	p2, err := BuildCompactProofPayload(1, 20, dKID, dTimestampMs, dSubjectHash, dBlockHash, []string{dEpochRoot1, dEpochRoot0})
	if err != nil {
		t.Fatalf("BuildCompactProofPayload: %s", err)
	}
	if BytesEqual(p1, p2) {
		t.Error("BuildCompactProofPayload: reordering the epoch roots left the proof hash unchanged")
	}

	// And so is the subject/block-hash pair, which are the same width and would
	// otherwise be interchangeable.
	p3, err := BuildCompactProofPayload(1, 20, dKID, dTimestampMs, dBlockHash, dSubjectHash, []string{dEpochRoot0, dEpochRoot1})
	if err != nil {
		t.Fatalf("BuildCompactProofPayload: %s", err)
	}
	if BytesEqual(p1, p3) {
		t.Error("BuildCompactProofPayload: swapping subject_hash and block_hash left the proof hash unchanged")
	}
}

// mustHex decodes a hex string that is part of a test fixture.
func mustHex(t *testing.T, s string) []byte {
	t.Helper()
	b, err := HexToBytes(s)
	if err != nil {
		t.Fatalf("decoding fixture %q: %s", s, err)
	}
	return b
}

func TestComputeItemHash(t *testing.T) {
	t.Parallel()
	// ComputeItemHash now takes 4 args: id, claimsHashHex, metadataHashHex, signingKeyIDHex
	result, err := ComputeItemHash(
		"01KKW15X4C9J75W2C30286JY00",
		"1bb362c54384315006393c34d10c4bd4e46abf1270707c35780872926e4e995d",
		"041c99740c6426282271e9341129e75393ed864575ab403a85e8120eed361661",
		"4ceefa4a",
	)
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	if len(result) != 64 {
		t.Errorf("item_hash should be 64 hex chars, got %d", len(result))
	}
}

func TestComputeBlockHash(t *testing.T) {
	t.Parallel()
	// ComputeBlockHash field order: id, prevHashHex, merkleRootHex, metadataHashHex, signingKeyIDHex
	result, err := ComputeBlockHash(
		"019cf813-99b8-730a-84f1-5a711a9c355e",
		"48b7f261f5a2e9dbc121d3541947e847665160d8d7baf4a74daca4ab3d17a09d",
		"076e537400835632e9de64edbf94e5825c8042c52d71187ba6d4af4ea0a18749",
		"14fe55ee4ce3cdbc8118a0a28e5a80a44f1f8a24d73d9949c0ecd91ee582ebe1",
		"4ceefa4a",
	)
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	if len(result) != 64 {
		t.Errorf("block_hash should be 64 hex chars, got %d", len(result))
	}
}

func TestHexToBytes_Empty(t *testing.T) {
	t.Parallel()
	result, err := HexToBytes("")
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	if len(result) != 0 {
		t.Errorf("expected empty slice, got %d bytes", len(result))
	}
}

// TestValidateLowercaseHex pins E.4's encoding rule, "hashes are lowercase
// hex", as this package states it. The uppercase rows are the ones that
// matter: Go's hex.DecodeString is case-insensitive, so every one of them
// decoded to the same bytes as its lowercase spelling before this check
// existed, and a bundle carrying any of them verified here while the
// reference verifier's Base.decode16lower!/1 aborted on it.
func TestValidateLowercaseHex(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name    string
		in      string
		wantErr bool
		// substr, when set, must appear in the error so the message keeps
		// naming the offending byte and its offset — a report that only
		// says "invalid" cannot tell an operator which character to fix.
		substr string
	}{
		{name: "empty is not an encoding defect", in: ""},
		{name: "lowercase digits and a-f", in: "0123456789abcdef"},
		{name: "32-byte lowercase digest", in: strings.Repeat("ab", 32)},
		{name: "uppercase kid", in: "F2C39DF9", wantErr: true, substr: `uppercase "F" at offset 0`},
		{name: "single uppercase byte late in the value", in: "abcdeF", wantErr: true, substr: `uppercase "F" at offset 5`},
		{name: "mixed case", in: "AbCd", wantErr: true, substr: "uppercase"},
		{name: "non-hex letter", in: "zzzz", wantErr: true, substr: "is not a hex digit"},
		{name: "odd length", in: "abc", wantErr: true, substr: "odd length 3"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			err := ValidateLowercaseHex(tc.in)
			if tc.wantErr && err == nil {
				t.Fatalf("ValidateLowercaseHex(%q) = nil, want an error", tc.in)
			}
			if !tc.wantErr && err != nil {
				t.Fatalf("ValidateLowercaseHex(%q) = %v, want nil", tc.in, err)
			}
			if tc.substr != "" && !strings.Contains(err.Error(), tc.substr) {
				t.Errorf("ValidateLowercaseHex(%q) = %q, want it to contain %q", tc.in, err, tc.substr)
			}
		})
	}
}

// TestHexToBytes_RejectsUppercase is the decoder half of the same rule.
// Enforcing at the decoder is what makes an uppercase field a graded step
// failure rather than a hard rejection E.6 does not authorize: every
// preimage builder in this package runs its hex inputs through here.
func TestHexToBytes_RejectsUppercase(t *testing.T) {
	t.Parallel()
	if _, err := HexToBytes("F2C39DF9"); err == nil {
		t.Fatal("HexToBytes accepted an uppercase value; hex.DecodeString is case-insensitive and E.4 is not")
	}
	if _, err := HexToBytes("f2c39df9"); err != nil {
		t.Fatalf("HexToBytes rejected a conforming value: %v", err)
	}
}

// TestPreimageBuilders_RejectUppercaseInputs walks every exported preimage
// builder with one field uppercased. Each must refuse and name the field it
// could not decode, because that error text is what the Subject Data, Block
// Hash and Proof Signature steps render.
func TestPreimageBuilders_RejectUppercaseInputs(t *testing.T) {
	t.Parallel()
	var (
		lowerDigest = strings.Repeat("ab", 32)
		upperDigest = strings.Repeat("AB", 32)
		lowerKid    = "f2c39df9"
		upperKid    = "F2C39DF9"
		ulid        = "01KY9ZWEX0248J48HK6D248NAN"
		uuid        = "019f93ff-2600-7c30-8000-000000000c30"
	)

	for _, tc := range []struct {
		name  string
		call  func() error
		field string
	}{
		{"ComputeItemHash claims_hash", func() error {
			_, err := ComputeItemHash(ulid, upperDigest, lowerDigest, lowerKid)
			return err
		}, "claims_hash"},
		{"ComputeItemHash s.mh", func() error {
			_, err := ComputeItemHash(ulid, lowerDigest, upperDigest, lowerKid)
			return err
		}, "metadata_hash"},
		{"ComputeItemHash s.kid", func() error {
			_, err := ComputeItemHash(ulid, lowerDigest, lowerDigest, upperKid)
			return err
		}, "signing_key_id"},
		{"ComputeObservationHash s.mh", func() error {
			_, err := ComputeObservationHash(uuid, lowerDigest, upperDigest, lowerKid)
			return err
		}, "metadata_hash"},
		{"ComputeBlockHash b.ph", func() error {
			_, err := ComputeBlockHash(uuid, upperDigest, lowerDigest, lowerDigest, lowerKid)
			return err
		}, "previous_block_hash"},
		{"ComputeBlockHash b.mr", func() error {
			_, err := ComputeBlockHash(uuid, lowerDigest, upperDigest, lowerDigest, lowerKid)
			return err
		}, "merkle_root"},
		{"ComputeBlockHash b.mh", func() error {
			_, err := ComputeBlockHash(uuid, lowerDigest, lowerDigest, upperDigest, lowerKid)
			return err
		}, "metadata_hash"},
		{"ComputeBlockHash b.kid", func() error {
			_, err := ComputeBlockHash(uuid, lowerDigest, lowerDigest, lowerDigest, upperKid)
			return err
		}, "signing_key_id"},
		{"BuildCompactProofPayload epoch root", func() error {
			_, err := BuildCompactProofPayload(1, 20, lowerKid, 0, lowerDigest, lowerDigest, []string{upperDigest})
			return err
		}, "epoch_root"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			err := tc.call()
			if err == nil {
				t.Fatal("uppercase input accepted; the same bytes verify under one signature in >= 2 spellings")
			}
			if !strings.Contains(err.Error(), tc.field) {
				t.Errorf("error %q does not name the field %q", err, tc.field)
			}
			if !strings.Contains(err.Error(), "not lowercase hex") {
				t.Errorf("error %q does not say the value is not lowercase hex", err)
			}
		})
	}
}

// TestHexEqual_StillFoldsCaseForItsTwoCallers guards the deliberate split
// made when the decoders went strict: HexEqual keeps folding case, and the
// two reasons are load-bearing rather than historical.
//
// E.7 instructs a verifier to normalize a caller-supplied hash ("trim,
// downcase") before comparing it, and the reference verifier does the same
// at that one site. The E.21 / E.18 / E.19 comparisons read a value chosen
// by an outside service — the NIST beacon API publishes outputValue in
// uppercase — so a case-sensitive compare there grades a sound proof a
// value mismatch. Bundle-carried hex no longer relies on the fold: it is
// rejected upstream by [ValidateLowercaseHex] before any compare sees it.
func TestHexEqual_StillFoldsCaseForItsTwoCallers(t *testing.T) {
	t.Parallel()
	// E.7: the caller types the hash, in whatever case their tool emitted.
	if !HexEqual("B47CC0F1", "b47cc0f1") {
		t.Error("E.7's expected-hash comparison must fold case")
	}
	// E.21: NIST publishes its beacon outputValue in uppercase.
	if !HexEqual(strings.ToUpper(strings.Repeat("ab", 64)), strings.Repeat("ab", 64)) {
		t.Error("E.21's NIST outputValue comparison must fold case")
	}
}

func TestHexEqual(t *testing.T) {
	t.Parallel()
	if !HexEqual("abcdef", "ABCDEF") {
		t.Error("HexEqual should be case-insensitive")
	}
	if !HexEqual("AABB", "aabb") {
		t.Error("HexEqual should be case-insensitive (reverse)")
	}
	if HexEqual("abcdef", "abcdee") {
		t.Error("HexEqual should detect differences")
	}
	// Both directions. The live call sites pass the operands in both orders —
	// HexEqual(opts.ExpectedHash, r.Claims.Hash) puts the short one first —
	// so a guard that only rejected "first longer than second" would let a
	// 4-character --hash "match" a 64-character claims hash.
	if HexEqual("abcdef", "abcd") {
		t.Error("HexEqual should detect length differences (longer first)")
	}
	if HexEqual("abcd", "abcdef") {
		t.Error("HexEqual should detect length differences (shorter first)")
	}
	if HexEqual("", "abcd") || HexEqual("abcd", "") {
		t.Error("HexEqual should never match an empty operand against a non-empty one")
	}
	if !HexEqual("", "") {
		t.Error("HexEqual should match empty strings")
	}
	if !HexEqual("aabb", "aabb") {
		t.Error("HexEqual should match identical lowercase")
	}
}

// hexEqualLegacy is the short-circuiting implementation HexEqual replaced.
// Kept here as a differential oracle: making HexEqual constant-time was a
// hardening, not a semantic change, and any divergence from this function is
// a regression.
func hexEqualLegacy(a, b string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range len(a) {
		ca, cb := a[i], b[i]
		if ca >= 'A' && ca <= 'F' {
			ca += 'a' - 'A'
		}
		if cb >= 'A' && cb <= 'F' {
			cb += 'a' - 'A'
		}
		if ca != cb {
			return false
		}
	}
	return true
}

func TestHexEqualSemanticParity(t *testing.T) {
	t.Parallel()
	cases := []struct {
		a, b string
		want bool
	}{
		{"abcdef", "ABCDEF", true},
		{"AABB", "aabb", true},
		{"deadbeef", "DEADBEEF", true},
		{"abcdef", "abcdee", false},
		// Length asymmetry in BOTH orders. Truncating one operand to the
		// other's length turns HexEqual into a prefix comparison, and the live
		// call site verify.go's HexEqual(opts.ExpectedHash, r.Claims.Hash)
		// passes the caller-supplied — possibly short — operand first.
		{"abcdef", "abcd", false},
		{"abcd", "abcdef", false},
		{"", "a", false},
		{"a", "", false},
		{"abcd", "abcdefabcdefabcdefabcdef", false},
		{"abcdefabcdefabcdefabcdef", "abcd", false},
		{"", "", true},
		{"aabb", "aabb", true},
		// Non-hex and odd-length operands compare as raw folded text. These
		// two rows are the reason HexEqual does NOT hex-decode its operands
		// before comparing: decoding would flip both to false, and both
		// operands can be remote-controlled.
		{"zz", "zz", true},
		{"abc", "abc", true},
		// Only 'A'-'F' fold. 'g'/'G' and the bytes bracketing the fold range
		// ('@' = 'A'-1, '`' = 'a'-1) must stay distinct.
		{"zz", "ZZ", false},
		{"g", "G", false},
		{"@", "`", false},
	}
	for _, tc := range cases {
		if got := HexEqual(tc.a, tc.b); got != tc.want {
			t.Errorf("HexEqual(%q, %q) = %v, want %v", tc.a, tc.b, got, tc.want)
		}
		if got := hexEqualLegacy(tc.a, tc.b); got != tc.want {
			t.Errorf("legacy oracle disagrees with the table for (%q, %q)", tc.a, tc.b)
		}
	}
}

func TestHexEqualParityOverByteRange(t *testing.T) {
	t.Parallel()
	// Every single-byte pair, so the fold boundaries are exhaustively pinned
	// rather than sampled. Both operands are one byte long, so this sweep says
	// nothing about length handling — TestHexEqualParityOverLengths is what
	// covers that axis.
	for a := 0; a < 256; a++ {
		for b := 0; b < 256; b++ {
			sa, sb := string([]byte{byte(a)}), string([]byte{byte(b)})
			if HexEqual(sa, sb) != hexEqualLegacy(sa, sb) {
				t.Fatalf("HexEqual(%q, %q) diverges from the legacy oracle", sa, sb)
			}
		}
	}
}

// TestHexEqualParityOverLengths sweeps the axis the single-byte parity test
// structurally cannot reach: operands of differing lengths, in both orders,
// including the prefix-of relation that a truncating comparison would accept.
// Every one of the live call sites feeds a bundle- or remote-controlled string
// into one side, so "a shorter operand matches a longer one" is a forgery, not
// a cosmetic bug.
func TestHexEqualParityOverLengths(t *testing.T) {
	t.Parallel()
	digest := "b47cc0f104b62d4c7c30bcd68fd8e67613e287dc4ad8c310ef10cbadea9c4380"
	operands := []string{""}
	for n := 1; n <= len(digest); n++ {
		operands = append(operands, digest[:n], strings.ToUpper(digest[:n]))
	}
	operands = append(operands, digest+"0", "zz", "z")

	for _, a := range operands {
		for _, b := range operands {
			got, want := HexEqual(a, b), hexEqualLegacy(a, b)
			if got != want {
				t.Fatalf("HexEqual(%q, %q) = %v, legacy oracle gives %v", a, b, got, want)
			}
			// The oracle shares the length guard, so assert the property
			// directly too rather than trusting the two to be wrong together.
			if len(a) != len(b) && got {
				t.Fatalf("HexEqual(%q, %q) matched operands of length %d and %d", a, b, len(a), len(b))
			}
		}
	}
}

func TestHexEqualNoEarlyExit(t *testing.T) {
	t.Parallel()
	// A behavioural net for the constant-time rewrite: a mismatch at the
	// first byte and at the last byte are both detected. The timing property
	// itself is documented by BenchmarkHexEqualEarlyDiff /
	// BenchmarkHexEqualLateDiff — timing is not reliably assertable in CI and
	// must not become a flaky gate.
	base := strings.Repeat("a", 64)
	early := "b" + base[1:]
	late := base[:63] + "b"
	if HexEqual(base, early) {
		t.Error("mismatch at offset 0 should not compare equal")
	}
	if HexEqual(base, late) {
		t.Error("mismatch at offset 63 should not compare equal")
	}
	if !HexEqual(base, base) {
		t.Error("identical 64-char digests should compare equal")
	}
}

func TestBytesEqual(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name string
		a, b []byte
		want bool
	}{
		{"equal", []byte{0x01, 0x02, 0x03}, []byte{0x01, 0x02, 0x03}, true},
		{"unequal same length", []byte{0x01, 0x02, 0x03}, []byte{0x01, 0x02, 0x04}, false},
		// Both orders of every asymmetric row: a truncating comparison would
		// reject one direction and accept the other, and a table that only
		// covers one direction cannot tell the two apart.
		{"length mismatch, prefix first", []byte{0x01, 0x02}, []byte{0x01, 0x02, 0x03}, false},
		{"length mismatch, prefix second", []byte{0x01, 0x02, 0x03}, []byte{0x01, 0x02}, false},
		{"both empty", []byte{}, []byte{}, true},
		{"nil and empty", nil, []byte{}, true},
		{"empty and nil", []byte{}, nil, true},
		{"nil and non-empty", nil, []byte{0x00}, false},
		{"non-empty and nil", []byte{0x00}, nil, false},
	}
	for _, tc := range cases {
		if got := BytesEqual(tc.a, tc.b); got != tc.want {
			t.Errorf("%s: BytesEqual = %v, want %v", tc.name, got, tc.want)
		}
	}
}

// hexOfLen builds a lowercase-hex string of exactly n characters.
func hexOfLen(n int) string {
	return strings.Repeat("0123456789abcdef", n/16+1)[:n]
}

func TestValidateClaimsHashRegistryMatchesSpec(t *testing.T) {
	t.Parallel()
	// The twelve registered algorithms of Appendix E.11, verbatim and in the
	// whitepaper's order. A thirteenth entry, a rename, or a dropped entry is
	// a wire-contract change, not a refactor.
	want := []string{
		"md5", "sha1", "sha224", "sha256", "sha384", "sha512",
		"sha3_224", "sha3_256", "sha3_384", "sha3_512", "blake2s", "blake2b",
	}
	got := make([]string, 0, len(hashTypes))
	for name := range hashTypes {
		got = append(got, name)
	}
	sort.Strings(got)
	sorted := append([]string(nil), want...)
	sort.Strings(sorted)
	if strings.Join(got, ",") != strings.Join(sorted, ",") {
		t.Errorf("registered hash types: got [%s], want [%s]", strings.Join(got, ","), strings.Join(sorted, ","))
	}
}

func TestValidateClaimsHashAcceptsLowercase(t *testing.T) {
	t.Parallel()
	// One valid lowercase digest per registered type at its exact width.
	widths := map[string]int{
		"md5": 32, "sha1": 40, "sha224": 56, "sha256": 64,
		"sha384": 96, "sha512": 128, "sha3_224": 56, "sha3_256": 64,
		"sha3_384": 96, "sha3_512": 128, "blake2s": 64, "blake2b": 128,
	}
	if len(widths) != len(hashTypes) {
		t.Fatalf("width table covers %d types, registry has %d", len(widths), len(hashTypes))
	}
	for hashType, width := range widths {
		if err := ValidateClaimsHash(hexOfLen(width), hashType); err != nil {
			t.Errorf("%s: unexpected error for a valid %d-char lowercase digest: %s", hashType, width, err)
		}
	}
}

func TestValidateClaimsHashCharset(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name string
		hash string
	}{
		{"all z", strings.Repeat("z", 64)},
		{"uppercase hex", strings.ToUpper(hexOfLen(64))},
		{"punctuation", hexOfLen(63) + "!"},
		{"leading uppercase", "A" + hexOfLen(63)},
	}
	for _, tc := range cases {
		err := ValidateClaimsHash(tc.hash, "sha256")
		if err == nil {
			t.Errorf("%s: expected a charset error, got nil", tc.name)
			continue
		}
		if !strings.Contains(err.Error(), "lowercase hex") {
			t.Errorf("%s: error should name the lowercase-hex requirement, got %q", tc.name, err)
		}
	}
}

func TestValidateClaimsHashLengthStillChecked(t *testing.T) {
	t.Parallel()
	// Short AND long. A guard written as "at least the expected width" passes
	// the short row and lets a padded digest through, and vice versa.
	cases := []struct {
		name string
		hash string
	}{
		{"too short", "deadbeef"},
		{"one char short", hexOfLen(63)},
		{"one char long", hexOfLen(65)},
		{"sha512-width value under sha256", hexOfLen(128)},
	}
	for _, tc := range cases {
		err := ValidateClaimsHash(tc.hash, "sha256")
		if err == nil {
			t.Errorf("%s: expected a length error for a %d-char sha256 hash", tc.name, len(tc.hash))
			continue
		}
		if !strings.Contains(err.Error(), "64 hex characters") {
			t.Errorf("%s: error should name the expected width, got %q", tc.name, err)
		}
	}
}

func TestValidateClaimsHashEmptyInputs(t *testing.T) {
	t.Parallel()
	for _, tc := range [][2]string{{"", ""}, {"abc", ""}, {"", "sha256"}} {
		if err := ValidateClaimsHash(tc[0], tc[1]); err != nil {
			t.Errorf("ValidateClaimsHash(%q, %q) = %s, want nil", tc[0], tc[1], err)
		}
	}
}

func TestValidateClaimsHashUnknownType(t *testing.T) {
	t.Parallel()
	err := ValidateClaimsHash(hexOfLen(64), "sha999")
	if err == nil {
		t.Fatal("expected an error for an unregistered hash type")
	}
	if !strings.Contains(err.Error(), "unknown hash type") {
		t.Errorf("error should name the unknown type, got %q", err)
	}
}

func TestComputeItemHash_InvalidClaimsHex(t *testing.T) {
	t.Parallel()
	_, err := ComputeItemHash("id", "ZZZZ", "aabb", "ccdd")
	if err == nil {
		t.Error("expected error for invalid claims_hash hex")
	}
}

func TestComputeItemHash_InvalidMetadataHex(t *testing.T) {
	t.Parallel()
	_, err := ComputeItemHash("id", "aabb", "ZZZZ", "ccdd")
	if err == nil {
		t.Error("expected error for invalid metadata_hash hex")
	}
}

func TestComputeItemHash_InvalidSigningKeyIDHex(t *testing.T) {
	t.Parallel()
	_, err := ComputeItemHash("id", "aabb", "ccdd", "ZZZZ")
	if err == nil {
		t.Error("expected error for invalid signing_key_id hex")
	}
}

func TestComputeBlockHash_InvalidPrevHash(t *testing.T) {
	t.Parallel()
	// Field order: id, prevHashHex, merkleRootHex, metadataHashHex, signingKeyIDHex
	_, err := ComputeBlockHash("id", "ZZZZ", "aabb", "ccdd", "eeff")
	if err == nil {
		t.Error("expected error for invalid previous_block_hash hex")
	}
}

func TestComputeBlockHash_InvalidMerkleRoot(t *testing.T) {
	t.Parallel()
	// Field order: id, prevHashHex, merkleRootHex, metadataHashHex, signingKeyIDHex
	_, err := ComputeBlockHash("id", "aabb", "ZZZZ", "ccdd", "eeff")
	if err == nil {
		t.Error("expected error for invalid merkle_root hex")
	}
}

func TestComputeBlockHash_InvalidMetadataHash(t *testing.T) {
	t.Parallel()
	// Field order: id, prevHashHex, merkleRootHex, metadataHashHex, signingKeyIDHex
	_, err := ComputeBlockHash("id", "aabb", "ccdd", "ZZZZ", "eeff")
	if err == nil {
		t.Error("expected error for invalid metadata_hash hex")
	}
}

func TestComputeBlockHash_InvalidSigningKeyID(t *testing.T) {
	t.Parallel()
	// Field order: id, prevHashHex, merkleRootHex, metadataHashHex, signingKeyIDHex
	_, err := ComputeBlockHash("id", "aabb", "ccdd", "eeff", "ZZZZ")
	if err == nil {
		t.Error("expected error for invalid signing_key_id hex")
	}
}

func TestComputeEntropyHash(t *testing.T) {
	t.Parallel()
	// Test vector: SHA256(0x21 || "truestamp") from hash_test.go's DomainHash test
	result := ComputeEntropyHash([]byte("truestamp"))
	expected := "22da1f4e92cfd2318f0cec8c3b8f7bf3c7c2531db0e5ac781f25e1e25ae65831"
	if result != expected {
		t.Errorf("ComputeEntropyHash: got %s, want %s", result, expected)
	}
}

func TestComputeEntropyHash_Deterministic(t *testing.T) {
	t.Parallel()
	data := []byte(`{"pulse":{"outputValue":"ABC123"}}`)
	r1 := ComputeEntropyHash(data)
	r2 := ComputeEntropyHash(data)
	if r1 != r2 {
		t.Error("ComputeEntropyHash should be deterministic")
	}
	if len(r1) != 64 {
		t.Errorf("ComputeEntropyHash should return 64 hex chars, got %d", len(r1))
	}
}

func TestComputeEntropyHash_DifferentFromClaims(t *testing.T) {
	t.Parallel()
	data := []byte("same input")
	entropy := ComputeEntropyHash(data)
	claims := BytesToHex(DomainHash(PrefixItemClaims, data))
	if entropy == claims {
		t.Error("entropy hash (0x21) and claims hash (0x11) should differ for same input")
	}
}

func TestComputeObservationHash(t *testing.T) {
	t.Parallel()
	// ComputeObservationHash takes id, entropyHashHex, metadataHashHex, signingKeyIDHex
	result, err := ComputeObservationHash(
		"019cf813-99b8-730a-84f1-5a711a9c355e",
		"22da1f4e92cfd2318f0cec8c3b8f7bf3c7c2531db0e5ac781f25e1e25ae65831",
		"041c99740c6426282271e9341129e75393ed864575ab403a85e8120eed361661",
		"4ceefa4a",
	)
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	if len(result) != 64 {
		t.Errorf("observation_hash should be 64 hex chars, got %d", len(result))
	}
	// Same inputs should produce the same hash (deterministic)
	result2, _ := ComputeObservationHash(
		"019cf813-99b8-730a-84f1-5a711a9c355e",
		"22da1f4e92cfd2318f0cec8c3b8f7bf3c7c2531db0e5ac781f25e1e25ae65831",
		"041c99740c6426282271e9341129e75393ed864575ab403a85e8120eed361661",
		"4ceefa4a",
	)
	if result != result2 {
		t.Error("observation_hash should be deterministic")
	}
	// Different ID should produce a different hash
	result3, _ := ComputeObservationHash(
		"019cf813-99b8-730a-84f1-000000000000",
		"22da1f4e92cfd2318f0cec8c3b8f7bf3c7c2531db0e5ac781f25e1e25ae65831",
		"041c99740c6426282271e9341129e75393ed864575ab403a85e8120eed361661",
		"4ceefa4a",
	)
	if result == result3 {
		t.Error("different ID should produce different observation_hash")
	}
}

func TestComputeObservationHash_InvalidEntropyHex(t *testing.T) {
	t.Parallel()
	_, err := ComputeObservationHash("id", "ZZZZ", "aabb", "ccdd")
	if err == nil {
		t.Error("expected error for invalid entropy_hash hex")
	}
}

func TestComputeObservationHash_InvalidMetadataHex(t *testing.T) {
	t.Parallel()
	_, err := ComputeObservationHash("id", "aabb", "ZZZZ", "ccdd")
	if err == nil {
		t.Error("expected error for invalid metadata_hash hex")
	}
}

func TestComputeObservationHash_InvalidSigningKeyIDHex(t *testing.T) {
	t.Parallel()
	_, err := ComputeObservationHash("id", "aabb", "ccdd", "ZZZZ")
	if err == nil {
		t.Error("expected error for invalid signing_key_id hex")
	}
}

func TestComputeObservationHash_DifferentFromItemHash(t *testing.T) {
	t.Parallel()
	// Same field values but different domain prefixes (0x23 vs 0x13) should differ
	obsHash, _ := ComputeObservationHash("id", "aabb", "ccdd", "eeff")
	itemHash, _ := ComputeItemHash("id", "aabb", "ccdd", "eeff")
	if obsHash == itemHash {
		t.Error("observation_hash (0x23) and item_hash (0x13) should differ for same fields")
	}
}
