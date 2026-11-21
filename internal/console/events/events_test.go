// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: Apache-2.0

package events

import (
	"encoding/json/jsontext"
	"encoding/json/v2"
	"strings"
	"testing"
	"time"
)

// TestProjectBlock checks the canonical block row: full id in ID,
// compact kind reflecting the state-as-suffix rule.
// Mutant killed: an id cut to 26 characters (the ID is not the whole
// UUIDv7).
func TestProjectBlock(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 4, 27, 16, 39, 0, 0, time.UTC)
	p := makePush(t, "blocks", "block.created", map[string]any{
		"id":         "019dd4a0-6a10-7c3e-9b2f-5d8e1a7c4b30",
		"state":      "finalized",
		"block_hash": "f4b30a22775deadbeef000000000000000000000000000000000000000000000000",
	})

	row := Project(now, p)

	// state=finalized != "created" → kind compacts to block.finalized
	if row.Kind != "block.finalized" {
		t.Errorf("Kind = %q, want block.finalized", row.Kind)
	}
	if row.RawKind != "block.created" {
		t.Errorf("RawKind = %q, want block.created", row.RawKind)
	}
	if got, want := row.ID, "019dd4a0-6a10-7c3e-9b2f-5d8e1a7c4b30"; got != want {
		t.Errorf("ID = %q, want %q (the full id, untruncated)", got, want)
	}
	if row.Severity != SeverityNormal {
		t.Errorf("Severity = %v, want SeverityNormal", row.Severity)
	}
	// Payload preserved verbatim for the Detail Panel.
	if got, _ := row.Payload["block_hash"].(string); got != "f4b30a22775deadbeef000000000000000000000000000000000000000000000000" {
		t.Errorf("Payload missing or wrong block_hash: %v", row.Payload["block_hash"])
	}
}

// TestProjectBlockCreatedDefault: when state matches the default
// "created", the verb is dropped and the kind is just the bare root, and
// the ID is the full id.
// Mutant killed: an id cut to 26 characters (the ID is not the whole
// UUIDv7).
func TestProjectBlockCreatedDefault(t *testing.T) {
	t.Parallel()

	p := makePush(t, "blocks", "block.created", map[string]any{
		"id":    "019dd4a0-6a10-7c3e-9b2f-5d8e1a7c4b30",
		"state": "created",
	})

	row := Project(time.Now(), p)

	if row.Kind != "block" {
		t.Errorf("Kind = %q, want bare 'block' when state=created", row.Kind)
	}
	if row.ID != "019dd4a0-6a10-7c3e-9b2f-5d8e1a7c4b30" {
		t.Errorf("ID = %q, want the full id", row.ID)
	}
}

// TestProjectBeacon: full UUIDv7 in ID; payload preserves all fields
// for the Detail Panel.
func TestProjectBeacon(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 4, 29, 12, 0, 0, 0, time.UTC)
	p := makePush(t, "beacons", "beacon.created", map[string]any{
		"id":            "019db702-b08c-73dc-a7cd-2c5e011f1dad",
		"hash":          "ffe86dc05a0c7b42279f7fa6afb016cd6928980d24673051fc58731492ce2a1b",
		"previous_hash": "1c4812bdfec2bf29333136d86bc996f866e38177acc90565a0554c7ec698029b",
		"timestamp":     "2026-04-29T12:00:00.000000Z",
	})

	row := Project(now, p)

	// beacons have no state field → bare root kind.
	if row.Kind != "beacon" {
		t.Errorf("Kind = %q, want bare 'beacon'", row.Kind)
	}
	if got, want := row.ID, "019db702-b08c-73dc-a7cd-2c5e011f1dad"; got != want {
		t.Errorf("ID = %q, want %q (full UUIDv7)", got, want)
	}
	if row.Severity != SeverityNormal {
		t.Errorf("Severity = %v, want SeverityNormal", row.Severity)
	}
	// Detail Panel needs the full hash and previous_hash, no truncation.
	if got, _ := row.Payload["hash"].(string); got != "ffe86dc05a0c7b42279f7fa6afb016cd6928980d24673051fc58731492ce2a1b" {
		t.Errorf("Payload.hash not preserved verbatim: %v", row.Payload["hash"])
	}
	if got, _ := row.Payload["previous_hash"].(string); got != "1c4812bdfec2bf29333136d86bc996f866e38177acc90565a0554c7ec698029b" {
		t.Errorf("Payload.previous_hash not preserved verbatim: %v", row.Payload["previous_hash"])
	}
}

