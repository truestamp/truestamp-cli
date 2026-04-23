// Copyright (c) 2021-2026 Truestamp, Inc.
// SPDX-License-Identifier: MIT

package hashing

import (
	"bytes"
	"context"
	"encoding/hex"
	"io"
	"strings"
	"testing"
)

// Published test vectors for each algorithm. Two anchors — empty and
// "abc" — cover the most-cited conformance vectors used by NIST (FIPS
// 180-4, FIPS 202) and IETF RFCs (6234, 7693). Passing these confirms
// that our per-algorithm constructors match the canonical outputs of
// sha256sum, shasum, openssl dgst, and b2sum.
var canonicalVectors = []struct {
	alg   string
	in    string
	outHx string
}{
	// MD5 — RFC 1321 Appendix A test suite
	{"md5", "", "d41d8cd98f00b204e9800998ecf8427e"},
	{"md5", "abc", "900150983cd24fb0d6963f7d28e17f72"},
	// SHA-1 — RFC 3174 Appendix A vectors
	{"sha1", "", "da39a3ee5e6b4b0d3255bfef95601890afd80709"},
	{"sha1", "abc", "a9993e364706816aba3e25717850c26c9cd0d89d"},
	// SHA-2 — NIST FIPS 180-4 examples
	{"sha224", "", "d14a028c2a3a2bc9476102bb288234c415a2b01f828ea62ac5b3e42f"},
	{"sha224", "abc", "23097d223405d8228642a477bda255b32aadbce4bda0b3f7e36c9da7"},
	{"sha256", "", "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"},
	{"sha256", "abc", "ba7816bf8f01cfea414140de5dae2223b00361a396177a9cb410ff61f20015ad"},
	{"sha384", "", "38b060a751ac96384cd9327eb1b1e36a21fdb71114be07434c0cc7bf63f6e1da274edebfe76f65fbd51ad2f14898b95b"},
	{"sha384", "abc", "cb00753f45a35e8bb5a03d699ac65007272c32ab0eded1631a8b605a43ff5bed8086072ba1e7cc2358baeca134c825a7"},
	{"sha512", "", "cf83e1357eefb8bdf1542850d66d8007d620e4050b5715dc83f4a921d36ce9ce47d0d13c5d85f2b0ff8318d2877eec2f63b931bd47417a81a538327af927da3e"},
	{"sha512", "abc", "ddaf35a193617abacc417349ae20413112e6fa4e89a97ea20a9eeee64b55d39a2192992a274fc1a836ba3c23a3feebbd454d4423643ce80e2a9ac94fa54ca49f"},
	// SHA-3 — NIST FIPS 202 examples
	{"sha3-224", "", "6b4e03423667dbb73b6e15454f0eb1abd4597f9a1b078e3f5b5a6bc7"},
	{"sha3-224", "abc", "e642824c3f8cf24ad09234ee7d3c766fc9a3a5168d0c94ad73b46fdf"},
	{"sha3-256", "", "a7ffc6f8bf1ed76651c14756a061d662f580ff4de43b49fa82d80a4b80f8434a"},
	{"sha3-256", "abc", "3a985da74fe225b2045c172d6bd390bd855f086e3e9d525b46bfe24511431532"},
	{"sha3-384", "", "0c63a75b845e4f7d01107d852e4c2485c51a50aaaa94fc61995e71bbee983a2ac3713831264adb47fb6bd1e058d5f004"},
	{"sha3-384", "abc", "ec01498288516fc926459f58e2c6ad8df9b473cb0fc08c2596da7cf0e49be4b298d88cea927ac7f539f1edf228376d25"},
	{"sha3-512", "", "a69f73cca23a9ac5c8b567dc185a756e97c982164fe25859e0d1dcc1475c80a615b2123af1f5f94c11e3e9402c3ac558f500199d95b6d3e301758586281dcd26"},
	{"sha3-512", "abc", "b751850b1a57168a5693cd924b6b096e08f621827444f70d884f5d0240d2712e10e116e9192af3c91a7ec57647e3934057340b4cf408d5a56592f8274eec53f0"},
	// BLAKE2 — RFC 7693 Appendix A + reproducible "abc" anchors
	{"blake2b-512", "abc", "ba80a53f981c4d0d6a2797b69f12f6e94c212f14685ac4b74b12bb6fdbffa2d17d87c5392aab792dc252d5de4533cc9518d38aa8dbf1925ab92386edd4009923"},
	{"blake2s-256", "abc", "508c5e8c327c14e2e1a72ba34eeb452f37458b209ed63a294d999b4c86675982"},
}

func TestCompute_CanonicalVectors(t *testing.T) {
	t.Parallel()
	for _, v := range canonicalVectors {
		alg, err := Lookup(v.alg)
		if err != nil {
			t.Fatalf("Lookup(%q): %v", v.alg, err)
		}
		digest, n, err := Compute(context.Background(), alg, strings.NewReader(v.in))
		if err != nil {
			t.Errorf("Compute(%q, %q): %v", v.alg, v.in, err)
			continue
		}
		if n != int64(len(v.in)) {
			t.Errorf("Compute(%q, %q): size %d, want %d", v.alg, v.in, n, len(v.in))
		}
		gotHex := hex.EncodeToString(digest)
		if gotHex != v.outHx {
			t.Errorf("Compute(%q, %q): got %s, want %s", v.alg, v.in, gotHex, v.outHx)
		}
		if len(digest) != alg.Size {
			t.Errorf("Compute(%q, %q): digest size %d, want %d", v.alg, v.in, len(digest), alg.Size)
		}
	}
}

