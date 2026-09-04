// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: MIT

package cmd

import (
	"errors"
	"fmt"
	"io"
	"os"
	"slices"
	"strings"

	"github.com/spf13/cobra"
	"github.com/truestamp/truestamp-cli/internal/inputsrc"
	"github.com/truestamp/truestamp-cli/internal/proof"
	"github.com/truestamp/truestamp-cli/internal/ui"
)

// Subject-type flag values for `truestamp proofs get --type` and
// `truestamp verify --type`. These map 1:1 to the server's /proof/generate
// `type` string enum. There is no "auto" and no bare "entropy", both were
// removed in the server's strict-type cutover. Callers that don't pass
// --type get a default: a ULID id is an item, and a UUIDv7 id is resolved
// against the server in one extra round trip.
//
// Wire values preserve the underscore form (entropy_nist) to match the
// server enum exactly; filename stems translate underscores to hyphens
// for friendlier filenames (truestamp-entropy-nist-<id>.json).
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

// proofTypesForUUIDv7 lists the --type values that may be used with a
// UUIDv7 id (everything except "item").
var proofTypesForUUIDv7 = []string{
	proofTypeEntropyNIST,
	proofTypeEntropyStellar,
	proofTypeEntropyBitcoin,
	proofTypeBlock,
	proofTypeBeacon,
}

