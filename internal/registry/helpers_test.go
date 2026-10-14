// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: Apache-2.0

package registry

import (
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"testing"

	"github.com/truestamp/truestamp-cli/internal/specdata"
	"github.com/truestamp/truestamp-cli/internal/strictjson"
)

// repoRoot is the module root, from this package's directory.
var repoRoot = filepath.Join("..", "..")

// vectorsDir is the copy of the spec's vectors/.
var vectorsDir = filepath.Join(repoRoot, "testdata", "verification", "vectors")

// mustParseFile reads a file of this repository with the strict reader.
func mustParseFile(t *testing.T, name string) any {
	t.Helper()
	data, err := os.ReadFile(name)
	if err != nil {
		t.Fatal(err)
	}
	return mustParse(t, name, data)
}

// mustParseSpecFile reads an embedded file under spec/ with the strict
// reader.
func mustParseSpecFile(t *testing.T, name string) any {
	t.Helper()
	data, err := specdata.File(name)
	if err != nil {
		t.Fatal(err)
	}
	return mustParse(t, "spec/"+name, data)
}

func mustParse(t *testing.T, name string, data []byte) any {
	t.Helper()
	value, err := strictjson.Parse(data)
	if err != nil {
		t.Fatalf("%s: %v", name, err)
	}
	return value
}

// pointer resolves the JSON Pointer at (RFC 6901) in value.
func pointer(t *testing.T, value any, at string) any {
	t.Helper()
	if at == "" {
		return value
	}
	if !strings.HasPrefix(at, "/") {
		t.Fatalf("pointer %q does not begin with /", at)
	}
	node := value
	for _, segment := range strings.Split(at[1:], "/") {
		segment = strings.ReplaceAll(strings.ReplaceAll(segment, "~1", "/"), "~0", "~")
		switch typed := node.(type) {
		case map[string]any:
			next, ok := typed[segment]
			if !ok {
				t.Fatalf("pointer %q: no member %q", at, segment)
			}
			node = next
		case []any:
			i, err := strconv.Atoi(segment)
			if err != nil || i < 0 || i >= len(typed) {
				t.Fatalf("pointer %q: no element %q", at, segment)
			}
			node = typed[i]
		default:
			t.Fatalf("pointer %q: %q is below a scalar", at, segment)
		}
	}
	return node
}

// soundProof is one of the positive proofs of the vectors (SPEC.md section
// 13.2), with its trace.
type soundProof struct {
	name     string
	document map[string]any
	proof    map[string]any
	trace    []map[string]any
}

// soundProofs reads vectors/positive/*.proof.json and their traces, sorted
// by name. There are 11.
func soundProofs(t *testing.T) []soundProof {
	t.Helper()
	names, err := filepath.Glob(filepath.Join(vectorsDir, "positive", "*.proof.json"))
	if err != nil {
		t.Fatal(err)
	}
	sort.Strings(names)
	if len(names) != 11 {
		t.Fatalf("found %d sound proofs, want 11", len(names))
	}
	var out []soundProof
	for _, name := range names {
		document, ok := mustParseFile(t, name).(map[string]any)
		if !ok {
			t.Fatalf("%s is not an object", name)
		}
		proof, ok := document["proof"].(map[string]any)
		if !ok {
			t.Fatalf("%s has no proof object", name)
		}
		rows, ok := mustParseFile(t, strings.TrimSuffix(name, ".proof.json")+".trace.json").([]any)
		if !ok || len(rows) == 0 {
			t.Fatalf("%s has no trace rows", name)
		}
		var trace []map[string]any
		for _, row := range rows {
			object, ok := row.(map[string]any)
			if !ok {
				t.Fatalf("%s: a trace row is not an object", name)
			}
			trace = append(trace, object)
		}
		out = append(out, soundProof{name: filepath.Base(name), document: document, proof: proof, trace: trace})
	}
	return out
}

// inputs returns a trace row's inputs.
func inputs(t *testing.T, row map[string]any) []string {
	t.Helper()
	list, ok := row["inputs"].([]any)
	if !ok {
		t.Fatalf("trace row %v has no inputs", row)
	}
	out := make([]string, len(list))
	for i, input := range list {
		text, ok := input.(string)
		if !ok {
			t.Fatalf("trace row %v has an input that is not a string", row)
		}
		out[i] = text
	}
	return out
}

// lastSegment is the last segment of a pointer.
func lastSegment(at string) string {
	return at[strings.LastIndex(at, "/")+1:]
}

// stringAt returns the string at a pointer.
func stringAt(t *testing.T, value any, at string) string {
	t.Helper()
	text, ok := pointer(t, value, at).(string)
	if !ok {
		t.Fatalf("%s is not a string", at)
	}
	return text
}
