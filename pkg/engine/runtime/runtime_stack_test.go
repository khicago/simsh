package runtime

import (
	"context"
	"os"
	"path/filepath"
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
