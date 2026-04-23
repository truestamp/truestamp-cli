// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: MIT

//go:build windows

package selfupgrade

import "errors"

// ErrReplaceUnsupported is returned from Replace on Windows — v1 of
// `truestamp upgrade` is print-only on Windows. The caller should detect
// Windows ahead of time and print `go install ...@latest` instructions
// instead.
var ErrReplaceUnsupported = errors.New("in-place upgrade not supported on Windows in this version")

// Replace always returns ErrReplaceUnsupported on Windows.
func Replace(destPath, newBinPath string) (string, error) {
	return "", ErrReplaceUnsupported
}
