// Copyright (c) 2021-2026 Truestamp, Inc.
// SPDX-License-Identifier: MIT

package beacons

import "testing"

// FuzzParseBeacon exercises the single-beacon JSON parser against garbage
// bytes. The parser must never panic and must reject malformed responses
// with an ordinary error. Anchors the beacons package to the repo's fuzz
// coverage requirement (see CLAUDE.md "Fuzz coverage on every parser").
func FuzzParseBeacon(f *testing.F) {
	f.Add([]byte(validBody))
	f.Add([]byte(wrappedOne))
	f.Add([]byte(`{"result":{"id":"bogus","hash":"bogus","timestamp":"","previous_hash":""}}`))
	f.Add([]byte(`{}`))
	f.Add([]byte(`null`))
	f.Add([]byte(`[`))
	f.Fuzz(func(_ *testing.T, data []byte) {
		_, _ = unmarshalBeacon(data)
	})
}

// FuzzParseBeaconList exercises the array parser symmetrically.
func FuzzParseBeaconList(f *testing.F) {
	f.Add([]byte(`[` + validBody + `]`))
	f.Add([]byte(`{"result":[` + validBody + `]}`))
	f.Add([]byte(`[]`))
	f.Add([]byte(`[null]`))
	f.Add([]byte(`{`))
	f.Fuzz(func(_ *testing.T, data []byte) {
		_, _ = unmarshalBeaconList(data)
	})
}

// FuzzValidateHash feeds arbitrary strings through the hash validator to
// make sure the regex branch can't panic on any input.
func FuzzValidateHash(f *testing.F) {
	f.Add(validHash)
	f.Add("")
	f.Add("ABCDEF")
	f.Fuzz(func(_ *testing.T, s string) {
		_ = ValidateHash(s)
	})
}

// FuzzValidateUUIDv7 feeds arbitrary strings through the UUIDv7 validator.
func FuzzValidateUUIDv7(f *testing.F) {
	f.Add(validID)
	f.Add("")
	f.Add("00000000-0000-0000-0000-000000000000")
	f.Fuzz(func(_ *testing.T, s string) {
		_ = ValidateUUIDv7(s)
	})
}
