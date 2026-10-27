// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/truestamp/truestamp-cli/internal/strictjson"
	"github.com/truestamp/truestamp-cli/internal/testfixtures"
	"github.com/truestamp/truestamp-cli/internal/vectorcases"
	"github.com/truestamp/truestamp-cli/internal/verifier"
)

// These tests build the actual binary and run it as a subprocess to verify
// exit codes and output. This is the idiomatic Go approach for testing CLI
// tools that call os.Exit. The binary is built with the verifier's two
// compiled-in addresses, Truestamp's keyring and NIST's Randomness Beacon,
// set at link time to deadEndpoints, so no test of this package can read
// either (verify_online_test.go).

var (
	binaryPath         string
	subprocessCoverDir string // set during TestMain when coverage is requested

	// runDir is this run's temporary directory, made by runTests and
	// removed before TestMain exits. cleanEnv points HOME and
	// XDG_CONFIG_HOME at paths under it that do not exist.
	runDir string
)

// deadBaseURL is the API origin every subprocess of this package
// inherits: a loopback port nothing listens on, so a test that forgets
// --base-url gets a refused connection, and the credential store is keyed
// by an origin no stored session uses.
const deadBaseURL = "http://127.0.0.1:9"

func TestMain(m *testing.M) {
	os.Exit(runTests(m))
}

// runTests builds the binary into a temporary directory, runs the tests
// and removes the directory before TestMain exits: os.Exit runs no
// deferred call, so the removal is deferred here instead.
func runTests(m *testing.M) int {
	tmp, err := os.MkdirTemp("", "truestamp-test-*")
	if err != nil {
		panic(err)
	}
	defer os.RemoveAll(tmp)
	runDir = tmp

	binaryPath = filepath.Join(tmp, "truestamp")

	modRoot, err := findModuleRoot()
	if err != nil {
		panic("cannot find module root: " + err.Error())
	}

	// When TRUESTAMP_COVERDIR is set (task test-coverage-full sets it to
	// coverage/bin), build the subprocess binary with -cover so its runtime
	// coverage is recorded there.
	// -buildvcs=false: the binary reports version "dev" in a checkout as
	// in a copy without VCS metadata, so the once-a-day upgrade notice,
	// which is off for "dev", never reaches api.github.com from a test.
	buildArgs := []string{"build", "-buildvcs=false", "-ldflags", verifierEndpointFlags(deadEndpoints)}
	subprocessCoverDir = os.Getenv("TRUESTAMP_COVERDIR")
	if subprocessCoverDir != "" {
		buildArgs = append(buildArgs, "-cover", "-coverpkg=./...")
		_ = os.Setenv("GOCOVERDIR", subprocessCoverDir)
	}
	buildArgs = append(buildArgs, "-o", binaryPath, "./cmd/truestamp")
	cmd := exec.Command("go", buildArgs...)
	cmd.Dir = modRoot
	if out, err := cmd.CombinedOutput(); err != nil {
		panic("failed to build binary: " + err.Error() + "\n" + string(out))
	}

	if err := isolatePackage(tmp); err != nil {
		panic("isolating the test environment: " + err.Error())
	}
	return m.Run()
}

// isolatePackage gives this test process, and so every subprocess it
// starts without an Env of its own, a HOME and XDG directories under dir,
// no TRUESTAMP_* variable but the upgrade-check opt-out, and deadBaseURL
// as TRUESTAMP_BASE_URL. No test of this package then reads the
// developer's configuration, OAuth session or keychain entry for a real
// origin. The go command's caches and settings file are pinned first,
// from the inherited HOME, so a test that runs `go build` still finds
// them.
func isolatePackage(dir string) error {
	goVars := []string{"GOCACHE", "GOMODCACHE", "GOPATH", "GOENV"}
	out, err := exec.Command("go", append([]string{"env"}, goVars...)...).Output()
	if err != nil {
		return fmt.Errorf("go env: %w", err)
	}
	values := strings.Split(strings.TrimRight(string(out), "\n"), "\n")
	if len(values) != len(goVars) {
		return fmt.Errorf("go env printed %d lines for %d variables", len(values), len(goVars))
	}
	for i, name := range goVars {
		if os.Getenv(name) == "" {
			if err := os.Setenv(name, values[i]); err != nil {
				return err
			}
		}
	}
	for _, e := range os.Environ() {
		if name, _, _ := strings.Cut(e, "="); strings.HasPrefix(name, "TRUESTAMP_") {
			if err := os.Unsetenv(name); err != nil {
				return err
			}
		}
	}
	home := filepath.Join(dir, "home")
	vars := map[string]string{
		"HOME":                       home,
		"XDG_CONFIG_HOME":            filepath.Join(home, "config"),
		"XDG_CACHE_HOME":             filepath.Join(home, "cache"),
		"XDG_DATA_HOME":              filepath.Join(home, "data"),
		"XDG_STATE_HOME":             filepath.Join(home, "state"),
		"TRUESTAMP_NO_UPGRADE_CHECK": "1",
		"TRUESTAMP_BASE_URL":         deadBaseURL,
	}
	for name, v := range vars {
		if strings.HasPrefix(name, "XDG_") || name == "HOME" {
			if err := os.MkdirAll(v, 0o700); err != nil {
				return err
			}
		}
		if err := os.Setenv(name, v); err != nil {
			return err
		}
	}
	return nil
}

