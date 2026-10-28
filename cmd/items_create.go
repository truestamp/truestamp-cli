// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/spf13/cobra"
	"github.com/truestamp/truestamp-cli/internal/inputsrc"
	"github.com/truestamp/truestamp-cli/internal/items"
	"github.com/truestamp/truestamp-cli/internal/jsonapi"
	"github.com/truestamp/truestamp-cli/internal/specdata"
	"github.com/truestamp/truestamp-cli/internal/strictjson"
	"github.com/truestamp/truestamp-cli/internal/ui"
)

var itemsCreateCmd = &cobra.Command{
	Use:   "create [file]",
	Short: "Create a new timestamped item",
	Long: `Create a new cryptographic timestamp item.

Truestamp supports two submission modes:

  External-hash mode (default for files):
    truestamp items create document.pdf
  Computes SHA-256 of the file locally and submits the hash. The file
  itself never leaves your device. Use this for any file you can keep.

  Claims-as-source-of-truth mode:
    truestamp items create -n "Invention" -d "On this day I claim ..."
  No external file. The claims content itself is timestamped. The server
  requires a description of at least 32 characters (or non-empty
  --metadata) so the proof commits to meaningful content.

The claims are sent exactly as given, and the server commits their
canonical form (claims_jcs). A claims file (--claims) or claims on stdin
(--claims-stdin) is sent byte for byte as given: it is read
locally only to refuse a text no verifier reads alike (a member name
repeated in one object, or a number outside the range every verifier
reads exactly), and nothing in it is trimmed, reordered or filled in. The
claim flags (--name, --data-hash, --hash-type, --description, --url,
--timestamp, --metadata, --location) build the claims instead, so they are
refused beside a claims file: put every claim in the file. A number
longer than 1,024 characters is refused too, in any claims: the server
reads no request that holds one, whatever its value.

The server checks every claims rule and reports each one broken with its
code, its JSON pointer into the claims and a message naming the fix.
--data-hash and --hash-type go together: a bare --data-hash sends no
hash_type, and the server refuses the half pair. On success the output
shows claims_jcs, the claims in canonical form (RFC 8785) exactly as the
server keeps them, and claims_hash, the hash computed over them.

Input methods (resolved in priority order):
  truestamp items create document.pdf              External hash: hash file, filename as name
  truestamp items create --file=document.pdf       External hash: explicit file path
  truestamp items create --file                    External hash: interactive file picker
  cat doc.pdf | truestamp items create -F -n Doc   External hash: hash file content from stdin
  truestamp items create -c=claims.json            Either mode: send claims from a JSON file
  truestamp items create --claims                  Interactive claims JSON file picker
  cat claims.json | truestamp items create -C      Send claims JSON from stdin
  truestamp items create -n Doc --data-hash <hex> --hash-type sha256
                                                   External hash: build claims from flags
  truestamp items create -n "Doc" -d "long desc"   Claims-only: timestamp the claims content

Beside a hashed file, --name replaces the file's name and the other claim
flags add their members, enabling combinations like:
  truestamp items create report.pdf -n "Q1 Report" -v public -t finance
--data-hash and --hash-type are refused beside a hashed file: the file
supplies the hash and its hash_type.

Requires authentication, run 'truestamp auth login', or set TRUESTAMP_API_KEY / --api-key for headless/CI use.`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		// Resolve claims from input sources first so `truestamp items create`
		// with no args shows help without requiring an API key.
		claimsText, err := resolveCreateInput(cmd, args)
		if err != nil {
			return err
		}
		if claimsText == nil {
			cmd.Help()
			return nil
		}
		if err := checkNumberLengths(claimsText); err != nil {
			return err
		}

		cfg := appConfig
		if err := requireAuth(cmd); err != nil {
			return err
		}

		jsonOutput, _ := outputMode(cmd)

		visibility, _ := cmd.Flags().GetString("visibility")
		if err := items.ValidateVisibility(visibility); err != nil {
			return err
		}
		rawTags, _ := cmd.Flags().GetStringSlice("tags")
		tags := items.NormalizeTags(rawTags)

		appLogger.Info("create_request",
			"visibility", visibility,
			"tag_count", len(tags),
			"claims_octets", len(claimsText),
		)
		resp, err := items.CreateItemCtx(cmd.Context(), cfg.APIURL, cfg.Team, claimsText, visibility, tags)
		if err != nil {
			appLogger.Error("create_failed", "err", err.Error())
			return explainCreateError(cmd, err)
		}
		appLogger.Info("create_completed", "id", resp.ID)

		if jsonOutput {
			return printCreateJSON(cmd.OutOrStdout(), resp)
		}
		presentCreate(cmd.OutOrStdout(), resp)
		return nil
	},
}

