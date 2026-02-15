package builtin

import (
	"fmt"
	"strings"

	"github.com/khicago/simsh/pkg/contract"
	"github.com/khicago/simsh/pkg/engine"
)

func specCat() engine.CommandSpec {
	return engine.CommandSpec{
		Name:   CommandCat,
		Manual: "cat [-n] ABS_FILE | cat (stdin passthrough)",
		Tips: []string{
			"Use cat with no args to pass stdin through a pipeline.",
			"Use -n to display line numbers.",
		},
		Examples:       ExamplesFor("cat"),
		DetailedManual: LoadEmbeddedManual("cat"),
		Run:            runCat,
	}
}

func runCat(runtime engine.CommandRuntime, args []string) (string, int) {
	showLineNumbers := false
	filePath := ""
	for _, arg := range args {
		if arg == "-n" {
			showLineNumbers = true
			continue
		}
		if strings.HasPrefix(arg, "-") && arg != "-" {
			return fmt.Sprintf("cat: unsupported flag %s", arg), contract.ExitCodeUsage
		}
		if filePath != "" {
			return "cat: expected exactly one absolute file path", contract.ExitCodeUsage
		}
		pathValue, err := runtime.Ops.RequireAbsolutePath(arg)
		if err != nil {
			return fmt.Sprintf("cat: %v", err), contract.ExitCodeUsage
		}
		filePath = pathValue
	}
	if runtime.HasStdin && filePath == "" {
		if showLineNumbers {
			return formatWithLineNumbers(runtime.Stdin), 0
		}
		return runtime.Stdin, 0
	}
	if filePath == "" {
		return "cat: expected exactly one absolute file path", contract.ExitCodeUsage
	}
	raw, err := runtime.Ops.ReadRawContent(runtime.Ctx, filePath)
	if err != nil {
		return fmt.Sprintf("cat: %v", err), contract.ExitCodeGeneral
	}
	if showLineNumbers {
		return formatWithLineNumbers(raw), 0
	}
	return raw, 0
}
