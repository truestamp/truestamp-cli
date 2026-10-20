// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: Apache-2.0

// This file holds the verifier's own checker for spec/schema.json
// (SPEC.md section 9 (What a verifier pins), "The schema."), check 3 of
// SHAPE (section 10.2, step 7). It interprets a schema directly, keyword by
// keyword, as JSON Schema draft 2020-12 defines each keyword of the subset
// section 9 lists:
//
//	$schema, $defs, $ref (to #/$defs only), allOf, if, then, else, not,
//	type, const, enum, properties, required, additionalProperties (only
//	false), minProperties, pattern, minLength, maxLength, minimum, maximum,
//	items and maxItems, and the boolean schemas true and false.
//
// # Loading
//
// [LoadSchema] builds a checker from a schema, a value as
// internal/strictjson parses it, and refuses, naming what it found, any
// other keyword; $defs anywhere but the root, or with a member whose name a $ref
// could not name without RFC 6901 escapes; a $ref to anything but a member
// of the root's $defs; an additionalProperties other than false; a keyword
// whose argument is not of its form; an unknown type; a pattern this
// checker cannot give ECMA-262's meaning (below); and a cycle of $refs that
// applies a definition to the value it is already checking, which would
// never end. The checker of spec/schema.json is built when the package is
// initialized ([proofSchema]), so a schema outside the subset stops every
// program that links this package at start-up, and its tests.
//
// # What it gives
//
// Not only whether a value satisfies the schema, but every failure, each at
// a pointer, in document order ([Pointer]), since SHAPE reports the failure
// whose part comes first. Each [SchemaFailure] is {Part, Keyword, Schema}:
// Part a pointer into the value checked, Keyword the keyword that fails
// ("false" for the boolean schema), and Schema where that keyword sits, a
// pointer into the schema written from the $def the checker last entered,
// such as "#/$defs/HEX64/pattern". A failure's part is the one SPEC.md
// section 10.2's rule gives:
//
//   - a member required names and the value lacks: that member's pointer;
//   - a member additionalProperties: false does not allow: that member's
//     pointer;
//   - a member whose schema is false, forbidden for the type: that member's
//     pointer;
//   - minProperties or maxItems, a count: the object's or the array's;
//   - every other keyword: the pointer of the value that breaks it.
//
// # The keywords' meaning
//
// From JSON Schema: a keyword about objects, strings, numbers or arrays says
// nothing about a value of another type; if fails nothing itself and applies
// then when its schema holds, else else; not fails when its schema holds;
// properties applies each member's schema to the member when present;
// additionalProperties sees only the properties of its own schema object;
// const and enum compare JSON values, numbers by value; integer is a number
// with no fractional part; and lengths count code points, never bytes.
//
// A pattern is an ECMA-262 regular expression, searched for anywhere in the
// string, whose ^ and $, with no multiline flag, match only at the start and
// the end of the input. Go's regexp is RE2, whose syntax and meaning differ
// from ECMA-262's in places, so a pattern loads only when it is written in
// a subset whose meaning the two share, and is then compiled with its
// leading ^ written \A and its trailing $ written \z: it reads code points,
// and its $ never matches before a final newline ([ecmaPattern]).
// SPEC.md section 2.4 (Pattern classes) requires exactly that: each class
// matches the whole string, as \A...\z does.
//
// A checker is built from any schema at run time, so the refusals are
// testable, and it refuses the forms and the $ref cycles above whatever
// schema it is given, spec/schema.json included. Each pattern is compiled
// once, when the schema loads. A schema object's keywords are applied in
// the byte order of their names, so of two failures at one part the first
// is always the same one; the failures are then sorted, stably, into
// document order.

package verifier

import (
	"errors"
	"fmt"
	"math"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/truestamp/truestamp-cli/internal/specdata"
	"github.com/truestamp/truestamp-cli/internal/strictjson"
)

// schemaKeywords are the keywords a schema may use, in SPEC.md section 9's
// order.
var schemaKeywords = []string{
	"$schema", "$defs", "$ref", "allOf", "if", "then", "else", "not",
	"type", "const", "enum", "properties", "required", "additionalProperties",
	"minProperties", "pattern", "minLength", "maxLength", "minimum", "maximum",
	"items", "maxItems",
}

