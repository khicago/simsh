package pathguard

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
)

var ErrEscape = errors.New("path escape is not allowed")

// Check ensures candidate stays within root after resolving any existing
// symlinks along both paths. Missing descendants are allowed as long as the
// deepest existing ancestor still resolves under root.
func Check(root string, candidate string) error {
	cleanRoot := filepath.Clean(strings.TrimSpace(root))
	cleanCandidate := filepath.Clean(strings.TrimSpace(candidate))
	if !withinRoot(cleanRoot, cleanCandidate) {
		return ErrEscape
	}

	resolvedRoot, err := resolveWithExistingPrefix(cleanRoot)
	if err != nil {
		return err
	}
	resolvedCandidate, err := resolveWithExistingPrefix(cleanCandidate)
	if err != nil {
		return err
	}
	if !withinRoot(resolvedRoot, resolvedCandidate) {
		return ErrEscape
	}
	return nil
}

func resolveWithExistingPrefix(pathValue string) (string, error) {
	cleaned := filepath.Clean(strings.TrimSpace(pathValue))
	resolved, err := filepath.EvalSymlinks(cleaned)
	if err == nil {
		return filepath.Clean(resolved), nil
	}
	if !os.IsNotExist(err) {
		return "", err
	}

	suffix := make([]string, 0)
	current := cleaned
	for {
		_, statErr := os.Lstat(current)
		if statErr == nil {
			resolvedCurrent, evalErr := filepath.EvalSymlinks(current)
			if evalErr != nil {
				return "", evalErr
			}
			resolvedCurrent = filepath.Clean(resolvedCurrent)
			for i := len(suffix) - 1; i >= 0; i-- {
				resolvedCurrent = filepath.Join(resolvedCurrent, suffix[i])
			}
			return filepath.Clean(resolvedCurrent), nil
		}
		if !os.IsNotExist(statErr) {
			return "", statErr
		}
		parent := filepath.Dir(current)
		if parent == current {
			return cleaned, nil
		}
		suffix = append(suffix, filepath.Base(current))
		current = parent
	}
}

func withinRoot(root string, candidate string) bool {
	root = filepath.Clean(root)
	candidate = filepath.Clean(candidate)
	rel, err := filepath.Rel(root, candidate)
	if err != nil {
		return false
	}
	return rel == "." || (rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)))
}
