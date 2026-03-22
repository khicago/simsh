package builtin

import (
	"errors"
	"fmt"
	"strings"

	"github.com/khicago/simsh/pkg/contract"
	"github.com/khicago/simsh/pkg/engine"
)

type commandLookup struct {
	name string
	path string
	kind string
}

type commandUsageError struct {
	msg string
}

func (e commandUsageError) Error() string {
	return e.msg
}

const (
	commandKindBuiltin  = "builtin"
	commandKindExternal = "external"
	commandKindAlias    = "alias"
)

func specCd() engine.CommandSpec {
	return engine.CommandSpec{
		Name:   CommandCd,
		Manual: "cd [PATH]",
		Tips: []string{
			"Changes the session-local virtual working directory.",
			"With no argument, cd returns to the virtual root.",
		},
		Examples:       ExamplesFor("cd"),
		DetailedManual: LoadEmbeddedManual("cd"),
		Run:            runCd,
	}
}

func runCd(runtime engine.CommandRuntime, args []string) (string, int) {
	if runtime.Ops.ChangeWorkingDir == nil {
		return "cd: not supported", contract.ExitCodeUnsupported
	}
	if len(args) > 1 {
		return "cd: expected zero or one argument", contract.ExitCodeUsage
	}
	target := ""
	if len(args) == 1 {
		target = args[0]
	}
	if _, err := runtime.Ops.ChangeWorkingDir(runtime.Ctx, target); err != nil {
		return fmt.Sprintf("cd: %v", err), contract.ExitCodeGeneral
	}
	return "", 0
}

func specPwd() engine.CommandSpec {
	return engine.CommandSpec{
		Name:   CommandPwd,
		Manual: "pwd",
		Tips: []string{
			"Prints the current session-local virtual working directory.",
		},
		Examples:       ExamplesFor("pwd"),
		DetailedManual: LoadEmbeddedManual("pwd"),
		Run:            runPwd,
	}
}

func runPwd(runtime engine.CommandRuntime, args []string) (string, int) {
	if len(args) > 0 {
		return "pwd: expected no arguments", contract.ExitCodeUsage
	}
	return normalizeAbsolutePath(currentWorkingDir(runtime.Ops)), 0
}

func specWhich() engine.CommandSpec {
	return engine.CommandSpec{
		Name:   CommandWhich,
		Manual: "which COMMAND...",
		Tips: []string{
			"Search order is alias -> /sys/bin (system builtin) -> /bin (custom external).",
		},
		Examples:       ExamplesFor("which"),
		DetailedManual: LoadEmbeddedManual("which"),
		Run:            runWhich,
	}
}

func runWhich(runtime engine.CommandRuntime, args []string) (string, int) {
	resolved, missing, err := resolveCommandLookups(runtime, args, "which")
	if err != nil {
		var usageErr commandUsageError
		if errors.As(err, &usageErr) {
			return err.Error(), contract.ExitCodeUsage
		}
		return err.Error(), contract.ExitCodeGeneral
	}
	out := make([]string, 0, len(resolved)+len(missing))
	for _, entry := range resolved {
		out = append(out, entry.path)
	}
	for _, name := range missing {
		out = append(out, fmt.Sprintf("which: %s: not found", name))
	}
	if len(missing) > 0 {
		return strings.Join(out, "\n"), contract.ExitCodeGeneral
	}
	return strings.Join(out, "\n"), 0
}

func specType() engine.CommandSpec {
	return engine.CommandSpec{
		Name:   CommandType,
		Manual: "type COMMAND...",
		Tips: []string{
			"Reports whether a command resolves to alias, builtin, or external.",
			"Lookup order matches command execution: alias -> /sys/bin builtin -> /bin custom external.",
		},
		Examples:       ExamplesFor("type"),
		DetailedManual: LoadEmbeddedManual("type"),
		Run:            runType,
	}
}

func runType(runtime engine.CommandRuntime, args []string) (string, int) {
	resolved, missing, err := resolveCommandLookups(runtime, args, "type")
	if err != nil {
		var usageErr commandUsageError
		if errors.As(err, &usageErr) {
			return err.Error(), contract.ExitCodeUsage
		}
		return err.Error(), contract.ExitCodeGeneral
	}
	out := make([]string, 0, len(resolved)+len(missing))
	for _, entry := range resolved {
		out = append(out, fmt.Sprintf("%s is %s (%s)", entry.name, entry.path, entry.kind))
	}
	for _, name := range missing {
		out = append(out, fmt.Sprintf("type: %s: not found", name))
	}
	if len(missing) > 0 {
		return strings.Join(out, "\n"), contract.ExitCodeGeneral
	}
	return strings.Join(out, "\n"), 0
}

