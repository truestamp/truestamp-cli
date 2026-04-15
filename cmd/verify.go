// Copyright (c) 2021-2026 Truestamp, Inc.
// SPDX-License-Identifier: MIT

package cmd

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/url"
	"os"
	"strings"

	"charm.land/huh/v2"
	"github.com/spf13/cobra"
	"github.com/truestamp/truestamp-cli/internal/proof"
	"github.com/truestamp/truestamp-cli/internal/ui"
	"github.com/truestamp/truestamp-cli/internal/verify"
)

// Sentinel values for flags passed without a value (via NoOptDefVal).
const (
	fileFlagPick  = ":pick"
	urlFlagPrompt = ":prompt"
)

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

Use --remote to delegate verification to the Truestamp server API instead
of performing local computation. Requires --api-key to be set.

Exit code 0 on success, 1 on verification failure.`,
	Args:          cobra.MaximumNArgs(1),
	SilenceUsage:  true,
	SilenceErrors: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg := appConfig

		// Validate mutual exclusivity of --silent and --json
		if cfg.Verify.Silent && cfg.Verify.JSON {
			return fmt.Errorf("--silent and --json are mutually exclusive")
		}

		filename, data, err := resolveProofInput(cmd, args)
		if err != nil {
			return err
		}

		// Empty filename means cmd.Help() was shown
		if filename == "" && data == nil {
			return nil
		}

		// Validate --hash flag
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

		var report *verify.Report

		if cfg.Verify.Remote {
			if cfg.Verify.SkipExternal || cfg.Verify.SkipSignatures {
				cmd.PrintErrln("Error: --skip-external and --skip-signatures cannot be used with --remote (server always runs full verification)")
				os.Exit(1)
			}

			if cfg.APIKey == "" {
				cmd.PrintErrln("Error: API key required for --remote verification (use --api-key or set TRUESTAMP_API_KEY)")
				os.Exit(1)
			}

			if data != nil {
				// Remote verification needs a file on disk -- write temp
				tmp, tErr := writeTempProof(data)
				if tErr != nil {
					return tErr
				}
				defer os.Remove(tmp)
				filename = tmp
			}

			report, err = verify.RunRemote(filename, verify.RemoteOptions{
				APIURL:       cfg.APIURL,
				APIKey:       cfg.APIKey,
				Team:         cfg.Team,
				ExpectedHash: hashFlag,
			})
		} else {
			opts := verify.Options{
				KeyringURL:     cfg.KeyringURL,
				SkipExternal:   cfg.Verify.SkipExternal,
				SkipSignatures: cfg.Verify.SkipSignatures,
				ExpectedHash:   hashFlag,
			}
			if data != nil {
				report, err = verify.RunFromBytes(data, filename, opts)
			} else {
				report, err = verify.Run(filename, opts)
			}
		}

		if err != nil {
			if !cfg.Verify.Silent {
				cmd.PrintErrln("Error:", err)
			}
			os.Exit(1)
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
			os.Exit(1)
		}
		return nil
	},
}

// resolveProofInput determines where the proof comes from based on flags and args.
// Returns (filename, data, error). If data is non-nil, the proof was downloaded
// from a URL and filename is the display name. If data is nil, filename is a
// local file path to read.
func resolveProofInput(cmd *cobra.Command, args []string) (string, []byte, error) {
	fileFlag, _ := cmd.Flags().GetString("file")
	urlFlag, _ := cmd.Flags().GetString("url")

	switch {
	case fileFlag == fileFlagPick:
		if len(args) > 0 {
			return args[0], nil, nil
		}
		path, err := pickFile()
		if err != nil {
			return "", nil, err
		}
		return path, nil, nil

	case fileFlag != "":
		return fileFlag, nil, nil

	case urlFlag == urlFlagPrompt:
		if len(args) > 0 {
			data, err := proof.Download(args[0])
			if err != nil {
				return "", nil, err
			}
			return args[0], data, nil
		}
		rawURL, err := promptURL()
		if err != nil {
			return "", nil, err
		}
		data, err := proof.Download(rawURL)
		if err != nil {
			return "", nil, err
		}
		return rawURL, data, nil

	case urlFlag != "":
		data, err := proof.Download(urlFlag)
		if err != nil {
			return "", nil, err
		}
		return urlFlag, data, nil

	case len(args) > 0:
		if u, err := url.Parse(args[0]); err == nil && (u.Scheme == "http" || u.Scheme == "https") && u.Host != "" {
			data, err := proof.Download(args[0])
			if err != nil {
				return "", nil, err
			}
			return args[0], data, nil
		}
		return args[0], nil, nil

	case isStdinPipe():
		data, err := io.ReadAll(io.LimitReader(os.Stdin, 1<<20))
		if err != nil {
			return "", nil, fmt.Errorf("reading stdin: %w", err)
		}
		if len(data) == 0 {
			return "", nil, fmt.Errorf("no data received on stdin")
		}
		return "(stdin)", data, nil

	default:
		cmd.Help()
		return "", nil, nil
	}
}

// isStdinPipe returns true if stdin is a pipe (not a terminal).
func isStdinPipe() bool {
	stat, err := os.Stdin.Stat()
	if err != nil {
		return false
	}
	return (stat.Mode() & os.ModeCharDevice) == 0
}

// pickFile launches an interactive file picker for selecting a proof file.
func pickFile() (string, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("getting working directory: %w", err)
	}

	var path string
	err = huh.NewForm(
		huh.NewGroup(
			huh.NewFilePicker().
				Title("Select proof file").
				Description("up/down move  right/enter open  left/backspace back  enter select").
				CurrentDirectory(cwd).
				AllowedTypes([]string{".json", ".cbor"}).
				FileAllowed(true).
				DirAllowed(false).
				ShowSize(true).
				ShowPermissions(false).
				Picking(true).
				Height(20).
				Value(&path),
		),
	).WithTheme(ui.HuhTheme()).Run()
	if err != nil {
		return "", fmt.Errorf("file selection: %w", err)
	}
	if path == "" {
		return "", fmt.Errorf("no file selected")
	}
	return path, nil
}

// promptURL launches an interactive text input for entering a proof URL.
func promptURL() (string, error) {
	var rawURL string
	err := huh.NewForm(
		huh.NewGroup(
			huh.NewInput().
				Title("Enter proof URL").
				Placeholder("https://example.com/proof.json").
				Value(&rawURL).
				Validate(func(s string) error {
					if s == "" {
						return nil
					}
					if !strings.HasPrefix(s, "http://") && !strings.HasPrefix(s, "https://") {
						return fmt.Errorf("must start with http:// or https://")
					}
					return nil
				}),
		),
	).WithTheme(ui.HuhTheme()).Run()
	if err != nil {
		return "", fmt.Errorf("URL input: %w", err)
	}
	if rawURL == "" {
		return "", fmt.Errorf("no URL provided")
	}
	return rawURL, nil
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
	f.Lookup("file").NoOptDefVal = fileFlagPick
	f.Lookup("url").NoOptDefVal = urlFlagPrompt
	f.String("hash", "", "Expected claims hash (hex) to compare against proof")
	f.BoolP("silent", "s", false, "No output, exit code only")
	f.Bool("json", false, "Output results as JSON")
	f.Bool("skip-external", false, "Skip all external API verification")
	f.Bool("skip-signatures", false, "Skip signing key and signature verification")
	f.Bool("remote", false, "Verify via server API instead of local computation (requires --api-key)")
	rootCmd.AddCommand(verifyCmd)
}
