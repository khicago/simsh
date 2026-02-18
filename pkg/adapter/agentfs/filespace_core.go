package agentfs

import (
	"context"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/khicago/simsh/pkg/contract"
)

const (
	VirtualTaskOutputRoot = "/task_outputs"
	VirtualTempWorkRoot   = "/temp_work"
	VirtualKnowledgeRoot  = "/knowledge_base"
)

var speakerLineRe = regexp.MustCompile(`^[^\s].{0,40}[:：]\s*.+$`)

type RelevanceEvaluator func(pathValue string, meta contract.PathMeta) string

type ZoneConfig struct {
	VirtualRoot string
	HostRoot    string
	Writable    bool
	Kind        string
}

type Options struct {
	Zones             []ZoneConfig
	PathEnv           []string
	RelevanceEval     RelevanceEvaluator
	EnsureHostRoots   bool
	DefaultDirPerm    os.FileMode
	DefaultFilePerm   os.FileMode
	WriteLimitedBytes int
}

type zone struct {
	virtualRoot string
	hostRoot    string
	writable    bool
	kind        string
}

type aiFilesystem struct {
	zones             []zone
	pathEnv           []string
	relevanceEval     RelevanceEvaluator
	dirPerm           os.FileMode
	filePerm          os.FileMode
	writeLimitedBytes int
}

func NewFilesystem(opts Options) (contract.Filesystem, error) {
	dirPerm := opts.DefaultDirPerm
	if dirPerm == 0 {
		dirPerm = 0o755
	}
	filePerm := opts.DefaultFilePerm
	if filePerm == 0 {
		filePerm = 0o644
	}

	zones := make([]zone, 0, len(opts.Zones))
	seen := map[string]struct{}{}
	for _, cfg := range opts.Zones {
		vr := normalizeVirtualPath(cfg.VirtualRoot)
		if vr == "/" {
			return nil, fmt.Errorf("zone virtual root must not be /")
		}
		hostRoot := strings.TrimSpace(cfg.HostRoot)
		if hostRoot == "" {
			return nil, fmt.Errorf("zone %s host root is required", vr)
		}
		absHost, err := filepath.Abs(hostRoot)
		if err != nil {
			return nil, fmt.Errorf("resolve host root for %s failed: %w", vr, err)
		}
		absHost = filepath.Clean(absHost)
		if _, ok := seen[vr]; ok {
			return nil, fmt.Errorf("duplicate zone %s", vr)
		}
		seen[vr] = struct{}{}
		if opts.EnsureHostRoots {
			if err := os.MkdirAll(absHost, dirPerm); err != nil {
				return nil, fmt.Errorf("create host root for %s failed: %w", vr, err)
			}
		}
		zones = append(zones, zone{virtualRoot: vr, hostRoot: absHost, writable: cfg.Writable, kind: strings.TrimSpace(cfg.Kind)})
	}

	if len(zones) == 0 {
		return nil, fmt.Errorf("at least one zone is required")
	}
	sort.SliceStable(zones, func(i, j int) bool {
		return len(zones[i].virtualRoot) > len(zones[j].virtualRoot)
	})
	return &aiFilesystem{
		zones:             zones,
		pathEnv:           dedupeNormalizedPaths(opts.PathEnv),
		relevanceEval:     opts.RelevanceEval,
		dirPerm:           dirPerm,
		filePerm:          filePerm,
		writeLimitedBytes: opts.WriteLimitedBytes,
	}, nil
}

