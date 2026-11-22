// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"bytes"
	"encoding/json/jsontext"
	"encoding/json/v2"
	"errors"
	"go/ast"
	"go/parser"
	"go/token"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"testing"

	"github.com/truestamp/truestamp-cli/internal/auth"
	"github.com/truestamp/truestamp-cli/internal/config"
	"github.com/truestamp/truestamp-cli/internal/jsonapi"
	"github.com/truestamp/truestamp-cli/internal/redact"
)

// fixedServer answers every request with one status and body, and records
// the tenant header of each request it saw.
type fixedServer struct {
	*httptest.Server
	mu      sync.Mutex
	tenants []string
}

func (s *fixedServer) sawTenants() []string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return slices.Clone(s.tenants)
}

func startFixedServer(t *testing.T, status int, body string) *fixedServer {
	t.Helper()
	s := &fixedServer{}
	s.Server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		s.mu.Lock()
		s.tenants = append(s.tenants, r.Header.Get("tenant"))
		s.mu.Unlock()
		w.Header().Set("Content-Type", "application/vnd.api+json")
		w.WriteHeader(status)
		_, _ = w.Write([]byte(body))
	}))
	t.Cleanup(s.Close)
	return s
}

// runCLIStdin is runCLI with standard input.
func runCLIStdin(t *testing.T, stdin string, args ...string) (stdout, stderr string, exit int) {
	t.Helper()
	cmd := exec.Command(binaryPath, args...)
	var outBuf, errBuf strings.Builder
	cmd.Stdin = strings.NewReader(stdin)
	cmd.Stdout = &outBuf
	cmd.Stderr = &errBuf
	cmd.Env = cleanEnv()
	err := cmd.Run()
	if ee, ok := err.(*exec.ExitError); ok {
		exit = ee.ExitCode()
	} else if err != nil {
		t.Fatalf("unexpected exec error: %v", err)
	}
	return outBuf.String(), errBuf.String(), exit
}

// oneFailureObject decodes stdout as exactly one JSON object and nothing
// after it, and checks its code is in the registry and its message is
// given.
func oneFailureObject(t *testing.T, stdout string) map[string]any {
	t.Helper()
	dec := jsontext.NewDecoder(bytes.NewReader([]byte(stdout)))
	var obj map[string]any
	if err := json.UnmarshalDecode(dec, &obj); err != nil {
		t.Fatalf("stdout is not one JSON object: %v\n%s", err, stdout)
	}
	if _, err := dec.ReadValue(); err == nil {
		t.Fatalf("stdout carries more than one JSON value:\n%s", stdout)
	}
	code, _ := obj["code"].(string)
	if !slices.ContainsFunc(errorCodes, func(e errorCode) bool { return e.Code == code }) {
		t.Errorf("code %q is not in the error-code registry:\n%s", code, stdout)
	}
	if msg, _ := obj["message"].(string); msg == "" {
		t.Errorf("the failure object carries no message:\n%s", stdout)
	}
	return obj
}

const (
	unauthorizedBody = `{"errors":[{"status":"401","code":"unauthorized","title":"Unauthorized","detail":"Authentication required."}]}`
	notFoundBody     = `{"errors":[{"status":"404","code":"not_found","title":"Not Found","detail":"not found"}]}`
	policyBody       = `{"errors":[{"status":"403","code":"forbidden","title":"Forbidden","detail":"forbidden"}]}`
	headerRefusal    = `{"errors":[{"status":"403","code":"forbidden","title":"Forbidden","detail":"Tenant refused","source":{"header":"tenant"}}]}`
	foreignTeam      = "019dbd00-0000-7000-8000-0000000000ff"
)

