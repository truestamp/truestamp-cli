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
)

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
		APIKey: "test-key",
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
		APIKey: "test-key",
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
		APIKey: "test-key",
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
		APIKey: "bad-key",
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
		APIKey: "test-key",
	})
	if err == nil {
		t.Fatal("expected error for 400")
	}
}

func TestRunRemote_MissingFile(t *testing.T) {
	_, err := RunRemote("/nonexistent/proof.json", RemoteOptions{
		APIURL: "http://localhost",
		APIKey: "test-key",
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
		APIKey: "test-key",
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
	if err := json.Unmarshal([]byte(`"future_status"`), &s); err != nil {
		t.Fatalf("Unmarshal unknown: %v", err)
	}
	if s != StatusInfo {
		t.Errorf("unknown status should map to StatusInfo, got %d", s)
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
		APIKey: "test-key",
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
		APIKey: "test-key",
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

	_, err = RunRemote(cborPath, RemoteOptions{APIURL: server.URL, APIKey: "k"})
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
	_, err := RunRemote(makeProofFile(t), RemoteOptions{APIURL: server.URL, APIKey: "k"})
	if err == nil {
		t.Fatal("expected parse error for non-JSON response")
	}
}

func TestRunRemote_EmptyResultField(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"errors":[]}`))
	}))
	defer server.Close()
	_, err := RunRemote(makeProofFile(t), RemoteOptions{APIURL: server.URL, APIKey: "k"})
	if err == nil || !contains(err.Error(), "result") {
		t.Errorf("expected result-missing error, got %v", err)
	}
}

func TestRunRemote_UnreachableHost(t *testing.T) {
	_, err := RunRemote(makeProofFile(t), RemoteOptions{
		APIURL: "http://127.0.0.1:1",
		APIKey: "k",
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
		APIKey:       "k",
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
