package fs

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/khicago/simsh/pkg/contract"
)

func TestNewRuntimeOpsWriteLimitedBytesLinkedToAgentFS(t *testing.T) {
	ops, err := NewRuntimeOps(EnvironmentOptions{
		HostRoot: t.TempDir(),
		Policy: contract.ExecutionPolicy{
			WriteMode:     contract.WriteModeWriteLimited,
			MaxWriteBytes: 5,
		},
	})
	if err != nil {
		t.Fatalf("NewRuntimeOps failed: %v", err)
	}

	if err := ops.WriteFile(context.Background(), "/task_outputs/ok.txt", "12345"); err != nil {
		t.Fatalf("expected write within limit to succeed: %v", err)
	}

	err = ops.WriteFile(context.Background(), "/task_outputs/big.txt", "123456")
	if err == nil {
		t.Fatalf("expected write beyond limit to fail")
	}
	if !strings.Contains(err.Error(), "write exceeds limit") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestNewRuntimeOpsFullModeDoesNotApplyWriteLimit(t *testing.T) {
	ops, err := NewRuntimeOps(EnvironmentOptions{
		HostRoot: t.TempDir(),
		Policy: contract.ExecutionPolicy{
			WriteMode:     contract.WriteModeFull,
			MaxWriteBytes: 5,
		},
	})
	if err != nil {
		t.Fatalf("NewRuntimeOps failed: %v", err)
	}

	if err := ops.WriteFile(context.Background(), "/task_outputs/full.txt", "1234567890"); err != nil {
		t.Fatalf("expected full mode write not to be limited: %v", err)
	}
}

func TestNewRuntimeOpsRejectsSymlinkEscapeWrite(t *testing.T) {
	hostRoot := t.TempDir()
	outside := t.TempDir()
	if err := os.MkdirAll(filepath.Join(hostRoot, "task_outputs"), 0o755); err != nil {
		t.Fatalf("create task_outputs dir failed: %v", err)
	}
	if err := os.Symlink(outside, filepath.Join(hostRoot, "task_outputs", "escape")); err != nil {
		t.Fatalf("create symlink failed: %v", err)
	}

	ops, err := NewRuntimeOps(EnvironmentOptions{
		HostRoot: hostRoot,
		Policy: contract.ExecutionPolicy{
			WriteMode: contract.WriteModeFull,
		},
	})
	if err != nil {
		t.Fatalf("NewRuntimeOps failed: %v", err)
	}

	target := "/task_outputs/escape/pwned.txt"
	if err := ops.CheckPathOp(context.Background(), contract.PathOpWrite, target); err == nil {
		t.Fatalf("expected symlink escape preflight to fail")
	}
	if err := ops.WriteFile(context.Background(), target, "owned"); err == nil {
		t.Fatalf("expected symlink escape write to fail")
	}
	if _, err := os.Stat(filepath.Join(outside, "pwned.txt")); !os.IsNotExist(err) {
		t.Fatalf("outside path unexpectedly written: err=%v", err)
	}
}

func TestNewRuntimeOpsRejectsNestedSymlinkEscapeWrite(t *testing.T) {
	hostRoot := t.TempDir()
	outside := t.TempDir()
	if err := os.MkdirAll(filepath.Join(hostRoot, "task_outputs"), 0o755); err != nil {
		t.Fatalf("create task_outputs dir failed: %v", err)
	}
	if err := os.Symlink(outside, filepath.Join(hostRoot, "task_outputs", "escape")); err != nil {
		t.Fatalf("create symlink failed: %v", err)
	}

	ops, err := NewRuntimeOps(EnvironmentOptions{
		HostRoot: hostRoot,
		Policy: contract.ExecutionPolicy{
			WriteMode: contract.WriteModeFull,
		},
	})
	if err != nil {
		t.Fatalf("NewRuntimeOps failed: %v", err)
	}

	target := "/task_outputs/escape/subdir/pwned.txt"
	if err := ops.CheckPathOp(context.Background(), contract.PathOpWrite, target); err == nil {
		t.Fatalf("expected nested symlink escape preflight to fail")
	}
	if err := ops.WriteFile(context.Background(), target, "owned"); err == nil {
		t.Fatalf("expected nested symlink escape write to fail")
	}
	if _, err := os.Stat(filepath.Join(outside, "subdir", "pwned.txt")); !os.IsNotExist(err) {
		t.Fatalf("outside path unexpectedly written: err=%v", err)
	}
}

func TestNewRuntimeOpsIsDirPathReturnsEscapeError(t *testing.T) {
	hostRoot := t.TempDir()
	outside := t.TempDir()
	if err := os.MkdirAll(filepath.Join(hostRoot, "task_outputs"), 0o755); err != nil {
		t.Fatalf("create task_outputs dir failed: %v", err)
	}
	if err := os.Symlink(outside, filepath.Join(hostRoot, "task_outputs", "escape")); err != nil {
		t.Fatalf("create symlink failed: %v", err)
	}

	ops, err := NewRuntimeOps(EnvironmentOptions{
		HostRoot: hostRoot,
		Policy: contract.ExecutionPolicy{
			WriteMode: contract.WriteModeFull,
		},
	})
	if err != nil {
		t.Fatalf("NewRuntimeOps failed: %v", err)
	}

	isDir, err := ops.IsDirPath(context.Background(), "/task_outputs/escape/pwned.txt")
	if err == nil {
		t.Fatalf("expected escape-aware IsDirPath to fail, got isDir=%v", isDir)
	}
	if !strings.Contains(err.Error(), "path escape is not allowed") {
		t.Fatalf("unexpected IsDirPath error: %v", err)
	}
}
