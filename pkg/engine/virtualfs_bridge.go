package engine

import (
	"context"
	"fmt"
	"path"
	"sort"
	"strings"

	"github.com/khicago/simsh/pkg/contract"
)

type mountedMount struct {
	point string
	mount contract.VirtualMount
}

type mountRouter struct {
	mounts []mountedMount
}

// isSyntheticPrefix reports whether pathValue is a synthetic directory that
// exists solely to parent one or more mount points (e.g. "/sys" for "/sys/bin").
//
// Note: this check is purely structural (no ctx), so it may return true even
// when the descendant mount is inactive. The actual directory existence and
// children are still determined by the ctx-aware synthetic helpers.
func (r mountRouter) isSyntheticPrefix(pathValue string) bool {
	pathValue = normalizeAbsolutePath(pathValue)
	if pathValue == "/" {
		return false
	}
	prefix := pathValue + "/"
	for _, mounted := range r.mounts {
		if mounted.point == pathValue {
			continue
		}
		if strings.HasPrefix(mounted.point, prefix) {
			return true
		}
	}
	return false
}

func newMountRouter(mounts []contract.VirtualMount) (mountRouter, error) {
	normalized := make([]mountedMount, 0, len(mounts))
	seen := map[string]struct{}{}
	for _, m := range mounts {
		if m == nil {
			continue
		}
		point := normalizeAbsolutePath(m.MountPoint())
		if point == "/" {
			return mountRouter{}, fmt.Errorf("virtual mount point must not be root")
		}
		if _, exists := seen[point]; exists {
			return mountRouter{}, fmt.Errorf("duplicate virtual mount point: %s", point)
		}
		seen[point] = struct{}{}
		normalized = append(normalized, mountedMount{point: point, mount: m})
	}
	sort.SliceStable(normalized, func(i, j int) bool {
		return len(normalized[i].point) > len(normalized[j].point)
	})
	return mountRouter{mounts: normalized}, nil
}

func (r mountRouter) isEmpty() bool {
	return len(r.mounts) == 0
}

func (r mountRouter) match(pathValue string) (mountedMount, bool) {
	pathValue = normalizeAbsolutePath(pathValue)
	for _, mounted := range r.mounts {
		if pathValue == mounted.point || strings.HasPrefix(pathValue, mounted.point+"/") {
			return mounted, true
		}
	}
	return mountedMount{}, false
}

func (r mountRouter) activeForPath(ctx context.Context, pathValue string) (mountedMount, bool, error) {
	mounted, ok := r.match(pathValue)
	if !ok {
		return mountedMount{}, false, nil
	}
	exists, err := mounted.mount.Exists(ctx)
	if err != nil {
		return mountedMount{}, false, err
	}
	if !exists {
		return mountedMount{}, false, nil
	}
	return mounted, true, nil
}

func (r mountRouter) activeMounts(ctx context.Context) ([]mountedMount, error) {
	active := make([]mountedMount, 0, len(r.mounts))
	for _, mounted := range r.mounts {
		exists, err := mounted.mount.Exists(ctx)
		if err != nil {
			return nil, err
		}
		if exists {
			active = append(active, mounted)
		}
	}
	return active, nil
}

func (r mountRouter) isSyntheticDir(ctx context.Context, pathValue string) (bool, error) {
	pathValue = normalizeAbsolutePath(pathValue)
	active, err := r.activeMounts(ctx)
	if err != nil {
		return false, err
	}
	for _, mounted := range active {
		if mounted.point == pathValue {
			return false, nil
		}
		if strings.HasPrefix(mounted.point, pathValue+"/") {
			return true, nil
		}
	}
	return false, nil
}

func (r mountRouter) syntheticChildren(ctx context.Context, dir string) ([]string, error) {
	dir = normalizeAbsolutePath(dir)
	active, err := r.activeMounts(ctx)
	if err != nil {
		return nil, err
	}
	prefix := dir + "/"
	if dir == "/" {
		prefix = "/"
	}
	children := make([]string, 0)
	for _, mounted := range active {
		if mounted.point == dir || !strings.HasPrefix(mounted.point, prefix) {
			continue
		}
		remainder := strings.TrimPrefix(mounted.point, prefix)
		if strings.TrimSpace(remainder) == "" {
			continue
		}
		next := strings.SplitN(remainder, "/", 2)[0]
		if strings.TrimSpace(next) == "" {
			continue
		}
		child := "/" + next
		if dir != "/" {
			child = dir + "/" + next
		}
		children = appendUniquePath(children, child)
	}
	return children, nil
}

