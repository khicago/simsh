package contracttest

import (
	"context"
	"slices"
	"testing"

	"github.com/khicago/simsh/pkg/contract"
)

type MountConformanceSpec struct {
	MountPoint               string
	DirectoryPath            string
	WantDirectoryKind        string
	FilePath                 string
	WantFileKind             string
	MissingPath              string
	RecursivePath            string
	WantChildren             []string
	WantCollectedFiles       []string
	WantRecursiveSearchPaths []string
	WantFileLineCount           int
	RequireNonRecursiveDirError bool
}

func (s Snapshot) AssertReadOnlyMountConformance(t *testing.T, spec MountConformanceSpec) {
	t.Helper()

	mount := RequireMountPoint(t, s.Projection.ProjectionMounts(), spec.MountPoint)
	if mount.MountPoint() != spec.MountPoint {
		t.Fatalf("%s mount point = %q, want %q", s.Phase, mount.MountPoint(), spec.MountPoint)
	}
	exists, err := mount.Exists(context.Background())
	if err != nil {
		t.Fatalf("%s Exists() error = %v, want nil", s.Phase, err)
	}
	if !exists {
		t.Fatalf("%s Exists() = false, want true", s.Phase)
	}

	if len(spec.WantChildren) > 0 {
		children, err := mount.ListChildren(context.Background(), spec.DirectoryPath)
		if err != nil {
			t.Fatalf("%s ListChildren(%q) error = %v, want nil", s.Phase, spec.DirectoryPath, err)
		}
		if !slices.Equal(children, spec.WantChildren) {
			t.Fatalf("%s ListChildren(%q) = %v, want %v", s.Phase, spec.DirectoryPath, children, spec.WantChildren)
		}
	}

	dirIsDir, err := mount.IsDirPath(context.Background(), spec.DirectoryPath)
	if err != nil {
		t.Fatalf("%s IsDirPath(%q) error = %v, want nil", s.Phase, spec.DirectoryPath, err)
	}
	if !dirIsDir {
		t.Fatalf("%s IsDirPath(%q) = false, want true", s.Phase, spec.DirectoryPath)
	}
	fileIsDir, err := mount.IsDirPath(context.Background(), spec.FilePath)
	if err != nil {
		t.Fatalf("%s IsDirPath(%q) error = %v, want nil", s.Phase, spec.FilePath, err)
	}
	if fileIsDir {
		t.Fatalf("%s IsDirPath(%q) = true, want false", s.Phase, spec.FilePath)
	}

	if _, err := mount.ReadRawContent(context.Background(), spec.FilePath); err != nil {
		t.Fatalf("%s ReadRawContent(%q) error = %v, want nil", s.Phase, spec.FilePath, err)
	}

	if len(spec.WantCollectedFiles) > 0 {
		collected, err := mount.CollectFilesUnder(context.Background(), spec.RecursivePath)
		if err != nil {
			t.Fatalf("%s CollectFilesUnder(%q) error = %v, want nil", s.Phase, spec.RecursivePath, err)
		}
		if !slices.Equal(collected, spec.WantCollectedFiles) {
			t.Fatalf("%s CollectFilesUnder(%q) = %v, want %v", s.Phase, spec.RecursivePath, collected, spec.WantCollectedFiles)
		}
	}

	fileSearch, err := mount.ResolveSearchPaths(context.Background(), spec.FilePath, false)
	if err != nil {
		t.Fatalf("%s ResolveSearchPaths(%q, false) error = %v, want nil", s.Phase, spec.FilePath, err)
	}
	if !slices.Equal(fileSearch, []string{spec.FilePath}) {
		t.Fatalf("%s ResolveSearchPaths(%q, false) = %v, want [%q]", s.Phase, spec.FilePath, fileSearch, spec.FilePath)
	}

	if len(spec.WantRecursiveSearchPaths) > 0 {
		searchPaths, err := mount.ResolveSearchPaths(context.Background(), spec.RecursivePath, true)
		if err != nil {
			t.Fatalf("%s ResolveSearchPaths(%q, true) error = %v, want nil", s.Phase, spec.RecursivePath, err)
		}
		if !slices.Equal(searchPaths, spec.WantRecursiveSearchPaths) {
			t.Fatalf("%s ResolveSearchPaths(%q, true) = %v, want %v", s.Phase, spec.RecursivePath, searchPaths, spec.WantRecursiveSearchPaths)
		}
	}
	if spec.RequireNonRecursiveDirError {
		if _, err := mount.ResolveSearchPaths(context.Background(), spec.RecursivePath, false); err == nil {
			t.Fatalf("%s ResolveSearchPaths(%q, false) unexpectedly succeeded", s.Phase, spec.RecursivePath)
		}
	}

	dirMeta, err := mount.DescribePath(context.Background(), spec.DirectoryPath)
	if err != nil {
		t.Fatalf("%s DescribePath(%q) error = %v, want nil", s.Phase, spec.DirectoryPath, err)
	}
	assertReadOnlyDirMeta(t, s.Phase, spec.DirectoryPath, spec.WantDirectoryKind, dirMeta)

	fileMeta, err := mount.DescribePath(context.Background(), spec.FilePath)
	if err != nil {
		t.Fatalf("%s DescribePath(%q) error = %v, want nil", s.Phase, spec.FilePath, err)
	}
	assertReadOnlyFileMeta(t, s.Phase, spec.FilePath, spec.WantFileKind, spec.WantFileLineCount, fileMeta)

	if spec.MissingPath != "" {
		if _, err := mount.DescribePath(context.Background(), spec.MissingPath); err == nil {
			t.Fatalf("%s DescribePath(%q) unexpectedly succeeded", s.Phase, spec.MissingPath)
		}
		if _, err := mount.ReadRawContent(context.Background(), spec.MissingPath); err == nil {
			t.Fatalf("%s ReadRawContent(%q) unexpectedly succeeded", s.Phase, spec.MissingPath)
		}
	}
}