// SchemaKeywords returns the keywords a schema may use (SPEC.md section 9),
// in that section's order. Each call returns a new slice.
func SchemaKeywords() []string { return slices.Clone(schemaKeywords) }

// schemaTypes are the names the type keyword may give.
var schemaTypes = []string{"object", "array", "string", "integer", "number", "boolean", "null"}

// oneSchema are the keywords whose argument is one schema.
var oneSchema = []string{"if", "then", "else", "not", "items"}

// defsPrefix is how every $ref begins: it names a member of the root's
// $defs.
const defsPrefix = "#/$defs/"

// Schema is a checker built from a schema by [LoadSchema].
type Schema struct {
	root any
	defs map[string]any
	// patterns are the schema's patterns, compiled, by the text the schema
	// writes.
	patterns map[string]*regexp.Regexp
}

// SchemaFailure is one failure of a value against a schema: where it is,
// the keyword that fails and where that keyword sits in the schema.
type SchemaFailure struct {
	Part    Pointer
	Keyword string
	Schema  string
}

// ErrSchemaOutsideSubset is the error [LoadSchema] wraps when it refuses a
// schema.
var ErrSchemaOutsideSubset = errors.New("a schema outside SPEC.md section 9's subset")

// LoadSchema builds a checker of schema, a value as internal/strictjson
// parses it, or refuses a schema outside the subset with an error wrapping
// [ErrSchemaOutsideSubset] that names what it found.
func LoadSchema(schema any) (*Schema, error) {
	s := &Schema{root: schema, defs: map[string]any{}, patterns: map[string]*regexp.Regexp{}}
	if object, ok := schema.(map[string]any); ok {
		if defs, present := object["$defs"]; present {
			m, ok := defs.(map[string]any)
			if !ok {
				return nil, refuse("$defs that is not an object")
			}
			for _, name := range sortedNames(m) {
				if name == "" || strings.ContainsAny(name, "/~") {
					return nil, refuse("a $defs member named %q", name)
				}
			}
			s.defs = m
		}
	}
	if err := s.walk(schema, true); err != nil {
		return nil, err
	}
	if err := s.acyclic(); err != nil {
		return nil, err
	}
	return s, nil
}

func refuse(format string, args ...any) error {
	return fmt.Errorf("%w: "+format, append([]any{ErrSchemaOutsideSubset}, args...)...)
}

// walk checks every keyword of every schema object under node, each once,
// and compiles its patterns.
func (s *Schema) walk(node any, root bool) error {
	switch node := node.(type) {
	case bool:
		return nil
	case map[string]any:
		for _, key := range sortedNames(node) {
			arg := node[key]
			if err := s.form(key, arg, root); err != nil {
				return err
			}
			if err := s.children(key, arg); err != nil {
				return err
			}
		}
		return nil
	}
	return refuse("a schema that is neither an object nor a boolean")
}

// children walks the schemas a keyword's argument holds.
func (s *Schema) children(key string, arg any) error {
	var subs []any
	switch {
	case key == "$defs" || key == "properties":
		m := arg.(map[string]any)
		for _, name := range sortedNames(m) {
			subs = append(subs, m[name])
		}
	case key == "allOf":
		subs = arg.([]any)
	case slices.Contains(oneSchema, key):
		subs = []any{arg}
	}
	for _, sub := range subs {
		if err := s.walk(sub, false); err != nil {
			return err
		}
	}
	return nil
}

