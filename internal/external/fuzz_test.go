// Copyright (c) 2021-2026 Truestamp, Inc.
// SPDX-License-Identifier: MIT

package external

import "testing"

// FuzzClassifyNetworkError: error-typing helper used when formatting
// Horizon / Blockstream / keyring fetch failures for display.
func FuzzClassifyNetworkError(f *testing.F) {
	for _, s := range []string{"", "dial tcp: connection refused", "timeout", "context deadline exceeded"} {
		f.Add(s)
	}
	f.Fuzz(func(t *testing.T, msg string) {
		_ = classifyNetworkError(stringError(msg))
	})
}

// FuzzCompactError: network error compactor.
func FuzzCompactError(f *testing.F) {
	for _, s := range []string{"", "short", "a very very very very very long error message"} {
		f.Add(s)
	}
	f.Fuzz(func(t *testing.T, s string) {
		_ = compactError(stringError(s))
	})
}

// stringError lets fuzz-provided strings satisfy the error interface.
type stringError string

func (e stringError) Error() string { return string(e) }
