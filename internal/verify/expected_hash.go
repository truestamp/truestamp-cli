// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: Apache-2.0

package verify

import (
	"fmt"
	"strings"

	"github.com/truestamp/truestamp-cli/internal/proof"
	"github.com/truestamp/truestamp-cli/internal/tscrypto"
)

// stepExpectedHash compares the caller's expected hash, r.ExpectedHash,
// with the subject.claims.hash of the bundle the remote path posted, and
// adds its row to r. It runs only for an item subject; for every other
// type a supplied hash is reported as a skip rather than discarded. When
// no hash is supplied the row warns and never fails.
func stepExpectedHash(r *Report, bundle *proof.Bundle) {
	expected := r.ExpectedHash
	if !bundle.IsItem() {
		if expected != "" {
			r.skip(groupHashComparison, CatDataIntegrity,
				"--expected-hash ignored: only an item subject commits to a file hash")
		}
		return
	}

	claimed := claimsHash(bundle)
	switch {
	case expected == "" && claimed != "":
		r.warn(groupHashComparison, CatDataIntegrity, fmt.Sprintf(
			"file hash not verified: the proof commits to %s but no --expected-hash was supplied", claimed))
	case expected == "":
		r.skip(groupHashComparison, CatDataIntegrity, "no file hash in this proof")
	case claimed == "":
		r.warn(groupHashComparison, CatDataIntegrity,
			"an expected hash was supplied but this proof commits to no file hash")
	default:
		r.check(groupHashComparison, CatDataIntegrity,
			tscrypto.SecureEqual(expected, strings.ToLower(claimed)),
			"supplied file hash matches subject.claims.hash",
			fmt.Sprintf("supplied file hash does NOT match subject.claims.hash (proof commits to %s)", claimed))
	}
}

// claimsHash returns subject.claims.hash when the claims map carries it as
// a non-empty string.
func claimsHash(bundle *proof.Bundle) string {
	if bundle.Subject == nil {
		return ""
	}
	claims, ok := bundle.Subject.Fields.Object("claims")
	if !ok {
		return ""
	}
	return claims.Str("hash")
}
