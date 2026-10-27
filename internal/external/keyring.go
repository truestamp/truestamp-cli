// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: Apache-2.0

// Package external reads Truestamp's keyring, /.well-known/keyring.json,
// for the keys commands (keys list, keys get and keys current). The
// verifier of `truestamp verify` reads and loads its keyring itself, in
// internal/verifier and internal/verifier/keyring. A keyring's bytes are
// read through internal/strictjson, the reader of SPEC.md section 10.1
// (Parsing) that every keyring a verifier loads goes through (section
// 10.2, step 0), so a body that breaks any of its rules is not a keyring:
//
//   - J1: a body over MAX_READ_OCTETS octets;
//   - J2: a body that begins with a byte order mark, or has content after
//     its value;
//   - J3: invalid UTF-8 anywhere in the body, or an escaped lone surrogate;
//   - J4: an object, at any depth, that repeats a member name;
//   - J5: a number that overflows binary64, or whose value is an integer
//     of magnitude above MAX_INTEGER and below 10^21, in any member, read
//     or not;
//   - J6: nesting deeper than MAX_DEPTH.
//
// Member names match exactly, case included: only keys and version name
// the keyring's members, and only key_id, public_key, sequence and active a
// key's, so a member named in another case (KEYS, KEY_ID, Active) is a
// member of no keyring and is ignored. A sequence is read by its value, so
// an integral spelling of it (1.0, 1e0) is that integer.
package external

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"net"
	"net/url"
	"strings"

	"github.com/truestamp/truestamp-cli/internal/httpclient"
	"github.com/truestamp/truestamp-cli/internal/strictjson"
)

// KeyringResponse is the shape of /.well-known/keyring.json.
type KeyringResponse struct {
	Version string       `json:"version"`
	Keys    []KeyringKey `json:"keys"`
}

// keyringDocument mirrors [KeyringResponse] with `keys` behind a pointer
// so an absent list is distinguishable from an empty one: `{}`, `null`,
// `{"version":"1"}` and every other such object leave Keys nil, which must
// not read as a keyring that published zero keys. Those are opposite
// facts: nothing was published, against a keyring that publishes no key.
type keyringDocument struct {
	Version string
	Keys    *[]KeyringKey
}

// errNotKeyringShape marks a body the strict reader reads whose value does
// not have a keyring's shape: not an object or null, or a member of the
// wrong type.
var errNotKeyringShape = errors.New("not the shape of a keyring")

// decodeKeyring reads body through the strict reader and takes the members
// of a keyring from the value it gives: a member that is absent or null is
// its zero value, a member of no keyring is ignored, an integral spelling
// of sequence (1.0, 1e0) is that integer, a null element of keys is a zero
// key, and only the exact name keys is the list. A member of the wrong
// type, an element of keys that is neither an object nor null, and a
// sequence that is not an integer of magnitude at most MAX_INTEGER are
// errors.
func decodeKeyring(body []byte) (keyringDocument, error) {
	value, err := strictjson.Decode(body)
	if err != nil {
		return keyringDocument{}, err
	}
	if value == nil {
		return keyringDocument{}, nil
	}
	object, ok := value.(map[string]any)
	if !ok {
		return keyringDocument{}, errNotKeyringShape
	}
	var doc keyringDocument
	if !optional(object["version"], &doc.Version) {
		return keyringDocument{}, errNotKeyringShape
	}
	if object["keys"] == nil {
		return doc, nil
	}
	elements, ok := object["keys"].([]any)
	if !ok {
		return keyringDocument{}, errNotKeyringShape
	}
	keys := make([]KeyringKey, 0, len(elements))
	for _, element := range elements {
		key, ok := keyringKeyOf(element)
		if !ok {
			return keyringDocument{}, errNotKeyringShape
		}
		keys = append(keys, key)
	}
	doc.Keys = &keys
	return doc, nil
}

