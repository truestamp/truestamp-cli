// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

// defaultPageLimit is the page size the list commands ask for when --limit
// is not given. It matches the server's own default for every paginated
// read, and it is declared once so the three list commands cannot drift.
const defaultPageLimit = 25

// addLimitFlag registers --limit with one wording and one default across
// the list commands, so a reader learns the flag once.
func addLimitFlag(cmd *cobra.Command, noun string) {
	cmd.Flags().Int("limit", defaultPageLimit,
		"How many "+noun+" per page; the server clamps a page above its ceiling (250) and the listing says so")
}

// pageLimit reads --limit and enforces only the bound the server's published
// contract actually guarantees.
//
// The OpenAPI document at /api/json/open_api declares `page.limit` with
// `"minimum": 1` and no maximum: the string `"maximum"` does not appear
// anywhere in it. The ceiling belongs to the server: every list route,
// beacons included, lowers a page[limit] above its action's max_page_size
// (250) to that size instead of refusing it, and the listing says so
// (paging.go). A ceiling held here as well would be a second source of
// truth that goes stale silently the day the server moves it, so this owns
// only the floor, which the contract does state.
//
// An unset flag yields the default registered by addLimitFlag, which is
// always above the floor, so `--limit 0` is the error it always should
// have been instead of silently meaning "the default".
func pageLimit(cmd *cobra.Command) (int, error) {
	limit, err := cmd.Flags().GetInt("limit")
	if err != nil {
		return 0, err
	}
	if limit < 1 {
		return 0, fmt.Errorf("--limit must be at least 1, got %d", limit)
	}
	return limit, nil
}
