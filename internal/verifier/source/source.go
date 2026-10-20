// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: Apache-2.0

// Package source holds the CLI verifier's readers of a Stellar or Bitcoin
// source, the servers SPEC.md section 9 (What a verifier pins) names under
// "Sources": a Horizon server for Stellar, and an Esplora server or a
// Bitcoin Core node for Bitcoin. Each read is a call of SPEC.md section
// 10.4's informative table (Common sources), made through
// internal/httpclient's SingleRead: one attempt, no redirect followed, one
// deadline over the whole read and a bound on the body. The reads are
// built by their caller for each read, never at package init.
//
// They are the reads SPEC.md sections 10.2, step 0, and 10.4 give every
// verifier in an online run, the reference verifier of section 13.1 among
// them, which reads chains and sources online when it is given a reader
// (section 8.3). They follow the Truestamp application's source readers,
// Truestamp.SourceCache.Identity, StellarSource and BitcoinSource
// (lib/truestamp/source_cache/identity.ex, stellar_source.ex and
// bitcoin_source.ex in truestamp-v2), adapted:
//
//   - only the identity reads of SPEC.md section 10.2, step 0
//     (CONFIGURATION), are here so far: Horizon's GET / for its
//     network_passphrase, Esplora's GET /block-height/0 and Bitcoin
//     Core's getblockhash 0 for the hash of the block at height 0; the
//     reads of CHAIN and SOURCES join with those steps;
//   - an identity read is found or unreachable, never not found (SPEC.md
//     section 13.2 (The vectors)): every answer but a value in its form,
//     a 404 and a Bitcoin Core error included, is an [*Unreachable],
//     whose reason is a fixed text that quotes nothing the server sent or
//     the transport gave: no body, no part of a JSON-RPC answer and no
//     transport error's text, which can carry the server's bytes (a
//     malformed status line or header line) and names the URL. So nothing
//     a server echoes back, such as the credentials a Core read sent, can
//     reach a message, and nothing in this package prints:
//     internal/registry's rule 2 scan refuses fmt's Print and Fprint
//     functions and the print builtins here, and its import list holds
//     neither log, log/slog nor os;
//   - Bitcoin Core's RPC credentials are a [Credentials], which prints
//     masked whatever verb formats it and is sent only in the request's
//     Authorization header, never in the URL; the application reaches its
//     own node through its configuration instead.
package source

import (
	"bytes"
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/truestamp/truestamp-cli/internal/httpclient"
	"github.com/truestamp/truestamp-cli/internal/strictjson"
)

// Provider is the kind of server a source is.
type Provider string

// The providers SPEC.md section 9 names.
const (
	// Horizon is a Stellar Horizon server.
	Horizon Provider = "horizon"
	// Esplora is a Bitcoin Esplora server.
	Esplora Provider = "esplora"
	// BitcoinCore is a Bitcoin Core node, read over its JSON-RPC interface.
	BitcoinCore Provider = "bitcoin_core"
)

// Chain is the chain a provider serves: stellar or bitcoin, or "" for a
// provider this package does not know.
func (p Provider) Chain() string {
	switch p {
	case Horizon:
		return "stellar"
	case Esplora, BitcoinCore:
		return "bitcoin"
	}
	return ""
}

// IdentityOctets bounds the body of an identity answer. Horizon's root is
// about two kilobytes, and the other two answers a hash.
const IdentityOctets = 65536

// Credentials are a Bitcoin Core node's RPC credentials, a user and a
// password. Every way of printing them prints [Masked]: fmt with any verb,
// encoding/json and any encoding.TextMarshaler. The members are unexported,
// so only this package reads them, to send them.
type Credentials struct {
	user, password string
}

// Masked is what [Credentials] print as.
const Masked = "[masked]"

// NewCredentials returns the credentials of user and password.
func NewCredentials(user, password string) Credentials {
	return Credentials{user: user, password: password}
}

// Empty reports whether neither a user nor a password is set.
func (c Credentials) Empty() bool { return c.user == "" && c.password == "" }

// String returns [Masked].
func (c Credentials) String() string { return Masked }

// GoString returns [Masked].
func (c Credentials) GoString() string { return Masked }

