package builtin

import (
	"fmt"
	"strings"

	"github.com/khicago/simsh/pkg/contract"
	"github.com/khicago/simsh/pkg/engine"
)

type headTailArgs struct {
	count int
	path  string
}

func specHead() engine.CommandSpec {
	return engine.CommandSpec{
		Name:   CommandHead,
		Manual: "head [-n N|-N] [ABS_FILE]",
		Tips: []string{
			"Use stdin input when no file path is provided.",
			"-n accepts non-negative integers only.",
		},
		Examples:       ExamplesFor("head"),
		DetailedManual: LoadEmbeddedManual("head"),
		Run:            runHead,
	}
}

func specTail() engine.CommandSpec {
	return engine.CommandSpec{
		Name:   CommandTail,
		Manual: "tail [-n N|-N] [ABS_FILE]",
		Tips: []string{
			"Use stdin input when no file path is provided.",
			"-n accepts non-negative integers only.",
		},
		Examples:       ExamplesFor("tail"),
		DetailedManual: LoadEmbeddedManual("tail"),
		Run:            runTail,
	}
}

func runHead(runtime engine.CommandRuntime, args []string) (string, int) {
	opts, errMsg := parseHeadTailArgs(CommandHead, args, runtime.Ops.RequireAbsolutePath)
	if errMsg != "" {
		return errMsg, contract.ExitCodeUsage
	}
	raw, out, code := loadHeadTailSource(runtime, CommandHead, opts.path)
	if code != 0 {
		return out, code
	}
	lines := splitRawLines(raw)
	if opts.count <= 0 || len(lines) == 0 {
		return "", 0
	}
	if opts.count >= len(lines) {
		return strings.Join(lines, "\n"), 0
	}
	return strings.Join(lines[:opts.count], "\n"), 0
}

func runTail(runtime engine.CommandRuntime, args []string) (string, int) {
	opts, errMsg := parseHeadTailArgs(CommandTail, args, runtime.Ops.RequireAbsolutePath)
	if errMsg != "" {
		return errMsg, contract.ExitCodeUsage
	}
	raw, out, code := loadHeadTailSource(runtime, CommandTail, opts.path)
	if code != 0 {
		return out, code
	}
	lines := splitRawLines(raw)
	if opts.count <= 0 || len(lines) == 0 {
		return "", 0
	}
	if opts.count >= len(lines) {
		return strings.Join(lines, "\n"), 0
	}
	return strings.Join(lines[len(lines)-opts.count:], "\n"), 0
}

func loadHeadTailSource(runtime engine.CommandRuntime, cmd string, pathValue string) (string, string, int) {
	if runtime.HasStdin && strings.TrimSpace(pathValue) == "" {
		return runtime.Stdin, "", 0
	}
	if strings.TrimSpace(pathValue) == "" {
		return "", fmt.Sprintf("%s: expected stdin input or one absolute file path", cmd), contract.ExitCodeUsage
	}
	raw, err := runtime.Ops.ReadRawContent(runtime.Ctx, pathValue)
	if err != nil {
		return "", fmt.Sprintf("%s: %v", cmd, err), contract.ExitCodeGeneral
	}
	return raw, "", 0
}

func parseHeadTailArgs(cmd string, args []string, requireAbsolutePath func(string) (string, error)) (headTailArgs, string) {
	opts := headTailArgs{count: 10}
	for i := 0; i < len(args); i++ {
		arg := args[i]
		if strings.HasPrefix(arg, "-") && arg != "-" {
			switch {
			case arg == "-n":
				i++
				if i >= len(args) {
					return opts, fmt.Sprintf("%s: -n requires a non-negative integer", cmd)
				}
				value, err := parseNonNegativeInt(args[i])
				if err != nil {
					return opts, fmt.Sprintf("%s: %v", cmd, err)
				}
				opts.count = value
			case strings.HasPrefix(arg, "-n="):
				value, err := parseNonNegativeInt(strings.TrimPrefix(arg, "-n="))
				if err != nil {
					return opts, fmt.Sprintf("%s: %v", cmd, err)
				}
				opts.count = value
			case strings.HasPrefix(arg, "-n") && len(arg) > 2:
				value, err := parseNonNegativeInt(strings.TrimPrefix(arg, "-n"))
				if err != nil {
					return opts, fmt.Sprintf("%s: %v", cmd, err)
				}
				opts.count = value
			case len(arg) > 1 && isDigits(arg[1:]):
				value, err := parseNonNegativeInt(arg[1:])
				if err != nil {
					return opts, fmt.Sprintf("%s: %v", cmd, err)
				}
				opts.count = value
			default:
				return opts, fmt.Sprintf("%s: unsupported flag %s", cmd, arg)
			}
			continue
		}
		if opts.path != "" {
			return opts, fmt.Sprintf("%s: unexpected argument: %s", cmd, arg)
		}
		pathValue, err := requireAbsolutePath(arg)
		if err != nil {
			return opts, fmt.Sprintf("%s: %v", cmd, err)
		}
		opts.path = pathValue
	}
	return opts, ""
}
