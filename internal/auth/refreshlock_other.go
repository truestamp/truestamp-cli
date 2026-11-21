// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: Apache-2.0

//go:build !unix && !windows

package auth

import (
	"errors"
	"os"
)

// errNoFileLock is tryLockFile's answer on a platform with no file lock;
// lockRefresh then lets the refresh go on unlocked.
var errNoFileLock = errors.New("no file lock on this platform")

func tryLockFile(*os.File) (bool, error) { return false, errNoFileLock }

func unlockFile(*os.File) error { return nil }
