// Copyright (c) 2021-2026 Truestamp, Inc.
// SPDX-License-Identifier: MIT

package bitcoin

import (
	"testing"

	"github.com/btcsuite/btcd/chaincfg/chainhash"
)

// Bitcoin's parsers operate on attacker-controlled hex strings pulled
// from proof bundles (`rtx`, `txp`). Any panic here would crash the
// CLI during `verify` on a bad proof, so the parsers are a high-value
// fuzz target.

func FuzzDecodeTxOutProof(f *testing.F) {
	for _, s := range []string{
		"",
		"00",
		"not-hex",
		// A realistic partial-merkle-block header + small tree.
		"000000203f9b114c953f5013001d1d010038b1904f9245108f4780e833d49f6c813b97183e4b2b567f00bf7d90895b01a06416725a6352bbbbc7bb72f10f6c1dba7ca0d3f58fd569ffff7f200100000002000000020ae2c4d1afd05969065e8abcdbbfd2ece14e39ecf71fc0267b0b43f7eda73fd9ceae064c75c99b3f8d65db991d401fcd4c67990d134fb22b3023f172943cf1e50105",
	} {
		f.Add(s)
	}
	f.Fuzz(func(t *testing.T, s string) {
		_, _ = DecodeTxOutProof(s)
	})
}

func FuzzParseBlockHash(f *testing.F) {
	f.Add("")
	f.Add("00")
	f.Add("000000203f9b114c953f5013001d1d010038b1904f9245108f4780e833d49f6c813b9718")

	f.Fuzz(func(t *testing.T, s string) {
		_, _ = ParseBlockHash(s)
	})
}

func FuzzDecodeTransaction(f *testing.F) {
	for _, s := range []string{
		"",
		"00",
		"not-hex",
		"020000000001018fe1d5b4f19484c0150ae9dffa89ade8a28010c4b031a86929f07a905f16870a0000000000fdffffff020000000000000000226a20309cb8bc7c5493b0275b754b92a61bd877f33d56c8f0a780ad300995898ffbdf61c3000000000000225120d2972046141d369b5eb51ad6fd4efd463c3458e9eb416ecccb521eb5ef38188d00000000",
	} {
		f.Add(s)
	}
	f.Fuzz(func(t *testing.T, s string) {
		_, _ = DecodeTransaction(s)
	})
}

func FuzzExtractOpReturn(f *testing.F) {
	f.Add("")
	f.Add("020000000000000000226a20309cb8bc7c5493b0275b754b92a61bd877f33d56c8f0a780ad300995898ffbdf")

	f.Fuzz(func(t *testing.T, s string) {
		_, _ = ExtractOpReturn(s)
	})
}

func FuzzComputeTxID(f *testing.F) {
	f.Add("")
	f.Add("020000000001018fe1d5b4f19484c0150ae9dffa89ade8a28010c4b031a86929f07a905f16870a0000000000fdffffff00000000")

	f.Fuzz(func(t *testing.T, s string) {
		_, _ = ComputeTxID(s)
	})
}

// FuzzVerifyPartialMerkleTree exercises the in-memory tree walker with
// attacker-controlled counts and hash slices. Uses chainhash.Hash
// arrays directly — the classic territory for slice-bounds panics.
func FuzzVerifyPartialMerkleTree(f *testing.F) {
	f.Add(uint32(2), []byte{0x01}, uint32(64))

	f.Fuzz(func(t *testing.T, totalTxs uint32, flagsRaw []byte, hashCount uint32) {
		// Cap ridiculous sizes so the fuzzer doesn't OOM on a
		// multi-GB slice allocation. These bounds are still well
		// above anything a real proof could carry.
		if hashCount > 1024 {
			return
		}
		if len(flagsRaw) > 4096 {
			return
		}
		hashes := make([]*chainhash.Hash, hashCount)
		for i := range hashes {
			var h chainhash.Hash
			h[0] = byte(i)
			hashes[i] = &h
		}
		var expected chainhash.Hash
		_ = VerifyPartialMerkleTree(hashes, flagsRaw, totalTxs, &expected)
	})
}
