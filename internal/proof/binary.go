// Copyright (c) 2021-2026 Truestamp, Inc.
// SPDX-License-Identifier: MIT

package proof

import (
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"

	"github.com/fxamacker/cbor/v2"
)

// IsCBORProof checks for CBOR self-describing tag (0xd9d9f7, RFC 8949 tag 55799).
func IsCBORProof(data []byte) bool {
	return len(data) >= 3 && data[0] == 0xd9 && data[1] == 0xd9 && data[2] == 0xf7
}

// cborDecMode is a lenient CBOR decode mode that accepts invalid UTF-8 in
// text strings. Some CBOR encoders may encode binary data as text strings.
var cborDecMode = func() cbor.DecMode {
	dm, err := cbor.DecOptions{UTF8: cbor.UTF8DecodeInvalid}.DecMode()
	if err != nil {
		panic("failed to create CBOR decode mode: " + err.Error())
	}
	return dm
}()

// ParseCBOR decodes a CBOR proof bundle into a ProofBundle.
// The output is structurally identical to what ParseBytes produces from JSON.
func ParseCBOR(data []byte) (*ProofBundle, error) {
	var raw map[string]any
	if err := cborDecMode.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("CBOR decode: %w", err)
	}

	publicKeyB64, err := bytesToBase64(raw, "pk")
	if err != nil {
		return nil, fmt.Errorf("pk: %w", err)
	}

	signatureB64, err := bytesToBase64(raw, "sig")
	if err != nil {
		return nil, fmt.Errorf("sig: %w", err)
	}

	subject, rawData, err := decodeSubjectCBOR(raw["s"])
	if err != nil {
		return nil, fmt.Errorf("s: %w", err)
	}

	block, err := decodeBlockCBOR(raw["b"])
	if err != nil {
		return nil, fmt.Errorf("b: %w", err)
	}

	commits, err := decodeCommitsCBOR(raw["cx"])
	if err != nil {
		return nil, fmt.Errorf("cx: %w", err)
	}

	proofType := "item"
	if subject.Source != "item" {
		proofType = "entropy"
	}

	timestamp, _ := raw["ts"].(string)
	inclusionProof := getStringField(raw, "ip")

	return &ProofBundle{
		Version:        toInt(raw["v"]),
		Type:           proofType,
		Timestamp:      timestamp,
		PublicKey:      publicKeyB64,
		Signature:      signatureB64,
		Subject:        subject,
		Block:          block,
		Commitments:    commits,
		InclusionProof: inclusionProof,
		RawData:        rawData,
	}, nil
}

func decodeSubjectCBOR(v any) (Subject, json.RawMessage, error) {
	m, ok := v.(map[any]any)
	if !ok {
		m2, ok2 := v.(map[string]any)
		if !ok2 {
			return Subject{}, nil, fmt.Errorf("expected map, got %T", v)
		}
		m = toAnyKeyMap(m2)
	}

	source, _ := m["src"].(string)
	id, _ := m["id"].(string)
	metadataHash := bytesFieldToHex(m, "mh")
	signingKeyID := bytesFieldToHex(m, "kid")

	dataJSON, err := anyToJSON(m["d"])
	if err != nil {
		return Subject{}, nil, fmt.Errorf("encoding data to JSON: %w", err)
	}

	subject := Subject{
		Source:       source,
		ID:           id,
		Data:         dataJSON,
		MetadataHash: metadataHash,
		SigningKeyID: signingKeyID,
	}

	return subject, dataJSON, nil
}

func decodeBlockCBOR(v any) (Block, error) {
	m := normalizeMap(v)
	if m == nil {
		return Block{}, fmt.Errorf("expected map, got %T", v)
	}

	return Block{
		ID:                getString(m, "id"),
		PreviousBlockHash: bytesFieldToHex(m, "ph"),
		MerkleRoot:        bytesFieldToHex(m, "mr"),
		MetadataHash:      bytesFieldToHex(m, "mh"),
		SigningKeyID:      bytesFieldToHex(m, "kid"),
	}, nil
}

