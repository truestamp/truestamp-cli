// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"encoding/json/jsontext"
	"encoding/json/v2"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"slices"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
)

// runCLIOut runs the binary and returns combined output plus the error,
// for cases where the distinction between the two streams does not matter.
// itemsServer is a mock JSON:API surface recording what the CLI actually
// asked for, so the tests can assert on the request and not only on the
// rendered output.
type itemsServer struct {
	*httptest.Server
	// The handler runs on the server's own goroutines while the test body
	// runs on its own, and `runCLI` shells out, so there is no
	// happens-before edge between the write and the read that the race
	// detector can see. Guard the recorded request rather than relying on
	// the subprocess boundary.
	mu         sync.Mutex
	lastPath   string
	lastQuery  url.Values
	lastBody   string
	lastMethod string
}

func (s *itemsServer) record(r *http.Request, body string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.lastPath, s.lastQuery, s.lastBody, s.lastMethod = r.URL.Path, r.URL.Query(), body, r.Method
}

func (s *itemsServer) query() url.Values {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.lastQuery
}

func (s *itemsServer) body() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.lastBody
}

func (s *itemsServer) method() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.lastMethod
}

func startItemsServer(t *testing.T, handler func(w http.ResponseWriter, r *http.Request)) *itemsServer {
	t.Helper()
	s := &itemsServer{}
	s.Server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		s.record(r, string(body))
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
	fields := s.query().Get("fields[item]")
	for _, want := range []string{"inserted_at", "updated_at", "expires_at"} {
		if !strings.Contains(fields, want) {
			t.Errorf("fields[item] must name %q (it is not a default field), got %q", want, fields)
		}
	}
	if s.query().Get("page[limit]") == "" {
		t.Error("a limit must always be sent: the server's block/item read declares no default page size")
	}
}

// TestCLI_Items_List_LimitIsBounded keeps the client-side guard, so the
// error names the flag rather than surfacing a server 400.
// TestCLI_Items_List_LimitFloorIsOursCeilingIsTheServers pins where each
// bound lives. The CLI used to reject anything over a local MaxLimit = 100,
// which was both unbacked -- the server's OpenAPI document declares
// page.limit with "minimum": 1 and no maximum anywhere -- and wrong: the
// server happily serves 250 blocks. The floor is the one bound the
// published contract states, so it is the one the CLI enforces.
func TestCLI_Items_List_LimitFloorIsOursCeilingIsTheServers(t *testing.T) {
	var gotLimit string
	s := startItemsServer(t, func(w http.ResponseWriter, r *http.Request) {
		gotLimit = r.URL.Query().Get("page[limit]")
		_, _ = w.Write([]byte(`{"data":[]}`))
	})

	// Below the documented minimum: refused locally, by name, no request.
	_, stderr, exit := runCLI(t, "--base-url", s.URL, "--api-key", "k",
		"items", "list", "--limit", "0")
	if exit == 0 {
		t.Error("--limit 0 is below the documented minimum and must be refused")
	}
	if !strings.Contains(stderr, "--limit") {
		t.Errorf("the error should name the flag, got %q", stderr)
	}

	// Large: forwarded verbatim, for the server to accept or refuse.
	_, _, exit = runCLI(t, "--base-url", s.URL, "--api-key", "k",
		"items", "list", "--limit", "500")
	if exit != 0 {
		t.Errorf("a large --limit must be forwarded, not judged locally; exit %d", exit)
	}
	if gotLimit != "500" {
		t.Errorf("page[limit] = %q, want the value forwarded verbatim", gotLimit)
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
	var calls atomic.Int64
	s := startItemsServer(t, func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		if r.URL.Query().Get("page[after]") == "" {
			// The `next` link is the server's, so the test proves the cursor
			// is extracted from it rather than synthesised by the client.
			// Built from r.Host rather than the server value the test is
			// still assigning, which was a read of that variable from the
			// handler goroutine while the test goroutine wrote it.
			_, _ = w.Write([]byte(`{"data":[{"type":"item","id":"01AAA","attributes":{` + itemAttrs + `}}],
			  "links":{"next":"http://` + r.Host + `/items?page%5Bafter%5D=CURSOR1"}}`))
			return
		}
		_, _ = w.Write([]byte(`{"data":[{"type":"item","id":"01BBB","attributes":{` + itemAttrs + `}}]}`))
	})
	stdout, stderr, exit := runCLI(t, "--base-url", s.URL, "--api-key", "k",
		"items", "list", "--max", "10")
	if exit != 0 {
		t.Fatalf("exit=%d stderr=%q", exit, stderr)
	}
	if got := calls.Load(); got != 2 {
		t.Errorf("--max should have followed the cursor: %d requests", got)
	}
	for _, want := range []string{"01AAA", "01BBB"} {
		if !strings.Contains(stdout, want) {
			t.Errorf("--max should return both pages, %q missing from:\n%s", want, stdout)
		}
	}
	if s.query().Get("page[after]") != "CURSOR1" {
		t.Errorf("the second request should carry the cursor, got %q", s.query().Get("page[after]"))
	}
}

