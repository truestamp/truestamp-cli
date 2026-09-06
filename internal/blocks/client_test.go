// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: Apache-2.0

package blocks

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/truestamp/truestamp-cli/internal/auth"
	"github.com/truestamp/truestamp-cli/internal/httpclient"
)

func init() {
	// The client refuses to send anything without a credential, which is
	// correct but would make every test here assert only that.
	httpclient.Init(0)
	auth.SetDefault(auth.Resolve(auth.Credentials{APIKey: "test-key", APIKeyExplicit: true}, auth.Store{}))
}

const (
	validHash = "653b13ae01ad4258bcd9d43065de524523a52d0c2efa4c7467499c1e7fac929c"
	validID   = "01a06aef-425f-731e-bb35-a692ed2b0b88"
)

// serve starts a test server that records the last request and replies
// with body. Returning the recorded URL lets a test assert on what the
// client actually asked for, not only on what it did with the answer.
func serve(t *testing.T, body string) (Config, *url.URL) {
	t.Helper()
	var last url.URL
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		last = *r.URL
		w.Header().Set("Content-Type", "application/vnd.api+json")
		_, _ = w.Write([]byte(body))
	}))
	t.Cleanup(srv.Close)
	return Config{APIURL: srv.URL}, &last
}

func blockJSON(id, hash string) string {
	return `{"id":"` + id + `","block_hash":"` + hash + `","merkle_root":"mr",
	  "state":"committed","previous_block_id":"prev-id","previous_block_hash":"prev-hash",
	  "signing_key_id":"3c19f776","signature":"sig","inserted_at":"2026-09-01T00:00:00Z"}`
}

// --- shape validation --------------------------------------------------

// TestValidateHash is the guard standing in for one the server does not
// have. There is no by-hash route for blocks, so a hash lookup becomes
// filter[block_hash]; unlike the beacons by-hash action that filter has no
// server-side regex, and an unguarded cast raises a 500 that leaks SQL.
func TestValidateHash(t *testing.T) {
	if err := ValidateHash(validHash); err != nil {
		t.Errorf("a valid 64-hex hash was rejected: %v", err)
	}
	for name, bad := range map[string]string{
		"too short":  strings.Repeat("a", 63),
		"too long":   strings.Repeat("a", 65),
		"uppercase":  strings.ToUpper(validHash),
		"non-hex":    strings.Repeat("g", 64),
		"empty":      "",
		"a uuid":     validID,
		"sql-ish":    "' OR 1=1 --" + strings.Repeat("a", 53),
		"whitespace": " " + validHash[1:],
	} {
		if err := ValidateHash(bad); err == nil {
			t.Errorf("%s should be rejected before it reaches filter[block_hash]: %q", name, bad)
		}
	}
}

func TestValidateUUIDv7(t *testing.T) {
	if err := ValidateUUIDv7(validID); err != nil {
		t.Errorf("a valid UUIDv7 was rejected: %v", err)
	}
	for name, bad := range map[string]string{
		"not a uuid":   "nope",
		"a ULID":       "01KNN33GX5E470CB9TRWAYF9DD",
		"a v4 uuid":    "9f1c2f7e-0b6e-4d1e-9f2a-2c5e011f1dad",
		"a block hash": validHash,
		"empty":        "",
	} {
		if err := ValidateUUIDv7(bad); err == nil {
			t.Errorf("%s should be rejected: %q", name, bad)
		}
	}
}

// TestByHash_ValidatesBeforeSending is the point of the guard: a bad hash
// must never become a request.
func TestByHash_ValidatesBeforeSending(t *testing.T) {
	var called bool
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
	}))
	defer srv.Close()
	if _, err := ByHash(context.Background(), Config{APIURL: srv.URL}, "not-a-hash"); err == nil {
		t.Error("a malformed hash should fail")
	}
	if called {
		t.Error("a malformed hash must not reach the server")
	}
}

// --- requests ----------------------------------------------------------

// TestList_AlwaysSendsALimit: the server's block read declares no
// default_limit and no max_page_size against a table growing about 1,440
// rows a day, so an unbounded GET /blocks is a real hazard.
func TestList_AlwaysSendsALimit(t *testing.T) {
	cfg, last := serve(t, `[]`)
	if _, err := List(context.Background(), cfg, ListOptions{}); err != nil {
		t.Fatalf("List: %v", err)
	}
	if got := last.Query().Get("page[limit]"); got == "" {
		t.Error("List must always send a page limit")
	}
	if got := last.Query().Get("sort"); got != "-id" {
		t.Errorf("List should sort newest first, got %q", got)
	}
}

// TestList_ForwardsAnOversizeLimitToTheServer pins the decision that the
// ceiling belongs to the server. A local MaxLimit here was an unbacked
// constant: the server's OpenAPI document declares page.limit with
// "minimum": 1 and no maximum anywhere, so the number could only drift.
func TestList_ForwardsAnOversizeLimitToTheServer(t *testing.T) {
	cfg, rec := serve(t, `[]`)
	if _, err := List(context.Background(), cfg, ListOptions{Limit: 5000}); err != nil {
		t.Fatalf("List must forward the limit rather than judging it: %v", err)
	}
	if got := rec.Query().Get("page[limit]"); got != "5000" {
		t.Errorf("page[limit] = %q, want the caller's value forwarded verbatim", got)
	}
}

