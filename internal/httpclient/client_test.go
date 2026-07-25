// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: MIT

package httpclient

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestTruncate(t *testing.T) {
	if Truncate("short", 10) != "short" {
		t.Error("short string should not be truncated")
	}
	result := Truncate("this is a long string", 10)
	if result != "this is a ..." {
		t.Errorf("truncate: got %q", result)
	}
}

func TestInit(t *testing.T) {
	Init(5 * time.Second)
	// Repeated initialization must not panic; subsequent clients can
	// service requests normally.
	req, _ := http.NewRequest("GET", "http://127.0.0.1:1/ignored", nil)
	_, _ = Do(req) // expected to error (no listener) — we just want to exercise Do
}

func TestGetJSON_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer srv.Close()
	body, err := GetJSON(srv.URL)
	if err != nil {
		t.Fatal(err)
	}
	if string(body) != `{"ok":true}` {
		t.Errorf("body: got %q", body)
	}
}

func TestGetJSONCtx_HTTP4xx(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte("not found"))
	}))
	defer srv.Close()
	_, err := GetJSONCtx(context.Background(), srv.URL)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "404") {
		t.Errorf("error should include status: %v", err)
	}
}

func TestGetJSONCtx_HTMLErrorPage(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadGateway)
		_, _ = w.Write([]byte("<html>error</html>"))
	}))
	defer srv.Close()
	_, err := GetJSONCtx(context.Background(), srv.URL)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "HTML") {
		t.Errorf("error should call out HTML body: %v", err)
	}
}

func TestGetJSONCtx_MalformedURL(t *testing.T) {
	_, err := GetJSONCtx(context.Background(), "://bad")
	if err == nil {
		t.Fatal("expected error for malformed URL")
	}
}

func TestGetJSONCtx_UnreachableHost(t *testing.T) {
	_, err := GetJSONCtx(context.Background(), "http://127.0.0.1:1/nope")
	if err == nil {
		t.Fatal("expected error for unreachable host")
	}
}

func TestGetJSONCtx_CancelledContext(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		time.Sleep(200 * time.Millisecond)
	}))
	defer srv.Close()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := GetJSONCtx(ctx, srv.URL)
	if err == nil {
		t.Fatal("expected context-cancel error")
	}
}

// TestSetUserAgent_StampedOnOutboundRequests verifies that after
// SetUserAgent is called, every request through GetJSONCtx/Do carries
// the configured User-Agent unless the caller already set one.
// Not t.Parallel() because it mutates the package-global userAgent.
func TestSetUserAgent_StampedOnOutboundRequests(t *testing.T) {
	var seen string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seen = r.Header.Get("User-Agent")
		_, _ = w.Write([]byte(`{}`))
	}))
	defer srv.Close()

	// Save and restore the package-global to avoid leaking into other tests.
	orig := userAgent
	t.Cleanup(func() { userAgent = orig })

	SetUserAgent("1.2.3")
	if _, err := GetJSONCtx(context.Background(), srv.URL); err != nil {
		t.Fatalf("GetJSONCtx: %v", err)
	}
	if !strings.Contains(seen, "truestamp-cli/1.2.3") {
		t.Errorf("User-Agent: got %q, want it to contain truestamp-cli/1.2.3", seen)
	}

	// Caller-set header must be preserved.
	req, _ := http.NewRequest("GET", srv.URL, nil)
	req.Header.Set("User-Agent", "caller-ua/9.9")
	if _, err := Do(req); err != nil {
		t.Fatalf("Do: %v", err)
	}
	if seen != "caller-ua/9.9" {
		t.Errorf("User-Agent (caller-set): got %q, want caller-ua/9.9", seen)
	}

	// Empty version disables the stamp.
	SetUserAgent("")
	req2, _ := http.NewRequest("GET", srv.URL, nil)
	if _, err := Do(req2); err != nil {
		t.Fatalf("Do: %v", err)
	}
	if strings.Contains(seen, "truestamp-cli") {
		t.Errorf("User-Agent with empty stamp: got %q, want no truestamp-cli prefix", seen)
	}
}

// --- Typed error coverage ---
//
// A verifier must report an unreachable source as `skip` and a
// definitive 404 on its own terms (whitepaper E.17/E.18/E.21/E.22), so
// the status code and the transport/response distinction have to reach
// the call site instead of collapsing into one opaque error.

