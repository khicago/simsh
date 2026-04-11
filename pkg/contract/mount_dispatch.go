package contract

import (
	"context"
	"fmt"
	"path"
	"regexp"
	"sort"
	"strings"
	"unicode"
)

func NormalizeMountProfile(profile MountProfile) MountProfile {
	if profile.TruthModel == "" {
		profile.TruthModel = MountTruthProjection
	}
	if profile.MaterializationMode == "" {
		profile.MaterializationMode = MountMaterializationSnapshot
	}
	if profile.WriteSemantics == "" {
		profile.WriteSemantics = MountWriteReadOnly
	}
	if profile.LatencyClass == "" {
		profile.LatencyClass = MountLatencyLocalFast
	}
	if len(profile.SupportedCLIClasses) > 0 {
		seen := map[MountCLIClass]struct{}{}
		out := make([]MountCLIClass, 0, len(profile.SupportedCLIClasses))
		for _, class := range profile.SupportedCLIClasses {
			if class == "" {
				continue
			}
			if _, exists := seen[class]; exists {
				continue
			}
			seen[class] = struct{}{}
			out = append(out, class)
		}
		sort.Slice(out, func(i, j int) bool { return string(out[i]) < string(out[j]) })
		profile.SupportedCLIClasses = out
	}
	return profile
}

func MountSupportsCLIClass(profile MountProfile, class MountCLIClass) bool {
	profile = NormalizeMountProfile(profile)
	for _, supported := range profile.SupportedCLIClasses {
		if supported == class {
			return true
		}
	}
	return false
}

func MountAllowsFallback(profile MountProfile) bool {
	profile = NormalizeMountProfile(profile)
	switch profile.LatencyClass {
	case MountLatencyLocalFast, MountLatencyLocalHeavy:
		return true
	default:
		return false
	}
}

func StatMountPath(ctx context.Context, mount VirtualMount, pathValue string) (MountEntry, error) {
	if mount == nil {
		return MountEntry{}, ErrUnsupported
	}
	return mount.StatPath(ctx, pathValue)
}

func DescribeMountPath(ctx context.Context, mount VirtualMount, pathValue string) (PathMeta, error) {
	entry, err := StatMountPath(ctx, mount, pathValue)
	if err != nil {
		return PathMeta{}, err
	}
	return entry.Meta, nil
}

func IsMountDir(ctx context.Context, mount VirtualMount, pathValue string) (bool, error) {
	meta, err := DescribeMountPath(ctx, mount, pathValue)
	if err != nil {
		return false, err
	}
	return meta.Exists && meta.IsDir, nil
}

func ListMountChildren(ctx context.Context, mount VirtualMount, dir string) ([]string, error) {
	if err := requireMountCLIClass(mount, MountCLIList, "entry listing"); err != nil {
		return nil, err
	}
	lister, ok := mount.(EntryLister)
	if !ok {
		return nil, unsupportedMountCapability(mount, "entry listing")
	}
	result, err := lister.ListEntries(ctx, ListEntriesRequest{Dir: dir})
	if err != nil {
		return nil, err
	}
	children := make([]string, 0, len(result.Entries))
	seen := map[string]struct{}{}
	for _, entry := range result.Entries {
		if entry.Path == "" {
			continue
		}
		if _, exists := seen[entry.Path]; exists {
			continue
		}
		seen[entry.Path] = struct{}{}
		children = append(children, entry.Path)
	}
	sort.Strings(children)
	return children, nil
}

func CheckMountListScope(mount VirtualMount, req ListEntriesRequest) error {
	profile := NormalizeMountProfile(mount.Profile())
	if req.MaxDepth > 0 && profile.SLO.MaxSearchPaths > 0 && req.MaxDepth > profile.SLO.MaxSearchPaths {
		return overMountBudget(mount, "entry listing depth", fmt.Sprintf("max_depth=%d exceeds declared mount budget=%d", req.MaxDepth, profile.SLO.MaxSearchPaths))
	}
	return nil
}

func EnumerateMountFiles(ctx context.Context, mount VirtualMount, target string, recursive bool) ([]string, error) {
	return EnumerateMountFilesWithRequest(ctx, mount, EnumeratePathsRequest{Target: target, Recursive: recursive})
}

