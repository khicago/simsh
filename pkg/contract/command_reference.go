package contract

import (
	"fmt"
	"path"
	"strings"
)

const (
	CommandNamespaceBuiltin  = "builtin"
	CommandNamespaceExternal = "external"
)

type CommandReference struct {
	Raw          string
	Name         string
	ResolvedPath string
	PathLike     bool
	Namespace    string
}

type CommandReferencePathError struct {
	Raw          string
	ResolvedPath string
}

func (e CommandReferencePathError) Error() string {
	raw := strings.TrimSpace(e.Raw)
	resolved := strings.TrimSpace(e.ResolvedPath)
	if resolved == "" {
		return fmt.Sprintf("%s refers to a path-like command reference that cannot be resolved", raw)
	}
	return fmt.Sprintf("%s refers to %s, which is not a command path; use a command name or a path under %s or %s", raw, resolved, VirtualSystemBinDir, VirtualExternalBinDir)
}

func NormalizeCommandReference(raw string, resolvePath func(string) (string, error)) (CommandReference, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return CommandReference{}, fmt.Errorf("command reference is required")
	}
	if !looksLikeCommandPath(trimmed) {
		return CommandReference{
			Raw:      trimmed,
			Name:     trimmed,
			PathLike: false,
		}, nil
	}
	if resolvePath == nil {
		return CommandReference{}, fmt.Errorf("%s: path-like command references are not supported", trimmed)
	}
	resolved, err := resolvePath(trimmed)
	if err != nil {
		return CommandReference{}, err
	}
	ref := CommandReference{
		Raw:          trimmed,
		Name:         path.Base(resolved),
		ResolvedPath: resolved,
		PathLike:     true,
	}
	switch {
	case IsCommandPathUnder(resolved, VirtualSystemBinDir):
		ref.Name = strings.TrimPrefix(resolved, VirtualSystemBinDir+"/")
		ref.Namespace = CommandNamespaceBuiltin
		return ref, nil
	case IsCommandPathUnder(resolved, VirtualExternalBinDir):
		ref.Name = strings.TrimPrefix(resolved, VirtualExternalBinDir+"/")
		ref.Namespace = CommandNamespaceExternal
		return ref, nil
	default:
		return ref, CommandReferencePathError{Raw: trimmed, ResolvedPath: resolved}
	}
}

func IsCommandPathUnder(pathValue string, dir string) bool {
	pathValue = normalizeVirtualCommandPath(pathValue)
	dir = normalizeVirtualCommandPath(dir)
	if !strings.HasPrefix(pathValue, dir+"/") {
		return false
	}
	name := strings.TrimPrefix(pathValue, dir+"/")
	return strings.TrimSpace(name) != "" && !strings.Contains(name, "/")
}

func normalizeVirtualCommandPath(raw string) string {
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

func looksLikeCommandPath(raw string) bool {
	trimmed := strings.TrimSpace(raw)
	return strings.Contains(trimmed, "/")
}
