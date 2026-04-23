// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: MIT

package proof

import (
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/fxamacker/cbor/v2"
	"github.com/truestamp/truestamp-cli/internal/proof/ptype"
)

// --- id.go (DetectIDType) --------------------------------------------------

func TestDetectIDType(t *testing.T) {
	cases := []struct {
		in   string
		want IDType
		ok   bool
	}{
		{"01HJHB01T8FYZ7YTR9P5N62K5B", IDTypeULID, true},
		{"01kn3ahv5gmc7y9z9y0s6r90p5", IDTypeULID, true}, // ULID is case-insensitive via upper
		{"019cf813-99b8-730a-84f1-5a711a9c355e", IDTypeUUIDv7, true},
		{"019d6a3213e672b097e53779231ea97b", IDTypeUUIDv7, true}, // no hyphens
		{"not-an-id", "", false},
		{"", "", false},
	}
	for _, c := range cases {
		got, err := DetectIDType(c.in)
		if c.ok && (err != nil || got != c.want) {
			t.Errorf("DetectIDType(%q) = (%q, %v), want (%q, nil)", c.in, got, err, c.want)
		}
		if !c.ok && err == nil {
			t.Errorf("DetectIDType(%q) should have errored", c.in)
		}
	}
}

// --- types.go (FindCommitByType) -------------------------------------------

func TestFindCommitByType(t *testing.T) {
	commits := []ExternalCommit{
		{Type: ptype.CommitmentStellar, TransactionHash: "aa"},
		{Type: ptype.CommitmentBitcoin, TransactionHash: "bb"},
	}
	if got := FindCommitByType(commits, ptype.CommitmentStellar); got == nil || got.TransactionHash != "aa" {
		t.Errorf("FindCommitByType(stellar) = %+v", got)
	}
	if got := FindCommitByType(commits, ptype.CommitmentBitcoin); got == nil || got.TransactionHash != "bb" {
		t.Errorf("FindCommitByType(bitcoin) = %+v", got)
	}
	if got := FindCommitByType(commits, ptype.Code(99)); got != nil {
		t.Errorf("FindCommitByType(other) should return nil, got %+v", got)
	}
	if got := FindCommitByType(nil, ptype.CommitmentStellar); got != nil {
		t.Errorf("FindCommitByType(nil, ...) should return nil")
	}
}

// --- binary.go helpers -----------------------------------------------------

func TestIsHexString(t *testing.T) {
	cases := []struct {
		in string
		ok bool
	}{
		{"", false},
		{"a", false}, // odd
		{"ab", true},
		{"AB", true},
		{"FF", true},
		{"ff00aa", true},
		{"zz", false},
		{"ab cd", false}, // space
	}
	for _, c := range cases {
		if got := isHexString(c.in); got != c.ok {
			t.Errorf("isHexString(%q) = %v, want %v", c.in, got, c.ok)
		}
	}
}

func TestToInt(t *testing.T) {
	cases := []struct {
		in   any
		want int
	}{
		{uint64(42), 42},
		{int64(42), 42},
		{float64(42), 42},
		{int(42), 42},
		{"string", 0}, // unknown type → 0
		{nil, 0},
	}
	for _, c := range cases {
		if got := toInt(c.in); got != c.want {
			t.Errorf("toInt(%v) = %d, want %d", c.in, got, c.want)
		}
	}
}

func TestGetInt(t *testing.T) {
	m := map[any]any{"a": uint64(7), "b": "x"}
	if got := getInt(m, "a"); got != 7 {
		t.Errorf("getInt present: got %d", got)
	}
	if got := getInt(m, "missing"); got != 0 {
		t.Errorf("getInt missing: got %d", got)
	}
}

func TestGetString(t *testing.T) {
	m := map[any]any{"a": "yes", "b": 1}
	if got := getString(m, "a"); got != "yes" {
		t.Errorf("getString present: got %q", got)
	}
	if got := getString(m, "missing"); got != "" {
		t.Errorf("getString missing: got %q", got)
	}
}

