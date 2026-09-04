// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: MIT

package cmd

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os/exec"
	"strings"
	"testing"
)

// runCLIOut runs the binary and returns combined output plus the error,
// for cases where the distinction between the two streams does not matter.
func runCLIOut(t *testing.T, args ...string) (string, error) {
	t.Helper()
	cmd := exec.Command(binaryPath, args...)
	cmd.Env = cleanEnv()
	out, err := cmd.CombinedOutput()
	return string(out), err
}

// itemsServer is a mock JSON:API surface recording what the CLI actually
// asked for, so the tests can assert on the request and not only on the
// rendered output.
type itemsServer struct {
	*httptest.Server
	lastPath   string
	lastQuery  url.Values
	lastBody   string
	lastMethod string
}

func startItemsServer(t *testing.T, handler func(w http.ResponseWriter, r *http.Request)) *itemsServer {
	t.Helper()
	s := &itemsServer{}
	s.Server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		s.lastPath, s.lastQuery, s.lastBody, s.lastMethod = r.URL.Path, r.URL.Query(), string(body), r.Method
		w.Header().Set("Content-Type", "application/vnd.api+json")
		handler(w, r)
	}))
	t.Cleanup(s.Close)
	return s
}

const itemAttrs = `"state":"committed","visibility":"private","tags":["q3"],
  "claims_hash":"aa","item_hash":"bb","display_name":"Contract",
  "inserted_at":"2026-08-19T21:41:00Z","updated_at":"2026-08-19T21:41:00Z"`

func itemDoc(id string) string {
	return `{"data":{"type":"item","id":"` + id + `","attributes":{` + itemAttrs + `}}}`
}

// TestCLI_Items_List_RequestsNonDefaultFields pins the trap: inserted_at,
// updated_at and expires_at are NOT in the resource's json_api
// default_fields, so a request that does not name them gets empty values
// back with no error at all. That is a silent hole, and the only way to
// notice it is to assert on the request.
func TestCLI_Items_List_RequestsNonDefaultFields(t *testing.T) {
	s := startItemsServer(t, func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"data":[{"type":"item","id":"01AAA","attributes":{` + itemAttrs + `}}]}`))
	})
	_, stderr, exit := runCLI(t, "--base-url", s.URL, "--api-key", "k", "items", "list")
	if exit != 0 {
		t.Fatalf("exit=%d stderr=%q", exit, stderr)
	}
	fields := s.lastQuery.Get("fields[item]")
	for _, want := range []string{"inserted_at", "updated_at", "expires_at"} {
		if !strings.Contains(fields, want) {
			t.Errorf("fields[item] must name %q (it is not a default field), got %q", want, fields)
		}
	}
	if s.lastQuery.Get("page[limit]") == "" {
		t.Error("a limit must always be sent: the server's block/item read declares no default page size")
	}
}

// TestCLI_Items_List_LimitIsBounded keeps the client-side guard, so the
// error names the flag rather than surfacing a server 400.
func TestCLI_Items_List_LimitIsBounded(t *testing.T) {
	s := startItemsServer(t, func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"data":[]}`))
	})
	_, stderr, exit := runCLI(t, "--base-url", s.URL, "--api-key", "k",
		"items", "list", "--limit", "500")
	if exit == 0 {
		t.Fatal("--limit 500 should be rejected")
	}
	if !strings.Contains(stderr, "--limit") {
		t.Errorf("the error should name the flag, got %q", stderr)
	}
}

