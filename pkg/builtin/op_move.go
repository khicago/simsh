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
		Manual: "mv [--confirm] [--json] SRC_PATH DEST_PATH",
		Tips: []string{
			"Moves a file from source to destination.",
			"Use --confirm or --json when you want explicit success feedback without changing the default silent behavior.",
			"Mount-backed virtual paths are immutable and cannot be moved.",
		},
		StructuredOutput: "move summary",
		StructuredFlags:  []string{"--confirm", "--json"},
		Examples:         ExamplesFor("mv"),
		DetailedManual:   LoadEmbeddedManual("mv"),
		Run:              runMv,
	}
}

func runMv(runtime engine.CommandRuntime, args []string) (string, int) {
	filteredArgs, confirm, jsonOutput, out, code, ok := extractMutationOutputFlags("mv", args)
	if !ok {
		return out, code
	}
	if len(filteredArgs) != 2 {
		return "mv: expected exactly two arguments: SRC DEST", contract.ExitCodeUsage
	}
	src, err := runtime.Ops.RequireAbsolutePath(filteredArgs[0])
	if err != nil {
		return fmt.Sprintf("mv: %v", err), contract.ExitCodeUsage
	}
	dest, err := runtime.Ops.RequireAbsolutePath(filteredArgs[1])
	if err != nil {
		return fmt.Sprintf("mv: %v", err), contract.ExitCodeUsage
	}
	if !runtime.Ops.Policy.AllowWrite() {
		traceDeniedPaths(runtime, src, dest)
		return "mv: write is not supported", contract.ExitCodeUnsupported
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
	rendered, _, err := renderTransferMutation(confirm, jsonOutput, mutationTransfer{
		Src:   src,
		Dest:  dest,
		Bytes: len(content),
	}, "moved")
	if err != nil {
		return fmt.Sprintf("mv: %v", err), contract.ExitCodeGeneral
	}
	return rendered, 0
}
