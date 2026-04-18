package builtin

import (
	"context"
	"encoding/json"
	"fmt"
	"path"
	"sort"
	"strings"

	"github.com/khicago/simsh/pkg/contract"
	"github.com/khicago/simsh/pkg/engine"
)

func specGlob() engine.CommandSpec {
	return engine.CommandSpec{
		Name:   CommandGlob,
		Manual: "glob [--fmt jsonl] [--] PATTERN [PATH ...]",
		Tips: []string{
			"A pattern without a slash matches basenames recursively, so glob '*.go' finds every Go file under the target.",
			"Use ** in a path pattern for explicit directory wildcards.",
			"Use --fmt jsonl for path records without changing the default path-per-line stream.",
		},
		StructuredOutput: "flat path records",
		StructuredFlags:  []string{"--fmt jsonl"},
		Examples:         ExamplesFor("glob"),
		DetailedManual:   LoadEmbeddedManual("glob"),
		Run:              runGlob,
	}
}

type globArgs struct {
	pattern string
	targets []string
	jsonl   bool
}

type globRecord struct {
	Path string `json:"path"`
	Name string `json:"name"`
	Kind string `json:"kind"`
}

type compiledGlobPattern struct {
	parts []string
}

type globMatchState struct {
	rel int
	pat int
}

func runGlob(runtime engine.CommandRuntime, args []string) (string, int) {
	opts, errMsg := parseGlobArgs(args, runtime.Ops.RequireAbsolutePath, currentWorkingDir(runtime.Ops))
	if errMsg != "" {
		return errMsg, contract.ExitCodeUsage
	}
	matcher, err := compileGlobPattern(opts.pattern)
	if err != nil {
		return fmt.Sprintf("glob: invalid pattern: %v", err), contract.ExitCodeUsage
	}
	matched := make([]string, 0)
	seen := make(map[string]struct{})
	for _, target := range opts.targets {
		files, err := collectGlobFiles(runtime, target)
		if err != nil {
			if isPathMissing(err) || strings.Contains(err.Error(), "No such file") {
				return fmt.Sprintf("glob: %s: No such file or directory", target), contract.ExitCodeGeneral
			}
			return formatCommandPathError("glob", target, "", err), contract.ExitCodeGeneral
		}
		for _, filePath := range files {
			rel := relativeGlobPath(target, filePath)
			ok, matchErr := matcher.match(runtime.Ctx, rel)
			if matchErr != nil {
				return fmt.Sprintf("glob: %v", matchErr), contract.ExitCodeGeneral
			}
			if !ok {
				continue
			}
			if _, exists := seen[filePath]; exists {
				continue
			}
			seen[filePath] = struct{}{}
			matched = append(matched, filePath)
		}
	}
	sort.Strings(matched)
	if !opts.jsonl {
		return strings.Join(matched, "\n"), 0
	}
	return renderGlobJSONL(matched), 0
}

func parseGlobArgs(args []string, requireAbsolutePath func(string) (string, error), cwd string) (globArgs, string) {
	opts := globArgs{targets: make([]string, 0, 1)}
	positional := make([]string, 0, 2)
	parseOptions := true
	for idx := 0; idx < len(args); idx++ {
		arg := args[idx]
		if parseOptions && arg == "--" {
			parseOptions = false
			continue
		}
		if !parseOptions {
			positional = append(positional, arg)
			continue
		}
		switch arg {
		case "--fmt":
			if idx+1 >= len(args) {
				return opts, "glob: --fmt requires jsonl"
			}
			idx++
			if args[idx] != "jsonl" {
				return opts, fmt.Sprintf("glob: unsupported --fmt value %q", args[idx])
			}
			opts.jsonl = true
		case "--fmt=jsonl":
			opts.jsonl = true
		default:
			if strings.HasPrefix(arg, "--") {
				return opts, fmt.Sprintf("glob: unsupported flag %s", arg)
			}
			positional = append(positional, arg)
		}
	}
	if len(positional) == 0 {
		return opts, "glob: PATTERN is required"
	}
	opts.pattern = positional[0]
	if strings.TrimSpace(opts.pattern) == "" {
		return opts, "glob: PATTERN is required"
	}
	if len(positional) == 1 {
		pathValue, err := requireAbsolutePath(cwd)
		if err != nil {
			return opts, fmt.Sprintf("glob: %v", err)
		}
		opts.targets = []string{pathValue}
		return opts, ""
	}
	for _, raw := range positional[1:] {
		pathValue, err := requireAbsolutePath(raw)
		if err != nil {
			return opts, fmt.Sprintf("glob: %v", err)
		}
		opts.targets = append(opts.targets, pathValue)
	}
	return opts, ""
}