func NewDefaultFilesystem(hostRoot string) (contract.Filesystem, error) {
	base := strings.TrimSpace(hostRoot)
	if base == "" {
		wd, err := os.Getwd()
		if err != nil {
			return nil, err
		}
		base = wd
	}
	base, err := filepath.Abs(base)
	if err != nil {
		return nil, err
	}
	return NewFilesystem(Options{
		Zones: []ZoneConfig{
			{VirtualRoot: VirtualTaskOutputRoot, HostRoot: filepath.Join(base, "task_outputs"), Writable: true, Kind: "task_output_dir"},
			{VirtualRoot: VirtualTempWorkRoot, HostRoot: filepath.Join(base, "temp_work"), Writable: true, Kind: "temp_work_dir"},
			{VirtualRoot: VirtualKnowledgeRoot, HostRoot: filepath.Join(base, "knowledge_base"), Writable: false, Kind: "knowledge_dir"},
		},
		EnsureHostRoots: true,
		PathEnv:         []string{VirtualTaskOutputRoot, VirtualTempWorkRoot, VirtualKnowledgeRoot},
	})
}

func (f *aiFilesystem) RootDir() string {
	return "/"
}

func (f *aiFilesystem) RequireAbsolutePath(raw string) (string, error) {
	p := strings.TrimSpace(raw)
	if p == "" {
		return "", fmt.Errorf("path is required")
	}
	if !strings.HasPrefix(p, "/") {
		return "", fmt.Errorf("path must be absolute: %s", raw)
	}
	p = normalizeVirtualPath(p)
	if p == "/" {
		return p, nil
	}
	if _, _, ok := f.resolveZone(p); !ok {
		return "", fmt.Errorf("path is outside allowed roots: %s", p)
	}
	return p, nil
}

func (f *aiFilesystem) ListChildren(ctx context.Context, dir string) ([]string, error) {
	_ = ctx
	dir = normalizeVirtualPath(dir)
	if dir == "/" {
		roots := make([]string, 0, len(f.zones))
		seen := map[string]struct{}{}
		for _, z := range f.zones {
			if _, ok := seen[z.virtualRoot]; ok {
				continue
			}
			seen[z.virtualRoot] = struct{}{}
			roots = append(roots, z.virtualRoot)
		}
		sort.Strings(roots)
		return roots, nil
	}
	z, hostPath, ok := f.resolveZone(dir)
	if !ok {
		return nil, fmt.Errorf("%s: No such file or directory", dir)
	}
	_ = z
	entries, err := os.ReadDir(hostPath)
	if err != nil {
		return nil, err
	}
	children := make([]string, 0, len(entries))
	for _, entry := range entries {
		children = append(children, path.Join(dir, entry.Name()))
	}
	sort.Strings(children)
	return children, nil
}

func (f *aiFilesystem) IsDirPath(ctx context.Context, pathValue string) (bool, error) {
	_ = ctx
	pathValue = normalizeVirtualPath(pathValue)
	if pathValue == "/" {
		return true, nil
	}
	if z, _, ok := f.resolveZone(pathValue); ok {
		if pathValue == z.virtualRoot {
			return true, nil
		}
	}
	_, hostPath, ok := f.resolveZone(pathValue)
	if !ok {
		return false, nil
	}
	info, err := os.Stat(hostPath)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	}
	return info.IsDir(), nil
}

func (f *aiFilesystem) ReadRawContent(ctx context.Context, pathValue string) (string, error) {
	_ = ctx
	_, hostPath, ok := f.resolveZone(pathValue)
	if !ok {
		return "", fmt.Errorf("%s: No such file or directory", pathValue)
	}
	raw, err := os.ReadFile(hostPath)
	if err != nil {
		return "", err
	}
	return string(raw), nil
}

func (f *aiFilesystem) ResolveSearchPaths(ctx context.Context, target string, recursive bool) ([]string, error) {
	_ = ctx
	isDir, err := f.IsDirPath(ctx, target)
	if err != nil {
		return nil, err
	}
	if !isDir {
		if _, _, ok := f.resolveZone(target); !ok {
			return nil, fmt.Errorf("%s: No such file or directory", target)
		}
		return []string{normalizeVirtualPath(target)}, nil
	}
	if !recursive {
		return nil, fmt.Errorf("%s: Is a directory (use -r to search recursively)", target)
	}
	return f.CollectFilesUnder(ctx, target)
}

