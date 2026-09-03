// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: MIT

// Package proof parses, downloads, and converts Truestamp proof bundles in
// both JSON and CBOR wire formats. Proofs are the self-contained artifacts
// consumers receive from the API; this package handles only serialization
// and I/O. Cryptographic verification lives in internal/verify.
//
// The model deliberately keeps every hashed map (the subject's claims or
// entropy payload, both metadata maps, every block metadata map, and every
// entropy witness payload) as the raw JSON the wire carried, so that RFC
// 8785 canonicalization sees the producer's bytes: integers stay exact, no
// value is coerced, and nothing is re-cased.
package proof

import (
	"encoding/json"
	"math"
	"sort"
	"strconv"
	"strings"

	"github.com/truestamp/truestamp-cli/internal/proof/ptype"
)

// Bundle is a parsed version 1 proof bundle. Normative reference: Appendix E
// of `truestamp-v2/whitepaper/whitepaper.typ`; the wire shape is described
// in `kb/proof-bundle-format.md`.
//
// Only the Appendix E.6 hard rejections abort a parse. Every other defect
// (a wrong `version`, an undecodable `public_key`, a mis-cased hash) parses
// cleanly so the verification pipeline can grade it as a step result.
type Bundle struct {
	// JSON is the document this bundle was parsed from: the input bytes for
	// JSON input, or the JSON-value-space conversion of the input for CBOR
	// input. It is what MarshalJSON returns and what MarshalCBOR encodes.
	JSON []byte

	// FromCBOR records which serialization the input arrived in.
	FromCBOR bool

	// Version is `version` when it is an exact JSON integer, else 0. The
	// literal as carried is kept for the E.8 message.
	Version        int
	VersionLiteral json.RawMessage

	// Type is the registry name and Code its frozen integer code (E.24).
	// Both are validated at parse: an unknown name is a hard rejection.
	Type string
	Code ptype.Code

	GeneratedAt string // "" unless carried as a JSON string
	PublicKey   string // padded base64, "" unless carried as a JSON string
	Signature   string // padded base64, "" unless carried as a JSON string

	// Subject is nil for block-like subjects and non-nil otherwise.
	Subject *Subject

	// InclusionProof is "" for block-like subjects and the carried string
	// otherwise (an empty string parses and fails the E.12 decode).
	InclusionProof string

	Block BlockMap

	// BlockPath is the ordered list of block maps linking the head block to
	// the containing block. nil when absent, not a list, or empty.
	BlockPath []BlockMap

	// Commitments has at least one entry, each of which passed the E.6
	// per-entry gate (a map, a registered chain, a string epoch_proof, a
	// present epoch_merkle_root).
	Commitments []Commitment

	// SigningKeyEvent is nil when the key is absent or null.
	SigningKeyEvent *SigningKeyEvent

	// Fields is the top-level object as carried.
	Fields Object
}

// IsBlockLike reports whether the subject is a block or a beacon.
func (b *Bundle) IsBlockLike() bool { return ptype.IsBlockLikeSubject(b.Code) }

// IsItem reports whether the subject is an item.
func (b *Bundle) IsItem() bool { return b.Code == ptype.Item }

// IsEntropy reports whether the subject is an entropy observation.
func (b *Bundle) IsEntropy() bool { return ptype.IsEntropySubject(b.Code) }

// SubjectID returns the subject's id, or the block id for a block-like
// subject.
func (b *Bundle) SubjectID() string {
	if b.Subject != nil {
		return b.Subject.ID
	}
	return b.Block.ID
}

// MarshalJSON returns the bundle's JSON document. For a bundle parsed from
// JSON this is the input verbatim; for CBOR input it is the JSON-value-space
// conversion of the input, with byte-string fields rendered per Appendix E.3
// and hashed maps kept exactly as decoded.
func (b *Bundle) MarshalJSON() ([]byte, error) {
	if len(b.JSON) == 0 {
		return nil, errNoDocument
	}
	return b.JSON, nil
}

// Subject is the `subject` map of a non-block-like bundle.
type Subject struct {
	Fields Object

	ID string // "" unless carried as a JSON string

	// Claims is `subject.claims` exactly as carried (nil when absent or
	// null); Entropy likewise for `subject.entropy`. Which one a bundle
	// uses follows its type.
	Claims  json.RawMessage
	Entropy json.RawMessage

	// Metadata is `subject.metadata` exactly as carried; the E.6 gate
	// guarantees it is a JSON object.
	Metadata json.RawMessage

	SigningKeyID string // "" unless carried as a JSON string

	// Witnesses is `subject.witnesses` as carried: witness name to the raw
	// detail. nil when absent or not a map. Unknown names are preserved so
	// the verifier can report them.
	Witnesses Object
}

