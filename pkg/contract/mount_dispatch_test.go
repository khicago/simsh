package contract

import (
	"context"
	"errors"
	"fmt"
	"path"
	"strings"
	"testing"
)

type dispatchTestMount struct {
	point    string
	profile  MountProfile
	entries  map[string]MountEntry
	contents map[string]string
}

func newDispatchTestMount(profile MountProfile) *dispatchTestMount {
	point := "/remote"
	rootEntry := MountEntry{
		Path: point,
		Name: path.Base(point),
		Meta: PathMeta{
			Exists: true,
			IsDir:  true,
			Access: PathAccessReadOnly,
		},
	}
	fileEntry := MountEntry{
		Path: point + "/data.json",
		Name: "data.json",
		Meta: PathMeta{
			Exists: true,
			IsDir:  false,
			Access: PathAccessReadOnly,
		},
	}
	return &dispatchTestMount{
		point:   point,
		profile: NormalizeMountProfile(profile),
		entries: map[string]MountEntry{
			point:                rootEntry,
			point + "/data.json": fileEntry,
		},
		contents: map[string]string{
			point + "/data.json": `{"ok":true}`,
		},
	}
}

func (m *dispatchTestMount) MountPoint() string { return m.point }
func (m *dispatchTestMount) Profile() MountProfile {
	return m.profile
}
func (m *dispatchTestMount) Exists(context.Context) (bool, error) { return true, nil }
func (m *dispatchTestMount) StatPath(_ context.Context, pathValue string) (MountEntry, error) {
	entry, ok := m.entries[pathValue]
	if !ok {
		return MountEntry{}, fmt.Errorf("%s: no such file or directory", pathValue)
	}
	return entry, nil
}
func (m *dispatchTestMount) ReadContent(_ context.Context, pathValue string) (string, error) {
	content, ok := m.contents[pathValue]
	if !ok {
		return "", fmt.Errorf("%s: no such file", pathValue)
	}
	return content, nil
}

type dispatchListerMount struct{ *dispatchTestMount }

func (m *dispatchListerMount) ListEntries(_ context.Context, req ListEntriesRequest) (ListEntriesResult, error) {
	if req.Dir != m.point {
		return ListEntriesResult{}, fmt.Errorf("unexpected dir %s", req.Dir)
	}
	return ListEntriesResult{Entries: []MountEntry{m.entries[m.point+"/data.json"]}}, nil
}

type dispatchRefreshMount struct {
	*dispatchTestMount
	result RefreshResult
}

func (m *dispatchRefreshMount) Refresh(_ context.Context, req RefreshRequest) (RefreshResult, error) {
	if len(m.result.EffectiveTargets) > 0 || len(m.result.RefreshedTargets) > 0 || len(m.result.RefusedTargets) > 0 {
		return RefreshResult{
			EffectiveTargets: append([]string(nil), m.result.EffectiveTargets...),
			RefreshedTargets: append([]string(nil), m.result.RefreshedTargets...),
			RefusedTargets:   append([]string(nil), m.result.RefusedTargets...),
		}, nil
	}
	return RefreshResult{
		EffectiveTargets: append([]string(nil), req.Targets...),
		RefreshedTargets: append([]string(nil), req.Targets...),
	}, nil
}

func TestAllowsUnsupportedFallbackPreservesLocalFallbacks(t *testing.T) {
	mount := newDispatchTestMount(MountProfile{
		LatencyClass:        MountLatencyLocalFast,
		SupportedCLIClasses: []MountCLIClass{MountCLIBulkRead},
	})
	err := unsupportedMountCapability(mount, "bulk read")
	if !errors.Is(err, ErrUnsupported) {
		t.Fatalf("unsupportedMountCapability(...) error = %v, want ErrUnsupported", err)
	}
	if !AllowsUnsupportedFallback(err) {
		t.Fatalf("AllowsUnsupportedFallback(...) = false, want true for local fallback")
	}
}

func TestListMountChildrenRemoteHighRequiresEntryLister(t *testing.T) {
	mount := newDispatchTestMount(MountProfile{
		LatencyClass:        MountLatencyRemoteHigh,
		SupportedCLIClasses: []MountCLIClass{MountCLIList},
	})
	_, err := ListMountChildren(context.Background(), mount, mount.point)
	if !errors.Is(err, ErrUnsupported) {
		t.Fatalf("ListMountChildren(...) error = %v, want ErrUnsupported", err)
	}
	if AllowsUnsupportedFallback(err) {
		t.Fatal("ListMountChildren(...) unexpectedly allowed fallback for remote_high_latency mount")
	}
	if !IsRemoteHighLatencyUnsupported(err) || !strings.Contains(err.Error(), "remote_high_latency") {
		t.Fatalf("ListMountChildren(...) error = %v, want explicit remote_high_latency refusal", err)
	}
}

