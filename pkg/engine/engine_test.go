package engine_test

import (
	"context"
	"encoding/json"
	"fmt"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/khicago/simsh/pkg/adapter/localfs"
	"github.com/khicago/simsh/pkg/builtin"
	"github.com/khicago/simsh/pkg/contract"
	"github.com/khicago/simsh/pkg/engine"
	"github.com/khicago/simsh/pkg/mount"
)

type testFS struct {
	root     string
	files    map[string]string
	dirs     map[string]struct{}
	external []contract.ExternalCommand
}

func newTestFS() *testFS {
	fs := &testFS{root: "/", files: map[string]string{}, dirs: map[string]struct{}{"/": {}}}
	fs.mustWrite("/workspace/readme.md", "hello\nworld\n")
	fs.mustWrite("/workspace/todo.txt", "todo item\n")
	fs.external = []contract.ExternalCommand{{Name: "report_tool", Summary: "report tool manual"}}
	return fs
}

func (f *testFS) RootDir() string { return f.root }

func (f *testFS) RequireAbsolutePath(raw string) (string, error) {
	if strings.TrimSpace(raw) == "" {
		return "", fmt.Errorf("path is required")
	}
	if !strings.HasPrefix(raw, "/") {
		return "", fmt.Errorf("path must be absolute: %s", raw)
	}
	return normalizePath(raw), nil
}

func (f *testFS) ListChildren(ctx context.Context, dir string) ([]string, error) {
	_ = ctx
	dir = normalizePath(dir)
	if _, ok := f.dirs[dir]; !ok {
		return nil, fmt.Errorf("%s: No such file or directory", dir)
	}
	children := make([]string, 0)
	prefix := dir
	if prefix != "/" {
		prefix += "/"
	}
	for d := range f.dirs {
		if d == dir || !strings.HasPrefix(d, prefix) {
			continue
		}
		rest := strings.TrimPrefix(d, prefix)
		if strings.Contains(rest, "/") {
			continue
		}
		children = append(children, d)
	}
	for filePath := range f.files {
		if !strings.HasPrefix(filePath, prefix) {
			continue
		}
		rest := strings.TrimPrefix(filePath, prefix)
		if strings.Contains(rest, "/") {
			continue
		}
		children = append(children, filePath)
	}
	sort.Strings(children)
	return children, nil
}

func (f *testFS) IsDirPath(ctx context.Context, pathValue string) (bool, error) {
	_ = ctx
	pathValue = normalizePath(pathValue)
	_, ok := f.dirs[pathValue]
	return ok, nil
}

func (f *testFS) ReadRawContent(ctx context.Context, pathValue string) (string, error) {
	_ = ctx
	pathValue = normalizePath(pathValue)
	raw, ok := f.files[pathValue]
	if !ok {
		return "", fmt.Errorf("%s: No such file or directory", pathValue)
	}
	return raw, nil
}

func (f *testFS) ResolveSearchPaths(ctx context.Context, target string, recursive bool) ([]string, error) {
	if _, ok := f.files[target]; ok {
		return []string{target}, nil
	}
	if _, ok := f.dirs[target]; !ok {
		return nil, fmt.Errorf("%s: No such file or directory", target)
	}
	if !recursive {
		return nil, fmt.Errorf("%s: Is a directory (use -r to search recursively)", target)
	}
	return f.CollectFilesUnder(ctx, target)
}

func (f *testFS) CollectFilesUnder(ctx context.Context, target string) ([]string, error) {
	_ = ctx
	target = normalizePath(target)
	if _, ok := f.files[target]; ok {
		return []string{target}, nil
	}
	if _, ok := f.dirs[target]; !ok {
		return nil, fmt.Errorf("%s: No such file or directory", target)
	}
	out := make([]string, 0)
	prefix := target
	if prefix != "/" {
		prefix += "/"
	}
	for filePath := range f.files {
		if target == "/" || strings.HasPrefix(filePath, prefix) {
			out = append(out, filePath)
		}
	}
	sort.Strings(out)
	return out, nil
}

func (f *testFS) WriteFile(ctx context.Context, filePath string, content string) error {
	_ = ctx
	f.mustWrite(filePath, content)
	return nil
}

func (f *testFS) AppendFile(ctx context.Context, filePath string, content string) error {
	_ = ctx
	filePath = normalizePath(filePath)
	f.ensureDir(path.Dir(filePath))
	f.files[filePath] += content
	return nil
}

func (f *testFS) EditFile(ctx context.Context, filePath string, oldString string, newString string, replaceAll bool) error {
	_ = ctx
	filePath = normalizePath(filePath)
	raw, ok := f.files[filePath]
	if !ok {
		return fmt.Errorf("%s: No such file or directory", filePath)
	}
	if replaceAll {
		f.files[filePath] = strings.ReplaceAll(raw, oldString, newString)
		return nil
	}
	f.files[filePath] = strings.Replace(raw, oldString, newString, 1)
	return nil
}

func (f *testFS) ListExternalCommands(ctx context.Context) ([]contract.ExternalCommand, error) {
	_ = ctx
	out := make([]contract.ExternalCommand, len(f.external))
	copy(out, f.external)
	return out, nil
}

func (f *testFS) ReadExternalManual(ctx context.Context, command string) (string, error) {
	_ = ctx
	if command == "report_tool" {
		return "report tool manual", nil
	}
	return "", contract.ErrUnsupported
}

func (f *testFS) RunExternalCommand(ctx context.Context, req contract.ExternalCommandRequest) (contract.ExternalCommandResult, error) {
	_ = ctx
	if req.Command == "report_tool" {
		return contract.ExternalCommandResult{Stdout: "report ok", ExitCode: 0}, nil
	}
	return contract.ExternalCommandResult{}, contract.ErrUnsupported
}

func (f *testFS) MakeDir(ctx context.Context, dirPath string) error {
	_ = ctx
	f.ensureDir(normalizePath(dirPath))
	return nil
}

func (f *testFS) RemoveFile(ctx context.Context, filePath string) error {
	_ = ctx
	filePath = normalizePath(filePath)
	if _, ok := f.files[filePath]; !ok {
		return fmt.Errorf("%s: No such file", filePath)
	}
	delete(f.files, filePath)
	return nil
}

func (f *testFS) RemoveDir(ctx context.Context, dirPath string) error {
	_ = ctx
	dirPath = normalizePath(dirPath)
	if dirPath == "/" {
		return fmt.Errorf("cannot remove root directory: %s", dirPath)
	}
	if _, ok := f.dirs[dirPath]; !ok {
		return fmt.Errorf("%s: No such directory", dirPath)
	}
	prefix := dirPath + "/"
	for existing := range f.dirs {
		if existing != dirPath && strings.HasPrefix(existing, prefix) {
			return fmt.Errorf("%s: Directory not empty", dirPath)
		}
	}
	for filePath := range f.files {
		if strings.HasPrefix(filePath, prefix) {
			return fmt.Errorf("%s: Directory not empty", dirPath)
		}
	}
	delete(f.dirs, dirPath)
	return nil
}

func (f *testFS) mustWrite(filePath string, content string) {
	filePath = normalizePath(filePath)
	f.ensureDir(path.Dir(filePath))
	f.files[filePath] = content
}

