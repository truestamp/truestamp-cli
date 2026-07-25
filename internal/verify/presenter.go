// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: MIT

package verify

import (
	"bytes"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	lipgloss "charm.land/lipgloss/v2"
	"charm.land/lipgloss/v2/table"
	"github.com/truestamp/truestamp-cli/internal/proof/ptype"
	"github.com/truestamp/truestamp-cli/internal/ui"
)

// subjectCodeForName maps the report's string subject_type (from ptype.Name)
// back to the integer code. Used by the presenter to pick a render path.
func subjectCodeForName(name string) ptype.Code {
	switch name {
	case "block":
		return ptype.Block
	case "beacon":
		return ptype.Beacon
	case "item":
		return ptype.Item
	case "entropy_nist":
		return ptype.EntropyNIST
	case "entropy_stellar":
		return ptype.EntropyStellar
	case "entropy_bitcoin":
		return ptype.EntropyBitcoin
	}
	return 0
}

// Present renders a Report to stdout with lipgloss styling.
func Present(r *Report) {
	sections := []string{
		renderResultBanner(r),
		"",
		renderProofSection(r),
		"",
		renderSubject(r),
	}

	// Hash mismatch detail (between Subject and Timeline). Gated on the
	// verdict, not on a hand-rolled predicate, so the expected/found
	// table appears exactly when the summary says HASH MISMATCH.
	if r.Verdict() == VerdictHashMismatch {
		sections = append(sections, "", renderHashMismatchDetail(r))
	}

	// Timeline (on success or hash-mismatch-but-proof-valid)
	if r.ProofPassed() {
		if temporal := renderTimeline(r); temporal != "" {
			sections = append(sections, "", temporal)
		}
	}

	// Commitments (on success or hash-mismatch-but-proof-valid)
	if r.ProofPassed() {
		if commits := renderCommitments(r); commits != "" {
			sections = append(sections, "", commits)
		}
	}

	// Verification Notes — workflow-level observations that are NOT
	// proof defects (e.g. external-hash item without --hash, or a
	// claims-only item explaining its mode). Sits before "Issues" so
	// the user sees it on every successful verify; the Issues section
	// stays reserved for actual problems.
	if notes := renderVerificationNotes(r); notes != "" {
		sections = append(sections, "", notes)
	}

	// Issues (only when there are failures or warnings)
	if issues := renderIssues(r); issues != "" {
		sections = append(sections, "", issues)
	}

	// Verification Summary (replaces old raw counts line)
	sections = append(sections, "", renderVerificationSummary(r))

	// Plain newline-join — NOT lipgloss.JoinVertical. JoinVertical pads
	// every section's lines with trailing spaces to match the widest line
	// across ALL sections. When any section contains a line wider than
	// the terminal (e.g. a long section subtitle, a 64-hex tx hash row,
	// a paragraph of prose), the padding pushes every other row over the
	// terminal width too, which the terminal then hard-wraps — producing
	// a blank visual line after every table row. The spacing collapses
	// on terminal resize because the JoinVertical output was already
	// correctly formed; only the terminal's line-wrap was inflating it.
	// strings.Join sidesteps the padding entirely.
	lipgloss.Println(strings.Join(sections, "\n"))
}

// --- Result Banner ---

func renderResultBanner(_ *Report) string {
	return ui.AccentBoldStyle().Render("  Truestamp Proof Verification")
}

// --- Proof Section ---