func (f *aiFilesystem) CollectFilesUnder(ctx context.Context, target string) ([]string, error) {
	_ = ctx
	target = normalizeVirtualPath(target)
	if target == "/" {
		out := make([]string, 0)
		for _, z := range f.zones {
			zoneFiles, err := f.collectZoneFiles(z, z.virtualRoot)
			if err != nil {
				return nil, err
			}
			out = append(out, zoneFiles...)
		}
		sort.Strings(out)
		return dedupe(out), nil
	}
	z, _, ok := f.resolveZone(target)
	if !ok {
		return nil, fmt.Errorf("%s: No such file or directory", target)
	}
	return f.collectZoneFiles(z, target)
}

func (f *aiFilesystem) collectZoneFiles(z zone, target string) ([]string, error) {
	target = normalizeVirtualPath(target)
	hostTarget, err := f.toHostPath(z, target)
	if err != nil {
		return nil, err
	}
	info, err := os.Stat(hostTarget)
	if err != nil {
		return nil, err
	}
	if !info.IsDir() {
		return []string{target}, nil
	}
	files := make([]string, 0)
	err = filepath.WalkDir(hostTarget, func(p string, d os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() {
			return nil
		}
		vpath, convErr := f.toVirtualPath(z, p)
		if convErr != nil {
			return convErr
		}
		files = append(files, vpath)
		return nil
	})
	if err != nil {
		return nil, err
	}
	sort.Strings(files)
	return files, nil
}

func (f *aiFilesystem) WriteFile(ctx context.Context, filePath string, contentValue string) error {
	_ = ctx
	z, hostPath, ok := f.resolveZone(filePath)
	if !ok {
		return fmt.Errorf("%s: path is outside allowed roots", filePath)
	}
	if !z.writable {
		return contract.ErrUnsupported
	}
	if f.writeLimitedBytes > 0 && len(contentValue) > f.writeLimitedBytes {
		return fmt.Errorf("write exceeds limit (%d bytes)", f.writeLimitedBytes)
	}
	if err := os.MkdirAll(filepath.Dir(hostPath), f.dirPerm); err != nil {
		return err
	}
	return os.WriteFile(hostPath, []byte(contentValue), f.filePerm)
}

func (f *aiFilesystem) AppendFile(ctx context.Context, filePath string, contentValue string) error {
	_ = ctx
	z, hostPath, ok := f.resolveZone(filePath)
	if !ok {
		return fmt.Errorf("%s: path is outside allowed roots", filePath)
	}
	if !z.writable {
		return contract.ErrUnsupported
	}
	if f.writeLimitedBytes > 0 && len(contentValue) > f.writeLimitedBytes {
		return fmt.Errorf("append exceeds limit (%d bytes)", f.writeLimitedBytes)
	}
	if err := os.MkdirAll(filepath.Dir(hostPath), f.dirPerm); err != nil {
		return err
	}
	fd, err := os.OpenFile(hostPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, f.filePerm)
	if err != nil {
		return err
	}
	defer fd.Close()
	_, err = fd.WriteString(contentValue)
	return err
}

func (f *aiFilesystem) EditFile(ctx context.Context, filePath string, oldString string, newString string, replaceAll bool) error {
	_ = ctx
	z, hostPath, ok := f.resolveZone(filePath)
	if !ok {
		return fmt.Errorf("%s: path is outside allowed roots", filePath)
	}
	if !z.writable {
		return contract.ErrUnsupported
	}
	raw, err := os.ReadFile(hostPath)
	if err != nil {
		return err
	}
	content := string(raw)
	count := strings.Count(content, oldString)
	if count <= 0 {
		return fmt.Errorf("old string not found")
	}
	if !replaceAll && count > 1 {
		return fmt.Errorf("old string appears %d times", count)
	}
	if replaceAll {
		content = strings.ReplaceAll(content, oldString, newString)
	} else {
		content = strings.Replace(content, oldString, newString, 1)
	}
	if f.writeLimitedBytes > 0 && len(content) > f.writeLimitedBytes {
		return fmt.Errorf("edit result exceeds limit (%d bytes)", f.writeLimitedBytes)
	}
	return os.WriteFile(hostPath, []byte(content), f.filePerm)
}

