package mount

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/khicago/simsh/pkg/contract"
)

type externalBinMount struct {
	listExternalCommands func(context.Context) ([]contract.ExternalCommand, error)
	readExternalManual   func(context.Context, string) (string, error)
}

func NewExternalBinMount(
	listExternalCommands func(context.Context) ([]contract.ExternalCommand, error),
	readExternalManual func(context.Context, string) (string, error),
) contract.VirtualMount {
	return &externalBinMount{listExternalCommands: listExternalCommands, readExternalManual: readExternalManual}
}

func (m *externalBinMount) MountPoint() string {
	return contract.VirtualExternalBinDir
}

func (m *externalBinMount) Exists(ctx context.Context) (bool, error) {
	paths, err := m.listPaths(ctx)
	if err != nil {
		return false, err
	}
	return len(paths) > 0, nil
}

func (m *externalBinMount) ListChildren(ctx context.Context, dir string) ([]string, error) {
	dir = normalizeAbsPath(dir)
	if dir != contract.VirtualExternalBinDir {
		return nil, fmt.Errorf("%s: Not a directory", dir)
	}
	return m.listPaths(ctx)
}

func (m *externalBinMount) IsDirPath(ctx context.Context, pathValue string) (bool, error) {
	pathValue = normalizeAbsPath(pathValue)
	if pathValue == contract.VirtualExternalBinDir {
		return m.Exists(ctx)
	}
	if isExecutableUnder(pathValue, contract.VirtualExternalBinDir) {
		exists, err := m.hasPath(ctx, pathValue)
		if err != nil {
			return false, err
		}
		if exists {
			return false, nil
		}
	}
	if strings.HasPrefix(pathValue, contract.VirtualExternalBinDir+"/") {
		return false, nil
	}
	return false, contract.ErrUnsupported
}

func (m *externalBinMount) ReadRawContent(ctx context.Context, pathValue string) (string, error) {
	pathValue = normalizeAbsPath(pathValue)
	if !isExecutableUnder(pathValue, contract.VirtualExternalBinDir) {
		return "", fmt.Errorf("%s: No such file or directory", pathValue)
	}
	name := strings.TrimPrefix(pathValue, contract.VirtualExternalBinDir+"/")
	if m.readExternalManual != nil {
		manual, err := m.readExternalManual(ctx, name)
		if err == nil {
			if strings.TrimSpace(manual) != "" {
				return manual, nil
			}
		} else if !errors.Is(err, contract.ErrUnsupported) {
			return "", err
		}
	}
	desc, found, err := m.lookup(ctx, name)
	if err != nil {
		return "", err
	}
	if !found {
		return "", fmt.Errorf("%s: No such file or directory", pathValue)
	}
	if strings.TrimSpace(desc.Summary) != "" {
		return desc.Summary, nil
	}
	return fmt.Sprintf("binary: %s", desc.Name), nil
}

func (m *externalBinMount) CollectFilesUnder(ctx context.Context, target string) ([]string, error) {
	target = normalizeAbsPath(target)
	if target == contract.VirtualExternalBinDir {
		return m.listPaths(ctx)
	}
	if isExecutableUnder(target, contract.VirtualExternalBinDir) {
		exists, err := m.hasPath(ctx, target)
		if err != nil {
			return nil, err
		}
		if exists {
			return []string{target}, nil
		}
	}
	if strings.HasPrefix(target, contract.VirtualExternalBinDir+"/") {
		return nil, fmt.Errorf("%s: No such file or directory", target)
	}
	return nil, contract.ErrUnsupported
}

