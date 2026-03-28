package agentfs

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/khicago/simsh/pkg/contract"
)

func TestListChildrenAndPathEnvReflectZoneRoots(t *testing.T) {
	fsys := newTestFilesystem(t, t.TempDir(), []string{
		"/task_outputs",
		"/temp_work/../temp_work",
		" knowledge_base ",
		"/",
		"/task_outputs",
	})

	children, err := fsys.ListChildren(context.Background(), "/")
	if err != nil {
		t.Fatalf("ListChildren(/) error = %v", err)
	}
	wantChildren := []string{VirtualKnowledgeRoot, VirtualTaskOutputRoot, VirtualTempWorkRoot}
	if !reflect.DeepEqual(children, wantChildren) {
		t.Fatalf("ListChildren(/) = %#v, want %#v", children, wantChildren)
	}

	wantPathEnv := []string{VirtualKnowledgeRoot, VirtualTaskOutputRoot, VirtualTempWorkRoot}
	if got := fsys.PathEnv(); !reflect.DeepEqual(got, wantPathEnv) {
		t.Fatalf("PathEnv() = %#v, want %#v", got, wantPathEnv)
	}
}

func TestDescribePathReportsZoneAndMarkdownMetadata(t *testing.T) {
	hostRoot := t.TempDir()
	fsys := newTestFilesystem(t, hostRoot, []string{"/task_outputs", "/knowledge_base", "/temp_work"})

	markdownPath := filepath.Join(hostRoot, "knowledge_base", "meeting.md")
	writeFile(t, markdownPath, "---\n{}\n---\nAlice: hello\nBob: hi\nNotes.\n")

	taskOutputsMeta, err := fsys.DescribePath(context.Background(), VirtualTaskOutputRoot)
	if err != nil {
		t.Fatalf("DescribePath(%q) error = %v", VirtualTaskOutputRoot, err)
	}
	if taskOutputsMeta.Kind != "task_output_dir" || taskOutputsMeta.Access != contract.PathAccessReadWrite || !taskOutputsMeta.IsDir {
		t.Fatalf("DescribePath(%q) = %#v, want rw task_output_dir", VirtualTaskOutputRoot, taskOutputsMeta)
	}
	assertCapabilities(t, taskOutputsMeta.Capabilities, []string{
		contract.PathCapabilityDescribe,
		contract.PathCapabilityList,
		contract.PathCapabilitySearch,
		contract.PathCapabilityMkdir,
		contract.PathCapabilityWrite,
	})

	knowledgeMeta, err := fsys.DescribePath(context.Background(), VirtualKnowledgeRoot)
	if err != nil {
		t.Fatalf("DescribePath(%q) error = %v", VirtualKnowledgeRoot, err)
	}
	if knowledgeMeta.Kind != "knowledge_dir" || knowledgeMeta.Access != contract.PathAccessReadOnly || !knowledgeMeta.IsDir {
		t.Fatalf("DescribePath(%q) = %#v, want ro knowledge_dir", VirtualKnowledgeRoot, knowledgeMeta)
	}
	assertCapabilities(t, knowledgeMeta.Capabilities, []string{
		contract.PathCapabilityDescribe,
		contract.PathCapabilityList,
		contract.PathCapabilitySearch,
	})

	fileMeta, err := fsys.DescribePath(context.Background(), "/knowledge_base/meeting.md")
	if err != nil {
		t.Fatalf("DescribePath(meeting.md) error = %v", err)
	}
	if fileMeta.Kind != "markdown" || fileMeta.Access != contract.PathAccessReadOnly || fileMeta.IsDir {
		t.Fatalf("DescribePath(meeting.md) = %#v, want ro markdown file", fileMeta)
	}
	if fileMeta.LineCount != 6 || fileMeta.FrontMatterLines != 3 || fileMeta.SpeakerRows != 2 {
		t.Fatalf("DescribePath(meeting.md) line metadata = %#v, want LineCount=6 FrontMatterLines=3 SpeakerRows=2", fileMeta)
	}
	assertCapabilities(t, fileMeta.Capabilities, []string{
		contract.PathCapabilityDescribe,
		contract.PathCapabilityRead,
	})

	if err := fsys.CheckPathOp(context.Background(), contract.PathOpWrite, "/knowledge_base/new.md"); !errors.Is(err, contract.ErrUnsupported) {
		t.Fatalf("CheckPathOp(write, readonly zone) error = %v, want ErrUnsupported", err)
	}
}

