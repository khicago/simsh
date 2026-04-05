package builtin

import (
	"context"
	"strings"
	"testing"

	"github.com/khicago/simsh/pkg/contract"
	"github.com/khicago/simsh/pkg/engine"
)

func TestRunGrepUsesRuntimeSearch(t *testing.T) {
	runtime := engine.CommandRuntime{
		Ctx: context.Background(),
		Ops: contract.Ops{
			RequireAbsolutePath: func(raw string) (string, error) { return raw, nil },
			IsDirPath: func(ctx context.Context, path string) (bool, error) {
				return path == "/docs", nil
			},
			SearchContent: func(ctx context.Context, req contract.SearchRequest) (contract.SearchResult, error) {
				if req.Pattern != "hello" || !req.Regex || !req.ListFiles || len(req.Targets) != 1 || req.Targets[0] != "/docs" || req.Before != 1 || req.After != 1 {
					t.Fatalf("unexpected SearchContent request: %+v", req)
				}
				return contract.SearchResult{
					Records: []contract.SearchRecord{
						{Path: "/docs/a.txt", Kind: "file"},
					},
				}, nil
			},
			ResolveSearchPaths: func(ctx context.Context, target string, recursive bool) ([]string, error) {
				t.Fatal("grep should not fall back to ResolveSearchPaths when SearchContent succeeds")
				return nil, nil
			},
			ReadRawContent: func(ctx context.Context, path string) (string, error) {
				t.Fatal("grep should not fall back to ReadRawContent when SearchContent succeeds")
				return "", nil
			},
		},
	}

	out, code := runGrep(runtime, []string{"-r", "-l", "-A", "1", "-B", "1", "-E", "hello", "/docs"})
	if code != 0 || strings.TrimSpace(out) != "/docs/a.txt" {
		t.Fatalf("runGrep(...) = (%q, %d), want matched path via runtime search", out, code)
	}
}

func TestRunRGUsesRuntimeSearch(t *testing.T) {
	runtime := engine.CommandRuntime{
		Ctx: context.Background(),
		Ops: contract.Ops{
			RequireAbsolutePath: func(raw string) (string, error) { return raw, nil },
			SearchContent: func(ctx context.Context, req contract.SearchRequest) (contract.SearchResult, error) {
				if req.Pattern != "hello" || req.Regex || req.CaseMode != contract.SearchCaseIgnore || len(req.Targets) != 1 || req.Targets[0] != "/docs" || len(req.Globs) != 1 || req.Globs[0] != "*.md" || !req.ListFiles {
					t.Fatalf("unexpected SearchContent request: %+v", req)
				}
				return contract.SearchResult{
					Records: []contract.SearchRecord{
						{Path: "/docs/guide.md", Kind: "file"},
					},
				}, nil
			},
			ResolveSearchPaths: func(ctx context.Context, target string, recursive bool) ([]string, error) {
				t.Fatal("rg should not fall back to ResolveSearchPaths when SearchContent succeeds")
				return nil, nil
			},
			ReadRawContent: func(ctx context.Context, path string) (string, error) {
				t.Fatal("rg should not fall back to ReadRawContent when SearchContent succeeds")
				return "", nil
			},
		},
	}

	out, code := runRG(runtime, []string{"-l", "--fmt", "jsonl", "-g", "*.md", "-i", "-F", "hello", "/docs"})
	if code != 0 || !strings.Contains(out, `"path":"/docs/guide.md"`) || !strings.Contains(out, `"name":"guide.md"`) || !strings.Contains(out, `"kind":"file"`) {
		t.Fatalf("runRG(...) = (%q, %d), want file jsonl via runtime search", out, code)
	}
}

func TestRunRGFallsBackWhenRuntimeSearchUnsupported(t *testing.T) {
	resolveCalled := false
	readCalled := false
	runtime := engine.CommandRuntime{
		Ctx: context.Background(),
		Ops: contract.Ops{
			RequireAbsolutePath: func(raw string) (string, error) { return raw, nil },
			SearchContent: func(ctx context.Context, req contract.SearchRequest) (contract.SearchResult, error) {
				return contract.SearchResult{}, contract.ErrUnsupported
			},
			ResolveSearchPaths: func(ctx context.Context, target string, recursive bool) ([]string, error) {
				resolveCalled = true
				return []string{"/docs/guide.md"}, nil
			},
			ReadRawContent: func(ctx context.Context, path string) (string, error) {
				readCalled = true
				return "hello\n", nil
			},
		},
	}

	out, code := runRG(runtime, []string{"hello", "/docs"})
	if code != 0 || !strings.Contains(out, "/docs/guide.md:1:hello") {
		t.Fatalf("runRG fallback = (%q, %d), want text search output", out, code)
	}
	if !resolveCalled || !readCalled {
		t.Fatalf("expected fallback path, got resolve=%v read=%v", resolveCalled, readCalled)
	}
}
