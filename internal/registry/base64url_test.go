// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: Apache-2.0

package registry

import (
	"encoding/base64"
	"encoding/hex"
	"errors"
	"path/filepath"
	"testing"

	"github.com/truestamp/truestamp-cli/internal/strictjson"
)

// The two classes' patterns and lengths are those of spec/encodings.json,
// read by the strict reader. Mutant killed: B64U64 compiled from B64U32's
// entry.
func TestB64U_PatternsEqualSpecEncodingsJSON(t *testing.T) {
	file := mustParseSpecFile(t, "encodings.json").(map[string]any)
	for class, name := range map[B64UClass]string{B64U32: "B64U32", B64U64: "B64U64"} {
		row, ok := file[name].(map[string]any)
		if !ok || len(row) != 2 {
			t.Fatalf("spec/encodings.json %s is %v, want an object of two members", name, file[name])
		}
		pattern, ok := B64UPattern(class)
		if !ok || pattern != row["pattern"] {
			t.Errorf("%s pattern %q, the file %v", name, pattern, row["pattern"])
		}
		octets, ok := B64UOctets(class)
		if !ok || strictjson.Number(octets) != row["decodes_to_octets"] {
			t.Errorf("%s decodes to %d octets, the file %v", name, octets, row["decodes_to_octets"])
		}
	}
	for _, class := range []B64UClass{0, B64U64 + 1} {
		if _, ok := B64UPattern(class); ok {
			t.Errorf("B64UPattern(%d) names a pattern", class)
		}
		if _, ok := B64UOctets(class); ok {
			t.Errorf("B64UOctets(%d) names a length", class)
		}
		if MatchesB64U(class, "") || MatchesB64U(class, "IVL40Zt5HSRFMkLhXy6rbLfP-ntqXtMAl5YOBpiB2xI") {
			t.Errorf("MatchesB64U(%d) matched", class)
		}
	}
}

// decodeClass decodes text under a class, as the corpus names it.
func decodeClass(t *testing.T, class, text string) ([]byte, error) {
	t.Helper()
	switch class {
	case "B64U32":
		out, err := DecodeB64U32(text)
		return out[:], err
	case "B64U64":
		out, err := DecodeB64U64(text)
		return out[:], err
	}
	t.Fatalf("no class %q", class)
	return nil, nil
}

func classOf(name string) B64UClass {
	if name == "B64U32" {
		return B64U32
	}
	return B64U64
}

// The known answers of vectors/corpus/base64url.json (SPEC.md section
// 13.2): each canonical text decodes to its octets, which encode back to
// it; each refused text matches no class and decodes to nothing, among
// them unused bits that are not zero, padding, the standard alphabet and
// an embedded CR LF, which Go's strict decoder skips; and each refused
// text's canonical spelling decodes. Mutants killed: decodeB64U decoding
// with RawURLEncoding.Strict() without matching the class first (the CR
// LF case decodes); the B64U32 pattern's last class widened to every
// base64url character (the unused bits case matches).
func TestB64U_Corpus(t *testing.T) {
	corpus := mustParseFile(t, filepath.Join(vectorsDir, "corpus", "base64url.json")).([]any)
	refusals := map[string]int{}
	accepted := 0
	for _, entry := range corpus {
		c := entry.(map[string]any)
		class, text := c["class"].(string), c["text"].(string)
		got, err := decodeClass(t, class, text)
		reason, refused := c["refused"].(string)
		if !refused {
			want, _ := hex.DecodeString(c["octets"].(string))
			if err != nil || hex.EncodeToString(got) != c["octets"] {
				t.Errorf("%s %q decodes to %x, %v, want %s", class, text, got, err, c["octets"])
			}
			if !MatchesB64U(classOf(class), text) || EncodeB64U(want) != text {
				t.Errorf("%s %q is not the canonical spelling of its octets", class, text)
			}
			accepted++
			continue
		}
		refusals[reason]++
		if !errors.Is(err, ErrNotCanonicalB64U) || MatchesB64U(classOf(class), text) {
			t.Errorf("%s %q (%s) was accepted: %x, %v", class, text, reason, got, err)
		}
		if canonical, ok := c["canonical"].(string); ok {
			octets, err := decodeClass(t, class, canonical)
			if err != nil || EncodeB64U(octets) != canonical {
				t.Errorf("%s: the canonical %q does not decode: %v", reason, canonical, err)
			}
		}
		if reason == "embedded_crlf" {
			if _, err := base64.RawURLEncoding.Strict().DecodeString(text); err != nil {
				t.Errorf("Go's strict decoder refused %q, so the case shows nothing", text)
			}
		}
	}
	if accepted != 8 || len(corpus) != 43 {
		t.Fatalf("%d entries, %d accepted, want 43 and 8", len(corpus), accepted)
	}
	for _, reason := range []string{"padded", "standard_alphabet", "unused_bits", "trailing_newline", "embedded_crlf", "leading_space", "too_short", "too_long", "other_character", "empty"} {
		if refusals[reason] == 0 {
			t.Errorf("no corpus entry is refused as %s", reason)
		}
	}
}

// The worked example of SPEC.md section 6.3: K decodes to A and V to S.
// Mutant killed: DecodeB64U32 decoding with base64.URLEncoding, which
// wants padding.
func TestB64U_WorkedExample(t *testing.T) {
	example := mustParseFile(t, filepath.Join(vectorsDir, "worked-example.json")).(map[string]any)
	key, err := DecodeB64U32(example["K"].(string))
	if err != nil || hex.EncodeToString(key[:]) != example["A"] {
		t.Errorf("K decodes to %x, %v, want A %v", key, err, example["A"])
	}
	signature, err := DecodeB64U64(example["V"].(string))
	if err != nil || hex.EncodeToString(signature[:]) != example["S"] {
		t.Errorf("V decodes to %x, %v, want S %v", signature, err, example["S"])
	}
	if EncodeB64U(key[:]) != example["K"] || EncodeB64U(signature[:]) != example["V"] {
		t.Error("A and S do not encode to K and V")
	}
	if _, err := DecodeB64U32(example["V"].(string)); !errors.Is(err, ErrNotCanonicalB64U) {
		t.Errorf("V decoded as a B64U32: %v", err)
	}
}
