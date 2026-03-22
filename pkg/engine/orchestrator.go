package engine

import (
	"context"
	"errors"
	"fmt"
	"path"
	"sort"
	"strings"
	"sync"
	"time"
	"unsafe"

	"github.com/khicago/simsh/pkg/contract"
	"github.com/khicago/simsh/pkg/mount"
)

// Engine executes shell-like scripts using a command registry and injected ops.
type Engine struct {
	registry *Registry

	cacheMu     sync.RWMutex
	cacheValid  bool
	cacheKey    preparedCacheKey
	cacheResult PreparedOps
}

// PreparedOps is a compiled execution callback set that can be safely reused
// across many Execute calls to avoid repeated normalize/wrap/bootstrap work.
type PreparedOps struct {
	normalized contract.Ops
	resetState func()
}

type preparedCacheKey struct {
	rootDir    string
	workingDir string
	profile    contract.CompatibilityProfile
	policy     contract.ExecutionPolicy

	requireAbsolutePathID uintptr
	listChildrenID        uintptr
	isDirPathID           uintptr
	describePathID        uintptr
	formatLSLongRowID     uintptr
	readRawContentID      uintptr
	resolveSearchPathsID  uintptr
	collectFilesUnderID   uintptr
	writeFileID           uintptr
	appendFileID          uintptr
	editFileID            uintptr
	makeDirID             uintptr
	removeFileID          uintptr
	removeDirID           uintptr
	checkPathOpID         uintptr

	listExternalCommandsID uintptr
	runExternalCommandID   uintptr
	readExternalManualID   uintptr
	auditSinkID            uintptr

	pathEnvHash uint64
}

type contextKey string

