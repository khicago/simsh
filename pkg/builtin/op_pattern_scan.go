package builtin

import (
	"fmt"
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
	jsonl     bool
	before    int
	after     int
}

func specGrep() engine.CommandSpec {
	return engine.CommandSpec{
		Name:   CommandGrep,
		Manual: "grep [-E|-F] [-r] [-l] [-A N] [-B N] [-C N] [--fmt jsonl] PATTERN [PATH]",
		Tips: []string{
			"Use -r for directory search and -l to list matched files only.",
			"Use --fmt jsonl when you want flat machine-readable records without changing the default text output.",
			"Context flags -A/-B/-C include neighboring lines around each match.",
		},
		StructuredOutput: "flat search records",
		StructuredFlags:  []string{"--fmt jsonl"},
		Examples:         ExamplesFor("grep"),
		DetailedManual:   LoadEmbeddedManual("grep"),
		Run:              runGrep,
	}
}

func runGrep(runtime engine.CommandRuntime, args []string) (string, int) {
	opts, errMsg := parseGrepArgs(args, runtime.Ops.RequireAbsolutePath)
	if errMsg != "" {
		return errMsg, contract.ExitCodeUsage
	}
	match, err := buildSearchMatcher(opts.pattern, searchMatcherOptions{Regex: opts.regex})
	if err != nil {
		return fmt.Sprintf("grep: %v", err), contract.ExitCodeUsage
	}
	if runtime.HasStdin && opts.path == "" {
		if opts.listFiles {
			found := searchHasMatch(runtime.Stdin, match)
			if !opts.jsonl {
				if found {
					return "(stdin)", 0
				}
				return "", contract.ExitCodeGeneral
			}
			records := []searchRecord{}
			if found {
				records = append(records, searchRecord{Stdin: true, Line: 0, Kind: "file"})
			}
			return renderSearchJSONL(records), grepExitCode(found)
		}
		if !opts.jsonl {
			out := searchTextWithContext(runtime.Stdin, match, opts.before, opts.after, "")
			if len(out) == 0 {
				return "", contract.ExitCodeGeneral
			}
			return strings.Join(out, "\n"), 0
		}
		records := searchRecordsWithContext(runtime.Stdin, match, opts.before, opts.after, "", true)
		return renderSearchJSONL(records), grepExitCode(len(records) > 0)
	}
	if !runtime.HasStdin && opts.path == "" {
		return "grep: expected stdin input or one file/directory path", contract.ExitCodeUsage
	}
	target := opts.path
	if target != "" && shouldUseRuntimeSearch(runtime, target, opts.recursive) {
		req := buildContractSearchRequest(opts.pattern, searchMatcherOptions{
			Regex:    opts.regex,
			CaseMode: searchCaseSensitive,
		}, nil, []string{target}, opts.listFiles, opts.before, opts.after, 0)
		if used, result, err := tryRuntimeSearch(runtime, req); err != nil {
			return fmt.Sprintf("grep: %v", err), contract.ExitCodeGeneral
		} else if used {
			return renderSearchRecords(result.Records, opts.listFiles, opts.jsonl, false)
		}
	}
	paths, err := runtime.Ops.ResolveSearchPaths(runtime.Ctx, target, opts.recursive)
	if err != nil {
		return fmt.Sprintf("grep: %v", err), contract.ExitCodeGeneral
	}

	if opts.listFiles {
		matchedPaths := make([]string, 0)
		records := make([]searchRecord, 0)
		for _, filePath := range paths {
			raw, err := runtime.Ops.ReadRawContent(runtime.Ctx, filePath)
			if err != nil {
				return fmt.Sprintf("grep: %v", err), contract.ExitCodeGeneral
			}
			if !searchHasMatch(raw, match) {
				continue
			}
			matchedPaths = append(matchedPaths, filePath)
			if opts.jsonl {
				records = append(records, searchRecord{Path: filePath, Line: 0, Kind: "file"})
			}
		}
		if !opts.jsonl {
			if len(matchedPaths) == 0 {
				return "", contract.ExitCodeGeneral
			}
			return strings.Join(matchedPaths, "\n"), 0
		}
		return renderSearchJSONL(records), grepExitCode(len(records) > 0)
	}

	if !opts.jsonl {
		lines := make([]string, 0)
		for _, filePath := range paths {
			raw, err := runtime.Ops.ReadRawContent(runtime.Ctx, filePath)
			if err != nil {
				return fmt.Sprintf("grep: %v", err), contract.ExitCodeGeneral
			}
			lines = append(lines, searchTextWithContext(raw, match, opts.before, opts.after, filePath)...)
		}
		if len(lines) == 0 {
			return "", contract.ExitCodeGeneral
		}
		return strings.Join(lines, "\n"), 0
	}

	records := make([]searchRecord, 0)
	for _, filePath := range paths {
		raw, err := runtime.Ops.ReadRawContent(runtime.Ctx, filePath)
		if err != nil {
			return fmt.Sprintf("grep: %v", err), contract.ExitCodeGeneral
		}
		records = append(records, searchRecordsWithContext(raw, match, opts.before, opts.after, filePath, false)...)
	}
	return renderSearchJSONL(records), grepExitCode(len(records) > 0)
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
		case arg == "--fmt":
			if idx+1 >= len(args) {
				return opts, "grep: --fmt requires one value: jsonl"
			}
			idx++
			if strings.TrimSpace(args[idx]) != "jsonl" {
				return opts, fmt.Sprintf("grep: unsupported --fmt value %q", args[idx])
			}
			opts.jsonl = true
		case strings.HasPrefix(arg, "--fmt="):
			if strings.TrimSpace(strings.TrimPrefix(arg, "--fmt=")) != "jsonl" {
				return opts, fmt.Sprintf("grep: unsupported --fmt value %q", strings.TrimPrefix(arg, "--fmt="))
			}
			opts.jsonl = true
		case arg == "-E":
			opts.regex = true
		case arg == "-F":
			opts.regex = false
		case arg == "-r":
			opts.recursive = true
		case arg == "-l":
			opts.listFiles = true
		case arg == "-A" || strings.HasPrefix(arg, "-A"):
			value, consumed, err := parseSearchContextArg(arg, idx, args)
			if err != nil {
				return opts, fmt.Sprintf("grep: %v", err)
			}
			opts.after = value
			idx += consumed
		case arg == "-B" || strings.HasPrefix(arg, "-B"):
			value, consumed, err := parseSearchContextArg(arg, idx, args)
			if err != nil {
				return opts, fmt.Sprintf("grep: %v", err)
			}
			opts.before = value
			idx += consumed
		case arg == "-C" || strings.HasPrefix(arg, "-C"):
			value, consumed, err := parseSearchContextArg(arg, idx, args)
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

func grepExitCode(found bool) int {
	if found {
		return 0
	}
	return contract.ExitCodeGeneral
}

func shouldUseRuntimeSearch(runtime engine.CommandRuntime, target string, recursive bool) bool {
	if recursive {
		return true
	}
	if runtime.Ops.IsDirPath == nil {
		return false
	}
	isDir, err := runtime.Ops.IsDirPath(runtime.Ctx, target)
	if err != nil {
		return false
	}
	return !isDir
}
