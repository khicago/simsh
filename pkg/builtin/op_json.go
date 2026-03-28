package builtin

import (
	"bytes"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/khicago/simsh/pkg/contract"
	"github.com/khicago/simsh/pkg/engine"
)

type jsonStatFormat string

const (
	jsonStatFormatCompact jsonStatFormat = "compact"
	jsonStatFormatJSON    jsonStatFormat = "json"
	jsonStatFormatMD      jsonStatFormat = "md"
)

type jsonStatRow struct {
	Path  string   `json:"path"`
	Valid bool     `json:"valid"`
	Kind  string   `json:"kind"`
	Size  int      `json:"size"`
	Keys  []string `json:"keys,omitempty"`
}

func specJSON() engine.CommandSpec {
	return engine.CommandSpec{
		Name:   CommandJSON,
		Manual: "json <stat|get> ...",
		Tips: []string{
			"Use json stat to inspect JSON shape across files and directories.",
			"Use json get --path QUERY to extract a JSON subtree without dumping the whole file.",
		},
		Examples:       ExamplesFor("json"),
		DetailedManual: LoadEmbeddedManual("json"),
		Run:            runJSON,
	}
}

func runJSON(runtime engine.CommandRuntime, args []string) (string, int) {
	if len(args) == 0 {
		return "json: expected subcommand: stat|get", contract.ExitCodeUsage
	}
	switch strings.TrimSpace(args[0]) {
	case "stat":
		return runJSONStat(runtime, args[1:])
	case "get":
		return runJSONGet(runtime, args[1:])
	default:
		return fmt.Sprintf("json: unsupported subcommand %s", strings.TrimSpace(args[0])), contract.ExitCodeUsage
	}
}

func runJSONStat(runtime engine.CommandRuntime, args []string) (string, int) {
	recursive := false
	format := jsonStatFormatCompact
	targets := make([]string, 0, len(args))
	for idx := 0; idx < len(args); idx++ {
		arg := args[idx]
		switch {
		case arg == "-r":
			recursive = true
		case arg == "--fmt":
			if idx+1 >= len(args) {
				return "json stat: --fmt requires one value: compact|json|md", contract.ExitCodeUsage
			}
			idx++
			parsed, ok := parseJSONStatFormat(args[idx])
			if !ok {
				return fmt.Sprintf("json stat: unsupported --fmt value %q", args[idx]), contract.ExitCodeUsage
			}
			format = parsed
		case strings.HasPrefix(arg, "--fmt="):
			parsed, ok := parseJSONStatFormat(strings.TrimPrefix(arg, "--fmt="))
			if !ok {
				return fmt.Sprintf("json stat: unsupported --fmt value %q", strings.TrimPrefix(arg, "--fmt=")), contract.ExitCodeUsage
			}
			format = parsed
		case strings.HasPrefix(arg, "-"):
			return fmt.Sprintf("json stat: unsupported flag %s", arg), contract.ExitCodeUsage
		default:
			pathValue, err := runtime.Ops.RequireAbsolutePath(arg)
			if err != nil {
				return fmt.Sprintf("json stat: %v", err), contract.ExitCodeUsage
			}
			targets = append(targets, pathValue)
		}
	}
	if len(targets) == 0 {
		return "json stat: expected at least one path", contract.ExitCodeUsage
	}
	files, out, code := expandJSONTargets(runtime, "json stat", targets, recursive)
	if code != 0 {
		return out, code
	}
	rows := make([]jsonStatRow, 0, len(files))
	for _, filePath := range files {
		raw, err := runtime.Ops.ReadRawContent(runtime.Ctx, filePath)
		if err != nil {
			return fmt.Sprintf("json stat: %v", err), contract.ExitCodeGeneral
		}
		row := jsonStatRow{Path: filePath}
		var value any
		if err := json.Unmarshal([]byte(raw), &value); err == nil {
			row.Valid = true
			row.Kind, row.Size, row.Keys = summarizeJSONValue(value)
		}
		if !row.Valid {
			row.Kind = "invalid"
			row.Size = -1
		}
		rows = append(rows, row)
	}
	return renderJSONStat(rows, format), 0
}

