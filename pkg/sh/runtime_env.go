package sh

import (
	"context"
	"sort"
	"strings"

	"github.com/khicago/simsh/pkg/builtin"
	"github.com/khicago/simsh/pkg/contract"
	"github.com/khicago/simsh/pkg/engine"
)

type Runtime struct {
	engine *engine.Engine
}

func NewRuntime() *Runtime {
	registry := engine.NewRegistry()
	builtin.RegisterDefaults(registry)
	return &Runtime{engine: engine.New(registry)}
}

func (r *Runtime) Execute(ctx context.Context, commandLine string, ops contract.Ops) (string, int) {
	return r.engine.Execute(ctx, commandLine, ops)
}

func (r *Runtime) BuiltinDocs() []contract.BuiltinCommandDoc {
	return r.engine.BuiltinCommandDocs()
}

func DescribeMarkdown() string {
	runtime := NewRuntime()
	docs := runtime.BuiltinDocs()
	sort.Slice(docs, func(i, j int) bool { return docs[i].Name < docs[j].Name })
	names := make([]string, 0, len(docs))
	for _, doc := range docs {
		if strings.TrimSpace(doc.Name) == "" {
			continue
		}
		names = append(names, doc.Name)
	}
	lines := []string{
		"## Shell Runtime",
		"- deterministic script operators: `;`, `&&`, `||`, `|`",
		"- deterministic redirections: `>`, `>>`, `<`, `<<`",
		"- command mounts: `/sys/bin` (builtin), `/bin` (injected external)",
		"- parent directories of mount points are exposed as synthetic virtual directories (e.g. `/sys`)",
		"- mount-backed virtual paths are immutable (no write/edit/remove/mkdir/cp/mv on those paths)",
		"- builtin commands: `" + strings.Join(names, "`, `") + "`",
		"- profile gates: `core-strict`, `bash-plus`, `zsh-lite`",
		"",
		"### Builtin Command Reference",
	}
	for _, doc := range docs {
		if strings.TrimSpace(doc.Name) == "" {
			continue
		}
		lines = append(lines, "")
		lines = append(lines, "#### "+doc.Name)
		lines = append(lines, "    "+strings.TrimSpace(doc.Manual))
		if len(doc.Tips) > 0 {
			for _, tip := range doc.Tips {
				lines = append(lines, "- "+tip)
			}
		}
		if len(doc.Examples) > 0 {
			lines = append(lines, "")
			lines = append(lines, "Examples:")
			for _, ex := range doc.Examples {
				lines = append(lines, "    "+ex)
			}
		}
	}
	return strings.Join(lines, "\n")
}
