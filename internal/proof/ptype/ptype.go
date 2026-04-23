// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: MIT

// Package ptype holds the frozen registry of proof type integer codes used
// by Truestamp proof bundles. The registry discriminates both subject
// types (bundle["t"]) and external commitment chain types (bundle["cx"][].t).
//
// The registry is a long-lived contract:
//   - Existing codes are never reused or renumbered.
//   - New codes claim unused integers in the appropriate category.
//
// Category ranges (blocks of 10):
//
//	10-19  subject: block family  (10 = block, 11 = beacon)
//	20-29  subject: item
//	30-39  subject: entropy sources
//	40-49  external commitment chains
//	50+    reserved for future categories
//
// Beacon (t=11) shares the block structural shape — no `s`, no `ip`, same
// `b` + `cx` — but is a distinct type code for domain separation. Because
// `t` is part of the signing payload, a t=10 and t=11 bundle for the same
// block produce different signatures.
package ptype

// Code is the integer type code used to identify the subject of a proof
// bundle or the chain of an external commitment. Codes are u16 on the wire
// (in the signed payload); all currently-registered codes fit in one byte.
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

// Registered external commitment type codes.
const (
	CommitmentStellar Code = 40
	CommitmentBitcoin Code = 41
)

// IsValidSubject reports whether c is a registered subject type code.
func IsValidSubject(c Code) bool {
	switch c {
	case Block, Beacon, Item, EntropyNIST, EntropyStellar, EntropyBitcoin:
		return true
	}
	return false
}

// IsValidExternalCommitment reports whether c is a registered external
// commitment type code.
func IsValidExternalCommitment(c Code) bool {
	switch c {
	case CommitmentStellar, CommitmentBitcoin:
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

// IsBlockLikeSubject reports whether c is a block-family subject type —
// either a plain block (t=10) or a beacon (t=11). The two share the same
// wire shape: no `s`, no `ip`, `subject_hash == block_hash`. Verification
// branches that skip subject / inclusion-proof / subject-hash-derivation
// steps should key off this helper rather than `c == Block`.
func IsBlockLikeSubject(c Code) bool {
	switch c {
	case Block, Beacon:
		return true
	}
	return false
}

// Name returns the lowercase atom-style name for a code, matching the
// reference Elixir implementation. Used as the JSON subject_type value and
// as an internal dispatch key. Returns "unknown" for unregistered codes.
func Name(c Code) string {
	switch c {
	case Block:
		return "block"
	case Beacon:
		return "beacon"
	case Item:
		return "item"
	case EntropyNIST:
		return "entropy_nist"
	case EntropyStellar:
		return "entropy_stellar"
	case EntropyBitcoin:
		return "entropy_bitcoin"
	case CommitmentStellar:
		return "commitment_stellar"
	case CommitmentBitcoin:
		return "commitment_bitcoin"
	}
	return "unknown"
}

// Humanize returns a display-friendly label for a code, used in the pretty
// verify output.
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
