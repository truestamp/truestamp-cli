// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: Apache-2.0

package verifier

import (
	"errors"
	"fmt"
	"math/rand"
	"testing"
	"time"

	"github.com/truestamp/truestamp-cli/internal/specdata"
)

// TIME and NISTTIME (SPEC.md section 2.4), as the reference's
// time_test.exs holds them: the pattern, each field's range at its limit
// and one past it, the round trip against a lenient parse made with Go's
// time.Date, formatting against Go's time package, and the times the
// sound proofs carry.

// parseOutcome writes what ParseTime gives for value.
func parseOutcome(class TimeClass, value any) string {
	instant, err := ParseTime(class, value)
	switch {
	case errors.Is(err, ErrTimePattern):
		return "pattern"
	case errors.Is(err, ErrTimeRoundTrip):
		return "round_trip"
	case err != nil:
		return err.Error()
	}
	return fmt.Sprint(instant)
}

// Each class takes its one layout and no other: no offset but Z, no lower
// case t or z, no field without its leading zeros, no fraction on a TIME,
// exactly three digits of one on a NISTTIME, nothing before or after,
// ASCII digits only, and only a string. The patterns are
// spec/encodings.json's, which asks for the round trip. Mutants killed:
// the patterns compiled without \A and \z (a final newline, a leading
// space and a trailing Z pass); the pattern matched with (?i) (a lower
// case t or z passes).
func TestTime_ThePatternOfEachClass(t *testing.T) {
	for class, want := range map[TimeClass]string{ClassTIME: "TIME", ClassNISTTIME: "NISTTIME"} {
		encoding := specdata.Encodings()[want]
		if class.String() != want || TimePattern(class) != encoding.Pattern || !encoding.RoundTrip {
			t.Errorf("%s: pattern %q, spec/encodings.json %+v", want, TimePattern(class), encoding)
		}
	}
	for _, c := range []struct {
		class TimeClass
		text  string
		want  string
	}{
		{ClassTIME, "1970-01-01T00:00:00Z", "0"},
		{ClassNISTTIME, "1970-01-01T00:00:00.000Z", "0"},
		{ClassTIME, "2026-09-28T12:34:56Z", "1790598896"},
		{ClassNISTTIME, "2026-09-28T12:34:56.789Z", "1790598896789"},
	} {
		if got := parseOutcome(c.class, c.text); got != c.want {
			t.Errorf("%s %q: %s, want %s", c.class, c.text, got, c.want)
		}
	}
	for _, text := range []string{
		"2026-09-28T12:34:56", "2026-09-28t12:34:56Z", "2026-09-28T12:34:56z", "2026-09-28 12:34:56Z",
		"2026-09-28T12:34:56+00:00", "2026-09-28T12:34:56-05:00", "2026-09-28T12:34:56+0000",
		"2026-09-28T12:34:56.789Z", "2026-09-28T12:34:56.Z", "2026-09-28T12:34Z",
		"26-09-28T12:34:56Z", "02026-09-28T12:34:56Z", "2026-9-28T12:34:56Z", "2026-09-8T12:34:56Z",
		"2026-09-28T1:34:56Z", "2026-09-28T12:4:56Z", "2026-09-28T12:34:6Z",
		"2026-09-28T12:34:56Z\n", "\n2026-09-28T12:34:56Z", " 2026-09-28T12:34:56Z", "2026-09-28T12:34:56Z ",
		"2026-09-28T12:34:56ZZ", "+2026-09-28T12:34:56Z", "2026-09-28T12:34:5aZ",
		"2026０09-28T12:34:56Z", "２026-09-28T12:34:56Z", "２０２６-09-28T12:34:56Z", "",
	} {
		if got := parseOutcome(ClassTIME, text); got != "pattern" {
			t.Errorf("TIME %q: %s, want pattern", text, got)
		}
	}
	for _, text := range []string{
		"2026-09-28T12:34:56Z", "2026-09-28T12:34:56.78Z", "2026-09-28T12:34:56.7890Z", "2026-09-28T12:34:56,789Z",
		"2026-09-28T12:34:56.789Z\n", "2026-09-28T12:34:56.789z", "2026-09-28T12:34:56.789+00:00", "2026-09-28T12:34:56.7Z",
	} {
		if got := parseOutcome(ClassNISTTIME, text); got != "pattern" {
			t.Errorf("NISTTIME %q: %s, want pattern", text, got)
		}
	}
	for _, value := range []any{nil, 0, []byte("1970-01-01T00:00:00Z"), map[string]any{}} {
		if got := parseOutcome(ClassTIME, value); got != "pattern" {
			t.Errorf("TIME %#v: %s, want pattern", value, got)
		}
	}
	if got := parseOutcome(TimeClass(7), "1970-01-01T00:00:00Z"); got != "pattern" {
		t.Errorf("an unknown class: %s", got)
	}
}

