package engine

import (
	"context"
	"fmt"
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
			mountFiles, collectErr := mounted.mount.CollectFilesUnder(ctx, mounted.point)
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

func (r mountRouter) wrapOps(ops contract.Ops) contract.Ops {
	origRequireAbsolutePath := ops.RequireAbsolutePath
	origListChildren := ops.ListChildren
	origIsDirPath := ops.IsDirPath
	origCollectFilesUnder := ops.CollectFilesUnder
	origResolveSearchPaths := ops.ResolveSearchPaths
	origDescribePath := ops.DescribePath
	origReadRawContent := ops.ReadRawContent
	origWriteFile := ops.WriteFile
	origAppendFile := ops.AppendFile
	origEditFile := ops.EditFile

	ops.RequireAbsolutePath = func(raw string) (string, error) {
		trimmed := strings.TrimSpace(raw)
		if strings.HasPrefix(trimmed, "/") {
			pathValue := normalizeAbsolutePath(trimmed)
			if _, ok := r.match(pathValue); ok {
				return pathValue, nil
			}
		}
		return origRequireAbsolutePath(raw)
	}

	ops.ListChildren = func(ctx context.Context, dir string) ([]string, error) {
		dir = normalizeAbsolutePath(dir)
		if mounted, ok, err := r.activeForPath(ctx, dir); err != nil {
			return nil, err
		} else if ok {
			children, err := mounted.mount.ListChildren(ctx, dir)
			if err != nil {
				return nil, err
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
		if mounted, ok, err := r.activeForPath(ctx, pathValue); err != nil {
			return false, err
		} else if ok {
			return mounted.mount.IsDirPath(ctx, pathValue)
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
		if mounted, ok, err := r.activeForPath(ctx, target); err != nil {
			return nil, err
		} else if ok {
			files, err := mounted.mount.CollectFilesUnder(ctx, target)
			if err != nil {
				return nil, err
			}
			return normalizePathList(files), nil
		}
		if isSynthetic, err := r.isSyntheticDir(ctx, target); err != nil {
			return nil, err
		} else if isSynthetic {
			return r.collectFilesForSyntheticDir(ctx, target)
		}
		return origCollectFilesUnder(ctx, target)
	}

	ops.ResolveSearchPaths = func(ctx context.Context, target string, recursive bool) ([]string, error) {
		target = normalizeAbsolutePath(target)
		if mounted, ok, err := r.activeForPath(ctx, target); err != nil {
			return nil, err
		} else if ok {
			paths, err := mounted.mount.ResolveSearchPaths(ctx, target, recursive)
			if err != nil {
				return nil, err
			}
			return normalizePathList(paths), nil
		}
		if isSynthetic, err := r.isSyntheticDir(ctx, target); err != nil {
			return nil, err
		} else if isSynthetic {
			if !recursive {
				return nil, fmt.Errorf("%s: Is a directory (use -r to search recursively)", target)
			}
			return r.collectFilesForSyntheticDir(ctx, target)
		}
		return origResolveSearchPaths(ctx, target, recursive)
	}

	ops.DescribePath = func(ctx context.Context, pathValue string) (contract.PathMeta, error) {
		pathValue = normalizeAbsolutePath(pathValue)
		if mounted, ok, err := r.activeForPath(ctx, pathValue); err != nil {
			return contract.PathMeta{}, err
		} else if ok {
			return mounted.mount.DescribePath(ctx, pathValue)
		}
		if isSynthetic, err := r.isSyntheticDir(ctx, pathValue); err != nil {
			return contract.PathMeta{}, err
		} else if isSynthetic {
			return contract.PathMeta{Exists: true, IsDir: true, Kind: "virtual_dir", LineCount: -1, FrontMatterLines: -1, SpeakerRows: -1, UserRelevance: "n/a"}, nil
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
			return mounted.mount.ReadRawContent(ctx, pathValue)
		}
		return origReadRawContent(ctx, pathValue)
	}

	ops.WriteFile = func(ctx context.Context, filePath string, content string) error {
		pathValue := normalizeAbsolutePath(filePath)
		if _, ok := r.match(pathValue); ok {
			return contract.ErrUnsupported
		}
		return origWriteFile(ctx, filePath, content)
	}
	ops.AppendFile = func(ctx context.Context, filePath string, content string) error {
		pathValue := normalizeAbsolutePath(filePath)
		if _, ok := r.match(pathValue); ok {
			return contract.ErrUnsupported
		}
		return origAppendFile(ctx, filePath, content)
	}
	ops.EditFile = func(ctx context.Context, filePath string, oldString string, newString string, replaceAll bool) error {
		pathValue := normalizeAbsolutePath(filePath)
		if _, ok := r.match(pathValue); ok {
			return contract.ErrUnsupported
		}
		return origEditFile(ctx, filePath, oldString, newString, replaceAll)
	}

	return ops
}
