// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: Apache-2.0

package items

import (
	"errors"
	"fmt"
	"strings"
)

// Visibilities is the closed set the server accepts for an item's
// visibility. It is declared once so `items create` and `items update`
// validate and describe the same three values.
var Visibilities = []string{"private", "team", "public"}

// ErrEmptyVisibility refuses an explicit empty --visibility: the flag given
// with no value would otherwise send "" (update) or drop the member
// (create), where the user named a visibility and gave none.
var ErrEmptyVisibility = errors.New("--visibility needs a value: private, team or public")

// ValidateVisibility rejects a value outside Visibilities. The empty
// string is accepted: it means "not supplied".
func ValidateVisibility(v string) error {
	if v == "" {
		return nil
	}
	for _, ok := range Visibilities {
		if v == ok {
			return nil
		}
	}
	return fmt.Errorf("--visibility must be private, team, or public, got %q", v)
}

// NormalizeTags trims whitespace and drops empty entries, so "a, b,,"
// from a flag becomes ["a" "b"] whichever command received it.
func NormalizeTags(in []string) []string {
	var out []string
	for _, t := range in {
		if t = strings.TrimSpace(t); t != "" {
			out = append(out, t)
		}
	}
	return out
}