// Each field at its limit and one past it, in the proleptic Gregorian
// calendar: the hour from 00 to 23, the minute and second from 00 to 59
// with no leap second, the month from 01 to 12, the day from 01 to the
// last of its month, with the last second of every month of a common and
// a leap year, February 29 in a leap year (2024, 2000, 2400 and 0000) and
// not in a common one (2026, and the centuries 1900, 2100, 2200 and
// 2300), and the years 0000 to 9999; and a NISTTIME's fraction from .000
// to .999. Mutants killed: leapYear without the century clause (1900-02-29
// passes); leapYear without the 400-year clause (2000-02-29 fails); the
// hour allowed 24 (24:00:00 passes); the second allowed 60 (the leap
// second 2016-12-31T23:59:60Z passes); daysInMonth giving April 31 days
// (2026-04-31 passes); the month allowed 0 (2026-00-01 passes).
func TestTime_EachFieldAtItsLimitAndOnePast(t *testing.T) {
	cases := map[string]string{
		"2026-09-28T23:59:59Z": "ok", "2026-09-28T24:00:00Z": "round_trip",
		"2026-09-28T12:59:59Z": "ok", "2026-09-28T12:60:00Z": "round_trip",
		"2026-09-28T12:34:60Z": "round_trip", "2016-12-31T23:59:60Z": "round_trip", "2015-06-30T23:59:60Z": "round_trip",
		"2026-01-01T00:00:00Z": "ok", "2026-12-31T00:00:00Z": "ok",
		"2026-00-01T00:00:00Z": "round_trip", "2026-13-01T00:00:00Z": "round_trip", "2026-99-01T00:00:00Z": "round_trip",
		"2026-01-00T00:00:00Z": "round_trip", "2026-01-32T00:00:00Z": "round_trip", "2026-02-30T00:00:00Z": "round_trip",
		"2024-02-29T00:00:00Z": "ok", "2000-02-29T00:00:00Z": "ok", "2400-02-29T00:00:00Z": "ok", "0000-02-29T00:00:00Z": "ok",
		"2026-02-29T00:00:00Z": "round_trip", "1900-02-29T00:00:00Z": "round_trip", "2100-02-29T00:00:00Z": "round_trip",
		"2200-02-29T00:00:00Z": "round_trip", "2300-02-29T00:00:00Z": "round_trip", "2024-02-30T00:00:00Z": "round_trip",
		"0000-01-01T00:00:00Z": "ok", "9999-12-31T23:59:59Z": "ok",
	}
	lastDays := []int{31, 28, 31, 30, 31, 30, 31, 31, 30, 31, 30, 31}
	for year, leap := range map[int]bool{2026: false, 2024: true} {
		for month, last := range lastDays {
			if leap && month == 1 {
				last = 29
			}
			cases[fmt.Sprintf("%04d-%02d-%02dT23:59:59Z", year, month+1, last)] = "ok"
			cases[fmt.Sprintf("%04d-%02d-%02dT00:00:00Z", year, month+1, last+1)] = "round_trip"
		}
	}
	for text, want := range cases {
		got := parseOutcome(ClassTIME, text)
		if want == "ok" && (got == "pattern" || got == "round_trip") || want != "ok" && got != want {
			t.Errorf("TIME %q: %s, want %s", text, got, want)
		}
		nist := text[:19] + ".000Z"
		if gotNIST := parseOutcome(ClassNISTTIME, nist); (want == "ok") != (gotNIST != "pattern" && gotNIST != "round_trip") {
			t.Errorf("NISTTIME %q: %s, want %s", nist, gotNIST, want)
		}
	}
	for text, want := range map[string]string{
		"0000-01-01T00:00:00Z": "-62167219200", "9999-12-31T23:59:59Z": "253402300799",
		"2024-02-29T23:59:59Z": "1709251199", "2026-12-31T23:59:59Z": "1798761599",
	} {
		if got := parseOutcome(ClassTIME, text); got != want {
			t.Errorf("TIME %q: %s, want %s", text, got, want)
		}
	}
	for text, want := range map[string]string{
		"1970-01-01T00:00:00.999Z": "999", "1969-12-31T23:59:59.001Z": "-999", "1970-01-01T00:00:00.001Z": "1",
		"9999-12-31T23:59:59.999Z": "253402300799999", "0000-01-01T00:00:00.000Z": "-62167219200000",
	} {
		if got := parseOutcome(ClassNISTTIME, text); got != want {
			t.Errorf("NISTTIME %q: %s, want %s", text, got, want)
		}
	}
}

