// Copyright (c) 2021-2026 Truestamp, Inc.
// SPDX-License-Identifier: MIT

package cmd

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	lipgloss "charm.land/lipgloss/v2"
	"charm.land/lipgloss/v2/table"
	"github.com/spf13/cobra"
	"github.com/truestamp/truestamp-cli/internal/items"
	"github.com/truestamp/truestamp-cli/internal/ui"
)

// Sentinels for flags passed without a value (via NoOptDefVal). See
// the equivalent block in verify.go for the rationale behind NUL bytes.
const (
	claimsFlagPick     = "\x00pick"
	fileFlagPickCreate = "\x00pick"
)

var createCmd = &cobra.Command{
	Use:   "create [file]",
	Short: "Create a new Truestamp item",
	Long: `Create a new cryptographic timestamp item.

The simplest usage hashes a file and creates an item in one step:
  truestamp create document.pdf

This computes SHA-256 of the file, uses the filename as the item name,
and submits it to the Truestamp API. The item will be included in the
next block for cryptographic commitment.

Input methods (resolved in priority order):
  truestamp create document.pdf              Hash file (SHA-256, filename as name)
  truestamp create --file document.pdf       Explicit file path to hash
  truestamp create --file                    Interactive file picker to hash
  cat doc.pdf | truestamp create -F -n Doc   Hash file content from stdin
  truestamp create -c claims.json            Load claims from JSON file
  truestamp create --claims                  Interactive claims JSON file picker
  cat claims.json | truestamp create -C      Read claims JSON from stdin
  truestamp create -n "Doc" --hash abc...    Build claims from flags

Flags override values from file/auto-hash, enabling combinations like:
  truestamp create report.pdf -n "Q1 Report" -v public -t finance

Requires --api-key to be set (via flag, env, or config file).`,
	Args:          cobra.MaximumNArgs(1),
	SilenceUsage:  true,
	SilenceErrors: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		// Resolve claims from input sources first so `truestamp create`
		// with no args shows help without requiring an API key.
		claims, err := resolveCreateInput(cmd, args)
		if err != nil {
			return err
		}
		if claims == nil {
			cmd.Help()
			return nil
		}

		cfg := appConfig
		if cfg.APIKey == "" {
			return fmt.Errorf("API key required (use --api-key or set TRUESTAMP_API_KEY)")
		}

		jsonOutput, _ := cmd.Flags().GetBool("json")

		// Overlay flag values onto claims
		if err := overlayFlags(cmd, claims); err != nil {
			return err
		}

		// Client-side validation
		if err := validateClaims(claims); err != nil {
			return err
		}

		// Resolve visibility and tags
		visibility, _ := cmd.Flags().GetString("visibility")
		tagsStr, _ := cmd.Flags().GetString("tags")
		var tags []string
		if tagsStr != "" {
			for _, t := range strings.Split(tagsStr, ",") {
				t = strings.TrimSpace(t)
				if t != "" {
					tags = append(tags, t)
				}
			}
		}

		// Validate visibility
		if visibility != "" {
			switch visibility {
			case "private", "team", "public":
			default:
				return fmt.Errorf("--visibility must be private, team, or public, got %q", visibility)
			}
		}

		// Create the item
		resp, err := items.CreateItemCtx(cmd.Context(), cfg.APIURL, cfg.APIKey, cfg.Team, claims, visibility, tags)
		if err != nil {
			return err
		}

		// Output
		if jsonOutput {
			return printCreateJSON(resp)
		}
		presentCreate(resp)
		return nil
	},
}

