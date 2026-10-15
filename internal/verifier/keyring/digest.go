// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: Apache-2.0

// This file ports digest/1 of the reference verifier's
// Truestamp.Verifier.Keyring (elixir/lib/truestamp/verifier/keyring.ex in
// truestamp-verification), checked against the Truestamp application's
// test seam, whose digest/1 makes the same hash
// (test/support/proof_verifier_seam.ex in truestamp-v2): a keyring's digest
// d, the lowercase hex SHA-256 of its bytes as read (SPEC.md section 8.3
// (External verifiers)), which trust_root carries and a supplied keyring's
// attribution keyring:<d> names (section 10.5 (Verdict and report)).
//
// d is a content digest, which SPEC.md section 4.1 puts outside the
// byte-prefix registry: it hashes the bytes as read, with no prefix byte,
// so two spellings of one keyring have two digests. [Digest] is the one
// function of the new verifier outside internal/registry that may hash,
// and internal/registry/rule2_test.go names it as the one exemption from
// its check that crypto/sha256 is used only in the registry's digest.

package keyring

import (
	"crypto/sha256"
	"encoding/hex"
)

// Digest returns d, the lowercase hex SHA-256 of data, a keyring's bytes as
// read.
func Digest(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}
