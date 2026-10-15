// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: Apache-2.0

package verifier

import (
	"net/http"
	"testing"
	"time"

	"github.com/truestamp/truestamp-cli/internal/httpclient"
)

// The test seams of this package's keyrings (SPEC.md section 8.3).
//
// The compiled pin, spec/keyring.production.json as built in, holds no key
// until Truestamp's production keys join the spec, so a test of a run the
// pin governs replaces it, for that test only, with a keyring holding
// pinKey, a key that exists only here: no environment uses it, and
// spec/keys.nonproduction.json does not list it, so the pin passes its load
// checks under prod.

// pinSigner is the private key of pinKey.
var pinSigner = signerOf("the compiled pin's test-only key")

// pinKey is the one key of the compiled pin a test installs with
// [usePinHolding].
var pinKey = publicOf(pinSigner)

// usePinHolding replaces the compiled pin with a keyring of keys until the
// test ends, and returns the keyring's bytes. A test that calls it must not
// run in parallel.
func usePinHolding(t *testing.T, keys ...[32]byte) []byte {
	t.Helper()
	var entries []string
	for i, key := range keys {
		entries = append(entries, keyringEntry(key, true, i))
	}
	data := keyringBytes(entries...)
	saved := compiledPin
	compiledPin = func() []byte { return data }
	t.Cleanup(func() { compiledPin = saved })
	return data
}

// byteReader is a keyring reader that answers each read with the bytes
// read gives.
func byteReader(read func(url string) ([]byte, error)) KeyringReader {
	return func(url string) (KeyringRead, error) {
		data, err := read(url)
		return KeyringRead{Bytes: data}, err
	}
}

// defaultReadsOf builds [DefaultKeyringReader] once with newSingleRead
// replaced, and returns the transport and the deadline its reads were
// built with. A test that calls it must not run in parallel.
func defaultReadsOf(t *testing.T) (http.RoundTripper, time.Duration) {
	t.Helper()
	var (
		transport http.RoundTripper
		deadline  time.Duration
		built     int
	)
	saved := newSingleRead
	newSingleRead = func(tr http.RoundTripper, d time.Duration) *httpclient.SingleRead {
		transport, deadline, built = tr, d, built+1
		return saved(tr, d)
	}
	defer func() { newSingleRead = saved }()
	if DefaultKeyringReader() == nil || built != 1 {
		t.Fatalf("DefaultKeyringReader built %d reads", built)
	}
	return transport, deadline
}
