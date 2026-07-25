// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: MIT

package proof

import (
	"bytes"
	"strings"
	"testing"

	"github.com/fxamacker/cbor/v2"
)

// cborBundle builds a tagged CBOR item bundle, applying mutate to the
// top-level map first so a test can vary one field at a time.
func cborBundle(t *testing.T, mutate func(m map[string]any)) []byte {
	t.Helper()
	m := map[string]any{
		"v":   1,
		"t":   uint16(20),
		"ts":  "2026-01-01T00:00:00Z",
		"pk":  bytes.Repeat([]byte{0x01}, 32),
		"sig": bytes.Repeat([]byte{0x02}, 64),
		"s": map[string]any{
			"id":  "01HJHB01T8FYZ7YTR9P5N62K5B",
			"d":   map[string]any{"name": "test"},
			"mh":  bytes.Repeat([]byte{0x03}, 32),
			"kid": bytes.Repeat([]byte{0x04}, 4),
		},
		"ip": []byte{0x00},
		"b": map[string]any{
			"id":  "019cf813-99b8-730a-84f1-5a711a9c355e",
			"ph":  bytes.Repeat([]byte{0x05}, 32),
			"mr":  bytes.Repeat([]byte{0x06}, 32),
			"mh":  bytes.Repeat([]byte{0x07}, 32),
			"kid": bytes.Repeat([]byte{0x08}, 4),
		},
		"cx": []any{map[string]any{
			"t":    uint16(40),
			"net":  "testnet",
			"memo": bytes.Repeat([]byte{0x09}, 32),
			"ep":   []byte{0x00},
		}},
	}
	if mutate != nil {
		mutate(m)
	}
	body, err := cbor.Marshal(m)
	if err != nil {
		t.Fatalf("cbor.Marshal: %v", err)
	}
	return append([]byte{0xd9, 0xd9, 0xf7}, body...)
}

// cborSubjectData builds a bundle whose `s.d` is the given raw CBOR bytes.
func cborSubjectData(t *testing.T, raw []byte) []byte {
	t.Helper()
	return cborBundle(t, func(m map[string]any) {
		m["s"].(map[string]any)["d"] = cbor.RawMessage(raw)
	})
}

// --- F13: bare-map CBOR ---------------------------------------------------

func TestIsCBORProof_BareMapAndTag(t *testing.T) {
	t.Parallel()
	tagged := cborBundle(t, nil)
	bare := tagged[3:]

	if !HasCBORTag(tagged) || HasCBORTag(bare) {
		t.Error("HasCBORTag must detect the 55799 wrapper and only that")
	}
	if !IsCBORProof(tagged) || !IsCBORProof(bare) {
		t.Error("IsCBORProof must accept both the tagged and the bare form")
	}
	// Bytes that can legitimately open a JSON document must never be
	// claimed by the CBOR path.
	for _, b := range [][]byte{[]byte("{"), []byte(" {"), []byte("\n{"), {0xef, 0xbb, 0xbf}, nil} {
		if IsCBORProof(b) {
			t.Errorf("IsCBORProof(%x) must be false", b)
		}
	}
}

func TestParseBytes_BareCBORMap(t *testing.T) {
	t.Parallel()
	bare := cborBundle(t, nil)[3:]
	bundle, err := ParseBytes(bare)
	if err != nil {
		t.Fatalf("bare CBOR map must parse: %v", err)
	}
	if bundle.Subject == nil || bundle.Subject.ID != "01HJHB01T8FYZ7YTR9P5N62K5B" {
		t.Errorf("bare CBOR map parsed to the wrong bundle: %+v", bundle.Subject)
	}
}

// --- F11: `s.d` value space ----------------------------------------------