// form refuses a keyword outside the subset, or one whose argument is not
// of the form its keyword takes.
func (s *Schema) form(key string, arg any, root bool) error {
	holds := func(ok bool, what string) error {
		if ok {
			return nil
		}
		return refuse("%s with an argument of another form", what)
	}
	switch key {
	case "$schema":
		_, ok := arg.(string)
		return holds(ok, "$schema")
	case "$defs":
		if !root {
			return refuse("$defs below the root")
		}
		return nil
	case "$ref":
		ref, _ := arg.(string)
		name, ok := strings.CutPrefix(ref, defsPrefix)
		if !ok {
			return refuse("$ref outside #/$defs")
		}
		if _, ok := s.defs[name]; !ok {
			return refuse("$ref, naming no member of #/$defs, %q", ref)
		}
		return nil
	case "allOf":
		list, ok := arg.([]any)
		return holds(ok && len(list) > 0, "allOf")
	case "if", "then", "else", "not", "items", "const":
		return nil
	case "type":
		names := []any{arg}
		if list, ok := arg.([]any); ok {
			names = list
		}
		known := len(names) > 0
		for _, name := range names {
			text, ok := name.(string)
			known = known && ok && slices.Contains(schemaTypes, text)
		}
		if !known {
			return refuse("type, giving an unknown type")
		}
		return nil
	case "enum":
		list, ok := arg.([]any)
		return holds(ok && len(list) > 0, "enum")
	case "properties":
		_, ok := arg.(map[string]any)
		return holds(ok, "properties")
	case "required":
		list, ok := arg.([]any)
		seen := map[string]bool{}
		for _, name := range list {
			text, isString := name.(string)
			ok = ok && isString && !seen[text]
			seen[text] = true
		}
		return holds(ok, "required")
	case "additionalProperties":
		if arg != false {
			return refuse("additionalProperties other than false")
		}
		return nil
	case "minProperties", "minLength", "maxLength", "maxItems":
		_, ok := countOf(arg)
		return holds(ok, key)
	case "minimum", "maximum":
		_, ok := arg.(strictjson.Number)
		return holds(ok, key)
	case "pattern":
		text, ok := arg.(string)
		if !ok {
			return refuse("pattern that is not a string")
		}
		if _, done := s.patterns[text]; done {
			return nil
		}
		re, err := ecmaPattern(text)
		if err != nil {
			return refuse("pattern %q: %v", text, err)
		}
		s.patterns[text] = re
		return nil
	}
	return refuse("%q, a keyword outside the subset", key)
}

// countOf returns a count's argument as an int: a non-negative integer.
func countOf(arg any) (int, bool) {
	n, ok := arg.(strictjson.Number)
	if !ok {
		return 0, false
	}
	i, ok := n.Int64()
	return int(i), ok && i >= 0
}

// acyclic refuses a definition that reaches itself through $refs applied
// to the same value, never descending through properties or items, which
// would never end.
func (s *Schema) acyclic() error {
	edges := map[string][]string{}
	for name, node := range s.defs {
		edges[name] = sameValueRefs(node)
	}
	for _, name := range sortedNames(s.defs) {
		if reaches(edges, name, edges[name], map[string]bool{}) {
			return refuse("a cycle of $ref through %q on one value", name)
		}
	}
	return nil
}

func reaches(edges map[string][]string, target string, names []string, seen map[string]bool) bool {
	for len(names) > 0 {
		name := names[0]
		names = names[1:]
		switch {
		case name == target:
			return true
		case seen[name]:
			continue
		}
		seen[name] = true
		names = append(slices.Clone(edges[name]), names...)
	}
	return false
}

// sameValueRefs are the definitions node applies, by $ref, to the value it
// checks itself.
func sameValueRefs(node any) []string {
	object, ok := node.(map[string]any)
	if !ok {
		return nil
	}
	var refs []string
	for _, key := range sortedNames(object) {
		switch arg := object[key]; key {
		case "$ref":
			refs = append(refs, strings.TrimPrefix(arg.(string), defsPrefix))
		case "allOf":
			for _, sub := range arg.([]any) {
				refs = append(refs, sameValueRefs(sub)...)
			}
		case "if", "then", "else", "not":
			refs = append(refs, sameValueRefs(arg)...)
		}
	}
	return refs
}

