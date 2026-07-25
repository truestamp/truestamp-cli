// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: MIT

package proof

import (
	"bytes"
	"encoding/hex"
	"strings"
	"testing"

	"github.com/fxamacker/cbor/v2"
	"github.com/truestamp/truestamp-cli/internal/jcs"
	"github.com/truestamp/truestamp-cli/internal/proof/ptype"
)

// The tests here all guard one invariant: a signed bundle has exactly one
// wire form per serialization, and the two serializations accept the same
// set of bundles. Every case below is a document that used to verify (or to
// be refused) on one side and not the other.

// --- E.6 / E.24: the type code is an exact integer -----------------------

// TestParseCBOR_TypeCodeIsExact pins the truncation E.24 forbids. ptype.Code
// is a uint16, so `t = 65546` folded onto 10 (block) and verified end to end
// — the signed payload carries `t` as a uint16 too, so the signature still
// checked out over the truncated value. E.24: an unregistered code MUST be
// rejected as unsupported and a subject shape MUST NOT be guessed for it.
func TestParseCBOR_TypeCodeIsExact(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name   string
		t      any
		code   string
		detail string
	}{
		{"registered code truncated from above", uint64(65546), CodeInvalidSubjectTypeCode, "65546"},
		{"registered code truncated twice over", uint64(131082), CodeInvalidSubjectTypeCode, "131082"},
		{"item code truncated from above", uint64(65556), CodeInvalidSubjectTypeCode, "65556"},
		{"negative", int64(-1), CodeInvalidSubjectTypeCode, "-1"},
		{"past int64", uint64(18446744073709551615), CodeInvalidSubjectTypeCode, "18446744073709551615"},
		{"unregistered", uint64(99), CodeInvalidSubjectTypeCode, "99"},
		{"whole float", float64(20), CodeMissingTypeCode, "t is not an integer"},
		{"fractional float", float64(20.5), CodeMissingTypeCode, "t is not an integer"},
		{"text", "20", CodeMissingTypeCode, "t is not an integer"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			_, err := ParseCBOR(cborBundle(t, func(m map[string]any) { m["t"] = c.t }))
			if err == nil {
				t.Fatalf("t = %v must be rejected", c.t)
			}
			if got := RejectionCode(err); got != c.code {
				t.Errorf("RejectionCode = %q, want %q (err: %v)", got, c.code, err)
			}
			// The detail has to name the value the bundle carries: `t = -1`
			// used to be reported as "invalid subject type code: 65535", a
			// number that appears nowhere in the input.
			if !strings.Contains(err.Error(), c.detail) {
				t.Errorf("detail must name %q, got: %v", c.detail, err)
			}
		})
	}
}

// TestParseBytes_TypeCodeIdentifiers is the JSON half. E.6 assigns
// `missing_type_code` to "t is not an integer" and `invalid_subject_type_code`
// to "t not in {10, 11, 20, 30, 31, 32}". Decoding straight into a uint16
// collapsed the two, so -1 and 70000 — both integers — were reported as
// non-integers, disagreeing with the reference verifier on the vocabulary
// E.23 exists to make comparable.
func TestParseBytes_TypeCodeIdentifiers(t *testing.T) {
	t.Parallel()
	cases := []struct{ name, t, code string }{
		{"negative", `-1`, CodeInvalidSubjectTypeCode},
		{"past uint16", `70000`, CodeInvalidSubjectTypeCode},
		{"past int64", `99999999999999999999`, CodeInvalidSubjectTypeCode},
		{"unregistered", `99`, CodeInvalidSubjectTypeCode},
		{"quoted integer", `"20"`, CodeMissingTypeCode},
		{"fractional", `20.5`, CodeMissingTypeCode},
		{"whole float literal", `20.0`, CodeMissingTypeCode},
		{"exponent literal", `2e1`, CodeMissingTypeCode},
		{"bool", `true`, CodeMissingTypeCode},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			body := `{"v":1,"t":` + c.t + `,"b":{},"cx":[{"t":40,"memo":"","ep":""}]}`
			_, err := ParseBytes([]byte(body))
			if got := RejectionCode(err); got != c.code {
				t.Errorf("RejectionCode = %q, want %q (err: %v)", got, c.code, err)
			}
		})
	}
}

