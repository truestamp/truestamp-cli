// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: MIT

// Package proof parses, downloads, and generates Truestamp proof bundles in
// both JSON and CBOR wire formats. Proofs are the self-contained artefacts
// consumers receive from the API; this package handles only serialization
// and I/O — cryptographic verification lives in internal/verify.
package proof

import (
	"encoding/base64"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math"
	"math/big"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/fxamacker/cbor/v2"
	"github.com/truestamp/truestamp-cli/internal/proof/ptype"
)

// HasCBORTag reports whether data begins with the RFC 8949 self-describing
// tag 55799 (0xd9 0xd9 0xf7).
func HasCBORTag(data []byte) bool {
	return len(data) >= 3 && data[0] == 0xd9 && data[1] == 0xd9 && data[2] == 0xf7
}

// IsCBORProof reports whether data is a CBOR proof bundle. E.3 requires a
// verifier to accept CBOR both wrapped in the self-describing tag 55799 and
// as a bare map, so a bare CBOR map — major type 5, first byte 0xa0-0xbf,
// definite or indefinite length — counts too. Those bytes are UTF-8
// continuation bytes and can never open a valid JSON document, so widening
// the check cannot steal input from the JSON path.
func IsCBORProof(data []byte) bool {
	if HasCBORTag(data) {
		return true
	}
	return len(data) > 0 && data[0] >= 0xa0 && data[0] <= 0xbf
}

// cborDecMode decodes with duplicate map keys enforced. RFC 8949 section 5.6
// makes a map with duplicate keys invalid, and accepting one is a split-view
// forgery: fxamacker resolves duplicates last-wins, so a single signed blob
// reads as the genuine `s.d` here and as attacker-chosen text under a
// first-wins decoder, while the equivalent JSON document is refused by the
// JCS canonicalizer. Enforcement also keeps the raw-byte scanner and the
// decoded tree in sync — with duplicates allowed the two disagree about
// which `d` is the real one.
//
// Invalid UTF-8 is still accepted at this layer; the `s.d` scanner rejects
// it where it would reach a hash preimage (E.3), while leaving a stray byte
// in a non-hashed text field to the steps that grade it.
//
// DecMode() can only fail for invalid option combinations; this one is
// statically valid, so we discard the error.
var cborDecMode, _ = cbor.DecOptions{
	UTF8:      cbor.UTF8DecodeInvalid,
	DupMapKey: cbor.DupMapKeyEnforcedAPF,
}.DecMode()

