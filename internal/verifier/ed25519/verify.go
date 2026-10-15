// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: Apache-2.0

// This file ports verify/3 of the reference verifier's
// Truestamp.Verifier.Ed25519 (elixir/lib/truestamp/verifier/ed25519.ex in
// truestamp-verification), checked against the Truestamp application's
// port, verify_signature/3 of Truestamp.Proof.Verifier.Gate
// (lib/truestamp/proof/verifier/gate.ex in truestamp-v2): Verify(A, msg, S)
// of SPEC.md section 6.2 (Strict verification), the rule SIGNATURE (section
// 10.2, step 6) checks V under K by.
//
//	| Step                                                          | Refusal            |
//	|---------------------------------------------------------------|--------------------|
//	| 2. y, A with its highest bit cleared, is below p              | [ErrYNotBelowP]    |
//	| 3. A is not one of SMALL_ORDER_KEYS                           | [ErrSmallOrderKey] |
//	| 4. A decodes as a point, as RFC 8032 section 5.1.3 does       | [ErrNotOnCurve]    |
//	| 5. s, the last 32 octets of S, little-endian, is below L      | [ErrSNotBelowL]    |
//	| 6. k = SHA-512(R || A || msg) mod L                           | (cannot refuse)    |
//	| 7. the encoding of [s]B - [k]A equals R, S's first 32 octets  | [ErrEquation]      |
//
// Step 1, the canonical base64url spelling of A and S, is the caller's:
// ENVELOPE decodes K and V through internal/registry, which matches B64U32
// and B64U64 before it decodes.
//
// Adapted from the reference, which makes every step with RFC 8032's own
// arithmetic: here steps 2 and 3 are [CheckKeyBytes], and Go's
// crypto/ed25519 then makes steps 4 to 7, as SPEC.md section 6.2
// (informative) says it does for every key that passes steps 2 and 3,
// cofactorless and with R compared as its encoding. The library alone
// decides whether a signature that passes steps 2 and 3 verifies. When it
// refuses one, the refusal is named by the first of steps 4, 5 and 7 that
// fails, as the reference names it, from a check of the point's decoding
// and of s made here with math/big; that check names the refusal and
// never turns a refusal into a pass.

package ed25519

import (
	goed25519 "crypto/ed25519"
	"errors"
	"math/big"
)

// The refusals of Verify past the checks of a key's bytes, by the step of
// SPEC.md section 6.2 that refuses.
var (
	// ErrNotOnCurve is step 4: no point of the curve has the encoding A.
	ErrNotOnCurve = errors.New("not_on_curve")
	// ErrSNotBelowL is step 5: s is not below L.
	ErrSNotBelowL = errors.New("s_not_below_l")
	// ErrEquation is step 7: the encoding of [s]B - [k]A is not R.
	ErrEquation = errors.New("equation")
)

// l is L = 2^252 + 27742317777372353535851937790883648493, the order of
// the base point (RFC 8032 section 5.1).
var l = func() *big.Int {
	tail, ok := new(big.Int).SetString("27742317777372353535851937790883648493", 10)
	if !ok {
		panic("ed25519: L does not parse")
	}
	return tail.Add(tail, new(big.Int).Lsh(big.NewInt(1), 252))
}()

// d is the curve constant -121665/121666 mod p (RFC 8032 section 5.1).
var d = func() *big.Int {
	inverse := new(big.Int).ModInverse(big.NewInt(121666), p)
	value := new(big.Int).Mul(big.NewInt(-121665), inverse)
	return value.Mod(value, p)
}()

// Verify is Verify(A, msg, S) of SPEC.md section 6.2: nil when the 64
// octets signature verify under the 32-octet public key key over message,
// and otherwise the refusal of the first step that fails: [ErrYNotBelowP],
// [ErrSmallOrderKey], [ErrNotOnCurve], [ErrSNotBelowL] or [ErrEquation].
// Every message Truestamp signs is 32 octets (section 4.1), but the rule
// holds for any.
func Verify(key [32]byte, message []byte, signature [64]byte) error {
	if err := CheckKeyBytes(key); err != nil {
		return err
	}
	if goed25519.Verify(key[:], message, signature[:]) {
		return nil
	}
	switch {
	case !onCurve(key):
		return ErrNotOnCurve
	case littleEndian(signature[32:]).Cmp(l) >= 0:
		return ErrSNotBelowL
	}
	return ErrEquation
}

// onCurve reports whether key, whose y is below p, decodes as a point (RFC
// 8032 section 5.1.3): whether x^2 = (y^2 - 1) / (d y^2 + 1) has a root mod
// p. The decoding also refuses x = 0 with the sign bit set, which only y =
// 1 and y = p - 1 give; steps 2 and 3 have refused all four of their
// encodings, which are in SMALL_ORDER_KEYS.
func onCurve(key [32]byte) bool {
	y2 := new(big.Int).Mul(y(key), y(key))
	numerator := new(big.Int).Sub(y2, big.NewInt(1))
	denominator := new(big.Int).Mul(d, y2)
	denominator.Add(denominator, big.NewInt(1)).Mod(denominator, p)
	x2 := numerator.Mul(numerator, new(big.Int).ModInverse(denominator, p))
	x2.Mod(x2, p)
	return new(big.Int).ModSqrt(x2, p) != nil
}

// littleEndian reads octets as a little-endian unsigned integer.
func littleEndian(octets []byte) *big.Int {
	bigEndian := make([]byte, len(octets))
	for i, b := range octets {
		bigEndian[len(octets)-1-i] = b
	}
	return new(big.Int).SetBytes(bigEndian)
}
