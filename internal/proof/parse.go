// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: MIT

package proof

import (
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"

	"github.com/truestamp/truestamp-cli/internal/proof/ptype"
)

// Parse reads and parses a proof JSON file, preserving raw JSON for JCS.
func Parse(filename string) (*ProofBundle, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("reading file: %w", err)
	}
	return ParseBytes(data)
}

// ParseBytes parses a proof from raw bytes. Detects CBOR format (self-
// describing tag 0xd9d9f7) and falls back to JSON parsing. Enforces:
//
//   - v == 1
//   - t ∈ {10, 11, 20, 30, 31, 32}
//   - cx non-empty, every cx[i].t ∈ {40, 41}
//   - t ∈ {10, 11} (block-like): s absent, ip absent
//   - t ∉ {10, 11}: s present with id/d/mh/kid, ip present and non-empty
func ParseBytes(data []byte) (*ProofBundle, error) {
	if IsCBORProof(data) {
		return ParseCBOR(data)
	}

	// JSON parsing: extract top-level fields with short keys
	var raw struct {
		Version        int             `json:"v"`
		T              *uint16         `json:"t"`
		Timestamp      string          `json:"ts"`
		PublicKey      string          `json:"pk"`
		Signature      string          `json:"sig"`
		Subject        json.RawMessage `json:"s"`
		Block          json.RawMessage `json:"b"`
		Commitments    json.RawMessage `json:"cx"`
		InclusionProof *string         `json:"ip"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("parsing JSON: %w", err)
	}

	// Top-level fields
	if raw.Version == 0 {
		return nil, fmt.Errorf("missing required field: v")
	}
	if raw.T == nil {
		return nil, fmt.Errorf("missing required field: t")
	}
	t := ptype.Code(*raw.T)
	if !ptype.IsValidSubject(t) {
		return nil, fmt.Errorf("invalid subject type code: %d", *raw.T)
	}
	if raw.PublicKey == "" {
		return nil, fmt.Errorf("missing required field: pk")
	}
	if raw.Signature == "" {
		return nil, fmt.Errorf("missing required field: sig")
	}

	// Block is always required
	if len(raw.Block) == 0 || string(raw.Block) == "null" {
		return nil, fmt.Errorf("missing required field: b")
	}
	var block Block
	if err := json.Unmarshal(raw.Block, &block); err != nil {
		return nil, fmt.Errorf("parsing block: %w", err)
	}

	// Commitments always required and non-empty
	if len(raw.Commitments) == 0 || string(raw.Commitments) == "null" {
		return nil, fmt.Errorf("missing required field: cx")
	}
	var commits []ExternalCommit
	if err := json.Unmarshal(raw.Commitments, &commits); err != nil {
		return nil, fmt.Errorf("parsing commitments: %w", err)
	}
	if len(commits) == 0 {
		return nil, fmt.Errorf("cx must not be empty")
	}
	for i, cx := range commits {
		if !ptype.IsValidExternalCommitment(cx.Type) {
			return nil, fmt.Errorf("cx[%d]: invalid commitment type code: %d", i, cx.Type)
		}
	}

	bundle := &ProofBundle{
		Version:     raw.Version,
		T:           t,
		Timestamp:   raw.Timestamp,
		PublicKey:   raw.PublicKey,
		Signature:   raw.Signature,
		Block:       block,
		Commitments: commits,
	}

	// Subject + inclusion proof rules depend on type code. Block-like
	// subjects (t ∈ {10, 11} — plain block and beacon) share the same
	// wire shape: no `s`, no `ip`, `subject_hash == block_hash`.
	if ptype.IsBlockLikeSubject(t) {
		shape := ptype.Name(t) // "block" or "beacon" — for error text
		if len(raw.Subject) > 0 && string(raw.Subject) != "null" {
			return nil, fmt.Errorf("%s proof must not include s", shape)
		}
		if raw.InclusionProof != nil && *raw.InclusionProof != "" {
			return nil, fmt.Errorf("%s proof must not include ip", shape)
		}
		if err := validateSizes(bundle); err != nil {
			return nil, err
		}
		return bundle, nil
	}

	// Non-block-like subjects (item, entropy_*): s and ip are required
	if len(raw.Subject) == 0 || string(raw.Subject) == "null" {
		return nil, fmt.Errorf("missing required field: s")
	}
	var subject Subject
	if err := json.Unmarshal(raw.Subject, &subject); err != nil {
		return nil, fmt.Errorf("parsing subject: %w", err)
	}
	if subject.ID == "" || subject.MetadataHash == "" || subject.SigningKeyID == "" {
		return nil, fmt.Errorf("subject missing required fields (id, mh, kid)")
	}
	if raw.InclusionProof == nil || *raw.InclusionProof == "" {
		return nil, fmt.Errorf("missing required field: ip")
	}

	bundle.Subject = &subject
	bundle.RawData = subject.Data
	bundle.InclusionProof = *raw.InclusionProof

	if err := validateSizes(bundle); err != nil {
		return nil, err
	}
	return bundle, nil
}

// validateSizes enforces the CDDL-declared field sizes for pk / sig /
// hash / kid. Surfaced at parse time so consumers see a precise error
// ("pk must be 32 bytes, got 31") rather than an opaque Ed25519 failure
// three steps later in the verification pipeline. Sizes match
// priv/cddl/proof.cddl in the truestamp-v2 reference repo.
func validateSizes(b *ProofBundle) error {
	if n, err := base64DecodedLen(b.PublicKey); err != nil {
		return fmt.Errorf("pk is not valid base64: %w", err)
	} else if n != 32 {
		return fmt.Errorf("pk must be 32 bytes, got %d", n)
	}
	if n, err := base64DecodedLen(b.Signature); err != nil {
		return fmt.Errorf("sig is not valid base64: %w", err)
	} else if n != 64 {
		return fmt.Errorf("sig must be 64 bytes, got %d", n)
	}

	if err := expectHexBytes(b.Block.PreviousBlockHash, 32, "b.ph"); err != nil {
		return err
	}
	if err := expectHexBytes(b.Block.MerkleRoot, 32, "b.mr"); err != nil {
		return err
	}
	if err := expectHexBytes(b.Block.MetadataHash, 32, "b.mh"); err != nil {
		return err
	}
	if err := expectHexBytes(b.Block.SigningKeyID, 4, "b.kid"); err != nil {
		return err
	}

	if b.Subject != nil {
		if err := expectHexBytes(b.Subject.MetadataHash, 32, "s.mh"); err != nil {
			return err
		}
		if err := expectHexBytes(b.Subject.SigningKeyID, 4, "s.kid"); err != nil {
			return err
		}
	}

	for i, c := range b.Commitments {
		pfx := fmt.Sprintf("cx[%d]", i)
		if err := expectHexBytes(c.TransactionHash, 32, pfx+".tx"); err != nil {
			return err
		}
		switch c.Type {
		case ptype.CommitmentStellar:
			if err := expectHexBytes(c.MemoHash, 32, pfx+".memo"); err != nil {
				return err
			}
		case ptype.CommitmentBitcoin:
			if err := expectHexBytes(c.OpReturn, 32, pfx+".op"); err != nil {
				return err
			}
			if err := expectHexBytes(c.BlockMerkleRoot, 32, pfx+".bmr"); err != nil {
				return err
			}
		}
	}
	return nil
}

// expectHexBytes verifies that s is a lowercase-or-uppercase hex string
// decoding to exactly `nBytes` bytes.
func expectHexBytes(s string, nBytes int, field string) error {
	if s == "" {
		return fmt.Errorf("%s is required", field)
	}
	wantHex := nBytes * 2
	if len(s) != wantHex {
		return fmt.Errorf("%s must be %d bytes (%d hex chars), got %d hex chars", field, nBytes, wantHex, len(s))
	}
	if _, err := hex.DecodeString(s); err != nil {
		return fmt.Errorf("%s is not valid hex: %w", field, err)
	}
	return nil
}

// base64DecodedLen returns the number of bytes b64-encoded in s using
// standard base64 (with or without padding). Returns an error if s is
// not valid base64.
func base64DecodedLen(s string) (int, error) {
	if s == "" {
		return 0, nil
	}
	dec, err := base64.StdEncoding.DecodeString(s)
	if err != nil {
		// Retry raw (unpadded) — some generators omit padding.
		dec2, err2 := base64.RawStdEncoding.DecodeString(s)
		if err2 != nil {
			return 0, err
		}
		return len(dec2), nil
	}
	return len(dec), nil
}

// FileSizeFromData returns the byte length of proof data.
func FileSizeFromData(data []byte) int64 { return int64(len(data)) }

// FileSize returns the size of the file in bytes.
func FileSize(filename string) int64 {
	info, err := os.Stat(filename)
	if err != nil {
		return 0
	}
	return info.Size()
}
