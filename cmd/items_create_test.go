// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/truestamp/truestamp-cli/internal/items"
	"github.com/truestamp/truestamp-cli/internal/strictjson"
)

// validSHA256Hex is a 64-char hex string with the shape of a SHA-256
// claims hash.
const validSHA256Hex = "0000000000000000000000000000000000000000000000000000000000000000"

// longDesc is a description of more than 32 code points.
const longDesc = "Lorem ipsum dolor sit amet, consectetur adipiscing elit."

// createdItemDoc is a 201 answer to a create, carrying the claims_jcs and
// claims_hash the server returns.
const createdItemDoc = `{"data":{"id":"019a0000-0000-7000-8000-000000000001","type":"item","attributes":{` +
	`"display_name":"Echo","visibility":"private","state":"created",` +
	`"claims_jcs":"{\"hash\":\"ab\",\"name\":\"Echo\"}",` +
	`"claims_hash":"5d0c6ffbd0e4e3e3b1b1c8e8b0b2d7e3a7f6c9a5b4d3c2e1f0a9b8c7d6e5f4a3"}}}`

// createServer is a loopback items API answering every request with status
// and body, recording the last request body and counting requests.
type createServer struct {
	*itemsServer
	hits *atomic.Int32
}

func startCreateServer(t *testing.T, status int, body string) createServer {
	t.Helper()
	var hits atomic.Int32
	s := startItemsServer(t, func(w http.ResponseWriter, _ *http.Request) {
		hits.Add(1)
		w.WriteHeader(status)
		_, _ = w.Write([]byte(body))
	})
	return createServer{itemsServer: s, hits: &hits}
}

// sentClaims is the claims text of the last request, as the bytes between
// `"claims":` and the attribute after it. createBody writes the claims
// first and then visibility, which every create sends.
func (s createServer) sentClaims(t *testing.T) string {
	t.Helper()
	body := s.body()
	const head = `{"data":{"type":"item","attributes":{"claims":`
	rest, ok := strings.CutPrefix(body, head)
	if !ok {
		t.Fatalf("request body does not begin with %s: %s", head, body)
	}
	at := strings.LastIndex(rest, `,"visibility":`)
	if at < 0 {
		t.Fatalf("request body has no visibility after the claims: %s", body)
	}
	return rest[:at]
}

// runCreate runs `truestamp items create args...` with stdin and the clean
// environment, and returns stdout, stderr and the exit code.
func runCreate(t *testing.T, stdin string, args ...string) (stdout, stderr string, exit int) {
	t.Helper()
	cmd := exec.Command(binaryPath, append([]string{"items", "create"}, args...)...)
	cmd.Env = cleanEnv()
	cmd.Stdin = strings.NewReader(stdin)
	var outBuf, errBuf bytes.Buffer
	cmd.Stdout, cmd.Stderr = &outBuf, &errBuf
	err := cmd.Run()
	var ee *exec.ExitError
	switch {
	case errors.As(err, &ee):
		exit = ee.ExitCode()
	case err != nil:
		t.Fatalf("running: %v", err)
	}
	return outBuf.String(), errBuf.String(), exit
}

func writeClaims(t *testing.T, text string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "claims.json")
	if err := os.WriteFile(path, []byte(text), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

// --- No-args / help ---

func TestCLI_ItemsCreate_NoArgs_ShowsHelp(t *testing.T) {
	cmd := exec.Command(binaryPath, "items", "create")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("create with no args should exit 0, got: %s", err)
	}
	if !containsString(string(out), "Usage") {
		t.Error("no-args output should contain Usage")
	}
}

// TestCLI_ItemsCreate_Help_ShowsAllFlags pins the flag set, the
// --timestamp help text and --hash-type's lack of a default. Mutants
// killed: --timestamp's help left as the ISO 8601 forms it once
// normalized, and --hash-type given the default sha256.
func TestCLI_ItemsCreate_Help_ShowsAllFlags(t *testing.T) {
	cmd := exec.Command(binaryPath, "items", "create", "--help")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("create --help failed: %s", err)
	}
	output := string(out)
	for _, flag := range []string{"--file", "--file-stdin", "--claims", "--claims-stdin", "--name", "--data-hash", "--hash-type", "--description", "--url", "--timestamp", "--metadata", "--location", "--visibility", "--tags", "--json"} {
		if !containsString(output, flag) {
			t.Errorf("help output missing flag %s", flag)
		}
	}
	const timestampHelp = "claimed timestamp: your own statement; Truestamp checks only that it is a well-formed date"
	if !containsString(output, timestampHelp) {
		t.Errorf("help output missing the --timestamp text %q:\n%s", timestampHelp, output)
	}
	if containsString(output, `(default "sha256")`) {
		t.Errorf("--hash-type shows a sha256 default:\n%s", output)
	}
}

// --- Local refusals, none of which sends a request ---

func TestCLI_ItemsCreate_NoAPIKey_Error(t *testing.T) {
	cmd := exec.Command(binaryPath, "items", "create", "-n", "Test", "--data-hash", validSHA256Hex)
	// Empty API key + a loopback base_url with no stored OAuth session ⇒
	// no credential ⇒ the "not authenticated" guard fires.
	cmd.Env = append(os.Environ(), "TRUESTAMP_API_KEY=", "TRUESTAMP_BASE_URL=http://127.0.0.1:0")
	out, _ := cmd.CombinedOutput()
	if !containsString(string(out), "Not authenticated") {
		t.Errorf("expected not-authenticated error, got: %s", out)
	}
}

