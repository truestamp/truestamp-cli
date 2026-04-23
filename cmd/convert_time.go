// Copyright (c) 2021-2026 Truestamp, Inc.
// SPDX-License-Identifier: MIT

package cmd

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/truestamp/truestamp-cli/internal/inputsrc"
)

var convertTimeCmd = &cobra.Command{
	Use:   "time [input]",
	Short: "Parse and re-format a timestamp across zones and Unix formats",
	Long: `Parse a timestamp from RFC 3339 or a Unix numeric form and re-emit
it in a target zone and format.

Input is the positional argument, stdin, or the literal "now".

Input forms detected with --from auto:
  RFC 3339 / ISO 8601   e.g. 2026-04-21T12:00:00Z, 2026-04-21T12:00:00+05:30
  unix-s                10-digit seconds
  unix-ms               13-digit milliseconds
  unix-us               16-digit microseconds
  unix-ns               19-digit nanoseconds
  "now"                 the current system time

Examples:
  truestamp convert time now --to-zone UTC
  truestamp convert time "2026-04-21T12:00:00Z" --to-zone America/New_York
  truestamp convert time 1700000000 --format unix-ms
  date | truestamp convert time --from rfc3339 --to-zone Asia/Kolkata`,
	SilenceUsage:  true,
	SilenceErrors: true,
	RunE:          runConvertTime,
}

func runConvertTime(cmd *cobra.Command, args []string) error {
	fromName, _ := cmd.Flags().GetString("from")
	toZone := appConfig.Convert.TimeZone
	if cmd.Flags().Changed("to-zone") {
		toZone, _ = cmd.Flags().GetString("to-zone")
	}
	format, _ := cmd.Flags().GetString("format")
	jsonOut, _ := cmd.Flags().GetBool("json")
	silent, _ := cmd.Flags().GetBool("silent")

	if silent && jsonOut {
		return fmt.Errorf("--silent and --json are mutually exclusive")
	}

	raw, err := gatherTimeInput(cmd, args)
	if err != nil {
		return err
	}

	t, err := parseTime(raw, strings.ToLower(fromName))
	if err != nil {
		return err
	}

	loc := time.UTC
	zoneName := "UTC"
	if toZone != "" {
		lower := strings.ToLower(toZone)
		switch lower {
		case "utc":
			loc = time.UTC
			zoneName = "UTC"
		case "local":
			loc = time.Local
			zoneName = "Local"
		default:
			l, lErr := time.LoadLocation(toZone)
			if lErr != nil {
				return fmt.Errorf("--to-zone %q: %w", toZone, lErr)
			}
			loc = l
			zoneName = toZone
		}
	}
	localized := t.In(loc)

	out, err := formatTime(localized, format)
	if err != nil {
		return err
	}

	if jsonOut {
		return emitJSON(cmd.OutOrStdout(), struct {
			Input       string `json:"input"`
			ParsedUTC   string `json:"parsed_utc"`
			OutputZone  string `json:"output_zone"`
			Output      string `json:"output"`
			UnixSeconds int64  `json:"unix_s"`
			UnixMillis  int64  `json:"unix_ms"`
		}{
			Input:       raw,
			ParsedUTC:   t.UTC().Format(time.RFC3339Nano),
			OutputZone:  zoneName,
			Output:      out,
			UnixSeconds: t.Unix(),
			UnixMillis:  t.UnixMilli(),
		})
	}

	if silent {
		return nil
	}
	fmt.Fprintln(cmd.OutOrStdout(), out)
	return nil
}

// gatherTimeInput accepts the positional arg, the literal "now", or a
// single line of stdin. Trailing whitespace is trimmed so piping from
// `date` works without a user-visible chomp.
func gatherTimeInput(cmd *cobra.Command, args []string) (string, error) {
	if len(args) > 0 {
		return strings.TrimSpace(args[0]), nil
	}
	if inputsrc.IsStdinPipe() {
		data, err := io.ReadAll(io.LimitReader(os.Stdin, 4096))
		if err != nil {
			return "", fmt.Errorf("reading stdin: %w", err)
		}
		s := strings.TrimSpace(string(data))
		if s == "" {
			return "", fmt.Errorf("empty stdin input")
		}
		return s, nil
	}
	// No explicit input.
	_ = cmd.Help()
	return "", errConvertNoInput
}

var errConvertNoInput = errors.New("no time input provided")

