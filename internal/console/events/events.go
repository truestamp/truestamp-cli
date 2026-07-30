// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: MIT

// Package events projects raw `stream` payloads from the console wire
// protocol into a canonical Row that drives the Monitor pane's table
// renderer. The waterfall renders a 3-column scan view (Time, Kind,
// ID); every other field of the server payload is preserved verbatim
// on Row.Payload so the Detail Panel below the waterfall can render
// the full record without truncation when a row is selected.
//
// The "smart compact Kind" encodes verb + state + (entropy) source
// into a single dotted token so the row tells the operator what
// happened without needing a Details column. Examples:
//
//	block.created  with state=finalized          → "block.finalized"
//	item.created   with state=created            → "item"
//	item.updated   with state=processing         → "item.processing"
//	item.deleted                                  → "item.deleted"
//	entropy.created  source=entropy_nist          → "entropy.nist"
//	entropy.updated  source=entropy_stellar
//	                 state=failed                 → "entropy.stellar.failed"
//	item.burst                                    → "item.burst"
//	entropy.burst    stream=entropy.bitcoin       → "entropy.bitcoin.burst"
//	block_healing.forward                         → "block_healing.forward"
//	server.down / server.up                       → unchanged
//
// The original server kind is preserved on Row.RawKind for the panel.
package events

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// Severity classifies a row's visual treatment. Normal rows render in
// the resource's accent color; Burst rows get a dim count badge in
// the ID column; Outage rows render in the error color and italic.
type Severity int

const (
	SeverityNormal Severity = iota
	SeverityBurst
	SeverityOutage
)

// Row is the canonical event the Monitor table renders. The waterfall
// uses Time / Kind / ID; the Detail Panel uses RawKind / Stream / At
// plus every field of Payload.
type Row struct {
	When       time.Time // server emit time (parsed from Push.At, falls back to ReceivedAt)
	ReceivedAt time.Time // local receive time — used for "(received +Nms)" skew display
	Kind       string    // compact display kind (e.g. "entropy.nist.failed")
	RawKind    string    // original server kind (e.g. "entropy.updated")
	ID         string    // full ULID/UUID, count for bursts, duration for outages
	Stream     string    // original stream id (e.g. "commitments.internal")
	Severity   Severity
	Payload    map[string]any // full decoded server payload (Push.Data) — used by Detail Panel
}

// Push mirrors the wire shape produced by ConsoleProjector +
// ConsoleChannel.push_stream/4.
type Push struct {
	Stream string          `json:"stream"`
	Kind   string          `json:"kind"`
	At     string          `json:"at"`
	Data   json.RawMessage `json:"data"`
}

// Project converts a Push into a Row. `now` is the local receive time
// used as a fallback when the server's `at` field is missing or
// unparseable, and as the ReceivedAt anchor for skew display in the
// Detail Panel.
func Project(now time.Time, p Push) Row {
	when := now
	if t, err := time.Parse(time.RFC3339Nano, p.At); err == nil {
		when = t
	} else if t, err := time.Parse(time.RFC3339, p.At); err == nil {
		when = t
	}

	payload := decodePayload(p.Data)

	row := Row{
		When:       when,
		ReceivedAt: now,
		RawKind:    p.Kind,
		Stream:     p.Stream,
		Payload:    payload,
		Severity:   SeverityNormal,
	}

	// Severity for bursts is set up-front; ID and the synthetic suffix
	// for entropy bursts come from the per-resource projector below.
	if strings.HasSuffix(p.Kind, ".burst") {
		row.Severity = SeverityBurst
	}

	row.Kind = compactKind(p.Stream, p.Kind, payload)
	row.ID = projectID(p.Kind, payload)
	return row
}

// Outage builds a synthetic Row for a connection outage marker. The
// duration is rendered in the ID column so the row fits the canonical
// 3-column shape; the panel renders the prose context.
func Outage(at time.Time, since time.Time, closing bool) Row {
	dur := at.Sub(since).Round(time.Second)
	kind := "server.down"
	message := "The CLI lost its WebSocket connection to the Truestamp server."
	if closing {
		kind = "server.up"
		message = "The CLI reconnected to the Truestamp server."
	}
	return Row{
		When:       at,
		ReceivedAt: at,
		Kind:       kind,
		RawKind:    kind,
		ID:         dur.String(),
		Severity:   SeverityOutage,
		Payload: map[string]any{
			"kind":     kind,
			"at":       at.UTC().Format(time.RFC3339Nano),
			"since":    since.UTC().Format(time.RFC3339Nano),
			"duration": dur.String(),
			"message":  message,
		},
	}
}

// =============================================================================
// Kind compaction
// =============================================================================

