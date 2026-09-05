// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: MIT

package cmd

import (
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"strings"

	"charm.land/lipgloss/v2/table"
	"github.com/spf13/cobra"
	"github.com/truestamp/truestamp-cli/internal/entropy"
	"github.com/truestamp/truestamp-cli/internal/ids"
	"github.com/truestamp/truestamp-cli/internal/inputsrc"
	"github.com/truestamp/truestamp-cli/internal/ui"
)

// entropyCmd is the parent for the `truestamp entropy ...` subtree. Like
// every group it is a namespace (asGroup): a bare `truestamp entropy`
// prints help.
var entropyCmd = &cobra.Command{
	Use:   "entropy",
	Short: "Read-only: public entropy observations (NIST, Stellar, Bitcoin)",
	Long: `Read the entropy observations Truestamp witnesses.

An entropy observation is a public random value captured from an
independent source, together with the moment it was captured: a NIST
Randomness Beacon pulse, a Stellar ledger close, a Bitcoin block. Every
item commits to the newest observation per source at submission, which
is what opens the submitted-after edge of its submission window: the item
cannot have been submitted before a value that did not yet exist.

Each observation is also a proof subject in its own right, and this group
is how you find one to ask for:

  truestamp proofs get $(truestamp entropy latest --json | jq -r .id) | truestamp verify

The same observations are published on the web at <base-url>/entropy. The
JSON:API behind this group needs a credential: run 'truestamp auth login',
or set TRUESTAMP_API_KEY / --api-key for headless/CI use.`,
}

var entropyListCmd = &cobra.Command{
	Use:   "list",
	Short: "Show the most recent observations (newest first)",
	Long: `List entropy observations, newest first, across all three sources or,
with --source, from one of them.

Examples:
  truestamp entropy list
  truestamp entropy list --source entropy_nist --limit 5
  truestamp entropy list --json | jq -r '.[].id'`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, _ []string) error {
		source, err := entropySourceFlag(cmd)
		if err != nil {
			return err
		}
		limit, err := pageLimit(cmd)
		if err != nil {
			return err
		}
		cfg, err := entropyConfig(cmd)
		if err != nil {
			return err
		}
		list, err := entropy.List(cmd.Context(), cfg, source, limit)
		if err != nil {
			return renderAPIError(cmd, err, "entropy observation")
		}
		return renderObservationList(cmd, list)
	},
}

var entropyGetCmd = &cobra.Command{
	Use:   "get <id-or-hash>",
	Short: "Show one observation by UUIDv7 id or by 64-hex-char entropy hash",
	Long: `Fetch one observation, addressed either way.

The two shapes are disjoint — a UUIDv7 has hyphens, an entropy hash is
exactly 64 lowercase hex characters — so which you passed is unambiguous.
The hash is the value an item's metadata commits to under
subject.metadata.witnesses, so this is how a witness named in a proof is
traced back to the observation it came from:

  truestamp entropy get $(jq -r .subject.metadata.witnesses.entropy_stellar proof.json)

There is no by-hash route on the server, so a hash lookup is a filter; the
hex is validated here before the request is sent, and more than one match
is reported as an error rather than resolved by picking one.

Examples:
  truestamp entropy get 01a07335-b8fe-7ef2-856c-c9b4eca99850
  truestamp entropy get 4142459a2859a57a5e5d6540579b977215d6329110adc0a890fdce6f05054f2d --json`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		arg := strings.TrimSpace(args[0])
		if !ids.LooksLikeHash64(arg) && !strings.Contains(arg, "-") {
			return fmt.Errorf("%q is neither a UUIDv7 id nor a 64-hex-char entropy hash", arg)
		}
		cfg, err := entropyConfig(cmd)
		if err != nil {
			return err
		}
		// The client validates the id or hash before any request is sent.
		var o *entropy.Observation
		if ids.LooksLikeHash64(arg) {
			o, err = entropy.ByHash(cmd.Context(), cfg, arg)
		} else {
			o, err = entropy.Get(cmd.Context(), cfg, arg)
		}
		if err != nil {
			return renderAPIError(cmd, err, "entropy observation")
		}
		return renderObservation(cmd, o)
	},
}

var entropyLatestCmd = &cobra.Command{
	Use:   "latest",
	Short: "Show the newest observation, from any source or one --source",
	Long: `Show the newest observation. Without --source that is the most recently
captured observation from any source; with --source it is the newest from
that source, the value a newly submitted item's metadata commits to for
it.

Examples:
  truestamp entropy latest
  truestamp entropy latest --source entropy_bitcoin --json`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, _ []string) error {
		source, err := entropySourceFlag(cmd)
		if err != nil {
			return err
		}
		cfg, err := entropyConfig(cmd)
		if err != nil {
			return err
		}
		o, err := entropy.Latest(cmd.Context(), cfg, source)
		if err != nil {
			return renderAPIError(cmd, err, "entropy observation")
		}
		return renderObservation(cmd, o)
	},
}

// entropySourceFlag reads --source and refuses a value outside the closed
// set before any request is made. Empty means every source.
func entropySourceFlag(cmd *cobra.Command) (string, error) {
	source, _ := cmd.Flags().GetString("source")
	source = strings.TrimSpace(source)
	if source == "" {
		return "", nil
	}
	return source, entropy.ValidateSource(source)
}

