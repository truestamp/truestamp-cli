// Copyright (c) 2021-2026 Truestamp, Inc.
// SPDX-License-Identifier: MIT

package encoding

import (
	"bytes"
	"errors"
	"testing"
)

// RFC 4648 §10 test vectors. These are the canonical "foobar" examples
// used to validate encoder/decoder implementations across the industry.
var rfc4648Vectors = []struct {
	plain  string
	hex    string
	b64    string
	b64url string // same as b64 when no URL-special chars appear; expressed padded here
}{
	{"", "", "", ""},
	{"f", "66", "Zg==", "Zg=="},
	{"fo", "666f", "Zm8=", "Zm8="},
	{"foo", "666f6f", "Zm9v", "Zm9v"},
	{"foob", "666f6f62", "Zm9vYg==", "Zm9vYg=="},
	{"fooba", "666f6f6261", "Zm9vYmE=", "Zm9vYmE="},
	{"foobar", "666f6f626172", "Zm9vYmFy", "Zm9vYmFy"},
}

func TestEncodeRFC4648(t *testing.T) {
	for _, v := range rfc4648Vectors {
		plain := []byte(v.plain)

		if got, err := Encode(Hex, plain); err != nil || string(got) != v.hex {
			t.Errorf("Encode(Hex, %q): got %q, want %q (err=%v)", v.plain, got, v.hex, err)
		}
		if got, err := Encode(Base64Std, plain); err != nil || string(got) != v.b64 {
			t.Errorf("Encode(Base64Std, %q): got %q, want %q (err=%v)", v.plain, got, v.b64, err)
		}
		// base64url canonical form in this package is unpadded; compare
		// after stripping padding from the RFC vector.
		wantURL := stripPadding(v.b64url)
		if got, err := Encode(Base64URL, plain); err != nil || string(got) != wantURL {
			t.Errorf("Encode(Base64URL, %q): got %q, want %q (err=%v)", v.plain, got, wantURL, err)
		}
	}
}

func TestDecodeRFC4648(t *testing.T) {
	for _, v := range rfc4648Vectors {
		if v.plain == "" {
			// Empty input is rejected by Decode; tested separately.
			continue
		}
		if got, err := Decode(Hex, []byte(v.hex)); err != nil || string(got) != v.plain {
			t.Errorf("Decode(Hex, %q): got %q, want %q (err=%v)", v.hex, got, v.plain, err)
		}
		if got, err := Decode(Base64Std, []byte(v.b64)); err != nil || string(got) != v.plain {
			t.Errorf("Decode(Base64Std, %q): got %q, want %q (err=%v)", v.b64, got, v.plain, err)
		}
		// base64url accepts both padded and unpadded inputs.
		for _, form := range []string{v.b64url, stripPadding(v.b64url)} {
			if got, err := Decode(Base64URL, []byte(form)); err != nil || string(got) != v.plain {
				t.Errorf("Decode(Base64URL, %q): got %q, want %q (err=%v)", form, got, v.plain, err)
			}
		}
	}
}

func TestDecodeEmpty(t *testing.T) {
	for _, enc := range []Encoding{Hex, Base64Std, Base64URL} {
		if _, err := Decode(enc, []byte("")); !errors.Is(err, ErrEmptyDecodeInput) {
			t.Errorf("Decode(%s, \"\") = %v, want ErrEmptyDecodeInput", enc.Name(), err)
		}
		if _, err := Decode(enc, []byte("   \n  ")); !errors.Is(err, ErrEmptyDecodeInput) {
			t.Errorf("Decode(%s, whitespace) = %v, want ErrEmptyDecodeInput", enc.Name(), err)
		}
	}
}

func TestDecodeTrimsWhitespace(t *testing.T) {
	// A trailing newline and surrounding spaces must not trip the decoder.
	cases := []struct {
		enc   Encoding
		in    string
		plain string
	}{
		{Hex, "666f6f\n", "foo"},
		{Hex, "  666f6f6261  ", "fooba"},
		{Base64Std, "Zm9vYmFy\n", "foobar"},
		{Base64URL, "Zm9vYmFy\n", "foobar"},
	}
	for _, c := range cases {
		got, err := Decode(c.enc, []byte(c.in))
		if err != nil || string(got) != c.plain {
			t.Errorf("Decode(%s, %q): got %q, want %q (err=%v)", c.enc.Name(), c.in, got, c.plain, err)
		}
	}
}

func TestDecodeRejectsCrossEncodingCharacters(t *testing.T) {
	// base64 (standard) alphabet: A-Za-z0-9+/  -> reject - _
	if _, err := Decode(Base64Std, []byte("Zm9_YmFy")); err == nil {
		t.Error("Decode(Base64Std) should reject '_'")
	}
	if _, err := Decode(Base64Std, []byte("Zm9-YmFy")); err == nil {
		t.Error("Decode(Base64Std) should reject '-'")
	}
	// base64url alphabet: A-Za-z0-9-_   -> reject + /
	if _, err := Decode(Base64URL, []byte("Zm9+YmFy")); err == nil {
		t.Error("Decode(Base64URL) should reject '+'")
	}
	if _, err := Decode(Base64URL, []byte("Zm9/YmFy")); err == nil {
		t.Error("Decode(Base64URL) should reject '/'")
	}
	// hex: not in 0-9a-fA-F
	if _, err := Decode(Hex, []byte("abxy")); err == nil {
		t.Error("Decode(Hex) should reject non-hex digits")
	}
}