func renderProofSection(r *Report) string {
	header := ui.SectionHeader("Proof")

	tbl := ui.CompactTable().
		StyleFunc(metadataStyleFunc)

	tbl = tbl.Row("ID", r.SubjectID)
	tbl = tbl.Row("Type", ptype.Humanize(subjectCodeForName(r.SubjectType)))

	// Append shareable public-web links as rows of the SAME table so
	// they inherit the right-aligned-label / value column alignment.
	// r.APIURL is set by the verify cmd via opts.APIURL so we don't
	// reach into a global from a pure-logic package. Empty APIURL →
	// no links (e.g. in unit tests that exercise Present without a
	// configured API host).
	if r.APIURL != "" && r.SubjectType != "" && r.SubjectID != "" {
		if detail := ui.SubjectDetailURL(r.APIURL, r.SubjectType, r.SubjectID); detail != "" {
			tbl = tbl.Row("Details", detail)
		}
		if verify := ui.SubjectVerifyURL(r.APIURL, r.SubjectType, r.SubjectID); verify != "" {
			tbl = tbl.Row("Verify", verify)
		}
	}

	return header + "\n" + tbl.String()
}

// --- Subject Section ---

func renderSubject(r *Report) string {
	// Block and beacon share the same subject shape (no claims, no
	// observation — the block itself is what was committed) so they
	// render through the same section. The Type row upstream already
	// reads "Block" vs "Beacon" via ptype.Humanize, so the distinction
	// is preserved for the user.
	switch subjectCodeForName(r.SubjectType) {
	case ptype.Block, ptype.Beacon:
		return renderBlockSubject(r)
	case ptype.EntropyNIST, ptype.EntropyStellar, ptype.EntropyBitcoin:
		return renderEntropySubject(r)
	}
	return renderItemSubject(r)
}

func renderBlockSubject(r *Report) string {
	// Slightly different copy for plain block vs beacon: both describe
	// a block as the subject, but a beacon is the finalized-block
	// projection used for "proof of life", so name it that way.
	header := ui.SectionHeader("Block Subject")
	subtitle := ui.FaintStyle().Render("  The subject is a Truestamp block. No user claims, no observation — the block itself is what was committed.")
	if subjectCodeForName(r.SubjectType) == ptype.Beacon {
		header = ui.SectionHeader("Beacon Subject")
		subtitle = ui.FaintStyle().Render("  The subject is a Truestamp beacon — a finalized block exposed as a \"proof of life\" commitment (t=11 on the wire).")
	}

	tbl := ui.CompactTable().
		StyleFunc(metadataStyleFunc)

	tbl = tbl.Row("Block ID", r.SubjectID)
	// The block's own b.kid — this section describes block fields. The
	// key that signed the proof (E.9's pk-derived id, which may differ
	// under rotation) is named in the Commitments section instead.
	tbl = tbl.Row("Signing Key", r.BlockKeyID())
	if r.Temporal.CommittedAt != "" {
		tbl = tbl.Row("Committed", truncateToSecond(r.Temporal.CommittedAt))
	}

	return header + "\n" + subtitle + "\n" + tbl.String()
}

func renderItemSubject(r *Report) string {
	header := ui.SectionHeader("Item Claims")
	subtitle := ui.FaintStyle().Render("  Claims made by the submitter. Not independently verified by Truestamp.")

	tbl := ui.CompactTable().
		StyleFunc(metadataStyleFunc)

	if r.Claims.Name != "" {
		tbl = tbl.Row("Name", r.Claims.Name)
	}
	if r.Claims.Description != "" {
		tbl = tbl.Row("Description", r.Claims.Description)
	}
	if r.Claims.Hash != "" {
		tbl = tbl.Row("Hash", renderHashValue(r.Claims.Hash, r.Claims.HashType, r))
	}
	if r.Claims.Timestamp != "" {
		ts := truncateToSecond(r.Claims.Timestamp)
		tbl = tbl.Row("Timestamp", renderTimestampValue(ts, r.Claims))
	}
	if r.Claims.Location != nil {
		tbl = tbl.Row("Location", fmt.Sprintf("%.5f, %.5f", r.Claims.Location.Latitude, r.Claims.Location.Longitude))
	}
	if r.Claims.URL != "" {
		tbl = tbl.Row("URL", r.Claims.URL)
	}

	result := header + "\n" + subtitle + "\n" + tbl.String()

	// Pretty-print metadata JSON as a separate block if present
	if r.Claims.HasMetadata && len(r.Claims.RawMetadata) > 0 {
		result += renderMetadataBlock(r.Claims.RawMetadata)
	}

	return result
}