// Failures returns every failure of value against the schema at path from
// the checker's root, each part a pointer into value, in document order.
// For the schema of the document's proof, path is "properties", "proof",
// and the failures are those of a document whose signature satisfies its
// schema, without the leading /proof. It panics for a path that names no
// schema, which only a caller's defect gives.
func (s *Schema) Failures(value any, path ...string) []SchemaFailure {
	schema := s.root
	for _, key := range path {
		object, ok := schema.(map[string]any)
		sub, present := object[key]
		if !ok || !present {
			panic(fmt.Sprintf("verifier: the schema has nothing at %q", key))
		}
		schema = sub
	}
	failures := s.check(schema, value, Pointer{}, append([]string{"#"}, path...))
	SortBy(failures, func(f SchemaFailure) Pointer { return f.Part })
	return failures
}

// check applies schema to value, at pointer at; loc is where schema sits,
// as its segments from "#", outermost first.
func (s *Schema) check(schema any, value any, at Pointer, loc []string) []SchemaFailure {
	switch schema := schema.(type) {
	case bool:
		if schema {
			return nil
		}
		return []SchemaFailure{failure(at, loc, "false")}
	case map[string]any:
		var out []SchemaFailure
		for _, key := range sortedNames(schema) {
			out = append(out, s.keyword(key, schema[key], schema, value, at, with(loc, key))...)
		}
		return out
	}
	panic("verifier: a loaded schema holds a schema that is neither an object nor a boolean")
}

// keyword applies one keyword; loc ends with the keyword.
func (s *Schema) keyword(key string, arg any, schema map[string]any, value any, at Pointer, loc []string) []SchemaFailure {
	object, isObject := value.(map[string]any)
	text, isString := value.(string)
	number, isNumber := value.(strictjson.Number)
	array, isArray := value.([]any)
	holds := func(ok bool) []SchemaFailure {
		if ok {
			return nil
		}
		return []SchemaFailure{failure(at, loc, key)}
	}
	switch key {
	case "$schema", "$defs", "then", "else":
		return nil
	case "$ref":
		name := strings.TrimPrefix(arg.(string), defsPrefix)
		return s.check(s.defs[name], value, at, []string{"#", "$defs", name})
	case "allOf":
		var out []SchemaFailure
		for i, sub := range arg.([]any) {
			out = append(out, s.check(sub, value, at, with(loc, strconv.Itoa(i)))...)
		}
		return out
	case "if":
		parent := loc[:len(loc)-1]
		branch := "else"
		if len(s.check(arg, value, at, parent)) == 0 {
			branch = "then"
		}
		sub, present := schema[branch]
		if !present {
			return nil
		}
		return s.check(sub, value, at, with(parent, branch))
	case "not":
		return holds(len(s.check(arg, value, at, loc)) > 0)
	case "type":
		names := []any{arg}
		if list, ok := arg.([]any); ok {
			names = list
		}
		return holds(slices.ContainsFunc(names, func(name any) bool { return typeHolds(name.(string), value) }))
	case "const":
		return holds(jsonEqual(arg, value))
	case "enum":
		return holds(slices.ContainsFunc(arg.([]any), func(x any) bool { return jsonEqual(x, value) }))
	case "properties":
		if !isObject {
			return nil
		}
		properties := arg.(map[string]any)
		var out []SchemaFailure
		for _, name := range sortedNames(properties) {
			if member, present := object[name]; present {
				out = append(out, s.check(properties[name], member, with(at, Name(name)), with(loc, name))...)
			}
		}
		return out
	case "required":
		if !isObject {
			return nil
		}
		var out []SchemaFailure
		for _, name := range arg.([]any) {
			if _, present := object[name.(string)]; !present {
				out = append(out, failure(with(at, Name(name.(string))), loc, key))
			}
		}
		return out
	case "additionalProperties":
		if !isObject {
			return nil
		}
		allowed, _ := schema["properties"].(map[string]any)
		var out []SchemaFailure
		for _, name := range sortedNames(object) {
			if _, listed := allowed[name]; !listed {
				out = append(out, failure(with(at, Name(name)), loc, key))
			}
		}
		return out
	case "minProperties":
		n, _ := countOf(arg)
		return when(isObject, func() bool { return len(object) >= n }, holds)
	case "pattern":
		re := s.patterns[arg.(string)]
		return when(isString, func() bool { return re.MatchString(text) }, holds)
	case "minLength":
		n, _ := countOf(arg)
		return when(isString, func() bool { return utf8.RuneCountInString(text) >= n }, holds)
	case "maxLength":
		n, _ := countOf(arg)
		return when(isString, func() bool { return utf8.RuneCountInString(text) <= n }, holds)
	case "minimum":
		return when(isNumber, func() bool { return number >= arg.(strictjson.Number) }, holds)
	case "maximum":
		return when(isNumber, func() bool { return number <= arg.(strictjson.Number) }, holds)
	case "items":
		if !isArray {
			return nil
		}
		var out []SchemaFailure
		for i, element := range array {
			out = append(out, s.check(arg, element, with(at, Index(i)), loc)...)
		}
		return out
	case "maxItems":
		n, _ := countOf(arg)
		return when(isArray, func() bool { return len(array) <= n }, holds)
	}
	panic(fmt.Sprintf("verifier: a loaded schema uses %q", key))
}