func (f *testFS) ensureDir(dir string) {
	dir = normalizePath(dir)
	for {
		f.dirs[dir] = struct{}{}
		if dir == "/" {
			return
		}
		dir = path.Dir(dir)
	}
}

func normalizePath(raw string) string {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return "/"
	}
	if !strings.HasPrefix(trimmed, "/") {
		trimmed = "/" + trimmed
	}
	return path.Clean(trimmed)
}

func newTestEngine() *engine.Engine {
	reg := engine.NewRegistry()
	builtin.RegisterDefaults(reg)
	return engine.New(reg)
}

func TestEngineBuiltinAndExternalMounts(t *testing.T) {
	eng := newTestEngine()
	fs := newTestFS()
	ops := contract.OpsFromFilesystem(fs)

	out, code := eng.Execute(context.Background(), "ls /sys/bin", ops)
	if code != 0 {
		t.Fatalf("ls /sys/bin failed: code=%d out=%q", code, out)
	}
	if !strings.Contains(out, "/sys/bin/ls") {
		t.Fatalf("expected /sys/bin/ls in output: %q", out)
	}

	out, code = eng.Execute(context.Background(), "ls /sys", ops)
	if code != 0 {
		t.Fatalf("ls /sys failed: code=%d out=%q", code, out)
	}
	if !strings.Contains(out, "/sys/bin") {
		t.Fatalf("expected /sys/bin in output: %q", out)
	}

	out, code = eng.Execute(context.Background(), "tree -L 1 /sys", ops)
	if code != 0 {
		t.Fatalf("tree -L 1 /sys failed: code=%d out=%q", code, out)
	}
	if !strings.Contains(out, "bin") {
		t.Fatalf("expected bin in tree output: %q", out)
	}

	out, code = eng.Execute(context.Background(), "ls /bin", ops)
	if code != 0 {
		t.Fatalf("ls /bin failed: code=%d out=%q", code, out)
	}
	if !strings.Contains(out, "/bin/report_tool") {
		t.Fatalf("expected /bin/report_tool in output: %q", out)
	}

	out, code = eng.Execute(context.Background(), "which ls report_tool", ops)
	if code != 0 {
		t.Fatalf("which ls report_tool failed: code=%d out=%q", code, out)
	}
	if !strings.Contains(out, "/sys/bin/ls") || !strings.Contains(out, "/bin/report_tool") {
		t.Fatalf("unexpected which output: %q", out)
	}

	out, code = eng.Execute(context.Background(), "type ls report_tool", ops)
	if code != 0 {
		t.Fatalf("type ls report_tool failed: code=%d out=%q", code, out)
	}
	if !strings.Contains(out, "ls is /sys/bin/ls (builtin)") || !strings.Contains(out, "report_tool is /bin/report_tool (external)") {
		t.Fatalf("unexpected type output: %q", out)
	}
}

func TestEngineDefaultCommandAliases(t *testing.T) {
	eng := newTestEngine()
	fs := newTestFS()
	ops := contract.OpsFromFilesystem(fs)

	out, code := eng.Execute(context.Background(), "ll /sys", ops)
	if code != 0 {
		t.Fatalf("ll /sys failed: code=%d out=%q", code, out)
	}
	if !strings.Contains(out, "# columns: mode access kind lines path") {
		t.Fatalf("ll should expand to ls -l: out=%q", out)
	}

	out, code = eng.Execute(context.Background(), "fm stat /workspace/readme.md", ops)
	if code != 0 {
		t.Fatalf("fm stat failed: code=%d out=%q", code, out)
	}
	if !strings.Contains(out, "n - 0 - /workspace/readme.md") {
		t.Fatalf("unexpected fm output: %q", out)
	}

	out, code = eng.Execute(context.Background(), "type ll", ops)
	if code != 0 || !strings.Contains(out, "ll is alias ll='ls -l' (alias)") {
		t.Fatalf("type ll should show alias resolution: code=%d out=%q", code, out)
	}
}

func TestEngineRCBootstrapFromMountedFile(t *testing.T) {
	eng := newTestEngine()
	fs := newTestFS()
	fs.mustWrite("/workspace/frontmatter.md", "---\ntitle: RC Loaded\n---\nbody\n")

	rcMount, err := mount.NewStaticMount("/etc", "config", map[string]string{
		"/etc/simshrc": strings.Join([]string{
			"export PROJECT=simsh",
			"export PATH=/sys/bin:/bin:/custom/bin",
			"alias fmget='frontmatter get --key title'",
		}, "\n") + "\n",
	})
	if err != nil {
		t.Fatalf("new rc static mount failed: %v", err)
	}

	ops := contract.OpsFromFilesystem(fs)
	ops.VirtualMounts = append(ops.VirtualMounts, rcMount)
	ops.RCFiles = []string{"/etc/simshrc"}

	out, code := eng.Execute(context.Background(), "env PROJECT", ops)
	if code != 0 || strings.TrimSpace(out) != "PROJECT=simsh" {
		t.Fatalf("expected env PROJECT from rc: code=%d out=%q", code, out)
	}

	out, code = eng.Execute(context.Background(), "env PATH", ops)
	if code != 0 || strings.TrimSpace(out) != "PATH=/sys/bin:/bin:/custom/bin" {
		t.Fatalf("expected env PATH override from rc: code=%d out=%q", code, out)
	}

	out, code = eng.Execute(context.Background(), "fmget /workspace/frontmatter.md", ops)
	if code != 0 || strings.TrimSpace(out) != "RC Loaded" {
		t.Fatalf("expected alias from rc to execute: code=%d out=%q", code, out)
	}
}

func TestEngineRCBootstrapRejectsUnsupportedStatement(t *testing.T) {
	eng := newTestEngine()
	fs := newTestFS()
	rcMount, err := mount.NewStaticMount("/etc", "config", map[string]string{
		"/etc/simshrc": "source /etc/bashrc\n",
	})
	if err != nil {
		t.Fatalf("new rc static mount failed: %v", err)
	}

	ops := contract.OpsFromFilesystem(fs)
	ops.VirtualMounts = append(ops.VirtualMounts, rcMount)
	ops.RCFiles = []string{"/etc/simshrc"}

	out, code := eng.Execute(context.Background(), "env", ops)
	if code == 0 {
		t.Fatalf("expected rc parse failure, out=%q", out)
	}
	if !strings.Contains(out, "rc: parse /etc/simshrc failed") || !strings.Contains(out, "unsupported statement") {
		t.Fatalf("unexpected rc parse error: %q", out)
	}
}

