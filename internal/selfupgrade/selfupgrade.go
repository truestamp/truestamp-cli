// Copyright (c) 2021-2026 Truestamp, Inc.
// SPDX-License-Identifier: MIT

// Package selfupgrade implements the in-place upgrade flow for users who
// installed truestamp via docs/install.sh or manual tarball extraction.
// The behavior intentionally mirrors install.sh so the security posture
// is identical: download archive + checksums + signature bundle, verify
// SHA-256 (mandatory, pure Go), verify cosign (best-effort shell-out),
// extract, atomic replace, clear quarantine xattr on darwin.
//
// Homebrew and `go install` users never reach this package — the cmd
// layer detects the install method and prints the correct package-
// manager instruction instead.
package selfupgrade

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/truestamp/truestamp-cli/internal/httpclient"
	"github.com/truestamp/truestamp-cli/internal/install"
)

// BinaryName is the binary inside each release tarball.
const BinaryName = "truestamp"

// ArchiveProject is the filename stem used by GoReleaser for the
// tarball. Matches docs/install.sh PROJECT.
const ArchiveProject = "truestamp-cli"

// ErrWindowsUnsupported is returned when self-upgrade is attempted on
// Windows. Callers should print `go install ...@latest` instead.
var ErrWindowsUnsupported = errors.New("in-place upgrade is not supported on Windows; run: go install github.com/truestamp/truestamp-cli/cmd/truestamp@latest")

// ErrNotUpgradable is returned when the running binary is in a directory
// the current user cannot write to (e.g. /usr/local/bin owned by root).
var ErrNotUpgradable = errors.New("current binary is not user-writable; re-run with sudo or reinstall via install.sh")

// ErrAlreadyCurrent is returned when the requested target version is
// already running. Callers treat this as a no-op success.
var ErrAlreadyCurrent = errors.New("already on the requested version")

// ErrPreRelease is returned when resolveVersion resolves `latest` to a
// pre-release tag and the caller did not pin --version explicitly. The
// cmd layer translates this to exit code 3 and a helpful message.
var ErrPreRelease = errors.New("latest release is a pre-release")

// Options configures an Upgrade() or Check() call.
type Options struct {
	// TargetVersion overrides the default of "latest" (i.e. the user
	// passed --version). Leading "v" is auto-added when missing so
	// callers can pass "0.4.0" or "v0.4.0".
	TargetVersion string

	// CurrentVersion is the running binary's version string, typically
	// version.Version. Used for comparison and backup naming.
	CurrentVersion string

	// RequireCosign corresponds to TRUESTAMP_REQUIRE_COSIGN=1 in
	// install.sh. When true, missing cosign binary or missing bundle
	// becomes a hard error.
	RequireCosign bool

	// SkipCosign disables cosign verification entirely (SHA-256 still
	// mandatory). Corresponds to `truestamp upgrade --no-verify` and
	// TRUESTAMP_SKIP_CHECKSUM=0 parity is intentional only for SHA-256.
	SkipCosign bool

	// Logger receives progress lines. May be nil for silent operation.
	Logger func(msg string)
}

// CheckResult describes the outcome of a dry-run check.
type CheckResult struct {
	CurrentVersion string
	LatestVersion  string
	UpgradeAvail   bool
	PreRelease     bool // latest is a pre-release (GitHub flag OR semver -suffix)
}

// Check resolves the latest release without downloading any binaries. It
// honors the same pre-release filtering as Upgrade().
func Check(ctx context.Context, opts Options) (*CheckResult, error) {
	rel, err := resolveRelease(ctx, opts)
	if err != nil {
		return nil, err
	}

	result := &CheckResult{
		CurrentVersion: opts.CurrentVersion,
		LatestVersion:  rel.TagName,
		PreRelease:     isPreReleaseTag(rel),
	}

	// If the resolved release is a pre-release and the user didn't pin
	// via --version, surface that explicitly.
	if result.PreRelease && opts.TargetVersion == "" {
		return result, ErrPreRelease
	}

	cur, curErr := ParseSemver(opts.CurrentVersion)
	latest, latestErr := ParseSemver(rel.TagName)
	if curErr != nil || latestErr != nil {
		// If either side is unparseable (e.g. current is "dev"), offer
		// the upgrade — it's better to over-report than let someone
		// stay on a development build.
		result.UpgradeAvail = curErr != nil && latestErr == nil
		return result, nil
	}
	result.UpgradeAvail = latest.Compare(cur) > 0
	return result, nil
}

