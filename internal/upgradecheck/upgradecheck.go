// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: MIT

package upgradecheck

import (
	"context"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/truestamp/truestamp-cli/internal/selfupgrade"
)

// terminalCheck is the hook used by Disabled to detect a TTY. Tests
// replace it to exercise the TTY-true branch without a real terminal.
var terminalCheck = isTerminal

// Disabled reports whether passive upgrade checks are suppressed in the
// current environment. The answer combines: explicit opt-out flag,
// TRUESTAMP_NO_UPGRADE_CHECK env var, CI detection, non-TTY stderr, and
// a "dev" current version.
//
// The cmd layer is responsible for reading the --no-upgrade-check flag
// and passing it in via flagDisabled.
func Disabled(flagDisabled bool, stderr *os.File, currentVersion string) bool {
	if flagDisabled {
		return true
	}
	if os.Getenv("TRUESTAMP_NO_UPGRADE_CHECK") != "" {
		return true
	}
	if currentVersion == "" || currentVersion == "dev" {
		return true
	}
	if inCI() {
		return true
	}
	if !terminalCheck(stderr) {
		return true
	}
	return false
}

// ciEnvVars are the env vars our suppression rules respect. A
// non-empty value for any of them flips the check off.
var ciEnvVars = []string{
	"CI",
	"GITHUB_ACTIONS",
	"GITLAB_CI",
	"CIRCLECI",
	"BUILDKITE",
	"JENKINS_HOME",
	"TF_BUILD", // Azure Pipelines
}

func inCI() bool {
	for _, v := range ciEnvVars {
		if os.Getenv(v) != "" {
			return true
		}
	}
	return false
}

// isTerminal reports whether f refers to a character device (a TTY).
// Uses os.Stat and the ModeCharDevice bit — no new dependencies needed.
func isTerminal(f *os.File) bool {
	if f == nil {
		return false
	}
	info, err := f.Stat()
	if err != nil {
		return false
	}
	return (info.Mode() & os.ModeCharDevice) != 0
}

// MaybeNotify writes a single-line "new version available" notice to w if
// an upgrade is available. No-op when Disabled() returns true.
//
// The flow:
//  1. If cache is fresh (<24h), use the cached result — no network.
//  2. Otherwise, spawn a goroutine with a 2-second deadline to refresh
//     the cache. Wait up to 500ms for it to complete; if it doesn't, we
//     skip emitting a notice this invocation. The cache write is still
//     committed when the goroutine finishes, so the next run picks it up.
//
// All errors are swallowed — upgrade checks must never interfere with
// the primary command's output.
func MaybeNotify(w io.Writer, flagDisabled bool, currentVersion string) {
	if Disabled(flagDisabled, os.Stderr, currentVersion) {
		return
	}

	cache, _ := ReadCache()
	if cache != nil && cache.IsFresh(DefaultTTL) {
		emitIfNewer(w, currentVersion, cache.LatestVersion)
		return
	}

	// Cache stale or missing: refresh asynchronously with a short wait.
	result := make(chan string, 1)
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		rel, err := selfupgrade.FetchLatest(ctx)
		if err != nil || rel == nil {
			return
		}
		_ = WriteCache(Cache{
			LastChecked:        time.Now().UTC(),
			LatestVersion:      rel.TagName,
			CheckedFromVersion: currentVersion,
		})
		// If the latest is a pre-release, we still cache the version
		// string (so we don't re-fetch for 24h), but we never notify.
		if isPreReleaseTagName(rel.TagName) || rel.Prerelease {
			return
		}
		select {
		case result <- rel.TagName:
		default:
		}
	}()

	select {
	case latest := <-result:
		emitIfNewer(w, currentVersion, latest)
	case <-time.After(500 * time.Millisecond):
		// Refresh didn't complete in time; next invocation will see it.
	}
}

func emitIfNewer(w io.Writer, current, latest string) {
	if latest == "" || current == "" {
		return
	}
	cur, errA := selfupgrade.ParseSemver(current)
	lat, errB := selfupgrade.ParseSemver(latest)
	if errA != nil || errB != nil {
		return
	}
	if lat.IsPreRelease() {
		return
	}
	// Same rule as the upgrade command: for git-describe dev builds
	// (e.g. "0.5.0-4-g356ee75-dirty") compare cores only, so we don't
	// nag a developer to "upgrade" back to the base tag they're already
	// ahead of. A real new release (0.5.1, 0.6.0, etc.) still notifies.
	if !selfupgrade.UpgradeAvailable(cur, lat) {
		return
	}
	fmt.Fprintf(w, "\nnote: truestamp %s is available (current: %s).\n", selfupgrade.Display(latest), selfupgrade.Display(current))
	fmt.Fprintf(w, "      run `truestamp upgrade` to install it, or set TRUESTAMP_NO_UPGRADE_CHECK=1 to silence this.\n\n")
}

// isPreReleaseTagName is a thin wrapper letting us check the tag string
// alone (without a full Release object) when we're reading from the
// cache.
func isPreReleaseTagName(tag string) bool {
	v, err := selfupgrade.ParseSemver(tag)
	if err != nil {
		return false
	}
	return v.IsPreRelease()
}