func TestEngineLSLongFormatAccessColumns(t *testing.T) {
	eng := newTestEngine()
	fs := newTestFS()
	ops := contract.OpsFromFilesystem(fs)

	out, code := eng.Execute(context.Background(), "ls -l /sys", ops)
	if code != 0 {
		t.Fatalf("ls -l /sys failed: code=%d out=%q", code, out)
	}
	lines := strings.Split(strings.TrimSpace(out), "\n")
	if len(lines) < 2 {
		t.Fatalf("expected at least 2 lines (row + legend), got %d: %q", len(lines), out)
	}
	if lines[len(lines)-1] != "# columns: mode access kind lines path" {
		t.Fatalf("expected legend line at end, got %q", lines[len(lines)-1])
	}
	legendCount := 0
	for _, line := range lines {
		if strings.HasPrefix(line, "# columns:") {
			legendCount++
		}
	}
	if legendCount != 1 {
		t.Fatalf("expected exactly one legend line, got %d: %q", legendCount, out)
	}

	foundSysBin := false
	for _, line := range lines {
		if strings.HasSuffix(line, "/sys/bin") {
			foundSysBin = true
			if !strings.HasPrefix(line, "d ro sys_bin_dir ") {
				t.Fatalf("unexpected /sys/bin row format: %q", line)
			}
		}
	}
	if !foundSysBin {
		t.Fatalf("expected /sys/bin row in output: %q", out)
	}
}

func TestEngineLSLongFormatMarkdown(t *testing.T) {
	eng := newTestEngine()
	fs := newTestFS()
	ops := contract.OpsFromFilesystem(fs)

	out, code := eng.Execute(context.Background(), "ls -l --fmt md /sys", ops)
	if code != 0 {
		t.Fatalf("ls -l --fmt md /sys failed: code=%d out=%q", code, out)
	}
	if !strings.HasPrefix(out, "| mode | access | kind | lines | path |") {
		t.Fatalf("expected markdown table header, got %q", out)
	}
	if !strings.Contains(out, "| d | ro | sys_bin_dir | - | /sys/bin |") {
		t.Fatalf("expected /sys/bin row in markdown output, got %q", out)
	}
}

func TestEngineLSLongFormatJSON(t *testing.T) {
	eng := newTestEngine()
	fs := newTestFS()
	ops := contract.OpsFromFilesystem(fs)

	out, code := eng.Execute(context.Background(), "ls -l --fmt json /sys", ops)
	if code != 0 {
		t.Fatalf("ls -l --fmt json /sys failed: code=%d out=%q", code, out)
	}
	var resp struct {
		Columns []string `json:"columns"`
		Entries []struct {
			Mode         string   `json:"mode"`
			Access       string   `json:"access"`
			Kind         string   `json:"kind"`
			Lines        int      `json:"lines"`
			Path         string   `json:"path"`
			Capabilities []string `json:"capabilities"`
		} `json:"entries"`
	}
	if err := json.Unmarshal([]byte(out), &resp); err != nil {
		t.Fatalf("unmarshal json failed: %v out=%q", err, out)
	}
	if len(resp.Columns) == 0 || len(resp.Entries) == 0 {
		t.Fatalf("expected non-empty json payload, got columns=%v entries=%v", resp.Columns, resp.Entries)
	}
	found := false
	for _, row := range resp.Entries {
		if row.Path == "/sys/bin" {
			found = true
			if row.Mode != "d" || row.Access != "ro" || row.Kind != "sys_bin_dir" {
				t.Fatalf("unexpected /sys/bin row: %+v", row)
			}
			if len(row.Capabilities) == 0 {
				t.Fatalf("expected capabilities for /sys/bin row: %+v", row)
			}
		}
	}
	if !found {
		t.Fatalf("expected /sys/bin entry in json output: %q", out)
	}
}

func TestEngineMountParentIsImmutable(t *testing.T) {
	eng := newTestEngine()
	fs := newTestFS()
	ops := writableOps(fs)

	out, code := eng.Execute(context.Background(), "mkdir /sys", ops)
	if code == 0 {
		t.Fatalf("expected mkdir /sys to fail: out=%q", out)
	}

	out, code = eng.Execute(context.Background(), "rm /sys/bin/ls", ops)
	if code == 0 {
		t.Fatalf("expected rm /sys/bin/ls to fail: out=%q", out)
	}

	out, code = eng.Execute(context.Background(), "cp /sys/bin/ls /workspace/copied", ops)
	if code == 0 {
		t.Fatalf("expected cp from /sys/bin to fail: out=%q", out)
	}

	out, code = eng.Execute(context.Background(), "cp /workspace/readme.md /sys/bin/copied", ops)
	if code == 0 {
		t.Fatalf("expected cp into /sys/bin to fail: out=%q", out)
	}

	out, code = eng.Execute(context.Background(), "mv /sys/bin/ls /workspace/moved", ops)
	if code == 0 {
		t.Fatalf("expected mv from /sys/bin to fail: out=%q", out)
	}
}

func TestEngineCompositeMutationsPreflightUnsupportedPaths(t *testing.T) {
	eng := newTestEngine()

	t.Run("rm keeps earlier file when later operand is unsupported", func(t *testing.T) {
		fs := newTestFS()
		ops := writableOps(fs)

		out, code := eng.Execute(context.Background(), "rm /workspace/todo.txt /sys/bin/ls", ops)
		if code == 0 {
			t.Fatalf("expected rm to fail: out=%q", out)
		}
		if _, ok := fs.files["/workspace/todo.txt"]; !ok {
			t.Fatalf("rm should not remove earlier file before unsupported operand")
		}
	})

	t.Run("mkdir keeps earlier dir absent when later operand is unsupported", func(t *testing.T) {
		fs := newTestFS()
		ops := writableOps(fs)

		out, code := eng.Execute(context.Background(), "mkdir /workspace/newdir /sys/bin/newdir", ops)
		if code == 0 {
			t.Fatalf("expected mkdir to fail: out=%q", out)
		}
		if _, ok := fs.dirs["/workspace/newdir"]; ok {
			t.Fatalf("mkdir should not create earlier dir before unsupported operand")
		}
	})

	t.Run("touch keeps earlier file absent when later operand is unsupported", func(t *testing.T) {
		fs := newTestFS()
		ops := writableOps(fs)

		out, code := eng.Execute(context.Background(), "touch /workspace/new.txt /sys/bin/new.txt", ops)
		if code == 0 {
			t.Fatalf("expected touch to fail: out=%q", out)
		}
		if _, ok := fs.files["/workspace/new.txt"]; ok {
			t.Fatalf("touch should not create earlier file before unsupported operand")
		}
	})

	t.Run("mv keeps source when destination is unsupported", func(t *testing.T) {
		fs := newTestFS()
		ops := writableOps(fs)

		out, code := eng.Execute(context.Background(), "mv /workspace/todo.txt /sys/bin/moved.txt", ops)
		if code == 0 {
			t.Fatalf("expected mv to fail: out=%q", out)
		}
		if _, ok := fs.files["/workspace/todo.txt"]; !ok {
			t.Fatalf("mv should keep source when destination preflight fails")
		}
		if _, ok := fs.files["/sys/bin/moved.txt"]; ok {
			t.Fatalf("mv should not create destination when destination preflight fails")
		}
	})
}