func (m *externalBinMount) ResolveSearchPaths(ctx context.Context, target string, recursive bool) ([]string, error) {
	target = normalizeAbsPath(target)
	if target == contract.VirtualExternalBinDir {
		if !recursive {
			return nil, fmt.Errorf("%s: Is a directory (use -r to search recursively)", target)
		}
		return m.listPaths(ctx)
	}
	if isExecutableUnder(target, contract.VirtualExternalBinDir) {
		exists, err := m.hasPath(ctx, target)
		if err != nil {
			return nil, err
		}
		if exists {
			return []string{target}, nil
		}
	}
	if strings.HasPrefix(target, contract.VirtualExternalBinDir+"/") {
		return nil, fmt.Errorf("%s: No such file or directory", target)
	}
	return nil, contract.ErrUnsupported
}

func (m *externalBinMount) DescribePath(ctx context.Context, pathValue string) (contract.PathMeta, error) {
	pathValue = normalizeAbsPath(pathValue)
	if pathValue == contract.VirtualExternalBinDir {
		exists, err := m.Exists(ctx)
		if err != nil {
			return contract.PathMeta{}, err
		}
		if exists {
			return contract.PathMeta{
				Exists:           true,
				IsDir:            true,
				Kind:             "binary_dir",
				Access:           contract.PathAccessReadOnly,
				Capabilities:     []string{contract.PathCapabilityDescribe, contract.PathCapabilityList, contract.PathCapabilitySearch},
				LineCount:        -1,
				FrontMatterLines: -1,
				SpeakerRows:      -1,
				UserRelevance:    "n/a",
			}, nil
		}
	}
	if isExecutableUnder(pathValue, contract.VirtualExternalBinDir) {
		exists, err := m.hasPath(ctx, pathValue)
		if err != nil {
			return contract.PathMeta{}, err
		}
		if exists {
			return contract.PathMeta{
				Exists:           true,
				IsDir:            false,
				Kind:             "binary",
				Access:           contract.PathAccessReadOnly,
				Capabilities:     []string{contract.PathCapabilityDescribe, contract.PathCapabilityRead},
				LineCount:        -1,
				FrontMatterLines: -1,
				SpeakerRows:      -1,
				UserRelevance:    "n/a",
			}, nil
		}
	}
	if strings.HasPrefix(pathValue, contract.VirtualExternalBinDir+"/") {
		return contract.PathMeta{}, fmt.Errorf("%s: No such file or directory", pathValue)
	}
	return contract.PathMeta{}, contract.ErrUnsupported
}

func (m *externalBinMount) listPaths(ctx context.Context) ([]string, error) {
	if m.listExternalCommands == nil {
		return nil, nil
	}
	commands, err := m.listExternalCommands(ctx)
	if err != nil {
		if errors.Is(err, contract.ErrUnsupported) {
			return nil, nil
		}
		return nil, err
	}
	out := make([]string, 0, len(commands))
	seen := map[string]struct{}{}
	for _, command := range commands {
		name := normalizeCommandName(command.Name)
		if name == "" {
			continue
		}
		pathValue := contract.VirtualExternalBinDir + "/" + name
		if _, exists := seen[pathValue]; exists {
			continue
		}
		seen[pathValue] = struct{}{}
		out = append(out, pathValue)
	}
	sort.Strings(out)
	return out, nil
}

func (m *externalBinMount) lookup(ctx context.Context, name string) (contract.ExternalCommand, bool, error) {
	if m.listExternalCommands == nil {
		return contract.ExternalCommand{}, false, nil
	}
	commands, err := m.listExternalCommands(ctx)
	if err != nil {
		if errors.Is(err, contract.ErrUnsupported) {
			return contract.ExternalCommand{}, false, nil
		}
		return contract.ExternalCommand{}, false, err
	}
	target := normalizeCommandName(name)
	for _, command := range commands {
		if normalizeCommandName(command.Name) == target {
			return command, true, nil
		}
	}
	return contract.ExternalCommand{}, false, nil
}

func (m *externalBinMount) hasPath(ctx context.Context, pathValue string) (bool, error) {
	name := strings.TrimPrefix(pathValue, contract.VirtualExternalBinDir+"/")
	_, found, err := m.lookup(ctx, name)
	return found, err
}
