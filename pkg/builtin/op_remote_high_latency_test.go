package builtin

import (
	"context"
	"strings"
	"testing"

	"github.com/khicago/simsh/pkg/contract"
	"github.com/khicago/simsh/pkg/engine"
)

type builtinRemoteHighMount struct {
	point   string
	profile contract.MountProfile
	entry   contract.MountEntry
}

func newBuiltinRemoteHighMount(classes ...contract.MountCLIClass) *builtinRemoteHighMount {
	point := "/remote"
	return &builtinRemoteHighMount{
		point: point,
		profile: contract.NormalizeMountProfile(contract.MountProfile{
			TruthModel:          contract.MountTruthFactual,
			WriteSemantics:      contract.MountWriteThrough,
			LatencyClass:        contract.MountLatencyRemoteHigh,
			SupportedCLIClasses: classes,
			Consistency: contract.MountConsistency{
				PathReadAfterWrite: true,
			},
		}),
		entry: contract.MountEntry{
			Path: point,
			Name: "remote",
			Meta: contract.PathMeta{
				Exists: true,
				IsDir:  true,
				Access: contract.PathAccessReadOnly,
			},
		},
	}
}

func (m *builtinRemoteHighMount) MountPoint() string { return m.point }
func (m *builtinRemoteHighMount) Profile() contract.MountProfile {
	return m.profile
}
func (m *builtinRemoteHighMount) Exists(context.Context) (bool, error) { return true, nil }
func (m *builtinRemoteHighMount) StatPath(context.Context, string) (contract.MountEntry, error) {
	return m.entry, nil
}
func (m *builtinRemoteHighMount) ReadContent(context.Context, string) (string, error) {
	return `{"value":1}`, nil
}

func TestRunRGDoesNotFallbackOnRemoteHighLatencyCapabilityRefusal(t *testing.T) {
	mount := newBuiltinRemoteHighMount(contract.MountCLIContentSearch)
	resolveCalled := false
	readCalled := false
	runtime := engine.CommandRuntime{
		Ctx: context.Background(),
		Ops: contract.Ops{
			RequireAbsolutePath: func(raw string) (string, error) { return raw, nil },
			SearchContent: func(ctx context.Context, req contract.SearchRequest) (contract.SearchResult, error) {
				return contract.SearchMountContent(ctx, mount, req)
			},
			ResolveSearchPaths: func(context.Context, string, bool) ([]string, error) {
				resolveCalled = true
				return []string{"/remote/file.txt"}, nil
			},
			ReadRawContent: func(context.Context, string) (string, error) {
				readCalled = true
				return "hello\n", nil
			},
		},
	}

	out, code := runRG(runtime, []string{"hello", "/remote"})
	if code == 0 || !strings.Contains(out, "remote_high_latency") {
		t.Fatalf("runRG(...) = (%q, %d), want explicit remote_high_latency failure", out, code)
	}
	if resolveCalled || readCalled {
		t.Fatalf("runRG unexpectedly fell back after remote_high refusal: resolve=%v read=%v", resolveCalled, readCalled)
	}
}

func TestRunGrepDoesNotFallbackOnRemoteHighLatencyCapabilityRefusal(t *testing.T) {
	mount := newBuiltinRemoteHighMount(contract.MountCLIContentSearch)
	resolveCalled := false
	readCalled := false
	runtime := engine.CommandRuntime{
		Ctx: context.Background(),
		Ops: contract.Ops{
			RequireAbsolutePath: func(raw string) (string, error) { return raw, nil },
			IsDirPath:           func(context.Context, string) (bool, error) { return true, nil },
			SearchContent: func(ctx context.Context, req contract.SearchRequest) (contract.SearchResult, error) {
				return contract.SearchMountContent(ctx, mount, req)
			},
			ResolveSearchPaths: func(context.Context, string, bool) ([]string, error) {
				resolveCalled = true
				return []string{"/remote/file.txt"}, nil
			},
			ReadRawContent: func(context.Context, string) (string, error) {
				readCalled = true
				return "hello\n", nil
			},
		},
	}

	out, code := runGrep(runtime, []string{"-r", "hello", "/remote"})
	if code == 0 || !strings.Contains(out, "remote_high_latency") {
		t.Fatalf("runGrep(...) = (%q, %d), want explicit remote_high_latency failure", out, code)
	}
	if resolveCalled || readCalled {
		t.Fatalf("runGrep unexpectedly fell back after remote_high refusal: resolve=%v read=%v", resolveCalled, readCalled)
	}
}

func TestRunGlobPreservesRemoteHighLatencyEnumerationRefusal(t *testing.T) {
	mount := newBuiltinRemoteHighMount(contract.MountCLIFind)
	readCalled := false
	runtime := engine.CommandRuntime{
		Ctx: context.Background(),
		Ops: contract.Ops{
			RequireAbsolutePath: func(raw string) (string, error) { return raw, nil },
			IsDirPath:           func(context.Context, string) (bool, error) { return true, nil },
			CollectFilesUnder: func(ctx context.Context, target string) ([]string, error) {
				return contract.EnumerateMountFiles(ctx, mount, target, true)
			},
			ReadRawContent: func(context.Context, string) (string, error) {
				readCalled = true
				return "", nil
			},
		},
	}

	out, code := runGlob(runtime, []string{"*.go", "/remote"})
	if code == 0 || !strings.Contains(out, "remote_high_latency") {
		t.Fatalf("runGlob(...) = (%q, %d), want explicit remote_high_latency failure", out, code)
	}
	if readCalled {
		t.Fatal("runGlob unexpectedly fell back to per-file reads after remote-high refusal")
	}
}

