package mkdir

import (
	"embed"
	"errors"
	"fmt"
	"strings"

	"github.com/khicago/simsh/pkg/contract"
	"github.com/khicago/simsh/pkg/engine"
)

var examples = []string{"mkdir /task_outputs/reports", "mkdir -p /task_outputs/a/b/c"}

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
		Name:   "mkdir",
		Manual: "mkdir [-p] PATH...",
		Tips: []string{
			"Creates directories. -p creates parent directories as needed.",
		},
		Examples:       append([]string(nil), examples...),
		DetailedManual: detailedManual(),
		Run:            run,
	}
}

func run(runtime engine.CommandRuntime, args []string) (string, int) {
	if !runtime.Ops.Policy.AllowWrite() {
		return "mkdir: write is not supported", contract.ExitCodeUnsupported
	}
	paths := make([]string, 0, len(args))
	for _, arg := range args {
		if arg == "-p" {
			continue
		}
		if strings.HasPrefix(arg, "-") {
			return fmt.Sprintf("mkdir: unsupported flag %s", arg), contract.ExitCodeUsage
		}
		pathValue, err := runtime.Ops.RequireAbsolutePath(arg)
		if err != nil {
			return fmt.Sprintf("mkdir: %v", err), contract.ExitCodeUsage
		}
		paths = append(paths, pathValue)
	}
	if len(paths) == 0 {
		return "mkdir: missing operand", contract.ExitCodeUsage
	}
	for _, p := range paths {
		if err := runtime.Ops.MakeDir(runtime.Ctx, p); err != nil {
			if errors.Is(err, contract.ErrUnsupported) {
				return "mkdir: not supported", contract.ExitCodeUnsupported
			}
			return fmt.Sprintf("mkdir: %v", err), contract.ExitCodeGeneral
		}
	}
	return "", 0
}