// claimFlags are the flags that build claims, in the order their members
// are written. Each is refused beside a claims file or claims on stdin,
// which are sent exactly as given.
var claimFlags = []string{"name", "data-hash", "hash-type", "description", "url", "timestamp", "location", "metadata"}

// fileHashFlags are the claim flags a hashed file supplies the members of
// itself: its digest and that digest's type, sha256.
var fileHashFlags = []string{"data-hash", "hash-type"}

// resolveCreateInput returns the claims text to send, or nil when there is
// no input (show help).
// Priority: --claims > --claims-stdin > --file > --file-stdin > positional arg > flags-only > help
func resolveCreateInput(cmd *cobra.Command, args []string) ([]byte, error) {
	claimsFlag, _ := cmd.Flags().GetString("claims")
	claimsStdin, _ := cmd.Flags().GetBool("claims-stdin")
	fileFlag, _ := cmd.Flags().GetString("file")
	fileStdin, _ := cmd.Flags().GetBool("file-stdin")

	// `--claims` and `--file` carry a NoOptDefVal, so a space-separated
	// value is not consumed by the flag: `--claims doc.json` sets the
	// picker sentinel and leaves doc.json as a positional argument. Left
	// alone that opens the interactive picker and silently discards the
	// path, which hangs a script and misleads a human.
	//
	// Reusing the positional is not an option here the way it is for the
	// shared resolver: for `create` a positional means "hash this file",
	// so honoring `--claims doc.json` as a positional would quietly
	// timestamp the file instead of reading claims out of it. The two
	// spellings mean different things, so the only safe move is to stop
	// and name the fix.
	if len(args) > 0 {
		if claimsFlag == inputsrc.FilePickSentinel {
			return nil, fmt.Errorf("--claims takes its path with an equals sign: did you mean --claims=%s ? (a space-separated path is read as a file to hash, not as claims)", args[0])
		}
		if fileFlag == inputsrc.FilePickSentinel {
			return nil, fmt.Errorf("--file takes its path with an equals sign: did you mean --file=%s ? (or drop the flag: `truestamp items create %s` hashes it directly)", args[0], args[0])
		}
	}

	switch {
	case claimsFlag != "" || claimsStdin:
		if err := refuseClaimFlags(cmd); err != nil {
			return nil, err
		}
	case fileFlag != "" || fileStdin || len(args) > 0:
		if err := refuseHashFlags(cmd); err != nil {
			return nil, err
		}
	}
	withFlags := func(b *claimsBuilder, err error) ([]byte, error) {
		return claimsFromFlags(cmd, b, err)
	}

	switch {
	// --claims: send the claims JSON of a file (picker if no path)
	case claimsFlag == inputsrc.FilePickSentinel:
		path, err := pickFile("Select claims JSON file", ".json")
		if err != nil {
			return nil, err
		}
		return readClaimsFile(path)

	case claimsFlag != "":
		return readClaimsFile(claimsFlag)

	// --claims-stdin: send the claims JSON read from stdin
	case claimsStdin:
		return readClaimsStdin()

	// --file: auto-hash a file (picker if no path)
	case fileFlag == inputsrc.FilePickSentinel:
		path, err := pickFile("Select file to hash")
		if err != nil {
			return nil, err
		}
		return withFlags(autoHashFile(path))

	case fileFlag != "":
		return withFlags(autoHashFileChecked(fileFlag))

	// --file-stdin: hash raw file content from stdin (requires --name)
	case fileStdin:
		return withFlags(autoHashStdin())

	// Positional arg: auto-hash file
	case len(args) > 0:
		return withFlags(autoHashFileChecked(args[0]))

	// Flag-only mode: build claims from the claim flags
	default:
		for _, flag := range claimFlags {
			if cmd.Flags().Changed(flag) {
				return withFlags(&claimsBuilder{}, nil)
			}
		}
		return nil, nil // no input = show help
	}
}

