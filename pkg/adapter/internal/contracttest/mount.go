package contracttest

import (
	"context"
	"fmt"
	"slices"
	"testing"

	"github.com/khicago/simsh/pkg/contract"
)

type MountConformanceSpec struct {
	MountPoint                  string
	DirectoryPath               string
	WantDirectoryKind           string
	FilePath                    string
	WantFileKind                string
	MissingPath                 string
	RecursivePath               string
	WantChildren                []string
	WantCollectedFiles          []string
	WantRecursiveSearchPaths    []string
	WantFileLineCount           int
	RequireNonRecursiveDirError bool
}

func (s Snapshot) AssertReadOnlyMountConformance(t *testing.T, spec MountConformanceSpec) {
	t.Helper()
	if err := s.checkReadOnlyMountConformance(spec); err != nil {
		t.Fatal(err)
	}
}

func (s Snapshot) checkReadOnlyMountConformance(spec MountConformanceSpec) error {
	mount, err := requireMountPoint(s.Projection.ProjectionMounts(), spec.MountPoint)
	if err != nil {
		return err
	}
	if mount.MountPoint() != spec.MountPoint {
		return fmt.Errorf("%s mount point = %q, want %q", s.Phase, mount.MountPoint(), spec.MountPoint)
	}
	exists, err := mount.Exists(context.Background())
	if err != nil {
		return fmt.Errorf("%s Exists() error = %v, want nil", s.Phase, err)
	}
	if !exists {
		return fmt.Errorf("%s Exists() = false, want true", s.Phase)
	}

	if len(spec.WantChildren) > 0 {
		children, err := contract.ListMountChildren(context.Background(), mount, spec.DirectoryPath)
		if err != nil {
			return fmt.Errorf("%s ListChildren(%q) error = %v, want nil", s.Phase, spec.DirectoryPath, err)
		}
		if !slices.Equal(children, spec.WantChildren) {
			return fmt.Errorf("%s ListChildren(%q) = %v, want %v", s.Phase, spec.DirectoryPath, children, spec.WantChildren)
		}
	}

	dirIsDir, err := contract.IsMountDir(context.Background(), mount, spec.DirectoryPath)
	if err != nil {
		return fmt.Errorf("%s IsDirPath(%q) error = %v, want nil", s.Phase, spec.DirectoryPath, err)
	}
	if !dirIsDir {
		return fmt.Errorf("%s IsDirPath(%q) = false, want true", s.Phase, spec.DirectoryPath)
	}
	fileIsDir, err := contract.IsMountDir(context.Background(), mount, spec.FilePath)
	if err != nil {
		return fmt.Errorf("%s IsDirPath(%q) error = %v, want nil", s.Phase, spec.FilePath, err)
	}
	if fileIsDir {
		return fmt.Errorf("%s IsDirPath(%q) = true, want false", s.Phase, spec.FilePath)
	}

	if _, err := contract.ReadMountContent(context.Background(), mount, spec.FilePath); err != nil {
		return fmt.Errorf("%s ReadRawContent(%q) error = %v, want nil", s.Phase, spec.FilePath, err)
	}

	if len(spec.WantCollectedFiles) > 0 {
		collected, err := contract.EnumerateMountFiles(context.Background(), mount, spec.RecursivePath, true)
		if err != nil {
			return fmt.Errorf("%s CollectFilesUnder(%q) error = %v, want nil", s.Phase, spec.RecursivePath, err)
		}
		if !slices.Equal(collected, spec.WantCollectedFiles) {
			return fmt.Errorf("%s CollectFilesUnder(%q) = %v, want %v", s.Phase, spec.RecursivePath, collected, spec.WantCollectedFiles)
		}
	}

	fileSearch, err := contract.EnumerateMountFiles(context.Background(), mount, spec.FilePath, false)
	if err != nil {
		return fmt.Errorf("%s ResolveSearchPaths(%q, false) error = %v, want nil", s.Phase, spec.FilePath, err)
	}
	if !slices.Equal(fileSearch, []string{spec.FilePath}) {
		return fmt.Errorf("%s ResolveSearchPaths(%q, false) = %v, want [%q]", s.Phase, spec.FilePath, fileSearch, spec.FilePath)
	}

	if len(spec.WantRecursiveSearchPaths) > 0 {
		searchPaths, err := contract.EnumerateMountFiles(context.Background(), mount, spec.RecursivePath, true)
		if err != nil {
			return fmt.Errorf("%s ResolveSearchPaths(%q, true) error = %v, want nil", s.Phase, spec.RecursivePath, err)
		}
		if !slices.Equal(searchPaths, spec.WantRecursiveSearchPaths) {
			return fmt.Errorf("%s ResolveSearchPaths(%q, true) = %v, want %v", s.Phase, spec.RecursivePath, searchPaths, spec.WantRecursiveSearchPaths)
		}
	}
	if spec.RequireNonRecursiveDirError {
		if _, err := contract.EnumerateMountFiles(context.Background(), mount, spec.RecursivePath, false); err == nil {
			return fmt.Errorf("%s ResolveSearchPaths(%q, false) unexpectedly succeeded", s.Phase, spec.RecursivePath)
		}
	}

	dirMeta, err := contract.DescribeMountPath(context.Background(), mount, spec.DirectoryPath)
	if err != nil {
		return fmt.Errorf("%s DescribePath(%q) error = %v, want nil", s.Phase, spec.DirectoryPath, err)
	}
	if err := assertReadOnlyDirMeta(s.Phase, spec.DirectoryPath, spec.WantDirectoryKind, dirMeta); err != nil {
		return err
	}

	fileMeta, err := contract.DescribeMountPath(context.Background(), mount, spec.FilePath)
	if err != nil {
		return fmt.Errorf("%s DescribePath(%q) error = %v, want nil", s.Phase, spec.FilePath, err)
	}
	if err := assertReadOnlyFileMeta(s.Phase, spec.FilePath, spec.WantFileKind, spec.WantFileLineCount, fileMeta); err != nil {
		return err
	}

	if spec.MissingPath != "" {
		if _, err := contract.DescribeMountPath(context.Background(), mount, spec.MissingPath); err == nil {
			return fmt.Errorf("%s DescribePath(%q) unexpectedly succeeded", s.Phase, spec.MissingPath)
		}
		if _, err := contract.ReadMountContent(context.Background(), mount, spec.MissingPath); err == nil {
			return fmt.Errorf("%s ReadRawContent(%q) unexpectedly succeeded", s.Phase, spec.MissingPath)
		}
	}
	return nil
}

