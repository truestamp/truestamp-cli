// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: MIT

package verify

import (
	"encoding/json"
	"strings"
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

// A status outside E.22's five is an error, not a value. Mapping it to
// info made it verdict-neutral: one word of server vocabulary drift
// turned a reported failure into a note and the run exited 0.
func TestStatus_UnmarshalJSON_UnknownIsAnError(t *testing.T) {
	var s Status
	if err := json.Unmarshal([]byte(`"future_status"`), &s); err == nil {
		t.Fatal("an unrecognized status string must not decode silently")
	}
	// Fail closed even for a caller that ignores the error.
	if got, err := StatusFromString("future_status"); err == nil || got != StatusFail {
		t.Errorf("StatusFromString(unknown) = (%d, %v), want (StatusFail, error)", got, err)
	}
}

// --- Fail-closed decoding of a remote verifier's step objects ---

func TestStep_UnmarshalJSON_MissingStatusIsAFailure(t *testing.T) {
	var s Step
	if err := json.Unmarshal([]byte(`{"group":"Proof Signature","category":"cryptographic","message":"Proof signature invalid (Ed25519)"}`), &s); err != nil {
		t.Fatal(err)
	}
	if s.Status != StatusFail {
		t.Fatalf("a step with no status key must not be scored as passing, got %d", s.Status)
	}
	if !strings.Contains(s.Message, "Proof signature invalid") {
		t.Errorf("the server's own message must survive: %q", s.Message)
	}
	if !strings.Contains(s.Message, "no status") {
		t.Errorf("the message must disclose why the step was graded fail: %q", s.Message)
	}
	if s.Group != "Proof Signature" || s.Category != CatCryptographic {
		t.Errorf("group/category dropped: %+v", s)
	}
}

func TestStep_UnmarshalJSON_UnknownStatusIsAFailure(t *testing.T) {
	cases := []string{
		`{"group":"Proof Signature","status":"catastrophe","message":"boom"}`,
		`{"group":"Proof Signature","status":"failed","message":"Ed25519 signature is INVALID"}`,
		`{"group":"Proof Signature","status":null,"message":"boom"}`,
		`{"group":"Proof Signature","status":7,"message":"boom"}`,
	}
	for _, in := range cases {
		var s Step
		if err := json.Unmarshal([]byte(in), &s); err != nil {
			t.Fatalf("%s: %v", in, err)
		}
		if s.Status != StatusFail {
			t.Errorf("%s: got status %d, want StatusFail", in, s.Status)
		}
		if !strings.Contains(s.Message, "cannot be read as having passed") {
			t.Errorf("%s: no disclosure in message %q", in, s.Message)
		}
	}
}

func TestStep_UnmarshalJSON_KnownStatusesRoundTrip(t *testing.T) {
	for str, want := range statusFromString {
		var s Step
		in := `{"group":"g","category":"cryptographic","status":"` + str + `","message":"m"}`
		if err := json.Unmarshal([]byte(in), &s); err != nil {
			t.Fatalf("%s: %v", in, err)
		}
		if s.Status != want {
			t.Errorf("%s: got %d, want %d", in, s.Status, want)
		}
		if s.Message != "m" {
			t.Errorf("%s: message annotated for a valid status: %q", in, s.Message)
		}
	}
}

// A hostile or broken server must not be able to splice an unbounded
// string into a rendered report through the status token.
func TestStep_UnmarshalJSON_BoundsTheEchoedToken(t *testing.T) {
	var s Step
	long := strings.Repeat("A", 500)
	if err := json.Unmarshal([]byte(`{"group":"g","status":"`+long+`","message":"m"}`), &s); err != nil {
		t.Fatal(err)
	}
	if len(s.Message) > 250 {
		t.Errorf("echoed token not bounded, message is %d bytes", len(s.Message))
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
	// Total counts every step. It used to exclude info, so the --json
	// `summary` did not reconcile with the `steps` array it summarizes:
	// on the Appendix D.4 bundle, total said 14 while steps carried 16.
	// E.22 defines five statuses; a total covering four of them is not a
	// total.
	if c.Total != len(r.Steps) {
		t.Errorf("Total: got %d, want %d (len(Steps))", c.Total, len(r.Steps))
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

// --- F18: check() must not route one message to both arms ---

func TestReportCheck_FailArmUsesFailMessage(t *testing.T) {
	cases := []struct {
		name string
		ok   bool
		want Status
		msg  string
	}{
		{"pass arm", true, StatusPass, "derived root matches"},
		{"fail arm", false, StatusFail, "derived root does not match block merkle root"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			r := &Report{}
			r.check(groupInclusion, CatCryptographic, c.ok,
				"derived root matches", "derived root does not match block merkle root")
			if len(r.Steps) != 1 {
				t.Fatalf("expected exactly one step, got %d", len(r.Steps))
			}
			got := r.Steps[0]
			if got.Status != c.want {
				t.Errorf("status: got %v, want %v", got.Status, c.want)
			}
			if got.Message != c.msg {
				t.Errorf("message: got %q, want %q", got.Message, c.msg)
			}
			if got.Group != groupInclusion || got.Category != CatCryptographic {
				t.Errorf("group/category changed by arm: %q / %q", got.Group, got.Category)
			}
		})
	}
}

// TestReportCheck_FailMessageIsIndependentOfPassMessage pins the
// primitive half of F18: check() must publish its fail argument on the
// false arm, never its pass argument. It says nothing about any
// production message — the pair below is a literal.
//
// The pipeline-wide half of the property, that no fail message a caller
// actually sees reads as a positive assertion (E.22), is
// TestFailMessages_NeverReadAsPositiveAssertions in
// status_pinning_test.go. This comment previously named a
// TestReportCheck_NoFailMessageReadsAsAssertion that does not exist and
// claimed the pipeline call sites were covered elsewhere; they were not.
func TestReportCheck_FailMessageIsIndependentOfPassMessage(t *testing.T) {
	r := &Report{}
	r.check(groupProof, CatCryptographic, false,
		"Proof signature valid (Ed25519)", "Proof signature invalid (Ed25519)")
	msg := r.Steps[0].Message
	if strings.Contains(msg, "valid (Ed25519)") && !strings.Contains(msg, "invalid") {
		t.Errorf("fail message reads as a positive assertion: %q", msg)
	}
}

// --- F9: the printed verdict and the exit code can never contradict ---

func TestReportVerdict_MatchesPassed(t *testing.T) {
	cases := []struct {
		name string
		r    *Report
		want Verdict
	}{
		{
			name: "clean proof, no expected hash",
			r:    &Report{Steps: []Step{{Group: groupSigningKey, Status: StatusPass}}},
			want: VerdictVerified,
		},
		{
			name: "clean proof, expected hash matched",
			r: &Report{HashProvided: "aabb", Steps: []Step{
				{Group: groupHashComparison, Status: StatusPass},
			}},
			want: VerdictFullyVerified,
		},
		{
			name: "expected hash mismatched, proof sound",
			r: &Report{HashProvided: "deadbeef", Steps: []Step{
				{Group: groupSigningKey, Status: StatusPass},
				{Group: groupHashComparison, Status: StatusFail},
			}},
			want: VerdictHashMismatch,
		},
		{
			name: "a cryptographic step failed",
			r: &Report{Steps: []Step{
				{Group: groupProof, Status: StatusFail},
			}},
			want: VerdictFailed,
		},
		{
			name: "skips alone never fail a report",
			r: &Report{Steps: []Step{
				{Group: groupSigningKey, Status: StatusSkip},
				{Group: groupStellar, Status: StatusSkip},
			}},
			want: VerdictVerified,
		},
		{
			// The F9 regression: a Hash Comparison failure raised for a
			// reason that is NOT an expected-hash mismatch (no hash was
			// supplied at all — e.g. a step handed to us by the remote
			// verifier). ProofPassed() ignores it, so the verdict used
			// to print VERIFIED while the exit code said 1.
			name: "hash-comparison failure with no expected hash supplied",
			r: &Report{Steps: []Step{
				{Group: groupSigningKey, Status: StatusPass},
				{Group: groupHashComparison, Status: StatusFail},
			}},
			want: VerdictFailed,
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := c.r.Verdict(); got != c.want {
				t.Errorf("Verdict: got %d, want %d", got, c.want)
			}
			// The invariant: a passing verdict if and only if Passed().
			passing := c.r.Verdict() == VerdictVerified || c.r.Verdict() == VerdictFullyVerified
			if passing != c.r.Passed() {
				t.Errorf("verdict %d contradicts Passed()=%v (exit code)", c.r.Verdict(), c.r.Passed())
			}
			// And the --json result must agree with both.
			result := computeResult(c.r)
			resultPassing := result == "verified" || result == "fully_verified"
			if resultPassing != c.r.Passed() {
				t.Errorf("--json result %q contradicts Passed()=%v", result, c.r.Passed())
			}
		})
	}
}

func TestComputeResult_Vocabulary(t *testing.T) {
	cases := map[Verdict]string{
		VerdictVerified:      "verified",
		VerdictFullyVerified: "fully_verified",
		VerdictHashMismatch:  "hash_mismatch",
		VerdictFailed:        "failed",
	}
	reports := map[Verdict]*Report{
		VerdictVerified:      {Steps: []Step{{Status: StatusPass}}},
		VerdictFullyVerified: {HashProvided: "aabb", Steps: []Step{{Group: groupHashComparison, Status: StatusPass}}},
		VerdictHashMismatch:  {HashProvided: "aabb", Steps: []Step{{Group: groupHashComparison, Status: StatusFail}}},
		VerdictFailed:        {Steps: []Step{{Group: groupProof, Status: StatusFail}}},
	}
	for v, want := range cases {
		if got := computeResult(reports[v]); got != want {
			t.Errorf("verdict %d: result %q, want %q", v, got, want)
		}
	}
}

func TestHexToBase64(t *testing.T) {
	cases := []struct {
		in, want string
	}{
		{"", ""},
		{"deadbeef", "3q2+7w=="},
		{"not-hex!", "not-hex!"}, // invalid hex → returned as-is
		// E.4's lowercase rule reaches the display surfaces too. Go's
		// hex.DecodeString is case-insensitive, so "DEADBEEF" used to
		// re-encode to the same "3q2+7w==" a conforming bundle produces:
		// the presenter and the --json committed_hash_base64 field
		// laundered the very defect the verification steps now fail on,
		// and a reader comparing the base64 line against a chain explorer
		// saw agreement where the bundle was non-conformant. Echoing the
		// offending value back keeps both surfaces telling one story.
		{"DEADBEEF", "DEADBEEF"},
		{"deadbeeF", "deadbeeF"},
	}
	for _, c := range cases {
		if got := HexToBase64(c.in); got != c.want {
			t.Errorf("HexToBase64(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

// --- F12: --json carries the complete step record ---

func TestBuildJSONOutput_StepsArray_IncludesSkipAndInfo(t *testing.T) {
	r := &Report{Steps: []Step{
		{Group: groupSigningKey, Category: CatCryptographic, Status: StatusPass, Message: "pass"},
		{Group: groupProof, Category: CatCryptographic, Status: StatusFail, Message: "fail"},
		{Group: groupStellar, Category: CatBlockchain, Status: StatusSkip, Message: "skip"},
		{Group: groupSubjectData, Category: CatDataIntegrity, Status: StatusWarn, Message: "warn"},
		{Group: groupTemporalInfo, Category: CatTiming, Status: StatusInfo, Message: "info"},
	}}
	out := BuildJSONOutput(r)
	if len(out.Steps) != 5 {
		t.Fatalf("steps: got %d entries, want 5 (skip and info rows must survive)", len(out.Steps))
	}
	seen := map[Status]bool{}
	for _, s := range out.Steps {
		seen[s.Status] = true
	}
	for _, want := range []Status{StatusPass, StatusFail, StatusSkip, StatusWarn, StatusInfo} {
		if !seen[want] {
			t.Errorf("status %v missing from steps array", want)
		}
	}
	// The filtered views stay filtered against info (which belongs to
	// verification_notes) and against passes, but skips are issues in
	// both surfaces — see TestSurfaces_IssuesAndNotesAgree.
	for _, iss := range out.Issues {
		if iss.Message == "info" || iss.Message == "pass" {
			t.Errorf("info/pass leaked into issues: %+v", iss)
		}
	}
}

func TestBuildJSONOutput_StepsArray_CategoryOrder(t *testing.T) {
	// Deliberately emitted in reverse of E.22's report order, with two
	// blockchain rows whose relative order carries meaning.
	r := &Report{Steps: []Step{
		{Group: groupEpoch, Category: CatBlockchain, Status: StatusPass, Message: "epoch 0"},
		{Group: groupEpoch, Category: CatBlockchain, Status: StatusPass, Message: "epoch 1"},
		{Group: groupSubmissionWindow, Category: CatTiming, Status: StatusPass, Message: "timing"},
		{Group: groupStructure, Category: CatStructural, Status: StatusPass, Message: "structural"},
		{Group: groupProof, Category: CatCryptographic, Status: StatusPass, Message: "cryptographic"},
		{Group: groupHashComparison, Category: CatDataIntegrity, Status: StatusPass, Message: "data integrity"},
	}}
	out := BuildJSONOutput(r)

	wantCats := []string{
		CatDataIntegrity, CatCryptographic, CatStructural, CatTiming, CatBlockchain, CatBlockchain,
	}
	if len(out.Steps) != len(wantCats) {
		t.Fatalf("steps: got %d, want %d", len(out.Steps), len(wantCats))
	}
	for i, want := range wantCats {
		if out.Steps[i].Category != want {
			t.Errorf("steps[%d].category: got %q, want %q", i, out.Steps[i].Category, want)
		}
	}
	// Emit order within a category must survive the sort: "epoch 0"
	// before "epoch 1" is cx order and is meaning-carrying.
	if out.Steps[4].Message != "epoch 0" || out.Steps[5].Message != "epoch 1" {
		t.Errorf("emit order within a category was not preserved: %q then %q",
			out.Steps[4].Message, out.Steps[5].Message)
	}
}

func TestBuildJSONOutput_StepsArray_EmptyCategoryCoerced(t *testing.T) {
	r := &Report{Steps: []Step{{Group: "Whatever", Status: StatusPass, Message: "no category"}}}
	out := BuildJSONOutput(r)
	if len(out.Steps) != 1 {
		t.Fatalf("steps: got %d, want 1", len(out.Steps))
	}
	if out.Steps[0].Category != CatStructural {
		t.Errorf("empty category: got %q, want %q", out.Steps[0].Category, CatStructural)
	}
	// The Report itself must not be mutated by building the DTO.
	if r.Steps[0].Category != "" {
		t.Errorf("BuildJSONOutput mutated the source Report: %q", r.Steps[0].Category)
	}
}

func TestBuildJSONOutput_StepsArray_MarshalsAsEmptyArray(t *testing.T) {
	data, err := json.Marshal(BuildJSONOutput(&Report{}))
	if err != nil {
		t.Fatalf("marshal: %s", err)
	}
	if !strings.Contains(string(data), `"steps":[]`) {
		t.Errorf(`expected "steps":[] for a report with no steps, got: %s`, data)
	}
}

// TestBuildJSONOutput_StepsArray_UnknownCategorySortsLast guards the
// remote path, where step categories arrive from the server and may name
// a category this build does not know.
func TestBuildJSONOutput_StepsArray_UnknownCategorySortsLast(t *testing.T) {
	r := &Report{Steps: []Step{
		{Category: "quantum", Status: StatusPass, Message: "future"},
		{Category: CatBlockchain, Status: StatusPass, Message: "blockchain"},
		{Category: CatDataIntegrity, Status: StatusPass, Message: "data integrity"},
	}}
	out := BuildJSONOutput(r)
	if out.Steps[0].Message != "data integrity" || out.Steps[2].Message != "future" {
		t.Errorf("unknown category did not sort last: %q, %q, %q",
			out.Steps[0].Message, out.Steps[1].Message, out.Steps[2].Message)
	}
}

// --- F12 / E.7: "a hash was supplied" is separate from "it matched" ---

func TestBuildJSONOutput_HashComparison_SuppliedIsSeparateFromMatched(t *testing.T) {
	cases := []struct {
		name         string
		r            *Report
		wantSupplied bool
		wantMatched  bool
	}{
		{
			name:         "no expected hash supplied",
			r:            &Report{Claims: Claims{Hash: "aabb"}},
			wantSupplied: false,
			wantMatched:  false,
		},
		{
			name: "supplied and matched",
			r: &Report{HashProvided: "aabb", Claims: Claims{Hash: "aabb"}, Steps: []Step{
				{Group: groupHashComparison, Status: StatusPass},
			}},
			wantSupplied: true,
			wantMatched:  true,
		},
		{
			name: "supplied and did not match",
			r: &Report{HashProvided: "deadbeef", Claims: Claims{Hash: "aabb"}, Steps: []Step{
				{Group: groupHashComparison, Status: StatusFail},
			}},
			wantSupplied: true,
			wantMatched:  false,
		},
		{
			// The case the field exists for: E.7 REQUIRES an
			// inapplicability skip for a subject that commits to no
			// file hash, and E.22 requires "one was supplied" to stay
			// readable separately from "it matched". Reporting
			// supplied:false here made the published object
			// byte-identical to the no --hash case.
			name: "step reported as a skip: supplied but not compared",
			r: &Report{HashProvided: "deadbeef", Claims: Claims{}, Steps: []Step{
				{Group: groupHashComparison, Status: StatusSkip},
			}},
			wantSupplied: true,
			wantMatched:  false,
		},
		{
			// Remote mode can carry both the server's row and the CLI's
			// own locally computed one. A server-reported pass must not
			// be able to publish matched:true over the CLI's refutation.
			name: "server says pass, this verifier says fail",
			r: &Report{HashProvided: "deadbeef", Claims: Claims{Hash: "aabb"}, Steps: []Step{
				{Group: groupHashComparison, Status: StatusPass},
				{Group: groupHashComparison, Status: StatusFail},
			}},
			wantSupplied: true,
			wantMatched:  false,
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			out := BuildJSONOutput(c.r)
			if out.HashComparison.Supplied != c.wantSupplied {
				t.Errorf("supplied: got %v, want %v", out.HashComparison.Supplied, c.wantSupplied)
			}
			if out.HashComparison.Matched != c.wantMatched {
				t.Errorf("matched: got %v, want %v", out.HashComparison.Matched, c.wantMatched)
			}
			// Always present, so a consumer never has to infer
			// "not supplied" from a missing object.
			data, err := json.Marshal(out)
			if err != nil {
				t.Fatalf("marshal: %s", err)
			}
			if !strings.Contains(string(data), `"hash_comparison":{`) {
				t.Errorf("hash_comparison object missing from output: %s", data)
			}
		})
	}
}

// --- F4: externally_verified is true only on a confirmed lookup ---

func TestBuildJSONOutput_ExternallyVerified_OnlyWhenConfirmed(t *testing.T) {
	cases := []struct {
		status ExternalStatus
		want   bool
	}{
		{ExternalConfirmed, true},
		{ExternalSkipped, false},
		{ExternalFailed, false},
	}
	for _, c := range cases {
		r := &Report{
			ChainLength: 1,
			CommitmentInfos: []CommitmentInfo{
				{Method: "stellar", Network: "testnet", Ledger: 42, ExternalCheck: c.status},
				{Method: "bitcoin", Network: "regtest", Height: 7, ExternalCheck: c.status},
			},
		}
		out := BuildJSONOutput(r)
		if out.Commitments == nil || out.Commitments.Stellar == nil || out.Commitments.Bitcoin == nil {
			t.Fatalf("expected both commitments in output for status %d", c.status)
		}
		if got := out.Commitments.Stellar.ExternallyVerified; got != c.want {
			t.Errorf("stellar externally_verified for status %d: got %v, want %v", c.status, got, c.want)
		}
		if got := out.Commitments.Bitcoin.ExternallyVerified; got != c.want {
			t.Errorf("bitcoin externally_verified for status %d: got %v, want %v", c.status, got, c.want)
		}
	}
}

func TestExternalCheckSuffix_ThreeDistinctLabels(t *testing.T) {
	seen := map[string]ExternalStatus{}
	for _, s := range []ExternalStatus{ExternalConfirmed, ExternalSkipped, ExternalFailed} {
		got := externalCheckSuffix(CommitmentInfo{ExternalCheck: s, Timestamp: "2026-04-06T23:25:06Z"})
		if prev, dup := seen[got]; dup {
			t.Errorf("status %d and %d render the same label %q", prev, s, got)
		}
		seen[got] = s
	}
	if got := externalCheckSuffix(CommitmentInfo{ExternalCheck: ExternalFailed}); !strings.Contains(got, "failed") {
		t.Errorf("failed lookup label: got %q, want it to say the lookup failed", got)
	}
	if got := externalCheckSuffix(CommitmentInfo{ExternalCheck: ExternalSkipped}); !strings.Contains(got, "skipped") {
		t.Errorf("skipped lookup label: got %q, want it to say the lookup was skipped", got)
	}
}

// --- F8c: failure details are deterministic and never positive ---

func TestLookupFailureDetail_Deterministic(t *testing.T) {
	// Each of these matches more than one keyword. Under the old map
	// the answer depended on Go's randomized iteration order.
	multiKeyword := []string{
		"Cannot verify proof signature (missing block hash)",
		"Bitcoin merkle proof valid",
		"Epoch proof 1 (Stellar): derived epoch root does not match the committed value",
	}
	for _, msg := range multiKeyword {
		first := lookupFailureDetail(msg)
		for i := 0; i < 50; i++ {
			if got := lookupFailureDetail(msg); got != first {
				t.Fatalf("detail for %q is nondeterministic: %q then %q", msg, first, got)
			}
		}
		if first == "" {
			t.Errorf("expected a detail for %q", msg)
		}
	}
	// A local Merkle walk over a Stellar commitment is not a Horizon
	// problem, so it must not be blamed on the Stellar network.
	local := lookupFailureDetail("Epoch proof 1 (Stellar): derived epoch root does not match the committed value")
	if strings.Contains(local, "Stellar blockchain could not confirm") {
		t.Errorf("local epoch-proof failure blamed on Horizon: %q", local)
	}
}

func TestLookupFailureDetail_KeyringDetailIsNegative(t *testing.T) {
	detail := lookupFailureDetail("Keyring verification failed: key not found")
	if detail == "" {
		t.Fatal("expected a detail for a keyring failure")
	}
	if strings.Contains(detail, "confirms") {
		t.Errorf("keyring failure detail reads as a positive assertion: %q", detail)
	}
	if !strings.Contains(detail, "does not vouch") {
		t.Errorf("keyring failure detail: got %q", detail)
	}
}

// --- Blocker R1: info rows are notes, not issues ---

func TestRenderIssues_ExcludesInfoRows(t *testing.T) {
	r := &Report{
		Steps: []Step{
			{Group: groupSubjectData, Category: CatDataIntegrity, Status: StatusInfo,
				Message: "Claims hash well formed for sha256"},
			{Group: groupStellar, Category: CatBlockchain, Status: StatusSkip,
				Message: "External Stellar verification skipped (--skip-external)"},
		},
	}
	issues := renderIssues(r)
	if strings.Contains(issues, "Claims hash well formed") {
		t.Errorf("info row rendered under Issues:\n%s", issues)
	}
	if !strings.Contains(issues, "External Stellar verification skipped") {
		t.Errorf("skip row should still render under Issues:\n%s", issues)
	}

	notes := renderVerificationNotes(r)
	if !strings.Contains(notes, "Claims hash well formed") {
		t.Errorf("info row missing from Verification Notes:\n%s", notes)
	}
	if strings.Contains(notes, "External Stellar verification skipped") {
		t.Errorf("skip row leaked into Verification Notes:\n%s", notes)
	}
}

// TestRenderIssues_SeverityOrderWithinCategory pins the order the Issues
// section renders rows in.
//
// It used to sort on the Status value, whose numbering is iota
// declaration order (Pass 0, Fail 1, Skip 2, Warn 3), so warnings landed
// BELOW skips. A warn qualifies the verdict — "this run establishes
// nothing about who signed the proof" — while a skip only records that a
// check was not attempted, and under --skip-external there can be
// several skips to scroll past.
func TestRenderIssues_SeverityOrderWithinCategory(t *testing.T) {
	// Emitted skip-first so a stable sort cannot produce the wanted order
	// by accident.
	r := &Report{Steps: []Step{
		{Group: groupKeyBinding, Category: CatCryptographic, Status: StatusSkip,
			Message: "SKIP-ROW keyring cross-check not performed"},
		{Group: groupProof, Category: CatCryptographic, Status: StatusWarn,
			Message: "WARN-ROW this run establishes nothing about who signed"},
		{Group: groupInclusion, Category: CatCryptographic, Status: StatusFail,
			Message: "FAIL-ROW inclusion proof root does not match"},
	}}

	out := renderIssues(r)
	fail := strings.Index(out, "FAIL-ROW")
	warn := strings.Index(out, "WARN-ROW")
	skip := strings.Index(out, "SKIP-ROW")
	if fail < 0 || warn < 0 || skip < 0 {
		t.Fatalf("not every row rendered (fail=%d warn=%d skip=%d):\n%s", fail, warn, skip, out)
	}
	if !(fail < warn && warn < skip) {
		t.Errorf("Issues order is not fail < warn < skip (fail=%d warn=%d skip=%d):\n%s",
			fail, warn, skip, out)
	}
}

func TestRenderIssues_EmptyWhenOnlyInfoRows(t *testing.T) {
	r := &Report{Steps: []Step{
		{Group: groupSigningKey, Category: CatCryptographic, Status: StatusPass, Message: "ok"},
		{Group: groupTemporalInfo, Category: CatTiming, Status: StatusInfo, Message: "Submitted 2026-04-06T23:25:06Z"},
	}}
	if got := renderIssues(r); got != "" {
		t.Errorf("a clean proof with only info rows must render no Issues section:\n%s", got)
	}
}

// TestRenderVerificationNotes_NoDuplicateForInfoInOwnGroup guards the
// one row that satisfies both collection predicates.
func TestRenderVerificationNotes_NoDuplicateForInfoInOwnGroup(t *testing.T) {
	r := &Report{Steps: []Step{
		{Group: groupVerificationNotes, Category: CatDataIntegrity, Status: StatusInfo,
			Message: "Claims-only item: nothing external to compare"},
	}}
	notes := renderVerificationNotes(r)
	if n := strings.Count(notes, "Claims-only item"); n != 1 {
		t.Errorf("note rendered %d times, want 1:\n%s", n, notes)
	}
}

// --- Failure details: reachable, and true of what reaches them ------
//
// productionFailureMessages is a corpus of messages the pipeline can
// actually emit, each naming its producer. It is the input to both
// tests below: one asserts no keyword is dead, the other asserts the
// specific mappings the judging pass found wrong.
var productionFailureMessages = []struct {
	source string // where the message is built
	msg    string
}{
	{"verify.go deriveClaimsHash / hash comparison", "Provided hash does not match claims.hash (expected: deadbeef, proof: b47cc0f1)"},
	{"verify.go verifyProofSignature, missing-input arms", "Cannot verify proof signature (missing derived data): no block hash"},
	{"verify.go verifyProofSignature, missing-input arms", "Cannot verify proof signature (missing derived data): no subject hash"},
	{"verify.go verifyProofSignature, missing-input arms", "Cannot verify proof signature (missing derived data): no usable public key"},
	{"verify.go verifyProofSignature, E.16 check", "Proof signature invalid (Ed25519)"},
	{"verify.go verifyKeyBinding + external.KeyBindingError", "Key binding failed: key f2c39df9 not found in keyring"},
	{"verify.go verifyClaimsTimestamp", "Claims timestamp is not before submission time (future-dated claim)"},
	{"verify.go verifySubjectTemporalWindow (E.20)", "Submission-window ordering violation: Item submitted at 2026-07-24T11:59:00.000Z is AFTER committed block time 2026-07-24T11:00:00.000Z"},
	{"verify.go verifySubjectTemporalWindow (E.20), entropy label", "Submission-window ordering violation: Entropy captured at 2026-07-24T13:00:00.000Z is AFTER committed block time 2026-07-24T12:00:00.000Z"},
	{"verify.go verifyBitcoinTxOutProof (E.19(b) step 3)", "Bitcoin partial merkle proof invalid: the derived root does not match the txoutproof header merkle root"},
	{"verify.go verifySingleBitcoin (E.19(b) step 4)", "Bitcoin merkle proof root does NOT match cx.bmr (txoutproof header carries 68daa685)"},
	{"verify.go verifyBitcoinTxOutProof (E.19(b) step 5)", "Bitcoin merkle proof does not place cx.tx in the matched transaction set"},
	{"verify.go verifyInclusionProof (E.13)", "Inclusion proof derived root does not match block merkle root"},
	{"verify.go verifyEpochProofs (E.15)", "Epoch proof 0 (Stellar): derived epoch root does not match the committed value"},
	{"verify.go deriveBlockHash (E.14)", "Cannot derive block hash: no usable value for b.mr (E.14 requires all five of id, ph, mr, mh, kid)"},
	{"verify.go verifySingleStellar (E.18)", "Stellar commitment not on chain: transaction 5e5e5e5e not found on public"},
	{"verify.go verifyBitcoinBinding (E.19)", "Bitcoin commitment not on chain: block 4f6d3d68 not found on mainnet"},
	// Remote mode shares this table with the server's own vocabulary.
	{"truestamp-v2 verification.ex, inclusion", "Inclusion proof INVALID (derived root does not match block merkle_root)"},
	{"truestamp-v2 verification.ex, bitcoin", "Bitcoin merkle verification failed: bad flags"},
}

func TestFailureDetails_EveryKeywordIsReachable(t *testing.T) {
	for _, d := range failureDetails {
		matched := false
		for _, m := range productionFailureMessages {
			if strings.Contains(strings.ToLower(m.msg), d.keyword) {
				matched = true
				break
			}
		}
		if !matched {
			t.Errorf("failureDetails keyword %q matches no message any verifier emits — "+
				"either it is dead and must be deleted, or its producer belongs in productionFailureMessages", d.keyword)
		}
	}
}

func TestLookupFailureDetail_ProductionMessages(t *testing.T) {
	cases := []struct {
		name string
		msg  string
		want string
	}{
		{
			// Was: no detail at all. The four keywords written for this
			// row ("submitted after"/"before", "captured after"/"before")
			// matched nothing the pipeline emits.
			name: "E.20 ordering violation, item",
			msg:  "Submission-window ordering violation: Item submitted at 2026-07-24T11:59:00.000Z is AFTER committed block time 2026-07-24T11:00:00.000Z",
			want: "The subject's own identifier is timestamped after the block that commits it, so the submission window does not hold.",
		},
		{
			name: "E.20 ordering violation, entropy",
			msg:  "Submission-window ordering violation: Entropy captured at 2026-07-24T13:00:00.000Z is AFTER committed block time 2026-07-24T12:00:00.000Z",
			want: "The subject's own identifier is timestamped after the block that commits it, so the submission window does not hold.",
		},
		{
			// Was: "The proof may have been tampered with or signed
			// with a different key." Nothing about tampering was
			// established — an upstream derivation simply could not run.
			name: "signature abort establishes nothing about tampering",
			msg:  "Cannot verify proof signature (missing derived data): no block hash",
			want: "An input this check needs was never derived, so the signature was neither confirmed nor refuted.",
		},
		{
			name: "a genuinely invalid signature still says so",
			msg:  "Proof signature invalid (Ed25519)",
			want: "The proof may have been tampered with or signed with a different key.",
		},
		{
			// Was: "The item cannot be verified as belonging to the
			// committed block." — asserted in reports whose Inclusion
			// Proof row was simultaneously passing.
			name: "bitcoin bmr cross-check is not an inclusion failure",
			msg:  "Bitcoin merkle proof root does NOT match cx.bmr (txoutproof header carries 68daa685)",
			want: "The Bitcoin transaction cannot be tied to the block header the entry carries.",
		},
		{
			name: "bitcoin placement is not an inclusion failure",
			msg:  "Bitcoin merkle proof does not place cx.tx in the matched transaction set",
			want: "The Bitcoin transaction cannot be tied to the block header the entry carries.",
		},
		{
			name: "bitcoin partial tree is not an inclusion failure",
			msg:  "Bitcoin partial merkle proof invalid: the derived root does not match the txoutproof header merkle root",
			want: "The Bitcoin transaction cannot be tied to the block header the entry carries.",
		},
		{
			// The noun is "subject": the same walk runs over entropy
			// observations, which are not items.
			name: "inclusion failure names the subject, not the item",
			msg:  "Inclusion proof derived root does not match block merkle root",
			want: "The subject cannot be verified as belonging to the committed block.",
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := lookupFailureDetail(c.msg); got != c.want {
				t.Errorf("detail for %q:\n got  %q\n want %q", c.msg, got, c.want)
			}
		})
	}
}

// --- The two surfaces render one Report ------------------------------

func TestSurfaces_IssuesAndNotesAgree(t *testing.T) {
	r := &Report{Steps: []Step{
		{Group: groupSigningKey, Category: CatCryptographic, Status: StatusPass, Message: "a pass"},
		{Group: groupProof, Category: CatCryptographic, Status: StatusFail, Message: "a failure"},
		{Group: groupKeyBinding, Category: CatCryptographic, Status: StatusSkip, Message: "a skip"},
		{Group: groupSubjectData, Category: CatDataIntegrity, Status: StatusWarn, Message: "a warning"},
		{Group: groupTemporalInfo, Category: CatTiming, Status: StatusInfo, Message: "an info row outside the notes group"},
		{Group: groupVerificationNotes, Category: CatDataIntegrity, Status: StatusWarn, Message: "a workflow nudge"},
	}}

	out := BuildJSONOutput(r)
	terminalIssues := renderIssues(r)
	terminalNotes := renderVerificationNotes(r)

	for _, s := range r.Steps {
		if s.Status == StatusPass {
			continue
		}
		inJSONIssues, inJSONNotes := false, false
		for _, i := range out.Issues {
			if i.Message == s.Message {
				inJSONIssues = true
			}
		}
		for _, n := range out.VerificationNotes {
			if n.Message == s.Message {
				inJSONNotes = true
			}
		}
		inTermIssues := strings.Contains(terminalIssues, s.Message)
		inTermNotes := strings.Contains(terminalNotes, s.Message)

		if inJSONIssues != inTermIssues {
			t.Errorf("%q: issues bucket disagrees (json=%v terminal=%v)", s.Message, inJSONIssues, inTermIssues)
		}
		if inJSONNotes != inTermNotes {
			t.Errorf("%q: notes bucket disagrees (json=%v terminal=%v)", s.Message, inJSONNotes, inTermNotes)
		}
		if !inJSONIssues && !inJSONNotes {
			t.Errorf("%q: non-passing row appears in neither bucket", s.Message)
		}
	}
	// D.4's three skip rows in particular must reach the JSON surface.
	if len(out.Issues) == 0 {
		t.Error("issues array is empty for a report the terminal renders an Issues section for")
	}
}

// --- Summary reconciles with the steps array ------------------------

func TestBuildJSONOutput_Summary_ReconcilesWithSteps(t *testing.T) {
	r := &Report{Steps: []Step{
		{Group: groupSigningKey, Category: CatCryptographic, Status: StatusPass},
		{Group: groupProof, Category: CatCryptographic, Status: StatusFail},
		{Group: groupSubjectData, Category: CatDataIntegrity, Status: StatusWarn},
		{Group: groupKeyBinding, Category: CatCryptographic, Status: StatusSkip},
		{Group: groupTemporalInfo, Category: CatTiming, Status: StatusInfo},
		{Group: groupTemporalInfo, Category: CatTiming, Status: StatusInfo},
	}}
	out := BuildJSONOutput(r)
	s := out.Summary
	if s.Info != 2 {
		t.Errorf("summary.info: got %d, want 2 (E.22 defines five statuses)", s.Info)
	}
	if got := s.Passed + s.Failed + s.Warnings + s.Skipped + s.Info; got != s.Total {
		t.Errorf("summary does not sum to total: %d vs %d", got, s.Total)
	}
	if s.Total != len(out.Steps) {
		t.Errorf("summary.total %d does not reconcile with len(steps) %d", s.Total, len(out.Steps))
	}
}

// --- Beacon (t=11) is a block-shaped subject ------------------------

func TestBuildJSONOutput_BeaconSubjectCarriesBlockFields(t *testing.T) {
	for _, subjectType := range []string{"block", "beacon"} {
		r := &Report{
			SubjectType:       subjectType,
			SubjectID:         "019dba82-1347-7d72-a32c-6530d2a9f792",
			SigningKeyID:      "0c15f1f3",
			BlockSigningKeyID: "4ceefa4a",
			Temporal:          TemporalSummary{CommittedAt: "2026-04-22T22:33:00Z"},
		}
		m, ok := BuildJSONOutput(r).Subject.(map[string]any)
		if !ok {
			t.Fatalf("%s: subject is not an object", subjectType)
		}
		if m["block_id"] != r.SubjectID || m["committed_at"] != r.Temporal.CommittedAt {
			t.Errorf("%s: block fields dropped from subject: %v", subjectType, m)
		}
		// A field of the block is b.kid, not the derived signer.
		if m["signing_key"] != "4ceefa4a" {
			t.Errorf("%s: signing_key = %v, want the block's own b.kid", subjectType, m["signing_key"])
		}
	}
}

// --- F4's tri-state survives into --json ----------------------------

func TestExternalStatus_ZeroValueIsSkipped(t *testing.T) {
	// A CommitmentInfo literal that forgets the field must publish
	// "not confirmed", not "confirmed".
	var ci CommitmentInfo
	if ci.ExternalCheck != ExternalSkipped {
		t.Errorf("zero value is %v, want ExternalSkipped", ci.ExternalCheck)
	}
}

func TestBuildJSONOutput_ExternalCheck_ExposesThreeStates(t *testing.T) {
	cases := []struct {
		status   ExternalStatus
		want     string
		verified bool
	}{
		{ExternalConfirmed, "confirmed", true},
		{ExternalSkipped, "skipped", false},
		{ExternalFailed, "failed", false},
	}
	for _, c := range cases {
		r := &Report{ChainLength: 1, CommitmentInfos: []CommitmentInfo{
			{Method: "stellar", Network: "public", ExternalCheck: c.status},
		}}
		bc := BuildJSONOutput(r).Commitments.Stellar
		if bc.ExternalCheck != c.want {
			t.Errorf("external_check: got %q, want %q", bc.ExternalCheck, c.want)
		}
		if bc.ExternallyVerified != c.verified {
			t.Errorf("externally_verified for %q: got %v, want %v", c.want, bc.ExternallyVerified, c.verified)
		}
	}
	// The distinction the boolean alone cannot carry.
	failed := BuildJSONOutput(&Report{ChainLength: 1, CommitmentInfos: []CommitmentInfo{
		{Method: "bitcoin", ExternalCheck: ExternalFailed}}}).Commitments.Bitcoin
	skipped := BuildJSONOutput(&Report{ChainLength: 1, CommitmentInfos: []CommitmentInfo{
		{Method: "bitcoin", ExternalCheck: ExternalSkipped}}}).Commitments.Bitcoin
	if failed.ExternalCheck == skipped.ExternalCheck {
		t.Error("a chain that disagreed and a lookup that never ran are still indistinguishable")
	}
}

// --- --skip-signatures is disclosed wherever an outcome is stated ---

func skipSigReport() *Report {
	return &Report{
		SubjectType: "item",
		Steps: []Step{
			{Group: groupSigningKey, Category: CatCryptographic, Status: StatusPass, Message: "Public key valid, key_id: f2c39df9"},
			{Group: groupProof, Category: CatCryptographic, Status: StatusSkip, Message: "Proof signature verification skipped (--skip-signatures)"},
		},
	}
}

func TestReport_SignaturesSkipped(t *testing.T) {
	if !skipSigReport().SignaturesSkipped() {
		t.Error("a run that skipped E.16 must report so")
	}
	checked := &Report{Steps: []Step{
		{Group: groupProof, Category: CatCryptographic, Status: StatusPass, Message: "Proof signature valid (Ed25519)"},
	}}
	if checked.SignaturesSkipped() {
		t.Error("a run that checked the signature must not report it skipped")
	}
}

func TestRenderVerificationSummary_DisclosesSkippedSignatures(t *testing.T) {
	r := skipSigReport()
	if !r.Passed() {
		t.Fatal("fixture should pass so the verdict line is a positive one")
	}
	out := renderVerificationSummary(r)
	if !strings.Contains(out, "VERIFIED") {
		t.Fatalf("fixture no longer renders a passing verdict: %s", out)
	}
	// The verdict line itself, not just a step buried in the report.
	verdictLine := ""
	for _, line := range strings.Split(out, "\n") {
		if strings.Contains(line, "VERIFIED") {
			verdictLine = line
		}
	}
	if !strings.Contains(verdictLine, "--skip-signatures") {
		t.Errorf("the verdict line states an outcome without disclosing the signature was never checked: %q", verdictLine)
	}
	if strings.Contains(verdictLine, "proof is valid") {
		t.Errorf("the verdict line still asserts the proof is valid: %q", verdictLine)
	}

	// And the same run with a matched hash.
	full := skipSigReport()
	full.HashProvided = "aabb"
	full.Claims = Claims{Hash: "aabb"}
	full.Steps = append(full.Steps, Step{Group: groupHashComparison, Category: CatDataIntegrity, Status: StatusPass})
	if full.Verdict() != VerdictFullyVerified {
		t.Fatalf("fixture verdict: %v", full.Verdict())
	}
	if !strings.Contains(renderVerificationSummary(full), "--skip-signatures") {
		t.Error("the fully-verified verdict line omits the disclosure")
	}
}

func TestBuildJSONOutput_SignaturesChecked(t *testing.T) {
	if BuildJSONOutput(skipSigReport()).SignaturesChecked {
		t.Error("signatures_checked must be false for a --skip-signatures run")
	}
	checked := &Report{Steps: []Step{
		{Group: groupProof, Category: CatCryptographic, Status: StatusPass, Message: "Proof signature valid (Ed25519)"},
	}}
	if !BuildJSONOutput(checked).SignaturesChecked {
		t.Error("signatures_checked must be true when E.16 ran")
	}
}

// --- Summary line honesty -------------------------------------------

func TestRenderVerificationSummary_DoesNotCallEveryStepCryptographic(t *testing.T) {
	r := &Report{Steps: []Step{
		{Group: groupHashComparison, Category: CatDataIntegrity, Status: StatusPass},
		{Group: groupStructure, Category: CatStructural, Status: StatusPass},
		{Group: groupSubmissionWindow, Category: CatTiming, Status: StatusPass},
		{Group: groupStellar, Category: CatBlockchain, Status: StatusSkip},
		{Group: groupTemporalInfo, Category: CatTiming, Status: StatusInfo},
	}}
	out := renderVerificationSummary(r)
	if strings.Contains(out, "cryptographic checks") {
		t.Errorf("the count covers four non-cryptographic categories: %s", out)
	}
	// The denominator reconciles with the step record.
	if !strings.Contains(out, "3 of 5 verification steps passed") {
		t.Errorf("summary line does not reconcile with the steps: %s", out)
	}
	if !strings.Contains(out, "1 info") {
		t.Errorf("info steps are not accounted for: %s", out)
	}
}
