// Copyright (c) 2021-2026 Truestamp, Inc.
// SPDX-License-Identifier: MIT

package selfupgrade

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// Semver is a minimal, lenient semver parse adequate for release-tag
// comparison. Release tags in this project are strict
// `v<MAJOR>.<MINOR>.<PATCH>[-<PRE>][+<BUILD>]` with no looser shapes. The
// PreRelease slot is populated from the string after `-`; any value there
// is treated as pre-release for the purposes of PreRelease().
type Semver struct {
	Major, Minor, Patch int
	PreRelease          string // empty for stable releases
	BuildMetadata       string // content after `+`
	Raw                 string // original input, leading "v" preserved
}

// ParseSemver accepts tags like "v0.3.0", "0.3.0", "v1.0.0-rc.1",
// "v1.0.0-beta.2+build.7". Returns an error for shapes it can't parse
// (not a version string, or non-numeric core).
func ParseSemver(s string) (Semver, error) {
	orig := s
	s = strings.TrimPrefix(s, "v")
	s = strings.TrimSpace(s)
	if s == "" {
		return Semver{}, fmt.Errorf("empty version")
	}

	// Split off build metadata first (everything after `+`).
	var build string
	if plus := strings.Index(s, "+"); plus >= 0 {
		build = s[plus+1:]
		s = s[:plus]
	}

	// Split off pre-release (everything after the first `-`).
	var pre string
	if dash := strings.Index(s, "-"); dash >= 0 {
		pre = s[dash+1:]
		s = s[:dash]
	}

	core := strings.Split(s, ".")
	if len(core) != 3 {
		return Semver{}, fmt.Errorf("not MAJOR.MINOR.PATCH: %q", orig)
	}

	major, err := strconv.Atoi(core[0])
	if err != nil {
		return Semver{}, fmt.Errorf("non-numeric major: %q", orig)
	}
	minor, err := strconv.Atoi(core[1])
	if err != nil {
		return Semver{}, fmt.Errorf("non-numeric minor: %q", orig)
	}
	patch, err := strconv.Atoi(core[2])
	if err != nil {
		return Semver{}, fmt.Errorf("non-numeric patch: %q", orig)
	}

	return Semver{
		Major: major, Minor: minor, Patch: patch,
		PreRelease: pre, BuildMetadata: build, Raw: orig,
	}, nil
}

// IsPreRelease reports whether the version carries a pre-release
// identifier (anything after `-`). Build metadata alone doesn't count.
func (v Semver) IsPreRelease() bool {
	return v.PreRelease != ""
}

// gitDescribeSuffixRE matches git-describe's `<N>-g<SHA>[-dirty]`
// post-tag suffix anywhere at the end of a pre-release identifier. The
// N is a non-negative commit count ahead of the nearest tag; SHA is
// 7+ hex characters prefixed with the literal "g"; -dirty is optional.
//
// Matches:
//
//	"4-g356ee75-dirty"         (tag + 4 commits, dirty worktree)
//	"4-g356ee75"               (tag + 4 commits, clean)
//	"rc.1-2-gdeadbeef"         (tag-rc.1 + 2 commits)
//	"rc.1-2-gdeadbeef-dirty"
//
// Does NOT match plain pre-releases like "rc.1", "beta.2", "alpha.12".
var gitDescribeSuffixRE = regexp.MustCompile(`(^|-)\d+-g[0-9a-f]{7,}(-dirty)?$`)

// IsGitDescribeDev reports whether s looks like a locally-built
// development binary whose version string came from
// `git describe --tags --always --dirty` — i.e. a
// `<base-version>-<N>-g<SHA>[-dirty]` shape where N is the count of
// commits past the nearest release tag.
//
// This matters because strict SemVer §11 ranks `M.m.p-<any-pre>` BELOW
// `M.m.p` — but git-describe's `-<N>-g<SHA>` suffix means the build is
// AHEAD of the tag, the opposite semantic. Without special-casing this
// shape, the upgrade flow would happily "upgrade" a dev build back to
// the base tag (an actual downgrade), and the passive notice would
// nag on every command run.
func IsGitDescribeDev(s string) bool {
	v, err := ParseSemver(s)
	if err != nil {
		return false
	}
	if v.PreRelease == "" {
		return false
	}
	return gitDescribeSuffixRE.MatchString(v.PreRelease)
}

