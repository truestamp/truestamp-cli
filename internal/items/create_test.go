// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: Apache-2.0

package items

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"reflect"
	"strings"
	"testing"

	"github.com/truestamp/truestamp-cli/internal/auth"
	"github.com/truestamp/truestamp-cli/internal/jsonapi"
)

// TestMain installs an api-key Authorizer for the whole package so
// CreateItemCtx stamps the Authorization header. The key
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
        "state":"new","claims_hash":"ch","claims_jcs":"{\"name\":\"doc.pdf\"}","item_hash":"ih",
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
	if r.ClaimsJCS != `{"name":"doc.pdf"}` {
		t.Errorf("ClaimsJCS: got %q", r.ClaimsJCS)
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

// --- CreateItemCtx (httptest) -----------------------------------------

func TestCreateItem_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer key" {
			t.Errorf("auth header: got %q", r.Header.Get("Authorization"))
		}
		if r.Header.Get("tenant") != "team-1" {
			t.Errorf("tenant header: got %q", r.Header.Get("tenant"))
		}
		w.WriteHeader(201)
		_, _ = w.Write([]byte(`{"data":{"id":"01NEW","attributes":{"state":"new","claims":{"hash":"aa","hash_type":"sha256"},` +
			`"claims_jcs":"{\"hash\":\"aa\",\"hash_type\":\"sha256\",\"name\":\"x\"}","claims_hash":"c1"}}}`))
	}))
	defer srv.Close()

	r, err := CreateItemCtx(context.Background(), srv.URL, "team-1",
		[]byte(`{"hash":"aa","hash_type":"sha256","name":"x"}`), "public", []string{"foo"})
	if err != nil {
		t.Fatal(err)
	}
	if r.ID != "01NEW" || r.ClaimsHash != "c1" || r.ClaimsJCS != `{"hash":"aa","hash_type":"sha256","name":"x"}` {
		t.Errorf("got %+v", r)
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
	if _, err := CreateItemCtx(context.Background(), srv.URL, "", []byte(`{}`), "", nil); err != nil {
		t.Fatal(err)
	}
}

func TestCreateItem_APIError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(400)
		_, _ = w.Write([]byte(`{"errors":[{"detail":"bad request"}]}`))
	}))
	defer srv.Close()
	_, err := CreateItemCtx(context.Background(), srv.URL, "", []byte(`{}`), "", nil)
	if err == nil || err.Error() != "HTTP 400: bad request" {
		t.Errorf("expected API error, got %v", err)
	}
}

func TestCreateItem_UnreachableHost(t *testing.T) {
	_, err := CreateItemCtx(context.Background(), "http://127.0.0.1:1", "", []byte(`{}`), "", nil)
	if err == nil || !strings.Contains(err.Error(), "API request failed") {
		t.Errorf("expected connection error, got %v", err)
	}
}

func TestCreateItem_MalformedResponse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("not json"))
	}))
	defer srv.Close()
	_, err := CreateItemCtx(context.Background(), srv.URL, "", []byte(`{}`), "", nil)
	if err == nil || !strings.HasPrefix(err.Error(), "parsing response:") {
		t.Errorf("expected parse error, got %v", err)
	}
}

