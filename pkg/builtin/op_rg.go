package builtin

import (
	"fmt"
	"path"
	"strings"

	"github.com/khicago/simsh/pkg/contract"
	"github.com/khicago/simsh/pkg/engine"
)

type rgArgs struct {
	pattern   string
	targets   []string
	globs     []string
	caseMode  searchCaseMode
	fixed     bool
	listFiles bool
	filesOnly bool
	jsonl     bool
	before    int
	after     int
}

func specRG() engine.CommandSpec {
	return engine.CommandSpec{
		Name:   CommandRG,
		Manual: "rg [-F] [-i|-S] [-l] [-g GLOB]... [-A N] [-B N] [-C N] [--fmt jsonl] PATTERN [PATH ...] | rg --files [-g GLOB]... [PATH ...]",
		Tips: []string{
			"rg searches recursively by default and falls back to the current working directory when no path is given.",
			"Use --files to list searchable files, optionally narrowed with one or more -g globs.",
			"Use --fmt jsonl as the canonical structured mode; --json is accepted only as a compatibility alias.",
		},
		StructuredOutput: "flat match/context/file records",
		StructuredFlags:  []string{"--fmt jsonl"},
		Examples:         ExamplesFor("rg"),
		DetailedManual:   LoadEmbeddedManual("rg"),
		Run:              runRG,
	}
}

func runRG(runtime engine.CommandRuntime, args []string) (string, int) {
	opts, errMsg := parseRGArgs(args, runtime.Ops.RequireAbsolutePath, currentWorkingDir(runtime.Ops))
	if errMsg != "" {
		return errMsg, contract.ExitCodeUsage
	}
	if opts.filesOnly {
		paths, err := resolveRGSearchPaths(runtime, opts.targets, opts.globs)
		if err != nil {
			return fmt.Sprintf("rg: %v", err), contract.ExitCodeGeneral
		}
		if !opts.jsonl {
			return strings.Join(paths, "\n"), 0
		}
		return renderSearchFileJSONL(paths), 0
	}

	match, err := buildSearchMatcher(opts.pattern, searchMatcherOptions{
		Regex:    !opts.fixed,
		CaseMode: opts.caseMode,
	})
	if err != nil {
		return fmt.Sprintf("rg: %v", err), contract.ExitCodeUsage
	}
	if runtime.HasStdin && len(opts.targets) == 0 {
		if opts.listFiles {
			found := searchHasMatch(runtime.Stdin, match)
			if !opts.jsonl {
				if found {
					return "(stdin)", 0
				}
				return "", contract.ExitCodeGeneral
			}
			records := make([]searchRecord, 0, 1)
			if found {
				records = append(records, searchRecord{Stdin: true, Line: 0, Kind: "file"})
			}
			return renderSearchJSONL(records), grepExitCode(found)
		}
		if !opts.jsonl {
			lines := searchTextWithContext(runtime.Stdin, match, opts.before, opts.after, "")
			if len(lines) == 0 {
				return "", contract.ExitCodeGeneral
			}
			return strings.Join(lines, "\n"), 0
		}
		records := searchRecordsWithContext(runtime.Stdin, match, opts.before, opts.after, "", true)
		return renderSearchJSONL(records), grepExitCode(len(records) > 0)
	}

	targets := opts.targets
	if len(targets) == 0 {
		targets = []string{currentWorkingDir(runtime.Ops)}
	}
	if !opts.filesOnly {
		req := buildContractSearchRequest(opts.pattern, searchMatcherOptions{
			Regex:    !opts.fixed,
			CaseMode: opts.caseMode,
		}, opts.globs, targets, opts.listFiles, opts.before, opts.after, 0)
		if used, result, err := tryRuntimeSearch(runtime, req); err != nil {
			return fmt.Sprintf("rg: %v", err), contract.ExitCodeGeneral
		} else if used {
			return renderSearchRecords(result.Records, opts.listFiles, opts.jsonl, opts.listFiles)
		}
	}
	paths, err := resolveRGSearchPaths(runtime, opts.targets, opts.globs)
	if err != nil {
		return fmt.Sprintf("rg: %v", err), contract.ExitCodeGeneral
	}
	if opts.listFiles {
		return runRGListFiles(runtime, paths, match, opts.jsonl)
	}
	if !opts.jsonl {
		lines, err := runRGText(runtime, paths, match, opts.before, opts.after)
		if err != nil {
			return fmt.Sprintf("rg: %v", err), contract.ExitCodeGeneral
		}
		if len(lines) == 0 {
			return "", contract.ExitCodeGeneral
		}
		return strings.Join(lines, "\n"), 0
	}
	records, err := runRGJSONL(runtime, paths, match, opts.before, opts.after)
	if err != nil {
		return fmt.Sprintf("rg: %v", err), contract.ExitCodeGeneral
	}
	return renderSearchJSONL(records), grepExitCode(len(records) > 0)
}

