// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: Apache-2.0

package auth

import (
	"context"
	"encoding/json/v2"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/zalando/go-keyring"
)

// TestStoreLoad_PrefersTheNewerSession pins that Load returns the newer of
// the keychain's and the file's sessions, the one whose access token
// expires later, whichever backend holds it. Save writes the file when the
// keychain write fails, so an older keychain entry can sit beside a newer
// file; returning the keychain's unconditionally presented an
// already-rotated refresh token, and the server revoked the whole chain.
// Mutant killed: Load reading the keychain first and stopping there.
func TestStoreLoad_PrefersTheNewerSession(t *testing.T) {
	useMockKeyring(t)
	store := NewStore("http://localhost:4000")
	now := time.Now()
	older := Session{RefreshToken: "rt-older", AccessToken: "at-older", Expiry: now.Add(time.Hour)}
	newer := Session{RefreshToken: "rt-newer", AccessToken: "at-newer", Expiry: now.Add(24 * time.Hour)}

	for _, c := range []struct {
		name               string
		inKeychain, inFile Session
	}{
		{"the file holds the newer", older, newer},
		{"the keychain holds the newer", newer, older},
	} {
		t.Run(c.name, func(t *testing.T) {
			b, err := json.Marshal(c.inKeychain)
			if err != nil {
				t.Fatal(err)
			}
			if err := keyring.Set(keyringService, store.origin, string(b)); err != nil {
				t.Fatal(err)
			}
			if err := store.fileSave(c.inFile); err != nil {
				t.Fatal(err)
			}
			got, err := store.Load()
			if err != nil || got.RefreshToken != "rt-newer" {
				t.Errorf("Load = %q (%v), want the newer session rt-newer", got.RefreshToken, err)
			}
		})
	}
}

// TestRefresh_ASessionNoBackendTookIsReported pins that a refresh whose
// rotated session neither the keychain nor the file takes still serves the
// request with the new token, and reports the failure through the
// installed notifier, since nothing recovers it later. Mutant killed: the
// save error discarded, so the next run's "session expired" was the first
// sign of it.
func TestRefresh_ASessionNoBackendTookIsReported(t *testing.T) {
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
	azr := Resolve(Credentials{}, store)

	// From here neither backend takes a write: the keychain refuses, and
	// the fallback file's directory cannot be made, its parent being a
	// regular file.
	keyring.MockInitWithError(fmt.Errorf("keychain locked"))
	t.Cleanup(keyring.MockInit)
	blocker := filepath.Join(t.TempDir(), "not-a-directory")
	if err := os.WriteFile(blocker, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("HOME", blocker)
	t.Setenv("XDG_CACHE_HOME", blocker)

	var reported error
	SetSaveFailureNotifier(func(err error) { reported = err })
	t.Cleanup(func() { SetSaveFailureNotifier(nil) })

	if err := azr.ForceRefresh(ctx); err != nil {
		t.Fatalf("ForceRefresh: %v; a lost save must not fail the request", err)
	}
	if tok, err := azr.BearerToken(ctx); err != nil || tok != "access-2" {
		t.Errorf("BearerToken after the refresh = %q, %v; want access-2", tok, err)
	}
	if reported == nil {
		t.Error("the lost save was not reported to the notifier")
	}
}
