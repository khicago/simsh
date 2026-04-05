package builtin

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/khicago/simsh/pkg/contract"
	"github.com/khicago/simsh/pkg/engine"
)

type mutationTestOps struct {
	contract.Ops
	content map[string]string
}

func newMutationTestOps() *mutationTestOps {
	ops := &mutationTestOps{
		content: map[string]string{
			"/sys/bin/ls":    "old",
			"/workspace/a":   "",
			"/workspace/tmp": "",
		},
	}
	ops.RootDir = "/"
	ops.RequireAbsolutePath = func(raw string) (string, error) { return raw, nil }
	ops.ReadRawContent = func(ctx context.Context, pathValue string) (string, error) {
		if raw, ok := ops.content[pathValue]; ok {
			return raw, nil
		}
		return "", fmt.Errorf("missing: %s", pathValue)
	}
	ops.WriteFile = func(ctx context.Context, pathValue string, content string) error {
		ops.content[pathValue] = content
		return nil
	}
	ops.RemoveFile = func(ctx context.Context, pathValue string) error {
		delete(ops.content, pathValue)
		return nil
	}
	ops.MakeDir = func(ctx context.Context, dirPath string) error { return nil }
	ops.ApplyMutations = func(ctx context.Context, req contract.MutationBatch) (contract.MutationResult, error) {
		return contract.MutationResult{}, errors.New("not implemented")
	}
	ops.Policy = contract.ExecutionPolicy{
		WriteMode:        contract.WriteModeFull,
		MaxPipelineDepth: 16,
		MaxOutputBytes:   4 << 20,
		Timeout:          contract.DefaultPolicy().Timeout,
	}
	ops.CheckPathOp = func(context.Context, contract.PathOp, string) error { return nil }
	return ops
}

func newRuntime() engine.CommandRuntime {
	ops := newMutationTestOps()
	return engine.CommandRuntime{
		Ctx: context.Background(),
		Ops: ops.Ops,
	}
}

func TestRunMvUsesApplyMutations(t *testing.T) {
	rt := newRuntime()
	applied := false
	rt.Ops.ApplyMutations = func(ctx context.Context, req contract.MutationBatch) (contract.MutationResult, error) {
		if len(req.Ops) != 2 {
			t.Fatalf("expected 2 ops, got %d", len(req.Ops))
		}
		if req.Ops[0].Kind != contract.MutationWriteFile || req.Ops[1].Kind != contract.MutationRemoveFile {
			t.Fatalf("unexpected mutation kinds %v", req.Ops)
		}
		applied = true
		return contract.MutationResult{}, nil
	}
	out, code := runMv(rt, []string{"/sys/bin/ls", "/workspace/copied"})
	if code != 0 {
		t.Fatalf("runMv failed: %q", out)
	}
	if !applied {
		t.Fatalf("expected ApplyMutations to run")
	}
}

func TestRunMvFallsBackWhenApplyMutationsUnsupported(t *testing.T) {
	rt := newRuntime()
	writeCalled := false
	rt.Ops.WriteFile = func(ctx context.Context, pathValue string, content string) error {
		writeCalled = true
		return nil
	}
	removeCalled := false
	rt.Ops.RemoveFile = func(ctx context.Context, pathValue string) error {
		removeCalled = true
		return nil
	}
	rt.Ops.ApplyMutations = func(ctx context.Context, req contract.MutationBatch) (contract.MutationResult, error) {
		return contract.MutationResult{}, contract.ErrUnsupported
	}
	out, code := runMv(rt, []string{"/sys/bin/ls", "/workspace/copied"})
	if code != 0 {
		t.Fatalf("runMv fallback failed: %q", out)
	}
	if !writeCalled || !removeCalled {
		t.Fatalf("expected fallback ops, got write=%v remove=%v", writeCalled, removeCalled)
	}
}

func TestRunRmUsesMutationBatch(t *testing.T) {
	rt := newRuntime()
	called := false
	rt.Ops.ApplyMutations = func(ctx context.Context, req contract.MutationBatch) (contract.MutationResult, error) {
		if len(req.Ops) != 1 {
			t.Fatalf("expected 1 op, got %d", len(req.Ops))
		}
		if req.Ops[0].Kind != contract.MutationRemoveFile {
			t.Fatalf("unexpected kind %s", req.Ops[0].Kind)
		}
		called = true
		return contract.MutationResult{}, nil
	}
	out, code := runRm(rt, []string{"/workspace/a"})
	if code != 0 {
		t.Fatalf("runRm failed: %q", out)
	}
	if !called {
		t.Fatalf("expected ApplyMutations to run")
	}
}

func TestRunMkdirUsesMutationBatch(t *testing.T) {
	rt := newRuntime()
	called := false
	rt.Ops.ApplyMutations = func(ctx context.Context, req contract.MutationBatch) (contract.MutationResult, error) {
		if len(req.Ops) != 1 {
			t.Fatalf("expected 1 op, got %d", len(req.Ops))
		}
		if req.Ops[0].Kind != contract.MutationMakeDir {
			t.Fatalf("unexpected kind %s", req.Ops[0].Kind)
		}
		called = true
		return contract.MutationResult{}, nil
	}
	out, code := runMkdir(rt, []string{" /workspace/new"})
	if code != 0 {
		t.Fatalf("runMkdir failed: %q", out)
	}
	if !called {
		t.Fatalf("expected ApplyMutations to run")
	}
}
