// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: Apache-2.0

package tscrypto

import (
	"crypto/ed25519"
	"testing"
)

func TestDecodePublicKey_Valid(t *testing.T) {
	t.Parallel()
	b64 := "CTwMqDZnPd/QTLSq8aTeSD3a+j2DQxKcGfhhIYJQ65Y="
	key, err := DecodePublicKey(b64)
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	if len(key) != ed25519.PublicKeySize {
		t.Errorf("key size: got %d, want %d", len(key), ed25519.PublicKeySize)
	}
}

func TestDecodePublicKey_InvalidBase64(t *testing.T) {
	t.Parallel()
	_, err := DecodePublicKey("not-valid-base64!!!")
	if err == nil {
		t.Error("expected error for invalid base64")
	}
}

func TestDecodePublicKey_WrongSize(t *testing.T) {
	t.Parallel()
	_, err := DecodePublicKey("AAAAAAAAAAAAAAAAAAAAAA==")
	if err == nil {
		t.Error("expected error for wrong key size")
	}
}