func TestEnumerateMountFilesRemoteHighRequiresPathEnumerator(t *testing.T) {
	mount := &dispatchListerMount{dispatchTestMount: newDispatchTestMount(MountProfile{
		LatencyClass:        MountLatencyRemoteHigh,
		SupportedCLIClasses: []MountCLIClass{MountCLIList, MountCLIFind},
	})}
	_, err := EnumerateMountFiles(context.Background(), mount, mount.point, true)
	if !errors.Is(err, ErrUnsupported) {
		t.Fatalf("EnumerateMountFiles(...) error = %v, want ErrUnsupported", err)
	}
	if AllowsUnsupportedFallback(err) {
		t.Fatal("EnumerateMountFiles(...) unexpectedly allowed fallback for remote_high_latency mount")
	}
	if !strings.Contains(err.Error(), "path enumeration") {
		t.Fatalf("EnumerateMountFiles(...) error = %v, want path enumeration refusal", err)
	}
}

func TestReadManyFromMountRemoteHighRequiresBulkReader(t *testing.T) {
	mount := newDispatchTestMount(MountProfile{
		LatencyClass:        MountLatencyRemoteHigh,
		SupportedCLIClasses: []MountCLIClass{MountCLIRead, MountCLIBulkRead},
	})
	_, err := ReadManyFromMount(context.Background(), mount, []string{mount.point + "/data.json", mount.point + "/data.json"})
	if !errors.Is(err, ErrUnsupported) {
		t.Fatalf("ReadManyFromMount(...) error = %v, want ErrUnsupported", err)
	}
	if AllowsUnsupportedFallback(err) {
		t.Fatal("ReadManyFromMount(...) unexpectedly allowed fallback for remote_high_latency mount")
	}
	if !strings.Contains(err.Error(), "bulk read") {
		t.Fatalf("ReadManyFromMount(...) error = %v, want bulk read refusal", err)
	}
}

func TestReadManyFromMountRejectsBatchCountOverSLO(t *testing.T) {
	mount := newDispatchTestMount(MountProfile{
		LatencyClass:        MountLatencyRemoteModerate,
		SupportedCLIClasses: []MountCLIClass{MountCLIRead, MountCLIBulkRead},
		SLO:                 MountSLO{MaxBatchCount: 1},
	})
	dataPath := mount.point + "/" + "data.json"
	_, err := ReadManyFromMountRequest(context.Background(), mount, ReadManyRequest{
		Paths:      []string{dataPath, dataPath},
		MaxEntries: 2,
	})
	if !errors.Is(err, ErrUnsupported) {
		t.Fatalf("ReadManyFromMountRequest(...) error = %v, want ErrUnsupported", err)
	}
	if !strings.Contains(err.Error(), "batch count") {
		t.Fatalf("ReadManyFromMountRequest(...) error = %v, want batch-count refusal", err)
	}
}

func TestSearchMountContentRejectsTargetCountOverSLO(t *testing.T) {
	mount := newDispatchTestMount(MountProfile{
		LatencyClass:        MountLatencyRemoteModerate,
		SupportedCLIClasses: []MountCLIClass{MountCLIContentSearch},
		SLO:                 MountSLO{MaxSearchPaths: 1},
	})
	otherPath := mount.point + "/" + "other"
	_, err := SearchMountContent(context.Background(), mount, SearchRequest{
		Pattern: "ok",
		Targets: []string{mount.point, otherPath},
	})
	if !errors.Is(err, ErrUnsupported) {
		t.Fatalf("SearchMountContent(...) error = %v, want ErrUnsupported", err)
	}
	if !strings.Contains(err.Error(), "search path budget") {
		t.Fatalf("SearchMountContent(...) error = %v, want search-budget refusal", err)
	}
}

func TestSearchMountContentRemoteHighRequiresContentSearcher(t *testing.T) {
	mount := newDispatchTestMount(MountProfile{
		LatencyClass:        MountLatencyRemoteHigh,
		SupportedCLIClasses: []MountCLIClass{MountCLIContentSearch},
	})
	_, err := SearchMountContent(context.Background(), mount, SearchRequest{Pattern: "ok", Targets: []string{mount.point}})
	if !errors.Is(err, ErrUnsupported) {
		t.Fatalf("SearchMountContent(...) error = %v, want ErrUnsupported", err)
	}
	if AllowsUnsupportedFallback(err) {
		t.Fatal("SearchMountContent(...) unexpectedly allowed fallback for remote_high_latency mount")
	}
	if !strings.Contains(err.Error(), "content search") {
		t.Fatalf("SearchMountContent(...) error = %v, want content search refusal", err)
	}
}

