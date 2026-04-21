// Copyright (c) 2021-2026 Truestamp, Inc.
// SPDX-License-Identifier: MIT

package cmd

import (
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/truestamp/truestamp-cli/internal/encoding"
	"github.com/truestamp/truestamp-cli/internal/inputsrc"
	"github.com/truestamp/truestamp-cli/internal/tscrypto"
)

var convertKeyIDCmd = &cobra.Command{
	Use:   "keyid [pubkey]",
	Short: "Derive the 4-byte Truestamp kid fingerprint from an Ed25519 public key",
	Long: `The Truestamp kid ("key ID") is computed as
truncate4(SHA256(0x51 || pubkey)) — the first 4 bytes of the SHA-256
digest of the key with domain prefix 0x51. Each proof's "kid" field is
this value rendered as 8 hex characters; deriving it locally lets you
cross-check the signing key your proof was issued against.

Examples:
  truestamp convert keyid 38fa3a80ba5....base64...=
  truestamp convert keyid --from hex 09a3c7ee...deadbeef...
  cat pubkey.b64 | truestamp convert keyid`,
	SilenceUsage:  true,
	SilenceErrors: true,
	RunE:          runConvertKeyID,
}

func runConvertKeyID(cmd *cobra.Command, args []string) error {
	fromName, _ := cmd.Flags().GetString("from")
	jsonOut, _ := cmd.Flags().GetBool("json")
	silent, _ := cmd.Flags().GetBool("silent")

	if silent && jsonOut {
		return fmt.Errorf("--silent and --json are mutually exclusive")
	}

	raw, err := gatherKeyInput(cmd, args)
	if err != nil {
		return err
	}

	pubkey, err := decodePublicKey(raw, fromName)
	if err != nil {
		return err
	}
	if len(pubkey) != 32 {
		return fmt.Errorf("Ed25519 public key must be 32 bytes (got %d)", len(pubkey))
	}

	kid := tscrypto.ComputeKeyID(pubkey)

	if jsonOut {
		b64Std := base64.StdEncoding.EncodeToString(pubkey)
		b64URL := base64.RawURLEncoding.EncodeToString(pubkey)
		return emitJSON(cmd.OutOrStdout(), struct {
			PublicKeyHex       string `json:"public_key_hex"`
			PublicKeyBase64    string `json:"public_key_base64"`
			PublicKeyBase64URL string `json:"public_key_base64url"`
			KIDHex             string `json:"kid_hex"`
		}{
			PublicKeyHex:       encodeHex(pubkey),
			PublicKeyBase64:    b64Std,
			PublicKeyBase64URL: b64URL,
			KIDHex:             kid,
		})
	}

	if silent {
		return nil
	}
	fmt.Fprintln(cmd.OutOrStdout(), kid)
	return nil
}

// decodePublicKey parses the public key from one of the accepted
// encodings. --from auto (default) tries hex → base64url → base64 in
// that order, which is unambiguous for 32-byte keys.
func decodePublicKey(raw, fromName string) ([]byte, error) {
	trimmed := strings.TrimSpace(raw)
	from := strings.ToLower(strings.TrimSpace(fromName))

	if from == "" || from == "auto" {
		// 32 bytes → 64 hex chars, 43 base64url chars, or 44 base64 chars.
		switch len(trimmed) {
		case 64:
			return encoding.Decode(encoding.Hex, []byte(trimmed))
		case 43:
			return encoding.Decode(encoding.Base64URL, []byte(trimmed))
		case 44:
			return encoding.Decode(encoding.Base64Std, []byte(trimmed))
		}
		// Fall back to trying each in turn.
		for _, enc := range []encoding.Encoding{encoding.Hex, encoding.Base64URL, encoding.Base64Std} {
			if out, err := encoding.Decode(enc, []byte(trimmed)); err == nil && len(out) == 32 {
				return out, nil
			}
		}
		return nil, fmt.Errorf("cannot auto-detect encoding of %q (try --from hex|base64|base64url)", trimmed)
	}

	enc, err := encoding.Parse(from)
	if err != nil {
		return nil, err
	}
	return encoding.Decode(enc, []byte(trimmed))
}

func gatherKeyInput(cmd *cobra.Command, args []string) (string, error) {
	if len(args) > 0 {
		return strings.TrimSpace(args[0]), nil
	}
	if inputsrc.IsStdinPipe() {
		data, err := io.ReadAll(io.LimitReader(os.Stdin, 1024))
		if err != nil {
			return "", fmt.Errorf("reading stdin: %w", err)
		}
		s := strings.TrimSpace(string(data))
		if s == "" {
			return "", fmt.Errorf("empty stdin input")
		}
		return s, nil
	}
	_ = cmd.Help()
	return "", errConvertNoKeyInput
}

var errConvertNoKeyInput = errors.New("no public key input provided")

func encodeHex(data []byte) string {
	out, _ := encoding.Encode(encoding.Hex, data)
	return string(out)
}

func init() {
	f := convertKeyIDCmd.Flags()
	f.String("from", "auto", "Public key encoding: auto, hex, base64, base64url")
	f.Bool("json", false, "Output as JSON")
	f.BoolP("silent", "s", false, "No output, exit code only")
	convertCmd.AddCommand(convertKeyIDCmd)
}
