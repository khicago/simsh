package localfs

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/khicago/simsh/pkg/contract"
)

func TestNewOpsAppliesDefaultsAndNormalizesState(t *testing.T) {
	root := filepath.Join(t.TempDir(), "workspace")
	if err := os.MkdirAll(root, 0o755); err != nil {
		t.Fatalf("mkdir root failed: %v", err)
	}

	ops, err := NewOps(Options{
		RootDir:             filepath.Join(root, "."),
		Policy:              contract.ExecutionPolicy{WriteMode: contract.WriteModeWriteLimited},
		WriteLimitedMaxByte: 12,
		CommandAliases: map[string][]string{
			" ll ": {" ls ", "-l"},
			"bad name": {"echo"},
		},
		EnvVars: map[string]string{
			"FOO":     "bar",
			"bad-key": "skip",
		},
		RCFiles:  []string{" /etc/a ", "/etc/a", "/etc/b"},
		PathEnv:  []string{"/bin"},
		Profile:  "",
		AuditSink: nil,
	})
	if err != nil {
		t.Fatalf("NewOps(...) error = %v", err)
	}

	wantRoot := filepath.ToSlash(filepath.Clean(root))
	if ops.RootDir != wantRoot || ops.WorkingDir != wantRoot {
		t.Fatalf("NewOps(...).RootDir/WorkingDir = (%q, %q), want (%q, %q)", ops.RootDir, ops.WorkingDir, wantRoot, wantRoot)
	}
	if ops.Profile != contract.ProfileCoreStrict {
		t.Errorf("NewOps(...).Profile = %q, want %q", ops.Profile, contract.ProfileCoreStrict)
	}
	if ops.Policy.WriteMode != contract.WriteModeWriteLimited || ops.Policy.MaxWriteBytes != 12 {
		t.Errorf("NewOps(...).Policy = %#v, want write-limited with MaxWriteBytes=12", ops.Policy)
	}
	if !reflect.DeepEqual(ops.CommandAliases, map[string][]string{"ll": {"ls", "-l"}}) {
		t.Errorf("NewOps(...).CommandAliases = %#v, want normalized ll alias", ops.CommandAliases)
	}
	if !reflect.DeepEqual(ops.EnvVars, map[string]string{"FOO": "bar"}) {
		t.Errorf("NewOps(...).EnvVars = %#v, want normalized env vars", ops.EnvVars)
	}
	if !reflect.DeepEqual(ops.RCFiles, []string{"/etc/a", "/etc/b"}) {
		t.Errorf("NewOps(...).RCFiles = %#v, want normalized rc files", ops.RCFiles)
	}
	if !reflect.DeepEqual(ops.PathEnv, []string{"/bin"}) {
		t.Errorf("NewOps(...).PathEnv = %#v, want %#v", ops.PathEnv, []string{"/bin"})
	}

	if _, err := NewOps(Options{RootDir: root, Profile: "bad-profile"}); err == nil {
		t.Fatalf("NewOps(...) unexpectedly accepted invalid profile")
	}
}

