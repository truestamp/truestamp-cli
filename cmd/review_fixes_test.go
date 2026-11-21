// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// runCLIEnv is runCLI with extra environment entries after cleanEnv's.
func runCLIEnv(t *testing.T, extra []string, args ...string) (stdout, stderr string, exit int) {
	t.Helper()
	cmd := exec.Command(binaryPath, args...)
	var outBuf, errBuf strings.Builder
	cmd.Stdout, cmd.Stderr = &outBuf, &errBuf
	cmd.Env = append(cleanEnv(), extra...)
	err := cmd.Run()
	if ee, ok := err.(*exec.ExitError); ok {
		exit = ee.ExitCode()
	} else if err != nil {
		t.Fatalf("unexpected exec error: %v", err)
	}
	return outBuf.String(), errBuf.String(), exit
}

// writeTOML writes a config.toml holding body and gives its path.
func writeTOML(t *testing.T, body string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "config.toml")
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

// TestAuthLogout_APIKeyFlagRemovesTheStoredKey pins that `auth logout
// --api-key` removes a key stored in config.toml. Its local boolean
// --api-key shadows the root string flag, and read as a key it made the
// key look explicit, so the command said "nothing is stored to clear" and
// left the key. Mutant killed: config.Load mapping the boolean to api_key.
func TestAuthLogout_APIKeyFlagRemovesTheStoredKey(t *testing.T) {
	cfg := writeTOML(t, `api_key = "sk_stored_test_value"`+"\n")
	stdout, stderr, exit := runCLI(t, "--config", cfg, "auth", "logout", "--api-key")
	if exit != 0 {
		t.Fatalf("exit %d, stderr %q", exit, stderr)
	}
	body, err := os.ReadFile(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(body), "sk_stored_test_value") || strings.Contains(stdout, "nothing is stored") {
		t.Errorf("the stored key was not removed: stdout %q, file %q", stdout, body)
	}
}

// TestOutputFlags_AnExplicitFlagBeatsTheAmbientSetting pins that a --json
// or --silent given on the command line wins over the opposite setting
// from config.toml, as flags beat the file everywhere; both on the command
// line stay an error. Mutant killed: config.Load refusing silent and json
// after merging every source, so `hash --json` failed under silent = true.
func TestOutputFlags_AnExplicitFlagBeatsTheAmbientSetting(t *testing.T) {
	input := filepath.Join(t.TempDir(), "plain.txt")
	if err := os.WriteFile(input, []byte("plain"), 0o644); err != nil {
		t.Fatal(err)
	}
	silentCfg := writeTOML(t, "silent = true\n")
	if stdout, stderr, exit := runCLI(t, "--config", silentCfg, "hash", "--json", input); exit != 0 || !strings.HasPrefix(stdout, "{") {
		t.Errorf("hash --json under silent = true: exit %d, stdout %q, stderr %q", exit, stdout, stderr)
	}
	jsonCfg := writeTOML(t, "json = true\n")
	if stdout, stderr, exit := runCLI(t, "--config", jsonCfg, "version", "--silent"); exit != 0 || stdout != "" {
		t.Errorf("version --silent under json = true: exit %d, stdout %q, stderr %q", exit, stdout, stderr)
	}
	if _, stderr, exit := runCLI(t, "hash", "--json", "--silent", input); exit != 1 || !strings.Contains(stderr, "mutually exclusive") {
		t.Errorf("--json --silent: exit %d, stderr %q; want the mutual-exclusion error", exit, stderr)
	}
}

// TestBeacons_HashOnlyBeatsAnAmbientJSON pins that an explicit --hash-only
// is not refused because config.toml sets json: it named --json, a flag
// the user never passed. Mutant killed: beaconSharedFlags comparing
// --hash-only with the merged output mode.
func TestBeacons_HashOnlyBeatsAnAmbientJSON(t *testing.T) {
	url, _ := countingServer(t)
	cfg := writeTOML(t, "json = true\n")
	_, stderr, _ := runCLI(t, "--config", cfg, "--base-url", url, "--api-key", "k", "beacons", "latest", "--hash-only")
	if strings.Contains(stderr, "mutually exclusive") {
		t.Errorf("--hash-only under json = true was refused: %q", stderr)
	}
}

