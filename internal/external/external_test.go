// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: MIT

package external

import (
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestVerifyKeyring_AllKeysMatch(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(KeyringResponse{
			Version: "1.0",
			Keys: []KeyringKey{
				{KeyID: "4ceefa4a", PublicKey: "CTwMqDZnPd/QTLSq8aTeSD3a+j2DQxKcGfhhIYJQ65Y=", Sequence: 0, Active: true},
			},
		})
	}))
	defer server.Close()

	err := VerifyKeyring(map[string]string{
		"4ceefa4a": "CTwMqDZnPd/QTLSq8aTeSD3a+j2DQxKcGfhhIYJQ65Y=",
	}, server.URL)
	if err != nil {
		t.Errorf("unexpected error: %s", err)
	}
}

func TestVerifyKeyring_KeyNotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(KeyringResponse{Version: "1.0", Keys: []KeyringKey{}})
	}))
	defer server.Close()

	err := VerifyKeyring(map[string]string{"missing": "key"}, server.URL)
	if err == nil {
		t.Error("expected error for missing key")
	}
}

func TestVerifyKeyring_KeyMismatch(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(KeyringResponse{
			Version: "1.0",
			Keys:    []KeyringKey{{KeyID: "4ceefa4a", PublicKey: "WRONG_KEY_DATA"}},
		})
	}))
	defer server.Close()

	err := VerifyKeyring(map[string]string{
		"4ceefa4a": "CTwMqDZnPd/QTLSq8aTeSD3a+j2DQxKcGfhhIYJQ65Y=",
	}, server.URL)
	if err == nil {
		t.Error("expected error for key mismatch")
	}
}

func TestVerifyKeyring_ServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(500)
		w.Write([]byte("internal error"))
	}))
	defer server.Close()

	err := VerifyKeyring(map[string]string{"key": "val"}, server.URL)
	if err == nil {
		t.Error("expected error for server error")
	}
}

func TestVerifyKeyring_ConnectionRefused(t *testing.T) {
	err := VerifyKeyring(map[string]string{"key": "val"}, "http://127.0.0.1:1")
	if err == nil {
		t.Error("expected error for connection refused")
	}
}

// txBodyFor builds a Horizon transaction body that echoes txHash, the
// way the real endpoint does. Every VerifyStellar fixture must echo it:
// a response that names a different transaction is not an answer to the
// question the lookup asked.
func txBodyFor(t *testing.T, txHash, memoB64 string, ledger int, createdAt string) string {
	t.Helper()
	b, err := json.Marshal(horizonTx{
		Hash:      txHash,
		MemoType:  "hash",
		Memo:      memoB64,
		Ledger:    ledger,
		CreatedAt: createdAt,
	})
	if err != nil {
		t.Fatalf("marshalling tx fixture: %v", err)
	}
	return string(b)
}

func TestVerifyStellar_ValidMemo(t *testing.T) {
	const txHash = "aaaabbbbccccddddeeeeffffaaaabbbbccccddddeeeeffffaaaabbbbccccdddd"
	expectedMemoHash := "1bff39ef69c68841898d37d5bbc86b29b4c6f6d285ac0233817884c808a10182"
	memoBytes, _ := hex.DecodeString(expectedMemoHash)
	memoB64 := base64.StdEncoding.EncodeToString(memoBytes)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = fmt.Fprint(w, txBodyFor(t, txHash, memoB64, 1537561, "2026-04-06T23:25:06Z"))
	}))
	defer server.Close()

	// Redirect the testnet URL to our mock server. The transaction hash
	// must be a 64-char hex string.
	origTestnet := HorizonTestnetURL
	HorizonTestnetURL = server.URL
	t.Cleanup(func() { HorizonTestnetURL = origTestnet })

	result, err := VerifyStellar(txHash, expectedMemoHash, "testnet", 1537561)
	if err != nil {
		t.Fatalf("VerifyStellar: %v", err)
	}
	if result.Ledger != 1537561 {
		t.Errorf("ledger: got %d", result.Ledger)
	}
}

func TestVerifyStellar_MemoMismatch(t *testing.T) {
	txHash := strings.Repeat("a", 64)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = fmt.Fprint(w, txBodyFor(t, txHash,
			base64.StdEncoding.EncodeToString([]byte("\x11\x22\x33")), 100, ""))
	}))
	defer server.Close()
	origTestnet := HorizonTestnetURL
	HorizonTestnetURL = server.URL
	t.Cleanup(func() { HorizonTestnetURL = origTestnet })

	_, err := VerifyStellar(txHash, "deadbeef", "testnet", 100)
	if err == nil || !strings.Contains(err.Error(), "memo mismatch") {
		t.Errorf("expected memo mismatch, got %v", err)
	}
}