// TestCLI_ItemsCreate_LocalRefusals pins each refusal the CLI still makes
// before sending: an unknown visibility, a --location that is not two JSON
// numbers, --metadata that is not JSON, a file that cannot be read, and a
// claims text the strict reader refuses (not JSON, or trailing content).
// Each names its fix and sends nothing. Mutant killed: the claims of a
// file read with a reader that stops after the first value (the trailing
// case is sent).
func TestCLI_ItemsCreate_LocalRefusals(t *testing.T) {
	notJSON := writeClaims(t, "not json")
	trailing := writeClaims(t, `{"name":"A"} {"name":"B"}`)
	for _, tc := range []struct {
		name  string
		stdin string
		args  []string
		want  string
	}{
		{"visibility", "", []string{"-n", "T", "-d", longDesc, "-v", "secret"}, "private, team, or public"},
		{"location shape", "", []string{"-n", "T", "-d", longDesc, "--location", "abc"}, "--location must be lat,lon"},
		{"location number", "", []string{"-n", "T", "-d", longDesc, "--location", "+37.7,1"}, "--location latitude: malformed_json (syntax)"},
		{"location not a number", "", []string{"-n", "T", "-d", longDesc, "--location", `"a",1`}, "--location latitude must be a JSON number"},
		{"metadata", "", []string{"-n", "T", "--metadata", "not json"}, "--metadata must be valid JSON"},
		{"missing file", "", []string{"/nonexistent/file.pdf"}, "cannot access"},
		{"directory", "", []string{os.TempDir()}, "is a directory, expected a file"},
		{"file flag", "", []string{"--file=/nonexistent/file.txt"}, "cannot access"},
		{"claims file not JSON", "", []string{"--claims=" + notJSON}, "parsing claims JSON: malformed_json (syntax)"},
		{"claims file trailing", "", []string{"--claims=" + trailing}, "parsing claims JSON: malformed_json (syntax)"},
		{"claims stdin not JSON", "not json", []string{"-C"}, "parsing claims JSON from stdin: malformed_json (syntax)"},
		{"claims stdin empty", "", []string{"-C"}, "no data received on stdin"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s := startCreateServer(t, http.StatusCreated, createdItemDoc)
			_, stderr, exit := runCreate(t, tc.stdin, append(tc.args, "--api-key", "fake", "--base-url", s.URL)...)
			if exit != 1 || !strings.Contains(stderr, tc.want) {
				t.Errorf("exit %d, stderr %q, want exit 1 and %q", exit, stderr, tc.want)
			}
			if n := s.hits.Load(); n != 0 {
				t.Errorf("%d requests sent, want 0", n)
			}
		})
	}
}

// TestCLI_ItemsCreate_RepeatedNameRefusedLocally: a member name twice in
// one object is refused before sending, from a claims file, from stdin and
// from --metadata, since the server's JSON:API reader would see one value.
// Mutant killed: the claims checked with encoding/json instead of the
// strict reader (the text is sent).
func TestCLI_ItemsCreate_RepeatedNameRefusedLocally(t *testing.T) {
	text := `{"name":"A","description":"` + longDesc + `","name":"B"}`
	path := writeClaims(t, text)
	for _, tc := range []struct {
		name  string
		stdin string
		args  []string
		want  string
	}{
		{"file", "", []string{"--claims=" + path}, "parsing claims JSON: malformed_json (duplicate_name)"},
		{"stdin", text, []string{"-C"}, "parsing claims JSON from stdin: malformed_json (duplicate_name)"},
		{"metadata", "", []string{"-n", "T", "--metadata", `{"k":1,"k":2}`}, "--metadata must be valid JSON: malformed_json (duplicate_name)"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s := startCreateServer(t, http.StatusCreated, createdItemDoc)
			_, stderr, exit := runCreate(t, tc.stdin, append(tc.args, "--api-key", "fake", "--base-url", s.URL)...)
			if exit != 1 || !strings.Contains(stderr, tc.want) || !strings.Contains(stderr, "a member name appears twice in one object") {
				t.Errorf("exit %d, stderr %q, want exit 1 and %q", exit, stderr, tc.want)
			}
			if n := s.hits.Load(); n != 0 {
				t.Errorf("%d requests sent, want 0", n)
			}
		})
	}
}

