package mount

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/khicago/simsh/pkg/contract"
)

func TestMountPathHelpers(t *testing.T) {
	tests := []struct {
		name string
		got  string
		want string
	}{
		{name: "normalize empty", got: normalizeAbsPath(""), want: "/"},
		{name: "normalize relative", got: normalizeAbsPath(" a/b "), want: "/a/b"},
		{name: "normalize dotted", got: normalizeAbsPath("/a/../b"), want: "/b"},
		{name: "normalize command name", got: normalizeCommandName(" /rg "), want: "rg"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.got != tt.want {
				t.Errorf("%s = %q, want %q", tt.name, tt.got, tt.want)
			}
		})
	}

	if got := normalizeCommandName("tools/rg"); got != "" {
		t.Errorf("normalizeCommandName(%q) = %q, want empty string", "tools/rg", got)
	}
	if !isExecutableUnder("/sys/bin/cat", contract.VirtualSystemBinDir) {
		t.Errorf("isExecutableUnder(%q, %q) = false, want true", "/sys/bin/cat", contract.VirtualSystemBinDir)
	}
	if isExecutableUnder("/sys/bin/tools/cat", contract.VirtualSystemBinDir) {
		t.Errorf("isExecutableUnder(%q, %q) = true, want false", "/sys/bin/tools/cat", contract.VirtualSystemBinDir)
	}

	lines := splitRawLines("a\nb\n")
	if !reflect.DeepEqual(lines, []string{"a", "b"}) {
		t.Errorf("splitRawLines(...) = %#v, want %#v", lines, []string{"a", "b"})
	}
}

func TestStaticMountOperations(t *testing.T) {
	files := map[string]string{
		"/docs/README.md":        "# docs\n",
		"/docs/scripts/build.sh": "echo hi\n",
		"/docs/nested/guide.txt": "guide\nline2\n",
		"/docs/nested/empty.txt": "",
	}
	rawMount, err := NewStaticMount("/docs", "docs", files)
	if err != nil {
		t.Fatalf("NewStaticMount(...) error = %v", err)
	}
	m := rawMount.(*staticMount)

	if got := m.MountPoint(); got != "/docs" {
		t.Fatalf("MountPoint() = %q, want %q", got, "/docs")
	}
	if exists, err := m.Exists(context.Background()); err != nil || !exists {
		t.Fatalf("Exists() = (%t, %v), want (true, nil)", exists, err)
	}
	children, err := contract.ListMountChildren(context.Background(), m, "/docs")
	if err != nil {
		t.Fatalf("ListChildren(/docs) error = %v", err)
	}
	wantChildren := []string{"/docs/README.md", "/docs/nested", "/docs/scripts"}
	if !reflect.DeepEqual(children, wantChildren) {
		t.Errorf("ListChildren(/docs) = %#v, want %#v", children, wantChildren)
	}

	if isDir, err := contract.IsMountDir(context.Background(), m, "/docs/scripts"); err != nil || !isDir {
		t.Fatalf("IsDirPath(/docs/scripts) = (%t, %v), want (true, nil)", isDir, err)
	}
	if isDir, err := contract.IsMountDir(context.Background(), m, "/docs/README.md"); err != nil || isDir {
		t.Fatalf("IsDirPath(/docs/README.md) = (%t, %v), want (false, nil)", isDir, err)
	}

	if got, err := contract.ReadMountContent(context.Background(), m, "/docs/README.md"); err != nil || got != "# docs\n" {
		t.Fatalf("ReadRawContent(/docs/README.md) = (%q, %v), want (%q, nil)", got, err, "# docs\n")
	}

	filesUnder, err := contract.EnumerateMountFiles(context.Background(), m, "/docs", true)
	if err != nil {
		t.Fatalf("CollectFilesUnder(/docs) error = %v", err)
	}
	wantFiles := []string{"/docs/README.md", "/docs/nested/empty.txt", "/docs/nested/guide.txt", "/docs/scripts/build.sh"}
	if !reflect.DeepEqual(filesUnder, wantFiles) {
		t.Errorf("CollectFilesUnder(/docs) = %#v, want %#v", filesUnder, wantFiles)
	}

	oneFile, err := contract.EnumerateMountFiles(context.Background(), m, "/docs/scripts/build.sh", false)
	if err != nil {
		t.Fatalf("ResolveSearchPaths(/docs/scripts/build.sh, false) error = %v", err)
	}
	if !reflect.DeepEqual(oneFile, []string{"/docs/scripts/build.sh"}) {
		t.Errorf("ResolveSearchPaths(/docs/scripts/build.sh, false) = %#v, want %#v", oneFile, []string{"/docs/scripts/build.sh"})
	}
	if _, err := contract.EnumerateMountFiles(context.Background(), m, "/docs", false); err == nil {
		t.Fatalf("ResolveSearchPaths(/docs, false) unexpectedly succeeded")
	}

	dirMeta, err := contract.DescribeMountPath(context.Background(), m, "/docs/scripts")
	if err != nil {
		t.Fatalf("DescribePath(/docs/scripts) error = %v", err)
	}
	if dirMeta.Kind != "docs_dir" || !dirMeta.IsDir {
		t.Errorf("DescribePath(/docs/scripts) = %#v, want docs_dir metadata", dirMeta)
	}
	fileMeta, err := contract.DescribeMountPath(context.Background(), m, "/docs/scripts/build.sh")
	if err != nil {
		t.Fatalf("DescribePath(/docs/scripts/build.sh) error = %v", err)
	}
	if fileMeta.Kind != "docs_script" || fileMeta.LineCount != 1 {
		t.Errorf("DescribePath(/docs/scripts/build.sh) = %#v, want docs_script with line_count=1", fileMeta)
	}
}

