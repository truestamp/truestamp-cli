// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: Apache-2.0

// Tests of JCS and the canonical form of a parsed value (SPEC.md sections
// 2.2, 9 and 10.1), over the corpus's known answers for JCS, the worked
// example of section 6.3 and every case and data file of the vectors
// (section 13.2).

package jcs

import (
	"bytes"
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"math"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/truestamp/truestamp-cli/internal/specdata"
	"github.com/truestamp/truestamp-cli/internal/strictjson"
	"github.com/truestamp/truestamp-cli/internal/vectorcases"
)

// parseFile reads a vectors/ file and parses it under J2 to J6 (a data file
// may be larger than MAX_READ_OCTETS, which binds a document).
func parseFile(t *testing.T, relative string) ([]byte, any) {
	t.Helper()
	data, err := os.ReadFile(vectorcases.Path(relative))
	if err != nil {
		t.Fatal(err)
	}
	value, err := strictjson.Parse(data)
	if err != nil {
		t.Fatalf("%s: %v", relative, err)
	}
	return data, value
}

func mustEncode(t *testing.T, v any) []byte {
	t.Helper()
	m, err := Encode(v)
	if err != nil {
		t.Fatalf("Encode: %v", err)
	}
	return m
}

// Every pair of vectors/corpus/jcs/ (SPEC.md section 13.2): JCS of the
// input's value is the output's bytes, and the output is its own JCS.
// Mutants killed: sortedNames comparing UTF-8 bytes in place of UTF-16 code
// units (utf16-order), appendString writing \u00xx in uppercase hex
// (escapes), and numberText using strconv's 'g' format in place of
// NumberToJSON (numbers, rfc8785-numbers).
func TestEncode_CorpusJCS(t *testing.T) {
	inputs, err := filepath.Glob(vectorcases.Path("corpus/jcs/*.input.json"))
	if err != nil {
		t.Fatal(err)
	}
	var names []string
	for _, input := range inputs {
		name := strings.TrimSuffix(filepath.Base(input), ".input.json")
		names = append(names, name)
		_, value := parseFile(t, "corpus/jcs/"+name+".input.json")
		want, err := os.ReadFile(vectorcases.Path("corpus/jcs/" + name + ".output.json"))
		if err != nil {
			t.Fatal(err)
		}
		if got := mustEncode(t, value); !bytes.Equal(got, want) {
			t.Errorf("%s: JCS = %s\nwant    %s", name, got, want)
		}
		_, outputValue := parseFile(t, "corpus/jcs/"+name+".output.json")
		if got := mustEncode(t, outputValue); !bytes.Equal(got, want) {
			t.Errorf("%s: the output is not its own JCS", name)
		}
	}
	slices.Sort(names)
	wantNames := []string{
		"empty-and-literals", "escapes", "numbers", "rfc8785-example", "rfc8785-numbers",
		"rfc8785-sorting", "utf16-order", "whitespace", "worked-example",
	}
	if !slices.Equal(names, wantNames) {
		t.Fatalf("corpus/jcs/ pairs = %v, want %v", names, wantNames)
	}
}

// The worked example of SPEC.md section 6.3: JCS of its proof is its M,
// proof_hash is SHA-256(0x50 || M), and the served document is the JCS of
// its own value. Mutant killed: appendValue writing a space after the ','
// between two members of an object.
func TestEncode_WorkedExample(t *testing.T) {
	_, value := parseFile(t, "worked-example.json")
	example := value.(map[string]any)
	m := mustEncode(t, example["proof"])
	if string(m) != example["M"].(string) {
		t.Fatalf("JCS(proof) = %s\nwant M     %s", m, example["M"])
	}
	sum := sha256.Sum256(append([]byte{proofHashPrefix(t)}, m...))
	if hex.EncodeToString(sum[:]) != example["proof_hash"].(string) {
		t.Fatalf("proof_hash = %x, want %s", sum, example["proof_hash"])
	}
	document := []byte(example["document"].(string))
	parsed, err := strictjson.Parse(document)
	if err != nil {
		t.Fatal(err)
	}
	if got := mustEncode(t, parsed); !bytes.Equal(got, document) {
		t.Fatalf("the served document is not its own JCS")
	}
}

