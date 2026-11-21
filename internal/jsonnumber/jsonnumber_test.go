// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: Apache-2.0

package jsonnumber

import (
	"encoding/json/jsontext"
	"encoding/json/v2"
	"math"
	"reflect"
	"testing"
)

// TestAsWritten_KeepsEveryNumberAsWritten decodes numbers at every depth an
// any can hold one, in a struct field, a map and an array, and checks that
// each is its spelling and that marshaling gives the text back. Mutants
// killed: decoding a number as float64 (61234567890123456789 rounds, 1.50
// loses its zero), skipping nested values, and decoding a non-number
// through the function (true or a string would become a jsontext.Value).
func TestAsWritten_KeepsEveryNumberAsWritten(t *testing.T) {
	const text = `{"n":61234567890123456789,"m":{"f":1.50,"list":[1,2.0,{"e":3e2}],"s":"9","t":true,"z":null}}`
	var got struct {
		N any            `json:"n"`
		M map[string]any `json:"m"`
	}
	if err := json.Unmarshal([]byte(text), &got, AsWritten); err != nil {
		t.Fatal(err)
	}
	want := map[string]any{
		"f":    jsontext.Value("1.50"),
		"list": []any{jsontext.Value("1"), jsontext.Value("2.0"), map[string]any{"e": jsontext.Value("3e2")}},
		"s":    "9",
		"t":    true,
		"z":    nil,
	}
	if !reflect.DeepEqual(got.N, jsontext.Value("61234567890123456789")) || !reflect.DeepEqual(got.M, want) {
		t.Fatalf("decoded %#v and %#v", got.N, got.M)
	}
	again, err := json.Marshal(got, json.Deterministic(true))
	if err != nil {
		t.Fatal(err)
	}
	if string(again) != text {
		t.Errorf("marshaled again\n%s\nwant\n%s", again, text)
	}
}

// TestReaders checks Int64, Uint64 and Float64 against what v1's
// Number.Int64 and Number.Float64 gave, and that each refuses a value that
// is not a number.
func TestReaders(t *testing.T) {
	if n, err := Int64(jsontext.Value("-9007199254740993")); err != nil || n != -9007199254740993 {
		t.Errorf("Int64: %d, %v", n, err)
	}
	if _, err := Int64(jsontext.Value("1.0")); err == nil {
		t.Error("Int64 read 1.0")
	}
	if n, err := Uint64(jsontext.Value("18446744073709551615")); err != nil || n != math.MaxUint64 {
		t.Errorf("Uint64: %d, %v", n, err)
	}
	if _, err := Uint64(jsontext.Value("-1")); err == nil {
		t.Error("Uint64 read -1")
	}
	if f, err := Float64(jsontext.Value("1e-400")); err != nil || f != 0 {
		t.Errorf("Float64: %v, %v", f, err)
	}
	for _, notNumber := range []string{`"1"`, `true`, `null`, `[1]`} {
		if _, err := Int64(jsontext.Value(notNumber)); err == nil {
			t.Errorf("Int64 read %s", notNumber)
		}
		if _, err := Uint64(jsontext.Value(notNumber)); err == nil {
			t.Errorf("Uint64 read %s", notNumber)
		}
		if _, err := Float64(jsontext.Value(notNumber)); err == nil {
			t.Errorf("Float64 read %s", notNumber)
		}
	}
}
