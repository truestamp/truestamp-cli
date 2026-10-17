// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: Apache-2.0

// This file holds the inclusion walk of SPEC.md section 5.3, merkle_path,
// the verification of RFC 9162 section 2.1.3.2 over the tree of section 5.1,
// and the reading of a carried MERKLE_PROOF (section 5.2) it walks. It ports
// the walk the reference verifier calls, Truestamp.Merkle.walk/3 of
// truestamp-merkle-ex (lib/truestamp/merkle.ex and
// lib/truestamp/merkle/paths.ex, at the commit the reference's
// elixir/mix.lock pins), as Truestamp.Verifier.Derive.Links
// (elixir/lib/truestamp/verifier/derive/links.ex in truestamp-verification)
// calls it, with at most MAX_PATH_STEPS nodes.
//
// Adapted from the library: the hashing is that of
// github.com/transparency-dev/merkle, its proof.RootFromInclusionProof with
// its RFC 6962 hasher, whose leaf and node bytes are the registry's 0x00 and
// 0x01 (SPEC.md section 4.1). This is the one place the verifier uses those
// two bytes (SPEC.md section 10.6, rule 2). The library checks that the
// index is below the size and that the path has exactly the length they
// imply, but not MAX_PATH_STEPS, so the walk checks the cap first; and it
// takes the leaf hash, so the walk computes SHA-256(0x00 || leaf) with the
// library's hasher before it calls the library. Where the Elixir library
// takes a leaf and path nodes as hex, [Walk] takes their 32 bytes, and
// [ReadInclusionProof] and [DecodeHex64] read the hex a proof carries. A
// walk that fails gives no root (section 5.3).

package registry

import (
	"errors"
	"fmt"

	"github.com/transparency-dev/merkle/proof"
	"github.com/transparency-dev/merkle/rfc6962"

	"github.com/truestamp/truestamp-cli/internal/specdata"
	"github.com/truestamp/truestamp-cli/internal/strictjson"
)

// MaxPathSteps is MAX_PATH_STEPS (SPEC.md section 9), the most nodes a
// path may hold.
const MaxPathSteps = specdata.MaxPathSteps

// The walk's refusals, and those of reading a carried proof. Each names the
// first check that fails, in the order [ReadInclusionProof] and [Walk] make
// them.
var (
	// ErrNotInclusionProof marks a value that is not an object with an
	// INT leaf_index, an INT tree_size and an array path.
	ErrNotInclusionProof = errors.New("registry: not a MERKLE_PROOF")
	// ErrInvalidNode marks a path node that is not HEX64.
	ErrInvalidNode = errors.New("registry: a path node is not HEX64")
	// ErrIndexOutOfRange marks a leaf_index that is not below tree_size.
	ErrIndexOutOfRange = errors.New("registry: leaf_index is not below tree_size")
	// ErrTooManySteps marks a path of more than MAX_PATH_STEPS nodes.
	ErrTooManySteps = errors.New("registry: the path has more than MAX_PATH_STEPS nodes")
	// ErrWrongPathLength marks a path whose length is not the one
	// leaf_index and tree_size imply.
	ErrWrongPathLength = errors.New("registry: the path's length does not fit leaf_index and tree_size")
	// ErrNotHex64 marks a value that is not 64 lowercase hex characters.
	ErrNotHex64 = errors.New("registry: not HEX64")
)

// InclusionProof is a MERKLE_PROOF (SPEC.md section 5.2): the leaf's
// position, the number of leaves, and the sibling hashes from the leaf up
// to the root.
type InclusionProof struct {
	LeafIndex uint64
	TreeSize  uint64
	Path      [][32]byte
}

// Walk is merkle_path(leaf, p) of SPEC.md section 5.3: the root p implies
// for the leaf value leaf, which the caller has recomputed and never takes
// from the document. It refuses, in this order, a leaf_index not below
// tree_size ([ErrIndexOutOfRange]), a path of more than MAX_PATH_STEPS
// nodes ([ErrTooManySteps]), and a path whose length is not the one the
// index and size imply ([ErrWrongPathLength]). The root is returned for the
// caller to compare: Walk compares nothing with a carried root, and checks
// nothing of tree_size but its fit with the path (section 5.4 checks it).
func Walk(leaf [32]byte, p InclusionProof) ([32]byte, error) {
	return walk(leaf, p, MaxPathSteps)
}

