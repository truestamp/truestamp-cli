// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: MIT

package jcs

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/rand/v2"
	"reflect"
	"strings"
	"testing"

	gojcs "github.com/gowebpki/jcs"
)

// digest0x11 is the claims_hash-style digest Appendix C.2a publishes for each
// of its vectors: SHA-256(0x11 || JCS(map)).
func digest0x11(canonical []byte) string {
	sum := sha256.Sum256(append([]byte{0x11}, canonical...))
	return hex.EncodeToString(sum[:])
}

// TestC2aVectors is the self-certification set Appendix E.25 points a port at.
// The last row, 2^53 + 1, is the regression pin for this package: with
// gowebpki/jcs alone it rounds down and collides with the 2^53 row above it,
// mapping two distinct normative vectors onto one digest.
func TestC2aVectors(t *testing.T) {
	vectors := []struct {
		name string
		in   string
		jcs  string
		want string
	}{
		{
			name: "explicit null",
			in:   `{"a": null, "b": 1}`,
			jcs:  `{"a":null,"b":1}`,
			want: "13ec6081eab8b87a582dc6c2d0f6ff561d79de8be10c652440b53e1b72a95099",
		},
		{
			name: "control character and tab",
			in:   `{"c": "\u0001", "tab": "\t"}`,
			jcs:  `{"c":"\u0001","tab":"\t"}`,
			want: "66d5ba7776a99a71bc63007419530297a231043bd74de64e9ef2b7b44f3deec2",
		},
		{
			name: "negative zero",
			in:   `{"z": -0.0}`,
			jcs:  `{"z":0}`,
			want: "901865b9e33220a2601150ae2f202ad45d172458c8eb5865ccfd695c353e2430",
		},
		{
			name: "exponential floats",
			in:   `{"big": 1.0e21, "small": 1.0e-7}`,
			jcs:  `{"big":1e+21,"small":1e-7}`,
			want: "2f01621ad310054550edf66468a6db72601e6e983464fb594cb884585beef1c7",
		},
		{
			name: "non-ASCII emitted unescaped",
			in:   "{\"name\": \"\u00c1vila\", \"emoji\": \"\U0001f600\"}",
			jcs:  "{\"emoji\":\"\U0001f600\",\"name\":\"\u00c1vila\"}",
			want: "65a18fbdd50f042c67de68433ede20023a529af601099cd735f4dd4805845eef",
		},
		{
			// Sorted on UTF-16 code units, so the astral key's leading
			// surrogate 0xD83D precedes 0xFFFF and the emoji comes first.
			name: "astral-plane key ordering",
			in:   "{\"\U0001f600\": 1, \"\uffff\": 2}",
			jcs:  "{\"\U0001f600\":1,\"\uffff\":2}",
			want: "d0951cfce65f8af38e98d67bd15833bb89afa2163e1b4c707047c126d53a6710",
		},
		{
			name: "2^53 is exactly representable",
			in:   `{"n": 9007199254740992}`,
			jcs:  `{"n":9007199254740992}`,
			want: "c2c43fad296f3c57c36919c193d533f212f5aeb88431960512c87d3db2a49fe6",
		},
		{
			name: "2^53 + 1 is emitted verbatim",
			in:   `{"n": 9007199254740993}`,
			jcs:  `{"n":9007199254740993}`,
			want: "f49f563152ef5c20c0aee41405be6342de6fbbb201985d7adc5bfebda9382407",
		},
	}

	for _, v := range vectors {
		t.Run(v.name, func(t *testing.T) {
			canonical, oversized, err := Canonicalize([]byte(v.in))
			if err != nil {
				t.Fatalf("Canonicalize: %v", err)
			}
			if string(canonical) != v.jcs {
				t.Errorf("canonical = %s, want %s", canonical, v.jcs)
			}
			if got := digest0x11(canonical); got != v.want {
				t.Errorf("0x11 digest = %s, want %s (oversized=%v)", got, v.want, oversized)
			}
		})
	}
}