// TestParseCBOR_SubjectDataValueSpace feeds hand-built `s.d` encodings, one
// per class E.3 declares unrepresentable in JSON. Hand-built bytes are
// required: the decoder collapses `undefined` onto `null` and unwraps a
// nested 55799 tag, so a walk over the decoded tree cannot see either.
func TestParseCBOR_SubjectDataValueSpace(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name   string
		sd     []byte
		detail string
	}{
		{"byte string value", []byte{0xa1, 0x61, 'k', 0x44, 0xde, 0xad, 0xbe, 0xef}, "CBOR byte string"},
		{"byte string in array", []byte{0xa1, 0x61, 'k', 0x81, 0x42, 0x01, 0x02}, "CBOR byte string"},
		{"nested 55799 tag", []byte{0xa1, 0x61, 'k', 0xd9, 0xd9, 0xf7, 0x61, 'x'}, "CBOR tag 55799"},
		{"epoch time tag", []byte{0xa1, 0x61, 'k', 0xc1, 0x1a, 0x65, 0x51, 0xf0, 0x80}, "CBOR tag 1"},
		{"undefined", []byte{0xa1, 0x61, 'k', 0xf7}, "CBOR undefined"},
		{"simple value 99", []byte{0xa1, 0x61, 'k', 0xf8, 0x63}, "CBOR simple value 99"},
		{"unassigned simple 16", []byte{0xa1, 0x61, 'k', 0xf0}, "CBOR simple value 16"},
		{"integer map key", []byte{0xa1, 0x01, 0x61, 'x'}, "map key is not a text string"},
		{"nested integer map key", []byte{0xa1, 0x61, 'k', 0xa1, 0x01, 0x61, 'x'}, "map key is not a text string"},
		{"NaN", []byte{0xa1, 0x61, 'k', 0xfb, 0x7f, 0xf8, 0, 0, 0, 0, 0, 0}, "CBOR non-finite float"},
		{"+Inf", []byte{0xa1, 0x61, 'k', 0xfb, 0x7f, 0xf0, 0, 0, 0, 0, 0, 0}, "CBOR non-finite float"},
		{"half-precision Inf", []byte{0xa1, 0x61, 'k', 0xf9, 0x7c, 0x00}, "CBOR non-finite float"},
		// Invalid UTF-8 does round-trip through the decoder, but only
		// because encoding/json substitutes U+FFFD for each bad byte — so
		// the wire bytes and the 0x11 preimage stop agreeing, and any two
		// invalid sequences of equal length collide.
		{"invalid UTF-8 value", []byte{0xa1, 0x61, 'k', 0x62, 0xff, 0xfe}, "not valid UTF-8"},
		{"invalid UTF-8 key", []byte{0xa1, 0x62, 0xff, 0xfe, 0x61, 'x'}, "not valid UTF-8"},
		// Indefinite-length spellings canonicalize to the same JSON as
		// their definite forms, so accepting them admits a second wire
		// encoding of one signed `s.d` (RFC 8949 section 4.2).
		{"indefinite map", []byte{0xbf, 0x61, 'k', 0x61, 'x', 0xff}, "indefinite-length CBOR item"},
		{"indefinite text string", []byte{0xa1, 0x61, 'k', 0x7f, 0x61, 'a', 0x61, 'b', 0xff}, "indefinite-length CBOR item"},
		{"indefinite array", []byte{0xa1, 0x61, 'k', 0x9f, 0x01, 0xff}, "indefinite-length CBOR item"},
		{"indefinite map key", []byte{0xa1, 0x7f, 0x61, 'a', 0xff, 0x61, 'x'}, "indefinite-length CBOR map key"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			_, err := ParseCBOR(cborSubjectData(t, c.sd))
			if err == nil {
				t.Fatal("expected a rejection")
			}
			if got := RejectionCode(err); got != CodeInvalidSubjectData {
				t.Fatalf("RejectionCode = %q, want %q (err: %v)", got, CodeInvalidSubjectData, err)
			}
			if !strings.Contains(err.Error(), c.detail) {
				t.Errorf("error must name the offending class %q, got: %v", c.detail, err)
			}
		})
	}
}

// TestParseCBOR_SubjectDataAccepted covers the values that DO have a JSON
// counterpart, including the empty-string key an earlier `key == ""` guard
// silently dropped from the signed preimage.
func TestParseCBOR_SubjectDataAccepted(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name string
		sd   []byte
		want string
	}{
		{"empty-string key", []byte{0xa1, 0x60, 0x61, 'x'}, `{"":"x"}`},
		{"text, bool, null", []byte{0xa3, 0x61, 'a', 0x61, 'x', 0x61, 'b', 0xf5, 0x61, 'c', 0xf6}, `{"a":"x","b":true,"c":null}`},
		{"integer past 2^53", []byte{0xa1, 0x61, 'n', 0x1b, 0x00, 0x20, 0x00, 0x00, 0x00, 0x00, 0x00, 0x01}, `{"n":9007199254740993}`},
		{"negative integer", []byte{0xa1, 0x61, 'n', 0x38, 0x63}, `{"n":-100}`},
		{"float", []byte{0xa1, 0x61, 'f', 0xfb, 0x3f, 0xf8, 0, 0, 0, 0, 0, 0}, `{"f":1.5}`},
		{"empty map", []byte{0xa0}, `{}`},
		{"array", []byte{0xa1, 0x61, 'l', 0x82, 0x01, 0x61, 'x'}, `{"l":[1,"x"]}`},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			bundle, err := ParseCBOR(cborSubjectData(t, c.sd))
			if err != nil {
				t.Fatalf("must parse: %v", err)
			}
			if string(bundle.RawData) != c.want {
				t.Errorf("s.d: got %s, want %s", bundle.RawData, c.want)
			}
		})
	}
}

