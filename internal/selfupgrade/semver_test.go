// Copyright (c) 2021-2026 Truestamp, Inc.
// SPDX-License-Identifier: MIT

package selfupgrade

import "testing"

func TestParseSemver(t *testing.T) {
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
		if got != c.want {
			t.Errorf("ParseSemver(%q) = %+v, want %+v", c.in, got, c.want)
		}
		if got.IsPreRelease() != c.wantPre {
			t.Errorf("ParseSemver(%q).IsPreRelease() = %v, want %v", c.in, got.IsPreRelease(), c.wantPre)
		}
		if got.BuildMetadata != c.wantBuild {
			t.Errorf("ParseSemver(%q).BuildMetadata = %q, want %q", c.in, got.BuildMetadata, c.wantBuild)
		}
	}
}

func TestSemverCompare(t *testing.T) {
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
