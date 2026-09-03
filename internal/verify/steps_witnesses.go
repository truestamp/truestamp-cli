// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: MIT

package verify

import (
	"encoding/json"
	"fmt"
	"sort"

	"github.com/truestamp/truestamp-cli/internal/jcs"
	"github.com/truestamp/truestamp-cli/internal/proof"
	"github.com/truestamp/truestamp-cli/internal/tscrypto"
)

// --- [E.17a] Step 10: witnesses ---
//
// A witness is a public record that existed before the submission and that
// the item's composite fingerprint commits to. The item's metadata is a map
// of witness name to committed hash, and that map is hashed into
// metadata_hash and from there into the composite fingerprint, so the
// choice of witnesses was frozen at submission and cannot be revised later.
// What the bundle optionally carries alongside it is the DETAIL of each
// witness: the head block map, or an entropy observation's captured
// payload. A verifier recomputes the detail's hash and compares.
//
// Extensibility is the point of the shape: the metadata map is hashed
// exactly as carried, so an unknown name inside it never breaks the
// fingerprint and is reported as a skip; an unknown name inside
// subject.witnesses is reported the same way; a committed name whose
// detail is not carried is a skip, not a failure.

// witnessEntry is what E.20 needs from a carried witness whose detail was
// hashed: whether it matched the committed hash, the time its source
// published, and the basis of that time.
type witnessEntry struct {
	Name        string
	Matched     bool
	PublishedMs int64
	HasTime     bool
	Basis       string
	Payload     json.RawMessage
}

// witnessSet is the carried witnesses by name.
type witnessSet map[string]*witnessEntry

// sortedNames returns the entries' names in lexicographic order.
func (w witnessSet) sortedNames() []string {
	names := make([]string, 0, len(w))
	for n := range w {
		names = append(names, n)
	}
	sort.Strings(names)
	return names
}

func stepWitnesses(r *Report, bundle *proof.Bundle) witnessSet {
	witnesses := witnessSet{}
	if bundle.IsBlockLike() {
		r.skip(groupWitnesses, CatTiming,
			"not applicable: a block-like subject carries no submission fingerprint")
		return witnesses
	}

	subject := bundle.Subject
	committed := subject.CommittedWitnesses()
	carried := subject.Witnesses

	seen := map[string]bool{}
	var names []string
	for _, n := range committed.Keys() {
		if !seen[n] {
			seen[n] = true
			names = append(names, n)
		}
	}
	for _, n := range carried.Keys() {
		if !seen[n] {
			seen[n] = true
			names = append(names, n)
		}
	}
	sort.Strings(names)

	if len(names) == 0 {
		r.skip(groupWitnesses, CatTiming,
			"no witnesses: this subject's metadata commits to none and none is carried")
		return witnesses
	}

	for _, name := range names {
		committedHash, isCommitted := committed[name]
		detail, isCarried := carried[name]
		switch {
		case !proof.IsCommittedWitnessName(name):
			r.skip(groupWitnesses, CatTiming,
				fmt.Sprintf("witness type %s not recognized by this verifier", name))
		case !isCarried || !fieldCarried(detail):
			r.skip(groupWitnesses, CatTiming,
				fmt.Sprintf("%s committed in the subject metadata but not carried in this bundle", name))
		case !isCommitted || !fieldCarried(committedHash):
			r.fail(groupWitnesses, CatTiming, fmt.Sprintf(
				"witness_hash_mismatch: a %s witness detail is carried but the subject metadata commits to no %s witness", name, name))
		case name == proof.WitnessBlock:
			verifyBlockWitness(r, witnesses, detail, committedString(committed, name), bundle)
		default:
			verifyEntropyWitness(r, witnesses, name, detail, committedString(committed, name), bundle)
		}
	}
	return witnesses
}

