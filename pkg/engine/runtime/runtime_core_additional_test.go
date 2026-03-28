package runtime

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/khicago/simsh/pkg/contract"
)

func TestStackNilAndStateHelpers(t *testing.T) {
	var stack *Stack

	result := stack.ExecuteResult(context.Background(), "echo hello")
	if result.ExitCode != contract.ExitCodeGeneral || result.Stdout != "execute: runtime is not initialized" {
		t.Fatalf("(*Stack)(nil).ExecuteResult(...) = %#v, want general error result", result)
	}
	if docs := stack.BuiltinDocs(); docs != nil {
		t.Fatalf("(*Stack)(nil).BuiltinDocs() = %#v, want nil", docs)
	}

	state := stack.SessionState([]string{" /task_outputs/simshrc ", "/task_outputs/simshrc"})
	if !reflect.DeepEqual(state.RCFiles, []string{"/task_outputs/simshrc"}) {
		t.Fatalf("(*Stack)(nil).SessionState(...) = %#v, want normalized rc files", state)
	}

	rawStack := &Stack{ops: contract.Ops{RootDir: "/root", WorkingDir: "/cwd"}}
	if got := rawStack.Ops(); got.WorkingDir != "/cwd" {
		t.Fatalf("Stack.Ops() = %#v, want raw ops with WorkingDir=/cwd", got)
	}

	if got := stackWorkingDir(contract.Ops{GetWorkingDir: func() string { return " /task_outputs " }, WorkingDir: "/fallback", RootDir: "/root"}); got != "/task_outputs" {
		t.Errorf("stackWorkingDir(getter) = %q, want %q", got, "/task_outputs")
	}
	if got := stackWorkingDir(contract.Ops{WorkingDir: "/fallback", RootDir: "/root"}); got != "/fallback" {
		t.Errorf("stackWorkingDir(working_dir) = %q, want %q", got, "/fallback")
	}
	if got := stackWorkingDir(contract.Ops{RootDir: "/root"}); got != "/root" {
		t.Errorf("stackWorkingDir(root) = %q, want %q", got, "/root")
	}
	if got := stackWorkingDir(contract.Ops{}); got != "/" {
		t.Errorf("stackWorkingDir(empty) = %q, want %q", got, "/")
	}
}

func TestStackBuiltinDocsAndSessionState(t *testing.T) {
	stack, err := New(Options{HostRoot: t.TempDir(), Policy: contract.DefaultPolicy()})
	if err != nil {
		t.Fatalf("New(...) error = %v", err)
	}

	docs := stack.BuiltinDocs()
	if len(docs) == 0 {
		t.Fatal("Stack.BuiltinDocs() returned no docs")
	}

	state := stack.SessionState([]string{" /task_outputs/simshrc "})
	if !reflect.DeepEqual(state.RCFiles, []string{"/task_outputs/simshrc"}) {
		t.Errorf("Stack.SessionState(...) = %#v, want normalized rc files", state)
	}
	if state.WorkingDir != "/" {
		t.Errorf("Stack.SessionState(...).WorkingDir = %q, want %q", state.WorkingDir, "/")
	}
}

func TestSessionManagerGetAndRuntimeOptionsForSession(t *testing.T) {
	manager := NewSessionManager(SessionManagerOptions{
		NewID: func() string { return "sess_get" },
		Now:   func() time.Time { return time.Date(2026, 3, 23, 10, 0, 0, 0, time.UTC) },
	})
	session, err := manager.Create(context.Background(), Options{
		HostRoot: t.TempDir(),
		Policy: contract.ExecutionPolicy{
			WriteMode:        contract.WriteModeFull,
			MaxWriteBytes:    1 << 20,
			MaxPipelineDepth: 16,
			MaxOutputBytes:   4 << 20,
			Timeout:          15 * time.Second,
		},
		CommandAliases: map[string][]string{" ll ": {" ls ", "-l"}},
		EnvVars:        map[string]string{"FOO": "bar"},
		RCFiles:        []string{" /task_outputs/simshrc "},
		WorkingDir:     "/task_outputs",
		PathEnv:        []string{"/custom/bin"},
	})
	if err != nil {
		t.Fatalf("Create(...) error = %v", err)
	}

	got, err := manager.Get(session.SessionID)
	if err != nil {
		t.Fatalf("Get(%q) error = %v", session.SessionID, err)
	}
	if got.SessionID != session.SessionID {
		t.Fatalf("Get(%q).SessionID = %q, want %q", session.SessionID, got.SessionID, session.SessionID)
	}
	if _, err := manager.Get("missing"); !errors.Is(err, ErrSessionNotFound) {
		t.Fatalf("Get(%q) error = %v, want ErrSessionNotFound", "missing", err)
	}

	record, err := manager.lookup(session.SessionID)
	if err != nil {
		t.Fatalf("lookup(%q) error = %v", session.SessionID, err)
	}
	opts := runtimeOptionsForSession(record, contract.ExecutionPolicy{WriteMode: contract.WriteModeReadOnly})
	if opts.Policy.WriteMode != contract.WriteModeReadOnly {
		t.Errorf("runtimeOptionsForSession(...).Policy = %#v, want read-only policy", opts.Policy)
	}
	if opts.WorkingDir != record.snapshot.State.WorkingDir {
		t.Errorf("runtimeOptionsForSession(...).WorkingDir = %q, want %q", opts.WorkingDir, record.snapshot.State.WorkingDir)
	}
	if !reflect.DeepEqual(opts.CommandAliases, record.snapshot.State.CommandAliases) {
		t.Errorf("runtimeOptionsForSession(...).CommandAliases = %#v, want %#v", opts.CommandAliases, record.snapshot.State.CommandAliases)
	}
	if !reflect.DeepEqual(opts.EnvVars, record.snapshot.State.EnvVars) {
		t.Errorf("runtimeOptionsForSession(...).EnvVars = %#v, want %#v", opts.EnvVars, record.snapshot.State.EnvVars)
	}
	if opts.RCFiles != nil {
		t.Errorf("runtimeOptionsForSession(...).RCFiles = %#v, want nil because rc bootstrap should not replay", opts.RCFiles)
	}
	if !reflect.DeepEqual(opts.PathEnv, record.base.PathEnv) {
		t.Errorf("runtimeOptionsForSession(...).PathEnv = %#v, want %#v", opts.PathEnv, record.base.PathEnv)
	}
}