func TestEngineScriptOpsAndRedirectionPolicy(t *testing.T) {
	eng := newTestEngine()
	fs := newTestFS()
	ops := contract.OpsFromFilesystem(fs)
	ops.Policy = contract.DefaultPolicy()

	out, code := eng.Execute(context.Background(), "echo hello | cat", ops)
	if code != 0 || strings.TrimSpace(out) != "hello" {
		t.Fatalf("pipe failed: code=%d out=%q", code, out)
	}

	out, code = eng.Execute(context.Background(), "echo x > /workspace/out.txt", ops)
	if code == 0 {
		t.Fatalf("expected write to fail in read-only policy: out=%q", out)
	}

	ops.Policy = contract.ExecutionPolicy{WriteMode: contract.WriteModeFull, MaxPipelineDepth: 16, MaxOutputBytes: 4 << 20, Timeout: contract.DefaultPolicy().Timeout}
	out, code = eng.Execute(context.Background(), "echo x > /workspace/out.txt; cat -n /workspace/out.txt", ops)
	if code != 0 {
		t.Fatalf("expected write in full policy: code=%d out=%q", code, out)
	}
	if !strings.Contains(out, "1:x") {
		t.Fatalf("unexpected cat output: %q", out)
	}
}

func TestEngineProfileGatesCapabilities(t *testing.T) {
	eng := newTestEngine()
	fs := newTestFS()
	ops := contract.OpsFromFilesystem(fs)
	ops.Profile = contract.ProfileCoreStrict

	out, code := eng.Execute(context.Background(), "date +%F", ops)
	if code == 0 {
		t.Fatalf("expected date to be blocked in core-strict profile: out=%q", out)
	}

	ops.Profile = contract.ProfileBashPlus
	out, code = eng.Execute(context.Background(), "date +%F", ops)
	if code != 0 {
		t.Fatalf("expected date to be enabled in bash-plus profile: code=%d out=%q", code, out)
	}
}

func TestEngineExtendedBuiltins(t *testing.T) {
	eng := newTestEngine()
	fs := newTestFS()
	ops := contract.OpsFromFilesystem(fs)
	ops.Policy = contract.ExecutionPolicy{
		WriteMode:        contract.WriteModeFull,
		MaxWriteBytes:    1 << 20,
		MaxPipelineDepth: 16,
		MaxOutputBytes:   4 << 20,
		Timeout:          contract.DefaultPolicy().Timeout,
	}
	ops.Profile = contract.ProfileBashPlus

	out, code := eng.Execute(context.Background(), "head -n 1 /workspace/readme.md", ops)
	if code != 0 || strings.TrimSpace(out) != "hello" {
		t.Fatalf("head failed: code=%d out=%q", code, out)
	}

	out, code = eng.Execute(context.Background(), "tail -n 1 /workspace/readme.md", ops)
	if code != 0 || strings.TrimSpace(out) != "world" {
		t.Fatalf("tail failed: code=%d out=%q", code, out)
	}

	out, code = eng.Execute(context.Background(), "grep hello /workspace/readme.md", ops)
	if code != 0 || !strings.Contains(out, "/workspace/readme.md:1:hello") {
		t.Fatalf("grep failed: code=%d out=%q", code, out)
	}

	out, code = eng.Execute(context.Background(), "find /workspace -name \"*.md\" -exec grep -l hello {} \\;", ops)
	if code != 0 || !strings.Contains(out, "/workspace/readme.md") {
		t.Fatalf("find -exec failed: code=%d out=%q", code, out)
	}

	out, code = eng.Execute(context.Background(), "echo alpha | tee /workspace/new.txt", ops)
	if code != 0 || strings.TrimSpace(out) != "alpha" {
		t.Fatalf("tee failed: code=%d out=%q", code, out)
	}

	out, code = eng.Execute(context.Background(), "sed -i 's/alpha/beta/' /workspace/new.txt; cat -n /workspace/new.txt", ops)
	if code != 0 || !strings.Contains(out, "1:beta") {
		t.Fatalf("sed -i failed: code=%d out=%q", code, out)
	}
}

func TestEngineEnvAndManuals(t *testing.T) {
	eng := newTestEngine()
	fs := newTestFS()
	ops := contract.OpsFromFilesystem(fs)
	ops.Profile = contract.ProfileBashPlus

	out, code := eng.Execute(context.Background(), "env PATH", ops)
	if code != 0 || !strings.Contains(out, "PATH=/sys/bin:/bin") {
		t.Fatalf("env PATH failed: code=%d out=%q", code, out)
	}

	out, code = eng.Execute(context.Background(), "man ls", ops)
	if code != 0 || !strings.Contains(out, "ls [-a] [-R] [-l]") {
		t.Fatalf("man ls failed: code=%d out=%q", code, out)
	}

	out, code = eng.Execute(context.Background(), "man report_tool", ops)
	if code != 0 || !strings.Contains(out, "report tool manual") {
		t.Fatalf("man external failed: code=%d out=%q", code, out)
	}
}

func TestEngineRegressionMount(t *testing.T) {
	eng := newTestEngine()
	root := t.TempDir()
	m, err := mount.NewBaselineCorpusMount()
	if err != nil {
		t.Fatalf("build baseline mount failed: %v", err)
	}
	ops, err := localfs.NewOps(localfs.Options{
		RootDir:       root,
		Profile:       contract.ProfileCoreStrict,
		Policy:        contract.DefaultPolicy(),
		VirtualMounts: []contract.VirtualMount{m},
	})
	if err != nil {
		t.Fatalf("build ops failed: %v", err)
	}

	out, code := eng.Execute(context.Background(), "ls /test", ops)
	if code != 0 {
		t.Fatalf("ls /test failed: code=%d out=%q", code, out)
	}
	if !strings.Contains(out, "/test/core-strict") {
		t.Fatalf("missing profile dir in /test listing: %q", out)
	}

	out, code = eng.Execute(context.Background(), "cat -n /test/core-strict/cases/echo-basic.sh", ops)
	if code != 0 {
		t.Fatalf("cat mounted case failed: code=%d out=%q", code, out)
	}
	if !strings.Contains(out, "1:echo hello") {
		t.Fatalf("unexpected mounted script content: %q", out)
	}

	// Ensure local absolute root constraints still apply while mount is active.
	abs := filepath.ToSlash(filepath.Join(root, "local.txt"))
	out, code = eng.Execute(context.Background(), "echo x > "+abs, ops)
	if code == 0 {
		t.Fatalf("expected read-only policy write block, got success: out=%q", out)
	}
}

// --- helpers for new tests ---

func readOnlyOps(fs contract.Filesystem) contract.Ops {
	ops := contract.OpsFromFilesystem(fs)
	ops.Policy = contract.DefaultPolicy() // read-only
	return ops
}

func writableOps(fs contract.Filesystem) contract.Ops {
	ops := contract.OpsFromFilesystem(fs)
	ops.Policy = contract.ExecutionPolicy{
		WriteMode:        contract.WriteModeFull,
		MaxWriteBytes:    1 << 20,
		MaxPipelineDepth: 16,
		MaxOutputBytes:   4 << 20,
		Timeout:          contract.DefaultPolicy().Timeout,
	}
	return ops
}

func writeLimitedOps(fs contract.Filesystem, maxBytes int) contract.Ops {
	ops := contract.OpsFromFilesystem(fs)
	ops.Policy = contract.ExecutionPolicy{
		WriteMode:        contract.WriteModeWriteLimited,
		MaxWriteBytes:    maxBytes,
		MaxPipelineDepth: 16,
		MaxOutputBytes:   4 << 20,
		Timeout:          contract.DefaultPolicy().Timeout,
	}
	return ops
}