// committedString renders a committed witness value the way the reference
// verifier's to_string/1 does: the string itself, or the literal of a
// non-string value, which then compares unequal to any hash.
func committedString(committed proof.Object, name string) string {
	if s, ok := committed.String(name); ok {
		return s
	}
	return committed.Literal(name)
}

func fieldCarried(raw json.RawMessage) bool {
	return len(raw) > 0 && string(raw) != "null"
}

// verifyBlockWitness recomputes the head block's hash, which proves the
// block the item's fingerprint named; the chain link (a direct predecessor,
// or a walked block_path) is what places the item at a position in the
// ledger.
func verifyBlockWitness(r *Report, witnesses witnessSet, detail json.RawMessage, committedHash string, bundle *proof.Bundle) {
	head := parseBlockMap(detail)
	headHash, _, reason := blockHashFromMap(head)
	if reason != "" {
		r.fail(groupWitnesses, CatTiming,
			"witness_hash_mismatch: cannot recompute the block witness hash: "+reason)
		return
	}
	matched := r.check(groupWitnesses, CatTiming,
		tscrypto.SecureEqual(headHash, committedHash),
		fmt.Sprintf("block witness recomputes to the hash the subject metadata commits to (head block %s)", head.ID),
		"witness_hash_mismatch: the block witness does not recompute to the hash the subject metadata commits to")
	checkBlockChainLink(r, headHash, bundle)

	entry := &witnessEntry{Name: proof.WitnessBlock, Matched: matched, Basis: proof.WitnessBasis(proof.WitnessBlock), Payload: detail}
	if ms, ok := uuidv7Ms(head.ID); ok {
		entry.PublishedMs, entry.HasTime = ms, true
	}
	witnesses[proof.WitnessBlock] = entry
}

// parseBlockMap reads a witness detail as a block map through the proof
// package's own reader, so the four block-map positions share one shape.
func parseBlockMap(detail json.RawMessage) proof.BlockMap {
	var b proof.Bundle
	b.Fields = proof.Object{"block": detail}
	obj, ok := b.Fields.Object("block")
	if !ok {
		return proof.BlockMap{}
	}
	out := proof.BlockMap{
		Fields:            obj,
		ID:                obj.Str("id"),
		PreviousBlockHash: obj.Str("previous_block_hash"),
		MerkleRoot:        obj.Str("merkle_root"),
		SigningKeyID:      obj.Str("signing_key_id"),
	}
	if meta, isMap := obj.Object("metadata"); isMap && meta != nil {
		out.Metadata = obj.Raw("metadata")
	}
	return out
}

// checkBlockChainLink reports exactly one of: the head block is the
// containing block's direct predecessor; a carried block_path walks from
// the head block to the containing block; or the bundle asserts a ledger
// position it does not evidence.
func checkBlockChainLink(r *Report, headHash string, bundle *proof.Bundle) {
	switch {
	case tscrypto.SecureEqual(bundle.Block.PreviousBlockHash, headHash):
		r.pass(groupWitnesses, CatTiming,
			"ledger position: the head block is the containing block's direct predecessor")
	case len(bundle.BlockPath) > 0:
		walkBlockPath(r, headHash, bundle)
	default:
		r.fail(groupWitnesses, CatTiming,
			"block_path_broken: the head block is not the containing block's predecessor and no block_path is carried")
	}
}

// walkBlockPath walks the path oldest first: the first entry's
// previous_block_hash is the head block hash, each later entry's is the
// hash of the entry before it, and the containing block's is the hash of
// the last entry.
func walkBlockPath(r *Report, headHash string, bundle *proof.Bundle) {
	expected := headHash
	for _, entry := range bundle.BlockPath {
		if !entry.IsMap() || !tscrypto.SecureEqual(entry.PreviousBlockHash, expected) {
			r.fail(groupWitnesses, CatTiming,
				"block_path_broken: the block path does not link the head block to the containing block")
			return
		}
		hash, _, reason := blockHashFromMap(entry)
		if reason != "" {
			r.fail(groupWitnesses, CatTiming,
				"block_path_broken: the block path does not link the head block to the containing block")
			return
		}
		expected = hash
	}
	r.check(groupWitnesses, CatTiming,
		tscrypto.SecureEqual(bundle.Block.PreviousBlockHash, expected),
		fmt.Sprintf("ledger position: a block path of %s links the head block to the containing block", blocksLabel(len(bundle.BlockPath))),
		"block_path_broken: the block path does not reach the containing block's previous_block_hash")
}

