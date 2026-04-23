// Copyright (c) 2021-2026 Truestamp, Inc.
// SPDX-License-Identifier: MIT

package cmd

import (
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/gofrs/uuid/v5"
	"github.com/oklog/ulid/v2"
	"github.com/spf13/cobra"
	"github.com/truestamp/truestamp-cli/internal/inputsrc"
	"github.com/truestamp/truestamp-cli/internal/tscrypto"
)

var convertIDCmd = &cobra.Command{
	Use:   "id [value]",
	Short: "Extract the embedded timestamp from a ULID or UUIDv7",
	Long: `Both ULID (used for Truestamp item IDs) and UUIDv7 (used for block IDs)
carry a millisecond timestamp in their first 48 bits. This sub-command
parses either form and emits the timestamp — useful when inspecting
proof bundles.

Examples:
  truestamp convert id 01KNN33GX5E470CB9TRWAYF9DD
  truestamp convert id 019cf813-99b8-730a-84f1-5a711a9c355e --to-zone Local
  truestamp convert id 019cf813-99b8-730a-84f1-5a711a9c355e --extract raw
  echo 01KNN33GX5E470CB9TRWAYF9DD | truestamp convert id --json`,
	SilenceUsage:  true,
	SilenceErrors: true,
	RunE:          runConvertID,
}

func runConvertID(cmd *cobra.Command, args []string) error {
	typeName, _ := cmd.Flags().GetString("type")
	extract, _ := cmd.Flags().GetString("extract")
	toZone := appConfig.Convert.TimeZone
	if cmd.Flags().Changed("to-zone") {
		toZone, _ = cmd.Flags().GetString("to-zone")
	}
	jsonOut, _ := cmd.Flags().GetBool("json")
	silent, _ := cmd.Flags().GetBool("silent")

	if silent && jsonOut {
		return fmt.Errorf("--silent and --json are mutually exclusive")
	}

	raw, err := gatherIDInput(cmd, args)
	if err != nil {
		return err
	}

	kind := strings.ToLower(strings.TrimSpace(typeName))
	if kind == "" || kind == "auto" {
		kind = detectIDKind(raw)
		if kind == "" {
			return fmt.Errorf("cannot auto-detect %q as a ULID or UUIDv7 (use --type)", raw)
		}
	}

	switch kind {
	case "ulid":
		return emitULIDInfo(cmd, raw, extract, toZone, jsonOut, silent)
	case "uuid7", "uuid", "uuidv7":
		return emitUUIDv7Info(cmd, raw, extract, toZone, jsonOut, silent)
	}
	return fmt.Errorf("--type must be ulid or uuid7 (got %q)", typeName)
}

// detectIDKind classifies common formats by shape. ULID is 26 chars of
// Crockford base32; UUID is either 32 hex chars or 8-4-4-4-12 dashed.
func detectIDKind(s string) string {
	s = strings.TrimSpace(s)
	if len(s) == 26 && !strings.ContainsRune(s, '-') {
		return "ulid"
	}
	if strings.Count(s, "-") == 4 && len(s) == 36 {
		return "uuid7"
	}
	if len(s) == 32 && !strings.ContainsRune(s, '-') {
		return "uuid7"
	}
	return ""
}

