// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: MIT

package cmd

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"testing"
)

const (
	testEntropyStellarID   = "01a07342-b988-7b3a-a707-8b1653bf175c"
	testEntropyNISTID      = "01a07335-b8fe-7ef2-856c-c9b4eca99850"
	testEntropyStellarHash = "4142459a2859a57a5e5d6540579b977215d6329110adc0a890fdce6f05054f2d"
	testEntropyNISTHash    = "5c5bc094d727186afce57af677ca17738f7ae1339be408a58c9b4ed02661eb5e"
)

func testObservation(id, source, hash string) string {
	return `{"type":"entropy_observation","id":"` + id + `","attributes":{"source":"` + source + `",
	  "state":"committed","entropy_hash":"` + hash + `","observation_hash":"oh","metadata_hash":"mh",
	  "signing_key_id":"96b1cd2f","signature":"sig","capture_method":"http_sse",
	  "source_published_at":"2026-09-05T20:29:07.000000Z","inserted_at":"2026-09-05T20:29:07.848738Z",
	  "block_id":null,"metadata":{},"entropy":{"closed_at":"2026-09-05T20:29:07Z","sequence":4523312,"pulse":{"pulseIndex":1928582}}}}`
}

// entropyServer is a mock of the two entropy routes that records what the
// CLI asked for, so tests assert on the request as well as the rendering.
type entropyServer struct {
	*httptest.Server
	mu        sync.Mutex
	lastQuery url.Values
	hits      int
}

func (s *entropyServer) query() url.Values {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.lastQuery
}

func (s *entropyServer) requests() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.hits
}

func startEntropyServer(t *testing.T) *entropyServer {
	t.Helper()
	s := &entropyServer{}
	stellar := testObservation(testEntropyStellarID, "entropy_stellar", testEntropyStellarHash)
	nist := testObservation(testEntropyNISTID, "entropy_nist", testEntropyNISTHash)
	mux := http.NewServeMux()
	mux.HandleFunc("/api/json/entropy_observations/", func(w http.ResponseWriter, r *http.Request) {
		requireBearer(t, r)
		w.Header().Set("Content-Type", "application/vnd.api+json")
		switch strings.TrimPrefix(r.URL.Path, "/api/json/entropy_observations/") {
		case testEntropyStellarID:
			_, _ = w.Write([]byte(`{"data":` + stellar + `}`))
		case testEntropyNISTID:
			_, _ = w.Write([]byte(`{"data":` + nist + `}`))
		default:
			w.WriteHeader(http.StatusNotFound)
			_, _ = w.Write([]byte(`{"errors":[{"status":"404","title":"Not Found","detail":"no such observation"}]}`))
		}
	})
	mux.HandleFunc("/api/json/entropy_observations", func(w http.ResponseWriter, r *http.Request) {
		requireBearer(t, r)
		w.Header().Set("Content-Type", "application/vnd.api+json")
		q := r.URL.Query()
		var rows []string
		switch {
		case q.Get("filter[entropy_hash]") == testEntropyStellarHash:
			rows = []string{stellar}
		case q.Has("filter[entropy_hash]"):
			rows = nil
		case q.Get("filter[source]") == "entropy_nist":
			rows = []string{nist}
		case q.Get("filter[source]") == "entropy_stellar":
			rows = []string{stellar}
		default:
			rows = []string{stellar, nist}
		}
		_, _ = w.Write([]byte(`{"data":[` + strings.Join(rows, ",") + `]}`))
	})
	s.Server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		s.mu.Lock()
		s.lastQuery, s.hits = r.URL.Query(), s.hits+1
		s.mu.Unlock()
		mux.ServeHTTP(w, r)
	}))
	t.Cleanup(s.Close)
	return s
}

func TestCLI_Entropy_BareGroupPrintsHelp(t *testing.T) {
	stdout, _, exit := runCLI(t, "entropy")
	if exit != 0 || !strings.Contains(stdout, "Available Commands:") {
		t.Errorf("a bare group must print help and exit 0, got exit=%d\n%s", exit, stdout)
	}
}

func TestCLI_Entropy_Latest_JSON(t *testing.T) {
	srv := startEntropyServer(t)
	stdout, stderr, exit := runCLI(t, "--base-url", srv.URL, "--api-key", "test-key", "entropy", "latest", "--json")
	if exit != 0 {
		t.Fatalf("exit=%d\n%s", exit, stderr)
	}
	var got struct {
		ID      string         `json:"id"`
		Source  string         `json:"source"`
		Entropy map[string]any `json:"entropy"`
	}
	if err := json.Unmarshal([]byte(stdout), &got); err != nil {
		t.Fatalf("invalid JSON: %v\n%s", err, stdout)
	}
	if got.ID != testEntropyStellarID || got.Source != "entropy_stellar" {
		t.Errorf("wrong observation: %+v", got)
	}
	if !strings.Contains(stdout, `"sequence": 4523312`) {
		t.Errorf("the ledger sequence must round-trip as an integer, got:\n%s", stdout)
	}
	q := srv.query()
	if q.Get("sort") != "-id" || q.Get("page[limit]") != "1" || q.Has("filter[source]") {
		t.Errorf("latest must ask for the newest row across sources, asked %v", q)
	}
}

func TestCLI_Entropy_Latest_Source(t *testing.T) {
	srv := startEntropyServer(t)
	stdout, _, exit := runCLI(t, "--base-url", srv.URL, "--api-key", "test-key",
		"entropy", "latest", "--source", "entropy_nist", "--json")
	if exit != 0 || !strings.Contains(stdout, testEntropyNISTID) {
		t.Fatalf("exit=%d\n%s", exit, stdout)
	}
	if srv.query().Get("filter[source]") != "entropy_nist" {
		t.Errorf("--source must become filter[source], asked %v", srv.query())
	}
}

