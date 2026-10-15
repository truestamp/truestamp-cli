// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: Apache-2.0

// This file ports the reference verifier's Truestamp.Verifier.Constructions
// (elixir/lib/truestamp/verifier/constructions.ex in truestamp-verification):
// CONSTRUCTIONS_ACCEPTED (SPEC.md sections 7.2 and 9), the digests of the
// constructions texts under spec/constructions/. Each file there holds a
// text's exact JCS bytes and is named <d>.json, where d is the lowercase hex
// SHA-256 of those bytes; the program stops on a file whose name is not the
// digest of its bytes, as the reference's build does.
//
// This file holds the digests only, and exports no hash. It never reads a
// proof: SHAPE's check 6 (SPEC.md section 10.2, step 7) computes a full
// text's digest itself, as the reference's Truestamp.Verifier.Shape does,
// or takes a reference's from its URL, and asks [IsAccepted].

package specdata

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io/fs"
	"path"
	"regexp"
	"slices"
	"strings"
)

// constructionsDir is the directory under spec/ that holds the texts.
const constructionsDir = "constructions"

// textName is the name of a text: its digest, 64 lowercase hex digits, and
// ".json".
var textName = regexp.MustCompile(`^[0-9a-f]{64}\.json$`)

// accepted is CONSTRUCTIONS_ACCEPTED, from the embedded texts.
var accepted = mustAcceptedDigests()

// acceptedDigests returns the digests of the texts in dir of fsys, sorted.
// Every entry there must be a regular file named by the SHA-256 of its
// bytes, and there must be one at least.
func acceptedDigests(fsys fs.FS, dir string) ([]string, error) {
	entries, err := fs.ReadDir(fsys, dir)
	if err != nil {
		return nil, fmt.Errorf("specdata: spec/%s/: %w", dir, err)
	}
	digests := make([]string, 0, len(entries))
	for _, entry := range entries {
		name := entry.Name()
		if !entry.Type().IsRegular() || !textName.MatchString(name) {
			return nil, fmt.Errorf("specdata: spec/%s/%s is not a file named <d>.json", dir, name)
		}
		data, err := fs.ReadFile(fsys, path.Join(dir, name))
		if err != nil {
			return nil, fmt.Errorf("specdata: spec/%s/%s: %w", dir, name, err)
		}
		sum := sha256.Sum256(data)
		digest := strings.TrimSuffix(name, ".json")
		if hex.EncodeToString(sum[:]) != digest {
			return nil, fmt.Errorf("specdata: spec/%s/%s is not named by the SHA-256 of its bytes", dir, name)
		}
		digests = append(digests, digest)
	}
	if len(digests) == 0 {
		return nil, fmt.Errorf("specdata: spec/%s/ holds no text", dir)
	}
	slices.Sort(digests)
	return digests, nil
}

func mustAcceptedDigests() []string {
	sub, err := fs.Sub(specFS, specRoot)
	if err == nil {
		var digests []string
		if digests, err = acceptedDigests(sub, constructionsDir); err == nil {
			return digests
		}
	}
	panic(err.Error())
}

// Accepted returns CONSTRUCTIONS_ACCEPTED, the accepted digests, sorted.
// Each call returns a new slice.
func Accepted() []string {
	return slices.Clone(accepted)
}

// IsAccepted reports whether digest, 64 lowercase hex digits, is in
// CONSTRUCTIONS_ACCEPTED.
func IsAccepted(digest string) bool {
	_, found := slices.BinarySearch(accepted, digest)
	return found
}
