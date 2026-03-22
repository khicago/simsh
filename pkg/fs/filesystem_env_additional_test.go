package fs

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing"

	"github.com/khicago/simsh/pkg/contract"
)

func TestNewAIFilesystemExposesSemanticZones(t *testing.T) {
	fsys, err := NewAIFilesystem(t.TempDir(), 0, nil)
	if err != nil {
		t.Fatalf("NewAIFilesystem(...) error = %v", err)
	}

	root, ok := fsys.(contract.PathRootProvider)
	if !ok {
		t.Fatalf("NewAIFilesystem(...) did not implement PathRootProvider")
	}
	if got := root.RootDir(); got != "/" {
		t.Fatalf("RootDir() = %q, want %q", got, "/")
	}

	children, err := fsys.ListChildren(context.Background(), "/")
	if err != nil {
		t.Fatalf("ListChildren(/) error = %v", err)
	}
	wantChildren := []string{"/knowledge_base", "/task_outputs", "/temp_work"}
	if !reflect.DeepEqual(children, wantChildren) {
		t.Errorf("ListChildren(/) = %#v, want %#v", children, wantChildren)
	}

	if _, err := fsys.RequireAbsolutePath("/task_outputs/report.md"); err != nil {
		t.Fatalf("RequireAbsolutePath(/task_outputs/report.md) error = %v", err)
	}
	if err := fsys.WriteFile(context.Background(), "/task_outputs/report.md", "ok"); err != nil {
		t.Fatalf("WriteFile(/task_outputs/report.md) error = %v", err)
	}
	if _, err := fsys.ReadRawContent(context.Background(), "/knowledge_base/missing.md"); err == nil {
		t.Fatalf("ReadRawContent(/knowledge_base/missing.md) unexpectedly succeeded")
	}
}

func TestNewAIFilesystemFallsBackToCurrentWorkingDirectory(t *testing.T) {
	prev, err := os.Getwd()
	if err != nil {
		t.Fatalf("os.Getwd() error = %v", err)
	}
	hostRoot := t.TempDir()
	if err := os.Chdir(hostRoot); err != nil {
		t.Fatalf("os.Chdir(%q) error = %v", hostRoot, err)
	}
	t.Cleanup(func() {
		if chdirErr := os.Chdir(prev); chdirErr != nil {
			t.Fatalf("restore cwd failed: %v", chdirErr)
		}
	})

	fsys, err := NewAIFilesystem("", 0, nil)
	if err != nil {
		t.Fatalf("NewAIFilesystem(blank hostRoot) error = %v", err)
	}
	if err := fsys.WriteFile(context.Background(), "/task_outputs/from-cwd.txt", "ok"); err != nil {
		t.Fatalf("WriteFile(/task_outputs/from-cwd.txt) error = %v", err)
	}
	if raw, err := os.ReadFile(filepath.Join(hostRoot, "task_outputs", "from-cwd.txt")); err != nil || string(raw) != "ok" {
		t.Fatalf("os.ReadFile(host task_outputs) = (%q, %v), want (%q, nil)", string(raw), err, "ok")
	}
}