func EnumerateMountFilesWithRequest(ctx context.Context, mount VirtualMount, req EnumeratePathsRequest) ([]string, error) {
	target := req.Target
	entry, err := StatMountPath(ctx, mount, target)
	if err != nil {
		return nil, err
	}
	requiredClass := MountCLIRead
	if req.Recursive || entry.Meta.IsDir {
		requiredClass = MountCLIFind
	}
	if err := requireMountCLIClass(mount, requiredClass, "path enumeration"); err != nil {
		return nil, err
	}
	if !entry.Meta.IsDir {
		return []string{entry.Path}, nil
	}
	if !req.Recursive {
		return nil, fmt.Errorf("%s: Is a directory (use -r to search recursively)", target)
	}
	if err := checkMountEnumerateScope(mount, req); err != nil {
		return nil, err
	}
	if enumerator, ok := mount.(PathEnumerator); ok {
		result, err := enumerator.EnumeratePaths(ctx, req)
		if err != nil {
			return nil, err
		}
		return mountPathsFromEntries(result.Entries), nil
	}
	if !MountAllowsFallback(mount.Profile()) {
		return nil, unsupportedMountCapability(mount, "path enumeration")
	}
	lister, ok := mount.(EntryLister)
	if !ok {
		return nil, unsupportedMountCapability(mount, "path enumeration")
	}
	result, err := lister.ListEntries(ctx, ListEntriesRequest{Dir: target, Recursive: true, MaxDepth: req.MaxDepth})
	if err != nil {
		return nil, err
	}
	files := make([]string, 0, len(result.Entries))
	for _, child := range result.Entries {
		if child.Meta.IsDir {
			continue
		}
		files = append(files, child.Path)
	}
	return normalizeMountPaths(files), nil
}

func ReadMountContent(ctx context.Context, mount VirtualMount, pathValue string) (string, error) {
	return mount.ReadContent(ctx, pathValue)
}

func ReadManyFromMount(ctx context.Context, mount VirtualMount, paths []string) ([]MountContentEntry, error) {
	return ReadManyFromMountRequest(ctx, mount, ReadManyRequest{Paths: paths})
}

func ReadManyFromMountRequest(ctx context.Context, mount VirtualMount, req ReadManyRequest) ([]MountContentEntry, error) {
	paths := append([]string(nil), req.Paths...)
	requiredClass := MountCLIRead
	if len(paths) > 1 {
		requiredClass = MountCLIBulkRead
	}
	if err := checkMountReadManyBudget(mount, req); err != nil {
		return nil, err
	}
	if err := requireMountCLIClass(mount, requiredClass, "bulk read"); err != nil {
		return nil, err
	}
	if bulk, ok := mount.(BulkReader); ok {
		req.Paths = paths
		result, err := bulk.ReadMany(ctx, req)
		if err != nil {
			return nil, err
		}
		return result.Entries, nil
	}
	if !MountAllowsFallback(mount.Profile()) {
		return nil, unsupportedMountCapability(mount, "bulk read")
	}
	files := make([]MountContentEntry, 0, len(paths))
	for _, pathValue := range paths {
		content, err := ReadMountContent(ctx, mount, pathValue)
		if err != nil {
			return nil, err
		}
		files = append(files, MountContentEntry{Path: pathValue, Content: content})
	}
	return files, nil
}

