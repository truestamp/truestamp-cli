// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"errors"
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"github.com/truestamp/truestamp-cli/internal/ids"
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
a committed item. --pending keeps the items still on their way into a
block, those created or processing; a redacted or deleted item is neither.

Each row gives the item's id and its item code, whole: the code, from the
API, names the item on its web pages and in 'items get', 'items update'
and 'proofs get'.

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
		// Refused before the credential check, as every local refusal is.
		if committed && pending {
			return fmt.Errorf("--committed and --pending are mutually exclusive")
		}
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
	Use:   "get <id-code-or-hash>",
	Short: "Show one item by id, item code or item_hash",
	Long: `Show one item, named by its UUIDv7 id, its item code (e.g.
9WBPKNV7SCP87VZW) or its item_hash (64 hex characters). Nothing is
trimmed: a UUIDv7 or an item_hash is read in either case, and an item
code as SPEC.md section 3.9 lets it be typed (any case, I and L read as 1
and O as 0, and one space, hyphen or underscore allowed between its
groups of four); a UUID's 32-hex, braced and urn: forms are refused. One
GET reads the item by whichever is given, and the item the server answers
with must carry the code or the whole item_hash given.

The card includes commitment state. For the authoritative answer, fetch
the proof and check it yourself:

  truestamp proofs get <id> | truestamp verify`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := requireAuth(cmd); err != nil {
			return err
		}
		shape, ref, err := itemRefArg("get", args[0])
		if err != nil {
			return err
		}
		it, err := getItemByRef(cmd, shape, ref)
		if err != nil {
			return err
		}
		return renderItem(cmd, it)
	},
}

var itemsUpdateCmd = &cobra.Command{
	Use:   "update <id-code-or-hash>",
	Short: "Update an item's mutable attributes",
	Long: `Update the attributes of an item that are not covered by its hash.

The item is named by its UUIDv7 id, its item code (e.g. 9WBPKNV7SCP87VZW)
or its item_hash (64 hex characters). Nothing is trimmed: a UUIDv7 or an
item_hash is read in either case, and an item code as SPEC.md section 3.9
lets it be typed (any case, I and L read as 1 and O as 0, and one space,
hyphen or underscore allowed between its groups of four); a UUID's
32-hex, braced and urn: forms are refused. The
server updates an item only by its UUIDv7 id, so a code or an item_hash is
first read with one GET, whose item must carry that code or that whole
item_hash, and the update is then sent by the id that item gives.

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
  truestamp items update 9WBPKNV7SCP87VZW --visibility team
  truestamp items update 01a0db66-c2e0-78a4-b8cc-12b11bc97493 --tags q3,contracts
  truestamp items update 01a0db66-c2e0-78a4-b8cc-12b11bc97493 --to-team 019dbd00-0000-7000-8000-000000000000`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := requireAuth(cmd); err != nil {
			return err
		}
		shape, ref, err := itemRefArg("update", args[0])
		if err != nil {
			return err
		}
		var opts items.UpdateOptions
		if cmd.Flags().Changed("visibility") {
			v, _ := cmd.Flags().GetString("visibility")
			if v == "" {
				return items.ErrEmptyVisibility
			}
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
			// Checked before any request, as teams get and teams use check a
			// team id.
			if !ids.IsUUIDv7Text(v) {
				return fmt.Errorf("%q is no team id: --to-team takes a team's UUIDv7 id (e.g. 019dbd00-0000-7000-8000-000000000000)", v)
			}
			opts.TeamID = &v
		}
		if opts.Empty() {
			return items.ErrNothingToUpdate
		}
		id := ref
		if shape != proof.IDTypeUUIDv7 {
			named, err := getItemByRef(cmd, shape, ref)
			if err != nil {
				return err
			}
			id = strings.ToLower(named.ID)
		}
		it, err := items.Update(cmd.Context(), appConfig.APIURL, appConfig.Team, id, opts)
		if err != nil {
			return renderAPIError(cmd, err, "item")
		}
		return renderItem(cmd, it)
	},
}

// itemRefArg reads the item argument of `items <verb>` exactly as typed,
// as proof.DetectIDType classifies it, and returns its shape and its
// canonical form: a UUIDv7 or an item_hash in lowercase, an item code in
// its one uppercase spelling. These are the three references the app's
// item read route takes (GET /items/:id, its :read_by_ref action).
func itemRefArg(verb, typed string) (proof.IDType, string, error) {
	shape, ref, err := proof.DetectIDType(typed)
	if err != nil {
		return "", "", nameTheFormTaken(err, fmt.Sprintf("items %s takes an item's UUIDv7 id (e.g. 01a0db66-c2e0-78a4-b8cc-12b11bc97493), "+
			"its item code (e.g. 9WBPKNV7SCP87VZW) or its item_hash", verb))
	}
	return shape, ref, nil
}

// errDifferentItem is an item the server answered a code or an item_hash
// with that does not carry it.
var errDifferentItem = errors.New("the server returned a different item")

