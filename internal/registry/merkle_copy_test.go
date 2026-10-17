// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: Apache-2.0

package registry

import (
	"bufio"
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"slices"
	"testing"
)

// merkleCommit is the commit of truestamp-merkle-ex the copy is taken
// from: the one the reference verifier's elixir/mix.lock pins, so the
// known answers are those of the library the reference walks with.
const merkleCommit = "0862f3e96f670ef605bdfa4077c96039040e585f"

// merkleDigest is the SHA-256 of vectors/merkle.json at merkleCommit. A sync
// of another commit changes it, and this test, with SOURCE.
const merkleDigest = "6ebae6ec8afa1c9b578f9d83dfe3b71ffcc2359fd502e0fc5d8c792b65f10eee"

// testdata/merkle/ is what task merkle-vectors-sync writes: merkle.json,
// byte for byte, and SOURCE, the commit's full id and a newline; and the
// root .gitattributes keeps git from converting its line endings. git is
// not run here: the file is read. Mutants killed: SOURCE naming main's
// commit 0c089838 in place of the pinned one; the .gitattributes line
// dropped; one byte of merkle.json changed.
func TestMerkleCopy_IsTheLibrarysFileAtThePinnedCommit(t *testing.T) {
	dir := filepath.Join(repoRoot, "testdata", "merkle")
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	var names []string
	for _, entry := range entries {
		if !entry.Type().IsRegular() {
			t.Errorf("%s/%s is not a regular file", dir, entry.Name())
		}
		names = append(names, entry.Name())
	}
	if !slices.Equal(names, []string{"SOURCE", "merkle.json"}) {
		t.Fatalf("%s holds %v, want SOURCE and merkle.json", dir, names)
	}
	source, err := os.ReadFile(filepath.Join(dir, "SOURCE"))
	if err != nil {
		t.Fatal(err)
	}
	if string(source) != merkleCommit+"\n" {
		t.Errorf("SOURCE is %q, want %s and a newline", source, merkleCommit)
	}
	data, err := os.ReadFile(filepath.Join(dir, "merkle.json"))
	if err != nil {
		t.Fatal(err)
	}
	if sum := sha256.Sum256(data); hex.EncodeToString(sum[:]) != merkleDigest {
		t.Errorf("merkle.json has SHA-256 %x, the file at %s %s", sum, merkleCommit, merkleDigest)
	}
	attributes, err := os.ReadFile(filepath.Join(repoRoot, ".gitattributes"))
	if err != nil {
		t.Fatal(err)
	}
	found := false
	scanner := bufio.NewScanner(bytes.NewReader(attributes))
	for scanner.Scan() {
		if string(bytes.TrimSpace(scanner.Bytes())) == "/testdata/merkle/** -text" {
			found = true
		}
	}
	if !found {
		t.Error(".gitattributes lacks the line /testdata/merkle/** -text")
	}
	for _, name := range []string{"testdata/.gitattributes"} {
		if _, err := os.Lstat(filepath.Join(repoRoot, filepath.FromSlash(name))); err == nil {
			t.Errorf("%s may set attributes on the copy", name)
		}
	}
}
