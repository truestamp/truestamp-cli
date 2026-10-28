// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"bytes"
	"encoding/json"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"testing"

	"github.com/spf13/cobra"
	"github.com/truestamp/truestamp-cli/internal/items"
	"github.com/truestamp/truestamp-cli/internal/vectorcases"
)

// otherItemCode and otherItemHash are the item code and the recomputed
// item_hash vectors/item-codes.json gives positive/item-genesis-witness,
// an item proof of an item other than the sound one.
const (
	otherItemProof = "positive/item-genesis-witness.proof.json"
	otherItemCode  = "HN56SH5HC26EZ9R4"
	otherItemHash  = "8d4a6cc4b1608cefa713b45d3fc3e09e3390d64556e3ab4e0725e386b0dab8c7"
	// soundClaimsHash is the claims_hash of the sound item's record.
	soundClaimsHash = "bd4341f1a42c9b0c085b7ae89df18bdab66eeb77284c1e53077e4ae45320f4f3"
)

// readVector is the bytes of the vectors' file at path.
func readVector(t *testing.T, path string) []byte {
	t.Helper()
	data, err := os.ReadFile(vectorcases.Path(path))
	if err != nil {
		t.Fatal(err)
	}
	return data
}

// --- proofs get by item code and by item_hash ---------------------------

// `proofs get <code>` with no --type is an item, asked for in one raw GET
// by the code's one spelling, however the code was typed, with no
// resolve-id request, and the served bytes, the proof of the item that
// code names, reach stdout exactly as served. Mutants killed: a code
// resolved through resolve-id (a resolve-id request is recorded); the code
// sent as typed (the stand-in refuses the lowercase and separated forms
// with 400 id_format_mismatch); the shape check dropped so a code is
// fetched as a block (the path names block).
func TestCLI_ProofsGet_AnItemCodeIsAnItemWithNoRoundTrip(t *testing.T) {
	sound := readVector(t, "positive/item.proof.json")
	for _, typed := range []string{soundItemCode, "wx49-5m26-thgq-bj68", "WX49 5M26 THGQ BJ68", "wx495m26thgqbj68"} {
		url, resolves, others := appStandIn(t, itemResolution, sound)
		stdout, stderr, exit := runCLI(t, "--base-url", url, "--api-key", "k", "proofs", "get", typed)
		if exit != 0 || stdout != string(sound) {
			t.Errorf("%q: exit %d, %d bytes on stdout; want exit 0 and the %d served\n%s", typed, exit, len(stdout), len(sound), stderr)
		}
		if got, want := others(), []string{rawProofPrefix + "item/" + soundItemCode}; len(resolves()) != 0 || !slices.Equal(got, want) {
			t.Errorf("%q: resolve-id bodies %q and other requests %q; want none and %q", typed, resolves(), got, want)
		}
	}
}

// `proofs get <item_hash> --type item` asks for the item by its
// item_hash in lowercase, however it was typed, and writes the proof whose
// recomputed item_hash is that hash; with no --type, 64 hex characters are
// refused before any request, asking for --type item, since the app's
// resolve-id reads no hash. Mutants killed: the hash sent as typed (the
// stand-in refuses the uppercase form); the no --type hash fetched as an
// item (a request is recorded and exit 0).
func TestCLI_ProofsGet_AnItemHashNeedsTypeItem(t *testing.T) {
	sound := readVector(t, "positive/item.proof.json")
	url, resolves, others := appStandIn(t, itemResolution, sound)
	for _, typed := range []string{soundItemHash, strings.ToUpper(soundItemHash)} {
		stdout, stderr, exit := runCLI(t, "--base-url", url, "--api-key", "k", "proofs", "get", "--type", "item", typed)
		if exit != 0 || stdout != string(sound) {
			t.Errorf("--type item %q: exit %d, %d bytes on stdout; want exit 0 and the %d served\n%s", typed, exit, len(stdout), len(sound), stderr)
		}
	}
	if got, want := others(), []string{rawProofPrefix + "item/" + soundItemHash, rawProofPrefix + "item/" + soundItemHash}; !slices.Equal(got, want) {
		t.Errorf("requests %q, want %q", got, want)
	}
	stdout, stderr, exit := runCLI(t, "--base-url", url, "--api-key", "k", "proofs", "get", soundItemHash)
	want := `"` + soundItemHash + `" is 64 hex characters, which names a subject only as an item's item_hash: pass --type item to fetch that item's proof` + "\n"
	if exit != 1 || stdout != "" || stderr != want {
		t.Errorf("no --type: exit %d, stdout %q, stderr %q; want exit 1 and %q", exit, stdout, stderr, want)
	}
	if len(resolves()) != 0 || len(others()) != 2 {
		t.Errorf("resolve-id bodies %q and other requests %q; want none and the two above", resolves(), others())
	}
}

