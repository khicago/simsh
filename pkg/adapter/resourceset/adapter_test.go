package resourceset

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/khicago/simsh/pkg/contract"
)

func TestResourceSetAdapterLifecycle(t *testing.T) {
	ctx := context.Background()
	adapter := New(Options{
		Resources: map[string]string{
			"manuals/guide.md":   "# Guide\nstart here\n",
			"templates/plan.txt": "plan template\n",
		},
		ResourceMetadata: map[string]ResourceMetadata{
			"manuals/guide.md": {Source: "sync", Freshness: "live"},
		},
	})

	created, err := adapter.CreateSession(ctx, contract.Session{})
	if err != nil {
		t.Fatalf("create session: %v", err)
	}
	resourceMount := requireResourceMount(t, created.VirtualMounts)
	if content, err := resourceMount.ReadRawContent(ctx, "/resources/manuals/guide.md"); err != nil {
		t.Fatalf("read guide: %v", err)
	} else if !strings.Contains(content, "# Guide") {
		t.Fatalf("unexpected guide content: %q", content)
	}
	indexRecords := requireResourceIndex(t, resourceMount)
	if len(indexRecords) != 2 {
		t.Fatalf("expected 2 index records, got %d", len(indexRecords))
	}
	if indexRecords[0].Source != "sync" {
		t.Fatalf("manual source mismatch: %s", indexRecords[0].Source)
	}
	if indexRecords[1].Source != defaultResourceSource {
		t.Fatalf("template source mismatch: %s", indexRecords[1].Source)
	}

	summary := readMemoryFile(t, created.Memory.Mount, "/memory/summary.md")
	requireContains(t, summary, "- phase: created")
	requireContains(t, summary, "- resource_count: 2")
	requireContains(t, summary, "- observations: 0")
	observations := readMemoryFile(t, created.Memory.Mount, "/memory/observations.md")
	requireContains(t, observations, "- none recorded yet")

	records := readMemoryRecords(t, created.Memory.Mount, "/memory/resources.json")
	if len(records) != 2 {
		t.Fatalf("expected 2 memory records, got %d", len(records))
	}

	session := sessionWithState(adapter, created.OpaqueState)
	result := contract.ExecutionResult{
		ExecutionID: "exec-1",
		Trace: contract.ExecutionTrace{
			ReadPaths:    []string{"/resources/manuals/guide.md"},
			WrittenPaths: []string{"/resources/templates/plan.txt"},
			DeniedPaths:  []string{"/resources/protected.txt"},
		},
	}
	observed, err := adapter.ObserveExecution(ctx, session, result)
	if err != nil {
		t.Fatalf("observe execution: %v", err)
	}

	obSummary := readMemoryFile(t, observed.Memory.Mount, "/memory/summary.md")
	requireContains(t, obSummary, "- phase: observed")
	requireContains(t, obSummary, "- observations: 3")
	obsContent := readMemoryFile(t, observed.Memory.Mount, "/memory/observations.md")
	requireContains(t, obsContent, "- denied:/resources/protected.txt")
	requireContains(t, obsContent, "- read:/resources/manuals/guide.md")
	requireContains(t, obsContent, "- write:/resources/templates/plan.txt")

	state := decodeState(t, observed.OpaqueState)
	if state.Phase != "observed" {
		t.Fatalf("state phase: %s", state.Phase)
	}
	if len(state.Observations) != 3 {
		t.Fatalf("expected 3 observations, got %d", len(state.Observations))
	}

	checkpointSession := sessionWithState(adapter, observed.OpaqueState)
	checkpointed, err := adapter.CheckpointSession(ctx, checkpointSession)
	if err != nil {
		t.Fatalf("checkpoint: %v", err)
	}
	requireContains(t, readMemoryFile(t, checkpointed.Memory.Mount, "/memory/summary.md"), "- phase: checkpointed")

	resumeSession := sessionWithState(adapter, checkpointed.OpaqueState)
	resumed, err := adapter.ResumeSession(ctx, resumeSession)
	if err != nil {
		t.Fatalf("resume: %v", err)
	}
	requireContains(t, readMemoryFile(t, resumed.Memory.Mount, "/memory/summary.md"), "- phase: resumed")

	closeSession := sessionWithState(adapter, resumed.OpaqueState)
	closed, err := adapter.CloseSession(ctx, closeSession)
	if err != nil {
		t.Fatalf("close: %v", err)
	}
	requireContains(t, readMemoryFile(t, closed.Memory.Mount, "/memory/summary.md"), "- phase: closed")
}

