// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: MIT

package cmd

import (
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/truestamp/truestamp-cli/internal/inputsrc"
	"github.com/truestamp/truestamp-cli/internal/proof"
	"github.com/truestamp/truestamp-cli/internal/verify"
)

// errVerificationFailed is returned when the proof report itself fails.
// The report has already been rendered so we only need a non-nil error for
// exit code 1. Execute() prints it via its default path.
var errVerificationFailed = errors.New("verification failed")

var verifyCmd = &cobra.Command{
	Use:   "verify [file-or-url]",
	Short: "Verify a Truestamp proof bundle",
	Long: `Cryptographically verify a Truestamp proof bundle JSON file.

Verifies the complete chain: signing keys, proof signature, item hashes,
Ed25519 signatures, Merkle proofs, block chain integrity, and public
blockchain commitments (Stellar, Bitcoin).

Proof input can be provided as:
  truestamp verify proof.json           Local file path
  truestamp verify https://host/p.json  URL (auto-detected)
  truestamp verify --file proof.json    Explicit file path
  truestamp verify --file               Interactive file picker
  truestamp verify --url https://...    Explicit URL download
  truestamp verify --url                Interactive URL prompt
  cat proof.json | truestamp verify     Pipe from stdin

The subject type is always read from the bundle's own signed 't'
field; the filename is never consulted. Pass --type to additionally
assert which type you expected. If the bundle's t doesn't match, the
report surfaces a Subject Type failure (local mode) or the server
rejects the request with subject_type_mismatch (--remote mode).
Values: item | entropy_nist | entropy_stellar | entropy_bitcoin |
block | beacon.

Use --remote to delegate verification to the Truestamp server API instead
of performing local computation. Requires authentication — run
'truestamp auth login', or set TRUESTAMP_API_KEY / --api-key for headless/CI use.

Exit code 0 on success, 1 on verification failure.`,
	Args:          cobra.MaximumNArgs(1),
	SilenceUsage:  true,
	SilenceErrors: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg := appConfig

		if cfg.Verify.Silent && cfg.Verify.JSON {
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
			if cfg.Verify.Silent {
				return errSilentFail
			}
			return err
		}

		hashFlag, _ := cmd.Flags().GetString("hash")
		if hashFlag != "" {
			hashFlag = strings.ToLower(strings.TrimSpace(hashFlag))
			if len(hashFlag)%2 != 0 {
				return fmt.Errorf("--hash must be even length hex string")
			}
			if _, err := hex.DecodeString(hashFlag); err != nil {
				return fmt.Errorf("--hash contains invalid hex characters: %w", err)
			}
		}

		// typeFlag is the value of --type and nothing else. Appendix E.24
		// requires a verifier to read the subject type from the bundle's
		// signed `t`, never from the downloaded filename: the beacon show
		// page legitimately names a t=10 block proof
		// `truestamp-beacon-<id>.json`, so deriving an assertion from the
		// stem failed sound proofs on the basis of a rename. The swap the
		// old inference claimed to catch is already caught anyway — `t` is
		// inside the signed payload (E.16), so relabelling a bundle breaks
		// its signature.
		typeFlag, _ := cmd.Flags().GetString("type")
		typeFlag = strings.ToLower(strings.TrimSpace(typeFlag))
		if typeFlag != "" && !validDownloadType(typeFlag) {
			return fmt.Errorf("--type must be one of %s, got %q",
				strings.Join(downloadTypeValues, " | "), typeFlag)
		}

		// displayName labels the source in the report and the log. It is a
		// presentation string only; nothing downstream derives semantics
		// from it.
		displayName := src.DisplayName()
		if src.Type == inputsrc.SourceStdin {
			displayName = "(stdin)"
		}

		var report *verify.Report

		if cfg.Verify.Remote {
			if cfg.Verify.SkipExternal || cfg.Verify.SkipSignatures {
				return fmt.Errorf("--skip-external and --skip-signatures cannot be used with --remote (server always runs full verification)")
			}
			if !authConfigured() {
				return fmt.Errorf("not authenticated — --remote verification needs `truestamp auth login`, or TRUESTAMP_API_KEY / --api-key for headless use")
			}

			// Remote verification needs a file on disk. For non-file
			// sources (stdin/url/picker that produced bytes) we write a
			// temp file; for regular file paths we pass through directly.
			tmpPath := ""
			sourceFile := displayName
			if src.Type != inputsrc.SourceFile {
				tmp, tErr := writeTempProof(data)
				if tErr != nil {
					return tErr
				}
				tmpPath = tmp
				sourceFile = tmp
				defer os.Remove(tmpPath)
			}

			report, err = verify.RunRemoteCtx(cmd.Context(), sourceFile, verify.RemoteOptions{
				APIURL:              cfg.APIURL,
				Team:                cfg.Team,
				ExpectedHash:        hashFlag,
				ExpectedSubjectType: typeFlag,
			})
		} else {
			opts := verify.Options{
				KeyringURL:          cfg.KeyringURL,
				APIURL:              cfg.APIURL,
				SkipExternal:        cfg.Verify.SkipExternal,
				SkipSignatures:      cfg.Verify.SkipSignatures,
				ExpectedHash:        hashFlag,
				ExpectedSubjectType: typeFlag,
			}
			report, err = verify.RunFromBytes(data, displayName, opts)
		}

		if err != nil {
			rejectionCode := proof.RejectionCode(err)
			appLogger.Error("verify_failed",
				"source", string(src.Type),
				"display_name", displayName,
				"remote", cfg.Verify.Remote,
				"rejection_code", rejectionCode,
				"err", err.Error(),
			)
			if cfg.Verify.Silent {
				return errSilentFail
			}
			// A hard rejection (E.6) aborts before any step runs, so there
			// is no Report to render. Under --json emit the E.23 identifier
			// as structured data rather than an English sentence: the whole
			// point of the taxonomy is that two independent verifiers can be
			// compared on the identifier without diffing prose.
			if rejectionCode != "" && cfg.Verify.JSON {
				return emitVerifyRejection(cmd, rejectionCode, err)
			}
			return err
		}

		appLogger.Info("verify_completed",
			"source", string(src.Type),
			"display_name", displayName,
			"remote", cfg.Verify.Remote,
			"subject_type", report.SubjectType,
			"passed", report.Passed(),
			"failed_step_count", report.FailedCount(),
		)

		switch {
		case cfg.Verify.JSON:
			jsonOutput := verify.BuildJSONOutput(report)
			jsonData, jErr := json.MarshalIndent(jsonOutput, "", "  ")
			if jErr != nil {
				return fmt.Errorf("marshaling JSON: %w", jErr)
			}
			fmt.Println(string(jsonData))
		case !cfg.Verify.Silent:
			verify.Present(report)
		}

		if !report.Passed() {
			if cfg.Verify.Silent {
				return errSilentFail
			}
			return errVerificationFailed
		}
		return nil
	},
}

