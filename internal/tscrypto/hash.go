// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: MIT

// Package tscrypto implements the Truestamp-specific cryptographic
// primitives used by proofs: SHA-256 with one-byte domain-separation
// prefixes (see docs/CRYPTOGRAPHY.md in truestamp-v2 for the prefix
// registry) and Ed25519 signature verification.
package tscrypto

import (
	"crypto/sha256"
	"crypto/subtle"
	"encoding/binary"
	"encoding/hex"
	"fmt"
)

// Domain separation prefix bytes per docs/CRYPTOGRAPHY.md.
//
// The registry is mirrored whole, not pruned to the prefixes this verifier
// consumes: it is the frozen numbering a reader has to be able to check a
// preimage against, and several entries (0x12, 0x22, 0x33, 0x34, 0x35) belong
// to producer-side hashes a verifier never recomputes. Keeping the gaps visible
// is the point — a constant cannot be called with the wrong arguments, unlike
// the producer-side hash builders that used to sit alongside it.
const (
	PrefixMerkleLeaf      = 0x00
	PrefixMerkleInternal  = 0x01
	PrefixItemClaims      = 0x11
	PrefixItemMetadata    = 0x12
	PrefixItemHash        = 0x13
	PrefixEntropy         = 0x21
	PrefixEntropyMetadata = 0x22
	PrefixObservationHash = 0x23
	PrefixBlockMetadata   = 0x33
	PrefixBlockHash       = 0x32
	PrefixCommitmentData  = 0x34
	PrefixCommitmentHash  = 0x35
	PrefixKeyID           = 0x51
	PrefixProofHash       = 0x61
)

// ValidateLowercaseHex reports whether s carries the encoding Appendix
// E.4 mandates for every hash field: "Encodings: hashes are lowercase
// hex". It returns nil for a conforming value and, for a
// non-conforming one, an error naming the first offending byte and its
// offset so a report can say which character is wrong rather than only
// that the field "is invalid".
//
// Empty is accepted. An absent field is not an encoding defect, and the
// steps that need one already grade its absence on their own terms
// (E.10's and E.14's "no usable value" arms); reporting "" here would
// claim the bundle carries a malformed value where it carries none.
//
// Uppercase is called out separately from a non-hex byte because the two
// are different mistakes with different fixes, and because uppercase is
// the one that silently verified before this check existed: Go's
// hex.DecodeString is case-insensitive, so `b.kid` = "F2C39DF9" decoded
// to the same four bytes as "f2c39df9" and every derivation downstream
// agreed. The reference verifier's Base.decode16lower!/1 does not, which
// made uppercase an interoperability break as well as a malleability
// one — the same wire bundle verified here and aborted there.
func ValidateLowercaseHex(s string) error {
	if len(s)%2 != 0 {
		return fmt.Errorf("not lowercase hex (E.4): odd length %d", len(s))
	}
	for i := range len(s) {
		c := s[i]
		if (c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') {
			continue
		}
		if c >= 'A' && c <= 'F' {
			return fmt.Errorf("not lowercase hex (E.4): uppercase %q at offset %d", string(c), i)
		}
		return fmt.Errorf("not lowercase hex (E.4): %q at offset %d is not a hex digit", string(c), i)
	}
	return nil
}

// HexToBytes decodes a lowercase-hex string to bytes. Returns empty slice
// for empty input.
//
// This is the decoder every preimage builder in this package runs its hex
// inputs through, and E.4's lowercase rule is enforced here rather than at
// parse time on purpose. E.6's hard-rejection table is exhaustive and
// carries no row for hex case, so aborting the whole run would invent a
// rejection the appendix does not authorize; blanking the field instead
// would make the report say a present field is absent. Failing the decode
// leaves it a graded step failure that names the field, which is what a
// bundle carrying an unusable value already gets.
func HexToBytes(h string) ([]byte, error) {
	if h == "" {
		return []byte{}, nil
	}
	if err := ValidateLowercaseHex(h); err != nil {
		return nil, err
	}
	return hex.DecodeString(h)
}

// BytesToHex encodes bytes to lowercase hex string.
func BytesToHex(b []byte) string {
	return hex.EncodeToString(b)
}

// DomainHash computes SHA256(prefix_byte || data).
func DomainHash(prefix byte, data []byte) []byte {
	h := sha256.New()
	h.Write([]byte{prefix})
	h.Write(data)
	return h.Sum(nil)
}

// ComputeKeyID derives key_id from public key: truncate4(SHA256(0x51 || pubkey)).
func ComputeKeyID(pubkey []byte) string {
	hash := DomainHash(PrefixKeyID, pubkey)
	return BytesToHex(hash[:4])
}