func (r mountRouter) collectFilesForSyntheticDir(ctx context.Context, dir string) ([]string, error) {
	dir = normalizeAbsolutePath(dir)
	active, err := r.activeMounts(ctx)
	if err != nil {
		return nil, err
	}
	files := make([]string, 0)
	for _, mounted := range active {
		if mounted.point == dir || strings.HasPrefix(mounted.point, dir+"/") {
			mountFiles, collectErr := contract.EnumerateMountFiles(ctx, mounted.mount, mounted.point, true)
			if collectErr != nil {
				return nil, collectErr
			}
			for _, pathValue := range mountFiles {
				files = appendUniquePath(files, pathValue)
			}
		}
	}
	return normalizePathList(files), nil
}

func syntheticDirEntry(pathValue string) contract.MountEntry {
	return contract.MountEntry{
		Path: pathValue,
		Name: path.Base(pathValue),
		Meta: contract.PathMeta{
			Exists:           true,
			IsDir:            true,
			Kind:             "virtual_dir",
			Access:           contract.PathAccessReadOnly,
			Capabilities:     []string{contract.PathCapabilityDescribe, contract.PathCapabilityList, contract.PathCapabilitySearch},
			LineCount:        -1,
			FrontMatterLines: -1,
			SpeakerRows:      -1,
			UserRelevance:    "n/a",
		},
	}
}

func (r mountRouter) statEntry(ctx context.Context, pathValue string) (contract.MountEntry, bool, error) {
	pathValue = normalizeAbsolutePath(pathValue)
	if mounted, ok, err := r.activeForPath(ctx, pathValue); err != nil {
		return contract.MountEntry{}, false, err
	} else if ok {
		entry, err := mounted.mount.StatPath(ctx, pathValue)
		return entry, true, err
	}
	if isSynthetic, err := r.isSyntheticDir(ctx, pathValue); err != nil {
		return contract.MountEntry{}, false, err
	} else if isSynthetic {
		return syntheticDirEntry(pathValue), true, nil
	}
	return contract.MountEntry{}, false, nil
}

func (r mountRouter) listEntries(ctx context.Context, dir string, recursive bool) (contract.ListEntriesResult, bool, error) {
	dir = normalizeAbsolutePath(dir)
	if mounted, ok, err := r.activeForPath(ctx, dir); err != nil {
		return contract.ListEntriesResult{}, false, err
	} else if ok {
		if !contract.MountSupportsCLIClass(mounted.mount.Profile(), contract.MountCLIList) {
			return contract.ListEntriesResult{}, true, fmt.Errorf("%s: mount profile does not declare list support", dir)
		}
		lister, ok := mounted.mount.(contract.EntryLister)
		if !ok {
			if mounted.mount.Profile().LatencyClass == contract.MountLatencyRemoteHigh {
				return contract.ListEntriesResult{}, true, &contract.MountUnsupportedError{
					MountPoint:   mounted.mount.MountPoint(),
					Capability:   "entry listing",
					LatencyClass: contract.MountLatencyRemoteHigh,
					Detail:       fmt.Sprintf("%s: entry listing requires EntryLister for remote_high_latency mount", dir),
				}
			}
			return contract.ListEntriesResult{}, true, fmt.Errorf("%s: mount does not support entry listing", dir)
		}
		result, err := lister.ListEntries(ctx, contract.ListEntriesRequest{
			Dir:       dir,
			Recursive: recursive,
		})
		return result, true, err
	}
	if isSynthetic, err := r.isSyntheticDir(ctx, dir); err != nil {
		return contract.ListEntriesResult{}, false, err
	} else if isSynthetic {
		children, err := r.syntheticChildren(ctx, dir)
		if err != nil {
			return contract.ListEntriesResult{}, false, err
		}
		entries := make([]contract.MountEntry, 0, len(children))
		for _, child := range children {
			entry, _, err := r.statEntry(ctx, child)
			if err != nil {
				return contract.ListEntriesResult{}, false, err
			}
			entries = append(entries, entry)
		}
		return contract.ListEntriesResult{Entries: entries}, true, nil
	}
	return contract.ListEntriesResult{}, false, nil
}

