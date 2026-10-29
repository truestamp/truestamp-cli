// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: Apache-2.0

// The report of SPEC.md section 10.5 and the bare rejection of section
// 10.2 (the gate) through the cases: each case's output as the verifier
// writes it, with JCS, read back here and held whole to the case and to
// its configuration, every member SPEC.md section 10.5 gives, not only
// what section 13.1 compares.

package vectorcases_test

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"slices"
	"testing"
	"time"

	"github.com/truestamp/truestamp-cli/internal/vectorcases"
	"github.com/truestamp/truestamp-cli/internal/verifier"
)

// reportMembers are the members of a report, SPEC.md section 10.5's
// table, in its order, which is JCS's; code is present exactly when the
// label is failed.
var reportMembers = []string{"attribution", "code", "environment", "label", "rows", "subject_id",
	"submitted_after", "submitted_before", "trust_root", "type", "verified", "window"}

// memberNames returns the names of the members of the JSON object encoded
// holds, in the order it writes them.
func memberNames(t *testing.T, encoded []byte) []string {
	t.Helper()
	decoder := json.NewDecoder(bytes.NewReader(encoded))
	if token, err := decoder.Token(); err != nil || token != json.Delim('{') {
		t.Fatalf("not an object: %s (%v)", encoded, err)
	}
	var names []string
	for decoder.More() {
		token, err := decoder.Token()
		if err != nil {
			t.Fatal(err)
		}
		names = append(names, token.(string))
		var skipped json.RawMessage
		if err := decoder.Decode(&skipped); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := decoder.Token(); err != nil {
		t.Fatal(err)
	}
	if _, err := decoder.Token(); !errors.Is(err, io.EOF) {
		t.Fatalf("more than one value: %s", encoded)
	}
	return names
}

// fileDigest is the lowercase hex SHA-256 of the bytes of the file at
// path.
func fileDigest(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

// expectedTrust is the attribution and trust_root, {digest, kind,
// location}, a case's configuration gives a run whose gate passes
// (SPEC.md sections 8.3 and 10.5), read here from the case and the files
// it names, never from the verifier: keyless mode is unattributed under
// the kind none; test or a keyring path is a supplied keyring, keyring:<d>
// with d the SHA-256 of the file's bytes; and no keyring option under prod
// is truestamp, offline under the compiled pin, whose bytes are
// spec/keyring.production.json's, and online under the keyring read from
// www.truestamp.com, which a replay holds as a value with no bytes, so its
// digest is null (SPEC.md section 13.2), as is an unavailable keyring's.
func expectedTrust(t *testing.T, c vectorcases.Case) (string, map[string]any) {
	t.Helper()
	keyring := c.Config.Keyring
	switch {
	case keyring != nil && *keyring == "none":
		return verifier.AttributionUnattributed, map[string]any{"digest": nil, "kind": "none", "location": nil}
	case keyring != nil && *keyring != "unavailable":
		file := *keyring
		if file == "test" {
			file = "keyring.test.json"
		}
		digest := fileDigest(t, vectorcases.Path(file))
		return "keyring:" + digest, map[string]any{"digest": digest, "kind": "supplied_keyring", "location": vectorcases.Path(file)}
	case c.Config.Online == nil:
		digest := fileDigest(t, filepath.Join(vectorcases.Dir(), "..", "spec", "keyring.production.json"))
		return verifier.AttributionTruestamp, map[string]any{"digest": digest, "kind": "compiled_pin", "location": nil}
	}
	return verifier.AttributionTruestamp, map[string]any{"digest": nil, "kind": "current_keyring", "location": verifier.CurrentKeyringURL}
}

// sameInstant reports whether two RFC 3339 times are one instant, as
// SPEC.md section 13.1 compares an edge's instant.
func sameInstant(t *testing.T, a, b string) bool {
	t.Helper()
	x, errX := time.Parse(time.RFC3339Nano, a)
	y, errY := time.Parse(time.RFC3339Nano, b)
	if errX != nil || errY != nil {
		t.Fatalf("an instant: %v, %v", errX, errY)
	}
	return x.Equal(y)
}

// checkReport holds the report of a run that passed steps 0 to 3, written
// with JCS and read back, to its case whole (SPEC.md sections 10.5 and
// 13.1): exactly the members of section 10.5, in JCS's order, code
// present exactly when the label is failed and then the code of the
// case's first failing row, and the code the case gives when it gives one;
// its label; every row, in order, each exactly
// {code, message, part, status, step, values} with code absent exactly on
// a pass, reduced to (code, part, status, step) and equal to the case's
// rows; each edge null or {basis, instant} as the case gives it, the
// instant compared as an instant, and null in every offline case and every
// failed one, each edge that is not null the object of the edge WINDOW
// drew; window established exactly when submitted_before is not null;
// verified exactly when the label is verified; environment the case's
// E_v; type and subject_id those the proof carries when the case's SHAPE
// row passes, and null otherwise; and attribution and trust_root those the
// case's configuration gives (expectedTrust), the attribution none when
// the case fails at the gate.
func checkReport(t *testing.T, c vectorcases.Case, r *verifier.Run) {
	t.Helper()
	encoded, err := verifier.NewReport(r).Encode()
	if err != nil {
		t.Fatalf("the report: %v", err)
	}
	failed := c.Expected.Label == verifier.LabelFailed
	want := slices.DeleteFunc(slices.Clone(reportMembers), func(name string) bool { return name == "code" && !failed })
	if got := memberNames(t, encoded); !slices.Equal(got, want) {
		t.Fatalf("the report's members %q, want %q", got, want)
	}
	var report map[string]any
	if err := json.Unmarshal(encoded, &report); err != nil {
		t.Fatal(err)
	}
	if report["label"] != c.Expected.Label {
		t.Fatalf("label %v, want %s", report["label"], c.Expected.Label)
	}
	if i := slices.IndexFunc(c.Expected.Rows, failing); failed && (i < 0 || report["code"] != *c.Expected.Rows[i].Code) {
		t.Fatalf("code %v; the case's first failing row is at %d", report["code"], i)
	}
	if c.Expected.Code != "" && report["code"] != c.Expected.Code {
		t.Fatalf("code %v; the case gives the code %s", report["code"], c.Expected.Code)
	}

	var rows []json.RawMessage
	if err := json.Unmarshal(mustMarshal(t, report["rows"]), &rows); err != nil {
		t.Fatal(err)
	}
	var reduced []vectorcases.Row
	for _, raw := range rows {
		var row vectorcases.Row
		var members map[string]any
		if err := json.Unmarshal(raw, &members); err != nil {
			t.Fatal(err)
		}
		names := []string{"code", "message", "part", "status", "step", "values"}
		if members["status"] == "pass" {
			names = names[1:]
		}
		if got := memberNames(t, raw); !slices.Equal(got, names) {
			t.Fatalf("a row's members %q, want %q: %s", got, names, raw)
		}
		if _, isObject := members["values"].(map[string]any); !isObject {
			t.Fatalf("a row's values: %s", raw)
		}
		if code, ok := members["code"].(string); ok {
			row.Code = &code
		}
		if part, ok := members["part"].(string); ok {
			row.Part = &part
		} else if members["part"] != nil {
			t.Fatalf("a row's part: %s", raw)
		}
		row.Status, _ = members["status"].(string)
		row.Step, _ = members["step"].(string)
		reduced = append(reduced, row)
	}
	if !rowsEqual(reduced, c.Expected.Rows) {
		t.Fatalf("the report's rows %v, want %v", reduced, c.Expected.Rows)
	}

	for _, e := range []struct {
		name  string
		want  json.RawMessage
		drawn *verifier.Edge
	}{{"submitted_after", c.Expected.SubmittedAfter, r.SubmittedAfter}, {"submitted_before", c.Expected.SubmittedBefore, r.SubmittedBefore}} {
		got, _ := report[e.name].(map[string]any)
		want := expectedEdgeOf(t, e.want)
		switch {
		case report[e.name] == nil && want == nil:
			continue
		case got == nil || want == nil || e.drawn == nil:
			t.Fatalf("%s: %v, want %+v", e.name, report[e.name], want)
		case got["basis"] != want.Basis || !sameInstant(t, got["instant"].(string), want.Instant) || !sameJCS(t, normalized(t, got), e.drawn.Value()):
			t.Fatalf("%s: %v, want %+v, drawn %v", e.name, got, want, e.drawn.Value())
		}
	}
	if (report["window"] == verifier.WindowEstablished) != (report["submitted_before"] != nil) ||
		(report["window"] != verifier.WindowEstablished && report["window"] != verifier.WindowNotEstablished) {
		t.Fatalf("window %v, submitted_before %v", report["window"], report["submitted_before"])
	}
	if report["verified"] != (c.Expected.Label == verifier.LabelVerified) || report["environment"] != c.Config.Environment {
		t.Fatalf("verified %v, environment %v", report["verified"], report["environment"])
	}

	typ, subject := any(nil), any(nil)
	if slices.ContainsFunc(c.Expected.Rows, func(row vectorcases.Row) bool { return row.Step == "SHAPE" && row.Status == "pass" }) {
		typ, subject = typeAndSubject(proofOf(t, c))
	}
	if report["type"] != typ || report["subject_id"] != subject {
		t.Fatalf("type %v, subject_id %v; want %v, %v", report["type"], report["subject_id"], typ, subject)
	}
	attribution, trustRoot := expectedTrust(t, c)
	if gateFails(c.Expected.Rows) {
		attribution = verifier.AttributionNone
	}
	if report["attribution"] != attribution || !sameJCS(t, normalized(t, report["trust_root"]), trustRoot) {
		t.Fatalf("attribution %v, trust_root %v; want %s, %v", report["attribution"], report["trust_root"], attribution, trustRoot)
	}
}

// checkRejection holds a case's bare rejection, written with JCS and read
// back, to SPEC.md section 10.2 (the gate): exactly {code, message,
// sub_reason}, in that order, sub_reason present only for malformed_json
// and invalid_envelope, the code and sub_reason the case's.
func checkRejection(t *testing.T, c vectorcases.Case, rejection *verifier.Rejection) {
	t.Helper()
	encoded, err := rejection.Encode()
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"code", "message"}
	if c.Expected.SubReason != "" {
		want = append(want, "sub_reason")
	}
	if got := memberNames(t, encoded); !slices.Equal(got, want) {
		t.Fatalf("the rejection's members %q, want %q", got, want)
	}
	var value map[string]any
	if err := json.Unmarshal(encoded, &value); err != nil {
		t.Fatal(err)
	}
	subReason, _ := value["sub_reason"].(string)
	message, _ := value["message"].(string)
	if value["code"] != c.Expected.Code || subReason != c.Expected.SubReason || message == "" {
		t.Fatalf("the rejection %s; want %s (%s)", encoded, c.Expected.Code, c.Expected.SubReason)
	}
	if (c.Expected.Code == "malformed_json" || c.Expected.Code == "invalid_envelope") != (c.Expected.SubReason != "") {
		t.Fatalf("the case's sub_reason %q for %s", c.Expected.SubReason, c.Expected.Code)
	}
}

// normalized is v as JSON gives it back, its numbers as float64s, so that
// sameJCS can compare it with a value written from the verifier's own.
func normalized(t *testing.T, v any) any {
	t.Helper()
	var out any
	if err := json.Unmarshal(mustMarshal(t, v), &out); err != nil {
		t.Fatal(err)
	}
	return out
}

func mustMarshal(t *testing.T, v any) []byte {
	t.Helper()
	data, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	return data
}

// EXPECTED_HASH failing after WINDOW passed (SPEC.md section 10.2, steps
// 13 and 14): each online case that is not failed and whose WINDOW drew a
// submitted-before edge, the 41 of them, run again with an expected hash
// that is not hex of any listed type's length, so its EXPECTED_HASH row
// is fail expected_hash_mismatch at /subject/claims/hash, whatever the
// proof's type, gives every other row as its case does, the label failed,
// the code expected_hash_mismatch, both edges null, window not_established
// and verified false, while the run keeps the edges WINDOW drew. Mutants
// killed: the report's edges written whatever the label (each report has
// its edges); the window read from the run's edge (each is established);
// verified set from the run's submitted-before edge (each is verified).
func TestCases_ExpectedHashFailsAfterWindow(t *testing.T) {
	cases, err := vectorcases.Load()
	if err != nil {
		t.Fatal(err)
	}
	seen := 0
	for _, c := range cases {
		if c.Config.Online == nil || c.Expected.Bare() || c.Expected.Label == verifier.LabelFailed || expectedEdgeOf(t, c.Expected.SubmittedBefore) == nil {
			continue
		}
		seen++
		given := "00"
		c.Config.ExpectedHash = &given
		got := run(t, c)
		r := got.run
		want := slices.Clone(c.Expected.Rows)
		code, part := "expected_hash_mismatch", "/subject/claims/hash"
		if last := want[len(want)-1]; last.Step != "EXPECTED_HASH" || last.Code == nil || *last.Code != "expected_hash_not_checked" {
			t.Fatalf("%s: the last row %v", c.ID, last)
		}
		want[len(want)-1] = vectorcases.Row{Code: &code, Part: &part, Status: "fail", Step: "EXPECTED_HASH"}
		report := verifier.NewReport(r)
		encoded, err := report.Encode()
		if err != nil {
			t.Fatal(err)
		}
		var value map[string]any
		if err := json.Unmarshal(encoded, &value); err != nil {
			t.Fatal(err)
		}
		if !rowsEqual(got.rows, want) || value["label"] != "failed" || value["code"] != code || value["submitted_after"] != nil ||
			value["submitted_before"] != nil || value["window"] != "not_established" || value["verified"] != false || r.SubmittedBefore == nil {
			t.Errorf("%s: rows %v, report %s", c.ID, got.rows, encoded)
		}
	}
	if seen != 41 {
		t.Fatalf("%d cases, want the 41 online cases with a submitted-before edge that are not failed", seen)
	}
}