// TestFailure_JSONIsOneObject: every --json failure prints exactly one
// {code, message} object on stdout, http_status when an HTTP refusal
// caused it, and nothing on stderr, whichever command failed and however:
// a refusal the shared renderer gives, the teams renderer's, a card that
// used to be a banner, a plain error Execute is handed, a usage error
// before the configuration loads, and the two creates whose objects used
// to have shapes of their own. Under --silent the same runs print
// nothing at all. Mutants killed: Execute printing a plain error as text
// under --json; renderAPIError or teamRenderError printing a banner under
// --json; the no-team card printed under --json; the keys get miss
// classed command_failed; http_status left out.
func TestFailure_JSONIsOneObject(t *testing.T) {
	keyring, err := os.ReadFile(filepath.Join("..", "testdata", "verification", "spec", "keyring.production.json"))
	if err != nil {
		t.Fatal(err)
	}
	keyringSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/.well-known/keyring.json" {
			_, _ = w.Write(keyring)
			return
		}
		http.NotFound(w, r)
	}))
	t.Cleanup(keyringSrv.Close)
	unauthorized := startFixedServer(t, 401, unauthorizedBody)
	notFound := startFixedServer(t, 404, notFoundBody)
	tenant := startFixedServer(t, 403, tenantRefusalBody)
	teamLimit := startFixedServer(t, 422, `{"errors":[{"status":"422","code":"invalid","detail":"You have reached your plan's team limit."}]}`)

	cases := []struct {
		name   string
		stdin  string
		args   []string
		code   string
		status int
	}{
		{"keys get unknown", "", []string{"keys", "get", "deadbeef", "--base-url", keyringSrv.URL}, codeNotFound, 0},
		{"teams current no team", "", []string{"teams", "current", "--api-key", "k", "--base-url", notFound.URL}, codeNoTeamConfigured, 0},
		{"blocks get missing", "", []string{"blocks", "get", testBlockID, "--api-key", "k", "--base-url", notFound.URL}, codeNotFound, 404},
		{"items get missing", "", []string{"items", "get", testItemID, "--api-key", "k", "--base-url", notFound.URL}, codeNotFound, 404},
		{"teams get missing", "", []string{"teams", "get", testTeamOtherID, "--api-key", "k", "--base-url", notFound.URL}, codeNotFound, 404},
		{"entropy latest bad source", "", []string{"entropy", "latest", "--source", "bogus", "--api-key", "k", "--base-url", notFound.URL}, codeCommandFailed, 0},
		{"jcs bad input", "not-json", []string{"jcs"}, codeMalformedJSON, 0},
		{"hash one missing input", "", []string{"hash", filepath.Join(t.TempDir(), "missing.bin")}, codeCommandFailed, 0},
		{"usage error", "", []string{"blocks", "get"}, codeInvalidUsage, 0},
		{"no credential", "", []string{"blocks", "latest", "--base-url", notFound.URL}, codeNotAuthenticated, 0},
		{"rejected key", "", []string{"blocks", "latest", "--api-key", "bad", "--base-url", unauthorized.URL}, codeCredentialRejected, 401},
		{"rejected key on teams", "", []string{"teams", "list", "--api-key", "bad", "--base-url", unauthorized.URL}, codeCredentialRejected, 401},
		{"refused team", "", []string{"items", "list", "--team", foreignTeam, "--api-key", "k", "--base-url", tenant.URL}, codeTeamNotAccessible, 403},
		{"teams create limit", "", []string{"teams", "create", "Acme", "--api-key", "k", "--base-url", teamLimit.URL}, codeTeamLimitReached, 422},
		{"items create rejected key", "", []string{"items", "create", "-n", "Doc", "-d", longDesc, "--api-key", "bad", "--base-url", unauthorized.URL}, codeCredentialRejected, 401},
		{"items create 404", "", []string{"items", "create", "-n", "Doc", "-d", longDesc, "--api-key", "k", "--base-url", notFound.URL}, codeNotFound, 404},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			stdout, stderr, exit := runCLIStdin(t, tc.stdin, append(tc.args, "--json")...)
			if exit != 1 {
				t.Fatalf("exit = %d, want 1\nstdout=%s\nstderr=%s", exit, stdout, stderr)
			}
			if stderr != "" {
				t.Errorf("stderr must be empty under --json, got %q", stderr)
			}
			obj := oneFailureObject(t, stdout)
			if obj["code"] != tc.code {
				t.Errorf("code = %v, want %s\n%s", obj["code"], tc.code, stdout)
			}
			status, hasStatus := obj["http_status"]
			switch {
			case tc.status == 0 && hasStatus:
				t.Errorf("http_status %v given for a failure no HTTP refusal caused", status)
			case tc.status != 0 && status != float64(tc.status):
				t.Errorf("http_status = %v, want %d", status, tc.status)
			}

			if tc.code == codeInvalidUsage {
				return // --silent is itself a flag the usage check reads first
			}
			stdout, stderr, exit = runCLIStdin(t, tc.stdin, append(tc.args, "--silent")...)
			if exit != 1 || stdout != "" || stderr != "" {
				t.Errorf("--silent: exit %d, stdout %q, stderr %q; want 1 and nothing printed", exit, stdout, stderr)
			}
		})
	}
}