func TestPathResolutionRejectsSymlinkEscapes(t *testing.T) {
	hostRoot := t.TempDir()
	outside := t.TempDir()
	fsys := newTestFilesystem(t, hostRoot, []string{"/task_outputs", "/knowledge_base", "/temp_work"})

	taskOutputs := filepath.Join(hostRoot, "task_outputs")
	if err := os.Symlink(outside, filepath.Join(taskOutputs, "escape")); err != nil {
		t.Fatalf("create escape symlink failed: %v", err)
	}

	taskZone, ok := fsys.matchZone(VirtualTaskOutputRoot)
	if !ok {
		t.Fatalf("matchZone(%q) = false, want true", VirtualTaskOutputRoot)
	}

	for _, candidate := range []string{
		"/task_outputs/escape/pwned.txt",
		"/task_outputs/escape/subdir/pwned.txt",
	} {
		if _, err := fsys.toHostPath(taskZone, candidate); err == nil || !strings.Contains(err.Error(), "path escape is not allowed") {
			t.Fatalf("toHostPath(%q) error = %v, want escape error", candidate, err)
		}
		if _, _, err := fsys.resolvePath(candidate); err == nil || !strings.Contains(err.Error(), "path escape is not allowed") {
			t.Fatalf("resolvePath(%q) error = %v, want escape error", candidate, err)
		}
		if err := fsys.CheckPathOp(context.Background(), contract.PathOpWrite, candidate); err == nil || !strings.Contains(err.Error(), "path escape is not allowed") {
			t.Fatalf("CheckPathOp(write, %q) error = %v, want escape error", candidate, err)
		}
	}
}

func TestRequireAbsolutePathValidatesAndNormalizes(t *testing.T) {
	fsys := newTestFilesystem(t, t.TempDir(), []string{"/task_outputs", "/knowledge_base", "/temp_work"})

	tests := []struct {
		name    string
		input   string
		want    string
		wantErr string
	}{
		{
			name:    "blank path",
			input:   "",
			wantErr: "path is required",
		},
		{
			name:    "relative path",
			input:   "task_outputs/report.txt",
			wantErr: "path must be absolute: task_outputs/report.txt",
		},
		{
			name:  "root path",
			input: " / ",
			want:  "/",
		},
		{
			name:  "normalized zone path",
			input: " /task_outputs/../task_outputs/report.txt ",
			want:  "/task_outputs/report.txt",
		},
		{
			name:    "outside allowed roots",
			input:   "/outside-zone/report.txt",
			wantErr: "path is outside allowed roots: /outside-zone/report.txt",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := fsys.RequireAbsolutePath(tc.input)
			if tc.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
					t.Fatalf("RequireAbsolutePath(%q) error = %v, want substring %q", tc.input, err, tc.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("RequireAbsolutePath(%q) error = %v", tc.input, err)
			}
			if got != tc.want {
				t.Fatalf("RequireAbsolutePath(%q) = %q, want %q", tc.input, got, tc.want)
			}
			assertVirtualPathAllowed(t, "RequireAbsolutePath", got)
		})
	}
}

