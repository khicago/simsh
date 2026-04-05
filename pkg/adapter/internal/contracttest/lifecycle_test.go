package contracttest

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"path"
	"slices"
	"strings"
	"testing"

	"github.com/khicago/simsh/pkg/contract"
)

type fakeLifecycleAdapter struct {
	id string

	createProjection     contract.AdapterProjection
	observeProjection    contract.AdapterProjection
	checkpointProjection contract.AdapterProjection
	resumeProjection     contract.AdapterProjection
	closeProjection      contract.AdapterProjection

	createErr     error
	observeErr    error
	checkpointErr error
	resumeErr     error
	closeErr      error

	calls []string
}

func (a *fakeLifecycleAdapter) AdapterID() string { return a.id }

func (a *fakeLifecycleAdapter) CreateSession(ctx context.Context, session contract.Session) (contract.AdapterProjection, error) {
	_ = ctx
	_ = session
	a.calls = append(a.calls, "create")
	return a.createProjection, a.createErr
}

func (a *fakeLifecycleAdapter) ResumeSession(ctx context.Context, session contract.Session) (contract.AdapterProjection, error) {
	_ = ctx
	_ = session
	a.calls = append(a.calls, "resume")
	return a.resumeProjection, a.resumeErr
}

func (a *fakeLifecycleAdapter) ObserveExecution(ctx context.Context, session contract.Session, result contract.ExecutionResult) (contract.AdapterProjection, error) {
	_ = ctx
	_ = session
	_ = result
	a.calls = append(a.calls, "observe")
	return a.observeProjection, a.observeErr
}

func (a *fakeLifecycleAdapter) CheckpointSession(ctx context.Context, session contract.Session) (contract.AdapterProjection, error) {
	_ = ctx
	_ = session
	a.calls = append(a.calls, "checkpoint")
	return a.checkpointProjection, a.checkpointErr
}

func (a *fakeLifecycleAdapter) CloseSession(ctx context.Context, session contract.Session) (contract.AdapterProjection, error) {
	_ = ctx
	_ = session
	a.calls = append(a.calls, "close")
	return a.closeProjection, a.closeErr
}

func TestRunLifecycleSuccess(t *testing.T) {
	mount := newFakeMount("/memory", map[string]string{
		"/memory/status.json": `{"phase":"ok"}`,
	}, nil)
	adapter := &fakeLifecycleAdapter{
		id:                   "fake",
		createProjection:     fakeProjection(mount, `{"phase":"created"}`),
		observeProjection:    fakeProjection(mount, `{"phase":"observed"}`),
		checkpointProjection: fakeProjection(mount, `{"phase":"checkpointed"}`),
		resumeProjection:     fakeProjection(mount, `{"phase":"resumed"}`),
		closeProjection:      fakeProjection(mount, `{"phase":"closed"}`),
	}

	gotPhases := []Phase{}
	RunLifecycle(t, LifecycleSpec{
		Adapter:            adapter,
		ObserveResult:      contract.ExecutionResult{ExecutionID: "exec"},
		RequireMemoryMount: true,
		RequireOpaqueState: true,
		WantMountPoints:    []string{"/memory"},
		CheckCreated: func(t *testing.T, snapshot Snapshot) {
			gotPhases = append(gotPhases, snapshot.Phase)
			if !snapshot.HasMountPoint("/memory") {
				t.Fatalf("created snapshot HasMountPoint(%q) = false, want true", "/memory")
			}
			if got := snapshot.ReadFile(t, "/memory/status.json"); got != `{"phase":"ok"}` {
				t.Fatalf("created snapshot ReadFile(...) = %q, want %q", got, `{"phase":"ok"}`)
			}
		},
		CheckObserved: func(t *testing.T, snapshot Snapshot) {
			gotPhases = append(gotPhases, snapshot.Phase)
		},
		CheckCheckpointed: func(t *testing.T, snapshot Snapshot) {
			gotPhases = append(gotPhases, snapshot.Phase)
		},
		CheckResumed: func(t *testing.T, snapshot Snapshot) {
			gotPhases = append(gotPhases, snapshot.Phase)
		},
		CheckClosed: func(t *testing.T, snapshot Snapshot) {
			gotPhases = append(gotPhases, snapshot.Phase)
			if snapshot.Session.State.Opaque["fake"] == nil {
				t.Fatal("closed snapshot opaque state missing")
			}
		},
	})

	if diff := slices.Compare(adapter.calls, []string{"create", "observe", "checkpoint", "resume", "close"}); diff != 0 {
		t.Fatalf("RunLifecycle(...) calls = %v, want %v", adapter.calls, []string{"create", "observe", "checkpoint", "resume", "close"})
	}
	if diff := slices.Compare(gotPhases, []Phase{PhaseCreated, PhaseObserved, PhaseCheckpointed, PhaseResumed, PhaseClosed}); diff != 0 {
		t.Fatalf("RunLifecycle(...) callbacks = %v, want %v", gotPhases, []Phase{PhaseCreated, PhaseObserved, PhaseCheckpointed, PhaseResumed, PhaseClosed})
	}
}