func TestNewStaticMountRejectsInvalidInputs(t *testing.T) {
	tests := []struct {
		name       string
		mountPoint string
		files      map[string]string
	}{
		{name: "root mount point", mountPoint: "/", files: map[string]string{"/README.md": "x"}},
		{name: "outside mount point", mountPoint: "/docs", files: map[string]string{"/elsewhere/a.txt": "x"}},
		{name: "empty file set", mountPoint: "/docs", files: map[string]string{}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := NewStaticMount(tt.mountPoint, "docs", tt.files); err == nil {
				t.Fatalf("NewStaticMount(%q, files=%#v) unexpectedly succeeded", tt.mountPoint, tt.files)
			}
		})
	}
}

func TestBaselineCorpusMountExposesEmbeddedCases(t *testing.T) {
	m, err := NewBaselineCorpusMount()
	if err != nil {
		t.Fatalf("NewBaselineCorpusMount() error = %v", err)
	}

	children, err := contract.ListMountChildren(context.Background(), m, "/test")
	if err != nil {
		t.Fatalf("ListChildren(/test) error = %v", err)
	}
	if !reflect.DeepEqual(children, []string{"/test/README.md", "/test/core-strict"}) {
		t.Fatalf("ListChildren(/test) = %#v, want README + core-strict", children)
	}

	paths, err := contract.EnumerateMountFiles(context.Background(), m, "/test/core-strict/cases", true)
	if err != nil {
		t.Fatalf("CollectFilesUnder(/test/core-strict/cases) error = %v", err)
	}
	wantPaths := []string{
		"/test/core-strict/cases/echo-basic.sh",
		"/test/core-strict/cases/pipeline-basic.sh",
	}
	if !reflect.DeepEqual(paths, wantPaths) {
		t.Errorf("CollectFilesUnder(/test/core-strict/cases) = %#v, want %#v", paths, wantPaths)
	}

	meta, err := contract.DescribeMountPath(context.Background(), m, "/test/core-strict/cases/echo-basic.sh")
	if err != nil {
		t.Fatalf("DescribePath(/test/core-strict/cases/echo-basic.sh) error = %v", err)
	}
	if meta.Kind != "test_script" {
		t.Errorf("DescribePath(/test/core-strict/cases/echo-basic.sh).Kind = %q, want %q", meta.Kind, "test_script")
	}

	if got := baselineProfileFromPath("testdata/baseline/core-strict/cases.json"); got != "core-strict" {
		t.Errorf("baselineProfileFromPath(...) = %q, want %q", got, "core-strict")
	}
	if got := sanitizeMountEntryName(`a /b:c?d`); got != "a__b_c_d" {
		t.Errorf("sanitizeMountEntryName(...) = %q, want %q", got, "a__b_c_d")
	}
}

