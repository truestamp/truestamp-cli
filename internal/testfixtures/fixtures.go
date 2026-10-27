// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: Apache-2.0

// Package testfixtures locates the bundles of the earlier layout under the
// repository's top-level testdata/ directory, which the commands that
// still read that layout through internal/proof are tested with: inspect,
// the bundle verify --remote posts, and internal/proof's own parser. The conformance cases
// of SPEC.md section 13.2 are under testdata/verification/, which
// internal/vectorcases locates.
//
// Layout:
//
//	testdata/prod-2026-09-03/  a production item of the earlier layout:
//	                           its complete, compact and partial bundles
//	testdata/tamper/           a mutation of the complete bundle
//	                           (tamper-type.json) and one bundle of a layout
//	                           before it (old-layout.json), each of which
//	                           internal/proof's parser refuses
//	testdata/whitepaper/       the earlier layout's worked bundle
package testfixtures

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

// Root returns the absolute path of the repository's testdata directory.
func Root() string {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		panic("testfixtures: cannot locate source file")
	}
	return filepath.Join(filepath.Dir(file), "..", "..", "testdata")
}

// Path joins parts under [Root].
func Path(parts ...string) string {
	return filepath.Join(append([]string{Root()}, parts...)...)
}

// Read returns a fixture's bytes, failing the test when it cannot be read.
func Read(t testing.TB, parts ...string) []byte {
	t.Helper()
	data, err := os.ReadFile(Path(parts...))
	if err != nil {
		t.Fatalf("reading fixture %s: %v", filepath.Join(parts...), err)
	}
	return data
}

// Fixture names, so a typo is a compile error rather than a missing file.
const (
	ProdDir       = "prod-2026-09-03"
	ProdComplete  = "proof-complete.json"
	ProdCompact   = "proof-compact.json"
	ProdPartial   = "proof-partial.json"
	TamperDir     = "tamper"
	WhitepaperDir = "whitepaper"
	AppendixD     = "bundle.json"
)
