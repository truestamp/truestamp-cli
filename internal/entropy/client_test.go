// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: Apache-2.0

package entropy

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/truestamp/truestamp-cli/internal/auth"
	"github.com/truestamp/truestamp-cli/internal/httpclient"
)

func init() {
	httpclient.Init(0)
	auth.SetDefault(auth.Resolve(auth.Credentials{APIKey: "test-key", APIKeyExplicit: true}, auth.Store{}))
}

const (
	validID   = "01a07335-b8fe-7ef2-856c-c9b4eca99850"
	validHash = "4142459a2859a57a5e5d6540579b977215d6329110adc0a890fdce6f05054f2d"
)

func serve(t *testing.T, body string) (Config, *url.URL) {
	t.Helper()
	var last url.URL
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		last = *r.URL
		w.Header().Set("Content-Type", "application/vnd.api+json")
		_, _ = w.Write([]byte(body))
	}))
	t.Cleanup(srv.Close)
	return Config{APIURL: srv.URL}, &last
}

func observationJSON(id, source string) string {
	return `{"type":"entropy_observation","id":"` + id + `","attributes":{"source":"` + source + `",
	  "state":"committed","entropy_hash":"` + validHash + `","observation_hash":"oh","metadata_hash":"mh",
	  "signing_key_id":"96b1cd2f","signature":"sig","capture_method":"http_sse",
	  "source_published_at":"2026-09-05T20:29:07.000000Z","inserted_at":"2026-09-05T20:29:07.848738Z",
	  "block_id":null,"metadata":{},
	  "entropy":{"closed_at":"2026-09-05T20:29:07Z","hash":"abc","paging_token":"19427477109604352","sequence":4523312}}}`
}

// TestGet_DecodesNumbersExactly pins the json.Number decode: a ledger
// sequence must survive as the digits the source published, never as a
// float64 that re-prints in scientific notation.
func TestGet_DecodesNumbersExactly(t *testing.T) {
	cfg, _ := serve(t, `{"data":`+observationJSON(validID, "entropy_stellar")+`}`)
	o, err := Get(context.Background(), cfg, validID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if o.ID != validID || o.Source != "entropy_stellar" || o.BlockID != "" {
		t.Errorf("decoded %+v", o)
	}
	seq, ok := o.Entropy["sequence"].(json.Number)
	if !ok || seq.String() != "4523312" {
		t.Errorf("sequence should decode as json.Number 4523312, got %T %v", o.Entropy["sequence"], o.Entropy["sequence"])
	}
}

func TestList_QueryShape(t *testing.T) {
	cfg, last := serve(t, `{"data":[`+observationJSON(validID, "entropy_nist")+`]}`)
	page, err := List(context.Background(), cfg, ListOptions{Source: "entropy_nist", Limit: 7})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(page.Observations) != 1 {
		t.Fatalf("want 1 row, got %d", len(page.Observations))
	}
	q := last.Query()
	if q.Get("sort") != "-id" || q.Get("page[limit]") != "7" || q.Get("filter[source]") != "entropy_nist" {
		t.Errorf("query = %v", q)
	}
	if _, err := List(context.Background(), cfg, ListOptions{Source: "nist", Limit: 1}); err == nil {
		t.Error("a bare source name must be refused; the wire names are the vocabulary")
	}
}

func TestByHash_FiltersAndDetectsAmbiguity(t *testing.T) {
	cfg, last := serve(t, `{"data":[`+observationJSON(validID, "entropy_stellar")+`]}`)
	o, err := ByHash(context.Background(), cfg, validHash)
	if err != nil || o.ID != validID {
		t.Fatalf("ByHash: %v %v", o, err)
	}
	if last.Query().Get("filter[entropy_hash]") != validHash || last.Query().Get("page[limit]") != "2" {
		t.Errorf("query = %v", last.Query())
	}
	cfg, _ = serve(t, `{"data":[`+observationJSON(validID, "entropy_stellar")+`,`+observationJSON(validID, "entropy_nist")+`]}`)
	if _, err := ByHash(context.Background(), cfg, validHash); !errors.Is(err, ErrAmbiguousHash) {
		t.Errorf("two matches must be ErrAmbiguousHash, got %v", err)
	}
	cfg, _ = serve(t, `{"data":[]}`)
	if _, err := ByHash(context.Background(), cfg, validHash); !errors.Is(err, ErrNotFound) {
		t.Errorf("no match must be ErrNotFound, got %v", err)
	}
	if _, err := ByHash(context.Background(), cfg, "ZZ"); err == nil {
		t.Error("a malformed hash must be refused before any request")
	}
}

func TestLatest_EmptyIsNotFound(t *testing.T) {
	cfg, last := serve(t, `{"data":[]}`)
	if _, err := Latest(context.Background(), cfg, ""); !errors.Is(err, ErrNotFound) {
		t.Errorf("want ErrNotFound, got %v", err)
	}
	if last.Query().Get("page[limit]") != "1" || last.Query().Has("filter[source]") {
		t.Errorf("latest across sources must ask for one row with no source filter, got %v", last.Query())
	}
}

func TestValidateSource(t *testing.T) {
	for _, s := range Sources {
		if err := ValidateSource(s); err != nil {
			t.Errorf("%s rejected: %v", s, err)
		}
	}
	for _, bad := range []string{"", "nist", "stellar", "ENTROPY_NIST", "entropy"} {
		if err := ValidateSource(bad); err == nil {
			t.Errorf("%q accepted", bad)
		}
	}
}