func renderEntropySubject(r *Report) string {
	header := ui.SectionHeader("Entropy Observation")
	subtitle := ui.FaintStyle().Render("  Verifiable entropy observed from a trusted external source.")

	tbl := ui.CompactTable().
		StyleFunc(metadataStyleFunc)

	tbl = tbl.Row("Source", r.EntropySubject.Source)
	tbl = tbl.Row("", "")
	tbl = renderEntropySourceFields(tbl, r.EntropySubject)
	if r.EntropySubject.CapturedAt != "" {
		tbl = tbl.Row("Captured", truncateToSecond(r.EntropySubject.CapturedAt))
	}

	return header + "\n" + subtitle + "\n" + tbl.String()
}

func renderMetadataBlock(raw json.RawMessage) string {
	var pretty bytes.Buffer
	if err := json.Indent(&pretty, raw, "      ", "  "); err != nil {
		return ""
	}

	label := ui.LabelStyle().Render("    Metadata")
	value := ui.FaintStyle().Render("      " + pretty.String())
	return "\n" + label + "\n" + value + "\n"
}

func renderEntropySourceFields(tbl *table.Table, es EntropySubject) *table.Table {
	switch es.RawSource {
	case "entropy_nist":
		if es.PulseIndex > 0 {
			pulseInfo := fmt.Sprintf("#%d", es.PulseIndex)
			var parts []string
			if es.ChainIndex > 0 {
				parts = append(parts, fmt.Sprintf("chain %d", es.ChainIndex))
			}
			if es.Version != "" {
				parts = append(parts, fmt.Sprintf("version %s", es.Version))
			}
			if len(parts) > 0 {
				pulseInfo += " (" + strings.Join(parts, ", ") + ")"
			}
			tbl = tbl.Row("Pulse", pulseInfo)
		}
		if es.OutputValue != "" {
			tbl = tbl.Row("Value", es.OutputValue)
		}

	case "entropy_bitcoin":
		if es.BlockHeight > 0 {
			tbl = tbl.Row("Block", fmt.Sprintf("%d", es.BlockHeight))
		}
		if es.BlockHash != "" {
			tbl = tbl.Row("Block Hash", es.BlockHash)
		}

	case "entropy_stellar":
		if es.LedgerSequence > 0 {
			tbl = tbl.Row("Ledger", fmt.Sprintf("%d", es.LedgerSequence))
		}
		if es.LedgerHash != "" {
			tbl = tbl.Row("Ledger Hash", es.LedgerHash)
		}
	}
	return tbl
}

func renderHashValue(hash, hashType string, r *Report) string {
	display := hash
	if hashType != "" {
		display += " (" + hashType + ")"
	}

	if r.HashProvided == "" {
		return lipgloss.NewStyle().Foreground(ui.Yellow).Render(display) +
			lipgloss.NewStyle().Foreground(ui.Dim).Render(" - not verified")
	}
	if r.HashMatched() {
		return lipgloss.NewStyle().Foreground(ui.Green).Render(display) +
			lipgloss.NewStyle().Foreground(ui.Green).Render(" - matches")
	}
	return lipgloss.NewStyle().Foreground(ui.Red).Render(display) +
		lipgloss.NewStyle().Foreground(ui.Red).Render(" - mismatch")
}

func renderTimestampValue(ts string, c Claims) string {
	switch c.TimestampStatus {
	case TimestampFuture:
		return lipgloss.NewStyle().Foreground(ui.Red).Render(ts) +
			lipgloss.NewStyle().Foreground(ui.Red).Render(" x "+c.TimestampNote)
	case TimestampStale:
		return lipgloss.NewStyle().Foreground(ui.Yellow).Render(ts) +
			lipgloss.NewStyle().Foreground(ui.Yellow).Render(" ! "+c.TimestampNote)
	default:
		return ts
	}
}