func TestResolveSearchPathsAndCollectFilesUnderStayVirtualAndSorted(t *testing.T) {
	hostRoot := t.TempDir()
	fsys := newTestFilesystem(t, hostRoot, []string{"/task_outputs", "/knowledge_base", "/temp_work"})
	ctx := context.Background()

	writeFile(t, filepath.Join(hostRoot, "knowledge_base", "docs", "guide.md"), "# guide\n")
	writeFile(t, filepath.Join(hostRoot, "knowledge_base", "docs", "nested", "notes.txt"), "nested\n")
	writeFile(t, filepath.Join(hostRoot, "task_outputs", "reports", "result.json"), "{\"ok\":true}\n")
	writeFile(t, filepath.Join(hostRoot, "temp_work", "scratch", "draft.txt"), "draft\n")

	t.Run("file target", func(t *testing.T) {
		got, err := fsys.ResolveSearchPaths(ctx, "/knowledge_base/docs/guide.md", false)
		if err != nil {
			t.Fatalf("ResolveSearchPaths(%q, false) error = %v", "/knowledge_base/docs/guide.md", err)
		}
		want := []string{"/knowledge_base/docs/guide.md"}
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("ResolveSearchPaths(%q, false) = %#v, want %#v", "/knowledge_base/docs/guide.md", got, want)
		}
		assertVirtualPathsAllowed(t, "ResolveSearchPaths(file)", got)
	})

	t.Run("directory requires recursive flag", func(t *testing.T) {
		_, err := fsys.ResolveSearchPaths(ctx, "/knowledge_base/docs", false)
		if err == nil || !strings.Contains(err.Error(), "Is a directory (use -r to search recursively)") {
			t.Fatalf("ResolveSearchPaths(%q, false) error = %v, want directory recursion guidance", "/knowledge_base/docs", err)
		}
	})

	t.Run("missing zone local target", func(t *testing.T) {
		_, err := fsys.ResolveSearchPaths(ctx, "/knowledge_base/docs/missing.md", false)
		if err == nil || !strings.Contains(err.Error(), "No such file or directory") {
			t.Fatalf("ResolveSearchPaths(%q, false) error = %v, want missing-file error", "/knowledge_base/docs/missing.md", err)
		}
	})

	t.Run("recursive directory traversal", func(t *testing.T) {
		got, err := fsys.ResolveSearchPaths(ctx, "/knowledge_base/docs", true)
		if err != nil {
			t.Fatalf("ResolveSearchPaths(%q, true) error = %v", "/knowledge_base/docs", err)
		}
		want := []string{
			"/knowledge_base/docs/guide.md",
			"/knowledge_base/docs/nested/notes.txt",
		}
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("ResolveSearchPaths(%q, true) = %#v, want %#v", "/knowledge_base/docs", got, want)
		}
		assertVirtualPathsAllowed(t, "ResolveSearchPaths(directory)", got)
	})

	wantAll := []string{
		"/knowledge_base/docs/guide.md",
		"/knowledge_base/docs/nested/notes.txt",
		"/task_outputs/reports/result.json",
		"/temp_work/scratch/draft.txt",
	}

	t.Run("recursive root search is sorted across zones", func(t *testing.T) {
		got, err := fsys.ResolveSearchPaths(ctx, "/", true)
		if err != nil {
			t.Fatalf("ResolveSearchPaths(%q, true) error = %v", "/", err)
		}
		if !reflect.DeepEqual(got, wantAll) {
			t.Fatalf("ResolveSearchPaths(%q, true) = %#v, want %#v", "/", got, wantAll)
		}
		assertVirtualPathsAllowed(t, "ResolveSearchPaths(root)", got)
	})

	t.Run("collect files under root is sorted across zones", func(t *testing.T) {
		got, err := fsys.CollectFilesUnder(ctx, "/")
		if err != nil {
			t.Fatalf("CollectFilesUnder(%q) error = %v", "/", err)
		}
		if !reflect.DeepEqual(got, wantAll) {
			t.Fatalf("CollectFilesUnder(%q) = %#v, want %#v", "/", got, wantAll)
		}
		assertVirtualPathsAllowed(t, "CollectFilesUnder(/)", got)
	})
}

func TestToVirtualPathNormalizesZoneRootsAndRejectsEscapes(t *testing.T) {
	fsys := newTestFilesystem(t, t.TempDir(), []string{"/task_outputs", "/knowledge_base", "/temp_work"})
	taskZone := requireZone(t, fsys, VirtualTaskOutputRoot)

	rootPath, err := fsys.toVirtualPath(taskZone, taskZone.hostRoot)
	if err != nil {
		t.Fatalf("toVirtualPath(%q, %q) error = %v", taskZone.virtualRoot, taskZone.hostRoot, err)
	}
	if rootPath != VirtualTaskOutputRoot {
		t.Fatalf("toVirtualPath(%q, %q) = %q, want %q", taskZone.virtualRoot, taskZone.hostRoot, rootPath, VirtualTaskOutputRoot)
	}
	assertVirtualPathAllowed(t, "toVirtualPath(root)", rootPath)

	nestedHost := filepath.Join(taskZone.hostRoot, "reports", "..", "reports", "run.txt")
	nestedPath, err := fsys.toVirtualPath(taskZone, nestedHost)
	if err != nil {
		t.Fatalf("toVirtualPath(%q, %q) error = %v", taskZone.virtualRoot, nestedHost, err)
	}
	if nestedPath != "/task_outputs/reports/run.txt" {
		t.Fatalf("toVirtualPath(%q, %q) = %q, want %q", taskZone.virtualRoot, nestedHost, nestedPath, "/task_outputs/reports/run.txt")
	}
	assertVirtualPathAllowed(t, "toVirtualPath(nested)", nestedPath)

	outside := filepath.Join(taskZone.hostRoot, "..", "escape.txt")
	if _, err := fsys.toVirtualPath(taskZone, outside); err == nil || !strings.Contains(err.Error(), "outside zone root") {
		t.Fatalf("toVirtualPath(%q, %q) error = %v, want outside-zone rejection", taskZone.virtualRoot, outside, err)
	}
}

