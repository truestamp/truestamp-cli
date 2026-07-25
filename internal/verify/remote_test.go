// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: MIT

package verify

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/truestamp/truestamp-cli/internal/auth"
)

// TestMain installs an api-key Authorizer for the whole package so
// RunRemote stamps the Authorization header. The key matches the
// "Bearer test-key" header TestRunRemote_Success asserts; tests that
// pass other key strings to RemoteOptions no longer carry a credential
// of their own (the field was removed) and rely on this default.
func TestMain(m *testing.M) {
	auth.SetDefault(auth.APIKeyAuthorizer("test-key"))
	code := m.Run()
	auth.SetDefault(nil)
	os.Exit(code)
}

// makeProofFile writes a minimal valid compact-format JSON proof to a temp file.
func makeProofFile(t *testing.T) string {
	t.Helper()
	p := map[string]any{
		"v":   1,
		"t":   20,
		"pk":  "CTwMqDZnPd/QTLSq8aTeSD3a+j2DQxKcGfhhIYJQ65Y=",
		"sig": "AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA==",
		"ts":  "2026-04-06T23:25:06Z",
		"s": map[string]any{
			"id":  "01HJHB01T8FYZ7YTR9P5N62K5B",
			"d":   map[string]any{"name": "test"},
			"mh":  "ccddccddccddccddccddccddccddccddccddccddccddccddccddccddccddccdd",
			"kid": "4ceefa4a",
		},
		"b": map[string]any{
			"id":  "019cf813-99b8-730a-84f1-5a711a9c355e",
			"ph":  "1111111111111111111111111111111111111111111111111111111111111111",
			"mr":  "2222222222222222222222222222222222222222222222222222222222222222",
			"mh":  "4444444444444444444444444444444444444444444444444444444444444444",
			"kid": "4ceefa4a",
		},
		"ip": "AA",
		"cx": []any{
			map[string]any{"t": 40, "net": "testnet", "tx": "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", "memo": "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb", "l": 1, "ep": "AA"},
		},
	}
	data, err := json.Marshal(p)
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "proof.json")
	if err := os.WriteFile(path, data, 0644); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestRunRemote_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify headers
		if r.Header.Get("Authorization") != "Bearer test-key" {
			t.Errorf("expected Authorization header, got %q", r.Header.Get("Authorization"))
		}
		if r.Header.Get("tenant") != "team-123" {
			t.Errorf("expected tenant header 'team-123', got %q", r.Header.Get("tenant"))
		}

		w.WriteHeader(201)
		resp := apiEnvelope{
			Result: &apiResult{
				ProofVersion: intPtr(1),
				ItemID:       strPtr("01TESTITEM"),
				GeneratedAt:  strPtr("2025-01-01T00:00:00Z"),
				Passed:       true,
				Temporal: TemporalSummary{
					SubmittedAt: "2025-01-01T00:01:00Z",
					CommittedAt: "2025-01-01T00:02:00Z",
				},
				Steps: []Step{
					{Group: "Signing Keys", Status: StatusPass, Message: "Key valid"},
					{Group: "Proof Bundle", Status: StatusWarn, Message: "Stale claim"},
				},
			},
		}
		data, _ := json.Marshal(resp)
		w.Write(data)
	}))
	defer server.Close()

	proofFile := makeProofFile(t)

	report, err := RunRemote(proofFile, RemoteOptions{
		APIURL: server.URL,
		Team:   "team-123",
	})
	if err != nil {
		t.Fatalf("RunRemote error: %v", err)
	}

	if !report.Passed() {
		t.Error("expected report to pass")
	}
	if report.ProofVersion != 1 {
		t.Errorf("ProofVersion: got %d, want 1", report.ProofVersion)
	}
	if report.SubjectID != "01TESTITEM" {
		t.Errorf("SubjectID: got %q, want 01TESTITEM", report.SubjectID)
	}
	if report.Filename != proofFile {
		t.Errorf("Filename: got %q, want %q", report.Filename, proofFile)
	}
	if len(report.Steps) != 2 {
		t.Fatalf("Steps: got %d, want 2", len(report.Steps))
	}
	if report.Steps[0].Status != StatusPass {
		t.Errorf("Steps[0].Status: got %d, want StatusPass", report.Steps[0].Status)
	}
	if report.Steps[1].Status != StatusWarn {
		t.Errorf("Steps[1].Status: got %d, want StatusWarn", report.Steps[1].Status)
	}
	if report.Temporal.CommittedAt != "2025-01-01T00:02:00Z" {
		t.Errorf("Temporal.CommittedAt: got %q", report.Temporal.CommittedAt)
	}
}

func TestRunRemote_VerificationFailed(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(201)
		resp := apiEnvelope{
			Result: &apiResult{
				ProofVersion: intPtr(1),
				ItemID:       strPtr("01TESTITEM"),
				Passed:       false,
				Steps: []Step{
					{Group: "Item", Status: StatusFail, Message: "Claims hash mismatch"},
				},
			},
		}
		data, _ := json.Marshal(resp)
		w.Write(data)
	}))
	defer server.Close()

	report, err := RunRemote(makeProofFile(t), RemoteOptions{
		APIURL: server.URL,
	})
	if err != nil {
		t.Fatalf("RunRemote error: %v", err)
	}
	if report.Passed() {
		t.Error("expected report to fail")
	}
	if report.FailedCount() != 1 {
		t.Errorf("FailedCount: got %d, want 1", report.FailedCount())
	}
}

