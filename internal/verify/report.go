// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: MIT

package verify

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/truestamp/truestamp-cli/internal/tscrypto"
)

// Status represents the outcome of a verification step.
type Status int

const (
	StatusPass Status = iota
	StatusFail
	StatusSkip
	StatusWarn
	StatusInfo
)

// statusStrings maps Status values to their JSON string representations.
// This is the single source of truth; statusFromString is derived from it.
var statusStrings = map[Status]string{
	StatusPass: "pass",
	StatusFail: "fail",
	StatusSkip: "skip",
	StatusWarn: "warn",
	StatusInfo: "info",
}

// statusFromString is the inverse of statusStrings, built at init time.
var statusFromString map[string]Status

func init() {
	statusFromString = make(map[string]Status, len(statusStrings))
	for k, v := range statusStrings {
		statusFromString[v] = k
	}
}

// MarshalJSON encodes a Status as a JSON string.
func (s Status) MarshalJSON() ([]byte, error) {
	str, ok := statusStrings[s]
	if !ok {
		return nil, fmt.Errorf("unknown status %d", s)
	}
	return json.Marshal(str)
}

// UnmarshalJSON decodes a JSON string into a Status. A string outside
// Appendix E.22's five-value vocabulary is an error, not a value.
//
// It used to map anything unrecognized onto StatusInfo "for forward
// compatibility". Info is verdict-neutral, so one word of drift in a
// server's status vocabulary silently turned a reported failure into a
// note, [Report.Passed] returned true, and the run printed VERIFIED at
// exit 0. E.22's verdict rule ("a proof passes when no step is fail")
// only holds when every status is known; a status this verifier cannot
// read is handled by [Step.UnmarshalJSON], which grades it fail and
// discloses it, rather than by guessing here.
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

// Category constants for user-friendly failure grouping.
const (
	CatDataIntegrity = "data_integrity"
	CatCryptographic = "cryptographic"
	CatStructural    = "structural"
	CatTiming        = "timing"
	CatBlockchain    = "blockchain"
)

// CategoryOrder defines the display order for categories. This is
// Appendix E.22's category report order, not a local preference — every
// surface that groups steps by category (the terminal Issues section and
// the --json `steps` array) must present them in it.
var CategoryOrder = map[string]int{
	CatDataIntegrity: 0,
	CatCryptographic: 1,
	CatStructural:    2,
	CatTiming:        3,
	CatBlockchain:    4,
}

// categoryRank returns a category's position in [CategoryOrder]. An
// unrecognized category sorts after every known one rather than before,
// so a category added by a future server release never displaces E.22's
// fixed order.
func categoryRank(cat string) int {
	if n, ok := CategoryOrder[cat]; ok {
		return n
	}
	return len(CategoryOrder)
}

// Step is a single verification result.
type Step struct {
	Group    string `json:"group"`
	Category string `json:"category"`
	Status   Status `json:"status"`
	Message  string `json:"message"`
}

// stepWire is Step's decode shape. Status is held raw so an absent key
// is distinguishable from a present one — the distinction Step's own
// decoder turns on.
type stepWire struct {
	Group    string          `json:"group"`
	Category string          `json:"category"`
	Status   json.RawMessage `json:"status"`
	Message  string          `json:"message"`
}

// UnmarshalJSON decodes a step supplied by a remote verifier, failing
// closed on any status this verifier cannot read as one of Appendix
// E.22's five.
//
// Two shapes used to be scored as passing. A step object carrying no
// `status` key never reached [Status.UnmarshalJSON] at all — Go leaves
// an absent field at its zero value and StatusPass is the zero value —
// so a row reading "Proof signature invalid (Ed25519)" counted as a
// pass. A status string outside the vocabulary was mapped to StatusInfo,
// which is verdict-neutral. In both cases [Report.Passed] returned true
// and the run printed VERIFIED at exit 0 over a server-reported failure.
//
// E.22's verdict rule is "a proof passes when no step is fail", which is
// only sound when every status is known. An unreadable status is graded
// fail and says so in its own message: the server reported something,
// and this verifier will not publish a verdict that depends on guessing
// what.
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

