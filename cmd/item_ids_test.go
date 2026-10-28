// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"slices"
	"strings"
	"sync"
	"testing"

	"github.com/truestamp/truestamp-cli/internal/proof"
)

// testItemIDUpper is testItemID typed in uppercase, which every command
// sends in its canonical lowercase form.
const testItemIDUpper = "01A0DB66-C2E0-78A4-B8CC-12B11BC97493"

// validateTypeVsShape takes, for --type item, a UUIDv7, an item code and
// an item_hash, the three references the app's raw proof route reads for
// an item, and for every other type a UUIDv7 only, naming what it got.
// Mutant killed: validateTypeVsShape taking every shape for every type
// (the code and hash rows of the five other types get nil).
func TestValidateTypeVsShape(t *testing.T) {
	const prefix = " requires a UUIDv7 id (e.g. 019d6a32-13e6-72b0-97e5-3779231ea97b); got "
	for _, typ := range proofTypeValues {
		for _, tc := range []struct {
			shape proof.IDType
			want  string
		}{
			{proof.IDTypeUUIDv7, ""},
			{proof.IDTypeItemCode, "--type " + typ + prefix + "an item code, which names only an item"},
			{proof.IDTypeItemHash, "--type " + typ + prefix + "64 hex characters, an item_hash, which names only an item"},
		} {
			if typ == proofTypeItem {
				tc.want = ""
			}
			err := validateTypeVsShape(typ, tc.shape)
			switch {
			case tc.want == "" && err != nil:
				t.Errorf("validateTypeVsShape(%s, %s) = %v; want nil", typ, tc.shape, err)
			case tc.want != "" && (err == nil || err.Error() != tc.want):
				t.Errorf("validateTypeVsShape(%s, %s) = %v; want %q", typ, tc.shape, err, tc.want)
			}
		}
	}
}

// appStandIn is a loopback stand-in for the two app routes `proofs get`
// reads: POST /api/json/utilities/resolve-id, which it answers with
// resolution, and the raw proof route, which refuses as the app does
// (standInRefusal) and otherwise serves served. It records the body of
// every resolve-id request and the path of every other request.
func appStandIn(t *testing.T, resolution string, served []byte) (string, func() []string, func() []string) {
	t.Helper()
	var mu sync.Mutex
	var resolves, others []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		mu.Lock()
		defer mu.Unlock()
		if r.Method == http.MethodPost && r.URL.Path == "/api/json/utilities/resolve-id" {
			resolves = append(resolves, string(body))
			w.Header().Set("Content-Type", "application/vnd.api+json")
			_, _ = w.Write([]byte(resolution))
			return
		}
		others = append(others, r.URL.Path)
		rest, ok := strings.CutPrefix(r.URL.Path, rawProofPrefix)
		parts := strings.Split(rest, "/")
		if !ok || len(parts) != 2 || r.Method != http.MethodGet {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		if refusal := standInRefusal(parts[0], parts[1]); refusal != nil {
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write(refusal)
			return
		}
		_, _ = w.Write(served)
	}))
	t.Cleanup(srv.Close)
	return srv.URL,
		func() []string { mu.Lock(); defer mu.Unlock(); return slices.Clone(resolves) },
		func() []string { mu.Lock(); defer mu.Unlock(); return slices.Clone(others) }
}

// itemResolution is the app's answer to resolve-id for the sound item's
// UUIDv7 (Truestamp.AI.IdResolver.resolve/2 and its item_match/1, under
// the JSON:API route's wrap_in_result?): one match, of kind item, whose
// proof_type is item.
const itemResolution = `{"result":{"id":"` + testItemID + `","normalized_id":"` + testItemID + `","id_format":"uuidv7","matches":[` +
	`{"kind":"item","verifiable":true,"proof_type":"item","also_verifiable_as":[],` +
	`"summary":{"id":"` + testItemID + `","item_code":"` + soundItemCode + `","state":"committed","visibility":"private","inserted_at":"2026-09-26T01:49:00.000000Z"}}]}}`

