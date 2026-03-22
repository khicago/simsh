package rm

import (
	"embed"
	"errors"
	"fmt"
	"strings"

	"github.com/khicago/simsh/pkg/contract"
	"github.com/khicago/simsh/pkg/engine"
)

var examples = []string{"rm /task_outputs/old.md", "rm /task_outputs/temp1.txt /task_outputs/temp2.txt"}

//go:embed manual.md
var manualFS embed.FS

func detailedManual() string {
	data, err := manualFS.ReadFile("manual.md")
	if err != nil {
		return ""
	}
	return string(data)
}

func Spec() engine.CommandSpec {
	return engine.CommandSpec{
		Name:   "rm",
		Manual: "rm PATH...",
		Tips: []string{
			"Removes files. Does not support directory removal.",
		},
		Examples:       append([]string(nil), examples...),
		DetailedManual: detailedManual(),
		Run:            run,
	}
}

func run(runtime engine.CommandRuntime, args []string) (string, int) {
	if !runtime.Ops.Policy.AllowWrite() {
		return "rm: write is not supported", contract.ExitCodeUnsupported
	}
	paths := make([]string, 0, len(args))
	for _, arg := range args {
		if strings.HasPrefix(arg, "-") {
			return fmt.Sprintf("rm: unsupported flag %s", arg), contract.ExitCodeUsage
		}
		pathValue, err := runtime.Ops.RequireAbsolutePath(arg)
		if err != nil {
			return fmt.Sprintf("rm: %v", err), contract.ExitCodeUsage
		}
		paths = append(paths, pathValue)
	}
	if len(paths) == 0 {
		return "rm: missing operand", contract.ExitCodeUsage
	}
	for _, p := range paths {
		if err := runtime.Ops.RemoveFile(runtime.Ctx, p); err != nil {
			if errors.Is(err, contract.ErrUnsupported) {
				return "rm: not supported", contract.ExitCodeUnsupported
			}
			return fmt.Sprintf("rm: %v", err), contract.ExitCodeGeneral
		}
	}
	return "", 0
}