// TestCLI_ItemsCreate_J5NumberRefusedLocally walks MAX_INTEGER, the one
// bound of rule J5 (SPEC.md section 10.1), through every way a number
// reaches the claims: 2^53, and 9007199254740993.0, whose nearest binary64
// is in J5's integer gap, are refused before sending, and 2^53 - 1 is sent
// as written. Mutant killed: the claims checked with encoding/json instead
// of the strict reader (2^53 is sent).
func TestCLI_ItemsCreate_J5NumberRefusedLocally(t *testing.T) {
	refused := []struct {
		name  string
		stdin string
		args  func(t *testing.T) []string
		want  string
	}{
		{"file", "", func(t *testing.T) []string {
			return []string{"--claims=" + writeClaims(t, `{"name":"B","metadata":{"n":9007199254740992}}`)}
		}, "parsing claims JSON: malformed_json (number)"},
		{"file float", "", func(t *testing.T) []string {
			return []string{"--claims=" + writeClaims(t, `{"name":"B","metadata":{"n":9007199254740993.0}}`)}
		}, "parsing claims JSON: malformed_json (number)"},
		{"stdin", `{"name":"B","metadata":{"n":-9007199254740992}}`, func(*testing.T) []string { return []string{"-C"} },
			"parsing claims JSON from stdin: malformed_json (number)"},
		{"metadata", "", func(*testing.T) []string { return []string{"-n", "B", "--metadata", `{"n":9007199254740992}`} },
			"--metadata must be valid JSON: malformed_json (number)"},
		{"location", "", func(*testing.T) []string { return []string{"-n", "B", "--location", "9007199254740992,1"} },
			"--location latitude: malformed_json (number)"},
	}
	for _, tc := range refused {
		t.Run(tc.name, func(t *testing.T) {
			s := startCreateServer(t, http.StatusCreated, createdItemDoc)
			_, stderr, exit := runCreate(t, tc.stdin, append(tc.args(t), "--api-key", "fake", "--base-url", s.URL)...)
			if exit != 1 || !strings.Contains(stderr, tc.want) || !strings.Contains(stderr, "send it as a string") {
				t.Errorf("exit %d, stderr %q, want exit 1 and %q", exit, stderr, tc.want)
			}
			if n := s.hits.Load(); n != 0 {
				t.Errorf("%d requests sent, want 0", n)
			}
		})
	}

	s := startCreateServer(t, http.StatusCreated, createdItemDoc)
	text := `{"name":"B","metadata":{"n":9007199254740991,"m":-9007199254740991}}`
	if _, stderr, exit := runCreate(t, "", "--claims="+writeClaims(t, text), "--api-key", "fake", "--base-url", s.URL); exit != 0 {
		t.Fatalf("2^53 - 1: exit %d, stderr %q", exit, stderr)
	}
	if got := s.sentClaims(t); got != text {
		t.Errorf("claims sent %s, want %s", got, text)
	}
}

// TestCLI_ItemsCreate_ClaimFlagsRefusedWithClaims: every claim flag is
// refused beside a claims file or claims on stdin, naming the flag, with
// nothing sent. Mutants killed: refuseClaimFlags removed (the flag
// rewrites the claims), and any one flag left out of claimFlags (its case
// is sent).
func TestCLI_ItemsCreate_ClaimFlagsRefusedWithClaims(t *testing.T) {
	text := `{"name":"Base","description":"` + longDesc + `"}`
	path := writeClaims(t, text)
	values := map[string]string{
		"name": "X", "data-hash": validSHA256Hex, "hash-type": "sha256", "description": longDesc,
		"url": "https://example.com", "timestamp": "2025", "location": "1,2", "metadata": `{"k":"v"}`,
	}
	for _, flag := range []string{"name", "data-hash", "hash-type", "description", "url", "timestamp", "location", "metadata"} {
		for _, source := range []string{"file", "stdin"} {
			t.Run(flag+" "+source, func(t *testing.T) {
				s := startCreateServer(t, http.StatusCreated, createdItemDoc)
				args := []string{"--" + flag, values[flag], "--api-key", "fake", "--base-url", s.URL}
				stdin := ""
				if source == "file" {
					args = append([]string{"--claims=" + path}, args...)
				} else {
					args = append([]string{"-C"}, args...)
					stdin = text
				}
				_, stderr, exit := runCreate(t, stdin, args...)
				want := "--" + flag + " cannot be combined with --claims or --claims-stdin"
				if exit != 1 || !strings.Contains(stderr, want) {
					t.Errorf("exit %d, stderr %q, want exit 1 and %q", exit, stderr, want)
				}
				if n := s.hits.Load(); n != 0 {
					t.Errorf("%d requests sent, want 0", n)
				}
			})
		}
	}
}

// --- What reaches the server ---

// TestCLI_ItemsCreate_ClaimsSentByteForByte: a claims file and claims on
// stdin reach the server exactly as given, with a leading space in name,
// the text's whitespace, member order, escapes and number spellings, and
// the item attributes beside them. Mutants killed: the claims decoded and
// re-encoded before sending (the whitespace, the order and 1.50e300 are
// rewritten), and name trimmed (" Big <a>" is sent as "Big <a>").
func TestCLI_ItemsCreate_ClaimsSentByteForByte(t *testing.T) {
	text := "{ \"name\" : \" Big <a>\",\n\t\"metadata\":{\"z\":1.50e300,\"a\":\"\\u00e9\",\"e\":1e21}, \"description\":\"" + longDesc + "\" }\n"
	for _, source := range []string{"file", "stdin"} {
		t.Run(source, func(t *testing.T) {
			s := startCreateServer(t, http.StatusCreated, createdItemDoc)
			args := []string{"-v", "public", "--tags", "a,b", "--api-key", "fake", "--base-url", s.URL}
			stdin := ""
			if source == "file" {
				args = append([]string{"--claims=" + writeClaims(t, text)}, args...)
			} else {
				args = append([]string{"-C"}, args...)
				stdin = text
			}
			if _, stderr, exit := runCreate(t, stdin, args...); exit != 0 {
				t.Fatalf("exit %d, stderr %q", exit, stderr)
			}
			want := `{"data":{"type":"item","attributes":{"claims":` + text + `,"visibility":"public","tags":["a","b"]}}}`
			if got := s.body(); got != want {
				t.Errorf("request body\n got: %q\nwant: %q", got, want)
			}
		})
	}
}

