package builtin

import (
	"context"
	"path/filepath"
	"strings"
	"testing"

	"github.com/khicago/simsh/pkg/adapter/localfs"
	"github.com/khicago/simsh/pkg/contract"
	"github.com/khicago/simsh/pkg/engine"
)

type testRuntime struct {
	eng  *engine.Engine
	root string
	ops  contract.Ops
}

func newTestRuntime(t *testing.T) *testRuntime {
	t.Helper()
	root := t.TempDir()
	realRoot := root
	if resolved, err := filepath.EvalSymlinks(root); err == nil {
		realRoot = resolved
	}
	reg := engine.NewRegistry()
	RegisterDefaults(reg)
	eng := engine.New(reg)
	ops, err := localfs.NewOps(localfs.Options{
		RootDir:   realRoot,
		Profile:   contract.ProfileBashPlus,
		Policy:    contract.ExecutionPolicy{WriteMode: contract.WriteModeFull, MaxWriteBytes: 1 << 20, MaxPipelineDepth: 16, MaxOutputBytes: 4 << 20, Timeout: contract.DefaultPolicy().Timeout},
		PathEnv:   nil,
		AuditSink: nil,
	})
	if err != nil {
		t.Fatalf("new localfs ops failed: %v", err)
	}
	return &testRuntime{eng: eng, root: filepath.ToSlash(realRoot), ops: ops}
}

func (rt *testRuntime) abs(parts ...string) string {
	all := append([]string{rt.root}, parts...)
	return filepath.ToSlash(filepath.Join(all...))
}

func (rt *testRuntime) exec(cmdline string) (string, int) {
	return rt.eng.Execute(context.Background(), cmdline, rt.ops)
}

