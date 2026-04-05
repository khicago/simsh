package contracttest

import (
	"context"
	"errors"
	"fmt"
	"path"
	"strings"
	"testing"

	"github.com/khicago/simsh/pkg/contract"
)

type fakeConformanceMount struct {
	point string

	exists    bool
	existsErr error

	children    map[string][]string
	childrenErr map[string]error

	dirs      map[string]bool
	isDirErrs map[string]error

	reads    map[string]string
	readErrs map[string]error

	collected    map[string][]string
	collectedErr map[string]error

	resolve    map[string]map[bool][]string
	resolveErr map[string]map[bool]error

	meta    map[string]contract.PathMeta
	metaErr map[string]error
}

func TestMountConformanceSuccess(t *testing.T) {
	snapshot := Snapshot{
		Phase: PhaseCreated,
		Projection: contract.AdapterProjection{
			VirtualMounts: []contract.VirtualMount{newMountFixture()},
		},
	}

	if err := snapshot.checkReadOnlyMountConformance(MountConformanceSpec{
		MountPoint:                  "/docs",
		DirectoryPath:               "/docs",
		WantDirectoryKind:           "docs_dir",
		FilePath:                    "/docs/guide.md",
		WantFileKind:                "docs_file",
		MissingPath:                 "/docs/missing.md",
		RecursivePath:               "/docs",
		WantChildren:                []string{"/docs/guide.md", "/docs/tutorials"},
		WantCollectedFiles:          []string{"/docs/guide.md", "/docs/tutorials/setup.md"},
		WantRecursiveSearchPaths:    []string{"/docs/guide.md", "/docs/tutorials/setup.md"},
		WantFileLineCount:           2,
		RequireNonRecursiveDirError: true,
	}); err != nil {
		t.Fatalf("checkReadOnlyMountConformance(...) error = %v, want nil", err)
	}
}

