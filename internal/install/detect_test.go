// Copyright (c) 2021-2026 Truestamp, Inc.
// SPDX-License-Identifier: MIT

package install

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestMethod_String(t *testing.T) {
	cases := []struct {
		m    Method
		want string
	}{
		{Unknown, "unknown"},
		{Homebrew, "homebrew"},
		{GoInstall, "go install"},
		{InstallScript, "install.sh"},
		{Method(99), "unknown"},
	}
	for _, c := range cases {
		if got := c.m.String(); got != c.want {
			t.Errorf("Method(%d).String() = %q, want %q", c.m, got, c.want)
		}
	}
}

func TestMethod_UpgradeCommand(t *testing.T) {
	cases := []struct {
		m       Method
		wantHas string // expected substring; "" means command should be empty
	}{
		{Homebrew, "brew upgrade"},
		{GoInstall, "go install"},
		{InstallScript, ""},
		{Unknown, ""},
	}
	for _, c := range cases {
		got := c.m.UpgradeCommand()
		if c.wantHas == "" {
			if got != "" {
				t.Errorf("Method(%v).UpgradeCommand() = %q, want empty", c.m, got)
			}
			continue
		}
		if !strings.Contains(got, c.wantHas) {
			t.Errorf("Method(%v).UpgradeCommand() = %q, want substring %q", c.m, got, c.wantHas)
		}
	}
}

func TestIsHomebrewPath(t *testing.T) {
	cases := []struct {
		path string
		want bool
	}{
		// Homebrew paths (post-symlink-resolve).
		{"/opt/homebrew/Cellar/truestamp-cli/0.3.0/bin/truestamp", true},
		{"/opt/homebrew/Caskroom/truestamp-cli/0.3.0/truestamp", true},
		{"/usr/local/Cellar/truestamp-cli/0.3.0/bin/truestamp", true},
		{"/usr/local/Caskroom/truestamp-cli/0.3.0/truestamp", true},
		{"/home/linuxbrew/.linuxbrew/Cellar/truestamp-cli/0.3.0/bin/truestamp", true},
		// Non-homebrew paths — symlink dirs alone shouldn't match.
		{"/usr/local/bin/truestamp", false},
		{"/home/linuxbrew/.linuxbrew/bin/truestamp", false},
		{"/opt/homebrew/bin/truestamp", false},
		{"/home/user/go/bin/truestamp", false},
		{"/tmp/truestamp", false},
	}
	for _, c := range cases {
		if got := isHomebrewPath(c.path); got != c.want {
			t.Errorf("isHomebrewPath(%q) = %v, want %v", c.path, got, c.want)
		}
	}
}

func TestIsInstallScriptPath(t *testing.T) {
	home, _ := os.UserHomeDir()
	// Matches install.sh's choose_install_dir() defaults.
	shouldMatch := []string{
		"/usr/local/bin/truestamp",
		"/usr/bin/truestamp",
	}
	if home != "" {
		shouldMatch = append(shouldMatch, home+"/.local/bin/truestamp")
		shouldMatch = append(shouldMatch, home+"/bin/truestamp")
	}
	for _, p := range shouldMatch {
		if !isInstallScriptPath(p) {
			t.Errorf("isInstallScriptPath(%q) = false, want true", p)
		}
	}

	shouldNotMatch := []string{
		"/opt/homebrew/Cellar/truestamp-cli/0.3.0/bin/truestamp",
		"/tmp/truestamp",
	}
	if home != "" {
		shouldNotMatch = append(shouldNotMatch, home+"/go/bin/truestamp")
	}
	for _, p := range shouldNotMatch {
		if isInstallScriptPath(p) {
			t.Errorf("isInstallScriptPath(%q) = true, want false", p)
		}
	}
}

func TestSameDir(t *testing.T) {
	if !sameDir("/foo/bar", "/foo/bar") {
		t.Error("sameDir should match identical absolute paths")
	}
	if !sameDir("/foo//bar/", "/foo/bar") {
		t.Error("sameDir should collapse double slashes and trailing slash")
	}
	if sameDir("/foo/bar", "/foo/baz") {
		t.Error("sameDir should not match different dirs")
	}
}

