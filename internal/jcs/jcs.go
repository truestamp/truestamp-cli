// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: MIT

// Package jcs canonicalizes JSON per RFC 8785 with the one deviation the
// Truestamp wire format requires: an integer literal is emitted exactly as it
// was parsed, never round-tripped through an IEEE-754 double.
//
// RFC 8785 §3.2.2.3 defines every JSON number by parsing it into a double and
// re-serializing it per ECMA-262, which silently rounds 2^53 + 1 down to 2^53.
// The Truestamp producer emits integers at arbitrary precision instead, so
// reproducing a claims_hash or entropy_hash means matching the producer rather
// than the strict reading of the RFC. Appendix C.2a of the whitepaper pins
// both halves as normative vectors: 2^53 and 2^53 + 1 canonicalize to distinct
// strings with distinct 0x11 digests.
//
// The cost of that choice is that such a bundle is not portably verifiable by
// a strict RFC 8785 implementation, which Appendix E.4 requires a verifier to
// report rather than hide. Canonicalize therefore hands back the offending
// literals alongside the canonical bytes so callers can surface them.
//
// Everything else, UTF-16 code-unit key ordering, string escaping, ES6 float
// formatting, negative-zero normalization, is delegated unchanged to
// github.com/gowebpki/jcs, which this package wraps rather than replaces. For
// any valid input free of oversized integers the output is byte-identical to
// calling that library directly, which is what makes adopting this package
// incapable of moving an existing digest.
//
// The one thing the wrapper adds on top of the library is a strict RFC 8259
// gate, because the library is not a validating parser and silently rewrites
// several malformed documents into different valid ones. See rejectNonRFC8259.
package jcs

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/big"
	"sort"
	"strconv"
	"strings"

	gojcs "github.com/gowebpki/jcs"
)

// MaxExactInteger is 2^53. Every integer at or below it in absolute value is
// exactly representable as an IEEE-754 double; 2^53 + 1 is the first that is
// not. The comparison against it is strictly greater-than, so 2^53 itself is
// in range, matching Appendix C.2a, which labels it "exactly representable",
// and the reference verifier's @max_exact_integer.
const MaxExactInteger int64 = 1 << 53

// MaxSafeInteger is 2^53 - 1, the PRODUCER bound: RFC 8785 Appendix B's SHOULD
// ("values to be interpreted as true integers SHOULD be in the range
// -9007199254740991 to 9007199254740991") and JavaScript's
// Number.MAX_SAFE_INTEGER. [UnsafeIntegers] rejects strictly greater in
// magnitude, so 2^53 - 1 is accepted and 2^53 is not.
//
// It is deliberately ONE LESS than [MaxExactInteger], the verifier bound
// directly above. Do not unify them, and do not "fix" the off-by-one:
//
//   - The producer is strict. Emitting 2^53 is legal by the letter of the
//     round-trip argument but violates Appendix B's SHOULD, and the Truestamp
//     server enforces exactly this bound (Truestamp.SafeIntegers in
//     truestamp-v2), so a value this package accepts and the server rejects
//     would be a 422 the user cannot act on.
//   - The verifier is lenient. 2^53 itself round-trips through a double
//     exactly, so warning on it would raise a false alarm about a bundle every
//     conforming implementation can in fact check.
//
// Be strict in what you emit, lenient in what you accept. The one-value gap is
// the whole point; TestThresholdsDifferByOne fails loudly if it closes.
const MaxSafeInteger int64 = (1 << 53) - 1

// placeholderPrefix labels the splice tokens so an unexpected survivor in the
// output is recognizable rather than mistaken for user data.
const placeholderPrefix = "TSJCS"

// maxExactInteger is MaxExactInteger as a big.Int: the literals being compared
// routinely overflow every fixed-width Go integer type (an unsigned 64-bit
// Stellar sequence, a 400-digit nonce), so the comparison cannot go through
// strconv.ParseInt.
var maxExactInteger = big.NewInt(MaxExactInteger)

// maxSafeInteger is MaxSafeInteger as a big.Int, for the same reason.
var maxSafeInteger = big.NewInt(MaxSafeInteger)

