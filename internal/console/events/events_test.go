// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: MIT

package events

import (
	"encoding/json"
	"strings"
	"testing"
	"time"
)

// TestProjectBlock checks the canonical block row: full ULID in ID,
// compact kind reflecting the state-as-suffix rule.
func TestProjectBlock(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 4, 27, 16, 39, 0, 0, time.UTC)
	p := makePush(t, "blocks", "block.created", map[string]any{
		"id":         "01KQ88MP3GTM6XNHSCBVR12345",
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
	if got, want := row.ID, "01KQ88MP3GTM6XNHSCBVR12345"; got != want {
		t.Errorf("ID = %q, want %q (full ULID, untruncated)", got, want)
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
// "created", the verb is dropped and the kind is just the bare root.
func TestProjectBlockCreatedDefault(t *testing.T) {
	t.Parallel()

	p := makePush(t, "blocks", "block.created", map[string]any{
		"id":    "01KQ88MP3GTM6XNHSCBVR12345",
		"state": "created",
	})

	row := Project(time.Now(), p)

	if row.Kind != "block" {
		t.Errorf("Kind = %q, want bare 'block' when state=created", row.Kind)
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

// TestProjectCommitment: id is the commitment's own ULID; payload
// carries block_id and item_id for the Detail Panel.
func TestProjectCommitment(t *testing.T) {
	t.Parallel()

	p := makePush(t, "commitments.internal", "commitment.created", map[string]any{
		"id":       "01KQ88MP3GTM6CCCCCCCCCCCCC",
		"block_id": "01KQ88MP3GTM6BBBBBBBBBBBBB",
		"item_id":  "01KQ88MP3GTM6IIIIIIIIIIIII",
	})

	row := Project(time.Now(), p)

	if row.Kind != "commitment" {
		t.Errorf("Kind = %q, want bare 'commitment'", row.Kind)
	}
	if row.ID != "01KQ88MP3GTM6CCCCCCCCCCCCC" {
		t.Errorf("ID = %q, want commitment ULID", row.ID)
	}
	if row.Stream != "commitments.internal" {
		t.Errorf("Stream = %q, want commitments.internal", row.Stream)
	}
	if got, _ := row.Payload["block_id"].(string); got != "01KQ88MP3GTM6BBBBBBBBBBBBB" {
		t.Errorf("Payload.block_id missing: %v", row.Payload["block_id"])
	}
}

// TestProjectExternalCommitment: ULID id, method preserved on payload.
func TestProjectExternalCommitment(t *testing.T) {
	t.Parallel()

	p := makePush(t, "commitments.external", "external_commitment.created", map[string]any{
		"id":       "01KQ88MP3GTM6XXXXXXXXXXXXX",
		"method":   "stellar",
		"block_id": "01KQ88MP3GTM6BBBBBBBBBBBBB",
		"epoch_id": "01KQ88MP3GTM6EPOCHEPOCHEPO",
	})

	row := Project(time.Now(), p)

	if row.Kind != "external_commitment" {
		t.Errorf("Kind = %q, want bare 'external_commitment'", row.Kind)
	}
	if row.ID != "01KQ88MP3GTM6XXXXXXXXXXXXX" {
		t.Errorf("ID = %q, want external commitment ULID", row.ID)
	}
	if got, _ := row.Payload["epoch_id"].(string); got != "01KQ88MP3GTM6EPOCHEPOCHEPO" {
		t.Errorf("Payload.epoch_id missing: %v", row.Payload["epoch_id"])
	}
}

// TestProjectEntropySource: source is encoded into the kind as a
// middle segment; entropy_ prefix is stripped.
func TestProjectEntropySource(t *testing.T) {
	t.Parallel()

	p := makePush(t, "entropy.stellar", "entropy.created", map[string]any{
		"id":               "01KQ88MP3GTM6EEEEEEEEEEEEEE",
		"source":           "entropy_stellar",
		"state":            "created",
		"observation_hash": "812c6486dd17951b3cc4e091530a685b98488b1ffdddab08c867b84d5bc2172d",
	})

	row := Project(time.Now(), p)

	if row.Kind != "entropy.stellar" {
		t.Errorf("Kind = %q, want entropy.stellar", row.Kind)
	}
	if row.ID != "01KQ88MP3GTM6EEEEEEEEEEEEEE" {
		t.Errorf("ID = %q, want entropy ULID", row.ID)
	}
}

// TestProjectEntropyFailedState: non-default state becomes the suffix.
func TestProjectEntropyFailedState(t *testing.T) {
	t.Parallel()

	p := makePush(t, "entropy.bitcoin", "entropy.updated", map[string]any{
		"id":     "01KQ88MP3GTM6EEEEEEEEEEEEEE",
		"source": "entropy_bitcoin",
		"state":  "failed",
	})

	row := Project(time.Now(), p)

	if row.Kind != "entropy.bitcoin.failed" {
		t.Errorf("Kind = %q, want entropy.bitcoin.failed", row.Kind)
	}
}

// TestProjectItemStateTransitions: item.created (state=created) →
// "item"; item.updated (state=processing) → "item.processing";
// item.updated (state=committed) → "item.committed";
// item.deleted → "item.deleted".
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
				"id":    "01KQ88MP3GTM6IIIIIIIIIIIII",
				"state": tc.state,
				"claims": map[string]any{
					"name": "doc-A",
				},
			})
			row := Project(time.Now(), p)
			if row.Kind != tc.wantKind {
				t.Errorf("Kind = %q, want %q", row.Kind, tc.wantKind)
			}
			if row.ID != "01KQ88MP3GTM6IIIIIIIIIIIII" {
				t.Errorf("ID = %q, want full ULID", row.ID)
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
func TestProjectBlockHealing(t *testing.T) {
	t.Parallel()

	p := makePush(t, "blocks.healing", "block_healing.forward", map[string]any{
		"block_id":   "01KQ88MP3GTM6BBBBBBBBBBBBB",
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
	if row.ID != "01KQ88MP3GTM6BBBBBBBBBBBBB" {
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
		{"commitment", "commitments.internal", "commitment.created", map[string]any{"id": "X", "block_id": "B", "item_id": "I"}},
		{"external", "commitments.external", "external_commitment.created", map[string]any{"id": "X", "method": "stellar"}},
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
				want, _ := json.Marshal(v)
				gotMar, _ := json.Marshal(got)
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
		Data:   json.RawMessage(`{"id":"X"}`),
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
	p := Push{Stream: "blocks", Kind: "block.created", At: "", Data: json.RawMessage(`{"id":"X"}`)}

	row := Project(now, p)

	if !row.When.Equal(now) {
		t.Errorf("Row.When fallback failed: got %v, want %v", row.When, now)
	}
}

// TestKindCompactionExhaustive locks down every concrete kind token
// the redesign should produce. If a future change widens the kind
// vocabulary the test must be updated explicitly.
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
		{"commitment", "commitments.internal", "commitment.created", "", "", "commitment"},
		{"external_commitment", "commitments.external", "external_commitment.created", "", "", "external_commitment"},
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
		"commitment", "commitment.burst",
		"external_commitment", "external_commitment.burst",
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
	raw, err := json.Marshal(data)
	if err != nil {
		t.Fatalf("makePush: marshal data: %v", err)
	}
	return Push{
		Stream: stream,
		Kind:   kind,
		Data:   raw,
	}
}
