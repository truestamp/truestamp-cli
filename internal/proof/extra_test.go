// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: MIT

package proof

import (
	"encoding/hex"
	"encoding/json"
	"math"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/fxamacker/cbor/v2"
	"github.com/truestamp/truestamp-cli/internal/auth"
	"github.com/truestamp/truestamp-cli/internal/proof/ptype"
)

// TestMain installs an api-key Authorizer so Generate authorizes its
// outbound request. These tests assert no auth header, but the request
// must still be authenticated end-to-end.
func TestMain(m *testing.M) {
	auth.SetDefault(auth.APIKeyAuthorizer("key"))
	code := m.Run()
	auth.SetDefault(nil)
	os.Exit(code)
}

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

// --- binary.go helpers -----------------------------------------------------

// TestCborInteger pins the integer / non-integer split E.6 grades `t` on.
// A float is never an integer, however whole its value: truncating one let a
// CBOR `v` of 1.9 report "Proof version 1" and a `t` of 20.5 verify.
func TestCborInteger(t *testing.T) {
	t.Parallel()
	cases := []struct {
		in    any
		ok    bool
		text  string
		fits  bool
		value int64
	}{
		{uint64(42), true, "42", true, 42},
		{int64(-1), true, "-1", true, -1},
		{int(42), true, "42", true, 42},
		{uint64(math.MaxUint64), true, "18446744073709551615", false, 0},
		{float64(42), false, "", false, 0},
		{float64(1.9), false, "", false, 0},
		{"20", false, "", false, 0},
		{nil, false, "", false, 0},
	}
	for _, c := range cases {
		got, ok := cborInteger(c.in)
		if ok != c.ok {
			t.Errorf("cborInteger(%v) ok = %v, want %v", c.in, ok, c.ok)
			continue
		}
		if !ok {
			continue
		}
		if got.Text != c.text || got.Fits != c.fits || (c.fits && got.N != c.value) {
			t.Errorf("cborInteger(%v) = %+v, want text %q fits %v value %d", c.in, got, c.text, c.fits, c.value)
		}
	}
}