func TestNewDefaultFilesystemUsesBaseAndCwdFallback(t *testing.T) {
	t.Run("explicit base path", func(t *testing.T) {
		base := t.TempDir()
		fsys := newDefaultFilesystemForTest(t, base)
		assertDefaultZoneLayout(t, fsys, base)
	})

	t.Run("cwd fallback", func(t *testing.T) {
		base := t.TempDir()
		t.Chdir(base)
		fsys := newDefaultFilesystemForTest(t, "")
		assertDefaultZoneLayout(t, fsys, base)
	})
}

func TestMutationMethodsHandleWritableZones(t *testing.T) {
	hostRoot := t.TempDir()
	fsys := newTestFilesystem(t, hostRoot, []string{"/task_outputs", "/knowledge_base", "/temp_work"})
	ctx := context.Background()

	taskFile := "/task_outputs/reports/run.txt"
	if err := fsys.WriteFile(ctx, taskFile, "draft"); err != nil {
		t.Fatalf("WriteFile(%q) error = %v", taskFile, err)
	}
	assertFileContent(t, filepath.Join(hostRoot, "task_outputs", "reports", "run.txt"), "draft")

	tempFile := "/temp_work/session.txt"
	writeFile(t, filepath.Join(hostRoot, "temp_work", "session.txt"), "alpha beta gamma")
	if err := fsys.AppendFile(ctx, tempFile, " omega delta delta"); err != nil {
		t.Fatalf("AppendFile(%q) error = %v", tempFile, err)
	}
	assertFileContent(t, filepath.Join(hostRoot, "temp_work", "session.txt"), "alpha beta gamma omega delta delta")

	if err := fsys.EditFile(ctx, tempFile, "beta", "BETA", false); err != nil {
		t.Fatalf("EditFile(%q, replaceAll=false) error = %v", tempFile, err)
	}
	assertFileContent(t, filepath.Join(hostRoot, "temp_work", "session.txt"), "alpha BETA gamma omega delta delta")

	if err := fsys.EditFile(ctx, tempFile, "delta", "DELTA", true); err != nil {
		t.Fatalf("EditFile(%q, replaceAll=true) error = %v", tempFile, err)
	}
	assertFileContent(t, filepath.Join(hostRoot, "temp_work", "session.txt"), "alpha BETA gamma omega DELTA DELTA")

	dirPath := "/task_outputs/archive/2026/03"
	if err := fsys.MakeDir(ctx, dirPath); err != nil {
		t.Fatalf("MakeDir(%q) error = %v", dirPath, err)
	}
	assertDirExists(t, filepath.Join(hostRoot, "task_outputs", "archive", "2026", "03"))

	if err := fsys.RemoveFile(ctx, taskFile); err != nil {
		t.Fatalf("RemoveFile(%q) error = %v", taskFile, err)
	}
	assertNotExist(t, filepath.Join(hostRoot, "task_outputs", "reports", "run.txt"))

	if err := fsys.RemoveFile(ctx, dirPath); err == nil || !strings.Contains(err.Error(), "cannot remove directory") {
		t.Fatalf("RemoveFile(%q) error = %v, want directory rejection", dirPath, err)
	}
}

func TestMutationMethodsRejectReadOnlyZone(t *testing.T) {
	hostRoot := t.TempDir()
	fsys := newTestFilesystem(t, hostRoot, []string{"/task_outputs", "/knowledge_base", "/temp_work"})
	ctx := context.Background()

	readonlyHost := filepath.Join(hostRoot, "knowledge_base", "notes.md")
	writeFile(t, readonlyHost, "immutable\n")

	for _, tc := range []struct {
		name string
		run  func() error
	}{
		{
			name: "write",
			run: func() error {
				return fsys.WriteFile(ctx, "/knowledge_base/notes.md", "rewritten\n")
			},
		},
		{
			name: "append",
			run: func() error {
				return fsys.AppendFile(ctx, "/knowledge_base/notes.md", "extra\n")
			},
		},
		{
			name: "edit",
			run: func() error {
				return fsys.EditFile(ctx, "/knowledge_base/notes.md", "immutable", "changed", false)
			},
		},
		{
			name: "mkdir",
			run: func() error {
				return fsys.MakeDir(ctx, "/knowledge_base/new")
			},
		},
		{
			name: "remove",
			run: func() error {
				return fsys.RemoveFile(ctx, "/knowledge_base/notes.md")
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if err := tc.run(); !errors.Is(err, contract.ErrUnsupported) {
				t.Fatalf("%s readonly mutation error = %v, want ErrUnsupported", tc.name, err)
			}
		})
	}

	assertFileContent(t, readonlyHost, "immutable\n")
	assertNotExist(t, filepath.Join(hostRoot, "knowledge_base", "new"))
}

