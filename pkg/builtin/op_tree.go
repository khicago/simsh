package builtin

import (
	"encoding/json"
	"fmt"
	"path"
	"strings"

	"github.com/khicago/simsh/pkg/contract"
	"github.com/khicago/simsh/pkg/engine"
)

type treeOptions struct {
	includeHidden bool
	maxDepth      int
	targets       []string
	format        treeFormat
}

type treeFormat string

const (
	treeFormatOutline treeFormat = "outline"
	treeFormatASCII   treeFormat = "ascii"
	treeFormatJSON    treeFormat = "json"
)

type treeEntry struct {
	Path  string `json:"path"`
	Depth int    `json:"depth"`
	Kind  string `json:"kind"`
}

type treeJSONTarget struct {
	Root    string      `json:"root"`
	Entries []treeEntry `json:"entries"`
}

func specTree() engine.CommandSpec {
	return engine.CommandSpec{
		Name:   CommandTree,
		Manual: "tree [-a] [-L N] [--fmt outline|ascii|json] [PATH...]",
		Tips: []string{
			"Default output is an outline optimized for dual readability and low token noise.",
			"Use --fmt ascii for classic branch rendering or --fmt json for machine-readable entries.",
			"Use -L to limit output depth for large directories.",
			"Use -a to include hidden entries.",
		},
		DefaultOutput:    "directory outline",
		StructuredOutput: "tree entry records",
		StructuredFlags:  []string{"--fmt json", "--fmt ascii"},
		Examples:         ExamplesFor("tree"),
		DetailedManual:   LoadEmbeddedManual("tree"),
		Run:              runTree,
	}
}

func runTree(runtime engine.CommandRuntime, args []string) (string, int) {
	opts, errMsg := parseTreeArgs(args, runtime.Ops.RequireAbsolutePath, currentWorkingDir(runtime.Ops))
	if errMsg != "" {
		return errMsg, contract.ExitCodeUsage
	}
	if opts.format == treeFormatJSON {
		targets := make([]treeJSONTarget, 0, len(opts.targets))
		for _, target := range opts.targets {
			payload, out, code := buildTreeJSONTarget(runtime, target, opts.includeHidden, opts.maxDepth)
			if code != 0 {
				return out, code
			}
			targets = append(targets, payload)
		}
		raw, err := json.MarshalIndent(struct {
			Targets []treeJSONTarget `json:"targets"`
		}{
			Targets: targets,
		}, "", "  ")
		if err != nil {
			return fmt.Sprintf("tree: %v", err), contract.ExitCodeGeneral
		}
		return string(raw), 0
	}

	sections := make([]string, 0, len(opts.targets))
	for _, target := range opts.targets {
		section, code := renderTreeTarget(runtime, target, opts.includeHidden, opts.maxDepth, opts.format)
		if code != 0 {
			return section, code
		}
		sections = append(sections, section)
	}
	return strings.Join(sections, "\n\n"), 0
}

