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

func TestVerifyStellar_ValidMemo(t *testing.T) {
	expectedMemoHash := "1bff39ef69c68841898d37d5bbc86b29b4c6f6d285ac0233817884c808a10182"
	memoBytes, _ := hex.DecodeString(expectedMemoHash)
	memoB64 := base64.StdEncoding.EncodeToString(memoBytes)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(horizonTx{
			MemoType:  "hash",
			Memo:      memoB64,
			Ledger:    1537561,
			CreatedAt: "2026-04-06T23:25:06Z",
		})
	}))
	defer server.Close()

	// Redirect the testnet URL to our mock server. The transaction hash
	// must be a 64-char hex string.
	origTestnet := HorizonTestnetURL
	HorizonTestnetURL = server.URL
	t.Cleanup(func() { HorizonTestnetURL = origTestnet })

	result, err := VerifyStellar(
		"aaaabbbbccccddddeeeeffffaaaabbbbccccddddeeeeffffaaaabbbbccccdddd",
		expectedMemoHash, "testnet", 1537561)
	if err != nil {
		t.Fatalf("VerifyStellar: %v", err)
	}
	if result.Ledger != 1537561 {
		t.Errorf("ledger: got %d", result.Ledger)
	}
}

func TestVerifyStellar_MemoMismatch(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode(horizonTx{
			MemoType: "hash",
			Memo:     base64.StdEncoding.EncodeToString([]byte("\x11\x22\x33")),
			Ledger:   100,
		})
	}))
	defer server.Close()
	origTestnet := HorizonTestnetURL
	HorizonTestnetURL = server.URL
	t.Cleanup(func() { HorizonTestnetURL = origTestnet })

	_, err := VerifyStellar(
		strings.Repeat("a", 64), "deadbeef", "testnet", 100)
	if err == nil || !strings.Contains(err.Error(), "memo mismatch") {
		t.Errorf("expected memo mismatch, got %v", err)
	}
}

func TestVerifyStellar_LedgerMismatch(t *testing.T) {
	memoHex := strings.Repeat("aa", 32)
	memoBytes, _ := hex.DecodeString(memoHex)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode(horizonTx{
			MemoType: "hash",
			Memo:     base64.StdEncoding.EncodeToString(memoBytes),
			Ledger:   999,
		})
	}))
	defer server.Close()
	origTestnet := HorizonTestnetURL
	HorizonTestnetURL = server.URL
	t.Cleanup(func() { HorizonTestnetURL = origTestnet })

	_, err := VerifyStellar(strings.Repeat("a", 64), memoHex, "testnet", 100)
	if err == nil || !strings.Contains(err.Error(), "ledger mismatch") {
		t.Errorf("expected ledger mismatch, got %v", err)
	}
}

func TestVerifyStellar_PublicNetworkURL(t *testing.T) {
	memoHex := strings.Repeat("bb", 32)
	memoBytes, _ := hex.DecodeString(memoHex)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode(horizonTx{MemoType: "hash", Memo: base64.StdEncoding.EncodeToString(memoBytes), Ledger: 1})
	}))
	defer server.Close()
	orig := HorizonPublicURL
	HorizonPublicURL = server.URL
	t.Cleanup(func() { HorizonPublicURL = orig })

	if _, err := VerifyStellar(strings.Repeat("c", 64), memoHex, "public", 0); err != nil {
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
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode(horizonTx{MemoType: "text", Memo: "hello"})
	}))
	defer server.Close()
	origTestnet := HorizonTestnetURL
	HorizonTestnetURL = server.URL
	t.Cleanup(func() { HorizonTestnetURL = origTestnet })

	_, err := VerifyStellar(strings.Repeat("a", 64), "deadbeef", "testnet", 0)
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
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode(horizonTx{MemoType: "hash", Memo: "!!!not-base64!!!"})
	}))
	defer server.Close()
	origTestnet := HorizonTestnetURL
	HorizonTestnetURL = server.URL
	t.Cleanup(func() { HorizonTestnetURL = origTestnet })

	_, err := VerifyStellar(strings.Repeat("a", 64), "deadbeef", "testnet", 0)
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

func TestVerifyBitcoinBlock_MainnetSuccess(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode(blockstreamBlock{Height: 850000, Timestamp: 1700000000})
	}))
	defer server.Close()
	origMain := BlockstreamMainnetURL
	BlockstreamMainnetURL = server.URL
	t.Cleanup(func() { BlockstreamMainnetURL = origMain })

	result, skipped, err := VerifyBitcoinBlock(strings.Repeat("a", 64), "mainnet")
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
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode(blockstreamBlock{Height: 1, Timestamp: 0})
	}))
	defer server.Close()
	orig := BlockstreamTestnetURL
	BlockstreamTestnetURL = server.URL
	t.Cleanup(func() { BlockstreamTestnetURL = orig })

	result, _, err := VerifyBitcoinBlock(strings.Repeat("b", 64), "testnet")
	if err != nil {
		t.Fatal(err)
	}
	if result.Timestamp != "" {
		t.Errorf("zero timestamp should render as empty string, got %q", result.Timestamp)
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
