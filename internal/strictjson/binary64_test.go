// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: Apache-2.0

// Tests of the nearest binary64 value of J5 (SPEC.md section 10.1), with
// exact rational arithmetic (math/big) as the answers' check.

package strictjson

import (
	"errors"
	"math"
	"math/big"
	"math/rand/v2"
	"strconv"
	"strings"
	"testing"
)

// dyadic returns the exact decimal spelling of m x 2^k, which terminates.
func dyadic(m *big.Int, k int) string {
	if k >= 0 {
		return new(big.Int).Lsh(m, uint(k)).String()
	}
	// m x 2^k = m x 5^-k / 10^-k.
	scaled := new(big.Int).Mul(m, new(big.Int).Exp(big.NewInt(5), big.NewInt(int64(-k)), nil)).String()
	places := -k
	if len(scaled) <= places {
		scaled = strings.Repeat("0", places-len(scaled)+1) + scaled
	}
	return scaled[:len(scaled)-places] + "." + scaled[len(scaled)-places:]
}

func two(k int) *big.Int { return new(big.Int).Lsh(big.NewInt(1), uint(k)) }

// overflowBoundary is 2^1024 - 2^970, the magnitude from which J5 refuses a
// number, as a decimal integer.
func overflowBoundary() string {
	return new(big.Int).Sub(two(1024), two(970)).String()
}

// dyadicMinusOne returns the decimal integer text minus one.
func dyadicMinusOne(text string) string {
	n, _ := new(big.Int).SetString(text, 10)
	return n.Sub(n, big.NewInt(1)).String()
}

// mustNearest reads text, failing on an overflow.
func mustNearest(t *testing.T, text string) float64 {
	t.Helper()
	f, ok := nearest([]byte(text))
	if !ok {
		t.Fatalf("nearest(%q) overflowed", abbreviate(text))
	}
	return f
}

func sameBits(a, b float64) bool { return math.Float64bits(a) == math.Float64bits(b) }

// nearestExact is nearest without the fast path, for the tests that hold
// the two paths to one answer.
func nearestExact(text []byte) (float64, bool) {
	s, ok := split(text)
	if !ok {
		return 0, false
	}
	return s.round(false)
}

// Halfway points round to the even neighbour, and anything past one, by
// however little and however far out, rounds away from it. Mutants killed:
// roundBinary rounding a tie up (kept&1 == 1 replaced by true), roundBinary
// ignoring sticky, keptDigits lowered to 40 (a sticky digit past the 40th
// would be lost), and roundBinary without its carry to the next exponent.
func TestNearest_TiesToEven(t *testing.T) {
	one := big.NewInt(1)
	three := big.NewInt(3)
	halfAboveOne := dyadic(new(big.Int).Add(two(53), one), -53) // 1 + 2^-53
	for _, tc := range []struct {
		about string
		text  string
		want  float64
	}{
		{"1 + 2^-53, a tie, goes down to the even 1", halfAboveOne, 1},
		{"1 + 3 x 2^-53, a tie, goes up to the even 1 + 2^-51", dyadic(new(big.Int).Add(two(53), three), -53), 1 + 0x1p-51},
		{"just past 1 + 2^-53 goes up", halfAboveOne + "0000000000000000000001", 1 + 0x1p-52},
		{"just below 1 + 2^-53 goes down", dyadicMinusOne(strings.ReplaceAll(halfAboveOne, ".", "")) + "e-53", 1},
		{"past the tie by a digit after the 800th goes up", halfAboveOne + strings.Repeat("0", 900) + "1", 1 + 0x1p-52},
		{"the tie followed by 900 zeros is still a tie", halfAboveOne + strings.Repeat("0", 900), 1},
		{"2^-1075, a tie between 0 and 2^-1074, goes to 0", dyadic(one, -1075), 0},
		{"just past 2^-1075 goes to 2^-1074", dyadic(one, -1075) + "000001", 0x1p-1074},
		{"3 x 2^-1075 goes up to the even 2^-1073", dyadic(three, -1075), 0x1p-1073},
		{"the tie below 2^-1022 goes up from the largest subnormal", dyadic(new(big.Int).Sub(two(53), one), -1075), 0x1p-1022},
		{"2 - 2^-53, a tie, carries to 2", dyadic(new(big.Int).Sub(two(54), one), -53), 2},
		{"2^1024 - 2^970 - 1 reads as the largest value", dyadicMinusOne(overflowBoundary()), math.MaxFloat64},
	} {
		if got := mustNearest(t, tc.text); !sameBits(got, tc.want) {
			t.Fatalf("%s: nearest = %v (%x), want %v (%x)", tc.about, got, math.Float64bits(got), tc.want, math.Float64bits(tc.want))
		}
		if got, _ := nearestExact([]byte(tc.text)); !sameBits(got, tc.want) {
			t.Fatalf("%s: the exact path gives %v", tc.about, got)
		}
	}
	if _, ok := nearest([]byte(overflowBoundary())); ok {
		t.Fatal("2^1024 - 2^970 did not overflow")
	}
}

