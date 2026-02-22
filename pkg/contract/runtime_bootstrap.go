package contract

import "strings"

const (
	defaultAliasLL = "ll"
	defaultAliasFM = "fm"
)

// DefaultCommandAliases are always available unless explicitly overridden.
func DefaultCommandAliases() map[string][]string {
	return map[string][]string{
		defaultAliasLL: {"ls", "-l"},
		defaultAliasFM: {"frontmatter"},
	}
}

// NormalizeCommandAliases trims keys/tokens and drops invalid entries.
func NormalizeCommandAliases(raw map[string][]string) map[string][]string {
	if len(raw) == 0 {
		return map[string][]string{}
	}
	out := map[string][]string{}
	for key, expansion := range raw {
		name := normalizeAliasName(key)
		if name == "" {
			continue
		}
		tokens := normalizeAliasExpansion(expansion)
		if len(tokens) == 0 {
			continue
		}
		out[name] = tokens
	}
	return out
}

// MergeCommandAliases merges overlays from left to right.
func MergeCommandAliases(maps ...map[string][]string) map[string][]string {
	out := map[string][]string{}
	for _, aliasMap := range maps {
		for name, tokens := range NormalizeCommandAliases(aliasMap) {
			out[name] = append([]string(nil), tokens...)
		}
	}
	return out
}

func normalizeAliasName(raw string) string {
	name := strings.TrimSpace(raw)
	if name == "" {
		return ""
	}
	if strings.HasPrefix(name, "-") || strings.Contains(name, "/") || strings.Contains(name, " ") {
		return ""
	}
	return name
}

func normalizeAliasExpansion(tokens []string) []string {
	out := make([]string, 0, len(tokens))
	for _, raw := range tokens {
		t := strings.TrimSpace(raw)
		if t == "" {
			continue
		}
		out = append(out, t)
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

// NormalizeEnvVars keeps valid KEY=VALUE entries.
func NormalizeEnvVars(raw map[string]string) map[string]string {
	if len(raw) == 0 {
		return map[string]string{}
	}
	out := map[string]string{}
	for key, value := range raw {
		k := normalizeEnvKey(key)
		if k == "" {
			continue
		}
		out[k] = value
	}
	return out
}

func normalizeEnvKey(raw string) string {
	k := strings.TrimSpace(raw)
	if k == "" {
		return ""
	}
	for idx, r := range k {
		switch {
		case r == '_' || (r >= 'A' && r <= 'Z') || (r >= 'a' && r <= 'z'):
			// OK
		case idx > 0 && r >= '0' && r <= '9':
			// OK
		default:
			return ""
		}
	}
	return k
}

// NormalizeRCFiles trims and deduplicates RC paths.
func NormalizeRCFiles(raw []string) []string {
	if len(raw) == 0 {
		return nil
	}
	seen := map[string]struct{}{}
	out := make([]string, 0, len(raw))
	for _, item := range raw {
		path := strings.TrimSpace(item)
		if path == "" {
			continue
		}
		if _, ok := seen[path]; ok {
			continue
		}
		seen[path] = struct{}{}
		out = append(out, path)
	}
	return out
}
