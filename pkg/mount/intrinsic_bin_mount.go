package mount

import (
	"context"
	"fmt"
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

func (m *sysBinMount) Exists(ctx context.Context) (bool, error) {
	_ = ctx
	if m.catalog == nil {
		return false, nil
	}
	return len(m.catalog.BuiltinCommandDocs()) > 0, nil
}

func (m *sysBinMount) ListChildren(ctx context.Context, dir string) ([]string, error) {
	_ = ctx
	dir = normalizeAbsPath(dir)
	if dir != contract.VirtualSystemBinDir {
		return nil, fmt.Errorf("%s: Not a directory", dir)
	}
	if m.catalog == nil {
		return []string{}, nil
	}
	docs := m.catalog.BuiltinCommandDocs()
	names := make([]string, 0, len(docs))
	for _, doc := range docs {
		name := strings.TrimSpace(doc.Name)
		if name == "" {
			continue
		}
		names = append(names, name)
	}
	sort.Strings(names)
	out := make([]string, 0, len(names))
	for _, name := range names {
		out = append(out, contract.VirtualSystemBinDir+"/"+name)
	}
	return out, nil
}

func (m *sysBinMount) IsDirPath(ctx context.Context, pathValue string) (bool, error) {
	_ = ctx
	pathValue = normalizeAbsPath(pathValue)
	if pathValue == contract.VirtualSystemBinDir {
		return true, nil
	}
	if strings.HasPrefix(pathValue, contract.VirtualSystemBinDir+"/") {
		return false, nil
	}
	return false, contract.ErrUnsupported
}

func (m *sysBinMount) ReadRawContent(ctx context.Context, pathValue string) (string, error) {
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
		return fmt.Sprintf("builtin: %s", name), nil
	}
	return manual, nil
}

func (m *sysBinMount) CollectFilesUnder(ctx context.Context, target string) ([]string, error) {
	target = normalizeAbsPath(target)
	if target == contract.VirtualSystemBinDir {
		return m.ListChildren(ctx, target)
	}
	if isExecutableUnder(target, contract.VirtualSystemBinDir) {
		name := strings.TrimPrefix(target, contract.VirtualSystemBinDir+"/")
		if m.catalog != nil {
			if _, ok := m.catalog.LookupBuiltinDoc(name); ok {
				return []string{target}, nil
			}
		}
	}
	if strings.HasPrefix(target, contract.VirtualSystemBinDir+"/") {
		return nil, fmt.Errorf("%s: No such file or directory", target)
	}
	return nil, contract.ErrUnsupported
}

func (m *sysBinMount) ResolveSearchPaths(ctx context.Context, target string, recursive bool) ([]string, error) {
	target = normalizeAbsPath(target)
	if isExecutableUnder(target, contract.VirtualSystemBinDir) {
		name := strings.TrimPrefix(target, contract.VirtualSystemBinDir+"/")
		if m.catalog != nil {
			if _, ok := m.catalog.LookupBuiltinDoc(name); ok {
				return []string{target}, nil
			}
		}
	}
	if target == contract.VirtualSystemBinDir {
		if !recursive {
			return nil, fmt.Errorf("%s: Is a directory (use -r to search recursively)", target)
		}
		return m.ListChildren(ctx, target)
	}
	if strings.HasPrefix(target, contract.VirtualSystemBinDir+"/") {
		return nil, fmt.Errorf("%s: No such file or directory", target)
	}
	return nil, contract.ErrUnsupported
}

func (m *sysBinMount) DescribePath(ctx context.Context, pathValue string) (contract.PathMeta, error) {
	_ = ctx
	pathValue = normalizeAbsPath(pathValue)
	if pathValue == contract.VirtualSystemBinDir {
		return contract.PathMeta{
			Exists:           true,
			IsDir:            true,
			Kind:             "sys_bin_dir",
			Access:           contract.PathAccessReadOnly,
			Capabilities:     []string{contract.PathCapabilityDescribe, contract.PathCapabilityList, contract.PathCapabilitySearch},
			LineCount:        -1,
			FrontMatterLines: -1,
			SpeakerRows:      -1,
			UserRelevance:    "n/a",
		}, nil
	}
	if isExecutableUnder(pathValue, contract.VirtualSystemBinDir) {
		name := strings.TrimPrefix(pathValue, contract.VirtualSystemBinDir+"/")
		if m.catalog != nil {
			if _, ok := m.catalog.LookupBuiltinDoc(name); ok {
				return contract.PathMeta{
					Exists:           true,
					IsDir:            false,
					Kind:             "sys_binary",
					Access:           contract.PathAccessReadOnly,
					Capabilities:     []string{contract.PathCapabilityDescribe, contract.PathCapabilityRead},
					LineCount:        -1,
					FrontMatterLines: -1,
					SpeakerRows:      -1,
					UserRelevance:    "n/a",
				}, nil
			}
		}
	}
	if strings.HasPrefix(pathValue, contract.VirtualSystemBinDir+"/") {
		return contract.PathMeta{}, fmt.Errorf("%s: No such file or directory", pathValue)
	}
	return contract.PathMeta{}, contract.ErrUnsupported
}
