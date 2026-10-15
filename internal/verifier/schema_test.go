// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: Apache-2.0

package verifier

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/truestamp/truestamp-cli/internal/specdata"
	"github.com/truestamp/truestamp-cli/internal/strictjson"
	"github.com/truestamp/truestamp-cli/internal/vectorcases"
)

// The checker of spec/schema.json (SPEC.md section 9), SHAPE's check 3:
// loading, which refuses a schema outside the keyword subset; the
// keywords' meaning on small schemas written here, where the vectors do not
// reach; patterns, which must mean in Go what ECMA-262 and SPEC.md section
// 2.4 say; and the spec's schema against the positive proofs.

// parseJSON parses text with the strict reader.
func parseJSON(t *testing.T, text string) any {
	t.Helper()
	value, err := strictjson.Parse([]byte(text))
	if err != nil {
		t.Fatalf("%s: %v", text, err)
	}
	return value
}

// loadSchema loads the schema text, failing the test on a refusal.
func loadSchema(t *testing.T, text string) *Schema {
	t.Helper()
	s, err := LoadSchema(parseJSON(t, text))
	if err != nil {
		t.Fatalf("%s: %v", text, err)
	}
	return s
}

// parts returns the parts of the failures of the value text against the
// schema text, each encoded, in the order Failures gives them.
func parts(t *testing.T, schema, value string) []string {
	t.Helper()
	var out []string
	for _, f := range loadSchema(t, schema).Failures(parseJSON(t, value)) {
		out = append(out, f.Part.Encode())
	}
	return out
}

// usedKeywords returns every keyword a schema object under node uses.
func usedKeywords(node any) []string {
	object, ok := node.(map[string]any)
	if !ok {
		return nil
	}
	var out []string
	for key, arg := range object {
		out = append(out, key)
		switch key {
		case "$defs", "properties":
			for _, sub := range arg.(map[string]any) {
				out = append(out, usedKeywords(sub)...)
			}
		case "allOf":
			for _, sub := range arg.([]any) {
				out = append(out, usedKeywords(sub)...)
			}
		case "if", "then", "else", "not", "items":
			out = append(out, usedKeywords(arg)...)
		}
	}
	return out
}

// specSchemaValue returns spec/schema.json, parsed.
func specSchemaValue(t *testing.T) map[string]any {
	t.Helper()
	data, err := specdata.File("schema.json")
	if err != nil {
		t.Fatal(err)
	}
	value, err := strictjson.Parse(data)
	if err != nil {
		t.Fatal(err)
	}
	return value.(map[string]any)
}

// nestings are the places where one schema holds another that applies to
// the same value or to an element: under items, inside an allOf element,
// under if, then (reached when if holds), else (reached when it does not)
// and not, under a member of properties, and as a member of the root's
// $defs.
func nestings(sub string) map[string]string {
	return map[string]string{
		"items":      `{"items": ` + sub + `}`,
		"allOf":      `{"allOf": [true, ` + sub + `]}`,
		"if":         `{"if": ` + sub + `}`,
		"then":       `{"if": true, "then": ` + sub + `}`,
		"else":       `{"if": false, "else": ` + sub + `}`,
		"not":        `{"not": ` + sub + `}`,
		"properties": `{"type": "object", "properties": {"a": ` + sub + `}}`,
		"$defs":      `{"$defs": {"A": ` + sub + `}}`,
	}
}

// The keywords are SPEC.md section 9's list, in its order, and
// spec/schema.json uses exactly those. Mutant killed: a keyword dropped
// from SchemaKeywords (the list differs); SchemaKeywords handing out its
// own slice (a caller's change shows in the next call); the spec schema's
// checker built by a loader that refuses one of them, such as maxItems
// (the package's initialization panics, so no test passes).
func TestSchema_TheKeywordsAreSection9s(t *testing.T) {
	// SPEC.md section 9, "The schema.", in its order.
	listed := []string{"$schema", "$defs", "$ref", "allOf", "if", "then", "else", "not", "type", "const",
		"enum", "properties", "required", "additionalProperties", "minProperties", "pattern", "minLength",
		"maxLength", "minimum", "maximum", "items", "maxItems"}
	if got := SchemaKeywords(); !slices.Equal(got, listed) {
		t.Fatalf("SchemaKeywords = %q", got)
	}
	SchemaKeywords()[0] = "changed"
	if got := SchemaKeywords(); !slices.Equal(got, listed) {
		t.Fatalf("after a caller's change, SchemaKeywords = %q", got)
	}
	used := usedKeywords(specSchemaValue(t))
	slices.Sort(used)
	sorted := slices.Clone(listed)
	slices.Sort(sorted)
	if got := slices.Compact(used); !slices.Equal(got, sorted) {
		t.Fatalf("spec/schema.json uses %q, want %q", got, sorted)
	}
	if proofSchema == nil || len(proofSchema.patterns) != 13 {
		t.Fatalf("the spec schema's patterns: %d", len(proofSchema.patterns))
	}
}