// intSpan locates one integer literal within the input bytes.
type intSpan struct {
	start, end int
	lit        []byte
	val        *big.Int
}

// Canonicalize returns the RFC 8785 canonical form of data.
//
// data MUST be a single well-formed RFC 8259 document; anything else is an
// error rather than a best-effort canonicalization, because the underlying
// library rewrites several malformed documents into different valid ones and
// the caller cannot tell a digest of the input from a digest of the rewrite.
// See rejectNonRFC8259.
//
// oversized names every integer literal in data whose absolute value exceeds
// MaxExactInteger, one entry per occurrence, numerically ascending so
// oversized[0] is the smallest offender. It is nil whenever data stays inside
// the safe range, and in that case the canonical bytes are byte-identical to
// github.com/gowebpki/jcs.Transform.
func Canonicalize(data []byte) (canonical []byte, oversized []string, err error) {
	if strictErr := rejectNonRFC8259(data); strictErr != nil {
		return nil, nil, strictErr
	}

	spans, err := scanIntegers(data)
	if err != nil {
		// The document is well-formed JSON, so the only way the scan can fail
		// is the span-location invariant below, a bug in this package, never
		// user input. Surfacing it beats emitting a digest of bytes we could
		// not account for.
		return nil, nil, err
	}
	if len(spans) == 0 {
		// Nothing to preserve: hand the input to the library untouched, which
		// is what makes the output byte-identical to calling it directly.
		canonical, err = gojcs.Transform(data)
		return canonical, nil, err
	}

	// Swap each oversized literal for a unique JSON string before handing the
	// document to the library, then swap the literals back into the canonical
	// output. Sorting, escaping and float formatting therefore run on the
	// unmodified library code path; only the bytes it would have rounded are
	// ever touched.
	placeholder := placeholderFor(data)

	var spliced bytes.Buffer
	spliced.Grow(len(data))
	prev := 0
	for i, sp := range spans {
		spliced.Write(data[prev:sp.start])
		spliced.Write(placeholderToken(placeholder, i))
		prev = sp.end
	}
	spliced.Write(data[prev:])

	transformed, err := gojcs.Transform(spliced.Bytes())
	if err != nil {
		return nil, nil, err
	}

	canonical, err = restore(transformed, placeholder, spans)
	if err != nil {
		return nil, nil, err
	}

	return canonical, literalsAscending(spans), nil
}

// rejectNonRFC8259 reports why data is not a single well-formed JSON document,
// or nil when it is.
//
// github.com/gowebpki/jcs is not a validating parser: its scalar reader skips
// whitespace inside a token and tolerates a leading '+', a leading zero, a bare
// fraction and a trailing dot, so it accepts several inputs that RFC 8259,
// and every conforming parser, Go's encoding/json and the Elixir reference
// verifier included, rejects. Worse, it does not merely accept them, it
// rewrites them: {"a":1 2} canonicalizes to {"a":12} and {"a":.5} to {"a":0.5}.
// Canonicalize would then hand back the canonical form of a DIFFERENT document
// with no error, and `truestamp jcs` / `truestamp hash --jcs`, the documented
// way to recompute a claims_hash locally, would print a confident wrong digest
// for a truncated or corrupted claims file, with two byte-distinct inputs
// colliding on one hash.
//
// It also protects the Appendix E.4 portability report: the integer scanner
// runs on the same encoding/json decoder, so before this gate a document the
// scanner rejected produced no oversized-integer signal at all, adding a
// single '+' to an oversized literal both suppressed the warning and emitted
// the rounded digest.
//
// Validation is deliberately encoding/json's, not a hand-rolled one: it is the
// same scanner internal/proof already runs over a whole bundle, so the `verify`
// pipeline cannot reach a subject-data blob this rejects.
func rejectNonRFC8259(data []byte) error {
	if json.Valid(data) {
		return nil
	}

	// When gowebpki/jcs rejects the bytes too, its message is the one callers
	// already surface (empty input, a stray BOM, an unterminated object), so
	// keep it verbatim; only the documents it would have silently rewritten
	// get the new error.
	if _, libErr := gojcs.Transform(data); libErr != nil {
		return libErr
	}

	// json.Valid reports no detail, so re-run the same bytes through Unmarshal
	// purely to recover the offset-carrying *json.SyntaxError for the message.
	var probe json.RawMessage
	err := json.Unmarshal(data, &probe)
	if err == nil {
		// Unreachable: Valid and Unmarshal share one scanner.
		err = errors.New("not a single well-formed JSON document")
	}
	return fmt.Errorf("jcs: input is not valid JSON: %w", err)
}

