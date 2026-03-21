package runtime

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/khicago/simsh/pkg/contract"
)

var (
	ErrSessionNotFound = errors.New("runtime session not found")
	ErrSessionClosed   = errors.New("runtime session is closed")
)

type SessionManagerOptions struct {
	Now   func() time.Time
	NewID func() string
}

type SessionExecution struct {
	Session contract.Session
	Runtime *Stack
	Result  contract.ExecutionResult
}

type SessionManager struct {
	mu       sync.RWMutex
	sessions map[string]*managedSession
	now      func() time.Time
	newID    func() string
}

type managedSession struct {
	snapshot      contract.Session
	checkpoint    contract.Session
	base          Options
	runtime       *Stack
	active        bool
	adapterMounts []contract.VirtualMount
}

var sessionCounter uint64

func NewSessionManager(opts SessionManagerOptions) *SessionManager {
	nowFn := opts.Now
	if nowFn == nil {
		nowFn = func() time.Time { return time.Now().UTC() }
	}
	idFn := opts.NewID
	if idFn == nil {
		idFn = func() string {
			value := atomic.AddUint64(&sessionCounter, 1)
			return fmt.Sprintf("sess_%d", value)
		}
	}
	return &SessionManager{
		sessions: map[string]*managedSession{},
		now:      nowFn,
		newID:    idFn,
	}
}

func (m *SessionManager) Create(ctx context.Context, opts Options) (contract.Session, error) {
	if m == nil {
		return contract.Session{}, fmt.Errorf("session manager is not initialized")
	}
	runtime, err := New(opts)
	if err != nil {
		return contract.Session{}, err
	}
	now := m.now()
	session := contract.Session{
		SessionID:     m.newID(),
		CreatedAt:     now,
		UpdatedAt:     now,
		Profile:       runtime.Ops().Profile,
		PolicyCeiling: runtime.Ops().Policy.Clone(),
		State:         runtime.SessionState(opts.RCFiles),
	}
	session, adapterMounts, err := applySessionAdapters(ctx, session, opts.Adapters, adapterPhaseCreate, contract.ExecutionResult{})
	if err != nil {
		return contract.Session{}, err
	}
	if len(adapterMounts) > 0 {
		runtime, err = New(runtimeOptionsFromSession(cloneOptions(opts), session, session.PolicyCeiling, adapterMounts))
		if err != nil {
			return contract.Session{}, err
		}
	}
	record := &managedSession{
		snapshot:      session.Clone(),
		checkpoint:    session.Clone(),
		base:          cloneOptions(opts),
		runtime:       runtime,
		active:        true,
		adapterMounts: cloneMounts(adapterMounts),
	}
	record.base.Profile = session.Profile
	record.base.Policy = session.PolicyCeiling.Clone()
	record.base.CommandAliases = session.State.Clone().CommandAliases
	record.base.EnvVars = session.State.Clone().EnvVars
	record.base.RCFiles = nil

	m.mu.Lock()
	defer m.mu.Unlock()
	m.sessions[session.SessionID] = record
	return session.Clone(), nil
}

func (m *SessionManager) Get(sessionID string) (contract.Session, error) {
	record, err := m.lookup(strings.TrimSpace(sessionID))
	if err != nil {
		return contract.Session{}, err
	}
	return record.snapshot.Clone(), nil
}

