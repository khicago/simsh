package engine

import (
	"context"
	"errors"
	"fmt"
	"path"
	"sort"
	"strings"
	"time"

	"github.com/khicago/simsh/pkg/contract"
	"github.com/khicago/simsh/pkg/mount"
)

// Engine executes shell-like scripts using a command registry and injected ops.
type Engine struct {
	registry *Registry
}

type contextKey string

const (
	dispatchDepthKey contextKey = "simsh.dispatch.depth"
	maxDispatchDepth            = 8
)

func New(registry *Registry) *Engine {
	if registry == nil {
		registry = NewRegistry()
	}
	return &Engine{registry: registry}
}

func (e *Engine) Registry() *Registry {
	return e.registry
}

func (e *Engine) BuiltinCommandDocs() []contract.BuiltinCommandDoc {
	return e.registry.BuiltinCommandDocs()
}

func (e *Engine) Execute(ctx context.Context, cmdline string, ops contract.Ops) (string, int) {
	normalized, err := e.normalizeOps(ops)
	if err != nil {
		return fmt.Sprintf("execute: %v", err), contract.ExitCodeGeneral
	}
	if normalized.Policy.Timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, normalized.Policy.Timeout)
		defer cancel()
	}
	return e.runScript(ctx, cmdline, normalized)
}

func (e *Engine) ExecuteWithFilesystem(ctx context.Context, cmdline string, fs contract.Filesystem) (string, int) {
	if fs == nil {
		return "execute: filesystem is required", contract.ExitCodeGeneral
	}
	return e.Execute(ctx, cmdline, contract.OpsFromFilesystem(fs))
}

func (e *Engine) normalizeOps(ops contract.Ops) (contract.Ops, error) {
	if strings.TrimSpace(ops.RootDir) == "" {
		ops.RootDir = "/"
	}
	if ops.RequireAbsolutePath == nil {
		return ops, fmt.Errorf("require absolute path function is required")
	}
	if ops.ListChildren == nil {
		return ops, fmt.Errorf("list children function is required")
	}
	if ops.IsDirPath == nil {
		return ops, fmt.Errorf("is-dir function is required")
	}
	if ops.ReadRawContent == nil {
		return ops, fmt.Errorf("read file function is required")
	}
	if ops.ResolveSearchPaths == nil {
		return ops, fmt.Errorf("resolve search paths function is required")
	}
	if ops.CollectFilesUnder == nil {
		return ops, fmt.Errorf("collect files function is required")
	}
	if ops.WriteFile == nil {
		ops.WriteFile = func(context.Context, string, string) error { return contract.ErrUnsupported }
	}
	if ops.AppendFile == nil {
		ops.AppendFile = func(context.Context, string, string) error { return contract.ErrUnsupported }
	}
	if ops.EditFile == nil {
		ops.EditFile = func(context.Context, string, string, string, bool) error { return contract.ErrUnsupported }
	}
	if ops.MakeDir == nil {
		ops.MakeDir = func(context.Context, string) error { return contract.ErrUnsupported }
	}
	if ops.RemoveFile == nil {
		ops.RemoveFile = func(context.Context, string) error { return contract.ErrUnsupported }
	}
	if ops.CheckPathOp == nil {
		ops.CheckPathOp = func(context.Context, contract.PathOp, string) error { return nil }
	}
	if ops.Policy.WriteMode == "" {
		ops.Policy = contract.DefaultPolicy()
	}
	if ops.Profile == "" {
		ops.Profile = contract.DefaultProfile()
	}
	if _, err := contract.ParseProfile(string(ops.Profile)); err != nil {
		return ops, err
	}
	withMounts, err := e.withMountRouter(ops)
	if err != nil {
		return ops, err
	}
	return withPathAccessPolicy(withMounts), nil
}

func (e *Engine) withMountRouter(ops contract.Ops) (contract.Ops, error) {
	allMounts := make([]contract.VirtualMount, 0, len(ops.VirtualMounts)+2)
	allMounts = append(allMounts, mount.NewSysBinMount(e.registry))
	for _, m := range ops.VirtualMounts {
		if m != nil {
			allMounts = append(allMounts, m)
		}
	}
	if ops.ListExternalCommands != nil {
		allMounts = append(allMounts, mount.NewExternalBinMount(ops.ListExternalCommands, ops.ReadExternalManual))
	}
	router, err := newMountRouter(allMounts)
	if err != nil {
		return ops, err
	}
	if router.isEmpty() {
		return ops, nil
	}
	return router.wrapOps(ops), nil
}