// refuseClaimFlags refuses every claim flag given beside a claims file or
// claims on stdin: those are sent exactly as given, and a flag would have
// to rewrite them.
func refuseClaimFlags(cmd *cobra.Command) error {
	var given []string
	for _, flag := range claimFlags {
		if cmd.Flags().Changed(flag) {
			given = append(given, "--"+flag)
		}
	}
	if len(given) == 0 {
		return nil
	}
	return fmt.Errorf("%s cannot be combined with --claims or --claims-stdin: the claims are sent exactly as given, so put every claim in the claims JSON itself",
		strings.Join(given, ", "))
}

// refuseHashFlags refuses --data-hash and --hash-type beside a hashed file
// (a file argument, --file or --file-stdin): the file supplies the hash
// and its type, and either flag would pair one with a value that does not
// describe it.
func refuseHashFlags(cmd *cobra.Command) error {
	var given []string
	for _, flag := range fileHashFlags {
		if cmd.Flags().Changed(flag) {
			given = append(given, "--"+flag)
		}
	}
	if len(given) == 0 {
		return nil
	}
	return fmt.Errorf("%s cannot be combined with a hashed file (a file argument, --file or --file-stdin): the file supplies the hash and its hash_type, sha256",
		strings.Join(given, ", "))
}

// checkNumberLengths refuses claims holding a number literal longer than
// items.MaxNumberOctets, before anything is sent. That is a limit of the
// service's request reader on what a request may carry, not a rule about
// the value: the service refuses the whole request with a bare 400.
func checkNumberLengths(claims []byte) error {
	offset, length, found := items.LongNumber(claims)
	if !found {
		return nil
	}
	return fmt.Errorf("the claims hold a number of %d characters at octet %d: the Truestamp service reads no request "+
		"holding a number longer than %d characters (its request reader, TruestampWeb.RequestJSON, holds every body "+
		"to JSONNumbers.max_octets), a limit on what can be sent, not a rule about the value; send it as a string",
		length, offset, items.MaxNumberOctets)
}

// claimsMember is one member of claims built from flags: its name and its
// value as JSON text.
type claimsMember struct {
	name  string
	value []byte
}

// claimsBuilder writes claims as a JSON text from members whose values are
// already JSON text, in the order they are first set. Nothing is trimmed,
// normalized or filled in: each value is the text the user gave.
type claimsBuilder struct {
	members []claimsMember
}

// set gives name the value text, replacing an earlier value in its place.
func (b *claimsBuilder) set(name string, value []byte) {
	for i := range b.members {
		if b.members[i].name == name {
			b.members[i].value = value
			return
		}
	}
	b.members = append(b.members, claimsMember{name: name, value: value})
}

// setString gives name the JSON string of s, which must be valid UTF-8 for
// a JSON text to hold it unchanged. flag names the flag s came from.
func (b *claimsBuilder) setString(name, s, flag string) error {
	if !utf8.ValidString(s) {
		return fmt.Errorf("%s is not valid UTF-8, so no JSON text can hold it as given", flag)
	}
	b.set(name, items.ClaimsJSONString(s))
	return nil
}

// text is the claims object as a JSON text.
func (b *claimsBuilder) text() []byte {
	var out bytes.Buffer
	out.WriteByte('{')
	for i, m := range b.members {
		if i > 0 {
			out.WriteByte(',')
		}
		out.Write(items.ClaimsJSONString(m.name))
		out.WriteByte(':')
		out.Write(m.value)
	}
	out.WriteByte('}')
	return out.Bytes()
}

