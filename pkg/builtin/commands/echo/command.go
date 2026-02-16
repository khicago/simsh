package echo

import (
	"embed"
	"strings"

	"github.com/khicago/simsh/pkg/engine"
)

var examples = []string{
	"echo hello world",
	"echo \"report data\" | tee /task_outputs/report.txt",
}

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
		Name:   "echo",
		Manual: "echo [ARGS...]",
		Tips: []string{
			"Use echo for deterministic plain text output.",
		},
		Examples:       append([]string(nil), examples...),
		DetailedManual: detailedManual(),
		Run:            run,
	}
}

func run(runtime engine.CommandRuntime, args []string) (string, int) {
	_ = runtime
	return strings.Join(args, " "), 0
}
