// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: MIT

package proof

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"

	"github.com/truestamp/truestamp-cli/internal/proof/ptype"
)

var errNoDocument = errors.New("bundle carries no JSON document")

// Parse reads and parses a proof file in either wire format.
func Parse(filename string) (*Bundle, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("reading file: %w", err)
	}
	return ParseBytes(data)
}

// ParseBytes parses a proof from raw bytes, dispatching to [ParseCBOR] for
// CBOR input and to [ParseJSON] otherwise.
//
// Only the hard rejections Appendix E.6 enumerates abort here, in E.6's
// order, so that two verifiers handed one malformed bundle name the same
// first defect. Every abort returns a [RejectionError] carrying its E.23
// identifier. Everything else parses cleanly for the pipeline to grade.
func ParseBytes(data []byte) (*Bundle, error) {
	if IsCBORProof(data) {
		return ParseCBOR(data)
	}
	return ParseJSON(data)
}

// ParseJSON parses a JSON proof bundle.
func ParseJSON(data []byte) (*Bundle, error) {
	return parseDocument(data, false)
}

// ParseCBOR parses a CBOR proof bundle, tagged (55799) or bare. The input is
// first converted to the JSON value space per Appendix E.3, which is where
// the `invalid_subject_data` rejection is raised, and the result is then
// gated and parsed exactly as a JSON bundle would be, so the two
// serializations grade one logical bundle identically.
func ParseCBOR(data []byte) (*Bundle, error) {
	doc, err := CBORToJSON(data)
	if err != nil {
		return nil, err
	}
	return parseDocument(doc, true)
}

// parseDocument applies the E.6 gates in order and populates the model.
func parseDocument(doc []byte, fromCBOR bool) (*Bundle, error) {
	if !isObject(doc) {
		return nil, Rejectf(CodeNotAJSONObject, "input is not a JSON object")
	}
	var top map[string]json.RawMessage
	if err := json.Unmarshal(doc, &top); err != nil {
		return nil, Rejectf(CodeNotAJSONObject, "parsing JSON: %s", err)
	}
	fields := Object(top)

	// Version 1 is the first published format. A top-level `v` or `t` is
	// the signature of a pre-publication draft, and no verifier keeps a
	// second code path for it: the holder is told to regenerate the proof.
	// The gate is on key presence, null included, matching the reference.
	if fields.HasKey("v") || fields.HasKey("t") {
		return nil, Rejectf(CodeUnsupportedLayout,
			"a top-level v or t key is present: this is the pre-publication draft layout, which version 1 replaced")
	}

	typeName, isString := fields.String("type")
	if !isString || !ptype.IsSubjectName(typeName) {
		return nil, Rejectf(CodeInvalidSubjectType,
			"type is %s; expected one of block, beacon, item, entropy_nist, entropy_stellar, entropy_bitcoin",
			fields.Literal("type"))
	}
	code, _ := ptype.FromName(typeName)
	blockLike := ptype.IsBlockLikeSubject(code)

	blockObj, ok := fields.Object("block")
	if !ok {
		return nil, Rejectf(CodeMissingBlock, "block is not a map (%s)", fields.Literal("block"))
	}

	if blockLike && (fields.Has("subject") || fields.Has("inclusion_proof")) {
		key := "subject"
		if !fields.Has("subject") {
			key = "inclusion_proof"
		}
		return nil, Rejectf(CodeUnexpectedSubjectFieldsForBlockLike, "a %s proof must not carry %s", typeName, key)
	}

	var subjectObj Object
	var inclusionProof string
	if !blockLike {
		subjectObj, ok = fields.Object("subject")
		if !ok {
			return nil, Rejectf(CodeMissingSubject, "subject is not a map (%s)", fields.Literal("subject"))
		}
		inclusionProof, ok = fields.String("inclusion_proof")
		if !ok {
			return nil, Rejectf(CodeMissingInclusionProof, "inclusion_proof is not a string (%s)", fields.Literal("inclusion_proof"))
		}
	}

	// No opaque hashes: the metadata maps ARE the preimages every metadata
	// hash is derived from, so a bundle that omits one carries no way to
	// reach the composite subject hash or the block hash. That is a
	// missing structure rather than a failed derivation.
	if !isObject(blockObj.Raw("metadata")) {
		return nil, Rejectf(CodeMissingMetadata, "block.metadata is not a map (%s)", blockObj.Literal("metadata"))
	}
	if !blockLike && !isObject(subjectObj.Raw("metadata")) {
		return nil, Rejectf(CodeMissingMetadata, "subject.metadata is not a map (%s)", subjectObj.Literal("metadata"))
	}

	entries, ok := fields.List("commitments")
	if !ok {
		return nil, Rejectf(CodeNoExternalCommitments, "commitments is not a list (%s)", fields.Literal("commitments"))
	}
	if len(entries) == 0 {
		return nil, Rejectf(CodeNoExternalCommitments, "commitments must not be empty")
	}
	commits := make([]Commitment, 0, len(entries))
	for i, entry := range entries {
		c, err := parseCommitment(entry, fmt.Sprintf("commitments[%d]", i))
		if err != nil {
			return nil, err
		}
		commits = append(commits, c)
	}

	bundle := &Bundle{
		JSON:            doc,
		FromCBOR:        fromCBOR,
		VersionLiteral:  fields.Raw("version"),
		Type:            typeName,
		Code:            code,
		GeneratedAt:     fields.Str("generated_at"),
		PublicKey:       fields.Str("public_key"),
		Signature:       fields.Str("signature"),
		InclusionProof:  inclusionProof,
		Block:           parseBlockMap(fields.Raw("block")),
		BlockPath:       parseBlockPath(fields),
		Commitments:     commits,
		SigningKeyEvent: parseSigningKeyEvent(fields.Raw("signing_key_event")),
		Fields:          fields,
	}
	// `version` is deliberately NOT a gate: E.6 exempts it, and E.8 reports
	// a wrong or absent version as a failing step.
	if v, ok := fields.Integer("version"); ok {
		if n, fits := v.Int(); fits {
			bundle.Version = n
		}
	}
	if !blockLike {
		bundle.Subject = parseSubject(subjectObj)
	}
	return bundle, nil
}

