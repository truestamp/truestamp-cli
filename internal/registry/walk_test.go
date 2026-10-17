// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: Apache-2.0

package registry

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/transparency-dev/merkle/rfc6962"
	"golang.org/x/mod/sumdb/tlog"

	"github.com/truestamp/truestamp-cli/internal/strictjson"
)

// emptyRoot is the root of the empty tree, SHA-256 of zero bytes (SPEC.md
// section 5.4).
const emptyRoot = "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"

// The library's RFC 6962 hasher, the one the walk uses, hashes a leaf and a
// node under the registry's 0x00 and 0x01 (SPEC.md section 4.1), and its
// empty root is SHA-256 of zero bytes. So the walk applies the registry's
// bytes, and no others. Mutant killed: the table's merkle_node byte changed
// to 0x02.
func TestWalk_TheLibraryHashesUnderTheRegistrysBytes(t *testing.T) {
	leaf, node := entryOf(t, MerkleLeaf), entryOf(t, MerkleNode)
	value := bytes.Repeat([]byte{0xab}, 32)
	left, right := bytes.Repeat([]byte{1}, 32), bytes.Repeat([]byte{2}, 32)
	hasher := rfc6962.DefaultHasher
	if got, want := hex.EncodeToString(hasher.HashLeaf(value)), sha256Hex([]byte{leaf.Byte}, value); got != want {
		t.Errorf("HashLeaf = %s, want SHA-256(%#02x || v) = %s", got, leaf.Byte, want)
	}
	if got, want := hex.EncodeToString(hasher.HashChildren(left, right)), sha256Hex([]byte{node.Byte}, left, right); got != want {
		t.Errorf("HashChildren = %s, want SHA-256(%#02x || l || r) = %s", got, node.Byte, want)
	}
	if got := hex.EncodeToString(hasher.EmptyRoot()); got != emptyRoot {
		t.Errorf("EmptyRoot = %s, want %s", got, emptyRoot)
	}
}

// The Merkle known answers, testdata/merkle/merkle.json, as
// truestamp-merkle-ex's README defines its members. Integers above
// 2^53 stay exact as json.Number.
type merkleVectors struct {
	Description string `json:"description"`
	Constants   struct {
		EmptyRoot   string      `json:"empty_root"`
		MaxSteps    int         `json:"max_steps"`
		MaxTreeSize json.Number `json:"max_tree_size"`
	} `json:"constants"`
	Trees []struct {
		Name    string `json:"name"`
		Entries []struct {
			Key    string `json:"key"`
			Digest string `json:"digest"`
		} `json:"entries"`
		Root     string      `json:"root"`
		Depth    int         `json:"depth"`
		TreeSize json.Number `json:"tree_size"`
		Paths    []struct {
			Key       string      `json:"key"`
			Digest    string      `json:"digest"`
			LeafIndex json.Number `json:"leaf_index"`
			TreeSize  json.Number `json:"tree_size"`
			Path      []string    `json:"path"`
			BinaryHex string      `json:"binary_hex"`
		} `json:"paths"`
	} `json:"trees"`
	EntryRefusals []json.RawMessage `json:"entry_refusals"`
	WalkAccepts   []struct {
		Name      string          `json:"name"`
		Digest    string          `json:"digest"`
		Proof     json.RawMessage `json:"proof"`
		MaxSteps  int             `json:"max_steps"`
		Root      string          `json:"root"`
		BinaryHex string          `json:"binary_hex"`
	} `json:"walk_accepts"`
	WalkRefusals []struct {
		Name     string          `json:"name"`
		Digest   string          `json:"digest"`
		Proof    json.RawMessage `json:"proof"`
		MaxSteps int             `json:"max_steps"`
		Error    string          `json:"error"`
	} `json:"walk_refusals"`
	BinaryRefusals []json.RawMessage `json:"binary_refusals"`
}

func loadMerkleVectors(t *testing.T) merkleVectors {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(repoRoot, "testdata", "merkle", "merkle.json"))
	if err != nil {
		t.Fatal(err)
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.UseNumber()
	decoder.DisallowUnknownFields()
	var vectors merkleVectors
	if err := decoder.Decode(&vectors); err != nil {
		t.Fatalf("merkle.json: %v", err)
	}
	if decoder.More() {
		t.Fatal("merkle.json has content after its value")
	}
	return vectors
}

