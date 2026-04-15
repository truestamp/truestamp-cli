// Copyright (c) 2021-2026 Truestamp, Inc.
// SPDX-License-Identifier: MIT

package tscrypto

import (
	"fmt"
	"time"

	"github.com/gofrs/uuid/v5"
)

// ExtractUUIDv7Timestamp extracts the millisecond timestamp from a UUIDv7 string.
func ExtractUUIDv7Timestamp(id string) (time.Time, error) {
	parsed, err := uuid.FromString(id)
	if err != nil {
		return time.Time{}, fmt.Errorf("parsing UUID: %w", err)
	}

	ts, err := uuid.TimestampFromV7(parsed)
	if err != nil {
		return time.Time{}, fmt.Errorf("extracting v7 timestamp: %w", err)
	}

	t, err := ts.Time()
	if err != nil {
		return time.Time{}, fmt.Errorf("converting timestamp: %w", err)
	}

	return t.UTC(), nil
}

// FormatBlockTime extracts and formats the timestamp from a UUIDv7 block ID.
// Returns "unknown" if extraction fails.
func FormatBlockTime(blockID string) string {
	t, err := ExtractUUIDv7Timestamp(blockID)
	if err != nil {
		return "unknown"
	}
	return t.Format(time.RFC3339)
}
