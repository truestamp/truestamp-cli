// Copyright (c) 2021-2026 Truestamp, Inc.
// SPDX-License-Identifier: MIT

package proof

import (
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"

	"github.com/fxamacker/cbor/v2"
	"github.com/truestamp/truestamp-cli/internal/proof/ptype"
)

// deterministicCBOR is a CBOR encode mode configured for RFC 8949 §4.2
// "Core Deterministic Encoding": shortest-form integers, definite-length
// maps and arrays, lexicographically-sorted map keys. This makes
// (decode → MarshalCBOR) normalize mis-ordered inputs to a stable form.
//
// EncMode() can only fail on invalid option combinations; this one is
// statically valid, so we discard the error.
var deterministicCBOR, _ = cbor.CoreDetEncOptions().EncMode()

// MarshalCBOR produces the canonical CBOR representation of the proof
// bundle. Byte-valued fields (`pk`, `sig`, hashes, epoch/inclusion
// proofs) are emitted as CBOR major-type-2 byte strings; identifier
// fields (ULID, UUIDv7, timestamps) remain text. The subject data
// (`s.d`) is decoded back from its preserved raw JSON and encoded as a
// nested CBOR structure. `t` is emitted as a CBOR integer at the top
// level and per commitment entry.
//
// Round-trip guarantee: `cbor → Parse → MarshalCBOR` is byte-stable for
// inputs that are themselves deterministically encoded. Non-deterministic
// source CBOR is normalized on the first round trip.
func (b *ProofBundle) MarshalCBOR() ([]byte, error) {
	pkBytes, err := base64.StdEncoding.DecodeString(b.PublicKey)
	if err != nil {
		return nil, fmt.Errorf("pk base64 decode: %w", err)
	}
	sigBytes, err := base64.StdEncoding.DecodeString(b.Signature)
	if err != nil {
		return nil, fmt.Errorf("sig base64 decode: %w", err)
	}

	blockMap, err := blockToCBORMap(b.Block)
	if err != nil {
		return nil, fmt.Errorf("b: %w", err)
	}
	commits, err := commitsToCBOR(b.Commitments)
	if err != nil {
		return nil, fmt.Errorf("cx: %w", err)
	}

	out := map[string]any{
		"v":   b.Version,
		"t":   uint16(b.T),
		"ts":  b.Timestamp,
		"pk":  pkBytes,
		"sig": sigBytes,
		"b":   blockMap,
		"cx":  commits,
	}

	if b.T != ptype.Block {
		if b.Subject == nil {
			return nil, fmt.Errorf("non-block proof missing subject")
		}
		subjectMap, err := subjectToCBORMap(*b.Subject, b.RawData)
		if err != nil {
			return nil, fmt.Errorf("s: %w", err)
		}
		ipBytes, err := decodeB64URLOrBytes(b.InclusionProof)
		if err != nil {
			return nil, fmt.Errorf("ip base64url decode: %w", err)
		}
		out["s"] = subjectMap
		out["ip"] = ipBytes
	}

	body, err := deterministicCBOR.Marshal(out)
	if err != nil {
		return nil, err
	}
	// Prepend the self-describing CBOR tag 55799 (0xd9d9f7). Matches the
	// wire format the Truestamp backend emits and lets IsCBORProof
	// auto-detect format on round-trips through the CLI.
	return append([]byte{0xd9, 0xd9, 0xf7}, body...), nil
}

func subjectToCBORMap(s Subject, rawData json.RawMessage) (map[string]any, error) {
	mh, err := decodeHexOrBytes(s.MetadataHash)
	if err != nil {
		return nil, fmt.Errorf("mh hex decode: %w", err)
	}
	kid, err := decodeHexOrBytes(s.SigningKeyID)
	if err != nil {
		return nil, fmt.Errorf("kid hex decode: %w", err)
	}

	// Subject data: parse the preserved raw JSON back into a generic
	// structure so the deterministic CBOR encoder can visit it (and
	// order its keys).
	var data any
	if len(rawData) > 0 {
		if err := json.Unmarshal(rawData, &data); err != nil {
			return nil, fmt.Errorf("d JSON parse: %w", err)
		}
	} else if len(s.Data) > 0 {
		if err := json.Unmarshal(s.Data, &data); err != nil {
			return nil, fmt.Errorf("d JSON parse: %w", err)
		}
	}

	return map[string]any{
		"id":  s.ID,
		"d":   data,
		"mh":  mh,
		"kid": kid,
	}, nil
}