func mustHex64(t *testing.T, text string) [32]byte {
	t.Helper()
	out, err := DecodeHex64(text)
	if err != nil {
		t.Fatalf("%q: %v", text, err)
	}
	return out
}

func mustUint64(t *testing.T, n json.Number) uint64 {
	t.Helper()
	out, err := strconv.ParseUint(n.String(), 10, 64)
	if err != nil {
		t.Fatalf("%s: %v", n, err)
	}
	return out
}

// typedProof reads a known answer's proof object, whose integers may pass
// 2^53, into an InclusionProof.
func typedProof(t *testing.T, raw json.RawMessage) InclusionProof {
	t.Helper()
	var p struct {
		LeafIndex json.Number `json:"leaf_index"`
		TreeSize  json.Number `json:"tree_size"`
		Path      []string    `json:"path"`
	}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber()
	if err := decoder.Decode(&p); err != nil {
		t.Fatal(err)
	}
	out := InclusionProof{LeafIndex: mustUint64(t, p.LeafIndex), TreeSize: mustUint64(t, p.TreeSize)}
	for _, node := range p.Path {
		out.Path = append(out.Path, mustHex64(t, node))
	}
	return out
}

// parsedProof reads a known answer's proof object into the form
// internal/strictjson gives a carried value: a number is a
// strictjson.Number, its nearest binary64 value.
func parsedProof(t *testing.T, raw json.RawMessage) any {
	t.Helper()
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber()
	var value any
	if err := decoder.Decode(&value); err != nil {
		t.Fatal(err)
	}
	var convert func(any) any
	convert = func(v any) any {
		switch typed := v.(type) {
		case json.Number:
			f, err := strconv.ParseFloat(typed.String(), 64)
			if err != nil {
				t.Fatal(err)
			}
			return strictjson.Number(f)
		case []any:
			for i := range typed {
				typed[i] = convert(typed[i])
			}
		case map[string]any:
			for name := range typed {
				typed[name] = convert(typed[name])
			}
		}
		return v
	}
	return convert(value)
}

// treeRoot is RFC 9162's Merkle Tree Hash of leaves, through tlog.
func treeRoot(t *testing.T, leaves [][32]byte) tlog.Hash {
	t.Helper()
	var stored memoryHashes
	for i, leaf := range leaves {
		hashes, err := tlog.StoredHashes(int64(i), leaf[:], &stored)
		if err != nil {
			t.Fatal(err)
		}
		stored = append(stored, hashes...)
	}
	root, err := tlog.TreeHash(int64(len(leaves)), &stored)
	if err != nil {
		t.Fatal(err)
	}
	return root
}

// memoryHashes is a tlog.HashReader over the hashes stored so far.
type memoryHashes []tlog.Hash

func (m *memoryHashes) ReadHashes(indexes []int64) ([]tlog.Hash, error) {
	out := make([]tlog.Hash, len(indexes))
	for i, index := range indexes {
		if index < 0 || index >= int64(len(*m)) {
			return nil, fmt.Errorf("no stored hash %d", index)
		}
		out[i] = (*m)[index]
	}
	return out, nil
}

