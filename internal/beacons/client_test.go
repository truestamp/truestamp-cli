// Copyright (c) 2021-2026 Truestamp, Inc.
// SPDX-License-Identifier: MIT

package beacons

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

const (
	validID    = "019db702-b08c-73dc-a7cd-2c5e011f1dad"
	validHash  = "ffe86dc05a0c7b42279f7fa6afb016cd6928980d24673051fc58731492ce2a1b"
	validPrev  = "1c4812bdfec2bf29333136d86bc996f866e38177acc90565a0554c7ec698029b"
	validTS    = "2026-04-22T21:05:00.000000Z"
	validBody  = `{"id":"` + validID + `","hash":"` + validHash + `","timestamp":"` + validTS + `","previous_hash":"` + validPrev + `"}`
	wrappedOne = `{"result":` + validBody + `}`
)

func newServer(t *testing.T, fn http.HandlerFunc) (Config, func()) {
	t.Helper()
	srv := httptest.NewServer(fn)
	return Config{APIURL: srv.URL, APIKey: "test-key"}, srv.Close
}

func TestLatest_Bare(t *testing.T) {
	cfg, stop := newServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/beacons/latest" {
			t.Errorf("unexpected path %q", r.URL.Path)
		}
		if r.Header.Get("Authorization") != "Bearer test-key" {
			t.Errorf("missing Bearer header")
		}
		if r.Header.Get("Accept") != "application/vnd.api+json" {
			t.Errorf("wrong Accept header: %q", r.Header.Get("Accept"))
		}
		_, _ = w.Write([]byte(validBody))
	})
	defer stop()

	b, err := Latest(context.Background(), cfg)
	if err != nil {
		t.Fatalf("Latest: %v", err)
	}
	if b.ID != validID || b.Hash != validHash || b.PreviousHash != validPrev {
		t.Errorf("wrong fields: %+v", b)
	}
}

func TestLatest_WrappedResult(t *testing.T) {
	cfg, stop := newServer(t, func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(wrappedOne))
	})
	defer stop()
	b, err := Latest(context.Background(), cfg)
	if err != nil {
		t.Fatalf("Latest: %v", err)
	}
	if b.Hash != validHash {
		t.Errorf("unwrap failed: %+v", b)
	}
}

func TestList_Bare(t *testing.T) {
	cfg, stop := newServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.RawQuery != "limit=3" {
			t.Errorf("expected limit=3 query, got %q", r.URL.RawQuery)
		}
		_, _ = w.Write([]byte(`[` + validBody + `,` + validBody + `]`))
	})
	defer stop()
	got, err := List(context.Background(), cfg, 3)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(got) != 2 {
		t.Errorf("want 2 entries, got %d", len(got))
	}
}

func TestList_WrappedArray(t *testing.T) {
	cfg, stop := newServer(t, func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"result":[` + validBody + `]}`))
	})
	defer stop()
	got, err := List(context.Background(), cfg, 0)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(got) != 1 {
		t.Errorf("want 1 entry, got %d", len(got))
	}
}

func TestGet_HappyPath(t *testing.T) {
	cfg, stop := newServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/beacons/"+validID {
			t.Errorf("unexpected path %q", r.URL.Path)
		}
		_, _ = w.Write([]byte(validBody))
	})
	defer stop()
	b, err := Get(context.Background(), cfg, validID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if b.ID != validID {
		t.Errorf("wrong id: %q", b.ID)
	}
}

func TestGet_ClientSideUUIDCheck(t *testing.T) {
	// Server should never be hit for a bad UUID.
	cfg, stop := newServer(t, func(_ http.ResponseWriter, _ *http.Request) {
		t.Fatal("server should not be called for invalid UUID")
	})
	defer stop()
	if _, err := Get(context.Background(), cfg, "not-a-uuid"); err == nil {
		t.Fatal("expected validation error")
	}
}

func TestByHash_HappyPath(t *testing.T) {
	cfg, stop := newServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/beacons/by-hash/"+validHash {
			t.Errorf("unexpected path %q", r.URL.Path)
		}
		_, _ = w.Write([]byte(validBody))
	})
	defer stop()
	b, err := ByHash(context.Background(), cfg, validHash)
	if err != nil {
		t.Fatalf("ByHash: %v", err)
	}
	if b.Hash != validHash {
		t.Errorf("wrong hash: %q", b.Hash)
	}
}

func TestByHash_ClientSideHashCheck(t *testing.T) {
	cfg, stop := newServer(t, func(_ http.ResponseWriter, _ *http.Request) {
		t.Fatal("server should not be called for invalid hash")
	})
	defer stop()
	if _, err := ByHash(context.Background(), cfg, "ABCDEF"); err == nil {
		t.Fatal("expected validation error for short/uppercase hash")
	}
}

func TestError_401(t *testing.T) {
	cfg, stop := newServer(t, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"errors":[{"detail":"Authentication required"}]}`))
	})
	defer stop()
	_, err := Latest(context.Background(), cfg)
	if !errors.Is(err, ErrUnauthorized) {
		t.Fatalf("expected ErrUnauthorized, got %v", err)
	}
	if !strings.Contains(err.Error(), "Authentication required") {
		t.Errorf("detail not preserved: %v", err)
	}
}

