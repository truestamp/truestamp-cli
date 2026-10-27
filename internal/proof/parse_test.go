// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: Apache-2.0

package proof

import (
	"bytes"
	"encoding/json"
	"reflect"
	"testing"

	"github.com/truestamp/truestamp-cli/internal/proof/ptype"
	"github.com/truestamp/truestamp-cli/internal/testfixtures"
)

const prodItemID = "01M1M0V3SE3C5P32TRAJSNX6QF"

func readProd(t *testing.T, name string) []byte {
	t.Helper()
	return testfixtures.Read(t, testfixtures.ProdDir, name)
}

func mustParse(t *testing.T, data []byte) *Bundle {
	t.Helper()
	b, err := ParseBytes(data)
	if err != nil {
		t.Fatalf("ParseBytes: %v", err)
	}
	return b
}

// normalized decodes a JSON document into a comparable tree with every
// number kept as its literal, so two documents can be compared for value
// equality regardless of key order and whitespace.
func normalized(t *testing.T, doc []byte) any {
	t.Helper()
	dec := json.NewDecoder(bytes.NewReader(doc))
	dec.UseNumber()
	var v any
	if err := dec.Decode(&v); err != nil {
		t.Fatalf("decoding document: %v", err)
	}
	return v
}

func TestParse_ProductionBundles(t *testing.T) {
	cases := []struct {
		file             string
		witnessesCarried []string
		keyEvent         bool
	}{
		{testfixtures.ProdComplete, []string{"block", "entropy_bitcoin", "entropy_nist", "entropy_stellar"}, true},
		{testfixtures.ProdCompact, nil, false},
		{testfixtures.ProdPartial, []string{"block", "entropy_nist"}, false},
	}
	for _, tc := range cases {
		t.Run(tc.file, func(t *testing.T) {
			b := mustParse(t, readProd(t, tc.file))
			if b.Type != ptype.NameItem || b.Code != ptype.Item || !b.IsItem() {
				t.Fatalf("type = %q code %d, want item/20", b.Type, b.Code)
			}
			if b.Version != 1 {
				t.Errorf("version = %d, want 1", b.Version)
			}
			if b.Subject == nil || b.Subject.ID != prodItemID {
				t.Fatalf("subject id = %v, want %s", b.Subject, prodItemID)
			}
			if b.Subject.SigningKeyID != "3c19f776" || b.Block.SigningKeyID != "3c19f776" {
				t.Errorf("signing_key_id: subject %q block %q", b.Subject.SigningKeyID, b.Block.SigningKeyID)
			}
			if got := b.Subject.CommittedWitnesses().Keys(); !reflect.DeepEqual(got, []string{"block", "entropy_bitcoin", "entropy_nist", "entropy_stellar"}) {
				t.Errorf("committed witnesses = %v", got)
			}
			if got := b.Subject.WitnessNamesCarried(); !reflect.DeepEqual(got, tc.witnessesCarried) {
				t.Errorf("carried witnesses = %v, want %v", got, tc.witnessesCarried)
			}
			if (b.SigningKeyEvent != nil) != tc.keyEvent {
				t.Errorf("signing key event carried = %v, want %v", b.SigningKeyEvent != nil, tc.keyEvent)
			}
			if len(b.Commitments) != 1 || b.Commitments[0].Chain != "stellar" {
				t.Fatalf("commitments = %+v", b.Commitments)
			}
			c := b.Commitments[0]
			if !c.HasLedger || c.Ledger != 64256842 || c.Network != "public" || c.Timestamp != "2026-09-03T16:20:05Z" {
				t.Errorf("commitment fields = %+v", c)
			}
			if c.TransactionHash != "9a58b4414607648452500674e83c778567a155573205932138ef5b25955979a2" {
				t.Errorf("transaction_hash = %q", c.TransactionHash)
			}
			if b.Block.Metadata == nil || string(b.Block.Metadata) != "{}" {
				t.Errorf("block.metadata = %s", b.Block.Metadata)
			}
			if b.BlockPath != nil {
				t.Errorf("block_path should be absent, got %d entries", len(b.BlockPath))
			}
			if b.InclusionProof == "" {
				t.Error("inclusion proof missing")
			}
			if len(b.Subject.Claims) == 0 || b.Subject.Entropy != nil {
				t.Errorf("claims/entropy: %d/%d bytes", len(b.Subject.Claims), len(b.Subject.Entropy))
			}
		})
	}
}

