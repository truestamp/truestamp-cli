// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: MIT

package proof

import (
	"bytes"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/fxamacker/cbor/v2"
	"github.com/truestamp/truestamp-cli/internal/tscrypto"
)

// deterministicCBOR is a CBOR encode mode configured for RFC 8949 section
// 4.2 core deterministic encoding: shortest-form integers, definite-length
// containers, lexicographically sorted map keys. A JSON bundle therefore
// converts to one CBOR encoding, whatever key order the JSON carried.
//
// EncMode() can only fail on invalid option combinations; this one is
// statically valid, so the error is discarded.
var deterministicCBOR, _ = cbor.CoreDetEncOptions().EncMode()

// MarshalCBOR produces the CBOR wire form of the bundle by converting its
// JSON document with [JSONToCBOR].
func (b *Bundle) MarshalCBOR() ([]byte, error) {
	if len(b.JSON) == 0 {
		return nil, errNoDocument
	}
	return JSONToCBOR(b.JSON)
}

// JSONToCBOR converts a JSON proof bundle to its CBOR wire form, the exact
// inverse of [CBORToJSON]: `public_key` and `signature` become byte strings
// from base64, every hash slot becomes a byte string from lowercase hex
// (uppercase is refused rather than laundered into a byte string the
// verifier would then accept), every other value keeps its JSON type, and
// integers are preserved exactly. The output is core-deterministic and
// wrapped in the self-describing tag 55799.
func JSONToCBOR(doc []byte) ([]byte, error) {
	dec := json.NewDecoder(bytes.NewReader(doc))
	dec.UseNumber()
	var v any
	if err := dec.Decode(&v); err != nil {
		return nil, fmt.Errorf("parsing JSON: %w", err)
	}
	if dec.More() {
		return nil, fmt.Errorf("trailing data after the JSON document")
	}
	if _, ok := v.(map[string]any); !ok {
		return nil, fmt.Errorf("input is not a JSON object")
	}
	converted, err := jsonValueToCBOR(v, roleBundle, "")
	if err != nil {
		return nil, err
	}
	body, err := deterministicCBOR.Marshal(converted)
	if err != nil {
		return nil, err
	}
	return append([]byte{0xd9, 0xd9, 0xf7}, body...), nil
}

func jsonValueToCBOR(v any, r role, path string) (any, error) {
	switch val := v.(type) {
	case map[string]any:
		out := make(map[string]any, len(val))
		for k, v2 := range val {
			childPath := k
			if path != "" {
				childPath = path + "." + k
			}
			conv, err := jsonValueToCBOR(v2, childRole(r, k), childPath)
			if err != nil {
				return nil, err
			}
			out[k] = conv
		}
		return out, nil
	case []any:
		out := make([]any, len(val))
		er := elementRole(r)
		for i, v2 := range val {
			conv, err := jsonValueToCBOR(v2, er, fmt.Sprintf("%s[%d]", path, i))
			if err != nil {
				return nil, err
			}
			out[i] = conv
		}
		return out, nil
	case json.Number:
		return jsonNumberToCBOR(val, path)
	case string:
		switch r {
		case roleHex:
			if err := tscrypto.ValidateLowercaseHex(val); err != nil {
				return nil, fmt.Errorf("%s: %w", path, err)
			}
			b, err := hex.DecodeString(val)
			if err != nil {
				return nil, fmt.Errorf("%s: %w", path, err)
			}
			return nonNilBytes(b), nil
		case roleBase64:
			b, err := base64.StdEncoding.DecodeString(val)
			if err != nil {
				return nil, fmt.Errorf("%s: base64 decode: %w", path, err)
			}
			return nonNilBytes(b), nil
		}
		return val, nil
	default:
		return v, nil
	}
}

// jsonNumberToCBOR maps a JSON number to the CBOR type that preserves it
// exactly: an integer literal becomes a CBOR integer, anything else an
// IEEE-754 double. A literal that fits neither is an error naming the key
// rather than a silently rounded, and therefore corrupted, bundle.
func jsonNumberToCBOR(n json.Number, path string) (any, error) {
	s := n.String()
	if !strings.ContainsAny(s, ".eE") {
		if i, err := strconv.ParseInt(s, 10, 64); err == nil {
			return i, nil
		}
		if u, err := strconv.ParseUint(s, 10, 64); err == nil {
			return u, nil
		}
		return nil, fmt.Errorf("%s: integer %s is outside the exactly representable 64-bit range and cannot be encoded without loss", path, s)
	}
	f, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	return f, nil
}

// nonNilBytes normalizes a nil slice to an empty one so the encoder emits a
// zero-length byte string rather than `null`. A null value counts as absent
// under E.6's presence rule, so an empty field must still round-trip as
// present.
func nonNilBytes(b []byte) []byte {
	if b == nil {
		return []byte{}
	}
	return b
}