// TestCLI_ItemsCreate_ClaimsFromFlags pins the claims text the claim flags
// build, member by member, each value exactly as typed: a leading space in
// name, an uppercase hash, a --timestamp with an offset, with a fraction
// and Z, and with +00:00, a --url and a --hash-type with spaces and
// capitals, a blank description, the --location numbers as typed and
// --metadata's own text. Mutants killed: name trimmed, the hash
// lowercased, the timestamp normalized to UTC ("2025-01-15T14:30:00Z" is
// sent), --metadata re-encoded (its space and member order are lost), a
// --timestamp that parses as RFC 3339 written again in that form (the
// fraction is dropped and +00:00 sent as Z), --url and --hash-type trimmed
// and lowercased, and a flag whose value is only whitespace left out of
// the claims.
func TestCLI_ItemsCreate_ClaimsFromFlags(t *testing.T) {
	upper := "ABCDEF" + strings.Repeat("0", 58)
	for _, tc := range []struct {
		name string
		args []string
		want string
	}{
		{"every flag", []string{
			"-n", " Doc", "--data-hash", upper, "--hash-type", "sha256", "-d", "a \"quoted\" line\n",
			"--url", "https://example.com/x", "--timestamp", "2025-01-15T09:30:00-05:00",
			"--location", "37.7749, -122.4194", "--metadata", `{"b":1, "a":[2.50]}`},
			`{"name":" Doc","hash":"` + upper + `","hash_type":"sha256","description":"a \"quoted\" line\u000a",` +
				`"url":"https://example.com/x","timestamp":"2025-01-15T09:30:00-05:00",` +
				`"location":{"latitude":37.7749,"longitude": -122.4194},"metadata":{"b":1, "a":[2.50]}}`},
		{"timestamp fraction and Z", []string{"-n", "Doc", "--timestamp", "2025-01-15T14:30:00.500Z"},
			`{"name":"Doc","timestamp":"2025-01-15T14:30:00.500Z"}`},
		{"timestamp +00:00", []string{"-n", "Doc", "--timestamp", "2025-01-15T09:30:00+00:00"},
			`{"name":"Doc","timestamp":"2025-01-15T09:30:00+00:00"}`},
		{"url and hash type as typed", []string{"-n", "Doc", "--data-hash", validSHA256Hex, "--hash-type", " SHA256", "--url", " https://Example.com/X "},
			`{"name":"Doc","hash":"` + validSHA256Hex + `","hash_type":" SHA256","url":" https://Example.com/X "}`},
		{"blank description", []string{"-n", "Doc", "-d", " "},
			`{"name":"Doc","description":" "}`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s := startCreateServer(t, http.StatusCreated, createdItemDoc)
			_, stderr, exit := runCreate(t, "", append(tc.args, "--api-key", "fake", "--base-url", s.URL)...)
			if exit != 0 {
				t.Fatalf("exit %d, stderr %q", exit, stderr)
			}
			if got := s.sentClaims(t); got != tc.want {
				t.Errorf("claims sent\n got: %s\nwant: %s", got, tc.want)
			}
		})
	}
}

// TestCLI_ItemsCreate_BareHashSendsNoHashType: --data-hash alone sends a
// hash and no hash_type, leaving the half pair for the server to refuse.
// Mutant killed: the sha256 default for hash_type restored (as the flag's
// default or filled in beside a bare hash).
func TestCLI_ItemsCreate_BareHashSendsNoHashType(t *testing.T) {
	s := startCreateServer(t, http.StatusCreated, createdItemDoc)
	if _, stderr, exit := runCreate(t, "", "-n", "Doc", "--data-hash", validSHA256Hex, "--api-key", "fake", "--base-url", s.URL); exit != 0 {
		t.Fatalf("exit %d, stderr %q", exit, stderr)
	}
	if got, want := s.sentClaims(t), `{"name":"Doc","hash":"`+validSHA256Hex+`"}`; got != want {
		t.Errorf("claims sent %s, want %s", got, want)
	}
}