func gatherIDInput(cmd *cobra.Command, args []string) (string, error) {
	if len(args) > 0 {
		return strings.TrimSpace(args[0]), nil
	}
	if inputsrc.IsStdinPipe() {
		data, err := io.ReadAll(io.LimitReader(os.Stdin, 256))
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
	return "", errConvertNoIDInput
}

var errConvertNoIDInput = errors.New("no ID input provided")

func emitULIDInfo(cmd *cobra.Command, raw, extract, toZone string, jsonOut, silent bool) error {
	parsed, err := ulid.ParseStrict(raw)
	if err != nil {
		return fmt.Errorf("ULID parse: %w", err)
	}
	ms := parsed.Time()
	t := time.UnixMilli(int64(ms))
	tz, err := resolveZone(toZone)
	if err != nil {
		return err
	}
	localized := t.In(tz.loc)

	if extract == "raw" {
		if jsonOut {
			return emitJSON(cmd.OutOrStdout(), struct {
				Type string `json:"type"`
				Raw  string `json:"raw_hex"`
			}{
				Type: "ulid",
				Raw:  hex.EncodeToString(parsed[:]),
			})
		}
		if !silent {
			fmt.Fprintln(cmd.OutOrStdout(), hex.EncodeToString(parsed[:]))
		}
		return nil
	}

	if jsonOut {
		return emitJSON(cmd.OutOrStdout(), struct {
			Type         string `json:"type"`
			Value        string `json:"value"`
			TimestampMS  uint64 `json:"timestamp_ms"`
			TimestampUTC string `json:"timestamp_utc"`
			Zone         string `json:"zone"`
			Local        string `json:"local"`
		}{
			Type:         "ulid",
			Value:        raw,
			TimestampMS:  ms,
			TimestampUTC: t.UTC().Format(time.RFC3339Nano),
			Zone:         tz.name,
			Local:        localized.Format(time.RFC3339Nano),
		})
	}
	if !silent {
		fmt.Fprintln(cmd.OutOrStdout(), localized.Format(time.RFC3339Nano))
	}
	return nil
}

func emitUUIDv7Info(cmd *cobra.Command, raw, extract, toZone string, jsonOut, silent bool) error {
	parsed, err := uuid.FromString(raw)
	if err != nil {
		return fmt.Errorf("UUID parse: %w", err)
	}
	// Prefer the shared helper for interop with the existing verifier.
	t, err := tscrypto.ExtractUUIDv7Timestamp(raw)
	if err != nil {
		return err
	}
	tz, err := resolveZone(toZone)
	if err != nil {
		return err
	}
	localized := t.In(tz.loc)

	if extract == "raw" {
		if jsonOut {
			return emitJSON(cmd.OutOrStdout(), struct {
				Type string `json:"type"`
				Raw  string `json:"raw_hex"`
			}{
				Type: "uuid7",
				Raw:  hex.EncodeToString(parsed[:]),
			})
		}
		if !silent {
			fmt.Fprintln(cmd.OutOrStdout(), hex.EncodeToString(parsed[:]))
		}
		return nil
	}

	if jsonOut {
		return emitJSON(cmd.OutOrStdout(), struct {
			Type         string `json:"type"`
			Value        string `json:"value"`
			TimestampMS  int64  `json:"timestamp_ms"`
			TimestampUTC string `json:"timestamp_utc"`
			Zone         string `json:"zone"`
			Local        string `json:"local"`
		}{
			Type:         "uuid7",
			Value:        raw,
			TimestampMS:  t.UnixMilli(),
			TimestampUTC: t.UTC().Format(time.RFC3339Nano),
			Zone:         tz.name,
			Local:        localized.Format(time.RFC3339Nano),
		})
	}
	if !silent {
		fmt.Fprintln(cmd.OutOrStdout(), localized.Format(time.RFC3339Nano))
	}
	return nil
}

// zoneInfo bundles a time.Location with the user-facing name for output.
type zoneInfo struct {
	loc  *time.Location
	name string
}

func resolveZone(zone string) (zoneInfo, error) {
	if zone == "" {
		return zoneInfo{loc: time.UTC, name: "UTC"}, nil
	}
	switch strings.ToLower(zone) {
	case "utc":
		return zoneInfo{loc: time.UTC, name: "UTC"}, nil
	case "local":
		return zoneInfo{loc: time.Local, name: "Local"}, nil
	}
	l, err := time.LoadLocation(zone)
	if err != nil {
		return zoneInfo{}, fmt.Errorf("--to-zone %q: %w", zone, err)
	}
	return zoneInfo{loc: l, name: zone}, nil
}

func init() {
	f := convertIDCmd.Flags()
	f.String("type", "auto", "ID type: auto, ulid, uuid7")
	f.String("extract", "time", "What to extract: time, raw")
	f.String("to-zone", "UTC", "Target IANA zone for time output (default UTC)")
	addConvertCommonFlags(convertIDCmd)
	convertCmd.AddCommand(convertIDCmd)
}