// The known answers' trees: each root is RFC 9162's tree hash of the
// entries sorted by key, byte-wise (SPEC.md section 5.1), computed through
// tlog, and every path walks from its digest to that root, under
// MAX_PATH_STEPS too, since no tree is deeper than 32. Mutant killed: walk
// handing the library tree_size - 1.
func TestWalk_MerkleKnownAnswersTrees(t *testing.T) {
	vectors := loadMerkleVectors(t)
	if vectors.Constants.EmptyRoot != emptyRoot {
		t.Fatalf("the known answers' empty root is %s", vectors.Constants.EmptyRoot)
	}
	paths := 0
	for _, tree := range vectors.Trees {
		entries := slices.Clone(tree.Entries)
		slices.SortFunc(entries, func(a, b struct {
			Key    string `json:"key"`
			Digest string `json:"digest"`
		}) int {
			return strings.Compare(a.Key, b.Key)
		})
		var leaves [][32]byte
		for _, entry := range entries {
			leaves = append(leaves, mustHex64(t, entry.Digest))
		}
		if mustUint64(t, tree.TreeSize) != uint64(len(leaves)) {
			t.Fatalf("%s: tree_size %s for %d entries", tree.Name, tree.TreeSize, len(leaves))
		}
		root := treeRoot(t, leaves)
		if got := hex.EncodeToString(root[:]); got != tree.Root {
			t.Errorf("%s: the tree hash of the sorted entries is %s, the known answer %s", tree.Name, got, tree.Root)
		}
		for _, p := range tree.Paths {
			proof := InclusionProof{LeafIndex: mustUint64(t, p.LeafIndex), TreeSize: mustUint64(t, p.TreeSize)}
			for _, node := range p.Path {
				proof.Path = append(proof.Path, mustHex64(t, node))
			}
			got, err := Walk(mustHex64(t, p.Digest), proof)
			if err != nil || hex.EncodeToString(got[:]) != tree.Root {
				t.Errorf("%s %s: the walk gives %x, %v, want %s", tree.Name, p.Key, got, err, tree.Root)
			}
			if leaves[proof.LeafIndex] != mustHex64(t, p.Digest) {
				t.Errorf("%s %s: leaf_index %d is not the entry's place in key order", tree.Name, p.Key, proof.LeafIndex)
			}
			paths++
		}
	}
	if len(vectors.Trees) != 11 || paths != 53 {
		t.Fatalf("%d trees and %d paths, want 11 and 53", len(vectors.Trees), paths)
	}
}

// The known answers' accepted walks reach their roots at their own caps,
// and under MAX_PATH_STEPS when their paths fit it; a longer path is
// refused under MAX_PATH_STEPS. Mutant killed: walk handing the library
// the path reversed.
func TestWalk_MerkleKnownAnswersAccepts(t *testing.T) {
	vectors := loadMerkleVectors(t)
	long := 0
	for _, c := range vectors.WalkAccepts {
		proof := typedProof(t, c.Proof)
		leaf := mustHex64(t, c.Digest)
		got, err := walk(leaf, proof, c.MaxSteps)
		if err != nil || hex.EncodeToString(got[:]) != c.Root {
			t.Errorf("%s: the walk gives %x, %v, want %s", c.Name, got, err, c.Root)
		}
		got, err = Walk(leaf, proof)
		switch {
		case len(proof.Path) <= MaxPathSteps:
			if err != nil || hex.EncodeToString(got[:]) != c.Root {
				t.Errorf("%s: Walk gives %x, %v, want %s", c.Name, got, err, c.Root)
			}
		case !errors.Is(err, ErrTooManySteps):
			t.Errorf("%s: Walk of %d nodes = %x, %v, want ErrTooManySteps", c.Name, len(proof.Path), got, err)
		default:
			long++
		}
	}
	if len(vectors.WalkAccepts) != 13 || long == 0 {
		t.Fatalf("%d accepted walks, %d of them past MAX_PATH_STEPS", len(vectors.WalkAccepts), long)
	}
}

// refusalBy names the refusal the walk gives where it differs from the
// Merkle library's, and why. The walk reads a carried proof whole before it
// walks, so a node is checked before the path's length; it checks the
// carried path's length against the cap, as SPEC.md section 5.3 states it,
// where the library checks the length the index and size imply; and a
// tree_size of 0 is an INT, refused because no index is below it.
var refusalBy = map[string]error{
	"the path length is checked before any node": ErrInvalidNode,
	"a 33-step path under a cap of 32":           ErrWrongPathLength,
	"tree_size 0":                                ErrIndexOutOfRange,
}

// The library's names for its refusals, and the walk's.
var refusalNames = map[string]error{
	"invalid_leaf":       ErrNotHex64,
	"invalid_proof":      ErrNotInclusionProof,
	"invalid_node":       ErrInvalidNode,
	"index_out_of_range": ErrIndexOutOfRange,
	"too_many_steps":     ErrTooManySteps,
	"wrong_path_length":  ErrWrongPathLength,
}

