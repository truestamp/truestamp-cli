// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: MIT

package verify

import (
	"fmt"
	"strings"

	"github.com/truestamp/truestamp-cli/internal/proof"
)

// --- [E.20] Step 12: the submission window ---
//
// A bundle alone yields exactly one timing fact of the verifier's own: the
// milliseconds embedded in subject.id and block.id. Both ids were minted by
// Truestamp, so their ordering is something Truestamp ASSERTS. It is worth
// checking, and a violation is a real failure, but it is not external
// evidence and must never be presented as such.
//
// The SUBMITTED-BEFORE edge is grounded by the earliest commitment whose
// chain confirmed in this run; unconfirmed, its timestamp is a candidate
// rather than an established edge.
//
// The SUBMITTED-AFTER edge is grounded by the witnesses the fingerprint
// commits to. Each carried witness names the time its own source published
// it, and the operative edge is the latest such time among the witnesses
// confirmed against their sources in this run. Unconfirmed, the edge rests
// on the fingerprint alone: still meaningful, because a witness hash was
// frozen into the item hash at submission, and still short of an
// externally confirmed edge.

func stepSubmissionWindow(r *Report, bundle *proof.Bundle, witnesses witnessSet, confirmedChains, confirmedWitnesses map[string]bool) {
	if bundle.IsBlockLike() {
		r.skip(groupSubmittedAfter, CatTiming,
			"not established: a block-like proof carries no submission fingerprint")
		if ms, ok := uuidv7Ms(bundle.Block.ID); ok {
			r.Temporal.CommittedAt = formatSeconds(ms)
		}
		reportSubmittedBefore(r, bundle, confirmedChains)
		return
	}

	var subjectMs int64
	var subjectOK bool
	if bundle.IsItem() {
		subjectMs, subjectOK = ulidMs(bundle.Subject.ID)
	} else {
		subjectMs, subjectOK = uuidv7Ms(bundle.Subject.ID)
	}
	blockMs, blockOK := uuidv7Ms(bundle.Block.ID)
	if subjectOK && blockOK {
		r.check(groupSubmissionWindow, CatTiming, subjectMs <= blockMs,
			fmt.Sprintf("subject id time %s is at or before block id time %s (asserted by Truestamp, not externally verified)",
				formatMs(subjectMs), formatMs(blockMs)),
			fmt.Sprintf("submission-window ordering violation: subject id time %s is AFTER block id time %s",
				formatMs(subjectMs), formatMs(blockMs)))
		r.info(groupTemporalInfo, CatTiming,
			fmt.Sprintf("submitted %s, committed into a block %s", formatMs(subjectMs), formatMs(blockMs)))
		r.Temporal.SubmittedAt = formatSeconds(subjectMs)
		r.Temporal.CommittedAt = formatSeconds(blockMs)
	}

	reportSubmittedAfter(r, bundle, witnesses, confirmedWitnesses)
	reportSubmittedBefore(r, bundle, confirmedChains)
}

func reportSubmittedAfter(r *Report, bundle *proof.Bundle, witnesses witnessSet, confirmed map[string]bool) {
	if len(witnesses) == 0 {
		committed := bundle.Subject.CommittedWitnesses().Keys()
		if len(committed) == 0 {
			r.info(groupSubmittedAfter, CatTiming,
				"not established from this bundle: this subject's metadata commits to no witness")
		} else {
			r.info(groupSubmittedAfter, CatTiming,
				"not established from this bundle: no witness details carried; the metadata commits to "+strings.Join(committed, ", "))
		}
		return
	}

	for _, name := range witnesses.sortedNames() {
		r.info(groupSubmittedAfter, CatTiming, witnessEdgeMessage(witnesses[name]))
	}

	// Only a witness whose hash matches the one the fingerprint commits to
	// can carry the edge: an unmatched detail is unbound caller data.
	var usable, confirmedUsable []*witnessEntry
	for _, name := range witnesses.sortedNames() {
		e := witnesses[name]
		if !e.Matched || !e.HasTime {
			continue
		}
		usable = append(usable, e)
		if confirmed[name] {
			confirmedUsable = append(confirmedUsable, e)
		}
	}

	if latest := latestWitness(confirmedUsable); latest != nil {
		r.pass(groupSubmittedAfter, CatTiming, fmt.Sprintf(
			"Submitted after %s: the latest confirmed source is %s (%s)",
			formatMs(latest.PublishedMs), latest.Name, latest.Basis))
		return
	}
	if latest := latestWitness(usable); latest != nil {
		r.info(groupSubmittedAfter, CatTiming, fmt.Sprintf(
			"submitted after %s rests on the %s witness the fingerprint commits to; no carried witness was confirmed against its source in this run (%s)",
			formatMs(latest.PublishedMs), latest.Name, latest.Basis))
		return
	}
	r.info(groupSubmittedAfter, CatTiming,
		"not established: no carried witness yielded a source-published time")
}