// proofHashPrefix reads proof_hash's byte from spec/prefixes.json.
func proofHashPrefix(t *testing.T) byte {
	t.Helper()
	data, err := specdata.File("prefixes.json")
	if err != nil {
		t.Fatal(err)
	}
	value, err := strictjson.Parse(data)
	if err != nil {
		t.Fatal(err)
	}
	b, err := specdata.PrefixByte(value.(map[string]any)["proof_hash"].(map[string]any)["byte"].(string))
	if err != nil {
		t.Fatal(err)
	}
	return b
}

// signatureCovers reports whether the envelope's V is a signature by its K
// over SHA-256(0x50 || JCS(proof)), so whether the JCS of the parsed proof
// is the M that was signed (SPEC.md sections 6.1 and 10.1). The stock
// ed25519.Verify stands in for the strict check of section 6.2 here: it is
// an oracle for M, not a verifier.
func signatureCovers(t *testing.T, prefix byte, envelope map[string]any) bool {
	t.Helper()
	signature := envelope["signature"].(map[string]any)
	key, err := base64.RawURLEncoding.DecodeString(signature["public_key"].(string))
	if err != nil || len(key) != ed25519.PublicKeySize {
		t.Fatalf("public_key: %v", err)
	}
	sig, err := base64.RawURLEncoding.DecodeString(signature["value"].(string))
	if err != nil {
		t.Fatalf("value: %v", err)
	}
	sum := sha256.Sum256(append([]byte{prefix}, mustEncode(t, envelope["proof"])...))
	return ed25519.Verify(key, sum[:], sig)
}

// Every sameness-* case (38, SPEC.md section 13.3) reads to the JCS that
// was signed: each reformatted sound proof reads to exactly its sound
// proof's served bytes, and every one's signature covers JCS of its parsed
// proof. Mutant killed: hex4 reading a lowercase hex digit without its
// + 10 (the sameness-format-* files spell every a in a string as an
// escape, whose hex ends in a lowercase digit).
func TestEncode_SamenessCasesReadToTheSignedJCS(t *testing.T) {
	cases, err := vectorcases.Load()
	if err != nil {
		t.Fatal(err)
	}
	prefix := proofHashPrefix(t)
	var count, reformatted int
	for _, c := range cases {
		if c.Family() != "sameness" {
			continue
		}
		count++
		data, err := strictjson.ReadFile(c.ProofPath())
		if err != nil {
			t.Fatalf("%s: %v", c.ID, err)
		}
		value, err := strictjson.Parse(data)
		if err != nil {
			t.Fatalf("%s: %v", c.ID, err)
		}
		if !signatureCovers(t, prefix, value.(map[string]any)) {
			t.Errorf("%s: the signature does not cover JCS of the parsed proof", c.ID)
		}
		if c.Expected.Trace == "" {
			continue
		}
		// A case that gives a sound proof's trace is that proof reformatted.
		reformatted++
		sound := strings.TrimSuffix(c.Expected.Trace, ".trace.json") + ".proof.json"
		want, err := os.ReadFile(vectorcases.Path(sound))
		if err != nil {
			t.Fatal(err)
		}
		if bytes.Equal(data, want) {
			t.Fatalf("%s: the file is not reformatted", c.ID)
		}
		if got := mustEncode(t, value); !bytes.Equal(got, want) {
			t.Errorf("%s: JCS of the envelope differs from %s", c.ID, sound)
		}
	}
	if count != 38 || reformatted != 12 {
		t.Fatalf("sameness cases = %d (%d reformatted), want 38 (12)", count, reformatted)
	}
}