// resolveCreateInput determines claims from flags, file, stdin, or auto-hash.
// Priority: --claims > --claims-stdin > --file > --file-stdin > positional arg > flags-only > help
func resolveCreateInput(cmd *cobra.Command, args []string) (map[string]any, error) {
	claimsFlag, _ := cmd.Flags().GetString("claims")
	claimsStdin, _ := cmd.Flags().GetBool("claims-stdin")
	fileFlag, _ := cmd.Flags().GetString("file")
	fileStdin, _ := cmd.Flags().GetBool("file-stdin")

	switch {
	// --claims: load claims JSON from file (picker if no path)
	case claimsFlag == claimsFlagPick:
		path, err := pickClaimsFile()
		if err != nil {
			return nil, err
		}
		return readClaimsFile(path)

	case claimsFlag != "":
		return readClaimsFile(claimsFlag)

	// --claims-stdin: read claims JSON from stdin
	case claimsStdin:
		return readClaimsStdin()

	// --file: auto-hash a file (picker if no path)
	case fileFlag == fileFlagPickCreate:
		path, err := pickAnyFile()
		if err != nil {
			return nil, err
		}
		return autoHashFile(path)

	case fileFlag != "":
		return autoHashFileChecked(fileFlag)

	// --file-stdin: hash raw file content from stdin (requires --name)
	case fileStdin:
		return autoHashStdin()

	// Positional arg: auto-hash file
	case len(args) > 0:
		return autoHashFileChecked(args[0])

	// Flag-only mode: build claims from --name + --hash
	default:
		name, _ := cmd.Flags().GetString("name")
		hash, _ := cmd.Flags().GetString("hash")
		if name != "" || hash != "" {
			return make(map[string]any), nil
		}
		return nil, nil // no input = show help
	}
}

// autoHashFileChecked validates the path then auto-hashes.
func autoHashFileChecked(path string) (map[string]any, error) {
	info, err := os.Stat(path)
	if err != nil {
		return nil, fmt.Errorf("cannot access %q: %w", path, err)
	}
	if info.IsDir() {
		return nil, fmt.Errorf("%q is a directory, expected a file", path)
	}
	return autoHashFile(path)
}

// autoHashStdin reads raw content from stdin and computes SHA-256.
// The caller must provide --name since there is no filename to derive it from.
func autoHashStdin() (map[string]any, error) {
	data, err := io.ReadAll(io.LimitReader(os.Stdin, 100<<20)) // 100 MB limit
	if err != nil {
		return nil, fmt.Errorf("reading stdin: %w", err)
	}
	if len(data) == 0 {
		return nil, fmt.Errorf("no data received on stdin")
	}

	h := sha256.New()
	h.Write(data)

	return map[string]any{
		"hash":      hex.EncodeToString(h.Sum(nil)),
		"hash_type": "sha256",
	}, nil
}

// autoHashFile computes SHA-256 of a file and returns claims.
func autoHashFile(path string) (map[string]any, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("opening file: %w", err)
	}
	defer f.Close()

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return nil, fmt.Errorf("hashing file: %w", err)
	}

	return map[string]any{
		"hash":      hex.EncodeToString(h.Sum(nil)),
		"hash_type": "sha256",
		"name":      filepath.Base(path),
	}, nil
}

// readClaimsFile reads and parses a JSON claims file.
func readClaimsFile(path string) (map[string]any, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading claims file: %w", err)
	}
	var claims map[string]any
	if err := json.Unmarshal(data, &claims); err != nil {
		return nil, fmt.Errorf("parsing claims JSON: %w", err)
	}
	return claims, nil
}

// readClaimsStdin reads claims JSON from stdin.
func readClaimsStdin() (map[string]any, error) {
	data, err := io.ReadAll(io.LimitReader(os.Stdin, 1<<20))
	if err != nil {
		return nil, fmt.Errorf("reading stdin: %w", err)
	}
	if len(data) == 0 {
		return nil, fmt.Errorf("no data received on stdin")
	}
	var claims map[string]any
	if err := json.Unmarshal(data, &claims); err != nil {
		return nil, fmt.Errorf("parsing claims JSON from stdin: %w", err)
	}
	return claims, nil
}

// pickAnyFile launches an interactive file picker for any file type.
func pickAnyFile() (string, error) {
	return ui.PickFile(ui.PickFileOptions{Title: "Select file to hash"})
}

