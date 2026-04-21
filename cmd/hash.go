// Copyright (c) 2021-2026 Truestamp, Inc.
// SPDX-License-Identifier: MIT

package cmd

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/gowebpki/jcs"
	"github.com/spf13/cobra"
	"github.com/truestamp/truestamp-cli/internal/encoding"
	"github.com/truestamp/truestamp-cli/internal/hashing"
	"github.com/truestamp/truestamp-cli/internal/inputsrc"
)

// errHashFailed is returned when one or more inputs failed to hash, so
// the command exits non-zero without an additional "Error:" line from
// Execute() (we already wrote a per-file diagnostic to stderr, matching
// sha256sum's UX).
var errHashFailed = errors.New("one or more inputs failed to hash")

var hashCmd = &cobra.Command{
	Use:   "hash [flags] [path ...]",
	Short: "Compute cryptographic digests (SHA-2 / SHA-3 / BLAKE2 / MD5 / SHA-1)",
	Long: `Compute cryptographic digests for files, URLs, stdin, or a picked file.

Default output is byte-identical to GNU coreutils' sha256sum / md5sum
(text mode: "<hex>  <filename>\n") so it drops into existing scripts.
Pass --style bsd for "SHA256 (filename) = <hex>" or --style bare for
just the digest.

Input modes follow the same convention as 'truestamp verify':
  truestamp hash doc.pdf                   Hash a file
  truestamp hash a.bin b.bin c.bin         Hash multiple files
  truestamp hash -                         Hash stdin (Unix "-" alias)
  truestamp hash --file doc.pdf            Explicit file path
  truestamp hash --file                    Interactive file picker
  truestamp hash --url https://...         Download then hash
  truestamp hash --url                     Interactive URL prompt
  cat doc.pdf | truestamp hash             Pipe from stdin

Truestamp-domain extras:
  truestamp hash --prefix 0x11 --jcs < claims.json
      Canonicalize (RFC 8785) then hash with a one-byte domain prefix.
      Matches what the Truestamp backend computes for a claims_hash.
  truestamp hash --list
      Print supported algorithms.

Exit code 0 on success, 1 if any input failed.`,
	SilenceUsage:  true,
	SilenceErrors: true,
	RunE:          runHash,
}

func runHash(cmd *cobra.Command, args []string) error {
	if list, _ := cmd.Flags().GetBool("list"); list {
		printAlgorithmList(cmd.OutOrStdout())
		return nil
	}

	// Read the resolved config (defaults → TOML → env → flags). Reading
	// from appConfig — instead of cmd.Flags() directly — lets a user
	// change defaults in ~/.config/truestamp/config.toml without passing
	// --algorithm / --encoding / --style on every invocation.
	cfg := appConfig
	alg, err := hashing.Lookup(cfg.Hash.Algorithm)
	if err != nil {
		return err
	}

	enc, err := encoding.Parse(cfg.Hash.Encoding)
	if err != nil {
		return err
	}
	if enc == encoding.Binary {
		return fmt.Errorf("--encoding must be one of hex, base64, base64url (not binary)")
	}

	style := strings.ToLower(strings.TrimSpace(cfg.Hash.Style))
	switch style {
	case "", "gnu":
		style = "gnu"
	case "bsd", "bare":
		// ok
	default:
		return fmt.Errorf("--style must be gnu, bsd, or bare (got %q)", style)
	}

	binaryMode, _ := cmd.Flags().GetBool("binary")
	jsonOut, _ := cmd.Flags().GetBool("json")
	silent, _ := cmd.Flags().GetBool("silent")
	noFilename, _ := cmd.Flags().GetBool("no-filename")
	prefixFlag, _ := cmd.Flags().GetString("prefix")
	useJCS, _ := cmd.Flags().GetBool("jcs")

	if silent && jsonOut {
		return fmt.Errorf("--silent and --json are mutually exclusive")
	}

	// Parse --prefix as a single byte if set.
	var prefixByte byte
	var prefixSet bool
	if prefixFlag != "" {
		b, err := parsePrefixByte(prefixFlag)
		if err != nil {
			return err
		}
		prefixByte = b
		prefixSet = true
	}

	// Legacy-algorithm warning: emit once before any output, but only
	// when the user is seeing human output (not --json, not --silent).
	if alg.Legacy && !jsonOut && !silent {
		fmt.Fprintf(cmd.ErrOrStderr(),
			"warning: %s is cryptographically broken and unsuitable for security uses\n",
			alg.Name)
	}

	fileFlag, _ := cmd.Flags().GetString("file")
	urlFlag, _ := cmd.Flags().GetString("url")

	// Build the list of input sources. Positional args are per-input;
	// --file / --url are mutually exclusive with positional args (they
	// describe a single source).
	var sources []inputsrc.Options
	if fileFlag != "" || urlFlag != "" {
		if len(args) > 0 {
			return fmt.Errorf("--file/--url cannot be combined with positional file arguments")
		}
		sources = append(sources, inputsrc.Options{
			FileFlag:             fileFlag,
			URLFlag:              urlFlag,
			PickerTitle:          "Select file to hash",
			URLPromptTitle:       "Enter URL to hash",
			URLPromptPlaceholder: "https://example.com/file.bin",
		})
	} else if len(args) > 0 {
		for _, a := range args {
			sources = append(sources, inputsrc.Options{
				PositionalArg: a,
				AllowStdin:    a == "-",
			})
		}
	} else if inputsrc.IsStdinPipe() {
		sources = append(sources, inputsrc.Options{AllowStdin: true})
	} else {
		_ = cmd.Help()
		return nil
	}

	results := make([]hashResult, 0, len(sources))
	hadError := false
	for _, src := range sources {
		r, err := runHashOne(cmd.Context(), src, alg, prefixByte, prefixSet, useJCS)
		if err != nil {
			hadError = true
			if !silent {
				fmt.Fprintf(cmd.ErrOrStderr(), "truestamp hash: %s\n", err)
			}
			continue
		}
		results = append(results, r)
	}

	if silent {
		if hadError {
			return errSilentFail
		}
		return nil
	}

	if jsonOut {
		if err := emitHashJSON(cmd.OutOrStdout(), alg, enc, prefixFlag, useJCS, len(sources) > 1, results); err != nil {
			return err
		}
	} else {
		emitHashText(cmd.OutOrStdout(), alg, enc, style, binaryMode, noFilename, results)
	}

	if hadError {
		return errHashFailed
	}
	return nil
}