func normalizeAbsolutePath(raw string) string {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return "/"
	}
	cleaned := path.Clean(trimmed)
	if !strings.HasPrefix(cleaned, "/") {
		cleaned = "/" + cleaned
	}
	return cleaned
}

func isVirtualBinExecutable(pathValue string) bool {
	pathValue = normalizeAbsolutePath(pathValue)
	if !strings.HasPrefix(pathValue, contract.VirtualExternalBinDir+"/") {
		return false
	}
	name := strings.TrimPrefix(pathValue, contract.VirtualExternalBinDir+"/")
	return strings.TrimSpace(name) != "" && !strings.Contains(name, "/")
}

func isVirtualSysBinExecutable(pathValue string) bool {
	pathValue = normalizeAbsolutePath(pathValue)
	if !strings.HasPrefix(pathValue, contract.VirtualSystemBinDir+"/") {
		return false
	}
	name := strings.TrimPrefix(pathValue, contract.VirtualSystemBinDir+"/")
	return strings.TrimSpace(name) != "" && !strings.Contains(name, "/")
}

func commandNameFromSysBinPath(pathValue string) string {
	pathValue = normalizeAbsolutePath(pathValue)
	if !isVirtualSysBinExecutable(pathValue) {
		return ""
	}
	return strings.TrimPrefix(pathValue, contract.VirtualSystemBinDir+"/")
}

func normalizeBuiltinCommandName(raw string) string {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return ""
	}
	if isVirtualSysBinExecutable(trimmed) {
		return commandNameFromSysBinPath(trimmed)
	}
	return trimmed
}

func normalizeExternalCommandName(raw string) string {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return ""
	}
	if strings.HasPrefix(trimmed, contract.VirtualExternalBinDir+"/") {
		trimmed = strings.TrimPrefix(trimmed, contract.VirtualExternalBinDir+"/")
	}
	trimmed = strings.TrimPrefix(trimmed, "/")
	if strings.Contains(trimmed, "/") {
		return ""
	}
	return trimmed
}

func commandNameFromPath(pathValue string) string {
	pathValue = normalizeAbsolutePath(pathValue)
	if !isVirtualBinExecutable(pathValue) {
		return ""
	}
	return strings.TrimPrefix(pathValue, contract.VirtualExternalBinDir+"/")
}

func listExternalCommandPaths(ctx context.Context, lister func(context.Context) ([]contract.ExternalCommand, error)) ([]string, error) {
	if lister == nil {
		return nil, nil
	}
	commands, err := lister(ctx)
	if err != nil {
		if errors.Is(err, contract.ErrUnsupported) {
			return nil, nil
		}
		return nil, err
	}
	out := make([]string, 0, len(commands))
	seen := map[string]struct{}{}
	for _, command := range commands {
		name := normalizeExternalCommandName(command.Name)
		if name == "" {
			continue
		}
		pathValue := contract.VirtualExternalBinDir + "/" + name
		if _, exists := seen[pathValue]; exists {
			continue
		}
		seen[pathValue] = struct{}{}
		out = append(out, pathValue)
	}
	sort.Strings(out)
	return out, nil
}

func lookupExternalCommand(ctx context.Context, name string, lister func(context.Context) ([]contract.ExternalCommand, error)) (contract.ExternalCommand, bool, error) {
	name = normalizeExternalCommandName(name)
	if name == "" || lister == nil {
		return contract.ExternalCommand{}, false, nil
	}
	commands, err := lister(ctx)
	if err != nil {
		if errors.Is(err, contract.ErrUnsupported) {
			return contract.ExternalCommand{}, false, nil
		}
		return contract.ExternalCommand{}, false, err
	}
	for _, command := range commands {
		if normalizeExternalCommandName(command.Name) == name {
			return command, true, nil
		}
	}
	return contract.ExternalCommand{}, false, nil
}

