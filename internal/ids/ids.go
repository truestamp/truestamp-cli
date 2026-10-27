// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: Apache-2.0

// Package ids validates the two identifier shapes the read-only resource
// groups accept on the command line: a UUIDv7 and a 64-hex-character
// hash. One home, so beacons, blocks and entropy refuse the same inputs
// with the same words, and one UUIDv7 rule, [IsUUIDv7Text], which
// internal/proof's DetectIDType reads every other id by.
package ids

import (
	"fmt"
	"regexp"
	"strings"
)

var (
	hash64     = regexp.MustCompile(`^[0-9a-f]{64}$`)
	uuidv7Text = regexp.MustCompile(`\A[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-7[0-9a-fA-F]{3}-[89abAB][0-9a-fA-F]{3}-[0-9a-fA-F]{12}\z`)
)

// ValidateHash64 returns nil iff s is exactly 64 lowercase hex characters.
// It is the client-side guard standing in for the server-side one a
// filter route does not have: an unguarded cast in a filter raises a 500.
func ValidateHash64(s string) error {
	if !hash64.MatchString(s) {
		return fmt.Errorf("hash must be 64 lowercase hex characters, got %q", s)
	}
	return nil
}

// IsUUIDv7Text reports whether s is the 36-character text of a UUIDv7
// (RFC 9562 section 5.7), in either case: its version nibble is 7 and its
// variant is RFC 9562's. s is read exactly as given, so a value with a
// space or line feed around it, and a UUID's 32-hex, braced and urn:
// forms, are not.
func IsUUIDv7Text(s string) bool {
	return uuidv7Text.MatchString(s)
}

// ValidateUUIDv7 returns nil iff s is the text of a UUIDv7 by
// [IsUUIDv7Text].
func ValidateUUIDv7(s string) error {
	if !IsUUIDv7Text(s) {
		return fmt.Errorf("invalid UUID %q: an id is the 36-character text of a UUIDv7 (e.g. 019d6a32-13e6-72b0-97e5-3779231ea97b)", s)
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