func TestGetStringField(t *testing.T) {
	m := map[string]any{
		"s": "text",
		"b": []byte{0x01, 0x02, 0x03},
		"n": 42,
	}
	if got := getStringField(m, "s"); got != "text" {
		t.Errorf("string: got %q", got)
	}
	if got := getStringField(m, "b"); got != "AQID" { // base64url raw
		t.Errorf("bytes: got %q", got)
	}
	if got := getStringField(m, "n"); got != "" {
		t.Errorf("unsupported type should yield empty, got %q", got)
	}
	if got := getStringField(m, "missing"); got != "" {
		t.Errorf("missing: got %q", got)
	}
}

func TestBytesFieldToHex_Variants(t *testing.T) {
	m := map[any]any{
		"raw":       []byte{0xde, 0xad, 0xbe, 0xef},
		"hex":       "deadbeef",
		"notHex":    "Not Hex!",
		"nonString": 42,
	}
	if got := bytesFieldToHex(m, "raw"); got != "deadbeef" {
		t.Errorf("raw bytes: got %q", got)
	}
	if got := bytesFieldToHex(m, "hex"); got != "deadbeef" {
		t.Errorf("hex passthrough: got %q", got)
	}
	// Non-hex string becomes hex-encoded bytes
	if got := bytesFieldToHex(m, "notHex"); got == "" {
		t.Error("non-hex string should be hex-encoded")
	}
	if got := bytesFieldToHex(m, "nonString"); got != "" {
		t.Errorf("non-string type should yield empty, got %q", got)
	}
	if got := bytesFieldToHex(m, "missing"); got != "" {
		t.Errorf("missing: got %q", got)
	}
}

func TestBytesToBase64_Variants(t *testing.T) {
	// []byte input
	m := map[string]any{"k": []byte{1, 2, 3}}
	if got, _ := bytesToBase64(m, "k"); got != "AQID" {
		t.Errorf("bytes: got %q", got)
	}
	// short string (<=64) that isn't hex → treated as raw bytes
	m2 := map[string]any{"k": "Hi!"}
	if got, _ := bytesToBase64(m2, "k"); got != "SGkh" {
		t.Errorf("short non-hex string: got %q", got)
	}
	// long string → passthrough
	m3 := map[string]any{"k": strings.Repeat("a", 100)}
	got, _ := bytesToBase64(m3, "k")
	if got != strings.Repeat("a", 100) {
		t.Errorf("long string should pass through, got %q", got)
	}
	// missing key
	if _, err := bytesToBase64(m, "missing"); err == nil {
		t.Error("missing key should error")
	}
	// unsupported type
	m4 := map[string]any{"k": 42}
	if _, err := bytesToBase64(m4, "k"); err == nil {
		t.Error("unsupported type should error")
	}
}

func TestNormalizeMap(t *testing.T) {
	// Already map[any]any
	in1 := map[any]any{"a": 1}
	if got := normalizeMap(in1); got == nil {
		t.Error("map[any]any should pass through")
	}
	// map[string]any → converted
	in2 := map[string]any{"a": 1}
	if got := normalizeMap(in2); got == nil {
		t.Error("map[string]any should convert")
	}
	// non-map
	if got := normalizeMap("not a map"); got != nil {
		t.Errorf("non-map: got %+v, want nil", got)
	}
}

func TestToAnyKeyMap(t *testing.T) {
	in := map[string]any{"a": 1, "b": 2}
	out := toAnyKeyMap(in)
	if out["a"] != 1 || out["b"] != 2 {
		t.Errorf("toAnyKeyMap: got %+v", out)
	}
}

// --- ParseCBOR error branches ---------------------------------------------

func TestParseCBOR_Empty(t *testing.T) {
	if _, err := ParseCBOR(nil); err == nil {
		t.Error("empty CBOR should error")
	}
}

func TestParseCBOR_Malformed(t *testing.T) {
	if _, err := ParseCBOR([]byte{0x00, 0x01, 0x02}); err == nil {
		t.Error("malformed CBOR should error")
	}
}

func TestParseCBOR_MissingPK(t *testing.T) {
	// Valid CBOR map with v+t but without pk.
	m := map[string]any{"v": 1, "t": uint16(20), "sig": []byte{0x01}}
	data, err := cbor.Marshal(m)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := ParseCBOR(data); err == nil {
		t.Error("missing pk should error")
	}
}

// --- MarshalCBOR error branches -------------------------------------------

