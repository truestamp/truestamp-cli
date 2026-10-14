// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: Apache-2.0

// This file ports the reference verifier's Truestamp.Verifier.Binary64
// (elixir/lib/truestamp/verifier/binary64.ex in truestamp-verification):
// the nearest binary64 value of a JSON number, rounding to nearest with ties
// to even (SPEC.md section 10.1, J5), computed with integer arithmetic only.
//
// Go's strconv.ParseFloat is correctly rounded for a spelling of a few dozen
// digits, but its decimal fallback keeps only the first 800 digits and
// places the decimal point by the digits it kept, so a spelling of more than
// 800 digits before its point can read wrongly whenever ParseFloat's faster
// path gives up and that fallback runs, even when its exponent makes up for
// its length: one, written as a 1, 800 zeros and e-800 (806 octets), reads
// as 0.1. The same 800-digit cut is why one written as a 1, 262,000
// zeros and e-262000 reads as 0. ParseFloat also stops reading an exponent
// once it passes 10,000, which matters only from an exponent of six digits.
// MAX_READ_OCTETS (SPEC.md section 9) admits such spellings, so this file
// does not call ParseFloat.
//
// Adapted from the reference: the same four steps and constants, with
// math/big in place of Elixir's integers; once the division of step 4 is
// made, the quotient has at most 57 bits, so the rounding works on uint64;
// and a spelling of at most 15 significant digits and a power of ten of at
// most 22 is read by Clinger's fast path, one correctly rounded IEEE 754
// multiplication or division of two exact values, which gives the same bits
// as the exact path.

package strictjson

import (
	"math"
	"math/big"
	"math/bits"
)

// The constants of the reference's steps 2 to 4.
const (
	// keptDigits is how many significant digits step 3 keeps before the
	// sticky digit.
	keptDigits = 800
	// maxPoint and minPoint are step 2's limits on P: 10^309 is above
	// 2^1024 - 2^970, and 10^-324 is below 2^-1075.
	maxPoint = 309
	minPoint = -323
	// maxExponentDigits is step 2's limit: an exponent of more digits than
	// this decides overflow or zero alone, since no spelling has 10^15
	// digits.
	maxExponentDigits = 15
	// quotientBits is the length step 4's division aims at: the quotient
	// has 56 or 57 bits.
	quotientBits = 56
	// significandBits is binary64's significand, its hidden bit included.
	significandBits = 53
	// minLSB is the exponent of the smallest subnormal, 2^-1074.
	minLSB = -1074
	// maxLSB is the exponent of the last bit of the largest finite value,
	// (2^53 - 1) x 2^971.
	maxLSB = 971
	// fastDigits and fastPower bound Clinger's fast path: 10^15 is below
	// 2^53, and 10^22 is the largest power of ten a binary64 holds exactly.
	fastDigits = 15
	fastPower  = 22
)

// fastPowers holds 10^0 to 10^22, each exact in binary64.
var fastPowers = [fastPower + 1]float64{
	1e0, 1e1, 1e2, 1e3, 1e4, 1e5, 1e6, 1e7, 1e8, 1e9, 1e10, 1e11,
	1e12, 1e13, 1e14, 1e15, 1e16, 1e17, 1e18, 1e19, 1e20, 1e21, 1e22,
}

// spelling is a JSON number taken apart as step 1 takes it.
type spelling struct {
	negative bool
	// digits are the significant digits S: the integer and fraction digits
	// with their leading zeros dropped. Empty means the value is zero.
	digits []byte
	// point is P before the exponent: the magnitude is 0.S x 10^P.
	point int64
	// exponentNegative and exponentDigits are the exponent's sign and its
	// digits with their leading zeros dropped.
	exponentNegative bool
	exponentDigits   []byte
}

