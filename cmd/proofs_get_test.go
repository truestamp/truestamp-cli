// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"testing"
)

const (
	testItemULID   = "01HJHB01T8FYZ7YTR9P5N62K5B"
	testSubjectID  = "019db702-b08c-73dc-a7cd-2c5e011f1dad"
	rawProofPrefix = "/api/json/proof/raw/"
	// soundItemCode is the item code vectors/item-codes.json gives the
	// sound item, positive/item.proof.json.
	soundItemCode = "WX495M26THGQBJ68"
)

// servedBytes is a stand-in for the proof served in each constructions
// mode: distinct bytes per mode, in the served form (no whitespace, no
// final line feed), so a download that re-indents, re-encodes, adds a
// line feed or asks for another mode writes other bytes.
func servedBytes(mode string) []byte {
	return []byte(`{"proof":{"constructions":"` + mode + `","type":"block","version":1},"signature":{"alg":"truestamp-ed25519-sha256-jcs","public_key":"k","value":"v"}}`)
}

// rawRequest is one request the stand-in for the raw proof route was sent.
type rawRequest struct {
	method, path, query string
	bearer              bool
}

// rawProofServer is a loopback stand-in for the raw proof route. It
// records every request and answers each with serve, given the request's
// type, id and constructions parameter ("" when it has none).
func rawProofServer(t *testing.T, serve func(typ, id, mode string) (int, []byte)) (string, func() []rawRequest) {
	t.Helper()
	var mu sync.Mutex
	var seen []rawRequest
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		seen = append(seen, rawRequest{r.Method, r.URL.Path, r.URL.RawQuery, strings.HasPrefix(r.Header.Get("Authorization"), "Bearer ")})
		mu.Unlock()
		rest, ok := strings.CutPrefix(r.URL.Path, rawProofPrefix)
		parts := strings.Split(rest, "/")
		if !ok || len(parts) != 2 || r.Method != http.MethodGet {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		status, body := serve(parts[0], parts[1], r.URL.Query().Get("constructions"))
		w.Header().Set("Content-Type", "application/vnd.truestamp.proof+json")
		w.WriteHeader(status)
		_, _ = w.Write(body)
	}))
	t.Cleanup(srv.Close)
	return srv.URL, func() []rawRequest {
		mu.Lock()
		defer mu.Unlock()
		return slices.Clone(seen)
	}
}

// byMode serves servedBytes of the mode the request names.
func byMode(_, _, mode string) (int, []byte) { return http.StatusOK, servedBytes(mode) }

func withTempCWD(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	orig, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(orig) })
	return dir
}

// dirEntries is the names in dir.
func dirEntries(t *testing.T, dir string) []string {
	t.Helper()
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	var names []string
	for _, e := range entries {
		names = append(names, e.Name())
	}
	return names
}

// Each constructions mode is asked of the raw GET, in one request, as GET
// /api/json/proof/raw/<type>/<id>?constructions=<mode> with the credential,
// full when no mode is given, and the served bytes reach stdout exactly as
// served, with nothing written to the working directory. Mutants killed:
// the body re-indented (json.Indent) or given a
// final line feed (stdout differs); the constructions parameter left out
// (the query is empty, and reference and none are served full's bytes); a
// default other than full.
func TestCLI_ProofsGet_EachConstructionsModeIsTheServedBytes(t *testing.T) {
	for _, tc := range []struct {
		name, mode string
		flag       []string
	}{
		{"default", "full", nil},
		{"full", "full", []string{"--constructions", "full"}},
		{"reference", "reference", []string{"--constructions", "reference"}},
		{"none", "none", []string{"--constructions", "none"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			url, requests := rawProofServer(t, byMode)
			dir := withTempCWD(t)
			args := append([]string{"--base-url", url, "--api-key", "test-key", "proofs", "get", "--type", "block"}, tc.flag...)
			stdout, stderr, exit := runCLI(t, append(args, testSubjectID)...)
			if exit != 0 || stdout != string(servedBytes(tc.mode)) {
				t.Fatalf("exit %d, stdout %q; want exit 0 and exactly %q\nstderr: %s", exit, stdout, servedBytes(tc.mode), stderr)
			}
			want := rawRequest{http.MethodGet, rawProofPrefix + "block/" + testSubjectID, "constructions=" + tc.mode, true}
			if got := requests(); len(got) != 1 || got[0] != want {
				t.Errorf("requests %+v, want exactly [%+v]", got, want)
			}
			if names := dirEntries(t, dir); len(names) != 0 {
				t.Errorf("stdout mode wrote files: %v", names)
			}
		})
	}
}

