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
		Manual: "touch ABS_PATH...",
		Tips: []string{
			"Creates empty files if they do not exist.",
		},
		Examples:       ExamplesFor("touch"),
		DetailedManual: LoadEmbeddedManual("touch"),
		Run:            runTouch,
	}
}

func runTouch(runtime engine.CommandRuntime, args []string) (string, int) {
	if !runtime.Ops.Policy.AllowWrite() {
		return "touch: write is not supported", contract.ExitCodeUnsupported
	}
	paths := make([]string, 0, len(args))
	for _, arg := range args {
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
	missingPaths := make([]string, 0, len(paths))
	for _, p := range paths {
		_, err := runtime.Ops.ReadRawContent(runtime.Ctx, p)
		if err == nil {
			continue
		}
		missingPaths = append(missingPaths, p)
	}
	checks := make([]pathCheck, 0, len(missingPaths))
	for _, p := range missingPaths {
		checks = append(checks, pathCheck{path: p, op: contract.PathOpWrite, unsupportedMessage: "touch: write is not supported"})
	}
	if out, code, ok := preflightPathChecks(runtime, "touch", checks); !ok {
		return out, code
	}
	for _, p := range missingPaths {
		if writeErr := runtime.Ops.WriteFile(runtime.Ctx, p, ""); writeErr != nil {
			if errors.Is(writeErr, contract.ErrUnsupported) {
				return "touch: write is not supported", contract.ExitCodeUnsupported
			}
			return fmt.Sprintf("touch: %v", writeErr), contract.ExitCodeGeneral
		}
	}
	return "", 0
}
