// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: MIT

package verify

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"strings"
	"testing"

	"github.com/truestamp/truestamp-cli/internal/jcs"
	"github.com/truestamp/truestamp-cli/internal/proof"
	"github.com/truestamp/truestamp-cli/internal/testfixtures"
	"github.com/truestamp/truestamp-cli/internal/tscrypto"
)

// The Appendix C source vectors (testdata/whitepaper/vectors.txt) and the
// Appendix D derivation trace (testdata/whitepaper/derivation.txt), turned
// into known-answer tests. Every value below is copied from those files.

func hexOf(b []byte) string { return hex.EncodeToString(b) }

func mustHex(t *testing.T, s string) []byte {
	t.Helper()
	b, err := hex.DecodeString(s)
	if err != nil {
		t.Fatal(err)
	}
	return b
}

// --- C.4 sentinels ---

func TestVectors_C4_Sentinels(t *testing.T) {
	if got := hexOf(sha256.New().Sum(nil)); got != "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855" {
		t.Errorf("empty tree root = %s", got)
	}
	emptyLeaf := tscrypto.DomainHash(tscrypto.PrefixMerkleLeaf, []byte{0x00})
	if hexOf(emptyLeaf) != "96a296d224f285c67bee93c30f8a309157f0daa35dc5b87e410b78630a09cfc7" {
		t.Errorf("empty leaf input = %s", hexOf(emptyLeaf))
	}
	if got := hexOf(tscrypto.DomainHash(tscrypto.PrefixMerkleLeaf, emptyLeaf)); got != "d37300dc2c6e038a83ee197ca0e181a77f6875afd9f537d31ca4995876481319" {
		t.Errorf("padding leaf hash = %s", got)
	}
	if got := tscrypto.ComputeBlockMetadataHash([]byte("{}")); got != "14fe55ee4ce3cdbc8118a0a28e5a80a44f1f8a24d73d9949c0ecd91ee582ebe1" {
		t.Errorf("empty block metadata 0x33 = %s", got)
	}
	if got := tscrypto.ComputeEntropyMetadataHash([]byte("{}")); got != "5dd81d4309de99c9f8e70822e760f0eece8c33370ff5fb5da3af75bab0cbaab8" {
		t.Errorf("empty entropy metadata 0x22 = %s", got)
	}
	genesis := "truestamp-blockchain-genesis-block-2026-08-04-btc-960940-000000000000000000017abaa21c7ae0cfecd6b8da996e78c1c2141056a29844"
	if got := hexOf(tscrypto.DomainHash(0x31, []byte(genesis))); got != "96118892a96ced7df9875cca4b09f5c867728f63adcea4f189acc2f38fe75c95" {
		t.Errorf("genesis previous block hash (0x31) = %s", got)
	}
}

// --- C.2a JCS conformance vectors ---

func TestVectors_C2a_JCS(t *testing.T) {
	cases := []struct{ name, input, jcs, hash string }{
		{"explicit null", `{"b":1,"a":null}`, `{"a":null,"b":1}`, "13ec6081eab8b87a582dc6c2d0f6ff561d79de8be10c652440b53e1b72a95099"},
		{"control character and tab", `{"tab":"\t","c":"\u0001"}`, `{"c":"\u0001","tab":"\t"}`, "66d5ba7776a99a71bc63007419530297a231043bd74de64e9ef2b7b44f3deec2"},
		{"negative zero", `{"z":-0.0}`, `{"z":0}`, "901865b9e33220a2601150ae2f202ad45d172458c8eb5865ccfd695c353e2430"},
		{"exponential floats", `{"small":0.0000001,"big":1000000000000000000000.0}`, `{"big":1e+21,"small":1e-7}`, "2f01621ad310054550edf66468a6db72601e6e983464fb594cb884585beef1c7"},
		{"non-ASCII emitted unescaped", `{"name":"Ávila","emoji":"😀"}`, `{"emoji":"😀","name":"Ávila"}`, "65a18fbdd50f042c67de68433ede20023a529af601099cd735f4dd4805845eef"},
		{"astral-plane key ordering", `{"￿":2,"😀":1}`, `{"😀":1,"￿":2}`, "d0951cfce65f8af38e98d67bd15833bb89afa2163e1b4c707047c126d53a6710"},
		{"2^53 (exactly representable)", `{"n":9007199254740992}`, `{"n":9007199254740992}`, "c2c43fad296f3c57c36919c193d533f212f5aeb88431960512c87d3db2a49fe6"},
		{"2^53 + 1 (not a double)", `{"n":9007199254740993}`, `{"n":9007199254740993}`, "f49f563152ef5c20c0aee41405be6342de6fbbb201985d7adc5bfebda9382407"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			canonical, oversized, err := jcs.Canonicalize([]byte(tc.input))
			if err != nil {
				t.Fatal(err)
			}
			if string(canonical) != tc.jcs {
				t.Errorf("JCS = %s, want %s", canonical, tc.jcs)
			}
			if got := tscrypto.ComputeClaimsHash(canonical); got != tc.hash {
				t.Errorf("0x11 = %s, want %s", got, tc.hash)
			}
			if strings.Contains(tc.name, "2^53 + 1") != (len(oversized) > 0) {
				t.Errorf("oversized = %v", oversized)
			}
		})
	}
}

