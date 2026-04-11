package runtime

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/khicago/simsh/pkg/contract"
	"github.com/khicago/simsh/pkg/fs"
	"github.com/khicago/simsh/pkg/mount"
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

	clock := newSteppingClock(
		time.Date(2026, 3, 1, 21, 2, 34, 0, time.UTC),
		time.Date(2026, 3, 1, 21, 2, 35, 0, time.UTC),
		time.Date(2026, 3, 1, 21, 2, 36, 0, time.UTC),
		time.Date(2026, 3, 1, 21, 2, 37, 0, time.UTC),
		time.Date(2026, 3, 1, 21, 2, 38, 0, time.UTC),
		time.Date(2026, 3, 1, 21, 2, 39, 0, time.UTC),
	)
	manager := NewSessionManager(SessionManagerOptions{
		Now:   clock.Now,
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
	if session.CreatedAt != clock.Values[0] || session.UpdatedAt != clock.Values[0] {
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
	if executed.Session.UpdatedAt != clock.Values[2] {
		t.Fatalf("updated_at = %s, want %s", executed.Session.UpdatedAt, clock.Values[2])
	}

	checkpoint, err := manager.Checkpoint(context.Background(), session.SessionID)
	if err != nil {
		t.Fatalf("checkpoint failed: %v", err)
	}
	if checkpoint.UpdatedAt != clock.Values[3] {
		t.Fatalf("checkpoint updated_at = %s, want %s", checkpoint.UpdatedAt, clock.Values[3])
	}
	if len(checkpoint.State.RCFiles) != 1 || checkpoint.State.RCFiles[0] != "/task_outputs/simshrc" {
		t.Fatalf("unexpected checkpoint rc files: %v", checkpoint.State.RCFiles)
	}

	closed, err := manager.Close(context.Background(), session.SessionID)
	if err != nil {
		t.Fatalf("close failed: %v", err)
	}
	if closed.UpdatedAt != clock.Values[4] {
		t.Fatalf("close updated_at = %s, want %s", closed.UpdatedAt, clock.Values[4])
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

func TestSessionManagerRejectsBlankIDsAndBlankCommand(t *testing.T) {
	manager := NewSessionManager(SessionManagerOptions{})

	executed, err := manager.Execute(context.Background(), "   ", "   ", contract.ExecutionPolicy{})
	if err != nil {
		t.Fatalf("Execute(blank id, blank command) error = %v", err)
	}
	if executed.Result.ExitCode != contract.ExitCodeUsage || executed.Result.Stdout != "execute: command is required" {
		t.Fatalf("Execute(blank id, blank command) = %+v, want usage result", executed.Result)
	}

	checks := []struct {
		name string
		run  func() error
	}{
		{
			name: "Get",
			run: func() error {
				_, err := manager.Get(" \t ")
				return err
			},
		},
		{
			name: "Execute",
			run: func() error {
				_, err := manager.Execute(context.Background(), " \t ", "echo hi", contract.ExecutionPolicy{})
				return err
			},
		},
		{
			name: "Checkpoint",
			run: func() error {
				_, err := manager.Checkpoint(context.Background(), " \t ")
				return err
			},
		},
		{
			name: "Resume",
			run: func() error {
				_, err := manager.Resume(context.Background(), " \t ")
				return err
			},
		},
		{
			name: "Close",
			run: func() error {
				_, err := manager.Close(context.Background(), " \t ")
				return err
			},
		},
	}
	for _, tc := range checks {
		t.Run(tc.name, func(t *testing.T) {
			if err := tc.run(); !errors.Is(err, ErrSessionNotFound) {
				t.Fatalf("%s(blank id) error = %v, want ErrSessionNotFound", tc.name, err)
			}
		})
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

func TestSessionManagerPersistsWorkingDirAcrossExecuteResume(t *testing.T) {
	manager := NewSessionManager(SessionManagerOptions{NewID: func() string { return "sess_cwd" }})
	session, err := manager.Create(context.Background(), Options{
		HostRoot: t.TempDir(),
		Profile:  contract.ProfileBashPlus,
		Policy: contract.ExecutionPolicy{
			WriteMode:        contract.WriteModeFull,
			MaxWriteBytes:    1 << 20,
			MaxPipelineDepth: 16,
			MaxOutputBytes:   4 << 20,
			Timeout:          contract.DefaultPolicy().Timeout,
		},
	})
	if err != nil {
		t.Fatalf("create session failed: %v", err)
	}

	if executed, err := manager.Execute(context.Background(), session.SessionID, "mkdir -p /task_outputs/project && cd /task_outputs/project", contract.ExecutionPolicy{}); err != nil {
		t.Fatalf("initial execute failed: %v", err)
	} else if executed.Session.State.WorkingDir != "/task_outputs/project" {
		t.Fatalf("unexpected working dir after cd: %+v", executed.Session.State)
	}

	executed, err := manager.Execute(context.Background(), session.SessionID, "pwd", contract.ExecutionPolicy{})
	if err != nil {
		t.Fatalf("pwd execute failed: %v", err)
	}
	if strings.TrimSpace(executed.Result.Stdout) != "/task_outputs/project" {
		t.Fatalf("pwd output = %q, want /task_outputs/project", executed.Result.Stdout)
	}

	if _, err := manager.Close(context.Background(), session.SessionID); err != nil {
		t.Fatalf("close failed: %v", err)
	}
	if _, err := manager.Resume(context.Background(), session.SessionID); err != nil {
		t.Fatalf("resume failed: %v", err)
	}
	executed, err = manager.Execute(context.Background(), session.SessionID, "echo hello > note.txt; cat note.txt", contract.ExecutionPolicy{})
	if err != nil {
		t.Fatalf("relative execute after resume failed: %v", err)
	}
	if strings.TrimSpace(executed.Result.Stdout) != "hello" {
		t.Fatalf("unexpected relative execute output: %+v", executed.Result)
	}
	if executed.Session.State.WorkingDir != "/task_outputs/project" {
		t.Fatalf("unexpected working dir after resume execute: %+v", executed.Session.State)
	}
}

func TestSessionManagerExecuteSerializesSameSession(t *testing.T) {
	adapter := &testMemoryAdapter{
		observeStarted: make(chan struct{}),
		observeRelease: make(chan struct{}),
	}
	hostRoot := t.TempDir()
	targetDir := filepath.Join(hostRoot, "task_outputs", "first")
	virtualTargetDir := "/" + filepath.ToSlash(filepath.Join("task_outputs", "first"))
	if err := os.MkdirAll(targetDir, 0o755); err != nil {
		t.Fatalf("mkdir target dir failed: %v", err)
	}
	manager := NewSessionManager(SessionManagerOptions{NewID: func() string { return "sess_serial" }})
	session, err := manager.Create(context.Background(), Options{
		HostRoot: hostRoot,
		Profile:  contract.ProfileBashPlus,
		Policy:   fullSessionPolicy(),
		Adapters: []contract.SessionAdapter{adapter},
	})
	if err != nil {
		t.Fatalf("create session failed: %v", err)
	}

	firstDone := make(chan struct{})
	secondDone := make(chan struct{})
	errCh := make(chan error, 2)
	var wg sync.WaitGroup
	wg.Add(2)
	results := make(chan SessionExecution, 2)

	go func() {
		defer wg.Done()
		executed, execErr := manager.Execute(context.Background(), session.SessionID, "cd "+virtualTargetDir, contract.ExecutionPolicy{})
		if execErr == nil {
			results <- executed
		}
		errCh <- execErr
		close(firstDone)
	}()

	select {
	case <-adapter.observeStarted:
	case <-time.After(time.Second):
		t.Fatal("first execute did not reach observe phase")
	}

	go func() {
		defer wg.Done()
		executed, execErr := manager.Execute(context.Background(), session.SessionID, "pwd", contract.ExecutionPolicy{})
		if execErr == nil {
			results <- executed
		}
		errCh <- execErr
		close(secondDone)
	}()

	select {
	case <-secondDone:
		t.Fatal("second execute completed before first released session lock")
	case <-time.After(50 * time.Millisecond):
	}
	if adapter.observeCalls != 1 {
		t.Fatalf("observeCalls while first execute blocked = %d, want 1", adapter.observeCalls)
	}

	close(adapter.observeRelease)
	wg.Wait()
	close(errCh)
	for execErr := range errCh {
		if execErr != nil {
			t.Fatalf("Execute(...) error = %v, want nil", execErr)
		}
	}
	close(results)

	seen := make([]SessionExecution, 0, 2)
	for executed := range results {
		seen = append(seen, executed)
	}
	if len(seen) != 2 {
		t.Fatalf("completed executes = %d, want 2", len(seen))
	}
	var pwdResult SessionExecution
	foundPwd := false
	for _, executed := range seen {
		if strings.TrimSpace(executed.Result.Stdout) == virtualTargetDir {
			pwdResult = executed
			foundPwd = true
		}
	}
	if !foundPwd {
		t.Fatalf("did not observe pwd result with updated session state: %+v", seen)
	}
	if got := pwdResult.Session.State.WorkingDir; got != virtualTargetDir {
		t.Fatalf("session working dir after serialized executes = %q, want %q", got, virtualTargetDir)
	}

	executed, err := manager.Execute(context.Background(), session.SessionID, "pwd", contract.ExecutionPolicy{})
	if err != nil {
		t.Fatalf("final pwd execute failed: %v", err)
	}
	if got := strings.TrimSpace(executed.Result.Stdout); got != virtualTargetDir {
		t.Fatalf("pwd after serialized execute = %q, want %q", got, virtualTargetDir)
	}
}

func TestSessionManagerTracksAndCancelsActiveExecution(t *testing.T) {
	clock := newSteppingClock(
		time.Date(2026, 5, 20, 9, 0, 0, 0, time.UTC),
		time.Date(2026, 5, 20, 9, 0, 1, 0, time.UTC),
		time.Date(2026, 5, 20, 9, 0, 2, 0, time.UTC),
		time.Date(2026, 5, 20, 9, 0, 3, 0, time.UTC),
	)
	manager := NewSessionManager(SessionManagerOptions{
		Now:            clock.Now,
		NewID:          func() string { return "sess_cancel" },
		NewExecutionID: func() string { return "exec_cancel" },
	})

	runStarted := make(chan struct{})
	runReleased := make(chan struct{})
	session, err := manager.Create(context.Background(), Options{
		HostRoot: t.TempDir(),
		Profile:  contract.ProfileCoreStrict,
		Policy:   contract.DefaultPolicy(),
		ExternalCallbacks: fs.ExternalCallbacks{
			ListExternalCommands: func(context.Context) ([]contract.ExternalCommand, error) {
				return []contract.ExternalCommand{{Name: "blocker", Summary: "blocks until canceled"}}, nil
			},
			RunExternalCommand: func(ctx context.Context, req contract.ExternalCommandRequest) (contract.ExternalCommandResult, error) {
				if req.Command != "blocker" {
					return contract.ExternalCommandResult{}, contract.ErrUnsupported
				}
				close(runStarted)
				<-ctx.Done()
				close(runReleased)
				return contract.ExternalCommandResult{}, ctx.Err()
			},
		},
	})
	if err != nil {
		t.Fatalf("create session failed: %v", err)
	}

	done := make(chan SessionExecution, 1)
	errCh := make(chan error, 1)
	go func() {
		executed, execErr := manager.Execute(context.Background(), session.SessionID, "blocker", contract.ExecutionPolicy{})
		if execErr == nil {
			done <- executed
		}
		errCh <- execErr
	}()

	select {
	case <-runStarted:
	case <-time.After(time.Second):
		t.Fatal("active execution did not start")
	}

	current, err := manager.Get(session.SessionID)
	if err != nil {
		t.Fatalf("Get(%q) error = %v", session.SessionID, err)
	}
	if current.ActiveExecution == nil {
		t.Fatalf("Get(%q).ActiveExecution = nil, want running state", session.SessionID)
	}
	if current.ActiveExecution.ExecutionID == "" {
		t.Fatalf("active execution_id empty: %+v", current.ActiveExecution)
	}
	if current.ActiveExecution.CommandLine != "blocker" {
		t.Fatalf("active command_line = %q, want blocker", current.ActiveExecution.CommandLine)
	}
	if current.ActiveExecution.Status != contract.SessionExecutionStatusRunning {
		t.Fatalf("active status = %q, want %q", current.ActiveExecution.Status, contract.SessionExecutionStatusRunning)
	}
	if !current.ActiveExecution.StatusUpdatedAt.Equal(current.ActiveExecution.StartedAt) {
		t.Fatalf("running status_updated_at = %v, want %v", current.ActiveExecution.StatusUpdatedAt, current.ActiveExecution.StartedAt)
	}

	canceled, err := manager.Cancel(session.SessionID, current.ActiveExecution.ExecutionID)
	if err != nil {
		t.Fatalf("Cancel(%q) error = %v", session.SessionID, err)
	}
	if canceled.ActiveExecution == nil {
		t.Fatalf("Cancel(%q) active_execution = nil, want canceling state", session.SessionID)
	}
	if canceled.ActiveExecution.Status != contract.SessionExecutionStatusCanceling {
		t.Fatalf("cancel status = %q, want %q", canceled.ActiveExecution.Status, contract.SessionExecutionStatusCanceling)
	}
	if canceled.ActiveExecution.ExecutionID != current.ActiveExecution.ExecutionID {
		t.Fatalf("cancel execution_id = %q, want %q", canceled.ActiveExecution.ExecutionID, current.ActiveExecution.ExecutionID)
	}
	if want := clock.Values[2]; !canceled.ActiveExecution.StatusUpdatedAt.Equal(want) {
		t.Fatalf("cancel status_updated_at = %v, want %v", canceled.ActiveExecution.StatusUpdatedAt, want)
	}

	select {
	case <-runReleased:
	case <-time.After(time.Second):
		t.Fatal("active execution was not canceled")
	}

	if execErr := <-errCh; execErr != nil {
		t.Fatalf("Execute(...) error = %v, want nil", execErr)
	}
	executed := <-done
	if executed.Result.ExecutionID != current.ActiveExecution.ExecutionID {
		t.Fatalf("result execution_id = %q, want %q", executed.Result.ExecutionID, current.ActiveExecution.ExecutionID)
	}
	if executed.Result.ExitCode != contract.ExitCodeGeneral {
		t.Fatalf("canceled execute exit_code = %d, want %d", executed.Result.ExitCode, contract.ExitCodeGeneral)
	}
	if !executed.Result.Trace.Canceled {
		t.Fatalf("canceled execute trace = %+v, want canceled=true", executed.Result.Trace)
	}

	finalSession, err := manager.Get(session.SessionID)
	if err != nil {
		t.Fatalf("Get(%q) after cancel error = %v", session.SessionID, err)
	}
	if finalSession.ActiveExecution != nil {
		t.Fatalf("final active execution = %+v, want nil", finalSession.ActiveExecution)
	}
	if want := clock.Values[3]; !finalSession.UpdatedAt.Equal(want) {
		t.Fatalf("final updated_at = %v, want %v", finalSession.UpdatedAt, want)
	}
	if _, err := manager.Cancel(session.SessionID, current.ActiveExecution.ExecutionID); !errors.Is(err, ErrSessionNotRunning) {
		t.Fatalf("Cancel(%q) after completion error = %v, want ErrSessionNotRunning", session.SessionID, err)
	}
}

func TestSessionManagerCancelRejectsChangedExecution(t *testing.T) {
	clock := newSteppingClock(
		time.Date(2026, 5, 20, 11, 0, 0, 0, time.UTC),
		time.Date(2026, 5, 20, 11, 0, 1, 0, time.UTC),
	)
	manager := NewSessionManager(SessionManagerOptions{
		Now:            clock.Now,
		NewID:          func() string { return "sess_changed" },
		NewExecutionID: func() string { return "exec_current" },
	})

	runStarted := make(chan struct{})
	runReleased := make(chan struct{})
	session, err := manager.Create(context.Background(), Options{
		HostRoot: t.TempDir(),
		Profile:  contract.ProfileCoreStrict,
		Policy:   contract.DefaultPolicy(),
		ExternalCallbacks: fs.ExternalCallbacks{
			ListExternalCommands: func(context.Context) ([]contract.ExternalCommand, error) {
				return []contract.ExternalCommand{{Name: "blocker", Summary: "blocks until canceled"}}, nil
			},
			RunExternalCommand: func(ctx context.Context, req contract.ExternalCommandRequest) (contract.ExternalCommandResult, error) {
				close(runStarted)
				<-ctx.Done()
				close(runReleased)
				return contract.ExternalCommandResult{}, ctx.Err()
			},
		},
	})
	if err != nil {
		t.Fatalf("create session failed: %v", err)
	}

	errCh := make(chan error, 1)
	go func() {
		_, execErr := manager.Execute(context.Background(), session.SessionID, "blocker", contract.ExecutionPolicy{})
		errCh <- execErr
	}()
	select {
	case <-runStarted:
	case <-time.After(time.Second):
		t.Fatal("active execution did not start")
	}

	if _, err := manager.Cancel(session.SessionID, "exec_other"); !errors.Is(err, ErrExecutionChanged) {
		t.Fatalf("Cancel(%q, mismatched execution) error = %v, want ErrExecutionChanged", session.SessionID, err)
	}
	if _, err := manager.Cancel(session.SessionID, "exec_current"); err != nil {
		t.Fatalf("Cancel(%q, current execution) error = %v", session.SessionID, err)
	}
	select {
	case <-runReleased:
	case <-time.After(time.Second):
		t.Fatal("active execution was not canceled")
	}
	if execErr := <-errCh; execErr != nil {
		t.Fatalf("Execute(...) error = %v, want nil", execErr)
	}
}

func TestSessionManagerCancelRequiresExecutionID(t *testing.T) {
	manager := NewSessionManager(SessionManagerOptions{
		Now:            func() time.Time { return time.Date(2026, 5, 20, 11, 30, 0, 0, time.UTC) },
		NewID:          func() string { return "sess_require_exec" },
		NewExecutionID: func() string { return "exec_require_exec" },
	})

	runStarted := make(chan struct{})
	runReleased := make(chan struct{})
	session, err := manager.Create(context.Background(), Options{
		HostRoot: t.TempDir(),
		Profile:  contract.ProfileCoreStrict,
		Policy:   contract.DefaultPolicy(),
		ExternalCallbacks: fs.ExternalCallbacks{
			ListExternalCommands: func(context.Context) ([]contract.ExternalCommand, error) {
				return []contract.ExternalCommand{{Name: "blocker", Summary: "blocks until canceled"}}, nil
			},
			RunExternalCommand: func(ctx context.Context, req contract.ExternalCommandRequest) (contract.ExternalCommandResult, error) {
				close(runStarted)
				<-ctx.Done()
				close(runReleased)
				return contract.ExternalCommandResult{}, ctx.Err()
			},
		},
	})
	if err != nil {
		t.Fatalf("create session failed: %v", err)
	}

	errCh := make(chan error, 1)
	go func() {
		_, execErr := manager.Execute(context.Background(), session.SessionID, "blocker", contract.ExecutionPolicy{})
		errCh <- execErr
	}()
	select {
	case <-runStarted:
	case <-time.After(time.Second):
		t.Fatal("active execution did not start")
	}

	if _, err := manager.Cancel(session.SessionID, ""); !errors.Is(err, ErrExecutionRequired) {
		t.Fatalf("Cancel(%q, empty execution id) error = %v, want ErrExecutionRequired", session.SessionID, err)
	}
	if _, err := manager.Cancel(session.SessionID, "exec_require_exec"); err != nil {
		t.Fatalf("Cancel(%q, current execution) error = %v", session.SessionID, err)
	}
	select {
	case <-runReleased:
	case <-time.After(time.Second):
		t.Fatal("active execution was not canceled")
	}
	if execErr := <-errCh; execErr != nil {
		t.Fatalf("Execute(...) error = %v, want nil", execErr)
	}
}

func TestSessionManagerGetClearsFinishedActiveExecution(t *testing.T) {
	manager := NewSessionManager(SessionManagerOptions{
		Now:            func() time.Time { return time.Date(2026, 5, 20, 12, 30, 0, 0, time.UTC) },
		NewID:          func() string { return "sess_get_clear" },
		NewExecutionID: func() string { return "exec_get_clear" },
	})
	session, err := manager.Create(context.Background(), Options{
		HostRoot: t.TempDir(),
		Profile:  contract.ProfileCoreStrict,
		Policy:   contract.DefaultPolicy(),
	})
	if err != nil {
		t.Fatalf("create session failed: %v", err)
	}

	record, err := manager.lookupManaged(session.SessionID)
	if err != nil {
		t.Fatalf("lookupManaged(%q) error = %v", session.SessionID, err)
	}
	record.activeExecution = &activeExecutionControl{
		state: *newActiveExecutionState("exec_get_clear", "echo hi", time.Date(2026, 5, 20, 12, 30, 1, 0, time.UTC), contract.SessionExecutionStatusRunning),
		done:  make(chan struct{}),
	}
	record.snapshot.ActiveExecution = cloneActiveExecutionPtr(&record.activeExecution.state)
	close(record.activeExecution.done)

	got, err := manager.Get(session.SessionID)
	if err != nil {
		t.Fatalf("Get(%q) error = %v", session.SessionID, err)
	}
	if got.ActiveExecution != nil {
		t.Fatalf("Get(%q).ActiveExecution = %+v, want nil after finished execution", session.SessionID, got.ActiveExecution)
	}
}

func TestSessionManagerGetClearsFinishedActiveExecutionWithoutDroppingSnapshotState(t *testing.T) {
	finalWorkingDir := strings.Join([]string{"", "task_outputs", "finalized"}, "/")
	manager := NewSessionManager(SessionManagerOptions{
		Now:            func() time.Time { return time.Date(2026, 5, 20, 12, 45, 0, 0, time.UTC) },
		NewID:          func() string { return "sess_get_final" },
		NewExecutionID: func() string { return "exec_get_final" },
	})
	session, err := manager.Create(context.Background(), Options{
		HostRoot: t.TempDir(),
		Profile:  contract.ProfileCoreStrict,
		Policy:   contract.DefaultPolicy(),
	})
	if err != nil {
		t.Fatalf("create session failed: %v", err)
	}

	record, err := manager.lookupManaged(session.SessionID)
	if err != nil {
		t.Fatalf("lookupManaged(%q) error = %v", session.SessionID, err)
	}
	record.activeExecution = &activeExecutionControl{
		state: *newActiveExecutionState("exec_get_final", "echo hi", time.Date(2026, 5, 20, 12, 45, 1, 0, time.UTC), contract.SessionExecutionStatusRunning),
		done:  make(chan struct{}),
	}
	record.snapshot.ActiveExecution = cloneActiveExecutionPtr(&record.activeExecution.state)
	record.snapshot.UpdatedAt = time.Date(2026, 5, 20, 12, 45, 2, 0, time.UTC)
	record.snapshot.State.WorkingDir = finalWorkingDir
	record.snapshot.State.EnvVars = map[string]string{"FINAL_FLAG": "set"}
	close(record.activeExecution.done)

	got, err := manager.Get(session.SessionID)
	if err != nil {
		t.Fatalf("Get(%q) error = %v", session.SessionID, err)
	}
	if got.ActiveExecution != nil {
		t.Fatalf("Get(%q).ActiveExecution = %+v, want nil after finished execution", session.SessionID, got.ActiveExecution)
	}
	if got.UpdatedAt != record.snapshot.UpdatedAt {
		t.Fatalf("Get(%q).UpdatedAt = %v, want %v", session.SessionID, got.UpdatedAt, record.snapshot.UpdatedAt)
	}
	if got.State.WorkingDir != finalWorkingDir {
		t.Fatalf("Get(%q).State.WorkingDir = %q, want finalized snapshot", session.SessionID, got.State.WorkingDir)
	}
	if got.State.EnvVars["FINAL_FLAG"] != "set" {
		t.Fatalf("Get(%q).State.EnvVars = %#v, want finalized snapshot", session.SessionID, got.State.EnvVars)
	}
}

type steppingClock struct {
	Values []time.Time
	idx    int
}

func newSteppingClock(values ...time.Time) *steppingClock {
	return &steppingClock{Values: append([]time.Time(nil), values...)}
}

func (c *steppingClock) Now() time.Time {
	if len(c.Values) == 0 {
		return time.Time{}
	}
	value := c.Values[c.idx]
	if c.idx < len(c.Values)-1 {
		c.idx++
	}
	return value
}

func TestSessionManagerAdapterCreateFailureDoesNotPersistSession(t *testing.T) {
	adapter := &testMemoryAdapter{failCreate: true}
	manager := NewSessionManager(SessionManagerOptions{NewID: func() string { return "sess_create_fail" }})

	_, err := manager.Create(context.Background(), Options{
		HostRoot: t.TempDir(),
		Profile:  contract.ProfileCoreStrict,
		Policy:   fullSessionPolicy(),
		Adapters: []contract.SessionAdapter{adapter},
	})
	if err == nil || !strings.Contains(err.Error(), "adapter memory_test create failed: create failed") {
		t.Fatalf("Create(...) error = %v, want wrapped create failure", err)
	}
	if _, err := manager.Get("sess_create_fail"); !errors.Is(err, ErrSessionNotFound) {
		t.Fatalf("Get(%q) error = %v, want ErrSessionNotFound", "sess_create_fail", err)
	}
}

func TestSessionManagerAdapterLifecycleMemoryProjection(t *testing.T) {
	adapter := &testMemoryAdapter{}
	manager := NewSessionManager(SessionManagerOptions{NewID: func() string { return "sess_memory" }})
	session, err := manager.Create(context.Background(), Options{
		HostRoot: t.TempDir(),
		Profile:  contract.ProfileCoreStrict,
		Policy:   contract.DefaultPolicy(),
		Adapters: []contract.SessionAdapter{adapter},
	})
	if err != nil {
		t.Fatalf("create session failed: %v", err)
	}

	if _, err := manager.Execute(context.Background(), session.SessionID, "echo hello", contract.ExecutionPolicy{}); err != nil {
		t.Fatalf("observe execute failed: %v", err)
	}
	memoryView, err := manager.Execute(context.Background(), session.SessionID, "cat /memory/log.md", contract.ExecutionPolicy{})
	if err != nil {
		t.Fatalf("cat /memory/log.md failed: %v", err)
	}
	if strings.TrimSpace(memoryView.Result.Stdout) != "hello" {
		t.Fatalf("unexpected memory log output: %+v", memoryView.Result)
	}

	checkpoint, err := manager.Checkpoint(context.Background(), session.SessionID)
	if err != nil {
		t.Fatalf("checkpoint failed: %v", err)
	}
	if len(checkpoint.State.Opaque[adapter.AdapterID()]) == 0 {
		t.Fatalf("expected adapter opaque state at checkpoint: %+v", checkpoint.State)
	}
	if _, err := manager.Close(context.Background(), session.SessionID); err != nil {
		t.Fatalf("close failed: %v", err)
	}
	if _, err := manager.Resume(context.Background(), session.SessionID); err != nil {
		t.Fatalf("resume failed: %v", err)
	}
	resumedMemory, err := manager.Execute(context.Background(), session.SessionID, "cat /memory/log.md", contract.ExecutionPolicy{})
	if err != nil {
		t.Fatalf("cat /memory/log.md after resume failed: %v", err)
	}
	if strings.TrimSpace(resumedMemory.Result.Stdout) != "hello" {
		t.Fatalf("unexpected resumed memory output: %+v", resumedMemory.Result)
	}
	if adapter.createCalls != 1 || adapter.observeCalls == 0 || adapter.checkpointCalls == 0 || adapter.resumeCalls == 0 || adapter.closeCalls == 0 {
		t.Fatalf("unexpected lifecycle call counts: %+v", adapter)
	}
}

func TestSessionManagerAdapterObserveFailureDoesNotPartiallyMaterialize(t *testing.T) {
	adapter := &testMemoryAdapter{failObserve: true}
	manager := NewSessionManager(SessionManagerOptions{NewID: func() string { return "sess_memory_fail" }})
	session, err := manager.Create(context.Background(), Options{
		HostRoot: t.TempDir(),
		Profile:  contract.ProfileCoreStrict,
		Policy:   contract.DefaultPolicy(),
		Adapters: []contract.SessionAdapter{adapter},
	})
	if err != nil {
		t.Fatalf("create session failed: %v", err)
	}

	if _, err := manager.Execute(context.Background(), session.SessionID, "echo hello", contract.ExecutionPolicy{}); err == nil || !strings.Contains(err.Error(), "observe failed") {
		t.Fatalf("expected observe failure, got %v", err)
	}

	adapter.failObserve = false
	memoryView, err := manager.Execute(context.Background(), session.SessionID, "cat /memory/log.md", contract.ExecutionPolicy{})
	if err != nil {
		t.Fatalf("cat /memory/log.md failed: %v", err)
	}
	if strings.TrimSpace(memoryView.Result.Stdout) != "" {
		t.Fatalf("expected empty memory log after failed observe, got %+v", memoryView.Result)
	}
}

func TestSessionManagerAdapterPhaseFailuresPreserveState(t *testing.T) {
	t.Run("Observe", func(t *testing.T) {
		adapter := &testMemoryAdapter{failObserve: true}
		manager := NewSessionManager(SessionManagerOptions{NewID: func() string { return "sess_observe_fail" }})
		session, err := manager.Create(context.Background(), Options{
			HostRoot: t.TempDir(),
			Profile:  contract.ProfileCoreStrict,
			Policy:   fullSessionPolicy(),
			Adapters: []contract.SessionAdapter{adapter},
		})
		if err != nil {
			t.Fatalf("create session failed: %v", err)
		}
		before := mustLookupSession(t, manager, session.SessionID)
		requireMountPoint(t, before.adapterMounts, "/memory")

		if _, err := manager.Execute(context.Background(), session.SessionID, "echo hello", contract.ExecutionPolicy{}); err == nil || !strings.Contains(err.Error(), "adapter memory_test observe failed: observe failed") {
			t.Fatalf("expected observe failure, got %v", err)
		}
		after := mustLookupSession(t, manager, session.SessionID)
		if !after.active || after.runtime == nil {
			t.Fatalf("observe failure mutated runtime state: active=%v runtime_nil=%v", after.active, after.runtime == nil)
		}
		requireMountPoint(t, after.adapterMounts, "/memory")
		if string(after.snapshot.State.Opaque[adapter.AdapterID()]) != string(before.snapshot.State.Opaque[adapter.AdapterID()]) {
			t.Fatalf("snapshot opaque changed on observe failure: before=%s after=%s", before.snapshot.State.Opaque[adapter.AdapterID()], after.snapshot.State.Opaque[adapter.AdapterID()])
		}
		if string(after.checkpoint.State.Opaque[adapter.AdapterID()]) != string(before.checkpoint.State.Opaque[adapter.AdapterID()]) {
			t.Fatalf("checkpoint opaque changed on observe failure: before=%s after=%s", before.checkpoint.State.Opaque[adapter.AdapterID()], after.checkpoint.State.Opaque[adapter.AdapterID()])
		}

		adapter.failObserve = false
		executed, err := manager.Execute(context.Background(), session.SessionID, "cat /memory/log.md", contract.ExecutionPolicy{WriteMode: contract.WriteModeReadOnly})
		if err != nil {
			t.Fatalf("read-only execute after observe failure failed: %v", err)
		}
		if strings.TrimSpace(executed.Result.Stdout) != "" {
			t.Fatalf("memory log after observe failure = %q, want empty", executed.Result.Stdout)
		}
		if executed.Runtime.Ops().Policy.WriteMode != contract.WriteModeReadOnly {
			t.Fatalf("execute runtime policy = %q, want read-only", executed.Runtime.Ops().Policy.WriteMode)
		}
		current := mustLookupSession(t, manager, session.SessionID)
		if current.runtime == nil || current.runtime.Ops().Policy.WriteMode != fullSessionPolicy().WriteMode {
			t.Fatalf("stored runtime policy after read-only execute = %q, want ceiling %q", current.runtime.Ops().Policy.WriteMode, fullSessionPolicy().WriteMode)
		}
	})

	t.Run("Checkpoint", func(t *testing.T) {
		adapter := &testMemoryAdapter{}
		manager := NewSessionManager(SessionManagerOptions{NewID: func() string { return "sess_checkpoint_fail" }})
		session, err := manager.Create(context.Background(), Options{
			HostRoot: t.TempDir(),
			Profile:  contract.ProfileCoreStrict,
			Policy:   fullSessionPolicy(),
			Adapters: []contract.SessionAdapter{adapter},
		})
		if err != nil {
			t.Fatalf("create session failed: %v", err)
		}
		if _, err := manager.Execute(context.Background(), session.SessionID, "echo hello", contract.ExecutionPolicy{}); err != nil {
			t.Fatalf("seed execute failed: %v", err)
		}
		before := mustLookupSession(t, manager, session.SessionID)
		requireMountPoint(t, before.adapterMounts, "/memory")

		adapter.failCheckpoint = true
		if _, err := manager.Checkpoint(context.Background(), session.SessionID); err == nil || !strings.Contains(err.Error(), "adapter memory_test checkpoint failed: checkpoint failed") {
			t.Fatalf("expected checkpoint failure, got %v", err)
		}
		after := mustLookupSession(t, manager, session.SessionID)
		requireMountPoint(t, after.adapterMounts, "/memory")
		if string(after.snapshot.State.Opaque[adapter.AdapterID()]) != string(before.snapshot.State.Opaque[adapter.AdapterID()]) {
			t.Fatalf("snapshot opaque changed on checkpoint failure: before=%s after=%s", before.snapshot.State.Opaque[adapter.AdapterID()], after.snapshot.State.Opaque[adapter.AdapterID()])
		}
		if string(after.checkpoint.State.Opaque[adapter.AdapterID()]) != string(before.checkpoint.State.Opaque[adapter.AdapterID()]) {
			t.Fatalf("checkpoint opaque changed on checkpoint failure: before=%s after=%s", before.checkpoint.State.Opaque[adapter.AdapterID()], after.checkpoint.State.Opaque[adapter.AdapterID()])
		}

		adapter.failCheckpoint = false
		executed, err := manager.Execute(context.Background(), session.SessionID, "cat /memory/log.md", contract.ExecutionPolicy{WriteMode: contract.WriteModeReadOnly})
		if err != nil {
			t.Fatalf("read-only execute after checkpoint failure failed: %v", err)
		}
		if strings.TrimSpace(executed.Result.Stdout) != "hello" {
			t.Fatalf("memory log after checkpoint failure = %q, want hello", executed.Result.Stdout)
		}
	})

	t.Run("Close", func(t *testing.T) {
		adapter := &testMemoryAdapter{}
		manager := NewSessionManager(SessionManagerOptions{NewID: func() string { return "sess_close_fail" }})
		session, err := manager.Create(context.Background(), Options{
			HostRoot: t.TempDir(),
			Profile:  contract.ProfileCoreStrict,
			Policy:   fullSessionPolicy(),
			Adapters: []contract.SessionAdapter{adapter},
		})
		if err != nil {
			t.Fatalf("create session failed: %v", err)
		}
		if _, err := manager.Execute(context.Background(), session.SessionID, "echo hello", contract.ExecutionPolicy{}); err != nil {
			t.Fatalf("seed execute failed: %v", err)
		}
		before := mustLookupSession(t, manager, session.SessionID)
		requireMountPoint(t, before.adapterMounts, "/memory")

		adapter.failClose = true
		if _, err := manager.Close(context.Background(), session.SessionID); err == nil || !strings.Contains(err.Error(), "adapter memory_test close failed: close failed") {
			t.Fatalf("expected close failure, got %v", err)
		}
		after := mustLookupSession(t, manager, session.SessionID)
		if !after.active || after.runtime == nil {
			t.Fatalf("close failure mutated session activity: active=%v runtime_nil=%v", after.active, after.runtime == nil)
		}
		requireMountPoint(t, after.adapterMounts, "/memory")
		if string(after.snapshot.State.Opaque[adapter.AdapterID()]) != string(before.snapshot.State.Opaque[adapter.AdapterID()]) {
			t.Fatalf("snapshot opaque changed on close failure: before=%s after=%s", before.snapshot.State.Opaque[adapter.AdapterID()], after.snapshot.State.Opaque[adapter.AdapterID()])
		}
		if string(after.checkpoint.State.Opaque[adapter.AdapterID()]) != string(before.checkpoint.State.Opaque[adapter.AdapterID()]) {
			t.Fatalf("checkpoint opaque changed on close failure: before=%s after=%s", before.checkpoint.State.Opaque[adapter.AdapterID()], after.checkpoint.State.Opaque[adapter.AdapterID()])
		}

		adapter.failClose = false
		executed, err := manager.Execute(context.Background(), session.SessionID, "cat /memory/log.md", contract.ExecutionPolicy{WriteMode: contract.WriteModeReadOnly})
		if err != nil {
			t.Fatalf("read-only execute after close failure failed: %v", err)
		}
		if strings.TrimSpace(executed.Result.Stdout) != "hello" {
			t.Fatalf("memory log after close failure = %q, want hello", executed.Result.Stdout)
		}
	})
}

func TestSessionManagerResumeLifecycleEdges(t *testing.T) {
	adapter := &testMemoryAdapter{}
	manager := NewSessionManager(SessionManagerOptions{NewID: func() string { return "sess_resume_edges" }})
	session, err := manager.Create(context.Background(), Options{
		HostRoot: t.TempDir(),
		Profile:  contract.ProfileCoreStrict,
		Policy:   fullSessionPolicy(),
		Adapters: []contract.SessionAdapter{adapter},
	})
	if err != nil {
		t.Fatalf("create session failed: %v", err)
	}

	active, err := manager.Resume(context.Background(), session.SessionID)
	if err != nil {
		t.Fatalf("resume active session failed: %v", err)
	}
	if active.SessionID != session.SessionID {
		t.Fatalf("resume active session returned %q, want %q", active.SessionID, session.SessionID)
	}
	if adapter.resumeCalls != 0 {
		t.Fatalf("resume active session should not invoke adapter, got %d calls", adapter.resumeCalls)
	}

	if _, err := manager.Execute(context.Background(), session.SessionID, "echo hello", contract.ExecutionPolicy{}); err != nil {
		t.Fatalf("seed execute failed: %v", err)
	}
	if _, err := manager.Close(context.Background(), session.SessionID); err != nil {
		t.Fatalf("close failed: %v", err)
	}
	before := mustLookupSession(t, manager, session.SessionID)
	if before.active || before.runtime != nil {
		t.Fatalf("closed session state = active:%v runtime_nil:%v, want inactive nil runtime", before.active, before.runtime == nil)
	}

	adapter.failResume = true
	if _, err := manager.Resume(context.Background(), session.SessionID); err == nil || !strings.Contains(err.Error(), "adapter memory_test resume failed: resume failed") {
		t.Fatalf("expected resume failure, got %v", err)
	}
	after := mustLookupSession(t, manager, session.SessionID)
	if after.active || after.runtime != nil {
		t.Fatalf("resume failure mutated session activity: active=%v runtime_nil=%v", after.active, after.runtime == nil)
	}
	if string(after.snapshot.State.Opaque[adapter.AdapterID()]) != string(before.snapshot.State.Opaque[adapter.AdapterID()]) {
		t.Fatalf("snapshot opaque changed on resume failure: before=%s after=%s", before.snapshot.State.Opaque[adapter.AdapterID()], after.snapshot.State.Opaque[adapter.AdapterID()])
	}
	requireMountPoint(t, after.adapterMounts, "/memory")
	if adapter.resumeCalls != 1 {
		t.Fatalf("resume adapter call count after failed resume = %d, want 1", adapter.resumeCalls)
	}

	adapter.failResume = false
	if _, err := manager.Resume(context.Background(), session.SessionID); err != nil {
		t.Fatalf("resume after failure failed: %v", err)
	}
	status, err := manager.Execute(context.Background(), session.SessionID, "cat /memory/status.json", contract.ExecutionPolicy{})
	if err != nil {
		t.Fatalf("cat /memory/status.json failed: %v", err)
	}
	if !strings.Contains(status.Result.Stdout, `"freshness":"resumed"`) {
		t.Fatalf("resume status output = %q, want freshness=resumed", status.Result.Stdout)
	}
	if adapter.resumeCalls != 2 {
		t.Fatalf("resume adapter call count = %d, want 2", adapter.resumeCalls)
	}
}

type testMemoryAdapter struct {
	failCreate         bool
	failObserve        bool
	failResume         bool
	failCheckpoint     bool
	failClose          bool
	observeStarted     chan struct{}
	observeRelease     chan struct{}
	observeStartedOnce sync.Once
	createCalls        int
	resumeCalls        int
	observeCalls       int
	checkpointCalls    int
	closeCalls         int
}

type testMemoryState struct {
	Entries   []string `json:"entries"`
	Freshness string   `json:"freshness"`
}

func (a *testMemoryAdapter) AdapterID() string { return "memory_test" }

func (a *testMemoryAdapter) CreateSession(ctx context.Context, session contract.Session) (contract.AdapterProjection, error) {
	_ = ctx
	a.createCalls++
	if a.failCreate {
		return contract.AdapterProjection{}, errors.New("create failed")
	}
	return a.buildProjection(a.stateFromSession(session, "created"))
}

func (a *testMemoryAdapter) ResumeSession(ctx context.Context, session contract.Session) (contract.AdapterProjection, error) {
	_ = ctx
	a.resumeCalls++
	if a.failResume {
		return contract.AdapterProjection{}, errors.New("resume failed")
	}
	return a.buildProjection(a.stateFromSession(session, "resumed"))
}

func (a *testMemoryAdapter) ObserveExecution(ctx context.Context, session contract.Session, result contract.ExecutionResult) (contract.AdapterProjection, error) {
	a.observeCalls++
	if a.observeStarted != nil {
		a.observeStartedOnce.Do(func() { close(a.observeStarted) })
	}
	if a.observeRelease != nil {
		select {
		case <-a.observeRelease:
		case <-ctx.Done():
			return contract.AdapterProjection{}, ctx.Err()
		}
	}
	if a.failObserve {
		return contract.AdapterProjection{}, errors.New("observe failed")
	}
	state := a.stateFromSession(session, "observed")
	if !referencesMemory(result.Trace) {
		trimmed := strings.TrimSpace(result.Stdout)
		if trimmed != "" {
			state.Entries = append(state.Entries, trimmed)
		}
	}
	return a.buildProjection(state)
}

func (a *testMemoryAdapter) CheckpointSession(ctx context.Context, session contract.Session) (contract.AdapterProjection, error) {
	_ = ctx
	a.checkpointCalls++
	if a.failCheckpoint {
		return contract.AdapterProjection{}, errors.New("checkpoint failed")
	}
	return a.buildProjection(a.stateFromSession(session, "checkpointed"))
}

func (a *testMemoryAdapter) CloseSession(ctx context.Context, session contract.Session) (contract.AdapterProjection, error) {
	_ = ctx
	a.closeCalls++
	if a.failClose {
		return contract.AdapterProjection{}, errors.New("close failed")
	}
	return a.buildProjection(a.stateFromSession(session, "closed"))
}

func (a *testMemoryAdapter) stateFromSession(session contract.Session, freshness string) testMemoryState {
	state := testMemoryState{Freshness: freshness}
	if raw := session.State.Opaque[a.AdapterID()]; len(raw) > 0 {
		_ = json.Unmarshal(raw, &state)
		state.Freshness = freshness
	}
	return state
}

func (a *testMemoryAdapter) buildProjection(state testMemoryState) (contract.AdapterProjection, error) {
	raw, err := json.Marshal(state)
	if err != nil {
		return contract.AdapterProjection{}, err
	}
	files := map[string]string{
		"/memory/log.md":      strings.Join(state.Entries, "\n"),
		"/memory/status.json": string(raw),
	}
	memoryMount, err := mount.NewStaticMount("/memory", "memory", files)
	if err != nil {
		return contract.AdapterProjection{}, err
	}
	return contract.AdapterProjection{
		OpaqueState: raw,
		Memory: contract.MemoryProjection{
			Mount:     memoryMount,
			Freshness: state.Freshness,
		},
	}, nil
}

func referencesMemory(trace contract.ExecutionTrace) bool {
	for _, pathValue := range trace.RequestedPaths {
		if strings.HasPrefix(pathValue, "/memory") {
			return true
		}
	}
	for _, pathValue := range trace.ReadPaths {
		if strings.HasPrefix(pathValue, "/memory") {
			return true
		}
	}
	return false
}

func fullSessionPolicy() contract.ExecutionPolicy {
	return contract.ExecutionPolicy{
		WriteMode:        contract.WriteModeFull,
		MaxWriteBytes:    1 << 20,
		MaxPipelineDepth: 16,
		MaxOutputBytes:   4 << 20,
		Timeout:          contract.DefaultPolicy().Timeout,
	}
}

func mustLookupSession(t *testing.T, manager *SessionManager, sessionID string) *managedSession {
	t.Helper()
	record, err := manager.lookup(sessionID)
	if err != nil {
		t.Fatalf("lookup(%q) error = %v", sessionID, err)
	}
	return record
}

func requireMountPoint(t *testing.T, mounts []contract.VirtualMount, want string) {
	t.Helper()
	if len(mounts) != 1 {
		t.Fatalf("mount count = %d, want 1 (%q)", len(mounts), want)
	}
	if got := mounts[0].MountPoint(); got != want {
		t.Fatalf("mount point = %q, want %q", got, want)
	}
}