func TestVerifyStellar_LedgerMismatch(t *testing.T) {
	txHash := strings.Repeat("a", 64)
	memoHex := strings.Repeat("aa", 32)
	memoBytes, _ := hex.DecodeString(memoHex)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = fmt.Fprint(w, txBodyFor(t, txHash, base64.StdEncoding.EncodeToString(memoBytes), 999, ""))
	}))
	defer server.Close()
	origTestnet := HorizonTestnetURL
	HorizonTestnetURL = server.URL
	t.Cleanup(func() { HorizonTestnetURL = origTestnet })

	_, err := VerifyStellar(txHash, memoHex, "testnet", 100)
	if err == nil || !strings.Contains(err.Error(), "ledger mismatch") {
		t.Errorf("expected ledger mismatch, got %v", err)
	}
}

func TestVerifyStellar_PublicNetworkURL(t *testing.T) {
	txHash := strings.Repeat("c", 64)
	memoHex := strings.Repeat("bb", 32)
	memoBytes, _ := hex.DecodeString(memoHex)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = fmt.Fprint(w, txBodyFor(t, txHash, base64.StdEncoding.EncodeToString(memoBytes), 1, ""))
	}))
	defer server.Close()
	orig := HorizonPublicURL
	HorizonPublicURL = server.URL
	t.Cleanup(func() { HorizonPublicURL = orig })

	if _, err := VerifyStellar(txHash, memoHex, "public", 0); err != nil {
		t.Errorf("unexpected: %v", err)
	}
}

func TestVerifyStellar_InvalidHash(t *testing.T) {
	if _, err := VerifyStellar("not-hex", "deadbeef", "testnet", 0); err == nil {
		t.Error("expected invalid hash error")
	}
	if _, err := VerifyStellar(strings.Repeat("a", 10), "deadbeef", "testnet", 0); err == nil {
		t.Error("expected invalid hash length error")
	}
}

func TestVerifyStellar_WrongMemoType(t *testing.T) {
	txHash := strings.Repeat("a", 64)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode(horizonTx{Hash: txHash, MemoType: "text", Memo: "hello"})
	}))
	defer server.Close()
	origTestnet := HorizonTestnetURL
	HorizonTestnetURL = server.URL
	t.Cleanup(func() { HorizonTestnetURL = origTestnet })

	_, err := VerifyStellar(txHash, "deadbeef", "testnet", 0)
	if err == nil || !strings.Contains(err.Error(), "memo_type") {
		t.Errorf("expected memo_type error, got %v", err)
	}
}

func TestVerifyStellar_MalformedResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = fmt.Fprint(w, "not json")
	}))
	defer server.Close()
	origTestnet := HorizonTestnetURL
	HorizonTestnetURL = server.URL
	t.Cleanup(func() { HorizonTestnetURL = origTestnet })

	_, err := VerifyStellar(strings.Repeat("a", 64), "deadbeef", "testnet", 0)
	if err == nil {
		t.Error("expected parse error")
	}
}

func TestVerifyStellar_NetworkError(t *testing.T) {
	origTestnet := HorizonTestnetURL
	HorizonTestnetURL = "http://127.0.0.1:1"
	t.Cleanup(func() { HorizonTestnetURL = origTestnet })

	_, err := VerifyStellar(strings.Repeat("a", 64), "deadbeef", "testnet", 0)
	if err == nil {
		t.Error("expected network error")
	}
}

func TestVerifyStellar_InvalidMemoB64(t *testing.T) {
	txHash := strings.Repeat("a", 64)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode(horizonTx{Hash: txHash, MemoType: "hash", Memo: "!!!not-base64!!!"})
	}))
	defer server.Close()
	origTestnet := HorizonTestnetURL
	HorizonTestnetURL = server.URL
	t.Cleanup(func() { HorizonTestnetURL = origTestnet })

	_, err := VerifyStellar(txHash, "deadbeef", "testnet", 0)
	if err == nil || !strings.Contains(err.Error(), "memo base64") {
		t.Errorf("expected memo base64 error, got %v", err)
	}
}

func TestVerifyBitcoinBlock_Regtest(t *testing.T) {
	result, skipped, err := VerifyBitcoinBlock("somehash", "regtest")
	if err != nil {
		t.Errorf("unexpected error: %s", err)
	}
	if !skipped {
		t.Error("regtest should be skipped")
	}
	if result != nil {
		t.Errorf("result should be nil for skipped, got %+v", result)
	}
}

func TestVerifyBitcoinBlock_UnknownNetwork(t *testing.T) {
	_, skipped, err := VerifyBitcoinBlock("somehash", "unknown_net")
	if err != nil {
		t.Errorf("unexpected error: %s", err)
	}
	if !skipped {
		t.Error("unknown network should be skipped")
	}
}

// blockBody builds a Blockstream block body that echoes the block id it
// was asked for, the way the real endpoint does.
func blockBody(id string, height int, timestamp int64) string {
	return fmt.Sprintf(`{"id":%q,"height":%d,"timestamp":%d}`, id, height, timestamp)
}

