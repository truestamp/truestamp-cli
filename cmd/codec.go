// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"errors"
	"fmt"

	"github.com/spf13/cobra"
	"github.com/truestamp/truestamp-cli/internal/encoding"
	"github.com/truestamp/truestamp-cli/internal/inputsrc"
	"github.com/truestamp/truestamp-cli/internal/jcs"
	"github.com/truestamp/truestamp-cli/internal/ui"
)

// codecJSON is the --json shape for encode / decode.
type codecJSON struct {
	From        string `json:"from"`
	To          string `json:"to"`
	InputBytes  int64  `json:"input_bytes"`
	OutputBytes int    `json:"output_bytes"`
	// Output carries the result. For textual To encodings it is the
	// text directly; for Binary To it is wrapped as base64url so the
	// JSON remains valid and callers can round-trip via the same pair.
	Output        string `json:"output"`
	OutputWrapped string `json:"output_wrapped,omitempty"`
}

// resolveCodecInput runs the shared six-mode resolver for encode/decode/jcs.
// Stdin is always allowed (these commands are pipeline primitives).
func resolveCodecInput(cmd *cobra.Command, args []string, pickerTitle string) ([]byte, inputsrc.Source, error) {
	positional := ""
	if len(args) > 0 {
		positional = args[0]
	}
	fileFlag, _ := cmd.Flags().GetString("file")
	urlFlag, _ := cmd.Flags().GetString("url")
	return inputsrc.Resolve(cmd.Context(), inputsrc.Options{
		PositionalArg:        positional,
		FileFlag:             fileFlag,
		URLFlag:              urlFlag,
		AllowStdin:           true,
		AutoDetectURL:        true,
		PickerTitle:          pickerTitle,
		URLPromptTitle:       "Enter URL",
		URLPromptPlaceholder: "https://example.com/...",
	})
}

// writeOutput writes bytes to stdout, respecting --silent (no output at
// all). For Binary output to a real TTY we emit a stderr warning so
// users don't accidentally dump raw bytes into their terminal.
func writeOutput(cmd *cobra.Command, silent bool, enc encoding.Encoding, out []byte) {
	if silent {
		return
	}
	if enc == encoding.Binary && inputsrc.IsStdoutTerminal() {
		ui.Fprintln(cmd.ErrOrStderr(),
			"warning: writing binary bytes to a terminal; redirect to a file or pipe")
	}
	_, _ = cmd.OutOrStdout().Write(out)
}

// ---------------- encode ----------------

var encodeCmd = &cobra.Command{
	Use:   "encode [flags] [file]",
	Args:  cobra.MaximumNArgs(1),
	Short: "Encode raw bytes into hex, base64, or base64url",
	Long: `Encode raw bytes (the default input) into a textual representation.

Defaults: --from binary, --to hex. Pipe-friendly so 'truestamp hash' and
other commands can feed into it.

Examples:
  cat image.png | truestamp encode --to base64
  truestamp encode --from hex --to base64 <<< 68656c6c6f
  truestamp encode --file doc.pdf --to base64url`,
	RunE: runEncode,
}

func runEncode(cmd *cobra.Command, args []string) error {
	return runCodec(cmd, args, codecSpec{
		DefaultFrom: encoding.Binary,
		DefaultTo:   encoding.Hex,
		PickerTitle: "Select file to encode",
	})
}

// ---------------- decode ----------------

var decodeCmd = &cobra.Command{
	Use:   "decode [flags] [file]",
	Args:  cobra.MaximumNArgs(1),
	Short: "Decode hex / base64 / base64url into raw bytes",
	Long: `Decode a textual representation back into raw bytes.

Defaults: --from hex, --to binary. Accepts either padded or unpadded
base64url inputs; tolerates a trailing newline.

Examples:
  echo Zm9vYmFy | truestamp decode --from base64 > fooBar.bin
  truestamp decode --from hex --to base64 <<< 68656c6c6f
  truestamp decode --from base64url --file token.txt > token.bin`,
	RunE: runDecode,
}

func runDecode(cmd *cobra.Command, args []string) error {
	return runCodec(cmd, args, codecSpec{
		DefaultFrom: encoding.Hex,
		DefaultTo:   encoding.Binary,
		PickerTitle: "Select file to decode",
	})
}

// codecSpec is the per-command defaults table consumed by the shared runner.
type codecSpec struct {
	DefaultFrom encoding.Encoding
	DefaultTo   encoding.Encoding
	PickerTitle string
}