// lenientRoundTrip reports whether the fields, each a number of at most
// its pattern's digits, write back to themselves when read leniently: Go's
// time.Date carries a field out of its range into the next, and the
// instant it gives is written back with Go's own Format.
func lenientRoundTrip(fields [6]int) bool {
	instant := time.Date(fields[0], time.Month(fields[1]), fields[2], fields[3], fields[4], fields[5], 0, time.UTC)
	written := fmt.Sprintf("%04d-%02d-%02dT%02d:%02d:%02d", fields[0], fields[1], fields[2], fields[3], fields[4], fields[5])
	return instant.Year() >= 0 && instant.Year() <= 9999 && instant.Format("2006-01-02T15:04:05") == written
}

// ParseTime refuses exactly the strings a lenient parse moves, over random
// fields of the pattern, in both classes, and gives the instant the
// lenient parse gives every other. Mutants killed: the minute allowed 60
// (12:60:00 passes, and a lenient parse moves it); daysFromCivil with
// truncating division (a date before 0000-03-01 gives another instant
// than Go's time.Date, in January and February of year 0000).
func TestTime_RefusesExactlyWhatALenientParseMoves(t *testing.T) {
	random := rand.New(rand.NewSource(789))
	moved := 0
	for range 20000 {
		fields := [6]int{random.Intn(10000), random.Intn(15), random.Intn(33), random.Intn(26), random.Intn(62), random.Intn(62)}
		if random.Intn(4) == 0 {
			fields[0] = random.Intn(3)
		}
		text := fmt.Sprintf("%04d-%02d-%02dT%02d:%02d:%02d", fields[0], fields[1], fields[2], fields[3], fields[4], fields[5])
		lenient := lenientRoundTrip(fields)
		if !lenient {
			moved++
		}
		for class, suffix := range map[TimeClass]string{ClassTIME: "Z", ClassNISTTIME: ".123Z"} {
			instant, err := ParseTime(class, text+suffix)
			switch {
			case lenient && err != nil, !lenient && !errors.Is(err, ErrTimeRoundTrip):
				t.Fatalf("%s %q: %v; the lenient parse moves it: %v", class, text+suffix, err, !lenient)
			case lenient:
				want := time.Date(fields[0], time.Month(fields[1]), fields[2], fields[3], fields[4], fields[5], 0, time.UTC).Unix()
				if class == ClassNISTTIME {
					want = want*1000 + 123
				}
				if instant != want {
					t.Fatalf("%s %q: %d, want %d", class, text+suffix, instant, want)
				}
			}
		}
	}
	if moved < 5000 || moved > 19000 {
		t.Fatalf("%d of 20000 strings moved: the draw does not test both ways", moved)
	}
}

