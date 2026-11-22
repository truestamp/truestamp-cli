// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"fmt"
	"io"
	"regexp"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
	"github.com/truestamp/truestamp-cli/internal/external"
	"github.com/truestamp/truestamp-cli/internal/ui"
)

// The keyring document carries exactly four fields per entry: key_id,
// public_key, sequence, active. There is NO revocation flag, NO validity
// interval and NO timestamp anywhere in it.
//
// That constrains this group hard: a compromised key appears as an
// ordinary inactive entry, indistinguishable from a routinely retired
// one, so `keys list` must never render a "status", "valid until",
// "expires" or "revoked" column. The honest columns are exactly the four
// that exist. Do not add a fifth that implies knowledge the document does
// not carry.

var keysCmd = &cobra.Command{
	Use:   "keys",
	Short: "Read-only: Truestamp's published signing keys",
	Long: `Read the published Truestamp signing keyring.

It needs no credential: the keyring is served unauthenticated from
/.well-known/keyring.json, and it is the same document 'truestamp verify'
consults for the key binding step.

What the keyring can and cannot tell you: it answers "is this key
Truestamp's", which a proof cannot answer about itself, because a forged
proof is self-consistent by construction and carries its forger's own
public key. It does not tell you a key was uncompromised, was valid at
signing time, or was authorized: the document carries no revocation
flag, no validity interval and no timestamps.

The stronger artifact is the signing key event a proof carries, which
ties a key to a block whose hash sits under a root committed to a public
blockchain, and stays checkable long after Truestamp is gone.`,
}

var keysListCmd = &cobra.Command{
	Use:   "list",
	Short: "Show every published signing key",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, _ []string) error {
		kr, err := external.FetchKeyring(appConfig.KeyringURL)
		if err != nil {
			return err
		}
		jsonOut, silent := outputMode(cmd)
		if silent {
			return nil
		}
		if jsonOut {
			return emitJSON(cmd.OutOrStdout(), kr)
		}
		renderKeyList(cmd.OutOrStdout(), kr)
		return nil
	},
}

var keysGetCmd = &cobra.Command{
	Use:   "get <kid>",
	Short: "Show one signing key by its 8-hex key id",
	Long: `Fetch one published key by its key id.

The key id is the 4-byte fingerprint rendered as 8 lowercase hex
characters. To derive it from a public key you already hold, use
'truestamp convert keyid'.`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		// The key id exactly as typed: nothing is trimmed, and a value that
		// is no key id is refused before the keyring is read.
		want := args[0]
		if !keyIDText.MatchString(want) {
			return fmt.Errorf("%q is no key id: keys get takes an 8-hex key id (e.g. 96b1cd2f)", want)
		}
		kr, err := external.FetchKeyring(appConfig.KeyringURL)
		if err != nil {
			return err
		}
		k, ok := kr.FindByID(want)
		if !ok {
			return &cliError{code: codeNotFound, message: fmt.Sprintf("no published key with id %q", want)}
		}
		return renderOneKey(cmd, k)
	},
}

// keyIDText is the text of a key id, 8 hex characters, in either case as
// the keyring lookup compares it.
var keyIDText = regexp.MustCompile(`\A[0-9a-fA-F]{8}\z`)

var keysCurrentCmd = &cobra.Command{
	Use:   "current",
	Short: "Show the key signing right now",
	Long: `Show the active signing key.

'current', not 'latest': the concept wanted is "the key signing right
now", not "the most recent row". Today they coincide, but the keyring is
populated by chain replay including prerotation events, so a
published-but-not-yet-active key would make them diverge.`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, _ []string) error {
		kr, err := external.FetchKeyring(appConfig.KeyringURL)
		if err != nil {
			return err
		}
		k, ok := kr.Active()
		if !ok {
			return fmt.Errorf("the published keyring contains no active key")
		}
		return renderOneKey(cmd, k)
	},
}

func renderOneKey(cmd *cobra.Command, k external.KeyringKey) error {
	jsonOut, silent := outputMode(cmd)
	if silent {
		return nil
	}
	if jsonOut {
		return emitJSON(cmd.OutOrStdout(), k)
	}
	header := ui.AccentBoldStyle().Render("  Signing Key")
	tbl := ui.CompactTable().
		StyleFunc(ui.LabelValueStyleFunc()).
		Row("Key ID", k.KeyID).
		Row("Public Key", k.PublicKey).
		Row("Sequence", fmt.Sprintf("%d", k.Sequence)).
		Row("Active", fmt.Sprintf("%v", k.Active))
	ui.Fprintln(cmd.OutOrStdout(), strings.Join([]string{header, "", tbl.String()}, "\n"))
	return nil
}

func renderKeyList(w io.Writer, kr *external.KeyringResponse) {
	if len(kr.Keys) == 0 {
		ui.Fprintln(w, ui.HintStyle().Render(hintIndent+"The published keyring contains no keys."))
		return
	}
	header := ui.AccentBoldStyle().Render(fmt.Sprintf("  Signing Keys (%d)", len(kr.Keys)))
	// Exactly the keyring's four facts, with a header row as every other
	// list has (kb/command-tree.md: SEQ · KEY ID · PUBLIC KEY · ACTIVE). An
	// inactive key reads "no", never a blank a reader could take for
	// unknown, and never a status the keyring does not carry.
	rows := [][]string{{"SEQ", "KEY ID", "PUBLIC KEY", "ACTIVE"}}
	for _, k := range kr.Keys {
		active := "no"
		if k.Active {
			active = "yes"
		}
		rows = append(rows, []string{strconv.Itoa(k.Sequence), k.KeyID, k.PublicKey, active})
	}
	tbl := ui.CompactTable().StyleFunc(ui.HeaderRowStyleFunc()).Rows(rows...)
	ui.Fprintln(w, strings.Join([]string{header, "", tbl.String()}, "\n"))
}

func init() {
	for _, c := range []*cobra.Command{keysListCmd, keysGetCmd, keysCurrentCmd} {
		addRecordOutputFlags(c)
		keysCmd.AddCommand(c)
	}
	keysCmd.GroupID = groupResources
	rootCmd.AddCommand(asGroup(keysCmd))
}
