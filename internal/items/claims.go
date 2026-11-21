// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: Apache-2.0

package items

import (
	"bytes"
	"fmt"
)

// ClaimsJSONString writes s, valid UTF-8, as a JSON string that reads back
// as s: a quotation mark, a reverse solidus and each control character
// below U+0020 are escaped, and every other character is written as
// itself. It is the one writer of a claims value that `truestamp items
// create` and the console's new-item pane build, so the two send the same
// claims text for the same input. encoding/json is not used: its v1 API
// rewrites invalid UTF-8 to U+FFFD and escapes "<", ">" and "&", and its v2
// API refuses invalid UTF-8.
func ClaimsJSONString(s string) []byte {
	var out bytes.Buffer
	out.WriteByte('"')
	for i := 0; i < len(s); i++ {
		c := s[i]
		switch {
		case c == '"' || c == '\\':
			out.WriteByte('\\')
			out.WriteByte(c)
		case c < 0x20:
			fmt.Fprintf(&out, `\u%04x`, c)
		default:
			out.WriteByte(c)
		}
	}
	out.WriteByte('"')
	return out.Bytes()
}

// MaxNumberOctets is the longest number literal, in octets as written
// (sign, point and exponent included), that the Truestamp service reads
// in a request body: its request reader, TruestampWeb.RequestJSON, refuses
// a body holding a longer one before anything else reads it, under
// Truestamp.JSONNumbers.max_octets (1,024). It is a limit on what can be
// sent, not a rule about the value: the service answers such a request
// with a bare 400 that names neither the claims nor the number.
const MaxNumberOctets = 1024

// LongNumber returns the offset, counted in octets from 0, and the length
// of the first number literal of text longer than [MaxNumberOctets], and
// true; or false when there is none. A literal is a run of the octets a
// JSON number is written with (0 to 9, "-", "+", ".", "e" and "E"),
// outside a string, that begins with a digit or "-"; a string is skipped
// with its escapes, so a number inside one is not a literal. That is the
// service's own scan (Truestamp.JSONNumbers), and in a JSON text each such
// run is exactly one number.
func LongNumber(text []byte) (offset, length int, found bool) {
	for i := 0; i < len(text); {
		c := text[i]
		switch {
		case c == '"':
			i++
			for i < len(text) && text[i] != '"' {
				if text[i] == '\\' {
					i++
				}
				i++
			}
			i++
		case c == '-' || ('0' <= c && c <= '9'):
			start := i
			for i < len(text) && isNumberOctet(text[i]) {
				i++
			}
			if i-start > MaxNumberOctets {
				return start, i - start, true
			}
		default:
			i++
		}
	}
	return 0, 0, false
}

// isNumberOctet reports whether c is one of the octets a JSON number is
// written with.
func isNumberOctet(c byte) bool {
	return ('0' <= c && c <= '9') || c == '-' || c == '+' || c == '.' || c == 'e' || c == 'E'
}
