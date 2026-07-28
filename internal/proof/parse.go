// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: MIT

package proof

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
	"strconv"
	"strings"

	"github.com/truestamp/truestamp-cli/internal/proof/ptype"
)

// Parse reads and parses a proof JSON file, preserving raw JSON for JCS.
func Parse(filename string) (*ProofBundle, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("reading file: %w", err)
	}
	return ParseBytes(data)
}

// ParseBytes parses a proof from raw bytes, dispatching to [ParseCBOR] for
// CBOR input and parsing JSON otherwise.
//
// Only the hard rejections E.6 enumerates abort here; everything else is a
// step failure the report has to surface. In particular a missing or wrong
// `v`, a missing or malformed `pk` / `sig`, and missing or wrong-sized
// hash / kid fields all parse cleanly so the pipeline can grade them.
// Every abort returns a [RejectionError] carrying its E.23 identifier.
func ParseBytes(data []byte) (*ProofBundle, error) {
	if IsCBORProof(data) {
		return ParseCBOR(data)
	}
	// Checking the opening byte before handing bytes to encoding/json keeps
	// an array / string / number / null input from surfacing the decoder's
	// anonymous target struct in user-visible output.
	if !looksLikeJSONObject(data) {
		return nil, rejectf(CodeNotAJSONObject, "input is not a JSON object")
	}

	// Every field is decoded as a raw message so that presence, null-ness,
	// and value type are all distinguishable — E.6's rules are written in
	// terms of all three, and Go's zero values collapse them.
	var raw struct {
		Version        json.RawMessage `json:"v"`
		T              json.RawMessage `json:"t"`
		Timestamp      json.RawMessage `json:"ts"`
		PublicKey      json.RawMessage `json:"pk"`
		Signature      json.RawMessage `json:"sig"`
		Subject        json.RawMessage `json:"s"`
		Block          json.RawMessage `json:"b"`
		Commitments    json.RawMessage `json:"cx"`
		InclusionProof json.RawMessage `json:"ip"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, rejectf(CodeNotAJSONObject, "parsing JSON: %s", err)
	}

	if !fieldCarried(raw.T) {
		return nil, rejectf(CodeMissingTypeCode, "missing required field: t")
	}
	// E.6 draws the line between "t is not an integer" (missing_type_code)
	// and "t is not a registered code" (invalid_subject_type_code). Decoding
	// straight into a uint16 collapsed the two: -1 and 70000 are integers,
	// and both were reported as non-integers.
	tValue, ok := jsonInteger(raw.T)
	if !ok {
		return nil, rejectf(CodeMissingTypeCode, "t is not an integer")
	}
	t, ok := subjectCodeFrom(tValue)
	if !ok {
		return nil, rejectf(CodeInvalidSubjectTypeCode, "invalid subject type code: %s", tValue)
	}

	// `v` is deliberately NOT a gate: E.6 exempts it, and E.8 reports a
	// wrong or absent version as a failing step. A non-integer `v` carries
	// through as 0 for the same reason.
	version := jsonIntOrZero(raw.Version)

	// pk / sig / ts are carried verbatim. E.6 authorises no rejection for
	// them; E.9 and E.16 report what they find.
	var timestamp, publicKey, signature string
	_ = json.Unmarshal(raw.Timestamp, &timestamp)
	_ = json.Unmarshal(raw.PublicKey, &publicKey)
	_ = json.Unmarshal(raw.Signature, &signature)

	if !fieldCarried(raw.Block) {
		return nil, rejectf(CodeMissingBlock, "missing required field: b")
	}
	// Only `b`'s map-ness is an E.6 rejection. Each member is read on its
	// own so a wrong-typed one — a numeric `id`, a boolean `kid` — carries
	// through as the zero value and reaches the step that grades it. A
	// whole-struct unmarshal turned all of those into hard rejections E.6
	// does not authorise, erasing reports the reference verifier produces.
	blockKeys, ok := jsonObject(raw.Block)
	if !ok {
		return nil, rejectf(CodeMissingBlock, "b is not a map")
	}
	block := Block{
		ID:                jsonString(blockKeys["id"]),
		PreviousBlockHash: jsonString(blockKeys["ph"]),
		MerkleRoot:        jsonString(blockKeys["mr"]),
		MetadataHash:      jsonString(blockKeys["mh"]),
		SigningKeyID:      jsonString(blockKeys["kid"]),
	}

	commits, err := parseCommitmentsJSON(raw.Commitments)
	if err != nil {
		return nil, err
	}

	bundle := &ProofBundle{
		Version:     version,
		T:           t,
		Timestamp:   timestamp,
		PublicKey:   publicKey,
		Signature:   signature,
		Block:       block,
		Commitments: commits,
	}

	// Subject + inclusion proof rules depend on type code. Block-like
	// subjects (t ∈ {10, 11} — plain block and beacon) share the same
	// wire shape: no `s`, no `ip`, `subject_hash == block_hash`.
	if ptype.IsBlockLikeSubject(t) {
		if err := checkBlockLikeShape(t, fieldCarried(raw.Subject), fieldCarried(raw.InclusionProof)); err != nil {
			return nil, err
		}
		return bundle, nil
	}

	// Non-block-like subjects (item, entropy_*): s and ip are required.
	if !fieldCarried(raw.Subject) {
		return nil, rejectf(CodeMissingSubject, "missing required field: s")
	}
	// As with `b`: map-ness is the only E.6 rejection, and `d` is carried
	// as the exact bytes the file holds so JCS canonicalizes what was
	// written rather than a Go round trip of it.
	subjectKeys, ok := jsonObject(raw.Subject)
	if !ok {
		return nil, rejectf(CodeMissingSubject, "s is not a map")
	}
	subject := Subject{
		ID:           jsonString(subjectKeys["id"]),
		Data:         subjectKeys["d"],
		MetadataHash: jsonString(subjectKeys["mh"]),
		SigningKeyID: jsonString(subjectKeys["kid"]),
	}
	// The `ip` gate is presence plus string-ness, not non-emptiness: an
	// empty `ip` parses and fails the E.12/E.13 inclusion-proof step.
	var inclusionProof string
	if !fieldCarried(raw.InclusionProof) || json.Unmarshal(raw.InclusionProof, &inclusionProof) != nil {
		return nil, rejectf(CodeMissingInclusionProof, "missing required field: ip")
	}

	bundle.Subject = &subject
	bundle.RawData = subject.Data
	bundle.InclusionProof = inclusionProof

	return bundle, nil
}

// parseCommitmentsJSON decodes and structurally validates `cx`. Presence of
// `ep` and of the chain-specific epoch-root key are the only per-entry field
// rejections E.6 authorises; everything else about an entry is graded later.
func parseCommitmentsJSON(raw json.RawMessage) ([]ExternalCommit, error) {
	if !fieldCarried(raw) {
		return nil, rejectf(CodeNoExternalCommitments, "missing required field: cx")
	}
	var entries []json.RawMessage
	if err := json.Unmarshal(raw, &entries); err != nil {
		return nil, rejectf(CodeNoExternalCommitments, "cx is not a list: %s", err)
	}
	if len(entries) == 0 {
		return nil, rejectf(CodeNoExternalCommitments, "cx must not be empty")
	}

	commits := make([]ExternalCommit, len(entries))
	for i, entry := range entries {
		keys, ok := jsonObject(entry)
		if !ok {
			return nil, rejectf(CodeInvalidExternalCommitmentEntry, "cx[%d] is not a map", i)
		}
		tValue, isInt := jsonInteger(keys["t"])
		if !isInt {
			return nil, rejectf(CodeInvalidExternalCommitmentEntry, "cx[%d]: t is not an integer", i)
		}
		code, valid := commitmentCodeFrom(tValue)
		if !valid {
			return nil, rejectf(CodeInvalidExternalCommitmentEntry, "cx[%d]: invalid commitment type code: %s", i, tValue)
		}

		// Every member other than `t` is read on its own: E.6's only
		// per-entry rejections are the two checked below, so a string `l` or
		// a numeric `net` must reach the report rather than abort the run.
		c := ExternalCommit{
			Type:            code,
			Network:         jsonString(keys["net"]),
			EpochProof:      jsonString(keys["ep"]),
			TransactionHash: jsonString(keys["tx"]),
			MemoHash:        jsonString(keys["memo"]),
			Ledger:          jsonIntOrZero(keys["l"]),
			Timestamp:       jsonString(keys["ts"]),
			OpReturn:        jsonString(keys["op"]),
			RawTxHex:        jsonString(keys["rtx"]),
			TxoutproofHex:   jsonString(keys["txp"]),
			BlockMerkleRoot: jsonString(keys["bmr"]),
			BlockHeight:     jsonIntOrZero(keys["h"]),
		}

		// `null` unmarshals into a string without error, so carriage has to
		// be tested before the type.
		var epString string
		epIsString := fieldCarried(keys["ep"]) && json.Unmarshal(keys["ep"], &epString) == nil
		if err := checkCommitmentKeys(i, c.Type, epIsString, fieldCarried(keys[rootKeyFor(c.Type)])); err != nil {
			return nil, err
		}
		commits[i] = c
	}
	return commits, nil
}

// jsonObject decodes a JSON object into its raw members. ok is false for
// every value that is not an object, which is the shape E.6 gates on for
// `b`, `s`, and each `cx` entry.
func jsonObject(raw json.RawMessage) (map[string]json.RawMessage, bool) {
	var keys map[string]json.RawMessage
	if err := json.Unmarshal(raw, &keys); err != nil || keys == nil {
		return nil, false
	}
	return keys, true
}

// jsonString reads a string member, yielding "" for one that is absent,
// null, or of another type. E.6 authorises no rejection for any of those,
// and the step that consumes the field reports what it finds.
func jsonString(raw json.RawMessage) string {
	var s string
	if json.Unmarshal(raw, &s) != nil {
		return ""
	}
	return s
}

// jsonInteger reports whether raw is a JSON integer literal, and returns it
// exactly. A literal carrying a fraction or an exponent is not an integer —
// JSON has one number type, so `20.0` and `2e1` are doubles that happen to
// be whole, which is how the reference verifier grades them too.
func jsonInteger(raw json.RawMessage) (intValue, bool) {
	// The literal has to be inspected before it is decoded: encoding/json
	// accepts a *quoted* number into a json.Number, so `"20"` — a JSON
	// string — would otherwise pass as the integer 20.
	text := strings.TrimSpace(string(raw))
	if text == "" || (text[0] != '-' && (text[0] < '0' || text[0] > '9')) {
		return intValue{}, false
	}
	var num json.Number
	if json.Unmarshal(raw, &num) != nil {
		return intValue{}, false
	}
	text = num.String()
	if strings.ContainsAny(text, ".eE") {
		return intValue{}, false
	}
	n, err := strconv.ParseInt(text, 10, 64)
	return intValue{N: n, Fits: err == nil, Text: text}, true
}

// jsonIntOrZero reads an integer member that E.6 does not gate (`v`, `l`,
// `h`), carrying anything else through as 0. A value too wide for an int on
// this platform is part of that "anything else": on a 32-bit build a bare
// conversion would truncate a ledger number or block height into a plausible
// but wrong one, so it is graded as unreadable instead.
func jsonIntOrZero(raw json.RawMessage) int {
	n, ok := jsonInteger(raw)
	if !ok || !n.Fits || n.N < math.MinInt || n.N > math.MaxInt {
		return 0
	}
	return int(n.N)
}

// intValue is an exactly parsed integer from either wire format. Text is its
// decimal rendering, which rejection details use so they can never name a
// truncated stand-in for the value the bundle carries; N holds the value
// where it fits an int64.
type intValue struct {
	N    int64
	Fits bool
	Text string
}

func (v intValue) String() string { return v.Text }

// subjectCodeFrom maps an exact integer onto a registered subject code.
// The range guard is the point of it: ptype.Code is a uint16, so converting
// first would fold 65546 onto 10 (block) and -1 onto 65535, and E.24 is
// explicit that an unregistered code MUST be rejected as unsupported rather
// than have a subject shape guessed for it. Hence the bound is checked on the
// int64 while it still holds the value the bundle actually carries.
func subjectCodeFrom(v intValue) (ptype.Code, bool) {
	if !v.Fits || v.N < 0 || v.N > math.MaxUint16 {
		return 0, false
	}
	c := ptype.Code(v.N)
	if !ptype.IsValidSubject(c) {
		return 0, false
	}
	return c, true
}

// commitmentCodeFrom is subjectCodeFrom for the `cx[].t` registry {40, 41}.
func commitmentCodeFrom(v intValue) (ptype.Code, bool) {
	if !v.Fits || v.N < 0 || v.N > math.MaxUint16 {
		return 0, false
	}
	c := ptype.Code(v.N)
	if !ptype.IsValidExternalCommitment(c) {
		return 0, false
	}
	return c, true
}

// looksLikeJSONObject reports whether the first non-whitespace byte of data
// opens a JSON object.
func looksLikeJSONObject(data []byte) bool {
	for _, c := range data {
		switch c {
		case ' ', '\t', '\n', '\r':
			continue
		case '{':
			return true
		default:
			return false
		}
	}
	return false
}

// fieldCarried implements E.6's presence rule for a JSON bundle field: a key
// that is missing, or whose value is null, counts as absent; any other value
// — including an empty string — counts as carried. The CBOR path applies the
// same rule through [cborFieldCarried] so the two serializations grade a
// bundle identically.
func fieldCarried(raw json.RawMessage) bool {
	return len(raw) > 0 && string(raw) != "null"
}

// checkBlockLikeShape is the single implementation of E.6's rule that a
// block-like bundle (t ∈ {10, 11}) MUST NOT carry `s` or `ip`. Both the JSON
// and the CBOR parser route through it after computing carriage with their
// encoding's presence rule.
func checkBlockLikeShape(t ptype.Code, carriesSubject, carriesInclusionProof bool) error {
	if !carriesSubject && !carriesInclusionProof {
		return nil
	}
	key := "s"
	if !carriesSubject {
		key = "ip"
	}
	return rejectf(CodeUnexpectedSubjectFieldsForBlockLike, "%s proof must not include %s", ptype.Name(t), key)
}

// rootKeyFor returns the key holding a commitment entry's epoch root.
func rootKeyFor(t ptype.Code) string {
	if t == ptype.CommitmentBitcoin {
		return "op"
	}
	return "memo"
}

// checkCommitmentKeys enforces the two per-entry hard rejections of E.6. The
// gates are deliberately asymmetric, matching the reference verifier: `ep`
// must be present *and* string-shaped, while the epoch-root key must merely
// be present. A present-but-malformed root key is not a rejection — it falls
// through to an E.15 epoch-proof step failure, where the mismatch is visible
// in the report instead of erasing it.
func checkCommitmentKeys(i int, t ptype.Code, epIsString, hasRootKey bool) error {
	if !epIsString {
		return rejectf(CodeInvalidExternalCommitmentEntry, "cx[%d] is missing ep", i)
	}
	if !hasRootKey {
		return rejectf(CodeInvalidExternalCommitmentEntry, "cx[%d] is missing %s", i, rootKeyFor(t))
	}
	return nil
}

// FileSizeFromData returns the byte length of proof data.
func FileSizeFromData(data []byte) int64 { return int64(len(data)) }

// FileSize returns the size of the file in bytes.
func FileSize(filename string) int64 {
	info, err := os.Stat(filename)
	if err != nil {
		return 0
	}
	return info.Size()
}