// When the server answers a code, an item_hash or a UUIDv7 with the proof
// of another item, the run refuses with "the server returned a proof for
// a different item", naming the recomputed item_hash and, for a code, its
// code, or, for a UUIDv7, the id of the served record, and writes
// nothing: no byte on stdout, no -o file and no --to-file file. A UUIDv7
// with no --type, resolved to an item, is checked alike. Mutants killed:
// the CheckServedItem call removed (exit 0, and the other item's proof is
// written to each destination); the UUIDv7 not checked (a file is
// written, and the id rows exit 0); the carried item_hash compared in
// place of the record id (the id rows name no record id).
func TestCLI_ProofsGet_AProofOfADifferentItemIsRefused(t *testing.T) {
	other := readVector(t, otherItemProof)
	byCode := "the server returned a proof for a different item: its recomputed item_hash " + otherItemHash +
		" has the item code " + otherItemCode + ", not " + soundItemCode + "; nothing was written\n"
	byHash := "the server returned a proof for a different item: its recomputed item_hash is " + otherItemHash +
		", not " + soundItemHash + "; nothing was written\n"
	byID := `the server returned a proof for a different item: its record's id is "01a0db6d-2b80-7928-9225-f8760ae1a795", not ` +
		testItemID + "; nothing was written\n"
	for _, tc := range []struct {
		name string
		args []string
		want string
	}{
		{"code to stdout", []string{soundItemCode}, byCode},
		{"code to -o", []string{"-o", "out.json", soundItemCode}, byCode},
		{"code to --to-file", []string{"--to-file", "wx49-5m26-thgq-bj68"}, byCode},
		{"hash to stdout", []string{"--type", "item", soundItemHash}, byHash},
		{"hash to --to-file", []string{"--type", "item", "--to-file", soundItemHash}, byHash},
		{"id to stdout", []string{"--type", "item", testItemID}, byID},
		{"id to -o", []string{"--type", "item", "-o", "out.json", testItemID}, byID},
		{"id to --to-file", []string{"--type", "item", "--to-file", testItemID}, byID},
		{"resolved id to --to-file", []string{"--to-file", testItemID}, byID},
	} {
		t.Run(tc.name, func(t *testing.T) {
			url, _, others := appStandIn(t, itemResolution, other)
			dir := withTempCWD(t)
			stdout, stderr, exit := runCLI(t, append([]string{"--base-url", url, "--api-key", "k", "proofs", "get"}, tc.args...)...)
			if exit != 1 || stdout != "" || stderr != tc.want {
				t.Errorf("exit %d, stdout %d bytes, stderr %q; want exit 1, nothing and %q", exit, len(stdout), stderr, tc.want)
			}
			if names := dirEntries(t, dir); len(names) != 0 {
				t.Errorf("files %v, want none", names)
			}
			if n := len(others()); n != 1 {
				t.Errorf("%d raw requests, want 1", n)
			}
		})
	}
}