func TestEncodeDecodeRoundTripRandom(t *testing.T) {
	// Property-style: for each encoding, a varied byte payload round-trips.
	payloads := [][]byte{
		nil,
		{0x00},
		{0xff, 0xfe, 0xfd, 0xfc},
		{0x11, 0x22, 0x33, 0x44, 0x55, 0x66, 0x77, 0x88, 0x99, 0xaa, 0xbb, 0xcc, 0xdd, 0xee, 0xff, 0x00},
		bytes.Repeat([]byte{0x5a}, 1024),
	}
	for _, enc := range []Encoding{Hex, Base64Std, Base64URL} {
		for _, p := range payloads {
			encoded, err := Encode(enc, p)
			if err != nil {
				t.Fatalf("Encode(%s): %v", enc.Name(), err)
			}
			if len(p) == 0 {
				// Decode of empty encoded form returns ErrEmptyDecodeInput.
				if _, err := Decode(enc, encoded); err == nil {
					t.Errorf("Decode(%s, empty): expected error", enc.Name())
				}
				continue
			}
			decoded, err := Decode(enc, encoded)
			if err != nil {
				t.Fatalf("Decode(%s, %q): %v", enc.Name(), encoded, err)
			}
			if !bytes.Equal(decoded, p) {
				t.Errorf("roundtrip(%s): got %x, want %x", enc.Name(), decoded, p)
			}
		}
	}
}

func TestParse(t *testing.T) {
	cases := []struct {
		in   string
		want Encoding
		ok   bool
	}{
		{"hex", Hex, true},
		{"HEX", Hex, true},
		{"base16", Hex, true},
		{"base64", Base64Std, true},
		{"b64", Base64Std, true},
		{"base64url", Base64URL, true},
		{"urlsafe", Base64URL, true},
		{"binary", Binary, true},
		{"raw", Binary, true},
		{"  bin  ", Binary, true},
		{"base58", 0, false},
		{"", 0, false},
	}
	for _, c := range cases {
		got, err := Parse(c.in)
		if c.ok {
			if err != nil || got != c.want {
				t.Errorf("Parse(%q): got (%v, %v), want (%v, nil)", c.in, got, err, c.want)
			}
		} else if err == nil {
			t.Errorf("Parse(%q): expected error", c.in)
		}
	}
}

func TestName(t *testing.T) {
	cases := []struct {
		enc  Encoding
		want string
	}{
		{Binary, "binary"},
		{Hex, "hex"},
		{Base64Std, "base64"},
		{Base64URL, "base64url"},
		{Encoding(99), "unknown"},
	}
	for _, c := range cases {
		if got := c.enc.Name(); got != c.want {
			t.Errorf("Encoding(%d).Name(): got %q, want %q", c.enc, got, c.want)
		}
	}
}

func TestAllNames(t *testing.T) {
	got := AllNames()
	want := []string{"binary", "hex", "base64", "base64url"}
	if len(got) != len(want) {
		t.Fatalf("AllNames len: got %d, want %d", len(got), len(want))
	}
	for i, name := range want {
		if got[i] != name {
			t.Errorf("AllNames[%d]: got %q, want %q", i, got[i], name)
		}
	}
}

func TestEncode_UnknownEncoding(t *testing.T) {
	if _, err := Encode(Encoding(99), []byte("x")); err == nil {
		t.Error("Encode with unknown encoding should error")
	}
}

func TestDecode_UnknownEncoding(t *testing.T) {
	if _, err := Decode(Encoding(99), []byte("x")); err == nil {
		t.Error("Decode with unknown encoding should error")
	}
}

// TestDecode_MalformedPaddedBase64 forces the error branch inside the
// padded (StdEncoding) path of Base64Std. Input contains '=' so we take
// the padded decoder branch, but the padding placement is invalid.
func TestDecode_MalformedPaddedBase64(t *testing.T) {
	if _, err := Decode(Base64Std, []byte("===")); err == nil {
		t.Error("expected error for malformed padded base64")
	}
}

// TestDecode_MalformedRawBase64 forces the error branch inside the
// unpadded (RawStdEncoding) path of Base64Std.
func TestDecode_MalformedRawBase64(t *testing.T) {
	// Odd length without padding is invalid.
	if _, err := Decode(Base64Std, []byte("Z")); err == nil {
		t.Error("expected error for malformed raw base64")
	}
}

// TestDecode_MalformedPaddedBase64URL forces the padded URLEncoding
// error branch inside Base64URL.
func TestDecode_MalformedPaddedBase64URL(t *testing.T) {
	if _, err := Decode(Base64URL, []byte("===")); err == nil {
		t.Error("expected error for malformed padded base64url")
	}
}

// TestDecode_MalformedRawBase64URL forces the unpadded RawURLEncoding
// error branch inside Base64URL.
func TestDecode_MalformedRawBase64URL(t *testing.T) {
	if _, err := Decode(Base64URL, []byte("Z")); err == nil {
		t.Error("expected error for malformed raw base64url")
	}
}

func TestBinaryPassThrough(t *testing.T) {
	data := []byte{0x00, 0xff, 0x7f}
	out, err := Encode(Binary, data)
	if err != nil || !bytes.Equal(out, data) {
		t.Errorf("Encode(Binary): got %x, want %x", out, data)
	}
	out, err = Decode(Binary, data)
	if err != nil || !bytes.Equal(out, data) {
		t.Errorf("Decode(Binary): got %x, want %x", out, data)
	}
}

// stripPadding returns s with trailing '=' characters removed.
func stripPadding(s string) string {
	for len(s) > 0 && s[len(s)-1] == '=' {
		s = s[:len(s)-1]
	}
	return s
}
