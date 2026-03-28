package reference

import (
	"context"
	"encoding/json"
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/khicago/simsh/pkg/contract"
)

func TestNormalizeDocumentName(t *testing.T) {
	tests := []struct {
		raw  string
		want string
	}{
		{raw: " guide ", want: "guide.md"},
		{raw: "notes.md", want: "notes.md"},
		{raw: "", want: "empty.md"},
	}

	for _, tt := range tests {
		if got := normalizeDocumentName(tt.raw); got != tt.want {
			t.Errorf("normalizeDocumentName(%q) = %q, want %q", tt.raw, got, tt.want)
		}
	}
}

func TestNormalizeSkillName(t *testing.T) {
	tests := []struct {
		raw  string
		want string
	}{
		{raw: " lint/check ", want: "lint/check/SKILL.md"},
		{raw: "catalog/review.md", want: "catalog/review.md"},
		{raw: "catalog/review", want: "catalog/review/SKILL.md"},
		{raw: "", want: "default/SKILL.md"},
	}

	for _, tt := range tests {
		if got := normalizeSkillName(tt.raw); got != tt.want {
			t.Fatalf("normalizeSkillName(%q) = %q, want %q", tt.raw, got, tt.want)
		}
	}
}

func TestNormalizeSkillMetadataDefaultsAndValidation(t *testing.T) {
	got := normalizeSkillMetadata(SkillMetadata{}, "skill_bundle", projectionFreshnessSnapshot)
	want := SkillMetadata{
		Source:    "skill_bundle",
		Freshness: projectionFreshnessSnapshot,
		Eligibility: SkillEligibility{
			State: skillEligibilityUnknown,
		},
		Precedence: SkillPrecedence{
			Tier: skillPrecedenceTierBundled,
			Rank: 100,
		},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("normalizeSkillMetadata(default) = %#v, want %#v", got, want)
	}

	got = normalizeSkillMetadata(SkillMetadata{
		Source:         "workspace_registry",
		Freshness:      projectionFreshnessLive,
		SelectionScope: "planning/default ",
		Eligibility: SkillEligibility{
			State:  skillEligibilityEligible,
			Reason: "bin:rg",
		},
		Precedence: SkillPrecedence{
			Tier: skillPrecedenceTierWorkspace,
			Rank: 1,
		},
	}, "skill_bundle", projectionFreshnessSnapshot)
	if got.Source != "workspace_registry" ||
		got.Freshness != projectionFreshnessLive ||
		got.SelectionScope != "planning/default" ||
		got.Eligibility.State != skillEligibilityEligible ||
		got.Eligibility.Reason != "bin:rg" ||
		got.Precedence.Tier != skillPrecedenceTierWorkspace ||
		got.Precedence.Rank != 1 {
		t.Fatalf("normalizeSkillMetadata(valid) = %#v, want preserved explicit metadata", got)
	}

	got = normalizeSkillMetadata(SkillMetadata{
		Freshness:      "invalid",
		SelectionScope: " / ",
		Eligibility: SkillEligibility{
			State:  "invalid",
			Reason: " missing env ",
		},
		Precedence: SkillPrecedence{
			Tier: "invalid",
			Rank: -7,
		},
	}, "skill_bundle", projectionFreshnessSnapshot)
	if got.Freshness != projectionFreshnessSnapshot ||
		got.SelectionScope != "" ||
		got.Eligibility.State != skillEligibilityUnknown ||
		got.Eligibility.Reason != "missing env" ||
		got.Precedence.Tier != skillPrecedenceTierBundled ||
		got.Precedence.Rank != 0 {
		t.Fatalf("normalizeSkillMetadata(invalid) = %#v, want normalized fallback metadata", got)
	}
}

func TestDeriveSkillSelectionOutcomesUsesExplicitScope(t *testing.T) {
	entries := map[string]skillEntry{
		"planning/draft-plan/SKILL.md": {
			Metadata: normalizeSkillMetadata(SkillMetadata{
				Source:         "workspace_catalog",
				Freshness:      projectionFreshnessLive,
				SelectionScope: "planning",
				Eligibility: SkillEligibility{
					State: skillEligibilityEligible,
				},
				Precedence: SkillPrecedence{
					Tier: skillPrecedenceTierWorkspace,
					Rank: 1,
				},
			}, "skill_bundle", projectionFreshnessSnapshot),
		},
		"planning/fallback/SKILL.md": {
			Metadata: normalizeSkillMetadata(SkillMetadata{
				Source:         "bundled_catalog",
				Freshness:      projectionFreshnessSnapshot,
				SelectionScope: "planning",
				Eligibility: SkillEligibility{
					State: skillEligibilityEligible,
				},
				Precedence: SkillPrecedence{
					Tier: skillPrecedenceTierBundled,
					Rank: 90,
				},
			}, "skill_bundle", projectionFreshnessSnapshot),
		},
		"planning/blocked/SKILL.md": {
			Metadata: normalizeSkillMetadata(SkillMetadata{
				Source:         "bundled_catalog",
				Freshness:      projectionFreshnessSnapshot,
				SelectionScope: "planning",
				Eligibility: SkillEligibility{
					State:  skillEligibilityIneligible,
					Reason: "missing_env:PLAN_TOKEN",
				},
				Precedence: SkillPrecedence{
					Tier: skillPrecedenceTierWorkspace,
					Rank: 0,
				},
			}, "skill_bundle", projectionFreshnessSnapshot),
		},
	}

	outcomes := deriveSkillSelectionOutcomes(entries)
	draft := outcomes["planning/draft-plan/SKILL.md"]
	if !draft.Selected || draft.Selection == nil || draft.Selection.State != skillSelectionStateSelected || draft.Selection.Mode != skillSelectionModeDerived || draft.Selection.Scope != "planning" || draft.Selection.Reason != "highest_precedence" || draft.Selection.WinnerPath != "" {
		t.Fatalf("draft selection outcome = %+v, want derived selected winner", draft)
	}
	fallback := outcomes["planning/fallback/SKILL.md"]
	if fallback.Selected || fallback.Selection == nil || fallback.Selection.State != skillSelectionStateNotSelected || fallback.Selection.Mode != skillSelectionModeDerived || fallback.Selection.Scope != "planning" || fallback.Selection.Reason != "higher_precedence_selected" || fallback.Selection.WinnerPath != "/skills/planning/draft-plan/SKILL.md" {
		t.Fatalf("fallback selection outcome = %+v, want derived loser pointing at winner", fallback)
	}
	blocked := outcomes["planning/blocked/SKILL.md"]
	if blocked.Selected || blocked.Selection == nil || blocked.Selection.State != skillSelectionStateNotSelected || blocked.Selection.Mode != skillSelectionModeDerived || blocked.Selection.Scope != "planning" || blocked.Selection.Reason != "ineligible" || blocked.Selection.WinnerPath != "" {
		t.Fatalf("blocked selection outcome = %+v, want derived ineligible loser without winner path", blocked)
	}
}

func TestDeriveSkillSelectionOutcomesUsesDeterministicTieBreak(t *testing.T) {
	entries := map[string]skillEntry{
		"planning/a/SKILL.md": {
			Metadata: normalizeSkillMetadata(SkillMetadata{
				SelectionScope: "planning",
				Eligibility:    SkillEligibility{State: skillEligibilityEligible},
				Precedence: SkillPrecedence{
					Tier: skillPrecedenceTierWorkspace,
					Rank: 1,
				},
			}, "skill_bundle", projectionFreshnessSnapshot),
		},
		"planning/b/SKILL.md": {
			Metadata: normalizeSkillMetadata(SkillMetadata{
				SelectionScope: "planning",
				Eligibility:    SkillEligibility{State: skillEligibilityEligible},
				Precedence: SkillPrecedence{
					Tier: skillPrecedenceTierWorkspace,
					Rank: 1,
				},
			}, "skill_bundle", projectionFreshnessSnapshot),
		},
	}

	outcomes := deriveSkillSelectionOutcomes(entries)
	if got := outcomes["planning/a/SKILL.md"]; !got.Selected || got.Selection == nil || got.Selection.Reason != "tie_breaker_path_order" {
		t.Fatalf("planning/a selection outcome = %+v, want lexical tie-break winner", got)
	}
	if got := outcomes["planning/b/SKILL.md"]; got.Selected || got.Selection == nil || got.Selection.Reason != "tie_breaker_path_order" || got.Selection.WinnerPath != "/skills/planning/a/SKILL.md" {
		t.Fatalf("planning/b selection outcome = %+v, want lexical tie-break loser", got)
	}
}