// Every walk the known answers refuse is refused when its digest is read as
// HEX64 and its proof as a carried MERKLE_PROOF, with the library's reason
// but for the three refusalBy names. Mutants killed: readINT accepting a
// negative integer; ReadInclusionProof naming a node that is not a string
// ErrNotInclusionProof.
func TestWalk_MerkleKnownAnswersRefusals(t *testing.T) {
	vectors := loadMerkleVectors(t)
	for _, c := range vectors.WalkRefusals {
		want, ok := refusalBy[c.Name]
		if !ok {
			if want, ok = refusalNames[c.Error]; !ok {
				t.Fatalf("%s: no walk refusal for %q", c.Name, c.Error)
			}
		}
		err := func() error {
			leaf, err := DecodeHex64(c.Digest)
			if err != nil {
				return err
			}
			proof, err := ReadInclusionProof(parsedProof(t, c.Proof))
			if err != nil {
				return err
			}
			_, err = walk(leaf, proof, c.MaxSteps)
			return err
		}()
		if !errors.Is(err, want) {
			t.Errorf("%s: %v, want %v (the library's %s)", c.Name, err, want, c.Error)
		}
	}
	if len(vectors.WalkRefusals) != 54 {
		t.Fatalf("%d refused walks, want 54", len(vectors.WalkRefusals))
	}
}

// crossCheckSize is the largest tree the tlog cross-check builds.
const crossCheckSize = 300

// crossCheckLeaves are SHA-256("leaf" || i) for i below crossCheckSize, with
// tlog's stored hashes for them.
func crossCheckLeaves(t *testing.T) ([][32]byte, *memoryHashes) {
	t.Helper()
	leaves := make([][32]byte, crossCheckSize)
	stored := &memoryHashes{}
	for i := range leaves {
		leaves[i] = sha256.Sum256([]byte("leaf" + strconv.Itoa(i)))
		hashes, err := tlog.StoredHashes(int64(i), leaves[i][:], stored)
		if err != nil {
			t.Fatal(err)
		}
		*stored = append(*stored, hashes...)
	}
	return leaves, stored
}

func walkPath(p tlog.RecordProof) [][32]byte {
	out := make([][32]byte, len(p))
	for i := range p {
		out[i] = [32]byte(p[i])
	}
	return out
}

// Cross-checked against golang.org/x/mod/sumdb/tlog, an independent RFC
// 6962 implementation: for every tree of 1 to 300 leaves and every leaf of
// it, the walk of tlog's proof gives tlog's tree hash, which tlog's own
// check accepts. Mutant killed: walk handing the library the leaf value as
// its leaf hash, never hashed under 0x00.
func TestWalk_CrossCheckWithTlog(t *testing.T) {
	leaves, stored := crossCheckLeaves(t)
	pairs := 0
	for size := int64(1); size <= crossCheckSize; size++ {
		root, err := tlog.TreeHash(size, stored)
		if err != nil {
			t.Fatal(err)
		}
		for index := int64(0); index < size; index++ {
			p, err := tlog.ProveRecord(size, index, stored)
			if err != nil {
				t.Fatal(err)
			}
			if err := tlog.CheckRecord(p, size, root, index, tlog.RecordHash(leaves[index][:])); err != nil {
				t.Fatalf("tlog refuses its own proof of %d in %d: %v", index, size, err)
			}
			got, err := Walk(leaves[index], InclusionProof{LeafIndex: uint64(index), TreeSize: uint64(size), Path: walkPath(p)})
			if err != nil || got != [32]byte(root) {
				t.Fatalf("leaf %d of %d: the walk gives %x, %v, tlog's root is %x", index, size, got, err, root)
			}
			pairs++
		}
	}
	if pairs != crossCheckSize*(crossCheckSize+1)/2 {
		t.Fatalf("checked %d pairs", pairs)
	}
}

