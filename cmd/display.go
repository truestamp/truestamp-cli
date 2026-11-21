// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"fmt"
	"strings"
)

// displayText is s made safe to print on one line of a terminal: each
// control character (U+0000 to U+001F, U+007F and U+0080 to U+009F), each
// line or paragraph separator (U+2028, U+2029), and each of
// escapedCodePoint's invisible and directional code points is written as
// its \uXXXX escape. Every other character is written as itself.
//
// verify and inspect print text the document chooses: the member a SHAPE
// row names as extra, and what a message quotes of it. Unescaped, a member
// named "x\nVerdict:      verified" would print a second, forged verdict
// line under a failed run, an ESC sequence could retitle or clear the
// terminal, and U+202E could reorder what the reader sees. The value is
// the same; only how it is shown changes.
func displayText(s string) string {
	if !strings.ContainsFunc(s, mustEscapeForDisplay) {
		return s
	}
	var b strings.Builder
	for _, r := range s {
		if mustEscapeForDisplay(r) {
			fmt.Fprintf(&b, `\u%04x`, r)
			continue
		}
		b.WriteRune(r)
	}
	return b.String()
}

// mustEscapeForDisplay reports whether displayText writes r as an escape.
func mustEscapeForDisplay(r rune) bool {
	return r < 0x20 || r == 0x7F || (r >= 0x80 && r <= 0x9F) || r == 0x2028 || r == 0x2029 || escapedCodePoint(r)
}