// Data returns the hashed data map for the subject type: claims for an
// item, the entropy payload for an entropy subject.
func (s *Subject) Data(code ptype.Code) json.RawMessage {
	if code == ptype.Item {
		return s.Claims
	}
	return s.Entropy
}

// CommittedWitnesses returns `subject.metadata.witnesses` as carried:
// witness name to the raw committed value. Empty when the metadata carries
// no witnesses map.
func (s *Subject) CommittedWitnesses() Object {
	meta, ok := parseObject(s.Metadata)
	if !ok {
		return Object{}
	}
	w, ok := meta.Object("witnesses")
	if !ok {
		return Object{}
	}
	return w
}

// WitnessNamesCarried returns the sorted names under `subject.witnesses`.
func (s *Subject) WitnessNamesCarried() []string {
	if s == nil || s.Witnesses == nil {
		return nil
	}
	return s.Witnesses.Keys()
}

// BlockMap is the five-field block shape that appears as the top-level
// `block`, each `block_path` entry, the `block` witness detail, and
// `signing_key_event.block`. Every one of them derives its hash by the
// single procedure of Appendix E.14.
type BlockMap struct {
	// Fields is the map as carried; nil when the value was not a map.
	Fields Object

	ID                string // "" unless carried as a JSON string
	PreviousBlockHash string
	MerkleRoot        string
	SigningKeyID      string

	// Metadata is the block metadata map exactly as carried; nil unless a
	// JSON object.
	Metadata json.RawMessage
}

// IsMap reports whether the block value was a JSON object at all.
func (b BlockMap) IsMap() bool { return b.Fields != nil }

// Commitment is one entry of `commitments` (or of
// `signing_key_event.commitments`). The E.6 gate guarantees Chain is a
// registered chain name, EpochProof was carried as a string, and
// epoch_merkle_root was present; everything else is graded downstream.
type Commitment struct {
	Fields Object

	Chain           string
	EpochMerkleRoot string // "" when present but not a JSON string
	EpochProof      string

	TransactionHash string
	Network         string
	Timestamp       string

	// Ledger and BlockHeight are exact integers when HasLedger /
	// HasBlockHeight is true. A present value that is not a JSON integer
	// leaves the flag false; the consuming confirmation step grades it.
	Ledger         int
	HasLedger      bool
	BlockHeight    int
	HasBlockHeight bool

	Txoutproof      string
	RawTransaction  string
	BlockMerkleRoot string
}

// SigningKeyEvent is the optional top-level witness of the signature: the
// ledger block whose metadata.key_event introduced the signing key, plus
// that block's own public-chain commitments.
type SigningKeyEvent struct {
	// IsMap is false when `signing_key_event` was present but not a map;
	// the reference verifier fails that shape rather than skipping it.
	IsMap  bool
	Fields Object

	Block BlockMap // Block.IsMap() is false when `block` is absent or not a map

	// Commitments holds the entries that pass the E.6 per-entry gate;
	// CommitmentCount is how many entries the list carried in total, so a
	// list with entries but none valid is distinguishable from no list.
	Commitments     []Commitment
	CommitmentCount int
}

// KeyEvent is the `key_event` object inside a key-event block's metadata.
type KeyEvent struct {
	Type      string
	KeyID     string
	PublicKey string
	// Sequence is the raw literal, rendered verbatim in the report.
	Sequence json.RawMessage
}

// KeyEvent reads `block.metadata.key_event` from the carried block map. ok
// is false when the block carries no key_event map.
func (e *SigningKeyEvent) KeyEvent() (KeyEvent, bool) {
	meta, ok := parseObject(e.Block.Metadata)
	if !ok {
		return KeyEvent{}, false
	}
	ke, ok := meta.Object("key_event")
	if !ok {
		return KeyEvent{}, false
	}
	return KeyEvent{
		Type:      ke.Str("type"),
		KeyID:     ke.Str("key_id"),
		PublicKey: ke.Str("public_key"),
		Sequence:  ke.Raw("sequence"),
	}, true
}

// Object is a JSON object as carried on the wire: each member's raw
// literal, so presence, null-ness and value type stay distinguishable.
// Appendix E.6's rules are written in terms of all three, and Go's zero
// values collapse them.
type Object map[string]json.RawMessage

// HasKey reports whether the key is present at all, null included. The
// `unsupported_layout` gate is keyed on presence, not on value.
func (o Object) HasKey(key string) bool {
	_, ok := o[key]
	return ok
}

// Has reports whether the key is present with a non-null value, which is
// E.6's presence rule for every other gate.
func (o Object) Has(key string) bool {
	return fieldCarried(o[key])
}

// Raw returns the member's raw literal, or nil when absent.
func (o Object) Raw(key string) json.RawMessage { return o[key] }

