package contract

import (
	"context"
	"encoding/json"
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestNormalizeCommandAliases(t *testing.T) {
	if got := DefaultCommandAliases(); !reflect.DeepEqual(got, map[string][]string{
		"ll": {"ls", "-l"},
		"fm": {"frontmatter"},
	}) {
		t.Fatalf("DefaultCommandAliases() = %#v, want ll/fm defaults", got)
	}

	got := NormalizeCommandAliases(map[string][]string{
		" ll ":       {" ls ", "", "-l"},
		" fm ":       {" frontmatter "},
		"-bad":       {"echo"},
		"path/name":  {"echo"},
		"two words":  {"echo"},
		"empty-only": {"", "   "},
	})

	want := map[string][]string{
		"ll": {"ls", "-l"},
		"fm": {"frontmatter"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("NormalizeCommandAliases(...) = %#v, want %#v", got, want)
	}
}

func TestMergeCommandAliases(t *testing.T) {
	got := MergeCommandAliases(
		map[string][]string{"ll": {"ls"}},
		map[string][]string{"ll": {"ls", "-l"}, "fm": {"frontmatter"}},
	)

	want := map[string][]string{
		"ll": {"ls", "-l"},
		"fm": {"frontmatter"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("MergeCommandAliases(...) = %#v, want %#v", got, want)
	}
}

func TestNormalizeEnvVarsAndRCFiles(t *testing.T) {
	env := NormalizeEnvVars(map[string]string{
		" PATH ":  "ignored",
		"FOO":     "x",
		"BAR_2":   "y",
		"9BAD":    "z",
		"bad-key": "skip",
	})
	wantEnv := map[string]string{
		"FOO":   "x",
		"BAR_2": "y",
		"PATH":  "ignored",
	}
	if !reflect.DeepEqual(env, wantEnv) {
		t.Errorf("NormalizeEnvVars(...) = %#v, want %#v", env, wantEnv)
	}

	rc := NormalizeRCFiles([]string{" /etc/a ", "/etc/a", "", "  ", "/etc/b"})
	wantRC := []string{"/etc/a", "/etc/b"}
	if !reflect.DeepEqual(rc, wantRC) {
		t.Errorf("NormalizeRCFiles(...) = %#v, want %#v", rc, wantRC)
	}
}

func TestExecutionPolicyHelpers(t *testing.T) {
	if got := DefaultPolicy(); got.WriteMode != WriteModeReadOnly || got.Timeout <= 0 {
		t.Errorf("DefaultPolicy() = %#v, want read-only policy with timeout", got)
	}

	preset, err := PolicyPreset(string(WriteModeWriteLimited))
	if err != nil {
		t.Fatalf("PolicyPreset(%q) error = %v", WriteModeWriteLimited, err)
	}
	if !preset.AllowWrite() {
		t.Errorf("PolicyPreset(%q).AllowWrite() = false, want true", WriteModeWriteLimited)
	}
	if _, err := PolicyPreset("broken"); err == nil {
		t.Fatalf("PolicyPreset(%q) unexpectedly succeeded", "broken")
	}
	if got, err := PolicyPreset(string(WriteModeDisabled)); err != nil || got.WriteMode != WriteModeDisabled {
		t.Errorf("PolicyPreset(%q) = (%#v, %v), want disabled preset", WriteModeDisabled, got, err)
	}

	inherited := ExecutionPolicy{WriteMode: WriteModeFull, Timeout: 30 * time.Second}
	requested := ExecutionPolicy{}
	got := requested.WithInheritedUnset(inherited)
	if got.WriteMode != WriteModeFull || got.Timeout != 30*time.Second {
		t.Errorf("ExecutionPolicy.WithInheritedUnset(...) = %#v, want inherited values", got)
	}

	ceiling := ExecutionPolicy{
		WriteMode:        WriteModeWriteLimited,
		MaxWriteBytes:    4,
		MaxPipelineDepth: 2,
		MaxOutputBytes:   64,
		Timeout:          2 * time.Second,
	}
	if err := PolicyWithinCeiling(
		ExecutionPolicy{
			WriteMode:        WriteModeFull,
			MaxWriteBytes:    4,
			MaxPipelineDepth: 2,
			MaxOutputBytes:   64,
			Timeout:          2 * time.Second,
		},
		ceiling,
	); !errors.Is(err, ErrPolicyCeilingExceeded) {
		t.Errorf("PolicyWithinCeiling(full, ceiling) error = %v, want ErrPolicyCeilingExceeded", err)
	}

	effective, err := EffectivePolicyWithinCeiling(
		ExecutionPolicy{WriteMode: WriteModeWriteLimited, MaxWriteBytes: 4},
		ceiling,
	)
	if err != nil {
		t.Fatalf("EffectivePolicyWithinCeiling(...) error = %v", err)
	}
	if effective.MaxPipelineDepth != ceiling.MaxPipelineDepth || effective.MaxOutputBytes != ceiling.MaxOutputBytes || effective.Timeout != ceiling.Timeout {
		t.Errorf("EffectivePolicyWithinCeiling(...) = %#v, want inherited unset values", effective)
	}

	if err := DefaultPolicy().CheckWriteSize(1); !errors.Is(err, ErrUnsupported) {
		t.Errorf("DefaultPolicy().CheckWriteSize(1) error = %v, want ErrUnsupported", err)
	}
	if err := (ExecutionPolicy{WriteMode: WriteModeWriteLimited, MaxWriteBytes: 2}).CheckWriteSize(3); err == nil {
		t.Fatalf("ExecutionPolicy.CheckWriteSize(3) unexpectedly succeeded for write-limited policy")
	}
	if err := (ExecutionPolicy{WriteMode: WriteModeFull}).CheckWriteSize(3); err != nil {
		t.Errorf("ExecutionPolicy{full}.CheckWriteSize(3) error = %v, want nil", err)
	}
	if err := PolicyWithinCeiling(
		ExecutionPolicy{
			WriteMode:        WriteModeWriteLimited,
			MaxWriteBytes:    8,
			MaxPipelineDepth: 2,
			MaxOutputBytes:   64,
			Timeout:          2 * time.Second,
		},
		ceiling,
	); !errors.Is(err, ErrPolicyCeilingExceeded) {
		t.Errorf("PolicyWithinCeiling(write-limited bytes exceed) error = %v, want ErrPolicyCeilingExceeded", err)
	}
	if err := PolicyWithinCeiling(
		ExecutionPolicy{
			WriteMode:        WriteModeWriteLimited,
			MaxWriteBytes:    4,
			MaxPipelineDepth: 2,
			MaxOutputBytes:   64,
			Timeout:          3 * time.Second,
		},
		ceiling,
	); !errors.Is(err, ErrPolicyCeilingExceeded) {
		t.Errorf("PolicyWithinCeiling(timeout exceed) error = %v, want ErrPolicyCeilingExceeded", err)
	}
}

func TestProfileHelpers(t *testing.T) {
	if got := DefaultProfile(); got != ProfileCoreStrict {
		t.Errorf("DefaultProfile() = %q, want %q", got, ProfileCoreStrict)
	}
	if got, err := ParseProfile(""); err != nil || got != ProfileCoreStrict {
		t.Errorf("ParseProfile(\"\") = (%q, %v), want (%q, nil)", got, err, ProfileCoreStrict)
	}
	if got, err := ParseProfile(string(ProfileBashPlus)); err != nil || got != ProfileBashPlus {
		t.Errorf("ParseProfile(%q) = (%q, %v), want (%q, nil)", ProfileBashPlus, got, err, ProfileBashPlus)
	}
	if _, err := ParseProfile("broken"); err == nil {
		t.Fatalf("ParseProfile(%q) unexpectedly succeeded", "broken")
	}

	bash := ProfileByName(ProfileBashPlus)
	if !bash.Capabilities["compat:date"] || bash.Capabilities["glob:extended"] {
		t.Errorf("ProfileByName(%q) = %#v, want compat:date only", ProfileBashPlus, bash.Capabilities)
	}
	zsh := ProfileByName(ProfileZshLite)
	if !zsh.Capabilities["compat:date"] || !zsh.Capabilities["glob:extended"] {
		t.Errorf("ProfileByName(%q) = %#v, want zsh-lite capabilities", ProfileZshLite, zsh.Capabilities)
	}
	core := ProfileByName(ProfileCoreStrict)
	if len(core.Capabilities) != 0 {
		t.Errorf("ProfileByName(%q) = %#v, want no capabilities", ProfileCoreStrict, core.Capabilities)
	}
}

func TestCommandReferenceHelpers(t *testing.T) {
	tests := []struct {
		name        string
		raw         string
		resolvePath func(string) (string, error)
		want        CommandReference
		wantErr     bool
		wantPathErr bool
	}{
		{
			name: "plain name",
			raw:  "echo",
			want: CommandReference{Raw: "echo", Name: "echo", PathLike: false},
		},
		{
			name: "builtin path",
			raw:  " /sys/bin/ls ",
			resolvePath: func(raw string) (string, error) {
				return "/sys/bin/ls", nil
			},
			want: CommandReference{
				Raw:          "/sys/bin/ls",
				Name:         "ls",
				ResolvedPath: "/sys/bin/ls",
				PathLike:     true,
				Namespace:    CommandNamespaceBuiltin,
			},
		},
		{
			name: "external path",
			raw:  "/bin/rg",
			resolvePath: func(raw string) (string, error) {
				return "/bin/rg", nil
			},
			want: CommandReference{
				Raw:          "/bin/rg",
				Name:         "rg",
				ResolvedPath: "/bin/rg",
				PathLike:     true,
				Namespace:    CommandNamespaceExternal,
			},
		},
		{
			name: "unsupported path-like without resolver",
			raw:  "/sys/bin/ls",
			wantErr: true,
		},
		{
			name: "path outside command roots",
			raw:  "/tmp/ls",
			resolvePath: func(raw string) (string, error) {
				return "/tmp/ls", nil
			},
			wantErr:     true,
			wantPathErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NormalizeCommandReference(tt.raw, tt.resolvePath)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("NormalizeCommandReference(%q) unexpectedly succeeded: %#v", tt.raw, got)
				}
				var pathErr CommandReferencePathError
				if gotPathErr := errors.As(err, &pathErr); gotPathErr != tt.wantPathErr {
					t.Fatalf("NormalizeCommandReference(%q) error = %T (%v), want path error = %t", tt.raw, err, err, tt.wantPathErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("NormalizeCommandReference(%q) error = %v", tt.raw, err)
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NormalizeCommandReference(%q) = %#v, want %#v", tt.raw, got, tt.want)
			}
		})
	}

	if got := IsCommandPathUnder("/sys/bin/tools/ls", VirtualSystemBinDir); got {
		t.Errorf("IsCommandPathUnder(%q, %q) = true, want false", "/sys/bin/tools/ls", VirtualSystemBinDir)
	}
	if got := normalizeVirtualCommandPath(" sys/bin/ls "); got != "/sys/bin/ls" {
		t.Errorf("normalizeVirtualCommandPath(...) = %q, want %q", got, "/sys/bin/ls")
	}

	pathErr := CommandReferencePathError{Raw: "/tmp/ls", ResolvedPath: "/tmp/ls"}
	if !strings.Contains(pathErr.Error(), VirtualSystemBinDir) || !strings.Contains(pathErr.Error(), VirtualExternalBinDir) {
		t.Errorf("CommandReferencePathError.Error() = %q, want runtime bin dirs in message", pathErr.Error())
	}
	if got := (CommandReferencePathError{Raw: " /tmp/ls "}).Error(); !strings.Contains(got, "cannot be resolved") {
		t.Errorf("CommandReferencePathError{without resolved}.Error() = %q, want unresolved-path message", got)
	}
}

func TestPathAccessHelpers(t *testing.T) {
	if got := NormalizePathAccess("bad"); got != PathAccessReadOnly {
		t.Errorf("NormalizePathAccess(%q) = %q, want %q", "bad", got, PathAccessReadOnly)
	}
	if got := NormalizePathAccess(PathAccessReadWrite); got != PathAccessReadWrite {
		t.Errorf("NormalizePathAccess(%q) = %q, want %q", PathAccessReadWrite, got, PathAccessReadWrite)
	}

	caps := NormalizePathCapabilities([]string{
		PathCapabilityWrite,
		"bad",
		PathCapabilityRead,
		PathCapabilityWrite,
		PathCapabilityDescribe,
	})
	wantCaps := []string{PathCapabilityDescribe, PathCapabilityRead, PathCapabilityWrite}
	if !reflect.DeepEqual(caps, wantCaps) {
		t.Errorf("NormalizePathCapabilities(...) = %#v, want %#v", caps, wantCaps)
	}

	got := StripWriteCapabilities([]string{PathCapabilityRead, PathCapabilityWrite, PathCapabilityEdit})
	want := []string{PathCapabilityRead}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("StripWriteCapabilities(...) = %#v, want %#v", got, want)
	}
}

func TestSessionCloneDeepCopiesState(t *testing.T) {
	original := Session{
		SessionID:     "sess-1",
		Profile:       ProfileBashPlus,
		PolicyCeiling: ExecutionPolicy{WriteMode: WriteModeWriteLimited, MaxWriteBytes: 4},
		State: SessionState{
			CommandAliases: map[string][]string{"ll": {"ls", "-l"}},
			EnvVars:        map[string]string{"FOO": "bar"},
			RCFiles:        []string{"/a", "/b"},
			WorkingDir:     " /task_outputs ",
			Opaque:         map[string]json.RawMessage{"a": json.RawMessage(`{"x":1}`)},
		},
	}

	clone := original.Clone()
	clone.State.CommandAliases["ll"][0] = "cat"
	clone.State.EnvVars["FOO"] = "changed"
	clone.State.RCFiles[0] = "/changed"
	clone.State.WorkingDir = "/other"
	clone.State.Opaque["a"][0] = '{'

	if got := original.State.CommandAliases["ll"][0]; got != "ls" {
		t.Errorf("Session.Clone() mutated original alias = %q, want %q", got, "ls")
	}
	if got := original.State.EnvVars["FOO"]; got != "bar" {
		t.Errorf("Session.Clone() mutated original env = %q, want %q", got, "bar")
	}
	if got := original.State.RCFiles[0]; got != "/a" {
		t.Errorf("Session.Clone() mutated original rc file = %q, want %q", got, "/a")
	}
	if got := original.State.WorkingDir; got != " /task_outputs " {
		t.Errorf("Session.Clone() mutated original working dir = %q, want %q", got, " /task_outputs ")
	}
	if got := clone.State.WorkingDir; got != "/other" {
		t.Errorf("Session.Clone() working dir = %q, want %q", got, "/other")
	}

	stateClone := original.State.Clone()
	if stateClone.WorkingDir != "/task_outputs" {
		t.Errorf("SessionState.Clone().WorkingDir = %q, want %q", stateClone.WorkingDir, "/task_outputs")
	}
}

func TestExecutionResultHelpers(t *testing.T) {
	tests := []struct {
		name string
		in   ExecutionResult
		want string
	}{
		{name: "stdout only", in: ExecutionResult{Stdout: "hello\n"}, want: "hello\n"},
		{name: "stderr only", in: ExecutionResult{Stderr: "boom"}, want: "boom"},
		{name: "both with newline", in: ExecutionResult{Stdout: "hello\n", Stderr: "boom"}, want: "hello\nboom"},
		{name: "both without newline", in: ExecutionResult{Stdout: "hello", Stderr: "boom"}, want: "hello\nboom"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.in.FlattenOutput(); got != tt.want {
				t.Errorf("ExecutionResult.FlattenOutput() = %q, want %q", got, tt.want)
			}
		})
	}

	got := ExecutionResult{SessionID: "old"}.WithSessionID("  new ")
	if got.SessionID != "new" {
		t.Errorf("ExecutionResult.WithSessionID(...) = %q, want %q", got.SessionID, "new")
	}
}

