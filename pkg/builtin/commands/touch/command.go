package touch

import (
	"embed"
	"errors"
	"fmt"
	"strings"

	"github.com/khicago/simsh/pkg/contract"
	"github.com/khicago/simsh/pkg/engine"
)

var examples = []string{"touch /task_outputs/notes.md"}

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
		Name:   "touch",
		Manual: "touch PATH...",
		Tips: []string{
			"Creates empty files if they do not exist.",
		},
		Examples:       append([]string(nil), examples...),
		DetailedManual: detailedManual(),
		Run:            run,
	}
}

func run(runtime engine.CommandRuntime, args []string) (string, int) {
	if !runtime.Ops.Policy.AllowWrite() {
		return "touch: write is not supported", contract.ExitCodeUnsupported
	}
	paths := make([]string, 0, len(args))
	for _, arg := range args {
		if strings.HasPrefix(arg, "-") {
			return fmt.Sprintf("touch: unsupported flag %s", arg), contract.ExitCodeUsage
		}
		pathValue, err := runtime.Ops.RequireAbsolutePath(arg)
		if err != nil {
			return fmt.Sprintf("touch: %v", err), contract.ExitCodeUsage
		}
		paths = append(paths, pathValue)
	}
	if len(paths) == 0 {
		return "touch: missing operand", contract.ExitCodeUsage
	}
	for _, p := range paths {
		_, err := runtime.Ops.ReadRawContent(runtime.Ctx, p)
		if err == nil {
			continue
		}
		if writeErr := runtime.Ops.WriteFile(runtime.Ctx, p, ""); writeErr != nil {
			if errors.Is(writeErr, contract.ErrUnsupported) {
				return "touch: write is not supported", contract.ExitCodeUnsupported
			}
			return fmt.Sprintf("touch: %v", writeErr), contract.ExitCodeGeneral
		}
	}
	return "", 0
}
