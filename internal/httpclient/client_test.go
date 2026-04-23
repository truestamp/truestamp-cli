// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: MIT

package httpclient

import (
	"context"
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
