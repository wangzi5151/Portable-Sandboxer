package sandbox

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCleanupStaleOnlyRemovesPrefixedDirs(t *testing.T) {
	root := os.TempDir()
	stale := filepath.Join(root, workspacePrefix+"test-stale")
	keep := filepath.Join(root, "portable-keep-me-test")
	if err := os.MkdirAll(stale, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(keep, 0o700); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(keep) })

	removed, err := CleanupStale()
	if err != nil {
		t.Fatal(err)
	}
	if removed < 1 {
		t.Fatalf("expected at least one workspace removed, got %d", removed)
	}
	if _, err := os.Stat(stale); !os.IsNotExist(err) {
		t.Fatal("stale workspace should be gone")
	}
	if _, err := os.Stat(keep); err != nil {
		t.Fatal("unprefixed directory must not be touched")
	}
}

func TestCopyTreePreservesSymlink(t *testing.T) {
	src := t.TempDir()
	dst := filepath.Join(t.TempDir(), "copy")
	if err := os.WriteFile(filepath.Join(src, "a.txt"), []byte("hello"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink("a.txt", filepath.Join(src, "link")); err != nil {
		t.Skipf("symlinks unsupported: %v", err)
	}
	if err := copyTree(src, dst); err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(filepath.Join(dst, "a.txt"))
	if err != nil || string(b) != "hello" {
		t.Fatalf("file copy failed: %v %q", err, b)
	}
	target, err := os.Readlink(filepath.Join(dst, "link"))
	if err != nil || target != "a.txt" {
		t.Fatalf("symlink not preserved: %v %q", err, target)
	}
}
