// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: MIT

// Package testfixtures locates the shared golden test data under the
// repository's top-level testdata/ directory, so tests in every package read
// one copy of each fixture rather than carrying their own.
//
// Layout:
//
//	testdata/prod-2026-09-03/  a real production item, end to end: the
//	                           complete, compact and partial bundles (JSON),
//	                           the complete bundle as CBOR, the production
//	                           keyring, the reference verifier's offline
//	                           reports and the server's online reports
//	testdata/tamper/           mutations of the complete bundle with the
//	                           server's and the reference verifier's reports
//	testdata/staging-2026-09-03/ a staging item whose bundle carries a real
//	                           Bitcoin regtest commitment (raw transaction,
//	                           txoutproof, block merkle root) ahead of its
//	                           Stellar testnet one, the staging keyring, and
//	                           the server's online report
//	testdata/whitepaper/       the Appendix D worked bundle, its derivation
//	                           trace, and the Appendix C vectors
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
	ProdCBOR      = "proof-complete.cbor"
	ProdKeyring   = "keyring.json"
	TamperDir     = "tamper"
	StagingDir    = "staging-2026-09-03"
	WhitepaperDir = "whitepaper"
	AppendixD     = "bundle.json"
)