// `proofs get <uuidv7>` with no --type asks resolve-id what the UUIDv7 is,
// sending it in its canonical lowercase form however it was typed, and
// fetches the item's proof the app's answer names, by that same form: one
// resolve-id request, one raw GET, and the served bytes on stdout.
// Mutants killed: the id sent as typed (the resolve-id body and the raw
// path hold the uppercase text, which the stand-in refuses); a UUIDv7
// taken as an item with no round trip (no resolve-id request).
func TestCLI_ProofsGet_ResolvesAnItemUUIDv7(t *testing.T) {
	item, err := os.ReadFile(soundItem())
	if err != nil {
		t.Fatal(err)
	}
	for _, typed := range []string{testItemID, testItemIDUpper} {
		url, resolves, others := appStandIn(t, itemResolution, item)
		stdout, stderr, exit := runCLI(t, "--base-url", url, "--api-key", "k", "proofs", "get", typed)
		if exit != 0 || stdout != string(item) {
			t.Fatalf("%s: exit %d, %d bytes on stdout; want exit 0 and the %d served\n%s", typed, exit, len(stdout), len(item), stderr)
		}
		if got, want := resolves(), []string{`{"data":{"id":"` + testItemID + `"}}`}; !slices.Equal(got, want) {
			t.Errorf("%s: resolve-id bodies %q, want %q", typed, got, want)
		}
		if got, want := others(), []string{rawProofPrefix + "item/" + testItemID}; !slices.Equal(got, want) {
			t.Errorf("%s: other requests %q, want %q", typed, got, want)
		}
	}
}

// The raw proof route's stand-in refuses what the app refuses, with the
// app's answer: an item id that is no item reference, 26 characters of
// Crockford base32 here, and a non-UUIDv7 for any other type are 400
// id_format_mismatch, and the sound item's UUIDv7 is served. Mutant
// killed: the stand-in serving any id (a 200 for each).
func TestCLI_RawProofStandIn_RefusesWhatTheAppRefuses(t *testing.T) {
	url, _ := rawProofServer(t, byMode)
	for _, tc := range []struct {
		path   string
		status int
		body   string
	}{
		{"item/01HJHB01T8FYZ7YTR9P5N62K5B", http.StatusBadRequest,
			`{"errors":[{"code":"invalid","status":"400","detail":"type=item needs an item id (UUIDv7), item code or item_hash","meta":{"code":"id_format_mismatch","requested_type":"item"}}]}`},
		{"block/" + soundItemCode, http.StatusBadRequest,
			`{"errors":[{"code":"invalid","status":"400","detail":"ID is not a valid UUIDv7 for type=block","meta":{"code":"id_format_mismatch","requested_type":"block"}}]}`},
		{"item/" + testItemID, http.StatusOK, string(servedBytes(""))},
	} {
		resp, err := http.Get(url + rawProofPrefix + tc.path)
		if err != nil {
			t.Fatal(err)
		}
		body, _ := io.ReadAll(resp.Body)
		_ = resp.Body.Close()
		if resp.StatusCode != tc.status || string(body) != tc.body {
			t.Errorf("%s: %d %s; want %d %s", tc.path, resp.StatusCode, body, tc.status, tc.body)
		}
	}
}

// countingServer answers every request 404 and counts them.
func countingServer(t *testing.T) (string, func() int) {
	t.Helper()
	var mu sync.Mutex
	var n int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		mu.Lock()
		n++
		mu.Unlock()
		w.WriteHeader(http.StatusNotFound)
	}))
	t.Cleanup(srv.Close)
	return srv.URL, func() int { mu.Lock(); defer mu.Unlock(); return n }
}

