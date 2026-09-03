// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: MIT

package console

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/truestamp/truestamp-cli/internal/console/events"
)

// TestRenderWaterfallShape exercises the 3-column scan view: each
// rendered row carries Time + compact Kind + full ID, and no row
// surfaces the (deleted) Detail content. The headers row is present.
func TestRenderWaterfallShape(t *testing.T) {
	t.Parallel()

	m := newMonitorModel(nil, nil)
	m.focus = focusWaterfall

	rows := []events.Row{
		mustProject(t, "blocks", "block.created", "2026-04-29T13:58:00.123Z", map[string]any{
			"id":         "01KQ88MP3GTM6XNHSCBVR12345",
			"state":      "finalized",
			"block_hash": "f4b30a22775deadbeef000000000000000000000000000000000000000000000000",
		}),
		mustProject(t, "beacons", "beacon.created", "2026-04-29T13:58:00.456Z", map[string]any{
			"id":            "019db702-b08c-73dc-a7cd-2c5e011f1dad",
			"hash":          "ffe86dc05a0c7b42279f7fa6afb016cd6928980d24673051fc58731492ce2a1b",
			"previous_hash": "1c4812bdfec2bf29333136d86bc996f866e38177acc90565a0554c7ec698029b",
		}),
		mustProject(t, "items.team", "item.updated", "2026-04-29T13:58:01.001Z", map[string]any{
			"id":    "01KQ88MP3GTM6IIIIIIIIIIIII",
			"state": "committed",
			"claims": map[string]any{
				"name": "doc-A",
			},
		}),
		mustProject(t, "items.team", "item.burst", "2026-04-29T13:58:01.500Z", map[string]any{
			"count":     27,
			"window_ms": 500,
			"by_kind":   map[string]any{"item.created": 18, "item.updated": 9},
		}),
		mustProject(t, "blocks.healing", "block_healing.forward", "2026-04-29T13:58:02.000Z", map[string]any{
			"block_id":   "01KQ88MP3GTM6BBBBBBBBBBBBB",
			"from_state": "finalized",
			"to_state":   "committed",
			"reason":     "all_commitments_valid",
		}),
	}
	m.events = rows
	m.selected = 2 // item.committed
	m.detailPanelHidden = true

	out := plain(m.renderWaterfall(120, 30))

	// Header row labels
	for _, want := range []string{"Time", "Kind", "ID"} {
		if !strings.Contains(out, want) {
			t.Errorf("header row missing %q\n--- output ---\n%s", want, out)
		}
	}

	// Compact kinds, full set
	for _, want := range []string{"block.finalized", "beacon", "item.committed", "item.burst", "block_healing.forward"} {
		if !strings.Contains(out, want) {
			t.Errorf("waterfall missing compact kind %q", want)
		}
	}

	// Full IDs preserved
	for _, want := range []string{
		"01KQ88MP3GTM6XNHSCBVR12345",           // block ULID
		"019db702-b08c-73dc-a7cd-2c5e011f1dad", // beacon UUIDv7
		"01KQ88MP3GTM6IIIIIIIIIIIII",           // item ULID
		"01KQ88MP3GTM6BBBBBBBBBBBBB",           // healing block ULID
	} {
		if !strings.Contains(out, want) {
			t.Errorf("waterfall missing full ID %q", want)
		}
	}

	// Burst row uses plain integer count, not parens
	if !strings.Contains(out, " 27 ") && !strings.Contains(out, "27\n") {
		t.Errorf("burst count '27' not surfaced as plain integer")
	}
	if strings.Contains(out, "(27 events)") {
		t.Errorf("burst count still uses parens form")
	}

	// Hashes and detail-column references must NOT appear in the row.
	if strings.Contains(out, "hash=") || strings.Contains(out, "block=") || strings.Contains(out, "name=") {
		t.Errorf("detail-style key=value content leaked into the row:\n%s", out)
	}
}

// TestRenderDetailPanelFullPayload: the panel renders the FULL
// untruncated payload of the selected row, every field, with long
// hashes shown in their entirety (possibly across multiple lines).
func TestRenderDetailPanelFullPayload(t *testing.T) {
	t.Parallel()

	fullHash := "ffe86dc05a0c7b42279f7fa6afb016cd6928980d24673051fc58731492ce2a1b"
	fullPrev := "1c4812bdfec2bf29333136d86bc996f866e38177acc90565a0554c7ec698029b"

	m := newMonitorModel(nil, nil)
	m.events = []events.Row{
		mustProject(t, "beacons", "beacon.created", "2026-04-29T13:58:00.456Z", map[string]any{
			"id":            "019db702-b08c-73dc-a7cd-2c5e011f1dad",
			"hash":          fullHash,
			"previous_hash": fullPrev,
			"timestamp":     "2026-04-29T13:58:00.000Z",
		}),
	}
	m.selected = 0

	out := plain(m.renderDetailPanel(160, 12))

	// Panel rule
	if !strings.Contains(out, "Selected") {
		t.Errorf("panel rule missing 'Selected' label:\n%s", out)
	}

	// Synthetic top fields
	for _, want := range []string{
		"kind", "beacon.created", // RawKind preserved
		"id", "019db702-b08c-73dc-a7cd-2c5e011f1dad", // full UUID
		"stream", "beacons",
		"at", "2026-04-29T13:58:00",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("panel missing %q\n--- output ---\n%s", want, out)
		}
	}

	// Full hash (untruncated). On a 160-col panel a 64-char hash
	// fits on one line; assert verbatim presence.
	if !strings.Contains(out, fullHash) {
		t.Errorf("panel did not show full hash; truncation occurred:\n%s", out)
	}
	if !strings.Contains(out, fullPrev) {
		t.Errorf("panel did not show full previous_hash; truncation occurred:\n%s", out)
	}
}