// The walk and tlog's check agree on every claimed position: for every
// tree of 1 to 40 leaves, every leaf's proof presented at every index of
// every size from 1 to 3 past the tree's, and at the first two indexes past
// each size, reaches the true root under the walk exactly when tlog accepts
// it, and an index at or past its size is ErrIndexOutOfRange. Some moved
// positions are accepted by both, as SPEC.md section 5.3 says, its own
// example among them: the last of three leaves walks to the root as index 1
// of two. So the walk shows inclusion, and the caller holds tree_size to
// the committed leaf count (section 5.4). Mutant killed: the index check
// made index > size, leaving index = size to the library.
func TestWalk_AgreesWithTlogOnEveryClaimedPosition(t *testing.T) {
	leaves, stored := crossCheckLeaves(t)
	moved, checked := 0, 0
	lastOfThreeAsOneOfTwo := false
	for size := int64(1); size <= 40; size++ {
		root, err := tlog.TreeHash(size, stored)
		if err != nil {
			t.Fatal(err)
		}
		for index := int64(0); index < size; index++ {
			p, err := tlog.ProveRecord(size, index, stored)
			if err != nil {
				t.Fatal(err)
			}
			leafHash := tlog.RecordHash(leaves[index][:])
			for claimedSize := int64(1); claimedSize <= size+3; claimedSize++ {
				for claimedIndex := int64(0); claimedIndex < claimedSize+2; claimedIndex++ {
					got, err := Walk(leaves[index], InclusionProof{LeafIndex: uint64(claimedIndex), TreeSize: uint64(claimedSize), Path: walkPath(p)})
					ours := err == nil && got == [32]byte(root)
					theirs := claimedIndex < claimedSize && tlog.CheckRecord(p, claimedSize, root, claimedIndex, leafHash) == nil
					if ours != theirs {
						t.Fatalf("leaf %d of %d claimed as %d of %d: the walk accepts %v, tlog %v", index, size, claimedIndex, claimedSize, ours, theirs)
					}
					if ours && (claimedIndex != index || claimedSize != size) {
						moved++
						if index == 2 && size == 3 && claimedIndex == 1 && claimedSize == 2 {
							lastOfThreeAsOneOfTwo = true
						}
					}
					if claimedIndex >= claimedSize && !errors.Is(err, ErrIndexOutOfRange) {
						t.Fatalf("%d of %d: %v, want ErrIndexOutOfRange", claimedIndex, claimedSize, err)
					}
					checked++
				}
			}
		}
	}
	if moved == 0 || checked < 100000 || !lastOfThreeAsOneOfTwo {
		t.Fatalf("%d claims checked, %d moved ones accepted, SPEC.md section 5.3's example among them: %v", checked, moved, lastOfThreeAsOneOfTwo)
	}
}

// pathOf returns tlog's proof of leaf index in a tree of size leaves, with
// its root.
func pathOf(t *testing.T, leaves [][32]byte, index, size int64) (InclusionProof, [32]byte) {
	t.Helper()
	var stored memoryHashes
	for i := int64(0); i < size; i++ {
		hashes, err := tlog.StoredHashes(i, leaves[i][:], &stored)
		if err != nil {
			t.Fatal(err)
		}
		stored = append(stored, hashes...)
	}
	root, err := tlog.TreeHash(size, &stored)
	if err != nil {
		t.Fatal(err)
	}
	p, err := tlog.ProveRecord(size, index, &stored)
	if err != nil {
		t.Fatal(err)
	}
	return InclusionProof{LeafIndex: uint64(index), TreeSize: uint64(size), Path: walkPath(p)}, [32]byte(root)
}

// A leaf_index at or past tree_size, a tree_size of 0, and an index past
// any size are refused before any node is read. Mutant killed: the index
// check dropped, leaving the library's refusal, which Walk names
// ErrWrongPathLength.
func TestWalk_RefusesAnIndexAtOrPastTheSize(t *testing.T) {
	leaves, _ := crossCheckLeaves(t)
	proof, _ := pathOf(t, leaves, 0, 3)
	for _, c := range []struct{ index, size uint64 }{{3, 3}, {9, 3}, {0, 0}, {1 << 63, 1 << 63}, {^uint64(0), 1}} {
		proof.LeafIndex, proof.TreeSize = c.index, c.size
		if got, err := Walk(leaves[0], proof); !errors.Is(err, ErrIndexOutOfRange) || got != [32]byte{} {
			t.Errorf("%d of %d: %x, %v, want no root and ErrIndexOutOfRange", c.index, c.size, got, err)
		}
	}
}