func SearchMountContent(ctx context.Context, mount VirtualMount, req SearchRequest) (SearchResult, error) {
	if err := checkMountSearchBudget(mount, req); err != nil {
		return SearchResult{}, err
	}
	if err := requireMountCLIClass(mount, MountCLIContentSearch, "content search"); err != nil {
		return SearchResult{}, err
	}
	if searcher, ok := mount.(ContentSearcher); ok {
		return searcher.SearchContent(ctx, req)
	}
	if !MountAllowsFallback(mount.Profile()) {
		return SearchResult{}, unsupportedMountCapability(mount, "content search")
	}
	targets := req.Targets
	if len(targets) == 0 {
		targets = []string{mount.MountPoint()}
	}
	paths := make([]string, 0)
	for _, target := range targets {
		targetPaths, err := EnumerateMountFilesWithRequest(ctx, mount, EnumeratePathsRequest{Target: target, Recursive: true})
		if err != nil {
			return SearchResult{}, err
		}
		paths = append(paths, targetPaths...)
	}
	paths = normalizeMountPaths(paths)
	filtered, err := filterMountPathsByGlob(paths, req.Globs)
	if err != nil {
		return SearchResult{}, err
	}
	files, err := ReadManyFromMountRequest(ctx, mount, ReadManyRequest{Paths: filtered})
	if err != nil {
		return SearchResult{}, err
	}
	match, err := buildMountSearchMatcher(req.Pattern, req.Regex, req.CaseMode)
	if err != nil {
		return SearchResult{}, err
	}
	records := make([]SearchRecord, 0)
	for _, file := range files {
		found := mountSearchRecords(file.Content, match, req.Before, req.After, file.Path)
		if req.ListFiles {
			if len(found) == 0 {
				continue
			}
			records = append(records, SearchRecord{Path: file.Path, Kind: "file"})
			continue
		}
		records = append(records, found...)
	}
	if req.MaxResults > 0 && len(records) > req.MaxResults {
		records = records[:req.MaxResults]
	}
	return SearchResult{Records: records}, nil
}

func ApplyMountMutations(ctx context.Context, mount VirtualMount, batch MutationBatch) (MutationResult, error) {
	profile := NormalizeMountProfile(mount.Profile())
	if err := checkMountMutationBudget(mount, batch); err != nil {
		return MutationResult{}, err
	}
	if err := requireMountCLIClass(mount, MountCLIMutate, "mutation batch"); err != nil {
		return MutationResult{}, err
	}
	if profile.WriteSemantics == MountWriteReadOnly {
		return MutationResult{}, ErrUnsupported
	}
	if !mountDeclaresConsistency(profile) {
		return MutationResult{}, fmt.Errorf("%w: %s: writable mounts must declare consistency or refresh semantics", ErrUnsupported, mount.MountPoint())
	}
	mutator, ok := mount.(Mutator)
	if !ok {
		return MutationResult{}, unsupportedMountCapability(mount, "mutation batch")
	}
	return mutator.ApplyMutations(ctx, batch)
}

func CheckMountPathOp(mount VirtualMount, op PathOp) error {
	profile := NormalizeMountProfile(mount.Profile())
	switch op {
	case PathOpRead:
		if err := requireMountCLIClass(mount, MountCLIRead, "read"); err != nil {
			return err
		}
		return nil
	case PathOpTransferSource:
		if err := requireMountCLIClass(mount, MountCLIRead, "transfer source read"); err != nil {
			return err
		}
		if profile.TruthModel == MountTruthProjection {
			return ErrUnsupported
		}
		return nil
	case PathOpWrite, PathOpMkdir, PathOpRemove:
		if err := requireMountCLIClass(mount, MountCLIMutate, "mutation"); err != nil {
			return err
		}
		if profile.WriteSemantics == MountWriteReadOnly {
			return ErrUnsupported
		}
		if !mountDeclaresConsistency(profile) {
			return fmt.Errorf("%w: %s: writable mounts must declare consistency or refresh semantics", ErrUnsupported, mount.MountPoint())
		}
		return nil
	default:
		return nil
	}
}

func unsupportedMountCapability(mount VirtualMount, capability string) error {
	if mount == nil {
		return ErrUnsupported
	}
	profile := NormalizeMountProfile(mount.Profile())
	if profile.LatencyClass == MountLatencyRemoteHigh {
		return &MountUnsupportedError{
			MountPoint:   mount.MountPoint(),
			Capability:   capability,
			LatencyClass: profile.LatencyClass,
			Detail:       fmt.Sprintf("%s: %s requires an explicit mount capability on remote_high_latency mounts", mount.MountPoint(), capability),
		}
	}
	return &MountUnsupportedError{
		MountPoint:   mount.MountPoint(),
		Capability:   capability,
		LatencyClass: profile.LatencyClass,
		Detail:       fmt.Sprintf("%s: mount does not implement %s", mount.MountPoint(), capability),
	}
}

