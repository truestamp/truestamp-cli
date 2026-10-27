// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: Apache-2.0

package verify

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/truestamp/truestamp-cli/internal/proof"
	"github.com/truestamp/truestamp-cli/internal/testfixtures"
)

// prodBundle is the production item of the earlier layout, its claims
// given hash when hash is not "", parsed as the remote path parses it.
func prodBundle(t *testing.T, hash string) *proof.Bundle {
	t.Helper()
	data := testfixtures.Read(t, testfixtures.ProdDir, testfixtures.ProdComplete)
	if hash != "" {
		var top map[string]any
		if err := json.Unmarshal(data, &top); err != nil {
			t.Fatal(err)
		}
		claims := top["subject"].(map[string]any)["claims"].(map[string]any)
		claims["hash"], claims["hash_type"] = hash, "sha256"
		var err error
		if data, err = json.Marshal(top); err != nil {
			t.Fatal(err)
		}
	}
	bundle, err := proof.ParseBytes(data)
	if err != nil {
		t.Fatal(err)
	}
	return bundle
}

// statuses is each step's group and status, in order.
func statuses(steps []Step) string {
	var out []string
	for _, s := range steps {
		out = append(out, s.Group+" "+s.Status.String())
	}
	return strings.Join(out, ", ")
}

// A step whose status the service leaves out, or gives as a word outside
// the five, decodes as fail, its message saying why, and a known word
// decodes as itself. Mutant killed: Step.UnmarshalJSON decoding an
// unknown word as pass (the "maybe" step reads pass).
func TestStep_AStatusItCannotReadIsFail(t *testing.T) {
	for _, tc := range []struct {
		wire    string
		status  Status
		message string
	}{
		{`{"group":"G","status":"maybe","message":"m"}`, StatusFail,
			`m [the server reported status "maybe", which is not one of pass/fail/warn/skip/info, so it cannot be read as having passed]`},
		{`{"group":"G","message":"m"}`, StatusFail,
			"m [the server reported no status for this step, so it cannot be read as having passed]"},
		{`{"group":"G","status":null}`, StatusFail,
			"the server reported no status for this step, so it cannot be read as having passed"},
		{`{"group":"G","status":"skip","message":"m"}`, StatusSkip, "m"},
	} {
		var s Step
		if err := json.Unmarshal([]byte(tc.wire), &s); err != nil {
			t.Fatalf("%s: %v", tc.wire, err)
		}
		if s.Group != "G" || s.Status != tc.status || s.Message != tc.message {
			t.Errorf("%s: %+v, want status %s and the message %q", tc.wire, s, tc.status, tc.message)
		}
	}
}

// The service's top-level passed is held to its steps: passed false with
// no failing step, or no steps at all, adds a failing Server Verdict
// step, and passed true over steps that pass adds none. Mutant killed:
// applyServerVerdict without its passed-false check (passed false over
// passing steps reads as passed).
func TestApplyServerVerdict_ThePassedMemberIsHeldToTheSteps(t *testing.T) {
	pass := []Step{{Group: "Signature", Status: StatusPass}}
	for _, tc := range []struct {
		name   string
		result apiResult
		want   string
	}{
		{"passed false, no failing step", apiResult{Passed: false, Steps: pass}, "Server Verdict fail"},
		{"no steps", apiResult{Passed: true}, "Server Verdict fail"},
		{"passed true over passing steps", apiResult{Passed: true, Steps: pass}, ""},
	} {
		r := &Report{}
		applyServerVerdict(r, &tc.result)
		if got := statuses(r.Steps); got != tc.want || r.Passed() != (tc.want == "") {
			t.Errorf("%s: steps %q, passed %v; want %q", tc.name, got, r.Passed(), tc.want)
		}
	}
}

// The expected-hash comparison is this process's own, on the bundle it
// posted: when the service says the supplied hash matches and this
// comparison does not pass, or says it does not match where this
// comparison passes, the run gets a failing Hash Comparison step beside
// this comparison's own; when the two agree nothing is added. Mutants
// killed: applyExpectedHash without its case for a match the service
// claims (the first row passes); without its case for a mismatch the
// service claims (the second row passes).
func TestApplyExpectedHash_ADisagreementWithTheServiceFails(t *testing.T) {
	const hash = "ab0000000000000000000000000000000000000000000000000000000000000c"
	echoed := hash
	for _, tc := range []struct {
		name     string
		bundle   *proof.Bundle
		expected string
		result   apiResult
		want     string
	}{
		{"the service claims a match where this proof holds no hash", prodBundle(t, ""), hash,
			apiResult{Steps: []Step{{Group: groupHashComparison, Status: StatusPass}}},
			"Hash Comparison pass, Hash Comparison warn, Hash Comparison fail"},
		{"the service claims no match where the hashes agree", prodBundle(t, hash), hash,
			apiResult{HashProvided: &echoed, HashMatched: false, Steps: []Step{}},
			"Hash Comparison pass, Hash Comparison fail"},
		{"the service and this comparison agree", prodBundle(t, hash), hash,
			apiResult{HashProvided: &echoed, HashMatched: true, Steps: []Step{{Group: groupHashComparison, Status: StatusPass}}},
			"Hash Comparison pass"},
	} {
		r := &Report{Steps: tc.result.Steps, ExpectedHash: tc.expected}
		applyExpectedHash(r, tc.bundle, &tc.result)
		if got := statuses(r.Steps); got != tc.want {
			t.Errorf("%s: steps %q, want %q", tc.name, got, tc.want)
		}
	}
}

