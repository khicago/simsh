package runtime

import (
	"context"
	"time"

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
	result := r.ExecuteResult(ctx, commandLine)
	return result.FlattenOutput(), result.ExitCode
}

func (r *Stack) ExecuteResult(ctx context.Context, commandLine string) contract.ExecutionResult {
	if r == nil || r.engine == nil {
		now := time.Now().UTC()
		return contract.ExecutionResult{
			ExecutionID: "",
			ExitCode:    contract.ExitCodeGeneral,
			Stdout:      "execute: runtime is not initialized",
			StartedAt:   now,
			FinishedAt:  now,
		}
	}
	return r.engine.ExecutePreparedResult(ctx, commandLine, r.prepared)
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

func (r *Stack) SessionState(rcFiles []string) contract.SessionState {
	if r == nil {
		return contract.SessionState{RCFiles: contract.NormalizeRCFiles(rcFiles)}
	}
	ops := r.Ops()
	return contract.SessionState{
		CommandAliases: contract.NormalizeCommandAliases(ops.CommandAliases),
		EnvVars:        contract.NormalizeEnvVars(ops.EnvVars),
		RCFiles:        contract.NormalizeRCFiles(rcFiles),
	}
}
