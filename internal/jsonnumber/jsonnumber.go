// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: Apache-2.0

// Package jsonnumber reads a JSON number as it is written, for the reads
// that must keep an integer past 2^53 exact or print a number as its source
// wrote it. encoding/json/v2 decodes every number that lands in an any as a
// float64 and has no counterpart to v1's UseNumber; [AsWritten] is that
// counterpart.
//
// It is not the reader of proofs, keyrings or source answers, which is
// internal/strictjson, and it applies none of SPEC.md section 10.1's rules
// beyond what encoding/json/v2 makes.
package jsonnumber

import (
	"bytes"
	"encoding/json/jsontext"
	"encoding/json/v2"
	"errors"
	"fmt"
	"strconv"
)

// AsWritten is an unmarshal option under which each JSON number decoded
// into an any, at any depth, is the [jsontext.Value] it is written as, and
// every other value decodes as it does by default. Such a number marshals
// again exactly as written, so "1.50" stays "1.50" and a ledger sequence is
// never re-printed as a float.
var AsWritten = json.WithUnmarshalers(json.UnmarshalFromFunc(func(dec *jsontext.Decoder, v *any) error {
	if dec.PeekKind() != '0' {
		return errors.ErrUnsupported // not a number: decode it as usual
	}
	raw, err := dec.ReadValue()
	if err != nil {
		return err
	}
	// ReadValue's bytes are valid only until the next read.
	*v = jsontext.Value(bytes.Clone(raw))
	return nil
}))

// Int64 reads n, a number as written, as an int64, as v1's Number.Int64
// did: an error for a fraction, an exponent or a value past the int64 range.
func Int64(n jsontext.Value) (int64, error) {
	if n.Kind() != '0' {
		return 0, fmt.Errorf("jsonnumber: %s is not a number", n)
	}
	return strconv.ParseInt(string(n), 10, 64)
}

// Uint64 reads n, a number as written, as a uint64: an error for a sign, a
// fraction, an exponent or a value past the uint64 range.
func Uint64(n jsontext.Value) (uint64, error) {
	if n.Kind() != '0' {
		return 0, fmt.Errorf("jsonnumber: %s is not a number", n)
	}
	return strconv.ParseUint(string(n), 10, 64)
}

// Float64 reads n, a number as written, as its nearest float64, as v1's
// Number.Float64 did.
func Float64(n jsontext.Value) (float64, error) {
	if n.Kind() != '0' {
		return 0, fmt.Errorf("jsonnumber: %s is not a number", n)
	}
	return strconv.ParseFloat(string(n), 64)
}
