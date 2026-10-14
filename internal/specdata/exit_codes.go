// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: Apache-2.0

// This file ports the reference verifier's Truestamp.Verifier.ExitCodes
// (elixir/lib/truestamp/verifier/exit_codes.ex in truestamp-verification):
// the exit codes of SPEC.md section 12.2, transcribed from
// spec/exit-codes.json in its order and held equal to it by
// TestAgreement_ExitCodes, and the code for each way a run of the command
// ends. Each ending is named by the words its row begins with, so a change to
// the Go table that leaves an ending beginning no row or two rows stops the
// program at init:
//
//	EndingVerified, ..., EndingFailed   the five labels of SPEC.md section 10.5
//	EndingRuntimeError                  an invalid command line or an input that cannot be read, with no report
//	EndingFault                         a fault the verifier did not catch, or caught before steps 0 to 3 passed, with no report
//	EndingBareRejection                 a bare rejection other than configuration_error
//	EndingConfigurationError            configuration_error

package specdata

import (
	"fmt"
	"slices"
	"strings"
)

// ExitEntry is one row of the table of SPEC.md section 12.2: an exit code
// and the run it is for, as spec/exit-codes.json writes it.
type ExitEntry struct {
	Code int
	Run  string
}

// exitTable is spec/exit-codes.json, in its order.
var exitTable = []ExitEntry{
	{Code: 0, Run: "label verified"},
	{Code: 1, Run: "a runtime error outside verification, such as an invalid command line or an input that cannot be read; no report"},
	{Code: 2, Run: "the verifier stopped on a fault it did not catch, or caught one before steps 0 to 3 passed; no report"},
	{Code: 3, Run: "label failed"},
	{Code: 4, Run: "a bare rejection other than configuration_error"},
	{Code: 5, Run: "label unattributed"},
	{Code: 6, Run: "label supplied-keyring"},
	{Code: 7, Run: "label internally-consistent"},
	{Code: 8, Run: "configuration_error"},
}

// Ending is how a run of the command ended. Its value is the words the row
// of SPEC.md section 12.2 for it begins with.
type Ending string

// The endings of a run, one for each row of SPEC.md section 12.2.
const (
	EndingVerified             Ending = "label verified"
	EndingInternallyConsistent Ending = "label internally-consistent"
	EndingSuppliedKeyring      Ending = "label supplied-keyring"
	EndingUnattributed         Ending = "label unattributed"
	EndingFailed               Ending = "label failed"
	EndingRuntimeError         Ending = "a runtime error outside verification"
	EndingFault                Ending = "the verifier stopped on a fault"
	EndingBareRejection        Ending = "a bare rejection other than configuration_error"
	EndingConfigurationError   Ending = "configuration_error"
)

// endings lists every Ending.
var endings = []Ending{
	EndingVerified, EndingInternallyConsistent, EndingSuppliedKeyring,
	EndingUnattributed, EndingFailed, EndingRuntimeError, EndingFault,
	EndingBareRejection, EndingConfigurationError,
}

// exitCodeByEnding gives each ending its code.
var exitCodeByEnding = mustEndingCodes(exitTable, endings)

// endingCodes gives each ending the code of the one row that begins with
// its words. An ending that begins no row or more than one, or a row that no
// ending or more than one begins, is an error.
func endingCodes(table []ExitEntry, all []Ending) (map[Ending]int, error) {
	codes := make(map[Ending]int, len(all))
	matched := make(map[int]int, len(table))
	for _, ending := range all {
		var found []int
		for i, row := range table {
			if strings.HasPrefix(row.Run, string(ending)) {
				found = append(found, i)
			}
		}
		if len(found) != 1 {
			return nil, fmt.Errorf("specdata: ending %q begins %d rows of the exit code table, not one", ending, len(found))
		}
		codes[ending] = table[found[0]].Code
		matched[found[0]]++
	}
	for i, row := range table {
		if matched[i] != 1 {
			return nil, fmt.Errorf("specdata: row %d of the exit code table (%q) is the row of %d endings, not one", row.Code, row.Run, matched[i])
		}
	}
	return codes, nil
}

func mustEndingCodes(table []ExitEntry, all []Ending) map[Ending]int {
	codes, err := endingCodes(table, all)
	if err != nil {
		panic(err.Error())
	}
	return codes
}

// ExitCodes returns the table of SPEC.md section 12.2, in its order. Each
// call returns a new slice.
func ExitCodes() []ExitEntry {
	return slices.Clone(exitTable)
}

// ExitCode returns the exit code of ending, and whether it is an ending of
// the table.
func ExitCode(ending Ending) (int, bool) {
	code, ok := exitCodeByEnding[ending]
	return code, ok
}
