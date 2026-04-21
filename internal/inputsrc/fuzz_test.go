// Copyright (c) 2021-2026 Truestamp, Inc.
// SPDX-License-Identifier: MIT

package inputsrc

import "testing"

// FuzzIsHTTPURL: the URL classifier inside the positional-arg path.
// Attacker-controlled strings (positional argv, downloaded content
// peeked at, etc.) pass through here. No panic allowed.
func FuzzIsHTTPURL(f *testing.F) {
	for _, s := range []string{
		"", "http://x", "https://y/z", "file:///tmp", "/path",
		"http://", "http:///nohost", "javascript:evil()",
	} {
		f.Add(s)
	}
	f.Fuzz(func(t *testing.T, s string) {
		_ = isHTTPURL(s)
	})
}

// FuzzValidateURL: the huh form's validator predicate. Pure function
// so crashes here are a Go stdlib regression, not ours — still cheap
// to guard against.
func FuzzValidateURL(f *testing.F) {
	for _, s := range []string{
		"", "https://x", "http://y", "ftp://z", "whatever",
	} {
		f.Add(s)
	}
	f.Fuzz(func(t *testing.T, s string) {
		_ = validateURL(s)
	})
}