func TestRunRemote_NoTenantHeader_WhenTeamEmpty(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("tenant") != "" {
			t.Errorf("expected no tenant header, got %q", r.Header.Get("tenant"))
		}
		w.WriteHeader(201)
		resp := apiEnvelope{Result: &apiResult{Passed: true}}
		data, _ := json.Marshal(resp)
		w.Write(data)
	}))
	defer server.Close()

	_, err := RunRemote(makeProofFile(t), RemoteOptions{
		APIURL: server.URL,
		Team:   "",
	})
	if err != nil {
		t.Fatalf("RunRemote error: %v", err)
	}
}

func TestRunRemote_APIError401(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(401)
		w.Write([]byte(`{"errors":[{"status":"401","detail":"Authentication required"}]}`))
	}))
	defer server.Close()

	_, err := RunRemote(makeProofFile(t), RemoteOptions{
		APIURL: server.URL,
	})
	if err == nil {
		t.Fatal("expected error for 401")
	}
	if got := err.Error(); got == "" {
		t.Error("error message should not be empty")
	}
}

func TestRunRemote_APIError400(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(400)
		w.Write([]byte(`{"errors":[{"status":"400","detail":"Invalid proof: missing_proof"}]}`))
	}))
	defer server.Close()

	_, err := RunRemote(makeProofFile(t), RemoteOptions{
		APIURL: server.URL,
	})
	if err == nil {
		t.Fatal("expected error for 400")
	}
}

func TestRunRemote_MissingFile(t *testing.T) {
	_, err := RunRemote("/nonexistent/proof.json", RemoteOptions{
		APIURL: "http://localhost",
	})
	if err == nil {
		t.Fatal("expected error for missing file")
	}
}

func TestRunRemote_InvalidJSON(t *testing.T) {
	path := filepath.Join(t.TempDir(), "bad.json")
	os.WriteFile(path, []byte("not json"), 0644)

	_, err := RunRemote(path, RemoteOptions{
		APIURL: "http://localhost",
	})
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}

func TestStatusJSON_RoundTrip(t *testing.T) {
	statuses := []Status{StatusPass, StatusFail, StatusSkip, StatusWarn, StatusInfo}
	for _, s := range statuses {
		data, err := json.Marshal(s)
		if err != nil {
			t.Fatalf("Marshal %d: %v", s, err)
		}
		var got Status
		if err := json.Unmarshal(data, &got); err != nil {
			t.Fatalf("Unmarshal %s: %v", data, err)
		}
		if got != s {
			t.Errorf("round-trip: got %d, want %d", got, s)
		}
	}
}

func TestStatusJSON_UnknownString(t *testing.T) {
	var s Status
	if err := json.Unmarshal([]byte(`"future_status"`), &s); err == nil {
		t.Error("an unrecognized status must be rejected, not silently accepted")
	}
}

// makeEntropyProofFile writes a minimal valid compact entropy proof (NIST beacon) to a temp file.
func makeEntropyProofFile(t *testing.T) string {
	t.Helper()
	p := map[string]any{
		"v":   1,
		"t":   30,
		"pk":  "CTwMqDZnPd/QTLSq8aTeSD3a+j2DQxKcGfhhIYJQ65Y=",
		"sig": "AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA==",
		"ts":  "2026-04-06T23:25:06Z",
		"s": map[string]any{
			"id":  "019d2ae3-865c-7651-9923-b14c55bc8e33",
			"d":   map[string]any{"pulse": map[string]any{"outputValue": "ABC", "pulseIndex": 100, "chainIndex": 2, "version": "2.0", "timeStamp": "2026-03-26T16:02:00.000Z"}},
			"mh":  "5555555555555555555555555555555555555555555555555555555555555555",
			"kid": "4ceefa4a",
		},
		"b": map[string]any{
			"id":  "019cf813-99b8-730a-84f1-5a711a9c355e",
			"ph":  "1111111111111111111111111111111111111111111111111111111111111111",
			"mr":  "2222222222222222222222222222222222222222222222222222222222222222",
			"mh":  "4444444444444444444444444444444444444444444444444444444444444444",
			"kid": "4ceefa4a",
		},
		"ip": "AA",
		"cx": []any{
			map[string]any{"t": 40, "net": "testnet", "tx": "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", "memo": "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb", "l": 1, "ep": "AA"},
		},
	}
	data, err := json.Marshal(p)
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "entropy-proof.json")
	if err := os.WriteFile(path, data, 0644); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestRunRemote_ItemProof_SubjectType(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(201)
		resp := apiEnvelope{Result: &apiResult{Passed: true}}
		data, _ := json.Marshal(resp)
		w.Write(data)
	}))
	defer server.Close()

	report, err := RunRemote(makeProofFile(t), RemoteOptions{
		APIURL: server.URL,
	})
	if err != nil {
		t.Fatalf("RunRemote error: %v", err)
	}
	if report.SubjectType != "item" {
		t.Errorf("SubjectType: got %q, want item", report.SubjectType)
	}
	if report.SubjectID == "" {
		t.Error("SubjectID should not be empty for item proof")
	}
	if report.Claims.Name != "test" {
		t.Errorf("Claims.Name: got %q, want test", report.Claims.Name)
	}
	if report.ChainLength != 1 {
		t.Errorf("ChainLength: got %d, want 1", report.ChainLength)
	}
}

