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
	profile    contract.MountProfile
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
		if _, ok := childSet[dirPath]; !ok {
			childSet[dirPath] = map[string]struct{}{}
		}
		if dirPath == point {
			continue
		}
		parent := path.Dir(dirPath)
		if _, ok := childSet[parent]; !ok {
			childSet[parent] = map[string]struct{}{}
		}
		childSet[parent][dirPath] = struct{}{}
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
	return &staticMount{
		point:      point,
		kindPrefix: kindPrefix,
		profile:    defaultStaticMountProfile(),
		files:      normalizedFiles,
		dirs:       dirs,
		children:   children,
	}, nil
}

func (m *staticMount) MountPoint() string {
	return m.point
}

func (m *staticMount) Profile() contract.MountProfile {
	return m.profile
}

func (m *staticMount) Exists(ctx context.Context) (bool, error) {
	_ = ctx
	return len(m.files) > 0, nil
}

func (m *staticMount) StatPath(ctx context.Context, pathValue string) (contract.MountEntry, error) {
	_ = ctx
	pathValue = normalizeAbsPath(pathValue)
	if _, ok := m.dirs[pathValue]; ok {
		return mountEntry(pathValue, staticDirMeta(m.kindPrefix)), nil
	}
	if raw, ok := m.files[pathValue]; ok {
		return mountEntry(pathValue, staticFileMeta(m.kindPrefix, pathValue, raw)), nil
	}
	return contract.MountEntry{}, fmt.Errorf("%s: No such file or directory", pathValue)
}

func (m *staticMount) ListEntries(ctx context.Context, req contract.ListEntriesRequest) (contract.ListEntriesResult, error) {
	_ = ctx
	dir := normalizeAbsPath(req.Dir)
	if _, ok := m.dirs[dir]; !ok {
		return contract.ListEntriesResult{}, fmt.Errorf("%s: No such file or directory", dir)
	}
	if !req.Recursive {
		children := m.children[dir]
		entries := make([]contract.MountEntry, 0, len(children))
		for _, child := range children {
			entry, err := m.StatPath(ctx, child)
			if err != nil {
				return contract.ListEntriesResult{}, err
			}
			entries = append(entries, entry)
		}
		return contract.ListEntriesResult{Entries: entries}, nil
	}

	entries := make([]contract.MountEntry, 0)
	prefix := dir + "/"
	for dirPath := range m.dirs {
		if dirPath == dir || !strings.HasPrefix(dirPath, prefix) {
			continue
		}
		entry, err := m.StatPath(ctx, dirPath)
		if err != nil {
			return contract.ListEntriesResult{}, err
		}
		entries = append(entries, entry)
	}
	for filePath, raw := range m.files {
		if !strings.HasPrefix(filePath, prefix) {
			continue
		}
		entries = append(entries, mountEntry(filePath, staticFileMeta(m.kindPrefix, filePath, raw)))
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].Path < entries[j].Path })
	return contract.ListEntriesResult{Entries: entries}, nil
}

func (m *staticMount) EnumeratePaths(ctx context.Context, req contract.EnumeratePathsRequest) (contract.EnumeratePathsResult, error) {
	target := normalizeAbsPath(req.Target)
	if raw, ok := m.files[target]; ok {
		return contract.EnumeratePathsResult{Entries: []contract.MountEntry{mountEntry(target, staticFileMeta(m.kindPrefix, target, raw))}}, nil
	}
	if _, ok := m.dirs[target]; !ok {
		return contract.EnumeratePathsResult{}, fmt.Errorf("%s: No such file or directory", target)
	}
	if !req.Recursive {
		return contract.EnumeratePathsResult{}, fmt.Errorf("%s: Is a directory (use -r to search recursively)", target)
	}
	return enumerateEntriesFromLister(ctx, m, req)
}

func (m *staticMount) ReadContent(ctx context.Context, pathValue string) (string, error) {
	_ = ctx
	pathValue = normalizeAbsPath(pathValue)
	raw, ok := m.files[pathValue]
	if !ok {
		return "", fmt.Errorf("%s: No such file or directory", pathValue)
	}
	return raw, nil
}

func (m *staticMount) ReadMany(ctx context.Context, req contract.ReadManyRequest) (contract.ReadManyResult, error) {
	files := make([]contract.MountContentEntry, 0, len(req.Paths))
	for _, pathValue := range req.Paths {
		entry, err := m.ReadContent(ctx, pathValue)
		if err != nil {
			return contract.ReadManyResult{}, err
		}
		files = append(files, contract.MountContentEntry{Path: normalizeAbsPath(pathValue), Content: entry})
	}
	return contract.ReadManyResult{Entries: files}, nil
}

func (m *staticMount) SearchContent(ctx context.Context, req contract.SearchRequest) (contract.SearchResult, error) {
	match, err := mountSearchMatcher(req.Pattern, req.Regex, req.CaseMode)
	if err != nil {
		return contract.SearchResult{}, err
	}
	targets := req.Targets
	if len(targets) == 0 {
		targets = []string{m.point}
	}
	files := make([]contract.MountContentEntry, 0)
	for _, target := range targets {
		paths, err := m.EnumeratePaths(ctx, contract.EnumeratePathsRequest{Target: target, Recursive: true})
		if err != nil {
			return contract.SearchResult{}, err
		}
		pathValues := make([]string, 0, len(paths.Entries))
		for _, entry := range paths.Entries {
			pathValues = append(pathValues, entry.Path)
		}
		pathValues, err = filterMountPathsByGlob(pathValues, req.Globs)
		if err != nil {
			return contract.SearchResult{}, err
		}
		readMany, err := m.ReadMany(ctx, contract.ReadManyRequest{Paths: pathValues})
		if err != nil {
			return contract.SearchResult{}, err
		}
		files = append(files, readMany.Entries...)
	}
	if req.ListFiles {
		records := make([]contract.SearchRecord, 0)
		for _, file := range files {
			if mountSearchHasMatch(file.Content, match) {
				records = append(records, contract.SearchRecord{Path: file.Path, Kind: "file"})
			}
		}
		return contract.SearchResult{Records: records}, nil
	}
	records := make([]contract.SearchRecord, 0)
	for _, file := range files {
		records = append(records, mountSearchRecords(file.Content, match, req.Before, req.After, file.Path)...)
	}
	if req.MaxResults > 0 && len(records) > req.MaxResults {
		records = records[:req.MaxResults]
	}
	return contract.SearchResult{Records: records}, nil
}
