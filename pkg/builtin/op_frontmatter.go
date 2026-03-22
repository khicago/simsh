package builtin

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/khicago/simsh/pkg/contract"
	"github.com/khicago/simsh/pkg/engine"
)

type frontmatterFormat string

const (
	frontmatterFormatCompact frontmatterFormat = "compact"
	frontmatterFormatJSON    frontmatterFormat = "json"
	frontmatterFormatMD      frontmatterFormat = "md"
)

type frontmatterStatRow struct {
	Path           string   `json:"path"`
	HasFrontMatter bool     `json:"has_frontmatter"`
	StartLine      int      `json:"start_line"`
	EndLine        int      `json:"end_line"`
	KeyCount       int      `json:"key_count"`
	Keys           []string `json:"keys"`
}

func specFrontmatter() engine.CommandSpec {
	return engine.CommandSpec{
		Name:   CommandFrontmatter,
		Manual: "frontmatter <stat|get|print> ...",
		Tips: []string{
			"Use stat to inspect frontmatter presence across many files.",
			"Use get --key KEY to read a specific top-level key value.",
			"Use print -C N to print key/frontmatter with line context.",
		},
		Examples:       ExamplesFor("frontmatter"),
		DetailedManual: LoadEmbeddedManual("frontmatter"),
		Run:            runFrontmatter,
	}
}

func runFrontmatter(runtime engine.CommandRuntime, args []string) (string, int) {
	if len(args) == 0 {
		return "frontmatter: expected subcommand: stat|get|print", contract.ExitCodeUsage
	}
	sub := strings.TrimSpace(args[0])
	switch sub {
	case "stat":
		return runFrontmatterStat(runtime, args[1:])
	case "get":
		return runFrontmatterGet(runtime, args[1:])
	case "print":
		return runFrontmatterPrint(runtime, args[1:])
	default:
		return fmt.Sprintf("frontmatter: unsupported subcommand %s", sub), contract.ExitCodeUsage
	}
}

func runFrontmatterStat(runtime engine.CommandRuntime, args []string) (string, int) {
	recursive := false
	format := frontmatterFormatCompact
	targets := make([]string, 0, len(args))

	for idx := 0; idx < len(args); idx++ {
		arg := args[idx]
		switch {
		case arg == "-r":
			recursive = true
		case arg == "--fmt":
			if idx+1 >= len(args) {
				return "frontmatter stat: --fmt requires one value: compact|json|md", contract.ExitCodeUsage
			}
			idx++
			parsed, ok := parseFrontmatterFormat(args[idx])
			if !ok {
				return fmt.Sprintf("frontmatter stat: unsupported --fmt value %q", args[idx]), contract.ExitCodeUsage
			}
			format = parsed
		case strings.HasPrefix(arg, "--fmt="):
			parsed, ok := parseFrontmatterFormat(strings.TrimPrefix(arg, "--fmt="))
			if !ok {
				return fmt.Sprintf("frontmatter stat: unsupported --fmt value %q", strings.TrimPrefix(arg, "--fmt=")), contract.ExitCodeUsage
			}
			format = parsed
		case strings.HasPrefix(arg, "-"):
			return fmt.Sprintf("frontmatter stat: unsupported flag %s", arg), contract.ExitCodeUsage
		default:
			pathValue, err := runtime.Ops.RequireAbsolutePath(arg)
			if err != nil {
				return fmt.Sprintf("frontmatter stat: %v", err), contract.ExitCodeUsage
			}
			targets = append(targets, pathValue)
		}
	}

	if len(targets) == 0 {
		return "frontmatter stat: expected at least one path", contract.ExitCodeUsage
	}

	files, out, code := expandFrontmatterTargets(runtime, "frontmatter stat", targets, recursive)
	if code != 0 {
		return out, code
	}

	rows := make([]frontmatterStatRow, 0, len(files))
	for _, filePath := range files {
		raw, err := runtime.Ops.ReadRawContent(runtime.Ctx, filePath)
		if err != nil {
			return fmt.Sprintf("frontmatter stat: %v", err), contract.ExitCodeGeneral
		}
		block := parseMarkdownFrontMatter(raw)
		keys := frontMatterKeys(block)
		rows = append(rows, frontmatterStatRow{
			Path:           filePath,
			HasFrontMatter: block.HasFrontMatter,
			StartLine:      block.StartLine,
			EndLine:        block.EndLine,
			KeyCount:       len(keys),
			Keys:           keys,
		})
	}
	return renderFrontmatterStat(rows, format), 0
}

