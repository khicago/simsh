package engine

import (
	"context"
	"fmt"
	"path"
	"strings"
	"sync"

	"github.com/khicago/simsh/pkg/contract"
)

type workingDirState struct {
	mu  sync.RWMutex
	cwd string
}

func (s *workingDirState) Get() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.cwd
}

func (s *workingDirState) Set(cwd string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.cwd = normalizeAbsolutePath(cwd)
}

func withPathResolution(ctx context.Context, ops contract.Ops) (contract.Ops, func(), error) {
	baseResolve := ops.RequireAbsolutePath
	baseListChildren := ops.ListChildren
	baseIsDirPath := ops.IsDirPath
	baseListEntries := ops.ListEntries
	baseEnumeratePaths := ops.EnumeratePaths
	baseDescribePath := ops.DescribePath
	baseReadRawContent := ops.ReadRawContent
	baseReadMany := ops.ReadMany
	baseResolveSearchPaths := ops.ResolveSearchPaths
	baseCollectFilesUnder := ops.CollectFilesUnder
	baseSearchContent := ops.SearchContent
	baseWriteFile := ops.WriteFile
	baseAppendFile := ops.AppendFile
	baseEditFile := ops.EditFile
	baseMakeDir := ops.MakeDir
	baseRemoveFile := ops.RemoveFile
	baseRemoveDir := ops.RemoveDir
	baseApplyMutations := ops.ApplyMutations
	baseCheckPathOp := ops.CheckPathOp

	root := normalizeAbsolutePath(ops.RootDir)
	initialWorkingDir, err := resolveInitialWorkingDir(ctx, root, strings.TrimSpace(ops.WorkingDir), baseResolve, baseIsDirPath)
	if err != nil {
		return ops, nil, err
	}
	state := &workingDirState{cwd: initialWorkingDir}

	resolvePath := func(raw string) (string, error) {
		trimmed := strings.TrimSpace(raw)
		if trimmed == "" {
			return "", fmt.Errorf("path is required")
		}
		candidate := trimmed
		if !strings.HasPrefix(trimmed, "/") {
			candidate = path.Join(state.Get(), trimmed)
		}
		return baseResolve(normalizeAbsolutePath(candidate))
	}

	ops.WorkingDir = initialWorkingDir
	ops.GetWorkingDir = state.Get
	ops.ResolvePath = resolvePath
	ops.RequireAbsolutePath = resolvePath
	ops.ChangeWorkingDir = func(ctx context.Context, raw string) (string, error) {
		target := raw
		if strings.TrimSpace(target) == "" {
			target = root
		}
		resolved, err := resolvePath(target)
		if err != nil {
			return "", err
		}
		isDir, err := baseIsDirPath(ctx, resolved)
		if err != nil {
			return "", err
		}
		if !isDir {
			return "", fmt.Errorf("not a directory: %s", resolved)
		}
		state.Set(resolved)
		return resolved, nil
	}

	ops.ListChildren = func(ctx context.Context, dir string) ([]string, error) {
		resolved, err := resolvePath(dir)
		if err != nil {
			return nil, err
		}
		return baseListChildren(ctx, resolved)
	}
	if baseListEntries != nil {
		ops.ListEntries = func(ctx context.Context, req contract.ListEntriesRequest) (contract.ListEntriesResult, error) {
			resolved, err := resolvePath(req.Dir)
			if err != nil {
				return contract.ListEntriesResult{}, err
			}
			req.Dir = resolved
			return baseListEntries(ctx, req)
		}
	}
	if baseEnumeratePaths != nil {
		ops.EnumeratePaths = func(ctx context.Context, req contract.EnumeratePathsRequest) (contract.EnumeratePathsResult, error) {
			resolved, err := resolvePath(req.Target)
			if err != nil {
				return contract.EnumeratePathsResult{}, err
			}
			req.Target = resolved
			return baseEnumeratePaths(ctx, req)
		}
	}
	ops.IsDirPath = func(ctx context.Context, pathValue string) (bool, error) {
		resolved, err := resolvePath(pathValue)
		if err != nil {
			return false, err
		}
		return baseIsDirPath(ctx, resolved)
	}
	if baseDescribePath != nil {
		ops.DescribePath = func(ctx context.Context, pathValue string) (contract.PathMeta, error) {
			resolved, err := resolvePath(pathValue)
			if err != nil {
				return contract.PathMeta{}, err
			}
			return baseDescribePath(ctx, resolved)
		}
	}
	ops.ReadRawContent = func(ctx context.Context, pathValue string) (string, error) {
		resolved, err := resolvePath(pathValue)
		if err != nil {
			return "", err
		}
		return baseReadRawContent(ctx, resolved)
	}
	if baseReadMany != nil {
		ops.ReadMany = func(ctx context.Context, req contract.ReadManyRequest) (contract.ReadManyResult, error) {
			resolved := make([]string, 0, len(req.Paths))
			for _, pathValue := range req.Paths {
				pathResolved, err := resolvePath(pathValue)
				if err != nil {
					return contract.ReadManyResult{}, err
				}
				resolved = append(resolved, pathResolved)
			}
			req.Paths = resolved
			return baseReadMany(ctx, req)
		}
	}
	ops.ResolveSearchPaths = func(ctx context.Context, target string, recursive bool) ([]string, error) {
		resolved, err := resolvePath(target)
		if err != nil {
			return nil, err
		}
		return baseResolveSearchPaths(ctx, resolved, recursive)
	}
	ops.CollectFilesUnder = func(ctx context.Context, target string) ([]string, error) {
		resolved, err := resolvePath(target)
		if err != nil {
			return nil, err
		}
		return baseCollectFilesUnder(ctx, resolved)
	}
	if baseSearchContent != nil {
		ops.SearchContent = func(ctx context.Context, req contract.SearchRequest) (contract.SearchResult, error) {
			if len(req.Targets) == 0 {
				return baseSearchContent(ctx, req)
			}
			resolved := make([]string, 0, len(req.Targets))
			for _, target := range req.Targets {
				pathResolved, err := resolvePath(target)
				if err != nil {
					return contract.SearchResult{}, err
				}
				resolved = append(resolved, pathResolved)
			}
			req.Targets = resolved
			return baseSearchContent(ctx, req)
		}
	}
	ops.WriteFile = func(ctx context.Context, filePath string, content string) error {
		resolved, err := resolvePath(filePath)
		if err != nil {
			return err
		}
		return baseWriteFile(ctx, resolved, content)
	}
	ops.AppendFile = func(ctx context.Context, filePath string, content string) error {
		resolved, err := resolvePath(filePath)
		if err != nil {
			return err
		}
		return baseAppendFile(ctx, resolved, content)
	}
	ops.EditFile = func(ctx context.Context, filePath string, oldString string, newString string, replaceAll bool) error {
		resolved, err := resolvePath(filePath)
		if err != nil {
			return err
		}
		return baseEditFile(ctx, resolved, oldString, newString, replaceAll)
	}
	ops.MakeDir = func(ctx context.Context, dirPath string) error {
		resolved, err := resolvePath(dirPath)
		if err != nil {
			return err
		}
		return baseMakeDir(ctx, resolved)
	}
	ops.RemoveFile = func(ctx context.Context, filePath string) error {
		resolved, err := resolvePath(filePath)
		if err != nil {
			return err
		}
		return baseRemoveFile(ctx, resolved)
	}
	ops.RemoveDir = func(ctx context.Context, dirPath string) error {
		resolved, err := resolvePath(dirPath)
		if err != nil {
			return err
		}
		return baseRemoveDir(ctx, resolved)
	}
	if baseApplyMutations != nil {
		ops.ApplyMutations = func(ctx context.Context, req contract.MutationBatch) (contract.MutationResult, error) {
			resolved := make([]contract.MutationSpec, 0, len(req.Ops))
			for _, op := range req.Ops {
				pathResolved, err := resolvePath(op.Path)
				if err != nil {
					return contract.MutationResult{}, err
				}
				op.Path = pathResolved
				resolved = append(resolved, op)
			}
			req.Ops = resolved
			return baseApplyMutations(ctx, req)
		}
	}
	if baseCheckPathOp != nil {
		ops.CheckPathOp = func(ctx context.Context, op contract.PathOp, pathValue string) error {
			resolved, err := resolvePath(pathValue)
			if err != nil {
				return err
			}
			return baseCheckPathOp(ctx, op, resolved)
		}
	}
	return ops, func() {
		state.Set(initialWorkingDir)
	}, nil
}

func resolveInitialWorkingDir(
	ctx context.Context,
	root string,
	raw string,
	resolveAbsolute func(string) (string, error),
	isDir func(context.Context, string) (bool, error),
) (string, error) {
	if strings.TrimSpace(raw) == "" {
		return root, nil
	}
	candidate := strings.TrimSpace(raw)
	if !strings.HasPrefix(candidate, "/") {
		candidate = path.Join(root, candidate)
	}
	resolved, err := resolveAbsolute(normalizeAbsolutePath(candidate))
	if err != nil {
		return "", err
	}
	ok, err := isDir(ctx, resolved)
	if err != nil {
		return "", err
	}
	if !ok {
		return "", fmt.Errorf("working directory is not a directory: %s", resolved)
	}
	return resolved, nil
}
