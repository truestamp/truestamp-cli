// Copyright (c) 2021-2026 Truestamp, Inc.
// SPDX-License-Identifier: MIT

package cmd

import (
	"runtime"
	"strings"
	"testing"

	"github.com/truestamp/truestamp-cli/internal/install"
)

func TestUpgradeInstructionFor(t *testing.T) {
	cases := []struct {
		name    string
		method  install.Method
		wantCmd string // substring expected in the returned instruction
		wantOk  bool   // true = print instruction; false = proceed with in-place upgrade
		skipOn  string // set to "windows" if the case only makes sense off-Windows
	}{
		{
			name:    "homebrew prints brew upgrade",
			method:  install.Homebrew,
			wantCmd: "brew upgrade",
			wantOk:  true,
			skipOn:  "windows", // Homebrew doesn't run on Windows; the OS check short-circuits
		},
		{
			name:    "go install prints go install @latest",
			method:  install.GoInstall,
			wantCmd: "go install",
			wantOk:  true,
		},
		{
			name:    "install.sh returns falls-through (in-place)",
			method:  install.InstallScript,
			wantCmd: "",
			wantOk:  false,
			skipOn:  "windows", // Windows always prints go-install regardless
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if c.skipOn == runtime.GOOS {
				t.Skipf("skipping on %s", runtime.GOOS)
			}
			cmdStr, ok := upgradeInstructionFor(c.method)
			if ok != c.wantOk {
				t.Errorf("upgradeInstructionFor(%v).ok = %v, want %v", c.method, ok, c.wantOk)
			}
			if c.wantCmd == "" {
				if cmdStr != "" {
					t.Errorf("upgradeInstructionFor(%v).cmd = %q, want empty", c.method, cmdStr)
				}
				return
			}
			if !strings.Contains(cmdStr, c.wantCmd) {
				t.Errorf("upgradeInstructionFor(%v).cmd = %q, want substring %q", c.method, cmdStr, c.wantCmd)
			}
		})
	}

	// Windows-specific: always returns `go install ...@latest` regardless
	// of detected method, because install.sh doesn't support Windows and
	// Homebrew doesn't exist there. We can't change runtime.GOOS, so
	// this test only runs when the test host IS Windows.
	if runtime.GOOS == "windows" {
		for _, m := range []install.Method{install.Homebrew, install.GoInstall, install.InstallScript, install.Unknown} {
			cmdStr, ok := upgradeInstructionFor(m)
			if !ok || !strings.Contains(cmdStr, "go install") {
				t.Errorf("on Windows: upgradeInstructionFor(%v) = (%q, %v), want (contains 'go install', true)", m, cmdStr, ok)
			}
		}
	}
}

func TestReadYes(t *testing.T) {
	cases := []struct {
		in   string
		want bool
	}{
		{"", true},   // EOF = default yes
		{"\n", true}, // empty line = default yes
		{"y\n", true},
		{"Y\n", true},
		{"yes\n", true},
		{"YES\n", true},
		{"  y  \n", true}, // whitespace-tolerant
		{"n\n", false},
		{"N\n", false},
		{"no\n", false},
		{"nope\n", false},
		{"maybe\n", false},
		{"1\n", false}, // only explicit "y"/"yes" is yes
	}
	for _, c := range cases {
		got := readYes(strings.NewReader(c.in))
		if got != c.want {
			t.Errorf("readYes(%q) = %v, want %v", c.in, got, c.want)
		}
	}
}

func TestExitCode(t *testing.T) {
	if ExitCode(nil) != 0 {
		t.Error("ExitCode(nil) should be 0")
	}
	if ExitCode(exitCodeErr{code: 3}) != 3 {
		t.Error("ExitCode should unwrap exitCodeErr.code")
	}
	if ExitCode(exitCodeErr{code: 0}) != 0 {
		t.Error("ExitCode of zero-code exitCodeErr should be 0")
	}
	// Generic errors fall back to 1.
	if ExitCode(errForTest("boom")) != 1 {
		t.Error("ExitCode for generic error should be 1")
	}
}

// errForTest is a tiny local error type to avoid importing errors for a
// one-line stdErr stand-in.
type errForTest string

func (e errForTest) Error() string { return string(e) }
