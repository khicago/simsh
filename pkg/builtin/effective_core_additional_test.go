package builtin

import (
	"context"
	"encoding/json"
	"errors"
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/khicago/simsh/pkg/contract"
	"github.com/khicago/simsh/pkg/engine"
)

func TestCurrentWorkingDirFallbackOrder(t *testing.T) {
	tests := []struct {
		name string
		ops  contract.Ops
		want string
	}{
		{
			name: "getter wins",
			ops: contract.Ops{
				GetWorkingDir: func() string { return " /task_outputs " },
				WorkingDir:    "/workspace",
				RootDir:       "/",
			},
			want: "/task_outputs",
		},
		{
			name: "working dir fallback",
			ops: contract.Ops{
				GetWorkingDir: func() string { return " " },
				WorkingDir:    " /workspace ",
				RootDir:       "/",
			},
			want: "/workspace",
		},
		{
			name: "root fallback",
			ops: contract.Ops{
				RootDir: " /root ",
			},
			want: "/root",
		},
		{
			name: "default slash",
			ops:  contract.Ops{},
			want: "/",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := currentWorkingDir(tt.ops); got != tt.want {
				t.Errorf("currentWorkingDir(%+v) = %q, want %q", tt.ops, got, tt.want)
			}
		})
	}
}

func TestPreflightPathChecks(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		runtime := engine.CommandRuntime{
			Ctx: context.Background(),
			Ops: contract.Ops{
				CheckPathOp: func(ctx context.Context, op contract.PathOp, path string) error {
					return nil
				},
			},
		}
		out, code, ok := preflightPathChecks(runtime, "mv", []pathCheck{{path: "/task_outputs/a.txt", op: contract.PathOpWrite}})
		if out != "" || code != 0 || !ok {
			t.Fatalf("preflightPathChecks(success) = (%q, %d, %t), want (\"\", 0, true)", out, code, ok)
		}
	})

	t.Run("unsupported custom message", func(t *testing.T) {
		runtime := engine.CommandRuntime{
			Ctx: context.Background(),
			Ops: contract.Ops{
				CheckPathOp: func(ctx context.Context, op contract.PathOp, path string) error {
					return contract.ErrUnsupported
				},
			},
		}
		out, code, ok := preflightPathChecks(runtime, "mv", []pathCheck{{
			path:               "/sys/bin/new.txt",
			op:                 contract.PathOpWrite,
			unsupportedMessage: "mv: target is immutable",
		}})
		if out != "mv: target is immutable" || code != contract.ExitCodeUnsupported || ok {
			t.Fatalf("preflightPathChecks(unsupported custom) = (%q, %d, %t), want custom unsupported failure", out, code, ok)
		}
	})

	t.Run("general error", func(t *testing.T) {
		runtime := engine.CommandRuntime{
			Ctx: context.Background(),
			Ops: contract.Ops{
				CheckPathOp: func(ctx context.Context, op contract.PathOp, path string) error {
					return errors.New("boom")
				},
			},
		}
		out, code, ok := preflightPathChecks(runtime, "cp", []pathCheck{{path: "/task_outputs/a.txt", op: contract.PathOpRead}})
		if out != "cp: boom" || code != contract.ExitCodeGeneral || ok {
			t.Fatalf("preflightPathChecks(general error) = (%q, %d, %t), want wrapped general failure", out, code, ok)
		}
	})
}

func TestLoadAllManualNamesSortedAndUsable(t *testing.T) {
	names := LoadAllManualNames()
	if len(names) == 0 {
		t.Fatal("LoadAllManualNames() returned no manual names")
	}
	if !slices.IsSorted(names) {
		t.Fatalf("LoadAllManualNames() = %#v, want sorted names", names)
	}
	for _, want := range []string{"ls", "grep", "json", "man"} {
		if !slices.Contains(names, want) {
			t.Fatalf("LoadAllManualNames() missing %q in %#v", want, names)
		}
	}
}

