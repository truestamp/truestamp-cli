// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: Apache-2.0

package verifier

import (
	"slices"
	"strings"
	"testing"

	"github.com/truestamp/truestamp-cli/internal/specdata"
	"github.com/truestamp/truestamp-cli/internal/vectorcases"
)

func panics(f func()) (panicked bool) {
	defer func() { panicked = recover() != nil }()
	f()
	return false
}

// A row takes its status from SPEC.md section 12.1's table, and no row
// carries a bare code, a code the table lacks, or a step that reports no
// rows. Mutants killed: NewRow without its bare-code refusal; NewRow
// giving every code the status fail.
func TestNewRow_StatusComesFromTheTable(t *testing.T) {
	for code, want := range map[specdata.Reason]Status{
		specdata.ReasonKeyBindingNotPerformed:    StatusSkip,
		specdata.ReasonNotAttributable:           StatusInfo,
		specdata.ReasonExpectedHashWeakAlgorithm: StatusWarn,
		specdata.ReasonSignatureInvalid:          StatusFail,
		"":                                       StatusPass,
	} {
		if got := NewRow("KEY_BINDING", code, nil, "m", nil).Status; got != want {
			t.Errorf("%q: %s, want %s", code, got, want)
		}
	}
	for name, f := range map[string]func(){
		"a bare code":     func() { NewRow("ENVELOPE", specdata.ReasonInvalidEnvelope, nil, "m", nil) },
		"an unknown code": func() { NewRow("ENVELOPE", "unsupported_layout", nil, "m", nil) },
		"VERDICT":         func() { PassRow("VERDICT", nil, "m", nil) },
		"an unknown step": func() { PassRow("LAYOUT", nil, "m", nil) },
	} {
		if !panics(f) {
			t.Errorf("%s: no panic", name)
		}
	}
	code, part, status, step := NewRow("ENVIRONMENT", specdata.ReasonEnvironmentMismatch, Part("/environment"), "m", nil).Reduce()
	if code == nil || *code != "environment_mismatch" || part == nil || *part != "/environment" || status != "fail" || step != "ENVIRONMENT" {
		t.Errorf("Reduce = %v %v %s %s", code, part, status, step)
	}
	if code, part, _, _ := PassRow("READ", nil, "m", nil).Reduce(); code != nil || part != nil {
		t.Errorf("a pass row reduces to %v %v", code, part)
	}
}

// A bare rejection carries one of the four bare codes, with a sub_reason
// exactly when the code has them, and one of its own. Mutant killed:
// NewRejection not refusing a sub_reason the code does not list.
func TestNewRejection_Refusals(t *testing.T) {
	for name, f := range map[string]func(){
		"a row code":                         func() { NewRejection(specdata.ReasonSignatureInvalid, "m", "") },
		"malformed_json without a sub":       func() { NewRejection(specdata.ReasonMalformedJSON, "m", "") },
		"configuration_error with a sub":     func() { NewRejection(specdata.ReasonConfigurationError, "m", specdata.SubReasonDocument) },
		"invalid_envelope, malformed's sub":  func() { InvalidEnvelope(specdata.SubReasonSyntax) },
		"malformed_json, invalid_envelope's": func() { MalformedJSON(specdata.SubReasonProof) },
	} {
		if !panics(f) {
			t.Errorf("%s: no panic", name)
		}
	}
	if r := InputTooLarge(); r.Code != specdata.ReasonInputTooLarge || !strings.Contains(r.Message, "262144") {
		t.Errorf("InputTooLarge = %+v", r)
	}
	if r := ConfigurationError("x"); r.Error() != "configuration_error: x" {
		t.Errorf("Error = %q", r.Error())
	}
}

// Pointers encode as RFC 6901 writes them, and compare in document order:
// a prefix first, names by UTF-16 code units, indices by value. Mutants
// killed: names compared as Go strings (UTF-8 order puts U+E000 before
// U+1F600); indices compared as their text (10 before 2); the prefix rule
// reversed.
func TestPointer_EncodeAndDocumentOrder(t *testing.T) {
	if got := (Pointer{Name("a/b"), Name("c~d"), Index(3)}).Encode(); got != "/a~1b/c~0d/3" {
		t.Errorf("Encode = %q", got)
	}
	if got := (Pointer{}).Encode(); got != "" {
		t.Errorf("the empty pointer = %q", got)
	}
	ordered := []Pointer{
		{},
		{Name("block")},
		{Name("block"), Name("record")},
		{Name("subject"), Index(2)},
		{Name("subject"), Index(10)},
		{Name("\U0001F600")},
		{Name("\ue000")},
	}
	for i := range ordered {
		for j := range ordered {
			if got, want := Compare(ordered[i], ordered[j]), compareInts(i, j); got != want {
				t.Errorf("Compare(%q, %q) = %d, want %d", ordered[i].Encode(), ordered[j].Encode(), got, want)
			}
		}
	}
	if !panics(func() { Index(-1) }) {
		t.Error("a negative index did not panic")
	}
}

func compareInts(i, j int) int {
	switch {
	case i < j:
		return -1
	case i > j:
		return 1
	}
	return 0
}

// SortBy is stable and FirstBy takes the earliest among equals. Mutant
// killed: SortBy through an unstable sort; FirstBy taking the last of two
// equal pointers.
func TestPointer_SortByAndFirstBy(t *testing.T) {
	type tagged struct {
		part Pointer
		tag  int
	}
	var list []tagged
	for i := range 40 {
		list = append(list, tagged{Pointer{Name([]string{"b", "a"}[i%2])}, i})
	}
	SortBy(list, func(x tagged) Pointer { return x.part })
	for i := 1; i < len(list); i++ {
		if Compare(list[i-1].part, list[i].part) == 0 && list[i-1].tag > list[i].tag {
			t.Fatalf("not stable at %d: %v", i, list)
		}
	}
	first, ok := FirstBy([]tagged{{Pointer{Name("b")}, 0}, {Pointer{Name("a")}, 1}, {Pointer{Name("a")}, 2}}, func(x tagged) Pointer { return x.part })
	if !ok || first.tag != 1 {
		t.Fatalf("FirstBy = %v", first)
	}
	if _, ok := FirstBy([]tagged{}, func(x tagged) Pointer { return x.part }); ok {
		t.Fatal("FirstBy of nothing")
	}
}

// EndRun adds one skip prior_failure row, part null, for each step after
// the last row's up to EXPECTED_HASH, and none for VERDICT. Mutant killed:
// EndRun adding a row for VERDICT.
func TestRun_EndRun(t *testing.T) {
	run := &Run{}
	run.AddRow(NewRow("ENVIRONMENT", specdata.ReasonEnvironmentMismatch, Part("/environment"), "m", nil))
	if !run.Failed() {
		t.Fatal("a failed row does not fail the run")
	}
	run.EndRun()
	var got []string
	for _, row := range run.Rows[1:] {
		if row.Code != specdata.ReasonPriorFailure || row.Part != nil || row.Status != StatusSkip {
			t.Errorf("row %+v", row)
		}
		got = append(got, row.Step)
	}
	if want := Steps()[5:14]; !slices.Equal(got, want) || !run.Ended {
		t.Fatalf("skipped %v, want %v", got, want)
	}
	if !slices.Equal(Steps(), vectorcases.Steps) {
		t.Fatalf("Steps = %v", Steps())
	}
}