// A spelling whose length its exponent makes up for reads as its value,
// though strconv.ParseFloat misreads it. Mutants killed: split leaving the
// leading zeros of the fraction out of P (one spelled 0., 261,999 zeros, a 1
// and e262000 would read as 10^261999 and overflow), and nearest calling
// ParseFloat when the exponent has at most four digits (one spelled 1, 800
// zeros and e-800 would read as 0.1).
func TestNearest_LongSpellings(t *testing.T) {
	zeros := strings.Repeat("0", 262000)
	for _, tc := range []struct {
		about string
		text  string
		want  float64
	}{
		{"one as a 1, 800 zeros and e-800", "1" + zeros[:800] + "e-800", 1},
		{"one as a 1, 262,000 zeros and e-262000", "1" + zeros + "e-262000", 1},
		{"one as 0., 261,999 zeros, a 1 and e262000", "0." + zeros[1:] + "1e262000", 1},
		{"1.5 with 262,000 zeros after it", "1.5" + zeros, 1.5},
		{"a fraction of 262,000 zeros and a 1", "0." + zeros + "1", 0},
		{"an exponent with leading zeros", "1e" + strings.Repeat("0", 30) + "2", 100},
		{"a negative exponent of 16 digits", "1e-1000000000000000", 0},
		{"zero with an exponent of 20 digits", "0e99999999999999999999", 0},
		{"0.0 with 262,000 zeros", "0." + zeros, 0},
	} {
		if got := mustNearest(t, tc.text); !sameBits(got, tc.want) {
			t.Fatalf("%s: nearest = %v, want %v", tc.about, got, tc.want)
		}
	}
	for _, text := range []string{"9" + zeros, "1e1000000000000000", "1" + zeros + "e-261690", "1e+310"} {
		if _, ok := nearest([]byte(text)); ok {
			t.Fatalf("nearest(%q) did not overflow", abbreviate(text))
		}
	}
	if got := mustNearest(t, "1"+zeros+"e-261692"); got != 1e308 {
		t.Fatalf("10^308 spelled long = %v", got)
	}
}

// nearest refuses what is not a JSON number. Mutant killed: split accepting
// a fraction with no digits.
func TestNearest_Grammar(t *testing.T) {
	for _, text := range []string{"", "-", "01", "-01", "1.", ".1", "1e", "1e+", "+1", "1x", "1.5.5", "1e5e5", " 1"} {
		if f, ok := nearest([]byte(text)); ok {
			t.Fatalf("nearest(%q) = %v, want a refusal", text, f)
		}
	}
}

