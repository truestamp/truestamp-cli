// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: MIT

package events

import (
	"encoding/json"
	"strings"
	"testing"
	"time"
)

// TestProjectBlock checks the canonical Block row shape: id is
// 13-char prefix, detail surfaces state + truncated hash.
func TestProjectBlock(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 4, 27, 16, 39, 0, 0, time.UTC)
	p := makePush(t, "blocks", "block.created", map[string]any{
		"id":         "01KQ88MP3GTM6XNHSCBVR12345",
		"state":      "finalized",
		"block_hash": "f4b30a22775deadbeef000000000000000000000000000000000000000000000000",
	})

	row := Project(now, p)

	if row.Kind != "block.created" {
		t.Errorf("Kind = %q, want block.created", row.Kind)
	}
	if got, want := row.ID, "01KQ88MP3GTM6"; got != want {
		t.Errorf("ID = %q, want %q (13-char prefix)", got, want)
	}
	if !strings.Contains(row.Detail, "state=finalized") {
		t.Errorf("Detail missing state= field: %q", row.Detail)
	}
	if !strings.Contains(row.Detail, "hash=f4b30a22775d") {
		t.Errorf("Detail missing 12-char hash prefix: %q", row.Detail)
	}
	if row.Severity != SeverityNormal {
		t.Errorf("Severity = %v, want SeverityNormal", row.Severity)
	}
}

// TestProjectCommitmentDropsRedundantStream verifies the "stream tag
// dropped when stream and kind share a resource prefix" invariant:
// stream `commitments.internal` against kind `commitment.created`
// should produce a Detail that does NOT prefix with `[commitments.internal]`.
func TestProjectCommitmentDropsRedundantStream(t *testing.T) {
	t.Parallel()

	p := makePush(t, "commitments.internal", "commitment.created", map[string]any{
		"id":       "01KQ88MP3GTM6CCCCCCCCCCCCC",
		"block_id": "01KQ88MP3GTM6BBBBBBBBBBBBB",
		"item_id":  "01KQ88MP3GTM6IIIIIIIIIIIII",
	})

	row := Project(time.Now(), p)

	if strings.Contains(row.Detail, "[commitments.internal]") {
		t.Errorf("Detail leaked redundant stream tag: %q", row.Detail)
	}
	if !strings.Contains(row.Detail, "block=") {
		t.Errorf("Detail missing block= reference: %q", row.Detail)
	}
}

// TestProjectItemPerItemWatchKeepsStreamTag confirms the per-item
// watch case: stream `items.<id>` against kind `item.updated` carries
// a useful disambiguator (the item id), so the stream tag SHOULD be
// surfaced as a `[stream]` prefix on Detail.
func TestProjectItemPerItemWatchKeepsStreamTag(t *testing.T) {
	t.Parallel()

	p := makePush(t, "items.01KQ88MP3GTM6IIIIIIIIIIIII", "item.updated", map[string]any{
		"id":    "01KQ88MP3GTM6IIIIIIIIIIIII",
		"state": "processing",
		"claims": map[string]any{
			"name": "doc-A",
		},
	})

	row := Project(time.Now(), p)

	if !strings.HasPrefix(row.Detail, "[items.01KQ88MP3GTM6IIIIIIIIIIIII]") {
		t.Errorf("expected per-item stream tag prefix in Detail, got %q", row.Detail)
	}
}

// TestProjectItemTeamStreamDropsTag verifies that the items.team
// stream against an item.* kind drops the redundant tag (item.* on
// items.team is the implied common case).
func TestProjectItemTeamStreamDropsTag(t *testing.T) {
	t.Parallel()

	p := makePush(t, "items.team", "item.created", map[string]any{
		"id":    "01KQ88MP3GTM6IIIIIIIIIIIII",
		"state": "created",
		"claims": map[string]any{
			"name": "doc-A",
		},
	})

	row := Project(time.Now(), p)

	if strings.Contains(row.Detail, "[items.team]") {
		t.Errorf("items.team tag leaked into Detail: %q", row.Detail)
	}
}

// TestProjectBurst verifies bursts get a normalised count badge in
// the ID column and a histogram in Detail.
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
	if row.ID != "(14 events)" {
		t.Errorf("ID = %q, want %q", row.ID, "(14 events)")
	}
	for _, want := range []string{"created=11", "updated=3", "window=500ms"} {
		if !strings.Contains(row.Detail, want) {
			t.Errorf("burst Detail missing %q: %q", want, row.Detail)
		}
	}
}

// TestProjectEntropy strips the entropy_ source prefix and surfaces
// the truncated hash.
func TestProjectEntropy(t *testing.T) {
	t.Parallel()

	p := makePush(t, "entropy.stellar", "entropy.created", map[string]any{
		"id":               "01KQ88MP3GTM6EEEEEEEEEEEEEE",
		"source":           "entropy_stellar",
		"observation_hash": "812c6486dd17951b3cc4e091530a685b98488b1ffdddab08c867b84d5bc2172d",
	})

	row := Project(time.Now(), p)

	if !strings.Contains(row.Detail, "source=stellar") {
		t.Errorf("Detail should strip entropy_ prefix: %q", row.Detail)
	}
	if !strings.Contains(row.Detail, "hash=812c6486dd17951b") {
		t.Errorf("Detail missing 16-char entropy hash prefix: %q", row.Detail)
	}
}

// TestOutageRowFormatting verifies outage rows use SeverityOutage and
// produce a sensible duration string.
func TestOutageRowFormatting(t *testing.T) {
	t.Parallel()

	since := time.Date(2026, 4, 27, 16, 38, 0, 0, time.UTC)
	at := since.Add(8 * time.Second)

	down := Outage(at, since, false)
	up := Outage(at, since, true)

	if down.Severity != SeverityOutage || up.Severity != SeverityOutage {
		t.Errorf("expected SeverityOutage on both rows, got down=%v up=%v",
			down.Severity, up.Severity)
	}
	if !strings.Contains(down.Detail, "8s") {
		t.Errorf("down marker missing duration: %q", down.Detail)
	}
	if !strings.Contains(up.Detail, "restored") {
		t.Errorf("up marker missing 'restored': %q", up.Detail)
	}
}

// TestDetailRespectsMaxLen ensures pathological input doesn't blow
// past the column budget.
func TestDetailRespectsMaxLen(t *testing.T) {
	t.Parallel()

	huge := strings.Repeat("a", 1000)
	p := makePush(t, "items.team", "item.created", map[string]any{
		"id":    "01KQ88MP3GTM6IIIIIIIIIIIII",
		"state": "created",
		"claims": map[string]any{
			"name": huge,
		},
	})

	row := Project(time.Now(), p)

	if got := len(row.Detail); got > detailMaxLen {
		t.Errorf("Detail exceeded max len: %d > %d (%q)", got, detailMaxLen, row.Detail)
	}
}

// makePush serializes a Data map into a Push so the test fixtures
// stay readable. The wire shape's `at` field isn't exercised by the
// projector (the renderer carries its own timestamp), so we leave
// it empty here.
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
