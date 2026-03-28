package reference

import (
	"context"
	"encoding/json"
	"errors"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"testing"

	"github.com/khicago/simsh/pkg/contract"
	runtimeengine "github.com/khicago/simsh/pkg/engine/runtime"
)

func TestReferenceAdapterEndToEnd(t *testing.T) {
	adapter := New(Options{
		Documents: map[string]string{
			"guide.md": "# Guide\nhello\n",
		},
		DocumentMetadata: map[string]ProjectionMetadata{
			"guide.md": {Source: "knowledge_sync", Freshness: "snapshot"},
		},
		Resources: map[string]string{
			"checklists/plan.json": "{\"steps\":[\"read\",\"write\"]}\n",
			"templates/report.md":  "# Report Template\n",
		},
		ResourceMetadata: map[string]ProjectionMetadata{
			"checklists/plan.json": {Source: "workflow_catalog", Freshness: "live"},
			"templates/report.md":  {Source: "workflow_catalog", Freshness: "snapshot"},
		},
		Workflows: []WorkflowSpec{
			{
				ID:              "draft-plan",
				Title:           "Draft plan",
				Summary:         "Read the planning checklist and write the first plan draft.",
				ResourcePaths:   []string{"/resources/checklists/plan.json"},
				ExpectedOutputs: []string{"/task_outputs/plan.txt"},
			},
			{
				ID:              "final-report",
				Title:           "Final report",
				Summary:         "Read the report template before writing the final report.",
				ResourcePaths:   []string{"/resources/templates/report.md"},
				ExpectedOutputs: []string{"/task_outputs/final.md"},
			},
		},
	})
	manager := runtimeengine.NewSessionManager(runtimeengine.SessionManagerOptions{NewID: func() string { return "sess_reference" }})
	session, err := manager.Create(context.Background(), runtimeengine.Options{
		HostRoot: t.TempDir(),
		Profile:  contract.ProfileBashPlus,
		Policy: contract.ExecutionPolicy{
			WriteMode:        contract.WriteModeFull,
			MaxPipelineDepth: 16,
			MaxOutputBytes:   4 << 20,
			Timeout:          contract.DefaultPolicy().Timeout,
		},
		Adapters: []contract.SessionAdapter{adapter},
	})
	if err != nil {
		t.Fatalf("create session failed: %v", err)
	}

	statusBeforeObserve, err := manager.Execute(context.Background(), session.SessionID, "cat /memory/status.json", contract.ExecutionPolicy{})
	if err != nil {
		t.Fatalf("read initial memory status failed: %v", err)
	}
	if state := decodeReferenceState(t, []byte(statusBeforeObserve.Result.Stdout)); state.Freshness != "created" {
		t.Fatalf("expected created freshness, got %+v", state)
	}
	createdWorkflows, err := manager.Execute(context.Background(), session.SessionID, "cat /memory/workflows.json", contract.ExecutionPolicy{})
	if err != nil {
		t.Fatalf("read initial workflows failed: %v", err)
	}
	createdState := decodeWorkflowViews(t, []byte(createdWorkflows.Result.Stdout))
	if got := workflowStatusByID(createdState, "draft-plan"); got != "pending" {
		t.Fatalf("draft-plan initial status = %q, want pending", got)
	}
	projections, err := manager.Execute(context.Background(), session.SessionID, "cat /memory/projections.json", contract.ExecutionPolicy{})
	if err != nil {
		t.Fatalf("read initial projections failed: %v", err)
	}
	projectionView := decodeProjectionView(t, []byte(projections.Result.Stdout))
	guideProjection := requireProjectionRecord(t, projectionView.Documents, "/knowledge_base/reference/guide.md")
	if guideProjection.Source != "knowledge_sync" || guideProjection.Freshness != "snapshot" {
		t.Fatalf("unexpected guide projection metadata: %+v", guideProjection)
	}
	planProjection := requireProjectionRecord(t, projectionView.Resources, "/resources/checklists/plan.json")
	if planProjection.Source != "workflow_catalog" || planProjection.Freshness != "live" {
		t.Fatalf("unexpected plan projection metadata: %+v", planProjection)
	}
	resourceIndex, err := manager.Execute(context.Background(), session.SessionID, "cat /resources/_index.json", contract.ExecutionPolicy{})
	if err != nil {
		t.Fatalf("read resource index failed: %v", err)
	}
	indexResources := decodeProjectionRecords(t, []byte(resourceIndex.Result.Stdout))
	if record := requireProjectionRecord(t, indexResources, "/resources/checklists/plan.json"); record.Name != "checklists/plan.json" {
		t.Fatalf("unexpected resource index record: %+v", record)
	}

	guide, err := manager.Execute(context.Background(), session.SessionID, "cat /knowledge_base/reference/guide.md", contract.ExecutionPolicy{})
	if err != nil {
		t.Fatalf("read reference doc failed: %v", err)
	}
	if !strings.Contains(guide.Result.Stdout, "# Guide") {
		t.Fatalf("unexpected guide output: %+v", guide.Result)
	}
	resource, err := manager.Execute(context.Background(), session.SessionID, "cat /resources/checklists/plan.json", contract.ExecutionPolicy{})
	if err != nil {
		t.Fatalf("read resource failed: %v", err)
	}
	if !strings.Contains(resource.Result.Stdout, "\"steps\"") {
		t.Fatalf("unexpected resource output: %+v", resource.Result)
	}
	inProgressWorkflows, err := manager.Execute(context.Background(), session.SessionID, "cat /memory/workflows.json", contract.ExecutionPolicy{})
	if err != nil {
		t.Fatalf("read workflows after resource access failed: %v", err)
	}
	inProgressState := decodeWorkflowViews(t, []byte(inProgressWorkflows.Result.Stdout))
	if got := workflowStatusByID(inProgressState, "draft-plan"); got != "in_progress" {
		t.Fatalf("draft-plan status after reading resource = %q, want in_progress", got)
	}
	if got := workflowStatusByID(inProgressState, "final-report"); got != "pending" {
		t.Fatalf("final-report status before its resource is read = %q, want pending", got)
	}

	if _, err := manager.Execute(context.Background(), session.SessionID, "echo plan > /task_outputs/plan.txt", contract.ExecutionPolicy{}); err != nil {
		t.Fatalf("write task output failed: %v", err)
	}
	denied, err := manager.Execute(context.Background(), session.SessionID, "echo blocked > /knowledge_base/reference/guide.md", contract.ExecutionPolicy{})
	if err != nil {
		t.Fatalf("denied write failed unexpectedly: %v", err)
	}
	if denied.Result.ExitCode == 0 {
		t.Fatalf("expected denied write to fail, got %+v", denied.Result)
	}
	observations, err := manager.Execute(context.Background(), session.SessionID, "cat /memory/observations.md", contract.ExecutionPolicy{})
	if err != nil {
		t.Fatalf("read memory observations failed: %v", err)
	}
	if !strings.Contains(observations.Result.Stdout, "read-ref:/knowledge_base/reference/guide.md") {
		t.Fatalf("expected read-ref observation, got %+v", observations.Result)
	}
	if !strings.Contains(observations.Result.Stdout, "wrote:/task_outputs/plan.txt") {
		t.Fatalf("expected wrote observation, got %+v", observations.Result)
	}
	if !strings.Contains(observations.Result.Stdout, "denied:/knowledge_base/reference/guide.md") {
		t.Fatalf("expected denied observation, got %+v", observations.Result)
	}
	if strings.Count(observations.Result.Stdout, "read-ref:/knowledge_base/reference/guide.md") != 1 {
		t.Fatalf("expected deduped read observation, got %+v", observations.Result)
	}
	if !strings.Contains(observations.Result.Stdout, "read-resource:/resources/checklists/plan.json") {
		t.Fatalf("expected resource observation, got %+v", observations.Result)
	}
	workflowsAfterWrite, err := manager.Execute(context.Background(), session.SessionID, "cat /memory/workflows.md", contract.ExecutionPolicy{})
	if err != nil {
		t.Fatalf("read workflow summary failed: %v", err)
	}
	if !strings.Contains(workflowsAfterWrite.Result.Stdout, "[completed] Draft plan (draft-plan)") {
		t.Fatalf("expected completed workflow summary, got %+v", workflowsAfterWrite.Result)
	}
	summary, err := manager.Execute(context.Background(), session.SessionID, "cat /memory/summary.md", contract.ExecutionPolicy{})
	if err != nil {
		t.Fatalf("read memory summary failed: %v", err)
	}
	if !strings.Contains(summary.Result.Stdout, "resource_reads: 2") || !strings.Contains(summary.Result.Stdout, "written_outputs: 1") {
		t.Fatalf("expected richer managed memory summary, got %+v", summary.Result)
	}
	if !strings.Contains(summary.Result.Stdout, "projections.documents: 1") || !strings.Contains(summary.Result.Stdout, "projections.resources: 2") {
		t.Fatalf("expected projection counts in managed memory summary, got %+v", summary.Result)
	}

	checkpoint, err := manager.Checkpoint(context.Background(), session.SessionID)
	if err != nil {
		t.Fatalf("checkpoint failed: %v", err)
	}
	if state := decodeReferenceState(t, checkpoint.State.Opaque[adapter.AdapterID()]); state.Freshness != "checkpointed" {
		t.Fatalf("expected checkpointed freshness, got %+v", state)
	}
	closed, err := manager.Close(context.Background(), session.SessionID)
	if err != nil {
		t.Fatalf("close failed: %v", err)
	}
	if state := decodeReferenceState(t, closed.State.Opaque[adapter.AdapterID()]); state.Freshness != "closed" {
		t.Fatalf("expected closed freshness, got %+v", state)
	}
	adapter.UpdateDocument("guide.md", "# Guide\nupdated\n")
	adapter.UpdateResource("checklists/plan.json", "{\"steps\":[\"updated\"]}\n")
	adapter.UpsertResource("catalog/live.json", "{\"live\":true}\n", ProjectionMetadata{Source: "control_plane", Freshness: "live"})
	adapter.UpsertWorkflow(WorkflowSpec{
		ID:              "deliver-report",
		Title:           "Deliver report",
		Summary:         "Read the report template before writing the final report.",
		ResourcePaths:   []string{"/resources/templates/report.md"},
		ExpectedOutputs: []string{"/task_outputs/final.md"},
	})
	resumed, err := manager.Resume(context.Background(), session.SessionID)
	if err != nil {
		t.Fatalf("resume failed: %v", err)
	}
	resumedState := decodeReferenceState(t, resumed.State.Opaque[adapter.AdapterID()])
	if resumedState.Freshness != "resumed" {
		t.Fatalf("expected resumed freshness, got %+v", resumedState)
	}
	if !containsObservation(resumedState.Observations, "denied:/knowledge_base/reference/guide.md") {
		t.Fatalf("expected resumed state to preserve denied observation, got %+v", resumedState)
	}
	if got := workflowStatusByID(resumedState.Workflows, "draft-plan"); got != "completed" {
		t.Fatalf("draft-plan resumed status = %q, want completed", got)
	}
	if got := workflowStatusByID(resumedState.Workflows, "deliver-report"); got != "pending" {
		t.Fatalf("deliver-report resumed status = %q, want pending", got)
	}

	updatedGuide, err := manager.Execute(context.Background(), session.SessionID, "cat /knowledge_base/reference/guide.md", contract.ExecutionPolicy{})
	if err != nil {
		t.Fatalf("read updated guide failed: %v", err)
	}
	if !strings.Contains(updatedGuide.Result.Stdout, "updated") {
		t.Fatalf("expected updated reference projection, got %+v", updatedGuide.Result)
	}
	updatedResource, err := manager.Execute(context.Background(), session.SessionID, "cat /resources/checklists/plan.json", contract.ExecutionPolicy{})
	if err != nil {
		t.Fatalf("read updated resource failed: %v", err)
	}
	if !strings.Contains(updatedResource.Result.Stdout, "\"updated\"") {
		t.Fatalf("expected updated resource projection, got %+v", updatedResource.Result)
	}
	liveResource, err := manager.Execute(context.Background(), session.SessionID, "cat /resources/catalog/live.json", contract.ExecutionPolicy{})
	if err != nil {
		t.Fatalf("read live resource failed: %v", err)
	}
	if !strings.Contains(liveResource.Result.Stdout, "\"live\":true") {
		t.Fatalf("expected control-plane resource projection, got %+v", liveResource.Result)
	}
	updatedProjections, err := manager.Execute(context.Background(), session.SessionID, "cat /memory/projections.json", contract.ExecutionPolicy{})
	if err != nil {
		t.Fatalf("read updated projections failed: %v", err)
	}
	updatedProjectionView := decodeProjectionView(t, []byte(updatedProjections.Result.Stdout))
	updatedGuideProjection := requireProjectionRecord(t, updatedProjectionView.Documents, "/knowledge_base/reference/guide.md")
	if updatedGuideProjection.Source != "control_plane" || updatedGuideProjection.Freshness != "updated" {
		t.Fatalf("expected updated guide metadata, got %+v", updatedGuideProjection)
	}
	liveProjection := requireProjectionRecord(t, updatedProjectionView.Resources, "/resources/catalog/live.json")
	if liveProjection.Source != "control_plane" || liveProjection.Freshness != "live" {
		t.Fatalf("expected control-plane resource metadata, got %+v", liveProjection)
	}

	current, err := manager.Get(session.SessionID)
	if err != nil {
		t.Fatalf("get session failed: %v", err)
	}
	currentState := decodeReferenceState(t, current.State.Opaque[adapter.AdapterID()])
	if currentState.Freshness != "observed" {
		t.Fatalf("expected observed freshness after post-resume read, got %+v", currentState)
	}
	if !containsObservation(currentState.Observations, "read-ref:/knowledge_base/reference/guide.md") ||
		!containsObservation(currentState.Observations, "wrote:/task_outputs/plan.txt") ||
		!containsObservation(currentState.Observations, "denied:/knowledge_base/reference/guide.md") {
		t.Fatalf("expected persisted observations, got %+v", currentState)
	}
	if !containsObservation(currentState.Observations, "read-resource:/resources/checklists/plan.json") {
		t.Fatalf("expected persisted resource observation, got %+v", currentState)
	}
	if !containsLine(currentState.ReadResources, "/resources/checklists/plan.json") {
		t.Fatalf("expected persisted resource read set, got %+v", currentState)
	}
}

