// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: Apache-2.0

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
