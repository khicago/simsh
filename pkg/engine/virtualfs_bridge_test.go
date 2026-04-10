package engine

import (
	"context"
	"errors"
	"fmt"
	"path"
	"testing"

	"github.com/khicago/simsh/pkg/contract"
)

func virtualPath(parts ...string) string {
	return "/" + path.Join(parts...)
}

type bridgeTestMount struct {
	point   string
	profile contract.MountProfile
}

func newBridgeTestMount(profile contract.MountProfile) *bridgeTestMount {
	return &bridgeTestMount{
		point:   virtualPath("mounted"),
		profile: contract.NormalizeMountProfile(profile),
	}
}

func (m *bridgeTestMount) MountPoint() string { return m.point }
func (m *bridgeTestMount) Profile() contract.MountProfile {
	return contract.NormalizeMountProfile(m.profile)
}
func (m *bridgeTestMount) Exists(context.Context) (bool, error) { return true, nil }
func (m *bridgeTestMount) StatPath(_ context.Context, pathValue string) (contract.MountEntry, error) {
	isDir := pathValue == m.point
	return contract.MountEntry{
		Path: pathValue,
		Name: path.Base(pathValue),
		Meta: contract.PathMeta{
			Exists: true,
			IsDir:  isDir,
			Access: contract.PathAccessReadWrite,
		},
	}, nil
}
func (m *bridgeTestMount) ReadContent(context.Context, string) (string, error) { return "", nil }

type bridgeMutatingMount struct {
	*bridgeTestMount
	mutatedBatches []contract.MutationBatch
}

func (m *bridgeMutatingMount) ApplyMutations(_ context.Context, req contract.MutationBatch) (contract.MutationResult, error) {
	m.mutatedBatches = append(m.mutatedBatches, req)
	records := make([]contract.MutationRecord, 0, len(req.Ops))
	for _, op := range req.Ops {
		records = append(records, contract.MutationRecord{Kind: op.Kind, Path: op.Path, Status: "ok"})
	}
	return contract.MutationResult{Records: records}, nil
}

type bridgeReadableMount struct {
	*bridgeTestMount
}

func (m *bridgeReadableMount) ReadMany(_ context.Context, req contract.ReadManyRequest) (contract.ReadManyResult, error) {
	entries := make([]contract.MountContentEntry, 0, len(req.Paths))
	for _, pathValue := range req.Paths {
		entries = append(entries, contract.MountContentEntry{Path: pathValue, Content: fmt.Sprintf("mounted:%s", pathValue)})
	}
	return contract.ReadManyResult{Entries: entries}, nil
}