// --- C.3 single-value domain hashes ---

const (
	illustrativePK    = "IVL40Zt5HSRFMkLhXy6rbLfP+ntqXtMAl5YOBpiB2xI="
	illustrativeKeyID = "f2c39df9"
)

func TestVectors_C3_DomainHashes(t *testing.T) {
	if got := hexOf(tscrypto.DomainHash(tscrypto.PrefixItemClaims, []byte("test data"))); got != "4b72c11e5d1192e35529c75ce8cff08aacb1e2186735a8e51c6a1ebfc9057d9b" {
		t.Errorf("0x11 test data = %s", got)
	}
	if got := hexOf(tscrypto.DomainHash(tscrypto.PrefixEntropy, []byte("entropy data"))); got != "439c7a900a9e3ba59b94a7e8583b628384ee1084acddd2cda3a4fe256ad716d0" {
		t.Errorf("0x21 entropy data = %s", got)
	}
	canonical, _, err := jcs.Canonicalize([]byte(`{"name":"Alice","age":30}`))
	if err != nil || string(canonical) != `{"age":30,"name":"Alice"}` {
		t.Errorf("JCS = %s (%v)", canonical, err)
	}
	raw := sha256.Sum256(canonical)
	if hexOf(raw[:]) != "1b3d1428a344b4a5f9e479024cd76449100aec4c3ace922a317dd45696412f77" {
		t.Errorf("raw sha256 = %s", hexOf(raw[:]))
	}
	nist := `{"chainIndex":1,"outputValue":"abababababababababababababababababababababababababababababababababababababababababababababababababababababababababababababababab","pulseIndex":2847561,"uri":"nist:1:2847561"}`
	if got := tscrypto.ComputeEntropyHash([]byte(nist)); got != "976ff222abddaf00d26304759f1145f7cafb3cb9f8aaeebaa5767cf502198e5c" {
		t.Errorf("0x21 nist example = %s", got)
	}
	pk, err := tscrypto.DecodePublicKey(illustrativePK)
	if err != nil {
		t.Fatal(err)
	}
	if got := tscrypto.ComputeKeyID(pk); got != illustrativeKeyID {
		t.Errorf("0x51 key id = %s", got)
	}
	if got := hexOf(tscrypto.DomainHash(0x53, pk)); got != "744989d95b39fe40f74249e31e3584a5e5d28f09d12b0809a49b4f1dc2ff1e2b" {
		t.Errorf("0x53 prerotation = %s", got)
	}
}

// --- C.3a witnesses and C.5 composites, from the Appendix D bundle ---

func appendixD(t *testing.T) *proof.Bundle {
	t.Helper()
	b, err := proof.ParseBytes(testfixtures.Read(t, testfixtures.WhitepaperDir, testfixtures.AppendixD))
	if err != nil {
		t.Fatal(err)
	}
	return b
}