func TestAdapterProjectionMounts(t *testing.T) {
	m1 := fakeMount{point: "/memory"}
	m2 := fakeMount{point: "/external"}
	projection := AdapterProjection{
		VirtualMounts: []VirtualMount{m1},
		Memory:        MemoryProjection{Mount: m2, Freshness: "fresh"},
	}
	got := projection.ProjectionMounts()
	if len(got) != 2 {
		t.Fatalf("AdapterProjection.ProjectionMounts() len = %d, want 2", len(got))
	}
	if got[0].MountPoint() != "/memory" || got[1].MountPoint() != "/external" {
		t.Errorf("AdapterProjection.ProjectionMounts() = [%q, %q], want [/memory, /external]", got[0].MountPoint(), got[1].MountPoint())
	}
}

func TestOpsFromFilesystemWiresInterfaces(t *testing.T) {
	fs := &fakeFilesystem{
		root:    "/workspace",
		pathEnv: []string{"/bin"},
		mounts:  []VirtualMount{fakeMount{point: "/memory"}},
	}

	ops := OpsFromFilesystem(fs)
	if ops.RootDir != "/workspace" || ops.WorkingDir != "/workspace" {
		t.Fatalf("OpsFromFilesystem(...) root/cwd = (%q, %q), want (/workspace, /workspace)", ops.RootDir, ops.WorkingDir)
	}
	if ops.Profile != DefaultProfile() {
		t.Errorf("OpsFromFilesystem(...).Profile = %q, want %q", ops.Profile, DefaultProfile())
	}
	if ops.Policy != DefaultPolicy() {
		t.Errorf("OpsFromFilesystem(...).Policy = %#v, want %#v", ops.Policy, DefaultPolicy())
	}
	if len(ops.VirtualMounts) != 1 || ops.VirtualMounts[0].MountPoint() != "/memory" {
		t.Errorf("OpsFromFilesystem(...).VirtualMounts = %#v, want mount /memory", ops.VirtualMounts)
	}
	if !reflect.DeepEqual(ops.PathEnv, []string{"/bin"}) {
		t.Errorf("OpsFromFilesystem(...).PathEnv = %#v, want %#v", ops.PathEnv, []string{"/bin"})
	}
	if _, err := ops.RequireAbsolutePath("/workspace/file.txt"); err != nil {
		t.Fatalf("OpsFromFilesystem(...).RequireAbsolutePath(...) error = %v", err)
	}
	if meta, err := ops.DescribePath(context.Background(), "/workspace/file.txt"); err != nil || !meta.Exists {
		t.Fatalf("OpsFromFilesystem(...).DescribePath(...) = (%#v, %v), want existing meta", meta, err)
	}
	if err := ops.CheckPathOp(context.Background(), PathOpWrite, "/workspace/file.txt"); err != nil {
		t.Fatalf("OpsFromFilesystem(...).CheckPathOp(...) error = %v", err)
	}
}

