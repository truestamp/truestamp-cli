// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: Apache-2.0

package bitcoin

import (
	"bytes"
	"encoding/hex"
	"testing"
)

// output is one output of a transaction the tests build: its 8-octet
// value and its script.
type output struct{ script []byte }

// transaction serializes a transaction without witness data, with inputs
// inputs, each with an empty script, and the outputs given; each count is
// a one-octet compact size.
func transaction(inputs int, outputs ...output) []byte {
	var b bytes.Buffer
	b.Write([]byte{2, 0, 0, 0})
	b.WriteByte(byte(inputs))
	for i := range inputs {
		b.Write(bytes.Repeat([]byte{byte(i + 1)}, 36))
		b.WriteByte(0)
		b.Write([]byte{0xff, 0xff, 0xff, 0xfd})
	}
	b.WriteByte(byte(len(outputs)))
	for _, o := range outputs {
		b.Write(make([]byte, 8))
		b.WriteByte(byte(len(o.script)))
		b.Write(o.script)
	}
	b.Write([]byte{0, 0, 0, 0})
	return b.Bytes()
}

// pushOf is the script 0x6a 0x20 followed by 32 bytes of fill.
func pushOf(fill byte) []byte {
	return append([]byte{0x6a, 0x20}, bytes.Repeat([]byte{fill}, 32)...)
}

// payTo is a script that does not begin with 0x6a.
var payTo = append([]byte{0x00, 0x14}, bytes.Repeat([]byte{0xaa}, 20)...)

// The rules of a well-formed raw_transaction (SPEC.md section 10.2, step 9
// (d)), each refused by name, and the one OP_RETURN's 32 bytes given when
// they hold. A witness serialization is refused, never stripped; so are
// bytes after the lock time and a transaction of 64 octets or with no
// input; and every OP_RETURN script is read, never only the first, so a
// second OP_RETURN after a good first one is refused, as is an OP_RETURN
// that pushes its 32 bytes with OP_PUSHDATA1 or pushes another count.
// Mutants killed: the trailing-bytes check left out; the at least one
// input rule left out (the no-input and witness transactions parse); the
// 64-octet rule left out; the first OP_RETURN taken (the two-OP_RETURN
// transactions pass); the OP_RETURN script's length not checked (the
// 31-byte push is not refused); the push opcode not checked (0x6a 0x21
// with 32 bytes passes); the length and op_return rules checked in the
// other order (the 64-octet transaction, which has no OP_RETURN, fails
// op_return).
func TestParseTransaction_TheRules(t *testing.T) {
	good := transaction(1, output{payTo}, output{pushOf(0x11)})
	sixtyFour := transaction(1, output{[]byte{1, 2, 3, 4}})
	witness := append(append(append([]byte{}, good[:4]...), 0x00, 0x01), good[4:len(good)-4]...)
	witness = append(append(witness, 0x01, 0x00), good[len(good)-4:]...)
	nonMinimal := append(append([]byte{}, good[:4]...), append([]byte{0xfd, 0x01, 0x00}, good[5:]...)...)
	cases := []struct {
		name string
		raw  []byte
		want error
	}{
		{"well formed", good, nil},
		{"the OP_RETURN first", transaction(1, output{pushOf(0x11)}, output{payTo}), nil},
		{"two inputs", transaction(2, output{pushOf(0x11)}), nil},
		{"a witness serialization", witness, RefusedParse},
		{"a byte after the lock time", append(append([]byte{}, good...), 0x00), RefusedParse},
		{"no lock time", good[:len(good)-1], RefusedParse},
		{"no input", transaction(0, output{pushOf(0x11)}), RefusedParse},
		{"a non-minimal count of inputs", nonMinimal, RefusedParse},
		{"a script longer than the bytes", append(append([]byte{}, good[:len(good)-4-34]...), 0xff), RefusedParse},
		{"empty", nil, RefusedParse},
		{"exactly 64 octets", sixtyFour, RefusedLength},
		{"64 octets and a trailing byte", append(append([]byte{}, sixtyFour...), 0x00), RefusedParse},
		{"no OP_RETURN", transaction(1, output{payTo}), RefusedOpReturn},
		{"no output", transaction(1), RefusedOpReturn},
		{"two OP_RETURNs", transaction(1, output{pushOf(0x11)}, output{pushOf(0x22)}), RefusedOpReturn},
		{"a good OP_RETURN, then a bare one", transaction(1, output{pushOf(0x11)}, output{[]byte{0x6a}}), RefusedOpReturn},
		{"OP_PUSHDATA1", transaction(1, output{append([]byte{0x6a, 0x4c}, pushOf(0x11)[1:]...)}), RefusedOpReturn},
		{"31 bytes", transaction(1, output{pushOf(0x11)[:33]}), RefusedOpReturn},
		{"33 bytes", transaction(1, output{append(pushOf(0x11), 0x11)}), RefusedOpReturn},
		{"another push opcode", transaction(1, output{append([]byte{0x6a, 0x21}, pushOf(0x11)[2:]...)}), RefusedOpReturn},
	}
	if len(sixtyFour) != 64 {
		t.Fatalf("the 64-octet transaction is %d octets", len(sixtyFour))
	}
	for _, c := range cases {
		pushed, err := ParseTransaction(c.raw)
		if err != c.want {
			t.Errorf("%s: %v, want %v", c.name, err, c.want)
			continue
		}
		want := [32]byte{}
		if c.want == nil {
			copy(want[:], bytes.Repeat([]byte{0x11}, 32))
		}
		if pushed != want {
			t.Errorf("%s: pushed %s", c.name, hex.EncodeToString(pushed[:]))
		}
	}
}