func TestSysBinMountOperations(t *testing.T) {
	m := NewSysBinMount(fakeCatalog{
		docs: []contract.BuiltinCommandDoc{
			{Name: "echo", Manual: "echo manual"},
			{Name: "cat"},
		},
	})

	if got := m.MountPoint(); got != contract.VirtualSystemBinDir {
		t.Fatalf("MountPoint() = %q, want %q", got, contract.VirtualSystemBinDir)
	}
	if exists, err := m.Exists(context.Background()); err != nil || !exists {
		t.Fatalf("Exists() = (%t, %v), want (true, nil)", exists, err)
	}
	children, err := contract.ListMountChildren(context.Background(), m, contract.VirtualSystemBinDir)
	if err != nil {
		t.Fatalf("ListChildren(%q) error = %v", contract.VirtualSystemBinDir, err)
	}
	wantChildren := []string{"/sys/bin/cat", "/sys/bin/echo"}
	if !reflect.DeepEqual(children, wantChildren) {
		t.Errorf("ListChildren(%q) = %#v, want %#v", contract.VirtualSystemBinDir, children, wantChildren)
	}

	if isDir, err := contract.IsMountDir(context.Background(), m, contract.VirtualSystemBinDir); err != nil || !isDir {
		t.Fatalf("IsDirPath(%q) = (%t, %v), want (true, nil)", contract.VirtualSystemBinDir, isDir, err)
	}
	if isDir, err := contract.IsMountDir(context.Background(), m, "/sys/bin/echo"); err != nil || isDir {
		t.Fatalf("IsDirPath(%q) = (%t, %v), want (false, nil)", "/sys/bin/echo", isDir, err)
	}
	if _, err := contract.IsMountDir(context.Background(), m, "/elsewhere"); !errors.Is(err, contract.ErrUnsupported) {
		t.Fatalf("IsDirPath(%q) error = %v, want ErrUnsupported", "/elsewhere", err)
	}

	if got, err := contract.ReadMountContent(context.Background(), m, "/sys/bin/echo"); err != nil || got != "echo manual" {
		t.Fatalf("ReadRawContent(/sys/bin/echo) = (%q, %v), want (%q, nil)", got, err, "echo manual")
	}
	if got, err := contract.ReadMountContent(context.Background(), m, "/sys/bin/cat"); err != nil || got != "builtin: cat" {
		t.Fatalf("ReadRawContent(/sys/bin/cat) = (%q, %v), want (%q, nil)", got, err, "builtin: cat")
	}

	search, err := contract.EnumerateMountFiles(context.Background(), m, contract.VirtualSystemBinDir, true)
	if err != nil {
		t.Fatalf("ResolveSearchPaths(%q, true) error = %v", contract.VirtualSystemBinDir, err)
	}
	if !reflect.DeepEqual(search, wantChildren) {
		t.Errorf("ResolveSearchPaths(%q, true) = %#v, want %#v", contract.VirtualSystemBinDir, search, wantChildren)
	}

	meta, err := contract.DescribeMountPath(context.Background(), m, "/sys/bin/echo")
	if err != nil {
		t.Fatalf("DescribePath(/sys/bin/echo) error = %v", err)
	}
	if meta.Kind != "sys_binary" || meta.IsDir {
		t.Errorf("DescribePath(/sys/bin/echo) = %#v, want sys_binary file", meta)
	}
}