func TestRunRemote_EntropyProof_SubjectType(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(201)
		resp := apiEnvelope{Result: &apiResult{Passed: true}}
		data, _ := json.Marshal(resp)
		w.Write(data)
	}))
	defer server.Close()

	report, err := RunRemote(makeEntropyProofFile(t), RemoteOptions{
		APIURL: server.URL,
	})
	if err != nil {
		t.Fatalf("RunRemote error: %v", err)
	}
	if report.SubjectType != "entropy_nist" {
		t.Errorf("SubjectType: got %q, want entropy_nist", report.SubjectType)
	}
	if report.SubjectID != "019d2ae3-865c-7651-9923-b14c55bc8e33" {
		t.Errorf("SubjectID: got %q", report.SubjectID)
	}
	if report.EntropySubject.Source != "NIST Beacon" {
		t.Errorf("EntropySubject.Source: got %q, want NIST Beacon", report.EntropySubject.Source)
	}
	if report.EntropySubject.RawSource != "entropy_nist" {
		t.Errorf("EntropySubject.RawSource: got %q, want entropy_nist", report.EntropySubject.RawSource)
	}
	if report.EntropySubject.PulseIndex != 100 {
		t.Errorf("EntropySubject.PulseIndex: got %d, want 100", report.EntropySubject.PulseIndex)
	}
	if report.ChainLength != 1 {
		t.Errorf("ChainLength: got %d, want 1", report.ChainLength)
	}
	// The server sent no temporal object, so the observation timestamp must
	// fall back to the subject-id (UUIDv7) derivation in populateFromBundle.
	if report.Temporal.CapturedAt == "" {
		t.Error("Temporal.CapturedAt: expected local subject-id fallback when server omits temporal, got empty")
	}
}

// TestRunRemote_EntropyProof_InsertedAtTemporal asserts the CLI reads the
// server's renamed entropy temporal key `inserted_at` into its report. The
// server value is distinct from the subject-id-derived local fallback, so a
// match proves the wire value flowed through (and was not overwritten by
// populateFromBundle). Raw JSON is written so the real wire key is exercised
// rather than a Go struct round-trip.
func TestRunRemote_EntropyProof_InsertedAtTemporal(t *testing.T) {
	const wantTemporal = "2020-01-02T03:04:05Z"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(201)
		w.Write([]byte(`{"result":{"passed":true,"temporal":{"inserted_at":"` + wantTemporal + `","committed_at":"2026-04-06T23:25:06Z"}}}`))
	}))
	defer server.Close()

	report, err := RunRemote(makeEntropyProofFile(t), RemoteOptions{APIURL: server.URL})
	if err != nil {
		t.Fatalf("RunRemote error: %v", err)
	}
	if report.Temporal.CapturedAt != wantTemporal {
		t.Errorf("Temporal.CapturedAt: got %q, want %q (server's inserted_at key must map through)", report.Temporal.CapturedAt, wantTemporal)
	}
}

func intPtr(i int) *int       { return &i }
func strPtr(s string) *string { return &s }

func TestParseAPIError_HTMLPage(t *testing.T) {
	err := parseAPIError(502, []byte("<html>oops</html>"))
	if err == nil {
		t.Fatal("expected error")
	}
	if !contains(err.Error(), "HTML") {
		t.Errorf("error should mention HTML: %v", err)
	}
}

func TestParseAPIError_TitleOnly(t *testing.T) {
	err := parseAPIError(404, []byte(`{"errors":[{"title":"Not Found"}]}`))
	if err == nil || !contains(err.Error(), "Not Found") {
		t.Errorf("expected title in error, got %v", err)
	}
}

func TestParseAPIError_Unparseable(t *testing.T) {
	err := parseAPIError(500, []byte("not json and not html"))
	if err == nil {
		t.Fatal("expected error")
	}
	if !contains(err.Error(), "not json") {
		t.Errorf("error should contain body prefix: %v", err)
	}
}

func TestRunRemote_CBORInput(t *testing.T) {
	// Build a CBOR proof from a JSON proof then submit it to RunRemote.
	dir := t.TempDir()
	jsonPath := makeProofFile(t)
	data, err := os.ReadFile(jsonPath)
	if err != nil {
		t.Fatal(err)
	}

	// The remote package doesn't expose proof.MarshalCBOR directly here.
	// We simulate a CBOR input by parsing via proof.ParseBytes then using
	// MarshalCBOR — do that through the public API.
	b, err := parseBundle(data)
	if err != nil {
		t.Fatalf("parseBundle: %v", err)
	}
	cborBytes, err := b.MarshalCBOR()
	if err != nil {
		t.Fatalf("MarshalCBOR: %v", err)
	}
	cborPath := filepath.Join(dir, "proof.cbor")
	if err := os.WriteFile(cborPath, cborBytes, 0644); err != nil {
		t.Fatal(err)
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Decode request body to verify the server saw JSON (not CBOR).
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Errorf("request body not JSON: %v", err)
		}
		w.WriteHeader(201)
		resp := apiEnvelope{Result: &apiResult{Passed: true}}
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	_, err = RunRemote(cborPath, RemoteOptions{APIURL: server.URL})
	if err != nil {
		t.Fatalf("RunRemote(cbor): %v", err)
	}
}

