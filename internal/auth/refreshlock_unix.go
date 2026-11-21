// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: Apache-2.0

//go:build unix

package auth

import (
	"errors"
	"os"
	"syscall"
)

// tryLockFile takes an exclusive flock(2) on f without waiting. It reports
// false with no error when another open file description holds the lock,
// which is so for another process and for another open of the file in this
// one.
func tryLockFile(f *os.File) (bool, error) {
	err := syscall.Flock(int(f.Fd()), syscall.LOCK_EX|syscall.LOCK_NB)
	switch {
	case err == nil:
		return true, nil
	case errors.Is(err, syscall.EWOULDBLOCK):
		return false, nil
	default:
		return false, err
	}
}

// unlockFile releases tryLockFile's lock; closing f releases it too.
func unlockFile(f *os.File) error {
	return syscall.Flock(int(f.Fd()), syscall.LOCK_UN)
}