func requireMountCLIClass(mount VirtualMount, class MountCLIClass, capability string) error {
	if mount == nil {
		return ErrUnsupported
	}
	profile := NormalizeMountProfile(mount.Profile())
	if MountSupportsCLIClass(profile, class) {
		return nil
	}
	return &MountUnsupportedError{
		MountPoint:   mount.MountPoint(),
		Capability:   capability,
		LatencyClass: profile.LatencyClass,
		Detail:       fmt.Sprintf("%s: mount profile does not declare %s support for %s", mount.MountPoint(), class, capability),
	}
}

func mountDeclaresConsistency(profile MountProfile) bool {
	profile = NormalizeMountProfile(profile)
	return profile.Consistency.PathReadAfterWrite ||
		profile.Consistency.ListAfterWrite ||
		profile.Consistency.SearchAfterWrite ||
		profile.Consistency.RefreshRequired
}

func checkMountEnumerateScope(mount VirtualMount, req EnumeratePathsRequest) error {
	profile := NormalizeMountProfile(mount.Profile())
	if req.MaxDepth > 0 && profile.SLO.MaxSearchPaths > 0 && req.MaxDepth > profile.SLO.MaxSearchPaths {
		return overMountBudget(mount, "path enumeration depth", fmt.Sprintf("max_depth=%d exceeds declared mount budget=%d", req.MaxDepth, profile.SLO.MaxSearchPaths))
	}
	return nil
}

func checkMountReadManyBudget(mount VirtualMount, req ReadManyRequest) error {
	profile := NormalizeMountProfile(mount.Profile())
	if profile.SLO.MaxBatchCount > 0 && len(req.Paths) > profile.SLO.MaxBatchCount {
		return overMountBudget(mount, "bulk read batch", fmt.Sprintf("requested paths=%d exceeds declared mount batch count=%d", len(req.Paths), profile.SLO.MaxBatchCount))
	}
	if profile.SLO.MaxBatchBytes > 0 && req.MaxBytes > profile.SLO.MaxBatchBytes {
		return overMountBudget(mount, "bulk read batch", fmt.Sprintf("requested max_bytes=%d exceeds declared mount batch bytes=%d", req.MaxBytes, profile.SLO.MaxBatchBytes))
	}
	return nil
}

func checkMountSearchBudget(mount VirtualMount, req SearchRequest) error {
	profile := NormalizeMountProfile(mount.Profile())
	if profile.SLO.MaxSearchPaths > 0 && len(req.Targets) > profile.SLO.MaxSearchPaths {
		return overMountBudget(mount, "content search", fmt.Sprintf("requested targets=%d exceeds declared mount search path budget=%d", len(req.Targets), profile.SLO.MaxSearchPaths))
	}
	return nil
}

func checkMountMutationBudget(mount VirtualMount, batch MutationBatch) error {
	profile := NormalizeMountProfile(mount.Profile())
	if profile.SLO.MaxBatchCount > 0 && len(batch.Ops) > profile.SLO.MaxBatchCount {
		return overMountBudget(mount, "mutation batch", fmt.Sprintf("requested ops=%d exceeds declared mount batch count=%d", len(batch.Ops), profile.SLO.MaxBatchCount))
	}
	if profile.SLO.MaxBatchBytes > 0 {
		var total int64
		for _, op := range batch.Ops {
			total += int64(len(op.Content) + len(op.NewString) + len(op.OldString))
		}
		if total > profile.SLO.MaxBatchBytes {
			return overMountBudget(mount, "mutation batch", fmt.Sprintf("requested batch bytes=%d exceeds declared mount batch bytes=%d", total, profile.SLO.MaxBatchBytes))
		}
	}
	return nil
}

func overMountBudget(mount VirtualMount, capability string, detail string) error {
	profile := NormalizeMountProfile(mount.Profile())
	return &MountUnsupportedError{
		MountPoint:   mount.MountPoint(),
		Capability:   capability,
		LatencyClass: profile.LatencyClass,
		Detail:       fmt.Sprintf("%s: %s", mount.MountPoint(), detail),
	}
}

