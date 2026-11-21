// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"go/parser"
	"go/token"
	"io/fs"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

// The CLI reads and writes JSON through encoding/json/v2 alone (and
// internal/strictjson and internal/jcs for proofs, keyrings and source
// answers). v1's encoding/json silently keeps the last of two repeated
// member names, replaces invalid UTF-8 with U+FFFD and matches field names
// without regard to case; one file going back to it would bring those back
// for whatever it reads, unnoticed. Mutant killed: any .go file in the
// module, test files included, importing "encoding/json".
func TestNoFileImportsEncodingJSONV1(t *testing.T) {
	root := repoRoot(t)
	fset := token.NewFileSet()
	scanned := 0
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			switch d.Name() {
			case ".git", ".jj", "testdata", "tmp", "build", "dist", "node_modules":
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(path, ".go") {
			return nil
		}
		file, err := parser.ParseFile(fset, path, nil, parser.ImportsOnly)
		if err != nil {
			return err
		}
		scanned++
		for _, spec := range file.Imports {
			if p, _ := strconv.Unquote(spec.Path.Value); p == "encoding/json" {
				rel, _ := filepath.Rel(root, path)
				t.Errorf("%s imports encoding/json; use encoding/json/v2 (and encoding/json/jsontext)", rel)
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if scanned < 100 {
		t.Fatalf("scanned %d Go files; the walk is not reaching the module", scanned)
	}
}
