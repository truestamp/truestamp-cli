// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: Apache-2.0

package verify

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/truestamp/truestamp-cli/internal/httpclient"
	"github.com/truestamp/truestamp-cli/internal/proof"
	"github.com/truestamp/truestamp-cli/internal/testfixtures"
)

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

// posted is one request the stand-in for the raw verify route was sent.
type posted struct {
	method, path, query, contentType string
	body                             []byte
}

// answering is a loopback stand-in for the raw verify route that answers
// every request with status and body, records the last request and
// counts them.
func answering(t *testing.T, status int, body string) (string, *posted, *atomic.Int32) {
	t.Helper()
	var hits atomic.Int32
	last := &posted{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits.Add(1)
		got, _ := io.ReadAll(r.Body)
		*last = posted{r.Method, r.URL.Path, r.URL.RawQuery, r.Header.Get("Content-Type"), got}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		_, _ = w.Write([]byte(body))
	}))
	t.Cleanup(srv.Close)
	return srv.URL, last, &hits
}

// A proof file that is not in the served form: indented, with a final
// line feed. The service is asked about exactly these bytes.
const unservedForm = "{\n  \"proof\": {\"type\": \"block\"},\n  \"signature\": {}\n}\n"

// AskRemoteCtx posts the file's bytes exactly as given to POST
// /proof/raw/verify, as the proof media type, with offline=true for an
// offline run and expected_hash exactly as given when one is, and gives
// the label of the 200's report. Mutants killed: the body parsed and
// re-encoded (the posted bytes differ); the offline parameter left out
// (the query is only the hash); expected_hash left out, or sent when none
// is given (the query differs); the label not read from the report.
func TestAskRemoteCtx_PostsTheBytesAsGiven(t *testing.T) {
	hash := " AB00 "
	for _, tc := range []struct {
		name  string
		opts  RemoteOptions
		query string
		label string
	}{
		{"online, no hash", RemoteOptions{}, "", "verified"},
		{"offline, no hash", RemoteOptions{Offline: true}, "offline=true", "internally-consistent"},
		{"offline, with a hash", RemoteOptions{Offline: true, ExpectedHash: &hash}, "expected_hash=+AB00+&offline=true", "failed"},
	} {
		url, last, hits := answering(t, http.StatusOK, `{"label":"`+tc.label+`","rows":[],"verified":false}`)
		tc.opts.APIURL = url
		label, err := AskRemoteCtx(context.Background(), []byte(unservedForm), tc.opts)
		if err != nil || label != tc.label || hits.Load() != 1 {
			t.Errorf("%s: label %q, error %v, %d requests; want %q, no error and one request", tc.name, label, err, hits.Load(), tc.label)
			continue
		}
		if last.method != http.MethodPost || last.path != "/proof/raw/verify" || last.query != tc.query ||
			last.contentType != "application/vnd.truestamp.proof+json" || string(last.body) != unservedForm {
			t.Errorf("%s: sent %s %s?%s as %q with the body %q; want POST /proof/raw/verify?%s as the proof media type with the file's bytes",
				tc.name, last.method, last.path, last.query, last.contentType, last.body, tc.query)
		}
	}
}

// Only a 200 is a verdict, and only when its report carries a label of
// SPEC.md section 10.5: a non-200 whose body holds a report labelled
// verified, a 200 with no label, one with a label the spec does not
// define and one that is not JSON each give no label and an error. Mutants
// killed: AskRemoteCtx without its status check (the 500's verified label
// is returned); without its label check (the "maybe" label is returned).
func TestAskRemoteCtx_OnlyALabelledReportIsAVerdict(t *testing.T) {
	const report = `{"label":"verified"}`
	for _, tc := range []struct {
		status int
		body   string
		want   string
	}{
		{http.StatusInternalServerError, report, "API error (HTTP 500): " + report},
		{http.StatusCreated, report, "API error (HTTP 201): " + report},
		{http.StatusOK, `{"rows":[]}`, "the service's report carries no label"},
		{http.StatusOK, `{"label":"maybe"}`, `the service's report carries the label "maybe", which SPEC.md section 10.5 does not define`},
		{http.StatusOK, `not json`, "the service's answer is not a report: invalid character 'o' in literal null (expecting 'u')"},
	} {
		url, _, hits := answering(t, tc.status, tc.body)
		label, err := AskRemoteCtx(context.Background(), []byte(unservedForm), RemoteOptions{APIURL: url})
		if label != "" || err == nil || err.Error() != tc.want || hits.Load() != 1 {
			t.Errorf("HTTP %d %s: label %q, error %v, %d requests; want no label and the error %q",
				tc.status, tc.body, label, err, hits.Load(), tc.want)
		}
	}
}