// -o writes the served bytes exactly as served to the path it names, and
// the receipt card, naming the mode, goes to stderr, never stdout. Mutant
// killed: the file written re-indented (its bytes differ).
func TestCLI_ProofsGet_OutWritesTheServedBytes(t *testing.T) {
	url, _ := rawProofServer(t, byMode)
	dir := withTempCWD(t)
	path := filepath.Join(dir, "custom-name.json")
	stdout, stderr, exit := runCLI(t, "--base-url", url, "--api-key", "test-key", "proofs", "get",
		"--type", "beacon", "--constructions", "reference", "-o", path, testSubjectID)
	got, err := os.ReadFile(path)
	if exit != 0 || err != nil || string(got) != string(servedBytes("reference")) {
		t.Fatalf("exit %d, %v, file %q; want exactly the served bytes\nstderr: %s", exit, err, got, stderr)
	}
	if stdout != "" || !strings.Contains(stderr, "Proof Downloaded") || !strings.Contains(stderr, "Constructions  reference") {
		t.Errorf("stdout %q, stderr %q; want an empty stdout and the receipt, naming the mode, on stderr", stdout, stderr)
	}
	if names := dirEntries(t, dir); !slices.Equal(names, []string{"custom-name.json"}) {
		t.Errorf("files %v, want only custom-name.json", names)
	}
}

// unservedBytes is a proof answered in a form that is not the served
// form: a leading space, two-space indentation, CRLF line endings and a
// final line feed.
const unservedBytes = " {\r\n  \"proof\": {\r\n    \"constructions\": \"full\",\r\n    \"type\": \"block\",\r\n    \"version\": 1\r\n  },\r\n" +
	"  \"signature\": {\r\n    \"alg\": \"truestamp-ed25519-sha256-jcs\",\r\n    \"public_key\": \"k\",\r\n    \"value\": \"v\"\r\n  }\r\n}\r\n"

// Bytes answered in any form reach stdout, the -o file and a block's
// --to-file file exactly as answered, whitespace and line endings
// included. Mutants killed: the download trimming the body
// (strings.TrimSpace: the leading space and the final CRLF are gone); the
// download canonicalizing it (jcs.Transform: the indentation and the CRLFs
// are gone).
func TestCLI_ProofsGet_EveryDestinationIsTheAnsweredBytes(t *testing.T) {
	url, _ := rawProofServer(t, func(_, _, _ string) (int, []byte) { return http.StatusOK, []byte(unservedBytes) })
	dir := withTempCWD(t)
	base := []string{"--base-url", url, "--api-key", "test-key", "proofs", "get", "--type", "block"}

	stdout, stderr, exit := runCLI(t, append(base, testSubjectID)...)
	if exit != 0 || stdout != unservedBytes {
		t.Errorf("stdout: exit %d, %q; want exit 0 and exactly %q\nstderr: %s", exit, stdout, unservedBytes, stderr)
	}
	if _, stderr, exit := runCLI(t, append(base, "-o", filepath.Join(dir, "out.json"), testSubjectID)...); exit != 0 {
		t.Errorf("-o: exit %d\n%s", exit, stderr)
	}
	if _, stderr, exit := runCLI(t, append(base, "--to-file", testSubjectID)...); exit != 0 {
		t.Errorf("--to-file: exit %d\n%s", exit, stderr)
	}
	for _, name := range []string{"out.json", testSubjectID + ".proof.json"} {
		if got, err := os.ReadFile(filepath.Join(dir, name)); err != nil || string(got) != unservedBytes {
			t.Errorf("%s: %q, %v; want exactly %q", name, got, err, unservedBytes)
		}
	}
	if names := dirEntries(t, dir); !slices.Equal(names, []string{testSubjectID + ".proof.json", "out.json"}) {
		t.Errorf("files %v, want exactly the two written", names)
	}
}

