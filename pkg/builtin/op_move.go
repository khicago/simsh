package builtin

import (
	"errors"
	"fmt"

	"github.com/khicago/simsh/pkg/contract"
	"github.com/khicago/simsh/pkg/engine"
)

func specMv() engine.CommandSpec {
	return engine.CommandSpec{
		Name:   CommandMv,
		Manual: "mv SRC_PATH DEST_PATH",
		Tips: []string{
			"Moves a file from source to destination.",
			"Mount-backed virtual paths are immutable and cannot be moved.",
		},
		Examples:       ExamplesFor("mv"),
		DetailedManual: LoadEmbeddedManual("mv"),
		Run:            runMv,
	}
}

func runMv(runtime engine.CommandRuntime, args []string) (string, int) {
	if !runtime.Ops.Policy.AllowWrite() {
		return "mv: write is not supported", contract.ExitCodeUnsupported
	}
	if len(args) != 2 {
		return "mv: expected exactly two arguments: SRC DEST", contract.ExitCodeUsage
	}
	src, err := runtime.Ops.RequireAbsolutePath(args[0])
	if err != nil {
		return fmt.Sprintf("mv: %v", err), contract.ExitCodeUsage
	}
	dest, err := runtime.Ops.RequireAbsolutePath(args[1])
	if err != nil {
		return fmt.Sprintf("mv: %v", err), contract.ExitCodeUsage
	}
	if out, code, ok := preflightPathChecks(runtime, "mv", []pathCheck{
		{path: src, op: contract.PathOpRead, unsupportedMessage: "mv: source path is not supported"},
		{path: src, op: contract.PathOpRemove, unsupportedMessage: "mv: remove is not supported"},
		{path: dest, op: contract.PathOpWrite, unsupportedMessage: "mv: write is not supported"},
	}); !ok {
		return out, code
	}
	content, err := runtime.Ops.ReadRawContent(runtime.Ctx, src)
	if err != nil {
		return fmt.Sprintf("mv: %v", err), contract.ExitCodeGeneral
	}
	if err := runtime.Ops.WriteFile(runtime.Ctx, dest, content); err != nil {
		if errors.Is(err, contract.ErrUnsupported) {
			return "mv: write is not supported", contract.ExitCodeUnsupported
		}
		return fmt.Sprintf("mv: %v", err), contract.ExitCodeGeneral
	}
	if err := runtime.Ops.RemoveFile(runtime.Ctx, src); err != nil {
		if errors.Is(err, contract.ErrUnsupported) {
			return "mv: remove is not supported", contract.ExitCodeUnsupported
		}
		return fmt.Sprintf("mv: %v", err), contract.ExitCodeGeneral
	}
	return "", 0
}
