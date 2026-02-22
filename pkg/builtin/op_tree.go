package builtin

import (
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
}

func specTree() engine.CommandSpec {
	return engine.CommandSpec{
		Name:   CommandTree,
		Manual: "tree [-a] [-L N] [ABS_PATH...]",
		Tips: []string{
			"Use -L to limit output depth for large directories.",
			"Use -a to include hidden entries.",
		},
		Examples:       ExamplesFor("tree"),
		DetailedManual: LoadEmbeddedManual("tree"),
		Run:            runTree,
	}
}

func runTree(runtime engine.CommandRuntime, args []string) (string, int) {
	opts, errMsg := parseTreeArgs(args, runtime.Ops.RequireAbsolutePath, runtime.Ops.RootDir)
	if errMsg != "" {
		return errMsg, contract.ExitCodeUsage
	}

	sections := make([]string, 0, len(opts.targets))
	for _, target := range opts.targets {
		section, code := renderTreeTarget(runtime, target, opts.includeHidden, opts.maxDepth)
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

func renderTreeTarget(
	runtime engine.CommandRuntime,
	target string,
	includeHidden bool,
	maxDepth int,
) (string, int) {
	isDir, err := runtime.Ops.IsDirPath(runtime.Ctx, target)
	if err != nil {
		return fmt.Sprintf("tree: %v", err), contract.ExitCodeGeneral
	}
	if !isDir {
		return target, 0
	}

	lines := []string{target}
	visited := map[string]struct{}{target: {}}
	if err := appendTreeChildren(runtime, target, "", 0, maxDepth, includeHidden, visited, &lines); err != nil {
		return fmt.Sprintf("tree: %v", err), contract.ExitCodeGeneral
	}
	return strings.Join(lines, "\n"), 0
}

func appendTreeChildren(
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
		*lines = append(*lines, prefix+branch+name)

		isDir, err := runtime.Ops.IsDirPath(runtime.Ctx, child)
		if err != nil {
			return err
		}
		if !isDir {
			continue
		}
		if _, seen := visited[child]; seen {
			continue
		}
		visited[child] = struct{}{}
		if err := appendTreeChildren(runtime, child, nextPrefix, depth+1, maxDepth, includeHidden, visited, lines); err != nil {
			return err
		}
	}
	return nil
}