func resolveCommandLookups(runtime engine.CommandRuntime, args []string, commandName string) ([]commandLookup, []string, error) {
	if len(args) == 0 {
		return nil, nil, commandUsageError{msg: fmt.Sprintf("%s: missing operand", commandName)}
	}

	builtins := make(map[string]struct{})
	if runtime.ListBuiltinNames != nil {
		for _, name := range runtime.ListBuiltinNames() {
			trimmed := strings.TrimSpace(name)
			if trimmed == "" {
				continue
			}
			builtins[trimmed] = struct{}{}
		}
	}

	externals := make(map[string]struct{})
	if runtime.Ops.ListExternalCommands != nil {
		commands, err := runtime.Ops.ListExternalCommands(runtime.Ctx)
		if err != nil {
			if !errors.Is(err, contract.ErrUnsupported) {
				return nil, nil, fmt.Errorf("%s: %v", commandName, err)
			}
		} else {
			for _, cmd := range commands {
				name := normalizeCommandName(cmd.Name)
				if name == "" {
					continue
				}
				externals[name] = struct{}{}
			}
		}
	}

	aliases := contract.NormalizeCommandAliases(runtime.Ops.CommandAliases)

	resolved := make([]commandLookup, 0, len(args))
	missing := make([]string, 0)
	for _, raw := range args {
		trimmed := strings.TrimSpace(raw)
		if trimmed == "" {
			return nil, nil, commandUsageError{msg: fmt.Sprintf("%s: command name must not be empty", commandName)}
		}
		if strings.HasPrefix(trimmed, "-") {
			return nil, nil, commandUsageError{msg: fmt.Sprintf("%s: unsupported flag %s", commandName, trimmed)}
		}
		ref, err := normalizeCommandReferenceForRuntime(runtime, trimmed)
		if err != nil {
			return nil, nil, fmt.Errorf("%s: %v", commandName, err)
		}
		if match, ok := resolveOneCommandLookup(ref, builtins, externals, aliases); ok {
			resolved = append(resolved, match)
			continue
		}
		missing = append(missing, trimmed)
	}
	return resolved, missing, nil
}

func resolveOneCommandLookup(ref contract.CommandReference, builtins map[string]struct{}, externals map[string]struct{}, aliases map[string][]string) (commandLookup, bool) {
	normalized := strings.TrimSpace(ref.Name)
	if normalized == "" {
		return commandLookup{}, false
	}

	if !ref.PathLike {
		if candidate, ok := resolveAliasLookup(normalized, aliases); ok {
			return candidate, true
		}
	}
	if ref.Namespace != contract.CommandNamespaceExternal {
		if candidate, ok := resolveBuiltinLookup(normalized, builtins); ok {
			return candidate, true
		}
	}
	if ref.Namespace != contract.CommandNamespaceBuiltin {
		if candidate, ok := resolveExternalLookup(normalized, externals); ok {
			return candidate, true
		}
	}
	return commandLookup{}, false
}

func normalizeCommandReferenceForRuntime(runtime engine.CommandRuntime, raw string) (contract.CommandReference, error) {
	resolve := runtime.Ops.ResolvePath
	if resolve == nil {
		resolve = runtime.Ops.RequireAbsolutePath
	}
	return contract.NormalizeCommandReference(raw, resolve)
}

func resolveAliasLookup(query string, aliases map[string][]string) (commandLookup, bool) {
	name := normalizeCommandName(query)
	if name == "" {
		return commandLookup{}, false
	}
	expansion, ok := aliases[name]
	if !ok || len(expansion) == 0 {
		return commandLookup{}, false
	}
	rendered := fmt.Sprintf("alias %s='%s'", name, strings.Join(expansion, " "))
	return commandLookup{name: name, path: rendered, kind: commandKindAlias}, true
}

func resolveBuiltinLookup(query string, builtins map[string]struct{}) (commandLookup, bool) {
	name := normalizeCommandName(query)
	if name == "" {
		name = normalizePrefixedCommandName(query, contract.VirtualSystemBinDir)
	}
	if name == "" {
		return commandLookup{}, false
	}
	if _, ok := builtins[name]; !ok {
		return commandLookup{}, false
	}
	return commandLookup{name: name, path: contract.VirtualSystemBinDir + "/" + name, kind: commandKindBuiltin}, true
}

func resolveExternalLookup(query string, externals map[string]struct{}) (commandLookup, bool) {
	name := normalizeCommandName(query)
	if name == "" {
		name = normalizePrefixedCommandName(query, contract.VirtualExternalBinDir)
	}
	if name == "" {
		return commandLookup{}, false
	}
	if _, ok := externals[name]; !ok {
		return commandLookup{}, false
	}
	return commandLookup{name: name, path: contract.VirtualExternalBinDir + "/" + name, kind: commandKindExternal}, true
}

func normalizeCommandName(raw string) string {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return ""
	}
	if strings.HasPrefix(trimmed, "/") {
		return ""
	}
	if strings.Contains(trimmed, "/") {
		return ""
	}
	return trimmed
}

func normalizePrefixedCommandName(raw string, prefix string) string {
	normalized := normalizeAbsolutePath(raw)
	if !strings.HasPrefix(normalized, prefix+"/") {
		return ""
	}
	name := strings.TrimPrefix(normalized, prefix+"/")
	if strings.Contains(name, "/") || strings.TrimSpace(name) == "" {
		return ""
	}
	return name
}
