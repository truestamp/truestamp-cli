// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: Apache-2.0

package cmd

import (
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