func TestRunRemote_MalformedJSONResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(200)
		_, _ = w.Write([]byte("not json"))
	}))
	defer server.Close()
	_, err := RunRemote(makeProofFile(t), RemoteOptions{APIURL: server.URL})
	if err == nil {
		t.Fatal("expected parse error for non-JSON response")
	}
}

func TestRunRemote_EmptyResultField(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"errors":[]}`))
	}))
	defer server.Close()
	_, err := RunRemote(makeProofFile(t), RemoteOptions{APIURL: server.URL})
	if err == nil || !contains(err.Error(), "result") {
		t.Errorf("expected result-missing error, got %v", err)
	}
}

func TestRunRemote_UnreachableHost(t *testing.T) {
	_, err := RunRemote(makeProofFile(t), RemoteOptions{
		APIURL: "http://127.0.0.1:1",
	})
	if err == nil {
		t.Fatal("expected connection error")
	}
}

func TestRunRemote_ExpectedHashIncluded(t *testing.T) {
	received := ""
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		_ = json.NewDecoder(r.Body).Decode(&body)
		if data, ok := body["data"].(map[string]any); ok {
			if hash, ok := data["expected_hash"].(string); ok {
				received = hash
			}
		}
		w.WriteHeader(201)
		_, _ = w.Write([]byte(`{"result":{"passed":true}}`))
	}))
	defer server.Close()

	_, err := RunRemote(makeProofFile(t), RemoteOptions{
		APIURL:       server.URL,
		ExpectedHash: "deadbeef",
	})
	if err != nil {
		t.Fatal(err)
	}
	if received != "deadbeef" {
		t.Errorf("expected_hash: got %q, want deadbeef", received)
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && indexOfSubstr(s, substr) >= 0
}

func indexOfSubstr(s, substr string) int {
	for i := 0; i+len(substr) <= len(s); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}

// parseBundle exposes proof.ParseBytes for the CBOR test. Keeps the
// import list of remote_test.go free of the proof package.
func parseBundle(data []byte) (bundle, error) {
	return parseProofBytes(data)
}

// bundle is the minimal interface the CBOR round-trip test needs from
// the proof package. The real ParseBytes result satisfies it via
// MarshalCBOR().
type bundle interface {
	MarshalCBOR() ([]byte, error)
}

// --- Fail-closed --remote (Appendix E.22's verdict rule) -------------
//
// Every case below is a real server response shape that used to be
// rendered "verified" at exit 0. The governing rule is that an
// unrecognised, absent or ambiguous result must never be scored as
// passing: E.22's verdict rule ("a proof passes when no step is fail")
// is only sound when every status is known.

// remoteServer stands up a /proof/verify stub returning the given raw
// JSON body, and returns the report RunRemote built from it.
func remoteServer(t *testing.T, body string, proofFile string, opts RemoteOptions) (*Report, error) {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(200)
		_, _ = w.Write([]byte(body))
	}))
	t.Cleanup(srv.Close)
	opts.APIURL = srv.URL
	return RunRemote(proofFile, opts)
}

func TestRunRemote_UnknownStatusVocabulary_IsNotAPass(t *testing.T) {
	// The judges' mock: a server whose step vocabulary is "failed" /
	// "error" rather than "fail". Both used to decode to StatusInfo,
	// which is verdict-neutral, so the run printed VERIFIED at exit 0
	// while rendering "Ed25519 signature is INVALID".
	body := `{"result":{"proof_version":1,"subject_type":"item","passed":false,"steps":[
	  {"group":"Proof Signature","category":"cryptographic","status":"failed","message":"Ed25519 signature is INVALID"},
	  {"group":"Inclusion Proof","category":"cryptographic","status":"error","message":"derived root does not match block merkle root"}]}}`
	report, err := remoteServer(t, body, makeProofFile(t), RemoteOptions{})
	if err != nil {
		t.Fatalf("RunRemote: %v", err)
	}
	if report.Passed() {
		t.Fatal("a report whose only statuses are unreadable must not pass")
	}
	if report.FailedCount() != 2 {
		t.Errorf("FailedCount: got %d, want 2", report.FailedCount())
	}
	for _, s := range report.Steps {
		if !contains(s.Message, "cannot be read as having passed") {
			t.Errorf("step does not disclose why it was graded fail: %q", s.Message)
		}
	}
	if got := computeResult(report); got != "failed" {
		t.Errorf("result: got %q, want failed", got)
	}
}

func TestRunRemote_AbsentStatusKey_IsNotAPass(t *testing.T) {
	// Step.Status's zero value is StatusPass and Status.UnmarshalJSON is
	// never invoked for an absent key, so a step reading "Proof
	// signature invalid (Ed25519)" scored as a pass.
	body := `{"result":{"proof_version":1,"subject_type":"item","passed":false,"steps":[
	  {"group":"Proof Signature","category":"cryptographic","message":"Proof signature invalid (Ed25519)"},
	  {"group":"Block Hash","category":"cryptographic","message":"whatever"}]}}`
	report, err := remoteServer(t, body, makeProofFile(t), RemoteOptions{})
	if err != nil {
		t.Fatalf("RunRemote: %v", err)
	}
	if report.Passed() {
		t.Fatal("a step with no status key must not be scored as a pass")
	}
	if report.PassCount() != 0 {
		t.Errorf("PassCount: got %d, want 0", report.PassCount())
	}
}

func TestRunRemote_ServerSaysNotPassedWithNoFailingStep(t *testing.T) {
	// passed:false with a step list that carries no failure. The
	// server's own verdict used to be parsed and never read.
	body := `{"result":{"proof_version":1,"subject_type":"item","passed":false,"steps":[
	  {"group":"Signing Key","category":"cryptographic","status":"pass","message":"ok"}]}}`
	report, err := remoteServer(t, body, makeProofFile(t), RemoteOptions{})
	if err != nil {
		t.Fatalf("RunRemote: %v", err)
	}
	if report.Passed() {
		t.Fatal("the server reported passed:false; the CLI must not report verified")
	}
	found := false
	for _, s := range report.Steps {
		if s.Group == groupServerVerdict && s.Status == StatusFail {
			found = true
		}
	}
	if !found {
		t.Errorf("no Server Verdict failure row: %+v", report.Steps)
	}
}

func TestRunRemote_NoSteps_IsNotAPass(t *testing.T) {
	body := `{"result":{"proof_version":1,"subject_type":"item","passed":true,"steps":[]}}`
	report, err := remoteServer(t, body, makeProofFile(t), RemoteOptions{})
	if err != nil {
		t.Fatalf("RunRemote: %v", err)
	}
	if report.Passed() {
		t.Fatal("a response carrying no steps establishes nothing and must not read as verified")
	}

	// And a row the CLI itself appended (here E.7's passing comparison)
	// must not rescue the empty response.
	real := "b47cc0f104b62d4c7c30bcd68fd8e67613e287dc4ad8c310ef10cbadea9c4380"
	report, err = remoteServer(t, body, makeItemProofFileWithHash(t, real),
		RemoteOptions{ExpectedHash: real})
	if err != nil {
		t.Fatalf("RunRemote: %v", err)
	}
	if report.Passed() {
		t.Fatalf("the CLI's own row rescued a response with no server steps: %+v", report.Steps)
	}
}

func TestRunRemote_ServerAgreesOnAPass_StaysVerified(t *testing.T) {
	// The control: a well-formed passing response is still a pass, so
	// the fail-closed rules above cannot be satisfied by failing
	// everything.
	body := `{"result":{"proof_version":1,"subject_type":"item","passed":true,"steps":[
	  {"group":"Signing Key","category":"cryptographic","status":"pass","message":"ok"},
	  {"group":"Key Binding","category":"cryptographic","status":"skip","message":"not performed"},
	  {"group":"Temporal Info","category":"timing","status":"info","message":"timeline"}]}}`
	report, err := remoteServer(t, body, makeProofFile(t), RemoteOptions{})
	if err != nil {
		t.Fatalf("RunRemote: %v", err)
	}
	if !report.Passed() {
		t.Fatalf("a well-formed passing response must still pass: %+v", report.Steps)
	}
	if got := computeResult(report); got != "verified" {
		t.Errorf("result: got %q, want verified", got)
	}
}

// --- E.7 in remote mode ---------------------------------------------

// makeItemProofFileWithHash writes an item proof whose s.d carries a
// file hash, so E.7's comparison has both operands.
func makeItemProofFileWithHash(t *testing.T, dataHash string) string {
	t.Helper()
	p := map[string]any{
		"v": 1, "t": 20,
		"pk":  "CTwMqDZnPd/QTLSq8aTeSD3a+j2DQxKcGfhhIYJQ65Y=",
		"sig": "AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA==",
		"ts":  "2026-04-06T23:25:06Z",
		"s": map[string]any{
			"id":  "01HJHB01T8FYZ7YTR9P5N62K5B",
			"d":   map[string]any{"name": "test", "hash": dataHash, "hash_type": "sha256"},
			"mh":  "ccddccddccddccddccddccddccddccddccddccddccddccddccddccddccddccdd",
			"kid": "4ceefa4a",
		},
		"b": map[string]any{
			"id":  "019cf813-99b8-730a-84f1-5a711a9c355e",
			"ph":  "1111111111111111111111111111111111111111111111111111111111111111",
			"mr":  "2222222222222222222222222222222222222222222222222222222222222222",
			"mh":  "4444444444444444444444444444444444444444444444444444444444444444",
			"kid": "4ceefa4a",
		},
		"ip": "AA",
		"cx": []any{map[string]any{
			"t": 40, "net": "testnet",
			"tx":   "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
			"memo": "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
			"l":    1, "ep": "AA",
		}},
	}
	data, err := json.Marshal(p)
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "proof.json")
	if err := os.WriteFile(path, data, 0644); err != nil {
		t.Fatal(err)
	}
	return path
}

const remoteAllPassBody = `{"result":{"proof_version":1,"subject_type":"item","passed":true,"steps":[
  {"group":"Signing Key","category":"cryptographic","status":"pass","message":"ok"}]}}`

func TestRunRemote_ExpectedHashMismatch_FailsLocally(t *testing.T) {
	// A server that ignores expected_hash entirely. --hash is the flag
	// whose whole purpose is "confirm my local file is the timestamped
	// one"; in remote mode it used to be decorative, and this exact
	// invocation exits 1 in local mode.
	real := "b47cc0f104b62d4c7c30bcd68fd8e67613e287dc4ad8c310ef10cbadea9c4380"
	wrong := "deadbeef00000000000000000000000000000000000000000000000000000000"
	report, err := remoteServer(t, remoteAllPassBody, makeItemProofFileWithHash(t, real),
		RemoteOptions{ExpectedHash: wrong})
	if err != nil {
		t.Fatalf("RunRemote: %v", err)
	}
	if report.Passed() {
		t.Fatal("a refuted --hash must not exit 0 in remote mode")
	}
	if got := computeResult(report); got != "hash_mismatch" {
		t.Errorf("result: got %q, want hash_mismatch", got)
	}
	out := BuildJSONOutput(report)
	if !out.HashComparison.Supplied || out.HashComparison.Matched {
		t.Errorf("hash_comparison: %+v", out.HashComparison)
	}
	if out.HashComparison.Found != real {
		t.Errorf("found: got %q, want the bundle's own s.d.hash", out.HashComparison.Found)
	}
}

func TestRunRemote_ExpectedHashMatch_IsFullyVerified(t *testing.T) {
	real := "b47cc0f104b62d4c7c30bcd68fd8e67613e287dc4ad8c310ef10cbadea9c4380"
	report, err := remoteServer(t, remoteAllPassBody, makeItemProofFileWithHash(t, real),
		RemoteOptions{ExpectedHash: real})
	if err != nil {
		t.Fatalf("RunRemote: %v", err)
	}
	if !report.Passed() {
		t.Fatalf("a matching --hash must still pass: %+v", report.Steps)
	}
	if got := computeResult(report); got != "fully_verified" {
		t.Errorf("result: got %q, want fully_verified", got)
	}
}

func TestRunRemote_ServerHashMatchedDisagreement_Fails(t *testing.T) {
	// The server echoes hash_provided and claims hash_matched:true over
	// a hash this process can see does not match the bundle it posted.
	real := "b47cc0f104b62d4c7c30bcd68fd8e67613e287dc4ad8c310ef10cbadea9c4380"
	wrong := "deadbeef00000000000000000000000000000000000000000000000000000000"
	body := `{"result":{"proof_version":1,"subject_type":"item","passed":true,
	  "hash_provided":"` + wrong + `","hash_matched":true,"steps":[
	  {"group":"Hash Comparison","category":"data_integrity","status":"pass","message":"Provided hash matches subject data hash"}]}}`
	report, err := remoteServer(t, body, makeItemProofFileWithHash(t, real),
		RemoteOptions{ExpectedHash: wrong})
	if err != nil {
		t.Fatalf("RunRemote: %v", err)
	}
	if report.Passed() {
		t.Fatal("a server-asserted match this verifier refutes must not pass")
	}
	if report.HashMatched() {
		t.Error("HashMatched must be false when the group carries a failure")
	}
	sawDisagreement := false
	for _, s := range report.Steps {
		if contains(s.Message, "Expected-hash disagreement") {
			sawDisagreement = true
		}
	}
	if !sawDisagreement {
		t.Errorf("the server's hash_matched echo was not reconciled: %+v", report.Steps)
	}
}

func TestRunRemote_ExpectedHashOnNonItem_SkipsButRecordsSupplied(t *testing.T) {
	// E.7 REQUIRES the inapplicability skip, and E.22 requires "one was
	// supplied" to stay readable separately from "it matched".
	body := `{"result":{"proof_version":1,"subject_type":"entropy_nist","passed":true,"steps":[
	  {"group":"Signing Key","category":"cryptographic","status":"pass","message":"ok"}]}}`
	report, err := remoteServer(t, body, makeEntropyProofFile(t),
		RemoteOptions{ExpectedHash: "deadbeef"})
	if err != nil {
		t.Fatalf("RunRemote: %v", err)
	}
	if !report.Passed() {
		t.Fatalf("--hash on a non-item subject must never fail the proof: %+v", report.Steps)
	}
	out := BuildJSONOutput(report)
	if !out.HashComparison.Supplied {
		t.Error("hash_comparison.supplied must be true when --hash was supplied")
	}
	sawSkip := false
	for _, s := range report.Steps {
		if s.Group == groupHashComparison && s.Status == StatusSkip {
			sawSkip = true
		}
	}
	if !sawSkip {
		t.Errorf("no Hash Comparison skip row: %+v", report.Steps)
	}
}

// --- --type in remote mode ------------------------------------------

func TestRunRemote_TypeMismatch_ProducesAReportNotABareError(t *testing.T) {
	var posted map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var payload struct {
			Data map[string]any `json:"data"`
		}
		_ = json.NewDecoder(r.Body).Decode(&payload)
		posted = payload.Data
		w.WriteHeader(200)
		_, _ = w.Write([]byte(remoteAllPassBody))
	}))
	defer srv.Close()

	report, err := RunRemote(makeProofFile(t), RemoteOptions{
		APIURL:              srv.URL,
		ExpectedSubjectType: "beacon", // the fixture is t=20
	})
	if err != nil {
		t.Fatalf("a --type mismatch must produce a report, not an error: %v", err)
	}
	if report.Passed() {
		t.Fatal("a --type mismatch must fail the run")
	}
	sawSubjectType := false
	for _, s := range report.Steps {
		if s.Group == groupSubjectType && s.Status == StatusFail {
			sawSubjectType = true
			if !contains(s.Message, "--type beacon was requested") {
				t.Errorf("message: %q", s.Message)
			}
		}
	}
	if !sawSubjectType {
		t.Errorf("no Subject Type failure row: %+v", report.Steps)
	}
	// The rest of the report must still be there — that is the whole
	// point of grading the assertion instead of aborting on a 422.
	if len(report.Steps) < 2 {
		t.Errorf("the server's own steps were dropped: %+v", report.Steps)
	}
	if _, ok := posted["type"]; ok {
		t.Errorf("a mismatching type must not be forwarded (the server would 4xx and no report would come back): %v", posted)
	}
}

func TestRunRemote_TypeMatch_IsForwarded(t *testing.T) {
	var posted map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var payload struct {
			Data map[string]any `json:"data"`
		}
		_ = json.NewDecoder(r.Body).Decode(&payload)
		posted = payload.Data
		w.WriteHeader(200)
		_, _ = w.Write([]byte(remoteAllPassBody))
	}))
	defer srv.Close()

	report, err := RunRemote(makeProofFile(t), RemoteOptions{
		APIURL:              srv.URL,
		ExpectedSubjectType: "item",
	})
	if err != nil {
		t.Fatalf("RunRemote: %v", err)
	}
	if !report.Passed() {
		t.Fatalf("a matching --type must not fail: %+v", report.Steps)
	}
	if got := posted["type"]; got != "item" {
		t.Errorf("a matching type must still be forwarded for the server to re-check: %v", posted)
	}
}

// --- E.9 key attribution --------------------------------------------

func TestRunRemote_SigningKeyIsDerivedFromPk(t *testing.T) {
	// A rotated bundle: b.kid is 11223344 while the key id DERIVED from
	// pk is 4ceefa4a. E.9 blesses that divergence and E.16 signs with
	// the derived id, so the report must name the derived one as the
	// signer and keep b.kid for the places that describe the block.
	// Reading b.kid for both made a rotated proof print "Public key
	// valid, key_id: 4ceefa4a" next to "signed with key 11223344".
	path := makeItemProofFileWithHash(t, "b47cc0f104b62d4c7c30bcd68fd8e67613e287dc4ad8c310ef10cbadea9c4380")
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var doc map[string]any
	if err := json.Unmarshal(raw, &doc); err != nil {
		t.Fatal(err)
	}
	doc["b"].(map[string]any)["kid"] = "11223344"
	rotated, err := json.Marshal(doc)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, rotated, 0644); err != nil {
		t.Fatal(err)
	}

	report, err := remoteServer(t, remoteAllPassBody, path, RemoteOptions{})
	if err != nil {
		t.Fatalf("RunRemote: %v", err)
	}
	if report.SigningKeyID != "4ceefa4a" {
		t.Errorf("SigningKeyID: got %q, want the pk-derived 4ceefa4a", report.SigningKeyID)
	}
	if report.BlockSigningKeyID != "11223344" {
		t.Errorf("BlockSigningKeyID: got %q, want the bundle's b.kid 11223344", report.BlockSigningKeyID)
	}
}

func TestRunRemote_ServerClaimsMatchOverASkip_Fails(t *testing.T) {
	// E.7 REQUIRES an inapplicability skip for a non-item subject, so
	// this process established nothing about the caller's file. A server
	// that nonetheless asserts a match is claiming something no
	// comparison here supports, and must not be published as verified.
	body := `{"result":{"proof_version":1,"subject_type":"entropy_nist","passed":true,
	  "hash_provided":"deadbeef","hash_matched":true,"steps":[
	  {"group":"Hash Comparison","category":"data_integrity","status":"pass","message":"Provided hash matches subject data hash"}]}}`
	report, err := remoteServer(t, body, makeEntropyProofFile(t), RemoteOptions{ExpectedHash: "deadbeef"})
	if err != nil {
		t.Fatalf("RunRemote: %v", err)
	}
	if report.Passed() {
		t.Fatalf("a server-asserted match over an E.7 skip must not pass: %+v", report.Steps)
	}
	if report.HashMatched() {
		t.Error("HashMatched must be false when this verifier made no comparison")
	}
}

func TestRunRemote_ServerVerdictSurvivesACLIRaisedFailure(t *testing.T) {
	// passed:false with an all-passing step list, alongside a --hash the
	// CLI refutes. Deriving "did the server report a failure" from the
	// CLI's own view of the report let the server's verdict vanish, and
	// the run rendered "HASH MISMATCH - proof is valid" over a proof the
	// server had reported as not verified.
	real := "b47cc0f104b62d4c7c30bcd68fd8e67613e287dc4ad8c310ef10cbadea9c4380"
	wrong := "deadbeef00000000000000000000000000000000000000000000000000000000"
	body := `{"result":{"proof_version":1,"subject_type":"item","passed":false,"steps":[
	  {"group":"Signing Key","category":"cryptographic","status":"pass","message":"ok"}]}}`
	report, err := remoteServer(t, body, makeItemProofFileWithHash(t, real),
		RemoteOptions{ExpectedHash: wrong})
	if err != nil {
		t.Fatalf("RunRemote: %v", err)
	}
	if report.ProofPassed() {
		t.Errorf("the server's passed:false was dropped, so the proof reads as sound: %+v", report.Steps)
	}
	if got := computeResult(report); got != "failed" {
		t.Errorf("result: got %q, want failed", got)
	}
}

// TestRunRemote_ServerHashNotMatchedDisagreement_Fails is the converse of
// TestRunRemote_ServerHashMatchedDisagreement_Fails, and the arm that had
// no test at all: the server echoes the expected hash and reports it as
// NOT matching, while this process's own comparison against the bundle it
// posted says it does match.
//
// One of the two verifiers is wrong about the caller's data either way,
// and E.22 forbids publishing that as verified. Downgrading this single
// `r.fail` to a skip left the whole suite green, so a server able to
// answer hash_matched:false could quietly neutralise --hash in the one
// direction the CLI's local comparison cannot arbitrate on its own.
func TestRunRemote_ServerHashNotMatchedDisagreement_Fails(t *testing.T) {
	real := "b47cc0f104b62d4c7c30bcd68fd8e67613e287dc4ad8c310ef10cbadea9c4380"
	body := `{"result":{"proof_version":1,"subject_type":"item","passed":true,
	  "hash_provided":"` + real + `","hash_matched":false,"steps":[
	  {"group":"Proof Signature","category":"cryptographic","status":"pass","message":"Proof signature valid (Ed25519)"}]}}`

	report, err := remoteServer(t, body, makeItemProofFileWithHash(t, real),
		RemoteOptions{ExpectedHash: real})
	if err != nil {
		t.Fatalf("RunRemote: %v", err)
	}
	if report.Passed() {
		t.Fatalf("a server-asserted mismatch this verifier contradicts must not pass: %+v", report.Steps)
	}
	if report.HashMatched() {
		t.Error("HashMatched must be false when the group carries a failure")
	}
	var sawDisagreement bool
	for _, s := range report.Steps {
		if s.Group != groupHashComparison || s.Status != StatusFail {
			continue
		}
		if contains(s.Message, "Expected-hash disagreement") && contains(s.Message, "NOT matching") {
			sawDisagreement = true
		}
	}
	if !sawDisagreement {
		t.Errorf("the server's hash_matched:false echo was not reconciled: %+v", report.Steps)
	}
}

// makeClaimsOnlyItemProofFile writes a t=20 bundle whose `s.d` carries no
// `hash` — the claims-as-source-of-truth mode, where the claims content
// itself is what was timestamped and there is no file hash on the proof
// side to compare a caller's argument against.
func makeClaimsOnlyItemProofFile(t *testing.T) string {
	t.Helper()
	p := map[string]any{
		"v": 1, "t": 20,
		"pk":  "CTwMqDZnPd/QTLSq8aTeSD3a+j2DQxKcGfhhIYJQ65Y=",
		"sig": "AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA==",
		"ts":  "2026-04-06T23:25:06Z",
		"s": map[string]any{
			"id":  "01HJHB01T8FYZ7YTR9P5N62K5B",
			"d":   map[string]any{"name": "test", "description": "no external file was hashed"},
			"mh":  "ccddccddccddccddccddccddccddccddccddccddccddccddccddccddccddccdd",
			"kid": "4ceefa4a",
		},
		"b": map[string]any{
			"id":  "019cf813-99b8-730a-84f1-5a711a9c355e",
			"ph":  "1111111111111111111111111111111111111111111111111111111111111111",
			"mr":  "2222222222222222222222222222222222222222222222222222222222222222",
			"mh":  "4444444444444444444444444444444444444444444444444444444444444444",
			"kid": "4ceefa4a",
		},
		"ip": "AA",
		"cx": []any{map[string]any{
			"t": 40, "net": "testnet",
			"tx":   "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
			"memo": "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
			"l":    1, "ep": "AA",
		}},
	}
	data, err := json.Marshal(p)
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "claims-only.json")
	if err := os.WriteFile(path, data, 0644); err != nil {
		t.Fatal(err)
	}
	return path
}

// TestRunRemote_ExpectedHashOnClaimsOnlyItem_SkipsNotPasses pins the
// remote half of E.7's claims-only branch, which no test reached:
// promoting that skip to a pass publishes "Provided hash matches
// claims.hash" for a bundle that carries no claims.hash at all, which is
// the E.22 defect in its purest form — a positive assertion about a
// comparison whose second operand does not exist.
//
// The local pipeline's equivalent is
// TestHashComparison_ClaimsOnlyItem_SkipsAndKeepsTheNote; remote mode
// re-implements the branch, so it needs its own guard.
func TestRunRemote_ExpectedHashOnClaimsOnlyItem_SkipsNotPasses(t *testing.T) {
	report, err := remoteServer(t, remoteAllPassBody, makeClaimsOnlyItemProofFile(t),
		RemoteOptions{ExpectedHash: "deadbeef00000000000000000000000000000000000000000000000000000000"})
	if err != nil {
		t.Fatalf("RunRemote: %v", err)
	}

	var row *Step
	for i := range report.Steps {
		if report.Steps[i].Group == groupHashComparison {
			if row != nil {
				t.Fatalf("expected exactly one Hash Comparison row: %+v", report.Steps)
			}
			row = &report.Steps[i]
		}
	}
	if row == nil {
		t.Fatalf("no Hash Comparison row at all: %+v", report.Steps)
	}
	if row.Status != StatusSkip {
		t.Errorf("status: got %v, want skip — %s", row.Status, row.Message)
	}
	if !contains(row.Message, "carries no s.d.hash") {
		t.Errorf("message does not name the reason: %q", row.Message)
	}
	// E.22: the caller supplied a hash, and that must stay readable
	// separately from whether it matched.
	out := BuildJSONOutput(report)
	if !out.HashComparison.Supplied {
		t.Error("hash_comparison.supplied: got false, want true — a hash WAS supplied")
	}
	if out.HashComparison.Matched {
		t.Error("hash_comparison.matched: got true, but nothing was compared")
	}
}
