// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: MIT

package proof

import (
	"encoding/json"

	"github.com/truestamp/truestamp-cli/internal/proof/ptype"
)

// ProofBundle is the top-level compact Truestamp proof structure (bundle
// version `v` is 1). Normative reference: Appendix E of
// `truestamp-v2/whitepaper/whitepaper.typ`; the wire shape is also
// described in `truestamp-v2/kb/verification/proof-bundle-format.md`.
//
// The top-level integer field `t` discriminates subject types:
//
//	10  block            (no Subject, no InclusionProof)
//	11  beacon           (no Subject, no InclusionProof — same shape as block)
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

	// Subject is nil for block-like subjects (t ∈ {10, 11}). For all other
	// subject types it is non-nil after a successful parse.
	Subject *Subject `json:"-"`

	Block       Block            `json:"-"`
	Commitments []ExternalCommit `json:"-"`

	// InclusionProof is "" for block-like subjects and non-empty otherwise.
	InclusionProof string `json:"-"` // base64url compact Merkle proof

	// RawData is the subject-data JSON preserved byte-for-byte for JCS.
	// Empty for block-like subjects.
	RawData json.RawMessage `json:"-"`
}

// Removed: IsBlock (t=10 only) and IsBeacon (t=11 only). Neither had a
// production caller, and the one predicate the pipeline needs is
// IsBlockLike — block and beacon share a wire shape, so every structural
// guard is a block-like guard. The strict pair existed mainly as a hazard
// to warn readers away from; deleting it removes the hazard instead.
//
// The t=10 / t=11 distinction is still cryptographically load-bearing —
// `t` is inside E.16's signed payload, so the two produce different
// signatures for the same block — and the places that care compare the
// raw code (ptype.Block / ptype.Beacon) directly, which is what is
// actually signed.

// IsBlockLike returns true if this bundle is either a block (t=10) or a
// beacon (t=11). Use this in verification-pipeline guards that skip
// subject / inclusion-proof / subject-hash-derivation steps — those are
// shape concerns and beacon proofs have the same shape as block proofs.
func (b *ProofBundle) IsBlockLike() bool { return ptype.IsBlockLikeSubject(b.T) }

// IsItem returns true if this bundle is an item subject (t=20).
func (b *ProofBundle) IsItem() bool { return b.T == ptype.Item }

// IsEntropy returns true if this bundle is an entropy subject (t in {30,31,32}).
func (b *ProofBundle) IsEntropy() bool { return ptype.IsEntropySubject(b.T) }

// MarshalJSON produces the compact JSON wire format from a parsed
// ProofBundle. Subject and InclusionProof are omitted for block-like
// subjects (t ∈ {10, 11}). Used when sending a CBOR-decoded proof to the
// API as JSON.
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
	if !b.IsBlockLike() {
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

// commitJSON mirrors ExternalCommit for marshalling. The two epoch-root
// keys are pointers so an *empty* value is still emitted while an
// *inapplicable* one stays absent — `omitempty` on a plain string cannot
// tell those apart.
type commitJSON struct {
	Type    ptype.Code `json:"t"`
	Network string     `json:"net"`

	EpochProof string `json:"ep"`

	TransactionHash string  `json:"tx,omitempty"`
	MemoHash        *string `json:"memo,omitempty"`
	Ledger          int     `json:"l,omitempty"`
	Timestamp       string  `json:"ts,omitempty"`

	OpReturn        *string `json:"op,omitempty"`
	RawTxHex        string  `json:"rtx,omitempty"`
	TxoutproofHex   string  `json:"txp,omitempty"`
	BlockMerkleRoot string  `json:"bmr,omitempty"`
	BlockHeight     int     `json:"h,omitempty"`
}

// MarshalJSON emits a commitment entry, always carrying `ep` and the
// chain-specific epoch-root key (`memo` for t=40, `op` for t=41) even when
// empty. E.6 hard-rejects an entry missing either, so letting `omitempty`
// drop an empty value would make the CLI emit a bundle its own parser
// refuses to read back.
func (c ExternalCommit) MarshalJSON() ([]byte, error) {
	out := commitJSON{
		Type:            c.Type,
		Network:         c.Network,
		EpochProof:      c.EpochProof,
		TransactionHash: c.TransactionHash,
		Ledger:          c.Ledger,
		Timestamp:       c.Timestamp,
		RawTxHex:        c.RawTxHex,
		TxoutproofHex:   c.TxoutproofHex,
		BlockMerkleRoot: c.BlockMerkleRoot,
		BlockHeight:     c.BlockHeight,
	}
	if c.Type == ptype.CommitmentStellar || c.MemoHash != "" {
		out.MemoHash = &c.MemoHash
	}
	if c.Type == ptype.CommitmentBitcoin || c.OpReturn != "" {
		out.OpReturn = &c.OpReturn
	}
	return json.Marshal(out)
}

// Removed: FindCommitByType. Appendix E.15 requires one graded result per
// `cx` entry, so every consumer in the verify pipeline iterates the slice
// in wire order and branches on each entry's own type — none of them wants
// "the first Stellar commitment". Its only caller was its own test.
