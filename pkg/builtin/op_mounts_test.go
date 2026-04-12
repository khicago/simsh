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
	status    contract.MountRuntimeStatus
	statusErr error
	refreshed []string
	refused   []string
}

func (m builtinRefreshMount) Refresh(context.Context, contract.RefreshRequest) (contract.RefreshResult, error) {
	return contract.RefreshResult{
		RefreshedTargets: append([]string(nil), m.refreshed...),
		RefusedTargets:   append([]string(nil), m.refused...),
	}, nil
}

func (m builtinRefreshMount) MountStatus(context.Context) (contract.MountRuntimeStatus, error) {
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
				builtinRefreshMount{
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
				builtinRefreshMount{
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
					refreshed: []string{nestedPath},
					status:    contract.MountRuntimeStatus{Freshness: "live", Materialization: "materialized"},
				},
			},
		},
	}

	text, code := runMounts(runtime, []string{"refresh", nestedPath})
	if code != 0 || !strings.Contains(text, "refreshed=1") || !strings.Contains(text, "refused=0") || !strings.Contains(text, "freshness=live") {
		t.Fatalf("runMounts(refresh text) = (%q, %d), want refresh summary", text, code)
	}

	jsonOut, code := runMounts(runtime, []string{"refresh", nestedPath, "--fmt", "json"})
	if code != 0 || !strings.Contains(jsonOut, "\"refresh\"") || !strings.Contains(jsonOut, "\"requested_targets\"") || !strings.Contains(jsonOut, "\"refreshed_targets\"") || !strings.Contains(jsonOut, nestedPath) {
		t.Fatalf("runMounts(refresh json) = (%q, %d), want refresh rows", jsonOut, code)
	}
}

func TestRunMountsStatusProviderFailureStaysSeparateFromMountTruth(t *testing.T) {
	alphaPath := "/" + "alpha"
	runtime := engine.CommandRuntime{
		Ctx: context.Background(),
		Ops: contract.Ops{
			VirtualMounts: []contract.VirtualMount{
				builtinRefreshMount{
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