// parseTime dispatches on --from, with "auto" inferring the shape.
func parseTime(raw, from string) (time.Time, error) {
	lower := strings.ToLower(raw)
	if lower == "now" {
		return time.Now(), nil
	}

	switch from {
	case "", "auto":
		if isAllDigits(raw) {
			return parseUnixByLength(raw)
		}
		return parseRFC3339(raw)
	case "rfc3339", "iso8601":
		return parseRFC3339(raw)
	case "unix-s", "unix", "s", "seconds":
		return parseUnixWith(raw, time.Second)
	case "unix-ms", "ms", "milliseconds":
		return parseUnixWith(raw, time.Millisecond)
	case "unix-us", "us", "microseconds":
		return parseUnixWith(raw, time.Microsecond)
	case "unix-ns", "ns", "nanoseconds":
		return parseUnixWith(raw, time.Nanosecond)
	}
	return time.Time{}, fmt.Errorf("--from %q not recognized (try auto, rfc3339, unix-s, unix-ms, unix-us, unix-ns)", from)
}

func parseRFC3339(raw string) (time.Time, error) {
	for _, layout := range []string{time.RFC3339Nano, time.RFC3339, "2006-01-02"} {
		if t, err := time.Parse(layout, raw); err == nil {
			return t, nil
		}
	}
	return time.Time{}, fmt.Errorf("cannot parse %q as RFC 3339 / ISO 8601 (try 2006-01-02T15:04:05Z)", raw)
}

// parseUnixByLength guesses the precision of a numeric Unix timestamp
// from its digit count. The transition points are deliberately generous
// so that times near the epoch still parse as seconds (10 digits covers
// year 2001 through 2286; 13 digits is the canonical milliseconds form).
func parseUnixByLength(raw string) (time.Time, error) {
	switch {
	case len(raw) <= 10:
		return parseUnixWith(raw, time.Second)
	case len(raw) <= 13:
		return parseUnixWith(raw, time.Millisecond)
	case len(raw) <= 16:
		return parseUnixWith(raw, time.Microsecond)
	default:
		return parseUnixWith(raw, time.Nanosecond)
	}
}

func parseUnixWith(raw string, unit time.Duration) (time.Time, error) {
	n, err := strconv.ParseInt(raw, 10, 64)
	if err != nil {
		return time.Time{}, fmt.Errorf("parse unix %s: %w", unit, err)
	}
	switch unit {
	case time.Second:
		return time.Unix(n, 0), nil
	case time.Millisecond:
		return time.UnixMilli(n), nil
	case time.Microsecond:
		return time.UnixMicro(n), nil
	case time.Nanosecond:
		return time.Unix(0, n), nil
	}
	return time.Time{}, fmt.Errorf("unsupported unit %s", unit)
}

// formatTime renders t in the requested --format. Empty/"rfc3339"
// produces RFC 3339 nanosecond precision; the unix-* forms produce
// plain integers; anything else is treated as a Go time layout.
func formatTime(t time.Time, format string) (string, error) {
	switch strings.ToLower(strings.TrimSpace(format)) {
	case "", "rfc3339", "iso8601":
		return t.Format(time.RFC3339Nano), nil
	case "unix-s", "unix", "s":
		return strconv.FormatInt(t.Unix(), 10), nil
	case "unix-ms", "ms":
		return strconv.FormatInt(t.UnixMilli(), 10), nil
	case "unix-us", "us":
		return strconv.FormatInt(t.UnixMicro(), 10), nil
	case "unix-ns", "ns":
		return strconv.FormatInt(t.UnixNano(), 10), nil
	default:
		// Treat as a literal Go time layout ("2006-01-02 15:04:05", etc.).
		return t.Format(format), nil
	}
}

func isAllDigits(s string) bool {
	s = strings.TrimPrefix(s, "-")
	if s == "" {
		return false
	}
	return bytes.IndexFunc([]byte(s), func(r rune) bool { return r < '0' || r > '9' }) == -1
}

func init() {
	f := convertTimeCmd.Flags()
	f.String("from", "auto", "Input format: auto, rfc3339, unix-s, unix-ms, unix-us, unix-ns")
	f.String("to-zone", "UTC", "Target IANA time zone (e.g. UTC, America/New_York, Local)")
	f.String("format", "rfc3339", "Output format: rfc3339, unix-s, unix-ms, unix-us, unix-ns, or a Go time layout")
	addConvertCommonFlags(convertTimeCmd)
	convertCmd.AddCommand(convertTimeCmd)
}
