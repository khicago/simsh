package mount

import (
	"path"
	"strings"
)

func normalizeAbsPath(raw string) string {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return "/"
	}
	cleaned := path.Clean(trimmed)
	if !strings.HasPrefix(cleaned, "/") {
		cleaned = "/" + cleaned
	}
	return cleaned
}

func isExecutableUnder(pathValue string, dir string) bool {
	pathValue = normalizeAbsPath(pathValue)
	dir = normalizeAbsPath(dir)
	if !strings.HasPrefix(pathValue, dir+"/") {
		return false
	}
	name := strings.TrimPrefix(pathValue, dir+"/")
	return strings.TrimSpace(name) != "" && !strings.Contains(name, "/")
}

func normalizeCommandName(raw string) string {
	trimmed := strings.TrimSpace(raw)
	trimmed = strings.TrimPrefix(trimmed, "/")
	if strings.Contains(trimmed, "/") {
		return ""
	}
	return trimmed
}

func splitRawLines(raw string) []string {
	raw = strings.TrimSuffix(raw, "\n")
	if raw == "" {
		return []string{}
	}
	return strings.Split(raw, "\n")
}
