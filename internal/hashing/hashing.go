// Copyright (c) 2021-2026 Truestamp, Inc.
// SPDX-License-Identifier: MIT

// Package hashing exposes the full set of cryptographic hash algorithms
// the Truestamp backend accepts for a claim's `hash_type` field. The
// goal is feature-parity with sha256sum / shasum / openssl dgst so
// users do not have to shell out to those tools for a one-off digest.
//
// Algorithm names match the string forms in
// internal/tscrypto/hash.go's hashTypes registry, with common aliases
// (sha3-256/sha3_256, blake2b vs blake2b-512) accepted case-insensitively.
// Output formatting is byte-identical to GNU coreutils' sha256sum (text
// and binary modes, with the standard `\` escaping for filenames
// containing `\` or `\n`) and to `shasum --tag` / BSD md5(1)'s tagged
// form.
package hashing

import (
	"context"
	"crypto/md5"  //nolint:gosec // legacy algorithm, surfaced with a warning in the CLI
	"crypto/sha1" //nolint:gosec // legacy algorithm, surfaced with a warning in the CLI
	"crypto/sha256"
	"crypto/sha512"
	"fmt"
	"hash"
	"io"
	"strings"

	"golang.org/x/crypto/blake2b"
	"golang.org/x/crypto/blake2s"
	"golang.org/x/crypto/sha3"
)

// Algorithm describes a single supported hash algorithm.
type Algorithm struct {
	Name    string // canonical name (kebab-case)
	Aliases []string
	Size    int                // digest length in bytes
	Legacy  bool               // true for broken algorithms (MD5, SHA-1)
	BSDName string             // upper-case name used in shasum --tag / BSD output
	New     func() hash.Hash   // constructor; never returns nil
}

// newBlake2s256 wraps blake2s.New256(nil) — keyless construction never
// errors, but the stdlib API returns (hash.Hash, error) so we unwrap it.
func newBlake2s256() hash.Hash {
	h, err := blake2s.New256(nil)
	if err != nil {
		panic(fmt.Sprintf("blake2s.New256: %v", err))
	}
	return h
}

func newBlake2b256() hash.Hash {
	h, err := blake2b.New256(nil)
	if err != nil {
		panic(fmt.Sprintf("blake2b.New256: %v", err))
	}
	return h
}

func newBlake2b384() hash.Hash {
	h, err := blake2b.New384(nil)
	if err != nil {
		panic(fmt.Sprintf("blake2b.New384: %v", err))
	}
	return h
}

func newBlake2b512() hash.Hash {
	h, err := blake2b.New512(nil)
	if err != nil {
		panic(fmt.Sprintf("blake2b.New512: %v", err))
	}
	return h
}

// algorithms is the canonical ordered registry. Order drives --list and
// --help output so keep it stable (fastest→slowest, generic→specialized).
var algorithms = []Algorithm{
	{Name: "md5", BSDName: "MD5", Aliases: nil, Size: 16, Legacy: true, New: func() hash.Hash { return md5.New() }},
	{Name: "sha1", BSDName: "SHA1", Aliases: nil, Size: 20, Legacy: true, New: func() hash.Hash { return sha1.New() }},
	{Name: "sha224", BSDName: "SHA224", Aliases: nil, Size: 28, New: func() hash.Hash { return sha256.New224() }},
	{Name: "sha256", BSDName: "SHA256", Aliases: nil, Size: 32, New: func() hash.Hash { return sha256.New() }},
	{Name: "sha384", BSDName: "SHA384", Aliases: nil, Size: 48, New: func() hash.Hash { return sha512.New384() }},
	{Name: "sha512", BSDName: "SHA512", Aliases: nil, Size: 64, New: func() hash.Hash { return sha512.New() }},
	{Name: "sha3-224", BSDName: "SHA3-224", Aliases: []string{"sha3_224"}, Size: 28, New: func() hash.Hash { return sha3.New224() }},
	{Name: "sha3-256", BSDName: "SHA3-256", Aliases: []string{"sha3_256"}, Size: 32, New: func() hash.Hash { return sha3.New256() }},
	{Name: "sha3-384", BSDName: "SHA3-384", Aliases: []string{"sha3_384"}, Size: 48, New: func() hash.Hash { return sha3.New384() }},
	{Name: "sha3-512", BSDName: "SHA3-512", Aliases: []string{"sha3_512"}, Size: 64, New: func() hash.Hash { return sha3.New512() }},
	{Name: "blake2s-256", BSDName: "BLAKE2s", Aliases: []string{"blake2s"}, Size: 32, New: newBlake2s256},
	{Name: "blake2b-256", BSDName: "BLAKE2b-256", Aliases: nil, Size: 32, New: newBlake2b256},
	{Name: "blake2b-384", BSDName: "BLAKE2b-384", Aliases: nil, Size: 48, New: newBlake2b384},
	{Name: "blake2b-512", BSDName: "BLAKE2b", Aliases: []string{"blake2b"}, Size: 64, New: newBlake2b512},
}