func TestVerifyBitcoinBlock_MainnetSuccess(t *testing.T) {
	blockHash := strings.Repeat("a", 64)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = fmt.Fprint(w, blockBody(blockHash, 850000, 1700000000))
	}))
	defer server.Close()
	origMain := BlockstreamMainnetURL
	BlockstreamMainnetURL = server.URL
	t.Cleanup(func() { BlockstreamMainnetURL = origMain })

	result, skipped, err := VerifyBitcoinBlock(blockHash, "mainnet")
	if err != nil || skipped {
		t.Fatalf("unexpected: err=%v skipped=%v", err, skipped)
	}
	if result.Height != 850000 {
		t.Errorf("height: got %d", result.Height)
	}
	if result.Timestamp == "" {
		t.Error("timestamp should be populated")
	}
}

func TestVerifyBitcoinBlock_TestnetSuccess(t *testing.T) {
	blockHash := strings.Repeat("b", 64)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = fmt.Fprint(w, blockBody(blockHash, 1, 0))
	}))
	defer server.Close()
	orig := BlockstreamTestnetURL
	BlockstreamTestnetURL = server.URL
	t.Cleanup(func() { BlockstreamTestnetURL = orig })

	result, _, err := VerifyBitcoinBlock(blockHash, "testnet")
	if err != nil {
		t.Fatal(err)
	}
	if result.Timestamp != "" {
		t.Errorf("zero timestamp should render as empty string, got %q", result.Timestamp)
	}
}

// The block id is compared case-insensitively (E.4's HexEqual), so an
// endpoint that upper-cases hex still confirms the block it was asked
// for rather than reading as a different one.
func TestVerifyBitcoinBlock_BlockIDComparisonIsCaseInsensitive(t *testing.T) {
	blockHash := strings.Repeat("ab", 32)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = fmt.Fprint(w, blockBody(strings.ToUpper(blockHash), 42, 1700000000))
	}))
	defer server.Close()
	orig := BlockstreamMainnetURL
	BlockstreamMainnetURL = server.URL
	t.Cleanup(func() { BlockstreamMainnetURL = orig })

	result, _, err := VerifyBitcoinBlock(blockHash, "mainnet")
	if err != nil {
		t.Fatalf("uppercase block id should still confirm: %v", err)
	}
	if result.Height != 42 {
		t.Errorf("height: got %d, want 42", result.Height)
	}
}

// Regression, E.19(b): the binding step "MUST be confirmed against
// something outside the bundle". Before the identity guard,
// blockstreamBlock carried no `id` at all, so a 200 describing ANY other
// block was reported by the caller as `pass ... confirmed on mainnet`
// with externally_verified true — a confirmation of a binding the lookup
// never made. It must grade Malformed (skip), not OK and not Mismatch:
// the bundle supplied only the lookup key, which was sent verbatim, so a
// mis-addressed answer is an upstream property, not evidence about the
// bundle.
func TestVerifyBitcoinBlock_ResponseNamingADifferentBlockIsMalformed(t *testing.T) {
	asked := strings.Repeat("22", 32)
	other := strings.Repeat("de", 32)

	tests := []struct {
		name string
		body string
	}{
		{"different block id at the claimed height", blockBody(other, 12853, 1700000000)},
		{"no block id at all", `{"height":12853,"timestamp":1700000000}`},
		{"empty object", `{}`},
		{"null", `null`},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			srv := statusServer(t, 200, tc.body)
			orig := BlockstreamMainnetURL
			BlockstreamMainnetURL = srv.URL
			t.Cleanup(func() { BlockstreamMainnetURL = orig })

			result, skipped, err := VerifyBitcoinBlock(asked, "mainnet")
			if got := Classify(err); got != OutcomeMalformed {
				t.Errorf("Classify(%v) = %s, want malformed", err, got)
			}
			if result != nil || skipped {
				t.Errorf("no result may be returned: result=%+v skipped=%v", result, skipped)
			}
		})
	}
}

// Regression, E.19(b) other direction: `height` decoded into a plain int
// meant a 200 body that said nothing about height produced Height 0, and
// the caller failed a sound proof with "the chain reports 0" — a
// positive assertion about Bitcoin derived from silence. A pointer
// distinguishes absence from a real height of 0.
func TestVerifyBitcoinBlock_MissingHeightIsMalformedNotZero(t *testing.T) {
	blockHash := strings.Repeat("33", 32)
	srv := statusServer(t, 200, fmt.Sprintf(`{"id":%q,"timestamp":1700000000}`, blockHash))
	orig := BlockstreamMainnetURL
	BlockstreamMainnetURL = srv.URL
	t.Cleanup(func() { BlockstreamMainnetURL = orig })

	result, _, err := VerifyBitcoinBlock(blockHash, "mainnet")
	if got := Classify(err); got != OutcomeMalformed {
		t.Fatalf("Classify(%v) = %s, want malformed", err, got)
	}
	if result != nil {
		t.Errorf("a body with no height must yield no height, got %+v", result)
	}
	if !strings.Contains(err.Error(), "carries no height") {
		t.Errorf("message must name the absent field, got %q", err)
	}
}

