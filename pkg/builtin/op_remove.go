package builtin

import (
	"errors"
	"fmt"
	"strings"

	"github.com/khicago/simsh/pkg/contract"
	"github.com/khicago/simsh/pkg/engine"
)

func specRm() engine.CommandSpec {
	return engine.CommandSpec{
		Name:   CommandRm,
		Manual: "rm [--confirm] [--json] PATH...",
		Tips: []string{
			"Removes files. Does not support directory removal.",
			"Use --confirm or --json when you want explicit success feedback without changing the default silent behavior.",
			"Projection-backed virtual paths are immutable; only mounts that declare mutate support accept removals.",
		},
		StructuredOutput: "path status entries",
		StructuredFlags:  []string{"--confirm", "--json"},
		Examples:         ExamplesFor("rm"),
		DetailedManual:   LoadEmbeddedManual("rm"),
		Run:              runRm,
	}
}

func runRm(runtime engine.CommandRuntime, args []string) (string, int) {
	filteredArgs, confirm, jsonOutput, out, code, ok := extractMutationOutputFlags("rm", args)
	if !ok {
		return out, code
	}
	paths := make([]string, 0, len(filteredArgs))
	for _, arg := range filteredArgs {
		if strings.HasPrefix(arg, "-") {
			return fmt.Sprintf("rm: unsupported flag %s", arg), contract.ExitCodeUsage
		}
		pathValue, err := runtime.Ops.RequireAbsolutePath(arg)
		if err != nil {
			return fmt.Sprintf("rm: %v", err), contract.ExitCodeUsage
		}
		paths = append(paths, pathValue)
	}
	if len(paths) == 0 {
		return "rm: missing operand", contract.ExitCodeUsage
	}
	if !runtime.Ops.Policy.AllowWrite() {
		traceDeniedPaths(runtime, paths...)
		return "rm: write is not supported", contract.ExitCodeUnsupported
	}
	checks := make([]pathCheck, 0, len(paths))
	for _, p := range paths {
		checks = append(checks, pathCheck{path: p, op: contract.PathOpRemove, unsupportedMessage: "rm: not supported"})
	}
	if out, code, ok := preflightPathChecks(runtime, "rm", checks); !ok {
		return out, code
	}
	batch := contract.MutationBatch{Ops: make([]contract.MutationSpec, 0, len(paths))}
	for _, p := range paths {
		batch.Ops = append(batch.Ops, contract.MutationSpec{Kind: contract.MutationRemoveFile, Path: p})
	}
	if runtime.Ops.ApplyMutations != nil {
		if result, err := runtime.Ops.ApplyMutations(runtime.Ctx, batch); err == nil {
			fallback := make(map[string]string, len(paths))
			for _, pathValue := range paths {
				fallback[pathValue] = "removed"
			}
			results, err := mutationStatusesFromRecords(paths, contract.MutationRemoveFile, result.Records, fallback)
			if err != nil {
				return fmt.Sprintf("rm: %v", err), contract.ExitCodeGeneral
			}
			rendered, _, err := renderPathStatusMutation(confirm, jsonOutput, results)
			if err != nil {
				return fmt.Sprintf("rm: %v", err), contract.ExitCodeGeneral
			}
			return rendered, 0
		} else if !contract.AllowsUnsupportedFallback(err) {
			return fmt.Sprintf("rm: %v", err), contract.ExitCodeGeneral
		}
	}
	results := make([]mutationPathStatus, 0, len(paths))
	for _, p := range paths {
		if err := runtime.Ops.RemoveFile(runtime.Ctx, p); err != nil {
			if errors.Is(err, contract.ErrUnsupported) {
				return "rm: not supported", contract.ExitCodeUnsupported
			}
			return fmt.Sprintf("rm: %v", err), contract.ExitCodeGeneral
		}
		results = append(results, mutationPathStatus{Path: p, Status: "removed"})
	}
	rendered, _, err := renderPathStatusMutation(confirm, jsonOutput, results)
	if err != nil {
		return fmt.Sprintf("rm: %v", err), contract.ExitCodeGeneral
	}
	return rendered, 0
}