// Every case whose SIGNATURE row passes carries a signature over JCS of its
// parsed proof, and every sound proof is the JCS of its own value. Mutant
// killed: sortedNames sorting by Go string order with names compared in
// reverse.
func TestEncode_EverySignedCaseReadsToItsM(t *testing.T) {
	cases, err := vectorcases.Load()
	if err != nil {
		t.Fatal(err)
	}
	prefix := proofHashPrefix(t)
	var count int
	for _, c := range cases {
		if !slices.ContainsFunc(c.Expected.Rows, func(r vectorcases.Row) bool {
			return r.Step == "SIGNATURE" && r.Status == "pass"
		}) {
			continue
		}
		count++
		data, err := strictjson.ReadFile(c.ProofPath())
		if err != nil {
			t.Fatalf("%s: %v", c.ID, err)
		}
		value, err := strictjson.Parse(data)
		if err != nil {
			t.Fatalf("%s: %v", c.ID, err)
		}
		if !signatureCovers(t, prefix, value.(map[string]any)) {
			t.Errorf("%s: the signature does not cover JCS of the parsed proof", c.ID)
		}
	}
	if count < 300 {
		t.Fatalf("only %d cases pass SIGNATURE", count)
	}

	sound, err := filepath.Glob(vectorcases.Path("positive/*.proof.json"))
	if err != nil || len(sound) != 11 {
		t.Fatalf("positive proofs = %d (%v)", len(sound), err)
	}
	for _, name := range sound {
		data, value := parseFile(t, "positive/"+filepath.Base(name))
		if got := mustEncode(t, value); !bytes.Equal(got, data) {
			t.Errorf("%s is not the JCS of its value", filepath.Base(name))
		}
		if len(data) > specdata.MaxDocumentOctets {
			t.Errorf("%s is over MAX_DOCUMENT_OCTETS", filepath.Base(name))
		}
	}
}

// Every data file SPEC.md sections 9 and 13.2 say is in the canonical form
// is that form of its own value: the files under spec/ but the constructions
// texts, the vectors' own files, the trace files, the corpus files but jcs/,
// and the keyrings under negative/; and a constructions text is the JCS of
// its value. Mutant killed: appendForm writing a member's name and value
// with ":" and no space between them.
func TestCanonicalForm_DataFiles(t *testing.T) {
	var files []string
	for _, name := range specdata.Names() {
		data, err := specdata.File(name)
		if err != nil {
			t.Fatal(err)
		}
		value, err := strictjson.Parse(data)
		if err != nil {
			t.Fatalf("spec/%s: %v", name, err)
		}
		if strings.HasPrefix(name, "constructions/") {
			if got := mustEncode(t, value); !bytes.Equal(got, data) {
				t.Errorf("spec/%s is not the JCS of its value", name)
			}
			continue
		}
		files = append(files, "spec/"+name)
		checkCanonical(t, "spec/"+name, data, value)
	}

	globs := []string{
		"*.json", "positive/*.trace.json", "corpus/*.json", "corpus/bitcoin/*.json",
		"corpus/replay/*.json", "negative/*.keyring.json",
	}
	for _, glob := range globs {
		matches, err := filepath.Glob(vectorcases.Path(glob))
		if err != nil {
			t.Fatal(err)
		}
		for _, match := range matches {
			relative, err := filepath.Rel(vectorcases.Dir(), match)
			if err != nil {
				t.Fatal(err)
			}
			relative = filepath.ToSlash(relative)
			if relative == "negative/configuration-keyring-duplicate_name.keyring.json" {
				// The case concerns the file's bytes: a repeated name.
				continue
			}
			files = append(files, relative)
			data, value := parseFile(t, relative)
			checkCanonical(t, relative, data, value)
		}
	}
	if len(files) < 120 {
		t.Fatalf("only %d canonical files checked", len(files))
	}
}

func checkCanonical(t *testing.T, name string, data []byte, value any) {
	t.Helper()
	got, err := CanonicalForm(value)
	if err != nil {
		t.Fatalf("%s: %v", name, err)
	}
	if !bytes.Equal(got, data) {
		t.Errorf("%s is not in the canonical form of its value", name)
	}
}

