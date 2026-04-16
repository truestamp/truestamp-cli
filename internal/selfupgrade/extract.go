// Copyright (c) 2021-2026 Truestamp, Inc.
// SPDX-License-Identifier: MIT

package selfupgrade

import (
	"archive/tar"
	"compress/gzip"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// ErrBinaryNotInArchive is returned when the extracted tarball does not
// contain the expected binary entry.
var ErrBinaryNotInArchive = errors.New("binary not found in archive")

// extractMaxBytes caps the amount extracted from the archive at 200 MB
// per file — matches httpclient.DefaultMaxDownloadSize and keeps a
// malicious tarball from filling disk.
const extractMaxBytes = 200 << 20

// ExtractBinary opens archivePath (a .tar.gz produced by GoReleaser) and
// writes the first entry named binaryName into destDir, returning the
// full path of the written file. The file is written with 0755 perms.
//
// Rejects:
//   - tar entries with `..` path components (defense against path
//     traversal — shouldn't happen with GoReleaser output, defensive).
//   - entries larger than extractMaxBytes.
func ExtractBinary(archivePath, binaryName, destDir string) (string, error) {
	f, err := os.Open(archivePath)
	if err != nil {
		return "", fmt.Errorf("open archive: %w", err)
	}
	defer f.Close()

	gz, err := gzip.NewReader(f)
	if err != nil {
		return "", fmt.Errorf("gunzip: %w", err)
	}
	defer gz.Close()

	tr := tar.NewReader(gz)
	for {
		hdr, err := tr.Next()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return "", fmt.Errorf("tar entry: %w", err)
		}
		// Match on the basename — GoReleaser tarballs have the binary
		// at the archive root as plain `truestamp`.
		if filepath.Base(hdr.Name) != binaryName {
			continue
		}
		// Defensive rejection of path-traversal entries.
		cleaned := filepath.Clean(hdr.Name)
		if filepath.IsAbs(cleaned) || hasParentTraversal(cleaned) {
			return "", fmt.Errorf("refusing suspicious archive path: %q", hdr.Name)
		}

		destPath := filepath.Join(destDir, binaryName)
		out, err := os.OpenFile(destPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0755)
		if err != nil {
			return "", fmt.Errorf("create %s: %w", destPath, err)
		}
		n, copyErr := io.Copy(out, io.LimitReader(tr, extractMaxBytes+1))
		closeErr := out.Close()
		if copyErr != nil {
			_ = os.Remove(destPath)
			return "", fmt.Errorf("extract: %w", copyErr)
		}
		if closeErr != nil {
			_ = os.Remove(destPath)
			return "", closeErr
		}
		if n > extractMaxBytes {
			_ = os.Remove(destPath)
			return "", fmt.Errorf("archive entry %q exceeded %d byte cap", hdr.Name, extractMaxBytes)
		}
		return destPath, nil
	}
	return "", fmt.Errorf("%w: %s", ErrBinaryNotInArchive, binaryName)
}

func hasParentTraversal(p string) bool {
	for _, seg := range filepath.SplitList(p) {
		_ = seg
	}
	// filepath.Clean normalizes; a traversal still present after Clean
	// starts with ".." or contains `/../`.
	return p == ".." || len(p) >= 3 && p[:3] == "../" ||
		filepath.ToSlash(p) != filepath.Clean(filepath.ToSlash(p)) ||
		containsParentSegment(filepath.ToSlash(p))
}

func containsParentSegment(p string) bool {
	// Split on forward slash since we've normalized.
	for _, seg := range splitSlash(p) {
		if seg == ".." {
			return true
		}
	}
	return false
}

func splitSlash(p string) []string {
	var out []string
	start := 0
	for i := 0; i <= len(p); i++ {
		if i == len(p) || p[i] == '/' {
			if i > start {
				out = append(out, p[start:i])
			}
			start = i + 1
		}
	}
	return out
}
