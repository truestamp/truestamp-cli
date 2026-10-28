// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: Apache-2.0

package ui

import "testing"

// TestSubjectDetailURL covers the six-way mapping from wire-type to
// server detail-page path, and that every host renders URLs, localhost
// and plain http included, as every *URL helper in this file does. An
// item's page is named by its item code. Mutant killed: pageRef refusing
// every item ref (the item rows are empty).
func TestSubjectDetailURL(t *testing.T) {
	const (
		item = "WX495M26THGQBJ68"
		uuid = "019db702-b08c-73dc-a7cd-2c5e011f1dad"
		prod = "https://www.truestamp.com/api/json"
	)
	cases := []struct {
		name         string
		apiURL, kind string
		id           string
		want         string
	}{
		{"item", prod, "item", item, "https://www.truestamp.com/items/" + item},
		{"entropy_nist", prod, "entropy_nist", uuid, "https://www.truestamp.com/entropy/" + uuid},
		{"entropy_stellar", prod, "entropy_stellar", uuid, "https://www.truestamp.com/entropy/" + uuid},
		{"entropy_bitcoin", prod, "entropy_bitcoin", uuid, "https://www.truestamp.com/entropy/" + uuid},
		{"block", prod, "block", uuid, "https://www.truestamp.com/blocks/" + uuid},
		// Beacon downloads have only the id, so the link routes to the
		// underlying block's page, valid, the beacon proof commits to
		// that block. /beacons/<hash> lives on the beacon listing card.
		{"beacon routes to /blocks/<id>", prod, "beacon", uuid, "https://www.truestamp.com/blocks/" + uuid},

		// Unknown type yields empty (only path-level failure).
		{"unknown type", prod, "bogus", item, ""},
		{"empty type", prod, "", item, ""},

		// Dev hosts render URLs too, no suppression. Users asked to
		// see local URLs when pointing at their dev server; the small
		// transcript-leak risk is accepted.
		{"plain http renders", "http://www.truestamp.com/api/json", "item", item, "http://www.truestamp.com/items/" + item},
		{"localhost renders", "https://localhost:4000/api/json", "item", item, "https://localhost:4000/items/" + item},
		{"127.0.0.1 renders", "https://127.0.0.1:4000/api/json", "item", item, "https://127.0.0.1:4000/items/" + item},
		{"localhost plain http renders", "http://localhost:4000/api/json", "beacon", uuid, "http://localhost:4000/blocks/" + uuid},

		// Trailing slash + api/json stripping, covered uniformly via publicWebBase.
		{"trailing slash", "https://www.truestamp.com/api/json/", "item", item, "https://www.truestamp.com/items/" + item},
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
// types. Path pattern is uniform (`/verify/<type>/<ref>`) so the coverage
// matrix is simpler than the detail map; an item's ref is its item code.
// Mutant killed: pageRef refusing every item ref (the item rows are
// empty).
func TestSubjectVerifyURL(t *testing.T) {
	const (
		item = "WX495M26THGQBJ68"
		uuid = "019db702-b08c-73dc-a7cd-2c5e011f1dad"
		prod = "https://www.truestamp.com/api/json"
	)
	cases := []struct {
		kind, id, want string
	}{
		{"item", item, "https://www.truestamp.com/verify/item/" + item},
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
	if got := SubjectVerifyURL("https://localhost:4000/api/json", "item", item); got != "https://localhost:4000/verify/item/"+item {
		t.Errorf("localhost should render: got %q", got)
	}
	if got := SubjectVerifyURL(prod, "", item); got != "" {
		t.Errorf("empty type should yield empty URL, got %q", got)
	}
}

// An item's pages are named only by its item code in its one spelling:
// its UUIDv7 id, its item_hash, the code in lowercase or with separators,
// a typo and the empty ref give no URL, for the detail page and the verify
// page alike, while every other subject's id still does. Mutants killed:
// pageRef taking any item ref (the id, the hash and the loose spellings
// give a URL); pageRef taking a code in any spelling Parse reads (the
// lowercase and hyphenated codes give a URL); pageRef refusing every
// other type's ref (the block's URLs are empty).
func TestSubjectURLs_AnItemPageIsNamedByItsCode(t *testing.T) {
	const prod = "https://www.truestamp.com/api/json"
	for _, ref := range []string{
		"01a0db66-c2e0-78a4-b8cc-12b11bc97493",
		"e74892d046d46175c8dbdecbbbc705a36816df58e65a2ee1861c100c16eb5f8c",
		"wx495m26thgqbj68", "WX49-5M26-THGQ-BJ68", "WX495M26THGQBJ69", "",
	} {
		if got := SubjectDetailURL(prod, "item", ref); got != "" {
			t.Errorf("SubjectDetailURL(item, %q) = %q, want none", ref, got)
		}
		if got := SubjectVerifyURL(prod, "item", ref); got != "" {
			t.Errorf("SubjectVerifyURL(item, %q) = %q, want none", ref, got)
		}
	}
	if got, want := SubjectDetailURL(prod, "item", "WX495M26THGQBJ68"), "https://www.truestamp.com/items/WX495M26THGQBJ68"; got != want {
		t.Errorf("SubjectDetailURL(item, code) = %q, want %q", got, want)
	}
	if got, want := SubjectVerifyURL(prod, "item", "WX495M26THGQBJ68"), "https://www.truestamp.com/verify/item/WX495M26THGQBJ68"; got != want {
		t.Errorf("SubjectVerifyURL(item, code) = %q, want %q", got, want)
	}
	const block = "019db702-b08c-73dc-a7cd-2c5e011f1dad"
	if got, want := SubjectVerifyURL(prod, "block", block), "https://www.truestamp.com/verify/block/"+block; got != want {
		t.Errorf("SubjectVerifyURL(block) = %q, want %q", got, want)
	}
	if got, want := SubjectDetailURL(prod, "block", block), "https://www.truestamp.com/blocks/"+block; got != want {
		t.Errorf("SubjectDetailURL(block) = %q, want %q", got, want)
	}
}