// An item's proof written to a file gets a receipt that names the item by
// its item code beside its whole item_hash, both from the item_hash
// recomputed from the proof's record, on rows of their own, and links its
// pages by that code, never by its id: asked for by a loosely typed code,
// with no ID row, and asked for by its UUIDv7, with that ID row. Mutants
// killed: the Item code row left out; the code written in the Item hash
// row; the pages linked by the reference asked for, unrecomputed (the
// UUIDv7 request has no Details or Verify row); an ID row for a code.
func TestCLI_ProofsGet_TheReceiptNamesTheItemByItsCode(t *testing.T) {
	sound := readVector(t, "positive/item.proof.json")
	for _, tc := range []struct {
		args []string
		id   string
	}{
		{[]string{"--to-file", "wx495m26thgqbj68"}, ""},
		{[]string{"--type", "item", "--to-file", testItemID}, testItemID},
	} {
		url, _, _ := appStandIn(t, itemResolution, sound)
		dir := withTempCWD(t)
		stdout, stderr, exit := runCLI(t, append([]string{"--base-url", url, "--api-key", "k", "proofs", "get"}, tc.args...)...)
		if exit != 0 || stdout != "" {
			t.Fatalf("%v: exit %d, stdout %q\n%s", tc.args, exit, stdout, stderr)
		}
		if got, _ := os.ReadFile(filepath.Join(dir, soundItemCode+".proof.json")); !bytes.Equal(got, sound) {
			t.Errorf("%v: %s.proof.json holds %d bytes, not the %d served", tc.args, soundItemCode, len(got), len(sound))
		}
		rows := map[string]string{}
		var order []string
		for _, line := range strings.Split(stderr, "\n") {
			if label, value, ok := strings.Cut(strings.TrimSpace(line), "  "); ok {
				rows[label] = strings.TrimSpace(value)
				order = append(order, label)
			}
		}
		for label, want := range map[string]string{
			"Item hash": soundItemHash,
			"Item code": soundItemCode,
			"Details":   url + "/items/" + soundItemCode,
			"Verify":    url + "/verify/item/" + soundItemCode,
			"ID":        tc.id,
		} {
			if rows[label] != want {
				t.Errorf("%v: row %s = %q, want %q\n%s", tc.args, label, rows[label], want, stderr)
			}
		}
		if i := slices.Index(order, "Item hash"); i < 0 || i+1 >= len(order) || order[i+1] != "Item code" {
			t.Errorf("%v: rows in order %q; want Item code right after Item hash", tc.args, order)
		}
	}
}

// --- items get and items update by item code and by item_hash -----------

// itemRefServer is a loopback stand-in for the app's item routes: GET
// /api/json/items/<ref> answers, for the sound item's UUIDv7, its item
// code or its item_hash in their canonical forms, the item answer gives,
// and 404 for any other ref; PATCH /api/json/items/<uuidv7> answers the
// sound item, and a PATCH by any other ref 404, as the app's update route,
// which reads only a UUIDv7, does. It records each request's method and
// path, in order.
func itemRefServer(t *testing.T, answer string) (string, func() []string) {
	t.Helper()
	var mu sync.Mutex
	var log []string
	sound := soundItemDoc(testItemID, soundItemCode, soundItemHash)
	s := startItemsServer(t, func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		log = append(log, r.Method+" "+r.URL.Path)
		mu.Unlock()
		ref, _ := strings.CutPrefix(r.URL.Path, "/api/json/items/")
		switch {
		case r.Method == http.MethodGet && (ref == testItemID || ref == soundItemCode || ref == soundItemHash):
			_, _ = w.Write([]byte(answer))
		case r.Method == http.MethodPatch && ref == testItemID:
			_, _ = w.Write([]byte(sound))
		default:
			w.WriteHeader(http.StatusNotFound)
			_, _ = w.Write([]byte(`{"errors":[{"code":"not_found","status":"404","detail":"not found"}]}`))
		}
	})
	return s.URL, func() []string { mu.Lock(); defer mu.Unlock(); return slices.Clone(log) }
}

// soundItemDoc is a JSON:API item document of the item id, with item code
// code and item_hash hash.
func soundItemDoc(id, code, hash string) string {
	return `{"data":{"type":"item","id":"` + id + `","attributes":{"state":"committed","visibility":"private",` +
		`"item_hash":"` + hash + `","item_code":"` + code + `","display_name":"Contract"}}}`
}

// items get reads an item by its UUIDv7, its item code or its item_hash in
// one GET of that reference in its canonical form, and items update by a
// code or a hash sends one GET of it and then the PATCH by the UUIDv7 the
// item it read gives, with that UUIDv7 as the update's resource id; by a
// UUIDv7 it sends the PATCH alone. The request log is pinned whole.
// Mutants killed: the update PATCHing by the code or hash (the PATCH path
// holds it, and the stand-in answers 404); the update sending no GET
// first (the log lacks it); a GET before a PATCH by UUIDv7 (the log has
// it); the ref sent as typed (the lowercase and separated forms are not
// in the path).
func TestCLI_ItemsGetAndUpdate_ResolveACodeOrHashByOneGet(t *testing.T) {
	sound := soundItemDoc(testItemID, soundItemCode, soundItemHash)
	get := "GET /api/json/items/"
	patch := "PATCH /api/json/items/" + testItemID
	for _, tc := range []struct {
		args []string
		want []string
	}{
		{[]string{"items", "get", testItemID}, []string{get + testItemID}},
		{[]string{"items", "get", "wx49-5m26-thgq-bj68"}, []string{get + soundItemCode}},
		{[]string{"items", "get", strings.ToUpper(soundItemHash)}, []string{get + soundItemHash}},
		{[]string{"items", "update", "WX49_5M26_THGQ_BJ68", "--visibility", "public"}, []string{get + soundItemCode, patch}},
		{[]string{"items", "update", soundItemHash, "--visibility", "public"}, []string{get + soundItemHash, patch}},
		{[]string{"items", "update", testItemIDUpper, "--visibility", "public"}, []string{patch}},
	} {
		url, requests := itemRefServer(t, sound)
		stdout, stderr, exit := runCLI(t, append(append([]string{"--base-url", url, "--api-key", "k"}, tc.args...), "--json")...)
		if exit != 0 || !slices.Equal(requests(), tc.want) {
			t.Errorf("%v: exit %d, requests %q; want exit 0 and %q\n%s", tc.args, exit, requests(), tc.want, stderr)
			continue
		}
		var got items.Item
		if err := json.Unmarshal([]byte(stdout), &got); err != nil || got.ID != testItemID || got.ItemCode != soundItemCode {
			t.Errorf("%v: --json %s: %v", tc.args, stdout, err)
		}
	}
}