func TestExternalBinMountOperations(t *testing.T) {
	ctx := context.Background()
	m := NewExternalBinMount(
		func(context.Context) ([]contract.ExternalCommand, error) {
			return []contract.ExternalCommand{
				{Name: " sed ", Summary: "stream editor"},
				{Name: "rg", Summary: "ripgrep"},
				{Name: "sed", Summary: "duplicate ignored"},
			}, nil
		},
		func(_ context.Context, command string) (string, error) {
			if command == "rg" {
				return "rg manual", nil
			}
			return "", contract.ErrUnsupported
		},
	)

	if got := m.MountPoint(); got != contract.VirtualExternalBinDir {
		t.Fatalf("MountPoint() = %q, want %q", got, contract.VirtualExternalBinDir)
	}
	if exists, err := m.Exists(ctx); err != nil || !exists {
		t.Fatalf("Exists() = (%t, %v), want (true, nil)", exists, err)
	}
	children, err := contract.ListMountChildren(ctx, m, contract.VirtualExternalBinDir)
	if err != nil {
		t.Fatalf("ListChildren(%q) error = %v", contract.VirtualExternalBinDir, err)
	}
	wantChildren := []string{"/bin/rg", "/bin/sed"}
	if !reflect.DeepEqual(children, wantChildren) {
		t.Errorf("ListChildren(%q) = %#v, want %#v", contract.VirtualExternalBinDir, children, wantChildren)
	}

	if isDir, err := contract.IsMountDir(ctx, m, contract.VirtualExternalBinDir); err != nil || !isDir {
		t.Fatalf("IsDirPath(%q) = (%t, %v), want (true, nil)", contract.VirtualExternalBinDir, isDir, err)
	}
	if _, err := contract.IsMountDir(ctx, m, "/bin/missing"); err == nil {
		t.Fatalf("IsDirPath(%q) unexpectedly succeeded", "/bin/missing")
	}

	if got, err := contract.ReadMountContent(ctx, m, "/bin/rg"); err != nil || got != "rg manual" {
		t.Fatalf("ReadRawContent(/bin/rg) = (%q, %v), want (%q, nil)", got, err, "rg manual")
	}
	if got, err := contract.ReadMountContent(ctx, m, "/bin/sed"); err != nil || got != "stream editor" {
		t.Fatalf("ReadRawContent(/bin/sed) = (%q, %v), want (%q, nil)", got, err, "stream editor")
	}
	if _, err := contract.ReadMountContent(ctx, m, "/bin/missing"); err == nil {
		t.Fatalf("ReadRawContent(/bin/missing) unexpectedly succeeded")
	}

	files, err := contract.EnumerateMountFiles(ctx, m, contract.VirtualExternalBinDir, true)
	if err != nil {
		t.Fatalf("CollectFilesUnder(%q) error = %v", contract.VirtualExternalBinDir, err)
	}
	if !reflect.DeepEqual(files, wantChildren) {
		t.Errorf("CollectFilesUnder(%q) = %#v, want %#v", contract.VirtualExternalBinDir, files, wantChildren)
	}
	if _, err := contract.EnumerateMountFiles(ctx, m, contract.VirtualExternalBinDir, false); err == nil {
		t.Fatalf("ResolveSearchPaths(%q, false) unexpectedly succeeded", contract.VirtualExternalBinDir)
	}

	meta, err := contract.DescribeMountPath(ctx, m, "/bin/sed")
	if err != nil {
		t.Fatalf("DescribePath(/bin/sed) error = %v", err)
	}
	if meta.Kind != "binary" || meta.IsDir {
		t.Errorf("DescribePath(/bin/sed) = %#v, want binary file metadata", meta)
	}
}

func TestExternalBinMountTreatsUnsupportedListerAsEmpty(t *testing.T) {
	m := NewExternalBinMount(
		func(context.Context) ([]contract.ExternalCommand, error) {
			return nil, contract.ErrUnsupported
		},
		nil,
	)

	exists, err := m.Exists(context.Background())
	if err != nil {
		t.Fatalf("Exists() error = %v", err)
	}
	if exists {
		t.Fatalf("Exists() = true, want false when lister is unsupported")
	}
}

type fakeCatalog struct {
	docs []contract.BuiltinCommandDoc
}

func (c fakeCatalog) BuiltinCommandDocs() []contract.BuiltinCommandDoc {
	return append([]contract.BuiltinCommandDoc(nil), c.docs...)
}

func (c fakeCatalog) LookupBuiltinDoc(name string) (contract.BuiltinCommandDoc, bool) {
	for _, doc := range c.docs {
		if strings.TrimSpace(doc.Name) == name {
			return doc, true
		}
	}
	return contract.BuiltinCommandDoc{}, false
}
