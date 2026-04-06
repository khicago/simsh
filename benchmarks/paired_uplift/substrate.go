package main

import (
	"context"
	"fmt"

	"github.com/khicago/simsh/pkg/builtin"
	"github.com/khicago/simsh/pkg/contract"
	"github.com/khicago/simsh/pkg/engine"
	runtimeengine "github.com/khicago/simsh/pkg/engine/runtime"
	"github.com/khicago/simsh/pkg/fs"
)

var thinCoreCommandNames = []string{
	builtin.CommandCd,
	builtin.CommandPwd,
	builtin.CommandCat,
	builtin.CommandEcho,
	builtin.CommandGrep,
	builtin.CommandMkdir,
	builtin.CommandSed,
}

type commandSubstrate interface {
	ID() string
	Run(context.Context, string) (contract.ExecutionResult, error)
	Close(context.Context) error
}

type simshSessionedSubstrate struct {
	manager   *runtimeengine.SessionManager
	sessionID string
}

func newSimshSessionedSubstrate(ctx context.Context, hostRoot string) (commandSubstrate, error) {
	manager := runtimeengine.NewSessionManager(runtimeengine.SessionManagerOptions{})
	session, err := manager.Create(ctx, runtimeengine.Options{
		HostRoot: hostRoot,
		Profile:  contract.ProfileBashPlus,
		Policy:   fullWritePolicy(),
	})
	if err != nil {
		return nil, err
	}
	return &simshSessionedSubstrate{manager: manager, sessionID: session.SessionID}, nil
}

func (s *simshSessionedSubstrate) ID() string {
	return substrateSimshFullSessioned
}

func (s *simshSessionedSubstrate) Run(ctx context.Context, command string) (contract.ExecutionResult, error) {
	if s == nil || s.manager == nil {
		return contract.ExecutionResult{}, fmt.Errorf("%s: substrate is not initialized", substrateSimshFullSessioned)
	}
	result, err := s.manager.Execute(ctx, s.sessionID, command, contract.ExecutionPolicy{})
	if err != nil {
		return contract.ExecutionResult{}, err
	}
	return result.Result, nil
}

func (s *simshSessionedSubstrate) Close(ctx context.Context) error {
	if s == nil || s.manager == nil {
		return nil
	}
	_, err := s.manager.Close(ctx, s.sessionID)
	return err
}

type thinCoreStatelessSubstrate struct {
	core     *engine.Engine
	prepared engine.PreparedOps
}

func newThinCoreStatelessSubstrate(ctx context.Context, hostRoot string) (commandSubstrate, error) {
	registry := engine.NewRegistry()
	if err := builtin.RegisterDefaultSubset(registry, thinCoreCommandNames); err != nil {
		return nil, err
	}
	core := engine.New(registry)
	ops, err := fs.NewRuntimeOps(fs.EnvironmentOptions{
		HostRoot: hostRoot,
		Profile:  contract.ProfileBashPlus,
		Policy:   fullWritePolicy(),
	})
	if err != nil {
		return nil, err
	}
	prepared, err := core.PrepareOps(ctx, ops)
	if err != nil {
		return nil, err
	}
	return &thinCoreStatelessSubstrate{core: core, prepared: prepared}, nil
}

func (s *thinCoreStatelessSubstrate) ID() string {
	return substrateThinCoreStateless
}

func (s *thinCoreStatelessSubstrate) Run(ctx context.Context, command string) (contract.ExecutionResult, error) {
	if s == nil || s.core == nil {
		return contract.ExecutionResult{}, fmt.Errorf("%s: substrate is not initialized", substrateThinCoreStateless)
	}
	s.prepared.ResetMutableState()
	return s.core.ExecutePreparedResult(ctx, command, s.prepared), nil
}

func (s *thinCoreStatelessSubstrate) Close(context.Context) error {
	return nil
}

func newSubstrate(ctx context.Context, id string, hostRoot string) (commandSubstrate, error) {
	switch id {
	case substrateSimshFullSessioned:
		return newSimshSessionedSubstrate(ctx, hostRoot)
	case substrateThinCoreStateless:
		return newThinCoreStatelessSubstrate(ctx, hostRoot)
	default:
		return nil, fmt.Errorf("unsupported substrate %q", id)
	}
}

func fullWritePolicy() contract.ExecutionPolicy {
	policy, err := contract.PolicyPreset(string(contract.WriteModeFull))
	if err != nil {
		panic(err)
	}
	return policy
}

func supportedScenarioIDs() []string {
	return []string{
		"relative_navigation_session",
		"inspect_edit_write_loop",
		"trace_consumable_planning",
	}
}

func scenarioIsSupported(id string) bool {
	for _, supported := range supportedScenarioIDs() {
		if supported == id {
			return true
		}
	}
	return false
}