// when applies a keyword about one type of value: it says nothing about a
// value of another type (JSON Schema).
func when(applies bool, ok func() bool, holds func(bool) []SchemaFailure) []SchemaFailure {
	if !applies {
		return nil
	}
	return holds(ok())
}

// with returns list with one more element, never sharing list's array.
func with[T any](list []T, element T) []T {
	return append(slices.Clip(list), element)
}

func failure(at Pointer, loc []string, keyword string) SchemaFailure {
	var b strings.Builder
	b.WriteString(loc[0])
	for _, segment := range loc[1:] {
		b.WriteString(Pointer{Name(segment)}.Encode())
	}
	return SchemaFailure{Part: slices.Clone(at), Keyword: keyword, Schema: b.String()}
}

// typeHolds reports whether value is of the JSON type name.
func typeHolds(name string, value any) bool {
	switch name {
	case "object":
		_, ok := value.(map[string]any)
		return ok
	case "array":
		_, ok := value.([]any)
		return ok
	case "string":
		_, ok := value.(string)
		return ok
	case "integer":
		n, ok := value.(strictjson.Number)
		return ok && math.Floor(float64(n)) == float64(n)
	case "number":
		_, ok := value.(strictjson.Number)
		return ok
	case "boolean":
		_, ok := value.(bool)
		return ok
	case "null":
		return value == nil
	}
	return false
}

// jsonEqual is JSON equality: numbers by value, strings and literals
// exactly, objects and arrays member by member.
func jsonEqual(a, b any) bool {
	switch a := a.(type) {
	case map[string]any:
		other, ok := b.(map[string]any)
		if !ok || len(a) != len(other) {
			return false
		}
		for name, x := range a {
			y, present := other[name]
			if !present || !jsonEqual(x, y) {
				return false
			}
		}
		return true
	case []any:
		other, ok := b.([]any)
		return ok && slices.EqualFunc(a, other, jsonEqual)
	case strictjson.Number, string, bool, nil:
		return a == b
	}
	return false
}

// sortedNames returns an object's member names in byte order.
func sortedNames[V any](object map[string]V) []string {
	names := make([]string, 0, len(object))
	for name := range object {
		names = append(names, name)
	}
	slices.Sort(names)
	return names
}