// TestBelowThresholdIsByteIdentical is the no-regression proof: for every
// integer literal inside the safe range the output must be exactly what
// gowebpki/jcs produces today, so adopting this package cannot move a digest
// that already verifies.
func TestBelowThresholdIsByteIdentical(t *testing.T) {
	values := []int64{
		0, 1, -1,
		MaxExactInteger, -MaxExactInteger,
		MaxExactInteger - 1, -(MaxExactInteger - 1),
		1000000, -999999999999999,
	}

	// A fixed seed keeps a divergence reproducible from the failure output.
	r := rand.New(rand.NewPCG(1, 2))
	count := 200000
	if testing.Short() {
		count = 5000
	}
	for i := 0; i < count; i++ {
		n := r.Int64N(MaxExactInteger + 1)
		if i%2 == 1 {
			n = -n
		}
		values = append(values, n)
	}

	divergences := 0
	for _, n := range values {
		for _, shape := range []string{`{"n":%d}`, `{"b":2,"a":[%d,1]}`} {
			in := []byte(fmt.Sprintf(shape, n))
			want, wantErr := gojcs.Transform(in)
			got, oversized, err := Canonicalize(in)
			if wantErr != nil || err != nil {
				t.Fatalf("%s: unexpected error: gojcs=%v ours=%v", in, wantErr, err)
			}
			if oversized != nil {
				t.Fatalf("%s: oversized = %v, want nil", in, oversized)
			}
			if !bytes.Equal(got, want) {
				divergences++
				if divergences <= 5 {
					t.Errorf("%s: got %s, gowebpki/jcs gives %s", in, got, want)
				}
			}
		}
	}
	t.Logf("checked %d literals in %d shapes, divergences=%d", len(values), 2, divergences)
}

func TestOversizedDetection(t *testing.T) {
	bigDigits := strings.Repeat("9", 400)

	tests := []struct {
		name string
		in   string
		want []string
	}{
		{"2^53 is in range", `{"n":9007199254740992}`, nil},
		{"negative 2^53 is in range", `{"n":-9007199254740992}`, nil},
		{"2^53 + 1", `{"n":9007199254740993}`, []string{"9007199254740993"}},
		{"negative 2^53 + 1", `{"n":-9007199254740993}`, []string{"-9007199254740993"}},
		{"uint64 max", `{"ledgerSequence":18446744073709551615}`, []string{"18446744073709551615"}},
		{"400 digits", `{"nonce":` + bigDigits + `}`, []string{bigDigits}},
		{"float with fraction", `{"n":1.5}`, nil},
		{"float with exponent", `{"n":1e2}`, nil},
		{"float beyond the safe range", `{"n":1e21}`, nil},
		{"negative zero float", `{"n":-0.0}`, nil},
		{"oversized magnitude spelled as a float", `{"n":9007199254740993e0}`, nil},
		{
			name: "nested, duplicated, ascending",
			in:   `{"neg":-9007199254740993,"dup":{"deep":[9007199254740995,9007199254740994]}}`,
			want: []string{"-9007199254740993", "9007199254740994", "9007199254740995"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, oversized, err := Canonicalize([]byte(tc.in))
			if err != nil {
				t.Fatalf("Canonicalize: %v", err)
			}
			if !reflect.DeepEqual(oversized, tc.want) {
				t.Errorf("oversized = %v, want %v", oversized, tc.want)
			}
			// The detection-only entry point must agree with the full one, so
			// a caller that skips canonicalization sees the same report.
			if got := OversizedIntegers([]byte(tc.in)); !reflect.DeepEqual(got, tc.want) {
				t.Errorf("OversizedIntegers = %v, want %v", got, tc.want)
			}
		})
	}
}

// TestStringLookalikeUntouched pins that detection keys on JSON structure, not
// on a byte search: the same digits inside a string are data, not a number.
func TestStringLookalikeUntouched(t *testing.T) {
	canonical, oversized, err := Canonicalize([]byte(`{"s":"9007199254740993","n":9007199254740993}`))
	if err != nil {
		t.Fatalf("Canonicalize: %v", err)
	}
	want := `{"n":9007199254740993,"s":"9007199254740993"}`
	if string(canonical) != want {
		t.Errorf("canonical = %s, want %s", canonical, want)
	}
	if !reflect.DeepEqual(oversized, []string{"9007199254740993"}) {
		t.Errorf("oversized = %v, want one entry", oversized)
	}
}

// TestArbitraryPrecisionInteger covers the interop asymmetry the splice closes
// for free: gowebpki/jcs fails outright on a literal too large for a float64,
// where the Elixir producer accepts it.
func TestArbitraryPrecisionInteger(t *testing.T) {
	in := []byte(`{"nonce":` + strings.Repeat("9", 400) + `}`)

	if _, err := gojcs.Transform(in); err == nil {
		t.Fatal("gowebpki/jcs unexpectedly accepted a 400-digit literal; this test no longer covers what it claims")
	}

	canonical, oversized, err := Canonicalize(in)
	if err != nil {
		t.Fatalf("Canonicalize: %v", err)
	}
	if !bytes.Equal(canonical, in) {
		t.Errorf("canonical = %s, want %s", canonical, in)
	}
	if len(oversized) != 1 {
		t.Errorf("oversized = %v, want one entry", oversized)
	}
}