// A real height of 0 (genesis) is a value, not an absence, and must
// survive the pointer round-trip.
func TestVerifyBitcoinBlock_ExplicitZeroHeightIsAValue(t *testing.T) {
	blockHash := strings.Repeat("44", 32)
	srv := statusServer(t, 200, blockBody(blockHash, 0, 1231006505))
	orig := BlockstreamMainnetURL
	BlockstreamMainnetURL = srv.URL
	t.Cleanup(func() { BlockstreamMainnetURL = orig })

	result, _, err := VerifyBitcoinBlock(blockHash, "mainnet")
	if err != nil {
		t.Fatalf("explicit height 0 must not be treated as absent: %v", err)
	}
	if result.Height != 0 {
		t.Errorf("height: got %d, want 0", result.Height)
	}
}

// E.5/E.19: `net` values that resolve to no public endpoint all skip,
// but the reason differs and a report must be able to say which. The
// grading (skip) is unchanged; only the explanation is now available.
func TestBitcoinNetworkSkipReason(t *testing.T) {
	tests := []struct {
		network  string
		want     string
		contains string
	}{
		{network: "mainnet", want: ""},
		{network: "testnet", want: ""},
		{network: "regtest", contains: "no public API for regtest"},
		{network: "", contains: "names no Bitcoin network"},
		{network: "bogusnet", contains: `unrecognised Bitcoin network ("bogusnet")`},
	}
	for _, tc := range tests {
		got := BitcoinNetworkSkipReason(tc.network)
		if tc.contains == "" {
			if got != tc.want {
				t.Errorf("BitcoinNetworkSkipReason(%q) = %q, want %q", tc.network, got, tc.want)
			}
			continue
		}
		if !strings.Contains(got, tc.contains) {
			t.Errorf("BitcoinNetworkSkipReason(%q) = %q, want it to contain %q", tc.network, got, tc.contains)
		}
		// Every non-lookup network must still report a skip signal, so
		// this helper can never be read as authorising a lookup.
		if _, skipped, err := VerifyBitcoinBlock(strings.Repeat("a", 64), tc.network); !skipped || err != nil {
			t.Errorf("net=%q: want skipped, got skipped=%v err=%v", tc.network, skipped, err)
		}
	}
}

func TestVerifyBitcoinBlock_InvalidHash(t *testing.T) {
	_, _, err := VerifyBitcoinBlock("not-hex", "mainnet")
	if err == nil {
		t.Error("expected error for invalid hash")
	}
}

func TestVerifyBitcoinBlock_MalformedResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = fmt.Fprint(w, "not json")
	}))
	defer server.Close()
	orig := BlockstreamMainnetURL
	BlockstreamMainnetURL = server.URL
	t.Cleanup(func() { BlockstreamMainnetURL = orig })

	_, _, err := VerifyBitcoinBlock(strings.Repeat("c", 64), "mainnet")
	if err == nil {
		t.Error("expected parse error")
	}
}

func TestVerifyBitcoinBlock_NetworkError(t *testing.T) {
	orig := BlockstreamMainnetURL
	BlockstreamMainnetURL = "http://127.0.0.1:1"
	t.Cleanup(func() { BlockstreamMainnetURL = orig })

	_, _, err := VerifyBitcoinBlock(strings.Repeat("d", 64), "mainnet")
	if err == nil {
		t.Error("expected network error")
	}
}

// --- Typed-outcome coverage (whitepaper E.17/E.18/E.21 availability grading) ---

// statusServer returns an httptest server that answers every request
// with the given status and body.
func statusServer(t *testing.T, status int, body string) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(status)
		_, _ = fmt.Fprint(w, body)
	}))
	t.Cleanup(srv.Close)
	return srv
}

// pointTestnetHorizon redirects the testnet Horizon endpoint at url for
// the duration of the test.
func pointTestnetHorizon(t *testing.T, url string) {
	t.Helper()
	orig := HorizonTestnetURL
	HorizonTestnetURL = url
	t.Cleanup(func() { HorizonTestnetURL = orig })
}

// defaultTxHash is the transaction hash the outcome-table fixtures look
// up; bodies must echo it or the identity guard grades them malformed.
var defaultTxHash = strings.Repeat("a", 64)

func memoTxBody(t *testing.T, memoHex string, ledger int) string {
	t.Helper()
	memoBytes, err := hex.DecodeString(memoHex)
	if err != nil {
		t.Fatalf("decoding memo fixture: %v", err)
	}
	return txBodyFor(t, defaultTxHash, base64.StdEncoding.EncodeToString(memoBytes), ledger, "")
}