// hashResult is the per-input computed digest with its source info.
type hashResult struct {
	Source inputsrc.Source
	Digest []byte
	Size   int64
}

// runHashOne resolves one input and hashes it. For raw byte-stream inputs
// (no --prefix, no --jcs) it streams the reader through io.Copy into the
// algorithm's hasher. For --prefix or --jcs the whole input must be
// buffered first (JCS needs to parse+canonicalize, and a prefix must
// enter the hash before the payload does).
func runHashOne(ctx context.Context, opts inputsrc.Options, alg hashing.Algorithm, prefix byte, prefixSet, useJCS bool) (hashResult, error) {
	if !useJCS && !prefixSet {
		r, src, err := inputsrc.ResolveStream(ctx, opts)
		if err != nil {
			return hashResult{}, err
		}
		defer r.Close()
		digest, n, err := hashing.Compute(ctx, alg, r)
		if err != nil {
			return hashResult{}, err
		}
		if src.Size < 0 {
			src.Size = n
		}
		return hashResult{Source: src, Digest: digest, Size: n}, nil
	}

	// Buffered path for --prefix / --jcs.
	data, src, err := inputsrc.Resolve(ctx, opts)
	if err != nil {
		return hashResult{}, err
	}
	payload := data
	if useJCS {
		canonical, cErr := jcs.Transform(data)
		if cErr != nil {
			return hashResult{}, fmt.Errorf("%s: JCS canonicalization: %w", src.DisplayName(), cErr)
		}
		payload = canonical
	}
	h := alg.New()
	if prefixSet {
		h.Write([]byte{prefix})
	}
	h.Write(payload)
	digest := h.Sum(nil)
	return hashResult{Source: src, Digest: digest, Size: int64(len(data))}, nil
}

// emitHashText writes sha256sum-style (or BSD/bare) lines to w.
func emitHashText(w io.Writer, alg hashing.Algorithm, enc encoding.Encoding, style string, binaryMode, noFilename bool, results []hashResult) {
	for _, r := range results {
		digestEnc, _ := encoding.Encode(enc, r.Digest)
		name := r.Source.DisplayName()
		if name == "" {
			name = "-"
		}
		switch style {
		case "bare":
			if noFilename {
				fmt.Fprintf(w, "%s\n", string(digestEnc))
			} else {
				fmt.Fprintf(w, "%s  %s\n", string(digestEnc), name)
			}
		case "bsd":
			if noFilename {
				fmt.Fprintf(w, "%s = %s\n", alg.BSDName, string(digestEnc))
			} else {
				fmt.Fprint(w, hashing.FormatBSD(alg.BSDName, string(digestEnc), name))
			}
		default: // gnu
			if noFilename {
				fmt.Fprintf(w, "%s\n", string(digestEnc))
			} else {
				fmt.Fprint(w, hashing.FormatGNU(string(digestEnc), name, binaryMode))
			}
		}
	}
}

// jsonDigest holds the three standard textual forms of a digest for
// downstream consumers who may prefer one representation over another.
type jsonDigest struct {
	Hex       string `json:"hex"`
	Base64    string `json:"base64"`
	Base64URL string `json:"base64url"`
}

type jsonInput struct {
	Type string `json:"type"`
	Path string `json:"path,omitempty"`
}