// nearest returns the binary64 value nearest text, ties to even, and true;
// or false when that rounding gives an infinity, that is, when the
// magnitude is at least 2^1024 - 2^970. An underflow is a zero, and -0 and
// every negative number that rounds to zero give -0. text must match RFC
// 8259's number grammar; the reader checks that before it calls nearest,
// and anything else here reads as false.
func nearest(text []byte) (float64, bool) {
	s, ok := split(text)
	if !ok {
		return 0, false
	}
	return s.round(true)
}

// split is step 1. It reports false for text outside the number grammar.
func split(text []byte) (spelling, bool) {
	var s spelling
	i := 0
	if i < len(text) && text[i] == '-' {
		s.negative = true
		i++
	}
	intStart := i
	switch {
	case i < len(text) && text[i] == '0':
		i++
	case i < len(text) && '1' <= text[i] && text[i] <= '9':
		i = runEnd(text, i)
	default:
		return s, false
	}
	intEnd := i
	fracStart, fracEnd := i, i
	if i < len(text) && text[i] == '.' {
		fracStart = i + 1
		fracEnd = runEnd(text, fracStart)
		if fracEnd == fracStart {
			return s, false
		}
		i = fracEnd
	}
	if i < len(text) && (text[i] == 'e' || text[i] == 'E') {
		i++
		if i < len(text) && (text[i] == '+' || text[i] == '-') {
			s.exponentNegative = text[i] == '-'
			i++
		}
		end := runEnd(text, i)
		if end == i {
			return s, false
		}
		s.exponentDigits = dropZeros(text[i:end])
		i = end
	}
	if i != len(text) {
		return s, false
	}

	all := make([]byte, 0, (intEnd-intStart)+(fracEnd-fracStart))
	all = append(all, text[intStart:intEnd]...)
	all = append(all, text[fracStart:fracEnd]...)
	s.digits = dropZeros(all)
	zeros := len(all) - len(s.digits)
	s.point = int64(intEnd-intStart) - int64(zeros)
	return s, true
}

// runEnd returns the index just past the run of digits that starts at i.
func runEnd(text []byte, i int) int {
	for i < len(text) && '0' <= text[i] && text[i] <= '9' {
		i++
	}
	return i
}

// dropZeros returns digits without their leading zeros.
func dropZeros(digits []byte) []byte {
	i := 0
	for i < len(digits) && digits[i] == '0' {
		i++
	}
	return digits[i:]
}

// round is step 2, then steps 3 and 4. fast allows Clinger's fast path.
func (s spelling) round(fast bool) (float64, bool) {
	if len(s.digits) == 0 {
		return build(s.negative, 0, 0), true
	}

	if len(s.exponentDigits) > maxExponentDigits {
		if s.exponentNegative {
			return build(s.negative, 0, 0), true
		}
		return 0, false
	}
	var magnitude int64
	for _, c := range s.exponentDigits {
		magnitude = magnitude*10 + int64(c-'0')
	}
	p := s.point + magnitude
	if s.exponentNegative {
		p = s.point - magnitude
	}
	switch {
	case p > maxPoint:
		return 0, false
	case p < minPoint:
		return build(s.negative, 0, 0), true
	}

	if fast && len(s.digits) <= fastDigits {
		if q := p - int64(len(s.digits)); -fastPower <= q && q <= fastPower {
			return clinger(s.negative, s.digits, q), true
		}
	}
	return roundScaled(s.negative, s.digits, p)
}

// clinger reads digits x 10^q, with at most 15 digits and |q| at most 22, as
// one correctly rounded operation on two exact binary64 values.
func clinger(negative bool, digits []byte, q int64) float64 {
	var m uint64
	for _, c := range digits {
		m = m*10 + uint64(c-'0')
	}
	value := float64(m)
	if q >= 0 {
		value *= fastPowers[q]
	} else {
		value /= fastPowers[-q]
	}
	if negative {
		value = -value
	}
	return value
}

