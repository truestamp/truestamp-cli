// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: Apache-2.0

package external

import (
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/truestamp/truestamp-cli/internal/strictjson"
)

// FuzzClassifyNetworkError: the helper that words a keyring fetch failure
// for display.
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

// FuzzFetchKeyringBody: a 200 body gives a keyring only when it is a
// keyring document, an object the strict reader reads (SPEC.md section
// 10.1) carrying a keys list; any other body is an error and no keyring.
// Mutant killed, by the seed that repeats keys: FetchKeyring decoding
// with encoding/json, which keeps the second, empty list.
func FuzzFetchKeyringBody(f *testing.F) {
	for _, s := range []string{
		`{}`, `null`, `{"version":"1"}`, `{"keys":null}`, `{"keys":[]}`,
		`{"keys":"x"}`, `[]`, `not json`,
		`{"keys":[{"key_id":"aa","public_key":"bb"}]}`,
		`{"keys":null,"keys":[]}`,
	} {
		f.Add(s)
	}
	f.Fuzz(func(t *testing.T, body string) {
		srv := newBodyServer(t)
		srv.serve([]byte(body))

		keyring, err := FetchKeyring(srv.srv.URL)
		if (keyring == nil) == (err == nil) {
			t.Fatalf("body %q: keyring %+v and error %v", body, keyring, err)
		}
		if err != nil {
			return
		}
		// The only bodies allowed to give a keyring are real keyrings:
		// an object the strict reader reads, with a keys list.
		value, parseErr := strictjson.Decode([]byte(body))
		object, isObject := value.(map[string]any)
		if _, isList := object["keys"].([]any); parseErr != nil || !isObject || !isList {
			t.Fatalf("body %q is not a keyring document but gave the keyring %+v", body, keyring)
		}
	})
}