// spellings returns a deterministic sweep of number spellings: random
// digits of a given count with a point and an exponent placed at random.
func spellings(seed uint64, n, maxDigits, maxExponent int) []string {
	r := rand.New(rand.NewPCG(seed, seed^0x5eed))
	out := make([]string, 0, n)
	for range n {
		digits := make([]byte, 1+r.IntN(maxDigits))
		for i := range digits {
			digits[i] = byte('0' + r.IntN(10))
		}
		if len(digits) > 1 && digits[0] == '0' {
			digits[0] = '1' + byte(r.IntN(9))
		}
		text := string(digits)
		if cut := r.IntN(len(digits) + 1); cut > 0 && cut < len(digits) && r.IntN(2) == 0 {
			text = text[:cut] + "." + text[cut:]
		}
		if r.IntN(3) > 0 {
			text += "e" + strconv.Itoa(r.IntN(2*maxExponent+1)-maxExponent)
		}
		if r.IntN(2) == 0 {
			text = "-" + text
		}
		out = append(out, text)
	}
	return out
}

// On a spelling of a few dozen digits, where strconv.ParseFloat is
// correctly rounded, nearest agrees with it bit for bit, overflow included.
// Mutant killed: build with the biased exponent one too large.
func TestNearest_AgreesWithParseFloat(t *testing.T) {
	texts := append(spellings(1, 20000, 40, 340), spellings(2, 5000, 20, 20)...)
	texts = append(texts, "5e-324", "4.9406564584124654e-324", "2.2250738585072011e-308",
		"2.2250738585072014e-308", "1.7976931348623157e308", "0.1", "0.3", "123456789.987654321e-30")
	for _, text := range texts {
		got, ok := nearest([]byte(text))
		want, err := strconv.ParseFloat(text, 64)
		if errors.Is(err, strconv.ErrRange) && math.IsInf(want, 0) {
			if ok {
				t.Fatalf("nearest(%q) = %v, ParseFloat overflows", text, got)
			}
			continue
		}
		if err != nil && !errors.Is(err, strconv.ErrRange) {
			t.Fatalf("ParseFloat(%q): %v", text, err)
		}
		if !ok || !sameBits(got, want) {
			t.Fatalf("nearest(%q) = %v (%x), ParseFloat = %v (%x)", text, got, math.Float64bits(got), want, math.Float64bits(want))
		}
	}
}

// Clinger's fast path gives the exact path's bits. Mutant killed: the fast
// path taking 17 digits (a mantissa above 2^53 is rounded twice).
func TestNearest_FastPathAgreesWithExactPath(t *testing.T) {
	texts := append(spellings(3, 20000, 17, 25), spellings(4, 5000, 15, 22)...)
	texts = append(texts, "9007199254740993e-3", "18014398509481985e-2", "999999999999999e22", "1e-22", "1e22")
	for _, text := range texts {
		fast, okFast := nearest([]byte(text))
		exact, okExact := nearestExact([]byte(text))
		if okFast != okExact || !sameBits(fast, exact) {
			t.Fatalf("%q: fast path %v (%v), exact path %v (%v)", text, fast, okFast, exact, okExact)
		}
	}
}

// exactRat returns the exact value of a finite binary64 f.
func exactRat(f float64) *big.Rat {
	r := new(big.Rat)
	r.SetFloat64(f)
	return r
}

