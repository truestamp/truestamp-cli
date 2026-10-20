// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: Apache-2.0

// This file reads a file under spec/, the data files of SPEC.md section 9
// (What a verifier pins), for the tables that compile its contents. The
// files are embedded, and File gives a file's bytes rather than its parsed
// value: the strict reader of SPEC.md section 10.1 parses them.

package specdata

import (
	"embed"
	"fmt"
	"io/fs"
	"path"
	"slices"
	"strings"
)

// specFS holds verification/spec/. The all: prefix embeds a file whose name
// begins with '_' too, which the manifest's grammar allows; one beginning
// with '.' the grammar and the sync refuse.
//
//go:embed all:verification/spec
var specFS embed.FS

// sourceFile is verification/SOURCE: the full id of the spec commit the copy
// came from and a newline.
//
//go:embed verification/SOURCE
var sourceFile string

// specRoot is the directory in specFS that is the spec repository's spec/.
const specRoot = "verification/spec"

// File returns the bytes of name, a path under spec/ such as
// "constants.json" or "constructions/<d>.json", exactly as the sync copied
// them. Each call returns a new slice. A name that is not a file under spec/,
// or not a clean relative path, is an error.
func File(name string) ([]byte, error) {
	if !fs.ValidPath(name) || name == "." {
		return nil, fmt.Errorf("specdata: %q is not a path under spec/", name)
	}
	data, err := specFS.ReadFile(path.Join(specRoot, name))
	if err != nil {
		return nil, fmt.Errorf("specdata: spec/%s: %w", name, err)
	}
	return data, nil
}

// Names returns the path under spec/ of every embedded file, sorted.
func Names() []string {
	var names []string
	err := fs.WalkDir(specFS, specRoot, func(p string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !entry.IsDir() {
			names = append(names, strings.TrimPrefix(p, specRoot+"/"))
		}
		return nil
	})
	if err != nil {
		// specFS is fixed at build time, so a walk of it cannot fail.
		panic(fmt.Sprintf("specdata: walking the embedded spec/: %v", err))
	}
	slices.Sort(names)
	return names
}

// Source returns the full id of the spec commit the embedded files came
// from, as verification/SOURCE names it, without its newline.
func Source() string {
	return strings.TrimSuffix(sourceFile, "\n")
}

// PrefixByte reads a registry byte as spec/prefixes.json writes it: "0x"
// and two lowercase hex digits, such as "0x50". Any other spelling is an
// error.
func PrefixByte(text string) (byte, error) {
	if len(text) != 4 || text[:2] != "0x" {
		return 0, fmt.Errorf("specdata: not a prefix byte: %q", text)
	}
	var value byte
	for i := 2; i < 4; i++ {
		c := text[i]
		switch {
		case '0' <= c && c <= '9':
			value = value<<4 | (c - '0')
		case 'a' <= c && c <= 'f':
			value = value<<4 | (c - 'a' + 10)
		default:
			return 0, fmt.Errorf("specdata: not a prefix byte: %q", text)
		}
	}
	return value, nil
}
