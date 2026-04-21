// Copyright (c) 2021-2026 Truestamp, Inc.
// SPDX-License-Identifier: MIT

package proof

import (
	"encoding/json"

	"github.com/truestamp/truestamp-cli/internal/proof/ptype"
)

// ProofBundle is the top-level compact Truestamp proof structure (bundle
// version `v` is 1). See `docs/PROOF_FORMAT.md` (authoritative spec) and
// `docs/PROOF_FORMAT_IMPLEMENTERS_GUIDE.md` (cross-implementation guide)
// in the truestamp-v2 reference repo.
//
// The top-level integer field `t` discriminates subject types:
//
//	10  block            (no Subject, no InclusionProof)
//	20  item
//	30  entropy_nist
//	31  entropy_stellar
//	32  entropy_bitcoin
type ProofBundle struct {
	Version   int        `json:"v"`
	T         ptype.Code `json:"t"`
	Timestamp string     `json:"ts"`
	PublicKey string     `json:"pk"`
	Signature string     `json:"sig"`

	// Subject is nil for block subjects (T == ptype.Block). For all other
	// subject types it is non-nil after a successful parse.
	Subject *Subject `json:"-"`

	Block       Block            `json:"-"`
	Commitments []ExternalCommit `json:"-"`

	// InclusionProof is "" for block subjects and non-empty otherwise.
	InclusionProof string `json:"-"` // base64url compact Merkle proof

	// RawData is the subject-data JSON preserved byte-for-byte for JCS.
	// Empty for block subjects.
	RawData json.RawMessage `json:"-"`
}

// IsBlock returns true if this bundle is a block subject (t=10).
func (b *ProofBundle) IsBlock() bool { return b.T == ptype.Block }

// IsItem returns true if this bundle is an item subject (t=20).
func (b *ProofBundle) IsItem() bool { return b.T == ptype.Item }

// IsEntropy returns true if this bundle is an entropy subject (t in {30,31,32}).
func (b *ProofBundle) IsEntropy() bool { return ptype.IsEntropySubject(b.T) }

// MarshalJSON produces the compact JSON wire format from a parsed
// ProofBundle. Subject and InclusionProof are omitted for block subjects.
// Used when sending a CBOR-decoded proof to the API as JSON.
func (b *ProofBundle) MarshalJSON() ([]byte, error) {
	m := map[string]any{
		"v":   b.Version,
		"t":   uint16(b.T),
		"ts":  b.Timestamp,
		"pk":  b.PublicKey,
		"sig": b.Signature,
		"b":   b.Block,
		"cx":  b.Commitments,
	}
	if b.T != ptype.Block {
		m["s"] = b.Subject
		m["ip"] = b.InclusionProof
	}
	return json.Marshal(m)
}

// Subject represents the unified subject within a non-block proof bundle.
// For item proofs (T=20), Data contains the claims map. For entropy
// subjects (T in {30,31,32}), Data contains the entropy observation.
// The source is carried in the top-level ProofBundle.T field; no per-
// subject `src` discriminator is emitted.
type Subject struct {
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

// ExternalCommit represents a commitment entry in the proof bundle.
// Type is an integer code from ptype (CommitmentStellar=40, CommitmentBitcoin=41).
// Each commitment carries an epoch Merkle proof (ep) linking the block hash
// to the committed value (Stellar: memo, Bitcoin: OP_RETURN payload).
type ExternalCommit struct {
	Type    ptype.Code `json:"t"`
	Network string     `json:"net"` // Stellar: "testnet"|"public" ; Bitcoin: "regtest"|"testnet"|"mainnet"

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

// FindCommitByType returns the first external commitment with the given
// type code, or nil.
func FindCommitByType(commits []ExternalCommit, t ptype.Code) *ExternalCommit {
	for i := range commits {
		if commits[i].Type == t {
			return &commits[i]
		}
	}
	return nil
}
