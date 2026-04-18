package builtin

import (
	"encoding/json"
	"fmt"
	"path"
	"regexp"
	"strings"
	"unicode"

	"github.com/khicago/simsh/pkg/contract"
	"github.com/khicago/simsh/pkg/engine"
)

type searchCaseMode int

const (
	searchCaseSensitive searchCaseMode = iota
	searchCaseIgnore
	searchCaseSmart
)

type searchMatcherOptions struct {
	Regex    bool
	CaseMode searchCaseMode
}

type searchRecord struct {
	Path  string `json:"path,omitempty"`
	Name  string `json:"name,omitempty"`
	Stdin bool   `json:"stdin,omitempty"`
	Line  int    `json:"line"`
	Kind  string `json:"kind"`
	Text  string `json:"text,omitempty"`
}

func buildSearchMatcher(pattern string, opts searchMatcherOptions) (func(string) bool, error) {
	caseMode := effectiveSearchCaseMode(pattern, opts.CaseMode)
	if opts.Regex {
		if caseMode == searchCaseIgnore {
			pattern = "(?i)" + pattern
		}
		re, err := regexp.Compile(pattern)
		if err != nil {
			return nil, err
		}
		return re.MatchString, nil
	}

	if caseMode == searchCaseIgnore {
		lowerPattern := strings.ToLower(pattern)
		return func(line string) bool {
			return strings.Contains(strings.ToLower(line), lowerPattern)
		}, nil
	}
	return func(line string) bool {
		return strings.Contains(line, pattern)
	}, nil
}

func effectiveSearchCaseMode(pattern string, mode searchCaseMode) searchCaseMode {
	if mode != searchCaseSmart {
		return mode
	}
	if searchPatternHasUpper(pattern) {
		return searchCaseSensitive
	}
	return searchCaseIgnore
}

func searchPatternHasUpper(pattern string) bool {
	for _, r := range pattern {
		if unicode.IsUpper(r) {
			return true
		}
	}
	return false
}

func searchHasMatch(raw string, match func(string) bool) bool {
	for _, line := range splitRawLines(raw) {
		if match(line) {
			return true
		}
	}
	return false
}

func renderSearchJSONL(records []searchRecord) string {
	if len(records) == 0 {
		return ""
	}
	lines := make([]string, 0, len(records))
	for _, record := range records {
		raw, err := json.Marshal(record)
		if err != nil {
			return fmt.Sprintf(`{"kind":"error","text":%q}`, err.Error())
		}
		lines = append(lines, string(raw))
	}
	return strings.Join(lines, "\n")
}

func searchRecordsWithContext(raw string, match func(string) bool, before int, after int, filePath string, stdin bool) []searchRecord {
	lines := splitRawLines(raw)
	if len(lines) == 0 {
		return nil
	}
	matched := make([]bool, len(lines))
	include := make([]bool, len(lines))
	for i, line := range lines {
		if !match(line) {
			continue
		}
		matched[i] = true
		start := i - before
		if start < 0 {
			start = 0
		}
		end := i + after
		if end >= len(lines) {
			end = len(lines) - 1
		}
		for j := start; j <= end; j++ {
			include[j] = true
		}
	}
	out := make([]searchRecord, 0)
	for i, line := range lines {
		if !include[i] {
			continue
		}
		record := searchRecord{
			Path:  filePath,
			Stdin: stdin,
			Line:  i + 1,
			Kind:  "context",
			Text:  line,
		}
		if matched[i] {
			record.Kind = "match"
		}
		out = append(out, record)
	}
	return out
}

func searchTextWithContext(raw string, match func(string) bool, before int, after int, filePath string) []string {
	records := searchRecordsWithContext(raw, match, before, after, filePath, false)
	out := make([]string, 0, len(records))
	for _, record := range records {
		if record.Path != "" {
			sep := '-'
			if record.Kind == "match" {
				sep = ':'
			}
			out = append(out, fmt.Sprintf("%s%c%d:%s", record.Path, sep, record.Line, record.Text))
			continue
		}
		if record.Kind == "match" {
			out = append(out, fmt.Sprintf("%d:%s", record.Line, record.Text))
		} else {
			out = append(out, fmt.Sprintf("%d-%s", record.Line, record.Text))
		}
	}
	return out
}

func parseSearchContextArg(arg string, idx int, args []string) (int, int, error) {
	if len(arg) < 2 {
		return 0, 0, fmt.Errorf("invalid context flag")
	}
	if len(arg) == 2 {
		if idx+1 >= len(args) {
			return 0, 0, fmt.Errorf("%s requires non-negative integer", arg)
		}
		value, err := parseNonNegativeInt(args[idx+1])
		if err != nil {
			return 0, 0, err
		}
		return value, 1, nil
	}
	suffix := strings.TrimPrefix(arg[2:], "=")
	value, err := parseNonNegativeInt(suffix)
	if err != nil {
		return 0, 0, err
	}
	return value, 0, nil
}

