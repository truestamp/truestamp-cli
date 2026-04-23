// Copyright (c) 2019-2026 Truestamp, Inc.
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
	"path/filepath"
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

	// PinnedPath is an absolute path to the cosign binary. When empty,
	// $PATH is searched. Populated from config.Config.CosignPath which
	// in turn sources from config.toml or $TRUESTAMP_COSIGN_PATH. The
	// config layer validates the path is absolute; resolveCosignBinary
	// re-checks existence and the executable bit at use time.
	PinnedPath string

	// CertIdentityRegexp is passed to cosign as
	// --certificate-identity-regexp. Must match the release workflow.
	CertIdentityRegexp string

	// OIDCIssuer is passed as --certificate-oidc-issuer.
	OIDCIssuer string
}

// DefaultCosignOptions returns the identity values wired into
// install.sh:256-261 — the release.yml workflow in truestamp/truestamp-cli
// signed by GitHub Actions' OIDC token endpoint. pinnedPath is optional
// (empty = $PATH lookup) and is forwarded verbatim to CosignOptions.
func DefaultCosignOptions(required bool, pinnedPath string) CosignOptions {
	return CosignOptions{
		Required:           required,
		PinnedPath:         pinnedPath,
		CertIdentityRegexp: `^https://github\.com/truestamp/truestamp-cli/\.github/workflows/release\.yml@`,
		OIDCIssuer:         "https://token.actions.githubusercontent.com",
	}
}

// resolveCosignBinary returns a usable path to the cosign binary.
// pinned, when non-empty, must be an absolute path to an executable
// file — operators use this to avoid $PATH hijacking in hardened
// environments. The absolute-path check is redundant with the
// validation in config.Load but defensive: this function is also
// callable from tests and any future direct caller.
//
// When pinned is empty, falls back to exec.LookPath("cosign").
//
// The G703 suppression below is the tightest surface gosec worries
// about: by the time we call os.Stat, the input is a caller-supplied
// absolute path, and the result only gates whether we return that
// exact path as a command to exec — no other filesystem access is
// derived from it.
func resolveCosignBinary(pinned string) (string, error) {
	pinned = strings.TrimSpace(pinned)
	if pinned != "" {
		if !filepath.IsAbs(pinned) {
			return "", fmt.Errorf("cosign_path %q must be an absolute path", pinned)
		}
		info, err := os.Stat(pinned) //#nosec G304 G703 -- operator-pinned absolute path
		if err != nil {
			return "", fmt.Errorf("cosign_path %q: %w", pinned, err)
		}
		if info.IsDir() || info.Mode()&0111 == 0 {
			return "", fmt.Errorf("cosign_path %q is not an executable file", pinned)
		}
		return pinned, nil
	}
	return exec.LookPath("cosign")
}

// VerifyCosign runs `cosign verify-blob --bundle=<bundlePath> ...
// <checksumsPath>`. If cosign is not on PATH or bundlePath is empty, the
// behavior depends on opts.Required: required=true returns an error,
// required=false returns (false, nil) to indicate "skipped, non-fatal".
// Returns (true, nil) when verification succeeds.
//
// Resolution order for the cosign binary:
//  1. $TRUESTAMP_COSIGN_PATH (absolute path; lets hardened environments
//     pin cosign to a known location and avoid $PATH hijacking).
//  2. exec.LookPath("cosign") (the usual $PATH search).
//
// The SHA-256 verification in VerifySHA256 is mandatory regardless of
// what this function does, so a cosign-layer miss is defense-in-depth,
// not a single point of failure.
func VerifyCosign(checksumsPath, bundlePath string, opts CosignOptions) (bool, error) {
	cosignPath, err := resolveCosignBinary(opts.PinnedPath)
	if err != nil {
		if opts.Required {
			return false, fmt.Errorf("%w: %w", ErrCosignMissing, err)
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