func runFrontmatterGet(runtime engine.CommandRuntime, args []string) (string, int) {
	key := ""
	rawOutput := false
	pathValue := ""

	for idx := 0; idx < len(args); idx++ {
		arg := args[idx]
		switch {
		case arg == "--raw":
			rawOutput = true
		case arg == "--key":
			if idx+1 >= len(args) {
				return "frontmatter get: --key requires a value", contract.ExitCodeUsage
			}
			idx++
			key = strings.TrimSpace(args[idx])
			if key == "" {
				return "frontmatter get: key must not be empty", contract.ExitCodeUsage
			}
		case strings.HasPrefix(arg, "--key="):
			key = strings.TrimSpace(strings.TrimPrefix(arg, "--key="))
			if key == "" {
				return "frontmatter get: key must not be empty", contract.ExitCodeUsage
			}
		case strings.HasPrefix(arg, "-"):
			return fmt.Sprintf("frontmatter get: unsupported flag %s", arg), contract.ExitCodeUsage
		default:
			if pathValue != "" {
				return "frontmatter get: expected exactly one file path", contract.ExitCodeUsage
			}
			resolved, err := runtime.Ops.RequireAbsolutePath(arg)
			if err != nil {
				return fmt.Sprintf("frontmatter get: %v", err), contract.ExitCodeUsage
			}
			pathValue = resolved
		}
	}

	if pathValue == "" {
		return "frontmatter get: expected one file path", contract.ExitCodeUsage
	}

	isDir, err := runtime.Ops.IsDirPath(runtime.Ctx, pathValue)
	if err != nil {
		return fmt.Sprintf("frontmatter get: %v", err), contract.ExitCodeGeneral
	}
	if isDir {
		return fmt.Sprintf("frontmatter get: %s: is a directory", pathValue), contract.ExitCodeUsage
	}

	raw, err := runtime.Ops.ReadRawContent(runtime.Ctx, pathValue)
	if err != nil {
		return fmt.Sprintf("frontmatter get: %v", err), contract.ExitCodeGeneral
	}
	block := parseMarkdownFrontMatter(raw)
	if !block.HasFrontMatter {
		return fmt.Sprintf("frontmatter get: %s: no frontmatter", pathValue), contract.ExitCodeGeneral
	}

	if key == "" {
		if rawOutput {
			return strings.Join(block.Lines[block.StartLine-1:block.EndLine], "\n"), 0
		}
		return strings.Join(block.HeaderLines, "\n"), 0
	}

	start, end, ok := findFrontMatterKeyRange(block, key)
	if !ok {
		return fmt.Sprintf("frontmatter get: %s: key %q not found", pathValue, key), contract.ExitCodeGeneral
	}
	keyLines := append([]string(nil), block.Lines[start-1:end]...)
	if rawOutput {
		return strings.Join(keyLines, "\n"), 0
	}
	return extractFrontMatterValue(keyLines), 0
}

