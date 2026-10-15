// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: Apache-2.0

package vectorcases

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"sync/atomic"
	"testing"

	"github.com/truestamp/truestamp-cli/internal/httpclient"
	"github.com/truestamp/truestamp-cli/internal/verifier"
)

// TestMain refuses the network on both of the transports this package's
// tests can read through, so a test that reached for a real host would
// fail here rather than leave the machine:
//
//   - the shared HTTP client's, through httpclient.SetTransport, which
//     refuses every request;
//   - http.DefaultTransport, which internal/verifier's real readers,
//     DefaultKeyringReader and DefaultIdentityReader, read through, and
//     which these tests can reach since they import internal/verifier: it
//     is replaced by loopbackOnly,
//     which sends a request on to the transport saved at package init only
//     when its host is a loopback IP literal and refuses every other one
//     before any lookup.
//
// The package's tests read only bytes in memory and files of this
// repository, and every online case's keyring and identity reads are
// answered by the replay adapter.
func TestMain(m *testing.M) {
	httpclient.SetTransport(refusingTransport{})
	http.DefaultTransport = loopbackOnly{}
	os.Exit(m.Run())
}

// errRefused is what refusingTransport and loopbackOnly answer each
// request they refuse with.
var errRefused = errors.New("vectorcases tests refuse every network request")

// refusals counts the requests refusingTransport and loopbackOnly have
// refused.
var refusals atomic.Int64

type refusingTransport struct{}

func (refusingTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	refusals.Add(1)
	return nil, fmt.Errorf("%w: %s", errRefused, req.URL.Redacted())
}

// savedTransport is http.DefaultTransport as it was at package init,
// before TestMain replaced it.
var savedTransport = http.DefaultTransport

// loopbackOnly is the http.DefaultTransport of this package's tests: it
// sends a request whose host is a loopback IP literal on through
// savedTransport and refuses every other request in process.
type loopbackOnly struct{}

func (loopbackOnly) RoundTrip(req *http.Request) (*http.Response, error) {
	if ip := net.ParseIP(req.URL.Hostname()); ip != nil && ip.IsLoopback() {
		return savedTransport.RoundTrip(req)
	}
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

// The control of http.DefaultTransport: a request to a host under .invalid,
// sent straight to it with its context already cancelled, is refused in
// process. Without the refusal the real transport answers the cancelled
// context and dials nothing, so the test fails before the second read,
// internal/verifier's DefaultKeyringReader's of www.truestamp.com's
// keyring, which the refusal then answers. Mutant killed: TestMain without
// the DefaultTransport swap.
func TestMain_RefusesNetworkOnTheDefaultTransport(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://keyring.invalid/.well-known/keyring.json", nil)
	if err != nil {
		t.Fatal(err)
	}
	before := refusals.Load()
	resp, err := http.DefaultTransport.RoundTrip(req)
	if resp != nil {
		_ = resp.Body.Close()
	}
	if !errors.Is(err, errRefused) {
		t.Fatalf("http.DefaultTransport error = %v, want the test transport's refusal", err)
	}
	if got := refusals.Load() - before; got != 1 {
		t.Fatalf("http.DefaultTransport refused %d requests, want 1", got)
	}
	before = refusals.Load()
	if _, err := verifier.DefaultKeyringReader()(verifier.CurrentKeyringURL); !errors.Is(err, errRefused) {
		t.Fatalf("DefaultKeyringReader error = %v, want the test transport's refusal", err)
	}
	if got := refusals.Load() - before; got != 1 {
		t.Fatalf("DefaultKeyringReader's read was refused %d times, want 1", got)
	}
}

// The control of internal/verifier's identity reader: DefaultIdentityReader
// over the compiled-in default of each network that has one is refused in
// process, once each, with no usable answer. Mutant killed: TestMain
// without the DefaultTransport swap (the test stops before any read).
func TestMain_RefusesNetworkOnTheIdentityReader(t *testing.T) {
	if _, ok := http.DefaultTransport.(loopbackOnly); !ok {
		t.Fatalf("http.DefaultTransport is %T, not the loopback-only refusal", http.DefaultTransport)
	}
	for _, key := range [][2]string{{"stellar", "public"}, {"stellar", "testnet"}, {"bitcoin", "mainnet"}, {"bitcoin", "testnet"}} {
		s, ok := verifier.DefaultSource(key[0], key[1])
		if !ok {
			t.Fatalf("no default for %v", key)
		}
		before := refusals.Load()
		if value, err := verifier.DefaultIdentityReader()(s); !errors.Is(err, errRefused) || value != "" {
			t.Fatalf("%s: %q, %v; want the test transport's refusal", s.URL, value, err)
		}
		if got := refusals.Load() - before; got != 1 {
			t.Fatalf("%s: %d requests refused, want 1", s.URL, got)
		}
	}
}
