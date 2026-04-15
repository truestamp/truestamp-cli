// Copyright (c) 2021-2026 Truestamp, Inc.
// SPDX-License-Identifier: MIT

package tscrypto

import (
	"crypto/ed25519"
	"encoding/base64"
	"testing"
)

func TestDecodePublicKey_Valid(t *testing.T) {
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
	_, err := DecodePublicKey("not-valid-base64!!!")
	if err == nil {
		t.Error("expected error for invalid base64")
	}
}

func TestDecodePublicKey_WrongSize(t *testing.T) {
	_, err := DecodePublicKey("AAAAAAAAAAAAAAAAAAAAAA==")
	if err == nil {
		t.Error("expected error for wrong key size")
	}
}

func TestVerifyEd25519_ValidSignature(t *testing.T) {
	pub, priv, _ := ed25519.GenerateKey(nil)
	msg := []byte("test message hash 32 bytes long!")
	sig := ed25519.Sign(priv, msg)
	sigB64 := base64.StdEncoding.EncodeToString(sig)

	valid, err := VerifyEd25519(msg, sigB64, pub)
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	if !valid {
		t.Error("valid signature should verify")
	}
}

func TestVerifyEd25519_WrongKey(t *testing.T) {
	_, priv, _ := ed25519.GenerateKey(nil)
	otherPub, _, _ := ed25519.GenerateKey(nil)
	msg := []byte("test message hash 32 bytes long!")
	sig := ed25519.Sign(priv, msg)
	sigB64 := base64.StdEncoding.EncodeToString(sig)

	valid, err := VerifyEd25519(msg, sigB64, otherPub)
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	if valid {
		t.Error("wrong key should not verify")
	}
}

func TestVerifyEd25519_InvalidSignatureBase64(t *testing.T) {
	pub, _, _ := ed25519.GenerateKey(nil)
	_, err := VerifyEd25519([]byte("hash"), "not-base64!!!", pub)
	if err == nil {
		t.Error("expected error for invalid signature base64")
	}
}

func TestVerifyEd25519_WrongSignatureSize(t *testing.T) {
	pub, _, _ := ed25519.GenerateKey(nil)
	shortSig := base64.StdEncoding.EncodeToString([]byte("too short"))
	_, err := VerifyEd25519([]byte("hash"), shortSig, pub)
	if err == nil {
		t.Error("expected error for wrong signature size")
	}
}