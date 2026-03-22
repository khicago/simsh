package builtin

import (
	"errors"
	"fmt"
	"strings"

	"github.com/khicago/simsh/pkg/contract"
	"github.com/khicago/simsh/pkg/engine"
)

func specTouch() engine.CommandSpec {
	return engine.CommandSpec{
		Name:   CommandTouch,
		Manual: "touch [--json] PATH...",
		Tips: []string{
			"Creates empty files if they do not exist.",
			"Use --json when you want explicit created/already_exists feedback without changing the default silent behavior.",
		},
		StructuredOutput: "path status entries",
		StructuredFlags:  []string{"--json"},
		Examples:         ExamplesFor("touch"),
		DetailedManual:   LoadEmbeddedManual("touch"),
		Run:              runTouch,
	}
}

func runTouch(runtime engine.CommandRuntime, args []string) (string, int) {
	filteredArgs, _, jsonOutput, out, code, ok := extractMutationOutputFlags("touch", args)
	if !ok {
		return out, code
	}
	paths := make([]string, 0, len(filteredArgs))
	for _, arg := range filteredArgs {
		if strings.HasPrefix(arg, "-") {
			return fmt.Sprintf("touch: unsupported flag %s", arg), contract.ExitCodeUsage
		}
		pathValue, err := runtime.Ops.RequireAbsolutePath(arg)
		if err != nil {
			return fmt.Sprintf("touch: %v", err), contract.ExitCodeUsage
		}
		paths = append(paths, pathValue)
	}
	if len(paths) == 0 {
		return "touch: missing operand", contract.ExitCodeUsage
	}
	if !runtime.Ops.Policy.AllowWrite() {
		traceDeniedPaths(runtime, paths...)
		return "touch: write is not supported", contract.ExitCodeUnsupported
	}
	checks := make([]pathCheck, 0, len(paths))
	for _, p := range paths {
		checks = append(checks, pathCheck{path: p, op: contract.PathOpWrite, unsupportedMessage: "touch: write is not supported"})
	}
	if out, code, ok := preflightPathChecks(runtime, "touch", checks); !ok {
		return out, code
	}
	results := make([]mutationPathStatus, 0, len(paths))
	for _, p := range paths {
		_, err := runtime.Ops.ReadRawContent(runtime.Ctx, p)
		if err == nil {
			results = append(results, mutationPathStatus{Path: p, Status: "already_exists"})
			continue
		}
		if !isPathMissing(err) {
			return fmt.Sprintf("touch: %v", err), contract.ExitCodeGeneral
		}
		if writeErr := runtime.Ops.WriteFile(runtime.Ctx, p, ""); writeErr != nil {
			if errors.Is(writeErr, contract.ErrUnsupported) {
				return "touch: write is not supported", contract.ExitCodeUnsupported
			}
			return fmt.Sprintf("touch: %v", writeErr), contract.ExitCodeGeneral
		}
		results = append(results, mutationPathStatus{Path: p, Status: "created"})
	}
	rendered, _, err := renderPathStatusMutation(false, jsonOutput, results)
	if err != nil {
		return fmt.Sprintf("touch: %v", err), contract.ExitCodeGeneral
	}
	return rendered, 0
}

func isPathMissing(err error) bool {
	if err == nil {
		return false
	}
	message := strings.ToLower(strings.TrimSpace(err.Error()))
	return strings.Contains(message, "no such file") || strings.Contains(message, "not found")
}
