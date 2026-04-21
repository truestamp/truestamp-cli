// Copyright (c) 2021-2026 Truestamp, Inc.
// SPDX-License-Identifier: MIT

package inputsrc

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestResolve_PositionalFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "payload.txt")
	content := []byte("hello world")
	if err := os.WriteFile(path, content, 0644); err != nil {
		t.Fatal(err)
	}

	data, src, err := Resolve(context.Background(), Options{PositionalArg: path})
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if string(data) != string(content) {
		t.Errorf("data: got %q, want %q", data, content)
	}
	if src.Type != SourceFile {
		t.Errorf("source type: got %q, want file", src.Type)
	}
	if src.Path != path {
		t.Errorf("source path: got %q, want %q", src.Path, path)
	}
	if src.Size != int64(len(content)) {
		t.Errorf("source size: got %d, want %d", src.Size, len(content))
	}
}

func TestResolve_FileFlagExplicit(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "a.bin")
	if err := os.WriteFile(path, []byte("abc"), 0644); err != nil {
		t.Fatal(err)
	}

	data, src, err := Resolve(context.Background(), Options{FileFlag: path})
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if string(data) != "abc" {
		t.Errorf("data: got %q", data)
	}
	if src.Type != SourceFile {
		t.Errorf("source type: got %q", src.Type)
	}
}

func TestResolve_FileFlagPickerWithPositional(t *testing.T) {
	// When FileFlag is the picker sentinel AND a positional arg is also
	// provided, the positional should win (historical verify behaviour).
	dir := t.TempDir()
	path := filepath.Join(dir, "proof.json")
	if err := os.WriteFile(path, []byte("{}"), 0644); err != nil {
		t.Fatal(err)
	}

	_, src, err := Resolve(context.Background(), Options{
		FileFlag:      FilePickSentinel,
		PositionalArg: path,
	})
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if src.Type != SourceFile {
		t.Errorf("source type: got %q, want file (picker fallback skipped)", src.Type)
	}
}

func TestResolve_PositionalURL_AutoDetect(t *testing.T) {
	// AutoDetectURL off: positional is treated as a file path (and the
	// file doesn't exist, so we get an error).
	_, _, err := Resolve(context.Background(), Options{PositionalArg: "https://example.com/x"})
	if err == nil {
		t.Fatal("expected read error for URL-like positional without AutoDetectURL")
	}

	// AutoDetectURL on but URL unreachable: we should hit the HTTP layer.
	// Using a port that's guaranteed refused keeps it fast and offline.
	_, _, err = Resolve(context.Background(), Options{
		PositionalArg: "http://127.0.0.1:1/nonexistent",
		AutoDetectURL: true,
	})
	if err == nil {
		t.Fatal("expected download error for unreachable URL")
	}
}

func TestResolve_DashPositionalNotAllowed(t *testing.T) {
	_, _, err := Resolve(context.Background(), Options{PositionalArg: "-"})
	if err == nil {
		t.Fatal("expected error for '-' positional when AllowStdin=false")
	}
}

func TestResolve_NoInput(t *testing.T) {
	// No file flag, no url flag, no positional, and AllowStdin=false.
	_, _, err := Resolve(context.Background(), Options{})
	if !errors.Is(err, ErrNoInput) {
		t.Errorf("expected ErrNoInput, got %v", err)
	}
}

func TestResolve_MissingFile(t *testing.T) {
	_, _, err := Resolve(context.Background(), Options{FileFlag: "/nonexistent/definitely/missing.bin"})
	if err == nil {
		t.Fatal("expected error for missing file")
	}
}

func TestResolve_URLFlagInvalid(t *testing.T) {
	_, _, err := Resolve(context.Background(), Options{URLFlag: "not-a-url"})
	if err == nil {
		t.Fatal("expected error for non-http(s) URL flag")
	}
}

func TestResolveStream_File(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "stream.txt")
	if err := os.WriteFile(path, []byte("stream me"), 0644); err != nil {
		t.Fatal(err)
	}

	r, src, err := ResolveStream(context.Background(), Options{PositionalArg: path})
	if err != nil {
		t.Fatalf("ResolveStream: %v", err)
	}
	defer r.Close()

	buf := make([]byte, 16)
	n, _ := r.Read(buf)
	if string(buf[:n]) != "stream me" {
		t.Errorf("stream content: got %q", buf[:n])
	}
	if src.Size != 9 {
		t.Errorf("source size: got %d, want 9", src.Size)
	}
}

func TestSource_DisplayName(t *testing.T) {
	cases := []struct {
		name string
		src  Source
		want string
	}{
		{"file", Source{Type: SourceFile, Path: "/tmp/a.bin"}, "/tmp/a.bin"},
		{"stdin", Source{Type: SourceStdin, Path: "-"}, "-"},
		{"url", Source{Type: SourceURL, Path: "https://x/y"}, "https://x/y"},
		{"empty", Source{}, "-"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := c.src.DisplayName(); got != c.want {
				t.Errorf("got %q, want %q", got, c.want)
			}
		})
	}
}

func TestIsHTTPURL(t *testing.T) {
	cases := []struct {
		in   string
		want bool
	}{
		{"https://example.com/path", true},
		{"http://example.com", true},
		{"file:///tmp/x", false},
		{"/usr/bin/local", false},
		{"example.com", false},
		{"", false},
	}
	for _, c := range cases {
		if got := isHTTPURL(c.in); got != c.want {
			t.Errorf("isHTTPURL(%q): got %v, want %v", c.in, got, c.want)
		}
	}
}
