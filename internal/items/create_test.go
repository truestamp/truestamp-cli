// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: MIT

package items

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/truestamp/truestamp-cli/internal/auth"
)

// TestMain installs an api-key Authorizer for the whole package so
// CreateItem/CreateItemCtx stamp the Authorization header. The key
// matches the "Bearer key" header TestCreateItem_Success asserts.
func TestMain(m *testing.M) {
	auth.SetDefault(auth.APIKeyAuthorizer("key"))
	code := m.Run()
	auth.SetDefault(nil)
	os.Exit(code)
}

// --- parseResponse ---------------------------------------------------

func TestParseResponse_Full(t *testing.T) {
	body := []byte(`{"data":{"id":"01TESTITEM","type":"item","attributes":{
        "state":"new","claims_hash":"ch","item_hash":"ih",
        "visibility":"private","team_id":"team-1","display_name":"doc.pdf",
        "claims":{"hash":"aaaa","hash_type":"sha256","name":"doc.pdf"},
        "tags":["finance","q1",123]
    }}}`)
	r, err := parseResponse(body)
	if err != nil {
		t.Fatalf("parseResponse: %v", err)
	}
	if r.ID != "01TESTITEM" {
		t.Errorf("ID: got %q", r.ID)
	}
	if r.State != "new" {
		t.Errorf("State: got %q", r.State)
	}
	if r.ClaimsHash != "ch" {
		t.Errorf("ClaimsHash: got %q", r.ClaimsHash)
	}
	if r.ItemHash != "ih" {
		t.Errorf("ItemHash: got %q", r.ItemHash)
	}
	if r.Name != "doc.pdf" {
		t.Errorf("Name: got %q", r.Name)
	}
	if r.Hash != "aaaa" || r.HashType != "sha256" {
		t.Errorf("Hash/HashType: got %q/%q", r.Hash, r.HashType)
	}
	if len(r.Tags) != 2 { // non-string 123 is dropped
		t.Errorf("Tags len: got %d, want 2", len(r.Tags))
	}
}

func TestParseResponse_Malformed(t *testing.T) {
	if _, err := parseResponse([]byte("not json")); err == nil {
		t.Error("malformed JSON should error")
	}
}

func TestParseResponse_NoClaims(t *testing.T) {
	body := []byte(`{"data":{"id":"x","attributes":{"state":"new"}}}`)
	r, err := parseResponse(body)
	if err != nil {
		t.Fatal(err)
	}
	if r.Hash != "" || r.HashType != "" {
		t.Errorf("no claims should yield empty hash fields")
	}
}

func TestParseResponse_ClaimsNotMap(t *testing.T) {
	body := []byte(`{"data":{"id":"x","attributes":{"claims":"not-a-map"}}}`)
	r, err := parseResponse(body)
	if err != nil {
		t.Fatal(err)
	}
	if r.Hash != "" {
		t.Errorf("non-map claims should not populate hash")
	}
}

func TestParseResponse_TagsNotSlice(t *testing.T) {
	body := []byte(`{"data":{"id":"x","attributes":{"tags":"not-a-slice"}}}`)
	r, err := parseResponse(body)
	if err != nil {
		t.Fatal(err)
	}
	if len(r.Tags) != 0 {
		t.Errorf("non-slice tags should be ignored")
	}
}

// --- getString -------------------------------------------------------

func TestGetString(t *testing.T) {
	m := map[string]any{"a": "text", "b": 42}
	if got := getString(m, "a"); got != "text" {
		t.Errorf("present: got %q", got)
	}
	if got := getString(m, "b"); got != "" {
		t.Errorf("non-string: got %q", got)
	}
	if got := getString(m, "missing"); got != "" {
		t.Errorf("missing: got %q", got)
	}
}

// --- parseError ------------------------------------------------------

func TestParseError_WithDetail(t *testing.T) {
	err := parseError(400, []byte(`{"errors":[{"detail":"bad claims"}]}`))
	if err == nil || !strings.Contains(err.Error(), "bad claims") {
		t.Errorf("expected detail in error: %v", err)
	}
}

func TestParseError_TitleOnly(t *testing.T) {
	err := parseError(404, []byte(`{"errors":[{"title":"Not Found"}]}`))
	if err == nil || !strings.Contains(err.Error(), "Not Found") {
		t.Errorf("expected title in error: %v", err)
	}
}

func TestParseError_HTMLBody(t *testing.T) {
	err := parseError(502, []byte("<html>oops</html>"))
	if err == nil || !strings.Contains(err.Error(), "HTML") {
		t.Errorf("expected HTML mention: %v", err)
	}
}

func TestParseError_UnparseableBody(t *testing.T) {
	err := parseError(500, []byte("server stack trace"))
	if err == nil {
		t.Fatal("expected error")
	}
}

// --- CreateItem / CreateItemCtx (httptest) ---------------------------

