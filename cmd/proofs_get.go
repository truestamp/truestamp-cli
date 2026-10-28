// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"slices"
	"strings"

	"github.com/spf13/cobra"
	"github.com/truestamp/truestamp-cli/internal/itemcode"
	"github.com/truestamp/truestamp-cli/internal/jsonapi"
	"github.com/truestamp/truestamp-cli/internal/proof"
	"github.com/truestamp/truestamp-cli/internal/ui"
)

// Subject-type flag values for `truestamp proofs get --type`. These map 1:1
// to the `type` of the server's raw proof route, the six type names of
// SPEC.md section 3.2 (Members of `proof`). There is no "auto" and no bare
// "entropy". Callers that don't pass --type get a default: an item code
// names an item, with no round trip, and a UUIDv7 id is resolved against
// the server in one extra round trip.
const (
	proofTypeItem           = "item"
	proofTypeEntropyNIST    = "entropy_nist"
	proofTypeEntropyStellar = "entropy_stellar"
	proofTypeEntropyBitcoin = "entropy_bitcoin"
	proofTypeBlock          = "block"
	proofTypeBeacon         = "beacon"
)

// proofTypeValues lists every accepted --type value in the order they
// appear in user-facing help text.
var proofTypeValues = []string{
	proofTypeItem,
	proofTypeEntropyNIST,
	proofTypeEntropyStellar,
	proofTypeEntropyBitcoin,
	proofTypeBlock,
	proofTypeBeacon,
}