// Format writes [Masked] for every verb, so neither %v, %+v, %#v, %s, %q
// nor %x shows the members.
func (c Credentials) Format(f fmt.State, _ rune) { _, _ = io.WriteString(f, Masked) }

// MarshalText returns [Masked].
func (c Credentials) MarshalText() ([]byte, error) { return []byte(Masked), nil }

// MarshalJSON returns [Masked] as a JSON string.
func (c Credentials) MarshalJSON() ([]byte, error) { return []byte(`"` + Masked + `"`), nil }

// header returns the Authorization header the credentials send, or an
// empty header for empty credentials, which a caller may add to.
func (c Credentials) header() http.Header {
	if c.Empty() {
		return http.Header{}
	}
	token := base64.StdEncoding.EncodeToString([]byte(c.user + ":" + c.password))
	return http.Header{"Authorization": {"Basic " + token}}
}

// Unreachable is an identity read that got no usable answer (SPEC.md
// section 10.4, "Answers"). Reason says why in a fixed text, which quotes
// nothing a server sent or the transport gave and names no URL. A read
// that failed before its answer arrived also wraps the transport's error
// as its cause, for [errors.Is] and [errors.As]; that error's text can
// carry bytes the server sent, such as a malformed status line, and names
// the URL, so it is never printed: Error returns Reason alone.
type Unreachable struct {
	Reason string
	cause  error
}

func (e *Unreachable) Error() string { return e.Reason }

// Unwrap returns the transport's error, or nil.
func (e *Unreachable) Unwrap() error { return e.cause }

// unreachable returns an [*Unreachable] with a reason.
func unreachable(format string, args ...any) error {
	return &Unreachable{Reason: fmt.Sprintf(format, args...)}
}

// The reasons of a read that failed before its answer arrived.
const (
	reasonDeadline = "no answer within the read's deadline"
	reasonNoAnswer = "no answer: the connection failed, or the answer is not HTTP"
)

// fromRead turns an error of a SingleRead into an [*Unreachable] with a
// fixed reason: an HTTP status that is not 200 OK (a 3xx, a 404, a 429
// and a 503 among them), named by its code alone; a body over its bound;
// the deadline passing; or any other failure before the whole answer
// arrived. The last two keep err as the cause, whose text is never
// printed.
func fromRead(err error) error {
	var status *httpclient.StatusError
	var truncated *httpclient.TruncatedError
	switch {
	case errors.As(err, &status):
		return unreachable("HTTP %d", status.StatusCode)
	case errors.As(err, &truncated):
		return unreachable("an answer over %d octets", truncated.Limit)
	case errors.Is(err, context.DeadlineExceeded):
		return &Unreachable{Reason: reasonDeadline, cause: err}
	}
	return &Unreachable{Reason: reasonNoAnswer, cause: err}
}

// Identity reads the identity value of the source of provider at base
// through reads: Horizon's network_passphrase, or the hash of the block at
// height 0 of an Esplora server or a Bitcoin Core node. creds are sent only
// to a Bitcoin Core node. Any answer that is not a value in its form is an
// [*Unreachable].
func Identity(ctx context.Context, reads *httpclient.SingleRead, provider Provider, base string, creds Credentials) (string, error) {
	switch provider {
	case Horizon:
		return StellarIdentity(ctx, reads, base)
	case Esplora:
		return EsploraIdentity(ctx, reads, base)
	case BitcoinCore:
		return CoreIdentity(ctx, reads, base, creds)
	}
	return "", fmt.Errorf("source: no identity read for the provider %q", provider)
}

// StellarIdentity reads a Horizon server's root, GET <base>/, and returns
// its network_passphrase, an ASCII128.
func StellarIdentity(ctx context.Context, reads *httpclient.SingleRead, base string) (string, error) {
	body, err := reads.Get(ctx, endpoint(base, "/"), IdentityOctets)
	if err != nil {
		return "", fromRead(err)
	}
	value, err := strictjson.Parse(body)
	root, isObject := value.(map[string]any)
	if err != nil || !isObject {
		return "", unreachable("Horizon's root is not a JSON object")
	}
	passphrase, isString := root["network_passphrase"].(string)
	if !isString || !ascii128(passphrase) {
		return "", unreachable("Horizon's root has no network_passphrase of 1 to 128 printable ASCII characters")
	}
	return passphrase, nil
}

