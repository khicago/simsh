package cmd

import (
	"context"
	"strings"

	runtimeengine "github.com/khicago/simsh/pkg/engine/runtime"
	"github.com/khicago/simsh/pkg/fs"
	"github.com/khicago/simsh/pkg/sh"
)

type EnvironmentOptions = runtimeengine.Options
type Environment = runtimeengine.Stack
type Executor interface {
	Execute(ctx context.Context, commandLine string) (string, int)
}

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
		"- package `engine`: command language, parser, dispatch, traces, and prepared execution",
		"- package `builtin`: default ACI command surface",
		"- package `fs`: AI-oriented virtual filesystem contract and adapters",
		"- package `sh`: compatibility wrapper plus generated runtime-profile text",
		"- package `cmd`: runtime entrypoints (CLI/TUI/serve) calling `engine/runtime`",
		"- engine runtime wires `engine + builtin + fs` into a request-safe runtime instance",
		"- CLI default interactive mode uses TUI and also exposes `serve -P` for HTTP integration",
	}
	return strings.Join(lines, "\n") + "\n"
}
