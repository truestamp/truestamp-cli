// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: MIT

package cmd

import (
	"bufio"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// TestExecutePanicRecovery exercises the deferred recover() in
// Execute() end-to-end. Triggering a panic in-process would require
// catching os.Exit, which the testing runtime doesn't allow, so we
// shell out to the real binary, point it at a temp log file via
// --log-file, run the hidden __test-panic command, and assert:
//
//   - exit code is 2 (matching Go's runtime panic exit code)
//   - stderr starts with "panic: intentional test panic"
//   - stderr contains a goroutine stack trace
//   - the log file contains a `command_end` JSON record with exit=2,
//     panic=<value>, and a non-empty stack attribute
//
// If a future refactor removes the defer, exit code stays 2 (Go's
// default) but the log assertion fails, which is the regression we
// want to catch.
func TestExecutePanicRecovery(t *testing.T) {
	logPath := filepath.Join(t.TempDir(), "panic.log")

	cmd := exec.Command(binaryPath,
		"--log-file", logPath,
		"--no-upgrade-check",
		"__test-panic",
	)
	cmd.Env = append(os.Environ(), "TRUESTAMP_INTERNAL_TEST_PANIC=1")
	stderr, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatalf("expected non-zero exit, got success. output: %s", stderr)
	}
	exitErr, ok := err.(*exec.ExitError)
	if !ok {
		t.Fatalf("expected *exec.ExitError, got %T: %v", err, err)
	}
	if got := exitErr.ExitCode(); got != 2 {
		t.Fatalf("exit code = %d, want 2 (output: %s)", got, stderr)
	}

	// Stderr should look like Go's default panic output.
	stderrStr := string(stderr)
	if !strings.HasPrefix(stderrStr, "panic: intentional test panic") {
		t.Errorf("stderr should start with the panic line, got:\n%s", stderrStr)
	}
	if !strings.Contains(stderrStr, "goroutine ") {
		t.Errorf("stderr should contain a goroutine stack trace, got:\n%s", stderrStr)
	}

	// Log file should contain a command_end record at error level with
	// the panic value and stack captured. Walk lines and look for the
	// matching JSON object, there may be multiple records (command_start
	// fires before the panic).
	data, rerr := os.ReadFile(logPath)
	if rerr != nil {
		t.Fatalf("read log: %v", rerr)
	}
	var endRec map[string]any
	scan := bufio.NewScanner(strings.NewReader(string(data)))
	scan.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for scan.Scan() {
		var rec map[string]any
		if jerr := json.Unmarshal(scan.Bytes(), &rec); jerr != nil {
			continue
		}
		if rec["msg"] == "command_end" {
			endRec = rec
		}
	}
	if endRec == nil {
		t.Fatalf("log missing command_end record. log contents:\n%s", data)
	}
	if endRec["level"] != "ERROR" {
		t.Errorf("command_end level = %v, want ERROR", endRec["level"])
	}
	if exit, _ := endRec["exit"].(float64); exit != 2 {
		t.Errorf("command_end exit = %v, want 2", endRec["exit"])
	}
	if pv, _ := endRec["panic"].(string); pv != "intentional test panic" {
		t.Errorf("command_end panic = %q, want %q", pv, "intentional test panic")
	}
	stackAttr, _ := endRec["stack"].(string)
	if !strings.Contains(stackAttr, "goroutine ") {
		t.Errorf("command_end stack should contain goroutine header, got:\n%s", stackAttr)
	}
}
