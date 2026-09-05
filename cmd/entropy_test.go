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
		case q.Get("page[after]") == "CURSOR1":
			// Page two forward: it can go back.
			_, _ = w.Write([]byte(`{"data":[` + nist + `],"links":{"prev":"http://` + r.Host +
				`/api/json/entropy_observations?page%5Bbefore%5D=CURSORB&page%5Blimit%5D=2&sort=-id"}}`))
			return
		case q.Get("page[before]") == "CURSORB":
			// Walking back from page two: page one, which can go back once more.
			_, _ = w.Write([]byte(`{"data":[` + stellar + `],"links":{"prev":"http://` + r.Host +
				`/api/json/entropy_observations?page%5Bbefore%5D=CURSORC&page%5Blimit%5D=2&sort=-id"}}`))
			return
		case q.Get("page[before]") == "CURSORC":
			rows = []string{nist}
		default:
			rows = []string{stellar, nist}
			// The first page of the unfiltered listing continues: a next
			// link the CLI must lift the cursor out of.
			_, _ = w.Write([]byte(`{"data":[` + strings.Join(rows, ",") + `],"links":{"next":"http://` + r.Host +
				`/api/json/entropy_observations?page%5Bafter%5D=CURSOR1&page%5Blimit%5D=2&sort=-id"},"meta":{"page":{"total":593419}}}`))
			return
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
	var got struct {
		Observations []map[string]any `json:"observations"`
		NextCursor   string           `json:"next_cursor"`
	}
	if err := json.Unmarshal([]byte(stdout), &got); err != nil || len(got.Observations) != 2 {
		t.Fatalf("want two observations under the noun, got %v\n%s", err, stdout)
	}
	if got.NextCursor != "CURSOR1" {
		t.Errorf("next_cursor should be lifted from links.next, got %q", got.NextCursor)
	}
	if srv.query().Get("page[limit]") != "2" || srv.query().Get("sort") != "-id" {
		t.Errorf("asked %v", srv.query())
	}
}

// TestCLI_Entropy_List_MaxFollowsCursors pins R14: --max follows the
// cursor until the cap or the end, asks each page for no more rows than
// still wanted, and reports where it stopped.
func TestCLI_Entropy_List_MaxFollowsCursors(t *testing.T) {
	srv := startEntropyServer(t)
	stdout, stderr, exit := runCLI(t, "--base-url", srv.URL, "--api-key", "test-key",
		"entropy", "list", "--limit", "2", "--max", "3", "--json")
	if exit != 0 {
		t.Fatalf("exit=%d\n%s", exit, stderr)
	}
	var got struct {
		Observations []map[string]any `json:"observations"`
		NextCursor   string           `json:"next_cursor"`
	}
	if err := json.Unmarshal([]byte(stdout), &got); err != nil {
		t.Fatalf("not JSON: %v\n%s", err, stdout)
	}
	if len(got.Observations) != 3 || got.NextCursor != "" {
		t.Errorf("want 3 rows and no cursor left, got %d rows, cursor %q", len(got.Observations), got.NextCursor)
	}
	if srv.requests() != 2 {
		t.Errorf("two pages should cost two requests, saw %d", srv.requests())
	}
	q := srv.query()
	if q.Get("page[after]") != "CURSOR1" || q.Get("page[limit]") != "1" {
		t.Errorf("the second request should continue from the cursor and ask for only the one row still wanted, asked %v", q)
	}
	_, stderr, exit = runCLI(t, "--base-url", srv.URL, "entropy", "list", "--max", "0")
	if exit == 0 || !strings.Contains(stderr, "--max must be at least 1") {
		t.Errorf("--max 0 must be refused locally: exit=%d %q", exit, stderr)
	}
}

// TestCLI_Entropy_List_CountAndHint: --count carries the server's total
// into both renderings, and a page that continues says how.
func TestCLI_Entropy_List_CountAndHint(t *testing.T) {
	srv := startEntropyServer(t)
	stdout, _, exit := runCLI(t, "--base-url", srv.URL, "--api-key", "test-key", "entropy", "list", "--count")
	if exit != 0 {
		t.Fatalf("exit=%d", exit)
	}
	if !strings.Contains(stdout, "(2 shown, 593,419 total)") || !strings.Contains(stdout, "More: --after CURSOR1") {
		t.Errorf("text listing should show the total and the continuation, got:\n%s", stdout)
	}
	if srv.query().Get("page[count]") != "true" {
		t.Errorf("--count must ask the server for the total, asked %v", srv.query())
	}
	stdout, _, _ = runCLI(t, "--base-url", srv.URL, "--api-key", "test-key", "entropy", "list", "--count", "--json")
	if !strings.Contains(stdout, `"total": 593419`) {
		t.Errorf("--json should carry total under --count, got:\n%s", stdout)
	}
}