func RequireMountPoint(t *testing.T, mounts []contract.VirtualMount, mountPoint string) contract.VirtualMount {
	t.Helper()
	mount, err := requireMountPoint(mounts, mountPoint)
	if err != nil {
		t.Fatal(err)
	}
	return mount
}

func requireMountPoint(mounts []contract.VirtualMount, mountPoint string) (contract.VirtualMount, error) {
	for _, mount := range mounts {
		if mount.MountPoint() == mountPoint {
			return mount, nil
		}
	}
	return nil, fmt.Errorf("projection mounts = %v, want mount point %q", mountPoints(mounts), mountPoint)
}

func assertReadOnlyDirMeta(phase Phase, pathValue string, wantKind string, meta contract.PathMeta) error {
	if !meta.Exists || !meta.IsDir {
		return fmt.Errorf("%s DescribePath(%q) = %+v, want existing directory metadata", phase, pathValue, meta)
	}
	if wantKind != "" && meta.Kind != wantKind {
		return fmt.Errorf("%s DescribePath(%q).Kind = %q, want %q", phase, pathValue, meta.Kind, wantKind)
	}
	if meta.Access != contract.PathAccessReadOnly {
		return fmt.Errorf("%s DescribePath(%q).Access = %q, want %q", phase, pathValue, meta.Access, contract.PathAccessReadOnly)
	}
	if err := assertCapabilities(phase, pathValue, meta.Capabilities, []string{
		contract.PathCapabilityDescribe,
		contract.PathCapabilityList,
		contract.PathCapabilitySearch,
	}); err != nil {
		return err
	}
	return assertNoWriteCapabilities(phase, pathValue, meta.Capabilities)
}

func assertReadOnlyFileMeta(phase Phase, pathValue string, wantKind string, wantLineCount int, meta contract.PathMeta) error {
	if !meta.Exists || meta.IsDir {
		return fmt.Errorf("%s DescribePath(%q) = %+v, want existing file metadata", phase, pathValue, meta)
	}
	if wantKind != "" && meta.Kind != wantKind {
		return fmt.Errorf("%s DescribePath(%q).Kind = %q, want %q", phase, pathValue, meta.Kind, wantKind)
	}
	if meta.Access != contract.PathAccessReadOnly {
		return fmt.Errorf("%s DescribePath(%q).Access = %q, want %q", phase, pathValue, meta.Access, contract.PathAccessReadOnly)
	}
	if wantLineCount >= 0 && meta.LineCount != wantLineCount {
		return fmt.Errorf("%s DescribePath(%q).LineCount = %d, want %d", phase, pathValue, meta.LineCount, wantLineCount)
	}
	if err := assertCapabilities(phase, pathValue, meta.Capabilities, []string{
		contract.PathCapabilityDescribe,
		contract.PathCapabilityRead,
	}); err != nil {
		return err
	}
	return assertNoWriteCapabilities(phase, pathValue, meta.Capabilities)
}

func assertCapabilities(phase Phase, pathValue string, got []string, required []string) error {
	for _, capability := range required {
		if !slices.Contains(got, capability) {
			return fmt.Errorf("%s DescribePath(%q).Capabilities = %v, missing %q", phase, pathValue, got, capability)
		}
	}
	return nil
}

func assertNoWriteCapabilities(phase Phase, pathValue string, got []string) error {
	deniedCapabilities := []string{
		contract.PathCapabilityWrite,
		contract.PathCapabilityAppend,
		contract.PathCapabilityEdit,
		contract.PathCapabilityMkdir,
		contract.PathCapabilityRemove,
	}
	for _, capability := range deniedCapabilities {
		if slices.Contains(got, capability) {
			return fmt.Errorf("%s DescribePath(%q).Capabilities = %v, contains write capability %q", phase, pathValue, got, capability)
		}
	}
	return nil
}
