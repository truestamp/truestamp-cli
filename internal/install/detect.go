// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: MIT

// Package install detects how the running truestamp binary was installed.
// The detection is best-effort: it inspects the resolved executable path,
// Go build metadata, and environment variables. Callers use the result to
// decide whether `truestamp upgrade` should perform an in-place replace or
// print a package-manager-specific instruction.
package install

import (
	"os"
	"path/filepath"
	"runtime"
	"runtime/debug"
	"strings"
	"sync"
)

// Method is how the truestamp binary was installed.
type Method int

const (
	// Unknown is used when the binary path does not match any known
	// install location and no Go build metadata points to `go install`.
	Unknown Method = iota
	// Homebrew means the binary lives under a Homebrew prefix
	// (/opt/homebrew, /usr/local/Cellar, /home/linuxbrew/.linuxbrew).
	Homebrew
	// GoInstall means the binary was produced by `go install` — detected
	// via Go build metadata and/or path under $GOBIN or $GOPATH/bin.
	GoInstall
	// InstallScript means the binary was placed by install.sh (or a
	// manual tarball extraction) into a standard user-writable path.
	InstallScript
)

// String returns a user-facing label. Zero-value Unknown prints "unknown"
// so version output stays readable.
func (m Method) String() string {
	switch m {
	case Homebrew:
		return "homebrew"
	case GoInstall:
		return "go install"
	case InstallScript:
		return "install.sh"
	default:
		return "unknown"
	}
}

// UpgradeCommand returns the command the user should run to upgrade this
// install method. Empty string for methods the upgrade command performs
// in-place (InstallScript, Unknown-but-writable).
func (m Method) UpgradeCommand() string {
	switch m {
	case Homebrew:
		return "brew upgrade --cask truestamp/tap/truestamp-cli"
	case GoInstall:
		return "go install github.com/truestamp/truestamp-cli/cmd/truestamp@latest"
	default:
		return ""
	}
}

var (
	cachedMethod Method
	cachedExe    string
	detectOnce   sync.Once
)

// Detect classifies the running binary. Result is cached for the lifetime
// of the process.
func Detect() Method {
	detectOnce.Do(func() {
		cachedMethod, cachedExe = detect()
	})
	return cachedMethod
}

// Executable returns the resolved path (symlinks followed) of the running
// binary used during detection. Empty string if detection couldn't resolve.
func Executable() string {
	Detect()
	return cachedExe
}

func detect() (Method, string) {
	exe, err := os.Executable()
	if err != nil {
		return Unknown, ""
	}
	if resolved, err := filepath.EvalSymlinks(exe); err == nil {
		exe = resolved
	}

	// Order matters: Homebrew paths can live under /usr/local on Intel
	// macOS, so we must check the Homebrew-specific directory names
	// first. go-install next (BuildInfo is authoritative). Fall through
	// to install.sh for user-writable standard paths.
	if isHomebrewPath(exe) {
		return Homebrew, exe
	}
	if isGoInstallBinary(exe) {
		return GoInstall, exe
	}
	if isInstallScriptPath(exe) {
		return InstallScript, exe
	}
	return Unknown, exe
}

// homebrewMarkers are substrings in a resolved executable path that
// reliably indicate a Homebrew installation. They are checked AFTER
// EvalSymlinks, so the /usr/local/bin symlink on Intel macOS resolves to
// /usr/local/Cellar/... and matches.
var homebrewMarkers = []string{
	// macOS ARM (default) and Intel Cellar/Caskroom paths.
	"/opt/homebrew/Cellar/",
	"/opt/homebrew/Caskroom/",
	"/usr/local/Cellar/",
	"/usr/local/Caskroom/",
	// Linux brew default prefix.
	"/home/linuxbrew/.linuxbrew/Cellar/",
	"/home/linuxbrew/.linuxbrew/Caskroom/",
}

func isHomebrewPath(exe string) bool {
	// Normalize to forward slashes for substring matching; Homebrew is
	// Unix-only, so Windows paths never match.
	p := filepath.ToSlash(exe)
	for _, marker := range homebrewMarkers {
		if strings.Contains(p, marker) {
			return true
		}
	}
	return false
}

