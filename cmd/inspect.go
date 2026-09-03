// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: MIT

package cmd

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	lipgloss "charm.land/lipgloss/v2"
	"github.com/spf13/cobra"
	"github.com/truestamp/truestamp-cli/internal/inputsrc"
	"github.com/truestamp/truestamp-cli/internal/proof"
	"github.com/truestamp/truestamp-cli/internal/tscrypto"
	"github.com/truestamp/truestamp-cli/internal/ui"
	"github.com/truestamp/truestamp-cli/internal/verify"
)

var inspectCmd = &cobra.Command{
	Use:   "inspect [file-or-url]",
	Short: "Print a proof bundle's key fields without verifying it",
	Long: `Parse a Truestamp proof bundle (JSON or CBOR) and print what it carries:
the format version and subject type, the ids, the witnesses the subject
metadata commits to and the witness details carried, the block, the
commitments, and whether a signing key event rides along.

Nothing is verified: no hash is recomputed and no signature is checked.
Use 'truestamp verify' for that. The structural gates of whitepaper
Appendix E.6 still apply, so a malformed bundle, or one in the
pre-publication draft layout, is rejected with the same identifier
'verify' would report.

Examples:
  truestamp inspect proof.json
  truestamp inspect proof.cbor --json
  cat proof.json | truestamp inspect

Exit code 0 when the bundle parses, 1 when it is rejected.`,
	Args:          cobra.MaximumNArgs(1),
	SilenceUsage:  true,
	SilenceErrors: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		jsonOut, _ := cmd.Flags().GetBool("json")
		silent, _ := cmd.Flags().GetBool("silent")
		if jsonOut && silent {
			return fmt.Errorf("--silent and --json are mutually exclusive")
		}

		positional := ""
		if len(args) > 0 {
			positional = args[0]
		}
		fileFlag, _ := cmd.Flags().GetString("file")
		urlFlag, _ := cmd.Flags().GetString("url")
		data, src, err := inputsrc.Resolve(cmd.Context(), inputsrc.Options{
			PositionalArg:        positional,
			FileFlag:             fileFlag,
			URLFlag:              urlFlag,
			AllowStdin:           true,
			AutoDetectURL:        true,
			PickerTitle:          "Select proof file",
			PickerExts:           []string{".json", ".cbor"},
			URLPromptTitle:       "Enter proof URL",
			URLPromptPlaceholder: "https://example.com/proof.json",
		})
		if err != nil {
			if errors.Is(err, inputsrc.ErrNoInput) {
				_ = cmd.Help()
				return nil
			}
			if silent {
				return errSilentFail
			}
			return err
		}

		bundle, err := proof.ParseBytes(data)
		if err != nil {
			if silent {
				return errSilentFail
			}
			if proof.RejectionCode(err) != "" {
				if jsonOut {
					if jErr := emitJSON(cmd.OutOrStdout(), verify.BuildJSONRejection(err)); jErr != nil {
						return jErr
					}
				} else {
					verify.PresentRejection(cmd.OutOrStdout(), err)
				}
				return errSilentFail
			}
			return err
		}

		summary := inspectSummary(bundle, src.DisplayName(), len(data))
		switch {
		case silent:
		case jsonOut:
			return emitJSON(cmd.OutOrStdout(), summary)
		default:
			lipgloss.Print(renderInspect(summary))
		}
		return nil
	},
}

// InspectSummary is the machine-readable shape of `truestamp inspect`.
type InspectSummary struct {
	Source          string           `json:"source"`
	Format          string           `json:"format"`
	Bytes           int              `json:"bytes"`
	Version         json.RawMessage  `json:"version"`
	Type            string           `json:"type"`
	TypeCode        int              `json:"type_code"`
	GeneratedAt     string           `json:"generated_at"`
	PublicKey       string           `json:"public_key"`
	KeyID           string           `json:"key_id,omitempty"`
	Subject         *InspectSubject  `json:"subject,omitempty"`
	InclusionProof  *InspectProof    `json:"inclusion_proof,omitempty"`
	Block           InspectBlock     `json:"block"`
	BlockPath       []InspectBlock   `json:"block_path,omitempty"`
	Commitments     []InspectCommit  `json:"commitments"`
	SigningKeyEvent *InspectKeyEvent `json:"signing_key_event,omitempty"`
}