func parseTreeArgs(
	args []string,
	requireAbsolutePath func(string) (string, error),
	root string,
) (treeOptions, string) {
	opts := treeOptions{
		includeHidden: false,
		maxDepth:      -1,
		targets:       make([]string, 0, len(args)),
		format:        treeFormatOutline,
	}
	for idx := 0; idx < len(args); idx++ {
		arg := args[idx]
		switch {
		case arg == "-a":
			opts.includeHidden = true
		case arg == "-L":
			if idx+1 >= len(args) {
				return opts, "tree: -L requires a non-negative depth"
			}
			idx++
			depth, err := parseNonNegativeInt(args[idx])
			if err != nil {
				return opts, fmt.Sprintf("tree: invalid depth %q", args[idx])
			}
			opts.maxDepth = depth
		case strings.HasPrefix(arg, "-L="):
			rawDepth := strings.TrimSpace(strings.TrimPrefix(arg, "-L="))
			depth, err := parseNonNegativeInt(rawDepth)
			if err != nil {
				return opts, fmt.Sprintf("tree: invalid depth %q", rawDepth)
			}
			opts.maxDepth = depth
		case arg == "--fmt":
			if idx+1 >= len(args) {
				return opts, "tree: --fmt requires one value: outline|ascii|json"
			}
			idx++
			parsed, ok := parseTreeFormat(args[idx])
			if !ok {
				return opts, fmt.Sprintf("tree: unsupported --fmt value %q", args[idx])
			}
			opts.format = parsed
		case strings.HasPrefix(arg, "--fmt="):
			parsed, ok := parseTreeFormat(strings.TrimPrefix(arg, "--fmt="))
			if !ok {
				return opts, fmt.Sprintf("tree: unsupported --fmt value %q", strings.TrimPrefix(arg, "--fmt="))
			}
			opts.format = parsed
		case strings.HasPrefix(arg, "--"):
			return opts, fmt.Sprintf("tree: unsupported flag %s", arg)
		case strings.HasPrefix(arg, "-"):
			return opts, fmt.Sprintf("tree: unsupported flag %s", arg)
		default:
			pathValue, err := requireAbsolutePath(arg)
			if err != nil {
				return opts, fmt.Sprintf("tree: %v", err)
			}
			opts.targets = append(opts.targets, pathValue)
		}
	}
	if len(opts.targets) == 0 {
		opts.targets = append(opts.targets, root)
	}
	return opts, ""
}

func parseTreeFormat(raw string) (treeFormat, bool) {
	switch strings.TrimSpace(raw) {
	case string(treeFormatOutline), "":
		return treeFormatOutline, true
	case string(treeFormatASCII):
		return treeFormatASCII, true
	case string(treeFormatJSON):
		return treeFormatJSON, true
	default:
		return treeFormatOutline, false
	}
}

func renderTreeTarget(
	runtime engine.CommandRuntime,
	target string,
	includeHidden bool,
	maxDepth int,
	format treeFormat,
) (string, int) {
	isDir, err := runtime.Ops.IsDirPath(runtime.Ctx, target)
	if err != nil {
		return fmt.Sprintf("tree: %v", err), contract.ExitCodeGeneral
	}
	if !isDir {
		return target, 0
	}
	if format == treeFormatASCII {
		return renderTreeTargetASCII(runtime, target, includeHidden, maxDepth)
	}

	entries, err := collectTreeEntries(runtime, target, includeHidden, maxDepth)
	if err != nil {
		return fmt.Sprintf("tree: %v", err), contract.ExitCodeGeneral
	}
	return renderTreeOutline(entries), 0
}

func renderTreeTargetASCII(
	runtime engine.CommandRuntime,
	target string,
	includeHidden bool,
	maxDepth int,
) (string, int) {
	lines := []string{target}
	visited := map[string]struct{}{target: {}}
	if err := appendTreeChildrenASCII(runtime, target, "", 0, maxDepth, includeHidden, visited, &lines); err != nil {
		return fmt.Sprintf("tree: %v", err), contract.ExitCodeGeneral
	}
	return strings.Join(lines, "\n"), 0
}

func collectTreeEntries(
	runtime engine.CommandRuntime,
	target string,
	includeHidden bool,
	maxDepth int,
) ([]treeEntry, error) {
	entries := []treeEntry{{Path: target, Depth: 0, Kind: "dir"}}
	visited := map[string]struct{}{target: {}}
	if err := appendTreeEntries(runtime, target, 0, maxDepth, includeHidden, visited, &entries); err != nil {
		return nil, err
	}
	return entries, nil
}