func TestMarshalCBOR_BadPublicKey(t *testing.T) {
	b, err := ParseBytes([]byte(validProofJSON))
	if err != nil {
		t.Fatal(err)
	}
	// Corrupt pk so base64 decode fails.
	b.PublicKey = "not-valid-base64!!!"
	if _, err := b.MarshalCBOR(); err == nil {
		t.Error("bad pk should error")
	}
}

func TestMarshalCBOR_BadSignature(t *testing.T) {
	b, err := ParseBytes([]byte(validProofJSON))
	if err != nil {
		t.Fatal(err)
	}
	b.Signature = "!!!"
	if _, err := b.MarshalCBOR(); err == nil {
		t.Error("bad sig should error")
	}
}

func TestMarshalCBOR_BadSubjectHex(t *testing.T) {
	b, err := ParseBytes([]byte(validProofJSON))
	if err != nil {
		t.Fatal(err)
	}
	b.Subject.MetadataHash = "zzzz"
	if _, err := b.MarshalCBOR(); err == nil {
		t.Error("bad mh should error")
	}
}

func TestMarshalCBOR_BadBlockHex(t *testing.T) {
	b, err := ParseBytes([]byte(validProofJSON))
	if err != nil {
		t.Fatal(err)
	}
	b.Block.PreviousBlockHash = "zz"
	if _, err := b.MarshalCBOR(); err == nil {
		t.Error("bad block.ph should error")
	}
}

func TestMarshalCBOR_BadCommitHex(t *testing.T) {
	b, err := ParseBytes([]byte(validProofJSON))
	if err != nil {
		t.Fatal(err)
	}
	b.Commitments[0].TransactionHash = "zz"
	if _, err := b.MarshalCBOR(); err == nil {
		t.Error("bad commit tx hex should error")
	}
}

func TestMarshalCBOR_BadInclusionProof(t *testing.T) {
	b, err := ParseBytes([]byte(validProofJSON))
	if err != nil {
		t.Fatal(err)
	}
	b.InclusionProof = "!!!"
	if _, err := b.MarshalCBOR(); err == nil {
		t.Error("bad ip base64url should error")
	}
}

func TestMarshalCBOR_SubjectDataInvalidJSON(t *testing.T) {
	b, err := ParseBytes([]byte(validProofJSON))
	if err != nil {
		t.Fatal(err)
	}
	// Corrupt RawData to invalid JSON.
	b.RawData = json.RawMessage("not json")
	b.Subject.Data = json.RawMessage("not json")
	if _, err := b.MarshalCBOR(); err == nil {
		t.Error("bad subject.d JSON should error")
	}
}

// TestMarshalCBOR_EmptyHexFields exercises the len(s) == 0 early return
// in decodeHexOrBytes / decodeB64URLOrBytes (unset optional fields).
func TestMarshalCBOR_EmptyHexFields(t *testing.T) {
	b, err := ParseBytes([]byte(validProofJSON))
	if err != nil {
		t.Fatal(err)
	}
	// Clear everything optional.
	b.Block.MetadataHash = ""
	b.Subject.MetadataHash = ""
	b.InclusionProof = ""
	out, err := b.MarshalCBOR()
	if err != nil {
		t.Fatalf("expected success with empty optional fields: %v", err)
	}
	if len(out) == 0 {
		t.Error("empty output")
	}
}

// TestMarshalCBOR_BadBlockMerkleRoot covers the mr hex decode error branch
// in blockToCBORMap (specific to the merkle_root field vs previous_block).
func TestMarshalCBOR_BadBlockMerkleRoot(t *testing.T) {
	b, err := ParseBytes([]byte(validProofJSON))
	if err != nil {
		t.Fatal(err)
	}
	b.Block.MerkleRoot = "zz"
	if _, err := b.MarshalCBOR(); err == nil {
		t.Error("bad mr should error")
	}
}

// TestMarshalCBOR_BadBlockMH covers the block.mh hex-decode error branch.
func TestMarshalCBOR_BadBlockMH(t *testing.T) {
	b, err := ParseBytes([]byte(validProofJSON))
	if err != nil {
		t.Fatal(err)
	}
	b.Block.MetadataHash = "zz"
	if _, err := b.MarshalCBOR(); err == nil {
		t.Error("bad block.mh should error")
	}
}

