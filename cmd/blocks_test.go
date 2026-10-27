// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"net/http"
	"net/http/httptest"
	"slices"
	"strconv"
	"strings"
	"sync"
	"testing"
)

const (
	testBlockID       = "01a06aef-425f-731e-bb35-a692ed2b0b88"
	testBlockHash     = "653b13ae01ad4258bcd9d43065de524523a52d0c2efa4c7467499c1e7fac929c"
	testBlockGenesis  = "01a06ae0-0000-7000-8000-000000000000"
	testBlockGenHash  = "e36f824dea9508d5dca570c77edc06eb4bc830b413b3db4e4d983ce3f85e9d78"
	testBlockEmptyMR  = "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"
	testBlockPrevHash = "5df6e0e2761359d30a8275058e299fcc0381534545f55cf43e41983f5d4c9456"
)

// testBlock is a block as the server's JSON:API serves it: a resource
// object whose attributes are exactly the twelve public attributes of
// Truestamp.Blockchain.Block (lib/truestamp/blockchain/block.ex of
// truestamp-v2, pinned by test/truestamp_web/json_api_blocks_test.exs
// there), height among them.
func testBlock(id, hash string, height int, previousID string) string {
	relationship := `{"links":{},"meta":{}}`
	return `{"type":"block","id":"` + id + `","attributes":{
	  "block_hash":"` + hash + `","height":` + strconv.Itoa(height) + `,"inserted_at":"2026-09-01T00:00:00.000000Z",
	  "leaf_count":0,"merkle_root":"` + testBlockEmptyMR + `","metadata":{},
	  "previous_block_hash":"` + testBlockPrevHash + `","previous_block_id":"` + previousID + `",
	  "signature":"` + strings.Repeat("A", 86) + `","signing_key_id":"3c19f776","state":"committed",
	  "updated_at":"2026-09-01T00:00:00.000000Z"},
	  "relationships":{"previous":` + relationship + `,"next":` + relationship + `,"items":` + relationship + `,
	  "entropy_observations":` + relationship + `,"commitments":` + relationship + `,"leaf_commitments":` + relationship + `},
	  "links":{"self":"http://localhost/api/json/blocks/` + id + `"},"meta":{}}`
}

// startBlocksServer stands in for the two blocks routes, recording the
// sort each list request asked for.
func startBlocksServer(t *testing.T) (*httptest.Server, func() []string) {
	t.Helper()
	var mu sync.Mutex
	var sorts []string
	head := testBlock(testBlockID, testBlockHash, 1234, testBlockGenesis)
	genesis := testBlock(testBlockGenesis, testBlockGenHash, 0, testBlockGenesis)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requireBearer(t, r)
		w.Header().Set("Content-Type", "application/vnd.api+json")
		switch r.URL.Path {
		case "/api/json/blocks/" + testBlockID:
			_, _ = w.Write([]byte(`{"data":` + head + `}`))
		case "/api/json/blocks":
			mu.Lock()
			sorts = append(sorts, r.URL.Query().Get("sort"))
			mu.Unlock()
			_, _ = w.Write([]byte(`{"data":[` + head + `,` + genesis + `],"links":{"next":null,"prev":null},"meta":{"page":{"limit":25}}}`))
		default:
			w.WriteHeader(http.StatusNotFound)
			_, _ = w.Write([]byte(`{"errors":[{"status":"404","title":"Not Found","detail":"no such block"}]}`))
		}
	}))
	t.Cleanup(srv.Close)
	return srv, func() []string {
		mu.Lock()
		defer mu.Unlock()
		return slices.Clone(sorts)
	}
}

// A block's card shows its height under the id, and the list, in height
// order, leads each row with it, the highest first. Mutants killed: the
// card's Height row dropped (its labels change); the list's HEIGHT
// column dropped (its header and rows change); the list asking for
// sort=-id (the recorded sort).
func TestCLI_Blocks_ShowTheHeight(t *testing.T) {
	srv, sorts := startBlocksServer(t)

	stdout, _, exit := runCLI(t, "--base-url", srv.URL, "--api-key", "test-key", "blocks", "get", testBlockID)
	if exit != 0 {
		t.Fatalf("exit=%d\n%s", exit, stdout)
	}
	var labels []string
	var height []string
	for _, line := range strings.Split(stdout, "\n")[2:] {
		if fields := strings.Fields(line); len(fields) > 0 {
			labels = append(labels, fields[0])
			if fields[0] == "Height" {
				height = fields
			}
		}
	}
	want := []string{"ID", "Height", "State", "Block", "Merkle", "Previous", "Signing", "Created"}
	if !slices.Equal(labels, want) || !slices.Equal(height, []string{"Height", "1234"}) {
		t.Errorf("card labels %q, height row %q; want %q and the row Height 1234\n%s", labels, height, want, stdout)
	}

	stdout, _, exit = runCLI(t, "--base-url", srv.URL, "--api-key", "test-key", "blocks", "list")
	if exit != 0 {
		t.Fatalf("exit=%d\n%s", exit, stdout)
	}
	var rows [][]string
	for _, line := range strings.Split(stdout, "\n")[2:] {
		if fields := strings.Fields(line); len(fields) > 0 {
			rows = append(rows, fields)
		}
	}
	wantRows := [][]string{
		{"HEIGHT", "STATE", "ID", "HASH"},
		{"1234", "committed", testBlockID, testBlockHash},
		{"0", "committed", testBlockGenesis, testBlockGenHash},
	}
	if !slices.EqualFunc(rows, wantRows, slices.Equal[[]string]) {
		t.Errorf("list rows %q; want %q\n%s", rows, wantRows, stdout)
	}
	if got := sorts(); !slices.Equal(got, []string{"-height"}) {
		t.Errorf("list sorts %q; want one -height", got)
	}
}