// ecmaPattern compiles pattern, an ECMA-262 regular expression as JSON
// Schema reads one (Unicode mode, no flags), into a Go regexp that matches
// exactly the strings it matches, or refuses it. RE2 and ECMA-262 agree on
// the meaning of the constructs this accepts, and of nothing it refuses:
//
//   - a ^ only as the pattern's first character and a $ only as its last,
//     written \A and \z, which match only at the start and the end of the
//     input, as ECMA-262's do with no multiline flag (Go's own ^ and $ do
//     too, but \A and \z say so whatever the flags), and no quantifier
//     right after the ^, which ECMA-262 refuses as repeating nothing and
//     RE2 reads as repeating \A;
//   - literal characters, but not ., which ECMA-262 lets match neither CR
//     nor U+2028 nor U+2029, where RE2's matches both;
//   - the escapes \xHH, \n, \r, \t and \d, and a backslash before one of
//     ^ $ \ . * + ? ( ) [ ] { } | / (and, in a class, -), each meaning the
//     character itself; never \s, \w, \b, \p and the rest, whose classes
//     differ or which one of the two lacks;
//   - groups ( ) and (?: ), and alternation, but no other (? form, such as
//     a lookaround, which RE2 lacks, or a flag;
//   - the quantifiers *, +, ?, {n}, {n,} and {n,m}, each optionally lazy,
//     and no { or } that is not one, which RE2 reads as a literal brace and
//     Unicode-mode ECMA-262 refuses;
//   - classes [ ] and [^ ], of characters, ranges and the escapes above,
//     with no [ inside one (RE2 reads [: as a POSIX class), none empty
//     (RE2 reads []a] as a class holding ]), and no \d as either end of a
//     range, which ECMA-262 refuses and RE2, at the lower end, reads as \d,
//     a - and the upper end (a - beside \d stands for itself only as the
//     class's first or last character, or right after a range).
//
// Both read a string as code points: Go's regexp matches runes, and
// internal/strictjson gives only valid UTF-8 with no lone surrogate.
func ecmaPattern(pattern string) (*regexp.Regexp, error) {
	translated, err := translateECMA(pattern)
	if err != nil {
		return nil, err
	}
	return regexp.Compile(translated)
}

// translateECMA writes pattern in Go's syntax, or refuses it (see
// [ecmaPattern]).
func translateECMA(pattern string) (string, error) {
	runes := []rune(pattern)
	var b strings.Builder
	inClass := false
	// In a class: low holds after an atom that can begin a range (one that
	// ends none), lowIsDigit when that atom is \d, and inRange after a
	// range's -, until its upper end.
	low, lowIsDigit, inRange := false, false, false
	classAtom := func(digit bool) error {
		if inRange {
			if digit {
				return errors.New(`\d as the upper end of a range`)
			}
			low, lowIsDigit, inRange = false, false, false
			return nil
		}
		low, lowIsDigit = true, digit
		return nil
	}
	for i := 0; i < len(runes); i++ {
		c := runes[i]
		switch {
		case c == '\\':
			if i+1 >= len(runes) {
				return "", errors.New("it ends in a lone backslash")
			}
			escape, width, err := ecmaEscape(runes[i+1:], inClass)
			if err != nil {
				return "", err
			}
			if inClass {
				if err := classAtom(runes[i+1] == 'd'); err != nil {
					return "", err
				}
			}
			b.WriteString(escape)
			i += width
		case inClass:
			switch {
			case c == '[':
				return "", errors.New("a [ inside a class")
			case c == ']':
				inClass = false
			case c == '-' && low && i+1 < len(runes) && runes[i+1] != ']':
				if lowIsDigit {
					return "", errors.New(`\d as the lower end of a range`)
				}
				low, lowIsDigit, inRange = false, false, true
			default:
				if err := classAtom(false); err != nil {
					return "", err
				}
			}
			b.WriteRune(c)
		case c == '^':
			if i != 0 {
				return "", errors.New("a ^ that does not begin it")
			}
			if quantifierAt(runes[i+1:]) {
				return "", errors.New("a quantifier after the leading ^, which repeats nothing")
			}
			b.WriteString(`\A`)
		case c == '$':
			if i != len(runes)-1 {
				return "", errors.New("a $ that does not end it")
			}
			b.WriteString(`\z`)
		case c == '.':
			return "", errors.New("a ., which ECMA-262 and RE2 read differently")
		case c == '[':
			inClass = true
			low, lowIsDigit, inRange = false, false, false
			b.WriteRune(c)
			if i+1 < len(runes) && runes[i+1] == '^' {
				b.WriteRune('^')
				i++
			}
			if i+1 < len(runes) && runes[i+1] == ']' {
				return "", errors.New("an empty class")
			}
		case c == '(':
			b.WriteRune(c)
			if i+1 < len(runes) && runes[i+1] == '?' {
				if i+2 >= len(runes) || runes[i+2] != ':' {
					return "", errors.New("a group of a form other than ( ) and (?: )")
				}
				b.WriteString("?:")
				i += 2
			}
		case c == '{':
			width, ok := ecmaBraces(runes[i:])
			if !ok {
				return "", errors.New("a { that begins no quantifier")
			}
			b.WriteString(string(runes[i : i+width]))
			i += width - 1
		case c == '}' || c == ']':
			return "", fmt.Errorf("a lone %c", c)
		default:
			b.WriteRune(c)
		}
	}
	if inClass {
		return "", errors.New("a class that does not close")
	}
	return b.String(), nil
}