func TestBuiltinCommandCoverage(t *testing.T) {
	rt := newTestRuntime(t)

	readme := rt.abs("workspace", "readme.md")
	logs := rt.abs("workspace", "logs.txt")
	other := rt.abs("workspace", "other.txt")
	fmDoc := rt.abs("workspace", "frontmatter.md")
	jsonDoc := rt.abs("workspace", "data.json")
	copyTarget := rt.abs("workspace", "copy.md")
	mvTarget := rt.abs("workspace", "moved.txt")
	newFile := rt.abs("workspace", "new.txt")
	emptyDir := rt.abs("workspace", "empty-dir")
	nestedDir := rt.abs("workspace", "nested")
	deeperDir := rt.abs("workspace", "nested", "deeper")
	nestedFile := rt.abs("workspace", "nested", "deeper", "notes.txt")

	if out, code := rt.exec("mkdir -p " + rt.abs("workspace")); code != 0 {
		t.Fatalf("setup mkdir failed: code=%d out=%q", code, out)
	}
	if out, code := rt.exec("mkdir -p " + emptyDir); code != 0 {
		t.Fatalf("setup empty dir failed: code=%d out=%q", code, out)
	}
	if out, code := rt.exec("mkdir -p " + deeperDir); code != 0 {
		t.Fatalf("setup nested dir failed: code=%d out=%q", code, out)
	}
	if out, code := rt.exec("mkdir -p " + rt.abs("workspace", "rmdir-confirm")); code != 0 {
		t.Fatalf("setup rmdir-confirm dir failed: code=%d out=%q", code, out)
	}
	if out, code := rt.exec("mkdir -p " + rt.abs("workspace", "rmdir-json")); code != 0 {
		t.Fatalf("setup rmdir-json dir failed: code=%d out=%q", code, out)
	}
	if err := rt.ops.WriteFile(context.Background(), readme, "hello\nworld\n"); err != nil {
		t.Fatalf("setup write readme failed: %v", err)
	}
	if err := rt.ops.WriteFile(context.Background(), logs, "3\n1\n3\n"); err != nil {
		t.Fatalf("setup write logs failed: %v", err)
	}
	if err := rt.ops.WriteFile(context.Background(), other, "hello\nworld\n"); err != nil {
		t.Fatalf("setup write other failed: %v", err)
	}
	if err := rt.ops.WriteFile(context.Background(), rt.abs("workspace", "mv-confirm-src.txt"), "abc"); err != nil {
		t.Fatalf("setup write mv-confirm-src failed: %v", err)
	}
	if err := rt.ops.WriteFile(context.Background(), rt.abs("workspace", "mv-json-src.txt"), "abc"); err != nil {
		t.Fatalf("setup write mv-json-src failed: %v", err)
	}
	if err := rt.ops.WriteFile(context.Background(), rt.abs("workspace", "rm-confirm.txt"), "abc"); err != nil {
		t.Fatalf("setup write rm-confirm failed: %v", err)
	}
	if err := rt.ops.WriteFile(context.Background(), rt.abs("workspace", "rm-json.txt"), "abc"); err != nil {
		t.Fatalf("setup write rm-json failed: %v", err)
	}
	if err := rt.ops.WriteFile(context.Background(), fmDoc, "---\ntitle: Coverage Fixture\ntags:\n  - a\n  - b\n---\nbody\n"); err != nil {
		t.Fatalf("setup write frontmatter fixture failed: %v", err)
	}
	if err := rt.ops.WriteFile(context.Background(), jsonDoc, "{\n  \"title\": \"Coverage Fixture\",\n  \"meta\": {\"author\": \"simsh\"},\n  \"items\": [{\"name\": \"first\"}, {\"name\": \"second\"}]\n}\n"); err != nil {
		t.Fatalf("setup write json fixture failed: %v", err)
	}
	if err := rt.ops.WriteFile(context.Background(), rt.abs("workspace", ".hidden.txt"), "hidden\n"); err != nil {
		t.Fatalf("setup write hidden file failed: %v", err)
	}
	if err := rt.ops.WriteFile(context.Background(), nestedFile, "nested\n"); err != nil {
		t.Fatalf("setup write nested file failed: %v", err)
	}

	tests := []struct {
		name string
		cmd  string
		want func(t *testing.T, out string, code int)
	}{
		{
			name: "ls",
			cmd:  "ls " + rt.abs("workspace"),
			want: func(t *testing.T, out string, code int) {
				if code != 0 || !strings.Contains(out, readme) {
					t.Fatalf("ls failed: code=%d out=%q", code, out)
				}
			},
		},
		{
			name: "ls-recursive-long",
			cmd:  "ls -R -l " + rt.abs("workspace"),
			want: func(t *testing.T, out string, code int) {
				if code != 0 {
					t.Fatalf("ls -R -l failed: code=%d out=%q", code, out)
				}
				for _, want := range []string{
					rt.abs("workspace") + ":",
					nestedDir + ":",
					deeperDir + ":",
					"- rw file 1 " + nestedFile,
					"# columns: mode access kind lines path",
				} {
					if !strings.Contains(out, want) {
						t.Fatalf("ls -R -l missing %q in output:\n%s", want, out)
					}
				}
			},
		},
		{
			name: "ls-long-md",
			cmd:  "ls -al --fmt md " + deeperDir,
			want: func(t *testing.T, out string, code int) {
				if code != 0 {
					t.Fatalf("ls -al --fmt md failed: code=%d out=%q", code, out)
				}
				for _, want := range []string{
					"| mode | access | kind | lines | path |",
					"| d | rw | dir | - | . |",
					"| d | rw | dir | - | .. |",
					"| - | rw | file | 1 | " + nestedFile + " |",
				} {
					if !strings.Contains(out, want) {
						t.Fatalf("ls -al --fmt md missing %q in output:\n%s", want, out)
					}
				}
			},
		},
		{
			name: "pwd",
			cmd:  "pwd",
			want: func(t *testing.T, out string, code int) {
				if code != 0 || strings.TrimSpace(out) != rt.root {
					t.Fatalf("pwd failed: code=%d out=%q root=%q", code, out, rt.root)
				}
			},
		},
		{
			name: "cd-pwd",
			cmd:  "cd workspace; pwd",
			want: func(t *testing.T, out string, code int) {
				if code != 0 || strings.TrimSpace(out) != rt.abs("workspace") {
					t.Fatalf("cd/pwd failed: code=%d out=%q", code, out)
				}
			},
		},
		{
			name: "cd-relative-cat",
			cmd:  "cd workspace; cat readme.md",
			want: func(t *testing.T, out string, code int) {
				if code != 0 || !strings.Contains(out, "hello") {
					t.Fatalf("cd/cat failed: code=%d out=%q", code, out)
				}
			},
		},
		{
			name: "tree",
			cmd:  "tree " + rt.abs("workspace"),
			want: func(t *testing.T, out string, code int) {
				if code != 0 || !strings.Contains(out, "readme.md") {
					t.Fatalf("tree failed: code=%d out=%q", code, out)
				}
				if !strings.Contains(out, rt.abs("workspace")+"/") {
					t.Fatalf("tree should show the root directory in outline format: %q", out)
				}
				if strings.Contains(out, "|--") || strings.Contains(out, "`--") {
					t.Fatalf("tree default should no longer use ASCII branch art: %q", out)
				}
				if strings.Contains(out, ".hidden.txt") {
					t.Fatalf("tree should hide dot files by default: %q", out)
				}
			},
		},
		{
			name: "tree-all",
			cmd:  "tree -a " + rt.abs("workspace"),
			want: func(t *testing.T, out string, code int) {
				if code != 0 || !strings.Contains(out, ".hidden.txt") {
					t.Fatalf("tree -a failed: code=%d out=%q", code, out)
				}
			},
		},
		{
			name: "tree-ascii",
			cmd:  "tree --fmt ascii " + rt.abs("workspace"),
			want: func(t *testing.T, out string, code int) {
				if code != 0 || !strings.Contains(out, "|--") {
					t.Fatalf("tree --fmt ascii failed: code=%d out=%q", code, out)
				}
			},
		},
		{
			name: "tree-json",
			cmd:  "tree --fmt json " + rt.abs("workspace"),
			want: func(t *testing.T, out string, code int) {
				if code != 0 || !strings.Contains(out, "\"entries\"") || !strings.Contains(out, "\"kind\": \"dir\"") {
					t.Fatalf("tree --fmt json failed: code=%d out=%q", code, out)
				}
			},
		},
		{
			name: "env",
			cmd:  "env PATH",
			want: func(t *testing.T, out string, code int) {
				if code != 0 || !strings.Contains(out, "/sys/bin:/bin") {
					t.Fatalf("env failed: code=%d out=%q", code, out)
				}
			},
		},
		{
			name: "env-json",
			cmd:  "env --json",
			want: func(t *testing.T, out string, code int) {
				if code != 0 || !strings.Contains(out, "\"vars\"") || !strings.Contains(out, "\"key\":\"PATH\"") {
					t.Fatalf("env --json failed: code=%d out=%q", code, out)
				}
			},
		},
		{
			name: "env-json-key",
			cmd:  "env --json PATH",
			want: func(t *testing.T, out string, code int) {
				if code != 0 || !strings.Contains(out, "\"key\":\"PATH\"") || !strings.Contains(out, "\"parts\":[\"/sys/bin\",\"/bin\"]") {
					t.Fatalf("env --json PATH failed: code=%d out=%q", code, out)
				}
			},
		},
		{
			name: "env-split",
			cmd:  "env --split PATH",
			want: func(t *testing.T, out string, code int) {
				if code != 0 || strings.TrimSpace(out) != "/sys/bin\n/bin" {
					t.Fatalf("env --split PATH failed: code=%d out=%q", code, out)
				}
			},
		},
		{
			name: "frontmatter-stat",
			cmd:  "frontmatter stat " + fmDoc,
			want: func(t *testing.T, out string, code int) {
				if code != 0 || !strings.Contains(out, "y 1:6") || !strings.Contains(out, "title,tags") {
					t.Fatalf("frontmatter stat failed: code=%d out=%q", code, out)
				}
				if !strings.Contains(out, "# columns: has fm_lines key_count keys path") {
					t.Fatalf("frontmatter stat missing legend: %q", out)
				}
			},
		},
		{
			name: "frontmatter-stat-md",
			cmd:  "frontmatter stat --fmt md " + fmDoc,
			want: func(t *testing.T, out string, code int) {
				if code != 0 || !strings.HasPrefix(out, "| has | fm_lines | key_count | keys | path |") {
					t.Fatalf("frontmatter stat --fmt md failed: code=%d out=%q", code, out)
				}
			},
		},
		{
			name: "frontmatter-stat-json",
			cmd:  "frontmatter stat --fmt json " + fmDoc,
			want: func(t *testing.T, out string, code int) {
				if code != 0 || !strings.Contains(out, "\"entries\"") || !strings.Contains(out, "\"has_frontmatter\":true") {
					t.Fatalf("frontmatter stat --fmt json failed: code=%d out=%q", code, out)
				}
			},
		},
		{
			name: "frontmatter-get-key",
			cmd:  "frontmatter get --key title " + fmDoc,
			want: func(t *testing.T, out string, code int) {
				if code != 0 || strings.TrimSpace(out) != "Coverage Fixture" {
					t.Fatalf("frontmatter get --key failed: code=%d out=%q", code, out)
				}
			},
		},
		{
			name: "frontmatter-print-context",
			cmd:  "frontmatter print --key tags -C 1 -n " + fmDoc,
			want: func(t *testing.T, out string, code int) {
				if code != 0 || !strings.Contains(out, "3:tags:") || !strings.Contains(out, "5:  - b") {
					t.Fatalf("frontmatter print failed: code=%d out=%q", code, out)
				}
			},
		},
		{
			name: "json-stat",
			cmd:  "json stat " + jsonDoc,
			want: func(t *testing.T, out string, code int) {
				if code != 0 || !strings.Contains(out, "y object 3") || !strings.Contains(out, jsonDoc) {
					t.Fatalf("json stat failed: code=%d out=%q", code, out)
				}
			},
		},
		{
			name: "json-stat-json",
			cmd:  "json stat --fmt json " + jsonDoc,
			want: func(t *testing.T, out string, code int) {
				if code != 0 || !strings.Contains(out, "\"entries\"") || !strings.Contains(out, "\"kind\":\"object\"") {
					t.Fatalf("json stat --fmt json failed: code=%d out=%q", code, out)
				}
			},
		},
		{
			name: "json-get-path",
			cmd:  "json get --path items[0].name " + jsonDoc,
			want: func(t *testing.T, out string, code int) {
				if code != 0 || strings.TrimSpace(out) != "first" {
					t.Fatalf("json get failed: code=%d out=%q", code, out)
				}
			},
		},
		{
			name: "json-get-raw",
			cmd:  "json get --raw --path meta " + jsonDoc,
			want: func(t *testing.T, out string, code int) {
				if code != 0 || strings.TrimSpace(out) != "{\"author\":\"simsh\"}" {
					t.Fatalf("json get --raw failed: code=%d out=%q", code, out)
				}
			},
		},
		{
			name: "json-stat-invalid-format",
			cmd:  "json stat --fmt yaml " + jsonDoc,
			want: func(t *testing.T, out string, code int) {
				if code != contract.ExitCodeUsage || !strings.Contains(out, `json stat: unsupported --fmt value "yaml"`) {
					t.Fatalf("json stat invalid format failed: code=%d out=%q", code, out)
				}
			},
		},
		{
			name: "json-stat-no-files",
			cmd:  "json stat " + emptyDir,
			want: func(t *testing.T, out string, code int) {
				if code != contract.ExitCodeGeneral || !strings.Contains(out, "json stat: no files found") {
					t.Fatalf("json stat empty dir failed: code=%d out=%q", code, out)
				}
			},
		},
		{
			name: "json-get-bad-flag",
			cmd:  "json get --bad --path meta " + jsonDoc,
			want: func(t *testing.T, out string, code int) {
				if code != contract.ExitCodeUsage || !strings.Contains(out, "json get: unsupported flag --bad") {
					t.Fatalf("json get bad flag failed: code=%d out=%q", code, out)
				}
			},
		},
		{
			name: "json-get-directory",
			cmd:  "json get --path meta " + rt.abs("workspace"),
			want: func(t *testing.T, out string, code int) {
				if code != contract.ExitCodeUsage || !strings.Contains(out, "json get: "+rt.abs("workspace")+": is a directory") {
					t.Fatalf("json get directory failed: code=%d out=%q", code, out)
				}
			},
		},
		{
			name: "cat",
			cmd:  "cat -n " + readme,
			want: func(t *testing.T, out string, code int) {
				if code != 0 || !strings.Contains(out, "1:hello") {
					t.Fatalf("cat failed: code=%d out=%q", code, out)
				}
			},
		},
		{
			name: "head",
			cmd:  "head -n 1 " + readme,
			want: func(t *testing.T, out string, code int) {
				if code != 0 || strings.TrimSpace(out) != "hello" {
					t.Fatalf("head failed: code=%d out=%q", code, out)
				}
			},
		},
		{
			name: "tail",
			cmd:  "tail -n 1 " + readme,
			want: func(t *testing.T, out string, code int) {
				if code != 0 || strings.TrimSpace(out) != "world" {
					t.Fatalf("tail failed: code=%d out=%q", code, out)
				}
			},
		},
		{
			name: "grep",
			cmd:  "grep hello " + readme,
			want: func(t *testing.T, out string, code int) {
				if code != 0 || !strings.Contains(out, ":hello") {
					t.Fatalf("grep failed: code=%d out=%q", code, out)
				}
			},
		},
		{
			name: "grep-jsonl",
			cmd:  "grep --fmt jsonl hello " + readme,
			want: func(t *testing.T, out string, code int) {
				if code != 0 || !strings.Contains(out, "\"kind\":\"match\"") || !strings.Contains(out, "\"path\":\""+readme+"\"") {
					t.Fatalf("grep --fmt jsonl failed: code=%d out=%q", code, out)
				}
			},
		},
		{
			name: "grep-jsonl-stdin",
			cmd:  "echo hello | grep --fmt jsonl hello",
			want: func(t *testing.T, out string, code int) {
				if code != 0 || !strings.Contains(out, "\"stdin\":true") || !strings.Contains(out, "\"kind\":\"match\"") {
					t.Fatalf("grep --fmt jsonl stdin failed: code=%d out=%q", code, out)
				}
			},
		},
		{
			name: "grep-jsonl-list",
			cmd:  "grep -l --fmt jsonl hello " + readme,
			want: func(t *testing.T, out string, code int) {
				if code != 0 || !strings.Contains(out, "\"kind\":\"file\"") || !strings.Contains(out, "\"path\":\""+readme+"\"") {
					t.Fatalf("grep -l --fmt jsonl failed: code=%d out=%q", code, out)
				}
			},
		},
		{
			name: "find",
			cmd:  "find " + rt.abs("workspace") + " -name \"*.md\"",
			want: func(t *testing.T, out string, code int) {
				if code != 0 || !strings.Contains(out, readme) {
					t.Fatalf("find failed: code=%d out=%q", code, out)
				}
			},
		},
		{
			name: "find-jsonl",
			cmd:  "find " + rt.abs("workspace") + " -name \"*.md\" --fmt jsonl",
			want: func(t *testing.T, out string, code int) {
				if code != 0 || !strings.Contains(out, "\"path\":\""+readme+"\"") || !strings.Contains(out, "\"kind\":\"file\"") {
					t.Fatalf("find --fmt jsonl failed: code=%d out=%q", code, out)
				}
			},
		},
		{
			name: "find-jsonl-relative",
			cmd:  "cd workspace; find . -name \"*.md\" --fmt jsonl",
			want: func(t *testing.T, out string, code int) {
				if code != 0 || !strings.Contains(out, "\"name\":\"readme.md\"") {
					t.Fatalf("find relative --fmt jsonl failed: code=%d out=%q", code, out)
				}
			},
		},
		{
			name: "which",
			cmd:  "which ls",
			want: func(t *testing.T, out string, code int) {
				if code != 0 || strings.TrimSpace(out) != "/sys/bin/ls" {
					t.Fatalf("which failed: code=%d out=%q", code, out)
				}
			},
		},
		{
			name: "which-json",
			cmd:  "which --fmt json ls missing_cmd",
			want: func(t *testing.T, out string, code int) {
				if code == 0 || !strings.Contains(out, "\"resolved_path\":\"/sys/bin/ls\"") || !strings.Contains(out, "\"error\":\"which: missing_cmd: not found\"") {
					t.Fatalf("which --fmt json failed: code=%d out=%q", code, out)
				}
			},
		},
		{
			name: "type",
			cmd:  "type ls",
			want: func(t *testing.T, out string, code int) {
				if code != 0 || strings.TrimSpace(out) != "ls builtin /sys/bin/ls" {
					t.Fatalf("type failed: code=%d out=%q", code, out)
				}
			},
		},
		{
			name: "type-json",
			cmd:  "type --json ls",
			want: func(t *testing.T, out string, code int) {
				if code != 0 || !strings.Contains(out, "\"name\":\"ls\"") || !strings.Contains(out, "\"kind\":\"builtin\"") || !strings.Contains(out, "\"target\":\"/sys/bin/ls\"") {
					t.Fatalf("type --json failed: code=%d out=%q", code, out)
				}
			},
		},
		{
			name: "echo",
			cmd:  "echo hello world",
			want: func(t *testing.T, out string, code int) {
				if code != 0 || strings.TrimSpace(out) != "hello world" {
					t.Fatalf("echo failed: code=%d out=%q", code, out)
				}
			},
		},
		{
			name: "tee",
			cmd:  "echo tee-data | tee " + rt.abs("workspace", "tee.txt"),
			want: func(t *testing.T, out string, code int) {
				if code != 0 || strings.TrimSpace(out) != "tee-data" {
					t.Fatalf("tee failed: code=%d out=%q", code, out)
				}
			},
		},
		{
			name: "tee-confirm",
			cmd:  "echo tee-data | tee --confirm " + rt.abs("workspace", "tee-confirm.txt"),
			want: func(t *testing.T, out string, code int) {
				expected := "wrote " + rt.abs("workspace", "tee-confirm.txt") + " bytes=8 mode=write"
				if code != 0 || strings.TrimSpace(out) != expected {
					t.Fatalf("tee --confirm failed: code=%d out=%q", code, out)
				}
			},
		},
		{
			name: "tee-json",
			cmd:  "echo tee-data | tee --json " + rt.abs("workspace", "tee-json.txt"),
			want: func(t *testing.T, out string, code int) {
				if code != 0 || !strings.Contains(out, "\"path\":\""+rt.abs("workspace", "tee-json.txt")+"\"") || !strings.Contains(out, "\"bytes\":8") || !strings.Contains(out, "\"mode\":\"write\"") {
					t.Fatalf("tee --json failed: code=%d out=%q", code, out)
				}
			},
		},
		{
			name: "sed",
			cmd:  "sed -i 's/hello/hi/' " + readme + "; cat " + readme,
			want: func(t *testing.T, out string, code int) {
				if code != 0 || !strings.Contains(out, "hi") {
					t.Fatalf("sed failed: code=%d out=%q", code, out)
				}
			},
		},
		{
			name: "sed-json",
			cmd:  "sed -i --json 's/body/text/' " + fmDoc,
			want: func(t *testing.T, out string, code int) {
				if code != 0 || !strings.Contains(out, "\"path\":\""+fmDoc+"\"") || !strings.Contains(out, "\"mode\":\"in_place_edit\"") || !strings.Contains(out, "\"old\":\"body\"") || !strings.Contains(out, "\"new\":\"text\"") {
					t.Fatalf("sed --json failed: code=%d out=%q", code, out)
				}
			},
		},
		{
			name: "man",
			cmd:  "man -v ls",
			want: func(t *testing.T, out string, code int) {
				if code != 0 || !strings.Contains(out, "SYNOPSIS") {
					t.Fatalf("man failed: code=%d out=%q", code, out)
				}
				if strings.HasPrefix(strings.TrimSpace(out), "---") {
					t.Fatalf("man verbose should strip frontmatter: %q", out)
				}
			},
		},
		{
			name: "man-summary-guidance",
			cmd:  "man ls",
			want: func(t *testing.T, out string, code int) {
				if code != 0 || !strings.Contains(out, "Use-When:") || !strings.Contains(out, "Avoid-When:") {
					t.Fatalf("man summary guidance missing: code=%d out=%q", code, out)
				}
				if !strings.Contains(out, "Contract:") || !strings.Contains(out, "pipe behavior:") {
					t.Fatalf("man summary contract missing: code=%d out=%q", code, out)
				}
			},
		},
		{
			name: "man-list-json",
			cmd:  "man --list --fmt json",
			want: func(t *testing.T, out string, code int) {
				if code != 0 || !strings.Contains(out, "\"builtins\"") || !strings.Contains(out, "\"name\": \"ls\"") {
					t.Fatalf("man --list --fmt json failed: code=%d out=%q", code, out)
				}
			},
		},
		{
			name: "man-list-text",
			cmd:  "man --list",
			want: func(t *testing.T, out string, code int) {
				if code != 0 {
					t.Fatalf("man --list failed: code=%d out=%q", code, out)
				}
				for _, want := range []string{
					"Builtins:",
					"name",
					"stdin",
					"structured",
					"ls",
				} {
					if !strings.Contains(out, want) {
						t.Fatalf("man --list missing %q in output:\n%s", want, out)
					}
				}
			},
		},
		{
			name: "date",
			cmd:  "date +%F",
			want: func(t *testing.T, out string, code int) {
				if code != 0 || len(strings.TrimSpace(out)) != 10 {
					t.Fatalf("date failed: code=%d out=%q", code, out)
				}
			},
		},
		{
			name: "mkdir",
			cmd:  "mkdir -p " + rt.abs("workspace", "a", "b"),
			want: func(t *testing.T, out string, code int) {
				if code != 0 || strings.TrimSpace(out) != "" {
					t.Fatalf("mkdir failed: code=%d out=%q", code, out)
				}
			},
		},
		{
			name: "mkdir-confirm",
			cmd:  "mkdir --confirm " + rt.abs("workspace", "confirm-dir"),
			want: func(t *testing.T, out string, code int) {
				if code != 0 || strings.TrimSpace(out) != "created "+rt.abs("workspace", "confirm-dir") {
					t.Fatalf("mkdir --confirm failed: code=%d out=%q", code, out)
				}
			},
		},
		{
			name: "mkdir-json",
			cmd:  "mkdir --json -p " + rt.abs("workspace", "cache"),
			want: func(t *testing.T, out string, code int) {
				if code != 0 || !strings.Contains(out, "\"status\":\"created\"") || !strings.Contains(out, "\"path\":\""+rt.abs("workspace", "cache")+"\"") {
					t.Fatalf("mkdir --json failed: code=%d out=%q", code, out)
				}
			},
		},
		{
			name: "mkdir-confirm-exists",
			cmd:  "mkdir --confirm -p " + rt.abs("workspace"),
			want: func(t *testing.T, out string, code int) {
				if code != 0 || strings.TrimSpace(out) != "exists "+rt.abs("workspace") {
					t.Fatalf("mkdir --confirm existing failed: code=%d out=%q", code, out)
				}
			},
		},
		{
			name: "cp",
			cmd:  "cp " + readme + " " + copyTarget,
			want: func(t *testing.T, out string, code int) {
				if code != 0 || strings.TrimSpace(out) != "" {
					t.Fatalf("cp failed: code=%d out=%q", code, out)
				}
			},
		},
		{
			name: "cp-confirm",
			cmd:  "cp --confirm " + readme + " " + rt.abs("workspace", "copy-confirm.md"),
			want: func(t *testing.T, out string, code int) {
				expected := "copied " + readme + " -> " + rt.abs("workspace", "copy-confirm.md")
				if code != 0 || strings.TrimSpace(out) != expected {
					t.Fatalf("cp --confirm failed: code=%d out=%q", code, out)
				}
			},
		},
		{
			name: "cp-json",
			cmd:  "cp --json " + readme + " " + rt.abs("workspace", "copy-json.md"),
			want: func(t *testing.T, out string, code int) {
				if code != 0 || !strings.Contains(out, "\"src\":\""+readme+"\"") || !strings.Contains(out, "\"dest\":\""+rt.abs("workspace", "copy-json.md")+"\"") || !strings.Contains(out, "\"bytes\":9") {
					t.Fatalf("cp --json failed: code=%d out=%q", code, out)
				}
			},
		},
		{
			name: "mv",
			cmd:  "mv " + logs + " " + mvTarget,
			want: func(t *testing.T, out string, code int) {
				if code != 0 || strings.TrimSpace(out) != "" {
					t.Fatalf("mv failed: code=%d out=%q", code, out)
				}
			},
		},
		{
			name: "mv-confirm",
			cmd:  "mv --confirm " + rt.abs("workspace", "mv-confirm-src.txt") + " " + rt.abs("workspace", "mv-confirm-dest.txt"),
			want: func(t *testing.T, out string, code int) {
				expected := "moved " + rt.abs("workspace", "mv-confirm-src.txt") + " -> " + rt.abs("workspace", "mv-confirm-dest.txt")
				if code != 0 || strings.TrimSpace(out) != expected {
					t.Fatalf("mv --confirm failed: code=%d out=%q", code, out)
				}
			},
		},
		{
			name: "mv-json",
			cmd:  "mv --json " + rt.abs("workspace", "mv-json-src.txt") + " " + rt.abs("workspace", "mv-json-dest.txt"),
			want: func(t *testing.T, out string, code int) {
				if code != 0 || !strings.Contains(out, "\"src\":\""+rt.abs("workspace", "mv-json-src.txt")+"\"") || !strings.Contains(out, "\"dest\":\""+rt.abs("workspace", "mv-json-dest.txt")+"\"") || !strings.Contains(out, "\"bytes\":3") {
					t.Fatalf("mv --json failed: code=%d out=%q", code, out)
				}
			},
		},
		{
			name: "touch",
			cmd:  "touch " + newFile,
			want: func(t *testing.T, out string, code int) {
				if code != 0 || strings.TrimSpace(out) != "" {
					t.Fatalf("touch failed: code=%d out=%q", code, out)
				}
			},
		},
		{
			name: "touch-json-created",
			cmd:  "touch --json " + rt.abs("workspace", "created-by-json.txt"),
			want: func(t *testing.T, out string, code int) {
				if code != 0 || !strings.Contains(out, "\"status\":\"created\"") {
					t.Fatalf("touch --json created failed: code=%d out=%q", code, out)
				}
			},
		},
		{
			name: "touch-json-existing",
			cmd:  "touch --json " + readme,
			want: func(t *testing.T, out string, code int) {
				if code != 0 || !strings.Contains(out, "\"status\":\"already_exists\"") || !strings.Contains(out, "\"path\":\""+readme+"\"") {
					t.Fatalf("touch --json existing failed: code=%d out=%q", code, out)
				}
			},
		},
		{
			name: "wc",
			cmd:  "wc -l " + readme,
			want: func(t *testing.T, out string, code int) {
				if code != 0 || strings.TrimSpace(out) != "2" {
					t.Fatalf("wc failed: code=%d out=%q", code, out)
				}
			},
		},
		{
			name: "wc-default",
			cmd:  "wc " + readme,
			want: func(t *testing.T, out string, code int) {
				if code != 0 || !strings.Contains(out, "lines=2") || !strings.Contains(out, "words=2") || !strings.Contains(out, "bytes=9") {
					t.Fatalf("wc default failed: code=%d out=%q", code, out)
				}
			},
		},
		{
			name: "wc-multi-flags",
			cmd:  "wc -lw " + readme,
			want: func(t *testing.T, out string, code int) {
				if code != 0 || strings.TrimSpace(out) != "lines=2 words=2" {
					t.Fatalf("wc -lw failed: code=%d out=%q", code, out)
				}
			},
		},
		{
			name: "wc-json",
			cmd:  "wc --json " + readme,
			want: func(t *testing.T, out string, code int) {
				if code != 0 || !strings.Contains(out, "\"lines\":2") || !strings.Contains(out, "\"words\":2") || !strings.Contains(out, "\"bytes\":9") {
					t.Fatalf("wc --json failed: code=%d out=%q", code, out)
				}
			},
		},
		{
			name: "wc-stdin-bytes",
			cmd:  "echo hello | wc -c",
			want: func(t *testing.T, out string, code int) {
				if code != 0 || strings.TrimSpace(out) != "5" {
					t.Fatalf("wc -c stdin failed: code=%d out=%q", code, out)
				}
			},
		},
		{
			name: "sort",
			cmd:  "sort -n " + mvTarget,
			want: func(t *testing.T, out string, code int) {
				if code != 0 || strings.TrimSpace(out) == "" {
					t.Fatalf("sort failed: code=%d out=%q", code, out)
				}
			},
		},
		{
			name: "uniq",
			cmd:  "sort " + mvTarget + " | uniq -c",
			want: func(t *testing.T, out string, code int) {
				if code != 0 || !strings.Contains(out, "1") {
					t.Fatalf("uniq failed: code=%d out=%q", code, out)
				}
			},
		},
		{
			name: "diff",
			cmd:  "diff " + readme + " " + readme,
			want: func(t *testing.T, out string, code int) {
				if code != 0 {
					t.Fatalf("diff identical failed: code=%d out=%q", code, out)
				}
			},
		},
		{
			name: "rm",
			cmd:  "rm " + copyTarget,
			want: func(t *testing.T, out string, code int) {
				if code != 0 || strings.TrimSpace(out) != "" {
					t.Fatalf("rm failed: code=%d out=%q", code, out)
				}
			},
		},
		{
			name: "rm-confirm",
			cmd:  "rm --confirm " + rt.abs("workspace", "rm-confirm.txt"),
			want: func(t *testing.T, out string, code int) {
				expected := "removed " + rt.abs("workspace", "rm-confirm.txt")
				if code != 0 || strings.TrimSpace(out) != expected {
					t.Fatalf("rm --confirm failed: code=%d out=%q", code, out)
				}
			},
		},
		{
			name: "rm-json",
			cmd:  "rm --json " + rt.abs("workspace", "rm-json.txt"),
			want: func(t *testing.T, out string, code int) {
				if code != 0 || !strings.Contains(out, "\"status\":\"removed\"") || !strings.Contains(out, "\"path\":\""+rt.abs("workspace", "rm-json.txt")+"\"") {
					t.Fatalf("rm --json failed: code=%d out=%q", code, out)
				}
			},
		},
		{
			name: "rmdir",
			cmd:  "rmdir " + emptyDir,
			want: func(t *testing.T, out string, code int) {
				if code != 0 || strings.TrimSpace(out) != "" {
					t.Fatalf("rmdir failed: code=%d out=%q", code, out)
				}
			},
		},
		{
			name: "rmdir-confirm",
			cmd:  "rmdir --confirm " + rt.abs("workspace", "rmdir-confirm"),
			want: func(t *testing.T, out string, code int) {
				expected := "removed " + rt.abs("workspace", "rmdir-confirm")
				if code != 0 || strings.TrimSpace(out) != expected {
					t.Fatalf("rmdir --confirm failed: code=%d out=%q", code, out)
				}
			},
		},
		{
			name: "rmdir-json",
			cmd:  "rmdir --json " + rt.abs("workspace", "rmdir-json"),
			want: func(t *testing.T, out string, code int) {
				if code != 0 || !strings.Contains(out, "\"status\":\"removed\"") || !strings.Contains(out, "\"path\":\""+rt.abs("workspace", "rmdir-json")+"\"") {
					t.Fatalf("rmdir --json failed: code=%d out=%q", code, out)
				}
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			out, code := rt.exec(tt.cmd)
			tt.want(t, out, code)
		})
	}
}

