package builtin

import (
	"errors"
	"fmt"
	"strings"

	"github.com/khicago/simsh/pkg/contract"
	"github.com/khicago/simsh/pkg/engine"
)

func specRmdir() engine.CommandSpec {
	return engine.CommandSpec{
		Name:   CommandRmdir,
		Manual: "rmdir [--confirm] [--json] PATH...",
		Tips: []string{
			"Removes empty directories only.",
			"Use --confirm or --json when you want explicit success feedback without changing the default silent behavior.",
			"Use rm for files; rmdir rejects non-empty directories.",
		},
		StructuredOutput: "path status entries",
		StructuredFlags:  []string{"--confirm", "--json"},
		Examples:         ExamplesFor("rmdir"),
		DetailedManual:   LoadEmbeddedManual("rmdir"),
		Run:              runRmdir,
	}
}

func runRmdir(runtime engine.CommandRuntime, args []string) (string, int) {
	filteredArgs, confirm, jsonOutput, out, code, ok := extractMutationOutputFlags("rmdir", args)
	if !ok {
		return out, code
	}
	if len(filteredArgs) == 0 {
		return "rmdir: missing operand", contract.ExitCodeUsage
	}
	if runtime.Ops.RemoveDir == nil {
		return "rmdir: not supported", contract.ExitCodeUnsupported
	}

	dirs := make([]string, 0, len(filteredArgs))
	for _, arg := range filteredArgs {
		if strings.HasPrefix(arg, "-") {
			return fmt.Sprintf("rmdir: unsupported flag %s", arg), contract.ExitCodeUsage
		}
		pathValue, err := runtime.Ops.RequireAbsolutePath(arg)
		if err != nil {
			return fmt.Sprintf("rmdir: %v", err), contract.ExitCodeUsage
		}
		dirs = append(dirs, pathValue)
	}
	if !runtime.Ops.Policy.AllowWrite() {
		traceDeniedPaths(runtime, dirs...)
		return "rmdir: write is not supported", contract.ExitCodeUnsupported
	}

	checks := make([]pathCheck, 0, len(dirs))
	for _, dirPath := range dirs {
		checks = append(checks, pathCheck{path: dirPath, op: contract.PathOpRemove, unsupportedMessage: "rmdir: not supported"})
	}
	if out, code, ok := preflightPathChecks(runtime, "rmdir", checks); !ok {
		return out, code
	}
	results := make([]mutationPathStatus, 0, len(dirs))
	for _, dirPath := range dirs {
		if err := runtime.Ops.RemoveDir(runtime.Ctx, dirPath); err != nil {
			if errors.Is(err, contract.ErrUnsupported) {
				return "rmdir: not supported", contract.ExitCodeUnsupported
			}
			return fmt.Sprintf("rmdir: %v", err), contract.ExitCodeGeneral
		}
		results = append(results, mutationPathStatus{Path: dirPath, Status: "removed"})
	}
	rendered, _, err := renderPathStatusMutation(confirm, jsonOutput, results)
	if err != nil {
		return fmt.Sprintf("rmdir: %v", err), contract.ExitCodeGeneral
	}
	return rendered, 0
}