// pickClaimsFile launches an interactive file picker for claims JSON.
func pickClaimsFile() (string, error) {
	return ui.PickFile(ui.PickFileOptions{
		Title:        "Select claims JSON file",
		AllowedTypes: []string{".json"},
	})
}

// normalizeTimestamp accepts the ISO 8601 forms documented on the
// --timestamp flag and returns the value in canonical RFC3339 form. Date-only
// inputs are promoted to midnight UTC to give the server a fully-specified
// instant without silently dropping the user's intent.
func normalizeTimestamp(raw string) (string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", fmt.Errorf("--timestamp is empty")
	}
	layouts := []string{
		time.RFC3339Nano,
		time.RFC3339,
		"2006-01-02",
	}
	for _, layout := range layouts {
		if t, err := time.Parse(layout, raw); err == nil {
			return t.UTC().Format(time.RFC3339), nil
		}
	}
	return "", fmt.Errorf("--timestamp %q is not a supported ISO 8601 form (try 2025-01-15 or 2025-01-15T14:30:00Z)", raw)
}

// overlayFlags applies flag values onto claims, overriding existing values.
func overlayFlags(cmd *cobra.Command, claims map[string]any) error {
	setIfChanged := func(flag, key string) {
		if cmd.Flags().Changed(flag) {
			v, _ := cmd.Flags().GetString(flag)
			claims[key] = v
		}
	}

	setIfChanged("name", "name")
	setIfChanged("hash", "hash")
	setIfChanged("hash-type", "hash_type")
	setIfChanged("description", "description")
	setIfChanged("url", "url")

	// Timestamp: validate ISO 8601 locally and re-emit in canonical RFC3339
	// form so the server sees a normalized value and the user gets a clear
	// local error for typos.
	if cmd.Flags().Changed("timestamp") {
		raw, _ := cmd.Flags().GetString("timestamp")
		normalized, err := normalizeTimestamp(raw)
		if err != nil {
			return err
		}
		claims["timestamp"] = normalized
	}

	// Metadata: parse JSON string
	if cmd.Flags().Changed("metadata") {
		metaStr, _ := cmd.Flags().GetString("metadata")
		var meta map[string]any
		if err := json.Unmarshal([]byte(metaStr), &meta); err != nil {
			return fmt.Errorf("--metadata must be valid JSON: %w", err)
		}
		claims["metadata"] = meta
	}

	// Location: parse "lat,lon"
	if cmd.Flags().Changed("location") {
		locStr, _ := cmd.Flags().GetString("location")
		parts := strings.SplitN(locStr, ",", 2)
		if len(parts) != 2 {
			return fmt.Errorf("--location must be lat,lon (e.g. 37.7749,-122.4194)")
		}
		lat, err := strconv.ParseFloat(strings.TrimSpace(parts[0]), 64)
		if err != nil {
			return fmt.Errorf("--location latitude: %w", err)
		}
		lon, err := strconv.ParseFloat(strings.TrimSpace(parts[1]), 64)
		if err != nil {
			return fmt.Errorf("--location longitude: %w", err)
		}
		claims["location"] = map[string]any{"latitude": lat, "longitude": lon}
	}

	// Default hash_type if hash is provided but hash_type is not set
	if _, hasHash := claims["hash"]; hasHash {
		if _, hasType := claims["hash_type"]; !hasType {
			claims["hash_type"] = "sha256"
		}
	}

	return nil
}

