// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: Apache-2.0

package bitcoin

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
)

// TestMain refuses the network on both transports a test of this package
// could reach it through, so a test that reached for a real host would
// fail here rather than leave the machine:
//
//   - the shared HTTP client's, through httpclient.SetTransport, which
//     refuses every request;
//   - http.DefaultTransport, replaced by loopbackOnly, which sends a
//     request on to the transport saved at package init only when its host
//     is a loopback IP literal, and refuses every other one before any
//     lookup.
//
// The package reads only bytes in memory, and its tests also read the
// Bitcoin corpus and the positive proofs of this repository's synced
// vectors.
func TestMain(m *testing.M) {
	httpclient.SetTransport(refusingTransport{})
	http.DefaultTransport = loopbackOnly{}
	os.Exit(m.Run())
}

// errRefused is what refusingTransport and loopbackOnly answer each
// request they refuse with.
var errRefused = errors.New("bitcoin tests refuse every network request")

// refusals counts the requests refusingTransport and loopbackOnly have
// refused, and sharedRefusals those refusingTransport has.
var refusals, sharedRefusals atomic.Int64

type refusingTransport struct{}

func (refusingTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	refusals.Add(1)
	sharedRefusals.Add(1)
	return nil, fmt.Errorf("%w: %s", errRefused, req.URL.Redacted())
}

// savedTransport is http.DefaultTransport as it was at package init,
// before TestMain replaced it.
var savedTransport = http.DefaultTransport

// loopbackOnly is the http.DefaultTransport of this package's tests.
type loopbackOnly struct{}

func (loopbackOnly) RoundTrip(req *http.Request) (*http.Response, error) {
	if ip := net.ParseIP(req.URL.Hostname()); ip != nil && ip.IsLoopback() {
		return savedTransport.RoundTrip(req)
	}
	refusals.Add(1)
	return nil, fmt.Errorf("%w: %s", errRefused, req.URL.Redacted())
}

// The control: a request through the shared client reaches the shared
// client's refusal, refusingTransport's, and goes no further. The host is
// under .invalid, so even without the refusal it names nothing. Mutant
// killed: TestMain without its SetTransport call (the request reaches
// loopbackOnly instead).
func TestMain_RefusesNetwork(t *testing.T) {
	before, shared := refusals.Load(), sharedRefusals.Load()
	_, err := httpclient.GetJSON("https://esplora.invalid/api/blocks/tip/height")
	if !errors.Is(err, errRefused) {
		t.Fatalf("GetJSON error = %v, want the test transport's refusal", err)
	}
	if got, gotShared := refusals.Load()-before, sharedRefusals.Load()-shared; got != 1 || gotShared != 1 {
		t.Fatalf("the test transports refused %d requests, the shared client's %d; want 1 and 1", got, gotShared)
	}
}

// The control of http.DefaultTransport: a request to a host under .invalid,
// sent straight to it with its context already cancelled, is refused in
// process; without the refusal the real transport answers the cancelled
// context and dials nothing, with an error that is not the refusal. A
// request to a loopback IP literal is passed on, and the cancelled context
// stops it there. Mutants killed: TestMain without the DefaultTransport
// swap (the first request is not refused); loopbackOnly refusing loopback
// too (the second is).
func TestMain_RefusesNetworkOnTheDefaultTransport(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	for _, c := range []struct {
		url     string
		refused bool
	}{{"https://esplora.invalid/api/blocks/tip/height", true}, {"http://127.0.0.1:9/", false}} {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.url, nil)
		if err != nil {
			t.Fatal(err)
		}
		before := refusals.Load()
		resp, err := http.DefaultTransport.RoundTrip(req)
		if resp != nil {
			_ = resp.Body.Close()
		}
		if errors.Is(err, errRefused) != c.refused || err == nil {
			t.Fatalf("%s: http.DefaultTransport error = %v, want refused %v", c.url, err, c.refused)
		}
		want := int64(0)
		if c.refused {
			want = 1
		}
		if got := refusals.Load() - before; got != want {
			t.Fatalf("%s: http.DefaultTransport refused %d requests, want %d", c.url, got, want)
		}
	}
}
