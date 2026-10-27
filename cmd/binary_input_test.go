// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"go/ast"
	"go/parser"
	"go/token"
	"io"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

// rfc8949 encodes the JSON text doc in the binary encoding of RFC 8949
// (section 4.2's preferred forms: definite lengths, shortest integer
// heads, keys sorted by their encoded bytes), so that a test can hand the
// commands a proof in that encoding without a fixture or a dependency.
// tagged prepends the self-describing tag 55799 (d9 d9 f7).
func rfc8949(t *testing.T, doc []byte, tagged bool) []byte {
	t.Helper()
	dec := json.NewDecoder(bytes.NewReader(doc))
	dec.UseNumber()
	var v any
	if err := dec.Decode(&v); err != nil {
		t.Fatalf("decoding the document: %v", err)
	}
	var out []byte
	if tagged {
		out = []byte{0xd9, 0xd9, 0xf7}
	}
	return append(out, rfc8949Value(t, v)...)
}

func rfc8949Head(major byte, n uint64) []byte {
	switch {
	case n < 24:
		return []byte{major<<5 | byte(n)}
	case n <= math.MaxUint8:
		return []byte{major<<5 | 24, byte(n)}
	case n <= math.MaxUint16:
		return binary.BigEndian.AppendUint16([]byte{major<<5 | 25}, uint16(n))
	case n <= math.MaxUint32:
		return binary.BigEndian.AppendUint32([]byte{major<<5 | 26}, uint32(n))
	default:
		return binary.BigEndian.AppendUint64([]byte{major<<5 | 27}, n)
	}
}

func rfc8949Value(t *testing.T, v any) []byte {
	t.Helper()
	switch x := v.(type) {
	case nil:
		return []byte{0xf6}
	case bool:
		if x {
			return []byte{0xf5}
		}
		return []byte{0xf4}
	case string:
		return append(rfc8949Head(3, uint64(len(x))), x...)
	case json.Number:
		if i, err := x.Int64(); err == nil {
			if i < 0 {
				return rfc8949Head(1, uint64(-1-i))
			}
			return rfc8949Head(0, uint64(i))
		}
		f, err := x.Float64()
		if err != nil {
			t.Fatalf("number %s: %v", x, err)
		}
		return binary.BigEndian.AppendUint64([]byte{0xfb}, math.Float64bits(f))
	case []any:
		out := rfc8949Head(4, uint64(len(x)))
		for _, e := range x {
			out = append(out, rfc8949Value(t, e)...)
		}
		return out
	case map[string]any:
		type member struct{ key, value []byte }
		members := make([]member, 0, len(x))
		for k, e := range x {
			members = append(members, member{rfc8949Value(t, k), rfc8949Value(t, e)})
		}
		slices.SortFunc(members, func(a, b member) int { return bytes.Compare(a.key, b.key) })
		out := rfc8949Head(5, uint64(len(x)))
		for _, m := range members {
			out = append(append(out, m.key...), m.value...)
		}
		return out
	}
	t.Fatalf("no encoding for %T", v)
	return nil
}

// TestRFC8949_KnownAnswers pins the encoder these tests build their input
// with against RFC 8949 Appendix A's examples, so that the refusals below
// are refusals of that encoding and not of some other bytes. Mutant
// killed: the one-octet head taken up to 24 (n <= 24), which writes 24 as
// 18 instead of 1818.
func TestRFC8949_KnownAnswers(t *testing.T) {
	for _, c := range []struct {
		doc  string
		want string
	}{
		{`0`, "00"},
		{`23`, "17"},
		{`24`, "1818"},
		{`1000000`, "1a000f4240"},
		{`-1000`, "3903e7"},
		{`1.1`, "fb3ff199999999999a"},
		{`null`, "f6"},
		{`true`, "f5"},
		{`"IETF"`, "6449455446"},
		{`[1,[2,3]]`, "8201820203"},
		{`{"b":[2,3],"a":1}`, "a26161016162820203"},
	} {
		if got := hexOf(rfc8949(t, []byte(c.doc), false)); got != c.want {
			t.Errorf("%s: %s, want %s", c.doc, got, c.want)
		}
	}
	if got := hexOf(rfc8949(t, []byte(`{}`), true)); got != "d9d9f7a0" {
		t.Errorf("tagged {}: %s, want d9d9f7a0", got)
	}
}

func hexOf(b []byte) string {
	const digits = "0123456789abcdef"
	out := make([]byte, 0, 2*len(b))
	for _, c := range b {
		out = append(out, digits[c>>4], digits[c&15])
	}
	return string(out)
}

