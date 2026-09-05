// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: MIT

package cmd

import (
	"fmt"
	"io"
	"strconv"

	"github.com/spf13/cobra"
	"github.com/truestamp/truestamp-cli/internal/ui"
)

// Every keyset-paged list (items, blocks, entropy) pages the same way
// (kb/command-tree.md R14): --limit is the page size, --after continues
// from the cursor a previous page printed, --max follows cursors until
// that many rows have been fetched, and --count asks for the total. There
// is deliberately no --all: the tables behind these lists grow by the
// minute (staging holds well over half a million entropy observations),
// and an unbounded walk is a footgun, so the walk always has a cap the
// caller wrote down.
//
// beacons is the exception: its endpoint takes ?limit= (at most 100) and
// returns no cursor, so it carries --limit alone.

// pagingOptions are the four flags, already validated.
type pagingOptions struct {
	Limit int
	After string
	Max   int // 0 means one page
	Count bool
}

// listPage is what a renderer needs to know beyond the rows: the cursor
// that continues the listing and, when --count asked, the total.
type listPage struct {
	Next    string
	Total   int
	Counted bool
}

// pageOf is one fetched page of any row type, the shape walkPages
// consumes.
type pageOf[T any] struct {
	Rows       []T
	NextCursor string
	Total      int
}

func addPagingFlags(cmd *cobra.Command, noun string) {
	addLimitFlag(cmd, noun)
	f := cmd.Flags()
	f.String("after", "", "Continue from a previous page's cursor (the More: hint, or next_cursor in --json)")
	f.Int("max", 0, "Follow cursors until this many "+noun+" have been fetched, then stop; without it one page is fetched")
	f.Bool("count", false, "Also report how many "+noun+" match in total")
}

func readPagingOptions(cmd *cobra.Command) (pagingOptions, error) {
	limit, err := pageLimit(cmd)
	if err != nil {
		return pagingOptions{}, err
	}
	after, _ := cmd.Flags().GetString("after")
	max, _ := cmd.Flags().GetInt("max")
	if cmd.Flags().Changed("max") && max < 1 {
		return pagingOptions{}, fmt.Errorf("--max must be at least 1, got %d", max)
	}
	count, _ := cmd.Flags().GetBool("count")
	return pagingOptions{Limit: limit, After: after, Max: max, Count: count}, nil
}

// walkPages fetches one page, or with --max as many as it takes to reach
// the cap or the end. The last request asks for exactly the rows still
// wanted, so a page is never cut and the cursor handed back always
// continues from the last row shown.
func walkPages[T any](opts pagingOptions, fetch func(after string, limit int) (*pageOf[T], error)) (rows []T, next string, total int, err error) {
	after := opts.After
	for {
		limit := opts.Limit
		if opts.Max > 0 && opts.Max-len(rows) < limit {
			limit = opts.Max - len(rows)
		}
		pg, err := fetch(after, limit)
		if err != nil {
			return nil, "", 0, err
		}
		rows = append(rows, pg.Rows...)
		total = pg.Total
		if opts.Max <= 0 || len(rows) >= opts.Max || pg.NextCursor == "" {
			if opts.Max > 0 && len(rows) > opts.Max {
				rows = rows[:opts.Max]
			}
			return rows, pg.NextCursor, total, nil
		}
		after = pg.NextCursor
	}
}

// listHeading renders "  Items (25)" or, under --count,
// "  Items (25 shown, 1,204 total)".
func listHeading(noun string, shown int, pg listPage) string {
	if pg.Counted {
		return fmt.Sprintf("  %s (%d shown, %s total)", noun, shown, humanInt(pg.Total))
	}
	return fmt.Sprintf("  %s (%d)", noun, shown)
}

// renderMoreHint prints the continuation line when a cursor remains.
func renderMoreHint(w io.Writer, pg listPage) {
	if pg.Next != "" {
		ui.Fprintln(w, ui.FaintStyle().Render("    More: --after "+pg.Next))
	}
}

// listEnvelope is the --json shape of every paged list: the rows under
// the noun, next_cursor (empty once the last page has been read) and,
// under --count, total.
func listEnvelope(noun string, rows any, pg listPage) map[string]any {
	out := map[string]any{noun: rows, "next_cursor": pg.Next}
	if pg.Counted {
		out["total"] = pg.Total
	}
	return out
}

// humanInt renders 1204 as "1,204".
func humanInt(n int) string {
	s := strconv.Itoa(n)
	if n < 0 {
		return "-" + humanInt(-n)
	}
	for i := len(s) - 3; i > 0; i -= 3 {
		s = s[:i] + "," + s[i:]
	}
	return s
}