// FormatTime writes an instant in its class's pattern, a negative one
// included, from the first second of year 0000 to the last of year 9999,
// and refuses one outside them; 2^32 - 1 seconds, the largest Bitcoin
// nTime, writes. Mutants killed: FormatTime dividing an instant by 86400
// with truncation (-1 writes no time of 1969-12-31); the range check left
// out (253402300800 writes 10000-01-01);
// civilFromDays with the month of March and later shifted by one (each
// date of the year moves).
func TestTime_FormatTime(t *testing.T) {
	for _, c := range []struct {
		class   TimeClass
		instant int64
		want    string
	}{
		{ClassTIME, 0, "1970-01-01T00:00:00Z"},
		{ClassTIME, 1790598896, "2026-09-28T12:34:56Z"},
		{ClassTIME, -1, "1969-12-31T23:59:59Z"},
		{ClassTIME, 4294967295, "2106-02-07T06:28:15Z"},
		{ClassTIME, firstTimeSecond, "0000-01-01T00:00:00Z"},
		{ClassTIME, lastTimeSecond, "9999-12-31T23:59:59Z"},
		{ClassTIME, 951782400, "2000-02-29T00:00:00Z"},
		{ClassTIME, -62162121600, "0000-02-29T00:00:00Z"},
		{ClassNISTTIME, 1790598896007, "2026-09-28T12:34:56.007Z"},
		{ClassNISTTIME, -1, "1969-12-31T23:59:59.999Z"},
		{ClassNISTTIME, -1000, "1969-12-31T23:59:59.000Z"},
		{ClassNISTTIME, lastTimeSecond*1000 + 999, "9999-12-31T23:59:59.999Z"},
	} {
		if got, ok := FormatTime(c.class, c.instant); !ok || got != c.want {
			t.Errorf("%s %d: %q, %v, want %q", c.class, c.instant, got, ok, c.want)
		}
	}
	for _, c := range []struct {
		class   TimeClass
		instant int64
	}{
		{ClassTIME, firstTimeSecond - 1}, {ClassTIME, lastTimeSecond + 1},
		{ClassNISTTIME, firstTimeSecond*1000 - 1}, {ClassNISTTIME, (lastTimeSecond + 1) * 1000},
	} {
		if got, ok := FormatTime(c.class, c.instant); ok {
			t.Errorf("%s %d: %q, want refused", c.class, c.instant, got)
		}
	}
}

// FormatTime agrees with Go's own calendar on random instants across the
// years 0000 to 9999, and ParseTime reads each string back to its instant,
// in both classes. Mutant killed: civilFromDays's era taken with
// truncating division (every date before 0000-03-01 moves, and the
// instants of year 0000 drawn here disagree with Go's time).
func TestTime_FormatAgreesWithGosCalendar(t *testing.T) {
	random := rand.New(rand.NewSource(456))
	for i := range 20000 {
		seconds := firstTimeSecond + random.Int63n(lastTimeSecond-firstTimeSecond+1)
		if i%10 == 0 {
			seconds = firstTimeSecond + random.Int63n(366*86400)
		}
		text, ok := FormatTime(ClassTIME, seconds)
		if want := time.Unix(seconds, 0).UTC().Format("2006-01-02T15:04:05Z"); !ok || text != want {
			t.Fatalf("%d: %q, %v, want %q", seconds, text, ok, want)
		}
		if back, err := ParseTime(ClassTIME, text); err != nil || back != seconds {
			t.Fatalf("%q: %d, %v, want %d", text, back, err, seconds)
		}
		milliseconds := seconds*1000 + random.Int63n(1000)
		nist, ok := FormatTime(ClassNISTTIME, milliseconds)
		if back, err := ParseTime(ClassNISTTIME, nist); !ok || err != nil || back != milliseconds {
			t.Fatalf("%d: %q, %d, %v", milliseconds, nist, back, err)
		}
	}
}

// Every TIME and NISTTIME value the sound proofs carry, header_time,
// closed_at and timeStamp, parses and formats back to itself. Mutant
// killed: FormatTime writing a NISTTIME's fraction without its leading
// zeros (each sound timeStamp's .000 writes back as .0).
func TestTime_TheSoundProofsTimesWriteBack(t *testing.T) {
	seen := map[string]int{}
	for name, proof := range positiveProofs(t) {
		for _, n := range nodes(proof) {
			if len(n.at) == 0 || n.at[len(n.at)-1].isIndex {
				continue
			}
			member := n.at[len(n.at)-1].name
			class := map[string]TimeClass{"header_time": ClassTIME, "closed_at": ClassTIME, "timeStamp": ClassNISTTIME}
			c, timed := class[member]
			if !timed {
				continue
			}
			seen[member]++
			instant, err := ParseTime(c, n.value)
			if text, ok := FormatTime(c, instant); err != nil || !ok || text != n.value {
				t.Errorf("%s %s: %v, %q", name, n.at.Encode(), err, text)
			}
		}
	}
	if seen["header_time"] == 0 || seen["closed_at"] == 0 || seen["timeStamp"] == 0 {
		t.Fatalf("times seen: %v", seen)
	}
}
