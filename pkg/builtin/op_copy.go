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
		Manual: "cp SRC_PATH DEST_PATH",
		Tips: []string{
			"Copies a file from source to destination.",
			"Mount-backed virtual paths are immutable and not valid copy operands.",
		},
		Examples:       ExamplesFor("cp"),
		DetailedManual: LoadEmbeddedManual("cp"),
		Run:            runCp,
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
	if out, code, ok := preflightPathChecks(runtime, "cp", []pathCheck{
		{path: src, op: contract.PathOpRead, unsupportedMessage: "cp: source path is not supported"},
		{path: dest, op: contract.PathOpWrite, unsupportedMessage: "cp: write is not supported"},
	}); !ok {
		return out, code
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
