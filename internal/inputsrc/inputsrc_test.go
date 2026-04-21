// Copyright (c) 2021-2026 Truestamp, Inc.
// SPDX-License-Identifier: MIT

package inputsrc

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
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

// --- URL / HTTP paths ----------------------------------------------------

func TestResolve_URLFlag(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("downloaded body"))
	}))
	defer srv.Close()

	data, src, err := Resolve(context.Background(), Options{URLFlag: srv.URL})
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if string(data) != "downloaded body" {
		t.Errorf("data: got %q", data)
	}
	if src.Type != SourceURL {
		t.Errorf("type: got %q, want url", src.Type)
	}
	if src.Path != srv.URL {
		t.Errorf("path: got %q", src.Path)
	}
	if src.Size != int64(len("downloaded body")) {
		t.Errorf("size: got %d", src.Size)
	}
}

func TestResolve_PositionalURL_AutoDetect_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("auto"))
	}))
	defer srv.Close()
	data, src, err := Resolve(context.Background(), Options{
		PositionalArg: srv.URL,
		AutoDetectURL: true,
	})
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if src.Type != SourceURL {
		t.Errorf("type: got %q", src.Type)
	}
	if string(data) != "auto" {
		t.Errorf("data: got %q", data)
	}
}

func TestResolve_URLPromptSentinel_WithPositional(t *testing.T) {
	// URLFlag=sentinel + an http-looking positional should download that
	// URL rather than invoking the (interactive) prompt.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("positional-wins"))
	}))
	defer srv.Close()

	data, src, err := Resolve(context.Background(), Options{
		URLFlag:       URLPromptSentinel,
		PositionalArg: srv.URL,
	})
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if src.Type != SourceURL {
		t.Errorf("type: got %q, want url", src.Type)
	}
	if string(data) != "positional-wins" {
		t.Errorf("data: got %q", data)
	}
}

func TestResolve_URL_MaxBytesCap(t *testing.T) {
	// Server returns 1024 bytes; cap at 32 → expect overflow error.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write(make([]byte, 1024))
	}))
	defer srv.Close()

	_, _, err := Resolve(context.Background(), Options{URLFlag: srv.URL, MaxBytes: 32})
	if err == nil {
		t.Fatal("expected overflow error")
	}
}

func TestResolveStream_URL(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("stream-url"))
	}))
	defer srv.Close()

	r, src, err := ResolveStream(context.Background(), Options{URLFlag: srv.URL})
	if err != nil {
		t.Fatalf("ResolveStream: %v", err)
	}
	defer r.Close()
	got, _ := io.ReadAll(r)
	if string(got) != "stream-url" {
		t.Errorf("body: got %q", got)
	}
	if src.Type != SourceURL {
		t.Errorf("type: got %q", src.Type)
	}
}

func TestResolveStream_NoInput(t *testing.T) {
	_, _, err := ResolveStream(context.Background(), Options{})
	if !errors.Is(err, ErrNoInput) {
		t.Errorf("expected ErrNoInput, got %v", err)
	}
}

func TestResolveStream_MissingFile(t *testing.T) {
	_, _, err := ResolveStream(context.Background(), Options{FileFlag: "/definitely/missing"})
	if err == nil {
		t.Fatal("expected open error")
	}
}

// --- File picker (injected test seam) -----------------------------------

func TestResolve_FilePicker_Stubbed(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "picked.bin")
	if err := os.WriteFile(path, []byte("via-picker"), 0644); err != nil {
		t.Fatal(err)
	}

	orig := pickFileFunc
	pickFileFunc = func(opts Options) (string, error) {
		if opts.PickerTitle == "" {
			t.Errorf("picker should have received a title")
		}
		return path, nil
	}
	t.Cleanup(func() { pickFileFunc = orig })

	data, src, err := Resolve(context.Background(), Options{
		FileFlag:    FilePickSentinel,
		PickerTitle: "Pick something",
		PickerExts:  []string{".bin"},
	})
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if string(data) != "via-picker" {
		t.Errorf("data: got %q", data)
	}
	if src.Type != SourcePicker {
		t.Errorf("type: got %q, want picker", src.Type)
	}
}

func TestResolve_FilePicker_Cancelled(t *testing.T) {
	orig := pickFileFunc
	pickFileFunc = func(Options) (string, error) { return "", errors.New("user aborted") }
	t.Cleanup(func() { pickFileFunc = orig })

	_, _, err := Resolve(context.Background(), Options{FileFlag: FilePickSentinel})
	if err == nil {
		t.Fatal("expected picker-aborted error to propagate")
	}
}

func TestResolve_URLPrompt_Stubbed(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("prompted"))
	}))
	defer srv.Close()

	orig := promptURLFunc
	promptURLFunc = func(opts Options) (string, error) {
		if opts.URLPromptTitle == "" {
			t.Errorf("prompt should have a title")
		}
		return srv.URL, nil
	}
	t.Cleanup(func() { promptURLFunc = orig })

	data, src, err := Resolve(context.Background(), Options{
		URLFlag:        URLPromptSentinel,
		URLPromptTitle: "Enter URL",
	})
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if src.Type != SourceURL {
		t.Errorf("type: got %q", src.Type)
	}
	if string(data) != "prompted" {
		t.Errorf("data: got %q", data)
	}
}