func decodeCommitsCBOR(v any) ([]ExternalCommit, error) {
	if v == nil {
		return nil, nil
	}
	items, ok := v.([]any)
	if !ok {
		return nil, fmt.Errorf("expected array, got %T", v)
	}

	commits := make([]ExternalCommit, len(items))
	for i, item := range items {
		m := normalizeMap(item)
		if m == nil {
			return nil, fmt.Errorf("commitment %d: expected map", i)
		}

		ec := ExternalCommit{
			Type:    getString(m, "t"),
			Network: getString(m, "net"),
		}

		// Epoch proof: may be bytes or string
		if ep, ok := m["ep"]; ok {
			switch val := ep.(type) {
			case []byte:
				ec.EpochProof = base64.RawURLEncoding.EncodeToString(val)
			case string:
				ec.EpochProof = val
			}
		}

		// Stellar fields
		ec.TransactionHash = bytesFieldToHex(m, "tx")
		ec.MemoHash = bytesFieldToHex(m, "memo")
		ec.Ledger = getInt(m, "l")
		ec.Timestamp = getString(m, "ts")

		// Bitcoin fields
		ec.OpReturn = bytesFieldToHex(m, "op")
		ec.RawTxHex = bytesFieldToHex(m, "rtx")
		ec.TxoutproofHex = bytesFieldToHex(m, "txp")
		ec.BlockMerkleRoot = bytesFieldToHex(m, "bmr")
		ec.BlockHeight = getInt(m, "h")

		commits[i] = ec
	}

	return commits, nil
}

// ── Helpers ─────────────────────────────────────────────────────────

func bytesToBase64(m map[string]any, key string) (string, error) {
	v, ok := m[key]
	if !ok {
		return "", fmt.Errorf("missing field %q", key)
	}
	switch val := v.(type) {
	case []byte:
		return base64.StdEncoding.EncodeToString(val), nil
	case string:
		// Already base64 or raw binary as string
		if len(val) <= 64 && !isHexString(val) {
			return base64.StdEncoding.EncodeToString([]byte(val)), nil
		}
		return val, nil
	default:
		return "", fmt.Errorf("expected bytes or string for %q, got %T", key, v)
	}
}

func bytesFieldToHex(m map[any]any, key string) string {
	v, ok := m[key]
	if !ok {
		return ""
	}
	switch val := v.(type) {
	case []byte:
		return hex.EncodeToString(val)
	case string:
		// If string looks like hex already, return as-is.
		// Otherwise it's raw binary encoded as a CBOR text string.
		if isHexString(val) {
			return val
		}
		return hex.EncodeToString([]byte(val))
	default:
		return ""
	}
}

func isHexString(s string) bool {
	if len(s) == 0 || len(s)%2 != 0 {
		return false
	}
	for _, c := range s {
		if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F')) {
			return false
		}
	}
	return true
}

func getString(m map[any]any, key string) string {
	v, ok := m[key]
	if !ok {
		return ""
	}
	s, _ := v.(string)
	return s
}

func getStringField(m map[string]any, key string) string {
	v, ok := m[key]
	if !ok {
		return ""
	}
	switch val := v.(type) {
	case string:
		return val
	case []byte:
		return base64.RawURLEncoding.EncodeToString(val)
	default:
		return ""
	}
}

func getInt(m map[any]any, key string) int {
	v, ok := m[key]
	if !ok {
		return 0
	}
	return toInt(v)
}

func toInt(v any) int {
	switch n := v.(type) {
	case uint64:
		return int(n)
	case int64:
		return int(n)
	case float64:
		return int(n)
	case int:
		return n
	default:
		return 0
	}
}

// normalizeMap converts both map[string]any and map[any]any to map[any]any.
func normalizeMap(v any) map[any]any {
	switch m := v.(type) {
	case map[any]any:
		return m
	case map[string]any:
		return toAnyKeyMap(m)
	default:
		return nil
	}
}

func toAnyKeyMap(m map[string]any) map[any]any {
	result := make(map[any]any, len(m))
	for k, v := range m {
		result[k] = v
	}
	return result
}

// anyToJSON converts a CBOR-decoded value to JSON bytes.
// CBOR maps may have []byte values for byte strings; these are hex-encoded.
func anyToJSON(v any) (json.RawMessage, error) {
	converted := convertForJSON(v)
	b, err := json.Marshal(converted)
	if err != nil {
		return nil, err
	}
	return json.RawMessage(b), nil
}

// convertForJSON recursively converts CBOR types to JSON-safe types.
func convertForJSON(v any) any {
	switch val := v.(type) {
	case map[any]any:
		result := make(map[string]any, len(val))
		for k, v2 := range val {
			key, _ := k.(string)
			if key == "" {
				continue
			}
			result[key] = convertForJSON(v2)
		}
		return result
	case map[string]any:
		result := make(map[string]any, len(val))
		for k, v2 := range val {
			result[k] = convertForJSON(v2)
		}
		return result
	case []any:
		result := make([]any, len(val))
		for i, v2 := range val {
			result[i] = convertForJSON(v2)
		}
		return result
	case []byte:
		// Byte strings become hex-encoded strings in JSON context
		return hex.EncodeToString(val)
	default:
		return val
	}
}