// ParseCBOR decodes a CBOR proof bundle into a ProofBundle. The output is
// structurally identical to what ParseBytes produces from JSON, and the two
// paths agree on the verdict for the same logical bundle: E.6's rejections
// are enforced from the same helpers, and every CBOR-only value-space class
// (E.3) is refused here rather than coerced into a JSON counterpart.
//
// The two do not always agree on the *kind* of refusal. A duplicate `s.d`
// key is a hard rejection on this path, because RFC 8949 section 5.6 makes
// the document invalid CBOR, while the JSON path reaches it as a failing
// JCS step. Both refuse the bundle; only the stage differs.
func ParseCBOR(data []byte) (*ProofBundle, error) {
	var raw map[string]any
	if err := cborDecMode.Unmarshal(data, &raw); err != nil {
		return nil, rejectf(CodeNotAJSONObject, "CBOR decode: %s", err)
	}
	if raw == nil {
		return nil, rejectf(CodeNotAJSONObject, "input is not a CBOR map")
	}

	tRaw, hasT := raw["t"]
	if !hasT || tRaw == nil {
		return nil, rejectf(CodeMissingTypeCode, "missing required field: t")
	}
	// E.6 separates "t is not an integer" from "t is not a registered code",
	// and E.24 forbids guessing a subject shape for an unregistered one. A
	// CBOR float or string is the first case; any integer that is not one of
	// the six registered codes is the second, and the detail names the value
	// the bundle actually carries rather than a truncated stand-in.
	tValue, ok := cborInteger(tRaw)
	if !ok {
		return nil, rejectf(CodeMissingTypeCode, "t is not an integer")
	}
	t, ok := subjectCodeFrom(tValue)
	if !ok {
		return nil, rejectf(CodeInvalidSubjectTypeCode, "invalid subject type code: %s", tValue)
	}

	// `v` is not a gate (E.6 exempts it); an absent or non-integer version
	// carries through as 0 and E.8 reports it as a failing step.
	version := cborIntOrZero(raw["v"])

	block, err := decodeBlockCBOR(raw["b"])
	if err != nil {
		return nil, rejectf(CodeMissingBlock, "b: %s", err)
	}

	commits, err := decodeCommitsCBOR(raw["cx"])
	if err != nil {
		return nil, err
	}
	if len(commits) == 0 {
		return nil, rejectf(CodeNoExternalCommitments, "cx must not be empty")
	}

	timestamp, _ := raw["ts"].(string)

	bundle := &ProofBundle{
		Version:     version,
		T:           t,
		Timestamp:   timestamp,
		PublicKey:   cborBytesToBase64(raw, "pk"),
		Signature:   cborBytesToBase64(raw, "sig"),
		Block:       block,
		Commitments: commits,
	}

	// Block-like subjects (t ∈ {10, 11}) have no `s` and no `ip`.
	if ptype.IsBlockLikeSubject(t) {
		if err := checkBlockLikeShape(t, cborFieldCarried(raw, "s"), cborFieldCarried(raw, "ip")); err != nil {
			return nil, err
		}
		return bundle, nil
	}

	if !cborFieldCarried(raw, "s") {
		return nil, rejectf(CodeMissingSubject, "missing required field: s")
	}
	// The value-space check in decodeSubjectCBOR needs the encoding of `s.d`
	// exactly as it sits on the wire, not the decoded tree: the decoder
	// erases the distinctions E.3 cares about (CBOR `undefined` and `null`
	// both become nil, and a 55799 tag is silently unwrapped so a tagged
	// value collides with the bare one). Unmarshaling into a
	// cbor.RawMessage is not enough — it strips a leading 55799 tag too,
	// which is precisely the wrapper E.3 permits only around the whole
	// bundle — so the bytes are located by walking the input directly.
	rawSD, located := rawSubjectData(data)

	subject, rawData, err := decodeSubjectCBOR(raw["s"], rawSD, located)
	if err != nil {
		return nil, err
	}

	// Presence plus string-ness, not non-emptiness: an empty `ip` parses
	// and fails the E.12/E.13 inclusion-proof step.
	inclusionProof, ok := cborStringField(raw, "ip")
	if !ok {
		return nil, rejectf(CodeMissingInclusionProof, "missing required field: ip")
	}

	bundle.Subject = &subject
	bundle.InclusionProof = inclusionProof
	bundle.RawData = rawData
	return bundle, nil
}

func decodeSubjectCBOR(v any, rawSD []byte, located bool) (Subject, json.RawMessage, error) {
	m, ok := v.(map[any]any)
	if !ok {
		m2, ok2 := v.(map[string]any)
		if !ok2 {
			return Subject{}, nil, rejectf(CodeMissingSubject, "s is not a map, got %T", v)
		}
		m = toAnyKeyMap(m2)
	}

	// Fail closed. The decoder resolves encodings the byte walker declines
	// to follow — an indefinite-length key, a second 55799 wrapper around
	// the whole bundle — and every one of those is a route to hashing `s.d`
	// bytes nothing validated, which is exactly the tag-unwrapping hole this
	// walker exists to close. If the decoded subject carries `d` and the
	// wire form cannot be walked to it, the bundle is refused rather than
	// graded on unvalidated bytes.
	if _, carriesData := m["d"]; carriesData && !located {
		return Subject{}, nil, rejectf(CodeInvalidSubjectData,
			"s.d: the bundle encoding cannot be walked to s.d, so the bytes that feed the data hash cannot be validated")
	}
	if len(rawSD) > 0 {
		if _, err := validateSubjectDataCBOR(rawSD, "s.d"); err != nil {
			return Subject{}, nil, err
		}
	}

	id, _ := m["id"].(string)
	dataJSON, err := anyToJSON(m["d"])
	if err != nil {
		return Subject{}, nil, err
	}

	subject := Subject{
		ID:           id,
		Data:         dataJSON,
		MetadataHash: bytesFieldToHex(m, "mh"),
		SigningKeyID: bytesFieldToHex(m, "kid"),
	}

	return subject, dataJSON, nil
}