// The update's PATCH names the item by the UUIDv7 the GET's item gives,
// in the path and as data.id. Mutant killed: the update sent by the code
// with no GET (the body's id is the code).
func TestCLI_ItemsUpdate_ByCodePatchesTheItemsUUIDv7(t *testing.T) {
	s := startItemsServer(t, func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(soundItemDoc(testItemID, soundItemCode, soundItemHash)))
	})
	if _, stderr, exit := runCLI(t, "--base-url", s.URL, "--api-key", "k", "items", "update", soundItemCode, "--tags", "q3"); exit != 0 {
		t.Fatalf("exit %d\n%s", exit, stderr)
	}
	var sent struct {
		Data struct {
			ID         string         `json:"id"`
			Attributes map[string]any `json:"attributes"`
		} `json:"data"`
	}
	if err := json.Unmarshal([]byte(s.body()), &sent); err != nil || sent.Data.ID != testItemID || s.method() != http.MethodPatch {
		t.Errorf("last request %s with body %s: data.id %q, %v; want a PATCH of %q", s.method(), s.body(), sent.Data.ID, err, testItemID)
	}
}

// An answer to a code or an item_hash that is not the item named, one
// whose item code or whole item_hash is another, or whose id is no
// UUIDv7, is refused: items get exits 1 without rendering it, and items
// update sends no PATCH. A no-op update sends nothing at all. Mutants
// killed: the check of the answer removed (items update PATCHes the other
// item, exit 0); the hash compared by its code (the answer whose hash
// differs only past the code's 75 bits passes); the update's GET sent
// before its no-op check (a request is recorded).
func TestCLI_ItemsGetAndUpdate_RefuseAnAnswerOfAnotherItem(t *testing.T) {
	nearHash := soundItemHash[:63] + "d"
	for _, tc := range []struct {
		name, answer, ref, want string
	}{
		{"another code", soundItemDoc(testItemID, otherItemCode, soundItemHash), soundItemCode,
			"the server returned a different item: asked for the item code " + soundItemCode + ", answered with an item whose item code is \"" + otherItemCode + "\""},
		{"another hash, the same code", soundItemDoc(testItemID, soundItemCode, nearHash), soundItemHash,
			"the server returned a different item: asked for the item_hash " + soundItemHash + ", answered with an item whose item_hash is \"" + nearHash + "\""},
		{"an id that is no UUIDv7", soundItemDoc("01HJHB01T8FYZ7YTR9P5N62K5B", soundItemCode, soundItemHash), soundItemCode,
			"the server returned a different item: asked for " + soundItemCode + ", answered with an item whose id \"01HJHB01T8FYZ7YTR9P5N62K5B\" is not a UUIDv7"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			url, requests := itemRefServer(t, tc.answer)
			for _, args := range [][]string{{"items", "get", tc.ref}, {"items", "update", tc.ref, "--visibility", "public"}} {
				stdout, stderr, exit := runCLI(t, append([]string{"--base-url", url, "--api-key", "k"}, args...)...)
				if exit != 1 || stdout != "" || stderr != tc.want+"\n" {
					t.Errorf("%v: exit %d, stdout %q, stderr %q; want exit 1 and %q", args, exit, stdout, stderr, tc.want)
				}
			}
			if got, want := requests(), []string{"GET /api/json/items/" + tc.ref, "GET /api/json/items/" + tc.ref}; !slices.Equal(got, want) {
				t.Errorf("requests %q, want %q", got, want)
			}
		})
	}
	url, requests := itemRefServer(t, soundItemDoc(testItemID, soundItemCode, soundItemHash))
	if _, stderr, exit := runCLI(t, "--base-url", url, "--api-key", "k", "items", "update", soundItemCode); exit != 1 || !strings.Contains(stderr, "nothing to update") || len(requests()) != 0 {
		t.Errorf("a no-op update: exit %d, stderr %q, requests %q; want exit 1, nothing to update and none", exit, stderr, requests())
	}
}