func TestWrapOpsApplyMutationsPrevalidatesMountedBatches(t *testing.T) {
	t.Run("invalid mount blocks local side effects", func(t *testing.T) {
		mount := &bridgeMutatingMount{bridgeTestMount: newBridgeTestMount(contract.MountProfile{
			TruthModel:          contract.MountTruthFactual,
			WriteSemantics:      contract.MountWriteThrough,
			LatencyClass:        contract.MountLatencyRemoteModerate,
			SupportedCLIClasses: []contract.MountCLIClass{contract.MountCLIMutate},
		})}
		router, err := newMountRouter([]contract.VirtualMount{mount})
		if err != nil {
			t.Fatalf("newMountRouter(...) error = %v", err)
		}

		localCalls := 0
		ops := router.wrapOps(contract.Ops{
			WriteFile: func(context.Context, string, string) error {
				localCalls++
				return nil
			},
		})

		_, err = ops.ApplyMutations(context.Background(), contract.MutationBatch{
			Ops: []contract.MutationSpec{
				{Kind: contract.MutationWriteFile, Path: virtualPath("workspace", "local.txt"), Content: "local"},
				{Kind: contract.MutationWriteFile, Path: virtualPath("mounted", "file.txt"), Content: "mount"},
			},
		})
		if !errors.Is(err, contract.ErrUnsupported) {
			t.Fatalf("ApplyMutations(...) error = %v, want ErrUnsupported", err)
		}
		if localCalls != 0 {
			t.Fatalf("local WriteFile calls = %d, want 0 before mounted validation passes", localCalls)
		}
		if len(mount.mutatedBatches) != 0 {
			t.Fatalf("mounted ApplyMutations calls = %d, want 0", len(mount.mutatedBatches))
		}
	})

	t.Run("mounted batch uses shared contract gate", func(t *testing.T) {
		mount := newBridgeTestMount(contract.MountProfile{
			TruthModel:          contract.MountTruthFactual,
			WriteSemantics:      contract.MountWriteThrough,
			LatencyClass:        contract.MountLatencyRemoteModerate,
			Consistency:         contract.MountConsistency{PathReadAfterWrite: true},
			SupportedCLIClasses: []contract.MountCLIClass{contract.MountCLIMutate},
		})
		router, err := newMountRouter([]contract.VirtualMount{mount})
		if err != nil {
			t.Fatalf("newMountRouter(...) error = %v", err)
		}

		ops := router.wrapOps(contract.Ops{})
		_, err = ops.ApplyMutations(context.Background(), contract.MutationBatch{
			Ops: []contract.MutationSpec{{Kind: contract.MutationWriteFile, Path: virtualPath("mounted", "file.txt"), Content: "mount"}},
		})
		if !errors.Is(err, contract.ErrUnsupported) {
			t.Fatalf("ApplyMutations(...) error = %v, want ErrUnsupported", err)
		}
	})

	t.Run("remote high mutator refusal keeps explicit detail", func(t *testing.T) {
		mount := newBridgeTestMount(contract.MountProfile{
			TruthModel:          contract.MountTruthFactual,
			WriteSemantics:      contract.MountWriteThrough,
			LatencyClass:        contract.MountLatencyRemoteHigh,
			Consistency:         contract.MountConsistency{PathReadAfterWrite: true},
			SupportedCLIClasses: []contract.MountCLIClass{contract.MountCLIMutate},
		})
		router, err := newMountRouter([]contract.VirtualMount{mount})
		if err != nil {
			t.Fatalf("newMountRouter(...) error = %v", err)
		}

		ops := router.wrapOps(contract.Ops{})
		_, err = ops.ApplyMutations(context.Background(), contract.MutationBatch{
			Ops: []contract.MutationSpec{{Kind: contract.MutationWriteFile, Path: virtualPath("mounted", "file.txt"), Content: "mount"}},
		})
		if !errors.Is(err, contract.ErrUnsupported) {
			t.Fatalf("ApplyMutations(...) error = %v, want ErrUnsupported", err)
		}
		if !contract.IsRemoteHighLatencyUnsupported(err) {
			t.Fatalf("ApplyMutations(...) error = %v, want explicit remote_high_latency refusal", err)
		}
	})
}

func TestWrapOpsReadManyBatchesFilesystemPaths(t *testing.T) {
	mount := &bridgeReadableMount{bridgeTestMount: newBridgeTestMount(contract.MountProfile{
		LatencyClass:        contract.MountLatencyRemoteModerate,
		SupportedCLIClasses: []contract.MountCLIClass{contract.MountCLIRead, contract.MountCLIBulkRead},
	})}
	router, err := newMountRouter([]contract.VirtualMount{mount})
	if err != nil {
		t.Fatalf("newMountRouter(...) error = %v", err)
	}

	var gotReqs []contract.ReadManyRequest
	ops := router.wrapOps(contract.Ops{
		ReadMany: func(_ context.Context, req contract.ReadManyRequest) (contract.ReadManyResult, error) {
			gotReqs = append(gotReqs, req)
			entries := make([]contract.MountContentEntry, 0, len(req.Paths))
			for _, pathValue := range req.Paths {
				entries = append(entries, contract.MountContentEntry{Path: pathValue, Content: fmt.Sprintf("local:%s", pathValue)})
			}
			return contract.ReadManyResult{Entries: entries}, nil
		},
	})

	result, err := ops.ReadMany(context.Background(), contract.ReadManyRequest{
		Paths: []string{
			virtualPath("workspace", "a.txt"),
			virtualPath("mounted", "file.txt"),
			virtualPath("workspace", "b.txt"),
		},
	})
	if err != nil {
		t.Fatalf("ReadMany(...) error = %v", err)
	}
	if len(gotReqs) != 1 {
		t.Fatalf("orig ReadMany calls = %d, want 1", len(gotReqs))
	}
	wantPaths := []string{
		virtualPath("workspace", "a.txt"),
		virtualPath("workspace", "b.txt"),
	}
	if got, want := gotReqs[0].Paths, wantPaths; len(got) != len(want) || got[0] != want[0] || got[1] != want[1] {
		t.Fatalf("orig ReadMany paths = %#v, want %#v", gotReqs[0].Paths, want)
	}
	if len(result.Entries) != 3 {
		t.Fatalf("ReadMany entries = %d, want 3", len(result.Entries))
	}
	if result.Entries[0].Path != virtualPath("workspace", "a.txt") ||
		result.Entries[1].Path != virtualPath("mounted", "file.txt") ||
		result.Entries[2].Path != virtualPath("workspace", "b.txt") {
		t.Fatalf("ReadMany result order = %#v, want request order", result.Entries)
	}
}