func witnessEdgeMessage(e *witnessEntry) string {
	switch {
	case !e.Matched:
		return fmt.Sprintf("%s: not counted, its hash does not match the one the fingerprint commits to (%s)", e.Name, e.Basis)
	case e.HasTime:
		return fmt.Sprintf("%s: source published %s (%s)", e.Name, formatMs(e.PublishedMs), e.Basis)
	default:
		return fmt.Sprintf("%s: no source-published time could be read from the carried detail (%s)", e.Name, e.Basis)
	}
}

func latestWitness(entries []*witnessEntry) *witnessEntry {
	var latest *witnessEntry
	for _, e := range entries {
		if latest == nil || e.PublishedMs > latest.PublishedMs {
			latest = e
		}
	}
	return latest
}

// timestampedCommitment is a commitment entry whose timestamp parsed.
type timestampedCommitment struct {
	chain     string
	timestamp string
	ms        int64
}

func reportSubmittedBefore(r *Report, bundle *proof.Bundle, confirmedChains map[string]bool) {
	var timestamped, confirmed []timestampedCommitment
	for _, c := range bundle.Commitments {
		if c.Timestamp == "" {
			continue
		}
		ms, ok := isoMs(c.Timestamp)
		if !ok {
			continue
		}
		tc := timestampedCommitment{chain: c.Chain, timestamp: c.Timestamp, ms: ms}
		timestamped = append(timestamped, tc)
		if confirmedChains[c.Chain] {
			confirmed = append(confirmed, tc)
		}
	}

	switch {
	case len(timestamped) == 0:
		r.info(groupSubmittedBefore, CatTiming, "not established: no commitment carries a timestamp")
	case len(confirmed) == 0:
		earliest := earliestCommitment(timestamped)
		r.info(groupSubmittedBefore, CatTiming, fmt.Sprintf(
			"submitted before %s is a candidate from the %s commitment; that commitment was not confirmed in this run",
			earliest.timestamp, earliest.chain))
	default:
		edge := earliestCommitment(confirmed)
		r.pass(groupSubmittedBefore, CatTiming, fmt.Sprintf(
			"Submitted before %s: the %s commitment carrying the earliest confirmed timestamp is confirmed on chain",
			edge.timestamp, edge.chain))
		// An unconfirmed commitment timestamped ahead of the confirmed edge
		// is named but never displaces it.
		var earlier []timestampedCommitment
		for _, tc := range timestamped {
			if tc.ms < edge.ms {
				earlier = append(earlier, tc)
			}
		}
		if len(earlier) > 0 {
			candidate := earliestCommitment(earlier)
			r.info(groupSubmittedBefore, CatTiming, fmt.Sprintf(
				"An earlier candidate, %s from the %s commitment, was not confirmed in this run; confirming it would tighten the edge",
				candidate.timestamp, candidate.chain))
		}
	}
}

func earliestCommitment(entries []timestampedCommitment) timestampedCommitment {
	earliest := entries[0]
	for _, e := range entries[1:] {
		if e.ms < earliest.ms {
			earliest = e
		}
	}
	return earliest
}