// InspectSubject describes the subject map.
type InspectSubject struct {
	ID                 string            `json:"id"`
	SigningKeyID       string            `json:"signing_key_id"`
	ClaimsKeys         []string          `json:"claims_keys,omitempty"`
	EntropyKeys        []string          `json:"entropy_keys,omitempty"`
	CommittedWitnesses map[string]string `json:"committed_witnesses"`
	CarriedWitnesses   []string          `json:"carried_witnesses"`
}

// InspectProof describes a compact Merkle proof without walking it.
type InspectProof struct {
	Depth int    `json:"depth"`
	Error string `json:"error,omitempty"`
}

// InspectBlock describes a block map.
type InspectBlock struct {
	ID                string   `json:"id"`
	PreviousBlockHash string   `json:"previous_block_hash"`
	MerkleRoot        string   `json:"merkle_root"`
	SigningKeyID      string   `json:"signing_key_id"`
	MetadataKeys      []string `json:"metadata_keys"`
}

// InspectCommit describes a commitment entry.
type InspectCommit struct {
	Chain           string        `json:"chain"`
	Network         string        `json:"network,omitempty"`
	EpochMerkleRoot string        `json:"epoch_merkle_root"`
	EpochProof      *InspectProof `json:"epoch_proof"`
	TransactionHash string        `json:"transaction_hash,omitempty"`
	Timestamp       string        `json:"timestamp,omitempty"`
	Ledger          *int          `json:"ledger,omitempty"`
	BlockHeight     *int          `json:"block_height,omitempty"`
	OfflineEvidence []string      `json:"offline_evidence,omitempty"`
}

// InspectKeyEvent describes the signing key event.
type InspectKeyEvent struct {
	Block       InspectBlock    `json:"block"`
	Type        string          `json:"type,omitempty"`
	KeyID       string          `json:"key_id,omitempty"`
	Sequence    json.RawMessage `json:"sequence,omitempty"`
	Commitments []InspectCommit `json:"commitments"`
}

func inspectSummary(b *proof.Bundle, source string, size int) InspectSummary {
	format := "json"
	if b.FromCBOR {
		format = "cbor"
	}
	version := b.VersionLiteral
	if len(version) == 0 {
		version = json.RawMessage("null")
	}
	s := InspectSummary{
		Source:      source,
		Format:      format,
		Bytes:       size,
		Version:     version,
		Type:        b.Type,
		TypeCode:    int(b.Code),
		GeneratedAt: b.GeneratedAt,
		PublicKey:   b.PublicKey,
		Block:       inspectBlock(b.Block),
		Commitments: inspectCommits(b.Commitments),
	}
	if pub, err := tscrypto.DecodePublicKey(b.PublicKey); err == nil {
		s.KeyID = tscrypto.ComputeKeyID(pub)
	}
	if sub := b.Subject; sub != nil {
		is := &InspectSubject{
			ID:                 sub.ID,
			SigningKeyID:       sub.SigningKeyID,
			CommittedWitnesses: map[string]string{},
			CarriedWitnesses:   sub.WitnessNamesCarried(),
		}
		if is.CarriedWitnesses == nil {
			is.CarriedWitnesses = []string{}
		}
		committed := sub.CommittedWitnesses()
		for _, name := range committed.Keys() {
			is.CommittedWitnesses[name] = committed.Str(name)
		}
		if claims, ok := sub.Fields.Object("claims"); ok {
			is.ClaimsKeys = claims.Keys()
		}
		if entropy, ok := sub.Fields.Object("entropy"); ok {
			is.EntropyKeys = entropy.Keys()
		}
		s.Subject = is
		s.InclusionProof = inspectProof(b.InclusionProof)
	}
	for _, entry := range b.BlockPath {
		s.BlockPath = append(s.BlockPath, inspectBlock(entry))
	}
	if e := b.SigningKeyEvent; e != nil {
		ke := &InspectKeyEvent{Block: inspectBlock(e.Block), Commitments: inspectCommits(e.Commitments)}
		if event, ok := e.KeyEvent(); ok {
			ke.Type, ke.KeyID, ke.Sequence = event.Type, event.KeyID, event.Sequence
		}
		s.SigningKeyEvent = ke
	}
	return s
}

