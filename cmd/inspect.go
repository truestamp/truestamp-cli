// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
	"github.com/truestamp/truestamp-cli/internal/inputsrc"
	"github.com/truestamp/truestamp-cli/internal/specdata"
	"github.com/truestamp/truestamp-cli/internal/strictjson"
	"github.com/truestamp/truestamp-cli/internal/ui"
	"github.com/truestamp/truestamp-cli/internal/verifier"
)

var inspectCmd = &cobra.Command{
	Use:   "inspect [file-or-url]",
	Short: "Print what a proof carries, without verifying it",
	Long: `Read a Truestamp proof, one JSON document, through the reader and the
SHAPE checks 'truestamp verify' uses (SPEC.md section 10.2, steps 1 to 3
and 7), and print every member it carries, each by its value class
(SPEC.md section 3.7):

  - a value a verification recomputes (CHECKED) as carried, beside the
    value recomputed from what the proof carries, as the checks of
    DERIVE recompute it; a handover_signature is given as the carried
    signature beside the signed message, the hex of the 32 octets it
    signs, which is not a value to compare with it;
  - a value the signer records (RECORDED), and every time the proof
    carries, only under "Declared by the signer (not attributed to
    Truestamp)", never as a fact;
  - the claims as their member names only: the submitter's statement,
    never checked as true;
  - every other member as carried.

It also prints proof_hash and the key id of the signer's key, and, for
an item proof, its item code (SPEC.md section 3.9), computed from the
recomputed item_hash and printed beside it: a name for the item, not a
check. Every value is printed whole, in the text and in --json, a
raw_transaction and a txoutproof among them.

Nothing is verified: the key and the signature are not checked, no chain
or source is read, and no verdict is given. Use 'truestamp verify' for
that. A document the reader refuses is reported with the verifier's bare
rejection, and one SHAPE refuses with SHAPE's row. verify checks the
environment marker, the key and the signature before SHAPE, so it may
refuse the same file at an earlier step, with another code.

Examples:
  truestamp inspect proof.json
  truestamp inspect proof.json --json
  cat proof.json | truestamp inspect

Exit code 0 when the proof passes SHAPE, 1 when it is refused or cannot
be read.`,
	Args: cobra.MaximumNArgs(1),
	RunE: runInspect,
}

func init() {
	f := inspectCmd.Flags()
	f.String("file", "", "Path to proof file (interactive picker if no path given)")
	f.String("url", "", "URL to download proof from (interactive prompt if no URL given)")
	f.Lookup("file").NoOptDefVal = inputsrc.FilePickSentinel
	f.Lookup("url").NoOptDefVal = inputsrc.URLPromptSentinel
	addRecordOutputFlags(inspectCmd)
	inspectCmd.GroupID = groupVerification
	rootCmd.AddCommand(inspectCmd)
}

// runInspect is `truestamp inspect`: it reads at most MAX_READ_OCTETS + 1
// octets of the proof, as verify does, has internal/verifier read them
// (verifier.Inspect), and writes what that gave.
func runInspect(cmd *cobra.Command, args []string) error {
	// inspect renders a record, like verify beside it, so it reads the
	// resolved CLI-wide json / silent settings rather than the raw flags.
	jsonOut, silent := outputMode(cmd)
	fail := func(err error) error {
		if silent {
			return errSilentFail
		}
		return err
	}
	positional := ""
	if len(args) > 0 {
		positional = args[0]
	}
	fileFlag, _ := cmd.Flags().GetString("file")
	urlFlag, _ := cmd.Flags().GetString("url")
	stream, src, err := inputsrc.ResolveStream(cmd.Context(), inputsrc.Options{
		PositionalArg:        positional,
		FileFlag:             fileFlag,
		URLFlag:              urlFlag,
		AllowStdin:           true,
		AutoDetectURL:        true,
		PickerTitle:          "Select proof file",
		PickerExts:           proofPickerExts,
		URLPromptTitle:       "Enter proof URL",
		URLPromptPlaceholder: "https://example.com/proof.json",
		ReadAtMost:           strictjson.MaxReadOctets + 1,
	})
	if err != nil {
		if errors.Is(err, inputsrc.ErrNoInput) {
			_ = cmd.Help()
			return nil
		}
		return fail(err)
	}
	defer func() { _ = stream.Close() }()
	data, err := io.ReadAll(io.LimitReader(stream, strictjson.MaxReadOctets+1))
	if err != nil {
		return fail(fmt.Errorf("the proof cannot be read: %w", err))
	}
	inspection, err := verifier.Inspect(data)
	if err != nil {
		return fail(err)
	}
	out := inspectOutputOf(inspection, src.DisplayName(), len(data))
	switch {
	case silent:
	case jsonOut:
		if err := emitJSON(cmd.OutOrStdout(), out); err != nil {
			return err
		}
	default:
		if _, err := ui.Fprint(cmd.OutOrStdout(), renderInspect(out)); err != nil {
			return err
		}
	}
	if out.Rejection != nil {
		return errSilentFail
	}
	return nil
}