func TestRunLifecycleFailures(t *testing.T) {
	baseProjection := fakeProjection(newFakeMount("/memory", map[string]string{
		"/memory/status.json": "{}",
	}, nil), `{"ok":true}`)

	testCases := []struct {
		name string
		spec LifecycleSpec
	}{
		{name: "nil_adapter", spec: LifecycleSpec{}},
		{name: "empty_adapter_id", spec: LifecycleSpec{Adapter: &fakeLifecycleAdapter{}}},
		{name: "create_error", spec: LifecycleSpec{Adapter: &fakeLifecycleAdapter{id: "fake", createErr: errors.New("create failed")}}},
		{name: "create_missing_memory", spec: LifecycleSpec{Adapter: &fakeLifecycleAdapter{id: "fake", createProjection: contract.AdapterProjection{OpaqueState: json.RawMessage(`{"ok":true}`)}}, RequireMemoryMount: true}},
		{name: "create_missing_opaque", spec: LifecycleSpec{Adapter: &fakeLifecycleAdapter{id: "fake", createProjection: contract.AdapterProjection{Memory: contract.MemoryProjection{Mount: newFakeMount("/memory", map[string]string{"/memory/status.json": "{}"}, nil)}}}, RequireMemoryMount: true, RequireOpaqueState: true}},
		{name: "create_missing_mount_point", spec: LifecycleSpec{Adapter: &fakeLifecycleAdapter{id: "fake", createProjection: baseProjection}, WantMountPoints: []string{"/skills"}}},
		{name: "observe_error", spec: LifecycleSpec{Adapter: &fakeLifecycleAdapter{id: "fake", createProjection: baseProjection, observeErr: errors.New("observe failed")}}},
		{name: "checkpoint_error", spec: LifecycleSpec{Adapter: &fakeLifecycleAdapter{id: "fake", createProjection: baseProjection, observeProjection: baseProjection, checkpointErr: errors.New("checkpoint failed")}}},
		{name: "resume_error", spec: LifecycleSpec{Adapter: &fakeLifecycleAdapter{id: "fake", createProjection: baseProjection, observeProjection: baseProjection, checkpointProjection: baseProjection, resumeErr: errors.New("resume failed")}}},
		{name: "close_error", spec: LifecycleSpec{Adapter: &fakeLifecycleAdapter{id: "fake", createProjection: baseProjection, observeProjection: baseProjection, checkpointProjection: baseProjection, resumeProjection: baseProjection, closeErr: errors.New("close failed")}}},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := runLifecycle(tc.spec, nil)
			if err == nil {
				t.Fatalf("runLifecycle(%s) error = nil, want non-nil", tc.name)
			}
		})
	}
}

func TestSessionWithOpaqueAndMountPoints(t *testing.T) {
	session := sessionWithOpaque("fake", json.RawMessage(`{"x":1}`))
	if got := string(session.State.Opaque["fake"]); got != `{"x":1}` {
		t.Fatalf("sessionWithOpaque(...) stored = %q, want %q", got, `{"x":1}`)
	}
	empty := sessionWithOpaque("fake", nil)
	if empty.State.Opaque != nil {
		t.Fatalf("sessionWithOpaque(nil) opaque = %v, want nil", empty.State.Opaque)
	}

	mounts := []contract.VirtualMount{
		newFakeMount("/skills", nil, nil),
		newFakeMount("/memory", nil, nil),
	}
	if got := mountPoints(mounts); !slices.Equal(got, []string{"/skills", "/memory"}) {
		t.Fatalf("mountPoints(...) = %v, want %v", got, []string{"/skills", "/memory"})
	}
}