// TestParseCBOR_ByteStringCollision is the concrete attack the value-space
// rule closes: a text `s.d.hash` and a byte-string one used to hash to the
// same 0x11 preimage, so one signature covered two distinct wire bundles.
func TestParseCBOR_ByteStringCollision(t *testing.T) {
	t.Parallel()
	text := cborSubjectData(t, []byte{0xa1, 0x64, 'h', 'a', 's', 'h', 0x68, 'd', 'e', 'a', 'd', 'b', 'e', 'e', 'f'})
	raw := cborSubjectData(t, []byte{0xa1, 0x64, 'h', 'a', 's', 'h', 0x44, 0xde, 0xad, 0xbe, 0xef})

	if _, err := ParseCBOR(text); err != nil {
		t.Fatalf("the text form must still parse: %v", err)
	}
	if got := RejectionCode(mustErr(t, raw)); got != CodeInvalidSubjectData {
		t.Errorf("the byte-string form must be rejected, got %q", got)
	}
}

func mustErr(t *testing.T, data []byte) error {
	t.Helper()
	_, err := ParseCBOR(data)
	if err == nil {
		t.Fatal("expected an error")
	}
	return err
}

// TestConvertForJSON_RejectsUnrepresentable exercises the second line of
// defence directly, for the decoded-tree types the byte scanner never sees.
func TestConvertForJSON_RejectsUnrepresentable(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name string
		in   any
	}{
		{"byte string", map[any]any{"k": []byte{1, 2}}},
		{"non-text key", map[any]any{uint64(1): "x"}},
		{"simple value", map[any]any{"k": cbor.SimpleValue(99)}},
		{"tag", map[any]any{"k": cbor.Tag{Number: 100, Content: uint64(1)}}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			_, err := convertForJSON(c.in, "s.d")
			if RejectionCode(err) != CodeInvalidSubjectData {
				t.Errorf("want %q, got: %v", CodeInvalidSubjectData, err)
			}
		})
	}
	// An empty-string key is legal JSON and must survive.
	out, err := convertForJSON(map[any]any{"": "x"}, "s.d")
	if err != nil {
		t.Fatalf("empty-string key must survive: %v", err)
	}
	if m, ok := out.(map[string]any); !ok || m[""] != "x" {
		t.Errorf(`empty-string key dropped: %+v`, out)
	}
}

// --- F21 / F14 / decision on the root key, on the CBOR path ---------------

// TestParseCBOR_BlockLikeShapeRule is the CBOR half of the E.6 shape rule.
// The two serializations used to disagree: JSON accepted a block-like
// bundle carrying `ip: ""` and CBOR rejected `s: null`.
func TestParseCBOR_BlockLikeShapeRule(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name     string
		mutate   func(m map[string]any)
		rejected bool
	}{
		{"neither", func(m map[string]any) { delete(m, "s"); delete(m, "ip") }, false},
		{"s null", func(m map[string]any) { m["s"] = nil; delete(m, "ip") }, false},
		{"ip null", func(m map[string]any) { delete(m, "s"); m["ip"] = nil }, false},
		{"ip empty string", func(m map[string]any) { delete(m, "s"); m["ip"] = "" }, true},
		{"ip empty bytes", func(m map[string]any) { delete(m, "s"); m["ip"] = []byte{} }, true},
		{"s object", func(m map[string]any) { delete(m, "ip") }, true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			data := cborBundle(t, func(m map[string]any) {
				m["t"] = uint16(10)
				c.mutate(m)
			})
			_, err := ParseCBOR(data)
			got := RejectionCode(err) == CodeUnexpectedSubjectFieldsForBlockLike
			if got != c.rejected {
				t.Errorf("rejected = %v, want %v (err: %v)", got, c.rejected, err)
			}
		})
	}
}