// claimsFromFlags sets each claim flag given onto b, a later flag's
// member replacing one b already holds (a hashed file's name, say), and
// returns the claims text. err is the error that built b, if any.
func claimsFromFlags(cmd *cobra.Command, b *claimsBuilder, err error) ([]byte, error) {
	if err != nil {
		return nil, err
	}
	stringMembers := []struct{ flag, member string }{
		{"name", "name"},
		{"data-hash", "hash"},
		{"hash-type", "hash_type"},
		{"description", "description"},
		{"url", "url"},
	}
	for _, sm := range stringMembers {
		if !cmd.Flags().Changed(sm.flag) {
			continue
		}
		v, _ := cmd.Flags().GetString(sm.flag)
		if err := b.setString(sm.member, v, "--"+sm.flag); err != nil {
			return nil, err
		}
	}

	if cmd.Flags().Changed("timestamp") {
		raw, _ := cmd.Flags().GetString("timestamp")
		if err := checkTimestamp(raw); err != nil {
			return nil, err
		}
		if err := b.setString("timestamp", raw, "--timestamp"); err != nil {
			return nil, err
		}
	}

	// Location: "lat,lon", each a JSON number written into the claims as
	// typed.
	if cmd.Flags().Changed("location") {
		locStr, _ := cmd.Flags().GetString("location")
		lat, lon, ok := strings.Cut(locStr, ",")
		if !ok {
			return nil, fmt.Errorf("--location must be lat,lon (e.g. 37.7749,-122.4194)")
		}
		latText, err := jsonNumberText(lat, "--location latitude")
		if err != nil {
			return nil, err
		}
		lonText, err := jsonNumberText(lon, "--location longitude")
		if err != nil {
			return nil, err
		}
		b.set("location", []byte(`{"latitude":`+latText+`,"longitude":`+lonText+`}`))
	}

	// Metadata: a JSON text read with the strict reader and written into
	// the claims as typed.
	if cmd.Flags().Changed("metadata") {
		metaStr, _ := cmd.Flags().GetString("metadata")
		if _, err := strictjson.Decode([]byte(metaStr)); err != nil {
			return nil, fmt.Errorf("--metadata must be valid JSON: %s", describeStrictRejection(err))
		}
		b.set("metadata", []byte(metaStr))
	}

	return b.text(), nil
}

// jsonNumberText returns s when it is one JSON number the strict reader
// reads, which the claims then carry as typed.
func jsonNumberText(s, what string) (string, error) {
	v, err := strictjson.Decode([]byte(s))
	if err != nil {
		return "", fmt.Errorf("%s: %s", what, describeStrictRejection(err))
	}
	if _, ok := v.(strictjson.Number); !ok {
		return "", fmt.Errorf("%s must be a JSON number, got %q", what, s)
	}
	return s, nil
}

// autoHashFileChecked validates the path then auto-hashes.
func autoHashFileChecked(path string) (*claimsBuilder, error) {
	info, err := os.Stat(path)
	if err != nil {
		return nil, fmt.Errorf("cannot access %q: %w", path, err)
	}
	if info.IsDir() {
		return nil, fmt.Errorf("%q is a directory, expected a file", path)
	}
	return autoHashFile(path)
}

// autoHashStdin reads raw content from stdin and computes its SHA-256.
// The caller provides --name since there is no filename to derive it from.
func autoHashStdin() (*claimsBuilder, error) {
	data, err := io.ReadAll(io.LimitReader(os.Stdin, 100<<20)) // 100 MB limit
	if err != nil {
		return nil, fmt.Errorf("reading stdin: %w", err)
	}
	if len(data) == 0 {
		return nil, fmt.Errorf("no data received on stdin")
	}

	h := sha256.Sum256(data)
	b := &claimsBuilder{}
	b.set("hash", items.ClaimsJSONString(hex.EncodeToString(h[:])))
	b.set("hash_type", items.ClaimsJSONString("sha256"))
	return b, nil
}

// autoHashFile computes the SHA-256 of a file and starts claims naming it
// by its base name.
func autoHashFile(path string) (*claimsBuilder, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("opening file: %w", err)
	}
	defer f.Close()

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return nil, fmt.Errorf("hashing file: %w", err)
	}

	b := &claimsBuilder{}
	b.set("hash", items.ClaimsJSONString(hex.EncodeToString(h.Sum(nil))))
	b.set("hash_type", items.ClaimsJSONString("sha256"))
	if err := b.setString("name", filepath.Base(path), "the file name"); err != nil {
		return nil, err
	}
	return b, nil
}