// TestSilentRewritesAreRejected is the regression net for the strict RFC 8259
// gate. Every row is a document gowebpki/jcs accepts and rewrites into a
// DIFFERENT valid document, the failure mode is not a crash but a confident
// wrong digest, so each row asserts both that Canonicalize refuses and that the
// library on its own would have produced the named impostor.
func TestSilentRewritesAreRejected(t *testing.T) {
	cases := []struct {
		name     string
		in       string
		impostor string
	}{
		{"whitespace-separated scalars concatenate", `{"a":1 2}`, `{"a":12}`},
		{"leading plus is dropped", `{"a":+1}`, `{"a":1}`},
		{"leading zeros are dropped", `{"a":0009}`, `{"a":9}`},
		{"single leading zero is dropped", `{"a":01}`, `{"a":1}`},
		{"bare fraction gains a zero", `{"a":.5}`, `{"a":0.5}`},
		{"trailing dot is dropped", `{"a":1.}`, `{"a":1}`},
		{"array elements concatenate", `[1 2]`, `[12]`},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// The premise: without the gate these bytes canonicalize, silently,
			// to something else.
			got, libErr := gojcs.Transform([]byte(tc.in))
			if libErr != nil {
				t.Fatalf("gowebpki/jcs now rejects %s on its own (%v); this row no longer covers what it claims", tc.in, libErr)
			}
			if string(got) != tc.impostor {
				t.Fatalf("gowebpki/jcs rewrote %s to %s, expected the impostor %s", tc.in, got, tc.impostor)
			}
			if json.Valid([]byte(tc.in)) {
				t.Fatalf("%s is valid JSON after all; this row no longer covers what it claims", tc.in)
			}

			if _, _, err := Canonicalize([]byte(tc.in)); err == nil {
				t.Fatalf("Canonicalize(%s) accepted RFC-8259-invalid input", tc.in)
			}
			if _, err := Transform([]byte(tc.in)); err == nil {
				t.Fatalf("Transform(%s) accepted RFC-8259-invalid input", tc.in)
			}
		})
	}
}

// TestSilentRewritesDoNotCollide is the user-facing half of the same defect:
// `truestamp hash --jcs` digests whatever Canonicalize returns, so before the
// gate two byte-distinct inputs, one malformed, one not, produced one digest.
func TestSilentRewritesDoNotCollide(t *testing.T) {
	malformed, wellFormed := []byte(`{"a":1 2}`), []byte(`{"a":12}`)

	impostor, err := gojcs.Transform(malformed)
	if err != nil {
		t.Fatalf("gowebpki/jcs rejected the fixture: %v", err)
	}
	honest, _, err := Canonicalize(wellFormed)
	if err != nil {
		t.Fatalf("Canonicalize(%s): %v", wellFormed, err)
	}
	if digest0x11(impostor) != digest0x11(honest) {
		t.Fatal("the two fixtures no longer collide under the library; this test no longer covers what it claims")
	}

	if _, _, err := Canonicalize(malformed); err == nil {
		t.Fatalf("Canonicalize(%s) produced the digest of %s", malformed, wellFormed)
	}
}

// TestStrictGateKeepsF3Signal covers the second half of the same deferral: the
// oversized-integer scan runs on encoding/json, so a malformed literal used to
// suppress the Appendix E.4 portability report AND emit the rounded digest.
// Both are now errors.
func TestStrictGateKeepsF3Signal(t *testing.T) {
	for _, in := range []string{
		`{"n":09007199254740993}`,
		`{"n":+9007199254740993}`,
		`{"a":1 2,"b":9007199254740993}`,
	} {
		canonical, oversized, err := Canonicalize([]byte(in))
		if err == nil {
			t.Errorf("Canonicalize(%s) = %s, oversized=%v, want an error", in, canonical, oversized)
		}
	}
}

