// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: Apache-2.0

package ed25519

import (
	"encoding/hex"
	"errors"
	"testing"

	"github.com/truestamp/truestamp-cli/internal/specdata"
)

func key(t *testing.T, text string) [32]byte {
	t.Helper()
	decoded, err := hex.DecodeString(text)
	if err != nil || len(decoded) != 32 {
		t.Fatalf("%q is not 32 bytes of hex", text)
	}
	return [32]byte(decoded)
}

// Every encoding of SMALL_ORDER_KEYS, the sign-bit variants included, is
// refused at step 3. Mutant killed: the SMALL_ORDER_KEYS loop left out.
func TestCheckKeyBytes_RefusesEverySmallOrderKey(t *testing.T) {
	keys := specdata.SmallOrderKeys()
	if len(keys) != 10 {
		t.Fatalf("%d small-order keys, want 10", len(keys))
	}
	for _, small := range keys {
		if err := CheckKeyBytes(small.Key); !errors.Is(err, ErrSmallOrderKey) {
			t.Errorf("%x: %v, want small_order_key", small.Key, err)
		}
	}
}

// Step 2 at its edge: y equal to p is refused, with the sign bit clear or
// set, and so is the largest y; y = p - 2 passes (it need not be a point:
// the curve is SIGNATURE's to check); and a key whose sign bit is set
// passes when its y is below p. No small-order key has y at or above p, so
// no key tells the two steps' order apart. Mutants killed: y compared with
// > (y = p passes); bit 255 not cleared (VECTOR_KEY with its sign bit set
// is refused).
func TestCheckKeyBytes_YBelowP(t *testing.T) {
	cases := []struct {
		name string
		key  string
		want error
	}{
		{"y = p", "edffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff7f", ErrYNotBelowP},
		{"y = p, sign bit set", "edffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff", ErrYNotBelowP},
		{"y = 2^255 - 1", "ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff7f", ErrYNotBelowP},
		{"y = p + 1", "eeffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff7f", ErrYNotBelowP},
		{"y = p - 2", "ebffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff7f", nil},
		{"VECTOR_KEY", specdata.VectorKey, nil},
		{"VECTOR_KEY, sign bit set", "2152f8d19b791d24453242e15f2eab6cb7cffa7b6a5ed30097960e069881db92", nil},
	}
	for _, c := range cases {
		if err := CheckKeyBytes(key(t, c.key)); !errors.Is(err, c.want) || (c.want == nil && err != nil) {
			t.Errorf("%s: %v, want %v", c.name, err, c.want)
		}
	}
	for _, small := range specdata.SmallOrderKeys() {
		if y(small.Key).Cmp(p) >= 0 {
			t.Errorf("small-order key %x has y at or above p", small.Key)
		}
	}
}

// p is 2^255 - 19, written as RFC 8032 writes it in hex. Mutant killed: p
// built as 2^255 + 19.
func TestP_IsTheFieldPrime(t *testing.T) {
	if got := p.Text(16); got != "7fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffed" {
		t.Fatalf("p = %s", got)
	}
}