// validateClaims performs client-side validation before sending to the API.
func validateClaims(claims map[string]any) error {
	name, _ := claims["name"].(string)
	hash, _ := claims["hash"].(string)
	hashType, _ := claims["hash_type"].(string)

	if name == "" {
		return fmt.Errorf("claims name is required (use --name or provide via file/auto-hash)")
	}
	if hash == "" {
		return fmt.Errorf("claims hash is required (use --hash or provide a file to auto-hash)")
	}
	if hashType == "" {
		return fmt.Errorf("claims hash_type is required")
	}

	// Validate hash is hex
	hash = strings.ToLower(hash)
	claims["hash"] = hash
	if len(hash)%2 != 0 {
		return fmt.Errorf("claims hash must be even-length hex string")
	}
	if _, err := hex.DecodeString(hash); err != nil {
		return fmt.Errorf("claims hash contains invalid hex: %w", err)
	}

	// Validate URL if present
	if urlStr, ok := claims["url"].(string); ok && urlStr != "" {
		if !strings.HasPrefix(urlStr, "https://") {
			return fmt.Errorf("claims url must start with https://")
		}
	}

	return nil
}

// printCreateJSON outputs the creation result as JSON for scripting.
func printCreateJSON(resp *items.CreateItemResponse) error {
	out := map[string]any{
		"id":         resp.ID,
		"name":       resp.Name,
		"hash":       resp.Hash,
		"hash_type":  resp.HashType,
		"visibility": resp.Visibility,
		"team_id":    resp.TeamID,
	}
	if len(resp.Tags) > 0 {
		out["tags"] = resp.Tags
	}
	data, err := json.MarshalIndent(out, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling JSON: %w", err)
	}
	fmt.Println(string(data))
	return nil
}

// presentCreate renders a styled success display.
func presentCreate(resp *items.CreateItemResponse) {
	header := ui.AccentBoldStyle().Render("  Item Created")

	tbl := table.New().
		Border(lipgloss.HiddenBorder()).
		StyleFunc(ui.LabelValueStyleFunc()).
		Row("ID", resp.ID).
		Row("Name", resp.Name)

	if resp.Hash != "" {
		hashDisplay := resp.Hash
		if resp.HashType != "" {
			hashDisplay += " (" + resp.HashType + ")"
		}
		tbl = tbl.Row("Hash", hashDisplay)
	}
	if resp.Visibility != "" {
		tbl = tbl.Row("Visibility", resp.Visibility)
	}
	if len(resp.Tags) > 0 {
		tbl = tbl.Row("Tags", strings.Join(resp.Tags, ", "))
	}
	if resp.TeamID != "" {
		tbl = tbl.Row("Team", resp.TeamID)
	}

	lipgloss.Println(lipgloss.JoinVertical(lipgloss.Left,
		header,
		tbl.String(),
	))
}

func init() {
	f := createCmd.Flags()

	// Input source: file to hash
	f.StringP("file", "f", "", "Path to file to hash (interactive picker if no path given)")
	f.Lookup("file").NoOptDefVal = fileFlagPickCreate
	f.BoolP("file-stdin", "F", false, "Hash raw file content from stdin (requires --name)")

	// Input source: claims JSON
	f.StringP("claims", "c", "", "Path to claims JSON file (interactive picker if no path given)")
	f.Lookup("claims").NoOptDefVal = claimsFlagPick
	f.BoolP("claims-stdin", "C", false, "Read claims JSON from stdin")

	// Claims fields
	f.StringP("name", "n", "", "Item name")
	f.String("hash", "", "Hex hash of the data")
	f.String("hash-type", "sha256", "Hash algorithm (sha256, sha512, md5, etc.)")
	f.StringP("description", "d", "", "Item description")
	f.String("url", "", "HTTPS URL associated with the item")
	f.String("timestamp", "", "ISO 8601 timestamp (e.g. 2025-01-15, 2025-01-15T14:30:00Z)")
	f.String("metadata", "", "JSON string of freeform metadata")
	f.String("location", "", "Geographic location as lat,lon (e.g. 37.7749,-122.4194)")

	// Item attributes
	f.StringP("visibility", "v", "private", `Item visibility: "private", "team", or "public"`)
	f.StringP("tags", "t", "", "Comma-separated tags")

	// Output
	f.Bool("json", false, "Output result as JSON")

	rootCmd.AddCommand(createCmd)
}
