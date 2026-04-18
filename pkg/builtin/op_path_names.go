package builtin

import (
	"fmt"
	"path"
	"strings"

	"github.com/khicago/simsh/pkg/contract"
	"github.com/khicago/simsh/pkg/engine"
)

func specDirname() engine.CommandSpec {
	return engine.CommandSpec{
		Name:   CommandDirname,
		Manual: "dirname [--] PATH",
		Tips: []string{
			"Prints the parent of a resolved virtual path.",
			"Relative paths resolve against the session working directory before dirname is applied.",
		},
		Examples:       ExamplesFor("dirname"),
		DetailedManual: LoadEmbeddedManual("dirname"),
		Run:            runDirname,
	}
}

func specBasename() engine.CommandSpec {
	return engine.CommandSpec{
		Name:   CommandBasename,
		Manual: "basename [--] PATH",
		Tips: []string{
			"Prints the final component of a resolved virtual path.",
			"Relative paths resolve against the session working directory before basename is applied.",
		},
		Examples:       ExamplesFor("basename"),
		DetailedManual: LoadEmbeddedManual("basename"),
		Run:            runBasename,
	}
}

func runDirname(runtime engine.CommandRuntime, args []string) (string, int) {
	pathValue, errMsg := parseSinglePathNameArg("dirname", args, runtime.Ops.RequireAbsolutePath)
	if errMsg != "" {
		return errMsg, contract.ExitCodeUsage
	}
	return path.Dir(pathValue), 0
}

func runBasename(runtime engine.CommandRuntime, args []string) (string, int) {
	pathValue, errMsg := parseSinglePathNameArg("basename", args, runtime.Ops.RequireAbsolutePath)
	if errMsg != "" {
		return errMsg, contract.ExitCodeUsage
	}
	return path.Base(pathValue), 0
}

func parseSinglePathNameArg(command string, args []string, requireAbsolutePath func(string) (string, error)) (string, string) {
	positional := make([]string, 0, 1)
	parseOptions := true
	for _, arg := range args {
		if parseOptions && arg == "--" {
			parseOptions = false
			continue
		}
		if parseOptions && strings.HasPrefix(arg, "-") {
			return "", fmt.Sprintf("%s: unsupported flag %s", command, arg)
		}
		positional = append(positional, arg)
	}
	if len(positional) != 1 {
		return "", fmt.Sprintf("%s: requires exactly one PATH", command)
	}
	pathValue, err := requireAbsolutePath(positional[0])
	if err != nil {
		return "", fmt.Sprintf("%s: %v", command, err)
	}
	return pathValue, ""
}
