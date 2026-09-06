// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: Apache-2.0

package items

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/truestamp/truestamp-cli/internal/auth"
	"github.com/truestamp/truestamp-cli/internal/httpclient"
)

func init() {
	httpclient.Init(0)
	auth.SetDefault(auth.Resolve(
		auth.Credentials{APIKey: "test-key", APIKeyExplicit: true}, auth.Store{}))
}

// recorder captures the request so tests can assert what was asked for,
// which is the only way to catch a silently-missing field selection.
type recorder struct {
	query  url.Values
	body   string
	method string
	path   string
}

func serveItems(t *testing.T, body string) (string, *recorder) {
	t.Helper()
	rec := &recorder{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		buf := make([]byte, r.ContentLength)
		if r.ContentLength > 0 {
			_, _ = r.Body.Read(buf)
		}
		rec.query, rec.body, rec.method, rec.path = r.URL.Query(), string(buf), r.Method, r.URL.Path
		w.Header().Set("Content-Type", "application/vnd.api+json")
		_, _ = w.Write([]byte(body))
	}))
	t.Cleanup(srv.Close)
	return srv.URL, rec
}

const oneItem = `{"data":{"type":"item","id":"01AAA","attributes":{
  "state":"committed","visibility":"private","tags":["q3"],
  "claims_hash":"aa","item_hash":"bb","display_name":"Contract",
  "inserted_at":"2026-08-19T21:41:00Z","expires_at":"2027-08-19T21:41:00Z"}}}`

// TestRequestFields_NamesTheNonDefaultAttributes is the trap this package
// exists to avoid: inserted_at, updated_at and expires_at are absent from
// the resource's json_api default_fields, so a request that does not name
// them returns empty values with NO error. Nothing downstream can tell
// "the server has no value" from "we forgot to ask".
func TestRequestFields_NamesTheNonDefaultAttributes(t *testing.T) {
	for _, want := range []string{"inserted_at", "updated_at", "expires_at"} {
		if !strings.Contains(requestFields, want) {
			t.Errorf("requestFields must name %q", want)
		}
	}
}

func TestList_SendsFieldsAndLimit(t *testing.T) {
	apiURL, rec := serveItems(t, `{"data":[]}`)
	if _, err := List(context.Background(), apiURL, "", ListOptions{}); err != nil {
		t.Fatalf("List: %v", err)
	}
	if got := rec.query.Get("fields[item]"); !strings.Contains(got, "expires_at") {
		t.Errorf("fields[item] not sent or incomplete: %q", got)
	}
	if got := rec.query.Get("page[limit]"); got == "" {
		t.Error("a page limit must always be sent")
	}
}

func TestGet_SendsFields(t *testing.T) {
	apiURL, rec := serveItems(t, oneItem)
	if _, err := Get(context.Background(), apiURL, "", "01AAA"); err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got := rec.query.Get("fields[item]"); !strings.Contains(got, "inserted_at") {
		t.Errorf("Get must ask for the non-default fields too, got %q", got)
	}
}

// TestList_ForwardsAnOversizeLimitToTheServer pins the decision that the
// ceiling belongs to the server. This client used to reject anything over a
// local MaxLimit = 100, but the server's OpenAPI document declares
// page.limit with "minimum": 1 and no maximum anywhere, so that constant was
// an unbacked second source of truth -- copied into three packages -- that
// would go stale silently the day the cap moved. The server refuses an
// over-large page and names its own cap.
func TestList_ForwardsAnOversizeLimitToTheServer(t *testing.T) {
	apiURL, rec := serveItems(t, `{"data":[]}`)
	if _, err := List(context.Background(), apiURL, "", ListOptions{Limit: 5000}); err != nil {
		t.Fatalf("List must forward the limit rather than judging it: %v", err)
	}
	if got := rec.query.Get("page[limit]"); got != "5000" {
		t.Errorf("page[limit] = %q, want the caller's value forwarded verbatim", got)
	}
}

func TestList_CommittedAndPendingConflict(t *testing.T) {
	apiURL, _ := serveItems(t, `{"data":[]}`)
	_, err := List(context.Background(), apiURL, "", ListOptions{Committed: true, Pending: true})
	if err == nil || !strings.Contains(err.Error(), "mutually exclusive") {
		t.Errorf("opposite filters should be refused, got %v", err)
	}
}

// TestList_PendingFiltersClientSide: the server has no "not committed"
// filter, so this is an exclusion after the fetch. Worth pinning because
// it is the one filter whose semantics are not the server's.
func TestList_PendingFiltersClientSide(t *testing.T) {
	body := `{"data":[
	  {"type":"item","id":"01AAA","attributes":{"state":"committed"}},
	  {"type":"item","id":"01BBB","attributes":{"state":"processing"}}]}`
	apiURL, rec := serveItems(t, body)
	page, err := List(context.Background(), apiURL, "", ListOptions{Pending: true})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if rec.query.Get("filter[state]") != "" {
		t.Error("there is no server-side pending filter; none should be sent")
	}
	if len(page.Items) != 1 || page.Items[0].ID != "01BBB" {
		t.Errorf("committed items should be excluded, got %+v", page.Items)
	}
}

func TestList_CommittedFiltersServerSide(t *testing.T) {
	apiURL, rec := serveItems(t, `{"data":[]}`)
	if _, err := List(context.Background(), apiURL, "", ListOptions{Committed: true}); err != nil {
		t.Fatalf("List: %v", err)
	}
	if got := rec.query.Get("filter[state]"); got != "committed" {
		t.Errorf("committed should filter server-side, got %q", got)
	}
}

