// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: Apache-2.0

// This file ports the reference verifier's Truestamp.Verifier.KeyId
// (elixir/lib/truestamp/verifier/key_id.ex in truestamp-verification): a
// key's key id of SPEC.md section 3.3, the hex of the first 4 bytes of
// SHA-256(0x60 || its 32 bytes). It adds the prerotation commitment of the
// same section, the hex of SHA-256(0x61 || the next key's 32 bytes), which
// the reference does not compute: a proof's prerotation_commitment is
// RECORDED (section 3.7), so no step recomputes it from the proof. Both are
// outside the rule of section 4.1: they hash the raw key bytes, so neither
// depends on how the key is written. Adapted from the reference: a key is a
// [32]byte, so a key of any other length cannot be passed.
//
// A key id is only an index (section 4.1): a verifier never reads one a
// document carries to choose or to check a signer (section 6.1). It
// computes one to check a KID the document carries (section 10.3) and when
// it loads a keyring (section 10.2, step 0).

package registry

import "encoding/hex"

// KeyIDOf returns the key id of the 32-octet public key key, as 8 lowercase
// hex characters (the class KID of SPEC.md section 2.4).
func KeyIDOf(key [32]byte) string {
	sum := digest(KeyID, key[:])
	return hex.EncodeToString(sum[:4])
}

// PrerotationCommitmentOf returns the prerotation commitment of the
// 32-octet public key next, as 64 lowercase hex characters (HEX64).
func PrerotationCommitmentOf(next [32]byte) string {
	sum := digest(PrerotationCommitment, next[:])
	return hex.EncodeToString(sum[:])
}
