// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: MIT

package external

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/url"
	"strings"

	"github.com/truestamp/truestamp-cli/internal/httpclient"
)

// KeyringResponse is the shape of /.well-known/keyring.json.
type KeyringResponse struct {
	Version string       `json:"version"`
	Keys    []KeyringKey `json:"keys"`
}

// keyringDocument mirrors [KeyringResponse] with `keys` behind a pointer
// so an absent list is distinguishable from an empty one. Decoding
// straight into KeyringResponse cannot tell them apart: `{}`, `null`,
// `{"version":"1"}` and every other JSON object unmarshal without error
// and leave Keys nil, which reads identically to a keyring that
// published zero keys. Those are opposite facts — nothing was published
// versus Truestamp published nothing that vouches for this key — and
// only the second may fail a proof (E.22).
type keyringDocument struct {
	Version string        `json:"version"`
	Keys    *[]KeyringKey `json:"keys"`
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
// (net.DNSError, net.OpError, context.DeadlineExceeded, etc.) and so
// [Classify] can still see the httpclient status/transport typing
// underneath the friendly text.
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

// compactError strips the verbose `Get "<url>": ` wrapper Go's HTTP
// client adds, so the message a user sees is just the underlying cause.
//
// A *StatusError is returned verbatim: its "HTTP <code>" prefix is the
// most useful part of the message, and splitting on the last ": " (what
// this did before) silently discarded it, leaving the user with a bare
// response body and no status.
func compactError(err error) string {
	var statusErr *httpclient.StatusError
	if errors.As(err, &statusErr) {
		return statusErr.Error()
	}
	var urlErr *url.Error
	if errors.As(err, &urlErr) && urlErr.Err != nil {
		return urlErr.Err.Error()
	}
	return err.Error()
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
//   - The keyring URL is derived from --base-url (or TRUESTAMP_BASE_URL,
//     or base_url in config.toml); set that origin from a source you
//     trust — e.g. official Truestamp docs — not from a bundle authored
//     by the same party whose proof you are verifying.
//   - The CLI enforces TLS chain validation (InsecureSkipVerify is never
//     set) and refuses any redirect that downgrades https to http, so a
//     server cannot talk the fetch out of TLS mid-chain
//     (httpclient.ErrRedirectDowngrade). Cross-host redirects that stay
//     on https ARE followed — they remain authenticated by the CA chain,
//     and banning them would break the GitHub-asset fetch in
//     internal/selfupgrade, which shares this client.
//
// A future revision may add pinning of the keyring payload's hash or a
// cosign/Sigstore signature over the keyring document itself. Until
// then, treat the configured base URL as a root of trust that deserves
// the same care as a CA root.
func VerifyKeyring(signingKeys map[string]string, keyringURL string) error {
	resp, err := httpclient.GetJSON(keyringURL)
	if err != nil {
		return &keyringNetError{friendly: classifyNetworkError(err), inner: err}
	}

	var doc keyringDocument
	if err := json.Unmarshal(resp, &doc); err != nil {
		return &MalformedResponseError{Source: "keyring", Detail: "response is not a JSON keyring document", Err: err}
	}

	// E.17 gives this step exactly one job: report whether the published
	// keyring vouches for the key. A 200 that is not a keyring at all —
	// a captive portal, a CDN stub, an API gateway, a misconfigured
	// origin — answers that question not at all, and falling through to
	// the loop below would manufacture "not found in keyring" (a chain
	// disagreement that fails a sound proof) out of a document that
	// never had a keys list. E.22: an answer this client cannot read
	// establishes nothing, so it must skip. The reference verifier
	// gates the same way (`{:ok, %{"keys" => keys}} when is_list(keys)`).
	if doc.Keys == nil {
		return &MalformedResponseError{Source: "keyring", Detail: `response carries no "keys" list`}
	}

	keys := *doc.Keys
	keyringMap := make(map[string]string, len(keys))
	for _, k := range keys {
		keyringMap[k.KeyID] = k.PublicKey
	}

	for keyID, pubkeyB64 := range signingKeys {
		published, ok := keyringMap[keyID]
		if !ok {
			return &KeyBindingError{KeyID: keyID, Reason: "not found in keyring"}
		}
		if published != pubkeyB64 {
			return &KeyBindingError{KeyID: keyID, Reason: "public key mismatch with keyring"}
		}
	}

	return nil
}
