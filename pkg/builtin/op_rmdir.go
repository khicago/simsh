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
		Manual: "rmdir ABS_DIR...",
		Tips: []string{
			"Removes empty directories only.",
			"Use rm for files; rmdir rejects non-empty directories.",
		},
		Examples:       ExamplesFor("rmdir"),
		DetailedManual: LoadEmbeddedManual("rmdir"),
		Run:            runRmdir,
	}
}

func runRmdir(runtime engine.CommandRuntime, args []string) (string, int) {
	if !runtime.Ops.Policy.AllowWrite() {
		return "rmdir: write is not supported", contract.ExitCodeUnsupported
	}
	if len(args) == 0 {
		return "rmdir: missing operand", contract.ExitCodeUsage
	}
	if runtime.Ops.RemoveDir == nil {
		return "rmdir: not supported", contract.ExitCodeUnsupported
	}

	dirs := make([]string, 0, len(args))
	for _, arg := range args {
		if strings.HasPrefix(arg, "-") {
			return fmt.Sprintf("rmdir: unsupported flag %s", arg), contract.ExitCodeUsage
		}
		pathValue, err := runtime.Ops.RequireAbsolutePath(arg)
		if err != nil {
			return fmt.Sprintf("rmdir: %v", err), contract.ExitCodeUsage
		}
		dirs = append(dirs, pathValue)
	}

	for _, dirPath := range dirs {
		if err := runtime.Ops.CheckPathOp(runtime.Ctx, contract.PathOpRemove, dirPath); err != nil {
			if errors.Is(err, contract.ErrUnsupported) {
				return "rmdir: not supported", contract.ExitCodeUnsupported
			}
			return fmt.Sprintf("rmdir: %v", err), contract.ExitCodeGeneral
		}
		if err := runtime.Ops.RemoveDir(runtime.Ctx, dirPath); err != nil {
			if errors.Is(err, contract.ErrUnsupported) {
				return "rmdir: not supported", contract.ExitCodeUnsupported
			}
			return fmt.Sprintf("rmdir: %v", err), contract.ExitCodeGeneral
		}
	}
	return "", 0
}
