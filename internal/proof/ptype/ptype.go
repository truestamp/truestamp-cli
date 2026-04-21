// Copyright (c) 2021-2026 Truestamp, Inc.
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
//	10-19  subject: block
//	20-29  subject: item
//	30-39  subject: entropy sources
//	40-49  external commitment chains
//	50+    reserved for future categories
package ptype

// Code is the integer type code used to identify the subject of a proof
// bundle or the chain of an external commitment. Codes are u16 on the wire
// (in the signed payload); all currently-registered codes fit in one byte.
type Code uint16

// Registered subject type codes.
const (
	Block          Code = 10
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
	case Block, Item, EntropyNIST, EntropyStellar, EntropyBitcoin:
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

// Name returns the lowercase atom-style name for a code, matching the
// reference Elixir implementation. Used as the JSON subject_type value and
// as an internal dispatch key. Returns "unknown" for unregistered codes.
func Name(c Code) string {
	switch c {
	case Block:
		return "block"
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
