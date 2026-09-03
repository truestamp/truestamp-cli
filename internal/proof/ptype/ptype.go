// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: MIT

// Package ptype holds the frozen registry of proof type codes and the
// registry names that travel on the wire.
//
// A bundle's `type` is a registry NAME (`block`, `beacon`, `item`,
// `entropy_nist`, `entropy_stellar`, `entropy_bitcoin`). The signed payload
// (Appendix E.16) carries the frozen integer CODE for that name, so the two
// halves of the registry are both permanent: a name is never renamed and
// never reused, a code is never renumbered and never reassigned.
//
// Category ranges (blocks of 10):
//
//	10-19  subject: block family  (10 = block, 11 = beacon)
//	20-29  subject: item
//	30-39  subject: entropy sources
//	40-49  commitment chains (reserved; never on the wire)
//	50+    reserved for future categories
//
// Beacon (11) shares the block structural shape (no `subject`, no
// `inclusion_proof`, same `block` + `commitments`) but is a distinct code
// for domain separation: because the code is part of the signed payload, a
// `block` and a `beacon` bundle for the same block produce different
// signatures.
//
// The commitment chains `stellar` (40) and `bitcoin` (41) are named on the
// wire by `commitments[].chain` only; their codes are documented history and
// are reserved so nothing else ever claims them.
package ptype

// Code is the frozen integer code for a registry name. Subject codes are
// emitted as uint16 big-endian in the signed payload.
type Code uint16

// Registered subject type codes.
const (
	Block          Code = 10
	Beacon         Code = 11
	Item           Code = 20
	EntropyNIST    Code = 30
	EntropyStellar Code = 31
	EntropyBitcoin Code = 32
)

// Reserved commitment chain codes. They are never written to the wire and
// never appear in a preimage; `commitments[].chain` names the chain instead.
const (
	CommitmentStellar Code = 40
	CommitmentBitcoin Code = 41
)

// Registry names of the six subject types, in code order.
const (
	NameBlock          = "block"
	NameBeacon         = "beacon"
	NameItem           = "item"
	NameEntropyNIST    = "entropy_nist"
	NameEntropyStellar = "entropy_stellar"
	NameEntropyBitcoin = "entropy_bitcoin"
)

// Chain names carried in `commitments[].chain`.
const (
	ChainStellar = "stellar"
	ChainBitcoin = "bitcoin"
)

// SubjectNames lists every registered subject type name, in code order. It
// is the vocabulary a `type` field must draw from (Appendix E.6, E.24).
var SubjectNames = []string{
	NameBlock, NameBeacon, NameItem, NameEntropyNIST, NameEntropyStellar, NameEntropyBitcoin,
}

var codeByName = map[string]Code{
	NameBlock:          Block,
	NameBeacon:         Beacon,
	NameItem:           Item,
	NameEntropyNIST:    EntropyNIST,
	NameEntropyStellar: EntropyStellar,
	NameEntropyBitcoin: EntropyBitcoin,
}

// FromName resolves a registry name to its frozen code. ok is false for any
// string that is not one of the six subject names; a verifier MUST reject
// such a `type` at E.6 rather than guess a code (Appendix E.16, E.24).
func FromName(name string) (Code, bool) {
	c, ok := codeByName[name]
	return c, ok
}

// IsSubjectName reports whether name is one of the six registered subject
// type names.
func IsSubjectName(name string) bool {
	_, ok := codeByName[name]
	return ok
}

// IsValidSubject reports whether c is a registered subject type code.
func IsValidSubject(c Code) bool {
	switch c {
	case Block, Beacon, Item, EntropyNIST, EntropyStellar, EntropyBitcoin:
		return true
	}
	return false
}

// IsEntropySubject reports whether c is one of the entropy subject types.
func IsEntropySubject(c Code) bool {
	switch c {
	case EntropyNIST, EntropyStellar, EntropyBitcoin:
		return true
	}
	return false
}

// IsBlockLikeSubject reports whether c is a block-family subject type, a
// plain block (10) or a beacon (11). The two share one wire shape: no
// `subject`, no `inclusion_proof`, `subject_hash == block_hash`.
func IsBlockLikeSubject(c Code) bool {
	switch c {
	case Block, Beacon:
		return true
	}
	return false
}

// IsChainName reports whether name is one of the two registered commitment
// chain names.
func IsChainName(name string) bool {
	return name == ChainStellar || name == ChainBitcoin
}

// Name returns the registry name for a code. Returns "unknown" for an
// unregistered code.
func Name(c Code) string {
	switch c {
	case Block:
		return NameBlock
	case Beacon:
		return NameBeacon
	case Item:
		return NameItem
	case EntropyNIST:
		return NameEntropyNIST
	case EntropyStellar:
		return NameEntropyStellar
	case EntropyBitcoin:
		return NameEntropyBitcoin
	case CommitmentStellar:
		return ChainStellar
	case CommitmentBitcoin:
		return ChainBitcoin
	}
	return "unknown"
}

// Humanize returns a display-friendly label for a code.
func Humanize(c Code) string {
	switch c {
	case Block:
		return "Block"
	case Beacon:
		return "Beacon"
	case Item:
		return "Item"
	case EntropyNIST:
		return "NIST Beacon"
	case EntropyStellar:
		return "Stellar Ledger"
	case EntropyBitcoin:
		return "Bitcoin Block"
	case CommitmentStellar:
		return "Stellar"
	case CommitmentBitcoin:
		return "Bitcoin"
	}
	return "unknown"
}
