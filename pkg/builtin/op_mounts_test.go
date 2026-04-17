package builtin

import (
	"context"
	"strings"
	"testing"

	"github.com/khicago/simsh/pkg/contract"
	"github.com/khicago/simsh/pkg/engine"
)

type builtinStatusMount struct {
	point   string
	profile contract.MountProfile
}

func (m builtinStatusMount) MountPoint() string                   { return m.point }
func (m builtinStatusMount) Profile() contract.MountProfile       { return m.profile }
func (m builtinStatusMount) Exists(context.Context) (bool, error) { return true, nil }
func (m builtinStatusMount) StatPath(context.Context, string) (contract.MountEntry, error) {
	return contract.MountEntry{}, nil
}
func (m builtinStatusMount) ReadContent(context.Context, string) (string, error) { return "", nil }

type builtinRefreshMount struct {
	builtinStatusMount
	requests   []contract.RefreshRequest
	status     contract.MountRuntimeStatus
	statusErr  error
	effective  []string
	refreshed  []string
	refused    []string
	refreshErr error
}

func newBuiltinRefreshTestMount(point string) *builtinRefreshMount {
	return &builtinRefreshMount{
		builtinStatusMount: builtinStatusMount{
			point: point,
			profile: contract.MountProfile{
				TruthModel:          contract.MountTruthProjection,
				MaterializationMode: contract.MountMaterializationCached,
				WriteSemantics:      contract.MountWriteReadOnly,
				LatencyClass:        contract.MountLatencyRemoteModerate,
				Consistency:         contract.MountConsistency{RefreshRequired: true},
			},
		},
	}
}

func mountRefreshTestPath(parts ...string) string {
	return "/" + strings.Join(parts, "/")
}

func mountRefreshRawPath(base string, parts ...string) string {
	return base + "/" + strings.Join(parts, "/")
}

func (m *builtinRefreshMount) Refresh(_ context.Context, req contract.RefreshRequest) (contract.RefreshResult, error) {
	m.requests = append(m.requests, req)
	if m.refreshErr != nil {
		return contract.RefreshResult{}, m.refreshErr
	}
	return contract.RefreshResult{
		EffectiveTargets: append([]string(nil), m.effective...),
		RefreshedTargets: append([]string(nil), m.refreshed...),
		RefusedTargets:   append([]string(nil), m.refused...),
	}, nil
}

func (m *builtinRefreshMount) MountStatus(context.Context) (contract.MountRuntimeStatus, error) {
	if m.statusErr != nil {
		return contract.MountRuntimeStatus{}, m.statusErr
	}
	return m.status, nil
}

type builtinStatsMount struct{ builtinStatusMount }

func (m builtinStatsMount) Stats(context.Context) (contract.MountStats, error) {
	return contract.MountStats{}, nil
}