func TestSnapshotMountLookupHelpers(t *testing.T) {
	parent := newFakeMount("/memory", map[string]string{"/memory/status.json": "root"}, nil)
	child := newFakeMount("/memory/views", map[string]string{"/memory/views/summary.md": "child"}, nil)
	snapshot := Snapshot{
		Phase: PhaseObserved,
		Projection: contract.AdapterProjection{
			VirtualMounts: []contract.VirtualMount{parent, child},
		},
	}

	if got := snapshot.ReadFile(t, "/memory/views/summary.md"); got != "child" {
		t.Fatalf("Snapshot.ReadFile(...) = %q, want %q", got, "child")
	}
	if got := snapshot.ReadFile(t, "/memory/status.json"); got != "root" {
		t.Fatalf("Snapshot.ReadFile(...) = %q, want %q", got, "root")
	}
	if snapshot.HasMountPoint("/skills") {
		t.Fatalf("Snapshot.HasMountPoint(%q) = true, want false", "/skills")
	}
	if !snapshot.HasMountPoint("/memory/views") {
		t.Fatalf("Snapshot.HasMountPoint(%q) = false, want true", "/memory/views")
	}

	if _, err := snapshot.mountForPathErr("/elsewhere/file.txt"); err == nil {
		t.Fatal("Snapshot.mountForPathErr(...) unexpectedly succeeded for missing path")
	}
	if _, err := snapshot.readFile("/memory/views/missing.md"); err == nil {
		t.Fatal("Snapshot.readFile(...) unexpectedly succeeded for missing file under existing mount")
	}
	if _, err := snapshot.readFile("/elsewhere/file.txt"); err == nil {
		t.Fatal("Snapshot.readFile(...) unexpectedly succeeded for missing path")
	}
}

func TestPathMatches(t *testing.T) {
	testCases := []struct {
		prefix string
		target string
		want   bool
	}{
		{prefix: "/memory", target: "/memory", want: true},
		{prefix: "/", target: "/anything", want: true},
		{prefix: "/memory", target: "/memory/status.json", want: true},
		{prefix: "/memory", target: "/memoryish/status.json", want: false},
	}

	for _, tc := range testCases {
		if got := pathMatches(tc.prefix, tc.target); got != tc.want {
			t.Fatalf("pathMatches(%q, %q) = %t, want %t", tc.prefix, tc.target, got, tc.want)
		}
	}
}

func TestRunPhaseHandlerAndAdapterIDFromSpec(t *testing.T) {
	phaseCalls := []Phase{}
	if err := runPhaseHandler(nil, PhaseCreated, Snapshot{}); err != nil {
		t.Fatalf("runPhaseHandler(nil, ...) error = %v, want nil", err)
	}
	if err := runPhaseHandler(func(phase Phase, snapshot Snapshot) error {
		_ = snapshot
		phaseCalls = append(phaseCalls, phase)
		return nil
	}, PhaseObserved, Snapshot{}); err != nil {
		t.Fatalf("runPhaseHandler(...) error = %v, want nil", err)
	}
	if !slices.Equal(phaseCalls, []Phase{PhaseObserved}) {
		t.Fatalf("runPhaseHandler(...) phases = %v, want %v", phaseCalls, []Phase{PhaseObserved})
	}
	if err := runPhaseHandler(func(phase Phase, snapshot Snapshot) error {
		_ = phase
		_ = snapshot
		return errors.New("stop")
	}, PhaseCreated, Snapshot{}); err == nil || !strings.Contains(err.Error(), "stop") {
		t.Fatalf("runPhaseHandler(error) = %v, want propagated error", err)
	}

	if _, err := adapterIDFromSpec(LifecycleSpec{}); err == nil || !strings.Contains(err.Error(), "adapter is nil") {
		t.Fatalf("adapterIDFromSpec(nil) error = %v, want adapter is nil", err)
	}
	if _, err := adapterIDFromSpec(LifecycleSpec{Adapter: &fakeLifecycleAdapter{id: "  "}}); err == nil || !strings.Contains(err.Error(), "returned empty string") {
		t.Fatalf("adapterIDFromSpec(empty) error = %v, want empty string", err)
	}
	id, err := adapterIDFromSpec(LifecycleSpec{Adapter: &fakeLifecycleAdapter{id: "adapter"}})
	if err != nil || id != "adapter" {
		t.Fatalf("adapterIDFromSpec(valid) = (%q, %v), want (%q, nil)", id, err, "adapter")
	}
}

func TestValidateProjectionAndNewSnapshot(t *testing.T) {
	memoryMount := newFakeMount("/memory", map[string]string{"/memory/status.json": "{}"}, nil)
	projection := fakeProjection(memoryMount, `{"ok":true}`)
	spec := LifecycleSpec{
		RequireMemoryMount: true,
		RequireOpaqueState: true,
		WantMountPoints:    []string{"/memory"},
	}
	if err := validateProjection(PhaseCreated, projection, spec); err != nil {
		t.Fatalf("validateProjection(...) error = %v, want nil", err)
	}
	snapshot, err := newSnapshot(PhaseCreated, "fake", projection, spec)
	if err != nil {
		t.Fatalf("newSnapshot(...) error = %v, want nil", err)
	}
	if snapshot.AdapterID != "fake" || snapshot.Phase != PhaseCreated {
		t.Fatalf("newSnapshot(...) = %+v, want adapterID=fake phase=created", snapshot)
	}
	if _, err := newSnapshot(PhaseCreated, "fake", contract.AdapterProjection{}, spec); err == nil {
		t.Fatal("newSnapshot(...) unexpectedly succeeded without required memory/opaque state")
	}
}