// checkClaimsText parses data, already read under J1, with the strict
// reader (SPEC.md section 10.1, J2 to J6), so a text no two verifiers read
// alike, such as one that repeats a member name or holds a number in J5's
// integer gap, is refused before anything is sent. The text itself is what
// is sent; the value read is discarded.
func checkClaimsText(data []byte) error {
	_, err := strictjson.Parse(data)
	return err
}

// describeStrictRejection names a strict reader's rejection and, for the
// rules a hand-written JSON text most often breaks, what to do.
func describeStrictRejection(err error) string {
	var rej *strictjson.Rejection
	if !errors.As(err, &rej) {
		return err.Error()
	}
	switch rej.SubReason {
	case specdata.SubReasonDuplicateName:
		return rej.Error() + ": a member name appears twice in one object; keep one"
	case specdata.SubReasonNumber:
		return rej.Error() + fmt.Sprintf(": a number is outside what every verifier reads exactly "+
			"(an integer of magnitude above %d and below 10^21, or one no binary64 value holds); send it as a string",
			strictjson.MaxInteger)
	case specdata.SubReasonDepth:
		return rej.Error() + fmt.Sprintf(": the JSON is nested deeper than %d levels", strictjson.MaxDepth)
	}
	return rej.Error()
}

// readClaimsFile reads a claims file with the strict reader, which reads
// at most MAX_READ_OCTETS + 1 octets (J1), and checks it.
func readClaimsFile(path string) ([]byte, error) {
	data, err := strictjson.ReadFile(path)
	if err != nil {
		var pathErr *fs.PathError
		if errors.As(err, &pathErr) {
			return nil, fmt.Errorf("reading claims file: %w", pathErr)
		}
		return nil, fmt.Errorf("reading claims file: %s", describeStrictRejection(err))
	}
	if err := checkClaimsText(data); err != nil {
		return nil, fmt.Errorf("parsing claims JSON: %s", describeStrictRejection(err))
	}
	return data, nil
}

// readClaimsStdin reads claims from stdin and checks them with the strict
// reader.
func readClaimsStdin() ([]byte, error) {
	data, err := strictjson.Read(os.Stdin)
	if err != nil {
		return nil, fmt.Errorf("reading stdin: %s", describeStrictRejection(err))
	}
	if len(data) == 0 {
		return nil, fmt.Errorf("no data received on stdin")
	}
	if err := checkClaimsText(data); err != nil {
		return nil, fmt.Errorf("parsing claims JSON from stdin: %s", describeStrictRejection(err))
	}
	return data, nil
}

// errNoTerminalForPicker is returned instead of opening a file picker when
// stdin is not a terminal. Without this guard the bubbletea picker starts,
// finds nothing to read, and blocks forever under any pty-allocating harness
// (CI, `docker -t`, an agent's shell). inputsrc guards its own prompts the
// same way; cmd/upgrade.go documents the underlying trap.
var errNoTerminalForPicker = errors.New(
	"interactive file picker requires a terminal: pass --file=<path> or --claims=<path>")

// pickFile launches an interactive file picker, refusing first when stdin
// is not a terminal (see errNoTerminalForPicker). exts restricts the
// picker to those extensions; none means any file.
func pickFile(title string, exts ...string) (string, error) {
	if !inputsrc.IsStdinTerminal() {
		return "", errNoTerminalForPicker
	}
	return ui.PickFile(ui.PickFileOptions{Title: title, AllowedTypes: exts})
}

// claimedTimestamp is the grammar the server accepts for claims.timestamp:
// YYYY, YYYY-MM, YYYY-MM-DD, or YYYY-MM-DDThh:mm with optional :ss and a
// fraction of 1 to 9 digits, and an optional Z, +hh:mm or -hh:mm offset
// (the colon optional).
var claimedTimestamp = regexp.MustCompile(`\A([0-9]{4})(?:-(0[1-9]|1[0-2])(?:-(0[1-9]|[12][0-9]|3[01])(?:T(?:[01][0-9]|2[0-3]):[0-5][0-9](?::[0-5][0-9](?:\.[0-9]{1,9})?)?(?:Z|[+-](?:[01][0-9]|2[0-3]):?[0-5][0-9])?)?)?)?\z`)