// TestMarshalCBOR_BadBlockKID covers block.kid hex-decode error branch.
func TestMarshalCBOR_BadBlockKID(t *testing.T) {
	b, err := ParseBytes([]byte(validProofJSON))
	if err != nil {
		t.Fatal(err)
	}
	b.Block.SigningKeyID = "zz"
	if _, err := b.MarshalCBOR(); err == nil {
		t.Error("bad block.kid should error")
	}
}

// TestMarshalCBOR_BadSubjectKID covers subject.kid hex-decode error branch.
func TestMarshalCBOR_BadSubjectKID(t *testing.T) {
	b, err := ParseBytes([]byte(validProofJSON))
	if err != nil {
		t.Fatal(err)
	}
	b.Subject.SigningKeyID = "zz"
	if _, err := b.MarshalCBOR(); err == nil {
		t.Error("bad subject.kid should error")
	}
}

func TestMarshalCBOR_BundleWithRawDataEmpty(t *testing.T) {
	// Parse then null out RawData to take the subject.Data fallback branch.
	b, err := ParseBytes([]byte(validProofJSON))
	if err != nil {
		t.Fatal(err)
	}
	b.RawData = nil
	out, err := b.MarshalCBOR()
	if err != nil {
		t.Fatalf("expected success via subject.Data fallback: %v", err)
	}
	if len(out) == 0 {
		t.Error("empty output")
	}
}

// --- download.go (Download / DownloadCtx / GenerateCtx) -------------------

func TestDownload_RejectNonHTTP(t *testing.T) {
	if _, err := Download("file:///etc/passwd"); err == nil {
		t.Error("non-http(s) URL should error")
	}
}

func TestDownload_NotAProof(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"something":"else"}`))
	}))
	defer srv.Close()
	if _, err := Download(srv.URL); err == nil {
		t.Error("non-proof JSON should be rejected")
	}
}

func TestDownload_EmptyHost(t *testing.T) {
	if _, err := Download("https:///path"); err == nil {
		t.Error("empty host should error")
	}
}

func TestGenerateCtx_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Respond with the wrapped envelope the client expects.
		_, _ = w.Write([]byte(`{"result":` + validProofJSON + `}`))
	}))
	defer srv.Close()
	data, err := Generate(srv.URL, "key", "team", "01HJHB01T8FYZ7YTR9P5N62K5B", "auto", "json")
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	if len(data) == 0 {
		t.Error("empty result")
	}
}

func TestGenerateCtx_CBORSuccess(t *testing.T) {
	// Build a base64-encoded CBOR payload (just any bytes).
	rawCBOR := []byte{0xd9, 0xd9, 0xf7, 0xa1, 0x61, 0x76, 0x01}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"result":"` + encodeBase64Std(rawCBOR) + `"}`))
	}))
	defer srv.Close()
	data, err := Generate(srv.URL, "k", "", "01HJHB01T8FYZ7YTR9P5N62K5B", "auto", "cbor")
	if err != nil {
		t.Fatalf("Generate(cbor): %v", err)
	}
	if !IsCBORProof(data) {
		t.Errorf("expected CBOR bytes, got %x", data[:min(10, len(data))])
	}
}

func TestGenerateCtx_APIError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"errors":[{"detail":"bad id","title":"invalid"}]}`))
	}))
	defer srv.Close()
	_, err := Generate(srv.URL, "k", "", "bad", "auto", "json")
	if err == nil {
		t.Error("expected API error")
	}
	if !strings.Contains(err.Error(), "bad id") {
		t.Errorf("error should include detail, got: %v", err)
	}
}

func TestGenerateCtx_APIErrorHTMLPage(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte("<html>oops</html>"))
	}))
	defer srv.Close()
	_, err := Generate(srv.URL, "k", "", "id", "auto", "json")
	if err == nil {
		t.Error("expected error for HTML response")
	}
	if !strings.Contains(err.Error(), "HTML") {
		t.Errorf("expected HTML hint in error, got: %v", err)
	}
}

func TestGenerateCtx_APIErrorTitleOnly(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"errors":[{"title":"Not Found"}]}`))
	}))
	defer srv.Close()
	_, err := Generate(srv.URL, "k", "", "id", "auto", "json")
	if err == nil {
		t.Error("expected error")
	}
	if !strings.Contains(err.Error(), "Not Found") {
		t.Errorf("expected title in error: %v", err)
	}
}

