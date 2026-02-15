package simsh

import (
	"context"

	"github.com/khicago/simsh/pkg/builtin"
	"github.com/khicago/simsh/pkg/contract"
	"github.com/khicago/simsh/pkg/engine"
)

// NewDefaultEngine creates an engine with the default builtin command set.
func NewDefaultEngine() *engine.Engine {
	registry := engine.NewRegistry()
	builtin.RegisterDefaults(registry)
	return engine.New(registry)
}

// Execute runs one command/script against callback ops using the default engine.
func Execute(ctx context.Context, cmdline string, ops contract.Ops) (string, int) {
	return NewDefaultEngine().Execute(ctx, cmdline, ops)
}

func BuiltinCommandDocs() []contract.BuiltinCommandDoc {
	return NewDefaultEngine().BuiltinCommandDocs()
}

func VirtualSystemBinDir() string {
	return contract.VirtualSystemBinDir
}

func VirtualExternalBinDir() string {
	return contract.VirtualExternalBinDir
}

func ScriptControlOperators() []string {
	return []string{"|", ";", "&&", "||"}
}

func ScriptRedirectionOperators() []string {
	return []string{">", ">>", "<", "<<"}
}

func UnsupportedShellSyntaxNotes() []string {
	return []string{
		"command substitution ($( ... ))",
		"process substitution",
		"fd-level redirection (2>&1, etc.)",
	}
}