func renderSearchFileJSONL(paths []string) string {
	if len(paths) == 0 {
		return ""
	}
	records := make([]searchRecord, 0, len(paths))
	for _, filePath := range paths {
		records = append(records, searchRecord{
			Path: filePath,
			Name: path.Base(filePath),
			Kind: "file",
		})
	}
	return renderSearchJSONL(records)
}

func contractRecordsToSearchRecords(records []contract.SearchRecord, includeFileNames bool) []searchRecord {
	out := make([]searchRecord, len(records))
	for idx, rec := range records {
		out[idx] = searchRecord{
			Path: rec.Path,
			Line: rec.Line,
			Kind: rec.Kind,
			Text: rec.Text,
		}
		if includeFileNames && rec.Kind == "file" && strings.TrimSpace(rec.Path) != "" {
			out[idx].Name = path.Base(rec.Path)
		}
	}
	return out
}

func searchCaseModeToContract(mode searchCaseMode) contract.SearchCaseMode {
	switch mode {
	case searchCaseSensitive:
		return contract.SearchCaseSensitive
	case searchCaseIgnore:
		return contract.SearchCaseIgnore
	default:
		return contract.SearchCaseSmart
	}
}

func buildContractSearchRequest(pattern string, opts searchMatcherOptions, globs, targets []string, listFiles bool, before, after int, maxResults int) contract.SearchRequest {
	if len(targets) == 0 {
		return contract.SearchRequest{}
	}
	return contract.SearchRequest{
		Pattern:    pattern,
		Regex:      opts.Regex,
		CaseMode:   searchCaseModeToContract(opts.CaseMode),
		Targets:    append([]string(nil), targets...),
		Globs:      append([]string(nil), globs...),
		Before:     before,
		After:      after,
		MaxResults: maxResults,
		ListFiles:  listFiles,
	}
}

func tryRuntimeSearch(runtime engine.CommandRuntime, req contract.SearchRequest) (bool, contract.SearchResult, error) {
	if len(req.Targets) == 0 {
		return false, contract.SearchResult{}, nil
	}
	if runtime.Ops.SearchContent == nil {
		return false, contract.SearchResult{}, nil
	}
	result, err := runtime.Ops.SearchContent(runtime.Ctx, req)
	if err != nil {
		if contract.AllowsUnsupportedFallback(err) {
			return false, contract.SearchResult{}, nil
		}
		return false, contract.SearchResult{}, err
	}
	return true, result, nil
}

func renderSearchRecords(records []contract.SearchRecord, listFiles bool, jsonl bool, includeFileNames bool) (string, int) {
	scr := contractRecordsToSearchRecords(records, includeFileNames)
	if listFiles {
		filePaths := uniqueFilePaths(scr)
		if jsonl {
			return renderSearchJSONL(filterFileRecords(scr)), grepExitCode(len(filePaths) > 0)
		}
		if len(filePaths) == 0 {
			return "", contract.ExitCodeGeneral
		}
		return strings.Join(filePaths, "\n"), 0
	}
	if jsonl {
		return renderSearchJSONL(scr), grepExitCode(len(scr) > 0)
	}
	lines := recordLines(scr)
	if len(lines) == 0 {
		return "", contract.ExitCodeGeneral
	}
	return strings.Join(lines, "\n"), 0
}

func uniqueFilePaths(records []searchRecord) []string {
	seen := map[string]struct{}{}
	out := make([]string, 0, len(records))
	for _, rec := range records {
		if rec.Path == "" || rec.Kind != "file" {
			continue
		}
		if _, exists := seen[rec.Path]; exists {
			continue
		}
		seen[rec.Path] = struct{}{}
		out = append(out, rec.Path)
	}
	return out
}

func filterFileRecords(records []searchRecord) []searchRecord {
	out := make([]searchRecord, 0, len(records))
	for _, rec := range records {
		if rec.Kind == "file" {
			out = append(out, rec)
		}
	}
	return out
}

func recordLines(records []searchRecord) []string {
	out := make([]string, 0, len(records))
	for _, rec := range records {
		lineText := rec.Text
		sep := '-'
		if rec.Kind == "match" {
			sep = ':'
		}
		switch {
		case rec.Path != "":
			out = append(out, fmt.Sprintf("%s%c%d:%s", rec.Path, sep, rec.Line, lineText))
		case rec.Stdin:
			if rec.Kind == "match" {
				out = append(out, fmt.Sprintf("%d:%s", rec.Line, lineText))
			} else {
				out = append(out, fmt.Sprintf("%d-%s", rec.Line, lineText))
			}
		}
	}
	return out
}