func parseRGArgs(args []string, requireAbsolutePath func(string) (string, error), root string) (rgArgs, string) {
	opts := rgArgs{caseMode: searchCaseSensitive}
	idx := 0
	for idx < len(args) {
		arg := args[idx]
		if !strings.HasPrefix(arg, "-") || arg == "-" {
			break
		}
		switch {
		case arg == "--files":
			opts.filesOnly = true
		case arg == "--json":
			opts.jsonl = true
		case arg == "--fmt":
			if idx+1 >= len(args) {
				return opts, "rg: --fmt requires one value: jsonl"
			}
			idx++
			if strings.TrimSpace(args[idx]) != "jsonl" {
				return opts, fmt.Sprintf("rg: unsupported --fmt value %q", args[idx])
			}
			opts.jsonl = true
		case strings.HasPrefix(arg, "--fmt="):
			if strings.TrimSpace(strings.TrimPrefix(arg, "--fmt=")) != "jsonl" {
				return opts, fmt.Sprintf("rg: unsupported --fmt value %q", strings.TrimPrefix(arg, "--fmt="))
			}
			opts.jsonl = true
		case arg == "-F":
			opts.fixed = true
		case arg == "-i":
			opts.caseMode = searchCaseIgnore
		case arg == "-S":
			opts.caseMode = searchCaseSmart
		case arg == "-l":
			if opts.filesOnly {
				return opts, "rg: -l is not supported with --files"
			}
			opts.listFiles = true
		case arg == "-n" || arg == "-r":
			// Accepted for ripgrep/grep caller compatibility.
		case arg == "-A" || strings.HasPrefix(arg, "-A"):
			value, consumed, err := parseSearchContextArg(arg, idx, args)
			if err != nil {
				return opts, fmt.Sprintf("rg: %v", err)
			}
			opts.after = value
			idx += consumed
		case arg == "-B" || strings.HasPrefix(arg, "-B"):
			value, consumed, err := parseSearchContextArg(arg, idx, args)
			if err != nil {
				return opts, fmt.Sprintf("rg: %v", err)
			}
			opts.before = value
			idx += consumed
		case arg == "-C" || strings.HasPrefix(arg, "-C"):
			value, consumed, err := parseSearchContextArg(arg, idx, args)
			if err != nil {
				return opts, fmt.Sprintf("rg: %v", err)
			}
			opts.before = value
			opts.after = value
			idx += consumed
		case arg == "-g" || arg == "--glob":
			if idx+1 >= len(args) {
				return opts, fmt.Sprintf("rg: %s requires pattern", arg)
			}
			idx++
			opts.globs = append(opts.globs, args[idx])
		case strings.HasPrefix(arg, "--glob="):
			opts.globs = append(opts.globs, strings.TrimPrefix(arg, "--glob="))
		case strings.HasPrefix(arg, "-g") && len(arg) > 2:
			opts.globs = append(opts.globs, strings.TrimPrefix(strings.TrimPrefix(arg, "-g"), "="))
		default:
			return opts, fmt.Sprintf("rg: unsupported flag %s", arg)
		}
		idx++
	}
	if opts.filesOnly && opts.listFiles {
		return opts, "rg: -l is not supported with --files"
	}

	if opts.filesOnly {
		for idx < len(args) {
			target, err := requireAbsolutePath(args[idx])
			if err != nil {
				return opts, fmt.Sprintf("rg: %v", err)
			}
			opts.targets = append(opts.targets, target)
			idx++
		}
		if len(opts.targets) == 0 {
			opts.targets = []string{root}
		}
		return opts, validateRGGlobs(opts.globs)
	}

	if idx >= len(args) {
		return opts, "rg: missing pattern"
	}
	opts.pattern = args[idx]
	idx++
	for idx < len(args) {
		target, err := requireAbsolutePath(args[idx])
		if err != nil {
			return opts, fmt.Sprintf("rg: %v", err)
		}
		opts.targets = append(opts.targets, target)
		idx++
	}
	return opts, validateRGGlobs(opts.globs)
}