func TestReadJSONInputsDoesNotFallbackOnRemoteHighLatencyBulkReadRefusal(t *testing.T) {
	mount := newBuiltinRemoteHighMount(contract.MountCLIRead, contract.MountCLIBulkRead)
	readRawCalled := false
	runtime := engine.CommandRuntime{
		Ctx: context.Background(),
		Ops: contract.Ops{
			ReadMany: func(ctx context.Context, req contract.ReadManyRequest) (contract.ReadManyResult, error) {
				entries, err := contract.ReadManyFromMount(ctx, mount, req.Paths)
				if err != nil {
					return contract.ReadManyResult{}, err
				}
				return contract.ReadManyResult{Entries: entries}, nil
			},
			ReadRawContent: func(context.Context, string) (string, error) {
				readRawCalled = true
				return `{"unexpected":true}`, nil
			},
		},
	}

	_, err := readJSONInputs(runtime, "json stat", []string{"/remote/a.json", "/remote/b.json"})
	if err == nil || !strings.Contains(err.Error(), "remote_high_latency") {
		t.Fatalf("readJSONInputs(...) error = %v, want explicit remote_high_latency failure", err)
	}
	if readRawCalled {
		t.Fatal("readJSONInputs unexpectedly fell back to ReadRawContent after remote_high refusal")
	}
}

func TestRunMkdirDoesNotFallbackOnRemoteHighLatencyMutationRefusal(t *testing.T) {
	mount := newBuiltinRemoteHighMount(contract.MountCLIMutate)
	makeDirCalled := false
	runtime := engine.CommandRuntime{
		Ctx: context.Background(),
		Ops: contract.Ops{
			RequireAbsolutePath: func(raw string) (string, error) { return raw, nil },
			Policy: contract.ExecutionPolicy{
				WriteMode:      contract.WriteModeFull,
				MaxOutputBytes: 4 << 20,
				Timeout:        contract.DefaultPolicy().Timeout,
			},
			CheckPathOp: func(context.Context, contract.PathOp, string) error { return nil },
			ApplyMutations: func(ctx context.Context, req contract.MutationBatch) (contract.MutationResult, error) {
				return contract.ApplyMountMutations(ctx, mount, req)
			},
			MakeDir: func(context.Context, string) error {
				makeDirCalled = true
				return nil
			},
		},
	}

	out, code := runMkdir(runtime, []string{"/remote/new"})
	if code == 0 || !strings.Contains(out, "remote_high_latency") {
		t.Fatalf("runMkdir(...) = (%q, %d), want explicit remote_high_latency failure", out, code)
	}
	if makeDirCalled {
		t.Fatal("runMkdir unexpectedly fell back to MakeDir after remote_high refusal")
	}
}

func TestRunMvDoesNotFallbackOnRemoteHighLatencyMutationRefusal(t *testing.T) {
	mount := newBuiltinRemoteHighMount(contract.MountCLIMutate)
	writeCalled := false
	removeCalled := false
	runtime := engine.CommandRuntime{
		Ctx: context.Background(),
		Ops: contract.Ops{
			RequireAbsolutePath: func(raw string) (string, error) { return raw, nil },
			Policy: contract.ExecutionPolicy{
				WriteMode:      contract.WriteModeFull,
				MaxOutputBytes: 4 << 20,
				Timeout:        contract.DefaultPolicy().Timeout,
			},
			CheckPathOp: func(context.Context, contract.PathOp, string) error { return nil },
			ReadRawContent: func(context.Context, string) (string, error) {
				return "payload", nil
			},
			ApplyMutations: func(ctx context.Context, req contract.MutationBatch) (contract.MutationResult, error) {
				return contract.ApplyMountMutations(ctx, mount, req)
			},
			WriteFile: func(context.Context, string, string) error {
				writeCalled = true
				return nil
			},
			RemoveFile: func(context.Context, string) error {
				removeCalled = true
				return nil
			},
		},
	}

	out, code := runMv(runtime, []string{"/remote/src", "/remote/dest"})
	if code == 0 || !strings.Contains(out, "remote_high_latency") {
		t.Fatalf("runMv(...) = (%q, %d), want explicit remote_high_latency failure", out, code)
	}
	if writeCalled || removeCalled {
		t.Fatalf("runMv unexpectedly fell back after remote_high refusal: write=%v remove=%v", writeCalled, removeCalled)
	}
}

func TestRunRmDoesNotFallbackOnRemoteHighLatencyMutationRefusal(t *testing.T) {
	mount := newBuiltinRemoteHighMount(contract.MountCLIMutate)
	removeCalled := false
	runtime := engine.CommandRuntime{
		Ctx: context.Background(),
		Ops: contract.Ops{
			RequireAbsolutePath: func(raw string) (string, error) { return raw, nil },
			Policy: contract.ExecutionPolicy{
				WriteMode:      contract.WriteModeFull,
				MaxOutputBytes: 4 << 20,
				Timeout:        contract.DefaultPolicy().Timeout,
			},
			CheckPathOp: func(context.Context, contract.PathOp, string) error { return nil },
			ApplyMutations: func(ctx context.Context, req contract.MutationBatch) (contract.MutationResult, error) {
				return contract.ApplyMountMutations(ctx, mount, req)
			},
			RemoveFile: func(context.Context, string) error {
				removeCalled = true
				return nil
			},
		},
	}

	out, code := runRm(runtime, []string{"/remote/obsolete.txt"})
	if code == 0 || !strings.Contains(out, "remote_high_latency") {
		t.Fatalf("runRm(...) = (%q, %d), want explicit remote_high_latency failure", out, code)
	}
	if removeCalled {
		t.Fatal("runRm unexpectedly fell back to RemoveFile after remote_high refusal")
	}
}