var proofsGetCmd = &cobra.Command{
	Use:   "get <id-code-or-hash>",
	Short: "Fetch the proof bundle for a subject",
	Long: `Download the proof Truestamp serves of a subject, its bytes exactly as
served: one JSON document, the envelope of SPEC.md section 3.1, signed
by Truestamp.

The subject is named by its UUIDv7 id, and an item also by its item code
or its item_hash, each read exactly as typed: nothing is trimmed, and a
UUID's 32-hex, braced and urn: forms are refused. --type is optional:

  an item code (e.g. 9WBPKNV7SCP87VZW)  names only an item: --type item,
                                        with no round trip
  an item_hash (64 hex characters)      needs --type item
  a UUIDv7                              could be an item, an entropy
                                        observation, a block or a beacon,
                                        so it is resolved against the
                                        server in one extra round trip

Pass --type to skip that call. A block's id is verifiable as a block and as
a beacon, which is refused rather than guessed: pass --type block or --type
beacon.

An item's proof fetched by its UUIDv7, its item code or its item_hash is
checked before anything is written: the record the proof carries must
have that id, or the item_hash recomputed from that record must give
that item code, or be that item_hash, all of it. Otherwise the run
refuses with "the server returned a proof for a different item" and
writes nothing. A matching code shows only that the
recomputed item_hash begins with the 75 bits the code is written from.

--constructions selects how the proof carries the constructions text
(SPEC.md section 7.2): 'full' (the default) carries the text itself,
'reference' its URL, which names its SHA-256, and 'none' neither. Each
mode is its own signed document, and the verdict never depends on it.

With no output flag the proof is written to stdout, so it can be piped.
-o/--out writes it to the path you name. --to-file writes it to a
conventionally-named file in the current directory, whatever the
constructions mode:

  an item                          -> <CODE>.proof.json, CODE its item code
  an entropy observation, a block
  or a beacon                      -> <uuidv7>.proof.json

The 'type' inside the proof is authoritative; the filename never is.
Block and beacon proofs share one structural shape but carry distinct
types in the signed proof, so a block and a beacon proof for the same
block have different signatures.

Examples:
  truestamp proofs get 01a0db66-c2e0-78a4-b8cc-12b11bc97493
  truestamp proofs get --to-file 9WBPKNV7SCP87VZW
  truestamp proofs get --type item 4f1769d767cb2c83eff1c48165939090d42fac433c5eabe6f02c5cf9d025181b
  truestamp proofs get --type item --constructions reference 01a0db66-c2e0-78a4-b8cc-12b11bc97493
  truestamp proofs get --type block   019d6a32-13e6-72b0-97e5-3779231ea97b
  truestamp proofs get --type beacon --to-file 019d6a32-13e6-72b0-97e5-3779231ea97b
  truestamp proofs get --type entropy_stellar 019cf813-99b8-730a-84f1-5a711a9c355e
  truestamp proofs get -o proof.json 01a0db66-c2e0-78a4-b8cc-12b11bc97493

Requires authentication, run 'truestamp auth login', or set TRUESTAMP_API_KEY / --api-key for headless/CI use.

Exit code 0 on success, 1 on any error (validation failure, network
error, missing API key, server rejection, or failure to write the
output file).`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) == 0 {
			return cmd.Help()
		}

		cfg := appConfig
		// The id exactly as typed: DetectIDType trims nothing.
		typed := args[0]

		if err := requireAuth(cmd); err != nil {
			return err
		}

		// Everything that can be refused locally is refused before the
		// first round trip: a bad --constructions or a conflicting output
		// choice must not cost a resolve-id request.
		constructions, _ := cmd.Flags().GetString(proof.ConstructionsParameter)
		if !slices.Contains(proof.ConstructionsModes, constructions) {
			return fmt.Errorf("--constructions must be one of %s, got %q",
				strings.Join(proof.ConstructionsModes, " | "), constructions)
		}
		outPath, _ := cmd.Flags().GetString("out")
		toFile, _ := cmd.Flags().GetBool("to-file")
		if outPath != "" && toFile {
			return fmt.Errorf("--out and --to-file are mutually exclusive: --out names a path, --to-file picks the conventional name")
		}

		typeFlag, _ := cmd.Flags().GetString("type")
		typeFlag = strings.ToLower(strings.TrimSpace(typeFlag))

		// Pre-flight id-shape validation, catches obvious typos before
		// the network round-trip and is also the basis for the smart
		// --type default. From here on the id is its canonical form.
		shape, id, err := proof.DetectIDType(typed)
		if err != nil {
			return nameTheFormTaken(err, proofsGetTakes)
		}

		// Resolve --type when not specified.
		//
		// An item code names only an item, so it needs no round trip. A
		// UUIDv7 could be an item, a block, a beacon or an entropy
		// observation, and nothing client-side can tell them apart: this
		// is the one place id-shape dispatch cannot work. Rather than
		// requiring --type, ask the server what the id refers to, at the
		// cost of one round trip. 64 hex characters name a subject only
		// as an item's item_hash, which the server's resolution does not
		// read, so they need --type item.
		//
		// The classification only decides which proof to REQUEST. The
		// proof that comes back is still verified against its own signed
		// type, so a wrong answer here cannot change a verdict.
		if typeFlag == "" {
			switch shape {
			case proof.IDTypeItemCode:
				typeFlag = proofTypeItem
			case proof.IDTypeItemHash:
				return fmt.Errorf("%q is 64 hex characters, which names a subject only as an item's item_hash: pass --type item to fetch that item's proof", typed)
			case proof.IDTypeUUIDv7:
				resolved, rErr := proof.ResolveSubjectType(cmd.Context(), cfg.APIURL, cfg.Team, id)
				if rErr != nil {
					if errors.Is(rErr, jsonapi.ErrRateLimited) {
						return renderAPIError(cmd, rErr, "proof")
					}
					if errors.Is(rErr, proof.ErrSeveralTypes) {
						return rErr
					}
					return fmt.Errorf("%w\n(or pass --type explicitly: %s)",
						rErr, strings.Join(proofTypeValues, " | "))
				}
				typeFlag = resolved
				appLogger.Info("proof_type_resolved", "id", id, "type", typeFlag)
			default:
				// DetectIDType should have errored already; belt-and-suspenders.
				return fmt.Errorf("unrecognised id shape %q", shape)
			}
		}

		if !validProofType(typeFlag) {
			return fmt.Errorf("--type must be one of %s, got %q",
				strings.Join(proofTypeValues, " | "), typeFlag)
		}

		// Shape vs type cross-check, surfaces obvious mismatches locally
		// before the server refuses them with id_format_mismatch.
		if err := validateTypeVsShape(typeFlag, shape); err != nil {
			return err
		}

		appLogger.Info("download_request", "id", id, "type", typeFlag, "constructions_mode", constructions)
		data, err := proof.FetchRawCtx(cmd.Context(), cfg.APIURL, cfg.Team, typeFlag, id, constructions)
		if err != nil {
			appLogger.Error("download_failed", "id", id, "type", typeFlag, "err", err.Error())
			var apiErr *proof.GenerateAPIError
			if errors.As(err, &apiErr) {
				return explainGenerateError(cmd, apiErr)
			}
			if errors.Is(err, jsonapi.ErrRateLimited) {
				return renderAPIError(cmd, err, "proof")
			}
			return err
		}

		// An item asked for by its UUIDv7, its item code or its item_hash:
		// the proof served must be of that item, by its record's id or by
		// the item_hash recomputed from its record, before any byte is
		// written anywhere.
		if err := proof.CheckServedItem(typeFlag, shape, id, data); err != nil {
			appLogger.Error("download_failed", "id", id, "type", typeFlag, "stage", "check", "err", err.Error())
			return err
		}

		// R10's payload triad: no flag writes the proof to stdout so it
		// can be piped, -o/--out names a path, --to-file uses the
		// conventional name. Every destination gets the served bytes
		// unchanged.
		if toFile {
			outPath, err = proof.FileName(typeFlag, id, data)
			if err != nil {
				appLogger.Error("download_failed", "id", id, "type", typeFlag, "stage", "name", "err", err.Error())
				return err
			}
		}

		if outPath == "" {
			if _, werr := cmd.OutOrStdout().Write(data); werr != nil {
				return fmt.Errorf("writing to stdout: %w", werr)
			}
			appLogger.Info("download_completed",
				"id", id, "type", typeFlag, "constructions_mode", constructions,
				"size_bytes", len(data), "output", "(stdout)",
			)
			return nil
		}

		if err := os.WriteFile(outPath, data, 0644); err != nil {
			appLogger.Error("download_failed", "id", id, "type", typeFlag, "stage", "write", "err", err.Error())
			return fmt.Errorf("writing file: %w", err)
		}

		appLogger.Info("download_completed",
			"id", id,
			"type", typeFlag,
			"constructions_mode", constructions,
			"size_bytes", len(data),
			"output", outPath,
		)

		// The receipt card goes to stderr, so stdout stays usable even
		// when a file was written.
		presentDownload(cmd.ErrOrStderr(), outPath, shape, id, typeFlag, constructions, data)
		return nil
	},
}

