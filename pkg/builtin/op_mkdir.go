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
		Manual: "mkdir [-p] PATH...",
		Tips: []string{
			"Creates directories. -p creates parent directories as needed.",
			"Mount-backed virtual paths are immutable and cannot be created.",
		},
		Examples:       ExamplesFor("mkdir"),
		DetailedManual: LoadEmbeddedManual("mkdir"),
		Run:            runMkdir,
	}
}

func runMkdir(runtime engine.CommandRuntime, args []string) (string, int) {
	if !runtime.Ops.Policy.AllowWrite() {
		return "mkdir: write is not supported", contract.ExitCodeUnsupported
	}
	paths := make([]string, 0, len(args))
	for _, arg := range args {
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
	checks := make([]pathCheck, 0, len(paths))
	for _, p := range paths {
		checks = append(checks, pathCheck{path: p, op: contract.PathOpMkdir, unsupportedMessage: "mkdir: not supported"})
	}
	if out, code, ok := preflightPathChecks(runtime, "mkdir", checks); !ok {
		return out, code
	}
	for _, p := range paths {
		if err := runtime.Ops.MakeDir(runtime.Ctx, p); err != nil {
			if errors.Is(err, contract.ErrUnsupported) {
				return "mkdir: not supported", contract.ExitCodeUnsupported
			}
			return fmt.Sprintf("mkdir: %v", err), contract.ExitCodeGeneral
		}
	}
	return "", 0
}
