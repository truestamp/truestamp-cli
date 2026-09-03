// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: MIT

package verify

import (
	"fmt"
	"regexp"
	"time"

	"github.com/truestamp/truestamp-cli/internal/jcs"
	"github.com/truestamp/truestamp-cli/internal/proof"
	"github.com/truestamp/truestamp-cli/internal/tscrypto"
)

// --- [E.10] Step 4: subject hash derivation ---
//
// Two prefix families, one shape:
//
//	items    claims 0x11, metadata 0x12, composite 0x13, id a 26-char ULID
//	entropy  entropy 0x21, metadata 0x22, composite 0x23, id a 36-char UUIDv7
//
//	data_hash     = SHA-256(prefix_data     || JCS(subject.claims | .entropy))
//	metadata_hash = SHA-256(prefix_metadata || JCS(subject.metadata))
//	subject_hash  = SHA-256(prefix_composite
//	                  || len32(id)            || id
//	                  || len32(data_hash)     || data_hash
//	                  || len32(metadata_hash) || metadata_hash
//	                  || len32(signing_key_id)|| signing_key_id)
//
// The metadata map itself rides in the bundle, so metadata_hash is derived
// here rather than read: an item's metadata is the witness map, and every
// witness in it is legible and checkable (E.17a).
func stepSubjectHash(r *Report, bundle *proof.Bundle) string {
	s := bundle.Subject
	dataKey := "entropy"
	dataPrefix, metaPrefix, compositePrefix := "0x21", "0x22", "0x23"
	if bundle.IsItem() {
		dataKey = "claims"
		dataPrefix, metaPrefix, compositePrefix = "0x11", "0x12", "0x13"
	}
	data := s.Data(bundle.Code)

	hexErr := firstHexError(hexField{name: "subject.signing_key_id", obj: s.Fields, key: "signing_key_id"})

	switch {
	case data == nil:
		// This result names no hash, so E.22's Subject Data exception does
		// not reach it and it stays under Cryptographic.
		r.fail(groupSubjectData, CatCryptographic,
			fmt.Sprintf("subject.%s is absent, so no data hash can be derived", dataKey))
		return ""
	case !s.Fields.IsString("id") || !s.Fields.IsString("signing_key_id"):
		// Names the composite subject hash and reports no completed
		// derivation: one of the two cases E.22 files under Data Integrity.
		r.fail(groupSubjectData, CatDataIntegrity,
			"missing fields for the composite subject hash: subject.id and subject.signing_key_id are both required")
		return ""
	case hexErr != "":
		// The E.4 sweep has already named it under Structure; this row is
		// the decoder refusing at the point of use.
		r.fail(groupSubjectData, CatDataIntegrity,
			"invalid_hex_encoding: cannot derive the composite subject hash: "+hexErr)
		return ""
	}

	canonicalData, oversized, err := jcs.Canonicalize(data)
	if err != nil {
		r.fail(groupSubjectData, CatCryptographic,
			fmt.Sprintf("cannot canonicalize subject.%s: %s", dataKey, err))
		return ""
	}
	canonicalMeta, _, err := jcs.Canonicalize(s.Metadata)
	if err != nil {
		r.fail(groupSubjectData, CatCryptographic,
			fmt.Sprintf("cannot canonicalize subject.metadata: %s", err))
		return ""
	}

	var dataHash, metaHash, subjectHash string
	if bundle.IsItem() {
		dataHash = tscrypto.ComputeClaimsHash(canonicalData)
		metaHash = tscrypto.ComputeItemMetadataHash(canonicalMeta)
		subjectHash, err = tscrypto.ComputeItemHash(s.ID, dataHash, metaHash, s.SigningKeyID)
	} else {
		dataHash = tscrypto.ComputeEntropyHash(canonicalData)
		metaHash = tscrypto.ComputeEntropyMetadataHash(canonicalMeta)
		subjectHash, err = tscrypto.ComputeObservationHash(s.ID, dataHash, metaHash, s.SigningKeyID)
	}
	if err != nil {
		r.fail(groupSubjectData, CatCryptographic,
			fmt.Sprintf("cannot derive the composite subject hash: %s", err))
		return ""
	}
	// 1 prefix byte + len32-framed id, data hash, metadata hash, key id.
	preimageSize := 1 + (4 + len(s.ID)) + (4 + 32) + (4 + 32) + (4 + len(s.SigningKeyID)/2)

	r.pass(groupSubjectData, CatCryptographic,
		fmt.Sprintf("data hash derived (%s) from JCS(subject.%s)", dataPrefix, dataKey))
	r.pass(groupSubjectData, CatCryptographic,
		fmt.Sprintf("metadata hash derived (%s) from JCS(subject.metadata), which the bundle carries", metaPrefix))
	r.pass(groupSubjectData, CatCryptographic,
		fmt.Sprintf("composite fingerprint derived (%s) from %d-byte preimage", compositePrefix, preimageSize))

	reportNumberPortability(r, oversized, "subject."+dataKey)
	if bundle.IsItem() {
		softChecks(r, bundle)
	}
	return subjectHash
}