// annotateStep appends the reason a status was graded fail to the
// server's own message, keeping both: the server's text is what a reader
// needs to act on, and the note is why this verifier did not take the
// server's word for the outcome.
func annotateStep(msg, note string) string {
	if msg == "" {
		return note
	}
	return msg + " [" + note + "]"
}

// truncateToken bounds an echoed wire token so a hostile or broken
// server cannot inject an unbounded string into a rendered report.
func truncateToken(s string) string {
	const max = 40
	if len(s) <= max {
		return s
	}
	return s[:max] + "…"
}

// TemporalSummary holds the verified temporal bracket timestamps. Its JSON
// tags decode the server's /proof/verify `temporal` object (they are never
// marshaled outbound — the --json surface uses JSONTimeline). The entropy
// observation timestamp arrives on the wire as `inserted_at`; the CLI keeps
// naming it CapturedAt and surfacing it under a "Captured" label.
type TemporalSummary struct {
	ClaimedAt   string `json:"claimed_at,omitempty"`
	SubmittedAt string `json:"submitted_at,omitempty"`
	CapturedAt  string `json:"inserted_at,omitempty"`
	CommittedAt string `json:"committed_at,omitempty"`
}

// TimestampStatus indicates the validation state of the claims timestamp.
type TimestampStatus int

const (
	TimestampOK      TimestampStatus = iota // within expected range
	TimestampFuture                         // not before submission time
	TimestampStale                          // >7 days before submission
	TimestampMissing                        // no timestamp in claims
)

// Claims holds the parsed user claims for display in the report.
type Claims struct {
	Hash            string          `json:"hash"`
	HashType        string          `json:"hash_type"`
	Name            string          `json:"name"`
	Description     string          `json:"description,omitempty"`
	Timestamp       string          `json:"timestamp,omitempty"`
	URL             string          `json:"url,omitempty"`
	Location        *LatLong        `json:"location,omitempty"`
	HasMetadata     bool            `json:"-"`
	RawMetadata     json.RawMessage `json:"-"`
	TimestampStatus TimestampStatus `json:"-"`
	TimestampNote   string          `json:"-"`
}

// LatLong holds geographic coordinates.
type LatLong struct {
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
}

// ExternalStatus records what happened when a commitment's public-chain
// lookup was attempted. It replaces a bare "skipped" boolean because
// two different facts were collapsing into one: a lookup that never ran
// (no public API for the network, or --skip-external) and a lookup that
// ran and disagreed. Reporting both as "not skipped" is what let a 404
// from Horizon be published as externally_verified: true.
type ExternalStatus int

const (
	// ExternalSkipped means no external lookup was attempted. It is the
	// zero value on purpose: a CommitmentInfo built without setting this
	// field has had nothing looked up, and the safe reading of "nothing
	// looked up" is "not confirmed". ExternalConfirmed held the zero
	// value before, which meant a literal that forgot the field
	// published externally_verified: true — the inverse of the defect
	// the tri-state was introduced to fix.
	ExternalSkipped ExternalStatus = iota
	// ExternalConfirmed means an external source was consulted and
	// agreed with the bundle. Only a lookup that ran and agreed may
	// carry it, and every call site assigns it explicitly.
	ExternalConfirmed
	// ExternalFailed means an external lookup ran and did not confirm
	// the commitment.
	ExternalFailed
)

// String renders an [ExternalStatus] as the token published in the
// --json surface. The three states are named rather than collapsed to a
// boolean because "the chain disagreed" and "we never asked" are
// different facts and a consumer must be able to tell them apart.
func (e ExternalStatus) String() string {
	switch e {
	case ExternalConfirmed:
		return "confirmed"
	case ExternalFailed:
		return "failed"
	default:
		return "skipped"
	}
}