// compactKind collapses <resource>.<verb> + state + (entropy) source
// into a single dotted token. See package doc for the full decision
// table.
func compactKind(stream, kind string, payload map[string]any) string {
	if kind == "" {
		return ""
	}

	// Synthetic / pass-through kinds.
	if strings.HasPrefix(kind, "server.") {
		return kind
	}

	// Bursts: <root>.burst, with entropy source spliced in from stream.
	if strings.HasSuffix(kind, ".burst") {
		return burstKind(stream, kind)
	}

	// block_healing.{forward,reverse} — verb carries the meaning
	// directly; no state field is meaningful here.
	if strings.HasPrefix(kind, "block_healing.") {
		return kind
	}

	root, verb := splitKind(kind)
	state := stringField(payload, "state")

	// Entropy: source becomes the middle segment; state (when
	// non-default) becomes the suffix.
	if root == "entropy" {
		source := strings.TrimPrefix(stringField(payload, "source"), "entropy_")
		if source == "" {
			source = entropySourceFromStream(stream)
		}
		base := "entropy"
		if source != "" {
			base = "entropy." + source
		}
		if state != "" && state != "created" {
			return base + "." + state
		}
		return base
	}

	// item.deleted (and any future verb without a state field):
	// preserve the verb as the suffix.
	if verb == "deleted" {
		return kind
	}

	// verb=created: drop the verb when state is also "created" or
	// absent (the implicit default). Otherwise the resource was born
	// in a non-default state — surface it (e.g. "block.finalized").
	if verb == "created" {
		if state == "" || state == "created" {
			return root
		}
		return root + "." + state
	}

	// verb=updated: replace verb with the new state when known
	// (e.g. "item.processing"). Fallback to the original kind if
	// state is unavailable so we never render a bare "item.updated"
	// silently.
	if verb == "updated" {
		if state != "" {
			return root + "." + state
		}
		return kind
	}

	// Any other verb (forward-compatibility for new server events):
	// pass through unchanged.
	return kind
}

func burstKind(stream, kind string) string {
	root, _ := splitKind(kind)
	if root == "entropy" {
		source := entropySourceFromStream(stream)
		if source != "" {
			return "entropy." + source + ".burst"
		}
	}
	return kind
}

func entropySourceFromStream(stream string) string {
	// streams: "entropy.nist", "entropy.stellar", "entropy.bitcoin"
	if rest, ok := strings.CutPrefix(stream, "entropy."); ok {
		return rest
	}
	return ""
}

// splitKind returns (root, verb) from "root.verb", splitting on the
// last dot. Multi-word resource names (e.g.
// "external_commitment.created", "block_healing.forward") split
// correctly into root="external_commitment", verb="created", and a
// three-segment kind such as "entropy.nist.failed" yields
// root="entropy.nist", verb="failed".
func splitKind(kind string) (string, string) {
	if i := strings.LastIndex(kind, "."); i >= 0 {
		return kind[:i], kind[i+1:]
	}
	return kind, ""
}

// =============================================================================
// ID extraction
// =============================================================================

// projectID returns the value that goes into the ID column. For
// individual events that's the resource's primary identifier (full
// ULID or UUIDv7). For bursts it's the integer count rendered as a
// plain number. The renderer styles burst counts differently from
// real IDs.
func projectID(kind string, payload map[string]any) string {
	if strings.HasSuffix(kind, ".burst") {
		if n, ok := numField(payload, "count"); ok {
			return fmt.Sprintf("%d", n)
		}
		return ""
	}

	// block_healing rows are keyed to the block being healed.
	if strings.HasPrefix(kind, "block_healing.") {
		if v := stringField(payload, "block_id"); v != "" {
			return v
		}
	}

	if v := stringField(payload, "id"); v != "" {
		return v
	}
	return ""
}

// =============================================================================
// Helpers
// =============================================================================

func decodePayload(raw json.RawMessage) map[string]any {
	if len(raw) == 0 {
		return map[string]any{}
	}
	var m map[string]any
	if err := json.Unmarshal(raw, &m); err != nil {
		return map[string]any{}
	}
	if m == nil {
		return map[string]any{}
	}
	return m
}

func stringField(m map[string]any, key string) string {
	if m == nil {
		return ""
	}
	v, ok := m[key]
	if !ok || v == nil {
		return ""
	}
	if s, ok := v.(string); ok {
		return s
	}
	return ""
}

func numField(m map[string]any, key string) (int64, bool) {
	if m == nil {
		return 0, false
	}
	v, ok := m[key]
	if !ok || v == nil {
		return 0, false
	}
	switch n := v.(type) {
	case float64:
		return int64(n), true
	case int:
		return int64(n), true
	case int64:
		return n, true
	case json.Number:
		i, err := n.Int64()
		if err != nil {
			return 0, false
		}
		return i, true
	}
	return 0, false
}
