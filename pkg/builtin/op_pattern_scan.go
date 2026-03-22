package builtin

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/khicago/simsh/pkg/contract"
	"github.com/khicago/simsh/pkg/engine"
)

type grepArgs struct {
	pattern   string
	path      string
	regex     bool
	recursive bool
	listFiles bool
	before    int
	after     int
}

func specGrep() engine.CommandSpec {
	return engine.CommandSpec{
		Name:   CommandGrep,
		Manual: "grep [-E|-F] [-r] [-l] [-A N] [-B N] [-C N] PATTERN [PATH]",
		Tips: []string{
			"Use -r for directory search and -l to list matched files only.",
			"Context flags -A/-B/-C include neighboring lines around each match.",
		},
		Examples:       ExamplesFor("grep"),
		DetailedManual: LoadEmbeddedManual("grep"),
		Run:            runGrep,
	}
}

func runGrep(runtime engine.CommandRuntime, args []string) (string, int) {
	opts, errMsg := parseGrepArgs(args, runtime.Ops.RequireAbsolutePath)
	if errMsg != "" {
		return errMsg, contract.ExitCodeUsage
	}
	match, err := buildMatcher(opts.pattern, opts.regex)
	if err != nil {
		return fmt.Sprintf("grep: %v", err), contract.ExitCodeUsage
	}
	if runtime.HasStdin && opts.path == "" {
		if opts.listFiles {
			if grepHasMatch(runtime.Stdin, match) {
				return "(stdin)", 0
			}
			return "", contract.ExitCodeGeneral
		}
		out := grepTextWithContext(runtime.Stdin, match, opts.before, opts.after, "")
		if len(out) == 0 {
			return "", contract.ExitCodeGeneral
		}
		return strings.Join(out, "\n"), 0
	}
	if !runtime.HasStdin && opts.path == "" {
		return "grep: expected stdin input or one file/directory path", contract.ExitCodeUsage
	}
	target := opts.path
	paths, err := runtime.Ops.ResolveSearchPaths(runtime.Ctx, target, opts.recursive)
	if err != nil {
		return fmt.Sprintf("grep: %v", err), contract.ExitCodeGeneral
	}
	lines := make([]string, 0)
	for _, filePath := range paths {
		raw, err := runtime.Ops.ReadRawContent(runtime.Ctx, filePath)
		if err != nil {
			return fmt.Sprintf("grep: %v", err), contract.ExitCodeGeneral
		}
		if opts.listFiles {
			if grepHasMatch(raw, match) {
				lines = append(lines, filePath)
			}
			continue
		}
		lines = append(lines, grepTextWithContext(raw, match, opts.before, opts.after, filePath)...)
	}
	if len(lines) == 0 {
		return "", contract.ExitCodeGeneral
	}
	return strings.Join(lines, "\n"), 0
}

func parseGrepArgs(args []string, requireAbsolutePath func(string) (string, error)) (grepArgs, string) {
	opts := grepArgs{}
	idx := 0
	for idx < len(args) {
		arg := args[idx]
		if !strings.HasPrefix(arg, "-") || arg == "-" {
			break
		}
		switch {
		case arg == "-E":
			opts.regex = true
		case arg == "-F":
			opts.regex = false
		case arg == "-r":
			opts.recursive = true
		case arg == "-l":
			opts.listFiles = true
		case arg == "-A" || strings.HasPrefix(arg, "-A"):
			value, consumed, err := parseGrepContextArg(arg, idx, args)
			if err != nil {
				return opts, fmt.Sprintf("grep: %v", err)
			}
			opts.after = value
			idx += consumed
		case arg == "-B" || strings.HasPrefix(arg, "-B"):
			value, consumed, err := parseGrepContextArg(arg, idx, args)
			if err != nil {
				return opts, fmt.Sprintf("grep: %v", err)
			}
			opts.before = value
			idx += consumed
		case arg == "-C" || strings.HasPrefix(arg, "-C"):
			value, consumed, err := parseGrepContextArg(arg, idx, args)
			if err != nil {
				return opts, fmt.Sprintf("grep: %v", err)
			}
			opts.before = value
			opts.after = value
			idx += consumed
		default:
			return opts, fmt.Sprintf("grep: unsupported flag %s", arg)
		}
		idx++
	}
	if idx >= len(args) {
		return opts, "grep: missing pattern"
	}
	opts.pattern = args[idx]
	idx++
	if idx < len(args) {
		pathValue, err := requireAbsolutePath(args[idx])
		if err != nil {
			return opts, fmt.Sprintf("grep: %v", err)
		}
		opts.path = pathValue
		idx++
	}
	if idx < len(args) {
		return opts, fmt.Sprintf("grep: unexpected argument: %s", args[idx])
	}
	return opts, ""
}

func grepHasMatch(raw string, match func(string) bool) bool {
	for _, line := range splitRawLines(raw) {
		if match(line) {
			return true
		}
	}
	return false
}

func parseGrepContextArg(arg string, idx int, args []string) (int, int, error) {
	if len(arg) < 2 {
		return 0, 0, fmt.Errorf("invalid context flag")
	}
	if len(arg) == 2 {
		if idx+1 >= len(args) {
			return 0, 0, fmt.Errorf("%s requires non-negative integer", arg)
		}
		value, err := parseNonNegativeInt(args[idx+1])
		if err != nil {
			return 0, 0, err
		}
		return value, 1, nil
	}
	suffix := strings.TrimPrefix(arg[2:], "=")
	value, err := parseNonNegativeInt(suffix)
	if err != nil {
		return 0, 0, err
	}
	return value, 0, nil
}

func buildMatcher(pattern string, regex bool) (func(string) bool, error) {
	if regex {
		re, err := regexp.Compile(pattern)
		if err != nil {
			return nil, err
		}
		return re.MatchString, nil
	}
	return func(line string) bool {
		return strings.Contains(line, pattern)
	}, nil
}

func grepTextWithContext(raw string, match func(string) bool, before int, after int, filePath string) []string {
	lines := splitRawLines(raw)
	if len(lines) == 0 {
		return nil
	}
	matched := make([]bool, len(lines))
	include := make([]bool, len(lines))
	for i, line := range lines {
		if !match(line) {
			continue
		}
		matched[i] = true
		start := i - before
		if start < 0 {
			start = 0
		}
		end := i + after
		if end >= len(lines) {
			end = len(lines) - 1
		}
		for j := start; j <= end; j++ {
			include[j] = true
		}
	}
	out := make([]string, 0)
	for i, line := range lines {
		if !include[i] {
			continue
		}
		lineNo := i + 1
		if filePath != "" {
			sep := '-'
			if matched[i] {
				sep = ':'
			}
			out = append(out, fmt.Sprintf("%s%c%d:%s", filePath, sep, lineNo, line))
			continue
		}
		if matched[i] {
			out = append(out, fmt.Sprintf("%d:%s", lineNo, line))
		} else {
			out = append(out, fmt.Sprintf("%d-%s", lineNo, line))
		}
	}
	return out
}