func TestBuiltinDocsCompleteness(t *testing.T) {
	reg := engine.NewRegistry()
	RegisterDefaults(reg)
	docs := reg.BuiltinCommandDocs()
	if len(docs) == 0 {
		t.Fatalf("no builtin docs registered")
	}
	for _, doc := range docs {
		if strings.TrimSpace(doc.Name) == "" {
			t.Fatalf("doc has empty name: %+v", doc)
		}
		if strings.TrimSpace(doc.Summary) == "" {
			t.Fatalf("doc %q missing summary", doc.Name)
		}
		if strings.TrimSpace(doc.Manual) == "" {
			t.Fatalf("doc %q missing manual synopsis", doc.Name)
		}
		if len(doc.Examples) == 0 {
			t.Fatalf("doc %q missing examples", doc.Name)
		}
		if strings.TrimSpace(doc.DetailedManual) == "" {
			t.Fatalf("doc %q missing detailed manual", doc.Name)
		}
		if strings.TrimSpace(doc.StdinMode) == "" {
			t.Fatalf("doc %q missing stdin mode", doc.Name)
		}
	}
}

func TestBuiltinCommandErrorCoverage(t *testing.T) {
	rt := newTestRuntime(t)
	readOnlyOps := rt.ops
	readOnlyOps.Policy = contract.DefaultPolicy()

	errCases := []struct {
		name string
		cmd  string
		ops  *contract.Ops
	}{
		{name: "ls-flag", cmd: "ls -z"},
		{name: "tree-flag", cmd: "tree -z"},
		{name: "cd-too-many-args", cmd: "cd a b"},
		{name: "pwd-arg", cmd: "pwd extra"},
		{name: "env-too-many-args", cmd: "env A B"},
		{name: "frontmatter-missing-subcommand", cmd: "frontmatter"},
		{name: "cat-missing-arg", cmd: "cat"},
		{name: "head-missing-input", cmd: "head -n 1"},
		{name: "tail-bad-flag", cmd: "tail -x"},
		{name: "grep-missing-pattern", cmd: "grep"},
		{name: "find-bad-expr", cmd: "find -o"},
		{name: "which-missing-operand", cmd: "which"},
		{name: "type-missing-operand", cmd: "type"},
		{name: "echo-ok", cmd: "echo x"}, // sanity command in error matrix
		{name: "tee-missing-stdin", cmd: "tee " + rt.abs("a.txt")},
		{name: "sed-bad-expr", cmd: "sed -i 'bad' " + rt.abs("a.txt")},
		{name: "man-missing-name", cmd: "man"},
		{name: "date-bad-format", cmd: "date +%Q"},
		{name: "frontmatter-bad-flag", cmd: "frontmatter stat -z /"},
		{name: "mkdir-missing-operand", cmd: "mkdir"},
		{name: "cp-missing-args", cmd: "cp " + rt.abs("a.txt")},
		{name: "mv-missing-args", cmd: "mv " + rt.abs("a.txt")},
		{name: "rm-missing-operand", cmd: "rm"},
		{name: "rmdir-missing-operand", cmd: "rmdir"},
		{name: "touch-missing-operand", cmd: "touch"},
		{name: "wc-bad-flag", cmd: "wc -z"},
		{name: "sort-bad-flag", cmd: "sort -z"},
		{name: "uniq-bad-flag", cmd: "uniq -z"},
		{name: "diff-missing-args", cmd: "diff " + rt.abs("a.txt")},
		{name: "tee-read-only", cmd: "echo x | tee " + rt.abs("deny.txt"), ops: &readOnlyOps},
		{name: "rmdir-read-only", cmd: "rmdir " + rt.abs("deny"), ops: &readOnlyOps},
	}

	for _, tc := range errCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			ops := rt.ops
			if tc.ops != nil {
				ops = *tc.ops
			}
			out, code := rt.eng.Execute(context.Background(), tc.cmd, ops)
			if tc.name == "echo-ok" {
				if code != 0 || strings.TrimSpace(out) != "x" {
					t.Fatalf("echo sanity failed: code=%d out=%q", code, out)
				}
				return
			}
			if code == 0 {
				t.Fatalf("expected non-zero for %q, out=%q", tc.cmd, out)
			}
		})
	}
}
