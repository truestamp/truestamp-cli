// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: Apache-2.0

//go:build windows

package auth

import (
	"errors"
	"os"

	"golang.org/x/sys/windows"
)

// tryLockFile takes an exclusive LockFileEx lock on the first byte of f
// without waiting. It reports false with no error when another handle
// holds the lock.
func tryLockFile(f *os.File) (bool, error) {
	err := windows.LockFileEx(windows.Handle(f.Fd()),
		windows.LOCKFILE_EXCLUSIVE_LOCK|windows.LOCKFILE_FAIL_IMMEDIATELY, 0, 1, 0, new(windows.Overlapped))
	switch {
	case err == nil:
		return true, nil
	case errors.Is(err, windows.ERROR_LOCK_VIOLATION):
		return false, nil
	default:
		return false, err
	}
}

// unlockFile releases tryLockFile's lock; closing f releases it too.
func unlockFile(f *os.File) error {
	return windows.UnlockFileEx(windows.Handle(f.Fd()), 0, 1, 0, new(windows.Overlapped))
}
