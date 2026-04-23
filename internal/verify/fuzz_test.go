// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: MIT

package verify

import (
	"os"
	"path/filepath"
	"testing"
)

// FuzzRunFromBytes is the end-to-end parse-then-verify fuzz target.
// Arbitrary bytes shouldn't crash the pipeline — every malformed
// proof must surface as a typed error from ParseBytes or as a
// StatusFail Step in the returned Report.
func FuzzRunFromBytes(f *testing.F) {
	f.Add([]byte(""))
	f.Add([]byte("not a proof"))
	if data := readFixture(filepath.Join("testdata", "proof_item.json")); len(data) > 0 {
		f.Add(data)
	}
	if data := readFixture(filepath.Join("testdata", "proof_item.cbor")); len(data) > 0 {
		f.Add(data)
	}

	f.Fuzz(func(t *testing.T, data []byte) {
		_, _ = RunFromBytes(data, "fuzz.json", Options{SkipExternal: true, SkipSignatures: true})
	})
}

// FuzzStatusUnmarshalJSON: the JSON decoder branch of the Status
// enum. Arbitrary bytes must round-trip to StatusInfo on unknown
// labels (the defined fallback) without crashing.
func FuzzStatusUnmarshalJSON(f *testing.F) {
	f.Add([]byte(`"pass"`))
	f.Add([]byte(`"fail"`))
	f.Add([]byte(`"unknown"`))
	f.Add([]byte(`null`))
	f.Add([]byte(`{`))

	f.Fuzz(func(t *testing.T, data []byte) {
		var s Status
		_ = s.UnmarshalJSON(data)
	})
}

func readFixture(path string) []byte {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	return data
}