// InspectOutput is what `truestamp inspect` writes: the --json shape, and
// the one source of the text. Octets is the size read, and null for an
// input over MAX_READ_OCTETS, which is read only one octet past that
// bound. Rejection is set exactly when the reader or SHAPE refused the
// document; Signature, KeyID and ProofHash once ENVELOPE passed; Proof
// once SHAPE passed, and Recomputed, DeclaredTimes and Recorded, empty
// before, with it. Each member of the proof is given once, by its value
// class (SPEC.md section 3.7): a CHECKED value in Recomputed, a time in
// DeclaredTimes, a RECORDED value in Recorded, and every other in Proof.
// ItemCode is no member of the proof: it is the item code of SPEC.md
// section 3.9 (The item code), computed from the item_hash recomputed from
// an item proof's record, its Recomputed row at /subject/item_hash, and
// null for any other proof, before SHAPE passes, and when that item_hash
// cannot be recomputed. A match of codes shows only that two item_hash
// values begin with the same 75 bits.
type InspectOutput struct {
	Source        string                `json:"source"`
	Octets        *int                  `json:"octets"`
	Rejection     *InspectRejection     `json:"rejection"`
	Signature     *InspectSignature     `json:"signature"`
	KeyID         *string               `json:"key_id"`
	ProofHash     *string               `json:"proof_hash"`
	Proof         *InspectProof         `json:"proof"`
	Recomputed    []InspectRecomputed   `json:"recomputed"`
	DeclaredTimes []InspectDeclaredTime `json:"declared_times"`
	Recorded      []InspectRecorded     `json:"recorded"`
	ItemCode      *string               `json:"item_code"`
}

// InspectRejection is the verifier's refusal: a bare rejection of READ,
// PARSE or ENVELOPE (SPEC.md section 10.2, the gate), with its sub_reason
// where it has one and part null, or SHAPE's failing row, with its part.
type InspectRejection struct {
	Code      string  `json:"code"`
	Message   string  `json:"message"`
	Part      *string `json:"part"`
	Step      string  `json:"step"`
	SubReason string  `json:"sub_reason,omitempty"`
}

// InspectSignature is the envelope's signature member, as carried.
type InspectSignature struct {
	Alg       string `json:"alg"`
	PublicKey string `json:"public_key"`
	Value     string `json:"value"`
}

// InspectProof is what the proof member carries but its CHECKED values,
// its times and its RECORDED values, each member as the proof holds it
// (verifier.Inspection's Carried), with two in another form: the claims as
// claims_members in the subject, their names, which are the submitter's
// statement and never checked as true, and the constructions member as
// the mode SHAPE found and, in full mode, the digest it computed of the
// text (SPEC.md section 7.2). Environment is the marker, null for a
// production proof, and SigningKeyEvent is null when the proof carries
// none.
type InspectProof struct {
	Block           any                  `json:"block"`
	Commitments     any                  `json:"commitments"`
	Constructions   InspectConstructions `json:"constructions"`
	Environment     any                  `json:"environment"`
	Format          any                  `json:"format"`
	Observations    any                  `json:"observations,omitempty"`
	SigningKeyEvent any                  `json:"signing_key_event"`
	Subject         any                  `json:"subject,omitempty"`
	Type            any                  `json:"type"`
	Version         any                  `json:"version"`
	Witnesses       any                  `json:"witnesses,omitempty"`
}

// InspectConstructions is the constructions mode SHAPE found and, in full
// mode, the digest it computed of the text (SPEC.md section 7.2).
type InspectConstructions struct {
	Mode   string `json:"mode"`
	Digest string `json:"digest,omitempty"`
}

