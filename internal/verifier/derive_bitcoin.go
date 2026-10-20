// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: Apache-2.0

// This file makes the Bitcoin evidence of DERIVE (SPEC.md section 10.2,
// step 9 (d)): for each Bitcoin entry N, at /commitments/bitcoin and
// /signing_key_event/commitments/bitcoin (layout.go's commitmentEntries),
// these checks, in this order, all under the constructions row bitcoin. A
// structural failure is bitcoin_evidence_invalid, and an inequality
// derivation_mismatch:
//
//	| Check                                                                                    | Part                 | Code                     | Traced |
//	|------------------------------------------------------------------------------------------|----------------------|--------------------------|--------|
//	| raw_transaction is well formed                                                           | N/raw_transaction    | bitcoin_evidence_invalid | no     |
//	| the 32 bytes its OP_RETURN script pushes equal the recomputed epoch_hash (section 5.4)   | N/raw_transaction    | derivation_mismatch      | no     |
//	| rev(SHA-256d(raw_transaction)) = transaction_hash                                        | N/transaction_hash   | derivation_mismatch      | yes    |
//	| txoutproof is well formed                                                                | N/txoutproof         | bitcoin_evidence_invalid | no     |
//	| the one hash it matches equals SHA-256d(raw_transaction)                                 | N/txoutproof         | derivation_mismatch      | no     |
//	| rev(SHA-256d(header)) = header_hash                                                      | N/header_hash        | derivation_mismatch      | yes    |
//	| rev(the header's Merkle root field) = header_merkle_root                                 | N/header_merkle_root | derivation_mismatch      | yes    |
//	| the header's nTime, formatted as TIME, = header_time                                     | N/header_time        | derivation_mismatch      | yes    |
//
// The rules of a well-formed raw_transaction and txoutproof, and the
// values read out of one, are internal/verifier/bitcoin's: its parsers
// judge the bytes as carried, the hex N holds decoded, and never strip a
// witness. Apart from the two checks of well-formedness, a check that
// reads a value out of raw_transaction or txoutproof is made only when
// that structure is well formed: the transaction_hash row is always made,
// since it reads only the bytes, the OP_RETURN row needs a well-formed
// raw_transaction, and the row of the matched hash and the three header
// rows need a well-formed txoutproof. A check not made reports no row: the
// failing row of well-formedness already explains it (section 10.2, Rows).
// A failing row of well-formedness holds construction and the first rule
// the bytes break (derive_check.go).
//
// The declared value of the OP_RETURN row is the 32 bytes the OP_RETURN
// script pushes, as hex in the order pushed, and of the row of the matched
// hash the hash the txoutproof matches, not the member at the part. The
// OP_RETURN is compared with H(0x40, N/epoch/record), the epoch_hash this
// run recomputes through internal/registry under the registry constant
// this file names at its call, never with N/epoch/epoch_hash, the one N
// carries (sections 5.4 and 10.3, and section 10.6, rule 4). Every Bitcoin
// hash is in display order (bitcoin.Display, the encoding row of section
// 7.3); the OP_RETURN's bytes and the epoch hash, which are no Bitcoin
// hash, are in the order they are pushed and written. header_time is the
// header's nTime formatted as TIME (time.go's FormatTime).
//
// raw_transaction and txoutproof are PREIMAGE (section 3.7), so their rows
// give no trace row; transaction_hash, header_hash, header_merkle_root and
// header_time are CHECKED, so theirs do (section 10.6).
//
// SHAPE has held raw_transaction and txoutproof to HEXBYTES, and each
// declared member to its class (section 2.4), so a value that does not
// decode as hex is a fault of the verifier, returned as an error, which
// [Derive] reports as internal_error.

package verifier

import (
	"encoding/hex"
	"fmt"

	"github.com/truestamp/truestamp-cli/internal/registry"
	"github.com/truestamp/truestamp-cli/internal/specdata"
	"github.com/truestamp/truestamp-cli/internal/verifier/bitcoin"
)

// bitcoinRow is the row of the constructions text every check of (d)
// names.
const bitcoinRow = "bitcoin"

