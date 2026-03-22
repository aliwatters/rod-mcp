package types

import (
	"os"
	"path/filepath"
	"testing"
)

// ---------------------------------------------------------------------------
// copyFile
// ---------------------------------------------------------------------------

func TestCopyFile_BasicCopy(t *testing.T) {
	tmp := t.TempDir()

	src := filepath.Join(tmp, "source.txt")
	dst := filepath.Join(tmp, "dest.txt")

	content := []byte("hello copy world")
	if err := os.WriteFile(src, content, 0644); err != nil {
		t.Fatalf("write source: %v", err)
	}

	if err := copyFile(src, dst); err != nil {
		t.Fatalf("copyFile: %v", err)
	}

	got, err := os.ReadFile(dst)
	if err != nil {
		t.Fatalf("read dest: %v", err)
	}
	if string(got) != string(content) {
		t.Errorf("content mismatch: got %q, want %q", got, content)
	}
}

func TestCopyFile_PreservesMode(t *testing.T) {
	tmp := t.TempDir()

	src := filepath.Join(tmp, "exec.sh")
	dst := filepath.Join(tmp, "exec_copy.sh")

	if err := os.WriteFile(src, []byte("#!/bin/sh\n"), 0755); err != nil {
		t.Fatalf("write source: %v", err)
	}

	if err := copyFile(src, dst); err != nil {
		t.Fatalf("copyFile: %v", err)
	}

	info, err := os.Stat(dst)
	if err != nil {
		t.Fatalf("stat dest: %v", err)
	}

	srcInfo, _ := os.Stat(src)
	if info.Mode() != srcInfo.Mode() {
		t.Errorf("mode mismatch: got %v, want %v", info.Mode(), srcInfo.Mode())
	}
}

func TestCopyFile_OverwritesExisting(t *testing.T) {
	tmp := t.TempDir()

	src := filepath.Join(tmp, "new.txt")
	dst := filepath.Join(tmp, "existing.txt")

	if err := os.WriteFile(dst, []byte("old content"), 0644); err != nil {
		t.Fatalf("write existing dest: %v", err)
	}
	if err := os.WriteFile(src, []byte("new content"), 0644); err != nil {
		t.Fatalf("write source: %v", err)
	}

	if err := copyFile(src, dst); err != nil {
		t.Fatalf("copyFile: %v", err)
	}

	got, _ := os.ReadFile(dst)
	if string(got) != "new content" {
		t.Errorf("overwrite failed: got %q", got)
	}
}

func TestCopyFile_MissingSource_Error(t *testing.T) {
	tmp := t.TempDir()

	src := filepath.Join(tmp, "does-not-exist.txt")
	dst := filepath.Join(tmp, "dest.txt")

	err := copyFile(src, dst)
	if err == nil {
		t.Error("copyFile with missing source: expected error, got nil")
	}
}

func TestCopyFile_MissingDestDir_Error(t *testing.T) {
	tmp := t.TempDir()

	src := filepath.Join(tmp, "source.txt")
	dst := filepath.Join(tmp, "nonexistent-dir", "dest.txt")

	if err := os.WriteFile(src, []byte("data"), 0644); err != nil {
		t.Fatalf("write source: %v", err)
	}

	err := copyFile(src, dst)
	if err == nil {
		t.Error("copyFile with missing dest directory: expected error, got nil")
	}
}

func TestCopyFile_EmptyFile(t *testing.T) {
	tmp := t.TempDir()

	src := filepath.Join(tmp, "empty.txt")
	dst := filepath.Join(tmp, "empty_copy.txt")

	if err := os.WriteFile(src, []byte{}, 0644); err != nil {
		t.Fatalf("write empty source: %v", err)
	}

	if err := copyFile(src, dst); err != nil {
		t.Fatalf("copyFile empty: %v", err)
	}

	info, err := os.Stat(dst)
	if err != nil {
		t.Fatalf("stat empty dest: %v", err)
	}
	if info.Size() != 0 {
		t.Errorf("empty file copy size = %d, want 0", info.Size())
	}
}

// ---------------------------------------------------------------------------
// copyDir
// ---------------------------------------------------------------------------