func TestDiffHelpers(t *testing.T) {
	table := lcsTable([]string{"a", "b", "c"}, []string{"a", "x", "c"})
	if got := table[len(table)-1][len(table[0])-1]; got != 2 {
		t.Fatalf("lcsTable(...) final score = %d, want 2", got)
	}

	diff := simpleDiff("/a.txt", "/b.txt", []string{"same", "left", "tail"}, []string{"same", "right", "tail"})
	for _, want := range []string{"--- /a.txt", "+++ /b.txt", "-left", "+right", " tail"} {
		if !strings.Contains(diff, want) {
			t.Fatalf("simpleDiff(...) missing %q in output:\n%s", want, diff)
		}
	}
}

func TestFindExecHelpers(t *testing.T) {
	got := expandFindExecArgs([]string{"echo", "{}", "--", "{}"}, []string{"/a.md", "/b.md"})
	want := []string{"echo", "/a.md", "/b.md", "--", "/a.md", "/b.md"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("expandFindExecArgs(with placeholders) = %#v, want %#v", got, want)
	}

	got = expandFindExecArgs([]string{"echo"}, []string{"/a.md"})
	want = []string{"echo", "/a.md"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("expandFindExecArgs(no placeholders) = %#v, want %#v", got, want)
	}

	out, code := runFindExec(nil, []string{"echo", "{}"}, false, func(args []string, input string, hasInput bool) (string, int) {
		return "should not run", 0
	})
	if out != "" || code != 0 {
		t.Fatalf("runFindExec(no matches) = (%q, %d), want (\"\", 0)", out, code)
	}

	out, code = runFindExec([]string{"/a.md"}, []string{"echo", "{}"}, false, nil)
	if out != "find: command dispatcher is required" || code != contract.ExitCodeGeneral {
		t.Fatalf("runFindExec(nil dispatch) = (%q, %d), want dispatcher error", out, code)
	}

	out, code = runFindExec([]string{"/a.md", "/b.md"}, []string{"echo", "{}"}, true, func(args []string, input string, hasInput bool) (string, int) {
		if !reflect.DeepEqual(args, []string{"echo", "/a.md", "/b.md"}) {
			t.Fatalf("runFindExec(exec+) dispatch args = %#v, want %#v", args, []string{"echo", "/a.md", "/b.md"})
		}
		return strings.Join(args[1:], ","), 0
	})
	if out != "/a.md,/b.md" || code != 0 {
		t.Fatalf("runFindExec(exec+) = (%q, %d), want batched output", out, code)
	}

	out, code = runFindExec([]string{"/a.md"}, []string{"report_tool", "{}"}, false, func(args []string, input string, hasInput bool) (string, int) {
		return "", 17
	})
	if !strings.Contains(out, "find: -exec report_tool failed for /a.md") || code != 17 {
		t.Fatalf("runFindExec(non-plus failure) = (%q, %d), want generated failure message", out, code)
	}
}

func TestFindParseArgsEdgeCases(t *testing.T) {
	requireAbs := func(raw string) (string, error) {
		if !strings.HasPrefix(raw, "/") {
			return "", errors.New("path must be absolute")
		}
		return raw, nil
	}

	tests := []struct {
		name string
		args []string
		want string
	}{
		{name: "unexpected -o", args: []string{"-o"}, want: "find: unexpected -o"},
		{name: "missing pattern", args: []string{"-name"}, want: "find: -name requires pattern"},
		{name: "bad fmt", args: []string{"--fmt", "json"}, want: `find: unsupported --fmt value "json"`},
		{name: "unsupported flag", args: []string{"--bad"}, want: "find: unsupported flag --bad"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, got := parseFindArgs(tt.args, requireAbs, "/")
			if got != tt.want {
				t.Fatalf("parseFindArgs(%#v) = %q, want %q", tt.args, got, tt.want)
			}
		})
	}
}