// InspectRecomputed is one CHECKED value of the proof: the row of the
// constructions text that defines it, its pointer, the value the proof
// carries there, and the value recomputed from what the proof carries,
// null when there is none (verifier.Recomputed). A handover_signature's
// row alone has CarriedIs "signature" and ValueIs "signed_message": its
// carried value is the signature, and its value the 32 octets of the
// message that signature signs, which is not compared with it.
type InspectRecomputed struct {
	Construction string `json:"construction"`
	Part         string `json:"part"`
	Carried      any    `json:"carried"`
	CarriedIs    string `json:"carried_is,omitempty"`
	Value        any    `json:"value"`
	ValueIs      string `json:"value_is,omitempty"`
}

// The labels of a handover_signature's row, in --json and in the text.
const (
	carriedIsSignature   = "signature"
	valueIsSignedMessage = "signed_message"
)

// InspectDeclaredTime is one time the proof carries, at its pointer, of
// its class, as carried: declared by the signer, never a fact and never an
// edge of a submission window (SPEC.md sections 11 and 11.5).
type InspectDeclaredTime struct {
	Part  string `json:"part"`
	Class string `json:"class"`
	Value any    `json:"value"`
}

// InspectRecorded is one RECORDED value of the proof, at its pointer:
// declared by the signer, and not attributed to Truestamp (SPEC.md
// sections 3.7 and 11.5).
type InspectRecorded struct {
	Part  string `json:"part"`
	Value any    `json:"value"`
}

// claimsName is a member name of the claims, which --json writes with
// each bidirectional and zero-width code point as a \uXXXX escape: the
// same JSON value, which no reader shows as a reordering or an invisible
// character (SPEC.md section 3.7, SUBMITTED).
type claimsName string

// escapedCodePoint reports whether r is one of the code points a claims
// member name never writes as itself in --json: the Arabic letter mark
// U+061C, the zero-width and directional marks U+200B to U+200F, the
// directional embeddings and overrides U+202A to U+202E, the invisible
// operators U+2060 to U+2064, the directional isolates U+2066 to U+2069,
// and the zero-width no-break space U+FEFF. Every code point of Unicode's
// Bidi_Control property is among them.
func escapedCodePoint(r rune) bool {
	return r == 0x061C || r >= 0x200B && r <= 0x200F || r >= 0x202A && r <= 0x202E || r >= 0x2060 && r <= 0x2064 ||
		r >= 0x2066 && r <= 0x2069 || r == 0xFEFF
}

// MarshalJSON writes n as a JSON string, each of escapedCodePoint's code
// points as its \uXXXX escape.
func (n claimsName) MarshalJSON() ([]byte, error) {
	encoded, err := json.Marshal(string(n))
	if err != nil {
		return nil, err
	}
	var b strings.Builder
	for _, r := range string(encoded) {
		if escapedCodePoint(r) {
			fmt.Fprintf(&b, `\u%04x`, r)
			continue
		}
		b.WriteRune(r)
	}
	return []byte(b.String()), nil
}

// inspectOutputOf is the output of in, an inspection of octets octets read
// from source.
func inspectOutputOf(in *verifier.Inspection, source string, octets int) InspectOutput {
	out := InspectOutput{
		Source: source, Recomputed: []InspectRecomputed{}, DeclaredTimes: []InspectDeclaredTime{}, Recorded: []InspectRecorded{},
	}
	if octets <= strictjson.MaxReadOctets {
		out.Octets = &octets
	}
	if r := in.Rejection; r != nil {
		out.Rejection = &InspectRejection{Code: string(r.Code), Message: r.Message, Step: rejectionStep(r), SubReason: string(r.SubReason)}
		return out
	}
	signature, _ := in.Document["signature"].(map[string]any)
	text := func(name string) string { s, _ := signature[name].(string); return s }
	out.Signature = &InspectSignature{Alg: text("alg"), PublicKey: text("public_key"), Value: text("value")}
	keyID, proofHash := in.KeyID, in.ProofHash
	out.KeyID, out.ProofHash = &keyID, &proofHash
	if r := in.Refusal; r != nil {
		out.Rejection = &InspectRejection{Code: string(r.Code), Message: r.Message, Part: r.Part, Step: r.Step}
		return out
	}
	out.Proof = inspectProofOf(in)
	for _, r := range in.Recomputed {
		row := InspectRecomputed{Construction: r.Construction, Part: r.Part, Carried: r.Carried, Value: r.Value}
		if r.SignedMessage {
			row.CarriedIs, row.ValueIs = carriedIsSignature, valueIsSignedMessage
		}
		out.Recomputed = append(out.Recomputed, row)
		if in.Type == "item" && isItemHashRow(row) {
			recomputed, _ := r.Value.(string)
			if code, ok := itemCodeOfHex(recomputed); ok {
				out.ItemCode = &code
			}
		}
	}
	for _, d := range in.DeclaredTimes {
		out.DeclaredTimes = append(out.DeclaredTimes, InspectDeclaredTime{Part: d.Part, Class: d.Class, Value: d.Value})
	}
	for _, r := range in.Recorded {
		out.Recorded = append(out.Recorded, InspectRecorded{Part: r.Part, Value: r.Value})
	}
	return out
}