func appendTreeEntries(
	runtime engine.CommandRuntime,
	dir string,
	depth int,
	maxDepth int,
	includeHidden bool,
	visited map[string]struct{},
	entries *[]treeEntry,
) error {
	if maxDepth >= 0 && depth >= maxDepth {
		return nil
	}
	children, err := runtime.Ops.ListChildren(runtime.Ctx, dir)
	if err != nil {
		return err
	}
	filtered := make([]string, 0, len(children))
	for _, child := range children {
		name := path.Base(child)
		if !includeHidden && strings.HasPrefix(name, ".") {
			continue
		}
		filtered = append(filtered, child)
	}
	for _, child := range filtered {
		isDir, err := runtime.Ops.IsDirPath(runtime.Ctx, child)
		if err != nil {
			return err
		}
		kind := "file"
		if isDir {
			kind = "dir"
		}
		*entries = append(*entries, treeEntry{Path: child, Depth: depth + 1, Kind: kind})
		if !isDir {
			continue
		}
		if _, seen := visited[child]; seen {
			continue
		}
		visited[child] = struct{}{}
		if err := appendTreeEntries(runtime, child, depth+1, maxDepth, includeHidden, visited, entries); err != nil {
			return err
		}
	}
	return nil
}

func renderTreeOutline(entries []treeEntry) string {
	if len(entries) == 0 {
		return ""
	}
	lines := make([]string, 0, len(entries))
	for idx, entry := range entries {
		if idx == 0 {
			lines = append(lines, appendTreeKindSuffix(entry.Path, entry.Kind))
			continue
		}
		name := appendTreeKindSuffix(path.Base(entry.Path), entry.Kind)
		lines = append(lines, strings.Repeat("  ", entry.Depth)+name)
	}
	return strings.Join(lines, "\n")
}

func buildTreeJSONTarget(
	runtime engine.CommandRuntime,
	target string,
	includeHidden bool,
	maxDepth int,
) (treeJSONTarget, string, int) {
	isDir, err := runtime.Ops.IsDirPath(runtime.Ctx, target)
	if err != nil {
		return treeJSONTarget{}, fmt.Sprintf("tree: %v", err), contract.ExitCodeGeneral
	}
	if !isDir {
		return treeJSONTarget{
			Root: target,
			Entries: []treeEntry{
				{Path: target, Depth: 0, Kind: "file"},
			},
		}, "", 0
	}
	entries, err := collectTreeEntries(runtime, target, includeHidden, maxDepth)
	if err != nil {
		return treeJSONTarget{}, fmt.Sprintf("tree: %v", err), contract.ExitCodeGeneral
	}
	return treeJSONTarget{Root: target, Entries: entries}, "", 0
}

func appendTreeKindSuffix(name string, kind string) string {
	if kind == "dir" && !strings.HasSuffix(name, "/") {
		return name + "/"
	}
	return name
}

func appendTreeChildrenASCII(
	runtime engine.CommandRuntime,
	dir string,
	prefix string,
	depth int,
	maxDepth int,
	includeHidden bool,
	visited map[string]struct{},
	lines *[]string,
) error {
	if maxDepth >= 0 && depth >= maxDepth {
		return nil
	}
	children, err := runtime.Ops.ListChildren(runtime.Ctx, dir)
	if err != nil {
		return err
	}
	filtered := make([]string, 0, len(children))
	for _, child := range children {
		name := path.Base(child)
		if !includeHidden && strings.HasPrefix(name, ".") {
			continue
		}
		filtered = append(filtered, child)
	}

	for idx, child := range filtered {
		name := path.Base(child)
		last := idx == len(filtered)-1
		branch := "|-- "
		nextPrefix := prefix + "|   "
		if last {
			branch = "`-- "
			nextPrefix = prefix + "    "
		}

		isDir, err := runtime.Ops.IsDirPath(runtime.Ctx, child)
		if err != nil {
			return err
		}
		displayName := name
		if isDir {
			displayName = appendTreeKindSuffix(displayName, "dir")
		}
		*lines = append(*lines, prefix+branch+displayName)
		if !isDir {
			continue
		}
		if _, seen := visited[child]; seen {
			continue
		}
		visited[child] = struct{}{}
		if err := appendTreeChildrenASCII(runtime, child, nextPrefix, depth+1, maxDepth, includeHidden, visited, lines); err != nil {
			return err
		}
	}
	return nil
}
