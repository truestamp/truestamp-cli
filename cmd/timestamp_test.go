// Copyright (c) 2021-2026 Truestamp, Inc.
// SPDX-License-Identifier: MIT

package cmd

import "testing"

func TestNormalizeTimestamp(t *testing.T) {
	cases := []struct {
		name    string
		input   string
		want    string
		wantErr bool
	}{
		{"date only promoted to UTC midnight", "2025-01-15", "2025-01-15T00:00:00Z", false},
		{"RFC3339 Z preserved", "2025-01-15T14:30:00Z", "2025-01-15T14:30:00Z", false},
		{"RFC3339 offset normalized to UTC", "2025-01-15T09:30:00-05:00", "2025-01-15T14:30:00Z", false},
		{"RFC3339 with fractional seconds", "2025-01-15T14:30:00.500Z", "2025-01-15T14:30:00Z", false},
		{"trims whitespace", "  2025-01-15  ", "2025-01-15T00:00:00Z", false},
		{"empty string", "", "", true},
		{"bogus", "not a date", "", true},
		{"year only", "2025", "", true},
		{"missing time separator", "2025-01-15 14:30:00", "", true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := normalizeTimestamp(tc.input)
			if (err != nil) != tc.wantErr {
				t.Fatalf("err = %v, wantErr = %v", err, tc.wantErr)
			}
			if got != tc.want {
				t.Errorf("got %q, want %q", got, tc.want)
			}
		})
	}
}