// --- Timeline Section ---

func renderTimeline(r *Report) string {
	t := r.Temporal
	if t.SubmittedAt == "" && t.CommittedAt == "" && t.CapturedAt == "" {
		return ""
	}

	header := ui.SectionHeader("Timeline")

	tbl := ui.CompactTable().
		StyleFunc(metadataStyleFunc)

	// Apply ui.TruncateToSecond at every display site — idempotent if the
	// value is already at second precision, and defensive against any
	// upstream change that starts storing higher-precision timestamps.
	if t.ClaimedAt != "" {
		claimed := ui.TruncateToSecond(t.ClaimedAt)
		row := claimed
		if r.Claims.TimestampStatus == TimestampStale {
			row = lipgloss.NewStyle().Foreground(ui.Yellow).Render(claimed) +
				lipgloss.NewStyle().Foreground(ui.Yellow).Render(" ! "+r.Claims.TimestampNote)
		} else if r.Claims.TimestampStatus == TimestampFuture {
			row = lipgloss.NewStyle().Foreground(ui.Red).Render(claimed) +
				lipgloss.NewStyle().Foreground(ui.Red).Render(" x "+r.Claims.TimestampNote)
		}
		tbl = tbl.Row("Claimed at", row)
	}
	if t.SubmittedAt != "" {
		tbl = tbl.Row("Submitted at", ui.TruncateToSecond(t.SubmittedAt))
	}
	if t.CapturedAt != "" {
		tbl = tbl.Row("Captured at", ui.TruncateToSecond(t.CapturedAt))
	}
	if t.CommittedAt != "" {
		tbl = tbl.Row("Committed at", ui.TruncateToSecond(t.CommittedAt))
	}

	return header + "\n" + tbl.String()
}

// --- Commitments Section ---

func renderCommitments(r *Report) string {
	if r.ChainLength == 0 {
		return ""
	}

	header := ui.SectionHeader("Commitments")

	tbl := ui.CompactTable().
		StyleFunc(metadataStyleFunc)

	// Truestamp chain info
	tbl = tbl.Row("Truestamp", fmt.Sprintf("%d-block chain, signed with key %s", r.ChainLength, r.SigningKeyID))

	dim := lipgloss.NewStyle().Foreground(ui.Dim)

	// Blockchain commitments
	for _, ci := range r.CommitmentInfos {
		switch ci.Method {
		case "stellar":
			label := "Stellar"
			tbl = tbl.Row(label, fmt.Sprintf("Ledger %d on %s %s", ci.Ledger, ci.Network, externalCheckSuffix(ci)))
			tbl = tbl.Row("", dim.Render("tx: "+ci.TxHash))
			if ci.CommittedHash != "" {
				tbl = tbl.Row("", dim.Render("memo_hash (hex): "+ci.CommittedHash))
				tbl = tbl.Row("", dim.Render("memo_hash (base64): "+HexToBase64(ci.CommittedHash)))
			}
		case "bitcoin":
			label := "Bitcoin"
			tbl = tbl.Row(label, fmt.Sprintf("Block %d on %s %s", ci.Height, ci.Network, externalCheckSuffix(ci)))
			tbl = tbl.Row("", dim.Render("tx: "+ci.TxHash))
			if ci.CommittedHash != "" {
				tbl = tbl.Row("", dim.Render("op_return (hex): "+ci.CommittedHash))
				tbl = tbl.Row("", dim.Render("op_return (base64): "+HexToBase64(ci.CommittedHash)))
			}
			if ci.BlockHash != "" {
				tbl = tbl.Row("", dim.Render("block_hash: "+ci.BlockHash))
			}
		}
	}

	return header + "\n" + tbl.String()
}

func formatCommitmentTimestamp(ts string) string {
	if ts == "" {
		return "timestamp unavailable"
	}
	return truncateToSecond(ts)
}

