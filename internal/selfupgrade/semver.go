// Copyright (c) 2021-2026 Truestamp, Inc.
// SPDX-License-Identifier: MIT

package selfupgrade

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"golang.org/x/mod/semver"
)

// Semver is the CLI's release-tag view over `golang.org/x/mod/semver`.
// Parsing, validation, and ordering are delegated to the Go team's
// canonical implementation; this struct keeps typed Major/Minor/Patch
// ints + a Raw copy of the original input around because it reads more
// clearly at call sites than passing bare strings, and because a few
// places (samePatch, UpgradeAvailable's core-only comparison) need
// structural access.
//
// Release tags in this project are strict
// `v<MAJOR>.<MINOR>.<PATCH>[-<PRE>][+<BUILD>]`. Any leading "v" is
// normalized internally (x/mod/semver requires it) but preserved in
// Raw for display.
type Semver struct {
	Major, Minor, Patch int
	PreRelease          string // content after `-`, empty for stable releases
	BuildMetadata       string // content after `+`
	Raw                 string // original input, preserved for display
	canonical           string // "v"-prefixed canonical form, used with semver.Compare
}

// ParseSemver accepts tags like "v0.3.0", "0.3.0", "v1.0.0-rc.1",
// "v1.0.0-beta.2+build.7", and git-describe shapes like
// "0.5.0-4-g356ee75-dirty". Validation and the canonical form come
// from x/mod/semver; the typed MAJOR.MINOR.PATCH fields are derived
// from the canonical string.
func ParseSemver(s string) (Semver, error) {
	orig := s
	s = strings.TrimSpace(s)
	if s == "" {
		return Semver{}, fmt.Errorf("empty version")
	}

	// x/mod/semver requires a leading "v"; normalize for the library
	// while preserving the original string for display in Raw.
	canonical := s
	if !strings.HasPrefix(canonical, "v") {
		canonical = "v" + canonical
	}
	if !semver.IsValid(canonical) {
		return Semver{}, fmt.Errorf("not a valid semver: %q", orig)
	}

	// Derive MAJOR.MINOR.PATCH by trimming the pre-release and build
	// suffixes from the canonical form. x/mod/semver has validated the
	// shape, so the split is guaranteed to yield exactly 3 integers.
	core := canonical
	if i := strings.IndexAny(core, "-+"); i >= 0 {
		core = core[:i]
	}
	parts := strings.Split(strings.TrimPrefix(core, "v"), ".")
	if len(parts) != 3 {
		// Defence in depth: should be unreachable given IsValid above.
		return Semver{}, fmt.Errorf("not MAJOR.MINOR.PATCH: %q", orig)
	}
	major, err := strconv.Atoi(parts[0])
	if err != nil {
		return Semver{}, fmt.Errorf("non-numeric major: %q", orig)
	}
	minor, err := strconv.Atoi(parts[1])
	if err != nil {
		return Semver{}, fmt.Errorf("non-numeric minor: %q", orig)
	}
	patch, err := strconv.Atoi(parts[2])
	if err != nil {
		return Semver{}, fmt.Errorf("non-numeric patch: %q", orig)
	}

	return Semver{
		Major:         major,
		Minor:         minor,
		Patch:         patch,
		PreRelease:    strings.TrimPrefix(semver.Prerelease(canonical), "-"),
		BuildMetadata: strings.TrimPrefix(semver.Build(canonical), "+"),
		Raw:           orig,
		canonical:     canonical,
	}, nil
}

// IsPreRelease reports whether the version carries a pre-release
// identifier (anything after `-`). Build metadata alone doesn't count.
func (v Semver) IsPreRelease() bool {
	return v.PreRelease != ""
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
// Delegates to golang.org/x/mod/semver.Compare, which implements the
// strict SemVer §11 ordering (pre-releases rank below the same core
// tagged release; pre-release identifiers compare numerically when
// both are numeric, lexically otherwise; build metadata is ignored).
//
// For the git-describe-vs-tag asymmetry — SemVer ranks
// `v0.5.0-4-g...-dirty` BELOW `v0.5.0` even though the dev build is
// conceptually ahead — callers should use [UpgradeAvailable] instead,
// which special-cases that shape.
func (v Semver) Compare(other Semver) int {
	return semver.Compare(v.canonical, other.canonical)
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
		curCore, _ := ParseSemver(fmt.Sprintf("v%d.%d.%d", current.Major, current.Minor, current.Patch))
		return latest.Compare(curCore) > 0
	}
	return latest.Compare(current) > 0
}
