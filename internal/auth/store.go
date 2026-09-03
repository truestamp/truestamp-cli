// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: MIT

package auth

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"time"

	"github.com/zalando/go-keyring"
)

// keyringService is the OS-keychain service name under which sessions are
// stored. The account/key within the service is the base origin, so dev
// (localhost) and prod sessions never collide.
const keyringService = "truestamp-cli"

// ErrNoSession means no OAuth session is stored for the origin.
var ErrNoSession = errors.New("no oauth session stored")

// Session is the persisted OAuth state for one server origin. The endpoint
// URLs are captured at login so token refresh and revocation never need a
// fresh discovery round-trip during normal operation.
type Session struct {
	Issuer        string    `json:"issuer"`
	AuthURL       string    `json:"auth_url"`
	TokenURL      string    `json:"token_url"`
	RevocationURL string    `json:"revocation_url,omitempty"`
	AccessToken   string    `json:"access_token"`
	RefreshToken  string    `json:"refresh_token"`
	TokenType     string    `json:"token_type"`
	Scope         string    `json:"scope,omitempty"`
	Expiry        time.Time `json:"expiry"`
}

// Store persists [Session]s for a single server origin. It prefers the OS
// keychain (macOS Keychain, libsecret/SecretService, Windows credential
// manager) and transparently falls back to a 0600 JSON file under the
// user cache dir when no keychain is available (e.g. headless Linux).
// Reads consult both backends so a session written under one is still
// found if availability changes between runs.
type Store struct {
	origin string
}

// NewStore returns a Store keyed by the canonical origin of baseURL.
func NewStore(baseURL string) Store {
	return Store{origin: canonicalOrigin(baseURL)}
}

// Load returns the stored session for the origin, or [ErrNoSession].
func (s Store) Load() (Session, error) {
	if v, err := keyring.Get(keyringService, s.origin); err == nil {
		var sess Session
		if json.Unmarshal([]byte(v), &sess) == nil && sess.RefreshToken != "" {
			return sess, nil
		}
	}
	return s.fileLoad()
}

// Save persists the session, preferring the keychain and falling back to
// the 0600 file when the keychain write fails (unsupported/unavailable).
func (s Store) Save(sess Session) error {
	b, err := json.Marshal(sess) //#nosec G117 -- deliberate: serializing the OAuth session for the OS keychain / 0600-file token store is the store's purpose, not a leak
	if err != nil {
		return err
	}
	if err := keyring.Set(keyringService, s.origin, string(b)); err == nil {
		// Keychain is the single source of truth on success: drop any stale
		// 0600-file copy for this origin so a later load can't return an
		// out-of-date session from the fallback backend.
		_ = s.fileClear()
		return nil
	}
	return s.fileSave(sess)
}

// Clear removes the session from both backends. Missing entries are not an
// error (logout is idempotent).
func (s Store) Clear() error {
	_ = keyring.Delete(keyringService, s.origin)
	return s.fileClear()
}

// fileStorePath is the fallback file location.
func (s Store) fileStorePath() (string, error) {
	dir, err := os.UserCacheDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "truestamp", "oauth.json"), nil
}

func (s Store) fileLoad() (Session, error) {
	p, err := s.fileStorePath()
	if err != nil {
		return Session{}, ErrNoSession
	}
	b, err := os.ReadFile(p)
	if err != nil {
		return Session{}, ErrNoSession
	}
	var m map[string]Session
	if json.Unmarshal(b, &m) != nil {
		return Session{}, ErrNoSession
	}
	sess, ok := m[s.origin]
	if !ok || sess.RefreshToken == "" {
		return Session{}, ErrNoSession
	}
	return sess, nil
}

func (s Store) fileSave(sess Session) error {
	p, err := s.fileStorePath()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(p), 0700); err != nil {
		return err
	}
	m := map[string]Session{}
	if b, err := os.ReadFile(p); err == nil {
		_ = json.Unmarshal(b, &m)
	}
	m[s.origin] = sess
	return writeFile0600(p, m)
}

func (s Store) fileClear() error {
	p, err := s.fileStorePath()
	if err != nil {
		return nil
	}
	b, err := os.ReadFile(p)
	if err != nil {
		return nil
	}
	m := map[string]Session{}
	if json.Unmarshal(b, &m) != nil {
		return nil
	}
	if _, ok := m[s.origin]; !ok {
		return nil
	}
	delete(m, s.origin)
	if len(m) == 0 {
		return os.Remove(p)
	}
	return writeFile0600(p, m)
}

// writeFile0600 atomically writes m as indented JSON with 0600 perms.
func writeFile0600(path string, m map[string]Session) error {
	b, err := json.MarshalIndent(m, "", "  ") //#nosec G117 -- deliberate: this is the 0600-file token store; the file is created mode 0600 and the bytes are the session we intend to persist
	if err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, b, 0600); err != nil {
		return err
	}
	if err := os.Chmod(tmp, 0600); err != nil {
		_ = os.Remove(tmp)
		return err
	}
	if err := os.Rename(tmp, path); err != nil {
		// Don't leave a 0600 temp file containing live tokens behind.
		_ = os.Remove(tmp)
		return err
	}
	return nil
}

// Location reports where a session for this origin is persisted, for
// display after login. It probes the keychain first, then the file
// fallback, so the message reflects the backend actually used.
func (s Store) Location() string {
	if _, err := keyring.Get(keyringService, s.origin); err == nil {
		return "OS keychain"
	}
	if p, err := s.fileStorePath(); err == nil {
		if _, ferr := os.Stat(p); ferr == nil {
			return p + " (0600, no OS keychain available)"
		}
	}
	return "OS keychain (0600 file fallback)"
}