// --to-file names an item's proof <CODE>.proof.json, CODE the item code of
// its recomputed item_hash, and every other subject's <id>.proof.json,
// whatever the constructions mode, and writes the served bytes exactly as
// served. Mutants killed: a suffix for a constructions mode (the
// reference files are named otherwise); a truestamp-<type>-<id>
// name; the item named by its id; the file re-indented.
func TestCLI_ProofsGet_ToFileNames(t *testing.T) {
	item, err := os.ReadFile(soundItem())
	if err != nil {
		t.Fatal(err)
	}
	serve := func(typ, _, mode string) (int, []byte) {
		if typ == "item" {
			return http.StatusOK, item
		}
		return http.StatusOK, servedBytes(mode)
	}
	for _, tc := range []struct {
		typ, id, mode, name string
		want                []byte
	}{
		{"item", testItemULID, "full", soundItemCode + ".proof.json", item},
		{"item", testItemULID, "reference", soundItemCode + ".proof.json", item},
		{"block", testSubjectID, "full", testSubjectID + ".proof.json", servedBytes("full")},
		{"beacon", testSubjectID, "none", testSubjectID + ".proof.json", servedBytes("none")},
		{"entropy_stellar", testSubjectID, "reference", testSubjectID + ".proof.json", servedBytes("reference")},
		{"entropy_nist", testSubjectID, "full", testSubjectID + ".proof.json", servedBytes("full")},
		{"entropy_bitcoin", testSubjectID, "full", testSubjectID + ".proof.json", servedBytes("full")},
	} {
		t.Run(tc.typ+"_"+tc.mode, func(t *testing.T) {
			url, _ := rawProofServer(t, serve)
			dir := withTempCWD(t)
			_, stderr, exit := runCLI(t, "--base-url", url, "--api-key", "test-key", "proofs", "get",
				"--type", tc.typ, "--constructions", tc.mode, "--to-file", tc.id)
			if exit != 0 {
				t.Fatalf("exit %d\n%s", exit, stderr)
			}
			if names := dirEntries(t, dir); !slices.Equal(names, []string{tc.name}) {
				t.Fatalf("files %v, want exactly [%s]", names, tc.name)
			}
			if got, _ := os.ReadFile(filepath.Join(dir, tc.name)); string(got) != string(tc.want) {
				t.Errorf("%s holds %d bytes that are not the %d served", tc.name, len(got), len(tc.want))
			}
		})
	}
}

// An item request whose served bytes are no item proof gets no file name:
// it exits 1 and writes nothing. Mutant killed: FileName naming any
// served proof by its id (a file is written).
func TestCLI_ProofsGet_ToFileRefusesAnUnnamableItemProof(t *testing.T) {
	url, _ := rawProofServer(t, byMode)
	dir := withTempCWD(t)
	_, stderr, exit := runCLI(t, "--base-url", url, "--api-key", "test-key", "proofs", "get", "--to-file", testItemULID)
	if exit != 1 || !strings.Contains(stderr, "no file name for the served proof: it is not an item proof") {
		t.Errorf("exit %d, stderr %q; want exit 1 and the refusal", exit, stderr)
	}
	if names := dirEntries(t, dir); len(names) != 0 {
		t.Errorf("files %v, want none", names)
	}
}

// A --constructions value that is not exactly one of the three is
// refused before any request, and so are --witnesses and --format (-f),
// which are not flags. Mutants killed: the mode check removed (the request
// is sent); --witnesses or --format registered (the flag is accepted).
func TestCLI_ProofsGet_UnknownAndInvalidFlagsAreRefused(t *testing.T) {
	url, requests := rawProofServer(t, byMode)
	_ = withTempCWD(t)
	for _, tc := range []struct {
		args []string
		want string
	}{
		{[]string{"--constructions", "compact"}, `--constructions must be one of full | reference | none, got "compact"`},
		{[]string{"--constructions", "Full"}, `--constructions must be one of full | reference | none, got "Full"`},
		{[]string{"--witnesses", "none"}, "unknown flag: --witnesses"},
		{[]string{"--format", "json"}, "unknown flag: --format"},
		{[]string{"-f", "json"}, "unknown shorthand flag: 'f' in -f"},
	} {
		args := append([]string{"--base-url", url, "--api-key", "test-key", "proofs", "get", "--type", "block"}, tc.args...)
		_, stderr, exit := runCLI(t, append(args, testSubjectID)...)
		if exit == 0 || !strings.Contains(stderr, tc.want) {
			t.Errorf("%v: exit %d, stderr %q; want a refusal naming %q", tc.args, exit, stderr, tc.want)
		}
	}
	if n := len(requests()); n != 0 {
		t.Errorf("%d requests sent, want none", n)
	}
}

