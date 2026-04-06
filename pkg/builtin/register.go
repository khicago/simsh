package builtin

import (
	"fmt"
	"strings"

	"github.com/khicago/simsh/pkg/engine"
)

func RegisterDefaults(reg *engine.Registry) {
	if reg == nil {
		return
	}
	for _, spec := range defaultCommandSpecs() {
		reg.MustRegister(spec)
	}
}

func RegisterDefaultSubset(reg *engine.Registry, names []string) error {
	if reg == nil {
		return nil
	}
	allowed := make(map[string]struct{}, len(names))
	for _, name := range names {
		trimmed := normalizeDefaultCommandName(name)
		if trimmed == "" {
			continue
		}
		allowed[trimmed] = struct{}{}
	}
	if len(allowed) == 0 {
		return nil
	}
	registered := make(map[string]struct{}, len(allowed))
	for _, spec := range defaultCommandSpecs() {
		if _, ok := allowed[spec.Name]; !ok {
			continue
		}
		if err := reg.Register(spec); err != nil {
			return err
		}
		registered[spec.Name] = struct{}{}
	}
	for name := range allowed {
		if _, ok := registered[name]; ok {
			continue
		}
		return engineErrf("unknown builtin command %q", name)
	}
	return nil
}

func defaultCommandSpecs() []engine.CommandSpec {
	return []engine.CommandSpec{
		applyCommandDocContract(specLS()),
		applyCommandDocContract(specTree()),
		applyCommandDocContract(specCd()),
		applyCommandDocContract(specPwd()),
		applyCommandDocContract(specEnv()),
		applyCommandDocContract(specFrontmatter()),
		applyCommandDocContract(specJSON()),
		applyCommandDocContract(specCat()),
		applyCommandDocContract(specHead()),
		applyCommandDocContract(specTail()),
		applyCommandDocContract(specGrep()),
		applyCommandDocContract(specRG()),
		applyCommandDocContract(specFind()),
		applyCommandDocContract(specWhich()),
		applyCommandDocContract(specType()),
		applyCommandDocContract(specEcho()),
		applyCommandDocContract(specTee()),
		applyCommandDocContract(specSed()),
		applyCommandDocContract(specMan()),
		applyCommandDocContract(specDate()),
		applyCommandDocContract(specMkdir()),
		applyCommandDocContract(specCp()),
		applyCommandDocContract(specMv()),
		applyCommandDocContract(specRm()),
		applyCommandDocContract(specRmdir()),
		applyCommandDocContract(specTouch()),
		applyCommandDocContract(specWc()),
		applyCommandDocContract(specSort()),
		applyCommandDocContract(specUniq()),
		applyCommandDocContract(specDiff()),
	}
}

func normalizeDefaultCommandName(name string) string {
	return strings.TrimSpace(name)
}

func engineErrf(format string, args ...any) error {
	return fmt.Errorf(format, args...)
}