// checkTimestamp checks raw against the claimed-timestamp grammar and, when
// it names a day, that the day exists. raw is then sent exactly as typed:
// a claimed timestamp is the submitter's own statement, never parsed,
// converted or normalized.
func checkTimestamp(raw string) error {
	m := claimedTimestamp.FindStringSubmatch(raw)
	if m == nil {
		return fmt.Errorf("--timestamp %q is not a date or time in an accepted form: YYYY, YYYY-MM, "+
			"YYYY-MM-DD, or YYYY-MM-DDThh:mm with optional :ss, an optional fraction of 1 to 9 "+
			"digits, and an optional Z, +hh:mm or -hh:mm offset", raw)
	}
	if m[3] == "" {
		return nil
	}
	year, _ := strconv.Atoi(m[1])
	month, _ := strconv.Atoi(m[2])
	day, _ := strconv.Atoi(m[3])
	if t := time.Date(year, time.Month(month), day, 0, 0, 0, 0, time.UTC); t.Day() != day {
		return fmt.Errorf("--timestamp names %s-%s-%s, a date that does not exist", m[1], m[2], m[3])
	}
	return nil
}

// explainCreateError renders the two refusals of a create the CLI knows
// how to explain, and returns any other error unchanged.
//
// A claims refusal lists every rule the server found broken, each with
// its JSON pointer into the claims, its code and its message, which names
// the fix. item_code_unavailable means nothing was created and the same
// request may be sent again. Under --json each is one object on stdout;
// under --silent nothing is printed.
func explainCreateError(cmd *cobra.Command, err error) error {
	jsonOutput, silent := outputMode(cmd)

	var refused *items.ClaimsRefusedError
	if errors.As(err, &refused) {
		if jsonOutput {
			out := map[string]any{
				"error":          "claims_refused",
				"message":        "the server refused the claims",
				"claims_reasons": refused.Reasons,
			}
			if len(refused.Others) > 0 {
				out["other_errors"] = refused.Others
			}
			if err := emitJSON(cmd.OutOrStdout(), out); err != nil {
				return err
			}
			return errSilentFail
		}
		if silent {
			return errSilentFail
		}
		w := cmd.ErrOrStderr()
		ui.Fprintln(w, ui.FailureBanner("Claims refused"))
		for _, r := range refused.Reasons {
			ui.Fprintln(w, ui.FaintStyle().Render(fmt.Sprintf("    pointer %s, %s: %s", strconv.Quote(r.Pointer), r.Code, r.Message)))
		}
		for _, o := range refused.Others {
			ui.Fprintln(w, ui.FaintStyle().Render("    "+o))
		}
		ui.Fprintln(w, ui.FaintStyle().Render(fmt.Sprintf("    Nothing was created. The server answered HTTP %d.", refused.Status)))
		return errSilentFail
	}

	if errors.Is(err, items.ErrItemCodeUnavailable) {
		detail := "Could not issue an item code."
		var apiErr *jsonapi.APIError
		if errors.As(err, &apiErr) && apiErr.Detail != "" {
			detail = apiErr.Detail
		}
		if jsonOutput {
			if err := emitJSON(cmd.OutOrStdout(), map[string]any{
				"error":     items.CodeItemCodeUnavailable,
				"message":   detail,
				"retryable": true,
			}); err != nil {
				return err
			}
			return errSilentFail
		}
		if silent {
			return errSilentFail
		}
		w := cmd.ErrOrStderr()
		ui.Fprintln(w, ui.FailureBanner("Item not created yet"))
		for _, l := range []string{
			detail,
			"Nothing was created. This clears on its own: run the same command again.",
			"The server said " + items.CodeItemCodeUnavailable + " (HTTP 409).",
		} {
			ui.Fprintln(w, ui.FaintStyle().Render("    "+l))
		}
		return errSilentFail
	}

	return err
}

