// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: MIT

package cmd

import (
	"encoding/json"
	"flag"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// Golden-output snapshot tests pin the externally-visible CLI output
// byte-for-byte to files under cmd/testdata/golden/. Run tests as usual
// to assert; regenerate when outputs intentionally change with:
//
//	UPDATE_GOLDEN=1 go test ./cmd/ -run Golden
//
// These guard against accidental wording / formatting / schema drift —
// the same class of regressions that quietly break downstream scripts.

var updateGolden = flag.Bool("update-golden", os.Getenv("UPDATE_GOLDEN") != "",
	"overwrite golden files with current output (also triggered by UPDATE_GOLDEN=1)")

// goldenAssert compares got against the content of testdata/golden/<name>.
// When updateGolden is set, it overwrites the file instead of asserting —
// the idiomatic "accept the new truth" workflow.
func goldenAssert(t *testing.T, name, got string) {
	t.Helper()
	path := filepath.Join("testdata", "golden", name)
	if *updateGolden {
		if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(got), 0644); err != nil {
			t.Fatal(err)
		}
		t.Logf("wrote golden: %s", path)
		return
	}
	want, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading golden %s: %v (regenerate with UPDATE_GOLDEN=1)", path, err)
	}
	if got != string(want) {
		t.Errorf("%s: output drifted from golden. Diff (first 1024 chars):\n--- want ---\n%s\n--- got ---\n%s",
			name, truncate(string(want), 1024), truncate(got, 1024))
	}
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}

// --- Help output: format stability ---------------------------------------

// TestGolden_Help pins the top-level --help output. Changes to the
// command tree or a misplaced flag description will break this test.
func TestGolden_Help(t *testing.T) {
	// Set a stable config path so the --config default doesn't include
	// the runner's $HOME.
	cmd := exec.Command(binaryPath, "--help")
	cmd.Env = append(os.Environ(), "NO_COLOR=1", "HOME=/tmp/golden-home",
		"XDG_CONFIG_HOME=/tmp/golden-config")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("truestamp --help: %v\n%s", err, out)
	}
	goldenAssert(t, "help_root.txt", stableHelp(string(out)))
}

func TestGolden_Help_Verify(t *testing.T) {
	cmd := exec.Command(binaryPath, "verify", "--help")
	cmd.Env = append(os.Environ(), "NO_COLOR=1", "HOME=/tmp/golden-home",
		"XDG_CONFIG_HOME=/tmp/golden-config")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("verify --help: %v", err)
	}
	goldenAssert(t, "help_verify.txt", stableHelp(string(out)))
}

func TestGolden_Help_Hash(t *testing.T) {
	cmd := exec.Command(binaryPath, "hash", "--help")
	cmd.Env = append(os.Environ(), "NO_COLOR=1", "HOME=/tmp/golden-home",
		"XDG_CONFIG_HOME=/tmp/golden-config")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatal(err)
	}
	goldenAssert(t, "help_hash.txt", stableHelp(string(out)))
}

// TestGolden_HashList pins the "truestamp hash --list" output — this
// is the canonical list of supported hash algorithms. If this drifts
// without an intentional algorithm-table change, a lot of user scripts
// break silently.
func TestGolden_HashList(t *testing.T) {
	cmd := exec.Command(binaryPath, "hash", "--list")
	cmd.Env = append(os.Environ(), "NO_COLOR=1")
	out, err := cmd.Output()
	if err != nil {
		t.Fatal(err)
	}
	goldenAssert(t, "hash_list.txt", string(out))
}

// --- JSON output: schema stability ---------------------------------------

// TestGolden_HashJSON pins the `truestamp hash --json` envelope shape.
// Downstream scripts rely on these field names.
func TestGolden_HashJSON(t *testing.T) {
	cmd := exec.Command(binaryPath, "hash", "-a", "sha256", "--json", "--no-filename")
	cmd.Stdin = strings.NewReader("abc")
	out, err := cmd.Output()
	if err != nil {
		t.Fatal(err)
	}
	// Re-serialize with sorted keys so the test ignores Go-map iteration
	// order (which is deliberately unstable in Go).
	goldenAssert(t, "hash_json.json", canonicalJSON(t, out))
}

func TestGolden_EncodeJSON(t *testing.T) {
	cmd := exec.Command(binaryPath, "encode", "--to", "hex", "--json")
	cmd.Stdin = strings.NewReader("hello")
	out, err := cmd.Output()
	if err != nil {
		t.Fatal(err)
	}
	goldenAssert(t, "encode_json.json", canonicalJSON(t, out))
}

// TestGolden_ConvertTimeJSON pins the `convert time` JSON shape.
func TestGolden_ConvertTimeJSON(t *testing.T) {
	cmd := exec.Command(binaryPath, "convert", "time", "1700000000",
		"--to-zone", "UTC", "--json")
	out, err := cmd.Output()
	if err != nil {
		t.Fatal(err)
	}
	goldenAssert(t, "convert_time_json.json", canonicalJSON(t, out))
}

// TestGolden_ConvertIDJSON pins the ULID-extraction JSON envelope.
func TestGolden_ConvertIDJSON(t *testing.T) {
	cmd := exec.Command(binaryPath, "convert", "id",
		"01HJHB01T8FYZ7YTR9P5N62K5B", "--to-zone", "UTC", "--json")
	out, err := cmd.Output()
	if err != nil {
		t.Fatal(err)
	}
	goldenAssert(t, "convert_id_json.json", canonicalJSON(t, out))
}

// TestGolden_ConvertKeyIDJSON pins the kid-derivation JSON envelope.
func TestGolden_ConvertKeyIDJSON(t *testing.T) {
	cmd := exec.Command(binaryPath, "convert", "keyid",
		"CTwMqDZnPd/QTLSq8aTeSD3a+j2DQxKcGfhhIYJQ65Y=", "--json")
	out, err := cmd.Output()
	if err != nil {
		t.Fatal(err)
	}
	goldenAssert(t, "convert_keyid_json.json", canonicalJSON(t, out))
}

// canonicalJSON re-serializes JSON with sorted keys and 2-space indent so
// the golden fixture stays stable across map-iteration-order changes.
func canonicalJSON(t *testing.T, raw []byte) string {
	t.Helper()
	var v any
	if err := json.Unmarshal(raw, &v); err != nil {
		t.Fatalf("not valid JSON: %v\n%s", err, raw)
	}
	// encoding/json's Marshal sorts object keys alphabetically for
	// map[string]any values — exactly what we want here.
	out, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	return string(out) + "\n"
}

// stableHelp strips runner-specific content from --help output so the
// golden fixture doesn't pick up $HOME or absolute paths. Cobra prints
// the default --config path as `(default: ...)`; we mask it.
func stableHelp(s string) string {
	// Normalize the --config default line, which includes $HOME.
	lines := strings.Split(s, "\n")
	for i, line := range lines {
		if strings.Contains(line, "default:") && strings.Contains(line, "/config.toml") {
			// Keep the flag text up to "default:", replace the path.
			idx := strings.Index(line, "(default:")
			if idx >= 0 {
				lines[i] = line[:idx] + "(default: <home>/config.toml)"
			}
		}
	}
	return strings.Join(lines, "\n")
}
