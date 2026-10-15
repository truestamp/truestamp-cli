// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: Apache-2.0

package source

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"sync/atomic"
	"testing"
	"time"

	"github.com/truestamp/truestamp-cli/internal/httpclient"
)

// TestMain refuses the network on both of the transports this package's
// code can read through, so a test that reached for a real host would fail
// here rather than leave the machine:
//
//   - the shared HTTP client's, through httpclient.SetTransport, which
//     refuses every request;
//   - http.DefaultTransport, which every reader here reads through when
//     its SingleRead is built with no transport of its own: it is replaced
//     by loopbackOnly, which sends a request on to the transport saved at
//     package init only when its host is a loopback IP literal, such as an
//     httptest server's, and refuses every other one before any lookup.
//
// Every answer the package's tests read is an httptest server's, on
// loopback.
func TestMain(m *testing.M) {
	httpclient.SetTransport(refusingTransport{})
	http.DefaultTransport = loopbackOnly{}
	os.Exit(m.Run())
}

// errRefused is what refusingTransport and loopbackOnly answer each
// request they refuse with.
var errRefused = errors.New("source tests refuse every network request")

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
	_, err := httpclient.GetJSON("https://horizon.invalid/")
	if !errors.Is(err, errRefused) {
		t.Fatalf("GetJSON error = %v, want the test transport's refusal", err)
	}
	if got := refusals.Load() - before; got != 1 {
		t.Fatalf("the test transport refused %d requests, want 1", got)
	}
}

// The control of every reader on http.DefaultTransport: each of the three
// identity reads, through a SingleRead built with no transport of its own,
// of a host under .invalid, is refused in process, once, and is
// unreachable, wrapping the refusal. Mutant killed: TestMain without the
// DefaultTransport swap (the test stops before any read).
func TestMain_RefusesNetworkOnEveryReader(t *testing.T) {
	if _, ok := http.DefaultTransport.(loopbackOnly); !ok {
		t.Fatalf("http.DefaultTransport is %T, not the loopback-only refusal", http.DefaultTransport)
	}
	for _, provider := range []Provider{Horizon, Esplora, BitcoinCore} {
		before := refusals.Load()
		reads := httpclient.NewSingleRead(nil, 5*time.Second)
		value, err := Identity(context.Background(), reads, provider, "https://"+string(provider)+".invalid", NewCredentials("u", "p"))
		var unreachable *Unreachable
		if !errors.Is(err, errRefused) || !errors.As(err, &unreachable) || value != "" {
			t.Fatalf("%s: %q, %v; want the test transport's refusal, unreachable", provider, value, err)
		}
		if got := refusals.Load() - before; got != 1 {
			t.Fatalf("%s: %d requests refused, want 1", provider, got)
		}
	}
}
