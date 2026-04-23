// Copyright (c) 2019-2026 Truestamp, Inc.
// SPDX-License-Identifier: MIT

//go:build !windows

package selfupgrade

import (
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

// ErrReplaceUnsupported is returned from Replace on platforms that do not
// support in-place binary replacement (currently Windows). Unix builds
// never return this.
var ErrReplaceUnsupported = errors.New("in-place upgrade not supported on this platform")

// Replace atomically swaps destPath with newBinPath, leaving a timestamped
// backup of the previous binary alongside it. On same-filesystem moves
// this is a single rename; on cross-filesystem it falls back to copy+
// rename (same behavior as install.sh:286-290).
//
// On darwin, the macOS quarantine xattr is cleared via the `xattr` CLI,
// matching install.sh:296-298. Missing `xattr` is NOT a fatal error.
//
// After a successful replace, backups older than 7 days in the binary's
// directory are pruned.
//
// The returned string is the path to the backup of the previous binary,
// or "" if no prior binary existed.
func Replace(destPath, newBinPath string) (backupPath string, err error) {
	// Make sure the new binary is executable before it takes over.
	if err := os.Chmod(newBinPath, 0755); err != nil {
		return "", fmt.Errorf("chmod new binary: %w", err)
	}

	// Create backup if a prior binary exists. We keep a timestamped copy
	// so users can manually revert if the new binary misbehaves.
	if _, err := os.Stat(destPath); err == nil {
		backupPath = destPath + ".bak." + time.Now().UTC().Format("20060102T150405Z")
		if err := os.Rename(destPath, backupPath); err != nil {
			// Cross-filesystem rename fails with EXDEV; fall back to copy.
			if err := copyFile(destPath, backupPath); err != nil {
				return "", fmt.Errorf("backup current binary: %w", err)
			}
			// Remove the original — we've got a copy. Ignore error:
			// the rename below will error out more usefully if this fails.
			_ = os.Remove(destPath)
		}
	}

	// Atomic-replace step: rename the new binary into place.
	if err := os.Rename(newBinPath, destPath); err != nil {
		// Cross-filesystem fallback.
		if err := copyFile(newBinPath, destPath); err != nil {
			return backupPath, fmt.Errorf("install new binary: %w", err)
		}
		_ = os.Remove(newBinPath)
	}

	if err := os.Chmod(destPath, 0755); err != nil {
		return backupPath, fmt.Errorf("chmod installed binary: %w", err)
	}

	if runtime.GOOS == "darwin" {
		clearQuarantineDarwin(destPath)
	}

	// Best-effort prune of old backups (>7 days).
	_ = pruneOldBackups(filepath.Dir(destPath), destPath)

	return backupPath, nil
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.OpenFile(dst, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0755)
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, in); err != nil {
		_ = out.Close()
		_ = os.Remove(dst)
		return err
	}
	return out.Close()
}

// clearQuarantineDarwin invokes `xattr -d com.apple.quarantine <path>`
// iff `xattr` is on PATH. The attribute is routinely absent (binaries
// that didn't arrive via curl/Safari won't carry it), so non-zero exit
// is ignored. Matches install.sh:296-298 exactly.
func clearQuarantineDarwin(path string) {
	if _, err := exec.LookPath("xattr"); err != nil {
		return
	}
	cmd := exec.Command("xattr", "-d", "com.apple.quarantine", path)
	cmd.Stdout = io.Discard
	cmd.Stderr = io.Discard
	_ = cmd.Run()
}

// pruneOldBackups deletes `<destPath>.bak.*` entries in dir whose mtime
// is older than 7 days. Errors are swallowed — cleanup is lazy and
// never load-bearing.
func pruneOldBackups(dir, destPath string) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return err
	}
	prefix := filepath.Base(destPath) + ".bak."
	cutoff := time.Now().Add(-7 * 24 * time.Hour)
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		if !strings.HasPrefix(name, prefix) {
			continue
		}
		info, err := e.Info()
		if err != nil {
			continue
		}
		if info.ModTime().Before(cutoff) {
			_ = os.Remove(filepath.Join(dir, name))
		}
	}
	return nil
}