func TestDeriveSkillSelectionOutcomesDoesNotInferScopeFromPath(t *testing.T) {
	entries := map[string]skillEntry{
		"planning/draft-plan/SKILL.md": {
			Metadata: normalizeSkillMetadata(SkillMetadata{
				Eligibility: SkillEligibility{State: skillEligibilityEligible},
				Precedence:  SkillPrecedence{Tier: skillPrecedenceTierWorkspace, Rank: 1},
			}, "skill_bundle", projectionFreshnessSnapshot),
		},
		"planning/fallback/SKILL.md": {
			Metadata: normalizeSkillMetadata(SkillMetadata{
				Eligibility: SkillEligibility{State: skillEligibilityEligible},
				Precedence:  SkillPrecedence{Tier: skillPrecedenceTierBundled, Rank: 90},
			}, "skill_bundle", projectionFreshnessSnapshot),
		},
		"planning/explicit/SKILL.md": {
			Metadata: normalizeSkillMetadata(SkillMetadata{
				Eligibility: SkillEligibility{State: skillEligibilityEligible},
				Precedence:  SkillPrecedence{Tier: skillPrecedenceTierWorkspace, Rank: 5},
				Selected:    true,
			}, "skill_bundle", projectionFreshnessSnapshot),
		},
		"planning/ineligible/SKILL.md": {
			Metadata: normalizeSkillMetadata(SkillMetadata{
				Eligibility: SkillEligibility{State: skillEligibilityIneligible, Reason: "missing_env:PLAN_TOKEN"},
				Precedence:  SkillPrecedence{Tier: skillPrecedenceTierWorkspace, Rank: 0},
				Selected:    true,
			}, "skill_bundle", projectionFreshnessSnapshot),
		},
	}

	outcomes := deriveSkillSelectionOutcomes(entries)
	draft := outcomes["planning/draft-plan/SKILL.md"]
	if draft.Selected || draft.Selection == nil || draft.Selection.Mode != skillSelectionModeExplicit || draft.Selection.State != skillSelectionStateNotSelected || draft.Selection.Reason != skillSelectionReasonExplicitNotSelected {
		t.Fatalf("draft-plan unscoped outcome = %+v, want explicit not-selected without path-inferred competition", draft)
	}
	fallback := outcomes["planning/fallback/SKILL.md"]
	if fallback.Selected || fallback.Selection == nil || fallback.Selection.Mode != skillSelectionModeExplicit || fallback.Selection.State != skillSelectionStateNotSelected || fallback.Selection.Reason != skillSelectionReasonExplicitNotSelected {
		t.Fatalf("fallback unscoped outcome = %+v, want explicit not-selected without path-inferred competition", fallback)
	}
	explicit := outcomes["planning/explicit/SKILL.md"]
	if !explicit.Selected || explicit.Selection == nil || explicit.Selection.Mode != skillSelectionModeExplicit || explicit.Selection.State != skillSelectionStateSelected || explicit.Selection.Reason != "explicit_selected" {
		t.Fatalf("explicit unscoped outcome = %+v, want explicit selected passthrough", explicit)
	}
	ineligible := outcomes["planning/ineligible/SKILL.md"]
	if ineligible.Selected || ineligible.Selection == nil || ineligible.Selection.Mode != skillSelectionModeExplicit || ineligible.Selection.State != skillSelectionStateNotSelected || ineligible.Selection.Reason != "ineligible" {
		t.Fatalf("ineligible unscoped outcome = %+v, want explicit ineligible not-selected", ineligible)
	}
}

func TestDedupeLinesSortsAndTrims(t *testing.T) {
	got := dedupeLines([]string{" wrote:/task_outputs/a ", "", "denied:/kb/x", "wrote:/task_outputs/a", "read-ref:/kb/y"})
	want := []string{"denied:/kb/x", "read-ref:/kb/y", "wrote:/task_outputs/a"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("dedupeLines(...) = %#v, want %#v", got, want)
	}
}

func TestSummarizeTraceFiltersReferenceReadsAndMutations(t *testing.T) {
	got := summarizeTrace(contract.ExecutionTrace{
		ReadPaths:    []string{"/knowledge_base/reference/guide.md", "/resources/checklists/plan.json", "/skills/draft-plan/SKILL.md", "/knowledge_base/other.md"},
		WrittenPaths: []string{"/task_outputs/report.md"},
		DeniedPaths:  []string{"/knowledge_base/reference/guide.md"},
	})
	want := []string{
		"read-ref:/knowledge_base/reference/guide.md",
		"read-resource:/resources/checklists/plan.json",
		"read-skill:/skills/draft-plan/SKILL.md",
		"wrote:/task_outputs/report.md",
		"denied:/knowledge_base/reference/guide.md",
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("summarizeTrace(...) = %#v, want %#v", got, want)
	}
}

func TestKnowledgeMountEmptyAndProjectionError(t *testing.T) {
	adapter := New(Options{})
	knowledgeMount, err := adapter.knowledgeMount(nil)
	if err != nil {
		t.Fatalf("knowledgeMount() error = %v", err)
	}

	raw, err := knowledgeMount.ReadRawContent(context.Background(), "/knowledge_base/reference/empty.md")
	if err != nil {
		t.Fatalf("ReadRawContent(empty.md) error = %v", err)
	}
	if raw != "# Empty\n" {
		t.Fatalf("ReadRawContent(empty.md) = %q, want %q", raw, "# Empty\n")
	}
	indexRaw, err := knowledgeMount.ReadRawContent(context.Background(), "/knowledge_base/reference/_index.json")
	if err != nil {
		t.Fatalf("ReadRawContent(_index.json) error = %v", err)
	}
	records := decodeProjectionRecordsHelper(t, []byte(indexRaw))
	if record := requireProjectionRecordHelper(t, records, "/knowledge_base/reference/empty.md"); record.Freshness != "generated" {
		t.Fatalf("empty projection record = %+v, want generated freshness", record)
	}

	adapter.SetProjectionError(errors.New("projection unavailable"))
	if _, err := adapter.knowledgeMount(nil); err == nil || err.Error() != "projection unavailable" {
		t.Fatalf("knowledgeMount() after projection error = %v, want projection unavailable", err)
	}
}

func TestBuildProjectionIncludesKnowledgeAndMemoryViews(t *testing.T) {
	adapter := New(Options{
		Documents: map[string]string{"guide": "# Guide\nhello\n"},
		DocumentMetadata: map[string]ProjectionMetadata{
			"guide": {Source: "knowledge_sync", Freshness: "snapshot"},
		},
		Resources: map[string]string{"checklists/plan.json": "{\"steps\":1}\n"},
		ResourceMetadata: map[string]ProjectionMetadata{
			"checklists/plan.json": {Source: "workflow_catalog", Freshness: "live"},
		},
	})
	projection, err := adapter.buildProjection(sessionState{
		Observations:   []string{"read-ref:/knowledge_base/reference/guide.md", "read-resource:/resources/checklists/plan.json"},
		Freshness:      "observed",
		ReadRefs:       []string{"/knowledge_base/reference/guide.md"},
		ReadResources:  []string{"/resources/checklists/plan.json"},
		WrittenOutputs: []string{"/task_outputs/plan.txt"},
	})
	if err != nil {
		t.Fatalf("buildProjection(...) error = %v", err)
	}

	if projection.Memory.Freshness != "observed" {
		t.Fatalf("buildProjection(...).Memory.Freshness = %q, want %q", projection.Memory.Freshness, "observed")
	}
	if len(projection.VirtualMounts) != 2 {
		t.Fatalf("buildProjection(...).VirtualMounts = %#v, want knowledge and resource mounts", projection.VirtualMounts)
	}

	guide, err := projection.VirtualMounts[0].ReadRawContent(context.Background(), "/knowledge_base/reference/guide.md")
	if err != nil {
		t.Fatalf("knowledge mount ReadRawContent(...) error = %v", err)
	}
	if guide != "# Guide\nhello\n" {
		t.Fatalf("knowledge mount content = %q, want guide markdown", guide)
	}

	obs, err := projection.Memory.Mount.ReadRawContent(context.Background(), "/memory/observations.md")
	if err != nil {
		t.Fatalf("memory mount observations error = %v", err)
	}
	if !strings.Contains(obs, "read-ref:/knowledge_base/reference/guide.md") || !strings.Contains(obs, "read-resource:/resources/checklists/plan.json") {
		t.Fatalf("memory observations = %q, want reference and resource observations", obs)
	}

	projectionIndex, err := projection.Memory.Mount.ReadRawContent(context.Background(), "/memory/projections.json")
	if err != nil {
		t.Fatalf("memory mount projections error = %v", err)
	}
	view := decodeProjectionViewHelper(t, []byte(projectionIndex))
	if record := requireProjectionRecordHelper(t, view.Documents, "/knowledge_base/reference/guide.md"); record.Source != "knowledge_sync" {
		t.Fatalf("unexpected document projection metadata: %+v", record)
	}
	if record := requireProjectionRecordHelper(t, view.Resources, "/resources/checklists/plan.json"); record.Freshness != "live" {
		t.Fatalf("unexpected resource projection metadata: %+v", record)
	}

	var decoded sessionState
	if err := json.Unmarshal(projection.OpaqueState, &decoded); err != nil {
		t.Fatalf("json.Unmarshal(projection.OpaqueState) error = %v", err)
	}
	if !reflect.DeepEqual(decoded, sessionState{
		Observations:         []string{"read-ref:/knowledge_base/reference/guide.md", "read-resource:/resources/checklists/plan.json"},
		Freshness:            "observed",
		ReadRefs:             []string{"/knowledge_base/reference/guide.md"},
		ReadResources:        []string{"/resources/checklists/plan.json"},
		WrittenOutputs:       []string{"/task_outputs/plan.txt"},
		ProjectionGeneration: 1,
		ControlPlaneEvents:   0,
	}) {
		t.Fatalf("decoded opaque state = %#v, want observed session state", decoded)
	}
}

