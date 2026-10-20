// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: Apache-2.0

// An independent model of the Bitcoin evidence of DERIVE (SPEC.md section
// 10.2, step 9 (d)), for the runner's model of DERIVE (derive_model_test.go).
// It shares no code with internal/verifier or internal/verifier/bitcoin:
// it reads each structure through a cursor of its own, expands the flag
// octets of a partial Merkle tree into bits before its walk, computes the
// target an nBits value encodes from its mantissa's bytes, formats a
// header's nTime with Go's time package, and hashes with crypto/sha256 and
// takes each epoch_hash from the model's own record hash, so a defect in
// the verifier's parsers cannot hide in the values it is compared with.
// It writes each rule a structure breaks by the names
// vectors/corpus/bitcoin/ gives a raw_transaction's (section 13.2) and,
// for a txoutproof's, which SPEC.md does not name, by the clause of step 9
// (d) each breaks, in the order the step lists them.

package vectorcases_test

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"math/big"
	"slices"
	"testing"
	"time"
)

// modelCursor reads a byte string from its front; once a read runs out,
// short is set and every later read gives nothing.
type modelCursor struct {
	b     []byte
	i     int
	short bool
}

// next reads n octets.
func (c *modelCursor) next(n uint64) []byte {
	if c.short || n > uint64(len(c.b)-c.i) {
		c.short = true
		return nil
	}
	out := c.b[c.i : c.i+int(n)]
	c.i += int(n)
	return out
}

// size reads a compact size, marking the cursor short when it is not in
// its shortest form.
func (c *modelCursor) size() uint64 {
	first := c.next(1)
	if c.short {
		return 0
	}
	widths := map[byte]uint64{0xfd: 2, 0xfe: 4, 0xff: 8}
	width, wide := widths[first[0]]
	if !wide {
		return uint64(first[0])
	}
	octets := c.next(width)
	if c.short {
		return 0
	}
	var n uint64
	for k := len(octets) - 1; k >= 0; k-- {
		n = n<<8 | uint64(octets[k])
	}
	if n < map[uint64]uint64{2: 0xfd, 4: 0x10000, 8: 0x100000000}[width] {
		c.short = true
	}
	return n
}

// done reports whether the cursor read every octet and never ran out.
func (c *modelCursor) done() bool { return !c.short && c.i == len(c.b) }

// modelSHA256d is SHA-256(SHA-256(b)).
func modelSHA256d(b []byte) []byte {
	first := sha256.Sum256(b)
	second := sha256.Sum256(first[:])
	return second[:]
}

// modelDisplay writes b reversed, as hex.
func modelDisplay(b []byte) string {
	reversed := slices.Clone(b)
	slices.Reverse(reversed)
	return hex.EncodeToString(reversed)
}

// modelTransaction judges raw as a raw_transaction: the 32 bytes its one
// OP_RETURN pushes, or the first rule it breaks.
func modelTransaction(raw []byte) (pushed []byte, rule string) {
	c := &modelCursor{b: raw}
	c.next(4)
	inputs := c.size()
	if c.short || inputs == 0 {
		return nil, "parse"
	}
	for k := uint64(0); k < inputs && !c.short; k++ {
		c.next(36)
		c.next(c.size())
		c.next(4)
	}
	outputs := c.size()
	var scripts [][]byte
	for k := uint64(0); k < outputs && !c.short; k++ {
		c.next(8)
		scripts = append(scripts, c.next(c.size()))
	}
	c.next(4)
	if !c.done() {
		return nil, "parse"
	}
	if len(raw) == 64 {
		return nil, "length"
	}
	var returns [][]byte
	for _, s := range scripts {
		if len(s) >= 1 && s[0] == 0x6a {
			returns = append(returns, s)
		}
	}
	if len(returns) != 1 || len(returns[0]) != 34 || returns[0][1] != 0x20 {
		return nil, "op_return"
	}
	return returns[0][2:], ""
}

// modelBlock is what a well-formed txoutproof gives the model.
type modelBlock struct {
	header  []byte
	total   uint32
	matched []byte
}

