package mount

import (
	"context"
	"fmt"
	"path"
	"regexp"
	"strings"

	"github.com/khicago/simsh/pkg/contract"
)

func defaultStaticMountProfile() contract.MountProfile {
	return contract.NormalizeMountProfile(contract.MountProfile{
		TruthModel:          contract.MountTruthProjection,
		MaterializationMode: contract.MountMaterializationSnapshot,
		WriteSemantics:      contract.MountWriteReadOnly,
		LatencyClass:        contract.MountLatencyLocalFast,
		SupportedCLIClasses: []contract.MountCLIClass{
			contract.MountCLIList,
			contract.MountCLITree,
			contract.MountCLIFind,
			contract.MountCLIRead,
			contract.MountCLIBulkRead,
			contract.MountCLIContentSearch,
		},
	})
}

func staticDirMeta(kindPrefix string) contract.PathMeta {
	return contract.PathMeta{
		Exists:           true,
		IsDir:            true,
		Kind:             kindPrefix + "_dir",
		Access:           contract.PathAccessReadOnly,
		Capabilities:     []string{contract.PathCapabilityDescribe, contract.PathCapabilityList, contract.PathCapabilitySearch},
		LineCount:        -1,
		FrontMatterLines: -1,
		SpeakerRows:      -1,
		UserRelevance:    "n/a",
	}
}

func staticFileMeta(kindPrefix string, filePath string, raw string) contract.PathMeta {
	kind := kindPrefix + "_file"
	if strings.HasSuffix(filePath, ".sh") {
		kind = kindPrefix + "_script"
	}
	return contract.PathMeta{
		Exists:           true,
		IsDir:            false,
		Kind:             kind,
		Access:           contract.PathAccessReadOnly,
		Capabilities:     []string{contract.PathCapabilityDescribe, contract.PathCapabilityRead},
		LineCount:        len(splitRawLines(raw)),
		FrontMatterLines: -1,
		SpeakerRows:      -1,
		UserRelevance:    "n/a",
	}
}

func mountEntry(pathValue string, meta contract.PathMeta) contract.MountEntry {
	return contract.MountEntry{
		Path: pathValue,
		Name: path.Base(pathValue),
		Meta: meta,
	}
}

func mountSearchMatcher(pattern string, regex bool, caseMode contract.SearchCaseMode) (func(string) bool, error) {
	effective := caseMode
	if effective == "" {
		effective = contract.SearchCaseSensitive
	}
	if effective == contract.SearchCaseSmart {
		if mountPatternHasUpper(pattern) {
			effective = contract.SearchCaseSensitive
		} else {
			effective = contract.SearchCaseIgnore
		}
	}
	if regex {
		if effective == contract.SearchCaseIgnore {
			pattern = "(?i)" + pattern
		}
		re, err := regexp.Compile(pattern)
		if err != nil {
			return nil, err
		}
		return re.MatchString, nil
	}
	if effective == contract.SearchCaseIgnore {
		lowerPattern := strings.ToLower(pattern)
		return func(line string) bool {
			return strings.Contains(strings.ToLower(line), lowerPattern)
		}, nil
	}
	return func(line string) bool {
		return strings.Contains(line, pattern)
	}, nil
}

func mountPatternHasUpper(pattern string) bool {
	for _, r := range pattern {
		if r >= 'A' && r <= 'Z' {
			return true
		}
	}
	return false
}

func mountSearchRecords(raw string, match func(string) bool, before int, after int, filePath string) []contract.SearchRecord {
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
	out := make([]contract.SearchRecord, 0)
	for i, line := range lines {
		if !include[i] {
			continue
		}
		record := contract.SearchRecord{
			Path: filePath,
			Line: i + 1,
			Kind: "context",
			Text: line,
		}
		if matched[i] {
			record.Kind = "match"
		}
		out = append(out, record)
	}
	return out
}

func mountSearchHasMatch(raw string, match func(string) bool) bool {
	for _, line := range splitRawLines(raw) {
		if match(line) {
			return true
		}
	}
	return false
}

func matchMountGlobs(filePath string, globs []string) (bool, error) {
	if len(globs) == 0 {
		return true, nil
	}
	fullPath := strings.TrimPrefix(filePath, "/")
	baseName := path.Base(filePath)
	for _, glob := range globs {
		pattern := strings.TrimSpace(glob)
		if pattern == "" {
			continue
		}
		pattern = strings.TrimPrefix(pattern, "/")
		ok, err := path.Match(pattern, baseName)
		if err != nil {
			return false, fmt.Errorf("invalid glob pattern: %v", err)
		}
		if ok {
			return true, nil
		}
		ok, err = path.Match(pattern, fullPath)
		if err != nil {
			return false, fmt.Errorf("invalid glob pattern: %v", err)
		}
		if ok {
			return true, nil
		}
	}
	return false, nil
}

func filterMountPathsByGlob(paths []string, globs []string) ([]string, error) {
	if len(globs) == 0 {
		return append([]string(nil), paths...), nil
	}
	filtered := make([]string, 0, len(paths))
	for _, pathValue := range paths {
		ok, err := matchMountGlobs(pathValue, globs)
		if err != nil {
			return nil, err
		}
		if ok {
			filtered = append(filtered, pathValue)
		}
	}
	return filtered, nil
}

func enumerateEntriesFromLister(ctx context.Context, lister contract.EntryLister, req contract.EnumeratePathsRequest) (contract.EnumeratePathsResult, error) {
	entries, err := lister.ListEntries(ctx, contract.ListEntriesRequest{
		Dir:       req.Target,
		Recursive: req.Recursive,
		MaxDepth:  req.MaxDepth,
	})
	if err != nil {
		return contract.EnumeratePathsResult{}, err
	}
	files := make([]contract.MountEntry, 0, len(entries.Entries))
	for _, entry := range entries.Entries {
		if entry.Meta.IsDir {
			continue
		}
		files = append(files, entry)
	}
	return contract.EnumeratePathsResult{Entries: files}, nil
}
