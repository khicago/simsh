package engine_test

import (
	"context"
	"testing"

	"github.com/khicago/simsh/pkg/builtin"
	"github.com/khicago/simsh/pkg/contract"
	"github.com/khicago/simsh/pkg/engine"
)

func TestEngineExecuteResultStructuredFields(t *testing.T) {
	registry := engine.NewRegistry()
	builtin.RegisterDefaults(registry)
	eng := engine.New(registry)
	ops := contract.OpsFromFilesystem(newTestFS())
	ops.Profile = contract.ProfileCoreStrict
	ops.Policy = contract.DefaultPolicy()

	result := eng.ExecuteResult(context.Background(), "echo hello", ops)
	if result.ExecutionID == "" {
		t.Fatalf("expected execution_id, got %+v", result)
	}
	if result.ExitCode != 0 {
		t.Fatalf("exit_code = %d, want 0", result.ExitCode)
	}
	if result.Stdout != "hello" || result.Stderr != "" {
		t.Fatalf("unexpected stdout/stderr: %+v", result)
	}
	if result.Trace.CommandLine != "echo hello" {
		t.Fatalf("unexpected trace: %+v", result.Trace)
	}
	if result.Trace.Command != "echo" || len(result.Trace.Pipeline) != 1 {
		t.Fatalf("unexpected command shape: %+v", result.Trace)
	}
	if result.Trace.EffectiveProfile != contract.ProfileCoreStrict {
		t.Fatalf("unexpected effective profile: %+v", result.Trace)
	}
	if result.Trace.EffectivePolicy.WriteMode != contract.WriteModeReadOnly {
		t.Fatalf("unexpected effective policy: %+v", result.Trace.EffectivePolicy)
	}
}

func TestEngineExecuteResultTracePaths(t *testing.T) {
	registry := engine.NewRegistry()
	builtin.RegisterDefaults(registry)
	eng := engine.New(registry)
	ops := contract.OpsFromFilesystem(newTestFS())
	ops.Profile = contract.ProfileCoreStrict
	ops.Policy = contract.ExecutionPolicy{
		WriteMode:        contract.WriteModeFull,
		MaxPipelineDepth: 16,
		MaxOutputBytes:   4 << 20,
		Timeout:          contract.DefaultPolicy().Timeout,
	}

	result := eng.ExecuteResult(context.Background(), "echo hello > /workspace/out.txt; cat /workspace/out.txt", ops)
	if result.ExitCode != 0 {
		t.Fatalf("unexpected exit_code=%d stdout=%q", result.ExitCode, result.Stdout)
	}
	if !containsTracePath(result.Trace.RequestedPaths, "/workspace/out.txt") {
		t.Fatalf("expected requested path in trace: %+v", result.Trace)
	}
	if !containsTracePath(result.Trace.WrittenPaths, "/workspace/out.txt") {
		t.Fatalf("expected written path in trace: %+v", result.Trace)
	}
	if !containsTracePath(result.Trace.ReadPaths, "/workspace/out.txt") {
		t.Fatalf("expected read path in trace: %+v", result.Trace)
	}
	if result.Trace.BytesWritten == 0 || result.Trace.BytesRead == 0 {
		t.Fatalf("expected resource summary bytes in trace: %+v", result.Trace)
	}
}

func TestEngineExecuteResultTraceEditWritesFinalFileBytes(t *testing.T) {
	registry := engine.NewRegistry()
	builtin.RegisterDefaults(registry)
	eng := engine.New(registry)
	ops := contract.OpsFromFilesystem(newTestFS())
	ops.Profile = contract.ProfileBashPlus
	ops.Policy = contract.ExecutionPolicy{
		WriteMode:        contract.WriteModeFull,
		MaxPipelineDepth: 16,
		MaxOutputBytes:   4 << 20,
		Timeout:          contract.DefaultPolicy().Timeout,
	}

	result := eng.ExecuteResult(context.Background(), "sed -i 's/hello/hi/' /workspace/readme.md", ops)
	if result.ExitCode != 0 {
		t.Fatalf("unexpected exit_code=%d stdout=%q", result.ExitCode, result.Stdout)
	}
	if !containsTracePath(result.Trace.EditedPaths, "/workspace/readme.md") {
		t.Fatalf("expected edited path in trace: %+v", result.Trace)
	}
	if result.Trace.BytesWritten != len("hi\nworld\n") {
		t.Fatalf("bytes_written=%d, want %d trace=%+v", result.Trace.BytesWritten, len("hi\nworld\n"), result.Trace)
	}
}

func TestEngineExecuteResultTraceDeniedAndOutputLimit(t *testing.T) {
	registry := engine.NewRegistry()
	builtin.RegisterDefaults(registry)
	eng := engine.New(registry)
	ops := contract.OpsFromFilesystem(newTestFS())
	ops.Profile = contract.ProfileCoreStrict
	ops.Policy = contract.ExecutionPolicy{WriteMode: contract.WriteModeReadOnly, MaxOutputBytes: 1024}

	denied := eng.ExecuteResult(context.Background(), "echo hello > /workspace/out.txt", ops)
	if !containsTracePath(denied.Trace.DeniedPaths, "/workspace/out.txt") {
		t.Fatalf("expected denied path in trace: %+v", denied.Trace)
	}

	ops.Policy = contract.ExecutionPolicy{WriteMode: contract.WriteModeReadOnly, MaxOutputBytes: 4}
	truncated := eng.ExecuteResult(context.Background(), "echo hello", ops)
	if !truncated.Trace.OutputTruncated {
		t.Fatalf("expected output_truncated trace flag: %+v", truncated.Trace)
	}
}

func TestEngineExecuteResultTraceMutationDenials(t *testing.T) {
	registry := engine.NewRegistry()
	builtin.RegisterDefaults(registry)
	eng := engine.New(registry)
	ops := contract.OpsFromFilesystem(newTestFS())
	ops.Profile = contract.ProfileBashPlus
	ops.Policy = contract.ExecutionPolicy{
		WriteMode:        contract.WriteModeFull,
		MaxPipelineDepth: 16,
		MaxOutputBytes:   4 << 20,
		Timeout:          contract.DefaultPolicy().Timeout,
	}

	deniedEdit := eng.ExecuteResult(context.Background(), "sed -i 's/ls/echo/' /sys/bin/ls", ops)
	if !containsTracePath(deniedEdit.Trace.DeniedPaths, "/sys/bin/ls") {
		t.Fatalf("expected edit denial path in trace: %+v", deniedEdit.Trace)
	}
	deniedMount := eng.ExecuteResult(context.Background(), "touch /sys/bin/new.txt", ops)
	if !containsTracePath(deniedMount.Trace.DeniedPaths, "/sys/bin/new.txt") {
		t.Fatalf("expected mounted path denial in trace: %+v", deniedMount.Trace)
	}
}

func containsTracePath(paths []string, target string) bool {
	for _, pathValue := range paths {
		if pathValue == target {
			return true
		}
	}
	return false
}
