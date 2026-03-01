package runtime

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/khicago/simsh/pkg/contract"
)

func TestSessionManagerLifecycleAndCheckpointResume(t *testing.T) {
	tmp := t.TempDir()
	rcHostPath := filepath.Join(tmp, "task_outputs", "simshrc")
	if err := os.MkdirAll(filepath.Dir(rcHostPath), 0o755); err != nil {
		t.Fatalf("mkdir rc dir failed: %v", err)
	}
	if err := os.WriteFile(rcHostPath, []byte("export SESSION_FLAG=enabled\n"), 0o644); err != nil {
		t.Fatalf("write rc failed: %v", err)
	}

	nowValues := []time.Time{
		time.Date(2026, 3, 1, 21, 2, 34, 0, time.UTC),
		time.Date(2026, 3, 1, 21, 2, 35, 0, time.UTC),
		time.Date(2026, 3, 1, 21, 2, 36, 0, time.UTC),
		time.Date(2026, 3, 1, 21, 2, 37, 0, time.UTC),
	}
	idx := 0
	manager := NewSessionManager(SessionManagerOptions{
		Now: func() time.Time {
			value := nowValues[idx]
			if idx < len(nowValues)-1 {
				idx++
			}
			return value
		},
		NewID: func() string { return "sess_test" },
	})

	session, err := manager.Create(context.Background(), Options{
		HostRoot: tmp,
		Profile:  contract.ProfileCoreStrict,
		Policy:   contract.DefaultPolicy(),
		RCFiles:  []string{"/task_outputs/simshrc"},
	})
	if err != nil {
		t.Fatalf("create session failed: %v", err)
	}
	if session.SessionID != "sess_test" {
		t.Fatalf("session_id = %q, want sess_test", session.SessionID)
	}
	if session.CreatedAt != nowValues[0] || session.UpdatedAt != nowValues[0] {
		t.Fatalf("unexpected timestamps: %+v", session)
	}

	executed, err := manager.Execute(context.Background(), session.SessionID, "env SESSION_FLAG", contract.ExecutionPolicy{})
	if err != nil {
		t.Fatalf("execute failed: %v", err)
	}
	if executed.Result.ExecutionID == "" {
		t.Fatalf("expected execution_id, got %+v", executed.Result)
	}
	if executed.Result.SessionID != session.SessionID {
		t.Fatalf("unexpected session_id in result: %+v", executed.Result)
	}
	if executed.Result.ExitCode != 0 || strings.TrimSpace(executed.Result.Stdout) != "SESSION_FLAG=enabled" {
		t.Fatalf("unexpected execute result: %+v", executed.Result)
	}
	if executed.Session.UpdatedAt != nowValues[1] {
		t.Fatalf("updated_at = %s, want %s", executed.Session.UpdatedAt, nowValues[1])
	}

	checkpoint, err := manager.Checkpoint(session.SessionID)
	if err != nil {
		t.Fatalf("checkpoint failed: %v", err)
	}
	if checkpoint.UpdatedAt != nowValues[2] {
		t.Fatalf("checkpoint updated_at = %s, want %s", checkpoint.UpdatedAt, nowValues[2])
	}
	if len(checkpoint.State.RCFiles) != 1 || checkpoint.State.RCFiles[0] != "/task_outputs/simshrc" {
		t.Fatalf("unexpected checkpoint rc files: %v", checkpoint.State.RCFiles)
	}

	closed, err := manager.Close(session.SessionID)
	if err != nil {
		t.Fatalf("close failed: %v", err)
	}
	if closed.UpdatedAt != nowValues[3] {
		t.Fatalf("close updated_at = %s, want %s", closed.UpdatedAt, nowValues[3])
	}
	if _, err := manager.Execute(context.Background(), session.SessionID, "env SESSION_FLAG", contract.ExecutionPolicy{}); !errors.Is(err, ErrSessionClosed) {
		t.Fatalf("execute after close err = %v, want ErrSessionClosed", err)
	}

	if err := os.Remove(rcHostPath); err != nil {
		t.Fatalf("remove rc file failed: %v", err)
	}
	if _, err := manager.Resume(context.Background(), session.SessionID); err != nil {
		t.Fatalf("resume failed: %v", err)
	}
	resumed, err := manager.Execute(context.Background(), session.SessionID, "env SESSION_FLAG", contract.ExecutionPolicy{})
	if err != nil {
		t.Fatalf("resume execute failed: %v", err)
	}
	if resumed.Result.ExitCode != 0 || strings.TrimSpace(resumed.Result.Stdout) != "SESSION_FLAG=enabled" {
		t.Fatalf("unexpected resumed output: %+v", resumed.Result)
	}
}

func TestSessionManagerRejectsPolicyEscalation(t *testing.T) {
	manager := NewSessionManager(SessionManagerOptions{NewID: func() string { return "sess_policy" }})
	session, err := manager.Create(context.Background(), Options{
		HostRoot: t.TempDir(),
		Profile:  contract.ProfileCoreStrict,
		Policy:   contract.ExecutionPolicy{WriteMode: contract.WriteModeReadOnly},
	})
	if err != nil {
		t.Fatalf("create session failed: %v", err)
	}

	_, err = manager.Execute(context.Background(), session.SessionID, "echo hi", contract.ExecutionPolicy{WriteMode: contract.WriteModeFull})
	if err == nil || !strings.Contains(err.Error(), "exceeds session ceiling") {
		t.Fatalf("expected policy ceiling error, got %v", err)
	}
}