// TestCLI_ItemsCreate_HashedFileClaims: a hashed file gives the hash, the
// type sha256 and the file's name, and --name replaces that name in its
// place; --file-stdin gives the hash and type and takes --name. Mutant
// killed: claimsBuilder.set appending instead of replacing (the claims
// carry "name" twice).
func TestCLI_ItemsCreate_HashedFileClaims(t *testing.T) {
	content := []byte("test content for hash verification\n")
	path := filepath.Join(t.TempDir(), "hashtest.txt")
	if err := os.WriteFile(path, content, 0o600); err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256(content)
	digest := hex.EncodeToString(sum[:])

	for _, tc := range []struct {
		name  string
		stdin string
		args  []string
		want  string
	}{
		{"positional", "", []string{path},
			`{"hash":"` + digest + `","hash_type":"sha256","name":"hashtest.txt"}`},
		{"name override", "", []string{"--file=" + path, "-n", "Q1 Report"},
			`{"hash":"` + digest + `","hash_type":"sha256","name":"Q1 Report"}`},
		{"stdin", string(content), []string{"-F", "-n", "Stdin File"},
			`{"hash":"` + digest + `","hash_type":"sha256","name":"Stdin File"}`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s := startCreateServer(t, http.StatusCreated, createdItemDoc)
			if _, stderr, exit := runCreate(t, tc.stdin, append(tc.args, "--api-key", "fake", "--base-url", s.URL)...); exit != 0 {
				t.Fatalf("exit %d, stderr %q", exit, stderr)
			}
			if got := s.sentClaims(t); got != tc.want {
				t.Errorf("claims sent %s, want %s", got, tc.want)
			}
		})
	}
}

// TestCLI_ItemsCreate_HashFlagsRefusedWithHashedFile: --data-hash and
// --hash-type are refused beside a hashed file, a file argument, --file or
// --file-stdin, naming the flag, with nothing sent: the file supplies the
// hash and its type. Mutant killed: refuseHashFlags removed (the file's
// digest is sent under the typed hash_type, or the typed hash under the
// file's sha256).
func TestCLI_ItemsCreate_HashFlagsRefusedWithHashedFile(t *testing.T) {
	content := "content to hash\n"
	path := filepath.Join(t.TempDir(), "hashed.txt")
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	values := map[string]string{"data-hash": strings.Repeat("ab", 32), "hash-type": "sha512"}
	for _, flag := range []string{"data-hash", "hash-type"} {
		for _, source := range []struct {
			name, stdin string
			args        []string
		}{
			{"argument", "", []string{path}},
			{"file flag", "", []string{"--file=" + path}},
			{"file stdin", content, []string{"-F", "-n", "Doc"}},
		} {
			t.Run(flag+" "+source.name, func(t *testing.T) {
				s := startCreateServer(t, http.StatusCreated, createdItemDoc)
				args := append(append([]string{}, source.args...), "--"+flag, values[flag], "--api-key", "fake", "--base-url", s.URL)
				_, stderr, exit := runCreate(t, source.stdin, args...)
				want := "--" + flag + " cannot be combined with a hashed file (a file argument, --file or --file-stdin): " +
					"the file supplies the hash and its hash_type, sha256"
				if exit != 1 || !strings.Contains(stderr, want) {
					t.Errorf("exit %d, stderr %q, want exit 1 and %q", exit, stderr, want)
				}
				if n := s.hits.Load(); n != 0 {
					t.Errorf("%d requests sent, want 0", n)
				}
			})
		}
	}
}

// TestCLI_ItemsCreate_LongNumberRefusedLocally: a number literal longer
// than 1,024 octets, which the service's request reader refuses whatever
// its value, is refused before sending, from a claims file, from stdin and
// from --metadata, and one of exactly 1,024 octets is sent as written.
// Mutants killed: checkNumberLengths not called (the claims are sent), and
// the limit compared with >= (the 1,024-octet number is refused).
func TestCLI_ItemsCreate_LongNumberRefusedLocally(t *testing.T) {
	over := `{"name":"Doc","metadata":{"n":1.` + strings.Repeat("0", 1023) + `}}`
	want := "the claims hold a number of 1025 characters at octet 30: the Truestamp service reads no request " +
		"holding a number longer than 1024 characters (its request reader, TruestampWeb.RequestJSON, holds every body " +
		"to JSONNumbers.max_octets), a limit on what can be sent, not a rule about the value; send it as a string"
	for _, tc := range []struct {
		name, stdin string
		args        []string
	}{
		{"file", "", []string{"--claims=" + writeClaims(t, over)}},
		{"stdin", over, []string{"-C"}},
		{"metadata", "", []string{"-n", "Doc", "--metadata", `{"n":1.` + strings.Repeat("0", 1023) + `}`}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s := startCreateServer(t, http.StatusCreated, createdItemDoc)
			_, stderr, exit := runCreate(t, tc.stdin, append(tc.args, "--api-key", "fake", "--base-url", s.URL)...)
			if exit != 1 || !strings.Contains(stderr, want) {
				t.Errorf("exit %d, stderr %q, want exit 1 and %q", exit, stderr, want)
			}
			if n := s.hits.Load(); n != 0 {
				t.Errorf("%d requests sent, want 0", n)
			}
		})
	}

	s := startCreateServer(t, http.StatusCreated, createdItemDoc)
	at := `{"name":"Doc","metadata":{"n":1.` + strings.Repeat("0", 1022) + `}}`
	if _, stderr, exit := runCreate(t, "", "--claims="+writeClaims(t, at), "--api-key", "fake", "--base-url", s.URL); exit != 0 {
		t.Fatalf("1,024 octets: exit %d, stderr %q", exit, stderr)
	}
	if got := s.sentClaims(t); got != at {
		t.Errorf("claims sent %s, want %s", got, at)
	}
}

