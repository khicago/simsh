package engine_test

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/khicago/simsh/pkg/adapter/localfs"
	"github.com/khicago/simsh/pkg/contract"
)

type remoteHighMount struct {
	point    string
	profile  contract.MountProfile
	entries  map[string]contract.MountEntry
	contents map[string]string
}

func newRemoteHighMount(profile contract.MountProfile, entries map[string]contract.MountEntry, contents map[string]string) remoteHighMount {
	return remoteHighMount{
		point:    "/remote",
		profile:  contract.NormalizeMountProfile(profile),
		entries:  entries,
		contents: contents,
	}
}

func (m remoteHighMount) MountPoint() string { return m.point }
func (m remoteHighMount) Profile() contract.MountProfile {
	return contract.NormalizeMountProfile(m.profile)
}
func (m remoteHighMount) Exists(context.Context) (bool, error) { return true, nil }
func (m remoteHighMount) StatPath(_ context.Context, pathValue string) (contract.MountEntry, error) {
	entry, ok := m.entries[pathValue]
	if !ok {
		return contract.MountEntry{}, fmt.Errorf("%s: No such file or directory", pathValue)
	}
	return entry, nil
}
func (m remoteHighMount) ReadContent(_ context.Context, pathValue string) (string, error) {
	content, ok := m.contents[pathValue]
	if !ok {
		return "", fmt.Errorf("%s: No such file or directory", pathValue)
	}
	return content, nil
}

type remoteHighEnumeratingMount struct {
	remoteHighMount
	enumerated []contract.MountEntry
}

func (m remoteHighEnumeratingMount) EnumeratePaths(context.Context, contract.EnumeratePathsRequest) (contract.EnumeratePathsResult, error) {
	return contract.EnumeratePathsResult{Entries: append([]contract.MountEntry(nil), m.enumerated...)}, nil
}

func TestEngineLSRemoteHighLatencyRequiresEntryLister(t *testing.T) {
	eng := newTestEngine()
	root := t.TempDir()
	ops, err := localfs.NewOps(localfs.Options{
		RootDir: root,
		Profile: contract.ProfileBashPlus,
		Policy:  contract.DefaultPolicy(),
		VirtualMounts: []contract.VirtualMount{
			newRemoteHighMount(contract.MountProfile{
				LatencyClass:        contract.MountLatencyRemoteHigh,
				SupportedCLIClasses: []contract.MountCLIClass{contract.MountCLIList},
			}, map[string]contract.MountEntry{
				"/remote": {Path: "/remote", Name: "remote", Meta: contract.PathMeta{Exists: true, IsDir: true}},
			}, nil),
		},
	})
	if err != nil {
		t.Fatalf("localfs.NewOps(...) error = %v", err)
	}

	out, code := eng.Execute(context.Background(), "ls /remote", ops)
	if code == 0 || !strings.Contains(out, "entry listing requires EntryLister for remote_high_latency mount") {
		t.Fatalf("ls remote_high_latency = (%q, %d), want explicit EntryLister refusal", out, code)
	}
}

func TestEngineFindRemoteHighLatencyRequiresPathEnumerator(t *testing.T) {
	eng := newTestEngine()
	root := t.TempDir()
	ops, err := localfs.NewOps(localfs.Options{
		RootDir: root,
		Profile: contract.ProfileBashPlus,
		Policy:  contract.DefaultPolicy(),
		VirtualMounts: []contract.VirtualMount{
			newRemoteHighMount(contract.MountProfile{
				LatencyClass:        contract.MountLatencyRemoteHigh,
				SupportedCLIClasses: []contract.MountCLIClass{contract.MountCLIFind},
			}, map[string]contract.MountEntry{
				"/remote":      {Path: "/remote", Name: "remote", Meta: contract.PathMeta{Exists: true, IsDir: true}},
				"/remote/a.md": {Path: "/remote/a.md", Name: "a.md", Meta: contract.PathMeta{Exists: true, IsDir: false}},
			}, nil),
		},
	})
	if err != nil {
		t.Fatalf("localfs.NewOps(...) error = %v", err)
	}

	out, code := eng.Execute(context.Background(), "find /remote", ops)
	if code == 0 || !strings.Contains(out, "path enumeration requires PathEnumerator for remote_high_latency mount") {
		t.Fatalf("find remote_high_latency = (%q, %d), want explicit PathEnumerator refusal", out, code)
	}
}

func TestEngineJSONStatRemoteHighLatencyRequiresBulkReader(t *testing.T) {
	eng := newTestEngine()
	root := t.TempDir()
	ops, err := localfs.NewOps(localfs.Options{
		RootDir: root,
		Profile: contract.ProfileBashPlus,
		Policy:  contract.DefaultPolicy(),
		VirtualMounts: []contract.VirtualMount{
			remoteHighEnumeratingMount{
				remoteHighMount: newRemoteHighMount(contract.MountProfile{
					LatencyClass: contract.MountLatencyRemoteHigh,
					SupportedCLIClasses: []contract.MountCLIClass{
						contract.MountCLIFind,
						contract.MountCLIBulkRead,
					},
				}, map[string]contract.MountEntry{
					"/remote":           {Path: "/remote", Name: "remote", Meta: contract.PathMeta{Exists: true, IsDir: true}},
					"/remote/a.json":    {Path: "/remote/a.json", Name: "a.json", Meta: contract.PathMeta{Exists: true, IsDir: false}},
					"/remote/b.json":    {Path: "/remote/b.json", Name: "b.json", Meta: contract.PathMeta{Exists: true, IsDir: false}},
				}, map[string]string{
					"/remote/a.json": `{"a":1}`,
					"/remote/b.json": `{"b":2}`,
				}),
				enumerated: []contract.MountEntry{
					{Path: "/remote/a.json", Name: "a.json", Meta: contract.PathMeta{Exists: true, IsDir: false}},
					{Path: "/remote/b.json", Name: "b.json", Meta: contract.PathMeta{Exists: true, IsDir: false}},
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("localfs.NewOps(...) error = %v", err)
	}

	out, code := eng.Execute(context.Background(), "json stat -r /remote", ops)
	if code == 0 || !strings.Contains(out, "bulk read requires an explicit mount capability on remote_high_latency mounts") {
		t.Fatalf("json stat remote_high_latency = (%q, %d), want explicit BulkReader refusal", out, code)
	}
}
