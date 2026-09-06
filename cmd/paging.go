// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"fmt"
	"io"
	"strconv"

	"github.com/spf13/cobra"
	"github.com/truestamp/truestamp-cli/internal/ui"
)

// Every keyset-paged list (items, blocks, entropy) pages the same way
// (kb/command-tree.md R14): --limit is the page size, --after and --before
// continue from a cursor a previous page printed, forward or backward,
// --oldest-first starts the walk at the beginning instead of the newest
// row, --max follows cursors until that many rows have been fetched, and
// --count asks for the total. There is deliberately no --all: the tables behind these lists grow by the
// minute (the entropy tables alone hold well over half a million rows),
// and an unbounded walk is a footgun, so the walk always has a cap the
// caller wrote down.
//
// beacons is the exception: its endpoint takes ?limit= (at most 100) and
// returns no cursor, so it carries --limit alone.

// pagingOptions are the four flags, already validated.
type pagingOptions struct {
	Limit       int
	After       string
	Before      string
	OldestFirst bool
	Max         int // 0 means one page
	Count       bool
}

// listPage is what a renderer needs to know beyond the rows: the cursors
// that continue the listing in either direction and, when --count asked,
// the total.
type listPage struct {
	Next    string
	Prev    string
	Total   int
	Counted bool
	// ClampedTo is the page size the server used when it was smaller than
	// the one asked for (every collection clamps to its max_page_size, 250
	// by default, rather than refusing); zero otherwise.
	ClampedTo int
}

// pageOf is one fetched page of any row type, the shape walkPages
// consumes.
type pageOf[T any] struct {
	Rows       []T
	NextCursor string
	PrevCursor string
	Total      int
	Limit      int // the page size the server reported using
}

func addPagingFlags(cmd *cobra.Command, noun string) {
	addLimitFlag(cmd, noun)
	f := cmd.Flags()
	f.String("after", "", "Continue forward from a previous page's cursor (the More: hint, or next_cursor in --json)")
	f.String("before", "", "Continue backward from a previous page's cursor (the Back: hint, or prev_cursor in --json)")
	f.Bool("oldest-first", false, "Start from the oldest "+noun+" and walk toward the newest (default: newest first)")
	f.Int("max", 0, "Follow cursors until this many "+noun+" have been fetched, then stop; without it one page is fetched")
	f.Bool("count", false, "Also report how many "+noun+" match in total")
}

func readPagingOptions(cmd *cobra.Command) (pagingOptions, error) {
	limit, err := pageLimit(cmd)
	if err != nil {
		return pagingOptions{}, err
	}
	after, _ := cmd.Flags().GetString("after")
	before, _ := cmd.Flags().GetString("before")
	if after != "" && before != "" {
		return pagingOptions{}, fmt.Errorf("--after and --before are mutually exclusive: a walk continues in one direction")
	}
	oldestFirst, _ := cmd.Flags().GetBool("oldest-first")
	max, _ := cmd.Flags().GetInt("max")
	if cmd.Flags().Changed("max") && max < 1 {
		return pagingOptions{}, fmt.Errorf("--max must be at least 1, got %d", max)
	}
	count, _ := cmd.Flags().GetBool("count")
	return pagingOptions{Limit: limit, After: after, Before: before, OldestFirst: oldestFirst, Max: max, Count: count}, nil
}

// walkPages fetches one page, or with --max as many as it takes to reach
// the cap or the end, forward from --after (or the start) or backward from
// --before. The last request asks for exactly the rows still wanted, so a
// page is never cut and the cursors handed back always continue from the
// rows shown. Rows come back in the listing's order whichever way the walk
// went: pages fetched backward are prepended.
func walkPages[T any](opts pagingOptions, fetch func(after, before string, limit int) (*pageOf[T], error)) (rows []T, pg listPage, err error) {
	backward := opts.Before != ""
	after, before := opts.After, opts.Before
	pg.Counted = opts.Count
	for first := true; ; first = false {
		limit := opts.Limit
		if opts.Max > 0 && opts.Max-len(rows) < limit {
			limit = opts.Max - len(rows)
		}
		got, err := fetch(after, before, limit)
		if err != nil {
			return nil, listPage{}, err
		}
		pg.Total = got.Total
		if got.Limit > 0 && got.Limit < limit {
			pg.ClampedTo = got.Limit
		}
		if backward {
			rows = append(append([]T(nil), got.Rows...), rows...)
			if first {
				pg.Next = got.NextCursor
			}
			pg.Prev = got.PrevCursor
		} else {
			rows = append(rows, got.Rows...)
			if first {
				pg.Prev = got.PrevCursor
			}
			pg.Next = got.NextCursor
		}
		onward := got.NextCursor
		if backward {
			onward = got.PrevCursor
		}
		if opts.Max <= 0 || len(rows) >= opts.Max || onward == "" {
			if opts.Max > 0 && len(rows) > opts.Max {
				if backward {
					rows = rows[len(rows)-opts.Max:]
				} else {
					rows = rows[:opts.Max]
				}
			}
			return rows, pg, nil
		}
		if backward {
			before = onward
		} else {
			after = onward
		}
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

// renderMoreHint prints the continuation lines: More continues in the
// listing's order, Back returns toward its start. Each appears only when
// there is somewhere to go.
func renderMoreHint(w io.Writer, pg listPage) {
	if pg.Next != "" {
		ui.Fprintln(w, ui.FaintStyle().Render("    More: --after "+pg.Next))
	}
	if pg.Prev != "" {
		ui.Fprintln(w, ui.FaintStyle().Render("    Back: --before "+pg.Prev))
	}
	// The clamp is only worth a line when there is something past it: the
	// server rewrites page[limit] to its cap even on a short collection,
	// so a last page of 10 rows can carry meta.page.limit 250.
	if pg.ClampedTo > 0 && pg.Next != "" {
		ui.Fprintln(w, ui.FaintStyle().Render(fmt.Sprintf("    (the server caps a page at %d rows; --max follows the cursor past it)", pg.ClampedTo)))
	}
}

// listEnvelope is the --json shape of every paged list: the rows under
// the noun, next_cursor and prev_cursor (empty at the end and the start
// of the listing respectively), under --count total, and page_limit
// whenever the server used a smaller page than was asked for. Unlike the
// human clamp note, page_limit is not gated on a following page: JSON
// reports the fact, next_cursor says whether there is more.
func listEnvelope(noun string, rows any, pg listPage) map[string]any {
	out := map[string]any{noun: rows, "next_cursor": pg.Next, "prev_cursor": pg.Prev}
	if pg.Counted {
		out["total"] = pg.Total
	}
	if pg.ClampedTo > 0 {
		out["page_limit"] = pg.ClampedTo
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