// ComputeEntropyHash computes SHA256(0x21 || JCS(entropy_data)).
func ComputeEntropyHash(jcsBytes []byte) string {
	return BytesToHex(DomainHash(PrefixEntropy, jcsBytes))
}

// ComputeObservationHash computes the length-prefixed observation hash with domain prefix 0x23.
// Field order: id, entropy_hash, metadata_hash, signing_key_id
// This mirrors ComputeItemHash but uses prefix 0x23 for entropy observations.
func ComputeObservationHash(id, entropyHashHex, metadataHashHex, signingKeyIDHex string) (string, error) {
	entropyBytes, err := HexToBytes(entropyHashHex)
	if err != nil {
		return "", fmt.Errorf("decoding entropy_hash: %w", err)
	}
	metaBytes, err := HexToBytes(metadataHashHex)
	if err != nil {
		return "", fmt.Errorf("decoding metadata_hash: %w", err)
	}
	keyIDBytes, err := HexToBytes(signingKeyIDHex)
	if err != nil {
		return "", fmt.Errorf("decoding signing_key_id: %w", err)
	}

	totalSize := (4 + len(id)) + (4 + len(entropyBytes)) + (4 + len(metaBytes)) + (4 + len(keyIDBytes))
	serialized := make([]byte, 0, totalSize)
	serialized = appendLenPrefixed(serialized, []byte(id))
	serialized = appendLenPrefixed(serialized, entropyBytes)
	serialized = appendLenPrefixed(serialized, metaBytes)
	serialized = appendLenPrefixed(serialized, keyIDBytes)

	return BytesToHex(DomainHash(PrefixObservationHash, serialized)), nil
}

// ComputeItemHash computes the length-prefixed item hash with domain prefix 0x13.
// Field order: id, claims_hash, metadata_hash, signing_key_id
func ComputeItemHash(id, claimsHashHex, metadataHashHex, signingKeyIDHex string) (string, error) {
	claimsBytes, err := HexToBytes(claimsHashHex)
	if err != nil {
		return "", fmt.Errorf("decoding claims_hash: %w", err)
	}
	metaBytes, err := HexToBytes(metadataHashHex)
	if err != nil {
		return "", fmt.Errorf("decoding metadata_hash: %w", err)
	}
	keyIDBytes, err := HexToBytes(signingKeyIDHex)
	if err != nil {
		return "", fmt.Errorf("decoding signing_key_id: %w", err)
	}

	totalSize := (4 + len(id)) + (4 + len(claimsBytes)) + (4 + len(metaBytes)) + (4 + len(keyIDBytes))
	serialized := make([]byte, 0, totalSize)
	serialized = appendLenPrefixed(serialized, []byte(id))
	serialized = appendLenPrefixed(serialized, claimsBytes)
	serialized = appendLenPrefixed(serialized, metaBytes)
	serialized = appendLenPrefixed(serialized, keyIDBytes)

	return BytesToHex(DomainHash(PrefixItemHash, serialized)), nil
}

// ComputeBlockHash computes the length-prefixed block hash with domain prefix 0x32.
// Field order: id, previous_block_hash, merkle_root, metadata_hash, signing_key_id
func ComputeBlockHash(id, prevHashHex, merkleRootHex, metadataHashHex, signingKeyIDHex string) (string, error) {
	prevBytes, err := HexToBytes(prevHashHex)
	if err != nil {
		return "", fmt.Errorf("decoding previous_block_hash: %w", err)
	}
	merkleBytes, err := HexToBytes(merkleRootHex)
	if err != nil {
		return "", fmt.Errorf("decoding merkle_root: %w", err)
	}
	metaBytes, err := HexToBytes(metadataHashHex)
	if err != nil {
		return "", fmt.Errorf("decoding metadata_hash: %w", err)
	}
	keyIDBytes, err := HexToBytes(signingKeyIDHex)
	if err != nil {
		return "", fmt.Errorf("decoding signing_key_id: %w", err)
	}

	idBytes := []byte(id)
	totalSize := (4 + len(idBytes)) + (4 + len(prevBytes)) + (4 + len(merkleBytes)) + (4 + len(metaBytes)) + (4 + len(keyIDBytes))
	serialized := make([]byte, 0, totalSize)
	serialized = appendLenPrefixed(serialized, idBytes)
	serialized = appendLenPrefixed(serialized, prevBytes)
	serialized = appendLenPrefixed(serialized, merkleBytes)
	serialized = appendLenPrefixed(serialized, metaBytes)
	serialized = appendLenPrefixed(serialized, keyIDBytes)

	return BytesToHex(DomainHash(PrefixBlockHash, serialized)), nil
}