// TestParseCBOR_CommitmentGates mirrors the JSON gates so a bundle grades
// identically whichever serialization carried it.
func TestParseCBOR_CommitmentGates(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name   string
		mutate func(cx map[string]any)
		code   string
	}{
		{"ep absent", func(cx map[string]any) { delete(cx, "ep") }, CodeInvalidExternalCommitmentEntry},
		{"ep null", func(cx map[string]any) { cx["ep"] = nil }, CodeInvalidExternalCommitmentEntry},
		{"ep not a string", func(cx map[string]any) { cx["ep"] = 7 }, CodeInvalidExternalCommitmentEntry},
		{"ep empty", func(cx map[string]any) { cx["ep"] = []byte{} }, ""},
		{"memo absent", func(cx map[string]any) { delete(cx, "memo") }, CodeInvalidExternalCommitmentEntry},
		{"memo null", func(cx map[string]any) { cx["memo"] = nil }, CodeInvalidExternalCommitmentEntry},
		{"memo empty", func(cx map[string]any) { cx["memo"] = []byte{} }, ""},
		{"memo malformed", func(cx map[string]any) { cx["memo"] = []byte{0xaa, 0xbb} }, ""},
		{"tx absent", func(cx map[string]any) { delete(cx, "tx") }, ""},
		{"bad code", func(cx map[string]any) { cx["t"] = uint16(99) }, CodeInvalidExternalCommitmentEntry},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			data := cborBundle(t, func(m map[string]any) {
				c.mutate(m["cx"].([]any)[0].(map[string]any))
			})
			_, err := ParseCBOR(data)
			if got := RejectionCode(err); got != c.code {
				t.Errorf("RejectionCode = %q, want %q (err: %v)", got, c.code, err)
			}
		})
	}
}

// TestParseCBOR_VersionIsNotAGate mirrors E.6's `v` exception on the CBOR
// path, where an absent version used to abort through toInt's zero value.
func TestParseCBOR_VersionIsNotAGate(t *testing.T) {
	t.Parallel()
	for _, c := range []struct {
		name   string
		mutate func(m map[string]any)
	}{
		{"absent", func(m map[string]any) { delete(m, "v") }},
		{"zero", func(m map[string]any) { m["v"] = 0 }},
	} {
		t.Run(c.name, func(t *testing.T) {
			bundle, err := ParseCBOR(cborBundle(t, c.mutate))
			if err != nil {
				t.Fatalf("v must not be a hard rejection: %v", err)
			}
			if bundle.Version != 0 {
				t.Errorf("version: got %d, want 0", bundle.Version)
			}
		})
	}
}

// TestParseCBOR_EmptyInclusionProof pins the `ip` gate as presence plus
// string-ness on the CBOR path too.
func TestParseCBOR_EmptyInclusionProof(t *testing.T) {
	t.Parallel()
	bundle, err := ParseCBOR(cborBundle(t, func(m map[string]any) { m["ip"] = []byte{} }))
	if err != nil {
		t.Fatalf("empty ip must parse: %v", err)
	}
	if bundle.InclusionProof != "" {
		t.Errorf("ip: got %q, want empty", bundle.InclusionProof)
	}
	_, err = ParseCBOR(cborBundle(t, func(m map[string]any) { m["ip"] = 7 }))
	if got := RejectionCode(err); got != CodeMissingInclusionProof {
		t.Errorf("RejectionCode = %q, want %q", got, CodeMissingInclusionProof)
	}
}

// TestParseCBOR_TextCommitmentFields pins the E.3 text-string routing for
// `rtx` / `txp`: a base64url value must survive, where the byte-string
// helper used to hex-encode its ASCII and corrupt it.
func TestParseCBOR_TextCommitmentFields(t *testing.T) {
	t.Parallel()
	data := cborBundle(t, func(m map[string]any) {
		cx := m["cx"].([]any)[0].(map[string]any)
		cx["t"] = uint16(41)
		delete(cx, "memo")
		cx["op"] = bytes.Repeat([]byte{0x0a}, 32)
		cx["rtx"] = "AgMEBQ"
		cx["txp"] = "AQIDBA"
	})
	bundle, err := ParseCBOR(data)
	if err != nil {
		t.Fatalf("ParseCBOR: %v", err)
	}
	if got := bundle.Commitments[0].RawTxHex; got != "AgMEBQ" {
		t.Errorf("rtx: got %q, want AgMEBQ", got)
	}
	if got := bundle.Commitments[0].TxoutproofHex; got != "AQIDBA" {
		t.Errorf("txp: got %q, want AQIDBA", got)
	}

	// The byte-string form still maps to hex, matching the JSON wire form.
	data = cborBundle(t, func(m map[string]any) {
		cx := m["cx"].([]any)[0].(map[string]any)
		cx["txp"] = []byte{0x01, 0x02}
	})
	bundle, err = ParseCBOR(data)
	if err != nil {
		t.Fatalf("ParseCBOR: %v", err)
	}
	if got := bundle.Commitments[0].TxoutproofHex; got != "0102" {
		t.Errorf("txp from bytes: got %q, want 0102", got)
	}
}

// --- F20: the encode direction -------------------------------------------

