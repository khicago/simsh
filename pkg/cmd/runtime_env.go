package cmd

import (
	"strings"

	runtimeengine "github.com/khicago/simsh/pkg/engine/runtime"
	"github.com/khicago/simsh/pkg/fs"
	"github.com/khicago/simsh/pkg/sh"
)

type EnvironmentOptions = runtimeengine.Options
type Environment = runtimeengine.Stack

func NewEnvironment(opts EnvironmentOptions) (*Environment, error) {
	return runtimeengine.New(opts)
}

func DescribeMarkdown() string {
	lines := []string{
		"# simsh Runtime Profile",
		"",
		sh.DescribeMarkdown(),
		"",
		fs.DescribeMarkdown(),
		"",
		"## Runtime Composition",
		"- package `engine`: runtime composition layer (`sh + fs + policy/profile`)",
		"- package `sh`: command language and execution semantics",
		"- package `fs`: AI-oriented virtual filesystem contract and adapters",
		"- package `cmd`: runtime entrypoints (CLI/TUI/serve) calling `engine`",
		"- engine runtime wires `sh + fs` into a request-safe runtime instance",
		"- CLI default interactive mode uses TUI and also exposes `serve -P` for HTTP integration",
	}
	return strings.Join(lines, "\n") + "\n"
}
