// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: MIT

package cmd

import (
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	lipgloss "charm.land/lipgloss/v2"
	"github.com/spf13/cobra"
	"github.com/truestamp/truestamp-cli/internal/inputsrc"
	"github.com/truestamp/truestamp-cli/internal/proof"
	"github.com/truestamp/truestamp-cli/internal/verify"
)

// errVerificationFailed is returned when the report itself fails. The
// report has already been rendered, so only a non-nil error for exit code 1
// is needed. Execute() prints it via its default path.
var errVerificationFailed = errors.New("verification failed")

var verifyCmd = &cobra.Command{
	Use:   "verify [file-or-url]",
	Short: "Verify a Truestamp proof bundle",
	Long: `Cryptographically verify a Truestamp proof bundle, JSON or CBOR.

Every value in the bundle is recomputed from bytes it carries and checked
against the signature over them (whitepaper Appendix E): the subject and
block hashes, the Merkle inclusion proof, each epoch proof to the root
committed on a public chain, the Ed25519 proof signature, the witnesses
that open the submitted-after edge, the signing key event, and the
submission window. Online runs additionally confirm the Stellar and
Bitcoin commitments on chain, re-fetch each entropy witness from its
source, and bind the signing key to the published keyring.

Proof input can be provided as:
  truestamp verify proof.json           Local file path
  truestamp verify proof.cbor           CBOR is detected automatically
  truestamp verify https://host/p.json  URL (auto-detected)
  truestamp verify --file proof.json    Explicit file path
  truestamp verify --file               Interactive file picker
  truestamp verify --url https://...    Explicit URL download
  truestamp verify --url                Interactive URL prompt
  cat proof.json | truestamp verify     Pipe from stdin

Offline verification is first class: --offline (or --skip-external) runs
every cryptographic step with no network access and reports each check
that needs a source as skipped, never failed. A skipped check is a check
this run did not perform, not a check that failed.

--expected-hash compares the hash of a file you hold against the hash an
item's claims commit to. --keyring pins a local copy of
/.well-known/keyring.json for the key binding check; without it an online
run fetches the live keyring and an offline run reports the binding as
not checked.

The subject type is always read from the bundle's own signed 'type'
field; the filename is never consulted. Pass --type to assert the type
you expected: a mismatch is the hard rejection subject_type_mismatch.
Values: item | entropy_nist | entropy_stellar | entropy_bitcoin |
block | beacon.

A bundle in the pre-publication draft layout (top-level 'v' or 't' keys)
is refused with unsupported_layout; ask the holder to regenerate it.

Use --remote to also ask the Truestamp server to verify the bundle. This
CLI's own verifier never depends on it. Requires authentication: run
'truestamp auth login', or set TRUESTAMP_API_KEY / --api-key.

Exit code 0 when the proof passes, 1 when it fails or is rejected.`,
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

		expectedHash, err := expectedHashFlag(cmd)
		if err != nil {
			return err
		}

		typeFlag, _ := cmd.Flags().GetString("type")
		typeFlag = strings.ToLower(strings.TrimSpace(typeFlag))
		if typeFlag != "" && !validDownloadType(typeFlag) {
			return fmt.Errorf("--type must be one of %s, got %q",
				strings.Join(downloadTypeValues, " | "), typeFlag)
		}

		keyringFile := cfg.Verify.Keyring
		if v, _ := cmd.Flags().GetString("keyring"); v != "" {
			keyringFile = v
		}

		displayName := src.DisplayName()
		if src.Type == inputsrc.SourceStdin {
			displayName = "(stdin)"
		}

		var report *verify.Report
		if cfg.Verify.Remote {
			if cfg.Verify.SkipSignatures {
				return fmt.Errorf("--skip-signatures cannot be used with --remote (the server always checks the signature)")
			}
			if !authConfigured() {
				return fmt.Errorf("not authenticated: --remote verification needs `truestamp auth login`, or TRUESTAMP_API_KEY / --api-key for headless use")
			}
			report, err = verify.RunRemoteBytesCtx(cmd.Context(), data, displayName, verify.RemoteOptions{
				APIURL:              cfg.APIURL,
				Team:                cfg.Team,
				ExpectedHash:        expectedHash,
				SkipExternal:        cfg.Verify.SkipExternal,
				ExpectedSubjectType: typeFlag,
			})
		} else {
			report, err = verify.RunFromBytes(data, displayName, verify.Options{
				ExpectedHash:        expectedHash,
				ExpectedSubjectType: typeFlag,
				SkipExternal:        cfg.Verify.SkipExternal,
				SkipSignatures:      cfg.Verify.SkipSignatures,
				KeyringFile:         keyringFile,
				KeyringURL:          cfg.KeyringURL,
			})
		}

		if err != nil {
			rejectionCode := proof.RejectionCode(err)
			var remoteRejection *verify.RemoteRejectionError
			if errors.As(err, &remoteRejection) {
				rejectionCode = remoteRejection.Reason
				err = proof.Rejectf(remoteRejection.Reason, "%s", remoteRejection.Detail)
			}
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
			// A hard rejection (Appendix E.6) aborts before any step
			// runs, so there is no report to render: only the E.23
			// identifier and one line of advice.
			if rejectionCode != "" {
				if cfg.Verify.JSON {
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
			out, jErr := json.MarshalIndent(verify.BuildJSONReport(report), "", "  ")
			if jErr != nil {
				return fmt.Errorf("marshaling JSON: %w", jErr)
			}
			fmt.Fprintln(cmd.OutOrStdout(), string(out))
		case !cfg.Verify.Silent:
			lipgloss.Print(verify.Render(report, true))
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

// expectedHashFlag reads --expected-hash (or its older spelling --hash),
// normalizing it the way Appendix E.7 requires: trimmed and lowercased.
func expectedHashFlag(cmd *cobra.Command) (string, error) {
	value, _ := cmd.Flags().GetString("expected-hash")
	if value == "" {
		value, _ = cmd.Flags().GetString("hash")
	}
	value = strings.ToLower(strings.TrimSpace(value))
	if value == "" {
		return "", nil
	}
	if len(value)%2 != 0 {
		return "", fmt.Errorf("--expected-hash must be an even-length hex string")
	}
	if _, err := hex.DecodeString(value); err != nil {
		return "", fmt.Errorf("--expected-hash contains invalid hex characters: %w", err)
	}
	return value, nil
}

func init() {
	f := verifyCmd.Flags()
	f.String("file", "", "Path to proof file (interactive picker if no path given)")
	f.String("url", "", "URL to download proof from (interactive prompt if no URL given)")
	f.Lookup("file").NoOptDefVal = inputsrc.FilePickSentinel
	f.Lookup("url").NoOptDefVal = inputsrc.URLPromptSentinel
	f.String("expected-hash", "", "Hash (hex) of the file you hold, compared against the item's claims.hash")
	f.String("hash", "", "Alias of --expected-hash")
	_ = f.MarkHidden("hash")
	f.String("keyring", "", "Path of a pinned copy of /.well-known/keyring.json for the key binding check")
	f.String("type", "",
		fmt.Sprintf("Assert the expected subject type; a mismatch is rejected. One of: %s",
			strings.Join(downloadTypeValues, " | ")))
	f.BoolP("silent", "s", false, "No output, exit code only")
	f.Bool("json", false, "Output the report as JSON, in the same field names the Truestamp API uses")
	f.Bool("offline", false, "Run with no network access: chain, source and keyring lookups are reported as skipped")
	f.Bool("skip-external", false, "Alias of --offline")
	f.Bool("skip-signatures", false, "Skip the Ed25519 proof signature and key binding checks (disclosed in the report)")
	f.Bool("remote", false, "Also ask the Truestamp server to verify the bundle (requires authentication)")
	rootCmd.AddCommand(verifyCmd)
}