// TestParseCBOR_CommitmentTypeCodeIsExact applies the same exactness to
// `cx[].t`, which reached ptype.Code through the same truncating conversion.
func TestParseCBOR_CommitmentTypeCodeIsExact(t *testing.T) {
	t.Parallel()
	for _, tc := range []any{uint64(65576), float64(40.9), float64(40), "40", int64(-1)} {
		data := cborBundle(t, func(m map[string]any) {
			m["cx"].([]any)[0].(map[string]any)["t"] = tc
		})
		_, err := ParseCBOR(data)
		if got := RejectionCode(err); got != CodeInvalidExternalCommitmentEntry {
			t.Errorf("cx[0].t = %v: RejectionCode = %q, want %q (err: %v)",
				tc, got, CodeInvalidExternalCommitmentEntry, err)
		}
	}
}

// --- E.8: a non-integer version is not a version -------------------------

// TestParseCBOR_VersionIsNotTruncated pins E.8's step over the CBOR path.
// A `v` of 1.9 used to be truncated to 1, so the Version step asserted
// "Proof version 1 (expected 1)" with status pass over a bundle carrying no
// such version — while the same document as JSON failed the step.
func TestParseCBOR_VersionIsNotTruncated(t *testing.T) {
	t.Parallel()
	for _, v := range []any{float64(1.9), float64(1), "1", true} {
		bundle, err := ParseCBOR(cborBundle(t, func(m map[string]any) { m["v"] = v }))
		if err != nil {
			t.Fatalf("v = %v must parse — E.6 exempts `v` from the hard rejections: %v", v, err)
		}
		if bundle.Version != 0 {
			t.Errorf("v = %v carried through as %d, want 0 so E.8 can fail the step", v, bundle.Version)
		}
	}
	// The integer form still carries through.
	bundle, err := ParseCBOR(cborBundle(t, func(m map[string]any) { m["v"] = uint64(1) }))
	if err != nil || bundle.Version != 1 {
		t.Errorf("an integer v must carry through: version %d, err %v", bundle.Version, err)
	}
}

// --- E.3: only the whole bundle may wear the 55799 tag -------------------

// TestParseCBOR_SubjectDataSelfDescribeTag closes an unbounded malleability
// family. fxamacker strips a leading self-describe tag before populating a
// cbor.RawMessage, so `s.d`, `55799(s.d)`, `55799(55799(s.d))`, … were
// byte-distinct bundles with an identical 0x11 digest that all verified
// under one signature — and the scanner never saw the one tag number its
// own comment named. Every other tag number was already caught, which is
// what isolated the gap.
func TestParseCBOR_SubjectDataSelfDescribeTag(t *testing.T) {
	t.Parallel()
	sd := []byte{0xa1, 0x61, 'k', 0x61, 'x'} // {"k":"x"}
	selfDescribe := []byte{0xd9, 0xd9, 0xf7}

	for depth := 1; depth <= 3; depth++ {
		wrapped := append([]byte{}, sd...)
		for i := 0; i < depth; i++ {
			wrapped = append(append([]byte{}, selfDescribe...), wrapped...)
		}
		_, err := ParseCBOR(cborSubjectData(t, wrapped))
		if got := RejectionCode(err); got != CodeInvalidSubjectData {
			t.Errorf("depth %d: RejectionCode = %q, want %q (err: %v)",
				depth, got, CodeInvalidSubjectData, err)
		}
		if err != nil && !strings.Contains(err.Error(), "CBOR tag 55799") {
			t.Errorf("depth %d: the detail must name the tag, got: %v", depth, err)
		}
	}

	// The bare form is what a bundle is supposed to carry, and it still
	// parses — the outer bundle wrapper is unaffected either way.
	for _, data := range [][]byte{cborSubjectData(t, sd), cborSubjectData(t, sd)[3:]} {
		bundle, err := ParseCBOR(data)
		if err != nil {
			t.Fatalf("the untagged form must parse: %v", err)
		}
		if string(bundle.RawData) != `{"k":"x"}` {
			t.Errorf("s.d = %s", bundle.RawData)
		}
	}
}

