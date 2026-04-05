package builtin

import (
	"context"
	"strings"
	"testing"

	"github.com/khicago/simsh/pkg/contract"
	"github.com/khicago/simsh/pkg/engine"
)

func TestAppendTreeChildrenASCIIUsesEntryMetadata(t *testing.T) {
	runtime := engine.CommandRuntime{
		Ctx: context.Background(),
		Ops: contract.Ops{
			ListEntries: func(ctx context.Context, req contract.ListEntriesRequest) (contract.ListEntriesResult, error) {
				switch req.Dir {
				case "/docs":
					return contract.ListEntriesResult{Entries: []contract.MountEntry{
						{Path: "/docs/guide.md", Meta: contract.PathMeta{Exists: true, IsDir: false}},
						{Path: "/docs/nested", Meta: contract.PathMeta{Exists: true, IsDir: true}},
					}}, nil
				case "/docs/nested":
					return contract.ListEntriesResult{Entries: []contract.MountEntry{
						{Path: "/docs/nested/more.md", Meta: contract.PathMeta{Exists: true, IsDir: false}},
					}}, nil
				default:
					return contract.ListEntriesResult{}, contract.ErrUnsupported
				}
			},
			IsDirPath: func(ctx context.Context, path string) (bool, error) {
				t.Fatal("tree ascii should not re-stat children when entry metadata already exists")
				return false, nil
			},
			ListChildren: func(ctx context.Context, dir string) ([]string, error) {
				t.Fatal("tree ascii should not fall back to ListChildren when ListEntries succeeds")
				return nil, nil
			},
		},
	}

	lines := []string{"/docs"}
	visited := map[string]struct{}{"/docs": {}}
	if err := appendTreeChildrenASCII(runtime, "/docs", "", 0, -1, false, visited, &lines); err != nil {
		t.Fatalf("appendTreeChildrenASCII(...) error = %v", err)
	}
	out := strings.Join(lines, "\n")
	if !strings.Contains(out, "|-- guide.md") || !strings.Contains(out, "`-- nested/") || !strings.Contains(out, "more.md") {
		t.Fatalf("appendTreeChildrenASCII(...) output = %q", out)
	}
}
