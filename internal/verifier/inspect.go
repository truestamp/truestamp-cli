// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: Apache-2.0

// This file holds what `truestamp inspect` reads of a proof, which shows
// what a proof carries and the values this verifier recomputes from it,
// and verifies nothing: it makes no step past SHAPE, checks no key and no
// signature, reads no chain or source and gives no label (SPEC.md section
// 10.5). [Inspect] reads the document as `truestamp verify` does, through
// the same functions:
//
//  | Step     | What inspect takes from it                                                  |
//  |----------|-----------------------------------------------------------------------------|
//  | READ, PARSE, ENVELOPE | the bare rejection of SPEC.md section 10.2 (the gate), when one fails, and nothing else |
//  | SHAPE    | its failing row, when it fails (section 10.2, step 7); its type, subject id and constructions mode when it passes |
//  | DERIVE   | when SHAPE passes, the values of its traced checks (section 10.6): each one's declared and recomputed value, never its status |
//
// SHAPE runs here on a proof whose key and signature are not checked,
// which a verification never does (section 10.2, the gate), and DERIVE's
// checks are made only for the values they recompute: inspect gives no
// verdict, so nothing it shows is a result of verification.
//
// It computes proof_hash, SHA-256(0x50 || JCS(P)) (section 6.1), and K's
// key id (section 3.3) once ENVELOPE passes, so they are given for a proof
// SHAPE refuses as well. Once SHAPE passes, it gives each member P carries
// once, by its value class (section 3.7):
//
//  | Class                                          | Where inspect gives it                                                         |
//  |------------------------------------------------|--------------------------------------------------------------------------------|
//  | CHECKED                                        | Recomputed: the value carried beside the value DERIVE recomputes for it, one for each traced check, and a Bitcoin header member with no recomputed value when the txoutproof does not parse |
//  | a time, of any class                           | DeclaredTimes, as declared (section 11.5)                                     |
//  | RECORDED                                       | Recorded, which a surface shows under the heading of a value not attributed to Truestamp |
//  | SUBMITTED                                      | ClaimsMembers: the names of the claims, never their values                    |
//  | PINNED, PREIMAGE, CHAIN POINTER, BLOCK HEIGHT, PATH INPUT | Carried, as P holds them                                           |
//
// Carried is P without every member given elsewhere, and without the
// claims and the constructions member, which ClaimsMembers and
// Constructions give. Each time P carries is among DeclaredTimes and in no
// other place, a Bitcoin commitment's header_time included, which is
// CHECKED: no time a proof carries is an edge of the submission window
// (SPEC.md section 11), and a surface shows each only as declared (section
// 11.5).

package verifier

import (
	"encoding/hex"
	"fmt"
	"slices"

	"github.com/truestamp/truestamp-cli/internal/registry"
)

// Inspection is a document as `truestamp inspect` reads it (see the
// file's doc).
type Inspection struct {
	// Rejection is the bare rejection of READ, PARSE or ENVELOPE, and nil
	// when they pass; every other member is then zero.
	Rejection *Rejection
	// Refusal is SHAPE's failing row, and nil when SHAPE passes.
	Refusal *Row
	// Document is the parsed document and Proof is P, its proof member;
	// set when ENVELOPE passes.
	Document map[string]any
	Proof    map[string]any
	// KeyID is the key id of K and ProofHash is proof_hash as hex, both
	// computed here; set when ENVELOPE passes.
	KeyID     string
	ProofHash string
	// Type, SubjectID and Constructions are what SHAPE gave the run; set
	// when SHAPE passes.
	Type          string
	SubjectID     string
	Constructions Constructions
	// Carried, ClaimsMembers, Recomputed, DeclaredTimes and Recorded are
	// the members of P by their classes (see the file's doc), each list in
	// the document order of its parts; set when SHAPE passes.
	Carried       map[string]any
	ClaimsMembers []string
	Recomputed    []Recomputed
	DeclaredTimes []DeclaredTime
	Recorded      []Recorded
}

// Recomputed is one CHECKED value of P: the row of the constructions text
// that defines it (SPEC.md section 7.3), its pointer, the value P carries
// there, and the value DERIVE recomputes for it, nil when there is none
// (a walk that gives no root, or a header of a txoutproof that does not
// parse). For a handover_signature, SignedMessage is true: Carried is the
// signature and Value is not a value to compare with it but the 32 octets
// of the message it signs, as hex.
type Recomputed struct {
	Construction  string
	Part          string
	Carried       any
	Value         any
	SignedMessage bool
}

// Recorded is one RECORDED value of P (SPEC.md section 3.7): its pointer
// and its value.
type Recorded struct {
	Part  string
	Value any
}

