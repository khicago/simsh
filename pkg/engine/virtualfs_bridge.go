package engine

import (
	"context"
	"fmt"
	"strings"

	"github.com/khicago/simsh/pkg/contract"
)

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
		if result, ok, err := r.listEntries(ctx, req); err != nil {
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
		if result, ok, err := r.enumeratePaths(ctx, req); err != nil {
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
		if result, ok, err := r.listEntries(ctx, contract.ListEntriesRequest{Dir: dir, Recursive: false}); err != nil {
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
		if result, ok, err := r.enumeratePaths(ctx, contract.EnumeratePathsRequest{Target: target, Recursive: true}); err != nil {
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
		if result, ok, err := r.enumeratePaths(ctx, contract.EnumeratePathsRequest{Target: target, Recursive: recursive}); err != nil {
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
			entries, err := contract.ReadManyFromMountRequest(ctx, mount, contract.ReadManyRequest{Paths: groupedPaths, MaxEntries: len(groupedPaths)})
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
		if len(mountedOps) > 0 && len(fsOps) > 0 {
			return contract.MutationResult{}, fmt.Errorf("mixed mounted and filesystem mutation batches are not supported without an explicit transaction contract")
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
		ordered, err := reorderMutationRecords(req.Ops, results)
		if err != nil {
			return contract.MutationResult{}, err
		}
		return contract.MutationResult{Records: ordered}, nil
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
