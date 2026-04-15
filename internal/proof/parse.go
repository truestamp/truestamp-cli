// Copyright (c) 2021-2026 Truestamp, Inc.
// SPDX-License-Identifier: MIT

package proof

import (
	"encoding/json"
	"fmt"
	"os"
)

// Parse reads and parses a proof JSON file, preserving raw JSON for JCS.
func Parse(filename string) (*ProofBundle, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("reading file: %w", err)
	}
	return ParseBytes(data)
}

// ParseBytes parses a proof from raw bytes. Detects CBOR format (self-describing
// tag 0xd9d9f7) and falls back to JSON parsing.
func ParseBytes(data []byte) (*ProofBundle, error) {
	if IsCBORProof(data) {
		return ParseCBOR(data)
	}

	// JSON parsing: extract top-level fields with short keys
	var raw struct {
		Version   int    `json:"v"`
		Timestamp string `json:"ts"`
		PublicKey string `json:"pk"`
		Signature string `json:"sig"`

		Subject        json.RawMessage `json:"s"`
		Block          json.RawMessage `json:"b"`
		Commitments    json.RawMessage `json:"cx"`
		InclusionProof string          `json:"ip"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("parsing JSON: %w", err)
	}

	// Validate top-level fields
	if raw.Version == 0 {
		return nil, fmt.Errorf("missing required field: v")
	}
	if raw.PublicKey == "" {
		return nil, fmt.Errorf("missing required field: pk")
	}
	if raw.Signature == "" {
		return nil, fmt.Errorf("missing required field: sig")
	}

	// Parse subject
	if len(raw.Subject) == 0 || string(raw.Subject) == "null" {
		return nil, fmt.Errorf("missing required field: s")
	}
	var subject Subject
	if err := json.Unmarshal(raw.Subject, &subject); err != nil {
		return nil, fmt.Errorf("parsing subject: %w", err)
	}

	// Preserve raw data from subject for JCS canonicalization
	rawData := subject.Data

	// Determine type from subject source
	proofType := "item"
	if subject.Source != "item" {
		proofType = "entropy"
	}

	// Parse block
	if len(raw.Block) == 0 || string(raw.Block) == "null" {
		return nil, fmt.Errorf("missing required field: b")
	}
	var block Block
	if err := json.Unmarshal(raw.Block, &block); err != nil {
		return nil, fmt.Errorf("parsing block: %w", err)
	}

	// Parse external commitments
	var commits []ExternalCommit
	if len(raw.Commitments) > 0 && string(raw.Commitments) != "null" {
		if err := json.Unmarshal(raw.Commitments, &commits); err != nil {
			return nil, fmt.Errorf("parsing commitments: %w", err)
		}
	}

	bundle := &ProofBundle{
		Version:        raw.Version,
		Type:           proofType,
		Timestamp:      raw.Timestamp,
		PublicKey:      raw.PublicKey,
		Signature:      raw.Signature,
		Subject:        subject,
		Block:          block,
		Commitments:    commits,
		InclusionProof: raw.InclusionProof,
		RawData:        rawData,
	}

	return bundle, nil
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
