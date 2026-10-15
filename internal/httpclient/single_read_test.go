// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: Apache-2.0

package httpclient

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

// Every answer here is an httptest server's, on loopback.

// countingServer is an httptest server that answers with handler and
// counts the requests it is sent.
func countingServer(t *testing.T, handler http.HandlerFunc) (*httptest.Server, *atomic.Int64) {
	t.Helper()
	var requests atomic.Int64
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		handler(w, r)
	}))
	t.Cleanup(srv.Close)
	return srv, &requests
}

// getWithin runs read.Get and fails the test if it has not returned within
// wait, so a read the deadline does not end fails rather than hangs.
func getWithin(t *testing.T, read *SingleRead, url string, limit int, wait time.Duration) ([]byte, time.Duration, error) {
	t.Helper()
	type result struct {
		body []byte
		err  error
	}
	done := make(chan result, 1)
	start := time.Now()
	go func() {
		body, err := read.Get(context.Background(), url, limit)
		done <- result{body, err}
	}()
	select {
	case r := <-done:
		return r.body, time.Since(start), r.err
	case <-time.After(wait):
		t.Fatalf("the read of %s had not ended after %v", url, wait)
		return nil, 0, nil
	}
}

// A 200 OK body of at most limit octets is the answer, one request made;
// one octet more is a *TruncatedError. Mutants killed: the body read to
// limit octets, not limit + 1 (the longer body is returned cut); the size
// check made with >= (the body of exactly limit octets is refused).
func TestSingleRead_ABodyWithinItsLimit(t *testing.T) {
	const limit = 16
	body := strings.Repeat("k", limit)
	srv, requests := countingServer(t, func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(body + strings.TrimPrefix(r.URL.Path, "/")))
	})
	read := NewSingleRead(nil, 5*time.Second)
	got, err := read.Get(context.Background(), srv.URL+"/", limit)
	if err != nil || string(got) != body {
		t.Fatalf("a body of %d octets: %q, %v", limit, got, err)
	}
	got, err = read.Get(context.Background(), srv.URL+"/x", limit)
	if !IsTruncated(err) || got != nil {
		t.Fatalf("a body of %d octets: %q, %v; want a *TruncatedError", limit+1, got, err)
	}
	if requests.Load() != 2 {
		t.Fatalf("%d requests, want 2", requests.Load())
	}
}