// TestCLI_ItemsCreate_OversizedClaimsFileRefused: a claims file over
// MAX_READ_OCTETS is refused as input_too_large, read no further than
// MAX_READ_OCTETS + 1 octets, with nothing sent. Mutant killed: the file
// read whole with os.ReadFile and then parsed (it is sent).
func TestCLI_ItemsCreate_OversizedClaimsFileRefused(t *testing.T) {
	head := `{"name":"Doc","description":"` + longDesc + `","metadata":{"pad":"`
	tail := `"}}`
	text := head + strings.Repeat("x", strictjson.MaxReadOctets+1-len(head)-len(tail)) + tail
	if len(text) != strictjson.MaxReadOctets+1 {
		t.Fatalf("built %d octets", len(text))
	}
	s := startCreateServer(t, http.StatusCreated, createdItemDoc)
	_, stderr, exit := runCreate(t, "", "--claims="+writeClaims(t, text), "--api-key", "fake", "--base-url", s.URL)
	want := fmt.Sprintf("reading claims file: input_too_large: the input is over %d octets", strictjson.MaxReadOctets)
	if exit != 1 || !strings.Contains(stderr, want) {
		t.Errorf("exit %d, stderr %q, want exit 1 and %q", exit, stderr, want)
	}
	if n := s.hits.Load(); n != 0 {
		t.Errorf("%d requests sent, want 0", n)
	}
}

// TestCLI_ItemsCreate_TimestampGrammarRefusedLocally: a --timestamp off
// the grammar, or naming a day that does not exist, is refused before
// sending. Mutant killed: checkTimestamp not called (the value is sent).
func TestCLI_ItemsCreate_TimestampGrammarRefusedLocally(t *testing.T) {
	for _, tc := range []struct{ in, want string }{
		{"2025-02-30", "--timestamp names 2025-02-30, a date that does not exist"},
		{"15 Jan 2025", `--timestamp "15 Jan 2025" is not a date or time in an accepted form`},
	} {
		s := startCreateServer(t, http.StatusCreated, createdItemDoc)
		_, stderr, exit := runCreate(t, "", "-n", "Doc", "-d", longDesc, "--timestamp", tc.in, "--api-key", "fake", "--base-url", s.URL)
		if exit != 1 || !strings.Contains(stderr, tc.want) {
			t.Errorf("%q: exit %d, stderr %q, want %q", tc.in, exit, stderr, tc.want)
		}
		if n := s.hits.Load(); n != 0 {
			t.Errorf("%q: %d requests sent, want 0", tc.in, n)
		}
	}
}

// --- What the server answers ---

// claimsRefusalDoc is a 422 as the server writes a claims refusal.
const claimsRefusalDoc = `{"errors":[` +
	`{"code":"claims_surrounding_whitespace","title":"Invalid claims","detail":"/name begins with a space (U+0020). Remove it.","source":{"pointer":"/data/attributes/claims/name"}},` +
	`{"code":"claims_hash_pair","title":"Invalid claims","detail":"The claims have hash without hash_type. Add hash_type.","source":{"pointer":"/data/attributes/claims"}}]}`

// TestCLI_ItemsCreate_ClaimsReasonsPrinted: each broken rule the server
// names is printed with its pointer into the claims, its code and its
// message, in the server's order, and the run exits 1. Mutants killed:
// only the first reason printed, and the pointer left out of the line.
func TestCLI_ItemsCreate_ClaimsReasonsPrinted(t *testing.T) {
	s := startCreateServer(t, http.StatusUnprocessableEntity, claimsRefusalDoc)
	_, stderr, exit := runCreate(t, "", "-n", " Doc", "--data-hash", validSHA256Hex, "--api-key", "fake", "--base-url", s.URL)
	if exit != 1 {
		t.Fatalf("exit %d, want 1; stderr %q", exit, stderr)
	}
	first := `pointer "/name", claims_surrounding_whitespace: /name begins with a space (U+0020). Remove it.`
	second := `pointer "", claims_hash_pair: The claims have hash without hash_type. Add hash_type.`
	i, j := strings.Index(stderr, first), strings.Index(stderr, second)
	if i < 0 || j < 0 || j < i {
		t.Errorf("stderr should list both reasons in order:\n%s", stderr)
	}
	for _, want := range []string{"Claims refused", "Nothing was created. The server answered HTTP 422."} {
		if !strings.Contains(stderr, want) {
			t.Errorf("stderr missing %q:\n%s", want, stderr)
		}
	}
}