func runJSONGet(runtime engine.CommandRuntime, args []string) (string, int) {
	pathQuery := ""
	rawOutput := false
	pathValue := ""
	for idx := 0; idx < len(args); idx++ {
		arg := args[idx]
		switch {
		case arg == "--raw":
			rawOutput = true
		case arg == "--path":
			if idx+1 >= len(args) {
				return "json get: --path requires a value", contract.ExitCodeUsage
			}
			idx++
			pathQuery = strings.TrimSpace(args[idx])
			if pathQuery == "" {
				return "json get: path must not be empty", contract.ExitCodeUsage
			}
		case strings.HasPrefix(arg, "--path="):
			pathQuery = strings.TrimSpace(strings.TrimPrefix(arg, "--path="))
			if pathQuery == "" {
				return "json get: path must not be empty", contract.ExitCodeUsage
			}
		case strings.HasPrefix(arg, "-"):
			return fmt.Sprintf("json get: unsupported flag %s", arg), contract.ExitCodeUsage
		default:
			if pathValue != "" {
				return "json get: expected exactly one file path", contract.ExitCodeUsage
			}
			resolved, err := runtime.Ops.RequireAbsolutePath(arg)
			if err != nil {
				return fmt.Sprintf("json get: %v", err), contract.ExitCodeUsage
			}
			pathValue = resolved
		}
	}
	if pathValue == "" {
		return "json get: expected one file path", contract.ExitCodeUsage
	}
	isDir, err := runtime.Ops.IsDirPath(runtime.Ctx, pathValue)
	if err != nil {
		return fmt.Sprintf("json get: %v", err), contract.ExitCodeGeneral
	}
	if isDir {
		return fmt.Sprintf("json get: %s: is a directory", pathValue), contract.ExitCodeUsage
	}
	raw, err := runtime.Ops.ReadRawContent(runtime.Ctx, pathValue)
	if err != nil {
		return fmt.Sprintf("json get: %v", err), contract.ExitCodeGeneral
	}
	var value any
	if err := json.Unmarshal([]byte(raw), &value); err != nil {
		return fmt.Sprintf("json get: invalid json: %v", err), contract.ExitCodeGeneral
	}
	steps, err := parseJSONPath(pathQuery)
	if err != nil {
		return fmt.Sprintf("json get: %v", err), contract.ExitCodeUsage
	}
	selected, err := applyJSONPath(value, steps)
	if err != nil {
		return fmt.Sprintf("json get: %v", err), contract.ExitCodeGeneral
	}
	switch v := selected.(type) {
	case string:
		return v, 0
	case float64, bool, nil:
		raw, err := json.Marshal(v)
		if err != nil {
			return fmt.Sprintf("json get: %v", err), contract.ExitCodeGeneral
		}
		return string(raw), 0
	default:
		if rawOutput {
			return compactJSONString(v), 0
		}
		return prettyJSONString(v), 0
	}
}

func parseJSONStatFormat(raw string) (jsonStatFormat, bool) {
	switch strings.TrimSpace(raw) {
	case string(jsonStatFormatCompact), "":
		return jsonStatFormatCompact, true
	case string(jsonStatFormatJSON):
		return jsonStatFormatJSON, true
	case string(jsonStatFormatMD):
		return jsonStatFormatMD, true
	default:
		return jsonStatFormatCompact, false
	}
}

func summarizeJSONValue(value any) (kind string, size int, keys []string) {
	switch v := value.(type) {
	case map[string]any:
		keys = make([]string, 0, len(v))
		for key := range v {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		return "object", len(v), keys
	case []any:
		return "array", len(v), nil
	case string:
		return "string", 1, nil
	case float64:
		return "number", 1, nil
	case bool:
		return "bool", 1, nil
	case nil:
		return "null", 0, nil
	default:
		return "unknown", -1, nil
	}
}

func renderJSONStat(rows []jsonStatRow, format jsonStatFormat) string {
	switch format {
	case jsonStatFormatJSON:
		return compactJSONString(struct {
			Columns []string      `json:"columns"`
			Entries []jsonStatRow `json:"entries"`
		}{
			Columns: []string{"valid", "kind", "size", "keys", "path"},
			Entries: rows,
		})
	case jsonStatFormatMD:
		lines := []string{
			"| valid | kind | size | keys | path |",
			"|---|---|---:|---|---|",
		}
		for _, row := range rows {
			lines = append(lines, fmt.Sprintf("| %s | %s | %s | %s | %s |",
				boolYN(row.Valid),
				row.Kind,
				renderJSONSize(row.Size),
				strings.ReplaceAll(strings.Join(row.Keys, ","), "|", "\\|"),
				strings.ReplaceAll(row.Path, "|", "\\|"),
			))
		}
		return strings.Join(lines, "\n")
	default:
		lines := make([]string, 0, len(rows)+1)
		for _, row := range rows {
			keys := "-"
			if len(row.Keys) > 0 {
				keys = strings.Join(row.Keys, ",")
			}
			lines = append(lines, fmt.Sprintf("%s %s %s %s %s",
				boolYN(row.Valid),
				row.Kind,
				renderJSONSize(row.Size),
				keys,
				row.Path,
			))
		}
		lines = append(lines, "# columns: valid kind size keys path")
		return strings.Join(lines, "\n")
	}
}

func renderJSONSize(size int) string {
	if size < 0 {
		return "-"
	}
	return fmt.Sprintf("%d", size)
}

func expandJSONTargets(runtime engine.CommandRuntime, label string, targets []string, recursive bool) ([]string, string, int) {
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

func compactJSONString(value any) string {
	raw, _ := json.Marshal(value)
	return normalizeJSONBytes(string(raw))
}

func prettyJSONString(value any) string {
	raw, _ := json.MarshalIndent(value, "", "  ")
	return normalizeJSONBytes(string(raw))
}

func normalizeJSONBytes(raw string) string {
	return string(bytes.TrimSpace([]byte(raw)))
}