func runCodec(cmd *cobra.Command, args []string, spec codecSpec) error {
	fromName, _ := cmd.Flags().GetString("from")
	toName, _ := cmd.Flags().GetString("to")
	jsonOut, _ := cmd.Flags().GetBool("json")
	silent, _ := cmd.Flags().GetBool("silent")

	from := spec.DefaultFrom
	if fromName != "" {
		p, err := encoding.Parse(fromName)
		if err != nil {
			return fmt.Errorf("--from: %w", err)
		}
		from = p
	}
	to := spec.DefaultTo
	if toName != "" {
		p, err := encoding.Parse(toName)
		if err != nil {
			return fmt.Errorf("--to: %w", err)
		}
		to = p
	}

	data, src, err := resolveCodecInput(cmd, args, spec.PickerTitle)
	if err != nil {
		if errors.Is(err, inputsrc.ErrNoInput) {
			_ = cmd.Help()
			return nil
		}
		return err
	}

	// Transform: decode (if From is text) then encode to the target.
	// Binary→Binary is a pass-through; Text→Text decodes and re-encodes.
	var raw []byte
	if from == encoding.Binary {
		raw = data
	} else {
		raw, err = encoding.Decode(from, data)
		if err != nil {
			return fmt.Errorf("input as %s: %w", from.Name(), err)
		}
	}
	out, err := encoding.Encode(to, raw)
	if err != nil {
		return fmt.Errorf("encoding as %s: %w", to.Name(), err)
	}

	if jsonOut {
		payload := codecJSON{
			From:        from.Name(),
			To:          to.Name(),
			InputBytes:  src.Size,
			OutputBytes: len(out),
		}
		if to == encoding.Binary {
			wrapped, _ := encoding.Encode(encoding.Base64URL, out)
			payload.OutputWrapped = string(wrapped)
		} else {
			payload.Output = string(out)
		}
		return emitJSON(cmd.OutOrStdout(), payload)
	}

	writeOutput(cmd, silent, to, out)
	// Add a trailing newline for textual outputs (mirrors `base64`, `xxd -p`).
	// Binary output is passed through untouched.
	if !silent && to != encoding.Binary && (len(out) == 0 || out[len(out)-1] != '\n') {
		ui.Fprintln(cmd.OutOrStdout())
	}
	return nil
}

// ---------------- jcs ----------------

var jcsCmd = &cobra.Command{
	Use:   "jcs [flags] [file]",
	Args:  cobra.MaximumNArgs(1),
	Short: "Canonicalize JSON per RFC 8785 (JSON Canonicalization Scheme)",
	Long: `Apply RFC 8785 JSON Canonicalization to the input. The output is a
byte-stable form suitable for hashing and signing, pipes directly into
'truestamp hash' (or use 'truestamp hash --jcs' as a shortcut).

The input is read as a Truestamp verifier reads a proof (SPEC.md section
10.1): a repeated member name, invalid UTF-8, a lone surrogate or nesting
deeper than 32 is refused, and so is an integer of magnitude above
2^53 - 1 and below 10^21, however it is written (9007199254740992,
9007199254740992.0 or 1e16): send such a value as a string. Every other
number is written as its nearest IEEE-754 double is, as RFC 8785 says.

Examples:
  truestamp jcs < claims.json | truestamp hash --prefix 0x10 -a sha256
  truestamp hash --prefix 0x10 --jcs < claims.json   # equivalent one-liner`,
	RunE: runJCS,
}

func runJCS(cmd *cobra.Command, args []string) error {
	jsonOut, _ := cmd.Flags().GetBool("json")
	silent, _ := cmd.Flags().GetBool("silent")
	newline, _ := cmd.Flags().GetBool("newline")

	data, src, err := resolveCodecInput(cmd, args, "Select JSON file")
	if err != nil {
		if errors.Is(err, inputsrc.ErrNoInput) {
			_ = cmd.Help()
			return nil
		}
		return err
	}

	canonical, err := jcs.Transform(data)
	if err != nil {
		return fmt.Errorf("JCS canonicalization: %w", err)
	}

	if jsonOut {
		return emitJSON(cmd.OutOrStdout(), struct {
			InputBytes  int64  `json:"input_bytes"`
			OutputBytes int    `json:"output_bytes"`
			Output      string `json:"output"`
		}{
			InputBytes:  src.Size,
			OutputBytes: len(canonical),
			Output:      string(canonical),
		})
	}

	if silent {
		return nil
	}
	_, _ = cmd.OutOrStdout().Write(canonical)
	if newline {
		_, _ = cmd.OutOrStdout().Write([]byte("\n"))
	}
	return nil
}

func init() {
	// Shared input flags for all three primitives.
	for _, c := range []*cobra.Command{encodeCmd, decodeCmd, jcsCmd} {
		f := c.Flags()
		f.String("file", "", "Path to input file (interactive picker if no path given)")
		f.String("url", "", "URL to download (interactive prompt if no URL given)")
		f.Lookup("file").NoOptDefVal = inputsrc.FilePickSentinel
		f.Lookup("url").NoOptDefVal = inputsrc.URLPromptSentinel
		f.Bool("json", false, "Output as JSON")
		f.BoolP("silent", "s", false, "No output, exit code only")
	}

	// encode / decode-specific flags
	for _, c := range []*cobra.Command{encodeCmd, decodeCmd} {
		c.Flags().String("from", "", "Input encoding: hex, base64, base64url, binary")
		c.Flags().String("to", "", "Output encoding: hex, base64, base64url, binary")
	}

	// jcs-specific
	jcsCmd.Flags().Bool("newline", false, "Append a trailing newline to the output")

	encodeCmd.GroupID = groupTools
	rootCmd.AddCommand(encodeCmd)
	decodeCmd.GroupID = groupTools
	rootCmd.AddCommand(decodeCmd)
	jcsCmd.GroupID = groupTools
	rootCmd.AddCommand(jcsCmd)
}