// E.5/E.18: an absent or unrecognised `net` resolves to the default
// (testnet) Horizon instance instead of erroring before the lookup.
func TestVerifyStellar_UnnamedNetworkUsesDefaultHorizon(t *testing.T) {
	memoHex := strings.Repeat("ab", 32)
	var hits int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		hits++
		_, _ = fmt.Fprint(w, memoTxBody(t, memoHex, 42))
	}))
	defer srv.Close()
	pointTestnetHorizon(t, srv.URL)

	for _, network := range []string{"", "futurenet", "PUBLIC"} {
		result, err := VerifyStellar(defaultTxHash, memoHex, network, 42)
		if err != nil {
			t.Fatalf("net=%q: unexpected error: %v", network, err)
		}
		if result.Ledger != 42 {
			t.Errorf("net=%q: ledger got %d, want 42", network, result.Ledger)
		}
	}
	if hits != 3 {
		t.Errorf("expected 3 Horizon lookups, got %d", hits)
	}
}

// The memo comparison moved from strings.EqualFold to the constant-time
// tscrypto.HexEqual (E.4); case-insensitivity must be preserved.
func TestVerifyStellar_MemoComparisonIsCaseInsensitive(t *testing.T) {
	memoHex := strings.Repeat("ab", 32)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = fmt.Fprint(w, memoTxBody(t, memoHex, 1))
	}))
	defer srv.Close()
	pointTestnetHorizon(t, srv.URL)

	if _, err := VerifyStellar(defaultTxHash, strings.ToUpper(memoHex), "testnet", 0); err != nil {
		t.Errorf("uppercase expected memo should still match: %v", err)
	}
}

func TestVerifyStellar_Outcomes(t *testing.T) {
	memoHex := strings.Repeat("cd", 32)

	tests := []struct {
		name     string
		status   int
		body     string
		expected string
		want     Outcome
	}{
		{"not found", 404, `{"title":"Resource Missing"}`, memoHex, OutcomeNotFound},
		{"rate limited", 429, `{"title":"Rate Limit Exceeded"}`, memoHex, OutcomeUnavailable},
		{"server error", 503, `{"title":"Unavailable"}`, memoHex, OutcomeUnavailable},
		{"body not json", 200, "not json", memoHex, OutcomeMalformed},
		{"wrong memo type", 200, `{"hash":"` + defaultTxHash + `","memo_type":"text","memo":"hello"}`, memoHex, OutcomeMismatch},
		{"undecodable memo", 200, `{"hash":"` + defaultTxHash + `","memo_type":"hash","memo":"!!!not-base64!!!"}`, memoHex, OutcomeMalformed},
		{"memo mismatch", 200, memoTxBody(t, strings.Repeat("11", 32), 1), memoHex, OutcomeMismatch},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			srv := statusServer(t, tc.status, tc.body)
			pointTestnetHorizon(t, srv.URL)

			_, err := VerifyStellar(defaultTxHash, tc.expected, "testnet", 0)
			if got := Classify(err); got != tc.want {
				t.Errorf("Classify(%v) = %s, want %s", err, got, tc.want)
			}
		})
	}
}

// Regression, E.18/E.22: `if tx.MemoType != "hash"` fired identically
// for a transaction whose memo type genuinely differs and for a 2xx body
// carrying no memo_type at all, so an uninterpretable answer failed the
// proof with "memo_type mismatch: expected hash, got " — a positive
// claim about the chain from a body that contained no transaction. Every
// sibling lookup grades a missing expected field malformed.
func TestVerifyStellar_UninterpretableBodyIsMalformedNotMismatch(t *testing.T) {
	memoHex := strings.Repeat("cd", 32)

	tests := []struct {
		name string
		body string
	}{
		{"empty object", `{}`},
		{"null", `null`},
		{"no transaction in it", `{"ledger":1}`},
		{"right transaction, no memo_type", fmt.Sprintf(`{"hash":%q,"ledger":1}`, defaultTxHash)},
		{"different transaction", fmt.Sprintf(`{"hash":%q,"memo_type":"hash","memo":"AA==","ledger":1}`,
			strings.Repeat("b", 64))},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			srv := statusServer(t, 200, tc.body)
			pointTestnetHorizon(t, srv.URL)

			result, err := VerifyStellar(defaultTxHash, memoHex, "testnet", 0)
			if got := Classify(err); got != OutcomeMalformed {
				t.Errorf("Classify(%v) = %s, want malformed", err, got)
			}
			if result != nil {
				t.Errorf("no result may be returned, got %+v", result)
			}
		})
	}
}

// The other side of the same gate: a transaction that IS in the response
// and carries a non-hash memo type is a real disagreement and still
// fails. The malformed grading above must not swallow it.
func TestVerifyStellar_PresentNonHashMemoTypeStillMismatches(t *testing.T) {
	srv := statusServer(t, 200,
		fmt.Sprintf(`{"hash":%q,"memo_type":"text","memo":"hello"}`, defaultTxHash))
	pointTestnetHorizon(t, srv.URL)

	_, err := VerifyStellar(defaultTxHash, strings.Repeat("cd", 32), "testnet", 0)
	if got := Classify(err); got != OutcomeMismatch {
		t.Errorf("Classify(%v) = %s, want mismatch", err, got)
	}
}

