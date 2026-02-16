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
	copyTarget := rt.abs("workspace", "copy.md")
	mvTarget := rt.abs("workspace", "moved.txt")
	newFile := rt.abs("workspace", "new.txt")

	if out, code := rt.exec("mkdir -p " + rt.abs("workspace")); code != 0 {
		t.Fatalf("setup mkdir failed: code=%d out=%q", code, out)
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
			name: "env",
			cmd:  "env PATH",
			want: func(t *testing.T, out string, code int) {
				if code != 0 || !strings.Contains(out, "/sys/bin:/bin") {
					t.Fatalf("env failed: code=%d out=%q", code, out)
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
			name: "find",
			cmd:  "find " + rt.abs("workspace") + " -name \"*.md\"",
			want: func(t *testing.T, out string, code int) {
				if code != 0 || !strings.Contains(out, readme) {
					t.Fatalf("find failed: code=%d out=%q", code, out)
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
			name: "sed",
			cmd:  "sed -i 's/hello/hi/' " + readme + "; cat " + readme,
			want: func(t *testing.T, out string, code int) {
				if code != 0 || !strings.Contains(out, "hi") {
					t.Fatalf("sed failed: code=%d out=%q", code, out)
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
				if code != 0 {
					t.Fatalf("mkdir failed: code=%d out=%q", code, out)
				}
			},
		},
		{
			name: "cp",
			cmd:  "cp " + readme + " " + copyTarget,
			want: func(t *testing.T, out string, code int) {
				if code != 0 {
					t.Fatalf("cp failed: code=%d out=%q", code, out)
				}
			},
		},
		{
			name: "mv",
			cmd:  "mv " + logs + " " + mvTarget,
			want: func(t *testing.T, out string, code int) {
				if code != 0 {
					t.Fatalf("mv failed: code=%d out=%q", code, out)
				}
			},
		},
		{
			name: "touch",
			cmd:  "touch " + newFile,
			want: func(t *testing.T, out string, code int) {
				if code != 0 {
					t.Fatalf("touch failed: code=%d out=%q", code, out)
				}
			},
		},
		{
			name: "wc",
			cmd:  "wc -l " + readme,
			want: func(t *testing.T, out string, code int) {
				if code != 0 || strings.TrimSpace(out) == "" {
					t.Fatalf("wc failed: code=%d out=%q", code, out)
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
				if code != 0 {
					t.Fatalf("rm failed: code=%d out=%q", code, out)
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
		if strings.TrimSpace(doc.Manual) == "" {
			t.Fatalf("doc %q missing manual synopsis", doc.Name)
		}
		if len(doc.Examples) == 0 {
			t.Fatalf("doc %q missing examples", doc.Name)
		}
		if strings.TrimSpace(doc.DetailedManual) == "" {
			t.Fatalf("doc %q missing detailed manual", doc.Name)
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
		{name: "env-too-many-args", cmd: "env A B"},
		{name: "cat-missing-arg", cmd: "cat"},
		{name: "head-missing-input", cmd: "head -n 1"},
		{name: "tail-bad-flag", cmd: "tail -x"},
		{name: "grep-missing-pattern", cmd: "grep"},
		{name: "find-bad-expr", cmd: "find -o"},
		{name: "echo-ok", cmd: "echo x"}, // sanity command in error matrix
		{name: "tee-missing-stdin", cmd: "tee " + rt.abs("a.txt")},
		{name: "sed-bad-expr", cmd: "sed -i 'bad' " + rt.abs("a.txt")},
		{name: "man-missing-name", cmd: "man"},
		{name: "date-bad-format", cmd: "date +%Q"},
		{name: "mkdir-missing-operand", cmd: "mkdir"},
		{name: "cp-missing-args", cmd: "cp " + rt.abs("a.txt")},
		{name: "mv-missing-args", cmd: "mv " + rt.abs("a.txt")},
		{name: "rm-missing-operand", cmd: "rm"},
		{name: "touch-missing-operand", cmd: "touch"},
		{name: "wc-bad-flag", cmd: "wc -z"},
		{name: "sort-bad-flag", cmd: "sort -z"},
		{name: "uniq-bad-flag", cmd: "uniq -z"},
		{name: "diff-missing-args", cmd: "diff " + rt.abs("a.txt")},
		{name: "tee-read-only", cmd: "echo x | tee " + rt.abs("deny.txt"), ops: &readOnlyOps},
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
