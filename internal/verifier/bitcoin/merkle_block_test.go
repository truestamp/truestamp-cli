// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: Apache-2.0

package bitcoin

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"testing"
)

// leaf is a 32-octet hash of fill.
func leaf(fill byte) [32]byte {
	var h [32]byte
	for i := range h {
		h[i] = fill
	}
	return h
}

// pair is SHA-256d(left || right), computed here.
func pair(left, right [32]byte) [32]byte {
	first := sha256.Sum256(append(left[:], right[:]...))
	return sha256.Sum256(first[:])
}

// regtestHeader is regtest's genesis header, whose nBits, 0x207fffff,
// any hash whose displayed first octet is below 0x7f meets.
func regtestHeader(t *testing.T) Header {
	t.Helper()
	for _, f := range corpus(t) {
		if f.name == "regtest-0-header.json" {
			return Header(bytesOf(t, f.Hex))
		}
	}
	t.Fatal("no regtest-0-header.json")
	return Header{}
}

// headerWith is regtest's genesis header with root as its Merkle root
// field, nBits n, and the first nonce from 0 whose hash, as displayed,
// begins below 0x7f, searched here with crypto/sha256.
func headerWith(t *testing.T, root [32]byte, n uint32) Header {
	t.Helper()
	h := regtestHeader(t)
	copy(h[36:68], root[:])
	binary.LittleEndian.PutUint32(h[72:], n)
	for nonce := uint32(0); ; nonce++ {
		binary.LittleEndian.PutUint32(h[76:], nonce)
		first := sha256.Sum256(h[:])
		if second := sha256.Sum256(first[:]); second[31] < 0x7f {
			return h
		}
	}
}

// merkleBlock serializes a txoutproof: header, total, a one-octet count
// and the hashes, a one-octet count and the flag octets.
func merkleBlock(header Header, total uint32, hashes [][32]byte, flags []byte) []byte {
	var b bytes.Buffer
	b.Write(header[:])
	_ = binary.Write(&b, binary.LittleEndian, total)
	b.WriteByte(byte(len(hashes)))
	for _, h := range hashes {
		b.Write(h[:])
	}
	b.WriteByte(byte(len(flags)))
	b.Write(flags)
	return b.Bytes()
}