// TestCLI_Items_List_ExposesCursorWithoutMax: without --max the caller
// must be told how to continue, or paging is undiscoverable.
func TestCLI_Items_List_ExposesCursorWithoutMax(t *testing.T) {
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
		_, _ = w.Write([]byte(itemDoc(testItemID)))
	})
	_, stderr, exit := runCLI(t, "--base-url", s.URL, "--api-key", "k",
		"items", "update", testItemID, "--visibility", "public")
	if exit != 0 {
		t.Fatalf("exit=%d stderr=%q", exit, stderr)
	}
	if s.method() != http.MethodPatch {
		t.Errorf("update must PATCH, got %s", s.method())
	}
	var sent struct {
		Data struct {
			Attributes map[string]any `json:"attributes"`
		} `json:"data"`
	}
	if err := json.Unmarshal([]byte(s.body()), &sent); err != nil {
		t.Fatalf("request body is not JSON: %v\n%s", err, s.body())
	}
	if len(sent.Data.Attributes) != 1 {
		t.Errorf("only the changed attribute should be sent, got %v", sent.Data.Attributes)
	}
	if sent.Data.Attributes["visibility"] != "public" {
		t.Errorf("visibility not sent: %v", sent.Data.Attributes)
	}
}

// TestCLI_Items_Update_ClearingTagsSendsAnEmptyList pins `--tags ""`: it
// clears the item's tags by sending "tags":[], never "tags":null, which the
// server's tags attribute (a list that is never nil) does not take. Mutant
// killed: the request body written with encoding/json's v1 semantics, which
// write the nil slice NormalizeTags gives as null.
func TestCLI_Items_Update_ClearingTagsSendsAnEmptyList(t *testing.T) {
	s := startItemsServer(t, func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(itemDoc(testItemID)))
	})
	_, stderr, exit := runCLI(t, "--base-url", s.URL, "--api-key", "k",
		"items", "update", testItemID, "--tags", "")
	if exit != 0 {
		t.Fatalf("exit=%d stderr=%q", exit, stderr)
	}
	var sent struct {
		Data struct {
			Attributes map[string]jsontext.Value `json:"attributes"`
		} `json:"data"`
	}
	if err := json.Unmarshal([]byte(s.body()), &sent); err != nil {
		t.Fatalf("request body is not JSON: %v\n%s", err, s.body())
	}
	if tags, ok := sent.Data.Attributes["tags"]; !ok || string(tags) != "[]" || len(sent.Data.Attributes) != 1 {
		t.Errorf("--tags \"\" should send only \"tags\":[], got %s", s.body())
	}
}

// TestCLI_Items_Update_CannotReachAHashedField pins that no flag on this
// command can touch claims. claims_hash covers them, so a `--claims` or a
// `--name` here would be a cryptographic hazard rather than a feature.
func TestCLI_Items_Update_CannotReachAHashedField(t *testing.T) {
	for _, flag := range []string{"--claims", "--name", "--description", "--data-hash"} {
		out, code := runCLIText(t, "items", "update", testItemID, flag, "x")
		if code == 0 {
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
		_, _ = w.Write([]byte(itemDoc(testItemID)))
	})
	_, stderr, exit := runCLI(t, "--base-url", s.URL, "--api-key", "k",
		"items", "update", testItemID, "--team", "")
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
		_, _ = w.Write([]byte(itemDoc(testItemID)))
	})
	_, _, exit := runCLI(t, "--base-url", s.URL, "--api-key", "k",
		"items", "update", testItemID, "--team", "scope-team", "--to-team", "019dbd00-0000-7000-8000-000000000002")
	if exit != 0 {
		t.Fatalf("exit=%d", exit)
	}
	var sent struct {
		Data struct {
			Attributes map[string]any `json:"attributes"`
		} `json:"data"`
	}
	_ = json.Unmarshal([]byte(s.body()), &sent)
	if sent.Data.Attributes["team_id"] != "019dbd00-0000-7000-8000-000000000002" {
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
	stdout, _, exit := runCLI(t, "--base-url", s.URL, "--api-key", "k", "items", "get", testItemID)
	if exit != 0 {
		t.Fatalf("exit=%d", exit)
	}
	if !strings.Contains(stdout, "cannot be generated") {
		t.Errorf("an uncommitted item should say a proof is not available yet, got:\n%s", stdout)
	}
}