func (f *aiFilesystem) MakeDir(ctx context.Context, dirPath string) error {
	z, hostPath, ok := f.resolveZone(dirPath)
	if !ok {
		return fmt.Errorf("%s: path is outside allowed roots", dirPath)
	}
	if !z.writable {
		return contract.ErrUnsupported
	}
	return os.MkdirAll(hostPath, f.dirPerm)
}

func (f *aiFilesystem) RemoveFile(ctx context.Context, filePath string) error {
	z, hostPath, ok := f.resolveZone(filePath)
	if !ok {
		return fmt.Errorf("%s: path is outside allowed roots", filePath)
	}
	if !z.writable {
		return contract.ErrUnsupported
	}
	info, err := os.Stat(hostPath)
	if err != nil {
		return err
	}
	if info.IsDir() {
		return fmt.Errorf("cannot remove directory: %s", filePath)
	}
	return os.Remove(hostPath)
}

func (f *aiFilesystem) DescribePath(ctx context.Context, pathValue string) (contract.PathMeta, error) {
	isDir, err := f.IsDirPath(ctx, pathValue)
	if err != nil {
		return contract.PathMeta{}, err
	}
	pathValue = normalizeVirtualPath(pathValue)
	access := contract.PathAccessReadOnly
	if z, _, ok := f.resolveZone(pathValue); ok && z.writable {
		access = contract.PathAccessReadWrite
	}
	if isDir {
		meta := contract.PathMeta{
			Exists:           true,
			IsDir:            true,
			Kind:             "dir",
			Access:           access,
			Capabilities:     []string{contract.PathCapabilityDescribe, contract.PathCapabilityList, contract.PathCapabilitySearch},
			LineCount:        -1,
			FrontMatterLines: -1,
			SpeakerRows:      -1,
			UserRelevance:    "unknown",
		}
		if z, _, ok := f.resolveZone(pathValue); ok && strings.TrimSpace(z.kind) != "" {
			meta.Kind = z.kind
			if z.writable {
				meta.Capabilities = append(meta.Capabilities, contract.PathCapabilityMkdir, contract.PathCapabilityWrite)
			}
		}
		if f.relevanceEval != nil {
			meta.UserRelevance = nonEmpty(f.relevanceEval(pathValue, meta), meta.UserRelevance)
		}
		return meta, nil
	}
	raw, err := f.ReadRawContent(ctx, pathValue)
	if err != nil {
		return contract.PathMeta{}, err
	}
	kind := detectFileKind(pathValue)
	lines := splitLines(raw)
	meta := contract.PathMeta{
		Exists:           true,
		IsDir:            false,
		Kind:             kind,
		Access:           access,
		Capabilities:     []string{contract.PathCapabilityDescribe, contract.PathCapabilityRead},
		LineCount:        len(lines),
		FrontMatterLines: countFrontMatterLines(lines),
		SpeakerRows:      countSpeakerRows(lines),
		UserRelevance:    "unknown",
	}
	if access == contract.PathAccessReadWrite {
		meta.Capabilities = append(meta.Capabilities, contract.PathCapabilityWrite, contract.PathCapabilityAppend, contract.PathCapabilityEdit, contract.PathCapabilityRemove)
	}
	if f.relevanceEval != nil {
		meta.UserRelevance = nonEmpty(f.relevanceEval(pathValue, meta), meta.UserRelevance)
	}
	return meta, nil
}

func (f *aiFilesystem) PathEnv() []string {
	return dedupeNormalizedPaths(f.pathEnv)
}

