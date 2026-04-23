// Copyright (c) 2021-2026 Truestamp, Inc.
// SPDX-License-Identifier: MIT

package cmd

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"github.com/truestamp/truestamp-cli/internal/encoding"
	"github.com/truestamp/truestamp-cli/internal/inputsrc"
	"github.com/truestamp/truestamp-cli/internal/proof"
)

var convertProofCmd = &cobra.Command{
	Use:   "proof [flags] [file]",
	Short: "Convert a Truestamp proof bundle between JSON and CBOR wire formats",
	Long: `Read a proof bundle in one wire format and emit it in the other.

The CBOR output uses RFC 8949 §4.2 core deterministic encoding
(lexicographically sorted keys, shortest-form integers, definite-length
containers) and is prefixed with the self-describing CBOR tag 55799 so
'truestamp verify' auto-detects the format.

Examples:
  truestamp convert proof --to cbor proof.json > proof.cbor
  truestamp convert proof --to json < proof.cbor | jq .
  truestamp convert proof --from cbor --to json proof.cbor`,
	SilenceUsage:  true,
	SilenceErrors: true,
	RunE:          runConvertProof,
}

func runConvertProof(cmd *cobra.Command, args []string) error {
	toName, _ := cmd.Flags().GetString("to")
	fromName, _ := cmd.Flags().GetString("from")
	compact, _ := cmd.Flags().GetBool("compact")
	jsonOut, _ := cmd.Flags().GetBool("json")
	silent, _ := cmd.Flags().GetBool("silent")

	if silent && jsonOut {
		return fmt.Errorf("--silent and --json are mutually exclusive")
	}

	to := strings.ToLower(strings.TrimSpace(toName))
	if to == "" {
		return fmt.Errorf("--to is required (json or cbor)")
	}
	if to != "json" && to != "cbor" {
		return fmt.Errorf("--to must be json or cbor (got %q)", to)
	}

	data, src, err := resolveCodecInput(cmd, args, "Select proof file")
	if err != nil {
		if errors.Is(err, inputsrc.ErrNoInput) {
			_ = cmd.Help()
			return nil
		}
		return err
	}

	from := strings.ToLower(strings.TrimSpace(fromName))
	var bundle *proof.ProofBundle
	actualFrom := from
	switch from {
	case "", "auto":
		bundle, err = proof.ParseBytes(data)
		if err != nil {
			return fmt.Errorf("parsing proof: %w", err)
		}
		if proof.IsCBORProof(data) || !looksLikeJSON(data) {
			actualFrom = "cbor"
		} else {
			actualFrom = "json"
		}
	case "json":
		bundle, err = parseJSON(data)
		if err != nil {
			return fmt.Errorf("parsing JSON proof: %w", err)
		}
	case "cbor":
		bundle, err = proof.ParseCBOR(data)
		if err != nil {
			return fmt.Errorf("parsing CBOR proof: %w", err)
		}
	default:
		return fmt.Errorf("--from must be auto, json, or cbor (got %q)", from)
	}

	var out []byte
	switch to {
	case "json":
		raw, mErr := bundle.MarshalJSON()
		if mErr != nil {
			return fmt.Errorf("marshaling JSON: %w", mErr)
		}
		if compact {
			out = raw
		} else {
			pretty, pErr := prettyJSON(raw)
			if pErr != nil {
				return pErr
			}
			out = pretty
		}
	case "cbor":
		out, err = bundle.MarshalCBOR()
		if err != nil {
			return fmt.Errorf("marshaling CBOR: %w", err)
		}
	}

	if jsonOut {
		// Wrap the output in a JSON envelope for scripting. Binary CBOR
		// is base64url-wrapped so the envelope stays valid JSON.
		outWrapped := ""
		outText := ""
		if to == "cbor" {
			wrapped := base64URLEncode(out)
			outWrapped = wrapped
		} else {
			outText = string(out)
		}
		return emitJSON(cmd.OutOrStdout(), struct {
			InputFormat   string `json:"input_format"`
			OutputFormat  string `json:"output_format"`
			InputBytes    int64  `json:"input_bytes"`
			OutputBytes   int    `json:"output_bytes"`
			CanonicalCBOR bool   `json:"canonical_cbor"`
			Output        string `json:"output,omitempty"`
			OutputWrapped string `json:"output_wrapped,omitempty"`
		}{
			InputFormat:   actualFrom,
			OutputFormat:  to,
			InputBytes:    src.Size,
			OutputBytes:   len(out),
			CanonicalCBOR: to == "cbor",
			Output:        outText,
			OutputWrapped: outWrapped,
		})
	}

	if silent {
		return nil
	}
	_, _ = cmd.OutOrStdout().Write(out)
	if to == "json" && (len(out) == 0 || out[len(out)-1] != '\n') {
		_, _ = cmd.OutOrStdout().Write([]byte("\n"))
	}
	return nil
}

// parseJSON is a thin wrapper that rejects inputs that don't actually
// start like JSON — useful when --from json is explicit and a user
// piped CBOR by mistake.
func parseJSON(data []byte) (*proof.ProofBundle, error) {
	if !looksLikeJSON(data) {
		return nil, fmt.Errorf("input does not start with '{' (try --from cbor or --from auto)")
	}
	return proof.ParseBytes(data)
}

// looksLikeJSON returns true when the first non-whitespace byte is '{'.
func looksLikeJSON(data []byte) bool {
	trimmed := bytes.TrimLeft(data, " \t\r\n")
	return len(trimmed) > 0 && trimmed[0] == '{'
}

// prettyJSON re-renders a JSON object with two-space indent. Preserves
// key order as emitted by the Go json encoder (alphabetical).
func prettyJSON(raw []byte) ([]byte, error) {
	var v any
	if err := json.Unmarshal(raw, &v); err != nil {
		return nil, fmt.Errorf("prettyJSON: %w", err)
	}
	return json.MarshalIndent(v, "", "  ")
}

// base64URLEncode wraps encoding.Encode for the binary-CBOR-inside-JSON
// case, where we need a textual representation to keep the JSON envelope
// valid.
func base64URLEncode(data []byte) string {
	out, _ := encoding.Encode(encoding.Base64URL, data)
	return string(out)
}

func init() {
	f := convertProofCmd.Flags()
	f.String("file", "", "Path to proof file (interactive picker if no path given)")
	f.String("url", "", "URL to download proof from (interactive prompt if no URL given)")
	f.Lookup("file").NoOptDefVal = inputsrc.FilePickSentinel
	f.Lookup("url").NoOptDefVal = inputsrc.URLPromptSentinel
	f.String("from", "auto", "Input format: auto, json, or cbor")
	f.String("to", "", "Output format: json or cbor (required)")
	f.Bool("compact", false, "JSON output: emit minified form (default: 2-space indent)")
	// convert proof's --json means "JSON envelope"; the help text above
	// stays specific to that, so we register it explicitly instead of
	// going through addConvertCommonFlags. --silent shares the family
	// default.
	f.Bool("json", false, "Emit a JSON envelope with input/output metadata instead of raw output")
	f.BoolP("silent", "s", false, "No output, exit code only")
	convertCmd.AddCommand(convertProofCmd)
}