func collectGlobFiles(runtime engine.CommandRuntime, target string) ([]string, error) {
	isDir, err := runtime.Ops.IsDirPath(runtime.Ctx, target)
	if err != nil {
		return nil, err
	}
	if isDir {
		return runtime.Ops.CollectFilesUnder(runtime.Ctx, target)
	}
	_, err = runtime.Ops.ReadRawContent(runtime.Ctx, target)
	if err != nil {
		if isPathMissing(err) {
			return nil, fmt.Errorf("%s: No such file or directory", target)
		}
		return nil, err
	}
	return []string{target}, nil
}

func relativeGlobPath(root, filePath string) string {
	root = path.Clean(root)
	filePath = path.Clean(filePath)
	if root == "/" {
		return strings.TrimPrefix(filePath, "/")
	}
	if filePath == root {
		return path.Base(filePath)
	}
	prefix := root + "/"
	if strings.HasPrefix(filePath, prefix) {
		return strings.TrimPrefix(filePath, prefix)
	}
	return path.Base(filePath)
}

func matchGlobPattern(rel, pattern string) (bool, error) {
	compiled, err := compileGlobPattern(pattern)
	if err != nil {
		return false, err
	}
	return compiled.match(context.Background(), rel)
}

func compileGlobPattern(pattern string) (compiledGlobPattern, error) {
	pattern = strings.ReplaceAll(strings.TrimSpace(pattern), "\\", "/")
	pattern = strings.TrimPrefix(path.Clean("/"+pattern), "/")
	if pattern == "." || pattern == "" {
		return compiledGlobPattern{}, fmt.Errorf("pattern must not be empty")
	}
	if !strings.Contains(pattern, "/") {
		pattern = "**/" + pattern
	}
	parts := make([]string, 0, len(splitGlob(pattern)))
	for _, part := range splitGlob(pattern) {
		if part == "**" && len(parts) > 0 && parts[len(parts)-1] == "**" {
			continue
		}
		if part != "**" {
			if _, err := path.Match(part, ""); err != nil {
				return compiledGlobPattern{}, err
			}
		}
		parts = append(parts, part)
	}
	return compiledGlobPattern{parts: parts}, nil
}

func splitGlob(value string) []string {
	if value == "" {
		return nil
	}
	return strings.Split(value, "/")
}

func (pattern compiledGlobPattern) match(ctx context.Context, rel string) (bool, error) {
	rel = strings.TrimPrefix(path.Clean("/"+strings.TrimSpace(rel)), "/")
	relParts := splitGlob(rel)
	memo := make(map[globMatchState]bool)
	seen := make(map[globMatchState]struct{})

	var visit func(int, int) (bool, error)
	visit = func(relIdx, patIdx int) (bool, error) {
		if err := ctx.Err(); err != nil {
			return false, err
		}
		state := globMatchState{rel: relIdx, pat: patIdx}
		if _, ok := seen[state]; ok {
			return memo[state], nil
		}

		matched := false
		switch {
		case patIdx == len(pattern.parts):
			matched = relIdx == len(relParts)
		case pattern.parts[patIdx] == "**":
			var err error
			matched, err = visit(relIdx, patIdx+1)
			if err != nil {
				return false, err
			}
			if !matched && relIdx < len(relParts) {
				matched, err = visit(relIdx+1, patIdx)
				if err != nil {
					return false, err
				}
			}
		case relIdx < len(relParts):
			partMatched, err := path.Match(pattern.parts[patIdx], relParts[relIdx])
			if err != nil {
				return false, err
			}
			if partMatched {
				matched, err = visit(relIdx+1, patIdx+1)
				if err != nil {
					return false, err
				}
			}
		}

		seen[state] = struct{}{}
		memo[state] = matched
		return matched, nil
	}

	return visit(0, 0)
}

func renderGlobJSONL(paths []string) string {
	if len(paths) == 0 {
		return ""
	}
	lines := make([]string, 0, len(paths))
	for _, filePath := range paths {
		raw, err := json.Marshal(globRecord{
			Path: filePath,
			Name: path.Base(filePath),
			Kind: "file",
		})
		if err != nil {
			return fmt.Sprintf(`{"kind":"error","name":"json","path":%q}`, filePath)
		}
		lines = append(lines, string(raw))
	}
	return strings.Join(lines, "\n")
}
