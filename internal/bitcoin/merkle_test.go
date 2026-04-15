// Copyright (c) 2021-2026 Truestamp, Inc.
// SPDX-License-Identifier: MIT

package bitcoin

import (
	"bytes"
	"encoding/hex"
	"testing"

	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/btcsuite/btcd/wire"
)

const testTxoutproofHex = "0000002026cf000a60782b1b57c018b6482e6837c1165d5e2aa06f92d199163481b6c07b150edc6623b16f07e3239de42b6f67d08f5d71b4d06e35685dcaab533f951d5e395ab869ffff7f200000000002000000028205b7caabd985090d0ba024a7d2ce998e53d49122989f8eac68b6f25558bbee52877124271894f27729ada539c4834cfdf4b3972d39a0e189ff108c578ec3290105"
const testExpectedTxid = "29c38e578c10ff89e1a0392d97b3f4fd4c83c439a5ad2977f294182724718752"

func parseMsgMerkleBlock(t *testing.T) *wire.MsgMerkleBlock {
	t.Helper()
	raw, err := hex.DecodeString(testTxoutproofHex)
	if err != nil {
		t.Fatalf("decode hex: %s", err)
	}
	var mb wire.MsgMerkleBlock
	if err := mb.BtcDecode(bytes.NewReader(raw), wire.ProtocolVersion, wire.LatestEncoding); err != nil {
		t.Fatalf("parse merkle block: %s", err)
	}
	return &mb
}

func TestVerifyPartialMerkleTree_FromProof(t *testing.T) {
	mb := parseMsgMerkleBlock(t)

	result := VerifyPartialMerkleTree(
		mb.Hashes,
		mb.Flags,
		mb.Transactions,
		&mb.Header.MerkleRoot,
	)

	if !result.Valid {
		t.Error("partial merkle tree verification should pass")
	}
	if len(result.MatchedTxIDs) == 0 {
		t.Error("should have at least one matched txid")
	}

	expectedTxid, err := chainhash.NewHashFromStr(testExpectedTxid)
	if err != nil {
		t.Fatalf("parse expected txid: %s", err)
	}
	found := false
	for _, m := range result.MatchedTxIDs {
		if m.IsEqual(expectedTxid) {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected txid not found in matched set")
	}
}

func TestVerifyPartialMerkleTree_EmptyTree(t *testing.T) {
	result := VerifyPartialMerkleTree(nil, nil, 0, nil)
	if result.Valid {
		t.Error("empty tree should not be valid")
	}
}

func TestVerifyPartialMerkleTree_SingleTx(t *testing.T) {
	var txHash chainhash.Hash
	txHash[0] = 0xAA

	result := VerifyPartialMerkleTree(
		[]*chainhash.Hash{&txHash},
		[]byte{0x01}, // single flag bit = true
		1,
		&txHash,
	)
	if !result.Valid {
		t.Error("single-tx tree should be valid")
	}
	if len(result.MatchedTxIDs) != 1 {
		t.Errorf("expected 1 matched txid, got %d", len(result.MatchedTxIDs))
	}
}

func TestVerifyPartialMerkleTree_OddTxCount(t *testing.T) {
	var tx1, tx2, tx3 chainhash.Hash
	tx1[0] = 0x11
	tx2[0] = 0x22
	tx3[0] = 0x33

	h01 := combineHashes(&tx1, &tx2)
	h23 := combineHashes(&tx3, &tx3) // duplicate
	root := combineHashes(h01, h23)

	// Partial merkle tree proving tx3 is included:
	// flags: root=1(descend), left=0(hash), right=1(descend), leaf=1(matched)
	result := VerifyPartialMerkleTree(
		[]*chainhash.Hash{h01, &tx3},
		[]byte{0x0D}, // bits: 1,0,1,1 = 0b00001101
		3,
		root,
	)
	if !result.Valid {
		t.Error("odd-count tree should verify")
	}
	if len(result.MatchedTxIDs) != 1 {
		t.Errorf("expected 1 matched txid, got %d", len(result.MatchedTxIDs))
	}
	if !result.MatchedTxIDs[0].IsEqual(&tx3) {
		t.Error("matched txid should be tx3")
	}
}

func TestVerifyPartialMerkleTree_UnconsumedHashes(t *testing.T) {
	var txHash, extra chainhash.Hash
	extra[0] = 0xFF

	result := VerifyPartialMerkleTree(
		[]*chainhash.Hash{&txHash, &extra},
		[]byte{0x01}, // single flag bit = true
		1,
		&txHash,
	)
	if result.Valid {
		t.Error("unconsumed hashes should invalidate the proof")
	}
}

func TestVerifyPartialMerkleTree_WrongRoot(t *testing.T) {
	mb := parseMsgMerkleBlock(t)

	var wrongRoot chainhash.Hash // all zeros
	result := VerifyPartialMerkleTree(
		mb.Hashes,
		mb.Flags,
		mb.Transactions,
		&wrongRoot,
	)

	if result.Valid {
		t.Error("verification with wrong root should fail")
	}
}

func TestVerifyPartialMerkleTree_DuplicateChildHashes(t *testing.T) {
	// BIP 37: two-child internal nodes must not have equal left and right hashes.
	// Construct a 2-tx tree where both leaves are identical (attack scenario).
	var tx chainhash.Hash
	tx[0] = 0xAA

	// With 2 txs: height=1. If both leaves are identical, the root would be
	// Hash(tx||tx). BIP 37 says this must be rejected.
	root := combineHashes(&tx, &tx)

	// Partial tree: root flag=1 (descend), left flag=1 (matched leaf), right flag=1 (matched leaf)
	result := VerifyPartialMerkleTree(
		[]*chainhash.Hash{&tx, &tx},
		[]byte{0x07}, // bits: 1,1,1
		2,
		root,
	)
	if result.Valid {
		t.Error("duplicate child hashes should be rejected per BIP 37")
	}
}

func TestParseBlockHash(t *testing.T) {
	blockHash, err := ParseBlockHash(testTxoutproofHex)
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	if blockHash == "" {
		t.Error("block hash should not be empty")
	}
	if len(blockHash) != 64 {
		t.Errorf("block hash length: got %d, want 64", len(blockHash))
	}
}