package builtin

import (
	"fmt"
	"path"
	"strings"

	"github.com/khicago/simsh/pkg/contract"
	"github.com/khicago/simsh/pkg/engine"
)

type findArgs struct {
	target   string
	patterns []string
	execArgs []string
	execPlus bool
}

func specFind() engine.CommandSpec {
	return engine.CommandSpec{
		Name:   CommandFind,
		Manual: "find [ABS_DIR] -name PATTERN [-o -name PATTERN ...] [-exec CMD {} ';'|+]",
		Tips: []string{
			"Use -o to combine multiple -name patterns.",
			"-exec ... + batches matched paths in one invocation.",
		},
		Examples:       ExamplesFor("find"),
		DetailedManual: LoadEmbeddedManual("find"),
		Run:            runFind,
	}
}

func runFind(runtime engine.CommandRuntime, args []string) (string, int) {
	opts, errMsg := parseFindArgs(args, runtime.Ops.RequireAbsolutePath, runtime.Ops.RootDir)
	if errMsg != "" {
		return errMsg, contract.ExitCodeUsage
	}
	ok, err := runtime.Ops.IsDirPath(runtime.Ctx, opts.target)
	if err != nil {
		return fmt.Sprintf("find: %v", err), contract.ExitCodeGeneral
	}
	if !ok {
		return fmt.Sprintf("find: %s: Not a directory", opts.target), contract.ExitCodeGeneral
	}
	files, err := runtime.Ops.CollectFilesUnder(runtime.Ctx, opts.target)
	if err != nil {
		return fmt.Sprintf("find: %v", err), contract.ExitCodeGeneral
	}
	matchedFiles := make([]string, 0)
	for _, filePath := range files {
		name := path.Base(filePath)
		matched := false
		for _, pattern := range opts.patterns {
			ok, matchErr := matchFindName(name, pattern)
			if matchErr != nil {
				return fmt.Sprintf("find: invalid -name pattern: %v", matchErr), contract.ExitCodeUsage
			}
			if ok {
				matched = true
				break
			}
		}
		if matched {
			matchedFiles = append(matchedFiles, filePath)
		}
	}
	if len(opts.execArgs) == 0 {
		return strings.Join(matchedFiles, "\n"), 0
	}
	return runFindExec(matchedFiles, opts.execArgs, opts.execPlus, runtime.Dispatch)
}

func parseFindArgs(args []string, requireAbsolutePath func(string) (string, error), root string) (findArgs, string) {
	opts := findArgs{target: root, patterns: make([]string, 0)}
	expectPattern := false
	sawPattern := false
	lastWasOr := false
	targetSet := false

	for idx := 0; idx < len(args); idx++ {
		arg := args[idx]
		if expectPattern {
			opts.patterns = append(opts.patterns, arg)
			expectPattern = false
			sawPattern = true
			lastWasOr = false
			continue
		}
		switch arg {
		case "-name":
			expectPattern = true
			continue
		case "-o":
			if !sawPattern || lastWasOr {
				return opts, "find: unexpected -o"
			}
			lastWasOr = true
			continue
		case "-exec":
			if len(opts.execArgs) > 0 {
				return opts, "find: only one -exec clause is supported"
			}
			idx++
			if idx >= len(args) {
				return opts, "find: -exec requires command"
			}
			start := idx
			for idx < len(args) && args[idx] != ";" && args[idx] != "+" {
				idx++
			}
			if idx >= len(args) {
				return opts, "find: -exec requires terminator ';' or '+'"
			}
			if idx == start {
				return opts, "find: -exec requires command"
			}
			opts.execArgs = append(opts.execArgs, args[start:idx]...)
			opts.execPlus = args[idx] == "+"
			lastWasOr = false
			continue
		}
		if strings.HasPrefix(arg, "-") {
			return opts, fmt.Sprintf("find: unsupported flag %s", arg)
		}
		if targetSet {
			return opts, fmt.Sprintf("find: unexpected argument: %s", arg)
		}
		pathValue, err := requireAbsolutePath(arg)
		if err != nil {
			return opts, fmt.Sprintf("find: %v", err)
		}
		opts.target = pathValue
		targetSet = true
	}

	if expectPattern {
		return opts, "find: -name requires pattern"
	}
	if lastWasOr {
		return opts, "find: expression cannot end with -o"
	}
	if len(opts.patterns) == 0 {
		opts.patterns = []string{"*"}
	}
	return opts, ""
}

func runFindExec(matchedFiles []string, execArgs []string, execPlus bool, dispatch func(args []string, input string, hasInput bool) (string, int)) (string, int) {
	if len(execArgs) == 0 || len(matchedFiles) == 0 {
		return "", 0
	}
	if dispatch == nil {
		return "find: command dispatcher is required", contract.ExitCodeGeneral
	}
	if execPlus {
		args := expandFindExecArgs(execArgs, matchedFiles)
		if len(args) == 0 {
			return "find: -exec requires command", contract.ExitCodeUsage
		}
		return dispatch(args, "", false)
	}

	out := make([]string, 0)
	for _, filePath := range matchedFiles {
		args := expandFindExecArgs(execArgs, []string{filePath})
		if len(args) == 0 {
			return "find: -exec requires command", contract.ExitCodeUsage
		}
		cmdOut, code := dispatch(args, "", false)
		if code != 0 {
			if strings.TrimSpace(cmdOut) == "" {
				cmdOut = fmt.Sprintf("find: -exec %s failed for %s", args[0], filePath)
			}
			return cmdOut, code
		}
		if strings.TrimSpace(cmdOut) != "" {
			out = append(out, cmdOut)
		}
	}
	return strings.Join(out, "\n"), 0
}

func expandFindExecArgs(execArgs []string, paths []string) []string {
	out := make([]string, 0, len(execArgs)+len(paths))
	sawPlaceholder := false
	for _, arg := range execArgs {
		if arg != "{}" {
			out = append(out, arg)
			continue
		}
		out = append(out, paths...)
		sawPlaceholder = true
	}
	if !sawPlaceholder {
		out = append(out, paths...)
	}
	return out
}

func matchFindName(name, pattern string) (bool, error) {
	if strings.ContainsAny(pattern, "*?[]") {
		return path.Match(pattern, name)
	}
	return name == pattern, nil
}
