// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: MIT

package verify

import (
	"fmt"
	"strings"

	"github.com/truestamp/truestamp-cli/internal/proof"
	"github.com/truestamp/truestamp-cli/internal/tscrypto"
)

// --- [E.7] Step 0: expected-hash comparison ---

// stepExpectedHash is the only step that connects the proof to a file on
// the caller's disk. It runs only for an item subject; for every other type
// a supplied hash is reported as a visible skip rather than silently
// discarded. When no hash is supplied the step warns and never fails, and
// the report keeps "was a hash provided" separate from "did it match".
func stepExpectedHash(r *Report, bundle *proof.Bundle, opts Options) {
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
		// The item timestamped its claims content itself: there is no
		// digest for the argument to mismatch, and the proof's soundness
		// does not depend on it.
		r.warn(groupHashComparison, CatDataIntegrity,
			"an expected hash was supplied but this proof commits to no file hash")
	default:
		// The expected hash is an operator's typed argument, so the fold is
		// on both operands; the wire value itself is graded by the E.11
		// shape check, never normalized.
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
	claims, ok := claimsObject(bundle)
	if !ok {
		return ""
	}
	return claims.Str("hash")
}

func claimsObject(bundle *proof.Bundle) (proof.Object, bool) {
	if bundle.Subject == nil {
		return nil, false
	}
	return bundle.Subject.Fields.Object("claims")
}

// --- [E.8] Step 1: version ---

func stepVersion(r *Report, bundle *proof.Bundle) {
	r.check(groupStructure, CatStructural, bundle.Version == 1,
		"proof format version 1",
		fmt.Sprintf("unsupported proof format version %s (expected 1)", literal(bundle.VersionLiteral)))
}

// --- [E.4] Step 2: the hex-encoding sweep ---

// hexField is one wire field the lowercase-hex rule covers, with the value
// as carried so an offender can be named precisely.
type hexField struct {
	name  string
	obj   proof.Object
	key   string
	value string // for values not read from an Object
	set   bool
}

// unhexError returns "" for a conforming value, or the reason the field
// refuses to decode as lowercase hex, in the reference verifier's words.
func (f hexField) unhexError() (present bool, reason string) {
	var raw string
	if f.obj != nil {
		if !f.obj.Has(f.key) {
			return false, ""
		}
		s, isString := f.obj.String(f.key)
		if !isString {
			return true, "not a string (E.4): " + f.obj.Literal(f.key)
		}
		raw = s
	} else {
		if !f.set {
			return false, ""
		}
		raw = f.value
	}
	if err := tscrypto.ValidateLowercaseHex(raw); err != nil {
		return true, err.Error()
	}
	return true, ""
}

// stepEncoding sweeps the closed list of hex fields before any hash is
// derived (E.4), so an encoding defect is reported as one rather than
// surfacing downstream as a root mismatch. It names every offender in wire
// order in one row, and emits nothing at all on a conforming bundle: E.25
// forbids adding a `pass` row that Appendix D.4 does not carry.
//
// The three hex fields of the head block carried as the `block` witness are
// deliberately not swept: they are graded where that block's hash is
// derived, in E.17a, so one defect does not produce two failures.
func stepEncoding(r *Report, bundle *proof.Bundle) {
	var fields []hexField
	if s := bundle.Subject; s != nil {
		fields = append(fields, hexField{name: "subject.signing_key_id", obj: s.Fields, key: "signing_key_id"})
		committed := s.CommittedWitnesses()
		for _, name := range committed.Keys() {
			fields = append(fields, hexField{name: "subject.metadata.witnesses." + name, obj: committed, key: name})
		}
	}
	fields = append(fields, blockHexFields("block", bundle.Block)...)
	for i, entry := range bundle.BlockPath {
		fields = append(fields, blockHexFields(fmt.Sprintf("block_path[%d]", i), entry)...)
	}
	fields = append(fields, commitmentHexFields("commitments", bundle.Commitments)...)
	if e := bundle.SigningKeyEvent; e != nil && e.IsMap {
		fields = append(fields, blockHexFields("signing_key_event.block", e.Block)...)
		fields = append(fields, commitmentHexFields("signing_key_event.commitments", e.Commitments)...)
	}

	var offenders []string
	for _, f := range fields {
		if present, reason := f.unhexError(); present && reason != "" {
			offenders = append(offenders, f.name+": "+reason)
		}
	}
	if len(offenders) == 0 {
		return
	}
	r.fail(groupStructure, CatStructural,
		"invalid_hex_encoding: hash fields do not carry E.4's required lowercase-hex encoding - "+
			strings.Join(offenders, "; "))
}

func blockHexFields(prefix string, b proof.BlockMap) []hexField {
	if !b.IsMap() {
		return nil
	}
	return []hexField{
		{name: prefix + ".previous_block_hash", obj: b.Fields, key: "previous_block_hash"},
		{name: prefix + ".merkle_root", obj: b.Fields, key: "merkle_root"},
		{name: prefix + ".signing_key_id", obj: b.Fields, key: "signing_key_id"},
	}
}

// commitmentHexFields lists the three hex fields of each entry.
// `raw_transaction` and `txoutproof` are absent on purpose: E.3 files them
// as text carrying EITHER base64url or hex, and since the hex alphabet is a
// subset of the base64url one, a lowercase rule there is not strict, it is
// undefined.
func commitmentHexFields(prefix string, entries []proof.Commitment) []hexField {
	var out []hexField
	for i, c := range entries {
		p := fmt.Sprintf("%s[%d]", prefix, i)
		out = append(out,
			hexField{name: p + ".epoch_merkle_root", obj: c.Fields, key: "epoch_merkle_root"},
			hexField{name: p + ".transaction_hash", obj: c.Fields, key: "transaction_hash"},
			hexField{name: p + ".block_merkle_root", obj: c.Fields, key: "block_merkle_root"},
		)
	}
	return out
}

// firstHexError returns the first field among fields that is not lowercase
// hex, as "name: reason", or "" when every value decodes. An absent field
// is graded as "not a string" here, matching the reference verifier, whose
// callers test presence before consulting it.
func firstHexError(fields ...hexField) string {
	for _, f := range fields {
		present, reason := f.unhexError()
		if !present {
			return f.name + ": not a string (E.4): nil"
		}
		if reason != "" {
			return f.name + ": " + reason
		}
	}
	return ""
}

// --- [E.9] Step 3: public key and key id ---

// stepPublicKey decodes `public_key` and derives
// key_id = SHA-256(0x51 || public_key)[0..3]. That derived value fills the
// kid slot of the signature payload and is what E.17 checks against the
// keyring; the subject-hash and block-hash preimages use the DIFFERENT,
// stored `signing_key_id` values, which diverge under key rotation.
func stepPublicKey(r *Report, bundle *proof.Bundle) ([]byte, string) {
	pub, err := tscrypto.DecodePublicKey(bundle.PublicKey)
	if err != nil {
		r.fail(groupSigningKey, CatCryptographic, "public key missing, or does not base64-decode to 32 bytes")
		return nil, ""
	}
	keyID := tscrypto.ComputeKeyID(pub)
	r.pass(groupSigningKey, CatCryptographic, fmt.Sprintf("public key valid, key_id %s", keyID))
	return pub, keyID
}