func TestRunMountsStatusTextAndJSON(t *testing.T) {
	alphaPath := "/" + "alpha"
	betaPath := "/" + "beta"
	runtime := engine.CommandRuntime{
		Ctx: context.Background(),
		Ops: contract.Ops{
			VirtualMounts: []contract.VirtualMount{
				&builtinRefreshMount{
					builtinStatusMount: builtinStatusMount{
						point: alphaPath,
						profile: contract.MountProfile{
							TruthModel:          contract.MountTruthProjection,
							MaterializationMode: contract.MountMaterializationCached,
							WriteSemantics:      contract.MountWriteReadOnly,
							LatencyClass:        contract.MountLatencyRemoteModerate,
							SupportedCLIClasses: []contract.MountCLIClass{contract.MountCLIList, contract.MountCLIFind},
							Consistency:         contract.MountConsistency{RefreshRequired: true},
							SLO:                 contract.MountSLO{MaxSearchPaths: 2},
						},
					},
					status: contract.MountRuntimeStatus{Freshness: "stale", Materialization: "partial", Detail: "awaiting refresh"},
				},
				builtinStatsMount{builtinStatusMount{
					point: betaPath,
					profile: contract.MountProfile{
						TruthModel:          contract.MountTruthFactual,
						MaterializationMode: contract.MountMaterializationLive,
						WriteSemantics:      contract.MountWriteThrough,
						LatencyClass:        contract.MountLatencyLocalFast,
						SupportedCLIClasses: []contract.MountCLIClass{contract.MountCLIRead},
					},
				}},
			},
		},
	}

	text, code := runMounts(runtime, nil)
	if code != 0 || !strings.Contains(text, alphaPath) || !strings.Contains(text, "refresh=true") || !strings.Contains(text, "status=true") || !strings.Contains(text, "freshness=stale") || !strings.Contains(text, betaPath) {
		t.Fatalf("runMounts(text) = (%q, %d), want mount status rows", text, code)
	}

	jsonOut, code := runMounts(runtime, []string{"--fmt", "json"})
	if code != 0 ||
		!strings.Contains(jsonOut, "\"mounts\"") ||
		!strings.Contains(jsonOut, "\"mount_point\": \""+alphaPath+"\"") ||
		!strings.Contains(jsonOut, "\"has_refresher\": true") ||
		!strings.Contains(jsonOut, "\"has_status\": true") ||
		!strings.Contains(jsonOut, "\"freshness\": \"stale\"") ||
		!strings.Contains(jsonOut, "\"has_stats\": true") {
		t.Fatalf("runMounts(json) = (%q, %d), want structured mount records", jsonOut, code)
	}
}

func TestRunMountsRefresh(t *testing.T) {
	alphaPath := "/" + "alpha"
	nestedPath := alphaPath + "/" + "nested"
	runtime := engine.CommandRuntime{
		Ctx: context.Background(),
		Ops: contract.Ops{
			VirtualMounts: []contract.VirtualMount{
				&builtinRefreshMount{
					builtinStatusMount: builtinStatusMount{
						point: alphaPath,
						profile: contract.MountProfile{
							TruthModel:          contract.MountTruthProjection,
							MaterializationMode: contract.MountMaterializationCached,
							WriteSemantics:      contract.MountWriteReadOnly,
							LatencyClass:        contract.MountLatencyRemoteModerate,
							Consistency:         contract.MountConsistency{RefreshRequired: true},
							SLO:                 contract.MountSLO{MaxRefreshTargets: 1},
						},
					},
					effective: []string{nestedPath},
					refreshed: []string{nestedPath},
					status:    contract.MountRuntimeStatus{Freshness: "live", Materialization: "materialized"},
				},
			},
		},
	}

	text, code := runMounts(runtime, []string{"refresh", nestedPath})
	if code != 0 || !strings.Contains(text, "refreshed=1") || !strings.Contains(text, "refused=0") || !strings.Contains(text, "effective=1") || !strings.Contains(text, "freshness=live") {
		t.Fatalf("runMounts(refresh text) = (%q, %d), want refresh summary", text, code)
	}

	jsonOut, code := runMounts(runtime, []string{"refresh", nestedPath, "--fmt", "json"})
	if code != 0 || !strings.Contains(jsonOut, "\"refresh\"") || !strings.Contains(jsonOut, "\"requested_targets\"") || !strings.Contains(jsonOut, "\"effective_targets\"") || !strings.Contains(jsonOut, "\"refreshed_targets\"") || !strings.Contains(jsonOut, nestedPath) {
		t.Fatalf("runMounts(refresh json) = (%q, %d), want refresh rows", jsonOut, code)
	}
}

