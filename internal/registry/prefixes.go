// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: Apache-2.0

// Package registry is the byte-prefix registry of SPEC.md section 4 and
// the hashes computed under it. It is named for that registry: each of the
// thirteen bytes of SPEC.md section 4.3 is applied here, so every record
// hash H(p, R), the key id, the prerotation commitment and the one Merkle
// walk the SPEC.md verifier computes go through this package. The hashes
// SPEC.md section 4.1 puts outside the registry do not: Bitcoin's SHA-256d
// values and the content digests (the constructions address, a keyring
// file, a served proof file).
//
//   - the table of section 4.3, with each row's name, [Construction.String]
//     (this file);
//   - the record hash H(p, R) of sections 2.2 and 4.1, [RecordHash]
//     (record_hash.go);
//   - the key id and the prerotation commitment of sections 3.3 and 4.1,
//     [KeyIDOf] and [PrerotationCommitmentOf], which sit outside H(p, R)
//     (key_id.go);
//   - the inclusion walk of section 5.3, [Walk], the one place the leaf and
//     node bytes 0x00 and 0x01 are used (walk.go);
//   - the canonical base64url of the pattern classes B64U32 and B64U64 of
//     section 2.4, through which the 32 key bytes a key id hashes are
//     decoded (base64url.go).
//
// A row's byte is not exported: a byte is applied only by digest, which
// refuses the leaf and node, and by the walk, so no other package can take
// the leaf or node byte from here.
//
// Rule 2 of SPEC.md section 10.6: prefix bytes and member sets come only
// from compiled constants. A construction is named by a [Construction]
// constant, never by a value a document carries, and nothing here reads a
// prefix byte or a member name from a document. rule2_test.go holds this
// package, and each package of the new verifier it lists, to that, and
// fails a package that imports this one without being on its list.
//
// This file holds the table of SPEC.md section 4.3 as compiled constants,
// as section 4 asks of each of Truestamp's implementations: transcribed
// from spec/prefixes.json and held equal to it, and to section 4.3, by the
// tests of prefixes_test.go. A row is named by a Construction constant, and
// digest, the one function that applies a row's byte, lives beside the
// table.
package registry

import (
	"crypto/sha256"
	"strconv"
)

// Construction names one row of the registry of SPEC.md section 4.3. Its
// values are the constants below, in the order of their bytes; the zero
// value names no row.
type Construction int

// The rows of SPEC.md section 4.3, in the order of their bytes.
const (
	// MerkleLeaf is 0x00, the Merkle leaf, used only inside [Walk].
	MerkleLeaf Construction = iota + 1
	// MerkleNode is 0x01, the Merkle node, used only inside [Walk].
	MerkleNode
	// ClaimsHash is 0x10, claims_hash.
	ClaimsHash
	// ItemHash is 0x11, item_hash.
	ItemHash
	// EntropyHash is 0x20, entropy_hash.
	EntropyHash
	// ObservationHash is 0x21, observation_hash.
	ObservationHash
	// GenesisPreviousBlockHash is 0x30, the genesis previous_block_hash.
	GenesisPreviousBlockHash
	// BlockHash is 0x31, block_hash.
	BlockHash
	// EpochHash is 0x40, epoch_hash.
	EpochHash
	// ProofHash is 0x50, proof_hash.
	ProofHash
	// KeyID is 0x60, the key id, outside H(p, R).
	KeyID
	// PrerotationCommitment is 0x61, the prerotation commitment, outside
	// H(p, R).
	PrerotationCommitment
	// Handover is 0x62, the handover a rotation's outgoing key signs.
	Handover
)

// entry is one row of the registry, with the members spec/prefixes.json
// gives it.
type entry struct {
	// Name is the row's key in spec/prefixes.json, such as
	// "genesis_previous_block_hash".
	Name string
	// Byte is the prefix byte.
	Byte byte
	// Construction is the construction's name as SPEC.md section 4.3
	// writes it, such as "genesis previous_block_hash".
	Construction string
	// Family is the family the high nibble names.
	Family string
	// Hashed is what is hashed after the byte.
	Hashed string
	// Signed says whether Ed25519 signs the hash (SPEC.md section 4.1).
	Signed bool
	// Signer names the key that signs a signed kind, and is empty for the
	// others.
	Signer string
}