// entropyConfig pulls the values the entropy client needs from the
// resolved application config, after the shared credential gate.
func entropyConfig(cmd *cobra.Command) (entropy.Config, error) {
	if err := requireAuth(cmd); err != nil {
		return entropy.Config{}, err
	}
	return entropy.Config{APIURL: appConfig.APIURL, Team: appConfig.Team}, nil
}

func renderObservation(cmd *cobra.Command, o *entropy.Observation) error {
	jsonOut, silent := outputMode(cmd)
	if silent {
		return nil
	}
	if jsonOut {
		return emitJSON(cmd.OutOrStdout(), o)
	}
	renderObservationCard(cmd.OutOrStdout(), appConfig.APIURL, o)
	return nil
}

// renderObservationCard prints the observation, then the source's own
// record under its own field names (a Stellar ledger's sequence, a NIST
// pulse's index): what the source published is rendered as published,
// never renamed. Two public-web links follow, built from base_url.
func renderObservationCard(w io.Writer, apiURL string, o *entropy.Observation) {
	header := ui.AccentBoldStyle().Render("  Entropy Observation")
	tbl := ui.CompactTable().
		StyleFunc(ui.LabelValueStyleFunc()).
		Row("ID", o.ID).
		Row("Source", o.Source).
		Row("State", o.State).
		Row("Entropy Hash", o.EntropyHash)
	if o.SourcePublishedAt != "" {
		tbl = tbl.Row("Published", timestampWithRelative(o.SourcePublishedAt))
	}
	if o.InsertedAt != "" {
		tbl = tbl.Row("Captured", timestampWithRelative(o.InsertedAt))
	}
	if o.BlockID != "" {
		tbl = tbl.Row("Block", o.BlockID)
	}
	tbl = tbl.Row("Signing Key", o.SigningKeyID)
	tbl = entropyRows(tbl, "", o.Entropy)
	if detail := ui.SubjectDetailURL(apiURL, o.Source, o.ID); detail != "" {
		tbl = tbl.Row("Details", detail)
	}
	if verify := ui.SubjectVerifyURL(apiURL, o.Source, o.ID); verify != "" {
		tbl = tbl.Row("Verify", verify)
	}
	ui.Fprintln(w, strings.Join([]string{header, "", tbl.String()}, "\n"))
}

// entropyRows adds one row per field of the source's record, under the
// source's own names. A nested object (a NIST pulse is one) becomes
// dotted rows, pulse.pulseIndex, rather than one line of JSON the width
// of the terminal; anything else structured is compact JSON.
func entropyRows(tbl *table.Table, prefix string, fields map[string]any) *table.Table {
	keys := make([]string, 0, len(fields))
	for k := range fields {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		label := k
		if prefix != "" {
			label = prefix + "." + k
		}
		if nested, ok := fields[k].(map[string]any); ok {
			tbl = entropyRows(tbl, label, nested)
			continue
		}
		tbl = tbl.Row(label, entropyValue(fields[k]))
	}
	return tbl
}

// entropyValue renders one scalar field as published: strings and numbers
// verbatim, anything else as compact JSON.
func entropyValue(v any) string {
	switch t := v.(type) {
	case string:
		return t
	case json.Number:
		return t.String()
	default:
		b, err := json.Marshal(v)
		if err != nil {
			return fmt.Sprint(v)
		}
		return string(b)
	}
}

func renderObservationList(cmd *cobra.Command, list []entropy.Observation) error {
	jsonOut, silent := outputMode(cmd)
	if silent {
		return nil
	}
	if jsonOut {
		return emitJSON(cmd.OutOrStdout(), list)
	}
	w := cmd.OutOrStdout()
	if len(list) == 0 {
		ui.Fprintln(w, ui.FaintStyle().Render("  No entropy observations."))
		return nil
	}
	header := ui.AccentBoldStyle().Render(fmt.Sprintf("  Entropy Observations (%d)", len(list)))
	// The hash is shortened for the column; the full value is on the card
	// and in --json, and `entropy get` accepts it whole.
	rows := [][]string{{"PUBLISHED", "SOURCE", "STATE", "ID", "HASH"}}
	for _, o := range list {
		rows = append(rows, []string{ui.TruncateToSecond(o.SourcePublishedAt), o.Source, o.State, o.ID, truncateHash(o.EntropyHash)})
	}
	tbl := ui.CompactTable().StyleFunc(ui.HeaderRowStyleFunc()).Rows(rows...)
	ui.Fprintln(w, strings.Join([]string{header, "", tbl.String()}, "\n"))
	if inputsrc.IsStdoutTerminal() {
		ui.Fprintln(cmd.ErrOrStderr(), ui.FaintStyle().Render(
			"  Hint: 'truestamp proofs get <id>' fetches a verifiable proof bundle for an observation."))
	}
	return nil
}

func init() {
	for _, c := range []*cobra.Command{entropyListCmd, entropyLatestCmd} {
		c.Flags().String("source", "", "Only this source: "+strings.Join(entropy.Sources, " | "))
	}
	addLimitFlag(entropyListCmd, "observations")
	for _, c := range []*cobra.Command{entropyListCmd, entropyGetCmd, entropyLatestCmd} {
		addRecordOutputFlags(c)
		entropyCmd.AddCommand(c)
	}
	entropyCmd.GroupID = groupResources
	rootCmd.AddCommand(asGroup(entropyCmd))
}
