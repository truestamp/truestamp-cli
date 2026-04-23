// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: MIT

package selfupgrade

import "testing"

func TestParseSemver(t *testing.T) {
	t.Parallel()
	cases := []struct {
		in        string
		want      Semver
		wantErr   bool
		wantPre   bool
		wantBuild string
	}{
		{in: "v0.3.0", want: Semver{Major: 0, Minor: 3, Patch: 0, Raw: "v0.3.0"}},
		{in: "0.3.0", want: Semver{Major: 0, Minor: 3, Patch: 0, Raw: "0.3.0"}},
		{in: "v1.0.0-rc.1", want: Semver{Major: 1, Minor: 0, Patch: 0, PreRelease: "rc.1", Raw: "v1.0.0-rc.1"}, wantPre: true},
		{in: "v1.0.0-alpha", want: Semver{Major: 1, Minor: 0, Patch: 0, PreRelease: "alpha", Raw: "v1.0.0-alpha"}, wantPre: true},
		{in: "v1.0.0-beta.2+build.7", want: Semver{Major: 1, Minor: 0, Patch: 0, PreRelease: "beta.2", BuildMetadata: "build.7", Raw: "v1.0.0-beta.2+build.7"}, wantPre: true, wantBuild: "build.7"},
		{in: "v10.20.30", want: Semver{Major: 10, Minor: 20, Patch: 30, Raw: "v10.20.30"}},
		{in: "", wantErr: true},
		{in: "v1.0", wantErr: true},
		{in: "v1.0.0.0", wantErr: true},
		{in: "va.b.c", wantErr: true},
	}
	for _, c := range cases {
		got, err := ParseSemver(c.in)
		if c.wantErr {
			if err == nil {
				t.Errorf("ParseSemver(%q) succeeded, want error", c.in)
			}
			continue
		}
		if err != nil {
			t.Errorf("ParseSemver(%q) unexpected error: %v", c.in, err)
			continue
		}
		// Compare public fields one-by-one — struct equality would trip on
		// the unexported `canonical` field populated from x/mod/semver.
		if got.Major != c.want.Major || got.Minor != c.want.Minor || got.Patch != c.want.Patch ||
			got.PreRelease != c.want.PreRelease || got.BuildMetadata != c.want.BuildMetadata ||
			got.Raw != c.want.Raw {
			t.Errorf("ParseSemver(%q) public fields = {M:%d m:%d p:%d Pre:%q Build:%q Raw:%q}, want %+v",
				c.in, got.Major, got.Minor, got.Patch, got.PreRelease, got.BuildMetadata, got.Raw, c.want)
		}
		if got.IsPreRelease() != c.wantPre {
			t.Errorf("ParseSemver(%q).IsPreRelease() = %v, want %v", c.in, got.IsPreRelease(), c.wantPre)
		}
		if got.BuildMetadata != c.wantBuild {
			t.Errorf("ParseSemver(%q).BuildMetadata = %q, want %q", c.in, got.BuildMetadata, c.wantBuild)
		}
	}
}