func TestExecutePreparedMatchesExecute(t *testing.T) {
	eng := newTestEngine()
	ops := readOnlyOps(newTestFS())
	prepared, err := eng.PrepareOps(context.Background(), ops)
	if err != nil {
		t.Fatalf("prepare ops failed: %v", err)
	}

	for _, cmd := range []string{
		"echo hello",
		"ll /sys",
		"man ls",
		"grep hello /workspace/readme.md",
	} {
		wantOut, wantCode := eng.Execute(context.Background(), cmd, ops)
		gotOut, gotCode := eng.ExecutePrepared(context.Background(), cmd, prepared)
		if gotCode != wantCode {
			t.Fatalf("command %q: code=%d want=%d", cmd, gotCode, wantCode)
		}
		if gotOut != wantOut {
			t.Fatalf("command %q: output mismatch\ngot:  %q\nwant: %q", cmd, gotOut, wantOut)
		}
	}
}

func TestExecutePreparedAllocReduction(t *testing.T) {
	eng := newTestEngine()
	ops := readOnlyOps(newTestFS())
	// Force non-cacheable Execute path so this test continues to measure
	// PrepareOps overhead against ExecutePrepared.
	ops.CommandAliases = map[string][]string{"noop": {"echo"}}
	prepared, err := eng.PrepareOps(context.Background(), ops)
	if err != nil {
		t.Fatalf("prepare ops failed: %v", err)
	}

	command := "echo hello"
	baseAllocs := testing.AllocsPerRun(1000, func() {
		out, code := eng.Execute(context.Background(), command, ops)
		if code != 0 || out != "hello" {
			t.Fatalf("execute failed: code=%d out=%q", code, out)
		}
	})
	preparedAllocs := testing.AllocsPerRun(1000, func() {
		out, code := eng.ExecutePrepared(context.Background(), command, prepared)
		if code != 0 || out != "hello" {
			t.Fatalf("execute prepared failed: code=%d out=%q", code, out)
		}
	})

	if preparedAllocs >= baseAllocs {
		t.Fatalf("expected prepared execution to allocate less: prepared=%.2f base=%.2f", preparedAllocs, baseAllocs)
	}
}

func TestExecuteCachesPreparedOpsForStableOps(t *testing.T) {
	eng := newTestEngine()
	ops := readOnlyOps(newTestFS())

	// Prime Execute-level cache.
	_, _ = eng.Execute(context.Background(), "echo hello", ops)

	warmAllocs := testing.AllocsPerRun(1000, func() {
		out, code := eng.Execute(context.Background(), "echo hello", ops)
		if code != 0 || out != "hello" {
			t.Fatalf("warm execute failed: code=%d out=%q", code, out)
		}
	})
	prepared, err := eng.PrepareOps(context.Background(), ops)
	if err != nil {
		t.Fatalf("prepare ops failed: %v", err)
	}
	preparedAllocs := testing.AllocsPerRun(1000, func() {
		out, code := eng.ExecutePrepared(context.Background(), "echo hello", prepared)
		if code != 0 || out != "hello" {
			t.Fatalf("prepared execute failed: code=%d out=%q", code, out)
		}
	})

	if warmAllocs > preparedAllocs+1 {
		t.Fatalf("expected warm execute to be close to prepared path: warm=%.2f prepared=%.2f", warmAllocs, preparedAllocs)
	}
}

func TestExecuteCacheIsolationAcrossOpsInstances(t *testing.T) {
	eng := newTestEngine()
	fsA := newTestFS()
	fsB := newTestFS()
	fsB.mustWrite("/workspace/readme.md", "second fs\n")
	opsA := readOnlyOps(fsA)
	opsB := readOnlyOps(fsB)

	outA, codeA := eng.Execute(context.Background(), "cat /workspace/readme.md", opsA)
	if codeA != 0 {
		t.Fatalf("execute with opsA failed: code=%d out=%q", codeA, outA)
	}
	outB, codeB := eng.Execute(context.Background(), "cat /workspace/readme.md", opsB)
	if codeB != 0 {
		t.Fatalf("execute with opsB failed: code=%d out=%q", codeB, outB)
	}
	if outA == outB {
		t.Fatalf("expected cache isolation across ops instances, both outputs=%q", outA)
	}
	if strings.TrimSpace(outB) != "second fs" {
		t.Fatalf("unexpected output from opsB: %q", outB)
	}
}

// ==================== Security Tests ====================

func TestSecurityTeeWritePolicy(t *testing.T) {
	eng := newTestEngine()
	fs := newTestFS()

	t.Run("read-only rejects tee", func(t *testing.T) {
		ops := readOnlyOps(fs)
		out, code := eng.Execute(context.Background(), "echo hello | tee /workspace/out.txt", ops)
		if code == 0 {
			t.Fatalf("expected tee to fail in read-only: out=%q", out)
		}
		if !strings.Contains(out, "write is not allowed") {
			t.Fatalf("expected policy error: out=%q", out)
		}
	})

	t.Run("writable allows tee", func(t *testing.T) {
		ops := writableOps(fs)
		out, code := eng.Execute(context.Background(), "echo hello | tee /workspace/tee_ok.txt", ops)
		if code != 0 {
			t.Fatalf("expected tee to succeed: code=%d out=%q", code, out)
		}
		if strings.TrimSpace(out) != "hello" {
			t.Fatalf("unexpected tee output: %q", out)
		}
	})

	t.Run("write-limited rejects large tee", func(t *testing.T) {
		ops := writeLimitedOps(fs, 5)
		out, code := eng.Execute(context.Background(), "echo toolongtoolongtoolong | tee /workspace/big.txt", ops)
		if code == 0 {
			t.Fatalf("expected tee to fail with size limit: out=%q", out)
		}
		if !strings.Contains(out, "exceeds write limit") {
			t.Fatalf("expected size limit error: out=%q", out)
		}
	})
}

func TestSecuritySedWritePolicy(t *testing.T) {
	eng := newTestEngine()

	t.Run("read-only rejects sed -i", func(t *testing.T) {
		fs := newTestFS()
		ops := readOnlyOps(fs)
		out, code := eng.Execute(context.Background(), "sed -i 's/hello/bye/' /workspace/readme.md", ops)
		if code == 0 {
			t.Fatalf("expected sed -i to fail in read-only: out=%q", out)
		}
		if !strings.Contains(out, "write is not allowed") {
			t.Fatalf("expected policy error: out=%q", out)
		}
	})

	t.Run("writable allows sed -i", func(t *testing.T) {
		fs := newTestFS()
		ops := writableOps(fs)
		out, code := eng.Execute(context.Background(), "sed -i 's/hello/bye/' /workspace/readme.md", ops)
		if code != 0 {
			t.Fatalf("expected sed -i to succeed: code=%d out=%q", code, out)
		}
	})
}

