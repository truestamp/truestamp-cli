// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: Apache-2.0

// This file ports the reference verifier's Truestamp.Verifier.TIME
// (elixir/lib/truestamp/verifier/time.ex in truestamp-verification),
// checked against the Truestamp application's port,
// Truestamp.Proof.Verifier.TIME (lib/truestamp/proof/verifier/time.ex in
// truestamp-v2): the pattern classes TIME and NISTTIME of SPEC.md section
// 2.4 (Pattern classes), RFC 3339 in UTC:
//
//   - TIME is [0-9]{4}-[0-9]{2}-[0-9]{2}T[0-9]{2}:[0-9]{2}:[0-9]{2}Z, and
//     format(parse(s)) = s;
//   - NISTTIME is TIME with a 3-digit fraction of a second before Z.
//
// The patterns are spec/encodings.json's (internal/specdata), each matched
// against the whole string, as \A...\z does; the program stops at start-up
// if they are not the layouts this file takes apart. A TIME value is an
// instant in whole Unix seconds, and a NISTTIME value one in whole Unix
// milliseconds. [ParseTime] gives the instant a string names, and
// [FormatTime] writes an instant in the class's pattern.
//
// The round trip refuses 24:00:00, February 30, a seconds value of 60 and
// February 29 of a year that is not a leap year, such as 1900 or 2100;
// year 0000 is a leap year (SPEC.md section 2.4). [ParseTime] makes it by
// reading each field strictly, in the proleptic Gregorian calendar of the
// years 0000 to 9999: the month from 1 to 12, the day within its month,
// the hour from 0 to 23 and the minute and second from 0 to 59, so there
// is no leap second. A string whose fields are all in range formats back
// to itself, and a lenient parse would move any other to a different
// instant's string, so the two readings refuse exactly the same strings;
// the tests hold ParseTime to the round trip itself, with Go's time.Date,
// which normalizes out-of-range fields, as the lenient parse. No offset but
// Z matches the pattern, so no string here names a local time.
//
// SHAPE's check 5 (SPEC.md section 10.2, step 7) uses this file, and so
// does DERIVE's header_time row (step 9 (d)).
//
// Adapted from the reference: the calendar arithmetic is this file's own
// (Howard Hinnant's day count from a civil date and back, with floor
// division), never Go's time package, whose Format writes a year outside
// 0000 to 9999 in more or fewer digits and whose Parse takes a layout, not
// a pattern; and a regular expression is Go's RE2, whose [0-9] is ASCII
// digits only, as PCRE's is.

package verifier

import (
	"errors"
	"fmt"
	"regexp"

	"github.com/truestamp/truestamp-cli/internal/specdata"
)

// TimeClass is TIME or NISTTIME (SPEC.md section 2.4).
type TimeClass int

// The two classes of a time.
const (
	ClassTIME TimeClass = iota
	ClassNISTTIME
)

// String names the class as SPEC.md section 2.4 does.
func (c TimeClass) String() string {
	if c == ClassNISTTIME {
		return "NISTTIME"
	}
	return "TIME"
}

// The ways a string can fail its class: [ErrTimePattern] when it does not
// match the class's pattern in full, and [ErrTimeRoundTrip] when it
// matches but does not format back to itself.
var (
	ErrTimePattern   = errors.New("the value does not match the pattern of its class")
	ErrTimeRoundTrip = errors.New("the value does not write back to itself")
)

// timeLayouts are the layouts parseTimeFields takes apart, by class.
var timeLayouts = map[TimeClass]string{
	ClassTIME:     `[0-9]{4}-[0-9]{2}-[0-9]{2}T[0-9]{2}:[0-9]{2}:[0-9]{2}Z`,
	ClassNISTTIME: `[0-9]{4}-[0-9]{2}-[0-9]{2}T[0-9]{2}:[0-9]{2}:[0-9]{2}\.[0-9]{3}Z`,
}

// timePatterns are the whole-string patterns of each class, from
// spec/encodings.json.
var timePatterns = mustTimePatterns()

func mustTimePatterns() map[TimeClass]*regexp.Regexp {
	encodings := specdata.Encodings()
	out := map[TimeClass]*regexp.Regexp{}
	for class, layout := range timeLayouts {
		encoding := encodings[class.String()]
		if encoding.Pattern != layout || !encoding.RoundTrip {
			panic(fmt.Sprintf("verifier: spec/encodings.json's %s is not the layout and round trip this verifier reads", class))
		}
		out[class] = regexp.MustCompile(`\A(?:` + layout + `)\z`)
	}
	return out
}

// TimePattern returns the pattern of class, written bare as in SPEC.md
// section 2.4.
func TimePattern(class TimeClass) string { return timeLayouts[class] }

// The instants of the first and the last second the classes can write:
// 0000-01-01T00:00:00Z and 9999-12-31T23:59:59Z, in Unix seconds.
const (
	firstTimeSecond = -62167219200
	lastTimeSecond  = 253402300799
)