func TestManHelpers(t *testing.T) {
	if got, ok := parseManListFormat("text"); !ok || got != "text" {
		t.Fatalf("parseManListFormat(text) = (%q, %t), want (text, true)", got, ok)
	}
	if got, ok := parseManListFormat("json"); !ok || got != "json" {
		t.Fatalf("parseManListFormat(json) = (%q, %t), want (json, true)", got, ok)
	}
	if _, ok := parseManListFormat("yaml"); ok {
		t.Fatal("parseManListFormat(yaml) unexpectedly succeeded")
	}

	guided := ensureSummaryGuidance("ls", "list files")
	if !strings.Contains(guided, "Use-When:") || !strings.Contains(guided, "Avoid-When:") {
		t.Fatalf("ensureSummaryGuidance(...) missing guidance sections:\n%s", guided)
	}
	alreadyGuided := "summary\n\nUse-When:\n  - now\nAvoid-When:\n  - later"
	if got := ensureSummaryGuidance("ls", alreadyGuided); got != alreadyGuided {
		t.Fatalf("ensureSummaryGuidance(already guided) = %q, want original text", got)
	}

	runtime := engine.CommandRuntime{
		Ctx: context.Background(),
		ListBuiltinNames: func() []string {
			return []string{"zeta", "alpha"}
		},
		LookupBuiltinDoc: func(name string) (contract.BuiltinCommandDoc, bool) {
			if name == "alpha" {
				return contract.BuiltinCommandDoc{Name: "alpha", Summary: "first"}, true
			}
			return contract.BuiltinCommandDoc{}, false
		},
		Ops: contract.Ops{
			ListExternalCommands: func(ctx context.Context) ([]contract.ExternalCommand, error) {
				return []contract.ExternalCommand{
					{Name: " zzz ", Summary: ""},
					{Name: " aaa ", Summary: "first external"},
				}, nil
			},
		},
	}

	docs := collectBuiltinDocs(runtime)
	if len(docs) != 2 || docs[0].Name != "alpha" || docs[1].Name != "zeta" {
		t.Fatalf("collectBuiltinDocs(...) = %#v, want sorted fallback docs", docs)
	}

	externals := collectExternalCommands(runtime)
	if len(externals) != 2 || strings.TrimSpace(externals[0].Name) != "aaa" || strings.TrimSpace(externals[1].Name) != "zzz" {
		t.Fatalf("collectExternalCommands(...) = %#v, want sorted externals", externals)
	}

	if got := externalSummary(contract.ExternalCommand{Name: "zzz"}); got != "(no description)" {
		t.Fatalf("externalSummary(blank) = %q, want %q", got, "(no description)")
	}
	if got := renderListValue(" "); got != "-" {
		t.Fatalf("renderListValue(blank) = %q, want %q", got, "-")
	}

	listText, code := runManList(runtime, "text")
	if code != 0 {
		t.Fatalf("runManList(text) code = %d, want 0 output=%q", code, listText)
	}
	for _, want := range []string{
		"Builtins:",
		"alpha",
		"zeta",
		"External:",
		"aaa",
		"first external",
		"zzz",
		"(no description)",
	} {
		if !strings.Contains(listText, want) {
			t.Fatalf("runManList(text) missing %q in output:\n%s", want, listText)
		}
	}
}

func TestLSLongFormatHonorsCustomFormatterAndJSONDisplayPath(t *testing.T) {
	runtime := engine.CommandRuntime{
		Ctx: context.Background(),
		Ops: contract.Ops{
			FormatLSLongRow: func(ctx context.Context, row contract.LSLongRow) (string, bool) {
				if row.DisplayPath == "." {
					return "custom-dot-row", true
				}
				return "", false
			},
		},
	}
	rows := []contract.LSLongRow{
		{
			DisplayPath:  ".",
			Path:         "/workspace",
			Exists:       true,
			IsDir:        true,
			Kind:         "dir",
			Access:       contract.PathAccessReadOnly,
			Capabilities: []string{contract.PathCapabilityDescribe},
			LineCount:    -1,
		},
		{
			DisplayPath:  "note.md",
			Path:         "/workspace/note.md",
			Exists:       true,
			IsDir:        false,
			Kind:         "file",
			Access:       contract.PathAccessReadOnly,
			Capabilities: []string{contract.PathCapabilityDescribe, contract.PathCapabilityRead},
			LineCount:    2,
		},
	}

	text := formatLongRows(runtime, rows, lsLongFormatText)
	if !strings.Contains(text, "custom-dot-row") || !strings.Contains(text, "- ro file 2 note.md") {
		t.Fatalf("formatLongRows(text) = %q, want custom formatter + fallback row", text)
	}

	raw := formatJSONLongRows(rows)
	var payload struct {
		Columns []string `json:"columns"`
		Entries []struct {
			DisplayPath string `json:"display_path"`
			Path        string `json:"path"`
		} `json:"entries"`
	}
	if err := json.Unmarshal([]byte(raw), &payload); err != nil {
		t.Fatalf("formatJSONLongRows(...) unmarshal error = %v raw=%q", err, raw)
	}
	if len(payload.Entries) != 2 || payload.Entries[0].DisplayPath != "." || payload.Entries[0].Path != "/workspace" {
		t.Fatalf("formatJSONLongRows(...) = %#v, want display_path + absolute path", payload)
	}
}