// checkNearest holds f to the rational value of text: no binary64 value is
// nearer, and of two that are equally near f is the even one; or, when
// overflow, the magnitude is at least 2^1024 - 2^970.
func checkNearest(t *testing.T, text string, f float64, ok bool) {
	t.Helper()
	value, valid := new(big.Rat).SetString(text)
	if !valid {
		t.Fatalf("big.Rat cannot read %q", text)
	}
	magnitude := new(big.Rat).Abs(value)
	boundary := new(big.Rat).SetInt(new(big.Int).Sub(two(1024), two(970)))
	if !ok {
		if magnitude.Cmp(boundary) < 0 {
			t.Fatalf("nearest(%q) overflowed below 2^1024 - 2^970", abbreviate(text))
		}
		return
	}
	if magnitude.Cmp(boundary) >= 0 {
		t.Fatalf("nearest(%q) = %v, want an overflow", abbreviate(text), f)
	}
	g := math.Abs(f)
	distance := func(x *big.Rat) *big.Rat { return new(big.Rat).Abs(new(big.Rat).Sub(magnitude, x)) }
	mine := distance(exactRat(g))
	below := math.Nextafter(g, 0)
	neighbours := []*big.Rat{exactRat(below)}
	if above := math.Nextafter(g, math.Inf(1)); !math.IsInf(above, 1) {
		neighbours = append(neighbours, exactRat(above))
	} else {
		neighbours = append(neighbours, new(big.Rat).SetInt(two(1024)))
	}
	for _, neighbour := range neighbours {
		switch distance(neighbour).Cmp(mine) {
		case -1:
			t.Fatalf("nearest(%q) = %v, but a neighbour is nearer", abbreviate(text), f)
		case 0:
			if g != 0 && math.Float64bits(g)&1 == 1 {
				t.Fatalf("nearest(%q) = %v, a tie that went to the odd value", abbreviate(text), f)
			}
		}
	}
	if (value.Sign() < 0) != math.Signbit(f) && g != 0 {
		t.Fatalf("nearest(%q) = %v has the wrong sign", abbreviate(text), f)
	}
}

// An independent oracle, exact rational arithmetic, holds nearest to the
// nearest value, ties to even, on spellings far longer than 800 digits and
// exponents from the subnormals to the overflow. Mutant killed: mantissa
// dropping its sticky digit, so a spelling of more than 800 digits that is
// just past a tie reads as the tie.
func TestNearest_ExactOracle(t *testing.T) {
	texts := append(spellings(5, 300, 1200, 330), spellings(6, 300, 30, 330)...)
	for k := -1076; k <= -1060; k++ {
		texts = append(texts, dyadic(big.NewInt(1), k), dyadic(big.NewInt(3), k), dyadic(big.NewInt(5), k))
	}
	// Halfway points between neighbours, at random, nudged by a digit far out.
	r := rand.New(rand.NewPCG(7, 7))
	for range 200 {
		bits := r.Uint64() &^ (1 << 63)
		f := math.Float64frombits(bits)
		next := math.Nextafter(f, math.Inf(1))
		if math.IsInf(next, 0) || math.IsNaN(f) {
			continue
		}
		mid := new(big.Rat).Add(exactRat(f), exactRat(next))
		mid.Quo(mid, big.NewRat(2, 1))
		if mid.Denom().BitLen() > 1100 {
			continue
		}
		text := mid.FloatString(1100)
		texts = append(texts, text, text+"0000001", strings.TrimRight(text, "0")+strings.Repeat("0", 900)+"1")
	}
	for _, text := range texts {
		f, ok := nearest([]byte(text))
		checkNearest(t, text, f, ok)
	}
}

// FuzzNearest holds nearest to the exact rational oracle on every spelling
// of the number grammar whose exponent has at most four digits (a longer
// one would make big.Rat build a power of ten too large to be quick), and
// the fast path to the exact path. Mutant killed (by the seeds alone, under
// go test): build without its sign bit.
func FuzzNearest(f *testing.F) {
	for _, seed := range []string{
		"0", "-0", "1", "-1.5", "0.1", "1e-400", "-1e-400", "9007199254740993", "5e-324",
		"2.4703282292062327e-324", "1.7976931348623158e308", "1.7976931348623159e308",
		"1.00000000000000011102230246251565404236316680908203125", "123456789012345678901234567890e-20",
	} {
		f.Add(seed)
	}
	f.Fuzz(func(t *testing.T, text string) {
		got, ok := nearest([]byte(text))
		s, valid := split([]byte(text))
		if !valid {
			if ok {
				t.Fatalf("nearest(%q) = %v for a text outside the grammar", text, got)
			}
			return
		}
		exact, okExact := nearestExact([]byte(text))
		if ok != okExact || !sameBits(got, exact) {
			t.Fatalf("%q: fast path %v (%v), exact path %v (%v)", text, got, ok, exact, okExact)
		}
		if len(s.exponentDigits) <= 4 {
			checkNearest(t, text, got, ok)
		}
	})
}