// CommitmentInfo holds summary data about a commitment for display.
type CommitmentInfo struct {
	Method        string         `json:"method"`               // "stellar" or "bitcoin"
	Network       string         `json:"network"`              // "testnet", "public", "mainnet", "regtest"
	Ledger        int            `json:"ledger"`               // Stellar ledger (0 if bitcoin)
	Height        int            `json:"height"`               // Bitcoin block height (0 if stellar)
	TxHash        string         `json:"tx_hash"`              // Transaction hash (full, for explorer lookup)
	CommittedHash string         `json:"committed_hash"`       // The value committed on-chain (memo_hash or op_return)
	BlockHash     string         `json:"block_hash,omitempty"` // Bitcoin block header hash (for explorer lookup)
	Timestamp     string         `json:"timestamp"`            // Public blockchain timestamp (ISO 8601)
	ExternalCheck ExternalStatus `json:"external_check"`       // outcome of the public-chain lookup
}

// EntropySubject holds parsed entropy data for display (internal use by presenter).
// Fields are populated based on the entropy source type.
type EntropySubject struct {
	RawSource  string // lowercase identifier emitted by ptype.Name (e.g. "entropy_nist")
	Source     string // humanized: "NIST Beacon"
	CapturedAt string // from entropy data timestamp (source-specific)

	// NIST Beacon fields
	PulseIndex  int    // pulse index number
	ChainIndex  int    // chain index
	Version     string // beacon version
	OutputValue string // pulse output value (hex)

	// Bitcoin Block fields
	BlockHash   string // block hash
	BlockHeight int    // block height
	BlockTime   int64  // block time (unix epoch)

	// Stellar Ledger fields
	LedgerHash     string // ledger hash
	LedgerSequence int    // ledger sequence number
	LedgerClosedAt string // ledger close timestamp
}

// Report holds the complete verification results (internal use only).
// For JSON output, use BuildJSONOutput() to create a presentation DTO.
type Report struct {
	Filename        string
	FileSize        int64
	ProofVersion    int
	SubjectID       string
	SubjectType     string // ptype.Name(bundle.T): "item" | "entropy_nist" | "entropy_stellar" | "entropy_bitcoin" | "block" | "beacon"
	APIURL          string // the resolved API base URL, used by the presenter to emit subject-detail + verify web links
	GeneratedAt     string
	Source          string // raw entropy source identifier
	Temporal        TemporalSummary
	Claims          Claims
	Steps           []Step
	Remote          bool
	HashProvided    string // non-empty if --hash was used
	SkippedExternal bool
	ChainLength     int

	// SigningKeyID names the key that actually signed this bundle: the
	// key id DERIVED from `pk` (E.9), which is what fills the kid slot
	// of E.16's signature payload. E.9 blesses key rotation, under which
	// it differs from the stored `b.kid`.
	SigningKeyID string

	// BlockSigningKeyID is the block's own `b.kid` field, verbatim. It
	// is an input to E.14's 0x32 preimage, not a statement about who
	// signed, so it is reported wherever a surface is describing the
	// block rather than the signature. Empty when the caller did not
	// populate it, in which case display sites fall back to
	// SigningKeyID — the two agree on every bundle whose key has not
	// rotated.
	BlockSigningKeyID string

	CommitmentInfos []CommitmentInfo
	EntropySubject  EntropySubject
}

// BlockKeyID returns the key id to display when describing the block
// itself, preferring the block's own b.kid over the derived signer.
func (r *Report) BlockKeyID() string {
	if r.BlockSigningKeyID != "" {
		return r.BlockSigningKeyID
	}
	return r.SigningKeyID
}

// Verdict is the report's single overall outcome. Every surface that
// states an outcome — the terminal verdict line, the --json `result`
// field, and (through [Report.Passed]) the process exit code — derives
// it from here. Before that was true a Hash Comparison failure raised
// for a reason other than a mismatch printed "VERIFIED" while the
// process exited 1.
type Verdict int