// TestFailure_HashOverSeveralInputsKeepsItsRecord: a --json hash over
// several inputs, one of which fails, prints the array of the digests it
// computed and no second object: the array is the record. Mutant killed:
// Execute writing a failure object after a record was written.
func TestFailure_HashOverSeveralInputsKeepsItsRecord(t *testing.T) {
	dir := t.TempDir()
	good := filepath.Join(dir, "a.bin")
	if err := os.WriteFile(good, []byte("abc"), 0o600); err != nil {
		t.Fatal(err)
	}
	stdout, stderr, exit := runCLI(t, "hash", good, filepath.Join(dir, "missing.bin"), "--json")
	if exit != 1 {
		t.Fatalf("exit = %d, want 1", exit)
	}
	var arr []map[string]any
	if err := json.Unmarshal([]byte(stdout), &arr); err != nil || len(arr) != 1 {
		t.Fatalf("stdout should be the one-element digest array: %v\n%s", err, stdout)
	}
	if !strings.Contains(stderr, "missing.bin") {
		t.Errorf("the failed input is named on stderr, got %q", stderr)
	}
}

// TestFailure_OwnRecordsUnchanged: auth status keeps its record and verify
// its bare rejection (SPEC.md section 10.5), neither replaced nor followed
// by a failure object.
func TestFailure_OwnRecordsUnchanged(t *testing.T) {
	stdout, _, exit := runCLI(t, "auth", "status", "--json")
	if exit != 1 {
		t.Fatalf("auth status exit = %d, want 1", exit)
	}
	var rec map[string]any
	if err := json.Unmarshal([]byte(stdout), &rec); err != nil {
		t.Fatalf("auth status stdout is not its record: %v\n%s", err, stdout)
	}
	if rec["ok"] != false || rec["reason"] != "not_authenticated" {
		t.Errorf("auth status record = %v", rec)
	}
	if _, has := rec["code"]; has {
		t.Errorf("auth status record carries a failure code: %v", rec)
	}

	stdout, _, _ = verifyCLI(t, binaryPath, nil, strings.NewReader("not-json"), "--json", "--offline")
	var rej map[string]any
	if err := json.Unmarshal([]byte(stdout), &rej); err != nil {
		t.Fatalf("verify stdout is not one bare rejection: %v\n%s", err, stdout)
	}
	if rej["code"] != "malformed_json" || rej["sub_reason"] != "syntax" {
		t.Errorf("verify rejection = %v, want SPEC.md's bare malformed_json", rej)
	}

	// A proof verify cannot read is a runtime error (SPEC.md section
	// 12.2), no report and no rejection: stdout stays empty, so nothing
	// there can be read as a rejection with a code SPEC.md does not define.
	stdout, stderr, exit := verifyCLI(t, binaryPath, nil, nil, filepath.Join(t.TempDir(), "missing.json"), "--json", "--offline")
	if exit != 1 || stdout != "" || stderr == "" {
		t.Errorf("verify of a missing file: exit %d, stdout %q, stderr %q; want 1, nothing on stdout, the error on stderr", exit, stdout, stderr)
	}
}