// A path of more than MAX_PATH_STEPS nodes is refused, though its length
// fits its index and size and the library alone would walk it; a path of
// exactly 32 is walked. Mutants killed: the cap check dropped; the cap
// check made "at least MAX_PATH_STEPS".
func TestWalk_RefusesMoreThanMaxPathStepsNodes(t *testing.T) {
	nodes := make([][32]byte, 33)
	for i := range nodes {
		nodes[i] = sha256.Sum256([]byte{byte(i)})
	}
	leaf := sha256.Sum256([]byte("leaf"))
	over := InclusionProof{LeafIndex: 0, TreeSize: 1 << 33, Path: nodes}
	if got, err := Walk(leaf, over); !errors.Is(err, ErrTooManySteps) || got != [32]byte{} {
		t.Errorf("33 nodes: %x, %v, want ErrTooManySteps", got, err)
	}
	if _, err := walk(leaf, over, 64); err != nil {
		t.Fatalf("the library refuses 33 nodes at 2^33 itself: %v", err)
	}
	at := InclusionProof{LeafIndex: 0, TreeSize: 1 << 32, Path: nodes[:32]}
	if _, err := Walk(leaf, at); err != nil {
		t.Errorf("32 nodes at 2^32: %v, want a root", err)
	}
	if MaxPathSteps != 32 {
		t.Errorf("MaxPathSteps = %d, want SPEC.md section 9's 32", MaxPathSteps)
	}
}

// A path one node short, one node long, empty where a node is needed, or
// holding a node where a single leaf needs none is refused, with no root.
// Mutant killed: walk ignoring the library's error.
func TestWalk_RefusesAWrongPathLength(t *testing.T) {
	leaves, _ := crossCheckLeaves(t)
	three, _ := pathOf(t, leaves, 0, 3)
	extra := sha256.Sum256([]byte("extra"))
	for name, proof := range map[string]InclusionProof{
		"one short":            {LeafIndex: 0, TreeSize: 3, Path: three.Path[:1]},
		"one long":             {LeafIndex: 0, TreeSize: 3, Path: append(slices.Clone(three.Path), extra)},
		"none of two":          {LeafIndex: 0, TreeSize: 2},
		"a node for one":       {LeafIndex: 0, TreeSize: 1, Path: [][32]byte{extra}},
		"short at 2^40 + 1":    {LeafIndex: 1 << 40, TreeSize: 1<<40 + 1},
		"another index's path": {LeafIndex: 2, TreeSize: 3, Path: three.Path},
	} {
		if got, err := Walk(leaves[0], proof); !errors.Is(err, ErrWrongPathLength) || got != [32]byte{} {
			t.Errorf("%s: %x, %v, want no root and ErrWrongPathLength", name, got, err)
		}
	}
}

// Every inclusion row of the sound proofs' traces walks from the leaf
// recomputed from its record (H(0x11) of an item's record, H(0x21) of an
// observation's, H(0x31) of a block's) to the merkle_root it declares and
// recomputes. Mutant killed: ReadInclusionProof reading the path in
// reverse.
func TestWalk_InclusionRowsOfTheSoundProofs(t *testing.T) {
	walked := 0
	for _, sound := range soundProofs(t) {
		for _, row := range sound.trace {
			part, _ := row["part"].(string)
			if row["construction"] != "inclusion" || lastSegment(part) != "merkle_root" {
				continue
			}
			var proofAt, recordAt string
			for _, input := range inputs(t, row) {
				if strings.HasSuffix(input, "/inclusion_proof") {
					proofAt = input
				} else {
					recordAt = input
				}
			}
			kind := ObservationHash
			switch {
			case strings.HasSuffix(recordAt, "block/record"):
				kind = BlockHash
			case recordAt == "/subject/record" && sound.proof["type"] == "item":
				kind = ItemHash
			}
			leaf, err := RecordHash(kind, pointer(t, sound.proof, recordAt).(map[string]any))
			if err != nil {
				t.Fatal(err)
			}
			proof, err := ReadInclusionProof(pointer(t, sound.proof, proofAt))
			if err != nil {
				t.Fatalf("%s %s: %v", sound.name, proofAt, err)
			}
			root, err := Walk(leaf, proof)
			if err != nil || hex.EncodeToString(root[:]) != row["recomputed"] || row["recomputed"] != row["declared"] {
				t.Errorf("%s %s: the walk gives %x, %v; the trace recomputes %v", sound.name, part, root, err, row["recomputed"])
			}
			walked++
		}
	}
	if walked < 30 {
		t.Fatalf("walked %d inclusion proofs of the sound proofs", walked)
	}
}

