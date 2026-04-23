// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: MIT

package selfupgrade

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSHA256File(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	path := filepath.Join(dir, "sample.bin")
	content := []byte("truestamp test vector\n")
	if err := os.WriteFile(path, content, 0644); err != nil {
		t.Fatal(err)
	}

	// Computed once with `printf 'truestamp test vector\n' | shasum -a 256`.
	const expected = "7a95b30a8d866a3f78ad149adf58042eb777f6b665a4ff0b96d3b0f2985381b9"

	got, err := SHA256File(path)
	if err != nil {
		t.Fatalf("SHA256File error: %v", err)
	}
	if got != expected {
		t.Errorf("SHA256File = %q, want %q", got, expected)
	}
}

func TestChecksumFor(t *testing.T) {
	t.Parallel()
	body := []byte(`# comment line
abcd1234  other-file.tar.gz
deadbeef  truestamp-cli_0.3.0_darwin_arm64.tar.gz
# another comment
cafef00d  truestamp-cli_0.3.0_linux_amd64.tar.gz
`)

	got, err := ChecksumFor(body, "truestamp-cli_0.3.0_darwin_arm64.tar.gz")
	if err != nil {
		t.Fatalf("ChecksumFor unexpected error: %v", err)
	}
	if got != "deadbeef" {
		t.Errorf("ChecksumFor = %q, want %q", got, "deadbeef")
	}

	got, err = ChecksumFor(body, "truestamp-cli_0.3.0_linux_amd64.tar.gz")
	if err != nil {
		t.Fatalf("ChecksumFor unexpected error: %v", err)
	}
	if got != "cafef00d" {
		t.Errorf("ChecksumFor (linux/amd64) = %q, want cafef00d", got)
	}

	// Missing entry.
	_, err = ChecksumFor(body, "does-not-exist.tar.gz")
	if !errors.Is(err, ErrChecksumMissing) {
		t.Errorf("ChecksumFor missing entry: got err %v, want ErrChecksumMissing", err)
	}
}

func TestVerifySHA256(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	path := filepath.Join(dir, "v.bin")
	content := []byte("truestamp test vector\n")
	if err := os.WriteFile(path, content, 0644); err != nil {
		t.Fatal(err)
	}
	const expected = "7a95b30a8d866a3f78ad149adf58042eb777f6b665a4ff0b96d3b0f2985381b9"

	if err := VerifySHA256(path, expected); err != nil {
		t.Errorf("VerifySHA256 good path: unexpected error %v", err)
	}
	// Case-insensitive match should still succeed.
	if err := VerifySHA256(path, "7A95B30A8D866A3F78AD149ADF58042EB777F6B665A4FF0B96D3B0F2985381B9"); err != nil {
		t.Errorf("VerifySHA256 uppercase hex: unexpected error %v", err)
	}
	// Mismatch.
	if err := VerifySHA256(path, "deadbeef"); !errors.Is(err, ErrChecksumMismatch) {
		t.Errorf("VerifySHA256 mismatch: got %v, want ErrChecksumMismatch", err)
	}
}

func TestVerifyCosign_missingBinary_nonRequired(t *testing.T) {
	t.Parallel()
	// Force cosign lookup to miss by pointing PinnedPath at a
	// non-existent absolute path. Using the pin (rather than t.Setenv on
	// PATH) keeps the test parallel-safe: resolveCosignBinary never
	// touches $PATH when pinned is non-empty.
	missing := filepath.Join(t.TempDir(), "no-such-cosign")
	verified, err := VerifyCosign("/nonexistent", "/nonexistent", DefaultCosignOptions(false, missing))
	if err != nil {
		t.Errorf("VerifyCosign (best-effort): unexpected error %v", err)
	}
	if verified {
		t.Errorf("VerifyCosign (best-effort): verified=true, want false (skipped)")
	}
}

func TestVerifyCosign_missingBinary_required(t *testing.T) {
	t.Parallel()
	missing := filepath.Join(t.TempDir(), "no-such-cosign")
	_, err := VerifyCosign("/nonexistent", "/nonexistent", DefaultCosignOptions(true, missing))
	if !errors.Is(err, ErrCosignMissing) {
		t.Errorf("VerifyCosign (required): err = %v, want ErrCosignMissing", err)
	}
}

// TestResolveCosignBinary_pinnedPath covers resolveCosignBinary's
// per-case behavior when a caller passes a pinned path: valid
// executable wins; invalid forms are rejected without $PATH fallback.
func TestResolveCosignBinary_pinnedPath(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	fake := filepath.Join(dir, "fake-cosign")
	if err := os.WriteFile(fake, []byte("#!/bin/sh\nexit 0\n"), 0755); err != nil {
		t.Fatalf("write fake cosign: %v", err)
	}

	// Valid executable: returned verbatim.
	got, err := resolveCosignBinary(fake)
	if err != nil {
		t.Fatalf("resolveCosignBinary with valid pin: unexpected error %v", err)
	}
	if got != fake {
		t.Errorf("resolveCosignBinary = %q, want %q", got, fake)
	}

	// Non-existent path.
	if _, err := resolveCosignBinary(filepath.Join(dir, "does-not-exist")); err == nil {
		t.Error("resolveCosignBinary with missing path: expected error, got nil")
	}

	// Directory, not a file.
	if _, err := resolveCosignBinary(dir); err == nil {
		t.Error("resolveCosignBinary with directory: expected error, got nil")
	}

	// Non-executable file.
	nonExec := filepath.Join(dir, "not-executable")
	if err := os.WriteFile(nonExec, []byte("plain"), 0644); err != nil {
		t.Fatalf("write non-exec: %v", err)
	}
	if _, err := resolveCosignBinary(nonExec); err == nil {
		t.Error("resolveCosignBinary with non-executable: expected error, got nil")
	}

	// Relative path: rejected up front.
	if _, err := resolveCosignBinary("cosign"); err == nil || !strings.Contains(err.Error(), "absolute") {
		t.Errorf("resolveCosignBinary with relative path: got %v, want error mentioning 'absolute'", err)
	}

	// Whitespace-only is treated as empty (falls back to $PATH); this
	// call may or may not succeed depending on whether cosign is
	// installed on the test host. Either is acceptable — we just want
	// to verify the path isn't pinned-validated.
	_, _ = resolveCosignBinary("   ")
}
