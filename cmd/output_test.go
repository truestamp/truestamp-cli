// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: MIT

package cmd

import (
	"encoding/json"
	"os/exec"
	"strings"
	"testing"

	"github.com/truestamp/truestamp-cli/internal/testfixtures"
)

// escByte is what must never reach a redirected stream. `truestamp ... >
// file` writing terminal escape sequences into that file is the bug this
// file exists to prevent.
const escByte = "\x1b"

// offlineInvocations are real, network-free, credential-free invocations
// that exercise the styled render paths. Every one of them writes through
// a lipgloss style somewhere; a command that only prints plain text would
// make this test vacuous, so the set is deliberately drawn from the ones
// that render cards, tables, banners, and reports.
func offlineInvocations() [][]string {
	bundle := testfixtures.Path(testfixtures.ProdDir, testfixtures.ProdComplete)
	keyring := testfixtures.Path(testfixtures.ProdDir, testfixtures.ProdKeyring)
	return [][]string{
		{"version"},
		{"config", "show"},
		{"config", "path"},
		{"auth", "status"},
		{"verify", bundle, "--offline", "--keyring", keyring},
		{"inspect", bundle},
		{"convert", "time", "2026-09-03T12:00:00Z"},
		{"convert", "keyid", "CTwMqDZnPd/QTLSq8aTeSD3a+j2DQxKcGfhhIYJQ65Y="},
		{"convert", "id", "01JBQ8Z9K2M4N6P8R0T2V4X6Y8"},
		{"hash", bundle},
	}
}

// TestCLI_RedirectedOutput_HasNoANSI runs each invocation with its stdout
// and stderr connected to pipes, i.e. exactly the shape of `truestamp ...
// > file`, and asserts no escape byte survives.
//
// This is the regression that motivated internal/ui's Fprint helpers:
// commands built a styled string with Style.Render and then handed it to
// fmt.Fprintln, which writes straight to the destination and keeps the
// escapes. lipgloss.Println was fine because it goes through a colour
// profile writer; the fmt path was not.
func TestCLI_RedirectedOutput_HasNoANSI(t *testing.T) {
	for _, args := range offlineInvocations() {
		name := strings.Join(args, "_")
		t.Run(name, func(t *testing.T) {
			// CombinedOutput gives pipes for both streams, so neither is
			// a terminal, which is the condition under test.
			out, _ := exec.Command(binaryPath, args...).CombinedOutput()
			if len(out) == 0 {
				t.Fatalf("%v produced no output, so this case asserts nothing", args)
			}
			if i := strings.Index(string(out), escByte); i >= 0 {
				t.Errorf("%v leaked an ANSI escape at byte %d:\n%q", args, i, snippet(string(out), i))
			}
		})
	}
}

// TestCLI_NoColorFlag_HasNoANSI runs the same set with --no-color. The
// flag must be sufficient on its own, independent of terminal detection,
// because that is what users reach for when the detection is wrong.
func TestCLI_NoColorFlag_HasNoANSI(t *testing.T) {
	for _, args := range offlineInvocations() {
		name := strings.Join(args, "_")
		t.Run(name, func(t *testing.T) {
			full := append(append([]string{}, args...), "--no-color")
			out, _ := exec.Command(binaryPath, full...).CombinedOutput()
			if len(out) == 0 {
				t.Fatalf("%v produced no output, so this case asserts nothing", full)
			}
			if i := strings.Index(string(out), escByte); i >= 0 {
				t.Errorf("%v leaked an ANSI escape at byte %d:\n%q", full, i, snippet(string(out), i))
			}
		})
	}
}

// TestCLI_NoColorFlag_IsAcceptedEverywhere guards the flag's reach: it is
// registered as a root persistent flag, so every command must accept it.
// A command that rejects it would fail the test above for the wrong
// reason (empty output), which this distinguishes.
func TestCLI_NoColorFlag_IsAcceptedEverywhere(t *testing.T) {
	for _, args := range offlineInvocations() {
		full := append(append([]string{}, args...), "--no-color")
		out, err := exec.Command(binaryPath, full...).CombinedOutput()
		if err != nil && strings.Contains(string(out), "unknown flag") {
			t.Errorf("%v rejected --no-color: %s", args, out)
		}
	}
}

