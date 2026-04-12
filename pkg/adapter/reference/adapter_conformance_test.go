package reference

import (
	"context"
	"strings"
	"testing"

	"github.com/khicago/simsh/pkg/adapter/internal/contracttest"
	"github.com/khicago/simsh/pkg/contract"
)

const referenceConformanceExternalObservation = "external-outcome:command_not_found:missing-tool"

func TestReferenceAdapterConformance(t *testing.T) {
	adapter := New(referenceConformanceOptions())

	contracttest.RunLifecycle(t, contracttest.LifecycleSpec{
		Adapter:            adapter,
		ObserveResult:      referenceConformanceObserveResult(),
		RequireMemoryMount: true,
		RequireOpaqueState: true,
		WantMountPoints:    []string{"/knowledge_base/reference", "/resources", "/skills", "/memory"},
		CheckCreated: func(t *testing.T, snapshot contracttest.Snapshot) {
			if content := snapshot.ReadFile(t, "/knowledge_base/reference/guide.md"); !strings.Contains(content, "# Guide") {
				t.Fatalf("created guide content = %q, want heading", content)
			}
			if content := snapshot.ReadFile(t, "/resources/checklists/plan.json"); !strings.Contains(content, "\"steps\"") {
				t.Fatalf("created resource content = %q, want steps payload", content)
			}
			if content := snapshot.ReadFile(t, "/skills/planning/draft-plan/SKILL.md"); !strings.Contains(content, "Use the checklist.") {
				t.Fatalf("created skill content = %q, want seeded skill body", content)
			}

			state := decodeReferenceState(t, []byte(snapshot.ReadFile(t, "/memory/status.json")))
			if state.Freshness != "created" {
				t.Fatalf("created freshness = %q, want %q", state.Freshness, "created")
			}

			view := decodeProjectionViewHelper(t, []byte(snapshot.ReadFile(t, "/memory/projections.json")))
			guide := requireProjectionRecordHelper(t, view.Documents, "/knowledge_base/reference/guide.md")
			if guide.Source != "knowledge_sync" || guide.Freshness != projectionFreshnessSnapshot {
				t.Fatalf("created guide projection = %+v, want knowledge_sync snapshot", guide)
			}
			resource := requireProjectionRecordHelper(t, view.Resources, "/resources/checklists/plan.json")
			if resource.Source != "workflow_catalog" || resource.Freshness != projectionFreshnessLive {
				t.Fatalf("created resource projection = %+v, want workflow_catalog live", resource)
			}
			skill := requireProjectionRecordHelper(t, view.Skills, "/skills/planning/draft-plan/SKILL.md")
			if !skill.Selected || skill.Selection == nil || skill.Selection.State != skillSelectionStateSelected {
				t.Fatalf("created skill projection = %+v, want selected skill", skill)
			}
		},
		CheckObserved: func(t *testing.T, snapshot contracttest.Snapshot) {
			state := decodeReferenceState(t, []byte(snapshot.ReadFile(t, "/memory/status.json")))
			if state.Freshness != "observed" {
				t.Fatalf("observed freshness = %q, want %q", state.Freshness, "observed")
			}
			summary := snapshot.ReadFile(t, "/memory/summary.md")
			if !strings.Contains(summary, "resource_reads: 1") {
				t.Fatalf("observed summary = %q, want resource_reads: 1", summary)
			}
			if !strings.Contains(summary, "written_outputs: 1") {
				t.Fatalf("observed summary = %q, want written_outputs: 1", summary)
			}
			if !strings.Contains(summary, "denied_paths: 1") {
				t.Fatalf("observed summary = %q, want denied_paths: 1", summary)
			}
			observations := snapshot.ReadFile(t, "/memory/observations.md")
			if !strings.Contains(observations, "read-ref:/knowledge_base/reference/guide.md") {
				t.Fatalf("observed observations missing guide read: %q", observations)
			}
			if !strings.Contains(observations, "read-resource:/resources/checklists/plan.json") {
				t.Fatalf("observed observations missing resource read: %q", observations)
			}
			if !strings.Contains(observations, "wrote:/task_outputs/plan.txt") {
				t.Fatalf("observed observations missing write: %q", observations)
			}
			if !strings.Contains(observations, "denied:/knowledge_base/reference/guide.md") {
				t.Fatalf("observed observations missing denial: %q", observations)
			}
			if !strings.Contains(observations, referenceConformanceExternalObservation) {
				t.Fatalf("observed observations missing external outcome: %q", observations)
			}
			if !containsObservation(state.Observations, referenceConformanceExternalObservation) {
				t.Fatalf("observed state observations = %v, want external outcome", state.Observations)
			}
		},
		CheckCheckpointed: func(t *testing.T, snapshot contracttest.Snapshot) {
			state := decodeReferenceState(t, []byte(snapshot.ReadFile(t, "/memory/status.json")))
			if state.Freshness != "checkpointed" {
				t.Fatalf("checkpointed freshness = %q, want %q", state.Freshness, "checkpointed")
			}
			if !containsObservation(state.Observations, referenceConformanceExternalObservation) {
				t.Fatalf("checkpointed observations = %v, want external outcome preserved", state.Observations)
			}
		},
		CheckResumed: func(t *testing.T, snapshot contracttest.Snapshot) {
			state := decodeReferenceState(t, []byte(snapshot.ReadFile(t, "/memory/status.json")))
			if state.Freshness != "resumed" {
				t.Fatalf("resumed freshness = %q, want %q", state.Freshness, "resumed")
			}
			if content := snapshot.ReadFile(t, "/skills/planning/draft-plan/SKILL.md"); !strings.Contains(content, "Use the checklist.") {
				t.Fatalf("resumed skill content = %q, want seeded skill body", content)
			}
			if !containsObservation(state.Observations, "denied:/knowledge_base/reference/guide.md") {
				t.Fatalf("resumed observations = %v, want denied guide observation preserved", state.Observations)
			}
			if !containsObservation(state.Observations, referenceConformanceExternalObservation) {
				t.Fatalf("resumed observations = %v, want external outcome preserved", state.Observations)
			}
			if observations := snapshot.ReadFile(t, virtualPath("memory", "observations.md")); !strings.Contains(observations, referenceConformanceExternalObservation) {
				t.Fatalf("resumed observations file missing external outcome: %q", observations)
			}
		},
		CheckClosed: func(t *testing.T, snapshot contracttest.Snapshot) {
			state := decodeReferenceState(t, []byte(snapshot.ReadFile(t, "/memory/status.json")))
			if state.Freshness != "closed" {
				t.Fatalf("closed freshness = %q, want %q", state.Freshness, "closed")
			}
		},
	})
}