func TestParse_SigningKeyEvent(t *testing.T) {
	b := mustParse(t, readProd(t, testfixtures.ProdComplete))
	e := b.SigningKeyEvent
	if e == nil || !e.IsMap || !e.Block.IsMap() {
		t.Fatalf("signing key event = %+v", e)
	}
	if e.Block.ID != "019fcf1d-b17b-7897-adb4-3dd23adc9d0e" {
		t.Errorf("key event block id = %q", e.Block.ID)
	}
	ke, ok := e.KeyEvent()
	if !ok || ke.Type != "genesis" || ke.KeyID != "3c19f776" || string(ke.Sequence) != "0" {
		t.Errorf("key event = %+v ok=%v", ke, ok)
	}
	if ke.PublicKey != b.PublicKey {
		t.Errorf("key event public key %q != bundle %q", ke.PublicKey, b.PublicKey)
	}
	if e.CommitmentCount != 1 || len(e.Commitments) != 1 || e.Commitments[0].Ledger != 63802260 {
		t.Errorf("key event commitments = %+v (count %d)", e.Commitments, e.CommitmentCount)
	}
}

func TestParse_AppendixDBundle(t *testing.T) {
	b := mustParse(t, testfixtures.Read(t, testfixtures.WhitepaperDir, testfixtures.AppendixD))
	if b.Subject == nil || b.Subject.ID != "01KY9ZWEX0248J48HK6D248NAN" {
		t.Fatalf("subject = %+v", b.Subject)
	}
	if len(b.Commitments) != 2 || b.Commitments[0].Chain != "stellar" || b.Commitments[1].Chain != "bitcoin" {
		t.Fatalf("commitments = %+v", b.Commitments)
	}
	if !b.Commitments[1].HasBlockHeight || b.Commitments[1].BlockHeight != 870456 {
		t.Errorf("bitcoin block_height = %+v", b.Commitments[1])
	}
	if b.Commitments[1].HasLedger {
		t.Error("bitcoin entry should carry no ledger")
	}
}

// --- E.6 hard rejections ---

func rejectionOf(t *testing.T, data []byte) (string, error) {
	t.Helper()
	_, err := ParseBytes(data)
	if err == nil {
		t.Fatal("expected a rejection, got none")
	}
	code := RejectionCode(err)
	if code == "" {
		t.Fatalf("error is not a RejectionError: %v", err)
	}
	return code, err
}

// tamper-type.json, the complete item bundle with its type made a block's,
// is refused for the subject fields a block-like type must not carry.
// Mutant killed: ParseBytes without that refusal (the bundle parses).
func TestRejection_Fixtures(t *testing.T) {
	cases := map[string]string{
		"tamper-type.json": CodeUnexpectedSubjectFieldsForBlockLike,
	}
	for file, want := range cases {
		data := testfixtures.Read(t, testfixtures.TamperDir, file)
		if code, err := rejectionOf(t, data); code != want {
			t.Errorf("%s: rejection %q, want %q (%v)", file, code, want, err)
		}
	}
}

// mutate applies edits to the complete production bundle and returns the
// JSON document.
func mutate(t *testing.T, edit func(top map[string]any)) []byte {
	t.Helper()
	top := normalized(t, readProd(t, testfixtures.ProdComplete)).(map[string]any)
	edit(top)
	out, err := json.Marshal(top)
	if err != nil {
		t.Fatal(err)
	}
	return out
}

// No member's name selects a layout: a sound bundle that also carries
// top-level v and t keys parses like any other, and a document in an
// earlier layout is graded by the gates every document meets, here the
// missing subject. Mutant killed: a rejection keyed on a top-level v or t.
func TestParse_TopLevelVAndTSelectNothing(t *testing.T) {
	data := mutate(t, func(m map[string]any) { m["v"] = nil; m["t"] = 20 })
	if _, err := ParseBytes(data); err != nil {
		t.Fatalf("a sound bundle with v and t: %v", err)
	}
	if code, err := rejectionOf(t, testfixtures.Read(t, testfixtures.TamperDir, "old-layout.json")); code != CodeMissingSubject {
		t.Fatalf("old-layout.json: %q (%v), want missing_subject", code, err)
	}
}