func TestStateFromSessionRestoresOpaqueAndOverridesFreshness(t *testing.T) {
	adapter := New(Options{})
	raw, err := json.Marshal(sessionState{
		Observations:   []string{"wrote:/task_outputs/report.md"},
		Freshness:      "checkpointed",
		WrittenOutputs: []string{"/task_outputs/report.md"},
	})
	if err != nil {
		t.Fatalf("json.Marshal(...) error = %v", err)
	}

	got := adapter.stateFromSession(contract.Session{
		State: contract.SessionState{
			Opaque: map[string]json.RawMessage{
				adapter.AdapterID(): raw,
			},
		},
	}, "resumed")
	want := sessionState{
		Observations:   []string{"wrote:/task_outputs/report.md"},
		Freshness:      "resumed",
		WrittenOutputs: []string{"/task_outputs/report.md"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("stateFromSession(...) = %#v, want %#v", got, want)
	}
}

func TestControlPlaneUpsertsAndProjectionMetadata(t *testing.T) {
	adapter := New(Options{})
	adapter.UpsertDocument("reports/today", "# Today\n", ProjectionMetadata{Source: "control_plane", Freshness: "live"})
	adapter.UpsertResource("catalog/index.json", "{\"ok\":true}\n", ProjectionMetadata{Source: "control_plane", Freshness: "updated"})
	adapter.UpsertWorkflow(WorkflowSpec{
		ID:              "deliver",
		Title:           "Deliver",
		ResourcePaths:   []string{"catalog/index.json"},
		ExpectedOutputs: []string{"/task_outputs/final.md"},
	})

	projection, err := adapter.buildProjection(sessionState{Freshness: "created"})
	if err != nil {
		t.Fatalf("buildProjection(control plane) error = %v", err)
	}
	docIndexRaw, err := projection.VirtualMounts[0].ReadRawContent(context.Background(), "/knowledge_base/reference/_index.json")
	if err != nil {
		t.Fatalf("read document index error = %v", err)
	}
	docRecords := decodeProjectionRecordsHelper(t, []byte(docIndexRaw))
	if record := requireProjectionRecordHelper(t, docRecords, "/knowledge_base/reference/reports/today.md"); record.Source != "control_plane" || record.Freshness != "live" {
		t.Fatalf("unexpected document control-plane metadata: %+v", record)
	}
	resourceIndexRaw, err := projection.VirtualMounts[1].ReadRawContent(context.Background(), "/resources/_index.json")
	if err != nil {
		t.Fatalf("read resource index error = %v", err)
	}
	resourceRecords := decodeProjectionRecordsHelper(t, []byte(resourceIndexRaw))
	if record := requireProjectionRecordHelper(t, resourceRecords, "/resources/catalog/index.json"); record.Source != "control_plane" || record.Freshness != "updated" {
		t.Fatalf("unexpected resource control-plane metadata: %+v", record)
	}
	workflowsRaw, err := projection.Memory.Mount.ReadRawContent(context.Background(), "/memory/workflows.json")
	if err != nil {
		t.Fatalf("read workflows json error = %v", err)
	}
	workflows := decodeWorkflowViewsHelper(t, []byte(workflowsRaw))
	if got := workflowStatusByID(workflows, "deliver"); got != "pending" {
		t.Fatalf("deliver workflow status = %q, want pending", got)
	}
}

func TestSkillControlPlaneUpsertAndRemoveRecomputeSelection(t *testing.T) {
	adapter := New(Options{
		Skills: map[string]string{
			"planning/fallback": "# Fallback Planner\n",
		},
		SkillMetadata: map[string]SkillMetadata{
			"planning/fallback": {
				Source:         "bundled_catalog",
				Freshness:      projectionFreshnessSnapshot,
				SelectionScope: "planning/default",
				Eligibility:    SkillEligibility{State: skillEligibilityEligible},
				Precedence:     SkillPrecedence{Tier: skillPrecedenceTierBundled, Rank: 80},
			},
		},
	})

	initialProjection, err := adapter.buildProjection(sessionState{Freshness: "created"})
	if err != nil {
		t.Fatalf("buildProjection(initial skill control plane) error = %v", err)
	}
	initialIndex := decodeProjectionViewHelper(t, []byte(`{"skills":`+mustReadFromMounts(t, initialProjection.VirtualMounts, "/skills/_index.json")+`}`))
	fallback := requireProjectionRecordHelper(t, initialIndex.Skills, "/skills/planning/fallback/SKILL.md")
	if !fallback.Selected || fallback.Selection == nil || fallback.Selection.Scope != "planning/default" || fallback.Selection.Mode != skillSelectionModeDerived {
		t.Fatalf("initial fallback skill = %+v, want selected derived fallback", fallback)
	}

	adapter.UpsertSkill("planning/primary", "# Primary Planner\n", SkillMetadata{
		Source:         "workspace_catalog",
		Freshness:      projectionFreshnessLive,
		SelectionScope: "planning/default",
		Eligibility:    SkillEligibility{State: skillEligibilityEligible},
		Precedence:     SkillPrecedence{Tier: skillPrecedenceTierWorkspace, Rank: 1},
	})

	updatedProjection, err := adapter.buildProjection(sessionState{Freshness: "observed"})
	if err != nil {
		t.Fatalf("buildProjection(updated skill control plane) error = %v", err)
	}
	updatedIndex := decodeProjectionViewHelper(t, []byte(`{"skills":`+mustReadFromMounts(t, updatedProjection.VirtualMounts, "/skills/_index.json")+`}`))
	primary := requireProjectionRecordHelper(t, updatedIndex.Skills, "/skills/planning/primary/SKILL.md")
	if !primary.Selected || primary.Source != "workspace_catalog" || primary.Freshness != projectionFreshnessLive {
		t.Fatalf("primary skill after upsert = %+v, want selected workspace/live skill", primary)
	}
	fallback = requireProjectionRecordHelper(t, updatedIndex.Skills, "/skills/planning/fallback/SKILL.md")
	if fallback.Selected || fallback.Selection == nil || fallback.Selection.WinnerPath != "/skills/planning/primary/SKILL.md" {
		t.Fatalf("fallback skill after upsert = %+v, want loser pointing at primary", fallback)
	}

	adapter.RemoveSkill("planning/primary")
	removedProjection, err := adapter.buildProjection(sessionState{Freshness: "resumed"})
	if err != nil {
		t.Fatalf("buildProjection(removed skill control plane) error = %v", err)
	}
	removedIndex := decodeProjectionViewHelper(t, []byte(`{"skills":`+mustReadFromMounts(t, removedProjection.VirtualMounts, "/skills/_index.json")+`}`))
	if len(removedIndex.Skills) != 1 {
		t.Fatalf("skills after remove = %+v, want one fallback skill", removedIndex.Skills)
	}
	fallback = requireProjectionRecordHelper(t, removedIndex.Skills, "/skills/planning/fallback/SKILL.md")
	if !fallback.Selected || fallback.Selection == nil || fallback.Selection.WinnerPath != "" {
		t.Fatalf("fallback after remove = %+v, want reselected singleton", fallback)
	}
}

func TestUpdateSkillPreservesSelectionMetadataWhileRefreshingControlPlaneSource(t *testing.T) {
	adapter := New(Options{
		Skills: map[string]string{
			"planning/draft-plan": "# Draft Planner\n",
		},
		SkillMetadata: map[string]SkillMetadata{
			"planning/draft-plan": {
				Source:         "workspace_catalog",
				Freshness:      projectionFreshnessLive,
				SelectionScope: "planning/default",
				Eligibility:    SkillEligibility{State: skillEligibilityEligible},
				Precedence:     SkillPrecedence{Tier: skillPrecedenceTierWorkspace, Rank: 1},
			},
		},
	})

	adapter.UpdateSkill("planning/draft-plan", "# Draft Planner\nupdated\n")
	projection, err := adapter.buildProjection(sessionState{Freshness: "observed"})
	if err != nil {
		t.Fatalf("buildProjection(update skill) error = %v", err)
	}
	skillRaw := mustReadFromMounts(t, projection.VirtualMounts, "/skills/planning/draft-plan/SKILL.md")
	if !strings.Contains(skillRaw, "updated") {
		t.Fatalf("updated skill content = %q, want updated content", skillRaw)
	}
	index := decodeProjectionViewHelper(t, []byte(`{"skills":`+mustReadFromMounts(t, projection.VirtualMounts, "/skills/_index.json")+`}`))
	record := requireProjectionRecordHelper(t, index.Skills, "/skills/planning/draft-plan/SKILL.md")
	if record.Source != "control_plane" || record.Freshness != projectionFreshnessUpdated {
		t.Fatalf("updated skill metadata = %+v, want control_plane/updated", record)
	}
	if record.Precedence == nil || record.Precedence.Tier != skillPrecedenceTierWorkspace || record.Precedence.Rank != 1 {
		t.Fatalf("updated skill precedence = %+v, want preserved workspace/1", record.Precedence)
	}
	if record.Selection == nil || record.Selection.Scope != "planning/default" || !record.Selected {
		t.Fatalf("updated skill selection = %+v, want preserved selected planning/default state", record)
	}
}

func TestNewNormalizesProjectionMetadataKeys(t *testing.T) {
	adapter := New(Options{
		Documents: map[string]string{
			" guide ": "# Guide\n",
		},
		DocumentMetadata: map[string]ProjectionMetadata{
			"guide.md": {Source: "manual_seed", Freshness: "live"},
		},
		Resources: map[string]string{
			" checklists/plan.json ": "{\"ok\":true}\n",
		},
		ResourceMetadata: map[string]ProjectionMetadata{
			"checklists/plan.json": {Source: "catalog_seed", Freshness: "snapshot"},
		},
	})

	docs := adapter.documentRecords()
	doc := requireProjectionRecordHelper(t, docs, "/knowledge_base/reference/guide.md")
	if doc.Source != "manual_seed" || doc.Freshness != "live" {
		t.Fatalf("normalized document metadata = %+v, want manual_seed/live", doc)
	}

	resources := adapter.resourceRecords()
	resource := requireProjectionRecordHelper(t, resources, "/resources/checklists/plan.json")
	if resource.Source != "catalog_seed" || resource.Freshness != "snapshot" {
		t.Fatalf("normalized resource metadata = %+v, want catalog_seed/snapshot", resource)
	}
}

func TestNewNormalizesSkillMetadataKeys(t *testing.T) {
	adapter := New(Options{
		Skills: map[string]string{
			" lint/check ": "# Lint check\n",
		},
		SkillMetadata: map[string]SkillMetadata{
			"lint/check/SKILL.md": {
				Source:    "workspace_registry",
				Freshness: projectionFreshnessLive,
				Eligibility: SkillEligibility{
					State: skillEligibilityEligible,
				},
				Precedence: SkillPrecedence{
					Tier: skillPrecedenceTierWorkspace,
					Rank: 2,
				},
			},
		},
	})

	records := adapter.skillRecords()
	record := requireProjectionRecordHelper(t, records, "/skills/lint/check/SKILL.md")
	if record.Source != "workspace_registry" || record.Freshness != projectionFreshnessLive {
		t.Fatalf("normalized skill metadata = %+v, want workspace_registry/live", record)
	}
	if record.Eligibility == nil || record.Eligibility.State != skillEligibilityEligible {
		t.Fatalf("normalized skill eligibility = %+v, want eligible", record.Eligibility)
	}
	if record.Precedence == nil || record.Precedence.Tier != skillPrecedenceTierWorkspace || record.Precedence.Rank != 2 {
		t.Fatalf("normalized skill precedence = %+v, want workspace/2", record.Precedence)
	}
}

func TestNormalizeProjectionFreshnessRejectsUnknownValues(t *testing.T) {
	tests := []struct {
		raw  string
		want string
	}{
		{raw: "live", want: "live"},
		{raw: " stale ", want: "stale"},
		{raw: "unexpected", want: ""},
		{raw: "", want: ""},
	}

	for _, tt := range tests {
		if got := normalizeProjectionFreshness(tt.raw); got != tt.want {
			t.Fatalf("normalizeProjectionFreshness(%q) = %q, want %q", tt.raw, got, tt.want)
		}
	}
}

func TestNormalizeProjectionMaterializationDefaultsAndValidation(t *testing.T) {
	got := normalizeProjectionMaterialization(ProjectionMaterialization{}, projectionMaterializationMaterialized)
	if got.State != projectionMaterializationMaterialized || got.Reason != "" {
		t.Fatalf("normalizeProjectionMaterialization(default) = %#v, want materialized state with empty reason", got)
	}

	got = normalizeProjectionMaterialization(ProjectionMaterialization{
		State:  " partial ",
		Reason: " upstream-truncated ",
	}, projectionMaterializationMaterialized)
	if got.State != projectionMaterializationPartial || got.Reason != "upstream-truncated" {
		t.Fatalf("normalizeProjectionMaterialization(partial) = %#v, want partial with normalized reason", got)
	}

	got = normalizeProjectionMaterialization(ProjectionMaterialization{
		State:  "unknown",
		Reason: "ignored",
	}, projectionMaterializationMaterialized)
	if got.State != projectionMaterializationMaterialized || got.Reason != "" {
		t.Fatalf("normalizeProjectionMaterialization(invalid) = %#v, want materialized fallback with cleared reason", got)
	}
}

func TestNormalizeCuratedEntryRequiresStableIDAndSourcePaths(t *testing.T) {
	if _, ok := normalizeCuratedEntry(CuratedEntry{
		ID:          "",
		SourcePaths: []string{"/memory/observations.md"},
	}, "control_plane"); ok {
		t.Fatalf("normalizeCuratedEntry(...) accepted empty id")
	}
	if _, ok := normalizeCuratedEntry(CuratedEntry{
		ID:          "decision/read-only-memory",
		SourcePaths: nil,
	}, "control_plane"); ok {
		t.Fatalf("normalizeCuratedEntry(...) accepted empty source paths")
	}

	got, ok := normalizeCuratedEntry(CuratedEntry{
		ID:          " decision/read-only-memory ",
		Title:       "",
		Summary:     " keep control-plane explicit ",
		Content:     " curated entry content ",
		SourcePaths: []string{"memory/projections.json", "/memory/observations.md", "memory/projections.json"},
	}, "control_plane")
	if !ok {
		t.Fatalf("normalizeCuratedEntry(...) rejected valid curated entry")
	}
	if got.ID != "decision/read-only-memory" || got.Title != "decision/read-only-memory" {
		t.Fatalf("normalizeCuratedEntry(...) id/title = %+v, want normalized id with title fallback", got)
	}
	if got.Source != "control_plane" || got.Revision != 1 {
		t.Fatalf("normalizeCuratedEntry(...) source/revision = %+v, want control_plane/rev=1", got)
	}
	if !reflect.DeepEqual(got.SourcePaths, []string{"/memory/observations.md", "/memory/projections.json"}) {
		t.Fatalf("normalizeCuratedEntry(...) source paths = %#v, want normalized deduped absolute paths", got.SourcePaths)
	}
}

func TestBuildProjectionIncludesSkillProjectionMetadata(t *testing.T) {
	adapter := New(Options{
		Skills: map[string]string{
			" triage/incident ": "# Incident Triage\n",
		},
		SkillMetadata: map[string]SkillMetadata{
			"triage/incident/SKILL.md": {
				Source:    "workspace_registry",
				Freshness: projectionFreshnessLive,
				Eligibility: SkillEligibility{
					State:  skillEligibilityIneligible,
					Reason: "missing:jq",
				},
				Precedence: SkillPrecedence{
					Tier: skillPrecedenceTierWorkspace,
					Rank: 3,
				},
			},
		},
	})

	projection, err := adapter.buildProjection(sessionState{Freshness: "created"})
	if err != nil {
		t.Fatalf("buildProjection(skills) error = %v", err)
	}
	if len(projection.VirtualMounts) != 2 {
		t.Fatalf("buildProjection(skills).VirtualMounts = %#v, want knowledge and skills mounts", projection.VirtualMounts)
	}

	skillContent := mustReadFromMounts(t, projection.VirtualMounts, "/skills/triage/incident/SKILL.md")
	if skillContent != "# Incident Triage\n" {
		t.Fatalf("skill content = %q, want projected markdown", skillContent)
	}

	indexRaw := mustReadFromMounts(t, projection.VirtualMounts, "/skills/_index.json")
	indexRecords := decodeProjectionRecordsHelper(t, []byte(indexRaw))
	record := requireProjectionRecordHelper(t, indexRecords, "/skills/triage/incident/SKILL.md")
	if record.Source != "workspace_registry" || record.Freshness != projectionFreshnessLive {
		t.Fatalf("unexpected skill source/freshness metadata: %+v", record)
	}
	if record.Eligibility == nil || record.Eligibility.State != skillEligibilityIneligible || record.Eligibility.Reason != "missing:jq" {
		t.Fatalf("unexpected skill eligibility metadata: %+v", record.Eligibility)
	}
	if record.Precedence == nil || record.Precedence.Tier != skillPrecedenceTierWorkspace || record.Precedence.Rank != 3 {
		t.Fatalf("unexpected skill precedence metadata: %+v", record.Precedence)
	}

	projectionsRaw, err := projection.Memory.Mount.ReadRawContent(context.Background(), "/memory/projections.json")
	if err != nil {
		t.Fatalf("memory mount projections error = %v", err)
	}
	view := decodeProjectionViewHelper(t, []byte(projectionsRaw))
	record = requireProjectionRecordHelper(t, view.Skills, "/skills/triage/incident/SKILL.md")
	if record.Eligibility == nil || record.Eligibility.State != skillEligibilityIneligible {
		t.Fatalf("memory projection view skills = %+v, want ineligible state", record)
	}
	summaryRaw, err := projection.Memory.Mount.ReadRawContent(context.Background(), "/memory/summary.md")
	if err != nil {
		t.Fatalf("memory mount summary error = %v", err)
	}
	if !strings.Contains(summaryRaw, "projections.skills: 1") {
		t.Fatalf("memory summary = %q, want skills projection count", summaryRaw)
	}
}

func TestControlPlaneUpsertSkillAddsSelection(t *testing.T) {
	adapter := New(Options{})
	adapter.UpsertSkill("planning/winner", "# Winner\n", SkillMetadata{
		SelectionScope: "planning",
		Eligibility:    SkillEligibility{State: skillEligibilityEligible},
		Precedence:     SkillPrecedence{Tier: skillPrecedenceTierWorkspace, Rank: 0},
	})
	records := adapter.skillRecords()
	record := requireProjectionRecordHelper(t, records, "/skills/planning/winner/SKILL.md")
	if !record.Selected || record.Selection == nil || record.Selection.Mode != skillSelectionModeDerived || record.Selection.Scope != "planning" {
		t.Fatalf("derived selection = %+v, want derived planning winner", record.Selection)
	}
	if record.Selection.Reason != skillSelectionReasonHighestPrecedence {
		t.Fatalf("selection reason = %q, want highest_precedence", record.Selection.Reason)
	}
	if record.Source != "control_plane" || record.Freshness != projectionFreshnessUpdated {
		t.Fatalf("control-plane metadata = %+v, want control_plane/updated", record)
	}
}

func TestControlPlaneUpdateSkillRefreshesContentAndMetadata(t *testing.T) {
	adapter := New(Options{})
	adapter.UpsertSkill("strategy/initiate", "# Initial\n", SkillMetadata{
		SelectionScope: "strategy",
		Eligibility:    SkillEligibility{State: skillEligibilityEligible},
		Precedence:     SkillPrecedence{Tier: skillPrecedenceTierWorkspace, Rank: 1},
	})
	adapter.UpdateSkill("strategy/initiate", "# Updated\n")
	projection, err := adapter.buildProjection(sessionState{Freshness: "observed"})
	if err != nil {
		t.Fatalf("buildProjection(update) error = %v", err)
	}
	if content := mustReadFromMounts(t, projection.VirtualMounts, "/skills/strategy/initiate/SKILL.md"); content != "# Updated\n" {
		t.Fatalf("updated content = %q, want updated markdown", content)
	}
	indexRaw := mustReadFromMounts(t, projection.VirtualMounts, "/skills/_index.json")
	records := decodeProjectionRecordsHelper(t, []byte(indexRaw))
	record := requireProjectionRecordHelper(t, records, "/skills/strategy/initiate/SKILL.md")
	if record.Source != "control_plane" || record.Freshness != projectionFreshnessUpdated {
		t.Fatalf("updated skill metadata = %+v, want control_plane/updated", record)
	}
}

func TestControlPlaneRemoveSkillRecomputesSelection(t *testing.T) {
	adapter := New(Options{})
	adapter.UpsertSkill("planning/alpha", "# Alpha\n", SkillMetadata{
		SelectionScope: "planning",
		Eligibility:    SkillEligibility{State: skillEligibilityEligible},
		Precedence:     SkillPrecedence{Tier: skillPrecedenceTierWorkspace, Rank: 0},
	})
	adapter.UpsertSkill("planning/beta", "# Beta\n", SkillMetadata{
		SelectionScope: "planning",
		Eligibility:    SkillEligibility{State: skillEligibilityEligible},
		Precedence:     SkillPrecedence{Tier: skillPrecedenceTierWorkspace, Rank: 1},
	})
	records := adapter.skillRecords()
	alpha := requireProjectionRecordHelper(t, records, "/skills/planning/alpha/SKILL.md")
	if !alpha.Selected {
		t.Fatalf("alpha selection = %+v, want selected", alpha.Selection)
	}
	adapter.RemoveSkill("planning/alpha")
	records = adapter.skillRecords()
	if len(records) != 1 {
		t.Fatalf("remaining skill records = %+v, want single beta", records)
	}
	beta := requireProjectionRecordHelper(t, records, "/skills/planning/beta/SKILL.md")
	if !beta.Selected {
		t.Fatalf("beta selection after remove = %+v, want selected", beta.Selection)
	}
	if beta.Selection == nil || beta.Selection.Reason != skillSelectionReasonHighestPrecedence {
		t.Fatalf("beta selection reason = %+v, want highest_precedence", beta.Selection)
	}
}

func TestSkillControlPlaneAuditVisibilityAndSummary(t *testing.T) {
	adapter := New(Options{
		Skills: map[string]string{
			"planning/fallback": "# Fallback\n",
		},
		SkillMetadata: map[string]SkillMetadata{
			"planning/fallback": {
				SelectionScope: "planning/default",
				Eligibility:    SkillEligibility{State: skillEligibilityEligible},
				Precedence:     SkillPrecedence{Tier: skillPrecedenceTierBundled, Rank: 80},
			},
		},
	})

	adapter.UpsertSkill("planning/primary", "# Primary\n", SkillMetadata{
		SelectionScope: "planning/default",
		Eligibility:    SkillEligibility{State: skillEligibilityEligible},
		Precedence:     SkillPrecedence{Tier: skillPrecedenceTierWorkspace, Rank: 1},
	})
	if adapter.projectionSeq != 0 {
		t.Fatalf("projectionSeq before build = %d, want 0", adapter.projectionSeq)
	}
	if len(adapter.controlPlaneEvents) != 1 {
		t.Fatalf("control-plane events before build = %+v, want one pending event", adapter.controlPlaneEvents)
	}
	event := adapter.controlPlaneEvents[0]
	if event.Op != controlPlaneEventKindSkillAdded || event.VisibleFromGeneration != 1 || event.VisibleAfter != controlPlaneVisibilityNextProjection {
		t.Fatalf("pending control-plane event = %+v, want skill_added visible from generation 1", event)
	}

	projection, err := adapter.buildProjection(sessionState{Freshness: "observed"})
	if err != nil {
		t.Fatalf("buildProjection(audit) error = %v", err)
	}
	status := decodeReferenceStateHelper(t, mustReadFromMounts(t, []contract.VirtualMount{projection.Memory.Mount}, "/memory/status.json"))
	if status.ProjectionGeneration != 1 || status.ControlPlaneEvents != 1 || status.LastControlPlaneKind != controlPlaneEventKindSkillAdded {
		t.Fatalf("status after build = %+v, want generation=1 event_count=1 last=skill_added", status)
	}
	auditRaw, err := projection.Memory.Mount.ReadRawContent(context.Background(), "/memory/skills_audit.json")
	if err != nil {
		t.Fatalf("read skills audit json error = %v", err)
	}
	audit := decodeControlPlaneEventViewsHelper(t, []byte(auditRaw))
	if len(audit) != 1 {
		t.Fatalf("skills audit events = %+v, want one event", audit)
	}
	if audit[0].Visibility != "visible" || audit[0].WinnerBefore != "/skills/planning/fallback/SKILL.md" || audit[0].WinnerAfter != "/skills/planning/primary/SKILL.md" || !audit[0].SelectedAfter {
		t.Fatalf("skills audit event = %+v, want visible winner flip to primary", audit[0])
	}
	summaryRaw, err := projection.Memory.Mount.ReadRawContent(context.Background(), "/memory/skills_audit.md")
	if err != nil {
		t.Fatalf("read skills audit summary error = %v", err)
	}
	if !strings.Contains(summaryRaw, "projection_generation: 1") || !strings.Contains(summaryRaw, "skill_added /skills/planning/primary/SKILL.md") {
		t.Fatalf("skills audit summary = %q, want projection generation and skill_added line", summaryRaw)
	}
}

func TestSkillControlPlaneAuditTracksUpdateAndRemove(t *testing.T) {
	adapter := New(Options{
		Skills: map[string]string{
			"planning/primary":  "# Primary\n",
			"planning/fallback": "# Fallback\n",
		},
		SkillMetadata: map[string]SkillMetadata{
			"planning/primary": {
				SelectionScope: "planning/default",
				Eligibility:    SkillEligibility{State: skillEligibilityEligible},
				Precedence:     SkillPrecedence{Tier: skillPrecedenceTierWorkspace, Rank: 1},
			},
			"planning/fallback": {
				SelectionScope: "planning/default",
				Eligibility:    SkillEligibility{State: skillEligibilityEligible},
				Precedence:     SkillPrecedence{Tier: skillPrecedenceTierBundled, Rank: 80},
			},
		},
	})

	adapter.UpdateSkill("planning/primary", "# Primary\nupdated\n")
	adapter.RemoveSkill("planning/primary")
	projection, err := adapter.buildProjection(sessionState{Freshness: "resumed"})
	if err != nil {
		t.Fatalf("buildProjection(update/remove audit) error = %v", err)
	}
	auditRaw, err := projection.Memory.Mount.ReadRawContent(context.Background(), "/memory/skills_audit.json")
	if err != nil {
		t.Fatalf("read skills audit json after update/remove error = %v", err)
	}
	audit := decodeControlPlaneEventViewsHelper(t, []byte(auditRaw))
	if len(audit) != 2 {
		t.Fatalf("skills audit events after update/remove = %+v, want two events", audit)
	}
	if audit[0].Op != controlPlaneEventKindSkillUpdated || audit[0].ReasonAfter == "" || !audit[0].SelectedAfter {
		t.Fatalf("update audit event = %+v, want selected update event", audit[0])
	}
	if audit[1].Op != controlPlaneEventKindSkillRemoved || !audit[1].SelectedBefore || audit[1].WinnerBefore != "/skills/planning/primary/SKILL.md" || audit[1].WinnerAfter != "/skills/planning/fallback/SKILL.md" {
		t.Fatalf("remove audit event = %+v, want fallback reselection after removal", audit[1])
	}
}

func TestRefreshAndInvalidateProjectionMetadata(t *testing.T) {
	adapter := New(Options{
		Documents: map[string]string{
			"guide.md": "# Guide\n",
		},
		DocumentMetadata: map[string]ProjectionMetadata{
			"guide.md": {Source: "knowledge_sync", Freshness: "snapshot"},
		},
		Resources: map[string]string{
			"catalog/index.json": "{\"ok\":true}\n",
		},
		ResourceMetadata: map[string]ProjectionMetadata{
			"catalog/index.json": {Source: "workflow_catalog", Freshness: "live"},
		},
	})

	adapter.InvalidateDocument("guide.md")
	adapter.InvalidateResource("catalog/index.json")
	staleProjection, err := adapter.buildProjection(sessionState{Freshness: "observed"})
	if err != nil {
		t.Fatalf("buildProjection(stale) error = %v", err)
	}

	staleIndexRaw, err := staleProjection.Memory.Mount.ReadRawContent(context.Background(), "/memory/projections.json")
	if err != nil {
		t.Fatalf("read stale projections error = %v", err)
	}
	staleView := decodeProjectionViewHelper(t, []byte(staleIndexRaw))
	if record := requireProjectionRecordHelper(t, staleView.Documents, "/knowledge_base/reference/guide.md"); record.Source != "knowledge_sync" || record.Freshness != "stale" {
		t.Fatalf("stale document projection = %+v, want knowledge_sync/stale", record)
	}
	if record := requireProjectionRecordHelper(t, staleView.Resources, "/resources/catalog/index.json"); record.Source != "workflow_catalog" || record.Freshness != "stale" {
		t.Fatalf("stale resource projection = %+v, want workflow_catalog/stale", record)
	}
	staleSummary, err := staleProjection.Memory.Mount.ReadRawContent(context.Background(), "/memory/summary.md")
	if err != nil {
		t.Fatalf("read stale summary error = %v", err)
	}
	if !strings.Contains(staleSummary, "- stale: 2") {
		t.Fatalf("stale summary = %q, want stale count", staleSummary)
	}
	staleResourceIndex, err := staleProjection.VirtualMounts[1].ReadRawContent(context.Background(), "/resources/_index.json")
	if err != nil {
		t.Fatalf("read stale resource index error = %v", err)
	}
	staleResourceRecords := decodeProjectionRecordsHelper(t, []byte(staleResourceIndex))
	if record := requireProjectionRecordHelper(t, staleResourceRecords, "/resources/catalog/index.json"); record.Freshness != "stale" {
		t.Fatalf("stale resource index record = %+v, want stale freshness", record)
	}

	adapter.RefreshDocument("guide.md", "# Guide\nfresh\n", ProjectionMetadata{})
	adapter.RefreshResource("catalog/index.json", "{\"ok\":false}\n", ProjectionMetadata{Source: "catalog_refresh"})
	adapter.RefreshDocument("missing.md", "# Missing\n", ProjectionMetadata{})
	adapter.RefreshResource("missing.json", "{\"missing\":true}\n", ProjectionMetadata{})
	adapter.InvalidateDocument("missing.md")
	adapter.InvalidateResource("missing.json")

	liveProjection, err := adapter.buildProjection(sessionState{Freshness: "resumed"})
	if err != nil {
		t.Fatalf("buildProjection(live) error = %v", err)
	}
	liveIndexRaw, err := liveProjection.Memory.Mount.ReadRawContent(context.Background(), "/memory/projections.json")
	if err != nil {
		t.Fatalf("read live projections error = %v", err)
	}
	liveView := decodeProjectionViewHelper(t, []byte(liveIndexRaw))
	if record := requireProjectionRecordHelper(t, liveView.Documents, "/knowledge_base/reference/guide.md"); record.Source != "knowledge_sync" || record.Freshness != "live" {
		t.Fatalf("live document projection = %+v, want knowledge_sync/live", record)
	}
	if record := requireProjectionRecordHelper(t, liveView.Resources, "/resources/catalog/index.json"); record.Source != "catalog_refresh" || record.Freshness != "live" {
		t.Fatalf("live resource projection = %+v, want catalog_refresh/live", record)
	}
	guideRaw, err := liveProjection.VirtualMounts[0].ReadRawContent(context.Background(), "/knowledge_base/reference/guide.md")
	if err != nil {
		t.Fatalf("read refreshed guide error = %v", err)
	}
	if guideRaw != "# Guide\nfresh\n" {
		t.Fatalf("refreshed guide = %q, want %q", guideRaw, "# Guide\nfresh\n")
	}
	liveSummary, err := liveProjection.Memory.Mount.ReadRawContent(context.Background(), "/memory/summary.md")
	if err != nil {
		t.Fatalf("read live summary error = %v", err)
	}
	if !strings.Contains(liveSummary, "- live: 2") {
		t.Fatalf("live summary = %q, want live count", liveSummary)
	}
	liveResourceIndex, err := liveProjection.VirtualMounts[1].ReadRawContent(context.Background(), "/resources/_index.json")
	if err != nil {
		t.Fatalf("read live resource index error = %v", err)
	}
	liveResourceRecords := decodeProjectionRecordsHelper(t, []byte(liveResourceIndex))
	if record := requireProjectionRecordHelper(t, liveResourceRecords, "/resources/catalog/index.json"); record.Source != "catalog_refresh" || record.Freshness != "live" {
		t.Fatalf("live resource index record = %+v, want catalog_refresh/live", record)
	}
	if len(liveView.Documents) != 1 {
		t.Fatalf("live document projections = %+v, want missing refresh to avoid creating a new projection", liveView.Documents)
	}
	if len(liveView.Resources) != 1 {
		t.Fatalf("live resource projections = %+v, want missing refresh to avoid creating a new projection", liveView.Resources)
	}
}

func TestProjectionMaterializationFailureAndPartialStatesAreAdapterLocalTruth(t *testing.T) {
	adapter := New(Options{
		Documents: map[string]string{
			"guide.md": "# Guide\nsnapshot\n",
		},
		DocumentMetadata: map[string]ProjectionMetadata{
			"guide.md": {Source: "knowledge_sync", Freshness: "snapshot"},
		},
		Resources: map[string]string{
			"catalog/index.json": "{\"ok\":true}\n",
		},
		ResourceMetadata: map[string]ProjectionMetadata{
			"catalog/index.json": {Source: "workflow_catalog", Freshness: "live"},
		},
	})

	adapter.SetDocumentMaterialization("guide.md", "failed", "source-timeout")
	adapter.SetResourceMaterialization("catalog/index.json", "partial", "truncated-payload")

	projection, err := adapter.buildProjection(sessionState{Freshness: "observed"})
	if err != nil {
		t.Fatalf("buildProjection(materialization truth) error = %v", err)
	}

	if _, err := projection.VirtualMounts[0].ReadRawContent(context.Background(), "/knowledge_base/reference/guide.md"); err == nil {
		t.Fatalf("failed materialization document unexpectedly readable")
	}
	guideIndexRaw, err := projection.VirtualMounts[0].ReadRawContent(context.Background(), "/knowledge_base/reference/_index.json")
	if err != nil {
		t.Fatalf("read guide index error = %v", err)
	}
	guideIndex := decodeProjectionRecordsHelper(t, []byte(guideIndexRaw))
	guide := requireProjectionRecordHelper(t, guideIndex, "/knowledge_base/reference/guide.md")
	if guide.Materialization.State != projectionMaterializationFailed || guide.Materialization.Reason != "source-timeout" {
		t.Fatalf("guide materialization = %+v, want failed/source-timeout", guide.Materialization)
	}

	resourceRaw, err := projection.VirtualMounts[1].ReadRawContent(context.Background(), "/resources/catalog/index.json")
	if err != nil {
		t.Fatalf("read partial resource content error = %v", err)
	}
	if resourceRaw != "{\"ok\":true}\n" {
		t.Fatalf("partial resource content = %q, want persisted content", resourceRaw)
	}
	resourceIndexRaw, err := projection.VirtualMounts[1].ReadRawContent(context.Background(), "/resources/_index.json")
	if err != nil {
		t.Fatalf("read resource index error = %v", err)
	}
	resourceIndex := decodeProjectionRecordsHelper(t, []byte(resourceIndexRaw))
	resource := requireProjectionRecordHelper(t, resourceIndex, "/resources/catalog/index.json")
	if resource.Materialization.State != projectionMaterializationPartial || resource.Materialization.Reason != "truncated-payload" {
		t.Fatalf("resource materialization = %+v, want partial/truncated-payload", resource.Materialization)
	}

	projectionsRaw, err := projection.Memory.Mount.ReadRawContent(context.Background(), "/memory/projections.json")
	if err != nil {
		t.Fatalf("read memory projections error = %v", err)
	}
	view := decodeProjectionViewHelper(t, []byte(projectionsRaw))
	guide = requireProjectionRecordHelper(t, view.Documents, "/knowledge_base/reference/guide.md")
	resource = requireProjectionRecordHelper(t, view.Resources, "/resources/catalog/index.json")
	if guide.Materialization.State != projectionMaterializationFailed {
		t.Fatalf("memory guide materialization = %+v, want failed", guide.Materialization)
	}
	if resource.Materialization.State != projectionMaterializationPartial {
		t.Fatalf("memory resource materialization = %+v, want partial", resource.Materialization)
	}

	summaryRaw, err := projection.Memory.Mount.ReadRawContent(context.Background(), "/memory/summary.md")
	if err != nil {
		t.Fatalf("read memory summary error = %v", err)
	}
	if !strings.Contains(summaryRaw, "- failed: 1") || !strings.Contains(summaryRaw, "- partial: 1") {
		t.Fatalf("memory summary = %q, want materialization state counters", summaryRaw)
	}
}

func TestWorkflowControlPlaneOverridesTraceStatus(t *testing.T) {
	adapter := New(Options{
		Workflows: []WorkflowSpec{
			{
				ID:              "deliver",
				Title:           "Deliver",
				ResourcePaths:   []string{"/resources/catalog/index.json"},
				ExpectedOutputs: []string{"/task_outputs/final.md"},
			},
		},
	})
	state := sessionState{
		Freshness:      "observed",
		ReadResources:  []string{"/resources/catalog/index.json"},
		WrittenOutputs: []string{"/task_outputs/final.md"},
	}

	adapter.SetWorkflowStatus("deliver", "blocked", "awaiting approval")
	projection, err := adapter.buildProjection(state)
	if err != nil {
		t.Fatalf("buildProjection(workflow override) error = %v", err)
	}
	workflowsRaw, err := projection.Memory.Mount.ReadRawContent(context.Background(), "/memory/workflows.json")
	if err != nil {
		t.Fatalf("read workflow override json error = %v", err)
	}
	workflows := decodeWorkflowViewsHelper(t, []byte(workflowsRaw))
	deliver := requireWorkflowViewHelper(t, workflows, "deliver")
	if deliver.Status != "blocked" || deliver.StatusSource != "control_plane" || deliver.StatusReason != "awaiting approval" {
		t.Fatalf("workflow override = %+v, want blocked/control_plane/awaiting approval", deliver)
	}
	if !containsLine(deliver.Evidence, "/task_outputs/final.md") {
		t.Fatalf("workflow override evidence = %+v, want preserved trace evidence", deliver.Evidence)
	}
	workflowsSummary, err := projection.Memory.Mount.ReadRawContent(context.Background(), "/memory/workflows.md")
	if err != nil {
		t.Fatalf("read workflow override summary error = %v", err)
	}
	if !strings.Contains(workflowsSummary, "source: control_plane") || !strings.Contains(workflowsSummary, "reason: awaiting approval") {
		t.Fatalf("workflow override summary = %q, want source and reason", workflowsSummary)
	}

	adapter.ClearWorkflowStatus("deliver")
	projection, err = adapter.buildProjection(state)
	if err != nil {
		t.Fatalf("buildProjection(clear workflow override) error = %v", err)
	}
	workflowsRaw, err = projection.Memory.Mount.ReadRawContent(context.Background(), "/memory/workflows.json")
	if err != nil {
		t.Fatalf("read cleared workflow json error = %v", err)
	}
	workflows = decodeWorkflowViewsHelper(t, []byte(workflowsRaw))
	deliver = requireWorkflowViewHelper(t, workflows, "deliver")
	if deliver.Status != "completed" || deliver.StatusSource != "trace" || deliver.StatusReason != "" {
		t.Fatalf("workflow after clear = %+v, want completed/trace/blank", deliver)
	}
}

func TestSkillProjectionViewsAndMemoryEvidence(t *testing.T) {
	adapter := New(Options{
		Skills: map[string]string{
			"draft-plan":         "# Draft Plan Skill\n",
			"reporting/SKILL.md": "# Reporting Skill\n",
		},
		SkillMetadata: map[string]SkillMetadata{
			"draft-plan": {
				Source:         "skill_bundle",
				Freshness:      projectionFreshnessSnapshot,
				SelectionScope: "planning",
				Eligibility:    SkillEligibility{State: skillEligibilityEligible},
				Precedence:     SkillPrecedence{Tier: skillPrecedenceTierBundled, Rank: 10},
			},
			"reporting/SKILL.md": {
				Source:         "skill_bundle",
				Freshness:      projectionFreshnessSnapshot,
				SelectionScope: "planning",
				Eligibility:    SkillEligibility{State: skillEligibilityIneligible, Reason: "missing template"},
				Precedence:     SkillPrecedence{Tier: skillPrecedenceTierBundled, Rank: 50},
			},
		},
	})

	projection, err := adapter.buildProjection(sessionState{
		Freshness:  "observed",
		ReadSkills: []string{"/skills/draft-plan/SKILL.md"},
	})
	if err != nil {
		t.Fatalf("buildProjection(skills) error = %v", err)
	}

	skillIndexRaw, err := projection.VirtualMounts[1].ReadRawContent(context.Background(), "/skills/_index.json")
	if err != nil {
		t.Fatalf("read skills index error = %v", err)
	}
	skillIndex := decodeProjectionViewHelper(t, []byte(`{"skills":`+string(skillIndexRaw)+`}`))
	if len(skillIndex.Skills) != 2 {
		t.Fatalf("skill projections = %+v, want 2", skillIndex.Skills)
	}
	draft := requireProjectionRecordHelper(t, skillIndex.Skills, "/skills/draft-plan/SKILL.md")
	if draft.Source != "skill_bundle" || draft.Freshness != projectionFreshnessSnapshot || draft.Eligibility == nil || draft.Eligibility.State != skillEligibilityEligible || draft.Precedence == nil || draft.Precedence.Rank != 10 || !draft.Selected {
		t.Fatalf("draft skill projection = %+v, want eligible selected bundled/10 snapshot", draft)
	}
	if draft.Selection == nil || draft.Selection.Mode != skillSelectionModeDerived || draft.Selection.Scope != "planning" || draft.Selection.Reason != "highest_precedence" {
		t.Fatalf("draft skill selection = %+v, want derived planning winner", draft.Selection)
	}
	reporting := requireProjectionRecordHelper(t, skillIndex.Skills, "/skills/reporting/SKILL.md")
	if reporting.Eligibility == nil || reporting.Eligibility.State != skillEligibilityIneligible || reporting.Eligibility.Reason != "missing template" || reporting.Selected {
		t.Fatalf("reporting skill projection = %+v, want ineligible not-selected", reporting)
	}
	if reporting.Selection == nil || reporting.Selection.Mode != skillSelectionModeDerived || reporting.Selection.Scope != "planning" || reporting.Selection.Reason != "ineligible" {
		t.Fatalf("reporting skill selection = %+v, want derived ineligible loser", reporting.Selection)
	}

	projectionsRaw, err := projection.Memory.Mount.ReadRawContent(context.Background(), "/memory/projections.json")
	if err != nil {
		t.Fatalf("read memory projections error = %v", err)
	}
	projectionView := decodeProjectionViewHelper(t, []byte(projectionsRaw))
	if len(projectionView.Skills) != 2 {
		t.Fatalf("memory projection skills = %+v, want 2", projectionView.Skills)
	}
	skillsRaw, err := projection.Memory.Mount.ReadRawContent(context.Background(), "/memory/skills.md")
	if err != nil {
		t.Fatalf("read memory skills evidence error = %v", err)
	}
	if !strings.Contains(skillsRaw, "/skills/draft-plan/SKILL.md") {
		t.Fatalf("memory skills evidence = %q, want draft-plan path", skillsRaw)
	}
	summaryRaw, err := projection.Memory.Mount.ReadRawContent(context.Background(), "/memory/summary.md")
	if err != nil {
		t.Fatalf("read memory summary error = %v", err)
	}
	if !strings.Contains(summaryRaw, "- projections.skills: 2") || !strings.Contains(summaryRaw, "- skill_reads: 1") {
		t.Fatalf("memory summary = %q, want skill projection and read counts", summaryRaw)
	}
}

func TestCuratedControlPlaneViewsAreDistinctAndAuditable(t *testing.T) {
	adapter := New(Options{
		Curated: []CuratedEntry{
			{
				ID:      "decision/read-only-memory",
				Title:   "Managed memory remains read-only",
				Summary: "curation writes stay in control plane",
				Source:  "seed_data",
				SourcePaths: []string{
					"/memory/observations.md",
					"memory/projections.json",
				},
			},
		},
	})

	initial, err := adapter.buildProjection(sessionState{Freshness: "created"})
	if err != nil {
		t.Fatalf("buildProjection(curated initial) error = %v", err)
	}
	curatedRaw, err := initial.Memory.Mount.ReadRawContent(context.Background(), "/memory/curated.json")
	if err != nil {
		t.Fatalf("read curated json error = %v", err)
	}
	curated := decodeCuratedRecordsHelper(t, []byte(curatedRaw))
	if len(curated) < 1 {
		t.Fatalf("curated entries = %+v, want at least one entry", curated)
	}
	first := requireCuratedRecordHelper(t, curated, "decision/read-only-memory")
	if first.ID != "decision/read-only-memory" || first.Source != "seed_data" || first.Revision != 1 {
		t.Fatalf("curated entry metadata = %+v, want stable id + seed source + rev=1", first)
	}
	if !reflect.DeepEqual(first.SourcePaths, []string{"/memory/observations.md", "/memory/projections.json"}) {
		t.Fatalf("curated entry source paths = %#v, want normalized source path references", first.SourcePaths)
	}
	curatedMD, err := initial.Memory.Mount.ReadRawContent(context.Background(), "/memory/curated.md")
	if err != nil {
		t.Fatalf("read curated md error = %v", err)
	}
	if !strings.Contains(curatedMD, "# Curated Memory") || !strings.Contains(curatedMD, "[decision/read-only-memory]") {
		t.Fatalf("curated md = %q, want curated-only view with stable id", curatedMD)
	}
	projectionsRaw, err := initial.Memory.Mount.ReadRawContent(context.Background(), "/memory/projections.json")
	if err != nil {
		t.Fatalf("read projections json error = %v", err)
	}
	if strings.Contains(projectionsRaw, "decision/read-only-memory") {
		t.Fatalf("projections json unexpectedly includes curated entry: %q", projectionsRaw)
	}

	adapter.UpsertCuratedEntry(CuratedEntry{
		ID:          "decision/read-only-memory",
		Title:       "Managed memory remains read-only",
		Summary:     "curation updates are explicit control-plane operations",
		SourcePaths: []string{"/memory/workflows.json"},
	})
	updated, err := adapter.buildProjection(sessionState{Freshness: "observed"})
	if err != nil {
		t.Fatalf("buildProjection(curated updated) error = %v", err)
	}
	curatedRaw, err = updated.Memory.Mount.ReadRawContent(context.Background(), "/memory/curated.json")
	if err != nil {
		t.Fatalf("read updated curated json error = %v", err)
	}
	curated = decodeCuratedRecordsHelper(t, []byte(curatedRaw))
	first = requireCuratedRecordHelper(t, curated, "decision/read-only-memory")
	if first.Source != "control_plane" || first.Revision != 2 || first.Summary != "curation updates are explicit control-plane operations" {
		t.Fatalf("updated curated entry = %+v, want control_plane rev=2 with updated summary", first)
	}
	if !reflect.DeepEqual(first.SourcePaths, []string{"/memory/workflows.json"}) {
		t.Fatalf("updated curated source paths = %#v, want workflows source path", first.SourcePaths)
	}
	summaryRaw, err := updated.Memory.Mount.ReadRawContent(context.Background(), "/memory/summary.md")
	if err != nil {
		t.Fatalf("read memory summary with curated entries error = %v", err)
	}
	if !strings.Contains(summaryRaw, "- curated_entries: 1") {
		t.Fatalf("memory summary = %q, want curated entry count", summaryRaw)
	}

	adapter.RemoveCuratedEntry("decision/read-only-memory")
	removed, err := adapter.buildProjection(sessionState{Freshness: "resumed"})
	if err != nil {
		t.Fatalf("buildProjection(curated removed) error = %v", err)
	}
	curatedRaw, err = removed.Memory.Mount.ReadRawContent(context.Background(), "/memory/curated.json")
	if err != nil {
		t.Fatalf("read curated json after remove error = %v", err)
	}
	curated = decodeCuratedRecordsHelper(t, []byte(curatedRaw))
	if len(curated) != 0 {
		t.Fatalf("curated entries after remove = %+v, want none", curated)
	}
}

func TestSkillControlPlaneAuditViewsNormalizePathsAndExposeVisibility(t *testing.T) {
	adapter := New(Options{})
	adapter.UpsertSkill(" planning/winner ", "# Winner\n", SkillMetadata{
		SelectionScope: "planning",
		Eligibility:    SkillEligibility{State: skillEligibilityEligible},
		Precedence:     SkillPrecedence{Tier: skillPrecedenceTierWorkspace, Rank: 0},
	})
	projection, err := adapter.buildProjection(sessionState{Freshness: "observed"})
	if err != nil {
		t.Fatalf("buildProjection(skill audit) error = %v", err)
	}

	summary := mustReadFromMounts(t, []contract.VirtualMount{projection.Memory.Mount}, "/memory/skills_audit.md")
	if !strings.Contains(summary, "/skills/planning/winner/SKILL.md") {
		t.Fatalf("skills audit summary = %q, want normalized path", summary)
	}
	if !strings.Contains(summary, "visible_from=1") {
		t.Fatalf("skills audit summary = %q, want visibility timing line", summary)
	}

	jsonRaw := mustReadFromMounts(t, []contract.VirtualMount{projection.Memory.Mount}, "/memory/skills_audit.json")
	events := decodeControlPlaneEventViewsHelper(t, []byte(jsonRaw))
	if len(events) != 1 {
		t.Fatalf("skills audit events = %+v, want a single entry", events)
	}
	event := events[0]
	if event.Path != "/skills/planning/winner/SKILL.md" {
		t.Fatalf("event path = %q, want normalized path", event.Path)
	}
	if event.VisibleAfter != controlPlaneVisibilityNextProjection {
		t.Fatalf("visible_after = %q, want %s", event.VisibleAfter, controlPlaneVisibilityNextProjection)
	}
	if event.VisibleFromGeneration != 1 {
		t.Fatalf("visible_from_generation = %d, want 1", event.VisibleFromGeneration)
	}
	if event.SelectionScope != "planning" {
		t.Fatalf("selection_scope = %q, want planning", event.SelectionScope)
	}
	if !event.SelectedAfter {
		t.Fatalf("SelectedAfter = %v, want true", event.SelectedAfter)
	}
}

func TestSkillControlPlaneEventsMaintainOrderAndSelectionTransitions(t *testing.T) {
	adapter := New(Options{})
	adapter.UpsertSkill("planning/fallback", "# Fallback\n", SkillMetadata{
		SelectionScope: "planning/default",
		Eligibility:    SkillEligibility{State: skillEligibilityEligible},
		Precedence:     SkillPrecedence{Tier: skillPrecedenceTierWorkspace, Rank: 5},
	})
	adapter.UpsertSkill("planning/primary", "# Primary\n", SkillMetadata{
		SelectionScope: "planning/default",
		Eligibility:    SkillEligibility{State: skillEligibilityEligible},
		Precedence:     SkillPrecedence{Tier: skillPrecedenceTierWorkspace, Rank: 1},
	})
	adapter.RemoveSkill("planning/primary")

	projection, err := adapter.buildProjection(sessionState{Freshness: "observed"})
	if err != nil {
		t.Fatalf("buildProjection(skill audit order) error = %v", err)
	}
	events := decodeControlPlaneEventViewsHelper(t, []byte(mustReadFromMounts(t, []contract.VirtualMount{projection.Memory.Mount}, "/memory/skills_audit.json")))
	if len(events) != 3 {
		t.Fatalf("skills audit events = %+v, want three entries", events)
	}
	if !(events[0].Seq < events[1].Seq && events[1].Seq < events[2].Seq) {
		t.Fatalf("event sequence = %v, want increasing seq", []int{events[0].Seq, events[1].Seq, events[2].Seq})
	}

	if events[0].Op != controlPlaneEventKindSkillAdded || events[0].Path != "/skills/planning/fallback/SKILL.md" {
		t.Fatalf("first event = %+v, want fallback added", events[0])
	}
	if events[1].Op != controlPlaneEventKindSkillAdded || events[1].Path != "/skills/planning/primary/SKILL.md" {
		t.Fatalf("second event = %+v, want primary added", events[1])
	}
	if events[2].Op != controlPlaneEventKindSkillRemoved || events[2].Path != "/skills/planning/primary/SKILL.md" {
		t.Fatalf("third event = %+v, want primary removed", events[2])
	}

	if !events[1].SelectedAfter || events[1].WinnerAfter != "/skills/planning/primary/SKILL.md" {
		t.Fatalf("primary event = %+v, want selected winner", events[1])
	}
	if !events[2].SelectedBefore {
		t.Fatalf("removal event = %+v, want SelectedBefore true", events[2])
	}
	if events[2].WinnerAfter != "/skills/planning/fallback/SKILL.md" {
		t.Fatalf("removal event winner_after = %q, want fallback path", events[2].WinnerAfter)
	}
}

func TestProjectionMetricsViewTracksProjectionTruth(t *testing.T) {
	adapter := New(Options{
		Documents: map[string]string{"guide.md": "# Guide\n"},
		DocumentMetadata: map[string]ProjectionMetadata{
			"guide.md": {Source: "knowledge_sync", Freshness: projectionFreshnessSnapshot},
		},
		Skills: map[string]string{"planning/fallback": "# Fallback\n"},
		SkillMetadata: map[string]SkillMetadata{
			"planning/fallback": {
				SelectionScope: "planning/default",
				Eligibility:    SkillEligibility{State: skillEligibilityEligible},
				Precedence:     SkillPrecedence{Tier: skillPrecedenceTierBundled, Rank: 80},
			},
		},
	})

	projection, err := adapter.buildProjection(sessionState{
		Freshness:   "observed",
		DeniedPaths: []string{"/skills/planning/fallback/SKILL.md"},
	})
	if err != nil {
		t.Fatalf("buildProjection(metrics) error = %v", err)
	}

	metricsRaw := mustReadFromMounts(t, []contract.VirtualMount{projection.Memory.Mount}, "/memory/projection_metrics.json")
	metrics := decodeProjectionMetricsViewHelper(t, []byte(metricsRaw))
	if metrics.ProjectionGeneration != 1 || metrics.ControlPlaneEvents != 0 || metrics.UniqueDeniedPaths != 1 {
		t.Fatalf("projection metrics = %+v, want generation=1 events=0 denials=1", metrics)
	}
	if metrics.CacheHitMetricsAvailable {
		t.Fatalf("cache-hit metrics should be unavailable without a real cache: %+v", metrics)
	}
	if metrics.ProjectionCounts["documents"] != 1 || metrics.ProjectionCounts["skills"] != 1 || metrics.ProjectionCounts["resources"] != 0 {
		t.Fatalf("projection counts = %+v, want documents=1 skills=1 resources=0", metrics.ProjectionCounts)
	}
	if metrics.FreshnessCounts[projectionFreshnessSnapshot] != 2 {
		t.Fatalf("freshness counts = %+v, want snapshot=2", metrics.FreshnessCounts)
	}

	summary := mustReadFromMounts(t, []contract.VirtualMount{projection.Memory.Mount}, "/memory/projection_metrics.md")
	if !strings.Contains(summary, "projection_generation: 1") || !strings.Contains(summary, "cache_hit_metrics_available: false") {
		t.Fatalf("projection metrics summary = %q, want generation and cache flag", summary)
	}
}

func TestBuildProjectionMetricsViewDedupsDeniedPaths(t *testing.T) {
	state := sessionState{
		ProjectionGeneration: 7,
		DeniedPaths: []string{
			"/knowledge_base/reference/guide.md",
			"/knowledge_base/reference/guide.md",
			"/skills/planning/draft-plan/SKILL.md",
		},
	}
	metrics := buildProjectionMetricsView(state, nil, nil, nil, nil)
	expectedUnique := len(dedupeLines(state.DeniedPaths))
	if metrics.UniqueDeniedPaths != expectedUnique {
		t.Fatalf("unique denied paths = %d, want %d", metrics.UniqueDeniedPaths, expectedUnique)
	}
	if metrics.ProjectionGeneration != state.ProjectionGeneration {
		t.Fatalf("projection generation = %d, want %d", metrics.ProjectionGeneration, state.ProjectionGeneration)
	}
	if metrics.ControlPlaneEvents != 0 {
		t.Fatalf("control plane events = %d, want 0", metrics.ControlPlaneEvents)
	}
	if metrics.CacheHitMetricsAvailable {
		t.Fatalf("cache hit metrics should be unavailable: %+v", metrics)
	}
}

func TestDenialViewClassifiesNamespacesAndDedupesPaths(t *testing.T) {
	adapter := New(Options{})
	projection, err := adapter.buildProjection(sessionState{
		Freshness: "observed",
		DeniedPaths: []string{
			"/knowledge_base/reference/guide.md",
			"/skills/planning/draft-plan/SKILL.md",
			"/memory/curated.json",
			"/skills/planning/draft-plan/SKILL.md",
			"/outside/path.txt",
		},
	})
	if err != nil {
		t.Fatalf("buildProjection(denials) error = %v", err)
	}

	denialsRaw := mustReadFromMounts(t, []contract.VirtualMount{projection.Memory.Mount}, "/memory/denials.json")
	denials := decodeDenialViewHelper(t, []byte(denialsRaw))
	if denials.ProjectionGeneration != 1 || denials.UniqueDeniedPaths != 4 {
		t.Fatalf("denial view = %+v, want generation=1 unique=4", denials)
	}
	if denials.ByNamespace["reference"] != 1 || denials.ByNamespace["skills"] != 1 || denials.ByNamespace["memory"] != 1 || denials.ByNamespace["external_or_unknown"] != 1 {
		t.Fatalf("denial namespace counts = %+v, want one per namespace bucket", denials.ByNamespace)
	}
	if !reflect.DeepEqual(denials.SamplePaths, []string{
		"/knowledge_base/reference/guide.md",
		"/memory/curated.json",
		"/outside/path.txt",
		"/skills/planning/draft-plan/SKILL.md",
	}) {
		t.Fatalf("denial sample paths = %#v, want deduped sorted paths", denials.SamplePaths)
	}

	denialSummary := mustReadFromMounts(t, []contract.VirtualMount{projection.Memory.Mount}, "/memory/denials.md")
	if !strings.Contains(denialSummary, "unique_denied_paths: 4") || !strings.Contains(denialSummary, "external_or_unknown: 1") {
		t.Fatalf("denial summary = %q, want unique count and namespace summary", denialSummary)
	}
}

func decodeProjectionRecordsHelper(t *testing.T, raw []byte) []projectionRecord {
	t.Helper()
	var records []projectionRecord
	if err := json.Unmarshal(raw, &records); err != nil {
		t.Fatalf("decode projection records failed: %v", err)
	}
	return records
}

func decodeProjectionViewHelper(t *testing.T, raw []byte) projectionView {
	t.Helper()
	var view projectionView
	if err := json.Unmarshal(raw, &view); err != nil {
		t.Fatalf("decode projection view failed: %v", err)
	}
	return view
}

func decodeControlPlaneEventViewsHelper(t *testing.T, raw []byte) []controlPlaneEventViewRecord {
	t.Helper()
	var events []controlPlaneEventViewRecord
	if err := json.Unmarshal(raw, &events); err != nil {
		t.Fatalf("decode control-plane event views failed: %v", err)
	}
	return events
}

func decodeProjectionMetricsViewHelper(t *testing.T, raw []byte) projectionMetricsView {
	t.Helper()
	var metrics projectionMetricsView
	if err := json.Unmarshal(raw, &metrics); err != nil {
		t.Fatalf("decode projection metrics failed: %v", err)
	}
	return metrics
}

func decodeDenialViewHelper(t *testing.T, raw []byte) denialView {
	t.Helper()
	var view denialView
	if err := json.Unmarshal(raw, &view); err != nil {
		t.Fatalf("decode denial view failed: %v", err)
	}
	return view
}

func decodeReferenceStateHelper(t *testing.T, raw string) sessionState {
	t.Helper()
	var state sessionState
	if err := json.Unmarshal([]byte(raw), &state); err != nil {
		t.Fatalf("decode reference state helper failed: %v", err)
	}
	return state
}

func decodeWorkflowViewsHelper(t *testing.T, raw []byte) []workflowView {
	t.Helper()
	var workflows []workflowView
	if err := json.Unmarshal(raw, &workflows); err != nil {
		t.Fatalf("decode workflow views failed: %v", err)
	}
	return workflows
}

func decodeCuratedRecordsHelper(t *testing.T, raw []byte) []curatedRecord {
	t.Helper()
	var records []curatedRecord
	if err := json.Unmarshal(raw, &records); err != nil {
		t.Fatalf("decode curated records failed: %v", err)
	}
	return records
}

func requireCuratedRecordHelper(t *testing.T, records []curatedRecord, id string) curatedRecord {
	t.Helper()
	for _, record := range records {
		if record.ID == id {
			return record
		}
	}
	t.Fatalf("curated record %q not found in %+v", id, records)
	return curatedRecord{}
}

func requireWorkflowViewHelper(t *testing.T, workflows []workflowView, id string) workflowView {
	t.Helper()
	for _, workflow := range workflows {
		if workflow.ID == id {
			return workflow
		}
	}
	t.Fatalf("workflow %q not found in %+v", id, workflows)
	return workflowView{}
}

func requireProjectionRecordHelper(t *testing.T, records []projectionRecord, target string) projectionRecord {
	t.Helper()
	for _, record := range records {
		if record.Path == target {
			return record
		}
	}
	t.Fatalf("projection record %q not found in %+v", target, records)
	return projectionRecord{}
}

func mustReadFromMounts(t *testing.T, mounts []contract.VirtualMount, target string) string {
	t.Helper()
	for _, mounted := range mounts {
		raw, err := mounted.ReadRawContent(context.Background(), target)
		if err == nil {
			return raw
		}
	}
	t.Fatalf("path %q not found across mounts", target)
	return ""
}