// UpgradeAvailable reports whether `latest` represents a real upgrade
// from `current`, taking git-describe dev builds into account.
//
// For non-git-describe versions this is plain `latest.Compare(current) > 0`.
//
// For git-describe dev builds (see [IsGitDescribeDev]), we compare the
// MAJOR.MINOR.PATCH cores only: a dev build `0.5.0-4-g356ee75-dirty`
// is considered at-or-above `v0.5.0` (no spurious upgrade offer), but
// `v0.5.1` or `v0.6.0` still register as upgrades.
func UpgradeAvailable(current, latest Semver) bool {
	if IsGitDescribeDev(current.Raw) {
		curCore := Semver{Major: current.Major, Minor: current.Minor, Patch: current.Patch}
		return latest.Compare(curCore) > 0
	}
	return latest.Compare(current) > 0
}

// Display returns a user-facing version string with any leading "v"
// stripped so versions printed from different sources — ldflags-injected
// build metadata (already stripped by the Taskfile build), GitHub tag
// names (keep the "v"), or cached upgrade-check values — render
// consistently. Safe to call on any string; a value without a leading
// "v" is returned unchanged. Never fails, never parses.
func Display(s string) string {
	return strings.TrimPrefix(s, "v")
}

// Compare returns -1, 0, or 1 for v < other, v == other, v > other.
// Semver ordering rules: a pre-release version has lower precedence than
// a normal one with the same MAJOR.MINOR.PATCH
// (https://semver.org/#spec-item-11). Build metadata is IGNORED for
// precedence per the spec.
func (v Semver) Compare(other Semver) int {
	if c := cmpInt(v.Major, other.Major); c != 0 {
		return c
	}
	if c := cmpInt(v.Minor, other.Minor); c != 0 {
		return c
	}
	if c := cmpInt(v.Patch, other.Patch); c != 0 {
		return c
	}
	// Pre-release comparison: no-pre > has-pre.
	if v.PreRelease == "" && other.PreRelease != "" {
		return 1
	}
	if v.PreRelease != "" && other.PreRelease == "" {
		return -1
	}
	if v.PreRelease == other.PreRelease {
		return 0
	}
	return cmpPreRelease(v.PreRelease, other.PreRelease)
}

func cmpInt(a, b int) int {
	switch {
	case a < b:
		return -1
	case a > b:
		return 1
	default:
		return 0
	}
}

// cmpPreRelease compares two non-empty pre-release strings per
// semver.org §11. Identifiers are split on `.`; numeric identifiers
// compare as integers, alphanumerics as ASCII, numeric < alphanumeric
// when equal-length prefix, shorter prefix loses when equal up to its
// length.
func cmpPreRelease(a, b string) int {
	ap := strings.Split(a, ".")
	bp := strings.Split(b, ".")
	for i := 0; i < len(ap) && i < len(bp); i++ {
		if c := cmpIdent(ap[i], bp[i]); c != 0 {
			return c
		}
	}
	return cmpInt(len(ap), len(bp))
}

func cmpIdent(a, b string) int {
	ai, aIsNum := strconv.Atoi(a)
	bi, bIsNum := strconv.Atoi(b)
	switch {
	case aIsNum == nil && bIsNum == nil:
		return cmpInt(ai, bi)
	case aIsNum == nil && bIsNum != nil:
		return -1 // numeric < alphanumeric
	case aIsNum != nil && bIsNum == nil:
		return 1
	default:
		return strings.Compare(a, b)
	}
}