func decodeBlockCBOR(v any) (Block, error) {
	m := normalizeMap(v)
	if m == nil {
		return Block{}, fmt.Errorf("expected map, got %T", v)
	}

	return Block{
		ID:                getString(m, "id"),
		PreviousBlockHash: bytesFieldToHex(m, "ph"),
		MerkleRoot:        bytesFieldToHex(m, "mr"),
		MetadataHash:      bytesFieldToHex(m, "mh"),
		SigningKeyID:      bytesFieldToHex(m, "kid"),
	}, nil
}

func decodeCommitsCBOR(v any) ([]ExternalCommit, error) {
	if v == nil {
		return nil, nil
	}
	items, ok := v.([]any)
	if !ok {
		return nil, rejectf(CodeNoExternalCommitments, "cx is not a list, got %T", v)
	}

	commits := make([]ExternalCommit, len(items))
	for i, item := range items {
		m := normalizeMap(item)
		if m == nil {
			return nil, rejectf(CodeInvalidExternalCommitmentEntry, "cx[%d] is not a map", i)
		}

		tValue, isInt := cborInteger(m["t"])
		if !isInt {
			return nil, rejectf(CodeInvalidExternalCommitmentEntry, "cx[%d]: t is not an integer", i)
		}
		code, valid := commitmentCodeFrom(tValue)
		if !valid {
			return nil, rejectf(CodeInvalidExternalCommitmentEntry, "cx[%d]: invalid commitment type code: %s", i, tValue)
		}
		ec := ExternalCommit{
			Type:    code,
			Network: getString(m, "net"),
		}

		// Epoch proof: E.3 lists it as a text string, but the backend emits
		// a byte string, so both are accepted.
		var epIsString bool
		switch val := m["ep"].(type) {
		case []byte:
			ec.EpochProof = base64.RawURLEncoding.EncodeToString(val)
			epIsString = true
		case string:
			ec.EpochProof = val
			epIsString = true
		}
		rootRaw, hasRoot := m[rootKeyFor(ec.Type)]
		if err := checkCommitmentKeys(i, ec.Type, epIsString, hasRoot && rootRaw != nil); err != nil {
			return nil, err
		}

		// Stellar fields
		ec.TransactionHash = bytesFieldToHex(m, "tx")
		ec.MemoHash = bytesFieldToHex(m, "memo")
		ec.Ledger = cborIntOrZero(m["l"])
		ec.Timestamp = getString(m, "ts")

		// Bitcoin fields
		ec.OpReturn = bytesFieldToHex(m, "op")
		ec.RawTxHex = textFieldToString(m, "rtx")
		ec.TxoutproofHex = textFieldToString(m, "txp")
		ec.BlockMerkleRoot = bytesFieldToHex(m, "bmr")
		ec.BlockHeight = cborIntOrZero(m["h"])

		commits[i] = ec
	}

	return commits, nil
}

// ── CBOR `s.d` value-space validation (E.3) ─────────────────────────