// TestCLI_ItemsCreate_ClaimsReasonsJSON: under --json the refusal is one
// object on stdout carrying every reason with its code, message and
// pointer, a whole-claims reason's pointer written as "". Mutants killed:
// the --json branch printing the English lines on stderr instead (stdout
// is empty), and ClaimsReason's pointer tagged omitempty (the key is left
// out).
func TestCLI_ItemsCreate_ClaimsReasonsJSON(t *testing.T) {
	s := startCreateServer(t, http.StatusUnprocessableEntity, claimsRefusalDoc)
	stdout, _, exit := runCreate(t, "", "-n", " Doc", "--data-hash", validSHA256Hex, "--json", "--api-key", "fake", "--base-url", s.URL)
	if exit != 1 {
		t.Fatalf("exit %d, want 1", exit)
	}
	var got struct {
		Error   string               `json:"error"`
		Message string               `json:"message"`
		Reasons []items.ClaimsReason `json:"claims_reasons"`
	}
	if err := json.Unmarshal([]byte(stdout), &got); err != nil {
		t.Fatalf("stdout is not JSON: %v\n%s", err, stdout)
	}
	want := []items.ClaimsReason{
		{Code: "claims_surrounding_whitespace", Message: "/name begins with a space (U+0020). Remove it.", Pointer: "/name"},
		{Code: "claims_hash_pair", Message: "The claims have hash without hash_type. Add hash_type.", Pointer: ""},
	}
	if got.Error != "claims_refused" || got.Message != "the server refused the claims" || !reflect.DeepEqual(got.Reasons, want) {
		t.Errorf("got %+v", got)
	}
	// A whole-claims reason carries the key "pointer" with the value "",
	// so a script can tell it from a reason with no pointer.
	var raw struct {
		Reasons []map[string]any `json:"claims_reasons"`
	}
	if err := json.Unmarshal([]byte(stdout), &raw); err != nil {
		t.Fatal(err)
	}
	if len(raw.Reasons) != 2 {
		t.Fatalf("claims_reasons %v, want 2", raw.Reasons)
	}
	if pointer, ok := raw.Reasons[1]["pointer"]; !ok || pointer != "" {
		t.Errorf(`claims_reasons[1] %v, want "pointer": ""`, raw.Reasons[1])
	}
}

// TestCLI_ItemsCreate_ItemCodeUnavailableIsRetryable: the 409 the server
// sends when it could not issue an item code is told as nothing created
// and safe to run again, in text and under --json. Mutant killed:
// explainCreateError not handling ErrItemCodeUnavailable (the run prints
// a bare "HTTP 409" error).
func TestCLI_ItemsCreate_ItemCodeUnavailableIsRetryable(t *testing.T) {
	const conflict = `{"errors":[{"code":"item_code_unavailable","title":"Conflict","detail":"Could not issue an item code. Try again."}]}`
	s := startCreateServer(t, http.StatusConflict, conflict)
	_, stderr, exit := runCreate(t, "", "-n", "Doc", "-d", longDesc, "--api-key", "fake", "--base-url", s.URL)
	if exit != 1 {
		t.Fatalf("exit %d, want 1", exit)
	}
	for _, want := range []string{
		"Item not created yet",
		"Could not issue an item code. Try again.",
		"Nothing was created. This clears on its own: run the same command again.",
		"The server said item_code_unavailable (HTTP 409).",
	} {
		if !strings.Contains(stderr, want) {
			t.Errorf("stderr missing %q:\n%s", want, stderr)
		}
	}

	stdout, _, exit := runCreate(t, "", "-n", "Doc", "-d", longDesc, "--json", "--api-key", "fake", "--base-url", s.URL)
	var got map[string]any
	if err := json.Unmarshal([]byte(stdout), &got); err != nil || exit != 1 {
		t.Fatalf("exit %d, stdout %q: %v", exit, stdout, err)
	}
	want := map[string]any{"error": "item_code_unavailable", "message": "Could not issue an item code. Try again.", "retryable": true}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

// TestCLI_ItemsCreate_PrintsClaimsJCSAndHash: a created item's claims_jcs
// and claims_hash are printed whole, in the card and under --json.
// Mutants killed: the Claims JCS and Claims hash rows left out of the
// card, and claims_jcs and claims_hash left out of the JSON.
func TestCLI_ItemsCreate_PrintsClaimsJCSAndHash(t *testing.T) {
	const jcs = `{"hash":"ab","name":"Echo"}`
	const hash = "5d0c6ffbd0e4e3e3b1b1c8e8b0b2d7e3a7f6c9a5b4d3c2e1f0a9b8c7d6e5f4a3"
	s := startCreateServer(t, http.StatusCreated, createdItemDoc)

	stdout, stderr, exit := runCreate(t, "", "-n", "Echo", "-d", longDesc, "--api-key", "fake", "--base-url", s.URL)
	if exit != 0 {
		t.Fatalf("exit %d, stderr %q", exit, stderr)
	}
	for _, want := range []string{"Claims JCS", jcs, "Claims hash", hash} {
		if !strings.Contains(stdout, want) {
			t.Errorf("card missing %q:\n%s", want, stdout)
		}
	}

	stdout, stderr, exit = runCreate(t, "", "-n", "Echo", "-d", longDesc, "--json", "--api-key", "fake", "--base-url", s.URL)
	if exit != 0 {
		t.Fatalf("exit %d, stderr %q", exit, stderr)
	}
	var got map[string]any
	if err := json.Unmarshal([]byte(stdout), &got); err != nil {
		t.Fatalf("stdout is not JSON: %v\n%s", err, stdout)
	}
	if got["claims_jcs"] != jcs || got["claims_hash"] != hash {
		t.Errorf("claims_jcs %v, claims_hash %v", got["claims_jcs"], got["claims_hash"])
	}
}

// TestCLI_ItemsCreate_JSONOutput_ClaimsOnly_OmitsHashKeys exercises
// printCreateJSON on a claims-only response (no hash or hash_type) and on
// an external-hash one.
func TestCLI_ItemsCreate_JSONOutput_ClaimsOnly_OmitsHashKeys(t *testing.T) {
	capture := func(resp *items.CreateItemResponse) map[string]any {
		var buf bytes.Buffer
		if err := printCreateJSON(&buf, resp); err != nil {
			t.Fatal(err)
		}
		var got map[string]any
		if err := json.Unmarshal(buf.Bytes(), &got); err != nil {
			t.Fatal(err)
		}
		return got
	}

	got := capture(&items.CreateItemResponse{ID: "01CLAIMSONLY", Name: "Invention", Visibility: "private", TeamID: "team_xyz", ClaimsJCS: `{"name":"Invention"}`, ClaimsHash: "ch"})
	want := map[string]any{"id": "01CLAIMSONLY", "name": "Invention", "visibility": "private", "team_id": "team_xyz", "claims_jcs": `{"name":"Invention"}`, "claims_hash": "ch"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("claims-only JSON %v, want %v", got, want)
	}

	got = capture(&items.CreateItemResponse{ID: "01EXT", Name: "Doc", Hash: validSHA256Hex, HashType: "sha256", Visibility: "private", TeamID: "team_xyz"})
	if got["hash"] != validSHA256Hex || got["hash_type"] != "sha256" {
		t.Errorf("external-hash JSON output should carry hash and hash_type, got: %v", got)
	}
}

// TestSetStringRefusesInvalidUTF8: a claim flag's value that is not valid
// UTF-8 is refused, as no JSON text holds it unchanged. Mutant killed: the
// utf8.ValidString check removed (the value is written).
func TestSetStringRefusesInvalidUTF8(t *testing.T) {
	var b claimsBuilder
	if err := b.setString("name", "bad\xff", "--name"); err == nil || err.Error() != "--name is not valid UTF-8, so no JSON text can hold it as given" {
		t.Errorf("invalid UTF-8: %v", err)
	}
}

// --- NoOptDefVal space-form guard -------------------------------------
//
// `--claims`, `--file` and their shorthands carry a pflag NoOptDefVal, so
// `--claims doc.json` does NOT bind doc.json to the flag: the flag takes
// its picker sentinel and doc.json is left as a positional argument.
// Untrapped that opens the interactive picker and discards the path,
// which hangs a script and misleads a human. These pin the targeted error
// instead, and pin that the legitimate spellings still work.

func TestCLI_ItemsCreate_SpaceSeparatedPathIsRejected(t *testing.T) {
	dir := t.TempDir()
	claims := filepath.Join(dir, "claims.json")
	if err := os.WriteFile(claims, []byte(`{"name":"T","description":"a description that is definitely longer than thirty two chars"}`), 0600); err != nil {
		t.Fatal(err)
	}
	doc := filepath.Join(dir, "doc.txt")
	if err := os.WriteFile(doc, []byte("hello"), 0600); err != nil {
		t.Fatal(err)
	}

	cases := []struct {
		name string
		args []string
		want string
	}{
		{"long claims", []string{"items", "create", "--claims", claims}, "--claims=" + claims},
		{"short claims", []string{"items", "create", "-c", claims}, "--claims=" + claims},
		{"long file", []string{"items", "create", "--file", doc}, "--file=" + doc},
		{"short file", []string{"items", "create", "-f", doc}, "--file=" + doc},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cmd := exec.Command(binaryPath, append(tc.args, "--api-key", "fake", "--base-url", "http://127.0.0.1:9")...)
			out, err := cmd.CombinedOutput()
			if err == nil {
				t.Fatalf("expected a non-zero exit, got success: %s", out)
			}
			// The suggestion must name the exact working spelling, so the
			// user can copy it rather than infer the rule.
			if !containsString(string(out), tc.want) {
				t.Errorf("error should suggest %q, got: %s", tc.want, out)
			}
			// It must never have reached the picker.
			if containsString(string(out), "file selection") || containsString(string(out), "TTY") {
				t.Errorf("the picker was opened instead of erroring: %s", out)
			}
		})
	}
}

