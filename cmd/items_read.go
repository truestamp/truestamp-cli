// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"github.com/truestamp/truestamp-cli/internal/items"
	"github.com/truestamp/truestamp-cli/internal/ui"
)

var itemsListCmd = &cobra.Command{
	Use:   "list",
	Short: "List items in the current team",
	Long: `List items, newest first.

Paging is by keyset cursor: --limit sets the page size, --after and
--before continue from a cursor a previous page printed (forward, or back
toward the start), --oldest-first starts at the beginning instead of the
newest item, and --max follows cursors until that many items have been
fetched. There is no unbounded walk: a cap you wrote down is the price of
following pages. --count adds the total.

--committed and --pending filter on commitment state, which is the
question most often asked of this list: a proof can only be generated for
a committed item.

Commitment state is a column here rather than a separate 'status'
command. A server-reported state is a claim; a verified proof is
evidence, and the authoritative answer is:

  truestamp proofs get <id> | truestamp verify`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, _ []string) error {
		paging, err := readPagingOptions(cmd)
		if err != nil {
			return err
		}
		committed, _ := cmd.Flags().GetBool("committed")
		pending, _ := cmd.Flags().GetBool("pending")
		if err := requireAuth(cmd); err != nil {
			return err
		}
		rows, pg, err := walkPages(paging, func(after, before string, limit int) (*pageOf[items.Item], error) {
			p, err := items.List(cmd.Context(), appConfig.APIURL, appConfig.Team, items.ListOptions{
				Limit: limit, After: after, Before: before, OldestFirst: paging.OldestFirst, Count: paging.Count,
				Committed: committed, Pending: pending,
			})
			if err != nil {
				return nil, err
			}
			return &pageOf[items.Item]{Rows: p.Items, NextCursor: p.NextCursor, PrevCursor: p.PrevCursor, Total: p.Total, Limit: p.Limit}, nil
		})
		if err != nil {
			return renderAPIError(cmd, err, "item")
		}
		return renderItemList(cmd, rows, pg)
	},
}

var itemsGetCmd = &cobra.Command{
	Use:   "get <id>",
	Short: "Show one item by id",
	Long: `Show one item by its ULID.

The card includes commitment state. For the authoritative answer, fetch
the proof and check it yourself:

  truestamp proofs get <id> | truestamp verify`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := requireAuth(cmd); err != nil {
			return err
		}
		it, err := items.Get(cmd.Context(), appConfig.APIURL, appConfig.Team, strings.TrimSpace(args[0]))
		if err != nil {
			return renderAPIError(cmd, err, "item")
		}
		return renderItem(cmd, it)
	},
}

var itemsUpdateCmd = &cobra.Command{
	Use:   "update <id>",
	Short: "Update an item's mutable attributes",
	Long: `Update the attributes of an item that are not covered by its hash.

Only three attributes are mutable, and the server enforces this:
visibility, tags, and the owning team. An item's claims — including its
name and description, which live inside claims — are immutable, because
claims_hash is signed. There is no flag here that can reach a signed
field.

--to-team moves the item to a different team you are a member of. It is
deliberately NOT the root --team flag: that one says which tenant scopes
the request, and letting one word mean both would make
'items update <id> --team ""' — a perfectly ordinary way to scope a
request — silently mean "move this item to team ''". One word, one
meaning.

Examples:
  truestamp items update 01KNN33GX5E470CB9TRWAYF9DD --visibility public
  truestamp items update 01KNN33GX5E470CB9TRWAYF9DD --tags q3,contracts
  truestamp items update 01KNN33GX5E470CB9TRWAYF9DD --to-team 019dbd00-0000-7000-8000-000000000000`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := requireAuth(cmd); err != nil {
			return err
		}
		var opts items.UpdateOptions
		if cmd.Flags().Changed("visibility") {
			v, _ := cmd.Flags().GetString("visibility")
			if err := items.ValidateVisibility(v); err != nil {
				return err
			}
			opts.Visibility = &v
		}
		if cmd.Flags().Changed("tags") {
			raw, _ := cmd.Flags().GetStringSlice("tags")
			t := items.NormalizeTags(raw)
			opts.Tags = &t
		}
		if cmd.Flags().Changed("to-team") {
			v, _ := cmd.Flags().GetString("to-team")
			opts.TeamID = &v
		}
		it, err := items.Update(cmd.Context(), appConfig.APIURL, appConfig.Team,
			strings.TrimSpace(args[0]), opts)
		if err != nil {
			return renderAPIError(cmd, err, "item")
		}
		return renderItem(cmd, it)
	},
}

