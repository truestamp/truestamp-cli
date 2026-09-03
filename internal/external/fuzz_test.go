// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: MIT

package external

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/truestamp/truestamp-cli/internal/tscrypto"
)

// FuzzClassifyNetworkError: error-typing helper used when formatting
// Horizon / Blockstream / keyring fetch failures for display.
func FuzzClassifyNetworkError(f *testing.F) {
	for _, s := range []string{"", "dial tcp: connection refused", "timeout", "context deadline exceeded"} {
		f.Add(s)
	}
	f.Fuzz(func(t *testing.T, msg string) {
		_ = classifyNetworkError(stringError(msg))
	})
}

// FuzzCompactError: network error compactor.
func FuzzCompactError(f *testing.F) {
	for _, s := range []string{"", "short", "a very very very very very long error message"} {
		f.Add(s)
	}
	f.Fuzz(func(t *testing.T, s string) {
		_ = compactError(stringError(s))
	})
}

// FuzzClassify: an error shape this package does not recognise must
// always grade as "no answer obtained", never as a success and never as
// a mismatch, E.22 forbids an unrecognised failure from failing a
// proof.
func FuzzClassify(f *testing.F) {
	for _, s := range []string{"", "HTTP 404: nope", "memo mismatch: expected a, got b", "context deadline exceeded"} {
		f.Add(s)
	}
	f.Fuzz(func(t *testing.T, msg string) {
		if got := Classify(stringError(msg)); got != OutcomeUnavailable {
			t.Errorf("Classify(%q) = %s, want unavailable", msg, got)
		}
		if got := Classify(fmt.Errorf("wrapped: %w", stringError(msg))); got != OutcomeUnavailable {
			t.Errorf("Classify(wrapped %q) = %s, want unavailable", msg, got)
		}
	})
}

// stringError lets fuzz-provided strings satisfy the error interface.
type stringError string

func (e stringError) Error() string { return string(e) }

// bodyServer serves whatever body the caller last stored, so a fuzz
// target can drive one httptest server across many iterations.
type bodyServer struct {
	mu   sync.Mutex
	body []byte
	srv  *httptest.Server
}

func newBodyServer(t *testing.T) *bodyServer {
	t.Helper()
	b := &bodyServer{}
	b.srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		b.mu.Lock()
		body := b.body
		b.mu.Unlock()
		_, _ = w.Write(body)
	}))
	t.Cleanup(b.srv.Close)
	return b
}

func (b *bodyServer) serve(body []byte) {
	b.mu.Lock()
	b.body = body
	b.mu.Unlock()
}

// FuzzVerifyKeyringBody pins E.17/E.22's separation: a 200 body may fail
// a proof (KeyBindingError -> OutcomeMismatch) only when it really is a
// keyring document, a JSON object carrying a `keys` array. Anything
// else is an answer this client cannot read, and E.22 forbids it from
// failing a sound proof. Before the `keys` gate, every JSON object
// reached the per-key loop and returned "not found in keyring".
func FuzzVerifyKeyringBody(f *testing.F) {
	for _, s := range []string{
		`{}`, `null`, `{"version":"1"}`, `{"keys":null}`, `{"keys":[]}`,
		`{"keys":"x"}`, `[]`, `not json`,
		`{"keys":[{"key_id":"aa","public_key":"bb"}]}`,
	} {
		f.Add(s)
	}
	f.Fuzz(func(t *testing.T, body string) {
		srv := newBodyServer(t)
		srv.serve([]byte(body))

		err := VerifyKeyring(map[string]string{"aa": "bb"}, srv.srv.URL)
		if Classify(err) != OutcomeMismatch {
			return
		}
		// The only bodies allowed to reach a verdict are real keyrings.
		var doc struct {
			Keys *[]KeyringKey `json:"keys"`
		}
		if json.Unmarshal([]byte(body), &doc) != nil || doc.Keys == nil {
			t.Fatalf("body %q is not a keyring document but graded mismatch: %v", body, err)
		}
	})
}

// FuzzVerifyBitcoinBlockBody pins E.19(b)'s binding: a 200 body may
// produce a confirmable result only when it names the block that was
// asked for and carries a height. Any other body must yield an error, so
// the caller can never report "confirmed on <net>" for an answer about a
// different block or for silence about the height.
func FuzzVerifyBitcoinBlockBody(f *testing.F) {
	const asked = "2222222222222222222222222222222222222222222222222222222222222222"
	for _, s := range []string{
		`{}`, `null`, `{"height":1}`, `{"id":"` + asked + `"}`,
		`{"id":"` + asked + `","height":0}`,
		`{"id":"dede","height":1}`, `not json`,
	} {
		f.Add(s)
	}
	f.Fuzz(func(t *testing.T, body string) {
		srv := newBodyServer(t)
		srv.serve([]byte(body))

		orig := BlockstreamMainnetURL
		BlockstreamMainnetURL = srv.srv.URL
		defer func() { BlockstreamMainnetURL = orig }()

		result, skipped, err := VerifyBitcoinBlock(asked, "mainnet")
		if err != nil || skipped {
			return
		}
		if result == nil {
			t.Fatalf("body %q: no error, no skip, and no result", body)
		}
		var block struct {
			ID     string `json:"id"`
			Height *int   `json:"height"`
		}
		if json.Unmarshal([]byte(body), &block) != nil ||
			!tscrypto.HexEqual(block.ID, asked) || block.Height == nil {
			t.Fatalf("body %q yielded a confirmable result %+v without naming block %s at a height",
				body, result, asked)
		}
	})
}
