// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: MIT

package proof

import (
	"bytes"
	"encoding/json"
	"reflect"
	"strings"
	"testing"

	"github.com/fxamacker/cbor/v2"
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

// TestParseCBOR_EqualsJSON is Appendix E.3's invariant in field form: the
// CBOR bundle decodes to the same values as the JSON bundle, apart from the
// two fields that differ because the CBOR fixture was generated separately.
func TestParseCBOR_EqualsJSON(t *testing.T) {
	cborBundle := mustParse(t, readProd(t, testfixtures.ProdCBOR))
	if !cborBundle.FromCBOR {
		t.Fatal("FromCBOR = false")
	}
	jsonDoc := readProd(t, testfixtures.ProdComplete)

	want := normalized(t, jsonDoc).(map[string]any)
	got := normalized(t, cborBundle.JSON).(map[string]any)
	for _, k := range []string{"generated_at", "signature"} {
		if got[k] == want[k] {
			t.Errorf("%s: expected the CBOR fixture to differ from the JSON one", k)
		}
		delete(got, k)
		delete(want, k)
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("CBOR bundle differs from JSON bundle:\n got: %v\nwant: %v", got, want)
	}
}

// TestCBORToJSON_Roles pins the field-type correspondence: hash slots
// become lowercase hex, keys and signature base64, proofs stay text, and
// the witness hashes inside the hashed metadata map stay text.
func TestCBORToJSON_Roles(t *testing.T) {
	doc, err := CBORToJSON(readProd(t, testfixtures.ProdCBOR))
	if err != nil {
		t.Fatal(err)
	}
	top := normalized(t, doc).(map[string]any)
	if top["public_key"] != "1hHbF5H5u8LiSp+nVMRb8duR2eGkOo5Q1JYfcmAtF28=" {
		t.Errorf("public_key = %v", top["public_key"])
	}
	block := top["block"].(map[string]any)
	if block["merkle_root"] != "d175d1ef04595ffbea81cf9864780889615a73bdb0184c6efa43de184f5c24a2" {
		t.Errorf("block.merkle_root = %v", block["merkle_root"])
	}
	if block["signing_key_id"] != "3c19f776" {
		t.Errorf("block.signing_key_id = %v", block["signing_key_id"])
	}
	subject := top["subject"].(map[string]any)
	witnesses := subject["metadata"].(map[string]any)["witnesses"].(map[string]any)
	if witnesses["block"] != "e36f824dea9508d5dca570c77edc06eb4bc830b413b3db4e4d983ce3f85e9d78" {
		t.Errorf("committed block witness = %v", witnesses["block"])
	}
	entry := top["commitments"].([]any)[0].(map[string]any)
	if entry["ledger"] != json.Number("64256842") {
		t.Errorf("ledger = %v (%T)", entry["ledger"], entry["ledger"])
	}
	if !strings.HasPrefix(entry["epoch_proof"].(string), "AwfwB4Jp") {
		t.Errorf("epoch_proof = %v", entry["epoch_proof"])
	}
	nist := subject["witnesses"].(map[string]any)["entropy_nist"].(map[string]any)["pulse"].(map[string]any)
	if nist["chainIndex"] != json.Number("2") || !strings.HasPrefix(nist["outputValue"].(string), "7E0BA0E4") {
		t.Errorf("nist pulse = %v", nist)
	}
}

func TestRoundTrip_JSONToCBORToJSON(t *testing.T) {
	for _, name := range []string{testfixtures.ProdComplete, testfixtures.ProdCompact, testfixtures.ProdPartial} {
		t.Run(name, func(t *testing.T) {
			src := readProd(t, name)
			encoded, err := JSONToCBOR(src)
			if err != nil {
				t.Fatalf("JSONToCBOR: %v", err)
			}
			if !HasCBORTag(encoded) {
				t.Error("encoded CBOR lacks tag 55799")
			}
			back, err := CBORToJSON(encoded)
			if err != nil {
				t.Fatalf("CBORToJSON: %v", err)
			}
			if !reflect.DeepEqual(normalized(t, src), normalized(t, back)) {
				t.Errorf("round trip changed the document:\n%s\n%s", src, back)
			}
		})
	}
	src := testfixtures.Read(t, testfixtures.WhitepaperDir, testfixtures.AppendixD)
	encoded, err := JSONToCBOR(src)
	if err != nil {
		t.Fatalf("JSONToCBOR: %v", err)
	}
	back, err := CBORToJSON(encoded)
	if err != nil {
		t.Fatalf("CBORToJSON: %v", err)
	}
	if !reflect.DeepEqual(normalized(t, src), normalized(t, back)) {
		t.Errorf("Appendix D round trip changed the document")
	}
}

func TestRoundTrip_CBORToJSONToCBOR(t *testing.T) {
	src := readProd(t, testfixtures.ProdCBOR)
	b := mustParse(t, src)
	re, err := b.MarshalCBOR()
	if err != nil {
		t.Fatalf("MarshalCBOR: %v", err)
	}
	var want, got any
	if err := cbor.Unmarshal(src, &want); err != nil {
		t.Fatal(err)
	}
	if err := cbor.Unmarshal(re, &got); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(want, got) {
		t.Errorf("re-encoded CBOR decodes differently from the fixture")
	}
	if !bytes.Equal(src, re) {
		t.Logf("note: re-encoded CBOR is not byte-identical to the server's (%d vs %d bytes); values agree", len(re), len(src))
	}
}

func TestParseCBOR_BareMap(t *testing.T) {
	src := readProd(t, testfixtures.ProdCBOR)
	if !HasCBORTag(src) {
		t.Fatal("fixture is not tagged")
	}
	bare := src[3:]
	if !IsCBORProof(bare) {
		t.Fatal("bare map not detected as CBOR")
	}
	b := mustParse(t, bare)
	if b.Subject == nil || b.Subject.ID != prodItemID {
		t.Errorf("bare-map parse: subject = %+v", b.Subject)
	}
}

func TestIsCBORProof(t *testing.T) {
	if IsCBORProof([]byte("{}")) || IsCBORProof(nil) || IsCBORProof([]byte(" {")) {
		t.Error("JSON detected as CBOR")
	}
	if !IsCBORProof([]byte{0xa0}) || !IsCBORProof([]byte{0xbf, 0xff}) || !IsCBORProof([]byte{0xd9, 0xd9, 0xf7, 0xa0}) {
		t.Error("CBOR map not detected")
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

func TestRejection_Fixtures(t *testing.T) {
	cases := map[string]string{
		"old-layout.json":  CodeUnsupportedLayout,
		"tamper-type.json": CodeUnexpectedSubjectFieldsForBlockLike,
	}
	for file, want := range cases {
		data := testfixtures.Read(t, testfixtures.TamperDir, file)
		if code, err := rejectionOf(t, data); code != want {
			t.Errorf("%s: rejection %q, want %q (%v)", file, code, want, err)
		}
	}
	// Every other tamper fixture is a well-formed bundle and parses.
	for _, file := range []string{"tamper-claims.json", "tamper-witness-hash.json", "tamper-witness-detail.json", "tamper-root.json", "tamper-epoch-root.json", "drop-key-event.json"} {
		if _, err := ParseBytes(testfixtures.Read(t, testfixtures.TamperDir, file)); err != nil {
			t.Errorf("%s: unexpected rejection %v", file, err)
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

func TestRejection_Order(t *testing.T) {
	cases := []struct {
		name string
		edit func(top map[string]any)
		want string
	}{
		{"draft v key even when null", func(m map[string]any) { m["v"] = nil }, CodeUnsupportedLayout},
		{"draft t key", func(m map[string]any) { m["t"] = 20 }, CodeUnsupportedLayout},
		{"draft key outranks a bad type", func(m map[string]any) { m["t"] = 20; m["type"] = "nope" }, CodeUnsupportedLayout},
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

// --- CBOR value space (E.3) ---

// cborWith re-encodes the complete bundle as plain CBOR after applying an
// edit to the decoded tree. Byte-string slots stay text here (the decoder
// passes text through), which keeps the test about the injected value.
func cborWith(t *testing.T, edit func(top map[string]any)) []byte {
	t.Helper()
	var top map[string]any
	dec := json.NewDecoder(bytes.NewReader(readProd(t, testfixtures.ProdComplete)))
	if err := dec.Decode(&top); err != nil {
		t.Fatal(err)
	}
	edit(top)
	out, err := cbor.Marshal(top)
	if err != nil {
		t.Fatal(err)
	}
	return append([]byte{0xd9, 0xd9, 0xf7}, out...)
}

func TestParseCBOR_ValueSpaceRejections(t *testing.T) {
	subject := func(m map[string]any) map[string]any { return m["subject"].(map[string]any) }
	cases := []struct {
		name string
		edit func(top map[string]any)
		path string
	}{
		{"bytes in claims", func(m map[string]any) { subject(m)["claims"].(map[string]any)["blob"] = []byte{1, 2} }, "subject.claims.blob"},
		{"tag in metadata", func(m map[string]any) {
			subject(m)["metadata"].(map[string]any)["witnesses"].(map[string]any)["block"] = cbor.Tag{Number: 42, Content: "x"}
		}, "subject.metadata.witnesses.block"},
		{"undefined in block metadata", func(m map[string]any) {
			m["block"].(map[string]any)["metadata"].(map[string]any)["u"] = cbor.SimpleValue(23)
		}, "block.metadata.u"},
		{"bytes in an entropy witness payload", func(m map[string]any) {
			subject(m)["witnesses"].(map[string]any)["entropy_nist"].(map[string]any)["raw"] = []byte{9}
		}, "subject.witnesses.entropy_nist.raw"},
		{"bytes in the block witness metadata", func(m map[string]any) {
			subject(m)["witnesses"].(map[string]any)["block"].(map[string]any)["metadata"].(map[string]any)["raw"] = []byte{9}
		}, "subject.witnesses.block.metadata.raw"},
		{"bytes in the key event block metadata", func(m map[string]any) {
			m["signing_key_event"].(map[string]any)["block"].(map[string]any)["metadata"].(map[string]any)["raw"] = []byte{9}
		}, "signing_key_event.block.metadata.raw"},
		{"non-finite float in claims", func(m map[string]any) { subject(m)["claims"].(map[string]any)["f"] = mathInf() }, "subject.claims.f"},
		{"nested array element", func(m map[string]any) { subject(m)["claims"].(map[string]any)["a"] = []any{1, []byte{1}} }, "subject.claims.a[1]"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			code, err := rejectionOf(t, cborWith(t, tc.edit))
			if code != CodeInvalidSubjectData {
				t.Fatalf("rejection %q, want %q (%v)", code, CodeInvalidSubjectData, err)
			}
			if !strings.HasPrefix(err.(*RejectionError).Detail, tc.path+": ") {
				t.Errorf("detail %q does not name path %q", err.(*RejectionError).Detail, tc.path)
			}
		})
	}
}

func mathInf() float64 { return 1 / zero }

var zero float64

func TestParseCBOR_NonTextKeyInHashedMap(t *testing.T) {
	var top map[string]any
	if err := json.Unmarshal(readProd(t, testfixtures.ProdComplete), &top); err != nil {
		t.Fatal(err)
	}
	claims := map[any]any{"name": "x", 1: "one"}
	top["subject"].(map[string]any)["claims"] = claims
	out, err := cbor.Marshal(top)
	if err != nil {
		t.Fatal(err)
	}
	if code, _ := rejectionOf(t, out); code != CodeInvalidSubjectData {
		t.Errorf("rejection %q, want invalid_subject_data", code)
	}
}

func TestParseCBOR_ValueSpaceOutranksLayout(t *testing.T) {
	// E.6 places invalid_subject_data before unsupported_layout.
	data := cborWith(t, func(m map[string]any) {
		m["t"] = 20
		m["subject"].(map[string]any)["claims"].(map[string]any)["blob"] = []byte{1}
	})
	if code, _ := rejectionOf(t, data); code != CodeInvalidSubjectData {
		t.Errorf("rejection %q, want invalid_subject_data first", code)
	}
	data = cborWith(t, func(m map[string]any) { m["t"] = 20 })
	if code, _ := rejectionOf(t, data); code != CodeUnsupportedLayout {
		t.Errorf("rejection %q, want unsupported_layout", code)
	}
}

func TestParseCBOR_GenericPositionsAreLenient(t *testing.T) {
	// A byte string in an unnamed slot renders as hex and a tag outside a
	// hashed map is unwrapped, so an unknown optional field is carried
	// rather than refused (E.24).
	data := cborWith(t, func(m map[string]any) {
		m["future"] = []byte{0xab}
		m["tagged"] = cbor.Tag{Number: 42, Content: "x"}
	})
	b := mustParse(t, data)
	if b.Fields.Str("future") != "ab" || b.Fields.Str("tagged") != "x" {
		t.Errorf("future=%q tagged=%q", b.Fields.Str("future"), b.Fields.Str("tagged"))
	}
}

// TestParseCBOR_ByteStringInTextSlot pins that a byte string where the
// schema fixes text is not the field: the required proofs are refused by
// the same E.6 gates the reference verifier applies, and the optional
// Bitcoin evidence reads as absent.
func TestParseCBOR_ByteStringInTextSlot(t *testing.T) {
	data := cborWith(t, func(m map[string]any) { m["inclusion_proof"] = []byte{0, 1} })
	if code, _ := rejectionOf(t, data); code != CodeMissingInclusionProof {
		t.Errorf("byte-string inclusion_proof: rejection %q, want missing_inclusion_proof", code)
	}
	data = cborWith(t, func(m map[string]any) {
		m["commitments"].([]any)[0].(map[string]any)["epoch_proof"] = []byte{0}
	})
	if code, _ := rejectionOf(t, data); code != CodeInvalidCommitmentEntry {
		t.Errorf("byte-string epoch_proof: rejection %q, want invalid_commitment_entry", code)
	}
	data = cborWith(t, func(m map[string]any) {
		m["commitments"].([]any)[0].(map[string]any)["txoutproof"] = []byte{0}
		m["commitments"].([]any)[0].(map[string]any)["raw_transaction"] = []byte{0}
	})
	b := mustParse(t, data)
	if c := b.Commitments[0]; c.Txoutproof != "" || c.RawTransaction != "" || c.Fields.Has("txoutproof") {
		t.Errorf("byte-string Bitcoin evidence should read as absent, got %+v", c)
	}
}

func TestParseCBOR_DuplicateKeyIsRefused(t *testing.T) {
	// {"type":"item","type":"item"} as a bare map.
	data := []byte{0xa2, 0x64, 't', 'y', 'p', 'e', 0x64, 'i', 't', 'e', 'm', 0x64, 't', 'y', 'p', 'e', 0x64, 'i', 't', 'e', 'm'}
	code, err := rejectionOf(t, data)
	if code != CodeNotAJSONObject || !strings.Contains(err.Error(), "duplicate") {
		t.Errorf("rejection %q (%v)", code, err)
	}
}

func TestParseCBOR_TextInByteSlotPassesThrough(t *testing.T) {
	// The server passes text through unchanged in a byte-string slot; so
	// does this decoder, and the E.4 sweep then grades the text.
	data := cborWith(t, func(m map[string]any) {})
	b := mustParse(t, data)
	if b.Block.MerkleRoot != "d175d1ef04595ffbea81cf9864780889615a73bdb0184c6efa43de184f5c24a2" {
		t.Errorf("merkle_root = %q", b.Block.MerkleRoot)
	}
}

func TestJSONToCBOR_RefusesUppercaseHex(t *testing.T) {
	doc := mutate(t, func(m map[string]any) { m["block"].(map[string]any)["merkle_root"] = "D175D1EF" })
	if _, err := JSONToCBOR(doc); err == nil || !strings.Contains(err.Error(), "block.merkle_root") {
		t.Errorf("expected an error naming block.merkle_root, got %v", err)
	}
}

func TestJSONToCBOR_PreservesLargeIntegers(t *testing.T) {
	doc := mutate(t, func(m map[string]any) {
		m["subject"].(map[string]any)["claims"].(map[string]any)["big"] = json.Number("9007199254740993")
	})
	encoded, err := JSONToCBOR(doc)
	if err != nil {
		t.Fatal(err)
	}
	back, err := CBORToJSON(encoded)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(back), `"big":9007199254740993`) {
		t.Errorf("large integer not preserved: %s", back)
	}
	doc = mutate(t, func(m map[string]any) {
		m["subject"].(map[string]any)["claims"].(map[string]any)["huge"] = json.Number("184467440737095516160")
	})
	if _, err := JSONToCBOR(doc); err == nil || !strings.Contains(err.Error(), "subject.claims.huge") {
		t.Errorf("expected an error naming the key, got %v", err)
	}
}

func TestFloatLiteral(t *testing.T) {
	for f, want := range map[float64]string{1.5: "1.5", 100: "100.0", 1e21: "1e+21", 1e-7: "1e-07"} {
		if got := floatLiteral(f); got != want {
			t.Errorf("floatLiteral(%v) = %q, want %q", f, got, want)
		}
	}
}

func TestHalfToFloat64(t *testing.T) {
	cases := map[uint16]float64{0x3c00: 1, 0xc000: -2, 0x7bff: 65504, 0x0001: 5.960464477539063e-08, 0x0000: 0}
	for bits, want := range cases {
		if got := halfToFloat64(bits); got != want {
			t.Errorf("half %#04x = %v, want %v", bits, got, want)
		}
	}
}

func TestWitnessSelection(t *testing.T) {
	all, err := ParseWitnessSelection("all")
	if err != nil || !all.IsAll() || all.FilenameSuffix() != "" || all.String() != "all" {
		t.Errorf("all: %+v %v", all, err)
	}
	none, err := ParseWitnessSelection("none")
	if err != nil || !none.IsNone() || none.FilenameSuffix() != "-compact" || none.String() != "none" {
		t.Errorf("none: %+v %v", none, err)
	}
	some, err := ParseWitnessSelection("block, entropy_nist,block")
	if err != nil || some.IsAll() || some.IsNone() || some.FilenameSuffix() != "-partial" || some.String() != "block,entropy_nist" {
		t.Errorf("subset: %+v %v", some, err)
	}
	if _, err := ParseWitnessSelection("block,nope"); err == nil || !strings.Contains(err.Error(), "nope") {
		t.Errorf("unknown name accepted: %v", err)
	}
}

func TestObjectHelpers(t *testing.T) {
	o, ok := parseObject([]byte(`{"s":"x","n":5,"f":1.5,"nul":null,"o":{},"l":[1]}`))
	if !ok {
		t.Fatal("parseObject failed")
	}
	if !o.HasKey("nul") || o.Has("nul") || o.Has("missing") || !o.Has("s") {
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