// Every command that takes an item id reads it exactly as typed: a space,
// tab or line feed around a UUIDv7 is refused, never trimmed, and so are a
// UUID's 32-hex, braced and urn: forms, a UUID of another version and 26
// characters of Crockford base32, each with the value quoted as given and
// before any request, and each refusal names only the form its command
// takes: proofs get, items get and items update a UUIDv7 id, an item code
// and an item_hash, and convert id a UUIDv7. Mutants killed: a
// strings.TrimSpace on the argument of items get, items update, proofs get
// or convert id (the padded values are sent, or convert id prints a time);
// a UUIDv7 read in the forms gofrs FromString takes, 32 hex characters,
// braces and urn:uuid: (those values are sent, or convert id prints a
// time); a refusal naming a form its command does not take, or leaving
// out one it does (the message differs).
func TestCLI_ItemIDsAreReadExactlyAsTyped(t *testing.T) {
	cases := map[string]string{
		" " + testItemID:                       `invalid ID format " ` + testItemID + `"`,
		testItemID + " ":                       `invalid ID format "` + testItemID + ` "`,
		testItemID + "\n":                      `invalid ID format "` + testItemID + `\n"`,
		"\t" + testItemID:                      `invalid ID format "\t` + testItemID + `"`,
		"{" + testItemID + "}":                 `invalid ID format "{` + testItemID + `}"`,
		"urn:uuid:" + testItemID:               `invalid ID format "urn:uuid:` + testItemID + `"`,
		"01a0db66c2e078a4b8cc12b11bc97493":     `"01a0db66c2e078a4b8cc12b11bc97493" is 32 hex characters, which is no id`,
		"550e8400-e29b-41d4-a716-446655440000": `"550e8400-e29b-41d4-a716-446655440000" is a UUID of another version`,
		"01HJHB01T8FYZ7YTR9P5N62K5B":           `invalid ID format "01HJHB01T8FYZ7YTR9P5N62K5B"`,
	}
	url, requests := countingServer(t)
	for _, tc := range []struct {
		command []string
		takes   string
	}{
		{[]string{"proofs", "get"}, "proofs get takes a UUIDv7 id (e.g. 019d6a32-13e6-72b0-97e5-3779231ea97b), an item code (e.g. 9WBPKNV7SCP87VZW) or, with --type item, an item_hash"},
		{[]string{"proofs", "get", "--type", "item"}, "proofs get takes a UUIDv7 id (e.g. 019d6a32-13e6-72b0-97e5-3779231ea97b), an item code (e.g. 9WBPKNV7SCP87VZW) or, with --type item, an item_hash"},
		{[]string{"items", "get"}, "items get takes an item's UUIDv7 id (e.g. 01a0db66-c2e0-78a4-b8cc-12b11bc97493), its item code (e.g. 9WBPKNV7SCP87VZW) or its item_hash"},
		{[]string{"items", "update", "--visibility", "public"}, "items update takes an item's UUIDv7 id (e.g. 01a0db66-c2e0-78a4-b8cc-12b11bc97493), its item code (e.g. 9WBPKNV7SCP87VZW) or its item_hash"},
		{[]string{"convert", "id"}, "convert id takes a UUIDv7 (e.g. 019d6a32-13e6-72b0-97e5-3779231ea97b)"},
	} {
		for typed, value := range cases {
			args := append(append([]string{"--base-url", url, "--api-key", "k"}, tc.command...), typed)
			stdout, stderr, exit := runCLI(t, args...)
			if want := value + ": " + tc.takes + "\n"; exit != 1 || stdout != "" || stderr != want {
				t.Errorf("%v %q: exit %d, stdout %q, stderr %q; want exit 1 and %q", tc.command, typed, exit, stdout, stderr, want)
			}
		}
	}
	if n := requests(); n != 0 {
		t.Errorf("%d requests sent, want none", n)
	}
}

func (s *itemsServer) path() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.lastPath
}

// items get and items update name the item by its UUIDv7 in its canonical
// lowercase form, in the path and in the update's resource id, however it
// was typed. Mutant killed: the id sent as typed (the uppercase text is in
// the path).
func TestCLI_ItemsGetAndUpdate_SendTheCanonicalUUIDv7(t *testing.T) {
	s := startItemsServer(t, func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(itemDoc(testItemID)))
	})
	if _, stderr, exit := runCLI(t, "--base-url", s.URL, "--api-key", "k", "items", "get", testItemIDUpper); exit != 0 || s.method() != http.MethodGet || s.path() != "/api/json/items/"+testItemID {
		t.Errorf("items get: exit %d, %s %s; want exit 0 and GET /api/json/items/%s\n%s", exit, s.method(), s.path(), testItemID, stderr)
	}
	if _, stderr, exit := runCLI(t, "--base-url", s.URL, "--api-key", "k", "items", "update", testItemIDUpper, "--visibility", "public"); exit != 0 || s.method() != http.MethodPatch || s.path() != "/api/json/items/"+testItemID {
		t.Errorf("items update: exit %d, %s %s; want exit 0 and PATCH /api/json/items/%s\n%s", exit, s.method(), s.path(), testItemID, stderr)
	}
	var sent struct {
		Data struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	if err := json.Unmarshal([]byte(s.body()), &sent); err != nil || sent.Data.ID != testItemID {
		t.Errorf("update body %s: data.id %q, %v; want %q", s.body(), sent.Data.ID, err, testItemID)
	}
}

// convertIDCLI runs `truestamp convert id args...` with stdin as its
// standard input, a pipe, and returns stdout, stderr and the exit code.
func convertIDCLI(t *testing.T, stdin string, args ...string) (string, string, int) {
	t.Helper()
	cmd := exec.Command(binaryPath, append([]string{"convert", "id"}, args...)...)
	cmd.Env = cleanEnv()
	cmd.Stdin = strings.NewReader(stdin)
	var out, errOut strings.Builder
	cmd.Stdout, cmd.Stderr = &out, &errOut
	exit := 0
	if err := cmd.Run(); err != nil {
		ee, ok := err.(*exec.ExitError)
		if !ok {
			t.Fatal(err)
		}
		exit = ee.ExitCode()
	}
	return out.String(), errOut.String(), exit
}

