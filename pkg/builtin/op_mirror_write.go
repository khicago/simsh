package builtin

import (
	"errors"
	"fmt"
	"strings"

	"github.com/khicago/simsh/pkg/contract"
	"github.com/khicago/simsh/pkg/engine"
)

func specTee() engine.CommandSpec {
	return engine.CommandSpec{
		Name:   CommandTee,
		Manual: "echo data | tee [-a] PATH",
		Tips: []string{
			"Use -a to append instead of replacing file content.",
			"tee requires stdin, usually from a pipeline or heredoc.",
		},
		Examples:       ExamplesFor("tee"),
		DetailedManual: LoadEmbeddedManual("tee"),
		Run:            runTee,
	}
}

func runTee(runtime engine.CommandRuntime, args []string) (string, int) {
	if !runtime.Ops.Policy.AllowWrite() {
		return "tee: write is not allowed by policy", contract.ExitCodeUnsupported
	}
	appendMode := false
	target := ""
	for _, arg := range args {
		if arg == "-a" {
			appendMode = true
			continue
		}
		if strings.HasPrefix(arg, "-") {
			return fmt.Sprintf("tee: unsupported flag %s", arg), contract.ExitCodeUsage
		}
		if target != "" {
			return "tee: expected exactly one target file", contract.ExitCodeUsage
		}
		pathValue, err := runtime.Ops.RequireAbsolutePath(arg)
		if err != nil {
			return fmt.Sprintf("tee: %v", err), contract.ExitCodeUsage
		}
		target = pathValue
	}
	if target == "" {
		return "tee: missing target file", contract.ExitCodeUsage
	}
	if !runtime.HasStdin {
		return "tee: missing stdin input (use pipeline, e.g. echo \"x\" | tee /task_outputs/a.md)", contract.ExitCodeUsage
	}
	if err := runtime.Ops.Policy.CheckWriteSize(len(runtime.Stdin)); err != nil {
		return fmt.Sprintf("tee: content exceeds write limit (%d bytes)", runtime.Ops.Policy.MaxWriteBytes), contract.ExitCodeGeneral
	}
	if appendMode {
		if runtime.Ops.AppendFile == nil {
			return "tee: append is not supported", contract.ExitCodeUnsupported
		}
		if err := runtime.Ops.AppendFile(runtime.Ctx, target, runtime.Stdin); err != nil {
			if errors.Is(err, contract.ErrUnsupported) {
				return "tee: append is not supported", contract.ExitCodeUnsupported
			}
			return fmt.Sprintf("tee: %v", err), contract.ExitCodeGeneral
		}
		return runtime.Stdin, 0
	}
	if err := runtime.Ops.WriteFile(runtime.Ctx, target, runtime.Stdin); err != nil {
		if errors.Is(err, contract.ErrUnsupported) {
			return "tee: write is not supported", contract.ExitCodeUnsupported
		}
		return fmt.Sprintf("tee: %v", err), contract.ExitCodeGeneral
	}
	return runtime.Stdin, 0
}
