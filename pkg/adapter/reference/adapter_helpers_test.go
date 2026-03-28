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

func TestDedupeLinesSortsAndTrims(t *testing.T) {
	got := dedupeLines([]string{" wrote:/task_outputs/a ", "", "denied:/kb/x", "wrote:/task_outputs/a", "read-ref:/kb/y"})
	want := []string{"denied:/kb/x", "read-ref:/kb/y", "wrote:/task_outputs/a"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("dedupeLines(...) = %#v, want %#v", got, want)
	}
}

func TestSummarizeTraceFiltersReferenceReadsAndMutations(t *testing.T) {
	got := summarizeTrace(contract.ExecutionTrace{
		ReadPaths:    []string{"/knowledge_base/reference/guide.md", "/resources/checklists/plan.json", "/knowledge_base/other.md"},
		WrittenPaths: []string{"/task_outputs/report.md"},
		DeniedPaths:  []string{"/knowledge_base/reference/guide.md"},
	})
	want := []string{
		"read-ref:/knowledge_base/reference/guide.md",
		"read-resource:/resources/checklists/plan.json",
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
		Observations:   []string{"read-ref:/knowledge_base/reference/guide.md", "read-resource:/resources/checklists/plan.json"},
		Freshness:      "observed",
		ReadRefs:       []string{"/knowledge_base/reference/guide.md"},
		ReadResources:  []string{"/resources/checklists/plan.json"},
		WrittenOutputs: []string{"/task_outputs/plan.txt"},
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

func decodeWorkflowViewsHelper(t *testing.T, raw []byte) []workflowView {
	t.Helper()
	var workflows []workflowView
	if err := json.Unmarshal(raw, &workflows); err != nil {
		t.Fatalf("decode workflow views failed: %v", err)
	}
	return workflows
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
