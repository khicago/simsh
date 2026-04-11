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
	enginepkg "github.com/khicago/simsh/pkg/engine"
)

var (
	ErrSessionNotFound   = errors.New("runtime session not found")
	ErrSessionClosed     = errors.New("runtime session is closed")
	ErrSessionNotRunning = errors.New("runtime session has no active execution")
	ErrExecutionChanged  = errors.New("runtime session active execution changed")
	ErrExecutionRequired = errors.New("runtime session cancel requires active execution id")
)

type SessionManagerOptions struct {
	Now            func() time.Time
	NewID          func() string
	NewExecutionID func() string
}

type SessionExecution struct {
	Session contract.Session
	Runtime *Stack
	Result  contract.ExecutionResult
}

type SessionManager struct {
	mu             sync.RWMutex
	sessions       map[string]*managedSession
	now            func() time.Time
	newID          func() string
	newExecutionID func() string
}

type managedSession struct {
	snapshot        contract.Session
	checkpoint      contract.Session
	base            Options
	runtime         *Stack
	active          bool
	adapterMounts   []contract.VirtualMount
	executeMu       sync.Mutex
	activeExecution *activeExecutionControl
}

type activeExecutionControl struct {
	state  contract.SessionExecutionState
	cancel context.CancelFunc
	done   chan struct{}
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
	executionIDFn := opts.NewExecutionID
	if executionIDFn == nil {
		executionIDFn = enginepkg.NextExecutionID
	}
	return &SessionManager{
		sessions:       map[string]*managedSession{},
		now:            nowFn,
		newID:          idFn,
		newExecutionID: executionIDFn,
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
	sessionID = strings.TrimSpace(sessionID)
	if m == nil {
		return contract.Session{}, fmt.Errorf("session manager is not initialized")
	}
	if sessionID == "" {
		return contract.Session{}, ErrSessionNotFound
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	record, ok := m.sessions[sessionID]
	if !ok {
		return contract.Session{}, ErrSessionNotFound
	}
	if record.activeExecution != nil && executionFinished(record.activeExecution) {
		record.activeExecution = nil
		record.snapshot.ActiveExecution = nil
	}
	return record.snapshot.Clone(), nil
}

func (m *SessionManager) Cancel(sessionID string, expectedExecutionID string) (contract.Session, error) {
	if m == nil {
		return contract.Session{}, fmt.Errorf("session manager is not initialized")
	}

	var cancel context.CancelFunc

	m.mu.Lock()
	sessionID = strings.TrimSpace(sessionID)
	current, ok := m.sessions[sessionID]
	if !ok {
		m.mu.Unlock()
		return contract.Session{}, ErrSessionNotFound
	}
	if !current.active || current.runtime == nil {
		m.mu.Unlock()
		return contract.Session{}, ErrSessionClosed
	}
	if current.activeExecution == nil || current.snapshot.ActiveExecution == nil {
		m.mu.Unlock()
		return contract.Session{}, ErrSessionNotRunning
	}
	if executionFinished(current.activeExecution) {
		current.activeExecution = nil
		current.snapshot.ActiveExecution = nil
		m.mu.Unlock()
		return contract.Session{}, ErrSessionNotRunning
	}
	expectedExecutionID = strings.TrimSpace(expectedExecutionID)
	if expectedExecutionID == "" {
		m.mu.Unlock()
		return contract.Session{}, ErrExecutionRequired
	}
	if current.snapshot.ActiveExecution.ExecutionID != expectedExecutionID {
		m.mu.Unlock()
		return contract.Session{}, ErrExecutionChanged
	}

	updatedAt := m.now()
	activeExecution := cloneActiveExecution(current.snapshot.ActiveExecution)
	if activeExecution.Status == contract.SessionExecutionStatusCanceling {
		snapshot := current.snapshot.Clone()
		m.mu.Unlock()
		return snapshot, nil
	}
	activeExecution.Status = contract.SessionExecutionStatusCanceling
	activeExecution.StatusUpdatedAt = updatedAt
	current.activeExecution.state = activeExecution
	current.snapshot.ActiveExecution = &activeExecution
	current.snapshot.UpdatedAt = updatedAt
	snapshot := current.snapshot.Clone()
	cancel = current.activeExecution.cancel
	m.mu.Unlock()

	cancel()
	return snapshot, nil
}

func (m *SessionManager) Execute(ctx context.Context, sessionID string, commandLine string, requested contract.ExecutionPolicy) (SessionExecution, error) {
	sessionID = strings.TrimSpace(sessionID)
	commandLine = strings.TrimSpace(commandLine)
	record, err := m.lookupManaged(sessionID)
	if err != nil {
		return SessionExecution{}, err
	}
	record.executeMu.Lock()
	defer record.executeMu.Unlock()
	if !record.active || record.runtime == nil {
		return SessionExecution{}, ErrSessionClosed
	}

	effectivePolicy, err := contract.EffectivePolicyWithinCeiling(requested, record.snapshot.PolicyCeiling)
	if err != nil {
		return SessionExecution{}, err
	}
	if commandLine == "" {
		session := record.snapshot.Clone()
		result := enginepkg.UsageResult("execute: command is required").WithSessionID(session.SessionID)
		return SessionExecution{
			Session: session,
			Runtime: record.runtime,
			Result:  result,
		}, nil
	}

	runtime := record.runtime
	if !samePolicy(effectivePolicy, record.snapshot.PolicyCeiling) {
		runtime, err = New(runtimeOptionsForSession(record, effectivePolicy))
		if err != nil {
			return SessionExecution{}, err
		}
	}

	execCtx, cancel := context.WithCancel(ctx)
	executionID := m.newExecutionID()
	startedAt := m.now()
	activeExecution := newActiveExecutionControl(executionID, commandLine, startedAt, cancel)

	m.mu.Lock()
	current, ok := m.sessions[record.snapshot.SessionID]
	if !ok {
		m.mu.Unlock()
		cancel()
		return SessionExecution{}, ErrSessionNotFound
	}
	current.activeExecution = activeExecution
	current.snapshot.ActiveExecution = cloneActiveExecutionPtr(&activeExecution.state)
	current.snapshot.UpdatedAt = startedAt
	m.mu.Unlock()

	result := runtime.ExecuteResult(execCtx, commandLine).WithSessionID(record.snapshot.SessionID)
	result.ExecutionID = executionID
	close(activeExecution.done)
	cancel()

	m.mu.Lock()
	defer m.mu.Unlock()
	current, ok = m.sessions[record.snapshot.SessionID]
	if !ok {
		return SessionExecution{}, ErrSessionNotFound
	}
	current.activeExecution = nil
	current.snapshot.ActiveExecution = nil
	nextSession := current.snapshot.Clone()
	nextSession.UpdatedAt = m.now()
	nextSession.ActiveExecution = nil
	nextSession.State = mergeSessionState(nextSession.State, runtime)
	if len(current.base.Adapters) > 0 {
		var adapterErr error
		var adapterMounts []contract.VirtualMount
		nextSession, adapterMounts, adapterErr = applySessionAdapters(ctx, nextSession, current.base.Adapters, adapterPhaseObserve, result)
		if adapterErr != nil {
			return SessionExecution{}, adapterErr
		}
		rebuiltRuntime, rebuildErr := New(runtimeOptionsFromSession(current.base, nextSession, nextSession.PolicyCeiling, adapterMounts))
		if rebuildErr != nil {
			return SessionExecution{}, rebuildErr
		}
		current.adapterMounts = adapterMounts
		current.runtime = rebuiltRuntime
	} else if runtime != current.runtime {
		rebuiltRuntime, rebuildErr := New(runtimeOptionsFromSession(current.base, nextSession, nextSession.PolicyCeiling, current.adapterMounts))
		if rebuildErr != nil {
			return SessionExecution{}, rebuildErr
		}
		current.runtime = rebuiltRuntime
	}
	current.snapshot = nextSession.Clone()
	return SessionExecution{
		Session: current.snapshot.Clone(),
		Runtime: runtime,
		Result:  result.WithSessionID(current.snapshot.SessionID),
	}, nil
}

func (m *SessionManager) Checkpoint(ctx context.Context, sessionID string) (contract.Session, error) {
	record, err := m.lookupManaged(strings.TrimSpace(sessionID))
	if err != nil {
		return contract.Session{}, err
	}
	record.executeMu.Lock()
	defer record.executeMu.Unlock()
	if !record.active || record.runtime == nil {
		return contract.Session{}, ErrSessionClosed
	}

	m.mu.Lock()
	defer m.mu.Unlock()
	nextSession := record.snapshot.Clone()
	nextSession.UpdatedAt = m.now()
	nextSession.State = mergeSessionState(nextSession.State, record.runtime)
	if len(record.base.Adapters) > 0 {
		var adapterMounts []contract.VirtualMount
		nextSession, adapterMounts, err = applySessionAdapters(ctx, nextSession, record.base.Adapters, adapterPhaseCheckpoint, contract.ExecutionResult{})
		if err != nil {
			return contract.Session{}, err
		}
		record.adapterMounts = adapterMounts
	}
	record.snapshot = nextSession.Clone()
	record.checkpoint = record.snapshot.Clone()
	return record.checkpoint.Clone(), nil
}

func (m *SessionManager) Resume(ctx context.Context, sessionID string) (contract.Session, error) {
	record, err := m.lookupManaged(strings.TrimSpace(sessionID))
	if err != nil {
		return contract.Session{}, err
	}
	record.executeMu.Lock()
	defer record.executeMu.Unlock()
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
	record.runtime = runtime
	record.active = true
	record.snapshot = resumed.Clone()
	record.adapterMounts = cloneMounts(adapterMounts)
	record.snapshot.UpdatedAt = m.now()
	return record.snapshot.Clone(), nil
}

func (m *SessionManager) Close(ctx context.Context, sessionID string) (contract.Session, error) {
	record, err := m.lookupManaged(strings.TrimSpace(sessionID))
	if err != nil {
		return contract.Session{}, err
	}
	record.executeMu.Lock()
	defer record.executeMu.Unlock()
	if !record.active || record.runtime == nil {
		return contract.Session{}, ErrSessionClosed
	}

	m.mu.Lock()
	defer m.mu.Unlock()
	nextSession := record.snapshot.Clone()
	nextSession.UpdatedAt = m.now()
	nextSession.State = mergeSessionState(nextSession.State, record.runtime)
	if len(record.base.Adapters) > 0 {
		var adapterMounts []contract.VirtualMount
		nextSession, adapterMounts, err = applySessionAdapters(ctx, nextSession, record.base.Adapters, adapterPhaseClose, contract.ExecutionResult{})
		if err != nil {
			return contract.Session{}, err
		}
		record.adapterMounts = adapterMounts
	}
	record.snapshot = nextSession.Clone()
	record.checkpoint = record.snapshot.Clone()
	record.runtime = nil
	record.active = false
	record.activeExecution = nil
	record.snapshot.ActiveExecution = nil
	return record.snapshot.Clone(), nil
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

func (m *SessionManager) lookupManaged(sessionID string) (*managedSession, error) {
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
	return record, nil
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

func newActiveExecutionState(executionID string, commandLine string, now time.Time, status contract.SessionExecutionStatus) *contract.SessionExecutionState {
	return &contract.SessionExecutionState{
		ExecutionID:     strings.TrimSpace(executionID),
		CommandLine:     strings.TrimSpace(commandLine),
		StartedAt:       now,
		Status:          status,
		StatusUpdatedAt: now,
	}
}

func cloneActiveExecution(active *contract.SessionExecutionState) contract.SessionExecutionState {
	if active == nil {
		return contract.SessionExecutionState{}
	}
	return *active
}

func cloneActiveExecutionPtr(active *contract.SessionExecutionState) *contract.SessionExecutionState {
	if active == nil {
		return nil
	}
	cloned := *active
	return &cloned
}

func newActiveExecutionControl(executionID string, commandLine string, startedAt time.Time, cancel context.CancelFunc) *activeExecutionControl {
	return &activeExecutionControl{
		state:  *newActiveExecutionState(executionID, commandLine, startedAt, contract.SessionExecutionStatusRunning),
		cancel: cancel,
		done:   make(chan struct{}),
	}
}

func executionFinished(active *activeExecutionControl) bool {
	if active == nil {
		return true
	}
	select {
	case <-active.done:
		return true
	default:
		return false
	}
}
