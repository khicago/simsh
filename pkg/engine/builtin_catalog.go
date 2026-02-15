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
}

// CommandSpec defines one builtin command and its metadata.
type CommandSpec struct {
	Name           string
	Manual         string
	Tips           []string
	Examples       []string
	DetailedManual string
	Capabilities   []string
	Run            func(runtime CommandRuntime, args []string) (string, int)
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
	var sb strings.Builder
	sb.WriteString(strings.TrimSpace(spec.Manual))
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
	return sb.String(), true
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
		out = append(out, contract.BuiltinCommandDoc{
			Name:           spec.Name,
			Manual:         strings.TrimSpace(spec.Manual),
			Tips:           append([]string(nil), spec.Tips...),
			Examples:       append([]string(nil), spec.Examples...),
			DetailedManual: spec.DetailedManual,
			Capabilities:   append([]string(nil), spec.Capabilities...),
		})
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
	return contract.BuiltinCommandDoc{
		Name:           spec.Name,
		Manual:         strings.TrimSpace(spec.Manual),
		Tips:           append([]string(nil), spec.Tips...),
		Examples:       append([]string(nil), spec.Examples...),
		DetailedManual: spec.DetailedManual,
		Capabilities:   append([]string(nil), spec.Capabilities...),
	}, true
}