// TestCLI_Items_List_CommittedAndPendingConflict: the two filters are
// opposites, and answering with an arbitrary one of them would be worse
// than refusing.
func TestCLI_Items_List_CommittedAndPendingConflict(t *testing.T) {
	s := startItemsServer(t, func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"data":[]}`))
	})
	_, stderr, exit := runCLI(t, "--base-url", s.URL, "--api-key", "k",
		"items", "list", "--committed", "--pending")
	if exit == 0 {
		t.Fatal("--committed with --pending should be rejected")
	}
	if !strings.Contains(stderr, "mutually exclusive") {
		t.Errorf("stderr should say they conflict, got %q", stderr)
	}
}

// TestCLI_Items_List_AllFollowsCursors proves --all actually pages rather
// than silently returning the first page. Two pages are served; the test
// asserts both are present and that the second request carried the cursor.
func TestCLI_Items_List_AllFollowsCursors(t *testing.T) {
	var calls int
	var s *itemsServer
	s = startItemsServer(t, func(w http.ResponseWriter, r *http.Request) {
		calls++
		if r.URL.Query().Get("page[after]") == "" {
			// The `next` link is the server's, so the test proves the cursor
			// is extracted from it rather than synthesised by the client.
			_, _ = w.Write([]byte(`{"data":[{"type":"item","id":"01AAA","attributes":{` + itemAttrs + `}}],
			  "links":{"next":"` + s.URL + `/items?page%5Bafter%5D=CURSOR1"}}`))
			return
		}
		_, _ = w.Write([]byte(`{"data":[{"type":"item","id":"01BBB","attributes":{` + itemAttrs + `}}]}`))
	})
	stdout, stderr, exit := runCLI(t, "--base-url", s.URL, "--api-key", "k",
		"items", "list", "--all")
	if exit != 0 {
		t.Fatalf("exit=%d stderr=%q", exit, stderr)
	}
	if calls != 2 {
		t.Errorf("--all should have followed the cursor: %d requests", calls)
	}
	for _, want := range []string{"01AAA", "01BBB"} {
		if !strings.Contains(stdout, want) {
			t.Errorf("--all should return both pages, %q missing from:\n%s", want, stdout)
		}
	}
	if s.lastQuery.Get("page[after]") != "CURSOR1" {
		t.Errorf("the second request should carry the cursor, got %q", s.lastQuery.Get("page[after]"))
	}
}

// TestCLI_Items_List_ExposesCursorWithoutAll: without --all the caller
// must be told how to continue, or paging is undiscoverable.
func TestCLI_Items_List_ExposesCursorWithoutAll(t *testing.T) {
	s := startItemsServer(t, func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"data":[{"type":"item","id":"01AAA","attributes":{` + itemAttrs + `}}],
		  "links":{"next":"http://x/items?page%5Bafter%5D=NEXTCUR"}}`))
	})
	stdout, _, exit := runCLI(t, "--base-url", s.URL, "--api-key", "k", "items", "list", "--json")
	if exit != 0 {
		t.Fatalf("exit=%d", exit)
	}
	var got struct {
		NextCursor string `json:"next_cursor"`
	}
	if err := json.Unmarshal([]byte(stdout), &got); err != nil {
		t.Fatalf("not JSON: %v\n%s", err, stdout)
	}
	if got.NextCursor != "NEXTCUR" {
		t.Errorf("next_cursor should be extracted from the server's link, got %q", got.NextCursor)
	}
}

// TestCLI_Items_Update_SendsOnlyChangedAttributes is the safety property.
// The server accepts exactly team_id, visibility and tags; sending an
// attribute the caller did not ask to change would be a silent write.
func TestCLI_Items_Update_SendsOnlyChangedAttributes(t *testing.T) {
	s := startItemsServer(t, func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(itemDoc("01AAA")))
	})
	_, stderr, exit := runCLI(t, "--base-url", s.URL, "--api-key", "k",
		"items", "update", "01AAA", "--visibility", "public")
	if exit != 0 {
		t.Fatalf("exit=%d stderr=%q", exit, stderr)
	}
	if s.lastMethod != http.MethodPatch {
		t.Errorf("update must PATCH, got %s", s.lastMethod)
	}
	var sent struct {
		Data struct {
			Attributes map[string]any `json:"attributes"`
		} `json:"data"`
	}
	if err := json.Unmarshal([]byte(s.lastBody), &sent); err != nil {
		t.Fatalf("request body is not JSON: %v\n%s", err, s.lastBody)
	}
	if len(sent.Data.Attributes) != 1 {
		t.Errorf("only the changed attribute should be sent, got %v", sent.Data.Attributes)
	}
	if sent.Data.Attributes["visibility"] != "public" {
		t.Errorf("visibility not sent: %v", sent.Data.Attributes)
	}
}