func TestMutationMethodsRespectWriteLimitsWithoutPartialMutation(t *testing.T) {
	hostRoot := t.TempDir()
	fsys := newTestFilesystemWithLimit(t, hostRoot, []string{"/task_outputs", "/knowledge_base", "/temp_work"}, 5)
	ctx := context.Background()

	newFile := filepath.Join(hostRoot, "task_outputs", "too-big.txt")
	if err := fsys.WriteFile(ctx, "/task_outputs/too-big.txt", "123456"); err == nil || !strings.Contains(err.Error(), "write exceeds limit") {
		t.Fatalf("WriteFile(oversized) error = %v, want limit failure", err)
	}
	assertNotExist(t, newFile)

	appendHost := filepath.Join(hostRoot, "temp_work", "append.txt")
	writeFile(t, appendHost, "seed")
	if err := fsys.AppendFile(ctx, "/temp_work/append.txt", "123456"); err == nil || !strings.Contains(err.Error(), "append exceeds limit") {
		t.Fatalf("AppendFile(oversized) error = %v, want limit failure", err)
	}
	assertFileContent(t, appendHost, "seed")

	editHost := filepath.Join(hostRoot, "task_outputs", "edit.txt")
	writeFile(t, editHost, "seed")
	if err := fsys.EditFile(ctx, "/task_outputs/edit.txt", "seed", "123456", false); err == nil || !strings.Contains(err.Error(), "edit result exceeds limit") {
		t.Fatalf("EditFile(oversized) error = %v, want limit failure", err)
	}
	assertFileContent(t, editHost, "seed")
}

func newTestFilesystem(t *testing.T, hostRoot string, pathEnv []string) *aiFilesystem {
	return newTestFilesystemWithLimit(t, hostRoot, pathEnv, 0)
}

func newDefaultFilesystemForTest(t *testing.T, hostRoot string) *aiFilesystem {
	t.Helper()

	fsys, err := NewDefaultFilesystem(hostRoot)
	if err != nil {
		t.Fatalf("NewDefaultFilesystem(%q) error = %v", hostRoot, err)
	}

	typed, ok := fsys.(*aiFilesystem)
	if !ok {
		t.Fatalf("NewDefaultFilesystem(%q) returned %T, want *aiFilesystem", hostRoot, fsys)
	}
	return typed
}

func newTestFilesystemWithLimit(t *testing.T, hostRoot string, pathEnv []string, writeLimit int) *aiFilesystem {
	t.Helper()

	fsys, err := NewFilesystem(Options{
		Zones: []ZoneConfig{
			{VirtualRoot: VirtualTaskOutputRoot, HostRoot: filepath.Join(hostRoot, "task_outputs"), Writable: true, Kind: "task_output_dir"},
			{VirtualRoot: VirtualTempWorkRoot, HostRoot: filepath.Join(hostRoot, "temp_work"), Writable: true, Kind: "temp_work_dir"},
			{VirtualRoot: VirtualKnowledgeRoot, HostRoot: filepath.Join(hostRoot, "knowledge_base"), Writable: false, Kind: "knowledge_dir"},
		},
		EnsureHostRoots:   true,
		PathEnv:           pathEnv,
		WriteLimitedBytes: writeLimit,
	})
	if err != nil {
		t.Fatalf("NewFilesystem(...) error = %v", err)
	}

	typed, ok := fsys.(*aiFilesystem)
	if !ok {
		t.Fatalf("NewFilesystem(...) returned %T, want *aiFilesystem", fsys)
	}
	return typed
}

func requireZone(t *testing.T, fsys *aiFilesystem, virtualRoot string) zone {
	t.Helper()

	z, ok := fsys.matchZone(virtualRoot)
	if !ok {
		t.Fatalf("matchZone(%q) = false, want true", virtualRoot)
	}
	return z
}

