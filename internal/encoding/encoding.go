// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: MIT

// Package encoding is a small translation layer over the stdlib
// encoders used across the Truestamp wire format: hex, base64 (standard,
// RFC 4648 §4) and base64url (RFC 4648 §5). "Binary" is a pass-through
// used as the counterpart to any textual encoding when bytes move
// through stdin/stdout.
//
// The goal is a uniform behaviour across platforms for the `truestamp
// encode`, `truestamp decode`, and `truestamp convert` sub-commands so
// users do not have to shell out to `xxd`, `base64`, or similar tools
// whose flags differ between macOS, BSD, GNU coreutils, and Git-Bash.
package encoding

import (
	"bytes"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
)

// Encoding enumerates the supported representations.
type Encoding int

// Supported encodings.
const (
	Binary Encoding = iota
	Hex
	Base64Std
	Base64URL
)

// Name returns the canonical string form (used by --json output).
func (e Encoding) Name() string {
	switch e {
	case Binary:
		return "binary"
	case Hex:
		return "hex"
	case Base64Std:
		return "base64"
	case Base64URL:
		return "base64url"
	default:
		return "unknown"
	}
}

// AllNames returns the canonical names in display order.
func AllNames() []string {
	return []string{"binary", "hex", "base64", "base64url"}
}

// Parse looks up an Encoding by any accepted spelling (case-insensitive):
//
//	binary   → bin, binary, raw, bytes
//	hex      → hex, base16
//	base64   → b64, base64, base64std, base64-std
//	base64url→ b64url, base64url, base64-url, base64_url, urlsafe
func Parse(s string) (Encoding, error) {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "bin", "binary", "raw", "bytes":
		return Binary, nil
	case "hex", "base16":
		return Hex, nil
	case "b64", "base64", "base64std", "base64-std", "base64_std":
		return Base64Std, nil
	case "b64url", "base64url", "base64-url", "base64_url", "urlsafe", "url-safe":
		return Base64URL, nil
	}
	return Binary, fmt.Errorf("unsupported encoding %q (try hex, base64, base64url, binary)", s)
}

// Encode renders raw bytes in the given encoding. Binary passes the bytes
// through unchanged.
func Encode(enc Encoding, data []byte) ([]byte, error) {
	switch enc {
	case Binary:
		return data, nil
	case Hex:
		out := make([]byte, hex.EncodedLen(len(data)))
		hex.Encode(out, data)
		return out, nil
	case Base64Std:
		out := make([]byte, base64.StdEncoding.EncodedLen(len(data)))
		base64.StdEncoding.Encode(out, data)
		return out, nil
	case Base64URL:
		out := make([]byte, base64.RawURLEncoding.EncodedLen(len(data)))
		base64.RawURLEncoding.Encode(out, data)
		return out, nil
	}
	return nil, fmt.Errorf("unknown encoding %d", enc)
}

// ErrEmptyDecodeInput signals that decode was asked to decode nothing.
// Returned as an error so callers can surface a clear message; decoders
// would otherwise silently produce an empty output which hides bugs.
var ErrEmptyDecodeInput = errors.New("empty input")

// Decode interprets data as the given encoding and returns raw bytes.
// Tolerates a single trailing newline and any leading/trailing
// whitespace (common when piping from shells or editors). Rejects
// mixed/cross-encoding characters inside the payload.
func Decode(enc Encoding, data []byte) ([]byte, error) {
	if enc == Binary {
		return data, nil
	}

	trimmed := bytes.TrimSpace(data)
	if len(trimmed) == 0 {
		return nil, ErrEmptyDecodeInput
	}

	switch enc {
	case Hex:
		out := make([]byte, hex.DecodedLen(len(trimmed)))
		n, err := hex.Decode(out, trimmed)
		if err != nil {
			return nil, fmt.Errorf("hex decode: %w", err)
		}
		return out[:n], nil

	case Base64Std:
		// Accept either padded (StdEncoding) or unpadded (RawStdEncoding)
		// variants transparently. Reject any URL-safe character to
		// prevent a silent mislabelling.
		if bytes.ContainsAny(trimmed, "-_") {
			return nil, fmt.Errorf("base64 (standard) does not allow '-' or '_'; use base64url instead")
		}
		if bytes.ContainsRune(trimmed, '=') {
			out, err := base64.StdEncoding.DecodeString(string(trimmed))
			if err != nil {
				return nil, fmt.Errorf("base64 decode: %w", err)
			}
			return out, nil
		}
		out, err := base64.RawStdEncoding.DecodeString(string(trimmed))
		if err != nil {
			return nil, fmt.Errorf("base64 decode: %w", err)
		}
		return out, nil

	case Base64URL:
		if bytes.ContainsAny(trimmed, "+/") {
			return nil, fmt.Errorf("base64url does not allow '+' or '/'; use base64 (standard) instead")
		}
		if bytes.ContainsRune(trimmed, '=') {
			out, err := base64.URLEncoding.DecodeString(string(trimmed))
			if err != nil {
				return nil, fmt.Errorf("base64url decode: %w", err)
			}
			return out, nil
		}
		out, err := base64.RawURLEncoding.DecodeString(string(trimmed))
		if err != nil {
			return nil, fmt.Errorf("base64url decode: %w", err)
		}
		return out, nil
	}
	return nil, fmt.Errorf("unknown encoding %d", enc)
}