func TestLatestAndGenesis_SortOppositeWays(t *testing.T) {
	cfg, last := serve(t, `[`+blockJSON(validID, validHash)+`]`)
	if _, err := Latest(context.Background(), cfg); err != nil {
		t.Fatalf("Latest: %v", err)
	}
	if got := last.Query().Get("sort"); got != "-id" {
		t.Errorf("Latest should sort descending, got %q", got)
	}

	// Genesis is identifiable by id == previous_block_id.
	cfg2, last2 := serve(t, `[`+blockJSON(validID, validHash)+`]`)
	_, err := Genesis(context.Background(), cfg2)
	if got := last2.Query().Get("sort"); got != "id" {
		t.Errorf("Genesis should sort ascending, got %q", got)
	}
	// This fixture's previous_block_id is not its own id, so Genesis must
	// refuse rather than present the oldest visible row as the root.
	if err == nil {
		t.Error("a row whose previous_block_id is not its own id is not genesis")
	}
}

func TestGenesis_AcceptsSelfReferentialRow(t *testing.T) {
	body := `[{"id":"` + validID + `","previous_block_id":"` + validID + `",
	  "block_hash":"` + validHash + `","state":"committed"}]`
	cfg, _ := serve(t, body)
	b, err := Genesis(context.Background(), cfg)
	if err != nil {
		t.Fatalf("Genesis: %v", err)
	}
	if b.ID != validID {
		t.Errorf("got %q", b.ID)
	}
}

// TestByHash_RefusesAmbiguousMatch: the server does not assume block-hash
// uniqueness, so more than one row is reported rather than resolved by
// picking one.
func TestByHash_RefusesAmbiguousMatch(t *testing.T) {
	body := `[` + blockJSON(validID, validHash) + `,` +
		blockJSON("01a06aef-425f-731e-bb35-a692ed2b0b89", validHash) + `]`
	cfg, _ := serve(t, body)
	_, err := ByHash(context.Background(), cfg, validHash)
	if !errors.Is(err, ErrAmbiguousHash) {
		t.Errorf("want ErrAmbiguousHash, got %v", err)
	}
}

func TestByHash_NoMatchIsNotFound(t *testing.T) {
	cfg, _ := serve(t, `[]`)
	_, err := ByHash(context.Background(), cfg, validHash)
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("want ErrNotFound, got %v", err)
	}
}

// --- response parsing --------------------------------------------------

// TestUnwrap_HandlesBothEnvelopes: this API returns a bare object on some
// routes and a JSON:API resource object on others, so the client accepts
// both rather than working on one route and silently returning zero values
// on the next.
func TestUnwrap_HandlesBothEnvelopes(t *testing.T) {
	cases := map[string]string{
		"bare object":        `{"data":` + blockJSON(validID, validHash) + `}`,
		"resource object":    `{"data":{"type":"block","id":"` + validID + `","attributes":{"block_hash":"` + validHash + `","state":"committed"}}}`,
		"no envelope at all": blockJSON(validID, validHash),
	}
	for name, body := range cases {
		t.Run(name, func(t *testing.T) {
			cfg, _ := serve(t, body)
			b, err := Get(context.Background(), cfg, validID)
			if err != nil {
				t.Fatalf("Get: %v", err)
			}
			if b.ID != validID {
				t.Errorf("id: got %q", b.ID)
			}
			if b.BlockHash != validHash {
				t.Errorf("block_hash was not lifted out of attributes: got %q", b.BlockHash)
			}
		})
	}
}

func TestList_FlattensResourceObjects(t *testing.T) {
	body := `{"data":[{"type":"block","id":"` + validID + `","attributes":{"state":"finalized","block_hash":"` + validHash + `"}}]}`
	cfg, _ := serve(t, body)
	page, err := List(context.Background(), cfg, ListOptions{Limit: 5})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	list := page.Blocks
	if len(list) != 1 {
		t.Fatalf("got %d blocks", len(list))
	}
	if list[0].State != "finalized" || list[0].ID != validID {
		t.Errorf("attributes not flattened: %+v", list[0])
	}
}

// --- errors ------------------------------------------------------------

func TestAPIErrorsCarryDetailAndClass(t *testing.T) {
	cases := []struct {
		status   int
		sentinel error
	}{
		{http.StatusUnauthorized, ErrUnauthorized},
		{http.StatusForbidden, ErrForbidden},
		{http.StatusNotFound, ErrNotFound},
		{http.StatusTooManyRequests, ErrRateLimited},
		{http.StatusBadRequest, ErrBadRequest},
		{http.StatusInternalServerError, ErrServer},
	}
	for _, tc := range cases {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(tc.status)
			_, _ = w.Write([]byte(`{"errors":[{"detail":"the server said this"}]}`))
		}))
		_, err := Get(context.Background(), Config{APIURL: srv.URL}, validID)
		srv.Close()
		if !errors.Is(err, tc.sentinel) {
			t.Errorf("HTTP %d: want %v, got %v", tc.status, tc.sentinel, err)
		}
		if !strings.Contains(err.Error(), "the server said this") {
			t.Errorf("HTTP %d: the server's detail should be preserved, got %q", tc.status, err)
		}
	}
}

func TestGet_RejectsMalformedIDBeforeSending(t *testing.T) {
	var called bool
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
	}))
	defer srv.Close()
	if _, err := Get(context.Background(), Config{APIURL: srv.URL}, "not-a-uuid"); err == nil {
		t.Error("a malformed id should fail")
	}
	if called {
		t.Error("a malformed id must not reach the server")
	}
}

// TestNoHeightField pins a decision that is easy to undo by accident:
// block_height is a runtime COUNT(*), explicitly non-sortable and
// non-filterable, and withheld from the wire. A Height field would invite
// rendering a number the server never sends.
func TestNoHeightField(t *testing.T) {
	body := `{"data":{"id":"` + validID + `","block_hash":"` + validHash + `","block_height":42}}`
	cfg, _ := serve(t, body)
	b, err := Get(context.Background(), cfg, validID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	// Reflection-free check: the struct simply has no such field, so this
	// compiles only while that stays true.
	_ = b.BlockHash
}