func TestCborIntOrZero(t *testing.T) {
	t.Parallel()
	cases := []struct {
		in   any
		want int
	}{
		{uint64(7), 7},
		{int64(7), 7},
		{float64(7), 0}, // a float is not an integer
		{"x", 0},
		{uint64(math.MaxUint64), 0}, // an integer, but not one that fits
		{nil, 0},
	}
	for _, c := range cases {
		if got := cborIntOrZero(c.in); got != c.want {
			t.Errorf("cborIntOrZero(%v) = %d, want %d", c.in, got, c.want)
		}
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

func TestCborStringField(t *testing.T) {
	m := map[string]any{
		"s":     "text",
		"b":     []byte{0x01, 0x02, 0x03},
		"empty": "",
		"n":     42,
	}
	cases := []struct {
		key  string
		want string
		ok   bool
	}{
		{"s", "text", true},
		{"b", "AQID", true}, // base64url raw
		{"empty", "", true}, // present-and-empty is still present
		{"n", "", false},
		{"missing", "", false},
	}
	for _, c := range cases {
		got, ok := cborStringField(m, c.key)
		if got != c.want || ok != c.ok {
			t.Errorf("cborStringField(%q) = (%q, %v), want (%q, %v)", c.key, got, ok, c.want, c.ok)
		}
	}
}

func TestCborFieldCarried(t *testing.T) {
	m := map[string]any{"present": "x", "empty": "", "null": nil}
	cases := []struct {
		key  string
		want bool
	}{
		{"present", true},
		{"empty", true}, // an empty string counts as carried
		{"null", false}, // a null value counts as absent
		{"missing", false},
	}
	for _, c := range cases {
		if got := cborFieldCarried(m, c.key); got != c.want {
			t.Errorf("cborFieldCarried(%q) = %v, want %v", c.key, got, c.want)
		}
	}
}

// TestBytesFieldToHex_Variants pins E.3's byte-string correspondence: only a
// CBOR byte string carries one of these fields. Accepting hex text as well
// gave every such field three spellings under one signature — bytes,
// lowercase hex text, and uppercase hex text.
func TestBytesFieldToHex_Variants(t *testing.T) {
	t.Parallel()
	m := map[any]any{
		"raw":       []byte{0xde, 0xad, 0xbe, 0xef},
		"hex":       "deadbeef",
		"upperHex":  "DEADBEEF",
		"notHex":    "Not Hex!",
		"nonString": 42,
	}
	if got := bytesFieldToHex(m, "raw"); got != "deadbeef" {
		t.Errorf("raw bytes: got %q", got)
	}
	for _, key := range []string{"hex", "upperHex", "notHex", "nonString", "missing"} {
		if got := bytesFieldToHex(m, key); got != "" {
			t.Errorf("%s: got %q, want empty — E.3 lists this field as a byte string", key, got)
		}
	}
}

func TestCborBytesToBase64_Variants(t *testing.T) {
	t.Parallel()
	m := map[string]any{"k": []byte{1, 2, 3}}
	if got := cborBytesToBase64(m, "k"); got != "AQID" {
		t.Errorf("bytes: got %q", got)
	}
	// E.3 lists pk / sig as byte strings. A text value is not one, and the
	// heuristic that used to sort text into "base64 already" and "raw bytes"
	// double-encoded correctly formed base64.
	for _, v := range []any{"Hi!", strings.Repeat("a", 100), "AQID", 42} {
		if got := cborBytesToBase64(map[string]any{"k": v}, "k"); got != "" {
			t.Errorf("text/other pk value %v: got %q, want empty", v, got)
		}
	}
	// A missing or wrong-typed pk / sig is not a rejection — E.9 and E.16
	// report it — so it yields "" instead of an error.
	if got := cborBytesToBase64(m, "missing"); got != "" {
		t.Errorf("missing key: got %q, want empty", got)
	}
}

func TestTextFieldToString(t *testing.T) {
	m := map[any]any{
		"b64url": "AQIDBA",
		"hex":    "01020304",
		"bytes":  []byte{0x01, 0x02},
		"n":      42,
	}
	// A base64url text value must survive verbatim — hex-encoding it would
	// corrupt it into hex-of-ASCII.
	if got := textFieldToString(m, "b64url"); got != "AQIDBA" {
		t.Errorf("base64url text: got %q, want AQIDBA", got)
	}
	if got := textFieldToString(m, "hex"); got != "01020304" {
		t.Errorf("hex text: got %q", got)
	}
	if got := textFieldToString(m, "bytes"); got != "0102" {
		t.Errorf("byte string: got %q", got)
	}
	if got := textFieldToString(m, "n"); got != "" {
		t.Errorf("unsupported type: got %q", got)
	}
	if got := textFieldToString(m, "missing"); got != "" {
		t.Errorf("missing: got %q", got)
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

// TestParseCBOR_MissingPKIsNotARejection mirrors the JSON path: a missing
// `pk` is not one of E.6's hard rejections, so the bundle must reach the
// report (this one still aborts, but on `s` — the first authorised gate it
// actually trips).
func TestParseCBOR_MissingPKIsNotARejection(t *testing.T) {
	m := map[string]any{"v": 1, "t": uint16(20), "sig": []byte{0x01}}
	data, err := cbor.Marshal(m)
	if err != nil {
		t.Fatal(err)
	}
	_, err = ParseCBOR(data)
	if got := RejectionCode(err); got != CodeMissingBlock {
		t.Errorf("RejectionCode = %q, want %q (err: %v)", got, CodeMissingBlock, err)
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
	data, err := Generate(srv.URL, "team", "01HJHB01T8FYZ7YTR9P5N62K5B", "auto", "json")
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
	data, err := Generate(srv.URL, "", "01HJHB01T8FYZ7YTR9P5N62K5B", "auto", "cbor")
	if err != nil {
		t.Fatalf("Generate(cbor): %v", err)
	}
	if !HasCBORTag(data) {
		t.Errorf("expected tagged CBOR bytes, got %x", data[:min(10, len(data))])
	}
}

func TestGenerateCtx_APIError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"errors":[{"detail":"bad id","title":"invalid"}]}`))
	}))
	defer srv.Close()
	_, err := Generate(srv.URL, "", "bad", "auto", "json")
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
	_, err := Generate(srv.URL, "", "id", "auto", "json")
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
	_, err := Generate(srv.URL, "", "id", "auto", "json")
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
	_, err := Generate(srv.URL, "", "id", "auto", "json")
	if err == nil {
		t.Error("expected error")
	}
}

func TestGenerateCtx_MissingResultField(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"nope": 1}`))
	}))
	defer srv.Close()
	_, err := Generate(srv.URL, "", "id", "auto", "json")
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
		// rtx / txp are absent here on purpose: E.3 makes them text
		// strings carrying base64url *or* hex, so they are emitted
		// verbatim and have no hex-decode branch to fail.
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