// TestLibraryErrorTextPreserved pins that the gate only adds rejections. Bytes
// gowebpki/jcs rejects on its own keep its exact message, which several callers
// surface verbatim.
func TestLibraryErrorTextPreserved(t *testing.T) {
	for _, in := range []string{`{"a":1}{"b":2}`, `{"a":1,}`, "\ufeff{}", "{"} {
		_, wantErr := gojcs.Transform([]byte(in))
		if wantErr == nil {
			t.Fatalf("gowebpki/jcs accepted %q; this row no longer covers what it claims", in)
		}
		_, _, err := Canonicalize([]byte(in))
		if err == nil {
			t.Errorf("Canonicalize(%q) = nil error, want %q", in, wantErr)
			continue
		}
		if err.Error() != wantErr.Error() {
			t.Errorf("Canonicalize(%q) = %q, want the library's %q", in, err, wantErr)
		}
	}
}

// TestValidInputStillCanonicalizes is the counterweight to the gate: everything
// RFC 8259 allows must still come through, including the shapes that make the
// gate non-trivial, a bare top-level scalar, a 400-digit integer the library
// alone cannot handle, trailing whitespace, and a duplicate key that only the
// library rejects.
func TestValidInputStillCanonicalizes(t *testing.T) {
	for _, in := range []string{
		`5`, `"s"`, `null`, `true`, `[]`, `{}`,
		`{"a":1}  `, "\t{\"a\":1}\n",
		`{"n":9007199254740993}`,
		`{"nonce":` + strings.Repeat("9", 400) + `}`,
		`{"a":"😀","b":-0.0,"c":1e21}`,
	} {
		if !json.Valid([]byte(in)) {
			t.Fatalf("%q is not valid JSON; fixture is wrong", in)
		}
		if _, _, err := Canonicalize([]byte(in)); err != nil {
			t.Errorf("Canonicalize(%q) = %v, want it to canonicalize", in, err)
		}
	}
	// Duplicate keys stay the library's rejection, not the gate's: encoding/json
	// tolerates them and gowebpki/jcs does not.
	if _, _, err := Canonicalize([]byte(`{"a":1,"a":2}`)); err == nil {
		t.Error("duplicate keys should still be rejected")
	}
}

// TestFloatsAreNeverFlagged pins the settled F3 rule that only integers reach
// the Appendix E.4 report. The classification lives in one place, the base-10
// big.Int parse, and this table is what keeps a future refactor of it from
// silently starting to flag floats, which would add a warn row E.25 containment
// forbids.
func TestFloatsAreNeverFlagged(t *testing.T) {
	for _, in := range []string{
		`{"n":1.5}`,
		`{"n":1e2}`,
		`{"n":1E2}`,
		`{"n":1e21}`,
		`{"n":-0.0}`,
		`{"n":9007199254740993e0}`,
		`{"n":9007199254740993.0}`,
		`{"n":12345678901234567.5}`,
		`{"n":-1234567890123456789.25}`,
		`{"n":1.7976931348623157e308}`,
	} {
		canonical, oversized, err := Canonicalize([]byte(in))
		if err != nil {
			t.Fatalf("Canonicalize(%s): %v", in, err)
		}
		if oversized != nil {
			t.Errorf("Canonicalize(%s) flagged %v; floats are never flagged", in, oversized)
		}
		if got := OversizedIntegers([]byte(in)); got != nil {
			t.Errorf("OversizedIntegers(%s) = %v, want nil", in, got)
		}
		// A flagged literal would have been spliced; the library's unmodified
		// ES6 formatting path is what produces these bytes.
		want, err := gojcs.Transform([]byte(in))
		if err != nil {
			t.Fatalf("gowebpki/jcs rejected %s: %v", in, err)
		}
		if !bytes.Equal(canonical, want) {
			t.Errorf("Canonicalize(%s) = %s, want the library's %s", in, canonical, want)
		}
	}
}

// TestPlaceholderNeverOccursInInput guards the splice's core assumption. The
// retry loop inside placeholderFor cannot be reached by a constructible input
// (it would need a document containing its own digest), so the invariant is
// what gets tested, not the branch.
func TestPlaceholderNeverOccursInInput(t *testing.T) {
	inputs := []string{
		`{}`,
		`{"n":9007199254740993}`,
		`{"s":"` + placeholderPrefix + strings.Repeat("ab", 32) + `","n":9007199254740993}`,
		`{"s":"` + placeholderPrefix + `","n":9007199254740993}`,
		strings.Repeat("x", 4096),
	}
	for _, in := range inputs {
		if token := placeholderFor([]byte(in)); strings.Contains(in, token) {
			t.Errorf("placeholderFor(%.32q) returned a token present in the input", in)
		}
	}
}

