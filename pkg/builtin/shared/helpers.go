package shared

import (
	"fmt"
	"strconv"
	"strings"
)

func SplitRawLines(raw string) []string {
	raw = strings.TrimSuffix(raw, "\n")
	if raw == "" {
		return []string{}
	}
	return strings.Split(raw, "\n")
}

func FormatWithLineNumbers(raw string) string {
	lines := SplitRawLines(raw)
	if len(lines) == 0 {
		return ""
	}
	out := make([]string, 0, len(lines))
	for idx, line := range lines {
		out = append(out, strings.Join([]string{strconv.Itoa(idx + 1), ":", line}, ""))
	}
	return strings.Join(out, "\n")
}

func ParseNonNegativeInt(raw string) (int, error) {
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

func IsDigits(s string) bool {
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

