package resourceset

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/khicago/simsh/pkg/adapter/internal/contracttest"
	"github.com/khicago/simsh/pkg/contract"
)

func TestResourceSetAdapterConformance(t *testing.T) {
	adapter := New(Options{
		Resources: map[string]string{
			"manuals/guide.md":   "# Guide\nstart here\n",
			"templates/plan.txt": "plan template\n",
		},
		ResourceMetadata: map[string]ResourceMetadata{
			"manuals/guide.md": {Source: "sync", Freshness: "live"},
		},
	})
	contracttest.RunLifecycle(t, contracttest.LifecycleSpec{
		Adapter:            adapter,
		ObserveResult:      resourceSetObserveResult(),
		RequireMemoryMount: true,
		RequireOpaqueState: true,
		WantMountPoints:    []string{"/resources", "/memory"},
		CheckCreated: func(t *testing.T, snapshot contracttest.Snapshot) {
			if content := snapshot.ReadFile(t, "/resources/manuals/guide.md"); !strings.Contains(content, "# Guide") {
				t.Fatalf("created guide content = %q, want heading", content)
			}
			indexRecords := decodeResourceRecords(t, snapshot.ReadFile(t, "/resources/_index.json"))
			if len(indexRecords) != 2 {
				t.Fatalf("created resource index count = %d, want 2", len(indexRecords))
			}
			if indexRecords[0].Source != "sync" {
				t.Fatalf("created manual source = %q, want sync", indexRecords[0].Source)
			}
			if indexRecords[1].Source != defaultResourceSource {
				t.Fatalf("created template source = %q, want %q", indexRecords[1].Source, defaultResourceSource)
			}
			summary := snapshot.ReadFile(t, "/memory/summary.md")
			requireContains(t, summary, "- phase: created")
			requireContains(t, summary, "- resource_count: 2")
			requireContains(t, summary, "- observations: 0")
			requireContains(t, snapshot.ReadFile(t, "/memory/observations.md"), "- none recorded yet")
			records := decodeResourceRecords(t, snapshot.ReadFile(t, "/memory/resources.json"))
			if len(records) != 2 {
				t.Fatalf("created memory records count = %d, want 2", len(records))
			}
		},
		CheckObserved: func(t *testing.T, snapshot contracttest.Snapshot) {
			summary := snapshot.ReadFile(t, "/memory/summary.md")
			requireContains(t, summary, "- phase: observed")
			requireContains(t, summary, "- observations: 3")
			observations := snapshot.ReadFile(t, "/memory/observations.md")
			requireContains(t, observations, "- denied:/resources/protected.txt")
			requireContains(t, observations, "- read:/resources/manuals/guide.md")
			requireContains(t, observations, "- write:/resources/templates/plan.txt")

			state := decodeState(t, snapshot.Projection.OpaqueState)
			if state.Phase != "observed" {
				t.Fatalf("observed state phase = %q, want observed", state.Phase)
			}
			if len(state.Observations) != 3 {
				t.Fatalf("observed state observations = %d, want 3", len(state.Observations))
			}
		},
		CheckCheckpointed: func(t *testing.T, snapshot contracttest.Snapshot) {
			requireContains(t, snapshot.ReadFile(t, "/memory/summary.md"), "- phase: checkpointed")
		},
		CheckResumed: func(t *testing.T, snapshot contracttest.Snapshot) {
			requireContains(t, snapshot.ReadFile(t, "/memory/summary.md"), "- phase: resumed")
			if content := snapshot.ReadFile(t, "/resources/manuals/guide.md"); !strings.Contains(content, "# Guide") {
				t.Fatalf("resumed guide content = %q, want heading", content)
			}
			state := decodeState(t, snapshot.Projection.OpaqueState)
			if len(state.Observations) != 3 {
				t.Fatalf("resumed state observations = %d, want 3", len(state.Observations))
			}
		},
		CheckClosed: func(t *testing.T, snapshot contracttest.Snapshot) {
			requireContains(t, snapshot.ReadFile(t, "/memory/summary.md"), "- phase: closed")
		},
	})
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

func resourceSetObserveResult() contract.ExecutionResult {
	return contract.ExecutionResult{
		ExecutionID: "exec-1",
		Trace: contract.ExecutionTrace{
			ReadPaths:    []string{"/resources/manuals/guide.md"},
			WrittenPaths: []string{"/resources/templates/plan.txt"},
			DeniedPaths:  []string{"/resources/protected.txt"},
		},
	}
}

func decodeResourceRecords(t *testing.T, raw string) []resourceRecord {
	t.Helper()
	var records []resourceRecord
	if err := json.Unmarshal([]byte(raw), &records); err != nil {
		t.Fatalf("decodeResourceRecords(...) error = %v, want nil", err)
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
