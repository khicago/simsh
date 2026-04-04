package externalmapping

import (
	"context"
	"fmt"
	"os/exec"
	"path/filepath"
)

type RefreshCommand struct {
	Name string
	Args []string
}

type RefreshPlan struct {
	RootDir            string
	NativeBaselinePath string
	PrototypeScopePath string
	PrototypeJSONPath  string
	PrototypeMDPath    string
	GoBinary           string
}

type CommandRunner interface {
	Run(context.Context, string, RefreshCommand) error
}

type ExecRunner struct{}

func (ExecRunner) Run(ctx context.Context, rootDir string, command RefreshCommand) error {
	cmd := exec.CommandContext(ctx, command.Name, command.Args...)
	cmd.Dir = rootDir
	cmd.Stdout = nil
	cmd.Stderr = nil
	return cmd.Run()
}

func DefaultTerminalBenchRefreshPlan(rootDir string) RefreshPlan {
	if rootDir == "" {
		rootDir = "."
	}
	return RefreshPlan{
		RootDir:            rootDir,
		NativeBaselinePath: DefaultNativeBaselineReportPath,
		PrototypeScopePath: DefaultTerminalBenchPrototypeScopePath,
		PrototypeJSONPath:  DefaultTerminalBenchArtifactPath,
		PrototypeMDPath:    DefaultTerminalBenchSummaryPath,
		GoBinary:           "go",
	}
}

func (plan RefreshPlan) Commands() []RefreshCommand {
	goBinary := plan.GoBinary
	if goBinary == "" {
		goBinary = "go"
	}
	return []RefreshCommand{
		{
			Name: goBinary,
			Args: []string{
				"run", "./benchmarks/simsh_native_reference",
				"-out", plan.NativeBaselinePath,
			},
		},
		{
			Name: goBinary,
			Args: []string{
				"run", "./benchmarks/terminal_bench_compare",
				"-scope", plan.PrototypeScopePath,
				"-report", plan.NativeBaselinePath,
				"-out-json", plan.PrototypeJSONPath,
				"-out-md", plan.PrototypeMDPath,
			},
		},
	}
}

func RunTerminalBenchRefresh(ctx context.Context, plan RefreshPlan, runner CommandRunner) error {
	if runner == nil {
		runner = ExecRunner{}
	}
	if plan.RootDir == "" {
		plan.RootDir = "."
	}
	for _, command := range plan.Commands() {
		if err := runner.Run(ctx, plan.RootDir, command); err != nil {
			return fmt.Errorf("run %s %v: %w", command.Name, command.Args, err)
		}
	}
	return nil
}

func (plan RefreshPlan) NativeBaselineAbsPath() string {
	return filepath.Join(plan.RootDir, plan.NativeBaselinePath)
}

func (plan RefreshPlan) PrototypeJSONAbsPath() string {
	return filepath.Join(plan.RootDir, plan.PrototypeJSONPath)
}

func (plan RefreshPlan) PrototypeMDAbsPath() string {
	return filepath.Join(plan.RootDir, plan.PrototypeMDPath)
}