func RequireMountPoint(t *testing.T, mounts []contract.VirtualMount, mountPoint string) contract.VirtualMount {
	t.Helper()
	for _, mount := range mounts {
		if mount.MountPoint() == mountPoint {
			return mount
		}
	}
	t.Fatalf("projection mounts = %v, want mount point %q", mountPoints(mounts), mountPoint)
	return nil
}

func assertReadOnlyDirMeta(t *testing.T, phase Phase, pathValue string, wantKind string, meta contract.PathMeta) {
	t.Helper()
	if !meta.Exists || !meta.IsDir {
		t.Fatalf("%s DescribePath(%q) = %+v, want existing directory metadata", phase, pathValue, meta)
	}
	if wantKind != "" && meta.Kind != wantKind {
		t.Fatalf("%s DescribePath(%q).Kind = %q, want %q", phase, pathValue, meta.Kind, wantKind)
	}
	if meta.Access != contract.PathAccessReadOnly {
		t.Fatalf("%s DescribePath(%q).Access = %q, want %q", phase, pathValue, meta.Access, contract.PathAccessReadOnly)
	}
	assertCapabilities(t, phase, pathValue, meta.Capabilities, []string{
		contract.PathCapabilityDescribe,
		contract.PathCapabilityList,
		contract.PathCapabilitySearch,
	})
	assertNoWriteCapabilities(t, phase, pathValue, meta.Capabilities)
}

func assertReadOnlyFileMeta(t *testing.T, phase Phase, pathValue string, wantKind string, wantLineCount int, meta contract.PathMeta) {
	t.Helper()
	if !meta.Exists || meta.IsDir {
		t.Fatalf("%s DescribePath(%q) = %+v, want existing file metadata", phase, pathValue, meta)
	}
	if wantKind != "" && meta.Kind != wantKind {
		t.Fatalf("%s DescribePath(%q).Kind = %q, want %q", phase, pathValue, meta.Kind, wantKind)
	}
	if meta.Access != contract.PathAccessReadOnly {
		t.Fatalf("%s DescribePath(%q).Access = %q, want %q", phase, pathValue, meta.Access, contract.PathAccessReadOnly)
	}
	if wantLineCount >= 0 && meta.LineCount != wantLineCount {
		t.Fatalf("%s DescribePath(%q).LineCount = %d, want %d", phase, pathValue, meta.LineCount, wantLineCount)
	}
	assertCapabilities(t, phase, pathValue, meta.Capabilities, []string{
		contract.PathCapabilityDescribe,
		contract.PathCapabilityRead,
	})
	assertNoWriteCapabilities(t, phase, pathValue, meta.Capabilities)
}

func assertCapabilities(t *testing.T, phase Phase, pathValue string, got []string, required []string) {
	t.Helper()
	for _, capability := range required {
		if !slices.Contains(got, capability) {
			t.Fatalf("%s DescribePath(%q).Capabilities = %v, missing %q", phase, pathValue, got, capability)
		}
	}
}

func assertNoWriteCapabilities(t *testing.T, phase Phase, pathValue string, got []string) {
	t.Helper()
	deniedCapabilities := []string{
		contract.PathCapabilityWrite,
		contract.PathCapabilityAppend,
		contract.PathCapabilityEdit,
		contract.PathCapabilityMkdir,
		contract.PathCapabilityRemove,
	}
	for _, capability := range deniedCapabilities {
		if slices.Contains(got, capability) {
			t.Fatalf("%s DescribePath(%q).Capabilities = %v, contains write capability %q", phase, pathValue, got, capability)
		}
	}
}
