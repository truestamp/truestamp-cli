// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"bytes"
	"encoding/json/v2"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/pflag"
	"github.com/truestamp/truestamp-cli/internal/config"
	"github.com/truestamp/truestamp-cli/internal/logging"
)

// runSplit runs the binary with args under the isolated environment and
// returns stdout, stderr and the error.
func runSplit(t *testing.T, args ...string) (string, string, error) {
	t.Helper()
	c := exec.Command(binaryPath, args...)
	c.Env = cleanEnv()
	var stdout, stderr bytes.Buffer
	c.Stdout, c.Stderr = &stdout, &stderr
	err := c.Run()
	return stdout.String(), stderr.String(), err
}

// A configuration file key no setting reads is said on stderr, once per
// key, while the command still succeeds and stdout stays the record; a
// silent run says nothing. Mutants killed: the warning loop removed from
// PersistentPreRunE; the warning printed under --silent.
func TestCLI_UnknownConfigKeyWarnsOnStderr(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.toml")
	if err := os.WriteFile(path, []byte("bogus_key = 1\n[verify]\nskip_external = true\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	stdout, stderr, err := runSplit(t, "config", "show", "--json", "--config", path)
	if err != nil {
		t.Fatalf("config show: %v\n%s", err, stderr)
	}
	var doc map[string]any
	if err := json.Unmarshal([]byte(stdout), &doc); err != nil {
		t.Fatalf("stdout is not JSON: %v\n%s", err, stdout)
	}
	for _, key := range []string{"bogus_key", "verify.skip_external"} {
		want := "warning: " + path + `: unknown config key "` + key + `" has no effect`
		if strings.Count(stderr, want) != 1 {
			t.Errorf("stderr lacks %q once:\n%s", want, stderr)
		}
	}

	stdout, stderr, err = runSplit(t, "hash", "--silent", "--config", path, "--list")
	if err != nil || stdout != "" || stderr != "" {
		t.Errorf("silent run: %v, stdout %q, stderr %q", err, stdout, stderr)
	}
}

// A base_url of plain http to a host that is not loopback is refused at
// load, exit 1, before any request: the API key would cross the network
// in cleartext. No network is needed, since Load refuses first.
func TestCLI_PlainHTTPBaseURLIsRefused(t *testing.T) {
	_, stderr, err := runSplit(t, "auth", "status", "--base-url", "http://example.com")
	if ee, ok := err.(*exec.ExitError); !ok || ee.ExitCode() != 1 {
		t.Fatalf("exit: %v, want 1\n%s", err, stderr)
	}
	if !strings.Contains(stderr, "uses http:// to a non-loopback host; credentials would be sent unencrypted, use https://") {
		t.Errorf("stderr:\n%s", stderr)
	}
	_, stderr, err = runSplit(t, "auth", "status", "--base-url", "ws://localhost:4000")
	if err == nil || !strings.Contains(stderr, "scheme must be https (or http for a loopback host)") {
		t.Errorf("ws:// base_url: %v\n%s", err, stderr)
	}
}

// console --ws-url carries the credential as base_url does, so a ws://
// override to a host that is not loopback is refused, exit 1, before the
// console dials or reads a terminal. Mutant killed: the override used as
// given.
func TestCLI_ConsolePlainWSURLIsRefused(t *testing.T) {
	_, stderr, err := runSplit(t, "console", "--api-key", "k", "--ws-url", "ws://example.com/console/websocket")
	if ee, ok := err.(*exec.ExitError); !ok || ee.ExitCode() != 1 {
		t.Fatalf("exit: %v, want 1\n%s", err, stderr)
	}
	if !strings.Contains(stderr, "uses ws:// to a non-loopback host; credentials would be sent unencrypted, use wss://") {
		t.Errorf("stderr:\n%s", stderr)
	}
}

// An invalid --log-level is an error, exit 1, like a bad --http-timeout,
// instead of a quiet fall back to info; a valid one runs.
func TestCLI_InvalidLogLevelIsRefused(t *testing.T) {
	_, stderr, err := runSplit(t, "--log-level", "bogus", "version")
	if ee, ok := err.(*exec.ExitError); !ok || ee.ExitCode() != 1 {
		t.Fatalf("exit: %v, want 1\n%s", err, stderr)
	}
	if !strings.Contains(stderr, `invalid logging.level "bogus"`) {
		t.Errorf("stderr:\n%s", stderr)
	}
	if _, stderr, err := runSplit(t, "--log-level", "debug", "--log-file", filepath.Join(t.TempDir(), "t.log"), "version"); err != nil {
		t.Errorf("--log-level debug: %v\n%s", err, stderr)
	}
}

// config.Load accepts exactly the log levels internal/logging knows (it
// keeps its own copy of the list, since internal/verifier imports config
// and must not import logging), and every value the schema publishes for
// --log-level is one of them.
func TestCLI_LogLevelAgreesAcrossConfigLoggingAndSchema(t *testing.T) {
	values := cliEnums()["log-level"]
	if len(values) == 0 {
		t.Fatal("no log-level enum")
	}
	for _, v := range values {
		if !logging.ValidLevel(v) {
			t.Errorf("schema log-level %q is refused by logging.ValidLevel", v)
		}
	}
	none := filepath.Join(t.TempDir(), "none.toml")
	for _, v := range append(values, "", "INFO", " warn ", "warning", "Error", "bogus", "trace", "verbose", "infoo") {
		flags := pflag.NewFlagSet("root", pflag.ContinueOnError)
		flags.String("log-level", "", "")
		if err := flags.Parse([]string{"--log-level", v}); err != nil {
			t.Fatal(err)
		}
		_, err := config.Load(none, flags)
		if (err == nil) != logging.ValidLevel(v) {
			t.Errorf("level %q: config.Load error %v, logging.ValidLevel %v", v, err, logging.ValidLevel(v))
		}
	}
}