// externalCheckSuffix renders the parenthetical that follows a
// commitment's chain position. All three [ExternalStatus] values get
// their own wording: a lookup that never ran and a lookup that ran and
// disagreed are different facts, and printing "external verification
// skipped" for the second one is the same collapse that made
// externally_verified untrustworthy in the --json output.
func externalCheckSuffix(ci CommitmentInfo) string {
	switch ci.ExternalCheck {
	case ExternalSkipped:
		return "(external verification skipped)"
	case ExternalFailed:
		return "(external verification failed)"
	default:
		return "(" + formatCommitmentTimestamp(ci.Timestamp) + ")"
	}
}

// --- Hash Mismatch Detail ---

func renderHashMismatchDetail(r *Report) string {
	tbl := ui.CompactTable().
		StyleFunc(metadataStyleFunc)

	// Show full 64-char hashes — the user is here specifically to spot
	// the mismatch, and a head8…tail8 truncation can hide the differing
	// bytes in the middle, or render two different hashes as visually
	// identical when they share prefix + suffix by coincidence.
	tbl = tbl.Row("Expected", r.HashProvided+
		lipgloss.NewStyle().Foreground(ui.Dim).Render(" (from --hash)"))
	tbl = tbl.Row("Found", r.Claims.Hash+
		lipgloss.NewStyle().Foreground(ui.Dim).Render(" (in proof)"))

	return tbl.String()
}

// --- Verification Notes Section ---
//
// Workflow-level observations rendered between "Commitments" and
// "Issues". Notes are emitted by verify.go under
// groupVerificationNotes — they are NOT proof defects, just things
// worth surfacing about the verify session itself:
//
//   - External-hash item without --hash: a workflow nudge that the
//     user can additionally confirm a local file against the
//     timestamped hash.
//   - Claims-only item: an explicit acknowledgment that the
//     timestamped data is embedded in s.d, so there is nothing
//     external to compare.
//
// Every StatusInfo step lands here too, whatever group it belongs to.
// An info row states something true about the proof — "the claims hash
// is well formed", "the signature was verified with the embedded key" —
// and rendering it under the "Issues" heading turns a positive
// assertion into an apparent defect.
//
// Rendering is intentionally simpler than renderIssues: a flat list
// with !/i icons. Notes do not have categories or detail lookups
// because they aren't failures; the message is the whole content.
func renderVerificationNotes(r *Report) string {
	var notes []Step
	for _, s := range r.Steps {
		if s.Group == groupVerificationNotes || s.Status == StatusInfo {
			notes = append(notes, s)
		}
	}
	if len(notes) == 0 {
		return ""
	}

	header := ui.SectionHeader("Verification Notes")
	warnStyle := lipgloss.NewStyle().Foreground(ui.Yellow)
	infoStyle := lipgloss.NewStyle().Foreground(ui.Dim)

	var lines []string
	for _, s := range notes {
		switch s.Status {
		case StatusWarn:
			lines = append(lines, warnStyle.Render("  ! "+s.Message))
		case StatusInfo:
			lines = append(lines, infoStyle.Render("  i "+s.Message))
		default:
			// Fallback rendering for unexpected statuses — kept
			// defensive so a future caller using this group with a
			// pass/fail/skip status still renders something readable
			// rather than dropping the line silently.
			lines = append(lines, infoStyle.Render("  · "+s.Message))
		}
	}

	return header + "\n" + strings.Join(lines, "\n")
}

// --- Issues Section ---

