// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: Apache-2.0

package bitcoin

import (
	"encoding/binary"
	"math/big"
	"testing"
)

// targetOf reads a target written as hex.
func targetOf(t *testing.T, text string) *big.Int {
	t.Helper()
	n, ok := new(big.Int).SetString(text, 16)
	if !ok {
		t.Fatalf("%q is not hex", text)
	}
	return n
}

// The target an nBits value encodes (SPEC.md section 10.2, step 9 (d)), at
// each edge of its rules: the shift for an exponent of at most 3, applied
// before the sign test (0x02800100's mantissa shifts to 1 and is negative,
// 0x01800100's to 0 and is not); a negative encoding only when the
// mantissa is not 0; each overflow bound and the value just inside it; and
// the targets of Bitcoin's mainnet and regtest. A negative or overflowing
// encoding is refused, never read as a target. Mutants killed: the sign
// test made before the shift (0x01800100 is negative); the sign test
// without w != 0 (0x04800000 is negative); each overflow clause left out
// or its bound moved by one (0x23000001, 0x22000100 or 0x21010000 passes,
// or 0x220000ff or 0x2100ffff overflows); the shift for e <= 3 left out
// (0x03123456's neighbours differ).
func TestTarget_TheEncoding(t *testing.T) {
	cases := []struct {
		n      uint32
		target string
		err    error
	}{
		{0x1d00ffff, "ffff0000000000000000000000000000000000000000000000000000", nil},
		{0x207fffff, "7fffff0000000000000000000000000000000000000000000000000000000000", nil},
		{0x03123456, "123456", nil},
		{0x02123456, "1234", nil},
		{0x01123456, "12", nil},
		{0x00123456, "0", nil},
		{0x04800000, "0", nil},
		{0x01800100, "0", nil},
		{0x23000000, "0", nil},
		{0x04800001, "", ErrNegativeTarget},
		{0x02800100, "", ErrNegativeTarget},
		{0x20ffffff, "", ErrNegativeTarget},
		{0x23000001, "", ErrTargetOverflow},
		{0x220000ff, "ff" + "00000000000000000000000000000000000000000000000000000000000000", nil},
		{0x22000100, "", ErrTargetOverflow},
		{0x2100ffff, "ffff" + "000000000000000000000000000000000000000000000000000000000000", nil},
		{0x21010000, "", ErrTargetOverflow},
		{0x21008000, "8000" + "000000000000000000000000000000000000000000000000000000000000", nil},
	}
	for _, c := range cases {
		target, err := Target(c.n)
		if err != c.err {
			t.Errorf("%#08x: %v, want %v", c.n, err, c.err)
			continue
		}
		if c.err == nil && target.Cmp(targetOf(t, c.target)) != 0 {
			t.Errorf("%#08x: target %x, want %s", c.n, target, c.target)
		}
	}
}

// reversedOf writes n as a 32-octet hash in the order it is hashed in:
// rev of its big-endian bytes.
func reversedOf(n *big.Int) [32]byte {
	var displayed, out [32]byte
	n.FillBytes(displayed[:])
	for i, b := range displayed {
		out[31-i] = b
	}
	return out
}

// A header meets its target when rev of its hash, read as a big-endian
// integer, is not above the target, and fails when the encoding is
// negative or overflows or the target is 0, whatever its hash (SPEC.md
// section 10.2, step 9 (d)). Mutants killed: the comparison made strictly
// below (a hash equal to the target fails); the hash read without its
// reversal (the target's own hash, which is small only as displayed,
// fails); a target of 0 met by a hash of 0.
func TestMeetsTarget_ComparesTheDisplayedHash(t *testing.T) {
	target := targetOf(t, "ffff0000000000000000000000000000000000000000000000000000")
	one := big.NewInt(1)
	cases := []struct {
		name string
		hash [32]byte
		n    uint32
		want bool
	}{
		{"the target itself", reversedOf(target), 0x1d00ffff, true},
		{"one below the target", reversedOf(new(big.Int).Sub(target, one)), 0x1d00ffff, true},
		{"one above the target", reversedOf(new(big.Int).Add(target, one)), 0x1d00ffff, false},
		{"0 under a target of 0", [32]byte{}, 0x1d000000, false},
		{"0 under a negative encoding", [32]byte{}, 0x04800001, false},
		{"0 under an overflowing encoding", [32]byte{}, 0x23000001, false},
		{"1 under a target of 1", reversedOf(one), 0x03000001, true},
		{"2 under a target of 1", reversedOf(big.NewInt(2)), 0x03000001, false},
	}
	for _, c := range cases {
		if got := meetsTarget(c.hash, c.n); got != c.want {
			t.Errorf("%s: %v, want %v", c.name, got, c.want)
		}
	}
}

// A real header meets its target, and the same header under a target of 1
// does not: regtest's genesis header with its nBits set to 0x03000001.
// Mutant killed: MeetsTarget reading nBits from another octet (the
// genesis headers fail, or the changed one passes).
func TestMeetsTarget_ReadsTheHeadersNBits(t *testing.T) {
	for _, f := range corpus(t) {
		if f.Kind != "header" {
			continue
		}
		h := Header(bytesOf(t, f.Hex))
		if !MeetsTarget(h) {
			t.Errorf("%s does not meet its target", f.name)
		}
		binary.LittleEndian.PutUint32(h[72:], 0x03000001)
		if MeetsTarget(h) || NBits(h) != 0x03000001 {
			t.Errorf("%s under a target of 1 meets it", f.name)
		}
	}
}