// commitmentPayload is a row as the server's console projection sends
// it (TruestampWeb.ConsoleProjector.project/1 for a Commitment, in
// lib/truestamp_web/channels/console_projector.ex of truestamp-v2): one
// shape for the three kinds, the links a kind does not have null.
func commitmentPayload(subjectKind string) map[string]any {
	row := map[string]any{
		"id": "019a3f2c-6b1e-7d40-9a51-3c2b1d0e4f60", "subject_kind": subjectKind,
		"tree_kind": "block", "chain": nil, "item_id": nil, "entropy_observation_id": nil,
		"subject_block_id": nil, "tree_block_id": nil, "epoch_id": nil,
		"inserted_at": "2026-10-27T16:00:00.000000Z",
	}
	switch subjectKind {
	case "item":
		row["item_id"], row["tree_block_id"] = "019a3f2c-6a00-7c11-8e22-0f1e2d3c4b5a", "019a3f2c-5f00-7a00-8b00-000000000001"
	case "observation":
		row["entropy_observation_id"], row["tree_block_id"] = "019a3f2c-6900-7b22-9c33-1a2b3c4d5e6f", "019a3f2c-5f00-7a00-8b00-000000000001"
	case "block":
		row["tree_kind"], row["chain"] = "epoch", "stellar"
		row["subject_block_id"], row["epoch_id"] = "019a3f2c-5f00-7a00-8b00-000000000001", "019a3f2c-7000-7d44-8a55-6b7c8d9e0f12"
	}
	return row
}

// TestProjectCommitmentEachSubjectKind: both commitment streams emit
// commitment.created, and the payload's subject_kind says which level
// the row is (TruestampWeb.ConsoleStreams.classify/3, in
// lib/truestamp_web/channels/console_streams.ex of truestamp-v2): an item
// or an observation on commitments.subjects, a block on
// commitments.blocks. Each row's kind names its subject_kind, its id is
// the row's own, and the stream and payload are kept for the panel.
// Mutants killed: compactKind without its commitment branch (every row
// "commitment"); the branch reading tree_kind instead of subject_kind
// (the item and observation rows both "commitment.block").
func TestProjectCommitmentEachSubjectKind(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct{ stream, subjectKind, wantKind string }{
		{"commitments.subjects", "item", "commitment.item"},
		{"commitments.subjects", "observation", "commitment.observation"},
		{"commitments.blocks", "block", "commitment.block"},
	} {
		t.Run(tc.subjectKind, func(t *testing.T) {
			t.Parallel()
			data := commitmentPayload(tc.subjectKind)
			row := Project(time.Now(), makePush(t, tc.stream, "commitment.created", data))
			if row.Kind != tc.wantKind || row.RawKind != "commitment.created" || row.Stream != tc.stream ||
				row.ID != data["id"] || row.Severity != SeverityNormal {
				t.Errorf("row %+v; want kind %q, raw kind commitment.created, stream %q, id %v", row, tc.wantKind, tc.stream, data["id"])
			}
			if row.Payload["subject_kind"] != tc.subjectKind || row.Payload["tree_kind"] != data["tree_kind"] {
				t.Errorf("payload %v; want the server's subject_kind and tree_kind kept", row.Payload)
			}
		})
	}
}

