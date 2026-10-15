// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: Apache-2.0

package keyring

import (
	"errors"
	"fmt"
	"net/http"
	"os"
	"sync/atomic"
	"testing"

	"github.com/truestamp/truestamp-cli/internal/httpclient"
)

// TestMain installs a transport on the shared HTTP client that refuses
// every request, so a test of this package that reached for the network
// through it would fail here rather than leave the machine. The package's
// tests read only bytes in memory, embedded spec data and files of this
// repository.
func TestMain(m *testing.M) {
	httpclient.SetTransport(refusingTransport{})
	os.Exit(m.Run())
}

// errRefused is what refusingTransport answers every request with.
var errRefused = errors.New("keyring tests refuse every network request")

// refusals counts the requests refusingTransport has refused.
var refusals atomic.Int64

type refusingTransport struct{}

func (refusingTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	refusals.Add(1)
	return nil, fmt.Errorf("%w: %s", errRefused, req.URL.Redacted())
}

// The control: a request through the shared client reaches the refusal and
// goes no further. The host is under .invalid, so even without the refusal
// it names nothing. Mutant killed: TestMain without its SetTransport call.
func TestMain_RefusesNetwork(t *testing.T) {
	before := refusals.Load()
	_, err := httpclient.GetJSON("https://keyring.invalid/.well-known/keyring.json")
	if !errors.Is(err, errRefused) {
		t.Fatalf("GetJSON error = %v, want the test transport's refusal", err)
	}
	if got := refusals.Load() - before; got != 1 {
		t.Fatalf("the test transport refused %d requests, want 1", got)
	}
}
