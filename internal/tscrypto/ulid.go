// Copyright (c) 2021-2026 Truestamp, Inc.
// SPDX-License-Identifier: MIT

package tscrypto

import (
	"fmt"
	"time"

	"github.com/oklog/ulid/v2"
)

// ExtractULIDTimestamp extracts the millisecond timestamp from a ULID string.
func ExtractULIDTimestamp(id string) (time.Time, error) {
	parsed, err := ulid.ParseStrict(id)
	if err != nil {
		return time.Time{}, fmt.Errorf("parsing ULID: %w", err)
	}
	return time.UnixMilli(int64(parsed.Time())).UTC(), nil
}

// FormatItemTime extracts and formats the timestamp from a ULID item ID.
// Returns "unknown" if extraction fails.
func FormatItemTime(itemID string) string {
	t, err := ExtractULIDTimestamp(itemID)
	if err != nil {
		return "unknown"
	}
	return t.Format(time.RFC3339)
}