func TestVerifyStellar_LedgerMismatchOutcome(t *testing.T) {
	memoHex := strings.Repeat("ef", 32)
	srv := statusServer(t, 200, memoTxBody(t, memoHex, 999))
	pointTestnetHorizon(t, srv.URL)

	_, err := VerifyStellar(defaultTxHash, memoHex, "testnet", 100)
	if got := Classify(err); got != OutcomeMismatch {
		t.Errorf("Classify(%v) = %s, want mismatch", err, got)
	}
}

// E.18: an entry that carries no `tx` has nothing to look up, and the
// caller must be able to tell that apart from a failed lookup.
func TestVerifyStellar_BadInputOutcomes(t *testing.T) {
	pointTestnetHorizon(t, "http://127.0.0.1:1")

	for _, txHash := range []string{"", "not-hex", strings.Repeat("a", 10)} {
		_, err := VerifyStellar(txHash, "deadbeef", "testnet", 0)
		if got := Classify(err); got != OutcomeBadInput {
			t.Errorf("tx=%q: Classify(%v) = %s, want bad_input", txHash, err, got)
		}
	}
}

func TestVerifyStellar_TransportFailureIsUnavailable(t *testing.T) {
	pointTestnetHorizon(t, "http://127.0.0.1:1")

	_, err := VerifyStellar(strings.Repeat("a", 64), "deadbeef", "testnet", 0)
	if got := Classify(err); got != OutcomeUnavailable {
		t.Errorf("Classify(%v) = %s, want unavailable", err, got)
	}
}

func TestGetStellarLedger_Success(t *testing.T) {
	srv := statusServer(t, 200, `{"sequence":56789012,"hash":"abc123","closed_at":"2026-04-06T23:25:06Z"}`)
	pointTestnetHorizon(t, srv.URL)

	// An unnamed network must still be looked up (E.5).
	ledger, err := GetStellarLedger(56789012, "")
	if err != nil {
		t.Fatalf("GetStellarLedger: %v", err)
	}
	if ledger.Hash != "abc123" || ledger.ClosedAt != "2026-04-06T23:25:06Z" {
		t.Errorf("ledger: got %+v", ledger)
	}
}

func TestGetStellarLedger_Outcomes(t *testing.T) {
	tests := []struct {
		name     string
		status   int
		body     string
		sequence int
		want     Outcome
	}{
		{"not found", 404, `{"title":"Resource Missing"}`, 1, OutcomeNotFound},
		{"server error", 500, "boom", 1, OutcomeUnavailable},
		{"body not json", 200, "not json", 1, OutcomeMalformed},
		{"no hash in response", 200, `{"sequence":1}`, 1, OutcomeMalformed},
		{"unusable sequence", 200, `{"hash":"aa"}`, 0, OutcomeBadInput},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			srv := statusServer(t, tc.status, tc.body)
			pointTestnetHorizon(t, srv.URL)

			_, err := GetStellarLedger(tc.sequence, "testnet")
			if got := Classify(err); got != tc.want {
				t.Errorf("Classify(%v) = %s, want %s", err, got, tc.want)
			}
		})
	}
}

// E.21's comparison is byte-for-byte against the value the caller looked
// up, so an answer about a different ledger would be reported as a value
// mismatch and fail a sound proof. Horizon echoes the sequence; a
// disagreement is upstream and must skip.
func TestGetStellarLedger_ResponseNamingADifferentSequenceIsMalformed(t *testing.T) {
	srv := statusServer(t, 200, `{"sequence":99,"hash":"abc123","closed_at":"2026-04-06T23:25:06Z"}`)
	pointTestnetHorizon(t, srv.URL)

	ledger, err := GetStellarLedger(56789012, "testnet")
	if got := Classify(err); got != OutcomeMalformed {
		t.Errorf("Classify(%v) = %s, want malformed", err, got)
	}
	if ledger != nil {
		t.Errorf("no ledger may be returned, got %+v", ledger)
	}
}

func TestVerifyKeyring_Outcomes(t *testing.T) {
	const keyID = "4ceefa4a"
	const pubkey = "CTwMqDZnPd/QTLSq8aTeSD3a+j2DQxKcGfhhIYJQ65Y="

	tests := []struct {
		name   string
		status int
		body   string
		want   Outcome
	}{
		{"not found", 404, "no keyring here", OutcomeNotFound},
		{"maintenance", 503, `{"error":"maintenance"}`, OutcomeUnavailable},
		{"rate limited", 429, "slow down", OutcomeUnavailable},
		{"body not json", 200, "not json", OutcomeMalformed},
		{"key absent", 200, `{"version":"1.0","keys":[]}`, OutcomeMismatch},
		{"key differs", 200, `{"version":"1.0","keys":[{"key_id":"4ceefa4a","public_key":"WRONG"}]}`, OutcomeMismatch},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			srv := statusServer(t, tc.status, tc.body)

			err := VerifyKeyring(map[string]string{keyID: pubkey}, srv.URL)
			if got := Classify(err); got != tc.want {
				t.Errorf("Classify(%v) = %s, want %s", err, got, tc.want)
			}
		})
	}
}