// validProofType reports whether v is one of the six canonical --type
// values.
func validProofType(v string) bool {
	return slices.Contains(proofTypeValues, v)
}

// validateTypeVsShape ensures --type matches the syntactic shape of the
// positional id. `item` takes a UUIDv7, an item code or an item_hash, the
// three references the app's raw proof route reads for an item; every
// other type takes only a UUIDv7, as that route requires. Runs locally
// before the network call so users get a targeted error rather than the
// server's id_format_mismatch.
func validateTypeVsShape(typeFlag string, shape proof.IDType) error {
	if typeFlag == proofTypeItem || shape == proof.IDTypeUUIDv7 {
		return nil
	}
	got := "an item code"
	if shape == proof.IDTypeItemHash {
		got = "64 hex characters, an item_hash"
	}
	return fmt.Errorf("--type %s requires a UUIDv7 id (e.g. 019d6a32-13e6-72b0-97e5-3779231ea97b); got %s, which names only an item", typeFlag, got)
}

// proofsGetTakes names the forms `proofs get` takes, for a refusal of a
// value that is none of them.
const proofsGetTakes = "proofs get takes a UUIDv7 id (e.g. 019d6a32-13e6-72b0-97e5-3779231ea97b), " +
	"an item code (e.g. 9WBPKNV7SCP87VZW) or, with --type item, an item_hash"

// nameTheFormTaken adds to a refusal of proof.DetectIDType, which says
// what a value is, the form the command takes, so each command names only
// what it reads. Any other error is returned as it is.
func nameTheFormTaken(err error, takes string) error {
	if errors.Is(err, proof.ErrNotAnID) {
		return fmt.Errorf("%w: %s", err, takes)
	}
	return err
}

// presentDownload writes the receipt card of a proof written to filename:
// the subject, named as it was asked for, and the web pages of it. An
// item's card gives its item code beside its whole item_hash, both from
// the item_hash recomputed from the record the proof carries, and its
// pages are named by that code; an item proof with no record to recompute
// from gives neither, and no pages.
func presentDownload(w io.Writer, filename string, shape proof.IDType, id, typeFlag, constructions string, served []byte) {
	header := ui.AccentBoldStyle().Render("  Proof Downloaded")

	tbl := ui.CompactTable().
		StyleFunc(ui.LabelValueStyleFunc()).
		Row("File", filename).
		Row("Size", formatSize(len(served))+" bytes")
	if shape == proof.IDTypeUUIDv7 {
		tbl = tbl.Row("ID", id)
	}
	pageID := id
	if typeFlag == proofTypeItem {
		pageID = ""
		if itemHash, err := proof.ServedItemHash(served); err == nil {
			pageID = itemcode.FromItemHash(itemHash)
			tbl = tbl.Row("Item hash", hex.EncodeToString(itemHash[:])).
				Row("Item code", pageID)
		}
	}
	tbl = tbl.Row("Type", typeFlag).
		Row("Constructions", constructions)

	// Append URL rows to the SAME table so they share the
	// right-aligned-label / value column alignment. ui.SubjectDetailURL
	// routes beacon downloads to /blocks/<id> because the id we have
	// is the block id; the hash-keyed /beacons/<hash> form lives on
	// the beacon listing card where the hash comes directly from the
	// API response.
	if detail := ui.SubjectDetailURL(appConfig.APIURL, typeFlag, pageID); detail != "" {
		tbl = tbl.Row("Details", detail)
	}
	if verify := ui.SubjectVerifyURL(appConfig.APIURL, typeFlag, pageID); verify != "" {
		tbl = tbl.Row("Verify", verify)
	}

	// Plain newline-join: a long file name does not inflate every other
	// table row on narrow terminals.
	ui.Fprintln(w, strings.Join([]string{header, "", tbl.String()}, "\n"))
}