func TestVectors_C3a_C5_AppendixD(t *testing.T) {
	b := appendixD(t)
	s := b.Subject

	claims, _, err := jcs.Canonicalize(s.Claims)
	if err != nil {
		t.Fatal(err)
	}
	if string(claims) != `{"description":"Illustrative item for the Truestamp whitepaper. Not a production record.","hash":"b47cc0f104b62d4c7c30bcd68fd8e67613e287dc4ad8c310ef10cbadea9c4380","hash_type":"sha256","name":"Appendix D worked example"}` {
		t.Errorf("JCS(claims) = %s", claims)
	}
	claimsHash := tscrypto.ComputeClaimsHash(claims)
	if claimsHash != "ae5cdc73a52359a4fb0e335f004a6cd0cb1247024e812263e362368915a4b924" {
		t.Errorf("claims_hash = %s", claimsHash)
	}
	meta, _, _ := jcs.Canonicalize(s.Metadata)
	metaHash := tscrypto.ComputeItemMetadataHash(meta)
	if metaHash != "813f13eb570f99f533500f4aba9d6e93c012bff95d3bb18c4720d8a22ece8cb6" {
		t.Errorf("metadata_hash = %s", metaHash)
	}
	itemHash, err := tscrypto.ComputeItemHash(s.ID, claimsHash, metaHash, s.SigningKeyID)
	if err != nil || itemHash != "d186ed38f08b5fc2e41dde049d2bf013823e927f206fbed3b47d4e4955b67465" {
		t.Errorf("item_hash = %s (%v)", itemHash, err)
	}

	// Witness hashes recomputed from the carried details.
	committed := s.CommittedWitnesses()
	for _, name := range []string{"entropy_stellar", "entropy_nist", "entropy_bitcoin"} {
		canonical, _, err := jcs.Canonicalize(s.Witnesses[name])
		if err != nil {
			t.Fatal(err)
		}
		if got := tscrypto.ComputeEntropyHash(canonical); got != committed.Str(name) {
			t.Errorf("%s witness = %s, committed %s", name, got, committed.Str(name))
		}
	}
	head := parseBlockMap(s.Witnesses["block"])
	headHash, size, reason := blockHashFromMap(head)
	if reason != "" || size != 157 || headHash != "f6912793a49f289affc65a9571993444ae33760307ee973d37a8f90ea3b2d8e9" {
		t.Errorf("block witness = %s size %d (%s)", headHash, size, reason)
	}
	if headHash != b.Block.PreviousBlockHash {
		t.Error("head block is not the containing block's predecessor")
	}

	// C.5 observation hash: the block leaf for the NIST observation carries
	// the C.3 NIST example's entropy hash and an empty metadata map.
	obs, err := tscrypto.ComputeObservationHash("019f93fe-3c04-70a1-8000-0000000000a1",
		"976ff222abddaf00d26304759f1145f7cafb3cb9f8aaeebaa5767cf502198e5c",
		tscrypto.ComputeEntropyMetadataHash([]byte("{}")), illustrativeKeyID)
	if err != nil || obs != "e4e5bf625168c678a5b15fc911c4714cade1c4b5e62f2b14e430610cd0808b9b" {
		t.Errorf("observation_hash = %s (%v)", obs, err)
	}

	// Block hash.
	blockHash, size, reason := blockHashFromMap(b.Block)
	if reason != "" || size != 157 || blockHash != "806ef5083f475ed7e2f2f3fc15d2d12d842f2c43686caab767e226664910a7c9" {
		t.Errorf("block_hash = %s size %d (%s)", blockHash, size, reason)
	}

	// Key event block.
	keHash, _, reason := blockHashFromMap(b.SigningKeyEvent.Block)
	if reason != "" || keHash != "547624cfa69c137785e74086ebae12737bacfbb32b35f9f35724365f3bacd6da" {
		t.Errorf("key event block hash = %s (%s)", keHash, reason)
	}
	keMeta, _, _ := jcs.Canonicalize(b.SigningKeyEvent.Block.Metadata)
	if string(keMeta) != `{"key_event":{"key_id":"f2c39df9","prerotation_commitment":null,"public_key":"IVL40Zt5HSRFMkLhXy6rbLfP+ntqXtMAl5YOBpiB2xI=","sequence":0,"type":"genesis"}}` {
		t.Errorf("JCS(key event metadata) = %s", keMeta)
	}
}

// --- D.3 proofs, payload and signature ---

