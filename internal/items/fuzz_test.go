// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: Apache-2.0

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