// walk is [Walk] with maxSteps in place of MAX_PATH_STEPS, so that the
// tests can run the Merkle library's known answers at their own caps. It is
// the one walk.
func walk(leaf [32]byte, p InclusionProof, maxSteps int) ([32]byte, error) {
	if p.LeafIndex >= p.TreeSize {
		return [32]byte{}, ErrIndexOutOfRange
	}
	if len(p.Path) > maxSteps {
		return [32]byte{}, ErrTooManySteps
	}
	nodes := make([][]byte, len(p.Path))
	for i := range p.Path {
		nodes[i] = p.Path[i][:]
	}
	hasher := rfc6962.DefaultHasher
	root, err := proof.RootFromInclusionProof(hasher, p.LeafIndex, p.TreeSize, hasher.HashLeaf(leaf[:]), nodes)
	if err != nil {
		// The index is below the size and the leaf hash is 32 bytes, so
		// the library's one refusal left is the path's length.
		return [32]byte{}, fmt.Errorf("%w: %v", ErrWrongPathLength, err)
	}
	// The root is the 32-byte leaf hash or a node the RFC 6962 hasher
	// gives, a SHA-256 sum, so its length is fixed: the conversion panics
	// only on a defect in the library, as digest does on one in this
	// package.
	return [32]byte(root), nil
}

// ReadInclusionProof reads a carried MERKLE_PROOF, a value as
// internal/strictjson parses it, into the [InclusionProof] [Walk] takes. It
// reads the members leaf_index, path and tree_size by those names, and no
// other: SHAPE (SPEC.md section 10.2, step 7) holds the object to exactly
// them. It refuses ([ErrNotInclusionProof]) a value that is not an object,
// a leaf_index or tree_size that is not an INT (an integer from 0 to
// MAX_INTEGER), and a path that is not an array; and ([ErrInvalidNode]) a
// node that is not HEX64.
func ReadInclusionProof(value any) (InclusionProof, error) {
	object, ok := value.(map[string]any)
	if !ok {
		return InclusionProof{}, ErrNotInclusionProof
	}
	index, okIndex := readINT(object["leaf_index"])
	size, okSize := readINT(object["tree_size"])
	path, okPath := object["path"].([]any)
	if !okIndex || !okSize || !okPath {
		return InclusionProof{}, ErrNotInclusionProof
	}
	nodes := make([][32]byte, len(path))
	for i, node := range path {
		text, isString := node.(string)
		if !isString {
			return InclusionProof{}, ErrInvalidNode
		}
		decoded, err := DecodeHex64(text)
		if err != nil {
			return InclusionProof{}, ErrInvalidNode
		}
		nodes[i] = decoded
	}
	return InclusionProof{LeafIndex: index, TreeSize: size, Path: nodes}, nil
}

// readINT reads an INT of SPEC.md section 2.4, an integer from 0 to
// MAX_INTEGER, from a parsed value.
func readINT(value any) (uint64, bool) {
	number, ok := value.(strictjson.Number)
	if !ok {
		return 0, false
	}
	n, ok := number.Int64()
	if !ok || n < 0 {
		return 0, false
	}
	return uint64(n), true
}

// DecodeHex64 returns the 32 bytes text spells when it is HEX64 (SPEC.md
// section 2.4): exactly 64 characters, each a digit or a lowercase a to f.
// Any other character refuses the whole value, so no case, space or other
// character is skipped as some decoders skip them. The refusal is
// [ErrNotHex64].
func DecodeHex64(text string) ([32]byte, error) {
	var out [32]byte
	if len(text) != 2*len(out) {
		return out, ErrNotHex64
	}
	for i := range out {
		high, okHigh := hexDigit(text[2*i])
		low, okLow := hexDigit(text[2*i+1])
		if !okHigh || !okLow {
			return [32]byte{}, ErrNotHex64
		}
		out[i] = high<<4 | low
	}
	return out, nil
}

// hexDigit reads one lowercase hex digit.
func hexDigit(c byte) (byte, bool) {
	switch {
	case '0' <= c && c <= '9':
		return c - '0', true
	case 'a' <= c && c <= 'f':
		return c - 'a' + 10, true
	}
	return 0, false
}
