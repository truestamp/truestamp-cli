// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: MIT

package jsonapi

import (
	"errors"
	"net/http"
	"strings"
	"testing"
)

func TestParseError_WithDetail(t *testing.T) {
	err := ParseError(400, []byte(`{"errors":[{"detail":"bad claims"}]}`))
	if !strings.Contains(err.Error(), "bad claims") {
		t.Errorf("expected detail in error: %v", err)
	}
	if !errors.Is(err, ErrBadRequest) {
		t.Errorf("400 should classify as ErrBadRequest, got %v", err)
	}
}

func TestParseError_TitleOnly(t *testing.T) {
	err := ParseError(404, []byte(`{"errors":[{"title":"Not Found"}]}`))
	if !strings.Contains(err.Error(), "Not Found") {
		t.Errorf("expected title in error: %v", err)
	}
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("404 should classify as ErrNotFound, got %v", err)
	}
}

func TestParseError_HTMLBody(t *testing.T) {
	err := ParseError(502, []byte("<html>oops</html>"))
	if !strings.Contains(err.Error(), "HTML") {
		t.Errorf("expected HTML mention: %v", err)
	}
}

func TestParseError_UnparseableBody(t *testing.T) {
	err := ParseError(500, []byte("server stack trace"))
	if !strings.Contains(err.Error(), "server stack trace") {
		t.Errorf("raw body should be carried as the detail: %v", err)
	}
	if !errors.Is(err, ErrServer) {
		t.Errorf("500 should classify as ErrServer, got %v", err)
	}
}

// TestParseError_PrefersPointerBearingError pins the discriminator rule:
// the server can return several errors at once, and the one carrying a
// source.pointer wins regardless of position.
func TestParseError_PrefersPointerBearingError(t *testing.T) {
	body := `{"errors":[{"detail":"plan limit"},{"detail":"not entitled","source":{"pointer":"/data/attributes/ownership_model"}}]}`
	err := ParseError(422, []byte(body))
	if err.Pointer != "/data/attributes/ownership_model" || err.Detail != "not entitled" {
		t.Errorf("pointer-bearing error should be chosen, got pointer=%q detail=%q", err.Pointer, err.Detail)
	}
}

func TestSentinelFor_Classes(t *testing.T) {
	for _, tc := range []struct {
		status int
		want   error
	}{
		{http.StatusUnauthorized, ErrUnauthorized},
		{http.StatusForbidden, ErrForbidden},
		{http.StatusNotFound, ErrNotFound},
		{http.StatusTooManyRequests, ErrRateLimited},
		{http.StatusBadRequest, ErrBadRequest},
		{http.StatusUnprocessableEntity, ErrBadRequest},
		{http.StatusInternalServerError, ErrServer},
		{http.StatusBadGateway, ErrServer},
	} {
		if got := sentinelFor(tc.status); !errors.Is(got, tc.want) {
			t.Errorf("HTTP %d: want %v, got %v", tc.status, tc.want, got)
		}
	}
}

// FuzzParseError: the error envelope parser. Attacker-controlled bytes
// (server responses) feed it; it must always classify and never panic.
func FuzzParseError(f *testing.F) {
	f.Add(400, []byte(`{"errors":[{"detail":"bad"}]}`))
	f.Add(500, []byte("<html>oops</html>"))
	f.Add(0, []byte(""))
	f.Add(404, []byte("gibberish"))

	f.Fuzz(func(t *testing.T, code int, body []byte) {
		err := ParseError(code, body)
		if err == nil || err.Sentinel == nil {
			t.Errorf("ParseError must always return a classified error")
		}
	})
}

func TestParsePage_LiftsCursorAndTotal(t *testing.T) {
	for body, want := range map[string]PageInfo{
		`{"data":[],"links":{"next":"https://x/items?page%5Bafter%5D=ABC&page%5Blimit%5D=25"}}`:                          {NextCursor: "ABC"},
		`{"data":[],"links":{"next":"https://x/items?page[after]=ABC"}}`:                                                 {NextCursor: "ABC"},
		`{"data":[],"links":{"next":"https://x/items?page%5Blimit%5D=25"}}`:                                              {},
		`{"data":[],"links":{"next":"://not a url"}}`:                                                                    {},
		`{"data":[],"meta":{"page":{"total":593419,"limit":1}}}`:                                                         {Total: 593419},
		`{"data":[],"links":{"prev":"https://x/items?page%5Bbefore%5D=P1","next":"https://x/items?page%5Bafter%5D=N1"}}`: {NextCursor: "N1", PrevCursor: "P1"},
		`{"data":[],"links":{"prev":null,"next":null}}`:                                                                  {},
		`{"data":[]}`: {},
		`not json`:    {},
	} {
		if got := ParsePage([]byte(body)); got != want {
			t.Errorf("ParsePage(%s) = %+v, want %+v", body, got, want)
		}
	}
}