// isItemHashRow reports whether r is the recomputed item_hash of an item
// proof's subject, the row the item code is computed from.
func isItemHashRow(r InspectRecomputed) bool {
	return r.Construction == "item_hash" && r.Part == "/subject/item_hash"
}

// rejectionStep is the step of SPEC.md section 10.2 a bare rejection comes
// from, by its code, as the table of section 12.1 gives it. It panics for a
// code the table lacks, which the verifier never gives.
func rejectionStep(r *verifier.Rejection) string {
	entry, ok := specdata.LookupReason(r.Code)
	if !ok {
		panic(fmt.Sprintf("cmd: the verifier gave the code %q, which is not in SPEC.md section 12.1's table", r.Code))
	}
	return entry.Step
}

// inspectProofOf is the proof SHAPE passed, as InspectProof gives it.
func inspectProofOf(in *verifier.Inspection) *InspectProof {
	c := in.Carried
	out := &InspectProof{
		Block: c["block"], Commitments: c["commitments"], Environment: c["environment"], Format: c["format"],
		Observations: c["observations"], SigningKeyEvent: c["signing_key_event"], Type: c["type"], Version: c["version"],
		Witnesses:     c["witnesses"],
		Constructions: InspectConstructions{Mode: in.Constructions.Mode, Digest: in.Constructions.Digest},
	}
	if subject, ok := c["subject"].(map[string]any); ok || in.ClaimsMembers != nil {
		given := map[string]any{}
		for name, value := range subject {
			given[name] = value
		}
		if in.ClaimsMembers != nil {
			names := make([]claimsName, 0, len(in.ClaimsMembers))
			for _, name := range in.ClaimsMembers {
				names = append(names, claimsName(name))
			}
			given["claims_members"] = names
		}
		out.Subject = given
	}
	return out
}

// inspectSortedKeys are a map's names in JCS order (SPEC.md section 2.5),
// so that every run lists them alike.
func inspectSortedKeys[T any](m map[string]T) []string {
	names := make([]string, 0, len(m))
	for name := range m {
		names = append(names, name)
	}
	verifier.SortBy(names, func(name string) verifier.Pointer { return verifier.Pointer{verifier.Name(name)} })
	return names
}

// declaredCaption heads the times and the RECORDED values the proof
// carries, as SPEC.md section 11.5 captions a value not attributed to
// Truestamp: inspect checks no key and no signature, so nothing it shows
// is attributed.
const declaredCaption = "Declared by the signer (not attributed to Truestamp)"

// declaredClosing is the last line of the declared section.
const declaredClosing = "None of these values is confirmed, and no time among them is an edge of a submission window."

// inspectRow writes one row of the text: label, padded, and value.
func inspectRow(b *strings.Builder, label, value string) {
	fmt.Fprintf(b, "  %-24s %s\n", label, value)
}

