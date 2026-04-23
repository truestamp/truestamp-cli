// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: MIT

package verify

import (
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
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

// UnmarshalJSON decodes a JSON string into a Status.
// Unknown strings are mapped to StatusInfo for forward compatibility.
func (s *Status) UnmarshalJSON(data []byte) error {
	var str string
	if err := json.Unmarshal(data, &str); err != nil {
		return err
	}
	if v, ok := statusFromString[str]; ok {
		*s = v
	} else {
		*s = StatusInfo
	}
	return nil
}

// StatusFromString parses a status string. Returns an error for unknown values.
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

// CategoryOrder defines the display order for categories (most critical first).
var CategoryOrder = map[string]int{
	CatDataIntegrity: 0,
	CatCryptographic: 1,
	CatStructural:    2,
	CatTiming:        3,
	CatBlockchain:    4,
}

// Step is a single verification result.
type Step struct {
	Group    string `json:"group"`
	Category string `json:"category"`
	Status   Status `json:"status"`
	Message  string `json:"message"`
}

// TemporalSummary holds the verified temporal bracket timestamps.
type TemporalSummary struct {
	ClaimedAt   string `json:"claimed_at,omitempty"`
	SubmittedAt string `json:"submitted_at,omitempty"`
	CapturedAt  string `json:"captured_at,omitempty"`
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

// CommitmentInfo holds summary data about a commitment for display.
type CommitmentInfo struct {
	Method        string `json:"method"`               // "stellar" or "bitcoin"
	Network       string `json:"network"`              // "testnet", "public", "mainnet", "regtest"
	Ledger        int    `json:"ledger"`               // Stellar ledger (0 if bitcoin)
	Height        int    `json:"height"`               // Bitcoin block height (0 if stellar)
	TxHash        string `json:"tx_hash"`              // Transaction hash (full, for explorer lookup)
	CommittedHash string `json:"committed_hash"`       // The value committed on-chain (memo_hash or op_return)
	BlockHash     string `json:"block_hash,omitempty"` // Bitcoin block header hash (for explorer lookup)
	Timestamp     string `json:"timestamp"`            // Public blockchain timestamp (ISO 8601)
	Skipped       bool   `json:"skipped"`              // true if external verification skipped
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
	SigningKeyID    string
	CommitmentInfos []CommitmentInfo
	EntropySubject  EntropySubject
}

// Passed returns true if no steps have StatusFail.
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

// HashMatched returns true if a hash was provided and matched.
func (r *Report) HashMatched() bool {
	if r.HashProvided == "" {
		return false
	}
	for _, s := range r.Steps {
		if s.Group == "Hash Comparison" && s.Status == StatusPass {
			return true
		}
	}
	return false
}

// ProofPassed returns true if all non-hash-comparison steps passed.
func (r *Report) ProofPassed() bool {
	for _, s := range r.Steps {
		if s.Status == StatusFail && s.Group != "Hash Comparison" {
			return false
		}
	}
	return true
}

// ProofFailedCount returns the number of failures excluding hash comparison.
func (r *Report) ProofFailedCount() int {
	count := 0
	for _, s := range r.Steps {
		if s.Status == StatusFail && s.Group != "Hash Comparison" {
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

func (r *Report) check(group, category string, ok bool, msg string) {
	if ok {
		r.pass(group, category, msg)
	} else {
		r.fail(group, category, msg)
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
	// Total excludes info steps (informational notes, not verification checks)
	c.Total = c.Passed + c.Failed + c.Warned + c.Skipped
	return c
}

// HexToBase64 converts a hex string to standard base64 encoding.
func HexToBase64(h string) string {
	if h == "" {
		return ""
	}
	b, err := hex.DecodeString(h)
	if err != nil {
		return h
	}
	return base64.StdEncoding.EncodeToString(b)
}