// The bytes on stdout are a proof `verify` reads from stdin and labels:
// the sound vector, under its test keyring, is supplied-keyring, exit 6.
// Mutant killed: verify's stdin read as empty (the run is malformed_json,
// exit 4).
func TestCLI_ProofsGet_StdoutIsPipeable(t *testing.T) {
	item, err := os.ReadFile(soundItem())
	if err != nil {
		t.Fatal(err)
	}
	url, _ := rawProofServer(t, func(_, _, _ string) (int, []byte) { return http.StatusOK, item })
	withTempCWD(t)
	stdout, stderr, exit := runCLI(t, "--base-url", url, "--api-key", "test-key", "proofs", "get", testItemULID)
	if exit != 0 || stdout != string(item) {
		t.Fatalf("download exit %d, %d bytes on stdout; want exit 0 and the %d served\n%s", exit, len(stdout), len(item), stderr)
	}
	out, errOut, code := verifyCLI(t, binaryPath, nil, strings.NewReader(stdout), append([]string{"-"}, underTest()...)...)
	if code != 6 {
		t.Errorf("piping download into verify: exit %d, want 6\n%s%s", code, out, errOut)
	}
}

// The two file destinations are mutually exclusive, and the error names
// both rather than silently letting one win; no request is sent.
func TestCLI_ProofsGet_OutAndToFileConflict(t *testing.T) {
	url, requests := rawProofServer(t, byMode)
	dir := withTempCWD(t)
	_, stderr, exit := runCLI(t, "--base-url", url, "--api-key", "test-key", "proofs", "get",
		"-o", filepath.Join(dir, "x.json"), "--to-file", testItemULID)
	if exit == 0 || !strings.Contains(stderr, "--out and --to-file are mutually exclusive") || len(requests()) != 0 {
		t.Errorf("exit %d, %d requests, stderr %q", exit, len(requests()), stderr)
	}
}

func TestCLI_ProofsGet_InvalidType(t *testing.T) {
	_, stderr, exit := runCLI(t, "--api-key", "test-key", "proofs", "get", "--type", "bogus", testSubjectID)
	if exit == 0 || !strings.Contains(stderr, "--type must be one of") {
		t.Fatalf("exit=%d stderr=%q", exit, stderr)
	}
	for _, want := range []string{"entropy_nist", "entropy_stellar", "entropy_bitcoin", "beacon"} {
		if !strings.Contains(stderr, want) {
			t.Errorf("want %q listed in error message, got %q", want, stderr)
		}
	}
	_, stderr, exit = runCLI(t, "--api-key", "test-key", "proofs", "get", "--type", "entropy", testSubjectID)
	if exit == 0 || !strings.Contains(stderr, "--type must be one of") {
		t.Errorf("bare entropy: exit=%d stderr=%q", exit, stderr)
	}
}

func TestCLI_ProofsGet_ShapeVsType(t *testing.T) {
	url, requests := rawProofServer(t, byMode)
	_ = withTempCWD(t)

	_, stderr, exit := runCLI(t, "--base-url", url, "--api-key", "test-key", "proofs", "get", "--type", "item", testSubjectID)
	if exit == 0 || !strings.Contains(stderr, "requires a ULID") {
		t.Errorf("item+uuid: exit=%d stderr=%q", exit, stderr)
	}
	_, stderr, exit = runCLI(t, "--base-url", url, "--api-key", "test-key", "proofs", "get", "--type", "block", testItemULID)
	if exit == 0 || !strings.Contains(stderr, "requires a UUIDv7") {
		t.Errorf("block+ulid: exit=%d stderr=%q", exit, stderr)
	}
	if n := len(requests()); n != 0 {
		t.Errorf("%d requests sent, want none", n)
	}
}

// --- --type resolution -------------------------------------------------

// resolvingServer answers /utilities/resolve-id with resolution and every
// other request with servedBytes("full"), and records both.
func resolvingServer(t *testing.T, resolution string) (string, func() int, func() []string) {
	t.Helper()
	var mu sync.Mutex
	var resolves int
	var proofs []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()
		if strings.HasSuffix(r.URL.Path, "/utilities/resolve-id") {
			resolves++
			_, _ = w.Write([]byte(resolution))
			return
		}
		proofs = append(proofs, r.URL.Path)
		_, _ = w.Write(servedBytes("full"))
	}))
	t.Cleanup(srv.Close)
	return srv.URL,
		func() int { mu.Lock(); defer mu.Unlock(); return resolves },
		func() []string { mu.Lock(); defer mu.Unlock(); return slices.Clone(proofs) }
}