func TestSessionManagerDefaultFactoriesAndNilCreate(t *testing.T) {
	var nilManager *SessionManager
	if _, err := nilManager.Create(context.Background(), Options{}); err == nil {
		t.Fatal("(*SessionManager)(nil).Create(...) unexpectedly succeeded")
	}

	manager := NewSessionManager(SessionManagerOptions{})
	session, err := manager.Create(context.Background(), Options{HostRoot: t.TempDir(), Policy: contract.DefaultPolicy()})
	if err != nil {
		t.Fatalf("Create(...) with default factories error = %v", err)
	}
	if !strings.HasPrefix(session.SessionID, "sess_") {
		t.Fatalf("Create(...).SessionID = %q, want sess_* default id", session.SessionID)
	}
	if session.CreatedAt.IsZero() || session.UpdatedAt.IsZero() {
		t.Fatalf("Create(...) timestamps = (%s, %s), want non-zero", session.CreatedAt, session.UpdatedAt)
	}
}

func TestApplySessionAdaptersNoopWithoutAdapters(t *testing.T) {
	session := contract.Session{SessionID: "sess_noop"}
	gotSession, mounts, err := applySessionAdapters(context.Background(), session, nil, adapterPhaseCreate, contract.ExecutionResult{})
	if err != nil {
		t.Fatalf("applySessionAdapters(nil adapters) error = %v", err)
	}
	if gotSession.SessionID != session.SessionID {
		t.Fatalf("applySessionAdapters(nil adapters) session = %#v, want %#v", gotSession, session)
	}
	if len(mounts) != 0 {
		t.Fatalf("applySessionAdapters(nil adapters) mounts = %#v, want empty", mounts)
	}
}

func TestApplySessionAdaptersRejectsBlankAdapterID(t *testing.T) {
	session := contract.Session{SessionID: "sess_blank_adapter"}
	gotSession, mounts, err := applySessionAdapters(context.Background(), session, []contract.SessionAdapter{nil, &blankIDSessionAdapter{}}, adapterPhaseCreate, contract.ExecutionResult{})
	if err == nil || !strings.Contains(err.Error(), "session adapter id is required") {
		t.Fatalf("applySessionAdapters(blank id) error = %v, want blank adapter id error", err)
	}
	if gotSession.SessionID != session.SessionID {
		t.Fatalf("applySessionAdapters(blank id) session = %#v, want %#v", gotSession, session)
	}
	if len(mounts) != 0 {
		t.Fatalf("applySessionAdapters(blank id) mounts = %#v, want empty", mounts)
	}
}

func TestInvokeAdapterPhaseRejectsUnsupportedPhase(t *testing.T) {
	projection, err := invokeAdapterPhase(context.Background(), &testMemoryAdapter{}, adapterPhase("explode"), contract.Session{SessionID: "sess_phase"}, contract.ExecutionResult{})
	if err == nil || !strings.Contains(err.Error(), `unsupported adapter phase "explode"`) {
		t.Fatalf("invokeAdapterPhase(unsupported) error = %v, want unsupported phase error", err)
	}
	if !reflect.DeepEqual(projection, contract.AdapterProjection{}) {
		t.Fatalf("invokeAdapterPhase(unsupported) = %#v, want empty projection", projection)
	}
}

type blankIDSessionAdapter struct{ testMemoryAdapter }

func (a *blankIDSessionAdapter) AdapterID() string { return "   " }
