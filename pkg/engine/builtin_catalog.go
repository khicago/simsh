package engine

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"sync"

	"github.com/khicago/simsh/pkg/contract"
)

// CommandRuntime carries execution context for one builtin invocation.
type CommandRuntime struct {
	Ctx                  context.Context
	Ops                  contract.Ops
	Stdin                string
	HasStdin             bool
	Dispatch             func(args []string, input string, hasInput bool) (string, int)
	LookupManual         func(name string) (string, bool)
	LookupDetailedManual func(name string) (string, bool)
	ListBuiltinNames     func() []string
	LookupBuiltinDoc     func(name string) (contract.BuiltinCommandDoc, bool)
	BuiltinCommandDocs   func() []contract.BuiltinCommandDoc
}

// CommandSpec defines one builtin command and its metadata.
type CommandSpec struct {
	Name             string
	Summary          string
	Manual           string
	Tips             []string
	Examples         []string
	DetailedManual   string
	Capabilities     []string
	StdinMode        string
	Operands         string
	DefaultOutput    string
	StructuredOutput string
	StructuredFlags  []string
	MutationKind     string
	SuccessOutput    string
	PipeBehavior     string
	ExitCodes        []string
	Run              func(runtime CommandRuntime, args []string) (string, int)
}

type Registry struct {
	mu       sync.RWMutex
	commands map[string]CommandSpec
}

func NewRegistry() *Registry {
	return &Registry{commands: map[string]CommandSpec{}}
}