// TestParseCBOR_UnwalkableSubjectDataIsRefused covers the fail-closed half
// of the tag fix. The decoder resolves encodings the byte walker declines to
// follow, and every one of them is a route back to hashing `s.d` bytes that
// nothing validated: an indefinite-length `d` key, or a second 55799
// wrapper around the whole bundle, both leave the tagged subject data
// unwrapped by the decoder and unseen by the scanner.
func TestParseCBOR_UnwalkableSubjectDataIsRefused(t *testing.T) {
	t.Parallel()
	selfDescribe := []byte{0xd9, 0xd9, 0xf7}
	taggedSD := append(append([]byte{}, selfDescribe...), 0xa1, 0x61, 'k', 0x61, 'x')

	// `s` with the key "d" written as an indefinite-length text string.
	var s bytes.Buffer
	s.WriteByte(0xa3)
	writeCBORText(&s, "id")
	writeCBORText(&s, "01HJHB01T8FYZ7YTR9P5N62K5B")
	s.Write([]byte{0x7f, 0x61, 'd', 0xff}) // indefinite "d"
	s.Write(taggedSD)
	writeCBORText(&s, "mh")
	s.Write([]byte{0x58, 0x20})
	s.Write(bytes.Repeat([]byte{0x03}, 32))

	cases := map[string][]byte{
		"indefinite-length d key": cborBundle(t, func(m map[string]any) {
			m["s"] = cbor.RawMessage(s.Bytes())
		}),
		"doubly wrapped bundle": append(append([]byte{}, selfDescribe...),
			cborSubjectData(t, taggedSD)...),
	}
	for name, data := range cases {
		t.Run(name, func(t *testing.T) {
			_, err := ParseCBOR(data)
			if got := RejectionCode(err); got != CodeInvalidSubjectData {
				t.Fatalf("RejectionCode = %q, want %q (err: %v)", got, CodeInvalidSubjectData, err)
			}
		})
	}
}

// TestParseCBOR_WalksPastIndefiniteMembers keeps the fail-closed guard from
// becoming a blanket refusal: members the walker only has to step over may
// be encoded any well-formed way, and a bare indefinite-length bundle map
// is one of the forms E.3 requires a verifier to accept.
func TestParseCBOR_WalksPastIndefiniteMembers(t *testing.T) {
	t.Parallel()
	// `ts` as an indefinite-length text string, sitting before `s`.
	indefiniteTS := cborBundle(t, func(m map[string]any) {
		m["ts"] = cbor.RawMessage(append([]byte{0x7f, 0x64, '2', '0', '2', '6'},
			0x70, '-', '0', '1', '-', '0', '1', 'T', '0', '0', ':', '0', '0', ':', '0', '0', 'Z', 0xff))
	})
	bare := cborBundle(t, nil)[3:]
	indefiniteMap := append([]byte{0xbf}, bare[1:]...)
	indefiniteMap = append(indefiniteMap, 0xff)

	for name, data := range map[string][]byte{
		"indefinite ts member":       indefiniteTS,
		"bare indefinite-length map": indefiniteMap,
	} {
		t.Run(name, func(t *testing.T) {
			bundle, err := ParseCBOR(data)
			if err != nil {
				t.Fatalf("must parse: %v", err)
			}
			if string(bundle.RawData) != `{"name":"test"}` {
				t.Errorf("s.d = %s, want the located subject data", bundle.RawData)
			}
		})
	}
}

// --- RFC 8949 section 5.6: duplicate map keys ----------------------------

