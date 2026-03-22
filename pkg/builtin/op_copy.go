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
		Manual: "cp [--confirm] [--json] SRC_PATH DEST_PATH",
		Tips: []string{
			"Copies a file from source to destination.",
			"Use --confirm or --json when you want explicit success feedback without changing the default silent behavior.",
			"Mount-backed virtual paths are immutable and not valid copy operands.",
		},
		StructuredOutput: "copy summary",
		StructuredFlags:  []string{"--confirm", "--json"},
		Examples:         ExamplesFor("cp"),
		DetailedManual:   LoadEmbeddedManual("cp"),
		Run:              runCp,
	}
}

func runCp(runtime engine.CommandRuntime, args []string) (string, int) {
	filteredArgs, confirm, jsonOutput, out, code, ok := extractMutationOutputFlags("cp", args)
	if !ok {
		return out, code
	}
	if len(filteredArgs) != 2 {
		return "cp: expected exactly two arguments: SRC DEST", contract.ExitCodeUsage
	}
	src, err := runtime.Ops.RequireAbsolutePath(filteredArgs[0])
	if err != nil {
		return fmt.Sprintf("cp: %v", err), contract.ExitCodeUsage
	}
	dest, err := runtime.Ops.RequireAbsolutePath(filteredArgs[1])
	if err != nil {
		return fmt.Sprintf("cp: %v", err), contract.ExitCodeUsage
	}
	if !runtime.Ops.Policy.AllowWrite() {
		traceDeniedPaths(runtime, dest)
		return "cp: write is not supported", contract.ExitCodeUnsupported
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
	rendered, _, err := renderTransferMutation(confirm, jsonOutput, mutationTransfer{
		Src:   src,
		Dest:  dest,
		Bytes: len(content),
	}, "copied")
	if err != nil {
		return fmt.Sprintf("cp: %v", err), contract.ExitCodeGeneral
	}
	return rendered, 0
}