// validateSubjectDataCBOR walks the raw CBOR encoding of `s.d` and rejects
// every value that cannot round-trip to JSON: byte strings, tags (the 55799
// self-describe tag included — E.3 permits it only around the whole bundle),
// `undefined`, simple values other than true/false/null, non-finite floats,
// non-text map keys, and text that is not valid UTF-8. Without this the 0x11
// / 0x21 preimage is silently coerced — encoding/json substitutes U+FFFD for
// every invalid byte — and two distinct wire bundles verify under one
// signature.
//
// Indefinite-length items are refused for the same reason even though they
// do round-trip: RFC 8949 section 4.2 core-deterministic encoding is what
// every Truestamp bundle is written with, and an indefinite-length spelling
// canonicalizes to the identical JSON, so accepting it would admit a second
// wire encoding of one signed `s.d`.
//
// Duplicate map keys are not checked here: cborDecMode refuses them for the
// whole document (RFC 8949 section 5.6), which is also what keeps these raw
// bytes and the decoded tree describing the same map.
//
// Returns the number of bytes the single data item at b[0:] occupies.
func validateSubjectDataCBOR(b []byte, path string) (int, error) {
	mt, arg, headLen, indefinite, err := cborHead(b)
	if err != nil {
		return 0, rejectSubjectData(path, err.Error())
	}
	if indefinite {
		return 0, rejectSubjectData(path, "indefinite-length CBOR item")
	}

	switch mt {
	case 0, 1: // unsigned / negative integer
		return headLen, nil

	case 2: // byte string
		return 0, rejectSubjectData(path, "CBOR byte string")

	case 3: // text string
		return definiteTextEnd(b, headLen, arg, path)

	case 4: // array
		n := headLen
		for i := uint64(0); i < arg; i++ {
			size, err := validateSubjectDataCBOR(b[n:], fmt.Sprintf("%s[%d]", path, i))
			if err != nil {
				return 0, err
			}
			n += size
		}
		return n, nil

	case 5: // map
		n := headLen
		for i := uint64(0); i < arg; i++ {
			key, keyLen, err := cborMapKey(b[n:], path)
			if err != nil {
				return 0, err
			}
			n += keyLen
			size, err := validateSubjectDataCBOR(b[n:], path+"."+key)
			if err != nil {
				return 0, err
			}
			n += size
		}
		return n, nil

	case 6: // tag
		return 0, rejectSubjectData(path, fmt.Sprintf("CBOR tag %d", arg))

	default: // major type 7: simple values and floats
		switch b[0] & 0x1f {
		case 20, 21, 22: // false, true, null
			return headLen, nil
		case 23:
			return 0, rejectSubjectData(path, "CBOR undefined")
		case 25, 26, 27:
			f, err := cborFloat(b, headLen)
			if err != nil {
				return 0, rejectSubjectData(path, err.Error())
			}
			if math.IsNaN(f) || math.IsInf(f, 0) {
				return 0, rejectSubjectData(path, "CBOR non-finite float")
			}
			return headLen, nil
		default:
			return 0, rejectSubjectData(path, fmt.Sprintf("CBOR simple value %d", arg))
		}
	}
}

// rawSubjectData returns the bytes of `s.d` exactly as they appear in data,
// or ok=false when the bundle carries no locatable `s.d`. Walking the input
// is the only way to see a tag wrapping `s.d`: both the generic decode and
// an unmarshal into cbor.RawMessage strip a leading 55799 tag, so a tagged
// subject-data map would arrive at the value-space scanner already unwrapped
// and every `tag55799(...)` nesting depth would verify under one signature.
//
// A malformed encoding returns ok=false rather than an error: this runs
// after the decoder has already accepted the document, so anything it cannot
// follow is a shape the decoder resolved some other way, and the value-space
// check is not the place to re-litigate well-formedness.
func rawSubjectData(data []byte) ([]byte, bool) {
	b := data
	if HasCBORTag(b) {
		b = b[3:]
	}
	subject, ok := cborMapValue(b, "s")
	if !ok {
		return nil, false
	}
	return cborMapValue(subject, "d")
}