func TestMountConformanceFailures(t *testing.T) {
	testCases := []struct {
		name string
		run  func() error
	}{
		{
			name: "missing_mount_point",
			run: func() error {
				snapshot := Snapshot{Phase: PhaseCreated}
				return snapshot.checkReadOnlyMountConformance(MountConformanceSpec{MountPoint: "/docs"})
			},
		},
		{
			name: "exists_false",
			run: func() error {
				mount := newMountFixture()
				mount.exists = false
				snapshot := snapshotWithMount(mount)
				return snapshot.checkReadOnlyMountConformance(baseMountSpec())
			},
		},
		{
			name: "children_mismatch",
			run: func() error {
				snapshot := snapshotWithMount(newMountFixture())
				spec := baseMountSpec()
				spec.WantChildren = []string{"/docs/guide.md"}
				return snapshot.checkReadOnlyMountConformance(spec)
			},
		},
		{
			name: "children_error",
			run: func() error {
				mount := newMountFixture()
				mount.childrenErr["/docs"] = errors.New("boom")
				snapshot := snapshotWithMount(mount)
				return snapshot.checkReadOnlyMountConformance(baseMountSpec())
			},
		},
		{
			name: "directory_not_dir",
			run: func() error {
				mount := newMountFixture()
				mount.dirs["/docs"] = false
				snapshot := snapshotWithMount(mount)
				return snapshot.checkReadOnlyMountConformance(baseMountSpec())
			},
		},
		{
			name: "directory_isdir_error",
			run: func() error {
				mount := newMountFixture()
				mount.isDirErrs["/docs"] = errors.New("boom")
				snapshot := snapshotWithMount(mount)
				return snapshot.checkReadOnlyMountConformance(baseMountSpec())
			},
		},
		{
			name: "file_is_dir",
			run: func() error {
				mount := newMountFixture()
				mount.dirs["/docs/guide.md"] = true
				snapshot := snapshotWithMount(mount)
				return snapshot.checkReadOnlyMountConformance(baseMountSpec())
			},
		},
		{
			name: "file_isdir_error",
			run: func() error {
				mount := newMountFixture()
				mount.isDirErrs["/docs/guide.md"] = errors.New("boom")
				snapshot := snapshotWithMount(mount)
				return snapshot.checkReadOnlyMountConformance(baseMountSpec())
			},
		},
		{
			name: "read_error",
			run: func() error {
				mount := newMountFixture()
				mount.readErrs["/docs/guide.md"] = errors.New("boom")
				snapshot := snapshotWithMount(mount)
				return snapshot.checkReadOnlyMountConformance(baseMountSpec())
			},
		},
		{
			name: "collected_error",
			run: func() error {
				mount := newMountFixture()
				mount.collectedErr["/docs"] = errors.New("boom")
				snapshot := snapshotWithMount(mount)
				return snapshot.checkReadOnlyMountConformance(baseMountSpec())
			},
		},
		{
			name: "collected_mismatch",
			run: func() error {
				snapshot := snapshotWithMount(newMountFixture())
				spec := baseMountSpec()
				spec.WantCollectedFiles = []string{"/docs/guide.md"}
				return snapshot.checkReadOnlyMountConformance(spec)
			},
		},
		{
			name: "recursive_search_error",
			run: func() error {
				mount := newMountFixture()
				mount.resolveErr["/docs"][true] = errors.New("boom")
				snapshot := snapshotWithMount(mount)
				return snapshot.checkReadOnlyMountConformance(baseMountSpec())
			},
		},
		{
			name: "recursive_search_mismatch",
			run: func() error {
				snapshot := snapshotWithMount(newMountFixture())
				spec := baseMountSpec()
				spec.WantRecursiveSearchPaths = []string{"/docs/guide.md"}
				return snapshot.checkReadOnlyMountConformance(spec)
			},
		},
		{
			name: "dir_describe_error",
			run: func() error {
				mount := newMountFixture()
				mount.metaErr["/docs"] = errors.New("boom")
				snapshot := snapshotWithMount(mount)
				return snapshot.checkReadOnlyMountConformance(baseMountSpec())
			},
		},
		{
			name: "dir_meta_bad_access",
			run: func() error {
				mount := newMountFixture()
				meta := mount.meta["/docs"]
				meta.Access = contract.PathAccessReadWrite
				mount.meta["/docs"] = meta
				snapshot := snapshotWithMount(mount)
				return snapshot.checkReadOnlyMountConformance(baseMountSpec())
			},
		},
		{
			name: "file_describe_error",
			run: func() error {
				mount := newMountFixture()
				mount.metaErr["/docs/guide.md"] = errors.New("boom")
				snapshot := snapshotWithMount(mount)
				return snapshot.checkReadOnlyMountConformance(baseMountSpec())
			},
		},
		{
			name: "file_meta_missing_capability",
			run: func() error {
				mount := newMountFixture()
				meta := mount.meta["/docs/guide.md"]
				meta.Capabilities = []string{contract.PathCapabilityDescribe}
				mount.meta["/docs/guide.md"] = meta
				snapshot := snapshotWithMount(mount)
				return snapshot.checkReadOnlyMountConformance(baseMountSpec())
			},
		},
		{
			name: "missing_path_read_succeeds",
			run: func() error {
				mount := newMountFixture()
				delete(mount.readErrs, "/docs/missing.md")
				mount.reads["/docs/missing.md"] = "surprise"
				snapshot := snapshotWithMount(mount)
				return snapshot.checkReadOnlyMountConformance(baseMountSpec())
			},
		},
		{
			name: "file_meta_has_write_capability",
			run: func() error {
				mount := newMountFixture()
				meta := mount.meta["/docs/guide.md"]
				meta.Capabilities = append(meta.Capabilities, contract.PathCapabilityWrite)
				mount.meta["/docs/guide.md"] = meta
				snapshot := snapshotWithMount(mount)
				return snapshot.checkReadOnlyMountConformance(baseMountSpec())
			},
		},
		{
			name: "missing_path_describe_succeeds",
			run: func() error {
				mount := newMountFixture()
				delete(mount.metaErr, "/docs/missing.md")
				mount.meta["/docs/missing.md"] = contract.PathMeta{
					Exists:       true,
					IsDir:        false,
					Kind:         "docs_file",
					Access:       contract.PathAccessReadOnly,
					Capabilities: []string{contract.PathCapabilityDescribe, contract.PathCapabilityRead},
				}
				snapshot := snapshotWithMount(mount)
				return snapshot.checkReadOnlyMountConformance(baseMountSpec())
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if err := tc.run(); err == nil {
				t.Fatalf("mount failure case %q unexpectedly succeeded", tc.name)
			}
		})
	}
}

