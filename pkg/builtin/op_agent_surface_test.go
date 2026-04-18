package builtin

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"
)

func TestEditReplacesUniqueSnippet(t *testing.T) {
	rt := newTestRuntime(t)
	path := rt.abs("workspace", "note.txt")
	if out, code := rt.exec("mkdir -p " + rt.abs("workspace")); code != 0 {
		t.Fatalf("mkdir failed: code=%d out=%q", code, out)
	}
	if out, code := rt.exec("echo hello unique world | tee " + path); code != 0 {
		t.Fatalf("tee failed: code=%d out=%q", code, out)
	}

	out, code := rt.exec("edit --json --old unique --new patched " + path)
	if code != 0 {
		t.Fatalf("edit unique failed: code=%d out=%q", code, out)
	}
	if !strings.Contains(out, `"replaced":1`) || !strings.Contains(out, `"matches":1`) {
		t.Fatalf("edit json = %q, want matches/replaced 1", out)
	}

	got, code := rt.exec("cat " + path)
	if code != 0 || !strings.Contains(got, "hello patched world") {
		t.Fatalf("cat after edit = (%q, %d)", got, code)
	}
}

func TestEditRefusesAmbiguousSnippetWithoutAll(t *testing.T) {
	rt := newTestRuntime(t)
	path := rt.abs("workspace", "dup.txt")
	if out, code := rt.exec("mkdir -p " + rt.abs("workspace")); code != 0 {
		t.Fatalf("mkdir failed: code=%d out=%q", code, out)
	}
	if out, code := rt.exec("echo foo foo | tee " + path); code != 0 {
		t.Fatalf("tee failed: code=%d out=%q", code, out)
	}

	out, code := rt.exec("edit --old foo --new bar " + path)
	if code == 0 || !strings.Contains(out, "appears") || !strings.Contains(out, "--all") || !strings.Contains(out, "lines") {
		t.Fatalf("edit ambiguous = (%q, %d), want unique-match refusal with lines", out, code)
	}
	got, _ := rt.exec("cat " + path)
	if !strings.Contains(got, "foo foo") {
		t.Fatalf("ambiguous edit mutated file: %q", got)
	}

	out, code = rt.exec("edit --all --old foo --new bar " + path)
	if code != 0 {
		t.Fatalf("edit --all failed: code=%d out=%q", code, out)
	}
	got, code = rt.exec("cat " + path)
	if code != 0 || strings.Contains(got, "foo") || !strings.Contains(got, "bar bar") {
		t.Fatalf("cat after --all = (%q, %d)", got, code)
	}
}

func TestEditCountDoesNotMutate(t *testing.T) {
	rt := newTestRuntime(t)
	path := rt.abs("workspace", "count.txt")
	if out, code := rt.exec("mkdir -p " + rt.abs("workspace")); code != 0 {
		t.Fatalf("mkdir failed: code=%d out=%q", code, out)
	}
	if out, code := rt.exec("echo aa aa aa | tee " + path); code != 0 {
		t.Fatalf("tee failed: code=%d out=%q", code, out)
	}

	out, code := rt.exec("edit --count --old aa " + path)
	if code != 0 || !strings.HasPrefix(strings.TrimSpace(out), "3") || !strings.Contains(out, "1:") {
		t.Fatalf("edit --count = (%q, %d), want count 3 with line numbers", out, code)
	}
	got, _ := rt.exec("cat " + path)
	if !strings.Contains(got, "aa aa aa") {
		t.Fatalf("count mutated file: %q", got)
	}
}

func TestGlobListsRecursiveBasenameMatches(t *testing.T) {
	rt := newTestRuntime(t)
	root := rt.abs("workspace")
	if out, code := rt.exec("mkdir -p " + rt.abs("workspace", "nested")); code != 0 {
		t.Fatalf("mkdir failed: code=%d out=%q", code, out)
	}
	if out, code := rt.exec("echo one | tee " + rt.abs("workspace", "keep.go")); code != 0 {
		t.Fatalf("tee go failed: code=%d out=%q", code, out)
	}
	if out, code := rt.exec("echo two | tee " + rt.abs("workspace", "nested", "deep.go")); code != 0 {
		t.Fatalf("tee nested go failed: code=%d out=%q", code, out)
	}
	if out, code := rt.exec("echo skip | tee " + rt.abs("workspace", "notes.md")); code != 0 {
		t.Fatalf("tee md failed: code=%d out=%q", code, out)
	}

	out, code := rt.exec("glob '*.go' " + root)
	if code != 0 {
		t.Fatalf("glob failed: code=%d out=%q", code, out)
	}
	if !strings.Contains(out, "keep.go") || !strings.Contains(out, "deep.go") || strings.Contains(out, "notes.md") {
		t.Fatalf("glob output = %q, want recursive go files only", out)
	}

	out, code = rt.exec("glob --fmt jsonl '*.md' " + root)
	if code != 0 {
		t.Fatalf("glob jsonl failed: code=%d out=%q", code, out)
	}
	var rec struct {
		Path string `json:"path"`
		Name string `json:"name"`
	}
	if err := json.Unmarshal([]byte(strings.TrimSpace(out)), &rec); err != nil {
		t.Fatalf("glob jsonl parse %q: %v", out, err)
	}
	if rec.Name != "notes.md" {
		t.Fatalf("glob jsonl name = %q, want notes.md", rec.Name)
	}
}

