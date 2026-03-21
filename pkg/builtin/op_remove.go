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
		Manual: "rm ABS_PATH...",
		Tips: []string{
			"Removes files. Does not support directory removal.",
			"Mount-backed virtual paths are immutable and cannot be removed.",
		},
		Examples:       ExamplesFor("rm"),
		DetailedManual: LoadEmbeddedManual("rm"),
		Run:            runRm,
	}
}

func runRm(runtime engine.CommandRuntime, args []string) (string, int) {
	if !runtime.Ops.Policy.AllowWrite() {
		return "rm: write is not supported", contract.ExitCodeUnsupported
	}
	paths := make([]string, 0, len(args))
	for _, arg := range args {
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
	checks := make([]pathCheck, 0, len(paths))
	for _, p := range paths {
		checks = append(checks, pathCheck{path: p, op: contract.PathOpRemove, unsupportedMessage: "rm: not supported"})
	}
	if out, code, ok := preflightPathChecks(runtime, "rm", checks); !ok {
		return out, code
	}
	for _, p := range paths {
		if err := runtime.Ops.RemoveFile(runtime.Ctx, p); err != nil {
			if errors.Is(err, contract.ErrUnsupported) {
				return "rm: not supported", contract.ExitCodeUnsupported
			}
			return fmt.Sprintf("rm: %v", err), contract.ExitCodeGeneral
		}
	}
	return "", 0
}
