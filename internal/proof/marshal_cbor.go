// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: MIT

package proof

import (
	"bytes"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/fxamacker/cbor/v2"
	"github.com/truestamp/truestamp-cli/internal/proof/ptype"
	"github.com/truestamp/truestamp-cli/internal/tscrypto"
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
// bundle. Byte-valued fields (`pk`, `sig`, hashes) are emitted as CBOR
// major-type-2 byte strings; identifier fields (ULID, UUIDv7, timestamps)
// and the E.3 text-string fields `rtx` / `txp` remain text. The subject data
// (`s.d`) is decoded back from its preserved raw JSON and encoded as a
// nested CBOR structure. `t` is emitted as a CBOR integer at the top level
// and per commitment entry.
//
// `ip` and `ep` are emitted as byte strings even though E.3's table lists
// them as text: that table governs what a verifier MUST accept on decode,
// and byte strings are what the backend puts on the wire. The decoder
// accepts both forms.
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

	// Block-like subjects (t ∈ {10, 11}) have no `s`, no `ip`.
	if !b.IsBlockLike() {
		if b.Subject == nil {
			return nil, fmt.Errorf("non-block-like proof missing subject")
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
		out["ip"] = nonNilBytes(ipBytes)
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
	raw := rawData
	if len(raw) == 0 {
		raw = s.Data
	}
	var data any
	if len(raw) > 0 {
		if data, err = decodeSubjectDataJSON(raw); err != nil {
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

// decodeSubjectDataJSON re-reads the preserved `s.d` JSON for CBOR encoding.
// Numbers are read as json.Number rather than float64: E.3 requires CBOR
// integers to map to JSON integers, and routing every literal through a
// float64 both rewrites integers as floats and silently rounds anything past
// 2^53 — which changes the signed 0x11 preimage on a `convert proof` trip.
func decodeSubjectDataJSON(raw json.RawMessage) (any, error) {
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.UseNumber()
	var v any
	if err := dec.Decode(&v); err != nil {
		return nil, err
	}
	if dec.More() {
		return nil, fmt.Errorf("trailing data after subject data")
	}
	return jsonNumbersToCBOR(v, "s.d")
}

// jsonNumbersToCBOR maps every json.Number to the CBOR type that preserves
// it exactly: an integer literal becomes a CBOR integer, anything else an
// IEEE-754 double. A literal that fits neither is an error naming the key
// rather than a silently rounded — and therefore corrupted — bundle.
func jsonNumbersToCBOR(v any, path string) (any, error) {
	switch val := v.(type) {
	case map[string]any:
		out := make(map[string]any, len(val))
		for k, v2 := range val {
			conv, err := jsonNumbersToCBOR(v2, path+"."+k)
			if err != nil {
				return nil, err
			}
			out[k] = conv
		}
		return out, nil
	case []any:
		out := make([]any, len(val))
		for i, v2 := range val {
			conv, err := jsonNumbersToCBOR(v2, fmt.Sprintf("%s[%d]", path, i))
			if err != nil {
				return nil, err
			}
			out[i] = conv
		}
		return out, nil
	case json.Number:
		s := val.String()
		if !strings.ContainsAny(s, ".eE") {
			if n, err := strconv.ParseInt(s, 10, 64); err == nil {
				return n, nil
			}
			if n, err := strconv.ParseUint(s, 10, 64); err == nil {
				return n, nil
			}
			return nil, fmt.Errorf("%s: integer %s is outside the exactly representable 64-bit range and cannot be encoded without loss", path, s)
		}
		f, err := strconv.ParseFloat(s, 64)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", path, err)
		}
		return f, nil
	default:
		return v, nil
	}
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
		// `ep` and the chain's epoch-root key are emitted unconditionally,
		// empty or not: E.6 hard-rejects an entry that is missing either,
		// so dropping an empty value would make the CLI emit a bundle its
		// own parser refuses.
		m["ep"] = nonNilBytes(ep)
		if c.TransactionHash != "" {
			tx, err := decodeHexOrBytes(c.TransactionHash)
			if err != nil {
				return nil, fmt.Errorf("cx[%d].tx: %w", i, err)
			}
			m["tx"] = tx
		}
		if c.Type == ptype.CommitmentStellar || c.MemoHash != "" {
			memo, err := decodeHexOrBytes(c.MemoHash)
			if err != nil {
				return nil, fmt.Errorf("cx[%d].memo: %w", i, err)
			}
			m["memo"] = nonNilBytes(memo)
		}
		if c.Ledger != 0 {
			m["l"] = c.Ledger
		}
		if c.Timestamp != "" {
			m["ts"] = c.Timestamp
		}
		if c.Type == ptype.CommitmentBitcoin || c.OpReturn != "" {
			op, err := decodeHexOrBytes(c.OpReturn)
			if err != nil {
				return nil, fmt.Errorf("cx[%d].op: %w", i, err)
			}
			m["op"] = nonNilBytes(op)
		}
		// `rtx` and `txp` are E.3 text strings carrying base64url or hex;
		// emitting them verbatim is what makes the decode side able to
		// hand back the base64url form unchanged.
		if c.RawTxHex != "" {
			m["rtx"] = c.RawTxHex
		}
		if c.TxoutproofHex != "" {
			m["txp"] = c.TxoutproofHex
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

// decodeHexOrBytes turns a lowercase-hex string into the underlying
// bytes. An empty string becomes nil (omitted from the CBOR output where
// applicable).
//
// The lowercase rule is E.4's and is enforced here, not merely relied on,
// because hex.DecodeString is case-insensitive and this function is the
// JSON→CBOR bridge: an uppercase `b.kid` decoded to the same four bytes
// and re-emitted as a byte string, so `convert proof --to cbor` silently
// laundered a bundle the verifier rejects into one it accepts. Refusing
// the conversion is the coherent half of that pair — the CLI does not
// hand back a repaired bundle for a defect it will not verify.
func decodeHexOrBytes(s string) ([]byte, error) {
	if s == "" {
		return nil, nil
	}
	if err := tscrypto.ValidateLowercaseHex(s); err != nil {
		return nil, err
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

// nonNilBytes normalizes a nil slice to an empty one so the encoder emits a
// zero-length byte string rather than `null`. A null value counts as absent
// under E.6's presence rule, so an empty field must still round-trip as
// present.
func nonNilBytes(b []byte) []byte {
	if b == nil {
		return []byte{}
	}
	return b
}