func TestCompute_EmptyAllAlgorithms(t *testing.T) {
	t.Parallel()
	// Every registered algorithm must produce a deterministic, non-empty
	// digest for empty input (and the size must match the declared Size).
	for _, alg := range Algorithms() {
		digest, _, err := Compute(context.Background(), alg, strings.NewReader(""))
		if err != nil {
			t.Errorf("Compute(%q, empty): %v", alg.Name, err)
			continue
		}
		if len(digest) != alg.Size {
			t.Errorf("Compute(%q, empty): digest size %d, want %d", alg.Name, len(digest), alg.Size)
		}
	}
}

func TestCompute_Streaming(t *testing.T) {
	t.Parallel()
	// 1 MiB buffer streamed through io.Copy must produce the same digest
	// as handing the slice to a one-shot hasher. Guards against any
	// accidental buffering bug.
	data := bytes.Repeat([]byte{0x5a}, 1<<20)

	alg, err := Lookup("sha256")
	if err != nil {
		t.Fatal(err)
	}
	streamed, _, err := Compute(context.Background(), alg, bytes.NewReader(data))
	if err != nil {
		t.Fatalf("Compute: %v", err)
	}

	h := alg.New()
	h.Write(data)
	oneShot := h.Sum(nil)

	if !bytes.Equal(streamed, oneShot) {
		t.Errorf("streamed %x != one-shot %x", streamed, oneShot)
	}
}

func TestLookup_AliasesAndCase(t *testing.T) {
	t.Parallel()
	cases := []struct {
		in        string
		canonical string
	}{
		{"SHA256", "sha256"},
		{"  sha256  ", "sha256"},
		{"sha3_256", "sha3-256"},
		{"SHA3-256", "sha3-256"},
		{"blake2b", "blake2b-512"},
		{"blake2s", "blake2s-256"},
	}
	for _, c := range cases {
		alg, err := Lookup(c.in)
		if err != nil {
			t.Errorf("Lookup(%q): %v", c.in, err)
			continue
		}
		if alg.Name != c.canonical {
			t.Errorf("Lookup(%q): got %q, want %q", c.in, alg.Name, c.canonical)
		}
	}
}

func TestLookup_Unknown(t *testing.T) {
	t.Parallel()
	_, err := Lookup("crc32")
	if err == nil {
		t.Fatal("Lookup(unknown) should return error")
	}
	// The error message should hint at the supported algorithms.
	if !strings.Contains(err.Error(), "sha256") {
		t.Errorf("error should list supported algorithms, got: %v", err)
	}
}

func TestAlgorithms_Stable(t *testing.T) {
	t.Parallel()
	// Algorithms() is part of the public contract — downstream tooling
	// may parse `truestamp hash --list`. Pin the order and size.
	got := Algorithms()
	if len(got) != 14 {
		t.Errorf("got %d algorithms, want 14", len(got))
	}
	if got[0].Name != "md5" || got[3].Name != "sha256" || got[len(got)-1].Name != "blake2b-512" {
		t.Errorf("algorithm order drifted: %s, %s, ..., %s", got[0].Name, got[3].Name, got[len(got)-1].Name)
	}
}

func TestLegacyFlag(t *testing.T) {
	t.Parallel()
	for _, alg := range Algorithms() {
		want := alg.Name == "md5" || alg.Name == "sha1"
		if alg.Legacy != want {
			t.Errorf("Legacy flag for %q: got %v, want %v", alg.Name, alg.Legacy, want)
		}
	}
}

func TestFormatGNU_TextAndBinary(t *testing.T) {
	t.Parallel()
	digest := "abc123"
	cases := []struct {
		bin  bool
		name string
		want string
	}{
		{false, "foo.bin", "abc123  foo.bin\n"},
		{true, "foo.bin", "abc123 *foo.bin\n"},
		{false, "-", "abc123  -\n"},
	}
	for _, c := range cases {
		got := FormatGNU(digest, c.name, c.bin)
		if got != c.want {
			t.Errorf("FormatGNU(%q, bin=%v): got %q, want %q", c.name, c.bin, got, c.want)
		}
	}
}

func TestFormatGNU_FilenameEscape(t *testing.T) {
	t.Parallel()
	// GNU coreutils prefixes the whole line with '\' when the filename
	// contained '\' or '\n' and escapes those characters inside the name.
	cases := []struct {
		name string
		want string
	}{
		{"a\\b", `\abc123  a\\b` + "\n"},
		{"a\nb", `\abc123  a\nb` + "\n"},
		{"normal", "abc123  normal\n"},
	}
	for _, c := range cases {
		got := FormatGNU("abc123", c.name, false)
		if got != c.want {
			t.Errorf("FormatGNU(%q): got %q, want %q", c.name, got, c.want)
		}
	}
}

func TestFormatBSD(t *testing.T) {
	t.Parallel()
	got := FormatBSD("SHA256", "abc123", "foo.bin")
	want := "SHA256 (foo.bin) = abc123\n"
	if got != want {
		t.Errorf("FormatBSD: got %q, want %q", got, want)
	}
}

func TestCompute_ReaderError(t *testing.T) {
	t.Parallel()
	alg, _ := Lookup("sha256")
	_, _, err := Compute(context.Background(), alg, errorReader{})
	if err == nil {
		t.Fatal("expected read error")
	}
}

func TestCompute_CancelledContext(t *testing.T) {
	t.Parallel()
	alg, _ := Lookup("sha256")
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel before call so the early ctx.Err() check fires
	_, _, err := Compute(ctx, alg, strings.NewReader("abc"))
	if err == nil {
		t.Fatal("expected context.Canceled")
	}
}

type errorReader struct{}

func (errorReader) Read([]byte) (int, error) { return 0, io.ErrUnexpectedEOF }
