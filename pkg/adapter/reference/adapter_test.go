package reference

import (
	"context"
	"encoding/json"
	"errors"
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
