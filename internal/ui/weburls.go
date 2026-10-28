// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: Apache-2.0

package ui

import (
	"strings"

	"github.com/truestamp/truestamp-cli/internal/itemcode"
)

// This file centralizes the construction of public-web URLs that the
// CLI surfaces in its post-action "card" output (after beacons get,
// download, create, etc.). Two flavours per subject:
//
//   - Details URL, the subject's own detail page (item / block /
//     entropy / beacon). An item's is keyed by its item code, the one
//     form its page canonicalizes to (/items/<CODE>); every other
//     subject's by its id, except beacon, which has a hash-keyed detail
//     route the server exposes at /beacons/<hash>. Callers with only the
//     id can still link via the underlying block's page (see
//     SubjectDetailURL's beacon row).
//
//   - Verify URL, the typed sub-path `/verify/<type>/<ref>` route,
//     available for all six subject types: an item's is keyed by its
//     item code (/verify/item/<CODE>, the page's canonical URL), every
//     other subject's by its id.
//
// URLs are emitted for every host, localhost and plain-http included, so
// developers see the card URLs against a local server. The small risk
// this carries, a dev-host URL in a shared transcript, is accepted.

// publicWebBase derives the public web origin from the configured API
// URL by stripping a trailing `/api/json` and any trailing slash.
// Returns the empty string only when apiURL itself is empty.
func publicWebBase(apiURL string) string {
	base := strings.TrimSuffix(apiURL, "/")
	base = strings.TrimSuffix(base, "/api/json")
	return base
}

// subjectDetailPath maps each of the six canonical wire-type values to
// the server's detail-page root. Id-keyed for all six. Beacon shares
// the block's detail page because a beacon proof commits to the same
// block a plain block proof does, pointing `--type beacon` downloads
// at `/blocks/<id>` is accurate and uses an id we have on hand (the
// hash-keyed `/beacons/<hash>` form requires computing the block hash
// from bundle bytes, which the beacon-listing card already does with
// the hash it received from the API).
var subjectDetailPath = map[string]string{
	"item":            "/items/",
	"entropy_nist":    "/entropy/",
	"entropy_stellar": "/entropy/",
	"entropy_bitcoin": "/entropy/",
	"block":           "/blocks/",
	"beacon":          "/blocks/",
}

// SubjectDetailURL returns the subject detail page URL for a download
// or items create card. `typeName` must be one of the six canonical wire
// values; `ref` is an item's item code (SPEC.md section 3.9 (The item
// code)), from the API or computed from its recomputed item_hash, and
// every other subject's UUIDv7. Returns "" for an unknown type, an empty
// apiURL, or an item ref that is not an item code in its one spelling,
// so no item page is ever linked by its id, a short form or a typo. Dev
// hosts are not filtered: see the package comment above.
func SubjectDetailURL(apiURL, typeName, ref string) string {
	base := publicWebBase(apiURL)
	if base == "" || !pageRef(typeName, ref) {
		return ""
	}
	path, ok := subjectDetailPath[typeName]
	if !ok {
		return ""
	}
	return base + path + ref
}

// SubjectVerifyURL returns the typed sub-path verify URL
// (`{host}/verify/<type>/<ref>`), `ref` as SubjectDetailURL takes it: an
// item's item code in its one spelling, and every other subject's id.
// Accepts any of the six canonical types; server-side also accepts bare
// `entropy` as a convenience form but this CLI sticks to strict subtypes
// end-to-end. Returns "" for an empty type or apiURL, and for an item ref
// that is not an item code in its one spelling.
func SubjectVerifyURL(apiURL, typeName, ref string) string {
	base := publicWebBase(apiURL)
	if base == "" || typeName == "" || !pageRef(typeName, ref) {
		return ""
	}
	return base + "/verify/" + typeName + "/" + ref
}

// pageRef reports whether ref can name a page of a subject of typeName:
// for an item, only its item code in its one spelling, which
// itemcode.Parse gives back unchanged; for every other type, any ref.
func pageRef(typeName, ref string) bool {
	if typeName != "item" {
		return true
	}
	code, err := itemcode.Parse(ref)
	return err == nil && code == ref
}

// BeaconDetailURL is the hash-keyed variant used only by the beacon
// listing card (`truestamp beacons {latest,list,get}`), where
// the hash is already in hand from the API response. Distinct from
// SubjectDetailURL's beacon row (which points at the underlying
// block's page when only the id is available).
func BeaconDetailURL(apiURL, hash string) string {
	base := publicWebBase(apiURL)
	if base == "" {
		return ""
	}
	return base + "/beacons/" + hash
}

// BeaconVerifyURL is a typed alias for readability at the beacon-card
// call site. Equivalent to SubjectVerifyURL(apiURL, "beacon", id).
func BeaconVerifyURL(apiURL, id string) string {
	return SubjectVerifyURL(apiURL, "beacon", id)
}

// TeamDetailURL returns the public-web team detail page URL,
// `{host}/teams/<id>`. Used by `truestamp teams get`. (The empty-state
// hint of `truestamp teams list` uses TeamCreateURL instead.)
func TeamDetailURL(apiURL, teamID string) string {
	base := publicWebBase(apiURL)
	if base == "" || teamID == "" {
		return ""
	}
	return base + "/teams/" + teamID
}

// TeamCreateURL returns the public-web teams URL. The CLI now creates teams
// itself (truestamp teams create), so this is used as the "manage teams &
// plans" pointer in the plan-limit-reached hint rather than as the only
// creation path.
func TeamCreateURL(apiURL string) string {
	base := publicWebBase(apiURL)
	if base == "" {
		return ""
	}
	return base + "/teams"
}