// No redirect is followed, to another host or to the same one: each 3xx is
// a *StatusError carrying its code, and the server it points at is sent
// nothing. Mutants killed: NewSingleRead's client without noRedirect (the
// redirect is followed, and the target answers); noRedirect giving
// checkRedirect's answer (an https-to-https or http-to-http redirect is
// followed).
func TestSingleRead_FollowsNoRedirect(t *testing.T) {
	target, reached := countingServer(t, func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"keys": [], "version": "1.0"}`))
	})
	// localhost names the target by another host than the 127.0.0.1 the
	// redirecting server is reached at.
	other := strings.Replace(target.URL, "127.0.0.1", "localhost", 1)
	for _, code := range []int{http.StatusMovedPermanently, http.StatusFound, http.StatusSeeOther, http.StatusTemporaryRedirect, http.StatusPermanentRedirect} {
		for _, location := range []string{other + "/keyring.json", target.URL + "/keyring.json"} {
			srv, _ := countingServer(t, func(w http.ResponseWriter, r *http.Request) {
				http.Redirect(w, r, location, code)
			})
			got, err := NewSingleRead(nil, 5*time.Second).Get(context.Background(), srv.URL, 1<<10)
			if Status(err) != code || got != nil {
				t.Errorf("%d to %s: %q, %v; want a *StatusError %d", code, location, got, err, code)
			}
		}
	}
	if n := reached.Load(); n != 0 {
		t.Fatalf("the redirect's target was sent %d requests", n)
	}
}

// Any status but 200 OK is a *StatusError: an HTTP error such as 503, and
// a 2xx that is not 200, whose body is no answer to a GET. Mutant killed:
// the status check accepting every 2xx (203 and 206 are answers).
func TestSingleRead_AnyStatusBut200IsAnHTTPError(t *testing.T) {
	for _, code := range []int{http.StatusServiceUnavailable, http.StatusNotFound, http.StatusNonAuthoritativeInfo, http.StatusPartialContent} {
		srv, _ := countingServer(t, func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(code)
			_, _ = w.Write([]byte(`{"keys": [], "version": "1.0"}`))
		})
		if got, err := NewSingleRead(nil, 5*time.Second).Get(context.Background(), srv.URL, 1<<10); Status(err) != code || got != nil {
			t.Errorf("HTTP %d: %q, %v", code, got, err)
		}
	}
}

// A server that closes the connection without an answer is a
// *TransportError. Mutant killed: the client's error returned as it is,
// not as a *TransportError.
func TestSingleRead_ATransportFailure(t *testing.T) {
	srv, requests := countingServer(t, func(w http.ResponseWriter, _ *http.Request) {
		conn, _, err := http.NewResponseController(w).Hijack()
		if err == nil {
			_ = conn.Close()
		}
	})
	got, err := NewSingleRead(nil, 5*time.Second).Get(context.Background(), srv.URL, 1<<10)
	if !IsTransport(err) || got != nil || requests.Load() != 1 {
		t.Fatalf("%q, %v after %d requests; want a *TransportError", got, err, requests.Load())
	}
}

// One deadline bounds the whole read: a server that sends no headers, and
// one that sends its headers and then trickles its body, are each cut when
// the deadline passes, a *TransportError wrapping
// context.DeadlineExceeded, long before the server would finish. Mutants
// killed: Get without its deadline (each read waits for the server, and
// the test fails at its own wait); the deadline's context cancelled once
// the headers arrive (the trickled body's read fails as cancelled, not as
// the deadline).
func TestSingleRead_CutAtTheDeadline(t *testing.T) {
	// Each server stops when the reader gives up, or after 10 s.
	silent, _ := countingServer(t, func(_ http.ResponseWriter, r *http.Request) {
		select {
		case <-r.Context().Done():
		case <-time.After(10 * time.Second):
		}
	})
	trickle, _ := countingServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		stop := time.After(10 * time.Second)
		for {
			if _, err := w.Write([]byte(" ")); err != nil {
				return
			}
			_ = http.NewResponseController(w).Flush()
			select {
			case <-r.Context().Done():
				return
			case <-stop:
				return
			case <-time.After(20 * time.Millisecond):
			}
		}
	})
	const deadline = 150 * time.Millisecond
	read := NewSingleRead(nil, deadline)
	for name, url := range map[string]string{"no headers": silent.URL, "a trickled body": trickle.URL} {
		got, took, err := getWithin(t, read, url, 1<<20, 5*time.Second)
		if !IsTransport(err) || !errors.Is(err, context.DeadlineExceeded) || got != nil {
			t.Errorf("%s: %q, %v; want the deadline", name, got, err)
		}
		if took < deadline || took > deadline+2*time.Second {
			t.Errorf("%s: ended after %v, with a deadline of %v", name, took, deadline)
		}
	}
}

// A single read is apart from the shared client: the transport
// SetTransport installs there does not carry it, and it sends no
// Authorization header. Mutant killed: NewSingleRead building on the
// shared client's transport (the refusing transport refuses the read).
func TestSingleRead_ApartFromTheSharedClient(t *testing.T) {
	SetTransport(refuseAll{})
	t.Cleanup(func() { SetTransport(nil) })
	srv, _ := countingServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "" {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		_, _ = w.Write([]byte("ok"))
	})
	if got, err := NewSingleRead(nil, 5*time.Second).Get(context.Background(), srv.URL, 1<<10); err != nil || string(got) != "ok" {
		t.Fatalf("%q, %v", got, err)
	}
	if _, err := GetJSON(srv.URL); err == nil || !strings.Contains(err.Error(), "refused") {
		t.Fatalf("the shared client: %v, want its transport's refusal", err)
	}
}

type refuseAll struct{}

func (refuseAll) RoundTrip(*http.Request) (*http.Response, error) {
	return nil, errors.New("refused by the shared client's test transport")
}