// Transform is Canonicalize with the portability report discarded, for the
// call sites that do not surface it.
func Transform(data []byte) ([]byte, error) {
	canonical, _, err := Canonicalize(data)
	return canonical, err
}

// OversizedIntegers reports the integer literals in data that fall outside the
// exactly representable double range, for callers that want the Appendix E.4
// portability signal without paying for canonicalization. It has no error
// channel, so a document the scanner cannot read reports nothing; that is not a
// silent accept the way it would be in Canonicalize, because a caller reaching
// this entry point still has to canonicalize the same bytes to get a digest,
// and Canonicalize rejects them.
func OversizedIntegers(data []byte) []string {
	spans, err := scanIntegers(data)
	if err != nil || len(spans) == 0 {
		return nil
	}
	return literalsAscending(spans)
}

// scanIntegers records the byte span of every integer literal in data whose
// absolute value exceeds MaxExactInteger.
//
// Floats are never recorded: they are lossy by construction and the reference
// verifier never flags them, so a float keeps travelling the library's
// unmodified ES6 formatting path, including the "-0" to "0" normalization
// Appendix C.2a pins. That is enforced by the base-10 big.Int parse below,
// which is the single classifier, a literal carrying a fraction or an
// exponent cannot parse as a base-10 integer.
func scanIntegers(data []byte) ([]intSpan, error) {
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.UseNumber()

	var spans []intSpan
	for {
		tok, err := dec.Token()
		if errors.Is(err, io.EOF) {
			return spans, nil
		}
		if err != nil {
			return nil, err
		}

		num, ok := tok.(json.Number)
		if !ok {
			continue
		}
		lit := num.String()
		// 15 digits reach at most 10^15 - 1, comfortably inside 2^53, so the
		// overwhelmingly common case never allocates a big.Int. A float short
		// enough to land here is skipped for free.
		if len(strings.TrimPrefix(lit, "-")) <= 15 {
			continue
		}
		// SetString with an explicit base of 10 accepts only an optional sign
		// followed by digits, so it doubles as the integer/float classifier:
		// "1.5", "1e30" and "9007199254740993e0" all land here and are skipped.
		val, ok := new(big.Int).SetString(lit, 10)
		if !ok {
			continue
		}
		if new(big.Int).Abs(val).Cmp(maxExactInteger) <= 0 {
			continue
		}

		// InputOffset lands just past the token just returned, so the literal
		// occupies the len(lit) bytes ending there. That is verified rather
		// than assumed, because a wrong span would splice over neighbouring
		// bytes and corrupt the document.
		end := int(dec.InputOffset())
		start := end - len(lit)
		if start < 0 || !bytes.Equal(data[start:end], []byte(lit)) {
			return nil, fmt.Errorf("jcs: cannot locate integer literal %q ending at offset %d", lit, end)
		}
		spans = append(spans, intSpan{start: start, end: end, lit: []byte(lit), val: val})
	}
}

// placeholderFor derives a token that does not occur anywhere in data, so
// splicing it in and swapping it back out is unambiguous even when the
// document already contains strings that look like placeholders. Deriving it
// from the input rather than from randomness keeps Canonicalize deterministic;
// the loop covers the (practically unreachable) case of a document that
// carries its own digest.
func placeholderFor(data []byte) string {
	sum := sha256.Sum256(data)
	token := placeholderPrefix + hex.EncodeToString(sum[:])
	for bytes.Contains(data, []byte(token)) {
		next := sha256.Sum256([]byte(token))
		token = placeholderPrefix + hex.EncodeToString(next[:])
	}
	return token
}