// TestPlaceholderLookalikeSurvives is the end-to-end companion: a string that
// looks like a splice token must come through untouched.
func TestPlaceholderLookalikeSurvives(t *testing.T) {
	lookalike := placeholderPrefix + strings.Repeat("ab", 32) + "-0"
	in := []byte(`{"s":"` + lookalike + `","n":9007199254740993}`)

	canonical, oversized, err := Canonicalize(in)
	if err != nil {
		t.Fatalf("Canonicalize: %v", err)
	}
	want := `{"n":9007199254740993,"s":"` + lookalike + `"}`
	if string(canonical) != want {
		t.Errorf("canonical = %s, want %s", canonical, want)
	}
	if !reflect.DeepEqual(oversized, []string{"9007199254740993"}) {
		t.Errorf("oversized = %v, want one entry", oversized)
	}
}

// TestSpliceLeavesEverythingElseAlone checks that swapping literals in and out
// does not disturb key ordering, escaping or float formatting when an
// oversized integer shares the document with them.
func TestSpliceLeavesEverythingElseAlone(t *testing.T) {
	in := []byte("{\"\U0001f600\":9007199254740993,\"\uffff\":-0.0,\"a\":null,\"tab\":\"\\t\",\"e\":1.0e21}")
	// ASCII keys sort ahead of both non-ASCII keys, and among those the
	// astral key's leading surrogate 0xD83D precedes 0xFFFF.
	want := "{\"a\":null,\"e\":1e+21,\"tab\":\"\\t\",\"\U0001f600\":9007199254740993,\"\uffff\":0}"

	canonical, oversized, err := Canonicalize(in)
	if err != nil {
		t.Fatalf("Canonicalize: %v", err)
	}
	if string(canonical) != want {
		t.Errorf("canonical = %s, want %s", canonical, want)
	}
	if !reflect.DeepEqual(oversized, []string{"9007199254740993"}) {
		t.Errorf("oversized = %v, want one entry", oversized)
	}
}

// TestRestorePairsLiteralsAfterReordering is the load-bearing case for the
// restore walk: canonicalization sorts the keys, so the placeholders come back
// in a different order than they were spliced in and only the index carried in
// each one pairs it with the right literal.
func TestRestorePairsLiteralsAfterReordering(t *testing.T) {
	in := []byte(`{"z":9007199254740993,"m":123456789012345678901234567890,"a":18446744073709551615}`)
	want := `{"a":18446744073709551615,"m":123456789012345678901234567890,"z":9007199254740993}`

	canonical, oversized, err := Canonicalize(in)
	if err != nil {
		t.Fatalf("Canonicalize: %v", err)
	}
	if string(canonical) != want {
		t.Errorf("canonical = %s, want %s", canonical, want)
	}
	wantOversized := []string{"9007199254740993", "18446744073709551615", "123456789012345678901234567890"}
	if !reflect.DeepEqual(oversized, wantOversized) {
		t.Errorf("oversized = %v, want %v", oversized, wantOversized)
	}
}

// TestManyOversizedIntegers exercises the restore walk past single-digit
// placeholder indices, and pins that every literal lands under its own key.
func TestManyOversizedIntegers(t *testing.T) {
	const n = 250

	var in, want strings.Builder
	in.WriteByte('{')
	want.WriteByte('{')
	for i := 0; i < n; i++ {
		if i > 0 {
			in.WriteByte(',')
			want.WriteByte(',')
		}
		// Descending keys on the way in, so canonicalization has to reorder
		// every member; the value encodes the key it belongs to.
		key := fmt.Sprintf("k%03d", n-1-i)
		fmt.Fprintf(&in, `"%s":%s%03d`, key, "900719925474099", n-1-i)
		key = fmt.Sprintf("k%03d", i)
		fmt.Fprintf(&want, `"%s":%s%03d`, key, "900719925474099", i)
	}
	in.WriteByte('}')
	want.WriteByte('}')

	canonical, oversized, err := Canonicalize([]byte(in.String()))
	if err != nil {
		t.Fatalf("Canonicalize: %v", err)
	}
	if string(canonical) != want.String() {
		t.Errorf("canonical = %s, want %s", canonical, want.String())
	}
	if len(oversized) != n {
		t.Errorf("oversized has %d entries, want %d", len(oversized), n)
	}
}