// The canonical form of small values, written out. Mutant killed:
// appendIndent writing one space per level.
func TestCanonicalForm_Shape(t *testing.T) {
	value := map[string]any{
		"b": []any{strictjson.Number(1), map[string]any{}, []any{}},
		"a": map[string]any{"x": nil, "\u00e9": "\n"},
	}
	want := "{\n  \"a\": {\n    \"x\": null,\n    \"\u00e9\": \"\\n\"\n  },\n  \"b\": [\n    1,\n    {},\n    []\n  ]\n}\n"
	got, err := CanonicalForm(value)
	if err != nil || string(got) != want {
		t.Fatalf("CanonicalForm = %q, %v\nwant %q", got, err, want)
	}
	if got, err := CanonicalForm("s"); err != nil || string(got) != "\"s\"\n" {
		t.Fatalf("CanonicalForm(scalar) = %q, %v", got, err)
	}
	if _, err := CanonicalForm(map[string]any{"a": 1.5}); !errors.Is(err, ErrOutsideProfile) {
		t.Fatalf("CanonicalForm(float64) error = %v", err)
	}
}

// Every control character is escaped as RFC 8785 says, and nothing else
// is. Mutant killed: appendString escaping 0x7f.
func TestEncode_StringEscapes(t *testing.T) {
	short := map[byte]string{0x08: `\b`, '\t': `\t`, '\n': `\n`, 0x0c: `\f`, '\r': `\r`}
	for c := range byte(0x20) {
		want, ok := short[c]
		if !ok {
			want = `\u00` + hex.EncodeToString([]byte{c})
		}
		if got := mustEncode(t, string([]byte{c})); string(got) != `"`+want+`"` {
			t.Fatalf("Encode(%#x) = %s, want %q", c, got, want)
		}
	}
	for _, s := range []string{"\x7f", "\u00e9", "\u2028\u2029", "\U0001f600", "/", "'"} {
		if got := mustEncode(t, s); string(got) != `"`+s+`"` {
			t.Fatalf("Encode(%q) = %s", s, got)
		}
	}
	if got := mustEncode(t, `a"b\c`); string(got) != `"a\"b\\c"` {
		t.Fatalf("Encode quote and backslash = %s", got)
	}
}

// Numbers are written as ECMAScript writes them; -0 is "0"; 1e21 is
// "1e+21". Mutant killed: numberText writing math.Trunc'd integers with
// FormatInt (1e21 would print as an integer).
func TestEncode_Numbers(t *testing.T) {
	for _, tc := range []struct {
		n    float64
		want string
	}{
		{0, "0"}, {math.Copysign(0, -1), "0"}, {1, "1"}, {-1, "-1"}, {0.5, "0.5"},
		{strictjson.MaxInteger, "9007199254740991"}, {-strictjson.MaxInteger, "-9007199254740991"},
		{1e21, "1e+21"}, {1.5e300, "1.5e+300"}, {1e-6, "0.000001"}, {1e-7, "1e-7"},
		{5e-324, "5e-324"}, {math.MaxFloat64, "1.7976931348623157e+308"}, {37.7749, "37.7749"},
		{1.23e21, "1.23e+21"}, {123456.789e-3, "123.456789"}, {0.30000000000000004, "0.30000000000000004"},
	} {
		if got := mustEncode(t, strictjson.Number(tc.n)); string(got) != tc.want {
			t.Fatalf("Encode(%v) = %s, want %s", tc.n, got, tc.want)
		}
	}
}

// Encode refuses every value no parse gives. Mutant killed: numberText
// without its refusal of an integer in J5's gap.
func TestEncode_RefusesWhatNoParseGives(t *testing.T) {
	for _, v := range []any{
		1.5, 1, int64(1), "\xff", map[string]any{"\xff": nil}, []string{"a"}, map[string]string{},
		struct{}{}, strictjson.Number(math.NaN()), strictjson.Number(math.Inf(1)),
		strictjson.Number(math.Inf(-1)), strictjson.Number(1 << 53), strictjson.Number(-(1 << 53)),
		strictjson.Number(1e20), []any{strictjson.Number(1e16)}, map[string]any{"a": []any{1.5}},
	} {
		if m, err := Encode(v); !errors.Is(err, ErrOutsideProfile) {
			t.Fatalf("Encode(%#v) = %q, %v; want ErrOutsideProfile", v, m, err)
		}
	}
}