// --- items get, items list and items create output -----------------------

// The item card gives the item code the API sends on its own row, on the
// line after the whole item_hash, which follows the whole claims_hash,
// the rows labelled as every other card labels them, and links the item's
// page by that code. Mutants killed: the Item code row left out; the
// Details URL built from the id (no Details row, since an item page is
// named only by its code); the old labels "Claims Hash" and "Item Hash"
// restored (those rows are missing).
func TestCLI_ItemsGet_TheCardGivesTheCodeBesideTheHash(t *testing.T) {
	doc := strings.Replace(soundItemDoc(testItemID, soundItemCode, soundItemHash), `"item_hash":`, `"claims_hash":"`+soundClaimsHash+`","item_hash":`, 1)
	url, _ := itemRefServer(t, doc)
	stdout, stderr, exit := runCLI(t, "--base-url", url, "--api-key", "k", "items", "get", soundItemCode)
	if exit != 0 {
		t.Fatalf("exit %d\n%s", exit, stderr)
	}
	var rows []string
	for _, line := range strings.Split(stdout, "\n") {
		if label, value, ok := strings.Cut(strings.TrimSpace(line), "  "); ok && (label == "Claims hash" || label == "Item hash" || label == "Item code" || label == "Details") {
			rows = append(rows, label+"="+strings.TrimSpace(value))
		}
	}
	want := []string{"Claims hash=" + soundClaimsHash, "Item hash=" + soundItemHash, "Item code=" + soundItemCode, "Details=" + url + "/items/" + soundItemCode}
	if !slices.Equal(rows, want) {
		t.Errorf("rows %q, want %q\n%s", rows, want, stdout)
	}
}

// The item card labels the time the item was submitted "Submitted", since
// Truestamp proves submission timing, never creation timing, and gives it
// in the card's order of rows. Mutants killed: the row labelled "Created"
// (the labels differ); the row left out (Submitted is missing); the
// expiry time given in it (its first field differs).
func TestCLI_ItemsGet_LabelsTheSubmissionTimeSubmitted(t *testing.T) {
	doc := strings.Replace(soundItemDoc(testItemID, soundItemCode, soundItemHash), `"display_name":`,
		`"inserted_at":"2026-09-26T01:49:00Z","expires_at":"2027-09-26T01:49:00Z","display_name":`, 1)
	url, _ := itemRefServer(t, doc)
	stdout, stderr, exit := runCLI(t, "--base-url", url, "--api-key", "k", "items", "get", soundItemCode)
	if exit != 0 {
		t.Fatalf("exit %d\n%s", exit, stderr)
	}
	var labels []string
	submitted := ""
	for _, line := range strings.Split(stdout, "\n")[2:] {
		label, value, ok := strings.Cut(strings.TrimSpace(line), "  ")
		if !ok {
			continue
		}
		labels = append(labels, label)
		if label == "Submitted" {
			submitted = strings.Fields(value)[0]
		}
	}
	want := []string{"ID", "State", "Visibility", "Name", "Item hash", "Item code", "Submitted", "Expires", "Details"}
	if !slices.Equal(labels, want) || submitted != "2026-09-26T01:49:00Z" {
		t.Errorf("card labels %q, Submitted %q; want %q and 2026-09-26T01:49:00Z\n%s", labels, submitted, want, stdout)
	}
}