// modelMerkleBlock judges b as a txoutproof: what it gives, or the first
// rule it breaks.
func modelMerkleBlock(b []byte) (modelBlock, string) {
	c := &modelCursor{b: b}
	header := c.next(80)
	count := c.next(4)
	var hashes [][]byte
	n := c.size()
	if !c.short && n <= uint64(len(b)) {
		for k := uint64(0); k < n && !c.short; k++ {
			hashes = append(hashes, c.next(32))
		}
	} else {
		c.short = true
	}
	flagOctets := c.next(c.size())
	if !c.done() {
		return modelBlock{}, "parse"
	}
	total := binary.LittleEndian.Uint32(count)
	if total == 0 || total > 16666 {
		return modelBlock{}, "total_txs"
	}
	var bits []bool
	for _, octet := range flagOctets {
		for k := range 8 {
			bits = append(bits, octet&(1<<k) != 0)
		}
	}
	if len(hashes) > int(total) || len(bits) < len(hashes) {
		return modelBlock{}, "partial_tree"
	}
	widthAt := func(h uint) uint64 { return (uint64(total) + (uint64(1) << h) - 1) >> h }
	var height uint
	for widthAt(height) > 1 {
		height++
	}
	usedBits, usedHashes := 0, 0
	var matches [][]byte
	rule := ""
	var node func(h uint, pos uint64) []byte
	node = func(h uint, pos uint64) []byte {
		if rule != "" {
			return nil
		}
		if usedBits == len(bits) {
			rule = "partial_tree"
			return nil
		}
		bit := bits[usedBits]
		usedBits++
		if h == 0 || !bit {
			if usedHashes == len(hashes) {
				rule = "partial_tree"
				return nil
			}
			hash := hashes[usedHashes]
			usedHashes++
			if h == 0 && bit {
				matches = append(matches, hash)
			}
			return hash
		}
		left := node(h-1, 2*pos)
		right := left
		if 2*pos+1 < widthAt(h-1) {
			right = node(h-1, 2*pos+1)
			if rule == "" && slices.Equal(left, right) {
				rule = "duplicate_pair"
			}
		}
		if rule != "" {
			return nil
		}
		return modelSHA256d(append(slices.Clone(left), right...))
	}
	root := node(height, 0)
	if rule != "" {
		return modelBlock{}, rule
	}
	if usedHashes != len(hashes) || (usedBits+7)/8 != len(flagOctets) || slices.Contains(bits[usedBits:], true) {
		return modelBlock{}, "partial_tree"
	}
	if len(matches) != 1 {
		return modelBlock{}, "matches"
	}
	if !slices.Equal(root, header[36:68]) {
		return modelBlock{}, "merkle_root"
	}
	if !modelMeetsTarget(header) {
		return modelBlock{}, "target"
	}
	return modelBlock{header: header, total: total, matched: matches[0]}, ""
}

// modelMeetsTarget is the target rule of step 9 (d): the nBits value's
// mantissa, its 3 octets below the sign bit, placed as the top octets of a
// number of exponent octets, and the header's hash in display order, read
// as a big-endian integer, not above it.
func modelMeetsTarget(header []byte) bool {
	n := binary.LittleEndian.Uint32(header[72:76])
	exponent := int(n >> 24)
	mantissa := []byte{byte(n >> 16 & 0x7f), byte(n >> 8), byte(n)}
	value := new(big.Int).SetBytes(mantissa)
	if exponent <= 3 {
		value.Rsh(value, uint(8*(3-exponent)))
	}
	if value.Sign() != 0 && n&0x00800000 != 0 {
		return false
	}
	if value.Sign() != 0 && value.BitLen()+8*(exponent-3) > 256 {
		return false
	}
	if exponent > 3 {
		value.Lsh(value, uint(8*(exponent-3)))
	}
	hash, _ := hex.DecodeString(modelDisplay(modelSHA256d(header)))
	return value.Sign() != 0 && new(big.Int).SetBytes(hash).Cmp(value) <= 0
}

// modelBitcoin is the model's checks of (d) for the Bitcoin entry at n, in
// the table's order.
func modelBitcoin(t *testing.T, proof map[string]any, n string) []modelCheck {
	t.Helper()
	decode := func(part string) []byte {
		b, err := hex.DecodeString(get(proof, part).(string))
		if err != nil {
			t.Fatalf("%s: %v", part, err)
		}
		return b
	}
	raw, txoutproof := decode(n+"/raw_transaction"), decode(n+"/txoutproof")
	txid := modelDisplay(modelSHA256d(raw))
	wellFormed := func(part, rule string) modelCheck {
		m := modelCheck{part: part, construction: "bitcoin", inputs: []string{part}, status: "pass"}
		if rule != "" {
			m.status, m.code, m.rule = "fail", "bitcoin_evidence_invalid", rule
		}
		return m
	}
	untraced := func(m modelCheck) modelCheck {
		m.traced = false
		return m
	}
	var out []modelCheck
	pushed, rule := modelTransaction(raw)
	out = append(out, wellFormed(n+"/raw_transaction", rule))
	if rule == "" {
		out = append(out, untraced(equal(n+"/raw_transaction", "bitcoin", sortedPointers(n+"/epoch/record", n+"/raw_transaction"),
			hex.EncodeToString(pushed), modelHash(t, "epoch_hash", get(proof, n+"/epoch/record")), "derivation_mismatch")))
	}
	out = append(out, equal(n+"/transaction_hash", "bitcoin", []string{n + "/raw_transaction"}, get(proof, n+"/transaction_hash"), txid, "derivation_mismatch"))
	block, rule := modelMerkleBlock(txoutproof)
	out = append(out, wellFormed(n+"/txoutproof", rule))
	if rule != "" {
		return out
	}
	header := func(member string, recomputed any) modelCheck {
		return equal(n+"/"+member, "bitcoin", []string{n + "/txoutproof"}, get(proof, n+"/"+member), recomputed, "derivation_mismatch")
	}
	nTime := int64(binary.LittleEndian.Uint32(block.header[68:72]))
	return append(out,
		untraced(equal(n+"/txoutproof", "bitcoin", sortedPointers(n+"/raw_transaction", n+"/txoutproof"), modelDisplay(block.matched), txid, "derivation_mismatch")),
		header("header_hash", modelDisplay(modelSHA256d(block.header))),
		header("header_merkle_root", modelDisplay(block.header[36:68])),
		header("header_time", time.Unix(nTime, 0).UTC().Format("2006-01-02T15:04:05Z")),
	)
}