func renderIssues(r *Report) string {
	// Collect failures, warnings and skips. Passing steps are omitted,
	// and so are info steps and the Verification Notes group — both are
	// rendered by renderVerificationNotes, because neither is a proof
	// defect and listing them under "Issues" misreports them.
	var issues []Step
	for _, s := range r.Steps {
		if s.Status == StatusPass || s.Status == StatusInfo {
			continue
		}
		if s.Group == groupVerificationNotes {
			continue
		}
		issues = append(issues, s)
	}

	if len(issues) == 0 {
		return ""
	}

	header := ui.SectionHeader("Issues")

	// Group by category
	type catGroup struct {
		name  string
		order int
		steps []Step
	}
	catMap := map[string]*catGroup{}
	for _, s := range issues {
		cat := s.Category
		if cat == "" {
			cat = CatStructural // fallback
		}
		if _, ok := catMap[cat]; !ok {
			catMap[cat] = &catGroup{name: cat, order: categoryRank(cat)}
		}
		catMap[cat].steps = append(catMap[cat].steps, s)
	}

	// Sort categories by order
	var cats []*catGroup
	for _, g := range catMap {
		cats = append(cats, g)
	}
	sort.Slice(cats, func(i, j int) bool { return cats[i].order < cats[j].order })

	// Sort steps within each category by severity: failures, then the
	// warnings that qualify the verdict, then the checks that were not
	// attempted at all.
	//
	// Sorting on the Status value itself is what this used to do, and its
	// comment ("StatusFail(1) before StatusWarn(3)") named only the two
	// endpoints — StatusSkip is 2, so every warn rendered BELOW every
	// skip. Under --skip-external that buries "this run establishes
	// nothing about who signed the proof" under a run of unattempted
	// external checks. The iota ordering of Status is a declaration
	// order, not a severity ranking, so the ranking is stated here
	// explicitly rather than inherited from it.
	for _, g := range cats {
		sort.SliceStable(g.steps, func(i, j int) bool {
			return issueSeverityRank(g.steps[i].Status) < issueSeverityRank(g.steps[j].Status)
		})
	}

	var sections []string
	failStyle := lipgloss.NewStyle().Foreground(ui.Red).Bold(true)
	warnStyle := lipgloss.NewStyle().Foreground(ui.Yellow)
	skipStyle := lipgloss.NewStyle().Foreground(ui.Dim)
	detailStyle := lipgloss.NewStyle().Foreground(ui.Dim).PaddingLeft(6)

	for _, g := range cats {
		catHeader := lipgloss.NewStyle().Foreground(ui.Label).Bold(true).PaddingLeft(2).Render(categoryDisplayName(g.name))
		var lines []string
		lines = append(lines, catHeader)
		for _, s := range g.steps {
			var line string
			switch s.Status {
			case StatusFail:
				line = failStyle.Render("    x " + s.Message)
			case StatusWarn:
				line = warnStyle.Render("    ! " + s.Message)
			default:
				// Skips, and defensively anything else that reaches
				// here, so no step is dropped without a line.
				line = skipStyle.Render("    - " + s.Message)
			}
			lines = append(lines, line)

			// Add detail line for failures and warnings only
			if s.Status == StatusFail || s.Status == StatusWarn {
				if detail := lookupFailureDetail(s.Message); detail != "" {
					lines = append(lines, detailStyle.Render(detail))
				}
			}
		}
		sections = append(sections, strings.Join(lines, "\n"))
	}

	return header + "\n" + strings.Join(sections, "\n\n")
}

// issueSeverityRank orders the three statuses the Issues section can
// carry. Pass and Info never reach it (renderIssues filters them), but
// they are ranked last anyway so an unexpected row sorts to the bottom
// rather than to the top.
func issueSeverityRank(s Status) int {
	switch s {
	case StatusFail:
		return 0
	case StatusWarn:
		return 1
	case StatusSkip:
		return 2
	default:
		return 3
	}
}

// inclusionDetail explains a failed leaf-to-root walk over the
// Truestamp block's Merkle tree. It says "subject" rather than "item"
// because the same walk runs over entropy observations, whose failure
// used to be reported as an item's.
const inclusionDetail = "The subject cannot be verified as belonging to the committed block."

