// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: Apache-2.0

package verify

import (
	"io"
	"strings"

	"github.com/truestamp/truestamp-cli/internal/proof"
	"github.com/truestamp/truestamp-cli/internal/ui"
)

// PresentRejection writes a rejection of the earlier bundle parser: no
// report exists, only its identifier and one line of advice.
func PresentRejection(w io.Writer, err error) {
	code := proof.RejectionCode(err)
	detail := err.Error()
	if re, ok := err.(*proof.RejectionError); ok {
		detail = re.Detail
	}
	ui.Fprintf(w, "\n  REJECTED: %s\n  %s\n%s\n", code, detail, indent(proof.RejectionAdvice(code)))
}

func indent(s string) string {
	lines := strings.Split(s, "\n")
	for i := range lines {
		lines[i] = "  " + lines[i]
	}
	return strings.Join(lines, "\n")
}
