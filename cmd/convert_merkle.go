// Copyright (c) 2021-2026 Truestamp, Inc.
// SPDX-License-Identifier: MIT

package cmd

import (
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/truestamp/truestamp-cli/internal/inputsrc"
	"github.com/truestamp/truestamp-cli/internal/tscrypto"
)

var convertMerkleCmd = &cobra.Command{
	Use:   "merkle [compact-base64url-proof]",
	Short: "Decode a compact base64url Merkle proof into its structure",
	Long: `Decode the compact base64url Merkle proof that appears as the 'ip'
(inclusion proof) or 'ep' (epoch proof) field in a Truestamp proof
bundle. The decoded structure lists each sibling hash with its
position ('left' or 'right') from leaf to root.

Examples:
  truestamp convert merkle "AQEA..."
  jq -r .ip proof.json | truestamp convert merkle`,
	SilenceUsage:  true,
	SilenceErrors: true,
	RunE:          runConvertMerkle,
}

func runConvertMerkle(cmd *cobra.Command, args []string) error {
	jsonOut, _ := cmd.Flags().GetBool("json")
	silent, _ := cmd.Flags().GetBool("silent")

	if silent && jsonOut {
		return fmt.Errorf("--silent and --json are mutually exclusive")
	}

	raw, err := gatherMerkleInput(cmd, args)
	if err != nil {
		return err
	}

	proof, err := tscrypto.DecodeCompactMerkleProof(raw)
	if err != nil {
		return err
	}

	type sibling struct {
		Position string `json:"position"`
		HashHex  string `json:"hash_hex"`
	}
	siblings := make([]sibling, 0, len(proof))
	for _, step := range proof {
		// Step format is "l:<hex>" or "r:<hex>" per internal/tscrypto/merkle.go.
		if idx := strings.IndexByte(step, ':'); idx == 1 {
			pos := "left"
			if step[0] == 'r' {
				pos = "right"
			}
			siblings = append(siblings, sibling{Position: pos, HashHex: step[idx+1:]})
		}
	}

	if jsonOut {
		return emitJSON(cmd.OutOrStdout(), struct {
			CompactBase64URL string    `json:"compact_base64url"`
			Depth            int       `json:"depth"`
			Siblings         []sibling `json:"siblings"`
		}{
			CompactBase64URL: raw,
			Depth:            len(siblings),
			Siblings:         siblings,
		})
	}

	if silent {
		return nil
	}

	out := cmd.OutOrStdout()
	fmt.Fprintf(out, "depth: %d\n", len(siblings))
	for i, s := range siblings {
		fmt.Fprintf(out, "  %2d  %-5s  %s\n", i, s.Position, s.HashHex)
	}
	return nil
}

func gatherMerkleInput(cmd *cobra.Command, args []string) (string, error) {
	if len(args) > 0 {
		return strings.TrimSpace(args[0]), nil
	}
	if inputsrc.IsStdinPipe() {
		data, err := io.ReadAll(io.LimitReader(os.Stdin, 64<<10))
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
	return "", errConvertNoMerkleInput
}

var errConvertNoMerkleInput = errors.New("no Merkle proof input provided")

func init() {
	f := convertMerkleCmd.Flags()
	f.Bool("json", false, "Output as JSON")
	f.BoolP("silent", "s", false, "No output, exit code only")
	convertCmd.AddCommand(convertMerkleCmd)
}
