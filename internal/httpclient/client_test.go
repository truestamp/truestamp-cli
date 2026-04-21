// Copyright (c) 2021-2026 Truestamp, Inc.
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