// btcMerkleDetail explains a failure inside a Bitcoin block's own
// partial Merkle tree. It is a separate statement from inclusionDetail:
// these steps are about a transaction's place in a Bitcoin header, not a
// subject's place in a Truestamp block, and reports carrying them
// routinely carry a PASSING Inclusion Proof row at the same time.
const btcMerkleDetail = "The Bitcoin transaction cannot be tied to the block header the entry carries."

// failureDetails maps a keyword in a failure message to a plain-language
// explanation of the consequence.
//
// It is an ordered slice and not a map because lookup is a
// case-insensitive substring scan and many messages match more than one
// keyword — "Cannot verify proof signature (missing derived data): no
// block hash" matches two, "Bitcoin merkle proof root does NOT match
// cx.bmr" matches three. Go randomizes map iteration, so those messages
// drew a different detail from run to run. The first entry that matches
// wins, so the order is the disambiguation rule: most specific cause
// first, and the local derivations (merkle, block hash) ahead of the
// chain names, so a failed local walk over a Stellar commitment is not
// blamed on Horizon.
//
// Every entry must be reachable from a message the pipeline can actually
// emit, and must be true of every message that reaches it — a detail is
// asserted alongside the message, so a detail that overstates undoes a
// message that was carefully hedged. Both properties are pinned by
// TestFailureDetails_EveryKeywordIsReachable and
// TestLookupFailureDetail_ProductionMessages.
var failureDetails = []struct {
	keyword string
	detail  string
}{
	{"hash does not match", "The proof covers different data than what you provided."},
	// Before the generic signature entry: these arms are reached
	// because an EARLIER step could not produce an input, so nothing
	// about the signature — or about tampering — was established. The
	// generic detail below asserted tampering over messages whose whole
	// point was to say only that an input was missing.
	{"cannot verify proof signature", "An input this check needs was never derived, so the signature was neither confirmed nor refuted."},
	{"proof signature", "The proof may have been tampered with or signed with a different key."},
	{"keyring", "The published Truestamp keyring does not vouch for this signing key."},
	{"future-dated", "The claimed timestamp is after the submission time."},
	// E.20's ordering failure. The four entries that used to sit here
	// ("submitted after", "submitted before", "captured after",
	// "captured before") matched no message the pipeline emits, so this
	// failure rendered with no detail at all in either surface — and
	// their texts were inverted relative to their keys, describing "the
	// previous block", which the message never mentions.
	{"submission-window ordering violation", "The subject's own identifier is timestamped after the block that commits it, so the submission window does not hold."},
	// Bitcoin's own merkle steps, ahead of the generic merkle entry:
	// they are about a transaction's place in a Bitcoin block header,
	// not about a subject's place in a Truestamp block, and were being
	// explained as an inclusion-proof failure in reports whose
	// Inclusion Proof row was simultaneously passing.
	{"bitcoin partial merkle proof", btcMerkleDetail},
	{"bitcoin merkle", btcMerkleDetail},
	{"inclusion proof", inclusionDetail},
	{"epoch proof", "The block cannot be shown to map to the value committed on the public chain."},
	{"block hash", "The chain integrity cannot be confirmed."},
	{"stellar", "The Stellar blockchain could not confirm this transaction."},
	{"bitcoin", "The Bitcoin blockchain could not confirm this transaction."},
}

func lookupFailureDetail(msg string) string {
	lower := strings.ToLower(msg)
	for _, d := range failureDetails {
		if strings.Contains(lower, d.keyword) {
			return d.detail
		}
	}
	return ""
}

var categoryLabels = map[string]string{
	CatDataIntegrity: "Data Integrity",
	CatCryptographic: "Cryptographic",
	CatStructural:    "Structural",
	CatTiming:        "Timing",
	CatBlockchain:    "Blockchain",
}

func categoryDisplayName(cat string) string {
	if label, ok := categoryLabels[cat]; ok {
		return label
	}
	return cat
}

// --- Verification Summary ---