func (m *SessionManager) Execute(ctx context.Context, sessionID string, commandLine string, requested contract.ExecutionPolicy) (SessionExecution, error) {
	commandLine = strings.TrimSpace(commandLine)
	if commandLine == "" {
		return SessionExecution{
			Result: contract.ExecutionResult{
				ExitCode: contract.ExitCodeUsage,
				Stdout:   "execute: command is required",
			},
		}, nil
	}

	record, err := m.lookup(strings.TrimSpace(sessionID))
	if err != nil {
		return SessionExecution{}, err
	}
	if !record.active || record.runtime == nil {
		return SessionExecution{}, ErrSessionClosed
	}

	effectivePolicy, err := contract.EffectivePolicyWithinCeiling(requested, record.snapshot.PolicyCeiling)
	if err != nil {
		return SessionExecution{}, err
	}

	runtime := record.runtime
	if !samePolicy(effectivePolicy, record.snapshot.PolicyCeiling) {
		runtime, err = New(runtimeOptionsForSession(record, effectivePolicy))
		if err != nil {
			return SessionExecution{}, err
		}
	}

	result := runtime.ExecuteResult(ctx, commandLine).WithSessionID(record.snapshot.SessionID)

	m.mu.Lock()
	defer m.mu.Unlock()
	current, ok := m.sessions[record.snapshot.SessionID]
	if !ok {
		return SessionExecution{}, ErrSessionNotFound
	}
	nextSession := current.snapshot.Clone()
	nextSession.UpdatedAt = m.now()
	nextSession.State = mergeSessionState(nextSession.State, runtime)
	if len(current.base.Adapters) > 0 {
		var adapterErr error
		nextSession, current.adapterMounts, adapterErr = applySessionAdapters(ctx, nextSession, current.base.Adapters, adapterPhaseObserve, result)
		if adapterErr != nil {
			return SessionExecution{}, adapterErr
		}
		current.runtime, err = New(runtimeOptionsFromSession(current.base, nextSession, nextSession.PolicyCeiling, current.adapterMounts))
		if err != nil {
			return SessionExecution{}, err
		}
	} else if runtime != current.runtime {
		current.runtime, err = New(runtimeOptionsFromSession(current.base, nextSession, nextSession.PolicyCeiling, current.adapterMounts))
		if err != nil {
			return SessionExecution{}, err
		}
	}
	current.snapshot = nextSession.Clone()
	return SessionExecution{
		Session: current.snapshot.Clone(),
		Runtime: runtime,
		Result:  result.WithSessionID(current.snapshot.SessionID),
	}, nil
}

func (m *SessionManager) Checkpoint(ctx context.Context, sessionID string) (contract.Session, error) {
	record, err := m.lookup(strings.TrimSpace(sessionID))
	if err != nil {
		return contract.Session{}, err
	}

	m.mu.Lock()
	defer m.mu.Unlock()
	current := m.sessions[record.snapshot.SessionID]
	nextSession := current.snapshot.Clone()
	nextSession.UpdatedAt = m.now()
	nextSession.State = mergeSessionState(nextSession.State, current.runtime)
	if len(current.base.Adapters) > 0 {
		nextSession, current.adapterMounts, err = applySessionAdapters(ctx, nextSession, current.base.Adapters, adapterPhaseCheckpoint, contract.ExecutionResult{})
		if err != nil {
			return contract.Session{}, err
		}
	}
	current.snapshot = nextSession.Clone()
	current.checkpoint = current.snapshot.Clone()
	return current.checkpoint.Clone(), nil
}

func (m *SessionManager) Resume(ctx context.Context, sessionID string) (contract.Session, error) {
	record, err := m.lookup(strings.TrimSpace(sessionID))
	if err != nil {
		return contract.Session{}, err
	}
	if record.active && record.runtime != nil {
		return record.snapshot.Clone(), nil
	}
	resumed := record.checkpoint.Clone()
	resumed, adapterMounts, err := applySessionAdapters(ctx, resumed, record.base.Adapters, adapterPhaseResume, contract.ExecutionResult{})
	if err != nil {
		return contract.Session{}, err
	}
	runtime, err := New(runtimeOptionsFromSession(record.base, resumed, resumed.PolicyCeiling, adapterMounts))
	if err != nil {
		return contract.Session{}, err
	}

	m.mu.Lock()
	defer m.mu.Unlock()
	current := m.sessions[record.snapshot.SessionID]
	current.runtime = runtime
	current.active = true
	current.snapshot = resumed.Clone()
	current.adapterMounts = cloneMounts(adapterMounts)
	current.snapshot.UpdatedAt = m.now()
	return current.snapshot.Clone(), nil
}

