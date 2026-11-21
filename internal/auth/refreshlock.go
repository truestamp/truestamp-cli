// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: Apache-2.0

package auth

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// The server rotates refresh tokens with reuse detection and no grace
// period: each refresh token works once, and a second use revokes the
// whole chain. Two truestamp processes that loaded one session and
// refresh at the same moment (the console and a command in another
// terminal, or a script running commands in parallel just after the
// access token expires) would present the same token twice and sign both
// out. So a refresh takes a lock shared by every truestamp process for the
// origin, a file beside the session file, and reads the session again
// once it holds it (oauthAuthorizer.token).

const (
	// refreshLockWait bounds how long a refresh waits for another
	// process's. A refresh is one HTTP round trip, so this is generous.
	refreshLockWait = 30 * time.Second
	// refreshLockPoll is how often a waiting refresh tries the lock again.
	refreshLockPoll = 50 * time.Millisecond
)

// lockRefresh takes the origin's refresh lock and returns the function
// that releases it. It waits at most refreshLockWait, and no longer than
// ctx allows. When the lock cannot be taken (no cache directory, a
// read-only one, a platform with no file lock, or the wait running out),
// it returns a release that does nothing and the refresh goes on unlocked,
// as it did before the lock existed: the lock guards against a race, and
// is never a reason to fail or hang a command. The OS drops the lock if
// the process dies holding it, so no stale lock outlives a crash.
func (s Store) lockRefresh(ctx context.Context) func() {
	noop := func() {}
	path, err := s.refreshLockPath()
	if err != nil {
		return noop
	}
	deadline := time.Now().Add(refreshLockWait)

	// Within this process the file lock is backed by a semaphore for the
	// same path: two authorizers of one process are then ordered in Go's
	// memory model as well, which a file lock alone does not give them
	// (and which the race detector cannot see through).
	sem := processRefreshLock(path)
	select {
	case sem <- struct{}{}:
	case <-ctx.Done():
		return noop
	case <-time.After(time.Until(deadline)):
		return noop
	}
	releaseSem := func() { <-sem }

	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return releaseSem
	}
	f, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE, 0o600) //#nosec G304 -- a fixed name under the user's cache directory
	if err != nil {
		return releaseSem
	}
	for {
		locked, err := tryLockFile(f)
		if locked {
			return func() {
				_ = unlockFile(f)
				_ = f.Close()
				releaseSem()
			}
		}
		if err != nil || !time.Now().Before(deadline) {
			_ = f.Close()
			return releaseSem
		}
		select {
		case <-ctx.Done():
			_ = f.Close()
			return releaseSem
		case <-time.After(refreshLockPoll):
		}
	}
}

// processRefreshLocks holds one semaphore, a channel of capacity one, per
// lock file path, for lockRefresh's in-process half.
var processRefreshLocks sync.Map

func processRefreshLock(path string) chan struct{} {
	sem, _ := processRefreshLocks.LoadOrStore(path, make(chan struct{}, 1))
	return sem.(chan struct{})
}

// refreshLockPath is the origin's lock file, beside the session file:
// <user cache dir>/truestamp/refresh-<hex SHA-256 of the origin>.lock.
func (s Store) refreshLockPath() (string, error) {
	dir, err := os.UserCacheDir()
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256([]byte(s.origin))
	return filepath.Join(dir, "truestamp", "refresh-"+hex.EncodeToString(sum[:])+".lock"), nil
}
