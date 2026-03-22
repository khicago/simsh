package builtin

import (
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/khicago/simsh/pkg/contract"
	"github.com/khicago/simsh/pkg/engine"
)

func specMan() engine.CommandSpec {
	return engine.CommandSpec{
		Name:   CommandMan,
		Manual: "man [-v] [-l|--list] [--fmt text|json] <command>",
		Tips: []string{
			"man CMD shows summary with tips and examples.",
			"man -v CMD shows full documentation with command-specific details.",
			"man --list shows the builtin and external command catalog; add --fmt json for a machine-readable view.",
		},
		StructuredOutput: "machine-readable command catalog",
		StructuredFlags:  []string{"--list --fmt json"},
		Examples:         ExamplesFor("man"),
		DetailedManual:   LoadEmbeddedManual("man"),
		Run:              runMan,
	}
}

func runMan(runtime engine.CommandRuntime, args []string) (string, int) {
	verbose := false
	listMode := false
	listFormat := "text"
	target := ""

	for idx := 0; idx < len(args); idx++ {
		arg := args[idx]
		switch arg {
		case "-v":
			verbose = true
		case "-l", "--list":
			listMode = true
		case "--fmt":
			if idx+1 >= len(args) {
				return "man: --fmt requires one value: text|json", contract.ExitCodeUsage
			}
			idx++
			parsed, ok := parseManListFormat(args[idx])
			if !ok {
				return fmt.Sprintf("man: unsupported --fmt value %q", args[idx]), contract.ExitCodeUsage
			}
			listFormat = parsed
		default:
			if strings.HasPrefix(arg, "--fmt=") {
				parsed, ok := parseManListFormat(strings.TrimPrefix(arg, "--fmt="))
				if !ok {
					return fmt.Sprintf("man: unsupported --fmt value %q", strings.TrimPrefix(arg, "--fmt=")), contract.ExitCodeUsage
				}
				listFormat = parsed
				continue
			}
			if strings.HasPrefix(arg, "-") {
				return fmt.Sprintf("man: unsupported flag %s", arg), contract.ExitCodeUsage
			}
			if target != "" {
				return "man: expected at most one command name", contract.ExitCodeUsage
			}
			target = arg
		}
	}

	if !listMode && listFormat != "text" {
		return "man: --fmt is only supported with --list", contract.ExitCodeUsage
	}

	if listMode {
		return runManList(runtime, listFormat)
	}

	if target == "" {
		return "man: expected command name (use --list to see all commands)", contract.ExitCodeUsage
	}

	ref, err := normalizeCommandReferenceForRuntime(runtime, target)
	if err != nil {
		return fmt.Sprintf("man: %v", err), contract.ExitCodeGeneral
	}
	target = strings.TrimSpace(ref.Name)

	// Verbose mode: try detailed manual first.
	if verbose {
		if runtime.LookupDetailedManual != nil {
			if manual, ok := runtime.LookupDetailedManual(target); ok {
				manual = stripMarkdownFrontMatter(manual)
				if manual != "" {
					return manual, 0
				}
			}
		}
	}

	// Summary mode (or verbose fallback): use LookupManual.
	if runtime.LookupManual != nil {
		if manual, ok := runtime.LookupManual(target); ok {
			manual = strings.TrimSpace(manual)
			if manual != "" {
				if !verbose {
					manual = ensureSummaryGuidance(target, manual)
				}
				return manual, 0
			}
		}
	}

	// External command manual lookup.
	if runtime.Ops.ReadExternalManual != nil {
		manual, err := runtime.Ops.ReadExternalManual(runtime.Ctx, target)
		if err == nil {
			manual = strings.TrimSpace(manual)
			if manual != "" {
				if verbose {
					manual = stripMarkdownFrontMatter(manual)
				} else {
					manual = ensureSummaryGuidance(target, manual)
				}
				return manual, 0
			}
		} else if !errors.Is(err, contract.ErrUnsupported) {
			return fmt.Sprintf("man: %v", err), contract.ExitCodeGeneral
		}
	}

	// Fallback: check external command list for summary.
	if runtime.Ops.ListExternalCommands != nil {
		cmds, err := runtime.Ops.ListExternalCommands(runtime.Ctx)
		if err != nil && !errors.Is(err, contract.ErrUnsupported) {
			return fmt.Sprintf("man: %v", err), contract.ExitCodeGeneral
		}
		for _, c := range cmds {
			if strings.TrimSpace(c.Name) == target {
				if strings.TrimSpace(c.Summary) != "" {
					summary := strings.TrimSpace(c.Summary)
					if !verbose {
						summary = ensureSummaryGuidance(target, summary)
					}
					return summary, 0
				}
				out := fmt.Sprintf("binary: %s", c.Name)
				if !verbose {
					out = ensureSummaryGuidance(target, out)
				}
				return out, 0
			}
		}
	}

	return fmt.Sprintf("man: %s: not found", target), contract.ExitCodeGeneral
}

func parseManListFormat(raw string) (string, bool) {
	switch strings.TrimSpace(raw) {
	case "text", "":
		return "text", true
	case "json":
		return "json", true
	default:
		return "", false
	}
}

func ensureSummaryGuidance(command string, manual string) string {
	manual = strings.TrimSpace(manual)
	if manual == "" {
		return ""
	}
	lower := strings.ToLower(manual)
	if strings.Contains(lower, "\nuse-when:") && strings.Contains(lower, "\navoid-when:") {
		return manual
	}
	var sb strings.Builder
	sb.WriteString(manual)
	sb.WriteString("\n\nUse-When:\n")
	sb.WriteString(fmt.Sprintf("  - Quick syntax, flags, and examples for %s.\n", command))
	sb.WriteString("Avoid-When:\n")
	sb.WriteString(fmt.Sprintf("  - Need full semantics or edge cases; run man -v %s.", command))
	return sb.String()
}

