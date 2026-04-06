package builtin

import (
	"bytes"
	"encoding/json"
	"errors"
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
		Manual: "json <stat|get|keys|len> ...",
		Tips: []string{
			"Use json stat to inspect JSON shape across files and directories.",
			"Use json get --path QUERY to extract one or a small set of JSON subtrees without dumping the whole file.",
			"Use json keys and json len for narrow structure-aware queries instead of re-reading full JSON into the model.",
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
	case "keys":
		return runJSONKeys(runtime, args[1:])
	case "len":
		return runJSONLen(runtime, args[1:])
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
	inputs, err := readJSONInputs(runtime, "json stat", files)
	if err != nil {
		return fmt.Sprintf("json stat: %v", err), contract.ExitCodeGeneral
	}
	rows := make([]jsonStatRow, 0, len(inputs))
	for _, input := range inputs {
		filePath := input.Path
		raw := input.Raw
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
	pathQueries := make([]string, 0, 1)
	rawOutput := false
	format := jsonQueryFormatText
	pathValue := ""
	for idx := 0; idx < len(args); idx++ {
		arg := args[idx]
		switch {
		case arg == "--raw":
			rawOutput = true
		case arg == "--fmt":
			if idx+1 >= len(args) {
				return "json get: --fmt requires one value: json|jsonl", contract.ExitCodeUsage
			}
			idx++
			parsed, ok := parseJSONQueryFormat(args[idx])
			if !ok || parsed == jsonQueryFormatText {
				return fmt.Sprintf("json get: unsupported --fmt value %q", args[idx]), contract.ExitCodeUsage
			}
			format = parsed
		case strings.HasPrefix(arg, "--fmt="):
			parsed, ok := parseJSONQueryFormat(strings.TrimPrefix(arg, "--fmt="))
			if !ok || parsed == jsonQueryFormatText {
				return fmt.Sprintf("json get: unsupported --fmt value %q", strings.TrimPrefix(arg, "--fmt=")), contract.ExitCodeUsage
			}
			format = parsed
		case arg == "--path":
			if idx+1 >= len(args) {
				return "json get: --path requires a value", contract.ExitCodeUsage
			}
			idx++
			pathQuery := strings.TrimSpace(args[idx])
			if pathQuery == "" {
				return "json get: path must not be empty", contract.ExitCodeUsage
			}
			pathQueries = append(pathQueries, pathQuery)
		case strings.HasPrefix(arg, "--path="):
			pathQuery := strings.TrimSpace(strings.TrimPrefix(arg, "--path="))
			if pathQuery == "" {
				return "json get: path must not be empty", contract.ExitCodeUsage
			}
			pathQueries = append(pathQueries, pathQuery)
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
	seenQueries := map[string]struct{}{}
	parsedQueries := make([]parsedJSONQuery, 0, len(pathQueries))
	for _, query := range pathQueries {
		if _, ok := seenQueries[query]; ok {
			return fmt.Sprintf("json get: duplicate --path value %q", query), contract.ExitCodeUsage
		}
		seenQueries[query] = struct{}{}
		parsed, err := parseJSONQuery(query)
		if err != nil {
			return fmt.Sprintf("json get: %v", err), contract.ExitCodeUsage
		}
		parsedQueries = append(parsedQueries, parsed)
	}
	if rawOutput && format == jsonQueryFormatJSONL {
		return "json get: --raw is not supported with --fmt jsonl", contract.ExitCodeUsage
	}
	if format == jsonQueryFormatJSONL && len(pathQueries) == 0 {
		return "json get: --fmt jsonl requires at least one --path", contract.ExitCodeUsage
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
	if len(pathQueries) <= 1 {
		query := parsedJSONQuery{}
		if len(parsedQueries) == 1 {
			query = parsedQueries[0]
		}
		selected, err := selectParsedJSONQuery(value, query)
		if err != nil {
			return fmt.Sprintf("json get: %v", err), contract.ExitCodeGeneral
		}
		if format == jsonQueryFormatJSONL {
			return renderJSONGetMulti(pathValue, []parsedJSONQuery{query}, map[string]any{query.raw: selected}, format, rawOutput), 0
		}
		if format == jsonQueryFormatJSON {
			return compactJSONString(selected), 0
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

	values := make(map[string]any, len(pathQueries))
	for _, query := range parsedQueries {
		selected, err := selectParsedJSONQuery(value, query)
		if err != nil {
			return fmt.Sprintf("json get: %v", err), contract.ExitCodeGeneral
		}
		values[query.raw] = selected
	}
	return renderJSONGetMulti(pathValue, parsedQueries, values, format, rawOutput), 0
}

type jsonBatchQueryOptions struct {
	recursive bool
	query     parsedJSONQuery
	format    jsonQueryFormat
	targets   []string
}

func runJSONKeys(runtime engine.CommandRuntime, args []string) (string, int) {
	opts, out, code, ok := parseJSONBatchQueryArgs(runtime, "json keys", args)
	if !ok {
		return out, code
	}
	files, out, code := expandJSONTargets(runtime, "json keys", opts.targets, opts.recursive)
	if code != 0 {
		return out, code
	}
	inputs, err := readJSONInputs(runtime, "json keys", files)
	if err != nil {
		return fmt.Sprintf("json keys: %v", err), contract.ExitCodeGeneral
	}
	rows := make([]jsonKeysRow, 0, len(inputs))
	for _, input := range inputs {
		rows = append(rows, buildJSONKeysRow(input.Path, opts.query, input.Raw))
	}
	return renderJSONKeys(rows, opts.format), 0
}

func runJSONLen(runtime engine.CommandRuntime, args []string) (string, int) {
	opts, out, code, ok := parseJSONBatchQueryArgs(runtime, "json len", args)
	if !ok {
		return out, code
	}
	files, out, code := expandJSONTargets(runtime, "json len", opts.targets, opts.recursive)
	if code != 0 {
		return out, code
	}
	inputs, err := readJSONInputs(runtime, "json len", files)
	if err != nil {
		return fmt.Sprintf("json len: %v", err), contract.ExitCodeGeneral
	}
	rows := make([]jsonLenRow, 0, len(inputs))
	for _, input := range inputs {
		rows = append(rows, buildJSONLenRow(input.Path, opts.query, input.Raw))
	}
	return renderJSONLens(rows, opts.format), 0
}

func parseJSONBatchQueryArgs(runtime engine.CommandRuntime, label string, args []string) (jsonBatchQueryOptions, string, int, bool) {
	opts := jsonBatchQueryOptions{
		format:  jsonQueryFormatText,
		targets: make([]string, 0, len(args)),
	}
	for idx := 0; idx < len(args); idx++ {
		arg := args[idx]
		switch {
		case arg == "-r":
			opts.recursive = true
		case arg == "--path":
			if idx+1 >= len(args) {
				return opts, fmt.Sprintf("%s: --path requires a value", label), contract.ExitCodeUsage, false
			}
			idx++
			query := strings.TrimSpace(args[idx])
			if query == "" {
				return opts, fmt.Sprintf("%s: path must not be empty", label), contract.ExitCodeUsage, false
			}
			parsed, err := parseJSONQuery(query)
			if err != nil {
				return opts, fmt.Sprintf("%s: %v", label, err), contract.ExitCodeUsage, false
			}
			opts.query = parsed
		case strings.HasPrefix(arg, "--path="):
			query := strings.TrimSpace(strings.TrimPrefix(arg, "--path="))
			if query == "" {
				return opts, fmt.Sprintf("%s: path must not be empty", label), contract.ExitCodeUsage, false
			}
			parsed, err := parseJSONQuery(query)
			if err != nil {
				return opts, fmt.Sprintf("%s: %v", label, err), contract.ExitCodeUsage, false
			}
			opts.query = parsed
		case arg == "--fmt":
			if idx+1 >= len(args) {
				return opts, fmt.Sprintf("%s: --fmt requires one value: text|json|jsonl", label), contract.ExitCodeUsage, false
			}
			idx++
			parsed, ok := parseJSONQueryFormat(args[idx])
			if !ok {
				return opts, fmt.Sprintf("%s: unsupported --fmt value %q", label, args[idx]), contract.ExitCodeUsage, false
			}
			opts.format = parsed
		case strings.HasPrefix(arg, "--fmt="):
			parsed, ok := parseJSONQueryFormat(strings.TrimPrefix(arg, "--fmt="))
			if !ok {
				return opts, fmt.Sprintf("%s: unsupported --fmt value %q", label, strings.TrimPrefix(arg, "--fmt=")), contract.ExitCodeUsage, false
			}
			opts.format = parsed
		case strings.HasPrefix(arg, "-"):
			return opts, fmt.Sprintf("%s: unsupported flag %s", label, arg), contract.ExitCodeUsage, false
		default:
			pathValue, err := runtime.Ops.RequireAbsolutePath(arg)
			if err != nil {
				return opts, fmt.Sprintf("%s: %v", label, err), contract.ExitCodeUsage, false
			}
			opts.targets = append(opts.targets, pathValue)
		}
	}
	if len(opts.targets) == 0 {
		return opts, fmt.Sprintf("%s: expected at least one path", label), contract.ExitCodeUsage, false
	}
	return opts, "", 0, true
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
		entries, err := listDirectoryEntries(runtime, target, false)
		if err != nil {
			return nil, fmt.Sprintf("%s: %v", label, err), contract.ExitCodeGeneral
		}
		for _, entry := range entries {
			if !entry.Meta.IsDir {
				appendFile(entry.Path)
			}
		}
	}
	sort.Strings(files)
	if len(files) == 0 {
		return nil, fmt.Sprintf("%s: no files found", label), contract.ExitCodeGeneral
	}
	return files, "", 0
}

type jsonInput struct {
	Path string
	Raw  string
}

func readJSONInputs(runtime engine.CommandRuntime, label string, files []string) ([]jsonInput, error) {
	if runtime.Ops.ReadMany != nil && len(files) > 1 {
		result, err := runtime.Ops.ReadMany(runtime.Ctx, contract.ReadManyRequest{Paths: files})
		if err == nil {
			index := make(map[string]string, len(result.Entries))
			for _, entry := range result.Entries {
				index[entry.Path] = entry.Content
			}
			inputs := make([]jsonInput, 0, len(files))
			for _, filePath := range files {
				raw, ok := index[filePath]
				if !ok {
					return nil, fmt.Errorf("%s: missing content for %s", label, filePath)
				}
				inputs = append(inputs, jsonInput{Path: filePath, Raw: raw})
			}
			return inputs, nil
		}
		if !errors.Is(err, contract.ErrUnsupported) {
			return nil, err
		}
	}
	inputs := make([]jsonInput, 0, len(files))
	for _, filePath := range files {
		raw, err := runtime.Ops.ReadRawContent(runtime.Ctx, filePath)
		if err != nil {
			return nil, err
		}
		inputs = append(inputs, jsonInput{Path: filePath, Raw: raw})
	}
	return inputs, nil
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