func (r *Registry) Register(spec CommandSpec) error {
	if r == nil {
		return fmt.Errorf("registry is nil")
	}
	name := strings.TrimSpace(spec.Name)
	if name == "" {
		return fmt.Errorf("builtin command name is required")
	}
	if spec.Run == nil {
		return fmt.Errorf("builtin command %q missing Run", name)
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, exists := r.commands[name]; exists {
		return fmt.Errorf("duplicate builtin command %q", name)
	}
	spec.Name = name
	r.commands[name] = spec
	return nil
}

func (r *Registry) MustRegister(spec CommandSpec) {
	if err := r.Register(spec); err != nil {
		panic(err)
	}
}

func (r *Registry) Lookup(name string) (CommandSpec, bool) {
	if r == nil {
		return CommandSpec{}, false
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	spec, ok := r.commands[strings.TrimSpace(name)]
	return spec, ok
}

func (r *Registry) BuiltinManual(name string) (string, bool) {
	spec, ok := r.Lookup(name)
	if !ok {
		return "", false
	}
	doc := builtinDocFromSpec(spec)
	var sb strings.Builder
	sb.WriteString(strings.TrimSpace(doc.Manual))
	if strings.TrimSpace(doc.Summary) != "" {
		sb.WriteString("\n\nSummary:\n")
		sb.WriteString("  " + strings.TrimSpace(doc.Summary))
	}
	if lines := builtinContractLines(doc); len(lines) > 0 {
		sb.WriteString("\n\nContract:\n")
		for _, line := range lines {
			sb.WriteString("  - " + line + "\n")
		}
	}
	if len(spec.Tips) > 0 {
		sb.WriteString("\n\nTips:\n")
		for _, tip := range spec.Tips {
			sb.WriteString("  - " + tip + "\n")
		}
	}
	if len(spec.Examples) > 0 {
		sb.WriteString("\nExamples:\n")
		for _, ex := range spec.Examples {
			sb.WriteString("  " + ex + "\n")
		}
	}
	if len(doc.ExitCodes) > 0 {
		sb.WriteString("\nExit-Codes:\n")
		for _, line := range doc.ExitCodes {
			sb.WriteString("  - " + line + "\n")
		}
	}
	return strings.TrimRight(sb.String(), "\n"), true
}

func (r *Registry) BuiltinDetailedManual(name string) (string, bool) {
	spec, ok := r.Lookup(name)
	if !ok {
		return "", false
	}
	if strings.TrimSpace(spec.DetailedManual) == "" {
		return r.BuiltinManual(name)
	}
	return strings.TrimSpace(spec.DetailedManual), true
}

func (r *Registry) BuiltinCommandDocs() []contract.BuiltinCommandDoc {
	if r == nil {
		return nil
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	names := make([]string, 0, len(r.commands))
	for name := range r.commands {
		trimmed := strings.TrimSpace(name)
		if trimmed == "" {
			continue
		}
		names = append(names, trimmed)
	}
	sort.Strings(names)
	out := make([]contract.BuiltinCommandDoc, 0, len(names))
	for _, name := range names {
		spec := r.commands[name]
		out = append(out, builtinDocFromSpec(spec))
	}
	return out
}

// ListNames returns all registered command names in sorted order.
func (r *Registry) ListNames() []string {
	if r == nil {
		return nil
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	names := make([]string, 0, len(r.commands))
	for name := range r.commands {
		trimmed := strings.TrimSpace(name)
		if trimmed != "" {
			names = append(names, trimmed)
		}
	}
	sort.Strings(names)
	return names
}

func (r *Registry) LookupBuiltinDoc(name string) (contract.BuiltinCommandDoc, bool) {
	spec, ok := r.Lookup(name)
	if !ok {
		return contract.BuiltinCommandDoc{}, false
	}
	return builtinDocFromSpec(spec), true
}

func builtinDocFromSpec(spec CommandSpec) contract.BuiltinCommandDoc {
	return contract.BuiltinCommandDoc{
		Name:             spec.Name,
		Summary:          builtinSummary(spec),
		Manual:           strings.TrimSpace(spec.Manual),
		Tips:             append([]string(nil), spec.Tips...),
		Examples:         append([]string(nil), spec.Examples...),
		DetailedManual:   spec.DetailedManual,
		Capabilities:     append([]string(nil), spec.Capabilities...),
		StdinMode:        strings.TrimSpace(spec.StdinMode),
		Operands:         strings.TrimSpace(spec.Operands),
		DefaultOutput:    strings.TrimSpace(spec.DefaultOutput),
		StructuredOutput: strings.TrimSpace(spec.StructuredOutput),
		StructuredFlags:  append([]string(nil), spec.StructuredFlags...),
		MutationKind:     strings.TrimSpace(spec.MutationKind),
		SuccessOutput:    strings.TrimSpace(spec.SuccessOutput),
		PipeBehavior:     strings.TrimSpace(spec.PipeBehavior),
		ExitCodes:        append([]string(nil), spec.ExitCodes...),
	}
}

func builtinSummary(spec CommandSpec) string {
	if summary := strings.TrimSpace(spec.Summary); summary != "" {
		return summary
	}
	return inferManualHeadingSummary(spec.DetailedManual)
}

func builtinContractLines(doc contract.BuiltinCommandDoc) []string {
	lines := make([]string, 0, 8)
	if value := strings.TrimSpace(doc.StdinMode); value != "" {
		lines = append(lines, "stdin: "+value)
	}
	if value := strings.TrimSpace(doc.Operands); value != "" {
		lines = append(lines, "operands: "+value)
	}
	if value := strings.TrimSpace(doc.DefaultOutput); value != "" {
		lines = append(lines, "default output: "+value)
	}
	if len(doc.StructuredFlags) > 0 || strings.TrimSpace(doc.StructuredOutput) != "" {
		line := "structured output: "
		desc := strings.TrimSpace(doc.StructuredOutput)
		flags := strings.Join(doc.StructuredFlags, ", ")
		switch {
		case desc != "" && flags != "":
			line += desc + " via " + flags
		case desc != "":
			line += desc
		default:
			line += flags
		}
		lines = append(lines, line)
	}
	if value := strings.TrimSpace(doc.PipeBehavior); value != "" {
		lines = append(lines, "pipe behavior: "+value)
	}
	if value := strings.TrimSpace(doc.MutationKind); value != "" {
		lines = append(lines, "mutation: "+value)
	}
	if value := strings.TrimSpace(doc.SuccessOutput); value != "" {
		lines = append(lines, "success stdout: "+value)
	}
	return lines
}

func inferManualHeadingSummary(raw string) string {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return ""
	}
	lines := strings.Split(trimmed, "\n")
	start := 0
	if len(lines) > 0 && strings.TrimSpace(lines[0]) == "---" {
		for idx := 1; idx < len(lines); idx++ {
			if strings.TrimSpace(lines[idx]) == "---" {
				start = idx + 1
				break
			}
		}
	}
	for _, line := range lines[start:] {
		trimmedLine := strings.TrimSpace(line)
		if !strings.HasPrefix(trimmedLine, "# ") {
			continue
		}
		heading := strings.TrimSpace(strings.TrimPrefix(trimmedLine, "# "))
		if idx := strings.Index(heading, " -- "); idx >= 0 {
			return strings.TrimSpace(heading[idx+4:])
		}
		if idx := strings.Index(heading, " - "); idx >= 0 {
			return strings.TrimSpace(heading[idx+3:])
		}
		return heading
	}
	return ""
}
