// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: MIT

package encoding

import (
	"bytes"
	"testing"
)

// FuzzDecode_Hex feeds arbitrary strings into the hex decoder. The
// invariant is simply "no panic" — any input should either succeed or
// return a typed error.
func FuzzDecode_Hex(f *testing.F) {
	seeds := []string{"", "00", "deadbeef", "DEADBEEF", "zz", " 6a ", "abcd\n"}
	for _, s := range seeds {
		f.Add(s)
	}
	f.Fuzz(func(t *testing.T, s string) {
		_, _ = Decode(Hex, []byte(s))
	})
}

// FuzzDecode_Base64Std fuzzes standard base64 (with and without padding).
func FuzzDecode_Base64Std(f *testing.F) {
	for _, s := range []string{"", "Zm9vYmFy", "Zm9vYmE=", "Zm9v", "==", "!!"} {
		f.Add(s)
	}
	f.Fuzz(func(t *testing.T, s string) {
		_, _ = Decode(Base64Std, []byte(s))
	})
}

// FuzzDecode_Base64URL fuzzes URL-safe base64 (padded or unpadded).
func FuzzDecode_Base64URL(f *testing.F) {
	for _, s := range []string{"", "Zm9vYmFy", "Zm9vYmE=", "a-_", "++"} {
		f.Add(s)
	}
	f.Fuzz(func(t *testing.T, s string) {
		_, _ = Decode(Base64URL, []byte(s))
	})
}

// FuzzEncodeDecodeRoundTrip asserts a stronger invariant: whatever raw
// bytes go in come out unchanged after a textual round trip. Catches
// any future regression where Decode loses information.
func FuzzEncodeDecodeRoundTrip(f *testing.F) {
	f.Add([]byte{})
	f.Add([]byte("hello world"))
	f.Add(bytes.Repeat([]byte{0xff}, 64))
	f.Fuzz(func(t *testing.T, data []byte) {
		for _, enc := range []Encoding{Hex, Base64Std, Base64URL} {
			encoded, err := Encode(enc, data)
			if err != nil {
				t.Fatalf("Encode(%s) unexpected err: %v", enc.Name(), err)
			}
			if len(data) == 0 {
				// Decode rejects empty input by design.
				continue
			}
			decoded, err := Decode(enc, encoded)
			if err != nil {
				t.Fatalf("Decode(%s, %q): %v", enc.Name(), encoded, err)
			}
			if !bytes.Equal(decoded, data) {
				t.Errorf("round-trip(%s) lost data: %x != %x", enc.Name(), decoded, data)
			}
		}
	})
}

// FuzzParseEncodingName exercises the Parse name-resolver.
func FuzzParseEncodingName(f *testing.F) {
	for _, s := range []string{"hex", "HEX", "base64", "b64", "base64url", "bin", "", "nope"} {
		f.Add(s)
	}
	f.Fuzz(func(t *testing.T, s string) {
		_, _ = Parse(s)
	})
}