// A UUIDv7 with no --type is resolved against the server in one round
// trip, and the resolved type is the one the raw GET asks for.
func TestCLI_ProofsGet_ResolvesTypeForUUIDv7(t *testing.T) {
	url, resolves, proofs := resolvingServer(t, `{"result":{"id_format":"uuidv7","matches":[
	  {"kind":"block","proof_type":"block","verifiable":true}]}}`)
	stdout, stderr, exit := runCLI(t, "--base-url", url, "--api-key", "k", "proofs", "get", testSubjectID)
	if exit != 0 || stdout != string(servedBytes("full")) {
		t.Fatalf("exit=%d stdout=%q stderr=%q", exit, stdout, stderr)
	}
	if resolves() != 1 || !slices.Equal(proofs(), []string{rawProofPrefix + "block/" + testSubjectID}) {
		t.Errorf("%d resolve-id calls and the proof requests %v; want one, and the raw GET of the block", resolves(), proofs())
	}
}

// --type is one fewer round trip and keeps an id-classification service
// out of the loop entirely.
func TestCLI_ProofsGet_ExplicitTypeSkipsResolution(t *testing.T) {
	url, resolves, proofs := resolvingServer(t, `{"result":{"id_format":"uuidv7","matches":[]}}`)
	if _, _, exit := runCLI(t, "--base-url", url, "--api-key", "k", "proofs", "get", "--type", "block", testSubjectID); exit != 0 {
		t.Fatalf("exit=%d", exit)
	}
	if resolves() != 0 || len(proofs()) != 1 {
		t.Errorf("%d resolve-id calls and %d proof requests; want none and one", resolves(), len(proofs()))
	}
}

// A subject verifiable as more than one type is not guessed at, and an id
// with no match, or a resolution the server refuses, says how to state
// the type; none of them asks for a proof.
func TestCLI_ProofsGet_UnresolvedTypesAreRefused(t *testing.T) {
	for name, resolution := range map[string]string{
		"ambiguous": `{"result":{"id_format":"uuidv7","matches":[
		  {"kind":"block","proof_type":"block"},{"kind":"beacon","proof_type":"beacon"}]}}`,
		"no match": `{"result":{"id_format":"uuidv7","matches":[]}}`,
		"refused":  `{"errors":[{"detail":"nope"}]}`,
	} {
		url, _, proofs := resolvingServer(t, resolution)
		_, stderr, exit := runCLI(t, "--base-url", url, "--api-key", "k", "proofs", "get", testSubjectID)
		if exit == 0 || !strings.Contains(stderr, "--type") || len(proofs()) != 0 {
			t.Errorf("%s: exit %d, %d proof requests, stderr %q; want a refusal naming --type and no proof request", name, exit, len(proofs()), stderr)
		}
	}
}

// --- refusals ----------------------------------------------------------