func TestRejection_Order(t *testing.T) {
	cases := []struct {
		name string
		edit func(top map[string]any)
		want string
	}{
		{"type absent", func(m map[string]any) { delete(m, "type") }, CodeInvalidSubjectType},
		{"type unknown", func(m map[string]any) { m["type"] = "commitment_stellar" }, CodeInvalidSubjectType},
		{"type not a string", func(m map[string]any) { m["type"] = 20 }, CodeInvalidSubjectType},
		{"bad type outranks missing block", func(m map[string]any) { m["type"] = "nope"; delete(m, "block") }, CodeInvalidSubjectType},
		{"block absent", func(m map[string]any) { delete(m, "block") }, CodeMissingBlock},
		{"block not a map", func(m map[string]any) { m["block"] = "x" }, CodeMissingBlock},
		{"block-like with subject", func(m map[string]any) { m["type"] = "block" }, CodeUnexpectedSubjectFieldsForBlockLike},
		{"block-like with inclusion_proof only", func(m map[string]any) { m["type"] = "beacon"; delete(m, "subject") }, CodeUnexpectedSubjectFieldsForBlockLike},
		{"subject absent", func(m map[string]any) { delete(m, "subject") }, CodeMissingSubject},
		{"subject null", func(m map[string]any) { m["subject"] = nil }, CodeMissingSubject},
		{"inclusion_proof absent", func(m map[string]any) { delete(m, "inclusion_proof") }, CodeMissingInclusionProof},
		{"inclusion_proof not a string", func(m map[string]any) { m["inclusion_proof"] = 5 }, CodeMissingInclusionProof},
		{"block.metadata absent", func(m map[string]any) { delete(m["block"].(map[string]any), "metadata") }, CodeMissingMetadata},
		{"block.metadata not a map", func(m map[string]any) { m["block"].(map[string]any)["metadata"] = "x" }, CodeMissingMetadata},
		{"subject.metadata absent", func(m map[string]any) { delete(m["subject"].(map[string]any), "metadata") }, CodeMissingMetadata},
		{"block metadata outranks commitments", func(m map[string]any) {
			delete(m["block"].(map[string]any), "metadata")
			delete(m, "commitments")
		}, CodeMissingMetadata},
		{"commitments absent", func(m map[string]any) { delete(m, "commitments") }, CodeNoExternalCommitments},
		{"commitments empty", func(m map[string]any) { m["commitments"] = []any{} }, CodeNoExternalCommitments},
		{"commitments not a list", func(m map[string]any) { m["commitments"] = map[string]any{} }, CodeNoExternalCommitments},
		{"entry not a map", func(m map[string]any) { m["commitments"] = []any{"x"} }, CodeInvalidCommitmentEntry},
		{"entry bad chain", func(m map[string]any) { m["commitments"].([]any)[0].(map[string]any)["chain"] = "ethereum" }, CodeInvalidCommitmentEntry},
		{"entry chain absent", func(m map[string]any) { delete(m["commitments"].([]any)[0].(map[string]any), "chain") }, CodeInvalidCommitmentEntry},
		{"entry epoch_proof absent", func(m map[string]any) { delete(m["commitments"].([]any)[0].(map[string]any), "epoch_proof") }, CodeInvalidCommitmentEntry},
		{"entry epoch_proof not a string", func(m map[string]any) { m["commitments"].([]any)[0].(map[string]any)["epoch_proof"] = 1 }, CodeInvalidCommitmentEntry},
		{"entry epoch_merkle_root absent", func(m map[string]any) { delete(m["commitments"].([]any)[0].(map[string]any), "epoch_merkle_root") }, CodeInvalidCommitmentEntry},
		{"entry epoch_merkle_root null", func(m map[string]any) { m["commitments"].([]any)[0].(map[string]any)["epoch_merkle_root"] = nil }, CodeInvalidCommitmentEntry},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if code, err := rejectionOf(t, mutate(t, tc.edit)); code != tc.want {
				t.Errorf("rejection %q, want %q (%v)", code, tc.want, err)
			}
		})
	}
}

func TestRejection_NotAJSONObject(t *testing.T) {
	for _, in := range []string{"", "[]", "null", "42", "\"x\"", "{", "{\"type\": }"} {
		if code, _ := rejectionOf(t, []byte(in)); code != CodeNotAJSONObject {
			t.Errorf("%q: rejection %q, want %q", in, code, CodeNotAJSONObject)
		}
	}
}

// binaryMap is the map {"proof": {}} in the binary encoding of RFC 8949,
// bare (a definite-length map), as an indefinite-length map, and behind
// the self-describing tag 55799 (d9 d9 f7).
var binaryMap = map[string][]byte{
	"bare":       {0xa1, 0x65, 'p', 'r', 'o', 'o', 'f', 0xa0},
	"indefinite": {0xbf, 0x65, 'p', 'r', 'o', 'o', 'f', 0xa0, 0xff},
	"tagged":     {0xd9, 0xd9, 0xf7, 0xa1, 0x65, 'p', 'r', 'o', 'o', 'f', 0xa0},
}

// TestRejection_BinaryEncodingIsNotAJSONObject pins that the parser reads
// JSON only: a map in RFC 8949's binary encoding is refused as not a JSON
// object at its first octet, never decoded. Mutant killed: ParseBytes
// detecting that encoding by its first octet and decoding it to JSON
// first (the decoded {"proof": {}} is refused as invalid_subject_type).
func TestRejection_BinaryEncodingIsNotAJSONObject(t *testing.T) {
	for name, in := range binaryMap {
		code, err := rejectionOf(t, in)
		if code != CodeNotAJSONObject || err.Error() != "not_a_json_object: input is not a JSON object" {
			t.Errorf("%s: rejection %q (%v), want %q with the message \"input is not a JSON object\"", name, code, err, CodeNotAJSONObject)
		}
	}
}