func TestGenerateCtx_APIErrorUnparseable(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadGateway)
		_, _ = w.Write([]byte("upstream exploded"))
	}))
	defer srv.Close()
	_, err := Generate(srv.URL, "k", "", "id", "auto", "json")
	if err == nil {
		t.Error("expected error")
	}
}

func TestGenerateCtx_MissingResultField(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"nope": 1}`))
	}))
	defer srv.Close()
	_, err := Generate(srv.URL, "k", "", "id", "auto", "json")
	if err == nil {
		t.Error("expected error for missing result")
	}
}

// --- decodeCommitsCBOR with malformed input ------------------------------

func TestDecodeCommitsCBOR_NotArray(t *testing.T) {
	if _, err := decodeCommitsCBOR("string not array"); err == nil {
		t.Error("non-array should error")
	}
}

func TestDecodeCommitsCBOR_Nil(t *testing.T) {
	out, err := decodeCommitsCBOR(nil)
	if err != nil || out != nil {
		t.Errorf("nil input: got %+v, %v", out, err)
	}
}

func TestDecodeCommitsCBOR_ElementNotMap(t *testing.T) {
	if _, err := decodeCommitsCBOR([]any{"string"}); err == nil {
		t.Error("array of non-maps should error")
	}
}

// TestMarshalCBOR_BitcoinCommit exercises the commitsToCBOR branches
// that only fire for bitcoin entries (op/rtx/txp/bmr/h fields).
func TestMarshalCBOR_BitcoinCommit(t *testing.T) {
	const proofWithBitcoin = `{
	  "v": 1,
	  "t": 20,
	  "pk": "CTwMqDZnPd/QTLSq8aTeSD3a+j2DQxKcGfhhIYJQ65Y=",
	  "sig": "AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA==",
	  "ts": "2026-04-06T23:25:06Z",
	  "s": {
	    "id": "01HJHB01T8FYZ7YTR9P5N62K5B",
	    "d": {"name": "test"},
	    "mh": "ccddccddccddccddccddccddccddccddccddccddccddccddccddccddccddccdd",
	    "kid": "4ceefa4a"
	  },
	  "b": {
	    "id": "019cf813-99b8-730a-84f1-5a711a9c355e",
	    "ph": "1111111111111111111111111111111111111111111111111111111111111111",
	    "mr": "2222222222222222222222222222222222222222222222222222222222222222",
	    "mh": "4444444444444444444444444444444444444444444444444444444444444444",
	    "kid": "4ceefa4a"
	  },
	  "ip": "AA",
	  "cx": [
	    {
	      "t": 41,
	      "net": "regtest",
	      "ep": "AA",
	      "tx": "eeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeee",
	      "op": "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
	      "rtx": "bbbbbbbb",
	      "txp": "cccccccc",
	      "bmr": "dddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddd",
	      "h": 850000,
	      "ts": "2026-04-06T23:25:06Z"
	    }
	  ]
	}`
	b, err := ParseBytes([]byte(proofWithBitcoin))
	if err != nil {
		t.Fatal(err)
	}
	data, err := b.MarshalCBOR()
	if err != nil {
		t.Fatalf("MarshalCBOR: %v", err)
	}
	// Round-trip must survive.
	b2, err := ParseCBOR(data)
	if err != nil {
		t.Fatalf("ParseCBOR: %v", err)
	}
	if b2.Commitments[0].BlockHeight != 850000 {
		t.Errorf("round-trip lost height: %+v", b2.Commitments[0])
	}
	if b2.Commitments[0].RawTxHex == "" {
		t.Error("round-trip lost rtx")
	}
}

func TestMarshalCBOR_BadCommitFields(t *testing.T) {
	// Build a bitcoin commit with one bad hex field at a time to exercise
	// each error branch in commitsToCBOR.
	base := func() *ProofBundle {
		b, _ := ParseBytes([]byte(validProofJSON))
		b.Commitments[0].Type = ptype.CommitmentBitcoin
		b.Commitments[0].Network = "regtest"
		b.Commitments[0].OpReturn = "aa"
		b.Commitments[0].RawTxHex = "bb"
		b.Commitments[0].TxoutproofHex = "cc"
		b.Commitments[0].BlockMerkleRoot = "dd"
		b.Commitments[0].BlockHeight = 10
		return b
	}

	type mut func(*ProofBundle)
	cases := []struct {
		name string
		f    mut
	}{
		{"op", func(b *ProofBundle) { b.Commitments[0].OpReturn = "zz" }},
		{"rtx", func(b *ProofBundle) { b.Commitments[0].RawTxHex = "zz" }},
		{"txp", func(b *ProofBundle) { b.Commitments[0].TxoutproofHex = "zz" }},
		{"bmr", func(b *ProofBundle) { b.Commitments[0].BlockMerkleRoot = "zz" }},
		{"memo", func(b *ProofBundle) { b.Commitments[0].MemoHash = "zz" }},
		{"ep", func(b *ProofBundle) { b.Commitments[0].EpochProof = "!!!" }},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			b := base()
			c.f(b)
			if _, err := b.MarshalCBOR(); err == nil {
				t.Errorf("expected error for bad %s hex", c.name)
			}
		})
	}
}

// TestConvertForJSON_AllBranches feeds convertForJSON a value with all
// the branch types it handles: nested map[any]any, map[string]any,
// []any, []byte. The MarshalJSON path is what actually calls
// convertForJSON via anyToJSON → so we go through MarshalCBOR
// followed by ParseCBOR which round-trips via convertForJSON.
func TestConvertForJSON_AllBranches(t *testing.T) {
	const proof = `{
	  "v": 1,
	  "t": 20,
	  "pk": "CTwMqDZnPd/QTLSq8aTeSD3a+j2DQxKcGfhhIYJQ65Y=",
	  "sig": "AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA==",
	  "ts": "2026-04-06T23:25:06Z",
	  "s": {
	    "id": "01HJHB01T8FYZ7YTR9P5N62K5B",
	    "d": {
	      "nested": {"deep": "value"},
	      "list": [1, 2, {"a": "b"}],
	      "leaf": "plain"
	    },
	    "mh": "cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc",
	    "kid": "4c4c4c4c"
	  },
	  "b": {
	    "id":"019cf813-99b8-730a-84f1-5a711a9c355e",
	    "mr":"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
	    "mh":"bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
	    "kid":"cccccccc",
	    "ph":"dddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddd"
	  },
	  "ip": "AA",
	  "cx": [{"t":40,"net":"testnet","ep":"AA",
	    "memo":"abababababababababababababababababababababababababababababababab",
	    "tx":"cdcdcdcdcdcdcdcdcdcdcdcdcdcdcdcdcdcdcdcdcdcdcdcdcdcdcdcdcdcdcdcd",
	    "l":1}]
	}`
	b, err := ParseBytes([]byte(proof))
	if err != nil {
		t.Fatal(err)
	}
	cborBytes, err := b.MarshalCBOR()
	if err != nil {
		t.Fatalf("MarshalCBOR: %v", err)
	}
	b2, err := ParseCBOR(cborBytes)
	if err != nil {
		t.Fatalf("ParseCBOR: %v", err)
	}
	// b2.RawData comes from anyToJSON → convertForJSON; look for markers.
	raw := string(b2.RawData)
	for _, want := range []string{"nested", "deep", "list", "a"} {
		if !strings.Contains(raw, want) {
			t.Errorf("RawData missing %q, got: %s", want, raw)
		}
	}
}

// --- Helpers -------------------------------------------------------------

func encodeBase64Std(data []byte) string {
	// Standalone so the test file doesn't depend on internal/encoding.
	const chars = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789+/"
	n := len(data)
	var out strings.Builder
	for i := 0; i < n; i += 3 {
		var b [3]byte
		copy(b[:], data[i:min(i+3, n)])
		ch := n - i
		out.WriteByte(chars[b[0]>>2])
		out.WriteByte(chars[((b[0]&0x03)<<4)|((b[1]&0xf0)>>4)])
		if ch > 1 {
			out.WriteByte(chars[((b[1]&0x0f)<<2)|((b[2]&0xc0)>>6)])
		} else {
			out.WriteByte('=')
		}
		if ch > 2 {
			out.WriteByte(chars[b[2]&0x3f])
		} else {
			out.WriteByte('=')
		}
	}
	return out.String()
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// Silence unused-import warning when all tests compile.
var _ = hex.EncodeToString