func inspectBlock(b proof.BlockMap) InspectBlock {
	out := InspectBlock{
		ID:                b.ID,
		PreviousBlockHash: b.PreviousBlockHash,
		MerkleRoot:        b.MerkleRoot,
		SigningKeyID:      b.SigningKeyID,
		MetadataKeys:      []string{},
	}
	if meta, ok := b.Fields.Object("metadata"); ok {
		out.MetadataKeys = meta.Keys()
	}
	return out
}

func inspectProof(encoded string) *InspectProof {
	steps, err := tscrypto.DecodeCompactMerkleProof(encoded)
	if err != nil {
		return &InspectProof{Error: err.Error()}
	}
	return &InspectProof{Depth: len(steps)}
}

func inspectCommits(entries []proof.Commitment) []InspectCommit {
	out := make([]InspectCommit, 0, len(entries))
	for _, c := range entries {
		ic := InspectCommit{
			Chain:           c.Chain,
			Network:         c.Network,
			EpochMerkleRoot: c.EpochMerkleRoot,
			EpochProof:      inspectProof(c.EpochProof),
			TransactionHash: c.TransactionHash,
			Timestamp:       c.Timestamp,
		}
		if c.HasLedger {
			ledger := c.Ledger
			ic.Ledger = &ledger
		}
		if c.HasBlockHeight {
			height := c.BlockHeight
			ic.BlockHeight = &height
		}
		if c.RawTransaction != "" {
			ic.OfflineEvidence = append(ic.OfflineEvidence, "raw_transaction")
		}
		if c.Txoutproof != "" {
			ic.OfflineEvidence = append(ic.OfflineEvidence, "txoutproof")
		}
		if c.BlockMerkleRoot != "" {
			ic.OfflineEvidence = append(ic.OfflineEvidence, "block_merkle_root")
		}
		out = append(out, ic)
	}
	return out
}

func renderInspect(s InspectSummary) string {
	var b strings.Builder
	header := ui.AccentBoldStyle().Render("  Proof Bundle")
	b.WriteString(header + "\n\n")
	row := func(label, value string) {
		fmt.Fprintf(&b, "  %-22s %s\n", label, value)
	}
	row("Source", fmt.Sprintf("%s (%s, %d bytes)", s.Source, s.Format, s.Bytes))
	row("Version", string(s.Version))
	row("Type", fmt.Sprintf("%s (code %d)", s.Type, s.TypeCode))
	row("Generated at", orDash(s.GeneratedAt))
	row("Public key", orDash(s.PublicKey))
	row("Derived key id", orDash(s.KeyID))
	if sub := s.Subject; sub != nil {
		b.WriteString("\n" + ui.SectionHeader("Subject") + "\n")
		row("ID", sub.ID)
		row("Signing key id", sub.SigningKeyID)
		if len(sub.ClaimsKeys) > 0 {
			row("Claims keys", strings.Join(sub.ClaimsKeys, ", "))
		}
		if len(sub.EntropyKeys) > 0 {
			row("Entropy keys", strings.Join(sub.EntropyKeys, ", "))
		}
		names := make([]string, 0, len(sub.CommittedWitnesses))
		for name := range sub.CommittedWitnesses {
			names = append(names, name)
		}
		sortStrings(names)
		for i, name := range names {
			label := ""
			if i == 0 {
				label = "Committed witnesses"
			}
			row(label, fmt.Sprintf("%s  %s", name, sub.CommittedWitnesses[name]))
		}
		carried := "(none)"
		if len(sub.CarriedWitnesses) > 0 {
			carried = strings.Join(sub.CarriedWitnesses, ", ")
		}
		row("Carried witnesses", carried)
		if s.InclusionProof != nil {
			row("Inclusion proof", describeProof(s.InclusionProof))
		}
	}
	b.WriteString("\n" + ui.SectionHeader("Block") + "\n")
	writeInspectBlock(&b, row, s.Block)
	if len(s.BlockPath) > 0 {
		row("Block path", fmt.Sprintf("%d blocks link the head block to this one", len(s.BlockPath)))
	}
	b.WriteString("\n" + ui.SectionHeader("Commitments") + "\n")
	writeInspectCommits(&b, row, s.Commitments)
	b.WriteString("\n" + ui.SectionHeader("Signing Key Event") + "\n")
	if s.SigningKeyEvent == nil {
		row("Carried", "no")
	} else {
		row("Carried", "yes")
		row("Key event", fmt.Sprintf("type %s, sequence %s, key id %s", orDash(s.SigningKeyEvent.Type), string(s.SigningKeyEvent.Sequence), orDash(s.SigningKeyEvent.KeyID)))
		writeInspectBlock(&b, row, s.SigningKeyEvent.Block)
		writeInspectCommits(&b, row, s.SigningKeyEvent.Commitments)
	}
	return b.String()
}

