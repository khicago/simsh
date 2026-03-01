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
	if result.Trace.EffectiveProfile != contract.ProfileCoreStrict {
		t.Fatalf("unexpected effective profile: %+v", result.Trace)
	}
	if result.Trace.EffectivePolicy.WriteMode != contract.WriteModeReadOnly {
		t.Fatalf("unexpected effective policy: %+v", result.Trace.EffectivePolicy)
	}
}
