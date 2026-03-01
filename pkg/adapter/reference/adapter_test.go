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
	})
	manager := runtimeengine.NewSessionManager(runtimeengine.SessionManagerOptions{NewID: func() string { return "sess_reference" }})
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

	statusBeforeObserve, err := manager.Execute(context.Background(), session.SessionID, "cat /memory/status.json", contract.ExecutionPolicy{})
	if err != nil {
		t.Fatalf("read initial memory status failed: %v", err)
	}
	if state := decodeReferenceState(t, []byte(statusBeforeObserve.Result.Stdout)); state.Freshness != "created" {
		t.Fatalf("expected created freshness, got %+v", state)
	}

	guide, err := manager.Execute(context.Background(), session.SessionID, "cat /knowledge_base/reference/guide.md", contract.ExecutionPolicy{})
	if err != nil {
		t.Fatalf("read reference doc failed: %v", err)
	}
	if !strings.Contains(guide.Result.Stdout, "# Guide") {
		t.Fatalf("unexpected guide output: %+v", guide.Result)
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

	updatedGuide, err := manager.Execute(context.Background(), session.SessionID, "cat /knowledge_base/reference/guide.md", contract.ExecutionPolicy{})
	if err != nil {
		t.Fatalf("read updated guide failed: %v", err)
	}
	if !strings.Contains(updatedGuide.Result.Stdout, "updated") {
		t.Fatalf("expected updated reference projection, got %+v", updatedGuide.Result)
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
