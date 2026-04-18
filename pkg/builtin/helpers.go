package builtin

import (
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/khicago/simsh/pkg/contract"
)

var hostPathInError = regexp.MustCompile(`(?:/Users/|/home/|/private/|/var/folders/|/tmp/)[^\s:]*`)

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

func formatCommandPathError(command, virtualPath, fallback string, err error) string {
	if err == nil {
		return command + ": unknown error"
	}
	var mountErr *contract.MountUnsupportedError
	if errors.As(err, &mountErr) {
		return fmt.Sprintf("%s: %s", command, mountErr.Error())
	}
	if errors.Is(err, contract.ErrUnsupported) {
		if fallback != "" {
			return fallback
		}
		return command + ": not supported"
	}
	if isPathMissing(err) {
		return fmt.Sprintf("%s: %s: No such file or directory", command, virtualPath)
	}
	msg := strings.TrimSpace(err.Error())
	if hostPathInError.MatchString(msg) {
		msg = strings.TrimSpace(hostPathInError.ReplaceAllString(msg, virtualPath))
	}
	return fmt.Sprintf("%s: %s", command, msg)
}

func matchLineNumbers(raw, old string) []int {
	if old == "" {
		return nil
	}
	lines := make([]int, 0)
	start := 0
	for start <= len(raw) {
		idx := strings.Index(raw[start:], old)
		if idx < 0 {
			break
		}
		abs := start + idx
		lines = append(lines, 1+strings.Count(raw[:abs], "\n"))
		if old == "" {
			break
		}
		start = abs + len(old)
	}
	return lines
}

func formatLineList(lines []int) string {
	if len(lines) == 0 {
		return ""
	}
	parts := make([]string, 0, len(lines))
	for _, line := range lines {
		parts = append(parts, strconv.Itoa(line))
	}
	return strings.Join(parts, ", ")
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