func assertCapabilities(t *testing.T, got []string, want []string) {
	t.Helper()
	got = contract.NormalizePathCapabilities(got)
	want = contract.NormalizePathCapabilities(want)
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("capabilities = %#v, want %#v", got, want)
	}
}

func assertDefaultZoneLayout(t *testing.T, fsys *aiFilesystem, base string) {
	t.Helper()

	wantPathEnv := []string{VirtualKnowledgeRoot, VirtualTaskOutputRoot, VirtualTempWorkRoot}
	if got := fsys.PathEnv(); !reflect.DeepEqual(got, wantPathEnv) {
		t.Fatalf("PathEnv() = %#v, want %#v", got, wantPathEnv)
	}

	wantZones := map[string]struct {
		hostRoot string
		writable bool
		kind     string
	}{
		VirtualTaskOutputRoot: {
			hostRoot: filepath.Join(base, "task_outputs"),
			writable: true,
			kind:     "task_output_dir",
		},
		VirtualTempWorkRoot: {
			hostRoot: filepath.Join(base, "temp_work"),
			writable: true,
			kind:     "temp_work_dir",
		},
		VirtualKnowledgeRoot: {
			hostRoot: filepath.Join(base, "knowledge_base"),
			writable: false,
			kind:     "knowledge_dir",
		},
	}

	if len(fsys.zones) != len(wantZones) {
		t.Fatalf("len(zones) = %d, want %d", len(fsys.zones), len(wantZones))
	}

	for virtualRoot, want := range wantZones {
		z := requireZone(t, fsys, virtualRoot)
		if z.hostRoot != want.hostRoot {
			t.Fatalf("zone %q hostRoot = %q, want %q", virtualRoot, z.hostRoot, want.hostRoot)
		}
		if z.writable != want.writable {
			t.Fatalf("zone %q writable = %t, want %t", virtualRoot, z.writable, want.writable)
		}
		if z.kind != want.kind {
			t.Fatalf("zone %q kind = %q, want %q", virtualRoot, z.kind, want.kind)
		}
		info, err := os.Stat(want.hostRoot)
		if err != nil {
			t.Fatalf("os.Stat(%q) error = %v", want.hostRoot, err)
		}
		if !info.IsDir() {
			t.Fatalf("os.Stat(%q).IsDir() = false, want true", want.hostRoot)
		}
	}
}

func writeFile(t *testing.T, pathValue string, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(pathValue), 0o755); err != nil {
		t.Fatalf("mkdir %q failed: %v", filepath.Dir(pathValue), err)
	}
	if err := os.WriteFile(pathValue, []byte(content), 0o644); err != nil {
		t.Fatalf("write %q failed: %v", pathValue, err)
	}
}

func assertFileContent(t *testing.T, pathValue string, want string) {
	t.Helper()
	raw, err := os.ReadFile(pathValue)
	if err != nil {
		t.Fatalf("read %q failed: %v", pathValue, err)
	}
	if got := string(raw); got != want {
		t.Fatalf("content of %q = %q, want %q", pathValue, got, want)
	}
}

func assertDirExists(t *testing.T, pathValue string) {
	t.Helper()
	info, err := os.Stat(pathValue)
	if err != nil {
		t.Fatalf("stat %q failed: %v", pathValue, err)
	}
	if !info.IsDir() {
		t.Fatalf("%q is not a directory", pathValue)
	}
}

func assertNotExist(t *testing.T, pathValue string) {
	t.Helper()
	if _, err := os.Stat(pathValue); !os.IsNotExist(err) {
		t.Fatalf("stat %q = %v, want not-exist", pathValue, err)
	}
}

func assertVirtualPathsAllowed(t *testing.T, fn string, got []string) {
	t.Helper()
	for _, pathValue := range got {
		assertVirtualPathAllowed(t, fn, pathValue)
	}
}

func assertVirtualPathAllowed(t *testing.T, fn string, pathValue string) {
	t.Helper()
	if pathValue == "/" {
		return
	}
	if !strings.HasPrefix(pathValue, "/") {
		t.Fatalf("%s returned %q, want absolute virtual path", fn, pathValue)
	}
	for _, root := range []string{VirtualKnowledgeRoot, VirtualTaskOutputRoot, VirtualTempWorkRoot} {
		if pathValue == root || strings.HasPrefix(pathValue, root+"/") {
			return
		}
	}
	t.Fatalf("%s returned %q, want path under %q, %q, or %q", fn, pathValue, VirtualKnowledgeRoot, VirtualTaskOutputRoot, VirtualTempWorkRoot)
}