func TestCLI_ItemsCreate_EqualsFormAndPositionalStillWork(t *testing.T) {
	dir := t.TempDir()
	claims := filepath.Join(dir, "claims.json")
	if err := os.WriteFile(claims, []byte(`{"name":"T","description":"a description that is definitely longer than thirty two chars"}`), 0600); err != nil {
		t.Fatal(err)
	}
	doc := filepath.Join(dir, "doc.txt")
	if err := os.WriteFile(doc, []byte("hello"), 0600); err != nil {
		t.Fatal(err)
	}

	// Each of these must get past input resolution and fail at the network
	// instead, proving the guard did not swallow a legitimate spelling.
	cases := [][]string{
		{"items", "create", doc},
		{"items", "create", doc, "-n", "Q1", "-v", "public"},
		{"items", "create", "--file=" + doc},
		{"items", "create", "--claims=" + claims},
		{"items", "create", "-c=" + claims},
	}
	for _, args := range cases {
		t.Run(strings.Join(args[2:], " "), func(t *testing.T) {
			cmd := exec.Command(binaryPath, append(args, "--api-key", "fake", "--base-url", "http://127.0.0.1:9")...)
			out, _ := cmd.CombinedOutput()
			if !containsString(string(out), "API request failed") {
				t.Errorf("expected to reach the API, got: %s", out)
			}
		})
	}
}
