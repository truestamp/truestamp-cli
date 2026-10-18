// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: Apache-2.0

package upgradecheck

import (
	"context"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/truestamp/truestamp-cli/internal/selfupgrade"
	"golang.org/x/term"
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

// isTerminal reports whether f is a terminal. It goes through
// golang.org/x/term.IsTerminal (tcgetattr), as internal/inputsrc does,
// rather than the os.ModeCharDevice bit: /dev/null is a character device
// too, so a stderr of /dev/null would count as a terminal and turn the
// notice on with no one to read it.
func isTerminal(f *os.File) bool {
	if f == nil {
		return false
	}
	return term.IsTerminal(int(f.Fd()))
}

// MaybeNotify writes a single-line "new version available" notice to w if
// an upgrade is available. No-op when Disabled() returns true.
//
// The flow:
//  1. If cache is fresh (<24h), use the cached result, no network.
//  2. Otherwise, record the attempt in the cache first, keeping the
//     latest version an earlier check found, so the attempt counts for
//     the TTL however it ends. Then spawn a goroutine with a 2-second
//     deadline to refresh the cache, and wait up to waitBudget for it.
//  3. A refresh that succeeds in time overwrites the record with what it
//     found. One that fails ends the wait at once, and the record stands.
//     One still running when the wait runs out is abandoned when the
//     command exits, and the record stands too. A run that cannot reach
//     GitHub (offline, rate limited, the releases gone, packets dropped)
//     therefore pays for one check per TTL, not the wait on every command.
//
// All errors are swallowed, upgrade checks must never interfere with
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

	known := ""
	if cache != nil {
		known = cache.LatestVersion
	}
	// Record the attempt before making it: the command exits when the wait
	// below ends, so a check still running then never gets to write.
	_ = WriteCache(Cache{
		LastChecked:        time.Now().UTC(),
		LatestVersion:      known,
		CheckedFromVersion: currentVersion,
	})

	// Cache stale or missing: refresh asynchronously with a short wait.
	// The goroutine sends exactly once: the version to notify about, which
	// for a failed check is the one an earlier check found, or "" when
	// there is none to show (no earlier check, or a pre-release latest).
	result := make(chan string, 1)
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		rel, err := fetchLatest(ctx)
		if err != nil || rel == nil {
			result <- known
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
			result <- ""
			return
		}
		result <- rel.TagName
	}()

	select {
	case latest := <-result:
		emitIfNewer(w, currentVersion, latest)
	case <-time.After(waitBudget):
		// Refresh didn't complete in time. The command exits and abandons
		// it; the attempt recorded above stands for the TTL.
	}
}

// fetchLatest is the hook MaybeNotify resolves the latest release with.
// Tests replace it to hold a check open past the wait.
var fetchLatest = selfupgrade.FetchLatest

// waitBudget is how long MaybeNotify waits for a refresh before the
// command exits without a notice. Tests lengthen it to tell a wait that
// ended because the check finished from one that ran out.
var waitBudget = 500 * time.Millisecond

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
