// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: Apache-2.0

package verify

// Step is one entry of a [JSONReport]'s step record, in the field names of
// the earlier bundle parser's report. A rejection, the only JSONReport
// written, holds none.
type Step struct {
	Group    string `json:"group"`
	Status   string `json:"status"`
	Category string `json:"category"`
	Message  string `json:"message"`
}

// Temporal carries the instants of the earlier bundle parser's report, in
// its field names.
type Temporal struct {
	SubmittedAt   string `json:"submitted_at,omitempty"`
	CommittedAt   string `json:"committed_at,omitempty"`
	StellarCommit string `json:"stellar_commit,omitempty"`
	BitcoinCommit string `json:"bitcoin_commit,omitempty"`
}
