package externalmapping

import (
	"context"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

type recordedCommand struct {
	rootDir string
	cmd     RefreshCommand
}

type fakeRunner struct {
	commands []recordedCommand
	failAt   int
}

func (r *fakeRunner) Run(_ context.Context, rootDir string, command RefreshCommand) error {
	r.commands = append(r.commands, recordedCommand{rootDir: rootDir, cmd: command})
	if r.failAt > 0 && len(r.commands) == r.failAt {
		return context.DeadlineExceeded
	}
	return nil
}

func TestDefaultTerminalBenchRefreshPlan(t *testing.T) {
	t.Parallel()

	plan := DefaultTerminalBenchRefreshPlan(".")
	if plan.NativeBaselinePath != DefaultNativeBaselineReportPath {
		t.Fatalf("plan.NativeBaselinePath = %q, want %q", plan.NativeBaselinePath, DefaultNativeBaselineReportPath)
	}
	if plan.PrototypeScopePath != DefaultTerminalBenchPrototypeScopePath {
		t.Fatalf("plan.PrototypeScopePath = %q, want %q", plan.PrototypeScopePath, DefaultTerminalBenchPrototypeScopePath)
	}
	if plan.PrototypeJSONPath != DefaultTerminalBenchArtifactPath {
		t.Fatalf("plan.PrototypeJSONPath = %q, want %q", plan.PrototypeJSONPath, DefaultTerminalBenchArtifactPath)
	}
	if plan.PrototypeMDPath != DefaultTerminalBenchSummaryPath {
		t.Fatalf("plan.PrototypeMDPath = %q, want %q", plan.PrototypeMDPath, DefaultTerminalBenchSummaryPath)
	}
}

func TestRunTerminalBenchRefreshInvokesCommandsInOrder(t *testing.T) {
	t.Parallel()

	plan := DefaultTerminalBenchRefreshPlan(".")
	runner := &fakeRunner{}
	if err := RunTerminalBenchRefresh(context.Background(), plan, runner); err != nil {
		t.Fatalf("RunTerminalBenchRefresh(...) error = %v", err)
	}
	want := []recordedCommand{
		{
			rootDir: ".",
			cmd: RefreshCommand{
				Name: "go",
				Args: []string{"run", "./benchmarks/simsh_native_reference", "-out", DefaultNativeBaselineReportPath},
			},
		},
		{
			rootDir: ".",
			cmd: RefreshCommand{
				Name: "go",
				Args: []string{
					"run", "./benchmarks/terminal_bench_compare",
					"-scope", DefaultTerminalBenchPrototypeScopePath,
					"-report", DefaultNativeBaselineReportPath,
					"-out-json", DefaultTerminalBenchArtifactPath,
					"-out-md", DefaultTerminalBenchSummaryPath,
				},
			},
		},
	}
	if !reflect.DeepEqual(want, runner.commands) {
		t.Fatalf("runner.commands = %#v, want %#v", runner.commands, want)
	}
}

func TestRunTerminalBenchRefreshReturnsRunnerError(t *testing.T) {
	t.Parallel()

	plan := DefaultTerminalBenchRefreshPlan(".")
	runner := &fakeRunner{failAt: 2}
	if err := RunTerminalBenchRefresh(context.Background(), plan, runner); err == nil {
		t.Fatal("RunTerminalBenchRefresh(...) error = nil, want runner error")
	}
}

func TestRunTerminalBenchRefreshRebuildsPrototypePair(t *testing.T) {
	t.Helper()

	tmpDir := t.TempDir()
	plan := DefaultTerminalBenchRefreshPlan(filepath.Join("..", ".."))
	plan.NativeBaselinePath = filepath.Join(tmpDir, "baseline.json")
	plan.PrototypeJSONPath = filepath.Join(tmpDir, "prototype.json")
	plan.PrototypeMDPath = filepath.Join(tmpDir, "prototype.md")

	if err := RunTerminalBenchRefresh(context.Background(), plan, ExecRunner{}); err != nil {
		t.Fatalf("RunTerminalBenchRefresh(...) error = %v", err)
	}

	artifact, err := BuildDefaultTerminalBenchComparison(plan.NativeBaselinePath)
	if err != nil {
		t.Fatalf("BuildDefaultTerminalBenchComparison(%q) error = %v", plan.NativeBaselinePath, err)
	}

	wantJSON, err := MarshalTerminalBenchComparisonJSON(artifact)
	if err != nil {
		t.Fatalf("MarshalTerminalBenchComparisonJSON(...) error = %v", err)
	}
	gotJSON, err := os.ReadFile(plan.PrototypeJSONPath)
	if err != nil {
		t.Fatalf("read generated prototype json: %v", err)
	}
	if !reflect.DeepEqual(wantJSON, gotJSON) {
		t.Fatal("generated prototype json drifted from refresh generator output")
	}

	wantMD := []byte(RenderTerminalBenchComparisonMarkdown(artifact))
	gotMD, err := os.ReadFile(plan.PrototypeMDPath)
	if err != nil {
		t.Fatalf("read generated prototype markdown: %v", err)
	}
	if !reflect.DeepEqual(wantMD, gotMD) {
		t.Fatal("generated prototype markdown drifted from refresh generator output")
	}
}