// TestParse_GradedDefectsAreNotRejections pins that E.6 gates on the
// presence and type of structure only: a wrong version, a broken key, an
// uppercase hash and a malformed proof all parse and reach the report.
func TestParse_GradedDefectsAreNotRejections(t *testing.T) {
	cases := []struct {
		name string
		edit func(top map[string]any)
	}{
		{"version 2", func(m map[string]any) { m["version"] = 2 }},
		{"version absent", func(m map[string]any) { delete(m, "version") }},
		{"version a string", func(m map[string]any) { m["version"] = "1" }},
		{"public_key garbage", func(m map[string]any) { m["public_key"] = "nope" }},
		{"public_key absent", func(m map[string]any) { delete(m, "public_key") }},
		{"signature absent", func(m map[string]any) { delete(m, "signature") }},
		{"generated_at absent", func(m map[string]any) { delete(m, "generated_at") }},
		{"uppercase merkle_root", func(m map[string]any) { m["block"].(map[string]any)["merkle_root"] = "D175D1EF" }},
		{"empty inclusion_proof", func(m map[string]any) { m["inclusion_proof"] = "" }},
		{"block id numeric", func(m map[string]any) { m["block"].(map[string]any)["id"] = 7 }},
		{"ledger a string", func(m map[string]any) { m["commitments"].([]any)[0].(map[string]any)["ledger"] = "64256842" }},
		{"witnesses not a map", func(m map[string]any) { m["subject"].(map[string]any)["witnesses"] = "x" }},
		{"block_path empty", func(m map[string]any) { m["block_path"] = []any{} }},
		{"signing_key_event not a map", func(m map[string]any) { m["signing_key_event"] = "x" }},
		{"unknown top-level field", func(m map[string]any) { m["future"] = map[string]any{"x": 1} }},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := ParseBytes(mutate(t, tc.edit)); err != nil {
				t.Errorf("unexpected rejection: %v", err)
			}
		})
	}
	b := mustParse(t, mutate(t, func(m map[string]any) { m["version"] = "1" }))
	if b.Version != 0 || string(b.VersionLiteral) != `"1"` {
		t.Errorf("string version: Version=%d literal=%s", b.Version, b.VersionLiteral)
	}
	b = mustParse(t, mutate(t, func(m map[string]any) { m["commitments"].([]any)[0].(map[string]any)["ledger"] = "64256842" }))
	if b.Commitments[0].HasLedger {
		t.Error("a string ledger must not read as an integer")
	}
	b = mustParse(t, mutate(t, func(m map[string]any) { m["signing_key_event"] = "x" }))
	if b.SigningKeyEvent == nil || b.SigningKeyEvent.IsMap {
		t.Errorf("non-map signing_key_event = %+v", b.SigningKeyEvent)
	}
}

func TestObjectHelpers(t *testing.T) {
	o, ok := parseObject([]byte(`{"s":"x","n":5,"f":1.5,"nul":null,"o":{},"l":[1]}`))
	if !ok {
		t.Fatal("parseObject failed")
	}
	if o.Has("nul") || o.Has("missing") || !o.Has("s") {
		t.Error("presence helpers")
	}
	if o.Str("s") != "x" || o.Str("n") != "" || o.IsString("n") || o.IsString("nul") {
		t.Error("string helpers")
	}
	if v, ok := o.Integer("n"); !ok || v.N != 5 {
		t.Error("integer helper")
	}
	if _, ok := o.Integer("f"); ok {
		t.Error("float read as integer")
	}
	if o.Literal("missing") != "absent" || o.Literal("f") != "1.5" {
		t.Errorf("literals: %q %q", o.Literal("missing"), o.Literal("f"))
	}
	if _, ok := o.List("l"); !ok {
		t.Error("list helper")
	}
	if got := o.Keys(); !reflect.DeepEqual(got, []string{"f", "l", "n", "nul", "o", "s"}) {
		t.Errorf("keys = %v", got)
	}
}

func TestPtypeRegistry(t *testing.T) {
	for name, want := range map[string]ptype.Code{"block": 10, "beacon": 11, "item": 20, "entropy_nist": 30, "entropy_stellar": 31, "entropy_bitcoin": 32} {
		c, ok := ptype.FromName(name)
		if !ok || c != want || ptype.Name(c) != name {
			t.Errorf("%s -> %d (ok=%v), Name=%s", name, c, ok, ptype.Name(c))
		}
	}
	if _, ok := ptype.FromName("stellar"); ok {
		t.Error("a chain name is not a subject type")
	}
}
