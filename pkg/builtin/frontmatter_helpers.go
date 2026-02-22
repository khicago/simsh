package builtin

import "strings"

type frontMatterBlock struct {
	HasFrontMatter bool
	StartLine      int
	EndLine        int
	Lines          []string
	HeaderLines    []string
	BodyLines      []string
}

func parseMarkdownFrontMatter(raw string) frontMatterBlock {
	lines := splitRawLines(raw)
	block := frontMatterBlock{Lines: append([]string(nil), lines...)}
	if len(lines) < 3 {
		return block
	}
	if strings.TrimSpace(lines[0]) != "---" {
		return block
	}
	end := -1
	for idx := 1; idx < len(lines); idx++ {
		if strings.TrimSpace(lines[idx]) == "---" {
			end = idx
			break
		}
	}
	if end <= 0 {
		return block
	}
	block.HasFrontMatter = true
	block.StartLine = 1
	block.EndLine = end + 1
	block.HeaderLines = append([]string(nil), lines[1:end]...)
	if end+1 < len(lines) {
		block.BodyLines = append([]string(nil), lines[end+1:]...)
	}
	return block
}

func stripMarkdownFrontMatter(raw string) string {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return ""
	}
	block := parseMarkdownFrontMatter(trimmed)
	if !block.HasFrontMatter {
		return trimmed
	}
	return strings.TrimSpace(strings.Join(block.BodyLines, "\n"))
}

func frontMatterKeys(block frontMatterBlock) []string {
	if !block.HasFrontMatter {
		return nil
	}
	seen := map[string]struct{}{}
	keys := make([]string, 0, len(block.HeaderLines))
	for _, line := range block.HeaderLines {
		if key, ok := parseTopLevelFrontMatterKey(line); ok {
			if _, exists := seen[key]; exists {
				continue
			}
			seen[key] = struct{}{}
			keys = append(keys, key)
		}
	}
	return keys
}

func findFrontMatterKeyRange(block frontMatterBlock, key string) (int, int, bool) {
	if !block.HasFrontMatter {
		return 0, 0, false
	}
	needle := strings.TrimSpace(key)
	if needle == "" {
		return 0, 0, false
	}
	for idx := 0; idx < len(block.HeaderLines); idx++ {
		currentKey, ok := parseTopLevelFrontMatterKey(block.HeaderLines[idx])
		if !ok || currentKey != needle {
			continue
		}
		startLine := block.StartLine + idx + 1
		endLine := block.EndLine - 1
		for next := idx + 1; next < len(block.HeaderLines); next++ {
			if _, isKey := parseTopLevelFrontMatterKey(block.HeaderLines[next]); isKey {
				endLine = block.StartLine + next
				break
			}
		}
		return startLine, endLine, true
	}
	return 0, 0, false
}

func parseTopLevelFrontMatterKey(line string) (string, bool) {
	if strings.TrimSpace(line) == "" {
		return "", false
	}
	if strings.HasPrefix(line, " ") || strings.HasPrefix(line, "\t") {
		return "", false
	}
	trimmed := strings.TrimSpace(line)
	if strings.HasPrefix(trimmed, "#") || strings.HasPrefix(trimmed, "-") {
		return "", false
	}
	idx := strings.Index(trimmed, ":")
	if idx <= 0 {
		return "", false
	}
	key := strings.TrimSpace(trimmed[:idx])
	if key == "" || strings.Contains(key, " ") {
		return "", false
	}
	return key, true
}

func dedentLines(lines []string) []string {
	if len(lines) == 0 {
		return nil
	}
	minIndent := -1
	for _, line := range lines {
		if strings.TrimSpace(line) == "" {
			continue
		}
		indent := countLeadingSpaces(line)
		if minIndent < 0 || indent < minIndent {
			minIndent = indent
		}
	}
	if minIndent <= 0 {
		return append([]string(nil), lines...)
	}
	out := make([]string, 0, len(lines))
	for _, line := range lines {
		if len(line) < minIndent {
			out = append(out, "")
			continue
		}
		out = append(out, line[minIndent:])
	}
	return out
}

func countLeadingSpaces(line string) int {
	count := 0
	for i := 0; i < len(line); i++ {
		if line[i] == ' ' {
			count++
			continue
		}
		break
	}
	return count
}