// Regression, E.17/E.22: a 200 that is not a keyring document at all
// used to fall through the parse guard into the per-key loop and return
// "not found in keyring" — a chain disagreement (fail) manufactured from
// a captive portal, a CDN stub or a misconfigured origin, taking a sound
// proof from exit 0 to exit 1. json.Unmarshal into KeyringResponse
// succeeds for every JSON object and leaves Keys nil, so presence of the
// `keys` list has to be tested explicitly. The reference verifier gates
// the same way and reports skip.
func TestVerifyKeyring_NotAKeyringDocumentIsMalformed(t *testing.T) {
	const keyID = "f2c39df9"
	const pubkey = "CTwMqDZnPd/QTLSq8aTeSD3a+j2DQxKcGfhhIYJQ65Y="

	for _, body := range []string{
		`{}`,
		`null`,
		`{"version":"1"}`,
		`{"ok":true}`,
		`{"keys":null}`,
		`{"version":"1.0","keys":"not-a-list"}`,
		`{"version":"1.0","keys":{"key_id":"f2c39df9"}}`,
	} {
		t.Run(body, func(t *testing.T) {
			srv := statusServer(t, 200, body)

			err := VerifyKeyring(map[string]string{keyID: pubkey}, srv.URL)
			if got := Classify(err); got != OutcomeMalformed {
				t.Errorf("Classify(%v) = %s, want malformed", err, got)
			}
			// The message must not assert that a keyring declined to
			// vouch for the key.
			if err != nil && strings.Contains(err.Error(), "not found in keyring") {
				t.Errorf("message asserts a keyring verdict that was never reached: %q", err)
			}
		})
	}
}

// The other side of the same gate: a keyring that IS published and
// genuinely does not carry the key still fails. An empty `keys` list is
// a published answer, not an unreadable one.
func TestVerifyKeyring_PublishedButEmptyListStillMismatches(t *testing.T) {
	srv := statusServer(t, 200, `{"version":"1.0","keys":[]}`)

	err := VerifyKeyring(map[string]string{"f2c39df9": "pk"}, srv.URL)
	if got := Classify(err); got != OutcomeMismatch {
		t.Errorf("Classify(%v) = %s, want mismatch", err, got)
	}
}

func TestVerifyKeyring_ConnectionRefusedIsUnavailable(t *testing.T) {
	err := VerifyKeyring(map[string]string{"k": "v"}, "http://127.0.0.1:1")
	if got := Classify(err); got != OutcomeUnavailable {
		t.Errorf("Classify(%v) = %s, want unavailable", err, got)
	}
}

// Regression: compactError used to split on the last ": " and throw the
// "HTTP 503" prefix away, leaving the user with a bare response body.
func TestVerifyKeyring_ServerErrorKeepsStatusCode(t *testing.T) {
	srv := statusServer(t, 503, `{"error":"maintenance"}`)

	err := VerifyKeyring(map[string]string{"k": "v"}, srv.URL)
	if err == nil {
		t.Fatal("expected an error")
	}
	if !strings.Contains(err.Error(), "503") {
		t.Errorf("status code must survive into the message: %q", err.Error())
	}
}

func TestGetNISTPulse_Success(t *testing.T) {
	srv := statusServer(t, 200, `{"pulse":{"chainIndex":1,"pulseIndex":2,"outputValue":"AABB","timeStamp":"2026-04-06T23:25:06.000Z","version":"2.0"}}`)
	orig := NISTBeaconURL
	NISTBeaconURL = srv.URL
	t.Cleanup(func() { NISTBeaconURL = orig })

	pulse, err := GetNISTPulse(1, 2)
	if err != nil {
		t.Fatalf("GetNISTPulse: %v", err)
	}
	if pulse.OutputValue != "AABB" {
		t.Errorf("outputValue: got %q", pulse.OutputValue)
	}
}

func TestGetNISTPulse_Outcomes(t *testing.T) {
	tests := []struct {
		name       string
		status     int
		body       string
		chainIndex int
		pulseIndex int
		want       Outcome
	}{
		{"not found", 404, "no such pulse", 1, 2, OutcomeNotFound},
		{"server error", 500, "boom", 1, 2, OutcomeUnavailable},
		{"body not json", 200, "not json", 1, 2, OutcomeMalformed},
		{"no output value", 200, `{"pulse":{"chainIndex":1}}`, 1, 2, OutcomeMalformed},
		{"unusable chain index", 200, `{}`, -1, 2, OutcomeBadInput},
		{"unusable pulse index", 200, `{}`, 1, -2, OutcomeBadInput},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			srv := statusServer(t, tc.status, tc.body)
			orig := NISTBeaconURL
			NISTBeaconURL = srv.URL
			t.Cleanup(func() { NISTBeaconURL = orig })

			_, err := GetNISTPulse(tc.chainIndex, tc.pulseIndex)
			if got := Classify(err); got != tc.want {
				t.Errorf("Classify(%v) = %s, want %s", err, got, tc.want)
			}
		})
	}
}

