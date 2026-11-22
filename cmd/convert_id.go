// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: Apache-2.0

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
	"github.com/spf13/cobra"
	"github.com/truestamp/truestamp-cli/internal/inputsrc"
	"github.com/truestamp/truestamp-cli/internal/proof"
	"github.com/truestamp/truestamp-cli/internal/tscrypto"
	"github.com/truestamp/truestamp-cli/internal/ui"
)

var convertIDCmd = &cobra.Command{
	Use:   "id [value]",
	Args:  cobra.MaximumNArgs(1),
	Short: "Read the time recorded inside a UUIDv7 id",
	Long: `Every Truestamp id, of an item, a block, an entropy observation or a
beacon, is a UUIDv7, whose leftmost 48 bits are a time: the milliseconds
since the Unix epoch at which Truestamp recorded the id (SPEC.md section
11.3). It is a recorded time, never a proven one: a proof's submission
window comes from its commitments and witnesses, not from its id. This
sub-command reads that time and prints it.

Nothing is trimmed from the value, a UUIDv7 is read in either case, and a UUID's
32-hex, braced and urn: forms and every other version are refused. Read
from a pipe, the one line ending that ends the line, a line feed or a
carriage return and line feed, is not part of the value. An item code
carries no time, and is refused with that reason.

Examples:
  truestamp convert id 01a0db66-c2e0-78a4-b8cc-12b11bc97493
  truestamp convert id 019cf813-99b8-730a-84f1-5a711a9c355e --to-zone Local
  truestamp convert id 019cf813-99b8-730a-84f1-5a711a9c355e --extract raw
  echo 01a0db66-c2e0-78a4-b8cc-12b11bc97493 | truestamp convert id --json`,
	RunE: runConvertID,
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

	switch strings.ToLower(strings.TrimSpace(typeName)) {
	case "", "auto", "uuid7":
	default:
		return fmt.Errorf("--type must be auto or uuid7 (got %q)", typeName)
	}

	raw, err := gatherIDInput(cmd, args)
	if err != nil {
		return err
	}
	id, err := uuidv7WithTime(raw)
	if err != nil {
		return err
	}
	return emitUUIDv7Info(cmd, id, extract, toZone, jsonOut, silent)
}

// uuidv7WithTime reads raw exactly as given, as proof.DetectIDType
// classifies it, and returns the UUIDv7 in its canonical lowercase form.
// An item code and an item_hash are refused for what they are: neither
// carries a time.
func uuidv7WithTime(raw string) (string, error) {
	shape, id, err := proof.DetectIDType(raw)
	if err != nil {
		return "", nameTheFormTaken(err, "convert id takes a UUIDv7 (e.g. 019d6a32-13e6-72b0-97e5-3779231ea97b)")
	}
	switch shape {
	case proof.IDTypeItemCode:
		return "", fmt.Errorf("%s is an item code: an item code carries no time", id)
	case proof.IDTypeItemHash:
		return "", fmt.Errorf("%q is 64 hex characters, an item_hash at most: a hash carries no time", raw)
	}
	return id, nil
}

// gatherIDInput is the positional value exactly as typed or, with none
// and a pipe on stdin, the line read from it: the one line ending that
// ends it, "\r\n" or "\n", is not part of the value, and nothing else is
// removed.
func gatherIDInput(cmd *cobra.Command, args []string) (string, error) {
	if len(args) > 0 {
		return args[0], nil
	}
	if inputsrc.IsStdinPipe() {
		data, err := io.ReadAll(io.LimitReader(os.Stdin, 256))
		if err != nil {
			return "", fmt.Errorf("reading stdin: %w", err)
		}
		s, crlf := strings.CutSuffix(string(data), "\r\n")
		if !crlf {
			s = strings.TrimSuffix(s, "\n")
		}
		if s == "" {
			return "", fmt.Errorf("empty stdin input")
		}
		return s, nil
	}
	_ = cmd.Help()
	return "", errConvertNoIDInput
}

var errConvertNoIDInput = errors.New("no ID input provided")

func emitUUIDv7Info(cmd *cobra.Command, id, extract, toZone string, jsonOut, silent bool) error {
	parsed, err := uuid.FromString(id)
	if err != nil {
		return fmt.Errorf("UUID parse: %w", err)
	}
	t, err := tscrypto.ExtractUUIDv7Timestamp(id)
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
			ui.Fprintln(cmd.OutOrStdout(), hex.EncodeToString(parsed[:]))
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
			Value:        id,
			TimestampMS:  t.UnixMilli(),
			TimestampUTC: t.UTC().Format(time.RFC3339Nano),
			Zone:         tz.name,
			Local:        localized.Format(time.RFC3339Nano),
		})
	}
	if !silent {
		ui.Fprintln(cmd.OutOrStdout(), localized.Format(time.RFC3339Nano))
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
	f.String("type", "auto", "ID type: auto, uuid7")
	f.String("extract", "time", "What to extract: time, raw")
	f.String("to-zone", "UTC", "Target IANA zone for time output")
	addRecordOutputFlags(convertIDCmd)
	convertCmd.AddCommand(convertIDCmd)
}