func (m *SessionManager) Close(ctx context.Context, sessionID string) (contract.Session, error) {
	record, err := m.lookup(strings.TrimSpace(sessionID))
	if err != nil {
		return contract.Session{}, err
	}

	m.mu.Lock()
	defer m.mu.Unlock()
	current := m.sessions[record.snapshot.SessionID]
	nextSession := current.snapshot.Clone()
	nextSession.UpdatedAt = m.now()
	nextSession.State = mergeSessionState(nextSession.State, current.runtime)
	if len(current.base.Adapters) > 0 {
		nextSession, current.adapterMounts, err = applySessionAdapters(ctx, nextSession, current.base.Adapters, adapterPhaseClose, contract.ExecutionResult{})
		if err != nil {
			return contract.Session{}, err
		}
	}
	current.snapshot = nextSession.Clone()
	current.checkpoint = current.snapshot.Clone()
	current.runtime = nil
	current.active = false
	return current.snapshot.Clone(), nil
}

func (m *SessionManager) lookup(sessionID string) (*managedSession, error) {
	if m == nil {
		return nil, fmt.Errorf("session manager is not initialized")
	}
	if sessionID == "" {
		return nil, ErrSessionNotFound
	}
	m.mu.RLock()
	defer m.mu.RUnlock()
	record, ok := m.sessions[sessionID]
	if !ok {
		return nil, ErrSessionNotFound
	}
	clone := *record
	clone.snapshot = record.snapshot.Clone()
	clone.checkpoint = record.checkpoint.Clone()
	clone.base = cloneOptions(record.base)
	clone.adapterMounts = cloneMounts(record.adapterMounts)
	return &clone, nil
}

func runtimeOptionsForSession(record *managedSession, policy contract.ExecutionPolicy) Options {
	return runtimeOptionsFromSession(record.base, record.snapshot, policy, record.adapterMounts)
}

func runtimeOptionsFromSession(base Options, session contract.Session, policy contract.ExecutionPolicy, adapterMounts []contract.VirtualMount) Options {
	opts := cloneOptions(base)
	opts.Profile = session.Profile
	opts.Policy = policy.Clone()
	opts.WorkingDir = strings.TrimSpace(session.State.WorkingDir)
	opts.CommandAliases = session.State.Clone().CommandAliases
	opts.EnvVars = session.State.Clone().EnvVars
	opts.RCFiles = nil
	opts.VirtualMounts = append(opts.VirtualMounts, adapterMounts...)
	return opts
}

func mergeSessionState(previous contract.SessionState, runtime *Stack) contract.SessionState {
	state := previous.Clone()
	if runtime == nil {
		return state
	}
	runtimeState := runtime.SessionState(state.RCFiles)
	state.CommandAliases = runtimeState.CommandAliases
	state.EnvVars = runtimeState.EnvVars
	state.WorkingDir = runtimeState.WorkingDir
	return state
}

func cloneOptions(opts Options) Options {
	return Options{
		HostRoot:          strings.TrimSpace(opts.HostRoot),
		WorkingDir:        strings.TrimSpace(opts.WorkingDir),
		Profile:           opts.Profile,
		Policy:            opts.Policy.Clone(),
		CommandAliases:    contract.NormalizeCommandAliases(opts.CommandAliases),
		EnvVars:           contract.NormalizeEnvVars(opts.EnvVars),
		RCFiles:           contract.NormalizeRCFiles(opts.RCFiles),
		VirtualMounts:     cloneMounts(opts.VirtualMounts),
		Adapters:          append([]contract.SessionAdapter(nil), opts.Adapters...),
		EnableTestCorpus:  opts.EnableTestCorpus,
		PathEnv:           append([]string(nil), opts.PathEnv...),
		ExternalCallbacks: opts.ExternalCallbacks,
		FormatLSLongRow:   opts.FormatLSLongRow,
		AuditSink:         opts.AuditSink,
	}
}

func cloneMounts(mounts []contract.VirtualMount) []contract.VirtualMount {
	if len(mounts) == 0 {
		return nil
	}
	return append([]contract.VirtualMount(nil), mounts...)
}

func samePolicy(left contract.ExecutionPolicy, right contract.ExecutionPolicy) bool {
	return left.WriteMode == right.WriteMode &&
		left.MaxWriteBytes == right.MaxWriteBytes &&
		left.MaxPipelineDepth == right.MaxPipelineDepth &&
		left.MaxOutputBytes == right.MaxOutputBytes &&
		left.Timeout == right.Timeout
}
