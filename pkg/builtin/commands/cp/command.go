package cp

import (
	"embed"
	"errors"
	"fmt"

	"github.com/khicago/simsh/pkg/contract"
	"github.com/khicago/simsh/pkg/engine"
)

var examples = []string{"cp /knowledge_base/template.md /task_outputs/report.md", "cp /task_outputs/input.txt /task_outputs/output.txt"}

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
		Name:   "cp",
		Manual: "cp SRC_ABS DEST_ABS",
		Tips: []string{
			"Copies a file from source to destination. Both paths must be absolute.",
		},
		Examples:       append([]string(nil), examples...),
		DetailedManual: detailedManual(),
		Run:            run,
	}
}

func run(runtime engine.CommandRuntime, args []string) (string, int) {
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