// BuildCompactProofPayload builds the compact proof signature payload
// and computes SHA256(0x61 || payload). The returned 32-byte hash is what
// the Ed25519 signature covers.
//
// Byte layout (big-endian throughout):
//
//	offset  size  field
//	0       1     v  (version, uint8)
//	1       2     t  (type code, uint16 BE)
//	3       4     kid (4 bytes, hex-decoded)
//	7       8     ts_ms (timestamp in ms since Unix epoch, uint64 BE)
//	15      32    subject_hash
//	47      32    block_hash
//	79      2     N (epoch root count, uint16 BE)
//	81      32*N  epoch_roots (concatenated, in cx order)
//
// keyIDHex MUST be the key id DERIVED from the bundle's `pk` — that is,
// ComputeKeyID(pk) — and MUST NOT be read from `b.kid` or `s.kid`. Appendix
// E.9 of the whitepaper is explicit about the split: this slot and the E.17
// keyring cross-check take the derived value, while the E.10 subject-hash and
// E.14 block-hash preimages take the bundle's stored `s.kid` / `b.kid`
// verbatim, because those composites were hashed at creation time with the
// then-current key and are frozen into the Merkle tree. Under legitimate key
// rotation the stored kids differ from the derived one, and feeding a stored
// kid in here makes every rotated proof fail signature verification.
//
// For block-like subjects (t ∈ {10, 11} — plain block and beacon),
// subject_hash == block_hash — the same 32 bytes appear in both slots.
// The `t` byte in the payload domain-separates block (t=10) and beacon
// (t=11) signatures for the same underlying block.
func BuildCompactProofPayload(version byte, typeCode uint16, keyIDHex string, timestampMs uint64, subjectHashHex, blockHashHex string, epochRootHexes []string) ([]byte, error) {
	keyIDBytes, err := HexToBytes(keyIDHex)
	if err != nil {
		return nil, fmt.Errorf("decoding key_id: %w", err)
	}
	subjectHashBytes, err := HexToBytes(subjectHashHex)
	if err != nil {
		return nil, fmt.Errorf("decoding subject_hash: %w", err)
	}
	blockHashBytes, err := HexToBytes(blockHashHex)
	if err != nil {
		return nil, fmt.Errorf("decoding block_hash: %w", err)
	}

	if len(epochRootHexes) > 65535 {
		return nil, fmt.Errorf("epoch root count %d exceeds maximum 65535", len(epochRootHexes))
	}

	payload := make([]byte, 0, 1+2+4+8+32+32+2+32*len(epochRootHexes))
	payload = append(payload, version)
	payload = append(payload, byte(typeCode>>8), byte(typeCode))
	payload = append(payload, keyIDBytes...)

	var tsBuf [8]byte
	binary.BigEndian.PutUint64(tsBuf[:], timestampMs)
	payload = append(payload, tsBuf[:]...)

	payload = append(payload, subjectHashBytes...)
	payload = append(payload, blockHashBytes...)

	payload = append(payload, byte(len(epochRootHexes)>>8), byte(len(epochRootHexes)))
	for _, er := range epochRootHexes {
		erBytes, err := HexToBytes(er)
		if err != nil {
			return nil, fmt.Errorf("decoding epoch_root: %w", err)
		}
		payload = append(payload, erBytes...)
	}

	hashBytes := DomainHash(PrefixProofHash, payload)
	return hashBytes, nil
}