func runFrontmatterPrint(runtime engine.CommandRuntime, args []string) (string, int) {
	key := ""
	recursive := false
	before := 0
	after := 0
	numbered := false
	targets := make([]string, 0, len(args))

	for idx := 0; idx < len(args); idx++ {
		arg := args[idx]
		switch {
		case arg == "-r":
			recursive = true
		case arg == "-n":
			numbered = true
		case arg == "--key":
			if idx+1 >= len(args) {
				return "frontmatter print: --key requires a value", contract.ExitCodeUsage
			}
			idx++
			key = strings.TrimSpace(args[idx])
			if key == "" {
				return "frontmatter print: key must not be empty", contract.ExitCodeUsage
			}
		case strings.HasPrefix(arg, "--key="):
			key = strings.TrimSpace(strings.TrimPrefix(arg, "--key="))
			if key == "" {
				return "frontmatter print: key must not be empty", contract.ExitCodeUsage
			}
		case arg == "-B":
			if idx+1 >= len(args) {
				return "frontmatter print: -B requires a non-negative value", contract.ExitCodeUsage
			}
			idx++
			v, err := parseNonNegativeInt(args[idx])
			if err != nil {
				return fmt.Sprintf("frontmatter print: invalid -B value %q", args[idx]), contract.ExitCodeUsage
			}
			before = v
		case arg == "-A":
			if idx+1 >= len(args) {
				return "frontmatter print: -A requires a non-negative value", contract.ExitCodeUsage
			}
			idx++
			v, err := parseNonNegativeInt(args[idx])
			if err != nil {
				return fmt.Sprintf("frontmatter print: invalid -A value %q", args[idx]), contract.ExitCodeUsage
			}
			after = v
		case arg == "-C":
			if idx+1 >= len(args) {
				return "frontmatter print: -C requires a non-negative value", contract.ExitCodeUsage
			}
			idx++
			v, err := parseNonNegativeInt(args[idx])
			if err != nil {
				return fmt.Sprintf("frontmatter print: invalid -C value %q", args[idx]), contract.ExitCodeUsage
			}
			before = v
			after = v
		case strings.HasPrefix(arg, "-B") && isDigits(strings.TrimPrefix(arg, "-B")):
			v, err := parseNonNegativeInt(strings.TrimPrefix(arg, "-B"))
			if err != nil {
				return fmt.Sprintf("frontmatter print: invalid -B value %q", arg), contract.ExitCodeUsage
			}
			before = v
		case strings.HasPrefix(arg, "-A") && isDigits(strings.TrimPrefix(arg, "-A")):
			v, err := parseNonNegativeInt(strings.TrimPrefix(arg, "-A"))
			if err != nil {
				return fmt.Sprintf("frontmatter print: invalid -A value %q", arg), contract.ExitCodeUsage
			}
			after = v
		case strings.HasPrefix(arg, "-C") && isDigits(strings.TrimPrefix(arg, "-C")):
			v, err := parseNonNegativeInt(strings.TrimPrefix(arg, "-C"))
			if err != nil {
				return fmt.Sprintf("frontmatter print: invalid -C value %q", arg), contract.ExitCodeUsage
			}
			before = v
			after = v
		case strings.HasPrefix(arg, "-"):
			return fmt.Sprintf("frontmatter print: unsupported flag %s", arg), contract.ExitCodeUsage
		default:
			pathValue, err := runtime.Ops.RequireAbsolutePath(arg)
			if err != nil {
				return fmt.Sprintf("frontmatter print: %v", err), contract.ExitCodeUsage
			}
			targets = append(targets, pathValue)
		}
	}

	if len(targets) == 0 {
		return "frontmatter print: expected at least one path", contract.ExitCodeUsage
	}

	files, out, code := expandFrontmatterTargets(runtime, "frontmatter print", targets, recursive)
	if code != 0 {
		return out, code
	}

	sections := make([]string, 0, len(files))
	hasFailure := false
	for _, filePath := range files {
		raw, err := runtime.Ops.ReadRawContent(runtime.Ctx, filePath)
		if err != nil {
			hasFailure = true
			sections = append(sections, fmt.Sprintf("frontmatter print: %s: %v", filePath, err))
			continue
		}
		block := parseMarkdownFrontMatter(raw)
		if !block.HasFrontMatter {
			hasFailure = true
			sections = append(sections, fmt.Sprintf("frontmatter print: %s: no frontmatter", filePath))
			continue
		}
		start := block.StartLine
		end := block.EndLine
		if key != "" {
			keyStart, keyEnd, ok := findFrontMatterKeyRange(block, key)
			if !ok {
				hasFailure = true
				sections = append(sections, fmt.Sprintf("frontmatter print: %s: key %q not found", filePath, key))
				continue
			}
			start = keyStart
			end = keyEnd
		}

		view := renderLineRange(block.Lines, start, end, before, after, numbered)
		if len(files) > 1 {
			view = "== " + filePath + " ==\n" + view
		}
		sections = append(sections, view)
	}

	output := strings.Join(sections, "\n\n")
	if hasFailure {
		return output, contract.ExitCodeGeneral
	}
	return output, 0
}

func parseFrontmatterFormat(raw string) (frontmatterFormat, bool) {
	switch strings.TrimSpace(raw) {
	case string(frontmatterFormatCompact):
		return frontmatterFormatCompact, true
	case string(frontmatterFormatJSON):
		return frontmatterFormatJSON, true
	case string(frontmatterFormatMD):
		return frontmatterFormatMD, true
	default:
		return frontmatterFormatCompact, false
	}
}