// TestSameDir_followsSymlinks is a regression test for the macOS-specific
// bug where $HOME under /tmp (symlink to /private/tmp) wouldn't match an
// executable path that had been resolved via filepath.EvalSymlinks. If
// sameDir stopped resolving symlinks, Homebrew + go-install + install.sh
// detection would all silently fail.
func TestSameDir_followsSymlinks(t *testing.T) {
	realDir := t.TempDir()
	linkDir := filepath.Join(t.TempDir(), "symlink")
	if err := os.Symlink(realDir, linkDir); err != nil {
		t.Skipf("symlink not supported in this environment: %v", err)
	}
	if !sameDir(realDir, linkDir) {
		t.Errorf("sameDir(%q, %q) = false, want true (symlink resolution broken)", realDir, linkDir)
	}
	// Also works the other way.
	if !sameDir(linkDir, realDir) {
		t.Errorf("sameDir(%q, %q) = false, want true (symlink resolution broken in reverse)", linkDir, realDir)
	}
}

func TestExeUnderGoBin(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("GOBIN", "")
	t.Setenv("GOPATH", "")

	defaultGoBin := filepath.Join(home, "go", "bin")
	if err := os.MkdirAll(defaultGoBin, 0755); err != nil {
		t.Fatal(err)
	}

	// Binary at $HOME/go/bin → matches.
	binInGoBin := filepath.Join(defaultGoBin, "truestamp")
	if !exeUnderGoBin(binInGoBin) {
		t.Errorf("exeUnderGoBin(%q) = false, want true (default $HOME/go/bin)", binInGoBin)
	}

	// Binary elsewhere → no match.
	elsewhere := filepath.Join(home, "bin", "truestamp")
	if exeUnderGoBin(elsewhere) {
		t.Errorf("exeUnderGoBin(%q) = true, want false (not in go/bin)", elsewhere)
	}

	// Custom GOBIN wins.
	customGobin := t.TempDir()
	t.Setenv("GOBIN", customGobin)
	binInCustom := filepath.Join(customGobin, "truestamp")
	if !exeUnderGoBin(binInCustom) {
		t.Errorf("exeUnderGoBin(%q) = false, want true (GOBIN=%s)", binInCustom, customGobin)
	}
	// The old $HOME/go/bin still matches (both are checked — no conflict).
	if !exeUnderGoBin(binInGoBin) {
		t.Errorf("exeUnderGoBin(%q) = false even with GOBIN set", binInGoBin)
	}

	// GOPATH-derived bin.
	t.Setenv("GOBIN", "")
	customGopath := t.TempDir()
	t.Setenv("GOPATH", customGopath)
	gopathBin := filepath.Join(customGopath, "bin", "truestamp")
	if !exeUnderGoBin(gopathBin) {
		t.Errorf("exeUnderGoBin(%q) = false, want true (GOPATH=%s)", gopathBin, customGopath)
	}
}

// TestIsGoInstallBinary_fromBuildInfo is a smoke test — when running
// under `go test`, the test binary IS compiled with BuildInfo that
// matches the "go install" heuristic (Main.Version non-empty, no
// vcs.revision embedded for the test binary itself in some cases).
// The exact classification depends on how Go built the test binary, so
// we just assert that the function doesn't panic and returns a bool.
func TestIsGoInstallBinary_fromBuildInfo(t *testing.T) {
	_ = goInstallFromBuildInfo()
}

func TestDirWritable_tempDir(t *testing.T) {
	dir := t.TempDir()
	if !dirWritable(dir) {
		t.Errorf("dirWritable(%q) = false, want true", dir)
	}
	if dirWritable("/does/not/exist/anywhere/at/all") {
		t.Error("dirWritable on nonexistent path should return false")
	}
}

func TestDetect_deterministic(t *testing.T) {
	// Result is sync.Once-cached for the process lifetime.
	first := Detect()
	second := Detect()
	if first != second {
		t.Errorf("Detect() not deterministic: got %v then %v", first, second)
	}
	if Executable() == "" {
		t.Error("Executable() should return a non-empty path for a running test binary")
	}
}