// A committed item is in a block, and the server refuses its proof until
// that block has a final commitment on a public chain (no_block_commitment
// while it has none, commitment_not_final while none is final), so the
// card's State row says which of the two the item is in, read from the
// included block: a block committed to a public chain names the command
// that fetches the proof; a finalized block says the proof follows its
// commitment; and with no block included (an older server) the row says
// only what is always true. The "Not yet committed" hint shows only for an
// item still waiting for its block. Mutants killed: the label put back to
// one caption for every committed item ("follows once" for a committed
// block); the label shown for every state; the hint shown for a redacted
// item.
func TestCLI_Items_Get_CommittedSaysTheProofFollowsTheBlock(t *testing.T) {
	block := func(state string) string {
		return `,"relationships":{"block":{"data":{"type":"block","id":"B1"}}}},"included":[{"type":"block","id":"B1","attributes":{"state":"` + state + `"}}]}`
	}
	for _, tc := range []struct {
		name, state, tail string
		want              []string
		hint              bool
	}{
		{"committed block", "committed", block("committed"), []string{"committed  (in a block committed to a public chain; 'truestamp proofs get WX495M26THGQBJ68' fetches its proof)"}, false},
		{"finalized block", "committed", block("finalized"), []string{"committed  (in a block; its proof follows once the block is committed to a public chain)"}, false},
		{"no block included", "committed", "}}", []string{"committed  (in a block; a proof is served once that block is committed to a public chain)"}, false},
		{"processing", "processing", "}}", []string{"processing"}, true},
		{"redacted", "redacted", "}}", []string{"redacted"}, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s := startItemsServer(t, func(w http.ResponseWriter, r *http.Request) {
				_, _ = w.Write([]byte(`{"data":{"type":"item","id":"01AAA","attributes":{"state":"` + tc.state + `","visibility":"private","item_code":"WX495M26THGQBJ68"}` + tc.tail))
			})
			stdout, _, exit := runCLI(t, "--base-url", s.URL, "--api-key", "k", "items", "get", testItemID)
			if exit != 0 {
				t.Fatalf("exit=%d\n%s", exit, stdout)
			}
			var rows []string
			for _, line := range strings.Split(stdout, "\n") {
				if label, value, ok := strings.Cut(strings.TrimSpace(line), " "); ok && label == "State" {
					rows = append(rows, strings.TrimSpace(value))
				}
			}
			if !slices.Equal(rows, tc.want) {
				t.Errorf("State rows %q; want %q\n%s", rows, tc.want, stdout)
			}
			if got := strings.Contains(stdout, "Not yet committed"); got != tc.hint {
				t.Errorf("the Not yet committed hint shown: %v; want %v\n%s", got, tc.hint, stdout)
			}
		})
	}
}

// TestCLI_Items_Update_CardHasNoBlockState: an update's response carries
// no included block, so a committed item's card after an update gives the
// neutral wording, never the claim that the proof follows or is ready.
func TestCLI_Items_Update_CardHasNoBlockState(t *testing.T) {
	s := startItemsServer(t, func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"data":{"type":"item","id":"` + testItemID + `","attributes":{"state":"committed","visibility":"public"}}}`))
	})
	stdout, stderr, exit := runCLI(t, "--base-url", s.URL, "--api-key", "k", "items", "update", testItemID, "--visibility", "public")
	if exit != 0 {
		t.Fatalf("exit=%d\n%s%s", exit, stdout, stderr)
	}
	if !strings.Contains(stdout, "committed  (in a block; a proof is served once that block is committed to a public chain)") {
		t.Errorf("the update card should give the neutral wording:\n%s", stdout)
	}
}

// The --committed filter's help says the same as the card: committed
// means in a block, with the proof following the block's public
// commitment. Mutant killed: the help put back to "Only items that have
// been committed (a proof can be generated)".
func TestCLI_Items_List_CommittedHelpSaysTheProofFollowsTheBlock(t *testing.T) {
	stdout, _, exit := runCLI(t, "items", "list", "--help")
	if exit != 0 {
		t.Fatalf("exit=%d\n%s", exit, stdout)
	}
	_, flags, _ := strings.Cut(stdout, "\nFlags:\n")
	flags, _, _ = strings.Cut(flags, "\n\n")
	var lines []string
	for _, line := range strings.Split(flags, "\n") {
		if fields := strings.Fields(line); len(fields) > 0 && fields[0] == "--committed" {
			lines = append(lines, strings.Join(fields, " "))
		}
	}
	want := []string{"--committed Only items committed in a block (a proof follows once the block is committed to a public chain)"}
	if !slices.Equal(lines, want) {
		t.Errorf("--committed help %q; want %q\n%s", lines, want, stdout)
	}
}

// TestCLI_Items_RequireAuth: every item command needs a credential and
// must say so rather than surfacing a raw 401.
func TestCLI_Items_RequireAuth(t *testing.T) {
	for _, args := range [][]string{
		{"items", "list"}, {"items", "get", testItemID},
		{"items", "update", testItemID, "--visibility", "public"},
	} {
		out, code := runCLIText(t, args...)
		if code == 0 {
			t.Errorf("%v should fail without a credential", args)
		}
		if !strings.Contains(out, "Not authenticated") {
			t.Errorf("%v should print the not-authenticated banner, got: %s", args, out)
		}
	}
}
