package env

import (
	"embed"
	"fmt"
	"sort"
	"strings"

	"github.com/khicago/simsh/pkg/contract"
	"github.com/khicago/simsh/pkg/engine"
)

var examples = []string{"env", "env PATH"}

//go:embed manual.md
var manualFS embed.FS

func detailedManual() string {
	data, err := manualFS.ReadFile("manual.md")
	if err != nil {
		return ""
	}
	return string(data)
}

func Spec() engine.CommandSpec {
	return engine.CommandSpec{
		Name:   "env",
		Manual: "env [KEY]",
		Tips: []string{
			"Use env PATH to inspect command search order.",
		},
		Examples:       append([]string(nil), examples...),
		DetailedManual: detailedManual(),
		Run:            run,
	}
}

func run(runtime engine.CommandRuntime, args []string) (string, int) {
	if len(args) > 1 {
		return "env: expected at most one variable name", contract.ExitCodeUsage
	}
	key := ""
	if len(args) == 1 {
		key = strings.TrimSpace(args[0])
		if key == "" {
			return "env: variable name must not be empty", contract.ExitCodeUsage
		}
	}
	vars := buildEnvVars(runtime.Ops)
	if key != "" {
		if val, ok := vars[key]; ok {
			return fmt.Sprintf("%s=%s", key, val), 0
		}
		return "", 0
	}
	keys := make([]string, 0, len(vars))
	for k := range vars {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	lines := make([]string, 0, len(keys))
	for _, k := range keys {
		lines = append(lines, fmt.Sprintf("%s=%s", k, vars[k]))
	}
	return strings.Join(lines, "\n"), 0
}

func buildEnvVars(ops contract.Ops) map[string]string {
	parts := make([]string, 0, 2+len(ops.PathEnv))
	parts = append(parts, contract.VirtualSystemBinDir)
	parts = append(parts, contract.VirtualExternalBinDir)
	for _, raw := range ops.PathEnv {
		p := normalizeAbsolutePath(raw)
		if p == "/" || containsString(parts, p) {
			continue
		}
		if p == contract.VirtualSystemBinDir || p == contract.VirtualExternalBinDir {
			continue
		}
		parts = append(parts, p)
	}
	return map[string]string{"PATH": strings.Join(parts, ":")}
}

func containsString(list []string, target string) bool {
	for _, item := range list {
		if item == target {
			return true
		}
	}
	return false
}

func normalizeAbsolutePath(raw string) string {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return "/"
	}
	if !strings.HasPrefix(trimmed, "/") {
		trimmed = "/" + trimmed
	}
	parts := strings.Split(trimmed, "/")
	stack := make([]string, 0, len(parts))
	for _, part := range parts {
		switch part {
		case "", ".":
			continue
		case "..":
			if len(stack) > 0 {
				stack = stack[:len(stack)-1]
			}
		default:
			stack = append(stack, part)
		}
	}
	if len(stack) == 0 {
		return "/"
	}
	return "/" + strings.Join(stack, "/")
}