func renderItem(cmd *cobra.Command, it *items.Item) error {
	jsonOut, silent := outputMode(cmd)
	if silent {
		return nil
	}
	if jsonOut {
		return emitJSON(cmd.OutOrStdout(), it)
	}
	w := cmd.OutOrStdout()
	header := ui.AccentBoldStyle().Render("  Item")
	tbl := ui.CompactTable().
		StyleFunc(ui.LabelValueStyleFunc()).
		Row("ID", it.ID).
		Row("State", commitmentLabel(it)).
		Row("Visibility", it.Visibility)
	if it.DisplayName != "" {
		tbl = tbl.Row("Name", it.DisplayName)
	}
	if len(it.Tags) > 0 {
		tbl = tbl.Row("Tags", strings.Join(it.Tags, ", "))
	}
	if it.ClaimsHash != "" {
		tbl = tbl.Row("Claims Hash", it.ClaimsHash)
	}
	if it.ItemHash != "" {
		tbl = tbl.Row("Item Hash", it.ItemHash)
	}
	if it.TeamID != "" {
		tbl = tbl.Row("Team", it.TeamID)
	}
	if it.InsertedAt != "" {
		tbl = tbl.Row("Created", timestampWithRelative(it.InsertedAt))
	}
	if it.ExpiresAt != "" {
		tbl = tbl.Row("Expires", timestampWithRelative(it.ExpiresAt))
	}
	if detail := ui.SubjectDetailURL(appConfig.APIURL, "item", it.ID); detail != "" {
		tbl = tbl.Row("Details", detail)
	}
	ui.Fprintln(w, strings.Join([]string{header, "", tbl.String()}, "\n"))
	if !it.Committed() {
		ui.Fprintln(w, ui.FaintStyle().Render(
			"    Not yet committed: a proof cannot be generated until it is."))
	}
	return nil
}

// commitmentLabel states plainly whether a proof is available, because
// that is the only thing most callers want from the state field.
func commitmentLabel(it *items.Item) string {
	if it.Committed() {
		return it.State + "  (a proof can be generated)"
	}
	return it.State
}

func renderItemList(cmd *cobra.Command, list []items.Item, pg listPage) error {
	jsonOut, silent := outputMode(cmd)
	if silent {
		return nil
	}
	if jsonOut {
		return emitJSON(cmd.OutOrStdout(), listEnvelope("items", list, pg))
	}
	w := cmd.OutOrStdout()
	if len(list) == 0 {
		ui.Fprintln(w, ui.FaintStyle().Render("  No items."))
		return nil
	}
	header := ui.AccentBoldStyle().Render(listHeading("Items", len(list), pg))
	tbl := ui.CompactTable().StyleFunc(ui.LabelValueStyleFunc())
	for _, it := range list {
		tbl = tbl.Row(it.ID, itemListLine(it))
	}
	ui.Fprintln(w, strings.Join([]string{header, "", tbl.String()}, "\n"))
	renderMoreHint(w, pg)
	return nil
}

func itemListLine(it items.Item) string {
	mark := " "
	if it.Committed() {
		mark = "✓"
	}
	name := it.DisplayName
	if name == "" {
		name = "(no name)"
	}
	return fmt.Sprintf("%s %-10s %s", mark, it.State, name)
}

func init() {
	lf := itemsListCmd.Flags()
	addPagingFlags(itemsListCmd, "items")
	lf.Bool("committed", false, "Only items that have been committed (a proof can be generated)")
	lf.Bool("pending", false, "Only items not yet committed")

	uf := itemsUpdateCmd.Flags()
	uf.String("visibility", "", `Item visibility: "private", "team", or "public"`)
	uf.StringSlice("tags", nil, "Replace the item's tags (comma-separated or repeated)")
	uf.String("to-team", "", "Move the item to this team (not the same as the root --team, which scopes the request)")

	for _, c := range []*cobra.Command{itemsListCmd, itemsGetCmd, itemsUpdateCmd} {
		addRecordOutputFlags(c)
		itemsCmd.AddCommand(c)
	}
}