func TestReferenceAdapterProjectionMetricsAndDenials(t *testing.T) {
	const (
		docPath   = "/knowledge_base/reference/guide.md"
		skillPath = "/skills/planning/draft-plan/SKILL.md"
	)

	adapter := New(Options{
		Documents: map[string]string{
			"guide.md": "# Guide\nmetrics\n",
		},
		DocumentMetadata: map[string]ProjectionMetadata{
			"guide.md": {Source: "knowledge_sync", Freshness: projectionFreshnessSnapshot},
		},
		Resources: map[string]string{
			"checklists/plan.json": "{\"steps\":[\"read\"]}\n",
		},
		ResourceMetadata: map[string]ProjectionMetadata{
			"checklists/plan.json": {Source: "workflow_catalog", Freshness: projectionFreshnessLive},
		},
		Skills: map[string]string{
			"planning/draft-plan": "# Draft Planner\n",
		},
		SkillMetadata: map[string]SkillMetadata{
			"planning/draft-plan": {
				Source:         "workspace_catalog",
				Freshness:      projectionFreshnessLive,
				SelectionScope: "planning/default",
				Eligibility: SkillEligibility{
					State: skillEligibilityEligible,
				},
				Precedence: SkillPrecedence{
					Tier: skillPrecedenceTierWorkspace,
					Rank: 1,
				},
				Selected: true,
			},
		},
	})
	manager := runtimeengine.NewSessionManager(runtimeengine.SessionManagerOptions{NewID: func() string { return "sess_projection_metrics" }})
	session, err := manager.Create(context.Background(), runtimeengine.Options{
		HostRoot: t.TempDir(),
		Profile:  contract.ProfileBashPlus,
		Policy: contract.ExecutionPolicy{
			WriteMode:        contract.WriteModeFull,
			MaxPipelineDepth: 16,
			MaxOutputBytes:   4 << 20,
			Timeout:          contract.DefaultPolicy().Timeout,
		},
		Adapters: []contract.SessionAdapter{adapter},
	})
	if err != nil {
		t.Fatalf("create session failed: %v", err)
	}

	if _, err := manager.Execute(context.Background(), session.SessionID, "cat "+docPath, contract.ExecutionPolicy{}); err != nil {
		t.Fatalf("read doc failed: %v", err)
	}
	if _, err := manager.Execute(context.Background(), session.SessionID, "cat /resources/checklists/plan.json", contract.ExecutionPolicy{}); err != nil {
		t.Fatalf("read resource failed: %v", err)
	}
	if _, err := manager.Execute(context.Background(), session.SessionID, "cat "+skillPath, contract.ExecutionPolicy{}); err != nil {
		t.Fatalf("read skill failed: %v", err)
	}

	deniedDoc, err := manager.Execute(context.Background(), session.SessionID, "echo blocked > "+docPath, contract.ExecutionPolicy{})
	if err != nil {
		t.Fatalf("denied doc write failed unexpectedly: %v", err)
	}
	if deniedDoc.Result.ExitCode == 0 {
		t.Fatalf("expected doc write to be denied, got %+v", deniedDoc.Result)
	}

	deniedSkill, err := manager.Execute(context.Background(), session.SessionID, "echo blocked > "+skillPath, contract.ExecutionPolicy{})
	if err != nil {
		t.Fatalf("denied skill write failed unexpectedly: %v", err)
	}
	if deniedSkill.Result.ExitCode == 0 {
		t.Fatalf("expected skill write to be denied, got %+v", deniedSkill.Result)
	}

	metricsResp, err := manager.Execute(context.Background(), session.SessionID, "cat /memory/projection_metrics.json", contract.ExecutionPolicy{})
	if err != nil {
		t.Fatalf("read projection metrics failed: %v", err)
	}
	if metricsResp.Result.ExitCode != 0 {
		t.Fatalf("projection metrics command failed: %+v", metricsResp.Result)
	}
	denialsResp, err := manager.Execute(context.Background(), session.SessionID, "cat /memory/denials.json", contract.ExecutionPolicy{})
	if err != nil {
		t.Fatalf("read denial view failed: %v", err)
	}
	if denialsResp.Result.ExitCode != 0 {
		t.Fatalf("denials command failed: %+v", denialsResp.Result)
	}
	auditResp, err := manager.Execute(context.Background(), session.SessionID, "cat /memory/skills_audit.json", contract.ExecutionPolicy{})
	if err != nil {
		t.Fatalf("read skills audit failed: %v", err)
	}
	if auditResp.Result.ExitCode != 0 {
		t.Fatalf("skills audit command failed: %+v", auditResp.Result)
	}
	statusResp, err := manager.Execute(context.Background(), session.SessionID, "cat /memory/status.json", contract.ExecutionPolicy{})
	if err != nil {
		t.Fatalf("read memory status failed: %v", err)
	}

	state := decodeReferenceState(t, []byte(statusResp.Result.Stdout))
	metrics := decodeProjectionMetricsView(t, []byte(metricsResp.Result.Stdout))
	denials := decodeDenialView(t, []byte(denialsResp.Result.Stdout))
	events := decodeControlPlaneEventViews(t, []byte(auditResp.Result.Stdout))

	if metrics.ProjectionGeneration <= 0 {
		t.Fatalf("projection metrics generation = %d, want positive generation", metrics.ProjectionGeneration)
	}
	if metrics.ControlPlaneEvents != len(events) {
		t.Fatalf("control-plane event count mismatch: metrics=%d audit=%d", metrics.ControlPlaneEvents, len(events))
	}
	if metrics.UniqueDeniedPaths != len(state.DeniedPaths) {
		t.Fatalf("denied-path count mismatch: metrics=%d state=%d", metrics.UniqueDeniedPaths, len(state.DeniedPaths))
	}
	if denials.ProjectionGeneration <= 0 {
		t.Fatalf("denial view projection generation = %d, want positive generation", denials.ProjectionGeneration)
	}
	if metrics.UniqueDeniedPaths != denials.UniqueDeniedPaths {
		t.Fatalf("denied-path count mismatch between metrics and denial view: %d vs %d", metrics.UniqueDeniedPaths, denials.UniqueDeniedPaths)
	}
	if len(denials.SamplePaths) != denials.UniqueDeniedPaths {
		t.Fatalf("denial view sample path length %d != unique count %d", len(denials.SamplePaths), denials.UniqueDeniedPaths)
	}

	for _, expected := range []string{docPath, skillPath} {
		if !containsLine(denials.SamplePaths, expected) {
			t.Fatalf("denial view missing %s paths=%v", expected, denials.SamplePaths)
		}
		if !containsLine(state.DeniedPaths, expected) {
			t.Fatalf("state missing %s denied path", expected)
		}
	}
	documents, docsOK := metrics.ProjectionCounts["documents"]
	resources, resOK := metrics.ProjectionCounts["resources"]
	skills, skillsOK := metrics.ProjectionCounts["skills"]
	if !docsOK || !resOK || !skillsOK || documents != 1 || resources != 1 || skills != 1 {
		t.Fatalf("projection counts unexpected: %+v", metrics.ProjectionCounts)
	}

	if got := metrics.FreshnessCounts[projectionFreshnessSnapshot]; got != 1 {
		t.Fatalf("snapshot freshness count = %d, want 1", got)
	}
	if got := metrics.FreshnessCounts[projectionFreshnessLive]; got != 2 {
		t.Fatalf("live freshness count = %d, want 2", got)
	}

	materialized := 0
	for _, count := range metrics.MaterializationCounts {
		materialized += count
	}
	totalRecords := metrics.ProjectionCounts["documents"] + metrics.ProjectionCounts["resources"] + metrics.ProjectionCounts["skills"]
	if materialized != totalRecords {
		t.Fatalf("materialization counts %d != total records %d", materialized, totalRecords)
	}
	if metrics.CacheHitMetricsAvailable {
		t.Fatalf("cache hit metrics flag should be false for reference adapter")
	}

	if _, err := manager.Close(context.Background(), session.SessionID); err != nil {
		t.Fatalf("close session failed: %v", err)
	}
}

func TestReferenceAdapterProjectionError(t *testing.T) {
	adapter := New(Options{})
	adapter.SetProjectionError(errors.New("projection unavailable"))
	manager := runtimeengine.NewSessionManager(runtimeengine.SessionManagerOptions{})

	_, err := manager.Create(context.Background(), runtimeengine.Options{
		HostRoot: t.TempDir(),
		Profile:  contract.ProfileCoreStrict,
		Policy:   contract.DefaultPolicy(),
		Adapters: []contract.SessionAdapter{adapter},
	})
	if err == nil || !strings.Contains(err.Error(), "projection unavailable") {
		t.Fatalf("expected projection error, got %v", err)
	}
}