func TestApplyMountMutationsRemoteHighRequiresMutator(t *testing.T) {
	mount := newDispatchTestMount(MountProfile{
		TruthModel:          MountTruthFactual,
		WriteSemantics:      MountWriteThrough,
		LatencyClass:        MountLatencyRemoteHigh,
		SupportedCLIClasses: []MountCLIClass{MountCLIMutate},
		Consistency: MountConsistency{
			PathReadAfterWrite: true,
		},
	})
	_, err := ApplyMountMutations(context.Background(), mount, MutationBatch{
		Ops: []MutationSpec{{Kind: MutationMakeDir, Path: mount.point + "/new"}},
	})
	if !errors.Is(err, ErrUnsupported) {
		t.Fatalf("ApplyMountMutations(...) error = %v, want ErrUnsupported", err)
	}
	if AllowsUnsupportedFallback(err) {
		t.Fatal("ApplyMountMutations(...) unexpectedly allowed fallback for remote_high_latency mount")
	}
	if !strings.Contains(err.Error(), "mutation batch") {
		t.Fatalf("ApplyMountMutations(...) error = %v, want mutation batch refusal", err)
	}
}

func TestRefreshMountRemoteHighRequiresExplicitScope(t *testing.T) {
	mount := &dispatchRefreshMount{dispatchTestMount: newDispatchTestMount(MountProfile{
		LatencyClass:        MountLatencyRemoteHigh,
		SupportedCLIClasses: []MountCLIClass{MountCLIRead},
		Consistency:         MountConsistency{RefreshRequired: true},
	})}
	_, err := RefreshMount(context.Background(), mount, RefreshRequest{})
	if !errors.Is(err, ErrUnsupported) {
		t.Fatalf("RefreshMount(...) error = %v, want ErrUnsupported", err)
	}
	if !strings.Contains(err.Error(), "requires explicit scoped targets") {
		t.Fatalf("RefreshMount(...) error = %v, want explicit scoped refresh refusal", err)
	}
}

func TestRefreshMountRemoteHighRejectsMountRootTarget(t *testing.T) {
	mount := &dispatchRefreshMount{dispatchTestMount: newDispatchTestMount(MountProfile{
		LatencyClass:        MountLatencyRemoteHigh,
		SupportedCLIClasses: []MountCLIClass{MountCLIRead},
		Consistency:         MountConsistency{RefreshRequired: true},
	})}
	_, err := RefreshMount(context.Background(), mount, RefreshRequest{Targets: []string{mount.point}})
	if !errors.Is(err, ErrUnsupported) {
		t.Fatalf("RefreshMount(...) error = %v, want ErrUnsupported", err)
	}
	if !strings.Contains(err.Error(), "not scoped below mount root") {
		t.Fatalf("RefreshMount(...) error = %v, want remote_high_latency narrow refusal", err)
	}
}

func TestRefreshMountRequireNarrowRejectsMountRootTarget(t *testing.T) {
	mount := &dispatchRefreshMount{dispatchTestMount: newDispatchTestMount(MountProfile{
		LatencyClass:        MountLatencyRemoteModerate,
		SupportedCLIClasses: []MountCLIClass{MountCLIRead},
		Consistency:         MountConsistency{RefreshRequired: true},
	})}
	_, err := RefreshMount(context.Background(), mount, RefreshRequest{
		Targets:       []string{mount.point},
		RequireNarrow: true,
	})
	if !errors.Is(err, ErrUnsupported) {
		t.Fatalf("RefreshMount(...) error = %v, want ErrUnsupported", err)
	}
	if !strings.Contains(err.Error(), "is not narrow relative to mount root") {
		t.Fatalf("RefreshMount(...) error = %v, want require_narrow refusal", err)
	}
}

func TestRefreshMountRejectsRefreshPathBudgetOverSLO(t *testing.T) {
	mount := &dispatchRefreshMount{dispatchTestMount: newDispatchTestMount(MountProfile{
		LatencyClass:        MountLatencyRemoteModerate,
		SupportedCLIClasses: []MountCLIClass{MountCLIRead},
		Consistency:         MountConsistency{RefreshRequired: true},
		SLO:                 MountSLO{MaxRefreshTargets: 1},
	})}
	_, err := RefreshMount(context.Background(), mount, RefreshRequest{
		Targets: []string{mount.point, mount.point + "/" + "data.json"},
	})
	if !errors.Is(err, ErrUnsupported) {
		t.Fatalf("RefreshMount(...) error = %v, want ErrUnsupported", err)
	}
	if !strings.Contains(err.Error(), "refresh target budget") {
		t.Fatalf("RefreshMount(...) error = %v, want refresh-target budget refusal", err)
	}
}