func TestReferenceAdapterMountConformance(t *testing.T) {
	adapter := New(referenceConformanceOptions())

	projection, err := adapter.CreateSession(context.Background(), contract.Session{})
	if err != nil {
		t.Fatalf("CreateSession(...) error = %v, want nil", err)
	}
	if len(projection.OpaqueState) == 0 {
		t.Fatal("CreateSession(...) OpaqueState = empty, want non-empty")
	}
	snapshot := contracttest.Snapshot{
		Phase:      contracttest.PhaseCreated,
		AdapterID:  adapter.AdapterID(),
		Projection: projection,
	}

	snapshot.AssertReadOnlyMountConformance(t, contracttest.MountConformanceSpec{
		MountPoint:                  "/knowledge_base/reference",
		DirectoryPath:               "/knowledge_base/reference",
		WantDirectoryKind:           "reference_dir",
		FilePath:                    "/knowledge_base/reference/guide.md",
		WantFileKind:                "reference_file",
		MissingPath:                 "/knowledge_base/reference/missing.md",
		RecursivePath:               "/knowledge_base/reference",
		WantChildren:                []string{"/knowledge_base/reference/_index.json", "/knowledge_base/reference/guide.md"},
		WantCollectedFiles:          []string{"/knowledge_base/reference/_index.json", "/knowledge_base/reference/guide.md"},
		WantRecursiveSearchPaths:    []string{"/knowledge_base/reference/_index.json", "/knowledge_base/reference/guide.md"},
		WantFileLineCount:           2,
		RequireNonRecursiveDirError: true,
	})

	snapshot.AssertReadOnlyMountConformance(t, contracttest.MountConformanceSpec{
		MountPoint:                  "/resources",
		DirectoryPath:               "/resources/checklists",
		WantDirectoryKind:           "resource_dir",
		FilePath:                    "/resources/checklists/plan.json",
		WantFileKind:                "resource_file",
		MissingPath:                 "/resources/checklists/missing.json",
		RecursivePath:               "/resources",
		WantChildren:                []string{"/resources/checklists/plan.json"},
		WantCollectedFiles:          []string{"/resources/_index.json", "/resources/checklists/plan.json"},
		WantRecursiveSearchPaths:    []string{"/resources/_index.json", "/resources/checklists/plan.json"},
		WantFileLineCount:           1,
		RequireNonRecursiveDirError: true,
	})

	snapshot.AssertReadOnlyMountConformance(t, contracttest.MountConformanceSpec{
		MountPoint:                  "/skills",
		DirectoryPath:               "/skills/planning/draft-plan",
		WantDirectoryKind:           "skill_dir",
		FilePath:                    "/skills/planning/draft-plan/SKILL.md",
		WantFileKind:                "skill_file",
		MissingPath:                 "/skills/planning/missing/SKILL.md",
		RecursivePath:               "/skills",
		WantChildren:                []string{"/skills/planning/draft-plan/SKILL.md"},
		WantCollectedFiles:          []string{"/skills/_index.json", "/skills/planning/draft-plan/SKILL.md"},
		WantRecursiveSearchPaths:    []string{"/skills/_index.json", "/skills/planning/draft-plan/SKILL.md"},
		WantFileLineCount:           2,
		RequireNonRecursiveDirError: true,
	})

	snapshot.AssertReadOnlyMountConformance(t, contracttest.MountConformanceSpec{
		MountPoint:                  "/memory",
		DirectoryPath:               "/memory",
		WantDirectoryKind:           "memory_dir",
		FilePath:                    "/memory/status.json",
		WantFileKind:                "memory_file",
		MissingPath:                 "/memory/missing.json",
		RecursivePath:               "/memory",
		WantChildren:                []string{"/memory/curated.json", "/memory/curated.md", "/memory/denials.json", "/memory/denials.md", "/memory/observations.md", "/memory/projection_metrics.json", "/memory/projection_metrics.md", "/memory/projections.json", "/memory/projections.md", "/memory/skills_audit.json", "/memory/skills_audit.md", "/memory/status.json", "/memory/summary.md", "/memory/workflows.json", "/memory/workflows.md"},
		WantCollectedFiles:          []string{"/memory/curated.json", "/memory/curated.md", "/memory/denials.json", "/memory/denials.md", "/memory/observations.md", "/memory/projection_metrics.json", "/memory/projection_metrics.md", "/memory/projections.json", "/memory/projections.md", "/memory/skills_audit.json", "/memory/skills_audit.md", "/memory/status.json", "/memory/summary.md", "/memory/workflows.json", "/memory/workflows.md"},
		WantRecursiveSearchPaths:    []string{"/memory/curated.json", "/memory/curated.md", "/memory/denials.json", "/memory/denials.md", "/memory/observations.md", "/memory/projection_metrics.json", "/memory/projection_metrics.md", "/memory/projections.json", "/memory/projections.md", "/memory/skills_audit.json", "/memory/skills_audit.md", "/memory/status.json", "/memory/summary.md", "/memory/workflows.json", "/memory/workflows.md"},
		WantFileLineCount:           -1,
		RequireNonRecursiveDirError: true,
	})
}