// TestParseCBOR_DuplicateMapKeys is the split-view forgery: one signed blob
// that reads as the genuine claims under a last-wins decoder and as
// attacker-chosen text under a first-wins one. The same document expressed
// as JSON is refused by the JCS canonicalizer, so accepting it also broke
// ParseCBOR's "rejects exactly the same set of bundles" contract.
//
// The duplicate `d` case is the sharper one: the raw-byte scanner read the
// first `d` while the hashed tree took the last, so the bytes that were
// validated and the bytes that were hashed were different maps — and which
// one won depended on the order the attacker chose.
func TestParseCBOR_DuplicateMapKeys(t *testing.T) {
	t.Parallel()
	// {"description":"ATTACKER","description":"genuine"}
	dupInsideSD := []byte{0xa2,
		0x6b, 'd', 'e', 's', 'c', 'r', 'i', 'p', 't', 'i', 'o', 'n', 0x68, 'A', 'T', 'T', 'A', 'C', 'K', 'E', 'R',
		0x6b, 'd', 'e', 's', 'c', 'r', 'i', 'p', 't', 'i', 'o', 'n', 0x67, 'g', 'e', 'n', 'u', 'i', 'n', 'e'}

	clean := []byte{0xa1, 0x61, 'k', 0x61, 'x'}
	smuggled := []byte{0xa1, 0x61, 'k', 0x68, 's', 'm', 'u', 'g', 'g', 'l', 'e', 'd'}

	cases := []struct {
		name string
		data []byte
	}{
		{"duplicate key inside s.d", cborSubjectData(t, dupInsideSD)},
		{"duplicate d, clean first", cborBundle(t, func(m map[string]any) {
			m["s"] = cbor.RawMessage(dupKeyMap(t, "d", clean, smuggled))
		})},
		{"duplicate d, smuggled first", cborBundle(t, func(m map[string]any) {
			m["s"] = cbor.RawMessage(dupKeyMap(t, "d", smuggled, clean))
		})},
		{"duplicate top-level key", cborBundle(t, func(m map[string]any) {
			m["dup"] = cbor.RawMessage([]byte{0x01})
		})},
	}
	for _, c := range cases {
		data := c.data
		if c.name == "duplicate top-level key" {
			data = appendTopLevelDuplicate(t, cborBundle(t, nil), "ts")
		}
		t.Run(c.name, func(t *testing.T) {
			_, err := ParseCBOR(data)
			if err == nil {
				t.Fatal("a duplicate map key must be refused (RFC 8949 section 5.6)")
			}
			if got := RejectionCode(err); got == "" {
				t.Errorf("the refusal must carry an E.23 identifier, got: %v", err)
			}
			if !strings.Contains(err.Error(), "duplicate map key") {
				t.Errorf("the detail must name the duplicate, got: %v", err)
			}
		})
	}
}

// dupKeyMap builds a CBOR map that carries key twice, plus the `s` members
// the parser needs, so the two `d` values differ only in position.
func dupKeyMap(t *testing.T, key string, first, second []byte) []byte {
	t.Helper()
	var buf bytes.Buffer
	buf.WriteByte(0xa4) // 4 pairs: id, d, d, mh
	writeCBORText(&buf, "id")
	writeCBORText(&buf, "01HJHB01T8FYZ7YTR9P5N62K5B")
	writeCBORText(&buf, key)
	buf.Write(first)
	writeCBORText(&buf, key)
	buf.Write(second)
	writeCBORText(&buf, "mh")
	buf.Write([]byte{0x58, 0x20})
	buf.Write(bytes.Repeat([]byte{0x03}, 32))
	return buf.Bytes()
}

// appendTopLevelDuplicate re-emits a tagged bundle with key repeated at the
// top level, bumping the map header by one pair.
func appendTopLevelDuplicate(t *testing.T, tagged []byte, key string) []byte {
	t.Helper()
	body := tagged[3:]
	if body[0]>>5 != 5 || body[0]&0x1f >= 24 {
		t.Fatalf("unexpected bundle map header %#x", body[0])
	}
	var buf bytes.Buffer
	buf.Write([]byte{0xd9, 0xd9, 0xf7})
	buf.WriteByte(body[0] + 1)
	buf.Write(body[1:])
	writeCBORText(&buf, key)
	writeCBORText(&buf, "duplicate")
	return buf.Bytes()
}

func writeCBORText(buf *bytes.Buffer, s string) {
	switch {
	case len(s) < 24:
		buf.WriteByte(0x60 | byte(len(s)))
	default:
		buf.Write([]byte{0x78, byte(len(s))})
	}
	buf.WriteString(s)
}

// --- E.3: byte-string fields are byte strings ----------------------------