func TestWrapOpsApplyMutationsPreservesRequestOrderAcrossFilesystemAndMount(t *testing.T) {
	mount := &bridgeMutatingMount{bridgeTestMount: newBridgeTestMount(contract.MountProfile{
		TruthModel:          contract.MountTruthFactual,
		WriteSemantics:      contract.MountWriteThrough,
		LatencyClass:        contract.MountLatencyRemoteModerate,
		Consistency:         contract.MountConsistency{PathReadAfterWrite: true},
		SupportedCLIClasses: []contract.MountCLIClass{contract.MountCLIMutate},
	})}
	router, err := newMountRouter([]contract.VirtualMount{mount})
	if err != nil {
		t.Fatalf("newMountRouter(...) error = %v", err)
	}

	ops := router.wrapOps(contract.Ops{
		ApplyMutations: func(_ context.Context, req contract.MutationBatch) (contract.MutationResult, error) {
			records := make([]contract.MutationRecord, 0, len(req.Ops))
			for _, op := range req.Ops {
				records = append(records, contract.MutationRecord{Kind: op.Kind, Path: op.Path, Status: "ok"})
			}
			return contract.MutationResult{Records: records}, nil
		},
	})

	requestOps := []contract.MutationSpec{
		{Kind: contract.MutationWriteFile, Path: virtualPath("workspace", "local-1.txt"), Content: "a"},
		{Kind: contract.MutationWriteFile, Path: virtualPath("mounted", "mount-1.txt"), Content: "b"},
		{Kind: contract.MutationWriteFile, Path: virtualPath("workspace", "local-2.txt"), Content: "c"},
	}
	result, err := ops.ApplyMutations(context.Background(), contract.MutationBatch{Ops: requestOps})
	if err != nil {
		t.Fatalf("ApplyMutations(...) error = %v", err)
	}
	if len(result.Records) != len(requestOps) {
		t.Fatalf("record count = %d, want %d", len(result.Records), len(requestOps))
	}
	for idx, op := range requestOps {
		record := result.Records[idx]
		if record.Kind != op.Kind || record.Path != op.Path {
			t.Fatalf("record[%d] = %+v, want kind=%s path=%s", idx, record, op.Kind, op.Path)
		}
	}
}

func TestWrapOpsApplyMutationsPreservesMountUnsupportedDetail(t *testing.T) {
	mount := newBridgeTestMount(contract.MountProfile{
		TruthModel:          contract.MountTruthFactual,
		WriteSemantics:      contract.MountWriteThrough,
		LatencyClass:        contract.MountLatencyRemoteModerate,
		Consistency:         contract.MountConsistency{PathReadAfterWrite: true},
		SupportedCLIClasses: []contract.MountCLIClass{contract.MountCLIMutate},
	})
	router, err := newMountRouter([]contract.VirtualMount{mount})
	if err != nil {
		t.Fatalf("newMountRouter(...) error = %v", err)
	}

	ops := router.wrapOps(contract.Ops{})
	_, err = ops.ApplyMutations(context.Background(), contract.MutationBatch{
		Ops: []contract.MutationSpec{{Kind: contract.MutationWriteFile, Path: virtualPath("mounted", "file.txt"), Content: "mount"}},
	})
	if !errors.Is(err, contract.ErrUnsupported) {
		t.Fatalf("ApplyMutations(...) error = %v, want ErrUnsupported", err)
	}
	var mountErr *contract.MountUnsupportedError
	if !errors.As(err, &mountErr) {
		t.Fatalf("ApplyMutations(...) error = %v, want MountUnsupportedError detail", err)
	}
	if mountErr.Capability != "mutation batch" {
		t.Fatalf("mount unsupported capability = %q, want %q", mountErr.Capability, "mutation batch")
	}
}