// reportNumberPortability emits E.4's non-portability warn. RFC 8785
// defines every JSON number by parsing it into an IEEE-754 double, so an
// integer beyond +/- 2^53 is read back as a DIFFERENT integer by a
// JavaScript or Python verifier. This verifier and Truestamp's producer
// both emit integers exactly as parsed, so the digest reproduces between
// them; the hazard is interoperability with a strictly conforming
// implementation, and E.4 says such a bundle MUST be reported as such
// rather than silently rounded. The row stays under Cryptographic: E.22's
// Subject Data exception is a closed list of two results, and this is not
// one of them.
func reportNumberPortability(r *Report, oversized []string, where string) {
	if len(oversized) == 0 {
		return
	}
	r.warn(groupSubjectData, CatCryptographic, fmt.Sprintf(
		"not portably verifiable: %s carries %d integer(s) outside the exactly representable range (e.g. %s); a verifier that parses numbers into IEEE-754 doubles per RFC 8785 will derive a different data hash",
		where, len(oversized), oversized[0]))
}

// --- [E.11] Step 4a: item-level soft checks ---
//
// These all warn and never fail. They describe the CLAIMS a submitter
// made, and a submitter can claim anything: a self-reported timestamp is
// not evidence. E.22's Subject Data exception files the hash-shape result
// under Data Integrity and the claim-timing warnings under Timing.

// hashTypeLengths maps the twelve registered algorithms to their digest
// sizes in hex characters, used only for the soft format check.
var hashTypeLengths = map[string]int{
	"md5": 32, "sha1": 40, "sha224": 56, "sha256": 64, "sha384": 96, "sha512": 128,
	"sha3_224": 56, "sha3_256": 64, "sha3_384": 96, "sha3_512": 128, "blake2s": 64, "blake2b": 128,
}

var lowercaseHex = regexp.MustCompile(`^[0-9a-f]+$`)

func softChecks(r *Report, bundle *proof.Bundle) {
	claims, ok := claimsObject(bundle)
	if !ok {
		return
	}
	checkHashFormat(r, claims)
	checkClaimedTimestamp(r, claims, bundle.Subject.ID)
}

func checkHashFormat(r *Report, claims proof.Object) {
	hash, hashOK := claims.String("hash")
	hashType, typeOK := claims.String("hash_type")
	if !hashOK || !typeOK {
		return
	}
	expected, known := hashTypeLengths[hashType]
	if !known {
		r.warn(groupSubjectData, CatDataIntegrity, fmt.Sprintf("unrecognized hash_type %s", hashType))
		return
	}
	if len(hash) == expected && lowercaseHex.MatchString(hash) {
		r.info(groupSubjectData, CatDataIntegrity, fmt.Sprintf("claimed %s hash is well formed", hashType))
	} else {
		r.warn(groupSubjectData, CatDataIntegrity,
			fmt.Sprintf("claimed hash is not %d lowercase hex characters for %s", expected, hashType))
	}
}

func checkClaimedTimestamp(r *Report, claims proof.Object, subjectID string) {
	claimed, ok := claims.String("timestamp")
	if !ok {
		return
	}
	claimedMs, ok := isoMs(claimed)
	if !ok {
		return
	}
	submittedMs, ok := ulidMs(subjectID)
	if !ok {
		return
	}
	const sevenDays = 7 * 24 * 60 * 60 * 1000
	switch {
	case claimedMs > submittedMs:
		r.warn(groupSubjectData, CatTiming,
			"claimed timestamp is later than the submission time embedded in subject.id")
	case submittedMs-claimedMs > sevenDays:
		r.warn(groupSubjectData, CatTiming,
			"claimed timestamp is more than 7 days before submission")
	}
}

// --- identifier and timestamp helpers (E.20) ---

func ulidMs(id string) (int64, bool) {
	t, err := tscrypto.ExtractULIDTimestamp(id)
	if err != nil {
		return 0, false
	}
	return t.UnixMilli(), true
}

func uuidv7Ms(id string) (int64, bool) {
	t, err := tscrypto.ExtractUUIDv7Timestamp(id)
	if err != nil {
		return 0, false
	}
	return t.UnixMilli(), true
}

// isoMs parses an ISO 8601 timestamp into Unix milliseconds.
func isoMs(value string) (int64, bool) {
	t, err := time.Parse(time.RFC3339Nano, value)
	if err != nil {
		return 0, false
	}
	return t.UnixMilli(), true
}

// formatMs renders an instant at millisecond precision, the way the
// reference verifier prints every time it derives.
func formatMs(ms int64) string {
	return time.UnixMilli(ms).UTC().Format("2006-01-02T15:04:05.000Z")
}

// formatSeconds renders an instant at whole-second precision, the way the
// server records its `temporal` values.
func formatSeconds(ms int64) string {
	return time.UnixMilli(ms).UTC().Format(time.RFC3339)
}