// TestParseCBOR_ByteStringFieldsRequireByteStrings pins E.3's
// field-correspondence MUST for the eight hash-shaped fields plus `pk` and
// `sig`. Hex carried as text used to be taken verbatim, and Go's hex
// alphabet check is case-insensitive, so one signed bundle had three
// spellings of every such field — bytes, lowercase hex text, uppercase hex
// text — that all decoded to the same value and verified identically.
func TestParseCBOR_ByteStringFieldsRequireByteStrings(t *testing.T) {
	t.Parallel()
	raw := bytes.Repeat([]byte{0x06}, 32)
	lower := hex.EncodeToString(raw)
	upper := strings.ToUpper(lower)

	for _, form := range []struct {
		name string
		val  any
	}{{"lowercase hex text", lower}, {"uppercase hex text", upper}, {"integer", uint64(1)}} {
		t.Run(form.name, func(t *testing.T) {
			bundle, err := ParseCBOR(cborBundle(t, func(m map[string]any) {
				m["b"].(map[string]any)["mr"] = form.val
			}))
			if err != nil {
				t.Fatalf("E.6 authorises no rejection for a wrong-typed b.mr: %v", err)
			}
			if bundle.Block.MerkleRoot != "" {
				t.Errorf("b.mr = %q, want empty so the step that needs it grades the anomaly",
					bundle.Block.MerkleRoot)
			}
		})
	}

	// The byte-string form is the one E.3 names, and it still decodes to
	// the lowercase hex the JSON wire form carries.
	bundle, err := ParseCBOR(cborBundle(t, nil))
	if err != nil {
		t.Fatalf("ParseCBOR: %v", err)
	}
	if bundle.Block.MerkleRoot != lower {
		t.Errorf("b.mr = %q, want %q", bundle.Block.MerkleRoot, lower)
	}
}

// TestParseCBOR_PublicKeyRequiresByteString covers the same rule for `pk`
// and `sig`, where the old text branch sorted a value by length and hex
// alphabet and double-base64-encoded anything it read as raw bytes.
func TestParseCBOR_PublicKeyRequiresByteString(t *testing.T) {
	t.Parallel()
	bundle, err := ParseCBOR(cborBundle(t, func(m map[string]any) {
		m["pk"] = "AQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQE=" // correct base64, wrong CBOR type
	}))
	if err != nil {
		t.Fatalf("E.9 reports a malformed pk; it is not a rejection: %v", err)
	}
	if bundle.PublicKey != "" {
		t.Errorf("pk = %q, want empty — E.3 lists pk as a byte string", bundle.PublicKey)
	}
}

// --- E.4: a CBOR float must not re-emit as an integer literal ------------

// TestParseCBOR_FloatStaysAFloat pins the JSON literal a CBOR double is
// re-emitted as. encoding/json writes every |x| in [1e-6, 1e21)
// positionally, so a double of 1e20 came back as the characters
// "100000000000000000000" — an integer literal to every downstream reader.
// Two things followed: E.4's non-portable-integer warn fired on a perfectly
// portable double, so one logical bundle produced two different reports
// depending on the serialization it arrived in; and re-encoding to CBOR
// failed because the literal is outside the exactly representable range.
func TestParseCBOR_FloatStaysAFloat(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name string
		sd   []byte
		want string
	}{
		// {"f": 1e20}
		{"large float", []byte{0xa1, 0x61, 'f', 0xfb, 0x44, 0x15, 0xaf, 0x1d, 0x78, 0xb5, 0x8c, 0x40}, `{"f":1e+20}`},
		// {"f": 3.0}
		{"whole float", []byte{0xa1, 0x61, 'f', 0xfb, 0x40, 0x08, 0, 0, 0, 0, 0, 0}, `{"f":3.0}`},
		// {"f": 1.5}
		{"fractional float", []byte{0xa1, 0x61, 'f', 0xfb, 0x3f, 0xf8, 0, 0, 0, 0, 0, 0}, `{"f":1.5}`},
		// {"n": 9007199254740993} — an integer stays an integer literal.
		{"large integer", []byte{0xa1, 0x61, 'n', 0x1b, 0x00, 0x20, 0x00, 0x00, 0x00, 0x00, 0x00, 0x01}, `{"n":9007199254740993}`},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			bundle, err := ParseCBOR(cborSubjectData(t, c.sd))
			if err != nil {
				t.Fatalf("ParseCBOR: %v", err)
			}
			if string(bundle.RawData) != c.want {
				t.Fatalf("s.d = %s, want %s", bundle.RawData, c.want)
			}
			// The literal spelling must not move the signed digest.
			// internal/jcs is the canonicalizer the verifier actually
			// digests through, so it is the one that has to accept the
			// emitted literal; the library alone would not notice the
			// wrapper's strict RFC 8259 gate rejecting it.
			canonical, _, err := jcs.Canonicalize(bundle.RawData)
			if err != nil {
				t.Fatalf("JCS must accept the emitted literal: %v", err)
			}
			// A second trip through the CBOR encoder has to survive too —
			// json → cbor → json → cbor was impossible for the large-float
			// case, because the integer-looking literal has no lossless
			// CBOR integer form.
			if _, err := decodeSubjectDataJSON(bundle.RawData); err != nil {
				t.Errorf("the emitted literal must re-encode to CBOR: %v", err)
			}
			t.Logf("canonical: %s", canonical)
		})
	}
}