func TestError_404(t *testing.T) {
	cfg, stop := newServer(t, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"errors":[{"detail":"Beacon not found"}]}`))
	})
	defer stop()
	_, err := ByHash(context.Background(), cfg, validHash)
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestError_400(t *testing.T) {
	cfg, stop := newServer(t, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"errors":[{"detail":"limit must be 1..100"}]}`))
	})
	defer stop()
	_, err := List(context.Background(), cfg, 500)
	if !errors.Is(err, ErrBadRequest) {
		t.Fatalf("expected ErrBadRequest, got %v", err)
	}
	if !strings.Contains(err.Error(), "1..100") {
		t.Errorf("detail not preserved: %v", err)
	}
}

func TestError_429_RetryAfter(t *testing.T) {
	cfg, stop := newServer(t, func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Retry-After", "42")
		w.WriteHeader(http.StatusTooManyRequests)
		_, _ = w.Write([]byte(`{"errors":[{"detail":"Too many requests"}]}`))
	})
	defer stop()
	_, err := Latest(context.Background(), cfg)
	if !errors.Is(err, ErrRateLimited) {
		t.Fatalf("expected ErrRateLimited, got %v", err)
	}
	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("not an *APIError: %v", err)
	}
	if apiErr.RetryAfter != "42" {
		t.Errorf("Retry-After not surfaced: %q", apiErr.RetryAfter)
	}
}

func TestError_5xx(t *testing.T) {
	cfg, stop := newServer(t, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte("<html>oops</html>"))
	})
	defer stop()
	_, err := Latest(context.Background(), cfg)
	if !errors.Is(err, ErrServer) {
		t.Fatalf("expected ErrServer, got %v", err)
	}
	if !strings.Contains(err.Error(), "HTML") {
		t.Errorf("HTML hint missing: %v", err)
	}
}

func TestError_MissingAPIKey(t *testing.T) {
	_, err := Latest(context.Background(), Config{APIURL: "http://unused"})
	if !errors.Is(err, ErrUnauthorized) {
		t.Fatalf("expected ErrUnauthorized, got %v", err)
	}
}

func TestParseShape_MissingField(t *testing.T) {
	cfg, stop := newServer(t, func(w http.ResponseWriter, _ *http.Request) {
		// Missing previous_hash.
		_, _ = w.Write([]byte(`{"id":"` + validID + `","hash":"` + validHash + `","timestamp":"` + validTS + `"}`))
	})
	defer stop()
	_, err := Latest(context.Background(), cfg)
	if err == nil {
		t.Fatal("expected validation error for missing previous_hash")
	}
}

func TestValidateHash(t *testing.T) {
	cases := []struct {
		in string
		ok bool
	}{
		{validHash, true},
		{strings.ToUpper(validHash), false}, // must be lowercase
		{"", false},
		{"abc", false},
		{strings.Repeat("z", 64), false}, // not hex
	}
	for _, c := range cases {
		err := ValidateHash(c.in)
		if c.ok && err != nil {
			t.Errorf("ValidateHash(%q) unexpected err: %v", c.in, err)
		}
		if !c.ok && err == nil {
			t.Errorf("ValidateHash(%q) should have errored", c.in)
		}
	}
}

func TestValidateUUIDv7(t *testing.T) {
	cases := []struct {
		in string
		ok bool
	}{
		{validID, true},
		{"019db702b08c73dca7cd2c5e011f1dad", true},      // no hyphens
		{"00000000-0000-4000-8000-000000000000", false}, // v4, not v7
		{"not-a-uuid", false},
		{"", false},
	}
	for _, c := range cases {
		err := ValidateUUIDv7(c.in)
		if c.ok && err != nil {
			t.Errorf("ValidateUUIDv7(%q) unexpected err: %v", c.in, err)
		}
		if !c.ok && err == nil {
			t.Errorf("ValidateUUIDv7(%q) should have errored", c.in)
		}
	}
}