const (
	dispatchDepthKey contextKey = "simsh.dispatch.depth"
	maxDispatchDepth int        = 8
	maxAliasDepth    int        = 8
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

// PrepareOps validates and compiles ops into an execution-ready form.
func (e *Engine) PrepareOps(ctx context.Context, ops contract.Ops) (PreparedOps, error) {
	normalized, resetState, err := e.normalizeOps(ctx, ops)
	if err != nil {
		return PreparedOps{}, err
	}
	return PreparedOps{normalized: normalized, resetState: resetState}, nil
}

// Ops returns the normalized callback set.
func (p PreparedOps) Ops() contract.Ops {
	return p.normalized
}

func (p PreparedOps) ResetMutableState() {
	if p.resetState != nil {
		p.resetState()
	}
}

func (e *Engine) Execute(ctx context.Context, cmdline string, ops contract.Ops) (string, int) {
	result := e.ExecuteResult(ctx, cmdline, ops)
	return result.FlattenOutput(), result.ExitCode
}

// ExecutePrepared runs a command with pre-normalized ops.
func (e *Engine) ExecutePrepared(ctx context.Context, cmdline string, prepared PreparedOps) (string, int) {
	result := e.ExecutePreparedResult(ctx, cmdline, prepared)
	return result.FlattenOutput(), result.ExitCode
}

func (e *Engine) ExecuteResult(ctx context.Context, cmdline string, ops contract.Ops) contract.ExecutionResult {
	if key, ok := buildPreparedCacheKey(ops); ok {
		if cached, found := e.loadPreparedFromCache(key); found {
			cached.ResetMutableState()
			return e.ExecutePreparedResult(ctx, cmdline, cached)
		}
		prepared, err := e.PrepareOps(ctx, ops)
		if err != nil {
			return contract.ExecutionResult{
				ExecutionID: nextExecutionID(),
				ExitCode:    contract.ExitCodeGeneral,
				Stdout:      fmt.Sprintf("execute: %v", err),
				StartedAt:   time.Now().UTC(),
				FinishedAt:  time.Now().UTC(),
			}
		}
		prepared.ResetMutableState()
		e.storePreparedInCache(key, prepared)
		return e.ExecutePreparedResult(ctx, cmdline, prepared)
	}
	prepared, err := e.PrepareOps(ctx, ops)
	if err != nil {
		return contract.ExecutionResult{
			ExecutionID: nextExecutionID(),
			ExitCode:    contract.ExitCodeGeneral,
			Stdout:      fmt.Sprintf("execute: %v", err),
			StartedAt:   time.Now().UTC(),
			FinishedAt:  time.Now().UTC(),
		}
	}
	prepared.ResetMutableState()
	return e.ExecutePreparedResult(ctx, cmdline, prepared)
}

func (e *Engine) ExecuteWithFilesystem(ctx context.Context, cmdline string, fs contract.Filesystem) (string, int) {
	if fs == nil {
		return "execute: filesystem is required", contract.ExitCodeGeneral
	}
	return e.Execute(ctx, cmdline, contract.OpsFromFilesystem(fs))
}

func (e *Engine) loadPreparedFromCache(key preparedCacheKey) (PreparedOps, bool) {
	e.cacheMu.RLock()
	defer e.cacheMu.RUnlock()
	if !e.cacheValid || e.cacheKey != key {
		return PreparedOps{}, false
	}
	return e.cacheResult, true
}

func (e *Engine) storePreparedInCache(key preparedCacheKey, prepared PreparedOps) {
	e.cacheMu.Lock()
	defer e.cacheMu.Unlock()
	e.cacheKey = key
	e.cacheResult = prepared
	e.cacheValid = true
}

func buildPreparedCacheKey(ops contract.Ops) (preparedCacheKey, bool) {
	// Cache is only safe for the stable no-bootstrap/no-overlay path.
	if len(ops.CommandAliases) > 0 || len(ops.EnvVars) > 0 || len(ops.RCFiles) > 0 || len(ops.VirtualMounts) > 0 {
		return preparedCacheKey{}, false
	}
	policy := ops.Policy
	if policy.WriteMode == "" {
		policy = contract.DefaultPolicy()
	}
	profile := ops.Profile
	if profile == "" {
		profile = contract.DefaultProfile()
	}
	if _, err := contract.ParseProfile(string(profile)); err != nil {
		return preparedCacheKey{}, false
	}

	key := preparedCacheKey{
		rootDir:    strings.TrimSpace(ops.RootDir),
		workingDir: strings.TrimSpace(ops.WorkingDir),
		profile:    profile,
		policy:     policy,

		requireAbsolutePathID: funcValueIdentity(ops.RequireAbsolutePath),
		listChildrenID:        funcValueIdentity(ops.ListChildren),
		isDirPathID:           funcValueIdentity(ops.IsDirPath),
		describePathID:        funcValueIdentity(ops.DescribePath),
		formatLSLongRowID:     funcValueIdentity(ops.FormatLSLongRow),
		readRawContentID:      funcValueIdentity(ops.ReadRawContent),
		resolveSearchPathsID:  funcValueIdentity(ops.ResolveSearchPaths),
		collectFilesUnderID:   funcValueIdentity(ops.CollectFilesUnder),
		writeFileID:           funcValueIdentity(ops.WriteFile),
		appendFileID:          funcValueIdentity(ops.AppendFile),
		editFileID:            funcValueIdentity(ops.EditFile),
		makeDirID:             funcValueIdentity(ops.MakeDir),
		removeFileID:          funcValueIdentity(ops.RemoveFile),
		removeDirID:           funcValueIdentity(ops.RemoveDir),
		checkPathOpID:         funcValueIdentity(ops.CheckPathOp),

		listExternalCommandsID: funcValueIdentity(ops.ListExternalCommands),
		runExternalCommandID:   funcValueIdentity(ops.RunExternalCommand),
		readExternalManualID:   funcValueIdentity(ops.ReadExternalManual),
		auditSinkID:            interfaceIdentity(ops.AuditSink),
		pathEnvHash:            stableStringSliceHash(ops.PathEnv),
	}
	if key.rootDir == "" {
		key.rootDir = "/"
	}
	if key.requireAbsolutePathID == 0 || key.listChildrenID == 0 || key.isDirPathID == 0 || key.readRawContentID == 0 || key.resolveSearchPathsID == 0 || key.collectFilesUnderID == 0 {
		return preparedCacheKey{}, false
	}
	return key, true
}

func stableStringSliceHash(values []string) uint64 {
	var h uint64 = 1469598103934665603
	for _, value := range values {
		for idx := 0; idx < len(value); idx++ {
			h ^= uint64(value[idx])
			h *= 1099511628211
		}
		h ^= 0
		h *= 1099511628211
	}
	return h
}

func funcValueIdentity(fn any) uintptr {
	return interfaceIdentity(fn)
}

func interfaceIdentity(v any) uintptr {
	if v == nil {
		return 0
	}
	type eface struct {
		typ  unsafe.Pointer
		data unsafe.Pointer
	}
	return uintptr((*eface)(unsafe.Pointer(&v)).data)
}

func (e *Engine) normalizeOps(ctx context.Context, ops contract.Ops) (contract.Ops, func(), error) {
	if strings.TrimSpace(ops.RootDir) == "" {
		ops.RootDir = "/"
	}
	if strings.TrimSpace(ops.WorkingDir) == "" {
		ops.WorkingDir = ops.RootDir
	}
	if ops.RequireAbsolutePath == nil {
		return ops, nil, fmt.Errorf("require absolute path function is required")
	}
	if ops.ListChildren == nil {
		return ops, nil, fmt.Errorf("list children function is required")
	}
	if ops.IsDirPath == nil {
		return ops, nil, fmt.Errorf("is-dir function is required")
	}
	if ops.ReadRawContent == nil {
		return ops, nil, fmt.Errorf("read file function is required")
	}
	if ops.ResolveSearchPaths == nil {
		return ops, nil, fmt.Errorf("resolve search paths function is required")
	}
	if ops.CollectFilesUnder == nil {
		return ops, nil, fmt.Errorf("collect files function is required")
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
	if ops.RemoveDir == nil {
		ops.RemoveDir = func(context.Context, string) error { return contract.ErrUnsupported }
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
		return ops, nil, err
	}
	ops.CommandAliases = contract.MergeCommandAliases(contract.DefaultCommandAliases(), ops.CommandAliases)
	ops.EnvVars = contract.NormalizeEnvVars(ops.EnvVars)
	ops.RCFiles = contract.NormalizeRCFiles(ops.RCFiles)

	withMounts, err := e.withMountRouter(ops)
	if err != nil {
		return ops, nil, err
	}
	withPathResolution, resetState, err := withPathResolution(ctx, withMounts)
	if err != nil {
		return ops, nil, err
	}
	withRuntimeBootstrap, err := e.withRuntimeBootstrap(ctx, withPathResolution)
	if err != nil {
		return ops, nil, err
	}
	return withPathAccessPolicy(withRuntimeBootstrap), resetState, nil
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

func (e *Engine) withRuntimeBootstrap(ctx context.Context, ops contract.Ops) (contract.Ops, error) {
	paths := contract.NormalizeRCFiles(ops.RCFiles)
	if len(paths) == 0 {
		return ops, nil
	}
	aliases := contract.NormalizeCommandAliases(ops.CommandAliases)
	envVars := contract.NormalizeEnvVars(ops.EnvVars)
	for _, rawPath := range paths {
		pathValue, err := ops.RequireAbsolutePath(rawPath)
		if err != nil {
			return ops, fmt.Errorf("rc: %s: %v", rawPath, err)
		}
		raw, err := ops.ReadRawContent(ctx, pathValue)
		if err != nil {
			if isNotFoundLikeError(err) {
				continue
			}
			return ops, fmt.Errorf("rc: read %s failed: %v", pathValue, err)
		}
		rcAliases, rcEnv, err := parseRCContent(raw)
		if err != nil {
			return ops, fmt.Errorf("rc: parse %s failed: %v", pathValue, err)
		}
		aliases = contract.MergeCommandAliases(aliases, rcAliases)
		for key, value := range rcEnv {
			envVars[key] = value
		}
	}
	ops.CommandAliases = aliases
	ops.EnvVars = envVars
	return ops, nil
}

func parseRCContent(raw string) (map[string][]string, map[string]string, error) {
	aliases := map[string][]string{}
	envVars := map[string]string{}
	raw = strings.TrimSuffix(raw, "\n")
	lines := []string{}
	if raw != "" {
		lines = strings.Split(raw, "\n")
	}
	for idx, line := range lines {
		lineNo := idx + 1
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}

		switch {
		case strings.HasPrefix(trimmed, "export "):
			key, value, err := parseExportLine(strings.TrimSpace(strings.TrimPrefix(trimmed, "export ")))
			if err != nil {
				return nil, nil, fmt.Errorf("line %d: %v", lineNo, err)
			}
			envVars[key] = value
		case strings.HasPrefix(trimmed, "alias "):
			name, expansion, err := parseAliasLine(strings.TrimSpace(strings.TrimPrefix(trimmed, "alias ")))
			if err != nil {
				return nil, nil, fmt.Errorf("line %d: %v", lineNo, err)
			}
			aliases[name] = expansion
		default:
			return nil, nil, fmt.Errorf("line %d: unsupported statement %q", lineNo, trimmed)
		}
	}
	return aliases, envVars, nil
}

func parseExportLine(raw string) (string, string, error) {
	name, value, ok := splitAssignment(raw)
	if !ok {
		return "", "", fmt.Errorf("export requires KEY=VALUE")
	}
	key := normalizeEnvKey(name)
	if key == "" {
		return "", "", fmt.Errorf("invalid export key %q", name)
	}
	return key, trimOptionalQuotes(value), nil
}

func parseAliasLine(raw string) (string, []string, error) {
	name, value, ok := splitAssignment(raw)
	if !ok {
		return "", nil, fmt.Errorf("alias requires NAME=VALUE")
	}
	name = strings.TrimSpace(name)
	if strings.Contains(name, "/") || strings.HasPrefix(name, "-") || strings.Contains(name, " ") || name == "" {
		return "", nil, fmt.Errorf("invalid alias name %q", name)
	}
	expansion, err := parseAliasExpansion(trimOptionalQuotes(value))
	if err != nil {
		return "", nil, err
	}
	return name, expansion, nil
}

func parseAliasExpansion(raw string) ([]string, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return nil, fmt.Errorf("alias expansion must not be empty")
	}
	out := make([]string, 0, 4)
	for idx := 0; idx < len(trimmed); {
		idx = skipInlineSpaces(trimmed, idx)
		if idx >= len(trimmed) {
			break
		}
		word, next, err := readShellWord(trimmed, idx)
		if err != nil {
			return nil, fmt.Errorf("invalid alias expansion: %v", err)
		}
		if strings.TrimSpace(word) == "" {
			return nil, fmt.Errorf("alias expansion must not contain empty token")
		}
		out = append(out, word)
		idx = next
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("alias expansion must not be empty")
	}
	return out, nil
}

func splitAssignment(raw string) (string, string, bool) {
	parts := strings.SplitN(strings.TrimSpace(raw), "=", 2)
	if len(parts) != 2 {
		return "", "", false
	}
	key := strings.TrimSpace(parts[0])
	if key == "" {
		return "", "", false
	}
	return key, strings.TrimSpace(parts[1]), true
}

func trimOptionalQuotes(raw string) string {
	if len(raw) >= 2 {
		if (raw[0] == '"' && raw[len(raw)-1] == '"') || (raw[0] == '\'' && raw[len(raw)-1] == '\'') {
			return raw[1 : len(raw)-1]
		}
	}
	return raw
}

func normalizeEnvKey(raw string) string {
	name := strings.TrimSpace(raw)
	if name == "" {
		return ""
	}
	for idx, r := range name {
		switch {
		case r == '_' || (r >= 'A' && r <= 'Z') || (r >= 'a' && r <= 'z'):
			// allowed
		case idx > 0 && r >= '0' && r <= '9':
			// allowed
		default:
			return ""
		}
	}
	return name
}

func isNotFoundLikeError(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(strings.TrimSpace(err.Error()))
	return strings.Contains(msg, "no such file") || strings.Contains(msg, "not found")
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

func expandCommandAlias(args []string, aliases map[string][]string) ([]string, error) {
	if len(args) == 0 {
		return nil, fmt.Errorf("missing command")
	}
	if len(aliases) == 0 {
		return append([]string(nil), args...), nil
	}
	current := append([]string(nil), args...)
	seen := map[string]struct{}{}
	for depth := 0; depth < maxAliasDepth; depth++ {
		name := strings.TrimSpace(current[0])
		expansion, ok := aliases[name]
		if !ok {
			return current, nil
		}
		if _, exists := seen[name]; exists {
			return nil, fmt.Errorf("alias loop detected at %q", name)
		}
		seen[name] = struct{}{}
		tokens := expansion
		if len(tokens) == 0 {
			return nil, fmt.Errorf("alias %q has empty expansion", name)
		}
		for _, token := range tokens {
			if strings.TrimSpace(token) == "" {
				return nil, fmt.Errorf("alias %q has empty expansion", name)
			}
		}
		next := make([]string, 0, len(tokens)+len(current)-1)
		next = append(next, tokens...)
		next = append(next, current[1:]...)
		current = next
	}
	return nil, fmt.Errorf("alias expansion depth exceeds limit (%d)", maxAliasDepth)
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

func (e *Engine) runCommand(ctx context.Context, args []string, input string, hasInput bool, ops contract.Ops) execOutput {
	if err := executionContextErr(ctx); err != nil {
		return contextExecOutput(err)
	}
	if len(args) == 0 {
		return execOutput{stdout: "execute: missing command", code: contract.ExitCodeUsage}
	}
	expandedArgs, err := expandCommandAlias(args, ops.CommandAliases)
	if err != nil {
		return execOutput{stdout: fmt.Sprintf("execute: %v", err), code: contract.ExitCodeUsage}
	}
	args = expandedArgs
	rawCmd := strings.TrimSpace(args[0])
	builtinName := normalizeBuiltinCommandName(rawCmd)
	if spec, ok := e.registry.Lookup(builtinName); ok {
		if !isCommandAllowedByProfile(spec, ops.Profile) {
			return execOutput{stdout: fmt.Sprintf("%s: not supported in profile %s", builtinName, ops.Profile), code: contract.ExitCodeUnsupported}
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
			result := e.runCommand(childCtx, dispatchArgs, dispatchInput, dispatchHasInput, ops)
			return flattenExecOutput(result), result.code
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
		result := execOutput{stdout: out, code: code}
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
			Message:  strings.TrimSpace(flattenExecOutput(result)),
		})
		return enforceOutputLimit(ctx, result, ops.Policy)
	}
	result := e.runExternalCommand(ctx, rawCmd, args[1:], input, hasInput, ops)
	return enforceOutputLimit(ctx, result, ops.Policy)
}

func (e *Engine) runExternalCommand(ctx context.Context, cmd string, args []string, input string, hasInput bool, ops contract.Ops) execOutput {
	if err := executionContextErr(ctx); err != nil {
		return contextExecOutput(err)
	}
	emitAudit(ctx, ops, contract.AuditEvent{
		Time:    time.Now(),
		Phase:   contract.AuditPhaseCommandStart,
		Command: strings.TrimSpace(cmd),
		Args:    append([]string(nil), args...),
	})
	if ops.RunExternalCommand == nil {
		out := fmt.Sprintf("%s: Not supported", cmd)
		emitAudit(ctx, ops, contract.AuditEvent{Time: time.Now(), Phase: contract.AuditPhaseCommandError, Command: cmd, Args: append([]string(nil), args...), ExitCode: contract.ExitCodeUnsupported, Message: out})
		return execOutput{stdout: out, code: contract.ExitCodeUnsupported}
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
			return execOutput{stdout: out, code: contract.ExitCodeUnsupported}
		}
		out := fmt.Sprintf("%s: %v", cmd, err)
		emitAudit(ctx, ops, contract.AuditEvent{Time: time.Now(), Phase: contract.AuditPhaseCommandError, Command: cmd, Args: append([]string(nil), args...), ExitCode: contract.ExitCodeGeneral, Message: out})
		return execOutput{stdout: out, code: contract.ExitCodeGeneral}
	}
	output := execOutput{stdout: result.Stdout, stderr: result.Stderr, code: result.ExitCode}
	if result.ExitCode == 0 {
		emitAudit(ctx, ops, contract.AuditEvent{Time: time.Now(), Phase: contract.AuditPhaseCommandEnd, Command: cmd, Args: append([]string(nil), args...), ExitCode: 0, Message: strings.TrimSpace(flattenExecOutput(output))})
		return output
	}
	if result.ExitCode < 0 {
		output.code = contract.ExitCodeGeneral
	}
	if strings.TrimSpace(output.stdout) == "" && strings.TrimSpace(output.stderr) == "" {
		output.stderr = fmt.Sprintf("%s: command failed", cmd)
	}
	emitAudit(ctx, ops, contract.AuditEvent{Time: time.Now(), Phase: contract.AuditPhaseCommandError, Command: cmd, Args: append([]string(nil), args...), ExitCode: output.code, Message: strings.TrimSpace(flattenExecOutput(output))})
	return output
}

func enforceOutputLimit(ctx context.Context, out execOutput, policy contract.ExecutionPolicy) execOutput {
	if policy.MaxOutputBytes > 0 && len(flattenExecOutput(out)) > policy.MaxOutputBytes {
		markTraceOutputTruncated(ctx)
		return execOutput{stdout: fmt.Sprintf("execute: output exceeds limit (%d bytes)", policy.MaxOutputBytes), code: contract.ExitCodeGeneral}
	}
	return out
}

func flattenExecOutput(out execOutput) string {
	switch {
	case out.stdout == "":
		return out.stderr
	case out.stderr == "":
		return out.stdout
	case strings.HasSuffix(out.stdout, "\n"):
		return out.stdout + out.stderr
	default:
		return out.stdout + "\n" + out.stderr
	}
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