const (
	// VerdictVerified: no step failed and no expected hash was matched
	// against the proof.
	VerdictVerified Verdict = iota
	// VerdictFullyVerified: no step failed and the caller's expected
	// hash matched the hash carried in the proof.
	VerdictFullyVerified
	// VerdictHashMismatch: every check on the proof itself passed, but
	// the caller's expected hash disagrees with it. Appendix E.7 makes
	// this a statement about the caller's data, not a defective proof —
	// hence its own verdict rather than a bare failure. It is still a
	// failure for exit-code purposes.
	VerdictHashMismatch
	// VerdictFailed: at least one verification step failed.
	VerdictFailed
)

// Verdict classifies the report. The two passing verdicts are returned
// if and only if [Report.Passed] is true, so a rendered verdict can
// never contradict the exit code derived from Passed.
func (r *Report) Verdict() Verdict {
	if r.Passed() {
		if r.HashProvided != "" && r.HashMatched() {
			return VerdictFullyVerified
		}
		return VerdictVerified
	}
	// Failing. The one distinction worth preserving is E.7's: a failure
	// confined to the expected-hash comparison means the caller's file
	// differs from the timestamped one. That reading only holds when an
	// expected hash was actually supplied — any other Hash Comparison
	// failure (a server-supplied step, say) is an ordinary failure.
	if r.ProofPassed() && r.HashProvided != "" {
		return VerdictHashMismatch
	}
	return VerdictFailed
}

// Passed returns true if no steps have StatusFail. This is Appendix
// E.22's verdict rule ("a proof passes when no step is fail") and the
// predicate behind the process exit code. To render or serialize the
// outcome use [Report.Verdict], which never disagrees with it.
func (r *Report) Passed() bool {
	for _, s := range r.Steps {
		if s.Status == StatusFail {
			return false
		}
	}
	return true
}

// PassCount returns the number of passed steps.
func (r *Report) PassCount() int {
	count := 0
	for _, s := range r.Steps {
		if s.Status == StatusPass {
			count++
		}
	}
	return count
}

// FailedCount returns the number of failed steps.
func (r *Report) FailedCount() int {
	count := 0
	for _, s := range r.Steps {
		if s.Status == StatusFail {
			count++
		}
	}
	return count
}

// WarnCount returns the number of warning steps.
func (r *Report) WarnCount() int {
	count := 0
	for _, s := range r.Steps {
		if s.Status == StatusWarn {
			count++
		}
	}
	return count
}

// InfoCount returns the number of info steps.
func (r *Report) InfoCount() int {
	count := 0
	for _, s := range r.Steps {
		if s.Status == StatusInfo {
			count++
		}
	}
	return count
}

// SkipCount returns the number of skipped steps.
func (r *Report) SkipCount() int {
	count := 0
	for _, s := range r.Steps {
		if s.Status == StatusSkip {
			count++
		}
	}
	return count
}

