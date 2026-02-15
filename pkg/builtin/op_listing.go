package builtin

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/khicago/simsh/pkg/contract"
	"github.com/khicago/simsh/pkg/engine"
)

func specLS() engine.CommandSpec {
	return engine.CommandSpec{
		Name:   CommandLS,
		Manual: "ls [-a] [-R] [-l] [ABS_PATH...]",
		Tips: []string{
			"Use -l to include semantic metadata.",
			"Use -R for recursive traversal.",
		},
		Examples:       ExamplesFor("ls"),
		DetailedManual: LoadEmbeddedManual("ls"),
		Run:            runLS,
	}
}

func runLS(runtime engine.CommandRuntime, args []string) (string, int) {
	targets := make([]string, 0, 1)
	includeDots := false
	recursive := false
	longFormat := false
	for _, arg := range args {
		if strings.HasPrefix(arg, "-") && arg != "-" {
			for i := 1; i < len(arg); i++ {
				switch arg[i] {
				case 'a':
					includeDots = true
				case 'R':
					recursive = true
				case 'l':
					longFormat = true
				default:
					return fmt.Sprintf("ls: unsupported flag -%c", arg[i]), contract.ExitCodeUsage
				}
			}
			continue
		}
		pathValue, err := runtime.Ops.RequireAbsolutePath(arg)
		if err != nil {
			return fmt.Sprintf("ls: %v", err), contract.ExitCodeUsage
		}
		targets = append(targets, pathValue)
	}
	if len(targets) == 0 {
		targets = append(targets, runtime.Ops.RootDir)
	}
	sections := make([]string, 0, len(targets))
	for _, target := range targets {
		out, code := runLSTarget(runtime, target, includeDots, recursive, longFormat)
		if code != 0 {
			return out, code
		}
		if len(targets) > 1 && !recursive {
			sections = append(sections, target+":\n"+out)
			continue
		}
		sections = append(sections, out)
	}
	sep := "\n"
	if len(targets) > 1 {
		sep = "\n\n"
	}
	return strings.Join(sections, sep), 0
}

func runLSTarget(runtime engine.CommandRuntime, target string, includeDots bool, recursive bool, longFormat bool) (string, int) {
	isDir, err := runtime.Ops.IsDirPath(runtime.Ctx, target)
	if err != nil {
		return fmt.Sprintf("ls: %v", err), contract.ExitCodeGeneral
	}
	if !isDir {
		if longFormat {
			return formatLongRow(runtime, target, target), 0
		}
		return target, 0
	}
	if !recursive {
		children, err := runtime.Ops.ListChildren(runtime.Ctx, target)
		if err != nil {
			return fmt.Sprintf("ls: %v", err), contract.ExitCodeGeneral
		}
		return renderLSOutput(runtime, target, children, includeDots, longFormat)
	}
	return runLSRecursive(runtime, target, includeDots, longFormat)
}

func runLSRecursive(runtime engine.CommandRuntime, target string, includeDots bool, longFormat bool) (string, int) {
	visited := map[string]struct{}{}
	queue := []string{target}
	out := make([]string, 0)
	for len(queue) > 0 {
		dir := queue[0]
		queue = queue[1:]
		if _, ok := visited[dir]; ok {
			continue
		}
		visited[dir] = struct{}{}
		children, err := runtime.Ops.ListChildren(runtime.Ctx, dir)
		if err != nil {
			return fmt.Sprintf("ls: %v", err), contract.ExitCodeGeneral
		}
		out = append(out, dir+":")
		section, code := renderLSOutput(runtime, dir, children, includeDots, longFormat)
		if code != 0 {
			return section, code
		}
		if strings.TrimSpace(section) != "" {
			out = append(out, strings.Split(section, "\n")...)
		}
		for _, child := range children {
			isDir, err := runtime.Ops.IsDirPath(runtime.Ctx, child)
			if err != nil {
				return fmt.Sprintf("ls: %v", err), contract.ExitCodeGeneral
			}
			if isDir {
				queue = append(queue, child)
			}
		}
		if len(queue) > 0 {
			out = append(out, "")
		}
	}
	return strings.Join(out, "\n"), 0
}

func renderLSOutput(runtime engine.CommandRuntime, target string, children []string, includeDots bool, longFormat bool) (string, int) {
	if !longFormat {
		lines := make([]string, 0, len(children)+2)
		if includeDots {
			lines = append(lines, ".", "..")
		}
		lines = append(lines, children...)
		return strings.Join(lines, "\n"), 0
	}
	rows := make([]string, 0, len(children)+2)
	if includeDots {
		rows = append(rows, formatLongRow(runtime, ".", target))
		parent := runtime.Ops.RootDir
		if target != runtime.Ops.RootDir {
			parent = parentDir(target)
		}
		rows = append(rows, formatLongRow(runtime, "..", parent))
	}
	for _, child := range children {
		rows = append(rows, formatLongRow(runtime, child, child))
	}
	return strings.Join(rows, "\n"), 0
}

func formatLongRow(runtime engine.CommandRuntime, displayPath string, absPath string) string {
	row := buildLongRow(runtime, displayPath, absPath)
	if runtime.Ops.FormatLSLongRow != nil {
		if formatted, handled := runtime.Ops.FormatLSLongRow(runtime.Ctx, row); handled {
			return formatted
		}
	}
	return formatDefaultLongRow(row)
}

func buildLongRow(runtime engine.CommandRuntime, displayPath string, absPath string) contract.LSLongRow {
	meta := contract.PathMeta{Exists: false, IsDir: false, Kind: "unknown", LineCount: -1, FrontMatterLines: -1, SpeakerRows: -1, UserRelevance: "unknown"}
	if runtime.Ops.DescribePath != nil {
		if described, err := runtime.Ops.DescribePath(runtime.Ctx, absPath); err == nil {
			meta = described
		}
	} else {
		if isDir, err := runtime.Ops.IsDirPath(runtime.Ctx, absPath); err == nil {
			meta = contract.PathMeta{Exists: true, IsDir: isDir, Kind: ternary(isDir, "dir", "file"), LineCount: -1, FrontMatterLines: -1, SpeakerRows: -1, UserRelevance: "unknown"}
		}
	}
	return contract.LSLongRow{
		DisplayPath:      displayPath,
		Path:             absPath,
		Exists:           meta.Exists,
		IsDir:            meta.IsDir,
		Kind:             meta.Kind,
		LineCount:        meta.LineCount,
		FrontMatterLines: meta.FrontMatterLines,
		SpeakerRows:      meta.SpeakerRows,
		UserRelevance:    meta.UserRelevance,
	}
}

func formatDefaultLongRow(row contract.LSLongRow) string {
	kind := strings.TrimSpace(row.Kind)
	if kind == "" {
		if row.IsDir {
			kind = "dir"
		} else {
			kind = "file"
		}
	}
	mode := "-"
	if row.IsDir {
		mode = "d"
	}
	return fmt.Sprintf("%s kind=%s lines=%s %s", mode, kind, dashIfNegative(row.LineCount), row.DisplayPath)
}

func dashIfNegative(v int) string {
	if v < 0 {
		return "-"
	}
	return strconv.Itoa(v)
}

func parentDir(pathValue string) string {
	if pathValue == "/" {
		return "/"
	}
	idx := strings.LastIndex(pathValue, "/")
	if idx <= 0 {
		return "/"
	}
	return pathValue[:idx]
}

func ternary(cond bool, a string, b string) string {
	if cond {
		return a
	}
	return b
}