func TestNewRuntimeOpsWiresOptionsAndCorpusMount(t *testing.T) {
	ops, err := NewRuntimeOps(EnvironmentOptions{
		HostRoot: t.TempDir(),
		Profile:  contract.ProfileBashPlus,
		Policy: contract.ExecutionPolicy{
			WriteMode:     contract.WriteModeWriteLimited,
			MaxWriteBytes: 7,
		},
		CommandAliases: map[string][]string{
			" ll ": {" ls ", "-l"},
		},
		EnvVars: map[string]string{
			"FOO": "bar",
		},
		RCFiles: []string{" /task_outputs/simshrc "},
		PathEnv: []string{"/custom/bin"},
			ExternalCallbacks: ExternalCallbacks{
				ListExternalCommands: func(_ context.Context) ([]contract.ExternalCommand, error) {
					return []contract.ExternalCommand{{Name: "rg", Summary: "ripgrep"}}, nil
				},
				RunExternalCommand: func(_ context.Context, req contract.ExternalCommandRequest) (contract.ExternalCommandResult, error) {
					if req.Command != "rg" {
						return contract.ExternalCommandResult{}, errors.New("unexpected command")
					}
					return contract.ExternalCommandResult{Stdout: "ok", ExitCode: 0}, nil
				},
				ReadExternalManual: func(_ context.Context, _ string) (string, error) {
					return "manual", nil
				},
			},
		EnableTestCorpus: true,
	})
	if err != nil {
		t.Fatalf("NewRuntimeOps(...) error = %v", err)
	}

	if ops.Profile != contract.ProfileBashPlus {
		t.Errorf("NewRuntimeOps(...).Profile = %q, want %q", ops.Profile, contract.ProfileBashPlus)
	}
	if ops.Policy.WriteMode != contract.WriteModeWriteLimited || ops.Policy.MaxWriteBytes != 7 {
		t.Errorf("NewRuntimeOps(...).Policy = %#v, want write-limited policy with MaxWriteBytes=7", ops.Policy)
	}
	if !reflect.DeepEqual(ops.CommandAliases, map[string][]string{"ll": {"ls", "-l"}}) {
		t.Errorf("NewRuntimeOps(...).CommandAliases = %#v, want normalized ll alias", ops.CommandAliases)
	}
	if !reflect.DeepEqual(ops.EnvVars, map[string]string{"FOO": "bar"}) {
		t.Errorf("NewRuntimeOps(...).EnvVars = %#v, want %#v", ops.EnvVars, map[string]string{"FOO": "bar"})
	}
	if !reflect.DeepEqual(ops.RCFiles, []string{"/task_outputs/simshrc"}) {
		t.Errorf("NewRuntimeOps(...).RCFiles = %#v, want %#v", ops.RCFiles, []string{"/task_outputs/simshrc"})
	}
	wantPathEnv := []string{"/task_outputs", "/temp_work", "/knowledge_base", "/custom/bin"}
	gotPathEnv := append([]string(nil), ops.PathEnv...)
	sort.Strings(gotPathEnv)
	sort.Strings(wantPathEnv)
	if !reflect.DeepEqual(gotPathEnv, wantPathEnv) {
		t.Errorf("NewRuntimeOps(...).PathEnv = %#v, want default zones plus custom path", ops.PathEnv)
	}
	if len(ops.VirtualMounts) == 0 {
		t.Fatalf("NewRuntimeOps(...).VirtualMounts = empty, want baseline corpus mount")
	}
	foundCorpus := false
	for _, mount := range ops.VirtualMounts {
		if mount.MountPoint() == "/test" {
			foundCorpus = true
			break
		}
	}
	if !foundCorpus {
		t.Errorf("NewRuntimeOps(...).VirtualMounts missing /test corpus mount: %#v", ops.VirtualMounts)
	}

	cmds, err := ops.ListExternalCommands(context.Background())
	if err != nil {
		t.Fatalf("ListExternalCommands() error = %v", err)
	}
	if !reflect.DeepEqual(cmds, []contract.ExternalCommand{{Name: "rg", Summary: "ripgrep"}}) {
		t.Errorf("ListExternalCommands() = %#v, want %#v", cmds, []contract.ExternalCommand{{Name: "rg", Summary: "ripgrep"}})
	}
	runResult, err := ops.RunExternalCommand(context.Background(), contract.ExternalCommandRequest{Command: "rg"})
	if err != nil || runResult.Stdout != "ok" {
		t.Fatalf("RunExternalCommand(...) = (%#v, %v), want stdout=ok", runResult, err)
	}
	manual, err := ops.ReadExternalManual(context.Background(), "rg")
	if err != nil || manual != "manual" {
		t.Fatalf("ReadExternalManual(...) = (%q, %v), want (%q, nil)", manual, err, "manual")
	}
}

func TestDescribeMarkdownMentionsSemanticZones(t *testing.T) {
	markdown := DescribeMarkdown()
	for _, want := range []string{"/task_outputs", "/temp_work", "/knowledge_base", "path policy is zone-scoped"} {
		if !strings.Contains(markdown, want) {
			t.Errorf("DescribeMarkdown() missing %q in output:\n%s", want, markdown)
		}
	}
}