func TestCopyDir_BasicStructure(t *testing.T) {
	tmp := t.TempDir()

	src := filepath.Join(tmp, "src")
	dst := filepath.Join(tmp, "dst")

	if err := os.MkdirAll(src, 0755); err != nil {
		t.Fatalf("mkdir src: %v", err)
	}
	if err := os.WriteFile(filepath.Join(src, "a.txt"), []byte("aaa"), 0644); err != nil {
		t.Fatalf("write a.txt: %v", err)
	}
	if err := os.WriteFile(filepath.Join(src, "b.txt"), []byte("bbb"), 0644); err != nil {
		t.Fatalf("write b.txt: %v", err)
	}

	if err := copyDir(src, dst); err != nil {
		t.Fatalf("copyDir: %v", err)
	}

	for _, name := range []string{"a.txt", "b.txt"} {
		content, err := os.ReadFile(filepath.Join(dst, name))
		if err != nil {
			t.Errorf("read %s: %v", name, err)
			continue
		}
		srcContent, _ := os.ReadFile(filepath.Join(src, name))
		if string(content) != string(srcContent) {
			t.Errorf("%s content mismatch", name)
		}
	}
}

func TestCopyDir_Recursive(t *testing.T) {
	tmp := t.TempDir()

	src := filepath.Join(tmp, "src")
	subdir := filepath.Join(src, "subdir")
	dst := filepath.Join(tmp, "dst")

	if err := os.MkdirAll(subdir, 0755); err != nil {
		t.Fatalf("mkdir subdir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(src, "root.txt"), []byte("root"), 0644); err != nil {
		t.Fatalf("write root.txt: %v", err)
	}
	if err := os.WriteFile(filepath.Join(subdir, "child.txt"), []byte("child"), 0644); err != nil {
		t.Fatalf("write child.txt: %v", err)
	}

	if err := copyDir(src, dst); err != nil {
		t.Fatalf("copyDir recursive: %v", err)
	}

	// Verify root file.
	got, err := os.ReadFile(filepath.Join(dst, "root.txt"))
	if err != nil {
		t.Fatalf("read dst/root.txt: %v", err)
	}
	if string(got) != "root" {
		t.Errorf("root.txt: got %q, want %q", got, "root")
	}

	// Verify nested file.
	got2, err := os.ReadFile(filepath.Join(dst, "subdir", "child.txt"))
	if err != nil {
		t.Fatalf("read dst/subdir/child.txt: %v", err)
	}
	if string(got2) != "child" {
		t.Errorf("child.txt: got %q, want %q", got2, "child")
	}
}

func TestCopyDir_SkipsSymlinks(t *testing.T) {
	tmp := t.TempDir()

	src := filepath.Join(tmp, "src")
	dst := filepath.Join(tmp, "dst")

	if err := os.MkdirAll(src, 0755); err != nil {
		t.Fatalf("mkdir src: %v", err)
	}

	realFile := filepath.Join(tmp, "real.txt")
	if err := os.WriteFile(realFile, []byte("real"), 0644); err != nil {
		t.Fatalf("write real.txt: %v", err)
	}

	// Create a symlink inside src pointing to the real file outside.
	link := filepath.Join(src, "link.txt")
	if err := os.Symlink(realFile, link); err != nil {
		t.Skip("symlinks not supported: " + err.Error())
	}

	// Also add a regular file.
	if err := os.WriteFile(filepath.Join(src, "regular.txt"), []byte("regular"), 0644); err != nil {
		t.Fatalf("write regular.txt: %v", err)
	}

	if err := copyDir(src, dst); err != nil {
		t.Fatalf("copyDir: %v", err)
	}

	// Symlink should NOT be copied.
	if _, err := os.Lstat(filepath.Join(dst, "link.txt")); err == nil {
		t.Error("symlink should have been skipped but was copied")
	}

	// Regular file should be copied.
	if _, err := os.Stat(filepath.Join(dst, "regular.txt")); err != nil {
		t.Error("regular.txt should have been copied")
	}
}

func TestCopyDir_MissingSource_Error(t *testing.T) {
	tmp := t.TempDir()
	src := filepath.Join(tmp, "does-not-exist")
	dst := filepath.Join(tmp, "dst")

	err := copyDir(src, dst)
	if err == nil {
		t.Error("copyDir with missing source: expected error, got nil")
	}
}

func TestCopyDir_EmptySource(t *testing.T) {
	tmp := t.TempDir()

	src := filepath.Join(tmp, "src")
	dst := filepath.Join(tmp, "dst")

	if err := os.MkdirAll(src, 0755); err != nil {
		t.Fatalf("mkdir src: %v", err)
	}

	if err := copyDir(src, dst); err != nil {
		t.Fatalf("copyDir empty: %v", err)
	}

	info, err := os.Stat(dst)
	if err != nil {
		t.Fatalf("stat dst: %v", err)
	}
	if !info.IsDir() {
		t.Error("dst should be a directory")
	}
}

// ---------------------------------------------------------------------------
// isValidDomainPattern (also tested via cookies_test.go, but profile.go owns it)
// ---------------------------------------------------------------------------

// Note: isValidDomainPattern is defined in profile.go and tested in cookies_test.go.
// Additional edge cases can be added here if needed for profile-specific contexts.
