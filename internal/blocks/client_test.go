// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: Apache-2.0

package blocks

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
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

// TestList_AlwaysSendsALimit: the blocks index route's action requires
// pagination, defaults to 25 and clamps to 250; List sends a limit of 25
// when the caller gives none, so the page size is explicit. Mutant
// killed: defaultLimit set to 26 (page[limit] is "26").
func TestList_AlwaysSendsALimit(t *testing.T) {
	cfg, last := serve(t, `[]`)
	if _, err := List(context.Background(), cfg, ListOptions{}); err != nil {
		t.Fatalf("List: %v", err)
	}
	if got := last.Query().Get("page[limit]"); got != "25" {
		t.Errorf("page[limit] = %q, want 25", got)
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

// serverBlock is a block as the server's JSON:API serves it: a resource
// object whose attributes are exactly the twelve public attributes of
// Truestamp.Blockchain.Block (lib/truestamp/blockchain/block.ex of
// truestamp-v2, pinned by test/truestamp_web/json_api_blocks_test.exs
// there), height among them, and relationship links for its six public
// relationships.
func serverBlock(id string, height int, previousID string) string {
	relationship := `{"links":{},"meta":{}}`
	return `{"type":"block","id":"` + id + `","attributes":{
	  "block_hash":"` + validHash + `","height":` + strconv.Itoa(height) + `,"inserted_at":"2026-09-01T00:00:00.000000Z",
	  "leaf_count":0,"merkle_root":"e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855","metadata":{},
	  "previous_block_hash":"5df6e0e2761359d30a8275058e299fcc0381534545f55cf43e41983f5d4c9456",
	  "previous_block_id":"` + previousID + `","signature":"` + strings.Repeat("A", 86) + `",
	  "signing_key_id":"3c19f776","state":"committed","updated_at":"2026-09-01T00:00:00.000000Z"},
	  "relationships":{"previous":` + relationship + `,"next":` + relationship + `,"items":` + relationship + `,
	  "entropy_observations":` + relationship + `,"commitments":` + relationship + `,"leaf_commitments":` + relationship + `},
	  "links":{"self":"http://localhost/api/json/blocks/` + id + `"},"meta":{}}`
}

// The chain is in height order, and the server sorts by height whatever
// the ids (json_api_blocks_test.exs's "sorts and filters by height"): a
// list asks for the highest first, or the lowest with OldestFirst; the
// head is the highest; genesis, at height 0, the lowest. Mutants killed:
// any of the four sorts left on the id (-id or id), or turned the other
// way.
func TestSorts_AreByHeight(t *testing.T) {
	page := `{"data":[` + serverBlock(validID, 0, validID) + `]}`
	for _, tc := range []struct {
		name string
		call func(Config) error
		want string
	}{
		{"list", func(c Config) error { _, err := List(context.Background(), c, ListOptions{Limit: 5}); return err }, "-height"},
		{"list oldest first", func(c Config) error {
			_, err := List(context.Background(), c, ListOptions{Limit: 5, OldestFirst: true})
			return err
		}, "height"},
		{"latest", func(c Config) error { _, err := Latest(context.Background(), c); return err }, "-height"},
		{"genesis", func(c Config) error { _, err := Genesis(context.Background(), c); return err }, "height"},
	} {
		cfg, last := serve(t, page)
		if err := tc.call(cfg); err != nil {
			t.Fatalf("%s: %v", tc.name, err)
		}
		if got := last.Query().Get("sort"); got != tc.want {
			t.Errorf("%s sorted by %q; want %q", tc.name, got, tc.want)
		}
	}

	// Genesis is identifiable by id == previous_block_id. A lowest row
	// whose previous_block_id is not its own id is refused rather than
	// presented as the root.
	cfg, _ := serve(t, `{"data":[`+serverBlock(validID, 0, "01a06aef-0000-7000-8000-000000000000")+`]}`)
	if _, err := Genesis(context.Background(), cfg); err == nil {
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
			if tc.status == http.StatusTooManyRequests {
				// Over the retry cap, so the transport surfaces the
				// 429 after one attempt instead of sleeping.
				w.Header().Set("Retry-After", "3600")
			}
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

// A block's height is the server's stored, 0-based height, served as
// `height` (block.ex's height attribute: 0 at genesis, then one more per
// block); the server serves no block_height (json_api_blocks_test.exs
// refuses it as a sparse field). The client decodes height from a block,
// and from each block of a list, genesis's 0 among them. Mutants killed:
// the Height field under the tag block_height (every height reads 0);
// the Height field dropped (this does not compile).
func TestHeight_IsTheServersStoredHeight(t *testing.T) {
	cfg, _ := serve(t, `{"data":`+serverBlock(validID, 1234, "01a06aef-0000-7000-8000-000000000000")+`}`)
	b, err := Get(context.Background(), cfg, validID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if b.Height != 1234 || b.ID != validID || b.BlockHash != validHash {
		t.Errorf("decoded height %d, id %q, hash %q; want 1234, %q, %q", b.Height, b.ID, b.BlockHash, validID, validHash)
	}

	cfg, _ = serve(t, `{"data":[`+serverBlock("01a06aef-425f-731e-bb35-a692ed2b0b89", 7, validID)+`,`+serverBlock(validID, 0, validID)+`]}`)
	page, err := List(context.Background(), cfg, ListOptions{Limit: 2})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(page.Blocks) != 2 || page.Blocks[0].Height != 7 || page.Blocks[1].Height != 0 {
		t.Errorf("list heights %+v; want 7 then 0", page.Blocks)
	}
}
