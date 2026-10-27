// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: Apache-2.0

package verify

import (
	"encoding/json"
	"fmt"
	"strings"
)

// Status is the outcome of a step the Truestamp service reports. Only
// `fail` fails a proof: a warn is something worth knowing that is not a
// defect, a skip is a check that did not run, an info is a recorded value
// carrying no judgement.
type Status int

const (
	StatusPass Status = iota
	StatusFail
	StatusSkip
	StatusWarn
	StatusInfo
)

// statusStrings maps Status values to their wire strings, the service's
// vocabulary.
var statusStrings = map[Status]string{
	StatusPass: "pass",
	StatusFail: "fail",
	StatusSkip: "skip",
	StatusWarn: "warn",
	StatusInfo: "info",
}

var statusFromString map[string]Status

func init() {
	statusFromString = make(map[string]Status, len(statusStrings))
	for k, v := range statusStrings {
		statusFromString[v] = k
	}
}

// String returns the wire string of a status.
func (s Status) String() string {
	if str, ok := statusStrings[s]; ok {
		return str
	}
	return fmt.Sprintf("status(%d)", int(s))
}

// MarshalJSON encodes a Status as a JSON string.
func (s Status) MarshalJSON() ([]byte, error) {
	str, ok := statusStrings[s]
	if !ok {
		return nil, fmt.Errorf("unknown status %d", s)
	}
	return json.Marshal(str)
}

// UnmarshalJSON decodes a JSON string into a Status. A string outside the
// five-value vocabulary is an error, not a value: the verdict rule ("a
// proof passes when no step is fail") only holds when every status is
// known, so a status this CLI cannot read is handled by
// [Step.UnmarshalJSON], which grades it fail and discloses it.
func (s *Status) UnmarshalJSON(data []byte) error {
	var str string
	if err := json.Unmarshal(data, &str); err != nil {
		return err
	}
	v, err := StatusFromString(str)
	if err != nil {
		return err
	}
	*s = v
	return nil
}

// StatusFromString parses a status string. Returns an error for unknown
// values, and StatusFail alongside it so a caller that ignores the error
// still fails closed.
func StatusFromString(s string) (Status, error) {
	if v, ok := statusFromString[s]; ok {
		return v, nil
	}
	return StatusFail, fmt.Errorf("unknown status %q", s)
}

// Category identifiers, in the wire spelling the service uses.
const (
	CatDataIntegrity = "data_integrity"
	CatStructural    = "structural"
)

// Step group names: the service's spelling of the expected-hash
// comparison, and the group of this CLI's own row when the service's
// verdict and its steps disagree, which is never a pass or a warn.
const (
	groupHashComparison = "Hash Comparison"
	groupServerVerdict  = "Server Verdict"
)

// Step is a single result the service reports, or one this CLI adds
// beside them.
type Step struct {
	Group    string `json:"group"`
	Status   Status `json:"status"`
	Category string `json:"category"`
	Message  string `json:"message"`
}

// stepWire is Step's decode shape. Status is held raw so an absent key is
// distinguishable from a present one.
type stepWire struct {
	Group    string          `json:"group"`
	Category string          `json:"category"`
	Status   json.RawMessage `json:"status"`
	Message  string          `json:"message"`
}

// UnmarshalJSON decodes a step supplied by the service, failing closed on
// any status this CLI cannot read as one of the five: an absent status
// key or an unknown word is graded fail and says so in its own message,
// so a server-reported outcome can never be scored as a pass by accident.
func (s *Step) UnmarshalJSON(data []byte) error {
	var w stepWire
	if err := json.Unmarshal(data, &w); err != nil {
		return err
	}
	s.Group = w.Group
	s.Category = w.Category
	s.Message = w.Message

	raw := strings.TrimSpace(string(w.Status))
	if raw == "" || raw == "null" {
		s.Status = StatusFail
		s.Message = annotateStep(w.Message, "the server reported no status for this step, so it cannot be read as having passed")
		return nil
	}

	var parsed Status
	if err := parsed.UnmarshalJSON(w.Status); err != nil {
		s.Status = StatusFail
		s.Message = annotateStep(w.Message, fmt.Sprintf(
			"the server reported status %s, which is not one of pass/fail/warn/skip/info, so it cannot be read as having passed",
			truncateToken(raw)))
		return nil
	}
	s.Status = parsed
	return nil
}

func annotateStep(msg, note string) string {
	if msg == "" {
		return note
	}
	return msg + " [" + note + "]"
}

// truncateToken bounds an echoed wire token so a hostile or broken server
// cannot inject an unbounded string into a message.
func truncateToken(s string) string {
	const max = 40
	if len(s) <= max {
		return s
	}
	return s[:max] + "..."
}

// Temporal carries the instants the service reports, in its field names.
type Temporal struct {
	SubmittedAt   string `json:"submitted_at,omitempty"`
	CommittedAt   string `json:"committed_at,omitempty"`
	StellarCommit string `json:"stellar_commit,omitempty"`
	BitcoinCommit string `json:"bitcoin_commit,omitempty"`
}

// Report is the service's verdict on a proof: the steps it reports, with
// the rows this CLI adds beside them.
type Report struct {
	Steps []Step

	// ExpectedHash is the caller's normalized expected hash, "" when none
	// was supplied.
	ExpectedHash string
}

// Passed returns true if no step has StatusFail.
func (r *Report) Passed() bool {
	for _, s := range r.Steps {
		if s.Status == StatusFail {
			return false
		}
	}
	return true
}

func (r *Report) add(group, category string, status Status, msg string) {
	r.Steps = append(r.Steps, Step{Group: group, Category: category, Status: status, Message: msg})
}

func (r *Report) pass(group, category, msg string) { r.add(group, category, StatusPass, msg) }
func (r *Report) fail(group, category, msg string) { r.add(group, category, StatusFail, msg) }
func (r *Report) skip(group, category, msg string) { r.add(group, category, StatusSkip, msg) }
func (r *Report) warn(group, category, msg string) { r.add(group, category, StatusWarn, msg) }

// check appends a pass carrying passMsg, or a fail carrying failMsg, so a
// failing message never reads as a positive assertion.
func (r *Report) check(group, category string, ok bool, passMsg, failMsg string) bool {
	if ok {
		r.pass(group, category, passMsg)
	} else {
		r.fail(group, category, failMsg)
	}
	return ok
}