func renderFrontmatterStat(rows []frontmatterStatRow, format frontmatterFormat) string {
	switch format {
	case frontmatterFormatJSON:
		type payload struct {
			Columns []string             `json:"columns"`
			Entries []frontmatterStatRow `json:"entries"`
		}
		raw, err := json.Marshal(payload{
			Columns: []string{"has", "fm_lines", "key_count", "keys", "path"},
			Entries: rows,
		})
		if err != nil {
			return "{}"
		}
		return string(raw)
	case frontmatterFormatMD:
		lines := []string{
			"| has | fm_lines | key_count | keys | path |",
			"|---|---|---:|---|---|",
		}
		for _, row := range rows {
			lines = append(lines, fmt.Sprintf("| %s | %s | %d | %s | %s |",
				boolYN(row.HasFrontMatter),
				formatFrontmatterRange(row),
				row.KeyCount,
				strings.ReplaceAll(frontmatterKeySummary(row.Keys), "|", "\\|"),
				strings.ReplaceAll(row.Path, "|", "\\|"),
			))
		}
		return strings.Join(lines, "\n")
	default:
		lines := make([]string, 0, len(rows)+1)
		for _, row := range rows {
			lines = append(lines, fmt.Sprintf("%s %s %d %s %s",
				boolYN(row.HasFrontMatter),
				formatFrontmatterRange(row),
				row.KeyCount,
				frontmatterKeySummary(row.Keys),
				row.Path,
			))
		}
		lines = append(lines, "# columns: has fm_lines key_count keys path")
		return strings.Join(lines, "\n")
	}
}

func boolYN(v bool) string {
	if v {
		return "y"
	}
	return "n"
}

func formatFrontmatterRange(row frontmatterStatRow) string {
	if !row.HasFrontMatter || row.StartLine <= 0 || row.EndLine <= 0 {
		return "-"
	}
	return fmt.Sprintf("%d:%d", row.StartLine, row.EndLine)
}

func frontmatterKeySummary(keys []string) string {
	if len(keys) == 0 {
		return "-"
	}
	if len(keys) <= 4 {
		return strings.Join(keys, ",")
	}
	return fmt.Sprintf("%s,+%d", strings.Join(keys[:4], ","), len(keys)-4)
}

func expandFrontmatterTargets(runtime engine.CommandRuntime, label string, targets []string, recursive bool) ([]string, string, int) {
	seen := map[string]struct{}{}
	files := make([]string, 0)
	appendFile := func(pathValue string) {
		if _, ok := seen[pathValue]; ok {
			return
		}
		seen[pathValue] = struct{}{}
		files = append(files, pathValue)
	}

	for _, target := range targets {
		isDir, err := runtime.Ops.IsDirPath(runtime.Ctx, target)
		if err != nil {
			return nil, fmt.Sprintf("%s: %v", label, err), contract.ExitCodeGeneral
		}
		if !isDir {
			appendFile(target)
			continue
		}
		if recursive {
			collected, err := runtime.Ops.CollectFilesUnder(runtime.Ctx, target)
			if err != nil {
				return nil, fmt.Sprintf("%s: %v", label, err), contract.ExitCodeGeneral
			}
			for _, filePath := range collected {
				appendFile(filePath)
			}
			continue
		}
		children, err := runtime.Ops.ListChildren(runtime.Ctx, target)
		if err != nil {
			return nil, fmt.Sprintf("%s: %v", label, err), contract.ExitCodeGeneral
		}
		for _, child := range children {
			childDir, err := runtime.Ops.IsDirPath(runtime.Ctx, child)
			if err != nil {
				return nil, fmt.Sprintf("%s: %v", label, err), contract.ExitCodeGeneral
			}
			if !childDir {
				appendFile(child)
			}
		}
	}

	sort.Strings(files)
	if len(files) == 0 {
		return nil, fmt.Sprintf("%s: no files found", label), contract.ExitCodeGeneral
	}
	return files, "", 0
}

func renderLineRange(lines []string, startLine int, endLine int, before int, after int, numbered bool) string {
	if len(lines) == 0 {
		return ""
	}
	if startLine < 1 {
		startLine = 1
	}
	if endLine < startLine {
		endLine = startLine
	}
	minLine := maxInt(1, startLine-before)
	maxLine := minInt(len(lines), endLine+after)
	out := make([]string, 0, maxLine-minLine+1)
	for lineNo := minLine; lineNo <= maxLine; lineNo++ {
		line := lines[lineNo-1]
		if numbered {
			out = append(out, fmt.Sprintf("%d:%s", lineNo, line))
			continue
		}
		out = append(out, line)
	}
	return strings.Join(out, "\n")
}

func extractFrontMatterValue(keyLines []string) string {
	if len(keyLines) == 0 {
		return ""
	}
	first := keyLines[0]
	idx := strings.Index(first, ":")
	if idx < 0 {
		return strings.Join(keyLines, "\n")
	}
	head := strings.TrimSpace(first[idx+1:])
	tail := dedentLines(keyLines[1:])
	tailText := strings.Join(tail, "\n")
	switch {
	case head == "" && tailText == "":
		return ""
	case head == "":
		return tailText
	case tailText == "":
		return head
	default:
		return head + "\n" + tailText
	}
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}