func TestCLI_Entropy_List_JSON(t *testing.T) {
	srv := startEntropyServer(t)
	stdout, _, exit := runCLI(t, "--base-url", srv.URL, "--api-key", "test-key", "entropy", "list", "--limit", "2", "--json")
	if exit != 0 {
		t.Fatalf("exit=%d", exit)
	}
	var list []map[string]any
	if err := json.Unmarshal([]byte(stdout), &list); err != nil || len(list) != 2 {
		t.Fatalf("want a two-element array, got %v\n%s", err, stdout)
	}
	if srv.query().Get("page[limit]") != "2" || srv.query().Get("sort") != "-id" {
		t.Errorf("asked %v", srv.query())
	}
}

func TestCLI_Entropy_List_Text(t *testing.T) {
	srv := startEntropyServer(t)
	stdout, _, exit := runCLI(t, "--base-url", srv.URL, "--api-key", "test-key", "entropy", "list")
	if exit != 0 {
		t.Fatalf("exit=%d", exit)
	}
	for _, want := range []string{"Entropy Observations (2)", "PUBLISHED", "SOURCE", "entropy_stellar", "entropy_nist", testEntropyStellarID} {
		if !strings.Contains(stdout, want) {
			t.Errorf("list should show %q, got:\n%s", want, stdout)
		}
	}
}

// TestCLI_Entropy_BadSource_RefusedLocally pins two things: the vocabulary
// is the wire names, and the refusal happens before the credential gate
// and before any request.
func TestCLI_Entropy_BadSource_RefusedLocally(t *testing.T) {
	srv := startEntropyServer(t)
	_, stderr, exit := runCLI(t, "--base-url", srv.URL, "entropy", "list", "--source", "nist")
	if exit == 0 || !strings.Contains(stderr, "--source must be one of") || !strings.Contains(stderr, "entropy_nist") {
		t.Errorf("exit=%d stderr=%q", exit, stderr)
	}
	if srv.requests() != 0 {
		t.Errorf("a bad --source must not cost a request, server saw %d", srv.requests())
	}
}

func TestCLI_Entropy_Get_Card(t *testing.T) {
	srv := startEntropyServer(t)
	stdout, _, exit := runCLI(t, "--base-url", srv.URL, "--api-key", "test-key", "entropy", "get", testEntropyStellarID)
	if exit != 0 {
		t.Fatalf("exit=%d\n%s", exit, stdout)
	}
	for _, want := range []string{"Entropy Observation", testEntropyStellarID, "entropy_stellar", testEntropyStellarHash,
		"sequence", "4523312", "pulse.pulseIndex", "1928582", "/entropy/" + testEntropyStellarID, "/verify/entropy_stellar/" + testEntropyStellarID} {
		if !strings.Contains(stdout, want) {
			t.Errorf("card should show %q, got:\n%s", want, stdout)
		}
	}
}

func TestCLI_Entropy_Get_ByHash(t *testing.T) {
	srv := startEntropyServer(t)
	stdout, _, exit := runCLI(t, "--base-url", srv.URL, "--api-key", "test-key", "entropy", "get", testEntropyStellarHash, "--json")
	if exit != 0 || !strings.Contains(stdout, testEntropyStellarID) {
		t.Fatalf("exit=%d\n%s", exit, stdout)
	}
	if srv.query().Get("filter[entropy_hash]") != testEntropyStellarHash {
		t.Errorf("a hash must become filter[entropy_hash], asked %v", srv.query())
	}
	_, stderr, exit := runCLI(t, "--base-url", srv.URL, "--api-key", "test-key", "entropy", "get", testEntropyNISTHash)
	if exit == 0 || !strings.Contains(stderr, "entropy observation not found") {
		t.Errorf("an unknown hash is not found, got exit=%d %q", exit, stderr)
	}
}

func TestCLI_Entropy_Get_BadShapes(t *testing.T) {
	srv := startEntropyServer(t)
	_, stderr, exit := runCLI(t, "--base-url", srv.URL, "--api-key", "test-key", "entropy", "get", "nope")
	if exit == 0 || !strings.Contains(stderr, "neither a UUIDv7 id nor a 64-hex-char entropy hash") {
		t.Errorf("exit=%d stderr=%q", exit, stderr)
	}
	_, stderr, exit = runCLI(t, "--base-url", srv.URL, "--api-key", "test-key", "entropy", "get", "019db702-b08c-73dc-a7cd-2c5e011f1dad")
	if exit == 0 || !strings.Contains(stderr, "entropy observation not found") {
		t.Errorf("unknown id: exit=%d stderr=%q", exit, stderr)
	}
	if srv.requests() != 1 {
		t.Errorf("only the well-formed unknown id should reach the server, saw %d requests", srv.requests())
	}
}

func TestCLI_Entropy_MissingAPIKey_NotAuthenticated(t *testing.T) {
	_, stderr, exit := runCLI(t, "--base-url", "http://127.0.0.1:0", "entropy", "latest")
	if exit == 0 || !strings.Contains(stderr, "Not authenticated") {
		t.Errorf("exit=%d stderr=%q", exit, stderr)
	}
}

func TestCLI_Entropy_MutualExclusion_SilentJSON(t *testing.T) {
	_, stderr, exit := runCLI(t, "--api-key", "test-key", "entropy", "latest", "--silent", "--json")
	if exit == 0 || !strings.Contains(stderr, "mutually exclusive") {
		t.Errorf("exit=%d stderr=%q", exit, stderr)
	}
}
