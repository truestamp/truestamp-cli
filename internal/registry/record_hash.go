// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: Apache-2.0

// This file ports the reference verifier's Truestamp.Verifier.RecordHash
// (elixir/lib/truestamp/verifier/record_hash.ex in truestamp-verification):
// the rule of SPEC.md section 4.1, the record hash of section 2.2,
//
//	H(p, R) = SHA-256(p || JCS(R))
//
// with p the one byte the registry (section 4.3) assigns to R's kind and R
// a JSON object within the value profile of section 10.1, hashed exactly as
// carried. Adapted from the reference: R is a parsed value of
// internal/strictjson, a map[string]any, written by internal/jcs's Encode;
// and where the reference raises on a kind outside the rule, RecordHash
// returns [ErrNotRecordKind]; and where the reference's is_map guard
// refuses a record that is not an object (raising on JSON null), RecordHash
// refuses a nil map, which would otherwise be written as {}, with
// [ErrNotObject].

package registry

import (
	"encoding/hex"
	"errors"
	"fmt"

	"github.com/truestamp/truestamp-cli/internal/jcs"
)

// ErrNotRecordKind marks a [Construction] that is not under the rule of
// SPEC.md section 4.1: the Merkle leaf and node, the key id and the
// prerotation commitment, which hash raw bytes rather than a record, and a
// value that names no row.
var ErrNotRecordKind = errors.New("registry: not a kind under the record rule")

// ErrNotObject marks a nil record. A record internal/strictjson parses is
// never nil: it parses {} as an empty map that is not nil. So a nil record
// is one that was never parsed, such as the zero value a failed type
// assertion gives for a member that is null or absent, and is refused
// rather than hashed as {}.
var ErrNotObject = errors.New("registry: the record is not an object")

// RecordKinds returns the kinds under the rule, in the order of their bytes:
// every row of SPEC.md section 4.3 that hashes JCS of a value. Each call
// returns a new slice.
func RecordKinds() []Construction {
	return []Construction{
		ClaimsHash, ItemHash, EntropyHash, ObservationHash,
		GenesisPreviousBlockHash, BlockHash, EpochHash, ProofHash, Handover,
	}
}

// isRecordKind reports whether kind is one of [RecordKinds].
func isRecordKind(kind Construction) bool {
	switch kind {
	case ClaimsHash, ItemHash, EntropyHash, ObservationHash,
		GenesisPreviousBlockHash, BlockHash, EpochHash, ProofHash, Handover:
		return true
	}
	return false
}

// RecordHash returns H(p, R), the 32 octets of SHA-256(p || JCS(record)),
// with p the byte of kind. It refuses, with [ErrNotRecordKind], a kind that
// is not one of [RecordKinds]; with [ErrNotObject], a nil record; and
// returns the error of [jcs.Encode] for a record that is not within the
// value profile, which no parsed value is. A parsed object is never nil, so
// a parsed {} is hashed as {}.
func RecordHash(kind Construction, record map[string]any) ([32]byte, error) {
	if !isRecordKind(kind) {
		return [32]byte{}, fmt.Errorf("%w: %s", ErrNotRecordKind, kind)
	}
	if record == nil {
		return [32]byte{}, fmt.Errorf("%w: %s", ErrNotObject, kind)
	}
	canonical, err := jcs.Encode(record)
	if err != nil {
		return [32]byte{}, err
	}
	return digest(kind, canonical), nil
}

// RecordHashHex returns [RecordHash] as 64 lowercase hex characters, the
// HEX64 a proof carries.
func RecordHashHex(kind Construction, record map[string]any) (string, error) {
	sum, err := RecordHash(kind, record)
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(sum[:]), nil
}
