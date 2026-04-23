// Copyright (c) 2021-2026 Truestamp, Inc.
// SPDX-License-Identifier: MIT

package external

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"strings"

	"github.com/truestamp/truestamp-cli/internal/httpclient"
)

// KeyringResponse is the shape of /.well-known/keyring.json.
type KeyringResponse struct {
	Version string       `json:"version"`
	Keys    []KeyringKey `json:"keys"`
}

// KeyringKey is a single entry in the keyring.
type KeyringKey struct {
	KeyID     string `json:"key_id"`
	PublicKey string `json:"public_key"`
	Sequence  int    `json:"sequence"`
	Active    bool   `json:"active"`
}

// keyringNetError wraps a raw network error with a friendlier message.
// Error() returns the friendly text; Unwrap() returns the underlying
// error so callers can errors.Is/errors.As against the original type
// (net.DNSError, net.OpError, context.DeadlineExceeded, etc.).
type keyringNetError struct {
	friendly string
	inner    error
}

func (e *keyringNetError) Error() string { return e.friendly }
func (e *keyringNetError) Unwrap() error { return e.inner }

// classifyNetworkError returns a human-friendly message for common network errors.
func classifyNetworkError(err error) string {
	var dnsErr *net.DNSError
	if errors.As(err, &dnsErr) {
		return "could not resolve the server hostname; check that the keyring URL is correct"
	}

	var opErr *net.OpError
	if errors.As(err, &opErr) {
		if opErr.Op == "dial" && strings.Contains(opErr.Err.Error(), "connection refused") {
			return "could not connect to the keyring server; the server may be offline"
		}
	}

	if errors.Is(err, context.DeadlineExceeded) || isTimeoutError(err) {
		return "the request to the keyring server timed out"
	}

	var tlsErr *tls.CertificateVerificationError
	if errors.As(err, &tlsErr) {
		return "TLS/SSL error connecting to the keyring server"
	}
	// Also catch tls.RecordHeaderError and similar
	if strings.Contains(err.Error(), "tls:") || strings.Contains(err.Error(), "certificate") {
		return "TLS/SSL error connecting to the keyring server"
	}

	return fmt.Sprintf("could not reach the keyring server: %s", compactError(err))
}

// isTimeoutError checks for timeout errors via the Timeout() interface.
func isTimeoutError(err error) bool {
	type timeoutErr interface {
		Timeout() bool
	}
	var t timeoutErr
	if errors.As(err, &t) {
		return t.Timeout()
	}
	return false
}

// compactError strips verbose URL wrapper text from Go HTTP errors.
func compactError(err error) string {
	s := err.Error()
	// Go wraps net errors as: Get "url": <underlying>
	// Strip the method + URL prefix to keep the message short.
	if idx := strings.LastIndex(s, ": "); idx != -1 {
		inner := s[idx+2:]
		if len(inner) > 0 {
			return inner
		}
	}
	return s
}

// VerifyKeyring checks that all signing keys in the proof match the published keyring.
//
// Trust model: the keyring's authenticity is rooted entirely in the TLS
// chain presented by keyringURL. There is no in-band signature over the
// keyring payload itself, so every key in the returned document is only
// as trustworthy as the URL you configured. In particular:
//
//   - Use https:// with a host whose certificate chain you trust (an
//     attacker able to mint a valid cert for the host — via DNS/BGP
//     hijack, a rogue CA, or a compromised certificate — can substitute
//     signing keys and every downstream signature will validate against
//     their key).
//   - Configure --keyring-url (or the TRUESTAMP_KEYRING_URL env var, or
//     the keyring_url setting in config.toml) from a source you trust —
//     e.g. official Truestamp docs — not from a bundle authored by the
//     same party whose proof you are verifying.
//   - The CLI enforces TLS chain validation (InsecureSkipVerify is never
//     set) and does not follow cross-host redirects that strip TLS.
//
// A future revision may add pinning of the keyring payload's hash or a
// cosign/Sigstore signature over the keyring document itself. Until
// then, treat keyring-url as a root of trust that deserves the same
// care as a CA root.
func VerifyKeyring(signingKeys map[string]string, keyringURL string) error {
	resp, err := httpclient.GetJSON(keyringURL)
	if err != nil {
		return &keyringNetError{friendly: classifyNetworkError(err), inner: err}
	}

	var keyring KeyringResponse
	if err := json.Unmarshal(resp, &keyring); err != nil {
		return fmt.Errorf("parsing keyring: %w", err)
	}

	keyringMap := make(map[string]string, len(keyring.Keys))
	for _, k := range keyring.Keys {
		keyringMap[k.KeyID] = k.PublicKey
	}

	for keyID, pubkeyB64 := range signingKeys {
		published, ok := keyringMap[keyID]
		if !ok {
			return fmt.Errorf("key %s not found in keyring", keyID)
		}
		if published != pubkeyB64 {
			return fmt.Errorf("key %s public key mismatch with keyring", keyID)
		}
	}

	return nil
}
