// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: MIT

package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

// pageLimit reads --limit and enforces only the bound the server's published
// contract actually guarantees.
//
// The OpenAPI document at /api/json/open_api declares `page.limit` with
// `"minimum": 1` and no maximum: the string `"maximum"` does not appear
// anywhere in it. The 100-row ceiling is real, but it lives only in the
// server's runtime validation, so a constant here is a second and unbacked
// source of truth that goes stale silently the day the server moves it --
// and it had already been copied into three packages. The server owns the
// ceiling and names its own cap when it refuses
// ("must be less than or equal to 100"); this owns the floor, which the
// contract does state.
//
// An unset flag keeps its default rather than being validated, so a caller
// that never passes --limit is unaffected, and `--limit 0` is now the error
// it always should have been instead of silently meaning "the default".
func pageLimit(cmd *cobra.Command, def int) (int, error) {
	if !cmd.Flags().Changed("limit") {
		return def, nil
	}
	limit, err := cmd.Flags().GetInt("limit")
	if err != nil {
		return 0, err
	}
	if limit < 1 {
		return 0, fmt.Errorf("--limit must be at least 1, got %d", limit)
	}
	return limit, nil
}