// String returns the member as a string; ok is false unless the member is
// a JSON string. A null member is not a string: encoding/json leaves a
// string untouched when it decodes null, which would otherwise read as a
// carried empty string.
func (o Object) String(key string) (string, bool) {
	raw, ok := o[key]
	if !ok || !fieldCarried(raw) {
		return "", false
	}
	var s string
	if err := json.Unmarshal(raw, &s); err != nil {
		return "", false
	}
	return s, true
}

// IsString reports whether the member is carried as a JSON string.
func (o Object) IsString(key string) bool {
	_, ok := o.String(key)
	return ok
}

// Str returns the member as a string, or "" when it is absent, null, or of
// another type.
func (o Object) Str(key string) string {
	s, _ := o.String(key)
	return s
}

// Object returns the member as an object; ok is false unless it is one.
func (o Object) Object(key string) (Object, bool) {
	return parseObject(o[key])
}

// List returns the member's elements; ok is false unless it is a JSON
// array.
func (o Object) List(key string) ([]json.RawMessage, bool) {
	raw, ok := o[key]
	if !ok || !isArray(raw) {
		return nil, false
	}
	var items []json.RawMessage
	if err := json.Unmarshal(raw, &items); err != nil {
		return nil, false
	}
	return items, true
}

// Integer returns the member as an exact integer; ok is false unless the
// member is a JSON integer literal.
func (o Object) Integer(key string) (Integer, bool) {
	return jsonInteger(o[key])
}

// Literal renders the member for a report message: the compact JSON
// literal as carried, or "absent" when the key is missing.
func (o Object) Literal(key string) string {
	raw, ok := o[key]
	if !ok {
		return "absent"
	}
	return compactLiteral(raw)
}

// Keys returns the member names in lexicographic order.
func (o Object) Keys() []string {
	keys := make([]string, 0, len(o))
	for k := range o {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// Integer is an exactly parsed JSON integer. Text is its decimal
// rendering, which messages use so they can never name a truncated
// stand-in for the value the bundle carries; N holds the value where it
// fits an int64.
type Integer struct {
	N    int64
	Fits bool
	Text string
}

func (v Integer) String() string { return v.Text }

// Int returns the value as an int, ok=false when it does not fit.
func (v Integer) Int() (int, bool) {
	if !v.Fits || v.N < math.MinInt || v.N > math.MaxInt {
		return 0, false
	}
	return int(v.N), true
}

// jsonInteger reports whether raw is a JSON integer literal, and returns it
// exactly. A literal carrying a fraction or an exponent is not an integer:
// JSON has one number type, so `20.0` and `2e1` are doubles that happen to
// be whole, which is how the reference verifier grades them too.
func jsonInteger(raw json.RawMessage) (Integer, bool) {
	// The literal has to be inspected before it is decoded: encoding/json
	// accepts a *quoted* number into a json.Number, so `"20"` (a JSON
	// string) would otherwise pass as the integer 20.
	text := strings.TrimSpace(string(raw))
	if text == "" || (text[0] != '-' && (text[0] < '0' || text[0] > '9')) {
		return Integer{}, false
	}
	var num json.Number
	if json.Unmarshal(raw, &num) != nil {
		return Integer{}, false
	}
	text = num.String()
	if strings.ContainsAny(text, ".eE") {
		return Integer{}, false
	}
	n, err := strconv.ParseInt(text, 10, 64)
	return Integer{N: n, Fits: err == nil, Text: text}, true
}

// parseObject decodes a JSON object into its raw members. ok is false for
// every value that is not an object, which is the shape E.6 gates on.
func parseObject(raw json.RawMessage) (Object, bool) {
	if !isObject(raw) {
		return nil, false
	}
	var keys map[string]json.RawMessage
	if err := json.Unmarshal(raw, &keys); err != nil || keys == nil {
		return nil, false
	}
	return Object(keys), true
}

// fieldCarried implements E.6's presence rule for a bundle field: a key
// that is missing, or whose value is null, counts as absent; any other
// value, including an empty string, counts as carried.
func fieldCarried(raw json.RawMessage) bool {
	return len(raw) > 0 && strings.TrimSpace(string(raw)) != "null"
}

// isObject reports whether the first non-whitespace byte opens a JSON
// object.
func isObject(raw json.RawMessage) bool { return firstByte(raw) == '{' }

// isArray reports whether the first non-whitespace byte opens a JSON array.
func isArray(raw json.RawMessage) bool { return firstByte(raw) == '[' }

func firstByte(raw []byte) byte {
	for _, c := range raw {
		switch c {
		case ' ', '\t', '\n', '\r':
			continue
		default:
			return c
		}
	}
	return 0
}

// compactLiteral renders a raw JSON value on one line, bounded so a hostile
// document cannot inject an unbounded string into a report.
func compactLiteral(raw json.RawMessage) string {
	s := strings.Join(strings.Fields(string(raw)), " ")
	const max = 64
	if len(s) > max {
		return s[:max] + "..."
	}
	return s
}
