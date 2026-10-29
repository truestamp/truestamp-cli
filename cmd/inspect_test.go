// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"slices"
	"sort"
	"strconv"
	"strings"
	"sync"
	"testing"

	"github.com/truestamp/truestamp-cli/internal/itemcode"
	"github.com/truestamp/truestamp-cli/internal/specdata"
	"github.com/truestamp/truestamp-cli/internal/strictjson"
	"github.com/truestamp/truestamp-cli/internal/vectorcases"
	"github.com/truestamp/truestamp-cli/internal/verifier"
)

// runInspectCLI runs `truestamp inspect` on path with args after it, and
// gives its standard output, ANSI stripped, and its exit code.
func runInspectCLI(t *testing.T, path string, args ...string) (string, int) {
	t.Helper()
	cmd := exec.Command(binaryPath, append([]string{"inspect", path}, args...)...)
	cmd.Env = cleanEnv()
	out, err := cmd.Output()
	code := 0
	if err != nil {
		var exitErr *exec.ExitError
		if !errors.As(err, &exitErr) {
			t.Fatalf("inspect %s: %v", path, err)
		}
		code = exitErr.ExitCode()
	}
	return stripANSI(string(out)), code
}

// inspectJSON runs `truestamp inspect --json` on path and gives what it
// wrote, decoded twice: into InspectOutput and as a generic value.
func inspectJSON(t *testing.T, path string) (InspectOutput, map[string]any, int) {
	t.Helper()
	out, code := runInspectCLI(t, path, "--json")
	var got InspectOutput
	var raw map[string]any
	if err := json.Unmarshal([]byte(out), &got); err != nil {
		t.Fatalf("inspect --json %s: %v\n%s", path, err, out)
	}
	if err := json.Unmarshal([]byte(out), &raw); err != nil {
		t.Fatal(err)
	}
	return got, raw, code
}

// vectorDocument is the parsed JSON of the vector file at relative, a
// path under vectors/.
func vectorDocument(t *testing.T, relative string) any {
	t.Helper()
	data, err := os.ReadFile(vectorcases.Path(relative))
	if err != nil {
		t.Fatal(err)
	}
	var v any
	if err := json.Unmarshal(data, &v); err != nil {
		t.Fatalf("%s: %v", relative, err)
	}
	return v
}

// atPointer is the value at the RFC 6901 pointer p in v, and false where v
// holds none.
func atPointer(v any, p string) (any, bool) {
	if p == "" {
		return v, true
	}
	for _, segment := range strings.Split(p, "/")[1:] {
		segment = strings.ReplaceAll(strings.ReplaceAll(segment, "~1", "/"), "~0", "~")
		switch node := v.(type) {
		case map[string]any:
			next, ok := node[segment]
			if !ok {
				return nil, false
			}
			v = next
		case []any:
			i, err := strconv.Atoi(segment)
			if err != nil || i < 0 || i >= len(node) {
				return nil, false
			}
			v = node[i]
		default:
			return nil, false
		}
	}
	return v, true
}

// traceRow is one row of a vector's trace file (SPEC.md section 10.6).
type traceRow struct {
	Construction string   `json:"construction"`
	Declared     any      `json:"declared"`
	Inputs       []string `json:"inputs"`
	Part         string   `json:"part"`
	Recomputed   any      `json:"recomputed"`
	Status       string   `json:"status"`
}

// givenAsRecomputed reports whether inspect gives the trace row among its
// recomputed values: every row but a Bitcoin commitment's header_time,
// which is a time, given only as declared (inspect's help).
func givenAsRecomputed(r traceRow) bool {
	return !strings.HasSuffix(r.Part, "/header_time")
}

// soundVector is a sound proof of the vectors with its trace.
type soundVector struct {
	proof string
	trace []traceRow
}

// soundVectors are the vectors' sound proofs under positive/, each with
// its trace, one or more of each of the six types.
func soundVectors(t *testing.T) []soundVector {
	t.Helper()
	cases, err := vectorcases.Load()
	if err != nil {
		t.Fatal(err)
	}
	var out []soundVector
	seen := map[string]bool{}
	types := map[string]bool{}
	for _, c := range cases {
		if c.Expected.Trace == "" || !strings.HasPrefix(c.Proof, "positive/") || seen[c.Proof] {
			continue
		}
		seen[c.Proof] = true
		data, err := os.ReadFile(vectorcases.Path(c.Expected.Trace))
		if err != nil {
			t.Fatal(err)
		}
		var rows []traceRow
		if err := json.Unmarshal(data, &rows); err != nil {
			t.Fatalf("%s: %v", c.Expected.Trace, err)
		}
		out = append(out, soundVector{c.Proof, rows})
		typ, _ := atPointer(vectorDocument(t, c.Proof), "/proof/type")
		types[typ.(string)] = true
	}
	if len(out) != 11 || len(types) != 6 {
		t.Fatalf("%d sound proofs with traces, of %d types; want 11 of all 6", len(out), len(types))
	}
	return out
}

// textRow is one row of inspect's text, as the test writes it: the label
// padded to 24 columns, then the value.
func textRow(label, value string) string {
	return fmt.Sprintf("  %-24s %s", label, value)
}

// testValueText writes a value decoded from --json as the text writes it:
// a string, an integer's digits, or null.
func testValueText(v any) string {
	switch x := v.(type) {
	case nil:
		return "null"
	case string:
		return x
	case float64:
		return strconv.FormatInt(int64(x), 10)
	}
	return fmt.Sprint(v)
}

// recomputedLines are the text's three lines of one recomputed value: a
// handover_signature's labelled as the carried signature and the signed
// message, every other's as carried and recomputed.
func recomputedLines(r InspectRecomputed) []string {
	value := testValueText(r.Value)
	if r.Value == nil {
		value = "none: it cannot be recomputed from what the proof carries"
	}
	carried, recomputed := "  carried", "  recomputed"
	if r.ValueIs == "signed_message" {
		carried, recomputed = "  carried signature", "  signed message"
	}
	return []string{textRow(r.Construction, "at "+r.Part), textRow(carried, testValueText(r.Carried)), textRow(recomputed, value)}
}

// labelled is r as --json gives it: a handover_signature's row with
// carried_is "signature" and value_is "signed_message", since the value
// recomputed for it is the message the carried signature signs, and every
// other row as it is.
func labelled(r InspectRecomputed) InspectRecomputed {
	if strings.HasSuffix(r.Part, "/handover_signature") {
		r.CarriedIs, r.ValueIs = "signature", "signed_message"
	}
	return r
}

// textSection is the lines of the section of text headed title: those
// after its heading and its rule, up to the blank line that ends it.
func textSection(t *testing.T, text, title string) []string {
	t.Helper()
	at := strings.Index(text, "\n  "+title+"\n")
	if at < 0 {
		t.Fatalf("the text has no section %q:\n%s", title, text)
	}
	lines := strings.Split(text[at+1:], "\n")[2:]
	end := slices.Index(lines, "")
	if end < 0 {
		end = len(lines)
	}
	return lines[:end]
}

