package contracttest

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/khicago/simsh/pkg/contract"
)

type Phase string

const (
	PhaseCreated      Phase = "created"
	PhaseObserved     Phase = "observed"
	PhaseCheckpointed Phase = "checkpointed"
	PhaseResumed      Phase = "resumed"
	PhaseClosed       Phase = "closed"
)

type Snapshot struct {
	Phase      Phase
	AdapterID  string
	Projection contract.AdapterProjection
	Session    contract.Session
}

type LifecycleSpec struct {
	Adapter            contract.SessionAdapter
	ObserveResult      contract.ExecutionResult
	RequireMemoryMount bool
	RequireOpaqueState bool
	WantMountPoints    []string

	CheckCreated      func(t *testing.T, snapshot Snapshot)
	CheckObserved     func(t *testing.T, snapshot Snapshot)
	CheckCheckpointed func(t *testing.T, snapshot Snapshot)
	CheckResumed      func(t *testing.T, snapshot Snapshot)
	CheckClosed       func(t *testing.T, snapshot Snapshot)
}

func RunLifecycle(t *testing.T, spec LifecycleSpec) {
	t.Helper()

	if err := runLifecycle(spec, func(phase Phase, snapshot Snapshot) error {
		switch phase {
		case PhaseCreated:
			if spec.CheckCreated != nil {
				spec.CheckCreated(t, snapshot)
			}
		case PhaseObserved:
			if spec.CheckObserved != nil {
				spec.CheckObserved(t, snapshot)
			}
		case PhaseCheckpointed:
			if spec.CheckCheckpointed != nil {
				spec.CheckCheckpointed(t, snapshot)
			}
		case PhaseResumed:
			if spec.CheckResumed != nil {
				spec.CheckResumed(t, snapshot)
			}
		case PhaseClosed:
			if spec.CheckClosed != nil {
				spec.CheckClosed(t, snapshot)
			}
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
}

func runLifecycle(spec LifecycleSpec, handlePhase func(Phase, Snapshot) error) error {
	adapterID, err := adapterIDFromSpec(spec)
	if err != nil {
		return err
	}

	ctx := context.Background()
	createdProjection, err := spec.Adapter.CreateSession(ctx, contract.Session{})
	if err != nil {
		return fmt.Errorf("CreateSession(...) error = %w, want nil", err)
	}
	created, err := newSnapshot(PhaseCreated, adapterID, createdProjection, spec)
	if err != nil {
		return err
	}
	if err := runPhaseHandler(handlePhase, PhaseCreated, created); err != nil {
		return err
	}

	observedProjection, err := spec.Adapter.ObserveExecution(ctx, created.Session, spec.ObserveResult)
	if err != nil {
		return fmt.Errorf("ObserveExecution(...) error = %w, want nil", err)
	}
	observed, err := newSnapshot(PhaseObserved, adapterID, observedProjection, spec)
	if err != nil {
		return err
	}
	if err := runPhaseHandler(handlePhase, PhaseObserved, observed); err != nil {
		return err
	}

	checkpointedProjection, err := spec.Adapter.CheckpointSession(ctx, observed.Session)
	if err != nil {
		return fmt.Errorf("CheckpointSession(...) error = %w, want nil", err)
	}
	checkpointed, err := newSnapshot(PhaseCheckpointed, adapterID, checkpointedProjection, spec)
	if err != nil {
		return err
	}
	if err := runPhaseHandler(handlePhase, PhaseCheckpointed, checkpointed); err != nil {
		return err
	}

	resumedProjection, err := spec.Adapter.ResumeSession(ctx, checkpointed.Session)
	if err != nil {
		return fmt.Errorf("ResumeSession(...) error = %w, want nil", err)
	}
	resumed, err := newSnapshot(PhaseResumed, adapterID, resumedProjection, spec)
	if err != nil {
		return err
	}
	if err := runPhaseHandler(handlePhase, PhaseResumed, resumed); err != nil {
		return err
	}

	closedProjection, err := spec.Adapter.CloseSession(ctx, resumed.Session)
	if err != nil {
		return fmt.Errorf("CloseSession(...) error = %w, want nil", err)
	}
	closed, err := newSnapshot(PhaseClosed, adapterID, closedProjection, spec)
	if err != nil {
		return err
	}
	return runPhaseHandler(handlePhase, PhaseClosed, closed)
}

func runPhaseHandler(handlePhase func(Phase, Snapshot) error, phase Phase, snapshot Snapshot) error {
	if handlePhase == nil {
		return nil
	}
	return handlePhase(phase, snapshot)
}

func adapterIDFromSpec(spec LifecycleSpec) (string, error) {
	if spec.Adapter == nil {
		return "", fmt.Errorf("RunLifecycle(...) adapter is nil")
	}
	adapterID := strings.TrimSpace(spec.Adapter.AdapterID())
	if adapterID == "" {
		return "", fmt.Errorf("RunLifecycle(...) AdapterID() returned empty string")
	}
	return adapterID, nil
}

func newSnapshot(phase Phase, adapterID string, projection contract.AdapterProjection, spec LifecycleSpec) (Snapshot, error) {
	if err := validateProjection(phase, projection, spec); err != nil {
		return Snapshot{}, err
	}
	return Snapshot{
		Phase:      phase,
		AdapterID:  adapterID,
		Projection: projection,
		Session:    sessionWithOpaque(adapterID, projection.OpaqueState),
	}, nil
}

func validateProjection(phase Phase, projection contract.AdapterProjection, spec LifecycleSpec) error {
	if spec.RequireMemoryMount && projection.Memory.Mount == nil {
		return fmt.Errorf("%s projection Memory.Mount = nil, want non-nil", phase)
	}
	if spec.RequireOpaqueState && len(projection.OpaqueState) == 0 {
		return fmt.Errorf("%s projection OpaqueState = empty, want non-empty", phase)
	}
	if len(spec.WantMountPoints) == 0 {
		return nil
	}
	gotMounts := map[string]struct{}{}
	for _, mount := range projection.ProjectionMounts() {
		gotMounts[mount.MountPoint()] = struct{}{}
	}
	for _, want := range spec.WantMountPoints {
		if _, ok := gotMounts[want]; !ok {
			return fmt.Errorf("%s projection mount points = %v, want %q present", phase, mountPoints(projection.ProjectionMounts()), want)
		}
	}
	return nil
}

func sessionWithOpaque(adapterID string, raw json.RawMessage) contract.Session {
	session := contract.Session{}
	if len(raw) == 0 {
		return session
	}
	session.State.Opaque = map[string]json.RawMessage{
		adapterID: append(json.RawMessage(nil), raw...),
	}
	return session
}

func mountPoints(mounts []contract.VirtualMount) []string {
	out := make([]string, 0, len(mounts))
	for _, mount := range mounts {
		out = append(out, mount.MountPoint())
	}
	return out
}

func (s Snapshot) ReadFile(t *testing.T, target string) string {
	t.Helper()
	raw, err := s.readFile(target)
	if err != nil {
		t.Fatalf("%s ReadRawContent(%q) error = %v, want nil", s.Phase, target, err)
	}
	return raw
}

func (s Snapshot) HasMountPoint(target string) bool {
	for _, mount := range s.Projection.ProjectionMounts() {
		if mount.MountPoint() == target {
			return true
		}
	}
	return false
}

func (s Snapshot) readFile(target string) (string, error) {
	mount, err := s.mountForPathErr(target)
	if err != nil {
		return "", err
	}
	raw, err := mount.ReadRawContent(context.Background(), target)
	if err != nil {
		return "", err
	}
	return raw, nil
}

func (s Snapshot) mountForPathErr(target string) (contract.VirtualMount, error) {
	var (
		best     contract.VirtualMount
		bestLen  = -1
		allMount = s.Projection.ProjectionMounts()
	)
	for _, mount := range allMount {
		prefix := mount.MountPoint()
		if !pathMatches(prefix, target) {
			continue
		}
		if len(prefix) > bestLen {
			best = mount
			bestLen = len(prefix)
		}
	}
	if best == nil {
		return nil, fmt.Errorf("%s projection has no mount for %q; mount points = %v", s.Phase, target, mountPoints(allMount))
	}
	return best, nil
}

func pathMatches(prefix string, target string) bool {
	if prefix == target {
		return true
	}
	if prefix == "/" {
		return true
	}
	return strings.HasPrefix(target, prefix+"/")
}
