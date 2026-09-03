// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: MIT

package cmd

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"

	"github.com/spf13/cobra"
	"github.com/truestamp/truestamp-cli/internal/encoding"
	"github.com/truestamp/truestamp-cli/internal/inputsrc"
	"github.com/truestamp/truestamp-cli/internal/jcs"
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
	if enc == encoding.Binary && stdoutIsTerminal() {
		fmt.Fprintln(cmd.ErrOrStderr(),
			"warning: writing binary bytes to a terminal; redirect to a file or pipe")
	}
	_, _ = cmd.OutOrStdout().Write(out)
}

func stdoutIsTerminal() bool {
	stat, err := os.Stdout.Stat()
	if err != nil {
		return false
	}
	return (stat.Mode() & os.ModeCharDevice) != 0
}

// ---------------- encode ----------------

var encodeCmd = &cobra.Command{
	Use:   "encode [flags] [file]",
	Short: "Encode raw bytes into hex, base64, or base64url",
	Long: `Encode raw bytes (the default input) into a textual representation.

Defaults: --from binary, --to hex. Pipe-friendly so 'truestamp hash' and
other commands can feed into it.

Examples:
  cat image.png | truestamp encode --to base64
  truestamp encode --from hex --to base64 <<< 68656c6c6f
  truestamp encode --file doc.pdf --to base64url`,
	SilenceUsage:  true,
	SilenceErrors: true,
	RunE:          runEncode,
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
	Short: "Decode hex / base64 / base64url into raw bytes",
	Long: `Decode a textual representation back into raw bytes.

Defaults: --from hex, --to binary. Accepts either padded or unpadded
base64url inputs; tolerates a trailing newline.

Examples:
  echo Zm9vYmFy | truestamp decode --from base64 > fooBar.bin
  truestamp decode --from hex --to base64 <<< 68656c6c6f
  truestamp decode --from base64url --file token.txt > token.bin`,
	SilenceUsage:  true,
	SilenceErrors: true,
	RunE:          runDecode,
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

	if silent && jsonOut {
		return fmt.Errorf("--silent and --json are mutually exclusive")
	}

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
		fmt.Fprintln(cmd.OutOrStdout())
	}
	return nil
}

func emitJSON(w io.Writer, v any) error {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling JSON: %w", err)
	}
	fmt.Fprintln(w, string(data))
	return nil
}

// ---------------- jcs ----------------

var jcsCmd = &cobra.Command{
	Use:   "jcs [flags] [file]",
	Short: "Canonicalize JSON per RFC 8785 (JSON Canonicalization Scheme)",
	Long: `Apply RFC 8785 JSON Canonicalization to the input. The output is a
byte-stable form suitable for hashing and signing, pipes directly into
'truestamp hash' (or use 'truestamp hash --jcs' as a shortcut).

One deviation from RFC 8785 matches the Truestamp producer: an integer
literal larger than 2^53 is emitted exactly as written instead of being
round-tripped through an IEEE-754 double (which would silently round it).
A warning naming such literals is written to stderr, because a document
containing them is not portably verifiable by a strict RFC 8785
implementation.

Examples:
  truestamp jcs < claims.json | truestamp hash -a sha256
  truestamp hash --prefix 0x11 --jcs < claims.json   # equivalent one-liner`,
	SilenceUsage:  true,
	SilenceErrors: true,
	RunE:          runJCS,
}

func runJCS(cmd *cobra.Command, args []string) error {
	jsonOut, _ := cmd.Flags().GetBool("json")
	silent, _ := cmd.Flags().GetBool("silent")
	newline, _ := cmd.Flags().GetBool("newline")

	if silent && jsonOut {
		return fmt.Errorf("--silent and --json are mutually exclusive")
	}

	data, src, err := resolveCodecInput(cmd, args, "Select JSON file")
	if err != nil {
		if errors.Is(err, inputsrc.ErrNoInput) {
			_ = cmd.Help()
			return nil
		}
		return err
	}

	canonical, oversized, err := jcs.Canonicalize(data)
	if err != nil {
		return fmt.Errorf("JCS canonicalization: %w", err)
	}

	if jsonOut {
		return emitJSON(cmd.OutOrStdout(), struct {
			InputBytes        int64    `json:"input_bytes"`
			OutputBytes       int      `json:"output_bytes"`
			Output            string   `json:"output"`
			OversizedIntegers []string `json:"oversized_integers,omitempty"`
		}{
			InputBytes:        src.Size,
			OutputBytes:       len(canonical),
			Output:            string(canonical),
			OversizedIntegers: oversized,
		})
	}

	warnOversizedIntegers(cmd, "", oversized, silent, jsonOut)

	if silent {
		return nil
	}
	_, _ = cmd.OutOrStdout().Write(canonical)
	if newline {
		_, _ = cmd.OutOrStdout().Write([]byte("\n"))
	}
	return nil
}

// warnOversizedIntegers reports the Appendix E.4 portability signal for a
// canonicalization that preserved an integer outside the exactly
// representable IEEE-754 double range.
//
// Truestamp emits such integers verbatim, so reproducing the producer's
// digest means preserving them, but a strict RFC 8785 implementation
// rounds them, and E.4 requires a verifier to say so rather than hide it.
// The warning is advisory: the digest above it is the correct one, so the
// exit code stays 0, matching how `truestamp hash` treats its legacy-
// algorithm notice. Suppressed under --silent and --json, both of which
// have their own channel for the same fact.
//
// label prefixes the line when the caller processes several inputs and a
// bare warning would not say which one it belongs to.
func warnOversizedIntegers(cmd *cobra.Command, label string, oversized []string, silent, jsonOut bool) {
	if len(oversized) == 0 || silent || jsonOut {
		return
	}
	prefix := ""
	if label != "" {
		prefix = label + ": "
	}
	// oversized is ascending, so [0] names the smallest offender, the
	// same one the reference verifier reports.
	suffix := ""
	if len(oversized) > 1 {
		suffix = fmt.Sprintf(" (and %d more)", len(oversized)-1)
	}
	fmt.Fprintf(cmd.ErrOrStderr(),
		"warning: %spreserved %d integer literal(s) larger than 2^53, e.g. %s%s; "+
			"this JSON is not portably verifiable by a strict RFC 8785 implementation\n",
		prefix, len(oversized), oversized[0], suffix)
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

	rootCmd.AddCommand(encodeCmd)
	rootCmd.AddCommand(decodeCmd)
	rootCmd.AddCommand(jcsCmd)
}