// TestRenderDetailPanelWrapsLongHashOnNarrowPane: with a narrow pane
// width the panel must wrap long values onto continuation lines,
// concatenated, those continuation lines must reproduce the full
// untruncated value.
func TestRenderDetailPanelWrapsLongHashOnNarrowPane(t *testing.T) {
	t.Parallel()

	fullHash := "ffe86dc05a0c7b42279f7fa6afb016cd6928980d24673051fc58731492ce2a1b"

	m := newMonitorModel(nil, nil)
	m.events = []events.Row{
		mustProject(t, "beacons", "beacon.created", "2026-04-29T13:58:00.456Z", map[string]any{
			"id":   "019db702-b08c-73dc-a7cd-2c5e011f1dad",
			"hash": fullHash,
		}),
	}
	m.selected = 0

	out := plain(m.renderDetailPanel(60, 12))

	// On 60 cols the hash will wrap. The bytes are still all there,
	// just split across lines, strip whitespace and concatenate to
	// confirm.
	flat := strings.Join(strings.Fields(out), "")
	if !strings.Contains(flat, fullHash) {
		t.Errorf("wrapped hash does not reconstruct to the full value:\n%s", out)
	}
}

// TestRenderDetailPanelNoSelection: with no events buffered the panel
// shows a hint, not a crash.
func TestRenderDetailPanelNoSelection(t *testing.T) {
	t.Parallel()

	m := newMonitorModel(nil, nil)
	out := plain(m.renderDetailPanel(120, 8))
	if !strings.Contains(out, "Select a row") {
		t.Errorf("expected 'Select a row' hint when nothing is selected, got:\n%s", out)
	}
}

// TestRenderDetailPanelOutageRow: synthetic outage rows render the
// prose context (`message`) and the duration in the panel.
func TestRenderDetailPanelOutageRow(t *testing.T) {
	t.Parallel()

	since := time.Date(2026, 4, 29, 13, 58, 0, 0, time.UTC)
	at := since.Add(8 * time.Second)

	m := newMonitorModel(nil, nil)
	m.events = []events.Row{events.Outage(at, since, false)}
	m.selected = 0

	out := plain(m.renderDetailPanel(120, 8))

	for _, want := range []string{"server.down", "lost", "8s"} {
		if !strings.Contains(out, want) {
			t.Errorf("outage panel missing %q:\n%s", want, out)
		}
	}
}

// TestRenderWaterfallWithDetailPanelOpen: with the panel visible, the
// waterfall body uses fewer rows. Both the table and the panel must
// be present in the output.
func TestRenderWaterfallWithDetailPanelOpen(t *testing.T) {
	t.Parallel()

	m := newMonitorModel(nil, nil)
	m.detailPanelHidden = false

	m.events = []events.Row{
		mustProject(t, "items.team", "item.created", "2026-04-29T13:58:01.001Z", map[string]any{
			"id":    "01KQ88MP3GTM6IIIIIIIIIIIII",
			"state": "created",
			"claims": map[string]any{
				"name": "doc-A",
			},
		}),
	}
	m.selected = 0

	out := plain(m.renderWaterfall(120, 30))

	// Header + row + panel rule + panel field
	for _, want := range []string{"Time", "item", "01KQ88MP3GTM6IIIIIIIIIIIII", "Selected", "kind"} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q in combined render:\n%s", want, out)
		}
	}
}

// =========================================================================
// Helpers
// =========================================================================

func mustProject(t *testing.T, stream, kind, at string, data map[string]any) events.Row {
	t.Helper()
	raw, err := json.Marshal(data)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	return events.Project(time.Date(2026, 4, 29, 13, 58, 0, 0, time.UTC), events.Push{
		Stream: stream,
		Kind:   kind,
		At:     at,
		Data:   raw,
	})
}

// plain returns s with ANSI CSI escape sequences fully removed. The
// existing stripANSI helper in connection_test.go has a tiny bug
// (treats the `[` introducer as the terminator, since 0x5b falls in
// its 0x40–0x7e final-byte range), which leaves SGR parameter bytes
// in the output. This helper does the right thing.
func plain(s string) string {
	var b strings.Builder
	i := 0
	for i < len(s) {
		if s[i] == 0x1b && i+1 < len(s) && s[i+1] == '[' {
			i += 2
			for i < len(s) {
				c := s[i]
				i++
				if c >= 0x40 && c <= 0x7e {
					break
				}
			}
			continue
		}
		b.WriteByte(s[i])
		i++
	}
	return b.String()
}