// cborMapValue returns the raw encoding of key's value in the CBOR map at
// b[0:], following both the definite and the indefinite-length map form
// (E.3 requires a bare map to be accepted and says nothing about its
// length encoding).
//
// A key it cannot compare — a non-text key, or a text key split into
// indefinite-length chunks — makes it give up rather than guess; the caller
// treats a give-up on a bundle that does carry `s.d` as a rejection.
func cborMapValue(b []byte, key string) ([]byte, bool) {
	mt, arg, headLen, indefinite, err := cborHead(b)
	if err != nil || mt != 5 {
		return nil, false
	}
	n := headLen
	for i := uint64(0); indefinite || i < arg; i++ {
		if indefinite {
			if n >= len(b) {
				return nil, false
			}
			if b[n] == 0xff {
				return nil, false
			}
		}
		keyMT, keyArg, keyHead, keyIndef, err := cborHead(b[n:])
		if err != nil || keyMT != 3 || keyIndef {
			return nil, false
		}
		keyEnd, err := definiteTextEnd(b[n:], keyHead, keyArg, "")
		if err != nil {
			return nil, false
		}
		keyEnd += n
		size, err := cborItemSize(b[keyEnd:])
		if err != nil {
			return nil, false
		}
		if string(b[n+keyHead:keyEnd]) == key {
			return b[keyEnd : keyEnd+size], true
		}
		n = keyEnd + size
	}
	return nil, false
}

// cborItemSize returns the byte length of the single CBOR data item at
// b[0:]. Unlike validateSubjectDataCBOR it grades nothing — it accepts every
// well-formed item, including the tags and byte strings a bundle legitimately
// carries outside `s.d` — because its only job is to step over a value while
// looking for a map key.
func cborItemSize(b []byte) (int, error) {
	mt, arg, headLen, indefinite, err := cborHead(b)
	if err != nil {
		return 0, err
	}
	switch mt {
	case 2, 3:
		if indefinite {
			return indefiniteChunksEnd(b, headLen, mt)
		}
		if arg > uint64(len(b)-headLen) {
			return 0, fmt.Errorf("truncated CBOR string")
		}
		return headLen + int(arg), nil
	case 4, 5:
		// A map head counts pairs, so the item count is twice the argument.
		// The guard keeps that doubling from wrapping on a declared length
		// no input could actually carry.
		items := arg
		if mt == 5 {
			if arg > math.MaxUint64/2 {
				return 0, fmt.Errorf("CBOR map length out of range")
			}
			items = arg * 2
		}
		n := headLen
		for i := uint64(0); indefinite || i < items; i++ {
			if indefinite {
				if n >= len(b) {
					return 0, fmt.Errorf("truncated CBOR item")
				}
				if b[n] == 0xff {
					return n + 1, nil
				}
			}
			size, err := cborItemSize(b[n:])
			if err != nil {
				return 0, err
			}
			n += size
		}
		return n, nil
	case 6:
		size, err := cborItemSize(b[headLen:])
		if err != nil {
			return 0, err
		}
		return headLen + size, nil
	default:
		if indefinite {
			return 0, fmt.Errorf("unexpected CBOR break")
		}
		return headLen, nil
	}
}

// indefiniteChunksEnd walks the definite-length chunks of an
// indefinite-length string of major type want through to its break.
func indefiniteChunksEnd(b []byte, headLen int, want byte) (int, error) {
	n := headLen
	for {
		if n >= len(b) {
			return 0, fmt.Errorf("truncated indefinite-length CBOR item")
		}
		if b[n] == 0xff {
			return n + 1, nil
		}
		mt, arg, chunkHead, indefinite, err := cborHead(b[n:])
		if err != nil || mt != want || indefinite {
			return 0, fmt.Errorf("malformed indefinite-length CBOR string")
		}
		if arg > uint64(len(b)-n-chunkHead) {
			return 0, fmt.Errorf("truncated CBOR string")
		}
		n += chunkHead + int(arg)
	}
}