func TestResourceSetAdapterObservationDedupe(t *testing.T) {
	ctx := context.Background()
	adapter := New(Options{
		Resources: map[string]string{
			"manuals/guide.md": "# Guide\n",
		},
	})

	created, err := adapter.CreateSession(ctx, contract.Session{})
	if err != nil {
		t.Fatalf("create session: %v", err)
	}

	session := sessionWithState(adapter, created.OpaqueState)
	trace := contract.ExecutionTrace{
		ReadPaths: []string{"/resources/manuals/guide.md"},
	}
	first, err := adapter.ObserveExecution(ctx, session, contract.ExecutionResult{ExecutionID: "exec-1", Trace: trace})
	if err != nil {
		t.Fatalf("first observe: %v", err)
	}
	second, err := adapter.ObserveExecution(ctx, sessionWithState(adapter, first.OpaqueState), contract.ExecutionResult{ExecutionID: "exec-2", Trace: trace})
	if err != nil {
		t.Fatalf("second observe: %v", err)
	}

	summary := readMemoryFile(t, second.Memory.Mount, "/memory/summary.md")
	requireContains(t, summary, "- observations: 1")
	observations := readMemoryFile(t, second.Memory.Mount, "/memory/observations.md")
	if strings.Count(observations, "- read:/resources/manuals/guide.md") != 1 {
		t.Fatalf("unexpected observation entries:\n%s", observations)
	}
	state := decodeState(t, second.OpaqueState)
	if len(state.Observations) != 1 {
		t.Fatalf("state observations: %d", len(state.Observations))
	}
}

func requireResourceMount(t *testing.T, mounts []contract.VirtualMount) contract.VirtualMount {
	t.Helper()
	for _, mount := range mounts {
		if mount.MountPoint() == "/resources" {
			return mount
		}
	}
	t.Fatalf("resources mount missing")
	return nil
}

func requireResourceIndex(t *testing.T, mount contract.VirtualMount) []resourceRecord {
	t.Helper()
	raw, err := mount.ReadRawContent(context.Background(), "/resources/_index.json")
	if err != nil {
		t.Fatalf("read index: %v", err)
	}
	var records []resourceRecord
	if err := json.Unmarshal([]byte(raw), &records); err != nil {
		t.Fatalf("decode index: %v", err)
	}
	return records
}

func readMemoryFile(t *testing.T, mount contract.VirtualMount, path string) string {
	t.Helper()
	raw, err := mount.ReadRawContent(context.Background(), path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return raw
}

func readMemoryRecords(t *testing.T, mount contract.VirtualMount, path string) []resourceRecord {
	t.Helper()
	raw := readMemoryFile(t, mount, path)
	var records []resourceRecord
	if err := json.Unmarshal([]byte(raw), &records); err != nil {
		t.Fatalf("decode memory records: %v", err)
	}
	return records
}

func requireContains(t *testing.T, value, substr string) {
	t.Helper()
	if !strings.Contains(value, substr) {
		t.Fatalf("expected %q to contain %q", value, substr)
	}
}

func sessionWithState(adapter *Adapter, raw []byte) contract.Session {
	session := contract.Session{}
	if len(raw) == 0 {
		return session
	}
	session.State.Opaque = map[string]json.RawMessage{
		adapter.AdapterID(): append(json.RawMessage(nil), raw...),
	}
	return session
}

func decodeState(t *testing.T, raw []byte) sessionState {
	t.Helper()
	var state sessionState
	if err := json.Unmarshal(raw, &state); err != nil {
		t.Fatalf("decode state: %v", err)
	}
	return state
}
