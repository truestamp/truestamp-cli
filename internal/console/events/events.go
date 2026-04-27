// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: MIT

// Package events projects raw `stream` payloads from the console wire
// protocol into a canonical Row that drives the Monitor pane's table
// renderer. Every kind of event collapses into the same four-column
// shape (Time / Kind / ID / Detail) so the waterfall reads as a
// uniform stream rather than a per-resource grab bag.
//
// The pre-existing `summarizeEvent` / `summarizeBurst` functions in
// monitor.go produced free-form strings of varying widths and
// duplicated info between the stream tag and the kind prefix. Rows
// from this package are explicitly column-aligned and de-duplicated:
// when the stream id and kind share the same resource prefix (the
// common case) the stream tag is dropped, since it carries no
// additional information.
package events

import (
	"encoding/json"
	"fmt"
	"sort"
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

// Row is the canonical four-column event the Monitor table renders.
// Keep this small — string-only fields, no nested objects — so the
// renderer can hand them straight to lipgloss/v2/table without
// further marshaling.
type Row struct {
	When     time.Time
	Kind     string // e.g. "block.created", "commitment.burst"
	ID       string // full ULID/UUID for the row's primary resource, "(N events)" for bursts, "" for outages
	Detail   string // ≤ 60 chars, schema documented per resource below
	Stream   string // populated only when stream and kind don't share a prefix
	Severity Severity
}

// Push mirrors the wire shape produced by ConsoleProjector +
// ConsoleChannel.push_stream/4. Lifted from monitor.go so the
// projector can live in its own package.
type Push struct {
	Stream string          `json:"stream"`
	Kind   string          `json:"kind"`
	At     string          `json:"at"`
	Data   json.RawMessage `json:"data"`
}

// idPrefixLen is the number of characters of a ULID we surface for
// IDs that appear inside the Detail column (block=…, item=…, etc).
// The PRIMARY ID column carries the full 26-char ULID / 36-char
// UUID without truncation — operators need the whole identifier to
// copy-paste into queries. Detail-column references stay shortened
// so the field doesn't blow past detailMaxLen at typical widths.
const idPrefixLen = 13

// detailMaxLen caps the Detail column. Truncation happens here so
// the renderer doesn't have to know about content; the table layer
// just lays the columns out.
const detailMaxLen = 60

// Project converts a Push into a Row. Returns SeverityNormal rows
// for all individual events, SeverityBurst rows for `*.burst`
// summaries.
//
// The Stream field is populated by per-resource projectors when the
// stream id carries information not already in the kind (today: only
// per-item watches, where the stream is `items.<item_id>` against a
// kind like `item.created`). The renderer prepends `[stream] ` to
// Detail when Stream is non-empty.
func Project(now time.Time, p Push) Row {
	var row Row
	switch {
	case strings.HasSuffix(p.Kind, ".burst"):
		row = projectBurst(now, p)
	case strings.HasPrefix(p.Kind, "block."):
		row = projectBlock(now, p)
	case strings.HasPrefix(p.Kind, "commitment."):
		row = projectCommitment(now, p)
	case strings.HasPrefix(p.Kind, "external_commitment."):
		row = projectExternalCommitment(now, p)
	case strings.HasPrefix(p.Kind, "entropy."):
		row = projectEntropy(now, p)
	case strings.HasPrefix(p.Kind, "item."):
		row = projectItem(now, p)
	default:
		// Unknown kind — surface what we have so the user still
		// sees something useful while we add a projector for it.
		row = Row{
			When:   now,
			Kind:   p.Kind,
			Detail: truncate(string(p.Data), detailMaxLen),
			Stream: streamTag(p.Stream, p.Kind),
		}
	}

	if row.Stream != "" {
		// Prepend the stream tag to Detail so the renderer's table
		// stays a clean 4 columns. Re-truncate to keep the bounded
		// width contract.
		row.Detail = truncate("["+row.Stream+"] "+row.Detail, detailMaxLen)
	}
	return row
}

// Outage builds a synthetic Row for an outage marker. `closing` flips
// the row into a "resumed" line after reconnect.
func Outage(at time.Time, since time.Time, closing bool) Row {
	dur := at.Sub(since).Round(time.Second)
	if closing {
		return Row{
			When:     at,
			Kind:     "server.up",
			ID:       "",
			Detail:   "connection restored after " + dur.String(),
			Severity: SeverityOutage,
		}
	}
	return Row{
		When:     at,
		Kind:     "server.down",
		ID:       "",
		Detail:   "server unreachable for " + dur.String(),
		Severity: SeverityOutage,
	}
}

// =============================================================================
// Per-resource projectors
// =============================================================================

func projectBlock(now time.Time, p Push) Row {
	var d struct {
		ID         string `json:"id"`
		State      string `json:"state"`
		BlockHash  string `json:"block_hash"`
		MerkleRoot string `json:"merkle_root"`
	}
	_ = json.Unmarshal(p.Data, &d)

	parts := []string{}
	if d.State != "" {
		parts = append(parts, "state="+d.State)
	}
	if d.BlockHash != "" {
		parts = append(parts, "hash="+shortHash(d.BlockHash, 12))
	}

	return Row{
		When:   now,
		Kind:   p.Kind,
		ID:     d.ID,
		Detail: truncate(strings.Join(parts, "  "), detailMaxLen),
		Stream: streamTag(p.Stream, p.Kind),
	}
}

func projectCommitment(now time.Time, p Push) Row {
	var d struct {
		ID            string `json:"id"`
		BlockID       string `json:"block_id"`
		ItemID        string `json:"item_id"`
		EntropyObsID  string `json:"entropy_observation_id"`
		CommitmentHsh string `json:"commitment_hash"`
	}
	_ = json.Unmarshal(p.Data, &d)

	// The ID column carries the commitment's own ULID. Detail
	// surfaces the linked resource (block + either item OR entropy
	// observation, whichever this commitment is for).
	parts := []string{}
	if d.BlockID != "" {
		parts = append(parts, "block="+prefix(d.BlockID, idPrefixLen))
	}
	if d.ItemID != "" {
		parts = append(parts, "item="+prefix(d.ItemID, idPrefixLen))
	} else if d.EntropyObsID != "" {
		parts = append(parts, "entropy="+prefix(d.EntropyObsID, idPrefixLen))
	}

	return Row{
		When:   now,
		Kind:   p.Kind,
		ID:     d.ID,
		Detail: truncate(strings.Join(parts, "  "), detailMaxLen),
		Stream: streamTag(p.Stream, p.Kind),
	}
}

func projectExternalCommitment(now time.Time, p Push) Row {
	var d struct {
		ID      string `json:"id"`
		Method  string `json:"method"`
		BlockID string `json:"block_id"`
	}
	_ = json.Unmarshal(p.Data, &d)

	parts := []string{}
	if d.Method != "" {
		parts = append(parts, "method="+d.Method)
	}
	if d.BlockID != "" {
		parts = append(parts, "block="+prefix(d.BlockID, idPrefixLen))
	}

	return Row{
		When:   now,
		Kind:   p.Kind,
		ID:     d.ID,
		Detail: truncate(strings.Join(parts, "  "), detailMaxLen),
		Stream: streamTag(p.Stream, p.Kind),
	}
}

func projectEntropy(now time.Time, p Push) Row {
	var d struct {
		ID              string `json:"id"`
		Source          string `json:"source"`
		State           string `json:"state"`
		ObservationHash string `json:"observation_hash"`
	}
	_ = json.Unmarshal(p.Data, &d)

	parts := []string{}
	if d.Source != "" {
		// Source atom comes prefixed (entropy_nist, entropy_stellar,
		// entropy_bitcoin); strip for display so the column stays
		// readable.
		src := strings.TrimPrefix(d.Source, "entropy_")
		parts = append(parts, "source="+src)
	}
	if d.State != "" && d.State != "created" {
		parts = append(parts, "state="+d.State)
	}
	if d.ObservationHash != "" {
		parts = append(parts, "hash="+shortHash(d.ObservationHash, 16))
	}

	return Row{
		When:   now,
		Kind:   p.Kind,
		ID:     d.ID,
		Detail: truncate(strings.Join(parts, "  "), detailMaxLen),
		Stream: streamTag(p.Stream, p.Kind),
	}
}

func projectItem(now time.Time, p Push) Row {
	var d struct {
		ID     string `json:"id"`
		State  string `json:"state"`
		Claims struct {
			Name string `json:"name"`
		} `json:"claims"`
	}
	_ = json.Unmarshal(p.Data, &d)

	parts := []string{}
	if d.State != "" {
		parts = append(parts, "state="+d.State)
	}
	if d.Claims.Name != "" {
		parts = append(parts, fmt.Sprintf("name=%q", truncate(d.Claims.Name, 32)))
	}

	return Row{
		When:   now,
		Kind:   p.Kind,
		ID:     d.ID,
		Detail: truncate(strings.Join(parts, "  "), detailMaxLen),
		Stream: streamTag(p.Stream, p.Kind),
	}
}

func projectBurst(now time.Time, p Push) Row {
	var d struct {
		Count    int            `json:"count"`
		WindowMs int            `json:"window_ms"`
		ByKind   map[string]int `json:"by_kind"`
	}
	_ = json.Unmarshal(p.Data, &d)

	// Render the per-kind breakdown like `created=14 updated=3` —
	// stripping the resource prefix so just the verb part shows.
	parts := make([]string, 0, len(d.ByKind))
	for k, n := range d.ByKind {
		verb := k
		if i := strings.Index(k, "."); i >= 0 {
			verb = k[i+1:]
		}
		parts = append(parts, fmt.Sprintf("%s=%d", verb, n))
	}
	sort.Strings(parts)

	detail := strings.Join(parts, "  ")
	if d.WindowMs > 0 {
		detail = fmt.Sprintf("%s  window=%dms", detail, d.WindowMs)
	}

	return Row{
		When:     now,
		Kind:     p.Kind,
		ID:       fmt.Sprintf("(%d events)", d.Count),
		Detail:   truncate(detail, detailMaxLen),
		Stream:   streamTag(p.Stream, p.Kind),
		Severity: SeverityBurst,
	}
}

// =============================================================================
// Helpers
// =============================================================================

// streamTag returns the stream id only if it doesn't share a prefix
// with the kind. The common case (e.g. stream `commitments.internal`,
// kind `commitment.created`) collapses to "" so the column can be
// dropped entirely. Per-item watches (stream `items.<id>` against a
// kind `item.created` broadcast on the team-wide topic) DO get a tag
// — the stream id carries the disambiguating item id.
func streamTag(stream, kind string) string {
	if stream == "" {
		return ""
	}
	streamRoot := splitFirst(stream, '.')
	kindRoot := splitFirst(kind, '.')
	// Stream "items.team" vs kind "item.*" — root differs by an "s",
	// covered by the prefix check below. Stream "items.<id>" vs kind
	// "item.created" — same; we still want the tag because the id is
	// useful disambiguation. Treat the per-item-watch case explicitly.
	if strings.HasPrefix(stream, "items.") && stream != "items.team" {
		return stream
	}
	if singularize(streamRoot) == kindRoot {
		return ""
	}
	return stream
}

// singularize trims a trailing "s". Items "commitments.internal" vs
// "commitment.created" share a singular root.
func singularize(s string) string {
	return strings.TrimSuffix(s, "s")
}

func splitFirst(s string, sep rune) string {
	if i := strings.IndexRune(s, sep); i >= 0 {
		return s[:i]
	}
	return s
}

// prefix is shorthand for "the first n chars of s, or all of s if
// it's shorter".
func prefix(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n]
}

// shortHash returns the first n chars of h. Identical to prefix today
// but kept distinct so we can later add a "…" tail if we change our
// mind about silent truncation.
func shortHash(h string, n int) string {
	return prefix(h, n)
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	if n < 1 {
		return ""
	}
	return s[:n-1] + "…"
}