// Algorithms returns a copy of the registry, ordered for display.
func Algorithms() []Algorithm {
	out := make([]Algorithm, len(algorithms))
	copy(out, algorithms)
	return out
}

// Lookup resolves a name (canonical or alias, case-insensitive) to an
// Algorithm entry. An unknown name returns a wrapped error listing the
// supported canonical names so CLI users get immediate guidance.
func Lookup(name string) (Algorithm, error) {
	key := strings.ToLower(strings.TrimSpace(name))
	for _, a := range algorithms {
		if a.Name == key {
			return a, nil
		}
		for _, alias := range a.Aliases {
			if alias == key {
				return a, nil
			}
		}
	}
	names := make([]string, len(algorithms))
	for i, a := range algorithms {
		names[i] = a.Name
	}
	return Algorithm{}, fmt.Errorf("unknown hash algorithm %q (supported: %s)", name, strings.Join(names, ", "))
}

// Compute streams r through alg's hasher and returns the digest bytes
// and the total byte count consumed. Never buffers the whole input.
// The context is checked once before starting; I/O errors are wrapped.
func Compute(ctx context.Context, alg Algorithm, r io.Reader) ([]byte, int64, error) {
	if err := ctx.Err(); err != nil {
		return nil, 0, err
	}
	h := alg.New()
	n, err := io.Copy(h, r)
	if err != nil {
		return nil, n, fmt.Errorf("hashing: %w", err)
	}
	return h.Sum(nil), n, nil
}

// FormatGNU produces output byte-identical to GNU coreutils' sha256sum
// (and friends):
//
//	text mode:   "<hex>  <filename>\n"
//	binary mode: "<hex> *<filename>\n"
//
// Filenames containing '\' or '\n' get GNU's escape treatment: the line
// is prefixed with '\' and the problematic bytes are escaped as "\\"
// and "\n". Callers on filesystems that cannot produce such names still
// get the unescaped form.
func FormatGNU(digestHex, filename string, binaryMode bool) string {
	escaped, needsEscape := gnuEscape(filename)
	sep := "  "
	if binaryMode {
		sep = " *"
	}
	lead := ""
	if needsEscape {
		lead = "\\"
	}
	return lead + digestHex + sep + escaped + "\n"
}

// FormatBSD produces output byte-identical to `shasum --tag` /
// BSD md5(1)'s tagged form:
//
//	"<NAME> (<filename>) = <hex>\n"
func FormatBSD(algBSDName, digestHex, filename string) string {
	return fmt.Sprintf("%s (%s) = %s\n", algBSDName, filename, digestHex)
}

// gnuEscape replicates GNU coreutils' filename escaping for checksum
// output. Returns the possibly-escaped filename and whether escaping
// was required (which triggers the leading '\' on the output line).
func gnuEscape(name string) (string, bool) {
	if !strings.ContainsAny(name, "\\\n") {
		return name, false
	}
	var b strings.Builder
	b.Grow(len(name) + 2)
	for i := 0; i < len(name); i++ {
		switch name[i] {
		case '\\':
			b.WriteString(`\\`)
		case '\n':
			b.WriteString(`\n`)
		default:
			b.WriteByte(name[i])
		}
	}
	return b.String(), true
}