// cborHead decodes the head of a CBOR data item into its major type,
// argument, head length in bytes, and whether it is indefinite-length.
func cborHead(b []byte) (mt byte, arg uint64, headLen int, indefinite bool, err error) {
	if len(b) == 0 {
		return 0, 0, 0, false, fmt.Errorf("truncated CBOR item")
	}
	mt = b[0] >> 5
	ai := b[0] & 0x1f
	switch {
	case ai < 24:
		return mt, uint64(ai), 1, false, nil
	case ai == 24:
		if len(b) < 2 {
			return 0, 0, 0, false, fmt.Errorf("truncated CBOR item")
		}
		return mt, uint64(b[1]), 2, false, nil
	case ai == 25:
		if len(b) < 3 {
			return 0, 0, 0, false, fmt.Errorf("truncated CBOR item")
		}
		return mt, uint64(binary.BigEndian.Uint16(b[1:3])), 3, false, nil
	case ai == 26:
		if len(b) < 5 {
			return 0, 0, 0, false, fmt.Errorf("truncated CBOR item")
		}
		return mt, uint64(binary.BigEndian.Uint32(b[1:5])), 5, false, nil
	case ai == 27:
		if len(b) < 9 {
			return 0, 0, 0, false, fmt.Errorf("truncated CBOR item")
		}
		return mt, binary.BigEndian.Uint64(b[1:9]), 9, false, nil
	case ai == 31:
		return mt, 0, 1, true, nil
	default:
		return 0, 0, 0, false, fmt.Errorf("reserved CBOR additional information %d", ai)
	}
}

// cborFloat reads the IEEE-754 value of a major-type-7 float item.
func cborFloat(b []byte, headLen int) (float64, error) {
	if len(b) < headLen {
		return 0, fmt.Errorf("truncated CBOR float")
	}
	switch headLen {
	case 3: // half precision: only the exponent matters for finiteness
		bits := binary.BigEndian.Uint16(b[1:3])
		if bits&0x7c00 == 0x7c00 {
			return math.Inf(1), nil // NaN and ±Inf are both rejected
		}
		return 0, nil
	case 5:
		return float64(math.Float32frombits(binary.BigEndian.Uint32(b[1:5]))), nil
	default:
		return math.Float64frombits(binary.BigEndian.Uint64(b[1:9])), nil
	}
}

// definiteTextEnd returns the total size of a definite-length text-string
// item, rejecting one whose content is not valid UTF-8. JSON strings are
// UTF-8 by definition (RFC 8259), and encoding/json silently substitutes
// U+FFFD for each invalid byte, so an unchecked text string reaches the
// 0x11 / 0x21 preimage as different bytes than the wire carried — and any
// two invalid sequences of equal length collide onto one preimage.
func definiteTextEnd(b []byte, headLen int, arg uint64, path string) (int, error) {
	if arg > uint64(len(b)-headLen) {
		return 0, rejectSubjectData(path, "truncated CBOR string")
	}
	end := headLen + int(arg)
	if !utf8.Valid(b[headLen:end]) {
		return 0, rejectSubjectData(path, "CBOR text string that is not valid UTF-8")
	}
	return end, nil
}

// cborMapKey validates and reads a map key, which E.3 requires to be a text
// string, and returns its size in bytes.
func cborMapKey(b []byte, path string) (string, int, error) {
	mt, arg, headLen, indefinite, err := cborHead(b)
	if err != nil {
		return "", 0, rejectSubjectData(path, err.Error())
	}
	if mt == 2 {
		return "", 0, rejectSubjectData(path, "map key is a CBOR byte string, not a text string")
	}
	if mt != 3 {
		return "", 0, rejectSubjectData(path, "map key is not a text string")
	}
	if indefinite {
		return "", 0, rejectSubjectData(path, "indefinite-length CBOR map key")
	}
	size, err := definiteTextEnd(b, headLen, arg, path)
	if err != nil {
		return "", 0, err
	}
	return string(b[headLen:size]), size, nil
}

// ── Helpers ─────────────────────────────────────────────────────────

// cborBytesToBase64 reads a byte-string field (`pk`, `sig`) as base64. A
// missing or wrong-typed field yields "" rather than an error: E.6
// authorises no rejection for either, and E.9 / E.16 report what they find.
//
// E.3 lists `pk` and `sig` as byte strings, and only a byte string is
// accepted. A text form was previously routed through a length-and-alphabet
// heuristic that double-encoded a correctly formed base64 value, and every
// spelling it accepted was another wire encoding of one signed bundle.
func cborBytesToBase64(m map[string]any, key string) string {
	if val, ok := m[key].([]byte); ok {
		return base64.StdEncoding.EncodeToString(val)
	}
	return ""
}

