package builtin

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/khicago/simsh/pkg/contract"
	"github.com/khicago/simsh/pkg/engine"
)

func specTee() engine.CommandSpec {
	return engine.CommandSpec{
		Name:   CommandTee,
		Manual: "echo data | tee [--confirm] [--json] [-a] PATH",
		Tips: []string{
			"Use -a to append instead of replacing file content.",
			"Default output preserves stdin passthrough; --confirm and --json replace stdout with an explicit success summary.",
			"tee requires stdin, usually from a pipeline or heredoc.",
		},
		StructuredOutput: "write summary",
		StructuredFlags:  []string{"--confirm", "--json"},
		Examples:         ExamplesFor("tee"),
		DetailedManual:   LoadEmbeddedManual("tee"),
		Run:              runTee,
	}
}

func runTee(runtime engine.CommandRuntime, args []string) (string, int) {
	filteredArgs, confirm, jsonOutput, out, code, ok := extractMutationOutputFlags("tee", args)
	if !ok {
		return out, code
	}
	appendMode := false
	target := ""
	for _, arg := range filteredArgs {
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
	if !runtime.Ops.Policy.AllowWrite() {
		traceDeniedPaths(runtime, target)
		return "tee: write is not allowed by policy", contract.ExitCodeUnsupported
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
		return renderTeeSuccess(runtime.Stdin, target, len(runtime.Stdin), "append", confirm, jsonOutput)
	}
	if err := runtime.Ops.WriteFile(runtime.Ctx, target, runtime.Stdin); err != nil {
		if errors.Is(err, contract.ErrUnsupported) {
			return "tee: write is not supported", contract.ExitCodeUnsupported
		}
		return fmt.Sprintf("tee: %v", err), contract.ExitCodeGeneral
	}
	return renderTeeSuccess(runtime.Stdin, target, len(runtime.Stdin), "write", confirm, jsonOutput)
}

func renderTeeSuccess(stdin string, path string, bytes int, mode string, confirm bool, jsonOutput bool) (string, int) {
	if jsonOutput {
		raw, err := json.Marshal(struct {
			Path  string `json:"path"`
			Bytes int    `json:"bytes"`
			Mode  string `json:"mode"`
		}{
			Path:  path,
			Bytes: bytes,
			Mode:  mode,
		})
		if err != nil {
			return fmt.Sprintf("tee: %v", err), contract.ExitCodeGeneral
		}
		return string(raw), 0
	}
	if confirm {
		return fmt.Sprintf("wrote %s bytes=%d mode=%s", path, bytes, mode), 0
	}
	return stdin, 0
}