// verifyRejectionJSON is the --json shape for an Appendix E.6 hard
// rejection. No Report exists on this path, so none of the step fields a
// successful run emits can be populated; `result` stays in the same
// position so a consumer can switch on it before reaching for anything
// else, and `rejection.code` carries the E.23 identifier.
type verifyRejectionJSON struct {
	Result    string          `json:"result"`
	Rejection verifyRejection `json:"rejection"`
}

type verifyRejection struct {
	Code   string `json:"code"`
	Detail string `json:"detail"`
}

// emitVerifyRejection writes the structured rejection to stdout and
// returns the sentinel that exits 1 without adding a second, English
// copy of the same failure on stderr.
//
// detail comes off the RejectionError itself rather than err.Error(), so
// it carries neither the caller's wrapping context nor a second copy of
// the code that already has its own field.
func emitVerifyRejection(cmd *cobra.Command, code string, err error) error {
	detail := err.Error()
	var re *proof.RejectionError
	if errors.As(err, &re) {
		detail = re.Detail
	}
	if jErr := emitJSON(cmd.OutOrStdout(), verifyRejectionJSON{
		Result:    "rejected",
		Rejection: verifyRejection{Code: code, Detail: detail},
	}); jErr != nil {
		return jErr
	}
	return errSilentFail
}

// writeTempProof writes proof data to a temporary file for remote verification.
func writeTempProof(data []byte) (string, error) {
	f, err := os.CreateTemp("", "truestamp-proof-*.json")
	if err != nil {
		return "", fmt.Errorf("creating temp file: %w", err)
	}
	if _, err := f.Write(data); err != nil {
		f.Close()
		os.Remove(f.Name())
		return "", fmt.Errorf("writing temp file: %w", err)
	}
	f.Close()
	return f.Name(), nil
}

func init() {
	f := verifyCmd.Flags()
	f.String("file", "", "Path to proof file (interactive picker if no path given)")
	f.String("url", "", "URL to download proof from (interactive prompt if no URL given)")
	f.Lookup("file").NoOptDefVal = inputsrc.FilePickSentinel
	f.Lookup("url").NoOptDefVal = inputsrc.URLPromptSentinel
	f.String("hash", "", "Expected claims hash (hex) to compare against proof")
	f.String("type", "",
		fmt.Sprintf("Assert expected subject type (guards against verifying the wrong file). One of: %s",
			strings.Join(downloadTypeValues, " | ")))
	f.BoolP("silent", "s", false, "No output, exit code only")
	f.Bool("json", false, "Output results as JSON")
	f.Bool("skip-external", false, "Skip all external API verification")
	// Appendix E.9's Signing Key row (does `pk` decode to 32 bytes, and
	// what key id does it derive to) runs unconditionally: it is local,
	// costs nothing, and its output is the kid E.16 would have fed to the
	// payload. What this flag suppresses is E.16's Ed25519 check and
	// E.17's keyring cross-check. The old wording ("Skip signing key and
	// signature verification") claimed the Signing Key row was skipped
	// while the report visibly passed it.
	f.Bool("skip-signatures", false, "Skip proof signature and keyring verification")
	f.Bool("remote", false, "Verify via server API instead of local computation (requires authentication)")
	rootCmd.AddCommand(verifyCmd)
}