func TestRunLifecycleStopsOnHandlerError(t *testing.T) {
	mount := newFakeMount("/memory", map[string]string{"/memory/status.json": "{}"}, nil)
	projection := fakeProjection(mount, `{"ok":true}`)
	adapter := &fakeLifecycleAdapter{
		id:                   "fake",
		createProjection:     projection,
		observeProjection:    projection,
		checkpointProjection: projection,
		resumeProjection:     projection,
		closeProjection:      projection,
	}

	err := runLifecycle(LifecycleSpec{
		Adapter:            adapter,
		RequireMemoryMount: true,
		RequireOpaqueState: true,
	}, func(phase Phase, snapshot Snapshot) error {
		_ = snapshot
		if phase == PhaseObserved {
			return errors.New("handler failed")
		}
		return nil
	})
	if err == nil || !strings.Contains(err.Error(), "handler failed") {
		t.Fatalf("runLifecycle(handler error) = %v, want propagated handler failure", err)
	}
	if !slices.Equal(adapter.calls, []string{"create", "observe"}) {
		t.Fatalf("runLifecycle(handler error) calls = %v, want stop after observe", adapter.calls)
	}
}

func fakeProjection(memory contract.VirtualMount, opaque string) contract.AdapterProjection {
	return contract.AdapterProjection{
		Memory:      contract.MemoryProjection{Mount: memory},
		OpaqueState: json.RawMessage(opaque),
	}
}

type fakeMount struct {
	point string
	files map[string]string
	dirs  map[string]struct{}
}

func newFakeMount(point string, files map[string]string, dirs []string) *fakeMount {
	fm := &fakeMount{
		point: point,
		files: map[string]string{},
		dirs:  map[string]struct{}{point: {}},
	}
	for k, v := range files {
		fm.files[k] = v
	}
	for _, dir := range dirs {
		fm.dirs[dir] = struct{}{}
	}
	return fm
}

func (m *fakeMount) MountPoint() string { return m.point }
func (m *fakeMount) Profile() contract.MountProfile {
	return contract.NormalizeMountProfile(contract.MountProfile{
		TruthModel:          contract.MountTruthProjection,
		MaterializationMode: contract.MountMaterializationSnapshot,
		WriteSemantics:      contract.MountWriteReadOnly,
		LatencyClass:        contract.MountLatencyLocalFast,
		SupportedCLIClasses: []contract.MountCLIClass{contract.MountCLIRead},
	})
}
func (m *fakeMount) Exists(ctx context.Context) (bool, error) {
	_ = ctx
	return true, nil
}
func (m *fakeMount) StatPath(ctx context.Context, pathValue string) (contract.MountEntry, error) {
	_ = ctx
	if _, ok := m.dirs[pathValue]; ok {
		return contract.MountEntry{
			Path: pathValue,
			Name: path.Base(pathValue),
			Meta: contract.PathMeta{
				Exists:       true,
				IsDir:        true,
				Kind:         "memory_dir",
				Access:       contract.PathAccessReadOnly,
				Capabilities: []string{contract.PathCapabilityDescribe, contract.PathCapabilityList, contract.PathCapabilitySearch},
			},
		}, nil
	}
	raw, ok := m.files[pathValue]
	if !ok {
		return contract.MountEntry{}, fmt.Errorf("missing file: %s", pathValue)
	}
	return contract.MountEntry{
		Path: pathValue,
		Name: path.Base(pathValue),
		Meta: contract.PathMeta{
			Exists:       true,
			IsDir:        false,
			Kind:         "memory_file",
			Access:       contract.PathAccessReadOnly,
			Capabilities: []string{contract.PathCapabilityDescribe, contract.PathCapabilityRead},
			LineCount:    len(strings.Split(strings.TrimSuffix(raw, "\n"), "\n")),
		},
	}, nil
}
func (m *fakeMount) ReadContent(ctx context.Context, pathValue string) (string, error) {
	_ = ctx
	raw, ok := m.files[pathValue]
	if !ok {
		return "", fmt.Errorf("missing file: %s", pathValue)
	}
	return raw, nil
}
