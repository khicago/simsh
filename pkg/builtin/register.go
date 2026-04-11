package builtin

import (
	"fmt"
	"strings"

	"github.com/khicago/simsh/pkg/engine"
)

const canonicalDefaultBuiltinSourceTree = "pkg/builtin/spec* implementations in pkg/builtin/*.go"

type defaultBuiltinRegistration struct {
	Name            string
	CanonicalSource string
	Build           func() engine.CommandSpec
}

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
	registrations := defaultBuiltinRegistrations()
	specs := make([]engine.CommandSpec, 0, len(registrations))
	seen := make(map[string]struct{}, len(registrations))
	for _, registration := range registrations {
		spec := applyCommandDocContract(registration.Build())
		if spec.Name != registration.Name {
			panic(fmt.Sprintf("builtin registration %q produced spec %q from %s", registration.Name, spec.Name, registration.CanonicalSource))
		}
		if _, exists := seen[spec.Name]; exists {
			panic(fmt.Sprintf("duplicate builtin registration %q", spec.Name))
		}
		seen[spec.Name] = struct{}{}
		specs = append(specs, spec)
	}
	return specs
}

func defaultBuiltinRegistrations() []defaultBuiltinRegistration {
	return []defaultBuiltinRegistration{
		{Name: CommandLS, CanonicalSource: "pkg/builtin/op_listing.go", Build: specLS},
		{Name: CommandTree, CanonicalSource: "pkg/builtin/op_tree.go", Build: specTree},
		{Name: CommandCd, CanonicalSource: "pkg/builtin/op_command_lookup.go", Build: specCd},
		{Name: CommandPwd, CanonicalSource: "pkg/builtin/op_command_lookup.go", Build: specPwd},
		{Name: CommandEnv, CanonicalSource: "pkg/builtin/op_environment.go", Build: specEnv},
		{Name: CommandFrontmatter, CanonicalSource: "pkg/builtin/op_frontmatter.go", Build: specFrontmatter},
		{Name: CommandJSON, CanonicalSource: "pkg/builtin/op_json.go", Build: specJSON},
		{Name: CommandCat, CanonicalSource: "pkg/builtin/op_readfile.go", Build: specCat},
		{Name: CommandHead, CanonicalSource: "pkg/builtin/op_window.go", Build: specHead},
		{Name: CommandTail, CanonicalSource: "pkg/builtin/op_window.go", Build: specTail},
		{Name: CommandGrep, CanonicalSource: "pkg/builtin/op_pattern_scan.go", Build: specGrep},
		{Name: CommandRG, CanonicalSource: "pkg/builtin/op_rg.go", Build: specRG},
		{Name: CommandFind, CanonicalSource: "pkg/builtin/op_path_discovery.go", Build: specFind},
		{Name: CommandWhich, CanonicalSource: "pkg/builtin/op_command_lookup.go", Build: specWhich},
		{Name: CommandType, CanonicalSource: "pkg/builtin/op_command_lookup.go", Build: specType},
		{Name: CommandEcho, CanonicalSource: "pkg/builtin/op_emit_text.go", Build: specEcho},
		{Name: CommandTee, CanonicalSource: "pkg/builtin/op_mirror_write.go", Build: specTee},
		{Name: CommandSed, CanonicalSource: "pkg/builtin/op_stream_edit.go", Build: specSed},
		{Name: CommandMan, CanonicalSource: "pkg/builtin/op_help_manual.go", Build: specMan},
		{Name: CommandDate, CanonicalSource: "pkg/builtin/op_clock.go", Build: specDate},
		{Name: CommandMkdir, CanonicalSource: "pkg/builtin/op_mkdir.go", Build: specMkdir},
		{Name: CommandCp, CanonicalSource: "pkg/builtin/op_copy.go", Build: specCp},
		{Name: CommandMv, CanonicalSource: "pkg/builtin/op_move.go", Build: specMv},
		{Name: CommandRm, CanonicalSource: "pkg/builtin/op_remove.go", Build: specRm},
		{Name: CommandRmdir, CanonicalSource: "pkg/builtin/op_rmdir.go", Build: specRmdir},
		{Name: CommandTouch, CanonicalSource: "pkg/builtin/op_touch.go", Build: specTouch},
		{Name: CommandWc, CanonicalSource: "pkg/builtin/op_wordcount.go", Build: specWc},
		{Name: CommandSort, CanonicalSource: "pkg/builtin/op_sort.go", Build: specSort},
		{Name: CommandUniq, CanonicalSource: "pkg/builtin/op_uniq.go", Build: specUniq},
		{Name: CommandDiff, CanonicalSource: "pkg/builtin/op_diff.go", Build: specDiff},
	}
}

func defaultBuiltinOwnershipByName() map[string]defaultBuiltinRegistration {
	registrations := defaultBuiltinRegistrations()
	ownership := make(map[string]defaultBuiltinRegistration, len(registrations))
	for _, registration := range registrations {
		ownership[registration.Name] = registration
	}
	return ownership
}

func normalizeDefaultCommandName(name string) string {
	return strings.TrimSpace(name)
}

func engineErrf(format string, args ...any) error {
	return fmt.Errorf(format, args...)
}
