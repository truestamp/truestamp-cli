// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"strconv"
	"strings"
	"testing"
)

// TestCheckTimestamp walks the claimed-timestamp grammar the server
// accepts: every accepted form passes, and each refusal names why.
// Mutants killed: the value trimmed before the check (" 2025-01-15" is
// accepted), the fraction allowed 10 digits, the real-date check dropped
// (2025-02-30 is accepted), and a date-only form required (2025 and
// 2025-01 are refused).
func TestCheckTimestamp(t *testing.T) {
	accepted := []string{
		"2025",
		"2025-01",
		"2025-01-15",
		"2024-02-29",
		"2025-01-15T14:30",
		"2025-01-15T14:30:00",
		"2025-01-15T14:30:00.5",
		"2025-01-15T14:30:00.123456789Z",
		"2025-01-15T09:30:00-05:00",
		"2025-01-15T09:30+0530",
	}
	for _, in := range accepted {
		if err := checkTimestamp(in); err != nil {
			t.Errorf("checkTimestamp(%q) = %v, want accepted", in, err)
		}
	}

	refused := []struct{ in, want string }{
		{"", "is not a date or time in an accepted form"},
		{" 2025-01-15", "is not a date or time in an accepted form"},
		{"2025-01-15 ", "is not a date or time in an accepted form"},
		{"2025-01-15 14:30:00", "is not a date or time in an accepted form"},
		{"2025-01-15T14:30:00.1234567890Z", "is not a date or time in an accepted form"},
		{"2025-13", "is not a date or time in an accepted form"},
		{"2025-01-15T24:00", "is not a date or time in an accepted form"},
		{"2025-01-15T14:30:00z", "is not a date or time in an accepted form"},
		{"2025-02-30", "names 2025-02-30, a date that does not exist"},
		{"2023-02-29T00:00Z", "names 2023-02-29, a date that does not exist"},
	}
	for _, tc := range refused {
		err := checkTimestamp(tc.in)
		if err == nil || !strings.Contains(err.Error(), tc.want) {
			t.Errorf("checkTimestamp(%q) = %v, want an error containing %q", tc.in, err, tc.want)
		}
	}
}

// TestCheckTimestamp_AsTheServerChecksIt holds checkTimestamp to the
// service's claimed-timestamp rule, the grammar and the real-date check
// the server applies to claims.timestamp, on the values the service's own
// tests of that rule use and on boundary values. Each row is the
// exact outcome: nil, or the whole message. A time follows only a whole
// date, so 2025T14:30 and 2025-06T14:30 are refused, and 2023-02-29 names
// no day. Mutants killed: the offset's colon required (+0530 refused); a
// time allowed after a year or a year and month (2025T14:30 accepted); the
// real-date check dropped (2023-02-29 accepted); Feb 29 refused in every
// century year (2000-02-29 refused, though 2000 is a leap year).
func TestCheckTimestamp_AsTheServerChecksIt(t *testing.T) {
	form := func(in string) string {
		return "--timestamp " + strconv.Quote(in) + " is not a date or time in an accepted form: YYYY, YYYY-MM, " +
			"YYYY-MM-DD, or YYYY-MM-DDThh:mm with optional :ss, an optional fraction of 1 to 9 " +
			"digits, and an optional Z, +hh:mm or -hh:mm offset"
	}
	for _, tc := range []struct{ in, want string }{
		{"2025", ""},
		{"2999", ""},
		{"2025-06", ""},
		{"2027-01", ""},
		{"2025-06-15", ""},
		{"2024-06-15", ""},
		{"2024-02-29", ""},
		{"2000-02-29", ""},
		{"2025-06-15T12:00:00Z", ""},
		{"2025-06-15T12:00:00+05:30", ""},
		{"2025-06-15T12:00:00+0530", ""},
		{"2025-06-15T12:00:00.123Z", ""},
		{"2025-06-15T12:30Z", ""},
		{"2025-06-15T12:30", ""},
		{"2026-10-28T01:02:03.123456Z", ""},
		{"2999-12-31T23:59:59Z", ""},
		{"2025-10-06T14:30:00.123456789+05:30", ""},
		{"2025-13", form("2025-13")},
		{"2025-01-32", form("2025-01-32")},
		{"2025-01-00", form("2025-01-00")},
		{"not-a-date", form("not-a-date")},
		{"1700000000", form("1700000000")},
		{"", form("")},
		{"01/15/2020", form("01/15/2020")},
		{"2024-06-15\n", form("2024-06-15\n")},
		{"2024-06-15T12:00:00Z\n", form("2024-06-15T12:00:00Z\n")},
		{"2025T14:30", form("2025T14:30")},
		{"2025-06T14:30", form("2025-06T14:30")},
		{"2025-06-15T12:00:00+24:00", form("2025-06-15T12:00:00+24:00")},
		{"2025-06-15T12:00:00+05:3", form("2025-06-15T12:00:00+05:3")},
		{"2025-06-15T12:00:60Z", form("2025-06-15T12:00:60Z")},
		{"2025-06-15T12:00:00.Z", form("2025-06-15T12:00:00.Z")},
		{"2025-06-15T12:00.5Z", form("2025-06-15T12:00.5Z")},
		{"2025-06-15T12:00:00UTC", form("2025-06-15T12:00:00UTC")},
		{"2023-02-29", "--timestamp names 2023-02-29, a date that does not exist"},
		{"2025-04-31", "--timestamp names 2025-04-31, a date that does not exist"},
		{"1900-02-29T00:00Z", "--timestamp names 1900-02-29, a date that does not exist"},
	} {
		got := ""
		if err := checkTimestamp(tc.in); err != nil {
			got = err.Error()
		}
		if got != tc.want {
			t.Errorf("checkTimestamp(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}
