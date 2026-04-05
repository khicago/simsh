package mount

import (
	"context"
	"fmt"
	"path"
	"sort"
	"strings"

	"github.com/khicago/simsh/pkg/contract"
)

type sysBinMount struct {
	catalog contract.BuiltinCatalog
}

func NewSysBinMount(catalog contract.BuiltinCatalog) contract.VirtualMount {
	return &sysBinMount{catalog: catalog}
}

func (m *sysBinMount) MountPoint() string {
	return contract.VirtualSystemBinDir
}

func (m *sysBinMount) Profile() contract.MountProfile {
	return contract.NormalizeMountProfile(contract.MountProfile{
		TruthModel:          contract.MountTruthProjection,
		MaterializationMode: contract.MountMaterializationSnapshot,
		WriteSemantics:      contract.MountWriteReadOnly,
		LatencyClass:        contract.MountLatencyLocalFast,
		SupportedCLIClasses: []contract.MountCLIClass{
			contract.MountCLIList,
			contract.MountCLITree,
			contract.MountCLIFind,
			contract.MountCLIRead,
		},
	})
}

func (m *sysBinMount) Exists(ctx context.Context) (bool, error) {
	_ = ctx
	if m.catalog == nil {
		return false, nil
	}
	return len(m.catalog.BuiltinCommandDocs()) > 0, nil
}

func (m *sysBinMount) StatPath(ctx context.Context, pathValue string) (contract.MountEntry, error) {
	_ = ctx
	pathValue = normalizeAbsPath(pathValue)
	if pathValue == contract.VirtualSystemBinDir {
		return contract.MountEntry{
			Path: contract.VirtualSystemBinDir,
			Name: path.Base(contract.VirtualSystemBinDir),
			Meta: contract.PathMeta{
				Exists:           true,
				IsDir:            true,
				Kind:             "sys_bin_dir",
				Access:           contract.PathAccessReadOnly,
				Capabilities:     []string{contract.PathCapabilityDescribe, contract.PathCapabilityList, contract.PathCapabilitySearch},
				LineCount:        -1,
				FrontMatterLines: -1,
				SpeakerRows:      -1,
				UserRelevance:    "n/a",
			},
		}, nil
	}
	if !isExecutableUnder(pathValue, contract.VirtualSystemBinDir) {
		if strings.HasPrefix(pathValue, contract.VirtualSystemBinDir+"/") {
			return contract.MountEntry{}, fmt.Errorf("%s: No such file or directory", pathValue)
		}
		return contract.MountEntry{}, contract.ErrUnsupported
	}
	name := strings.TrimPrefix(pathValue, contract.VirtualSystemBinDir+"/")
	if m.catalog == nil {
		return contract.MountEntry{}, fmt.Errorf("%s: No such file or directory", pathValue)
	}
	if _, ok := m.catalog.LookupBuiltinDoc(name); !ok {
		return contract.MountEntry{}, fmt.Errorf("%s: No such file or directory", pathValue)
	}
	return contract.MountEntry{
		Path: pathValue,
		Name: name,
		Meta: contract.PathMeta{
			Exists:           true,
			IsDir:            false,
			Kind:             "sys_binary",
			Access:           contract.PathAccessReadOnly,
			Capabilities:     []string{contract.PathCapabilityDescribe, contract.PathCapabilityRead},
			LineCount:        -1,
			FrontMatterLines: -1,
			SpeakerRows:      -1,
			UserRelevance:    "n/a",
		},
	}, nil
}

func (m *sysBinMount) ListEntries(ctx context.Context, req contract.ListEntriesRequest) (contract.ListEntriesResult, error) {
	_ = ctx
	dir := normalizeAbsPath(req.Dir)
	if dir != contract.VirtualSystemBinDir {
		return contract.ListEntriesResult{}, fmt.Errorf("%s: Not a directory", dir)
	}
	if m.catalog == nil {
		return contract.ListEntriesResult{Entries: nil}, nil
	}
	docs := m.catalog.BuiltinCommandDocs()
	entries := make([]contract.MountEntry, 0, len(docs))
	for _, doc := range docs {
		name := strings.TrimSpace(doc.Name)
		if name == "" {
			continue
		}
		entry, err := m.StatPath(ctx, contract.VirtualSystemBinDir+"/"+name)
		if err != nil {
			return contract.ListEntriesResult{}, err
		}
		entries = append(entries, entry)
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].Path < entries[j].Path })
	return contract.ListEntriesResult{Entries: entries}, nil
}

func (m *sysBinMount) EnumeratePaths(ctx context.Context, req contract.EnumeratePathsRequest) (contract.EnumeratePathsResult, error) {
	target := normalizeAbsPath(req.Target)
	if target == contract.VirtualSystemBinDir {
		if !req.Recursive {
			return contract.EnumeratePathsResult{}, fmt.Errorf("%s: Is a directory (use -r to search recursively)", target)
		}
		return enumerateEntriesFromLister(ctx, m, req)
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

func (m *sysBinMount) ReadContent(ctx context.Context, pathValue string) (string, error) {
	_ = ctx
	pathValue = normalizeAbsPath(pathValue)
	if !isExecutableUnder(pathValue, contract.VirtualSystemBinDir) {
		return "", fmt.Errorf("%s: No such file or directory", pathValue)
	}
	name := strings.TrimPrefix(pathValue, contract.VirtualSystemBinDir+"/")
	if m.catalog == nil {
		return "", fmt.Errorf("%s: No such file or directory", pathValue)
	}
	doc, ok := m.catalog.LookupBuiltinDoc(name)
	if !ok {
		return "", fmt.Errorf("%s: No such file or directory", pathValue)
	}
	manual := strings.TrimSpace(doc.Manual)
	if manual == "" {
		manual = fmt.Sprintf("builtin: %s", name)
	}
	return manual, nil
}
