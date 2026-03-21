package localfs

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"

	"github.com/khicago/simsh/pkg/adapter/internal/pathguard"
	"github.com/khicago/simsh/pkg/contract"
)

type ExternalCallbacks struct {
	ListExternalCommands func(ctx context.Context) ([]contract.ExternalCommand, error)
	RunExternalCommand   func(ctx context.Context, req contract.ExternalCommandRequest) (contract.ExternalCommandResult, error)
	ReadExternalManual   func(ctx context.Context, command string) (string, error)
}

type Options struct {
	RootDir             string
	Profile             contract.CompatibilityProfile
	Policy              contract.ExecutionPolicy
	CommandAliases      map[string][]string
	EnvVars             map[string]string
	RCFiles             []string
	PathEnv             []string
	VirtualMounts       []contract.VirtualMount
	ExternalCallbacks   ExternalCallbacks
	FormatLSLongRow     contract.LSLongRowFormatter
	DescribePath        func(ctx context.Context, pathValue string) (contract.PathMeta, error)
	AuditSink           contract.AuditSink
	WriteLimitedMaxByte int
}

func NewOps(opts Options) (contract.Ops, error) {
	root := strings.TrimSpace(opts.RootDir)
	if root == "" {
		wd, err := os.Getwd()
		if err != nil {
			return contract.Ops{}, fmt.Errorf("resolve working directory failed: %w", err)
		}
		root = wd
	}
	absRoot, err := filepath.Abs(root)
	if err != nil {
		return contract.Ops{}, fmt.Errorf("resolve root path failed: %w", err)
	}
	root = normalizeCLIPath(absRoot)

	policy := opts.Policy
	if policy.WriteMode == "" {
		policy = contract.DefaultPolicy()
	}
	profile := opts.Profile
	if profile == "" {
		profile = contract.DefaultProfile()
	}
	if _, err := contract.ParseProfile(string(profile)); err != nil {
		return contract.Ops{}, err
	}
	if policy.WriteMode == contract.WriteModeWriteLimited && policy.MaxWriteBytes <= 0 {
		if opts.WriteLimitedMaxByte > 0 {
			policy.MaxWriteBytes = opts.WriteLimitedMaxByte
		} else {
			policy.MaxWriteBytes = 1 << 20
		}
	}

	requireAbsolutePath := func(raw string) (string, error) {
		pathValue := strings.TrimSpace(raw)
		if pathValue == "" {
			return "", fmt.Errorf("path is required")
		}
		if !path.IsAbs(pathValue) {
			return "", fmt.Errorf("path must be absolute: %s", raw)
		}
		normalized := normalizeCLIPath(pathValue)
		if !pathWithinRoot(root, normalized) {
			return "", fmt.Errorf("path is outside root %s: %s", root, normalized)
		}
		// Resolve symlinks early so callers cannot smuggle escape paths that
		// lexically look inside root.
		if _, err := resolveAndCheckPath(root, normalized); err != nil {
			return "", err
		}
		return normalized, nil
	}

	checkPathOp := func(ctx context.Context, op contract.PathOp, pathValue string) error {
		_ = ctx
		switch op {
		case contract.PathOpWrite, contract.PathOpMkdir, contract.PathOpRemove:
			if !policy.AllowWrite() {
				return contract.ErrUnsupported
			}
		}
		_, err := resolveAndCheckPath(root, pathValue)
		return err
	}

	toNativePath := func(pathValue string) string {
		return filepath.FromSlash(pathValue)
	}

	readRawContent := func(ctx context.Context, pathValue string) (string, error) {
		_ = ctx
		native := toNativePath(pathValue)
		if _, err := resolveAndCheckPath(root, pathValue); err != nil {
			return "", err
		}
		raw, err := os.ReadFile(native)
		if err != nil {
			return "", err
		}
		return string(raw), nil
	}

	isDirPath := func(ctx context.Context, pathValue string) (bool, error) {
		_ = ctx
		if _, err := resolveAndCheckPath(root, pathValue); err != nil {
			if os.IsNotExist(err) {
				return false, nil
			}
			return false, err
		}
		info, err := os.Stat(toNativePath(pathValue))
		if err != nil {
			if os.IsNotExist(err) {
				return false, nil
			}
			return false, err
		}
		return info.IsDir(), nil
	}

	listChildren := func(ctx context.Context, dir string) ([]string, error) {
		_ = ctx
		if _, err := resolveAndCheckPath(root, dir); err != nil {
			return nil, err
		}
		entries, err := os.ReadDir(toNativePath(dir))
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

	collectFilesUnder := func(ctx context.Context, target string) ([]string, error) {
		_ = ctx
		if _, err := resolveAndCheckPath(root, target); err != nil {
			return nil, err
		}
		info, err := os.Stat(toNativePath(target))
		if err != nil {
			return nil, err
		}
		if !info.IsDir() {
			return []string{target}, nil
		}
		files := make([]string, 0)
		err = filepath.WalkDir(toNativePath(target), func(p string, d os.DirEntry, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}
			if d.Type()&os.ModeSymlink != 0 {
				return nil
			}
			if d.IsDir() {
				return nil
			}
			files = append(files, normalizeCLIPath(p))
			return nil
		})
		if err != nil {
			return nil, err
		}
		sort.Strings(files)
		return files, nil
	}

	resolveSearchPaths := func(ctx context.Context, target string, recursive bool) ([]string, error) {
		_ = ctx
		if _, err := resolveAndCheckPath(root, target); err != nil {
			return nil, err
		}
		info, err := os.Stat(toNativePath(target))
		if err != nil {
			return nil, err
		}
		if !info.IsDir() {
			return []string{target}, nil
		}
		if !recursive {
			return nil, fmt.Errorf("%s: Is a directory (use -r to search recursively)", target)
		}
		return collectFilesUnder(ctx, target)
	}

	describePath := opts.DescribePath
	if describePath == nil {
		describePath = func(ctx context.Context, pathValue string) (contract.PathMeta, error) {
			isDir, err := isDirPath(ctx, pathValue)
			if err != nil {
				return contract.PathMeta{}, err
			}
			if isDir {
				return contract.PathMeta{
					Exists:           true,
					IsDir:            true,
					Kind:             "dir",
					Access:           contract.PathAccessReadWrite,
					Capabilities:     []string{contract.PathCapabilityDescribe, contract.PathCapabilityList, contract.PathCapabilityMkdir, contract.PathCapabilityRemove, contract.PathCapabilitySearch, contract.PathCapabilityWrite},
					LineCount:        -1,
					FrontMatterLines: -1,
					SpeakerRows:      -1,
					UserRelevance:    "unknown",
				}, nil
			}
			raw, err := readRawContent(ctx, pathValue)
			if err != nil {
				return contract.PathMeta{}, err
			}
			lineCount := len(strings.Split(strings.TrimSuffix(raw, "\n"), "\n"))
			if strings.TrimSpace(raw) == "" {
				lineCount = 0
			}
			return contract.PathMeta{
				Exists:           true,
				IsDir:            false,
				Kind:             "file",
				Access:           contract.PathAccessReadWrite,
				Capabilities:     []string{contract.PathCapabilityAppend, contract.PathCapabilityDescribe, contract.PathCapabilityEdit, contract.PathCapabilityRead, contract.PathCapabilityRemove, contract.PathCapabilityWrite},
				LineCount:        lineCount,
				FrontMatterLines: -1,
				SpeakerRows:      -1,
				UserRelevance:    "unknown",
			}, nil
		}
	}

	writeFile := func(ctx context.Context, filePath string, content string) error {
		_ = ctx
		if !policy.AllowWrite() {
			return contract.ErrUnsupported
		}
		if policy.WriteMode == contract.WriteModeWriteLimited && policy.MaxWriteBytes > 0 && len(content) > policy.MaxWriteBytes {
			return fmt.Errorf("write exceeds write-limited cap (%d bytes)", policy.MaxWriteBytes)
		}
		if _, err := resolveAndCheckPath(root, filePath); err != nil {
			return err
		}
		native := toNativePath(filePath)
		if err := os.MkdirAll(filepath.Dir(native), 0o755); err != nil {
			return err
		}
		return os.WriteFile(native, []byte(content), 0o644)
	}

	appendFile := func(ctx context.Context, filePath string, content string) error {
		_ = ctx
		if !policy.AllowWrite() {
			return contract.ErrUnsupported
		}
		if policy.WriteMode == contract.WriteModeWriteLimited && policy.MaxWriteBytes > 0 && len(content) > policy.MaxWriteBytes {
			return fmt.Errorf("append exceeds write-limited cap (%d bytes)", policy.MaxWriteBytes)
		}
		if _, err := resolveAndCheckPath(root, filePath); err != nil {
			return err
		}
		native := toNativePath(filePath)
		if err := os.MkdirAll(filepath.Dir(native), 0o755); err != nil {
			return err
		}
		f, err := os.OpenFile(native, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
		if err != nil {
			return err
		}
		defer f.Close()
		_, err = f.WriteString(content)
		return err
	}

	editFile := func(ctx context.Context, filePath string, oldString string, newString string, replaceAll bool) error {
		_ = ctx
		if !policy.AllowWrite() {
			return contract.ErrUnsupported
		}
		if _, err := resolveAndCheckPath(root, filePath); err != nil {
			return err
		}
		native := toNativePath(filePath)
		rawBytes, err := os.ReadFile(native)
		if err != nil {
			return err
		}
		raw := string(rawBytes)
		occurrences := strings.Count(raw, oldString)
		if occurrences <= 0 {
			return fmt.Errorf("old string not found")
		}
		if !replaceAll && occurrences > 1 {
			return fmt.Errorf("old string appears %d times", occurrences)
		}
		if replaceAll {
			raw = strings.ReplaceAll(raw, oldString, newString)
		} else {
			raw = strings.Replace(raw, oldString, newString, 1)
		}
		if policy.WriteMode == contract.WriteModeWriteLimited && policy.MaxWriteBytes > 0 && len(raw) > policy.MaxWriteBytes {
			return fmt.Errorf("edit result exceeds write-limited cap (%d bytes)", policy.MaxWriteBytes)
		}
		return os.WriteFile(native, []byte(raw), 0o644)
	}

	makeDir := func(ctx context.Context, dirPath string) error {
		_ = ctx
		if !policy.AllowWrite() {
			return contract.ErrUnsupported
		}
		if _, err := resolveAndCheckPath(root, dirPath); err != nil {
			return err
		}
		native := toNativePath(dirPath)
		return os.MkdirAll(native, 0o755)
	}

	removeFile := func(ctx context.Context, filePath string) error {
		_ = ctx
		if !policy.AllowWrite() {
			return contract.ErrUnsupported
		}
		if _, err := resolveAndCheckPath(root, filePath); err != nil {
			return err
		}
		native := toNativePath(filePath)
		info, err := os.Stat(native)
		if err != nil {
			return err
		}
		if info.IsDir() {
			return fmt.Errorf("cannot remove directory: %s", filePath)
		}
		return os.Remove(native)
	}

	removeDir := func(ctx context.Context, dirPath string) error {
		_ = ctx
		if !policy.AllowWrite() {
			return contract.ErrUnsupported
		}
		if _, err := resolveAndCheckPath(root, dirPath); err != nil {
			return err
		}
		if normalizeCLIPath(dirPath) == root {
			return fmt.Errorf("cannot remove root directory: %s", dirPath)
		}
		native := toNativePath(dirPath)
		info, err := os.Stat(native)
		if err != nil {
			return err
		}
		if !info.IsDir() {
			return fmt.Errorf("not a directory: %s", dirPath)
		}
		return os.Remove(native)
	}

	ops := contract.Ops{
		WorkingDir:           root,
		RootDir:              root,
		RequireAbsolutePath:  requireAbsolutePath,
		ListChildren:         listChildren,
		IsDirPath:            isDirPath,
		DescribePath:         describePath,
		FormatLSLongRow:      opts.FormatLSLongRow,
		ReadRawContent:       readRawContent,
		ResolveSearchPaths:   resolveSearchPaths,
		CollectFilesUnder:    collectFilesUnder,
		WriteFile:            writeFile,
		AppendFile:           appendFile,
		EditFile:             editFile,
		MakeDir:              makeDir,
		RemoveFile:           removeFile,
		RemoveDir:            removeDir,
		CheckPathOp:          checkPathOp,
		ListExternalCommands: opts.ExternalCallbacks.ListExternalCommands,
		RunExternalCommand:   opts.ExternalCallbacks.RunExternalCommand,
		ReadExternalManual:   opts.ExternalCallbacks.ReadExternalManual,
		CommandAliases:       contract.NormalizeCommandAliases(opts.CommandAliases),
		EnvVars:              contract.NormalizeEnvVars(opts.EnvVars),
		RCFiles:              contract.NormalizeRCFiles(opts.RCFiles),
		PathEnv:              append([]string(nil), opts.PathEnv...),
		VirtualMounts:        append([]contract.VirtualMount(nil), opts.VirtualMounts...),
		Profile:              profile,
		Policy:               policy,
		AuditSink:            opts.AuditSink,
	}
	return ops, nil
}

func resolveAndCheckPath(root string, pathValue string) (string, error) {
	native := filepath.FromSlash(pathValue)
	if err := pathguard.Check(filepath.FromSlash(root), native); err != nil {
		if errors.Is(err, pathguard.ErrEscape) {
			return "", fmt.Errorf("path escape is not allowed: %s", pathValue)
		}
		return "", err
	}
	return native, nil
}

func normalizeCLIPath(pathValue string) string {
	clean := filepath.Clean(pathValue)
	return path.Clean(filepath.ToSlash(clean))
}

func pathWithinRoot(root string, candidate string) bool {
	root = normalizeCLIPath(root)
	candidate = normalizeCLIPath(candidate)
	if root == "/" {
		return strings.HasPrefix(candidate, "/")
	}
	return candidate == root || strings.HasPrefix(candidate, root+"/")
}
