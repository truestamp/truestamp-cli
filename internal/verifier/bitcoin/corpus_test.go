// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: Apache-2.0

package bitcoin

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"encoding/json/v2"
	"math/rand/v2"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/truestamp/truestamp-cli/internal/specdata"
	"github.com/truestamp/truestamp-cli/internal/vectorcases"
)

// corpusFile is one file of vectors/corpus/bitcoin/ (SPEC.md section
// 13.2): real Bitcoin bytes, and what a parser concludes of them under
// section 10.2, step 9 (d).
type corpusFile struct {
	name     string
	Hex      string `json:"hex"`
	Kind     string `json:"kind"`
	Expected struct {
		Fails            string `json:"fails"`
		HeaderHash       string `json:"header_hash"`
		HeaderMerkleRoot string `json:"header_merkle_root"`
		HeaderTime       string `json:"header_time"`
		Matched          string `json:"matched"`
		MeetsTarget      *bool  `json:"meets_target"`
		OpReturn         string `json:"op_return"`
		TotalTxs         uint32 `json:"total_txs"`
		TransactionHash  string `json:"transaction_hash"`
		WellFormed       *bool  `json:"well_formed"`
	} `json:"expected"`
}

// corpus reads every file of vectors/corpus/bitcoin/.
func corpus(t *testing.T) []corpusFile {
	t.Helper()
	paths, err := filepath.Glob(filepath.Join(vectorcases.Dir(), "corpus", "bitcoin", "*.json"))
	if err != nil || len(paths) == 0 {
		t.Fatalf("the Bitcoin corpus: %q, %v", paths, err)
	}
	var out []corpusFile
	for _, p := range paths {
		data, err := os.ReadFile(p)
		if err != nil {
			t.Fatal(err)
		}
		var f corpusFile
		if err := json.Unmarshal(data, &f); err != nil {
			t.Fatalf("%s: %v", p, err)
		}
		f.name = filepath.Base(p)
		out = append(out, f)
	}
	return out
}

// bytesOf decodes hex text, failing the test when it is not.
func bytesOf(t *testing.T, text string) []byte {
	t.Helper()
	b, err := hex.DecodeString(text)
	if err != nil {
		t.Fatalf("%q: %v", text, err)
	}
	return b
}

// formatTIME writes Unix seconds as SPEC.md section 2.4's TIME, through
// Go's time package, for comparison here only.
func formatTIME(seconds uint32) string {
	return time.Unix(int64(seconds), 0).UTC().Format("2006-01-02T15:04:05Z")
}

// Every file of vectors/corpus/bitcoin/ reads as its expected member says
// (SPEC.md section 13.2): a transaction's transaction_hash is rev(SHA-256d)
// of its bytes as carried, the witness files' included, and it is well
// formed with the 32 bytes its OP_RETURN pushes, or refused by the rule
// fails names, parse for every -witness file; a merkleblock is well formed
// with its matched hash, total_txs and header values; a header's hash,
// Merkle root, time and target are as given. The corpus holds 17 files,
// three of them -witness. Mutants killed: TransactionHash dropping the BIP
// 144 marker and flag before it hashes, or SHA-256 applied once (the
// -witness hashes differ); Display without the reversal (every hash
// differs); a node of the partial tree with no right child refused as a
// duplicate pair (the real merkleblocks of 230 and 3,228 transactions are
// refused); nBits read from the octets of nTime (the real headers fail
// their targets).
func TestCorpus_EachFileReadsAsExpected(t *testing.T) {
	files := corpus(t)
	witness, kinds := 0, map[string]int{}
	for _, f := range files {
		b := bytesOf(t, f.Hex)
		kinds[f.Kind]++
		e := f.Expected
		switch f.Kind {
		case "transaction":
			if got := Display(TransactionHash(b)); got != e.TransactionHash {
				t.Errorf("%s: transaction_hash %s, want %s", f.name, got, e.TransactionHash)
			}
			pushed, err := ParseTransaction(b)
			switch {
			case e.WellFormed == nil:
				t.Fatalf("%s: no well_formed", f.name)
			case *e.WellFormed && (err != nil || hex.EncodeToString(pushed[:]) != e.OpReturn):
				t.Errorf("%s: %x, %v; want the OP_RETURN %s", f.name, pushed, err, e.OpReturn)
			case !*e.WellFormed && (err == nil || err.Error() != e.Fails):
				t.Errorf("%s: %v, want the rule %s", f.name, err, e.Fails)
			}
			if strings.HasSuffix(f.name, "-witness.json") {
				witness++
				if err != RefusedParse {
					t.Errorf("%s: a witness serialization gives %v, want parse", f.name, err)
				}
			}
		case "merkleblock":
			block, err := ParseMerkleBlock(b)
			if e.WellFormed == nil || !*e.WellFormed || err != nil {
				t.Fatalf("%s: %v, want well formed", f.name, err)
			}
			if Display(block.Matched) != e.Matched || block.TotalTxs != e.TotalTxs ||
				Display(HeaderHash(block.Header)) != e.HeaderHash || Display(MerkleRootField(block.Header)) != e.HeaderMerkleRoot ||
				formatTIME(HeaderTime(block.Header)) != e.HeaderTime {
				t.Errorf("%s: matched %s, total %d, header %s %s %s", f.name, Display(block.Matched), block.TotalTxs,
					Display(HeaderHash(block.Header)), Display(MerkleRootField(block.Header)), formatTIME(HeaderTime(block.Header)))
			}
		case "header":
			if len(b) != HeaderOctets || e.MeetsTarget == nil {
				t.Fatalf("%s: %d octets", f.name, len(b))
			}
			h := Header(b)
			if Display(HeaderHash(h)) != e.HeaderHash || Display(MerkleRootField(h)) != e.HeaderMerkleRoot ||
				formatTIME(HeaderTime(h)) != e.HeaderTime || MeetsTarget(h) != *e.MeetsTarget {
				t.Errorf("%s: %s %s %s %v", f.name, Display(HeaderHash(h)), Display(MerkleRootField(h)), formatTIME(HeaderTime(h)), MeetsTarget(h))
			}
		default:
			t.Fatalf("%s: kind %q", f.name, f.Kind)
		}
	}
	if len(files) != 17 || witness != 3 || kinds["transaction"] != 9 || kinds["merkleblock"] != 6 || kinds["header"] != 2 {
		t.Fatalf("%d files, %d witness files, kinds %v", len(files), witness, kinds)
	}
}