// verifyEntropyWitness hashes the captured payload of an observation under
// the same 0x21 prefix an entropy subject uses. No new byte prefix was
// introduced for witnesses. The payload travels exactly as captured and is
// never reshaped: reshaping it would change its JCS encoding and therefore
// its hash.
func verifyEntropyWitness(r *Report, witnesses witnessSet, name string, detail json.RawMessage, committedHash string, bundle *proof.Bundle) {
	payload, ok := detailObject(detail)
	if !ok {
		r.fail(groupWitnesses, CatTiming,
			fmt.Sprintf("witness_hash_mismatch: the %s witness detail is not a map, so no hash can be derived", name))
		return
	}
	canonical, _, err := jcs.Canonicalize(detail)
	if err != nil {
		r.fail(groupWitnesses, CatTiming,
			fmt.Sprintf("witness_hash_mismatch: cannot recompute the %s witness hash: %s", name, err))
		return
	}
	witnessHash := tscrypto.ComputeEntropyHash(canonical)
	matched := r.check(groupWitnesses, CatTiming,
		tscrypto.SecureEqual(witnessHash, committedHash),
		fmt.Sprintf("%s witness recomputes to the hash the subject metadata commits to (0x21)", name),
		fmt.Sprintf("witness_hash_mismatch: the %s witness does not recompute to the hash the subject metadata commits to", name))

	entry := &witnessEntry{Name: name, Matched: matched, Basis: proof.WitnessBasis(name), Payload: detail}
	if ms, ok := witnessPublishedMs(name, payload); ok {
		entry.PublishedMs, entry.HasTime = ms, true
	}
	witnesses[name] = entry

	// No published time is read out of a detail whose hash did not match,
	// so the ordering warning applies only to a matched witness.
	if matched && entry.HasTime {
		if submittedMs, ok := ulidMs(bundle.Subject.ID); ok && entry.PublishedMs > submittedMs {
			r.warn(groupWitnesses, CatTiming, fmt.Sprintf(
				"%s witness was published at %s, after the subject's own submission time %s",
				name, formatMs(entry.PublishedMs), formatMs(submittedMs)))
		}
	}
}

func detailObject(detail json.RawMessage) (proof.Object, bool) {
	holder := proof.Object{"d": detail}
	return holder.Object("d")
}

// witnessPublishedMs reads the time each source publishes, from the field
// Truestamp's capture adapters read: the Stellar ledger closed_at, the NIST
// pulse timeStamp, the Bitcoin header time.
func witnessPublishedMs(name string, detail proof.Object) (int64, bool) {
	switch name {
	case proof.WitnessEntropyStellar:
		ledger := detail
		if nested, ok := detail.Object("ledger"); ok {
			ledger = nested
		}
		if closedAt, ok := ledger.String("closed_at"); ok {
			return isoMs(closedAt)
		}
	case proof.WitnessEntropyNIST:
		if pulse, ok := detail.Object("pulse"); ok {
			if ts, ok := pulse.String("timeStamp"); ok {
				return isoMs(ts)
			}
		}
	case proof.WitnessEntropyBitcoin:
		block := detail
		if nested, ok := detail.Object("block"); ok {
			block = nested
		}
		if seconds, ok := block.Integer("time"); ok && seconds.Fits {
			return seconds.N * 1000, true
		}
	}
	return 0, false
}