// TestMarshalCBOR_PreservesIntegers pins the round trip `convert proof`
// depends on. Re-reading `s.d` through float64 rewrote every integer as a
// CBOR float and rounded anything past 2^53, which silently changed the
// signed 0x11 preimage.
func TestMarshalCBOR_PreservesIntegers(t *testing.T) {
	t.Parallel()
	const sd = `{"big":9007159254740993,"huge":18446744073709551615,"neg":-9007199254740993,"small":3,"f":1.5}`
	src := `{"v":1,"t":20,"pk":"","sig":"","ts":"2026-01-01T00:00:00Z",
		"s":{"id":"x","d":` + sd + `,"mh":"","kid":""},
		"b":{"id":"e","ph":"","mr":"","mh":"","kid":""},
		"ip":"AA",
		"cx":[{"t":40,"net":"testnet","memo":"","ep":"AA"}]}`

	b1, err := ParseBytes([]byte(src))
	if err != nil {
		t.Fatalf("ParseBytes: %v", err)
	}
	cborBytes, err := b1.MarshalCBOR()
	if err != nil {
		t.Fatalf("MarshalCBOR: %v", err)
	}
	b2, err := ParseCBOR(cborBytes)
	if err != nil {
		t.Fatalf("ParseCBOR: %v", err)
	}
	for _, want := range []string{"9007159254740993", "18446744073709551615", "-9007199254740993", `"small":3`, `"f":1.5`} {
		if !strings.Contains(string(b2.RawData), want) {
			t.Errorf("json → cbor → json lost %s, got: %s", want, b2.RawData)
		}
	}
}

// TestMarshalCBOR_UnrepresentableIntegerNamesTheKey checks that a literal
// with no exact CBOR encoding aborts with the key in the message instead of
// emitting a rounded, and therefore corrupted, bundle.
func TestMarshalCBOR_UnrepresentableIntegerNamesTheKey(t *testing.T) {
	t.Parallel()
	src := `{"v":1,"t":20,"pk":"","sig":"","ts":"2026-01-01T00:00:00Z",
		"s":{"id":"x","d":{"outer":{"way_too_big":184467440737095516150}},"mh":"","kid":""},
		"b":{"id":"e","ph":"","mr":"","mh":"","kid":""},
		"ip":"AA",
		"cx":[{"t":40,"net":"testnet","memo":"","ep":"AA"}]}`

	b, err := ParseBytes([]byte(src))
	if err != nil {
		t.Fatalf("ParseBytes: %v", err)
	}
	_, err = b.MarshalCBOR()
	if err == nil {
		t.Fatal("an unrepresentable integer must abort the encode")
	}
	if !strings.Contains(err.Error(), "s.d.outer.way_too_big") {
		t.Errorf("error must name the key, got: %v", err)
	}
}

// TestMarshalCBOR_EmptyRootKeysRoundTrip covers the interaction between the
// E.6 presence gates and `omitempty`: an empty `ep` / `memo` must come back
// out of the encoder as a present, empty value rather than a dropped key
// the parser would then reject.
func TestMarshalCBOR_EmptyRootKeysRoundTrip(t *testing.T) {
	t.Parallel()
	src := `{"v":1,"t":20,"pk":"","sig":"","ts":"2026-01-01T00:00:00Z",
		"s":{"id":"x","d":{},"mh":"","kid":""},
		"b":{"id":"e","ph":"","mr":"","mh":"","kid":""},
		"ip":"",
		"cx":[{"t":40,"net":"testnet","memo":"","ep":""},{"t":41,"net":"regtest","op":"","ep":""}]}`

	b, err := ParseBytes([]byte(src))
	if err != nil {
		t.Fatalf("ParseBytes: %v", err)
	}

	cborBytes, err := b.MarshalCBOR()
	if err != nil {
		t.Fatalf("MarshalCBOR: %v", err)
	}
	if _, err := ParseCBOR(cborBytes); err != nil {
		t.Errorf("CBOR round trip must survive its own parser: %v", err)
	}

	jsonBytes, err := b.MarshalJSON()
	if err != nil {
		t.Fatalf("MarshalJSON: %v", err)
	}
	if _, err := ParseBytes(jsonBytes); err != nil {
		t.Errorf("JSON round trip must survive its own parser: %v\n%s", err, jsonBytes)
	}
	// The inapplicable root key stays absent — this is not a licence to
	// emit both on every entry.
	if bytes.Contains(jsonBytes, []byte(`"op":""},{`)) || strings.Count(string(jsonBytes), `"memo"`) != 1 {
		t.Errorf("only the chain's own root key may be forced: %s", jsonBytes)
	}
}