type fakeFilesystem struct {
	root    string
	pathEnv []string
	mounts  []VirtualMount
}

func (f *fakeFilesystem) RootDir() string { return f.root }

func (f *fakeFilesystem) RequireAbsolutePath(raw string) (string, error) {
	return raw, nil
}

func (f *fakeFilesystem) ListChildren(ctx context.Context, dir string) ([]string, error) {
	return []string{dir + "/child"}, nil
}

func (f *fakeFilesystem) IsDirPath(ctx context.Context, path string) (bool, error) {
	return path == f.root, nil
}

func (f *fakeFilesystem) ReadRawContent(ctx context.Context, path string) (string, error) {
	return "content", nil
}

func (f *fakeFilesystem) ResolveSearchPaths(ctx context.Context, target string, recursive bool) ([]string, error) {
	return []string{target}, nil
}

func (f *fakeFilesystem) CollectFilesUnder(ctx context.Context, target string) ([]string, error) {
	return []string{target + "/file.txt"}, nil
}

func (f *fakeFilesystem) WriteFile(ctx context.Context, filePath string, content string) error { return nil }
func (f *fakeFilesystem) AppendFile(ctx context.Context, filePath string, content string) error { return nil }
func (f *fakeFilesystem) EditFile(ctx context.Context, filePath string, oldString string, newString string, replaceAll bool) error {
	return nil
}
func (f *fakeFilesystem) MakeDir(ctx context.Context, dirPath string) error     { return nil }
func (f *fakeFilesystem) RemoveFile(ctx context.Context, filePath string) error { return nil }
func (f *fakeFilesystem) RemoveDir(ctx context.Context, dirPath string) error   { return nil }
func (f *fakeFilesystem) DescribePath(ctx context.Context, path string) (PathMeta, error) {
	return PathMeta{Exists: true, Kind: "file"}, nil
}
func (f *fakeFilesystem) ListExternalCommands(ctx context.Context) ([]ExternalCommand, error) {
	return []ExternalCommand{{Name: "rg"}}, nil
}
func (f *fakeFilesystem) RunExternalCommand(ctx context.Context, req ExternalCommandRequest) (ExternalCommandResult, error) {
	return ExternalCommandResult{Stdout: "ok"}, nil
}
func (f *fakeFilesystem) ReadExternalManual(ctx context.Context, command string) (string, error) {
	return "manual", nil
}
func (f *fakeFilesystem) PathEnv() []string                      { return f.pathEnv }
func (f *fakeFilesystem) VirtualMounts() []VirtualMount         { return f.mounts }
func (f *fakeFilesystem) CheckPathOp(ctx context.Context, op PathOp, path string) error {
	return nil
}

type fakeMount struct {
	point string
}

func (m fakeMount) MountPoint() string { return m.point }
func (m fakeMount) Exists(context.Context) (bool, error) {
	return true, nil
}
func (m fakeMount) ListChildren(context.Context, string) ([]string, error) {
	return nil, nil
}
func (m fakeMount) IsDirPath(context.Context, string) (bool, error) {
	return true, nil
}
func (m fakeMount) ReadRawContent(context.Context, string) (string, error) {
	return "", nil
}
func (m fakeMount) CollectFilesUnder(context.Context, string) ([]string, error) {
	return nil, nil
}
func (m fakeMount) ResolveSearchPaths(context.Context, string, bool) ([]string, error) {
	return nil, nil
}
func (m fakeMount) DescribePath(context.Context, string) (PathMeta, error) {
	return PathMeta{}, nil
}
