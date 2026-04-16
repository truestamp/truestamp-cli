// Copyright (c) 2021-2026 Truestamp, Inc.
// SPDX-License-Identifier: MIT

package httpclient

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func init() {
	// The shared httpClient defaults to a 10-second timeout in Init();
	// tests run before Init() is called, so set something reasonable.
	Init(5 * time.Second)
}

func TestDownloadCtx_happyPath(t *testing.T) {
	payload := bytes.Repeat([]byte("truestamp"), 1024) // ~9 KB
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(payload)
	}))
	defer srv.Close()

	dest := filepath.Join(t.TempDir(), "artifact.bin")
	n, err := DownloadCtx(context.Background(), srv.URL, dest, 0)
	if err != nil {
		t.Fatalf("DownloadCtx: %v", err)
	}
	if n != int64(len(payload)) {
		t.Errorf("bytes written = %d, want %d", n, len(payload))
	}
	got, err := os.ReadFile(dest)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, payload) {
		t.Error("downloaded content does not match server payload")
	}
}

func TestDownloadCtx_rejectsOversize(t *testing.T) {
	// Server sends 100 bytes; caller caps at 50.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(bytes.Repeat([]byte("a"), 100))
	}))
	defer srv.Close()

	dest := filepath.Join(t.TempDir(), "big.bin")
	_, err := DownloadCtx(context.Background(), srv.URL, dest, 50)
	if err == nil {
		t.Fatal("DownloadCtx(cap=50, size=100): expected error, got nil")
	}
	if !strings.Contains(err.Error(), "cap") {
		t.Errorf("DownloadCtx cap error message = %q, want something mentioning cap", err.Error())
	}
	// Destination file must be cleaned up on failure.
	if _, err := os.Stat(dest); !os.IsNotExist(err) {
		t.Errorf("oversize download left %s behind (err=%v)", dest, err)
	}
}

func TestDownloadCtx_httpError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "not found", http.StatusNotFound)
	}))
	defer srv.Close()

	dest := filepath.Join(t.TempDir(), "missing.bin")
	_, err := DownloadCtx(context.Background(), srv.URL, dest, 0)
	if err == nil {
		t.Fatal("DownloadCtx on 404: expected error, got nil")
	}
	if !strings.Contains(err.Error(), "404") {
		t.Errorf("DownloadCtx 404 error = %q, want 'HTTP 404'", err.Error())
	}
	if _, err := os.Stat(dest); !os.IsNotExist(err) {
		t.Errorf("failed download created file: %s", dest)
	}
}

func TestDownloadCtx_ctxCancel(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Slow server: wait long enough that ctx.Done() fires first.
		time.Sleep(500 * time.Millisecond)
		_, _ = w.Write([]byte("too late"))
	}))
	defer srv.Close()

	dest := filepath.Join(t.TempDir(), "cancel.bin")
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	_, err := DownloadCtx(ctx, srv.URL, dest, 0)
	if err == nil {
		t.Fatal("DownloadCtx with cancelled ctx: expected error, got nil")
	}
}

func TestDownloadBytesCtx_happyPath(t *testing.T) {
	body := []byte("hello checksums.txt\n")
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(body)
	}))
	defer srv.Close()

	got, err := DownloadBytesCtx(context.Background(), srv.URL, 0)
	if err != nil {
		t.Fatalf("DownloadBytesCtx: %v", err)
	}
	if !bytes.Equal(got, body) {
		t.Errorf("body mismatch: got %q", string(got))
	}
}

func TestDownloadBytesCtx_rejectsOversize(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write(bytes.Repeat([]byte("x"), 200))
	}))
	defer srv.Close()

	_, err := DownloadBytesCtx(context.Background(), srv.URL, 50)
	if err == nil {
		t.Fatal("DownloadBytesCtx(cap=50, size=200): expected error, got nil")
	}
	if !strings.Contains(err.Error(), "cap") {
		t.Errorf("error message = %q, want 'cap'", err.Error())
	}
}

func TestDownloadBytesCtx_httpError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "gone", http.StatusGone)
	}))
	defer srv.Close()

	_, err := DownloadBytesCtx(context.Background(), srv.URL, 0)
	if err == nil {
		t.Fatal("DownloadBytesCtx on 410: expected error, got nil")
	}
	if !strings.Contains(err.Error(), "410") {
		t.Errorf("error = %q, want '410'", err.Error())
	}
}
