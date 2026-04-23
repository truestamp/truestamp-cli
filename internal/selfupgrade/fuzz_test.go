// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: MIT

package selfupgrade

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// FuzzParseSemver: arbitrary strings into the version parser.
func FuzzParseSemver(f *testing.F) {
	for _, s := range []string{
		"", "v1.0.0", "1.0.0", "1.0.0-rc.1",
		"v1.2.3-alpha.1+build.42", "vx.y.z", "1",
	} {
		f.Add(s)
	}
	f.Fuzz(func(t *testing.T, s string) {
		v, err := ParseSemver(s)
		if err != nil {
			return
		}
		// If it parsed, the pre-release predicate must not panic.
		_ = v.IsPreRelease()
	})
}

// FuzzChecksumFor feeds arbitrary bodies into the checksums.txt
// parser. The parser must not panic on malformed content.
func FuzzChecksumFor(f *testing.F) {
	f.Add([]byte(""), "a.tar.gz")
	f.Add([]byte("deadbeef  a.tar.gz\ncafebabe  b.tar.gz\n"), "b.tar.gz")
	f.Add([]byte("# comment\n"), "any")
	f.Add([]byte("malformed line with only one field"), "any")

	f.Fuzz(func(t *testing.T, body []byte, name string) {
		_, _ = ChecksumFor(body, name)
	})
}

// FuzzContainsParentSegment exercises the ".." path-traversal detector.
// This is the defense against an attacker crafting a tarball with
// "../../etc/passwd" entries; a miss here would be a security bug.
func FuzzContainsParentSegment(f *testing.F) {
	for _, s := range []string{
		"", "./foo", "foo/bar", "../etc/passwd", "a/../b",
		"..", "./..", `a\..\b`, "truestamp",
	} {
		f.Add(s)
	}
	f.Fuzz(func(t *testing.T, s string) {
		_ = containsParentSegment(s)
	})
}

// FuzzExtractBinary: the highest-stakes target here. Builds a
// synthetic tar.gz from fuzzer-provided header fields (filename,
// content) and feeds it to ExtractBinary. A successful extract that
// lands outside destDir is a path-traversal exploit.
func FuzzExtractBinary(f *testing.F) {
	f.Add("truestamp", []byte("hello\n"))
	f.Add("../escape", []byte("pwned"))
	f.Add("", []byte{})
	f.Add("truestamp/../../etc/passwd", []byte("evil"))

	f.Fuzz(func(t *testing.T, tarName string, body []byte) {
		// Cap body so the fuzzer doesn't generate multi-GB tarballs.
		if len(body) > 1<<16 {
			return
		}
		// Reject non-UTF8 names defensively — not the thing we're
		// fuzzing. (Real tarballs carry arbitrary bytes but that's
		// filesystem-dependent and not our attack surface here.)
		if !isPrintableName(tarName) {
			return
		}

		tmp := t.TempDir()
		archivePath := filepath.Join(tmp, "archive.tar.gz")
		destDir := filepath.Join(tmp, "dest")
		if err := os.MkdirAll(destDir, 0755); err != nil {
			t.Fatal(err)
		}
		if err := makeFuzzTarGz(archivePath, tarName, body); err != nil {
			return
		}

		outPath, err := ExtractBinary(archivePath, "truestamp", destDir)
		if err != nil {
			return
		}

		// Invariant: extracted path MUST be inside destDir.
		resolvedDest, _ := filepath.Abs(destDir)
		resolvedOut, _ := filepath.Abs(outPath)
		if !strings.HasPrefix(resolvedOut, resolvedDest) {
			t.Errorf("path-traversal: ExtractBinary wrote %q outside destDir %q (tarName=%q)",
				resolvedOut, resolvedDest, tarName)
		}
	})
}

// makeFuzzTarGz builds a minimal tar.gz with one regular-file entry.
func makeFuzzTarGz(path, name string, body []byte) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	gw := gzip.NewWriter(f)
	tw := tar.NewWriter(gw)
	hdr := &tar.Header{
		Name: name,
		Mode: 0755,
		Size: int64(len(body)),
	}
	if err := tw.WriteHeader(hdr); err != nil {
		_ = tw.Close()
		_ = gw.Close()
		return err
	}
	if _, err := tw.Write(body); err != nil {
		_ = tw.Close()
		_ = gw.Close()
		return err
	}
	if err := tw.Close(); err != nil {
		_ = gw.Close()
		return err
	}
	return gw.Close()
}

// isPrintableName keeps the fuzzer within the set of names a real
// GoReleaser tarball could plausibly carry — no NUL bytes, no control
// characters — so the test focuses on semantic escapes (../) rather
// than OS-filesystem edge cases.
func isPrintableName(s string) bool {
	for _, r := range s {
		if r < 0x20 || r == 0x7f {
			return false
		}
	}
	// Trim anything pathologically long so we don't blow out ARG_MAX
	// on synthesized paths.
	return len(s) < 256
}

// FuzzExtractBinary_Raw feeds arbitrary bytes (not necessarily a valid
// tar.gz) to ExtractBinary. The parser should reject every invalid
// archive with a typed error, never a panic.
func FuzzExtractBinary_Raw(f *testing.F) {
	f.Add([]byte{})
	f.Add([]byte("not a tarball"))
	// Minimal valid gzip+tar from the helper, pre-computed.
	valid, _ := minimalValidTarGz("truestamp", []byte("hi"))
	f.Add(valid)

	f.Fuzz(func(t *testing.T, data []byte) {
		if len(data) > 1<<20 {
			return
		}
		tmp := t.TempDir()
		archivePath := filepath.Join(tmp, "archive.tar.gz")
		if err := os.WriteFile(archivePath, data, 0644); err != nil {
			t.Fatal(err)
		}
		_, _ = ExtractBinary(archivePath, "truestamp", tmp)
	})
}

func minimalValidTarGz(name string, body []byte) ([]byte, error) {
	var buf bytes.Buffer
	gw := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gw)
	_ = tw.WriteHeader(&tar.Header{Name: name, Mode: 0755, Size: int64(len(body))})
	_, _ = tw.Write(body)
	_ = tw.Close()
	_ = gw.Close()
	return buf.Bytes(), nil
}