func TestResolve_URLPrompt_Cancelled(t *testing.T) {
	orig := promptURLFunc
	promptURLFunc = func(Options) (string, error) { return "", errors.New("user aborted") }
	t.Cleanup(func() { promptURLFunc = orig })

	_, _, err := Resolve(context.Background(), Options{URLFlag: URLPromptSentinel})
	if err == nil {
		t.Fatal("expected prompt-aborted error to propagate")
	}
}

// (The previous TestDefaultPickFile_Forwards / TestDefaultPromptURL_Forwards
// tests were removed: they invoked the real interactive huh form, which
// in a TTY environment panics inside bubbletea during rendering. The
// pickFileFunc / promptURLFunc indirection is already exercised by the
// stubbed tests above; the two `default*` wrappers are one-liners that
// just call into ui.PickFile / huh.Form.Run and can't be tested without
// a headless terminal harness.)

// --- validateURL ---------------------------------------------------------

func TestValidateURL(t *testing.T) {
	cases := []struct {
		in string
		ok bool
	}{
		{"", true}, // empty is allowed (huh treats as pending input)
		{"https://example.com", true},
		{"http://x/y", true},
		{"ftp://no", false},
		{"just-a-string", false},
	}
	for _, c := range cases {
		err := validateURL(c.in)
		if c.ok && err != nil {
			t.Errorf("validateURL(%q) unexpected err: %v", c.in, err)
		}
		if !c.ok && err == nil {
			t.Errorf("validateURL(%q) should have rejected", c.in)
		}
	}
}

// --- stdin pipe & TTY helpers --------------------------------------------

func TestIsStdinPipe_WithReplacedStdin(t *testing.T) {
	// Replace stdin with a pipe so IsStdinPipe reports true.
	orig := os.Stdin
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	os.Stdin = r
	t.Cleanup(func() {
		os.Stdin = orig
		_ = w.Close()
		_ = r.Close()
	})
	if !IsStdinPipe() {
		t.Error("IsStdinPipe: want true with piped stdin")
	}
	if IsStdinTerminal() {
		t.Error("IsStdinTerminal: want false with piped stdin")
	}
}

func TestResolve_StdinFallback(t *testing.T) {
	orig := os.Stdin
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := w.Write([]byte("stdin-data")); err != nil {
		t.Fatal(err)
	}
	_ = w.Close()
	os.Stdin = r
	t.Cleanup(func() {
		os.Stdin = orig
		_ = r.Close()
	})
	data, src, err := Resolve(context.Background(), Options{AllowStdin: true})
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if string(data) != "stdin-data" {
		t.Errorf("data: got %q", data)
	}
	if src.Type != SourceStdin {
		t.Errorf("type: got %q, want stdin", src.Type)
	}
}

func TestResolve_DashStdin(t *testing.T) {
	orig := os.Stdin
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := w.Write([]byte("via-dash")); err != nil {
		t.Fatal(err)
	}
	_ = w.Close()
	os.Stdin = r
	t.Cleanup(func() { os.Stdin = orig; _ = r.Close() })

	data, src, err := Resolve(context.Background(), Options{
		PositionalArg: "-",
		AllowStdin:    true,
	})
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if src.Type != SourceStdin {
		t.Errorf("type: got %q", src.Type)
	}
	if string(data) != "via-dash" {
		t.Errorf("data: got %q", data)
	}
}

func TestResolve_StdinMaxBytesOverflow(t *testing.T) {
	orig := os.Stdin
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	// Send more bytes than the cap so the overflow branch fires.
	go func() {
		defer w.Close()
		_, _ = w.Write([]byte(strings.Repeat("x", 128)))
	}()
	os.Stdin = r
	t.Cleanup(func() { os.Stdin = orig; _ = r.Close() })

	_, _, err = Resolve(context.Background(), Options{AllowStdin: true, MaxBytes: 32})
	if err == nil {
		t.Fatal("expected overflow error")
	}
}

func TestResolveStream_Stdin(t *testing.T) {
	orig := os.Stdin
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := w.Write([]byte("streamed")); err != nil {
		t.Fatal(err)
	}
	_ = w.Close()
	os.Stdin = r
	t.Cleanup(func() { os.Stdin = orig; _ = r.Close() })

	rc, src, err := ResolveStream(context.Background(), Options{AllowStdin: true})
	if err != nil {
		t.Fatalf("ResolveStream: %v", err)
	}
	defer rc.Close()
	got, _ := io.ReadAll(rc)
	if string(got) != "streamed" {
		t.Errorf("body: got %q", got)
	}
	if src.Type != SourceStdin {
		t.Errorf("type: got %q", src.Type)
	}
}

// --- downloadURL errors --------------------------------------------------

func TestDownloadURL_NonHTTPScheme(t *testing.T) {
	_, err := downloadURL(context.Background(), "file:///tmp/nope", Options{})
	if err == nil {
		t.Fatal("expected scheme rejection")
	}
}
