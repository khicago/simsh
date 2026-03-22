package mv

import (
	"embed"
	"errors"
	"fmt"

	"github.com/khicago/simsh/pkg/contract"
	"github.com/khicago/simsh/pkg/engine"
)

var examples = []string{"mv /task_outputs/draft.md /task_outputs/final.md"}

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
		Name:   "mv",
		Manual: "mv SRC_PATH DEST_PATH",
		Tips: []string{
			"Moves a file from source to destination.",
		},
		Examples:       append([]string(nil), examples...),
		DetailedManual: detailedManual(),
		Run:            run,
	}
}

func run(runtime engine.CommandRuntime, args []string) (string, int) {
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
