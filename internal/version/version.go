// Package version provides build-time version information.
// Variables are set via ldflags at compile time.
package version

import (
	"fmt"
	"runtime"
	"runtime/debug"
	"strconv"
	"time"
)

// Set at build time via -ldflags "-X ...". Defaults are used when
// building without ldflags; init() then tries runtime/debug.ReadBuildInfo
// to populate plausible values (e.g., for `go install` or `go run` users).
var (
	Version   = "dev"
	GitCommit = "unknown"
	BuildDate = "unknown"
	BuildYear = ""

	// Path is the Go module import path for this binary. Defaults to the
	// expected public path; init() overrides from BuildInfo.Main.Path
	// when available (reliable for `go install`-produced binaries).
	Path = "github.com/truestamp/truestamp-cli"
)

func init() {
	// Fill in any field not set by ldflags using runtime/debug build info.
	// This gives `go install ...@version` users proper version/commit/date
	// strings derived from the Go module proxy and embedded VCS metadata.
	if info, ok := debug.ReadBuildInfo(); ok {
		populateFromBuildInfo(info)
	}

	if BuildYear == "" {
		if t, err := time.Parse(time.RFC3339, BuildDate); err == nil {
			BuildYear = strconv.Itoa(t.Year())
		} else {
			BuildYear = strconv.Itoa(time.Now().Year())
		}
	}
}

func populateFromBuildInfo(info *debug.BuildInfo) {
	// Main.Path is the module path from go.mod. Reliable for
	// `go install`-produced binaries; may be "command-line-arguments"
	// for `go run` or local `go build .` — in that case keep the default.
	if info.Main.Path != "" && info.Main.Path != "command-line-arguments" {
		Path = info.Main.Path
	}

	// Main.Version is the module version ("v0.1.0" for tagged installs,
	// pseudo-version like "v0.0.0-20260414181527-897bd82b6abc" otherwise,
	// or "(devel)" for local builds).
	usedInfoVersion := false
	if Version == "dev" && info.Main.Version != "" && info.Main.Version != "(devel)" {
		Version = info.Main.Version
		usedInfoVersion = true
	}

	for _, s := range info.Settings {
		switch s.Key {
		case "vcs.revision":
			if GitCommit == "unknown" && s.Value != "" {
				GitCommit = s.Value
				if len(GitCommit) > 9 {
					GitCommit = GitCommit[:9]
				}
			}
		case "vcs.time":
			if BuildDate == "unknown" && s.Value != "" {
				BuildDate = s.Value
			}
		case "vcs.modified":
			// Mark dirty builds, but only when the version string came from
			// BuildInfo — an explicit ldflags Version is trusted as-is.
			if s.Value == "true" && usedInfoVersion {
				Version += "-dirty"
			}
		}
	}
}

// Full returns the complete version string with commit and build date.
func Full() string {
	return fmt.Sprintf("truestamp %s (commit: %s, built: %s)", Version, GitCommit, BuildDate)
}

// Short returns just the version number.
func Short() string {
	return Version
}

// Copyright returns the copyright notice with dynamic end year.
func Copyright() string {
	return fmt.Sprintf("Truestamp, Inc. — https://truestamp.com\nCopyright (c) 2019-%s Truestamp, Inc. All rights reserved.", BuildYear)
}

// Platform returns the GOOS/GOARCH pair for this binary.
func Platform() string {
	return runtime.GOOS + "/" + runtime.GOARCH
}

// GoVersion returns the Go toolchain version used to build this binary.
func GoVersion() string {
	return runtime.Version()
}

// GoFor returns the Go toolchain version with platform, e.g.
// "go1.26.1 for darwin/arm64".
func GoFor() string {
	return GoVersion() + " for " + Platform()
}