func TestCreateItem_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer key" {
			t.Errorf("auth header: got %q", r.Header.Get("Authorization"))
		}
		if r.Header.Get("tenant") != "team-1" {
			t.Errorf("tenant header: got %q", r.Header.Get("tenant"))
		}
		// Echo back a valid envelope.
		body, _ := io.ReadAll(r.Body)
		_ = body
		w.WriteHeader(201)
		_, _ = w.Write([]byte(`{"data":{"id":"01NEW","attributes":{"state":"new","claims":{"hash":"aa","hash_type":"sha256"}}}}`))
	}))
	defer srv.Close()

	r, err := CreateItem(srv.URL, "team-1",
		map[string]any{"hash": "aa", "hash_type": "sha256", "name": "x"},
		"public", []string{"foo"})
	if err != nil {
		t.Fatal(err)
	}
	if r.ID != "01NEW" {
		t.Errorf("ID: got %q", r.ID)
	}
}

func TestCreateItem_NoTenantWhenEmpty(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("tenant") != "" {
			t.Errorf("tenant header should be absent, got %q", r.Header.Get("tenant"))
		}
		w.WriteHeader(201)
		_, _ = w.Write([]byte(`{"data":{"id":"x"}}`))
	}))
	defer srv.Close()
	_, err := CreateItemCtx(context.Background(), srv.URL, "",
		map[string]any{}, "", nil)
	if err != nil {
		t.Fatal(err)
	}
}

func TestCreateItem_APIError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(400)
		_, _ = w.Write([]byte(`{"errors":[{"detail":"bad request"}]}`))
	}))
	defer srv.Close()
	_, err := CreateItem(srv.URL, "", map[string]any{}, "", nil)
	if err == nil || !strings.Contains(err.Error(), "bad request") {
		t.Errorf("expected API error, got %v", err)
	}
}

func TestCreateItem_UnreachableHost(t *testing.T) {
	_, err := CreateItem("http://127.0.0.1:1", "", map[string]any{}, "", nil)
	if err == nil {
		t.Error("expected connection error")
	}
}

func TestCreateItem_MalformedResponse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("not json"))
	}))
	defer srv.Close()
	_, err := CreateItem(srv.URL, "", map[string]any{}, "", nil)
	if err == nil {
		t.Error("expected parse error")
	}
}

func TestCreateItem_TagsEchoedInRequest(t *testing.T) {
	var received map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewDecoder(r.Body).Decode(&received)
		_, _ = w.Write([]byte(`{"data":{"id":"x"}}`))
	}))
	defer srv.Close()
	_, err := CreateItem(srv.URL, "", map[string]any{"x": 1}, "private",
		[]string{"a", "b"})
	if err != nil {
		t.Fatal(err)
	}
	attrs, _ := received["data"].(map[string]any)
	a, _ := attrs["attributes"].(map[string]any)
	tags, ok := a["tags"].([]any)
	if !ok || len(tags) != 2 {
		t.Errorf("tags not echoed: %+v", a)
	}
}

// TestCreateItemCtx_PreservesIntegerLiteralOnTheWire is the wire-level
// regression for the claims-corruption bug.
//
// The defect lived in cmd/create's decode, which used json.Unmarshal into a
// map[string]any with no UseNumber: every JSON number became a float64, so
// 18446744073709551615 was rewritten to 18446744073709552000 and
// 9007199254740993 to 9007199254740992 before any code could inspect them. A
// user timestamping a 64-bit id got a proof committing to a number they never
// submitted.
//
// The fix is json.Number, and this test asserts the second half of that
// contract: that a json.Number in the claims map survives THIS package's
// json.Marshal and lands in the request body as the exact bytes the user
// typed. It asserts on the raw body read off the connection rather than on a
// re-decoded map, because the corruption happened during decode — a test that
// decoded the body again in-process would round the value a second time and
// happily agree with itself.
//
// The values here are deliberately outside the producer's portable range, so
// `truestamp create` refuses them at the CLI layer (see
// TestCLI_Create_UnsafeIntegerRejectedBeforeNetwork). That guard is only
// correct because of the preservation proved here: it can name the offending
// value in its error message only if the literal was never rounded on the way
// in.
func TestCreateItemCtx_PreservesIntegerLiteralOnTheWire(t *testing.T) {
	var captured []byte
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		captured, _ = io.ReadAll(r.Body)
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"data":{"id":"01TESTITEM","type":"item","attributes":{}}}`))
	}))
	defer srv.Close()

	// Decode exactly the way cmd/create does.
	dec := json.NewDecoder(strings.NewReader(
		`{"name":"Big","id":18446744073709551615,"metadata":{"rows":[{"n":9007199254740993}]}}`))
	dec.UseNumber()
	var claims map[string]any
	if err := dec.Decode(&claims); err != nil {
		t.Fatalf("decoding claims: %v", err)
	}

	if _, err := CreateItemCtx(context.Background(), srv.URL, "", claims, "private", nil); err != nil {
		t.Fatalf("CreateItemCtx: %v", err)
	}

	body := string(captured)
	for _, want := range []string{`"id":18446744073709551615`, `"n":9007199254740993`} {
		if !strings.Contains(body, want) {
			t.Errorf("request body missing %s\nbody: %s", want, body)
		}
	}
	// The exact corruptions the float64 path produced.
	for _, corrupted := range []string{"18446744073709552000", "9007199254740992", "1.8446744073709552e+19"} {
		if strings.Contains(body, corrupted) {
			t.Errorf("request body carries rounded value %s\nbody: %s", corrupted, body)
		}
	}
}