// bitcoinEvidenceChecks are the checks of (d) for each Bitcoin entry P
// carries, in document order, each entry's in the table's order.
func bitcoinEvidenceChecks(proof map[string]any) ([]deriveCheck, error) {
	var out []deriveCheck
	for _, entry := range commitmentEntries(proof) {
		if entry.name != ChainBitcoin {
			continue
		}
		checks, err := bitcoinEntryChecks(proof, entry.at)
		if err != nil {
			return nil, err
		}
		out = append(out, checks...)
	}
	return out, nil
}

// bitcoinEntryChecks are the checks of (d) for the Bitcoin entry at at.
func bitcoinEntryChecks(proof map[string]any, at Pointer) ([]deriveCheck, error) {
	rawAt, proofAt := under(at, "raw_transaction"), under(at, "txoutproof")
	raw, err := hexBytesAt(proof, rawAt)
	if err != nil {
		return nil, err
	}
	merkleBlock, err := hexBytesAt(proof, proofAt)
	if err != nil {
		return nil, err
	}
	epochHash, err := recordHashAt(proof, registry.EpochHash, under(at, "epoch", "record"))
	if err != nil {
		return nil, err
	}
	txid := bitcoin.TransactionHash(raw)

	pushed, txErr := bitcoin.ParseTransaction(raw)
	out := []deriveCheck{wellFormedCheck(rawAt, "raw_transaction", txErr)}
	if txErr == nil {
		c := equalCheck(rawAt, bitcoinRow, []Pointer{under(at, "epoch", "record"), rawAt}, hex.EncodeToString(pushed[:]), epochHash,
			"The epoch hash the transaction's OP_RETURN holds")
		c.traced = false
		out = append(out, c)
	}
	out = append(out, equalCheck(under(at, "transaction_hash"), bitcoinRow, []Pointer{rawAt},
		valueAt(proof, under(at, "transaction_hash")), bitcoin.Display(txid), "The transaction hash"))

	block, blockErr := bitcoin.ParseMerkleBlock(merkleBlock)
	out = append(out, wellFormedCheck(proofAt, "txoutproof", blockErr))
	if blockErr != nil {
		return out, nil
	}
	matched := equalCheck(proofAt, bitcoinRow, []Pointer{rawAt, proofAt}, bitcoin.Display(block.Matched), bitcoin.Display(txid),
		"The transaction hash the txoutproof matches")
	matched.traced = false
	headerTime, ok := FormatTime(ClassTIME, int64(bitcoin.HeaderTime(block.Header)))
	if !ok {
		return nil, fmt.Errorf("the header time of the txoutproof at %s cannot be formatted as TIME", proofAt.Encode())
	}
	header := func(member string, recomputed any, noun string) deriveCheck {
		part := under(at, member)
		return equalCheck(part, bitcoinRow, []Pointer{proofAt}, valueAt(proof, part), recomputed, noun)
	}
	return append(out, matched,
		header("header_hash", bitcoin.Display(bitcoin.HeaderHash(block.Header)), "The header hash"),
		header("header_merkle_root", bitcoin.Display(bitcoin.MerkleRootField(block.Header)), "The header's Merkle root"),
		header("header_time", headerTime, "The header time"),
	), nil
}

// wellFormedCheck is the untraced check that the structure member, at
// part, is well formed: a pass when refusal is nil, and otherwise
// bitcoin_evidence_invalid naming the first rule it breaks.
func wellFormedCheck(part Pointer, member string, refusal error) deriveCheck {
	c := newCheck(part, bitcoinRow, []Pointer{part}, nil, nil, fmt.Sprintf("The %s at %s is well formed.", member, part.Encode()))
	if refusal != nil {
		c.code, c.rule = specdata.ReasonBitcoinEvidenceInvalid, refusal.Error()
		c.message = fmt.Sprintf("The %s at %s is not well formed: it breaks the rule %s.", member, part.Encode(), refusal.Error())
	}
	return c
}

// hexBytesAt is the bytes the HEXBYTES value at at spells.
func hexBytesAt(proof map[string]any, at Pointer) ([]byte, error) {
	text, ok := valueAt(proof, at).(string)
	if !ok {
		return nil, fmt.Errorf("the value at %s is not a string", at.Encode())
	}
	b, err := hex.DecodeString(text)
	if err != nil {
		return nil, fmt.Errorf("the value at %s: %w", at.Encode(), err)
	}
	return b, nil
}