func TestVectors_D3_ProofsAndSignature(t *testing.T) {
	b := appendixD(t)
	itemHash := "d186ed38f08b5fc2e41dde049d2bf013823e927f206fbed3b47d4e4955b67465"
	blockHash := "806ef5083f475ed7e2f2f3fc15d2d12d842f2c43686caab767e226664910a7c9"

	steps, err := tscrypto.DecodeCompactMerkleProof(b.InclusionProof)
	if err != nil {
		t.Fatal(err)
	}
	wantSteps := []string{"r:01a4bde69bd17911f48c5142e20a9974665bfe8c4bc28c9d3652b80cb69032aa", "l:3ad6603cccba87eb7a5bcfe89a8e3d511e9f5559ef39c74de3158601736aa614"}
	if strings.Join(steps, ",") != strings.Join(wantSteps, ",") {
		t.Errorf("inclusion steps = %v", steps)
	}
	raw, _ := base64.RawURLEncoding.DecodeString(b.InclusionProof)
	if hexOf(raw) != "020101a4bde69bd17911f48c5142e20a9974665bfe8c4bc28c9d3652b80cb69032aa3ad6603cccba87eb7a5bcfe89a8e3d511e9f5559ef39c74de3158601736aa614" {
		t.Errorf("compact bytes = %s", hexOf(raw))
	}
	root, err := tscrypto.WalkMerkleProof(itemHash, steps)
	if err != nil || root != "7c40e8809aafdbc946bdc51484dc0f76811350ce4855744edfdcc5d62442b98c" || root != b.Block.MerkleRoot {
		t.Errorf("merkle_root = %s (%v)", root, err)
	}

	epochs := []struct {
		chain, root string
		steps       []string
	}{
		{"stellar", "f5c2df8d5fdc24b0aef14c7ab3813c46c6d17c0545d9422f9272fe6a2e672e46", []string{"l:235ee977773d2b7cf21ee00a1c8a58ef84f82dbf37e738019afa58e173a27df0", "r:47633684d35b1e312d815e7a90f7c35c20e9492ab45767a22550f780775f242c"}},
		{"bitcoin", "d1526bd295f222761a81bef1389ee0108bc133db0c66dff03a9c617f6ff2604e", []string{"r:48d22956ed7700d0637bbc6f91e55ec7583359a74b9d91b202086fdb9983198b"}},
	}
	for i, e := range epochs {
		c := b.Commitments[i]
		if c.Chain != e.chain || c.EpochMerkleRoot != e.root {
			t.Errorf("commitment %d = %s %s", i, c.Chain, c.EpochMerkleRoot)
		}
		steps, err := tscrypto.DecodeCompactMerkleProof(c.EpochProof)
		if err != nil || strings.Join(steps, ",") != strings.Join(e.steps, ",") {
			t.Errorf("%s epoch steps = %v (%v)", e.chain, steps, err)
		}
		if root, err := tscrypto.WalkMerkleProof(blockHash, steps); err != nil || root != e.root {
			t.Errorf("%s epoch root = %s (%v)", e.chain, root, err)
		}
	}
	ke := b.SigningKeyEvent.Commitments[0]
	steps, _ = tscrypto.DecodeCompactMerkleProof(ke.EpochProof)
	if root, err := tscrypto.WalkMerkleProof("547624cfa69c137785e74086ebae12737bacfbb32b35f9f35724365f3bacd6da", steps); err != nil || root != "5ee58a111980722fb7bf2cb4dfd54697188bd3d12a6bfb2e2c480113a09f4c0f" {
		t.Errorf("key event epoch root = %s (%v)", root, err)
	}

	// The 145-byte payload and its 0x61 digest.
	tsMs, _ := isoMs(b.GeneratedAt)
	proofHash, err := tscrypto.BuildCompactProofPayload(1, 20, illustrativeKeyID, uint64(tsMs), itemHash, blockHash,
		[]string{epochs[0].root, epochs[1].root})
	if err != nil {
		t.Fatal(err)
	}
	if hexOf(proofHash) != "d0262b93c8015c31f5617a127f5c043d5e9527d2d37709bf813728213238d788" {
		t.Errorf("proof_hash = %s", hexOf(proofHash))
	}
	payload := "010014f2c39df90000019f93ff2600" + itemHash + blockHash + "0002" + epochs[0].root + epochs[1].root
	if len(mustHex(t, payload)) != 145 {
		t.Errorf("payload is %d bytes", len(mustHex(t, payload)))
	}
	if got := hexOf(tscrypto.DomainHash(tscrypto.PrefixProofHash, mustHex(t, payload))); got != hexOf(proofHash) {
		t.Errorf("payload hex does not hash to the proof hash: %s", got)
	}
	pk, _ := tscrypto.DecodePublicKey(b.PublicKey)
	if ok, err := tscrypto.VerifyEd25519(proofHash, b.Signature, pk); err != nil || !ok {
		t.Errorf("ed25519 verify = %v (%v)", ok, err)
	}
}