// printCreateJSON outputs the creation result as JSON for scripting.
// item_hash and item_code are given when the response carries them, as
// the styled table presenter gives them. hash and hash_type are omitted
// when the response carries no external hash (claims-as-source-of-truth
// mode), matching the verify JSON output convention and the styled table
// presenter.
func printCreateJSON(w io.Writer, resp *items.CreateItemResponse) error {
	out := map[string]any{
		"id":          resp.ID,
		"name":        resp.Name,
		"visibility":  resp.Visibility,
		"team_id":     resp.TeamID,
		"claims_jcs":  resp.ClaimsJCS,
		"claims_hash": resp.ClaimsHash,
	}
	if resp.ItemHash != "" {
		out["item_hash"] = resp.ItemHash
	}
	if resp.ItemCode != "" {
		out["item_code"] = resp.ItemCode
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
	return emitJSON(w, out)
}

// presentCreate renders a styled success display.
func presentCreate(w io.Writer, resp *items.CreateItemResponse) {
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
	tbl = tbl.Row("Claims JCS", resp.ClaimsJCS).
		Row("Claims hash", resp.ClaimsHash)
	if resp.ItemHash != "" {
		tbl = tbl.Row("Item hash", resp.ItemHash)
	}
	if resp.ItemCode != "" {
		tbl = tbl.Row("Item code", resp.ItemCode)
	}

	// Shareable public-web links as rows of the SAME table, they share
	// the right-aligned-label / value column alignment, each named by the
	// item code the server issued, the form the pages canonicalize to.
	// Note the verify link will render "not yet committed" until the item
	// lands in a finalized block, still useful to surface the stable URL
	// so the user can come back later.
	if detail := ui.SubjectDetailURL(appConfig.APIURL, "item", resp.ItemCode); detail != "" {
		tbl = tbl.Row("Details", detail)
	}
	if verify := ui.SubjectVerifyURL(appConfig.APIURL, "item", resp.ItemCode); verify != "" {
		tbl = tbl.Row("Verify", verify)
	}

	// Plain newline-join, see note in internal/verify/presenter.go
	// Present(). lipgloss.JoinVertical pad-to-widest can cause phantom
	// blank lines after every table row on narrow terminals.
	ui.Fprintln(w, strings.Join([]string{header, tbl.String()}, "\n"))
}

func init() {
	f := itemsCreateCmd.Flags()

	// Input source: file to hash
	f.StringP("file", "f", "", "Path to file to hash (interactive picker if no path given)")
	f.Lookup("file").NoOptDefVal = inputsrc.FilePickSentinel
	f.BoolP("file-stdin", "F", false, "Hash raw file content from stdin (requires --name)")

	// Input source: claims JSON
	f.StringP("claims", "c", "", "Path to claims JSON file, sent exactly as given (interactive picker if no path given)")
	f.Lookup("claims").NoOptDefVal = inputsrc.FilePickSentinel
	f.BoolP("claims-stdin", "C", false, "Read claims JSON from stdin, sent exactly as given")

	// Claims fields
	f.StringP("name", "n", "", "Item name")
	f.String("data-hash", "", "Hex hash of the data (with --hash-type)")
	f.String("hash-type", "", "Hash algorithm of --data-hash (sha256, sha512, md5, etc.)")
	f.StringP("description", "d", "", "Item description")
	f.String("url", "", "HTTPS URL associated with the item")
	f.String("timestamp", "", "claimed timestamp: your own statement; Truestamp checks only that it is a well-formed date")
	f.String("metadata", "", "JSON string of freeform metadata")
	f.String("location", "", "Geographic location as lat,lon (e.g. 37.7749,-122.4194)")

	// Item attributes
	f.StringP("visibility", "v", "private", `Item visibility: "private", "team", or "public"`)
	f.StringSlice("tags", nil, "Tags to attach (comma-separated or repeated)")

	// Output
	addRecordOutputFlags(itemsCreateCmd)

	itemsCmd.AddCommand(itemsCreateCmd)
}
