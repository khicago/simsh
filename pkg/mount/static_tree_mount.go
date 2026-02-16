package mount

import (
	"context"
	"fmt"
	"path"
	"sort"
	"strings"

	"github.com/khicago/simsh/pkg/contract"
)

type staticMount struct {
	point      string
	kindPrefix string
	files      map[string]string
	dirs       map[string]struct{}
	children   map[string][]string
}

func NewStaticMount(mountPoint string, kindPrefix string, files map[string]string) (contract.VirtualMount, error) {
	point := normalizeAbsPath(mountPoint)
	if point == "/" {
		return nil, fmt.Errorf("static mount point must not be root")
	}
	normalizedFiles := make(map[string]string, len(files))
	for filePath, raw := range files {
		normalizedPath := normalizeAbsPath(filePath)
		if normalizedPath == point || !strings.HasPrefix(normalizedPath, point+"/") {
			return nil, fmt.Errorf("file %s is outside mount point %s", filePath, point)
		}
		normalizedFiles[normalizedPath] = raw
	}
	if len(normalizedFiles) == 0 {
		return nil, fmt.Errorf("static mount %s has no files", point)
	}
	dirs := map[string]struct{}{point: {}}
	for filePath := range normalizedFiles {
		dir := path.Dir(filePath)
		for {
			dirs[dir] = struct{}{}
			if dir == point {
				break
			}
			dir = path.Dir(dir)
		}
	}
	childSet := map[string]map[string]struct{}{}
	for dirPath := range dirs {
		// Do not overwrite existing entries. Parent directories may already
		// contain children discovered while visiting other dirs.
		if _, ok := childSet[dirPath]; !ok {
			childSet[dirPath] = map[string]struct{}{}
		}
		if dirPath != point {
			parent := path.Dir(dirPath)
			if _, ok := dirs[parent]; ok {
				if _, ok := childSet[parent]; !ok {
					childSet[parent] = map[string]struct{}{}
				}
				childSet[parent][dirPath] = struct{}{}
			}
		}
	}
	for filePath := range normalizedFiles {
		parent := path.Dir(filePath)
		if _, ok := childSet[parent]; !ok {
			childSet[parent] = map[string]struct{}{}
		}
		childSet[parent][filePath] = struct{}{}
	}
	children := make(map[string][]string, len(childSet))
	for dirPath, set := range childSet {
		list := make([]string, 0, len(set))
		for child := range set {
			list = append(list, child)
		}
		sort.Strings(list)
		children[dirPath] = list
	}
	if strings.TrimSpace(kindPrefix) == "" {
		kindPrefix = "mount"
	}
	return &staticMount{point: point, kindPrefix: kindPrefix, files: normalizedFiles, dirs: dirs, children: children}, nil
}

func (m *staticMount) MountPoint() string {
	return m.point
}

func (m *staticMount) Exists(ctx context.Context) (bool, error) {
	_ = ctx
	return len(m.files) > 0, nil
}

func (m *staticMount) ListChildren(ctx context.Context, dir string) ([]string, error) {
	_ = ctx
	dir = normalizeAbsPath(dir)
	if _, ok := m.dirs[dir]; !ok {
		return nil, fmt.Errorf("%s: No such file or directory", dir)
	}
	children := m.children[dir]
	if len(children) == 0 {
		return []string{}, nil
	}
	out := make([]string, len(children))
	copy(out, children)
	return out, nil
}

func (m *staticMount) IsDirPath(ctx context.Context, pathValue string) (bool, error) {
	_ = ctx
	pathValue = normalizeAbsPath(pathValue)
	_, ok := m.dirs[pathValue]
	return ok, nil
}

func (m *staticMount) ReadRawContent(ctx context.Context, pathValue string) (string, error) {
	_ = ctx
	pathValue = normalizeAbsPath(pathValue)
	raw, ok := m.files[pathValue]
	if !ok {
		return "", fmt.Errorf("%s: No such file or directory", pathValue)
	}
	return raw, nil
}

func (m *staticMount) CollectFilesUnder(ctx context.Context, target string) ([]string, error) {
	_ = ctx
	target = normalizeAbsPath(target)
	if _, ok := m.files[target]; ok {
		return []string{target}, nil
	}
	if _, ok := m.dirs[target]; !ok {
		return nil, fmt.Errorf("%s: No such file or directory", target)
	}
	files := make([]string, 0)
	prefix := target + "/"
	for filePath := range m.files {
		if target == m.point || strings.HasPrefix(filePath, prefix) {
			files = append(files, filePath)
		}
	}
	sort.Strings(files)
	return files, nil
}

func (m *staticMount) ResolveSearchPaths(ctx context.Context, target string, recursive bool) ([]string, error) {
	target = normalizeAbsPath(target)
	if _, ok := m.files[target]; ok {
		return []string{target}, nil
	}
	if _, ok := m.dirs[target]; !ok {
		return nil, fmt.Errorf("%s: No such file or directory", target)
	}
	if !recursive {
		return nil, fmt.Errorf("%s: Is a directory (use -r to search recursively)", target)
	}
	return m.CollectFilesUnder(ctx, target)
}

func (m *staticMount) DescribePath(ctx context.Context, pathValue string) (contract.PathMeta, error) {
	_ = ctx
	pathValue = normalizeAbsPath(pathValue)
	if _, ok := m.dirs[pathValue]; ok {
		return contract.PathMeta{Exists: true, IsDir: true, Kind: m.kindPrefix + "_dir", LineCount: -1, FrontMatterLines: -1, SpeakerRows: -1, UserRelevance: "n/a"}, nil
	}
	if raw, ok := m.files[pathValue]; ok {
		kind := m.kindPrefix + "_file"
		if strings.HasSuffix(pathValue, ".sh") {
			kind = m.kindPrefix + "_script"
		}
		return contract.PathMeta{Exists: true, IsDir: false, Kind: kind, LineCount: len(splitRawLines(raw)), FrontMatterLines: -1, SpeakerRows: -1, UserRelevance: "n/a"}, nil
	}
	return contract.PathMeta{}, fmt.Errorf("%s: No such file or directory", pathValue)
}