// A carried MERKLE_PROOF is read only when its leaf_index and tree_size are
// INTs and its path an array of HEX64 nodes. Mutants killed: readINT
// taking a fractional number's integer part; readINT accepting a number
// above MAX_INTEGER; DecodeHex64 accepting uppercase.
func TestReadInclusionProof_RefusesAValueThatIsNotAMerkleProof(t *testing.T) {
	node := `"` + strings.Repeat("ab", 32) + `"`
	good := `{"leaf_index": 0, "path": [` + node + `], "tree_size": 2}`
	proof, err := ReadInclusionProof(mustParse(t, "proof", []byte(good)))
	if err != nil || proof.LeafIndex != 0 || proof.TreeSize != 2 || len(proof.Path) != 1 || proof.Path[0] != [32]byte(bytes.Repeat([]byte{0xab}, 32)) {
		t.Fatalf("ReadInclusionProof(%s) = %+v, %v", good, proof, err)
	}
	for text, want := range map[string]error{
		`[]`:                                ErrNotInclusionProof,
		`"proof"`:                           ErrNotInclusionProof,
		`{"path": [], "tree_size": 1}`:      ErrNotInclusionProof,
		`{"leaf_index": 0, "tree_size": 1}`: ErrNotInclusionProof,
		`{"leaf_index": 0, "path": []}`:     ErrNotInclusionProof,
		`{"leaf_index": "0", "path": [], "tree_size": 1}`:                                 ErrNotInclusionProof,
		`{"leaf_index": -1, "path": [], "tree_size": 1}`:                                  ErrNotInclusionProof,
		`{"leaf_index": 0.5, "path": [], "tree_size": 1}`:                                 ErrNotInclusionProof,
		`{"leaf_index": 0, "path": [], "tree_size": 1e300}`:                               ErrNotInclusionProof,
		`{"leaf_index": 0, "path": [], "tree_size": 1e21}`:                                ErrNotInclusionProof,
		`{"leaf_index": 0, "path": {}, "tree_size": 1}`:                                   ErrNotInclusionProof,
		`{"leaf_index": 0, "path": [null], "tree_size": 2}`:                               ErrInvalidNode,
		`{"leaf_index": 0, "path": [7], "tree_size": 2}`:                                  ErrInvalidNode,
		`{"leaf_index": 0, "path": ["` + strings.Repeat("AB", 32) + `"], "tree_size": 2}`: ErrInvalidNode,
	} {
		if got, err := ReadInclusionProof(mustParse(t, "proof", []byte(text))); !errors.Is(err, want) {
			t.Errorf("ReadInclusionProof(%s) = %+v, %v, want %v", text, got, err, want)
		}
	}
	at, err := ReadInclusionProof(mustParse(t, "proof", []byte(`{"leaf_index": 9007199254740990, "path": [], "tree_size": 9007199254740991}`)))
	if err != nil || at.LeafIndex != strictjson.MaxInteger-1 || at.TreeSize != strictjson.MaxInteger {
		t.Errorf("MAX_INTEGER is an INT: %+v, %v", at, err)
	}
}

// HEX64 is exactly 64 characters of 0 to 9 and a to f; every other
// spelling is refused whole, among them each the known answers name. Mutant
// killed: hexDigit accepting 'g' (its lowercase range widened by one).
func TestDecodeHex64_TakesOnlyHEX64(t *testing.T) {
	good := strings.Repeat("0123456789abcdef", 4)
	got, err := DecodeHex64(good)
	if err != nil || hex.EncodeToString(got[:]) != good {
		t.Fatalf("DecodeHex64(%s) = %x, %v", good, got, err)
	}
	base := good[:62]
	for _, text := range []string{
		"", good[:63], good + "0", good + good, strings.ToUpper(good),
		base + "0g", base + "0/", base + "0:", base + "0`", base + "  ", base + "zz", base + "0\x00",
		base + "0G", base + "0@", " " + good[:63], good[:63] + "\n",
	} {
		if got, err := DecodeHex64(text); !errors.Is(err, ErrNotHex64) || got != [32]byte{} {
			t.Errorf("DecodeHex64(%q) = %x, %v, want ErrNotHex64", text, got, err)
		}
	}
}