// The classes of a time a proof carries (SPEC.md sections 2.4 and 3.5).
const (
	DeclaredTIME     = "TIME"
	DeclaredNISTTIME = "NISTTIME"
	// DeclaredUnixSeconds is a Bitcoin entropy object's time, an INT
	// counting seconds.
	DeclaredUnixSeconds = "INT"
)

// DeclaredTime is one time P carries: its pointer, its class and the
// value as P carries it, a string or a strictjson.Number.
type DeclaredTime struct {
	Part  string
	Class string
	Value any
}

// carriedMembers are the members of P that Carried may hold: SPEC.md
// section 3.2's, but the constructions member, which SHAPE alone reads.
var carriedMembers = []string{
	"block", "commitments", "environment", "format", "observations", "signing_key_event", "subject", "type", "version", "witnesses",
}

// recordedKeyEventMembers are the members of a key event that are
// RECORDED wherever they appear (SPEC.md section 3.7).
var recordedKeyEventMembers = []string{"prerotation_commitment", "previous_public_key", "reason", "revoked_key_id", "sequence"}

// keyIntroducingEvents are the key events under which a block record's
// signing_key_id is CHECKED (SPEC.md section 3.7): those that introduce a
// key, and a prerotation.
var keyIntroducingEvents = []string{"genesis", "rotation", "emergency_rotation", "prerotation"}

// bitcoinHeaderMembers are the CHECKED members of a Bitcoin commitment
// entry that only a txoutproof that parses gives a recomputed value for,
// but the header_time, which is among the declared times.
var bitcoinHeaderMembers = []string{"header_hash", "header_merkle_root"}

// Inspect reads data as `truestamp inspect` does (see the file's doc). It
// gives an error only for a fault: a proof_hash or a DERIVE value that
// cannot be computed over a value the strict reader and SHAPE passed.
func Inspect(data []byte) (*Inspection, error) {
	run := &Run{}
	if _, rejection, _ := afterConfiguration(run, data); rejection != nil {
		return &Inspection{Rejection: rejection}, nil
	}
	proofHash, err := registry.RecordHash(registry.ProofHash, run.Proof)
	if err != nil {
		return nil, fmt.Errorf("the proof_hash of this proof cannot be computed: %w", err)
	}
	document, _ := run.Document.(map[string]any)
	out := &Inspection{
		Document:  document,
		Proof:     run.Proof,
		KeyID:     registry.KeyIDOf(run.PublicKey),
		ProofHash: hex.EncodeToString(proofHash[:]),
	}
	Shape(run)
	if run.Type == "" {
		for _, row := range run.Rows {
			if row.Step == "SHAPE" {
				out.Refusal = &row
				break
			}
		}
		return out, nil
	}
	out.Type, out.SubjectID, out.Constructions = run.Type, run.SubjectID, run.Constructions
	out.DeclaredTimes = declaredTimes(run.Proof)
	given := map[string]bool{}
	for _, d := range out.DeclaredTimes {
		given[d.Part] = true
	}
	if out.Recomputed, err = recomputedValues(run.Proof, run.PublicKey, given); err != nil {
		return nil, err
	}
	for _, r := range out.Recomputed {
		given[r.Part] = true
	}
	out.Recorded = recordedValues(run.Proof)
	for _, r := range out.Recorded {
		given[r.Part] = true
	}
	claimsAt := Pointer{Name("subject"), Name("claims")}
	if claims, ok := valueAt(run.Proof, claimsAt).(map[string]any); ok && run.Type == "item" {
		given[claimsAt.Encode()] = true
		out.ClaimsMembers = []string{}
		for name := range claims {
			out.ClaimsMembers = append(out.ClaimsMembers, name)
		}
		SortBy(out.ClaimsMembers, func(name string) Pointer { return Pointer{Name(name)} })
	}
	out.Carried = map[string]any{}
	for _, name := range carriedMembers {
		if value, ok := run.Proof[name]; ok {
			if kept, keep := carriedValue(value, Pointer{Name(name)}, given); keep {
				out.Carried[name] = kept
			}
		}
	}
	return out, nil
}