func formatSize(size int) string {
	switch {
	case size >= 1_000_000:
		return fmt.Sprintf("%d,%03d,%03d", size/1_000_000, (size/1_000)%1_000, size%1_000)
	case size >= 1_000:
		return fmt.Sprintf("%d,%03d", size/1_000, size%1_000)
	default:
		return fmt.Sprintf("%d", size)
	}
}

func init() {
	f := proofsGetCmd.Flags()
	f.String(proof.ConstructionsParameter, proof.ConstructionsFull, "How the proof carries the constructions text (SPEC.md section 7.2): "+strings.Join(proof.ConstructionsModes, " | "))
	f.StringP("out", "o", "", "Write the proof to this path (default: stdout)")
	f.Bool("to-file", false, "Write the proof to <CODE>.proof.json (an item) or <id>.proof.json in the current directory")
	f.String("type", "",
		fmt.Sprintf(`Subject type. Optional: an item code is an item, an item_hash needs item, and a UUIDv7 is resolved against the server in one extra round trip. One of: %s`,
			strings.Join(proofTypeValues, " | ")))
	proofsCmd.AddCommand(proofsGetCmd)
}

// generateCodeAdvice is what a holder can do next for each refusal of the
// raw proof route this CLI recognises: whether waiting helps, and the
// context worth adding to the server's own detail.
//
// Terminality is a property of the CODE, never of the wording of `detail`:
// detail is human-readable text the server may reword at any time. A code
// not in this table gets no verdict at all, so a code added server-side
// before the CLI learns about it renders plainly rather than inheriting a
// guess about whether waiting helps. The transient set mirrors the
// server's own retryable set.
var generateCodeAdvice = map[string]struct {
	transient bool
	advice    []string
}{
	proof.GenerateCodeNoBlockCommitment: {true, []string{
		"A proof exists once the subject's block is committed to a public chain.",
		"Items reach a Truestamp block within about a minute and Stellar within",
		"about five.",
	}},
	proof.GenerateCodeCommitmentNotFinal: {true, []string{
		"The block's commitment is on its chain and not final there yet.",
	}},
	proof.GenerateCodeEvidenceUnconfirmed: {true, []string{
		"Truestamp signs a proof only once its own chain reads confirm the",
		"evidence the proof carries.",
	}},
	proof.GenerateCodeSubjectNotReady: {true, []string{
		"The subject is not yet in a state a proof can be built from.",
	}},
	proof.GenerateCodeWitnessUnavailable: {true, []string{
		"Truestamp never serves an item's proof without every witness its",
		"fingerprint commits to.",
	}},
	proof.GenerateCodeSigningUnavailable: {true, nil},
	proof.GenerateCodeLookupFailed:       {true, nil},
	proof.GenerateCodeSubjectRedacted: {false, []string{
		"Restoring the item makes its proof available again.",
	}},
	proof.GenerateCodeProofTooLarge: {false, []string{
		"Report the subject id to Truestamp.",
	}},
	proof.GenerateCodeGenerationFailed: {false, nil},
}

// explainGenerateError renders a refusal of the raw proof route the way
// every other refusal in the tree is rendered: a banner naming the
// outcome, the server's own detail, what the holder can do next, and the
// code to quote in a bug report. Nothing under --silent.
//
// The server says accurately what went wrong; what it cannot say is what
// the caller should do next, and the single most useful part of that is
// whether waiting will help. `no_block_commitment` clears on its own in
// minutes; `subject_redacted` does not.
func explainGenerateError(cmd *cobra.Command, e *proof.GenerateAPIError) error {
	if _, silent := outputMode(cmd); silent {
		return errSilentFail
	}
	a, known := generateCodeAdvice[e.Code]
	banner := "Proof request refused"
	switch {
	case known && a.transient:
		banner = "Proof not available yet"
	case known:
		banner = "Proof cannot be generated"
	}

	var lines []string
	if e.Detail != "" {
		lines = append(lines, e.Detail)
	}
	lines = append(lines, a.advice...)
	switch {
	case known && a.transient:
		lines = append(lines, "This clears on its own: try again shortly.")
	case known:
		lines = append(lines, "This will not clear on its own.")
	}
	code := e.Code
	if code == "" {
		code = "no error code"
	}
	lines = append(lines, fmt.Sprintf("The server said %s (HTTP %d).", code, e.StatusCode))

	w := cmd.ErrOrStderr()
	ui.Fprintln(w, ui.FailureBanner(banner))
	for _, l := range lines {
		ui.Fprintln(w, ui.FaintStyle().Render("    "+l))
	}
	return errSilentFail
}