// TestPackageEnvironment_IsIsolated: a subprocess started with the
// inherited environment, and one started with cleanEnv, both resolve
// deadBaseURL and a configuration file under this run's runDir, never
// the developer's HOME or a fixed shared path.
func TestPackageEnvironment_IsIsolated(t *testing.T) {
	inherited, err := exec.Command(binaryPath, "config", "show", "--json").Output()
	if err != nil {
		t.Fatalf("config show --json with the inherited environment: %v\n%s", err, inherited)
	}
	clean, stderr, code := runCLI(t, "config", "show", "--json")
	if code != 0 {
		t.Fatalf("config show --json with cleanEnv: exit %d\n%s", code, stderr)
	}
	for name, out := range map[string][]byte{"inherited": inherited, "cleanEnv": []byte(clean)} {
		var rec struct {
			ConfigFile string `json:"config_file"`
			BaseURL    string `json:"base_url"`
		}
		if err := json.Unmarshal(out, &rec); err != nil {
			t.Fatalf("%s: config show --json: %v\n%s", name, err, out)
		}
		if rec.BaseURL != deadBaseURL {
			t.Errorf("%s: base_url = %q, want %q", name, rec.BaseURL, deadBaseURL)
		}
		if !strings.HasPrefix(rec.ConfigFile, runDir+string(filepath.Separator)) {
			t.Errorf("%s: config_file = %q, want a path under %q", name, rec.ConfigFile, runDir)
		}
	}
}

// findModuleRoot walks up from the current working directory to find go.mod.
func findModuleRoot() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", os.ErrNotExist
		}
		dir = parent
	}
}

func containsString(s, substr string) bool { return strings.Contains(s, substr) }

// stripANSI removes ANSI escape sequences from output.
func stripANSI(s string) string {
	out := make([]byte, 0, len(s))
	skip := false
	for i := 0; i < len(s); i++ {
		b := s[i]
		if b == 0x1b {
			skip = true
			continue
		}
		if skip {
			if b == 'm' {
				skip = false
			}
			continue
		}
		out = append(out, b)
	}
	return string(out)
}

// Fixture paths under the repository's shared testdata/, read by the
// commands that still parse the earlier bundle layout (inspect, and the
// bundle verify --remote posts).
func prodPath(name string) string   { return testfixtures.Path(testfixtures.ProdDir, name) }
func tamperPath(name string) string { return testfixtures.Path(testfixtures.TamperDir, name) }

// runCLIText runs the binary and returns stdout+stderr (ANSI stripped) and
// the exit code.
func runCLIText(t *testing.T, args ...string) (string, int) {
	t.Helper()
	cmd := exec.Command(binaryPath, args...)
	cmd.Env = cleanEnv()
	out, err := cmd.CombinedOutput()
	code := 0
	if err != nil {
		var exitErr *exec.ExitError
		if !errors.As(err, &exitErr) {
			t.Fatalf("running: %v", err)
		}
		code = exitErr.ExitCode()
	}
	return stripANSI(string(out)), code
}

// --- the vectors these tests run ---

// vectorCase is the case of the shared vectors named id.
func vectorCase(t *testing.T, id string) vectorcases.Case {
	t.Helper()
	cases, err := vectorcases.Load()
	if err != nil {
		t.Fatal(err)
	}
	for _, c := range cases {
		if c.ID == id {
			return c
		}
	}
	t.Fatalf("no case %q", id)
	return vectorcases.Case{}
}

// testKeyring is the keyring of the vectors' test environment, which holds
// VECTOR_KEY (SPEC.md section 13.2).
func testKeyring() string { return vectorcases.Path("keyring.test.json") }

// soundItem is the sound item proof of the vectors, signed by VECTOR_KEY
// and marked test.
func soundItem() string { return vectorcases.Path("positive/item.proof.json") }

// underTest are the options of an offline run of a vector under E_v test
// with the vectors' keyring: supplied-keyring for a sound proof.
func underTest() []string {
	return []string{"--offline", "--environment", "test", "--keyring", testKeyring()}
}

// verifyReport is the subset of the report of SPEC.md section 10.5 these
// tests read from --json.
type verifyReport struct {
	Attribution     string         `json:"attribution"`
	Code            *string        `json:"code"`
	Environment     string         `json:"environment"`
	Label           string         `json:"label"`
	Rows            []verifyRow    `json:"rows"`
	SubjectID       *string        `json:"subject_id"`
	SubmittedAfter  map[string]any `json:"submitted_after"`
	SubmittedBefore map[string]any `json:"submitted_before"`
	TrustRoot       map[string]any `json:"trust_root"`
	Type            *string        `json:"type"`
	Verified        bool           `json:"verified"`
	Window          string         `json:"window"`
}

// verifyRow is one row of a report.
type verifyRow struct {
	Code    *string        `json:"code"`
	Message string         `json:"message"`
	Part    *string        `json:"part"`
	Status  string         `json:"status"`
	Step    string         `json:"step"`
	Values  map[string]any `json:"values"`
}

