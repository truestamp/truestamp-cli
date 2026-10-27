// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"github.com/truestamp/truestamp-cli/internal/items"
	"github.com/truestamp/truestamp-cli/internal/proof"
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
	Long: `Show one item by its UUIDv7 id, read exactly as typed: nothing is
trimmed, and a UUID's 32-hex, braced and urn: forms are refused. An item
is shown by its id, not by its item code or its item_hash.

The card includes commitment state. For the authoritative answer, fetch
the proof and check it yourself:

  truestamp proofs get <id> | truestamp verify`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := requireAuth(cmd); err != nil {
			return err
		}
		id, err := itemIDArg("get", args[0])
		if err != nil {
			return err
		}
		it, err := items.Get(cmd.Context(), appConfig.APIURL, appConfig.Team, id)
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

The item is named by its UUIDv7 id, read exactly as typed: nothing is
trimmed, and a UUID's 32-hex, braced and urn: forms are refused. An item
is updated by its id, not by its item code or its item_hash.

Only three attributes are mutable, and the server enforces this:
visibility, tags, and the owning team. An item's claims, including its
name and description, which live inside claims, are immutable, because
claims_hash covers them and the item hash binds claims_hash into its
block's Merkle tree. There is no flag here that can reach a hashed
field.

--to-team moves the item to a different team you are a member of. It is
deliberately NOT the root --team flag: that one says which tenant scopes
the request, and letting one word mean both would make
'items update <id> --team ""', a perfectly ordinary way to scope a
request, silently mean "move this item to team ''". One word, one
meaning.

Examples:
  truestamp items update 01a0db66-c2e0-78a4-b8cc-12b11bc97493 --visibility public
  truestamp items update 01a0db66-c2e0-78a4-b8cc-12b11bc97493 --tags q3,contracts
  truestamp items update 01a0db66-c2e0-78a4-b8cc-12b11bc97493 --to-team 019dbd00-0000-7000-8000-000000000000`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := requireAuth(cmd); err != nil {
			return err
		}
		id, err := itemIDArg("update", args[0])
		if err != nil {
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
		it, err := items.Update(cmd.Context(), appConfig.APIURL, appConfig.Team, id, opts)
		if err != nil {
			return renderAPIError(cmd, err, "item")
		}
		return renderItem(cmd, it)
	},
}

// itemIDArg reads the item argument of `items <verb>` exactly as typed, as
// proof.DetectIDType classifies it, and returns the UUIDv7 in its
// canonical lowercase form. An item code or an item_hash is refused: these
// commands take an item by its id, and the app's update route reads only a
// UUIDv7, so a code or a hash would first have to be resolved to the id the
// item's own record gives.
func itemIDArg(verb, typed string) (string, error) {
	shape, id, err := proof.DetectIDType(typed)
	if err != nil {
		return "", nameTheFormTaken(err, fmt.Sprintf("items %s takes an item's UUIDv7 id (e.g. 01a0db66-c2e0-78a4-b8cc-12b11bc97493)", verb))
	}
	switch shape {
	case proof.IDTypeItemCode:
		return "", fmt.Errorf("items %s takes an item's UUIDv7 id (e.g. 01a0db66-c2e0-78a4-b8cc-12b11bc97493), not its item code", verb)
	case proof.IDTypeItemHash:
		return "", fmt.Errorf("items %s takes an item's UUIDv7 id (e.g. 01a0db66-c2e0-78a4-b8cc-12b11bc97493), not its item_hash", verb)
	}
	return id, nil
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
	if it.State == "created" || it.State == "processing" {
		ui.Fprintln(w, ui.FaintStyle().Render(
			"    Not yet committed: a proof cannot be generated until it is."))
	}
	return nil
}

// commitmentLabel says plainly what a committed item's state means for
// its proof, because that is what most callers want from the state
// field: the item is in a block, and its proof follows once that block
// is committed to a public chain.
func commitmentLabel(it *items.Item) string {
	if it.Committed() {
		return it.State + "  (in a block; its proof follows once the block is committed to a public chain)"
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
		ui.Fprintln(w, ui.HintStyle().Render(hintIndent+"No items."))
		return nil
	}
	header := ui.AccentBoldStyle().Render(listHeading("Items", len(list), pg))
	tbl := ui.CompactTable().StyleFunc(ui.LabelValueStyleFunc())
	for _, it := range list {
		tbl = tbl.Row(it.ID, itemListLine(it))
	}
	ui.Fprintln(w, strings.Join([]string{header, "", tbl.String()}, "\n"))
	renderListHints(cmd.ErrOrStderr(), pg,
		"Hint: 'truestamp items get <id>' shows one item; 'truestamp proofs get <id>' fetches its proof bundle.")
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
	lf.Bool("committed", false, "Only items committed in a block (a proof follows once the block is committed to a public chain)")
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
