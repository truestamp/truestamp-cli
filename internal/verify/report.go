// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: MIT

package verify

import (
	"encoding/json"
	"fmt"
	"strings"
)

// Status represents the outcome of a verification step (Appendix E.22).
// Only `fail` fails a proof: a warn is something worth knowing that is not
// a defect, a skip is a check that did not run, an info is a recorded
// value carrying no judgement.
type Status int

const (
	StatusPass Status = iota
	StatusFail
	StatusSkip
	StatusWarn
	StatusInfo
)

// statusStrings maps Status values to their wire strings, which are the
// server's and the reference verifier's vocabulary.
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

// Badge returns the bracketed badge the reference verifier prints.
func (s Status) Badge() string {
	return "[" + strings.ToUpper(s.String()) + "]"
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
// five-value vocabulary is an error, not a value: E.22's verdict rule
// ("a proof passes when no step is fail") only holds when every status is
// known, so a status this verifier cannot read is handled by
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

// Category identifiers, in the wire spelling the server uses.
const (
	CatDataIntegrity = "data_integrity"
	CatCryptographic = "cryptographic"
	CatStructural    = "structural"
	CatTiming        = "timing"
	CatBlockchain    = "blockchain"
)

// CategoryOrder is Appendix E.22's fixed category display order.
var CategoryOrder = []string{CatDataIntegrity, CatCryptographic, CatStructural, CatTiming, CatBlockchain}

var categoryRanks = func() map[string]int {
	m := make(map[string]int, len(CategoryOrder))
	for i, c := range CategoryOrder {
		m[c] = i
	}
	return m
}()

// categoryRank returns a category's position in [CategoryOrder]. An
// unrecognized category sorts after every known one, so a category a
// future server release adds never displaces E.22's fixed order.
func categoryRank(cat string) int {
	if n, ok := categoryRanks[cat]; ok {
		return n
	}
	return len(CategoryOrder)
}

// CategoryLabel returns the display title of a category.
func CategoryLabel(cat string) string {
	switch cat {
	case CatDataIntegrity:
		return "Data Integrity"
	case CatCryptographic:
		return "Cryptographic"
	case CatStructural:
		return "Structural"
	case CatTiming:
		return "Timing"
	case CatBlockchain:
		return "Blockchain"
	}
	return cat
}

// Step group names. Every group Appendix E.22 defines carries E.22's
// spelling verbatim, because a consumer that keys on a group name is
// reading the appendix, not this file. groupServerVerdict is the one
// CLI-specific group; it is reachable only on the --remote path and never
// emits a pass or a warn.
const (
	groupHashComparison   = "Hash Comparison"
	groupSigningKey       = "Signing Key"
	groupSubjectData      = "Subject Data"
	groupInclusion        = "Inclusion Proof"
	groupBlockHash        = "Block Hash"
	groupEpoch            = "Epoch Proof"
	groupProof            = "Proof Signature"
	groupKeyBinding       = "Key Binding"
	groupSigningKeyEvent  = "Signing Key Event"
	groupStructure        = "Structure"
	groupWitnesses        = "Witnesses"
	groupSubmissionWindow = "Submission Window"
	groupSubmittedAfter   = "Submitted After"
	groupSubmittedBefore  = "Submitted Before"
	groupTemporalInfo     = "Temporal Info"
	groupStellar          = "Stellar Commitment"
	groupBitcoin          = "Bitcoin Commitment"
	groupEntropySource    = "Entropy Source"
	groupServerVerdict    = "Server Verdict"
)

// Step is a single verification result.
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

// UnmarshalJSON decodes a step supplied by a remote verifier, failing
// closed on any status this verifier cannot read as one of Appendix E.22's
// five: an absent status key or an unknown word is graded fail and says so
// in its own message, so a server-reported outcome can never be scored as
// a pass by accident.
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
// cannot inject an unbounded string into a rendered report.
func truncateToken(s string) string {
	const max = 40
	if len(s) <= max {
		return s
	}
	return s[:max] + "..."
}

// Temporal carries the submission window's recorded instants, in the
// server's field names so a CLI report and an API report are directly
// comparable. Every value is ISO 8601 at whole-second precision.
type Temporal struct {
	SubmittedAt   string `json:"submitted_at,omitempty"`
	CommittedAt   string `json:"committed_at,omitempty"`
	StellarCommit string `json:"stellar_commit,omitempty"`
	BitcoinCommit string `json:"bitcoin_commit,omitempty"`
}

// IsZero reports whether no instant was recorded.
func (t Temporal) IsZero() bool {
	return t == Temporal{}
}

// CommitmentSummary describes one commitment entry for the report header.
type CommitmentSummary struct {
	Chain   string
	Network string
}

// Report holds the complete verification results.
type Report struct {
	Filename string
	FileSize int64
	Format   string // "json" or "cbor"

	// Bundle facts for the header. VersionLiteral is the `version` value
	// as carried, so a wrong one is shown as it is rather than as 0.
	ProofVersion     int
	VersionLiteral   string
	SubjectType      string
	SubjectID        string
	BlockID          string
	GeneratedAt      string
	WitnessesCarried []string
	Commitments      []CommitmentSummary
	KeyEventCarried  bool

	// KeyringSource says where the E.17 keyring came from: "pinned <file>",
	// "fetched <url>", or "" when none was in hand.
	KeyringSource string

	Steps    []Step
	Temporal Temporal

	// ExpectedHash is the caller's normalized expected hash, "" when none
	// was supplied. Appendix E.7 requires "supplied" to be reportable
	// separately from "matched".
	ExpectedHash string

	SkippedExternal bool
	Remote          bool
}

// Passed returns true if no step has StatusFail. This is Appendix E.22's
// verdict rule and the predicate behind the process exit code.
func (r *Report) Passed() bool {
	for _, s := range r.Steps {
		if s.Status == StatusFail {
			return false
		}
	}
	return true
}

// HashProvided reports whether the caller supplied an expected hash that
// reached a comparison against `subject.claims.hash`.
func (r *Report) HashProvided() bool {
	if r.ExpectedHash == "" {
		return false
	}
	for _, s := range r.Steps {
		if s.Group == groupHashComparison && (s.Status == StatusPass || s.Status == StatusFail) {
			return true
		}
	}
	return false
}

// HashMatched reports whether the supplied expected hash matched
// `subject.claims.hash`. A fail anywhere in the group is decisive against
// a match, so a remote row that reports a match this verifier refutes can
// never publish hash_matched: true.
func (r *Report) HashMatched() bool {
	if r.ExpectedHash == "" {
		return false
	}
	sawPass := false
	for _, s := range r.Steps {
		if s.Group != groupHashComparison {
			continue
		}
		if s.Status == StatusFail {
			return false
		}
		if s.Status == StatusPass {
			sawPass = true
		}
	}
	return sawPass
}

// StepCounts holds all step status counts computed in a single pass.
type StepCounts struct {
	Passed  int
	Failed  int
	Warned  int
	Skipped int
	Info    int
	Total   int
}

// Counts computes all step status counts in a single pass.
func (r *Report) Counts() StepCounts {
	var c StepCounts
	for _, s := range r.Steps {
		switch s.Status {
		case StatusPass:
			c.Passed++
		case StatusFail:
			c.Failed++
		case StatusWarn:
			c.Warned++
		case StatusSkip:
			c.Skipped++
		case StatusInfo:
			c.Info++
		}
	}
	c.Total = c.Passed + c.Failed + c.Warned + c.Skipped + c.Info
	return c
}

// FailedCount returns the number of failed steps.
func (r *Report) FailedCount() int { return r.Counts().Failed }

// SignaturesSkipped reports whether this run left Appendix E.16's Ed25519
// check unperformed (the --skip-signatures path emits the group's only
// skip). E.25 does not list E.16 among the steps a verifier MAY skip and
// still call a run verified, so surfaces that state an outcome disclose
// it.
func (r *Report) SignaturesSkipped() bool {
	for _, s := range r.Steps {
		if s.Group == groupProof && s.Status == StatusSkip {
			return true
		}
	}
	return false
}

// StepsByCategory returns the steps grouped in Appendix E.22's category
// display order, each category's rows in emission order.
func (r *Report) StepsByCategory() [][]Step {
	out := make([][]Step, len(CategoryOrder)+1)
	for _, s := range r.Steps {
		out[categoryRank(s.Category)] = append(out[categoryRank(s.Category)], s)
	}
	return out
}

func (r *Report) add(group, category string, status Status, msg string) {
	r.Steps = append(r.Steps, Step{Group: group, Category: category, Status: status, Message: msg})
}

func (r *Report) pass(group, category, msg string) { r.add(group, category, StatusPass, msg) }
func (r *Report) fail(group, category, msg string) { r.add(group, category, StatusFail, msg) }
func (r *Report) skip(group, category, msg string) { r.add(group, category, StatusSkip, msg) }
func (r *Report) warn(group, category, msg string) { r.add(group, category, StatusWarn, msg) }
func (r *Report) info(group, category, msg string) { r.add(group, category, StatusInfo, msg) }

// check appends a pass carrying passMsg, or a fail carrying failMsg. The
// two messages are separate on purpose: Appendix E.22 requires that a
// failing message never read as a positive assertion.
func (r *Report) check(group, category string, ok bool, passMsg, failMsg string) bool {
	if ok {
		r.pass(group, category, passMsg)
	} else {
		r.fail(group, category, failMsg)
	}
	return ok
}
