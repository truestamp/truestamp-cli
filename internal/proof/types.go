// Copyright (c) 2021-2026 Truestamp, Inc.
// SPDX-License-Identifier: MIT

package proof

import "encoding/json"

// ProofBundle is the top-level proof structure (v1 compact format).
type ProofBundle struct {
	Version   int    `json:"v"`
	Type      string `json:"-"` // derived from Subject.Source during parsing
	Timestamp string `json:"ts"`
	PublicKey string `json:"pk"`
	Signature string `json:"sig"`

	// Typed access to proof contents (populated during parsing)
	Subject        Subject          `json:"-"`
	Block          Block            `json:"-"`
	Commitments    []ExternalCommit `json:"-"`
	InclusionProof string           `json:"-"` // base64url compact Merkle proof

	// Raw JSON preserved for JCS canonicalization (extracted from subject.data)
	RawData json.RawMessage `json:"-"`
}

// IsEntropy returns true if this proof bundle is for an entropy observation.
func (b *ProofBundle) IsEntropy() bool {
	return b.Subject.Source != "item"
}

// MarshalJSON produces the compact JSON format from a parsed ProofBundle.
// Used when sending a CBOR-decoded proof to the API as JSON.
func (b *ProofBundle) MarshalJSON() ([]byte, error) {
	m := map[string]any{
		"v":   b.Version,
		"ts":  b.Timestamp,
		"pk":  b.PublicKey,
		"sig": b.Signature,
		"s":   b.Subject,
		"b":   b.Block,
		"ip":  b.InclusionProof,
		"cx":  b.Commitments,
	}
	return json.Marshal(m)
}

// Subject represents the unified subject within a proof bundle.
// For item proofs, Source is "item" and Data contains claims.
// For entropy proofs, Source is the entropy source (e.g., "nist_beacon").
type Subject struct {
	Source       string          `json:"src"`
	ID           string          `json:"id"`
	Data         json.RawMessage `json:"d"`
	MetadataHash string          `json:"mh"`
	SigningKeyID string          `json:"kid"`
}

// Block represents the single block in the proof.
type Block struct {
	ID                string `json:"id"`
	PreviousBlockHash string `json:"ph"`
	MerkleRoot        string `json:"mr"`
	MetadataHash      string `json:"mh"`
	SigningKeyID      string `json:"kid"`
}

// ExternalCommit represents a commitment entry in the proof bundle (stellar or bitcoin).
// Each has an epoch Merkle proof (ep) linking the block hash to the committed value.
type ExternalCommit struct {
	Type    string `json:"t"`   // "stellar" or "bitcoin"
	Network string `json:"net"` // "testnet", "public", "mainnet", "regtest"

	// Epoch Merkle proof (base64url compact binary)
	EpochProof string `json:"ep"`

	// Stellar fields
	TransactionHash string `json:"tx,omitempty"`
	MemoHash        string `json:"memo,omitempty"`
	Ledger          int    `json:"l,omitempty"`
	Timestamp       string `json:"ts,omitempty"`

	// Bitcoin fields
	OpReturn        string `json:"op,omitempty"`
	RawTxHex        string `json:"rtx,omitempty"`
	TxoutproofHex   string `json:"txp,omitempty"`
	BlockMerkleRoot string `json:"bmr,omitempty"`
	BlockHeight     int    `json:"h,omitempty"`
}

// FindCommitByType returns the first external commitment with the given type, or nil.
func FindCommitByType(commits []ExternalCommit, typ string) *ExternalCommit {
	for i := range commits {
		if commits[i].Type == typ {
			return &commits[i]
		}
	}
	return nil
}