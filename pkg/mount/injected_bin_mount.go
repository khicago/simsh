package mount

import (
	"context"
	"errors"
	"fmt"
	"path"
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

func (m *externalBinMount) Profile() contract.MountProfile {
	return contract.NormalizeMountProfile(contract.MountProfile{
		TruthModel:          contract.MountTruthProjection,
		MaterializationMode: contract.MountMaterializationLive,
		WriteSemantics:      contract.MountWriteReadOnly,
		LatencyClass:        contract.MountLatencyLocalHeavy,
		SupportedCLIClasses: []contract.MountCLIClass{
			contract.MountCLIList,
			contract.MountCLITree,
			contract.MountCLIFind,
			contract.MountCLIRead,
		},
	})
}

func (m *externalBinMount) Exists(ctx context.Context) (bool, error) {
	paths, err := m.listPaths(ctx)
	if err != nil {
		return false, err
	}
	return len(paths) > 0, nil
}

func (m *externalBinMount) StatPath(ctx context.Context, pathValue string) (contract.MountEntry, error) {
	pathValue = normalizeAbsPath(pathValue)
	if pathValue == contract.VirtualExternalBinDir {
		exists, err := m.Exists(ctx)
		if err != nil {
			return contract.MountEntry{}, err
		}
		if !exists {
			return contract.MountEntry{}, fmt.Errorf("%s: No such file or directory", pathValue)
		}
		return contract.MountEntry{
			Path: contract.VirtualExternalBinDir,
			Name: path.Base(contract.VirtualExternalBinDir),
			Meta: contract.PathMeta{
				Exists:           true,
				IsDir:            true,
				Kind:             "binary_dir",
				Access:           contract.PathAccessReadOnly,
				Capabilities:     []string{contract.PathCapabilityDescribe, contract.PathCapabilityList, contract.PathCapabilitySearch},
				LineCount:        -1,
				FrontMatterLines: -1,
				SpeakerRows:      -1,
				UserRelevance:    "n/a",
			},
		}, nil
	}
	if !isExecutableUnder(pathValue, contract.VirtualExternalBinDir) {
		if strings.HasPrefix(pathValue, contract.VirtualExternalBinDir+"/") {
			return contract.MountEntry{}, fmt.Errorf("%s: No such file or directory", pathValue)
		}
		return contract.MountEntry{}, contract.ErrUnsupported
	}
	exists, err := m.hasPath(ctx, pathValue)
	if err != nil {
		return contract.MountEntry{}, err
	}
	if !exists {
		return contract.MountEntry{}, fmt.Errorf("%s: No such file or directory", pathValue)
	}
	name := strings.TrimPrefix(pathValue, contract.VirtualExternalBinDir+"/")
	return contract.MountEntry{
		Path: pathValue,
		Name: name,
		Meta: contract.PathMeta{
			Exists:           true,
			IsDir:            false,
			Kind:             "binary",
			Access:           contract.PathAccessReadOnly,
			Capabilities:     []string{contract.PathCapabilityDescribe, contract.PathCapabilityRead},
			LineCount:        -1,
			FrontMatterLines: -1,
			SpeakerRows:      -1,
			UserRelevance:    "n/a",
		},
	}, nil
}

func (m *externalBinMount) ListEntries(ctx context.Context, req contract.ListEntriesRequest) (contract.ListEntriesResult, error) {
	dir := normalizeAbsPath(req.Dir)
	if dir != contract.VirtualExternalBinDir {
		return contract.ListEntriesResult{}, fmt.Errorf("%s: Not a directory", dir)
	}
	paths, err := m.listPaths(ctx)
	if err != nil {
		return contract.ListEntriesResult{}, err
	}
	entries := make([]contract.MountEntry, 0, len(paths))
	for _, pathValue := range paths {
		entry, err := m.StatPath(ctx, pathValue)
		if err != nil {
			return contract.ListEntriesResult{}, err
		}
		entries = append(entries, entry)
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].Path < entries[j].Path })
	return contract.ListEntriesResult{Entries: entries}, nil
}

func (m *externalBinMount) EnumeratePaths(ctx context.Context, req contract.EnumeratePathsRequest) (contract.EnumeratePathsResult, error) {
	target := normalizeAbsPath(req.Target)
	if target == contract.VirtualExternalBinDir {
		if !req.Recursive {
			return contract.EnumeratePathsResult{}, fmt.Errorf("%s: Is a directory (use -r to search recursively)", target)
		}
		paths, err := m.listPaths(ctx)
		if err != nil {
			return contract.EnumeratePathsResult{}, err
		}
		entries := make([]contract.MountEntry, 0, len(paths))
		for _, pathValue := range paths {
			entry, err := m.StatPath(ctx, pathValue)
			if err != nil {
				return contract.EnumeratePathsResult{}, err
			}
			entries = append(entries, entry)
		}
		return contract.EnumeratePathsResult{Entries: entries}, nil
	}
	entry, err := m.StatPath(ctx, target)
	if err != nil {
		return contract.EnumeratePathsResult{}, err
	}
	if entry.Meta.IsDir {
		return contract.EnumeratePathsResult{}, fmt.Errorf("%s: Is a directory (use -r to search recursively)", target)
	}
	return contract.EnumeratePathsResult{Entries: []contract.MountEntry{entry}}, nil
}

func (m *externalBinMount) ReadContent(ctx context.Context, pathValue string) (string, error) {
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
	content := strings.TrimSpace(desc.Summary)
	if content == "" {
		content = fmt.Sprintf("binary: %s", desc.Name)
	}
	return content, nil
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
