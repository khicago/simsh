package builtin

import (
	"errors"
	"fmt"
	"strings"

	"github.com/khicago/simsh/pkg/contract"
	"github.com/khicago/simsh/pkg/engine"
)

func specMkdir() engine.CommandSpec {
	return engine.CommandSpec{
		Name:   CommandMkdir,
		Manual: "mkdir [--confirm] [--json] [-p] PATH...",
		Tips: []string{
			"Creates directories. -p creates parent directories as needed.",
			"Use --confirm or --json when you want explicit success feedback without changing the default silent behavior.",
			"Mount-backed virtual paths are immutable and cannot be created.",
		},
		StructuredOutput: "path status entries",
		StructuredFlags:  []string{"--confirm", "--json"},
		Examples:         ExamplesFor("mkdir"),
		DetailedManual:   LoadEmbeddedManual("mkdir"),
		Run:              runMkdir,
	}
}

func runMkdir(runtime engine.CommandRuntime, args []string) (string, int) {
	filteredArgs, confirm, jsonOutput, out, code, ok := extractMutationOutputFlags("mkdir", args)
	if !ok {
		return out, code
	}
	paths := make([]string, 0, len(filteredArgs))
	for _, arg := range filteredArgs {
		if arg == "-p" {
			continue
		}
		if strings.HasPrefix(arg, "-") {
			return fmt.Sprintf("mkdir: unsupported flag %s", arg), contract.ExitCodeUsage
		}
		pathValue, err := runtime.Ops.RequireAbsolutePath(arg)
		if err != nil {
			return fmt.Sprintf("mkdir: %v", err), contract.ExitCodeUsage
		}
		paths = append(paths, pathValue)
	}
	if len(paths) == 0 {
		return "mkdir: missing operand", contract.ExitCodeUsage
	}
	if !runtime.Ops.Policy.AllowWrite() {
		traceDeniedPaths(runtime, paths...)
		return "mkdir: write is not supported", contract.ExitCodeUnsupported
	}
	checks := make([]pathCheck, 0, len(paths))
	for _, p := range paths {
		checks = append(checks, pathCheck{path: p, op: contract.PathOpMkdir, unsupportedMessage: "mkdir: not supported"})
	}
	if out, code, ok := preflightPathChecks(runtime, "mkdir", checks); !ok {
		return out, code
	}
	results := make([]mutationPathStatus, 0, len(paths))
	for _, p := range paths {
		exists := false
		if isDir, err := runtime.Ops.IsDirPath(runtime.Ctx, p); err == nil && isDir {
			exists = true
		}
		if err := runtime.Ops.MakeDir(runtime.Ctx, p); err != nil {
			if errors.Is(err, contract.ErrUnsupported) {
				return "mkdir: not supported", contract.ExitCodeUnsupported
			}
			return fmt.Sprintf("mkdir: %v", err), contract.ExitCodeGeneral
		}
		status := "created"
		if exists {
			status = "exists"
		}
		results = append(results, mutationPathStatus{Path: p, Status: status})
	}
	rendered, _, err := renderPathStatusMutation(confirm, jsonOutput, results)
	if err != nil {
		return fmt.Sprintf("mkdir: %v", err), contract.ExitCodeGeneral
	}
	return rendered, 0
}
