package service

import (
	"context"
	"errors"
	"testing"

	"github.com/khicago/simsh/pkg/builtin"
	"github.com/khicago/simsh/pkg/contract"
	engpkg "github.com/khicago/simsh/pkg/engine"
	simfs "github.com/khicago/simsh/pkg/fs"
)

func TestExecutorServiceExecuteInitializationErrors(t *testing.T) {
	eng := newTestExecutorEngine()

	cases := []struct {
		name    string
		service *ExecutorService
		wantErr string
	}{
		{
			name:    "nil service",
			service: nil,
			wantErr: "executor service is not initialized",
		},
		{
			name:    "nil engine",
			service: &ExecutorService{},
			wantErr: "executor service is not initialized",
		},
		{
			name:    "missing ops factory",
			service: &ExecutorService{Engine: eng},
			wantErr: "ops factory is required",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := tc.service.Execute(context.Background(), ExecuteRequest{Command: "echo hello"})
			if err == nil || err.Error() != tc.wantErr {
				t.Fatalf("Execute(...) error = %v, want %q", err, tc.wantErr)
			}
		})
	}
}

func TestExecutorServiceExecuteCommandRequiredFastPath(t *testing.T) {
	called := false
	svc := &ExecutorService{
		Engine: newTestExecutorEngine(),
		OpsFactory: func(ctx context.Context, req ExecuteRequest) (contract.Ops, error) {
			called = true
			return contract.Ops{}, errors.New("ops factory should not run for empty command")
		},
	}

	resp, err := svc.Execute(context.Background(), ExecuteRequest{Command: " \n\t "})
	if err != nil {
		t.Fatalf("Execute(...) error = %v", err)
	}
	if called {
		t.Fatal("Execute(...) unexpectedly called OpsFactory for empty command")
	}
	if resp.ExitCode != contract.ExitCodeUsage {
		t.Fatalf("Execute(...).ExitCode = %d, want %d", resp.ExitCode, contract.ExitCodeUsage)
	}
	if resp.ExecutionID == "" || resp.StartedAt.IsZero() || resp.FinishedAt.IsZero() {
		t.Fatalf("Execute(...) blank-command result missing structured fields: %+v", resp.ExecutionResult)
	}
	if resp.Output != "execute: command is required" {
		t.Fatalf("Execute(...).Output = %q, want %q", resp.Output, "execute: command is required")
	}
	if resp.Output != resp.ExecutionResult.FlattenOutput() {
		t.Fatalf("Execute(...).Output = %q, want flattened output %q", resp.Output, resp.ExecutionResult.FlattenOutput())
	}
}

func TestExecutorServiceExecutePropagatesOpsFactoryError(t *testing.T) {
	wantErr := errors.New("ops factory boom")
	svc := &ExecutorService{
		Engine: newTestExecutorEngine(),
		OpsFactory: func(ctx context.Context, req ExecuteRequest) (contract.Ops, error) {
			return contract.Ops{}, wantErr
		},
	}

	_, err := svc.Execute(context.Background(), ExecuteRequest{Command: "echo hello"})
	if !errors.Is(err, wantErr) {
		t.Fatalf("Execute(...) error = %v, want %v", err, wantErr)
	}
}

func TestExecutorServiceExecuteSuccess(t *testing.T) {
	hostRoot := t.TempDir()
	svc := &ExecutorService{
		Engine: newTestExecutorEngine(),
		OpsFactory: func(ctx context.Context, req ExecuteRequest) (contract.Ops, error) {
			return simfs.NewRuntimeOps(simfs.EnvironmentOptions{
				HostRoot: hostRoot,
				Profile:  contract.ProfileCoreStrict,
				Policy:   contract.DefaultPolicy(),
			})
		},
	}

	resp, err := svc.Execute(context.Background(), ExecuteRequest{Command: "echo hello"})
	if err != nil {
		t.Fatalf("Execute(...) error = %v", err)
	}
	if resp.ExitCode != 0 {
		t.Fatalf("Execute(...).ExitCode = %d, want 0", resp.ExitCode)
	}
	if resp.Stdout != "hello" || resp.Stderr != "" {
		t.Fatalf("Execute(...) stdout/stderr = (%q, %q), want (%q, %q)", resp.Stdout, resp.Stderr, "hello", "")
	}
	if resp.Output != "hello" {
		t.Fatalf("Execute(...).Output = %q, want %q", resp.Output, "hello")
	}
	if resp.Output != resp.ExecutionResult.FlattenOutput() {
		t.Fatalf("Execute(...).Output = %q, want flattened output %q", resp.Output, resp.ExecutionResult.FlattenOutput())
	}
}

func newTestExecutorEngine() *engpkg.Engine {
	registry := engpkg.NewRegistry()
	builtin.RegisterDefaults(registry)
	return engpkg.New(registry)
}