// placeholderToken renders the i-th placeholder as the JSON string spliced
// into the input. Both the splice and the restore call it so the two can
// never drift out of step.
func placeholderToken(placeholder string, i int) []byte {
	return []byte(`"` + placeholder + "-" + strconv.Itoa(i) + `"`)
}

// restore swaps every placeholder in the canonicalized document back to the
// integer literal it stands for. Canonicalization reorders object members, so
// the placeholders come back in an arbitrary order and the index carried in
// each one is what pairs it with its literal.
//
// The walk is a single forward pass rather than one bytes.Replace per span:
// subject data can carry thousands of oversized integers, and rescanning the
// whole document once per literal would be quadratic in the size of an
// attacker-supplied bundle.
//
// Every anomaly below is an error, never a best-effort repair. A placeholder
// cannot occur in the input by construction, but a document whose *escaped*
// content unescapes to one would need to embed its own SHA-256 digest; if that
// preimage ever existed, the accounting here rejects it rather than emitting a
// silently wrong digest.
func restore(canonical []byte, placeholder string, spans []intSpan) ([]byte, error) {
	marker := []byte(`"` + placeholder + "-")
	restored := make([]byte, 0, len(canonical))
	filled := make([]bool, len(spans))

	rest := canonical
	for {
		at := bytes.Index(rest, marker)
		if at < 0 {
			break
		}
		restored = append(restored, rest[:at]...)
		rest = rest[at+len(marker):]

		end := bytes.IndexByte(rest, '"')
		if end < 0 {
			return nil, errors.New("jcs: unterminated placeholder in canonical output")
		}
		i, err := strconv.Atoi(string(rest[:end]))
		if err != nil || i < 0 || i >= len(spans) || filled[i] {
			return nil, fmt.Errorf("jcs: unexpected placeholder %q in canonical output", rest[:end])
		}
		filled[i] = true
		restored = append(restored, spans[i].lit...)
		rest = rest[end+1:]
	}
	restored = append(restored, rest...)

	for i, ok := range filled {
		if !ok {
			return nil, fmt.Errorf("jcs: placeholder %d missing from canonical output", i)
		}
	}
	if bytes.Contains(restored, []byte(placeholder)) {
		return nil, errors.New("jcs: placeholder survived canonicalization")
	}

	return restored, nil
}

// UnsafeInteger names one integer in a decoded document that a conforming
// producer must not emit, together with the dotted key path that locates it.
type UnsafeInteger struct {
	// Path is the dotted key path to the value, rooted at the label the
	// caller passed to [UnsafeIntegers]: object keys joined with ".", array
	// indices as "[i]", e.g. "claims.metadata.rows[0].id". The syntax
	// matches Truestamp.SafeIntegers on the server so the local and remote
	// rejections name the same location for the same value.
	Path string

	// Literal is the integer exactly as the user wrote it, never
	// round-tripped through a float. Rendering it anywhere, an error
	// message, a JSON field, MUST use this string rather than a numeric
	// type, or the report reproduces the very rounding it is warning about.
	Literal string
}

// UnsafeIntegers reports every integer in v whose magnitude exceeds
// [MaxSafeInteger], one entry per occurrence, so a user fixes them all in one
// pass instead of one 422 at a time. It returns nil when v is portable.
//
// root labels the top of each returned path; the Truestamp producer passes
// "claims" to match the server's field name.
//
// v is a document decoded with [encoding/json.Decoder.UseNumber], that is
// load-bearing, not incidental. Without it every number arrives as a float64
// with the offending literal already destroyed, and this walk would inspect
// the rounded value and report nothing. json.Number is therefore the only
// decoded form that can be judged; Go's own integer types are accepted too
// (nothing in the producer path builds claims that way today, but they marshal
// to exact integer literals, so leaving a hole there would be a latent
// regression). Floats in any form are deliberately never reported: the
// producer rule is about integer literals, exactly as the verifier guard is,
// and a large-magnitude value SPELLED as a float ("1e21", "1.5") is not an
// integer literal. That classification is delegated to the same base-10
// big.Int parse [scanIntegers] uses, so the two cannot drift apart.
//
// Ordering is deterministic, Go map iteration is randomized, so object keys
// are walked in sorted order, arrays in index order. Two runs over the same
// document produce the same slice in the same order, which is what lets a
// caller print the list and a test assert on it.
func UnsafeIntegers(root string, v any) []UnsafeInteger {
	var found []UnsafeInteger
	walkUnsafeIntegers(root, v, &found)
	return found
}

