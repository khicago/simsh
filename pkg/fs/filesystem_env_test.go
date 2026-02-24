package fs

import (
	"context"
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