// renderInspect is the text of out, every value whole.
func renderInspect(out InspectOutput) string {
	var b strings.Builder
	row := func(label, value string) { inspectRow(&b, label, value) }
	section := func(name string) { b.WriteString("\n" + ui.SectionHeader(name) + "\n") }
	b.WriteString(ui.AccentBoldStyle().Render("  Proof") + "\n\n")
	if out.Octets == nil {
		row("Source", fmt.Sprintf("%s (more than %d octets)", out.Source, strictjson.MaxReadOctets))
	} else {
		row("Source", fmt.Sprintf("%s (%d octets)", out.Source, *out.Octets))
	}
	if r := out.Rejection; r != nil {
		where := r.Step
		if r.Part != nil {
			where += ", at " + *r.Part
		}
		if r.SubReason != "" {
			where += ", " + r.SubReason
		}
		row("Refused", fmt.Sprintf("%s (%s)", r.Code, where))
		row("", r.Message)
	}
	p := out.Proof
	if p != nil {
		row("Format", fmt.Sprintf("%s, version %s", inspectValueText(p.Format), inspectValueText(p.Version)))
		row("Type", inspectValueText(p.Type))
		environment := "none: a production proof"
		if p.Environment != nil {
			environment = inspectValueText(p.Environment)
		}
		row("Environment marker", environment)
		constructions := p.Constructions.Mode
		if p.Constructions.Digest != "" {
			constructions += ", digest " + p.Constructions.Digest
		}
		row("Constructions", constructions)
	}
	if s := out.Signature; s != nil {
		section("Signature")
		row("Alg", s.Alg)
		row("Public key", s.PublicKey)
		row("Value", s.Value)
	}
	if p != nil {
		section("Carried by the proof")
		for _, member := range []struct {
			name  string
			value any
		}{
			{"block", p.Block}, {"commitments", p.Commitments}, {"observations", p.Observations},
			{"signing_key_event", p.SigningKeyEvent}, {"subject", p.Subject}, {"witnesses", p.Witnesses},
		} {
			renderCarried(row, "/"+member.name, member.value)
		}
	}
	if out.ProofHash != nil {
		section("Recomputed from what the proof carries")
		row("proof_hash", *out.ProofHash)
		row("key id of the public key", *out.KeyID)
		for _, r := range out.Recomputed {
			carried, recomputed := "  carried", "  recomputed"
			if r.ValueIs == valueIsSignedMessage {
				carried, recomputed = "  carried signature", "  signed message"
			}
			row(r.Construction, "at "+r.Part)
			row(carried, inspectValueText(r.Carried))
			row(recomputed, inspectRecomputedText(r.Value))
			if out.ItemCode != nil && isItemHashRow(r) {
				row("  Item code", *out.ItemCode+" (of the recomputed item_hash: a name for the item, not a check)")
			}
		}
	}
	if p != nil {
		section(declaredCaption)
		for _, d := range out.DeclaredTimes {
			row(d.Part, inspectValueText(d.Value)+" ("+d.Class+")")
		}
		for _, r := range out.Recorded {
			row(r.Part, inspectValueText(r.Value))
		}
		row("", declaredClosing)
	}
	b.WriteString("\n  Nothing was verified. Run 'truestamp verify' to verify the proof.\n")
	return b.String()
}

// renderCarried writes value, at the pointer at, one row for each value it
// holds that is not an object, in document order: a path's nodes one row
// each, an object or an array the proof holds empty as {} or [], every
// other value whole, and the claims' names quoted, as the submitter's
// statement. A value that is absent, such as a signing key
// event the proof does not carry, writes nothing.
func renderCarried(row func(string, string), at string, value any) {
	switch v := value.(type) {
	case nil:
	case map[string]any:
		if len(v) == 0 {
			row(at, "{}")
		}
		for _, name := range inspectSortedKeys(v) {
			renderCarried(row, at+"/"+escapePointerSegment(name), v[name])
		}
	case []claimsName:
		quoted := make([]string, 0, len(v))
		for _, name := range v {
			quoted = append(quoted, strconv.Quote(string(name)))
		}
		row(strings.TrimSuffix(at, "/claims_members")+"/claims", strings.Join(quoted, ", ")+" (member names; submitted, never checked as true)")
	case []any:
		if len(v) == 0 {
			row(at, "[]")
		}
		for i, element := range v {
			renderCarried(row, at+"/"+strconv.Itoa(i), element)
		}
	default:
		row(at, inspectValueText(v))
	}
}

// escapePointerSegment writes name as an RFC 6901 segment: ~ as ~0 and /
// as ~1.
func escapePointerSegment(name string) string {
	return strings.ReplaceAll(strings.ReplaceAll(name, "~", "~0"), "/", "~1")
}

// inspectValueText writes a value the proof carries as it carries it: a
// string, a number's digits, or null.
func inspectValueText(v any) string {
	switch x := v.(type) {
	case nil:
		return "null"
	case string:
		return x
	case strictjson.Number:
		if n, ok := x.Int64(); ok {
			return strconv.FormatInt(n, 10)
		}
		return strconv.FormatFloat(float64(x), 'g', -1, 64)
	case bool:
		return strconv.FormatBool(x)
	}
	return fmt.Sprint(v)
}

// inspectRecomputedText writes a recomputed value, or says that there is
// none.
func inspectRecomputedText(v any) string {
	if v == nil {
		return "none: it cannot be recomputed from what the proof carries"
	}
	return inspectValueText(v)
}