func (r mountRouter) enumeratePaths(ctx context.Context, target string, recursive bool) (contract.EnumeratePathsResult, bool, error) {
	target = normalizeAbsolutePath(target)
	if mounted, ok, err := r.activeForPath(ctx, target); err != nil {
		return contract.EnumeratePathsResult{}, false, err
	} else if ok {
		if !contract.MountSupportsCLIClass(mounted.mount.Profile(), contract.MountCLIFind) {
			return contract.EnumeratePathsResult{}, true, fmt.Errorf("%s: mount profile does not declare find support", target)
		}
		if enumerator, ok := mounted.mount.(contract.PathEnumerator); ok {
			result, err := enumerator.EnumeratePaths(ctx, contract.EnumeratePathsRequest{
				Target:    target,
				Recursive: recursive,
			})
			return result, true, err
		}
		if mounted.mount.Profile().LatencyClass == contract.MountLatencyRemoteHigh {
			return contract.EnumeratePathsResult{}, true, &contract.MountUnsupportedError{
				MountPoint:   mounted.mount.MountPoint(),
				Capability:   "path enumeration",
				LatencyClass: contract.MountLatencyRemoteHigh,
				Detail:       fmt.Sprintf("%s: path enumeration requires PathEnumerator for remote_high_latency mount", target),
			}
		}
		if lister, ok := mounted.mount.(contract.EntryLister); ok {
			result, err := enumerateEntriesViaListing(ctx, lister, target, recursive)
			return result, true, err
		}
		return contract.EnumeratePathsResult{}, true, fmt.Errorf("%s: mount does not support path enumeration", target)
	}
	if isSynthetic, err := r.isSyntheticDir(ctx, target); err != nil {
		return contract.EnumeratePathsResult{}, false, err
	} else if isSynthetic {
		if !recursive {
			return contract.EnumeratePathsResult{}, false, fmt.Errorf("%s: Is a directory (use -r to search recursively)", target)
		}
		files, err := r.collectFilesForSyntheticDir(ctx, target)
		if err != nil {
			return contract.EnumeratePathsResult{}, false, err
		}
		entries := make([]contract.MountEntry, 0, len(files))
		for _, filePath := range files {
			entry, entryOK, entryErr := r.statEntry(ctx, filePath)
			if entryErr != nil {
				return contract.EnumeratePathsResult{}, false, entryErr
			}
			if !entryOK {
				return contract.EnumeratePathsResult{}, false, fmt.Errorf("%s: No such file or directory", filePath)
			}
			entries = append(entries, entry)
		}
		return contract.EnumeratePathsResult{Entries: entries}, true, nil
	}
	return contract.EnumeratePathsResult{}, false, nil
}

func enumerateEntriesViaListing(ctx context.Context, lister contract.EntryLister, target string, recursive bool) (contract.EnumeratePathsResult, error) {
	result, err := lister.ListEntries(ctx, contract.ListEntriesRequest{
		Dir:       target,
		Recursive: recursive,
	})
	if err != nil {
		return contract.EnumeratePathsResult{}, err
	}
	files := make([]contract.MountEntry, 0, len(result.Entries))
	for _, entry := range result.Entries {
		if entry.Meta.IsDir {
			continue
		}
		files = append(files, entry)
	}
	return contract.EnumeratePathsResult{Entries: files}, nil
}

func normalizePathList(paths []string) []string {
	out := make([]string, 0, len(paths))
	for _, pathValue := range paths {
		trimmed := strings.TrimSpace(pathValue)
		if trimmed == "" {
			continue
		}
		out = append(out, normalizeAbsolutePath(trimmed))
	}
	sort.Strings(out)
	return out
}

func containsPathCapability(caps []string, target string) bool {
	for _, capability := range caps {
		if capability == target {
			return true
		}
	}
	return false
}

func mutationSpecPathOp(kind contract.MutationKind) (contract.PathOp, error) {
	switch kind {
	case contract.MutationWriteFile, contract.MutationAppend, contract.MutationEdit:
		return contract.PathOpWrite, nil
	case contract.MutationMakeDir:
		return contract.PathOpMkdir, nil
	case contract.MutationRemoveFile, contract.MutationRemoveDir:
		return contract.PathOpRemove, nil
	default:
		return "", fmt.Errorf("unsupported mutation kind %q", kind)
	}
}

