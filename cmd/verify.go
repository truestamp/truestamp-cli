// Copyright (c) 2021-2026 Truestamp, Inc.
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

Pass --type to assert the expected subject type. If the bundle's t
doesn't match, the report surfaces a Subject Type failure (local
mode) or the server rejects the request with subject_type_mismatch
(--remote mode). Values: item | entropy_nist | entropy_stellar |
entropy_bitcoin | block | beacon. Useful as a guard against verifying
the wrong file — e.g. a beacon-named file that was swapped with a
plain block proof (both verify on their own, but only --type beacon
catches the swap).

Use --remote to delegate verification to the Truestamp server API instead
of performing local computation. Requires --api-key to be set.

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

		typeFlag, _ := cmd.Flags().GetString("type")
		typeFlag = strings.ToLower(strings.TrimSpace(typeFlag))
		if typeFlag != "" && !validDownloadType(typeFlag) {
			return fmt.Errorf("--type must be one of %s, got %q",
				strings.Join(downloadTypeValues, " | "), typeFlag)
		}

		displayName := src.DisplayName()
		if src.Type == inputsrc.SourceStdin {
			displayName = "(stdin)"
		}

		var report *verify.Report

		if cfg.Verify.Remote {
			if cfg.Verify.SkipExternal || cfg.Verify.SkipSignatures {
				return fmt.Errorf("--skip-external and --skip-signatures cannot be used with --remote (server always runs full verification)")
			}
			if cfg.APIKey == "" {
				return fmt.Errorf("API key required for --remote verification (use --api-key or set TRUESTAMP_API_KEY)")
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

			report, err = verify.RunRemote(sourceFile, verify.RemoteOptions{
				APIURL:              cfg.APIURL,
				APIKey:              cfg.APIKey,
				Team:                cfg.Team,
				ExpectedHash:        hashFlag,
				ExpectedSubjectType: typeFlag,
			})
		} else {
			opts := verify.Options{
				KeyringURL:          cfg.KeyringURL,
				SkipExternal:        cfg.Verify.SkipExternal,
				SkipSignatures:      cfg.Verify.SkipSignatures,
				ExpectedHash:        hashFlag,
				ExpectedSubjectType: typeFlag,
			}
			report, err = verify.RunFromBytes(data, displayName, opts)
		}

		if err != nil {
			if cfg.Verify.Silent {
				return errSilentFail
			}
			return err
		}

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
	f.Bool("skip-signatures", false, "Skip signing key and signature verification")
	f.Bool("remote", false, "Verify via server API instead of local computation (requires --api-key)")
	rootCmd.AddCommand(verifyCmd)
}