func renderVerificationSummary(r *Report) string {
	header := ui.SectionHeader("Verification Summary")

	c := r.Counts()
	passStyle := lipgloss.NewStyle().Foreground(ui.Green)
	warnStyle := lipgloss.NewStyle().Foreground(ui.Yellow)
	failStyle := lipgloss.NewStyle().Foreground(ui.Red)
	dimStyle := lipgloss.NewStyle().Foreground(ui.Dim)
	faint := ui.FaintStyle()

	// "steps", not "cryptographic checks": the denominator counts Hash
	// Comparison (data_integrity), Structure (structural), Submission
	// Window (timing) and the two commitment rows (blockchain) — four of
	// E.22's five categories are not cryptographic. It also counts info
	// rows now, so the line reconciles with the --json `steps` array.
	parts := []string{
		passStyle.Render(fmt.Sprintf("  %d of %d verification steps passed", c.Passed, c.Total)),
	}
	if c.Failed > 0 {
		parts = append(parts, failStyle.Render(fmt.Sprintf("  %d failed", c.Failed)))
	}
	if c.Warned > 0 {
		parts = append(parts, warnStyle.Render(fmt.Sprintf("  %d warning", c.Warned)))
	}
	if c.Skipped > 0 {
		parts = append(parts, dimStyle.Render(fmt.Sprintf("  %d skipped", c.Skipped)))
	}
	if c.Info > 0 {
		parts = append(parts, dimStyle.Render(fmt.Sprintf("  %d info", c.Info)))
	}
	checksLine := strings.Join(parts, "")

	// Verdict line. Read from Report.Verdict so the printed wording and
	// the process exit code (Report.Passed) cannot disagree.
	//
	// A passing verdict additionally discloses a run that never checked
	// the signature. E.25 does not list E.16 among the steps a verifier
	// MAY skip and still call a run verified, and without the
	// disclosure the strongest assertion this tool makes — "proof is
	// valid" — printed identically for a genuine bundle and for one
	// whose Ed25519 signature had been forged.
	skippedSigs := r.SignaturesSkipped()
	var verdict string
	switch r.Verdict() {
	case VerdictFailed:
		verdict = failStyle.Render("  FAILED") + faint.Render(" - proof verification failed")
	case VerdictHashMismatch:
		verdict = failStyle.Render("  HASH MISMATCH") + faint.Render(" - proof is valid but does not match your data")
	case VerdictFullyVerified:
		if skippedSigs {
			verdict = passStyle.Render("  VERIFIED") + faint.Render(" - matches your data, but the signature was NOT checked (--skip-signatures)")
		} else {
			verdict = passStyle.Render("  VERIFIED") + faint.Render(" - proof is valid and matches your data")
		}
	default:
		if skippedSigs {
			verdict = passStyle.Render("  VERIFIED") + faint.Render(" - but the signature was NOT checked (--skip-signatures)")
		} else {
			verdict = passStyle.Render("  VERIFIED") + faint.Render(" - proof is valid")
		}
	}

	lines := []string{header, "", verdict, "", checksLine}

	// Contextual notes
	if r.SkippedExternal {
		lines = append(lines, faint.Render("  Skipped external blockchain verification (--skip-external)"))
	}
	if skippedSigs {
		lines = append(lines, faint.Render("  Skipped Ed25519 signature verification (--skip-signatures); nothing here establishes who signed this proof"))
	}

	return strings.Join(lines, "\n")
}

// --- Helpers ---

func metadataStyleFunc(row, col int) lipgloss.Style {
	if col == 0 {
		return lipgloss.NewStyle().
			Foreground(ui.Label).
			PaddingLeft(2).
			Align(lipgloss.Right).
			PaddingRight(1)
	}
	return lipgloss.NewStyle().Foreground(ui.Value)
}

// truncateToSecond is a package-local alias for ui.TruncateToSecond so
// existing call sites here don't need to thread the ui import. The
// shared helper in internal/ui is the single source of truth; this
// wrapper exists only for readability.
func truncateToSecond(ts string) string { return ui.TruncateToSecond(ts) }