type hashJSON struct {
	Algorithm string      `json:"algorithm"`
	Digest    jsonDigest  `json:"digest"`
	Encoded   string      `json:"encoded"` // digest rendered in the requested --encoding (for convenience)
	Encoding  string      `json:"encoding"`
	SizeBytes int64       `json:"size_bytes"`
	Prefix    string      `json:"prefix,omitempty"`
	JCS       bool        `json:"jcs,omitempty"`
	Input     jsonInput   `json:"input"`
}

func emitHashJSON(w io.Writer, alg hashing.Algorithm, enc encoding.Encoding, prefix string, useJCS, multi bool, results []hashResult) error {
	build := func(r hashResult) hashJSON {
		hexBytes, _ := encoding.Encode(encoding.Hex, r.Digest)
		b64Bytes, _ := encoding.Encode(encoding.Base64Std, r.Digest)
		b64URLBytes, _ := encoding.Encode(encoding.Base64URL, r.Digest)
		encBytes, _ := encoding.Encode(enc, r.Digest)
		return hashJSON{
			Algorithm: alg.Name,
			Digest: jsonDigest{
				Hex:       string(hexBytes),
				Base64:    string(b64Bytes),
				Base64URL: string(b64URLBytes),
			},
			Encoded:   string(encBytes),
			Encoding:  enc.Name(),
			SizeBytes: r.Size,
			Prefix:    prefix,
			JCS:       useJCS,
			Input: jsonInput{
				Type: string(r.Source.Type),
				Path: r.Source.Path,
			},
		}
	}

	var out any
	if multi {
		arr := make([]hashJSON, 0, len(results))
		for _, r := range results {
			arr = append(arr, build(r))
		}
		out = arr
	} else if len(results) == 1 {
		out = build(results[0])
	} else {
		out = []hashJSON{}
	}

	data, err := json.MarshalIndent(out, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling JSON: %w", err)
	}
	fmt.Fprintln(w, string(data))
	return nil
}

// parsePrefixByte accepts the user-friendly forms "0x11", "0X11", "11",
// or any two-hex-digit string and returns the single byte value. It
// rejects multi-byte values because the Truestamp wire format only uses
// one-byte domain prefixes.
func parsePrefixByte(s string) (byte, error) {
	s = strings.TrimSpace(s)
	s = strings.TrimPrefix(s, "0x")
	s = strings.TrimPrefix(s, "0X")
	if len(s) == 0 || len(s) > 2 {
		return 0, fmt.Errorf("--prefix must be a single byte in hex (e.g. 0x11, 11, 0xff)")
	}
	if len(s) == 1 {
		s = "0" + s
	}
	b, err := encoding.Decode(encoding.Hex, []byte(s))
	if err != nil {
		return 0, fmt.Errorf("--prefix: %w", err)
	}
	if len(b) != 1 {
		return 0, fmt.Errorf("--prefix must decode to exactly one byte")
	}
	return b[0], nil
}

// printAlgorithmList writes a stable, pipe-friendly table of supported
// algorithms — one per line with the canonical name, digest size, and
// any aliases in parentheses.
func printAlgorithmList(w io.Writer) {
	var buf bytes.Buffer
	for _, a := range hashing.Algorithms() {
		fmt.Fprintf(&buf, "%-13s %3d bytes", a.Name, a.Size)
		if len(a.Aliases) > 0 {
			fmt.Fprintf(&buf, "  (aliases: %s)", strings.Join(a.Aliases, ", "))
		}
		if a.Legacy {
			fmt.Fprint(&buf, "  [legacy]")
		}
		buf.WriteByte('\n')
	}
	_, _ = w.Write(buf.Bytes())
}

func init() {
	f := hashCmd.Flags()
	f.StringP("algorithm", "a", "sha256", "Hash algorithm (see --list)")
	f.Bool("list", false, "List supported algorithms and exit")
	f.StringP("encoding", "e", "hex", "Digest output encoding: hex, base64, base64url")
	f.String("style", "gnu", "Output style: gnu (default), bsd, bare")
	f.Bool("binary", false, "Binary mode (gnu style uses space+asterisk instead of two spaces)")
	f.String("file", "", "Path to file (interactive picker if no path given)")
	f.String("url", "", "URL to download (interactive prompt if no URL given)")
	f.Lookup("file").NoOptDefVal = inputsrc.FilePickSentinel
	f.Lookup("url").NoOptDefVal = inputsrc.URLPromptSentinel
	f.String("prefix", "", "Prepend single domain-separation byte before hashing (e.g. 0x11)")
	f.Bool("jcs", false, "Apply RFC 8785 JCS canonicalization before hashing (input must be JSON)")
	f.Bool("json", false, "Output as JSON")
	f.BoolP("silent", "s", false, "No output, exit code only")
	f.Bool("no-filename", false, "Omit the filename from text output")
	rootCmd.AddCommand(hashCmd)
}