func TestSecurityStatementCountLimit(t *testing.T) {
	eng := newTestEngine()
	fs := newTestFS()
	ops := readOnlyOps(fs)

	// Build a command with >1024 statements
	parts := make([]string, 1030)
	for i := range parts {
		parts[i] = "echo x"
	}
	cmd := strings.Join(parts, "; ")
	_, code := eng.Execute(context.Background(), cmd, ops)
	if code == 0 {
		t.Fatalf("expected statement count limit to trigger")
	}
}

func TestSecurityExecuteStatementLimit(t *testing.T) {
	eng := newTestEngine()
	fs := newTestFS()
	ops := readOnlyOps(fs)

	// Build a command with >64 statements (the executeStatements limit)
	parts := make([]string, 70)
	for i := range parts {
		parts[i] = "echo x"
	}
	cmd := strings.Join(parts, "; ")
	_, code := eng.Execute(context.Background(), cmd, ops)
	if code == 0 {
		t.Fatalf("expected execute statement limit to trigger")
	}
}

func TestSecurityCheckWriteSize(t *testing.T) {
	t.Run("write-limited within limit", func(t *testing.T) {
		p := contract.ExecutionPolicy{WriteMode: contract.WriteModeWriteLimited, MaxWriteBytes: 100}
		if err := p.CheckWriteSize(50); err != nil {
			t.Fatalf("expected nil: %v", err)
		}
	})
	t.Run("write-limited exceeds limit", func(t *testing.T) {
		p := contract.ExecutionPolicy{WriteMode: contract.WriteModeWriteLimited, MaxWriteBytes: 10}
		if err := p.CheckWriteSize(50); err == nil {
			t.Fatalf("expected error for exceeding limit")
		}
	})
	t.Run("full mode no limit", func(t *testing.T) {
		p := contract.ExecutionPolicy{WriteMode: contract.WriteModeFull, MaxWriteBytes: 10}
		if err := p.CheckWriteSize(50); err != nil {
			t.Fatalf("expected nil in full mode: %v", err)
		}
	})
	t.Run("read-only rejects", func(t *testing.T) {
		p := contract.ExecutionPolicy{WriteMode: contract.WriteModeReadOnly}
		if err := p.CheckWriteSize(1); err == nil {
			t.Fatalf("expected error in read-only mode")
		}
	})
}

// ==================== New Command Tests ====================

func TestCommandMkdir(t *testing.T) {
	eng := newTestEngine()

	t.Run("basic mkdir", func(t *testing.T) {
		fs := newTestFS()
		ops := writableOps(fs)
		_, code := eng.Execute(context.Background(), "mkdir /workspace/newdir", ops)
		if code != 0 {
			t.Fatalf("mkdir failed: code=%d", code)
		}
	})

	t.Run("mkdir -p", func(t *testing.T) {
		fs := newTestFS()
		ops := writableOps(fs)
		_, code := eng.Execute(context.Background(), "mkdir -p /workspace/a/b/c", ops)
		if code != 0 {
			t.Fatalf("mkdir -p failed: code=%d", code)
		}
	})

	t.Run("mkdir read-only rejected", func(t *testing.T) {
		fs := newTestFS()
		ops := readOnlyOps(fs)
		_, code := eng.Execute(context.Background(), "mkdir /workspace/nope", ops)
		if code == 0 {
			t.Fatalf("expected mkdir to fail in read-only")
		}
	})

	t.Run("mkdir no args", func(t *testing.T) {
		fs := newTestFS()
		ops := writableOps(fs)
		_, code := eng.Execute(context.Background(), "mkdir", ops)
		if code == 0 {
			t.Fatalf("expected mkdir to fail with no args")
		}
	})
}

func TestCommandCp(t *testing.T) {
	eng := newTestEngine()

	t.Run("basic copy", func(t *testing.T) {
		fs := newTestFS()
		ops := writableOps(fs)
		_, code := eng.Execute(context.Background(), "cp /workspace/readme.md /workspace/copy.md", ops)
		if code != 0 {
			t.Fatalf("cp failed: code=%d", code)
		}
		out, code := eng.Execute(context.Background(), "cat /workspace/copy.md", ops)
		if code != 0 || !strings.Contains(out, "hello") {
			t.Fatalf("copied content mismatch: code=%d out=%q", code, out)
		}
	})

	t.Run("cp nonexistent", func(t *testing.T) {
		fs := newTestFS()
		ops := writableOps(fs)
		_, code := eng.Execute(context.Background(), "cp /workspace/nope.txt /workspace/x.txt", ops)
		if code == 0 {
			t.Fatalf("expected cp to fail for nonexistent source")
		}
	})

	t.Run("cp read-only rejected", func(t *testing.T) {
		fs := newTestFS()
		ops := readOnlyOps(fs)
		_, code := eng.Execute(context.Background(), "cp /workspace/readme.md /workspace/copy.md", ops)
		if code == 0 {
			t.Fatalf("expected cp to fail in read-only")
		}
	})
}

func TestCommandMv(t *testing.T) {
	eng := newTestEngine()

	t.Run("basic move", func(t *testing.T) {
		fs := newTestFS()
		ops := writableOps(fs)
		_, code := eng.Execute(context.Background(), "mv /workspace/todo.txt /workspace/done.txt", ops)
		if code != 0 {
			t.Fatalf("mv failed: code=%d", code)
		}
		// source should be gone
		_, code = eng.Execute(context.Background(), "cat /workspace/todo.txt", ops)
		if code == 0 {
			t.Fatalf("expected source to be removed after mv")
		}
		// dest should have content
		out, code := eng.Execute(context.Background(), "cat /workspace/done.txt", ops)
		if code != 0 || !strings.Contains(out, "todo item") {
			t.Fatalf("moved content mismatch: code=%d out=%q", code, out)
		}
	})

	t.Run("mv read-only rejected", func(t *testing.T) {
		fs := newTestFS()
		ops := readOnlyOps(fs)
		_, code := eng.Execute(context.Background(), "mv /workspace/todo.txt /workspace/done.txt", ops)
		if code == 0 {
			t.Fatalf("expected mv to fail in read-only")
		}
	})
}

func TestCommandRm(t *testing.T) {
	eng := newTestEngine()

	t.Run("basic rm", func(t *testing.T) {
		fs := newTestFS()
		ops := writableOps(fs)
		_, code := eng.Execute(context.Background(), "rm /workspace/todo.txt", ops)
		if code != 0 {
			t.Fatalf("rm failed: code=%d", code)
		}
		_, code = eng.Execute(context.Background(), "cat /workspace/todo.txt", ops)
		if code == 0 {
			t.Fatalf("expected file to be removed")
		}
	})

	t.Run("rm nonexistent", func(t *testing.T) {
		fs := newTestFS()
		ops := writableOps(fs)
		_, code := eng.Execute(context.Background(), "rm /workspace/nope.txt", ops)
		if code == 0 {
			t.Fatalf("expected rm to fail for nonexistent")
		}
	})

	t.Run("rm read-only rejected", func(t *testing.T) {
		fs := newTestFS()
		ops := readOnlyOps(fs)
		_, code := eng.Execute(context.Background(), "rm /workspace/todo.txt", ops)
		if code == 0 {
			t.Fatalf("expected rm to fail in read-only")
		}
	})
}