func validateMountedMutationBatch(mount contract.VirtualMount, ops []contract.MutationSpec) error {
	for _, op := range ops {
		pathOp, err := mutationSpecPathOp(op.Kind)
		if err != nil {
			return err
		}
		if err := contract.CheckMountPathOp(mount, pathOp); err != nil {
			return err
		}
	}
	if _, ok := mount.(contract.Mutator); ok {
		return nil
	}
	profile := contract.NormalizeMountProfile(mount.Profile())
	if profile.LatencyClass == contract.MountLatencyRemoteHigh {
		return &contract.MountUnsupportedError{
			MountPoint:   mount.MountPoint(),
			Capability:   "mutation batch",
			LatencyClass: profile.LatencyClass,
			Detail:       fmt.Sprintf("%s: mutation batch requires an explicit mount capability on remote_high_latency mounts", mount.MountPoint()),
		}
	}
	return contract.ErrUnsupported
}

func validateFilesystemFallbackMutation(
	op contract.MutationSpec,
	writeFile func(context.Context, string, string) error,
	appendFile func(context.Context, string, string) error,
	editFile func(context.Context, string, string, string, bool) error,
	makeDir func(context.Context, string) error,
	removeFile func(context.Context, string) error,
	removeDir func(context.Context, string) error,
) error {
	switch op.Kind {
	case contract.MutationWriteFile:
		if writeFile == nil {
			return contract.ErrUnsupported
		}
	case contract.MutationAppend:
		if appendFile == nil {
			return contract.ErrUnsupported
		}
	case contract.MutationEdit:
		if editFile == nil {
			return contract.ErrUnsupported
		}
	case contract.MutationMakeDir:
		if makeDir == nil {
			return contract.ErrUnsupported
		}
	case contract.MutationRemoveFile:
		if removeFile == nil {
			return contract.ErrUnsupported
		}
	case contract.MutationRemoveDir:
		if removeDir == nil {
			return contract.ErrUnsupported
		}
	default:
		return fmt.Errorf("unsupported mutation kind %q", op.Kind)
	}
	return nil
}

func applyFilesystemFallbackMutation(
	ctx context.Context,
	op contract.MutationSpec,
	writeFile func(context.Context, string, string) error,
	appendFile func(context.Context, string, string) error,
	editFile func(context.Context, string, string, string, bool) error,
	makeDir func(context.Context, string) error,
	removeFile func(context.Context, string) error,
	removeDir func(context.Context, string) error,
) error {
	switch op.Kind {
	case contract.MutationWriteFile:
		return writeFile(ctx, op.Path, op.Content)
	case contract.MutationAppend:
		return appendFile(ctx, op.Path, op.Content)
	case contract.MutationEdit:
		return editFile(ctx, op.Path, op.OldString, op.NewString, op.ReplaceAll)
	case contract.MutationMakeDir:
		return makeDir(ctx, op.Path)
	case contract.MutationRemoveFile:
		return removeFile(ctx, op.Path)
	case contract.MutationRemoveDir:
		return removeDir(ctx, op.Path)
	default:
		return fmt.Errorf("unsupported mutation kind %q", op.Kind)
	}
}