// walkUnsafeIntegers is the recursive half of [UnsafeIntegers], appending to
// found so the arbitrary-depth walk needs no per-level allocation.
func walkUnsafeIntegers(path string, v any, found *[]UnsafeInteger) {
	switch t := v.(type) {
	case map[string]any:
		keys := make([]string, 0, len(t))
		for k := range t {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			walkUnsafeIntegers(path+"."+k, t[k], found)
		}

	case []any:
		for i, elem := range t {
			walkUnsafeIntegers(path+"["+strconv.Itoa(i)+"]", elem, found)
		}

	case json.Number:
		lit := t.String()
		// 15 digits reach at most 10^15 - 1, comfortably inside 2^53 - 1
		// (16 digits), so the overwhelmingly common case never allocates a
		// big.Int and a short float is skipped for free.
		if len(strings.TrimPrefix(lit, "-")) <= 15 {
			return
		}
		// Base 10 accepts only an optional sign followed by digits, so it
		// doubles as the integer/float classifier: "1.5", "1e30" and
		// "9007199254740993e0" all fail here and are correctly skipped.
		val, ok := new(big.Int).SetString(lit, 10)
		if !ok {
			return
		}
		if new(big.Int).Abs(val).Cmp(maxSafeInteger) <= 0 {
			return
		}
		*found = append(*found, UnsafeInteger{Path: path, Literal: lit})

	case int:
		appendIfUnsafe(path, big.NewInt(int64(t)), found)
	case int64:
		appendIfUnsafe(path, big.NewInt(t), found)
	case uint64:
		appendIfUnsafe(path, new(big.Int).SetUint64(t), found)
	}
}

// appendIfUnsafe records n when it falls outside the safe range. The literal
// is rendered from the big.Int, which is exact for every fixed-width integer.
func appendIfUnsafe(path string, n *big.Int, found *[]UnsafeInteger) {
	if new(big.Int).Abs(n).Cmp(maxSafeInteger) <= 0 {
		return
	}
	*found = append(*found, UnsafeInteger{Path: path, Literal: n.String()})
}

// UnsafeIntegerMessage renders the submitter-facing explanation of one
// offending value.
//
// The wording is a deliberate byte-for-byte mirror of
// Truestamp.SafeIntegers.message/2 in truestamp-v2, which the server surfaces
// as an Ash InvalidAttribute on the claims field. A user who hits the local
// guard and a user who hits the server guard must read the same sentence, or
// the two halves of one rule look like two unrelated rules.
func UnsafeIntegerMessage(path, literal string) string {
	return fmt.Sprintf(
		"The integer %s at %s is outside the range %d to %d (+/- 2^53 - 1). "+
			"A value outside this range cannot be reproduced by a verifier that parses JSON "+
			"numbers as IEEE-754 doubles, which is most of them, so the resulting proof would "+
			"not be portably verifiable. Send the value as a string instead.",
		literal, path, -MaxSafeInteger, MaxSafeInteger)
}

// literalsAscending returns one entry per occurrence, numerically ascending,
// so a caller reporting "e.g. <n>" names the smallest offender the way the
// reference verifier does.
func literalsAscending(spans []intSpan) []string {
	ordered := make([]intSpan, len(spans))
	copy(ordered, spans)
	sort.SliceStable(ordered, func(i, j int) bool { return ordered[i].val.Cmp(ordered[j].val) < 0 })

	lits := make([]string, len(ordered))
	for i, sp := range ordered {
		lits[i] = string(sp.lit)
	}
	return lits
}