// TestFloatErrorsUnchanged pins that the float path is untouched: a literal
// too large for a float64 but spelled as a float still fails the way it always
// has.
func TestFloatErrorsUnchanged(t *testing.T) {
	in := []byte(`{"huge":1e400}`)
	if _, gojcsErr := gojcs.Transform(in); gojcsErr == nil {
		t.Fatal("gowebpki/jcs unexpectedly accepted 1e400")
	}
	if _, _, err := Canonicalize(in); err == nil {
		t.Error("Canonicalize accepted 1e400, want the library's error")
	}
}

func TestCanonicalizeIsIdempotent(t *testing.T) {
	inputs := []string{
		`{"b":2,"a":1}`,
		`{"n":9007199254740993,"ledgerSequence":18446744073709551615}`,
		`{"z":-0.0,"big":1.0e21}`,
		`[9007199254740993,{"x":-9007199254740994}]`,
		`9007199254740993`,
	}
	for _, in := range inputs {
		once, _, err := Canonicalize([]byte(in))
		if err != nil {
			t.Fatalf("%s: %v", in, err)
		}
		twice, _, err := Canonicalize(once)
		if err != nil {
			t.Fatalf("%s (second pass): %v", once, err)
		}
		if !bytes.Equal(once, twice) {
			t.Errorf("%s: not idempotent: %s then %s", in, once, twice)
		}
	}
}

func TestTransformDiscardsReport(t *testing.T) {
	in := []byte(`{"n":9007199254740993}`)

	canonical, err := Transform(in)
	if err != nil {
		t.Fatalf("Transform: %v", err)
	}
	if string(canonical) != `{"n":9007199254740993}` {
		t.Errorf("canonical = %s", canonical)
	}
}

// TestEmptyAndNilInputMatchLibrary keeps the degenerate inputs on the library's
// error text, which several callers surface verbatim.
func TestEmptyAndNilInputMatchLibrary(t *testing.T) {
	for _, in := range [][]byte{nil, {}, []byte("   ")} {
		_, wantErr := gojcs.Transform(in)
		_, oversized, err := Canonicalize(in)
		if (err == nil) != (wantErr == nil) {
			t.Errorf("%q: err = %v, gowebpki/jcs gives %v", in, err, wantErr)
		}
		if err != nil && wantErr != nil && err.Error() != wantErr.Error() {
			t.Errorf("%q: err = %q, want %q", in, err, wantErr)
		}
		if oversized != nil {
			t.Errorf("%q: oversized = %v, want nil", in, oversized)
		}
	}
}

// TestOversizedIntegersOnUnreadableInput pins the detection-only deferral:
// bytes the scanner cannot read report nothing rather than panicking or
// guessing.
func TestOversizedIntegersOnUnreadableInput(t *testing.T) {
	for _, in := range []string{"", "{", `{"a":1 2}`, "not json"} {
		if got := OversizedIntegers([]byte(in)); got != nil {
			t.Errorf("OversizedIntegers(%q) = %v, want nil", in, got)
		}
	}
}

// --- Producer-side safe-integer guard (MaxSafeInteger / UnsafeIntegers) ---

// TestThresholdsDifferByOne is the tripwire. The producer bound and the
// verifier bound differ by exactly one, on purpose, and a future "cleanup"
// that unifies them has to delete this test and the comments it points at
// before it can go green.
//
// MaxSafeInteger is 2^53 - 1: RFC 8785 Appendix B's SHOULD, the largest
// magnitude a producer may emit. The Truestamp server enforces the same bound.
// MaxExactInteger is 2^53: the largest a verifier may warn about. 2^53 itself
// round-trips through a double exactly, so warning on it would be a false
// alarm about a bundle every conforming implementation can check.
//
// Be strict in what you emit, lenient in what you accept.
func TestThresholdsDifferByOne(t *testing.T) {
	if MaxSafeInteger != MaxExactInteger-1 {
		t.Fatalf("MaxSafeInteger = %d, MaxExactInteger = %d: the producer bound must be "+
			"exactly one below the verifier bound. If you are here to unify them, read the "+
			"doc comments on both constants first, the gap is deliberate.",
			MaxSafeInteger, MaxExactInteger)
	}
	if MaxSafeInteger != 9007199254740991 {
		t.Errorf("MaxSafeInteger = %d, want 9007199254740991 (2^53 - 1, JavaScript Number.MAX_SAFE_INTEGER)", MaxSafeInteger)
	}
	if MaxExactInteger != 9007199254740992 {
		t.Errorf("MaxExactInteger = %d, want 9007199254740992 (2^53)", MaxExactInteger)
	}
}