// TestCLI_RejectedKey_CredentialRejected: a 401 on a request that carried
// an API key reads the same in every group: the banner auth status uses
// and a hint naming the API key, never the no-credential advice to set
// TRUESTAMP_API_KEY, which the holder just did. Mutants killed:
// renderAPIError sending every 401 to the not-authenticated banner;
// teamRenderError keeping its own 401 wording; items create's 401 left to
// Execute's bare "HTTP 401" line; proofs get's raw route keeping its
// "Proof request refused" banner for a 401; the resolve step of a UUIDv7
// turning a 401 into "pass --type explicitly".
func TestCLI_RejectedKey_CredentialRejected(t *testing.T) {
	srv := startFixedServer(t, 401, unauthorizedBody)
	for _, args := range [][]string{
		{"blocks", "latest"},
		{"beacons", "latest"},
		{"items", "list"},
		{"entropy", "latest"},
		{"teams", "list"},
		{"items", "create", "-n", "Doc", "-d", longDesc},
		// The raw proof route, and the resolve step a UUIDv7 with no
		// --type takes first.
		{"proofs", "get", soundItemCode},
		{"proofs", "get", testItemID},
	} {
		t.Run(strings.Join(args[:min(len(args), 3)], " "), func(t *testing.T) {
			_, stderr, exit := runCLI(t, append(args, "--api-key", "bad", "--base-url", srv.URL)...)
			if exit != 1 {
				t.Fatalf("exit = %d, want 1\n%s", exit, stderr)
			}
			for _, want := range []string{"Credential rejected", "The server rejected the API key"} {
				if !strings.Contains(stderr, want) {
					t.Errorf("stderr missing %q:\n%s", want, stderr)
				}
			}
			for _, bad := range []string{"Not authenticated", "or set TRUESTAMP_API_KEY", "HTTP 401", "pass --type"} {
				if strings.Contains(stderr, bad) {
					t.Errorf("stderr must not say %q:\n%s", bad, stderr)
				}
			}
		})
	}
}

// TestCLI_ItemsCreate_404KeepsTheServersAnswer: a 404 on POST /items (a
// wrong --base-url path, or a proxy answering for the API) prints the
// server's status and detail, never renderAPIError's "item not found",
// which makes no sense for a create. Mutant killed: explainCreateError
// sending every API error to renderAPIError.
func TestCLI_ItemsCreate_404KeepsTheServersAnswer(t *testing.T) {
	srv := startFixedServer(t, 404, `{"errors":[{"status":"404","code":"not_found","title":"Not Found","detail":"no route here"}]}`)
	_, stderr, exit := runCLI(t, "items", "create", "-n", "Doc", "-d", longDesc, "--api-key", "k", "--base-url", srv.URL)
	if exit != 1 {
		t.Fatalf("exit = %d, want 1\n%s", exit, stderr)
	}
	if !strings.Contains(stderr, "HTTP 404: no route here") || strings.Contains(stderr, "item not found") {
		t.Errorf("stderr should carry the server's answer, not \"item not found\":\n%s", stderr)
	}
}

// oauthModeAuthorizer is an Authorizer in OAuth mode, for the hint.
type oauthModeAuthorizer struct{ auth.Authorizer }

func (oauthModeAuthorizer) Mode() auth.Mode { return auth.ModeOAuth }

// TestCredentialRejectedHint_NamesTheMode: the hint names the credential
// the run resolved: an OAuth session is told to sign in again, an API key
// to check the key. Mutant killed: one hint for both modes.
func TestCredentialRejectedHint_NamesTheMode(t *testing.T) {
	orig := auth.Default()
	t.Cleanup(func() { auth.SetDefault(orig) })

	auth.SetDefault(oauthModeAuthorizer{auth.APIKeyAuthorizer("k")})
	if got := credentialRejectedHint(); got != "The server rejected the OAuth session. Run 'truestamp auth login' to sign in again." {
		t.Errorf("OAuth hint = %q", got)
	}
	auth.SetDefault(auth.APIKeyAuthorizer("k"))
	if got := credentialRejectedHint(); !strings.Contains(got, "rejected the API key") {
		t.Errorf("API key hint = %q", got)
	}
}