// EsploraIdentity reads an Esplora server's GET <base>/block-height/0 and
// returns the hash of the block at height 0, a HEX64.
func EsploraIdentity(ctx context.Context, reads *httpclient.SingleRead, base string) (string, error) {
	body, err := reads.Get(ctx, endpoint(base, "/block-height/0"), IdentityOctets)
	if err != nil {
		return "", fromRead(err)
	}
	if hash := string(body); hex64(hash) {
		return hash, nil
	}
	return "", unreachable("Esplora's answer is not the 64 lowercase hex digits of a block hash")
}

// coreCallID is the id of every JSON-RPC call this package makes, which
// the answer must echo.
const coreCallID = "truestamp-verify"

// CoreIdentity calls a Bitcoin Core node's getblockhash 0 at url and
// returns the hash of the block at height 0, a HEX64. The call is a
// JSON-RPC 1.0 POST, with creds in its Authorization header; the answer
// must be {result, error, id} with error null, id this call's and result
// the hash. A Core error, such as -8 for a height past the tip, comes with
// an HTTP status other than 200 OK, and is unreachable like any other.
func CoreIdentity(ctx context.Context, reads *httpclient.SingleRead, url string, creds Credentials) (string, error) {
	call := []byte(`{"jsonrpc":"1.0","id":"` + coreCallID + `","method":"getblockhash","params":[0]}`)
	header := creds.header()
	header.Set("Content-Type", "application/json")
	body, err := reads.Post(ctx, url, call, header, IdentityOctets)
	if err != nil {
		return "", fromRead(err)
	}
	value, err := strictjson.Parse(bytes.TrimSpace(body))
	answer, isObject := value.(map[string]any)
	if err != nil || !isObject {
		return "", unreachable("Bitcoin Core's answer is not a JSON object")
	}
	if rpcError, present := answer["error"]; present && rpcError != nil {
		return "", unreachable("Bitcoin Core answered with an error")
	}
	if id, _ := answer["id"].(string); id != coreCallID {
		return "", unreachable("Bitcoin Core's answer is not to this call")
	}
	hash, isString := answer["result"].(string)
	if !isString || !hex64(hash) {
		return "", unreachable("Bitcoin Core's result is not the 64 lowercase hex digits of a block hash")
	}
	return hash, nil
}

// endpoint joins base, with any trailing slashes dropped, and path.
func endpoint(base, path string) string {
	return strings.TrimRight(base, "/") + path
}

// CheckURL refuses a source URL no read may be made to: one that does not
// parse, whose scheme is not http or https, that names no host, that
// carries user information, a query or a fragment. Its error never quotes
// the URL, which could hold credentials.
func CheckURL(raw string) error {
	u, err := url.Parse(raw)
	switch {
	case err != nil:
		return errors.New("the URL does not parse")
	case u.User != nil:
		return errors.New("the URL carries user information")
	case u.Scheme != "https" && u.Scheme != "http":
		return errors.New("the URL's scheme is not http or https")
	case u.Hostname() == "":
		return errors.New("the URL names no host")
	case u.RawQuery != "" || u.ForceQuery || u.Fragment != "":
		return errors.New("the URL carries a query or a fragment")
	}
	return nil
}

// ascii128 reports whether s is an ASCII128 (SPEC.md section 2.4): 1 to
// 128 printable ASCII characters other than the quote and the backslash.
func ascii128(s string) bool {
	if len(s) < 1 || len(s) > 128 {
		return false
	}
	for _, c := range []byte(s) {
		if c < ' ' || c > '~' || c == '"' || c == '\\' {
			return false
		}
	}
	return true
}

// hex64 reports whether s is a HEX64 (SPEC.md section 2.4): 64 lowercase
// hex digits.
func hex64(s string) bool {
	if len(s) != 64 {
		return false
	}
	for _, c := range []byte(s) {
		if !('0' <= c && c <= '9' || 'a' <= c && c <= 'f') {
			return false
		}
	}
	return true
}