// ParseTime returns the instant value names, a string of class: Unix
// seconds for TIME, Unix milliseconds for NISTTIME. It gives
// [ErrTimePattern] for a value that is not a string, or one that does not
// match the class's pattern in full, and [ErrTimeRoundTrip] for one that
// matches but does not format back to itself.
func ParseTime(class TimeClass, value any) (int64, error) {
	text, ok := value.(string)
	pattern, known := timePatterns[class]
	if !ok || !known || !pattern.MatchString(text) {
		return 0, ErrTimePattern
	}
	seconds, err := parseTimeFields(text[:19])
	if err != nil {
		return 0, err
	}
	if class == ClassTIME {
		return seconds, nil
	}
	return seconds*1000 + int64(digits(text[20:23])), nil
}

// parseTimeFields reads YYYY-MM-DDThh:mm:ss, every field digits, as Unix
// seconds, or gives ErrTimeRoundTrip when a field is out of its range.
func parseTimeFields(text string) (int64, error) {
	year, month, day := digits(text[0:4]), digits(text[5:7]), digits(text[8:10])
	hour, minute, second := digits(text[11:13]), digits(text[14:16]), digits(text[17:19])
	if month < 1 || month > 12 || day < 1 || day > daysInMonth(year, month) ||
		hour > 23 || minute > 59 || second > 59 {
		return 0, ErrTimeRoundTrip
	}
	return daysFromCivil(year, month, day)*86400 + int64(hour*3600+minute*60+second), nil
}

// digits reads a run of ASCII digits, which the pattern has already
// matched, as a number.
func digits(text string) int {
	n := 0
	for i := 0; i < len(text); i++ {
		n = n*10 + int(text[i]-'0')
	}
	return n
}

// leapYear reports whether year is a leap year of the proleptic Gregorian
// calendar: divisible by 4, and not by 100 unless by 400, so 0000 and 2000
// are and 1900 and 2100 are not.
func leapYear(year int) bool {
	return year%4 == 0 && (year%100 != 0 || year%400 == 0)
}

// daysInMonth returns the number of days of month in year.
func daysInMonth(year, month int) int {
	switch month {
	case 2:
		if leapYear(year) {
			return 29
		}
		return 28
	case 4, 6, 9, 11:
		return 30
	}
	return 31
}

// floorDiv returns a / b rounded toward negative infinity, for b > 0.
func floorDiv(a, b int64) int64 {
	q := a / b
	if a%b < 0 {
		q--
	}
	return q
}

// daysFromCivil returns the days from 1970-01-01 to the proleptic
// Gregorian date year-month-day (Howard Hinnant's days_from_civil).
func daysFromCivil(year, month, day int) int64 {
	y := int64(year)
	if month <= 2 {
		y--
	}
	era := floorDiv(y, 400)
	yoe := y - era*400
	mp := int64((month + 9) % 12)
	doy := (153*mp+2)/5 + int64(day) - 1
	doe := yoe*365 + yoe/4 - yoe/100 + doy
	return era*146097 + doe - 719468
}

// civilFromDays returns the proleptic Gregorian date days after
// 1970-01-01 (Howard Hinnant's civil_from_days).
func civilFromDays(days int64) (year, month, day int) {
	z := days + 719468
	era := floorDiv(z, 146097)
	doe := z - era*146097
	yoe := (doe - doe/1460 + doe/36524 - doe/146096) / 365
	doy := doe - (365*yoe + yoe/4 - yoe/100)
	mp := (5*doy + 2) / 153
	d := doy - (153*mp+2)/5 + 1
	m := mp + 3
	if mp >= 10 {
		m = mp - 9
	}
	y := yoe + era*400
	if m <= 2 {
		y++
	}
	return int(y), int(m), int(d)
}

// FormatTime returns instant written in the pattern of class: instant is
// Unix seconds for TIME and Unix milliseconds for NISTTIME. It gives false
// for an instant outside the years 0000 to 9999, which the pattern cannot
// write.
func FormatTime(class TimeClass, instant int64) (string, bool) {
	seconds, fraction := instant, int64(0)
	if class == ClassNISTTIME {
		seconds, fraction = floorDiv(instant, 1000), instant-floorDiv(instant, 1000)*1000
	}
	if seconds < firstTimeSecond || seconds > lastTimeSecond {
		return "", false
	}
	days := floorDiv(seconds, 86400)
	rest := seconds - days*86400
	year, month, day := civilFromDays(days)
	text := fmt.Sprintf("%04d-%02d-%02dT%02d:%02d:%02d", year, month, day, rest/3600, rest%3600/60, rest%60)
	if class == ClassNISTTIME {
		return fmt.Sprintf("%s.%03dZ", text, fraction), true
	}
	return text + "Z", true
}
