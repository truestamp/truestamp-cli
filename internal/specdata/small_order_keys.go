// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: Apache-2.0

// This file holds SMALL_ORDER_KEYS (SPEC.md sections 6.2 and 9),
// transcribed from spec/small-order-keys.json and held equal to it by
// TestAgreement_SmallOrderKeys. Strict Ed25519 verification (section 6.2),
// which internal/verifier/ed25519 makes, reads this table.

package specdata

import (
	"encoding/hex"
	"fmt"
	"slices"
)

// SmallOrderKey is one entry of SMALL_ORDER_KEYS (SPEC.md sections 6.2 and
// 9): a 32-byte encoding of a point of small order, the point's order, and
// whether the encoding is the sign-bit variant of a canonical one.
type SmallOrderKey struct {
	Key            [32]byte
	Order          int
	SignBitVariant bool
}

// smallOrderKeys is spec/small-order-keys.json, in its order.
var smallOrderKeys = []SmallOrderKey{
	{Key: key32("0100000000000000000000000000000000000000000000000000000000000000"), Order: 1},
	{Key: key32("ecffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff7f"), Order: 2},
	{Key: key32("0000000000000000000000000000000000000000000000000000000000000000"), Order: 4},
	{Key: key32("0000000000000000000000000000000000000000000000000000000000000080"), Order: 4},
	{Key: key32("26e8958fc2b227b045c3f489f2ef98f0d5dfac05d3c63339b13802886d53fc05"), Order: 8},
	{Key: key32("26e8958fc2b227b045c3f489f2ef98f0d5dfac05d3c63339b13802886d53fc85"), Order: 8},
	{Key: key32("c7176a703d4dd84fba3c0b760d10670f2a2053fa2c39ccc64ec7fd7792ac037a"), Order: 8},
	{Key: key32("c7176a703d4dd84fba3c0b760d10670f2a2053fa2c39ccc64ec7fd7792ac03fa"), Order: 8},
	{Key: key32("0100000000000000000000000000000000000000000000000000000000000080"), Order: 1, SignBitVariant: true},
	{Key: key32("ecffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff"), Order: 2, SignBitVariant: true},
}

// SmallOrderKeys returns SMALL_ORDER_KEYS in the order of
// spec/small-order-keys.json. Each call returns a new slice.
func SmallOrderKeys() []SmallOrderKey {
	return slices.Clone(smallOrderKeys)
}

// key32 decodes a 64-digit hex literal of the table above.
func key32(text string) [32]byte {
	var key [32]byte
	decoded, err := hex.DecodeString(text)
	if err != nil || len(decoded) != len(key) {
		panic(fmt.Sprintf("specdata: %q is not 32 bytes of hex", text))
	}
	copy(key[:], decoded)
	return key
}
