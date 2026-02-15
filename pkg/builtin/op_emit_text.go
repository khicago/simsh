package builtin

import (
	"strings"

	"github.com/khicago/simsh/pkg/engine"
)

func specEcho() engine.CommandSpec {
	return engine.CommandSpec{
		Name:   CommandEcho,
		Manual: "echo [ARGS...]",
		Tips: []string{
			"Use echo for deterministic plain text output.",
		},
		Examples:       ExamplesFor("echo"),
		DetailedManual: LoadEmbeddedManual("echo"),
		Run:            runEcho,
	}
}

func runEcho(runtime engine.CommandRuntime, args []string) (string, int) {
	_ = runtime
	return strings.Join(args, " "), 0
}