// HexEqual reports whether two hex strings are equal, ignoring ASCII case in
// the range 'A'-'F'.
//
// The comparison is constant-time in the CONTENTS of the operands: every byte
// is folded and accumulated, and only the length short-circuits (a length
// difference is not secret). Appendix E.4 of the whitepaper makes this a MUST
// for all hash and digest comparisons, restated for the E.7 hash comparison
// and the E.13 inclusion-proof root.
//
// Nothing is hex-decoded, so non-hex and odd-length operands compare exactly
// as they always have, on hex TEXT rather than decoded bytes — which is what
// the reference verifier's secure_equal?/2 compares too.
//
// The case fold is NOT shared with the reference verifier, and keeping it is
// a deliberate split rather than an oversight. secure_equal?/2 is a raw
// binary compare; the reference downcases only at its one caller-supplied
// operand, the expected hash. This function's live call sites are the two
// places where a case fold is required rather than merely tolerated:
//
//   - E.7's expected-hash comparison, which the appendix instructs a verifier
//     to normalize ("trim, downcase") before comparing against s.d.hash, and
//     which the reference implements the same way;
//   - the E.21 and E.18/E.19 comparisons against a value fetched from an
//     outside service, where the remote party chooses the case. The NIST
//     beacon API emits its outputValue in uppercase, so a case-sensitive
//     compare there would grade a sound entropy proof as a value mismatch.
//
// Bundle-carried hex no longer needs the fold, because [ValidateLowercaseHex]
// and [HexToBytes] reject a non-lowercase field before any comparison reaches
// it. The fold is therefore not what lets an uppercase b.mr or cx[].memo
// through; enforcement happens upstream, and by the time a bundle value gets
// here it is already known to be lowercase.
func HexEqual(a, b string) bool {
	if len(a) != len(b) {
		return false
	}
	var diff byte
	for i := range len(a) {
		diff |= foldASCIIHex(a[i]) ^ foldASCIIHex(b[i])
	}
	return subtle.ConstantTimeByteEq(diff, 0) == 1
}

// foldASCIIHex lowercases the ASCII range 'A'-'F' without branching on the
// byte's value, so the case fold does not reintroduce the side channel
// HexEqual exists to close.
func foldASCIIHex(c byte) byte {
	isUpperHex := subtle.ConstantTimeLessOrEq('A', int(c)) & subtle.ConstantTimeLessOrEq(int(c), 'F')
	return byte(subtle.ConstantTimeSelect(isUpperHex, int(c)+('a'-'A'), int(c)))
}

// BytesEqual reports whether two byte slices are equal, in constant time.
func BytesEqual(a, b []byte) bool {
	return subtle.ConstantTimeCompare(a, b) == 1
}

// appendLenPrefixed appends a 4-byte big-endian length prefix followed by data to dst.
func appendLenPrefixed(dst, data []byte) []byte {
	var prefix [4]byte
	binary.BigEndian.PutUint32(prefix[:], uint32(len(data)))
	dst = append(dst, prefix[:]...)
	dst = append(dst, data...)
	return dst
}

// hashTypeInfo holds metadata for a supported hash type.
type hashTypeInfo struct {
	Bytes int
	Name  string
}

// hashTypes maps hash type keys to their expected output sizes.
var hashTypes = map[string]hashTypeInfo{
	"md5":      {Bytes: 16, Name: "MD5"},
	"sha1":     {Bytes: 20, Name: "SHA-1"},
	"sha224":   {Bytes: 28, Name: "SHA-224"},
	"sha256":   {Bytes: 32, Name: "SHA-256"},
	"sha384":   {Bytes: 48, Name: "SHA-384"},
	"sha512":   {Bytes: 64, Name: "SHA-512"},
	"sha3_224": {Bytes: 28, Name: "SHA3-224"},
	"sha3_256": {Bytes: 32, Name: "SHA3-256"},
	"sha3_384": {Bytes: 48, Name: "SHA3-384"},
	"sha3_512": {Bytes: 64, Name: "SHA3-512"},
	"blake2s":  {Bytes: 32, Name: "BLAKE2s"},
	"blake2b":  {Bytes: 64, Name: "BLAKE2b"},
}

// ValidateClaimsHash checks that a claimed hash has both the length AND the
// lowercase-hex character set required by its named hash type.
//
// Appendix E.11 of the whitepaper: "check that the hex length equals twice the
// algorithm's output size and that the character set is lowercase hex, for one
// of the twelve registered algorithms". Callers MUST render a non-nil error as
// a warn and MUST NOT fail on it — every E.11 soft check is advisory.
//
// The scan is a byte loop, not a rune loop: a multi-byte UTF-8 hash trips the
// length check first rather than the charset check, but either way the caller
// warns, which is all E.11 constrains.
func ValidateClaimsHash(hash, hashType string) error {
	if hash == "" || hashType == "" {
		return nil
	}

	info, ok := hashTypes[hashType]
	if !ok {
		return fmt.Errorf("unknown hash type: %s", hashType)
	}

	expectedHex := info.Bytes * 2
	if len(hash) != expectedHex {
		return fmt.Errorf("expected %d hex characters for %s, got %d", expectedHex, hashType, len(hash))
	}

	for i := range len(hash) {
		c := hash[i]
		if (c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') {
			continue
		}
		return fmt.Errorf("expected %d lowercase hex characters for %s, got %q at offset %d",
			expectedHex, hashType, string(c), i)
	}

	return nil
}
