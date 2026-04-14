package tscrypto

import (
	"crypto/ed25519"
	"encoding/base64"
	"fmt"
)

// VerifyEd25519 verifies an Ed25519 signature over a hash.
func VerifyEd25519(hashBytes []byte, signatureB64 string, pubkey ed25519.PublicKey) (bool, error) {
	sigBytes, err := base64.StdEncoding.DecodeString(signatureB64)
	if err != nil {
		return false, fmt.Errorf("decoding signature: %w", err)
	}
	if len(sigBytes) != ed25519.SignatureSize {
		return false, fmt.Errorf("invalid signature size: %d (expected %d)", len(sigBytes), ed25519.SignatureSize)
	}
	return ed25519.Verify(pubkey, hashBytes, sigBytes), nil
}

// DecodePublicKey decodes a base64-encoded Ed25519 public key.
func DecodePublicKey(b64 string) (ed25519.PublicKey, error) {
	keyBytes, err := base64.StdEncoding.DecodeString(b64)
	if err != nil {
		return nil, fmt.Errorf("decoding public key: %w", err)
	}
	if len(keyBytes) != ed25519.PublicKeySize {
		return nil, fmt.Errorf("invalid public key size: %d (expected %d)", len(keyBytes), ed25519.PublicKeySize)
	}
	return ed25519.PublicKey(keyBytes), nil
}