// TestCLI_TenantRefused_NamesTeam: the server's refusal of the team a
// request sent as its tenant names that team, in every group that renders
// through renderAPIError and in items create, whether the refusal carries
// source.header "tenant" or only the server's tenant detail; a 403 that
// refuses the record itself (an update the role does not allow) still
// names the record. Mutants killed: the 403 branch blaming "that item"
// for a tenant refusal; the source.header check dropped (the second body
// has another detail); the detail fallback dropped; every 403 under a
// team read as a tenant refusal; proofs get's raw route and its resolve
// step of a UUIDv7 not naming the team (or reading source.header).
func TestCLI_TenantRefused_NamesTeam(t *testing.T) {
	for _, body := range []string{tenantRefusalBody, headerRefusal} {
		srv := startFixedServer(t, 403, body)
		for _, args := range [][]string{
			{"items", "list"},
			{"blocks", "latest"},
			{"items", "create", "-n", "Doc", "-d", longDesc},
			{"proofs", "get", soundItemCode},
			{"proofs", "get", testItemID},
		} {
			_, stderr, exit := runCLI(t, append(args, "--team", foreignTeam, "--api-key", "k", "--base-url", srv.URL)...)
			if exit != 1 {
				t.Fatalf("%v: exit = %d, want 1\n%s", args, exit, stderr)
			}
			if !strings.Contains(stderr, "refused team "+foreignTeam) {
				t.Errorf("%v: stderr should name the refused team:\n%s", args, stderr)
			}
			if strings.Contains(stderr, "that item") || strings.Contains(stderr, "that block") ||
				strings.Contains(stderr, "that proof") || strings.Contains(stderr, "pass --type") {
				t.Errorf("%v: stderr blames the record:\n%s", args, stderr)
			}
		}
		if got := srv.sawTenants(); !slices.Contains(got, foreignTeam) {
			t.Errorf("the requests carried tenants %q, want %s", got, foreignTeam)
		}
	}

	policy := startFixedServer(t, 403, policyBody)
	_, stderr, exit := runCLI(t, "items", "update", testItemID, "--visibility", "public",
		"--team", foreignTeam, "--api-key", "k", "--base-url", policy.URL)
	if exit != 1 {
		t.Fatalf("items update: exit = %d, want 1\n%s", exit, stderr)
	}
	if !strings.Contains(stderr, "you do not have access to that item") || strings.Contains(stderr, "refused team") {
		t.Errorf("a policy refusal names the item, not the team:\n%s", stderr)
	}
}

// TestFailure_BannersGoThroughFailWith: a failure banner printed anywhere
// but failWith would print under --json too, beside or instead of the one
// object. auth status is the one exception: its banner follows its own
// record's --json branch.
func TestFailure_BannersGoThroughFailWith(t *testing.T) {
	files, err := filepath.Glob("*.go")
	if err != nil {
		t.Fatal(err)
	}
	for _, f := range files {
		if strings.HasSuffix(f, "_test.go") {
			continue
		}
		src, err := os.ReadFile(f)
		if err != nil {
			t.Fatal(err)
		}
		n := strings.Count(string(src), "ui.FailureBanner(")
		want := 0
		switch f {
		case "failure.go", "auth.go":
			want = 1
		}
		if n != want {
			t.Errorf("%s calls ui.FailureBanner %d times, want %d: render a failure with failWith", f, n, want)
		}
	}
}

// TestFailure_CodesAreTheRegistry: every code a failure is built with is
// one of the registry's constants, every constant is in the registry,
// and no code is listed twice. Mutant killed: a cliError literal written
// with a code string of its own.
func TestFailure_CodesAreTheRegistry(t *testing.T) {
	fset := token.NewFileSet()
	files, err := filepath.Glob("*.go")
	if err != nil {
		t.Fatal(err)
	}
	consts := map[string]bool{}
	var used []string
	var listed []string
	for _, name := range files {
		if strings.HasSuffix(name, "_test.go") {
			continue
		}
		f, err := parser.ParseFile(fset, name, nil, 0)
		if err != nil {
			t.Fatal(err)
		}
		ast.Inspect(f, func(n ast.Node) bool {
			switch n := n.(type) {
			case *ast.ValueSpec:
				for _, id := range n.Names {
					if strings.HasPrefix(id.Name, "code") && len(id.Name) > 4 && name == "failure.go" {
						consts[id.Name] = true
					}
				}
				if len(n.Names) == 1 && n.Names[0].Name == "errorCodes" {
					for _, el := range n.Values[0].(*ast.CompositeLit).Elts {
						listed = append(listed, el.(*ast.CompositeLit).Elts[0].(*ast.Ident).Name)
					}
				}
			case *ast.CompositeLit:
				if id, ok := n.Type.(*ast.Ident); !ok || id.Name != "cliError" {
					return true
				}
				for _, el := range n.Elts {
					kv, ok := el.(*ast.KeyValueExpr)
					if !ok || kv.Key.(*ast.Ident).Name != "code" {
						continue
					}
					switch v := kv.Value.(type) {
					case *ast.Ident:
						used = append(used, v.Name)
					case *ast.CallExpr:
						if fn, ok := v.Fun.(*ast.Ident); !ok || fn.Name != "apiFailureCode" {
							t.Errorf("%s: a cliError's code is a registry constant or apiFailureCode, not %T", fset.Position(v.Pos()), v)
						}
					default:
						t.Errorf("%s: a cliError's code is a registry constant, not %T", fset.Position(v.Pos()), v)
					}
				}
			}
			return true
		})
	}
	for _, u := range used {
		if !consts[u] {
			t.Errorf("cliError built with %s, which is not a registry constant", u)
		}
	}
	for c := range consts {
		if !slices.Contains(listed, c) {
			t.Errorf("%s is not listed in errorCodes", c)
		}
	}
	seen := map[string]bool{}
	for _, e := range errorCodes {
		if seen[e.Code] {
			t.Errorf("errorCodes lists %s twice", e.Code)
		}
		seen[e.Code] = true
	}
}

