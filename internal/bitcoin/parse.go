// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: MIT

package bitcoin

import (
	"bytes"
	"encoding/hex"
	"fmt"

	"github.com/btcsuite/btcd/txscript/v2"
	"github.com/btcsuite/btcd/wire/v2"
)

// DecodeTxOutProof decodes a txoutproof hex string into a wire.MsgMerkleBlock.
func DecodeTxOutProof(txoutproofHex string) (*wire.MsgMerkleBlock, error) {
	raw, err := hex.DecodeString(txoutproofHex)
	if err != nil {
		return nil, fmt.Errorf("decoding txoutproof hex: %w", err)
	}
	var mb wire.MsgMerkleBlock
	if err := mb.BtcDecode(bytes.NewReader(raw), wire.ProtocolVersion, wire.LatestEncoding); err != nil {
		return nil, fmt.Errorf("parsing merkle block: %w", err)
	}
	return &mb, nil
}

// ParseBlockHash decodes a txoutproof hex string and returns the block hash (display order).
func ParseBlockHash(txoutproofHex string) (string, error) {
	mb, err := DecodeTxOutProof(txoutproofHex)
	if err != nil {
		return "", err
	}
	return mb.Header.BlockHash().String(), nil
}

// DecodeTransaction decodes raw transaction hex into a wire.MsgTx.
func DecodeTransaction(rawTxHex string) (*wire.MsgTx, error) {
	raw, err := hex.DecodeString(rawTxHex)
	if err != nil {
		return nil, fmt.Errorf("decoding transaction hex: %w", err)
	}
	var tx wire.MsgTx
	if err := tx.Deserialize(bytes.NewReader(raw)); err != nil {
		return nil, fmt.Errorf("parsing transaction: %w", err)
	}
	return &tx, nil
}

// ExtractOpReturn parses a raw transaction and returns the OP_RETURN data as hex.
//
// Appendix E.19 of the whitepaper fixes the selection rule: the payload is the
// push data of the FIRST output whose script begins with 0x6a. Selection is
// therefore on the leading opcode alone, not on txscript.IsNullData, which
// additionally enforces a single push and the 80-byte MaxDataCarrierSize
// policy limit. Once that output is chosen the parse runs on it and nothing
// else — scanning past a malformed or oversized first OP_RETURN to a later,
// well-formed one would let a crafted transaction present a payload
// generation never emitted.
func ExtractOpReturn(rawTxHex string) (string, error) {
	tx, err := DecodeTransaction(rawTxHex)
	if err != nil {
		return "", err
	}
	for _, txout := range tx.TxOut {
		if len(txout.PkScript) == 0 || txout.PkScript[0] != txscript.OP_RETURN {
			continue
		}
		pushes, err := txscript.PushedData(txout.PkScript)
		if err != nil {
			return "", fmt.Errorf("OP_RETURN output does not parse: %w", err)
		}
		if len(pushes) == 0 {
			return "", fmt.Errorf("OP_RETURN output carries no push data")
		}
		return hex.EncodeToString(pushes[0]), nil
	}
	return "", fmt.Errorf("no OP_RETURN output found")
}

// ComputeTxID computes the txid from raw transaction hex (display order, handles segwit).
func ComputeTxID(rawTxHex string) (string, error) {
	tx, err := DecodeTransaction(rawTxHex)
	if err != nil {
		return "", err
	}
	return tx.TxHash().String(), nil
}