// The rules of a well-formed txoutproof (SPEC.md section 10.2, step 9 (d)),
// each refused by name, on trees of two, three and four leaves A, B, C, D
// built here: the walk of two leaves matching A; three leaves, whose last
// node pairs C with itself, which is no duplicate pair; and each way to
// break a rule. Every flag bit after the last one the walk uses is read,
// and one set is refused; every header is held to its target. A count of
// hashes past the bytes is refused at parse before anything is allocated,
// however large: the 95-octet txoutproof that claims 2^59 hashes and
// carries none, and the 127-octet one that claims 2^59+1 and carries one,
// each a count that, at 32 octets a hash, comes modulo 2^64 to the octets
// of the hashes it carries. More hashes than total_txs are refused before
// the walk, as Bitcoin Core refuses them: three hashes, [A, A, C], over two
// leaves are refused as a partial tree, never as the duplicate pair the
// walk would meet. Mutants killed: the unused-bit check left out (0x83
// passes); the octet count check left out (the extra flag octet passes);
// the unused-hash check left out (the fourth hash of four passes); the
// duplicate-pair check left out ([A, A] passes) or made on a node with no
// right child (the three-leaf tree fails); the match count taken as at
// least one (two matches pass); the root not compared (another root
// passes); the target not checked (a target of 1 passes); total_txs' bounds
// moved (0 or 16,667 passes, or 16,666 is refused for total_txs); 16,666
// refused by a rule before its partial tree (parse); the hash-count guard
// left out (the 95-octet and 127-octet txoutproofs panic with makeslice:
// len out of range); the refusal before the walk left out ([A, A, C] gives
// duplicate_pair).
func TestParseMerkleBlock_TheRules(t *testing.T) {
	a, b, c, d := leaf(0xa1), leaf(0xb2), leaf(0xc3), leaf(0xd4)
	two := pair(a, b)
	good := headerWith(t, two, 0x207fffff)
	three := pair(pair(a, b), pair(c, c))
	four := pair(pair(a, b), pair(c, d))
	withTrailing := append(merkleBlock(good, 2, [][32]byte{a, b}, []byte{0x03}), 0x00)
	nonMinimal := merkleBlock(good, 2, [][32]byte{a, b}, []byte{0x03})
	nonMinimal = append(append(append([]byte{}, nonMinimal[:84]...), 0xfd, 0x02, 0x00), nonMinimal[85:]...)
	// claiming is a txoutproof of two leaves whose count of hashes is 0xff
	// followed by the 8 octets count, then the hashes present and the one
	// flag octet 0x03.
	claiming := func(count []byte, present ...[32]byte) []byte {
		out := append(append([]byte{}, good[:]...), 0x02, 0x00, 0x00, 0x00, 0xff)
		out = append(out, count...)
		for _, h := range present {
			out = append(out, h[:]...)
		}
		return append(out, 0x01, 0x03)
	}
	none := claiming([]byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x08})
	one := claiming([]byte{0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x08}, a)
	if len(none) != 95 || len(one) != 127 {
		t.Fatalf("the txoutproofs claiming 2^59 hashes are %d and %d octets", len(none), len(one))
	}
	cases := []struct {
		name    string
		b       []byte
		want    error
		matched [32]byte
	}{
		{"two leaves, A matched", merkleBlock(good, 2, [][32]byte{a, b}, []byte{0x03}), nil, a},
		{"three leaves, C matched, paired with itself", merkleBlock(headerWith(t, three, 0x207fffff), 3, [][32]byte{pair(a, b), c}, []byte{0x0d}), nil, c},
		{"a byte after the flags", withTrailing, RefusedParse, [32]byte{}},
		{"a non-minimal count of hashes", nonMinimal, RefusedParse, [32]byte{}},
		{"a count of hashes past the bytes", merkleBlock(good, 2, [][32]byte{a, b}, nil)[:84+1+32], RefusedParse, [32]byte{}},
		{"2^59 hashes claimed, none present", none, RefusedParse, [32]byte{}},
		{"2^59+1 hashes claimed, one present", one, RefusedParse, [32]byte{}},
		{"a header cut short", good[:79], RefusedParse, [32]byte{}},
		{"total_txs 0", merkleBlock(good, 0, [][32]byte{a, b}, []byte{0x03}), RefusedTotalTxs, [32]byte{}},
		{"total_txs 16,666", merkleBlock(good, 16666, [][32]byte{a, b}, []byte{0x03}), RefusedPartialTree, [32]byte{}},
		{"total_txs 16,667", merkleBlock(good, 16667, [][32]byte{a, b}, []byte{0x03}), RefusedTotalTxs, [32]byte{}},
		{"an unused flag bit set", merkleBlock(good, 2, [][32]byte{a, b}, []byte{0x83}), RefusedPartialTree, [32]byte{}},
		{"an unused flag octet", merkleBlock(good, 2, [][32]byte{a, b}, []byte{0x03, 0x00}), RefusedPartialTree, [32]byte{}},
		{"more hashes than leaves", merkleBlock(good, 2, [][32]byte{a, b, c}, []byte{0x03}), RefusedPartialTree, [32]byte{}},
		{"more hashes than leaves, the first two a duplicate pair", merkleBlock(good, 2, [][32]byte{a, a, c}, []byte{0x03}), RefusedPartialTree, [32]byte{}},
		{"an unused hash", merkleBlock(headerWith(t, four, 0x207fffff), 4, [][32]byte{a, b, pair(c, d), d}, []byte{0x07}), RefusedPartialTree, [32]byte{}},
		{"too few hashes", merkleBlock(good, 2, [][32]byte{a}, []byte{0x03}), RefusedPartialTree, [32]byte{}},
		{"a duplicate pair", merkleBlock(headerWith(t, pair(a, a), 0x207fffff), 2, [][32]byte{a, a}, []byte{0x03}), RefusedDuplicatePair, [32]byte{}},
		{"two matches", merkleBlock(good, 2, [][32]byte{a, b}, []byte{0x07}), RefusedMatches, [32]byte{}},
		{"no match", merkleBlock(good, 2, [][32]byte{a, b}, []byte{0x01}), RefusedMatches, [32]byte{}},
		{"another root", merkleBlock(headerWith(t, pair(b, a), 0x207fffff), 2, [][32]byte{a, b}, []byte{0x03}), RefusedMerkleRoot, [32]byte{}},
		{"a target of 1", merkleBlock(headerWith(t, two, 0x03000001), 2, [][32]byte{a, b}, []byte{0x03}), RefusedTarget, [32]byte{}},
	}
	for _, c := range cases {
		block, err := ParseMerkleBlock(c.b)
		if err != c.want {
			t.Errorf("%s: %v, want %v", c.name, err, c.want)
			continue
		}
		if block.Matched != c.matched || (c.want == nil && (block.TotalTxs == 0 || block.Header != Header(c.b[:80]))) {
			t.Errorf("%s: %+v", c.name, block)
		}
	}
}
