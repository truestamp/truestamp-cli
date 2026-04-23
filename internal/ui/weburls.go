// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: MIT

package ui

import "strings"

// This file centralizes the construction of public-web URLs that the
// CLI surfaces in its post-action "card" output (after beacon get,
// download, create, etc.). Two flavours per subject:
//
//   - Details URL — the subject's own detail page (item / block /
//     entropy / beacon). Keyed by id for everything except beacon,
//     which has a hash-keyed detail route the server exposes at
//     /beacons/<hash>. Callers with only the id can still link via the
//     underlying block's page (see SubjectDetailURL's beacon row).
//
//   - Verify URL — the typed sub-path `/verify/<type>/<id>` route
//     introduced in the t=11 cutover. Always id-keyed and always
//     available for all six subject types.
//
// URLs are emitted unconditionally — localhost and plain-http hosts
// render URLs too. The earlier "suppress dev hosts" filter was removed
// so developers can see the card URLs against their local server. The
// corresponding small risk (a dev-host URL in a shared transcript) is
// considered acceptable by the CLI maintainers.

// publicWebBase derives the public web origin from the configured API
// URL by stripping a trailing `/api/json` and any trailing slash.
// Returns the empty string only when apiURL itself is empty.
func publicWebBase(apiURL string) string {
	base := strings.TrimSuffix(apiURL, "/")
	base = strings.TrimSuffix(base, "/api/json")
	return base
}

// subjectDetailPath maps each of the six canonical wire-type values to
// the server's detail-page root. Id-keyed for all five. Beacon shares
// the block's detail page because a beacon proof commits to the same
// block a plain block proof does — pointing `--type beacon` downloads
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
// or create card. `typeName` must be one of the six canonical wire
// values; `id` is the ULID (item) or UUIDv7 (every other type). Returns
// "" for unknown types or dev-host API URLs.
func SubjectDetailURL(apiURL, typeName, id string) string {
	base := publicWebBase(apiURL)
	if base == "" {
		return ""
	}
	path, ok := subjectDetailPath[typeName]
	if !ok {
		return ""
	}
	return base + path + id
}

// SubjectVerifyURL returns the typed sub-path verify URL
// (`{host}/verify/<type>/<id>`). Accepts any of the six canonical
// types; server-side also accepts bare `entropy` as a convenience
// form but this CLI sticks to strict subtypes end-to-end.
func SubjectVerifyURL(apiURL, typeName, id string) string {
	base := publicWebBase(apiURL)
	if base == "" || typeName == "" {
		return ""
	}
	return base + "/verify/" + typeName + "/" + id
}

// BeaconDetailURL is the hash-keyed variant used only by the beacon
// listing card (`truestamp beacon {latest,list,get,by-hash}`), where
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