func TestReadListSearchAndDescribePath(t *testing.T) {
	root := t.TempDir()
	docsDir := filepath.Join(root, "docs")
	if err := os.MkdirAll(filepath.Join(docsDir, "nested"), 0o755); err != nil {
		t.Fatalf("mkdir docs failed: %v", err)
	}
	if err := os.WriteFile(filepath.Join(docsDir, "nested", "guide.txt"), []byte("line1\nline2\n"), 0o644); err != nil {
		t.Fatalf("write file failed: %v", err)
	}
	if err := os.WriteFile(filepath.Join(docsDir, "README.md"), []byte("# docs\n"), 0o644); err != nil {
		t.Fatalf("write readme failed: %v", err)
	}

	ops, err := NewOps(Options{RootDir: root, Policy: mustFullPolicy(t)})
	if err != nil {
		t.Fatalf("NewOps(...) error = %v", err)
	}

	readmePath := filepath.ToSlash(filepath.Join(root, "docs", "README.md"))
	gotRaw, err := ops.ReadRawContent(context.Background(), readmePath)
	if err != nil || gotRaw != "# docs\n" {
		t.Fatalf("ReadRawContent(%q) = (%q, %v), want (%q, nil)", readmePath, gotRaw, err, "# docs\n")
	}

	docsPath := filepath.ToSlash(filepath.Join(root, "docs"))
	children, err := ops.ListChildren(context.Background(), docsPath)
	if err != nil {
		t.Fatalf("ListChildren(%q) error = %v", docsPath, err)
	}
	wantChildren := []string{
		filepath.ToSlash(filepath.Join(root, "docs", "README.md")),
		filepath.ToSlash(filepath.Join(root, "docs", "nested")),
	}
	if !reflect.DeepEqual(children, wantChildren) {
		t.Errorf("ListChildren(%q) = %#v, want %#v", docsPath, children, wantChildren)
	}

	if isDir, err := ops.IsDirPath(context.Background(), docsPath); err != nil || !isDir {
		t.Fatalf("IsDirPath(%q) = (%t, %v), want (true, nil)", docsPath, isDir, err)
	}

	filesUnder, err := ops.CollectFilesUnder(context.Background(), docsPath)
	if err != nil {
		t.Fatalf("CollectFilesUnder(%q) error = %v", docsPath, err)
	}
	wantFiles := []string{
		filepath.ToSlash(filepath.Join(root, "docs", "README.md")),
		filepath.ToSlash(filepath.Join(root, "docs", "nested", "guide.txt")),
	}
	if !reflect.DeepEqual(filesUnder, wantFiles) {
		t.Errorf("CollectFilesUnder(%q) = %#v, want %#v", docsPath, filesUnder, wantFiles)
	}

	if _, err := ops.ResolveSearchPaths(context.Background(), docsPath, false); err == nil {
		t.Fatalf("ResolveSearchPaths(%q, false) unexpectedly succeeded", docsPath)
	}
	searchPaths, err := ops.ResolveSearchPaths(context.Background(), docsPath, true)
	if err != nil {
		t.Fatalf("ResolveSearchPaths(%q, true) error = %v", docsPath, err)
	}
	if !reflect.DeepEqual(searchPaths, wantFiles) {
		t.Errorf("ResolveSearchPaths(%q, true) = %#v, want %#v", docsPath, searchPaths, wantFiles)
	}

	meta, err := ops.DescribePath(context.Background(), filepath.ToSlash(filepath.Join(root, "docs", "nested", "guide.txt")))
	if err != nil {
		t.Fatalf("DescribePath(guide.txt) error = %v", err)
	}
	if meta.Kind != "file" || meta.LineCount != 2 || meta.Access != contract.PathAccessReadWrite {
		t.Errorf("DescribePath(guide.txt) = %#v, want rw file with 2 lines", meta)
	}
}

func TestWriteAppendEditAndRemoveFlows(t *testing.T) {
	root := t.TempDir()
	ops, err := NewOps(Options{RootDir: root, Policy: mustFullPolicy(t)})
	if err != nil {
		t.Fatalf("NewOps(...) error = %v", err)
	}

	ctx := context.Background()
	target := filepath.ToSlash(filepath.Join(root, "out", "note.txt"))
	if err := ops.WriteFile(ctx, target, "hello"); err != nil {
		t.Fatalf("WriteFile(%q) error = %v", target, err)
	}
	if err := ops.AppendFile(ctx, target, " world"); err != nil {
		t.Fatalf("AppendFile(%q) error = %v", target, err)
	}
	if err := ops.EditFile(ctx, target, "world", "simsh", false); err != nil {
		t.Fatalf("EditFile(%q) error = %v", target, err)
	}
	raw, err := os.ReadFile(filepath.FromSlash(target))
	if err != nil {
		t.Fatalf("os.ReadFile(%q) error = %v", target, err)
	}
	if got := string(raw); got != "hello simsh" {
		t.Fatalf("file contents after write/append/edit = %q, want %q", got, "hello simsh")
	}

	dirPath := filepath.ToSlash(filepath.Join(root, "scratch"))
	if err := ops.MakeDir(ctx, dirPath); err != nil {
		t.Fatalf("MakeDir(%q) error = %v", dirPath, err)
	}
	if err := ops.RemoveDir(ctx, dirPath); err != nil {
		t.Fatalf("RemoveDir(%q) error = %v", dirPath, err)
	}
	if err := ops.RemoveFile(ctx, target); err != nil {
		t.Fatalf("RemoveFile(%q) error = %v", target, err)
	}
	if _, err := os.Stat(filepath.FromSlash(target)); !os.IsNotExist(err) {
		t.Fatalf("os.Stat(%q) = %v, want not-exist", target, err)
	}
}