func TestGetJSONCtx_StatusErrorCarriesCode(t *testing.T) {
	for _, code := range []int{404, 429, 500, 503} {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(code)
			_, _ = w.Write([]byte("nope"))
		}))
		_, err := GetJSONCtx(context.Background(), srv.URL)
		srv.Close()

		if got := Status(err); got != code {
			t.Errorf("Status() = %d, want %d (err=%v)", got, code, err)
		}
		if IsTransport(err) {
			t.Errorf("HTTP %d must not be classed as a transport failure", code)
		}
	}
}

func TestGetJSONCtx_StatusErrorSurvivesWrapping(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	_, err := GetJSONCtx(context.Background(), srv.URL)
	wrapped := fmt.Errorf("fetching thing: %w", err)
	if got := Status(wrapped); got != http.StatusNotFound {
		t.Errorf("Status() through a %%w chain = %d, want 404", got)
	}
}

func TestGetJSONCtx_TransportErrorTyped(t *testing.T) {
	_, err := GetJSONCtx(context.Background(), "http://127.0.0.1:1/nope")
	if !IsTransport(err) {
		t.Errorf("connection refused should be a transport failure: %v", err)
	}
	if got := Status(err); got != 0 {
		t.Errorf("Status() on a transport failure = %d, want 0", got)
	}
	// The underlying net error stays reachable for callers that want to
	// classify it further.
	var opErr *net.OpError
	if !errors.As(err, &opErr) {
		t.Errorf("underlying net.OpError should remain unwrappable: %v", err)
	}
}

func TestGetJSONCtx_CancelledContextIsTransport(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		time.Sleep(200 * time.Millisecond)
	}))
	defer srv.Close()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := GetJSONCtx(ctx, srv.URL)
	if !IsTransport(err) {
		t.Errorf("cancellation should be a transport failure: %v", err)
	}
	if !errors.Is(err, context.Canceled) {
		t.Errorf("context.Canceled should remain unwrappable: %v", err)
	}
}

// An oversize body used to be silently cut at the cap and surface as a
// JSON syntax error, which reads as "the server sent junk" when the
// real cause is our own limit.
func TestGetJSONCtx_OversizeBodyIsTruncatedError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write(bytes.Repeat([]byte("a"), MaxResponseSize+64))
	}))
	defer srv.Close()

	_, err := GetJSONCtx(context.Background(), srv.URL)
	if !IsTruncated(err) {
		t.Fatalf("oversize body should be a *TruncatedError: %v", err)
	}
	if IsTransport(err) || Status(err) != 0 {
		t.Errorf("truncation is neither transport nor status: %v", err)
	}
}

// A body exactly at the cap is complete and must still be returned.
func TestGetJSONCtx_BodyAtLimitIsReturned(t *testing.T) {
	body := bytes.Repeat([]byte("a"), MaxResponseSize)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write(body)
	}))
	defer srv.Close()

	got, err := GetJSONCtx(context.Background(), srv.URL)
	if err != nil {
		t.Fatalf("body at the limit should be accepted: %v", err)
	}
	if len(got) != MaxResponseSize {
		t.Errorf("body length: got %d, want %d", len(got), MaxResponseSize)
	}
}

func TestStatusAndIsTransport_OnUnrelatedErrors(t *testing.T) {
	err := errors.New("something else")
	if Status(err) != 0 || IsTransport(err) || IsTruncated(err) {
		t.Error("an unrelated error must not be classed as status, transport, or truncation")
	}
	if Status(nil) != 0 || IsTransport(nil) || IsTruncated(nil) {
		t.Error("nil must classify as nothing")
	}
}

// The typed errors keep the message text they replaced, so callers that
// format them (and the tests above that match on "404" / "HTML") are
// unaffected.
func TestTypedErrorMessages(t *testing.T) {
	tests := []struct {
		err  error
		want string
	}{
		{&StatusError{StatusCode: 404, Body: "not found"}, "HTTP 404: not found"},
		{&StatusError{StatusCode: 502, HTML: true}, "HTTP 502 (server returned HTML error page)"},
		{&TransportError{Err: errors.New("dial tcp: connection refused")}, "dial tcp: connection refused"},
		{&TruncatedError{Limit: 1024}, "response body exceeds the 1024 byte limit"},
	}
	for _, tc := range tests {
		if got := tc.err.Error(); got != tc.want {
			t.Errorf("Error() = %q, want %q", got, tc.want)
		}
	}
}