// bytesFieldToHex reads one of E.3's byte-string fields (`mh`, `ph`, `mr`,
// `kid`, and per commitment entry `tx`, `memo`, `op`, `bmr`) as lowercase
// hex, which is the JSON wire form of the same field.
//
// Only a CBOR byte string is accepted, per E.3's field-correspondence MUST.
// Hex carried as a text string used to be taken verbatim, which meant one
// signed bundle had at least three spellings of every such field — bytes,
// lowercase hex text, and (because Go's hex alphabet check is
// case-insensitive) uppercase hex text — all hashing to the same value. A
// wrong-typed field yields "" and is graded by the step that needs it; E.6
// authorises no rejection here.
func bytesFieldToHex(m map[any]any, key string) string {
	if val, ok := m[key].([]byte); ok {
		return hex.EncodeToString(val)
	}
	return ""
}

// textFieldToString reads an E.3 text-string field (`rtx`, `txp`), which
// carries either base64url or hex. A text value is taken verbatim —
// routing it through bytesFieldToHex would hex-encode the base64url form
// into hex-of-ASCII and corrupt it. A byte string is hex-encoded to match
// the JSON wire form.
func textFieldToString(m map[any]any, key string) string {
	switch val := m[key].(type) {
	case []byte:
		return hex.EncodeToString(val)
	case string:
		return val
	default:
		return ""
	}
}

func getString(m map[any]any, key string) string {
	v, ok := m[key]
	if !ok {
		return ""
	}
	s, _ := v.(string)
	return s
}

// cborStringField reads an E.3 text-string field, tolerating the byte-string
// form the backend emits for `ip`. ok reports whether the key was present
// with a string-shaped value.
func cborStringField(m map[string]any, key string) (string, bool) {
	switch val := m[key].(type) {
	case string:
		return val, true
	case []byte:
		return base64.RawURLEncoding.EncodeToString(val), true
	default:
		return "", false
	}
}

// cborFieldCarried implements E.6's presence rule for a CBOR bundle field:
// a key that is missing, or whose value is null, counts as absent; any other
// value — including an empty string — counts as carried. The JSON path
// applies the same rule through [fieldCarried].
func cborFieldCarried(m map[string]any, key string) bool {
	v, ok := m[key]
	return ok && v != nil
}

// cborInteger reports whether a CBOR-decoded value is an integer, and
// returns it exactly. A float is not an integer however round its value is:
// E.3 maps `v`, `t`, `l`, and `h` to JSON integers, and truncating 1.9 to 1
// let the E.8 version step assert "Proof version 1" over a bundle that
// carries no such thing while the same document as JSON failed.
func cborInteger(v any) (intValue, bool) {
	switch n := v.(type) {
	case uint64:
		return intValue{N: int64(n), Fits: n <= math.MaxInt64, Text: strconv.FormatUint(n, 10)}, true
	case int64:
		return intValue{N: n, Fits: true, Text: strconv.FormatInt(n, 10)}, true
	case int:
		return intValue{N: int64(n), Fits: true, Text: strconv.Itoa(n)}, true
	case big.Int:
		return intValue{Text: n.String()}, true
	case *big.Int:
		return intValue{Text: n.String()}, true
	default:
		return intValue{}, false
	}
}

// cborIntOrZero reads an integer field that E.6 does not gate (`v`, `l`,
// `h`). Anything that is not an integer, or that does not fit one, carries
// through as 0 for the step that grades it to report — the JSON path does
// the same.
func cborIntOrZero(v any) int {
	n, ok := cborInteger(v)
	if !ok || !n.Fits {
		return 0
	}
	return int(n.N)
}