// ecmaEscape reads the escape after a backslash, rest beginning with the
// character after it: what it writes in Go's syntax, how many characters
// of rest it takes, or a refusal.
func ecmaEscape(rest []rune, inClass bool) (string, int, error) {
	switch c := rest[0]; {
	case c == 'x':
		if len(rest) < 3 || !isHexDigit(rest[1]) || !isHexDigit(rest[2]) {
			return "", 0, errors.New(`a \x not followed by two hex digits`)
		}
		return `\x` + string(rest[1:3]), 3, nil
	case c == 'n' || c == 'r' || c == 't' || c == 'd':
		return `\` + string(c), 1, nil
	case strings.ContainsRune(`^$\.*+?()[]{}|/`, c) || (inClass && c == '-'):
		return `\` + string(c), 1, nil
	default:
		return "", 0, fmt.Errorf(`the escape \%c, which ECMA-262 and RE2 read differently or one of them lacks`, c)
	}
}

func isHexDigit(c rune) bool {
	return ('0' <= c && c <= '9') || ('a' <= c && c <= 'f') || ('A' <= c && c <= 'F')
}

// quantifierAt reports whether rest begins with a quantifier: *, +, ?, or
// a {n}, {n,} or {n,m}.
func quantifierAt(rest []rune) bool {
	if len(rest) == 0 {
		return false
	}
	switch rest[0] {
	case '*', '+', '?':
		return true
	case '{':
		_, ok := ecmaBraces(rest)
		return ok
	}
	return false
}

// ecmaBraces returns the width of the quantifier {n}, {n,} or {n,m} that
// begins rest, and false when rest begins none.
func ecmaBraces(rest []rune) (int, bool) {
	digits := func(from int) int {
		to := from
		for to < len(rest) && '0' <= rest[to] && rest[to] <= '9' {
			to++
		}
		return to
	}
	i := digits(1)
	if i == 1 || i >= len(rest) {
		return 0, false
	}
	if rest[i] == ',' {
		i = digits(i + 1)
		if i >= len(rest) {
			return 0, false
		}
	}
	if rest[i] != '}' {
		return 0, false
	}
	return i + 1, true
}

// proofSchema is the checker of spec/schema.json, built when the package is
// initialized: a schema outside the subset, or one that gives the
// document's proof member no schema, stops the program.
var proofSchema = mustLoadSpecSchema()

func mustLoadSpecSchema() *Schema {
	data, err := specdata.File("schema.json")
	if err != nil {
		panic(fmt.Sprintf("verifier: spec/schema.json: %v", err))
	}
	value, err := strictjson.Parse(data)
	if err != nil {
		panic(fmt.Sprintf("verifier: spec/schema.json: %v", err))
	}
	schema, err := LoadSchema(value)
	if err != nil {
		panic(fmt.Sprintf("verifier: spec/schema.json: %v", err))
	}
	properties, _ := value.(map[string]any)["properties"].(map[string]any)
	if _, ok := properties["proof"]; !ok {
		panic("verifier: spec/schema.json gives the document's proof member no schema")
	}
	return schema
}

// ProofFailures returns every failure of P, the parsed value of proof,
// against the schema spec/schema.json gives the document's proof member,
// each part a pointer into P, in document order.
func ProofFailures(proof any) []SchemaFailure {
	return proofSchema.Failures(proof, "properties", "proof")
}