// A keyword outside the subset is refused when the schema loads, wherever
// it stands: at the root, under items, allOf, if, then, else, not, a
// member of properties and a member of $defs, and deep in the real schema.
// Mutants killed: walk not descending into else, into not, or into allOf
// (the keyword under it loads); form passing every keyword it does not name
// (every one loads).
func TestLoadSchema_RefusesAKeywordOutsideTheSubset(t *testing.T) {
	outside := []string{
		"format", "oneOf", "anyOf", "patternProperties", "dependentRequired", "dependentSchemas",
		"maxProperties", "minItems", "uniqueItems", "contains", "prefixItems", "propertyNames",
		"exclusiveMinimum", "exclusiveMaximum", "multipleOf", "unevaluatedProperties", "unevaluatedItems",
		"$id", "$dynamicRef", "$vocabulary", "$comment", "definitions",
		"default", "title", "description", "examples", "contentEncoding", "readOnly", "x-extension", "",
	}
	for _, keyword := range outside {
		sub := fmt.Sprintf(`{%q: 1}`, keyword)
		places := nestings(sub)
		places["the root"] = sub
		for place, text := range places {
			_, err := LoadSchema(parseJSON(t, text))
			if !errors.Is(err, ErrSchemaOutsideSubset) || !strings.Contains(err.Error(), fmt.Sprintf("%q, a keyword outside the subset", keyword)) {
				t.Errorf("%q under %s: %v", keyword, place, err)
			}
		}
	}
	schema := specSchemaValue(t)
	schema["$defs"].(map[string]any)["HEX64"].(map[string]any)["format"] = "hex"
	if _, err := LoadSchema(schema); err == nil || !strings.Contains(err.Error(), `"format", a keyword outside the subset`) {
		t.Fatalf("the spec schema with HEX64's format: %v", err)
	}
}

// A schema outside the subset's forms is refused when it loads, and a $ref
// that descends through properties or items is no cycle. Mutants killed:
// the $defs check made anywhere (a $defs below the root loads); the cycle
// check left out (the two cycles load); the cycle check following
// properties (the tree is refused); additionalProperties allowed as true;
// required's repeat check left out.
func TestLoadSchema_RefusesFormsOutsideTheSubset(t *testing.T) {
	refused := []string{
		`{"$ref": "#/definitions/A"}`,
		`{"$ref": "#/$defs/MISSING", "$defs": {"A": true}}`,
		`{"$ref": 7}`,
		`{"additionalProperties": true}`,
		`{"additionalProperties": {"type": "string"}}`,
		`{"pattern": "("}`,
		`{"pattern": 7}`,
		`{"type": "date"}`,
		`{"type": []}`,
		`{"type": ["string", 1]}`,
		`{"required": "a"}`,
		`{"required": ["a", "a"]}`,
		`{"required": [1]}`,
		`{"maxLength": -1}`,
		`{"maxItems": 1.5}`,
		`{"minProperties": "1"}`,
		`{"minimum": "0"}`,
		`{"enum": []}`,
		`{"enum": "a"}`,
		`{"allOf": []}`,
		`{"allOf": {}}`,
		`{"properties": []}`,
		`{"$schema": 1}`,
		`{"properties": {"a": {"$defs": {"B": true}}}}`,
		`{"properties": {"a": 1}}`,
		`{"items": "string"}`,
		`{"$defs": {"A/B": true}}`,
		`{"$defs": {"A~B": true}}`,
		`{"$defs": {"": true}}`,
		`{"$defs": []}`,
		`{"$ref": "#/$defs/A", "$defs": {"A": {"allOf": [{"$ref": "#/$defs/B"}]}, "B": {"not": {"$ref": "#/$defs/A"}}}}`,
		`{"$defs": {"A": {"if": {"$ref": "#/$defs/A"}}}}`,
		`1`,
		`"schema"`,
		`null`,
	}
	for _, text := range refused {
		if _, err := LoadSchema(parseJSON(t, text)); !errors.Is(err, ErrSchemaOutsideSubset) {
			t.Errorf("%s: %v", text, err)
		}
	}
	for _, sub := range []string{`{"$ref": "#/$defs/MISSING"}`, `{"pattern": "("}`, `{"pattern": "."}`} {
		for place, text := range nestings(sub) {
			if _, err := LoadSchema(parseJSON(t, text)); !errors.Is(err, ErrSchemaOutsideSubset) {
				t.Errorf("%s under %s: %v", sub, place, err)
			}
		}
	}
	tree := `{"$defs": {"N": {"properties": {"n": {"$ref": "#/$defs/N"}}, "items": {"$ref": "#/$defs/N"}}}, "$ref": "#/$defs/N"}`
	if got := parts(t, tree, `{"n": {"n": [{"n": 1}]}}`); len(got) != 0 {
		t.Fatalf("the tree: %q", got)
	}
	for _, text := range []string{`true`, `false`, `{}`, `{"$schema": "https://json-schema.org/draft/2020-12/schema"}`} {
		loadSchema(t, text)
	}
}

