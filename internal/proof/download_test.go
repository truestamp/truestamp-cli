package proof

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestDownload_ValidProof(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(validProofJSON))
	}))
	defer server.Close()

	data, err := Download(server.URL + "/proof.json")
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	if len(data) == 0 {
		t.Error("expected non-empty data")
	}
}

func TestDownload_InvalidScheme(t *testing.T) {
	_, err := Download("ftp://example.com/proof.json")
	if err == nil {
		t.Error("expected error for ftp scheme")
	}
}

func TestDownload_MissingHost(t *testing.T) {
	_, err := Download("http://")
	if err == nil {
		t.Error("expected error for missing host")
	}
}

func TestDownload_NotJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("<html>not json</html>"))
	}))
	defer server.Close()

	_, err := Download(server.URL + "/page.html")
	if err == nil {
		t.Error("expected error for non-JSON response")
	}
}

func TestDownload_NotProofJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"name": "not a proof"}`))
	}))
	defer server.Close()

	_, err := Download(server.URL + "/data.json")
	if err == nil {
		t.Error("expected error for JSON without proof fields")
	}
}

func TestDownload_HTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "not found", http.StatusNotFound)
	}))
	defer server.Close()

	_, err := Download(server.URL + "/missing.json")
	if err == nil {
		t.Error("expected error for 404 response")
	}
}