func TestViewPrintsNumberedWindow(t *testing.T) {
	rt := newTestRuntime(t)
	path := rt.abs("workspace", "lines.txt")
	if out, code := rt.exec("mkdir -p " + rt.abs("workspace")); code != 0 {
		t.Fatalf("mkdir failed: code=%d out=%q", code, out)
	}
	if err := rt.ops.WriteFile(context.Background(), path, "a\nb\nc\nd\n"); err != nil {
		t.Fatalf("write lines failed: %v", err)
	}

	out, code := rt.exec("view --start 2 --lines 2 " + path)
	if code != 0 {
		t.Fatalf("view failed: code=%d out=%q", code, out)
	}
	if !strings.Contains(out, "2:") || !strings.Contains(out, "3:") || strings.Contains(out, "1:") || strings.Contains(out, "4:") {
		t.Fatalf("view window = %q, want lines 2-3 numbered", out)
	}
	if !strings.Contains(out, "2:b") || !strings.Contains(out, "3:c") {
		t.Fatalf("view window = %q, want 2:b and 3:c", out)
	}
	if !strings.Contains(out, "shown 2/4 from 2") {
		t.Fatalf("view window = %q, want shown/total footer", out)
	}
}

func TestViewPastEOFReportsTotal(t *testing.T) {
	rt := newTestRuntime(t)
	path := rt.abs("workspace", "short.txt")
	if out, code := rt.exec("mkdir -p " + rt.abs("workspace")); code != 0 {
		t.Fatalf("mkdir failed: code=%d out=%q", code, out)
	}
	if err := rt.ops.WriteFile(context.Background(), path, "a\nb\n"); err != nil {
		t.Fatalf("write failed: %v", err)
	}
	out, code := rt.exec("view --start 200 --lines 10 " + path)
	if code == 0 || !strings.Contains(out, "file has 2 lines") {
		t.Fatalf("view past EOF = (%q, %d), want file has 2 lines", out, code)
	}
	out, code = rt.exec("view --json " + path)
	if code == 0 || !strings.Contains(out, "--fmt jsonl") {
		t.Fatalf("view --json = (%q, %d), want --fmt jsonl hint", out, code)
	}
}

func TestGlobMissingRootErrors(t *testing.T) {
	rt := newTestRuntime(t)
	missing := rt.abs("workspace", "nope-dir")
	out, code := rt.exec("glob '*' " + missing)
	if code == 0 || strings.Contains(out, `"kind":"file"`) || !strings.Contains(out, "No such file") {
		t.Fatalf("glob missing = (%q, %d), want missing-root error not ghost file", out, code)
	}
	if strings.Contains(out, "open ") {
		t.Fatalf("glob missing leaked host open() error: %q", out)
	}
}

func TestEditEmptyNewDeletesSnippet(t *testing.T) {
	rt := newTestRuntime(t)
	path := rt.abs("workspace", "del.txt")
	if out, code := rt.exec("mkdir -p " + rt.abs("workspace")); code != 0 {
		t.Fatalf("mkdir failed: code=%d out=%q", code, out)
	}
	if out, code := rt.exec("echo keep MARK gone | tee " + path); code != 0 {
		t.Fatalf("tee failed: code=%d out=%q", code, out)
	}
	out, code := rt.exec("edit --json --old MARK --new '' " + path)
	if code != 0 {
		t.Fatalf("edit empty new failed: code=%d out=%q", code, out)
	}
	got, _ := rt.exec("cat " + path)
	if strings.Contains(got, "MARK") || !strings.Contains(got, "keep") {
		t.Fatalf("cat after empty new = %q", got)
	}
}

func TestEditOldFlagWithoutValue(t *testing.T) {
	rt := newTestRuntime(t)
	path := rt.abs("workspace", "x.txt")
	if out, code := rt.exec("mkdir -p " + rt.abs("workspace")); code != 0 {
		t.Fatalf("mkdir failed: code=%d out=%q", code, out)
	}
	if out, code := rt.exec("echo x | tee " + path); code != 0 {
		t.Fatalf("tee failed: code=%d out=%q", code, out)
	}
	out, code := rt.exec("edit --old --new DONE " + path)
	if code == 0 || !strings.Contains(out, "--old requires a value") {
		t.Fatalf("edit --old --new = (%q, %d), want missing --old value", out, code)
	}
}

