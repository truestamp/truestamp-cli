// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: Apache-2.0

package external

import (
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/truestamp/truestamp-cli/internal/httpclient"
)

// statusServer returns an httptest server that answers every request
// with the given status and body.
func statusServer(t *testing.T, status int, body string) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(status)
		_, _ = fmt.Fprint(w, body)
	}))
	t.Cleanup(srv.Close)
	return srv
}

// A keyring the server publishes is read whole, its keys in order, and
// one that publishes an empty keys list is a keyring of no keys, not a
// malformed answer. Mutants killed: ParseKeyring refusing an empty keys
// list as it refuses an absent one (the empty keyring becomes an error);
// keyringKeyOf dropping active (the first key reads inactive).
func TestFetchKeyring_ReadsAPublishedKeyring(t *testing.T) {
	srv := statusServer(t, 200, `{"version":"1.0","keys":[`+
		`{"key_id":"4ceefa4a","public_key":"CTwMqDZnPd_QTLSq8aTeSD3a-j2DQxKcGfhhIYJQ65Y","sequence":0,"active":true},`+
		`{"key_id":"f2c39df9","public_key":"pk2","sequence":1,"active":false}]}`)
	keyring, err := FetchKeyring(srv.URL)
	if err != nil {
		t.Fatal(err)
	}
	want := []KeyringKey{
		{KeyID: "4ceefa4a", PublicKey: "CTwMqDZnPd_QTLSq8aTeSD3a-j2DQxKcGfhhIYJQ65Y", Sequence: 0, Active: true},
		{KeyID: "f2c39df9", PublicKey: "pk2", Sequence: 1, Active: false},
	}
	if keyring.Version != "1.0" || len(keyring.Keys) != len(want) || keyring.Keys[0] != want[0] || keyring.Keys[1] != want[1] {
		t.Fatalf("FetchKeyring = %+v, want version 1.0 and %+v", keyring, want)
	}

	empty, err := FetchKeyring(statusServer(t, 200, `{"version":"1.0","keys":[]}`).URL)
	if err != nil || empty == nil || len(empty.Keys) != 0 || empty.Keys == nil {
		t.Fatalf("an empty keys list: %+v, %v; want a keyring of no keys", empty, err)
	}
}

// A 2xx body that is not a keyring document, one with no keys list or
// a keys member that is not a list, is a MalformedResponseError and no
// keyring. Mutant killed: ParseKeyring without its check for an absent
// keys list (`{}` and the others become keyrings of no keys).
func TestFetchKeyring_NotAKeyringDocumentIsMalformed(t *testing.T) {
	for _, body := range []string{
		`{}`,
		`null`,
		`{"version":"1"}`,
		`{"ok":true}`,
		`{"keys":null}`,
		`{"version":"1.0","keys":"not-a-list"}`,
		`{"version":"1.0","keys":{"key_id":"f2c39df9"}}`,
	} {
		t.Run(body, func(t *testing.T) {
			keyring, err := FetchKeyring(statusServer(t, 200, body).URL)
			var malformed *MalformedResponseError
			if keyring != nil || !errors.As(err, &malformed) || malformed.Source != "keyring" {
				t.Fatalf("FetchKeyring = %+v, %v; want a malformed keyring response", keyring, err)
			}
		})
	}
}

// A non-2xx answer gives no keyring and an error that keeps the status
// beneath the friendly message, both in the message and for
// httpclient.Status. Mutants killed: keyringNetError without Unwrap
// (httpclient.Status gives 0); classifyNetworkError writing its fallback
// without the cause (the message loses "HTTP 503").
func TestFetchKeyring_AStatusIsKept(t *testing.T) {
	for _, tc := range []struct {
		status int
		body   string
		want   string
	}{
		{404, "no keyring here", "could not reach the keyring server: HTTP 404: no keyring here"},
		{503, `{"error":"maintenance"}`, `could not reach the keyring server: HTTP 503: {"error":"maintenance"}`},
	} {
		keyring, err := FetchKeyring(statusServer(t, tc.status, tc.body).URL)
		if keyring != nil || err == nil || err.Error() != tc.want || httpclient.Status(err) != tc.status {
			t.Errorf("HTTP %d: FetchKeyring = %+v, %v (status %d); want the error %q", tc.status, keyring, err, httpclient.Status(err), tc.want)
		}
	}
}

// A refused connection gives no keyring and the message for a server that
// is offline, over the transport error it wraps. Mutant killed:
// classifyNetworkError without its dial branch (the message becomes the
// fallback's).
func TestFetchKeyring_ARefusedConnection(t *testing.T) {
	keyring, err := FetchKeyring("http://127.0.0.1:1")
	if keyring != nil || err == nil || err.Error() != "could not connect to the keyring server; the server may be offline" || !httpclient.IsTransport(err) {
		t.Fatalf("FetchKeyring = %+v, %v; want the offline message over a transport error", keyring, err)
	}
}
