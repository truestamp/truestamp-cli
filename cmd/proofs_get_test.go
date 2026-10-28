// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"sync"
	"testing"
)

const (
	// testItemID is the id of the sound item, positive/item.proof.json.
	testItemID     = "01a0db66-c2e0-78a4-b8cc-12b11bc97493"
	testSubjectID  = "019db702-b08c-73dc-a7cd-2c5e011f1dad"
	rawProofPrefix = "/api/json/proof/raw/"
	// soundItemCode is the item code vectors/item-codes.json gives the
	// sound item, positive/item.proof.json.
	soundItemCode = "WX495M26THGQBJ68"
	// soundItemHash is that item's recomputed item_hash, from the same
	// file.
	soundItemHash = "e74892d046d46175c8dbdecbbbc705a36816df58e65a2ee1861c100c16eb5f8c"
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

// standInUUIDv7, standInItemCode and standInItemHash are the forms of an
// id the stand-in for the raw proof route takes: the canonical text of a
// UUIDv7, of an item code and of an item_hash, the forms this CLI sends.
var (
	standInUUIDv7   = regexp.MustCompile(`\A[0-9a-f]{8}-[0-9a-f]{4}-7[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}\z`)
	standInItemCode = regexp.MustCompile(`\A[0-9A-HJKMNP-TV-Z]{16}\z`)
	standInItemHash = regexp.MustCompile(`\A[0-9a-f]{64}\z`)
)

// idFormatMismatch is the app's refusal of an id the raw proof route
// cannot read for the type requested (Truestamp.Proofs.Actions.Generate):
// a 400 whose meta.code is id_format_mismatch.
func idFormatMismatch(typ, detail string) []byte {
	return []byte(fmt.Sprintf(`{"errors":[{"code":"invalid","status":"400","detail":%q,"meta":{"code":"id_format_mismatch","requested_type":%q}}]}`, detail, typ))
}

// standInRefusal is the app's answer to a raw proof request for an id its
// route cannot read for typ, or nil for an id it reads.
func standInRefusal(typ, id string) []byte {
	switch {
	case typ == "item" && !standInUUIDv7.MatchString(id) && !standInItemCode.MatchString(id) && !standInItemHash.MatchString(id):
		return idFormatMismatch(typ, "type=item needs an item id (UUIDv7), item code or item_hash")
	case typ != "item" && !standInUUIDv7.MatchString(id):
		return idFormatMismatch(typ, "ID is not a valid UUIDv7 for type="+typ)
	}
	return nil
}

// rawProofServer is a loopback stand-in for the raw proof route. It
// records every request and answers each with serve, given the request's
// type, id and constructions parameter ("" when it has none), once the id
// is one it reads for that type: for an item a UUIDv7, an item code or an
// item_hash, and for every other type a UUIDv7, each only in the canonical
// form this CLI sends (standInUUIDv7, standInItemCode, standInItemHash).
// That is stricter than the app's route, whose Truestamp.IdShape.item_ref/1
// also reads an uppercase UUIDv7 or item_hash and every spelling of a
// code. Any other id, 26 characters of Crockford base32 among them, is
// answered with the app's 400 id_format_mismatch.
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
		typ, id := parts[0], parts[1]
		if refusal := standInRefusal(typ, id); refusal != nil {
			w.Header().Set("Content-Type", "application/vnd.api+json")
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write(refusal)
			return
		}
		status, body := serve(typ, id, r.URL.Query().Get("constructions"))
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
		{"item", testItemID, "full", soundItemCode + ".proof.json", item},
		{"item", testItemID, "reference", soundItemCode + ".proof.json", item},
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

// An item request by UUIDv7 whose served bytes are no item proof is
// refused before anything is written, since it has no record whose id
// could be the one asked for: it exits 1 with that refusal alone and
// writes nothing. Mutant killed: the UUIDv7 not checked (FileName's
// refusal, "no file name for the served proof", is given instead).
func TestCLI_ProofsGet_ToFileRefusesAnUnnamableItemProof(t *testing.T) {
	url, _ := rawProofServer(t, byMode)
	dir := withTempCWD(t)
	_, stderr, exit := runCLI(t, "--base-url", url, "--api-key", "test-key", "proofs", "get", "--type", "item", "--to-file", testItemID)
	if want := "the served proof cannot be checked against " + testItemID + ", so nothing was written: it is not an item proof\n"; exit != 1 || stderr != want {
		t.Errorf("exit %d, stderr %q; want exit 1 and %q", exit, stderr, want)
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
	stdout, stderr, exit := runCLI(t, "--base-url", url, "--api-key", "test-key", "proofs", "get", "--type", "item", testItemID)
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
		"-o", filepath.Join(dir, "x.json"), "--to-file", testItemID)
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

// An item code or an item_hash given with a --type other than item is
// refused before any request, naming the UUIDv7 the type takes. Mutant
// killed: validateTypeVsShape accepting every shape (the refusal names
// the item's UUIDv7 instead).
func TestCLI_ProofsGet_ShapeVsType(t *testing.T) {
	url, requests := rawProofServer(t, byMode)
	_ = withTempCWD(t)
	for _, tc := range []struct{ typ, id, want string }{
		{"block", "9WBPKNV7SCP87VZW", "--type block requires a UUIDv7 id (e.g. 019d6a32-13e6-72b0-97e5-3779231ea97b); got an item code, which names only an item"},
		{"entropy_nist", soundItemHash, "--type entropy_nist requires a UUIDv7 id (e.g. 019d6a32-13e6-72b0-97e5-3779231ea97b); got 64 hex characters, an item_hash, which names only an item"},
	} {
		_, stderr, exit := runCLI(t, "--base-url", url, "--api-key", "test-key", "proofs", "get", "--type", tc.typ, tc.id)
		if exit != 1 || !strings.Contains(stderr, tc.want) {
			t.Errorf("--type %s %s: exit %d, stderr %q; want exit 1 and %q", tc.typ, tc.id, exit, stderr, tc.want)
		}
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

// entropyResolution and blockResolution are the app's answers to
// resolve-id for an entropy observation's UUIDv7 and for a block's
// (Truestamp.AI.IdResolver.resolve/2, its entropy_match/1 and
// block_match/1): one match each, the observation's proof_type its source
// with no other type, and the block's proof_type block, also verifiable as
// a beacon.
const (
	entropyResolution = `{"result":{"id":"` + testSubjectID + `","normalized_id":"` + testSubjectID + `","id_format":"uuidv7","matches":[` +
		`{"kind":"entropy_observation","verifiable":true,"proof_type":"entropy_stellar","also_verifiable_as":[],` +
		`"summary":{"id":"` + testSubjectID + `","source":"entropy_stellar","state":"committed"}}]}}`
	blockResolution = `{"result":{"id":"` + testSubjectID + `","normalized_id":"` + testSubjectID + `","id_format":"uuidv7","matches":[` +
		`{"kind":"block","verifiable":true,"proof_type":"block","also_verifiable_as":["beacon"],` +
		`"summary":{"id":"` + testSubjectID + `","state":"committed","block_hash":"` + soundItemHash + `"}}]}}`
)

// A UUIDv7 with no --type is resolved against the server in one round
// trip, and the resolved type is the one the raw GET asks for: an entropy
// observation, verifiable as one type, is fetched as that type. Mutant
// killed: the resolved type ignored (the raw GET is not of
// entropy_stellar).
func TestCLI_ProofsGet_ResolvesTypeForUUIDv7(t *testing.T) {
	url, resolves, proofs := resolvingServer(t, entropyResolution)
	stdout, stderr, exit := runCLI(t, "--base-url", url, "--api-key", "k", "proofs", "get", testSubjectID)
	if exit != 0 || stdout != string(servedBytes("full")) {
		t.Fatalf("exit=%d stdout=%q stderr=%q", exit, stdout, stderr)
	}
	if resolves() != 1 || !slices.Equal(proofs(), []string{rawProofPrefix + "entropy_stellar/" + testSubjectID}) {
		t.Errorf("%d resolve-id calls and the proof requests %v; want one, and the raw GET of the observation", resolves(), proofs())
	}
}

// A block's id, which the app answers with proof_type block and
// also_verifiable_as ["beacon"], is refused rather than guessed, naming
// both types and the --type that picks each, with no proof request; with
// --type block or --type beacon it is fetched as that type with no
// resolution. Mutant killed: also_verifiable_as not read (the block's
// proof is fetched, exit 0).
func TestCLI_ProofsGet_ABlockIdIsABlockOrABeacon(t *testing.T) {
	url, resolves, proofs := resolvingServer(t, blockResolution)
	stdout, stderr, exit := runCLI(t, "--base-url", url, "--api-key", "k", "proofs", "get", testSubjectID)
	want := testSubjectID + " is verifiable as more than one subject type, block | beacon, which is refused rather than guessed: pass --type block or --type beacon\n"
	if exit != 1 || stdout != "" || stderr != want || resolves() != 1 || len(proofs()) != 0 {
		t.Errorf("exit %d, stdout %q, stderr %q, %d resolve-id calls, proof requests %v; want exit 1, %q, one call and none", exit, stdout, stderr, resolves(), proofs(), want)
	}
	for _, typ := range []string{"block", "beacon"} {
		if _, stderr, exit := runCLI(t, "--base-url", url, "--api-key", "k", "proofs", "get", "--type", typ, testSubjectID); exit != 0 {
			t.Errorf("--type %s: exit %d\n%s", typ, exit, stderr)
		}
	}
	if got, want := proofs(), []string{rawProofPrefix + "block/" + testSubjectID, rawProofPrefix + "beacon/" + testSubjectID}; resolves() != 1 || !slices.Equal(got, want) {
		t.Errorf("%d resolve-id calls, proof requests %v; want one and %v", resolves(), got, want)
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
// with no match, one that names a team, whose proof_type is null, or a
// resolution the server refuses, says how to state the type; none of them
// asks for a proof. Mutants killed: a null proof_type read as item (the
// team's id is fetched as an item proof); the block's also_verifiable_as
// ignored (the block's proof is requested).
func TestCLI_ProofsGet_UnresolvedTypesAreRefused(t *testing.T) {
	for name, resolution := range map[string]string{
		"a block": blockResolution,
		"a team": `{"result":{"id_format":"uuidv7","matches":[{"kind":"team","verifiable":false,"proof_type":null,"also_verifiable_as":[],` +
			`"summary":{"id":"` + testSubjectID + `","name":"T","personal":false,"role":"team_owner"}}]}}`,
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
			stdout, stderr, exit := runCLI(t, "--base-url", url, "--api-key", "k", "proofs", "get", "--type", "item", testItemID)
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