// --- C.6 identifiers ---

func TestVectors_C6_Identifiers(t *testing.T) {
	if ms, ok := ulidMs("01KY9ZWEX0248J48HK6D248NAN"); !ok || ms != 1784894340000 {
		t.Errorf("ULID ms = %d ok=%v", ms, ok)
	}
	cases := map[string]int64{
		"019f93ff-2600-7c30-8000-000000000c30": 1784894400000,
		"019f93fd-c670-7001-8000-000000000001": 1784894310000,
		"019f93fe-3c04-70a1-8000-0000000000a1": 1784894340100,
		"019f9403-b9e0-70e1-8000-0000000000e1": 1784894700000,
		"019f9404-a440-70f1-8000-0000000000f1": 1784894760000,
	}
	for id, want := range cases {
		if ms, ok := uuidv7Ms(id); !ok || ms != want {
			t.Errorf("UUIDv7 %s ms = %d ok=%v, want %d", id, ms, ok, want)
		}
	}
}

// --- C.7 Merkle trees ---

// leafHash is the C.7 leaf rule: sha256("leaf<i>"), before the 0x00 leaf
// prefix the walk applies.
func leafHash(i int) string {
	h := sha256.Sum256([]byte(fmt.Sprintf("leaf%d", i)))
	return hexOf(h[:])
}

func TestVectors_C7_SmallTrees(t *testing.T) {
	cases := []struct {
		n     int
		root  string
		proof string
	}{
		{1, "e6f3e0324c47532b4584166b9cdcbfb5f1dceaac9b097512d0e9e8501977daa0", "AA"},
		{2, "5d3d9c89b11a0055ba0e43c2aaf4d3814717c01a8079bc1d05db80c41852b0f5", "AQHXisvDVvoXHOQLty_6dMveBsNq78JniprxjTl1WB6Wnw"},
		{3, "738707d8051d65bb5b11d36cac93f7e5dccdee4676e836800a5d4e5c444103f1", "AgPXisvDVvoXHOQLty_6dMveBsNq78JniprxjTl1WB6WnwpaiKaXUNgr2l7zaXej_e_K07FvdZzDBGR8oS9jdqZT"},
		{4, "1d8219ac8846f635dab3201c241583de32a73ca2f1b361cec04a419ae7806324", "AgPXisvDVvoXHOQLty_6dMveBsNq78JniprxjTl1WB6Wn9L6rrJFX-bbqgjmTqwuSTXNK8ySEpNot-suxdxUYLdR"},
		{5, "2012533b81a14bd8c0ba9172d14ca4bd761449bf10065c5baf89bc487497e891", "AwfXisvDVvoXHOQLty_6dMveBsNq78JniprxjTl1WB6Wn9L6rrJFX-bbqgjmTqwuSTXNK8ySEpNot-suxdxUYLdRvwKRsLzm6CsEBq1W6ZUGsz229IXvpMBBgHFE5jkej8M"},
		{6, "148337559cd25959c1c5f80dc1b0518a05ec308fbff4db47f2e133a67234f898", "AwfXisvDVvoXHOQLty_6dMveBsNq78JniprxjTl1WB6Wn9L6rrJFX-bbqgjmTqwuSTXNK8ySEpNot-suxdxUYLdRyFjVrlVrv68NB3q3LhpsY8fK7cZ-gIpz6RQ_JjthXuA"},
		{7, "a08d49ae18a1ad35c5925064aa8fba66a6dfdcc24cd1ba9047d0e87493230ed4", "AwfXisvDVvoXHOQLty_6dMveBsNq78JniprxjTl1WB6Wn9L6rrJFX-bbqgjmTqwuSTXNK8ySEpNot-suxdxUYLdRTiYYWTzoEDuqwvXG_S9skQvk2SroR8-RqaLFV0sxnVQ"},
	}
	for _, tc := range cases {
		steps, err := tscrypto.DecodeCompactMerkleProof(tc.proof)
		if err != nil {
			t.Errorf("n=%d: decode: %v", tc.n, err)
			continue
		}
		root, err := tscrypto.WalkMerkleProof(leafHash(1), steps)
		if err != nil || root != tc.root {
			t.Errorf("n=%d: root = %s (%v), want %s", tc.n, root, err, tc.root)
		}
	}
	// The RFC 6962 comparison roots differ at non-power-of-two counts,
	// which is the tree shape's business, never the walk's.
}

