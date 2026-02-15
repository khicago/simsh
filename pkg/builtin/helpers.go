package builtin

import (
	"fmt"
	"strconv"
	"strings"
)

func splitRawLines(raw string) []string {
	raw = strings.TrimSuffix(raw, "\n")
	if raw == "" {
		return []string{}
	}
	return strings.Split(raw, "\n")
}

func formatWithLineNumbers(raw string) string {
	lines := splitRawLines(raw)
	if len(lines) == 0 {
		return ""
	}
	out := make([]string, 0, len(lines))
	for idx, line := range lines {
		out = append(out, strings.Join([]string{strconv.Itoa(idx + 1), ":", line}, ""))
	}
	return strings.Join(out, "\n")
}

func parseNonNegativeInt(raw string) (int, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return 0, fmt.Errorf("count must not be empty")
	}
	parsed, err := strconv.Atoi(trimmed)
	if err != nil {
		return 0, fmt.Errorf("count must be integer")
	}
	if parsed < 0 {
		return 0, fmt.Errorf("count must be non-negative")
	}
	return parsed, nil
}

func isDigits(s string) bool {
	if strings.TrimSpace(s) == "" {
		return false
	}
	for _, r := range s {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}
