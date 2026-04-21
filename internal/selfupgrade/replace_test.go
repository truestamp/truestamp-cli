// Copyright (c) 2021-2026 Truestamp, Inc.
// SPDX-License-Identifier: MIT

//go:build !windows

package selfupgrade

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestReplace_FreshInstall(t *testing.T) {
	dir := t.TempDir()
	dest := filepath.Join(dir, "truestamp")
	newBin := filepath.Join(dir, "truestamp-new")
	if err := os.WriteFile(newBin, []byte("#!/bin/sh\necho new\n"), 0644); err != nil {
		t.Fatal(err)
	}

	backup, err := Replace(dest, newBin)
	if err != nil {
		t.Fatalf("Replace: %v", err)
	}
	if backup != "" {
		t.Errorf("fresh install: expected empty backup path, got %q", backup)
	}
	// New binary should be in place with 0755 perms.
	info, err := os.Stat(dest)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0755 {
		t.Errorf("installed binary perms: got %o, want 0755", info.Mode().Perm())
	}
}

func TestReplace_WithExistingBinary_CreatesBackup(t *testing.T) {
	dir := t.TempDir()
	dest := filepath.Join(dir, "truestamp")
	if err := os.WriteFile(dest, []byte("old"), 0755); err != nil {
		t.Fatal(err)
	}
	newBin := filepath.Join(dir, "truestamp-new")
	if err := os.WriteFile(newBin, []byte("new"), 0644); err != nil {
		t.Fatal(err)
	}

	backup, err := Replace(dest, newBin)
	if err != nil {
		t.Fatalf("Replace: %v", err)
	}
	if backup == "" {
		t.Error("expected a backup path for in-place replacement")
	}
	body, err := os.ReadFile(backup)
	if err != nil {
		t.Fatalf("backup missing: %v", err)
	}
	if string(body) != "old" {
		t.Errorf("backup contents: got %q, want old", body)
	}
	body, err = os.ReadFile(dest)
	if err != nil {
		t.Fatal(err)
	}
	if string(body) != "new" {
		t.Errorf("dest contents: got %q, want new", body)
	}
}

func TestReplace_MissingNewBin(t *testing.T) {
	dir := t.TempDir()
	dest := filepath.Join(dir, "truestamp")
	if _, err := Replace(dest, filepath.Join(dir, "does-not-exist")); err == nil {
		t.Error("expected error when new binary doesn't exist")
	}
}

func TestCopyFile(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "src.bin")
	dst := filepath.Join(dir, "dst.bin")
	content := []byte("hello world")
	if err := os.WriteFile(src, content, 0644); err != nil {
		t.Fatal(err)
	}
	if err := copyFile(src, dst); err != nil {
		t.Fatalf("copyFile: %v", err)
	}
	got, err := os.ReadFile(dst)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != string(content) {
		t.Errorf("copied: got %q, want %q", got, content)
	}
}

func TestCopyFile_MissingSource(t *testing.T) {
	dir := t.TempDir()
	if err := copyFile("/does/not/exist", filepath.Join(dir, "out")); err == nil {
		t.Error("expected error")
	}
}

func TestCopyFile_UnwriteableDest(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "src")
	_ = os.WriteFile(src, []byte("x"), 0644)
	dst := filepath.Join(dir, "missing", "out")
	if err := copyFile(src, dst); err == nil {
		t.Error("expected error for unwriteable dest")
	}
}

func TestPruneOldBackups(t *testing.T) {
	dir := t.TempDir()
	dest := filepath.Join(dir, "truestamp")

	// Create three backup files: one fresh, two old.
	fresh := filepath.Join(dir, "truestamp.bak.fresh")
	old1 := filepath.Join(dir, "truestamp.bak.old1")
	old2 := filepath.Join(dir, "truestamp.bak.old2")
	other := filepath.Join(dir, "unrelated.txt")
	for _, p := range []string{fresh, old1, old2, other} {
		if err := os.WriteFile(p, []byte("x"), 0644); err != nil {
			t.Fatal(err)
		}
	}
	// Backdate the two "old" entries past the 7-day cutoff.
	past := time.Now().Add(-30 * 24 * time.Hour)
	_ = os.Chtimes(old1, past, past)
	_ = os.Chtimes(old2, past, past)

	if err := pruneOldBackups(dir, dest); err != nil {
		t.Fatal(err)
	}

	if _, err := os.Stat(fresh); err != nil {
		t.Errorf("fresh backup should remain: %v", err)
	}
	if _, err := os.Stat(old1); !os.IsNotExist(err) {
		t.Errorf("old backup should be pruned, err=%v", err)
	}
	if _, err := os.Stat(old2); !os.IsNotExist(err) {
		t.Errorf("old backup should be pruned, err=%v", err)
	}
	if _, err := os.Stat(other); err != nil {
		t.Errorf("unrelated file should remain: %v", err)
	}
}

func TestPruneOldBackups_MissingDir(t *testing.T) {
	if err := pruneOldBackups("/does/not/exist", "/dest"); err == nil {
		t.Error("expected error for missing dir")
	}
}

func TestClearQuarantineDarwin_NoOp(t *testing.T) {
	// Just exercise the function; on non-darwin it may return without
	// running xattr. On darwin without xattr it also returns. Either way
	// no panic, no error.
	dir := t.TempDir()
	path := filepath.Join(dir, "x")
	_ = os.WriteFile(path, []byte("x"), 0644)
	clearQuarantineDarwin(path)
}