func TestRefreshMountRequireNarrowRejectsBroadenedEffectiveTargets(t *testing.T) {
	mount := &dispatchRefreshMount{
		dispatchTestMount: newDispatchTestMount(MountProfile{
			LatencyClass:        MountLatencyRemoteModerate,
			SupportedCLIClasses: []MountCLIClass{MountCLIRead},
			Consistency:         MountConsistency{RefreshRequired: true},
		}),
		result: RefreshResult{
			EffectiveTargets: []string{mountRefreshTestRoot()},
			RefreshedTargets: []string{mountRefreshTestRoot()},
		},
	}
	target := mount.point + "/" + "data.json"
	_, err := RefreshMount(context.Background(), mount, RefreshRequest{
		Targets:       []string{target},
		RequireNarrow: true,
	})
	if !errors.Is(err, ErrUnsupported) {
		t.Fatalf("RefreshMount(...) error = %v, want ErrUnsupported", err)
	}
	if !strings.Contains(err.Error(), "broadened effective target") {
		t.Fatalf("RefreshMount(...) error = %v, want broaden refusal", err)
	}
}

func TestRefreshMountRemoteHighRejectsBroadenedEffectiveTargets(t *testing.T) {
	mount := &dispatchRefreshMount{
		dispatchTestMount: newDispatchTestMount(MountProfile{
			LatencyClass:        MountLatencyRemoteHigh,
			SupportedCLIClasses: []MountCLIClass{MountCLIRead},
			Consistency:         MountConsistency{RefreshRequired: true},
		}),
		result: RefreshResult{
			EffectiveTargets: []string{mountRefreshTestRoot()},
			RefreshedTargets: []string{mountRefreshTestRoot()},
		},
	}
	target := mount.point + "/" + "data.json"
	_, err := RefreshMount(context.Background(), mount, RefreshRequest{
		Targets: []string{target},
	})
	if !errors.Is(err, ErrUnsupported) {
		t.Fatalf("RefreshMount(...) error = %v, want ErrUnsupported", err)
	}
	if !strings.Contains(err.Error(), "broadened effective target") {
		t.Fatalf("RefreshMount(...) error = %v, want remote_high_latency broaden refusal", err)
	}
}

func TestRefreshMountRequireNarrowRejectsBroadenedReportedTargets(t *testing.T) {
	mount := &dispatchRefreshMount{
		dispatchTestMount: newDispatchTestMount(MountProfile{
			LatencyClass:        MountLatencyRemoteModerate,
			SupportedCLIClasses: []MountCLIClass{MountCLIRead},
			Consistency:         MountConsistency{RefreshRequired: true},
		}),
		result: RefreshResult{
			EffectiveTargets: []string{mountRefreshTestRoot() + "/" + "data.json"},
			RefreshedTargets: []string{mountRefreshTestRoot()},
		},
	}
	target := mount.point + "/" + "data.json"
	_, err := RefreshMount(context.Background(), mount, RefreshRequest{
		Targets:       []string{target},
		RequireNarrow: true,
	})
	if !errors.Is(err, ErrUnsupported) {
		t.Fatalf("RefreshMount(...) error = %v, want ErrUnsupported", err)
	}
	if !strings.Contains(err.Error(), "broadened refreshed target") {
		t.Fatalf("RefreshMount(...) error = %v, want reported-target broaden refusal", err)
	}
}

func TestRefreshMountDefaultsEffectiveTargetsToRequestedTargets(t *testing.T) {
	mount := &dispatchRefreshMount{dispatchTestMount: newDispatchTestMount(MountProfile{
		LatencyClass:        MountLatencyRemoteModerate,
		SupportedCLIClasses: []MountCLIClass{MountCLIRead},
		Consistency:         MountConsistency{RefreshRequired: true},
	})}
	target := mount.point + "/" + "data.json"
	result, err := RefreshMount(context.Background(), mount, RefreshRequest{
		Targets: []string{target},
	})
	if err != nil {
		t.Fatalf("RefreshMount(...) error = %v", err)
	}
	if len(result.EffectiveTargets) != 1 || result.EffectiveTargets[0] != target {
		t.Fatalf("RefreshMount(...).EffectiveTargets = %#v, want requested target", result.EffectiveTargets)
	}
}

func mountRefreshTestRoot() string {
	return "/" + "remote"
}
