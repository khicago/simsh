package builtin

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/khicago/simsh/pkg/contract"
	"github.com/khicago/simsh/pkg/engine"
)

type envVarRecord struct {
	Key   string   `json:"key"`
	Value string   `json:"value,omitempty"`
	Parts []string `json:"parts,omitempty"`
	Found bool     `json:"found,omitempty"`
}

func specEnv() engine.CommandSpec {
	return engine.CommandSpec{
		Name:   CommandEnv,
		Manual: "env [--json] [--split] [KEY]",
		Tips: []string{
			"Use env PATH to inspect command search order.",
			"Use --split with one key such as PATH when you want list-like values one item per line.",
			"Use --json for machine-readable environment output.",
		},
		StructuredOutput: "environment snapshot",
		StructuredFlags:  []string{"--json", "--split"},
		Examples:         ExamplesFor("env"),
		DetailedManual:   LoadEmbeddedManual("env"),
		Run:              runEnv,
	}
}

func runEnv(runtime engine.CommandRuntime, args []string) (string, int) {
	jsonOutput := false
	splitOutput := false
	key := ""

	for _, arg := range args {
		switch arg {
		case "--json":
			jsonOutput = true
			continue
		case "--split":
			splitOutput = true
			continue
		}
		if key != "" {
			return "env: expected at most one variable name", contract.ExitCodeUsage
		}
		key = strings.TrimSpace(arg)
		if key == "" {
			return "env: variable name must not be empty", contract.ExitCodeUsage
		}
	}
	if splitOutput && key == "" {
		return "env: --split requires exactly one variable name", contract.ExitCodeUsage
	}

	vars := buildEnvVars(runtime.Ops)
	if key != "" {
		val, ok := vars[key]
		if splitOutput {
			if !ok {
				return "", 0
			}
			parts := splitEnvVarValue(key, val)
			return strings.Join(parts, "\n"), 0
		}
		if jsonOutput {
			record := envVarRecord{Key: key}
			if ok {
				record.Value = val
				record.Found = true
				record.Parts = splitEnvVarValue(key, val)
			}
			raw, err := json.Marshal(record)
			if err != nil {
				return fmt.Sprintf("env: %v", err), contract.ExitCodeGeneral
			}
			return string(raw), 0
		}
		if ok {
			return fmt.Sprintf("%s=%s", key, val), 0
		}
		return "", 0
	}

	keys := make([]string, 0, len(vars))
	for k := range vars {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	if jsonOutput {
		records := make([]envVarRecord, 0, len(keys))
		for _, k := range keys {
			record := envVarRecord{
				Key:   k,
				Value: vars[k],
				Found: true,
			}
			record.Parts = splitEnvVarValue(k, vars[k])
			records = append(records, record)
		}
		raw, err := json.Marshal(struct {
			Vars []envVarRecord `json:"vars"`
		}{
			Vars: records,
		})
		if err != nil {
			return fmt.Sprintf("env: %v", err), contract.ExitCodeGeneral
		}
		return string(raw), 0
	}

	lines := make([]string, 0, len(keys))
	for _, k := range keys {
		lines = append(lines, fmt.Sprintf("%s=%s", k, vars[k]))
	}
	return strings.Join(lines, "\n"), 0
}

func buildEnvVars(ops contract.Ops) map[string]string {
	vars := contract.NormalizeEnvVars(ops.EnvVars)

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
	if strings.TrimSpace(vars["PATH"]) == "" {
		vars["PATH"] = strings.Join(parts, ":")
	}
	return vars
}

func splitEnvVarValue(key string, value string) []string {
	if strings.TrimSpace(key) != "PATH" {
		return nil
	}
	if strings.TrimSpace(value) == "" {
		return nil
	}
	rawParts := strings.Split(value, ":")
	parts := make([]string, 0, len(rawParts))
	for _, raw := range rawParts {
		part := strings.TrimSpace(raw)
		if part == "" {
			continue
		}
		parts = append(parts, part)
	}
	return parts
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