func TestCommandTouch(t *testing.T) {
	eng := newTestEngine()

	t.Run("touch new file", func(t *testing.T) {
		fs := newTestFS()
		ops := writableOps(fs)
		_, code := eng.Execute(context.Background(), "touch /workspace/new.txt", ops)
		if code != 0 {
			t.Fatalf("touch failed: code=%d", code)
		}
		out, code := eng.Execute(context.Background(), "cat /workspace/new.txt", ops)
		if code != 0 {
			t.Fatalf("expected touched file to exist: code=%d out=%q", code, out)
		}
	})

	t.Run("touch read-only rejected", func(t *testing.T) {
		fs := newTestFS()
		ops := readOnlyOps(fs)
		_, code := eng.Execute(context.Background(), "touch /workspace/new.txt", ops)
		if code == 0 {
			t.Fatalf("expected touch to fail in read-only")
		}
	})
}

func TestCommandWc(t *testing.T) {
	eng := newTestEngine()

	t.Run("wc -l file", func(t *testing.T) {
		fs := newTestFS()
		ops := readOnlyOps(fs)
		out, code := eng.Execute(context.Background(), "wc -l /workspace/readme.md", ops)
		if code != 0 {
			t.Fatalf("wc -l failed: code=%d out=%q", code, out)
		}
		if !strings.Contains(out, "2") {
			t.Fatalf("expected 2 lines: out=%q", out)
		}
	})

	t.Run("wc stdin pipe", func(t *testing.T) {
		fs := newTestFS()
		ops := readOnlyOps(fs)
		out, code := eng.Execute(context.Background(), "echo hello | wc -c", ops)
		if code != 0 {
			t.Fatalf("wc -c pipe failed: code=%d out=%q", code, out)
		}
		if !strings.Contains(out, "5") {
			t.Fatalf("expected 5 bytes: out=%q", out)
		}
	})
}

func TestCommandSort(t *testing.T) {
	eng := newTestEngine()
	fs := newTestFS()
	fs.mustWrite("/workspace/unsorted.txt", "cherry\napple\nbanana\n")

	t.Run("basic sort", func(t *testing.T) {
		ops := readOnlyOps(fs)
		out, code := eng.Execute(context.Background(), "sort /workspace/unsorted.txt", ops)
		if code != 0 {
			t.Fatalf("sort failed: code=%d out=%q", code, out)
		}
		lines := strings.Split(strings.TrimSpace(out), "\n")
		if len(lines) != 3 || lines[0] != "apple" || lines[1] != "banana" || lines[2] != "cherry" {
			t.Fatalf("unexpected sort output: %q", out)
		}
	})

	t.Run("sort -r", func(t *testing.T) {
		ops := readOnlyOps(fs)
		out, code := eng.Execute(context.Background(), "sort -r /workspace/unsorted.txt", ops)
		if code != 0 {
			t.Fatalf("sort -r failed: code=%d out=%q", code, out)
		}
		lines := strings.Split(strings.TrimSpace(out), "\n")
		if len(lines) != 3 || lines[0] != "cherry" {
			t.Fatalf("unexpected sort -r output: %q", out)
		}
	})
}

func TestCommandUniq(t *testing.T) {
	eng := newTestEngine()
	fs := newTestFS()
	fs.mustWrite("/workspace/dupes.txt", "a\na\nb\nb\nb\nc\n")

	t.Run("basic uniq", func(t *testing.T) {
		ops := readOnlyOps(fs)
		out, code := eng.Execute(context.Background(), "uniq /workspace/dupes.txt", ops)
		if code != 0 {
			t.Fatalf("uniq failed: code=%d out=%q", code, out)
		}
		lines := strings.Split(strings.TrimSpace(out), "\n")
		if len(lines) != 3 || lines[0] != "a" || lines[1] != "b" || lines[2] != "c" {
			t.Fatalf("unexpected uniq output: %q", out)
		}
	})

	t.Run("uniq -c", func(t *testing.T) {
		ops := readOnlyOps(fs)
		out, code := eng.Execute(context.Background(), "uniq -c /workspace/dupes.txt", ops)
		if code != 0 {
			t.Fatalf("uniq -c failed: code=%d out=%q", code, out)
		}
		if !strings.Contains(out, "2") && !strings.Contains(out, "3") {
			t.Fatalf("expected counts in output: %q", out)
		}
	})
}

func TestCommandDiff(t *testing.T) {
	eng := newTestEngine()
	fs := newTestFS()
	fs.mustWrite("/workspace/a.txt", "line1\nline2\n")
	fs.mustWrite("/workspace/b.txt", "line1\nline2\n")
	fs.mustWrite("/workspace/c.txt", "line1\nchanged\n")

	t.Run("identical files", func(t *testing.T) {
		ops := readOnlyOps(fs)
		_, code := eng.Execute(context.Background(), "diff /workspace/a.txt /workspace/b.txt", ops)
		if code != 0 {
			t.Fatalf("expected exit 0 for identical files: code=%d", code)
		}
	})

	t.Run("different files", func(t *testing.T) {
		ops := readOnlyOps(fs)
		out, code := eng.Execute(context.Background(), "diff /workspace/a.txt /workspace/c.txt", ops)
		if code == 0 {
			t.Fatalf("expected exit 1 for different files")
		}
		if !strings.Contains(out, "line2") || !strings.Contains(out, "changed") {
			t.Fatalf("expected diff content: out=%q", out)
		}
	})

	t.Run("nonexistent file", func(t *testing.T) {
		ops := readOnlyOps(fs)
		_, code := eng.Execute(context.Background(), "diff /workspace/a.txt /workspace/nope.txt", ops)
		if code == 0 {
			t.Fatalf("expected error for nonexistent file")
		}
	})
}

// ==================== Man System Tests ====================

func TestManSummaryMode(t *testing.T) {
	eng := newTestEngine()
	fs := newTestFS()
	ops := readOnlyOps(fs)

	out, code := eng.Execute(context.Background(), "man ls", ops)
	if code != 0 {
		t.Fatalf("man ls failed: code=%d out=%q", code, out)
	}
	// Should contain synopsis
	if !strings.Contains(out, "ls") {
		t.Fatalf("expected synopsis in man ls: %q", out)
	}
	// Should contain tips
	if !strings.Contains(out, "Tips:") {
		t.Fatalf("expected tips in man ls: %q", out)
	}
	// Should contain examples
	if !strings.Contains(out, "Examples:") {
		t.Fatalf("expected examples in man ls: %q", out)
	}
}

func TestManVerboseMode(t *testing.T) {
	eng := newTestEngine()
	fs := newTestFS()
	ops := readOnlyOps(fs)

	for _, cmd := range []string{"ls", "cp", "rm", "wc"} {
		out, code := eng.Execute(context.Background(), "man -v "+cmd, ops)
		if code != 0 {
			t.Fatalf("man -v %s failed: code=%d out=%q", cmd, code, out)
		}
		// Should contain full markdown headers
		if !strings.Contains(out, "SYNOPSIS") || !strings.Contains(out, "DESCRIPTION") {
			t.Fatalf("expected full markdown in man -v %s: %q", cmd, out)
		}
	}
}