// snippet returns a short window around index i for a readable failure.
func snippet(s string, i int) string {
	start := max(i-40, 0)
	end := min(i+40, len(s))
	return s[start:end]
}

// TestCLI_JSONFlagIsNeverANoOp: a flag that is advertised, accepted, and
// changes nothing is a flag that lies about what it does.
//
// `schema get` shipped that way — its documents are JSON by nature, so
// --json had nothing to switch, yet it was offered on every invocation.
// The fix was to stop advertising it, not to keep it as a courtesy.
//
// This checks the property rather than the instance: for every offline
// command that offers --json, passing it must change the output.
func TestCLI_JSONFlagIsNeverANoOp(t *testing.T) {
	bundle := testfixtures.Path(testfixtures.ProdDir, testfixtures.ProdComplete)
	for _, args := range [][]string{
		{"version"},
		{"config", "show"},
		{"schema", "list"},
		{"auth", "status"},
		{"inspect", bundle},
		{"hash", bundle},
		{"schema", "get", "witnesses"},
		{"schema", "get", "commands"},
	} {
		t.Run(strings.Join(args, "_"), func(t *testing.T) {
			help, err := exec.Command(binaryPath, append(append([]string{}, args...), "--help")...).CombinedOutput()
			if err != nil {
				t.Fatalf("--help: %v", err)
			}
			if !strings.Contains(string(help), "--json") {
				t.Skipf("%v does not offer --json", args)
			}
			plain, _ := exec.Command(binaryPath, args...).Output()
			asJSON, _ := exec.Command(binaryPath, append(append([]string{}, args...), "--json")...).Output()
			if len(plain) == 0 && len(asJSON) == 0 {
				t.Fatalf("%v produced no output either way", args)
			}
			if string(plain) == string(asJSON) {
				t.Errorf("%v advertises --json but it changes nothing; either give it "+
					"a text rendering or stop offering the flag", args)
			}
		})
	}
}

// TestCLI_SchemaGetRendersTextByDefault: --json is universal across the
// tree, so `schema get` is not an exception to it. It renders a table (or
// the command outline) by default and the document with --json.
//
// This briefly went the other way — the flag was dropped because the
// output was JSON either way and an inert flag is worse than none. The
// better fix was to give the command the text rendering every other
// command has, so the contract stays uniform.
func TestCLI_SchemaGetRendersTextByDefault(t *testing.T) {
	help, err := exec.Command(binaryPath, "schema", "get", "--help").CombinedOutput()
	if err != nil {
		t.Fatalf("schema get --help: %v", err)
	}
	if !strings.Contains(string(help), "--json") {
		t.Error("--json is universal; schema get must offer it like everything else")
	}

	text, err := exec.Command(binaryPath, "schema", "get", "witnesses").Output()
	if err != nil {
		t.Fatalf("schema get witnesses: %v", err)
	}
	if json.Valid(text) {
		t.Errorf("the default rendering should be text, got JSON:\n%s", text)
	}
	if !strings.Contains(string(text), "WITNESS") {
		t.Errorf("the text rendering should be a table, got:\n%s", text)
	}

	asJSON, err := exec.Command(binaryPath, "schema", "get", "witnesses", "--json").Output()
	if err != nil {
		t.Fatalf("schema get witnesses --json: %v", err)
	}
	if !json.Valid(asJSON) {
		t.Errorf("--json should emit JSON, got:\n%s", asJSON)
	}

	// --silent stays: "does this schema exist" has an exit code for an answer.
	if err := exec.Command(binaryPath, "schema", "get", "witnesses", "--silent").Run(); err != nil {
		t.Errorf("schema get --silent should succeed quietly: %v", err)
	}
	if err := exec.Command(binaryPath, "schema", "get", "nope", "--silent").Run(); err == nil {
		t.Error("schema get --silent on an unknown name should exit non-zero")
	}
}