func validateRGGlobs(globs []string) string {
	for _, glob := range globs {
		if strings.TrimSpace(glob) == "" {
			return "rg: glob pattern must not be empty"
		}
		if strings.HasPrefix(glob, "!") {
			return "rg: negated glob patterns are not supported"
		}
	}
	return ""
}

func runRGListFiles(runtime engine.CommandRuntime, paths []string, match func(string) bool, jsonl bool) (string, int) {
	matched := make([]string, 0)
	for _, filePath := range paths {
		raw, err := runtime.Ops.ReadRawContent(runtime.Ctx, filePath)
		if err != nil {
			return fmt.Sprintf("rg: %v", err), contract.ExitCodeGeneral
		}
		if !searchHasMatch(raw, match) {
			continue
		}
		matched = append(matched, filePath)
	}
	if !jsonl {
		if len(matched) == 0 {
			return "", contract.ExitCodeGeneral
		}
		return strings.Join(matched, "\n"), 0
	}
	return renderSearchFileJSONL(matched), grepExitCode(len(matched) > 0)
}

func runRGText(runtime engine.CommandRuntime, paths []string, match func(string) bool, before int, after int) ([]string, error) {
	lines := make([]string, 0)
	for _, filePath := range paths {
		raw, err := runtime.Ops.ReadRawContent(runtime.Ctx, filePath)
		if err != nil {
			return nil, err
		}
		lines = append(lines, searchTextWithContext(raw, match, before, after, filePath)...)
	}
	return lines, nil
}

func runRGJSONL(runtime engine.CommandRuntime, paths []string, match func(string) bool, before int, after int) ([]searchRecord, error) {
	records := make([]searchRecord, 0)
	for _, filePath := range paths {
		raw, err := runtime.Ops.ReadRawContent(runtime.Ctx, filePath)
		if err != nil {
			return nil, err
		}
		records = append(records, searchRecordsWithContext(raw, match, before, after, filePath, false)...)
	}
	return records, nil
}

func resolveRGSearchPaths(runtime engine.CommandRuntime, targets []string, globs []string) ([]string, error) {
	if len(targets) == 0 {
		targets = []string{currentWorkingDir(runtime.Ops)}
	}
	seen := make(map[string]struct{}, len(targets))
	out := make([]string, 0)
	for _, target := range targets {
		paths, err := runtime.Ops.ResolveSearchPaths(runtime.Ctx, target, true)
		if err != nil {
			return nil, err
		}
		for _, filePath := range paths {
			matched, err := matchRGGlobs(filePath, globs)
			if err != nil {
				return nil, err
			}
			if !matched {
				continue
			}
			if _, exists := seen[filePath]; exists {
				continue
			}
			seen[filePath] = struct{}{}
			out = append(out, filePath)
		}
	}
	return out, nil
}

func matchRGGlobs(filePath string, globs []string) (bool, error) {
	if len(globs) == 0 {
		return true, nil
	}
	fullPath := strings.TrimPrefix(filePath, "/")
	baseName := path.Base(filePath)
	for _, glob := range globs {
		pattern := strings.TrimSpace(glob)
		if pattern == "" {
			continue
		}
		pattern = strings.TrimPrefix(pattern, "/")
		ok, err := path.Match(pattern, baseName)
		if err != nil {
			return false, fmt.Errorf("invalid glob pattern: %v", err)
		}
		if ok {
			return true, nil
		}
		ok, err = path.Match(pattern, fullPath)
		if err != nil {
			return false, fmt.Errorf("invalid glob pattern: %v", err)
		}
		if ok {
			return true, nil
		}
	}
	return false, nil
}
