// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: MIT

// Package ids validates the two identifier shapes the read-only resource
// groups accept on the command line: a UUIDv7 and a 64-hex-character
// hash. One home, so beacons, blocks and entropy refuse the same inputs
// with the same words.
package ids

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/gofrs/uuid/v5"
)

var hash64 = regexp.MustCompile(`^[0-9a-f]{64}$`)

// ValidateHash64 returns nil iff s is exactly 64 lowercase hex characters.
// It is the client-side guard standing in for the server-side one a
// filter route does not have: an unguarded cast in a filter raises a 500.
func ValidateHash64(s string) error {
	if !hash64.MatchString(s) {
		return fmt.Errorf("hash must be 64 lowercase hex characters, got %q", s)
	}
	return nil
}

// ValidateUUIDv7 returns nil iff s parses as a UUID whose version is 7.
func ValidateUUIDv7(s string) error {
	u, err := uuid.FromString(s)
	if err != nil {
		return fmt.Errorf("invalid UUID: %w", err)
	}
	if u.Version() != 7 {
		return fmt.Errorf("id must be a UUIDv7 (version=7), got version=%d", u.Version())
	}
	return nil
}

// LooksLikeHash64 reports whether s has the SHAPE of a 64-hex hash rather
// than a UUID: no hyphens and 64 characters. It is a shape test on a value
// the user typed, used to pick a route, not validation; the filename-
// independence rule in CLAUDE.md is about a bundle's subject type and does
// not apply.
func LooksLikeHash64(s string) bool {
	return !strings.Contains(s, "-") && len(s) == 64
}