func mountPathsFromEntries(entries []MountEntry) []string {
	paths := make([]string, 0, len(entries))
	for _, entry := range entries {
		if entry.Meta.IsDir {
			continue
		}
		paths = append(paths, entry.Path)
	}
	return normalizeMountPaths(paths)
}

func normalizeMountPaths(paths []string) []string {
	seen := map[string]struct{}{}
	out := make([]string, 0, len(paths))
	for _, pathValue := range paths {
		trimmed := strings.TrimSpace(pathValue)
		if trimmed == "" {
			continue
		}
		if _, exists := seen[trimmed]; exists {
			continue
		}
		seen[trimmed] = struct{}{}
		out = append(out, trimmed)
	}
	sort.Strings(out)
	return out
}

func filterMountPathsByGlob(paths []string, globs []string) ([]string, error) {
	if len(globs) == 0 {
		return append([]string(nil), paths...), nil
	}
	filtered := make([]string, 0, len(paths))
	for _, pathValue := range paths {
		ok, err := matchMountGlobs(pathValue, globs)
		if err != nil {
			return nil, err
		}
		if ok {
			filtered = append(filtered, pathValue)
		}
	}
	return filtered, nil
}

func matchMountGlobs(filePath string, globs []string) (bool, error) {
	fullPath := strings.TrimPrefix(filePath, "/")
	baseName := path.Base(filePath)
	for _, glob := range globs {
		pattern := strings.TrimSpace(glob)
		if pattern == "" {
			continue
		}
		pattern = strings.TrimPrefix(pattern, "/")
		ok, err := path.Match(pattern, baseName)
		if err != nil {
			return false, fmt.Errorf("invalid glob pattern: %v", err)
		}
		if ok {
			return true, nil
		}
		ok, err = path.Match(pattern, fullPath)
		if err != nil {
			return false, fmt.Errorf("invalid glob pattern: %v", err)
		}
		if ok {
			return true, nil
		}
	}
	return false, nil
}

func buildMountSearchMatcher(pattern string, regex bool, caseMode SearchCaseMode) (func(string) bool, error) {
	effective := caseMode
	if effective == "" {
		effective = SearchCaseSensitive
	}
	if effective == SearchCaseSmart {
		if mountPatternHasUpper(pattern) {
			effective = SearchCaseSensitive
		} else {
			effective = SearchCaseIgnore
		}
	}
	if regex {
		if effective == SearchCaseIgnore {
			pattern = "(?i)" + pattern
		}
		re, err := regexp.Compile(pattern)
		if err != nil {
			return nil, err
		}
		return re.MatchString, nil
	}
	if effective == SearchCaseIgnore {
		lowerPattern := strings.ToLower(pattern)
		return func(line string) bool {
			return strings.Contains(strings.ToLower(line), lowerPattern)
		}, nil
	}
	return func(line string) bool {
		return strings.Contains(line, pattern)
	}, nil
}

func mountPatternHasUpper(pattern string) bool {
	for _, r := range pattern {
		if unicode.IsUpper(r) {
			return true
		}
	}
	return false
}

func mountSearchRecords(raw string, match func(string) bool, before int, after int, filePath string) []SearchRecord {
	lines := splitContractRawLines(raw)
	if len(lines) == 0 {
		return nil
	}
	matched := make([]bool, len(lines))
	include := make([]bool, len(lines))
	for i, line := range lines {
		if !match(line) {
			continue
		}
		matched[i] = true
		start := i - before
		if start < 0 {
			start = 0
		}
		end := i + after
		if end >= len(lines) {
			end = len(lines) - 1
		}
		for j := start; j <= end; j++ {
			include[j] = true
		}
	}
	out := make([]SearchRecord, 0)
	for i, line := range lines {
		if !include[i] {
			continue
		}
		record := SearchRecord{
			Path: filePath,
			Line: i + 1,
			Kind: "context",
			Text: line,
		}
		if matched[i] {
			record.Kind = "match"
		}
		out = append(out, record)
	}
	return out
}

func splitContractRawLines(raw string) []string {
	raw = strings.TrimSuffix(raw, "\n")
	if raw == "" {
		return nil
	}
	return strings.Split(raw, "\n")
}