// TestFloatLiteral covers the rendering rule directly: whatever the value,
// the literal has to stay recognizable as a float.
func TestFloatLiteral(t *testing.T) {
	t.Parallel()
	for _, f := range []float64{0, 1, 3, -0, 1e20, 1e16, 1.5, 1e-7, 1e21, -2} {
		got := string(floatLiteral(f))
		if !strings.ContainsAny(got, ".eE") {
			t.Errorf("floatLiteral(%v) = %q, which reads as an integer literal", f, got)
		}
	}
}

// --- Cross-format parity -------------------------------------------------

// TestParseBytes_WrongTypedFieldsAreGraded pins ParseCBOR's own contract
// ("rejects exactly the same set of bundles" as the JSON path) for the field
// types E.6 does not gate. The JSON path used to route any struct-unmarshal
// error into a hard rejection, so nine wrong-typed-but-present fields
// aborted with no report at all while the CBOR path graded them — and the
// reference verifier produced a full report for every one, four of them
// VERDICT: PASSED.
func TestParseBytes_WrongTypedFieldsAreGraded(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name  string
		json  string
		zerod func(*ProofBundle) bool
	}{
		{"b.id integer", `"b":{"id":5,"ph":"","mr":"","mh":"","kid":""}`,
			func(b *ProofBundle) bool { return b.Block.ID == "" }},
		{"b.mh integer", `"b":{"id":"e","ph":"","mr":"","mh":7,"kid":""}`,
			func(b *ProofBundle) bool { return b.Block.MetadataHash == "" }},
		{"b.kid bool", `"b":{"id":"e","ph":"","mr":"","mh":"","kid":true}`,
			func(b *ProofBundle) bool { return b.Block.SigningKeyID == "" }},
		{"s.id integer", `"s":{"id":5,"d":{},"mh":"","kid":""}`,
			func(b *ProofBundle) bool { return b.Subject.ID == "" }},
		{"s.mh list", `"s":{"id":"x","d":{},"mh":[1],"kid":""}`,
			func(b *ProofBundle) bool { return b.Subject.MetadataHash == "" }},
		{"cx.l string", `"cx":[{"t":40,"memo":"","ep":"","l":"18"}]`,
			func(b *ProofBundle) bool { return b.Commitments[0].Ledger == 0 }},
		{"cx.net integer", `"cx":[{"t":40,"memo":"","ep":"","net":5}]`,
			func(b *ProofBundle) bool { return b.Commitments[0].Network == "" }},
		{"cx.h string", `"cx":[{"t":41,"op":"","ep":"","h":"850000"}]`,
			func(b *ProofBundle) bool { return b.Commitments[0].BlockHeight == 0 }},
		{"cx.tx integer", `"cx":[{"t":40,"memo":"","ep":"","tx":5}]`,
			func(b *ProofBundle) bool { return b.Commitments[0].TransactionHash == "" }},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			bundle, err := ParseBytes([]byte(buildJSONBundle(c.json)))
			if err != nil {
				t.Fatalf("E.6 authorises no rejection for %s, so this must reach the report: %v", c.name, err)
			}
			// The wrong-typed member carries through as its zero value, so
			// a field feeding a hash preimage is still caught downstream —
			// by the step that derives the hash, not by an abort.
			if !c.zerod(bundle) {
				t.Errorf("%s: the wrong-typed member must not be salvaged into the bundle", c.name)
			}
		})
	}
}