// TestCreateItemCtx_SendsClaimsTextVerbatim pins the request body byte for
// byte: the claims text, with its whitespace, its member order, its
// escapes, a leading space in a value and its number spellings, sits
// unchanged as the value of attributes.claims, beside the visibility and
// tags. Mutant killed: the claims spliced in as a json.RawMessage through
// json.Marshal (the text is compacted, and "<" is written \u003c).
func TestCreateItemCtx_SendsClaimsTextVerbatim(t *testing.T) {
	var captured []byte
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		captured, _ = io.ReadAll(r.Body)
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"data":{"id":"01TESTITEM","type":"item","attributes":{}}}`))
	}))
	defer srv.Close()

	claims := "{ \"name\" : \" Big <a>\",\n\t\"metadata\":{\"z\":1.50e300,\"a\":\"\\u00e9\"} }"
	if _, err := CreateItemCtx(context.Background(), srv.URL, "", []byte(claims), "private", []string{"a", "b"}); err != nil {
		t.Fatalf("CreateItemCtx: %v", err)
	}
	want := `{"data":{"type":"item","attributes":{"claims":` + claims + `,"visibility":"private","tags":["a","b"]}}}`
	if string(captured) != want {
		t.Errorf("request body\n got: %s\nwant: %s", captured, want)
	}
}

// TestCreateItemCtx_ClaimsRefusal reads a 422 the way the server writes a
// claims refusal (one error per broken rule, source.pointer under
// /data/attributes/claims) as every reason, in order, with each pointer
// made relative to the claims, and keeps an error about another attribute
// apart. Mutant killed: parseClaimsRefusal keeping only the first error
// (the second reason is lost), and claimsMemberPointer returning the full
// source pointer (/data/attributes/claims/name is reported as the pointer).
func TestCreateItemCtx_ClaimsRefusal(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnprocessableEntity)
		_, _ = w.Write([]byte(`{"errors":[` +
			`{"code":"claims_surrounding_whitespace","title":"Invalid claims","detail":"/name begins with a space (U+0020). Remove it.","source":{"pointer":"/data/attributes/claims/name"}},` +
			`{"code":"claims_content","title":"Invalid claims","detail":"The claims need a description.","source":{"pointer":"/data/attributes/claims"}},` +
			`{"code":"invalid_attribute","detail":"tags is invalid","source":{"pointer":"/data/attributes/tags"}}]}`))
	}))
	defer srv.Close()

	_, err := CreateItemCtx(context.Background(), srv.URL, "", []byte(`{"name":" x"}`), "", nil)
	var refused *ClaimsRefusedError
	if !errors.As(err, &refused) {
		t.Fatalf("want *ClaimsRefusedError, got %T %v", err, err)
	}
	want := []ClaimsReason{
		{Code: "claims_surrounding_whitespace", Message: "/name begins with a space (U+0020). Remove it.", Pointer: "/name"},
		{Code: "claims_content", Message: "The claims need a description.", Pointer: ""},
	}
	if !reflect.DeepEqual(refused.Reasons, want) {
		t.Errorf("reasons\n got: %#v\nwant: %#v", refused.Reasons, want)
	}
	if !reflect.DeepEqual(refused.Others, []string{"invalid_attribute: tags is invalid"}) {
		t.Errorf("others: %#v", refused.Others)
	}
	if refused.Status != http.StatusUnprocessableEntity {
		t.Errorf("status %d", refused.Status)
	}
}

// TestCreateItemCtx_NonClaims422IsAPIError keeps a 422 that names no
// member of the claims, or names a pointer that only begins with the same
// letters, an ordinary API error. Mutant killed: claimsMemberPointer
// matching on the bare prefix (/data/attributes/claimsx is read as a
// member of the claims).
func TestCreateItemCtx_NonClaims422IsAPIError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnprocessableEntity)
		_, _ = w.Write([]byte(`{"errors":[{"code":"invalid","detail":"nope","source":{"pointer":"/data/attributes/claimsx"}}]}`))
	}))
	defer srv.Close()

	_, err := CreateItemCtx(context.Background(), srv.URL, "", []byte(`{}`), "", nil)
	var refused *ClaimsRefusedError
	if errors.As(err, &refused) {
		t.Fatalf("a 422 about another attribute read as a claims refusal: %v", err)
	}
	var apiErr *jsonapi.APIError
	if !errors.As(err, &apiErr) || apiErr.Status != 422 || apiErr.Detail != "nope" {
		t.Errorf("want the 422 as an APIError, got %v", err)
	}
}

// TestCreateItemCtx_ItemCodeUnavailable classes the 409 the server sends
// when it could not issue an item code as ErrItemCodeUnavailable, keeping
// its detail. Mutant killed: the 409 left in jsonapi's bad-request class
// (errors.Is(err, ErrItemCodeUnavailable) is false).
func TestCreateItemCtx_ItemCodeUnavailable(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusConflict)
		_, _ = w.Write([]byte(`{"errors":[{"code":"item_code_unavailable","title":"Conflict","detail":"Could not issue an item code. Try again."}]}`))
	}))
	defer srv.Close()

	_, err := CreateItemCtx(context.Background(), srv.URL, "", []byte(`{}`), "", nil)
	if !errors.Is(err, ErrItemCodeUnavailable) {
		t.Fatalf("want ErrItemCodeUnavailable, got %v", err)
	}
	if err.Error() != "HTTP 409: Could not issue an item code. Try again." {
		t.Errorf("message: %v", err)
	}
}

// TestCreateItemCtx_OtherConflictKeepsItsClass leaves a 409 with another
// code in jsonapi's bad-request class. Mutant killed: every 409 classed as
// ErrItemCodeUnavailable.
func TestCreateItemCtx_OtherConflictKeepsItsClass(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusConflict)
		_, _ = w.Write([]byte(`{"errors":[{"code":"conflict","detail":"taken"}]}`))
	}))
	defer srv.Close()

	_, err := CreateItemCtx(context.Background(), srv.URL, "", []byte(`{}`), "", nil)
	if errors.Is(err, ErrItemCodeUnavailable) || !errors.Is(err, jsonapi.ErrBadRequest) {
		t.Errorf("want the bad-request class, got %v", err)
	}
}