// keyringKeyOf reads one element of keys: null, or an object whose
// members key_id, public_key, sequence and active, each when present and
// not null, are of their types.
func keyringKeyOf(element any) (KeyringKey, bool) {
	if element == nil {
		return KeyringKey{}, true
	}
	object, ok := element.(map[string]any)
	if !ok {
		return KeyringKey{}, false
	}
	var key KeyringKey
	if !optional(object["key_id"], &key.KeyID) || !optional(object["public_key"], &key.PublicKey) ||
		!optional(object["active"], &key.Active) {
		return KeyringKey{}, false
	}
	if sequence := object["sequence"]; sequence != nil {
		number, isNumber := sequence.(strictjson.Number)
		n, isInteger := number.Int64()
		if !isNumber || !isInteger {
			return KeyringKey{}, false
		}
		key.Sequence = int(n)
	}
	return key, true
}

// optional sets *into to value when value is of its type, leaves it when
// value is absent or null, and reports whether value was either.
func optional[T any](value any, into *T) bool {
	if value == nil {
		return true
	}
	typed, ok := value.(T)
	if ok {
		*into = typed
	}
	return ok
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
// (net.DNSError, net.OpError, context.DeadlineExceeded, etc.) and see the
// httpclient status and transport typing underneath the friendly text.
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

// FetchKeyring retrieves and decodes the keyring document at keyringURL.
// A transport failure or a non-2xx status is an error that keeps the
// httpclient typing beneath a friendly message, and a body that is not a
// keyring is a [MalformedResponseError].
//
// Trust model: the keyring's authenticity is rooted entirely in the TLS
// chain presented by keyringURL. There is no in-band signature over the
// keyring payload itself, so every key in the returned document is only
// as trustworthy as the URL you configured. In particular:
//
//   - Use https:// with a host whose certificate chain you trust (an
//     attacker able to mint a valid cert for the host, via DNS/BGP
//     hijack, a rogue CA, or a compromised certificate, can substitute
//     signing keys and every downstream signature will validate against
//     their key).
//   - The keyring URL is derived from --base-url (or TRUESTAMP_BASE_URL,
//     or base_url in config.toml); set that origin from a source you
//     trust, e.g. official Truestamp docs, not from a bundle authored
//     by the same party whose proof you are verifying.
//   - The CLI enforces TLS chain validation (InsecureSkipVerify is never
//     set) and refuses any redirect that downgrades https to http, so a
//     server cannot talk the fetch out of TLS mid-chain
//     (httpclient.ErrRedirectDowngrade). Cross-host redirects that stay
//     on https ARE followed, they remain authenticated by the CA chain,
//     and banning them would break the GitHub-asset fetch in
//     internal/selfupgrade, which shares this client.
//
// A future revision may add pinning of the keyring payload's hash or a
// cosign/Sigstore signature over the keyring document itself. Until
// then, treat the configured base URL as a root of trust that deserves
// the same care as a CA root.
func FetchKeyring(keyringURL string) (*KeyringResponse, error) {
	resp, err := httpclient.GetJSON(keyringURL)
	if err != nil {
		return nil, &keyringNetError{friendly: classifyNetworkError(err), inner: err}
	}
	return ParseKeyring(resp)
}

// ParseKeyring decodes a keyring document through the strict reader. A
// body without a `keys` list is not a keyring at all (a captive portal, a
// CDN stub) and is malformed, never a keyring that publishes no key.
func ParseKeyring(body []byte) (*KeyringResponse, error) {
	doc, err := decodeKeyring(body)
	if err != nil {
		return nil, &MalformedResponseError{Source: "keyring", Detail: "response is not a JSON keyring document", Err: err}
	}
	if doc.Keys == nil {
		return nil, &MalformedResponseError{Source: "keyring", Detail: `response carries no "keys" list`}
	}
	return &KeyringResponse{Version: doc.Version, Keys: *doc.Keys}, nil
}

// FindByID returns the entry with the given key id. The id is hex, so the
// comparison is case-insensitive. This is a lookup, not a binding: it
// answers "which entry has this id", never "is this key Truestamp's".
func (k *KeyringResponse) FindByID(keyID string) (KeyringKey, bool) {
	for _, entry := range k.Keys {
		if strings.EqualFold(entry.KeyID, keyID) {
			return entry, true
		}
	}
	return KeyringKey{}, false
}

// Active returns the entry the keyring marks as signing right now.
func (k *KeyringResponse) Active() (KeyringKey, bool) {
	for _, entry := range k.Keys {
		if entry.Active {
			return entry, true
		}
	}
	return KeyringKey{}, false
}