func blockToCBORMap(b Block) (map[string]any, error) {
	ph, err := decodeHexOrBytes(b.PreviousBlockHash)
	if err != nil {
		return nil, fmt.Errorf("ph hex decode: %w", err)
	}
	mr, err := decodeHexOrBytes(b.MerkleRoot)
	if err != nil {
		return nil, fmt.Errorf("mr hex decode: %w", err)
	}
	mh, err := decodeHexOrBytes(b.MetadataHash)
	if err != nil {
		return nil, fmt.Errorf("mh hex decode: %w", err)
	}
	kid, err := decodeHexOrBytes(b.SigningKeyID)
	if err != nil {
		return nil, fmt.Errorf("kid hex decode: %w", err)
	}
	return map[string]any{
		"id":  b.ID,
		"ph":  ph,
		"mr":  mr,
		"mh":  mh,
		"kid": kid,
	}, nil
}

func commitsToCBOR(commits []ExternalCommit) ([]any, error) {
	out := make([]any, 0, len(commits))
	for i, c := range commits {
		m := map[string]any{
			"t":   uint16(c.Type),
			"net": c.Network,
		}
		ep, err := decodeB64URLOrBytes(c.EpochProof)
		if err != nil {
			return nil, fmt.Errorf("cx[%d].ep: %w", i, err)
		}
		if ep != nil {
			m["ep"] = ep
		}
		if c.TransactionHash != "" {
			tx, err := decodeHexOrBytes(c.TransactionHash)
			if err != nil {
				return nil, fmt.Errorf("cx[%d].tx: %w", i, err)
			}
			m["tx"] = tx
		}
		if c.MemoHash != "" {
			memo, err := decodeHexOrBytes(c.MemoHash)
			if err != nil {
				return nil, fmt.Errorf("cx[%d].memo: %w", i, err)
			}
			m["memo"] = memo
		}
		if c.Ledger != 0 {
			m["l"] = c.Ledger
		}
		if c.Timestamp != "" {
			m["ts"] = c.Timestamp
		}
		if c.OpReturn != "" {
			op, err := decodeHexOrBytes(c.OpReturn)
			if err != nil {
				return nil, fmt.Errorf("cx[%d].op: %w", i, err)
			}
			m["op"] = op
		}
		if c.RawTxHex != "" {
			rtx, err := decodeHexOrBytes(c.RawTxHex)
			if err != nil {
				return nil, fmt.Errorf("cx[%d].rtx: %w", i, err)
			}
			m["rtx"] = rtx
		}
		if c.TxoutproofHex != "" {
			txp, err := decodeHexOrBytes(c.TxoutproofHex)
			if err != nil {
				return nil, fmt.Errorf("cx[%d].txp: %w", i, err)
			}
			m["txp"] = txp
		}
		if c.BlockMerkleRoot != "" {
			bmr, err := decodeHexOrBytes(c.BlockMerkleRoot)
			if err != nil {
				return nil, fmt.Errorf("cx[%d].bmr: %w", i, err)
			}
			m["bmr"] = bmr
		}
		if c.BlockHeight != 0 {
			m["h"] = c.BlockHeight
		}
		out = append(out, m)
	}
	return out, nil
}

// decodeHexOrBytes turns a hex string into the underlying bytes. An
// empty string becomes nil (omitted from the CBOR output where
// applicable).
func decodeHexOrBytes(s string) ([]byte, error) {
	if s == "" {
		return nil, nil
	}
	return hex.DecodeString(s)
}

// decodeB64URLOrBytes turns a base64url (raw, unpadded) string into the
// underlying bytes.
func decodeB64URLOrBytes(s string) ([]byte, error) {
	if s == "" {
		return nil, nil
	}
	return base64.RawURLEncoding.DecodeString(s)
}
