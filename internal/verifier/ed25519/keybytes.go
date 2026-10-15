// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: Apache-2.0

// Package ed25519 holds the strict Ed25519 rule of SPEC.md section 6.2
// (Strict verification) for the new verifier: [CheckKeyBytes], the checks
// of a key's bytes (this file), and [Verify], Verify(A, msg, S), which makes
// them and then calls Go's crypto/ed25519 (verify.go).
//
// This file ports the checks of a key's bytes of the reference verifier's
// Truestamp.Verifier.Ed25519 (elixir/lib/truestamp/verifier/ed25519.ex in
// truestamp-verification), check_key_bytes/1: steps 2 and 3 of Verify(A,
// msg, S), which a verifier also makes outside Verify, on K at ENVELOPE
// (SPEC.md section 10.2, step 3), on each key of a keyring it loads (step
// 0), and on each key event's public_key at DERIVE. Go's crypto/ed25519
// makes neither step, so they are made here, before any library is called.
//
// Adapted from the reference: a key is a [32]byte, so a key of any other
// length cannot be passed; y is compared with p through math/big; and
// SMALL_ORDER_KEYS is internal/specdata's transcription of
// spec/small-order-keys.json, which specdata's tests hold to the file.
package ed25519

import (
	"errors"
	"math/big"

	"github.com/truestamp/truestamp-cli/internal/specdata"
)

// The refusals of the checks of a key's bytes, by the step of SPEC.md
// section 6.2 that refuses.
var (
	// ErrYNotBelowP is step 2: y, the key read as a little-endian integer
	// with its highest bit cleared, is not below p = 2^255 - 19.
	ErrYNotBelowP = errors.New("y_not_below_p")
	// ErrSmallOrderKey is step 3: the key is one of SMALL_ORDER_KEYS.
	ErrSmallOrderKey = errors.New("small_order_key")
)

// p is 2^255 - 19, the field prime of RFC 8032 section 5.1.
var p = new(big.Int).Sub(new(big.Int).Lsh(big.NewInt(1), 255), big.NewInt(19))

// smallOrderKeys is SMALL_ORDER_KEYS, read once.
var smallOrderKeys = specdata.SmallOrderKeys()

// CheckKeyBytes makes steps 2 and 3 of SPEC.md section 6.2 on key, in that
// order: nil when key passes both, and otherwise [ErrYNotBelowP] or
// [ErrSmallOrderKey], the first that refuses.
func CheckKeyBytes(key [32]byte) error {
	if y(key).Cmp(p) >= 0 {
		return ErrYNotBelowP
	}
	for _, small := range smallOrderKeys {
		if key == small.Key {
			return ErrSmallOrderKey
		}
	}
	return nil
}

// y is the little-endian integer of key with bit 255, the sign of x,
// cleared (RFC 8032 section 5.1.3).
func y(key [32]byte) *big.Int {
	var bigEndian [32]byte
	for i, b := range key {
		bigEndian[len(key)-1-i] = b
	}
	value := new(big.Int).SetBytes(bigEndian[:])
	return value.SetBit(value, 255, 0)
}
