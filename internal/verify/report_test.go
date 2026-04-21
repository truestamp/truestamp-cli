// Copyright (c) 2021-2026 Truestamp, Inc.
// SPDX-License-Identifier: MIT

package verify

import (
	"encoding/json"
	"testing"
)

func TestStatus_MarshalJSON(t *testing.T) {
	for s, str := range statusStrings {
		data, err := json.Marshal(s)
		if err != nil {
			t.Errorf("MarshalJSON(%d): %v", s, err)
			continue
		}
		if string(data) != `"`+str+`"` {
			t.Errorf("MarshalJSON(%d): got %s, want %q", s, data, str)
		}
	}
}

func TestStatus_MarshalJSON_Unknown(t *testing.T) {
	unknown := Status(99)
	if _, err := json.Marshal(unknown); err == nil {
		t.Error("unknown status should fail to marshal")
	}
}

func TestStatus_UnmarshalJSON(t *testing.T) {
	cases := map[string]Status{
		`"pass"`: StatusPass,
		`"fail"`: StatusFail,
		`"skip"`: StatusSkip,
		`"warn"`: StatusWarn,
		`"info"`: StatusInfo,
	}
	for in, want := range cases {
		var got Status
		if err := json.Unmarshal([]byte(in), &got); err != nil {
			t.Errorf("Unmarshal(%s): %v", in, err)
			continue
		}
		if got != want {
			t.Errorf("Unmarshal(%s): got %d, want %d", in, got, want)
		}
	}
}

func TestStatus_UnmarshalJSON_UnknownMapsToInfo(t *testing.T) {
	var s Status
	if err := json.Unmarshal([]byte(`"future_status"`), &s); err != nil {
		t.Fatal(err)
	}
	if s != StatusInfo {
		t.Errorf("unknown status should map to info, got %d", s)
	}
}

func TestStatus_UnmarshalJSON_BadJSON(t *testing.T) {
	var s Status
	if err := json.Unmarshal([]byte(`{`), &s); err == nil {
		t.Error("malformed JSON should error")
	}
}

func TestStatusFromString(t *testing.T) {
	for str, want := range statusFromString {
		got, err := StatusFromString(str)
		if err != nil || got != want {
			t.Errorf("StatusFromString(%q) = (%d, %v), want (%d, nil)", str, got, err, want)
		}
	}
	if _, err := StatusFromString("unknown"); err == nil {
		t.Error("unknown string should error")
	}
}

func TestReport_Counts(t *testing.T) {
	r := &Report{
		Steps: []Step{
			{Status: StatusPass},
			{Status: StatusPass},
			{Status: StatusFail},
			{Status: StatusWarn},
			{Status: StatusInfo},
			{Status: StatusSkip},
		},
	}
	if c := r.PassCount(); c != 2 {
		t.Errorf("PassCount: got %d, want 2", c)
	}
	if c := r.FailedCount(); c != 1 {
		t.Errorf("FailedCount: got %d, want 1", c)
	}
	if c := r.WarnCount(); c != 1 {
		t.Errorf("WarnCount: got %d, want 1", c)
	}
	if c := r.InfoCount(); c != 1 {
		t.Errorf("InfoCount: got %d, want 1", c)
	}
	if c := r.SkipCount(); c != 1 {
		t.Errorf("SkipCount: got %d, want 1", c)
	}
	if r.Passed() {
		t.Error("Passed: should be false when any step failed")
	}
}

func TestReport_Passed_AllPassMulti(t *testing.T) {
	r := &Report{Steps: []Step{{Status: StatusPass}, {Status: StatusPass}}}
	if !r.Passed() {
		t.Error("Passed: should be true when all pass")
	}
}

func TestReport_CountsStruct(t *testing.T) {
	r := &Report{Steps: []Step{
		{Status: StatusPass},
		{Status: StatusFail},
		{Status: StatusWarn},
		{Status: StatusSkip},
		{Status: StatusInfo},
	}}
	c := r.Counts()
	if c.Passed != 1 || c.Failed != 1 || c.Warned != 1 || c.Skipped != 1 || c.Info != 1 {
		t.Errorf("Counts: %+v", c)
	}
	// Total excludes info.
	if c.Total != 4 {
		t.Errorf("Total: got %d, want 4", c.Total)
	}
}

func TestReport_HashMatched(t *testing.T) {
	r := &Report{HashProvided: "abcd", Steps: []Step{
		{Group: "Hash Comparison", Status: StatusPass},
	}}
	if !r.HashMatched() {
		t.Error("HashMatched should be true")
	}

	r2 := &Report{Steps: []Step{{Group: "Hash Comparison", Status: StatusPass}}}
	if r2.HashMatched() {
		t.Error("HashMatched should be false when HashProvided is empty")
	}

	r3 := &Report{HashProvided: "abcd", Steps: []Step{
		{Group: "Hash Comparison", Status: StatusFail},
	}}
	if r3.HashMatched() {
		t.Error("HashMatched should be false when hash comparison failed")
	}
}

func TestReport_ProofPassedAndFailedCount(t *testing.T) {
	// A failure inside Hash Comparison shouldn't count as a proof failure.
	r := &Report{Steps: []Step{
		{Group: "Signing Key", Status: StatusPass},
		{Group: "Hash Comparison", Status: StatusFail},
	}}
	if !r.ProofPassed() {
		t.Error("ProofPassed should ignore hash-comparison failure")
	}
	if got := r.ProofFailedCount(); got != 0 {
		t.Errorf("ProofFailedCount: got %d, want 0", got)
	}

	r2 := &Report{Steps: []Step{
		{Group: "Signing Key", Status: StatusFail},
		{Group: "Hash Comparison", Status: StatusFail},
	}}
	if r2.ProofPassed() {
		t.Error("ProofPassed should be false for non-hash failures")
	}
	if got := r2.ProofFailedCount(); got != 1 {
		t.Errorf("ProofFailedCount: got %d, want 1", got)
	}
}

func TestHexToBase64(t *testing.T) {
	cases := []struct {
		in, want string
	}{
		{"", ""},
		{"deadbeef", "3q2+7w=="},
		{"not-hex!", "not-hex!"}, // invalid hex → returned as-is
	}
	for _, c := range cases {
		if got := HexToBase64(c.in); got != c.want {
			t.Errorf("HexToBase64(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}