func (r mountRouter) wrapOps(ops contract.Ops) contract.Ops {
	origRequireAbsolutePath := ops.RequireAbsolutePath
	origListChildren := ops.ListChildren
	origIsDirPath := ops.IsDirPath
	origListEntries := ops.ListEntries
	origEnumeratePaths := ops.EnumeratePaths
	origCollectFilesUnder := ops.CollectFilesUnder
	origResolveSearchPaths := ops.ResolveSearchPaths
	origDescribePath := ops.DescribePath
	origReadRawContent := ops.ReadRawContent
	origReadMany := ops.ReadMany
	origSearchContent := ops.SearchContent
	origWriteFile := ops.WriteFile
	origAppendFile := ops.AppendFile
	origEditFile := ops.EditFile
	origMakeDir := ops.MakeDir
	origRemoveFile := ops.RemoveFile
	origRemoveDir := ops.RemoveDir
	origApplyMutations := ops.ApplyMutations
	origCheckPathOp := ops.CheckPathOp

	ops.RequireAbsolutePath = func(raw string) (string, error) {
		trimmed := strings.TrimSpace(raw)
		if strings.HasPrefix(trimmed, "/") {
			pathValue := normalizeAbsolutePath(trimmed)
			if _, ok := r.match(pathValue); ok || r.isSyntheticPrefix(pathValue) {
				return pathValue, nil
			}
		}
		return origRequireAbsolutePath(raw)
	}

	ops.ListEntries = func(ctx context.Context, req contract.ListEntriesRequest) (contract.ListEntriesResult, error) {
		req.Dir = normalizeAbsolutePath(req.Dir)
		if result, ok, err := r.listEntries(ctx, req.Dir, req.Recursive); err != nil {
			return contract.ListEntriesResult{}, err
		} else if ok {
			return result, nil
		}
		if origListEntries != nil {
			return origListEntries(ctx, req)
		}
		return contract.ListEntriesResult{}, contract.ErrUnsupported
	}

	ops.EnumeratePaths = func(ctx context.Context, req contract.EnumeratePathsRequest) (contract.EnumeratePathsResult, error) {
		req.Target = normalizeAbsolutePath(req.Target)
		if result, ok, err := r.enumeratePaths(ctx, req.Target, req.Recursive); err != nil {
			return contract.EnumeratePathsResult{}, err
		} else if ok {
			return result, nil
		}
		if origEnumeratePaths != nil {
			return origEnumeratePaths(ctx, req)
		}
		return contract.EnumeratePathsResult{}, contract.ErrUnsupported
	}

	ops.ListChildren = func(ctx context.Context, dir string) ([]string, error) {
		dir = normalizeAbsolutePath(dir)
		if result, ok, err := r.listEntries(ctx, dir, false); err != nil {
			return nil, err
		} else if ok {
			children := make([]string, 0, len(result.Entries))
			for _, entry := range result.Entries {
				children = append(children, entry.Path)
			}
			return normalizePathList(children), nil
		}

		children, err := origListChildren(ctx, dir)
		synthetic, syntheticErr := r.syntheticChildren(ctx, dir)
		if syntheticErr != nil {
			return nil, syntheticErr
		}
		if err != nil {
			if len(synthetic) == 0 {
				return nil, err
			}
			return normalizePathList(synthetic), nil
		}
		for _, child := range synthetic {
			children = appendUniquePath(children, child)
		}
		return children, nil
	}

	ops.IsDirPath = func(ctx context.Context, pathValue string) (bool, error) {
		pathValue = normalizeAbsolutePath(pathValue)
		if entry, ok, err := r.statEntry(ctx, pathValue); err != nil {
			return false, err
		} else if ok {
			return entry.Meta.IsDir, nil
		}
		isDir, err := origIsDirPath(ctx, pathValue)
		if err != nil {
			return false, err
		}
		if isDir {
			return true, nil
		}
		return r.isSyntheticDir(ctx, pathValue)
	}

	ops.CollectFilesUnder = func(ctx context.Context, target string) ([]string, error) {
		target = normalizeAbsolutePath(target)
		if result, ok, err := r.enumeratePaths(ctx, target, true); err != nil {
			return nil, err
		} else if ok {
			paths := make([]string, 0, len(result.Entries))
			for _, entry := range result.Entries {
				paths = append(paths, entry.Path)
			}
			return normalizePathList(paths), nil
		}
		return origCollectFilesUnder(ctx, target)
	}

	ops.ResolveSearchPaths = func(ctx context.Context, target string, recursive bool) ([]string, error) {
		target = normalizeAbsolutePath(target)
		if result, ok, err := r.enumeratePaths(ctx, target, recursive); err != nil {
			return nil, err
		} else if ok {
			paths := make([]string, 0, len(result.Entries))
			for _, entry := range result.Entries {
				paths = append(paths, entry.Path)
			}
			return normalizePathList(paths), nil
		}
		return origResolveSearchPaths(ctx, target, recursive)
	}

	ops.DescribePath = func(ctx context.Context, pathValue string) (contract.PathMeta, error) {
		pathValue = normalizeAbsolutePath(pathValue)
		if entry, ok, err := r.statEntry(ctx, pathValue); err != nil {
			return contract.PathMeta{}, err
		} else if ok {
			return entry.Meta, nil
		}
		if origDescribePath != nil {
			return origDescribePath(ctx, pathValue)
		}
		return contract.PathMeta{}, contract.ErrUnsupported
	}

	ops.ReadRawContent = func(ctx context.Context, pathValue string) (string, error) {
		pathValue = normalizeAbsolutePath(pathValue)
		if mounted, ok, err := r.activeForPath(ctx, pathValue); err != nil {
			return "", err
		} else if ok {
			return contract.ReadMountContent(ctx, mounted.mount, pathValue)
		}
		return origReadRawContent(ctx, pathValue)
	}

	ops.ReadMany = func(ctx context.Context, req contract.ReadManyRequest) (contract.ReadManyResult, error) {
		if len(req.Paths) == 0 {
			return contract.ReadManyResult{}, nil
		}
		resultsByPath := map[string][]contract.MountContentEntry{}
		groups := map[string][]string{}
		groupMounts := map[string]contract.VirtualMount{}
		groupOrder := make([]string, 0)
		fsPaths := make([]string, 0, len(req.Paths))
		for _, rawPath := range req.Paths {
			pathValue := normalizeAbsolutePath(rawPath)
			if mounted, ok, err := r.activeForPath(ctx, pathValue); err != nil {
				return contract.ReadManyResult{}, err
			} else if ok {
				groupKey := mounted.point
				if _, exists := groups[groupKey]; !exists {
					groupOrder = append(groupOrder, groupKey)
				}
				groups[groupKey] = append(groups[groupKey], pathValue)
				groupMounts[groupKey] = mounted.mount
				continue
			}
			if origReadMany != nil {
				fsPaths = append(fsPaths, pathValue)
				continue
			}
			raw, err := origReadRawContent(ctx, pathValue)
			if err != nil {
				return contract.ReadManyResult{}, err
			}
			appendReadManyEntries(resultsByPath, []contract.MountContentEntry{{Path: pathValue, Content: raw}})
		}
		if len(fsPaths) > 0 {
			partial, err := origReadMany(ctx, contract.ReadManyRequest{Paths: fsPaths})
			if err != nil {
				return contract.ReadManyResult{}, err
			}
			appendReadManyEntries(resultsByPath, partial.Entries)
		}
		for _, groupKey := range groupOrder {
			groupedPaths := groups[groupKey]
			mount := groupMounts[groupKey]
			entries, err := contract.ReadManyFromMount(ctx, mount, groupedPaths)
			if err != nil {
				return contract.ReadManyResult{}, err
			}
			appendReadManyEntries(resultsByPath, entries)
		}
		results := make([]contract.MountContentEntry, 0, len(req.Paths))
		for _, rawPath := range req.Paths {
			pathValue := normalizeAbsolutePath(rawPath)
			queued := resultsByPath[pathValue]
			if len(queued) == 0 {
				return contract.ReadManyResult{}, fmt.Errorf("%s: missing read result", pathValue)
			}
			results = append(results, queued[0])
			resultsByPath[pathValue] = queued[1:]
		}
		return contract.ReadManyResult{Entries: results}, nil
	}

	ops.SearchContent = func(ctx context.Context, req contract.SearchRequest) (contract.SearchResult, error) {
		if len(req.Targets) == 0 {
			if origSearchContent != nil {
				return origSearchContent(ctx, req)
			}
			return contract.SearchResult{}, contract.ErrUnsupported
		}
		targets := make([]string, 0, len(req.Targets))
		for _, target := range req.Targets {
			targets = append(targets, normalizeAbsolutePath(target))
		}
		req.Targets = targets
		var (
			mountedPoint   string
			mountedMount   contract.VirtualMount
			mountedTargets []string
			fsTargets      []string
		)
		for _, target := range targets {
			mounted, ok, err := r.activeForPath(ctx, target)
			if err != nil {
				return contract.SearchResult{}, err
			}
			if ok {
				if mountedPoint == "" {
					mountedPoint = mounted.point
					mountedMount = mounted.mount
				} else if mounted.point != mountedPoint {
					return contract.SearchResult{}, fmt.Errorf("cross-mount search targets are not supported")
				}
				mountedTargets = append(mountedTargets, target)
				continue
			}
			if synthetic, err := r.isSyntheticDir(ctx, target); err != nil {
				return contract.SearchResult{}, err
			} else if synthetic {
				return contract.SearchResult{}, fmt.Errorf("cross-mount search targets are not supported")
			}
			fsTargets = append(fsTargets, target)
		}
		if len(mountedTargets) > 0 && len(fsTargets) > 0 {
			return contract.SearchResult{}, fmt.Errorf("mixed mounted and filesystem search targets are not supported")
		}
		if len(mountedTargets) == 0 {
			if origSearchContent != nil {
				return origSearchContent(ctx, req)
			}
			return contract.SearchResult{}, contract.ErrUnsupported
		}
		req.Targets = mountedTargets
		return contract.SearchMountContent(ctx, mountedMount, req)
	}

	ops.WriteFile = func(ctx context.Context, filePath string, content string) error {
		pathValue := normalizeAbsolutePath(filePath)
		if r.isSyntheticPrefix(pathValue) {
			return contract.ErrUnsupported
		}
		if _, ok := r.match(pathValue); ok {
			if ops.ApplyMutations != nil {
				_, err := ops.ApplyMutations(ctx, contract.MutationBatch{
					Ops: []contract.MutationSpec{{Kind: contract.MutationWriteFile, Path: pathValue, Content: content}},
				})
				return err
			}
			return contract.ErrUnsupported
		}
		return origWriteFile(ctx, filePath, content)
	}
	ops.AppendFile = func(ctx context.Context, filePath string, content string) error {
		pathValue := normalizeAbsolutePath(filePath)
		if r.isSyntheticPrefix(pathValue) {
			return contract.ErrUnsupported
		}
		if _, ok := r.match(pathValue); ok {
			if ops.ApplyMutations != nil {
				_, err := ops.ApplyMutations(ctx, contract.MutationBatch{
					Ops: []contract.MutationSpec{{Kind: contract.MutationAppend, Path: pathValue, Content: content}},
				})
				return err
			}
			return contract.ErrUnsupported
		}
		return origAppendFile(ctx, filePath, content)
	}
	ops.EditFile = func(ctx context.Context, filePath string, oldString string, newString string, replaceAll bool) error {
		pathValue := normalizeAbsolutePath(filePath)
		if r.isSyntheticPrefix(pathValue) {
			return contract.ErrUnsupported
		}
		if _, ok := r.match(pathValue); ok {
			if ops.ApplyMutations != nil {
				_, err := ops.ApplyMutations(ctx, contract.MutationBatch{
					Ops: []contract.MutationSpec{{Kind: contract.MutationEdit, Path: pathValue, OldString: oldString, NewString: newString, ReplaceAll: replaceAll}},
				})
				return err
			}
			return contract.ErrUnsupported
		}
		return origEditFile(ctx, filePath, oldString, newString, replaceAll)
	}

	ops.MakeDir = func(ctx context.Context, dirPath string) error {
		pathValue := normalizeAbsolutePath(dirPath)
		if r.isSyntheticPrefix(pathValue) {
			return contract.ErrUnsupported
		}
		if _, ok := r.match(pathValue); ok {
			if ops.ApplyMutations != nil {
				_, err := ops.ApplyMutations(ctx, contract.MutationBatch{
					Ops: []contract.MutationSpec{{Kind: contract.MutationMakeDir, Path: pathValue}},
				})
				return err
			}
			return contract.ErrUnsupported
		}
		return origMakeDir(ctx, dirPath)
	}

	ops.RemoveFile = func(ctx context.Context, filePath string) error {
		pathValue := normalizeAbsolutePath(filePath)
		if r.isSyntheticPrefix(pathValue) {
			return contract.ErrUnsupported
		}
		if _, ok := r.match(pathValue); ok {
			if ops.ApplyMutations != nil {
				_, err := ops.ApplyMutations(ctx, contract.MutationBatch{
					Ops: []contract.MutationSpec{{Kind: contract.MutationRemoveFile, Path: pathValue}},
				})
				return err
			}
			return contract.ErrUnsupported
		}
		return origRemoveFile(ctx, filePath)
	}
	ops.RemoveDir = func(ctx context.Context, dirPath string) error {
		pathValue := normalizeAbsolutePath(dirPath)
		if r.isSyntheticPrefix(pathValue) {
			return contract.ErrUnsupported
		}
		if _, ok := r.match(pathValue); ok {
			if ops.ApplyMutations != nil {
				_, err := ops.ApplyMutations(ctx, contract.MutationBatch{
					Ops: []contract.MutationSpec{{Kind: contract.MutationRemoveDir, Path: pathValue}},
				})
				return err
			}
			return contract.ErrUnsupported
		}
		return origRemoveDir(ctx, dirPath)
	}

	ops.ApplyMutations = func(ctx context.Context, req contract.MutationBatch) (contract.MutationResult, error) {
		mountedOps := make([]contract.MutationSpec, 0, len(req.Ops))
		fsOps := make([]contract.MutationSpec, 0, len(req.Ops))
		var mountedPoint string
		var mountedMount contract.VirtualMount
		for _, op := range req.Ops {
			pathValue := normalizeAbsolutePath(op.Path)
			if r.isSyntheticPrefix(pathValue) {
				return contract.MutationResult{}, contract.ErrUnsupported
			}
			op.Path = pathValue
			if mounted, ok := r.match(pathValue); ok {
				if mountedPoint == "" {
					mountedPoint = mounted.point
					mountedMount = mounted.mount
				} else if mounted.point != mountedPoint {
					return contract.MutationResult{}, fmt.Errorf("cross-mount mutation batches are not supported")
				}
				mountedOps = append(mountedOps, op)
				continue
			}
			fsOps = append(fsOps, op)
		}
		if len(mountedOps) > 0 {
			if err := validateMountedMutationBatch(mountedMount, mountedOps); err != nil {
				return contract.MutationResult{}, err
			}
		}
		if len(fsOps) > 0 && origApplyMutations == nil {
			for _, op := range fsOps {
				if err := validateFilesystemFallbackMutation(op, origWriteFile, origAppendFile, origEditFile, origMakeDir, origRemoveFile, origRemoveDir); err != nil {
					return contract.MutationResult{}, err
				}
			}
		}
		results := make([]contract.MutationRecord, 0, len(req.Ops))
		if len(fsOps) > 0 {
			if origApplyMutations != nil {
				result, err := origApplyMutations(ctx, contract.MutationBatch{Ops: fsOps})
				if err != nil {
					return contract.MutationResult{}, err
				}
				results = append(results, result.Records...)
			} else {
				for _, op := range fsOps {
					if err := applyFilesystemFallbackMutation(ctx, op, origWriteFile, origAppendFile, origEditFile, origMakeDir, origRemoveFile, origRemoveDir); err != nil {
						return contract.MutationResult{}, err
					}
					results = append(results, contract.MutationRecord{Kind: op.Kind, Path: op.Path, Status: "ok"})
				}
			}
		}
		if len(mountedOps) > 0 {
			result, err := contract.ApplyMountMutations(ctx, mountedMount, contract.MutationBatch{Ops: mountedOps})
			if err != nil {
				return contract.MutationResult{}, err
			}
			results = append(results, result.Records...)
		}
		return contract.MutationResult{Records: results}, nil
	}

	ops.CheckPathOp = func(ctx context.Context, op contract.PathOp, pathValue string) error {
		abs := normalizeAbsolutePath(pathValue)
		switch op {
		case contract.PathOpRead, contract.PathOpTransferSource:
			if mounted, ok, err := r.activeForPath(ctx, abs); err != nil {
				return err
			} else if ok {
				entry, err := mounted.mount.StatPath(ctx, abs)
				if err != nil {
					return err
				}
				if !entry.Meta.IsDir && containsPathCapability(entry.Meta.Capabilities, contract.PathCapabilityRead) {
					return contract.CheckMountPathOp(mounted.mount, op)
				}
				return contract.ErrUnsupported
			}
		case contract.PathOpWrite, contract.PathOpMkdir, contract.PathOpRemove:
			if r.isSyntheticPrefix(abs) {
				return contract.ErrUnsupported
			}
			if mounted, ok := r.match(abs); ok {
				return contract.CheckMountPathOp(mounted.mount, op)
			}
		}
		if origCheckPathOp != nil {
			return origCheckPathOp(ctx, op, abs)
		}
		return nil
	}

	return ops
}

func appendReadManyEntries(index map[string][]contract.MountContentEntry, entries []contract.MountContentEntry) {
	for _, entry := range entries {
		pathValue := normalizeAbsolutePath(entry.Path)
		entry.Path = pathValue
		index[pathValue] = append(index[pathValue], entry)
	}
}
