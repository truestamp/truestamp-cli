// Copyright (c) 2021-2026 Truestamp, Inc.
// SPDX-License-Identifier: MIT

package items

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

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

	r, err := CreateItem(srv.URL, "key", "team-1",
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
	_, err := CreateItemCtx(context.Background(), srv.URL, "key", "",
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
	_, err := CreateItem(srv.URL, "key", "", map[string]any{}, "", nil)
	if err == nil || !strings.Contains(err.Error(), "bad request") {
		t.Errorf("expected API error, got %v", err)
	}
}

func TestCreateItem_UnreachableHost(t *testing.T) {
	_, err := CreateItem("http://127.0.0.1:1", "key", "", map[string]any{}, "", nil)
	if err == nil {
		t.Error("expected connection error")
	}
}

func TestCreateItem_MalformedResponse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("not json"))
	}))
	defer srv.Close()
	_, err := CreateItem(srv.URL, "key", "", map[string]any{}, "", nil)
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
	_, err := CreateItem(srv.URL, "key", "", map[string]any{"x": 1}, "private",
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