// recomputedValues are P's CHECKED values (see the file's doc), in the
// document order of their parts: one for each traced check of DERIVE whose
// part is not among times, the parts of the declared times, and each
// Bitcoin header member no traced check recomputes, with no recomputed
// value.
func recomputedValues(proof map[string]any, publicKey [32]byte, times map[string]bool) ([]Recomputed, error) {
	checks, err := deriveChecks(proof, publicKey, AttributionUnattributed)
	if err != nil {
		return nil, fmt.Errorf("a value of this proof cannot be recomputed: %w", err)
	}
	type value struct {
		at Pointer
		r  Recomputed
	}
	var kept []value
	traced := map[string]bool{}
	for _, c := range checks {
		if !c.traced {
			continue
		}
		traced[c.part.Encode()] = true
		if !times[c.part.Encode()] {
			signed := c.construction == "handover" && c.part[len(c.part)-1] == Name("handover_signature")
			kept = append(kept, value{c.part, Recomputed{
				Construction: c.construction, Part: c.part.Encode(), Carried: c.declared, Value: c.recomputed, SignedMessage: signed,
			}})
		}
	}
	for _, entry := range commitmentEntries(proof) {
		if entry.name != ChainBitcoin {
			continue
		}
		for _, member := range bitcoinHeaderMembers {
			if part := under(entry.at, member); !traced[part.Encode()] {
				kept = append(kept, value{part, Recomputed{Construction: bitcoinRow, Part: part.Encode(), Carried: valueAt(proof, part)}})
			}
		}
	}
	SortBy(kept, func(v value) Pointer { return v.at })
	out := make([]Recomputed, 0, len(kept))
	for _, v := range kept {
		out = append(out, v.r)
	}
	return out, nil
}

// recordedValues are P's RECORDED values (SPEC.md section 3.7), in
// document order: each block node's record id, the RECORDED members of
// its key event, its previous_block_hash unless its record is the genesis
// block's, and its signing_key_id unless its key event introduces a key or
// is a prerotation; the subject's and each observation's record id; and
// each NIST pulse's version.
func recordedValues(proof map[string]any) []Recorded {
	var at []Pointer
	for _, block := range blockNodes(proof) {
		record := under(block.at, "record")
		at = append(at, under(record, "id"))
		event, _ := valueAt(proof, under(record, "metadata", "key_event")).(map[string]any)
		for _, member := range recordedKeyEventMembers {
			if _, carried := event[member]; carried {
				at = append(at, under(record, "metadata", "key_event", member))
			}
		}
		if valueAt(proof, under(record, "metadata", "genesis")) == nil {
			at = append(at, under(record, "previous_block_hash"))
		}
		if typ, _ := event["type"].(string); !slices.Contains(keyIntroducingEvents, typ) {
			at = append(at, under(record, "signing_key_id"))
		}
	}
	at = append(at, Pointer{Name("subject"), Name("record"), Name("id")})
	if observations, ok := proof["observations"].(map[string]any); ok {
		for name := range observations {
			at = append(at, Pointer{Name("observations"), Name(name), Name("record"), Name("id")})
		}
	}
	for _, object := range entropyObjects(proof) {
		if object.name == "nist" {
			at = append(at, under(object.at, "payload", "pulse", "version"))
		}
	}
	SortBy(at, func(p Pointer) Pointer { return p })
	var out []Recorded
	for _, p := range at {
		if value := valueAt(proof, p); value != nil {
			out = append(out, Recorded{Part: p.Encode(), Value: value})
		}
	}
	return out
}

// carriedValue is value, at at in P, without each member whose pointer
// given holds, and whether it is kept: an object that had members and has
// none left is not kept, and an object P holds empty is. An array is kept
// whole: P holds none but a path, which is one PATH INPUT value.
func carriedValue(value any, at Pointer, given map[string]bool) (any, bool) {
	if given[at.Encode()] {
		return nil, false
	}
	object, ok := value.(map[string]any)
	if !ok {
		return value, true
	}
	kept := map[string]any{}
	for name, member := range object {
		if v, keep := carriedValue(member, under(at, name), given); keep {
			kept[name] = v
		}
	}
	return kept, len(kept) > 0 || len(object) == 0
}

// declaredTimes are the times P carries (see the file's doc), in the
// document order of their parts.
func declaredTimes(proof map[string]any) []DeclaredTime {
	var at []Pointer
	var classes []string
	for _, p := range timePositions(proof) {
		class := DeclaredTIME
		if p.class == ClassNISTTIME {
			class = DeclaredNISTTIME
		}
		at, classes = append(at, p.at), append(classes, class)
	}
	for _, object := range entropyObjects(proof) {
		if object.name == ChainBitcoin {
			at, classes = append(at, under(object.at, "payload", "time")), append(classes, DeclaredUnixSeconds)
		}
	}
	out := make([]DeclaredTime, 0, len(at))
	order := make([]int, len(at))
	for i := range order {
		order[i] = i
	}
	SortBy(order, func(i int) Pointer { return at[i] })
	for _, i := range order {
		out = append(out, DeclaredTime{Part: at[i].Encode(), Class: classes[i], Value: valueAt(proof, at[i])})
	}
	return out
}