func TestDisplay(t *testing.T) {
	t.Parallel()
	cases := []struct {
		in, want string
	}{
		{"v0.3.2", "0.3.2"},
		{"0.3.2", "0.3.2"}, // already stripped
		{"v1.0.0-rc.1", "1.0.0-rc.1"},
		{"1.0.0-rc.1", "1.0.0-rc.1"},
		{"", ""},
		{"v", ""},
		{"vdev", "dev"}, // non-semver strings still get the prefix stripped (best-effort)
		{"dev", "dev"},
		{"0.3.0-5-g12d62b0-dirty", "0.3.0-5-g12d62b0-dirty"}, // git describe output
	}
	for _, c := range cases {
		if got := Display(c.in); got != c.want {
			t.Errorf("Display(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestSemverCompare(t *testing.T) {
	t.Parallel()
	cases := []struct {
		a, b string
		want int
	}{
		{"v0.3.0", "v0.3.0", 0},
		{"v0.3.0", "v0.3.1", -1},
		{"v0.3.1", "v0.3.0", 1},
		{"v0.3.0", "v0.4.0", -1},
		{"v1.0.0", "v0.99.99", 1},
		// Pre-release < release at same core.
		{"v1.0.0-rc.1", "v1.0.0", -1},
		{"v1.0.0", "v1.0.0-rc.1", 1},
		{"v1.0.0-alpha", "v1.0.0-beta", -1},
		{"v1.0.0-rc.1", "v1.0.0-rc.2", -1},
		{"v1.0.0-rc.2", "v1.0.0-rc.10", -1}, // numeric compare, not string
		// Build metadata is ignored for precedence.
		{"v1.0.0+build.1", "v1.0.0+build.99", 0},
		// Numeric < alphanumeric at same position.
		{"v1.0.0-1", "v1.0.0-alpha", -1},
	}
	for _, c := range cases {
		va, errA := ParseSemver(c.a)
		vb, errB := ParseSemver(c.b)
		if errA != nil || errB != nil {
			t.Fatalf("setup: ParseSemver(%q) err=%v, ParseSemver(%q) err=%v", c.a, errA, c.b, errB)
		}
		got := va.Compare(vb)
		if got != c.want {
			t.Errorf("Compare(%q, %q) = %d, want %d", c.a, c.b, got, c.want)
		}
	}
}

func TestIsGitDescribeDev(t *testing.T) {
	t.Parallel()
	cases := []struct {
		in   string
		want bool
	}{
		// git-describe outputs — should match
		{"0.5.0-4-g356ee75-dirty", true},
		{"0.5.0-4-g356ee75", true},
		{"v0.5.0-4-g356ee75-dirty", true},
		{"1.2.3-99-g0000000", true},
		{"0.5.0-rc.1-2-gdeadbeef", true}, // dev build ahead of pre-release tag
		{"0.5.0-rc.1-2-gdeadbeef-dirty", true},
		// Not git-describe — should NOT match
		{"0.5.0", false},
		{"v0.5.0", false},
		{"0.5.0-rc.1", false},
		{"0.5.0-beta.2", false},
		{"0.5.0-alpha", false},
		{"0.5.0+build.1", false}, // build metadata only, no pre-release
		// Hex too short to be a git short-sha (regex requires 7+)
		{"0.5.0-4-gabc", false},
		// Leading "g" without the digit-prefix
		{"0.5.0-gabc1234567", false},
		// Garbage
		{"not-a-version", false},
		{"", false},
	}
	for _, c := range cases {
		if got := IsGitDescribeDev(c.in); got != c.want {
			t.Errorf("IsGitDescribeDev(%q) = %v, want %v", c.in, got, c.want)
		}
	}
}

func TestUpgradeAvailable(t *testing.T) {
	t.Parallel()
	cases := []struct {
		current, latest string
		want            bool
		note            string
	}{
		// Normal (non-git-describe) behavior — strict semver.
		{"v0.5.0", "v0.5.1", true, "patch bump"},
		{"v0.5.0", "v0.5.0", false, "same version"},
		{"v0.5.1", "v0.5.0", false, "latest is older"},
		{"v1.0.0-rc.1", "v1.0.0", true, "rc → final is an upgrade"},
		// Git-describe dev builds — compare cores only.
		{"0.5.0-4-g356ee75-dirty", "v0.5.0", false,
			"dev build 4 commits past v0.5.0 should NOT offer v0.5.0 as an upgrade (it's a downgrade)"},
		{"0.5.0-4-g356ee75", "v0.5.0", false, "clean git-describe, same outcome"},
		{"0.5.0-4-g356ee75-dirty", "v0.5.1", true,
			"dev build past v0.5.0 SHOULD offer v0.5.1 — real upgrade"},
		{"0.5.0-4-g356ee75-dirty", "v0.6.0", true, "minor bump past dev-build's base"},
		{"0.5.0-4-g356ee75-dirty", "v1.0.0", true, "major bump past dev-build's base"},
		{"0.5.0-4-g356ee75-dirty", "v0.4.9", false, "latest is behind dev-build's base"},
	}
	for _, c := range cases {
		cur, errA := ParseSemver(c.current)
		lat, errB := ParseSemver(c.latest)
		if errA != nil || errB != nil {
			t.Fatalf("setup %q/%q: %v/%v", c.current, c.latest, errA, errB)
		}
		if got := UpgradeAvailable(cur, lat); got != c.want {
			t.Errorf("UpgradeAvailable(%q → %q) = %v, want %v  // %s",
				c.current, c.latest, got, c.want, c.note)
		}
	}
}