// decodeClaims parses a document the way cmd/create does: UseNumber, so every
// integer literal survives as text. A test that used json.Unmarshal here would
// be testing the bug rather than the fix.
func decodeClaims(t *testing.T, doc string) map[string]any {
	t.Helper()
	dec := json.NewDecoder(strings.NewReader(doc))
	dec.UseNumber()
	var out map[string]any
	if err := dec.Decode(&out); err != nil {
		t.Fatalf("decoding %s: %v", doc, err)
	}
	return out
}

// TestUnsafeIntegersBoundary walks the producer threshold from both directions
// and in both signs. The 2^53 rows are the interesting ones: the verifier
// tolerates that value, and this side must not.
func TestUnsafeIntegersBoundary(t *testing.T) {
	for _, tc := range []struct {
		name   string
		lit    string
		reject bool
	}{
		{"2^53 - 1 is the largest producible integer", "9007199254740991", false},
		{"2^53 is rejected by the producer even though the verifier tolerates it", "9007199254740992", true},
		{"2^53 + 1 is rejected", "9007199254740993", true},
		{"-(2^53 - 1) is the smallest producible integer", "-9007199254740991", false},
		{"-2^53 is rejected", "-9007199254740992", true},
		{"unsigned 64-bit id is rejected", "18446744073709551615", true},
		{"small integer is fine", "42", false},
		{"zero is fine", "0", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got := UnsafeIntegers("claims", decodeClaims(t, `{"n":`+tc.lit+`}`))
			if tc.reject {
				if len(got) != 1 {
					t.Fatalf("UnsafeIntegers(%s) = %v, want exactly one violation", tc.lit, got)
				}
				if got[0].Path != "claims.n" {
					t.Errorf("path = %q, want %q", got[0].Path, "claims.n")
				}
				if got[0].Literal != tc.lit {
					t.Errorf("literal = %q, want %q, the reported value must be the user's own bytes", got[0].Literal, tc.lit)
				}
				return
			}
			if len(got) != 0 {
				t.Errorf("UnsafeIntegers(%s) = %v, want no violations", tc.lit, got)
			}
		})
	}
}

// TestUnsafeIntegersIgnoresFloats pins the classifier: the producer rule is
// about integer literals, exactly as the verifier guard is. A huge value
// SPELLED as a float is not an integer literal and must not be flagged, or
// every geolocation and every scientific measurement becomes a false failure.
func TestUnsafeIntegersIgnoresFloats(t *testing.T) {
	doc := `{
	  "a": 1.5,
	  "b": 1e21,
	  "c": 1.0e-7,
	  "d": -0.0,
	  "e": 9007199254740993.0,
	  "f": 9007199254740993e0,
	  "g": 1.7976931348623157e308,
	  "lat": 37.7749,
	  "lon": -122.4194
	}`
	if got := UnsafeIntegers("claims", decodeClaims(t, doc)); len(got) != 0 {
		t.Errorf("UnsafeIntegers = %v, want none: floats are never a producer violation", got)
	}
}

// TestUnsafeIntegersIgnoresStrings confirms the documented escape hatch works:
// the fix a user is told to apply, send the value as a string, actually
// clears the guard.
func TestUnsafeIntegersIgnoresStrings(t *testing.T) {
	doc := `{"id":"18446744073709551615","nested":{"n":"9007199254740993"}}`
	if got := UnsafeIntegers("claims", decodeClaims(t, doc)); len(got) != 0 {
		t.Errorf("UnsafeIntegers = %v, want none: a stringified integer is the prescribed fix", got)
	}
}