// roundScaled is steps 3 and 4: the value is M x 10^Q, divided by the
// power of two 2^e that leaves a quotient of 56 or 57 bits.
func roundScaled(negative bool, digits []byte, p int64) (float64, bool) {
	m, count := mantissa(digits)
	q := p - count

	numerator := m
	denominator := big.NewInt(1)
	ten := big.NewInt(10)
	if q >= 0 {
		numerator = new(big.Int).Mul(m, new(big.Int).Exp(ten, big.NewInt(q), nil))
	} else {
		denominator = new(big.Int).Exp(ten, big.NewInt(-q), nil)
	}

	e := numerator.BitLen() - denominator.BitLen() - quotientBits
	quotient, sticky := divide(numerator, denominator, e)
	return roundBinary(negative, quotient, sticky, e)
}

// mantissa is step 3: the first 800 significant digits as an integer, and a
// sticky digit 1 after them when a later digit is not zero, with the count
// of digits the integer holds.
func mantissa(digits []byte) (*big.Int, int64) {
	if len(digits) <= keptDigits {
		return decimalInt(digits), int64(len(digits))
	}
	m := decimalInt(digits[:keptDigits])
	if len(dropZeros(digits[keptDigits:])) == 0 {
		return m, keptDigits
	}
	m.Mul(m, big.NewInt(10))
	m.Add(m, big.NewInt(1))
	return m, keptDigits + 1
}

// decimalInt reads a run of ASCII digits as an integer.
func decimalInt(digits []byte) *big.Int {
	n, ok := new(big.Int).SetString(string(digits), 10)
	if !ok {
		// digits are ASCII digits by construction, so SetString cannot
		// refuse them.
		panic("strictjson: not a run of digits")
	}
	return n
}

// divide returns floor(numerator / (denominator x 2^e)), which has 56 or 57
// bits, and whether the division dropped anything.
func divide(numerator, denominator *big.Int, e int) (uint64, bool) {
	n, d := numerator, denominator
	if e >= 0 {
		d = new(big.Int).Lsh(denominator, uint(e))
	} else {
		n = new(big.Int).Lsh(numerator, uint(-e))
	}
	quotient, remainder := new(big.Int).QuoRem(n, d, new(big.Int))
	return quotient.Uint64(), remainder.Sign() != 0
}

// roundBinary is step 4's rounding: the value is (quotient + f) x 2^e, with
// 0 <= f < 1 and f > 0 exactly when sticky. lsb is the exponent of the last
// bit kept: 53 bits, or 2^-1074 at the subnormals. The quotient has at least
// 56 bits, so at least three are dropped.
func roundBinary(negative bool, quotient uint64, sticky bool, e int) (float64, bool) {
	lsb := max(e+bits.Len64(quotient)-significandBits, minLSB)
	shift := uint(lsb - e)
	kept := quotient >> shift
	dropped := quotient & (1<<shift - 1)
	half := uint64(1) << (shift - 1)

	up := dropped > half || (dropped == half && (sticky || kept&1 == 1))
	significand := kept
	if up {
		significand++
	}
	// Rounding up from 2^53 - 1 gives 2^53, which is 2^52 at the next
	// exponent.
	if significand == 1<<significandBits {
		significand = 1 << (significandBits - 1)
		lsb++
	}
	if lsb > maxLSB {
		return 0, false
	}
	return build(negative, significand, lsb), true
}

// build returns the binary64 value of sign, significand and lsb: a normal
// value when the significand has 53 bits, and otherwise a subnormal or zero,
// whose lsb is -1074 and biased exponent 0.
func build(negative bool, significand uint64, lsb int) float64 {
	const normal = uint64(1) << (significandBits - 1)
	var biased, fraction uint64
	if significand >= normal {
		biased = uint64(lsb - minLSB + 1) // #nosec G115 -- lsb is at least minLSB and at most maxLSB on this branch, so the biased exponent is 1 to 2046
		fraction = significand - normal
	} else {
		fraction = significand
	}
	var sign uint64
	if negative {
		sign = 1
	}
	return math.Float64frombits(sign<<63 | biased<<52 | fraction)
}