// TestProjectEntropySource: source is encoded into the kind as a
// middle segment; entropy_ prefix is stripped.
// Mutant killed: an id cut to 26 characters (the ID is not the whole
// UUIDv7).
func TestProjectEntropySource(t *testing.T) {
	t.Parallel()

	p := makePush(t, "entropy.stellar", "entropy.created", map[string]any{
		"id":               "019dd4a0-6d48-7f63-8ae5-2a8b4dbf7c39",
		"source":           "entropy_stellar",
		"state":            "created",
		"observation_hash": "812c6486dd17951b3cc4e091530a685b98488b1ffdddab08c867b84d5bc2172d",
	})

	row := Project(time.Now(), p)

	if row.Kind != "entropy.stellar" {
		t.Errorf("Kind = %q, want entropy.stellar", row.Kind)
	}
	if row.ID != "019dd4a0-6d48-7f63-8ae5-2a8b4dbf7c39" {
		t.Errorf("ID = %q, want the entropy id", row.ID)
	}
}

// TestProjectEntropyFailedState: non-default state becomes the suffix,
// and the ID is the full id.
// Mutant killed: an id cut to 26 characters (the ID is not the whole
// UUIDv7).
func TestProjectEntropyFailedState(t *testing.T) {
	t.Parallel()

	p := makePush(t, "entropy.bitcoin", "entropy.updated", map[string]any{
		"id":     "019dd4a0-6d48-7f63-8ae5-2a8b4dbf7c39",
		"source": "entropy_bitcoin",
		"state":  "failed",
	})

	row := Project(time.Now(), p)

	if row.Kind != "entropy.bitcoin.failed" {
		t.Errorf("Kind = %q, want entropy.bitcoin.failed", row.Kind)
	}
	if row.ID != "019dd4a0-6d48-7f63-8ae5-2a8b4dbf7c39" {
		t.Errorf("ID = %q, want the full id", row.ID)
	}
}

// TestProjectItemStateTransitions: item.created (state=created) →
// "item"; item.updated (state=processing) → "item.processing";
// item.updated (state=committed) → "item.committed";
// item.deleted → "item.deleted".
// Mutant killed: an id cut to 26 characters (the ID is not the whole
// UUIDv7).
func TestProjectItemStateTransitions(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name     string
		kind     string
		state    string
		wantKind string
	}{
		{"created default", "item.created", "created", "item"},
		{"updated processing", "item.updated", "processing", "item.processing"},
		{"updated committed", "item.updated", "committed", "item.committed"},
		{"deleted", "item.deleted", "", "item.deleted"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			p := makePush(t, "items.team", tc.kind, map[string]any{
				"id":    "019dd4a0-6b22-7d41-a8c3-0e6f2b9d5a17",
				"state": tc.state,
				"claims": map[string]any{
					"name": "doc-A",
				},
			})
			row := Project(time.Now(), p)
			if row.Kind != tc.wantKind {
				t.Errorf("Kind = %q, want %q", row.Kind, tc.wantKind)
			}
			if row.ID != "019dd4a0-6b22-7d41-a8c3-0e6f2b9d5a17" {
				t.Errorf("ID = %q, want the full id", row.ID)
			}
		})
	}
}

// TestProjectBurst: count goes into ID column as a plain integer
// (no parens); kind gets a .burst suffix.
func TestProjectBurst(t *testing.T) {
	t.Parallel()

	p := makePush(t, "items.team", "item.burst", map[string]any{
		"count":     14,
		"window_ms": 500,
		"by_kind": map[string]int{
			"item.created": 11,
			"item.updated": 3,
		},
	})

	row := Project(time.Now(), p)

	if row.Severity != SeverityBurst {
		t.Errorf("Severity = %v, want SeverityBurst", row.Severity)
	}
	if row.Kind != "item.burst" {
		t.Errorf("Kind = %q, want item.burst", row.Kind)
	}
	if row.ID != "14" {
		t.Errorf("ID = %q, want plain integer count '14'", row.ID)
	}
	// Burst payload preserved for the Detail Panel.
	if n, _ := row.Payload["count"].(float64); int(n) != 14 {
		t.Errorf("Payload.count missing or wrong: %v", row.Payload["count"])
	}
}