// TestUnsafeIntegersNestedPaths exercises arbitrary-depth nesting through both
// maps and arrays, and pins the dotted-path syntax against the example the
// server's COMMS.md publishes: claims.metadata.rows[0].id.
func TestUnsafeIntegersNestedPaths(t *testing.T) {
	doc := `{
	  "metadata": {
	    "rows": [
	      {"id": 9007199254740993},
	      {"id": 42},
	      {"deep": {"deeper": [[18446744073709551615]]}}
	    ]
	  }
	}`
	got := UnsafeIntegers("claims", decodeClaims(t, doc))

	want := []UnsafeInteger{
		{Path: "claims.metadata.rows[0].id", Literal: "9007199254740993"},
		{Path: "claims.metadata.rows[2].deep.deeper[0][0]", Literal: "18446744073709551615"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("UnsafeIntegers = %#v,\nwant %#v", got, want)
	}
}

// TestUnsafeIntegersReportsEveryViolationDeterministically covers the two
// properties a user-facing list needs: completeness (fix them all in one pass,
// not one 422 at a time) and stable ordering despite Go's randomized map
// iteration.
func TestUnsafeIntegersReportsEveryViolationDeterministically(t *testing.T) {
	doc := `{
	  "zeta": 9007199254740993,
	  "alpha": 18446744073709551615,
	  "mid": {"beta": 9007199254740992, "ok": 7},
	  "arr": [1, 99999999999999999999, 3]
	}`
	claims := decodeClaims(t, doc)

	want := []UnsafeInteger{
		{Path: "claims.alpha", Literal: "18446744073709551615"},
		{Path: "claims.arr[1]", Literal: "99999999999999999999"},
		{Path: "claims.mid.beta", Literal: "9007199254740992"},
		{Path: "claims.zeta", Literal: "9007199254740993"},
	}

	// Repeat: one pass could agree with the expectation by luck of the map
	// iteration order. Many passes over a freshly decoded map cannot.
	for i := 0; i < 50; i++ {
		got := UnsafeIntegers("claims", decodeClaims(t, doc))
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("iteration %d: UnsafeIntegers = %#v,\nwant %#v", i, got, want)
		}
	}

	if got := len(UnsafeIntegers("claims", claims)); got != 4 {
		t.Errorf("violation count = %d, want 4 (every offender, not just the first)", got)
	}
}

// TestUnsafeIntegersGoNativeIntegers covers the values Go code could put into
// a claims map directly. Nothing in the producer path does today, but they
// marshal to exact integer literals, so the walk stays total over anything
// encoding/json can emit as an integer.
func TestUnsafeIntegersGoNativeIntegers(t *testing.T) {
	claims := map[string]any{
		"i":    int(9007199254740993),
		"i64":  int64(-9007199254740992),
		"u64":  uint64(18446744073709551615),
		"okay": int64(MaxSafeInteger),
		"f":    float64(1e30),
	}
	got := UnsafeIntegers("claims", claims)
	want := []UnsafeInteger{
		{Path: "claims.i", Literal: "9007199254740993"},
		{Path: "claims.i64", Literal: "-9007199254740992"},
		{Path: "claims.u64", Literal: "18446744073709551615"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("UnsafeIntegers = %#v,\nwant %#v", got, want)
	}
}

// TestUnsafeIntegersEmptyAndPortable confirms the common case returns nil
// rather than an empty non-nil slice, so `if len(...) == 0` and `if ... == nil`
// agree at every call site.
func TestUnsafeIntegersEmptyAndPortable(t *testing.T) {
	for _, doc := range []string{`{}`, `{"name":"doc","n":42,"nested":{"a":[1,2,3]}}`} {
		if got := UnsafeIntegers("claims", decodeClaims(t, doc)); got != nil {
			t.Errorf("UnsafeIntegers(%s) = %v, want nil", doc, got)
		}
	}
}

// TestUnsafeIntegerMessage pins the four facts the message must carry, path,
// value, allowed range, and the stringify remedy, and the exact wording, which
// mirrors Truestamp.SafeIntegers.message/2 server-side so the local and remote
// rejections read as one system.
func TestUnsafeIntegerMessage(t *testing.T) {
	got := UnsafeIntegerMessage("claims.metadata.rows[0].id", "9007199254740993")

	want := "The integer 9007199254740993 at claims.metadata.rows[0].id is outside the range " +
		"-9007199254740991 to 9007199254740991 (+/- 2^53 - 1). " +
		"A value outside this range cannot be reproduced by a verifier that parses JSON " +
		"numbers as IEEE-754 doubles, which is most of them, so the resulting proof would " +
		"not be portably verifiable. Send the value as a string instead."

	if got != want {
		t.Errorf("UnsafeIntegerMessage =\n%q\nwant\n%q", got, want)
	}
}

// TestUnsafeIntegersPreservesArbitraryPrecision confirms the reported literal
// is the user's own bytes at any width, not a value that went through a
// fixed-width Go type on its way to the report.
func TestUnsafeIntegersPreservesArbitraryPrecision(t *testing.T) {
	huge := strings.Repeat("9", 400)
	got := UnsafeIntegers("claims", decodeClaims(t, `{"nonce":`+huge+`}`))
	if len(got) != 1 {
		t.Fatalf("UnsafeIntegers = %v, want one violation", got)
	}
	if got[0].Literal != huge {
		t.Errorf("literal has %d digits, want the original %d", len(got[0].Literal), len(huge))
	}
}
