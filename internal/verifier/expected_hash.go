// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: Apache-2.0

// This file holds step 13 of SPEC.md section 10.2, EXPECTED_HASH: the
// caller's hash ([Config.ExpectedHash]) against /subject/claims/hash. The
// step reports one row.
//
//   - No hash given: skip expected_hash_not_checked, part null.
//   - A hash given: one row at /subject/claims/hash, whatever the proof
//     holds there. It is pass only when the type is item,
//     /subject/claims/hash_type is a name of the table below,
//     /subject/claims/hash is a string, and the caller's value, lowercased,
//     is hex of the length the table gives for that hash_type and equals
//     /subject/claims/hash. Such a match under md5 or sha1 is warn
//     expected_hash_weak_algorithm in place of pass, since collisions are
//     practical for both, and the label does not change. Anything else is
//     fail expected_hash_mismatch, never a skip, with values saying why
//     under reason: no_listed_hash (the proof is not an item, or its claims
//     have no hash of a listed type), not_hex_of_length (the caller's value
//     is not hex of that type's length, given under hex_characters) or
//     differs (with the proof's hash under declared).
//   - When the attribution is not truestamp, the message says that the
//     match is against a document not attributed to Truestamp.
//
// The caller's value is lowercased, ASCII letters only, and never trimmed,
// so a value with a space or a newline is not hex of any length; the
// values hold it, lowercased, under expected_hash, when it is UTF-8. No
// member of claims but hash and hash_type is read. After a failure at the
// gate or at SHAPE the run has ended, its EXPECTED_HASH row is already skip
// prior_failure, and the step passes it through. A failure at POLICY,
// DERIVE, CHAIN, SOURCES or WINDOW does not stop the step.

package verifier

import (
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/truestamp/truestamp-cli/internal/specdata"
	"github.com/truestamp/truestamp-cli/internal/strictjson"
)

// expectedHashPart is the part of EXPECTED_HASH's row when a hash is given.
const expectedHashPart = "/subject/claims/hash"

// hashHexCharacters is the table of SPEC.md section 10.2, step 13: each
// hash_type and the number of hex characters of its hash.
var hashHexCharacters = map[string]int{
	"md5":      32,
	"sha1":     40,
	"sha224":   56,
	"sha3_224": 56,
	"sha256":   64,
	"sha3_256": 64,
	"blake2s":  64,
	"sha384":   96,
	"sha3_384": 96,
	"sha512":   128,
	"sha3_512": 128,
	"blake2b":  128,
}

// weakHashTypes are the hash_types whose match is
// expected_hash_weak_algorithm.
var weakHashTypes = map[string]bool{"md5": true, "sha1": true}

// ExpectedHash is step 13, EXPECTED_HASH (see the file's doc).
func ExpectedHash(run *Run) {
	if run.Ended {
		return
	}
	given := run.Config.ExpectedHash
	if given == nil {
		run.AddRow(NewRow("EXPECTED_HASH", specdata.ReasonExpectedHashNotChecked, nil,
			"No expected hash was given, so the claims' hash was not compared with one.", nil))
		return
	}
	expected := lowerASCII(*given)
	values := map[string]any{}
	if utf8.ValidString(expected) {
		values["expected_hash"] = expected
	}
	mismatch := func(reason, message string) {
		values["reason"] = reason
		run.AddRow(NewRow("EXPECTED_HASH", specdata.ReasonExpectedHashMismatch, Part(expectedHashPart), message, values))
	}
	hashType, hash, listed := listedHash(run)
	if !listed {
		mismatch("no_listed_hash", "The expected hash was not matched: this proof's claims hold no hash of a type the comparison lists.")
		return
	}
	length := hashHexCharacters[hashType]
	if len(expected) != length || strings.Trim(expected, "0123456789abcdef") != "" {
		values["hex_characters"] = strictjson.Number(float64(length))
		mismatch("not_hex_of_length", "The expected hash was not matched: it is not "+hashType+" hex, which is "+
			strconv.Itoa(length)+" hex characters.")
		return
	}
	if expected != hash {
		values["declared"] = hash
		mismatch("differs", "The expected hash differs from the claims' hash.")
		return
	}
	message := "The expected hash equals the claims' " + hashType + " hash."
	if run.Attribution != AttributionTruestamp {
		message = "The expected hash equals the claims' " + hashType + " hash, in a document not attributed to Truestamp."
	}
	if weakHashTypes[hashType] {
		run.AddRow(NewRow("EXPECTED_HASH", specdata.ReasonExpectedHashWeakAlgorithm, Part(expectedHashPart),
			message+" Collisions are practical for "+hashType+", so the match is weak evidence.", values))
		return
	}
	run.AddRow(PassRow("EXPECTED_HASH", Part(expectedHashPart), message, values))
}

// listedHash returns the claims' hash_type and hash when the proof is an
// item whose claims hold a hash that is a string, of a hash_type the table
// lists.
func listedHash(run *Run) (string, string, bool) {
	if run.Type != "item" {
		return "", "", false
	}
	claims, _ := valueAt(run.Proof, Pointer{Name("subject"), Name("claims")}).(map[string]any)
	hashType, _ := claims["hash_type"].(string)
	hash, isString := claims["hash"].(string)
	_, known := hashHexCharacters[hashType]
	return hashType, hash, known && isString
}

// lowerASCII lowercases the ASCII letters of s and nothing else.
func lowerASCII(s string) string {
	b := []byte(s)
	for i, c := range b {
		if 'A' <= c && c <= 'Z' {
			b[i] = c + ('a' - 'A')
		}
	}
	return string(b)
}
