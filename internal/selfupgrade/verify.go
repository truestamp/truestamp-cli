// Copyright (c) 2021-2026 Truestamp, Inc.
// SPDX-License-Identifier: MIT

package selfupgrade

import (
	"bufio"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
)

// ErrChecksumMismatch is returned when a computed SHA-256 does not match
// the value parsed from checksums.txt.
var ErrChecksumMismatch = errors.New("checksum mismatch")

// ErrChecksumMissing is returned when checksums.txt does not contain an
// entry for the expected archive name.
var ErrChecksumMissing = errors.New("archive not listed in checksums.txt")

// ErrCosignMissing is returned when TRUESTAMP_REQUIRE_COSIGN=1 is set but
// the `cosign` binary is not available on PATH.
var ErrCosignMissing = errors.New("cosign required but not on PATH")

// ErrCosignBundleMissing is returned when require-cosign is set and the
// signature bundle could not be downloaded for the release.
var ErrCosignBundleMissing = errors.New("cosign bundle not published for release")

// ErrCosignVerify is returned when `cosign verify-blob` exits non-zero.
var ErrCosignVerify = errors.New("cosign verification failed")

// SHA256File computes the lowercase hex SHA-256 digest of a file.
func SHA256File(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

// ChecksumFor parses a GoReleaser-style checksums.txt (`<hex>  <file>`
// per line) and returns the expected hex digest for archiveName, or
// ErrChecksumMissing if no such line exists.
func ChecksumFor(checksumsBody []byte, archiveName string) (string, error) {
	scanner := bufio.NewScanner(strings.NewReader(string(checksumsBody)))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		// GoReleaser emits `<hex><two-spaces><name>` for binary mode
		// but various hashers differ. Split on whitespace and take the
		// last field as the filename.
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		name := fields[len(fields)-1]
		if name == archiveName {
			return strings.ToLower(fields[0]), nil
		}
	}
	if err := scanner.Err(); err != nil {
		return "", err
	}
	return "", ErrChecksumMissing
}

// VerifySHA256 reads archivePath and asserts its SHA-256 matches
// expectedHex (case-insensitive). Returns ErrChecksumMismatch otherwise.
func VerifySHA256(archivePath, expectedHex string) error {
	actual, err := SHA256File(archivePath)
	if err != nil {
		return err
	}
	if !strings.EqualFold(actual, expectedHex) {
		return fmt.Errorf("%w: expected %s, got %s", ErrChecksumMismatch, expectedHex, actual)
	}
	return nil
}

// CosignOptions configures a cosign signature verification pass.
type CosignOptions struct {
	// Required mirrors TRUESTAMP_REQUIRE_COSIGN=1 from install.sh. When
	// true, missing cosign binary or bundle is a hard error.
	Required bool

	// CertIdentityRegexp is passed to cosign as
	// --certificate-identity-regexp. Must match the release workflow.
	CertIdentityRegexp string

	// OIDCIssuer is passed as --certificate-oidc-issuer.
	OIDCIssuer string
}

// DefaultCosignOptions returns the identity values wired into
// install.sh:256-261 — the release.yml workflow in truestamp/truestamp-cli
// signed by GitHub Actions' OIDC token endpoint.
func DefaultCosignOptions(required bool) CosignOptions {
	return CosignOptions{
		Required:           required,
		CertIdentityRegexp: `^https://github\.com/truestamp/truestamp-cli/\.github/workflows/release\.yml@`,
		OIDCIssuer:         "https://token.actions.githubusercontent.com",
	}
}

// VerifyCosign runs `cosign verify-blob --bundle=<bundlePath> ...
// <checksumsPath>`. If cosign is not on PATH or bundlePath is empty, the
// behavior depends on opts.Required: required=true returns an error,
// required=false returns (false, nil) to indicate "skipped, non-fatal".
// Returns (true, nil) when verification succeeds.
func VerifyCosign(checksumsPath, bundlePath string, opts CosignOptions) (bool, error) {
	cosignPath, err := exec.LookPath("cosign")
	if err != nil {
		if opts.Required {
			return false, ErrCosignMissing
		}
		return false, nil
	}
	if bundlePath == "" {
		if opts.Required {
			return false, ErrCosignBundleMissing
		}
		return false, nil
	}
	args := []string{
		"verify-blob",
		"--bundle", bundlePath,
		"--certificate-identity-regexp", opts.CertIdentityRegexp,
		"--certificate-oidc-issuer", opts.OIDCIssuer,
		checksumsPath,
	}
	cmd := exec.Command(cosignPath, args...)
	// Swallow cosign's own output; we only care about exit status.
	cmd.Stdout = io.Discard
	cmd.Stderr = io.Discard
	if err := cmd.Run(); err != nil {
		return false, fmt.Errorf("%w: %v", ErrCosignVerify, err)
	}
	return true, nil
}