func TestRunMountsRefreshDispatchesCanonicalDotDotTarget(t *testing.T) {
	alphaPath := mountRefreshTestPath("alpha")
	betaPath := mountRefreshTestPath("beta")
	target := mountRefreshTestPath("beta", "item")
	alphaMount := newBuiltinRefreshTestMount(alphaPath)
	betaMount := newBuiltinRefreshTestMount(betaPath)
	runtime := engine.CommandRuntime{
		Ctx: context.Background(),
		Ops: contract.Ops{
			VirtualMounts: []contract.VirtualMount{alphaMount, betaMount},
		},
	}

	out, code := runMounts(runtime, []string{"refresh", mountRefreshRawPath(alphaPath, "..", "beta", "item")})
	if code != 0 {
		t.Fatalf("runMounts(refresh canonical dotdot) = (%q, %d), want beta refresh", out, code)
	}
	if len(alphaMount.requests) != 0 {
		t.Fatalf("alpha refresh calls = %d, want 0 after canonical target dispatch", len(alphaMount.requests))
	}
	if len(betaMount.requests) != 1 {
		t.Fatalf("beta refresh calls = %d, want 1", len(betaMount.requests))
	}
	if got := betaMount.requests[0].Targets; len(got) != 1 || got[0] != target {
		t.Fatalf("beta Refresh targets = %#v, want %#v", got, []string{target})
	}
}

func TestRunMountsRefreshCanonicalTargetCanSelectMoreSpecificMount(t *testing.T) {
	repoPath := mountRefreshTestPath("repo")
	subPath := mountRefreshTestPath("repo", "sub")
	target := mountRefreshTestPath("repo", "sub", "file")
	repoMount := newBuiltinRefreshTestMount(repoPath)
	subMount := newBuiltinRefreshTestMount(subPath)
	runtime := engine.CommandRuntime{
		Ctx: context.Background(),
		Ops: contract.Ops{
			VirtualMounts: []contract.VirtualMount{repoMount, subMount},
		},
	}

	out, code := runMounts(runtime, []string{"refresh", mountRefreshRawPath(repoPath, "other", "..", "sub", "file")})
	if code != 0 {
		t.Fatalf("runMounts(refresh canonical specific) = (%q, %d), want sub refresh", out, code)
	}
	if len(repoMount.requests) != 0 {
		t.Fatalf("repo refresh calls = %d, want 0 after canonical target dispatch", len(repoMount.requests))
	}
	if len(subMount.requests) != 1 {
		t.Fatalf("sub refresh calls = %d, want 1", len(subMount.requests))
	}
	if got := subMount.requests[0].Targets; len(got) != 1 || got[0] != target {
		t.Fatalf("sub Refresh targets = %#v, want %#v", got, []string{target})
	}
}

func TestRunMountsRefreshCanonicalTargetCanEscapeMoreSpecificMount(t *testing.T) {
	repoPath := mountRefreshTestPath("repo")
	subPath := mountRefreshTestPath("repo", "sub")
	target := mountRefreshTestPath("repo", "file")
	repoMount := newBuiltinRefreshTestMount(repoPath)
	subMount := newBuiltinRefreshTestMount(subPath)
	runtime := engine.CommandRuntime{
		Ctx: context.Background(),
		Ops: contract.Ops{
			VirtualMounts: []contract.VirtualMount{repoMount, subMount},
		},
	}

	out, code := runMounts(runtime, []string{"refresh", mountRefreshRawPath(subPath, "..", "file")})
	if code != 0 {
		t.Fatalf("runMounts(refresh canonical escape) = (%q, %d), want repo refresh", out, code)
	}
	if len(repoMount.requests) != 1 {
		t.Fatalf("repo refresh calls = %d, want 1", len(repoMount.requests))
	}
	if len(subMount.requests) != 0 {
		t.Fatalf("sub refresh calls = %d, want 0 after canonical target dispatch", len(subMount.requests))
	}
	if got := repoMount.requests[0].Targets; len(got) != 1 || got[0] != target {
		t.Fatalf("repo Refresh targets = %#v, want %#v", got, []string{target})
	}
}

func TestRunMountsRefreshCanonicalTargetOutsideMountRefusesBeforeDispatch(t *testing.T) {
	alphaPath := mountRefreshTestPath("alpha")
	alphaMount := newBuiltinRefreshTestMount(alphaPath)
	runtime := engine.CommandRuntime{
		Ctx: context.Background(),
		Ops: contract.Ops{
			VirtualMounts: []contract.VirtualMount{alphaMount},
		},
	}

	out, code := runMounts(runtime, []string{"refresh", mountRefreshRawPath(alphaPath, "..", "outside", "item")})
	if code == 0 || !strings.Contains(out, mountRefreshTestPath("outside", "item")+": mount not found") {
		t.Fatalf("runMounts(refresh canonical outside) = (%q, %d), want canonical mount-not-found refusal", out, code)
	}
	if len(alphaMount.requests) != 0 {
		t.Fatalf("alpha refresh calls = %d, want 0 after canonical target leaves mount scope", len(alphaMount.requests))
	}
}