// HashMatched returns true if a hash was provided and the Hash
// Comparison group establishes that it matched.
//
// A pass alone is not enough: in remote mode the group can carry both
// the server's row and the CLI's own locally computed E.7 row, and a
// server that reports a match the CLI's own comparison refutes must not
// be able to publish hash_comparison.matched: true. A fail anywhere in
// the group is decisive against a match.
func (r *Report) HashMatched() bool {
	if r.HashProvided == "" {
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

// ProofPassed returns true if all non-hash-comparison steps passed.
//
// It exists so a caller whose local file differs from the timestamped
// one is told the hash mismatched rather than that the proof is broken —
// in that case the proof is sound. It is neither the verdict nor the
// exit-code predicate: use [Report.Passed] for E.22's verdict rule and
// [Report.Verdict] for what to display.
func (r *Report) ProofPassed() bool {
	for _, s := range r.Steps {
		if s.Status == StatusFail && s.Group != groupHashComparison {
			return false
		}
	}
	return true
}

// ProofFailedCount returns the number of failures excluding hash comparison.
func (r *Report) ProofFailedCount() int {
	count := 0
	for _, s := range r.Steps {
		if s.Status == StatusFail && s.Group != groupHashComparison {
			count++
		}
	}
	return count
}

func (r *Report) pass(group, category, msg string) {
	r.Steps = append(r.Steps, Step{Group: group, Category: category, Status: StatusPass, Message: msg})
}

func (r *Report) fail(group, category, msg string) {
	r.Steps = append(r.Steps, Step{Group: group, Category: category, Status: StatusFail, Message: msg})
}

func (r *Report) skip(group, category, msg string) {
	r.Steps = append(r.Steps, Step{Group: group, Category: category, Status: StatusSkip, Message: msg})
}

func (r *Report) warn(group, category, msg string) {
	r.Steps = append(r.Steps, Step{Group: group, Category: category, Status: StatusWarn, Message: msg})
}

func (r *Report) info(group, category, msg string) {
	r.Steps = append(r.Steps, Step{Group: group, Category: category, Status: StatusInfo, Message: msg})
}

// check appends a pass carrying passMsg, or a fail carrying failMsg.
// The two messages are separate parameters on purpose: Appendix E.22
// requires that a failing message never read as a positive assertion,
// and E.23 requires independent verifiers to share a failure vocabulary
// so their reports are comparable. A single message routed to both arms
// satisfies neither — it reports "Proof signature valid (Ed25519)" as
// the reason a proof was rejected.
func (r *Report) check(group, category string, ok bool, passMsg, failMsg string) {
	if ok {
		r.pass(group, category, passMsg)
	} else {
		r.fail(group, category, failMsg)
	}
}

// StepCounts holds all step status counts computed in a single pass.
type StepCounts struct {
	Passed  int
	Failed  int
	Warned  int
	Info    int
	Skipped int
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
		case StatusInfo:
			c.Info++
		case StatusSkip:
			c.Skipped++
		}
	}
	// Total counts every step, info included. It used to exclude info,
	// which meant a consumer summing the --json `summary` got a smaller
	// number than len(steps) with no field explaining the difference —
	// on the Appendix D.4 bundle, 14 against 16. E.22 defines five
	// statuses; a summary that accounts for four of them does not
	// reconcile with the step record it summarizes.
	c.Total = c.Passed + c.Failed + c.Warned + c.Skipped + c.Info
	return c
}

// SignaturesSkipped reports whether this run left Appendix E.16's
// Ed25519 check unperformed (the --skip-signatures path emits the
// group's only skip).
//
// E.25 lists the steps a verifier MAY skip and still call a run
// verified — the external ones and E.17's keyring cross-check. E.16 is
// not among them, so a surface that states an outcome has to disclose
// that the signature was never checked; without it the same bundle
// prints "VERIFIED - proof is valid" whether its signature is genuine
// or forged.
func (r *Report) SignaturesSkipped() bool {
	for _, s := range r.Steps {
		if s.Group == groupProof && s.Status == StatusSkip {
			return true
		}
	}
	return false
}

// HexToBase64 converts a lowercase-hex string to standard base64 encoding,
// returning the input unchanged when it is not one.
//
// It decodes through [tscrypto.HexToBytes] rather than hex.DecodeString so
// that E.4's lowercase rule holds on the display surfaces too. Go's decoder
// is case-insensitive, so an uppercase cx[].memo used to be re-encoded into
// a base64 string indistinguishable from the one a conforming bundle
// produces — the presenter and the --json `committed_hash_base64` field
// laundered the very defect the verification steps now fail on. Echoing the
// offending value back instead keeps the two surfaces telling the same
// story.
func HexToBase64(h string) string {
	if h == "" {
		return ""
	}
	b, err := tscrypto.HexToBytes(h)
	if err != nil {
		return h
	}
	return base64.StdEncoding.EncodeToString(b)
}