// A transaction's hash is SHA-256d of its bytes as carried (SPEC.md
// section 3.4), never the hash of its serialization without witness data:
// for each -witness file, TransactionHash is the witness hash the file
// gives, two SHA-256s of its bytes computed here, and differs from the
// txid of the same transaction without its witness, which the file
// without -witness gives. Mutant killed: TransactionHash dropping the BIP
// 144 marker and flag before it hashes (each witness hash differs).
func TestTransactionHash_IsOfTheBytesAsCarried(t *testing.T) {
	byName := map[string]corpusFile{}
	for _, f := range corpus(t) {
		byName[f.name] = f
	}
	for _, name := range []string{"mainnet-540107-transaction", "mainnet-932662-transaction", "regtest-56062-transaction"} {
		withWitness, without := byName[name+"-witness.json"], byName[name+".json"]
		b := bytesOf(t, withWitness.Hex)
		first := sha256.Sum256(b)
		twice := sha256.Sum256(first[:])
		got := TransactionHash(b)
		if got != twice || Display(got) != withWitness.Expected.TransactionHash || Display(got) == without.Expected.TransactionHash ||
			Display(TransactionHash(bytesOf(t, without.Hex))) != without.Expected.TransactionHash {
			t.Errorf("%s: %s, the stripped transaction's %s", name, Display(got), without.Expected.TransactionHash)
		}
	}
}

// Display is rev of a hash, as lowercase hex (the encoding row of SPEC.md
// section 7.3). Mutant killed: Display without the reversal.
func TestDisplay_ReversesTheBytes(t *testing.T) {
	var h [32]byte
	for i := range h {
		h[i] = byte(i)
	}
	if got, want := Display(h), "1f1e1d1c1b1a191817161514131211100f0e0d0c0b0a09080706050403020100"; got != want {
		t.Fatalf("Display = %s, want %s", got, want)
	}
}

// A compact size is read only in its shortest form (SPEC.md section 10.2,
// step 9 (d)): one octet below 0xfd; 0xfd with 2 octets of at least 0xfd;
// 0xfe with 4 octets above 0xffff; 0xff with 8 octets above 0xffffffff;
// and nothing when the octets run out. Mutants killed: each bound off by
// one (0xfd 0xfc 0x00, 0xfe 0xffff and 0xff 0xffffffff read, or 0xfd
// 0xfd 0x00, 0xfe 0x10000 and 0xff 0x100000000 refused); 0xfe read with
// 8 octets.
func TestCompactSize_ShortestFormOnly(t *testing.T) {
	cases := []struct {
		in   string
		n    uint64
		rest string
		ok   bool
	}{
		{"00", 0, "", true},
		{"fcaa", 0xfc, "aa", true},
		{"fdfc00", 0, "", false},
		{"fdfd00aa", 0xfd, "aa", true},
		{"fdffff", 0xffff, "", true},
		{"feffff0000", 0, "", false},
		{"fe00000100", 0x10000, "", true},
		{"feffffffffaa", 0xffffffff, "aa", true},
		{"ffffffffff00000000", 0, "", false},
		{"ff0000000001000000", 0x100000000, "", true},
		{"", 0, "", false},
		{"fd01", 0, "", false},
		{"fe010000", 0, "", false},
		{"ff01000000000000", 0, "", false},
	}
	for _, c := range cases {
		n, rest, ok := compactSize(bytesOf(t, c.in))
		if ok != c.ok || (ok && (n != c.n || hex.EncodeToString(rest) != c.rest)) {
			t.Errorf("%s: %d %x %v, want %d %s %v", c.in, n, rest, ok, c.n, c.rest, c.ok)
		}
	}
}