// inspect on each sound proof of the vectors, one or more of every type,
// gives exactly the trace's rows but a header_time, in its order: each
// CHECKED value at its part, with its constructions row, the value carried
// and the value recomputed, in --json and in the text, where the section
// "Recomputed from what the proof carries" is exactly proof_hash, the key
// id and those rows. Mutants killed: the witness rows of the link step
// left out (an item's /subject/record/witness_hashes/entropy_X and block);
// the Bitcoin rows left out (transaction_hash, header_hash and
// header_merkle_root); the carried value given as the recomputed one; the
// text printing a value without its part; the Item code row left out of
// the text, or placed before the recomputed item_hash row. A
// handover_signature's row is labelled as the carried signature and the
// signed message (TestCLI_Inspect_LabelsTheHandoverSignatureRow).
func TestCLI_Inspect_GivesTheTracesRecomputedValues(t *testing.T) {
	witnesses, bitcoin := 0, 0
	for _, v := range soundVectors(t) {
		t.Run(v.proof, func(t *testing.T) {
			got, _, code := inspectJSON(t, vectorcases.Path(v.proof))
			if code != 0 || got.Rejection != nil {
				t.Fatalf("exit %d, rejection %+v", code, got.Rejection)
			}
			var want []InspectRecomputed
			for _, r := range v.trace {
				if givenAsRecomputed(r) {
					want = append(want, labelled(InspectRecomputed{Construction: r.Construction, Part: r.Part, Carried: r.Declared, Value: r.Recomputed}))
					switch r.Construction {
					case "witnesses":
						witnesses++
					case "bitcoin":
						bitcoin++
					}
				}
			}
			if len(want) < 4 {
				t.Fatalf("the trace gives only %d recomputed values; the test would prove little", len(want))
			}
			if !reflect.DeepEqual(got.Recomputed, want) {
				t.Errorf("recomputed\n%v\nwant\n%v", got.Recomputed, want)
			}
			text, code := runInspectCLI(t, vectorcases.Path(v.proof))
			if code != 0 {
				t.Fatalf("text: exit %d", code)
			}
			lines := []string{textRow("proof_hash", *got.ProofHash), textRow("key id of the public key", *got.KeyID)}
			for _, r := range want {
				lines = append(lines, recomputedLines(r)...)
				if isItemHashRow(r) && got.ItemCode != nil {
					lines = append(lines, textRow("  Item code", *got.ItemCode+" (of the recomputed item_hash: a name for the item, not a check)"))
				}
			}
			if section := textSection(t, text, "Recomputed from what the proof carries"); !slices.Equal(section, lines) {
				t.Errorf("the recomputed section reads\n%s\nwant\n%s", strings.Join(section, "\n"), strings.Join(lines, "\n"))
			}
		})
	}
	if witnesses < 8 || bitcoin < 12 {
		t.Fatalf("the traces give %d witness rows and %d Bitcoin rows; the test would prove little", witnesses, bitcoin)
	}
}

// inspect on each vector that declares one value wrong (a declared-* case
// whose first failing row is DERIVE's derivation_mismatch at a value
// inspect gives as recomputed) gives at that part the value recomputed
// from the preimage, which is the value a sound vector's trace gives for
// the same preimage, beside the value the proof declares as carried.
// Mutant killed: inspect giving the declared value, the value at the
// part, in place of the recomputed one, in --json or in the text.
func TestCLI_Inspect_GivesTheRecomputedValueNotTheDeclaredOne(t *testing.T) {
	sound := soundVectors(t)
	cases, err := vectorcases.Load()
	if err != nil {
		t.Fatal(err)
	}
	checked := 0
	for _, c := range cases {
		if !strings.HasPrefix(c.ID, "declared-") {
			continue
		}
		i := slices.IndexFunc(c.Expected.Rows, func(r vectorcases.Row) bool { return r.Status == "fail" })
		if i < 0 || c.Expected.Rows[i].Step != "DERIVE" || *c.Expected.Rows[i].Code != "derivation_mismatch" {
			continue
		}
		part := *c.Expected.Rows[i].Part
		negative := vectorDocument(t, c.Proof)
		declared, _ := atPointer(negative, "/proof"+part)
		var want *traceRow
		for _, v := range sound {
			positive := vectorDocument(t, v.proof)
			for j, r := range v.trace {
				if r.Part != part || !givenAsRecomputed(r) || reflect.DeepEqual(declared, r.Recomputed) {
					continue
				}
				same := true
				for _, input := range r.Inputs {
					a, _ := atPointer(positive, "/proof"+input)
					b, _ := atPointer(negative, "/proof"+input)
					same = same && reflect.DeepEqual(a, b)
				}
				if same {
					want = &v.trace[j]
				}
			}
		}
		if want == nil {
			continue
		}
		checked++
		t.Run(c.ID, func(t *testing.T) {
			got, _, code := inspectJSON(t, vectorcases.Path(c.Proof))
			if code != 0 {
				t.Fatalf("exit %d", code)
			}
			wantRow := labelled(InspectRecomputed{Construction: want.Construction, Part: part, Carried: declared, Value: want.Recomputed})
			k := slices.IndexFunc(got.Recomputed, func(r InspectRecomputed) bool { return r.Part == part })
			if k < 0 || !reflect.DeepEqual(got.Recomputed[k], wantRow) {
				t.Errorf("at %s inspect gives %v, want %v", part, got.Recomputed, wantRow)
			}
			text, _ := runInspectCLI(t, vectorcases.Path(c.Proof))
			if lines := strings.Join(recomputedLines(wantRow), "\n") + "\n"; !strings.Contains(text, lines) {
				t.Errorf("the text lacks\n%s", lines)
			}
		})
	}
	if checked < 15 {
		t.Fatalf("only %d declared-value vectors checked", checked)
	}
}

// inspect labels a handover_signature's row, whose recomputed value is not
// a value to compare with the one carried but the 32-octet message the
// carried signature signs: on the rotation block, each of its two
// handover_signature rows reads, in the text, "carried signature" beside
// the signature and "signed message" beside the hex of SHA-256(0x62 ||
// JCS(the key event without handover_signature)), and in --json has
// exactly the members construction, part, carried, carried_is
// ("signature"), value and value_is ("signed_message"), while each other
// handover row is labelled carried and recomputed and has neither
// carried_is nor value_is. Mutants killed: the old labels back (the text's
// "carried" and "recomputed" on the signature's row, and --json without
// carried_is and value_is).
func TestCLI_Inspect_LabelsTheHandoverSignatureRow(t *testing.T) {
	const file = "positive/block-rotation.proof.json"
	const message = "7d5ec839df0deaf94d8c33c2c88d9a97b34d2c301342d634d9cf59178a2b08e3"
	const signature = "lGwR5OSy9L_PQXi4cA8jZCdM6wU-P4FVhtRa4WMVb3D04NTxq90SXJ_dLYETb5leVcMVCj9PTjYTVELk1Pm1AA"
	_, raw, code := inspectJSON(t, vectorcases.Path(file))
	if code != 0 {
		t.Fatalf("exit %d", code)
	}
	text, _ := runInspectCLI(t, vectorcases.Path(file))
	signed, others := 0, 0
	for _, r := range raw["recomputed"].([]any) {
		row := r.(map[string]any)
		part := row["part"].(string)
		if row["construction"] != "handover" {
			continue
		}
		if !strings.HasSuffix(part, "/handover_signature") {
			others++
			_, hasCarriedIs := row["carried_is"]
			_, hasValueIs := row["value_is"]
			if hasCarriedIs || hasValueIs {
				t.Errorf("the handover row at %s is labelled: %v", part, row)
			}
			lines := strings.Join([]string{
				textRow("handover", "at "+part), textRow("  carried", testValueText(row["carried"])), textRow("  recomputed", testValueText(row["value"])),
			}, "\n")
			if !strings.Contains(text, "\n"+lines+"\n") {
				t.Errorf("the text lacks\n%s", lines)
			}
			continue
		}
		signed++
		want := map[string]any{
			"construction": "handover", "part": part, "carried": signature, "carried_is": "signature", "value": message, "value_is": "signed_message",
		}
		if !reflect.DeepEqual(row, want) {
			t.Errorf("--json gives the row\n%v\nwant\n%v", row, want)
		}
		lines := strings.Join([]string{
			textRow("handover", "at "+part), textRow("  carried signature", signature), textRow("  signed message", message),
		}, "\n")
		if !strings.Contains(text, "\n"+lines+"\n") {
			t.Errorf("the text lacks\n%s", lines)
		}
	}
	if signed != 2 || others != 6 {
		t.Fatalf("%d handover_signature rows and %d other handover rows; want 2 and 6", signed, others)
	}
}

