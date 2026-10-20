// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: Apache-2.0

// This file says where P, the parsed value of proof, holds each kind of
// node (SPEC.md sections 3.3 to 3.6), each with its pointer ([Pointer]),
// and reads the value at a pointer.
//
// The positions come from this file's compiled lists and from type, never
// from the names a document happens to hold: type is the only member that
// selects what runs (SPEC.md section 10.6, rule 3), and the chains and
// entropy sources are the alternatives the schema lists. Each function
// reads a node only where the layout of P's type puts one, and skips a
// position P leaves empty.
//
//	| Function            | Positions                                                                 |
//	|---------------------|---------------------------------------------------------------------------|
//	| blockNodes          | /block; /signing_key_event/block; /witnesses/block in an item              |
//	| commitmentEntries   | /commitments/C and /signing_key_event/commitments/C, C bitcoin or stellar  |
//	| entropyObjects      | /subject/entropy in an entropy type; /witnesses/entropy_X in an item; /observations/entropy_X/entropy in a beacon |
//	| genesisBitcoin      | B/record/metadata/genesis/bitcoin for each block node B whose metadata carries genesis |
//
// SHAPE reads the subject's id through valueAt once check 3 has held P to
// the schema (SPEC.md section 10.2, step 7, and section 10.5), and the
// TIME and NISTTIME values of the commitment entries and entropy objects at
// check 5; POLICY (step 8) reads their networks and those of genesis.bitcoin.
//
// A node is a [layoutNode], with the chain or source its position names;
// the entropy sources are the list layoutSources (SPEC.md section 3.5), and
// the chains the list chains (sources.go).

package verifier

// layoutSources are the entropy sources, in JCS order (SPEC.md section
// 3.5).
var layoutSources = []string{"bitcoin", "nist", "stellar"}

// genesisBitcoinAt is where a block node holds genesis.bitcoin.
var genesisBitcoinAt = Pointer{Name("record"), Name("metadata"), Name("genesis"), Name("bitcoin")}

// layoutNode is a node of P at its pointer: for a commitment entry, the
// chain it is for; for an entropy object, the source its position names.
type layoutNode struct {
	at   Pointer
	name string
	node map[string]any
}

// blockNodes returns every block node P carries, in document order.
func blockNodes(proof map[string]any) []layoutNode {
	positions := []Pointer{{Name("block")}, {Name("signing_key_event"), Name("block")}}
	if proof["type"] == "item" {
		positions = append(positions, Pointer{Name("witnesses"), Name("block")})
	}
	return present(proof, positions, nil)
}

// commitmentEntries returns every commitment entry P carries, each with
// its chain.
func commitmentEntries(proof map[string]any) []layoutNode {
	var positions []Pointer
	var names []string
	for _, base := range []Pointer{{Name("commitments")}, {Name("signing_key_event"), Name("commitments")}} {
		for _, chain := range chains {
			positions, names = append(positions, with(base, Name(chain))), append(names, chain)
		}
	}
	return present(proof, positions, names)
}

// entropyObjects returns every entropy object P carries, each with the
// source its position names: the proof's type for /subject/entropy, and
// the member's name in /witnesses and /observations.
func entropyObjects(proof map[string]any) []layoutNode {
	var positions []Pointer
	var names []string
	switch typ := proof["type"]; typ {
	case "item", "beacon":
		for _, source := range layoutSources {
			at := Pointer{Name("witnesses"), Name("entropy_" + source)}
			if typ == "beacon" {
				at = Pointer{Name("observations"), Name("entropy_" + source), Name("entropy")}
			}
			positions, names = append(positions, at), append(names, source)
		}
	default:
		for _, source := range layoutSources {
			if typ == "entropy_"+source {
				positions, names = []Pointer{{Name("subject"), Name("entropy")}}, []string{source}
			}
		}
	}
	return present(proof, positions, names)
}

// genesisBitcoin returns every genesis.bitcoin object P carries, one for
// each block node whose record's metadata holds genesis.
func genesisBitcoin(proof map[string]any) []layoutNode {
	var positions []Pointer
	for _, block := range blockNodes(proof) {
		positions = append(positions, append(append(Pointer{}, block.at...), genesisBitcoinAt...))
	}
	return present(proof, positions, nil)
}

// present returns the objects P holds at positions, each with its name
// from names when names is given, skipping a position P leaves empty.
func present(proof map[string]any, positions []Pointer, names []string) []layoutNode {
	var out []layoutNode
	for i, at := range positions {
		if node, ok := valueAt(proof, at).(map[string]any); ok {
			n := layoutNode{at: at, node: node}
			if names != nil {
				n.name = names[i]
			}
			out = append(out, n)
		}
	}
	return out
}

// valueAt returns the value at pointer p in value, and nil where value
// holds none: past a member value lacks, an index past an array's end, or
// a segment that names a member of an array or an element of an object.
func valueAt(value any, p Pointer) any {
	for _, s := range p {
		switch v := value.(type) {
		case map[string]any:
			if s.isIndex {
				return nil
			}
			value = v[s.name]
		case []any:
			if !s.isIndex || s.index >= len(v) {
				return nil
			}
			value = v[s.index]
		default:
			return nil
		}
	}
	return value
}