func writeInspectBlock(b *strings.Builder, row func(string, string), blk InspectBlock) {
	row("Block id", orDash(blk.ID))
	row("Previous block hash", orDash(blk.PreviousBlockHash))
	row("Merkle root", orDash(blk.MerkleRoot))
	row("Signing key id", orDash(blk.SigningKeyID))
	meta := "{} (empty)"
	if len(blk.MetadataKeys) > 0 {
		meta = strings.Join(blk.MetadataKeys, ", ")
	}
	row("Metadata keys", meta)
}

func writeInspectCommits(b *strings.Builder, row func(string, string), commits []InspectCommit) {
	for i, c := range commits {
		where := c.Chain
		if c.Network != "" {
			where += " " + c.Network
		}
		row(fmt.Sprintf("Commitment %d", i+1), where)
		row("  Epoch root", c.EpochMerkleRoot)
		row("  Epoch proof", describeProof(c.EpochProof))
		if c.TransactionHash != "" {
			row("  Transaction", c.TransactionHash)
		}
		if c.Ledger != nil {
			row("  Ledger", fmt.Sprintf("%d", *c.Ledger))
		}
		if c.BlockHeight != nil {
			row("  Block height", fmt.Sprintf("%d", *c.BlockHeight))
		}
		if c.Timestamp != "" {
			row("  Timestamp", c.Timestamp)
		}
		if len(c.OfflineEvidence) > 0 {
			row("  Offline evidence", strings.Join(c.OfflineEvidence, ", "))
		}
	}
}

func describeProof(p *InspectProof) string {
	if p == nil {
		return "-"
	}
	if p.Error != "" {
		return "undecodable: " + p.Error
	}
	if p.Depth == 1 {
		return "1 step"
	}
	return fmt.Sprintf("%d steps", p.Depth)
}

func orDash(s string) string {
	if s == "" {
		return "-"
	}
	return s
}

func sortStrings(s []string) {
	for i := 1; i < len(s); i++ {
		for j := i; j > 0 && s[j] < s[j-1]; j-- {
			s[j], s[j-1] = s[j-1], s[j]
		}
	}
}

func init() {
	f := inspectCmd.Flags()
	f.String("file", "", "Path to proof file (interactive picker if no path given)")
	f.String("url", "", "URL to download proof from (interactive prompt if no URL given)")
	f.Lookup("file").NoOptDefVal = inputsrc.FilePickSentinel
	f.Lookup("url").NoOptDefVal = inputsrc.URLPromptSentinel
	f.Bool("json", false, "Output the summary as JSON")
	f.BoolP("silent", "s", false, "No output, exit code only")
	rootCmd.AddCommand(inspectCmd)
}