func TestRunMountsRefreshOverlappingPrefixesUseLongestSegmentMatch(t *testing.T) {
	fooPath := mountRefreshTestPath("foo")
	fooBarPath := mountRefreshTestPath("foo", "bar")
	foobarPath := mountRefreshTestPath("foobar")
	fooMount := newBuiltinRefreshTestMount(fooPath)
	fooBarMount := newBuiltinRefreshTestMount(fooBarPath)
	foobarMount := newBuiltinRefreshTestMount(foobarPath)
	runtime := engine.CommandRuntime{
		Ctx: context.Background(),
		Ops: contract.Ops{
			VirtualMounts: []contract.VirtualMount{fooMount, fooBarMount, foobarMount},
		},
	}

	out, code := runMounts(runtime, []string{"refresh", mountRefreshRawPath(fooBarPath, "file"), mountRefreshRawPath(foobarPath, "file"), mountRefreshRawPath(fooPath, "baz")})
	if code != 0 {
		t.Fatalf("runMounts(refresh overlapping prefixes) = (%q, %d), want one refresh row per selected mount", out, code)
	}
	if len(fooMount.requests) != 1 {
		t.Fatalf("foo refresh calls = %d, want 1", len(fooMount.requests))
	}
	if got := fooMount.requests[0].Targets; len(got) != 1 || got[0] != mountRefreshTestPath("foo", "baz") {
		t.Fatalf("foo Refresh targets = %#v, want %#v", got, []string{mountRefreshTestPath("foo", "baz")})
	}
	if len(fooBarMount.requests) != 1 {
		t.Fatalf("foo/bar refresh calls = %d, want 1", len(fooBarMount.requests))
	}
	if got := fooBarMount.requests[0].Targets; len(got) != 1 || got[0] != mountRefreshTestPath("foo", "bar", "file") {
		t.Fatalf("foo/bar Refresh targets = %#v, want %#v", got, []string{mountRefreshTestPath("foo", "bar", "file")})
	}
	if len(foobarMount.requests) != 1 {
		t.Fatalf("foobar refresh calls = %d, want 1", len(foobarMount.requests))
	}
	if got := foobarMount.requests[0].Targets; len(got) != 1 || got[0] != mountRefreshTestPath("foobar", "file") {
		t.Fatalf("foobar Refresh targets = %#v, want %#v", got, []string{mountRefreshTestPath("foobar", "file")})
	}
}

func TestRunMountsRefreshRequireNarrowRejectsMountRootTarget(t *testing.T) {
	alphaPath := "/" + "alpha"
	runtime := engine.CommandRuntime{
		Ctx: context.Background(),
		Ops: contract.Ops{
			VirtualMounts: []contract.VirtualMount{
				&builtinRefreshMount{
					builtinStatusMount: builtinStatusMount{
						point: alphaPath,
						profile: contract.MountProfile{
							TruthModel:          contract.MountTruthProjection,
							MaterializationMode: contract.MountMaterializationCached,
							WriteSemantics:      contract.MountWriteReadOnly,
							LatencyClass:        contract.MountLatencyRemoteModerate,
							Consistency:         contract.MountConsistency{RefreshRequired: true},
						},
					},
				},
			},
		},
	}

	out, code := runMounts(runtime, []string{"refresh", "--require-narrow", alphaPath})
	if code != contract.ExitCodeUnsupported || !strings.Contains(out, "is not narrow relative to mount root") {
		t.Fatalf("runMounts(refresh require narrow root) = (%q, %d), want narrow refusal", out, code)
	}
}