func referenceConformanceObserveResult() contract.ExecutionResult {
	return contract.ExecutionResult{
		ExecutionID: "exec-conformance",
		Trace: contract.ExecutionTrace{
			ReadPaths:    []string{"/knowledge_base/reference/guide.md", "/resources/checklists/plan.json"},
			WrittenPaths: []string{"/task_outputs/plan.txt"},
			DeniedPaths:  []string{"/knowledge_base/reference/guide.md"},
			ExternalOutcomes: []contract.ExecutionTraceStep{
				{Command: "missing-tool", OutcomeKind: contract.ExternalOutcomeCommandNotFound},
			},
		},
	}
}

func referenceConformanceOptions() Options {
	return Options{
		Documents: map[string]string{
			"guide.md": "# Guide\nhello\n",
		},
		DocumentMetadata: map[string]ProjectionMetadata{
			"guide.md": {Source: "knowledge_sync", Freshness: "snapshot"},
		},
		Resources: map[string]string{
			"checklists/plan.json": "{\"steps\":[\"read\",\"write\"]}\n",
		},
		ResourceMetadata: map[string]ProjectionMetadata{
			"checklists/plan.json": {Source: "workflow_catalog", Freshness: "live"},
		},
		Skills: map[string]string{
			"planning/draft-plan": "# Draft plan\nUse the checklist.\n",
		},
		SkillMetadata: map[string]SkillMetadata{
			"planning/draft-plan": {
				SelectionScope: "planning",
				Eligibility: SkillEligibility{
					State: skillEligibilityEligible,
				},
				Precedence: SkillPrecedence{
					Tier: skillPrecedenceTierWorkspace,
					Rank: 1,
				},
			},
		},
	}
}