// items list has a CODE column beside the ID column, under a header, with
// each id and each code whole; an item the API sends no code for has an
// empty cell, never another value. Mutants killed: the CODE column left
// out (the header and the code are missing); the code cut to a short form
// (the whole code is missing).
func TestItemList_HasAWholeCodeColumn(t *testing.T) {
	var out bytes.Buffer
	c := &cobra.Command{}
	addRecordOutputFlags(c)
	c.SetOut(&out)
	c.SetErr(&bytes.Buffer{})
	err := renderItemList(c, []items.Item{
		{ID: testItemID, ItemCode: soundItemCode, State: "committed", DisplayName: "Contract"},
		{ID: "019a0000-0000-7000-8000-000000000001", State: "created"},
	}, listPage{})
	if err != nil {
		t.Fatal(err)
	}
	var rows [][]string
	for _, line := range strings.Split(out.String(), "\n") {
		if fields := strings.Fields(line); len(fields) > 0 && (fields[0] == "ID" || strings.HasPrefix(fields[0], "01")) {
			rows = append(rows, fields)
		}
	}
	want := [][]string{
		{"ID", "CODE", "STATE", "NAME"},
		{testItemID, soundItemCode, "✓", "committed", "Contract"},
		{"019a0000-0000-7000-8000-000000000001", "created", "(no", "name)"},
	}
	if !slices.EqualFunc(rows, want, slices.Equal) {
		t.Errorf("rows %q, want %q\n%s", rows, want, out.String())
	}
}