// The service's refusal, an error whose meta.code names why, is a
// RemoteRejectionError carrying the status, the code and the detail, and
// its message names the detail and not the code; any other error is a
// plain error naming its detail, or its title where it has no detail.
// Mutant killed: parseAPIError without its meta.code case (the refusal
// becomes a plain error and errors.As finds no RemoteRejectionError).
func TestAskRemoteCtx_TheServicesRefusalIsARemoteRejection(t *testing.T) {
	for _, tc := range []struct {
		name      string
		status    int
		body      string
		rejection *RemoteRejectionError
		message   string
	}{
		{"a bare rejection", http.StatusBadRequest,
			`{"errors":[{"status":"400","code":"invalid","detail":"This file is not a Truestamp proof.","meta":{"code":"invalid_envelope","sub_reason":"members"}}]}`,
			&RemoteRejectionError{StatusCode: 400, Code: "invalid_envelope", Detail: "This file is not a Truestamp proof."},
			"the service refused the request (HTTP 400): This file is not a Truestamp proof."},
		{"too large", http.StatusRequestEntityTooLarge,
			`{"errors":[{"status":"413","title":"Request Entity Too Large","meta":{"code":"input_too_large"}}]}`,
			&RemoteRejectionError{StatusCode: 413, Code: "input_too_large", Detail: "Request Entity Too Large"},
			"the service refused the request (HTTP 413): Request Entity Too Large"},
		{"no code, a detail", http.StatusBadRequest,
			`{"errors":[{"status":"400","title":"Bad request","detail":"offline must be a boolean."}]}`,
			nil, "API error (HTTP 400): offline must be a boolean."},
		{"no code, only a title", http.StatusBadRequest,
			`{"errors":[{"status":"400","title":"Bad request"}]}`,
			nil, "API error (HTTP 400): Bad request"},
	} {
		url, _, _ := answering(t, tc.status, tc.body)
		label, err := AskRemoteCtx(context.Background(), []byte(unservedForm), RemoteOptions{APIURL: url})
		if label != "" || err == nil || err.Error() != tc.message {
			t.Errorf("%s: label %q, error %v; want no label and the error %q", tc.name, label, err, tc.message)
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

// A request that never reaches the service gives no label and an error,
// after one attempt. Mutant killed: AskRemoteCtx returning a label with
// the transport error (the label is not empty).
func TestAskRemoteCtx_AFailedRequestIsNoVerdict(t *testing.T) {
	srv := httptest.NewServer(http.NotFoundHandler())
	url := srv.URL
	srv.Close()
	label, err := AskRemoteCtx(context.Background(), []byte(unservedForm), RemoteOptions{APIURL: url})
	if label != "" || err == nil || !strings.HasPrefix(err.Error(), "API request failed: ") {
		t.Errorf("a closed server: label %q, error %v; want no label and an error that the request failed", label, err)
	}
}

// slowly is a loopback stand-in for the raw verify route that answers a
// verified report after delay, or answers nothing when the request ends
// first.
func slowly(t *testing.T, delay time.Duration) string {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Read the body first, so the server sees the request end.
		_, _ = io.ReadAll(r.Body)
		select {
		case <-r.Context().Done():
			return
		case <-time.After(delay):
		}
		_, _ = w.Write([]byte(`{"label":"verified"}`))
	}))
	t.Cleanup(srv.Close)
	return srv.URL
}

// The request waits for the larger of the configured http_timeout and the
// floor, whatever the shared client's timeout, and no longer: an answer
// after the shared client's timeout but within the floor, or within a
// timeout above the floor, is the verdict, and a service that answers
// within neither gives no label and the deadline's error, no sooner than
// the larger of the two. The shared client's timeout and the floor are
// shortened here. Mutants killed: the shared client's timeout governing
// the request (the first case's answer comes after it); the request
// without any bound (the third case's late answer is a verdict); the
// floor alone bounding it (the second case's answer comes after the
// floor).
func TestAskRemoteCtx_WaitsForTheLargerOfTheTimeoutAndTheFloor(t *testing.T) {
	floor := remoteFloor
	t.Cleanup(func() {
		remoteFloor = floor
		httpclient.Init(10 * time.Second)
	})
	httpclient.Init(100 * time.Millisecond)
	for _, tc := range []struct {
		name                  string
		timeout, floor, delay time.Duration
		label                 string
	}{
		{"an answer within the floor", 100 * time.Millisecond, 2 * time.Second, 500 * time.Millisecond, "verified"},
		{"an answer within the timeout", 2 * time.Second, 100 * time.Millisecond, 500 * time.Millisecond, "verified"},
		{"no answer within either", 100 * time.Millisecond, 300 * time.Millisecond, 5 * time.Second, ""},
	} {
		remoteFloor = tc.floor
		start := time.Now()
		label, err := AskRemoteCtx(context.Background(), []byte(unservedForm), RemoteOptions{APIURL: slowly(t, tc.delay), HTTPTimeout: tc.timeout})
		elapsed := time.Since(start)
		if tc.label != "" {
			if label != tc.label || err != nil {
				t.Errorf("%s: label %q, error %v; want %q and no error", tc.name, label, err, tc.label)
			}
			continue
		}
		if label != "" || !errors.Is(err, context.DeadlineExceeded) || !strings.HasPrefix(err.Error(), "API request failed: ") || elapsed < tc.floor {
			t.Errorf("%s: label %q, error %v after %v; want no label and the deadline's error after %v or more", tc.name, label, err, elapsed, tc.floor)
		}
	}
}
