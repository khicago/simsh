package builtin

import (
	"context"
	"strings"
	"testing"

	"github.com/khicago/simsh/pkg/contract"
	"github.com/khicago/simsh/pkg/engine"
)

func TestListDirectoryEntriesUsesListEntries(t *testing.T) {
	ctx := context.Background()
	called := false
	runtime := engine.CommandRuntime{
		Ctx: ctx,
		Ops: contract.Ops{
			ListEntries: func(ctx context.Context, req contract.ListEntriesRequest) (contract.ListEntriesResult, error) {
				called = true
				return contract.ListEntriesResult{
					Entries: []contract.MountEntry{
						{Path: "/docs/file", Meta: contract.PathMeta{Exists: true, IsDir: false}},
					},
				}, nil
			},
		},
	}

	entries, err := listDirectoryEntries(runtime, "/docs", false)
	if err != nil {
		t.Fatalf("listDirectoryEntries error = %v, want nil", err)
	}
	if !called {
		t.Fatalf("listDirectoryEntries did not use ListEntries")
	}
	if len(entries) != 1 || entries[0].Path != "/docs/file" {
		t.Fatalf("listDirectoryEntries returned %v, want /docs/file", entries)
	}
}

func TestListDirectoryEntriesHitsFallback(t *testing.T) {
	ctx := context.Background()
	fallbackChildren := []string{"/docs/a", "/docs/b"}
	listChildrenCalled := false
	describeCalled := 0
	runtime := engine.CommandRuntime{
		Ctx: ctx,
		Ops: contract.Ops{
			ListChildren: func(ctx context.Context, dir string) ([]string, error) {
				listChildrenCalled = true
				return fallbackChildren, nil
			},
			DescribePath: func(ctx context.Context, path string) (contract.PathMeta, error) {
				describeCalled++
				return contract.PathMeta{
					Exists: true,
					IsDir:  strings.HasSuffix(path, "/b"),
					Access: contract.PathAccessReadOnly,
				}, nil
			},
		},
	}

	entries, err := listDirectoryEntries(runtime, "/docs", false)
	if err != nil {
		t.Fatalf("listDirectoryEntries error = %v", err)
	}
	if !listChildrenCalled {
		t.Fatalf("fallback path did not call ListChildren")
	}
	if describeCalled != len(fallbackChildren) {
		t.Fatalf("DescribePath called %d times, want %d", describeCalled, len(fallbackChildren))
	}
	if len(entries) != len(fallbackChildren) {
		t.Fatalf("expected %d entries, got %d", len(fallbackChildren), len(entries))
	}
}

func TestListDirectoryEntriesPropagatesNonUnsupportedErrors(t *testing.T) {
	runtime := engine.CommandRuntime{
		Ctx: context.Background(),
		Ops: contract.Ops{
			ListEntries: func(ctx context.Context, req contract.ListEntriesRequest) (contract.ListEntriesResult, error) {
				return contract.ListEntriesResult{}, context.DeadlineExceeded
			},
			ListChildren: func(ctx context.Context, dir string) ([]string, error) {
				t.Fatal("unexpected fallback to ListChildren")
				return nil, nil
			},
		},
	}

	if _, err := listDirectoryEntries(runtime, "/docs", false); err != context.DeadlineExceeded {
		t.Fatalf("listDirectoryEntries error = %v, want %v", err, context.DeadlineExceeded)
	}
}

func TestListDirectoryEntriesDoesNotFallbackWhenRemoteHighUnsupported(t *testing.T) {
	runtime := engine.CommandRuntime{
		Ctx: context.Background(),
		Ops: contract.Ops{
			ListEntries: func(ctx context.Context, req contract.ListEntriesRequest) (contract.ListEntriesResult, error) {
				return contract.ListEntriesResult{}, &contract.MountUnsupportedError{
					MountPoint:   "/remote",
					Capability:   "entry listing",
					LatencyClass: contract.MountLatencyRemoteHigh,
					Detail:       "/remote: entry listing requires EntryLister for remote_high_latency mount",
				}
			},
			ListChildren: func(ctx context.Context, dir string) ([]string, error) {
				t.Fatal("listDirectoryEntries should not fall back to ListChildren for remote_high_latency refusal")
				return nil, nil
			},
		},
	}

	err := func() error {
		_, err := listDirectoryEntries(runtime, "/remote", false)
		return err
	}()
	if err == nil || !strings.Contains(err.Error(), "remote_high_latency") {
		t.Fatalf("listDirectoryEntries remote_high_latency error = %v, want explicit refusal", err)
	}
}
