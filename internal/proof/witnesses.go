// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: MIT

package proof

// The witness registry (Appendix E.17a). A witness is a public record that
// existed before the submission and that the subject's composite
// fingerprint commits to; witnesses open the submitted-after edge of the
// submission window. A commitment, by contrast, is a public-chain
// transaction recorded after the submission; commitments close the
// submitted-before edge and are never witnesses.
//
// Names are never renamed and never removed. A new witness type claims a
// new name, and a verifier that meets a name it does not know reports a
// visible `skip`, never a failure (E.24).
const (
	WitnessBlock           = "block"
	WitnessEntropyStellar  = "entropy_stellar"
	WitnessEntropyNIST     = "entropy_nist"
	WitnessEntropyBitcoin  = "entropy_bitcoin"
	WitnessSigningKeyEvent = "signing_key_event"
)

// WitnessNames is every registered witness name, in registry order. It is
// the vocabulary the `witnesses` argument of proof generation accepts.
var WitnessNames = []string{
	WitnessBlock, WitnessEntropyStellar, WitnessEntropyNIST, WitnessEntropyBitcoin, WitnessSigningKeyEvent,
}

// CommittedWitnessNames are the witnesses an item's metadata commits to.
// `signing_key_event` is a witness of the signature, carried at the top
// level, and is never committed in the subject metadata.
var CommittedWitnessNames = []string{
	WitnessBlock, WitnessEntropyStellar, WitnessEntropyNIST, WitnessEntropyBitcoin,
}

// EntropyWitnessNames are the three entropy witnesses, each hashed under
// 0x21 over the JCS of its carried payload.
var EntropyWitnessNames = []string{
	WitnessEntropyStellar, WitnessEntropyNIST, WitnessEntropyBitcoin,
}

// IsWitnessName reports whether name is a registered witness name.
func IsWitnessName(name string) bool {
	for _, n := range WitnessNames {
		if n == name {
			return true
		}
	}
	return false
}

// IsCommittedWitnessName reports whether name is a witness the subject
// metadata can commit to (every registered name but signing_key_event).
func IsCommittedWitnessName(name string) bool {
	for _, n := range CommittedWitnessNames {
		if n == name {
			return true
		}
	}
	return false
}

// IsEntropyWitnessName reports whether name is one of the three entropy
// witnesses.
func IsEntropyWitnessName(name string) bool {
	for _, n := range EntropyWitnessNames {
		if n == name {
			return true
		}
	}
	return false
}

// WitnessBasis names the time a witness publishes, in the words Appendix
// E.20's report rows are required to use. "" for a witness that publishes
// no time of its own.
func WitnessBasis(name string) string {
	switch name {
	case WitnessBlock:
		return "Truestamp-asserted: the head block id's mint time"
	case WitnessEntropyStellar:
		return "Stellar ledger close time"
	case WitnessEntropyNIST:
		return "NIST beacon pulse publication time"
	case WitnessEntropyBitcoin:
		return "Bitcoin block header time, read conservatively (a header time may run up to two hours ahead of real time)"
	}
	return ""
}