// TestCLI_Verify_RefusesABinaryEncodedProof: verify reads one JSON
// document (SPEC.md section 10.1), so the vectors' sound item proof in
// RFC 8949's binary encoding, bare or behind tag 55799, from a file or on
// stdin, is the bare rejection malformed_json, sub_reason invalid_utf8,
// exit 4 (SPEC.md section 12.2), and is never decoded. Mutant killed: the
// read detecting that encoding by its first octet and decoding it into
// the JSON value space before the strict reader (the decoded proof gives
// a supplied-keyring report, exit 6).
func TestCLI_Verify_RefusesABinaryEncodedProof(t *testing.T) {
	doc, err := os.ReadFile(soundItem())
	if err != nil {
		t.Fatal(err)
	}
	if _, _, code := verifyCLI(t, binaryPath, nil, nil, append([]string{soundItem(), "--json"}, underTest()...)...); code != 6 {
		t.Fatalf("the JSON proof itself: exit %d, want 6 (supplied-keyring)", code)
	}
	for _, tagged := range []bool{false, true} {
		encoded := rfc8949(t, doc, tagged)
		file := filepath.Join(t.TempDir(), "proof.bin")
		if err := os.WriteFile(file, encoded, 0o600); err != nil {
			t.Fatal(err)
		}
		for _, c := range []struct {
			name  string
			args  []string
			stdin []byte
		}{
			{"a file", []string{file}, nil},
			{"stdin", []string{"-"}, encoded},
		} {
			var stdin io.Reader
			if c.stdin != nil {
				stdin = bytes.NewReader(c.stdin)
			}
			stdout, stderr, code := verifyCLI(t, binaryPath, nil, stdin, append(c.args, append(underTest(), "--json")...)...)
			var rejection map[string]any
			if err := json.Unmarshal([]byte(stdout), &rejection); err != nil || code != 4 ||
				rejection["code"] != "malformed_json" || rejection["sub_reason"] != "invalid_utf8" {
				t.Errorf("tagged=%v, %s: exit %d, %v, stdout %s, stderr %s; want exit 4, malformed_json invalid_utf8",
					tagged, c.name, code, err, stdout, stderr)
			}
		}
	}
}

// TestCLI_Inspect_RefusesABinaryEncodedProof: inspect reads one JSON
// document through the verifier's strict reader (SPEC.md section 10.1), so
// the vectors' sound item proof, which it prints from JSON, given in RFC
// 8949's binary encoding, bare or behind tag 55799, from a file or on
// stdin, is the verifier's bare rejection malformed_json, sub_reason
// invalid_utf8, at PARSE, exit 1, in text and in --json alike, and is
// never decoded. Mutants killed: every bare rejection given the step
// ENVELOPE (rejectionStep); the sub_reason left out of the rejection.
func TestCLI_Inspect_RefusesABinaryEncodedProof(t *testing.T) {
	doc, err := os.ReadFile(soundItem())
	if err != nil {
		t.Fatal(err)
	}
	if out, code := runCLIText(t, "inspect", soundItem()); code != 0 {
		t.Fatalf("the JSON proof itself: exit %d\n%s", code, out)
	}
	run := func(stdin []byte, args ...string) (string, int) {
		cmd := exec.Command(binaryPath, append([]string{"inspect"}, args...)...)
		cmd.Env = cleanEnv()
		if stdin != nil {
			cmd.Stdin = bytes.NewReader(stdin)
		}
		out, _ := cmd.CombinedOutput()
		return stripANSI(string(out)), cmd.ProcessState.ExitCode()
	}
	for _, tagged := range []bool{false, true} {
		encoded := rfc8949(t, doc, tagged)
		file := filepath.Join(t.TempDir(), "proof.bin")
		if err := os.WriteFile(file, encoded, 0o600); err != nil {
			t.Fatal(err)
		}
		for _, c := range []struct {
			name  string
			args  []string
			stdin []byte
		}{
			{"a file", []string{file}, nil},
			{"stdin", nil, encoded},
		} {
			out, code := run(c.stdin, c.args...)
			if code != 1 || !strings.Contains(out, "Refused                  malformed_json (PARSE, invalid_utf8)\n") {
				t.Errorf("tagged=%v, %s: exit %d, want 1 and the refusal malformed_json (PARSE, invalid_utf8)\n%s", tagged, c.name, code, out)
			}
			out, code = run(c.stdin, append(c.args, "--json")...)
			var got InspectOutput
			if err := json.Unmarshal([]byte(out), &got); err != nil || code != 1 || got.Rejection == nil ||
				got.Rejection.Code != "malformed_json" || got.Rejection.SubReason != "invalid_utf8" || got.Rejection.Step != "PARSE" ||
				got.Rejection.Part != nil || got.Proof != nil || got.Signature != nil {
				t.Errorf("tagged=%v, %s --json: exit %d, %v, want 1 and the rejection malformed_json invalid_utf8\n%s", tagged, c.name, code, err, out)
			}
		}
	}
}

// TestProofPicker_OffersOnlyJSON: the interactive picker of verify and
// inspect offers exactly .json, and both commands take that list from
// proofPickerExts rather than from a literal of their own. Mutants killed:
// the RFC 8949 file extension added to proofPickerExts; either command's
// PickerExts written as its own literal again.
func TestProofPicker_OffersOnlyJSON(t *testing.T) {
	if !slices.Equal(proofPickerExts, []string{".json"}) {
		t.Errorf("proofPickerExts = %q, want [\".json\"]", proofPickerExts)
	}
	for _, file := range []string{"verify.go", "inspect.go"} {
		f, err := parser.ParseFile(token.NewFileSet(), file, nil, 0)
		if err != nil {
			t.Fatal(err)
		}
		var values []string
		ast.Inspect(f, func(n ast.Node) bool {
			kv, ok := n.(*ast.KeyValueExpr)
			if !ok {
				return true
			}
			if key, ok := kv.Key.(*ast.Ident); ok && key.Name == "PickerExts" {
				if id, ok := kv.Value.(*ast.Ident); ok {
					values = append(values, id.Name)
				} else {
					values = append(values, "a literal")
				}
			}
			return true
		})
		if !slices.Equal(values, []string{"proofPickerExts"}) {
			t.Errorf("%s sets PickerExts from %q, want exactly once from proofPickerExts", file, values)
		}
	}
}