// getItemByRef reads the item ref names, of shape shape, with one GET by
// ref as given, and returns it once it is the item named: for an item
// code, an item whose item_code is ref; for an item_hash, one whose
// item_hash is ref, all of it; and for either, one whose id is a UUIDv7,
// the only reference the app's update route reads. Any other answer is
// errDifferentItem. A refusal of the GET is rendered as every API
// refusal is.
func getItemByRef(cmd *cobra.Command, shape proof.IDType, ref string) (*items.Item, error) {
	it, err := items.Get(cmd.Context(), appConfig.APIURL, appConfig.Team, ref)
	if err != nil {
		return nil, renderAPIError(cmd, err, "item")
	}
	switch {
	case shape == proof.IDTypeItemCode && it.ItemCode != ref:
		return nil, fmt.Errorf("%w: asked for the item code %s, answered with an item whose item code is %q", errDifferentItem, ref, it.ItemCode)
	case shape == proof.IDTypeItemHash && it.ItemHash != ref:
		return nil, fmt.Errorf("%w: asked for the item_hash %s, answered with an item whose item_hash is %q", errDifferentItem, ref, it.ItemHash)
	case shape != proof.IDTypeUUIDv7 && !ids.IsUUIDv7Text(it.ID):
		return nil, fmt.Errorf("%w: asked for %s, answered with an item whose id %q is not a UUIDv7", errDifferentItem, ref, it.ID)
	}
	return it, nil
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
		tbl = tbl.Row("Claims hash", it.ClaimsHash)
	}
	if it.ItemHash != "" {
		tbl = tbl.Row("Item hash", it.ItemHash)
	}
	if it.ItemCode != "" {
		tbl = tbl.Row("Item code", it.ItemCode)
	}
	if it.TeamID != "" {
		tbl = tbl.Row("Team", it.TeamID)
	}
	if it.InsertedAt != "" {
		tbl = tbl.Row("Submitted", timestampWithRelative(it.InsertedAt))
	}
	if it.ExpiresAt != "" {
		tbl = tbl.Row("Expires", timestampWithRelative(it.ExpiresAt))
	}
	if detail := ui.SubjectDetailURL(appConfig.APIURL, "item", it.ItemCode); detail != "" {
		tbl = tbl.Row("Details", detail)
	}
	ui.Fprintln(w, strings.Join([]string{header, "", tbl.String()}, "\n"))
	if it.Pending() {
		ui.Fprintln(w, ui.FaintStyle().Render(
			"    Not yet committed: a proof cannot be generated until it is."))
	}
	return nil
}

// commitmentLabel says plainly what a committed item's state means for
// its proof, because that is what most callers want from the state
// field: the item is in a block, and the server serves its proof once
// that block is committed to a public chain. The block's state says which
// of the two it is, when the response included the block; without it
// (an update's response, or a server that sent none) the label says only
// what is always true.
func commitmentLabel(it *items.Item) string {
	switch {
	case !it.Committed():
		return it.State
	case it.BlockCommitted():
		ref := it.ItemCode
		if ref == "" {
			ref = it.ID
		}
		return it.State + "  (in a block committed to a public chain; 'truestamp proofs get " + ref + "' fetches its proof)"
	case it.BlockState == "finalized":
		return it.State + "  (in a block; its proof follows once the block is committed to a public chain)"
	}
	return it.State + "  (in a block; a proof is served once that block is committed to a public chain)"
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
	// The id and the item code are rendered whole, each in a column of
	// its own: the code is a value of its own, never a short form of the
	// id or of item_hash.
	rows := [][]string{{"ID", "CODE", "STATE", "NAME"}}
	for _, it := range list {
		rows = append(rows, itemListRow(it))
	}
	tbl := ui.CompactTable().StyleFunc(ui.HeaderRowStyleFunc()).Rows(rows...)
	ui.Fprintln(w, strings.Join([]string{header, "", tbl.String()}, "\n"))
	renderListHints(cmd.ErrOrStderr(), pg,
		"Hint: 'truestamp items get <id | code>' shows one item; 'truestamp proofs get <id | code>' fetches its proof.")
	return nil
}

// itemListRow is one item's row of the list: its id, its item code, its
// state, marked when committed, and its name.
func itemListRow(it items.Item) []string {
	mark := " "
	if it.Committed() {
		mark = "✓"
	}
	name := it.DisplayName
	if name == "" {
		name = "(no name)"
	}
	return []string{it.ID, it.ItemCode, mark + " " + it.State, name}
}

func init() {
	lf := itemsListCmd.Flags()
	addPagingFlags(itemsListCmd, "items")
	lf.Bool("committed", false, "Only items committed in a block (a proof follows once the block is committed to a public chain)")
	lf.Bool("pending", false, "Only items not yet in a block: those created or processing")

	uf := itemsUpdateCmd.Flags()
	uf.String("visibility", "", `Item visibility: "private", "team", or "public"`)
	uf.StringSlice("tags", nil, "Replace the item's tags (comma-separated or repeated)")
	uf.String("to-team", "", "Move the item to this team (not the same as the root --team, which scopes the request)")

	for _, c := range []*cobra.Command{itemsListCmd, itemsGetCmd, itemsUpdateCmd} {
		addRecordOutputFlags(c)
		itemsCmd.AddCommand(c)
	}
}
