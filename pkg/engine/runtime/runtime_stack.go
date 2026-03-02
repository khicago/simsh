package runtime

import (
	"context"

	"github.com/khicago/simsh/pkg/builtin"
	"github.com/khicago/simsh/pkg/contract"
	"github.com/khicago/simsh/pkg/engine"
	"github.com/khicago/simsh/pkg/fs"
)

// Options defines how the runtime stack is composed.
type Options struct {
	HostRoot          string
	Profile           contract.CompatibilityProfile
	Policy            contract.ExecutionPolicy
	CommandAliases    map[string][]string
	EnvVars           map[string]string
	RCFiles           []string
	EnableTestCorpus  bool
	PathEnv           []string
	ExternalCallbacks fs.ExternalCallbacks
	FormatLSLongRow   contract.LSLongRowFormatter
	AuditSink         contract.AuditSink
}

// Stack composes shell execution engine with filesystem callbacks.
type Stack struct {
	engine   *engine.Engine
	ops      contract.Ops
	prepared engine.PreparedOps
}

// New composes the runtime from builtin shell engine and fs adapters.
func New(opts Options) (*Stack, error) {
	registry := engine.NewRegistry()
	builtin.RegisterDefaults(registry)
	core := engine.New(registry)

	ops, err := fs.NewRuntimeOps(fs.EnvironmentOptions{
		HostRoot:          opts.HostRoot,
		Profile:           opts.Profile,
		Policy:            opts.Policy,
		CommandAliases:    opts.CommandAliases,
		EnvVars:           opts.EnvVars,
		RCFiles:           opts.RCFiles,
		PathEnv:           opts.PathEnv,
		EnableTestCorpus:  opts.EnableTestCorpus,
		ExternalCallbacks: opts.ExternalCallbacks,
		FormatLSLongRow:   opts.FormatLSLongRow,
		AuditSink:         opts.AuditSink,
	})
	if err != nil {
		return nil, err
	}
	prepared, err := core.PrepareOps(context.Background(), ops)
	if err != nil {
		return nil, err
	}
	return &Stack{engine: core, ops: ops, prepared: prepared}, nil
}

func (r *Stack) Execute(ctx context.Context, commandLine string) (string, int) {
	if r == nil || r.engine == nil {
		return "execute: runtime is not initialized", contract.ExitCodeGeneral
	}
	return r.engine.ExecutePrepared(ctx, commandLine, r.prepared)
}

func (r *Stack) Ops() contract.Ops {
	if r == nil {
		return contract.Ops{}
	}
	if prepared := r.prepared.Ops(); prepared.RequireAbsolutePath != nil {
		return prepared
	}
	return r.ops
}

func (r *Stack) BuiltinDocs() []contract.BuiltinCommandDoc {
	if r == nil || r.engine == nil {
		return nil
	}
	return r.engine.BuiltinCommandDocs()
}