func TestWriteLimitedPolicyEnforcedAcrossMutations(t *testing.T) {
	root := t.TempDir()
	policy := contract.ExecutionPolicy{WriteMode: contract.WriteModeWriteLimited, MaxWriteBytes: 4}
	ops, err := NewOps(Options{RootDir: root, Policy: policy})
	if err != nil {
		t.Fatalf("NewOps(...) error = %v", err)
	}

	ctx := context.Background()
	target := filepath.ToSlash(filepath.Join(root, "note.txt"))
	if err := ops.WriteFile(ctx, target, "hello"); err == nil {
		t.Fatalf("WriteFile(%q, oversized) unexpectedly succeeded", target)
	}
	if err := ops.AppendFile(ctx, target, "hello"); err == nil {
		t.Fatalf("AppendFile(%q, oversized) unexpectedly succeeded", target)
	}
	if err := os.WriteFile(filepath.FromSlash(target), []byte("ab"), 0o644); err != nil {
		t.Fatalf("seed file failed: %v", err)
	}
	if err := ops.EditFile(ctx, target, "ab", "abcdef", false); err == nil {
		t.Fatalf("EditFile(%q, oversized result) unexpectedly succeeded", target)
	}
}

func TestDescribePathOverrideAndHelperFunctions(t *testing.T) {
	root := t.TempDir()
	wantMeta := contract.PathMeta{Exists: true, Kind: "custom"}
	ops, err := NewOps(Options{
		RootDir: root,
		Policy:  mustFullPolicy(t),
		DescribePath: func(ctx context.Context, pathValue string) (contract.PathMeta, error) {
			return wantMeta, nil
		},
	})
	if err != nil {
		t.Fatalf("NewOps(...) error = %v", err)
	}

	got, err := ops.DescribePath(context.Background(), filepath.ToSlash(filepath.Join(root, "anything.txt")))
	if err != nil {
		t.Fatalf("DescribePath override error = %v", err)
	}
	if !reflect.DeepEqual(got, wantMeta) {
		t.Errorf("DescribePath override = %#v, want %#v", got, wantMeta)
	}

	if got := normalizeCLIPath(filepath.Join(root, "..", filepath.Base(root), "docs", ".", "a.txt")); filepath.Base(got) != "a.txt" {
		t.Errorf("normalizeCLIPath(...) = %q, want path ending with a.txt", got)
	}
	if !pathWithinRoot("/workspace", "/workspace/docs/a.txt") {
		t.Errorf("pathWithinRoot(%q, %q) = false, want true", "/workspace", "/workspace/docs/a.txt")
	}
	if pathWithinRoot("/workspace", "/elsewhere/docs/a.txt") {
		t.Errorf("pathWithinRoot(%q, %q) = true, want false", "/workspace", "/elsewhere/docs/a.txt")
	}
	if err := checkContext(context.TODO()); err != nil {
		t.Errorf("checkContext(non-canceled) error = %v, want nil", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := checkContext(ctx); !errors.Is(err, context.Canceled) {
		t.Errorf("checkContext(canceled) error = %v, want context.Canceled", err)
	}
}
