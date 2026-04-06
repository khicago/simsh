package builtin

import (
	"context"
	"strings"
	"testing"

	"github.com/khicago/simsh/pkg/contract"
	"github.com/khicago/simsh/pkg/engine"
)

func TestRunJSONGetRejectsDuplicatePaths(t *testing.T) {
	runtime := engine.CommandRuntime{
		Ctx: context.Background(),
		Ops: contract.Ops{
			RequireAbsolutePath: func(raw string) (string, error) { return raw, nil },
		},
	}

	out, code := runJSONGet(runtime, []string{"--path", "meta.author", "--path", "meta.author", "/tmp/data.json"})
	if code != contract.ExitCodeUsage || !strings.Contains(out, "duplicate --path") {
		t.Fatalf("runJSONGet duplicate path = (%q, %d), want duplicate path usage error", out, code)
	}
}

func TestRunJSONGetRejectsRawWithJSONL(t *testing.T) {
	runtime := engine.CommandRuntime{
		Ctx: context.Background(),
		Ops: contract.Ops{
			RequireAbsolutePath: func(raw string) (string, error) { return raw, nil },
		},
	}

	out, code := runJSONGet(runtime, []string{"--raw", "--fmt", "jsonl", "--path", "meta.author", "/tmp/data.json"})
	if code != contract.ExitCodeUsage || !strings.Contains(out, "--raw is not supported with --fmt jsonl") {
		t.Fatalf("runJSONGet raw/jsonl = (%q, %d), want usage error", out, code)
	}
}

func TestRunJSONKeysReportsExplicitErrorRows(t *testing.T) {
	runtime := engine.CommandRuntime{
		Ctx: context.Background(),
		Ops: contract.Ops{
			RequireAbsolutePath: func(raw string) (string, error) { return raw, nil },
			IsDirPath:           func(context.Context, string) (bool, error) { return false, nil },
			ReadRawContent: func(context.Context, string) (string, error) {
				return `{"items":[{"name":"first"}]}`, nil
			},
		},
	}

	out, code := runJSONKeys(runtime, []string{"--path", "items", "/tmp/data.json"})
	if code != 0 || !strings.Contains(out, "n items array - - /tmp/data.json selected value is not an object") {
		t.Fatalf("runJSONKeys explicit error row = (%q, %d)", out, code)
	}
}

func TestRunJSONLenSupportsStrings(t *testing.T) {
	runtime := engine.CommandRuntime{
		Ctx: context.Background(),
		Ops: contract.Ops{
			RequireAbsolutePath: func(raw string) (string, error) { return raw, nil },
			IsDirPath:           func(context.Context, string) (bool, error) { return false, nil },
			ReadRawContent: func(context.Context, string) (string, error) {
				return `{"title":"simsh"}`, nil
			},
		},
	}

	out, code := runJSONLen(runtime, []string{"--path", "title", "/tmp/data.json"})
	if code != 0 || !strings.Contains(out, "y title string 5 /tmp/data.json -") {
		t.Fatalf("runJSONLen string length = (%q, %d)", out, code)
	}
}

func TestReadJSONInputsUsesReadMany(t *testing.T) {
	called := false
	runtime := engine.CommandRuntime{
		Ctx: context.Background(),
		Ops: contract.Ops{
			ReadMany: func(ctx context.Context, req contract.ReadManyRequest) (contract.ReadManyResult, error) {
				called = true
				return contract.ReadManyResult{
					Entries: []contract.MountContentEntry{
						{Path: "/a.json", Content: `{"a":1}`},
						{Path: "/b.json", Content: `{"b":2}`},
					},
				}, nil
			},
			ReadRawContent: func(context.Context, string) (string, error) {
				t.Fatal("readJSONInputs should not fall back to ReadRawContent when ReadMany succeeds")
				return "", nil
			},
		},
	}

	inputs, err := readJSONInputs(runtime, "json stat", []string{"/a.json", "/b.json"})
	if err != nil {
		t.Fatalf("readJSONInputs error = %v", err)
	}
	if !called {
		t.Fatal("readJSONInputs did not call ReadMany")
	}
	if len(inputs) != 2 || inputs[0].Path != "/a.json" || inputs[1].Path != "/b.json" {
		t.Fatalf("readJSONInputs returned %#v", inputs)
	}
}