func TestManListMode(t *testing.T) {
	eng := newTestEngine()
	fs := newTestFS()
	ops := readOnlyOps(fs)

	out, code := eng.Execute(context.Background(), "man --list", ops)
	if code != 0 {
		t.Fatalf("man --list failed: code=%d out=%q", code, out)
	}
	for _, cmd := range []string{"cat", "ls", "grep", "mkdir", "wc", "sort", "diff"} {
		if !strings.Contains(out, cmd) {
			t.Fatalf("expected %q in man --list output: %q", cmd, out)
		}
	}
}

func TestManNotFound(t *testing.T) {
	eng := newTestEngine()
	fs := newTestFS()
	ops := readOnlyOps(fs)

	out, code := eng.Execute(context.Background(), "man nonexistent_cmd", ops)
	if code == 0 {
		t.Fatalf("expected man to fail for nonexistent: out=%q", out)
	}
	if !strings.Contains(out, "not found") {
		t.Fatalf("expected 'not found' error: out=%q", out)
	}
}

// ==================== Behavior Fix Tests ====================

func TestGrepExitCodeNoMatch(t *testing.T) {
	eng := newTestEngine()
	fs := newTestFS()
	ops := readOnlyOps(fs)

	t.Run("match returns 0", func(t *testing.T) {
		_, code := eng.Execute(context.Background(), "grep hello /workspace/readme.md", ops)
		if code != 0 {
			t.Fatalf("expected exit 0 for match: code=%d", code)
		}
	})

	t.Run("no match returns 1", func(t *testing.T) {
		_, code := eng.Execute(context.Background(), "grep zzzznotfound /workspace/readme.md", ops)
		if code != 1 {
			t.Fatalf("expected exit 1 for no match: code=%d", code)
		}
	})
}

func TestCatLineNumbers(t *testing.T) {
	eng := newTestEngine()
	fs := newTestFS()
	ops := readOnlyOps(fs)

	t.Run("default no line numbers", func(t *testing.T) {
		out, code := eng.Execute(context.Background(), "cat /workspace/readme.md", ops)
		if code != 0 {
			t.Fatalf("cat failed: code=%d", code)
		}
		if strings.Contains(out, "1:") {
			t.Fatalf("expected no line numbers by default: %q", out)
		}
		if !strings.Contains(out, "hello") {
			t.Fatalf("expected content: %q", out)
		}
	})

	t.Run("-n adds line numbers", func(t *testing.T) {
		out, code := eng.Execute(context.Background(), "cat -n /workspace/readme.md", ops)
		if code != 0 {
			t.Fatalf("cat -n failed: code=%d", code)
		}
		if !strings.Contains(out, "1:hello") {
			t.Fatalf("expected line numbers: %q", out)
		}
	})

	t.Run("stdin passthrough no line numbers", func(t *testing.T) {
		out, code := eng.Execute(context.Background(), "echo test | cat", ops)
		if code != 0 {
			t.Fatalf("cat stdin failed: code=%d", code)
		}
		if strings.TrimSpace(out) != "test" {
			t.Fatalf("unexpected stdin output: %q", out)
		}
	})
}

// ==================== Integration/Pipeline Tests ====================

func TestPipelineIntegration(t *testing.T) {
	eng := newTestEngine()

	t.Run("find pipe wc", func(t *testing.T) {
		fs := newTestFS()
		ops := readOnlyOps(fs)
		out, code := eng.Execute(context.Background(), "find /workspace -name \"*.md\" | wc -l", ops)
		if code != 0 {
			t.Fatalf("find | wc -l failed: code=%d out=%q", code, out)
		}
		if !strings.Contains(out, "1") {
			t.Fatalf("expected 1 md file: out=%q", out)
		}
	})

	t.Run("cat pipe sort", func(t *testing.T) {
		fs := newTestFS()
		ops := readOnlyOps(fs)
		out, code := eng.Execute(context.Background(), "cat /workspace/readme.md | sort", ops)
		if code != 0 {
			t.Fatalf("cat | sort failed: code=%d out=%q", code, out)
		}
		lines := strings.Split(strings.TrimSpace(out), "\n")
		if len(lines) < 2 {
			t.Fatalf("expected sorted lines: %q", out)
		}
	})

	t.Run("cp then diff identical", func(t *testing.T) {
		fs := newTestFS()
		ops := writableOps(fs)
		_, code := eng.Execute(context.Background(), "cp /workspace/readme.md /workspace/backup.md", ops)
		if code != 0 {
			t.Fatalf("cp failed: code=%d", code)
		}
		_, code = eng.Execute(context.Background(), "diff /workspace/readme.md /workspace/backup.md", ops)
		if code != 0 {
			t.Fatalf("expected identical diff: code=%d", code)
		}
	})

	t.Run("echo tee wc pipeline", func(t *testing.T) {
		fs := newTestFS()
		ops := writableOps(fs)
		out, code := eng.Execute(context.Background(), "echo data | tee /workspace/pipe.txt | wc -c", ops)
		if code != 0 {
			t.Fatalf("echo | tee | wc failed: code=%d out=%q", code, out)
		}
		if !strings.Contains(out, "4") {
			t.Fatalf("expected 4 bytes: out=%q", out)
		}
	})

	t.Run("grep conditional chain", func(t *testing.T) {
		fs := newTestFS()
		ops := readOnlyOps(fs)
		out, code := eng.Execute(context.Background(), "grep hello /workspace/readme.md && echo found", ops)
		if code != 0 {
			t.Fatalf("grep && echo failed: code=%d out=%q", code, out)
		}
		if strings.TrimSpace(out) != "found" {
			t.Fatalf("expected 'found': out=%q", out)
		}
	})

	t.Run("grep no match stops chain", func(t *testing.T) {
		fs := newTestFS()
		ops := readOnlyOps(fs)
		_, code := eng.Execute(context.Background(), "grep zzzzz /workspace/readme.md && echo found", ops)
		if code == 0 {
			t.Fatalf("expected chain to stop on grep no-match")
		}
	})
}

// ==================== Embed Manual Tests ====================

func TestEmbedManualLoading(t *testing.T) {
	for _, name := range []string{"ls", "tree", "pwd", "env", "frontmatter", "cat", "grep", "find", "which", "type", "echo", "sed", "tee", "head", "tail", "man", "date", "mkdir", "cp", "mv", "rm", "rmdir", "touch", "wc", "sort", "uniq", "diff"} {
		manual := builtin.LoadEmbeddedManual(name)
		if manual == "" {
			t.Errorf("missing embedded manual for %q", name)
		}
		if !strings.Contains(manual, "SYNOPSIS") {
			t.Errorf("manual for %q missing SYNOPSIS section", name)
		}
	}
}

func TestExamplesForAllCommands(t *testing.T) {
	for _, name := range []string{"ls", "tree", "pwd", "env", "frontmatter", "cat", "grep", "find", "which", "type", "echo", "sed", "tee", "man", "mkdir", "cp", "mv", "rm", "rmdir", "touch", "wc", "sort", "uniq", "diff"} {
		examples := builtin.ExamplesFor(name)
		if len(examples) == 0 {
			t.Errorf("missing examples for %q", name)
		}
	}
}
