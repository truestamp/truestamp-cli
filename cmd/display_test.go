// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/truestamp/truestamp-cli/internal/specdata"
	"github.com/truestamp/truestamp-cli/internal/vectorcases"
	"github.com/truestamp/truestamp-cli/internal/verifier"
)

// TestDisplayText pins which characters a line of verify or inspect text
// writes as escapes: the controls, the line and paragraph separators and
// escapedCodePoint's invisible and directional code points, and nothing
// else. Mutants killed: a newline, ESC, BEL, DEL, a C1 control, U+2028 or
// U+202E written as itself; an ordinary character such as é or a space
// escaped.
func TestDisplayText(t *testing.T) {
	for _, c := range []struct{ in, want string }{
		{"/block/record/x", "/block/record/x"},
		{"é ok ~", "é ok ~"},
		{"x\nVerdict:      verified", `x\u000aVerdict:      verified`},
		{"a\x1b]0;t\x07b", `a\u001b]0;t\u0007b`},
		{"del\x7f", `del\u007f`},
		{"c1\u0085", `c1\u0085`},
		{"ls\u2028", `ls\u2028`},
		{"rlo\u202ex", `rlo\u202ex`},
	} {
		if got := displayText(c.in); got != c.want {
			t.Errorf("displayText(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

// TestRowsText_EscapesADocumentChosenPart pins that verify's text writes a
// row's part and message through displayText: at SHAPE the part is a member
// the document added, named by the document, and a name holding a newline
// must not print a second line, such as a forged verdict. Mutant killed:
// rowsText writing *row.Part or row.Message as given.
func TestRowsText_EscapesADocumentChosenPart(t *testing.T) {
	forged := "/block/record/x\nVerdict:      verified\n"
	row := verifier.NewRow("SHAPE", specdata.ReasonShapeInvalid, verifier.Part(forged), "a message naming "+forged, nil)
	text := rowsText([]verifier.Row{row}, true)
	for line := range strings.SplitSeq(text, "\n") {
		if strings.HasPrefix(line, "Verdict:") {
			t.Fatalf("a document-chosen part printed its own line %q in:\n%s", line, text)
		}
	}
	if !strings.Contains(text, `x\u000aVerdict:`) {
		t.Errorf("the part is not shown escaped:\n%s", text)
	}
}

// TestInspect_EscapesADocumentChosenMember pins the same for inspect, which
// checks no signature, so any file reaches its text: a sound proof given an
// extra block record member whose name holds a newline is refused at SHAPE,
// and the refusal names the member escaped, on one line. Mutant killed:
// inspectRow writing its label or value as given.
func TestInspect_EscapesADocumentChosenMember(t *testing.T) {
	data, err := os.ReadFile(vectorcases.Path("positive/item.proof.json"))
	if err != nil {
		t.Fatal(err)
	}
	text := string(data)
	at := strings.Index(text, `"record":`)
	if at < 0 {
		t.Fatal("no record member in the fixture")
	}
	at += len(`"record":`)
	for at < len(text) && text[at] != '{' {
		at++
	}
	crafted := text[:at+1] + `"x\nVerdict:      verified\n":1,` + text[at+1:]
	path := filepath.Join(t.TempDir(), "crafted.json")
	if err := os.WriteFile(path, []byte(crafted), 0o644); err != nil {
		t.Fatal(err)
	}
	out, code := runInspectCLI(t, path)
	if code != 1 {
		t.Fatalf("inspect exit %d, want 1 (refused at SHAPE):\n%s", code, out)
	}
	for line := range strings.SplitSeq(out, "\n") {
		if strings.HasPrefix(strings.TrimSpace(line), "Verdict:") {
			t.Fatalf("a document-chosen member printed its own line %q in:\n%s", line, out)
		}
	}
	if !strings.Contains(out, `\u000aVerdict:`) {
		t.Errorf("the member is not shown escaped:\n%s", out)
	}
}