func TestRunMountsRefreshRequireNarrowAcceptsDescendantTarget(t *testing.T) {
	alphaPath := "/" + "alpha"
	target := alphaPath + "/" + "nested"
	runtime := engine.CommandRuntime{
		Ctx: context.Background(),
		Ops: contract.Ops{
			VirtualMounts: []contract.VirtualMount{
				&builtinRefreshMount{
					builtinStatusMount: builtinStatusMount{
						point: alphaPath,
						profile: contract.MountProfile{
							TruthModel:          contract.MountTruthProjection,
							MaterializationMode: contract.MountMaterializationCached,
							WriteSemantics:      contract.MountWriteReadOnly,
							LatencyClass:        contract.MountLatencyRemoteModerate,
							Consistency:         contract.MountConsistency{RefreshRequired: true},
						},
					},
					effective: []string{target},
					refreshed: []string{target},
				},
			},
		},
	}

	out, code := runMounts(runtime, []string{"refresh", "--require-narrow", target})
	if code != 0 || !strings.Contains(out, "refreshed=1") || !strings.Contains(out, "effective=1") {
		t.Fatalf("runMounts(refresh require narrow descendant) = (%q, %d), want refresh summary", out, code)
	}
}

func TestRunMountsRefreshPreservesUnsupportedDetail(t *testing.T) {
	alphaPath := "/" + "alpha"
	target := alphaPath + "/" + "nested"
	runtime := engine.CommandRuntime{
		Ctx: context.Background(),
		Ops: contract.Ops{
			VirtualMounts: []contract.VirtualMount{
				&builtinRefreshMount{
					builtinStatusMount: builtinStatusMount{
						point: alphaPath,
						profile: contract.MountProfile{
							TruthModel:          contract.MountTruthProjection,
							MaterializationMode: contract.MountMaterializationCached,
							WriteSemantics:      contract.MountWriteReadOnly,
							LatencyClass:        contract.MountLatencyRemoteHigh,
							Consistency:         contract.MountConsistency{RefreshRequired: true},
						},
					},
					refreshErr: &contract.MountUnsupportedError{
						MountPoint:   alphaPath,
						Capability:   "refresh",
						LatencyClass: contract.MountLatencyRemoteHigh,
						Detail:       alphaPath + ": adapter refused refresh target " + target,
					},
				},
			},
		},
	}

	out, code := runMounts(runtime, []string{"refresh", target})
	if code != contract.ExitCodeUnsupported || !strings.Contains(out, "adapter refused refresh target") {
		t.Fatalf("runMounts(refresh unsupported detail) = (%q, %d), want preserved refusal detail", out, code)
	}
}

func TestRunMountsStatusProviderFailureStaysSeparateFromMountTruth(t *testing.T) {
	alphaPath := "/" + "alpha"
	runtime := engine.CommandRuntime{
		Ctx: context.Background(),
		Ops: contract.Ops{
			VirtualMounts: []contract.VirtualMount{
				&builtinRefreshMount{
					builtinStatusMount: builtinStatusMount{
						point: alphaPath,
						profile: contract.MountProfile{
							TruthModel:          contract.MountTruthProjection,
							MaterializationMode: contract.MountMaterializationCached,
							WriteSemantics:      contract.MountWriteReadOnly,
							LatencyClass:        contract.MountLatencyRemoteModerate,
						},
					},
					statusErr: assertErr("status unavailable"),
				},
			},
		},
	}

	jsonOut, code := runMounts(runtime, []string{"--fmt", "json"})
	if code != 0 || !strings.Contains(jsonOut, "\"status_error\": \"status unavailable\"") {
		t.Fatalf("runMounts(status failure json) = (%q, %d), want status_error truth", jsonOut, code)
	}
	if strings.Contains(jsonOut, "\"freshness\": \"unknown\"") || strings.Contains(jsonOut, "\"materialization\": \"failed\"") {
		t.Fatalf("runMounts(status failure json) = %q, status transport failure must not masquerade as mount truth", jsonOut)
	}
}

type assertErr string

func (e assertErr) Error() string { return string(e) }