func hasExternalCommandPath(ctx context.Context, pathValue string, lister func(context.Context) ([]contract.ExternalCommand, error)) (bool, error) {
	name := commandNameFromPath(pathValue)
	if strings.TrimSpace(name) == "" {
		return false, nil
	}
	_, found, err := lookupExternalCommand(ctx, name, lister)
	return found, err
}

func appendUniquePath(paths []string, candidate string) []string {
	candidate = normalizeAbsolutePath(candidate)
	for _, pathValue := range paths {
		if normalizeAbsolutePath(pathValue) == candidate {
			return paths
		}
	}
	paths = append(paths, candidate)
	sort.Strings(paths)
	return paths
}

func withPathAccessPolicy(ops contract.Ops) contract.Ops {
	origDescribePath := ops.DescribePath
	ops.DescribePath = func(ctx context.Context, pathValue string) (contract.PathMeta, error) {
		if origDescribePath == nil {
			return contract.PathMeta{}, contract.ErrUnsupported
		}
		meta, err := origDescribePath(ctx, pathValue)
		if err != nil {
			return contract.PathMeta{}, err
		}
		if strings.TrimSpace(meta.Access) == "" {
			meta.Access = contract.PathAccessReadOnly
			if ops.Policy.AllowWrite() {
				meta.Access = contract.PathAccessReadWrite
			}
		}
		meta.Access = contract.NormalizePathAccess(meta.Access)
		meta.Capabilities = contract.NormalizePathCapabilities(meta.Capabilities)
		if !ops.Policy.AllowWrite() || meta.Access == contract.PathAccessReadOnly {
			meta.Access = contract.PathAccessReadOnly
			meta.Capabilities = contract.StripWriteCapabilities(meta.Capabilities)
		}
		return meta, nil
	}
	return ops
}

func isNullDevice(pathValue string) bool {
	return path.Clean(strings.TrimSpace(pathValue)) == "/dev/null"
}

func (e *Engine) runCommand(ctx context.Context, args []string, input string, hasInput bool, ops contract.Ops) (string, int) {
	if len(args) == 0 {
		return "execute: missing command", contract.ExitCodeUsage
	}
	rawCmd := strings.TrimSpace(args[0])
	builtinName := normalizeBuiltinCommandName(rawCmd)
	if spec, ok := e.registry.Lookup(builtinName); ok {
		if !isCommandAllowedByProfile(spec, ops.Profile) {
			return fmt.Sprintf("%s: not supported in profile %s", builtinName, ops.Profile), contract.ExitCodeUnsupported
		}
		emitAudit(ctx, ops, contract.AuditEvent{
			Time:    time.Now(),
			Phase:   contract.AuditPhaseCommandStart,
			Command: builtinName,
			Args:    append([]string(nil), args[1:]...),
		})
		runtime := CommandRuntime{Ctx: ctx, Ops: ops, Stdin: input, HasStdin: hasInput}
		runtime.Dispatch = func(dispatchArgs []string, dispatchInput string, dispatchHasInput bool) (string, int) {
			depth, _ := ctx.Value(dispatchDepthKey).(int)
			if depth >= maxDispatchDepth {
				return fmt.Sprintf("execute: dispatch recursion depth exceeds limit (%d)", maxDispatchDepth), contract.ExitCodeGeneral
			}
			childCtx := context.WithValue(ctx, dispatchDepthKey, depth+1)
			return e.runCommand(childCtx, dispatchArgs, dispatchInput, dispatchHasInput, ops)
		}
		runtime.LookupManual = func(name string) (string, bool) {
			return e.registry.BuiltinManual(name)
		}
		runtime.LookupDetailedManual = func(name string) (string, bool) {
			return e.registry.BuiltinDetailedManual(name)
		}
		runtime.ListBuiltinNames = func() []string {
			return e.registry.ListNames()
		}
		out, code := spec.Run(runtime, args[1:])
		phase := contract.AuditPhaseCommandEnd
		if code != 0 {
			phase = contract.AuditPhaseCommandError
		}
		emitAudit(ctx, ops, contract.AuditEvent{
			Time:     time.Now(),
			Phase:    phase,
			Command:  builtinName,
			Args:     append([]string(nil), args[1:]...),
			ExitCode: code,
			Message:  strings.TrimSpace(out),
		})
		return enforceOutputLimit(out, code, ops.Policy)
	}
	out, code := e.runExternalCommand(ctx, rawCmd, args[1:], input, hasInput, ops)
	return enforceOutputLimit(out, code, ops.Policy)
}

