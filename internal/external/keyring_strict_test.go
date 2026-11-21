// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: Apache-2.0

package external

import (
	"errors"
	"testing"

	"github.com/truestamp/truestamp-cli/internal/strictjson"
)

// A keyring body is read through the strict reader of SPEC.md section 10.1:
// one that repeats a member name, begins with a byte order mark, has
// content after its value, holds a lone surrogate or invalid UTF-8, or a
// number J5 refuses is malformed, never a keyring, and the reader's
// rejection is kept; and one that does not have a keyring's shape is
// malformed too. In a body the strict reader reads, absent and null
// members are zero, unknown members are ignored, an integral spelling of
// sequence (1.0, 1e0) is that integer, a null element of keys is a zero
// key, and only the exact name keys is the list, so a body whose list is
// named KEYS, like one with no list, is not a keyring. Mutants killed:
// ParseKeyring decoding with v1's encoding/json (the repeated name and the lone
// surrogate read as keyrings, "KEYS" as the keys list, and the sequences
// 1.0 and 1e0 refused); keyringKeyOf accepting a sequence that is not an
// integer; decodeKeyring refusing a null element.
func TestParseKeyring_ReadsThroughTheStrictReader(t *testing.T) {
	for name, body := range map[string]string{
		"a repeated name":        `{"keys": [], "keys": [{"key_id": "aa", "public_key": "bb"}]}`,
		"a byte order mark":      "\xef\xbb\xbf" + `{"keys": []}`,
		"trailing content":       `{"keys": []} {}`,
		"a lone surrogate":       `{"keys": [{"key_id": "\ud800"}]}`,
		"invalid UTF-8":          "{\"keys\": [{\"key_id\": \"\xff\"}]}",
		"the integer gap":        `{"keys": [{"sequence": 9007199254740993}]}`,
		"a fraction":             `{"keys": [{"sequence": 1.5}]}`,
		"a key id not a string":  `{"keys": [{"key_id": 5}]}`,
		"keys not a list":        `{"keys": "x"}`,
		"an element not a key":   `{"keys": [1]}`,
		"a version not a string": `{"version": 1, "keys": []}`,
		"an array":               `[]`,
	} {
		_, err := ParseKeyring([]byte(body))
		var malformed *MalformedResponseError
		if !errors.As(err, &malformed) || malformed.Detail != "response is not a JSON keyring document" {
			t.Errorf("%s: %v, want malformed", name, err)
		}
	}
	var rejection *strictjson.Rejection
	if _, err := ParseKeyring([]byte(`{"keys": [], "keys": []}`)); !errors.As(err, &rejection) {
		t.Errorf("a repeated name: %v, want the strict reader's rejection", err)
	}

	for name, body := range map[string]string{
		"null":      `null`,
		"no keys":   `{"version": "1.0"}`,
		"keys null": `{"keys": null}`,
		"KEYS":      `{"KEYS": []}`,
	} {
		var malformed *MalformedResponseError
		if _, err := ParseKeyring([]byte(body)); !errors.As(err, &malformed) || malformed.Detail != `response carries no "keys" list` {
			t.Errorf("%s: %v, want no keys list", name, err)
		}
	}

	keyring, err := ParseKeyring([]byte(`{"version": "1.0", "extra": [1], "keys": [` +
		`{"key_id": "aa", "public_key": "bb", "sequence": 1.0, "active": true, "note": null}, null, ` +
		`{"key_id": null, "public_key": "cc"}, {"key_id": "dd", "sequence": 1e0}]}`))
	if err != nil {
		t.Fatal(err)
	}
	want := []KeyringKey{{KeyID: "aa", PublicKey: "bb", Sequence: 1, Active: true}, {}, {PublicKey: "cc"}, {KeyID: "dd", Sequence: 1}}
	if keyring.Version != "1.0" || len(keyring.Keys) != len(want) {
		t.Fatalf("keyring %+v", keyring)
	}
	for i := range want {
		if keyring.Keys[i] != want[i] {
			t.Errorf("key %d = %+v, want %+v", i, keyring.Keys[i], want[i])
		}
	}
}

// FetchKeyring reads the body it fetches through the same reader: a body
// that repeats the keys name is malformed, the strict reader's rejection
// kept, never the keyring its second, empty list would make. Mutant
// killed: FetchKeyring decoding with v1's encoding/json (it gives a keyring of
// no keys).
func TestFetchKeyring_ARepeatedNameIsMalformed(t *testing.T) {
	srv := statusServer(t, 200, `{"version":"1.0","keys":[{"key_id":"aa","public_key":"bb"}],"keys":[]}`)
	keyring, err := FetchKeyring(srv.URL)
	var malformed *MalformedResponseError
	var rejection *strictjson.Rejection
	if keyring != nil || !errors.As(err, &malformed) || !errors.As(err, &rejection) {
		t.Fatalf("FetchKeyring = %+v, %v; want malformed, with the strict reader's rejection", keyring, err)
	}
}
