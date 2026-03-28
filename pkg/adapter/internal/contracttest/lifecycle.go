package contracttest

import (
	"context"
	"encoding/json"
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

	if spec.Adapter == nil {
		t.Fatal("RunLifecycle(...) adapter is nil")
	}
	adapterID := strings.TrimSpace(spec.Adapter.AdapterID())
	if adapterID == "" {
		t.Fatal("RunLifecycle(...) AdapterID() returned empty string")
	}

	ctx := context.Background()
	createdProjection, err := spec.Adapter.CreateSession(ctx, contract.Session{})
	if err != nil {
		t.Fatalf("CreateSession(...) error = %v, want nil", err)
	}
	created := newSnapshot(t, PhaseCreated, adapterID, createdProjection, spec)
	runPhaseCheck(t, spec.CheckCreated, created)

	observedProjection, err := spec.Adapter.ObserveExecution(ctx, created.Session, spec.ObserveResult)
	if err != nil {
		t.Fatalf("ObserveExecution(...) error = %v, want nil", err)
	}
	observed := newSnapshot(t, PhaseObserved, adapterID, observedProjection, spec)
	runPhaseCheck(t, spec.CheckObserved, observed)

	checkpointedProjection, err := spec.Adapter.CheckpointSession(ctx, observed.Session)
	if err != nil {
		t.Fatalf("CheckpointSession(...) error = %v, want nil", err)
	}
	checkpointed := newSnapshot(t, PhaseCheckpointed, adapterID, checkpointedProjection, spec)
	runPhaseCheck(t, spec.CheckCheckpointed, checkpointed)

	resumedProjection, err := spec.Adapter.ResumeSession(ctx, checkpointed.Session)
	if err != nil {
		t.Fatalf("ResumeSession(...) error = %v, want nil", err)
	}
	resumed := newSnapshot(t, PhaseResumed, adapterID, resumedProjection, spec)
	runPhaseCheck(t, spec.CheckResumed, resumed)

	closedProjection, err := spec.Adapter.CloseSession(ctx, resumed.Session)
	if err != nil {
		t.Fatalf("CloseSession(...) error = %v, want nil", err)
	}
	closed := newSnapshot(t, PhaseClosed, adapterID, closedProjection, spec)
	runPhaseCheck(t, spec.CheckClosed, closed)
}

func runPhaseCheck(t *testing.T, fn func(t *testing.T, snapshot Snapshot), snapshot Snapshot) {
	t.Helper()
	if fn == nil {
		return
	}
	fn(t, snapshot)
}

func newSnapshot(t *testing.T, phase Phase, adapterID string, projection contract.AdapterProjection, spec LifecycleSpec) Snapshot {
	t.Helper()
	validateProjection(t, phase, projection, spec)
	return Snapshot{
		Phase:      phase,
		AdapterID:  adapterID,
		Projection: projection,
		Session:    sessionWithOpaque(adapterID, projection.OpaqueState),
	}
}

func validateProjection(t *testing.T, phase Phase, projection contract.AdapterProjection, spec LifecycleSpec) {
	t.Helper()
	if spec.RequireMemoryMount && projection.Memory.Mount == nil {
		t.Fatalf("%s projection Memory.Mount = nil, want non-nil", phase)
	}
	if spec.RequireOpaqueState && len(projection.OpaqueState) == 0 {
		t.Fatalf("%s projection OpaqueState = empty, want non-empty", phase)
	}
	if len(spec.WantMountPoints) == 0 {
		return
	}
	gotMounts := map[string]struct{}{}
	for _, mount := range projection.ProjectionMounts() {
		gotMounts[mount.MountPoint()] = struct{}{}
	}
	for _, want := range spec.WantMountPoints {
		if _, ok := gotMounts[want]; !ok {
			t.Fatalf("%s projection mount points = %v, want %q present", phase, mountPoints(projection.ProjectionMounts()), want)
		}
	}
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
	mount := s.mountForPath(t, target)
	raw, err := mount.ReadRawContent(context.Background(), target)
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

func (s Snapshot) mountForPath(t *testing.T, target string) contract.VirtualMount {
	t.Helper()
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
		t.Fatalf("%s projection has no mount for %q; mount points = %v", s.Phase, target, mountPoints(allMount))
	}
	return best
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