func TestRequireMountPoint(t *testing.T) {
	mount := newMountFixture()
	got := RequireMountPoint(t, []contract.VirtualMount{mount}, "/docs")
	if got != mount {
		t.Fatalf("RequireMountPoint(...) = %v, want %v", got, mount)
	}

	if _, err := requireMountPoint([]contract.VirtualMount{mount}, "/elsewhere"); err == nil {
		t.Fatal("requireMountPoint(...) unexpectedly succeeded for missing mount")
	}
}

func TestMountHelperErrorHelpers(t *testing.T) {
	dirMeta := contract.PathMeta{
		Exists:       true,
		IsDir:        true,
		Kind:         "docs_dir",
		Access:       contract.PathAccessReadOnly,
		Capabilities: []string{contract.PathCapabilityDescribe, contract.PathCapabilityList, contract.PathCapabilitySearch},
	}
	if err := assertReadOnlyDirMeta(PhaseCreated, "/docs", "docs_dir", dirMeta); err != nil {
		t.Fatalf("assertReadOnlyDirMeta(...) error = %v, want nil", err)
	}
	fileMeta := contract.PathMeta{
		Exists:       true,
		IsDir:        false,
		Kind:         "docs_file",
		Access:       contract.PathAccessReadOnly,
		Capabilities: []string{contract.PathCapabilityDescribe, contract.PathCapabilityRead},
		LineCount:    2,
	}
	if err := assertReadOnlyFileMeta(PhaseCreated, "/docs/guide.md", "docs_file", 2, fileMeta); err != nil {
		t.Fatalf("assertReadOnlyFileMeta(...) error = %v, want nil", err)
	}
	if err := assertCapabilities(PhaseCreated, "/docs/guide.md", []string{contract.PathCapabilityDescribe}, []string{contract.PathCapabilityDescribe, contract.PathCapabilityRead}); err == nil {
		t.Fatal("assertCapabilities(...) unexpectedly succeeded")
	}
	if err := assertNoWriteCapabilities(PhaseCreated, "/docs/guide.md", []string{contract.PathCapabilityRead, contract.PathCapabilityWrite}); err == nil {
		t.Fatal("assertNoWriteCapabilities(...) unexpectedly succeeded")
	}
	if err := assertReadOnlyDirMeta(PhaseCreated, "/docs", "other_dir", dirMeta); err == nil {
		t.Fatal("assertReadOnlyDirMeta(kind mismatch) unexpectedly succeeded")
	}
	if err := assertReadOnlyFileMeta(PhaseCreated, "/docs/guide.md", "docs_file", 3, fileMeta); err == nil {
		t.Fatal("assertReadOnlyFileMeta(line count mismatch) unexpectedly succeeded")
	}
	if err := assertReadOnlyFileMeta(PhaseCreated, "/docs/guide.md", "docs_file", 2, contract.PathMeta{Exists: true, IsDir: false, Kind: "docs_file", Access: contract.PathAccessReadWrite, Capabilities: []string{contract.PathCapabilityDescribe, contract.PathCapabilityRead}, LineCount: 2}); err == nil {
		t.Fatal("assertReadOnlyFileMeta(access mismatch) unexpectedly succeeded")
	}
}

func TestReadOnlyMountConformanceWrapper(t *testing.T) {
	snapshot := snapshotWithMount(newMountFixture())
	snapshot.AssertReadOnlyMountConformance(t, baseMountSpec())
}

