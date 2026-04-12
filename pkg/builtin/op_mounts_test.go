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

func (m builtinStatusMount) MountPoint() string { return m.point }
func (m builtinStatusMount) Profile() contract.MountProfile { return m.profile }
func (m builtinStatusMount) Exists(context.Context) (bool, error) { return true, nil }
func (m builtinStatusMount) StatPath(context.Context, string) (contract.MountEntry, error) { return contract.MountEntry{}, nil }
func (m builtinStatusMount) ReadContent(context.Context, string) (string, error) { return "", nil }

type builtinStatusRefreshMount struct{ builtinStatusMount }

func (m builtinStatusRefreshMount) Refresh(context.Context, contract.RefreshRequest) (contract.RefreshResult, error) {
	return contract.RefreshResult{}, nil
}

type builtinStatusStatsMount struct{ builtinStatusMount }

func (m builtinStatusStatsMount) Stats(context.Context) (contract.MountStats, error) {
	return contract.MountStats{}, nil
}

func TestRunMountsTextAndJSON(t *testing.T) {
	alphaPath := "/" + "alpha"
	betaPath := "/" + "beta"
	runtime := engine.CommandRuntime{
		Ctx: context.Background(),
		Ops: contract.Ops{
			VirtualMounts: []contract.VirtualMount{
				builtinStatusRefreshMount{builtinStatusMount{
					point: alphaPath,
					profile: contract.MountProfile{
						TruthModel:          contract.MountTruthProjection,
						MaterializationMode: contract.MountMaterializationCached,
						WriteSemantics:      contract.MountWriteReadOnly,
						LatencyClass:        contract.MountLatencyRemoteModerate,
						SupportedCLIClasses: []contract.MountCLIClass{contract.MountCLIList, contract.MountCLIFind},
						SLO:                 contract.MountSLO{MaxSearchPaths: 2},
					},
				}},
				builtinStatusStatsMount{builtinStatusMount{
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
	if code != 0 || !strings.Contains(text, alphaPath) || !strings.Contains(text, "refresh=true") || !strings.Contains(text, betaPath) {
		t.Fatalf("runMounts(text) = (%q, %d), want mount summary rows", text, code)
	}

	jsonOut, code := runMounts(runtime, []string{"--fmt", "json"})
	if code != 0 || !strings.Contains(jsonOut, "\"mounts\"") || !strings.Contains(jsonOut, "\"mount_point\": \""+alphaPath+"\"") || !strings.Contains(jsonOut, "\"has_refresher\": true") || !strings.Contains(jsonOut, "\"has_stats\": true") {
		t.Fatalf("runMounts(json) = (%q, %d), want structured mount records", jsonOut, code)
	}
}