// TestItemsCreate_SilentPrintsNothing pins R10's --silent on a successful
// create: exit 0 and nothing on stdout. Mutant killed: the success path
// ignoring silent and printing the Item Created card.
func TestItemsCreate_SilentPrintsNothing(t *testing.T) {
	s := startItemsServer(t, func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(itemDoc(testItemID)))
	})
	stdout, stderr, exit := runCLI(t, "--base-url", s.URL, "--api-key", "k",
		"items", "create", "-n", "Silent check", "-d", "A description long enough for the claims rule.", "--silent")
	if exit != 0 || stdout != "" {
		t.Errorf("exit %d, stdout %q, stderr %q; want exit 0 and no output", exit, stdout, stderr)
	}
}

// TestLocalRefusals pins refusals that must come before any request:
// each command exits 1 with its message and the server sees nothing.
// Mutants killed, one per row: --exit-code alone falling through to an
// in-place upgrade; an extra positional argument dropped; a team name given
// twice with one dropped; --type case-folded; an empty --visibility sent;
// --to-team sent unchecked; --committed with --pending refused only after
// the credential check; proofs get with no id printing help with exit 0.
func TestLocalRefusals(t *testing.T) {
	url, requests := countingServer(t)
	const id = "01a1ff38-e0b3-7503-a649-ab8188121763"
	api := []string{"--base-url", url, "--api-key", "k"}
	for _, c := range []struct {
		args []string
		want string
	}{
		{[]string{"upgrade", "--exit-code"}, "--exit-code works only with --check"},
		{[]string{"encode", "a", "b"}, "accepts at most 1 arg"},
		{append(api, "teams", "create", "Alpha", "--name", "Beta"), "give the team name once"},
		{append(api, "proofs", "get", "--type", "BLOCK", id), "--type must be one of"},
		{append(api, "items", "update", id, "--visibility", ""), "--visibility needs a value"},
		{append(api, "items", "update", id, "--to-team", "not a team"), "is no team id"},
		{append(api, "items", "list", "--committed", "--pending"), "--committed and --pending are mutually exclusive"},
		{append(api, "proofs", "get"), "accepts 1 arg"},
	} {
		stdout, stderr, exit := runCLI(t, c.args...)
		if exit != 1 || stdout != "" || !strings.Contains(stderr, c.want) {
			t.Errorf("%v: exit %d, stdout %q, stderr %q; want exit 1 and %q", c.args, exit, stdout, stderr, c.want)
		}
	}
	if n := requests(); n != 0 {
		t.Errorf("%d requests reached the server; every refusal must come first", n)
	}
}

// TestUpgrade_NoVerifyIsRefusedWhenCosignIsRequired pins that --no-verify
// cannot skip the signature check an operator required with
// TRUESTAMP_REQUIRE_COSIGN=1. Mutant killed: SkipCosign set beside
// RequireCosign, so the upgrade installed with no signature check.
func TestUpgrade_NoVerifyIsRefusedWhenCosignIsRequired(t *testing.T) {
	_, stderr, exit := runCLIEnv(t, []string{"TRUESTAMP_REQUIRE_COSIGN=1"}, "upgrade", "--no-verify")
	if exit != 1 || !strings.Contains(stderr, "--no-verify cannot be used while TRUESTAMP_REQUIRE_COSIGN=1") {
		t.Errorf("exit %d, stderr %q; want the refusal", exit, stderr)
	}
}

// TestInspect_NoInputIsAnError pins that inspect with nothing to read exits
// 1 with nothing on stdout, as verify does, under --silent and --json too.
// Mutant killed: the help printed to stdout with exit 0.
func TestInspect_NoInputIsAnError(t *testing.T) {
	for _, args := range [][]string{{"inspect"}, {"inspect", "--silent"}, {"inspect", "--json"}} {
		stdout, _, exit := runCLI(t, args...)
		if exit != 1 || stdout != "" {
			t.Errorf("%v: exit %d, stdout %d octets; want exit 1 and none", args, exit, len(stdout))
		}
	}
}
