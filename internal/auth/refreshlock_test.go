// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: Apache-2.0

package auth

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"
)

// TestRefresh_TwoProcessesRefreshingAtOnceRotateOnce pins the shared
// refresh lock and the re-read under it. Two authorizers loaded from one
// saved session stand for two truestamp processes, and both force a
// refresh at once. The test server, like the real one, takes each refresh
// token once and refuses it after. Both must end with the same new access
// token, and the server must see one refresh grant, not two. Mutants
// killed: no lock (both refresh with the same token, and the second is
// refused); no re-read under the lock (the second waits, then refreshes
// with the token the first already rotated, and is refused).
func TestRefresh_TwoProcessesRefreshingAtOnceRotateOnce(t *testing.T) {
	useMockKeyring(t)
	as := newFakeAS(t)
	store := NewStore(as.origin())
	orig := loopbackPorts
	loopbackPorts = []int{0}
	defer func() { loopbackPorts = orig }()

	ctx := context.Background()
	if _, err := Login(ctx, as.origin(), store, LoginOptions{Open: browserOpener(t), Out: io.Discard}); err != nil {
		t.Fatalf("Login: %v", err)
	}
	first, second := Resolve(Credentials{}, store), Resolve(Credentials{}, store)

	start := make(chan struct{})
	errs := make([]error, 2)
	var wg sync.WaitGroup
	for i, azr := range []Authorizer{first, second} {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			errs[i] = azr.ForceRefresh(ctx)
		}()
	}
	close(start)
	wg.Wait()

	for i, err := range errs {
		if err != nil {
			t.Errorf("refresh %d: %v; a concurrent refresh must not end the session", i+1, err)
		}
	}
	tok1, err1 := first.BearerToken(ctx)
	tok2, err2 := second.BearerToken(ctx)
	if err1 != nil || err2 != nil || tok1 != "access-2" || tok2 != "access-2" {
		t.Errorf("tokens %q (%v) and %q (%v); want both access-2", tok1, err1, tok2, err2)
	}
	as.mu.Lock()
	issued := as.tokenCounter
	as.mu.Unlock()
	if issued != 2 {
		t.Errorf("the server issued %d token sets; want 2, the sign-in and one refresh", issued)
	}
}

// TestLockRefresh_WaitsForTheHolder pins that a second lock on the same
// origin waits until the first is released, as a second process's would:
// flock and LockFileEx locks belong to an open file, so two opens conflict
// even within one process. Mutant killed: tryLockFile succeeding while
// another open of the file holds the lock.
func TestLockRefresh_WaitsForTheHolder(t *testing.T) {
	useMockKeyring(t)
	store := NewStore("http://localhost:4000")
	ctx := context.Background()
	release := store.lockRefresh(ctx)

	acquired := make(chan struct{})
	go func() {
		releaseSecond := store.lockRefresh(ctx)
		close(acquired)
		releaseSecond()
	}()
	select {
	case <-acquired:
		t.Fatal("the second lock was taken while the first was held")
	case <-time.After(300 * time.Millisecond):
	}
	release()
	select {
	case <-acquired:
	case <-time.After(5 * time.Second):
		t.Fatal("the second lock was not taken after the first was released")
	}
}

// TestTryLockFile_ExcludesAnotherOpen pins the cross-process half of the
// lock: while one open of the lock file holds it, a second open, as
// another process's would be, is refused at once, and takes it after the
// first is released. Mutant killed: tryLockFile succeeding while another
// open of the file holds the lock.
func TestTryLockFile_ExcludesAnotherOpen(t *testing.T) {
	path := filepath.Join(t.TempDir(), "refresh.lock")
	open := func() *os.File {
		f, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE, 0o600)
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { _ = f.Close() })
		return f
	}
	holder, other := open(), open()
	if locked, err := tryLockFile(holder); !locked || err != nil {
		t.Fatalf("first lock: %v, %v", locked, err)
	}
	if locked, err := tryLockFile(other); locked || err != nil {
		t.Fatalf("second open took the held lock: %v, %v", locked, err)
	}
	if err := unlockFile(holder); err != nil {
		t.Fatal(err)
	}
	if locked, err := tryLockFile(other); !locked || err != nil {
		t.Fatalf("second open after release: %v, %v", locked, err)
	}
}

// TestLockRefresh_GoesOnUnlockedWithoutACacheDirectory pins that a lock
// that cannot be taken never fails or hangs a refresh: with a cache
// directory that cannot be made, lockRefresh returns at once with a
// release that does nothing. Mutant killed: lockRefresh waiting out
// refreshLockWait, or failing, when the lock file cannot be opened.
func TestLockRefresh_GoesOnUnlockedWithoutACacheDirectory(t *testing.T) {
	useMockKeyring(t)
	blocker := filepath.Join(t.TempDir(), "not-a-directory")
	if err := os.WriteFile(blocker, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("HOME", blocker)
	t.Setenv("XDG_CACHE_HOME", blocker)
	store := NewStore("http://localhost:4000")

	begun := time.Now()
	release := store.lockRefresh(context.Background())
	release()
	if waited := time.Since(begun); waited > time.Second {
		t.Errorf("lockRefresh took %v without a cache directory; want an immediate fallback", waited)
	}
}