// inspect --json writes a rejection of the earlier bundle parser with that
// parser's code, its detail and its advice, passed false and no steps.
// Mutant killed: BuildJSONRejection writing an empty code.
func TestBuildJSONRejection_WritesTheParsersCode(t *testing.T) {
	_, err := proof.ParseBytes(testfixtures.Read(t, testfixtures.TamperDir, "tamper-type.json"))
	var rejection *proof.RejectionError
	if !errors.As(err, &rejection) {
		t.Fatalf("tamper-type.json: %v, want a rejection", err)
	}
	got := BuildJSONRejection(err)
	want := Rejection{Code: proof.CodeUnexpectedSubjectFieldsForBlockLike, Detail: rejection.Detail,
		Advice: proof.RejectionAdvice(proof.CodeUnexpectedSubjectFieldsForBlockLike)}
	if got.Passed || got.Rejection == nil || *got.Rejection != want || got.Steps == nil || len(got.Steps) != 0 || got.Verifier != ThisVerifier() {
		t.Fatalf("BuildJSONRejection = %+v, want the rejection %+v", got, want)
	}
}

// answering is a loopback stand-in for /proof/verify that answers every
// request with status and body, and counts the requests it is sent.
func answering(t *testing.T, status int, body string) (string, *atomic.Int32) {
	t.Helper()
	var hits atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits.Add(1)
		if r.Method != http.MethodPost || r.URL.Path != "/proof/verify" {
			t.Errorf("request %s %s, want POST /proof/verify", r.Method, r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/vnd.api+json")
		w.WriteHeader(status)
		_, _ = w.Write([]byte(body))
	}))
	t.Cleanup(srv.Close)
	return srv.URL, &hits
}

// A non-2xx answer is the service's refusal, never its verdict, even when
// its body carries a result whose steps all pass: the run returns no
// report and the error names the status and the body. Mutant killed:
// RunRemoteBytesCtx without its non-2xx check (the 500's passing result
// reads as the service's verdict).
func TestRunRemoteBytesCtx_ANon2xxAnswerIsNeverAVerdict(t *testing.T) {
	const body = `{"result":{"passed":true,"steps":[{"group":"Signature","status":"pass","message":"m"}]}}`
	for _, code := range []int{http.StatusInternalServerError, http.StatusBadRequest, http.StatusMultipleChoices} {
		url, hits := answering(t, code, body)
		data := testfixtures.Read(t, testfixtures.ProdDir, testfixtures.ProdComplete)
		report, err := RunRemoteBytesCtx(context.Background(), data, RemoteOptions{APIURL: url})
		want := "API error (HTTP " + strconv.Itoa(code) + "): " + body
		if report != nil || err == nil || err.Error() != want || hits.Load() != 1 {
			t.Errorf("HTTP %d: report %+v, error %v, %d requests; want no report, the error %q and one request",
				code, report, err, hits.Load(), want)
		}
	}
}

// The service's refusal of a proof, an error whose meta.code is
// invalid_proof with a reason, is a RemoteRejectionError carrying the
// status, the reason and the detail; any other error is a plain error
// naming its detail, or its title where it has no detail. Mutant killed:
// parseAPIError without its invalid_proof case (the refusal becomes a
// plain error and errors.As finds no RemoteRejectionError).
func TestRunRemoteBytesCtx_TheServicesRefusalIsARemoteRejection(t *testing.T) {
	for _, tc := range []struct {
		name      string
		body      string
		rejection *RemoteRejectionError
		message   string
	}{
		{"invalid_proof with a reason",
			`{"errors":[{"status":"400","code":"invalid","title":"Invalid proof","detail":"The signature does not verify.","meta":{"code":"invalid_proof","reason":"signature_invalid"}}]}`,
			&RemoteRejectionError{StatusCode: 400, Reason: "signature_invalid", Detail: "The signature does not verify."},
			"the server rejected the proof (HTTP 400): signature_invalid: The signature does not verify."},
		{"invalid_proof with no reason",
			`{"errors":[{"status":"400","title":"Invalid proof","detail":"The proof is malformed.","meta":{"code":"invalid_proof"}}]}`,
			nil, "API error (HTTP 400): The proof is malformed."},
		{"another code with a detail",
			`{"errors":[{"status":"400","title":"Bad request","detail":"skip_external must be a boolean.","meta":{"code":"invalid_attribute","reason":"type"}}]}`,
			nil, "API error (HTTP 400): skip_external must be a boolean."},
		{"another code with only a title",
			`{"errors":[{"status":"400","title":"Bad request"}]}`,
			nil, "API error (HTTP 400): Bad request"},
	} {
		url, _ := answering(t, http.StatusBadRequest, tc.body)
		data := testfixtures.Read(t, testfixtures.ProdDir, testfixtures.ProdComplete)
		report, err := RunRemoteBytesCtx(context.Background(), data, RemoteOptions{APIURL: url})
		if report != nil || err == nil || err.Error() != tc.message {
			t.Errorf("%s: report %+v, error %v; want no report and the error %q", tc.name, report, err, tc.message)
			continue
		}
		var rejection *RemoteRejectionError
		found := errors.As(err, &rejection)
		switch {
		case tc.rejection == nil && found:
			t.Errorf("%s: a RemoteRejectionError %+v, want a plain error", tc.name, *rejection)
		case tc.rejection != nil && (!found || *rejection != *tc.rejection):
			t.Errorf("%s: %v (RemoteRejectionError found: %v), want %+v", tc.name, err, found, *tc.rejection)
		}
	}
}