// TestProjectEntropyBurstSourceFromStream: entropy bursts get the
// source as a middle segment, derived from the stream id.
func TestProjectEntropyBurstSourceFromStream(t *testing.T) {
	t.Parallel()

	p := makePush(t, "entropy.stellar", "entropy.burst", map[string]any{
		"count":     142,
		"window_ms": 500,
	})

	row := Project(time.Now(), p)

	if row.Kind != "entropy.stellar.burst" {
		t.Errorf("Kind = %q, want entropy.stellar.burst", row.Kind)
	}
	if row.ID != "142" {
		t.Errorf("ID = %q, want plain integer count '142'", row.ID)
	}
}

// TestProjectBlockHealing: server emits block_healing.forward and
// block_healing.reverse on the blocks.healing stream. The CLI now
// projects them with block_id as the ID and the verb preserved in
// the kind.
// Mutant killed: an id cut to 26 characters (the ID is not the whole
// UUIDv7).
func TestProjectBlockHealing(t *testing.T) {
	t.Parallel()

	p := makePush(t, "blocks.healing", "block_healing.forward", map[string]any{
		"block_id":   "019dd4a0-6c35-7e52-b9d4-1f7a3cae6b28",
		"from_state": "finalized",
		"to_state":   "committed",
		"direction":  "forward",
		"reason":     "all_commitments_valid",
		"node":       "truestamp@127.0.0.1",
	})

	row := Project(time.Now(), p)

	if row.Kind != "block_healing.forward" {
		t.Errorf("Kind = %q, want block_healing.forward", row.Kind)
	}
	if row.ID != "019dd4a0-6c35-7e52-b9d4-1f7a3cae6b28" {
		t.Errorf("ID = %q, want block_id (key the row to the block being healed)", row.ID)
	}
	if got, _ := row.Payload["reason"].(string); got != "all_commitments_valid" {
		t.Errorf("Payload.reason missing: %v", row.Payload["reason"])
	}
}

// TestOutageRow: synthetic outage rows use SeverityOutage and put
// the duration in the ID column so the row fits the canonical 3-col
// shape. The Detail Panel uses Payload for the prose context.
func TestOutageRow(t *testing.T) {
	t.Parallel()

	since := time.Date(2026, 4, 27, 16, 38, 0, 0, time.UTC)
	at := since.Add(8 * time.Second)

	down := Outage(at, since, false)
	up := Outage(at, since, true)

	if down.Severity != SeverityOutage || up.Severity != SeverityOutage {
		t.Errorf("expected SeverityOutage on both rows, got down=%v up=%v",
			down.Severity, up.Severity)
	}
	if down.Kind != "server.down" {
		t.Errorf("down.Kind = %q, want server.down", down.Kind)
	}
	if up.Kind != "server.up" {
		t.Errorf("up.Kind = %q, want server.up", up.Kind)
	}
	if down.ID != "8s" {
		t.Errorf("down.ID = %q, want '8s' (duration in ID column)", down.ID)
	}
	if up.ID != "8s" {
		t.Errorf("up.ID = %q, want '8s' (duration in ID column)", up.ID)
	}
	if got, _ := down.Payload["message"].(string); !strings.Contains(got, "lost") {
		t.Errorf("down.Payload.message missing 'lost': %v", down.Payload["message"])
	}
	if got, _ := up.Payload["message"].(string); !strings.Contains(got, "reconnected") {
		t.Errorf("up.Payload.message missing 'reconnected': %v", up.Payload["message"])
	}
}

