package builtin

import (
	"errors"
	"fmt"

	"github.com/khicago/simsh/pkg/contract"
	"github.com/khicago/simsh/pkg/engine"
)

func specCp() engine.CommandSpec {
	return engine.CommandSpec{
		Name:   CommandCp,
		Manual: "cp SRC_ABS DEST_ABS",
		Tips: []string{
			"Copies a file from source to destination. Both paths must be absolute.",
		},
		Run: runCp,
	}
}

func runCp(runtime engine.CommandRuntime, args []string) (string, int) {
	if !runtime.Ops.Policy.AllowWrite() {
		return "cp: write is not supported", contract.ExitCodeUnsupported
	}
	if len(args) != 2 {
		return "cp: expected exactly two arguments: SRC DEST", contract.ExitCodeUsage
	}
	src, err := runtime.Ops.RequireAbsolutePath(args[0])
	if err != nil {
		return fmt.Sprintf("cp: %v", err), contract.ExitCodeUsage
	}
	dest, err := runtime.Ops.RequireAbsolutePath(args[1])
	if err != nil {
		return fmt.Sprintf("cp: %v", err), contract.ExitCodeUsage
	}
	content, err := runtime.Ops.ReadRawContent(runtime.Ctx, src)
	if err != nil {
		return fmt.Sprintf("cp: %v", err), contract.ExitCodeGeneral
	}
	if err := runtime.Ops.WriteFile(runtime.Ctx, dest, content); err != nil {
		if errors.Is(err, contract.ErrUnsupported) {
			return "cp: write is not supported", contract.ExitCodeUnsupported
		}
		return fmt.Sprintf("cp: %v", err), contract.ExitCodeGeneral
	}
	return "", 0
}