func TestEditAcceptsFlagLikeSnippetsWithEquals(t *testing.T) {
	rt := newTestRuntime(t)
	path := rt.abs("workspace", "flags.txt")
	if out, code := rt.exec("mkdir -p " + rt.abs("workspace")); code != 0 {
		t.Fatalf("mkdir failed: code=%d out=%q", code, out)
	}
	if out, code := rt.exec("echo --json | tee " + path); code != 0 {
		t.Fatalf("tee failed: code=%d out=%q", code, out)
	}

	out, code := rt.exec("edit --old=--json --new=--done " + path)
	if code != 0 {
		t.Fatalf("edit flag-like snippets failed: code=%d out=%q", code, out)
	}
	got, code := rt.exec("cat " + path)
	if code != 0 || !strings.Contains(got, "--done") || strings.Contains(got, "--json") {
		t.Fatalf("cat after flag-like edit = (%q, %d)", got, code)
	}
}

func TestDirnameUnknownFlag(t *testing.T) {
	rt := newTestRuntime(t)
	out, code := rt.exec("dirname --json /temp_work/x")
	if code == 0 || !strings.Contains(out, "unsupported flag") {
		t.Fatalf("dirname --json = (%q, %d), want unsupported flag", out, code)
	}
}

func TestEditMissingFileUsesVirtualPath(t *testing.T) {
	rt := newTestRuntime(t)
	missing := rt.abs("workspace", "missing.txt")
	if out, code := rt.exec("mkdir -p " + rt.abs("workspace")); code != 0 {
		t.Fatalf("mkdir failed: code=%d out=%q", code, out)
	}
	out, code := rt.exec("edit --old a --new b " + missing)
	if code == 0 || !strings.Contains(out, missing) || !strings.Contains(out, "No such file") {
		t.Fatalf("edit missing = (%q, %d), want virtual missing path", out, code)
	}
	if strings.Contains(out, "open ") {
		t.Fatalf("edit missing leaked host open() error: %q", out)
	}
}

func TestMatchGlobPattern(t *testing.T) {
	cases := []struct {
		rel, pattern string
		want         bool
	}{
		{"keep.go", "*.go", true},
		{"nested/deep.go", "*.go", true},
		{"notes.md", "*.go", false},
		{"pkg/builtin/op.go", "pkg/*.go", false},
		{"pkg/op.go", "pkg/*.go", true},
		{"pkg/builtin/op.go", "pkg/**/*.go", true},
	}
	for _, tc := range cases {
		got, err := matchGlobPattern(tc.rel, tc.pattern)
		if err != nil {
			t.Fatalf("matchGlobPattern(%q, %q) err = %v", tc.rel, tc.pattern, err)
		}
		if got != tc.want {
			t.Errorf("matchGlobPattern(%q, %q) = %v, want %v", tc.rel, tc.pattern, got, tc.want)
		}
	}
}

func TestGlobRejectsInvalidPatternBeforeScanningEmptyDirectory(t *testing.T) {
	rt := newTestRuntime(t)
	root := rt.abs("workspace", "empty")
	if out, code := rt.exec("mkdir -p " + root); code != 0 {
		t.Fatalf("mkdir failed: code=%d out=%q", code, out)
	}

	out, code := rt.exec("glob '[' " + root)
	if code == 0 || !strings.Contains(out, "invalid pattern") {
		t.Fatalf("glob invalid pattern = (%q, %d), want syntax error", out, code)
	}
}

func TestCompiledGlobPatternBoundsDoubleStarBacktracking(t *testing.T) {
	pattern := strings.Repeat("**/x/", 24) + "missing"
	rel := strings.Repeat("x/", 24) + "file"
	compiled, err := compileGlobPattern(pattern)
	if err != nil {
		t.Fatalf("compileGlobPattern(%q) error = %v", pattern, err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	matched, err := compiled.match(ctx, rel)
	if err != nil {
		t.Fatalf("compiled glob match exceeded bounded work: %v", err)
	}
	if matched {
		t.Fatalf("compiled glob unexpectedly matched %q", rel)
	}

	canceled, stop := context.WithCancel(context.Background())
	stop()
	if _, err := compiled.match(canceled, rel); !errors.Is(err, context.Canceled) {
		t.Fatalf("compiled glob canceled error = %v, want context.Canceled", err)
	}
}

func TestDirnameAndBasename(t *testing.T) {
	rt := newTestRuntime(t)
	path := rt.abs("workspace", "nested", "file.txt")

	out, code := rt.exec("dirname " + path)
	if code != 0 || strings.TrimSpace(out) != rt.abs("workspace", "nested") {
		t.Fatalf("dirname = (%q, %d), want %q", out, code, rt.abs("workspace", "nested"))
	}
	out, code = rt.exec("basename " + path)
	if code != 0 || strings.TrimSpace(out) != "file.txt" {
		t.Fatalf("basename = (%q, %d), want file.txt", out, code)
	}
}

func TestBasenameEndOfOptionsAllowsDashPrefixedPath(t *testing.T) {
	rt := newTestRuntime(t)
	out, code := rt.exec("basename -- -report.md")
	if code != 0 || strings.TrimSpace(out) != "-report.md" {
		t.Fatalf("basename -- -report.md = (%q, %d)", out, code)
	}
}