// TestProjectorPreservesPayload: every projector path must populate
// Row.Payload from the server's data field so the Detail Panel can
// render any field without losing information.
func TestProjectorPreservesPayload(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name   string
		stream string
		kind   string
		data   map[string]any
	}{
		{"block", "blocks", "block.created", map[string]any{"id": "X", "state": "finalized", "block_hash": "abc"}},
		{"beacon", "beacons", "beacon.created", map[string]any{"id": "X", "hash": "abc", "previous_hash": "def"}},
		{"item commitment", "commitments.subjects", "commitment.created", commitmentPayload("item")},
		{"observation commitment", "commitments.subjects", "commitment.created", commitmentPayload("observation")},
		{"block commitment", "commitments.blocks", "commitment.created", commitmentPayload("block")},
		{"entropy", "entropy.nist", "entropy.created", map[string]any{"id": "X", "source": "entropy_nist"}},
		{"item", "items.team", "item.created", map[string]any{"id": "X", "state": "created"}},
		{"healing", "blocks.healing", "block_healing.forward", map[string]any{"block_id": "B", "reason": "ok"}},
		{"burst", "items.team", "item.burst", map[string]any{"count": 5}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			p := makePush(t, tc.stream, tc.kind, tc.data)
			row := Project(time.Now(), p)
			if row.Payload == nil {
				t.Fatalf("%s: Row.Payload is nil", tc.name)
			}
			for k, v := range tc.data {
				got, ok := row.Payload[k]
				if !ok {
					t.Errorf("%s: Payload missing key %q", tc.name, k)
					continue
				}
				// Maps and ints round-trip through JSON in their JSON-decoded
				// forms (float64 / map[string]any), so just compare via
				// json-marshal for shape equality.
				want, _ := json.Marshal(v, json.Deterministic(true))
				gotMar, _ := json.Marshal(got, json.Deterministic(true))
				if string(want) != string(gotMar) {
					t.Errorf("%s: Payload[%q] = %s, want %s", tc.name, k, gotMar, want)
				}
			}
		})
	}
}

// TestProjectAtParse: when Push.At is a parseable RFC3339 timestamp,
// Row.When uses it; otherwise it falls back to the local receive time.
func TestProjectAtParse(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 4, 29, 14, 22, 33, 123_000_000, time.UTC)
	p := Push{
		Stream: "blocks",
		Kind:   "block.created",
		At:     "2026-04-29T13:58:00.456789Z",
		Data:   jsontext.Value(`{"id":"X"}`),
	}

	row := Project(now, p)

	if row.When.UTC().Hour() != 13 || row.When.UTC().Minute() != 58 {
		t.Errorf("Row.When did not parse Push.At: got %v", row.When)
	}
	if !row.ReceivedAt.Equal(now) {
		t.Errorf("Row.ReceivedAt = %v, want %v", row.ReceivedAt, now)
	}
}

// TestProjectAtFallback: bad/empty At falls back to `now`.
func TestProjectAtFallback(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 4, 29, 14, 22, 33, 0, time.UTC)
	p := Push{Stream: "blocks", Kind: "block.created", At: "", Data: jsontext.Value(`{"id":"X"}`)}

	row := Project(now, p)

	if !row.When.Equal(now) {
		t.Errorf("Row.When fallback failed: got %v, want %v", row.When, now)
	}
}

