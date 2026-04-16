// Copyright (c) 2021-2026 Truestamp, Inc.
// SPDX-License-Identifier: MIT

package selfupgrade

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestExtractBinary(t *testing.T) {
	archive := makeTarGz(t, []tarEntry{
		{name: "truestamp", content: []byte("#!/bin/sh\necho ok\n"), mode: 0755},
		{name: "LICENSE", content: []byte("MIT"), mode: 0644},
	})
	archivePath := writeToTemp(t, "sample_0.3.0.tar.gz", archive)

	dest := t.TempDir()
	got, err := ExtractBinary(archivePath, "truestamp", dest)
	if err != nil {
		t.Fatalf("ExtractBinary err: %v", err)
	}
	if filepath.Base(got) != "truestamp" {
		t.Errorf("ExtractBinary returned %q, want basename 'truestamp'", got)
	}
	body, err := os.ReadFile(got)
	if err != nil {
		t.Fatalf("read extracted binary: %v", err)
	}
	if !bytes.Equal(body, []byte("#!/bin/sh\necho ok\n")) {
		t.Errorf("extracted content mismatch: got %q", string(body))
	}
	// Executable bit.
	info, err := os.Stat(got)
	if err != nil {
		t.Fatalf("stat: %v", err)
	}
	if info.Mode().Perm()&0100 == 0 {
		t.Errorf("extracted binary is not executable: perm %v", info.Mode().Perm())
	}
}

func TestExtractBinary_missing(t *testing.T) {
	archive := makeTarGz(t, []tarEntry{
		{name: "README.md", content: []byte("hi"), mode: 0644},
	})
	archivePath := writeToTemp(t, "sample.tar.gz", archive)

	dest := t.TempDir()
	_, err := ExtractBinary(archivePath, "truestamp", dest)
	if !errors.Is(err, ErrBinaryNotInArchive) {
		t.Errorf("ExtractBinary missing: got %v, want ErrBinaryNotInArchive", err)
	}
}

func TestExtractBinary_rejectsTraversal(t *testing.T) {
	archive := makeTarGz(t, []tarEntry{
		{name: "../../etc/truestamp", content: []byte("pwned"), mode: 0755},
	})
	archivePath := writeToTemp(t, "bad.tar.gz", archive)

	dest := t.TempDir()
	_, err := ExtractBinary(archivePath, "truestamp", dest)
	if err == nil {
		t.Fatal("ExtractBinary with ../../ path: want error, got nil")
	}
}

// --- helpers -----------------------------------------------------------------

type tarEntry struct {
	name    string
	content []byte
	mode    int64
}

func makeTarGz(t *testing.T, entries []tarEntry) []byte {
	t.Helper()
	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gz)
	for _, e := range entries {
		hdr := &tar.Header{
			Name: e.name,
			Mode: e.mode,
			Size: int64(len(e.content)),
		}
		if err := tw.WriteHeader(hdr); err != nil {
			t.Fatalf("tar header: %v", err)
		}
		if _, err := tw.Write(e.content); err != nil {
			t.Fatalf("tar write: %v", err)
		}
	}
	if err := tw.Close(); err != nil {
		t.Fatalf("tar close: %v", err)
	}
	if err := gz.Close(); err != nil {
		t.Fatalf("gz close: %v", err)
	}
	return buf.Bytes()
}

func writeToTemp(t *testing.T, name string, content []byte) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), name)
	if err := os.WriteFile(path, content, 0644); err != nil {
		t.Fatal(err)
	}
	return path
}