// anyCount is a compact size in its shortest form, chosen from rng: present
// in one octet; any one-octet value; a value in the 0xfd, 0xfe or 0xff
// form; or, in the 0xff form, present plus a multiple of 2^59, a count
// that, at 32 octets a hash, comes modulo 2^64 to present hashes' octets.
// wraps reports the last.
func anyCount(rng *rand.Rand, present int) (count []byte, wraps bool) {
	switch rng.IntN(6) {
	case 0:
		return []byte{byte(present)}, false
	case 1:
		return []byte{byte(rng.IntN(0xfd))}, false
	case 2:
		return binary.LittleEndian.AppendUint16([]byte{0xfd}, uint16(0xfd+rng.IntN(0x10000-0xfd))), false
	case 3:
		return binary.LittleEndian.AppendUint32([]byte{0xfe}, uint32(0x10000+rng.Uint64N(1<<32-0x10000))), false
	case 4:
		return binary.LittleEndian.AppendUint64([]byte{0xff}, 1<<32+rng.Uint64N(1<<63)), false
	default:
		return binary.LittleEndian.AppendUint64([]byte{0xff}, uint64(1+rng.IntN(31))<<59+uint64(present)), true
	}
}

// ParseTransaction and ParseMerkleBlock never panic: each input gives a
// value, or one of the package's refusals and the zero value. The inputs,
// all fixed by the seeds here, are every prefix of every corpus file; 2,000
// random byte strings; 2,000 corpus files with up to four octets
// overwritten, some cut short; 2,000 txoutproofs built here, each a random
// header, a total_txs within the bound, a count of hashes from anyCount, up
// to three hashes, up to three flag octets and, in some, a trailing byte;
// and 2,000 transactions built the same way, a version, a count of inputs
// from anyCount and random bytes. Every input is given to both. Mutant
// killed: ParseMerkleBlock's check that the count of hashes is not past the
// bytes left out (a built txoutproof whose count wraps, by anyCount's last
// form, panics with makeslice: len out of range).
func TestParsers_NeverPanic(t *testing.T) {
	txRefusals := []error{RefusedParse, RefusedLength, RefusedOpReturn}
	blockRefusals := []error{RefusedParse, RefusedTotalTxs, RefusedPartialTree, RefusedDuplicatePair, RefusedMatches, RefusedMerkleRoot, RefusedTarget}
	judge := func(b []byte) {
		defer func() {
			if r := recover(); r != nil {
				t.Fatalf("%x panics: %v", b, r)
			}
		}()
		pushed, err := ParseTransaction(b)
		if err != nil && (!slices.Contains(txRefusals, err) || pushed != [32]byte{}) {
			t.Errorf("ParseTransaction(%x) = %x, %v", b, pushed, err)
		}
		block, err := ParseMerkleBlock(b)
		switch {
		case err != nil && (!slices.Contains(blockRefusals, err) || block != MerkleBlock{}):
			t.Errorf("ParseMerkleBlock(%x) = %+v, %v", b, block, err)
		case err == nil && (block.TotalTxs < 1 || block.TotalTxs > specdata.BitcoinMaxTotalTxs || block.Header != Header(b[:HeaderOctets])):
			t.Errorf("ParseMerkleBlock(%x) = %+v", b, block)
		}
	}
	var files [][]byte
	for _, f := range corpus(t) {
		files = append(files, bytesOf(t, f.Hex))
	}
	for _, b := range files {
		for n := range len(b) + 1 {
			judge(b[:n])
		}
	}
	rng := rand.New(rand.NewPCG(1, 2))
	randomBytes := func(n int) []byte {
		out := make([]byte, n)
		for i := range out {
			out[i] = byte(rng.UintN(256))
		}
		return out
	}
	wrapping := 0
	for range 2000 {
		judge(randomBytes(rng.IntN(300)))

		changed := slices.Clone(files[rng.IntN(len(files))])
		for range 1 + rng.IntN(4) {
			changed[rng.IntN(len(changed))] = byte(rng.UintN(256))
		}
		if rng.IntN(4) == 0 {
			changed = changed[:rng.IntN(len(changed))]
		}
		judge(changed)

		present := rng.IntN(4)
		count, wraps := anyCount(rng, present)
		block := binary.LittleEndian.AppendUint32(randomBytes(HeaderOctets), uint32(1+rng.IntN(int(specdata.BitcoinMaxTotalTxs))))
		block = append(append(block, count...), randomBytes(32*present)...)
		flags := rng.IntN(4)
		block = append(append(block, byte(flags)), randomBytes(flags)...)
		if rng.IntN(8) == 0 {
			block = append(block, byte(rng.UintN(256)))
		}
		if wraps {
			wrapping++
		}
		judge(block)

		inputs, _ := anyCount(rng, rng.IntN(4))
		judge(append(append(randomBytes(versionOctets), inputs...), randomBytes(rng.IntN(200))...))
	}
	if wrapping < 100 {
		t.Fatalf("%d of the built txoutproofs have a count that wraps", wrapping)
	}
}