// --pending lists only the items created or processing, as the card's
// hint has it. Mutant killed: the exclusion kept as every state but
// committed (the redacted item is listed).
func TestCLI_ItemsList_PendingIsCreatedOrProcessing(t *testing.T) {
	s := startItemsServer(t, func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"data":[` +
			`{"type":"item","id":"019a0000-0000-7000-8000-00000000000a","attributes":{"state":"created"}},` +
			`{"type":"item","id":"019a0000-0000-7000-8000-00000000000b","attributes":{"state":"redacted"}},` +
			`{"type":"item","id":"019a0000-0000-7000-8000-00000000000c","attributes":{"state":"processing"}},` +
			`{"type":"item","id":"019a0000-0000-7000-8000-00000000000d","attributes":{"state":"committed"}}]}`))
	})
	stdout, stderr, exit := runCLI(t, "--base-url", s.URL, "--api-key", "k", "items", "list", "--pending", "--json")
	if exit != 0 {
		t.Fatalf("exit %d\n%s", exit, stderr)
	}
	var got struct {
		Items []items.Item `json:"items"`
	}
	if err := json.Unmarshal([]byte(stdout), &got); err != nil {
		t.Fatalf("%v\n%s", err, stdout)
	}
	var listed []string
	for _, it := range got.Items {
		listed = append(listed, it.State)
	}
	if want := []string{"created", "processing"}; !slices.Equal(listed, want) {
		t.Errorf("--pending listed %q, want %q", listed, want)
	}
}

// items create prints the item code the server issued on its own row,
// beside the whole item_hash, and the item's Details and Verify URLs by
// that code; --json carries it as item_code beside the whole item_hash it
// is written from. Mutants killed: the Verify URL built from the id (no
// Verify row); the Item code row left out; item_code left out of the
// JSON; item_hash left out of --json.
func TestCLI_ItemsCreate_PrintsTheItemCodeAndItsVerifyURL(t *testing.T) {
	doc := `{"data":{"id":"` + testItemID + `","type":"item","attributes":{"display_name":"Echo","visibility":"private","state":"created",` +
		`"claims_jcs":"{}","claims_hash":"ch","item_hash":"` + soundItemHash + `","item_code":"` + soundItemCode + `"}}}`
	s := startCreateServer(t, http.StatusCreated, doc)
	stdout, stderr, exit := runCreate(t, "", "-n", "Echo", "-d", longDesc, "--api-key", "fake", "--base-url", s.URL)
	if exit != 0 {
		t.Fatalf("exit %d\n%s", exit, stderr)
	}
	rows := map[string]string{}
	var order []string
	for _, line := range strings.Split(stdout, "\n") {
		if label, value, ok := strings.Cut(strings.TrimSpace(line), "  "); ok {
			rows[label] = strings.TrimSpace(value)
			order = append(order, label)
		}
	}
	for label, want := range map[string]string{
		"Item hash": soundItemHash,
		"Item code": soundItemCode,
		"Details":   s.URL + "/items/" + soundItemCode,
		"Verify":    s.URL + "/verify/item/" + soundItemCode,
	} {
		if rows[label] != want {
			t.Errorf("row %s = %q, want %q\n%s", label, rows[label], want, stdout)
		}
	}
	if i := slices.Index(order, "Item hash"); i < 0 || i+1 >= len(order) || order[i+1] != "Item code" {
		t.Errorf("rows in order %q; want Item code right after Item hash", order)
	}
	stdout, stderr, exit = runCreate(t, "", "-n", "Echo", "-d", longDesc, "--json", "--api-key", "fake", "--base-url", s.URL)
	var got map[string]any
	if err := json.Unmarshal([]byte(stdout), &got); exit != 0 || err != nil || got["item_code"] != soundItemCode || got["item_hash"] != soundItemHash {
		t.Errorf("--json: exit %d, %v, item_code %v, item_hash %v\n%s", exit, err, got["item_code"], got["item_hash"], stderr)
	}
}

// --- verify and inspect ---------------------------------------------------

// specReportMembers are the members of SPEC.md section 10.5's report table
// for a run that is not labelled failed, which has no code.
var specReportMembers = []string{"attribution", "environment", "label", "rows", "subject_id", "submitted_after", "submitted_before", "trust_root", "type", "verified", "window"}

// verify's text gives an item proof's item code on the line after the
// whole item_hash DERIVE recomputed, captioned as the item code and with
// no status, for the sound item and for declared-item_hash, whose carried
// item_hash is another but whose record is the sound item's: both give the
// sound item's code. verify --json is exactly the spec's report: the
// members of SPEC.md section 10.5's table, its label and rows those the
// vectors' case gives, and no item code anywhere in it. Mutants killed:
// the code computed from the carried item_hash (declared-item_hash shows
// another code); the code written in the Item hash line's place (that line
// holds the code); an item_code member added to the report (the members
// differ, and the code is in the JSON); the Item code line given a pass
// status.
func TestCLI_Verify_GivesTheItemCodeBesideTheItemHashInTextOnly(t *testing.T) {
	hashLine := "Item hash:    " + soundItemHash + " (recomputed from the subject's record)"
	codeLine := "Item code:    " + soundItemCode + " (of that item_hash: a name for the item, not a check)"
	for _, id := range []string{"sound-item", "declared-item_hash"} {
		t.Run(id, func(t *testing.T) {
			c := vectorCase(t, id)
			text, _, _ := verifyCLI(t, binaryPath, nil, nil, append([]string{c.ProofPath()}, underTest()...)...)
			lines := strings.Split(text, "\n")
			i := slices.Index(lines, hashLine)
			if i < 0 || i+1 >= len(lines) || lines[i+1] != codeLine {
				t.Errorf("the text lacks %q followed by %q:\n%s", hashLine, codeLine, text)
			}
			for _, line := range lines {
				if strings.Contains(line, soundItemCode) && line != codeLine {
					t.Errorf("the code is on another line too: %q", line)
				}
			}
			stdout, _, _ := verifyCLI(t, binaryPath, nil, nil, append([]string{c.ProofPath(), "--json"}, underTest()...)...)
			if strings.Contains(stdout, soundItemCode) || strings.Contains(stdout, "item_code") {
				t.Errorf("verify --json carries the item code:\n%s", stdout)
			}
			var raw map[string]any
			if err := json.Unmarshal([]byte(stdout), &raw); err != nil {
				t.Fatalf("%v\n%s", err, stdout)
			}
			want := specReportMembers
			if c.Expected.Label == "failed" {
				want = append([]string{"code"}, want...)
				slices.Sort(want)
			}
			if got := sortedKeys(raw); !slices.Equal(got, want) {
				t.Errorf("report members %q, want %q", got, want)
			}
			var report verifyReport
			if err := json.Unmarshal([]byte(stdout), &report); err != nil {
				t.Fatal(err)
			}
			if report.Label != c.Expected.Label || len(report.Rows) != len(c.Expected.Rows) {
				t.Fatalf("label %s and %d rows, want %s and %d", report.Label, len(report.Rows), c.Expected.Label, len(c.Expected.Rows))
			}
			for j, row := range report.Rows {
				got := vectorcases.Row{Code: row.Code, Part: row.Part, Status: row.Status, Step: row.Step}
				if want := c.Expected.Rows[j]; got.String() != want.String() {
					t.Errorf("row %d = %s, want %s", j, got, want)
				}
			}
		})
	}
}

// inspect gives an item proof's item code, from the item_hash recomputed
// from its record, in --json as item_code and in the text on the row after
// that recomputed item_hash; for declared-item_hash, whose carried
// item_hash is another, it is still the sound item's code; and for a
// proof of any other type item_code is null and no Item code row is
// written. Mutants killed: the code computed from the carried item_hash
// (declared-item_hash gives another code); item_code left null for an item
// proof; the Item code row left out of the text.
func TestCLI_Inspect_GivesTheItemCodeOfTheRecomputedItemHash(t *testing.T) {
	codeRow := textRow("  Item code", soundItemCode+" (of the recomputed item_hash: a name for the item, not a check)")
	for _, tc := range []struct {
		proof string
		code  *string
	}{
		{"positive/item.proof.json", ptr(soundItemCode)},
		{"negative/declared-item_hash.proof.json", ptr(soundItemCode)},
		{"positive/block.proof.json", nil},
		{"positive/beacon.proof.json", nil},
	} {
		got, _, _ := inspectJSON(t, vectorcases.Path(tc.proof))
		if (got.ItemCode == nil) != (tc.code == nil) || got.ItemCode != nil && *got.ItemCode != *tc.code {
			t.Errorf("%s: item_code %v, want %v", tc.proof, deref(got.ItemCode), deref(tc.code))
		}
		text, _ := runInspectCLI(t, vectorcases.Path(tc.proof))
		recomputed := textRow("  recomputed", soundItemHash)
		switch {
		case tc.code != nil && !strings.Contains(text, "\n"+recomputed+"\n"+codeRow+"\n"):
			t.Errorf("%s: the text lacks %q right after %q:\n%s", tc.proof, codeRow, recomputed, text)
		case tc.code == nil && strings.Contains(text, "Item code"):
			t.Errorf("%s: the text has an Item code row:\n%s", tc.proof, text)
		}
	}
}

func ptr(s string) *string { return &s }

func deref(s *string) string {
	if s == nil {
		return "null"
	}
	return *s
}

// --- schema ----------------------------------------------------------------

// schema get item-code names the alphabet and length of internal/itemcode,
// the commands that take a code, and each place one is written, each
// naming a command the binary has, in --json and in the text, whose
// "taken by" line lists the commands in order; and the item_code member it
// names for inspect --json is one inspect writes. Mutants killed: a field
// naming a command that does not exist; the alphabet written from another
// string; the text's "taken by" line left out.
func TestSchema_ItemCodeFieldsAreTheCommands(t *testing.T) {
	doc := runSchema(t, "item-code")
	if doc["alphabet"] != "0123456789ABCDEFGHJKMNPQRSTVWXYZ" || doc["length"] != float64(16) {
		t.Errorf("alphabet %v, length %v", doc["alphabet"], doc["length"])
	}
	for _, name := range append(slices.Clone(itemCodeTakenBy), func() []string {
		var out []string
		for _, f := range itemCodeFields {
			out = append(out, f.Command)
		}
		return out
	}()...) {
		found, _, err := rootCmd.Find(strings.Fields(name))
		if err != nil || found.CommandPath() != "truestamp "+name {
			t.Errorf("%q names no command of the binary (%v)", name, err)
		}
	}
	text, err := exec.Command(binaryPath, "schema", "get", "item-code").CombinedOutput()
	if want := "\n  taken by  items get, items update, proofs get\n"; err != nil || !strings.Contains(string(text), want) {
		t.Errorf("schema get item-code: %v; the text lacks %q:\n%s", err, want, text)
	}
	_, raw, _ := inspectJSON(t, soundItem())
	if _, ok := raw["item_code"]; !ok {
		t.Errorf("inspect --json has no item_code member: %v", sortedKeys(raw))
	}
}

// itemCodeOfHex gives the item code of every known answer of the derive
// list of vectors/item-codes.json from the HEX64 text of its item_hash,
// and no code for any other text: the same hash in uppercase, one
// character short or long, a character that is not hex, and nothing.
// Mutants killed: the case check left out (the uppercase hash gives a
// code); the length check left out (the 66-character text, whose first 32
// bytes are the hash, gives a code).
func TestItemCodeOfHex_GivesTheKnownAnswersOfHex64Only(t *testing.T) {
	var file struct {
		Derive []struct {
			ItemHash string `json:"item_hash"`
			ItemCode string `json:"item_code"`
		} `json:"derive"`
	}
	if err := json.Unmarshal(readVector(t, "item-codes.json"), &file); err != nil || len(file.Derive) != 10 {
		t.Fatalf("%v: %d derive answers, want 10", err, len(file.Derive))
	}
	for _, answer := range file.Derive {
		if got, ok := itemCodeOfHex(answer.ItemHash); !ok || got != answer.ItemCode {
			t.Errorf("itemCodeOfHex(%s) = %q, %v; want %q", answer.ItemHash, got, ok, answer.ItemCode)
		}
	}
	for _, text := range []string{strings.ToUpper(soundItemHash), soundItemHash[:63], soundItemHash + "00", soundItemHash[:63] + "g", ""} {
		if got, ok := itemCodeOfHex(text); ok || got != "" {
			t.Errorf("itemCodeOfHex(%q) = %q, %v; want no code", text, got, ok)
		}
	}
}