// String writes a row as "<status> <step> <code> <part>".
func (r verifyRow) String() string {
	fields := []string{r.Status, r.Step}
	if r.Code != nil {
		fields = append(fields, *r.Code)
	}
	if r.Part != nil {
		fields = append(fields, *r.Part)
	}
	return strings.Join(fields, " ")
}

// parseReport reads a --json report, failing the test for anything else.
func parseReport(t *testing.T, stdout string) verifyReport {
	t.Helper()
	var report verifyReport
	if err := json.Unmarshal([]byte(stdout), &report); err != nil || report.Label == "" {
		t.Fatalf("stdout is not a report: %v\n%s", err, stdout)
	}
	return report
}

// rowsWith are the rows of report at step, as their String.
func rowsWith(report verifyReport, step string) []string {
	var out []string
	for _, row := range report.Rows {
		if row.Step == step {
			out = append(out, row.String())
		}
	}
	return out
}

// --- basics ---

// A run of verify that names no proof, with standard input not a pipe (no
// argument, an empty one, --file= or --url=), is a runtime error, exit 1
// (SPEC.md section 12.2), never 0: nothing on stdout and the usage on
// stderr, and nothing at all under --silent. verify --help is exit 0, the
// usage on stdout. Mutant killed: runVerify returning cmd.Help() for such
// a run (exit 0, the help on stdout, under --silent too).
func TestCLI_NoArgs_ShowsHelp(t *testing.T) {
	for _, named := range [][]string{{}, {""}, {"--file="}, {"--url="}} {
		for _, silent := range []bool{false, true} {
			args := slices.Clone(named)
			if silent {
				args = append(args, "--silent")
			}
			stdout, stderr, code := verifyCLI(t, binaryPath, nil, nil, args...)
			switch {
			case code != 1 || stdout != "":
				t.Errorf("verify %q: exit %d, stdout %q", args, code, stdout)
			case silent && stderr != "":
				t.Errorf("verify %q: stderr %q", args, stderr)
			case !silent && (!strings.Contains(stderr, "no proof was given") || !strings.Contains(stderr, "Usage:")):
				t.Errorf("verify %q: stderr %q", args, stderr)
			}
		}
	}
	stdout, _, code := verifyCLI(t, binaryPath, nil, nil, "--help")
	if code != 0 || !strings.Contains(stdout, "Usage:") {
		t.Errorf("verify --help: exit %d\n%s", code, stdout)
	}
}