// inspect gives the worked example's proof_hash and K's key id (SPEC.md
// section 6.3), computed from the document as parsed, and, since SHAPE
// refuses that four-member proof, the verifier's SHAPE rejection beside
// them and nothing of the proof. Mutants killed: proof_hash computed over
// the envelope rather than P; the key id written as the first four octets
// of K rather than of SHA-256(0x60 || K); proof_hash and key_id given only
// for a proof SHAPE passes.
func TestCLI_Inspect_TheWorkedExample(t *testing.T) {
	var worked struct {
		Document  string `json:"document"`
		KeyID     string `json:"key_id"`
		ProofHash string `json:"proof_hash"`
	}
	data, err := os.ReadFile(vectorcases.Path("worked-example.json"))
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(data, &worked); err != nil {
		t.Fatal(err)
	}
	file := filepath.Join(t.TempDir(), "worked.proof.json")
	if err := os.WriteFile(file, []byte(" \n"+worked.Document+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	got, _, code := inspectJSON(t, file)
	if code != 1 || got.ProofHash == nil || *got.ProofHash != worked.ProofHash || got.KeyID == nil || *got.KeyID != worked.KeyID {
		t.Fatalf("exit %d, proof_hash %v, key_id %v; want 1, %s and %s", code, got.ProofHash, got.KeyID, worked.ProofHash, worked.KeyID)
	}
	if got.Rejection == nil || got.Rejection.Code != "shape_invalid" || got.Rejection.Step != "SHAPE" || got.Rejection.Part == nil ||
		got.Proof != nil || len(got.Recomputed) != 0 || len(got.DeclaredTimes) != 0 || len(got.Recorded) != 0 {
		t.Fatalf("worked example: %+v", got)
	}
	text, _ := runInspectCLI(t, file)
	for _, line := range []string{
		textRow("proof_hash", worked.ProofHash),
		textRow("key id of the public key", worked.KeyID),
		textRow("Refused", fmt.Sprintf("shape_invalid (SHAPE, at %s)", *got.Rejection.Part)),
	} {
		if !strings.Contains(text, line+"\n") {
			t.Errorf("the text lacks %q\n%s", line, text)
		}
	}
}

// verifyRefusal is what verify gives a vector under the configuration of
// an offline run under E_v test with the vectors' keyring: its bare
// rejection, or its report's SHAPE row when that row fails.
func verifyRefusal(t *testing.T, path string) (code, step, message string, ok bool) {
	t.Helper()
	outcome := verifier.Verify(offlineTestConfig(t), func() ([]byte, error) { return strictjson.ReadFile(path) })
	if r := outcome.Rejection; r != nil {
		entry, _ := specdata.LookupReason(r.Code)
		return string(r.Code), entry.Step, r.Message, true
	}
	if outcome.Report == nil {
		return "", "", "", false
	}
	for _, row := range outcome.Report.Rows {
		if row.Step == "SHAPE" && row.Status == verifier.StatusFail {
			return string(row.Code), row.Step, row.Message, true
		}
	}
	return "", "", "", false
}

// inspect reports each vector the verifier refuses before a report, or
// whose SHAPE fails, with the verifier's rejection: a bare rejection's
// code and sub_reason at its step, part null, and SHAPE's failing row's
// code and part; and passes, exit 0, each vector whose SHAPE passes. The
// cases whose code is configuration_error, which a run's configuration
// gives and inspect has none of, and those whose gate fails first, which
// say nothing of SHAPE, are left out. Where verify, run offline under E_v
// test with the vectors' keyring, refuses the same file with the same code
// at the same step, inspect's message is verify's, word for word. Mutants
// killed: inspect giving shape_invalid for every SHAPE failure (an
// unsupported_format, unsupported_version or construction_mismatch case
// differs); SHAPE's part left out; a sub_reason left out; the read cut at
// MAX_READ_OCTETS rather than one octet past it (the input_too_large case
// reads as malformed_json); SHAPE not made (no SHAPE case is refused);
// a bare code given another step than SPEC.md section 12.1's
// (rejectionStep giving ENVELOPE for input_too_large); the message of a
// SHAPE refusal left empty.
func TestCLI_Inspect_RefusesWithTheVerifiersRejection(t *testing.T) {
	cases, err := vectorcases.Load()
	if err != nil {
		t.Fatal(err)
	}
	type outcome struct {
		passes          bool
		code, sub, step string
		part            *string
	}
	want := map[string]outcome{}
	for _, c := range cases {
		e := c.Expected
		var o outcome
		switch {
		case e.Bare() && e.Code == string(specdata.ReasonConfigurationError):
			continue
		case e.Bare():
			entry, _ := specdata.LookupReason(specdata.Reason(e.Code))
			o = outcome{code: e.Code, sub: e.SubReason, step: entry.Step}
		default:
			i := slices.IndexFunc(e.Rows, func(r vectorcases.Row) bool { return r.Step == "SHAPE" })
			if i < 0 || e.Rows[i].Status == "skip" {
				continue
			}
			row := e.Rows[i]
			o = outcome{passes: row.Status == "pass"}
			if !o.passes {
				o = outcome{code: *row.Code, step: "SHAPE", part: row.Part}
			}
		}
		if prior, seen := want[c.Proof]; seen && !reflect.DeepEqual(prior, o) {
			t.Fatalf("%s: two cases expect %v and %v", c.Proof, prior, o)
		}
		want[c.Proof] = o
	}
	var proofs []string
	counts := map[string]int{}
	for proof, o := range want {
		proofs = append(proofs, proof)
		counts[o.code]++
	}
	sort.Strings(proofs)
	for _, code := range []string{"", "input_too_large", "malformed_json", "invalid_envelope", "unsupported_format", "unsupported_version", "shape_invalid", "construction_mismatch"} {
		if counts[code] == 0 {
			t.Fatalf("no vector gives %q; counts %v", code, counts)
		}
	}
	messages := map[string]int{}
	for _, proof := range proofs {
		o := want[proof]
		got, _, code := inspectJSON(t, vectorcases.Path(proof))
		if o.passes {
			if code != 0 || got.Rejection != nil || got.Proof == nil {
				t.Errorf("%s: exit %d, rejection %+v; SHAPE passes it", proof, code, got.Rejection)
			}
			continue
		}
		r := got.Rejection
		if code != 1 || r == nil || r.Code != o.code || r.SubReason != o.sub || r.Step != o.step || !reflect.DeepEqual(r.Part, o.part) || got.Proof != nil {
			t.Errorf("%s: exit %d, rejection %+v, want %+v", proof, code, r, o)
			continue
		}
		if vCode, vStep, message, ok := verifyRefusal(t, vectorcases.Path(proof)); ok && vCode == r.Code && vStep == r.Step {
			messages[r.Step]++
			if r.Message != message || message == "" {
				t.Errorf("%s: inspect's message %q, verify's %q", proof, r.Message, message)
			}
		}
	}
	if messages["SHAPE"] < 20 || messages["ENVELOPE"] < 5 || messages["READ"] < 1 || messages["PARSE"] < 5 {
		t.Fatalf("messages compared with verify's, by step: %v; the test would prove little", messages)
	}
}

// carriedTimes adds to out the times the value v at pointer at carries,
// found by walking it rather than by the verifier's layout: each
// header_time, closed_at and timeStamp, each time of an entropy payload,
// and the claims' timestamp, each at its pointer.
func carriedTimes(v any, at string, out map[string]any) {
	switch node := v.(type) {
	case map[string]any:
		for name, value := range node {
			p := at + "/" + escapePointerSegment(name)
			switch {
			case name == "header_time" || name == "closed_at" || name == "timeStamp":
				out[p] = value
			case name == "time" && strings.HasSuffix(at, "/payload"):
				out[p] = value
			case name == "timestamp" && strings.HasSuffix(at, "/claims"):
				out[p] = value
			default:
				carriedTimes(value, p, out)
			}
		}
	case []any:
		for i, value := range node {
			carriedTimes(value, at+"/"+strconv.Itoa(i), out)
		}
	}
}

// timeClassOf is the class of the time at part, by the member that holds
// it (SPEC.md sections 3.4 and 3.5): a NIST timeStamp NISTTIME, a Bitcoin
// entropy payload's time INT, and a header_time or a closed_at TIME.
func timeClassOf(part string) string {
	switch {
	case strings.HasSuffix(part, "/timeStamp"):
		return "NISTTIME"
	case strings.HasSuffix(part, "/payload/time"):
		return "INT"
	}
	return "TIME"
}

// inspect never prints a time a proof carries as a fact: on every sound
// proof, --json gives each carried time but the claims' timestamp under
// declared_times, at its pointer, of its class, and nowhere else, and the
// text prints each only in the section captioned "Declared by the signer
// (not attributed to Truestamp)", and the claims' timestamp nowhere. That
// section is exactly each declared time, as --json gives it, with its
// class, then each RECORDED value, as --json gives it, then its closing
// line. Mutants killed: a commitment entry's header_time kept in its JSON
// and printed in the commitments section; a Stellar entropy object's
// closed_at kept with the object; a Bitcoin entropy object's time left out
// of declared_times; the caption written otherwise; every NIST timeStamp
// given the class TIME; a "Submitted before" row printed inside the
// section; an integer time written as nothing in the text.
func TestCLI_Inspect_NeverPrintsACarriedTimeAsAFact(t *testing.T) {
	declared := 0
	classes := map[string]int{}
	for _, v := range soundVectors(t) {
		t.Run(v.proof, func(t *testing.T) {
			proof, _ := atPointer(vectorDocument(t, v.proof), "/proof")
			times := map[string]any{}
			carriedTimes(proof, "", times)
			got, raw, code := inspectJSON(t, vectorcases.Path(v.proof))
			if code != 0 {
				t.Fatalf("exit %d", code)
			}
			wantDeclared := map[string]string{}
			for p, value := range times {
				if !strings.HasSuffix(p, "/claims/timestamp") {
					wantDeclared[p] = timeText(value) + " " + timeClassOf(p)
				}
			}
			gotDeclared := map[string]string{}
			for _, d := range got.DeclaredTimes {
				gotDeclared[d.Part] = timeText(d.Value) + " " + d.Class
				classes[d.Class]++
			}
			if !reflect.DeepEqual(gotDeclared, wantDeclared) {
				t.Errorf("declared_times %v, want %v", gotDeclared, wantDeclared)
			}
			declared += len(wantDeclared)
			delete(raw, "declared_times")
			rest, err := json.Marshal(raw)
			if err != nil {
				t.Fatal(err)
			}
			text, _ := runInspectCLI(t, vectorcases.Path(v.proof))
			var lines []string
			for _, d := range got.DeclaredTimes {
				lines = append(lines, textRow(d.Part, testValueText(d.Value)+" ("+d.Class+")"))
			}
			for _, r := range got.Recorded {
				lines = append(lines, textRow(r.Part, testValueText(r.Value)))
			}
			lines = append(lines, textRow("", "None of these values is confirmed, and no time among them is an edge of a submission window."))
			if section := textSection(t, text, "Declared by the signer (not attributed to Truestamp)"); !slices.Equal(section, lines) {
				t.Errorf("the declared section reads\n%s\nwant\n%s", strings.Join(section, "\n"), strings.Join(lines, "\n"))
			}
			start := strings.Index(text, "\n  Declared by the signer (not attributed to Truestamp)\n")
			end := strings.Index(text, "\n  Nothing was verified.")
			if start < 0 || end < start {
				t.Fatalf("the text has no declared section before its last line:\n%s", text)
			}
			for p, value := range times {
				s := timeText(value)
				if strings.Contains(string(rest), s) {
					t.Errorf("--json gives the carried time %s (at %s) outside declared_times", s, p)
				}
				if strings.Contains(text[:start], s) || strings.Contains(text[end:], s) {
					t.Errorf("the text prints the carried time %s (at %s) outside its declared section", s, p)
				}
				if strings.HasSuffix(p, "/claims/timestamp") && strings.Contains(text, s) {
					t.Errorf("the text prints the claims' timestamp %s", s)
				}
			}
		})
	}
	if declared < 20 || classes["TIME"] == 0 || classes["NISTTIME"] == 0 || classes["INT"] == 0 {
		t.Fatalf("the sound proofs carry %d times, of the classes %v; the test would prove little", declared, classes)
	}
}

// timeText is a carried time as it reads in output: the string, or the
// integer's digits.
func timeText(v any) string {
	if n, ok := v.(float64); ok {
		return strconv.FormatInt(int64(n), 10)
	}
	return v.(string)
}

// inspectTop are the members of inspect --json.
var inspectTop = []string{"declared_times", "item_code", "key_id", "octets", "proof", "proof_hash", "recomputed", "recorded", "rejection", "signature", "source"}

// sortedKeys are m's names, sorted.
func sortedKeys(m map[string]any) []string {
	var out []string
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

// inspect --json's shape: exactly its eleven members, item_code among
// them, and exactly the proof members of each type's layout (SPEC.md
// section 3.6), with the signature as carried, the environment marker,
// the constructions mode and digest, and the source and its size; the
// text's header gives the format, the type and the marker; item_code is a
// string for an item proof, the code of the item_hash of its record, and
// null for every other type; a bare rejection has the same eleven
// members, null and empty where nothing was read, item_code null. Mutants
// killed: a member added beside proof_hash; item_code given omitempty (a
// bare rejection lacks it); item_code set for a bare rejection; item_code
// null for an item proof; witnesses written for every type; the
// environment marker read as production, in --json or in the text's
// marker line; the octets written as 0.
func TestCLI_Inspect_JSONShape(t *testing.T) {
	common := []string{"block", "commitments", "constructions", "environment", "format", "signing_key_event", "type", "version"}
	byType := map[string][]string{
		"item": {"subject", "witnesses"}, "entropy_nist": {"subject"}, "entropy_stellar": {"subject"},
		"entropy_bitcoin": {"subject"}, "block": nil, "beacon": {"observations"},
	}
	for _, v := range soundVectors(t) {
		t.Run(v.proof, func(t *testing.T) {
			doc := vectorDocument(t, v.proof)
			got, raw, _ := inspectJSON(t, vectorcases.Path(v.proof))
			if k := sortedKeys(raw); !slices.Equal(k, inspectTop) {
				t.Errorf("members %q, want %q", k, inspectTop)
			}
			typ, _ := atPointer(doc, "/proof/type")
			want := append(slices.Clone(common), byType[typ.(string)]...)
			sort.Strings(want)
			if k := sortedKeys(raw["proof"].(map[string]any)); !slices.Equal(k, want) {
				t.Errorf("proof members %q, want %q", k, want)
			}
			signature, _ := atPointer(doc, "/signature")
			s := signature.(map[string]any)
			if got.Signature == nil || got.Signature.Alg != s["alg"] || got.Signature.PublicKey != s["public_key"] || got.Signature.Value != s["value"] {
				t.Errorf("signature %+v, want %v", got.Signature, s)
			}
			if typ == "item" {
				record, _ := atPointer(doc, "/proof/subject/record")
				code, err := itemcode.OfItemRecord(record.(map[string]any))
				if err != nil || raw["item_code"] != code {
					t.Errorf("item_code %v, want %q (%v)", raw["item_code"], code, err)
				}
			} else if raw["item_code"] != nil {
				t.Errorf("item_code %v for a %v proof, want null", raw["item_code"], typ)
			}
			marker, _ := atPointer(doc, "/proof/environment")
			p := got.Proof
			if p.Environment != marker || marker == nil || p.Type != typ || p.Format != "truestamp-proof" || p.Version != float64(1) {
				t.Errorf("proof %+v: environment %v", p, marker)
			}
			if p.Constructions.Mode != "full" || !specdata.IsAccepted(p.Constructions.Digest) {
				t.Errorf("constructions %+v", p.Constructions)
			}
			info, err := os.Stat(vectorcases.Path(v.proof))
			if err != nil {
				t.Fatal(err)
			}
			if got.Source != vectorcases.Path(v.proof) || got.Octets == nil || int64(*got.Octets) != info.Size() {
				t.Errorf("source %q, octets %v, want %d", got.Source, got.Octets, info.Size())
			}
			text, _ := runInspectCLI(t, vectorcases.Path(v.proof))
			for _, line := range []string{
				textRow("Source", fmt.Sprintf("%s (%d octets)", got.Source, info.Size())),
				textRow("Format", "truestamp-proof, version 1"),
				textRow("Type", typ.(string)),
				textRow("Environment marker", marker.(string)),
				textRow("Constructions", "full, digest "+p.Constructions.Digest),
			} {
				if !strings.Contains(text, "\n"+line+"\n") {
					t.Errorf("the text lacks %q", line)
				}
			}
		})
	}
	got, raw, code := inspectJSON(t, vectorcases.Path("negative/envelope-document-array.proof.json"))
	if k := sortedKeys(raw); code != 1 || !slices.Equal(k, inspectTop) || got.Rejection == nil || got.Rejection.Code != "invalid_envelope" ||
		got.Signature != nil || got.KeyID != nil || got.ProofHash != nil || got.Proof != nil || got.Recomputed == nil || got.DeclaredTimes == nil ||
		got.Recorded == nil || got.ItemCode != nil {
		t.Errorf("a bare rejection: exit %d, members %q, %+v", code, k, got)
	}
}

// valueClass is one member of spec/value-classes.json (SPEC.md section
// 3.7).
type valueClass struct {
	Class     string `json:"class"`
	Otherwise string `json:"otherwise"`
	Pointer   string `json:"pointer"`
	When      string `json:"when"`
	Document  string `json:"document"`
}

// valueClasses are the members of spec/value-classes.json.
var valueClasses = sync.OnceValues(func() ([]valueClass, error) {
	data, err := os.ReadFile(filepath.Join(vectorcases.Dir(), "..", "spec", "value-classes.json"))
	if err != nil {
		return nil, err
	}
	var file struct {
		Members []valueClass `json:"members"`
	}
	if err := json.Unmarshal(data, &file); err != nil {
		return nil, err
	}
	return file.Members, nil
})

// classOf is the value class of the member of P at part, by
// spec/value-classes.json, or "" when it lists none: a member's pointer
// may hold * for any one segment and entropy_* for a segment that begins
// entropy_, and a member with a condition has its class when the condition
// holds in P, and its otherwise class when it does not.
func classOf(t *testing.T, proof any, part string) string {
	t.Helper()
	members, err := valueClasses()
	if err != nil {
		t.Fatal(err)
	}
	segments := strings.Split(part, "/")
	for _, m := range members {
		pattern := strings.Split(m.Pointer, "/")
		if m.Document != "" || len(pattern) != len(segments) {
			continue
		}
		matches := true
		for i, want := range pattern {
			if want != "*" && want != segments[i] && (want != "entropy_*" || !strings.HasPrefix(segments[i], "entropy_")) {
				matches = false
			}
		}
		if !matches {
			continue
		}
		if m.When == "" || classCondition(t, proof, part, m.When) {
			return m.Class
		}
		return m.Otherwise
	}
	return ""
}

// classCondition reads one of spec/value-classes.json's conditions in P,
// for the member at part.
func classCondition(t *testing.T, proof any, part, condition string) bool {
	t.Helper()
	var record any
	for _, block := range []string{"/signing_key_event/block", "/witnesses/block", "/block"} {
		if strings.HasPrefix(part, block+"/") {
			record, _ = atPointer(proof, block+"/record")
			break
		}
	}
	switch condition {
	case "genesis_block":
		_, ok := atPointer(record, "/metadata/genesis")
		return ok
	case "introduces_key_or_prerotation":
		typ, _ := atPointer(record, "/metadata/key_event/type")
		return slices.Contains([]any{"genesis", "rotation", "emergency_rotation", "prerotation"}, typ)
	case "walked":
		typ, _ := atPointer(proof, "/type")
		observations, _ := atPointer(proof, "/observations")
		o, _ := observations.(map[string]any)
		return typ != "block" && (typ != "beacon" || len(o) > 0)
	case "stellar_source":
		source, _ := atPointer(proof, part[:strings.LastIndex(part[:strings.LastIndex(part, "/")], "/")]+"/source")
		return source == "stellar"
	}
	t.Fatalf("spec/value-classes.json names the condition %s, which this test does not read", condition)
	return false
}

// proofLeaves adds to out each value P holds at a pointer that is not an
// object with members: an array, a scalar, an empty object, and the claims
// and the constructions member whole.
func proofLeaves(v any, at string, out map[string]any) {
	if object, ok := v.(map[string]any); ok && len(object) > 0 && at != "/subject/claims" && at != "/constructions" {
		for name, member := range object {
			proofLeaves(member, at+"/"+escapePointerSegment(name), out)
		}
		return
	}
	out[at] = v
}

// recordHashRows are the constructions rows whose recomputed value is a
// record hash.
var recordHashRows = []string{"block_hash", "claims_hash", "entropy_hash", "epoch_hash", "item_hash", "observation_hash", "witnesses"}

// carriedLines are the text's lines of the "Carried by the proof" section
// for --json's proof member, as the test writes them.
func carriedLines(at string, v any, out *[]string) {
	switch x := v.(type) {
	case nil:
	case map[string]any:
		if len(x) == 0 {
			*out = append(*out, textRow(at, "{}"))
		}
		for _, name := range inspectSortedKeys(x) {
			if at == "/subject" && name == "claims_members" {
				var quoted []string
				for _, n := range x[name].([]any) {
					quoted = append(quoted, strconv.Quote(n.(string)))
				}
				*out = append(*out, textRow("/subject/claims", strings.Join(quoted, ", ")+" (member names; submitted, never checked as true)"))
				continue
			}
			carriedLines(at+"/"+escapePointerSegment(name), x[name], out)
		}
	case []any:
		if len(x) == 0 {
			*out = append(*out, textRow(at, "[]"))
		}
		for i, element := range x {
			carriedLines(at+"/"+strconv.Itoa(i), element, out)
		}
	default:
		*out = append(*out, textRow(at, testValueText(x)))
	}
}

// inspect gives each member of every proof SHAPE passes (every vector proof
// it exits 0 on, the sound ones and those whose fault DERIVE finds) once,
// by its value class in spec/value-classes.json (SPEC.md section 3.7): a
// CHECKED value only in recomputed, as carried, a time only in
// declared_times, a RECORDED value only in recorded, the claims only as
// their member names, and every other value (PINNED, PREIMAGE, CHAIN
// POINTER, BLOCK HEIGHT, PATH INPUT) in proof at its own pointer, as
// carried; proof holds nothing else, so every nested object's member set
// is the proof's; no record hash of recomputed is a value in proof; and the
// text's "Carried by the proof" section is exactly proof's values, each
// whole, every raw_transaction and txoutproof among them as its full hex
// on the row of its pointer. Mutants killed: a carried
// member dropped from --json (a Stellar entry's transaction_hash); a
// carried block_hash added to the block node; a CHECKED value kept in
// proof and shown in the carried section unlabelled (signing_key_id kept
// where its key event introduces a key); a RECORDED value left outside
// the declared section (each record id kept in proof); a raw_transaction
// or a txoutproof shortened in the text (its first 64 hex digits and
// "...").
func TestCLI_Inspect_GivesEveryMemberByItsClass(t *testing.T) {
	cases, err := vectorcases.Load()
	if err != nil {
		t.Fatal(err)
	}
	seen := map[string]bool{}
	classes := map[string]int{}
	checkedProofs, wholeHex := 0, 0
	for _, c := range cases {
		if seen[c.Proof] {
			continue
		}
		seen[c.Proof] = true
		got, raw, code := inspectJSON(t, vectorcases.Path(c.Proof))
		if code != 0 {
			continue
		}
		checkedProofs++
		proof, _ := atPointer(vectorDocument(t, c.Proof), "/proof")
		leaves := map[string]any{}
		proofLeaves(proof, "", leaves)
		times := map[string]any{}
		carriedTimes(proof, "", times)
		recomputed := map[string][]InspectRecomputed{}
		for _, r := range got.Recomputed {
			recomputed[r.Part] = append(recomputed[r.Part], r)
		}
		declared := map[string]any{}
		for _, d := range got.DeclaredTimes {
			declared[d.Part] = d.Value
		}
		recorded := map[string]any{}
		for _, r := range got.Recorded {
			recorded[r.Part] = r.Value
		}
		carried := map[string]any{}
		proofLeaves(raw["proof"], "", carried)
		for part, value := range leaves {
			class := classOf(t, proof, part)
			classes[class]++
			_, isTime := times[part]
			switch {
			case isTime:
				if !reflect.DeepEqual(declared[part], value) || len(recomputed[part]) != 0 {
					t.Errorf("%s: the time at %s: declared %v, recomputed %v; carried %v", c.Proof, part, declared[part], recomputed[part], value)
				}
				delete(declared, part)
			case class == "CHECKED":
				rows := recomputed[part]
				for _, r := range rows {
					if !reflect.DeepEqual(r.Carried, value) {
						t.Errorf("%s: recomputed at %s carries %v, the proof %v", c.Proof, part, r.Carried, value)
					}
				}
				if len(rows) == 0 {
					t.Errorf("%s: the CHECKED value at %s is not among recomputed", c.Proof, part)
				}
				delete(recomputed, part)
			case class == "RECORDED":
				if got, ok := recorded[part]; !ok || !reflect.DeepEqual(got, value) {
					t.Errorf("%s: the RECORDED value at %s: recorded %v, carried %v", c.Proof, part, got, value)
				}
				delete(recorded, part)
			case class == "SUBMITTED":
				names := inspectSortedKeys(value.(map[string]any))
				gotNames, _ := atPointer(raw["proof"], "/subject/claims_members")
				var want []any
				for _, n := range names {
					want = append(want, n)
				}
				if !reflect.DeepEqual(gotNames, want) {
					t.Errorf("%s: claims_members %v, want %v", c.Proof, gotNames, want)
				}
				delete(carried, "/subject/claims_members")
			case part == "/constructions":
			case class == "" && !reflect.DeepEqual(value, map[string]any{}):
				t.Errorf("%s: spec/value-classes.json gives %s no class", c.Proof, part)
			default:
				if g, ok := carried[part]; !ok || !reflect.DeepEqual(g, value) {
					t.Errorf("%s: the %s value at %s: proof gives %v, the proof carries %v", c.Proof, class, part, g, value)
				}
				delete(carried, part)
			}
		}
		if _, ok := leaves["/constructions"]; !ok && !reflect.DeepEqual(carried["/constructions"], map[string]any{"mode": "none"}) {
			t.Errorf("%s: a proof without constructions gives %v", c.Proof, carried["/constructions"])
		}
		delete(carried, "/constructions")
		for _, absent := range []string{"/environment", "/signing_key_event"} {
			if v, ok := carried[absent]; ok && v == nil {
				delete(carried, absent)
			}
		}
		if len(declared)+len(recomputed)+len(recorded)+len(carried) > 0 {
			t.Errorf("%s: given beyond the proof's members: declared %v, recomputed %v, recorded %v, proof %v", c.Proof, declared, recomputed, recorded, carried)
		}
		values := map[string]any{}
		proofLeaves(raw["proof"], "", values)
		for _, r := range got.Recomputed {
			if !slices.Contains(recordHashRows, r.Construction) || r.Value == nil {
				continue
			}
			for part, v := range values {
				if v == r.Value {
					t.Errorf("%s: the record hash %v of %s is also proof's value at %s", c.Proof, r.Value, r.Part, part)
				}
			}
		}
		if !strings.HasPrefix(c.Proof, "positive/") {
			continue
		}
		var lines []string
		for _, name := range []string{"block", "commitments", "observations", "signing_key_event", "subject", "witnesses"} {
			carriedLines("/"+name, raw["proof"].(map[string]any)[name], &lines)
		}
		text, _ := runInspectCLI(t, vectorcases.Path(c.Proof))
		section := textSection(t, text, "Carried by the proof")
		if !slices.Equal(section, lines) {
			t.Errorf("%s: the carried section reads\n%s\nwant\n%s", c.Proof, strings.Join(section, "\n"), strings.Join(lines, "\n"))
		}
		for part, value := range leaves {
			hexValue, ok := value.(string)
			if !ok || !strings.HasSuffix(part, "/raw_transaction") && !strings.HasSuffix(part, "/txoutproof") {
				continue
			}
			wholeHex++
			if !slices.Contains(section, textRow(part, hexValue)) {
				t.Errorf("%s: the text does not give the %d hex digits at %s whole", c.Proof, len(hexValue), part)
			}
		}
	}
	if wholeHex < 8 {
		t.Fatalf("only %d raw_transaction and txoutproof values checked in the text; the test would prove little", wholeHex)
	}
	for _, class := range []string{"PINNED", "CHECKED", "PATH INPUT", "PREIMAGE", "CHAIN POINTER", "BLOCK HEIGHT", "SUBMITTED", "RECORDED"} {
		if classes[class] == 0 {
			t.Fatalf("no proof carries a %s value; classes %v", class, classes)
		}
	}
	if checkedProofs < 100 {
		t.Fatalf("only %d proofs pass SHAPE; the test would prove little", checkedProofs)
	}
}

// inspect gives the constructions mode SHAPE found, and a digest only for
// the full text, which this run computed: a proof that names the accepted
// text by reference is mode reference with no digest, one that carries
// none is mode none, and one that names a text no verifier accepts is
// SHAPE's construction_mismatch. Mutants killed: Shape giving the run the
// full mode whatever check 6 found; the reference's digest, the proof's,
// given as this run's.
func TestCLI_Inspect_TheConstructionsMode(t *testing.T) {
	full, _, code := inspectJSON(t, soundItem())
	if code != 0 || full.Proof.Constructions.Mode != "full" || len(full.Proof.Constructions.Digest) != 64 {
		t.Fatalf("full: exit %d, %+v", code, full.Proof)
	}
	doc := vectorDocument(t, "positive/item.proof.json").(map[string]any)
	reference := func(d string) string { return "https://www.truestamp.com/.well-known/constructions-" + d + ".json" }
	for _, c := range []struct {
		member any
		want   InspectConstructions
		code   string
	}{
		{nil, InspectConstructions{Mode: "none"}, ""},
		{reference(full.Proof.Constructions.Digest), InspectConstructions{Mode: "reference"}, ""},
		{reference(strings.Repeat("0", 64)), InspectConstructions{}, "construction_mismatch"},
	} {
		changed := map[string]any{}
		for k, val := range doc["proof"].(map[string]any) {
			changed[k] = val
		}
		delete(changed, "constructions")
		if c.member != nil {
			changed["constructions"] = c.member
		}
		file := writeProof(t, map[string]any{"proof": changed, "signature": doc["signature"]})
		got, _, code := inspectJSON(t, file)
		switch {
		case c.code != "":
			if code != 1 || got.Rejection == nil || got.Rejection.Code != c.code {
				t.Errorf("%v: exit %d, %+v", c.member, code, got.Rejection)
			}
		case code != 0 || got.Proof == nil || got.Proof.Constructions != c.want:
			t.Errorf("%v: exit %d, %+v, want %+v", c.member, code, got.Proof, c.want)
		}
	}
}

// writeProof writes doc as JSON to a new file and gives its path.
func writeProof(t *testing.T, doc any) string {
	t.Helper()
	data, err := json.Marshal(doc)
	if err != nil {
		t.Fatal(err)
	}
	file := filepath.Join(t.TempDir(), "proof.json")
	if err := os.WriteFile(file, data, 0o600); err != nil {
		t.Fatal(err)
	}
	return file
}

// inspect --json writes a claims member name with each code point of the
// list SPEC.md section 3.7's SUBMITTED row escapes at least (U+061C,
// U+200B to U+200F, U+202A to U+202E, U+2060 to U+2064, U+2066 to U+2069,
// U+FEFF), every code point of Unicode's Bidi_Control property among
// them, as its \uXXXX escape, the same JSON value, and every other code
// point as itself, its neighbours outside the list among them; the text
// quotes the name with each escaped.
// Mutants killed: the escape left out (claimsName without its
// MarshalJSON); a range cut short at its upper end (U+200F, U+202E, U+2064
// or U+2069 written as itself); U+061C ARABIC LETTER MARK left out of the
// list and written raw.
func TestCLI_Inspect_EscapesClaimsMemberNames(t *testing.T) {
	var listed, neighbours []rune
	for _, span := range [][2]rune{{0x061C, 0x061C}, {0x200B, 0x200F}, {0x202A, 0x202E}, {0x2060, 0x2064}, {0x2066, 0x2069}, {0xFEFF, 0xFEFF}} {
		for r := span[0]; r <= span[1]; r++ {
			listed = append(listed, r)
		}
	}
	neighbours = []rune{0x061B, 0x061D, 0x200A, 0x2010, 0x2070, 0xFEFE}
	name := "a" + string(listed) + "b" + string(neighbours)
	var wantJSON strings.Builder
	wantJSON.WriteString(`"a`)
	for _, r := range listed {
		fmt.Fprintf(&wantJSON, `\u%04x`, r)
	}
	wantJSON.WriteString("b" + string(neighbours) + `"`)

	doc := vectorDocument(t, "positive/item.proof.json").(map[string]any)
	proof := doc["proof"].(map[string]any)
	subject := proof["subject"].(map[string]any)
	claims := map[string]any{name: "x"}
	for k, v := range subject["claims"].(map[string]any) {
		claims[k] = v
	}
	changedSubject := map[string]any{}
	for k, v := range subject {
		changedSubject[k] = v
	}
	changedSubject["claims"] = claims
	changed := map[string]any{}
	for k, v := range proof {
		changed[k] = v
	}
	changed["subject"] = changedSubject
	file := writeProof(t, map[string]any{"proof": changed, "signature": doc["signature"]})

	out, code := runInspectCLI(t, file, "--json")
	if code != 0 {
		t.Fatalf("exit %d\n%s", code, out)
	}
	if !strings.Contains(out, wantJSON.String()) {
		t.Errorf("--json lacks the escaped name %s", wantJSON.String())
	}
	for _, r := range listed {
		if bytes.ContainsRune([]byte(out), r) {
			t.Errorf("--json writes U+%04X as itself", r)
		}
	}
	got, _, _ := inspectJSON(t, file)
	names, _ := atPointer(map[string]any{"subject": got.Proof.Subject}, "/subject/claims_members")
	if list, _ := names.([]any); !slices.Contains(list, any(name)) {
		t.Errorf("claims_members %v lacks the name as a JSON value", names)
	}
	text, _ := runInspectCLI(t, file)
	if !strings.Contains(text, strconv.Quote(name)) || strings.ContainsRune(text, 0x202E) {
		t.Errorf("the text does not quote the name %s with its escapes", strconv.Quote(name))
	}
}

// inspect reports an input over MAX_READ_OCTETS, a file of 1,048,576
// octets, as more than 262,144 octets, never as the count it read before
// refusing it: --json's octets is null and the text says "more than
// 262144 octets", with the bare rejection input_too_large at READ. A file
// of exactly MAX_READ_OCTETS is read whole, and its size given. Mutants
// killed: the count read, 262,145, given as its size; the bound taken as
// one octet less (a file of exactly 262,144 octets given no size).
func TestCLI_Inspect_AnOversizedInput(t *testing.T) {
	dir := t.TempDir()
	large := filepath.Join(dir, "large.proof.json")
	if err := os.WriteFile(large, bytes.Repeat([]byte(" "), 1<<20), 0o600); err != nil {
		t.Fatal(err)
	}
	got, raw, code := inspectJSON(t, large)
	if code != 1 || raw["octets"] != nil || got.Rejection == nil || got.Rejection.Code != "input_too_large" || got.Rejection.Step != "READ" {
		t.Errorf("--json: exit %d, octets %v, rejection %+v", code, raw["octets"], got.Rejection)
	}
	text, _ := runInspectCLI(t, large)
	if line := textRow("Source", large+" (more than 262144 octets)"); !strings.Contains(text, line+"\n") {
		t.Errorf("the text lacks %q\n%s", line, text)
	}
	exact := filepath.Join(dir, "exact.proof.json")
	if err := os.WriteFile(exact, bytes.Repeat([]byte(" "), specdata.MaxReadOctets), 0o600); err != nil {
		t.Fatal(err)
	}
	got, _, code = inspectJSON(t, exact)
	if code != 1 || got.Octets == nil || *got.Octets != specdata.MaxReadOctets || got.Rejection == nil || got.Rejection.Code != "malformed_json" {
		t.Errorf("a file of MAX_READ_OCTETS: exit %d, octets %v, rejection %+v", code, got.Octets, got.Rejection)
	}
}

// inspect's help says what it reports for a refused file: the reader's
// refusal as the verifier's bare rejection and a SHAPE failure as SHAPE's
// row, and that verify checks the environment marker, the key and the
// signature first and may refuse the same file earlier. On
// environment-null-under-test, inspect gives SHAPE's shape_invalid at
// /environment, and verify, under the case's configuration, the
// ENVIRONMENT row environment_mismatch, at the same part. Mutant killed:
// the help saying that a refused file gets the code verify would report.
func TestCLI_Inspect_HelpSaysVerifyMayRefuseEarlier(t *testing.T) {
	help, code := runCLIText(t, "inspect", "--help")
	if code != 0 {
		t.Fatalf("inspect --help: exit %d", code)
	}
	flat := strings.Join(strings.Fields(help), " ")
	for _, want := range []string{
		"A document the reader refuses is reported with the verifier's bare rejection, and one SHAPE refuses with SHAPE's row.",
		"verify checks the environment marker, the key and the signature before SHAPE, so it may refuse the same file at an earlier step, with another code.",
	} {
		if !strings.Contains(flat, want) {
			t.Errorf("the help lacks %q:\n%s", want, help)
		}
	}
	if strings.Contains(flat, "would report") {
		t.Errorf("the help still promises verify's code:\n%s", help)
	}
	c := vectorCase(t, "environment-null-under-test")
	got, _, code := inspectJSON(t, c.ProofPath())
	if code != 1 || got.Rejection == nil || got.Rejection.Code != "shape_invalid" || got.Rejection.Step != "SHAPE" ||
		got.Rejection.Part == nil || *got.Rejection.Part != "/environment" {
		t.Fatalf("inspect: exit %d, %+v", code, got.Rejection)
	}
	outcome := verifier.Verify(offlineTestConfig(t), func() ([]byte, error) { return strictjson.ReadFile(c.ProofPath()) })
	if outcome.Report == nil {
		t.Fatalf("verify gave no report: %+v", outcome.Rejection)
	}
	i := slices.IndexFunc(outcome.Report.Rows, func(r verifier.Row) bool { return r.Status == verifier.StatusFail })
	if i < 0 || outcome.Report.Rows[i].Step != "ENVIRONMENT" || outcome.Report.Rows[i].Code != specdata.ReasonEnvironmentMismatch ||
		outcome.Report.Rows[i].Part == nil || *outcome.Report.Rows[i].Part != "/environment" {
		t.Fatalf("verify's first failing row: %+v", outcome.Report.Rows)
	}
}

// inspect's help says every value is printed whole, a raw_transaction and
// a txoutproof among them, and what a handover_signature row gives.
// Mutants killed: the help saying the text gives a raw_transaction or a
// txoutproof as its length in octets; the help giving a handover_signature
// as a recomputed value beside the carried one.
func TestCLI_Inspect_HelpSaysEveryValueIsWhole(t *testing.T) {
	help, code := runCLIText(t, "inspect", "--help")
	if code != 0 {
		t.Fatalf("inspect --help: exit %d", code)
	}
	flat := strings.Join(strings.Fields(help), " ")
	for _, want := range []string{
		"Every value is printed whole, in the text and in --json, a raw_transaction and a txoutproof among them.",
		"a handover_signature is given as the carried signature beside the signed message, the hex of the 32 octets it signs, which is not a value to compare with it;",
	} {
		if !strings.Contains(flat, want) {
			t.Errorf("the help lacks %q:\n%s", want, help)
		}
	}
	if strings.Contains(flat, "length in octets") {
		t.Errorf("the help says a value is given as its length:\n%s", help)
	}
}

// rejectionStep panics for a code SPEC.md section 12.1's table lacks, and
// gives each bare code its table's step. Mutant killed: rejectionStep
// giving a step for a code the table lacks.
func TestRejectionStep_ReadsTheTable(t *testing.T) {
	for _, entry := range specdata.ReasonCodes() {
		if entry.Status != "bare" {
			continue
		}
		if got := rejectionStep(&verifier.Rejection{Code: entry.Code}); got != entry.Step {
			t.Errorf("%s: step %s, want %s", entry.Code, got, entry.Step)
		}
	}
	defer func() {
		if recover() == nil {
			t.Error("rejectionStep gave a step for a code the table lacks")
		}
	}()
	rejectionStep(&verifier.Rejection{Code: "no_such_code"})
}

// TestCLI_Inspect_HonorsCLIWideOutputSettings: --json and --silent are
// CLI-wide settings, resolvable from config.toml and from TRUESTAMP_JSON
// and TRUESTAMP_SILENT, not just from a flag typed on the line. inspect
// renders a record, like verify, so it reads them through outputMode.
// Mutant killed: inspect reading the raw --json and --silent flags.
func TestCLI_Inspect_HonorsCLIWideOutputSettings(t *testing.T) {
	proof := soundItem()

	t.Run("TRUESTAMP_JSON", func(t *testing.T) {
		cmd := exec.Command(binaryPath, "inspect", proof)
		cmd.Env = append(cleanEnv(), "TRUESTAMP_JSON=true")
		out, err := cmd.Output()
		if err != nil {
			t.Fatalf("inspect with TRUESTAMP_JSON: %v", err)
		}
		var got map[string]any
		if err := json.Unmarshal(out, &got); err != nil {
			t.Fatalf("TRUESTAMP_JSON=true did not produce JSON: %v\ngot: %s", err, firstLine(string(out)))
		}
		if got["source"] != proof {
			t.Errorf("JSON source is %v, want %s", got["source"], proof)
		}
	})

	t.Run("TRUESTAMP_SILENT", func(t *testing.T) {
		cmd := exec.Command(binaryPath, "inspect", proof)
		cmd.Env = append(cleanEnv(), "TRUESTAMP_SILENT=true")
		out, err := cmd.Output()
		if err != nil {
			t.Fatalf("inspect with TRUESTAMP_SILENT: %v", err)
		}
		if len(out) != 0 {
			t.Errorf("TRUESTAMP_SILENT=true still printed %d bytes:\n%s", len(out), firstLine(string(out)))
		}
	})

	// Asking for both is incoherent whatever the two settings came from;
	// inspect answers exactly as verify does.
	t.Run("json plus silent is rejected, the same way verify rejects it", func(t *testing.T) {
		for _, args := range [][]string{
			{"inspect", proof, "--json"},
			{"verify", proof, "--offline", "--json"},
		} {
			cmd := exec.Command(binaryPath, args...)
			cmd.Env = append(cleanEnv(), "TRUESTAMP_SILENT=true")
			out, err := cmd.CombinedOutput()
			if err == nil {
				t.Errorf("%v with TRUESTAMP_SILENT: expected an error, got none", args)
				continue
			}
			if !strings.Contains(string(out), "mutually exclusive") {
				t.Errorf("%v with TRUESTAMP_SILENT: got %q, want a mutual-exclusion error", args, firstLine(string(out)))
			}
		}
	})
}