func newMountFixture() *fakeConformanceMount {
	return &fakeConformanceMount{
		point:  "/docs",
		exists: true,
		children: map[string][]string{
			"/docs":           {"/docs/guide.md", "/docs/tutorials"},
			"/docs/tutorials": {"/docs/tutorials/setup.md"},
		},
		dirs: map[string]bool{
			"/docs":                    true,
			"/docs/tutorials":          true,
			"/docs/guide.md":           false,
			"/docs/tutorials/setup.md": false,
		},
		reads: map[string]string{
			"/docs/guide.md":           "line1\nline2\n",
			"/docs/tutorials/setup.md": "setup\n",
		},
		readErrs: map[string]error{
			"/docs/missing.md": errors.New("missing"),
		},
		collected: map[string][]string{
			"/docs": {"/docs/guide.md", "/docs/tutorials/setup.md"},
		},
		resolve: map[string]map[bool][]string{
			"/docs/guide.md": {false: {"/docs/guide.md"}},
			"/docs":          {true: {"/docs/guide.md", "/docs/tutorials/setup.md"}},
		},
		resolveErr: map[string]map[bool]error{
			"/docs": {false: errors.New("is a directory")},
		},
		meta: map[string]contract.PathMeta{
			"/docs": {
				Exists:       true,
				IsDir:        true,
				Kind:         "docs_dir",
				Access:       contract.PathAccessReadOnly,
				Capabilities: []string{contract.PathCapabilityDescribe, contract.PathCapabilityList, contract.PathCapabilitySearch},
			},
			"/docs/guide.md": {
				Exists:       true,
				IsDir:        false,
				Kind:         "docs_file",
				Access:       contract.PathAccessReadOnly,
				Capabilities: []string{contract.PathCapabilityDescribe, contract.PathCapabilityRead},
				LineCount:    2,
			},
		},
		metaErr: map[string]error{
			"/docs/missing.md": errors.New("missing"),
		},
		childrenErr:  map[string]error{},
		isDirErrs:    map[string]error{},
		collectedErr: map[string]error{},
	}
}

func snapshotWithMount(mount contract.VirtualMount) Snapshot {
	return Snapshot{
		Phase: PhaseCreated,
		Projection: contract.AdapterProjection{
			VirtualMounts: []contract.VirtualMount{mount},
		},
	}
}

func baseMountSpec() MountConformanceSpec {
	return MountConformanceSpec{
		MountPoint:                  "/docs",
		DirectoryPath:               "/docs",
		WantDirectoryKind:           "docs_dir",
		FilePath:                    "/docs/guide.md",
		WantFileKind:                "docs_file",
		MissingPath:                 "/docs/missing.md",
		RecursivePath:               "/docs",
		WantChildren:                []string{"/docs/guide.md", "/docs/tutorials"},
		WantCollectedFiles:          []string{"/docs/guide.md", "/docs/tutorials/setup.md"},
		WantRecursiveSearchPaths:    []string{"/docs/guide.md", "/docs/tutorials/setup.md"},
		WantFileLineCount:           2,
		RequireNonRecursiveDirError: true,
	}
}