// normalizeMap converts both map[string]any and map[any]any to map[any]any.
func normalizeMap(v any) map[any]any {
	switch m := v.(type) {
	case map[any]any:
		return m
	case map[string]any:
		return toAnyKeyMap(m)
	default:
		return nil
	}
}

func toAnyKeyMap(m map[string]any) map[any]any {
	result := make(map[any]any, len(m))
	for k, v := range m {
		result[k] = v
	}
	return result
}

// anyToJSON converts a CBOR-decoded value to JSON bytes.
func anyToJSON(v any) (json.RawMessage, error) {
	converted, err := convertForJSON(v, "s.d")
	if err != nil {
		return nil, err
	}
	b, err := json.Marshal(converted)
	if err != nil {
		return nil, err
	}
	return json.RawMessage(b), nil
}

// convertForJSON recursively converts CBOR types to JSON-safe types, and is
// the second line of defence behind validateSubjectDataCBOR: the scanner
// works on raw bytes, so this catches anything that reaches the decoded tree
// by another route. Everything E.3 declares unrepresentable in JSON is a
// rejection here rather than a silent coercion.
func convertForJSON(v any, path string) (any, error) {
	switch val := v.(type) {
	case map[any]any:
		result := make(map[string]any, len(val))
		for k, v2 := range val {
			// Test the assertion, not the resulting string: "" is a legal
			// JSON object key and must survive.
			key, ok := k.(string)
			if !ok {
				return nil, rejectSubjectData(path, "map key is not a text string")
			}
			conv, err := convertForJSON(v2, path+"."+key)
			if err != nil {
				return nil, err
			}
			result[key] = conv
		}
		return result, nil
	case map[string]any:
		result := make(map[string]any, len(val))
		for k, v2 := range val {
			conv, err := convertForJSON(v2, path+"."+k)
			if err != nil {
				return nil, err
			}
			result[k] = conv
		}
		return result, nil
	case []any:
		result := make([]any, len(val))
		for i, v2 := range val {
			conv, err := convertForJSON(v2, fmt.Sprintf("%s[%d]", path, i))
			if err != nil {
				return nil, err
			}
			result[i] = conv
		}
		return result, nil
	case []byte:
		return nil, rejectSubjectData(path, "CBOR byte string")
	case cbor.Tag:
		return nil, rejectSubjectData(path, fmt.Sprintf("CBOR tag %d", val.Number))
	case cbor.RawTag:
		return nil, rejectSubjectData(path, fmt.Sprintf("CBOR tag %d", val.Number))
	case cbor.SimpleValue:
		return nil, rejectSubjectData(path, fmt.Sprintf("CBOR simple value %d", uint8(val)))
	case time.Time:
		return nil, rejectSubjectData(path, "CBOR date/time tag")
	case big.Int:
		return nil, rejectSubjectData(path, "CBOR bignum tag")
	case *big.Int:
		return nil, rejectSubjectData(path, "CBOR bignum tag")
	case float64:
		if math.IsNaN(val) || math.IsInf(val, 0) {
			return nil, rejectSubjectData(path, "CBOR non-finite float")
		}
		return floatLiteral(val), nil
	default:
		return val, nil
	}
}

// floatLiteral renders a CBOR float as a JSON number literal that still
// reads as a float. encoding/json writes every |x| in [1e-6, 1e21)
// positionally, so a CBOR double of 1e20 came back as the characters
// "100000000000000000000" — syntactically an integer literal. That
// misclassification was observable twice: E.4's non-portable-integer report
// fired on a value that is a perfectly portable IEEE-754 double, so the same
// logical bundle produced one report as JSON and a different one as CBOR;
// and re-encoding the bundle to CBOR failed, because the literal is outside
// the exactly representable integer range. JCS parses either spelling into
// the same double, so the 0x11 / 0x21 digest is unchanged.
func floatLiteral(f float64) json.Number {
	s := strconv.FormatFloat(f, 'g', -1, 64)
	if !strings.ContainsAny(s, ".eE") {
		s += ".0"
	}
	return json.Number(s)
}