func (f *aiFilesystem) resolveZone(pathValue string) (zone, string, bool) {
	pathValue = normalizeVirtualPath(pathValue)
	for _, z := range f.zones {
		if pathValue == z.virtualRoot || strings.HasPrefix(pathValue, z.virtualRoot+"/") {
			hostPath, err := f.toHostPath(z, pathValue)
			if err != nil {
				return zone{}, "", false
			}
			return z, hostPath, true
		}
	}
	return zone{}, "", false
}

func (f *aiFilesystem) toHostPath(z zone, virtualPath string) (string, error) {
	virtualPath = normalizeVirtualPath(virtualPath)
	rel := strings.TrimPrefix(virtualPath, z.virtualRoot)
	rel = strings.TrimPrefix(rel, "/")
	joined := filepath.Join(z.hostRoot, filepath.FromSlash(rel))
	cleaned := filepath.Clean(joined)
	relCheck, err := filepath.Rel(z.hostRoot, cleaned)
	if err != nil {
		return "", err
	}
	if relCheck == ".." || strings.HasPrefix(relCheck, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("path escape is not allowed")
	}
	return cleaned, nil
}

func (f *aiFilesystem) toVirtualPath(z zone, hostPath string) (string, error) {
	hostPath = filepath.Clean(hostPath)
	rel, err := filepath.Rel(z.hostRoot, hostPath)
	if err != nil {
		return "", err
	}
	if rel == "." {
		return z.virtualRoot, nil
	}
	rel = filepath.ToSlash(rel)
	if strings.HasPrefix(rel, "../") || rel == ".." {
		return "", fmt.Errorf("host path is outside zone root")
	}
	return normalizeVirtualPath(path.Join(z.virtualRoot, rel)), nil
}

func normalizeVirtualPath(raw string) string {
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

func detectFileKind(pathValue string) string {
	lower := strings.ToLower(pathValue)
	switch {
	case strings.HasSuffix(lower, ".md"):
		return "markdown"
	case strings.HasSuffix(lower, ".txt"):
		return "text"
	case strings.HasSuffix(lower, ".json"):
		return "json"
	case strings.HasSuffix(lower, ".yaml"), strings.HasSuffix(lower, ".yml"):
		return "yaml"
	case strings.HasSuffix(lower, ".go"):
		return "go_source"
	case strings.HasSuffix(lower, ".sh"):
		return "shell_script"
	default:
		return "file"
	}
}

func splitLines(raw string) []string {
	if raw == "" {
		return []string{}
	}
	lines := strings.Split(raw, "\n")
	if len(lines) > 0 && lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}
	return lines
}

func countFrontMatterLines(lines []string) int {
	if len(lines) < 3 {
		return -1
	}
	if strings.TrimSpace(lines[0]) != "---" {
		return -1
	}
	for i := 1; i < len(lines); i++ {
		if strings.TrimSpace(lines[i]) == "---" {
			return i + 1
		}
	}
	return -1
}

func countSpeakerRows(lines []string) int {
	count := 0
	for _, line := range lines {
		if speakerLineRe.MatchString(strings.TrimSpace(line)) {
			count++
		}
	}
	if count == 0 {
		return -1
	}
	return count
}

func dedupeNormalizedPaths(paths []string) []string {
	out := make([]string, 0, len(paths))
	for _, raw := range paths {
		p := normalizeVirtualPath(raw)
		if p == "/" {
			continue
		}
		out = append(out, p)
	}
	return dedupe(out)
}

func dedupe(paths []string) []string {
	seen := map[string]struct{}{}
	out := make([]string, 0, len(paths))
	for _, p := range paths {
		if _, ok := seen[p]; ok {
			continue
		}
		seen[p] = struct{}{}
		out = append(out, p)
	}
	sort.Strings(out)
	return out
}

func nonEmpty(v string, fallback string) string {
	if strings.TrimSpace(v) == "" {
		return fallback
	}
	return v
}