func (m *fakeConformanceMount) MountPoint() string { return m.point }
func (m *fakeConformanceMount) Profile() contract.MountProfile {
	return contract.NormalizeMountProfile(contract.MountProfile{
		TruthModel:          contract.MountTruthProjection,
		MaterializationMode: contract.MountMaterializationSnapshot,
		WriteSemantics:      contract.MountWriteReadOnly,
		LatencyClass:        contract.MountLatencyLocalFast,
		SupportedCLIClasses: []contract.MountCLIClass{
			contract.MountCLIList,
			contract.MountCLIFind,
			contract.MountCLIRead,
		},
	})
}
func (m *fakeConformanceMount) Exists(ctx context.Context) (bool, error) {
	_ = ctx
	return m.exists, m.existsErr
}
func (m *fakeConformanceMount) StatPath(ctx context.Context, pathValue string) (contract.MountEntry, error) {
	_ = ctx
	if err, ok := m.isDirErrs[pathValue]; ok {
		return contract.MountEntry{}, err
	}
	if err, ok := m.metaErr[pathValue]; ok {
		return contract.MountEntry{}, err
	}
	meta, ok := m.meta[pathValue]
	if ok {
		if isDir, hasDir := m.dirs[pathValue]; hasDir {
			meta.IsDir = isDir
		}
		return contract.MountEntry{Path: pathValue, Name: path.Base(pathValue), Meta: meta}, nil
	}
	if isDir, hasDir := m.dirs[pathValue]; hasDir {
		meta := contract.PathMeta{
			Exists: true,
			IsDir:  isDir,
			Access: contract.PathAccessReadOnly,
		}
		if isDir {
			meta.Kind = "docs_dir"
			meta.Capabilities = []string{contract.PathCapabilityDescribe, contract.PathCapabilityList, contract.PathCapabilitySearch}
		} else {
			meta.Kind = "docs_file"
			meta.Capabilities = []string{contract.PathCapabilityDescribe, contract.PathCapabilityRead}
			if raw, ok := m.reads[pathValue]; ok {
				meta.LineCount = len(strings.Split(strings.TrimSuffix(raw, "\n"), "\n"))
			}
		}
		return contract.MountEntry{Path: pathValue, Name: path.Base(pathValue), Meta: meta}, nil
	}
	if raw, ok := m.reads[pathValue]; ok {
		return contract.MountEntry{
			Path: pathValue,
			Name: path.Base(pathValue),
			Meta: contract.PathMeta{
				Exists:       true,
				IsDir:        false,
				Kind:         "docs_file",
				Access:       contract.PathAccessReadOnly,
				Capabilities: []string{contract.PathCapabilityDescribe, contract.PathCapabilityRead},
				LineCount:    len(strings.Split(strings.TrimSuffix(raw, "\n"), "\n")),
			},
		}, nil
	}
	return contract.MountEntry{}, fmt.Errorf("missing meta: %s", pathValue)
}
func (m *fakeConformanceMount) ListEntries(ctx context.Context, req contract.ListEntriesRequest) (contract.ListEntriesResult, error) {
	_ = ctx
	if err, ok := m.childrenErr[req.Dir]; ok {
		return contract.ListEntriesResult{}, err
	}
	children := m.children[req.Dir]
	entries := make([]contract.MountEntry, 0, len(children))
	for _, child := range children {
		entry, err := m.StatPath(ctx, child)
		if err != nil {
			return contract.ListEntriesResult{}, err
		}
		entries = append(entries, entry)
	}
	return contract.ListEntriesResult{Entries: entries}, nil
}
func (m *fakeConformanceMount) ReadContent(ctx context.Context, pathValue string) (string, error) {
	_ = ctx
	if err, ok := m.readErrs[pathValue]; ok {
		return "", err
	}
	raw, ok := m.reads[pathValue]
	if !ok {
		return "", fmt.Errorf("missing file: %s", pathValue)
	}
	return raw, nil
}
func (m *fakeConformanceMount) EnumeratePaths(ctx context.Context, req contract.EnumeratePathsRequest) (contract.EnumeratePathsResult, error) {
	_ = ctx
	if errByRecursive, ok := m.resolveErr[req.Target]; ok {
		if err, ok := errByRecursive[req.Recursive]; ok {
			return contract.EnumeratePathsResult{}, err
		}
	}
	if req.Recursive {
		if err, ok := m.collectedErr[req.Target]; ok {
			return contract.EnumeratePathsResult{}, err
		}
	}
	if pathsByRecursive, ok := m.resolve[req.Target]; ok {
		return m.entriesForPaths(ctx, pathsByRecursive[req.Recursive])
	}
	if req.Recursive {
		if collected, ok := m.collected[req.Target]; ok {
			return m.entriesForPaths(ctx, collected)
		}
	}
	if !req.Recursive {
		entry, err := m.StatPath(ctx, req.Target)
		if err != nil {
			return contract.EnumeratePathsResult{}, err
		}
		if entry.Meta.IsDir {
			return contract.EnumeratePathsResult{}, fmt.Errorf("%s: Is a directory (use -r to search recursively)", req.Target)
		}
		return contract.EnumeratePathsResult{Entries: []contract.MountEntry{entry}}, nil
	}
	return contract.EnumeratePathsResult{}, fmt.Errorf("missing search target: %s", req.Target)
}
func (m *fakeConformanceMount) entriesForPaths(ctx context.Context, paths []string) (contract.EnumeratePathsResult, error) {
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