// parseCommitment applies E.6's per-entry gate and reads the entry. Every
// member other than the three gated ones is read on its own so a
// wrong-typed optional field reaches the report rather than aborting the
// run.
func parseCommitment(raw json.RawMessage, label string) (Commitment, error) {
	obj, ok := parseObject(raw)
	if !ok {
		return Commitment{}, Rejectf(CodeInvalidCommitmentEntry, "%s is not a map", label)
	}
	chain, isString := obj.String("chain")
	if !isString || !ptype.IsChainName(chain) {
		return Commitment{}, Rejectf(CodeInvalidCommitmentEntry, "%s: chain is %s; expected \"stellar\" or \"bitcoin\"", label, obj.Literal("chain"))
	}
	epochProof, isString := obj.String("epoch_proof")
	if !isString {
		return Commitment{}, Rejectf(CodeInvalidCommitmentEntry, "%s is missing epoch_proof", label)
	}
	if !obj.Has("epoch_merkle_root") {
		return Commitment{}, Rejectf(CodeInvalidCommitmentEntry, "%s is missing epoch_merkle_root", label)
	}
	c := Commitment{
		Fields:          obj,
		Chain:           chain,
		EpochMerkleRoot: obj.Str("epoch_merkle_root"),
		EpochProof:      epochProof,
		TransactionHash: obj.Str("transaction_hash"),
		Network:         obj.Str("network"),
		Timestamp:       obj.Str("timestamp"),
		Txoutproof:      obj.Str("txoutproof"),
		RawTransaction:  obj.Str("raw_transaction"),
		BlockMerkleRoot: obj.Str("block_merkle_root"),
	}
	if v, ok := obj.Integer("ledger"); ok {
		if n, fits := v.Int(); fits {
			c.Ledger, c.HasLedger = n, true
		}
	}
	if v, ok := obj.Integer("block_height"); ok {
		if n, fits := v.Int(); fits {
			c.BlockHeight, c.HasBlockHeight = n, true
		}
	}
	return c, nil
}

// parseBlockMap reads a block map wherever one appears. A value that is
// not a map yields a BlockMap whose IsMap() is false.
func parseBlockMap(raw json.RawMessage) BlockMap {
	obj, ok := parseObject(raw)
	if !ok {
		return BlockMap{}
	}
	b := BlockMap{
		Fields:            obj,
		ID:                obj.Str("id"),
		PreviousBlockHash: obj.Str("previous_block_hash"),
		MerkleRoot:        obj.Str("merkle_root"),
		SigningKeyID:      obj.Str("signing_key_id"),
	}
	if isObject(obj.Raw("metadata")) {
		b.Metadata = obj.Raw("metadata")
	}
	return b
}

// parseBlockPath reads `block_path`. Absent, not a list, and an empty list
// all yield nil: the reference verifier treats `[]` as no path carried.
func parseBlockPath(fields Object) []BlockMap {
	entries, ok := fields.List("block_path")
	if !ok || len(entries) == 0 {
		return nil
	}
	path := make([]BlockMap, 0, len(entries))
	for _, e := range entries {
		path = append(path, parseBlockMap(e))
	}
	return path
}

func parseSubject(obj Object) *Subject {
	s := &Subject{
		Fields:       obj,
		ID:           obj.Str("id"),
		Metadata:     obj.Raw("metadata"),
		SigningKeyID: obj.Str("signing_key_id"),
	}
	if obj.Has("claims") {
		s.Claims = obj.Raw("claims")
	}
	if obj.Has("entropy") {
		s.Entropy = obj.Raw("entropy")
	}
	if w, ok := obj.Object("witnesses"); ok {
		s.Witnesses = w
	}
	return s
}

func parseSigningKeyEvent(raw json.RawMessage) *SigningKeyEvent {
	if !fieldCarried(raw) {
		return nil
	}
	obj, ok := parseObject(raw)
	if !ok {
		return &SigningKeyEvent{IsMap: false}
	}
	e := &SigningKeyEvent{
		IsMap:  true,
		Fields: obj,
		Block:  parseBlockMap(obj.Raw("block")),
	}
	if entries, ok := obj.List("commitments"); ok {
		e.CommitmentCount = len(entries)
		for i, entry := range entries {
			c, err := parseCommitment(entry, fmt.Sprintf("signing_key_event.commitments[%d]", i))
			if err != nil {
				continue
			}
			e.Commitments = append(e.Commitments, c)
		}
	}
	return e
}

// FileSizeFromData returns the byte length of proof data.
func FileSizeFromData(data []byte) int64 { return int64(len(data)) }

// FileSize returns the size of the file in bytes.
func FileSize(filename string) int64 {
	info, err := os.Stat(filename)
	if err != nil {
		return 0
	}
	return info.Size()
}
