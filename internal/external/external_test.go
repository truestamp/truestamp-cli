// Copyright (c) 2021-2026 Truestamp, Inc.
// SPDX-License-Identifier: MIT

package external

import (
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
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
		json.NewEncoder(w).Encode(horizonTx{
			MemoType: "hash",
			Memo:     memoB64,
			Ledger:   1537561,
		})
	}))
	defer server.Close()

	// Monkey-patch by testing directly with the mock — we need to override the URL.
	// Instead, test the memo comparison logic via the exported function
	// by pointing to a mock server. Since VerifyStellar constructs its own URL,
	// we test the internal logic differently.
	// For now, test the happy path with a real-ish structure.
	t.Skip("VerifyStellar hardcodes Horizon URL — integration test only")
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