func TestJSONFormattingHelpersAndTargetExpansion(t *testing.T) {
	if got := compactJSONString(map[string]any{"b": 2}); got != `{"b":2}` {
		t.Fatalf("compactJSONString(...) = %q, want compact json", got)
	}
	if got := prettyJSONString(map[string]any{"b": 2}); !strings.Contains(got, "\n  \"b\": 2\n") {
		t.Fatalf("prettyJSONString(...) = %q, want indented json", got)
	}
	if got := normalizeJSONBytes(" \n {\"x\":1}\n "); got != `{"x":1}` {
		t.Fatalf("normalizeJSONBytes(...) = %q, want trimmed json", got)
	}

	rows := []jsonStatRow{{Path: "/workspace/data.json", Valid: true, Kind: "object", Size: 2, Keys: []string{"a", "b"}}}
	jsonOut := renderJSONStat(rows, jsonStatFormatJSON)
	if !strings.Contains(jsonOut, `"columns":["valid","kind","size","keys","path"]`) {
		t.Fatalf("renderJSONStat(json) = %q, want columns payload", jsonOut)
	}
	mdOut := renderJSONStat(rows, jsonStatFormatMD)
	if !strings.Contains(mdOut, "| valid | kind | size | keys | path |") {
		t.Fatalf("renderJSONStat(md) = %q, want markdown table", mdOut)
	}
	compactOut := renderJSONStat(rows, jsonStatFormatCompact)
	if !strings.Contains(compactOut, "# columns: valid kind size keys path") {
		t.Fatalf("renderJSONStat(compact) = %q, want compact legend", compactOut)
	}

	runtime := engine.CommandRuntime{
		Ctx: context.Background(),
		Ops: contract.Ops{
			IsDirPath: func(ctx context.Context, path string) (bool, error) {
				return path == "/workspace" || path == "/workspace/subdir" || path == "/empty", nil
			},
			ListChildren: func(ctx context.Context, dir string) ([]string, error) {
				if dir == "/empty" {
					return nil, nil
				}
				return []string{"/workspace/a.json", "/workspace/subdir"}, nil
			},
			CollectFilesUnder: func(ctx context.Context, target string) ([]string, error) {
				if target == "/empty" {
					return nil, nil
				}
				return []string{"/workspace/a.json", "/workspace/b.json"}, nil
			},
		},
	}
	files, out, code := expandJSONTargets(runtime, "json stat", []string{"/workspace"}, false)
	if code != 0 || out != "" || !reflect.DeepEqual(files, []string{"/workspace/a.json"}) {
		t.Fatalf("expandJSONTargets(non-recursive dir) = (%#v, %q, %d), want one immediate file", files, out, code)
	}
	files, out, code = expandJSONTargets(runtime, "json stat", []string{"/workspace"}, true)
	if code != 0 || out != "" || !reflect.DeepEqual(files, []string{"/workspace/a.json", "/workspace/b.json"}) {
		t.Fatalf("expandJSONTargets(recursive dir) = (%#v, %q, %d), want collected files", files, out, code)
	}
	files, out, code = expandJSONTargets(runtime, "json stat", []string{"/empty"}, true)
	if code == 0 || !strings.Contains(out, "no files found") || files != nil {
		t.Fatalf("expandJSONTargets(no files) = (%#v, %q, %d), want no-files error", files, out, code)
	}
}
