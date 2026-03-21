package localfs

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/khicago/simsh/pkg/contract"
)

func mustFullPolicy(t *testing.T) contract.ExecutionPolicy {
	t.Helper()
	p, err := contract.PolicyPreset(string(contract.WriteModeFull))
	if err != nil {
		t.Fatalf("policy preset failed: %v", err)
	}
	return p
}

func TestRequireAbsolutePathRejectsSymlinkEscape(t *testing.T) {
	root := t.TempDir()
	outside := t.TempDir()
	if err := os.Symlink(outside, filepath.Join(root, "link")); err != nil {
		t.Fatalf("create symlink failed: %v", err)
	}

	ops, err := NewOps(Options{RootDir: root, Policy: mustFullPolicy(t)})
	if err != nil {
		t.Fatalf("new ops failed: %v", err)
	}

	_, err = ops.RequireAbsolutePath(filepath.ToSlash(filepath.Join(root, "link", "x.txt")))
	if err == nil {
		t.Fatalf("expected symlink escape to be rejected")
	}
}

func TestMakeDirRejectsSymlinkEscape(t *testing.T) {
	root := t.TempDir()
	outside := t.TempDir()
	if err := os.Symlink(outside, filepath.Join(root, "link")); err != nil {
		t.Fatalf("create symlink failed: %v", err)
	}

	ops, err := NewOps(Options{RootDir: root, Policy: mustFullPolicy(t)})
	if err != nil {
		t.Fatalf("new ops failed: %v", err)
	}

	target := filepath.ToSlash(filepath.Join(root, "link", "newdir"))
	if err := ops.MakeDir(context.Background(), target); err == nil {
		t.Fatalf("expected mkdir through symlink escape to fail")
	}
	if _, err := os.Stat(filepath.Join(outside, "newdir")); !os.IsNotExist(err) {
		t.Fatalf("outside path unexpectedly created: err=%v", err)
	}
}

func TestRemoveFileRejectsSymlinkEscape(t *testing.T) {
	root := t.TempDir()
	outside := t.TempDir()
	if err := os.Symlink(outside, filepath.Join(root, "link")); err != nil {
		t.Fatalf("create symlink failed: %v", err)
	}
	victim := filepath.Join(outside, "victim.txt")
	if err := os.WriteFile(victim, []byte("x"), 0o644); err != nil {
		t.Fatalf("write victim failed: %v", err)
	}

	ops, err := NewOps(Options{RootDir: root, Policy: mustFullPolicy(t)})
	if err != nil {
		t.Fatalf("new ops failed: %v", err)
	}

	target := filepath.ToSlash(filepath.Join(root, "link", "victim.txt"))
	if err := ops.RemoveFile(context.Background(), target); err == nil {
		t.Fatalf("expected remove through symlink escape to fail")
	}
	if _, err := os.Stat(victim); err != nil {
		t.Fatalf("outside file should remain untouched: %v", err)
	}
}

func TestListChildrenRejectsSymlinkEscape(t *testing.T) {
	root := t.TempDir()
	outside := t.TempDir()
	if err := os.WriteFile(filepath.Join(outside, "secret.txt"), []byte("x"), 0o644); err != nil {
		t.Fatalf("write outside file failed: %v", err)
	}
	if err := os.Symlink(outside, filepath.Join(root, "link")); err != nil {
		t.Fatalf("create symlink failed: %v", err)
	}

	ops, err := NewOps(Options{RootDir: root, Policy: mustFullPolicy(t)})
	if err != nil {
		t.Fatalf("new ops failed: %v", err)
	}

	target := filepath.ToSlash(filepath.Join(root, "link"))
	if _, err := ops.ListChildren(context.Background(), target); err == nil {
		t.Fatalf("expected list through symlink escape to fail")
	}
}

func TestRemoveDirRejectsSymlinkEscape(t *testing.T) {
	root := t.TempDir()
	outside := t.TempDir()
	if err := os.Symlink(outside, filepath.Join(root, "link")); err != nil {
		t.Fatalf("create symlink failed: %v", err)
	}
	targetDir := filepath.Join(outside, "victim")
	if err := os.MkdirAll(targetDir, 0o755); err != nil {
		t.Fatalf("create outside dir failed: %v", err)
	}

	ops, err := NewOps(Options{RootDir: root, Policy: mustFullPolicy(t)})
	if err != nil {
		t.Fatalf("new ops failed: %v", err)
	}

	target := filepath.ToSlash(filepath.Join(root, "link", "victim"))
	if err := ops.RemoveDir(context.Background(), target); err == nil {
		t.Fatalf("expected rmdir through symlink escape to fail")
	}
	if _, err := os.Stat(targetDir); err != nil {
		t.Fatalf("outside dir should remain untouched: %v", err)
	}
}

func TestWriteFileRejectsNestedSymlinkEscape(t *testing.T) {
	root := t.TempDir()
	outside := t.TempDir()
	if err := os.Symlink(outside, filepath.Join(root, "link")); err != nil {
		t.Fatalf("create symlink failed: %v", err)
	}

	ops, err := NewOps(Options{RootDir: root, Policy: mustFullPolicy(t)})
	if err != nil {
		t.Fatalf("new ops failed: %v", err)
	}

	target := filepath.ToSlash(filepath.Join(root, "link", "subdir", "pwned.txt"))
	if err := ops.WriteFile(context.Background(), target, "owned"); err == nil {
		t.Fatalf("expected nested write through symlink escape to fail")
	}
	if _, err := os.Stat(filepath.Join(outside, "subdir", "pwned.txt")); !os.IsNotExist(err) {
		t.Fatalf("outside path unexpectedly written: err=%v", err)
	}
}

func TestCheckPathOpRejectsNestedSymlinkEscape(t *testing.T) {
	root := t.TempDir()
	outside := t.TempDir()
	if err := os.Symlink(outside, filepath.Join(root, "link")); err != nil {
		t.Fatalf("create symlink failed: %v", err)
	}

	ops, err := NewOps(Options{RootDir: root, Policy: mustFullPolicy(t)})
	if err != nil {
		t.Fatalf("new ops failed: %v", err)
	}

	target := filepath.ToSlash(filepath.Join(root, "link", "subdir", "pwned.txt"))
	if err := ops.CheckPathOp(context.Background(), contract.PathOpWrite, target); err == nil {
		t.Fatalf("expected nested write preflight through symlink escape to fail")
	}
}

func TestRemoveDirRejectsRootDir(t *testing.T) {
	root := t.TempDir()
	ops, err := NewOps(Options{RootDir: root, Policy: mustFullPolicy(t)})
	if err != nil {
		t.Fatalf("new ops failed: %v", err)
	}
	rootPath := filepath.ToSlash(root)
	if err := ops.RemoveDir(context.Background(), rootPath); err == nil {
		t.Fatalf("expected remove root directory to fail")
	}
}