// A report or a bare rejection that cannot be written to standard output,
// a descriptor open only for reading, is a runtime error, exit 1, said on
// stderr, as JSON and as text, never the code of the label it was not
// given. Mutant killed: writeVerification discarding the write's error
// (exit 6 for the sound item proof, and 4 for the bare rejection).
func TestVerify_AnUnwritableReportIsARuntimeError(t *testing.T) {
	readOnly := filepath.Join(t.TempDir(), "read-only")
	if err := os.WriteFile(readOnly, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	empty := filepath.Join(t.TempDir(), "empty.json")
	if err := os.WriteFile(empty, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	for _, input := range []string{soundItem(), empty} {
		for _, extra := range [][]string{{"--json"}, {}} {
			stdout, err := os.Open(readOnly)
			if err != nil {
				t.Fatal(err)
			}
			args := append(append([]string{input}, extra...), underTest()...)
			stderr, code := verifyCLITo(t, stdout, binaryPath, nil, nil, args...)
			_ = stdout.Close()
			if code != 1 || !strings.Contains(stderr, "the report cannot be written to standard output") {
				t.Errorf("verify %q: exit %d, stderr %q", args, code, stderr)
			}
		}
	}
}

// --silent and --json together are refused before anything runs, exit 1.
// Mutant killed: config.Load without its check of the pair (the run is
// supplied-keyring, exit 6).
func TestCLI_SilentAndJSON_Exclusive(t *testing.T) {
	_, stderr, code := verifyCLI(t, binaryPath, nil, nil, append([]string{soundItem(), "--silent", "--json"}, underTest()...)...)
	if code != 1 || !strings.Contains(stderr, "mutually exclusive") {
		t.Errorf("exit %d\n%s", code, stderr)
	}
}

// --- exit codes (SPEC.md section 12.2) ---

// Each exit code that an offline run can give, through the built binary,
// with --json, as text and with --silent, which carries the same code and
// prints nothing; the online endings 0 and 7, and 2, are in
// verify_online_test.go. Mutants killed: --silent exiting 1 for every
// ending but verified (errSilentFail); a configuration_error taken as a
// bare rejection (exit 4).
func TestVerify_ExitCodes(t *testing.T) {
	keyringDir := t.TempDir()
	badKeyring := filepath.Join(keyringDir, "bad.json")
	if err := os.WriteFile(badKeyring, []byte(`{"keys":[],"keys":[],"version":"1.0"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	for _, c := range []struct {
		name  string
		args  []string
		code  int
		label string
	}{
		{"label failed", append([]string{vectorCase(t, "key_binding-failed-other_key").ProofPath()}, underTest()...), 3, "failed"},
		{"label unattributed", []string{soundItem(), "--offline", "--environment", "test", "--no-keyring"}, 5, "unattributed"},
		{"label supplied-keyring", append([]string{soundItem()}, underTest()...), 6, "supplied-keyring"},
		{"a bare rejection", append([]string{vectorCase(t, "envelope-public_key-padded").ProofPath()}, underTest()...), 4, ""},
		{"configuration_error, no keyring source", []string{soundItem(), "--offline", "--environment", "test"}, 8, ""},
		{"configuration_error, a refused keyring", []string{soundItem(), "--offline", "--environment", "test", "--keyring", badKeyring}, 8, ""},
		{"an input that cannot be read", []string{filepath.Join(keyringDir, "missing.json"), "--offline"}, 1, ""},
		{"an invalid command line", []string{soundItem(), "--offline", "--environment", "production"}, 1, ""},
		{"a keyring file that cannot be read", []string{soundItem(), "--offline", "--environment", "test", "--keyring", keyringDir}, 1, ""},
	} {
		t.Run(c.name, func(t *testing.T) {
			stdout, stderr, code := verifyCLI(t, binaryPath, nil, nil, append(c.args, "--json")...)
			if code != c.code {
				t.Fatalf("--json: exit %d, want %d\n%s\n%s", code, c.code, stdout, stderr)
			}
			switch {
			case c.label != "":
				if report := parseReport(t, stdout); report.Label != c.label {
					t.Errorf("label %q, want %q", report.Label, c.label)
				}
			case c.code == 4 || c.code == 8:
				var rejection map[string]any
				if err := json.Unmarshal([]byte(stdout), &rejection); err != nil || rejection["code"] == nil || rejection["rows"] != nil {
					t.Errorf("stdout is not a bare rejection: %v\n%s", err, stdout)
				}
			default:
				if stdout != "" || stderr == "" {
					t.Errorf("a runtime error wrote stdout %q, stderr %q", stdout, stderr)
				}
			}
			text, _, code := verifyCLI(t, binaryPath, nil, nil, c.args...)
			if code != c.code || (c.code != 1 && text == "") {
				t.Errorf("text: exit %d, want %d, stdout %q", code, c.code, text)
			}
			stdout, stderr, code = verifyCLI(t, binaryPath, nil, nil, append(c.args, "--silent")...)
			if code != c.code || stdout != "" || stderr != "" {
				t.Errorf("--silent: exit %d, want %d, stdout %q, stderr %q", code, c.code, stdout, stderr)
			}
		})
	}
}

// --- --json is exactly the report ---

// --json writes exactly the report of SPEC.md section 10.5 as
// internal/verifier encodes it, JCS and one line feed, for a report and
// for a bare rejection: the command's stdout equals the bytes of the same
// run made in this process. Mutant killed: the report written through
// encoding/json with indentation.
func TestVerify_JSONIsExactlyTheReport(t *testing.T) {
	for _, c := range []struct {
		path string
		cfg  verifier.Config
	}{
		{soundItem(), offlineTestConfig(t)},
		{vectorCase(t, "envelope-public_key-padded").ProofPath(), offlineTestConfig(t)},
	} {
		outcome := verifier.Verify(c.cfg, func() ([]byte, error) { return strictjson.ReadFile(c.path) })
		var want []byte
		var err error
		if outcome.Report != nil {
			want, err = outcome.Report.Encode()
		} else {
			want, err = outcome.Rejection.Encode()
		}
		if err != nil {
			t.Fatal(err)
		}
		stdout, _, _ := verifyCLI(t, binaryPath, nil, nil, append([]string{c.path, "--json"}, underTest()...)...)
		if stdout != string(want)+"\n" {
			t.Errorf("%s: --json is not exactly the report:\n got %s\nwant %s", c.path, stdout, want)
		}
	}
}

// offlineTestConfig is the configuration underTest gives: an offline run
// under E_v test with the vectors' keyring.
func offlineTestConfig(t *testing.T) verifier.Config {
	t.Helper()
	cfg, err := verifier.NewConfig("test")
	if err != nil {
		t.Fatal(err)
	}
	cfg.Keyring = verifier.KeyringSource{Kind: verifier.SourceFile, Path: testKeyring()}
	return cfg
}

// --- inputs read through J1 (SPEC.md section 10.1) ---

// A file, stdin and --url are each read to at most MAX_READ_OCTETS + 1
// octets, so one over MAX_READ_OCTETS is the bare rejection
// input_too_large (exit 4), never an error of its own (exit 1); and an
// empty or blank input is malformed_json, sub_reason syntax (exit 4), from
// a file and from stdin alike. Mutants killed: the URL read capped as
// DownloadBytesCtx caps it (exit 1, "exceeded"); an empty input refused
// as no proof to verify (exit 1).
func TestVerify_InputsAreReadThroughJ1(t *testing.T) {
	big := append([]byte(`{"pad":"`), bytes.Repeat([]byte("a"), strictjson.MaxReadOctets)...)
	big = append(big, `"}`...)
	var served atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		served.Add(1)
		_, _ = w.Write(big)
	}))
	defer srv.Close()
	bigFile := filepath.Join(t.TempDir(), "big.json")
	blankFile := filepath.Join(t.TempDir(), "blank.json")
	if err := os.WriteFile(bigFile, big, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(blankFile, []byte(" \n\t"), 0o600); err != nil {
		t.Fatal(err)
	}
	for _, c := range []struct {
		name  string
		stdin []byte
		args  []string
		code  string
		sub   string
	}{
		{"a file over the bound", nil, []string{bigFile}, "input_too_large", ""},
		{"stdin over the bound", big, []string{"-"}, "input_too_large", ""},
		{"a URL over the bound", nil, []string{"--url", srv.URL + "/proof.json"}, "input_too_large", ""},
		{"an empty stdin", []byte{}, []string{"-"}, "malformed_json", "syntax"},
		{"a blank file", nil, []string{blankFile}, "malformed_json", "syntax"},
	} {
		t.Run(c.name, func(t *testing.T) {
			var stdin io.Reader
			if c.stdin != nil {
				stdin = bytes.NewReader(c.stdin)
			}
			stdout, stderr, code := verifyCLI(t, binaryPath, nil, stdin, append(c.args, append(underTest(), "--json")...)...)
			var rejection map[string]string
			if err := json.Unmarshal([]byte(stdout), &rejection); err != nil || code != 4 || rejection["code"] != c.code || rejection["sub_reason"] != c.sub {
				t.Errorf("exit %d, %v, stdout %s, stderr %s; want exit 4, %s %s", code, err, stdout, stderr, c.code, c.sub)
			}
		})
	}
	if served.Load() != 1 {
		t.Errorf("the URL was read %d times, want 1", served.Load())
	}
}

// --- --expected-hash, as given ---

// --expected-hash is a value given whenever the flag is, "" included: the
// run's EXPECTED_HASH row is then fail expected_hash_mismatch at
// /subject/claims/hash, and with no flag it is skip
// expected_hash_not_checked; an uppercase hash of the claims matches, and
// a value with a space is a value that is not hex, the row's failure, not
// exit 1 (SPEC.md section 10.2, step 13). Mutants killed: the command
// reading the flag's value instead of its Changed state (the empty value
// is not given, so the row skips); the value trimmed before the run (the
// spaced hash matches).
func TestVerify_ExpectedHashAsGiven(t *testing.T) {
	claimsHash := "0be5be687aa93ca2f437e810631742601bd9534be176678c7750f295ecac58e2"
	proof := vectorcases.Path("positive/item-genesis-witness.proof.json")
	for _, c := range []struct {
		name  string
		extra []string
		want  string
		exit  int
	}{
		{"no flag", nil, "skip EXPECTED_HASH expected_hash_not_checked", 6},
		{"given empty", []string{"--expected-hash", ""}, "fail EXPECTED_HASH expected_hash_mismatch /subject/claims/hash", 3},
		{"given empty, with =", []string{"--expected-hash="}, "fail EXPECTED_HASH expected_hash_mismatch /subject/claims/hash", 3},
		{"the claims hash in uppercase", []string{"--expected-hash", strings.ToUpper(claimsHash)}, "pass EXPECTED_HASH /subject/claims/hash", 6},
		{"the claims hash with a space", []string{"--expected-hash", " " + claimsHash}, "fail EXPECTED_HASH expected_hash_mismatch /subject/claims/hash", 3},
	} {
		t.Run(c.name, func(t *testing.T) {
			stdout, stderr, code := verifyCLI(t, binaryPath, nil, nil, append(append([]string{proof, "--json"}, underTest()...), c.extra...)...)
			report := parseReport(t, stdout)
			if got := rowsWith(report, "EXPECTED_HASH"); code != c.exit || len(got) != 1 || got[0] != c.want {
				t.Errorf("exit %d, EXPECTED_HASH %q, want exit %d and %q\n%s", code, got, c.exit, c.want, stderr)
			}
		})
	}
}

// --- a refused keyring file ---

// A keyring file the strict reader refuses (a duplicate name, one over
// MAX_READ_OCTETS, one past MAX_DEPTH) is configuration_error, exit 8,
// and never lets a run pass key binding (SPEC.md section 10.2, step 0).
// Mutant killed: an oversized keyring file taken as one that cannot be
// read (exit 1).
func TestVerify_ARefusedKeyringFileIsConfigurationError(t *testing.T) {
	dir := t.TempDir()
	files := map[string][]byte{
		"duplicate.json": []byte(`{"keys":[],"version":"1.0","version":"1.0"}`),
		"oversized.json": append([]byte(`{"keys":[],"version":"1.0","pad":"`), append(bytes.Repeat([]byte("a"), strictjson.MaxReadOctets), `"}`...)...),
		"deep.json":      []byte(`{"keys":[],"version":"1.0","deep":` + strings.Repeat("[", 40) + strings.Repeat("]", 40) + `}`),
	}
	for name, data := range files {
		t.Run(name, func(t *testing.T) {
			path := filepath.Join(dir, name)
			if err := os.WriteFile(path, data, 0o600); err != nil {
				t.Fatal(err)
			}
			stdout, _, code := verifyCLI(t, binaryPath, nil, nil, soundItem(), "--offline", "--environment", "test", "--keyring", path, "--json")
			var rejection map[string]string
			if err := json.Unmarshal([]byte(stdout), &rejection); err != nil || code != 8 || rejection["code"] != "configuration_error" ||
				!strings.Contains(rejection["message"], "fails a load check") {
				t.Errorf("exit %d, %v: %s", code, err, stdout)
			}
		})
	}
}

// --- --type is retired ---

// verify has no --type: the flag is unknown (exit 1) and no help names
// it. Mutant killed: --type registered again.
func TestVerify_NoTypeFlag(t *testing.T) {
	_, stderr, code := verifyCLI(t, binaryPath, nil, nil, append([]string{soundItem(), "--type", "item"}, underTest()...)...)
	if code != 1 || !strings.Contains(stderr, "unknown flag: --type") {
		t.Errorf("exit %d\n%s", code, stderr)
	}
	if help := helpText(t, "verify"); strings.Contains(help, "--type") {
		t.Errorf("verify --help names --type\n%s", help)
	}
}

// --- the human output ---

// The human output of a supplied-keyring report names the label and what
// it says, the attribution, the trust root with its digest and location,
// the type and subject id, the absent edges with the codes of the rows
// that could have given them, and every row that does not pass; and,
// under a non-production E_v, the environment beside the verdict.
// Mutant killed: the environment left off the verdict line.
func TestVerify_HumanOutput(t *testing.T) {
	stdout, _, code := verifyCLI(t, binaryPath, nil, nil, append([]string{soundItem()}, underTest()...)...)
	if code != 6 {
		t.Fatalf("exit %d\n%s", code, stdout)
	}
	for _, want := range []string{
		"Verdict:      supplied-keyring (environment test, not production)\n",
		"The proof is intact and signed by a key in the keyring the operator supplied; this says nothing about Truestamp.",
		"Attribution:  keyring:",
		"Trust root:   supplied_keyring, from " + testKeyring() + ", SHA-256 ",
		"Environment:  test, not production",
		"Type:         item\n",
		"Submitted before: not established in this run (offline at /commitments/bitcoin, offline at /commitments/stellar)\n",
		"  skip CHAIN offline at /commitments/bitcoin: ",
	} {
		if !strings.Contains(stdout, want) {
			t.Errorf("the output lacks %q\n%s", want, stdout)
		}
	}
	if strings.Contains(stdout, "\n  pass ") || strings.Contains(stdout, "Keyless") {
		t.Errorf("the output lists a pass row or says keyless\n%s", stdout)
	}
}

// A staging proof (SPEC.md section 6.4), marked staging and signed by
// VECTOR_KEY: under the default E_v prod it fails environment_mismatch
// (exit 3) with the message naming staging and the offline command that
// works; under --environment staging with no keyring source it is
// configuration_error (exit 8); and under --environment staging
// --no-keyring it is unattributed (exit 5), with the environment beside
// the verdict and keyless mode in words. Mutants killed: the keyless line
// dropped; the environment left off the verdict line.
func TestVerify_AStagingProof(t *testing.T) {
	path := writeTestProof(t, testProofOptions{seed: vectorSeed, environment: "staging", stellar: "testnet", bitcoin: "regtest"})

	stdout, _, code := verifyCLI(t, binaryPath, nil, nil, path, "--offline", "--json")
	report := parseReport(t, stdout)
	if code != 3 || report.Code == nil || *report.Code != "environment_mismatch" || report.Environment != "prod" {
		t.Fatalf("under prod: exit %d, %+v", code, report)
	}
	if message := report.Rows[4].Message; !strings.Contains(message, "Truestamp's staging environment") ||
		!strings.Contains(message, "run again offline with --environment staging and --keyring FILE") {
		t.Errorf("the ENVIRONMENT message: %q", message)
	}

	_, _, code = verifyCLI(t, binaryPath, nil, nil, path, "--offline", "--environment", "staging")
	if code != 8 {
		t.Errorf("under staging with no keyring source: exit %d, want 8", code)
	}

	stdout, _, code = verifyCLI(t, binaryPath, nil, nil, path, "--offline", "--environment", "staging", "--no-keyring")
	if code != 5 {
		t.Fatalf("under staging, keyless: exit %d\n%s", code, stdout)
	}
	for _, want := range []string{
		"Verdict:      unattributed (environment staging, not production)\n",
		"Keyless:      this run checked the signature only against the key the proof carries, against no keyring\n",
		"Environment:  staging, not production",
	} {
		if !strings.Contains(stdout, want) {
			t.Errorf("the output lacks %q\n%s", want, stdout)
		}
	}
}

// The human output never quotes a marker the proof carries (SPEC.md
// section 6.4): a proof marked with a value no Truestamp environment uses
// fails environment_mismatch under prod, and the text names neither the
// marker nor any environment but prod. Mutant killed: rowsText writing a
// row's values (the declared marker reaches the text).
func TestVerify_TheTextQuotesNoMarker(t *testing.T) {
	const marker = "planted-marker-7q"
	path := writeTestProof(t, testProofOptions{seed: vectorSeed, environment: marker, stellar: "testnet", bitcoin: "regtest"})
	stdout, stderr, code := verifyCLI(t, binaryPath, nil, nil, path, "--offline")
	if code != 3 || strings.Contains(stdout+stderr, marker) || !strings.Contains(stdout, "no Truestamp environment uses") {
		t.Errorf("exit %d\n%s%s", code, stdout, stderr)
	}
	for _, other := range []string{"staging", "production"} {
		if strings.Contains(stdout, other) {
			t.Errorf("the text names %q\n%s", other, stdout)
		}
	}
}

// --- --trace FILE (SPEC.md section 10.6) ---

// --trace FILE writes the run's trace in the canonical form of SPEC.md
// section 9: for the sound item proof, the rows of the vectors' own
// trace, every member but inputs, in order; for a bare rejection, an
// empty list; and a FILE in a directory that does not exist is refused
// before the run, exit 1, with nothing on stdout. Mutants killed: the
// trace written with JCS (one line, no indentation); the directory check
// dropped (the run gives a report and the write fails after it).
func TestVerify_TraceFile(t *testing.T) {
	dir := t.TempDir()
	trace := filepath.Join(dir, "trace.json")
	_, _, code := verifyCLI(t, binaryPath, nil, nil, append([]string{soundItem(), "--trace", trace}, underTest()...)...)
	if code != 6 {
		t.Fatalf("exit %d", code)
	}
	got, err := os.ReadFile(trace)
	if err != nil {
		t.Fatal(err)
	}
	want, err := os.ReadFile(vectorcases.Path("positive/item.trace.json"))
	if err != nil {
		t.Fatal(err)
	}
	if strip := func(data []byte) []byte {
		var rows []map[string]any
		if err := json.Unmarshal(data, &rows); err != nil {
			t.Fatalf("not a trace: %v\n%s", err, data)
		}
		for _, row := range rows {
			delete(row, "inputs")
		}
		out, _ := json.Marshal(rows)
		return out
	}; !bytes.Equal(strip(got), strip(want)) || !bytes.HasPrefix(got, []byte("[\n  {\n    \"construction\": ")) {
		t.Errorf("the trace differs from the vector's:\n%s", got)
	}

	rejected := filepath.Join(dir, "rejected.json")
	_, _, code = verifyCLI(t, binaryPath, nil, nil, append([]string{vectorCase(t, "envelope-public_key-padded").ProofPath(), "--trace", rejected}, underTest()...)...)
	if got, err := os.ReadFile(rejected); code != 4 || err != nil || string(got) != "[]\n" {
		t.Errorf("a bare rejection's trace: exit %d, %v, %q", code, err, got)
	}

	stdout, stderr, code := verifyCLI(t, binaryPath, nil, nil, append([]string{soundItem(), "--trace", filepath.Join(dir, "missing", "trace.json")}, underTest()...)...)
	if code != 1 || stdout != "" || !strings.Contains(stderr, "--trace names a file in a directory that does not exist") {
		t.Errorf("a trace in a missing directory: exit %d, stdout %q, stderr %q", code, stdout, stderr)
	}
}

// --- --remote ---

// --remote with no credential is refused before the run, exit 1, and sends
// nothing. Mutant killed: runVerify without its check of the credential
// (the run is supplied-keyring, exit 6).
func TestCLI_Verify_RemoteRequiresAuth(t *testing.T) {
	srv, hits := countingAPI(t)
	env := []string{"HOME=" + t.TempDir(), "XDG_CONFIG_HOME=" + t.TempDir()}
	_, stderr, code := verifyCLI(t, binaryPath, env, nil, soundItem(), "--offline", "--remote", "--base-url", srv.URL)
	if code != 1 || !strings.Contains(stderr, "not authenticated") {
		t.Errorf("remote without a credential: exit %d\n%s", code, stderr)
	}
	if n := hits.Load(); n != 0 {
		t.Errorf("verify --remote without a credential sent %d requests, want 0", n)
	}
}

// Under --remote the exit code and stdout are the local run's alone, and
// the Truestamp service's verdict, or why there is none, is printed on
// stderr beside the report, never as a row of it: a proof the service
// says passed exits as this run's own label says, and so does one the
// service is not asked about. Mutants killed: the service's verdict
// deciding the exit code (exit 0 for the passed answer); the verdict
// written to stdout (the --json report does not parse). The service is a
// loopback stub whose answer passes the proof with one step, and the proof
// the earlier bundle parser posts is a bundle of its layout, which the
// local run refuses as invalid_envelope.
func TestCLI_Verify_Remote(t *testing.T) {
	serverReport := []byte(`{"result":{"passed":true,"steps":[{"category":"structural","group":"Structure","message":"Proof version 1 (expected 1)","status":"pass"}]}}`)
	var posts atomic.Int32
	mux := http.NewServeMux()
	mux.HandleFunc("/api/json/proof/verify", func(w http.ResponseWriter, r *http.Request) {
		posts.Add(1)
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write(serverReport)
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()
	remote := []string{"--remote", "--base-url", srv.URL, "--api-key", "test-key", "--offline"}

	stdout, stderr, code := verifyCLI(t, binaryPath, nil, nil, append([]string{prodPath(testfixtures.ProdComplete), "--json"}, remote...)...)
	var rejection map[string]string
	if err := json.Unmarshal([]byte(stdout), &rejection); err != nil || code != 4 || rejection["code"] != "invalid_envelope" {
		t.Errorf("a proof the service passes: exit %d, %v, %s", code, err, stdout)
	}
	if !strings.Contains(stderr, "The Truestamp service's verdict, beside this report and not part of it: passed.") || posts.Load() != 1 {
		t.Errorf("the service's verdict (%d posts): %s", posts.Load(), stderr)
	}

	stdout, stderr, code = verifyCLI(t, binaryPath, nil, nil, append(append([]string{soundItem(), "--json"}, remote...), "--environment", "test", "--keyring", testKeyring())...)
	if report := parseReport(t, stdout); code != 6 || report.Label != "supplied-keyring" || !strings.Contains(stderr, "The Truestamp service gave no verdict") {
		t.Errorf("a proof the service is not asked about: exit %d, %s\n%s", code, report.Label, stderr)
	}
}

// --- config ---

func TestCLI_ConfigPath(t *testing.T) {
	out, code := runCLIText(t, "config", "path")
	if code != 0 || len(out) == 0 {
		t.Errorf("config path: exit %d %q", code, out)
	}
}

func TestCLI_ConfigShow(t *testing.T) {
	out, code := runCLIText(t, "config", "show")
	if code != 0 || !containsString(out, "API URL") || !containsString(out, "Verification") {
		t.Errorf("config show: exit %d\n%s", code, out)
	}
}

func TestCLI_EnvVarOverride(t *testing.T) {
	cmd := exec.Command(binaryPath, "config", "show")
	cmd.Env = append(os.Environ(), "TRUESTAMP_BASE_URL=https://custom.example.com")
	out, err := cmd.CombinedOutput()
	if err != nil || !containsString(string(out), "https://custom.example.com") {
		t.Errorf("env var override: %v\n%s", err, out)
	}
}

func TestCLI_FlagOverridesEnv(t *testing.T) {
	cmd := exec.Command(binaryPath, "config", "show", "--base-url=https://flag.example.com")
	cmd.Env = append(os.Environ(), "TRUESTAMP_BASE_URL=https://env.example.com")
	out, err := cmd.CombinedOutput()
	if err != nil || !containsString(string(out), "https://flag.example.com") {
		t.Errorf("flag override: %v\n%s", err, out)
	}
}

func TestCLI_Completion(t *testing.T) {
	for _, shell := range []string{"zsh", "bash", "fish"} {
		cmd := exec.Command(binaryPath, "completion", shell)
		var stderr bytes.Buffer
		cmd.Stderr = &stderr
		out, err := cmd.Output()
		if err != nil || len(out) == 0 {
			t.Errorf("completion %s: %v", shell, err)
		}
		if shell == "zsh" && stderr.Len() > 0 {
			t.Errorf("completion zsh wrote to stderr: %s", stderr.String())
		}
	}
}

// The flag, env var and config key that once skipped the signature check.
// They are assembled from pieces so that no file under cmd/ or internal/
// spells them: a grep for the old name finds nothing, and these tests
// still hand every spelling to the binary.
var (
	retiredSignatureFlag   = "--skip" + "-signatures"
	retiredSignatureEnv    = "TRUESTAMP_VERIFY_SKIP" + "_SIGNATURES"
	retiredSignatureConfig = "skip" + "_signatures"
)

// Nothing skips the signature check (SPEC.md section 6.2): a proof whose
// signature is over the wrong message fails signature_invalid, exit 3,
// with the retired environment variable or configuration key set, and the
// retired flag is an unknown flag, which no help offers. Mutant killed:
// Signature skipped when a skip setting is present.
func TestCLI_NothingSkipsTheSignature(t *testing.T) {
	proof := vectorCase(t, "signature-over-m").ProofPath()
	configPath := filepath.Join(t.TempDir(), "config.toml")
	if err := os.WriteFile(configPath, []byte("[verify]\n"+retiredSignatureConfig+" = true\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	for name, run := range map[string]struct {
		env  []string
		args []string
	}{
		"plain":      {nil, nil},
		"env var":    {[]string{retiredSignatureEnv + "=true"}, nil},
		"config key": {nil, []string{"--config", configPath}},
	} {
		stdout, _, code := verifyCLI(t, binaryPath, run.env, nil, append(append([]string{proof, "--json"}, underTest()...), run.args...)...)
		if report := parseReport(t, stdout); code != 3 || report.Code == nil || *report.Code != "signature_invalid" {
			t.Errorf("%s: exit %d, %+v", name, code, report.Code)
		}
	}
	_, stderr, code := verifyCLI(t, binaryPath, nil, nil, append([]string{proof, retiredSignatureFlag}, underTest()...)...)
	if code != 1 || !strings.Contains(stderr, "unknown flag: "+retiredSignatureFlag) {
		t.Errorf("the retired flag: exit %d\n%s", code, stderr)
	}
	if help := helpText(t, "verify"); strings.Contains(help, retiredSignatureFlag) {
		t.Errorf("verify --help still offers %s", retiredSignatureFlag)
	}
}