// isGoInstallBinary returns true when the binary was produced by
// `go install`. The heuristic combines two signals:
//
//  1. runtime/debug.BuildInfo reports Main.Version that is NOT "(devel)"
//     AND no VCS revision was embedded — `go install` from the module
//     proxy produces a real version (pseudo-version or tag) and strips
//     VCS settings. A local `go build .` always records VCS when
//     available, so the absence of vcs.revision is a strong signal.
//  2. The executable sits under $GOBIN or $GOPATH/bin.
//
// Either signal alone is sufficient — the Homebrew and install.sh paths
// never match #1, and custom paths still match if the user ran
// `GOBIN=/somewhere go install`.
func isGoInstallBinary(exe string) bool {
	if goInstallFromBuildInfo() {
		return true
	}
	return exeUnderGoBin(exe)
}

func goInstallFromBuildInfo() bool {
	info, ok := debug.ReadBuildInfo()
	if !ok {
		return false
	}
	// A `go install` binary has a Main.Version that is either a real
	// tag (e.g. "v0.3.0") or a pseudo-version
	// ("v0.0.0-20260416173000-abc1234def56"). Local `go build` gives
	// "(devel)".
	if info.Main.Version == "" || info.Main.Version == "(devel)" {
		return false
	}

	// Local `go build .` ALSO produces a non-"(devel)" version when
	// the module is versioned, but it additionally embeds VCS settings
	// (vcs.revision, vcs.modified, vcs.time) whereas `go install` does
	// not. Use their absence as a second signal.
	for _, s := range info.Settings {
		if s.Key == "vcs.revision" && s.Value != "" {
			return false
		}
	}
	return true
}

func exeUnderGoBin(exe string) bool {
	if exe == "" {
		return false
	}
	exeDir := filepath.Dir(exe)
	if gobin := os.Getenv("GOBIN"); gobin != "" {
		if sameDir(exeDir, gobin) {
			return true
		}
	}
	if gopath := os.Getenv("GOPATH"); gopath != "" {
		if sameDir(exeDir, filepath.Join(gopath, "bin")) {
			return true
		}
	}
	// Default GOPATH when unset is $HOME/go on all platforms.
	if home, err := os.UserHomeDir(); err == nil {
		if sameDir(exeDir, filepath.Join(home, "go", "bin")) {
			return true
		}
	}
	return false
}

func sameDir(a, b string) bool {
	ca, errA := filepath.Abs(a)
	cb, errB := filepath.Abs(b)
	if errA != nil || errB != nil {
		return false
	}
	ca = filepath.Clean(ca)
	cb = filepath.Clean(cb)
	if ca == cb {
		return true
	}
	// Resolve symlinks on either side before the second-chance
	// comparison. On macOS, /tmp is a symlink to /private/tmp and
	// /var is a symlink to /private/var, so a $HOME under either one
	// would never match the binary path (which has been EvalSymlinks'd
	// in detect()) without this step. EvalSymlinks fails if the path
	// doesn't exist, so we only call it when both look resolvable.
	if rca, err := filepath.EvalSymlinks(ca); err == nil {
		ca = filepath.Clean(rca)
	}
	if rcb, err := filepath.EvalSymlinks(cb); err == nil {
		cb = filepath.Clean(rcb)
	}
	return ca == cb
}

// isInstallScriptPath returns true for the standard destinations chosen
// by docs/install.sh's choose_install_dir(): /usr/local/bin or
// $HOME/.local/bin. Also accepts /usr/bin for package-manager-adjacent
// manual drops on Linux.
func isInstallScriptPath(exe string) bool {
	exeDir := filepath.Dir(exe)
	candidates := []string{
		"/usr/local/bin",
		"/usr/bin",
	}
	if home, err := os.UserHomeDir(); err == nil {
		candidates = append(candidates, filepath.Join(home, ".local", "bin"))
		candidates = append(candidates, filepath.Join(home, "bin"))
	}
	for _, c := range candidates {
		if sameDir(exeDir, c) {
			return true
		}
	}
	return false
}

// BinaryWritable reports whether the current user can write a new binary
// into the directory containing the running executable. Used to decide
// whether an Unknown install method can be handled by in-place replace.
func BinaryWritable() bool {
	exe := Executable()
	if exe == "" {
		return false
	}
	return dirWritable(filepath.Dir(exe))
}

func dirWritable(dir string) bool {
	// os.Stat + mode check is unreliable on macOS ACLs. Try to create a
	// temp file instead and remove it.
	f, err := os.CreateTemp(dir, ".truestamp-write-test-*")
	if err != nil {
		return false
	}
	name := f.Name()
	_ = f.Close()
	_ = os.Remove(name)
	return true
}

// GOOS returns runtime.GOOS. Exposed as a function so tests can stub it.
var GOOS = func() string { return runtime.GOOS }
