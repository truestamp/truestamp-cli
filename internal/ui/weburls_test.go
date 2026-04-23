// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: MIT

package ui

import "testing"

// TestSubjectDetailURL covers the six-way mapping from wire-type to
// server detail-page path plus the dev-host suppression rules (shared
// with every *URL helper in this file).
func TestSubjectDetailURL(t *testing.T) {
	const (
		ulid = "01HJHB01T8FYZ7YTR9P5N62K5B"
		uuid = "019db702-b08c-73dc-a7cd-2c5e011f1dad"
		prod = "https://www.truestamp.com/api/json"
	)
	cases := []struct {
		name         string
		apiURL, kind string
		id           string
		want         string
	}{
		{"item", prod, "item", ulid, "https://www.truestamp.com/items/" + ulid},
		{"entropy_nist", prod, "entropy_nist", uuid, "https://www.truestamp.com/entropy/" + uuid},
		{"entropy_stellar", prod, "entropy_stellar", uuid, "https://www.truestamp.com/entropy/" + uuid},
		{"entropy_bitcoin", prod, "entropy_bitcoin", uuid, "https://www.truestamp.com/entropy/" + uuid},
		{"block", prod, "block", uuid, "https://www.truestamp.com/blocks/" + uuid},
		// Beacon downloads have only the id, so the link routes to the
		// underlying block's page — valid, the beacon proof commits to
		// that block. /beacons/<hash> lives on the beacon listing card.
		{"beacon routes to /blocks/<id>", prod, "beacon", uuid, "https://www.truestamp.com/blocks/" + uuid},

		// Unknown type yields empty (only path-level failure).
		{"unknown type", prod, "bogus", ulid, ""},
		{"empty type", prod, "", ulid, ""},

		// Dev hosts render URLs too — no suppression. Users asked to
		// see local URLs when pointing at their dev server; the small
		// transcript-leak risk is accepted.
		{"plain http renders", "http://www.truestamp.com/api/json", "item", ulid, "http://www.truestamp.com/items/" + ulid},
		{"localhost renders", "https://localhost:4000/api/json", "item", ulid, "https://localhost:4000/items/" + ulid},
		{"127.0.0.1 renders", "https://127.0.0.1:4000/api/json", "item", ulid, "https://127.0.0.1:4000/items/" + ulid},
		{"localhost plain http renders", "http://localhost:4000/api/json", "beacon", uuid, "http://localhost:4000/blocks/" + uuid},

		// Trailing slash + api/json stripping — covered uniformly via publicWebBase.
		{"trailing slash", "https://www.truestamp.com/api/json/", "item", ulid, "https://www.truestamp.com/items/" + ulid},
		{"no /api/json suffix", "https://www.truestamp.com", "block", uuid, "https://www.truestamp.com/blocks/" + uuid},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := SubjectDetailURL(c.apiURL, c.kind, c.id); got != c.want {
				t.Errorf("SubjectDetailURL(%q, %q, %q) = %q, want %q", c.apiURL, c.kind, c.id, got, c.want)
			}
		})
	}
}

// TestSubjectVerifyURL covers the verify-page URL for all six canonical
// types. Path pattern is uniform (`/verify/<type>/<id>`) so the coverage
// matrix is simpler than the detail map.
func TestSubjectVerifyURL(t *testing.T) {
	const (
		ulid = "01HJHB01T8FYZ7YTR9P5N62K5B"
		uuid = "019db702-b08c-73dc-a7cd-2c5e011f1dad"
		prod = "https://www.truestamp.com/api/json"
	)
	cases := []struct {
		kind, id, want string
	}{
		{"item", ulid, "https://www.truestamp.com/verify/item/" + ulid},
		{"entropy_nist", uuid, "https://www.truestamp.com/verify/entropy_nist/" + uuid},
		{"entropy_stellar", uuid, "https://www.truestamp.com/verify/entropy_stellar/" + uuid},
		{"entropy_bitcoin", uuid, "https://www.truestamp.com/verify/entropy_bitcoin/" + uuid},
		{"block", uuid, "https://www.truestamp.com/verify/block/" + uuid},
		{"beacon", uuid, "https://www.truestamp.com/verify/beacon/" + uuid},
	}
	for _, c := range cases {
		if got := SubjectVerifyURL(prod, c.kind, c.id); got != c.want {
			t.Errorf("SubjectVerifyURL(%q, %q) = %q, want %q", c.kind, c.id, got, c.want)
		}
	}

	// localhost renders (no suppression); only the empty-type case
	// returns the empty string.
	if got := SubjectVerifyURL("https://localhost:4000/api/json", "item", ulid); got != "https://localhost:4000/verify/item/"+ulid {
		t.Errorf("localhost should render: got %q", got)
	}
	if got := SubjectVerifyURL(prod, "", ulid); got != "" {
		t.Errorf("empty type should yield empty URL, got %q", got)
	}
}
