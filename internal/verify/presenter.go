package verify

import (
	"bytes"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	lipgloss "charm.land/lipgloss/v2"
	"charm.land/lipgloss/v2/table"
	"github.com/truestamp/truestamp-cli/internal/ui"
)

// Present renders a Report to stdout with lipgloss styling.
func Present(r *Report) {
	sections := []string{
		renderResultBanner(r),
		"",
		renderProofSection(r),
		"",
		renderSubject(r),
	}

	// Hash mismatch detail (between Subject and Timeline)
	if r.HashProvided != "" && !r.HashMatched() && r.ProofPassed() {
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

	// Issues (only when there are failures or warnings)
	if issues := renderIssues(r); issues != "" {
		sections = append(sections, "", issues)
	}

	// Verification Summary (replaces old raw counts line)
	sections = append(sections, "", renderVerificationSummary(r))

	lipgloss.Println(lipgloss.JoinVertical(lipgloss.Left, sections...))
}

// --- Result Banner ---

func renderResultBanner(_ *Report) string {
	return ui.AccentBoldStyle().Render("  Truestamp Proof Verification")
}

// --- Proof Section ---

func renderProofSection(r *Report) string {
	header := ui.SectionHeader("Proof")

	tbl := table.New().
		Border(lipgloss.HiddenBorder()).
		StyleFunc(metadataStyleFunc)

	tbl = tbl.Row("ID", r.SubjectID)

	if r.SubjectType == "entropy" {
		tbl = tbl.Row("Type", "Entropy")
	} else {
		tbl = tbl.Row("Type", "Item")
	}

	return header + "\n" + tbl.String()
}

// --- Subject Section ---

func renderSubject(r *Report) string {
	if r.SubjectType == "entropy" {
		return renderEntropySubject(r)
	}
	return renderItemSubject(r)
}

func renderItemSubject(r *Report) string {
	header := ui.SectionHeader("Item Claims")
	subtitle := ui.FaintStyle().Render("  Claims made by the submitter. Not independently verified by Truestamp.")

	tbl := table.New().
		Border(lipgloss.HiddenBorder()).
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

	tbl := table.New().
		Border(lipgloss.HiddenBorder()).
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
	case "nist_beacon":
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

	case "bitcoin_block":
		if es.BlockHeight > 0 {
			tbl = tbl.Row("Block", fmt.Sprintf("%d", es.BlockHeight))
		}
		if es.BlockHash != "" {
			tbl = tbl.Row("Block Hash", es.BlockHash)
		}

	case "stellar_ledger":
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

	tbl := table.New().
		Border(lipgloss.HiddenBorder()).
		StyleFunc(metadataStyleFunc)

	if t.ClaimedAt != "" {
		row := t.ClaimedAt
		if r.Claims.TimestampStatus == TimestampStale {
			row = lipgloss.NewStyle().Foreground(ui.Yellow).Render(t.ClaimedAt) +
				lipgloss.NewStyle().Foreground(ui.Yellow).Render(" ! "+r.Claims.TimestampNote)
		} else if r.Claims.TimestampStatus == TimestampFuture {
			row = lipgloss.NewStyle().Foreground(ui.Red).Render(t.ClaimedAt) +
				lipgloss.NewStyle().Foreground(ui.Red).Render(" x "+r.Claims.TimestampNote)
		}
		tbl = tbl.Row("Claimed at", row)
	}
	if t.SubmittedAt != "" {
		tbl = tbl.Row("Submitted at", t.SubmittedAt)
	}
	if t.CapturedAt != "" {
		tbl = tbl.Row("Captured at", t.CapturedAt)
	}
	if t.CommittedAt != "" {
		tbl = tbl.Row("Committed at", t.CommittedAt)
	}

	return header + "\n" + tbl.String()
}

// --- Commitments Section ---

func renderCommitments(r *Report) string {
	if r.ChainLength == 0 {
		return ""
	}

	header := ui.SectionHeader("Commitments")

	tbl := table.New().
		Border(lipgloss.HiddenBorder()).
		StyleFunc(metadataStyleFunc)

	// Truestamp chain info
	tbl = tbl.Row("Truestamp", fmt.Sprintf("%d-block chain, signed with key %s", r.ChainLength, r.SigningKeyID))

	dim := lipgloss.NewStyle().Foreground(ui.Dim)

	// Blockchain commitments
	for _, ci := range r.CommitmentInfos {
		switch ci.Method {
		case "stellar":
			label := "Stellar"
			if ci.Skipped {
				tbl = tbl.Row(label, fmt.Sprintf("Ledger %d on %s (external verification skipped)", ci.Ledger, ci.Network))
			} else {
				ts := formatCommitmentTimestamp(ci.Timestamp)
				tbl = tbl.Row(label, fmt.Sprintf("Ledger %d on %s (%s)", ci.Ledger, ci.Network, ts))
			}
			tbl = tbl.Row("", dim.Render("tx: "+ci.TxHash))
			if ci.CommittedHash != "" {
				tbl = tbl.Row("", dim.Render("memo_hash (hex): "+ci.CommittedHash))
				tbl = tbl.Row("", dim.Render("memo_hash (base64): "+HexToBase64(ci.CommittedHash)))
			}
		case "bitcoin":
			label := "Bitcoin"
			if ci.Skipped {
				tbl = tbl.Row(label, fmt.Sprintf("Block %d on %s (external verification skipped)", ci.Height, ci.Network))
			} else {
				ts := formatCommitmentTimestamp(ci.Timestamp)
				tbl = tbl.Row(label, fmt.Sprintf("Block %d on %s (%s)", ci.Height, ci.Network, ts))
			}
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

// --- Hash Mismatch Detail ---

func renderHashMismatchDetail(r *Report) string {
	tbl := table.New().
		Border(lipgloss.HiddenBorder()).
		StyleFunc(metadataStyleFunc)

	tbl = tbl.Row("Expected", truncateHash(r.HashProvided)+
		lipgloss.NewStyle().Foreground(ui.Dim).Render(" (from --hash)"))
	tbl = tbl.Row("Found", truncateHash(r.Claims.Hash)+
		lipgloss.NewStyle().Foreground(ui.Dim).Render(" (in proof)"))

	return tbl.String()
}

// --- Issues Section ---

func renderIssues(r *Report) string {
	// Collect all non-passing steps (failures, warnings, skipped, info)
	var issues []Step
	for _, s := range r.Steps {
		if s.Status != StatusPass {
			issues = append(issues, s)
		}
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
			order, exists := CategoryOrder[cat]
			if !exists {
				order = 99
			}
			catMap[cat] = &catGroup{name: cat, order: order}
		}
		catMap[cat].steps = append(catMap[cat].steps, s)
	}

	// Sort categories by order
	var cats []*catGroup
	for _, g := range catMap {
		cats = append(cats, g)
	}
	sort.Slice(cats, func(i, j int) bool { return cats[i].order < cats[j].order })

	// Sort steps within each category: failures first, then warnings
	for _, g := range cats {
		sort.SliceStable(g.steps, func(i, j int) bool {
			return g.steps[i].Status < g.steps[j].Status // StatusFail(1) before StatusWarn(3)
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
			case StatusSkip:
				line = skipStyle.Render("    - " + s.Message)
			case StatusInfo:
				line = skipStyle.Render("    i " + s.Message)
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

// failureDetails maps message keywords to user-friendly explanations.
var failureDetails = map[string]string{
	"proof signature":     "The proof may have been tampered with or signed with a different key.",
	"merkle proof":        "The item cannot be verified as belonging to the committed block.",
	"block hash":          "The chain integrity cannot be confirmed.",
	"chain link":          "The blocks are not properly connected.",
	"keyring":             "The keyring confirms the signing key is a trusted Truestamp key.",
	"hash does not match": "The proof covers different data than what you provided.",
	"stellar":             "The Stellar blockchain could not confirm this transaction.",
	"bitcoin":             "The Bitcoin blockchain could not confirm this transaction.",
	"temporal ordering":   "Block timestamps are not in ascending order.",
	"submitted after":     "The item appears before the previous block was created.",
	"submitted before":    "The item appears after the committed block was created.",
	"captured after":      "The entropy observation appears before the previous block was created.",
	"captured before":     "The entropy observation appears after the committed block was created.",
	"future-dated":        "The claimed timestamp is after the submission time.",
}

func lookupFailureDetail(msg string) string {
	lower := strings.ToLower(msg)
	for keyword, detail := range failureDetails {
		if strings.Contains(lower, keyword) {
			return detail
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

	parts := []string{
		passStyle.Render(fmt.Sprintf("  %d of %d cryptographic checks passed", c.Passed, c.Total)),
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
	checksLine := strings.Join(parts, "")

	// Verdict line
	proofOK := r.ProofPassed()
	hashProvided := r.HashProvided != ""
	hashOK := r.HashMatched()

	var verdict string
	switch {
	case !proofOK:
		verdict = failStyle.Render("  FAILED") + faint.Render(" - proof verification failed")
	case hashProvided && !hashOK:
		verdict = failStyle.Render("  HASH MISMATCH") + faint.Render(" - proof is valid but does not match your data")
	case hashProvided && hashOK:
		verdict = passStyle.Render("  VERIFIED") + faint.Render(" - proof is valid and matches your data")
	default:
		verdict = passStyle.Render("  VERIFIED") + faint.Render(" - proof is valid")
	}

	lines := []string{header, "", verdict, "", checksLine}

	// Contextual notes
	if r.SkippedExternal {
		lines = append(lines, faint.Render("  Skipped external blockchain verification (--skip-external)"))
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

// truncateHash shows first 8 and last 8 chars of a hex hash with ellipsis.
func truncateHash(hash string) string {
	if len(hash) <= 20 {
		return hash
	}
	return hash[:8] + "..." + hash[len(hash)-8:]
}

// truncateToSecond parses an RFC3339 timestamp and truncates to second precision.
func truncateToSecond(ts string) string {
	t, err := time.Parse(time.RFC3339Nano, ts)
	if err != nil {
		return ts
	}
	return t.Truncate(time.Second).Format(time.RFC3339)
}