// TestKindCompactionExhaustive locks down every concrete kind token
// the redesign should produce. If a future change widens the kind
// vocabulary the test must be updated explicitly. A commitment burst
// says it is of blocks only on commitments.blocks, whose rows are all
// blocks; on commitments.subjects it may mix items and observations.
// Mutants killed: burstKind without its commitments.blocks branch (the
// blocks burst reads "commitment.burst"); the branch keyed on any
// commitment stream (the subjects burst reads "commitment.block.burst").
func TestKindCompactionExhaustive(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name     string
		stream   string
		kind     string
		state    string
		source   string
		wantKind string
	}{
		{"block created/created", "blocks", "block.created", "created", "", "block"},
		{"block created/finalized", "blocks", "block.created", "finalized", "", "block.finalized"},
		{"block updated/committed", "blocks", "block.updated", "committed", "", "block.committed"},
		{"beacon", "beacons", "beacon.created", "", "", "beacon"},
		{"entropy.nist created", "entropy.nist", "entropy.created", "created", "entropy_nist", "entropy.nist"},
		{"entropy.stellar processing", "entropy.stellar", "entropy.updated", "processing", "entropy_stellar", "entropy.stellar.processing"},
		{"entropy.bitcoin failed", "entropy.bitcoin", "entropy.updated", "failed", "entropy_bitcoin", "entropy.bitcoin.failed"},
		{"item created", "items.team", "item.created", "created", "", "item"},
		{"item processing", "items.team", "item.updated", "processing", "", "item.processing"},
		{"item committed", "items.team", "item.updated", "committed", "", "item.committed"},
		{"item deleted", "items.team", "item.deleted", "", "", "item.deleted"},
		{"healing forward", "blocks.healing", "block_healing.forward", "", "", "block_healing.forward"},
		{"healing reverse", "blocks.healing", "block_healing.reverse", "", "", "block_healing.reverse"},
		{"item.burst", "items.team", "item.burst", "", "", "item.burst"},
		{"entropy.stellar.burst", "entropy.stellar", "entropy.burst", "", "", "entropy.stellar.burst"},
		{"commitments.subjects burst", "commitments.subjects", "commitment.burst", "", "", "commitment.burst"},
		{"commitments.blocks burst", "commitments.blocks", "commitment.burst", "", "", "commitment.block.burst"},
		{"server.down", "", "server.down", "", "", "server.down"},
		{"server.up", "", "server.up", "", "", "server.up"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			data := map[string]any{}
			if tc.state != "" {
				data["state"] = tc.state
			}
			if tc.source != "" {
				data["source"] = tc.source
			}
			// Burst rows need a count for the ID column.
			if strings.HasSuffix(tc.kind, ".burst") {
				data["count"] = 1
			}
			p := makePush(t, tc.stream, tc.kind, data)
			got := compactKind(p.Stream, p.Kind, decodePayload(p.Data))
			if got != tc.wantKind {
				t.Errorf("compactKind(%q, %q, state=%q, source=%q) = %q, want %q",
					tc.stream, tc.kind, tc.state, tc.source, got, tc.wantKind)
			}
		})
	}
}

// TestKindMaxWidth guards the column-width invariant: the longest
// generated kind must fit the table's reserved Kind-column width.
// Update both this constant and the renderer's column width together
// when a longer kind is added.
func TestKindMaxWidth(t *testing.T) {
	t.Parallel()

	const kindColWidth = 30 // must match monitor.go's reserved width

	// Every kind we know how to produce.
	kinds := []string{
		"block", "block.finalized", "block.committed", "block.burst",
		"beacon", "beacon.burst",
		"commitment", "commitment.item", "commitment.observation", "commitment.block",
		"commitment.burst", "commitment.block.burst",
		"entropy.nist", "entropy.nist.processing", "entropy.nist.failed", "entropy.nist.burst",
		"entropy.stellar", "entropy.stellar.processing", "entropy.stellar.failed", "entropy.stellar.burst",
		"entropy.bitcoin", "entropy.bitcoin.processing", "entropy.bitcoin.failed", "entropy.bitcoin.burst",
		"item", "item.processing", "item.committed", "item.deleted", "item.burst",
		"block_healing.forward", "block_healing.reverse",
		"server.down", "server.up",
	}
	for _, k := range kinds {
		if len(k) > kindColWidth {
			t.Errorf("kind %q (%d chars) exceeds kindColWidth=%d", k, len(k), kindColWidth)
		}
	}
}

// makePush serializes a Data map into a Push so the test fixtures
// stay readable.
func makePush(t *testing.T, stream, kind string, data map[string]any) Push {
	t.Helper()
	raw, err := json.Marshal(data, json.Deterministic(true))
	if err != nil {
		t.Fatalf("makePush: marshal data: %v", err)
	}
	return Push{
		Stream: stream,
		Kind:   kind,
		Data:   raw,
	}
}
