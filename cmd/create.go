// Copyright (c) 2019-2026 Truestamp, Inc.
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
	"github.com/spf13/cobra"
	"github.com/truestamp/truestamp-cli/internal/inputsrc"
	"github.com/truestamp/truestamp-cli/internal/items"
	"github.com/truestamp/truestamp-cli/internal/tscrypto"
	"github.com/truestamp/truestamp-cli/internal/ui"
)

// claimsOnlyMinDescription is the minimum length (in non-whitespace
// characters, after trimming) of claims.description for an item submitted
// without an external hash. Mirrors the threshold enforced server-side in
// lib/truestamp/items/validations/validate_claims_only_content.ex — do not
// drift from this value without coordinating with the backend.
const claimsOnlyMinDescription = 32

var createCmd = &cobra.Command{
	Use:   "create [file]",
	Short: "Create a new Truestamp item",
	Long: `Create a new cryptographic timestamp item.

Truestamp supports two submission modes:

  External-hash mode (default for files):
    truestamp create document.pdf
  Computes SHA-256 of the file locally and submits the hash. The file
  itself never leaves your device. Use this for any file you can keep.

  Claims-as-source-of-truth mode:
    truestamp create -n "Invention" -d "On this day I claim ..."
  No external file. The claims content itself is timestamped. Requires
  at least a 32-character description (or non-empty --metadata) so the
  proof commits to meaningful content.

Submitting exactly one of --hash / --hash-type is rejected; the pair
must be both supplied (external-hash mode) or both omitted
(claims-as-source-of-truth mode).

Input methods (resolved in priority order):
  truestamp create document.pdf              External hash: hash file, filename as name
  truestamp create --file document.pdf       External hash: explicit file path
  truestamp create --file                    External hash: interactive file picker
  cat doc.pdf | truestamp create -F -n Doc   External hash: hash file content from stdin
  truestamp create -c claims.json            Either mode: load claims from JSON file
  truestamp create --claims                  Interactive claims JSON file picker
  cat claims.json | truestamp create -C      Read claims JSON from stdin
  truestamp create -n "Doc" --hash abc...    External hash: build claims from flags
  truestamp create -n "Doc" -d "long desc"   Claims-only: timestamp the claims content

Flags override values from file/auto-hash, enabling combinations like:
  truestamp create report.pdf -n "Q1 Report" -v public -t finance

Requires authentication — run 'truestamp auth login', or set TRUESTAMP_API_KEY / --api-key for headless/CI use.`,
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
		if !authConfigured() {
			return fmt.Errorf("not authenticated — run `truestamp auth login`, or set TRUESTAMP_API_KEY / --api-key for headless use")
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
		appLogger.Info("create_request",
			"visibility", visibility,
			"tag_count", len(tags),
			"claim_keys", len(claims),
		)
		resp, err := items.CreateItemCtx(cmd.Context(), cfg.APIURL, cfg.Team, claims, visibility, tags)
		if err != nil {
			appLogger.Error("create_failed", "err", err.Error())
			return err
		}
		appLogger.Info("create_completed", "id", resp.ID)

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
	case claimsFlag == inputsrc.FilePickSentinel:
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
	case fileFlag == inputsrc.FilePickSentinel:
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
//
// Two submission modes are supported, mirroring the server contract:
//
//   - External-hash mode: claims.hash + claims.hash_type are both present.
//     Hash must be valid hex of the correct length for hash_type.
//   - Claims-as-source-of-truth mode: claims.hash + claims.hash_type are
//     both absent. The claims content itself is what gets timestamped.
//     To prevent meaningless empty submissions, either claims.description
//     must be >= claimsOnlyMinDescription non-whitespace characters or
//     claims.metadata must be a non-empty object.
//
// Submitting exactly one of hash / hash_type is rejected (co-required pair).
// Whitespace-only strings are treated as absent, matching the server.
func validateClaims(claims map[string]any) error {
	name, _ := claims["name"].(string)
	name = strings.TrimSpace(name)
	if name == "" {
		return fmt.Errorf("claims name is required (use --name or provide via file/auto-hash)")
	}
	claims["name"] = name

	rawHash, _ := claims["hash"].(string)
	rawHashType, _ := claims["hash_type"].(string)
	hash := strings.TrimSpace(rawHash)
	hashType := strings.TrimSpace(rawHashType)

	// Treat whitespace-only as absent. Strip the keys so the outgoing
	// payload doesn't carry empty strings — the server treats empty as
	// nil but explicit omission is cleaner over the wire.
	if hash == "" {
		delete(claims, "hash")
	}
	if hashType == "" {
		delete(claims, "hash_type")
	}

	switch {
	// External-hash mode: both supplied.
	case hash != "" && hashType != "":
		hash = strings.ToLower(hash)
		claims["hash"] = hash
		if len(hash)%2 != 0 {
			return fmt.Errorf("claims hash must be even-length hex string")
		}
		if _, err := hex.DecodeString(hash); err != nil {
			return fmt.Errorf("claims hash contains invalid hex: %w", err)
		}
		// Same length-vs-algorithm check the verify path uses, so the two
		// modes share the same hash-shape rule and "sha256 but 32 hex
		// chars" is caught locally instead of at the server.
		if err := tscrypto.ValidateClaimsHash(hash, hashType); err != nil {
			return fmt.Errorf("claims hash: %w", err)
		}

	// Co-required violations: exactly one of the pair is supplied.
	case hash != "" && hashType == "":
		return fmt.Errorf("claims hash_type is required when hash is supplied")
	case hash == "" && hashType != "":
		return fmt.Errorf("claims hash is required when hash_type is supplied")

	// Claims-as-source-of-truth mode: both absent. Mirror the server's
	// "meaningful content" rule locally so users get inline feedback
	// instead of a round-trip for an obvious mistake.
	default:
		if !hasMeaningfulClaimsContent(claims) {
			return fmt.Errorf(
				"claims content is required: provide --description of at least %d characters or non-empty --metadata "+
					"(or pass --hash + --hash-type to use external-hash mode)",
				claimsOnlyMinDescription)
		}
	}

	// Validate URL if present (applies to both modes).
	if urlStr, ok := claims["url"].(string); ok && urlStr != "" {
		if !strings.HasPrefix(urlStr, "https://") {
			return fmt.Errorf("claims url must start with https://")
		}
	}

	return nil
}

// hasMeaningfulClaimsContent reports whether a claims-only submission
// carries enough content to be useful. Either description (>=
// claimsOnlyMinDescription non-whitespace characters after trimming) or
// non-empty metadata is required. Mirrors the server-side validator.
func hasMeaningfulClaimsContent(claims map[string]any) bool {
	if desc, _ := claims["description"].(string); strings.TrimSpace(desc) != "" {
		if len(strings.TrimSpace(desc)) >= claimsOnlyMinDescription {
			return true
		}
	}
	if meta, ok := claims["metadata"].(map[string]any); ok && len(meta) > 0 {
		return true
	}
	return false
}

// printCreateJSON outputs the creation result as JSON for scripting.
// hash and hash_type are omitted when the response carries no external
// hash (claims-as-source-of-truth mode), matching the verify JSON output
// convention and the styled table presenter.
func printCreateJSON(resp *items.CreateItemResponse) error {
	out := map[string]any{
		"id":         resp.ID,
		"name":       resp.Name,
		"visibility": resp.Visibility,
		"team_id":    resp.TeamID,
	}
	if resp.Hash != "" {
		out["hash"] = resp.Hash
	}
	if resp.HashType != "" {
		out["hash_type"] = resp.HashType
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

	tbl := ui.CompactTable().
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

	// Shareable public-web links as rows of the SAME table — they share
	// the right-aligned-label / value column alignment. Note the verify
	// link will render "not yet committed" until the item lands in a
	// finalized block — still useful to surface the stable URL format
	// so the user can come back later.
	if detail := ui.SubjectDetailURL(appConfig.APIURL, "item", resp.ID); detail != "" {
		tbl = tbl.Row("Details", detail)
	}
	if verify := ui.SubjectVerifyURL(appConfig.APIURL, "item", resp.ID); verify != "" {
		tbl = tbl.Row("Verify", verify)
	}

	// Plain newline-join — see note in internal/verify/presenter.go
	// Present(). lipgloss.JoinVertical pad-to-widest can cause phantom
	// blank lines after every table row on narrow terminals.
	lipgloss.Println(strings.Join([]string{header, tbl.String()}, "\n"))
}

func init() {
	f := createCmd.Flags()

	// Input source: file to hash
	f.StringP("file", "f", "", "Path to file to hash (interactive picker if no path given)")
	f.Lookup("file").NoOptDefVal = inputsrc.FilePickSentinel
	f.BoolP("file-stdin", "F", false, "Hash raw file content from stdin (requires --name)")

	// Input source: claims JSON
	f.StringP("claims", "c", "", "Path to claims JSON file (interactive picker if no path given)")
	f.Lookup("claims").NoOptDefVal = inputsrc.FilePickSentinel
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