func runManList(runtime engine.CommandRuntime, format string) (string, int) {
	builtins := collectBuiltinDocs(runtime)
	externals := collectExternalCommands(runtime)
	if format == "json" {
		payload := struct {
			Builtins []manListBuiltinRow  `json:"builtins"`
			External []manListExternalRow `json:"external,omitempty"`
		}{
			Builtins: make([]manListBuiltinRow, 0, len(builtins)),
			External: make([]manListExternalRow, 0, len(externals)),
		}
		for _, doc := range builtins {
			payload.Builtins = append(payload.Builtins, manListBuiltinRowFromDoc(doc))
		}
		for _, cmd := range externals {
			payload.External = append(payload.External, manListExternalRow{
				Name:    strings.TrimSpace(cmd.Name),
				Summary: externalSummary(cmd),
			})
		}
		raw, err := json.MarshalIndent(payload, "", "  ")
		if err != nil {
			return fmt.Sprintf("man: %v", err), contract.ExitCodeGeneral
		}
		return string(raw), 0
	}

	var sb strings.Builder
	sb.WriteString("Builtins:\n")
	sb.WriteString(fmt.Sprintf("  %-12s %-8s %-9s %-16s %s\n", "name", "stdin", "pipe", "structured", "summary"))
	for _, doc := range builtins {
		sb.WriteString(fmt.Sprintf(
			"  %-12s %-8s %-9s %-16s %s\n",
			doc.Name,
			renderListValue(doc.StdinMode),
			renderListValue(doc.PipeBehavior),
			renderListValue(strings.Join(doc.StructuredFlags, ",")),
			renderListValue(doc.Summary),
		))
	}
	if len(externals) > 0 {
		sb.WriteString("\nExternal:\n")
		sb.WriteString(fmt.Sprintf("  %-12s %s\n", "name", "summary"))
		for _, cmd := range externals {
			sb.WriteString(fmt.Sprintf("  %-12s %s\n", strings.TrimSpace(cmd.Name), externalSummary(cmd)))
		}
	}
	return strings.TrimRight(sb.String(), "\n"), 0
}

type manListBuiltinRow struct {
	Name            string   `json:"name"`
	Summary         string   `json:"summary"`
	StdinMode       string   `json:"stdin_mode,omitempty"`
	Operands        string   `json:"operands,omitempty"`
	DefaultOutput   string   `json:"default_output,omitempty"`
	StructuredFlags []string `json:"structured_flags,omitempty"`
	PipeBehavior    string   `json:"pipe_behavior,omitempty"`
	MutationKind    string   `json:"mutation_kind,omitempty"`
	SuccessOutput   string   `json:"success_output,omitempty"`
	ExitCodes       []string `json:"exit_codes,omitempty"`
}

type manListExternalRow struct {
	Name    string `json:"name"`
	Summary string `json:"summary"`
}

func collectBuiltinDocs(runtime engine.CommandRuntime) []contract.BuiltinCommandDoc {
	if runtime.BuiltinCommandDocs != nil {
		docs := runtime.BuiltinCommandDocs()
		sort.SliceStable(docs, func(i, j int) bool {
			return docs[i].Name < docs[j].Name
		})
		return docs
	}
	names := []string{}
	if runtime.ListBuiltinNames != nil {
		names = runtime.ListBuiltinNames()
		sort.Strings(names)
	}
	out := make([]contract.BuiltinCommandDoc, 0, len(names))
	for _, name := range names {
		if runtime.LookupBuiltinDoc != nil {
			if doc, ok := runtime.LookupBuiltinDoc(name); ok {
				out = append(out, doc)
				continue
			}
		}
		out = append(out, contract.BuiltinCommandDoc{Name: name})
	}
	return out
}

func collectExternalCommands(runtime engine.CommandRuntime) []contract.ExternalCommand {
	if runtime.Ops.ListExternalCommands == nil {
		return nil
	}
	cmds, err := runtime.Ops.ListExternalCommands(runtime.Ctx)
	if err != nil {
		return nil
	}
	sort.SliceStable(cmds, func(i, j int) bool {
		return strings.TrimSpace(cmds[i].Name) < strings.TrimSpace(cmds[j].Name)
	})
	return cmds
}

func manListBuiltinRowFromDoc(doc contract.BuiltinCommandDoc) manListBuiltinRow {
	return manListBuiltinRow{
		Name:            strings.TrimSpace(doc.Name),
		Summary:         renderListValue(doc.Summary),
		StdinMode:       strings.TrimSpace(doc.StdinMode),
		Operands:        strings.TrimSpace(doc.Operands),
		DefaultOutput:   strings.TrimSpace(doc.DefaultOutput),
		StructuredFlags: append([]string(nil), doc.StructuredFlags...),
		PipeBehavior:    strings.TrimSpace(doc.PipeBehavior),
		MutationKind:    strings.TrimSpace(doc.MutationKind),
		SuccessOutput:   strings.TrimSpace(doc.SuccessOutput),
		ExitCodes:       append([]string(nil), doc.ExitCodes...),
	}
}

func renderListValue(value string) string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return "-"
	}
	return trimmed
}

func externalSummary(cmd contract.ExternalCommand) string {
	summary := strings.TrimSpace(cmd.Summary)
	if summary == "" {
		return "(no description)"
	}
	return summary
}
