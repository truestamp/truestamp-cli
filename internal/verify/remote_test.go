package verify

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

// makeProofFile writes a minimal valid v1 compact-format JSON proof to a temp file.
func makeProofFile(t *testing.T) string {
	t.Helper()
	p := map[string]any{
		"v":   1,
		"pk":  "CTwMqDZnPd/QTLSq8aTeSD3a+j2DQxKcGfhhIYJQ65Y=",
		"sig": "AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA",
		"ts":  "2026-04-06T23:25:06Z",
		"s": map[string]any{
			"src": "item",
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
		"cx": []any{},
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

// makeEntropyProofFile writes a minimal valid v1 compact entropy proof to a temp file.
func makeEntropyProofFile(t *testing.T) string {
	t.Helper()
	p := map[string]any{
		"v":   1,
		"pk":  "CTwMqDZnPd/QTLSq8aTeSD3a+j2DQxKcGfhhIYJQ65Y=",
		"sig": "AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA",
		"ts":  "2026-04-06T23:25:06Z",
		"s": map[string]any{
			"src": "nist_beacon",
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
		"cx": []any{},
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
	if report.SubjectType != "entropy" {
		t.Errorf("SubjectType: got %q, want entropy", report.SubjectType)
	}
	if report.SubjectID != "019d2ae3-865c-7651-9923-b14c55bc8e33" {
		t.Errorf("SubjectID: got %q", report.SubjectID)
	}
	if report.EntropySubject.Source != "NIST Beacon" {
		t.Errorf("EntropySubject.Source: got %q, want NIST Beacon", report.EntropySubject.Source)
	}
	if report.EntropySubject.RawSource != "nist_beacon" {
		t.Errorf("EntropySubject.RawSource: got %q, want nist_beacon", report.EntropySubject.RawSource)
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