func TestReferenceAdapterRefreshAndInvalidationAcrossResume(t *testing.T) {
	adapter := New(Options{
		Documents: map[string]string{
			"guide.md": "# Guide\nhello\n",
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
	manager := runtimeengine.NewSessionManager(runtimeengine.SessionManagerOptions{NewID: func() string { return "sess_refresh" }})

	session, err := manager.Create(context.Background(), runtimeengine.Options{
		HostRoot: t.TempDir(),
		Profile:  contract.ProfileCoreStrict,
		Policy: contract.ExecutionPolicy{
			WriteMode:        contract.WriteModeFull,
			MaxPipelineDepth: 16,
			MaxOutputBytes:   4 << 20,
			Timeout:          contract.DefaultPolicy().Timeout,
		},
		Adapters: []contract.SessionAdapter{adapter},
	})
	if err != nil {
		t.Fatalf("create session failed: %v", err)
	}
	if _, err := manager.Close(context.Background(), session.SessionID); err != nil {
		t.Fatalf("initial close failed: %v", err)
	}

	adapter.InvalidateDocument("guide.md")
	adapter.InvalidateResource("catalog/index.json")
	if _, err := manager.Resume(context.Background(), session.SessionID); err != nil {
		t.Fatalf("resume stale projections failed: %v", err)
	}

	staleProjections, err := manager.Execute(context.Background(), session.SessionID, "cat /memory/projections.json", contract.ExecutionPolicy{})
	if err != nil {
		t.Fatalf("read stale projections failed: %v", err)
	}
	staleView := decodeProjectionView(t, []byte(staleProjections.Result.Stdout))
	if record := requireProjectionRecord(t, staleView.Documents, "/knowledge_base/reference/guide.md"); record.Source != "knowledge_sync" || record.Freshness != "stale" {
		t.Fatalf("stale document projection = %+v, want knowledge_sync/stale", record)
	}
	if record := requireProjectionRecord(t, staleView.Resources, "/resources/catalog/index.json"); record.Source != "workflow_catalog" || record.Freshness != "stale" {
		t.Fatalf("stale resource projection = %+v, want workflow_catalog/stale", record)
	}
	staleSummary, err := manager.Execute(context.Background(), session.SessionID, "cat /memory/summary.md", contract.ExecutionPolicy{})
	if err != nil {
		t.Fatalf("read stale summary failed: %v", err)
	}
	if !strings.Contains(staleSummary.Result.Stdout, "- stale: 2") {
		t.Fatalf("stale summary = %+v, want stale projection count", staleSummary.Result)
	}

	if _, err := manager.Close(context.Background(), session.SessionID); err != nil {
		t.Fatalf("close before refresh failed: %v", err)
	}
	adapter.RefreshDocument("guide.md", "# Guide\nfresh\n", ProjectionMetadata{})
	adapter.RefreshResource("catalog/index.json", "{\"ok\":false}\n", ProjectionMetadata{Source: "catalog_refresh"})
	if _, err := manager.Resume(context.Background(), session.SessionID); err != nil {
		t.Fatalf("resume refreshed projections failed: %v", err)
	}

	liveProjections, err := manager.Execute(context.Background(), session.SessionID, "cat /memory/projections.json", contract.ExecutionPolicy{})
	if err != nil {
		t.Fatalf("read refreshed projections failed: %v", err)
	}
	liveView := decodeProjectionView(t, []byte(liveProjections.Result.Stdout))
	if record := requireProjectionRecord(t, liveView.Documents, "/knowledge_base/reference/guide.md"); record.Source != "knowledge_sync" || record.Freshness != "live" {
		t.Fatalf("live document projection = %+v, want knowledge_sync/live", record)
	}
	if record := requireProjectionRecord(t, liveView.Resources, "/resources/catalog/index.json"); record.Source != "catalog_refresh" || record.Freshness != "live" {
		t.Fatalf("live resource projection = %+v, want catalog_refresh/live", record)
	}
	liveSummary, err := manager.Execute(context.Background(), session.SessionID, "cat /memory/summary.md", contract.ExecutionPolicy{})
	if err != nil {
		t.Fatalf("read refreshed summary failed: %v", err)
	}
	if !strings.Contains(liveSummary.Result.Stdout, "- live: 2") {
		t.Fatalf("live summary = %+v, want live projection count", liveSummary.Result)
	}
	refreshedGuide, err := manager.Execute(context.Background(), session.SessionID, "cat /knowledge_base/reference/guide.md", contract.ExecutionPolicy{})
	if err != nil {
		t.Fatalf("read refreshed guide failed: %v", err)
	}
	if !strings.Contains(refreshedGuide.Result.Stdout, "fresh") {
		t.Fatalf("refreshed guide = %+v, want refreshed content", refreshedGuide.Result)
	}
}

func TestReferenceAdapterProjectionMaterializationStateForStaleSlice(t *testing.T) {
	adapter := New(Options{
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
			"planning/draft-plan": "# Draft Planner\n",
		},
		SkillMetadata: map[string]SkillMetadata{
			"planning/draft-plan": {
				Source:         "workspace_catalog",
				Freshness:      "live",
				SelectionScope: "planning/default",
				Eligibility: SkillEligibility{
					State: "eligible",
				},
				Precedence: SkillPrecedence{
					Tier: "workspace",
					Rank: 1,
				},
				Selected: true,
			},
		},
	})
	manager := runtimeengine.NewSessionManager(runtimeengine.SessionManagerOptions{NewID: func() string { return "sess_projection_materialization" }})

	session, err := manager.Create(context.Background(), runtimeengine.Options{
		HostRoot: t.TempDir(),
		Profile:  contract.ProfileBashPlus,
		Policy: contract.ExecutionPolicy{
			WriteMode:        contract.WriteModeFull,
			MaxPipelineDepth: 16,
			MaxOutputBytes:   4 << 20,
			Timeout:          contract.DefaultPolicy().Timeout,
		},
		Adapters: []contract.SessionAdapter{adapter},
	})
	if err != nil {
		t.Fatalf("create session failed: %v", err)
	}

	initialProjections, err := manager.Execute(context.Background(), session.SessionID, "cat /memory/projections.json", contract.ExecutionPolicy{})
	if err != nil {
		t.Fatalf("read initial projections failed: %v", err)
	}
	initialView := decodeProjectionMaterializationView(t, []byte(initialProjections.Result.Stdout))
	assertProjectionMaterializationStatePresent(
		t,
		requireProjectionMaterializationRecord(t, initialView.Documents, "/knowledge_base/reference/guide.md"),
		"initial document projection",
	)
	assertProjectionMaterializationStatePresent(
		t,
		requireProjectionMaterializationRecord(t, initialView.Resources, "/resources/checklists/plan.json"),
		"initial resource projection",
	)
	assertProjectionMaterializationStatePresent(
		t,
		requireProjectionMaterializationRecord(t, initialView.Skills, "/skills/planning/draft-plan/SKILL.md"),
		"initial skill projection",
	)

	if _, err := manager.Close(context.Background(), session.SessionID); err != nil {
		t.Fatalf("close before invalidation failed: %v", err)
	}
	adapter.InvalidateDocument("guide.md")
	if _, err := manager.Resume(context.Background(), session.SessionID); err != nil {
		t.Fatalf("resume stale projections failed: %v", err)
	}

	staleProjections, err := manager.Execute(context.Background(), session.SessionID, "cat /memory/projections.json", contract.ExecutionPolicy{})
	if err != nil {
		t.Fatalf("read stale projections failed: %v", err)
	}
	staleView := decodeProjectionMaterializationView(t, []byte(staleProjections.Result.Stdout))
	assertProjectionMaterializationPartialOrError(
		t,
		requireProjectionMaterializationRecord(t, staleView.Documents, "/knowledge_base/reference/guide.md"),
		"stale document projection",
	)
	assertProjectionMaterializationStatePresent(
		t,
		requireProjectionMaterializationRecord(t, staleView.Resources, "/resources/checklists/plan.json"),
		"stale resource projection",
	)
	assertProjectionMaterializationStatePresent(
		t,
		requireProjectionMaterializationRecord(t, staleView.Skills, "/skills/planning/draft-plan/SKILL.md"),
		"stale skill projection",
	)
}

func TestReferenceAdapterWorkflowStatusOverrideAcrossResume(t *testing.T) {
	adapter := New(Options{
		Resources: map[string]string{
			"checklists/plan.json": "{\"steps\":[\"read\",\"write\"]}\n",
		},
		ResourceMetadata: map[string]ProjectionMetadata{
			"checklists/plan.json": {Source: "workflow_catalog", Freshness: "live"},
		},
		Workflows: []WorkflowSpec{
			{
				ID:              "draft-plan",
				Title:           "Draft plan",
				ResourcePaths:   []string{"/resources/checklists/plan.json"},
				ExpectedOutputs: []string{"/task_outputs/plan.txt"},
			},
		},
	})
	manager := runtimeengine.NewSessionManager(runtimeengine.SessionManagerOptions{NewID: func() string { return "sess_workflow_status" }})

	session, err := manager.Create(context.Background(), runtimeengine.Options{
		HostRoot: t.TempDir(),
		Profile:  contract.ProfileBashPlus,
		Policy: contract.ExecutionPolicy{
			WriteMode:        contract.WriteModeFull,
			MaxPipelineDepth: 16,
			MaxOutputBytes:   4 << 20,
			Timeout:          contract.DefaultPolicy().Timeout,
		},
		Adapters: []contract.SessionAdapter{adapter},
	})
	if err != nil {
		t.Fatalf("create session failed: %v", err)
	}
	if _, err := manager.Execute(context.Background(), session.SessionID, "cat /resources/checklists/plan.json", contract.ExecutionPolicy{}); err != nil {
		t.Fatalf("read resource failed: %v", err)
	}
	writeResult, err := manager.Execute(context.Background(), session.SessionID, "touch /task_outputs/plan.txt", contract.ExecutionPolicy{})
	if err != nil {
		t.Fatalf("write output failed: %v", err)
	}
	if writeResult.Result.ExitCode != 0 {
		t.Fatalf("write output exit code = %d, want 0", writeResult.Result.ExitCode)
	}
	if _, err := manager.Close(context.Background(), session.SessionID); err != nil {
		t.Fatalf("close before override failed: %v", err)
	}

	adapter.SetWorkflowStatus("draft-plan", "blocked", "awaiting review")
	if _, err := manager.Resume(context.Background(), session.SessionID); err != nil {
		t.Fatalf("resume with workflow override failed: %v", err)
	}
	overrideWorkflows, err := manager.Execute(context.Background(), session.SessionID, "cat /memory/workflows.json", contract.ExecutionPolicy{})
	if err != nil {
		t.Fatalf("read workflow override failed: %v", err)
	}
	overrideState := decodeWorkflowViews(t, []byte(overrideWorkflows.Result.Stdout))
	deliver := requireWorkflowView(t, overrideState, "draft-plan")
	if deliver.Status != "blocked" || deliver.StatusSource != "control_plane" || deliver.StatusReason != "awaiting review" {
		t.Fatalf("workflow override = %+v, want blocked/control_plane/awaiting review", deliver)
	}
	if !containsLine(deliver.Evidence, "/task_outputs/plan.txt") {
		t.Fatalf("workflow override evidence = %+v, want preserved output evidence", deliver.Evidence)
	}
	statusView, err := manager.Execute(context.Background(), session.SessionID, "cat /memory/status.json", contract.ExecutionPolicy{})
	if err != nil {
		t.Fatalf("read status view failed: %v", err)
	}
	statusState := decodeReferenceState(t, []byte(statusView.Result.Stdout))
	statusWorkflow := requireWorkflowView(t, statusState.Workflows, "draft-plan")
	if statusWorkflow.StatusSource != "control_plane" {
		t.Fatalf("status workflow source = %+v, want control_plane", statusWorkflow)
	}

	if _, err := manager.Close(context.Background(), session.SessionID); err != nil {
		t.Fatalf("close before clear failed: %v", err)
	}
	adapter.ClearWorkflowStatus("draft-plan")
	if _, err := manager.Resume(context.Background(), session.SessionID); err != nil {
		t.Fatalf("resume after clearing workflow override failed: %v", err)
	}
	traceWorkflows, err := manager.Execute(context.Background(), session.SessionID, "cat /memory/workflows.json", contract.ExecutionPolicy{})
	if err != nil {
		t.Fatalf("read trace workflow failed: %v", err)
	}
	traceState := decodeWorkflowViews(t, []byte(traceWorkflows.Result.Stdout))
	deliver = requireWorkflowView(t, traceState, "draft-plan")
	if deliver.Status != "completed" || deliver.StatusSource != "trace" || deliver.StatusReason != "" {
		t.Fatalf("trace workflow state = %+v, want completed/trace/blank", deliver)
	}
}

func TestReferenceAdapterSkillMetadataNormalization(t *testing.T) {
	adapter := New(Options{
		Skills: map[string]string{
			" planner/draft ":  "# Draft skill\n",
			"planner/fallback": "# Fallback skill\n",
		},
		SkillMetadata: map[string]SkillMetadata{
			"planner/draft/SKILL.md": {
				Source:    "workspace_catalog",
				Freshness: "live",
				Eligibility: SkillEligibility{
					State: "eligible",
				},
				Precedence: SkillPrecedence{
					Tier: "workspace",
					Rank: 1,
				},
				Selected: true,
			},
			"planner/fallback/SKILL.md": {
				Source:    "user_catalog",
				Freshness: "snapshot",
				Eligibility: SkillEligibility{
					State: "blocked",
				},
				Precedence: SkillPrecedence{
					Tier: "invalid",
					Rank: -2,
				},
			},
		},
	})

	records := adapter.skillRecords()
	primary := requireProjectionRecord(t, records, "/skills/planner/draft/SKILL.md")
	assertSkillProjectionMetadata(t, primary, "workspace_catalog", "live", "eligible", "", "workspace", 1, true)

	fallback := requireProjectionRecord(t, records, "/skills/planner/fallback/SKILL.md")
	assertSkillProjectionMetadata(t, fallback, "user_catalog", "snapshot", "unknown", "", "bundled", 0, false)
}

func TestReferenceAdapterSkillsProjectionReadOnlyAcrossResume(t *testing.T) {
	adapter := New(Options{
		Skills: map[string]string{
			"planning/draft-plan": "# Draft Planner\nUse deterministic plan steps.\n",
			"planning/fallback":   "# Fallback Planner\nUse manual checklist fallback.\n",
		},
		SkillMetadata: map[string]SkillMetadata{
			"planning/draft-plan": {
				Source:         "workspace_catalog",
				Freshness:      "live",
				SelectionScope: "planning/default",
				Eligibility: SkillEligibility{
					State: "eligible",
				},
				Precedence: SkillPrecedence{
					Tier: "workspace",
					Rank: 1,
				},
				Selected: true,
			},
			"planning/fallback": {
				Source:         "bundled_catalog",
				Freshness:      "snapshot",
				SelectionScope: "planning/default",
				Eligibility: SkillEligibility{
					State:  "ineligible",
					Reason: "missing_env:PLAN_FALLBACK_TOKEN",
				},
				Precedence: SkillPrecedence{
					Tier: "bundled",
					Rank: 90,
				},
			},
		},
	})
	manager := runtimeengine.NewSessionManager(runtimeengine.SessionManagerOptions{NewID: func() string { return "sess_skills" }})

	session, err := manager.Create(context.Background(), runtimeengine.Options{
		HostRoot: t.TempDir(),
		Profile:  contract.ProfileBashPlus,
		Policy: contract.ExecutionPolicy{
			WriteMode:        contract.WriteModeFull,
			MaxPipelineDepth: 16,
			MaxOutputBytes:   4 << 20,
			Timeout:          contract.DefaultPolicy().Timeout,
		},
		Adapters: []contract.SessionAdapter{adapter},
	})
	if err != nil {
		t.Fatalf("create session failed: %v", err)
	}

	readSkill, err := manager.Execute(context.Background(), session.SessionID, "cat /skills/planning/draft-plan/SKILL.md", contract.ExecutionPolicy{})
	if err != nil {
		t.Fatalf("read skill projection failed: %v", err)
	}
	if !strings.Contains(readSkill.Result.Stdout, "Draft Planner") {
		t.Fatalf("unexpected skill output: %+v", readSkill.Result)
	}
	skillIndex, err := manager.Execute(context.Background(), session.SessionID, "cat /skills/_index.json", contract.ExecutionPolicy{})
	if err != nil {
		t.Fatalf("read skills index failed: %v", err)
	}
	indexSkills := decodeProjectionRecords(t, []byte(skillIndex.Result.Stdout))
	draftSkill := requireProjectionRecord(t, indexSkills, "/skills/planning/draft-plan/SKILL.md")
	assertSkillProjectionMetadata(t, draftSkill, "workspace_catalog", "live", "eligible", "", "workspace", 1, true)
	fallbackSkill := requireProjectionRecord(t, indexSkills, "/skills/planning/fallback/SKILL.md")
	assertSkillProjectionMetadata(t, fallbackSkill, "bundled_catalog", "snapshot", "ineligible", "missing_env:PLAN_FALLBACK_TOKEN", "bundled", 90, false)

	projections, err := manager.Execute(context.Background(), session.SessionID, "cat /memory/projections.json", contract.ExecutionPolicy{})
	if err != nil {
		t.Fatalf("read projection view failed: %v", err)
	}
	projectionView := decodeProjectionView(t, []byte(projections.Result.Stdout))
	projectedDraft := requireProjectionRecord(t, projectionView.Skills, "/skills/planning/draft-plan/SKILL.md")
	assertSkillProjectionMetadata(t, projectedDraft, "workspace_catalog", "live", "eligible", "", "workspace", 1, true)
	projectedFallback := requireProjectionRecord(t, projectionView.Skills, "/skills/planning/fallback/SKILL.md")
	assertSkillProjectionMetadata(t, projectedFallback, "bundled_catalog", "snapshot", "ineligible", "missing_env:PLAN_FALLBACK_TOKEN", "bundled", 90, false)

	summary, err := manager.Execute(context.Background(), session.SessionID, "cat /memory/summary.md", contract.ExecutionPolicy{})
	if err != nil {
		t.Fatalf("read memory summary failed: %v", err)
	}
	if !strings.Contains(summary.Result.Stdout, "projections.skills: 2") || !strings.Contains(summary.Result.Stdout, "skill_reads: 2") {
		t.Fatalf("expected skill projection count in memory summary, got %+v", summary.Result)
	}
	skillsEvidence, err := manager.Execute(context.Background(), session.SessionID, "cat /memory/skills.md", contract.ExecutionPolicy{})
	if err != nil {
		t.Fatalf("read skills evidence failed: %v", err)
	}
	if !strings.Contains(skillsEvidence.Result.Stdout, "/skills/planning/draft-plan/SKILL.md") {
		t.Fatalf("expected skill read evidence, got %+v", skillsEvidence.Result)
	}

	deniedWrite, err := manager.Execute(context.Background(), session.SessionID, "echo blocked > /skills/planning/draft-plan/SKILL.md", contract.ExecutionPolicy{})
	if err != nil {
		t.Fatalf("write to /skills failed unexpectedly: %v", err)
	}
	if deniedWrite.Result.ExitCode == 0 {
		t.Fatalf("expected write to /skills to be denied, got %+v", deniedWrite.Result)
	}
	observations, err := manager.Execute(context.Background(), session.SessionID, "cat /memory/observations.md", contract.ExecutionPolicy{})
	if err != nil {
		t.Fatalf("read observations failed: %v", err)
	}
	if !strings.Contains(observations.Result.Stdout, "denied:/skills/planning/draft-plan/SKILL.md") {
		t.Fatalf("expected denied /skills observation, got %+v", observations.Result)
	}
	if !strings.Contains(observations.Result.Stdout, "read-skill:/skills/planning/draft-plan/SKILL.md") {
		t.Fatalf("expected read /skills observation, got %+v", observations.Result)
	}

	if _, err := manager.Close(context.Background(), session.SessionID); err != nil {
		t.Fatalf("close session failed: %v", err)
	}
	if _, err := manager.Resume(context.Background(), session.SessionID); err != nil {
		t.Fatalf("resume session failed: %v", err)
	}
	resumedIndex, err := manager.Execute(context.Background(), session.SessionID, "cat /skills/_index.json", contract.ExecutionPolicy{})
	if err != nil {
		t.Fatalf("read resumed skills index failed: %v", err)
	}
	resumedSkills := decodeProjectionRecords(t, []byte(resumedIndex.Result.Stdout))
	resumedDraft := requireProjectionRecord(t, resumedSkills, "/skills/planning/draft-plan/SKILL.md")
	assertSkillProjectionMetadata(t, resumedDraft, "workspace_catalog", "live", "eligible", "", "workspace", 1, true)
}

func TestReferenceAdapterSkillsSelectionTruthAcrossResume(t *testing.T) {
	adapter := New(Options{
		Skills: map[string]string{
			"planning/draft-plan": "# Draft Planner\n",
			"planning/alternate":  "# Alternate Planner\n",
			"planning/fallback":   "# Fallback Planner\n",
		},
		SkillMetadata: map[string]SkillMetadata{
			"planning/draft-plan": {
				Source:         "workspace_catalog",
				Freshness:      "live",
				SelectionScope: "planning/default",
				Eligibility: SkillEligibility{
					State: "eligible",
				},
				Precedence: SkillPrecedence{
					Tier: "workspace",
					Rank: 1,
				},
				Selected: false,
			},
			"planning/alternate": {
				Source:         "workspace_catalog",
				Freshness:      "live",
				SelectionScope: "planning/default",
				Eligibility: SkillEligibility{
					State: "eligible",
				},
				Precedence: SkillPrecedence{
					Tier: "workspace",
					Rank: 5,
				},
				Selected: true,
			},
			"planning/fallback": {
				Source:         "bundled_catalog",
				Freshness:      "snapshot",
				SelectionScope: "planning/default",
				Eligibility: SkillEligibility{
					State:  "ineligible",
					Reason: "missing_env:PLAN_FALLBACK_TOKEN",
				},
				Precedence: SkillPrecedence{
					Tier: "bundled",
					Rank: 90,
				},
				Selected: true,
			},
		},
	})
	manager := runtimeengine.NewSessionManager(runtimeengine.SessionManagerOptions{NewID: func() string { return "sess_skills_selection_truth" }})
	session, err := manager.Create(context.Background(), runtimeengine.Options{
		HostRoot: t.TempDir(),
		Profile:  contract.ProfileBashPlus,
		Policy: contract.ExecutionPolicy{
			WriteMode:        contract.WriteModeFull,
			MaxPipelineDepth: 16,
			MaxOutputBytes:   4 << 20,
			Timeout:          contract.DefaultPolicy().Timeout,
		},
		Adapters: []contract.SessionAdapter{adapter},
	})
	if err != nil {
		t.Fatalf("create session failed: %v", err)
	}

	skillIndex, err := manager.Execute(context.Background(), session.SessionID, "cat /skills/_index.json", contract.ExecutionPolicy{})
	if err != nil {
		t.Fatalf("read skills index failed: %v", err)
	}
	indexSkills := decodeProjectionRecords(t, []byte(skillIndex.Result.Stdout))
	draftSkill := requireProjectionRecord(t, indexSkills, "/skills/planning/draft-plan/SKILL.md")
	alternateSkill := requireProjectionRecord(t, indexSkills, "/skills/planning/alternate/SKILL.md")
	fallbackSkill := requireProjectionRecord(t, indexSkills, "/skills/planning/fallback/SKILL.md")
	assertSkillProjectionMetadata(t, draftSkill, "workspace_catalog", "live", "eligible", "", "workspace", 1, true)
	assertSkillProjectionMetadata(t, alternateSkill, "workspace_catalog", "live", "eligible", "", "workspace", 5, false)
	assertSkillProjectionMetadata(t, fallbackSkill, "bundled_catalog", "snapshot", "ineligible", "missing_env:PLAN_FALLBACK_TOKEN", "bundled", 90, false)

	indexSkillMaps := decodeProjectionRecordMaps(t, []byte(skillIndex.Result.Stdout))
	draftMap := requireProjectionRecordMap(t, indexSkillMaps, "/skills/planning/draft-plan/SKILL.md")
	alternateMap := requireProjectionRecordMap(t, indexSkillMaps, "/skills/planning/alternate/SKILL.md")
	fallbackMap := requireProjectionRecordMap(t, indexSkillMaps, "/skills/planning/fallback/SKILL.md")

	draftSelection := requireSelectionMap(t, draftMap)
	alternateSelection := requireSelectionMap(t, alternateMap)
	fallbackSelection := requireSelectionMap(t, fallbackMap)

	draftScope := mapStringField(t, draftSelection, "scope")
	alternateScope := mapStringField(t, alternateSelection, "scope")
	fallbackScope := mapStringField(t, fallbackSelection, "scope")
	if draftScope == "" || alternateScope == "" || fallbackScope == "" {
		t.Fatalf("selection scope must be explicit for all skills, got draft=%q alternate=%q fallback=%q", draftScope, alternateScope, fallbackScope)
	}
	if draftScope != alternateScope || draftScope != fallbackScope {
		t.Fatalf("expected same explicit selection scope across planning skills, got draft=%q alternate=%q fallback=%q", draftScope, alternateScope, fallbackScope)
	}
	if mapStringField(t, draftSelection, "mode") == "" || mapStringField(t, alternateSelection, "mode") == "" || mapStringField(t, fallbackSelection, "mode") == "" {
		t.Fatalf("selection mode must be machine-readable for winner/loser/ineligible skills")
	}
	if mapStringField(t, alternateSelection, "reason") == "" {
		t.Fatalf("loser skill must expose machine-readable selection reason: %+v", alternateSelection)
	}
	if mapStringField(t, fallbackSelection, "reason") == "" {
		t.Fatalf("ineligible skill must expose machine-readable selection reason: %+v", fallbackSelection)
	}

	projections, err := manager.Execute(context.Background(), session.SessionID, "cat /memory/projections.json", contract.ExecutionPolicy{})
	if err != nil {
		t.Fatalf("read /memory/projections.json failed: %v", err)
	}
	projectionView := decodeProjectionView(t, []byte(projections.Result.Stdout))
	projectedDraft := requireProjectionRecord(t, projectionView.Skills, "/skills/planning/draft-plan/SKILL.md")
	projectedAlternate := requireProjectionRecord(t, projectionView.Skills, "/skills/planning/alternate/SKILL.md")
	projectedFallback := requireProjectionRecord(t, projectionView.Skills, "/skills/planning/fallback/SKILL.md")
	if !projectedDraft.Selected || projectedAlternate.Selected || projectedFallback.Selected {
		t.Fatalf("projection view selected states mismatch: draft=%v alternate=%v fallback=%v", projectedDraft.Selected, projectedAlternate.Selected, projectedFallback.Selected)
	}

	if _, err := manager.Close(context.Background(), session.SessionID); err != nil {
		t.Fatalf("close session failed: %v", err)
	}
	if _, err := manager.Resume(context.Background(), session.SessionID); err != nil {
		t.Fatalf("resume session failed: %v", err)
	}
	resumedIndex, err := manager.Execute(context.Background(), session.SessionID, "cat /skills/_index.json", contract.ExecutionPolicy{})
	if err != nil {
		t.Fatalf("read resumed skills index failed: %v", err)
	}
	resumedSkillMaps := decodeProjectionRecordMaps(t, []byte(resumedIndex.Result.Stdout))
	resumedDraft := requireProjectionRecordMap(t, resumedSkillMaps, "/skills/planning/draft-plan/SKILL.md")
	resumedAlternate := requireProjectionRecordMap(t, resumedSkillMaps, "/skills/planning/alternate/SKILL.md")
	resumedFallback := requireProjectionRecordMap(t, resumedSkillMaps, "/skills/planning/fallback/SKILL.md")
	if !mapBoolField(resumedDraft, "selected") || mapBoolField(resumedAlternate, "selected") || mapBoolField(resumedFallback, "selected") {
		t.Fatalf("selection state should remain stable across resume: draft=%v alternate=%v fallback=%v", mapBoolField(resumedDraft, "selected"), mapBoolField(resumedAlternate, "selected"), mapBoolField(resumedFallback, "selected"))
	}

	if mapStringField(t, requireSelectionMap(t, resumedDraft), "scope") != draftScope {
		t.Fatalf("winner scope drift across resume: before=%q after=%q", draftScope, mapStringField(t, requireSelectionMap(t, resumedDraft), "scope"))
	}
	if mapStringField(t, requireSelectionMap(t, resumedAlternate), "scope") != alternateScope {
		t.Fatalf("loser scope drift across resume: before=%q after=%q", alternateScope, mapStringField(t, requireSelectionMap(t, resumedAlternate), "scope"))
	}
	if mapStringField(t, requireSelectionMap(t, resumedFallback), "scope") != fallbackScope {
		t.Fatalf("ineligible scope drift across resume: before=%q after=%q", fallbackScope, mapStringField(t, requireSelectionMap(t, resumedFallback), "scope"))
	}
}

func TestReferenceAdapterSkillControlPlaneLifecycle(t *testing.T) {
	const selectionScope = "planning/default"
	adapter := New(Options{
		Skills: map[string]string{
			"planning/draft-plan": "# Draft Planner\n",
			"planning/alternate":  "# Alternate Planner\n",
		},
		SkillMetadata: map[string]SkillMetadata{
			"planning/draft-plan": {
				Source:         "workspace_catalog",
				Freshness:      "live",
				SelectionScope: selectionScope,
				Eligibility: SkillEligibility{
					State: skillEligibilityEligible,
				},
				Precedence: SkillPrecedence{
					Tier: skillPrecedenceTierWorkspace,
					Rank: 1,
				},
			},
			"planning/alternate": {
				Source:         "workspace_catalog",
				Freshness:      "live",
				SelectionScope: selectionScope,
				Eligibility: SkillEligibility{
					State: skillEligibilityEligible,
				},
				Precedence: SkillPrecedence{
					Tier: skillPrecedenceTierWorkspace,
					Rank: 5,
				},
			},
		},
	})
	manager := runtimeengine.NewSessionManager(runtimeengine.SessionManagerOptions{NewID: func() string { return "sess_skill_control_plane" }})
	session, err := manager.Create(context.Background(), runtimeengine.Options{
		HostRoot: t.TempDir(),
		Profile:  contract.ProfileBashPlus,
		Policy: contract.ExecutionPolicy{
			WriteMode:        contract.WriteModeFull,
			MaxPipelineDepth: 16,
			MaxOutputBytes:   4 << 20,
			Timeout:          contract.DefaultPolicy().Timeout,
		},
		Adapters: []contract.SessionAdapter{adapter},
	})
	if err != nil {
		t.Fatalf("create session failed: %v", err)
	}

	initialIndex, err := manager.Execute(context.Background(), session.SessionID, "cat /skills/_index.json", contract.ExecutionPolicy{})
	if err != nil {
		t.Fatalf("read initial skills index failed: %v", err)
	}
	initialRecords := decodeProjectionRecords(t, []byte(initialIndex.Result.Stdout))
	draftInitial := requireProjectionRecord(t, initialRecords, "/skills/planning/draft-plan/SKILL.md")
	if !draftInitial.Selected {
		t.Fatalf("expected draft skill selected initially, got %+v", draftInitial)
	}
	if draftInitial.Selection == nil || draftInitial.Selection.Scope != selectionScope {
		t.Fatalf("initial draft skill selection missing or wrong scope: %+v", draftInitial.Selection)
	}
	alternateInitial := requireProjectionRecord(t, initialRecords, "/skills/planning/alternate/SKILL.md")
	if alternateInitial.Selected {
		t.Fatalf("alternate skill should not be selected initially: %+v", alternateInitial)
	}

	adapter.UpsertSkill("planning/priority", "# Priority Planner\n", SkillMetadata{
		Source:         "control_plane",
		Freshness:      projectionFreshnessLive,
		SelectionScope: selectionScope,
		Eligibility: SkillEligibility{
			State: skillEligibilityEligible,
		},
		Precedence: SkillPrecedence{
			Tier: skillPrecedenceTierWorkspace,
			Rank: 0,
		},
	})
	if _, err := manager.Close(context.Background(), session.SessionID); err != nil {
		t.Fatalf("close after skill upsert failed: %v", err)
	}
	if _, err := manager.Resume(context.Background(), session.SessionID); err != nil {
		t.Fatalf("resume after skill upsert failed: %v", err)
	}
	postAddIndex, err := manager.Execute(context.Background(), session.SessionID, "cat /skills/_index.json", contract.ExecutionPolicy{})
	if err != nil {
		t.Fatalf("read skills index after addition failed: %v", err)
	}
	postAddRecords := decodeProjectionRecords(t, []byte(postAddIndex.Result.Stdout))
	priorityRecord := requireProjectionRecord(t, postAddRecords, "/skills/planning/priority/SKILL.md")
	if !priorityRecord.Selected {
		t.Fatalf("priority skill should be selected after addition: %+v", priorityRecord)
	}
	if priorityRecord.Selection == nil || priorityRecord.Selection.Scope != selectionScope {
		t.Fatalf("priority selection scope wrong: %+v", priorityRecord.Selection)
	}
	if priorityRecord.Selection.Mode != skillSelectionModeDerived {
		t.Fatalf("expected derived selection mode for priority skill, got %+v", priorityRecord.Selection.Mode)
	}
	if priorityRecord.Selection.Reason != skillSelectionReasonHighestPrecedence {
		t.Fatalf("expected highest precedence reason, got %+v", priorityRecord.Selection.Reason)
	}
	if priorityRecord.Source != "control_plane" || priorityRecord.Freshness != projectionFreshnessLive {
		t.Fatalf("priority projection metadata mismatch: %+v", priorityRecord)
	}
	if priorityRecord.Precedence == nil || priorityRecord.Precedence.Tier != skillPrecedenceTierWorkspace || priorityRecord.Precedence.Rank != 0 {
		t.Fatalf("priority precedence metadata wrong: %+v", priorityRecord.Precedence)
	}
	draftAfterAdd := requireProjectionRecord(t, postAddRecords, "/skills/planning/draft-plan/SKILL.md")
	if draftAfterAdd.Selected {
		t.Fatalf("draft skill should yield selection to priority, got %+v", draftAfterAdd)
	}
	if draftAfterAdd.Selection == nil || draftAfterAdd.Selection.Scope != selectionScope {
		t.Fatalf("draft selection scope drifted: %+v", draftAfterAdd.Selection)
	}

	deniedWrite, err := manager.Execute(context.Background(), session.SessionID, "echo blocked > /skills/planning/priority/SKILL.md", contract.ExecutionPolicy{})
	if err != nil {
		t.Fatalf("write to /skills failed unexpectedly: %v", err)
	}
	if deniedWrite.Result.ExitCode == 0 {
		t.Fatalf("expected /skills write to be denied, got %+v", deniedWrite.Result)
	}

	adapter.UpdateSkill("planning/priority", "# Priority Planner\nUpdated content.\n")
	if _, err := manager.Close(context.Background(), session.SessionID); err != nil {
		t.Fatalf("close after skill update failed: %v", err)
	}
	if _, err := manager.Resume(context.Background(), session.SessionID); err != nil {
		t.Fatalf("resume after skill update failed: %v", err)
	}
	updatedSkill, err := manager.Execute(context.Background(), session.SessionID, "cat /skills/planning/priority/SKILL.md", contract.ExecutionPolicy{})
	if err != nil {
		t.Fatalf("read updated priority skill failed: %v", err)
	}
	if !strings.Contains(updatedSkill.Result.Stdout, "Updated content") {
		t.Fatalf("updated priority skill missing refreshed indicator: %+v", updatedSkill.Result)
	}
	updatedIndex, err := manager.Execute(context.Background(), session.SessionID, "cat /skills/_index.json", contract.ExecutionPolicy{})
	if err != nil {
		t.Fatalf("read skills index after update failed: %v", err)
	}
	updatedRecords := decodeProjectionRecords(t, []byte(updatedIndex.Result.Stdout))
	updatedPriority := requireProjectionRecord(t, updatedRecords, "/skills/planning/priority/SKILL.md")
	if updatedPriority.Freshness != projectionFreshnessUpdated {
		t.Fatalf("expected updated freshness after UpdateSkill, got %+v", updatedPriority.Freshness)
	}
	if updatedPriority.Selection == nil || updatedPriority.Selection.Scope != selectionScope {
		t.Fatalf("updated priority selection incompatible: %+v", updatedPriority.Selection)
	}

	adapter.RemoveSkill("planning/priority")
	if _, err := manager.Close(context.Background(), session.SessionID); err != nil {
		t.Fatalf("close after skill removal failed: %v", err)
	}
	if _, err := manager.Resume(context.Background(), session.SessionID); err != nil {
		t.Fatalf("resume after skill removal failed: %v", err)
	}
	afterRemovalIndex, err := manager.Execute(context.Background(), session.SessionID, "cat /skills/_index.json", contract.ExecutionPolicy{})
	if err != nil {
		t.Fatalf("read skills index after removal failed: %v", err)
	}
	afterRemovalRecords := decodeProjectionRecords(t, []byte(afterRemovalIndex.Result.Stdout))
	for _, record := range afterRemovalRecords {
		if record.Path == "/skills/planning/priority/SKILL.md" {
			t.Fatalf("priority skill still present after removal: %+v", record)
		}
	}
	draftAfterRemoval := requireProjectionRecord(t, afterRemovalRecords, "/skills/planning/draft-plan/SKILL.md")
	if !draftAfterRemoval.Selected {
		t.Fatalf("draft skill should regain selection after removal, got %+v", draftAfterRemoval)
	}
	if draftAfterRemoval.Selection == nil || draftAfterRemoval.Selection.Scope != selectionScope {
		t.Fatalf("draft selection scope not preserved after removal: %+v", draftAfterRemoval.Selection)
	}

	if _, err := manager.Close(context.Background(), session.SessionID); err != nil {
		t.Fatalf("close session failed: %v", err)
	}
	if _, err := manager.Resume(context.Background(), session.SessionID); err != nil {
		t.Fatalf("resume session failed: %v", err)
	}
	resumedIndex, err := manager.Execute(context.Background(), session.SessionID, "cat /skills/_index.json", contract.ExecutionPolicy{})
	if err != nil {
		t.Fatalf("read resumed skills index failed: %v", err)
	}
	resumedRecords := decodeProjectionRecords(t, []byte(resumedIndex.Result.Stdout))
	for _, record := range resumedRecords {
		if record.Path == "/skills/planning/priority/SKILL.md" {
			t.Fatalf("priority skill reappeared after resume: %+v", record)
		}
	}
	resumedDraft := requireProjectionRecord(t, resumedRecords, "/skills/planning/draft-plan/SKILL.md")
	if !resumedDraft.Selected {
		t.Fatalf("draft skill selection lost after resume: %+v", resumedDraft)
	}
	if resumedDraft.Selection == nil || resumedDraft.Selection.Scope != selectionScope {
		t.Fatalf("draft selection scope drifted after resume: %+v", resumedDraft.Selection)
	}
}

func TestReferenceAdapterSkillControlPlaneAuditEvents(t *testing.T) {
	const selectionScope = "planning/default"
	const skillPath = "/skills/planning/live-hotfix/SKILL.md"

	adapter := New(Options{})
	manager := runtimeengine.NewSessionManager(runtimeengine.SessionManagerOptions{
		NewID: func() string { return "sess_skill_control_plane_audit" },
	})
	session, err := manager.Create(context.Background(), runtimeengine.Options{
		HostRoot: t.TempDir(),
		Profile:  contract.ProfileBashPlus,
		Policy: contract.ExecutionPolicy{
			WriteMode:        contract.WriteModeFull,
			MaxPipelineDepth: 16,
			MaxOutputBytes:   4 << 20,
			Timeout:          contract.DefaultPolicy().Timeout,
		},
		Adapters: []contract.SessionAdapter{adapter},
	})
	if err != nil {
		t.Fatalf("create session failed: %v", err)
	}

	if _, err := manager.Execute(context.Background(), session.SessionID, "cat /memory/status.json", contract.ExecutionPolicy{}); err != nil {
		t.Fatalf("initial memory status failed: %v", err)
	}

	adapter.UpsertSkill("planning/live-hotfix", "# Live hotfix skill\n", SkillMetadata{
		Source:         "control_plane",
		Freshness:      "updated",
		SelectionScope: selectionScope,
		Eligibility: SkillEligibility{
			State: skillEligibilityEligible,
		},
		Precedence: SkillPrecedence{
			Tier: skillPrecedenceTierWorkspace,
			Rank: 0,
		},
	})
	adapter.UpdateSkill("planning/live-hotfix", "# Live hotfix skill\nUpdated guidance.\n")
	adapter.RemoveSkill("planning/live-hotfix")

	skillsIndex, err := manager.Execute(context.Background(), session.SessionID, "cat /skills/_index.json", contract.ExecutionPolicy{})
	if err != nil {
		t.Fatalf("read skills index for removal check failed: %v", err)
	}
	if skillsIndex.Result.ExitCode == 0 {
		t.Fatalf("/skills/_index.json should be missing after removal, got %q", skillsIndex.Result.Stdout)
	}

	auditResp, err := manager.Execute(context.Background(), session.SessionID, "cat /memory/skills_audit.json", contract.ExecutionPolicy{})
	if err != nil {
		t.Fatalf("read skills audit view failed: %v", err)
	}
	if auditResp.Result.ExitCode != 0 {
		t.Fatalf("skills audit command failed: %+v", auditResp.Result)
	}
	events := decodeControlPlaneEventViews(t, []byte(auditResp.Result.Stdout))
	if len(events) != 3 {
		t.Fatalf("expected 3 control-plane events, got %d", len(events))
	}

	added := events[0]
	if added.Op != controlPlaneEventKindSkillAdded || added.Path != skillPath {
		t.Fatalf("unexpected add event = %+v", added)
	}
	if added.SelectionScope != selectionScope {
		t.Fatalf("added selection scope = %q, want %q", added.SelectionScope, selectionScope)
	}
	if added.SelectedBefore || !added.SelectedAfter {
		t.Fatalf("added selection transition = %+v", added)
	}
	if added.Result != "applied" || added.VisibleAfter != controlPlaneVisibilityNextProjection {
		t.Fatalf("added event metadata wrong = %+v", added)
	}
	if added.WinnerAfter != skillPath {
		t.Fatalf("added winner path = %q, want %q", added.WinnerAfter, skillPath)
	}

	updated := events[1]
	if updated.Op != controlPlaneEventKindSkillUpdated {
		t.Fatalf("expected update event second, got %+v", updated)
	}
	if !updated.SelectedBefore || !updated.SelectedAfter {
		t.Fatalf("update selection transition = %+v", updated)
	}
	if updated.WinnerBefore != skillPath || updated.WinnerAfter != skillPath {
		t.Fatalf("update winner tracking = %+v", updated)
	}

	removed := events[2]
	if removed.Op != controlPlaneEventKindSkillRemoved {
		t.Fatalf("expected removal third, got %+v", removed)
	}
	if removed.SelectedAfter {
		t.Fatalf("removal should leave selection false, got %+v", removed)
	}
	if removed.WinnerBefore != skillPath {
		t.Fatalf("removal winner_before = %q, want %q", removed.WinnerBefore, skillPath)
	}
	if removed.VisibleAfter != controlPlaneVisibilityNextProjection {
		t.Fatalf("removal visibility metadata = %+v", removed)
	}

	for i := 1; i < len(events); i++ {
		if events[i].VisibleFromGeneration < events[i-1].VisibleFromGeneration {
			t.Fatalf("visible_from_generation should be non-decreasing: %v vs %v", events[i-1], events[i])
		}
	}
	for _, event := range events {
		if event.Result != "applied" || event.VisibleAfter != controlPlaneVisibilityNextProjection {
			t.Fatalf("event metadata corrupt: %+v", event)
		}
	}

	metricsResp, err := manager.Execute(context.Background(), session.SessionID, "cat /memory/projection_metrics.json", contract.ExecutionPolicy{})
	if err != nil {
		t.Fatalf("read projection metrics failed: %v", err)
	}
	metrics := decodeProjectionMetricsView(t, []byte(metricsResp.Result.Stdout))
	denialsResp, err := manager.Execute(context.Background(), session.SessionID, "cat /memory/denials.json", contract.ExecutionPolicy{})
	if err != nil {
		t.Fatalf("read denials view failed: %v", err)
	}
	denials := decodeDenialView(t, []byte(denialsResp.Result.Stdout))

	snapshot, err := manager.Get(session.SessionID)
	if err != nil {
		t.Fatalf("snapshot failed: %v", err)
	}
	state := decodeReferenceState(t, snapshot.State.Opaque[adapter.AdapterID()])
	if state.ControlPlaneEvents != len(events) {
		t.Fatalf("state reports %d events, want %d", state.ControlPlaneEvents, len(events))
	}
	if state.LastControlPlaneKind != removed.Op {
		t.Fatalf("state last kind = %q, want %q", state.LastControlPlaneKind, removed.Op)
	}
	if state.ProjectionGeneration < removed.VisibleFromGeneration {
		t.Fatalf("state projection generation = %d, want >=%d", state.ProjectionGeneration, removed.VisibleFromGeneration)
	}
	if metrics.ControlPlaneEvents != len(events) || metrics.ProjectionCounts["skills"] != 0 || metrics.ProjectionGeneration > state.ProjectionGeneration {
		t.Fatalf("projection metrics = %+v, want events=%d skills=0 generation<=%d", metrics, len(events), state.ProjectionGeneration)
	}
	if metrics.CacheHitMetricsAvailable {
		t.Fatalf("projection metrics cache flag = %+v, want unavailable", metrics)
	}
	if denials.UniqueDeniedPaths != 0 || denials.ProjectionGeneration > state.ProjectionGeneration {
		t.Fatalf("denials view = %+v, want generation<=%d unique=0", denials, state.ProjectionGeneration)
	}

	summaryResp, err := manager.Execute(context.Background(), session.SessionID, "cat /memory/summary.md", contract.ExecutionPolicy{})
	if err != nil {
		t.Fatalf("read memory summary failed: %v", err)
	}
	summaryText := summaryResp.Result.Stdout
	if count, ok := summaryLineInt(summaryText, "control_plane_events"); !ok {
		t.Fatalf("summary missing control_plane_events line")
	} else if count != len(events) {
		t.Fatalf("summary control_plane_events = %d, want %d", count, len(events))
	}
	expectedLast := removed.Op + " " + removed.Path
	if lastValue, ok := summaryLineValue(summaryText, "last_control_plane_event"); !ok {
		t.Fatalf("summary missing last_control_plane_event line")
	} else if lastValue != expectedLast {
		t.Fatalf("summary last_control_plane_event = %q, want %q", lastValue, expectedLast)
	}

	metricsSummaryResp, err := manager.Execute(context.Background(), session.SessionID, "cat /memory/projection_metrics.md", contract.ExecutionPolicy{})
	if err != nil {
		t.Fatalf("read projection metrics summary failed: %v", err)
	}
	if !strings.Contains(metricsSummaryResp.Result.Stdout, "projection_generation: ") || !strings.Contains(metricsSummaryResp.Result.Stdout, "cache_hit_metrics_available: false") {
		t.Fatalf("projection metrics summary = %q, want generation and cache flag", metricsSummaryResp.Result.Stdout)
	}
}

func TestReferenceAdapterSkillSelectionTieBreakDeterministic(t *testing.T) {
	adapter := New(Options{
		Skills: map[string]string{
			"triage/alpha": "# Triage Alpha\n",
			"triage/beta":  "# Triage Beta\n",
		},
		SkillMetadata: map[string]SkillMetadata{
			"triage/alpha": {
				Source:         "workspace_catalog",
				Freshness:      "live",
				SelectionScope: "triage/default",
				Eligibility: SkillEligibility{
					State: "eligible",
				},
				Precedence: SkillPrecedence{
					Tier: "workspace",
					Rank: 1,
				},
				Selected: false,
			},
			"triage/beta": {
				Source:         "workspace_catalog",
				Freshness:      "live",
				SelectionScope: "triage/default",
				Eligibility: SkillEligibility{
					State: "eligible",
				},
				Precedence: SkillPrecedence{
					Tier: "workspace",
					Rank: 1,
				},
				Selected: false,
			},
		},
	})
	manager := runtimeengine.NewSessionManager(runtimeengine.SessionManagerOptions{NewID: func() string { return "sess_skills_tie_break" }})
	session, err := manager.Create(context.Background(), runtimeengine.Options{
		HostRoot: t.TempDir(),
		Profile:  contract.ProfileBashPlus,
		Policy: contract.ExecutionPolicy{
			WriteMode:        contract.WriteModeFull,
			MaxPipelineDepth: 16,
			MaxOutputBytes:   4 << 20,
			Timeout:          contract.DefaultPolicy().Timeout,
		},
		Adapters: []contract.SessionAdapter{adapter},
	})
	if err != nil {
		t.Fatalf("create session failed: %v", err)
	}

	firstWinner, firstLoser, firstScope := readTieBreakSnapshot(t, manager, session.SessionID)
	if firstWinner == "" || firstLoser == "" {
		t.Fatalf("expected exactly one winner and one loser, got winner=%q loser=%q", firstWinner, firstLoser)
	}
	if firstScope == "" {
		t.Fatalf("expected explicit competition scope for tie-break selection")
	}

	for idx := 0; idx < 2; idx++ {
		if _, err := manager.Close(context.Background(), session.SessionID); err != nil {
			t.Fatalf("close #%d failed: %v", idx+1, err)
		}
		if _, err := manager.Resume(context.Background(), session.SessionID); err != nil {
			t.Fatalf("resume #%d failed: %v", idx+1, err)
		}
		winner, loser, scope := readTieBreakSnapshot(t, manager, session.SessionID)
		if winner != firstWinner || loser != firstLoser {
			t.Fatalf("tie-break selection drifted across resume #%d: first=(%q,%q) now=(%q,%q)", idx+1, firstWinner, firstLoser, winner, loser)
		}
		if scope != firstScope {
			t.Fatalf("tie-break selection scope drifted across resume #%d: first=%q now=%q", idx+1, firstScope, scope)
		}
	}
}

func TestReferenceAdapterSkillControlPlaneLifecycleAcrossResume(t *testing.T) {
	adapter := New(Options{
		Skills: map[string]string{
			"planning/fallback": "# Fallback Planner\n",
		},
		SkillMetadata: map[string]SkillMetadata{
			"planning/fallback": {
				Source:         "bundled_catalog",
				Freshness:      "snapshot",
				SelectionScope: "planning/default",
				Eligibility:    SkillEligibility{State: "eligible"},
				Precedence:     SkillPrecedence{Tier: "bundled", Rank: 90},
			},
		},
	})
	manager := runtimeengine.NewSessionManager(runtimeengine.SessionManagerOptions{NewID: func() string { return "sess_skill_control_plane" }})
	session, err := manager.Create(context.Background(), runtimeengine.Options{
		HostRoot: t.TempDir(),
		Profile:  contract.ProfileBashPlus,
		Policy: contract.ExecutionPolicy{
			WriteMode:        contract.WriteModeFull,
			MaxPipelineDepth: 16,
			MaxOutputBytes:   4 << 20,
			Timeout:          contract.DefaultPolicy().Timeout,
		},
		Adapters: []contract.SessionAdapter{adapter},
	})
	if err != nil {
		t.Fatalf("create session failed: %v", err)
	}

	initialIndex, err := manager.Execute(context.Background(), session.SessionID, "cat /skills/_index.json", contract.ExecutionPolicy{})
	if err != nil {
		t.Fatalf("read initial skills index failed: %v", err)
	}
	initialSkills := decodeProjectionRecords(t, []byte(initialIndex.Result.Stdout))
	fallback := requireProjectionRecord(t, initialSkills, "/skills/planning/fallback/SKILL.md")
	assertSkillProjectionMetadata(t, fallback, "bundled_catalog", "snapshot", "eligible", "", "bundled", 90, true)

	adapter.UpsertSkill("planning/primary", "# Primary Planner\n", SkillMetadata{
		Source:         "workspace_catalog",
		Freshness:      "live",
		SelectionScope: "planning/default",
		Eligibility:    SkillEligibility{State: "eligible"},
		Precedence:     SkillPrecedence{Tier: "workspace", Rank: 1},
	})
	if _, err := manager.Close(context.Background(), session.SessionID); err != nil {
		t.Fatalf("close after upsert failed: %v", err)
	}
	if _, err := manager.Resume(context.Background(), session.SessionID); err != nil {
		t.Fatalf("resume after upsert failed: %v", err)
	}

	updatedIndex, err := manager.Execute(context.Background(), session.SessionID, "cat /skills/_index.json", contract.ExecutionPolicy{})
	if err != nil {
		t.Fatalf("read updated skills index failed: %v", err)
	}
	updatedSkills := decodeProjectionRecords(t, []byte(updatedIndex.Result.Stdout))
	primary := requireProjectionRecord(t, updatedSkills, "/skills/planning/primary/SKILL.md")
	assertSkillProjectionMetadata(t, primary, "workspace_catalog", "live", "eligible", "", "workspace", 1, true)
	fallback = requireProjectionRecord(t, updatedSkills, "/skills/planning/fallback/SKILL.md")
	assertSkillProjectionMetadata(t, fallback, "bundled_catalog", "snapshot", "eligible", "", "bundled", 90, false)
	updatedSkillMaps := decodeProjectionRecordMaps(t, []byte(updatedIndex.Result.Stdout))
	primaryMap := requireProjectionRecordMap(t, updatedSkillMaps, "/skills/planning/primary/SKILL.md")
	fallbackMap := requireProjectionRecordMap(t, updatedSkillMaps, "/skills/planning/fallback/SKILL.md")
	primarySelection := requireSelectionMap(t, primaryMap)
	fallbackSelection := requireSelectionMap(t, fallbackMap)
	if mapStringField(t, primarySelection, "scope") != "planning/default" || mapStringField(t, primarySelection, "mode") == "" {
		t.Fatalf("primary selection after upsert = %+v, want explicit planning/default provenance", primarySelection)
	}
	if mapStringField(t, fallbackSelection, "reason") == "" || mapStringField(t, fallbackSelection, "winner_path") != "/skills/planning/primary/SKILL.md" {
		t.Fatalf("fallback selection after upsert = %+v, want loser reason and winner path", fallbackSelection)
	}

	adapter.UpdateSkill("planning/primary", "# Primary Planner\nupdated\n")
	if _, err := manager.Close(context.Background(), session.SessionID); err != nil {
		t.Fatalf("close after update failed: %v", err)
	}
	if _, err := manager.Resume(context.Background(), session.SessionID); err != nil {
		t.Fatalf("resume after update failed: %v", err)
	}

	updatedSkill, err := manager.Execute(context.Background(), session.SessionID, "cat /skills/planning/primary/SKILL.md", contract.ExecutionPolicy{})
	if err != nil {
		t.Fatalf("read updated skill failed: %v", err)
	}
	if !strings.Contains(updatedSkill.Result.Stdout, "updated") {
		t.Fatalf("updated skill content = %+v, want updated marker", updatedSkill.Result)
	}
	updatedProjection, err := manager.Execute(context.Background(), session.SessionID, "cat /memory/projections.json", contract.ExecutionPolicy{})
	if err != nil {
		t.Fatalf("read updated projection view failed: %v", err)
	}
	updatedProjectionView := decodeProjectionView(t, []byte(updatedProjection.Result.Stdout))
	primary = requireProjectionRecord(t, updatedProjectionView.Skills, "/skills/planning/primary/SKILL.md")
	if primary.Source != "control_plane" || primary.Freshness != projectionFreshnessUpdated || !primary.Selected {
		t.Fatalf("updated primary projection = %+v, want control_plane/updated selected", primary)
	}
	if primary.Selection == nil || primary.Selection.Scope != "planning/default" || strings.TrimSpace(primary.Selection.Reason) == "" {
		t.Fatalf("updated primary selection = %+v, want preserved planning/default selected state", primary.Selection)
	}

	adapter.RemoveSkill("planning/primary")
	if _, err := manager.Close(context.Background(), session.SessionID); err != nil {
		t.Fatalf("close after remove failed: %v", err)
	}
	if _, err := manager.Resume(context.Background(), session.SessionID); err != nil {
		t.Fatalf("resume after remove failed: %v", err)
	}

	removedIndex, err := manager.Execute(context.Background(), session.SessionID, "cat /skills/_index.json", contract.ExecutionPolicy{})
	if err != nil {
		t.Fatalf("read removed skills index failed: %v", err)
	}
	removedSkills := decodeProjectionRecords(t, []byte(removedIndex.Result.Stdout))
	if len(removedSkills) != 1 {
		t.Fatalf("skills after remove = %+v, want only fallback", removedSkills)
	}
	fallback = requireProjectionRecord(t, removedSkills, "/skills/planning/fallback/SKILL.md")
	assertSkillProjectionMetadata(t, fallback, "bundled_catalog", "snapshot", "eligible", "", "bundled", 90, true)
	removedSkillMaps := decodeProjectionRecordMaps(t, []byte(removedIndex.Result.Stdout))
	fallbackMap = requireProjectionRecordMap(t, removedSkillMaps, "/skills/planning/fallback/SKILL.md")
	fallbackSelection = requireSelectionMap(t, fallbackMap)
	if mapStringField(t, fallbackSelection, "scope") != "planning/default" || mapStringField(t, fallbackSelection, "winner_path") != "" {
		t.Fatalf("fallback selection after remove = %+v, want reselected singleton without winner path", fallbackSelection)
	}
}

func TestReferenceAdapterCuratedMemoryViewDistinctFromRawSignals(t *testing.T) {
	adapter := New(Options{
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
	})
	manager := runtimeengine.NewSessionManager(runtimeengine.SessionManagerOptions{NewID: func() string { return "sess_curated_distinct" }})

	session, err := manager.Create(context.Background(), runtimeengine.Options{
		HostRoot: t.TempDir(),
		Profile:  contract.ProfileBashPlus,
		Policy: contract.ExecutionPolicy{
			WriteMode:        contract.WriteModeFull,
			MaxPipelineDepth: 16,
			MaxOutputBytes:   4 << 20,
			Timeout:          contract.DefaultPolicy().Timeout,
		},
		Adapters: []contract.SessionAdapter{adapter},
	})
	if err != nil {
		t.Fatalf("create session failed: %v", err)
	}

	if _, err := manager.Execute(context.Background(), session.SessionID, "cat /knowledge_base/reference/guide.md", contract.ExecutionPolicy{}); err != nil {
		t.Fatalf("read reference projection failed: %v", err)
	}
	if _, err := manager.Execute(context.Background(), session.SessionID, "cat /resources/checklists/plan.json", contract.ExecutionPolicy{}); err != nil {
		t.Fatalf("read resource projection failed: %v", err)
	}
	if _, err := manager.Execute(context.Background(), session.SessionID, "echo plan > /task_outputs/plan.txt", contract.ExecutionPolicy{}); err != nil {
		t.Fatalf("write task output failed: %v", err)
	}
	deniedWrite, err := manager.Execute(context.Background(), session.SessionID, "echo blocked > /knowledge_base/reference/guide.md", contract.ExecutionPolicy{})
	if err != nil {
		t.Fatalf("write to read-only projection failed unexpectedly: %v", err)
	}
	if deniedWrite.Result.ExitCode == 0 {
		t.Fatalf("expected denied projection write to fail, got %+v", deniedWrite.Result)
	}
	adapter.UpsertCuratedEntry(CuratedEntry{
		ID:          "plan-context",
		Title:       "Plan Context",
		Summary:     "Curated entry for the current plan workflow.",
		SourcePaths: []string{"/knowledge_base/reference/guide.md", "/resources/checklists/plan.json", "/task_outputs/plan.txt"},
	})

	curatedPath := requireCuratedMemoryJSONPath(t, manager, session.SessionID)
	curatedView, err := manager.Execute(context.Background(), session.SessionID, "cat "+curatedPath, contract.ExecutionPolicy{})
	if err != nil {
		t.Fatalf("read curated memory view %q failed: %v", curatedPath, err)
	}
	curatedSnapshot := decodeCuratedMemorySnapshot(t, []byte(curatedView.Result.Stdout))

	observations, err := manager.Execute(context.Background(), session.SessionID, "cat /memory/observations.md", contract.ExecutionPolicy{})
	if err != nil {
		t.Fatalf("read observations failed: %v", err)
	}
	projections, err := manager.Execute(context.Background(), session.SessionID, "cat /memory/projections.json", contract.ExecutionPolicy{})
	if err != nil {
		t.Fatalf("read projections failed: %v", err)
	}

	observationPaths := extractObservationPaths(observations.Result.Stdout)
	if !hasPathIntersection(curatedSnapshot.SourcePaths, observationPaths) {
		t.Fatalf("expected curated source paths to reference observed paths, curated=%v observed=%v", curatedSnapshot.SourcePaths, observationPaths)
	}
	if strings.Contains(curatedView.Result.Stdout, "read-ref:") || strings.Contains(curatedView.Result.Stdout, "read-resource:") || strings.Contains(curatedView.Result.Stdout, "read-skill:") {
		t.Fatalf("expected curated view to stay distinct from raw observations, got %q", curatedView.Result.Stdout)
	}

	var projectionLike projectionView
	if err := json.Unmarshal([]byte(curatedView.Result.Stdout), &projectionLike); err == nil && (len(projectionLike.Documents) > 0 || len(projectionLike.Resources) > 0 || len(projectionLike.Skills) > 0) {
		t.Fatalf("expected curated view to stay distinct from projection index shape, got %+v", projectionLike)
	}
	if strings.TrimSpace(curatedView.Result.Stdout) == strings.TrimSpace(observations.Result.Stdout) {
		t.Fatalf("expected curated view content to differ from raw observations")
	}
	if strings.TrimSpace(curatedView.Result.Stdout) == strings.TrimSpace(projections.Result.Stdout) {
		t.Fatalf("expected curated view content to differ from projection index")
	}

	deniedCuratedWrite, err := manager.Execute(context.Background(), session.SessionID, "echo blocked > "+curatedPath, contract.ExecutionPolicy{})
	if err != nil {
		t.Fatalf("write to curated memory view failed unexpectedly: %v", err)
	}
	if deniedCuratedWrite.Result.ExitCode == 0 {
		t.Fatalf("expected write to curated memory view to be denied, got %+v", deniedCuratedWrite.Result)
	}
}

func TestReferenceAdapterCuratedMemoryViewSurvivesCheckpointResume(t *testing.T) {
	adapter := New(Options{
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
	})
	manager := runtimeengine.NewSessionManager(runtimeengine.SessionManagerOptions{NewID: func() string { return "sess_curated_resume" }})

	session, err := manager.Create(context.Background(), runtimeengine.Options{
		HostRoot: t.TempDir(),
		Profile:  contract.ProfileBashPlus,
		Policy: contract.ExecutionPolicy{
			WriteMode:        contract.WriteModeFull,
			MaxPipelineDepth: 16,
			MaxOutputBytes:   4 << 20,
			Timeout:          contract.DefaultPolicy().Timeout,
		},
		Adapters: []contract.SessionAdapter{adapter},
	})
	if err != nil {
		t.Fatalf("create session failed: %v", err)
	}

	if _, err := manager.Execute(context.Background(), session.SessionID, "cat /knowledge_base/reference/guide.md", contract.ExecutionPolicy{}); err != nil {
		t.Fatalf("read reference projection failed: %v", err)
	}
	if _, err := manager.Execute(context.Background(), session.SessionID, "cat /resources/checklists/plan.json", contract.ExecutionPolicy{}); err != nil {
		t.Fatalf("read resource projection failed: %v", err)
	}
	if _, err := manager.Execute(context.Background(), session.SessionID, "echo plan > /task_outputs/plan.txt", contract.ExecutionPolicy{}); err != nil {
		t.Fatalf("write task output failed: %v", err)
	}
	adapter.UpsertCuratedEntry(CuratedEntry{
		ID:          "plan-context",
		Title:       "Plan Context",
		Summary:     "Curated entry for the current plan workflow.",
		SourcePaths: []string{"/knowledge_base/reference/guide.md", "/resources/checklists/plan.json", "/task_outputs/plan.txt"},
	})

	curatedPath := requireCuratedMemoryJSONPath(t, manager, session.SessionID)
	beforeCurated, err := manager.Execute(context.Background(), session.SessionID, "cat "+curatedPath, contract.ExecutionPolicy{})
	if err != nil {
		t.Fatalf("read curated view before checkpoint failed: %v", err)
	}
	beforeSnapshot := decodeCuratedMemorySnapshot(t, []byte(beforeCurated.Result.Stdout))

	checkpoint, err := manager.Checkpoint(context.Background(), session.SessionID)
	if err != nil {
		t.Fatalf("checkpoint failed: %v", err)
	}
	if len(checkpoint.State.Opaque[adapter.AdapterID()]) == 0 {
		t.Fatalf("checkpoint missing adapter opaque state")
	}
	if _, err := manager.Close(context.Background(), session.SessionID); err != nil {
		t.Fatalf("close failed: %v", err)
	}
	if _, err := manager.Resume(context.Background(), session.SessionID); err != nil {
		t.Fatalf("resume failed: %v", err)
	}

	resumedCuratedPath := requireCuratedMemoryJSONPath(t, manager, session.SessionID)
	if resumedCuratedPath != curatedPath {
		t.Fatalf("curated view path drifted across resume: before=%q after=%q", curatedPath, resumedCuratedPath)
	}
	afterCurated, err := manager.Execute(context.Background(), session.SessionID, "cat "+resumedCuratedPath, contract.ExecutionPolicy{})
	if err != nil {
		t.Fatalf("read curated view after resume failed: %v", err)
	}
	afterSnapshot := decodeCuratedMemorySnapshot(t, []byte(afterCurated.Result.Stdout))
	if beforeSnapshot.EntryCount != afterSnapshot.EntryCount {
		t.Fatalf("curated entry count changed across resume: before=%d after=%d", beforeSnapshot.EntryCount, afterSnapshot.EntryCount)
	}
	if !reflect.DeepEqual(beforeSnapshot.SourcePaths, afterSnapshot.SourcePaths) {
		t.Fatalf("curated source paths changed across resume: before=%v after=%v", beforeSnapshot.SourcePaths, afterSnapshot.SourcePaths)
	}
}

type curatedMemorySnapshot struct {
	EntryCount  int
	SourcePaths []string
}

type projectionMaterializationFailure struct {
	Code string `json:"code,omitempty"`
}

type projectionMaterialization struct {
	State   string                            `json:"state"`
	Reason  string                            `json:"reason,omitempty"`
	Failure *projectionMaterializationFailure `json:"failure,omitempty"`
}

type projectionMaterializationRecord struct {
	Path            string                     `json:"path"`
	Materialization *projectionMaterialization `json:"materialization,omitempty"`
}

type projectionMaterializationView struct {
	Documents []projectionMaterializationRecord `json:"documents,omitempty"`
	Resources []projectionMaterializationRecord `json:"resources,omitempty"`
	Skills    []projectionMaterializationRecord `json:"skills,omitempty"`
}

func requireCuratedMemoryJSONPath(t *testing.T, manager *runtimeengine.SessionManager, sessionID string) string {
	t.Helper()
	probe, err := manager.Execute(context.Background(), sessionID, "cat /memory/curated.json", contract.ExecutionPolicy{})
	if err != nil {
		t.Fatalf("read /memory/curated.json failed: %v", err)
	}
	if probe.Result.ExitCode != 0 {
		t.Fatalf("expected /memory/curated.json to exist, got %+v", probe.Result)
	}
	return "/memory/curated.json"
}

func decodeCuratedMemorySnapshot(t *testing.T, raw []byte) curatedMemorySnapshot {
	t.Helper()
	var payload any
	if err := json.Unmarshal(raw, &payload); err != nil {
		t.Fatalf("decode curated memory view failed: %v", err)
	}
	entryCount, sourcePaths := collectCuratedEntrySources(payload)
	if entryCount == 0 {
		t.Fatalf("expected at least one curated entry, got payload=%s", strings.TrimSpace(string(raw)))
	}
	if len(sourcePaths) == 0 {
		t.Fatalf("expected curated entries to contain source-path references, got payload=%s", strings.TrimSpace(string(raw)))
	}
	return curatedMemorySnapshot{
		EntryCount:  entryCount,
		SourcePaths: sourcePaths,
	}
}

func collectCuratedEntrySources(payload any) (int, []string) {
	unique := map[string]struct{}{}
	entryCount := 0
	var walk func(any)
	walk = func(node any) {
		switch typed := node.(type) {
		case map[string]any:
			paths := sourcePathsFromCuratedRecord(typed)
			if len(paths) > 0 {
				entryCount++
				for _, pathValue := range paths {
					unique[pathValue] = struct{}{}
				}
			}
			for _, child := range typed {
				walk(child)
			}
		case []any:
			for _, child := range typed {
				walk(child)
			}
		}
	}
	walk(payload)
	paths := make([]string, 0, len(unique))
	for pathValue := range unique {
		paths = append(paths, pathValue)
	}
	sort.Strings(paths)
	return entryCount, paths
}

func sourcePathsFromCuratedRecord(record map[string]any) []string {
	keys := []string{"source_path", "source_paths", "sourcePath", "sourcePaths"}
	paths := make([]string, 0)
	for _, key := range keys {
		value, ok := record[key]
		if !ok {
			continue
		}
		paths = append(paths, extractCuratedPaths(value)...)
	}
	return dedupeLines(paths)
}

func extractCuratedPaths(value any) []string {
	switch typed := value.(type) {
	case string:
		pathValue := strings.TrimSpace(typed)
		if strings.HasPrefix(pathValue, "/") {
			return []string{pathValue}
		}
	case []any:
		paths := make([]string, 0, len(typed))
		for _, item := range typed {
			if text, ok := item.(string); ok {
				pathValue := strings.TrimSpace(text)
				if strings.HasPrefix(pathValue, "/") {
					paths = append(paths, pathValue)
				}
			}
		}
		return dedupeLines(paths)
	}
	return nil
}

func extractObservationPaths(raw string) []string {
	paths := make([]string, 0)
	for _, line := range strings.Split(raw, "\n") {
		value := strings.TrimSpace(line)
		if value == "" {
			continue
		}
		sep := strings.Index(value, ":")
		if sep <= 0 || sep >= len(value)-1 {
			continue
		}
		pathValue := strings.TrimSpace(value[sep+1:])
		if strings.HasPrefix(pathValue, "/") {
			paths = append(paths, pathValue)
		}
	}
	return dedupeLines(paths)
}

func hasPathIntersection(left []string, right []string) bool {
	for _, value := range left {
		if containsLine(right, value) {
			return true
		}
	}
	return false
}

func decodeReferenceState(t *testing.T, raw []byte) sessionState {
	t.Helper()
	var state sessionState
	if err := json.Unmarshal(raw, &state); err != nil {
		t.Fatalf("decode session state failed: %v", err)
	}
	return state
}

func containsObservation(observations []string, target string) bool {
	for _, observation := range observations {
		if observation == target {
			return true
		}
	}
	return false
}

func decodeWorkflowViews(t *testing.T, raw []byte) []workflowView {
	t.Helper()
	var workflows []workflowView
	if err := json.Unmarshal(raw, &workflows); err != nil {
		t.Fatalf("decode workflow views failed: %v", err)
	}
	return workflows
}

func workflowStatusByID(workflows []workflowView, id string) string {
	for _, workflow := range workflows {
		if workflow.ID == id {
			return workflow.Status
		}
	}
	return ""
}

func requireWorkflowView(t *testing.T, workflows []workflowView, id string) workflowView {
	t.Helper()
	for _, workflow := range workflows {
		if workflow.ID == id {
			return workflow
		}
	}
	t.Fatalf("workflow %q not found in %+v", id, workflows)
	return workflowView{}
}

func decodeProjectionView(t *testing.T, raw []byte) projectionView {
	t.Helper()
	var view projectionView
	if err := json.Unmarshal(raw, &view); err != nil {
		t.Fatalf("decode projection view failed: %v", err)
	}
	return view
}

func decodeProjectionMetricsView(t *testing.T, raw []byte) projectionMetricsView {
	t.Helper()
	var view projectionMetricsView
	if err := json.Unmarshal(raw, &view); err != nil {
		t.Fatalf("decode projection metrics view failed: %v", err)
	}
	return view
}

func decodeDenialView(t *testing.T, raw []byte) denialView {
	t.Helper()
	var view denialView
	if err := json.Unmarshal(raw, &view); err != nil {
		t.Fatalf("decode denial view failed: %v", err)
	}
	return view
}

func decodeProjectionMaterializationView(t *testing.T, raw []byte) projectionMaterializationView {
	t.Helper()
	var view projectionMaterializationView
	if err := json.Unmarshal(raw, &view); err != nil {
		t.Fatalf("decode projection materialization view failed: %v", err)
	}
	return view
}

func decodeProjectionRecords(t *testing.T, raw []byte) []projectionRecord {
	t.Helper()
	var records []projectionRecord
	if err := json.Unmarshal(raw, &records); err != nil {
		t.Fatalf("decode projection records failed: %v", err)
	}
	return records
}

func requireProjectionRecord(t *testing.T, records []projectionRecord, target string) projectionRecord {
	t.Helper()
	for _, record := range records {
		if record.Path == target {
			return record
		}
	}
	t.Fatalf("projection record %q not found in %+v", target, records)
	return projectionRecord{}
}

func requireProjectionMaterializationRecord(t *testing.T, records []projectionMaterializationRecord, target string) projectionMaterializationRecord {
	t.Helper()
	for _, record := range records {
		if record.Path == target {
			return record
		}
	}
	t.Fatalf("projection materialization record %q not found in %+v", target, records)
	return projectionMaterializationRecord{}
}

func assertProjectionMaterializationStatePresent(t *testing.T, record projectionMaterializationRecord, label string) {
	t.Helper()
	if record.Materialization == nil {
		t.Fatalf("%s materialization is missing for %+v", label, record)
	}
	if strings.TrimSpace(record.Materialization.State) == "" {
		t.Fatalf("%s materialization state is empty for %+v", label, record.Materialization)
	}
}

func assertProjectionMaterializationPartialOrError(t *testing.T, record projectionMaterializationRecord, label string) {
	t.Helper()
	assertProjectionMaterializationStatePresent(t, record, label)
	state := strings.TrimSpace(record.Materialization.State)
	if state != "partial" && state != "error" {
		t.Fatalf("%s materialization state = %q, want partial or error", label, state)
	}
	if strings.TrimSpace(record.Materialization.Reason) == "" &&
		(record.Materialization.Failure == nil || strings.TrimSpace(record.Materialization.Failure.Code) == "") {
		t.Fatalf("%s partial/error materialization lacks machine-readable detail: %+v", label, record.Materialization)
	}
}

func assertSkillProjectionMetadata(t *testing.T, record projectionRecord, source string, freshness string, eligibilityState string, eligibilityReason string, precedenceTier string, precedenceRank int, selected bool) {
	t.Helper()
	if record.Source != source || record.Freshness != freshness {
		t.Fatalf("projection record metadata = %+v, want source=%q freshness=%q", record, source, freshness)
	}
	if record.Eligibility == nil {
		t.Fatalf("projection record eligibility missing: %+v", record)
	}
	if record.Eligibility.State != eligibilityState || record.Eligibility.Reason != eligibilityReason {
		t.Fatalf("projection record eligibility = %+v, want state=%q reason=%q", record.Eligibility, eligibilityState, eligibilityReason)
	}
	if record.Precedence == nil {
		t.Fatalf("projection record precedence missing: %+v", record)
	}
	if record.Precedence.Tier != precedenceTier || record.Precedence.Rank != precedenceRank {
		t.Fatalf("projection record precedence = %+v, want tier=%q rank=%d", record.Precedence, precedenceTier, precedenceRank)
	}
	if record.Selected != selected {
		t.Fatalf("projection record selected = %v, want %v", record.Selected, selected)
	}
}

func decodeProjectionRecordMaps(t *testing.T, raw []byte) []map[string]any {
	t.Helper()
	var records []map[string]any
	if err := json.Unmarshal(raw, &records); err != nil {
		t.Fatalf("decode projection record maps failed: %v", err)
	}
	return records
}

func requireProjectionRecordMap(t *testing.T, records []map[string]any, target string) map[string]any {
	t.Helper()
	for _, record := range records {
		if strings.TrimSpace(mapStringField(t, record, "path")) == target {
			return record
		}
	}
	t.Fatalf("projection record map %q not found in %+v", target, records)
	return nil
}

func requireSelectionMap(t *testing.T, record map[string]any) map[string]any {
	t.Helper()
	selectionValue, ok := record["selection"]
	if !ok {
		t.Fatalf("selection field missing in projection record: %+v", record)
	}
	selection, ok := selectionValue.(map[string]any)
	if !ok {
		t.Fatalf("selection field has unexpected type %T in projection record: %+v", selectionValue, record)
	}
	return selection
}

func mapStringField(t *testing.T, record map[string]any, key string) string {
	t.Helper()
	value, ok := record[key]
	if !ok {
		return ""
	}
	text, ok := value.(string)
	if !ok {
		t.Fatalf("field %q has unexpected type %T in record %+v", key, value, record)
	}
	return strings.TrimSpace(text)
}

func mapOptionalStringField(t *testing.T, record map[string]any, key string) string {
	t.Helper()
	if value, ok := record[key]; ok {
		text, ok := value.(string)
		if !ok {
			t.Fatalf("field %q has unexpected type %T in record %+v", key, value, record)
		}
		return strings.TrimSpace(text)
	}
	return ""
}

func mapBoolField(record map[string]any, key string) bool {
	value, ok := record[key]
	if !ok {
		return false
	}
	flag, ok := value.(bool)
	if !ok {
		return false
	}
	return flag
}

func readTieBreakSnapshot(t *testing.T, manager *runtimeengine.SessionManager, sessionID string) (winner string, loser string, scope string) {
	t.Helper()
	skillIndex, err := manager.Execute(context.Background(), sessionID, "cat /skills/_index.json", contract.ExecutionPolicy{})
	if err != nil {
		t.Fatalf("read skills index failed: %v", err)
	}
	records := decodeProjectionRecordMaps(t, []byte(skillIndex.Result.Stdout))
	alpha := requireProjectionRecordMap(t, records, "/skills/triage/alpha/SKILL.md")
	beta := requireProjectionRecordMap(t, records, "/skills/triage/beta/SKILL.md")

	alphaSelected := mapBoolField(alpha, "selected")
	betaSelected := mapBoolField(beta, "selected")
	if alphaSelected == betaSelected {
		t.Fatalf("tie-break must choose exactly one winner: alpha_selected=%v beta_selected=%v", alphaSelected, betaSelected)
	}

	alphaSelection := requireSelectionMap(t, alpha)
	betaSelection := requireSelectionMap(t, beta)
	alphaScope := mapStringField(t, alphaSelection, "scope")
	betaScope := mapStringField(t, betaSelection, "scope")
	if alphaScope == "" || betaScope == "" {
		t.Fatalf("tie-break selection scope must be explicit: alpha=%q beta=%q", alphaScope, betaScope)
	}
	if alphaScope != betaScope {
		t.Fatalf("tie-break skills must share explicit scope: alpha=%q beta=%q", alphaScope, betaScope)
	}
	if mapStringField(t, alphaSelection, "mode") == "" || mapStringField(t, betaSelection, "mode") == "" {
		t.Fatalf("tie-break selection mode must be explicit: alpha=%+v beta=%+v", alphaSelection, betaSelection)
	}

	if alphaSelected {
		winner = "/skills/triage/alpha/SKILL.md"
		loser = "/skills/triage/beta/SKILL.md"
		if mapStringField(t, betaSelection, "reason") == "" || mapOptionalStringField(t, betaSelection, "winner_path") != winner {
			t.Fatalf("tie-break loser must carry machine-readable reason: %+v", betaSelection)
		}
	} else {
		winner = "/skills/triage/beta/SKILL.md"
		loser = "/skills/triage/alpha/SKILL.md"
		if mapStringField(t, alphaSelection, "reason") == "" || mapOptionalStringField(t, alphaSelection, "winner_path") != winner {
			t.Fatalf("tie-break loser must carry machine-readable reason: %+v", alphaSelection)
		}
	}
	return winner, loser, alphaScope
}

type controlPlaneEventView struct {
	Seq                   int    `json:"seq"`
	Op                    string `json:"op"`
	Path                  string `json:"path"`
	SelectionScope        string `json:"selection_scope,omitempty"`
	Result                string `json:"result"`
	Visibility            string `json:"visibility"`
	VisibleAfter          string `json:"visible_after"`
	VisibleFromGeneration int    `json:"visible_from_generation"`
	SelectedBefore        bool   `json:"selected_before"`
	SelectedAfter         bool   `json:"selected_after"`
	WinnerBefore          string `json:"winner_before,omitempty"`
	WinnerAfter           string `json:"winner_after,omitempty"`
	ReasonAfter           string `json:"reason_after,omitempty"`
}

func decodeControlPlaneEventViews(t *testing.T, raw []byte) []controlPlaneEventView {
	t.Helper()
	var events []controlPlaneEventView
	if err := json.Unmarshal(raw, &events); err != nil {
		t.Fatalf("decode control-plane events failed: %v", err)
	}
	return events
}

func summaryLineValue(raw string, key string) (string, bool) {
	prefix := "- " + key + ": "
	for _, line := range strings.Split(raw, "\n") {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, prefix) {
			continue
		}
		return strings.TrimSpace(strings.TrimPrefix(line, prefix)), true
	}
	return "", false
}

func summaryLineInt(raw string, key string) (int, bool) {
	value, ok := summaryLineValue(raw, key)
	if !ok {
		return 0, false
	}
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return 0, false
	}
	return parsed, true
}