func TestCLI_Entropy_List_Text(t *testing.T) {
	srv := startEntropyServer(t)
	stdout, _, exit := runCLI(t, "--base-url", srv.URL, "--api-key", "test-key", "entropy", "list")
	if exit != 0 {
		t.Fatalf("exit=%d", exit)
	}
	for _, want := range []string{"Entropy Observations (2)", "PUBLISHED", "SOURCE", "stellar", "nist", testEntropyStellarID} {
		if !strings.Contains(stdout, want) {
			t.Errorf("list should show %q, got:\n%s", want, stdout)
		}
	}
	if strings.Contains(stdout, "entropy_stellar") {
		t.Errorf("the SOURCE column should drop the entropy_ prefix, got:\n%s", stdout)
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
	for _, want := range []string{"Entropy Observation", testEntropyStellarID, "Source", "stellar", testEntropyStellarHash,
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

// TestCLI_Entropy_List_Backward pins the other direction of R14: --before
// continues from a prev cursor, --max follows prev cursors, rows come
// back in the listing's order (pages fetched backward are prepended), and
// the cursors handed back are the ones that continue from the rows shown.
func TestCLI_Entropy_List_Backward(t *testing.T) {
	srv := startEntropyServer(t)
	stdout, stderr, exit := runCLI(t, "--base-url", srv.URL, "--api-key", "test-key",
		"entropy", "list", "--before", "CURSORB", "--max", "5", "--json")
	if exit != 0 {
		t.Fatalf("exit=%d\n%s", exit, stderr)
	}
	var got struct {
		Observations []struct {
			ID string `json:"id"`
		} `json:"observations"`
		NextCursor string `json:"next_cursor"`
		PrevCursor string `json:"prev_cursor"`
	}
	if err := json.Unmarshal([]byte(stdout), &got); err != nil {
		t.Fatalf("not JSON: %v\n%s", err, stdout)
	}
	ids := []string{}
	for _, o := range got.Observations {
		ids = append(ids, o.ID)
	}
	if len(ids) != 2 || ids[0] != testEntropyNISTID || ids[1] != testEntropyStellarID {
		t.Errorf("backward pages must be prepended so rows stay in listing order, got %v", ids)
	}
	if got.PrevCursor != "" {
		t.Errorf("the walk reached the start, prev_cursor should be empty, got %q", got.PrevCursor)
	}
	if srv.requests() != 2 || srv.query().Get("page[before]") != "CURSORC" {
		t.Errorf("want two requests, the second continuing from CURSORC; saw %d, last %v", srv.requests(), srv.query())
	}
	stdout, _, _ = runCLI(t, "--base-url", srv.URL, "--api-key", "test-key", "entropy", "list", "--after", "CURSOR1")
	if !strings.Contains(stdout, "Back: --before CURSORB") {
		t.Errorf("a page that can go back should say how, got:\n%s", stdout)
	}
	_, stderr, exit = runCLI(t, "--base-url", srv.URL, "entropy", "list", "--after", "A", "--before", "B")
	if exit == 0 || !strings.Contains(stderr, "--after and --before are mutually exclusive") {
		t.Errorf("both directions at once must be refused locally: exit=%d %q", exit, stderr)
	}
}

// TestCLI_Entropy_List_OldestFirst pins the sort flip: the walk starts at
// the beginning and the server is asked for ascending order.
func TestCLI_Entropy_List_OldestFirst(t *testing.T) {
	srv := startEntropyServer(t)
	_, _, exit := runCLI(t, "--base-url", srv.URL, "--api-key", "test-key", "entropy", "list", "--oldest-first", "--json")
	if exit != 0 {
		t.Fatalf("exit=%d", exit)
	}
	if srv.query().Get("sort") != "id" {
		t.Errorf("--oldest-first must ask for ascending order, asked %v", srv.query())
	}
}