var proofsGetCmd = &cobra.Command{
	Use:   "get <id>",
	Short: "Fetch the proof bundle for a subject",
	Long: `Download a cryptographic proof bundle for a Truestamp subject.

--type is optional. A ULID is unambiguously an item. A UUIDv7 could be an
entropy observation, a block or a beacon, so it is resolved against the
server in one extra round trip; pass --type to skip that call, or when an
id is verifiable as more than one subject type, which is refused rather
than guessed.

--witnesses selects which witness details an item bundle carries:
'all' (the default, the complete bundle), 'none' (the compact bundle:
the committed witness hashes stay in the subject metadata, the details
are left out), or a comma-separated subset of block, entropy_stellar,
entropy_nist, entropy_bitcoin, signing_key_event (a partial bundle).
All three are ordinary version 1 bundles; the only difference a verifier
sees is how many witness rows it can report.

With no output flag the bundle is written to stdout, so it can be piped.
-o/--out writes it to the path you name. --to-file writes it to a
conventionally-named file in the current directory:
truestamp-<stem>-<id><variant>.<ext>, where <stem> is the resolved type
with underscores translated to hyphens and <variant> is empty, -compact,
or -partial:

  --type item                      -> truestamp-item-<ulid>.<ext>
  --type item --witnesses none     -> truestamp-item-<ulid>-compact.<ext>
  --type item --witnesses block    -> truestamp-item-<ulid>-partial.<ext>
  --type entropy_nist              -> truestamp-entropy-nist-<uuidv7>.<ext>
  --type block                     -> truestamp-block-<uuidv7>.<ext>
  --type beacon                    -> truestamp-beacon-<uuidv7>.<ext>

The 'type' inside the file is authoritative; the filename never is.
Block and beacon proofs share one structural shape but carry distinct
type codes in the signed payload, so a block and a beacon proof for the
same block have different signatures.

Examples:
  truestamp proofs get 01KNN33GX5E470CB9TRWAYF9DD
  truestamp proofs get --witnesses none 01KNN33GX5E470CB9TRWAYF9DD
  truestamp proofs get --witnesses block,entropy_nist 01KNN33GX5E470CB9TRWAYF9DD
  truestamp proofs get --type block   019d6a32-13e6-72b0-97e5-3779231ea97b
  truestamp proofs get --type beacon -f cbor 019d6a32-13e6-72b0-97e5-3779231ea97b
  truestamp proofs get --type entropy_stellar 019cf813-99b8-730a-84f1-5a711a9c355e
  truestamp proofs get -o proof.json 01KNN33GX5E470CB9TRWAYF9DD

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
		id := args[0]

		if err := requireAuth(cmd); err != nil {
			return err
		}

		format, _ := cmd.Flags().GetString("format")
		format = strings.ToLower(strings.TrimSpace(format))
		if format != "json" && format != "cbor" {
			return fmt.Errorf("--format must be \"json\" or \"cbor\", got %q", format)
		}

		// Everything that can be refused locally is refused before the
		// first round trip: a bad --witnesses or a conflicting output
		// choice must not cost a resolve-id request.
		witnessFlag, _ := cmd.Flags().GetString("witnesses")
		witnesses, err := proof.ParseWitnessSelection(witnessFlag)
		if err != nil {
			return fmt.Errorf("--witnesses: %w", err)
		}
		outPath, _ := cmd.Flags().GetString("out")
		toFile, _ := cmd.Flags().GetBool("to-file")
		if outPath != "" && toFile {
			return fmt.Errorf("--out and --to-file are mutually exclusive: --out names a path, --to-file picks the conventional name")
		}
		if outPath == "" && !toFile && format == "cbor" && inputsrc.IsStdoutTerminal() {
			// Refuse to spray CBOR at a terminal. The message names both
			// ways out rather than just failing.
			return fmt.Errorf("refusing to write CBOR to a terminal: redirect it, or pass -o <path> or --to-file")
		}

		typeFlag, _ := cmd.Flags().GetString("type")
		typeFlag = strings.ToLower(strings.TrimSpace(typeFlag))

		// Pre-flight id-shape validation, catches obvious typos before
		// the network round-trip and is also the basis for the smart
		// --type default.
		shape, err := proof.DetectIDType(id)
		if err != nil {
			return err
		}

		// Resolve --type when not specified.
		//
		// A ULID is unambiguously an item, so that case needs no help. A
		// UUIDv7 could be a block, a beacon or an entropy observation, and
		// nothing client-side can tell them apart — this is the one place
		// id-shape dispatch cannot work. Rather than requiring --type, ask
		// the server what the id refers to, at the cost of one round trip.
		//
		// The classification only decides which proof to REQUEST. The
		// bundle that comes back is still verified against its own signed
		// type, so a wrong answer here cannot change a verdict.
		if typeFlag == "" {
			switch shape {
			case proof.IDTypeULID:
				typeFlag = proofTypeItem
			case proof.IDTypeUUIDv7:
				resolved, rErr := proof.ResolveSubjectType(cmd.Context(), cfg.APIURL, cfg.Team, id)
				if rErr != nil {
					return fmt.Errorf("%w\n(or pass --type explicitly: %s)",
						rErr, strings.Join(proofTypesForUUIDv7, " | "))
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

		appLogger.Info("download_request", "id", id, "type", typeFlag, "format", format, "witnesses", witnesses.String())
		data, err := proof.GenerateCtx(cmd.Context(), cfg.APIURL, cfg.Team, id, typeFlag, format, witnesses)
		if err != nil {
			appLogger.Error("download_failed", "id", id, "type", typeFlag, "err", err.Error())
			var apiErr *proof.GenerateAPIError
			if errors.As(err, &apiErr) {
				return explainGenerateError(apiErr)
			}
			return err
		}

		stem := proofFileStem(typeFlag)

		// R10's payload triad: no flag writes the bundle to stdout so it
		// can be piped, -o/--out names a path, --to-file uses the
		// conventional auto-name. Previously this command always wrote a
		// file into the cwd, which made
		// `truestamp proofs get <id> | truestamp verify` impossible without
		// a temp file in a CLI that advertises pipeline recipes.
		if toFile {
			outPath = fmt.Sprintf("truestamp-%s-%s%s.%s", stem, id, witnesses.FilenameSuffix(), format)
		}

		if outPath == "" {
			if _, werr := cmd.OutOrStdout().Write(data); werr != nil {
				return fmt.Errorf("writing to stdout: %w", werr)
			}
			appLogger.Info("download_completed",
				"id", id, "type", typeFlag, "format", format,
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
			"format", format,
			"size_bytes", len(data),
			"output", outPath,
		)

		// The receipt card goes to stderr, so stdout stays usable even
		// when a file was written.
		presentDownload(cmd.ErrOrStderr(), outPath, format, id, typeFlag, witnesses.String(), len(data))
		return nil
	},
}

// downloadStem converts a resolved --type value to a filename stem. Wire
// values use underscores (entropy_nist) to match the server enum;
// filename stems use hyphens (entropy-nist) for friendlier filenames.
func proofFileStem(typeFlag string) string {
	return strings.ReplaceAll(typeFlag, "_", "-")
}

// validProofType reports whether v is one of the six canonical --type
// values.
func validProofType(v string) bool {
	return slices.Contains(proofTypeValues, v)
}

// validateTypeVsShape ensures --type matches the syntactic shape of the
// positional id. `item` requires a ULID; every other type requires a
// UUIDv7. Runs locally before the network call so users get a targeted
// error rather than a generic server 422.
func validateTypeVsShape(typeFlag string, shape proof.IDType) error {
	switch typeFlag {
	case proofTypeItem:
		if shape != proof.IDTypeULID {
			return fmt.Errorf("--type %s requires a ULID id (e.g. 01KNN33GX5E470CB9TRWAYF9DD); got a UUIDv7", typeFlag)
		}
	default:
		// entropy_*, block, beacon, all UUIDv7.
		if shape != proof.IDTypeUUIDv7 {
			return fmt.Errorf("--type %s requires a UUIDv7 id (e.g. 019d6a32-13e6-72b0-97e5-3779231ea97b); got a ULID", typeFlag)
		}
	}
	return nil
}

func presentDownload(w io.Writer, filename, format, id, typeFlag, witnesses string, size int) {
	header := ui.AccentBoldStyle().Render("  Proof Downloaded")

	formatDisplay := strings.ToUpper(format)

	tbl := ui.CompactTable().
		StyleFunc(ui.LabelValueStyleFunc()).
		Row("File", filename).
		Row("Format", fmt.Sprintf("%s (%s bytes)", formatDisplay, formatSize(size))).
		Row("ID", id).
		Row("Type", typeFlag).
		Row("Witnesses", witnesses)

	// Append URL rows to the SAME table so they share the
	// right-aligned-label / value column alignment. ui.SubjectDetailURL
	// routes beacon downloads to /blocks/<id> because the id we have
	// is the block id; the hash-keyed /beacons/<hash> form lives on
	// the beacon listing card where the hash comes directly from the
	// API response.
	if detail := ui.SubjectDetailURL(appConfig.APIURL, typeFlag, id); detail != "" {
		tbl = tbl.Row("Details", detail)
	}
	if verify := ui.SubjectVerifyURL(appConfig.APIURL, typeFlag, id); verify != "" {
		tbl = tbl.Row("Verify", verify)
	}

	// Plain newline-join, see note in internal/verify/presenter.go
	// Present(). Long filenames (e.g. truestamp-entropy-bitcoin-<uuidv7>.cbor)
	// won't inflate every other table row on narrow terminals.
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
	f.StringP("format", "f", "json", `Output format: "json" or "cbor"`)
	f.StringP("out", "o", "", "Write the bundle to this path (default: stdout)")
	f.Bool("to-file", false, "Write the bundle to a conventionally-named file in the current directory")
	f.String("type", "",
		fmt.Sprintf(`Subject type. Optional: a ULID is an item, and a UUIDv7 is resolved against the server in one extra round trip. One of: %s`,
			strings.Join(proofTypeValues, " | ")))
	f.String("witnesses", "all", "Witness details to carry: all, none, or a comma-separated list of "+strings.Join(proof.WitnessNames, ","))
	proofsCmd.AddCommand(proofsGetCmd)
}

// retryableGenerateCodes mirrors the server's own closed retryable set.
// Terminality is a property of the CODE, never of the wording of `detail`:
// detail is human-readable text the server may reword at any time, and it
// has been reworded once already. A code this CLI does not recognise
// appears in neither map and gets no verdict at all, so a code added
// server-side before the CLI learns about it renders plainly rather than
// inheriting a guess about whether waiting helps.
var retryableGenerateCodes = map[string]bool{
	proof.GenerateCodeNoExternalCommitments: true,
	proof.GenerateCodeSubjectNotReady:       true,
}

var terminalGenerateCodes = map[string]bool{
	proof.GenerateCodeSubjectNotRecomputable: true,
	proof.GenerateCodeGenerationFailed:       true,
}

// explainGenerateError turns a /proof/generate refusal into something the
// holder of the subject can act on.
//
// The server says accurately what went wrong; what it cannot say is what
// the caller should do next, and the single most useful part of that is
// whether waiting will help. `no_external_commitments` clears on its own
// in minutes; `subject_not_recomputable` never clears. Reporting both as
// "API error" left a holder polling forever against a permanent condition.
func explainGenerateError(e *proof.GenerateAPIError) error {
	var b strings.Builder
	// e.Error() carries the HTTP status and the server's meta.code, both of
	// which are worth keeping: the status separates "refused" from
	// "unreachable", and the code is the string to quote in a bug report.
	b.WriteString(e.Error())

	// A labelled verdict rather than a sentence, so it cannot read as a
	// clumsy echo of whatever the server's own prose already says.
	switch {
	case retryableGenerateCodes[e.Code]:
		b.WriteString("\n\nRetry: yes, this is transient.")
	case terminalGenerateCodes[e.Code]:
		b.WriteString("\n\nRetry: no, this condition is permanent.")
	}

	switch e.Code {
	case proof.GenerateCodeNoExternalCommitments:
		b.WriteString(" A proof exists only after the subject's first")
		b.WriteString("\npublic-chain commitment; items commit to a Truestamp block within about")
		b.WriteString("\na minute and to Stellar within about five. Try again shortly.")

	case proof.GenerateCodeSubjectNotReady:
		b.WriteString(" The subject is not yet in a state a proof can be")
		b.WriteString("\nbuilt from. Try again shortly.")

	case proof.GenerateCodeSubjectNotRecomputable:
		if e.Drifted != "" {
			fmt.Fprintf(&b, "\n\nWhat drifted: %s.", e.Drifted)
		}
		b.WriteString("\n\nThe subject's commitment on the public chain is unaffected; what")
		b.WriteString("\nchanged is that its stored data no longer reproduces the hash that")
		b.WriteString("\nwas committed. Report the subject id to Truestamp.")

	case proof.GenerateCodeGenerationFailed:
		// Comparing two values the server sent, not sniffing its prose for
		// a keyword: the steps are already inside `detail` today, and
		// printing the identical list twice reads as two failures.
		if e.FailedSteps != "" && !strings.Contains(e.Detail, e.FailedSteps) {
			fmt.Fprintf(&b, "\n\nFailed checks: %s", e.FailedSteps)
		}
	}

	return errors.New(b.String())
}