// `convert id` prints the time recorded inside a UUIDv7, its leftmost 48
// bits as milliseconds since the Unix epoch (SPEC.md section 11.3), in
// either case and with --type uuid7. Mutant killed: the time read to the
// second (the .056 is lost).
func TestCLI_ConvertID_PrintsTheTimeRecordedInAUUIDv7(t *testing.T) {
	for _, tc := range []struct {
		args []string
		want string
	}{
		{[]string{testItemID}, "2026-09-26T01:49:00Z\n"},
		{[]string{testItemIDUpper}, "2026-09-26T01:49:00Z\n"},
		{[]string{"019cf813-99b8-730a-84f1-5a711a9c355e", "--type", "uuid7"}, "2026-03-16T19:16:00.056Z\n"},
	} {
		stdout, stderr, exit := convertIDCLI(t, "", tc.args...)
		if exit != 0 || stdout != tc.want {
			t.Errorf("%v: exit %d, stdout %q; want exit 0 and %q\n%s", tc.args, exit, stdout, tc.want, stderr)
		}
	}
}

// An item code carries no time, and `convert id` says so of a code in any
// spelling, naming it in its one spelling; an item_hash carries none
// either. Mutant killed: the item code branch removed (the code reaches
// the UUID parser, whose error names no item code).
func TestCLI_ConvertID_AnItemCodeCarriesNoTime(t *testing.T) {
	for _, tc := range []struct{ typed, want string }{
		{"9wbp-knv7-scp8-7vzw", "9WBPKNV7SCP87VZW is an item code: an item code carries no time"},
		{soundItemCode, soundItemCode + " is an item code: an item code carries no time"},
		{soundItemHash, `"` + soundItemHash + `" is 64 hex characters, an item_hash at most: a hash carries no time`},
	} {
		stdout, stderr, exit := convertIDCLI(t, "", tc.typed)
		if exit != 1 || stdout != "" || !strings.Contains(stderr, tc.want) {
			t.Errorf("%q: exit %d, stdout %q, stderr %q; want exit 1 and %q", tc.typed, exit, stdout, stderr, tc.want)
		}
	}
}

// Read from a pipe, the one line ending that ends the line, a line feed
// or a carriage return and line feed, is not part of the value, and
// nothing else is removed: a second line ending, a carriage return alone,
// a space or a tab is refused, quoted as read. Mutants killed:
// strings.TrimSpace on stdin (the padded values print a time); the line
// feed kept (the echo form is refused); only a line feed stripped (the
// carriage return and line feed form is refused); all trailing space
// stripped (the lone carriage return and the trailing space and tab print
// a time).
func TestCLI_ConvertID_ReadsOneLineFromAPipe(t *testing.T) {
	for _, tc := range []struct {
		stdin string
		exit  int
		want  string
	}{
		{testItemID + "\n", 0, "2026-09-26T01:49:00Z\n"},
		{testItemID, 0, "2026-09-26T01:49:00Z\n"},
		{testItemID + "\r\n", 0, "2026-09-26T01:49:00Z\n"},
		{testItemID + "\n\n", 1, testItemID + `\n`},
		{testItemID + "\r\n\r\n", 1, testItemID + `\r\n`},
		{testItemID + "\r", 1, testItemID + `\r`},
		{testItemID + "\n\r", 1, testItemID + `\n\r`},
		{testItemID + " \n", 1, testItemID + ` `},
		{testItemID + "\t\r\n", 1, testItemID + `\t`},
		{" " + testItemID + "\n", 1, ` ` + testItemID},
	} {
		stdout, stderr, exit := convertIDCLI(t, tc.stdin)
		if tc.exit == 0 && (exit != 0 || stdout != tc.want) {
			t.Errorf("stdin %q: exit %d, stdout %q; want exit 0 and %q\n%s", tc.stdin, exit, stdout, tc.want, stderr)
		}
		want := `invalid ID format "` + tc.want + `": convert id takes a UUIDv7 (e.g. 019d6a32-13e6-72b0-97e5-3779231ea97b)` + "\n"
		if tc.exit == 1 && (exit != 1 || stdout != "" || stderr != want) {
			t.Errorf("stdin %q: exit %d, stdout %q, stderr %q; want exit 1 and %q", tc.stdin, exit, stdout, stderr, want)
		}
	}
}

// --type takes auto and uuid7, and nothing else. Mutant killed: any --type
// value accepted (exit 0 and a time).
func TestCLI_ConvertID_TypeIsAutoOrUUIDv7(t *testing.T) {
	for _, typ := range []string{"uuid", "uuidv7", "item_code", "Auto7"} {
		stdout, stderr, exit := convertIDCLI(t, "", testItemID, "--type", typ)
		want := `--type must be auto or uuid7 (got "` + typ + `")`
		if exit != 1 || stdout != "" || !strings.Contains(stderr, want) {
			t.Errorf("--type %s: exit %d, stdout %q, stderr %q; want exit 1 and %q", typ, exit, stdout, stderr, want)
		}
	}
}
