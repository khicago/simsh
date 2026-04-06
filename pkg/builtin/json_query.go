package builtin

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"unicode/utf8"
)

type jsonQueryFormat string

const (
	jsonQueryFormatText  jsonQueryFormat = "text"
	jsonQueryFormatJSON  jsonQueryFormat = "json"
	jsonQueryFormatJSONL jsonQueryFormat = "jsonl"
)

type jsonKeysRow struct {
	Path  string   `json:"path"`
	Query string   `json:"query"`
	OK    bool     `json:"ok"`
	Kind  string   `json:"kind"`
	Count int      `json:"count"`
	Keys  []string `json:"keys,omitempty"`
	Error string   `json:"error,omitempty"`
}

type jsonLenRow struct {
	Path   string `json:"path"`
	Query  string `json:"query"`
	OK     bool   `json:"ok"`
	Kind   string `json:"kind"`
	Length int    `json:"length"`
	Error  string `json:"error,omitempty"`
}

type jsonGetRecord struct {
	Path  string `json:"path"`
	Query string `json:"query"`
	Value any    `json:"value"`
}

type parsedJSONQuery struct {
	raw   string
	steps []jsonPathStep
}

func parseJSONQueryFormat(raw string) (jsonQueryFormat, bool) {
	switch strings.TrimSpace(raw) {
	case string(jsonQueryFormatText), "":
		return jsonQueryFormatText, true
	case string(jsonQueryFormatJSON):
		return jsonQueryFormatJSON, true
	case string(jsonQueryFormatJSONL):
		return jsonQueryFormatJSONL, true
	default:
		return jsonQueryFormatText, false
	}
}

func jsonQueryDisplay(query string) string {
	if strings.TrimSpace(query) == "" {
		return "."
	}
	return strings.TrimSpace(query)
}

func parseJSONQuery(query string) (parsedJSONQuery, error) {
	steps, err := parseJSONPath(query)
	if err != nil {
		return parsedJSONQuery{}, err
	}
	return parsedJSONQuery{raw: query, steps: steps}, nil
}

func selectParsedJSONQuery(value any, query parsedJSONQuery) (any, error) {
	return applyJSONPath(value, query.steps)
}

func buildJSONKeysRow(filePath string, query parsedJSONQuery, raw string) jsonKeysRow {
	row := jsonKeysRow{
		Path:  filePath,
		Query: jsonQueryDisplay(query.raw),
		OK:    false,
		Kind:  "invalid",
		Count: -1,
	}
	var value any
	if err := json.Unmarshal([]byte(raw), &value); err != nil {
		row.Error = fmt.Sprintf("invalid json: %v", err)
		return row
	}
	selected, err := selectParsedJSONQuery(value, query)
	if err != nil {
		row.Kind = "error"
		row.Error = err.Error()
		return row
	}
	kind, _, _ := summarizeJSONValue(selected)
	row.Kind = kind
	obj, ok := selected.(map[string]any)
	if !ok {
		row.Error = "selected value is not an object"
		return row
	}
	keys := make([]string, 0, len(obj))
	for key := range obj {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	row.OK = true
	row.Count = len(keys)
	row.Keys = keys
	return row
}

func buildJSONLenRow(filePath string, query parsedJSONQuery, raw string) jsonLenRow {
	row := jsonLenRow{
		Path:   filePath,
		Query:  jsonQueryDisplay(query.raw),
		OK:     false,
		Kind:   "invalid",
		Length: -1,
	}
	var value any
	if err := json.Unmarshal([]byte(raw), &value); err != nil {
		row.Error = fmt.Sprintf("invalid json: %v", err)
		return row
	}
	selected, err := selectParsedJSONQuery(value, query)
	if err != nil {
		row.Kind = "error"
		row.Error = err.Error()
		return row
	}
	kind, _, _ := summarizeJSONValue(selected)
	row.Kind = kind
	switch v := selected.(type) {
	case map[string]any:
		row.OK = true
		row.Length = len(v)
	case []any:
		row.OK = true
		row.Length = len(v)
	case string:
		row.OK = true
		row.Length = utf8.RuneCountInString(v)
	default:
		row.Error = "selected value does not have a length"
	}
	return row
}

func renderJSONKeys(rows []jsonKeysRow, format jsonQueryFormat) string {
	switch format {
	case jsonQueryFormatJSON:
		return compactJSONString(struct {
			Columns []string      `json:"columns"`
			Entries []jsonKeysRow `json:"entries"`
		}{
			Columns: []string{"ok", "query", "kind", "count", "keys", "path", "error"},
			Entries: rows,
		})
	case jsonQueryFormatJSONL:
		lines := make([]string, 0, len(rows))
		for _, row := range rows {
			raw, _ := json.Marshal(row)
			lines = append(lines, string(raw))
		}
		return strings.Join(lines, "\n")
	default:
		lines := make([]string, 0, len(rows)+1)
		for _, row := range rows {
			keys := "-"
			if len(row.Keys) > 0 {
				keys = strings.Join(row.Keys, ",")
			}
			errText := "-"
			if strings.TrimSpace(row.Error) != "" {
				errText = row.Error
			}
			lines = append(lines, fmt.Sprintf("%s %s %s %s %s %s %s",
				boolYN(row.OK),
				row.Query,
				row.Kind,
				renderJSONSize(row.Count),
				keys,
				row.Path,
				errText,
			))
		}
		lines = append(lines, "# columns: ok query kind count keys path error")
		return strings.Join(lines, "\n")
	}
}

func renderJSONLens(rows []jsonLenRow, format jsonQueryFormat) string {
	switch format {
	case jsonQueryFormatJSON:
		return compactJSONString(struct {
			Columns []string     `json:"columns"`
			Entries []jsonLenRow `json:"entries"`
		}{
			Columns: []string{"ok", "query", "kind", "length", "path", "error"},
			Entries: rows,
		})
	case jsonQueryFormatJSONL:
		lines := make([]string, 0, len(rows))
		for _, row := range rows {
			raw, _ := json.Marshal(row)
			lines = append(lines, string(raw))
		}
		return strings.Join(lines, "\n")
	default:
		lines := make([]string, 0, len(rows)+1)
		for _, row := range rows {
			errText := "-"
			if strings.TrimSpace(row.Error) != "" {
				errText = row.Error
			}
			lines = append(lines, fmt.Sprintf("%s %s %s %s %s %s",
				boolYN(row.OK),
				row.Query,
				row.Kind,
				renderJSONSize(row.Length),
				row.Path,
				errText,
			))
		}
		lines = append(lines, "# columns: ok query kind length path error")
		return strings.Join(lines, "\n")
	}
}

func renderJSONGetMulti(pathValue string, queries []parsedJSONQuery, values map[string]any, format jsonQueryFormat, rawOutput bool) string {
	switch format {
	case jsonQueryFormatJSONL:
		lines := make([]string, 0, len(queries))
		for _, query := range queries {
			raw, _ := json.Marshal(jsonGetRecord{
				Path:  pathValue,
				Query: jsonQueryDisplay(query.raw),
				Value: values[query.raw],
			})
			lines = append(lines, string(raw))
		}
		return strings.Join(lines, "\n")
	case jsonQueryFormatJSON:
		return compactJSONString(values)
	default:
		if rawOutput {
			return compactJSONString(values)
		}
		return prettyJSONString(values)
	}
}
