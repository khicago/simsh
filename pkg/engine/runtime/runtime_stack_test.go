package runtime

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/khicago/simsh/pkg/contract"
)

func TestNewRuntimeAndExecute(t *testing.T) {
	tmp := t.TempDir()
	runtime, err := New(Options{
		HostRoot: tmp,
		Profile:  contract.ProfileCoreStrict,
		Policy:   contract.DefaultPolicy(),
	})
	if err != nil {
		t.Fatalf("new runtime failed: %v", err)
	}

	out, code := runtime.Execute(context.Background(), "echo hello")
	if code != 0 {
		t.Fatalf("exit code = %d, want 0", code)
	}
	if out != "hello" {
		t.Fatalf("output = %q, want %q", out, "hello")
	}

	for _, dir := range []string{"task_outputs", "temp_work", "knowledge_base"} {
		pathValue := filepath.Join(tmp, dir)
		info, err := os.Stat(pathValue)
		if err != nil {
			t.Fatalf("stat %s failed: %v", pathValue, err)
		}
		if !info.IsDir() {
			t.Fatalf("%s is not a directory", pathValue)
		}
	}
}

func TestRuntimeRespectsPolicy(t *testing.T) {
	runtime, err := New(Options{
		HostRoot: t.TempDir(),
		Profile:  contract.ProfileCoreStrict,
		Policy: contract.ExecutionPolicy{
			WriteMode: contract.WriteModeReadOnly,
		},
	})
	if err != nil {
		t.Fatalf("new runtime failed: %v", err)
	}

	_, code := runtime.Execute(context.Background(), "echo x > /task_outputs/out.txt")
	if code != contract.ExitCodeUnsupported {
		t.Fatalf("exit code = %d, want %d", code, contract.ExitCodeUnsupported)
	}
}

func TestRuntimeOpsUsesPreparedPathMetadata(t *testing.T) {
	runtime, err := New(Options{
		HostRoot: t.TempDir(),
		Profile:  contract.ProfileCoreStrict,
		Policy:   contract.DefaultPolicy(),
	})
	if err != nil {
		t.Fatalf("new runtime failed: %v", err)
	}

	meta, err := runtime.Ops().DescribePath(context.Background(), "/sys")
	if err != nil {
		t.Fatalf("describe /sys failed: %v", err)
	}
	if !meta.Exists || !meta.IsDir {
		t.Fatalf("expected /sys to be a virtual dir, got %+v", meta)
	}
	if meta.Access != contract.PathAccessReadOnly {
		t.Fatalf("expected /sys access=ro, got %q", meta.Access)
	}
}

func TestRuntimeWorkingDirPersistsAcrossExecuteCalls(t *testing.T) {
	runtime, err := New(Options{
		HostRoot: t.TempDir(),
		Profile:  contract.ProfileBashPlus,
		Policy: contract.ExecutionPolicy{
			WriteMode:        contract.WriteModeFull,
			MaxWriteBytes:    1 << 20,
			MaxPipelineDepth: 16,
			MaxOutputBytes:   4 << 20,
			Timeout:          contract.DefaultPolicy().Timeout,
		},
	})
	if err != nil {
		t.Fatalf("new runtime failed: %v", err)
	}

	if out, code := runtime.Execute(context.Background(), "mkdir -p /task_outputs/project"); code != 0 {
		t.Fatalf("mkdir failed: code=%d out=%q", code, out)
	}
	if out, code := runtime.Execute(context.Background(), "cd /task_outputs/project"); code != 0 {
		t.Fatalf("cd failed: code=%d out=%q", code, out)
	}
	if out, code := runtime.Execute(context.Background(), "pwd"); code != 0 || strings.TrimSpace(out) != "/task_outputs/project" {
		t.Fatalf("pwd after cd failed: code=%d out=%q", code, out)
	}
	if out, code := runtime.Execute(context.Background(), "echo hello > note.txt"); code != 0 {
		t.Fatalf("relative write failed: code=%d out=%q", code, out)
	}
	if out, code := runtime.Execute(context.Background(), "cat note.txt"); code != 0 || strings.TrimSpace(out) != "hello" {
		t.Fatalf("relative read failed: code=%d out=%q", code, out)
	}
}

func TestRuntimeExecuteResultStructuredFields(t *testing.T) {
	runtime, err := New(Options{
		HostRoot: t.TempDir(),
		Profile:  contract.ProfileCoreStrict,
		Policy:   contract.DefaultPolicy(),
	})
	if err != nil {
		t.Fatalf("new runtime failed: %v", err)
	}

	result := runtime.ExecuteResult(context.Background(), "echo hello")
	if result.ExecutionID == "" {
		t.Fatalf("expected execution_id, got %+v", result)
	}
	if result.ExitCode != 0 {
		t.Fatalf("exit_code = %d, want 0", result.ExitCode)
	}
	if result.Stdout != "hello" || result.Stderr != "" {
		t.Fatalf("unexpected stdout/stderr: %+v", result)
	}
	if result.StartedAt.IsZero() || result.FinishedAt.IsZero() {
		t.Fatalf("expected timestamps, got %+v", result)
	}
	if result.Trace.CommandLine != "echo hello" {
		t.Fatalf("unexpected trace command_line: %+v", result.Trace)
	}
	if result.Trace.Command != "echo" || len(result.Trace.Pipeline) != 1 {
		t.Fatalf("unexpected trace shape: %+v", result.Trace)
	}
	if result.Trace.EffectiveProfile != contract.ProfileCoreStrict {
		t.Fatalf("unexpected effective profile: %+v", result.Trace)
	}
}