// withTeam sets the resolved configuration to one with team in scope, for
// the rest of the test.
func withTeam(t *testing.T, team string) {
	t.Helper()
	orig := appConfig
	t.Cleanup(func() { appConfig = orig })
	appConfig = &config.Config{Team: team}
}

// TestIsTenantRefusal: a 403 naming the tenant header (or carrying the
// server's tenant detail) is a tenant refusal only while a team is in
// scope, and a policy 403 never is.
func TestIsTenantRefusal(t *testing.T) {
	orig := appConfig
	t.Cleanup(func() { appConfig = orig })
	byHeader := jsonapi.ParseError(403, []byte(headerRefusal))
	byDetail := jsonapi.ParseError(403, []byte(tenantRefusalBody))
	policy := jsonapi.ParseError(403, []byte(policyBody))

	appConfig = nil
	if isTenantRefusal(byHeader) {
		t.Error("no configuration: no team in scope, so no tenant refusal")
	}
	withTeam(t, foreignTeam)
	if !isTenantRefusal(byHeader) || !isTenantRefusal(byDetail) {
		t.Error("a team in scope: source.header and the detail each mark a tenant refusal")
	}
	if isTenantRefusal(policy) {
		t.Error("a policy 403 is not a tenant refusal")
	}
	withTeam(t, "")
	if isTenantRefusal(byHeader) {
		t.Error("no team in scope: no tenant was sent, so no tenant refusal")
	}
}

// TestFailureRecord_RedactsTheMessage: the failure object goes to stdout,
// which logs and CI artifacts keep, so a message that echoes a bearer
// token or an api_key query comes out redacted, whether the error was
// classified here or carried its own code. Mutant killed: the message
// written as the error gave it.
func TestFailureRecord_RedactsTheMessage(t *testing.T) {
	orig := appConfig
	t.Cleanup(func() { appConfig = orig })
	appConfig = nil
	for _, err := range []error{
		errors.New("refresh failed: Authorization: Bearer abc123secret, url https://x.test/ws?api_key=k-SECRET&vsn=2"),
		&cliError{code: codeCommandFailed, message: "token Bearer abc123secret api_key=k-SECRET"},
	} {
		msg, _ := failureRecord(err)["message"].(string)
		if strings.Contains(msg, "abc123secret") || strings.Contains(msg, "k-SECRET") {
			t.Errorf("message keeps a secret: %q", msg)
		}
		if !strings.Contains(msg, redact.REDACTED) {
			t.Errorf("message has no %s marker: %q", redact.REDACTED, msg)
		}
	}
}

// TestSchema_ErrorCodesAreTheRegistry: schema get error-codes publishes
// errorCodes as it stands, every code with its meaning, in its order.
// Mutant killed: the document built from a list that drops a code.
func TestSchema_ErrorCodesAreTheRegistry(t *testing.T) {
	doc := runSchema(t, "error-codes")
	got, _ := doc["error_codes"].([]any)
	if len(got) != len(errorCodes) {
		t.Fatalf("error_codes has %d entries, want %d", len(got), len(errorCodes))
	}
	for i, e := range errorCodes {
		entry, _ := got[i].(map[string]any)
		if entry["code"] != e.Code || entry["meaning"] != e.Meaning {
			t.Errorf("error_codes[%d] = %v, want %+v", i, entry, e)
		}
	}
}
