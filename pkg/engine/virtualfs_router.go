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
// exists solely to parent one or more nested mount points.
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

func (r mountRouter) listEntries(ctx context.Context, req contract.ListEntriesRequest) (contract.ListEntriesResult, bool, error) {
	dir := normalizeAbsolutePath(req.Dir)
	dir = normalizeAbsolutePath(dir)
	if mounted, ok, err := r.activeForPath(ctx, dir); err != nil {
		return contract.ListEntriesResult{}, false, err
	} else if ok {
		if err := contract.CheckMountListScope(mounted.mount, req); err != nil {
			return contract.ListEntriesResult{}, true, err
		}
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
			Recursive: req.Recursive,
			MaxDepth:  req.MaxDepth,
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

func (r mountRouter) enumeratePaths(ctx context.Context, req contract.EnumeratePathsRequest) (contract.EnumeratePathsResult, bool, error) {
	target := normalizeAbsolutePath(req.Target)
	target = normalizeAbsolutePath(target)
	if mounted, ok, err := r.activeForPath(ctx, target); err != nil {
		return contract.EnumeratePathsResult{}, false, err
	} else if ok {
		if err := contract.CheckMountListScope(mounted.mount, contract.ListEntriesRequest{Dir: target, Recursive: req.Recursive, MaxDepth: req.MaxDepth}); err != nil {
			return contract.EnumeratePathsResult{}, true, err
		}
		if !contract.MountSupportsCLIClass(mounted.mount.Profile(), contract.MountCLIFind) {
			return contract.EnumeratePathsResult{}, true, fmt.Errorf("%s: mount profile does not declare find support", target)
		}
		if enumerator, ok := mounted.mount.(contract.PathEnumerator); ok {
			req.Target = target
			result, err := enumerator.EnumeratePaths(ctx, req)
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
			result, err := enumerateEntriesViaListing(ctx, lister, target, req.Recursive, req.MaxDepth)
			return result, true, err
		}
		return contract.EnumeratePathsResult{}, true, fmt.Errorf("%s: mount does not support path enumeration", target)
	}
	if isSynthetic, err := r.isSyntheticDir(ctx, target); err != nil {
		return contract.EnumeratePathsResult{}, false, err
	} else if isSynthetic {
		if !req.Recursive {
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

func enumerateEntriesViaListing(ctx context.Context, lister contract.EntryLister, target string, recursive bool, maxDepth int) (contract.EnumeratePathsResult, error) {
	result, err := lister.ListEntries(ctx, contract.ListEntriesRequest{
		Dir:       target,
		Recursive: recursive,
		MaxDepth:  maxDepth,
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
