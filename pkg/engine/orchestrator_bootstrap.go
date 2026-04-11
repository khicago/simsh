package engine

import (
	"context"
	"fmt"
	"strings"

	"github.com/khicago/simsh/pkg/contract"
)

func (e *Engine) withRuntimeBootstrap(ctx context.Context, ops contract.Ops) (contract.Ops, error) {
	paths := contract.NormalizeRCFiles(ops.RCFiles)
	if len(paths) == 0 {
		return ops, nil
	}
	aliases := contract.NormalizeCommandAliases(ops.CommandAliases)
	envVars := contract.NormalizeEnvVars(ops.EnvVars)
	for _, rawPath := range paths {
		pathValue, err := ops.RequireAbsolutePath(rawPath)
		if err != nil {
			return ops, fmt.Errorf("rc: %s: %v", rawPath, err)
		}
		raw, err := ops.ReadRawContent(ctx, pathValue)
		if err != nil {
			if isNotFoundLikeError(err) {
				continue
			}
			return ops, fmt.Errorf("rc: read %s failed: %v", pathValue, err)
		}
		rcAliases, rcEnv, err := parseRCContent(raw)
		if err != nil {
			return ops, fmt.Errorf("rc: parse %s failed: %v", pathValue, err)
		}
		aliases = contract.MergeCommandAliases(aliases, rcAliases)
		for key, value := range rcEnv {
			envVars[key] = value
		}
	}
	ops.CommandAliases = aliases
	ops.EnvVars = envVars
	return ops, nil
}

func parseRCContent(raw string) (map[string][]string, map[string]string, error) {
	aliases := map[string][]string{}
	envVars := map[string]string{}
	raw = strings.TrimSuffix(raw, "\n")
	lines := []string{}
	if raw != "" {
		lines = strings.Split(raw, "\n")
	}
	for idx, line := range lines {
		lineNo := idx + 1
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}

		switch {
		case strings.HasPrefix(trimmed, "export "):
			key, value, err := parseExportLine(strings.TrimSpace(strings.TrimPrefix(trimmed, "export ")))
			if err != nil {
				return nil, nil, fmt.Errorf("line %d: %v", lineNo, err)
			}
			envVars[key] = value
		case strings.HasPrefix(trimmed, "alias "):
			name, expansion, err := parseAliasLine(strings.TrimSpace(strings.TrimPrefix(trimmed, "alias ")))
			if err != nil {
				return nil, nil, fmt.Errorf("line %d: %v", lineNo, err)
			}
			aliases[name] = expansion
		default:
			return nil, nil, fmt.Errorf("line %d: unsupported statement %q", lineNo, trimmed)
		}
	}
	return aliases, envVars, nil
}

func parseExportLine(raw string) (string, string, error) {
	name, value, ok := splitAssignment(raw)
	if !ok {
		return "", "", fmt.Errorf("export requires KEY=VALUE")
	}
	key := normalizeEnvKey(name)
	if key == "" {
		return "", "", fmt.Errorf("invalid export key %q", name)
	}
	return key, trimOptionalQuotes(value), nil
}

func parseAliasLine(raw string) (string, []string, error) {
	name, value, ok := splitAssignment(raw)
	if !ok {
		return "", nil, fmt.Errorf("alias requires NAME=VALUE")
	}
	name = strings.TrimSpace(name)
	if strings.Contains(name, "/") || strings.HasPrefix(name, "-") || strings.Contains(name, " ") || name == "" {
		return "", nil, fmt.Errorf("invalid alias name %q", name)
	}
	expansion, err := parseAliasExpansion(trimOptionalQuotes(value))
	if err != nil {
		return "", nil, err
	}
	return name, expansion, nil
}

func parseAliasExpansion(raw string) ([]string, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return nil, fmt.Errorf("alias expansion must not be empty")
	}
	out := make([]string, 0, 4)
	for idx := 0; idx < len(trimmed); {
		idx = skipInlineSpaces(trimmed, idx)
		if idx >= len(trimmed) {
			break
		}
		word, next, err := readShellWord(trimmed, idx)
		if err != nil {
			return nil, fmt.Errorf("invalid alias expansion: %v", err)
		}
		if strings.TrimSpace(word) == "" {
			return nil, fmt.Errorf("alias expansion must not contain empty token")
		}
		out = append(out, word)
		idx = next
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("alias expansion must not be empty")
	}
	return out, nil
}

func splitAssignment(raw string) (string, string, bool) {
	parts := strings.SplitN(strings.TrimSpace(raw), "=", 2)
	if len(parts) != 2 {
		return "", "", false
	}
	key := strings.TrimSpace(parts[0])
	if key == "" {
		return "", "", false
	}
	return key, strings.TrimSpace(parts[1]), true
}

func trimOptionalQuotes(raw string) string {
	if len(raw) >= 2 {
		if (raw[0] == '"' && raw[len(raw)-1] == '"') || (raw[0] == '\'' && raw[len(raw)-1] == '\'') {
			return raw[1 : len(raw)-1]
		}
	}
	return raw
}

func normalizeEnvKey(raw string) string {
	name := strings.TrimSpace(raw)
	if name == "" {
		return ""
	}
	for idx, r := range name {
		switch {
		case r == '_' || (r >= 'A' && r <= 'Z') || (r >= 'a' && r <= 'z'):
		case idx > 0 && r >= '0' && r <= '9':
		default:
			return ""
		}
	}
	return name
}

func isNotFoundLikeError(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(strings.TrimSpace(err.Error()))
	return strings.Contains(msg, "no such file") || strings.Contains(msg, "not found")
}
