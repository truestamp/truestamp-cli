// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: MIT

// Package tscrypto implements the Truestamp-specific cryptographic
// primitives used by proofs: SHA-256 with one-byte domain-separation
// prefixes (see kb/cryptography/byte-prefix-registry.md in truestamp-v2
// for the prefix registry) and Ed25519 signature verification.
package tscrypto

import (
	"crypto/sha256"
	"crypto/subtle"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
)

// ErrNotLowercaseHex is the sentinel every E.4 encoding refusal wraps, so
// a caller can tell "this field is not lowercase hex" from any other
// decode failure and tag its step with E.23's `invalid_hex_encoding`
// identifier. Appendix E.4 makes carrying that identifier a MUST, and the
// report layer cannot infer it from an error string without matching prose.
var ErrNotLowercaseHex = errors.New("not lowercase hex (E.4)")

// Domain separation prefix bytes per kb/cryptography/byte-prefix-registry.md
// (in truestamp-v2).
//
// The block carries the Merkle, Items, Entropy, Blockchain, Key-management and
// Proofs prefixes a verifier recomputes a bundle's hashes under. Since the
// published format carries every metadata map rather than its digest, the
// metadata prefixes (0x12, 0x22, 0x33) are computed by a verifier too. The
// commitment-record prefixes (0x34, 0x35) are kept so the frozen numbering
// stays legible; a verifier never recomputes them. The registry's remaining
// prefixes are deliberately omitted as out of scope for proof verification:
// 0x31 (genesis block constant), 0x41-0x45 (random tools) and 0x53
// (pre-rotation commitment).
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
// hex.DecodeString is case-insensitive, so a signing_key_id of "F2C39DF9" decoded
// to the same four bytes as "f2c39df9" and every derivation downstream
// agreed. The reference verifier's Base.decode16lower!/1 does not, which
// made uppercase an interoperability break as well as a malleability
// one, the same wire bundle verified here and aborted there.
func ValidateLowercaseHex(s string) error {
	if len(s)%2 != 0 {
		return fmt.Errorf("%w: odd length %d", ErrNotLowercaseHex, len(s))
	}
	for i := range len(s) {
		c := s[i]
		if (c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') {
			continue
		}
		if c >= 'A' && c <= 'F' {
			return fmt.Errorf("%w: uppercase %q at offset %d", ErrNotLowercaseHex, string(c), i)
		}
		return fmt.Errorf("%w: %q at offset %d is not a hex digit", ErrNotLowercaseHex, string(c), i)
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

// ComputeClaimsHash computes SHA256(0x11 || JCS(subject.claims)).
func ComputeClaimsHash(jcsBytes []byte) string {
	return BytesToHex(DomainHash(PrefixItemClaims, jcsBytes))
}

// ComputeEntropyHash computes SHA256(0x21 || JCS(payload)). The same
// derivation hashes an entropy subject's `subject.entropy` and every
// entropy witness payload carried under `subject.witnesses` (E.17a).
func ComputeEntropyHash(jcsBytes []byte) string {
	return BytesToHex(DomainHash(PrefixEntropy, jcsBytes))
}

// ComputeItemMetadataHash computes SHA256(0x12 || JCS(subject.metadata))
// for an item: the derived timing fingerprint over the witness map the
// bundle carries (E.10).
func ComputeItemMetadataHash(jcsBytes []byte) string {
	return BytesToHex(DomainHash(PrefixItemMetadata, jcsBytes))
}

// ComputeEntropyMetadataHash computes SHA256(0x22 || JCS(subject.metadata))
// for an entropy subject (E.10). The carried map is always empty, but the
// hash is derived from it rather than substituted, so a non-empty map is
// caught.
func ComputeEntropyMetadataHash(jcsBytes []byte) string {
	return BytesToHex(DomainHash(PrefixEntropyMetadata, jcsBytes))
}

// ComputeBlockMetadataHash computes SHA256(0x33 || JCS(block.metadata)) for
// any block map (E.14).
func ComputeBlockMetadataHash(jcsBytes []byte) string {
	return BytesToHex(DomainHash(PrefixBlockMetadata, jcsBytes))
}

// ComputeObservationHash computes the length-prefixed observation hash with domain prefix 0x23.
// Field order: id, entropy_hash, metadata_hash, signing_key_id
// This mirrors ComputeItemHash but uses prefix 0x23 for entropy observations.
//
// Decode failures name the field that refused (`signing_key_id`,
// `metadata_hash`), and the caller adds which map it belongs to: the same
// procedure runs over the subject and over every block map, and only the
// caller knows which one it is grading.
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
//
// Decode failures name the field that refused; see [ComputeObservationHash].
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
//
// The metadata hash is DERIVED by the caller from the carried metadata map
// (SHA-256(0x33 || JCS(metadata)), see [BlockMetadataHash]); a bundle never
// carries it. The same procedure hashes the top-level block, every
// block_path entry, the head block carried as the block witness, and the
// signing key event's block (Appendix E.14).
//
// Decode failures name the field that refused; see [ComputeObservationHash].
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
// keyIDHex MUST be the key id DERIVED from the bundle's `public_key`, that
// is ComputeKeyID(public_key), and MUST NOT be read from any
// `signing_key_id` the bundle carries. Appendix E.9 of the whitepaper is
// explicit about the split: this slot and the E.17 key binding take the
// derived value, while the E.10 subject-hash and E.14 block-hash preimages
// take the bundle's stored `signing_key_id` values verbatim, because those
// composites were hashed at creation time with the then-current key and are
// frozen into the Merkle tree. Under legitimate key rotation the stored ids
// differ from the derived one, and feeding a stored id in here makes every
// rotated proof fail signature verification.
//
// For block-like subjects (block and beacon), subject_hash == block_hash:
// the same 32 bytes appear in both slots. The type code in the payload
// domain-separates block (10) and beacon (11) signatures for the same
// underlying block.
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
// as they always have, on hex TEXT rather than decoded bytes, which is what
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
// it. The fold is therefore not what lets an uppercase merkle_root or
// epoch_merkle_root through; enforcement happens upstream, and by the time a
// bundle value gets here it is already known to be lowercase.
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
// a warn and MUST NOT fail on it, every E.11 soft check is advisory.
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

// SecureEqual reports whether two strings are byte-for-byte equal, in
// constant time for equal-length inputs. It is the comparison Appendix E.4
// requires for every hash and digest a verifier derives against one the
// bundle carries: exact, with no case folding, so a mis-cased wire value
// never compares equal to a derived one (the E.4 sweep names the encoding
// defect; this refuses to paper over it).
func SecureEqual(a, b string) bool {
	if len(a) != len(b) {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(a), []byte(b)) == 1
}