// TestCLI_Items_Update_CannotReachASignedField pins that no flag on this
// command can touch claims. claims_hash is signed, so a `--claims` or a
// `--name` here would be a cryptographic hazard rather than a feature.
func TestCLI_Items_Update_CannotReachASignedField(t *testing.T) {
	for _, flag := range []string{"--claims", "--name", "--description", "--data-hash"} {
		out, err := runCLIOut(t, "items", "update", "01AAA", flag, "x")
		if err == nil {
			t.Errorf("items update must not accept %s: claims are immutable", flag)
		}
		if !strings.Contains(out, "unknown flag") {
			t.Errorf("%s should be an unknown flag on items update, got: %s", flag, out)
		}
	}
}

// TestCLI_Items_Update_NoOpIsRejectedClientSide: an update with nothing to
// change must not become a request. This is also the regression for the
// root --team collision: `items update <id> --team ""` is an ordinary way
// to scope a request, and it used to be read as "move to team ”", which
// reached the server and 403'd.
func TestCLI_Items_Update_NoOpIsRejectedClientSide(t *testing.T) {
	var calls int
	s := startItemsServer(t, func(w http.ResponseWriter, r *http.Request) {
		calls++
		_, _ = w.Write([]byte(itemDoc("01AAA")))
	})
	_, stderr, exit := runCLI(t, "--base-url", s.URL, "--api-key", "k",
		"items", "update", "01AAA", "--team", "")
	if exit == 0 {
		t.Fatal("an update with nothing to change should fail")
	}
	if calls != 0 {
		t.Errorf("a no-op update must not reach the server, got %d requests", calls)
	}
	if !strings.Contains(stderr, "--to-team") {
		t.Errorf("the error should point at --to-team, got %q", stderr)
	}
}

// TestCLI_Items_Update_ToTeamIsSeparateFromRequestScope pins the fix: the
// destination is --to-team, and the root --team keeps scoping the request.
func TestCLI_Items_Update_ToTeamIsSeparateFromRequestScope(t *testing.T) {
	s := startItemsServer(t, func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(itemDoc("01AAA")))
	})
	_, _, exit := runCLI(t, "--base-url", s.URL, "--api-key", "k",
		"items", "update", "01AAA", "--team", "scope-team", "--to-team", "dest-team")
	if exit != 0 {
		t.Fatalf("exit=%d", exit)
	}
	var sent struct {
		Data struct {
			Attributes map[string]any `json:"attributes"`
		} `json:"data"`
	}
	_ = json.Unmarshal([]byte(s.lastBody), &sent)
	if sent.Data.Attributes["team_id"] != "dest-team" {
		t.Errorf("--to-team should set the destination, got %v", sent.Data.Attributes)
	}
	if len(sent.Data.Attributes) != 1 {
		t.Errorf("the request-scoping --team must not become an attribute, got %v", sent.Data.Attributes)
	}
}

// TestCLI_Items_Get_ReportsCommitmentPlainly: whether a proof can be
// generated is the question most callers have, so the card must answer it
// rather than leaving the reader to interpret a state string.
func TestCLI_Items_Get_ReportsCommitmentPlainly(t *testing.T) {
	s := startItemsServer(t, func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"data":{"type":"item","id":"01AAA","attributes":{"state":"processing","visibility":"private"}}}`))
	})
	stdout, _, exit := runCLI(t, "--base-url", s.URL, "--api-key", "k", "items", "get", "01AAA")
	if exit != 0 {
		t.Fatalf("exit=%d", exit)
	}
	if !strings.Contains(stdout, "cannot be generated") {
		t.Errorf("an uncommitted item should say a proof is not available yet, got:\n%s", stdout)
	}
}

// TestCLI_Items_RequireAuth: every item command needs a credential and
// must say so rather than surfacing a raw 401.
func TestCLI_Items_RequireAuth(t *testing.T) {
	for _, args := range [][]string{
		{"items", "list"}, {"items", "get", "01AAA"},
		{"items", "update", "01AAA", "--visibility", "public"},
	} {
		out, err := runCLIOut(t, args...)
		if err == nil {
			t.Errorf("%v should fail without a credential", args)
		}
		if !strings.Contains(out, "Not authenticated") {
			t.Errorf("%v should print the not-authenticated banner, got: %s", args, out)
		}
	}
}
