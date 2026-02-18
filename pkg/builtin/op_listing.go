package builtin

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/khicago/simsh/pkg/contract"
	"github.com/khicago/simsh/pkg/engine"
)

func specLS() engine.CommandSpec {
	return engine.CommandSpec{
		Name:   CommandLS,
		Manual: "ls [-a] [-R] [-l] [--fmt text|md|json] [ABS_PATH...]",
		Tips: []string{
			"Use -l to include semantic metadata.",
			"Use --fmt md|json only with -l and a single non-recursive target.",
			"Use -R for recursive traversal.",
		},
		Examples:       ExamplesFor("ls"),
		DetailedManual: LoadEmbeddedManual("ls"),
		Run:            runLS,
	}
}

type lsLongFormat string

const (
	lsLongFormatText     lsLongFormat = "text"
	lsLongFormatMarkdown lsLongFormat = "md"
	lsLongFormatJSON     lsLongFormat = "json"
)

func runLS(runtime engine.CommandRuntime, args []string) (string, int) {
	targets := make([]string, 0, 1)
	includeDots := false
	recursive := false
	longFormat := false
	longFormatStyle := lsLongFormatText

	for idx := 0; idx < len(args); idx++ {
		arg := args[idx]
		if arg == "--fmt" {
			if idx+1 >= len(args) {
				return "ls: --fmt requires one value: text|md|json", contract.ExitCodeUsage
			}
			idx++
			v, ok := parseLSFormat(args[idx])
			if !ok {
				return fmt.Sprintf("ls: unsupported --fmt value %q", args[idx]), contract.ExitCodeUsage
			}
			longFormatStyle = v
			continue
		}
		if strings.HasPrefix(arg, "--fmt=") {
			v, ok := parseLSFormat(strings.TrimPrefix(arg, "--fmt="))
			if !ok {
				return fmt.Sprintf("ls: unsupported --fmt value %q", strings.TrimPrefix(arg, "--fmt=")), contract.ExitCodeUsage
			}
			longFormatStyle = v
			continue
		}
		if strings.HasPrefix(arg, "--") {
			return fmt.Sprintf("ls: unsupported flag %s", arg), contract.ExitCodeUsage
		}
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

	if longFormatStyle != lsLongFormatText && !longFormat {
		return "ls: --fmt is only supported with -l", contract.ExitCodeUsage
	}
	if longFormatStyle != lsLongFormatText && recursive {
		return "ls: --fmt md|json does not support -R", contract.ExitCodeUsage
	}
	if len(targets) == 0 {
		targets = append(targets, runtime.Ops.RootDir)
	}
	if longFormatStyle != lsLongFormatText && len(targets) > 1 {
		return "ls: --fmt md|json supports one target only", contract.ExitCodeUsage
	}

	sections := make([]string, 0, len(targets))
	for _, target := range targets {
		out, code := runLSTarget(runtime, target, includeDots, recursive, longFormat, longFormatStyle)
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
	out := strings.Join(sections, sep)
	if longFormat && longFormatStyle == lsLongFormatText {
		out = strings.TrimRight(out, "\n")
		if out != "" {
			out += "\n"
		}
		out += "# columns: mode access kind lines path"
	}
	return out, 0
}

func parseLSFormat(raw string) (lsLongFormat, bool) {
	switch strings.TrimSpace(raw) {
	case string(lsLongFormatText):
		return lsLongFormatText, true
	case string(lsLongFormatMarkdown):
		return lsLongFormatMarkdown, true
	case string(lsLongFormatJSON):
		return lsLongFormatJSON, true
	default:
		return lsLongFormatText, false
	}
}

func runLSTarget(runtime engine.CommandRuntime, target string, includeDots bool, recursive bool, longFormat bool, longFormatStyle lsLongFormat) (string, int) {
	isDir, err := runtime.Ops.IsDirPath(runtime.Ctx, target)
	if err != nil {
		return fmt.Sprintf("ls: %v", err), contract.ExitCodeGeneral
	}
	if !isDir {
		if longFormat {
			return formatLongRows([]contract.LSLongRow{buildLongRow(runtime, target, target)}, longFormatStyle), 0
		}
		return target, 0
	}
	if !recursive {
		children, err := runtime.Ops.ListChildren(runtime.Ctx, target)
		if err != nil {
			return fmt.Sprintf("ls: %v", err), contract.ExitCodeGeneral
		}
		return renderLSOutput(runtime, target, children, includeDots, longFormat, longFormatStyle)
	}
	return runLSRecursive(runtime, target, includeDots, longFormat, longFormatStyle)
}

func runLSRecursive(runtime engine.CommandRuntime, target string, includeDots bool, longFormat bool, longFormatStyle lsLongFormat) (string, int) {
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
		section, code := renderLSOutput(runtime, dir, children, includeDots, longFormat, longFormatStyle)
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

func renderLSOutput(runtime engine.CommandRuntime, target string, children []string, includeDots bool, longFormat bool, longFormatStyle lsLongFormat) (string, int) {
	if !longFormat {
		lines := make([]string, 0, len(children)+2)
		if includeDots {
			lines = append(lines, ".", "..")
		}
		lines = append(lines, children...)
		return strings.Join(lines, "\n"), 0
	}
	rows := make([]contract.LSLongRow, 0, len(children)+2)
	if includeDots {
		rows = append(rows, buildLongRow(runtime, ".", target))
		parent := runtime.Ops.RootDir
		if target != runtime.Ops.RootDir {
			parent = parentDir(target)
		}
		rows = append(rows, buildLongRow(runtime, "..", parent))
	}
	for _, child := range children {
		rows = append(rows, buildLongRow(runtime, child, child))
	}
	return formatLongRows(rows, longFormatStyle), 0
}

func formatLongRows(rows []contract.LSLongRow, style lsLongFormat) string {
	switch style {
	case lsLongFormatMarkdown:
		return formatMarkdownLongRows(rows)
	case lsLongFormatJSON:
		return formatJSONLongRows(rows)
	default:
		out := make([]string, 0, len(rows))
		for _, row := range rows {
			out = append(out, formatDefaultLongRow(row))
		}
		return strings.Join(out, "\n")
	}
}

func formatLongRow(runtime engine.CommandRuntime, displayPath string, absPath string, style lsLongFormat) string {
	row := buildLongRow(runtime, displayPath, absPath)
	if runtime.Ops.FormatLSLongRow != nil {
		if formatted, handled := runtime.Ops.FormatLSLongRow(runtime.Ctx, row); handled {
			return formatted
		}
	}
	if style != lsLongFormatText {
		return formatLongRows([]contract.LSLongRow{row}, style)
	}
	return formatDefaultLongRow(row)
}

func buildLongRow(runtime engine.CommandRuntime, displayPath string, absPath string) contract.LSLongRow {
	meta := contract.PathMeta{
		Exists:           false,
		IsDir:            false,
		Kind:             "unknown",
		Access:           contract.PathAccessReadOnly,
		Capabilities:     []string{contract.PathCapabilityDescribe},
		LineCount:        -1,
		FrontMatterLines: -1,
		SpeakerRows:      -1,
		UserRelevance:    "unknown",
	}
	if runtime.Ops.DescribePath != nil {
		if described, err := runtime.Ops.DescribePath(runtime.Ctx, absPath); err == nil {
			meta = described
		}
	} else {
		if isDir, err := runtime.Ops.IsDirPath(runtime.Ctx, absPath); err == nil {
			meta = contract.PathMeta{
				Exists:           true,
				IsDir:            isDir,
				Kind:             ternary(isDir, "dir", "file"),
				Access:           contract.PathAccessReadOnly,
				Capabilities:     []string{contract.PathCapabilityDescribe},
				LineCount:        -1,
				FrontMatterLines: -1,
				SpeakerRows:      -1,
				UserRelevance:    "unknown",
			}
		}
	}

	// Fill defaults for older adapters that don't emit SSOT access/capabilities yet.
	if strings.TrimSpace(meta.Access) == "" {
		meta.Access = contract.PathAccessReadOnly
		if runtime.Ops.Policy.AllowWrite() {
			meta.Access = contract.PathAccessReadWrite
		}
	}
	meta.Access = contract.NormalizePathAccess(meta.Access)
	if len(meta.Capabilities) == 0 {
		if meta.IsDir {
			meta.Capabilities = []string{contract.PathCapabilityDescribe, contract.PathCapabilityList, contract.PathCapabilitySearch}
		} else {
			meta.Capabilities = []string{contract.PathCapabilityDescribe, contract.PathCapabilityRead}
		}
	}
	meta.Capabilities = contract.NormalizePathCapabilities(meta.Capabilities)
	if meta.Access == contract.PathAccessReadOnly {
		meta.Capabilities = contract.StripWriteCapabilities(meta.Capabilities)
	}
	return contract.LSLongRow{
		DisplayPath:      displayPath,
		Path:             absPath,
		Exists:           meta.Exists,
		IsDir:            meta.IsDir,
		Kind:             meta.Kind,
		Access:           meta.Access,
		Capabilities:     meta.Capabilities,
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
	access := contract.NormalizePathAccess(strings.TrimSpace(row.Access))
	return fmt.Sprintf("%s %s %s %s %s", mode, access, kind, dashIfNegative(row.LineCount), row.DisplayPath)
}

func formatMarkdownLongRows(rows []contract.LSLongRow) string {
	lines := []string{
		"| mode | access | kind | lines | path |",
		"|---|---|---|---:|---|",
	}
	for _, row := range rows {
		lines = append(lines, fmt.Sprintf("| %s | %s | %s | %s | %s |",
			longRowMode(row),
			contract.NormalizePathAccess(strings.TrimSpace(row.Access)),
			longRowKind(row),
			dashIfNegative(row.LineCount),
			strings.ReplaceAll(row.DisplayPath, "|", "\\|"),
		))
	}
	return strings.Join(lines, "\n")
}

type lsJSONRow struct {
	Mode         string   `json:"mode"`
	Access       string   `json:"access"`
	Kind         string   `json:"kind"`
	Lines        int      `json:"lines"`
	Path         string   `json:"path"`
	Capabilities []string `json:"capabilities"`
}

func formatJSONLongRows(rows []contract.LSLongRow) string {
	type payload struct {
		Columns []string    `json:"columns"`
		Entries []lsJSONRow `json:"entries"`
	}
	resp := payload{
		Columns: []string{"mode", "access", "kind", "lines", "path"},
		Entries: make([]lsJSONRow, 0, len(rows)),
	}
	for _, row := range rows {
		resp.Entries = append(resp.Entries, lsJSONRow{
			Mode:         longRowMode(row),
			Access:       contract.NormalizePathAccess(strings.TrimSpace(row.Access)),
			Kind:         longRowKind(row),
			Lines:        row.LineCount,
			Path:         row.DisplayPath,
			Capabilities: contract.NormalizePathCapabilities(row.Capabilities),
		})
	}
	raw, err := json.Marshal(resp)
	if err != nil {
		return "{}"
	}
	return string(raw)
}

func longRowMode(row contract.LSLongRow) string {
	if row.IsDir {
		return "d"
	}
	return "-"
}

func longRowKind(row contract.LSLongRow) string {
	kind := strings.TrimSpace(row.Kind)
	if kind != "" {
		return kind
	}
	if row.IsDir {
		return "dir"
	}
	return "file"
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