// buildJSONBundle splices one member into an otherwise well-formed item
// bundle, replacing whichever top-level key the fragment names.
func buildJSONBundle(fragment string) string {
	members := map[string]string{
		"b":  `"b":{"id":"019cf813-99b8-730a-84f1-5a711a9c355e","ph":"","mr":"","mh":"","kid":""}`,
		"s":  `"s":{"id":"01HJHB01T8FYZ7YTR9P5N62K5B","d":{},"mh":"","kid":""}`,
		"cx": `"cx":[{"t":40,"memo":"","ep":""}]`,
	}
	key := fragment[1 : strings.Index(fragment[1:], `"`)+1]
	members[key] = fragment
	return `{"v":1,"t":20,"ts":"2026-01-01T00:00:00Z","pk":"","sig":"","ip":"AA",` +
		members["s"] + "," + members["b"] + "," + members["cx"] + "}"
}

// TestParseBytes_MapShapeIsStillGated makes sure the relaxation above did
// not take E.6's three map-shape rejections with it.
func TestParseBytes_MapShapeIsStillGated(t *testing.T) {
	t.Parallel()
	cases := []struct{ name, fragment, code string }{
		{"b is a list", `"b":[1]`, CodeMissingBlock},
		{"b is a string", `"b":"x"`, CodeMissingBlock},
		{"b is a number", `"b":5`, CodeMissingBlock},
		{"s is a list", `"s":[1]`, CodeMissingSubject},
		{"s is a number", `"s":5`, CodeMissingSubject},
		{"cx entry is a list", `"cx":[[1]]`, CodeInvalidExternalCommitmentEntry},
		{"cx entry is a number", `"cx":[5]`, CodeInvalidExternalCommitmentEntry},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			_, err := ParseBytes([]byte(buildJSONBundle(c.fragment)))
			if got := RejectionCode(err); got != c.code {
				t.Errorf("RejectionCode = %q, want %q (err: %v)", got, c.code, err)
			}
		})
	}
}

// TestParseBytes_SubjectDataBytesArePreserved guards the one thing the
// member-by-member decode must not change: `s.d` is canonicalized as the
// file wrote it, never as a Go round trip of it.
func TestParseBytes_SubjectDataBytesArePreserved(t *testing.T) {
	t.Parallel()
	body := `{"v":1,"t":20,"ts":"2026-01-01T00:00:00Z","pk":"","sig":"","ip":"AA",
		"s":{"id":"x","d":{ "b" : 1 , "a" : 9007199254740993 },"mh":"","kid":""},
		"b":{"id":"e","ph":"","mr":"","mh":"","kid":""},
		"cx":[{"t":40,"memo":"","ep":""}]}`
	bundle, err := ParseBytes([]byte(body))
	if err != nil {
		t.Fatalf("ParseBytes: %v", err)
	}
	want := `{ "b" : 1 , "a" : 9007199254740993 }`
	if string(bundle.RawData) != want {
		t.Errorf("s.d = %q, want the file's bytes %q", bundle.RawData, want)
	}
}

// TestParseCBOR_TypeCodeRegistryIsClosed is the belt to
// TestParseCBOR_TypeCodeIsExact's braces: every registered code parses, and
// nothing congruent to one modulo 2^16 does.
func TestParseCBOR_TypeCodeRegistryIsClosed(t *testing.T) {
	t.Parallel()
	for _, code := range []ptype.Code{ptype.Block, ptype.Beacon, ptype.Item, ptype.EntropyNIST, ptype.EntropyStellar, ptype.EntropyBitcoin} {
		blockLike := ptype.IsBlockLikeSubject(code)
		data := cborBundle(t, func(m map[string]any) {
			m["t"] = uint64(code)
			if blockLike {
				delete(m, "s")
				delete(m, "ip")
			}
		})
		bundle, err := ParseCBOR(data)
		if err != nil {
			t.Fatalf("registered code %d must parse: %v", code, err)
		}
		if bundle.T != code {
			t.Errorf("t = %d parsed as %d", code, bundle.T)
		}

		wrapped := cborBundle(t, func(m map[string]any) {
			m["t"] = uint64(code) + 65536
			if blockLike {
				delete(m, "s")
				delete(m, "ip")
			}
		})
		if got := RejectionCode(mustErr(t, wrapped)); got != CodeInvalidSubjectTypeCode {
			t.Errorf("t = %d + 2^16: RejectionCode = %q, want %q", code, got, CodeInvalidSubjectTypeCode)
		}
	}
}
