// Copyright (c) 2021-2026 Truestamp, Inc.
// SPDX-License-Identifier: MIT

package selfupgrade

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestSHA256File(t *testing.T) {
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
	// Force cosign lookup to miss by pointing PATH at an empty dir.
	emptyDir := t.TempDir()
	t.Setenv("PATH", emptyDir)

	verified, err := VerifyCosign("/nonexistent", "/nonexistent", DefaultCosignOptions(false))
	if err != nil {
		t.Errorf("VerifyCosign (best-effort): unexpected error %v", err)
	}
	if verified {
		t.Errorf("VerifyCosign (best-effort): verified=true, want false (skipped)")
	}
}

func TestVerifyCosign_missingBinary_required(t *testing.T) {
	emptyDir := t.TempDir()
	t.Setenv("PATH", emptyDir)

	_, err := VerifyCosign("/nonexistent", "/nonexistent", DefaultCosignOptions(true))
	if !errors.Is(err, ErrCosignMissing) {
		t.Errorf("VerifyCosign (required): err = %v, want ErrCosignMissing", err)
	}
}