// Same class as the Stellar-ledger guard: the caller byte-compares
// outputValue and timeStamp against the entropy subject, so a pulse
// other than the one requested would be reported as a value mismatch and
// fail a sound proof.
func TestGetNISTPulse_ResponseNamingADifferentPulseIsMalformed(t *testing.T) {
	tests := []struct {
		name string
		body string
	}{
		{"different pulse index", `{"pulse":{"chainIndex":1,"pulseIndex":9,"outputValue":"AABB","timeStamp":"2026-04-06T23:25:06.000Z"}}`},
		{"different chain index", `{"pulse":{"chainIndex":7,"pulseIndex":2,"outputValue":"AABB","timeStamp":"2026-04-06T23:25:06.000Z"}}`},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			srv := statusServer(t, 200, tc.body)
			orig := NISTBeaconURL
			NISTBeaconURL = srv.URL
			t.Cleanup(func() { NISTBeaconURL = orig })

			pulse, err := GetNISTPulse(1, 2)
			if got := Classify(err); got != OutcomeMalformed {
				t.Errorf("Classify(%v) = %s, want malformed", err, got)
			}
			if pulse != nil {
				t.Errorf("no pulse may be returned, got %+v", pulse)
			}
		})
	}
}

func TestGetNISTPulse_TransportFailureIsUnavailable(t *testing.T) {
	orig := NISTBeaconURL
	NISTBeaconURL = "http://127.0.0.1:1"
	t.Cleanup(func() { NISTBeaconURL = orig })

	_, err := GetNISTPulse(1, 2)
	if got := Classify(err); got != OutcomeUnavailable {
		t.Errorf("Classify(%v) = %s, want unavailable", err, got)
	}
}

func TestGetBitcoinBlockHeader_Success(t *testing.T) {
	srv := statusServer(t, 200, `{"id":"00ff","height":850000,"timestamp":1700000000,"merkle_root":"aabb"}`)
	orig := BlockstreamMainnetURL
	BlockstreamMainnetURL = srv.URL
	t.Cleanup(func() { BlockstreamMainnetURL = orig })

	header, skipped, err := GetBitcoinBlockHeader(strings.Repeat("a", 64), "mainnet")
	if err != nil || skipped {
		t.Fatalf("unexpected: err=%v skipped=%v", err, skipped)
	}
	if header.Height != 850000 || header.Hash != "00ff" {
		t.Errorf("header: got %+v", header)
	}
}

func TestGetBitcoinBlockHeader_Outcomes(t *testing.T) {
	tests := []struct {
		name   string
		status int
		body   string
		hash   string
		want   Outcome
	}{
		{"not found", 404, "Block not found", strings.Repeat("a", 64), OutcomeNotFound},
		{"server error", 502, "<html>bad gateway</html>", strings.Repeat("a", 64), OutcomeUnavailable},
		{"body not json", 200, "not json", strings.Repeat("a", 64), OutcomeMalformed},
		{"no block id", 200, `{"height":1}`, strings.Repeat("a", 64), OutcomeMalformed},
		{"unusable hash", 200, `{"id":"00ff"}`, "not-hex", OutcomeBadInput},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			srv := statusServer(t, tc.status, tc.body)
			orig := BlockstreamMainnetURL
			BlockstreamMainnetURL = srv.URL
			t.Cleanup(func() { BlockstreamMainnetURL = orig })

			_, _, err := GetBitcoinBlockHeader(tc.hash, "mainnet")
			if got := Classify(err); got != tc.want {
				t.Errorf("Classify(%v) = %s, want %s", err, got, tc.want)
			}
		})
	}
}

// regtest has no public API: still a plain skip signal, not an error.
func TestGetBitcoinBlockHeader_RegtestSkips(t *testing.T) {
	header, skipped, err := GetBitcoinBlockHeader(strings.Repeat("a", 64), "regtest")
	if err != nil || !skipped || header != nil {
		t.Errorf("regtest: got header=%+v skipped=%v err=%v", header, skipped, err)
	}
}

func TestVerifyBitcoinBlock_Outcomes(t *testing.T) {
	tests := []struct {
		name   string
		status int
		body   string
		hash   string
		want   Outcome
	}{
		{"not found", 404, "Block not found", strings.Repeat("a", 64), OutcomeNotFound},
		{"server error", 500, "boom", strings.Repeat("a", 64), OutcomeUnavailable},
		{"body not json", 200, "not json", strings.Repeat("a", 64), OutcomeMalformed},
		{"unusable hash", 200, `{"height":1}`, "not-hex", OutcomeBadInput},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			srv := statusServer(t, tc.status, tc.body)
			orig := BlockstreamMainnetURL
			BlockstreamMainnetURL = srv.URL
			t.Cleanup(func() { BlockstreamMainnetURL = orig })

			_, _, err := VerifyBitcoinBlock(tc.hash, "mainnet")
			if got := Classify(err); got != tc.want {
				t.Errorf("Classify(%v) = %s, want %s", err, got, tc.want)
			}
		})
	}
}