// TestCursorFromLink extracts the cursor from the server's own next link
// rather than synthesising one, so paging follows what the server says.
func TestList_SurfacesNextCursor(t *testing.T) {
	body := `{"data":[],"links":{"next":"https://x/items?page%5Bafter%5D=CUR"}}`
	apiURL, _ := serveItems(t, body)
	page, err := List(context.Background(), apiURL, "", ListOptions{})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if page.NextCursor != "CUR" {
		t.Errorf("next cursor: got %q", page.NextCursor)
	}
}

func TestParseOne_LiftsAttributes(t *testing.T) {
	apiURL, _ := serveItems(t, oneItem)
	it, err := Get(context.Background(), apiURL, "", "01AAA")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if it.ID != "01AAA" || it.State != "committed" || it.DisplayName != "Contract" {
		t.Errorf("attributes not lifted out of the resource object: %+v", it)
	}
	if it.ExpiresAt == "" {
		t.Error("expires_at was requested and returned; it must survive parsing")
	}
	if !it.Committed() {
		t.Error("Committed() should be true for a committed item")
	}
}

func TestCommitted_OnlyForCommittedState(t *testing.T) {
	for state, want := range map[string]bool{
		"committed": true, "processing": false, "created": false, "": false,
	} {
		if got := (Item{State: state}).Committed(); got != want {
			t.Errorf("state %q: Committed() = %v, want %v", state, got, want)
		}
	}
}

// TestUpdate_SendsOnlyWhatChanged is the safety property. The server
// accepts exactly team_id, visibility and tags; anything else the caller
// did not ask for would be a silent write.
func TestUpdate_SendsOnlyWhatChanged(t *testing.T) {
	apiURL, rec := serveItems(t, oneItem)
	vis := "public"
	if _, err := Update(context.Background(), apiURL, "", "01AAA",
		UpdateOptions{Visibility: &vis}); err != nil {
		t.Fatalf("Update: %v", err)
	}
	if rec.method != http.MethodPatch {
		t.Errorf("want PATCH, got %s", rec.method)
	}
	var sent struct {
		Data struct {
			Attributes map[string]any `json:"attributes"`
		} `json:"data"`
	}
	if err := json.Unmarshal([]byte(rec.body), &sent); err != nil {
		t.Fatalf("body is not JSON: %v (%s)", err, rec.body)
	}
	if len(sent.Data.Attributes) != 1 || sent.Data.Attributes["visibility"] != "public" {
		t.Errorf("only the changed attribute should be sent, got %v", sent.Data.Attributes)
	}
}

func TestUpdate_EmptyIsRejectedBeforeSending(t *testing.T) {
	var called bool
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
	}))
	defer srv.Close()
	_, err := Update(context.Background(), srv.URL, "", "01AAA", UpdateOptions{})
	if err == nil {
		t.Fatal("an update with nothing to change should fail")
	}
	if !strings.Contains(err.Error(), "--to-team") {
		t.Errorf("the error should name the flags, got %q", err)
	}
	if called {
		t.Error("a no-op update must not reach the server")
	}
}

// TestUpdateOptions_EmptyDistinguishesUnsetFromZero: a pointer to an
// empty string is a request to set it empty, not an absence. Collapsing
// the two is how `--team ""` became "move to team ”".
func TestUpdateOptions_EmptyDistinguishesUnsetFromZero(t *testing.T) {
	if !(UpdateOptions{}).Empty() {
		t.Error("no fields set should be Empty")
	}
	blank := ""
	if (UpdateOptions{TeamID: &blank}).Empty() {
		t.Error("a pointer to an empty string is a deliberate value, not an absence")
	}
}

func TestTenantHeaderSentOnlyWhenScoped(t *testing.T) {
	var tenant string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		tenant = r.Header.Get("tenant")
		_, _ = w.Write([]byte(`{"data":[]}`))
	}))
	defer srv.Close()

	_, _ = List(context.Background(), srv.URL, "", ListOptions{})
	if tenant != "" {
		t.Errorf("an empty team must not send a tenant header, got %q", tenant)
	}
	_, _ = List(context.Background(), srv.URL, "team-1", ListOptions{})
	if tenant != "team-1" {
		t.Errorf("tenant header: got %q", tenant)
	}
}

// TestList_SortsNewestFirst.
//
// `items list --help` promises "List items, newest first", and its two
// sibling list verbs deliver it: internal/blocks sends sort=-id. items
// sent no sort at all, so the server applied its default ascending order
// and this was the one list command in the tree that answered
// oldest-first while documenting the opposite in three places.
func TestList_SortsNewestFirst(t *testing.T) {
	apiURL, rec := serveItems(t, `{"data":[]}`)
	if _, err := List(context.Background(), apiURL, "", ListOptions{}); err != nil {
		t.Fatalf("List: %v", err)
	}
	if got := rec.query.Get("sort"); got != "-id" {
		t.Errorf("sort = %q, want %q so the newest item comes back first", got, "-id")
	}
}

// TestList_KeepsSortingWhenFollowingACursor.
//
// List does not follow the server's `next` link; it rebuilds the query
// and lifts only the cursor out. That makes re-supplying `sort` this
// client's job on every page, and this pins it: drop the parameter from
// the cursor path and page two silently reverts to ascending order.
func TestList_KeepsSortingWhenFollowingACursor(t *testing.T) {
	apiURL, rec := serveItems(t, `{"data":[]}`)
	_, err := List(context.Background(), apiURL, "", ListOptions{After: "cursor-from-page-one"})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if got := rec.query.Get("sort"); got != "-id" {
		t.Errorf("sort = %q on a cursor page, want %q", got, "-id")
	}
	if got := rec.query.Get("page[after]"); got != "cursor-from-page-one" {
		t.Errorf("page[after] = %q, want the cursor to still be sent", got)
	}
}