// Each refusal of the raw proof route is rendered from its meta.code alone,
// line for line: a banner, the server's detail, the code's advice, whether
// it clears on its own, and the code to quote. Retryable are
// no_block_commitment, commitment_not_final, evidence_unconfirmed,
// subject_not_ready, witness_unavailable, signing_unavailable and
// lookup_failed; subject_redacted, proof_too_large and generation_failed
// are not; a code the table does not hold gets no verdict and no advice.
// The status is 400, as the server answers these. Mutants killed: any one
// code's retryable flag flipped (its banner and verdict line change); any
// one code dropped from the table (it gets no verdict); a code's advice
// dropped or changed.
func TestCLI_ProofsGet_ExplainsEachRefusal(t *testing.T) {
	const (
		transient = "This clears on its own: try again shortly."
		terminal  = "This will not clear on its own."
		waitFor   = "Proof not available yet"
		never     = "Proof cannot be generated"
	)
	for _, tc := range []struct {
		code, banner, verdict string
		advice                []string
	}{
		{"no_block_commitment", waitFor, transient, []string{
			"A proof exists once the subject's block is committed to a public chain.",
			"Items reach a Truestamp block within about a minute and Stellar within",
			"about five.",
		}},
		{"commitment_not_final", waitFor, transient, []string{"The block's commitment is on its chain and not final there yet."}},
		{"evidence_unconfirmed", waitFor, transient, []string{
			"Truestamp signs a proof only once its own chain reads confirm the",
			"evidence the proof carries.",
		}},
		{"subject_not_ready", waitFor, transient, []string{"The subject is not yet in a state a proof can be built from."}},
		{"witness_unavailable", waitFor, transient, []string{
			"Truestamp never serves an item's proof without every witness its",
			"fingerprint commits to.",
		}},
		{"signing_unavailable", waitFor, transient, nil},
		{"lookup_failed", waitFor, transient, nil},
		{"subject_redacted", never, terminal, []string{"Restoring the item makes its proof available again."}},
		{"proof_too_large", never, terminal, []string{"Report the subject id to Truestamp."}},
		{"generation_failed", never, terminal, nil},
		{"some_future_code", "Proof request refused", "", nil},
	} {
		t.Run(tc.code, func(t *testing.T) {
			detail := "The server's own words for " + tc.code + "."
			url, _ := rawProofServer(t, func(_, _, _ string) (int, []byte) {
				return http.StatusBadRequest, []byte(fmt.Sprintf(`{"errors":[{"code":"invalid","status":"400","detail":%q,"meta":{"code":%q}}]}`, detail, tc.code))
			})
			stdout, stderr, exit := runCLI(t, "--base-url", url, "--api-key", "k", "proofs", "get", "--type", "item", testItemULID)
			if exit != 1 || stdout != "" {
				t.Fatalf("exit %d, stdout %q; want exit 1 and nothing on stdout", exit, stdout)
			}
			var got []string
			for _, l := range strings.Split(strings.TrimRight(stderr, "\n"), "\n") {
				got = append(got, strings.TrimSpace(l))
			}
			want := append([]string{tc.banner, detail}, tc.advice...)
			if tc.verdict != "" {
				want = append(want, tc.verdict)
			}
			want = append(want, "The server said "+tc.code+" (HTTP 400).")
			if !slices.Equal(got, want) {
				t.Errorf("stderr lines\n%q\nwant\n%q", got, want)
			}
		})
	}
}

// A refusal that is not a generate refusal, here the subject type
// mismatch, still surfaces the server's detail and its code.
func TestCLI_ProofsGet_SubjectTypeMismatchError(t *testing.T) {
	url, _ := rawProofServer(t, func(_, _, _ string) (int, []byte) {
		return http.StatusBadRequest, []byte(`{"errors":[{"code":"invalid","detail":"Requested type entropy_nist but subject 019db702-b08c-73dc-a7cd-2c5e011f1dad has source entropy_stellar","meta":{"code":"subject_type_mismatch"}}]}`)
	})
	_ = withTempCWD(t)
	_, stderr, exit := runCLI(t, "--base-url", url, "--api-key", "test-key", "proofs", "get", "--type", "entropy_nist", testSubjectID)
	if exit == 0 || !strings.Contains(stderr, "Requested type entropy_nist") || !strings.Contains(stderr, "The server said subject_type_mismatch (HTTP 400).") {
		t.Errorf("exit=%d stderr=%q", exit, stderr)
	}
}

// A 200 with an empty body is not a proof: exit 1, nothing on stdout.
// Mutant killed: FetchRawCtx without its empty-body check (exit 0).
func TestCLI_ProofsGet_AnEmptyAnswerIsNoProof(t *testing.T) {
	url, _ := rawProofServer(t, func(_, _, _ string) (int, []byte) { return http.StatusOK, nil })
	stdout, stderr, exit := runCLI(t, "--base-url", url, "--api-key", "k", "proofs", "get", "--type", "block", testSubjectID)
	if exit != 1 || stdout != "" || !strings.Contains(stderr, "the server answered with an empty proof") {
		t.Errorf("exit %d, stdout %q, stderr %q", exit, stdout, stderr)
	}
}

// A body over MAX_READ_OCTETS is no proof a verifier reads: exit 1,
// nothing written. Mutant killed: FetchRawCtx without its size check
// (exit 0 and the bytes on stdout).
func TestCLI_ProofsGet_AnOversizedAnswerIsNoProof(t *testing.T) {
	big := []byte(strings.Repeat(" ", 262_145))
	url, _ := rawProofServer(t, func(_, _, _ string) (int, []byte) { return http.StatusOK, big })
	stdout, stderr, exit := runCLI(t, "--base-url", url, "--api-key", "k", "proofs", "get", "--type", "block", testSubjectID)
	if exit != 1 || stdout != "" || !strings.Contains(stderr, "the served proof is over 262144 octets") {
		t.Errorf("exit %d, %d bytes on stdout, stderr %q", exit, len(stdout), stderr)
	}
}