// Upgrade performs the full in-place upgrade flow. Returns the installed
// version on success.
func Upgrade(ctx context.Context, opts Options) (installedVersion string, backupPath string, err error) {
	if runtime.GOOS == "windows" {
		return "", "", ErrWindowsUnsupported
	}

	rel, err := resolveRelease(ctx, opts)
	if err != nil {
		return "", "", err
	}

	if isPreReleaseTag(rel) && opts.TargetVersion == "" {
		return rel.TagName, "", ErrPreRelease
	}

	logf := opts.Logger
	if logf == nil {
		logf = func(string) {}
	}

	// Short-circuit if already at target.
	if samePatch(opts.CurrentVersion, rel.TagName) {
		return rel.TagName, "", ErrAlreadyCurrent
	}

	// Pre-flight: is the destination writable?
	if !install.BinaryWritable() {
		return "", "", ErrNotUpgradable
	}

	osName, archName, err := hostOSArch()
	if err != nil {
		return "", "", err
	}

	verNoV := strings.TrimPrefix(rel.TagName, "v")
	archiveName := fmt.Sprintf("%s_%s_%s_%s.tar.gz", ArchiveProject, verNoV, osName, archName)
	baseURL := fmt.Sprintf("https://github.com/%s/releases/download/%s", ReleasesRepo, rel.TagName)
	archiveURL := baseURL + "/" + archiveName
	checksumsURL := baseURL + "/checksums.txt"
	bundleURL := baseURL + "/checksums.txt.sigstore"

	// Temp workspace; cleaned up on return.
	tmpDir, err := os.MkdirTemp("", "truestamp-upgrade-*")
	if err != nil {
		return "", "", fmt.Errorf("create temp dir: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	// 1. Download archive.
	logf(fmt.Sprintf("downloading %s", archiveName))
	archivePath := filepath.Join(tmpDir, archiveName)
	if _, err := httpclient.DownloadCtx(ctx, archiveURL, archivePath, 0); err != nil {
		return "", "", fmt.Errorf("download archive: %w", err)
	}

	// 2. Download checksums.txt.
	logf("verifying SHA-256")
	checksumsPath := filepath.Join(tmpDir, "checksums.txt")
	checksumsBody, err := httpclient.DownloadBytesCtx(ctx, checksumsURL, 0)
	if err != nil {
		return "", "", fmt.Errorf("download checksums: %w", err)
	}
	if err := os.WriteFile(checksumsPath, checksumsBody, 0644); err != nil {
		return "", "", fmt.Errorf("write checksums: %w", err)
	}

	// 3. Best-effort / required cosign verification of checksums.txt.
	if !opts.SkipCosign {
		bundlePath := filepath.Join(tmpDir, "checksums.txt.sigstore")
		bundleBody, bundleErr := httpclient.DownloadBytesCtx(ctx, bundleURL, 0)
		if bundleErr == nil {
			_ = os.WriteFile(bundlePath, bundleBody, 0644)
		} else {
			bundlePath = "" // signals "bundle missing" to VerifyCosign
		}
		verified, err := VerifyCosign(checksumsPath, bundlePath, DefaultCosignOptions(opts.RequireCosign))
		if err != nil {
			return "", "", err
		}
		if verified {
			logf("cosign signature verified")
		} else if opts.RequireCosign {
			// VerifyCosign should have already errored; defensive.
			return "", "", ErrCosignVerify
		} else {
			logf("cosign not available; signature verification skipped (SHA-256 still enforced)")
		}
	}

	// 4. Verify SHA-256 of archive against checksums.txt entry.
	expected, err := ChecksumFor(checksumsBody, archiveName)
	if err != nil {
		return "", "", fmt.Errorf("lookup checksum: %w", err)
	}
	if err := VerifySHA256(archivePath, expected); err != nil {
		return "", "", err
	}

	// 5. Extract binary from tarball.
	logf("extracting")
	newBinPath, err := ExtractBinary(archivePath, BinaryName, tmpDir)
	if err != nil {
		return "", "", err
	}

	// 6. Atomic replace.
	currentExe := install.Executable()
	if currentExe == "" {
		return "", "", fmt.Errorf("cannot determine current binary path")
	}
	logf(fmt.Sprintf("installing to %s", currentExe))
	backup, err := Replace(currentExe, newBinPath)
	if err != nil {
		return "", "", err
	}

	return rel.TagName, backup, nil
}

// resolveRelease picks the right Release for an Upgrade / Check call.
// Honors opts.TargetVersion when non-empty, else pulls /latest.
func resolveRelease(ctx context.Context, opts Options) (*Release, error) {
	if opts.TargetVersion != "" {
		tag := opts.TargetVersion
		if !strings.HasPrefix(tag, "v") {
			tag = "v" + tag
		}
		return FetchByTag(ctx, tag)
	}
	return FetchLatest(ctx)
}

// isPreReleaseTag combines GitHub's prerelease flag with a semver-suffix
// defense. Either signal is sufficient.
func isPreReleaseTag(rel *Release) bool {
	if rel.Prerelease {
		return true
	}
	v, err := ParseSemver(rel.TagName)
	if err != nil {
		return false
	}
	return v.IsPreRelease()
}

// samePatch returns true if two version strings parse to the same
// MAJOR.MINOR.PATCH (pre-release differences are considered "different"
// so a --version pin to an rc DOES reinstall).
func samePatch(a, b string) bool {
	va, errA := ParseSemver(a)
	vb, errB := ParseSemver(b)
	if errA != nil || errB != nil {
		return false
	}
	return va.Major == vb.Major && va.Minor == vb.Minor &&
		va.Patch == vb.Patch && va.PreRelease == vb.PreRelease
}

// hostOSArch returns ("darwin"|"linux", "amd64"|"arm64") for the
// currently running binary, or an error for unsupported combinations.
// Matches docs/install.sh detect_os_arch().
func hostOSArch() (string, string, error) {
	var osName string
	switch runtime.GOOS {
	case "darwin":
		osName = "darwin"
	case "linux":
		osName = "linux"
	default:
		return "", "", fmt.Errorf("unsupported OS for in-place upgrade: %s", runtime.GOOS)
	}
	var archName string
	switch runtime.GOARCH {
	case "amd64":
		archName = "amd64"
	case "arm64":
		archName = "arm64"
	default:
		return "", "", fmt.Errorf("unsupported architecture for in-place upgrade: %s", runtime.GOARCH)
	}
	return osName, archName, nil
}
