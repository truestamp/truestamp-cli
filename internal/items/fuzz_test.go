// Copyright (c) 2021-2026 Truestamp, Inc.
// SPDX-License-Identifier: MIT

package items

import "testing"

// FuzzParseResponse: JSON:API response envelope parser. Runs against
// arbitrary bodies the server might return (including garbled bytes
// from a proxy). No panic allowed.
func FuzzParseResponse(f *testing.F) {
	f.Add([]byte(`{"data":{"id":"x","attributes":{}}}`))
	f.Add([]byte(""))
	f.Add([]byte("not json"))
	f.Add([]byte(`{"data":null}`))
	f.Add([]byte(`{"data":{"id":"x","attributes":{"claims":"not-a-map"}}}`))

	f.Fuzz(func(t *testing.T, body []byte) {
		_, _ = parseResponse(body)
	})
}

// FuzzParseError: API error envelope parser. Attacker-controlled bytes
// (server responses) feed it; must always return a typed error.
func FuzzParseError(f *testing.F) {
	f.Add(400, []byte(`{"errors":[{"detail":"bad"}]}`))
	f.Add(500, []byte("<html>oops</html>"))
	f.Add(0, []byte(""))
	f.Add(404, []byte("gibberish"))

	f.Fuzz(func(t *testing.T, code int, body []byte) {
		err := parseError(code, body)
		if err == nil {
			t.Errorf("parseError should always return a non-nil error")
		}
	})
}

// FuzzGetString: the small `attributes -> string` helper. Fuzz with
// varied maps.
func FuzzGetString(f *testing.F) {
	f.Add("hello", "key")
	f.Add("", "")

	f.Fuzz(func(t *testing.T, value, key string) {
		m := map[string]any{key: value}
		_ = getString(m, key)
	})
}
