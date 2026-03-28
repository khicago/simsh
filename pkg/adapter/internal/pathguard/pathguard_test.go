package pathguard

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestCheckAllowsMissingDescendantWithinRoot(t *testing.T) {
	root := t.TempDir()
	candidate := filepath.Join(root, "nested", "missing", "note.txt")

	if err := Check(root, candidate); err != nil {
		t.Fatalf("Check(%q, %q) error = %v, want nil", root, candidate, err)
	}
}

func TestCheckRejectsDirectAndNestedSymlinkEscapes(t *testing.T) {
	root := t.TempDir()
	outside := t.TempDir()
	escapeLink := filepath.Join(root, "escape")
	if err := os.Symlink(outside, escapeLink); err != nil {
		t.Fatalf("create symlink failed: %v", err)
	}

	for _, candidate := range []string{
		filepath.Join(escapeLink, "pwned.txt"),
		filepath.Join(escapeLink, "subdir", "pwned.txt"),
	} {
		if err := Check(root, candidate); !errors.Is(err, ErrEscape) {
			t.Fatalf("Check(%q, %q) error = %v, want ErrEscape", root, candidate, err)
		}
	}
}

func TestResolveWithExistingPrefixResolvesSymlinkAncestorAndKeepsSuffix(t *testing.T) {
	root := t.TempDir()
	realDir := filepath.Join(root, "real")
	if err := os.MkdirAll(realDir, 0o755); err != nil {
		t.Fatalf("mkdir real dir failed: %v", err)
	}
	linkPath := filepath.Join(root, "link")
	if err := os.Symlink(realDir, linkPath); err != nil {
		t.Fatalf("create symlink failed: %v", err)
	}

	candidate := filepath.Join(linkPath, "subdir", "file.txt")
	got, err := resolveWithExistingPrefix(candidate)
	if err != nil {
		t.Fatalf("resolveWithExistingPrefix(%q) error = %v", candidate, err)
	}

	resolvedRealDir, err := filepath.EvalSymlinks(realDir)
	if err != nil {
		t.Fatalf("EvalSymlinks(%q) error = %v", realDir, err)
	}
	want := filepath.Join(resolvedRealDir, "subdir", "file.txt")
	if got != want {
		t.Fatalf("resolveWithExistingPrefix(%q) = %q, want %q", candidate, got, want)
	}
}