// table is spec/prefixes.json, indexed by Construction - 1.
var table = [...]entry{
	{Name: "merkle_leaf", Byte: 0x00, Construction: "leaf", Family: "Merkle", Hashed: "the 32 bytes of the leaf value"},
	{Name: "merkle_node", Byte: 0x01, Construction: "node", Family: "Merkle", Hashed: "the 32 bytes of the left child, then of the right"},
	{Name: "claims_hash", Byte: 0x10, Construction: "claims_hash", Family: "Items", Hashed: "JCS(claims)"},
	{Name: "item_hash", Byte: 0x11, Construction: "item_hash", Family: "Items", Hashed: "JCS(item record)"},
	{Name: "entropy_hash", Byte: 0x20, Construction: "entropy_hash", Family: "Entropy", Hashed: "JCS(entropy object)"},
	{Name: "observation_hash", Byte: 0x21, Construction: "observation_hash", Family: "Entropy", Hashed: "JCS(observation record)"},
	{Name: "genesis_previous_block_hash", Byte: 0x30, Construction: "genesis previous_block_hash", Family: "Blocks", Hashed: "JCS(genesis object)"},
	{Name: "block_hash", Byte: 0x31, Construction: "block_hash", Family: "Blocks", Hashed: "JCS(block record)", Signed: true, Signer: "chain_assigned_key"},
	{Name: "epoch_hash", Byte: 0x40, Construction: "epoch_hash", Family: "Epochs", Hashed: "JCS(epoch record)"},
	{Name: "proof_hash", Byte: 0x50, Construction: "proof_hash", Family: "Proofs", Hashed: "JCS(proof)", Signed: true, Signer: "active_key"},
	{Name: "key_id", Byte: 0x60, Construction: "key_id", Family: "Keys", Hashed: "the 32 key bytes; the hash cut to 4 bytes"},
	{Name: "prerotation_commitment", Byte: 0x61, Construction: "prerotation_commitment", Family: "Keys", Hashed: "the 32 bytes of the next key"},
	{Name: "handover", Byte: 0x62, Construction: "handover", Family: "Keys", Hashed: "JCS(rotation event without handover_signature)", Signed: true, Signer: "outgoing_key"},
}

// constructions returns every row of the registry, in the order of their
// bytes. Each call returns a new slice.
func constructions() []Construction {
	return []Construction{
		MerkleLeaf, MerkleNode, ClaimsHash, ItemHash, EntropyHash,
		ObservationHash, GenesisPreviousBlockHash, BlockHash, EpochHash,
		ProofHash, KeyID, PrerotationCommitment, Handover,
	}
}

// entry returns c's row of the registry, and false when c names no row.
func (c Construction) entry() (entry, bool) {
	if c < MerkleLeaf || c > Handover {
		return entry{}, false
	}
	return table[c-MerkleLeaf], true
}

// String returns c's name in spec/prefixes.json, or "Construction(n)" when
// c names no row.
func (c Construction) String() string {
	if row, ok := c.entry(); ok {
		return row.Name
	}
	return "Construction(" + strconv.Itoa(int(c)) + ")"
}

// digest returns SHA-256 over c's byte followed by data: the one place this
// package applies a prefix byte, the walk aside. Every caller names c by a
// constant of the registry or, in RecordHash, by a kind it has checked is
// under the rule. The Merkle leaf and node bytes are used only inside the
// walk (SPEC.md section 10.6, rule 2), so digest refuses them, and a value
// that names no row, by panicking: either is a defect in this package.
func digest(c Construction, data []byte) [32]byte {
	row, ok := c.entry()
	if !ok || c == MerkleLeaf || c == MerkleNode {
		panic("registry: digest under " + c.String())
	}
	h := sha256.New()
	h.Write([]byte{row.Byte})
	h.Write(data)
	var sum [32]byte
	h.Sum(sum[:0])
	return sum
}
