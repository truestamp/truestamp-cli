// Copyright (c) 2021-2026 Truestamp, Inc.
// SPDX-License-Identifier: MIT

package verify

import "github.com/truestamp/truestamp-cli/internal/proof"

// parseProofBytes wraps proof.ParseBytes for tests that need to build
// CBOR payloads without importing the proof package directly.
func parseProofBytes(data []byte) (*proof.ProofBundle, error) {
	return proof.ParseBytes(data)
}