// The patterns this checker loads, each written in Go's syntax with ^ as
// \A and $ as \z, and those it refuses, whose meaning ECMA-262 and RE2
// differ on or one of them lacks. Mutants killed: translateECMA writing $
// as (?m:$) (the translations differ); the refusal of . left out;
// any escape allowed (\s, \w and the rest load); a group of any (? form
// allowed (the lookarounds load); a lone brace passed as a literal; [ allowed
// inside a class (the POSIX class loads); the empty class allowed; the
// refusal of a quantifier right after the leading ^ left out (^*a and
// ^{2}a load); the refusal of \d as a range's end left out ([\d-z],
// [\d--], [a-\d] and [--\d] load); the range state not reset when a
// class opens ([\d][-a] and [a][-\d] are refused). A - beside \d that ends
// no range, as the class's first or last character, right after a range or
// first in a later class, still loads.
func TestECMAPattern_TranslatesOrRefuses(t *testing.T) {
	translated := map[string]string{
		`^[0-9a-f]{64}$`:                        `\A[0-9a-f]{64}\z`,
		`^(?:[0-9a-f]{2})+$`:                    `\A(?:[0-9a-f]{2})+\z`,
		`^[\x20\x21\x23-\x5b\x5d-\x7e]{1,128}$`: `\A[\x20\x21\x23-\x5b\x5d-\x7e]{1,128}\z`,
		`^[A-Za-z0-9_-]{42}[AEIMQUYcgkosw048]$`: `\A[A-Za-z0-9_-]{42}[AEIMQUYcgkosw048]\z`,
		`^https://www\.truestamp\.com/\.well-known/constructions-[0-9a-f]{64}\.json$`: `\Ahttps://www\.truestamp\.com/\.well-known/constructions-[0-9a-f]{64}\.json\z`,
		`\n`:            `\n`,
		`b`:             `b`,
		`^a|b$`:         `\Aa|b\z`,
		`[$^]`:          `[$^]`,
		`[^a-z\-\]]`:    `[^a-z\-\]]`,
		`a{2,}?(b)*?\d`: `a{2,}?(b)*?\d`,
		`\^\$\.\/\{\}`:  `\^\$\.\/\{\}`,
		`[\d-]`:         `[\d-]`,
		`[-\d]`:         `[-\d]`,
		`[^-\d]`:        `[^-\d]`,
		`[a-c-\d]`:      `[a-c-\d]`,
		`^a*`:           `\Aa*`,
		`[\d][-a]`:      `[\d][-a]`,
		`[a][-\d]`:      `[a][-\d]`,
	}
	for pattern, want := range translated {
		got, err := translateECMA(pattern)
		if err != nil || got != want {
			t.Errorf("%s: %q, %v; want %q", pattern, got, err, want)
		}
		if _, err := ecmaPattern(pattern); err != nil {
			t.Errorf("%s does not compile: %v", pattern, err)
		}
	}
	refused := []string{
		`.`, `^a.$`, `\s`, `\S`, `\w`, `\W`, `\b`, `\B`, `\D`, `\p{L}`, `\pL`, `\P{L}`, `\A`, `\z`, `\Z`, `\Q.\E`,
		`\u0041`, `\x4`, `\x4g`, `\1`, `\0`, `\v`, `\f`, `\-`, `\`, `\cJ`, `\k<n>`,
		`(?=a)`, `(?!a)`, `(?<=a)`, `(?<!a)`, `(?i)a`, `(?i:a)`, `(?P<n>a)`, `(?<n>a)`, `(?`,
		`a{`, `a{,2}`, `a{x}`, `a{2`, `a}`, `]`, `[[:alpha:]]`, `[a[]`, `[]a]`, `[^]`, `[a`,
		`a^`, `$a`, `a$b`, `^^a`, `a$$`, `[\s]`, `[\w]`, `[\b]`,
		`^*a`, `^+a`, `^?a`, `^{2}a`, `^{2,}a`, `^{2,3}a`, `[\d-z]`, `[\d--]`, `[\d-\d]`, `[a\d-z]`, `[a-\d]`, `[--\d]`,
	}
	for _, pattern := range refused {
		if got, err := translateECMA(pattern); err == nil {
			t.Errorf("%s loads, as %q", pattern, got)
		}
	}
	if _, err := ecmaPattern(`(`); err == nil {
		t.Error("an unclosed group compiles")
	}
}

// A pattern is searched for anywhere in the string; its ^ and $ match only
// at the start and the end of the input, never before a final newline or
// after a first one; it reads code points; and the newline guard finds a
// newline anywhere. Mutants killed: $ written as Go's $ under the
// multiline flag (a\n matches ^a$); ^ written under the multiline flag
// (\na matches ^a$); MatchString replaced by a match of the whole string
// (b in abc fails).
func TestSchema_APatternsMeaning(t *testing.T) {
	cases := []struct {
		pattern, value string
		fails          bool
	}{
		{`^a$`, `"a"`, false},
		{`^a$`, `"a\n"`, true},
		{`^a$`, `"\na"`, true},
		{`^a$`, `"a\r"`, true},
		{`^a$`, `"a\u2028"`, true},
		{`^a$`, `" a"`, true},
		{`b`, `"abc"`, false},
		{`^b`, `"abc"`, true},
		{`b$`, `"abc"`, true},
		{`^a|c$`, `"ab"`, false},
		{`^a|c$`, `"bc"`, false},
		{`^a|c$`, `"bab"`, true},
		{`^[\x20-\x7e]{2}$`, `"é1"`, true},
		{`^[^a]{2}$`, `"é1"`, false},
		{`^[^a]{1}$`, `"\ud83d\ude00"`, false},
		{`^[0-9]$`, `"０"`, true},
		{`^[0-9]$`, `"٠"`, true},
		{`^\d$`, `"٠"`, true},
		{`^[a-f]$`, `"A"`, true},
	}
	for _, c := range cases {
		schema := fmt.Sprintf(`{"pattern": %q}`, c.pattern)
		if got := parts(t, schema, c.value); (len(got) == 1) != c.fails || len(got) > 1 {
			t.Errorf("%s against %s: %q, want a failure %v", c.pattern, c.value, got, c.fails)
		}
	}
	guard := `{"not": {"pattern": "\\n"}}`
	for value, fails := range map[string]bool{`"abc\n"`: true, `"a\nb"`: true, `"\n"`: true, `"abc"`: false, `"abc\r"`: false} {
		if got := parts(t, guard, value); (len(got) == 1) != fails {
			t.Errorf("the newline guard against %s: %q", value, got)
		}
	}
}

// A length counts code points, never bytes: two code points of four bytes
// pass a length of 2, and one code point of four bytes, a surrogate pair
// in the JSON text, is one. Mutant killed: lengths counted with len (é
// is two bytes).
func TestSchema_ALengthCountsCodePoints(t *testing.T) {
	schema := `{"type": "string", "minLength": 2, "maxLength": 2}`
	for value, ok := range map[string]bool{
		`"éé"`: true, `"é"`: true, `"éé"`: false,
		`"\ud83d\ude00"`: false, `"\ud83d\ude00\ud83d\ude00"`: true, `"ab"`: true, `"a"`: false,
	} {
		if got := parts(t, schema, value); (len(got) == 0) != ok {
			t.Errorf("%s: %q", value, got)
		}
	}
}

// A missing, extra or forbidden member fails at the member, a count at its
// object or array, each with its keyword and where it sits, in document
// order. Mutants killed: required failing at the object (the parts differ);
// additionalProperties left out (x passes); the false schema passing every
// value (b passes).
func TestSchema_TheFailuresParts(t *testing.T) {
	schema := loadSchema(t, `{"type": "object", "additionalProperties": false,
		"properties": {"a": {"type": "integer"}, "b": false, "c": {"maxItems": 1}},
		"required": ["a", "z"], "minProperties": 4}`)
	var got []string
	for _, f := range schema.Failures(parseJSON(t, `{"b": 1, "c": [1, 2], "x": true}`)) {
		got = append(got, fmt.Sprintf("%s %s %s", f.Part.Encode(), f.Keyword, f.Schema))
	}
	want := []string{
		" minProperties #/minProperties",
		"/a required #/required",
		"/b false #/properties/b",
		"/c maxItems #/properties/c/maxItems",
		"/x additionalProperties #/additionalProperties",
		"/z required #/required",
	}
	if !slices.Equal(got, want) {
		t.Fatalf("failures = %q, want %q", got, want)
	}
	nested := loadSchema(t, `{"allOf": [{"properties": {"a": true}}], "additionalProperties": false}`)
	if f := nested.Failures(parseJSON(t, `{"a": 1}`)); len(f) != 1 || f[0].Part.Encode() != "/a" {
		t.Fatalf("additionalProperties beside an allOf's properties: %v", f)
	}
}

// A failure's schema location is written from the $def last entered, with
// a name escaped as RFC 6901 writes it, and from the path a check starts
// at. Mutant killed: $ref keeping the location it was reached from (the
// location is #/items/$ref/...).
func TestSchema_AFailuresLocation(t *testing.T) {
	schema := loadSchema(t, `{"$defs": {"HEX": {"type": "string", "pattern": "^[0-9a-f]+$"}},
		"items": {"$ref": "#/$defs/HEX"}, "properties": {"a/b": {"properties": {"c~d": {"allOf": [false]}}}}}`)
	f := schema.Failures(parseJSON(t, `["ab", "AB"]`))
	if len(f) != 1 || f[0].Part.Encode() != "/1" || f[0].Keyword != "pattern" || f[0].Schema != "#/$defs/HEX/pattern" {
		t.Fatalf("failures = %+v", f)
	}
	f = schema.Failures(parseJSON(t, `{"a/b": {"c~d": 1}}`))
	if len(f) != 1 || f[0].Part.Encode() != "/a~1b/c~0d" || f[0].Schema != "#/properties/a~1b/properties/c~0d/allOf/0" {
		t.Fatalf("failures = %+v", f)
	}
	f = schema.Failures(parseJSON(t, `{"c~d": 1}`), "properties", "a/b")
	if len(f) != 1 || f[0].Part.Encode() != "/c~0d" || f[0].Schema != "#/properties/a~1b/properties/c~0d/allOf/0" {
		t.Fatalf("failures at a path = %+v", f)
	}
	defer func() {
		if recover() == nil {
			t.Fatal("a path that names no schema did not panic")
		}
	}()
	schema.Failures(parseJSON(t, `1`), "properties", "missing")
}

// Failures come in document order, a missing member in its place, indices
// by value, and of failures at one part in the order of their keywords'
// names, the same on every call. Mutants killed: the failures left unsorted
// (zeta's comes before path's); indices compared as text (10 before 2); a
// schema object's keywords applied in map order (the keywords at one part
// differ between calls).
func TestSchema_FailuresComeInDocumentOrder(t *testing.T) {
	schema := `{"properties": {"id": {"type": "string"}, "path": {"items": {"type": "string"}}, "zeta": {"type": "string"}},
		"required": ["leaf_count"]}`
	value := `{"zeta": 0, "path": ["a", "b", 1, "c", "d", "e", "f", "g", "h", "i", 2], "id": 1}`
	want := []string{"/id", "/leaf_count", "/path/2", "/path/10", "/zeta"}
	if got := parts(t, schema, value); !slices.Equal(got, want) {
		t.Fatalf("parts = %q, want %q", got, want)
	}
	many := loadSchema(t, `{"type": "string", "const": "a", "enum": ["b"], "not": true, "allOf": [false, false],
		"minimum": 1, "maxItems": 0, "minProperties": 1}`)
	var first []string
	for i := range 50 {
		var keywords []string
		for _, f := range many.Failures(parseJSON(t, `7`)) {
			keywords = append(keywords, f.Keyword+" "+f.Schema)
		}
		if i == 0 {
			first = keywords
		}
		if !slices.Equal(keywords, first) {
			t.Fatalf("call %d: %q, the first call %q", i, keywords, first)
		}
	}
	want = []string{"false #/allOf/0", "false #/allOf/1", "const #/const", "enum #/enum", "not #/not", "type #/type"}
	if !slices.Equal(first, want) {
		t.Fatalf("keywords at one part = %q, want %q", first, want)
	}
}

// if applies then or else, and fails nothing itself; not fails when its
// schema holds. Mutants killed: if applying then whatever its schema gives
// (n 1 with the empty root passes); not passing what its schema fails
// (every value fails not).
func TestSchema_IfThenElseAndNot(t *testing.T) {
	schema := `{"if": {"properties": {"n": {"const": 0}}, "required": ["n"]},
		"then": {"properties": {"root": {"const": "empty"}}},
		"else": {"properties": {"root": {"not": {"const": "empty"}}}}}`
	for value, want := range map[string][]string{
		`{"n": 0, "root": "empty"}`: nil,
		`{"n": 0, "root": "full"}`:  {"/root"},
		`{"n": 1, "root": "empty"}`: {"/root"},
		`{"n": 1, "root": "full"}`:  nil,
		`{"root": "empty"}`:         {"/root"},
	} {
		if got := parts(t, schema, value); !slices.Equal(got, want) {
			t.Errorf("%s: %q, want %q", value, got, want)
		}
	}
	if got := parts(t, `{"if": {"type": "string"}}`, `1`); len(got) != 0 {
		t.Errorf("an if with no branch: %q", got)
	}
}

// const and enum compare JSON values, numbers by value; integer is a
// number with no fraction; maximum and minimum are inclusive, at MAX_INTEGER
// (2^53 - 1) and 0; and each type is its own. Mutants killed: jsonEqual
// not comparing the objects' sizes (the object with b passes); integer
// taken as a Number Int64 accepts, at most MAX_INTEGER (1e21 fails
// integer); maximum made strict (MAX_INTEGER fails); null typed as an
// object.
func TestSchema_ConstEnumTypesAndBounds(t *testing.T) {
	cases := []struct {
		schema, value string
		ok            bool
	}{
		{`{"const": 1}`, `1.0`, true},
		{`{"const": 1}`, `1e0`, true},
		{`{"const": 1}`, `"1"`, false},
		{`{"const": 0}`, `-0`, true},
		{`{"enum": ["item", "block"]}`, `"Item"`, false},
		{`{"enum": ["item", "block"]}`, `"block"`, true},
		{`{"const": {"a": [1, 2]}}`, `{"a": [1.0, 2]}`, true},
		{`{"const": {"a": [1, 2]}}`, `{"a": [2, 1]}`, false},
		{`{"const": {"a": [1, 2]}}`, `{"a": [1, 2], "b": null}`, false},
		{`{"const": null}`, `null`, true},
		{`{"const": false}`, `0`, false},
		{`{"const": "e3b0"}`, `"e3b0\n"`, false},
		{`{"type": "integer"}`, `1e21`, true},
		{`{"type": "integer"}`, `2.5`, false},
		{`{"type": "integer"}`, `1.0`, true},
		{`{"type": "integer", "maximum": 9007199254740991}`, `1e21`, false},
		{`{"maximum": 9007199254740991}`, `9007199254740991`, true},
		{`{"maximum": 9007199254740991}`, `9007199254740990.75`, true},
		{`{"maximum": 9007199254740991}`, `1e21`, false},
		{`{"minimum": 0}`, `0`, true},
		{`{"type": "integer", "minimum": 0}`, `-1`, false},
		{`{"type": ["string", "null"]}`, `null`, true},
		{`{"type": "number"}`, `"1"`, false},
		{`{"type": "boolean"}`, `0`, false},
		{`{"type": "array"}`, `{}`, false},
		{`{"type": "object"}`, `[]`, false},
	}
	for _, c := range cases {
		if got := parts(t, c.schema, c.value); (len(got) == 0) != c.ok {
			t.Errorf("%s against %s: %q, want ok %v", c.schema, c.value, got, c.ok)
		}
	}
	for _, typ := range []string{"object", "array", "string", "integer", "number", "boolean"} {
		if got := parts(t, fmt.Sprintf(`{"type": %q}`, typ), `null`); len(got) != 1 {
			t.Errorf("null against %s: %q", typ, got)
		}
	}
}

// A keyword about another type of value says nothing about this one.
// Mutant killed: pattern applied to a value that is not a string (as a
// failure, since it cannot match).
func TestSchema_AKeywordOfAnotherTypeSaysNothing(t *testing.T) {
	schema := `{"pattern": "^a$", "minLength": 2, "maxLength": 0, "maxItems": 0, "minProperties": 3, "maximum": 0, "minimum": 9,
		"properties": {"a": false}, "required": ["a"], "additionalProperties": false, "items": false}`
	for value, want := range map[string]int{`5`: 2, `true`: 0, `null`: 0, `"a"`: 2, `[]`: 0, `{}`: 2, `[1]`: 2} {
		if got := parts(t, schema, value); len(got) != want {
			t.Errorf("%s: %q, want %d failures", value, got, want)
		}
	}
}

// positiveDocuments returns the 11 positive proofs, each parsed, by name.
func positiveDocuments(t *testing.T) map[string]map[string]any {
	t.Helper()
	paths, err := filepath.Glob(filepath.Join(vectorcases.Dir(), "positive", "*.proof.json"))
	if err != nil || len(paths) != 11 {
		t.Fatalf("positive proofs: %d, %v", len(paths), err)
	}
	out := map[string]map[string]any{}
	for _, path := range paths {
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		value, err := strictjson.Parse(data)
		if err != nil {
			t.Fatal(err)
		}
		out[strings.TrimSuffix(filepath.Base(path), ".proof.json")] = value.(map[string]any)
	}
	return out
}

// The spec schema passes every positive document, and P of each against
// the schema the document gives proof; a failure in a document's proof is
// at the same part of P without /proof. Mutant killed: ProofFailures
// checking P against the document's schema (every proof fails at its
// members).
func TestSpecSchema_PassesEveryPositiveProof(t *testing.T) {
	for name, document := range positiveDocuments(t) {
		if f := proofSchema.Failures(document); len(f) != 0 {
			t.Errorf("%s: %+v", name, f)
		}
		proof := document["proof"].(map[string]any)
		if f := ProofFailures(proof); len(f) != 0 {
			t.Errorf("%s: %+v", name, f)
		}
		proof["block"].(map[string]any)["record"].(map[string]any)["signing_key_id"] = strictjson.Number(7)
		whole := proofSchema.Failures(document)
		own := ProofFailures(proof)
		if len(whole) != 1 || len(own) != 1 || whole[0].Part.Encode() != "/proof/block/record/signing_key_id" ||
			own[0].Part.Encode() != "/block/record/signing_key_id" || whole[0].Schema != own[0].Schema {
			t.Errorf("%s: %+v, %+v", name, whole, own)
		}
	}
}

// Every pattern class of SPEC.md section 2.4, as the spec schema's $def of
// that name holds it, against a value of the class and against values that
// differ from it by a trailing or leading newline, a leading or trailing
// space, a Unicode look-alike of one character, and case: each class
// matches exactly what its row says, the whole string as \A...\z does. Its
// pattern alone, with no length or newline guard beside it, is held to the
// same, but where the class allows the change. Each def's pattern is the
// class's pattern of spec/encodings.json between ^ and $. Mutants killed:
// translateECMA writing $ as (?m:$), Go's $ under the multiline flag (each
// pattern alone matches a value with a trailing newline); the patterns
// compiled to fold case (each case variant of a hex, UUID7, TIME or NISTTIME
// value matches); ^ dropped, so the pattern is searched for anywhere (each
// pattern alone matches a value with a leading newline or space).
func TestSpecSchema_EveryPatternClass(t *testing.T) {
	type variant struct {
		value string
		// ok is whether the class holds it; patternOK whether the pattern
		// alone does.
		ok, patternOK bool
	}
	fail := func(value string) variant { return variant{value: value} }
	lookAlikes := func(value string, swaps map[rune]rune) []variant {
		var out []variant
		for from, to := range swaps {
			if i := strings.IndexRune(value, from); i >= 0 {
				out = append(out, fail(value[:i]+string(to)+value[i+len(string(from)):]))
			}
		}
		if len(out) == 0 {
			panic("no look-alike swaps in " + value)
		}
		return out
	}
	// Look-alikes of ASCII characters: fullwidth forms, Arabic-Indic and
	// mathematical digits, Cyrillic and Greek letters.
	digits := map[rune]rune{'0': '０', '1': '١', '7': '\U0001D7D5'}
	lower := map[rune]rune{'a': 'а', 'e': 'е', 'c': 'с'}
	upper := map[rune]rune{'A': 'А', 'E': 'Ε', 'B': 'Ｂ'}
	hex64 := strings.Repeat("0123456789abcdef", 4)
	b64u32 := strings.Repeat("A", 42) + "E"
	b64u64 := strings.Repeat("B", 85) + "A"
	nisthex := strings.Repeat("0123456789ABCDEF", 8)
	uuid7 := "01890a5d-ac96-774b-bcce-b302099a8057"
	classes := map[string]struct {
		value    string
		variants []variant
	}{
		"HEX64": {hex64, append(lookAlikes(hex64, mergeRunes(digits, lower)),
			fail(strings.ToUpper(hex64)), fail(hex64[:63]+"A"))},
		"KID": {"0a1b2c3e", append(lookAlikes("0a1b2c3e", mergeRunes(digits, lower)), fail("0A1B2C3E"))},
		"HEXBYTES": {"00ff1a", append(lookAlikes("00ff1a", mergeRunes(digits, lower)),
			fail("00FF1A"), fail("00ff1"), fail(""))},
		"B64U32": {b64u32, append(lookAlikes(b64u32, upper),
			fail(strings.Repeat("A", 42)+"e"), fail(strings.Repeat("a", 42)+"F"), fail(strings.Repeat("A", 42)+"+"),
			fail(strings.Repeat("A", 41)+"/E"), fail(b64u32+"="))},
		"B64U64": {b64u64, append(lookAlikes(b64u64, upper),
			fail(strings.Repeat("B", 85)+"a"), fail(strings.Repeat("B", 85)+"B"), fail(b64u64+"=="))},
		"TIME": {"2026-10-15T07:30:00Z", append(lookAlikes("2026-10-15T07:30:00Z", digits),
			fail("2026-10-15t07:30:00Z"), fail("2026-10-15T07:30:00z"), fail("2026-10-15 07:30:00Z"),
			fail("2026-10-15T07:30:00.000Z"), fail("2026-10-15T07:30:00+00:00"))},
		"NISTTIME": {"2026-10-15T07:30:00.000Z", append(lookAlikes("2026-10-15T07:30:00.000Z", digits),
			fail("2026-10-15t07:30:00.000Z"), fail("2026-10-15T07:30:00.000z"), fail("2026-10-15T07:30:00Z"),
			fail("2026-10-15T07:30:00,000Z"))},
		"NISTHEX": {nisthex, append(lookAlikes(nisthex, mergeRunes(digits, upper)), fail(strings.ToLower(nisthex)))},
		"UUID7": {uuid7, append(lookAlikes(uuid7, mergeRunes(digits, lower)),
			fail(strings.ToUpper(uuid7)), fail("01890a5d-ac96-474b-bcce-b302099a8057"),
			fail("01890a5d-ac96-774b-ccce-b302099a8057"), fail("01890a5dac96774bbcceb302099a8057"))},
		"ASCII128": {"key compromise reported", append(lookAlikes("key compromise reported", lower),
			variant{"KEY COMPROMISE REPORTED", true, true}, fail("tab\there"), fail(`quote"`), fail(`back\slash`),
			fail("no break"), fail("del\u007f"), fail(""), fail(strings.Repeat("r", 129)),
			variant{strings.Repeat("r", 128), true, true}, fail(strings.Repeat("r", 127)+"é"))},
		"ASCII16": {"2.0", append(lookAlikes("2.0", digits),
			variant{strings.Repeat("v", 16), true, true}, fail(strings.Repeat("v", 17)), fail("2.0　"))},
	}
	encodings := specdata.Encodings()
	defs := specSchemaValue(t)["$defs"].(map[string]any)
	tested := 0
	for name, class := range encodings {
		if class.Pattern == "" {
			continue
		}
		tested++
		c, ok := classes[name]
		if !ok {
			t.Errorf("no test values for the pattern class %s", name)
			continue
		}
		pattern := defs[name].(map[string]any)["pattern"].(string)
		if pattern != "^"+class.Pattern+"$" {
			t.Errorf("%s: the schema's pattern %q, the class's %q", name, pattern, class.Pattern)
		}
		alone := loadSchema(t, fmt.Sprintf(`{"pattern": %q}`, pattern))
		// A space is a character of the ASCII classes, so a value with one
		// more passes while it stays within the cap.
		spaces := (name == "ASCII128" && len(c.value) < 128) || (name == "ASCII16" && len(c.value) < 16)
		variants := append(slices.Clone(c.variants),
			variant{value: c.value, ok: true, patternOK: true},
			fail(c.value+"\n"), fail("\n"+c.value), fail(c.value+"\r\n"),
			variant{" " + c.value, spaces, spaces},
			variant{c.value + " ", spaces, spaces})
		for _, v := range variants {
			if v.ok && !v.patternOK {
				t.Fatalf("%s %q: a value the class holds and its pattern does not", name, v.value)
			}
			quoted := fmt.Sprintf("%q", v.value)
			value, err := strictjson.Parse(jsonString(v.value))
			if err != nil {
				t.Fatalf("%s: %v", quoted, err)
			}
			if got := proofSchema.Failures(value, "$defs", name); (len(got) == 0) != v.ok {
				t.Errorf("%s against %s: %+v, want ok %v", quoted, name, got, v.ok)
			}
			if got := alone.Failures(value); (len(got) == 0) != v.patternOK {
				t.Errorf("%s against %s's pattern alone: %+v, want ok %v", quoted, name, got, v.patternOK)
			}
		}
	}
	if tested != len(classes) {
		t.Fatalf("%d classes with a pattern, %d tested", tested, len(classes))
	}
	literals := map[string][]string{
		"ENV":             {"dev", "staging", "test"},
		"BITCOIN_NETWORK": {"mainnet", "testnet", "regtest"},
		"STELLAR_NETWORK": {"public", "testnet"},
	}
	for name, values := range literals {
		for _, v := range values {
			for _, other := range []string{v, v + "\n", "\n" + v, " " + v, v + " ", strings.ToUpper(v[:1]) + v[1:], strings.Replace(v, "e", "е", 1)} {
				value, err := strictjson.Parse(jsonString(other))
				if err != nil {
					t.Fatal(err)
				}
				if got := proofSchema.Failures(value, "$defs", name); (len(got) == 0) != (other == v) {
					t.Errorf("%q against %s: %+v", other, name, got)
				}
			}
		}
	}
}

func mergeRunes(maps ...map[rune]rune) map[rune]rune {
	out := map[rune]rune{}
	for _, m := range maps {
		for k, v := range m {
			out[k] = v
		}
	}
	return out
}

// jsonString writes s as a JSON string.
func jsonString(s string) []byte {
	var b strings.Builder
	b.WriteByte('"')
	for _, r := range s {
		switch {
		case r == '"' || r == '\\':
			b.WriteByte('\\')
			b.WriteRune(r)
		case r < 0x20:
			fmt.Fprintf(&b, `\u%04x`, r)
		default:
			b.WriteRune(r)
		}
	}
	b.WriteByte('"')
	return []byte(b.String())
}