// TestVectors_C7_LargeTree decodes the 300-leaf depth-9 proof (the exact
// size equation, the little-endian direction field, the base64url form
// against the hex form) and walks it to the published root. The large
// tree's leaf rule differs from the small trees': keys lk0001..lk0300 with
// leaf value sha256("bigleaf<i>"), i unpadded, so lk0001 is
// sha256("bigleaf1"). Byte-wise key sort puts lk0001 first and the tree pads
// 300 leaves to 512, which is why the proof is depth 9.
func TestVectors_C7_LargeTree(t *testing.T) {
	b64 := "Cf8BeuVHNzEnTcggyDPBTAF_qqu0C0oYJJUcFcja4jakVDn3XyFH7-jCmizUb2PZ3p68hhxyTmX-MywPDndtC7lItPBfUT_1BBVZnrrT2ARAWXUZ8M_HggW4ab8nXDexR_dLEOcutSukTdiWylDM_pBLplODqhPt1eU5nIkcpwfB4NDV-jH5_tYZe92GxfPj5R_qshEwnlyTDm4qUBELCH6AzVqO52d4CjcnARxMagti8LlHXlk-zu3YOBDUHa5mDD4dQdkppuiQP4zCGVIfkDEdqdV9RpsBfHtXhSU9AacT_BRtaVGStjUrqo8gejyJAmTlO9G8pNP4MSn0fle7DWvRGDyd0ExsR_vkjy4xWYoEcmoY7L9lFGc5DdLgIX1PWYwb"
	raw, err := base64.RawURLEncoding.DecodeString(b64)
	if err != nil {
		t.Fatal(err)
	}
	if len(raw) != 291 || raw[0] != 9 {
		t.Errorf("proof is %d bytes, depth %d", len(raw), raw[0])
	}
	if !strings.HasPrefix(hexOf(raw), "09ff017ae547") {
		t.Errorf("hex form = %s", hexOf(raw)[:12])
	}
	steps, err := tscrypto.DecodeCompactMerkleProof(b64)
	if err != nil || len(steps) != 9 {
		t.Fatalf("decode: %d steps (%v)", len(steps), err)
	}
	// Direction field ff 01, little-endian: bits 0..7 set (right), bit 8
	// set (right) too. A big-endian or top-first reading disagrees.
	for i, s := range steps {
		if !strings.HasPrefix(s, "r:") {
			t.Errorf("step %d direction = %s, want right", i, s[:2])
		}
	}
	leaf := sha256.Sum256([]byte("bigleaf1"))
	if hexOf(leaf[:]) != "668799785ad06e8d58ba2dbe5c25f43909b131f594953211235e251b2d07d2c8" {
		t.Errorf("lk0001 leaf value = %s", hexOf(leaf[:]))
	}
	root, err := tscrypto.WalkMerkleProof(hexOf(leaf[:]), steps)
	if err != nil || root != "f8c3f9a207a67fc22a297edcdf1dc0f17840ee9c8648ee3c8a7e5a71e5e42b92" {
		t.Errorf("300-leaf root = %s (%v)", root, err)
	}
	// A big-endian or top-first reading of the direction field passes every
	// depth-8 proof and fails here: reversing the steps must not reach the
	// root.
	reversed := make([]string, len(steps))
	for i := range steps {
		reversed[len(steps)-1-i] = steps[i]
	}
	if wrong, _ := tscrypto.WalkMerkleProof(hexOf(leaf[:]), reversed); wrong == root {
		t.Error("a top-first reading of the direction field must not reproduce the root")
	}
}