func (e *Engine) runExternalCommand(ctx context.Context, cmd string, args []string, input string, hasInput bool, ops contract.Ops) (string, int) {
	emitAudit(ctx, ops, contract.AuditEvent{
		Time:    time.Now(),
		Phase:   contract.AuditPhaseCommandStart,
		Command: strings.TrimSpace(cmd),
		Args:    append([]string(nil), args...),
	})
	if ops.RunExternalCommand == nil {
		out := fmt.Sprintf("%s: Not supported", cmd)
		emitAudit(ctx, ops, contract.AuditEvent{Time: time.Now(), Phase: contract.AuditPhaseCommandError, Command: cmd, Args: append([]string(nil), args...), ExitCode: contract.ExitCodeUnsupported, Message: out})
		return out, contract.ExitCodeUnsupported
	}
	request := contract.ExternalCommandRequest{
		Command:  normalizeExternalCommandName(cmd),
		RawPath:  strings.TrimSpace(cmd),
		Args:     append([]string(nil), args...),
		Stdin:    input,
		HasStdin: hasInput,
	}
	if request.Command == "" {
		request.Command = strings.TrimSpace(cmd)
	}
	result, err := ops.RunExternalCommand(ctx, request)
	if err != nil {
		if errors.Is(err, contract.ErrUnsupported) {
			out := fmt.Sprintf("%s: Not supported", cmd)
			emitAudit(ctx, ops, contract.AuditEvent{Time: time.Now(), Phase: contract.AuditPhaseCommandError, Command: cmd, Args: append([]string(nil), args...), ExitCode: contract.ExitCodeUnsupported, Message: out})
			return out, contract.ExitCodeUnsupported
		}
		out := fmt.Sprintf("%s: %v", cmd, err)
		emitAudit(ctx, ops, contract.AuditEvent{Time: time.Now(), Phase: contract.AuditPhaseCommandError, Command: cmd, Args: append([]string(nil), args...), ExitCode: contract.ExitCodeGeneral, Message: out})
		return out, contract.ExitCodeGeneral
	}
	if result.ExitCode == 0 {
		emitAudit(ctx, ops, contract.AuditEvent{Time: time.Now(), Phase: contract.AuditPhaseCommandEnd, Command: cmd, Args: append([]string(nil), args...), ExitCode: 0})
		return result.Stdout, 0
	}
	if result.ExitCode < 0 {
		result.ExitCode = contract.ExitCodeGeneral
	}
	if strings.TrimSpace(result.Stdout) == "" {
		result.Stdout = fmt.Sprintf("%s: command failed", cmd)
	}
	emitAudit(ctx, ops, contract.AuditEvent{Time: time.Now(), Phase: contract.AuditPhaseCommandError, Command: cmd, Args: append([]string(nil), args...), ExitCode: result.ExitCode, Message: strings.TrimSpace(result.Stdout)})
	return result.Stdout, result.ExitCode
}

func enforceOutputLimit(out string, code int, policy contract.ExecutionPolicy) (string, int) {
	if policy.MaxOutputBytes > 0 && len(out) > policy.MaxOutputBytes {
		return fmt.Sprintf("execute: output exceeds limit (%d bytes)", policy.MaxOutputBytes), contract.ExitCodeGeneral
	}
	return out, code
}

func emitAudit(ctx context.Context, ops contract.Ops, event contract.AuditEvent) {
	if ops.AuditSink == nil {
		return
	}
	ops.AuditSink.Emit(ctx, event)
}

func isCommandAllowedByProfile(spec CommandSpec, profile contract.CompatibilityProfile) bool {
	if len(spec.Capabilities) == 0 {
		return true
	}
	profileSpec := contract.ProfileByName(profile)
	for _, cap := range spec.Capabilities {
		if strings.TrimSpace(cap) == "" {
			continue
		}
		if !profileSpec.Capabilities[cap] {
			return false
		}
	}
	return true
}
